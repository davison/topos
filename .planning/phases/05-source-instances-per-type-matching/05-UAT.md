---
status: complete
phase: 05-source-instances-per-type-matching
source: [05-VERIFICATION.md]
started: 2026-08-06T14:24:37Z
updated: 2026-08-06T16:30:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Two-instance source chips and health UI in the browser
expected: With two instances of the same plugin type configured under distinct display names, the stream, source filter, and health UI render them as separate named sources (ROADMAP criterion 1). 05-05 substituted API/curl checks for the plan's own human-check; tracked open in .planning/WINDOWS.md entry #4. Requires restarting the stale `make dev` session (pre-Phase-5 binary against schema-v2 index).
result: pass

### 2. Rendition pixel parity after the kernel-owned rendition move
expected: Opening a real email, markdown note, and Signal chat item in the detail pane renders visually identical to pre-Phase-5 — CSS rule carry-forward is test-proven (05-04 D6), actual browser rendering is not.
result: pass

### 3. Interrupted schema-rebuild recoverability
expected: Killing the kernel mid index schema rebuild and restarting recovers cleanly (rebuild restarts or completes; no corrupted index, no orphaned rows). Structural guarantee from 05-01 D4, never exercised by a kill-mid-rebuild test.
result: pass

### 4. Stale-binary handshake fail-fast
expected: Launching a pre-Phase-5 plugin binary against the new kernel fails cleanly at the go-plugin handshake (ProtocolVersion 1 vs 2) with a clear error, not confusingly at first Match. Contract-pinned in tests (05-02 D4); no live stale-binary launch was exercised.
result: pass

### 5. Subprocess teardown on match-vocabulary validation failure
expected: When startup validation fails by name against a plugin's declared vocabulary (05-03), plugin subprocesses are torn down cleanly — no orphaned plugin processes after the kernel exits with the validation error. Verified by inspection only (05-03 D6).
result: skipped
reason: "User chose not to exercise manually (offered as optional); teardown mirrors existing error paths, verified by code inspection during execution (05-03 D6)."

## Summary

total: 5
passed: 4
issues: 0
pending: 0
skipped: 1
blocked: 0

## Gaps
