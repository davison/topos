---
status: testing
phase: 08-whatsapp-conversations-managed-risk
source: [08-VERIFICATION.md]
started: 2026-08-10T20:45:00Z
updated: 2026-08-10T20:45:00Z
supersedes: previous 08-UAT.md cycle (19 tests, 18 passed, 1 issue → G-08-1, closed by plans 08-05/08-06/08-07 and re-verified after 08-08)
---

## Current Test

number: 1
name: Real-device re-test of the QR pairing flow (G-08-1 fix)
expected: |
  `make dev`, Add Source → New WhatsApp… with a display name and the seeded local
  path; scan the QR with a real phone and accept the pairing; observe the panel.
  The panel leaves the QR state within a few seconds of the phone reporting
  success ("Scan accepted — completing login…", then the linked confirmation,
  then Step 2); it must not sit on a code with a ticking countdown. Restart the
  kernel and confirm it reconnects with no second QR.
awaiting: user response

## Tests

### 1. Real-device re-test of the QR pairing flow (G-08-1 fix)
expected: The panel leaves the QR state within a few seconds of the phone reporting success ("Scan accepted — completing login…", then the linked confirmation, then Step 2); it must not sit on a code with a ticking countdown. Restart the kernel and confirm it reconnects with no second QR.
context: Carried forward from the prior verification. Requires a live WhatsApp account and a real kernel run — inherently outside the automated environment.
result: [pending]

### 2. Real-kernel confirmation of the CR-01 fix (in-flight teardown releases real resources)
expected: "`make dev`, open the SPA, start Add Source → New WhatsApp… with a display name and the seeded local path, click Next, then press Escape immediately — inside the window where the panel still shows its skeleton and no QR has appeared. The kernel log shows the link session cancelled within about a second, not reaped roughly five minutes later. Repeat five times in quick succession, then start one more link attempt and let it run: that final attempt must reach the QR state normally, not be rejected for exceeding the 4-slot concurrent link-session cap."
context: 08-08-PLAN.md Task 1's deferred <human-check> (`verification: backstop` truth). The e2e harness intercepts the link HTTP routes and never spawns a real topos-plugin-whatsapp subprocess, so no automated test can observe a real subprocess exiting or a suspended source instance resuming.
result: [pending]

## Summary

total: 2
passed: 0
issues: 0
pending: 2
skipped: 0
blocked: 0

## Gaps
