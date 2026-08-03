---
status: partial
phase: 04-signal-conversations
source: [04-VERIFICATION.md]
started: 2026-08-03T20:49:56Z
updated: 2026-08-03T23:02:44Z
---

## Current Test

[testing paused — 1 item outstanding (test 1 blocked: server)]

## Tests

### 1. E.164 contact-form sgnl:// deep link raises the correct conversation
expected: With Signal Desktop running, clicking "Open in Signal" on a 1:1 (E.164-bearing) conversation's digest raises Signal Desktop and navigates to that contact's conversation. (The bare-scheme group form was already visually confirmed and approved during 04-01's checkpoint.)
result: blocked
blocked_by: server
reason: "server is not running for signal-smoke"

### 2. Judgment-tier prohibitions sign-off (privacy/safety)
expected: |
  Human sign-off on the three judgment-verification prohibitions the plans
  authored without a wired mechanical check (the verifier's code inspection
  found no violations, but the plan's own protocol requires human review):
  (a) No SQLCipher key, config.json key/encryptedKey value, or Signal
  message body text appears in any log line or returned error string —
  diagnostics carry counts, step names and the keyring backend name only.
  (b) No second unencrypted on-disk copy of Signal message plaintext is
  ever created — no temp-file copy or snapshot of db.sqlite, no decrypted
  export, no cache file; only the bounded D-03 tail snippet leaves the
  SQLCipher file (into the local index/FTS).
  (c) A 1:1 conversation is never matched on the contact's self-chosen
  profile name or any derived title falling back to it — only the user's
  own nickname / address-book contact name (anti-spoofing, D-06).
result: pass

## Summary

total: 2
passed: 1
issues: 0
pending: 0
skipped: 0
blocked: 1

## Gaps
