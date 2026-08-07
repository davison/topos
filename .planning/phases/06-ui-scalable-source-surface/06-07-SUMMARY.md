---
phase: 06-ui-scalable-source-surface
plan: 07
subsystem: ui
tags: [svelte5, tailwind-v4, wcag-contrast, date-markers, gap-closure]

requires:
  - phase: 06-ui-scalable-source-surface plan 03
    provides: the original StreamDateMarkers.svelte overlay, dateMarkers derivation and its shipped-but-unverified visual treatment

provides:
  - streamScrolls/markerLaneTop pure derivations plus a major/minor flag on DateMarker (format.ts), unit tested
  - Dedicated --stream-marker/--stream-marker-strong tokens (app.css), computed >= 3:1 contrast against --background and --card
  - A 12px marker lane clear of the native scrollbar, in a reserved stream-pane gutter, with a rail, two-grade tick hierarchy, pointer/focus affordances and a real scroll-overflow gate (StreamDateMarkers.svelte, +page.svelte)
  - A computed-contrast + structural source-scan guard (marker-overlay.test.ts) so the tone/geometry regression class this gap closes can never silently recur
  - 06-UI-SPEC.md's Date marker overlay section amended with measurable floors, dated 2026-08-07 against G-06-6

affects: [06-ui-scalable-source-surface UAT, any future phase touching the stream pane scroll region or app.css palette tokens]

actuals:
  tokens: 11539
  tasks: 3
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Marker-only design tokens (--stream-marker/--stream-marker-strong) derived via color-mix() from an existing palette token, computed-contrast-tested at authoring time rather than trusted by prose"
    - "Edge-safe overlay positioning via an affine remap (markerLaneTop) applied at render time, leaving the underlying position formula (dateMarkers) untouched and its existing tests intact"
    - "Scroll-overflow gate (streamScrolls) as a standalone predicate rather than folding a third parameter into an existing pure function, keeping every pre-existing call site and assertion stable"

key-files:
  created:
    - web/src/lib/components/marker-overlay.test.ts
  modified:
    - web/src/lib/format.ts
    - web/src/lib/components/markers.test.ts
    - web/src/app.css
    - web/src/lib/components/StreamDateMarkers.svelte
    - web/src/routes/w/[webspace]/+page.svelte
    - .planning/phases/06-ui-scalable-source-surface/06-UI-SPEC.md

key-decisions:
  - "65% color-mix of --muted-foreground chosen for --stream-marker's rest tone (computed 3.81:1 vs --background, 3.67:1 vs --card) — both clear the 3:1 WCAG 1.4.11 floor with headroom, verified by direct computation rather than trusted from the UI-SPEC's prior 'reuse the thumb token' instruction, which had produced 1.86:1"
  - "markerLaneTop's degenerate case (track shorter than twice the inset) clamps the usable range to zero rather than allowing it to go negative, avoiding an inverted (decreasing) position mapping — a conservative choice matching the plan's own 'clamp rather than invert' instruction"
  - "Declined date labels at major boundaries (recorded in 06-UI-SPEC.md) — a 12px lane cannot host legible label text without either shrinking the stream pane or pushing the ruler back over the row list, reintroducing the defect being fixed; the tooltip and major/minor hierarchy already carry that information"
  - "The overlay container's right offset and lane width are expressed as arbitrary-value Tailwind utilities (right-[12px]/w-[12px]) rather than the named spacing scale, specifically so marker-overlay.test.ts's geometry guard can parse an exact pixel value and compare it against app.css's own declared scrollbar width without a second hardcoded copy of either number"

patterns-established:
  - "Design tokens with a stated numeric floor (contrast, geometry) get a computed guard test at the same time they're introduced, not deferred to a later human-judgment UAT pass — closing the exact process gap (06-03-SUMMARY.md's unverified human_judgment claim) that let G-06-6 ship"

requirements-completed: [UI-11]

coverage:
  - id: D1
    description: "Pure derivations (streamScrolls, markerLaneTop, major/minor flag) added to format.ts, fully unit tested including boundary/degenerate cases, with every pre-existing dateMarkers assertion still passing unmodified"
    requirement: UI-11
    verification:
      - kind: unit
        ref: "web/src/lib/components/markers.test.ts (33 tests)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Marker overlay moved into its own 12px gutter lane clear of the native scrollbar, with dedicated >=3:1-contrast tokens, a grouping rail, major/minor tick hierarchy, cursor-pointer and a real focus-visible ring, and a scroll-overflow gate so a non-scrolling stream renders nothing"
    requirement: UI-11
    verification:
      - kind: unit
        ref: "web/src/lib/components/marker-overlay.test.ts (20 tests: computed contrast + structural geometry/form guard)"
        status: pass
      - kind: manual_procedural
        ref: "06-07-PLAN.md Task 2 <human-check>: live browser confirmation of the ruler's visual read, tick hierarchy, edge clipping, hover/focus/click behavior, and the non-scrolling/search-hidden cases"
        status: unknown
    human_judgment: true
    rationale: "No live browser/display session was available in this execution environment (same limitation recorded in 06-03-SUMMARY.md, the plan this gap closure fixes) — the visual read of the shipped ruler still needs a human UAT pass against the running app, even though every computable property (contrast, geometry, token wiring) is now proven automatically."
  - id: D3
    description: "06-UI-SPEC.md's Date marker overlay section and Color table amended with measurable floors (tone, lane geometry, form contract, both render gates) and the declined label option, dated against G-06-6"
    requirement: UI-11
    verification:
      - kind: other
        ref: "grep -q 'G-06-6' .planning/phases/06-ui-scalable-source-surface/06-UI-SPEC.md"
        status: pass
    human_judgment: false

duration: 32min
completed: 2026-08-07
status: complete
---

# Phase 6 Plan 7: Stream Date-Marker Ruler Rebuild Summary

**Closed gap G-06-6 by giving the stream date-marker overlay its own 12px gutter lane, dedicated >=3:1-contrast tokens, a rail-and-hierarchy visual form, edge-safe positioning, a real scroll-overflow gate, and a computed guard test that proves the tone and geometry rather than trusting prose.**

## Performance

- **Duration:** 32 min
- **Started:** 2026-08-07T01:08:27+01:00
- **Completed:** 2026-08-07T01:17:14+01:00
- **Tasks:** 3 completed
- **Files modified:** 7 (1 created, 6 modified)

## Accomplishments

- Added `streamScrolls`, `markerLaneTop`/`MARKER_LANE_INSET_PX`, and a `major` flag on every `DateMarker` to `format.ts`, following a strict RED→GREEN TDD cycle — every pre-existing `dateMarkers` assertion in `markers.test.ts` still passes byte-identical.
- Rebuilt `StreamDateMarkers.svelte`'s visual treatment entirely: a 12px lane offset 12px clear of the native scrollbar (covering both the ~11px `scrollbar-width: thin` render and the 10px webkit fallback), a faint `--border` rail, minor/major tick grades painting the two new marker-only tokens, `cursor-pointer`, a real inset focus-visible ring, and a bounded `observeResize`-driven scroll-overflow gate mirroring `WebspaceHeader.svelte`'s own chip-row measurement pattern.
- Reserved a 24px right gutter on the stream pane's scroll region (`+page.svelte`) so the lane composites against one uniform surface instead of banding across the alternating row-list tones.
- Declared `--stream-marker`/`--stream-marker-strong` in `app.css`, derived via `color-mix()` from `--muted-foreground` at 65%/100% — computed at 3.81:1/3.67:1 (rest) against `--background`/`--card`, both clearing the WCAG 1.4.11 3:1 non-text floor with headroom, versus the retired `--scrollbar-thumb` reuse's 1.86:1.
- Wrote `marker-overlay.test.ts`: a from-disk, computed-WCAG-contrast + structural source-scan guard (20 assertions) proving the tone floor, the lane's scrollbar clearance, the rail, the major/minor branch, the pointer/focus affordances, and the reserved gutter — so this class of regression (an arithmetically obvious geometry error shipped as "human judgment: never exercised") cannot silently recur.
- Amended `06-UI-SPEC.md`'s Date marker overlay section and Color table with a dated G-06-6 note, restating the tone/geometry/form contract as measurable floors and recording the declined date-label option.

## Task Commits

Each task was committed atomically (Task 1 followed the plan's own `tdd="true"` RED/GREEN cycle):

1. **Task 1 (RED): failing tests for streamScrolls, markerLaneTop, major flag** - `b321ba2` (test)
2. **Task 1 (GREEN): implement streamScrolls, markerLaneTop, major-flag hierarchy** - `5475a00` (feat)
3. **Task 2: move the ruler into its own gutter lane** - `e7b85dd` (feat)
4. **Task 3: computed contrast + geometry guard, amend design contract** - `fa89dff` (test)

_TDD Gate Compliance: Task 1 carries both a `test(...)` (RED) and a `feat(...)` (GREEN) commit, per its `tdd="true"` attribute. No `refactor(...)` commit was needed — the GREEN implementation required no cleanup pass._

## Files Created/Modified

- `web/src/lib/format.ts` - `streamScrolls`, `markerLaneTop`/`MARKER_LANE_INSET_PX`, `yearKey`, and the `major` field threaded through `candidateMarkers`/`DateMarker`
- `web/src/lib/components/markers.test.ts` - 14 new tests across `streamScrolls`, `markerLaneTop`, and the major-flag rule at all three granularities plus the degenerate thinned case
- `web/src/app.css` - `--stream-marker`/`--stream-marker-strong` tokens, declared alongside the scrollbar tokens with an extended explanatory comment
- `web/src/lib/components/StreamDateMarkers.svelte` - full visual rebuild: lane geometry, rail, tick hierarchy, scroll-overflow gate, pointer/focus affordances
- `web/src/routes/w/[webspace]/+page.svelte` - `pr-6` gutter reservation on the stream scroll region, with a load-bearing-padding comment
- `web/src/lib/components/marker-overlay.test.ts` - new: computed contrast + structural geometry/form guard (20 tests)
- `.planning/phases/06-ui-scalable-source-surface/06-UI-SPEC.md` - Date marker overlay section and Color table amended, dated against G-06-6

## Decisions Made

- 65% `color-mix` of `--muted-foreground` for `--stream-marker`'s rest tone, chosen after directly computing WCAG contrast (3.81:1 vs `--background`, 3.67:1 vs `--card`) rather than trusting the prior "reuse the thumb token" instruction that had produced 1.86:1.
- `markerLaneTop`'s degenerate case (track shorter than twice the inset) clamps the usable range to zero instead of letting it go negative, avoiding an inverted position mapping — per the plan's own "clamp rather than invert" instruction.
- Declined date labels at major boundaries (recorded in 06-UI-SPEC.md): a 12px lane can't host legible label text without either shrinking the stream pane or reintroducing the row-banding defect; the tooltip and major/minor hierarchy already carry that information.
- The overlay's right offset and lane width use arbitrary-value Tailwind utilities (`right-[12px]`/`w-[12px]`) rather than the named spacing scale, so `marker-overlay.test.ts` can parse an exact pixel value and compare it against app.css's own declared scrollbar width without a second hardcoded copy of either number.

## Deviations from Plan

None — plan executed exactly as written, including the TDD RED/GREEN split on Task 1 that the task's own `tdd="true"` attribute called for.

## Issues Encountered

- The week-granularity major-flag test fixture initially chose a track height (300px) that produced a partial leading ISO week too short (15px) to clear the 24px spacing floor, causing the adaptive algorithm to fall through to month granularity instead of the week granularity the test intended to exercise. Fixed by raising the fixture's track height to 600px, which keeps even the shortest (3-day) partial week's marker spacing above the floor. This was a test-fixture-only fix, made before the GREEN commit — no production code was affected.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Every computable property of the date-marker overlay (rest-tone contrast, lane-vs-scrollbar clearance, token wiring, form/hierarchy structure, gutter reservation) is now proven by an automated guard rather than resting on an unexercised human-judgment claim — the exact process gap that let G-06-6 ship in 06-03.
- One item remains genuinely deferred to human judgment: a live-browser visual confirmation of the rebuilt ruler (Task 2's `<human-check>` — tick hierarchy readable, no edge clipping, hover/focus/click behavior, non-scrolling and search-hidden cases). No display/browser session was available in this execution environment, the same limitation 06-03-SUMMARY.md recorded. This should be exercised as part of 06-UAT.md's Test 6 re-run before the phase is considered fully closed.
- No blockers for subsequent phases — this plan touched only the stream pane's own scroll region, `app.css`'s token declarations, and the marker overlay component; no contract, API, or cross-phase surface changed.

---
*Phase: 06-ui-scalable-source-surface*
*Completed: 2026-08-07*

## Self-Check: PASSED

All key files (format.ts, markers.test.ts, app.css, StreamDateMarkers.svelte, +page.svelte, marker-overlay.test.ts, 06-UI-SPEC.md) confirmed present on disk. All 4 task/RED-GREEN commits (b321ba2, 5475a00, e7b85dd, fa89dff) confirmed in git log.
