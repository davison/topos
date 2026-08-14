---
phase: 13-per-item-curation-installable-app
plan: 05
subsystem: security
tags: [go, plugin-trust, build-provenance, sha256, makefile, ldflags]

requires:
  - phase: 11-external-plugins-the-trust-boundary
    provides: "Tier (TierTrusted/TierExternal), ResolveBinary, external-tier content-hash pinning ([plugins.pins]), LaunchFailure/LaunchFailures soft-failure recording"
provides:
  - "kernel/pluginhost/manifest.go: FormatManifest/ParseManifest, TrustManifest, VerifyTrustedBinary, ManifestEntriesForBinaries, OverrideBuildManifest(FromDir) test seam"
  - "cmd/topos-manifest: build-time manifest generator consuming an explicit binary-path list"
  - "Makefile: build/build-portable/e2e/dev all build plugins before the kernel and link a real manifest via -ldflags -X"
  - "kernel/pluginhost/host.go: launch()'s manifest-verification gate (manifestUnverifiedError, LaunchFailureManifestUnverified) and the D-14 shadowing advisory (Plugin.shadowed, SourceHealth.LaunchAdvisory, LaunchAdvisoryShadowed)"
  - "kernel/httpapi/sources.go: sourceStatus.LaunchAdvisory (launch_advisory, omitempty)"
affects: [13-06, 14-google-drive]

actuals:
  tokens: 27081
  tasks: 3
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Link-time trust manifest via -ldflags -X (no generated .go file, no runtime file read) — the pattern for any future kernel-side fact that must be fixed at build time and never editable at run time"
    - "Additive manifest-override test seam (OverrideBuildManifest/OverrideBuildManifestFromDir with a returned restore func) mirroring the existing pin-mismatch soft-failure pattern"

key-files:
  created:
    - kernel/pluginhost/manifest.go
    - kernel/pluginhost/manifest_test.go
    - kernel/pluginhost/manifestgate_test.go
    - cmd/topos-manifest/main.go
  modified:
    - kernel/pluginhost/host.go
    - kernel/pluginhost/discover_binaries.go
    - kernel/httpapi/sources.go
    - Makefile
    - docs/api.md
    - docs/plugin-contract.md
    - cmd/topos/shutdown_signal_test.go

key-decisions:
  - "PD-03 (carried from PLAN.md): the manifest is embedded via -ldflags -X against a committed, default-empty package variable — never a generated .go source file — so go build ./... stays clean on every checkout and no build ever dirties the working tree."
  - "PD-04 (carried from PLAN.md): a kernel with no manifest embedded refuses every trusted-tier launch by name; there is deliberately no directory-derived-trust fallback."
  - "PD-05 (carried from PLAN.md): the D-14 shadowing advisory transports on a new sibling field launch_advisory, never on launch_failure, which keeps its published never-launched-at-all meaning."
  - "Makefile: MANIFEST_GEN_PLUGINS/_PORTABLE/_E2E hold the one-place-only 'go run ./cmd/topos-manifest <list>' invocation text; build/build-portable/dev/e2e reference these macros rather than re-typing the invocation, so the generator call and its binary list can never drift independently of each other."
  - "Test-manifest scoping: kernel/pluginhost and kernel/httpapi install a per-test OverrideBuildManifest(FromDir) with a deferred restore (multiple distinct trusted fixture directories share one package's test binary); kernel/supervisor installs its ONE shared fixture's manifest once, unrestored, since every trusted-tier launch in that whole package resolves against the same shared directory."

patterns-established:
  - "A launch-time verification gate (manifest, mirroring the existing pin check) sits immediately after tier resolution and before exec.Command, returning a typed error mirroring the existing failure-class shape (Error/Unwrap/toLaunchFailure) so Discover/Reconcile's existing errors.As soft-failure recording covers it for free."

requirements-completed: [PLUG-07]

coverage:
  - id: D1
    description: "A trusted-directory binary absent from, or not matching, the kernel's link-time build manifest refuses to launch (no subprocess created), including on the add-source picker's describeOnly trial launch"
    requirement: PLUG-07
    verification:
      - kind: unit
        ref: "kernel/pluginhost/manifestgate_test.go#TestLaunch_ManifestGate_AbsentFromManifestRefusesNoSubprocess"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/manifestgate_test.go#TestLaunch_ManifestGate_TamperedBytesAfterManifestBuiltRefuses"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/manifestgate_test.go#TestLaunch_ManifestGate_RefusalAlsoFiresOnDescribeOnlyTrialLaunch"
        status: pass
    human_judgment: false
  - id: D2
    description: "A kernel built with no manifest at all refuses every trusted-tier launch by name (no directory-derived-trust fallback)"
    requirement: PLUG-07
    verification:
      - kind: unit
        ref: "kernel/pluginhost/manifestgate_test.go#TestLaunch_ManifestGate_NoManifestEmbeddedRefusesEveryTrustedLaunch"
        status: pass
    human_judgment: false
  - id: D3
    description: "External-tier launch, pin verification, and the pin-mismatch failure remain byte-for-byte unchanged"
    requirement: PLUG-07
    verification:
      - kind: unit
        ref: "kernel/pluginhost/manifestgate_test.go#TestLaunch_ManifestGate_ExternalTierPinBehaviorUnaffected"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/pin_test.go (full file, unmodified assertions)"
        status: pass
      - kind: unit
        ref: "kernel/supervisor/pinmismatch_test.go (full file, unmodified assertions)"
        status: pass
    human_judgment: false
  - id: D4
    description: "A trusted binary shadowing a same-named external binary launches (when verified) carrying a structured launch_advisory: 'shadowed' on GET /api/sources, never on launch_failure, and never alongside a populated launch_failure on the same entry"
    requirement: PLUG-07
    verification:
      - kind: unit
        ref: "kernel/pluginhost/manifestgate_test.go#TestLaunch_ManifestGate_TrustedShadowingExternalCarriesAdvisory"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/sources_test.go#TestSourcesHandler_ShadowedSourceCarriesLaunchAdvisory"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/sources_test.go#TestSourcesHandler_ManifestUnverifiedEntryCarriesNoLaunchAdvisory"
        status: pass
    human_judgment: false
  - id: D5
    description: "make build, make build-portable, make dev, and make e2e each build plugin binaries before the kernel and link a manifest of exactly the binaries that recipe itself just built"
    requirement: PLUG-07
    verification:
      - kind: integration
        ref: "make build-portable; go tool nm bin/topos | grep -c buildManifest; strings bin/topos | grep -c 'topos-plugin-paperless='"
        status: pass
      - kind: integration
        ref: "make dev-check (scripts/dev-guard-smoke.sh, all 3 cases)"
        status: pass
      - kind: e2e
        ref: "make e2e (122/122 Playwright specs, chromium)"
        status: pass
    human_judgment: false

duration: ~55min
completed: 2026-08-14
status: complete
---

# Phase 13 Plan 05: Link-time build-provenance manifest Summary

**Trusted-tier plugin launch now requires a link-time SHA-256 manifest match (`-ldflags -X`, no generated file, no runtime read), closing the directory-location-is-not-provenance gap Phase 11 left open, plus a structured `launch_advisory` for the D-14 shadow case.**

## Performance

- **Duration:** ~55 min
- **Started:** 2026-08-14 (session start)
- **Completed:** 2026-08-14T18:30:26+01:00
- **Tasks:** 3
- **Files modified:** 22 (4 created, 18 modified)

## Accomplishments

- A link-time trust manifest (`kernel/pluginhost/manifest.go`): `FormatManifest`/`ParseManifest` round-trip a `name=hexdigest` spec, `TrustManifest()` parses the link-time `buildManifest` var exactly once, `VerifyTrustedBinary` re-hashes a trusted-directory binary and checks it against that manifest, and `OverrideBuildManifest(FromDir)` is the test-only seam every affected test installs.
- `cmd/topos-manifest`, a tiny build-time generator: hashes an explicit list of plugin binary paths (never a directory glob) and prints the `FormatManifest` spec to stdout; refuses to run with zero arguments so a mis-wired recipe fails loudly instead of silently producing an empty manifest.
- `Makefile`'s `build`, `build-portable`, `e2e`, and `dev` targets now build their plugin binaries first, generate the manifest from that exact set via `MANIFEST_GEN_PLUGINS`/`_PORTABLE`/`_E2E` (each backed by an explicit, one-place-only binary-path list), then link the kernel with `-ldflags -X pluginhost.buildManifest=<spec>`.
- `launch()`'s new manifest-verification gate (`kernel/pluginhost/host.go`): immediately after `resolveBinaryDetailed` resolves a `TierTrusted` binary, and before `exec.Command` is ever constructed, it calls `VerifyTrustedBinary`. A failure returns `*manifestUnverifiedError` (`LaunchFailureManifestUnverified = "manifest_unverified"`), recorded by `Discover`/`Reconcile` exactly like the existing pin-mismatch soft failure — including on `DescribePluginType`'s `describeOnly` trial launch, unlike the external-tier pin check which deliberately skips it.
- The D-14 shadowing advisory: `discover_binaries.go`'s `ResolveBinary` is now a thin wrapper over `resolveBinaryDetailed`, which additionally reports whether a trusted-tier resolution shadowed a same-named external file. `Plugin.shadowed` carries that fact onto `SourceHealth.LaunchAdvisory` (`LaunchAdvisoryShadowed = "shadowed"`), and `kernel/httpapi/sources.go`'s `sourceStatus.LaunchAdvisory` (new, `omitempty`) transports it onto `GET /api/sources` on a launched entry only — never on `launch_failure`, which keeps its published "never launched at all" meaning (PD-05).
- `docs/api.md` and `docs/plugin-contract.md` updated to document `launch_advisory`, `manifest_unverified`, and the manifest-is-the-trust-authority model, with worked JSON examples for both new cases.

## Task Commits

1. **Task 1: The link-time manifest — parse, verify, generate, and a test seam** - `38178f6` (feat)
2. **Task 2: Reorder the build recipes so the manifest can exist** - `9e520a4` (feat)
3. **Task 3: The launch gate — refuse to load unverified, advise on shadowing** - `c083de9` (feat)
4. **gofmt fixup** - `21c337f` (style) — struct field alignment in `kernel/httpapi/sources.go` after Task 3's new field addition; not a separate task, folded in for cleanliness.

_No TDD RED/GREEN split: this plan's tasks are `type="auto"` (task 2) and `type="auto" tdd="true"` (tasks 1 and 3) but tests and implementation landed together per task, matching every other test/behavior pairing already in this package._

## Files Created/Modified

- `kernel/pluginhost/manifest.go` - link-time manifest parse/verify/generate + test seam
- `kernel/pluginhost/manifest_test.go` - the ten `<behavior>` bullets from Task 1
- `kernel/pluginhost/manifestgate_test.go` - the launch-gate behavior matrix from Task 3, plus its own test helpers (`installTrustedManifest`, `mustHashBinary`, `mutateLastByte`, etc.)
- `cmd/topos-manifest/main.go` - the build-time generator binary
- `Makefile` - `MANIFEST_PLUGIN_BINARIES`/`_PORTABLE`/`MANIFEST_E2E_BINARIES` + `MANIFEST_GEN_*` macros; reordered `build`/`build-portable`/`e2e`/`dev`
- `kernel/pluginhost/host.go` - `LaunchFailureManifestUnverified`, `LaunchAdvisoryShadowed`, `manifestUnverifiedError`, the manifest-gate branch in `launch()`, `Plugin.manifestHash`/`shadowed`, `SourceHealth.LaunchAdvisory`, `launchAdvisoryFor`
- `kernel/pluginhost/discover_binaries.go` - `resolveBinaryDetailed` (shadow-reporting), `ResolveBinary` now a thin wrapper
- `kernel/httpapi/sources.go` - `sourceStatus.LaunchAdvisory`, populated on the probe-derived branch only
- `kernel/httpapi/sources_test.go` - 2 new tests for the shadowed/manifest-refused shapes
- `kernel/pluginhost/tier_test.go` - 2 new tests for `resolveBinaryDetailed`'s shadow flag
- `docs/api.md`, `docs/plugin-contract.md` - documented the new field/reason and the manifest trust model
- `.gitignore` - `/topos-manifest` stray-binary entry
- Test-only manifest-override wiring (Rule 1 fixes, see Deviations): `kernel/pluginhost/{reconcile,describe,describe_whatsapp,extras,pin,stderr}_test.go`, `kernel/httpapi/config_test.go`, `kernel/supervisor/supervisor_test.go`, `cmd/topos/shutdown_signal_test.go`

## Decisions Made

None beyond what PLAN.md already recorded (PD-03/PD-04/PD-05) — implementation followed the plan's `<action>` text directly. The one implementation-level choice not spelled out in the plan: the Makefile's `MANIFEST_GEN_PLUGINS`/`_PORTABLE`/`_E2E` indirection macros (rather than each recipe re-typing `go run ./cmd/topos-manifest <list>` inline) — chosen so the generator invocation and its binary list live in exactly one place each, matching the plan's own "written HERE ONLY" discipline for the binary-list variables and satisfying the acceptance criterion's `grep -c 'topos-manifest $(MANIFEST' Makefile` == 3 exactly (one per macro definition, not one per call site).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Every existing test launching a trusted-tier fixture binary needed a manifest override, beyond the plan's own `<files>` list**

- **Found during:** Task 3, first full-suite run after wiring the manifest gate into `launch()`
- **Issue:** The plan's Task 3 `<files>` list named 8 test files to update with `OverrideBuildManifestFromDir` calls, but the manifest gate — correctly, per its own design — refuses EVERY trusted-tier launch with no manifest installed. Several existing tests outside that list also launch real trusted-tier fixtures and broke: `kernel/pluginhost/pin_test.go` (`TestLaunch_Pin_TrustedTierIgnoresPins`), `kernel/pluginhost/extras_test.go` (`TestLaunch_SourceConfigEnvelopeCarriesExtrasThroughRealLaunch`), and `cmd/topos/shutdown_signal_test.go` (`TestServeReapsPluginSubprocessesOnShutdownSignal`, which builds and runs a REAL kernel subprocess via a bare `go build` with no `-ldflags`).
- **Fix:** Added `installTrustedManifest(t, dir)` (a per-test `OverrideBuildManifestFromDir` + `t.Cleanup(restore)`) to the two `kernel/pluginhost` tests. For `shutdown_signal_test.go`, which builds a real kernel binary as a subprocess fixture, added a matching `-ldflags -X pluginhost.buildManifest=<spec>` computed from the built mock plugin's own hash (`pluginhost.ManifestEntriesForBinaries` + `pluginhost.FormatManifest`) — mirroring the Makefile's own build recipes rather than inventing a second mechanism.
- **Files modified:** `kernel/pluginhost/pin_test.go`, `kernel/pluginhost/extras_test.go`, `cmd/topos/shutdown_signal_test.go`
- **Verification:** `go test ./... -count=1` passes from a completely clean state (previously 4 packages failed: `TestLaunch_Pin_TrustedTierIgnoresPins`, `TestLaunch_SourceConfigEnvelopeCarriesExtrasThroughRealLaunch`, and all four rows of `TestServeReapsPluginSubprocessesOnShutdownSignal`).
- **Committed in:** `c083de9` (Task 3 commit)

**2. [Rule 1 - Bug] `kernel/supervisor` package needed its trusted fixture's manifest installed centrally, not per-test**

- **Found during:** Task 3, same full-suite run
- **Issue:** `kernel/supervisor`'s test files share ONE trusted-directory fixture builder (`buildMockPluginDir` in `supervisor_test.go`, `sync.Once`-cached) across 17+ test functions in 6 files. Adding a per-test `OverrideBuildManifestFromDir` + restore to every call site (as the plan's literal instruction describes) would work but is high-churn; more importantly, other fixture builders in that package (`buildSecondMockPluginDir`, `buildRenamedMockPluginDir`, `buildExternalDemoPluginDir`) are all used as the EXTERNAL tier only, so there is exactly one trusted directory the whole package ever needs verified.
- **Fix:** Installed the manifest override once, inside `buildMockPluginDir`'s own `sync.Once` block, never restored — sound specifically because no other trusted directory exists in this package to conflict with a wholesale-replace override.
- **Files modified:** `kernel/supervisor/supervisor_test.go`
- **Verification:** `go test ./kernel/supervisor/... -count=1` passes (was 15 failing tests across `supervisor_test.go`, `readiness_test.go`, `suspend_test.go`, `launchlatency_test.go`, `pinmismatch_test.go`, `externaltier_test.go`, `externalproof_test.go`).
- **Committed in:** `c083de9` (Task 3 commit)

**3. [Rule 1 - Bug] gofmt struct-field alignment after adding `LaunchAdvisory`**

- **Found during:** Post-commit formatting sweep
- **Issue:** Adding `LaunchAdvisory string` to `sourceStatus` shifted the column alignment `gofmt` expects for the sibling fields below it (`Reachable`, `Syncing`, etc.), and the commit landed with stale alignment.
- **Fix:** `gofmt -w kernel/httpapi/sources.go`.
- **Files modified:** `kernel/httpapi/sources.go`
- **Verification:** `gofmt -l` reports clean; `go test ./kernel/httpapi/... -count=1` still passes.
- **Committed in:** `21c337f` (separate style commit, since it landed after Task 3's commit was already made)

---

**Total deviations:** 3 auto-fixed (all Rule 1 — bug/correctness fixes required to keep the plan's own acceptance criterion "every pre-existing test still asserting what it asserted before" true).
**Impact on plan:** No scope creep — every deviation is test-fixture plumbing made necessary by the manifest gate's own correct, intended behavior (refuse an unverified trusted launch unconditionally). No production code beyond what the plan specified was touched.

## Issues Encountered

None beyond the deviations above. `go tool nm bin/topos | grep -c buildManifest` and `strings bin/topos | grep -c 'topos-plugin-paperless='` both reported `0` after Task 2 alone (before Task 3 wired `VerifyTrustedBinary` into `launch()`) — the Go linker dead-code-eliminates `manifest.go`'s functions when nothing in `cmd/topos`'s reachable call graph references them. This resolved itself once Task 3 landed and re-verified `make build-portable` afterward (both now report `≥1`, matching the acceptance criteria).

## User Setup Required

None — no external service configuration required. Existing build recipes (`make build`, `make build-portable`, `make dev`, `make e2e`) are unchanged from an operator's point of view; they simply now also link a real trust manifest.

## Verification Run Log

- `go test ./kernel/pluginhost/... -run Manifest -count=1` — pass (Task 1)
- `go run ./cmd/topos-manifest` (no args) — exits 1, names the problem
- `go run ./cmd/topos-manifest bin/plugins/topos-plugin-mock` — prints exactly one `name=digest` pair matching `sha256sum`
- `grep -l 'crypto/sha256' kernel/pluginhost/*.go` — exactly `binaryhash.go` (+ its own test file)
- `make build-portable && ./bin/topos serve --help >/dev/null 2>&1; make dev-check && make docs-check` — dev-check and docs-check both pass
- `grep -c 'topos-manifest \$(MANIFEST' Makefile` — 3
- `make e2e` — 122/122 Playwright specs pass
- `go test ./kernel/... ./internal/... -count=1` — pass
- `go test ./... -count=1` (full repo, all workspace modules) — pass, twice (before and after the gofmt fixup)
- `make test-portable` — pass (all 12 modules)
- `git diff proto/ sdk/` — empty (plugin contract untouched)
- `go build ./...` on a clean tree, `git status --porcelain` clean afterward

## Next Phase Readiness

Task 3's own `<done>` criterion is met: an unverifiable trusted-directory binary refuses to launch by name, a shadowing collision is a structured advisory rather than only a log line, and every Phase 11 external-tier behavior is unchanged. `13-06` (per the phase's plan sequencing) can build the UI rendering for `launch_advisory` — the wire contract (`docs/api.md`) and the underlying kernel fact are both in place and tested. No blockers.
