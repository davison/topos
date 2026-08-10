---
status: testing
phase: 08-whatsapp-conversations-managed-risk
source: [08-VERIFICATION.md]
started: 2026-08-11T00:35:00Z
updated: 2026-08-11T00:35:00Z
supersedes: previous 08-UAT.md cycle (3 tests, 2 passed, 1 issue → G-08-3, diagnosed in .planning/debug/whatsapp-grpc-closing-fails-webspace.md, closed in code by plans 08-09/08-10)
---

## Current Test

number: 1
name: Real-device re-test of G-08-3's fix — webspace stream loads after WhatsApp pairing
expected: |
  Pair or re-link a real WhatsApp account via `make dev`, then open its webspace
  immediately after (including mid-restart of the plugin connection): the stream
  loads normally — no "Couldn't load this webspace" / grpc client-connection-closing
  error. If the whatsapp source is unavailable it degrades per-source (chip-level
  health error / StreamSyncDegraded notice) while the rest of the webspace keeps
  working, and an unrelated failing source never affects this webspace.
awaiting: user response

## Tests

### 1. Real-device re-test of G-08-3's fix — webspace stream loads after WhatsApp pairing
expected: Pair or re-link a real WhatsApp account via `make dev`, then open its webspace immediately after; the stream loads normally (no "Couldn't load this webspace" / grpc-closing error). An unavailable whatsapp source degrades per-source (StreamSyncDegraded / chip health error), previously captured messages stay browsable, and an unrelated failing source never affects this webspace.
context: The one item automation cannot cover — the hermetic Go tests drive a mock plugin subprocess and the Playwright specs script the HTTP routes, but no test drives a real topos-plugin-whatsapp subprocess through an actual pair/re-link/degrade cycle. Both 08-09-PLAN.md and 08-10-PLAN.md explicitly defer G-08-3's real-device confirmation to this re-run.
result: [pending]

## Summary

total: 1
passed: 0
issues: 0
pending: 1
skipped: 0
blocked: 0

## Gaps
