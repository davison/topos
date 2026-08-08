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
