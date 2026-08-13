package pluginhost

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// HashBinary returns the lowercase hex-encoded SHA-256 digest of the file at
// path — the plugin-BINARY sibling of kernel/config/store.go's fileHash
// (the identical sha256.New/io.Copy/hex.EncodeToString shape, reused rather
// than reinvented, per this project's one-hashing-convention discipline).
// It deliberately streams via io.Copy rather than os.ReadFile, because a
// compiled plugin binary is orders of magnitude larger than config.toml —
// loading it whole into memory just to hash it would be wasteful for no
// benefit.
//
// This is an integrity/identity check (content addressing), never an
// authenticity one: a matching digest proves "the same bytes I saw before",
// never "published by someone I trust" — there is no signature or publisher
// verification anywhere in this design (11-RESEARCH.md's V6 Cryptography
// note: SHA-256 here is narrowly an integrity control, not a cryptographic
// authentication feature).
func HashBinary(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("pluginhost: hash binary %s: %w", path, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("pluginhost: hash binary %s: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
