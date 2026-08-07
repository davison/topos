// writer_test.go pins WriteCanonical's own load-bearing guarantees
// (07-01-PLAN.md Task 3): the single rolling backup (D-04), the atomic
// write (D-01 — no leftover temp file, and the written file always starts
// with CanonicalHeader), and the marshal-write-reload-write fixed point
// (idempotency) that makes a save-of-a-save a genuine no-op on disk.
package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWriteCanonical_BackupHoldsPreviousContentAndOverwritesOnSecondWrite
// is the load-bearing proof for D-04 (single rolling backup, never a
// timestamped set) plus D-01's atomic-write guarantee: after a first
// WriteCanonical over an existing file, the sibling .bak holds that
// PREVIOUS file's content byte for byte; after a SECOND WriteCanonical,
// the .bak now holds the first write's output (overwritten in place, not
// a second file); the directory holds exactly config.toml and
// config.toml.bak — no leftover os.CreateTemp artifact — and the written
// file always starts with CanonicalHeader.
func TestWriteCanonical_BackupHoldsPreviousContentAndOverwritesOnSecondWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	preWriteContent := []byte("# a hand-authored file, not yet canonical\n")
	if err := os.WriteFile(path, preWriteContent, 0o600); err != nil {
		t.Fatalf("seed initial file: %v", err)
	}

	first := &Config{Webspaces: map[string]Webspace{
		"house-move": {Keywords: []string{"house-move"}},
	}}
	if err := WriteCanonical(path, first); err != nil {
		t.Fatalf("WriteCanonical (first): %v", err)
	}

	backupPath := path + BackupSuffix
	backup1, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("read backup after first write: %v", err)
	}
	if string(backup1) != string(preWriteContent) {
		t.Fatalf("expected the backup to hold the PREVIOUS file's content byte for byte:\ngot=%s\nwant=%s", backup1, preWriteContent)
	}

	afterFirst, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config after first write: %v", err)
	}
	if !strings.HasPrefix(string(afterFirst), CanonicalHeader) {
		t.Fatalf("expected the written file to start with CanonicalHeader, got: %s", afterFirst)
	}

	second := &Config{Webspaces: map[string]Webspace{
		"house-move": {Keywords: []string{"house-move"}, Filter: []string{"boiler"}},
	}}
	if err := WriteCanonical(path, second); err != nil {
		t.Fatalf("WriteCanonical (second): %v", err)
	}

	backup2, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("read backup after second write: %v", err)
	}
	if string(backup2) != string(afterFirst) {
		t.Fatalf("expected the SECOND write's backup to hold the FIRST write's output (overwritten in place, a single rolling backup, never a second file):\ngot=%s\nwant=%s", backup2, afterFirst)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	if len(entries) != 2 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("expected exactly 2 files in the directory (config.toml and config.toml.bak), got %d: %v", len(entries), names)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".config-") {
			t.Errorf("found a leftover os.CreateTemp artifact: %q", e.Name())
		}
	}
}

// TestWriteCanonical_MarshalWriteReloadWriteIsAFixedPoint is the
// idempotency proof Task 3 requires: marshalling a loaded raw config,
// writing it, reloading it from disk, and writing it again must produce
// byte-identical output both times — a save-of-a-save is a true no-op,
// never a slow drift in formatting or field order across repeated saves.
func TestWriteCanonical_MarshalWriteReloadWriteIsAFixedPoint(t *testing.T) {
	t.Setenv("PAPERLESS_TOKEN", "test-token-value")

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")

	original := `
[sources.paperless]
plugin = "topos-plugin-paperless"
base_url = "http://paperless.lan:8000"
token = "${PAPERLESS_TOKEN}"

[webspaces.house-move]
keywords = ["house-move", "House"]
filter = ["boiler", "quote"]
`
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatalf("seed initial file: %v", err)
	}

	_, raw1, _, _, err := LoadRaw(path)
	if err != nil {
		t.Fatalf("LoadRaw (first): %v", err)
	}
	if err := WriteCanonical(path, raw1); err != nil {
		t.Fatalf("WriteCanonical (first): %v", err)
	}
	firstWrite, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after first write: %v", err)
	}

	_, raw2, _, _, err := LoadRaw(path)
	if err != nil {
		t.Fatalf("LoadRaw (reload): %v", err)
	}
	if err := WriteCanonical(path, raw2); err != nil {
		t.Fatalf("WriteCanonical (second): %v", err)
	}
	secondWrite, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after second write: %v", err)
	}

	if string(firstWrite) != string(secondWrite) {
		t.Fatalf("expected marshal->write->reload->write to be a fixed point (byte-identical output):\nfirst=%s\nsecond=%s", firstWrite, secondWrite)
	}
}
