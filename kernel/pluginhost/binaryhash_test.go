package pluginhost

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

// TestHashBinary_KnownBytesMatchesSHA256Hex proves HashBinary produces
// exactly the digest sha256sum would report for the same bytes: an
// independently computed sha256.Sum256 over the identical fixture content
// (never a copy of HashBinary's own implementation), hex-encoded, lowercase,
// 64 characters.
func TestHashBinary_KnownBytesMatchesSHA256Hex(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fixture.bin")
	content := []byte("topos plugin binary hash fixture — deterministic content\n")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("seed fixture: %v", err)
	}

	got, err := HashBinary(path)
	if err != nil {
		t.Fatalf("HashBinary: %v", err)
	}

	sum := sha256.Sum256(content)
	want := hex.EncodeToString(sum[:])

	if got != want {
		t.Fatalf("HashBinary(%q) = %q, want %q", path, got, want)
	}
	if len(got) != 64 {
		t.Fatalf("expected a 64-character lowercase hex digest, got %d chars: %q", len(got), got)
	}
	for _, r := range got {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			t.Fatalf("expected a lowercase hex digest, got non-hex/uppercase rune %q in %q", r, got)
		}
	}
}

// TestHashBinary_MissingFileReturnsError proves a missing file produces an
// error rather than a zero-value digest that could be mistaken for a real
// hash.
func TestHashBinary_MissingFileReturnsError(t *testing.T) {
	_, err := HashBinary(filepath.Join(t.TempDir(), "does-not-exist"))
	if err == nil {
		t.Fatal("expected an error for a missing file, got nil")
	}
}
