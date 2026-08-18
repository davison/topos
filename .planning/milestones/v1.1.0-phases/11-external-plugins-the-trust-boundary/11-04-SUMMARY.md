---
phase: 11-external-plugins-the-trust-boundary
plan: 04
subsystem: plugin-host
tags: [go-plugin, go-workspace, testdata-fixture, trust-boundary, integration-test]

requires:
  - phase: 11-01
    provides: "kernel/pluginhost.Tier/Dirs/TieredBinary/ResolveBinary/DiscoverTiered/DiscoverAllTiered — the two-tier plugin discovery/launch-time provenance authority"
  - phase: 11-02
    provides: "kernel/pluginhost.HashBinary, ErrPinMismatch/LaunchFailure/Host.LaunchFailures, allowedEnv/sourceConfigEnvelope, kernel/config.EnvRefNames, topos.v1.ExtrasField/DescribeResponse.extras — pin verification, env allowlist, and wire extras, all previously proven only against plugins/mock rebuilt under a different name"
provides:
  - "testdata/external-plugin/ — a genuinely separate Go module (example.com/acme/topos-plugin-external-demo, outside github.com/davison/topos) written entirely from the published contract, standing in for a third party's own out-of-repo plugin build"
  - "make external-demo — builds the fixture CGO_ENABLED=0 into its own bin/plugins-external/ directory, never bin/plugins/; make e2e now depends on it"
  - "kernel/supervisor/externalproof_test.go#TestExternalProof_OutOfRepoBinaryEndToEnd — the standing, mechanical gate for ROADMAP success criterion 5"
affects: []

actuals:
  tokens: 14800
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "A go.work member module can live under testdata/ deliberately: the go tool never matches \"testdata\" in a \"./...\" pattern, and this repo's own AST audits (internal/audit) skip any directory named \"testdata\" everywhere they walk — so a fixture module gains local, replace-directive-free sdk resolution without ever joining the in-repo plugin set's build or audit surface"
    - "A pin-mismatched instance still gets a normal, hard-fail-free reboot ONLY when no webspace's participation (keywords fallback or explicit match block) still names it — pluginhost.ValidateMatchConfig has no ValidateMatchConfigWithSuspended-style exemption for a pin mismatch, so a test (or an operator) proving a tampered binary's refusal must first retire that instance's webspace membership, exactly like kernel/supervisor/pinmismatch_test.go's own established workaround"
    - "Replacing a binary a just-killed subprocess may still be exiting: write the new bytes to a sibling temp file and os.Rename over the target path, never open-for-write in place — Kill() returning is not a guarantee the kernel has reaped the process yet, and an in-place write can race a held ETXTBSY on that exact inode where rename() never does"

key-files:
  created:
    - testdata/external-plugin/go.mod
    - testdata/external-plugin/go.sum
    - testdata/external-plugin/main.go
    - testdata/external-plugin/plugin.go
    - testdata/external-plugin/README.md
    - kernel/supervisor/externalproof_test.go
  modified:
    - go.work
    - go.work.sum
    - Makefile
    - docs/testing.md

key-decisions:
  - "testdata/external-plugin/go.mod's require block is pinned to the SAME transitive dependency versions already used elsewhere in the workspace (grpc v1.82.1, x/net v0.53.0, etc. — copied from sdk/go.mod) rather than left at whatever `go mod tidy` picked on its own (grpc v1.83.0 initially) — go.work's MVS resolves the EFFECTIVE version for the whole workspace as the max across every member's go.mod, so an unpinned fixture module could have silently bumped the shared dependency graph for the entire repo as a side effect of adding a testdata fixture. Verified empirically (`go list -m` before/after) that the final pins reproduce the exact pre-existing effective versions."
  - "The proof plugin's item set is derived from its OWN launch inputs (extras map, os.Environ()) rather than a fixed in-memory corpus like plugins/mock — every claim this phase makes (extras passthrough, environment scrubbing) is observable in the synced index, not asserted by the test's own mock, per the plan's own must_haves."
  - "The end-to-end test reboots a SECOND supervisor (fresh Discover) rather than driving Apply to observe the tampered-binary refusal — Reconcile only re-launches an instance whose config.Source value actually changed, so an unchanged config's already-running subprocess would never be re-hashed by Apply at all; only a full reboot re-verifies every external-tier binary's pin."
  - "Before that second boot, the test retires the demo instance's only participating webspace (cfgStore.Save, deleting the webspace) rather than modifying kernel/pluginhost/matchconfig.go to add a pin-mismatch exemption — the latter is out of this plan's files_modified scope and is an already-documented, deliberately deferred gap (11-02-SUMMARY.md's Issues Encountered); this test's own workaround exactly mirrors kernel/supervisor/pinmismatch_test.go's identical, pre-existing pattern."

requirements-completed: [PLUG-06, PLUG-09]

coverage:
  - id: D1
    description: "A plugin binary built from a module outside the in-repo plugin set (its own go.mod, its own module path, its own build target, its own output directory) is discovered from the external directory, launched under a content-hash pin, and syncs items end to end (ROADMAP success criterion 5)"
    requirement: "PLUG-06"
    verification:
      - kind: integration
        ref: "kernel/supervisor/externalproof_test.go#TestExternalProof_OutOfRepoBinaryEndToEnd (real supervisor boot, real out-of-repo subprocess launch)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The proof binary receives provider-specific extras keys the kernel has never heard of and emits them back as items, with a ${VAR}-referenced value already expanded — the passthrough is observable, not asserted"
    requirement: "PLUG-09"
    verification:
      - kind: integration
        ref: "kernel/supervisor/externalproof_test.go#TestExternalProof_OutOfRepoBinaryEndToEnd (extras workspace_id/referenced_var item-title assertions)"
        status: pass
    human_judgment: false
  - id: D3
    description: "The proof binary emits the names of every environment variable it can actually see: PATH (allowlisted) and a ${VAR}-referenced variable are visible; a variable set on the kernel process but referenced nowhere in the instance's config never reaches it"
    verification:
      - kind: integration
        ref: "kernel/supervisor/externalproof_test.go#TestExternalProof_OutOfRepoBinaryEndToEnd (env/PATH, env/TOPOS_PROOF_REFERENCED present; env/TOPOS_PROOF_UNREFERENCED absent)"
        status: pass
    human_judgment: false
  - id: D4
    description: "A tampered copy of the proof binary is refused at the next launch by name, and a healthy source in the same config still boots and stays reachable (PLUG-07 proven on a genuinely external binary, not only on a fixture)"
    verification:
      - kind: integration
        ref: "kernel/supervisor/externalproof_test.go#TestExternalProof_OutOfRepoBinaryEndToEnd (one-byte rewrite + reboot; LaunchFailures/Plugins/ProbeSources assertions)"
        status: pass
    human_judgment: false
  - id: D5
    description: "The proof binary lives under testdata/, never enters the in-repo plugin set's audit scope, and make test-portable/CGO_ENABLED=0 go build ./... from the repo root are unaffected by the new module"
    verification:
      - kind: unit
        ref: "go test ./internal/audit/... (unchanged pass, scanned-file count unaffected)"
        status: pass
      - kind: other
        ref: "CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go test ./... from repo root"
        status: pass
    human_judgment: false

duration: ~9min (commit span); active engineering time materially longer, dominated by an unplanned go.work/MVS dependency-version investigation
completed: 2026-08-13
status: complete
---

# Phase 11 Plan 04: Out-of-Repo Proof Plugin & the Criterion-5 Gate Summary

**A genuinely separate Go module (`example.com/acme/topos-plugin-external-demo`, outside `github.com/davison/topos`) written from nothing but the published plugin contract, built by its own `make external-demo` target into `bin/plugins-external/`, and gated end to end by a real supervisor boot (`TestExternalProof_OutOfRepoBinaryEndToEnd`) that proves discovery, tier, extras passthrough, environment scrubbing, and pin refusal against the plugin's own observable output — not against a rebuilt copy of `plugins/mock`.**

## Performance

- **Duration:** ~9min by git commit span (c8f13b4 → e230b5b); active engineering time was materially longer — most of it spent diagnosing and pinning a `go.work`-wide MVS dependency-version side effect (see Deviations)
- **Started:** 2026-08-13T10:19:09+01:00
- **Completed:** 2026-08-13T10:27:36+01:00
- **Tasks:** 3/3
- **Files modified:** 10 (6 created, 4 modified)

## Accomplishments

- `testdata/external-plugin/` is a real, standalone Go module — module path `example.com/acme/topos-plugin-external-demo`, deliberately outside `github.com/davison/topos` — built entirely from `docs/plugin-contract.md`, `proto/topos/v1/plugin.proto`, and the `sdk` module, standing in for a genuine third party's own separate build (never a copy of an in-repo plugin's package)
- Its `main.go` fails startup loudly, by name, non-zero, when its one required `path` key is empty, mirroring `plugins/mockstrict`/`plugins/signal`'s exact pre-Serve fatal-guard shape
- Its `plugin.go` `Describe` declares two `ExtrasField` entries (one required non-secret, one optional secret) and its `Match` reports back exactly what it was launched with: one item per configured extras key and one item per environment variable actually visible to the subprocess — variable **names only**, never values — plus one fixed anchor item
- `go.work` gained the module as a workspace member (comment explaining why a fixture module belongs there and why `testdata/` specifically); its `go.mod`'s indirect requires were pinned to match the workspace's pre-existing effective dependency versions rather than left at whatever `go mod tidy` picked unassisted
- `make external-demo` builds the fixture `CGO_ENABLED=0` into its own `bin/plugins-external/topos-plugin-external-demo` — never `bin/plugins/`, never built by `build`/`plugins`/`plugins-portable`; `make e2e` now depends on it for plans 11-05/11-06's browser harness
- `docs/testing.md` documents the new fixture alongside the two mock-shaped plugins, with a dated "What changed" entry
- `kernel/supervisor/externalproof_test.go#TestExternalProof_OutOfRepoBinaryEndToEnd` is the standing, mechanical gate for ROADMAP success criterion 5: a real supervisor boot against the built binary proves tier `external`, pin-gated launch, extras passthrough (including live `${VAR}` expansion), environment scrubbing, and — after a one-byte rewrite and reboot — pin refusal by name with a healthy control source unaffected

## Task Commits

Each task was committed atomically:

1. **Task 1: A standalone, out-of-repo-shaped source plugin written against the published contract** - `c8f13b4` (feat)
2. **Task 2: Build wiring — workspace membership, its own make target, its own output directory** - `3b62818` (feat)
3. **Task 3: The end-to-end proof — discovered, untrusted, pinned, extras delivered, environment scrubbed** - `e230b5b` (test)

_All three tasks were declared `type="auto"` (Task 3 also `tdd="true"`); consistent with this repo's established convention for `type="auto"` tasks with a declared `<behavior>` spec (see plan 11-02's own precedent), Task 3's test and its target behavior are proven together in a single commit — the underlying mechanism (pin verification, extras wire path, env allowlist) was already built and tested in isolation by plans 11-01/11-02; this task's own job is a genuine integration proof over a real out-of-repo binary, not new production code._

## Files Created/Modified

**Task 1 (testdata/external-plugin/):**
- `go.mod` - module `example.com/acme/topos-plugin-external-demo`, go 1.25.0
- `go.sum` - resolved via `go mod tidy` against the workspace's own module graph
- `main.go` - `WEBSPACES_SOURCE_CONFIG` decode, pre-Serve fatal guard on `path`, `goplugin.Serve`
- `plugin.go` - `Describe`/`Match`/`Fetch`/`Health`; `Match` derives its item set from the instance's own extras map and `os.Environ()`
- `README.md` - what this module is, why it lives under `testdata/`, how it is built and consumed, and that it must never reach a real trusted plugin directory

**Task 2 (build wiring):**
- `go.work` - `./testdata/external-plugin` added to the `use` block with a comment
- `go.work.sum` - regenerated
- `Makefile` - new `external-demo` target (`.PHONY`-registered); `e2e` target now depends on it
- `docs/testing.md` - new `topos-plugin-external-demo` subsection alongside "The two mock-shaped plugins"; dated "What changed" entry

**Task 3 (the proof):**
- `kernel/supervisor/externalproof_test.go` - new; `TestExternalProof_OutOfRepoBinaryEndToEnd` plus `buildExternalDemoPluginDir`/`copyExecutableFile`/`assertItemTitle`/`assertItemPresent`/`assertItemAbsent` helpers

## Decisions Made

- **Pinned `testdata/external-plugin/go.mod`'s indirect requires to the workspace's existing effective versions.** `go mod tidy` run unassisted picked `grpc v1.83.0` and newer `golang.org/x/{net,sys,text}` versions than any other workspace module declared — since `go.work`'s MVS resolves the EFFECTIVE version across the WHOLE workspace as the max of every member's own `go.mod`, this would have silently bumped the shared dependency graph for the entire repo as a side effect of adding a testdata fixture. Verified empirically with `go list -m` before/after adding the module to `go.work`: with the pinned versions, every workspace-wide effective version is byte-identical to before this plan.
- **Rebooted a second supervisor rather than driving `Apply`** to observe the tampered-binary refusal — `Reconcile` only re-launches an instance whose `config.Source` value itself changed; since nothing about `demo`'s config changes across the tamper, `Apply` would never re-hash its already-running subprocess at all. Only a full `Discover` (a fresh boot) re-verifies every external-tier binary's pin.
- **Retired the `demo` instance's only participating webspace before the second boot**, via `cfgStore.Save` deleting the `[webspaces.proof]` table, rather than touching `kernel/pluginhost/matchconfig.go` — `pluginhost.ValidateMatchConfig` has no `ValidateMatchConfigWithSuspended`-style exemption for a pin-mismatched instance (a documented, deliberately out-of-scope gap per 11-02-SUMMARY.md's "Issues Encountered"), and a fix there is outside this plan's `files_modified`. This test's own workaround exactly mirrors `kernel/supervisor/pinmismatch_test.go`'s pre-existing, identical pattern.
- **Wrote the tampered binary via a sibling temp file + `os.Rename`, not an in-place `os.WriteFile`** — discovered live: an in-place write immediately after `sup.Shutdown()` intermittently failed with `text file busy` when the full test package ran (never when this test ran alone), because `Kill()` returning is not a guarantee the kernel has finished reaping the just-exited subprocess. `rename()` is never blocked by that race; an in-place open-for-write can be.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `go mod tidy`'s auto-picked dependency versions would have silently bumped the whole workspace's effective dependency graph**
- **Found during:** Task 1, verifying `go build ./...` from the repo root after adding the fixture module to `go.work`
- **Issue:** An unconstrained `go mod tidy` inside `testdata/external-plugin` resolved `google.golang.org/grpc` to `v1.83.0` (the repo's existing pin, everywhere else, is `v1.82.1`) and newer `golang.org/x/{net,sys,text}` versions than any other workspace module declared. Since Go workspace mode computes each dependency's EFFECTIVE version as the max across every member module's own `go.mod` (MVS), this would have silently upgraded the shared dependency graph for the ENTIRE repository as an unintended side effect of adding a testdata fixture — confirmed via `go list -m google.golang.org/grpc` before/after, which showed the bump.
- **Fix:** Manually pinned the affected `require` entries in `testdata/external-plugin/go.mod` to the versions already declared in `sdk/go.mod` (the module with the identical dependency footprint), then re-ran `go mod tidy`, which settled without complaint. Verified with `go list -m` for every affected module that the workspace-wide effective version is now byte-identical to what it was before this module existed in `go.work`.
- **Files modified:** `testdata/external-plugin/go.mod`, `testdata/external-plugin/go.sum`
- **Verification:** `go list -m google.golang.org/grpc golang.org/x/net golang.org/x/sys golang.org/x/text` matched the pre-existing baseline (captured by temporarily reverting `go.work` and re-checking) after the pin.
- **Committed in:** `c8f13b4` (Task 1 commit)

**2. [Rule 1 - Bug] `pluginhost.ValidateMatchConfig` rejects the whole boot when a pin-mismatched instance's webspace participation survives unchanged**
- **Found during:** Task 3, writing the tampered-binary reboot assertion
- **Issue:** A straightforward "reboot against the exact same config" attempt failed with `config: webspace "proof" relies on the keywords fallback for source "demo", which has no launched plugin` — a pre-existing, already-documented gap (11-02-SUMMARY.md's "Issues Encountered"): a pin-mismatched instance gets no `ValidateMatchConfigWithSuspended`-style exemption, so ANY webspace still naming it (via keywords fallback or an explicit match block) fails validation outright, unrelated to the pin check this test exists to prove.
- **Fix:** Split webspace participation into `[webspaces.control]` (`mockctrl` only) and `[webspaces.proof]` (`demo` only), and — before the second boot — `cfgStore.Save` a config with `[webspaces.proof]` deleted, retiring `demo`'s only participation while it remains fully configured under `[sources.demo]` (still launch-attempted, still recorded in `LaunchFailures()`). Mirrors `kernel/supervisor/pinmismatch_test.go`'s own pre-existing workaround for the identical gap.
- **Files modified:** `kernel/supervisor/externalproof_test.go` (test-internal only; no production code touched — a fix in `kernel/pluginhost/matchconfig.go` is outside this plan's `files_modified` and remains the documented, deliberately deferred gap)
- **Verification:** `go test ./kernel/supervisor/ -run TestExternalProof -v` passes; the reboot's own assertions (`LaunchFailures`, `Host().Plugins()`, `ProbeSources`) confirm the intended refusal-with-healthy-control shape.
- **Committed in:** `e230b5b` (Task 3 commit)

**3. [Rule 1 - Bug] Intermittent `text file busy` writing the tampered binary in place**
- **Found during:** Task 3, running the full `kernel/supervisor` package (not just this test in isolation)
- **Issue:** `os.WriteFile(binPath, tampered, 0o755)` immediately after `sup.Shutdown()` failed with `open ...: text file busy` when the whole package ran, but never when this test ran alone — `Kill()` returning does not guarantee the OS has finished reaping the just-exited subprocess, and an in-place open-for-write can race a still-held `ETXTBSY` on that exact inode.
- **Fix:** Write the tampered bytes to a sibling `<binPath>.tampered` file, then `os.Rename` it over `binPath` — `rename()` is never blocked by `ETXTBSY`, the standard technique for safely replacing a binary that may have just been running.
- **Files modified:** `kernel/supervisor/externalproof_test.go`
- **Verification:** `go test ./kernel/supervisor/... -count=1` (full package) and `go test ./kernel/supervisor/ -run TestExternalProof -count=5 -race` both pass repeatably.
- **Committed in:** `e230b5b` (Task 3 commit)

---

**Total deviations:** 3 auto-fixed (all Rule 1 — bugs/gaps this plan's own work surfaced live, none scope creep beyond making the plan's stated `<behavior>` genuinely observable)
**Impact on plan:** All three were necessary to make this plan's own must_haves true rather than merely appear true — an unpinned dependency bump would have silently widened the whole repo's build graph, an unaddressed validation gap would have made the tamper-refusal assertion unreachable, and the file-busy race would have made the gate itself intermittently flaky under full-package runs (exactly the kind of false-negative CI can't afford). No file outside this plan's stated intent was touched — the pin-mismatch/webspace-validation gap remains open in production code, exactly as 11-02-SUMMARY.md already documented it.

## Issues Encountered

- **`go.work`'s MVS resolves per-dependency effective versions across the WHOLE workspace, not per-module** — worth calling out explicitly for any future plan adding a new workspace member: an unconstrained `go mod tidy` inside a new module can silently change what every OTHER module in the workspace actually builds against, with no signal beyond a diff in that new module's own `go.mod`. This plan's own fix (pin to the existing baseline, verify with `go list -m` before/after) is the general recipe.
- **The pin-mismatch/webspace-validation interaction gap** (`pluginhost.ValidateMatchConfig` has no exemption for a pin-mismatched instance, unlike a suspended one) is now exercised by TWO tests (`pinmismatch_test.go`'s pre-existing one and this plan's new one) but remains unresolved in production code — both tests work around it rather than fix it, consistent with 11-02-SUMMARY.md's explicit statement that resolving it is out of scope unless a later UAT round surfaces it as user-visible.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- ROADMAP success criterion 5 now has a standing, automated gate (`TestExternalProof_OutOfRepoBinaryEndToEnd`) — everything downstream in v1.1.0 (Phase 12's rehearsal, Phase 14's real out-of-repo Google Drive source) can now build on a PROVEN mechanism rather than an assumed one.
- `bin/plugins-external/topos-plugin-external-demo` and `testdata/external-plugin/` are available for plans 11-05/11-06's browser harness to link into a fixture's external directory, exactly as `make e2e`'s new dependency on `external-demo` anticipates — no further build-wiring changes should be needed there.
- The pin-mismatch/webspace-validation gap (Issues Encountered, above) remains open; if a later UAT round or a real operator's own config hits it, the fix belongs in `kernel/pluginhost/matchconfig.go` (a `ValidateMatchConfigWithSuspended`-shaped exemption for pin-mismatched instances), not in either test file that currently works around it.
- Full verification suite green: `make external-demo && CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go test ./...` (whole repo), `go test ./internal/audit/...`, `go test ./kernel/supervisor/ -run TestExternalProof -v` (also re-verified with `-count=5 -race`), `make docs-check`.

---
*Phase: 11-external-plugins-the-trust-boundary*
*Completed: 2026-08-13*

## Self-Check: PASSED

- Verified files exist: `testdata/external-plugin/go.mod`, `testdata/external-plugin/go.sum`, `testdata/external-plugin/main.go`, `testdata/external-plugin/plugin.go`, `testdata/external-plugin/README.md`, `kernel/supervisor/externalproof_test.go`, `Makefile`, `docs/testing.md`, `go.work`, `go.work.sum`
- Verified commits exist in `git log --oneline --all`: `c8f13b4`, `3b62818`, `e230b5b`
