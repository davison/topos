---
phase: quick-260811-r5d
plan: 01
subsystem: plugins
tags: [pluginhost, httpapi, playwright, e2e, mockstrict]

requires:
  - phase: 07.1
    provides: browser e2e harness with hermetic per-file kernel, mock plugin fixture, and the uat-08 route-injection idiom this plan generalises
provides:
  - kernel/pluginhost.ExcludedPluginBinaries unconditionally excludes topos-plugin-mockstrict from GET /api/config/plugin-types, mirroring the existing topos-plugin-mock exclusion
  - web/e2e/fixtures/plugin-types.ts (offerPluginType) — shared route-injection helper restoring a kernel-excluded catalog entry inside the Playwright harness only
  - mockstrict-discovery.spec.ts as the permanent two-direction exclusion gate (catalog listing excludes it; a configured instance is unaffected)
affects: [09-ui-polish-and-source-management-rework, 10-docs-and-release-readiness]

actuals:
  tokens: 5955
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Kernel-side catalog exclusions (ExcludedPluginBinaries) are UI-policy only — DiscoverAllBinaries/DescribePluginHandler must always stay unfiltered so already-configured instances of an excluded type keep working"
    - "e2e specs that need a kernel-excluded plugin type to appear in the picker restore it via page.route interception of GET /api/config/plugin-types (web/e2e/fixtures/plugin-types.ts), never by weakening the kernel exclusion"

key-files:
  created:
    - web/e2e/fixtures/plugin-types.ts
  modified:
    - kernel/pluginhost/discover_binaries.go
    - kernel/pluginhost/discover_binaries_test.go
    - kernel/httpapi/config_test.go
    - web/e2e/specs/mockstrict-discovery.spec.ts
    - web/e2e/specs/uat-05-two-step-connect.spec.ts
    - web/e2e/specs/09-picker-groups.spec.ts
    - docs/testing.md
    - Makefile

key-decisions:
  - "Retargeted TestDiscoverBinaries_SymlinkedRegularFileIsDiscovered off topos-plugin-mockstrict onto topos-plugin-silverbullet before adding the exclusion, per the plan's own ordering — excluding mockstrict would otherwise have inverted that test's meaning rather than merely failing it"
  - "09-picker-groups.spec.ts's 'two headed groups render with their exact copy' test also needed the offerPluginType injection — a gap in the plan's own file-scoped injection list, found live by running the full suite as the plan's own action text required, not by reasoning about it (Rule 1 auto-fix, in-scope file)"

patterns-established:
  - "web/e2e/fixtures/plugin-types.ts's offerPluginType generalises uat-08-whatsapp-qr-link.spec.ts's local offerWhatsAppPluginType idiom for any future spec needing a kernel-excluded plugin type restored to the picker catalog"

requirements-completed: [QUICK-260811-r5d]

coverage:
  - id: D1
    description: "GET /api/config/plugin-types never lists topos-plugin-mockstrict, unconditionally, with no env var/config/build-tag escape hatch"
    requirement: "QUICK-260811-r5d"
    verification:
      - kind: unit
        ref: "kernel/pluginhost/discover_binaries_test.go#TestDiscoverBinaries_ExcludesMockstrictBinary"
        status: pass
      - kind: integration
        ref: "kernel/httpapi/config_test.go#TestPluginTypesHandler_ReturnsSortedMockFreeList"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/mockstrict-discovery.spec.ts#GET /api/config/plugin-types excludes BOTH topos-plugin-mock and topos-plugin-mockstrict"
        status: pass
    human_judgment: false
  - id: D2
    description: "The picker's Install a new source group renders no Mockstrict tile"
    requirement: "QUICK-260811-r5d"
    verification:
      - kind: e2e
        ref: "web/e2e/specs/mockstrict-discovery.spec.ts#the picker offers no Mockstrict catalog tile, while the configured mockstrict instance still appears as a Group 1 row"
        status: pass
    human_judgment: false
  - id: D3
    description: "An already-configured topos-plugin-mockstrict instance is unaffected — Group 1 row, describe-plugin 200, config write still accepted"
    requirement: "QUICK-260811-r5d"
    verification:
      - kind: unit
        ref: "kernel/pluginhost/discover_binaries_test.go#TestDiscoverAllBinaries_IncludesMockstrictBinary"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/mockstrict-discovery.spec.ts#describe-plugin for the configured mockstrict instance returns 200 with its tags match vocabulary"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/uat-05-two-step-connect.spec.ts#the required field arrives pre-filled, a blank field blocks with zero requests, and restoring it advances to a real describe-driven Match step"
        status: pass
    human_judgment: false
  - id: D4
    description: "make e2e is green — every spec that drove mockstrict through the catalog still proves the same UI behaviour via route-injection, never deletion"
    requirement: "QUICK-260811-r5d"
    verification:
      - kind: e2e
        ref: "make e2e (67/67 passing, same count as before this plan)"
        status: pass
    human_judgment: false
  - id: D5
    description: "make test is green, including a retargeted symlink test that no longer uses an excluded binary name as the fixture it asserts is RETURNED"
    requirement: "QUICK-260811-r5d"
    verification:
      - kind: unit
        ref: "kernel/pluginhost/discover_binaries_test.go#TestDiscoverBinaries_SymlinkedRegularFileIsDiscovered"
        status: pass
      - kind: unit
        ref: "make test (portable + signal halves)"
        status: pass
    human_judgment: false
  - id: D6
    description: "docs/testing.md and the Makefile name kernel/pluginhost.ExcludedPluginBinaries as the real guarantee, correcting the stale 'keeping mockstrict out of make plugins is enough' claim"
    requirement: "QUICK-260811-r5d"
    verification:
      - kind: other
        ref: "grep -c ExcludedPluginBinaries docs/testing.md Makefile (1, 1); git diff Makefile has 0 non-comment added lines"
        status: pass
    human_judgment: false

duration: 8min
completed: 2026-08-11
status: complete
---

# Quick Task 260811-r5d: Exclude topos-plugin-mockstrict from the picker catalog Summary

**`kernel/pluginhost.ExcludedPluginBinaries` now excludes `topos-plugin-mockstrict` alongside the existing `topos-plugin-mock` entry, with both Go and Playwright gates pinning the exclusion in both directions, and every previously mockstrict-dependent e2e spec preserved via a new shared `offerPluginType` route-injection helper.**

## Performance

- **Duration:** 8 min
- **Started:** 2026-08-11T18:42:51Z
- **Completed:** 2026-08-11T18:51:00Z
- **Tasks:** 3
- **Files modified:** 9 (1 created, 8 modified)

## Accomplishments
- `ExcludedPluginBinaries` map entry for `topos-plugin-mockstrict`, with the doc comment extended to record the `make e2e` -> `bin/plugins/` exposure path that let the fixture reach a live operator's picker
- Falsifiable Go coverage at both the package boundary (`kernel/pluginhost`) and the HTTP route boundary (`kernel/httpapi`), plus a retargeted symlink test that no longer accidentally proves exclusion instead of symlink-following
- A shared `web/e2e/fixtures/plugin-types.ts` (`offerPluginType`) generalising `uat-08`'s local route-interception idiom, applied to every browser spec that drove the picker's catalog through mockstrict — zero specs deleted, skipped, or `fixme`d; `make e2e` stayed at 67/67
- `mockstrict-discovery.spec.ts` rewritten from an inclusion spec into the permanent two-direction exclusion gate, including a real `describe-plugin` round trip against an already-configured instance
- `docs/testing.md` and the `Makefile`'s `e2e` target comment corrected to name the real guarantee (`ExcludedPluginBinaries`), replacing the incomplete "keeping mockstrict out of `make plugins` protects the picker" claim

## Task Commits

Each task was committed atomically:

1. **Task 1: Exclude mockstrict from the picker catalog — kernel, end to end** - `5ed3b21` (feat, TDD RED/GREEN)
2. **Task 2: Preserve every browser-harness catalog flow via type injection** - `53c47ce` (feat)
3. **Task 3: Correct the now-stale "keeping it out of make plugins is enough" claim** - `160853a` (docs)

_Task 1 was TDD: the retargeted symlink test was confirmed green first, then the two new tests (`TestDiscoverBinaries_ExcludesMockstrictBinary`, the extended `TestPluginTypesHandler_ReturnsSortedMockFreeList`) were observed RED (`expected topos-plugin-mockstrict to be excluded, got [...]`) before the map entry landed — all in the same commit per this plan's single-task-per-commit structure, RED evidence captured in this SUMMARY rather than as a separate commit._

**Plan metadata:** not committed by this executor — the orchestrator commits SUMMARY.md/STATE.md/ROADMAP.md/REQUIREMENTS.md separately per this quick task's constraints.

## Files Created/Modified
- `kernel/pluginhost/discover_binaries.go` - `ExcludedPluginBinaries` gains `topos-plugin-mockstrict`; doc comment extended with the exposure-path rationale
- `kernel/pluginhost/discover_binaries_test.go` - retargeted symlink test onto `topos-plugin-silverbullet`; added `TestDiscoverBinaries_ExcludesMockstrictBinary` and `TestDiscoverAllBinaries_IncludesMockstrictBinary`
- `kernel/httpapi/config_test.go` - `TestPluginTypesHandler_ReturnsSortedMockFreeList` extended to write both excluded fixtures and assert the sorted, mock-free response
- `web/e2e/fixtures/plugin-types.ts` (new) - `offerPluginType(page, binary)` route-injection helper
- `web/e2e/specs/mockstrict-discovery.spec.ts` - rewritten into the permanent two-direction exclusion gate with a configured-but-unattached mockstrict instance fixture
- `web/e2e/specs/uat-05-two-step-connect.spec.ts` - `offerPluginType` call added before navigation preceding the catalog-tile click
- `web/e2e/specs/09-picker-groups.spec.ts` - `offerPluginType` calls added to the three tests that need Group 2 populated (including the plan-gap fix, see Deviations)
- `docs/testing.md` - two-mock-shaped-plugins section's trailing paragraph corrected
- `Makefile` - `e2e` target's comment corrected (comment-only, no recipe line changed)

## Decisions Made
- Retargeted `TestDiscoverBinaries_SymlinkedRegularFileIsDiscovered` off `topos-plugin-mockstrict` onto `topos-plugin-silverbullet` before adding the exclusion (plan's own required ordering) — excluding mockstrict first would have inverted the test's meaning (silently proving exclusion rather than symlink-following) rather than merely failing it.
- Kept `DiscoverAllBinaries`, `DescribePluginHandler`, and the whatsapp-link handlers byte-identical to the pre-plan commit (verified via `git diff` against the plan's own commit), preserving the split that fixes the 07.1-04 404 regression class.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `09-picker-groups.spec.ts`'s "two headed groups render with their exact copy" test also needed the injection helper**
- **Found during:** Task 2, running `make e2e` per the task's own instruction to "confirm that by running the full suite, not by reasoning about it"
- **Issue:** The plan named only two of `09-picker-groups.spec.ts`'s five tests (the computed-style distinctness test and the both-add-flows test) as needing `offerPluginType`. A third test, "two headed groups render with their exact copy", also asserts the `Install a new source` heading is visible — which requires Group 2 to be non-empty — and failed once the kernel-side exclusion landed, since it registered no injection.
- **Fix:** Added the same `offerPluginType(page, 'topos-plugin-mockstrict')` call, registered before `page.goto`, to that test; corrected the file's own top-of-fixture comment to name three tests (not two) as needing Group 2 restored.
- **Files modified:** `web/e2e/specs/09-picker-groups.spec.ts`
- **Verification:** Full `npx playwright test --project=chromium` run: 67/67 passing (same count as before this plan)
- **Committed in:** `53c47ce` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug, in-scope file)
**Impact on plan:** No scope creep — the fix stayed inside a file the plan already declared in `<files>` for this task, and restored coverage the plan itself required ("Coverage is preserved by adaptation, never by removal").

## Issues Encountered
- `web/node_modules` did not exist in this worktree (each worktree needs its own install) — ran `npm install` (fast, cache-backed) before `npm run check` and `make e2e` could run. Not a deviation from the plan's own scope, just local environment setup.
- `make e2e`'s SPA build step deleted `kernel/webui/build/.gitkeep` as a side effect of `adapter-static` writing the build output directory. Restored via `git checkout -- kernel/webui/build/.gitkeep` before staging Task 2's commit — not part of this plan's intended changes.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- The exclusion is fully falsifiable in both directions (Go unit test, HTTP route test, e2e catalog-listing test, e2e configured-instance test) — reverting the map entry fails a Go test and an e2e spec, closing the gap this quick task was filed to fix.
- `<verification>` item 5 ("manual confirmation on the operator's own install") is covered by automated equivalent: `mockstrict-discovery.spec.ts`'s "the picker offers no Mockstrict catalog tile, while the configured mockstrict instance still appears as a Group 1 row" test exercises exactly that scenario against a real (hermetic) kernel and browser, per this project's own CLAUDE.md convention that a UAT item a browser can drive becomes a spec rather than a manual check. No live `make build`/manual browser session was additionally run.
- Follow-up noted (not fixed here, deliberately out of scope per Task 3's action text): the cleanest root-cause fix would be for `make e2e` to build its fixture binaries into a directory of their own instead of the shared `bin/plugins/` — left out because `ExcludedPluginBinaries` already makes a stale fixture binary in `bin/plugins/` harmless.

---
*Quick task: 260811-r5d*
*Completed: 2026-08-11*

## Self-Check: PASSED

All 10 files listed in `key-files`/`Files Created/Modified` (plus this SUMMARY.md) verified present on disk. All 3 task commit hashes (`5ed3b21`, `53c47ce`, `160853a`) verified present in `git log --oneline --all`.
