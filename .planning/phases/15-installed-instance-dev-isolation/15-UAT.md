---
status: testing
phase: 15-installed-instance-dev-isolation
source: [15-VERIFICATION.md]
started: 2026-08-19T00:55:00Z
updated: 2026-08-19T00:55:00Z
---

## Current Test

number: 1
name: Signal on the installed instance — install-signal plus the one-time consent flow
expected: |
  With the installed instance replaced by `make install`, run
  `make install-signal`, restart the installed kernel, and add the
  Signal source through the app's untrusted-add consent flow. The
  source syncs and the chip shows the untrusted badge rather than a
  launch failure.
awaiting: user response

## Tests

### 1. Signal on the installed instance — install-signal plus the one-time consent flow
expected: With the installed instance replaced by `make install`, run `make install-signal`, restart the installed kernel, and add the Signal source through the app's untrusted-add consent flow. The source syncs and the chip shows the untrusted badge rather than a launch failure.
result: [pending]

### 2. The roadmap's end-to-end migration and simultaneity proof
expected: |
  Follow the 5-step UAT on the real instance: (1) note the running
  checkout instance's webspace list, one item's mark, and each source's
  health state; (2) stop it, `make install`, start `topos` from PATH;
  (3) confirm it serves on 7777 with identical webspaces, mark, and
  source health — including WhatsApp still linked; (4) run `make dev`
  from the checkout alongside and confirm the dev instance comes up on
  7778 with neither instance showing the other's data; (5) `make
  uninstall` and confirm the config, index, and marks all survive.
  All 5 steps succeed with no data loss and no visible clash.
result: [pending]

## Summary

total: 2
passed: 0
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps
