package pluginhost

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// buildManifest is the link-time build-provenance manifest this kernel
// binary was linked with — populated ONLY via:
//
//	go build -ldflags "-X github.com/davison/topos/kernel/pluginhost.buildManifest=<spec>"
//
// <spec> is FormatManifest's comma-separated "name=hexdigest" output. This
// variable is NEVER read from a file on disk, an environment variable, or
// configuration at run time (D-12) — it is link-time data only, and every
// production build recipe (Makefile's build/build-portable/dev/e2e
// targets) sets it via the exact -X path above.
//
// An empty value — a bare `go build` with no -ldflags, or any build recipe
// that skipped the generator — means this kernel was built WITHOUT a
// manifest, and therefore trusts NO trusted-directory binary at all
// (PD-04). There is deliberately no fallback to directory-derived trust: a
// silent downgrade of a security control is exactly what this project's
// fail-loudly-by-name convention forbids.
var buildManifest string

// ErrManifestUnverified is returned (always wrapped, never bare) by
// VerifyTrustedBinary when a trusted-directory binary's name is absent
// from the link-time build manifest, is present but its on-disk SHA-256 no
// longer matches the manifest's recorded digest, or no manifest was
// embedded in this kernel build at all — all three collapse to the same
// sentinel because the caller's remedy is identical in every case: this
// binary's only path to running is the existing external-tier consent and
// pin flow (D-13) — verification never demotes-and-runs.
var ErrManifestUnverified = errors.New("pluginhost: trusted binary is not verified by the kernel's build manifest")

// FormatManifest renders entries as the comma-separated "name=hexdigest"
// spec buildManifest holds at link time, sorted by name so the identical
// binary set always produces the identical string — a build is
// reproducible, not merely parseable. This is the ONE producer of the
// spec shape (cmd/topos-manifest and every test helper that builds an
// OverrideBuildManifest value go through this, never hand-format a
// spec), matching ParseManifest's own round-trip contract below.
func FormatManifest(entries map[string]string) string {
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)

	parts := make([]string, 0, len(names))
	for _, name := range names {
		parts = append(parts, name+"="+entries[name])
	}
	return strings.Join(parts, ",")
}

// manifestDigestLen is the fixed length of a lowercase-hex SHA-256 digest
// (32 bytes, hex-encoded) — the only digest shape ParseManifest accepts.
const manifestDigestLen = 64

// isHexDigest reports whether s is exactly manifestDigestLen lowercase hex
// characters — the shape hex.EncodeToString(sha256.Sum(...)) always
// produces (see binaryhash.go's HashBinary), and the only shape
// ParseManifest accepts for a digest.
func isHexDigest(s string) bool {
	if len(s) != manifestDigestLen {
		return false
	}
	for _, r := range s {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

// ParseManifest parses spec — the comma-separated "name=hexdigest" format
// FormatManifest produces — into a name -> digest map. An empty (or
// whitespace-only) spec is NOT an error: it returns an empty, non-nil map,
// meaning "no manifest was embedded in this kernel build" (PD-04's
// trust-nothing state, not a parse failure). Every other malformed shape —
// a segment with no "=", an empty or otherwise invalid binary name, or a
// digest that isn't manifestDigestLen lowercase hex characters — is
// REJECTED outright with a named error identifying the offending segment;
// nothing is ever silently dropped, because a manifest that quietly lost
// an entry is a security regression, not a parsing nicety.
func ParseManifest(spec string) (map[string]string, error) {
	entries := make(map[string]string)

	trimmed := strings.TrimSpace(spec)
	if trimmed == "" {
		return entries, nil
	}

	for _, segment := range strings.Split(trimmed, ",") {
		idx := strings.Index(segment, "=")
		if idx < 0 {
			return nil, fmt.Errorf("pluginhost: malformed manifest segment %q: missing \"=\"", segment)
		}
		name := segment[:idx]
		digest := segment[idx+1:]
		if err := validatePluginBinaryName(name); err != nil {
			return nil, fmt.Errorf("pluginhost: malformed manifest segment %q: %w", segment, err)
		}
		if !isHexDigest(digest) {
			return nil, fmt.Errorf("pluginhost: malformed manifest segment %q: digest is not %d lowercase hex characters", segment, manifestDigestLen)
		}
		entries[name] = digest
	}
	return entries, nil
}

var (
	// trustManifestOnce/trustManifestParsed cache buildManifest's ONE
	// parse — go build's own -ldflags -X mechanism sets buildManifest
	// before main() ever runs, so it never changes for the lifetime of a
	// real kernel process, and re-parsing it on every VerifyTrustedBinary
	// call would be pure waste.
	trustManifestOnce   sync.Once
	trustManifestParsed map[string]string

	// manifestOverrideMu guards manifestOverride/manifestOverrideSet — the
	// TEST-ONLY seam OverrideBuildManifest/OverrideBuildManifestFromDir
	// install below. Never written outside those two functions and this
	// file's own TrustManifest reader.
	manifestOverrideMu  sync.RWMutex
	manifestOverride    map[string]string
	manifestOverrideSet bool
)

// TrustManifest returns a defensive copy of the currently effective trust
// manifest: an installed OverrideBuildManifest/OverrideBuildManifestFromDir
// value when a test has set one, otherwise buildManifest — this kernel
// binary's own link-time value, parsed exactly once. A copy is returned
// (never the package-level map itself) so no caller can mutate the trust
// authority through the value it was handed.
func TrustManifest() map[string]string {
	manifestOverrideMu.RLock()
	if manifestOverrideSet {
		out := make(map[string]string, len(manifestOverride))
		for k, v := range manifestOverride {
			out[k] = v
		}
		manifestOverrideMu.RUnlock()
		return out
	}
	manifestOverrideMu.RUnlock()

	trustManifestOnce.Do(func() {
		parsed, err := ParseManifest(buildManifest)
		if err != nil {
			// A malformed link-time manifest cannot be repaired at run
			// time — trusting nothing is the fail-safe state (PD-04's own
			// discipline, extended to this should-never-happen case: a
			// well-formed FormatManifest output should never fail its own
			// ParseManifest, so reaching this branch means the -ldflags -X
			// value was hand-corrupted, not machine-generated).
			trustManifestParsed = map[string]string{}
			return
		}
		trustManifestParsed = parsed
	})

	out := make(map[string]string, len(trustManifestParsed))
	for k, v := range trustManifestParsed {
		out[k] = v
	}
	return out
}

// manifestEmpty reports whether manifest — a value already obtained from
// TrustManifest() — names no binaries at all: the "this kernel build was
// linked with no -ldflags -X manifest at all" case VerifyTrustedBinary's
// error text names explicitly, distinct from "a non-empty manifest just
// doesn't happen to name this one binary."
func manifestEmpty(manifest map[string]string) bool {
	return len(manifest) == 0
}

// VerifyTrustedBinary computes path's current SHA-256 (via the existing
// HashBinary — see binaryhash.go's one-hashing-convention discipline; no
// second implementation is ever added here) and checks it against name's
// entry in TrustManifest(). Success (nil error) requires an entry to exist
// AND its digest to equal the freshly computed hash; every other outcome —
// no entry for name, a mismatched digest, or no manifest embedded at all —
// returns the computed hash alongside a wrapped ErrManifestUnverified
// naming name and the on-disk digest, so a caller (launch, in host.go)
// never needs to re-hash the file a second time to build a LaunchFailure
// record.
func VerifyTrustedBinary(name, path string) (hash string, err error) {
	hash, err = HashBinary(path)
	if err != nil {
		return "", err
	}

	manifest := TrustManifest()
	expected, ok := manifest[name]
	switch {
	case !ok && manifestEmpty(manifest):
		return hash, fmt.Errorf("%w: %q — this kernel build embeds no manifest at all", ErrManifestUnverified, name)
	case !ok:
		return hash, fmt.Errorf("%w: %q — no entry in the build manifest", ErrManifestUnverified, name)
	case expected != hash:
		return hash, fmt.Errorf("%w: %q — on-disk hash %s does not match its manifest entry", ErrManifestUnverified, name, hash)
	default:
		return hash, nil
	}
}

// ManifestEntriesForBinaries hashes each of paths (via HashBinary — never
// a second hashing implementation) and keys the result by
// filepath.Base(path), so cmd/topos-manifest and this package's own tests
// share ONE function for turning a build recipe's explicit binary-path
// list into the map FormatManifest renders. A missing or unreadable path
// is a named error identifying the offending path — never a silently
// skipped entry, which is exactly the failure shape RESEARCH Pitfall 6
// warns a mis-wired recipe could otherwise produce (an empty or
// incomplete manifest with no visible cause).
func ManifestEntriesForBinaries(paths ...string) (map[string]string, error) {
	entries := make(map[string]string, len(paths))
	for _, p := range paths {
		hash, err := HashBinary(p)
		if err != nil {
			return nil, fmt.Errorf("pluginhost: manifest entry for %s: %w", p, err)
		}
		entries[filepath.Base(p)] = hash
	}
	return entries, nil
}

// manifestEntriesFromDir hashes every regular plugin binary in dir —
// following a symlink exactly as isRegularFileFollowingSymlinks does, so
// the e2e browser harness's symlinked fixture binaries
// (web/e2e/fixtures/plugin-binaries.ts) verify identically to a real `go
// build` output — keyed by name. The shared scan both
// OverrideBuildManifestFromDir (below) and this package's own test helpers
// use.
func manifestEntriesFromDir(dir string) (map[string]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("pluginhost: scan manifest fixture dir %s: %w", dir, err)
	}

	out := make(map[string]string)
	for _, e := range entries {
		name := e.Name()
		if !strings.HasPrefix(name, PluginBinaryPrefix) {
			continue
		}
		if !isRegularFileFollowingSymlinks(dir, e) {
			continue
		}
		hash, hashErr := HashBinary(filepath.Join(dir, name))
		if hashErr != nil {
			return nil, fmt.Errorf("pluginhost: hash %s: %w", name, hashErr)
		}
		out[name] = hash
	}
	return out, nil
}

// OverrideBuildManifest is a TEST-ONLY seam: it installs entries as the
// CURRENT effective trust manifest (TrustManifest() returns exactly this
// map, ignoring buildManifest entirely, until restore is called) and
// returns a restore func that puts the previous override state —
// installed or not — back exactly. Every kernel/pluginhost,
// kernel/httpapi, and kernel/supervisor test that launches a trusted-tier
// fixture binary calls this (or OverrideBuildManifestFromDir below) right
// after building the fixture, so the manifest verification gate
// (VerifyTrustedBinary, launch) sees a real entry for that binary rather
// than refusing it as unverified.
//
// MUST NEVER be called from production code — production code has no
// legitimate reason to override the link-time manifest at run time; doing
// so would recreate exactly the "trust the directory instead" bypass D-12
// closes.
func OverrideBuildManifest(entries map[string]string) (restore func()) {
	manifestOverrideMu.Lock()
	prevOverride := manifestOverride
	prevSet := manifestOverrideSet

	cp := make(map[string]string, len(entries))
	for k, v := range entries {
		cp[k] = v
	}
	manifestOverride = cp
	manifestOverrideSet = true
	manifestOverrideMu.Unlock()

	return func() {
		manifestOverrideMu.Lock()
		manifestOverride = prevOverride
		manifestOverrideSet = prevSet
		manifestOverrideMu.Unlock()
	}
}

// OverrideBuildManifestFromDir is OverrideBuildManifest's directory-driven
// convenience form (TEST-ONLY, same prohibition as above): it hashes every
// regular plugin binary in dir (manifestEntriesFromDir) and installs the
// resulting name -> digest map via OverrideBuildManifest.
func OverrideBuildManifestFromDir(dir string) (restore func(), err error) {
	entries, err := manifestEntriesFromDir(dir)
	if err != nil {
		return nil, err
	}
	return OverrideBuildManifest(entries), nil
}
