---
phase: 07-webspace-builder-ui
plan: 08
subsystem: ui
tags: [svelte5, edit-modal, state-management, gap-closure, cr-02]

requires:
  - phase: 07-webspace-builder-ui
    provides: EditSourceModal.svelte, the chip ⋮ menu edit flow, and ManageSourcesModal.svelte (07-04/07-05), whose stale-state bug this plan closes
provides:
  - resetEditSession — the single site that clears the webspace route's edit-session state on every close (Cancel, outside-click/Escape, and save)
  - edit-modal-state.ts — the single, pure seeding site for EditSourceModal's connection/match form state (seedConnectionValues, seedMatchBlock)
  - a defensive, correctly-untracked reset-on-open effect inside EditSourceModal
  - a structural + behavioural regression suite proving CR-02 cannot silently reopen
affects: [07-verification, 07-review, future-webspace-builder-ui-work]

actuals:
  tokens: 6400
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Single edit-session reset site, mirroring AddSourceModal's resetFlowState — every close path routes through one function, never duplicates the clearing logic"
    - "Untracked reset-on-open $effect: read the open flag first (tracked), wrap every subsequent config/prop read and state assignment in svelte's untrack — first use of untrack in this codebase"
    - "Pure seeding module returning fresh objects/arrays (never aliasing the source config document) so mutated component $state can never leak back into the browser-held config"

key-files:
  created:
    - web/src/lib/edit-modal-state.ts
    - web/src/lib/edit-modal-state.test.ts
    - web/src/lib/components/edit-modal-reset.test.ts
  modified:
    - web/src/routes/w/[webspace]/+page.svelte
    - web/src/lib/components/EditSourceModal.svelte

key-decisions:
  - "editMode resets to 'connection' (its declared initial value), not null — its type is a two-value union and the render site passes it straight into a prop of that same type; clearing editInstance alone already destroys the {#if} guard's subtree, which is the whole fix mechanism"
  - "Clearing editInstance unmounts the dialog immediately, without its exit transition — intentional, matching ManageSourcesModal's own edit-modal close behavior; no delay/transition-end hook was added"
  - "Both seeding helpers return fresh objects (with fresh nested agent-grant object and fresh match value arrays) rather than aliasing the config document — a deliberate change from the previous inline seeding, load-bearing for the CR-02 regression test"

requirements-completed: [KERN-08, UI-12]

coverage:
  - id: D1
    description: "Closing the edit modal (Cancel, outside-click/Escape, or a successful save) always ends the edit session via one shared resetEditSession, so the {#if configResponse && editInstance} guard destroys the EditSourceModal subtree on every close and the next open always mounts a fresh component"
    requirement: "UI-12"
    verification:
      - kind: unit
        ref: "web/src/lib/components/edit-modal-reset.test.ts#CR-02: closing the edit modal ends the edit session (route-side)"
        status: pass
    human_judgment: false
  - id: D2
    description: "EditSourceModal has exactly one seeding site (edit-modal-state.ts's seedConnectionValues/seedMatchBlock) plus a defensive reset-on-open effect that tracks only the open flag — every config/instance/webspace read inside it is untracked so a parent refresh mid-edit cannot wipe in-progress typing"
    requirement: "UI-12"
    verification:
      - kind: unit
        ref: "web/src/lib/components/edit-modal-reset.test.ts#CR-02: EditSourceModal has exactly one seeding site plus an untracked reset-on-open effect"
        status: pass
    human_judgment: false
  - id: D3
    description: "The seeding helpers are proven, behaviourally, to have no memory of a discarded edit session: seeding, mutating the returned value the way an in-progress edit does, then seeding again from the same unchanged config yields the pre-mutation snapshot"
    requirement: "KERN-08"
    verification:
      - kind: unit
        ref: "web/src/lib/edit-modal-state.test.ts#CR-02 regression: a re-seed has no memory of a discarded session"
        status: pass
    human_judgment: false
  - id: D4
    description: "Live make dev walkthrough: Cancel-then-reopen on Edit connection… and Edit match settings… (both via the chip ⋮ menu and via Manage sources…) shows current config, not discarded typing; agent grants are unchanged after the sequence; in-progress typing survives a background sync refresh"
    verification: []
    human_judgment: true
    rationale: "Requires a live kernel (make dev) and real browser interaction against a real config.toml — this is the plan's own <human-check> backstop, folded into the pending end-of-phase UAT walkthrough per 07-VERIFICATION.md's chip ⋮ menu behavior-unverified item"

duration: ~20min
completed: 2026-08-08
status: complete
---

# Phase 07 Plan 08: Close CR-02 — Edit Modal Stale-State Resurfacing Summary

**Every close of the source edit modal now ends its edit session — one shared `resetEditSession` on the route, plus a pure, untracked-effect-backed seeding module inside `EditSourceModal` — so Cancel-then-reopen can never resurface, or save, a discarded draft over a real connection.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-08-08T16:39:32Z
- **Tasks:** 2
- **Files modified:** 5 (2 created source files, 2 test files, 2 modified components — 3 net-new files, 2 modified)

## Accomplishments

- `handleEditClose` and `handleEditSaved` in the webspace route now both call a single `resetEditSession`, which nulls `editInstance`, resets `editMode` to `'connection'`, and clears `editVocabulary` — destroying the `{#if configResponse && editInstance}` guard's subtree on every close, so the next open of the same source in the same mode always mounts a fresh `EditSourceModal`
- `web/src/lib/edit-modal-state.ts` is the single seeding site for the modal's form state — `seedConnectionValues`/`seedMatchBlock` return fresh objects (with fresh nested `agent` object and fresh match value arrays), never the live config document, closing the aliasing hole that would otherwise let a form edit reach into the browser-held config
- `EditSourceModal.svelte`'s two `$state` initialisers now call those helpers instead of holding inline seeding expressions, and a new reset-on-open `$effect` re-seeds both pieces of form state whenever `open` transitions to true — tracking the open flag alone via `svelte`'s `untrack` (first use in this codebase) so a parent config refresh landing mid-edit can never wipe in-progress typing
- A structural + behavioural regression suite closes the gap for good: `edit-modal-reset.test.ts` scans both the route and the modal's comment-stripped source for the single reset site, the correct `resetEditSession`-before-`await` ordering, exactly two call sites per seeding helper, and `untrack(` ordered before the seeding calls; `edit-modal-state.test.ts` proves the named CR-02 regression behaviourally (mutate a seeded result, re-seed from the same unchanged config, assert no memory of the mutation)

## Task Commits

Each task was committed atomically:

1. **Task 1: Close CR-02 — closing the edit modal ends the edit session, so every reopen remounts from current config** - `ae19c52` (fix)
2. **Task 2: One seeding site plus a reset-on-open second layer, with the behavioural CR-02 regression proof** - `ade4287` (feat)

_Note: no TDD tasks in this plan; each task is a single implementation + test commit._

## Files Created/Modified

- `web/src/routes/w/[webspace]/+page.svelte` - added `resetEditSession`; `handleEditClose`/`handleEditSaved` route through it (the latter before its awaited reload triple); updated the edit-session state's doc comment to record why
- `web/src/lib/components/EditSourceModal.svelte` - `$state` initialisers now call `seedConnectionValues`/`seedMatchBlock`; added the `untrack`-wrapped reset-on-open `$effect`
- `web/src/lib/edit-modal-state.ts` - new pure module: `seedConnectionValues`, `seedMatchBlock`, both returning fresh objects/arrays, never mutating the passed config
- `web/src/lib/edit-modal-state.test.ts` - new behavioural suite, including the named CR-02 regression case and never-mutates-input coverage
- `web/src/lib/components/edit-modal-reset.test.ts` - new structural guard, two describe groups (route-side reset, component-side seeding/untrack ordering) plus the ManageSourcesModal cross-check

## Decisions Made

- `editMode` resets to `'connection'` (its declared initial value), not `null` — widening its type to nullable would fail `npm run check` at the render site's prop, and it isn't needed: clearing `editInstance` alone already makes the `{#if}` guard false, which is the entire fix mechanism. The mode/vocabulary reset is session hygiene layered on top.
- Clearing `editInstance` unmounts the dialog immediately, with no exit transition — intentional, matching `ManageSourcesModal`'s own edit-modal close behavior. No delay or transition-end hook was added, since either would reopen a window where the stale component is still mounted.
- Both seeding helpers return **fresh** objects (deliberately diverging from the previous inline seeding, which handed back the live config object) — this is what makes the CR-02 regression test meaningful and prevents a mutated form value from ever reaching the config document the browser holds outside a save path.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 07-VERIFICATION.md's `gaps[1]` (edit-source modal stale-state resurfacing / CR-02) is now closed: reset-on-close on the route side, reset-on-open on the modal side, and a regression suite pinning both.
- Sibling plan 07-07 (same wave) closed `gaps[0]` (agent-route config staleness / CR-01) earlier in this wave. With both plans' SUMMARY.md now on disk, the shared requirement `KERN-08` can be marked complete.
- Remaining open items from 07-VERIFICATION.md are explicitly out of this plan's scope (WR-01, WR-02, WR-04 warnings; the broader `behavior_unverified` UAT list) and are deferred to a follow-up `/gsd-code-review 7 --fix` pass and the pending end-of-phase `make dev` walkthrough, per this plan's own Scope discipline note.
- No blockers for phase completion once the human `<human-check>` walkthrough (D4 above) is performed live.

## Self-Check: PASSED

- FOUND: web/src/routes/w/[webspace]/+page.svelte
- FOUND: web/src/lib/components/EditSourceModal.svelte
- FOUND: web/src/lib/edit-modal-state.ts
- FOUND: web/src/lib/edit-modal-state.test.ts
- FOUND: web/src/lib/components/edit-modal-reset.test.ts
- FOUND commit ae19c52 (Task 1)
- FOUND commit ade4287 (Task 2)
- `cd web && npm run check`: 0 errors (9 pre-existing warnings, unchanged count)
- `cd web && npm run test`: 31 test files, 492 tests, all passing (up from 30/478 pre-plan)
- `CGO_ENABLED=0 go build ./...`: clean
- `go test ./kernel/... -count=1`: all packages pass
- `git diff --stat web/package.json`: empty
- `git diff --name-only ae19c52~1..ade4287`: exactly the 5 declared files

---
*Phase: 07-webspace-builder-ui*
*Completed: 2026-08-08*
