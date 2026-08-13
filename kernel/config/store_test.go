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

// TestStore_Save_RecursiveKeyOmittedWhenNeverDeclaredPreservedWhenTrue is
// the load-bearing proof for 12-03-PLAN.md Task 1's omitempty discipline:
// a canonical rewrite of a config whose sources never declared `recursive`
// must not introduce the key into any source block, and a rewrite of a
// source that DID declare it true must preserve it.
func TestStore_Save_RecursiveKeyOmittedWhenNeverDeclaredPreservedWhenTrue(t *testing.T) {
	path := writeTempConfig(t, `
[sources.docs-flat]
plugin = "topos-plugin-filesystem"
path = "/mnt/docs-flat"

[sources.docs-nested]
plugin = "topos-plugin-filesystem"
path = "/mnt/docs-nested"
recursive = true

[webspaces.house-move]
keywords = ["house-move"]
`)
	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	// Save a config that changes something UNRELATED to recursion (a
	// webspace's filter) — the sources block, including each source's
	// (absent or present) recursive key, is carried through untouched.
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
	doc := string(onDisk)

	if !strings.Contains(doc, "recursive = true") {
		t.Errorf("expected docs-nested's recursive = true to survive the canonical rewrite, got: %s", doc)
	}

	reopened, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore (reload): %v", err)
	}
	if !reopened.Raw().Sources["docs-nested"].Recursive {
		t.Errorf("expected docs-nested.Recursive == true after reload, got false")
	}
	if reopened.Raw().Sources["docs-flat"].Recursive {
		t.Errorf("expected docs-flat.Recursive == false after reload (key never declared), got true")
	}

	// docs-flat's own on-disk block must carry no recursive key at all —
	// proving the canonical rewrite doesn't spuriously introduce the key
	// into a block that never declared it.
	flatBlockStart := strings.Index(doc, "[sources.docs-flat]")
	if flatBlockStart == -1 {
		t.Fatalf("expected a [sources.docs-flat] block, got: %s", doc)
	}
	flatBlock := doc[flatBlockStart:]
	if nextBlockOffset := strings.Index(doc[flatBlockStart+1:], "[sources."); nextBlockOffset != -1 {
		flatBlock = doc[flatBlockStart : flatBlockStart+1+nextBlockOffset]
	}
	if strings.Contains(flatBlock, "recursive") {
		t.Errorf("expected no recursive key in docs-flat's block, got: %s", flatBlock)
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

// TestSave_CreateWebspaceThenAddFirstSource_RoundTrips is the proof
// 07-UAT.md G-07-3.missing[1] names as absent (07-11-PLAN.md Task 3): 07-03's
// addWebspace() test (web/src/lib/config-edit.test.ts) asserted only the
// shape of a JavaScript object and never submitted it to a real
// config.Validate — which is why a defect that made UI webspace creation
// impossible on every real installation reached UAT undetected.
// Store.Save is the closest reachable stand-in for the live PUT
// /api/config: ConfigSaveHandler's own doc comment (kernel/httpapi/config.go)
// names Save as the single place every rule up to and including the write
// lives.
//
// Seeds a temp config.toml carrying two [sources.*] blocks and one
// fully-populated webspace — the shape of a real installation, since the
// defect was invisible on a config with no sources configured at all —
// then, in sequence, asserting after each step:
//
//  1. Adds a webspace whose Keywords/Sources/Match are all present but
//     empty — the literal document addWebspace() produces — and saves it
//     with the store's current hash.
//  2. Re-opens a second Store over the same path and asserts the new
//     webspace is present in its loaded config and is still an empty
//     shell. This is the half a pure-shape test cannot do: it proves
//     WriteCanonical emitted a [webspaces.<name>] table for a webspace
//     with no keys, that the reload parsed it back, and that LoadRaw's
//     own Validate call accepted it on the way in. A shell silently
//     dropped by the canonical rewrite would fail here, not at the save.
//  3. From the reloaded store, adds the first source to that webspace —
//     an explicit allowlist naming exactly one configured instance, plus
//     a match block for that same instance — and saves with the reloaded
//     store's hash.
//  4. Re-opens once more and asserts the webspace now carries exactly
//     that one instance in its allowlist and exactly that one match
//     block.
func TestSave_CreateWebspaceThenAddFirstSource_RoundTrips(t *testing.T) {
	t.Setenv("TEST_RT_URL", "http://x.lan")
	t.Setenv("TEST_RT_TOKEN", "tok")
	path := writeTempConfig(t, `
[sources.home-email]
plugin = "topos-plugin-proton"
base_url = "${TEST_RT_URL}"
token = "${TEST_RT_TOKEN}"

[sources.work-email]
plugin = "topos-plugin-proton"
base_url = "${TEST_RT_URL}"
token = "${TEST_RT_TOKEN}"

[webspaces.house-move]
keywords = ["house-move"]
`)

	s, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	// Step 1: create the empty shell — the literal document
	// web/src/lib/config-edit.ts's addWebspace() PUTs as the create-
	// webspace modal's first write.
	next := *s.Raw()
	next.Webspaces = map[string]Webspace{
		"house-move":  next.Webspaces["house-move"],
		"new-project": {Keywords: []string{}, Sources: []string{}, Match: map[string]MatchBlock{}},
	}
	if err := s.Save(&next, s.Hash()); err != nil {
		t.Fatalf("Save (create empty webspace shell): expected the exact document addWebspace() produces to be accepted — a rejection here means 07-UAT.md G-07-3 (\"declares neither a keywords fallback nor any match block\") has regressed, got: %v", err)
	}

	// Step 2: reload from disk and confirm the shell survived the
	// canonical write/reload round trip, not just accepted in memory.
	reopened, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore (reload after creating shell): %v", err)
	}
	shell, ok := reopened.Raw().Webspaces["new-project"]
	if !ok {
		t.Fatal("expected the newly created webspace to survive the canonical write/reload round trip, but it was absent")
	}
	if !shell.IsEmptyShell() {
		t.Errorf("expected the reloaded webspace to still be an empty shell, got %+v", shell)
	}

	// Step 3: add the first source — an explicit allowlist naming exactly
	// one instance, plus a match block for it.
	next2 := *reopened.Raw()
	next2.Webspaces = map[string]Webspace{
		"house-move": next2.Webspaces["house-move"],
		"new-project": {
			Keywords: []string{},
			Sources:  []string{"home-email"},
			Match: map[string]MatchBlock{
				"home-email": {"folders": {"Home"}},
			},
		},
	}
	if err := reopened.Save(&next2, reopened.Hash()); err != nil {
		t.Fatalf("Save (add first source to the freshly created webspace): a rejection here means the allowlist was seeded with every configured instance and validateFallbackCoverage found uncovered participants — the follow-on defect Task 2 fixes, asserted here at the document level, got: %v", err)
	}

	// Step 4: reload once more and confirm exactly that one instance and
	// match block persisted.
	final, err := NewStore(path)
	if err != nil {
		t.Fatalf("NewStore (reload after adding first source): %v", err)
	}
	composed, ok := final.Raw().Webspaces["new-project"]
	if !ok {
		t.Fatal("expected the composed webspace to survive the final round trip, but it was absent")
	}
	if len(composed.Sources) != 1 || composed.Sources[0] != "home-email" {
		t.Errorf("expected the reloaded webspace's sources allowlist to name exactly [\"home-email\"], got %+v", composed.Sources)
	}
	if len(composed.Match) != 1 {
		t.Errorf("expected exactly one match block, got %+v", composed.Match)
	}
	if got := composed.Match["home-email"]["folders"]; len(got) != 1 || got[0] != "Home" {
		t.Errorf("expected Match[\"home-email\"][\"folders\"] == [\"Home\"], got %+v", got)
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
