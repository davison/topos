---
status: testing
phase: 03-email-in-the-webspace
source: [03-VERIFICATION.md]
started: 2026-07-31T16:57:00Z
updated: 2026-07-31T16:57:00Z
---

## Current Test

number: 1
name: Live stream and detail-pane rendering across multiple webspaces
expected: |
  With a real, reachable Proton Bridge account configured, at least one real Proton
  email appears in the stream interleaved with paperless/SilverBullet items, showing
  sender before the date, with the subject as the row title. Clicking it opens the
  detail pane and shows the message body (plain text, or formatted HTML with
  headings/links/colors if the message has an HTML part). If more than one configured
  webspace matches Proton mail, the SAME check holds for a webspace that is NOT the
  last one synced in that cycle.
awaiting: user response

## Tests

### 1. Live stream and detail-pane rendering across multiple webspaces
expected: The email is visible, correctly dated, and its body renders in the detail pane for every configured webspace that matches Proton mail — not just the one processed last in a sync cycle. This is the direct live confirmation of the mailboxCache fix (03-06) plus the CVE-remediated rendering path (03-07).
result: [pending]

### 2. Live-Bridge \Seen flag test passes
expected: |
  Run
  `WEBSPACES_PROTON_LIVE_IT=1 PROTON_BRIDGE_ADDR=<addr> PROTON_BRIDGE_USER=<user> PROTON_BRIDGE_PASS=<pass> go test -run TestSeenFlagUnchanged_LiveBridge -v ./plugins/proton/...`
  against the real Bridge account once its credential issue is corrected. The test
  passes, directly proving SRC-01's second success criterion (\Seen flag unchanged
  across a full Match+Fetch cycle) against the real mailbox, not just the
  wire-transcript proxy for it.
result: [pending]

### 3. Email stays unread in Proton's own client
expected: After running a webspaces sync and opening an email in the detail pane, the same email checked in the real Proton web or mobile client is still shown as unread there — the direct, human-observable proof of the never-mark-read guarantee end to end.
result: [pending]

### 4. Live in-webspace search UX
expected: Typing a word present in a document, a note, and an email into the webspace search box shows ranked cross-source results with the matched word emboldened; a result opens the detail pane on click; clearing restores the unfiltered stream; and a nonsense or malformed (lone double-quote) query shows the no-matches state rather than an error.
result: [pending]

## Summary

total: 4
passed: 0
issues: 0
pending: 4
skipped: 0
blocked: 0

## Gaps
