package main

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"go.mau.fi/whatsmeow/store/sqlstore"
)

// TestWhatsmeowSessionDSN_EnablesForeignKeys proves the DSN both link.go
// and connect.go build for whatsmeow's own sqlstore carries the
// modernc.org/sqlite pragma syntax (`_pragma=foreign_keys(1)`) — NOT the
// `_foreign_keys=on` shorthand whatsmeow's own doc comment illustrates,
// which is a DIFFERENT sqlite driver's DSN convention and is silently
// ignored by modernc.org/sqlite, leaving foreign keys off and
// sqlstore.Container.Upgrade refusing to run (observed live, 2026-08-10:
// "failed to upgrade database: foreign keys are not enabled").
func TestWhatsmeowSessionDSN_EnablesForeignKeys(t *testing.T) {
	dsn := whatsmeowSessionDSN("/tmp/example/whatsmeow.db")

	if !strings.Contains(dsn, "_pragma=foreign_keys(1)") {
		t.Fatalf("whatsmeowSessionDSN() = %q, want it to contain modernc.org/sqlite's _pragma=foreign_keys(1) syntax", dsn)
	}
	if strings.Contains(dsn, "_foreign_keys=on") {
		t.Fatalf("whatsmeowSessionDSN() = %q, contains the WRONG (mattn/go-sqlite3-style) _foreign_keys=on shorthand, which modernc.org/sqlite silently ignores", dsn)
	}
}

// TestWhatsmeowSessionDSN_MigrationsRunAgainstRealSQLStore actually opens
// whatsmeow's own sqlstore.New against a fresh temp-file database using
// whatsmeowSessionDSN — the same call link.go's runLinkCLI and
// connect.go's startBackgroundClient both make. This is the regression
// test for the live failure a real -link run hit: a wrong DSN fails HERE,
// at Container.Upgrade's own foreign-keys precondition check, without
// needing a phone or a network connection to reproduce.
func TestWhatsmeowSessionDSN_MigrationsRunAgainstRealSQLStore(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "whatsmeow.db")

	container, err := sqlstore.New(context.Background(), "sqlite", whatsmeowSessionDSN(dbPath), newPluginLogger("whatsmeow/test"))
	if err != nil {
		t.Fatalf("sqlstore.New with whatsmeowSessionDSN: %v (this is exactly the failure mode a wrong DSN produces: 'failed to upgrade database: foreign keys are not enabled')", err)
	}
	defer container.Close()

	// GetFirstDevice on a brand-new store creates and persists a fresh,
	// unlinked device row — proving the migrated schema is actually
	// usable, not just that Upgrade returned nil.
	if _, err := container.GetFirstDevice(context.Background()); err != nil {
		t.Fatalf("GetFirstDevice against freshly migrated store: %v", err)
	}
}
