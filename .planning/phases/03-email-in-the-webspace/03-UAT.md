---
status: partial
phase: 03-email-in-the-webspace
source: [03-VERIFICATION.md]
started: 2026-07-31T16:57:00Z
updated: 2026-07-31T20:40:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Live stream and detail-pane rendering across multiple webspaces
expected: The email is visible, correctly dated, and its body renders in the detail pane for every configured webspace that matches Proton mail — not just the one processed last in a sync cycle. This is the direct live confirmation of the mailboxCache fix (03-06) plus the CVE-remediated rendering path (03-07).
result: issue
reported: "proton mail shows as unavailable in the UI (red dot)"
severity: major

### 2. Live-Bridge \Seen flag test passes
expected: |
  Run
  `WEBSPACES_PROTON_LIVE_IT=1 PROTON_BRIDGE_ADDR=<addr> PROTON_BRIDGE_USER=<user> PROTON_BRIDGE_PASS=<pass> go test -run TestSeenFlagUnchanged_LiveBridge -v ./plugins/proton/...`
  against the real Bridge account once its credential issue is corrected. The test
  passes, directly proving SRC-01's second success criterion (\Seen flag unchanged
  across a full Match+Fetch cycle) against the real mailbox, not just the
  wire-transcript proxy for it.
result: blocked
blocked_by: third-party
reason: "TestSeenFlagUnchanged_LiveBridge FAIL — live login: no such user. User has since ruled out credential-entry error: correct account email AND several aliases all return the same 'no such user'. The 03-01-SUMMARY.md 'credential finding' explanation no longer holds as stated — same root cause under diagnosis as G-03-1."

### 3. Email stays unread in Proton's own client
expected: After running a webspaces sync and opening an email in the detail pane, the same email checked in the real Proton web or mobile client is still shown as unread there — the direct, human-observable proof of the never-mark-read guarantee end to end.
result: blocked
blocked_by: third-party
reason: "blocked — same Bridge 'no such user' login failure as Tests 1–2; no live sync possible until it is resolved"

### 4. Live in-webspace search UX
expected: Typing a word present in a document, a note, and an email into the webspace search box shows ranked cross-source results with the matched word emboldened; a result opens the detail pane on click; clearing restores the unfiltered stream; and a nonsense or malformed (lone double-quote) query shows the no-matches state rather than an error.
result: pass
note: "User caveat: email hits could not be verified — Proton source is down (same Bridge blocker as Tests 1–3). Document/note search, ranking, snippet emphasis, detail-pane open, clear-restore, and malformed-query handling all confirmed working."

## Summary

total: 4
passed: 1
issues: 1
pending: 0
skipped: 0
blocked: 2

## Gaps

- gap_id: G-03-1
  truth: "The email is visible, correctly dated, and its body renders in the detail pane for every configured webspace that matches Proton mail — not just the one processed last in a sync cycle."
  status: failed
  reason: "User reported: proton mail shows as unavailable in the UI (red dot). Corroborating evidence from Test 2: live IMAP LOGIN to the Bridge returns 'no such user' even with the correct account email address and several aliases (user has ruled out credential-entry error). The long-standing 'Bridge-account credential finding' from 03-01-SUMMARY.md is now in question — diagnosis must cover Bridge-side config (Bridge username/password semantics, combined vs split address mode, LAN exposure/proxy of the localhost-bound Bridge) AND the plugin/test LOGIN handling."
  severity: major
  test: 1
  artifacts: []  # Filled by diagnosis
  missing: []    # Filled by diagnosis
