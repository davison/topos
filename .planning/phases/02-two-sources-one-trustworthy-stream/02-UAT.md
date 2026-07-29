---
status: testing
phase: 02-two-sources-one-trustworthy-stream
source: [02-VERIFICATION.md]
started: 2026-07-29T10:50:00Z
updated: 2026-07-29T10:50:00Z
---

## Current Test

number: 1
name: Visual confirmation of health chips, filter chips, and staleness states in a real browser
expected: |
  With both sources configured, deliberately stop the SilverBullet service and reload
  the webspace in a browser. Confirm: two health chips render with correct colors
  (paperless green, SilverBullet red); hovering SilverBullet's chip shows the
  unreachable tooltip with a relative time and the full untruncated error; SilverBullet
  rows carry an amber stale marker while paperless rows are visually unchanged; opening
  a SilverBullet item shows its cached title/date/labels/open-in-source link with the
  unreachable alert layered over the cached preview (never a blank pane); clicking the
  source filter chips narrows/restores the list and the URL gains/loses the `source`
  query parameter, surviving a reload. Matches 02-UI-SPEC.md's Copywriting Contract and
  Design System exactly (colors, wording, layout).
awaiting: user response

## Tests

### 1. Visual confirmation of health chips, filter chips, and staleness states in a real browser
expected: Two health chips with correct colors; unreachable tooltip with relative time and full error; amber stale markers on SilverBullet rows only; cached detail pane with unreachable alert (never blank); source filter chips update the list and `source` query parameter, surviving reload — all matching 02-UI-SPEC.md.
result: [pending]

### 2. PLUG-05 third-party self-sufficiency from a genuinely fresh/isolated context
expected: A truly isolated agent (real dispatched subagent, or human with no prior exposure to this repository) builds a SourcePlugin using only proto/webspaces/v1/plugin.proto, docs/plugin-contract.md, the sdk module, and plugins/mock — no access to plugins/paperless or plugins/silverbullet — producing a clean build with zero or few gaps, corroborating 02-04's in-session two-gap-then-zero-gap result.
result: [pending]

## Summary

total: 2
passed: 0
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps
