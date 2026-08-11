---
phase: 09-ui-polish-and-source-management-rework
plan: 05
subsystem: ui
tags: [svelte5, dropdown-menu, playwright, intl-relativetimeformat, aria]

requires:
  - phase: 09-01
    provides: "SourceChip.svelte's plugin-declared identity icon (PluginIcon between the health dot and the display name) — this plan edits the same file and builds on that state"
provides:
  - "SourceChip.svelte tooltipText: success/warning branches use formatRelativeTime's phrase verbatim, no appended ' ago' — correct for every Intl.RelativeTimeFormat numeric-auto shape (numeric delta, 'yesterday', 'last week', 'now')"
  - "SourceChip.svelte: one trailing hover/focus-visible control (the overflow trigger only) — 'Refresh now' folded into the DropdownMenu as its first item, with a syncing-disabled guard the removed standalone button never had"
  - "Overflow trigger aria-label broadened from 'Edit {name}' to '{name} actions'"
  - "web/e2e/specs/09-chip-menu.spec.ts: permanent Playwright gate for one-control/menu-order/refresh-wired/tooltip-dedup"
affects: [09-06, 09-07]

actuals:
  tokens: 14688
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "DropdownMenuItem carrying its own disabled/spin-while-in-flight guard, rather than a standalone Button — the syncing guard moved with the control, and gained a check it never had as a standalone button"

key-files:
  created:
    - web/src/lib/components/source-chip-tooltip.test.ts
    - web/e2e/specs/09-chip-menu.spec.ts
  modified:
    - web/src/lib/components/SourceChip.svelte
    - web/src/lib/components/chip-edit-menu.test.ts
    - web/src/lib/components/source-chip-pill.test.ts
    - web/src/lib/components/source-chip-selected.test.ts
    - web/e2e/specs/09-plugin-icon.spec.ts
    - web/e2e/specs/uat-02-remove-source-items.spec.ts
    - web/e2e/specs/uat-03-empty-shell-first-source.spec.ts
    - web/e2e/specs/uat-05-two-step-connect.spec.ts
    - web/e2e/specs/uat-06-remove-chip-roundtrip.spec.ts
    - web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts
    - web/e2e/specs/uat-10-describe-race.spec.ts

key-decisions:
  - "Retargeted (not deleted) two Phase 6 structural guards (source-chip-pill.test.ts's G-06-3b geometry guard, source-chip-selected.test.ts's G-06-3 colour guard) from the removed standalone refresh Button to the chip's sole remaining Button, the overflow trigger — same geometry/reveal/re-toning properties, still guarded on the one control that now carries that surface"
  - "Updated seven pre-existing e2e specs whose selectors encoded the removed standalone-refresh-button / edit-only aria-label state (Rule 3 — same-package test dependency directly broken by this task's aria-label/control change, same precedent as 05-02/07-11): 'Edit {name}' -> '{name} actions' across uat-02/03/05/06/08/10, and the three 'Refresh {name}' clicks in uat-02 now open the actions menu and click its 'Refresh now' item"

patterns-established: []

requirements-completed: []

coverage:
  - id: D1
    description: "Every tooltip variant (success/warning/destructive/unknown, and every Intl.RelativeTimeFormat numeric-auto shape) reads as a correct, non-doubled sentence"
    verification:
      - kind: unit
        ref: "web/src/lib/components/source-chip-tooltip.test.ts (12 assertions: structural template + behavioral against real formatRelativeTime output for a numeric delta, 'yesterday', 'last week', 'now')"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/09-chip-menu.spec.ts#the tooltip reads a de-duplicated sync phrase, catching every \"word ago\" variant, not just the common one"
        status: pass
    human_judgment: false
  - id: D2
    description: "The chip reveals exactly one trailing hover/focus-visible control (the overflow trigger); the standalone refresh Button is gone"
    verification:
      - kind: e2e
        ref: "web/e2e/specs/09-chip-menu.spec.ts#the chip reveals exactly one trailing hover-revealed control"
        status: pass
      - kind: unit
        ref: "web/src/lib/components/source-chip-pill.test.ts / source-chip-selected.test.ts (retargeted geometry/reveal/colour guards)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Refresh now is the menu's first item, disabled+spinning while syncing, calls the existing onrefresh(source.name), and the menu order is Refresh now / separator / Edit connection… / Edit match settings… / (Re-link…, WhatsApp only) / separator / Remove from this webspace"
    verification:
      - kind: unit
        ref: "web/src/lib/components/chip-edit-menu.test.ts (18 assertions incl. new Refresh-now-syncing-guard block)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/09-chip-menu.spec.ts#the overflow menu renders in the exact Fix 5 order for a non-WhatsApp source, #Refresh now triggers a real refresh request against the kernel"
        status: pass
    human_judgment: false

duration: ~20min
completed: 2026-08-11
status: complete
---

# Phase 9 Plan 5: Chip Tooltip De-duplication and Refresh-into-Menu Summary

**Removed a literal " ago" appended after `formatRelativeTime`'s already-complete phrase, and folded the chip's standalone refresh button into its overflow menu as a syncing-guarded "Refresh now" first item — the chip now has exactly one trailing control, proven in a real browser.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-08-11
- **Tasks:** 3
- **Files modified:** 13 (2 created, 11 modified)

## Accomplishments

- `SourceChip.svelte`'s `tooltipText` success/warning branches now use `formatRelativeTime`'s `${relative}` phrase verbatim — the formatter's own `Intl.RelativeTimeFormat({numeric:'auto'})` output ("5 minutes ago", "yesterday", "last week", "now") is already a complete sentence, so the literal `" ago"` suffix was wrong in every case, not just the commonly-hit one.
- `source-chip-tooltip.test.ts` guards every branch structurally (exact template text) and behaviorally (real `formatRelativeTime` output for a numeric delta and all three special-cased deltas), asserting the sentence a user actually sees.
- The standalone `RefreshCw` `Button` is gone. `Refresh now` is the menu's first `DropdownMenuItem`, `disabled={source.syncing}` (a guard the removed button never had) with its icon carrying `animate-spin` while syncing, calling the unchanged `onrefresh(source.name)`. The chip's public prop shape (`onrefresh`/`onedit`) is byte-identical, so `WebspaceHeader.svelte` needed no edit — confirmed by `git diff --stat` staying empty for that file across all three commits.
- The overflow trigger's `aria-label` broadened from `Edit {name}` to `{name} actions`, and the menu now carries two separators (one after Refresh now, one before Remove).
- `chip-edit-menu.test.ts`'s three stale assertions (aria-label, separator count, five-label ordered set) updated in place; new assertions added for the syncing-disable guard and the spin class.
- `web/e2e/specs/09-chip-menu.spec.ts`: four permanent Playwright cases — exactly one revealed trailing control, the ordered menu sequence for a non-WhatsApp source, a real successful `POST /api/sources/{name}/refresh` round trip fired by the relocated control, and a tooltip assertion that no word repeats immediately after itself (the general form of the "ago ago"/"yesterday ago" defect class).

## Task Commits

1. **Task 1: Stop appending a literal suffix to an already-complete relative phrase** - `06b81b7` (fix)
2. **Task 2: Fold refresh into the overflow menu, leaving one trailing control** - `00331a8` (fix)
3. **Task 3: Browser proof for the chip's single control, menu order and tooltip copy** - `4426c6a` (test)

_No separate plan-metadata commit — worktree mode; the orchestrator commits SUMMARY.md centrally after the wave._

## Files Created/Modified

- `web/src/lib/components/SourceChip.svelte` - tooltip copy fix (Task 1); standalone refresh Button removed, Refresh now added as the menu's first item with a syncing guard, trigger aria-label broadened (Task 2)
- `web/src/lib/components/source-chip-tooltip.test.ts` - new: structural + behavioral guard over every tooltip branch
- `web/src/lib/components/chip-edit-menu.test.ts` - three stale assertions updated in place, syncing-guard/spin assertions added
- `web/src/lib/components/source-chip-pill.test.ts`, `source-chip-selected.test.ts` - Phase 6 structural guards retargeted from the removed refresh Button to the chip's sole remaining Button (the overflow trigger)
- `web/e2e/specs/09-chip-menu.spec.ts` - new: the plan's own Task 3 browser-proof spec
- `web/e2e/specs/09-plugin-icon.spec.ts`, `uat-02/03/05/06/08/10` - selector updates for the removed standalone refresh button / broadened aria-label

## Decisions Made

- **Retargeted two Phase 6 structural test guards rather than deleting them.** `source-chip-pill.test.ts` (G-06-3b geometry) and `source-chip-selected.test.ts` (G-06-3 colour) both located their subject via the first `<Button` in `SourceChip.svelte`, which was the removed standalone refresh Button. Since it's gone, `<Button` now resolves to the overflow trigger — the same geometry (rounded-full, no size-11), reveal-scoping (hover/keyboard-focus, no focus-within pin) and selected-state re-toning properties this file guarded are still real properties of the one remaining Button, so the assertions were retargeted and reworded rather than dropped. The one assertion that genuinely had no equivalent (the refresh Button forcing itself visible while `source.syncing`) was removed with a comment pointing to its replacement in `chip-edit-menu.test.ts`'s new syncing-disable/spin block.
- **Updated seven pre-existing e2e specs outside this plan's declared `files_modified` (Rule 3 — same-package test dependency).** The trigger's aria-label change (`Edit {name}` -> `{name} actions`) and the refresh control's relocation broke every existing spec's selectors for it: `uat-02/03/05/06/08/10-*.spec.ts` and a stale comment in `09-plugin-icon.spec.ts`. Same precedent as 05-02/07-11 (a task's own change breaking a same-package test fixture is fixed in place, not left broken). All were re-verified green by running the full e2e suite (49 specs) after the change, not merely by inspection.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Retargeted two Phase 6 structural test guards broken by removing the standalone refresh Button**
- **Found during:** Task 2, running `npm test` after the markup change
- **Issue:** `source-chip-pill.test.ts` and `source-chip-selected.test.ts` both located `refreshButtonBlock` via `extractBetween(strippedChip, '<Button', '</Button>')` — the first `<Button` in the file. With the standalone refresh Button removed, this now silently resolved to the overflow trigger instead, and one assertion in each file (the syncing-forces-visible check) then failed outright since that behaviour no longer exists on any `Button` element.
- **Fix:** Renamed `refreshButtonBlock` to `triggerButtonBlock` in both files, reworded assertions to describe the overflow trigger (the properties themselves — rounded-full, no size-11, hover/keyboard reveal, no focus-within, selected re-toning — are all still true of it), and removed/replaced the syncing-forces-visible assertions with comments pointing to their new home in `chip-edit-menu.test.ts`.
- **Files modified:** `web/src/lib/components/source-chip-pill.test.ts`, `web/src/lib/components/source-chip-selected.test.ts`
- **Verification:** `cd web && npm test` — all 41 files / 701 tests pass.
- **Committed in:** `00331a8` (Task 2 commit)

**2. [Rule 3 - Blocking] Updated seven e2e specs whose selectors encoded the pre-fix chip state**
- **Found during:** Task 3, running the full e2e suite to verify `09-chip-menu.spec.ts` didn't regress the rest of the suite
- **Issue:** `uat-02/03/05/06/08/10-*.spec.ts` and `09-plugin-icon.spec.ts`'s own comment all referenced the trigger's old `Edit {name}` aria-label or the removed standalone `Refresh {name}` button. Left unfixed, these specs would have started failing (or, worse, silently locating the wrong element) the moment this plan's chip change landed.
- **Fix:** Renamed every `'Edit {name}'` selector to `'{name} actions'` (safe regex excluding the ellipsis-bearing `Edit connection…`/`Edit match settings…` menu items), and replaced the three `'Refresh Mock One'` button clicks in `uat-02-remove-source-items.spec.ts` with opening the actions menu then clicking its `Refresh now` item.
- **Files modified:** `web/e2e/specs/09-plugin-icon.spec.ts`, `uat-02-remove-source-items.spec.ts`, `uat-03-empty-shell-first-source.spec.ts`, `uat-05-two-step-connect.spec.ts`, `uat-06-remove-chip-roundtrip.spec.ts`, `uat-08-whatsapp-qr-link.spec.ts`, `uat-10-describe-race.spec.ts`
- **Verification:** Full e2e suite (`npx playwright test --project=chromium`, 49 specs) green, including every one of these files individually re-run first.
- **Committed in:** `4426c6a` (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 3 — blocking test dependencies directly caused by this task's own markup/aria-label change, in the same package).
**Impact on plan:** No scope creep — both fixes are strictly necessary to keep the suite green after this plan's own declared change. `WebspaceHeader.svelte`'s contract with `SourceChip.svelte` is provably unchanged (`git diff --stat` empty for that file across all three commits), matching the plan's own success criterion.

## Issues Encountered

- `web/node_modules` was absent at session start (same one-time setup cost 09-01 recorded) — resolved with `npm install`.
- Playwright's browser install required `--with-deps` (sudo, no interactive password available in this environment); installed the Chromium browser binary alone (`npx playwright install chromium`), which was sufficient — the sandbox already had the needed system libraries.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- `SourceChip.svelte`'s public prop shape (`onrefresh`/`onedit`) is unchanged, so any later plan touching `WebspaceHeader.svelte` or its measurement clone can rely on this plan having made no contract change.
- The de-duplicated tooltip copy and the single-trailing-control menu are both permanent Playwright gates (`09-chip-menu.spec.ts`), not manual re-checks — per `docs/testing.md`'s standing rule.
- No blockers for 09-06/09-07.

---
*Phase: 09-ui-polish-and-source-management-rework*
*Completed: 2026-08-11*

## Self-Check: PASSED

All 3 created/output files verified present on disk; all 3 task commit hashes (`06b81b7`, `00331a8`, `4426c6a`) verified present in `git log --oneline --all`.
