---
phase: 16-provenance-based-plugin-trust
plan: 01
subsystem: security
tags: [ed25519, plugin-trust, provenance, pluginhost, cli]

# Dependency graph
requires: []
provides:
  - "Signed release-manifest trust arm end-to-end: on-disk format (D-05/D-06/D-07), producers (BuildProvenanceManifest/SignProvenanceManifest), verifier (VerifySignedProvenance), compiled-in accepted key set (D-03/D-12), TEST-ONLY override seam (OverrideProvenanceKeys)"
  - "EvaluateTrust: the single trust authority launch (host.go) now consults, coexisting with the link-time build manifest arm (D-10) — either arm grants TierTrusted, neither silently substitutes for the other"
  - "cmd/topos-provenance CLI (keygen/sign/verify) wrapping the kernel's own producers/verifier — the release-side and install-time (D-09) verification entry point"
  - "make provenance-check: a committed hermetic round-trip gate (keygen -> sign -> verify -> tamper -> refuse) exercising the real -ldflags provenanceKeysExtra link-time seam"
affects: [16-02, 16-03, 16-04, 16-05]

# Actuals (#2632)
actuals:
  tokens: 19696
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Signed release manifest as a second evidence source composing with, never competing against, the link-time build manifest (D-10 'one verifier with two evidence sources')"
    - "Exhaustive-scan-then-decide over all candidate manifests before resolving MATCH > NAMED-MISMATCH > no-evidence precedence (D-08), so directory read order never affects the outcome"
    - "Link-time-only key injection (embeddedProvenanceKeys compiled-in slice + provenanceKeysExtra -ldflags -X var), mirroring buildManifest's existing D-12 discipline exactly"
    - "Multi-error Unwrap() []error (Go 1.20+) so one wrapped launch error resolves to both the wire-vocabulary-preserving sentinel and the arm-specific sentinel"

key-files:
  created:
    - kernel/pluginhost/provenance.go
    - kernel/pluginhost/provenance_test.go
    - cmd/topos-provenance/main.go
    - scripts/provenance-smoke.sh
  modified:
    - kernel/pluginhost/host.go
    - Makefile
    - .gitignore

key-decisions:
  - "Widened VerifySignedProvenance's return shape to (hash, evidence, diagnostics, err) beyond the plan action text's three-value sketch — required so per-candidate diagnostics (unknown key id, platform mismatch, etc.) can reach Trust.Diagnostics, which the same task requires EvaluateTrust to carry out to a caller holding a logger."
  - "manifestUnverifiedError now implements Unwrap() []error (Go 1.20+ multi-error form) instead of a single Unwrap() error, so errors.Is resolves BOTH ErrManifestUnverified (preserving the existing wire vocabulary, TRUST-03) and ErrProvenanceUnverified (required by the digest-mismatch acceptance criterion) from one wrapped launch error."
  - "Added /topos-provenance to .gitignore's stray-root-binary list, matching the existing /topos-manifest convention (a bare `go build ./cmd/topos-provenance` writes a binary to the repo root)."

patterns-established:
  - "Trust struct (Tier, Hash, Evidence, Diagnostics) as EvaluateTrust's uniform result shape — a future evidence source (or Phase 17's link-time-arm removal) changes EvaluateTrust's body, never its callers' contract."

requirements-completed: [TRUST-01]

coverage:
  - id: D1
    description: "Signed release-manifest trust arm: format, producers, verifier, key set, test seam — a plugin binary with a validly-signed manifest launches TierTrusted with an empty link-time build manifest"
    requirement: "TRUST-01"
    verification:
      - kind: unit
        ref: "kernel/pluginhost/provenance_test.go#TestLaunch_Provenance_TracerSuccessPath"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/provenance_test.go#TestVerifySignedProvenance_UnknownKeyIDGrantsNoTrust"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/provenance_test.go#TestVerifySignedProvenance_CorruptedSignatureBytesGrantsNoTrust"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/provenance_test.go#TestVerifySignedProvenance_SignatureOverDifferentBytesGrantsNoTrust"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/provenance_test.go#TestVerifySignedProvenance_PlatformMismatchGrantsNoTrust"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/provenance_test.go#TestVerifySignedProvenance_DigestMismatchRefusesAndCreatesNoSubprocess"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/provenance_test.go#TestVerifySignedProvenance_TwoManifestsOnlySecondNamesBinary"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/provenance_test.go#TestVerifySignedProvenance_MultipleManifestsCoexistExhaustiveScanOrderIndependent"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/provenance_test.go#TestEvaluateTrust_ManifestNamingDifferentBinaryDoesNotAffectFirst"
        status: pass
    human_judgment: false
  - id: D2
    description: "launch's trusted-tier gate decides through the single EvaluateTrust authority instead of calling VerifyTrustedBinary directly; the whole pre-existing kernel/pluginhost suite (pin, tier, manifest-gate) stays green"
    requirement: "TRUST-01"
    verification:
      - kind: unit
        ref: "kernel/pluginhost/manifestgate_test.go (all 10 pre-existing cases)"
        status: pass
      - kind: unit
        ref: "go test ./kernel/pluginhost/... -count=1"
        status: pass
    human_judgment: false
  - id: D3
    description: "cmd/topos-provenance CLI (keygen/sign/verify) wraps the kernel's own producers/verifier; keygen never leaks the private key, sign refuses on zero binaries, verify enforces the compiled-in key policy and refuses on an empty directory"
    verification:
      - kind: other
        ref: "go run ./cmd/topos-provenance (no args) exits 1, stderr contains 'usage'"
        status: pass
      - kind: other
        ref: "go run ./cmd/topos-provenance sign --key-id x --out-dir /tmp with zero binary args exits 1"
        status: pass
      - kind: other
        ref: "go run ./cmd/topos-provenance keygen --key-id t --out-dir <tmp>: exits 0, creates t.key (mode 600) + t.pub, stdout matches ^t=[base64]+$, private key content absent from stdout+stderr"
        status: pass
      - kind: other
        ref: "manual sign -> verify -> tamper round trip against a real fixture binary (see 16-01-SUMMARY session log): sign succeeds, verify succeeds naming the manifest, tampered-binary verify fails naming the binary"
        status: pass
    human_judgment: false
  - id: D4
    description: "scripts/provenance-smoke.sh is a committed, hermetic gate wired in as `make provenance-check`, proving the producer/verifier/key-injection seam agree end to end including a real -ldflags rebuild"
    verification:
      - kind: other
        ref: "make provenance-check"
        status: pass
      - kind: other
        ref: "./scripts/provenance-smoke.sh run twice consecutively — both exit 0, no leftover /tmp/topos-provenance-smoke.* directories"
        status: pass
    human_judgment: false

# Metrics
duration: 28min
completed: 2026-08-20
status: complete
---

# Phase 16 Plan 1: Provenance-Based Plugin Trust — Signed Release-Manifest Arm Summary

**Ed25519-signed release manifests now grant TierTrusted alongside the existing link-time build manifest — a plugin binary with a validly-signed `*.provenance.json`/`*.provenance.sig` pair launches trusted even when the kernel's compiled-in build manifest is empty, verified via one authority (`EvaluateTrust`) that composes both evidence sources.**

## Performance

- **Duration:** 28 min
- **Started:** 2026-08-20T00:57:41+01:00 (base commit)
- **Completed:** 2026-08-20T01:25:43+01:00
- **Tasks:** 3
- **Files modified:** 7 (4 created, 3 modified)

## Accomplishments

- `kernel/pluginhost/provenance.go`: the full signed-manifest format (D-05/D-06/D-07), an exhaustive multi-manifest scanner honoring D-08's coexistence rule (MATCH > NAMED-MISMATCH > no-evidence, decided only after scanning every candidate), a link-time-only accepted key set (D-03/D-12: `embeddedProvenanceKeys` compiled-in + `provenanceKeysExtra` via `-ldflags -X`), and `EvaluateTrust` as the single trust authority consulting both the link-time and signed arms (D-10).
- `kernel/pluginhost/host.go`: `launch`'s trusted-tier gate now calls `EvaluateTrust` instead of `VerifyTrustedBinary` directly; `manifestUnverifiedError` widened with a real cause chain (`Unwrap() []error`) so `errors.Is` resolves both `ErrManifestUnverified` (preserving the existing wire vocabulary, TRUST-03) and `ErrProvenanceUnverified` from one returned error.
- `cmd/topos-provenance/main.go`: `keygen` (0600 private key, never printed), `sign` (hashes via the existing `ManifestEntriesForBinaries`, private key from `--key-file` or `TOPOS_PROVENANCE_SIGNING_KEY` env var — never argv), and `verify` (checks against this binary's OWN compiled-in key policy — the D-09 install-time entry point).
- `scripts/provenance-smoke.sh` + `make provenance-check`: a hermetic, committed round-trip gate that rebuilds the CLI with a real `-ldflags provenanceKeysExtra` injection (proving the link-time seam works, not just an in-process override), then proves tampered binary / missing signature / tampered manifest JSON each refuse by name.

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end "a signed binary launches trusted"** — `08a47d5` (feat)
2. **Task 2: topos-provenance CLI — keygen, sign, verify** — `5500bee` (feat)
3. **Task 3: Committed round-trip gate — scripts/provenance-smoke.sh** — `d49d326` (test)

**Plan metadata:** commit pending (this SUMMARY + REQUIREMENTS.md, parallel/worktree mode)

## Files Created/Modified

- `kernel/pluginhost/provenance.go` — signed release-manifest format, producers, verifier, key set, `EvaluateTrust` authority
- `kernel/pluginhost/provenance_test.go` — tracer success path + every named failure cause + D-08 exhaustive-scan invariant
- `cmd/topos-provenance/main.go` — keygen/sign/verify CLI over the kernel's own producers/verifier
- `scripts/provenance-smoke.sh` — hermetic round-trip gate, real `-ldflags` rebuild
- `kernel/pluginhost/host.go` — `launch`'s trusted-tier gate now calls `EvaluateTrust`; `manifestUnverifiedError` widened with a cause chain
- `Makefile` — `provenance-check` target + `.PHONY` entry
- `.gitignore` — `/topos-provenance` added to the stray-root-binary list

## Decisions Made

- Widened `VerifySignedProvenance`'s return shape to `(hash, evidence, diagnostics, err)` beyond the plan action text's three-value sketch (`hash, evidence, err`) — the extra `diagnostics []string` return is what lets per-candidate failure reasons (unknown key id, platform mismatch, corrupted signature, etc.) reach `Trust.Diagnostics`, which the same task's action text explicitly requires `EvaluateTrust` to carry out to a caller holding a logger. Without this the requirement was unsatisfiable with the literal three-value signature.
- `manifestUnverifiedError` now implements `Unwrap() []error` (Go 1.20+ multi-error form) rather than a single `Unwrap() error` — this lets `errors.Is` resolve to both `ErrManifestUnverified` (the pre-existing sentinel every caller/test already checks, and the one TRUST-03's wire-vocabulary-preservation depends on) and, when the underlying refusal came from the signed arm, `ErrProvenanceUnverified` — satisfying the plan's explicit acceptance criterion (`errors.Is(err, ErrProvenanceUnverified)` after a launch-level digest tamper) without picking a single "primary" cause or losing the existing behavior.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Widened `VerifySignedProvenance`'s signature to surface per-candidate diagnostics**
- **Found during:** Task 1, while implementing `EvaluateTrust`
- **Issue:** The plan's action text sketches `VerifySignedProvenance(dirs, name, path) (hash, evidence, err)`. With only those three return values, a candidate manifest that failed signature/schema/platform verification had nowhere to report its diagnostic string, yet the same task requires `EvaluateTrust` to "carry every diagnostic string out through `Trust.Diagnostics`."
- **Fix:** Added a fourth return value, `diagnostics []string`, documented inline in `VerifySignedProvenance`'s own doc comment as an explicit, reasoned widening of the sketched signature.
- **Files modified:** `kernel/pluginhost/provenance.go`
- **Verification:** `TestVerifySignedProvenance_UnknownKeyIDGrantsNoTrust` and siblings assert on the returned diagnostics directly.
- **Committed in:** `08a47d5`

**2. [Rule 1 - Bug] `manifestUnverifiedError` needed a real cause chain, not a cause string**
- **Found during:** Task 1, first test run of `TestVerifySignedProvenance_DigestMismatchRefusesAndCreatesNoSubprocess`
- **Issue:** An initial implementation carried `cause string` and unwrapped only to `ErrManifestUnverified`. The plan's own acceptance criteria require `errors.Is(err, ErrProvenanceUnverified)` to be true for a launch-level digest-tamper refusal via the signed arm — impossible with a single-sentinel `Unwrap() error`.
- **Fix:** Changed the field to `cause error` and implemented `Unwrap() []error`, returning both `ErrManifestUnverified` and the underlying cause (when present) — Go 1.20+'s multi-error unwrap support (this repo is on Go 1.25).
- **Files modified:** `kernel/pluginhost/host.go`
- **Verification:** `TestVerifySignedProvenance_DigestMismatchRefusesAndCreatesNoSubprocess` passes; the full pre-existing `manifestgate_test.go` suite (which asserts `errors.Is(err, ErrManifestUnverified)`) is unaffected.
- **Committed in:** `08a47d5`

**3. [Rule 3 - Blocking] `scripts/provenance-smoke.sh`'s own pristine-backup files were picked up by `verify`'s directory scan**
- **Found during:** Task 3, first `make provenance-check` run
- **Issue:** The script copied restore-point backups (`topos-plugin-mock.pristine`, etc.) directly into the temp work directory that `verify --dir` also scans; since `cmd/topos-provenance verify`'s default target list matches any file with the `topos-plugin-` prefix, `topos-plugin-mock.pristine` was picked up as a second, unsigned candidate binary and failed verification, breaking the gate's own happy path.
- **Fix:** Moved all pristine backups into a `backup/` subdirectory that `verify --dir "$WORK"`'s non-recursive scan never sees.
- **Files modified:** `scripts/provenance-smoke.sh`
- **Verification:** `make provenance-check` and two consecutive direct invocations of `./scripts/provenance-smoke.sh` both exit 0 with no leftover temp directories.
- **Committed in:** `d49d326`

**4. [Rule 2 - Missing Critical] Added `/topos-provenance` to `.gitignore`**
- **Found during:** Task 2, pre-commit `git status` check
- **Issue:** A bare `go build ./cmd/topos-provenance` (run during manual CLI verification) leaves a `topos-provenance` binary at the repo root, matching this repo's own documented "stray top-level binaries from a bare `go build ./...`" convention — which already lists `/topos-manifest` for the sibling CLI but had no entry for the new one.
- **Fix:** Added `/topos-provenance` to the existing stray-binary block in `.gitignore`, alongside `/topos-manifest`.
- **Files modified:** `.gitignore`
- **Verification:** `git status --short` shows no untracked stray binary after a bare build.
- **Committed in:** `5500bee`

---

**Total deviations:** 4 auto-fixed (1 bug, 2 missing-critical, 1 blocking)
**Impact on plan:** All four were necessary to satisfy the plan's own stated acceptance criteria or this repo's existing conventions. No scope creep — no new files, endpoints, or behavior beyond what Task 1–3's action text and acceptance criteria specify.

## Issues Encountered

None beyond the four deviations documented above, each resolved within the task that surfaced it.

## User Setup Required

None — no external service configuration required. `embeddedProvenanceKeys` is deliberately empty in this plan (16-04 adds the real `topos-plugins` signing key); no operator action is needed until that plan lands.

## Next Phase Readiness

- The signed release-manifest trust arm is proven end-to-end (tracer path + every named failure) and committed, with a hermetic regression gate (`make provenance-check`) that also exercises the real `-ldflags` key-injection seam.
- 16-02 (tier rewrite consuming `EvaluateTrust`/D-11's pure-search-path directories), 16-03 (escalation suite), 16-04 (real topos-plugins signing key + `embeddedProvenanceKeys` populated, standing up the sibling repo), and 16-05 (install-time verification via `topos-provenance verify`, D-09) can all build directly on `EvaluateTrust`, `AcceptedProvenanceKeys`, and the CLI without further changes to this plan's surface.
- No blockers. `go build ./...`, `go vet ./kernel/... ./cmd/...`, `go test ./... -count=1`, `make test`, and `make provenance-check` all pass locally.

---
*Phase: 16-provenance-based-plugin-trust*
*Completed: 2026-08-20*

## Self-Check: PASSED

- `kernel/pluginhost/provenance.go` — FOUND
- `kernel/pluginhost/provenance_test.go` — FOUND
- `cmd/topos-provenance/main.go` — FOUND
- `scripts/provenance-smoke.sh` — FOUND
- Commit `08a47d5` (Task 1) — FOUND in `git log --oneline --all`
- Commit `5500bee` (Task 2) — FOUND in `git log --oneline --all`
- Commit `d49d326` (Task 3) — FOUND in `git log --oneline --all`
- All plan-level `<verification>` commands re-run and green: `go build ./...`, `go vet ./kernel/... ./cmd/...`, `go test ./... -count=1`, `make provenance-check`, `make test`
- Named tracer test `TestLaunch_Provenance_TracerSuccessPath` in `kernel/pluginhost/provenance_test.go` — PASS
