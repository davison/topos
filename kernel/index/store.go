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

	"github.com/davison/webspaces/kernel/item"
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
// webspace_items rows for (webspaceName, sourceType), in a single
// transaction — so a sync that fails or is interrupted leaves the pre-sync
// item set for this webspace/source intact, never a partially-written set.
//
// Every item in items MUST have SourceType == sourceType — this method
// trusts its own sourceType parameter for the scoped delete (not each
// item's own SourceType field), so a source's sync can never delete rows
// belonging to a different source_type for the same webspace, even
// transiently between the delete and the reinsert. This is what makes it
// safe for the source-major sync loop in the correlation package to call
// this once per (webspace, source) pair, independently of whether any
// other configured source succeeded or failed in the same sync cycle
// (promoting the sync identity from "webspace" to "(webspace,
// source_type)" — see 02-01-PLAN.md's objective).
//
// items may be nil/empty — a source that matched zero items for this
// webspace still calls this to clear its own previous rows and still marks
// webspaces.synced_unix, registering the webspace as known even if this is
// the only source configured (or the only one that has synced
// successfully so far).
func (s *Store) ReplaceWebspaceSourceItems(ctx context.Context, webspaceName, sourceType string, items []item.Item) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("index: begin webspace source sync transaction: %w", err)
	}
	defer tx.Rollback()

	if err := upsertItemsTx(ctx, tx, items); err != nil {
		return err
	}

	// Source-scoped delete: only rows in webspace_items for this
	// (webspace, source_type) pair are removed. A sibling source's rows
	// for the same webspace are never touched by this statement, even
	// transiently — the whole delete-then-reinsert happens inside this one
	// transaction.
	if _, err := tx.ExecContext(ctx, `
DELETE FROM webspace_items
WHERE webspace_name = ?
  AND item_id IN (SELECT id FROM items WHERE source_type = ?)
`, webspaceName, sourceType); err != nil {
		return fmt.Errorf("index: clear webspace_items for %s/%s: %w", webspaceName, sourceType, err)
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
  id, source_type, source_id, title, preview,
  timestamp_unix, secondary_timestamp_unix, fidelity, deep_link,
  labels_json, provenance_json, group_id, group_label, has_thumbnail, synced_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, unixepoch())
ON CONFLICT(id) DO UPDATE SET
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
			it.ID, it.SourceType, it.SourceID, it.Title, it.Preview,
			it.TimestampUnix, it.SecondaryTimestampUnix, string(it.Fidelity), it.DeepLink,
			string(labelsJSON), string(provJSON), it.GroupID, it.GroupLabel, hasThumb,
		); err != nil {
			return fmt.Errorf("index: upsert item %s: %w", it.ID, err)
		}
	}
	return nil
}

// StreamItems returns every item associated with webspaceName, ordered
// timestamp_unix DESC, secondary_timestamp_unix DESC, id ASC — a total,
// stable chronological order that never depends on SQLite's natural row
// order.
func (s *Store) StreamItems(ctx context.Context, webspaceName string) ([]item.Item, error) {
	const q = `
SELECT items.id, items.source_type, items.source_id, items.title, items.preview,
       items.timestamp_unix, items.secondary_timestamp_unix, items.fidelity, items.deep_link,
       items.labels_json, items.provenance_json, items.group_id, items.group_label, items.has_thumbnail,
       items.synced_at
FROM items
JOIN webspace_items ON webspace_items.item_id = items.id
WHERE webspace_items.webspace_name = ?
ORDER BY items.timestamp_unix DESC, items.secondary_timestamp_unix DESC, items.id ASC
`
	rows, err := s.db.QueryContext(ctx, q, webspaceName)
	if err != nil {
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
			&it.ID, &it.SourceType, &it.SourceID, &it.Title, &it.Preview,
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
SELECT items.id, items.source_type, items.source_id, items.title, items.preview,
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
// preview match rawQuery, ranked best-first by bm25 relevance, each
// carrying a snippet highlighting the matched term between SnippetOpen and
// SnippetClose. A missing, empty, or whitespace/quote-only rawQuery (one
// that sanitizes to "") returns an empty slice and a nil error without
// querying the database; a rawQuery that matches no item also returns an
// empty slice and a nil error. Results are capped at 50 rows. This method
// only reads the local index — it never triggers a live source call.
func (s *Store) Search(ctx context.Context, webspaceName, rawQuery string) ([]SearchResult, error) {
	query := ftsQuery(rawQuery)
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
			&it.ID, &it.SourceType, &it.SourceID, &it.Title, &it.Preview,
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
// ("{source_type}:{source_id}"), or ok=false if no such item is indexed.
// Used by the item-open routes (kernel/httpapi/item.go) to resolve an id
// to a source_type/source_id pair before making a request-time plugin
// call — this is an index read like StreamItems, never a plugin call.
func (s *Store) GetItem(ctx context.Context, id string) (item.Item, bool, error) {
	const q = `
SELECT id, source_type, source_id, title, preview,
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
		&it.ID, &it.SourceType, &it.SourceID, &it.Title, &it.Preview,
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
// (i.e. is "known"), regardless of whether that sync matched any items.
// Used to distinguish a known-but-empty webspace (200, empty array) from
// an unconfigured/never-synced one (404).
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

// SyncRun mirrors a completed sync_runs row.
type SyncRun struct {
	SourceType   string
	StartedUnix  int64
	FinishedUnix int64
	Status       string // "ok" | "error"
	Error        string
	ItemCount    int
}

// LatestSyncRun returns the most recently recorded sync run across all
// sources, or ok=false if none has been recorded yet.
func (s *Store) LatestSyncRun(ctx context.Context) (run SyncRun, ok bool, err error) {
	row := s.db.QueryRowContext(ctx, `
SELECT source_type, started_unix, finished_unix, status, error, item_count
FROM sync_runs ORDER BY id DESC LIMIT 1
`)
	var finished sql.NullInt64
	if scanErr := row.Scan(&run.SourceType, &run.StartedUnix, &finished, &run.Status, &run.Error, &run.ItemCount); scanErr != nil {
		if scanErr == sql.ErrNoRows {
			return SyncRun{}, false, nil
		}
		return SyncRun{}, false, fmt.Errorf("index: latest sync run: %w", scanErr)
	}
	run.FinishedUnix = finished.Int64
	return run, true, nil
}

// StartSyncRun inserts a new sync_runs row for sourceType with status
// "running" and a NULL finished_unix, and returns its id. This is the
// first half of the two-phase write that lets SyncingSourceTypes report
// "is this source syncing right now" between StartSyncRun and the
// matching FinishSyncRun call — the coordinator (kernel/syncer) is the
// only caller of either method (D-06).
func (s *Store) StartSyncRun(ctx context.Context, sourceType string) (int64, error) {
	res, err := s.db.ExecContext(ctx, `
INSERT INTO sync_runs (source_type, started_unix, status) VALUES (?, unixepoch(), 'running')
`, sourceType)
	if err != nil {
		return 0, fmt.Errorf("index: start sync run for %s: %w", sourceType, err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("index: start sync run id for %s: %w", sourceType, err)
	}
	return id, nil
}

// FinishSyncRun updates the sync_runs row started by StartSyncRun (runID)
// with its outcome — it never inserts a new row, so a source's sync
// leaves exactly one row per attempt, running-then-finished, never two.
func (s *Store) FinishSyncRun(ctx context.Context, runID int64, status, errMsg string, itemCount int) error {
	_, err := s.db.ExecContext(ctx, `
UPDATE sync_runs SET finished_unix = unixepoch(), status = ?, error = ?, item_count = ? WHERE id = ?
`, status, errMsg, itemCount, runID)
	if err != nil {
		return fmt.Errorf("index: finish sync run %d: %w", runID, err)
	}
	return nil
}

// LatestSyncRunPerSource returns the most recently started sync_runs row
// for every source_type that has ever recorded one, keyed by source_type.
// A still-running row (NULL finished_unix) scans as FinishedUnix 0,
// exactly as LatestSyncRun already does.
func (s *Store) LatestSyncRunPerSource(ctx context.Context) (map[string]SyncRun, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT source_type, started_unix, finished_unix, status, error, item_count
FROM sync_runs
WHERE id IN (SELECT MAX(id) FROM sync_runs GROUP BY source_type)
`)
	if err != nil {
		return nil, fmt.Errorf("index: latest sync run per source: %w", err)
	}
	defer rows.Close()

	out := map[string]SyncRun{}
	for rows.Next() {
		var run SyncRun
		var finished sql.NullInt64
		if err := rows.Scan(&run.SourceType, &run.StartedUnix, &finished, &run.Status, &run.Error, &run.ItemCount); err != nil {
			return nil, fmt.Errorf("index: scan latest sync run per source row: %w", err)
		}
		run.FinishedUnix = finished.Int64
		out[run.SourceType] = run
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("index: iterate latest sync run per source rows: %w", err)
	}
	return out, nil
}

// SyncingSourceTypes returns the set of source types with an unfinished
// (status "running", finished_unix still NULL) sync_runs row right now.
func (s *Store) SyncingSourceTypes(ctx context.Context) (map[string]bool, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT DISTINCT source_type FROM sync_runs WHERE status = 'running' AND finished_unix IS NULL
`)
	if err != nil {
		return nil, fmt.Errorf("index: syncing source types: %w", err)
	}
	defer rows.Close()

	out := map[string]bool{}
	for rows.Next() {
		var sourceType string
		if err := rows.Scan(&sourceType); err != nil {
			return nil, fmt.Errorf("index: scan syncing source type row: %w", err)
		}
		out[sourceType] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("index: iterate syncing source types: %w", err)
	}
	return out, nil
}
