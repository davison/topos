---
phase: 03-email-in-the-webspace
verified: 2026-07-31T02:35:53Z
status: gaps_found
score: 3/5 must-haves verified
behavior_unverified: 1
overrides_applied: 0
gaps:
  - truth: "Emails in Proton folders or labels matching the webspace keyword appear in the stream with sender, subject, and date, and render their body inline in the detail pane (ROADMAP SC1; 03-01-PLAN.md must_have: 'the date is the message's INTERNALDATE')"
    status: failed
    reason: >
      plugins/proton/plugin.go's toItem() (the only place a webspacesv1.Item
      is constructed for this plugin) never sets Item.TimestampUnix. Match's
      FETCH explicitly requests imap.FetchInternalDate (plugin.go line 151),
      but the per-message `matched` struct (line 42-46) has no field to hold
      it, so the fetched INTERNALDATE is discarded and never reaches toItem().
      Every Proton item is therefore persisted with TimestampUnix == 0.
      kernel/item/item.go's FromProto takes TimestampUnix directly from the
      plugin-supplied proto field with no fallback, and
      kernel/correlate/correlate.go's validateCorrelatedItem does not check
      it, so the zero value passes through silently into the index. On the
      frontend, both web/src/lib/format.ts's formatItemDate() and
      web/src/lib/components/DetailPane.svelte's local formatDate() treat a
      falsy timestamp as "no date" and render an empty string — so every
      Proton email's row and detail pane show no date at all, not merely a
      wrong one. Separately, kernel/index/store.go's stream query orders
      `ORDER BY items.timestamp_unix DESC, items.secondary_timestamp_unix
      DESC` — every email ties at timestamp_unix=0 and sorts as a block at
      the oldest end of the stream, not interleaved chronologically with
      paperless/SilverBullet items as the must-have and ROADMAP SC1 require.
      No automated test in the phase (imap_transcript_test.go asserts item
      count and labels only) or any SUMMARY.md caught or documented this as a
      deviation.
    artifacts:
      - path: "plugins/proton/plugin.go"
        issue: "matched struct (~line 42) never captures msg.InternalDate from the Match FETCH loop (~line 156-174); toItem() (~line 304-343) never assigns Item.TimestampUnix — SecondaryTimestampUnix (envelope Date) is set, but the primary TimestampUnix (INTERNALDATE) is silently omitted."
    missing:
      - "Add an internalDate time.Time field to the matched struct and populate it from msg.InternalDate in Match's fetch loop"
      - "Set webspacesv1.Item.TimestampUnix = m.internalDate.Unix() in toItem()"
      - "A regression test (extending imap_transcript_test.go or a new unit test) that seeds a message with a known INTERNALDATE and asserts the resulting Item.TimestampUnix is non-zero and matches it — the existing transcript test does not check this field at all"
    related_minor_issue: >
      In the same function, `skippedNoMessageID` (Match's counted skip of
      messages with an empty Message-Id) is incremented but never logged —
      `_ = skippedNoMessageID // counted for the Match log line below`
      discards it with no log line following, so the 03-01 must-have
      ("skipped ... with a counted, logged skip") is only half-implemented
      (counted, not logged). Low severity relative to the TimestampUnix gap
      above; noted for the same fix pass.
human_verification:
  - test: "Open a webspace with a real, currently-reachable Proton Bridge account configured. Confirm at least one real Proton email appears in the stream interleaved with paperless/SilverBullet items, showing sender before the date, with the subject as the row title. Click it: the detail pane opens and shows the message's body (plain text, or formatted HTML with headings/links/colors if the message has an HTML part)."
    expected: "The email is visible, correctly dated once the TimestampUnix gap above is fixed, and its body renders in the detail pane. This is 03-01-PLAN.md Task 3's and 03-02-PLAN.md Task 1's own human-check step, never run this session per WINDOWS.md item 1 — no live Bridge email was available (Bridge account credential rejected LOGIN, see 03-01-SUMMARY.md Deviations)."
    why_human: "Requires a running kernel/webUI against a real, currently-authenticating Proton Mail Bridge account and visual confirmation of rendered HTML/CSS in a browser — not mechanically checkable from source alone."
  - test: "Run `WEBSPACES_PROTON_LIVE_IT=1 PROTON_BRIDGE_ADDR=<addr> PROTON_BRIDGE_USER=<user> PROTON_BRIDGE_PASS=<pass> go test -run TestSeenFlagUnchanged_LiveBridge -v ./plugins/proton/...` against the real Bridge account once its credential issue is corrected."
    expected: "The test passes, directly proving SRC-01's second success criterion (\\Seen flag unchanged across a full Match+Fetch cycle) against the real mailbox, not just the no-live-Bridge wire-transcript proxy for it."
    why_human: "Requires a live, currently-authenticating Bridge connection and real mailbox state; WINDOWS.md item 2 records this test as implemented and skipping cleanly, but never executed to a PASS this session — the Bridge account's 'no such user' LOGIN rejection (and subsequent rate-limiting) documented in 03-01-SUMMARY.md blocked it, and retrying was deliberately avoided to prevent extending Bridge's own lockout."
  - test: "After running a webspaces sync and opening an email in the detail pane, check the same email in the real Proton web or mobile client and confirm it is still shown as unread there."
    expected: "The email remains unread in Proton's own client — the direct, human-observable proof of the never-mark-read guarantee end to end."
    why_human: "Requires a live Bridge account, a real sync, and cross-checking read/unread state in Proton's own UI; WINDOWS.md item 3 records this as not run this session, blocked on the same Bridge credential issue."
  - test: "Open a webspace in the browser, type a word present in a document, a note, and an email into the search box, and confirm ranked cross-source results appear with the matched word emboldened, a result opens the detail pane on click, clearing restores the unfiltered stream, and a nonsense or malformed (lone double-quote) query shows the no-matches state rather than an error."
    expected: "All of 03-04-PLAN.md's Task 1 human-check behaviors hold visually in a running browser."
    why_human: "Visual/interactive confirmation of debounce feel, snippet emphasis rendering, and detail-pane opening — beyond what the passing unit tests over parseSnippet/searchVariant can prove; not run this session per 03-04-SUMMARY.md (no running kernel/webUI session with live data was exercised for this human-check)."
---

# Phase 3: Email in the Webspace Verification Report

**Phase Goal:** User's Proton mail for a topic appears in the webspace stream and stays readable there, and the volume it brings is navigable by searching within the webspace
**Verified:** 2026-07-31T02:35:53Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (mapped to ROADMAP Success Criteria)

| # | Truth (ROADMAP SC) | Status | Evidence |
|---|---------|--------|----------|
| 1 | Emails appear in the stream with sender, subject, and date, and render their body inline in the detail pane | FAILED | Sender and subject work (`GroupLabel`/`Title` set, `StreamRow.svelte`/`DetailPane.svelte` render `item.group_label`). **Date does not work**: `plugins/proton/plugin.go`'s `toItem()` never sets `Item.TimestampUnix` despite fetching `INTERNALDATE` via IMAP — see gap below. HTML body-in-detail-pane is implemented and unit-tested (`render_test.go`, `bluemonday` sanitizer, `text/html` Fetch path) but not live-verified in a browser this session (human_needed). |
| 2 | Reading an email never marks it read in Proton — proven by an automated test asserting `\Seen` is unchanged after a full sync and a detail fetch | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | Strong indirect proof exists and passes without a live Bridge: `readonly_test.go`'s build-failing AST scan (with a negative control) forbids any IMAP-mutating identifier in non-test source, and `imap_transcript_test.go`'s wire-level transcript test proves a full Describe/Match/Fetch/Health cycle issues only `EXAMINE `/`BODY.PEEK[` and none of `SELECT /STORE /EXPUNGE/APPEND /COPY/MOVE /DELETE `. The literal test named by this criterion, `TestSeenFlagUnchanged_LiveBridge`, is implemented and skips cleanly but has never been run to a PASS against the real Bridge (WINDOWS.md item 2, blocked on a Bridge-account credential rejection, not a code defect) — routed to human verification below rather than failed, per this session's explicit instruction. |
| 3 | An email carrying several Proton labels appears exactly once, deduped by Message-ID | ✓ VERIFIED | `imap_transcript_test.go`'s `TestIMAPTranscript_ExamineAndPeekOnly` seeds two distinctly-named mailboxes (`Labels/AlphaTeam`, `Labels/BetaTeam`) sharing one Message-ID and asserts `Match` returns exactly one item carrying both leaf names in `labels`. `go test ./plugins/proton/...` passes. |
| 4 | User can type a query inside a webspace and get ranked, clickable matches across every source | ✓ VERIFIED | `kernel/index/store.go`'s `Search` (bm25-ranked, webspace-scoped, snippet-carrying) and `kernel/httpapi/search.go`'s `SearchHandler` are fully unit-tested (`TestSearch_*`, `TestSearchHandler_*`, all passing) and wired to a real browser UI (`SearchBox.svelte`, `SearchResults.svelte`, `+page.svelte`) with its own extensive unit-tested pure-decision layer (`parseSnippet`, `searchVariant`, `searchCopy`). Search itself does not depend on Proton being reachable — it reads whatever is indexed. |
| 5 | Bridge unreachable produces a specific, timely, actionable health error rather than a hang | ✓ VERIFIED | `Health()` (`plugins/proton/plugin.go`) uses a bounded 5s dial+login and returns `Reachable:false` + a specific `LastError`, never a gRPC error. Confirmed **live** in 03-01-SUMMARY.md: the real Bridge account rejected LOGIN ("no such user"), and `GET /api/sources` correctly reported `"reachable":false` with a specific, actionable `last_error` — a real end-to-end confirmation of this exact criterion, independent of the credential issue itself. |

**Score:** 3/5 truths verified (1 present-but-behavior-unverified, 1 failed)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `plugins/proton/main.go`, `client.go`, `plugin.go`, `body.go` | Full `sdk.SourcePlugin` over IMAP | ✓ VERIFIED | All present, build clean (`CGO_ENABLED=0 go build/vet ./...` both pass in module and repo root), 15 tests pass in `plugins/proton` |
| `kernel/config/types.go` `Username`/`WebmailBaseURL` | New Source fields | ✓ VERIFIED | Present with correct `toml:"username,omitempty"` / `toml:"webmail_base_url,omitempty"` tags |
| `kernel/pluginhost/host.go` config plumbing | `username`/`webmail_base_url` in `WEBSPACES_SOURCE_CONFIG` | ✓ VERIFIED | `grep` confirms `"username": src.Username` and `"webmail_base_url": src.WebmailBaseURL` present |
| `web/src/lib/components/StreamRow.svelte` sender rendering | `item.group_label` first in metadata strip | ✓ VERIFIED | Present, guarded by `{#if}`, before the date span |
| `web/src/lib/components/DetailPane.svelte` sender rendering | `item.group_label` line under title | ✓ VERIFIED | Present, truncated, `title=` attribute, before date/labels row |
| `kernel/index/schema.go` FTS5 table + triggers | `items_fts` + 3 sync triggers | ✓ VERIFIED | `CREATE VIRTUAL TABLE IF NOT EXISTS items_fts`, `items_ai`/`items_ad`/`items_au` all present |
| `kernel/index/store.go` `Search`, backfill | bm25-ranked search + first-open backfill | ✓ VERIFIED | `TestSearch_*` (6 tests) and `TestBackfill_ReopeningAPreexistingIndexFindsItsItems` all pass |
| `kernel/httpapi/search.go` `SearchHandler` | Store-only handler, no plugin import | ✓ VERIFIED | Imports only `kernel/index`, `chi`, `net/http`, `strings` — structurally cannot reach a plugin |
| `web/src/lib/components/SearchBox.svelte`, `SearchResults.svelte` | Debounced search UI | ✓ VERIFIED | Present, correct copy strings, no `disabled=`, no `{@html}`, no client-side `.sort()`/`filterItemsBySource` |
| `docs/api.md` search section | Documented route contract | ✓ VERIFIED | Search route section present; "not yet implemented" bullet removed |
| `plugins/proton/plugin.go` item construction | `Item.TimestampUnix` set from INTERNALDATE | ✗ **STUB (silent)** | Field is never assigned — see gap above. All surrounding fields (Title, GroupLabel, Fidelity, DeepLink, Labels, Provenance) ARE correctly populated, so this is a partial, not total, defect in one otherwise-complete function |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `kernel/config/types.go` | `kernel/pluginhost/host.go` | `launch()` marshals `Username`/`WebmailBaseURL` into env JSON | ✓ WIRED | grep confirms both keys present |
| `plugins/proton/plugin.go` | `kernel/correlate/correlate.go` | Items satisfy `validateCorrelatedItem` (fidelity + deep_link) | ✓ WIRED (but see artifact gap above — validation does not catch the missing timestamp, since it only checks fidelity/deep_link) |
| `web/src/lib/components/StreamRow.svelte` | `kernel/httpapi/stream.go` | Renders `item.group_label` | ✓ WIRED |
| `plugins/proton/plugin.go` | `kernel/httpapi/item.go` | `Fetch` returns `MimeType: "text/html"`, served through the existing allowlisted rendition route | ✓ WIRED (code-level; visual confirmation pending, see human verification) |
| `kernel/index/schema.go` | `kernel/index/store.go` | AFTER INSERT/UPDATE/DELETE triggers keep `items_fts` in sync | ✓ WIRED, tested |
| `kernel/httpapi/routes.go` | `kernel/httpapi/search.go` | `r.Get("/api/webspaces/{webspace}/search", SearchHandler(store))` | ✓ WIRED |
| `web/src/lib/api.ts` | `kernel/httpapi/search.go` | `searchWebspace()` → `GET .../search?q=` | ✓ WIRED |
| `web/src/lib/components/SearchResults.svelte` | `web/src/lib/components/StreamRow.svelte` | Renders one `StreamRow` per result with `snippet` prop | ✓ WIRED |

### Behavioral Spot-Checks / Test Execution

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Repo build | `CGO_ENABLED=0 go build ./...` | clean | ✓ PASS |
| Repo tests | `CGO_ENABLED=0 go test ./... -count=1` | all packages ok | ✓ PASS |
| Proton module build/vet | `cd plugins/proton && go build ./... && go vet ./...` | clean | ✓ PASS |
| Proton module tests | `cd plugins/proton && go test ./... -v` | 11 pass, 1 skip (live Bridge, expected) | ✓ PASS |
| Read-only AST scan | `go test -run TestPluginIssuesNoIMAPMutatingCommands` | pass, negative control fires | ✓ PASS |
| Wire transcript | `go test -run TestIMAPTranscript_ExamineAndPeekOnly` | pass — EXAMINE/BODY.PEEK present, no mutating substrings | ✓ PASS |
| Outbound host allowlist | `go test -run TestAllowHost_PredicateTable` | pass | ✓ PASS |
| Egress/read-only audit | `go test ./internal/audit/...` | pass | ✓ PASS |
| Sibling plugin regression | `cd plugins/paperless && go test ./...` | pass | ✓ PASS |
| Search store + handler tests | `go test ./kernel/index/... ./kernel/httpapi/... -run 'TestSearch\|TestBackfill\|TestSearchHandler'` | all pass | ✓ PASS |
| Frontend typecheck | `npm --prefix web run check` | 0 errors, 1 benign Svelte 5 reactivity warning in `SearchBox.svelte` (unrelated to correctness — an `$effect` already re-syncs the flagged value) | ✓ PASS |
| Frontend tests | `npm --prefix web run test` | 69/69 pass | ✓ PASS |
| Frontend build + stylesheet guard | `npm --prefix web run build && ./scripts/assert-stylesheet.sh` | exit 0 | ✓ PASS |
| `InsecureSkipVerify` never present | `! grep -rq InsecureSkipVerify plugins/proton/` (plan's own acceptance criterion, "including in comments") | **grep matches** `client.go:56`, inside a comment reading "...no InsecureSkipVerify anywhere in this file" | ⚠️ Minor — plan's own literal acceptance criterion technically fails on a reassuring comment mentioning the term, not an actual verification bypass. TLS verification is genuinely never disabled in the code (confirmed by reading `client.go` in full). Not counted as a blocker; noted for hygiene. |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| SRC-01 | 03-01, 03-02 | Email plugin (IMAP) works against Proton Mail Bridge; BODY.PEEK; matches keyword against folders/labels; dedups by Message-ID | ⚠️ PARTIAL | Read-only guarantee, dedup, TLS pinning, health error all verified; the date-field bug (above) means the "appears in the stream with ... date" clause of this requirement's own text is not yet met. REQUIREMENTS.md marks SRC-01 `[x]` complete — premature given the date bug. |
| KERN-05 | 03-03, 03-04 | User can full-text search within a webspace (FTS5) | ✓ SATISFIED | Fully implemented, tested end to end (index → HTTP → UI). REQUIREMENTS.md still shows `[ ]` unchecked for KERN-05 (line 13) and the requirement-mapping table (line 74) still says "Pending" despite ROADMAP.md marking Phase 3 complete — a documentation bookkeeping gap, not a code gap; recommend updating REQUIREMENTS.md's checkbox and status column once the gap above is closed. |

No orphaned requirements found — both IDs declared in this phase's plans (SRC-01 across 03-01/03-02, KERN-05 across 03-03/03-04) match ROADMAP.md's Phase 3 requirements list exactly.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `plugins/proton/plugin.go` | ~189 | `_ = skippedNoMessageID // counted for the Match log line below` — comment promises a log line that does not exist anywhere in the file | ℹ️ Info | The must-have "a counted, logged skip" is half-met (counted, not logged); low impact since no other plugin in this repo does structured logging either, but the comment is misleading about what the code actually does |
| `plugins/proton/client.go` | 56 | Comment contains the literal string `InsecureSkipVerify` (to reassure the reader it's absent) | ℹ️ Info | Technically fails the plan's own mechanical acceptance grep (`! grep -rq 'InsecureSkipVerify' plugins/proton/`), which explicitly said "including in comments" — no actual security impact, TLS verification is genuinely intact |

No debt markers (`TBD`/`FIXME`/`XXX`) found in any file this phase modified. No unresolved `TODO`/`HACK`/`PLACEHOLDER` markers found (the two "placeholder" hits are the legitimate `noSubjectPlaceholder` constant and an HTML input placeholder, not debt markers).

### Human Verification Required

See frontmatter `human_verification` for the four items in full detail. Summary:

1. **Live browser walkthrough of a real email in the stream and detail pane** (WINDOWS.md item 1) — blocked on the same Bridge-account credential rejection documented in 03-01-SUMMARY.md.
2. **Run `TestSeenFlagUnchanged_LiveBridge` to a PASS against the real Bridge** (WINDOWS.md item 2) — this is the literal test named by ROADMAP SC2; it exists, compiles, and skips cleanly, but has never executed successfully.
3. **Confirm in the real Proton client that an opened email stays unread** (WINDOWS.md item 3) — same credential blocker.
4. **Live browser walkthrough of the search UI** (03-04-PLAN.md Task 1 human-check) — not run this session.

### Gaps Summary

One confirmed, code-level BLOCKER: `plugins/proton/plugin.go`'s `toItem()` fetches each message's `INTERNALDATE` over IMAP but never stores or forwards it — `Item.TimestampUnix` is always zero for every Proton email. This is independent of the Bridge-credential issue that blocks the three live/human-check items above (it would reproduce identically against a working Bridge account, since it's a pure code omission, not an environment problem). Concretely: every Proton email row and detail pane will show no date at all, and the whole-stream chronological ordering (`ORDER BY items.timestamp_unix DESC`) will bucket every email at the oldest end rather than interleaving it correctly with paperless-ngx documents and SilverBullet pages — directly failing ROADMAP Success Criterion 1's "appear in the stream with sender, subject, and date" and its "interleaved in the same chronological stream" must-have. No automated test in either 03-01 or 03-02 asserts a non-zero `TimestampUnix`, so this shipped without any test failing.

A second, much lower-severity issue in the same function: the counted skip of messages with an empty Message-Id is never actually logged despite a comment claiming it is (must-have partially met).

All other success criteria and must-haves either fully pass automated verification or are legitimately blocked on the pre-existing, already-documented Bridge account credential issue (not a code defect) — those are routed to human verification rather than treated as gaps, per this session's explicit guidance.

**Recommended fix for the blocking gap:** add an `internalDate time.Time` field to `plugins/proton/plugin.go`'s `matched` struct, populate it from `msg.InternalDate` in `Match`'s fetch loop (the value is already being fetched via `imap.FetchInternalDate`, just discarded), and set `Item.TimestampUnix: m.internalDate.Unix()` in `toItem()`. Add a regression assertion (extending `imap_transcript_test.go`, which already has full plumbing for a Match/Fetch cycle) that a Match'd item's `TimestampUnix` is non-zero and matches the seeded message's date, so this class of omission cannot silently recur.

---

*Verified: 2026-07-31T02:35:53Z*
*Verifier: Claude (gsd-verifier)*
