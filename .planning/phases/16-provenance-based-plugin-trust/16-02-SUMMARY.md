---
phase: 16-provenance-based-plugin-trust
plan: 02
subsystem: security
tags: [plugin-trust, provenance, pluginhost, d-11, tier-derivation]

# Dependency graph
requires:
  - phase: 16-01
    provides: "EvaluateTrust (the single provenance authority: link-time build manifest + signed release manifest arms), Trust struct, VerifySignedProvenance, ErrProvenanceUnverified"
provides:
  - "Tier as a pure function of provenance: resolveBinaryDetailed/DiscoverAllTiered evaluate trust per binary via EvaluateTrust; Dirs.Trusted/Dirs.External are pure search paths with no bearing on tier"
  - "launch (host.go) gates directly on resolveBinaryDetailed's returned Trust value instead of re-evaluating trust in a second step"
  - "Provenance-ordered collision rule replacing the obsolete D-14 trusted-shadows-external rule: both candidates are evaluated, whichever earns TierTrusted wins, ties keep the trusted-first search order, every collision and diagnostic logged by name"
  - "Location-independent and reverting-fails-the-suite regression coverage across tier_test.go, discover_binaries_test.go, manifestgate_test.go, and a new TRUST-03 end-to-end regression net in kernel/httpapi/sources_test.go"
affects: [16-03, 16-04, 16-05]

# Actuals (#2632)
actuals:
  tokens: 18472
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Provenance-ordered collision resolution: evaluate both candidate paths via EvaluateTrust, prefer whichever earns TierTrusted, fall back to trusted-first search order only when neither (or both) carry evidence — replaces directory-order-decides-everything"
    - "resolveBinaryDetailed widened to return the full Trust value (Tier, Hash, Evidence, Diagnostics), not just Tier, so launch can gate on one authority's single evaluation instead of re-deriving trust in a second call"
    - "Test isolation: nested global-state overrides (build-manifest override, provenance-key override) within one test function must use the SAME cleanup mechanism (t.Cleanup, not a mix of defer and t.Cleanup) — mixing lets a later t.Cleanup re-instate an earlier override's stale captured state after an earlier defer already cleared it"

key-files:
  created: []
  modified:
    - kernel/pluginhost/discover_binaries.go
    - kernel/pluginhost/host.go
    - kernel/config/types.go
    - kernel/pluginhost/tier_test.go
    - kernel/pluginhost/discover_binaries_test.go
    - kernel/pluginhost/manifestgate_test.go
    - kernel/httpapi/config_test.go
    - kernel/httpapi/sources_test.go
    - kernel/supervisor/externaltier_test.go

key-decisions:
  - "resolveBinaryDetailed's collision branch evaluates BOTH trusted and external candidates unconditionally (rather than short-circuiting once trusted evidence is found on one side) so the operator-facing collision warning can always name which evidence (or absence of it) each side carried — a small extra EvaluateTrust call, accepted per the plan's own per-call-hashing-is-the-price framing (T-16-12)."
  - "launch distinguishes 'binary not found in either directory' from 'binary found but EvaluateTrust refused it' by checking whether resolveBinaryDetailed's returned path is empty — a non-empty path with a non-nil error is always a tamper refusal (manifestUnverifiedError), never a not-found wrap."
  - "Several manifestgate_test.go cases whose premise was 'a manifest exists but simply omits this binary's name, therefore refuse' were rewritten (not merely re-fixtured) because that state is no longer a manifest-gate refusal at all under D-11 — it is a legitimate external-tier resolution that falls to the pin gate. Rewritten to assert the new external-tier-fallback behavior directly (LaunchFailurePinMismatch, never LaunchFailureManifestUnverified), and a NEW case (DescribeOnlyTrialLaunchOfUnprovenBinarySucceeds) explicitly proves the T-13-06 concern this changes: a genuinely evidence-free binary's describeOnly trial launch is now allowed to run, exactly like any other external-tier trial launch (T-11-14's existing accepted risk) — because it was never trusted-tier to begin with."
  - "kernel/supervisor/externaltier_test.go's external-only tracer test was rebuilt under a RENAMED binary (buildRenamedMockPluginDir) instead of reusing buildMockPluginDir's shared 'topos-plugin-mock' fixture: that shared fixture's name+hash is permanently registered in the package's global build-manifest override (supervisor_test.go's installTrustedManifestOnce, never restored), so under D-11 a second reproducible build of the identical source anywhere in that package's test binary run would legitimately resolve TierTrusted too — proving the plan's own success criterion (same bytes, same tier, any directory) rather than a bug. A genuinely evidence-free binary needed a name the global override does not cover."

patterns-established:
  - "Tier derivation as a single-authority call (resolveBinaryDetailed -> EvaluateTrust) consumed by both the launch-time resolver and the discovery-time listing (via the shared evaluateListingTier helper) — a future evidence source or Phase 17's link-time-arm removal changes EvaluateTrust's body, never either caller's contract."

requirements-completed: [TRUST-01, TRUST-03]

coverage:
  - id: D1
    description: "Tier is a pure function of provenance: resolveBinaryDetailed and DiscoverAllTiered derive tier through EvaluateTrust per binary, never from which Dirs field yielded the hit; Dirs.Trusted/Dirs.External are pure search paths"
    requirement: "TRUST-01"
    verification:
      - kind: unit
        ref: "kernel/pluginhost/tier_test.go#TestResolveBinary_ExternalWithSignedManifestResolvesTrustedTier"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/tier_test.go#TestResolveBinary_TrustedWithNoEvidenceResolvesExternalTier"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/tier_test.go#TestResolveBinary_LocationSymmetric"
        status: pass
      - kind: other
        ref: "manual regression-teeth check: temporarily reverting resolveBinaryDetailed's trusted-path branch to unconditional TierTrusted made TestResolveBinary_TrustedWithNoEvidenceResolvesExternalTier fail (restored before commit, git diff clean)"
        status: pass
    human_judgment: false
  - id: D2
    description: "D-11's collision rule replaces the obsolete D-14 trusted-shadows-external rule: both candidates are evaluated on collision, whichever earns TierTrusted wins, ties keep trusted-first ordering, and the collision plus every Trust.Diagnostics entry is logged by name"
    requirement: "TRUST-01"
    verification:
      - kind: unit
        ref: "kernel/pluginhost/tier_test.go#TestResolveBinary_CollisionResolvesToWhicheverCopyCarriesEvidence"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/tier_test.go#TestResolveBinary_CollisionResolvesTrustedAndLogsByName"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/discover_binaries_test.go#TestDiscoverAllTiered_DigestMismatchStillAppearsTaggedExternal"
        status: pass
    human_judgment: false
  - id: D3
    description: "launch (host.go) gates directly on resolveBinaryDetailed's Trust value; a non-nil resolveErr with a non-empty path is a tamper refusal (manifestUnverifiedError), distinct from a genuine not-found; TierExternal falls through to the existing pin block completely untouched"
    requirement: "TRUST-03"
    verification:
      - kind: unit
        ref: "kernel/pluginhost/manifestgate_test.go#TestLaunch_ManifestGate_D10CoexistenceBothArmsGrantTrusted"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/manifestgate_test.go#TestLaunch_ManifestGate_SignedDigestMismatchIsManifestUnverifiedError"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/manifestgate_test.go#TestLaunch_ManifestGate_NoManifestEmbeddedFallsToExternalTierNeverTrusted"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/manifestgate_test.go#TestLaunch_ManifestGate_DescribeOnlyTrialLaunchOfUnprovenBinarySucceeds"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/pin_test.go (all 5 pre-existing cases, unmodified, git diff empty)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The unsigned external consent-and-pin path is byte-for-byte unchanged end to end through the real GET /api/sources HTTP surface: no pin -> pin_mismatch, pinned+matching -> launches, changed bytes under the same pin -> pin_mismatch again, a re-pin write -> launches again"
    requirement: "TRUST-03"
    verification:
      - kind: integration
        ref: "kernel/httpapi/sources_test.go#TestSources_UnsignedExternalBinaryConsentAndPinPathUnchanged"
        status: pass
      - kind: integration
        ref: "kernel/supervisor/externaltier_test.go (both cases, green)"
        status: pass
      - kind: integration
        ref: "kernel/supervisor/externalproof_test.go#TestExternalProof_OutOfRepoBinaryEndToEnd (unmodified, green)"
        status: pass
    human_judgment: false
  - id: D5
    description: "kernel/config/types.go's PluginsConfig doc comments state Dir/ExternalDir are pure search paths; the sentence asserting trust derives from WHICH directory a binary resolved from no longer exists"
    verification:
      - kind: other
        ref: "grep -c 'trust is derived purely from WHICH directory' kernel/config/types.go == 0; grep -ci 'search path' kernel/config/types.go >= 1"
        status: pass
    human_judgment: false

# Metrics
duration: 38min
completed: 2026-08-20
status: complete
---

# Phase 16 Plan 2: Provenance-Based Plugin Trust — Tier Derivation Rewrite Summary

**`resolveBinaryDetailed`/`DiscoverAllTiered` now derive every plugin binary's trust tier by calling `EvaluateTrust` per binary instead of trusting whichever `Dirs` field produced the hit — `Dirs.Trusted`/`Dirs.External` are pure search paths, and the same bytes earn the same tier from either configured directory.**

## Performance

- **Duration:** 38 min
- **Started:** 2026-08-20T00:26:00+01:00 (approx., right after 16-01's worktree forked)
- **Completed:** 2026-08-20T02:03:17+01:00
- **Tasks:** 3
- **Files modified:** 9 (0 created, 9 modified)

## Accomplishments

- `kernel/pluginhost/discover_binaries.go`: `resolveBinaryDetailed` now returns the full `Trust` value from `EvaluateTrust(dirs, name, path)` instead of a directory-derived `Tier`; `ResolveBinary` stays a thin `(path, tier, err)` wrapper. `DiscoverAllTiered` evaluates every discovered binary's trust per-name via a new `evaluateListingTier` helper — hashing on every call, caching nothing, and never aborting the listing on a tamper refusal (tagged `TierExternal` instead, per T-16-11). The obsolete D-14 "trusted shadows external" rule is replaced by a provenance-ordered collision rule: both candidate paths are evaluated, whichever earns `TierTrusted` wins, ties keep the existing trusted-first search order, and every collision plus `Trust.Diagnostics` entry is logged by name.
- `kernel/pluginhost/host.go`: `launch` gates directly on the `Trust` value `resolveBinaryDetailed` already produced — no second `EvaluateTrust` call. A non-nil `resolveErr` paired with a non-empty `binPath` is a tamper refusal (`*manifestUnverifiedError`); an empty `binPath` is a genuine not-found. `TierExternal` falls through to the existing pin block completely untouched (TRUST-03).
- `kernel/config/types.go`: `PluginsConfig.Dir`/`ExternalDir` doc comments rewritten to state both fields are search paths only; the sentence asserting trust derives from which directory a binary resolved from is gone.
- Three `kernel/pluginhost` suites (`tier_test.go`, `discover_binaries_test.go`, `manifestgate_test.go`) realigned onto the provenance model — every case that previously earned `TierTrusted` from a bare directory placement now installs real evidence — plus new cases proving location-independence, location symmetry (a single table over both placements), evidence-based collision resolution, D-10 coexistence (link-time alone / signed alone / both), and a digest-mismatch case asserting the `*manifestUnverifiedError` type directly.
- Two out-of-package suites (`kernel/httpapi/config_test.go`, `kernel/httpapi/sources_test.go`) and `kernel/supervisor/externaltier_test.go` realigned, plus the explicit TRUST-03 regression net this phase owes: `TestSources_UnsignedExternalBinaryConsentAndPinPathUnchanged` exercises the real `pluginhost.Discover`/launch machinery end to end through `GET /api/sources`, proving the unsigned consent-and-pin lifecycle (no pin → pinned → tampered → re-pinned) is unchanged.

## Task Commits

Each task was committed atomically:

1. **Task 1: Tier derives from provenance, not from Dirs** — `4b1891d` (feat)
2. **Task 2: Realign the existing tier, discovery, and manifest-gate suites** — `bdec52d` (test)
3. **Task 3: Realign the out-of-package suites and prove the external path unchanged** — `68d002d` (test)

**Plan metadata:** commit pending (this SUMMARY + REQUIREMENTS.md, parallel/worktree mode — STATE.md/ROADMAP.md are updated centrally by the orchestrator)

## Files Created/Modified

- `kernel/pluginhost/discover_binaries.go` — provenance-driven `resolveBinaryDetailed`/`DiscoverAllTiered`, `evaluateListingTier`, rewritten `Tier`/`Dirs` doc comments
- `kernel/pluginhost/host.go` — `launch` gates on the resolver's `Trust` value directly, distinguishes tamper refusal from not-found
- `kernel/config/types.go` — `PluginsConfig` doc rewrite: pure search paths, no directory-derived trust
- `kernel/pluginhost/tier_test.go` — realigned + 4 new tests (location-independence, negative, location-symmetric table, evidence-based collision)
- `kernel/pluginhost/discover_binaries_test.go` — realigned + 1 new test (digest-mismatch still appears, tagged external)
- `kernel/pluginhost/manifestgate_test.go` — realigned (rewritten where the old premise no longer holds) + 2 new tests (D-10 coexistence, digest-mismatch error type)
- `kernel/httpapi/config_test.go` — 1 test gains evidence installation
- `kernel/httpapi/sources_test.go` — new TRUST-03 end-to-end regression net
- `kernel/supervisor/externaltier_test.go` — external-only tracer rebuilt under a renamed binary to avoid the package's shared trusted-fixture collision

## Decisions Made

- `resolveBinaryDetailed`'s collision branch always evaluates BOTH candidates (never short-circuits) so the operator-facing warning can name which evidence each side carried, or its absence — accepted extra hashing cost per T-16-12.
- `launch` tells "not found" from "found but refused" by checking whether `resolveBinaryDetailed`'s returned path is empty, rather than adding a second boolean return value.
- Several `manifestgate_test.go` cases whose premise was "a manifest exists but doesn't mention this binary, therefore refuse" were rewritten rather than re-fixtured, because that state is no longer a manifest-gate refusal under D-11 — it's a legitimate external-tier resolution that now falls to the pin gate. Added `TestLaunch_ManifestGate_DescribeOnlyTrialLaunchOfUnprovenBinarySucceeds` to explicitly prove the resulting T-13-06 behavior change is intentional, not a regression.
- `kernel/supervisor/externaltier_test.go`'s external-only tracer test was rebuilt under a renamed binary rather than reusing the package's shared `buildMockPluginDir` fixture, whose name+hash is permanently registered in a global (never-restored) build-manifest override — reusing it would make the "external-only" proof accidentally resolve trusted under D-11's own success criterion (same bytes, same tier, any directory), which is correct behavior but not what that specific test needed to prove.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Test-isolation ordering hazard: mixing `defer restore()` with a nested `t.Cleanup`-based override leaked global build-manifest state**
- **Found during:** Task 2, while realigning `TestLaunch_ManifestGate_ReconcileClearsFailureOnceRestored` to install a genuinely mismatched (not empty) first-override manifest entry
- **Issue:** The test installed a first `OverrideBuildManifest` via `defer restore()`, then a second (via `installTrustedManifest`'s `t.Cleanup`) nested inside the same test function. `t.Cleanup` callbacks run strictly AFTER a function's own deferred statements return, in LIFO order among themselves — so the SECOND override's cleanup (registered later, therefore running first among cleanups) restored to its own captured "previous" state, which was the FIRST override's value, re-instating it AFTER the first override's own `defer restore()` had already correctly cleared it. With the first override changed from an empty map to a genuinely mismatched entry (`{"topos-plugin-mock": "0"*64}`), this leaked a mismatched manifest entry into every later test in the package's run — silently broke `pin_test.go`'s external-tier tests, which never otherwise touch the build manifest at all.
- **Fix:** Changed the first override to also use `t.Cleanup` (matching `installTrustedManifest`'s own mechanism), so both nested overrides restore in correct LIFO order relative to each other.
- **Files modified:** `kernel/pluginhost/manifestgate_test.go`
- **Verification:** `go test ./kernel/pluginhost/... -count=1` (full package, in order) passes; `go test ./kernel/pluginhost/... -count=1 -race` passes.
- **Committed in:** `bdec52d` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug)
**Impact on plan:** The fix was necessary for the realigned test suite (Task 2's own scope) to be correct and green; the underlying pattern (mixing `defer` and `t.Cleanup` for the same global state) was pre-existing but latent — it only became visible once Task 2's edit changed a leaked override from an empty (functionally invisible) map to a genuinely mismatched one. No scope creep — fixed within the one test that exhibited it.

## Issues Encountered

None beyond the deviation documented above, resolved within the task that surfaced it.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Tier is a pure function of provenance across the whole discovery/resolve/launch surface; `Dirs.Trusted`/`Dirs.External` are documented and implemented as pure search paths.
- Every existing Go suite across `kernel/pluginhost`, `kernel/httpapi`, and `kernel/supervisor` is green against the new model; `kernel/pluginhost/pin_test.go` is unmodified.
- `go build ./...`, `go vet ./...`, `go test ./... -count=1`, `make test` (including the cgo `test-signal` target), and `make provenance-check` all pass locally.
- 16-03 (the TRUST-04 escalation suite closing the config-edit/file-drop/shadowing paths by test) can build directly on this plan's collision rule and location-independence proofs without further changes to this plan's surface. 16-04/16-05 are unaffected by this plan's scope.
- No blockers.

---
*Phase: 16-provenance-based-plugin-trust*
*Completed: 2026-08-20*

## Self-Check: PASSED

- `kernel/pluginhost/discover_binaries.go` — FOUND
- `kernel/pluginhost/host.go` — FOUND
- `kernel/config/types.go` — FOUND
- `kernel/pluginhost/tier_test.go` — FOUND
- `kernel/pluginhost/discover_binaries_test.go` — FOUND
- `kernel/pluginhost/manifestgate_test.go` — FOUND
- `kernel/httpapi/config_test.go` — FOUND
- `kernel/httpapi/sources_test.go` — FOUND
- `kernel/supervisor/externaltier_test.go` — FOUND
- Commit `4b1891d` (Task 1) — FOUND in `git log --oneline --all`
- Commit `bdec52d` (Task 2) — FOUND in `git log --oneline --all`
- Commit `68d002d` (Task 3) — FOUND in `git log --oneline --all`
- All plan-level `<verification>` commands re-run and green: `go build ./...`, `go vet ./...`, `go test ./... -count=1`, `make test`, `make provenance-check`
- Regression-teeth check performed live (temporarily reverted resolveBinaryDetailed to directory-derived trust, confirmed `TestResolveBinary_TrustedWithNoEvidenceResolvesExternalTier` fails, restored — `git diff` clean afterward)
- `kernel/pluginhost/pin_test.go` — unmodified (`git diff --stat` empty) and green
