---
phase: 13-per-item-curation-installable-app
plan: 07
subsystem: ui
tags: [svelte, playwright, e2e, undo, kern-09, kern-10]

# Dependency graph
requires:
  - phase: 13-01
    provides: the per-item exclude/include mark write path (handleExclude/handleInclude), the shared toast.ts undo mechanism, and the 13-exclude-tracer.spec.ts fixture shape this plan's spec reuses
  - phase: 13-03
    provides: the multi-select bulk action bar (handleBulkPrimary) and the excluded-items view toggle this plan's spec drives
provides:
  - the WR-01 fix: handleExclude/handleInclude/handleBulkPrimary now snapshot webspace/navGeneration at entry, so a mid-undo-window webspace switch can never retarget the mirror write
  - web/e2e/specs/13-undo-across-webspace-switch.spec.ts, a standing regression gate over all three mark-write paths under a real WebspaceSwitcher navigation
  - reconciled KERN-09/KERN-10 traceability status (Pending -> Gaps Found, matching UI-13/UI-14)
affects: [13-VERIFICATION.md re-run, any future phase touching the mark-write handlers or the WebspaceSwitcher navigation seam]

# Actuals (#2632)
actuals:
  tokens: 3872
  tasks: 3
  commits: 4

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Snapshot reactive route bindings (webspace, navGeneration) into local consts at the top of any handler whose async closure (onUndo) can fire after a delay window a navigation could interrupt — the same discipline writeFilter/handleSearch/ensurePolling already established, now applied uniformly across all three mark-write handlers."
    - "Assert against direct kernel GETs, not rendered DOM, when the SPA's own generation-guard makes a post-navigation UI update deliberately absent by design (load(gen) no-ops for a stale generation)."

key-files:
  created:
    - web/e2e/specs/13-undo-across-webspace-switch.spec.ts
  modified:
    - "web/src/routes/w/[webspace]/+page.svelte"
    - docs/testing.md
    - .planning/REQUIREMENTS.md

key-decisions:
  - "Fixed a bug discovered in the spec's own Task 1 assertion (rows.toHaveCount(0) after excluding one item of a four-item corpus, which only passed by catching a transient render flicker) rather than leaving a spec that could pass for the wrong reason — replaced with the exclude-tracer's established per-id + count-delta pattern."
  - "Used exact: true on every 'Undo' button locator: the fixture's own undo-nav-* webspace names contain 'undo' as a case-insensitive substring, which collided with the WebspaceSwitcher trigger's accessible name under Playwright's default substring name matching."

patterns-established:
  - "A gap-closure plan's RED-before-fix discipline: write the failing spec first, run it against unfixed source, confirm it fails on the SPECIFIC assertion the defect predicts (not a locator/timeout artifact), only then apply the fix."

requirements-completed: [KERN-09, KERN-10]

coverage:
  - id: D1
    description: "Single-item exclude/include undo toast targets the webspace it was created in, not the one navigated to, after a mid-window WebspaceSwitcher navigation"
    requirement: "KERN-09"
    verification:
      - kind: e2e
        ref: "web/e2e/specs/13-undo-across-webspace-switch.spec.ts#single-item exclude in A, switch to B, Undo — the exclusion is reversed in A, not B"
        status: pass
    human_judgment: false
  - id: D2
    description: "Bulk exclude undo (N=2) targets the originating webspace after a mid-window switch — the bulk path shares the single-item path's snapshot discipline"
    requirement: "KERN-09"
    verification:
      - kind: e2e
        ref: "web/e2e/specs/13-undo-across-webspace-switch.spec.ts#bulk exclude in A, switch to B, Undo — every excluded id is restored in A"
        status: pass
    human_judgment: false
  - id: D3
    description: "Detail-pane include undo (the corrupting `add` mirror direction) re-excludes in the originating webspace and never manufactures an exclusion in the webspace navigated to"
    requirement: "KERN-10"
    verification:
      - kind: e2e
        ref: "web/e2e/specs/13-undo-across-webspace-switch.spec.ts#detail-pane include in A, switch to B, Undo — A is re-excluded and B gains no mark"
        status: pass
    human_judgment: false
  - id: D4
    description: "KERN-09/KERN-10 traceability rows reconciled to Gaps Found, matching UI-13/UI-14; no requirement checkbox ticked by this plan"
    requirement: "KERN-09"
    verification:
      - kind: other
        ref: "grep -c '^| KERN-09 | Phase 13 | Gaps Found |$' .planning/REQUIREMENTS.md"
        status: pass
    human_judgment: false

duration: ~30min
completed: 2026-08-14
status: complete
---

# Phase 13 Plan 07: Undo-Across-Webspace-Switch Gap Closure Summary

**Snapshot `webspace`/`navGeneration` into local constants at the top of `handleExclude`, `handleInclude` and `handleBulkPrimary` — closing 13-VERIFICATION.md's single recorded gap — pinned by a three-test Playwright regression spec that drives a real WebspaceSwitcher navigation inside the undo toast's 5000ms window.**

## Performance

- **Duration:** ~30 min (interactive build/test cycles dominate; the actual code change is ~24 lines)
- **Tasks:** 3
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments

- The exclude/include undo toast's mirror write now targets the webspace the toast was **created** in, for all three write paths (single-item exclude, bulk exclude, detail-pane include), even after the user switches webspaces inside the toast's 5000ms window.
- `web/e2e/specs/13-undo-across-webspace-switch.spec.ts` — a new hermetic Playwright spec, three tests, each owning its own webspace pair (six webspaces total) so the absolute `excluded_count` assertions stay hermetic under the worker-scoped kernel fixture. Every assertion reads direct kernel `GET /api/webspaces/{ws}/stream` responses rather than rendered rows, because the route's own generation guard deliberately makes the SPA render no signal for a reversal issued after a navigation.
- `docs/testing.md` names the new spec in the per-item-curation section.
- KERN-09/KERN-10's `.planning/REQUIREMENTS.md` traceability rows now read `Gaps Found`, matching UI-13/UI-14, instead of the stale `Pending`.

## Task Commits

Each task was committed atomically:

1. **Task 1 (RED half): add failing regression spec** — `449c654` (test)
2. **Task 1 (GREEN half): snapshot ws/gen in the three mark handlers** — `405141d` (feat)
3. **Task 2: extend spec to bulk/include paths + docs** — `46cd36d` (test)
4. **Task 3: reconcile KERN-09/KERN-10 traceability** — `7ba1e51` (docs)

_TDD task (Task 1) produced two commits (RED test, GREEN implementation) rather than three — no REFACTOR commit was needed; the fix is three local-constant captures with nothing left to clean up._

## Files Created/Modified

- `web/e2e/specs/13-undo-across-webspace-switch.spec.ts` — new: the three-test regression spec (single-item exclude, bulk exclude, detail-pane include), each proving the undo toast's reversal survives a mid-window `WebspaceSwitcher` navigation.
- `web/src/routes/w/[webspace]/+page.svelte` — `handleExclude`, `handleInclude`, `handleBulkPrimary` each gain `const ws = webspace` / `const gen = navGeneration` as their first statements, used throughout the function body and inside each `onUndo` closure in place of the reactive `webspace`/`navGeneration` bindings.
- `docs/testing.md` — names the new spec in the "per-item curation, real end to end" section and its heading.
- `.planning/REQUIREMENTS.md` — KERN-09/KERN-10 Traceability rows: `Pending` -> `Gaps Found`.

## Decisions Made

- **Fixed a bug in the spec itself, found while building Task 2**: Task 1's own single-item test asserted `await expect(rows).toHaveCount(0)` immediately after excluding ONE item from a four-item mock corpus — an assertion that can never legitimately converge to true, and had only passed (in the first, single-test run) by catching a transient render flicker during the stream's list re-render. Running all three tests together in parallel exposed the flakiness. Replaced with the `13-exclude-tracer.spec.ts`-established pattern: assert the specific excluded item's own locator hits count 0, and the total row count drops by exactly one (`rowCountBefore - 1`). Re-verified: all three tests pass reliably, both individually and inside the full 138-test suite.
- **`exact: true` on every `'Undo'` button locator**: the fixture's own webspace names (`undo-nav-a1`, etc.) contain "undo" as a case-insensitive substring. Playwright's default (non-exact) `getByRole` name matching is substring-based, so `getByRole('button', { name: 'Undo' })` ambiguously matched both the toast's real Undo button and the `WebspaceSwitcher` trigger button (whose accessible name is the current webspace name). `exact: true` on the Undo locator resolves the ambiguity; `switchWebspace`'s own webspace-name locators already used `exact: true` from the start.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed Task 1's own broken row-count assertion**
- **Found during:** Task 2, while adding tests 2 and 3 and running the full spec file in parallel (3 workers)
- **Issue:** `await expect(rows).toHaveCount(0)` immediately after excluding one item of a four-item corpus — the assertion should never converge to true (three items always remain), and had only passed in isolation by catching a transient DOM flicker during the stream's re-render, not by proving the intended post-exclude state.
- **Fix:** Replaced with `await expect(page.locator(`[data-item-id="${itemId}"]`)).toHaveCount(0)` (the specific excluded item is gone) plus `await expect(rows).toHaveCount(rowCountBefore - 1)` (exactly one fewer row) — matching `13-exclude-tracer.spec.ts`'s established pattern.
- **Files modified:** `web/e2e/specs/13-undo-across-webspace-switch.spec.ts`
- **Verification:** All three tests pass reliably, individually and inside the full 138-test `chromium` suite (`npx playwright test --project=chromium`).
- **Committed in:** `46cd36d` (Task 2 commit)

**2. [Rule 1 - Bug] Fixed ambiguous 'Undo' locator during RED confirmation**
- **Found during:** Task 1, the first RED run against unfixed source
- **Issue:** `page.getByRole('button', { name: 'Undo' })` (non-exact) resolved to two elements — the toast's Undo button AND the WebspaceSwitcher trigger button reading `undo-nav-b1`, since Playwright's default name match is a case-insensitive substring match and "undo-nav-b1" contains "undo".
- **Fix:** Added `exact: true` to the Undo button locator.
- **Files modified:** `web/e2e/specs/13-undo-across-webspace-switch.spec.ts`
- **Verification:** Re-ran RED against unfixed source; the spec then failed on the intended assertion (webspace A's `excluded_count`/`items`), not on the locator.
- **Committed in:** `449c654` (Task 1 RED commit — the fix was applied before that commit was made, so no separate commit was needed)

---

**Total deviations:** 2 auto-fixed (both Rule 1 — test-authoring bugs discovered while proving the plan's own acceptance criteria; no source-code deviations beyond the plan's own specified WR-01 fix)
**Impact on plan:** Both fixes were necessary for the spec to actually prove what it claims to prove. No scope creep — the underlying WR-01 fix matches the plan's `<action>` exactly (three local-constant captures, nothing more).

## Verification Evidence

**RED (Task 1, pre-fix):** `npx playwright test --project=chromium e2e/specs/13-undo-across-webspace-switch.spec.ts` against the unmodified route failed on:
```
Error: expect(received).toEqual(expected) // deep equality
  Object {
-   "excludedCount": 0,
-   "hasItem": true,
+   "excludedCount": 1,
+   "hasItem": false,
  }
```
— webspace A still shows the item excluded and `excluded_count: 1` after the Undo click, exactly the defect the spec exists to catch. Not a locator or timeout failure.

**GREEN (Task 1, post-fix):** Same command, 1 test passed.

**Behaviour-assertion probe (Task 2, required by the plan's acceptance criteria):** Temporarily reverted `handleInclude`'s `ws`/`gen` capture alone (restored the reactive `webspace`/`navGeneration` reads), rebuilt, and ran test 3 in isolation. It failed — but on the `WS_A3` poll assertion (10s timeout, never converges: `excludedCount: 0, hasItem: true` instead of the expected `excludedCount: 1, hasItem: false`), one assertion earlier than the `WS_B3` check the plan's acceptance-criteria wording named, because the `WS_A3` poll never resolves and the test aborts there before reaching the `WS_B3` line. This still proves the intended fact: with `handleInclude`'s snapshot removed, the reversal never reaches the originating webspace (A3) — it goes to whichever webspace is live when Undo fires instead. Restored the fix afterward; `git diff --stat` on the route file was empty, confirming a byte-identical round trip.

**Full verification sequence** (plan's `<verification>` block, run in order):
1. `npm --prefix web run check` — 0 errors (10 pre-existing unrelated warnings).
2. `npm --prefix web run test` — 58 files / 1078 tests passed, no new failures.
3. `npx playwright test --project=chromium e2e/specs/13-undo-across-webspace-switch.spec.ts` — 3 tests, 0 failures.
4. `npx playwright test --project=chromium` (full suite, equivalent to `make e2e`'s final step against the already-built binary) — 138 tests, 138 passed, including all 3 new tests and the pre-existing `13-exclude-tracer.spec.ts`/`13-multi-select-bulk-exclude.spec.ts`/`13-excluded-view.spec.ts` specs (proving the snapshot fix did not disturb the same-webspace undo paths they already cover).
5. `go build ./...` — clean.

## Issues Encountered

None beyond the two Rule 1 deviations documented above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Phase 13's single recorded verification gap (13-VERIFICATION.md, KERN-09/KERN-10) is closed and pinned by a standing regression gate.
- KERN-09/KERN-10/UI-13/UI-14 traceability rows now agree (`Gaps Found`); a re-run of `/gsd-verify-work 13` (or equivalent phase re-verification) is what would flip them to `Complete` — this plan deliberately left every checkbox unticked, per its own acceptance criteria, since re-verification's sign-off is not this plan's call.
- Plans 13-01 through 13-06 and their SUMMARYs are unmodified.

## Self-Check: PASSED

- `web/e2e/specs/13-undo-across-webspace-switch.spec.ts` exists on disk: confirmed (`[ -f ]`).
- `web/src/routes/w/[webspace]/+page.svelte` modified as described: confirmed.
- `docs/testing.md` and `.planning/REQUIREMENTS.md` modified as described: confirmed.
- All 4 commit hashes (`449c654`, `405141d`, `46cd36d`, `7ba1e51`) present in `git log --oneline`: confirmed.
- Re-ran plan-level `<verification>` commands 1-5: all pass (see "Verification Evidence" above).
- Re-ran every task's `<acceptance_criteria>` grep/command checks: all pass.

---
*Phase: 13-per-item-curation-installable-app*
*Completed: 2026-08-14*
