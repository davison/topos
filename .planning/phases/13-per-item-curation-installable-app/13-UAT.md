---
status: testing
phase: 13-per-item-curation-installable-app
source: [13-VERIFICATION.md]
started: 2026-08-15T01:15:00Z
updated: 2026-08-15T01:15:00Z
---

## Current Test

number: 1
name: Real human timing feasibility of the undo-across-webspace-switch window
expected: |
  With a real mouse/trackpad and ordinary (non-speedrun) reaction time: exclude an
  item from a webspace, use the WebspaceSwitcher to switch to a different webspace,
  and click the still-visible "Undo" toast button — all within the toast's real
  5000ms lifetime. A person can comfortably complete this sequence inside the
  window during normal use; the Undo button is still visible and clickable when
  reached (it should not typically require rushing).
awaiting: user response

## Tests

### 1. Real human timing feasibility of the undo-across-webspace-switch window
expected: A human can comfortably complete exclude → WebspaceSwitcher switch → click Undo inside the toast's real 5000ms window with normal reaction time; the Undo button is still visible and clickable when reached, and the undo correctly reverses the exclusion in the ORIGINAL webspace (not the one navigated to).
result: [pending]

## Summary

total: 1
passed: 0
issues: 0
pending: 1
skipped: 0
blocked: 0

## Gaps
