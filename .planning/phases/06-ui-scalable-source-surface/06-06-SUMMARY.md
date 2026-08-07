---
phase: 06-ui-scalable-source-surface
plan: 06
subsystem: ui
tags: [svelte, tailwind, design-tokens, source-chip, source-scan-test]

requires:
  - phase: 06-02
    provides: the merged SourceChip.svelte component (D-01/D-02/D-03/D-04 wiring, ARIA and behaviour) whose rendered contrast this plan fixes
  - phase: 06-04
    provides: the ResizeObserver-driven overflow measurement that depends on WebspaceHeader.svelte's chip row staying overflow-hidden
provides:
  - A solid --primary fill for the selected source chip (border-primary bg-primary), replacing the invisible neutral bg-accent/10 ring-2 ring-accent treatment
  - Display name, health-dot ring and refresh icon/hover surface all re-toned to --primary-foreground when selected
  - A standing incident note on app.css's --accent trap comment, naming G-06-3
  - 06-UI-SPEC.md's chip anatomy bullet and Color table brought into agreement with each other and with the shipped fill treatment
  - source-chip-selected.test.ts — the component's first automated guard, proving the treatment resolves through --primary (never --accent), that every child re-tones, that D-01/D-02/D-03/D-04 behaviour is untouched, and that WebspaceHeader.svelte's chip row still clips
affects: [phase-07-webspace-builder-ui, any-future-source-chip-or-header-work]

actuals:
  tokens: 4000
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Selected-state fills paint inside the border box (bg-primary), sidestepping outset box-shadow ring clipping under an overflow-hidden ancestor entirely, rather than fighting the clip"
    - "Comment-stripped source-scan guard (extractBetween scoping, house pattern from pane-layout.test.ts/scrollbar-theme.test.ts/search-emphasis.test.ts) as the recurrence-proof instrument for a component-mount-free (environment: 'node') test runner"

key-files:
  created:
    - web/src/lib/components/source-chip-selected.test.ts
  modified:
    - web/src/lib/components/SourceChip.svelte
    - web/src/app.css
    - .planning/phases/06-ui-scalable-source-surface/06-UI-SPEC.md

key-decisions:
  - "Chose a solid bg-primary fill over a corrected ring — a fill paints inside the border box and needs no change to WebspaceHeader.svelte's load-bearing overflow-hidden, whereas a ring (even correctly colored) would still be clipped top/bottom by that same ancestor"
  - "Re-toned the health dot with an inset ring (ring-1 ring-primary-foreground) rather than remapping any of its four tone colors — the dot's color must keep encoding health, only its separation from the new fill needed help"
  - "Extended app.css's existing --accent trap comment in place, naming G-06-3 as the confirmed incident, rather than adding a second comment elsewhere"

patterns-established:
  - "A selected/active affordance on this palette must resolve through --primary/--ring, never the literally-named --accent token — now enforced by both a standing app.css comment and an automated source-scan guard"

requirements-completed: [UI-07]

coverage:
  - id: D1
    description: "A selected source chip renders a solid, high-contrast --primary fill instead of the neutral bg-accent/10 ring-2 ring-accent treatment"
    requirement: "UI-07"
    verification:
      - kind: unit
        ref: "web/src/lib/components/source-chip-selected.test.ts#the selected treatment resolves through the real accent blue"
        status: pass
      - kind: manual_procedural
        ref: "06-06-PLAN.md verify human-check: run make dev, select a chip in the browser, confirm an obvious contrasting blue fill"
        status: unknown
    human_judgment: true
    rationale: "The source-scan guard proves the class expression resolves through --primary and never --accent, but actual pixel-level contrast in a running browser needs a human or a screenshot-driving UAT pass — no component-mount harness exists in this test runner (environment: 'node')."
  - id: D2
    description: "The display name, health dot and refresh icon all stay legible against the fill (re-tone to --primary-foreground / an inset ring)"
    requirement: "UI-07"
    verification:
      - kind: unit
        ref: "web/src/lib/components/source-chip-selected.test.ts#every child re-tones when selected, not only the wrapper"
        status: pass
    human_judgment: false
  - id: D3
    description: "The selected treatment survives the chip row's overflow-hidden clipping and reads identically inline and inside the overflow popover"
    requirement: "UI-07"
    verification:
      - kind: unit
        ref: "web/src/lib/components/source-chip-selected.test.ts#the chip row's overflow clipping (load-bearing for 06-04) is untouched"
        status: pass
      - kind: manual_procedural
        ref: "06-06-PLAN.md verify human-check: narrow the window until chips overflow, select a chip inside the popover, confirm it reads identically to an inline selected chip"
        status: unknown
    human_judgment: true
    rationale: "A fill paints inside the border box so it structurally cannot be clipped the way the retired ring was — the guard proves the row still clips and the wrapper resolves through --primary — but confirming the popover chip visually matches the inline chip needs a live render."
  - id: D4
    description: "D-01/D-02/D-03/D-04 behaviour (aria-pressed, filter toggle, stopPropagation, hover/focus-within reveal, syncing spinner) is unchanged by the styling fix"
    requirement: "UI-07"
    verification:
      - kind: unit
        ref: "web/src/lib/components/source-chip-selected.test.ts#D-01/D-02/D-03/D-04 behaviour is untouched by the styling fix"
        status: pass
      - kind: unit
        ref: "cd web && npx vitest run (full 215-test suite, all green)"
        status: pass
    human_judgment: false
  - id: D5
    description: "06-UI-SPEC.md's chip anatomy prose and Color table agree with each other and with the shipped code"
    requirement: "UI-07"
    verification:
      - kind: other
        ref: "grep -q 'G-06-3' .planning/phases/06-ui-scalable-source-surface/06-UI-SPEC.md && grep -q 'fill' .planning/phases/06-ui-scalable-source-surface/06-UI-SPEC.md"
        status: pass
    human_judgment: false
  - id: D6
    description: "source-chip-selected.test.ts fails if the neutral token returns or a child stops re-toning"
    verification:
      - kind: manual_procedural
        ref: "Manually reverted the wrapper's selected branch to the retired 'bg-accent/10 ring-2 ring-accent' during execution; confirmed 2 of 12 guard assertions failed with the expected messages, then restored"
        status: pass
    human_judgment: false

duration: ~15min
completed: 2026-08-07
status: complete
---

# Phase 06 Plan 06: Restore the Selected Source Chip's Visual Affordance Summary

**Closed gap G-06-3 by replacing the invisible neutral `bg-accent/10 ring-2 ring-accent` selected-chip treatment with a solid `--primary` fill and re-toned children, adding the component's first automated guard.**

## Performance

- **Duration:** ~15 min
- **Completed:** 2026-08-07
- **Tasks:** 2
- **Files modified:** 4 (3 source/doc, 1 new test)

## Accomplishments

- `SourceChip.svelte`'s selected wrapper now resolves through `border-primary bg-primary` — the real accent blue in this palette — instead of the neutral `--accent` token that is byte-identical to `--border`.
- A fill paints inside the border box, sidestepping the outset-box-shadow clipping the retired ring suffered under `WebspaceHeader.svelte`'s load-bearing `overflow-hidden` chip row — no change to that row was needed or made.
- Display name span, health-dot span and refresh `Button` all re-tone to `text-primary-foreground` (plus an inset `ring-1 ring-primary-foreground` on the dot, and a blue-legible hover surface on the refresh button) when `selected` is true — verified via `tailwind-merge`'s own resolution that `bg-card`/`border-border` correctly lose the background-color conflict group to the selected branch.
- `app.css`'s existing `--accent` trap comment extended in place to record G-06-3 as the confirmed incident.
- `06-UI-SPEC.md`'s chip anatomy bullet and Color table both rewritten to describe "fill" (not "ring") and now agree with each other and with the shipped code, with a dated note naming G-06-3.
- New `source-chip-selected.test.ts`: a comment-stripped source-scan guard (12 assertions across 5 groups) proving the selected treatment resolves through `--primary`, never `--accent`; that the display name, health dot and refresh button all re-tone (not only the wrapper — the exact shape of the original defect); that `aria-pressed`, `stopPropagation`, the hover/focus-within reveal and the syncing-spinner force-visible behaviour are all untouched; and that `WebspaceHeader.svelte`'s chip row still carries `overflow-hidden`. Manually verified to fail loudly (2 of 12 assertions) when reverted to the retired neutral-token treatment.

## Task Commits

1. **Task 1: Fill the selected chip with the palette's real accent blue and re-tone every child** - `26f5d03` (fix)
2. **Task 2: First automated guard for the chip's selected treatment** - `a54407f` (test)

_No separate plan-metadata commit beyond the final STATE/ROADMAP commit below (`commit_docs` handled by the SDK's final-commit step)._

## Files Created/Modified

- `web/src/lib/components/SourceChip.svelte` - Selected wrapper fills with `border-primary bg-primary`; display name, health dot and refresh button all re-tone to `--primary-foreground` when selected
- `web/src/app.css` - Extended the standing `--accent` trap comment with the G-06-3 incident note (no declaration changed)
- `.planning/phases/06-ui-scalable-source-surface/06-UI-SPEC.md` - Chip anatomy bullet and Color table both corrected to "fill" and dated, sourced to G-06-3
- `web/src/lib/components/source-chip-selected.test.ts` - New comment-stripped source-scan recurrence guard (created)

## Decisions Made

- Chose a solid `bg-primary` fill over a color-corrected ring — a fill paints inside the border box and needs no change to the chip row's load-bearing `overflow-hidden`, whereas even a correctly-colored ring would still be clipped top/bottom by that same ancestor (per the diagnosis in `.planning/debug/chip-selected-state-visibility.md`).
- Re-toned the health dot with an inset ring rather than remapping any of its four tone colors — the dot's color must keep encoding health; only its separation from the new fill needed help.
- Extended the existing `--accent` trap comment in `app.css` in place (rather than adding a second, separate comment) since it already anticipated exactly this confusion.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None — the debug session's root-cause diagnosis (`.planning/debug/chip-selected-state-visibility.md`) correctly identified all three conjoint causes (colour, clipping, spec contradiction) going in, so no new investigation was needed during execution.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- G-06-3 is closed: the selected chip's visual affordance is restored with an automated recurrence guard in place, and the spec contradiction that could have reintroduced the bug is resolved.
- Remaining verification: the plan's `<verify><human-check>` (live browser confirmation of the fill's contrast and popover parity) is recorded as `human_judgment: true` coverage items (D1, D3) for the phase's end-of-phase UAT pass — not run in this execution (no browser-driving tool available in this environment).
- No blockers for Phase 7 (Webspace Builder UI) or any future work touching `SourceChip.svelte`/`WebspaceHeader.svelte`.

---
*Phase: 06-ui-scalable-source-surface*
*Completed: 2026-08-07*

## Self-Check: PASSED

All 5 claimed files found on disk; both claimed commit hashes (`26f5d03`, `a54407f`) found in git history.
