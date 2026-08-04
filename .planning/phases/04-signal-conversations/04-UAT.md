---
status: complete
phase: 04-signal-conversations
source: [04-VERIFICATION.md]
started: 2026-08-03T20:49:56Z
updated: 2026-08-04T00:43:24Z
---

## Current Test

[testing complete]

## Tests

### 1. E.164 contact-form sgnl:// deep link raises the correct conversation
expected: With Signal Desktop running, clicking "Open in Signal" on a 1:1 (E.164-bearing) conversation's digest raises Signal Desktop and navigates to that contact's conversation. (The bare-scheme group form was already visually confirmed and approved during 04-01's checkpoint.)
result: pass
retest_context: "First re-run (01:08 BST) reported 'accepted but no navigation' — diagnosed as a test artifact: the clicks hit GROUP digests (bare sgnl://, raise-only by design, approved at 04-01); journalctl argv capture proved the 1:1 contact-form link was never exercised. Corrected re-run against webspace 'test' (1:1 'Dad' digests): PASS — Signal Desktop raised and navigated to the contact's conversation, no error modal. See gap G-04-1b diagnosis."

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
passed: 2
issues: 0
pending: 0
skipped: 0
blocked: 0

## Deferred Follow-Ups

- test: 1
  idea: "Differentiate raise-only vs navigating deep links in the UI (OpenInSource.svelte renders both identically — this made designed group raise-only behavior look like a bug and produced a false blocker report). Include distinct button copy/explanation for raise-only links, and optionally emit sgnl://show-window instead of bare sgnl:// (matches Signal's registered showWindow route). Source: G-04-1b diagnosis, .planning/debug/sgnl-link-no-navigation.md fix directions B and C."
  deferred_at: 2026-08-04

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
  status: resolved
  resolved_by: "corrected UAT re-run (test artifact — no code change needed); latent UI-differentiation item captured in Deferred Follow-Ups"
  resolved_at: 2026-08-04
  reason: "User reported: the error dialog no longer appears, but the correct conversation is not opened, it remains on whatever was last opened in the UI"
  severity: major
  test: 1
  regression_of: G-04-1
  note: "New failure mode distinct from G-04-1 (validation rejection, now resolved): the literal-'+' link is accepted by Signal Desktop's validator (no error modal) but produces no navigation — window raises/stays on the previously open conversation."
  root_cause: "Not a deep-link defect — AND-gate of two conditions. (1) Proximate: the retest clicked GROUP digests. journalctl argv capture shows all four Signal launches in the retest window received the bare 'sgnl://' (the by-design group fallback, approved conversation-only fidelity at 04-01); the literal-'+' contact-form URL never entered Signal's argv after the 04-04 fix merged. Signal raises its window unconditionally BEFORE argv parse, then silently drops the routeless bare scheme — exactly 'window up, no dialog, no navigation'. Transport exonerated: portal OpenURI and gio both delivered a literal '+' verbatim and Signal logged 'Matched signal route: contactByPhoneNumber'. (2) Latent, actionable: OpenInSource.svelte renders raise-only and navigating links identically (same button copy, same conversation-only badge), making designed raise-only behavior indistinguishable from a bug. Upstream fact: Signal Desktop 8.21.0 has NO sgnl: route that can navigate to an existing group conversation — no emittable link can do it."
  artifacts:
    - path: "web/src/lib/components/OpenInSource.svelte"
      issue: "Renders raise-only (bare sgnl://) and navigating (contact-form) links identically — the actual gap that turned intended behavior into a false blocker report"
    - path: "plugins/signal/deeplink.go"
      issue: "Correct as-is for this gap; optionally emit sgnl://show-window instead of bare sgnl:// (matches Signal's registered showWindow route rather than raising as a side effect of a null route parse)"
  missing:
    - "Re-run UAT test 1 against a 1:1 digest (webspace 'test', keyword 'Dad' — 105 contact-form rows) and verify mechanically: journalctl --user -o json _CMDLINE shows the +E.164 argv AND ~/.config/Signal/logs/main.log shows 'Matched signal route: contactByPhoneNumber'"
    - "Differentiate raise-only vs navigating links in the UI: distinct fidelity flag/value surfaced through the API or link shape, different button copy, one-line explanation for raise-only"
    - "Optional hardening: emit sgnl://show-window for the no-navigation fallback instead of bare sgnl://"
  debug_session: ".planning/debug/sgnl-link-no-navigation.md"
