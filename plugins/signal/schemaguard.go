package main

import (
	"database/sql"
	"fmt"
)

// highestSupportedSchemaVersion is the highest Signal Desktop database
// schema PRAGMA user_version this plugin has been built and tested
// against. Read directly off a real, live db.sqlite (PRAGMA user_version
// = 1730) running Signal Desktop 8.21.0 (Arch package signal-desktop
// 8.21.0-1), verified 2026-08-03 — NOT carried over from
// 04-RESEARCH.md's illustrative "1640" snippet, which its own doc
// comment flagged as never independently confirmed against a real
// install. Raising this constant is a deliberate act, performed only
// after re-verifying the digest/matching logic against a newer Signal
// Desktop release's actual schema (re-run the same PRAGMA
// user_version / table_info introspection this value was pinned from,
// against that newer release's real database) — never bumped
// speculatively, and never in response to a single failing sync alone.
const highestSupportedSchemaVersion = 1730

// guardSchemaVersion reads PRAGMA user_version on db and fails loudly,
// naming both the version found and the highest supported, if it exceeds
// highestSupportedSchemaVersion. Must be called before any query against
// messages/conversations (04-RESEARCH.md Pattern 2).
func guardSchemaVersion(db *sql.DB) error {
	var found int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&found); err != nil {
		return fmt.Errorf("signal: read schema version: %w", err)
	}
	if found > highestSupportedSchemaVersion {
		return fmt.Errorf(
			"signal: unrecognised database schema version %d (this plugin was built against up to %d) — refusing to import, not silently skipping",
			found, highestSupportedSchemaVersion,
		)
	}
	return nil
}
