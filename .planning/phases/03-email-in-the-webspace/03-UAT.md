---
status: complete
phase: 03-email-in-the-webspace
source: [03-VERIFICATION.md]
started: 2026-07-31T16:57:00Z
updated: 2026-08-02T09:30:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Live stream and detail-pane rendering across multiple webspaces
expected: The email is visible, correctly dated, and its body renders in the detail pane for every configured webspace that matches Proton mail — not just the one processed last in a sync cycle. Round 3 adds: detail pane body readable (plain-text preferred; no dark-on-dark, no broken-image litter) and "Open in Proton Mail" opens an All Mail subject search, not the bare inbox.
result: pass
round2_result: issue
round3_note: "2026-08-02: user confirmed pass — detail pane readable and deep link opens All Mail subject search (G-03-2/G-03-3 fixes verified live)"
reported: |
  PARTIAL PASS after .env credential fix + adding label leaf "house+home" to webspace
  keywords: proton source green, items visible in the stream, correctly dated,
  interleaved with paperless/SilverBullet. (Earlier red-dot report resolved — see
  G-03-1.) Two NEW issues found in the detail pane:
  (1) rich HTML rendering unreadable — all images broken (CSP default-src 'none'
  blocks every fetch by design) and email inline color/background-color styles
  clash with the fixed dark theme (dark-on-dark text). See G-03-2.
  (2) "Open in Proton Mail" opens the inbox view with no email selected and no
  label filter — label-name URL path is not addressable in Proton webmail. See G-03-3.
severity: major
note: "Code side closed 2026-08-01 by 03-09 (plain-text preferred rendition; readable HTML fallback with theme colours outranking email colours; unloadable images no longer painted as broken icons) and 03-10 (deep link now an All Mail subject-search URL). Awaiting live round-3 re-test."

### 2. Live-Bridge \Seen flag test passes
expected: |
  Run
  `WEBSPACES_PROTON_LIVE_IT=1 PROTON_BRIDGE_ADDR=<addr> PROTON_BRIDGE_USER=<user> PROTON_BRIDGE_PASS=<pass> go test -run TestSeenFlagUnchanged_LiveBridge -v ./plugins/proton/...`
  against the real Bridge account once its credential issue is corrected. The test
  passes, directly proving SRC-01's second success criterion (\Seen flag unchanged
  across a full Match+Fetch cycle) against the real mailbox, not just the
  wire-transcript proxy for it.
result: pass
note: "2026-08-02: user ran the live test against the real Bridge account — pass. \\Seen flag unchanged across full Match+Fetch cycle, proven live."

### 3. Email stays unread in Proton's own client
expected: After running a webspaces sync and opening an email in the detail pane, the same email checked in the real Proton web or mobile client is still shown as unread there — the direct, human-observable proof of the never-mark-read guarantee end to end.
result: pass
note: "2026-08-02: user cross-checked in Proton's own client after sync + detail-pane open — email still unread. Never-mark-read guarantee confirmed end to end."

### 4. Live in-webspace search UX
expected: Typing a word present in a document, a note, and an email into the webspace search box shows ranked cross-source results with the matched word emboldened; a result opens the detail pane on click; clearing restores the unfiltered stream; and a nonsense or malformed (lone double-quote) query shows the no-matches state rather than an error.
result: pass
note: "User caveat: email hits could not be verified — Proton source is down (same Bridge blocker as Tests 1–3). Document/note search, ranking, snippet emphasis, detail-pane open, clear-restore, and malformed-query handling all confirmed working."

### 5. Corrected red-dot diagnostic text after credential fix
expected: |
  After correcting PROTON_BRIDGE_PASS in .env, if a sync still fails for any reason,
  the surfaced error (Health and the UI red-dot detail) is the server's own message
  plus — only if the configured token is still shape-suspect — the actionable
  bridgeTokenShapeWarningText advice pointing at the Bridge app password. It never
  shows the old username-pointing "credential finding — not a code defect" /
  03-01-SUMMARY.md framing, since that text no longer exists in the codebase
  (closed by 03-08; code-level guarantee verified, rendered appearance is not).
result: pass
note: "2026-08-02: user confirmed pass."

## Summary

total: 5
passed: 5
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

- gap_id: G-03-1
  truth: "The email is visible, correctly dated, and its body renders in the detail pane for every configured webspace that matches Proton mail — not just the one processed last in a sync cycle."
  status: resolved
  resolution: "2026-08-01: user replaced PROTON_BRIDGE_PASS in .env with the real Bridge app password (retrieved via containerized Bridge CLI: stop daemon, run `docker run --rm -it -v <vol>:/root <image> init`, `info`). Also added Proton label leaf 'house+home' to [webspaces.house-move] keywords — matching is exact-leaf-equality (D-03), and no existing label equaled the original keywords. Confirmed live: proton source green, items visible, correctly dated, interleaved in the stream. Code-side items were closed by 03-08 previously."
  reason: "User reported: proton mail shows as unavailable in the UI (red dot). Corroborating evidence from Test 2: live IMAP LOGIN to the Bridge returns 'no such user' even with the correct account email address and several aliases (user has ruled out credential-entry error). The long-standing 'Bridge-account credential finding' from 03-01-SUMMARY.md is now in question — diagnosis must cover Bridge-side config (Bridge username/password semantics, combined vs split address mode, LAN exposure/proxy of the localhost-bound Bridge) AND the plugin/test LOGIN handling."
  severity: major
  test: 1
  root_cause: "ENVIRONMENTAL, not a code defect: PROTON_BRIDGE_PASS in .env (desktop) is not the Bridge-generated app password — it is a 37-char symbol-heavy string impossible in Bridge's base64url password alphabet [A-Za-z0-9-_] (likely the Proton account password). Bridge's CheckAuth base64url-decodes the password BEFORE comparing the username, and gluon returns its single auth-failure string 'no such user' for every rejected (username, password) pair — which is why the correct address and every alias fail identically. Live probe confirmed the LAN forwarder and Bridge itself are healthy (genuine Bridge 03.25.00 greeting at monroe:1143). Plugin/kernel pass credentials through verbatim — code path audited clean."
  artifacts:
    - path: ".env"
      issue: "PROTON_BRIDGE_PASS holds a non-Bridge password; user must replace it with the Bridge app password (user-side action, not plannable)"
    - path: "plugins/proton/live_bridge_test.go"
      issue: "Hint text at ~line 160 steers diagnosis to the username ('credential finding — not a code defect') but Bridge checks the password first — misleading, caused four verification rounds of misdiagnosis"
  missing:
    - "USER ACTION (not plannable): read the real Bridge credentials on monroe (Bridge GUI → account → Mailbox details, or `protonmail-bridge --cli` then `info`; sign in first if signed out), replace PROTON_BRIDGE_PASS in .env with the ~20–22 char [A-Za-z0-9-_] app password, wait out Bridge's login jail, restart kernel, re-run live test"
    - "CLOSED by 03-08 (verified 2026-08-01): live_bridge_test.go hint corrected — now states Bridge validates the password before the username via the shared bridgeAuthOrderNote constant, pointing at the password"
    - "CLOSED by 03-08 (verified 2026-08-01): shape check self-diagnosing — bridgeTokenShapeWarning in plugins/proton/credentials.go appends actionable advice at client.connect's LOGIN-failure branch, reaching both HealthResponse.LastError and the UI red-dot detail via Match → sync_runs; never blocks a connection attempt, never echoes the token"
  debug_session: .planning/debug/proton-bridge-no-such-user.md

- gap_id: G-03-2
  truth: "An email opened in the detail pane is readable: no dark-on-dark text, no broken-image litter. When a text/plain part exists it is shown in preference to the HTML rendition; the HTML rendition remains the fallback for HTML-only emails."
  status: resolved
  resolved_by: 03-09-PLAN.md
  resolved_at: 2026-08-02
  reason: "User reported (with screenshot): rich HTML rendition shows every image as a broken icon and email-supplied text colours are often unreadable against the dark pane background."
  severity: major
  test: 1
  root_cause: |
    Two deliberate security choices compound: (1) emailSanitizePolicy
    (plugins/proton/body.go:159) allows color/background-color inline styles
    through on styledElements, so light-background email designs set
    near-black text inside the fixed dark themeStyle wrapper; (2) the
    kernel rendition route serves under CSP default-src 'none'; sandbox
    (tracking-pixel defense, T-03-10/T-03-11), so <img> survives
    sanitization but every fetch is blocked. Meanwhile Fetch ALREADY
    extracts the text/plain part into FetchResponse.Text for every message
    (plugins/proton/plugin.go:515,526), but DetailPane.svelte:126 prefers
    the text/html rendition whenever one exists, so the plain text is
    never displayed. User direction: prefer plain text when available or
    computable.
  artifacts:
    - path: "plugins/proton/plugin.go"
      issue: "fetchFull always emits the HTML rendition when an HTML part exists, even when a readable text/plain part was extracted"
    - path: "web/src/lib/components/DetailPane.svelte"
      issue: "prefers rendition over extracted text whenever mime_type is text/html"
  missing:
    - "CLOSED by 03-09 (verified 2026-08-01): plugin-side selection — Fetch returns the plain-text part as content whenever one exists, so no rendition is offered for it; DetailPane renders from one named decision (detailBodyVariant)"
    - "CLOSED by 03-09 (verified 2026-08-01): sanitized rendition kept as fallback for HTML-only mail and made readable — theme colours outrank email-supplied inline colours; images that can never load under the rendition CSP no longer paint as broken icons"
    - "CLOSED by 03-09 (verified 2026-08-01): sanitizer allowlist, CSP, and every standing security assertion unchanged — confirmed by the full plugins/proton suite and internal/audit"
    - "REMAINING (human): live browser confirmation of readability — backstop-verified truth, only confirmable via 03-UAT.md Test 1 re-run"

- gap_id: G-03-3
  truth: "The detail pane's 'Open in Proton Mail' affordance lands the user somewhere useful in Proton webmail — at minimum a view adjacent to the message (ANCHORED fidelity), never silently the bare inbox."
  status: resolved
  resolved_by: 03-10-PLAN.md
  resolved_at: 2026-08-02
  reason: "User reported: clicking 'Open in Proton Mail' opens https://mail.proton.me/u/1/inbox — inbox view, no email selected, no label filter."
  severity: minor
  test: 1
  root_cause: |
    toItem (plugins/proton/plugin.go:373) builds {webmail_base}/{label-leaf},
    e.g. /u/1/house%2Bhome. Proton webmail addresses custom labels/folders by
    internal ID, not name (only system folders have name paths), so the
    unknown path is redirected to the inbox. 03-RESEARCH.md Pitfall 5 already
    ruled out per-message URLs (no Message-ID -> webmail id mapping); the
    label-name URL chosen as the ANCHORED fallback was never actually
    addressable. Candidate fix: hash-based search deep link that Proton
    webmail does honor, e.g. {base}/all-mail#keyword=<subject>, still
    declared ANCHORED.
  artifacts:
    - path: "plugins/proton/plugin.go"
      issue: "deepLink uses a label-name URL path that Proton webmail cannot resolve; redirects to inbox"
  missing:
    - "CLOSED by 03-10 (verified 2026-08-01): label-name path replaced by webmailSearchDeepLink — percent-encoded, rune-capped All Mail search over the message's subject (plugins/proton/deeplink.go), absent when there is no subject, still declared ANCHORED; config descriptions corrected to match"
    - "REMAINING (human): live click-through confirming Proton webmail honors the search-link form — backstop-verified truth, only confirmable via 03-UAT.md Test 1 re-run"
