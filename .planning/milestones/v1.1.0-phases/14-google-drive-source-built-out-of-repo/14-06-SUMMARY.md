---
phase: 14-google-drive-source-built-out-of-repo
plan: 06
subsystem: web-ui
tags: [gap-closure, chip-row, overflow, accessibility, playwright]
gap_closure: true
gap_ids: [G-14-2]

# Dependency graph
requires:
  - "14-02's option-b accessible-description surface (preserved untouched)"
  - "Phase 6 UI-07 chip-row overflow design (visibleChipCount)"
provides:
  - "visibleChipCount minInlineChipWidth floor (6th defaulted parameter) — one chip guaranteed inline whenever the overflow budget can seat a minimum-width chip"
  - "MIN_INLINE_CHIP_PX=88 named constant in WebspaceHeader (live-measured chip furniture), row chips marked shrinkable"
  - "SourceChip opt-in shrinkable prop — forced chip truncates its name instead of clipping reserved controls"
  - "web/e2e/specs/14-chip-health-narrow-viewport.spec.ts — the 375px browser gate (G-14-2's UAT item as a spec)"
affects: [phase-14-verification, 09-1-header-touch.spec, 09-1-mobile-takeover.spec]

# Actuals (#2632)
actuals:
  tokens: inline-interactive
  tasks: 4
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Budget-gated floor on a pure layout function: a defaulted parameter that only ever raises a zero, gated on the already-computed overflow budget, so the default preserves the prior contract byte-for-byte and existing tests hold unmodified"
    - "Live width probe before pinning a layout constant: a temporary Playwright spec dumped the real budget numbers at 375px (row 327, trailing 87, add-source 44, trigger 73 → overflow budget 99), correcting the plan's 112px estimate to the measured 88px furniture width"

key-files:
  created:
    - web/e2e/specs/14-chip-health-narrow-viewport.spec.ts
  modified:
    - web/src/lib/format.ts
    - web/src/lib/components/WebspaceHeader.svelte
    - web/src/lib/components/SourceChip.svelte
    - web/src/lib/components/sources.test.ts
    - web/e2e/specs/09-1-header-touch.spec.ts
    - web/e2e/specs/09-1-mobile-takeover.spec.ts
    - docs/testing.md

key-decisions:
  - "MIN_INLINE_CHIP_PX corrected from the plan's 112 estimate to 88 by live measurement (the plan's own guardrail: 'adjust only if a live measurement at 375px contradicts it'). At 375px the overflow budget is 99px — 112 never fired; 88 is the chip's exact non-text furniture (dot 8 + two 6px gaps + icon 14 + button padding 16 + wrapper padding 4 + actions trigger 32 + borders 2), so a floor-width chip still shows dot, icon, ellipsized name and menu — a functional hover target, not a sliver."
  - "09-1-header-touch required NO assertion changes — case 3 still passes because Clear filters widens the reservation enough to drop the overflow budget (4px) below the minimum, so its single chip legitimately relegates: the budget gate working as designed. Only its module-header and case-6 comments were updated to state the new one-chip-guaranteed rule."
  - "09-1-mobile-takeover case 6 repointed from Mock 01 (now floor-held inline) to Mock 02's popover clone, preserving the case's real subject (popover is the only reachable path while the header is concealed)."

# Coverage (#1602)
coverage:
  - id: D1
    description: "At 375px with a long-named external source the chip row renders at least one real chip and its health popover opens, wraps, and stays inside the viewport"
    verification: "web/e2e/specs/14-chip-health-narrow-viewport.spec.ts cases 1-3 (passing)"
    status: auto
  - id: D2
    description: "Exactly one chip is forced (six instances → one chip + '+5'), desktop widths unchanged, option-b surface intact"
    verification: "cases 4-6 (passing); grep gates: aria-describedby={chipDescId}=1, sr-only=1, source-chip-tooltip.test.ts diff empty"
    status: auto
  - id: D3
    description: "The whole suite holds under the new guarantee at zero retries"
    verification: "make e2e — 145 passed, 5 skipped (gdrive rehearsal skip-path), retries 0; npm test 1090/1090; svelte-check 0 errors"
    status: auto
  - id: D4
    description: "The operator's own failed UAT Test 2 re-run passes at a narrow viewport"
    verification: "Task 4 checkpoint:human-verify — operator replied 'approved' (2026-08-21)"
    status: human_verified

## What was built

The G-14-2 fix, exactly as the plan's fix_direction_decision chose: a
budget-gated floor of one inline chip. `visibleChipCount` gained a sixth,
defaulted `minInlineChipWidth` parameter that only ever raises a zero —
when at least one chip exists and the overflow budget can seat the
minimum — so the default of 0 preserves the previous contract exactly
and all eleven pre-existing unit cases pass unmodified (six new cases
added). `WebspaceHeader` passes `MIN_INLINE_CHIP_PX = 88` and marks only
the visible row's chips `shrinkable`; the overflow-popover clones
(column layout) and measurement clones (natural-width reporters) are
deliberately not marked. `SourceChip`'s new opt-in prop swaps
`shrink-0` for `min-w-0 shrink` as mutually exclusive alternatives, the
filter button gains `min-w-0` so the truncating name span can give way,
and the actions trigger gains an explicit `shrink-0` so the fixed
control is never squashed.

The new Playwright spec pins the UAT item at 375×812: chip in row,
popover legible (inside viewport, ≤320px, wrapped), no affordance traded
away, exactly one chip forced, option-b intact, desktop parity at 1280.

## Deviations

- **Rule-anticipated constant correction (not a deviation from intent):**
  the plan's 112px starting value never fired at 375px (measured
  overflow budget: 99px). Corrected to the measured 88px furniture
  width per the plan's own instruction; the arithmetic and probe numbers
  are recorded in the WebspaceHeader comment.
- **Rule 1 (spec-bug fixes while verifying):** the RED spec initially
  scoped locators through `getByRole('banner')` — the header is nested
  inside `<main>` and exposes no banner role — and case 6 used
  non-waiting `.count()`. Both fixed before the RED commit; desktop
  parity then passed pre-fix as intended.
- **Latent pre-existing issue surfaced (not introduced here):** a
  non-first chip clone inside the overflow popover cannot be clicked
  under hit-testing — its own zero-delay hoverable tooltip overlays its
  trigger and intercepts pointers. The old case 6 clicked the first
  clone and never saw it. Worked around with the case's own documented
  dispatchEvent technique and captured as todo
  `2026-08-21-popover-clone-tooltip-intercepts-clicks.md`.
- **Predicted breakage that did not break:** 09-1-header-touch's
  assertions all held under the floor (the plan predicted case 3 would
  break; it does not, because the widened Clear-filters reservation
  legitimately drops the budget below the minimum). Comments updated,
  assertions untouched — recorded here rather than edited to look busy.

## Self-Check: PASSED

- npm --prefix web run test: 1090/1090 (58 files)
- npm --prefix web run check: 0 errors
- npm --prefix web run check:e2e: 0 errors
- make e2e: 145 passed, 5 skipped, 0 failed, retries 0
- make docs-check: pass
- Human verify (UAT Test 2 re-run at narrow viewport): approved
