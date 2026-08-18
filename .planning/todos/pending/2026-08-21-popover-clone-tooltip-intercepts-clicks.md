---
created: 2026-08-21T00:00:00.000Z
title: Non-first chip clones in the overflow popover have clicks intercepted by their own tooltip
area: web
severity: minor
files:
  - web/src/lib/components/SourceChip.svelte
  - web/src/lib/components/ui/tooltip/tooltip-content.svelte
  - web/e2e/specs/09-1-mobile-takeover.spec.ts
---

## Problem

Discovered while repointing `09-1-mobile-takeover.spec.ts` case 6 during
14-06 (G-14-2 gap closure): a chip clone rendered in the "+N more
sources" popover column — any row EXCEPT the first — cannot be clicked
under Playwright's hit-testing. The clone's own zero-delay health
tooltip opens positioned over its trigger inside the popover stack, and
bits-ui tooltip content is hoverable (no `pointer-events: none`), so the
tooltip's floating wrapper persistently intercepts pointer events at
every point of the button. The first row never hits this because its
tooltip escapes upward past the popover's top edge into free page space.

This is latent and pre-existing — not introduced by 14-06's inline-chip
floor. The old case 6 clicked the FIRST clone (Mock 01) and passed; the
floor forced Mock 01 inline, the case was repointed at Mock 02, and the
overlap surfaced. A real mouse user hovering a non-first clone and
clicking may have the click swallowed by the tooltip the same way.

## Solution

Investigate why floating-ui positions a non-first clone's side-top
tooltip over its own trigger inside the popover (collision/flip inside
the portal stack?), then either fix the positioning or make the tooltip
content non-hoverable (`pointer-events: none`, matching shadcn-react's
default) so an overlapping tooltip can never eat clicks. When fixed,
restore a real `.click()` in 09-1-mobile-takeover case 6 (currently
`dispatchEvent('click')` with a comment naming this todo) and add a
regression assertion that a non-first popover clone is genuinely
clickable.
