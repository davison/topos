---
status: complete
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
  artifacts: []  # Filled by diagnosis
  missing: []    # Filled by diagnosis
