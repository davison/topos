---
status: complete
phase: 08-whatsapp-conversations-managed-risk
source: [08-VERIFICATION.md]
started: 2026-08-11T00:35:00Z
updated: 2026-08-10T23:44:03Z
supersedes: previous 08-UAT.md cycle (3 tests, 2 passed, 1 issue → G-08-3, diagnosed in .planning/debug/whatsapp-grpc-closing-fails-webspace.md, closed in code by plans 08-09/08-10)
---

## Current Test

[testing complete]

## Tests

### 1. Real-device re-test of G-08-3's fix — webspace stream loads after WhatsApp pairing
expected: Pair or re-link a real WhatsApp account via `make dev`, then open its webspace immediately after; the stream loads normally (no "Couldn't load this webspace" / grpc-closing error). An unavailable whatsapp source degrades per-source (StreamSyncDegraded / chip health error), previously captured messages stay browsable, and an unrelated failing source never affects this webspace.
context: The one item automation cannot cover — the hermetic Go tests drive a mock plugin subprocess and the Playwright specs script the HTTP routes, but no test drives a real topos-plugin-whatsapp subprocess through an actual pair/re-link/degrade cycle. Both 08-09-PLAN.md and 08-10-PLAN.md explicitly defer G-08-3's real-device confirmation to this re-run.
result: issue
reported: "after login and opening the webspace: A source couldn't sync / Nothing to show here yet. Your other sources are unaffected — check the source chips above, then retry. / whatsapp: match against source \"whatsapp\": rpc error: code = Unavailable desc = whatsapp: Not linked — pair this device with WhatsApp to start syncing. Use this source's chip menu (\"Re-link…\") or run this plugin binary's -link flag."
severity: major
note: "G-08-3's presentation half is CONFIRMED FIXED — the webspace loaded with the per-source StreamSyncDegraded notice instead of the old full-page fake outage, and the copy correctly says other sources are unaffected. The new issue is upstream: immediately after a successful real-device pairing, the whatsapp plugin still reports Not linked, so the fresh session was not picked up by the running instance."

## Summary

total: 1
passed: 0
issues: 1
pending: 0
skipped: 0
blocked: 0

## Gaps

- gap_id: G-08-4   # next free id in this phase's gap history — G-08-1/G-08-3 are already claimed by executed gap-closure plans' frontmatter, so reusing the test-number formula (G-08-1) would make reconciliation instantly mark this new gap resolved
  truth: "Immediately after a successful real-device WhatsApp pairing, the running whatsapp source instance uses the fresh session: the source syncs (or reports a transient connecting state) rather than staying pinned on the pre-pairing 'Not linked' health error"
  status: failed
  reason: "User reported: after login and opening the webspace: A source couldn't sync / Nothing to show here yet. Your other sources are unaffected — check the source chips above, then retry. / whatsapp: match against source \"whatsapp\": rpc error: code = Unavailable desc = whatsapp: Not linked — pair this device with WhatsApp to start syncing. Use this source's chip menu (\"Re-link…\") or run this plugin binary's -link flag."
  severity: major
  test: 1
  artifacts: []  # Filled by diagnosis
  missing: []    # Filled by diagnosis
  context: "G-08-3's presentation half is confirmed fixed (per-source StreamSyncDegraded, no fake outage). This gap is the kernel/plugin seam: the freshly paired session is not picked up by the running instance."
