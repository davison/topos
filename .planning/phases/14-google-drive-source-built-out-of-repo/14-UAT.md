---
status: diagnosed
phase: 14-google-drive-source-built-out-of-repo
source: [14-VERIFICATION.md]
started: 2026-08-18T15:05:00.000Z
updated: 2026-08-18T16:20:00.000Z
---

## Current Test

[testing complete]

## Tests

### 1. Touch-only accessibility regression — accept or reject
expected: On a viewport below 768px with touch input and no screen reader, chip health detail is no longer reachable by long-press (native tooltip removed by design). Human decision required: accept as a known, documented regression, or file a follow-up item to restore a touch-reachable path.
result: pass — operator explicitly ACCEPTED the regression as-is (2026-08-18). Recorded here in lieu of the unavailable WINDOWS.md ledger tooling.

### 2. Long health-message legibility on narrow viewports
expected: With a chip whose health sentence is long (e.g. an external untrusted plugin with a stale sync time), the popover/described text remains legible and usefully laid out on a narrow (mobile-width) viewport. This is 14-UI-SPEC's single unconfirmed item, declared `verification: backstop` in 14-04-PLAN.md — it was correctly never claimed VERIFIED and needs one human visual check.
result: issue
reported: "the popover is not shown at all unless the viewport is wide enough to accommodate it"
severity: major

## Summary

total: 2
passed: 1
issues: 1
pending: 0
skipped: 0
blocked: 0

## Gaps

- gap_id: G-14-2
  truth: "With a long chip health sentence, the popover remains legible and usefully laid out on a narrow (mobile-width) viewport"
  status: failed
  reason: "User reported: the popover is not shown at all unless the viewport is wide enough to accommodate it"
  severity: major
  test: 2
  root_cause: "Not a tooltip-rendering bug: WebspaceHeader's chip-row overflow relegation (visibleChipCount, floor 0) removes the chip — the popover's only hover trigger — from the row at <=~400px, leaving zero chips and only the '+N' pill; the tooltip itself renders correctly at every width whenever a trigger exists (proven 375px, Chromium+Firefox). Pre-existing Phase 6 overflow design, neither introduced nor exposed by 14-02; the clone chip inside the '+N' popover already shows the tooltip correctly but is undiscoverable."
  artifacts:
    - path: "web/src/lib/format.ts"
      issue: "visibleChipCount floors at 0 — the row may render no chips while health detail has no other visible surface"
    - path: "web/src/lib/components/WebspaceHeader.svelte"
      issue: "overflow relegation leaves '+N' pill as sole affordance; nothing signals relegated chips' health lives inside it"
    - path: "web/src/lib/components/SourceChip.svelte"
      issue: "health sentence is hover/focus-tooltip-only (plus AT-only aria-describedby), so trigger absence means total absence"
  missing:
    - "A reachable/discoverable narrow-viewport path to chip health detail (e.g. keep >=1 harder-truncated chip in the row, or make '+N' visibly carry relegated chips' health surface)"
    - "A Playwright spec pinning 'health detail reachable at 375px' (UAT-becomes-spec convention)"
  debug_session: .planning/debug/popover-hidden-narrow-viewport.md
