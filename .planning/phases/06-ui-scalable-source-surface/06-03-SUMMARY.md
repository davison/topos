---
phase: 06-ui-scalable-source-surface
plan: 03
subsystem: ui
tags: [svelte, tailwind, lucide, vitest, scrollbar, deep-link, fidelity]

requires:
  - phase: 06-ui-scalable-source-surface (plan 02)
    provides: multi-select source filter chips, StreamList's filterItemsBySource/streamVariant derivations
provides:
  - fidelityAffordance (format.ts) — two-class icon/verb/title mapping for OpenInSource.svelte, closing the 04-UAT deep-link-fidelity follow-up (UI-08)
  - dateMarkers (format.ts) — adaptive-thinning date-tick derivation for the stream pane's scrollbar (UI-11)
  - StreamDateMarkers.svelte — the pointer-events-none overlay with clickable tick hit-areas, rendered from +page.svelte
affects: [ui-scalable-source-surface, any future phase touching OpenInSource.svelte or the stream pane's scroll region]

actuals:
  tokens: 8124
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Two-class UI split layered over a three-value data enum (fidelityAffordance vs formatFidelity) — the badge keeps the raw power-user value while a separate pure function derives the coarser at-a-glance affordance."
    - "Absolutely-positioned, pointer-events-none overlay with individually pointer-events-auto hit areas, as a non-scrolling sibling of a scrolling region (two-div wrapper: outer relative container, inner scroll div)."
    - "Adaptive granularity thinning with a floor invariant enforced by construction in the degenerate case (drop candidates within the floor of the last KEPT item, not the last candidate)."

key-files:
  created:
    - web/src/lib/components/StreamDateMarkers.svelte
    - web/src/lib/components/fidelity.test.ts
    - web/src/lib/components/markers.test.ts
  modified:
    - web/src/lib/components/OpenInSource.svelte
    - web/src/lib/components/StreamRow.svelte
    - web/src/lib/format.ts
    - web/src/routes/w/[webspace]/+page.svelte

key-decisions:
  - "The stream-pane scroll div's width/shrink-0 classes are duplicated onto a new outer relative wrapper (the real flex item main.svelte sizes) rather than moved off the scroll div, preserving pane-layout.test.ts's existing source-scan guard unchanged."
  - "ISO week key computed via the standard nearest-Thursday algorithm (no library) since the project has no date library and the plan explicitly rules out new npm dependencies."

patterns-established:
  - "fidelityAffordance/formatFidelity split: badge = raw enum truth, affordance = derived at-a-glance UI signal — reusable whenever a data enum needs both a precise and a coarse rendering."
  - "enforceSpacingFloor's last-KEPT-marker (not last-candidate) comparison is the general pattern for any 'floor holds by construction' adaptive-thinning requirement."

requirements-completed: [UI-08, UI-11]

coverage:
  - id: D1
    description: "Raise-window-only links (conversation-only fidelity) render a distinct icon (AppWindow), verb (Show in), and title from navigating links (exact/anchored: ArrowUpRight, Open in) — the fidelity badge still shows all three raw enum values unchanged"
    requirement: "UI-08"
    verification:
      - kind: unit
        ref: "web/src/lib/components/fidelity.test.ts#fidelityAffordance"
        status: pass
    human_judgment: true
    rationale: "The visual affordance (icon rendering, colour inheritance, truncation at long names) needs a human to confirm it reads correctly in the browser — the unit tests prove the underlying data mapping only."
  - id: D2
    description: "Stream pane scrollbar carries adaptive date-tick markers (day -> week -> month thinning, 24px spacing floor) that click-to-scroll and tooltip-on-hover, without taking pointer control from the native scrollbar or stream rows"
    requirement: "UI-11"
    verification:
      - kind: unit
        ref: "web/src/lib/components/markers.test.ts#dateMarkers"
        status: pass
    human_judgment: true
    rationale: "Overlay positioning, scrollbar-thumb-token hover styling, click-to-scroll behaviour, and pointer-event pass-through to the native scrollbar/rows require live browser interaction to confirm — the unit tests prove the derivation only."

duration: ~20min
completed: 2026-08-06
status: complete
---

# Phase 6 Plan 3: Deep-Link Fidelity Differentiation & Stream Date Markers Summary

**Fidelity-differentiated OpenInSource affordance (icon+verb+title, two-class split over the existing three-value badge) plus an adaptive day/week/month date-marker overlay on the stream pane's scrollbar, both pure-derivation-backed and unit-tested.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-08-06T21:08:45Z
- **Tasks:** 3/3 completed
- **Files modified:** 7 (3 created, 4 modified)

## Accomplishments

- `fidelityAffordance` (format.ts) closes the 04-UAT deep-link-fidelity follow-up: `OpenInSource.svelte` now shows a leading `ArrowUpRight`/"Open in {source}" for navigating links and `AppWindow`/"Show in {source}" for `conversation-only` links, with an explanatory title on the latter — the existing three-value badge (`formatFidelity`) is unchanged.
- `dateMarkers` (format.ts) derives index-proportional, adaptively-thinned date markers (day -> ISO week -> calendar month) from the stream's already-in-memory items, with a 24px spacing floor enforced by construction even in a deliberately degenerate over-dense case.
- `StreamDateMarkers.svelte` renders the markers as a `pointer-events-none` overlay with individually `pointer-events-auto` tick hit-areas, reusing the existing `--scrollbar-thumb`/`--scrollbar-thumb-hover` tokens (no new colour), wired into `+page.svelte` as a non-scrolling sibling of the stream's scroll region.
- Two new colocated vitest files (`fidelity.test.ts`, `markers.test.ts`) cover both derivations exhaustively, including the unrecognised-fidelity fallback and every rung of the adaptive-thinning ladder.

## Task Commits

Each task was committed atomically:

1. **Task 1: Differentiate raise-window-only links from navigating links** - `73950de` (feat)
2. **Task 2: Date-marker overlay on the stream pane's scrollbar** - `899504f` (feat)
3. **Task 3: Unit tests for marker derivation and adaptive thinning** - `31998ce` (test)

**Plan metadata:** committed as part of state-update step below.

## Files Created/Modified

- `web/src/lib/format.ts` - Added `fidelityAffordance`/`FidelityAffordance` (UI-08) and `dateMarkers`/`DateMarker`/`MarkerGranularity` plus the adaptive-thinning helpers (UI-11)
- `web/src/lib/components/OpenInSource.svelte` - Renders the two-class icon/verb/title affordance; badge unchanged
- `web/src/lib/components/StreamRow.svelte` - Added `data-item-id` to the root button (scroll-target hook)
- `web/src/lib/components/StreamDateMarkers.svelte` - New: the overlay component
- `web/src/routes/w/[webspace]/+page.svelte` - Wraps the stream scroll div in a `relative` container, binds scroll element + client height, renders `StreamDateMarkers` gated to the stream-over-search-results, loaded-successfully condition
- `web/src/lib/components/fidelity.test.ts` - New: unit tests for `fidelityAffordance`
- `web/src/lib/components/markers.test.ts` - New: unit tests for `dateMarkers`

## Decisions Made

- Kept the stream-pane scroll div's own overflow/width classes untouched and duplicated the conditional width onto the new outer `relative` wrapper, rather than moving it — this is what keeps `pane-layout.test.ts`'s existing source-scan guard (which asserts a fixed px width on the class attribute containing `overflow-x-hidden`) passing unchanged while still giving `main`'s actual flex child the sizing it needs.
- Implemented ISO week keys via the standard nearest-Thursday algorithm directly in `format.ts` (no date library) — consistent with the plan's "no new npm dependency" constraint and the threat register's `T-06-SC` disposition.
- `dayKey` derives the UTC calendar-day boundary via `Math.floor(timestampUnix / 86400)` rather than constructing a `Date` and reading UTC fields — algebraically identical to `formatItemDate`'s `timeZone: 'UTC'` day boundary, so the two can never disagree, and it's the simplest implementation.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Adjusted +page.svelte's stream-pane wrapper structure to keep pane-layout.test.ts passing**
- **Found during:** Task 2 (Date-marker overlay on the stream pane's scrollbar)
- **Issue:** The plan's literal instruction ("keeping the scroll div's own classes and its conditional width behaviour untouched") conflicts with the functional requirement that the *actual* flex item `main` sizes (now the new outer wrapper, since the overlay needs a non-scrolling sibling container) must carry the width/shrink-0 classes for the two-pane layout to size correctly. Naively moving the width classes off the scroll div broke `pane-layout.test.ts`'s existing source-scan guard, which asserts a fixed px width on the class attribute containing `overflow-x-hidden`.
- **Fix:** Duplicated the conditional width/shrink-0 classes onto both the new outer `relative` wrapper (the real flex item) and the inner scroll div (unchanged classes, matching the guard's expectation), plus added `h-full` to the inner div so it fills the wrapper's height.
- **Files modified:** web/src/routes/w/[webspace]/+page.svelte
- **Verification:** `npm test` (all 173 tests, including pane-layout.test.ts) and `npm run check` (0 errors) both pass.
- **Committed in:** 899504f (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 bug/layout-structure fix necessitated by a pre-existing test guard)
**Impact on plan:** No scope creep — the fix reconciles the plan's literal wording with a pre-existing invariant the plan didn't anticipate; the resulting behaviour matches the plan's intent (fixed-width stream pane, flexible detail pane, non-scrolling overlay) exactly.

## Issues Encountered

None beyond the deviation above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- UI-08 and UI-11 are both closed; Phase 6's remaining success criteria (2 and 4) are satisfied.
- Automated verification is complete: `cd web && npm run check` (0 errors), `cd web && npm test` (173/173 passing), `make test` (kernel unaffected), `git diff --quiet -- web/src/app.css` and `web/package.json web/package-lock.json` (both untouched).
- The plan's `<verify>` `<human-check>` items (visual confirmation of the icon/verb swap in a live Signal vs. paperless item, and interactive confirmation of tick hover/click/drag behaviour against a real multi-date stream) were not exercised in this autonomous run — no live browser session was available. These are the two coverage entries above marked `human_judgment: true`; a follow-up UAT pass against the running app should confirm both before this phase is considered fully closed end-to-end.

---
*Phase: 06-ui-scalable-source-surface*
*Completed: 2026-08-06*
