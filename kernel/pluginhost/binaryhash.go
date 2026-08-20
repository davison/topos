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
// authenticity one on its own: a matching digest proves "the same bytes I
// saw before", never by itself "published by someone I trust" (11-RESEARCH.md's
// V6 Cryptography note: SHA-256 here is narrowly an integrity control, not a
// cryptographic authentication feature). Publisher authentication DOES exist
// elsewhere in this design, as of Phase 16: a validly-signed release manifest
// (see provenance.go's VerifySignedProvenance/EvaluateTrust, and
// docs/plugin-trust.md for the full model) is exactly that — an ed25519
// signature over a manifest naming this HashBinary digest, verified against
// the kernel's own embedded accepted-key set. HashBinary itself still only
// ever proves integrity; it is the caller (EvaluateTrust) that adds
// authenticity on top, by also checking a signature.
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
