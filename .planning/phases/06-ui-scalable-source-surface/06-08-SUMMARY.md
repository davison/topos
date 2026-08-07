---
phase: 06-ui-scalable-source-surface
plan: 08
subsystem: ui
tags: [svelte, tailwind, source-chip, accessibility, recurrence-guard]

# Dependency graph
requires:
  - phase: 06-06
    provides: "Selected-chip fill treatment (border-primary bg-primary) and its source-chip-selected.test.ts guard"
  - phase: 06-07
    provides: "Header/chip-row layout this plan's chip geometry sits inside"
provides:
  - "SourceChip.svelte's pill declares its own h-11 height (matching the overflow trigger) with no wrapper padding of its own"
  - "The filter button self-stretches to the pill's full content height, so the whole chip body is a filter target"
  - "The refresh control's hover surface is an explicit rounded-full disc, not an implicit rounded-square"
  - "The refresh reveal is scoped to group-has-[:focus-visible], dropping the mouse-click focus pin while keeping the keyboard path"
  - "06-UI-SPEC.md's refresh-reveal wording, chip geometry and 44px touch-target floor reconciled with the shipped code, dated to G-06-3b"
  - "source-chip-pill.test.ts — a geometry/interaction recurrence guard, manually proven to trip on each of the three reintroduced defects"
affects: [ui-scalable-source-surface, source-chip, webspace-header]

# Actuals (#2632)
actuals:
  tokens: 6390
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Chip-scoped keyboard-only reveal via group-has-[:focus-visible]:opacity-100 (arbitrary Tailwind has- variant, confirmed to compile via emitted-stylesheet grep)"
    - "Pill declares its own height utility rather than inheriting from its tallest child; content-owning child uses self-stretch to fill it"

key-files:
  created:
    - web/src/lib/components/source-chip-pill.test.ts
  modified:
    - web/src/lib/components/SourceChip.svelte
    - web/src/lib/components/source-chip-selected.test.ts
    - .planning/phases/06-ui-scalable-source-surface/06-UI-SPEC.md

key-decisions:
  - "Kept source-chip-pill.test.ts as a separate file from source-chip-selected.test.ts (house pattern) — the two guard different axes (colour vs. geometry/interaction) of the same shipped defect class and should fail as distinguishable incidents"
  - "Fell back to no fallback: the primary group-has-[:focus-visible] Tailwind arbitrary variant compiled on the first attempt (confirmed via grep on kernel/webui/build/), so the plan's named group-has-focus-visible fallback was not needed"

patterns-established:
  - "A pill/oval control that nests a smaller interactive child must declare its own height and radius rather than letting the child's own size utilities drive either — recorded as a shipped constraint in 06-UI-SPEC.md, not just a one-off fix"

requirements-completed: [UI-07]

coverage:
  - id: D1
    description: "Chip pill declares h-11 (matching the overflow trigger) and the filter button's self-stretch makes the entire pill body clickable, closing the dead-band defect"
    requirement: "UI-07"
    verification:
      - kind: unit
        ref: "web/src/lib/components/source-chip-pill.test.ts#height parity: the chip and the overflow trigger declare the same height"
        status: pass
      - kind: unit
        ref: "web/src/lib/components/source-chip-pill.test.ts#no dead bands on the pill"
        status: pass
      - kind: unit
        ref: "web/src/lib/components/source-chip-pill.test.ts#the filter button owns the full pill surface"
        status: pass
    human_judgment: true
    rationale: "The source-scan guard proves the class strings are structurally correct, but full-surface clickability and visual height parity with the adjacent overflow trigger require a rendered-pixel check (UAT Test 3) that no node-environment test harness in this repo can perform."
  - id: D2
    description: "Refresh control's hover highlight is a circular disc (rounded-full + size-8), not a rounded-square inside the oval"
    requirement: "UI-07"
    verification:
      - kind: unit
        ref: "web/src/lib/components/source-chip-pill.test.ts#circular refresh surface, and no 44px height driver"
        status: pass
    human_judgment: true
    rationale: "The guard proves the radius/size class strings are present, but the actual rendered hover-fill shape requires visual confirmation."
  - id: D3
    description: "Refresh icon reveal rescoped to group-has-[:focus-visible] — a mouse click no longer pins the icon visible, keyboard tab still reveals it, and the syncing spinner stays force-visible"
    requirement: "UI-07"
    verification:
      - kind: unit
        ref: "web/src/lib/components/source-chip-pill.test.ts#reveal scoping: hover and keyboard focus only, never a mouse-click pin"
        status: pass
      - kind: unit
        ref: "web/src/lib/components/source-chip-pill.test.ts#syncing spinner survives the reveal-scoping change"
        status: pass
      - kind: other
        ref: "grep -rq ':has(:focus-visible)' kernel/webui/build/ (production build)"
        status: pass
    human_judgment: true
    rationale: "Structural presence of the class and its compiled CSS is proven, but the actual pointer-leave-hides / tab-reveals / spinner-stays-visible interaction sequence needs a live browser check (UAT Test 3)."
  - id: D4
    description: "06-UI-SPEC.md reconciled: refresh-reveal wording, new chip-geometry constraint bullet, Tab-order bullet, and both touch-target-floor statements now agree with the shipped code, dated to G-06-3b"
    requirement: "UI-07"
    verification:
      - kind: other
        ref: "grep -q 'G-06-3b' .planning/phases/06-ui-scalable-source-surface/06-UI-SPEC.md"
        status: pass
    human_judgment: false

duration: 13min
completed: 2026-08-07
status: complete
---

# Phase 06 Plan 08: Source Chip Pill Polish (G-06-3b) Summary

**Restructured SourceChip.svelte into one 44px pill whose filter button fills the full surface, gave the refresh control an explicit circular hover disc, rescoped its reveal to keyboard-focus-only, and reconciled 06-UI-SPEC.md's three contradicting prose statements with the shipped geometry.**

## Performance

- **Duration:** 13 min
- **Started:** 2026-08-07T13:05:26+01:00 (previous plan's commit)
- **Completed:** 2026-08-07T13:18:19+01:00
- **Tasks:** 2
- **Files modified:** 4 (1 created, 3 modified)

## Accomplishments
- Chip wrapper now declares `h-11` directly and carries no vertical or left padding of its own; the filter `<button>` gained `self-stretch` plus its own left/right padding, so every pixel of the pill toggles the filter (closes the ~17px dead-band defect and the ~10px height mismatch against the adjacent `h-11` overflow trigger)
- Refresh `Button` override dropped from `size-11` to `size-8 rounded-full` — the explicit radius wins tailwind-merge's border-radius conflict group against `buttonVariants`' base `rounded-lg`, so the ghost hover fill is a disc, not a rounded square
- Refresh reveal rescoped from `group-focus-within:opacity-100` to `group-has-[:focus-visible]:opacity-100` — confirmed via production build + stylesheet grep that the arbitrary `:has()` variant actually compiles (not silently dropped); a mouse click on refresh no longer pins the icon visible after the pointer leaves, while Tab-to-chip keyboard reveal and the `source.syncing` force-visible branch both still work
- `source-chip-selected.test.ts` kept green: its wrapper-class extraction re-pointed at the div's opening tag (immune to future class reordering) and its D-03 reveal assertion updated to require the focus-visible-scoped literal instead of the retired focus-within one
- New `source-chip-pill.test.ts` — 8 assertions across 6 describe blocks (height parity, no dead bands, filter-button surface ownership, circular refresh + no 44px driver, reveal scoping, syncing-spinner survival) — manually proven to trip with its intended message when each of the three original defects is reintroduced one at a time, then confirmed `SourceChip.svelte` reverts to a clean `git diff` before committing
- `06-UI-SPEC.md` reconciled on all four flagged points: the refresh-reveal wording, a new "Chip geometry" constraint bullet (no vertical wrapper padding; any refresh-control size override must also declare a radius), the Tab-order parenthetical, and both touch-target-floor statements (the 44px floor now applies to the chip **body** as a whole, with the chip's own refresh control recorded as this phase's second documented desktop-only touch-target exception alongside the scrollbar tick)

## Task Commits

Each task was committed atomically:

1. **Task 1: Make the chip one 44px pill whose whole surface is clickable, with a circular refresh disc and a keyboard-only reveal** - `7c69cf2` (fix)
2. **Task 2: Recurrence guard for the chip's pill geometry, hit area, radius and reveal scoping** - `c020f60` (test)

**Plan metadata:** pending (this commit)

## Files Created/Modified
- `web/src/lib/components/SourceChip.svelte` - Wrapper declares `h-11` with no self-padding; filter button gained `self-stretch` + own padding; refresh `Button` override changed to `size-8 rounded-full` and `group-has-[:focus-visible]:opacity-100`
- `web/src/lib/components/source-chip-selected.test.ts` - Wrapper-class extraction re-pointed at the opening `<div>` tag; D-03 reveal assertion updated for the focus-visible-scoped reveal
- `.planning/phases/06-ui-scalable-source-surface/06-UI-SPEC.md` - Refresh-reveal wording, new chip-geometry constraint bullet, Tab-order parenthetical, and both touch-target-floor statements reconciled with the shipped code, dated to G-06-3b
- `web/src/lib/components/source-chip-pill.test.ts` - New recurrence guard for the chip's geometry, hit-area, refresh radius and reveal scoping (created)

## Decisions Made
- Kept `source-chip-pill.test.ts` as a file separate from `source-chip-selected.test.ts`, matching the plan's stated house pattern — the two guards fail for different reasons (colour vs. geometry/interaction) and should read as distinguishable incidents rather than one growing file
- The plan's primary Tailwind variant (`group-has-[:focus-visible]:opacity-100`) compiled on the first build attempt (confirmed by grepping `kernel/webui/build/` for the emitted `:has(:focus-visible)` rule), so the plan's named fallback (`group-has-focus-visible:opacity-100`) was not needed

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- G-06-3b is closed: the chip's geometry, refresh-hover shape, and reveal-scoping defects are fixed in code, guarded by `source-chip-pill.test.ts`, and reconciled in 06-UI-SPEC.md so a future spec re-read cannot reintroduce them.
- Remaining work before this phase's UAT can be signed off: a live `make dev` re-run of UAT Test 3 (height parity against the overflow trigger, full-surface clickability, circular hover disc, pointer-leave hiding after a refresh click, keyboard reveal, syncing-spinner force-visibility, and the selected-fill parity between inline and popover chips) — flagged as `human_judgment: true` on every coverage deliverable above since no component-mount test harness exists in this repo (`web/vite.config.ts`'s test environment is `node`).
- No blockers for subsequent phases.

---
*Phase: 06-ui-scalable-source-surface*
*Completed: 2026-08-07*
