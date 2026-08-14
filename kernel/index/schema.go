package index

// schemaVersion is the current PRAGMA user_version this schema targets.
// Store.Open compares it against the on-disk value and, on a mismatch in a
// non-empty index file, drops and recreates every table below rather than
// writing a data migration (D-07) — every row here is re-derivable from a
// fresh sync, so there is nothing worth migrating. Bump this whenever a
// schema change (like this phase's items.source / sync_runs.source
// addition, or 12-09-PLAN.md's sync_runs.notice column below) makes
// previously-indexed rows structurally incompatible with the new shape.
const schemaVersion = 3

// schema is applied on every Open. Statements are idempotent (CREATE TABLE
// IF NOT EXISTS / CREATE INDEX IF NOT EXISTS) so opening an existing index
// file is safe.
//
// items is a normal rowid table (SQLite's default; no ROWID-less clause
// applied) so an external-content
// FTS5 virtual table can be added over items(title, preview) in Phase 3
// (KERN-05) with content='items', content_rowid='rowid' and no migration.
// Phase 3 filled in that design below (items_fts + its three synchronising
// triggers) — see Store.Open's first-creation backfill for the companion
// piece that makes items synced before this schema addition searchable too.
const schema = `
PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;

CREATE TABLE IF NOT EXISTS items (
  id                       TEXT PRIMARY KEY,   -- "{source}:{source_id}"
  source                   TEXT NOT NULL,      -- source INSTANCE id ([sources.<id>] config key) — the identity key (D-08)
  source_type              TEXT NOT NULL,      -- Describe-learned plugin kind — descriptive provenance only, never identity
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
  source        TEXT NOT NULL,                 -- source INSTANCE id ([sources.<id>] config key)
  started_unix  INTEGER NOT NULL,
  finished_unix INTEGER,
  status        TEXT NOT NULL,          -- "running" | "ok" | "error"
  error         TEXT NOT NULL DEFAULT '',
  item_count    INTEGER NOT NULL DEFAULT 0,
  notice        TEXT NOT NULL DEFAULT ''  -- non-fatal advisory recorded alongside an otherwise-successful run (12-09-PLAN.md, G-12-1/G-12-3) — distinct from the error column, never written by a genuine sync failure
);

CREATE INDEX IF NOT EXISTS idx_items_chrono
  ON items(timestamp_unix DESC, secondary_timestamp_unix DESC, id ASC);

-- External-content FTS5 index over items(title, preview) (KERN-05).
-- items is a normal rowid table, so content_rowid='rowid' refers to its
-- hidden rowid column, which the TEXT PRIMARY KEY id does not alias —
-- joining items.rowid = items_fts.rowid recovers each row's real id.
-- Verified end-to-end against this repo's pinned modernc.org/sqlite
-- dependency (03-RESEARCH.md Pattern 3).
CREATE VIRTUAL TABLE IF NOT EXISTS items_fts USING fts5(
  title, preview, content='items', content_rowid='rowid'
);

-- Three triggers keep items_fts in sync with every write to items made
-- through UpsertItems/ReplaceWebspaceSourceItems, with no application-level
-- index maintenance. They only fire on writes made AFTER they exist —
-- Store.Open's first-creation backfill ('rebuild') is what makes rows
-- written before this schema addition searchable too.
CREATE TRIGGER IF NOT EXISTS items_ai AFTER INSERT ON items BEGIN
  INSERT INTO items_fts(rowid, title, preview) VALUES (new.rowid, new.title, new.preview);
END;

CREATE TRIGGER IF NOT EXISTS items_ad AFTER DELETE ON items BEGIN
  INSERT INTO items_fts(items_fts, rowid, title, preview) VALUES('delete', old.rowid, old.title, old.preview);
END;

CREATE TRIGGER IF NOT EXISTS items_au AFTER UPDATE ON items BEGIN
  INSERT INTO items_fts(items_fts, rowid, title, preview) VALUES('delete', old.rowid, old.title, old.preview);
  INSERT INTO items_fts(rowid, title, preview) VALUES (new.rowid, new.title, new.preview);
END;
`
