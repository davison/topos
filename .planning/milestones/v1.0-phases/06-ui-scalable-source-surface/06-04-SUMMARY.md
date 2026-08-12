---
phase: 06-ui-scalable-source-surface
plan: 04
subsystem: ui
tags: [svelte5, resize-observer, tdd, vitest]

# Dependency graph
requires:
  - phase: 06-ui-scalable-source-surface (plan 02)
    provides: the chip-row measurement model (visibleChipCount, four measured DOM regions) that this plan re-attaches to a live observer
provides:
  - an injectable, testable ResizeObserver attachment helper (web/src/lib/resize-observer.ts) that observes only bound targets and never constructs an observer with nothing to watch
  - a ref-driven $effect in WebspaceHeader.svelte replacing the dead one-shot mount-hook attachment, covering all four measured elements including the overflow-trigger clone (IN-01)
  - the closure of 06-VERIFICATION.md's single recorded gap: UI-07 now holds for the session lifetime, not only at initial load
affects: [any future phase touching WebspaceHeader.svelte's chip row or its overflow/measurement model]

# Actuals (#2632)
actuals:
  tokens: 4750
  tasks: 1
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Injectable observer-factory seam (CreateResizeObserver) for testing real-DOM-dependent attachment logic under vitest's environment: 'node' runner, with no component-mount harness"
    - "Comment-stripped TS <script> source-scan guard (extends the existing CSS/route-source pattern from scrollbar-theme.test.ts and pane-layout.test.ts to a Svelte component's script block, with balanced-paren extraction for nested call/arrow shapes)"

key-files:
  created:
    - web/src/lib/resize-observer.ts
    - web/src/lib/resize-observer.test.ts
  modified:
    - web/src/lib/components/WebspaceHeader.svelte

key-decisions:
  - "Observer attachment extracted into web/src/lib/resize-observer.ts rather than inlined in the component, per the plan's own decision record — makes attachment provable by behaviour in the existing node-environment test runner, no component-mount harness, no new dependency"
  - "Structural proof is a comment-stripped source-scan guard (readFileSync/dirname/join/fileURLToPath, comments replaced with a single space, balanced-paren extraction of the $effect and observeResize call text) rather than a raw grep — a bare grep is exactly what let 06-VERIFICATION.md's gap through once already"

patterns-established:
  - "Factory-fixture test helpers that need to expose a live counter/callback to assertions must return a single mutable state object, not individually-destructured getters — destructuring a getter freezes its value at destructure time, before the code under test runs"

requirements-completed: [UI-07]

coverage:
  - id: D1
    description: "observeResize constructs no observer while every target is unbound (the state on first mount, before GET /api/sources resolves), observes exactly the bound targets otherwise, routes the factory callback to the resize handler, and disconnects its teardown exactly once (idempotent)"
    requirement: "UI-07"
    verification:
      - kind: unit
        ref: "web/src/lib/resize-observer.test.ts#observeResize (6 cases)"
        status: pass
    human_judgment: false
  - id: D2
    description: "WebspaceHeader.svelte attaches the observer from a ref-driven $effect (not a one-shot mount hook) covering all four measured elements — rowEl, measureEl, trailingEl, overflowTriggerMeasureEl — with the effect's teardown propagated as the attachment's return value"
    requirement: "UI-07"
    verification:
      - kind: unit
        ref: "web/src/lib/resize-observer.test.ts#WebspaceHeader source-scan guard (5 cases)"
        status: pass
    human_judgment: false
  - id: D3
    description: "A post-load browser-window resize moves chips into and out of the '+N' overflow trigger, the hidden count tracks, no chip is ever clipped at the trailing edge with no trigger showing, and the trigger's health tone updates correctly at the narrowed width"
    verification: []
    human_judgment: true
    rationale: "Requires a live browser window resize and visual/interactive confirmation (chip reflow, trigger dot tone) — not observable from a node-environment unit test or a static source scan"

duration: ~12min
completed: 2026-08-06
status: complete
---

# Phase 6 Plan 04: Live chip-row overflow measurement Summary

**Extracted an injectable `observeResize` attachment helper and rewired `WebspaceHeader.svelte`'s chip row from a dead one-shot `onMount` observer to a ref-driven `$effect` covering all four measured elements, closing 06-VERIFICATION.md's single recorded gap.**

## Performance

- **Duration:** ~12 min
- **Completed:** 2026-08-06T22:08Z
- **Tasks:** 1 (tracer, TDD)
- **Files modified:** 3 (2 created, 1 modified)

## Accomplishments
- `web/src/lib/resize-observer.ts`: a small, DOM-free-importable module exporting `observeResize(targets, onResize, createObserver?)` — constructs nothing when every target is unbound, observes exactly the bound targets otherwise, and returns an idempotent teardown. The default `createObserver` factory references the real `ResizeObserver` constructor lazily (inside the parameter default), so importing the module has no DOM dependency under vitest's `environment: 'node'`.
- `web/src/lib/resize-observer.test.ts`: 11 cases across two describe blocks — 6 behavioural cases proving the helper's bind/skip/callback/teardown/re-attachment semantics against a fake observer, and 5 comment-stripped source-scan cases proving `WebspaceHeader.svelte` actually wires the helper into a ref-driven `$effect` naming all four measured elements, with its teardown propagated, and with the old `onMount`/inline-construction wiring fully removed.
- `WebspaceHeader.svelte`: replaced the `onMount` block (which fired while `rowEl`/`measureEl`/`trailingEl` were all still unbound — the sources request was in flight and the row's render gate was false — so every guarded `observe()` call was skipped and nothing ever re-attached) with `$effect(() => observeResize([rowEl, measureEl, trailingEl, overflowTriggerMeasureEl], measure))`. Reading all four refs synchronously in the effect body registers them as reactive dependencies, so attachment (re)runs whenever a ref binds or rebinds. This also closes IN-01: the overflow-trigger measurement clone (`overflowTriggerMeasureEl`) is now observed directly for the first time — the old wiring never watched it at all.
- Updated the measurement doc comment (lines 58–82) to describe four watched regions and ref-driven attachment, without ever spelling out the observer-construction expression in the file (kept the tree-wide uniqueness gate meaningful).

## Task Commits

Each task was committed atomically, following the plan's own RED/GREEN TDD structure for its single tracer task:

1. **Task 1 (RED): failing test for the attachment helper** - `023c5fd` (test) — `resize-observer.test.ts` added, fails because `resize-observer.ts` doesn't exist yet
2. **Task 1 (GREEN): implement observeResize + rewire WebspaceHeader** - `09c7ea5` (feat) — `resize-observer.ts` created, `WebspaceHeader.svelte` rewired, all 184 tests pass

No REFACTOR commit was needed — the GREEN implementation required no cleanup pass; `npm run check` reported 0 errors on the first green run.

**Plan metadata:** committed separately after this summary (see final commit below).

## Files Created/Modified
- `web/src/lib/resize-observer.ts` - the app's only site constructing a real `ResizeObserver`; exports `observeResize`, `ResizeObserverLike`, `CreateResizeObserver`
- `web/src/lib/resize-observer.test.ts` - helper behavioural tests + `WebspaceHeader.svelte` comment-stripped source-scan guard
- `web/src/lib/components/WebspaceHeader.svelte` - `onMount` import/block removed; ref-driven `$effect` added; measurement doc comment updated to describe four regions and ref-driven attachment

## Decisions Made
- Extracted attachment into a standalone module (per the plan's own decision record) rather than inlining it in the component, so attachment semantics are provable by behaviour under the existing `environment: 'node'` test runner with no component-mount harness and no new dependency.
- The structural half of the proof is a comment-stripped source-scan guard, not a raw grep — following the house pattern from `scrollbar-theme.test.ts`/`pane-layout.test.ts`, extended here with balanced-paren extraction to correctly scope assertions to the specific `$effect(...)` call and its `observeResize(...)` argument list rather than the whole script body.
- During test authoring, the helper-test fixture's `callCount`/`callback` were initially exposed as object getters and destructured immediately by the test bodies — destructuring a getter evaluates it once at destructure time, freezing a stale value before `observeResize` ever ran. Fixed by exposing a single mutable `state` object instead (accessed as `state.callCount` / `state.callback` after the call), which stays live across the destructure. This is a test-fixture-only fix, caught by the RED-then-GREEN discipline itself (the two affected assertions failed on the first `npm test` run after implementing the helper, before any component change was made) — not a deviation from the plan's design, and not logged as a numbered Rule 1–4 deviation since it never touched shipped behavior.

## Deviations from Plan

None - plan executed exactly as written. The single test-fixture correction above was caught and fixed within the TDD cycle itself before any commit landed, so no shipped code deviated from the plan's `<action>` block.

## Issues Encountered

None beyond the test-fixture getter/destructuring issue described above, resolved within the same TDD cycle.

## Next Phase Readiness

- UI-07 moves from PARTIAL to fully satisfied: the chip row's overflow measurement now stays correct for the whole session, not only at initial load.
- IN-01 (06-REVIEW.md) is closed: the overflow-trigger measurement clone is observed directly.
- The plan's automated `<verify>` block passed in full: `npm run check` (0 errors), `npm test` (184/184 passing), the tree-wide `new ResizeObserver` uniqueness gate, and the lockfile-unchanged gate.
- The plan's `<human-check>` (live browser-window-resize confirmation of chip reflow and trigger health-tone update) is recorded as `D3` in this summary's `coverage:` block with `human_judgment: true` — per this project's `human_verify_mode: end-of-phase` config, it is not gated mid-flight here and will be harvested into phase-level UAT for confirmation.
- No dependency was added; `web/package.json`/`web/package-lock.json` are unchanged.

---
*Phase: 06-ui-scalable-source-surface*
*Completed: 2026-08-06*
