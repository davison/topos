---
status: complete
phase: 04-signal-conversations
source: [04-VERIFICATION.md]
started: 2026-08-03T20:49:56Z
updated: 2026-08-04T00:08:23Z
---

## Current Test

[testing complete]

## Tests

### 1. E.164 contact-form sgnl:// deep link raises the correct conversation
expected: With Signal Desktop running, clicking "Open in Signal" on a 1:1 (E.164-bearing) conversation's digest raises Signal Desktop and navigates to that contact's conversation. (The bare-scheme group form was already visually confirmed and approved during 04-01's checkpoint.)
result: issue
reported: "the error dialog no longer appears, but the correct conversation is not opened, it remains on whatever was last opened in the UI"
severity: major
retest_context: "Re-run after 04-04 gap closure — deep-link builder now emits literal '+' (E.164 allowlist, verbatim); live index re-synced (111 rows corrected, 0 rejected-shape rows remain); server restarted from rebuilt main tree. Prior validation-rejection failure preserved in gap G-04-1 below (resolved); this run surfaced a NEW failure mode: link accepted, no navigation."

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
issues: 1
pending: 0
skipped: 0
blocked: 0

## Gaps

- gap_id: G-04-1
  fix_applied: "04-04 gap-closure plan, merged to main 2756532 (2026-08-04) — awaiting UAT test 1 re-run to confirm resolution"
  truth: "Clicking 'Open in Signal' on a 1:1 (E.164-bearing) conversation's digest raises Signal Desktop and navigates to that contact's conversation via the sgnl://signal.me/#p/+44... form"
  status: resolved
  resolved_by: 04-04-PLAN.md
  resolved_at: 2026-08-04
  reason: "User reported: signal desktop showed an error when activated (see ~/Pictures/.clip.png) — dialog: 'Something went wrong! Sorry, that sgnl:// link didn't make sense!'"
  severity: blocker
  test: 1
  root_cause: "plugins/signal/deeplink.go encodePhoneFragment() uses url.QueryEscape, which percent-encodes the E.164's mandatory leading '+' as %2B (sgnl://signal.me/#p/%2B44...). Signal Desktop never percent-decodes the hash-captured phone number and validates it against /^\\+[1-9]\\d{1,14}$/ requiring a literal '+', so validation fails and it shows the 'unknown-sgnl-link' error modal. The bug is test-enshrined: TestEncodePhoneFragment_PlusSignEscaped asserts the %2B encoding."
  artifacts:
    - path: "plugins/signal/deeplink.go"
      issue: "encodePhoneFragment uses url.QueryEscape (form-encoding class), converting '+' to %2B; '+' is a legal RFC 3986 fragment sub-delim needing no encoding"
    - path: "plugins/signal/deeplink_test.go"
      issue: "TestEncodePhoneFragment_PlusSignEscaped and TestDeepLink_UnsafeCharactersAreEscapedNotEmittedRaw enshrine the broken escaping; both need inverting with the fix"
  missing:
    - "Validate the E.164 against ^\\+[1-9][0-9]{1,14}$ (Signal's accepted shape) and emit it verbatim (literal '+') when valid"
    - "Fall back to the bare sgnl:// form when the value doesn't match (validate-and-fallback instead of escape)"
    - "Rework the two tests so the regression guard asserts a literal '+' appears in the emitted fragment"
  debug_session: ".planning/debug/sgnl-link-didnt-make-sense.md"

- gap_id: G-04-1b
  truth: "Clicking 'Open in Signal' on a 1:1 (E.164-bearing) conversation's digest raises Signal Desktop AND navigates to that contact's conversation"
  status: failed
  reason: "User reported: the error dialog no longer appears, but the correct conversation is not opened, it remains on whatever was last opened in the UI"
  severity: major
  test: 1
  regression_of: G-04-1
  note: "New failure mode distinct from G-04-1 (validation rejection, now resolved): the literal-'+' link is accepted by Signal Desktop's validator (no error modal) but produces no navigation — window raises/stays on the previously open conversation."
  artifacts: []  # Filled by diagnosis
  missing: []    # Filled by diagnosis
