---
created: 2026-08-14T22:40:00.000Z
title: Suppress native browser tooltips that duplicate/cover source-chip popovers
area: web
severity: minor
files:
  - web/src/lib/components/SourceChip.svelte
---

## Problem

The CSS-based popovers on source chips are duplicated by the browser's own
native hover tooltips, rendered from `title` (and possibly `alt`) attributes
on the chip elements. Depending on cursor position, the native tooltip either
repeats the popover's content (e.g. last sync time) or just duplicates the
source instance name — and it often renders *on top of* the app's popover,
covering it. Raised by the user during Phase 13 UAT (13-04 checkpoint,
2026-08-14): "Both need to be suppressed ideally."

## Solution

Remove the `title`/`alt` attributes from chip elements that already have a
richer app popover (the popover supersedes them), or move their text to
`aria-label` so accessibility is preserved without triggering the native
tooltip. Audit other components using the same chip/popover pattern for the
same duplication. Add/extend a component test asserting the popover-bearing
elements carry no `title` attribute.

Note: `web/src/lib/components/SourceChip.svelte` is already in plan 13-06's
declared file scope (trust-state chips + `source-chip-tooltip.test.ts`) — if
13-06 executes before this todo is picked up, it may be cheapest to fold the
suppression into that work; otherwise this stands alone as a small fix.
