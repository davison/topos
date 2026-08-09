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
