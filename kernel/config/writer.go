package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// CanonicalHeader is prepended to every canonically-rewritten config.toml
// (D-02: minimal + header pointer — no per-key generated doc comments).
// Exactly two comment lines followed by a blank line, matching
// 07-CONTEXT.md D-02 verbatim.
const CanonicalHeader = "# managed by the topos UI — hand-edits are honored via Reload\n# see config.example.toml for full field documentation\n\n"

// BackupSuffix is appended to path to name the single rolling backup
// WriteCanonical writes before every save (D-04: one backup file,
// overwritten each time — never a timestamped set or a backup directory).
const BackupSuffix = ".bak"

// WriteCanonical serializes rawCfg — and ONLY the raw, pre-expansion form
// of a config (D-05's hard requirement; see config.go's LoadRaw doc
// comment) — as canonical TOML and writes it to path.
//
// rawCfg must never be the expanded runtime *Config: marshalling that form
// would write a resolved secret VALUE into config.toml where an operator
// authored "${VAR}" — a privacy breach D-05 treats as one-way (it cannot be
// un-shipped once it happens even once). Every caller of this function is
// responsible for passing config.Store.Raw() or an equivalent raw-parsed
// document, never Store.Expanded().
//
// Write order (D-01, D-04): marshal rawCfg deterministically (go-toml/v2's
// Marshal already sorts map keys), prepend CanonicalHeader, copy the
// CURRENT file at path to path+BackupSuffix when one exists (overwriting
// any previous backup — a single rolling backup, never a set), then write
// the new content via a same-directory temp file + fsync + rename so a
// kernel killed mid-write can never leave path truncated or half-written —
// the temp file is either fully renamed into place or never touched.
func WriteCanonical(path string, rawCfg *Config) error {
	body, err := toml.Marshal(rawCfg)
	if err != nil {
		return fmt.Errorf("config: marshal canonical form: %w", err)
	}
	out := append([]byte(CanonicalHeader), body...)

	dir := filepath.Dir(path)

	if existing, err := os.ReadFile(path); err == nil {
		if err := os.WriteFile(path+BackupSuffix, existing, 0o600); err != nil {
			return fmt.Errorf("config: write backup %s: %w", path+BackupSuffix, err)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("config: read %s for backup: %w", path, err)
	}

	tmp, err := os.CreateTemp(dir, ".config-*.toml")
	if err != nil {
		return fmt.Errorf("config: create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // no-op once the rename below succeeds

	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		return fmt.Errorf("config: write temp file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return fmt.Errorf("config: fsync temp file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("config: close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("config: rename into place: %w", err)
	}
	return nil
}
