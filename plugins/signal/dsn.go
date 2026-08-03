package main

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

// openReadOnly opens Signal Desktop's db.sqlite strictly read-only via a
// mode=ro URI DSN, using rawHexKey (already resolved by resolveKey) as
// the SQLCipher raw key, and performs one trivial read (PRAGMA
// user_version) before returning, so a wrong key surfaces here as a
// clear error rather than a confusing failure deep inside a later query
// (04-RESEARCH.md Pitfall 4 — AES/SQLCipher decryption with the wrong
// key does not error, it silently produces garbage).
//
// Deliberately NOT adding &immutable=1: Signal Desktop is a live
// concurrent writer (journal_mode=WAL) whenever it's running —
// immutable=1 tells SQLite the file will never change and disables its
// own change-detection/locking, which risks stale or torn reads exactly
// in the "Signal Desktop running at the same time" case this phase's
// own success criteria name explicitly.
//
// DSN parameter names: this driver (github.com/mattn/go-sqlite3's
// SQLiteDriver.Open, as vendored by the Task 1-authorised
// jgiannuzzi/go-sqlite3 fork — see go.mod's replace directive) accepts
// "_key=X" and "_cipher_page_size=X", NOT "_pragma_key"/
// "_pragma_cipher_page_size". This diverges from 04-RESEARCH.md's
// illustrative DSN snippet, which was written against a different
// driver's (mutecomm/go-sqlcipher's) DSN convention before Task 1
// selected the current one; the parameter names below were confirmed
// directly against this machine's real, live db.sqlite during this
// task's own schema-introspection step (04-01-SUMMARY.md records this
// deviation).
//
// The key value MUST use the SQLCipher raw-key hex-literal form
// (x'<hex>'), never a bare hex string: SQLCipher treats an unquoted
// string as a passphrase and runs it through its own key-derivation
// function, which silently derives the WRONG key from an already-raw key
// like Signal's — this is exactly the "decrypts to garbage instead of
// erroring" failure mode 04-RESEARCH.md Pitfall 4 warns about, confirmed
// hands-on: the unquoted form failed with "file is not a database" while
// the x'...' form opened correctly against the same real key.
func openReadOnly(dbPath, rawHexKey string) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"file:%s?mode=ro&_key=x'%s'&_cipher_page_size=4096",
		dbPath, rawHexKey,
	)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("signal: open %s read-only: %w", dbPath, err)
	}
	db.SetMaxOpenConns(1)

	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		db.Close()
		return nil, fmt.Errorf("signal: verify key by reading schema version: %w", err)
	}

	return db, nil
}
