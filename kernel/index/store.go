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

	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("index: apply schema: %w", err)
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
// safe for the source-major sync loop in kernel/correlate/correlate.go to
// call this once per (webspace, source) pair, independently of whether any
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

// RecordSyncRun records one completed sync attempt for a source.
func (s *Store) RecordSyncRun(ctx context.Context, run SyncRun) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO sync_runs (source_type, started_unix, finished_unix, status, error, item_count)
VALUES (?, ?, ?, ?, ?, ?)
`, run.SourceType, run.StartedUnix, run.FinishedUnix, run.Status, run.Error, run.ItemCount)
	if err != nil {
		return fmt.Errorf("index: record sync run for %s: %w", run.SourceType, err)
	}
	return nil
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
