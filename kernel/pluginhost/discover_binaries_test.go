package pluginhost

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFixtureFile(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o755); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}
}

// TestDiscoverBinaries_MixedDirectoryReturnsOnlyPrefixedRegularFilesSorted
// proves the picker's plugin-type source: a directory of prefixed
// binaries, an unrelated file, a subdirectory that happens to carry the
// prefix, and a non-prefixed file all coexist — only the prefixed regular
// files are returned, sorted.
func TestDiscoverBinaries_MixedDirectoryReturnsOnlyPrefixedRegularFilesSorted(t *testing.T) {
	dir := t.TempDir()
	writeFixtureFile(t, dir, "topos-plugin-silverbullet")
	writeFixtureFile(t, dir, "topos-plugin-paperless")
	writeFixtureFile(t, dir, "README.md")
	writeFixtureFile(t, dir, "topos-plugin-proton")
	if err := os.Mkdir(filepath.Join(dir, "topos-plugin-not-a-file"), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}

	got, err := DiscoverBinaries(dir)
	if err != nil {
		t.Fatalf("DiscoverBinaries: %v", err)
	}

	want := []string{"topos-plugin-paperless", "topos-plugin-proton", "topos-plugin-silverbullet"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("expected sorted order %v, got %v", want, got)
			break
		}
	}
}

// TestDiscoverBinaries_ExcludesMockBinary proves the mock reference
// fixture is never offered even though it is discovered by the identical
// naming convention every real plugin uses (07-RESEARCH.md Open Question
// 1).
func TestDiscoverBinaries_ExcludesMockBinary(t *testing.T) {
	dir := t.TempDir()
	writeFixtureFile(t, dir, "topos-plugin-mock")
	writeFixtureFile(t, dir, "topos-plugin-paperless")

	got, err := DiscoverBinaries(dir)
	if err != nil {
		t.Fatalf("DiscoverBinaries: %v", err)
	}
	for _, name := range got {
		if name == "topos-plugin-mock" {
			t.Fatalf("expected topos-plugin-mock to be excluded, got %v", got)
		}
	}
	if len(got) != 1 || got[0] != "topos-plugin-paperless" {
		t.Fatalf("expected exactly [\"topos-plugin-paperless\"], got %v", got)
	}
}

// TestDiscoverBinaries_MissingDirectoryReturnsEmptyNilError proves an
// operator with no plugins installed yet is a legitimate state, not an
// error.
func TestDiscoverBinaries_MissingDirectoryReturnsEmptyNilError(t *testing.T) {
	got, err := DiscoverBinaries(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("expected a nil error for a missing directory, got: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected an empty slice, got %v", got)
	}
}

// TestDiscoverBinaries_SymlinkedRegularFileIsDiscovered proves a
// symlinked plugin binary is discovered exactly like a real regular
// file — the browser E2E harness's own fixture
// (web/e2e/fixtures/plugin-binaries.ts) populates its temp plugins
// directory this way, and a naive os.DirEntry.Type().IsRegular() check
// (which reflects an Lstat, never following the link) would silently
// exclude it. The fixture name is deliberately a NON-excluded binary
// (topos-plugin-silverbullet, not topos-plugin-mockstrict) so this test
// proves symlink following rather than accidentally proving exclusion.
func TestDiscoverBinaries_SymlinkedRegularFileIsDiscovered(t *testing.T) {
	realDir := t.TempDir()
	writeFixtureFile(t, realDir, "topos-plugin-silverbullet")

	dir := t.TempDir()
	if err := os.Symlink(filepath.Join(realDir, "topos-plugin-silverbullet"), filepath.Join(dir, "topos-plugin-silverbullet")); err != nil {
		t.Fatalf("symlink fixture: %v", err)
	}

	got, err := DiscoverBinaries(dir)
	if err != nil {
		t.Fatalf("DiscoverBinaries: %v", err)
	}
	if len(got) != 1 || got[0] != "topos-plugin-silverbullet" {
		t.Fatalf("expected exactly [\"topos-plugin-silverbullet\"], got %v", got)
	}
}

// TestDiscoverBinaries_BrokenSymlinkIsSkippedNotAnError proves a dangling
// symlink (target removed) is silently skipped rather than failing the
// whole discovery call.
func TestDiscoverBinaries_BrokenSymlinkIsSkippedNotAnError(t *testing.T) {
	dir := t.TempDir()
	if err := os.Symlink(filepath.Join(dir, "topos-plugin-does-not-exist-target"), filepath.Join(dir, "topos-plugin-broken")); err != nil {
		t.Fatalf("symlink fixture: %v", err)
	}
	writeFixtureFile(t, dir, "topos-plugin-paperless")

	got, err := DiscoverBinaries(dir)
	if err != nil {
		t.Fatalf("expected a nil error for a broken symlink entry, got: %v", err)
	}
	if len(got) != 1 || got[0] != "topos-plugin-paperless" {
		t.Fatalf("expected exactly [\"topos-plugin-paperless\"] (broken symlink skipped), got %v", got)
	}
}

// TestDiscoverBinaries_SymlinkToDirectoryIsExcluded proves a symlink
// pointing at a directory is excluded exactly like a real subdirectory —
// os.Stat's Mode().IsRegular() is false for a directory regardless of
// whether it was reached via a symlink.
func TestDiscoverBinaries_SymlinkToDirectoryIsExcluded(t *testing.T) {
	realDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(realDir, "topos-plugin-a-directory"), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}

	dir := t.TempDir()
	if err := os.Symlink(filepath.Join(realDir, "topos-plugin-a-directory"), filepath.Join(dir, "topos-plugin-a-directory")); err != nil {
		t.Fatalf("symlink fixture: %v", err)
	}

	got, err := DiscoverBinaries(dir)
	if err != nil {
		t.Fatalf("DiscoverBinaries: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected an empty slice (symlink-to-directory excluded), got %v", got)
	}
}

// TestDiscoverBinaries_ExcludesMockstrictBinary proves the mockstrict
// harness-fixture binary is never offered even though it is discovered by
// the identical naming convention every real plugin uses — the sibling
// of TestDiscoverBinaries_ExcludesMockBinary above.
// plugins/mockstrict exists purely as browser-harness fixture
// infrastructure (introduced by 07.1-02) and is never a real source.
func TestDiscoverBinaries_ExcludesMockstrictBinary(t *testing.T) {
	dir := t.TempDir()
	writeFixtureFile(t, dir, "topos-plugin-mockstrict")
	writeFixtureFile(t, dir, "topos-plugin-paperless")

	got, err := DiscoverBinaries(dir)
	if err != nil {
		t.Fatalf("DiscoverBinaries: %v", err)
	}
	for _, name := range got {
		if name == "topos-plugin-mockstrict" {
			t.Fatalf("expected topos-plugin-mockstrict to be excluded, got %v", got)
		}
	}
	if len(got) != 1 || got[0] != "topos-plugin-paperless" {
		t.Fatalf("expected exactly [\"topos-plugin-paperless\"], got %v", got)
	}
}

// TestDiscoverAllBinaries_IncludesMockstrictBinary proves the unfiltered
// listing must still report topos-plugin-mockstrict alongside every other
// real binary, sorted — this is the direct regression gate for the bug
// 07.1-04 discovered live: a shared filtered result made
// POST /api/config/describe-plugin 404 for an already-configured mock
// instance, breaking the picker's one-step existing-instance add flow.
// Widening ExcludedPluginBinaries is exactly the class of change that
// re-breaks it, so this test must keep passing after mockstrict is added
// to the exclusion table.
func TestDiscoverAllBinaries_IncludesMockstrictBinary(t *testing.T) {
	dir := t.TempDir()
	writeFixtureFile(t, dir, "topos-plugin-mockstrict")
	writeFixtureFile(t, dir, "topos-plugin-paperless")

	got, err := DiscoverAllBinaries(dir)
	if err != nil {
		t.Fatalf("DiscoverAllBinaries: %v", err)
	}
	want := []string{"topos-plugin-mockstrict", "topos-plugin-paperless"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected sorted, unfiltered %v, got %v", want, got)
		}
	}
}

// TestDiscoverAllBinaries_IncludesMockBinary proves the split from
// DiscoverBinaries (07.1-04-PLAN.md deviation): the raw, security-relevant
// listing DescribePluginHandler's T-07-09 check uses must still report a
// real on-disk topos-plugin-mock binary — ExcludedPluginBinaries is a
// UI-policy filter that belongs only to the "+" picker's offered-types
// list (DiscoverBinaries), never to what may legitimately be described.
func TestDiscoverAllBinaries_IncludesMockBinary(t *testing.T) {
	dir := t.TempDir()
	writeFixtureFile(t, dir, "topos-plugin-mock")
	writeFixtureFile(t, dir, "topos-plugin-paperless")

	got, err := DiscoverAllBinaries(dir)
	if err != nil {
		t.Fatalf("DiscoverAllBinaries: %v", err)
	}
	want := []string{"topos-plugin-mock", "topos-plugin-paperless"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected sorted, unfiltered %v, got %v", want, got)
		}
	}
}

// TestDiscoverAllBinaries_MissingDirectoryReturnsEmptyNilError mirrors
// DiscoverBinaries' own missing-directory contract.
func TestDiscoverAllBinaries_MissingDirectoryReturnsEmptyNilError(t *testing.T) {
	got, err := DiscoverAllBinaries(filepath.Join(t.TempDir(), "does-not-exist"))
	if err != nil {
		t.Fatalf("expected a nil error for a missing directory, got: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected an empty slice, got %v", got)
	}
}

// TestDiscoverBinaries_FollowsSymlinkedBinaries is the regression test for
// the second bug 07.1-04-PLAN.md discovered live: os.ReadDir's own
// DirEntry.Type() reports a symlink's own mode bits, which
// fs.FileMode.IsRegular() always reports false for — before this fix, a
// symlinked plugin binary (exactly how web/e2e/fixtures/
// plugin-binaries.ts's linkPluginBinaries populates a hermetic e2e
// kernel's plugins directory, deliberately, per 07.1-01-SUMMARY.md) was
// invisible to both DiscoverBinaries and DiscoverAllBinaries, so
// PluginTypesHandler's "+ New <plugin type>…" list and
// DescribePluginHandler's security check both silently found nothing at
// all inside the e2e harness, for every plugin type, mock included.
func TestDiscoverBinaries_FollowsSymlinkedBinaries(t *testing.T) {
	realDir := t.TempDir()
	realBin := filepath.Join(realDir, "topos-plugin-paperless")
	writeFixtureFile(t, realDir, "topos-plugin-paperless")

	linkDir := t.TempDir()
	if err := os.Symlink(realBin, filepath.Join(linkDir, "topos-plugin-paperless")); err != nil {
		t.Fatalf("symlink fixture: %v", err)
	}
	// A dangling symlink (no real target) must be skipped, not error the
	// whole listing.
	if err := os.Symlink(filepath.Join(realDir, "does-not-exist"), filepath.Join(linkDir, "topos-plugin-dangling")); err != nil {
		t.Fatalf("dangling symlink fixture: %v", err)
	}

	got, err := DiscoverBinaries(linkDir)
	if err != nil {
		t.Fatalf("DiscoverBinaries: %v", err)
	}
	if len(got) != 1 || got[0] != "topos-plugin-paperless" {
		t.Fatalf("expected exactly a symlinked topos-plugin-paperless to be discovered, got %v", got)
	}
}

// TestDiscoverAllTiered_NameInBothDirectoriesResolvesToTrustedOnly proves
// D-11's replacement collision rule at the discovery level: a binary name
// present in BOTH the trusted and external directories, where the trusted
// copy carries evidence (installTrustedManifest — the link-time arm),
// appears EXACTLY ONCE in DiscoverAllTiered's result, tagged TierTrusted
// — the evidence-carrying copy wins a two-tier name collision, never the
// directory alone.
func TestDiscoverAllTiered_NameInBothDirectoriesResolvesToTrustedOnly(t *testing.T) {
	trustedDir := t.TempDir()
	writeFixtureFile(t, trustedDir, "topos-plugin-shadowed")
	writeFixtureFile(t, trustedDir, "topos-plugin-paperless")
	installTrustedManifest(t, trustedDir)

	externalDir := t.TempDir()
	writeFixtureFile(t, externalDir, "topos-plugin-shadowed")
	writeFixtureFile(t, externalDir, "topos-plugin-example")

	got, err := DiscoverAllTiered(Dirs{Trusted: trustedDir, External: externalDir})
	if err != nil {
		t.Fatalf("DiscoverAllTiered: %v", err)
	}

	want := []TieredBinary{
		{Name: "topos-plugin-example", Tier: TierExternal},
		{Name: "topos-plugin-paperless", Tier: TierTrusted},
		{Name: "topos-plugin-shadowed", Tier: TierTrusted},
	}
	if len(got) != len(want) {
		t.Fatalf("expected %+v, got %+v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected sorted, de-duplicated %+v, got %+v", want, got)
		}
	}
}

// TestDiscoverTiered_ExclusionPolicyAppliesUniformlyAcrossBothTiers
// proves the mock/mockstrict UI-policy exclusion is applied identically
// regardless of which directory the fixture binary sits in — a fixture
// binary in the EXTERNAL directory is excluded from the catalog exactly
// like one in the trusted directory always has been.
func TestDiscoverTiered_ExclusionPolicyAppliesUniformlyAcrossBothTiers(t *testing.T) {
	trustedDir := t.TempDir()
	writeFixtureFile(t, trustedDir, "topos-plugin-mock")
	writeFixtureFile(t, trustedDir, "topos-plugin-paperless")
	installTrustedManifest(t, trustedDir)

	externalDir := t.TempDir()
	writeFixtureFile(t, externalDir, "topos-plugin-mockstrict")
	writeFixtureFile(t, externalDir, "topos-plugin-example")

	got, err := DiscoverTiered(Dirs{Trusted: trustedDir, External: externalDir})
	if err != nil {
		t.Fatalf("DiscoverTiered: %v", err)
	}

	for _, tb := range got {
		if tb.Name == "topos-plugin-mock" || tb.Name == "topos-plugin-mockstrict" {
			t.Fatalf("expected both fixture binaries excluded regardless of tier, got %+v", got)
		}
	}

	want := []TieredBinary{
		{Name: "topos-plugin-example", Tier: TierExternal},
		{Name: "topos-plugin-paperless", Tier: TierTrusted},
	}
	if len(got) != len(want) {
		t.Fatalf("expected %+v, got %+v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected sorted %+v, got %+v", want, got)
		}
	}
}

// TestDiscoverAllTiered_AbsentExternalDirectoryIsEmptyTierNotAnError
// proves a missing/absent external directory contributes an empty set —
// a legitimate empty tier, mirroring DiscoverAllBinaries' own
// missing-directory-is-empty-state contract, never an error.
func TestDiscoverAllTiered_AbsentExternalDirectoryIsEmptyTierNotAnError(t *testing.T) {
	trustedDir := t.TempDir()
	writeFixtureFile(t, trustedDir, "topos-plugin-paperless")
	installTrustedManifest(t, trustedDir)

	got, err := DiscoverAllTiered(Dirs{Trusted: trustedDir, External: filepath.Join(t.TempDir(), "does-not-exist")})
	if err != nil {
		t.Fatalf("expected a nil error for an absent external directory, got: %v", err)
	}
	want := []TieredBinary{{Name: "topos-plugin-paperless", Tier: TierTrusted}}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("expected %+v, got %+v", want, got)
	}
}

// TestDiscoverAllTiered_BothDirectoriesEmptyStringReturnsEmptyNotError
// proves the zero-value Dirs{} (both fields the empty string, e.g. an
// omitted [plugins] external_dir before cmd/topos resolves a default)
// returns an empty, never-nil slice with a nil error.
func TestDiscoverAllTiered_BothDirectoriesEmptyStringReturnsEmptyNotError(t *testing.T) {
	got, err := DiscoverAllTiered(Dirs{})
	if err != nil {
		t.Fatalf("expected a nil error for an empty Dirs, got: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected an empty slice, got %+v", got)
	}
}

// TestDiscoverAllTiered_SymlinkedExternalBinaryIsDiscovered proves a
// symlinked binary in the EXTERNAL directory is discovered exactly like
// a symlinked binary in the trusted directory already is (the e2e
// harness's own fixture shape, extended to the external tier by plan
// 11-01's Task 3) — tagged TierExternal.
func TestDiscoverAllTiered_SymlinkedExternalBinaryIsDiscovered(t *testing.T) {
	realDir := t.TempDir()
	writeFixtureFile(t, realDir, "topos-plugin-example")

	externalDir := t.TempDir()
	if err := os.Symlink(filepath.Join(realDir, "topos-plugin-example"), filepath.Join(externalDir, "topos-plugin-example")); err != nil {
		t.Fatalf("symlink fixture: %v", err)
	}

	got, err := DiscoverAllTiered(Dirs{External: externalDir})
	if err != nil {
		t.Fatalf("DiscoverAllTiered: %v", err)
	}
	want := []TieredBinary{{Name: "topos-plugin-example", Tier: TierExternal}}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("expected a symlinked external binary %+v, got %+v", want, got)
	}
}

// TestDiscoverAllTiered_DigestMismatchStillAppearsTaggedExternal proves
// T-16-11: a binary whose signed manifest names it with a MISMATCHED
// digest (a tamper refusal from EvaluateTrust) still APPEARS in
// DiscoverAllTiered's listing — so the picker and describe endpoints keep
// seeing it — but is tagged TierExternal, never silently dropped or
// mistaken for a legitimately trusted binary. The listing call itself
// returns a nil error; the refusal is re-asserted at launch, the only
// place that runs code.
func TestDiscoverAllTiered_DigestMismatchStillAppearsTaggedExternal(t *testing.T) {
	trustedDir := t.TempDir()
	writeFixtureFile(t, trustedDir, "topos-plugin-tampered")
	keyID, priv := installProvenanceTestKey(t)
	hash := mustHashBinary(t, filepath.Join(trustedDir, "topos-plugin-tampered"))
	writeSignedManifest(t, trustedDir, "topos-plugins-v0.1.0", nativeRelease(),
		[]ProvenanceEntry{{Name: "topos-plugin-tampered", SHA256: hash, Version: "0.1.0", Contract: "topos.v1"}},
		keyID, priv)

	// Tamper the binary bytes AFTER the manifest above was computed from
	// the original ones.
	mutateLastByte(t, filepath.Join(trustedDir, "topos-plugin-tampered"))

	got, err := DiscoverAllTiered(Dirs{Trusted: trustedDir})
	if err != nil {
		t.Fatalf("expected a nil error even with a tamper refusal present, got: %v", err)
	}
	want := []TieredBinary{{Name: "topos-plugin-tampered", Tier: TierExternal}}
	if len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("expected the tampered binary to still appear, tagged %+v, got %+v", want, got)
	}
}

