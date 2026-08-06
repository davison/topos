---
status: testing
phase: 05-source-instances-per-type-matching
source: [05-VERIFICATION.md]
started: 2026-08-06T14:24:37Z
updated: 2026-08-06T14:24:37Z
---

## Current Test

number: 1
name: Two-instance source chips and health UI in the browser
expected: |
  With two instances of the same plugin type configured (distinct display names),
  the stream, source filter chips, and health UI show them as separate named
  sources. Restart your `make dev` session first — the running pre-Phase-5
  process on port 7777 errors against the rebuilt schema-v2 index until restarted.
awaiting: user response

## Tests

### 1. Two-instance source chips and health UI in the browser
expected: With two instances of the same plugin type configured under distinct display names, the stream, source filter, and health UI render them as separate named sources (ROADMAP criterion 1). 05-05 substituted API/curl checks for the plan's own human-check; tracked open in .planning/WINDOWS.md entry #4. Requires restarting the stale `make dev` session (pre-Phase-5 binary against schema-v2 index).
result: [pending]

### 2. Rendition pixel parity after the kernel-owned rendition move
expected: Opening a real email, markdown note, and Signal chat item in the detail pane renders visually identical to pre-Phase-5 — CSS rule carry-forward is test-proven (05-04 D6), actual browser rendering is not.
result: [pending]

### 3. Interrupted schema-rebuild recoverability
expected: Killing the kernel mid index schema rebuild and restarting recovers cleanly (rebuild restarts or completes; no corrupted index, no orphaned rows). Structural guarantee from 05-01 D4, never exercised by a kill-mid-rebuild test.
result: [pending]

### 4. Stale-binary handshake fail-fast
expected: Launching a pre-Phase-5 plugin binary against the new kernel fails cleanly at the go-plugin handshake (ProtocolVersion 1 vs 2) with a clear error, not confusingly at first Match. Contract-pinned in tests (05-02 D4); no live stale-binary launch was exercised.
result: [pending]

### 5. Subprocess teardown on match-vocabulary validation failure
expected: When startup validation fails by name against a plugin's declared vocabulary (05-03), plugin subprocesses are torn down cleanly — no orphaned plugin processes after the kernel exits with the validation error. Verified by inspection only (05-03 D6).
result: [pending]

## Summary

total: 5
passed: 0
issues: 0
pending: 5
skipped: 0
blocked: 0

## Gaps
