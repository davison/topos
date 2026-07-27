package index

// schema is applied on every Open. Statements are idempotent (CREATE TABLE
// IF NOT EXISTS / CREATE INDEX IF NOT EXISTS) so opening an existing index
// file is safe.
//
// items is a normal rowid table (SQLite's default; no ROWID-less clause
// applied) so an external-content
// FTS5 virtual table can be added over items(title, preview) in Phase 3
// (KERN-05) with content='items', content_rowid='rowid' and no migration.
const schema = `
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;

CREATE TABLE IF NOT EXISTS items (
  id                       TEXT PRIMARY KEY,   -- "{source_type}:{source_id}"
  source_type              TEXT NOT NULL,
  source_id                TEXT NOT NULL,
  title                    TEXT NOT NULL,
  preview                  TEXT NOT NULL,
  timestamp_unix           INTEGER NOT NULL,
  secondary_timestamp_unix INTEGER NOT NULL,
  fidelity                 TEXT NOT NULL,      -- "exact" | "anchored" | "conversation-only"
  deep_link                TEXT NOT NULL,
  labels_json              TEXT NOT NULL,
  provenance_json          TEXT NOT NULL,
  group_id                 TEXT NOT NULL DEFAULT '',
  group_label              TEXT NOT NULL DEFAULT '',
  has_thumbnail            INTEGER NOT NULL DEFAULT 0,
  synced_at                INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS webspace_items (
  webspace_name TEXT NOT NULL,
  item_id       TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE,
  PRIMARY KEY (webspace_name, item_id)
);

-- Registry of every webspace that has completed at least one sync, even if
-- that sync matched zero items. Without this, a webspace with zero items
-- (200, empty array — AGENT-02/empty) is indistinguishable at the SQL
-- level from a webspace name that was never configured/synced (404) —
-- webspace_items alone carries no rows for either case. Added beyond the
-- plan's originally sketched two-table shape (deviation, Rule 2: missing
-- critical functionality) so kernel/httpapi/stream.go can keep its locked
-- signature StreamHandler(store *index.Store) http.HandlerFunc while still
-- returning the correct status code in both cases.
CREATE TABLE IF NOT EXISTS webspaces (
  name        TEXT PRIMARY KEY,
  synced_unix INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS sync_runs (
  id            INTEGER PRIMARY KEY AUTOINCREMENT,
  source_type   TEXT NOT NULL,
  started_unix  INTEGER NOT NULL,
  finished_unix INTEGER,
  status        TEXT NOT NULL,          -- "running" | "ok" | "error"
  error         TEXT NOT NULL DEFAULT '',
  item_count    INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_items_chrono
  ON items(timestamp_unix DESC, secondary_timestamp_unix DESC, id ASC);
`
