---
phase: 03-email-in-the-webspace
verified: 2026-07-31T18:15:00Z
status: gaps_found
score: 3/5 must-haves verified
behavior_unverified: 1
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 3/5
  gaps_closed:
    - "plugins/proton/plugin.go's setMailboxCache replaced the whole mailboxCache map on every Match call instead of merging, so a sync cycle covering two or more webspaces left only the last-processed webspace's items Fetch-able — now fixed by mergeMailboxCache, an under-lock entry-wise upsert with a single call site on Match's success path, and the zero-mailboxes-matched branch now performs no cache mutation at all. Confirmed by reading plugin.go directly (mergeMailboxCache at ~line 404, single call site at line 227, no `p.mailboxCache =` reassignment anywhere in the file) and by TestMatch_MailboxCacheSurvivesASecondWebspaceMatch / TestMatch_ZeroMailboxMatchPreservesMailboxCache both passing."
    - "web/src/lib/components/DetailPane.svelte declared its own local date formatter that omitted the timeZone: 'UTC' pin web/src/lib/format.ts's formatItemDate enforces, so the same item's date could show a different calendar day in the detail pane than in its stream row — now fixed: DetailPane.svelte imports and calls formatItemDate from $lib/format, the local formatter is deleted, and a new source-scan guard (web/src/lib/components/date-format.test.ts) fails the suite if any first-party component reintroduces an un-pinned formatter. Confirmed by reading DetailPane.svelte and by the guard test passing (72/72 frontend tests)."
  gaps_remaining: []
  regressions: []
gaps:
  - truth: "Emails in Proton folders or labels matching the webspace keyword appear in the stream with sender, subject, and date, and render their body inline in the detail pane (ROADMAP SC1)"
    status: partial
    reason: >
      The two functional defects that broke this truth in the prior
      verification round (multi-webspace Fetch NotFound; detail-pane/
      stream-row date disagreement) are both genuinely fixed and locked
      by passing regression tests — confirmed independently in this
      round by reading plugin.go and DetailPane.svelte and by re-running
      the full Go and frontend suites (16 proton tests pass + 1
      environment-gated skip; 72/72 frontend tests; svelte-check 0
      errors). However, this round's independent code review
      (03-REVIEW.md, committed aee46d1, superseding the prior round's
      review) surfaced a NEW, unaddressed CRITICAL finding directly in
      the same code path this truth depends on: plugins/proton/go.mod
      pins golang.org/x/net v0.26.0 (indirect, pulled in by
      github.com/microcosm-cc/bluemonday, which uses
      golang.org/x/net/html as its HTML tokenizer). Every
      golang.org/x/net release before v0.33.0 carries CVE-2024-45338
      (GO-2024-3333): a crafted HTML document can be tokenized
      non-linearly with respect to its length, a CPU/memory exhaustion
      DoS vector. plugins/proton/body.go's RenderSanitizedEmail — called
      from fetchFull, the exact function this truth's "render their body
      inline in the detail pane" clause requires — runs
      bluemonday.SanitizeBytes (and therefore this tokenizer) directly
      over the HTML body of an arbitrary inbound email, with nothing
      upstream filtering or pre-validating the bytes. Any sender able to
      deliver mail to the monitored Proton account can trigger this
      code path with fully attacker-chosen input. This is new to this
      round (the plugin's HTML-rendering feature that introduced
      bluemonday shipped in 03-02; no reviewer had previously flagged
      this specific pinned version), and it directly threatens the
      phase goal's "stays readable there" clause — a malicious inbound
      email could hang or exhaust the plugin process on open, rather
      than merely fail to render. Independently confirmed in this
      verification pass by reading plugins/proton/go.mod:14 and go.sum:
      23-24 at current HEAD — still v0.26.0, unbumped.
    artifacts:
      - path: "plugins/proton/go.mod"
        issue: "golang.org/x/net pinned to v0.26.0 (indirect, via github.com/microcosm-cc/bluemonday), predates the CVE-2024-45338/GO-2024-3333 fix landed in v0.33.0"
      - path: "plugins/proton/body.go"
        issue: "RenderSanitizedEmail runs bluemonday.SanitizeBytes (and therefore the vulnerable golang.org/x/net/html tokenizer) directly over an arbitrary inbound email's HTML body with no upstream validation — this is the new attacker-controlled-input surface that makes the stale dependency pin exploitable rather than theoretical"
    missing:
      - "Bump golang.org/x/net to >= v0.33.0 in plugins/proton/go.mod's require block and regenerate plugins/proton/go.sum"
      - "Check plugins/silverbullet/go.mod for the same stale golang.org/x/net pin while addressing this (it also renders third-party-sourced content via goldmark/bluemonday, per 03-REVIEW.md CR-01's own note) — outside this phase's own scope but flagged since the same fix motion applies"
      - "Re-run the plugin's full test suite after the bump to confirm no wire-format or behavioral change in bluemonday's tokenizer output breaks TestRenderSanitizedEmail_* or TestWrapDocument_*"
human_verification:
  - test: "Open a webspace with a real, currently-reachable Proton Bridge account configured. Confirm at least one real Proton email appears in the stream interleaved with paperless/SilverBullet items, showing sender before the date, with the subject as the row title. Click it: the detail pane opens and shows the message's body (plain text, or formatted HTML with headings/links/colors if the message has an HTML part). If more than one configured webspace matches Proton mail, additionally confirm the SAME check holds for a webspace that is NOT the last one synced in that cycle."
    expected: "The email is visible, correctly dated, and its body renders in the detail pane for every configured webspace that matches Proton mail — not just the one processed last in a sync cycle. This is the direct live confirmation of the mailboxCache fix closed this round."
    why_human: "Requires a running kernel/webUI against a real, currently-authenticating Proton Mail Bridge account and visual confirmation of rendered HTML/CSS in a browser — not mechanically checkable from source alone. Blocked again this session: the Bridge account credential rejection recorded in 03-01-SUMMARY.md's 'Notable Live-Environment Finding' is unchanged across three verification passes now."
  - test: "Run `WEBSPACES_PROTON_LIVE_IT=1 PROTON_BRIDGE_ADDR=<addr> PROTON_BRIDGE_USER=<user> PROTON_BRIDGE_PASS=<pass> go test -run TestSeenFlagUnchanged_LiveBridge -v ./plugins/proton/...` against the real Bridge account once its credential issue is corrected."
    expected: "The test passes, directly proving SRC-01's second success criterion (\\Seen flag unchanged across a full Match+Fetch cycle) against the real mailbox, not just the no-live-Bridge wire-transcript proxy for it (TestIMAPTranscript_ExamineAndPeekOnly / TestPluginIssuesNoIMAPMutatingCommands, both of which still pass)."
    why_human: "Requires a live, currently-authenticating Bridge connection and real mailbox state; the test is implemented and skips cleanly (confirmed again this session: SKIP, not FAIL) but has never been run to a PASS — same Bridge credential blocker as before, unchanged."
  - test: "After running a webspaces sync and opening an email in the detail pane, check the same email in the real Proton web or mobile client and confirm it is still shown as unread there."
    expected: "The email remains unread in Proton's own client — the direct, human-observable proof of the never-mark-read guarantee end to end."
    why_human: "Requires a live Bridge account, a real sync, and cross-checking read/unread state in Proton's own UI; unchanged blocker."
  - test: "Open a webspace in the browser, type a word present in a document, a note, and an email into the search box, and confirm ranked cross-source results appear with the matched word emboldened, a result opens the detail pane on click, clearing restores the unfiltered stream, and a nonsense or malformed (lone double-quote) query shows the no-matches state rather than an error."
    expected: "All of 03-04-PLAN.md's Task 1 human-check behaviors hold visually in a running browser."
    why_human: "Visual/interactive confirmation of debounce feel, snippet emphasis rendering, and detail-pane opening — beyond what the passing unit tests over parseSnippet/searchVariant can prove; not run this session (no running kernel/webUI session with live data was exercised)."
---

# Phase 3: Email in the Webspace Verification Report

**Phase Goal:** User's Proton mail for a topic appears in the webspace stream and stays readable there, and the volume it brings is navigable by searching within the webspace
**Verified:** 2026-07-31T18:15:00Z
**Status:** gaps_found
**Re-verification:** Yes — after gap closure plan 03-06 (mailboxCache accumulates across webspaces instead of being replaced; DetailPane date renders through the shared UTC-pinned formatter)

**Note on MVP mode:** ROADMAP.md marks this phase `Mode: mvp`, but its `Goal:` line is not phrased as a User Story (`As a … I want to … so that …`) — a discrepancy the phase's own plans (03-05, 03-06) already noted and deliberately did not resolve by inventing one. Running `user-story.validate` against the goal text confirms `valid: false`. The verification task for this round was dispatched with the standard goal-backward success-criteria list (not a User Story), matching how the two prior verification rounds for this exact phase were conducted. This report follows that established precedent and uses standard goal-backward verification rather than the MVP User Flow Coverage format; flagging this here rather than silently proceeding, per the MVP-mode format guard's intent.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Emails in Proton folders/labels matching the webspace keyword appear in the stream with sender, subject, date, and render body inline in the detail pane (SC1) | ✗ FAILED (partial) | Both prior functional blockers (mailboxCache replace-not-merge; DetailPane date drift) are genuinely fixed and test-locked. A new, independently-confirmed CRITICAL supply-chain vulnerability (stale `golang.org/x/net`, CVE-2024-45338) sits unaddressed in the exact HTML-rendering code path this truth requires. See gap below. |
| 2 | Reading an email inside webspaces never marks it read in Proton — proven by an automated test asserting `\Seen` unchanged after a full sync and a detail fetch (SC2) | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | `plugins/proton/readonly_test.go`'s `TestPluginIssuesNoIMAPMutatingCommands` (AST scan forbidding IMAP-mutating identifiers) and `plugins/proton/imap_transcript_test.go`'s `TestIMAPTranscript_ExamineAndPeekOnly` (wire-transcript assertion: `EXAMINE`/`BODY.PEEK[` present, `SELECT`/`STORE`/`EXPUNGE`/`APPEND`/`COPY`/`MOVE`/`DELETE` absent, across a full Match+Fetch cycle) both pass — code-level proof no `\Seen`-setting command is ever issued. The live-Bridge test that would assert the actual flag value on a real mailbox (`TestSeenFlagUnchanged_LiveBridge`) still SKIPs — confirmed again this session — due to the unchanged Bridge credential blocker (03-01-SUMMARY.md). No behavioral test has ever run to PASS against a real account. |
| 3 | An email carrying several Proton labels appears exactly once in the stream, deduped by Message-ID (SC3) | ✓ VERIFIED | `plugins/proton/plugin.go` `Match`'s `byMessageID` map merges matches by `normalizeMessageID(msg.Envelope.MessageId)`, appending the mailbox leaf to `labels` on a repeat hit rather than creating a second item. `TestIMAPTranscript_ExamineAndPeekOnly` asserts exactly 1 item is returned for a message present in two matching mailboxes, with both leaf names present in `item.GetLabels()`. Confirmed passing. |
| 4 | User can type a query and get ranked, clickable matches across every source in the webspace (SC4) | ✓ VERIFIED | `kernel/httpapi/search.go` + `kernel/index` FTS5 bm25-ranked query, `kernel/httpapi/search_test.go`'s 7 tests all pass (unknown webspace 404, empty/whitespace query → empty results not error, single-quote malformed query → 200 not 500, zero-match, one-match-with-snippet). Frontend `SearchBox.svelte`/`SearchResults.svelte`/`format.ts`'s `parseSnippet`/`searchVariant` unchanged this round, covered by passing `format.test.ts`. Unchanged and previously verified; re-confirmed passing this round. |
| 5 | The plugin reaches Proton Mail Bridge over LAN despite its self-signed certificate, and reports a clear, actionable health error rather than hanging when unreachable (SC5) | ✓ VERIFIED | `plugins/proton/client.go`: `tlsConfig := &tls.Config{ServerName: bridgeCertServerName}` with an explicit `RootCAs` pool built from the pinned exported Bridge certificate (falls back to nil/system pool with a logged warning, never a silent `InsecureSkipVerify` — confirmed absent via `grep -rq InsecureSkipVerify plugins/proton/` finding nothing). `plugin.go`'s `Health` calls `p.client.connect(healthDialTimeout)` (a bounded dial, not an unbounded one) and on error returns `&webspacesv1.HealthResponse{Reachable: false, LastError: err.Error()}, nil` — a structured, non-hanging, non-panicking response. |

**Score:** 3/5 truths verified (1 present, behavior-unverified; 1 failed/partial — see gap)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `plugins/proton/plugin.go` | `mergeMailboxCache` entry-wise upsert; single call site on Match's success path; zero-mailboxes-matched branch performs no cache mutation | ✓ VERIFIED | Confirmed by reading: `mergeMailboxCache` at ~line 404 (`p.mailboxMu.Lock()`, iterates `discovered`, upserts per key, never re-binds the map); single call site `p.mergeMailboxCache(discovered)` at line 227; `grep -c 'p\.mailboxCache = '` finds zero re-bindings. |
| `plugins/proton/mailbox_cache_test.go` | Two regression tests simulating the per-webspace Match loop | ✓ VERIFIED | `TestMatch_MailboxCacheSurvivesASecondWebspaceMatch` and `TestMatch_ZeroMailboxMatchPreservesMailboxCache` both exist and pass; SUMMARY quotes their pre-fix RED output (`codes.NotFound`) confirming they were observed failing for the right reason before the fix. |
| `plugins/proton/imap_transcript_test.go` | Fourth seeded mailbox (`Labels/GammaTeam`) with a distinct Message-Id | ✓ VERIFIED | `gammaMessageID`/`sharedMessageID` consts present; `Labels/GammaTeam` mailbox seeded; `TestIMAPTranscript_ExamineAndPeekOnly`, `TestMatch_ItemTimestampIsInternalDate`, `TestMatch_EmptyMessageIDSkipIsLogged` all still pass against the extended fixture. |
| `web/src/lib/components/DetailPane.svelte` | Renders date through shared `formatItemDate`, no local formatter | ✓ VERIFIED | `import { detailPaneState, formatItemDate } from '$lib/format';` present; `{formatItemDate(item.timestamp_unix)}` at the header date span; `toLocaleDateString` absent from the file (`grep` confirms zero matches). |
| `web/src/lib/components/date-format.test.ts` | Source-scan guard over top-level components | ✓ VERIFIED | New file exists, well-formed (not a stub): enumerates top-level `.svelte` files, asserts non-empty list, asserts no `toLocaleDateString`, asserts every `timestamp_unix`-referencing component imports `formatItemDate`. Passes. |
| `plugins/proton/go.mod` | Dependency set free of known-exploitable CVEs in code paths handling attacker-controlled input | ✗ GAP | `golang.org/x/net v0.26.0` (indirect via bluemonday) still pinned; CVE-2024-45338 unaddressed. Not an artifact any plan claimed as a must-have, but surfaced by 03-REVIEW.md and independently confirmed here. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `plugins/proton/plugin.go Match` | `plugins/proton/plugin.go fetchFull` | Accumulated `mailboxCache` map read by `mailboxForSourceID` | ✓ WIRED | `mailboxForSourceID` (RLock read) is the only source `fetchFull` consults to resolve which mailbox to `EXAMINE`; confirmed the map now accumulates (not replaced) across calls, verified by the two new regression tests passing. |
| `kernel/correlate/correlate.go SyncSource` | `plugins/proton/plugin.go Match` | One long-lived plugin instance receiving one `Match` per configured webspace per cycle | ✓ WIRED (unchanged) | `git diff --exit-code kernel/correlate/correlate.go kernel/pluginhost/host.go` against the pre-03-06 tree shows no changes — confirmed the kernel-side loop was correctly scoped already and was deliberately left untouched; only the plugin-side state needed the fix. |
| `web/src/lib/components/DetailPane.svelte` | `web/src/lib/format.ts` | Shared `formatItemDate`, already covered by `format.test.ts`'s UTC-pin assertion | ✓ WIRED | Import line and call site both confirmed present; `date-format.test.ts` makes a future re-divergence a test failure rather than a silent visual bug. |
| `plugins/proton/body.go RenderSanitizedEmail` | `golang.org/x/net/html` (via `bluemonday`) | Untrusted email HTML parsed by a pinned, CVE-affected tokenizer | ✗ NOT REMEDIATED | This link is wired and functionally correct for ordinary input (all `TestRenderSanitizedEmail_*`/`TestWrapDocument_*` pass), but the tokenizer version behind it is a known DoS vector against the exact attacker-controlled input class this function processes. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| mailboxCache accumulates across two webspaces' Match calls | `go test -run TestMatch_MailboxCacheSurvivesASecondWebspaceMatch -v ./plugins/proton/...` | PASS | ✓ PASS |
| Zero-mailbox-matched Match does not erase prior entries | `go test -run TestMatch_ZeroMailboxMatchPreservesMailboxCache -v ./plugins/proton/...` | PASS | ✓ PASS |
| DetailPane / StreamRow date agreement guard | `npm run test` (date-format.test.ts) | 72/72 pass | ✓ PASS |
| Full proton plugin suite, repo build/vet/test, frontend suite + svelte-check | see below | All green except env-gated skip | ✓ PASS |
| `golang.org/x/net` still pinned to the vulnerable version | `grep -n "golang.org/x/net" plugins/proton/go.mod` | `v0.26.0 // indirect` | ✗ FAIL (gap) |

Full regression evidence gathered this session:
- `CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go vet ./... && CGO_ENABLED=0 go test ./... -count=1` — clean, all packages `ok`.
- `cd plugins/proton && go build ./... && go vet ./... && go test ./... -count=1 -v` — 16 tests PASS, 1 SKIP (`TestSeenFlagUnchanged_LiveBridge`, environment-blocked).
- `npm run test` (web) — 4 test files, 72/72 tests pass.
- `npm run check` (web) — `COMPLETED 746 FILES 0 ERRORS 1 WARNINGS` (pre-existing `SearchBox.svelte` `state_referenced_locally` warning, untouched, out of scope).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| SRC-01 | 03-01, 03-02, 03-05, 03-06 | Email plugin (IMAP) works against Proton Mail Bridge (self-signed cert handling); uses `BODY.PEEK` so mail is never marked read; matches webspace keyword against folders/labels; dedups by Message-ID | ✓ SATISFIED (with 1 flagged security gap) | Self-signed cert: `client.go` TLS pinning (SC5 row). `BODY.PEEK`: `TestIMAPTranscript_ExamineAndPeekOnly` (SC2 row, code-level proof; live proof still blocked). Keyword matching: `Match`'s `matchesAnyKeyword`/`leafName`. Dedup: `byMessageID` map (SC3 row). All functional clauses hold; the unaddressed `golang.org/x/net` CVE sits in the adjacent body-rendering path this requirement's "works against Proton Mail Bridge" implicitly covers (see gap). |
| KERN-05 | 03-03, 03-04 | User can full-text search within a webspace (FTS5 over indexed metadata/previews) | ✓ SATISFIED | `kernel/index` FTS5 schema/triggers, `kernel/httpapi/search.go`, all `search_test.go` cases pass; frontend `SearchBox`/`SearchResults`/`parseSnippet` covered by passing tests. Unchanged this round. |

No orphaned requirements: REQUIREMENTS.md maps exactly SRC-01 and KERN-05 to Phase 3, and both are declared in at least one plan's `requirements:` frontmatter field.

### Anti-Patterns Found

Mechanical grep for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`/empty-implementation patterns across this round's modified files (`plugins/proton/plugin.go`, `mailbox_cache_test.go`, `imap_transcript_test.go`, `DetailPane.svelte`, `date-format.test.ts`, `node-builtins.d.ts`) found **none**.

The phase's own code review (`03-REVIEW.md`, `aee46d1`) surfaced non-textual findings beyond grep's reach:

| File | Finding | Severity | Status |
|------|---------|----------|--------|
| `plugins/proton/go.mod:14` | CR-01: `golang.org/x/net v0.26.0` predates CVE-2024-45338 fix, reachable from arbitrary inbound email HTML | 🛑 Blocker | Unaddressed — promoted to a phase gap above |
| `plugins/proton/plugin.go` (`Match`/`Fetch`/`fetchFull`), `client.go:196` | WR-01: `context.Context` accepted but never honored in the IMAP call path (no cancellation on dial/fetch) | ⚠️ Warning | Not blocking any stated success criterion; robustness gap only |
| `plugins/proton/plugin.go:352` (`toItem`) | WR-02: Email `Subject` stored verbatim as `Item.Title` with no control-character stripping; a crafted Subject could inject FTS5 snippet delimiter bytes (STX/ETX) | ⚠️ Warning | Bounded impact (`parseSnippet` degrades gracefully, no XSS); not blocking SC1/SC4 today |
| `plugins/proton/plugin.go:474-484` | WR-03: `fetchFull` trusts `UID SEARCH HEADER Message-Id`'s substring match without verifying exact equality against the resolved message | ⚠️ Warning | Theoretical wrong-body-served risk under adversarial Message-Id crafting; no test demonstrates an actual failure |
| `web/src/routes/w/[webspace]/+page.svelte:144-168` | WR-04: Clearing the search box does not advance `searchRequestSeq`, so a stale in-flight search response is not fully guarded | ⚠️ Warning | No visible effect today (`searchVariant` gates rendering); internal state can go inconsistent |
| `plugins/proton/plugin.go:459-462` | WR-05: Cold `mailboxCache` post-restart is indistinguishable from a genuinely deleted message | ⚠️ Warning | Transient UX regression on kernel restart, not a data-correctness issue |
| `plugins/proton/plugin.go:190-214`, `plugins/proton/body.go:29,112-121`, `plugins/proton/client.go:22-44`, `web/src/lib/api.ts:121-161` | IN-01 through IN-04: unused fetched field, unused `Snippet` helper, dead sentinel errors, duplicated `getJSON`/`postJSON` logic | ℹ️ Info | Code-quality only, carried over from an earlier review round, not blocking any success criterion |

Only CR-01 is treated as a phase-blocking gap in this report: it sits directly in the code path SC1 depends on, is independently confirmed present in the current tree, and has a mechanical, low-effort fix (a `go.mod` version bump). The WR-*/IN-* items are recorded for completeness but do not falsify any of the five stated success criteria and are left to the project's own backlog process.

### Human Verification Required

See the `human_verification` block in this report's frontmatter (four items, all environment-blocked by the unchanged Proton Mail Bridge credential issue first recorded in 03-01-SUMMARY.md, now unchanged across three verification passes).

### Gaps Summary

Both functional blockers identified in the prior verification round are genuinely closed: the Proton plugin's mailbox resolution state now accumulates across every configured webspace's `Match` call instead of being replaced by the most recently processed one (so opening an email now works for every webspace in a multi-webspace deployment, not just the last one synced), and the detail pane's date now agrees with the stream row's date in every timezone. Both fixes are locked by regression tests that were observed failing for the right reason before the fix landed, and the full Go and frontend suites pass cleanly.

One new gap was found by this round's independent code review and confirmed directly against the current codebase: `plugins/proton/go.mod` still pins `golang.org/x/net v0.26.0`, which predates the fix for CVE-2024-45338 (GO-2024-3333) — a tokenizer DoS reachable through `plugins/proton/body.go`'s `RenderSanitizedEmail`, the function that implements SC1's "render their body inline in the detail pane" clause, run directly against arbitrary attacker-controlled inbound email HTML. This is a mechanically simple fix (bump the dependency, regenerate `go.sum`, re-run the suite) but is unaddressed as of the current HEAD and is treated as a blocking gap given its severity and direct relevance to this phase's core deliverable.

The phase's four environment-blocked human-verification items (live Proton Bridge account required) remain unchanged across three verification rounds and are not new to this round.

---

_Verified: 2026-07-31T18:15:00Z_
_Verifier: Claude (gsd-verifier)_
