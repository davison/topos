---
phase: 03-email-in-the-webspace
verified: 2026-07-31T15:05:27Z
status: gaps_found
score: 3/5 must-haves verified
behavior_unverified: 1
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 3/5
  gaps_closed:
    - "plugins/proton/plugin.go's toItem() never set Item.TimestampUnix despite fetching INTERNALDATE — now fixed: matched.internalDate carries msg.InternalDate through Match's fetch loop, and toItem() assigns a guarded TimestampUnix (0 for a zero INTERNALDATE, never the -62135596800 sentinel). Confirmed by reading plugin.go and by TestMatch_ItemTimestampIsInternalDate / TestToItem_ZeroInternalDateYieldsZeroTimestamp passing."
    - "skippedNoMessageID was counted but never logged (03-01's 'counted, logged skip' must-have was half-implemented) — now fixed: a guarded, count-only fmt.Fprintf to the new SourcePlugin.logOut field replaces the blank-identifier discard. Confirmed by reading plugin.go and by TestMatch_EmptyMessageIDSkipIsLogged passing."
    - "client.go:56's comment contained the literal string 'InsecureSkipVerify', failing 03-01-PLAN.md's own mechanical acceptance grep — now fixed: the parenthetical is restated by concept, with the TLS configuration itself (RootCAs, ServerName) byte-for-byte unchanged. Confirmed: grep -rq InsecureSkipVerify plugins/proton/ now finds nothing."
  gaps_remaining: []
  regressions: []
gaps:
  - truth: "Emails in Proton folders or labels matching the webspace keyword appear in the stream with sender, subject, and date, and render their body inline in the detail pane (ROADMAP SC1)"
    status: failed
    reason: >
      A new, code-level defect independent of the now-closed TimestampUnix
      gap: plugins/proton/plugin.go's SourcePlugin.mailboxCache is the
      ONLY mechanism Fetch/fetchFull uses to resolve a source_id back to
      the IMAP mailbox it must EXAMINE to read the body
      (mailboxForSourceID, ~line 391, used by fetchFull ~line 424).
      setMailboxCache (~line 381) unconditionally REPLACES the whole map
      rather than merging into it, and Match calls it on both the
      zero-mailboxes-matched path (~line 146, replaces with an empty map)
      and the success path (~line 204, replaces with only the mailboxes
      *this* Match call discovered). kernel/correlate/correlate.go's
      SyncSource (~line 77-82) calls src.Match once PER CONFIGURED
      WEBSPACE inside a single sync cycle, sequentially, against the same
      long-lived plugin subprocess instance (kernel/pluginhost/host.go
      launches exactly one Plugin per source name and looks it up by
      source type for every Fetch call — confirmed by reading host.go's
      bySourceType and Discover). Because each webspace's Match call only
      rediscovers messages living in mailboxes matching THAT webspace's
      own keywords, its resulting newCache never contains entries for
      messages that only matched an earlier webspace's keywords in the
      same cycle — and setMailboxCache then discards those earlier
      entries wholesale. Concretely: with two or more configured
      webspaces that both match Proton mail (config.example.toml's
      [webspaces.<name>] block format explicitly supports and expects
      multiple such blocks sharing one [sources.proton] — this is the
      product's designed core use case per PROJECT.md, not an edge case),
      after every sync cycle only the webspace processed last in Go's
      randomized map iteration order has a working Fetch: opening any
      item belonging to any OTHER webspace returns codes.NotFound
      ("source_id %q is not known — the index has not been synced since
      this plugin started") even though the item is visible in the stream
      and the index genuinely was synced. This directly breaks this
      truth's "render their body inline in the detail pane" clause for
      the non-last-synced webspace(s). No test in the phase exercises two
      sequential Match calls against one plugin instance followed by a
      Fetch for an item only the first call discovered, so this shipped
      with every suite green. First surfaced by an independent code
      review pass after 03-05's gap closure landed (03-REVIEW.md CR-01,
      committed as 38e4261 on top of 03-05's own commits) and
      independently re-confirmed here by re-reading plugin.go,
      correlate.go, and host.go directly — unfixed as of current HEAD.
      A second, lower-severity issue from the same review remains present
      too: web/src/lib/components/DetailPane.svelte defines its own local
      formatDate() (~line 33-40) that omits the timeZone: 'UTC' pin
      web/src/lib/format.ts's tested formatItemDate() deliberately
      enforces (format.test.ts asserts this exact pin), so the SAME
      item's date can render on a different calendar day in the detail
      pane header than in its own stream row, for any viewer west of UTC.
    artifacts:
      - path: "plugins/proton/plugin.go"
        issue: "setMailboxCache (~line 381) replaces p.mailboxCache wholesale instead of merging; called from both Match's empty-match path (~line 146) and success path (~line 204), which SyncSource invokes once per configured webspace against the one shared plugin instance"
      - path: "kernel/correlate/correlate.go"
        issue: "SyncSource (~line 77) calls src.Match once per e.Config.Webspaces entry within one sync cycle against a stateful plugin whose only cross-call state (mailboxCache) is not webspace-scoped — the loop itself is correct (per-webspace persistence via ReplaceWebspaceSourceItems is properly isolated), but it exposes the plugin-side defect"
      - path: "web/src/lib/components/DetailPane.svelte"
        issue: "local formatDate() (~line 33) omits the UTC pin that web/src/lib/format.ts's formatItemDate() enforces and web/src/lib/format.test.ts asserts, so the detail pane can show a different calendar day than the stream row for the same item"
    missing:
      - "Merge into plugins/proton/plugin.go's mailboxCache per Match call instead of replacing it (e.g. an under-lock mergeMailboxCache that adds entries rather than assigning a new map), and stop resetting it to an empty map on the zero-mailboxes-matched path — that path has nothing new to contribute for its own webspace but must not erase what an earlier webspace's Match call in the same cycle already contributed"
      - "A regression test in plugins/proton exercising two sequential Match calls against one plugin instance for disjoint keyword sets (simulating SyncSource's per-webspace loop), followed by a Fetch for a source_id that only the FIRST Match call discovered, asserting it still resolves instead of returning NotFound"
      - "web/src/lib/components/DetailPane.svelte should import and render the shared, UTC-pinned formatItemDate from $lib/format instead of defining its own local formatDate"
human_verification:
  - test: "Open a webspace with a real, currently-reachable Proton Bridge account configured. Confirm at least one real Proton email appears in the stream interleaved with paperless/SilverBullet items, showing sender before the date, with the subject as the row title. Click it: the detail pane opens and shows the message's body (plain text, or formatted HTML with headings/links/colors if the message has an HTML part)."
    expected: "The email is visible, correctly dated (the TimestampUnix gap is now fixed), and its body renders in the detail pane. If more than one configured webspace matches Proton mail, additionally confirm the SAME check holds for a webspace that is NOT the last one synced in that cycle — this is exactly the scenario the CR-01 gap above breaks."
    why_human: "Requires a running kernel/webUI against a real, currently-authenticating Proton Mail Bridge account and visual confirmation of rendered HTML/CSS in a browser — not mechanically checkable from source alone. Blocked this session: the Bridge account credential rejected LOGIN (03-01-SUMMARY.md 'Notable Live-Environment Finding'), an environment condition unchanged since the prior verification."
  - test: "Run `WEBSPACES_PROTON_LIVE_IT=1 PROTON_BRIDGE_ADDR=<addr> PROTON_BRIDGE_USER=<user> PROTON_BRIDGE_PASS=<pass> go test -run TestSeenFlagUnchanged_LiveBridge -v ./plugins/proton/...` against the real Bridge account once its credential issue is corrected."
    expected: "The test passes, directly proving SRC-01's second success criterion (\\Seen flag unchanged across a full Match+Fetch cycle) against the real mailbox, not just the no-live-Bridge wire-transcript proxy for it."
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
**Verified:** 2026-07-31T15:05:27Z
**Status:** gaps_found
**Re-verification:** Yes — after gap closure plan 03-05 (INTERNALDATE reaches Item.TimestampUnix, the empty-Message-Id skip is now logged, TLS-comment hygiene)

## Goal Achievement

### Observable Truths (mapped to ROADMAP Success Criteria)

| # | Truth (ROADMAP SC) | Status | Evidence |
|---|---------|--------|----------|
| 1 | Emails appear in the stream with sender, subject, and date, and render their body inline in the detail pane | ✗ FAILED | Sender, subject, and date now all work: `plugins/proton/plugin.go`'s `toItem()` (line ~336-358) now assigns a guarded `Item.TimestampUnix` from `matched.internalDate`, populated from `msg.InternalDate` in `Match`'s fetch loop (line 188). Confirmed by reading the code and by `TestMatch_ItemTimestampIsInternalDate`/`TestToItem_ZeroInternalDateYieldsZeroTimestamp` passing, and by tracing the value through `kernel/item/item.go FromProto` → `kernel/index/store.go`'s `ORDER BY items.timestamp_unix DESC` → `web/src/lib/format.ts`'s UTC-pinned `formatItemDate`. **But "render their body inline in the detail pane" is newly found broken**: `plugins/proton/plugin.go`'s `mailboxCache` — the only state `Fetch` uses to resolve a `source_id` to a mailbox — is replaced (not merged) on every `Match` call, and `kernel/correlate/correlate.go`'s `SyncSource` calls `Match` once per configured webspace within one sync cycle against the same long-lived plugin instance. With 2+ webspaces both matching Proton mail, opening an email belonging to any webspace but the last one synced in that cycle returns `codes.NotFound` from `fetchFull`, even though the item is visible in the stream. See gap below (CR-01). A related, lower-severity issue also remains: `DetailPane.svelte`'s local `formatDate()` omits the UTC pin `format.ts`'s tested `formatItemDate()` enforces, so the same item's date can render on a different calendar day in the detail pane vs. the stream row. |
| 2 | Reading an email never marks it read in Proton — proven by an automated test asserting `\Seen` is unchanged after a full sync and a detail fetch | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | Unchanged from the prior verification: `readonly_test.go`'s AST scan and `imap_transcript_test.go`'s wire-transcript test still prove only `EXAMINE`/`BODY.PEEK[` are ever issued, no mutating IMAP commands. `TestSeenFlagUnchanged_LiveBridge` is implemented and still skips cleanly (confirmed this session — SKIP, not FAIL) but has never run to a PASS against a real Bridge account; routed to human verification, same environment blocker as before (Bridge credential rejection, unchanged). |
| 3 | An email carrying several Proton labels appears exactly once, deduped by Message-ID | ✓ VERIFIED | `TestIMAPTranscript_ExamineAndPeekOnly` (unchanged) and the new `TestMatch_ItemTimestampIsInternalDate` (which also asserts exactly one item results from two matching mailboxes sharing one Message-ID) both pass. |
| 4 | User can type a query inside a webspace and get ranked, clickable matches across every source | ✓ VERIFIED | Unaffected by this plan's changes. `kernel/index/store.go`'s `Search` and `kernel/httpapi/search.go`'s `SearchHandler` remain fully unit-tested and wired to `SearchBox.svelte`/`SearchResults.svelte`. `go test ./kernel/index/... ./kernel/httpapi/...` passes. |
| 5 | Bridge unreachable produces a specific, timely, actionable health error rather than a hang | ✓ VERIFIED | Unaffected by this plan's changes. `Health()` still uses a bounded 5s dial+login and returns `Reachable:false` + a specific `LastError`. Previously confirmed live (03-01-SUMMARY.md). |

**Score:** 3/5 truths verified (1 present-but-behavior-unverified, 1 failed)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `plugins/proton/plugin.go` `matched.internalDate` / `toItem` `TimestampUnix` | INTERNALDATE reaches the item's primary timestamp | ✓ VERIFIED | Field present (line ~53), populated at line 188, assigned with a zero-guard at line ~357. `go test -run 'TestMatch_ItemTimestampIsInternalDate|TestToItem_ZeroInternalDateYieldsZeroTimestamp'` passes. |
| `plugins/proton/plugin.go` `SourcePlugin.logOut` / skip log line | Counted empty-Message-Id skip is also logged, count-only | ✓ VERIFIED | `logOut io.Writer` field present, initialized to `os.Stderr`; guarded `fmt.Fprintf` at line ~211 emits `"webspaces-plugin-proton: match: skipped %d message(s) with no Message-Id header"`. `TestMatch_EmptyMessageIDSkipIsLogged` passes and asserts the seeded subject never leaks into the log. |
| `plugins/proton/client.go` `bridgeCertServerName` comment | No textual match for the TLS opt-out field, TLS config unchanged | ✓ VERIFIED | `grep -rq InsecureSkipVerify plugins/proton/` finds nothing; `RootCAs`/`ServerName` still present and unchanged. |
| `plugins/proton/plugin.go` `mailboxCache` / `setMailboxCache` | Fetch can resolve any synced item's mailbox regardless of which webspace was synced last | ✗ **STUB (logic bug)** | `setMailboxCache` replaces the whole map on every `Match` call instead of merging; confirmed broken for any deployment with 2+ webspaces sharing the Proton source (see gap CR-01). No test exercises this path. |
| `web/src/lib/components/DetailPane.svelte` date rendering | Detail pane date matches the stream row's date (both UTC-pinned) | ⚠️ ORPHANED-FROM-FIX | Local `formatDate()` does not use the shared `formatItemDate` from `$lib/format`; `format.ts`'s UTC pin is not applied here, so the two renderers can disagree for viewers west of UTC. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `plugins/proton/plugin.go` Match fetch loop | `plugins/proton/plugin.go` toItem | `matched.internalDate` | ✓ WIRED | Populated at line 188, read at line ~337 |
| `plugins/proton/plugin.go` toItem | `kernel/index/store.go` stream ordering | `Item.TimestampUnix` → `FromProto` (no fallback) → `items.timestamp_unix` → `ORDER BY` | ✓ WIRED | Traced by reading `kernel/item/item.go` and `kernel/index/store.go`; both unmodified by this plan and already covered by `TestStreamItems_*` |
| `plugins/proton/plugin.go` Match | kernel log stream | `p.logOut` (stderr, forwarded by go-plugin) | ✓ WIRED | `TestMatch_EmptyMessageIDSkipIsLogged` passes |
| `kernel/correlate/correlate.go` SyncSource | `plugins/proton/plugin.go` Match/Fetch | one shared plugin instance, `Match` called once per configured webspace, `Fetch` called later against the same instance | ✗ **NOT SAFELY WIRED** | `mailboxCache` is not scoped per webspace and is replaced (not merged) per `Match` call — see gap CR-01. The persistence side of this same loop (`ReplaceWebspaceSourceItems`) IS correctly scoped per webspace; only the plugin's in-memory `Fetch` cache is not. |
| `web/src/lib/components/DetailPane.svelte` | `web/src/lib/format.ts` | Shared, UTC-pinned `formatItemDate` | ✗ NOT WIRED | `DetailPane.svelte` defines and uses its own local `formatDate` instead of importing `formatItemDate` |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
|----------|---------------|--------|---------------------|--------|
| `web/src/lib/components/StreamRow.svelte` | `item.timestamp_unix` | `plugins/proton/plugin.go toItem` → gRPC `Item.TimestampUnix` → `kernel/item/item.go FromProto` → `items.timestamp_unix` column | ✓ FLOWING | Real value now (INTERNALDATE seconds), not a hardcoded/zero fallback; confirmed by passing regression tests and by reading the full chain |
| `web/src/lib/components/DetailPane.svelte` | body content (`Fetch` response) | `plugins/proton/plugin.go fetchFull` → `mailboxForSourceID` lookup against `mailboxCache` | ⚠️ CONDITIONALLY DISCONNECTED | Flows correctly for the webspace processed last in a sync cycle; returns `codes.NotFound` for any other webspace sharing the same Proton source, because `mailboxCache` only ever holds the last `Match` call's discoveries (see gap CR-01) |

### Behavioral Spot-Checks / Test Execution

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Proton module tests | `cd plugins/proton && go test ./... -v -count=1` | 14 pass, 1 skip (`TestSeenFlagUnchanged_LiveBridge`, environment-blocked) | ✓ PASS |
| Repo build/vet/test | `CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go vet ./... && CGO_ENABLED=0 go test ./... -count=1` | clean, all packages `ok` or `[no test files]` | ✓ PASS |
| Sibling plugin regression | `cd plugins/paperless && go test ./... -count=1` | pass | ✓ PASS |
| Frontend tests | `npm --prefix web run test` | 69/69 pass | ✓ PASS |
| `InsecureSkipVerify` textual match | `grep -rq InsecureSkipVerify plugins/proton/` | no match anywhere (code or comments) | ✓ PASS |
| Debt markers | `grep -n 'TBD\|FIXME\|XXX' -r plugins/proton/*.go` | none found | ✓ PASS |
| Multi-webspace Fetch regression (CR-01) | *(no such test exists)* | not exercised by any test in the repo | ✗ MISSING COVERAGE |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| SRC-01 | 03-01, 03-02, 03-05 | Email plugin (IMAP) works against Proton Mail Bridge; BODY.PEEK; matches keyword against folders/labels; dedups by Message-ID | ⚠️ PARTIAL | Read-only guarantee, dedup, TLS pinning, health error, and the date field are all now verified and correct. The requirement's implicit "readable" contract (body reachable in the detail pane) is not reliably met across the product's designed multi-webspace usage — see CR-01 gap above. REQUIREMENTS.md still shows `[ ]` unchecked for SRC-01, correctly reflecting this. |
| KERN-05 | 03-03, 03-04 | User can full-text search within a webspace (FTS5) | ✓ SATISFIED | Unchanged from prior verification — fully implemented, tested end to end. REQUIREMENTS.md still shows `[ ]` unchecked and "Pending" in the traceability table; this is bookkeeping the SUMMARY explicitly and deliberately left for the seal/ship step, not a code gap. |

No orphaned requirements found — both IDs declared in this phase's plans (SRC-01, KERN-05) match ROADMAP.md's Phase 3 requirements list exactly.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `plugins/proton/plugin.go` | ~381-384 | `setMailboxCache` replaces the whole map instead of merging, called once per `Match` invocation while `Match` is itself called once per configured webspace against one shared plugin instance | 🛑 Blocker | Breaks Fetch/detail-pane body rendering for every webspace but the last-synced one whenever 2+ webspaces share the Proton source — see gap CR-01 above |
| `web/src/lib/components/DetailPane.svelte` | ~33-40 | Local `formatDate()` reimplements date formatting without the `timeZone: 'UTC'` pin `web/src/lib/format.ts`'s tested `formatItemDate()` enforces | ⚠️ Warning | Same item's date can differ between the stream row and the detail pane for viewers west of UTC |

No debt markers (`TBD`/`FIXME`/`XXX`) found in any file this phase modified. No unresolved `TODO`/`HACK`/`PLACEHOLDER` markers found.

### Human Verification Required

See frontmatter `human_verification` for the four items in full detail (unchanged in substance from the prior verification, all still blocked on the same Proton Bridge account credential rejection documented in 03-01-SUMMARY.md). Summary:

1. **Live browser walkthrough of a real email in the stream and detail pane** — now additionally worth checking against a second, non-last-synced webspace once the CR-01 gap is closed, to directly confirm the fix.
2. **Run `TestSeenFlagUnchanged_LiveBridge` to a PASS against the real Bridge** — unchanged blocker.
3. **Confirm in the real Proton client that an opened email stays unread** — unchanged blocker.
4. **Live browser walkthrough of the search UI** — unchanged, not run this session.

### Gaps Summary

The one BLOCKER confirmed in the prior verification (`Item.TimestampUnix` never set from INTERNALDATE) is genuinely closed by 03-05: `matched.internalDate` now carries the fetched value through to a guarded `toItem()` assignment, proven by two new passing regression tests and confirmed by re-reading the code end to end through `kernel/item/item.go` and `kernel/index/store.go`. The related minor issue (counted-but-unlogged empty-Message-Id skip) and the TLS-comment hygiene row are also both genuinely closed, each confirmed by reading the code and by a passing test or grep.

However, this re-verification surfaces a **new, confirmed BLOCKER** that was not part of the prior gap list: `plugins/proton/plugin.go`'s `mailboxCache` is replaced, not merged, on every `Match` call, while `kernel/correlate/correlate.go`'s `SyncSource` calls `Match` once per configured webspace within a single sync cycle against the one long-lived Proton plugin instance. This means `Fetch` (and therefore "render their body inline in the detail pane," this phase's own ROADMAP Success Criterion 1) only reliably works for whichever webspace happened to be processed last in that cycle — a real, deterministic, code-verifiable defect, not a live-Bridge-environment limitation, that reproduces on any deployment with two or more webspaces sharing the Proton source (the project's documented core use case). This was first surfaced by an independent code review pass (03-REVIEW.md, CR-01) committed after 03-05's own commits, and is independently re-confirmed here by re-reading `plugin.go`, `correlate.go`, and `pluginhost/host.go` directly. No test in the repository exercises this path. A second, lower-severity issue from the same review (`DetailPane.svelte`'s un-pinned local date formatter, WR-01) also remains unaddressed.

**Recommended fix for the blocking gap:** change `setMailboxCache` to merge entries into the existing map under lock instead of replacing it wholesale, and remove the "reset to empty map" call on `Match`'s zero-mailboxes-matched path (that path has nothing new for its own webspace but must not erase what an earlier webspace's `Match` call in the same cycle already contributed). Add a regression test that runs two sequential `Match` calls against one plugin instance for disjoint keyword sets, then a `Fetch` for a `source_id` only the first call discovered, asserting it still resolves. Separately, have `DetailPane.svelte` import and use the shared `formatItemDate` from `$lib/format` instead of its own local `formatDate`.

---

*Verified: 2026-07-31T15:05:27Z*
*Verifier: Claude (gsd-verifier)*
