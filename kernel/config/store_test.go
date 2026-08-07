// store_test.go pins the load-bearing invariants config.Store's Save
// method exists to guarantee (07-01-PLAN.md Task 3): the secret round trip
// (D-05), the content-hash clobber guard (D-03), and the lossless-rewrite
// prohibition (D-01, "flattens comments only", never data). Uses the same
// table-driven-adjacent style, writeTempConfig fixture, and t.Setenv
// convention config_test.go already establishes in this package.
package config

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

// TestStore_Save_SecretRoundTrip_NeverPersistsResolvedValue is the
// load-bearing proof for D-05 (T-07-01/T-07-02): a source's token is
// authored as a literal ${VAR} reference; after NewStore resolves it in
// memory and a Save changes something unrelated (a webspace's filter),
// the on-disk bytes, and Store.Raw() itself, must still carry the literal
// reference — never the resolved secret value. Store.Expanded() DOES hold
// the resolved value, which is what proves this is a genuine split rather
// than the variable having simply gone unset (a test that never checked
// Expanded() could pass vacuously against a build that broke expansion
// entirely).
func TestStore_Save_SecretRoundTrip_NeverPersistsResolvedValue(t *testing.T) {
	t.Setenv("TEST_STORE_TOKEN_SENTINEL", "sentinel-secret-value")

	path := writeTempConfig(t, `
[sources.paperless]
plugin = "topos-plugin-paperless"
base_url = "http://paperless.lan:8000"
token = "${TEST_STORE_TOKEN_SENTINEL}"

[webspaces.house-move]
keywords = ["house-move"]
`)

	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	if got := s.Expanded().Sources["paperless"].Token; got != "sentinel-secret-value" {
		t.Fatalf("expected Store.Expanded() to hold the RESOLVED secret value (proving the split, not just an unset variable), got %q", got)
	}
	if got := s.Raw().Sources["paperless"].Token; got != "${TEST_STORE_TOKEN_SENTINEL}" {
		t.Fatalf("expected Store.Raw() to hold the literal ${VAR} reference before any save, got %q", got)
	}

	// Save a config that changes something UNRELATED to the secret (a
	// webspace's filter) — the sources block, including its token
	// reference, is carried through untouched.
	next := *s.Raw()
	next.Webspaces = map[string]Webspace{
		"house-move": {Keywords: []string{"house-move"}, Filter: []string{"boiler"}},
	}

	if err := s.Save(&next, s.Hash()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	onDisk, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config after save: %v", err)
	}
	if strings.Contains(string(onDisk), "sentinel-secret-value") {
		t.Fatalf("config.toml on disk contains the RESOLVED secret value — D-05 violated: %s", onDisk)
	}
	if !strings.Contains(string(onDisk), `${TEST_STORE_TOKEN_SENTINEL}`) {
		t.Fatalf("expected the literal ${VAR} reference to survive the save, got: %s", onDisk)
	}

	if got := s.Raw().Sources["paperless"].Token; got != "${TEST_STORE_TOKEN_SENTINEL}" {
		t.Errorf("expected Store.Raw() to still hold the literal reference after save, got %q", got)
	}
	if got := s.Expanded().Sources["paperless"].Token; got != "sentinel-secret-value" {
		t.Errorf("expected Store.Expanded() to still hold the resolved value after save (the running config), got %q", got)
	}
}

// TestStore_Save_ClobberGuard_StaleHashRejectedFileUnchanged is the
// load-bearing proof for D-03: a Save carrying a base_hash that no longer
// matches the file's current on-disk hash must be rejected outright — no
// merge, no partial write — leaving the file exactly as the out-of-band
// edit left it.
func TestStore_Save_ClobberGuard_StaleHashRejectedFileUnchanged(t *testing.T) {
	path := writeTempConfig(t, `
[webspaces.house-move]
keywords = ["house-move"]
`)
	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	staleHash := s.Hash()

	outOfBand := []byte(`
[webspaces.house-move]
keywords = ["changed-out-of-band"]
`)
	if err := os.WriteFile(path, outOfBand, 0o600); err != nil {
		t.Fatalf("simulate an out-of-band edit behind the store's back: %v", err)
	}

	next := &Config{Webspaces: map[string]Webspace{
		"house-move": {Keywords: []string{"house-move"}, Filter: []string{"boiler"}},
	}}

	err = s.Save(next, staleHash)
	if !errors.Is(err, ErrConfigChangedOnDisk) {
		t.Fatalf("expected errors.Is(err, ErrConfigChangedOnDisk), got: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config after rejected save: %v", err)
	}
	if !bytes.Equal(got, outOfBand) {
		t.Fatalf("expected the rejected save to leave NO TRACE — file must be exactly the out-of-band edit's content:\ngot=%s\nwant=%s", got, outOfBand)
	}
}

// TestStore_Save_UnknownKeysRejectedFileUnchanged is the load-bearing
// proof for the lossless-rewrite prohibition (D-01's "flattens comments
// only", never data): a config.toml carrying a table or key the Config
// struct does not model must refuse EVERY save, naming the offending
// key(s), and must leave the file byte-for-byte unchanged — a canonical
// rewrite that silently dropped it would be undetectable data loss.
func TestStore_Save_UnknownKeysRejectedFileUnchanged(t *testing.T) {
	original := `
[webspaces.house-move]
keywords = ["house-move"]

[a_stray_table]
some_key = "some_value"
`
	path := writeTempConfig(t, original)

	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	next := &Config{Webspaces: map[string]Webspace{
		"house-move": {Keywords: []string{"house-move"}, Filter: []string{"boiler"}},
	}}

	err = s.Save(next, s.Hash())
	if !errors.Is(err, ErrConfigHasUnknownKeys) {
		t.Fatalf("expected errors.Is(err, ErrConfigHasUnknownKeys), got: %v", err)
	}
	if !strings.Contains(err.Error(), "a_stray_table") {
		t.Errorf("expected the error to name the unrecognised key \"a_stray_table\", got: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config after rejected save: %v", err)
	}
	if string(got) != original {
		t.Fatalf("expected the rejected save to leave the file byte-for-byte unchanged:\ngot=%s\nwant=%s", got, original)
	}
}

// TestStore_Save_NoUnknownKeysSucceeds is the non-vacuous sibling of the
// guard above: a config with no unrecognised keys must actually be able
// to save — proving the guard rejects the specific defect it exists for,
// not saves in general.
func TestStore_Save_NoUnknownKeysSucceeds(t *testing.T) {
	path := writeTempConfig(t, `
[webspaces.house-move]
keywords = ["house-move"]
`)
	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	next := &Config{Webspaces: map[string]Webspace{
		"house-move": {Keywords: []string{"house-move"}, Filter: []string{"boiler"}},
	}}

	if err := s.Save(next, s.Hash()); err != nil {
		t.Fatalf("expected Save to succeed for a config with no unrecognised keys, got: %v", err)
	}
	if got := s.Raw().Webspaces["house-move"].Filter; len(got) != 1 || got[0] != "boiler" {
		t.Errorf("expected the save to have applied, got filter %v", got)
	}
}
