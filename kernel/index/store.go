// Package index implements the kernel's local SQLite metadata/preview
// index (the hybrid data model's persisted half — KERN-03). Everything in
// this package is a plain database/sql read or write; nothing here ever
// calls a plugin.
package index

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/davison/topos/kernel/item"
)

// Store wraps the local SQLite index.
type Store struct {
	db *sql.DB
}

// Open opens (creating if necessary) the SQLite index at path, applying the
// schema, WAL mode, and foreign keys.
func Open(path string) (*Store, error) {
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("index: create index directory %s: %w", dir, err)
		}
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("index: open %s: %w", path, err)
	}
	// SQLite handles one writer at a time; keep a single connection so WAL
	// checkpointing and cross-statement PRAGMAs behave predictably.
	db.SetMaxOpenConns(1)

	if err := rebuildOnSchemaChange(db); err != nil {
		db.Close()
		return nil, err
	}

	// Recorded BEFORE the schema is applied, so this reflects whether
	// items_fts existed in this index file prior to this Open call — an
	// index file that already holds items synced before Phase 3 would
	// otherwise have an empty FTS index forever, since the sync triggers
	// only fire on writes made after they exist.
	var ftsExistedBefore int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'items_fts'`,
	).Scan(&ftsExistedBefore); err != nil {
		db.Close()
		return nil, fmt.Errorf("index: check items_fts existence: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("index: apply schema: %w", err)
	}

	if ftsExistedBefore == 0 {
		// First creation of items_fts in this index file (a brand new
		// index, or a pre-Phase-3 file opened for the first time since
		// this schema addition landed) — 'rebuild' re-derives the FTS
		// index from every existing items row, backfilling anything
		// synced before the table/triggers existed.
		if _, err := db.Exec(`INSERT INTO items_fts(items_fts) VALUES('rebuild')`); err != nil {
			db.Close()
			return nil, fmt.Errorf("index: backfill items_fts: %w", err)
		}
	}

	return &Store{db: db}, nil
}

// rebuildOnSchemaChange compares the on-disk PRAGMA user_version against
// schemaVersion and, when they differ on a non-empty index file, drops
// every schema.go-owned table/trigger/virtual-table inside a single
// transaction before recording the new version — D-07's drop-and-resync
// migration strategy: no data migration is ever written, because every
// indexed row is re-derivable from the next sync. A brand new database file
// (no items table yet) is not a stale schema, just new, so it skips the
// drop step and simply records the current version.
//
// The drop and the PRAGMA user_version write happen inside one transaction,
// so a rebuild interrupted part-way (kernel killed mid-rebuild) leaves the
// on-disk user_version at its OLD value: SQLite rolls back the incomplete
// transaction, and the next Open re-detects the stale version and repeats
// the whole rebuild from scratch, rather than ever serving a half-dropped
// index with some tables gone and others still in the old shape.
func rebuildOnSchemaChange(db *sql.DB) error {
	var userVersion int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&userVersion); err != nil {
		return fmt.Errorf("index: read schema version: %w", err)
	}
	if userVersion == schemaVersion {
		return nil
	}

	var itemsTableExists int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = 'items'`,
	).Scan(&itemsTableExists); err != nil {
		return fmt.Errorf("index: check items table existence: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("index: begin schema version transaction: %w", err)
	}
	defer tx.Rollback()

	if itemsTableExists != 0 {
		for _, stmt := range []string{
			`DROP TABLE IF EXISTS items_fts`,
			`DROP TRIGGER IF EXISTS items_ai`,
			`DROP TRIGGER IF EXISTS items_ad`,
			`DROP TRIGGER IF EXISTS items_au`,
			`DROP TABLE IF EXISTS webspace_items`,
			`DROP TABLE IF EXISTS webspaces`,
			`DROP TABLE IF EXISTS sync_runs`,
			`DROP TABLE IF EXISTS items`,
		} {
			if _, err := tx.Exec(stmt); err != nil {
				return fmt.Errorf("index: schema rebuild (%s): %w", stmt, err)
			}
		}
	}

	if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, schemaVersion)); err != nil {
		return fmt.Errorf("index: set schema version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("index: commit schema version transaction: %w", err)
	}
	return nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

// UpsertItems inserts or replaces the given items in the items table, in
// its own transaction.
func (s *Store) UpsertItems(ctx context.Context, items []item.Item) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("index: begin upsert transaction: %w", err)
	}
	defer tx.Rollback()

	if err := upsertItemsTx(ctx, tx, items); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("index: commit upsert transaction: %w", err)
	}
	return nil
}

// ReplaceWebspaceSourceItems upserts items and replaces ONLY the
// webspace_items rows for (webspaceName, source), in a single
// transaction — so a sync that fails or is interrupted leaves the pre-sync
// item set for this webspace/source intact, never a partially-written set.
//
// Every item in items MUST have Item.Source == source — this method
// trusts its own source parameter (the instance id) for the scoped delete
// (not each item's own Source field), so a source instance's sync can never
// delete rows belonging to a different instance for the same webspace, even
// transiently between the delete and the reinsert. Two configured instances
// of the same plugin type therefore never merge or coalesce their rows
// (D-10) — this is what makes it safe for the source-major sync loop in the
// correlation package to call this once per (webspace, instance) pair,
// independently of whether any other configured source succeeded or failed
// in the same sync cycle (promoting the sync identity from "webspace" to
// "(webspace, source)" — see 02-01-PLAN.md's objective, generalized to
// instance identity by D-08).
//
// items may be nil/empty — a source that matched zero items for this
// webspace still calls this to clear its own previous rows and still marks
// webspaces.synced_unix, registering the webspace as known even if this is
// the only source configured (or the only one that has synced
// successfully so far).
func (s *Store) ReplaceWebspaceSourceItems(ctx context.Context, webspaceName, source string, items []item.Item) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("index: begin webspace source sync transaction: %w", err)
	}
	defer tx.Rollback()

	if err := upsertItemsTx(ctx, tx, items); err != nil {
		return err
	}

	// Source-scoped delete: only rows in webspace_items for this
	// (webspace, source instance) pair are removed. A sibling source
	// instance's rows for the same webspace — including another instance
	// of the SAME plugin type — are never touched by this statement, even
	// transiently — the whole delete-then-reinsert happens inside this one
	// transaction.
	if _, err := tx.ExecContext(ctx, `
DELETE FROM webspace_items
WHERE webspace_name = ?
  AND item_id IN (SELECT id FROM items WHERE source = ?)
`, webspaceName, source); err != nil {
		return fmt.Errorf("index: clear webspace_items for %s/%s: %w", webspaceName, source, err)
	}

	for _, it := range items {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO webspace_items (webspace_name, item_id) VALUES (?, ?)`,
			webspaceName, it.ID,
		); err != nil {
			return fmt.Errorf("index: insert webspace_items row for %s/%s: %w", webspaceName, it.ID, err)
		}
	}

	// Marked on ANY source's successful contribution, not gated on every
	// configured source having synced at least once — otherwise a
	// webspace with one working source and one not-yet-configured/still-
	// erroring source would incorrectly 404 as "never synced"
	// (02-RESEARCH.md Critical Architecture Finding).
	if _, err := tx.ExecContext(ctx, `
INSERT INTO webspaces (name, synced_unix) VALUES (?, unixepoch())
ON CONFLICT(name) DO UPDATE SET synced_unix = excluded.synced_unix
`, webspaceName); err != nil {
		return fmt.Errorf("index: mark webspace %s as synced: %w", webspaceName, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("index: commit webspace source sync transaction: %w", err)
	}
	return nil
}

func upsertItemsTx(ctx context.Context, tx *sql.Tx, items []item.Item) error {
	const stmt = `
INSERT INTO items (
  id, source, source_type, source_id, title, preview,
  timestamp_unix, secondary_timestamp_unix, fidelity, deep_link,
  labels_json, provenance_json, group_id, group_label, has_thumbnail, synced_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, unixepoch())
ON CONFLICT(id) DO UPDATE SET
  source = excluded.source,
  source_type = excluded.source_type,
  source_id = excluded.source_id,
  title = excluded.title,
  preview = excluded.preview,
  timestamp_unix = excluded.timestamp_unix,
  secondary_timestamp_unix = excluded.secondary_timestamp_unix,
  fidelity = excluded.fidelity,
  deep_link = excluded.deep_link,
  labels_json = excluded.labels_json,
  provenance_json = excluded.provenance_json,
  group_id = excluded.group_id,
  group_label = excluded.group_label,
  has_thumbnail = excluded.has_thumbnail,
  synced_at = excluded.synced_at
`
	for _, it := range items {
		labelsJSON, err := json.Marshal(it.Labels)
		if err != nil {
			return fmt.Errorf("index: marshal labels for %s: %w", it.ID, err)
		}
		provJSON, err := json.Marshal(it.Provenance)
		if err != nil {
			return fmt.Errorf("index: marshal provenance for %s: %w", it.ID, err)
		}
		hasThumb := 0
		if it.HasThumbnail {
			hasThumb = 1
		}
		if _, err := tx.ExecContext(ctx, stmt,
			it.ID, it.Source, it.SourceType, it.SourceID, it.Title, it.Preview,
			it.TimestampUnix, it.SecondaryTimestampUnix, string(it.Fidelity), it.DeepLink,
			string(labelsJSON), string(provJSON), it.GroupID, it.GroupLabel, hasThumb,
		); err != nil {
			return fmt.Errorf("index: upsert item %s: %w", it.ID, err)
		}
	}
	return nil
}

// DeleteSourceItems removes every items row belonging to source instance
// source, in one statement (D-07's removed-instance cleanup,
// 07-02-PLAN.md Task 1). The existing ON DELETE CASCADE on
// webspace_items.item_id (schema.go) already clears that instance's rows
// across every webspace it participated in, and the existing items_ad
// trigger already keeps items_fts in sync on any items delete — no new
// trigger is needed. Safe to call for a source with zero rows (a no-op).
func (s *Store) DeleteSourceItems(ctx context.Context, source string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM items WHERE source = ?`, source); err != nil {
		return fmt.Errorf("index: delete items for source %q: %w", source, err)
	}
	return nil
}

// DeleteSyncRuns clears source's entire sync_runs history. Paired with
// DeleteSourceItems (T-07-13): a source instance removed from config and
// later re-added under the same [sources.<id>] key must start with no
// inherited items and no inherited sync history — without this, a
// removed-then-re-added instance would show phantom old runs even though
// its items were correctly cleared.
func (s *Store) DeleteSyncRuns(ctx context.Context, source string) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM sync_runs WHERE source = ?`, source); err != nil {
		return fmt.Errorf("index: delete sync runs for source %q: %w", source, err)
	}
	return nil
}

// StreamItems returns every item associated with webspaceName, ordered
// timestamp_unix DESC, secondary_timestamp_unix DESC, id ASC — a total,
// stable chronological order that never depends on SQLite's natural row
// order.
// StreamItems returns webspaceName's items in chronological order
// (newest first), optionally narrowed to only those matching every term in
// filterTerms (D-16/D-18: a webspace's saved permanent filter — an AND-ed
// FTS query-time narrowing, never a sync-time one). With no surviving
// terms, this runs the identical SQL Phase 1-6 always ran, so a webspace
// with no filter key streams byte-identically to its pre-Phase-7 output.
// With terms, the query joins through items_fts exactly like Search does,
// but keeps StreamItems' own chronological ORDER BY (never bm25 rank) and
// no LIMIT — the filtered view is still the whole stream, just narrower.
// An FTS5-hostile filter term degrades to an empty slice with a nil error,
// mirroring Search's own fts5-error degradation, rather than a 500.
func (s *Store) StreamItems(ctx context.Context, webspaceName string, filterTerms []string) ([]item.Item, error) {
	match := BuildMatchQuery(filterTerms, "")

	const baseColumns = `
SELECT items.id, items.source, items.source_type, items.source_id, items.title, items.preview,
       items.timestamp_unix, items.secondary_timestamp_unix, items.fidelity, items.deep_link,
       items.labels_json, items.provenance_json, items.group_id, items.group_label, items.has_thumbnail,
       items.synced_at
`

	var rows *sql.Rows
	var err error
	if match == "" {
		const q = baseColumns + `
FROM items
JOIN webspace_items ON webspace_items.item_id = items.id
WHERE webspace_items.webspace_name = ?
ORDER BY items.timestamp_unix DESC, items.secondary_timestamp_unix DESC, items.id ASC
`
		rows, err = s.db.QueryContext(ctx, q, webspaceName)
	} else {
		const q = baseColumns + `
FROM items_fts
JOIN items ON items.rowid = items_fts.rowid
JOIN webspace_items ON webspace_items.item_id = items.id
WHERE webspace_items.webspace_name = ?
  AND items_fts MATCH ?
ORDER BY items.timestamp_unix DESC, items.secondary_timestamp_unix DESC, items.id ASC
`
		rows, err = s.db.QueryContext(ctx, q, webspaceName, match)
	}
	if err != nil {
		// Mirror Search's fts5-error degradation (03-RESEARCH.md Pattern
		// 3): BuildMatchQuery's phrase-quoting makes a genuine MATCH
		// syntax error unreachable in principle, but a malformed filter
		// term must degrade to "no results" rather than propagate as a
		// 500 if it ever occurs.
		if match != "" && strings.Contains(strings.ToLower(err.Error()), "fts5") {
			return []item.Item{}, nil
		}
		return nil, fmt.Errorf("index: stream items for %s: %w", webspaceName, err)
	}
	defer rows.Close()

	var out []item.Item
	for rows.Next() {
		var it item.Item
		var fidelity string
		var labelsJSON, provJSON string
		var hasThumb int
		if err := rows.Scan(
			&it.ID, &it.Source, &it.SourceType, &it.SourceID, &it.Title, &it.Preview,
			&it.TimestampUnix, &it.SecondaryTimestampUnix, &fidelity, &it.DeepLink,
			&labelsJSON, &provJSON, &it.GroupID, &it.GroupLabel, &hasThumb,
			&it.SyncedAtUnix,
		); err != nil {
			return nil, fmt.Errorf("index: scan item row: %w", err)
		}
		it.Fidelity = item.Fidelity(fidelity)
		it.HasThumbnail = hasThumb != 0
		if err := json.Unmarshal([]byte(labelsJSON), &it.Labels); err != nil {
			return nil, fmt.Errorf("index: unmarshal labels for %s: %w", it.ID, err)
		}
		if err := json.Unmarshal([]byte(provJSON), &it.Provenance); err != nil {
			return nil, fmt.Errorf("index: unmarshal provenance for %s: %w", it.ID, err)
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("index: iterate stream items for %s: %w", webspaceName, err)
	}
	return out, nil
}

// BuildMatchQuery composes one FTS5 MATCH expression from a webspace's
// saved permanent filter stack plus an optional live search query (D-16/
// D-18) — the single shared builder StreamItems, Search and every /agent/v1
// stream caller all go through, so the filtered view can never disagree
// with itself across consumers. filterTerms is iterated in its given
// (stored array) order, never sorted — the persisted order is also the
// chip-rendering order (UI-12 ordering edge), so the MATCH expression and
// the UI can never disagree about that order either. Each term has any `"`
// characters stripped, blanks are skipped, and survivors are phrase-quoted
// with NO trailing `*` — a saved filter means the word, not a prefix,
// unlike a live in-progress search term. liveQuery is run through the
// existing ftsQuery (unchanged: still phrase-quotes and prefix-matches its
// own last term). The parts join with a single space, FTS5's implicit AND
// — so a filter stack of ["boiler","quote"] plus a live query of "invoice"
// requires all three to match. Returns "" when both filterTerms and
// liveQuery sanitize to nothing, exactly like ftsQuery's own "no query"
// convention.
func BuildMatchQuery(filterTerms []string, liveQuery string) string {
	parts := make([]string, 0, len(filterTerms)+1)
	for _, term := range filterTerms {
		term = strings.ReplaceAll(term, `"`, "")
		if term == "" {
			continue
		}
		parts = append(parts, `"`+term+`"`)
	}
	if live := ftsQuery(liveQuery); live != "" {
		parts = append(parts, live)
	}
	return strings.Join(parts, " ")
}

// SnippetOpen and SnippetClose are the delimiter runes Search wraps a
// matched term with inside SearchResult.Snippet. They are the ASCII
// control characters STX (0x02) and ETX (0x03) rather than any printable
// character, because no real subject line or preview text can contain
// them — a delimiter can therefore never be mistaken for source content.
// Exported so the HTTP layer and docs/api.md can name the same values
// instead of duplicating the literal.
const (
	SnippetOpen  = "\x02"
	SnippetClose = "\x03"
)

// SearchResult is one ranked, snippet-carrying match returned by Search.
type SearchResult struct {
	Item    item.Item
	Snippet string
	// Rank is the raw bm25() score: more-negative is a better match.
	Rank float64
}

// ftsQuery sanitizes raw search-box text into a literal-phrase FTS5 query
// string safe to hand to MATCH. Handing a caller's raw text straight to
// MATCH would let a lone double-quote or a leading hyphen produce an FTS5
// query-syntax error instead of a result set — every caller of Search goes
// through this helper, never MATCH directly.
//
// raw is split on whitespace; each field has any double-quote characters
// stripped, and fields left empty afterward are dropped; every surviving
// field is wrapped in double quotes (so FTS5 treats it as a literal phrase
// rather than query syntax) and joined with spaces (FTS5's implicit AND);
// the final term is suffixed with * so an in-progress word prefix-matches.
// Returns "" when no field survives (raw was empty, whitespace-only, or
// contained only quote characters) — Search treats that as "no query".
func ftsQuery(raw string) string {
	fields := strings.Fields(raw)
	kept := make([]string, 0, len(fields))
	for _, f := range fields {
		f = strings.ReplaceAll(f, `"`, "")
		if f == "" {
			continue
		}
		kept = append(kept, `"`+f+`"`)
	}
	if len(kept) == 0 {
		return ""
	}
	kept[len(kept)-1] += "*"
	return strings.Join(kept, " ")
}

// searchQuery selects the identical column list StreamItems selects, plus
// a highlighted snippet and a bm25 relevance rank, scoped to one webspace
// via the same webspace_items join StreamItems uses. bm25() returns a
// more-negative score for a better match, so ORDER BY rank ASC is correct
// (03-RESEARCH.md Pattern 3). Both the webspace name and the sanitized FTS5
// query are bound parameters — nothing here is built by string
// concatenation.
const searchQuery = `
SELECT items.id, items.source, items.source_type, items.source_id, items.title, items.preview,
       items.timestamp_unix, items.secondary_timestamp_unix, items.fidelity, items.deep_link,
       items.labels_json, items.provenance_json, items.group_id, items.group_label, items.has_thumbnail,
       items.synced_at,
       snippet(items_fts, -1, ?, ?, '…', 12) AS snip,
       bm25(items_fts) AS rank
FROM items_fts
JOIN items ON items.rowid = items_fts.rowid
JOIN webspace_items ON webspace_items.item_id = items.id
WHERE webspace_items.webspace_name = ?
  AND items_fts MATCH ?
ORDER BY rank ASC
LIMIT 50
`

// Search returns items associated with webspaceName whose indexed title or
// preview match BOTH every term in filterTerms (the webspace's saved
// permanent filter, D-16/D-18) AND rawQuery (the live in-progress search
// text), ranked best-first by bm25 relevance, each carrying a snippet
// highlighting the matched term between SnippetOpen and SnippetClose. The
// combined MATCH expression is built by the same BuildMatchQuery StreamItems
// uses, so a live search AND-combines with the active filter stack rather
// than escaping it — a further search always refines within the saved
// filter, never replaces it. The empty-result short circuit now depends on
// BOTH inputs: it fires only when filterTerms and rawQuery both sanitize to
// nothing, so a filter-only call (empty rawQuery, non-empty filterTerms)
// still queries and ranks by relevance rather than returning early. A
// rawQuery/filterTerms combination that matches no item also returns an
// empty slice and a nil error. Results are capped at 50 rows. This method
// only reads the local index — it never triggers a live source call.
func (s *Store) Search(ctx context.Context, webspaceName, rawQuery string, filterTerms []string) ([]SearchResult, error) {
	query := BuildMatchQuery(filterTerms, rawQuery)
	if query == "" {
		return []SearchResult{}, nil
	}

	rows, err := s.db.QueryContext(ctx, searchQuery, SnippetOpen, SnippetClose, webspaceName, query)
	if err != nil {
		// ftsQuery's phrase-quoting makes a genuine MATCH syntax error
		// unreachable in principle, but a malformed query must degrade to
		// "no results" rather than propagate as a 500 if it ever occurs.
		if strings.Contains(strings.ToLower(err.Error()), "fts5") {
			return []SearchResult{}, nil
		}
		return nil, fmt.Errorf("index: search %s: %w", webspaceName, err)
	}
	defer rows.Close()

	out := []SearchResult{}
	for rows.Next() {
		var it item.Item
		var fidelity string
		var labelsJSON, provJSON string
		var hasThumb int
		var snip string
		var rank float64
		if err := rows.Scan(
			&it.ID, &it.Source, &it.SourceType, &it.SourceID, &it.Title, &it.Preview,
			&it.TimestampUnix, &it.SecondaryTimestampUnix, &fidelity, &it.DeepLink,
			&labelsJSON, &provJSON, &it.GroupID, &it.GroupLabel, &hasThumb,
			&it.SyncedAtUnix, &snip, &rank,
		); err != nil {
			return nil, fmt.Errorf("index: scan search result row: %w", err)
		}
		it.Fidelity = item.Fidelity(fidelity)
		it.HasThumbnail = hasThumb != 0
		if err := json.Unmarshal([]byte(labelsJSON), &it.Labels); err != nil {
			return nil, fmt.Errorf("index: unmarshal labels for %s: %w", it.ID, err)
		}
		if err := json.Unmarshal([]byte(provJSON), &it.Provenance); err != nil {
			return nil, fmt.Errorf("index: unmarshal provenance for %s: %w", it.ID, err)
		}
		out = append(out, SearchResult{Item: it, Snippet: snip, Rank: rank})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("index: iterate search results for %s: %w", webspaceName, err)
	}
	return out, nil
}

// GetItem returns the single item with the given composite id
// ("{source}:{source_id}"), or ok=false if no such item is indexed. Used by
// the item-open routes (kernel/httpapi/item.go) to resolve an id to a
// source (instance id) / source_id pair before making a request-time
// plugin call — this is an index read like StreamItems, never a plugin
// call.
func (s *Store) GetItem(ctx context.Context, id string) (item.Item, bool, error) {
	const q = `
SELECT id, source, source_type, source_id, title, preview,
       timestamp_unix, secondary_timestamp_unix, fidelity, deep_link,
       labels_json, provenance_json, group_id, group_label, has_thumbnail,
       synced_at
FROM items WHERE id = ?
`
	var it item.Item
	var fidelity string
	var labelsJSON, provJSON string
	var hasThumb int
	err := s.db.QueryRowContext(ctx, q, id).Scan(
		&it.ID, &it.Source, &it.SourceType, &it.SourceID, &it.Title, &it.Preview,
		&it.TimestampUnix, &it.SecondaryTimestampUnix, &fidelity, &it.DeepLink,
		&labelsJSON, &provJSON, &it.GroupID, &it.GroupLabel, &hasThumb,
		&it.SyncedAtUnix,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return item.Item{}, false, nil
		}
		return item.Item{}, false, fmt.Errorf("index: get item %s: %w", id, err)
	}
	it.Fidelity = item.Fidelity(fidelity)
	it.HasThumbnail = hasThumb != 0
	if err := json.Unmarshal([]byte(labelsJSON), &it.Labels); err != nil {
		return item.Item{}, false, fmt.Errorf("index: unmarshal labels for %s: %w", id, err)
	}
	if err := json.Unmarshal([]byte(provJSON), &it.Provenance); err != nil {
		return item.Item{}, false, fmt.Errorf("index: unmarshal provenance for %s: %w", id, err)
	}
	return it, true, nil
}

// Webspaces returns the current item count for every webspace that has
// completed at least one sync (including zero-item syncs), keyed by
// webspace name.
func (s *Store) Webspaces(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT webspaces.name, COUNT(webspace_items.item_id)
FROM webspaces
LEFT JOIN webspace_items ON webspace_items.webspace_name = webspaces.name
GROUP BY webspaces.name
`)
	if err != nil {
		return nil, fmt.Errorf("index: webspace counts: %w", err)
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var name string
		var count int
		if err := rows.Scan(&name, &count); err != nil {
			return nil, fmt.Errorf("index: scan webspace count row: %w", err)
		}
		counts[name] = count
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("index: iterate webspace counts: %w", err)
	}
	return counts, nil
}

// WebspaceExists reports whether name has completed at least one sync
// (i.e. its ReplaceWebspaceSourceItems insert has ever run for it),
// regardless of whether that sync matched any items. This answers only
// the sync-history half of "does this webspace exist" — it is NOT the
// definition of existence. kernel/httpapi's webspaceIsKnown
// (07-15-PLAN.md, closes 07-UAT.md G-07-1) is the actual gate every HTTP
// surface asks through: it checks the running configuration FIRST (a
// webspace is servable the instant its `PUT /api/config` returns, before
// any sync has run) and falls through to this query only when the config
// half answers false, so a webspace whose block was later removed from
// the file while its rows survive still answers true. Call this directly
// only when sync history specifically, not existence, is the question.
func (s *Store) WebspaceExists(ctx context.Context, name string) (bool, error) {
	var exists int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM webspaces WHERE name = ?`, name).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("index: check webspace exists %s: %w", name, err)
	}
	return true, nil
}

// SyncRun mirrors a completed sync_runs row, keyed by source INSTANCE id
// (the [sources.<id>] config key, not the plugin kind — D-08).
type SyncRun struct {
	Source       string
	StartedUnix  int64
	FinishedUnix int64
	Status       string // "ok" | "error"
	Error        string
	ItemCount    int
	// Notice is a non-fatal, human-readable advisory recorded alongside
	// this run's own outcome (12-09-PLAN.md, G-12-1/G-12-3) — see
	// kernel/correlate.WebspaceResult.Notice and
	// kernel/syncer.joinNotices for where it comes from. Empty for any
	// run finished through the plain FinishSyncRun spelling. Never an
	// error, and never touched by a genuine sync failure.
	Notice string
}

// LatestSyncRun returns the most recently recorded sync run across all
// sources, or ok=false if none has been recorded yet.
func (s *Store) LatestSyncRun(ctx context.Context) (run SyncRun, ok bool, err error) {
	row := s.db.QueryRowContext(ctx, `
SELECT source, started_unix, finished_unix, status, error, item_count, notice
FROM sync_runs ORDER BY id DESC LIMIT 1
`)
	var finished sql.NullInt64
	if scanErr := row.Scan(&run.Source, &run.StartedUnix, &finished, &run.Status, &run.Error, &run.ItemCount, &run.Notice); scanErr != nil {
		if scanErr == sql.ErrNoRows {
			return SyncRun{}, false, nil
		}
		return SyncRun{}, false, fmt.Errorf("index: latest sync run: %w", scanErr)
	}
	run.FinishedUnix = finished.Int64
	return run, true, nil
}

// StartSyncRun inserts a new sync_runs row for the source instance id with
// status "running" and a NULL finished_unix, and returns its id. This is
// the first half of the two-phase write that lets SyncingSources report
// "is this source syncing right now" between StartSyncRun and the
// matching FinishSyncRun call — the coordinator (kernel/syncer) is the
// only caller of either method (D-06). The single-flight key the
// coordinator uses is this same instance id, so two instances of one
// plugin type sync concurrently and never coalesce into one another's run.
func (s *Store) StartSyncRun(ctx context.Context, source string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
INSERT INTO sync_runs (source, started_unix, status) VALUES (?, unixepoch(), 'running')
`, source)
	if err != nil {
		return 0, fmt.Errorf("index: start sync run for %s: %w", source, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("index: start sync run id for %s: %w", source, err)
	}
	return id, nil
}

// FinishSyncRun updates the sync_runs row started by StartSyncRun (runID)
// with its outcome — it never inserts a new row, so a source's sync
// leaves exactly one row per attempt, running-then-finished, never two.
//
// This spelling delegates to FinishSyncRunWithNotice with an empty
// notice, keeping its own signature and behaviour byte-identical for its
// roughly twenty-five existing callers — mirroring the sibling-method
// shape kernel/pluginhost/matchconfig.go's ValidateMatchConfig /
// ValidateMatchConfigWithSuspended already establish in this repo:
// FinishSyncRun is for a caller with no notice to report (which is every
// caller except one); FinishSyncRunWithNotice is for
// kernel/syncer.Coordinator, the only caller that ever has one.
func (s *Store) FinishSyncRun(ctx context.Context, runID int64, status, errMsg string, itemCount int) error {
	return s.FinishSyncRunWithNotice(ctx, runID, status, errMsg, "", itemCount)
}

// FinishSyncRunWithNotice is FinishSyncRun's sibling (12-09-PLAN.md,
// G-12-1/G-12-3): it additionally records a non-fatal advisory in the
// SAME single UPDATE as status, error and item_count, so a run's outcome
// and its advisory can never be half-recorded by a write interrupted
// between the two.
func (s *Store) FinishSyncRunWithNotice(ctx context.Context, runID int64, status, errMsg, notice string, itemCount int) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE sync_runs SET finished_unix = unixepoch(), status = ?, error = ?, item_count = ?, notice = ? WHERE id = ?
`, status, errMsg, itemCount, notice, runID)
	if err != nil {
		return fmt.Errorf("index: finish sync run %d: %w", runID, err)
	}
	return nil
}

// ReconcileInterruptedSyncRuns finalises every sync_runs row still left at
// status "running" with a NULL finished_unix, marking it "error" with an
// "interrupted" message, and returns how many rows it repaired.
//
// This is a startup-only repair and is safe precisely because the
// coordinator (kernel/syncer) is the only writer of these rows and holds
// its in-flight run IDs in memory: a freshly-started kernel has no
// in-flight runs by definition, so any row still "running" at boot was
// stranded by a previous process that died or was cancelled before it
// could finalise. Without this, such a row survives every restart forever
// — there is no other path in the system that ever finalises it.
//
// Rows are recorded as "error"/interrupted rather than deleted: the run
// genuinely was attempted and genuinely did not complete, and silently
// dropping it would misreport sync history as cleaner than it was.
func (s *Store) ReconcileInterruptedSyncRuns(ctx context.Context) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
UPDATE sync_runs
SET finished_unix = unixepoch(), status = 'error', error = 'interrupted: kernel stopped before this sync run finished'
WHERE status = 'running' AND finished_unix IS NULL
`)
	if err != nil {
		return 0, fmt.Errorf("index: reconcile interrupted sync runs: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("index: reconcile interrupted sync runs rows affected: %w", err)
	}
	return n, nil
}

// LatestSyncRunPerSource returns the most recently started sync_runs row
// for every source INSTANCE that has ever recorded one, keyed by instance
// id — one entry per instance, so two instances of the same plugin type
// each get their own independent series rather than being conflated into
// one (D-08). A still-running row (NULL finished_unix) scans as
// FinishedUnix 0, exactly as LatestSyncRun already does.
func (s *Store) LatestSyncRunPerSource(ctx context.Context) (map[string]SyncRun, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT source, started_unix, finished_unix, status, error, item_count, notice
FROM sync_runs
WHERE id IN (SELECT MAX(id) FROM sync_runs GROUP BY source)
`)
	if err != nil {
		return nil, fmt.Errorf("index: latest sync run per source: %w", err)
	}
	defer rows.Close()

	out := map[string]SyncRun{}
	for rows.Next() {
		var run SyncRun
		var finished sql.NullInt64
		if err := rows.Scan(&run.Source, &run.StartedUnix, &finished, &run.Status, &run.Error, &run.ItemCount, &run.Notice); err != nil {
			return nil, fmt.Errorf("index: scan latest sync run per source row: %w", err)
		}
		run.FinishedUnix = finished.Int64
		out[run.Source] = run
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("index: iterate latest sync run per source rows: %w", err)
	}
	return out, nil
}

// SyncRunsForSourceForTesting returns every sync_runs row recorded for one
// source INSTANCE id, oldest first — that instance's full run history
// rather than the single latest row LatestSyncRunPerSource answers with.
// Rows are ordered by id, which sync_runs' INTEGER PRIMARY KEY makes
// monotonic in insertion order, so index 0 is always that instance's
// first-ever recorded run. A still-running row (NULL finished_unix) scans
// as FinishedUnix 0, exactly as the sibling readers above do.
//
// TEST-ONLY surface, named for it after config.NewStoreForTesting: no
// production code path needs a source's full run history today, and this
// reader exists solely because a test needed to address one SPECIFIC run.
// Keeping the ForTesting suffix stops it being mistaken for part of the
// supported read API — if a real production caller ever appears, rename it
// then rather than letting the untested-in-production shape leak out now.
//
// The question it answers is the one the latest-row readers cannot, and
// the distinction is exactly what KB-002 records
// (.planning/debug/knowledge-base.md): "is this source syncing right now"
// is a property of its latest row, but "was the run that was in flight at
// moment T finalised" is a property of THAT row, and a latest-row
// aggregate silently answers it with whichever later run has since
// superseded it. TestApply_MidFlightSyncLeavesNoStrandedRunningRow
// (kernel/supervisor) is the caller that needs the second question: it
// pins that Apply finalises the sync that was mid-flight when it was
// called, while Apply's own failure branch legitimately starts a fresh
// generation whose immediate first refresh records a NEWER run for the
// same instance before Apply returns.
func (s *Store) SyncRunsForSourceForTesting(ctx context.Context, source string) ([]SyncRun, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT source, started_unix, finished_unix, status, error, item_count, notice
FROM sync_runs
WHERE source = ?
ORDER BY id
`, source)
	if err != nil {
		return nil, fmt.Errorf("index: sync runs for source %s: %w", source, err)
	}
	defer rows.Close()

	var out []SyncRun
	for rows.Next() {
		var run SyncRun
		var finished sql.NullInt64
		if err := rows.Scan(&run.Source, &run.StartedUnix, &finished, &run.Status, &run.Error, &run.ItemCount, &run.Notice); err != nil {
			return nil, fmt.Errorf("index: scan sync run row for source %s: %w", source, err)
		}
		run.FinishedUnix = finished.Int64
		out = append(out, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("index: iterate sync run rows for source %s: %w", source, err)
	}
	return out, nil
}

// SyncingSources returns the set of source INSTANCE ids whose LATEST
// sync_runs row is unfinished (status "running", finished_unix still
// NULL) — i.e. the source instances syncing right now. Two instances of
// one plugin type appear (or don't) entirely independently, since each has
// its own sync_runs series keyed by instance id.
//
// The latest-row restriction is load-bearing, not an optimisation. This
// query previously matched ANY running row (`WHERE status = 'running'`
// alone), which meant a single orphaned row — one left unfinalised by an
// interrupted sync — reported its source as syncing forever, outvoting
// every subsequent successful run and pinning the UI's spinner on
// permanently. A source's current sync state is a property of its current
// run, so only the newest row can answer it: a started run inserts the
// newest row (syncing true) and finishing updates that same row in place
// (syncing false), while any older stranded row is correctly ignored.
// ReconcileInterruptedSyncRuns clears such orphans at startup; this bounds
// the damage if one is ever created while the kernel is running.
func (s *Store) SyncingSources(ctx context.Context) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT source FROM sync_runs
WHERE id IN (SELECT MAX(id) FROM sync_runs GROUP BY source)
  AND status = 'running' AND finished_unix IS NULL
`)
	if err != nil {
		return nil, fmt.Errorf("index: syncing sources: %w", err)
	}
	defer rows.Close()

	out := map[string]bool{}
	for rows.Next() {
		var source string
		if err := rows.Scan(&source); err != nil {
			return nil, fmt.Errorf("index: scan syncing source row: %w", err)
		}
		out[source] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("index: iterate syncing sources: %w", err)
	}
	return out, nil
}
