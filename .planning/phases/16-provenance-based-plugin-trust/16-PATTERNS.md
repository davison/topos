# Phase 16: Provenance-Based Plugin Trust - Pattern Map

**Mapped:** 2026-08-19
**Files analyzed:** 9 (new/modified)
**Analogs found:** 9 / 9 (all from in-repo code; no RESEARCH.md existed for this phase)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|-----------------|---------------|
| `kernel/pluginhost/provenance.go` (new) | service (crypto verifier) | request-response (verify-on-call) | `kernel/pluginhost/manifest.go` | exact — same package, same "verify a binary against an authority" shape |
| `kernel/pluginhost/provenance_test.go` (new) | test | request-response | `kernel/pluginhost/manifest_test.go` / `manifestgate_test.go` | exact |
| `kernel/pluginhost/discover_binaries.go` (modified) | service (tier derivation) | CRUD-like (compute+classify) | itself (existing file, D-11 rewrite) | exact — self-modification |
| `kernel/pluginhost/host.go` (modified, `launch`) | service (launch orchestration) | request-response | itself (existing `launch` func, coexistence arm added) | exact — self-modification |
| `kernel/config/types.go` (modified, `PluginsConfig` doc) | model/config | CRUD (config load) | itself | exact — doc/comment rewrite only, D-11 |
| `cmd/topos-manifest/main.go` (modified or sibling `cmd/topos-release-manifest`) | utility (CLI generator) | file-I/O (hash → stdout) | `cmd/topos-manifest/main.go` | exact |
| `topos-plugins/` sibling repo: signing workflow (`.github/workflows/release.yml`) + trivial plugin (new repo, D-04) | config (CI workflow) | event-driven (tag push → sign) | none in this repo — pattern from `Makefile`'s `MANIFEST_GEN_*` recipes + this repo's own `.github/workflows/` (if present) | role-match only |
| `web/e2e/specs/16-*.spec.ts` (new, if tier UI surfaces change) | test (e2e) | request-response (browser assertions) | `web/e2e/specs/11-external-tier-badge.spec.ts`, `11-untrusted-add.spec.ts`, `11-binary-changed-repin.spec.ts` | exact |
| `web/e2e/fixtures/plugin-binaries.ts` (modified, if signed-manifest fixtures needed) | test fixture | file-I/O (symlink/fixture setup) | itself (existing symlink-based manifest fixture) | exact — extend, don't redesign |

## Pattern Assignments

### `kernel/pluginhost/provenance.go` (new) — signed release-manifest verifier

**Analog:** `kernel/pluginhost/manifest.go` (the link-time build manifest this file coexists with per D-10)

**Package/imports pattern** (manifest.go lines 1-11):
```go
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
```
Provenance.go will additionally need `crypto/ed25519` (stdlib; D-01 says "pure in-kernel verification via `golang.org/x/crypto`" but `crypto/ed25519` has been in the standard library since Go 1.13 — this repo is on Go 1.25 per `go.mod` line 3 — so no new dependency is actually required unless a specific `x/crypto` primitive beyond stdlib `ed25519.Verify`/`ed25519.PublicKey` is needed; confirm during planning whether `golang.org/x/crypto` is genuinely needed or whether stdlib suffices, since D-01 explicitly favors "near-zero new dependencies").

**Sentinel-error pattern to mirror exactly** (manifest.go lines 32-40):
```go
var ErrManifestUnverified = errors.New("pluginhost: trusted binary is not verified by the kernel's build manifest")
```
→ New file should define a distinct sentinel, e.g. `ErrProvenanceUnverified`, following the identical doc-comment discipline: name every collapsed failure mode (missing manifest file, malformed signature, signature doesn't verify, name/hash not in manifest) and state explicitly that verification never demotes-and-runs (D-13, this repo's fail-loudly-by-name convention already established at `ErrManifestUnverified`'s own site).

**Embedded-key-set pattern** (D-03: SET of keys with IDs) — no existing in-repo analog for multi-key acceptance; model this after `buildManifest`'s link-time `-X` injection convention (manifest.go lines 13-30) but for an embedded public key set:
```go
// buildManifest is the link-time build-provenance manifest this kernel
// binary was linked with — populated ONLY via:
//
//	go build -ldflags "-X github.com/davison/topos/kernel/pluginhost.buildManifest=<spec>"
```
The accepted-key SET (D-03) should follow the same "link-time or embedded constant, never runtime-configurable" discipline — trust inputs are never read from files the user can edit at runtime (D-12's established discipline, extended).

**Hash reuse — MUST call, never reimplement** (binaryhash.go lines 26-38):
```go
func HashBinary(path string) (string, error) {
	f, err := os.Open(path)
	...
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil { ... }
	return hex.EncodeToString(h.Sum(nil)), nil
}
```
`provenance.go`'s verifier reuses `HashBinary` exactly as `VerifyTrustedBinary` does (manifest.go line 197) — "the ONE hashing convention" per canonical refs.

**Verify-function shape to mirror** (manifest.go lines 186-214, `VerifyTrustedBinary`):
```go
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
```
New `VerifySignedProvenance(name, path string) (hash string, err error)` (or similar name — Claude's discretion per CONTEXT) should follow this exact "hash first, then check identity/signature, exhaustive named-cause switch, wrapped sentinel" shape — and per D-05 must ALSO check the manifest signature is valid (ed25519.Verify over the manifest bytes with an accepted key) BEFORE trusting any entry inside it, and per D-08 must scan ALL present versioned manifest files, accepting the binary if ANY validly-signed one names it with a matching hash.

**Test-only override seam to mirror** (manifest.go lines 268-314, `OverrideBuildManifest`/`OverrideBuildManifestFromDir`):
```go
// OverrideBuildManifest is a TEST-ONLY seam: it installs entries as the
// CURRENT effective trust manifest ...
// MUST NEVER be called from production code ...
func OverrideBuildManifest(entries map[string]string) (restore func()) { ... }
```
D-12's "keep the manifest-injection seam cleanly factored" constraint means the new signed-provenance verifier needs an equivalent test seam (e.g. `OverrideSignedManifest`/inject a test keypair + signed fixture manifest) built the same way: package-level guarded state, `sync.RWMutex`, defensive copies, explicit "MUST NEVER be called from production code" doc comment.

**Name→hash binding rationale to preserve in docs** (D-05: names bound to hashes so a renamed binary can't shadow another) — mirror `FormatManifest`/`ParseManifest`'s "name=hexdigest" round-trip contract (manifest.go lines 42-117) as the base vocabulary the signed manifest extends with version + gRPC contract generation + release tag + platform/arch fields (D-06).

---

### `kernel/pluginhost/discover_binaries.go` (modified) — provenance-driven tier

**Analog:** itself — `DiscoverAllTiered` (lines 224-269), `resolveBinaryDetailed` (lines 353-418), and `Tier`/`Dirs` (lines 160-195)

**Current directory-derived tier logic being replaced** (lines 224-235, 392-407):
```go
if dirs.Trusted != "" {
	names, err := DiscoverAllBinaries(dirs.Trusted)
	...
	for _, name := range names {
		tierOf[name] = TierTrusted
	}
}
```
and
```go
if dirs.Trusted != "" {
	trustedPath := filepath.Join(dirs.Trusted, name)
	if info, statErr := os.Stat(trustedPath); statErr == nil && info.Mode().IsRegular() {
		...
		return trustedPath, TierTrusted, shadowed, nil
	}
}
```
Per D-11, both of these must become **provenance-driven**: tier is computed per binary by calling the new provenance/manifest verifiers (link-time OR signed, D-10 "either arm grants trusted"), not by which `Dirs` field the path came from. The `Dirs{Trusted, External}` struct itself survives only as **pure search paths** (D-11) — `DiscoverAllTiered`/`resolveBinaryDetailed` should search both directories for a binary by name, then classify the tier of whatever is found via verification, not via which field yielded the hit. The D-14 shadow-detection/warn-log pattern (lines 396-406, `logger.Warn("plugin binary name shadowed...")`) becomes obsolete in its current form per D-11's own doc note at CONTEXT.md line 43 — but the fail-loudly-by-name discipline it embodies (never silent) should be preserved for whatever replaces it (e.g., a name colliding across directories with different verified tiers).

**Doc-comment discipline to preserve**: every exported function in this file carries an extensive rationale comment naming the exact decision/task IDs it encodes (e.g. "T-11-01", "D-11", "D-14") — new/modified functions must keep this same citation style so future readers can trace behavior back to CONTEXT.md decisions.

---

### `kernel/pluginhost/host.go` (modified, `launch`) — coexistence arm (D-10)

**Analog:** itself, lines 886-936 (the existing `VerifyTrustedBinary` call site and `manifestUnverifiedError`)

**Current single-arm gate** (lines 920-936):
```go
var manifestHash string
if tier == TierTrusted {
	hash, verifyErr := VerifyTrustedBinary(src.Plugin, binPath)
	if verifyErr != nil {
		return nil, &manifestUnverifiedError{
			instance:    name,
			plugin:      src.Plugin,
			displayName: instanceDisplayName,
			tier:        tier,
			currentHash: hash,
		}
	}
	manifestHash = hash
}
```
Per D-10, this becomes a two-arm OR: verify via `VerifyTrustedBinary` (link-time) OR via the new signed-provenance verifier — either succeeding grants `TierTrusted`; both failing surfaces the same fail-loudly-by-name error shape. Mirror the existing `manifestUnverifiedError` struct pattern (host.go lines 97-139, modeled on `pinMismatchError`) for whatever combined/named error type wraps a "neither arm verified" outcome — structured fields (`instance`, `plugin`, `displayName`, `tier`, `currentHash`), an `Error()` method naming instance/plugin/hash, an `Unwrap()` making `errors.Is` work against a sentinel, and a `toLaunchFailure()` converter. Per D-13 (canonical refs), verification must never demote-and-run — this coexistence check happens BEFORE `exec.Command`, exactly like today.

**Sentinel + LaunchFailure-reason pattern** (host.go lines 42-57, `ErrPinMismatch`/`LaunchFailurePinMismatch`):
```go
var ErrPinMismatch = errors.New("pluginhost: external plugin binary does not match its pinned hash")
const LaunchFailurePinMismatch = "pin_mismatch"
```
A parallel named constant (e.g. `LaunchFailureProvenanceUnverified`) should be added the same way — not an inline string literal — for the new failure reason surfaced to callers/UI.

---

### `kernel/config/types.go` (modified doc comment on `PluginsConfig`)

**Analog:** itself, lines 32-65

**Current comment asserting directory-derived trust** (line 39-40, being rewritten per D-11):
```go
// tagged pluginhost.TierExternal — never pluginhost.TierTrusted —
// because trust is derived purely from WHICH directory a binary
// resolved from, never from anything the binary declares about
// itself.
```
This exact sentence is the one CONTEXT.md line 69 names as needing rewriting: replace "trust is derived purely from WHICH directory" with the provenance-based framing (trust derives from a verified link-time or signed manifest entry; `Dir`/`ExternalDir` are pure search paths per D-11). Keep the same terse, decision-ID-citing doc style used throughout this struct.

---

### `cmd/topos-manifest/main.go` (existing — pattern for the release-side signer)

**Analog:** itself, full file (43 lines)

**CLI generator shape to mirror for the topos-plugins release signer** (lines 20-42):
```go
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "topos-manifest: refusing to run with zero binary arguments — a mis-wired build recipe must fail loudly, not silently produce an empty manifest (RESEARCH Pitfall 6)")
		fmt.Fprintln(os.Stderr, "usage: topos-manifest <plugin-binary-path> [<plugin-binary-path> ...]")
		os.Exit(1)
	}

	entries, err := pluginhost.ManifestEntriesForBinaries(os.Args[1:]...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "topos-manifest: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(pluginhost.FormatManifest(entries))
}
```
Reuses `pluginhost.ManifestEntriesForBinaries` + `pluginhost.FormatManifest` (binaryhash.go's one hashing convention, transitively) — canonical refs call this "candidate to grow a signing/release mode or to pattern the topos-plugins release tooling after." The release-side generator (living in the new `topos-plugins` sibling repo, D-04) should follow the identical "refuse loudly on zero/misconfigured input, print exactly one machine-readable artifact to stdout, nothing else" shape, then additionally sign the resulting manifest with the ed25519 private key (GitHub Actions secret, D-02) and emit manifest+signature as sibling files alongside the release binaries (D-07).

---

### `web/e2e/specs/16-*.spec.ts` (new, only if tier UI surfacing changes — Claude's discretion per CONTEXT "Failure behavior & operator visibility")

**Analog:** `web/e2e/specs/11-external-tier-badge.spec.ts`, `web/e2e/specs/11-untrusted-add.spec.ts`, `web/e2e/specs/11-binary-changed-repin.spec.ts`

These three specs are the existing regression net for tier/badge/consent UI — per TRUST-03 (external path unchanged) and the phase's own "no new UI surface" scoping, any new spec should assert on the SAME chip/badge/interstitial surfaces, only adding coverage for: (1) a link-time-trusted binary still shows trusted with no badge, (2) a signed-provenance-trusted binary (D-04's proving artifact / any signed-fixture manifest) also shows trusted with no badge, (3) each of TRUST-04's three closed escalation paths (config edit, file drop, name-shadowing) now surfaces as external/untrusted rather than trusted. Read one of these three specs directly during planning to mirror its exact page-object/selector conventions before writing new assertions — do not redesign the badge/consent UI itself.

---

## Shared Patterns

### Fail-loudly-by-name, never demote-and-run (D-13, PD-04)
**Source:** `kernel/pluginhost/manifest.go` lines 32-40 (`ErrManifestUnverified` doc comment), `kernel/pluginhost/host.go` lines 886-901 (launch's manifest-verification doc comment)
**Apply to:** `provenance.go`'s new verifier, `discover_binaries.go`'s tier computation, `host.go`'s coexistence gate — every verification failure must name the binary and the specific cause (missing manifest, bad signature, hash mismatch, unknown key ID), collapse to a wrapped sentinel error, and NEVER fall back to running the binary at a lower-trust tier silently or at all under the trusted label.

### One-hashing-convention discipline
**Source:** `kernel/pluginhost/binaryhash.go` (`HashBinary`)
**Apply to:** All new provenance code — never reimplement SHA-256 hashing; always call `HashBinary`.

### Test-only override seam, explicitly forbidden in production code
**Source:** `kernel/pluginhost/manifest.go` lines 268-314 (`OverrideBuildManifest`/`OverrideBuildManifestFromDir`)
**Apply to:** Any new signed-manifest test injection point — same guarded package-level state + `sync.RWMutex` + defensive-copy + restore-func shape, with an explicit "MUST NEVER be called from production code" doc comment.

### Structured launch-failure error type
**Source:** `kernel/pluginhost/host.go` lines 42-57, 97-140 (`ErrPinMismatch`, `pinMismatchError`, `LaunchFailurePinMismatch`)
**Apply to:** The new provenance-unverified error type and its `LaunchFailure` reason constant — struct with named fields, `Error()`, `Unwrap()` to a sentinel, `toLaunchFailure()` converter.

### Decision-ID doc-comment citation style
**Source:** pervasive across `kernel/pluginhost/*.go` and `kernel/config/types.go` (e.g. "D-11", "T-11-01", "13-05-PLAN.md Task 3")
**Apply to:** All new/modified files in this phase — every non-trivial function or struct field's doc comment should cite the CONTEXT.md decision ID(s) it encodes, matching the existing convention so future readers can trace behavior back to 16-CONTEXT.md.

## No Analog Found

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `topos-plugins/.github/workflows/release.yml` (new sibling repo) | config (CI workflow) | event-driven (tag push) | No existing GitHub Actions release-signing workflow in this repo to copy from; planner should pattern this after the Makefile's `MANIFEST_GEN_*` recipes for WHAT gets hashed/signed, and any existing `.github/workflows/*.yml` in this repo (if present) for CI conventions — check `.github/workflows/` directly during planning. |
| ed25519 key generation/rotation tooling (D-03 embedded key SET) | utility | file-I/O (one-time keygen) | No prior in-repo cryptographic-signing code exists; this is genuinely new territory — use Go stdlib `crypto/ed25519` (`GenerateKey`, `Sign`, `Verify`) directly per D-01's "near-zero new dependencies" intent. |

## Metadata

**Analog search scope:** `kernel/pluginhost/`, `kernel/config/`, `cmd/topos-manifest/`, `Makefile`, `web/e2e/specs/`
**Files scanned:** `manifest.go`, `manifest_test.go`, `manifestgate_test.go`, `discover_binaries.go`, `discover_binaries_test.go`, `binaryhash.go`, `host.go`, `config/types.go`, `cmd/topos-manifest/main.go`, `Makefile` (MANIFEST_* recipes), three `web/e2e/specs/11-*.spec.ts` files
**Pattern extraction date:** 2026-08-19
**RESEARCH.md:** does not exist for this phase (research skipped) — all patterns extracted from CONTEXT.md's canonical_refs and direct codebase reads.
