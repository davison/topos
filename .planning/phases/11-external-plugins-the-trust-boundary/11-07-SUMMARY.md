---
phase: 11-external-plugins-the-trust-boundary
plan: 07
subsystem: security
tags: [go, path-traversal, config-validation, plugin-host, gap-closure]

# Dependency graph
requires:
  - phase: 11-external-plugins-the-trust-boundary
    provides: "the two-tier trusted/external plugin directory mechanism (11-01 through 11-06) whose trust premise this plan restores to truth"
provides:
  - "pluginhost.ResolveBinary confinement + regular-file guard — a caller-supplied plugin name can no longer resolve outside dirs.Trusted/dirs.External or to a non-regular file"
  - "config.Validate rejection of an absent or non-bare Source.Plugin value, naming the offending source"
  - "twelve new regression tests proving the confinement contract across kernel/pluginhost, kernel/config, and kernel/httpapi"
affects: [pluginhost, config, httpapi, plugin-contract-docs]

# Actuals (#2632)
actuals:
  tokens: 7150
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Hand-duplicated validation rule across an import-cycle boundary (config cannot import pluginhost): pinKeyPluginPrefix's existing precedent extended to the bare-filename plugin-name rule, cross-referenced by doc comment on both sides"

key-files:
  created: []
  modified:
    - kernel/pluginhost/discover_binaries.go
    - kernel/pluginhost/tier_test.go
    - kernel/config/config.go
    - kernel/config/config_test.go
    - kernel/httpapi/config_test.go
    - docs/plugin-contract.md

key-decisions:
  - "No architectural changes — this is a gap-closure plan restoring an already-decided premise (trust decided by directory provenance), not introducing new identity, tier, or configurability (assumption_delta_decision: no-change)."
  - "Did not require the topos-plugin- prefix in either new validator — prefix filtering is DiscoverAllBinaries' catalog policy (D-10), not a resolution-shape or config-validity rule; enforcing it here would have broken existing short-name fixtures and split ownership of a rule that already has a home."

patterns-established:
  - "Confinement-then-regular-file-check as ResolveBinary's opening statement, before any directory is consulted — the same discipline any future path-join-from-caller-input function in this codebase should follow."

requirements-completed: [PLUG-07]

coverage:
  - id: D1
    description: "pluginhost.ResolveBinary rejects any non-bare plugin binary name (traversal, absolute path, Windows separator, '.', '..', empty) as its first statement, before either configured directory is consulted, and never returns TierTrusted for such a name."
    requirement: "PLUG-07"
    verification:
      - kind: unit
        ref: "kernel/pluginhost/tier_test.go#TestResolveBinary_TraversalNameIsRejectedBeforeAnyDirectoryIsConsulted"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/tier_test.go#TestResolveBinary_AbsolutePathNameIsRejected"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/tier_test.go#TestResolveBinary_WindowsSeparatorNameIsRejected"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/tier_test.go#TestResolveBinary_EmptyNameIsRejectedNotResolvedToTheDirectoryItself"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/tier_test.go#TestResolveBinary_DotAndDotDotNamesAreRejected"
        status: pass
    human_judgment: false
  - id: D2
    description: "ResolveBinary requires info.Mode().IsRegular() at all three os.Stat sites, so a directory sharing a binary's name is never resolved, while a symlinked regular file (the e2e harness's fixture shape) still resolves unchanged."
    requirement: "PLUG-07"
    verification:
      - kind: unit
        ref: "kernel/pluginhost/tier_test.go#TestResolveBinary_DirectoryNamedLikeABinaryIsNotResolved"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/tier_test.go#TestResolveBinary_SymlinkedBinaryStillResolvesAfterRegularFileCheck"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/11-external-tier-badge.spec.ts (make e2e E2E_ARGS=specs/11-external-tier-badge.spec.ts, 3 passed)"
        status: pass
    human_judgment: false
  - id: D3
    description: "config.Validate rejects an absent, whitespace-only, or non-bare Source.Plugin value at config load, naming the offending source and (for an unset ${VAR}) the missing variable, with deterministic multi-error reporting."
    requirement: "PLUG-07"
    verification:
      - kind: unit
        ref: "kernel/config/config_test.go#TestValidate_SourcePluginTraversalIsRejectedNamingTheSource"
        status: pass
      - kind: unit
        ref: "kernel/config/config_test.go#TestValidate_SourcePluginEmptyIsRejectedNamingTheSource"
        status: pass
      - kind: unit
        ref: "kernel/config/config_test.go#TestValidate_SourcePluginErrorIsDeterministicAcrossRuns"
        status: pass
    human_judgment: false
  - id: D4
    description: "PUT /api/config rejects a traversal or empty Source.Plugin value with 422 config_invalid, naming the source, leaving config.toml byte-identical to before the request."
    requirement: "PLUG-07"
    verification:
      - kind: integration
        ref: "kernel/httpapi/config_test.go#TestConfigSaveHandler_TraversalPluginValueReturns422AndLeavesFileUnchanged"
        status: pass
      - kind: integration
        ref: "kernel/httpapi/config_test.go#TestConfigSaveHandler_EmptyPluginValueReturns422NamingTheSource"
        status: pass
    human_judgment: false
  - id: D5
    description: "The newly-enforced bare-filename rule for [sources.<id>] plugin is published in docs/plugin-contract.md's Trust tiers section."
    verification:
      - kind: other
        ref: "grep -n 'bare binary filename' docs/plugin-contract.md (1 match) && make docs-check (exit 0)"
        status: pass
    human_judgment: false

duration: ~40min
completed: 2026-08-13
status: complete
---

# Phase 11 Plan 07: Confine Plugin Binary Resolution Against Path Traversal Summary

**Closes CR-01 (11-REVIEW.md, Critical): `pluginhost.ResolveBinary` now rejects any non-bare plugin binary name before consulting either configured directory, and `config.Validate` rejects the identical shapes at config load and `PUT /api/config` — restoring ROADMAP success criterion 3's premise that trust is decided purely by which directory a binary's bytes live in.**

## Performance

- **Duration:** ~40min
- **Started:** 2026-08-13T15:20:00Z (approx.)
- **Completed:** 2026-08-13T15:44:00Z
- **Tasks:** 3 completed
- **Files modified:** 6

## Accomplishments

- `pluginhost.ResolveBinary` now validates the caller-supplied plugin name (`validatePluginBinaryName`) as its very first statement — rejecting empty, path-separator-containing, `.`/`..`, and any name that doesn't equal `filepath.Base(name)` — before either `dirs.Trusted` or `dirs.External` is touched. No traversal, absolute path, or empty name can resolve outside the two configured directories or be classified `TierTrusted`.
- All three `os.Stat` sites inside `ResolveBinary` now require `info.Mode().IsRegular()`, closing the directory-shadowing gap (including the empty-name `filepath.Join(dir, "") == dir` case) while preserving symlink-following behavior the browser e2e harness's fixture depends on.
- `config.Validate` gained `validateSourcePlugins`, the hand-kept config-side twin of the same rule (duplicated rather than imported, since `config` must never import `pluginhost`), rejecting an absent, whitespace-only, or non-bare `Source.Plugin` at config load and `PUT /api/config`, naming the offending source deterministically.
- Twelve new regression tests (seven in `kernel/pluginhost`, three in `kernel/config`, two in `kernel/httpapi`) prove the confinement contract end to end, from `ResolveBinary`'s launch-time authority through `config.Validate`'s load-time gate to the `PUT /api/config` HTTP boundary.
- The newly-enforced bare-filename rule is published in `docs/plugin-contract.md`'s Trust tiers section.

## Task Commits

Each task was committed atomically:

1. **Task 1: Confine and regular-file-gate ResolveBinary, the launch-time trust authority** - `910e6d7` (fix)
2. **Task 2: Reject a non-bare or absent Source.Plugin inside config.Validate, naming the source** - `efbfc4f` (fix)
3. **Task 3: Pin the HTTP-boundary behaviour and run the full portable gate** - `a793b46` (test)

_No plan-metadata commit is included here — the orchestrator commits STATE.md/ROADMAP.md updates centrally after all wave agents complete (worktree mode)._

## Files Created/Modified

- `kernel/pluginhost/discover_binaries.go` - Added `validatePluginBinaryName`; `ResolveBinary` now calls it first and requires `IsRegular()` at all three `os.Stat` sites
- `kernel/pluginhost/tier_test.go` - Added seven `TestResolveBinary_*` adversarial regression tests
- `kernel/config/config.go` - Added `validateSourcePlugins`, called from `Validate` before `validatePins`; added `path/filepath` import
- `kernel/config/config_test.go` - Added three `TestValidate_SourcePlugin*` test functions
- `kernel/httpapi/config_test.go` - Added two `TestConfigSaveHandler_*` HTTP-boundary tests
- `docs/plugin-contract.md` - Added the bare-filename rule paragraph to the Trust tiers section

## Decisions Made

- No architectural changes — restores an already-decided premise (two-tier trust-by-directory-provenance) rather than introducing new identity, tier, or configurability (see plan's `assumption_delta_decision`).
- Deliberately did not require the `topos-plugin-` prefix in either new validator — that's `DiscoverAllBinaries`' catalog policy (D-10), a separately-owned rule, and several existing tests/fixtures use short names.
- Set `Sync.Interval: "15m"` in the two new `kernel/httpapi/config_test.go` fixtures (not specified verbatim in the plan's action text) so the locally-called `invalid.Validate(nil)` used for the message-verbatim assertion matches what `dryRunExpand`'s `applyDefaults`-then-`Validate` path actually validates during the real `PUT` — otherwise the local call would hit the unrelated `[sync] interval` check first and the "verbatim" assertion would compare against the wrong error text.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Test fixture mismatch between local `Validate(nil)` call and the actual `PUT` validation path**

- **Found during:** Task 3, writing `TestConfigSaveHandler_TraversalPluginValueReturns422AndLeavesFileUnchanged` and `TestConfigSaveHandler_EmptyPluginValueReturns422NamingTheSource`
- **Issue:** The initial fixture omitted `Sync.Interval`, so the local `invalid.Validate(nil)` call (used to compute the expected verbatim error message) failed on the unrelated `[sync] interval` check before ever reaching the plugin-shape check — while the real `PUT` request goes through `dryRunExpand`, which runs `applyDefaults` (defaulting `Sync.Interval`) before `Validate`, so the actual response's error text differed from the test's expectation.
- **Fix:** Added `Sync: config.SyncConfig{Interval: "15m"}` to both fixtures, matching the value `applyDefaults` would produce anyway, so the local pre-computed `wantErr` and the real HTTP response agree.
- **Files modified:** `kernel/httpapi/config_test.go`
- **Verification:** `go test ./kernel/httpapi/... -run TestConfigSaveHandler -v` — all 7 cases (5 pre-existing + 2 new) PASS
- **Committed in:** `a793b46` (part of Task 3 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1)
**Impact on plan:** Test-only fix, necessary for the new tests to correctly prove the verbatim-message assertion against the real save path. No scope creep — no fixture anywhere else in the repository needed correction, since no pre-existing plugin value was non-bare or absent.

## Issues Encountered

None beyond the deviation above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All twelve new regression tests pass; `CGO_ENABLED=0 go build ./...` and `make test-portable` both exit 0 across the root kernel module and every workspace plugin module — no pre-existing fixture anywhere in the repo carried a non-bare or absent plugin value, so no fixture corrections were needed.
- `make docs-check` exits 0; the newly-published bare-filename rule is present in `docs/plugin-contract.md`'s Trust tiers section.
- All three Phase 11 Playwright e2e specs (`11-external-tier-badge`, `11-untrusted-add`, `11-binary-changed-repin`) pass unchanged, confirming the symlinked-binary fixture path the regular-file check touches is unaffected.
- CR-01 is closed. ROADMAP success criterion 3's premise ("trust is decided by the kernel from where the binary lives") now holds for every reachable input — PLUG-07 is unblocked, pending the orchestrator's post-wave requirement/roadmap updates.
- No blockers or concerns for subsequent phases.

---
*Phase: 11-external-plugins-the-trust-boundary*
*Completed: 2026-08-13*
