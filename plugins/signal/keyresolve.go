package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// signalConfig is the shape of Signal Desktop's own config.json that this
// plugin needs to resolve the SQLCipher decryption key — Signal Desktop's
// own schema, not this project's. Exactly one of Key or
// EncryptedKey+SafeStorageBackend is present on any real install
// (04-RESEARCH.md Pattern 1); the field names below match the file
// verbatim, confirmed by direct inspection of a real, live config.json
// during this task's own schema-introspection step (field NAMES only —
// this plugin never logs a value read from this struct).
type signalConfig struct {
	Key                string `json:"key,omitempty"`
	EncryptedKey       string `json:"encryptedKey,omitempty"`
	SafeStorageBackend string `json:"safeStorageBackend,omitempty"`
}

// errSafeStorageUnsupported is returned when config.json carries the
// encryptedKey/safeStorageBackend shape — the modern Electron safeStorage
// wrap. Resolving it requires a freedesktop Secret Service D-Bus
// round-trip plus an AES-128-CBC/PBKDF2 unwrap, completed in plan 04-02
// (04-01-PLAN.md Task 2's own scope note: "a functionality gap, not an
// architectural one" — this branch point and its call site already exist
// here).
var errSafeStorageUnsupported = fmt.Errorf("signal: config.json carries the safeStorage-wrapped key shape (encryptedKey/safeStorageBackend) — resolving it is not implemented in this build (see plan 04-02)")

// readSignalConfig reads and parses configPath (Signal Desktop's own
// config.json). Never logs configPath's contents.
func readSignalConfig(configPath string) (signalConfig, error) {
	raw, err := os.ReadFile(configPath)
	if err != nil {
		return signalConfig{}, fmt.Errorf("signal: read %s: %w", configPath, err)
	}
	var cfg signalConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return signalConfig{}, fmt.Errorf("signal: parse %s: %w", configPath, err)
	}
	return cfg, nil
}

// resolveKey branches on which of cfg.Key / cfg.EncryptedKey is present
// (04-RESEARCH.md Pattern 1 — "dual-shape key resolution, branch on field
// presence, never assume"). Neither present, or both present, is the same
// fail-loud case — reported by field PRESENCE only, never by value.
func resolveKey(cfg signalConfig) (rawHexKey string, err error) {
	hasKey := cfg.Key != ""
	hasEncrypted := cfg.EncryptedKey != ""

	switch {
	case hasKey && !hasEncrypted:
		// Legacy, unmigrated install — the key IS the raw hex SQLCipher
		// key already, no unwrap needed. Confirmed the live/current shape
		// on this machine's real Signal Desktop install (04-RESEARCH.md
		// finding 2).
		return cfg.Key, nil
	case hasEncrypted && !hasKey:
		return "", errSafeStorageUnsupported
	default:
		return "", fmt.Errorf("signal: config.json has an unrecognized key shape (key present=%v, encryptedKey present=%v) — refusing to guess", hasKey, hasEncrypted)
	}
}
