---
phase: 09-ui-polish-and-source-management-rework
plan: 06
subsystem: ui
tags: [svelte5, bits-ui, dropdown-menu, playwright, config-reload]

requires:
  - phase: 09-ui-polish-and-source-management-rework
    provides: "PluginIcon.svelte (09-01 tracer): the kernel-served <img> path with mandatory Puzzle fallback, used here in Manage-sources instance rows"
provides:
  - "WebspaceSwitcher's onreload/reloadBusy props and a Reload config DropdownMenuItem at the menu root, widening D-13's two-item rule to three"
  - "WebspaceHeader's onreload/reloadBusy/reloadError props, threading to WebspaceSwitcher and rendering reloadError as a header-region destructive Alert"
  - "The webspace route's own reload handler (handleReload), owning the reloadConfig() call the same way it already owns oncreatewebspace/onmanagesources"
  - "ManageSourcesModal with no reload control, state, or footer text — one entry point for the action, not two"
  - "Manage-sources instance rows rendering the plugin's own PluginIcon at size-4"
  - web/e2e/specs/09-switcher-reload.spec.ts
affects: [09-07]

actuals:
  tokens: 8700
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Modal-less write ownership relocated to the route (mirrors the existing oncreatewebspace/onmanagesources pattern) rather than living inside the modal it used to be buried in — a third instance of 07-UI-SPEC.md's header-region destructive-Alert pattern, alongside filterError and chip-remove"
    - "Structural test guards assert a control-label SET by equality (not a grep for an absent string) so a lingering control fails as a set mismatch — manage-sources.test.ts's new reload-relocation guard"

key-files:
  created:
    - web/e2e/specs/09-switcher-reload.spec.ts
  modified:
    - web/src/lib/components/WebspaceSwitcher.svelte
    - web/src/lib/components/webspace-switcher.test.ts
    - web/src/lib/components/WebspaceHeader.svelte
    - web/src/routes/w/[webspace]/+page.svelte
    - web/src/lib/components/ManageSourcesModal.svelte
    - web/src/lib/components/manage-sources.test.ts
    - web/src/lib/components/save-state.test.ts

key-decisions:
  - "Task 1's onreload prop was made required (not optional) per the plan's own explicit prop signature (`onreload: () => void`) — since Task 2 wires the real handler, Task 1's own commit pulled forward a minimal one-line placeholder (`onreload={() => {}}`) into WebspaceHeader.svelte's WebspaceSwitcher call site so the tree kept compiling at that intermediate commit (Rule 3, blocking compile dependency), replaced with the real handler in Task 2"
  - "reloadError is a prop distinct from filterError (not merged into one shared error slot) per the plan's own explicit instruction — a filter write failing and a config reload failing are different events with different recovery, and conflating them would make the copy ambiguous about which action failed"
  - "reloadError clears only on a subsequent successful reload (mirroring filterError's own clear-on-success discipline), not on webspace navigation — a reload failure is a whole-config event, not scoped to the webspace being viewed when it happened, so leaving it visible across a navigation is the intentionally correct behavior, not an oversight"

patterns-established:
  - "manage-sources.test.ts's extractButtonLabels helper neutralises '=>' before scanning <Button> opening tags for their real closing '>' — an arrow-function onclick handler's own '=>' would otherwise be mistaken for the tag's terminator by a naive [^>]* scan"

requirements-completed: []

coverage:
  - id: D1
    description: "The webspace switcher's create-webspace entry shows a single \"+\" (New webspace, no doubled text) and the root menu carries exactly three static items in order — New webspace, Reload config, Manage sources… — with separators before the first and after the last"
    verification:
      - kind: unit
        ref: "web/src/lib/components/webspace-switcher.test.ts (static-item-count=3, order, icon, in-flight disable assertions)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/09-switcher-reload.spec.ts > menu structure: New webspace, Reload config, Manage sources… in that order, after the webspace list"
        status: pass
    human_judgment: false
  - id: D2
    description: "Reload config is one click from the webspace menu root; clicking it reaches the kernel, adopts the new config/hash without a page reload, and refreshes sources/the stream"
    verification:
      - kind: e2e
        ref: "web/e2e/specs/09-switcher-reload.spec.ts > reload works: a hand-edited config.toml becomes visible without a page reload, and the menu closes"
        status: pass
    human_judgment: false
  - id: D3
    description: "A failed reload surfaces as a modal-less destructive Alert in the header region (kernel's verbatim message + fixed reassurance sentence), with no dialog opening, and the Reload config item disables while in flight"
    verification:
      - kind: e2e
        ref: "web/e2e/specs/09-switcher-reload.spec.ts > reload failure surfaces without a modal: a destructive header Alert, no dialog"
        status: pass
      - kind: unit
        ref: "web/src/lib/components/webspace-switcher.test.ts > Reload config disables while a reload is in flight"
        status: pass
    human_judgment: false
  - id: D4
    description: "ManageSourcesModal offers no second reload entry point — no button, no footer text, no local reload state"
    verification:
      - kind: unit
        ref: "web/src/lib/components/manage-sources.test.ts > reload control relocation: the Dialog region's own control-label set equals exactly {Delete, Edit}"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/09-switcher-reload.spec.ts > no second entry point: Manage sources… offers no reload control"
        status: pass
    human_judgment: false
  - id: D5
    description: "Manage-sources instance rows show each instance's plugin icon at 16px (size-4), matching the Pencil/Trash2 sizing already there, with the row's two-line layout otherwise unchanged"
    verification:
      - kind: unit
        ref: "web/src/lib/components/manage-sources.test.ts > instance rows: leading PluginIcon at size-4"
        status: pass
    human_judgment: false
  - id: D6
    description: "07-UI-SPEC.md D-13's two-static-item rule is widened to three deliberately, with the superseding test updated in place rather than routed around"
    verification:
      - kind: unit
        ref: "web/src/lib/components/webspace-switcher.test.ts > menu contents: exactly the three 09-UI-SPEC.md Fix 7 static items, no fourth action item"
        status: pass
    human_judgment: false

duration: ~45min
completed: 2026-08-11
status: complete
---

# Phase 9 Plan 6: Switcher-Root Reload Config + Manage-Sources Icons Summary

**`Reload config` relocated from a footer button nobody saw to a one-click item at the webspace switcher's menu root — owned by the route, failing into the existing header-region Alert pattern — plus the create-webspace entry's doubled "+" fixed and plugin icons added to Manage-sources instance rows, all proven end to end by a new four-spec Playwright file.**

## Performance

- **Duration:** ~45 min
- **Completed:** 2026-08-11
- **Tasks:** 3
- **Files modified:** 8 (1 created, 7 modified)

## Accomplishments

- `WebspaceSwitcher.svelte`'s create-webspace item reads `New webspace` (was `+ New webspace`) — the leading `Plus` icon is now the only "+" affordance, matching every other icon-led item in the app.
- A new `Reload config` `DropdownMenuItem` (leading `RotateCw`, disabled while in flight) sits at the menu root between `New webspace` and the separator before `Manage sources…` — deliberately widening 07-UI-SPEC.md D-13's locked "exactly two static items" rule to three, per 09-UI-SPEC.md Fix 7's explicit supersession, with the widening documented in the test itself rather than silently routed around.
- The webspace route (`+page.svelte`) now owns the reload call outright (`handleReload`), mirroring how it already owns `oncreatewebspace`/`onmanagesources`: it calls `reloadConfig()`, adopts the returned config/hash into the same state `getConfig()`/`putConfig()` already populate (so a hand-edited `config.toml` becomes visible without a page reload), and refreshes sources/the stream through the identical D-07 eager-reconcile paths every other config-changing handler uses. Failure sets a `reloadError` carrying the kernel's verbatim message plus the fixed reassurance sentence; success clears it.
- `WebspaceHeader.svelte` gained `onreload`/`reloadBusy`/`reloadError` props — the first two threaded straight to `WebspaceSwitcher`, and a non-null `reloadError` rendering a destructive `Alert` in the header region using the identical treatment as the existing `filterError` Alert (kept as a distinct prop by design: a filter failure and a reload failure are different events with different recovery).
- `ManageSourcesModal.svelte` dropped its own `Reload config` button, footer helper text, `reloading`/`reloadError` state, `handleReload`, and its `reloadConfig`/`RotateCw` imports entirely — one entry point for this action, not two.
- Each Manage-sources instance row now renders a leading `PluginIcon` at `size-4` (16px, matching the row's existing `Pencil`/`Trash2` sizing) — landed on top of 09-01's kernel-served icon contract, `shrink-0` so it survives a long truncating display name.
- `web/e2e/specs/09-switcher-reload.spec.ts` proves all of the above live: the switcher's three-item order, a real reload making a hand-edited `config.toml` visible with no page reload (the stronger of the plan's two offered assertion shapes), a failed reload's modal-less header Alert with no dialog opening, and the absence of any reload control inside Manage sources….

## Task Commits

1. **Task 1: Three static items at the switcher root, one "+" among them** - `0a6d4a0` (feat)
2. **Task 2: The route owns the reload; the modal drops its duplicate** - `5d4dff2` (feat)
3. **Task 3: Plugin icons on Manage-sources rows, and browser proof for the reload path** - `37f40e7` (feat)

_No separate plan-metadata commit — worktree mode; the orchestrator commits SUMMARY.md centrally after the wave._

## Files Created/Modified

- `web/src/lib/components/WebspaceSwitcher.svelte` - single-"+" create label, new `onreload`/`reloadBusy` props, `Reload config` item at the menu root
- `web/src/lib/components/webspace-switcher.test.ts` - static-item-count widened from two to three (D-13 supersession documented in place), plus order/icon/in-flight-disable assertions
- `web/src/lib/components/WebspaceHeader.svelte` - `onreload`/`reloadBusy`/`reloadError` props, threaded to `WebspaceSwitcher`, `reloadError` header Alert
- `web/src/routes/w/[webspace]/+page.svelte` - `handleReload` owning the `reloadConfig()` call, `reloadBusy`/`reloadError` state, wired into `WebspaceHeader`
- `web/src/lib/components/ManageSourcesModal.svelte` - reload button/state/imports removed; leading `PluginIcon` added to each instance row
- `web/src/lib/components/manage-sources.test.ts` - reload-relocation control-label-set guard, `PluginIcon` sizing/presence assertions
- `web/src/lib/components/save-state.test.ts` - `ManageSourcesModal` case updated to drop the now-nonexistent reload-button assertion (Rule 1, see Deviations)
- `web/e2e/specs/09-switcher-reload.spec.ts` - four-spec browser proof of the whole relocation

## Decisions Made

- **`onreload` made a required prop, not optional (Task 1 boundary decision).** The plan's own action text specifies `onreload: () => void` (no `?`). Since `WebspaceHeader.svelte` (Task 2's declared file) is the only call site, Task 1's commit pulled forward a minimal one-line placeholder (`onreload={() => {}}`) into that call site so `npm run check` stayed green at Task 1's own commit boundary — Rule 3, a blocking compile dependency, not scope creep on Task 2's actual work (the real handler and `reloadBusy`/`reloadError` wiring landed in Task 2 as planned).
- **`reloadError` kept as a prop distinct from `filterError`**, per the plan's explicit instruction — a filter write failing and a config reload failing are different events with different recovery; merging them would make the header Alert's copy ambiguous about which action actually failed.
- **`reloadError` is NOT cleared on webspace navigation**, unlike `filterError`. A reload failure is a whole-application-config event (the previous configuration is still running, globally), not scoped to whichever webspace was on screen when it happened — clearing it on navigation would hide a still-relevant warning, so only a subsequent successful reload clears it.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `WebspaceHeader.svelte`'s `WebspaceSwitcher` call site needed a temporary `onreload` placeholder in Task 1's own commit**
- **Found during:** Task 1 (adding the required `onreload` prop to `WebspaceSwitcher.svelte`)
- **Issue:** `WebspaceSwitcher.svelte`'s new `onreload: () => void` prop is required, but its only call site (`WebspaceHeader.svelte`) isn't touched until Task 2 per the plan's own file scoping — leaving Task 1's own commit failing `npm run check`.
- **Fix:** Added a one-line `onreload={() => {}}` placeholder at the `WebspaceSwitcher` call site in Task 1's commit; Task 2 replaced it with the real `{onreload}` threading to the route's `handleReload`.
- **Files modified:** `web/src/lib/components/WebspaceHeader.svelte`
- **Verification:** `npm run check` reports 0 errors at both the Task 1 and Task 2 commit boundaries.
- **Committed in:** `0a6d4a0` (Task 1 commit), superseded by `5d4dff2` (Task 2 commit)

**2. [Rule 1 - Bug] `save-state.test.ts`'s ManageSourcesModal case asserted a control this task removes**
- **Found during:** Task 2 (removing `ManageSourcesModal`'s own reload button/state)
- **Issue:** `save-state.test.ts` (a cross-component guard, not in this plan's declared file list) asserted `onclick={handleReload}` and `disabled={reloading}` existed inside `ManageSourcesModal.svelte` — both removed by this task, so the pre-existing test would fail outright as a direct, mechanical consequence of Task 2's own change.
- **Fix:** Updated the test's `ManageSourcesModal` case to drop the now-nonexistent reload-button assertion, keeping only the two `AlertDialogAction` disabled-state checks it already covered; removed the now-unused `extractBetween` helper this case was the file's only caller of.
- **Files modified:** `web/src/lib/components/save-state.test.ts`
- **Verification:** `npm test` — all 691 tests passing after the fix (was 1 failing before).
- **Committed in:** `5d4dff2` (Task 2 commit)

**3. [Rule 1 - Bug] The new e2e spec's menu-structure test used raw, un-normalised text for comparison**
- **Found during:** Task 3, first live run of `web/e2e/specs/09-switcher-reload.spec.ts`
- **Issue:** `menu.getByRole('menuitem').allTextContents()` returns each element's raw `textContent`, which still carries the surrounding template whitespace each multi-line `DropdownMenuItem` leaves around its icon and label — `indexOf('New webspace')` failed even though the item was correctly present and its Playwright-computed accessible name was correctly trimmed.
- **Fix:** Normalised each entry (`.trim().replace(/\s+/g, ' ')`) before the `indexOf` comparisons.
- **Files modified:** `web/e2e/specs/09-switcher-reload.spec.ts`
- **Verification:** `make e2e E2E_ARGS="specs/09-switcher-reload.spec.ts"` — 4/4 passing after the fix (was 3/4 before).
- **Committed in:** `37f40e7` (Task 3 commit)

---

**Total deviations:** 3 auto-fixed (1 blocking compile dependency, 2 bugs — one in a pre-existing cross-component test, one in this plan's own new spec).
**Impact on plan:** No scope creep. All three fixes are direct, mechanical consequences of this task's own declared changes (the required prop's compile dependency, and two tests that needed updating to match behavior this plan's own tasks changed) — none touch files or behavior outside what the plan itself specifies.

## Issues Encountered

None beyond the deviations documented above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The reload relocation and Manage-sources icon work are both production-quality and self-contained; no follow-up work is implied for this plan's own scope.
- 09-07 (the "+" add-source picker rework) can proceed independently — it touches `AddSourceModal.svelte`, not any file this plan modified, and already depends on 09-01's `PluginIcon.svelte` the same way this plan's Task 3 did.
- No blockers.

---
*Phase: 09-ui-polish-and-source-management-rework*
*Completed: 2026-08-11*

## Self-Check: PASSED

All 8 created/modified files verified present on disk; all 3 task commit hashes (`0a6d4a0`, `5d4dff2`, `37f40e7`) verified present in `git log --oneline --all`.
