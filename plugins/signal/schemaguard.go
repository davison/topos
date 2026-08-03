package main

import (
	"database/sql"
	"fmt"
)

// highestSupportedSchemaVersion is the highest Signal Desktop database
// schema PRAGMA user_version this plugin has been built and tested
// against. Read directly off a real, live db.sqlite during this task's
// own schema-introspection step (PRAGMA user_version = 1730) — NOT
// carried over from 04-RESEARCH.md's illustrative "1640" snippet, which
// its own doc comment flagged as never independently confirmed against a
// real install. Raising this constant is a deliberate act, performed
// only after re-verifying the digest/matching logic against a newer
// Signal Desktop release's actual schema — never bumped speculatively.
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
