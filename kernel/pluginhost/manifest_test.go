// manifest_test.go pins the link-time build-provenance manifest's parse,
// verify, and generate contract (D-12, PD-03/PD-04): FormatManifest and
// ParseManifest round-trip exactly, VerifyTrustedBinary refuses anything
// not byte-identical to a real manifest entry, and the test seam
// (OverrideBuildManifest/OverrideBuildManifestFromDir) installs and
// restores cleanly.
package pluginhost

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// TestFormatManifest_ParseManifest_RoundTrips proves FormatManifest then
// ParseManifest round-trips a map of binary name to hex digest exactly,
// for zero, one, and several entries.
func TestFormatManifest_ParseManifest_RoundTrips(t *testing.T) {
	cases := []struct {
		name    string
		entries map[string]string
	}{
		{"zero entries", map[string]string{}},
		{"one entry", map[string]string{
			"topos-plugin-mock": strings.Repeat("a", 64),
		}},
		{"several entries", map[string]string{
			"topos-plugin-mock":         strings.Repeat("a", 64),
			"topos-plugin-paperless":    strings.Repeat("b", 64),
			"topos-plugin-silverbullet": strings.Repeat("c", 64),
		}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			spec := FormatManifest(tc.entries)
			got, err := ParseManifest(spec)
			if err != nil {
				t.Fatalf("ParseManifest(%q): %v", spec, err)
			}
			if !reflect.DeepEqual(got, tc.entries) {
				t.Errorf("round-trip mismatch: got %v, want %v", got, tc.entries)
			}
		})
	}
}

// TestFormatManifest_SortsByName proves the same binary set always
// produces the identical spec string regardless of map iteration order —
// a build is reproducible, not merely parseable.
func TestFormatManifest_SortsByName(t *testing.T) {
	entries := map[string]string{
		"topos-plugin-silverbullet": strings.Repeat("c", 64),
		"topos-plugin-mock":         strings.Repeat("a", 64),
		"topos-plugin-paperless":    strings.Repeat("b", 64),
	}
	want := "topos-plugin-mock=" + strings.Repeat("a", 64) +
		",topos-plugin-paperless=" + strings.Repeat("b", 64) +
		",topos-plugin-silverbullet=" + strings.Repeat("c", 64)
	if got := FormatManifest(entries); got != want {
		t.Errorf("expected sorted spec %q, got %q", want, got)
	}
}

// TestParseManifest_EmptyStringReturnsEmptyMapNoError proves ParseManifest
// of the empty string returns an empty map and reports no error — "no
// manifest was embedded" is not a parse failure.
func TestParseManifest_EmptyStringReturnsEmptyMapNoError(t *testing.T) {
	for _, spec := range []string{"", "   "} {
		got, err := ParseManifest(spec)
		if err != nil {
			t.Fatalf("ParseManifest(%q): unexpected error: %v", spec, err)
		}
		if len(got) != 0 {
			t.Errorf("ParseManifest(%q): expected an empty map, got %v", spec, got)
		}
	}
}

// TestParseManifest_MalformedSegmentsAreRejected proves ParseManifest
// rejects a malformed spec — a segment with no separator, an empty name,
// a non-hex digest — with a named error rather than silently dropping the
// segment.
func TestParseManifest_MalformedSegmentsAreRejected(t *testing.T) {
	validDigest := strings.Repeat("a", 64)
	cases := []struct {
		name string
		spec string
	}{
		{"missing separator", "topos-plugin-mock" + validDigest},
		{"empty name", "=" + validDigest},
		{"non-hex digest", "topos-plugin-mock=not-hex-at-all"},
		{"short digest", "topos-plugin-mock=abc123"},
		{"uppercase digest", "topos-plugin-mock=" + strings.ToUpper(validDigest)},
		{"traversal name", "../evil=" + validDigest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseManifest(tc.spec)
			if err == nil {
				t.Fatalf("ParseManifest(%q): expected an error, got nil", tc.spec)
			}
		})
	}
}

// TestParseManifest_MalformedSegmentAmongValidOnesFailsTheWholeParse
// proves a malformed segment is never silently dropped while its siblings
// survive — the whole parse fails, naming the offending segment.
func TestParseManifest_MalformedSegmentAmongValidOnesFailsTheWholeParse(t *testing.T) {
	validDigest := strings.Repeat("a", 64)
	spec := "topos-plugin-mock=" + validDigest + ",badsegment,topos-plugin-paperless=" + validDigest
	_, err := ParseManifest(spec)
	if err == nil {
		t.Fatal("expected an error for a spec containing one malformed segment")
	}
	if !strings.Contains(err.Error(), "badsegment") {
		t.Errorf("expected the error to name the offending segment %q, got: %v", "badsegment", err)
	}
}

// TestVerifyTrustedBinary_MatchingManifestHashSucceeds proves VerifyTrustedBinary
// returns the computed hash and a nil error when the manifest entry
// matches the file's real hash.
func TestVerifyTrustedBinary_MatchingManifestHashSucceeds(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "topos-plugin-mock")
	if err := os.WriteFile(binPath, []byte("fixture-bytes"), 0o755); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	realHash, err := HashBinary(binPath)
	if err != nil {
		t.Fatalf("HashBinary: %v", err)
	}

	restore := OverrideBuildManifest(map[string]string{"topos-plugin-mock": realHash})
	defer restore()

	hash, err := VerifyTrustedBinary("topos-plugin-mock", binPath)
	if err != nil {
		t.Fatalf("VerifyTrustedBinary: unexpected error: %v", err)
	}
	if hash != realHash {
		t.Errorf("expected hash %q, got %q", realHash, hash)
	}
}

// TestVerifyTrustedBinary_NoEntryForNameReturnsErrManifestUnverified
// proves VerifyTrustedBinary returns ErrManifestUnverified when the
// manifest holds no entry for the name.
func TestVerifyTrustedBinary_NoEntryForNameReturnsErrManifestUnverified(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "topos-plugin-mock")
	if err := os.WriteFile(binPath, []byte("fixture-bytes"), 0o755); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	restore := OverrideBuildManifest(map[string]string{
		"topos-plugin-paperless": strings.Repeat("a", 64),
	})
	defer restore()

	_, err := VerifyTrustedBinary("topos-plugin-mock", binPath)
	if !errors.Is(err, ErrManifestUnverified) {
		t.Fatalf("expected errors.Is(err, ErrManifestUnverified), got: %v", err)
	}
	if !strings.Contains(err.Error(), "topos-plugin-mock") {
		t.Errorf("expected the error to name the binary, got: %v", err)
	}
}

// TestVerifyTrustedBinary_MismatchedBytesReturnsErrManifestUnverified
// proves VerifyTrustedBinary returns ErrManifestUnverified when the entry
// exists but the file's bytes differ.
func TestVerifyTrustedBinary_MismatchedBytesReturnsErrManifestUnverified(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "topos-plugin-mock")
	if err := os.WriteFile(binPath, []byte("fixture-bytes"), 0o755); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	restore := OverrideBuildManifest(map[string]string{
		"topos-plugin-mock": strings.Repeat("f", 64), // guaranteed not to match
	})
	defer restore()

	_, err := VerifyTrustedBinary("topos-plugin-mock", binPath)
	if !errors.Is(err, ErrManifestUnverified) {
		t.Fatalf("expected errors.Is(err, ErrManifestUnverified), got: %v", err)
	}
}

// TestVerifyTrustedBinary_NoManifestEmbeddedReturnsErrManifestUnverified
// proves VerifyTrustedBinary returns ErrManifestUnverified when no
// manifest is embedded at all (an empty override, standing in for a bare
// `go build` with no -ldflags -X value).
func TestVerifyTrustedBinary_NoManifestEmbeddedReturnsErrManifestUnverified(t *testing.T) {
	dir := t.TempDir()
	binPath := filepath.Join(dir, "topos-plugin-mock")
	if err := os.WriteFile(binPath, []byte("fixture-bytes"), 0o755); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	restore := OverrideBuildManifest(map[string]string{})
	defer restore()

	_, err := VerifyTrustedBinary("topos-plugin-mock", binPath)
	if !errors.Is(err, ErrManifestUnverified) {
		t.Fatalf("expected errors.Is(err, ErrManifestUnverified), got: %v", err)
	}
	if !strings.Contains(err.Error(), "no manifest") {
		t.Errorf("expected the error to name the empty-manifest case, got: %v", err)
	}
}

// TestManifestEntriesForBinaries_HashesEachPathKeyedByBaseName proves
// ManifestEntriesForBinaries hashes each supplied path and keys the
// result by the path's base name.
func TestManifestEntriesForBinaries_HashesEachPathKeyedByBaseName(t *testing.T) {
	dir := t.TempDir()
	mockPath := filepath.Join(dir, "topos-plugin-mock")
	paperlessPath := filepath.Join(dir, "topos-plugin-paperless")
	if err := os.WriteFile(mockPath, []byte("mock-bytes"), 0o755); err != nil {
		t.Fatalf("write mock fixture: %v", err)
	}
	if err := os.WriteFile(paperlessPath, []byte("paperless-bytes"), 0o755); err != nil {
		t.Fatalf("write paperless fixture: %v", err)
	}

	wantMockHash, err := HashBinary(mockPath)
	if err != nil {
		t.Fatalf("HashBinary(mock): %v", err)
	}
	wantPaperlessHash, err := HashBinary(paperlessPath)
	if err != nil {
		t.Fatalf("HashBinary(paperless): %v", err)
	}

	entries, err := ManifestEntriesForBinaries(mockPath, paperlessPath)
	if err != nil {
		t.Fatalf("ManifestEntriesForBinaries: %v", err)
	}
	if entries["topos-plugin-mock"] != wantMockHash {
		t.Errorf("expected topos-plugin-mock hash %q, got %q", wantMockHash, entries["topos-plugin-mock"])
	}
	if entries["topos-plugin-paperless"] != wantPaperlessHash {
		t.Errorf("expected topos-plugin-paperless hash %q, got %q", wantPaperlessHash, entries["topos-plugin-paperless"])
	}
}

// TestManifestEntriesForBinaries_MissingPathIsNamedError proves a missing
// path is a named error, not a skipped entry.
func TestManifestEntriesForBinaries_MissingPathIsNamedError(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "topos-plugin-does-not-exist")

	_, err := ManifestEntriesForBinaries(missing)
	if err == nil {
		t.Fatal("expected an error for a missing path")
	}
	if !strings.Contains(err.Error(), missing) {
		t.Errorf("expected the error to name the missing path %q, got: %v", missing, err)
	}
}

// TestOverrideBuildManifest_InstallsAndRestoresExactly proves
// OverrideBuildManifest installs entries and its returned restore func
// puts the previous value back exactly.
func TestOverrideBuildManifest_InstallsAndRestoresExactly(t *testing.T) {
	before := TrustManifest()

	restoreOuter := OverrideBuildManifest(map[string]string{
		"topos-plugin-mock": strings.Repeat("a", 64),
	})

	restoreInner := OverrideBuildManifest(map[string]string{
		"topos-plugin-paperless": strings.Repeat("b", 64),
	})
	if got := TrustManifest(); got["topos-plugin-paperless"] != strings.Repeat("b", 64) {
		t.Fatalf("expected the inner override installed, got %v", got)
	}

	restoreInner()
	if got := TrustManifest(); got["topos-plugin-mock"] != strings.Repeat("a", 64) || len(got) != 1 {
		t.Fatalf("expected restoring the inner override to put back the outer one exactly, got %v", got)
	}

	restoreOuter()
	if got := TrustManifest(); !reflect.DeepEqual(got, before) {
		t.Fatalf("expected restoring the outer override to put back the original state, got %v want %v", got, before)
	}
}

// TestOverrideBuildManifest_ReturnedMapIsACopy proves TrustManifest never
// hands out the live backing map — mutating a caller's returned value must
// never affect what a later VerifyTrustedBinary call sees.
func TestOverrideBuildManifest_ReturnedMapIsACopy(t *testing.T) {
	restore := OverrideBuildManifest(map[string]string{
		"topos-plugin-mock": strings.Repeat("a", 64),
	})
	defer restore()

	got := TrustManifest()
	got["topos-plugin-mock"] = "tampered"

	if again := TrustManifest(); again["topos-plugin-mock"] != strings.Repeat("a", 64) {
		t.Errorf("expected mutating a returned TrustManifest() copy to have no effect, got %v", again)
	}
}

// TestOverrideBuildManifestFromDir_PicksUpSymlinkedBinary proves
// OverrideBuildManifestFromDir picks up a symlinked plugin binary (the
// e2e harness's fixture shape) with the same hash the symlink target has.
func TestOverrideBuildManifestFromDir_PicksUpSymlinkedBinary(t *testing.T) {
	realDir := t.TempDir()
	realPath := filepath.Join(realDir, "topos-plugin-symlinked")
	if err := os.WriteFile(realPath, []byte("real-bytes"), 0o755); err != nil {
		t.Fatalf("write real fixture: %v", err)
	}
	wantHash, err := HashBinary(realPath)
	if err != nil {
		t.Fatalf("HashBinary: %v", err)
	}

	linkDir := t.TempDir()
	linkPath := filepath.Join(linkDir, "topos-plugin-symlinked")
	if err := os.Symlink(realPath, linkPath); err != nil {
		t.Fatalf("symlink fixture: %v", err)
	}

	restore, err := OverrideBuildManifestFromDir(linkDir)
	if err != nil {
		t.Fatalf("OverrideBuildManifestFromDir: %v", err)
	}
	defer restore()

	got := TrustManifest()
	if got["topos-plugin-symlinked"] != wantHash {
		t.Errorf("expected symlinked binary hash %q, got %q", wantHash, got["topos-plugin-symlinked"])
	}
}

// TestOverrideBuildManifestFromDir_IgnoresNonPrefixedAndNonRegularEntries
// proves the directory scan applies the same PluginBinaryPrefix and
// regular-file rules DiscoverAllBinaries does, rather than hashing every
// file in the directory.
func TestOverrideBuildManifestFromDir_IgnoresNonPrefixedAndNonRegularEntries(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "topos-plugin-mock"), []byte("x"), 0o755); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("not a plugin"), 0o644); err != nil {
		t.Fatalf("write unrelated file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(dir, "topos-plugin-subdir"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	restore, err := OverrideBuildManifestFromDir(dir)
	if err != nil {
		t.Fatalf("OverrideBuildManifestFromDir: %v", err)
	}
	defer restore()

	got := TrustManifest()
	if len(got) != 1 {
		t.Fatalf("expected exactly one manifest entry, got %v", got)
	}
	if _, ok := got["topos-plugin-mock"]; !ok {
		t.Errorf("expected topos-plugin-mock in the manifest, got %v", got)
	}
}
