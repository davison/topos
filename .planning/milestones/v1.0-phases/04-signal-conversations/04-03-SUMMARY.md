---
phase: 04-signal-conversations
plan: 03
subsystem: sync
tags: [go, sqlite, sqlcipher, signal, html, bluemonday, chat-transcript]

# Dependency graph
requires:
  - phase: 04-signal-conversations
    provides: "04-01/04-02's working Signal plugin — SQLCipher driver, dsn.go, schemaguard.go, dual-shape key resolution, plugin.go's Match/Health, digest.go's day-grouping, deeplink.go's builder — this plan completes the Fetch stub and validates the deep link hands-on"
provides:
  - "messageRecord (message.go): the single parsed-once view of a Signal message's full richness (deleted, edited, attachments, reactions, quote) shared by the digest tail snippet and the transcript renderer"
  - "The chat-transcript HTML renderer (render.go: renderTranscript/WrapDocument/buildMessageRuns) — explicitly the source-agnostic renderer Phase 5's WhatsApp plugin reuses"
  - "Fetch's FULL/PREVIEW implementation (plugin.go's fetchTranscript) — a Signal digest now opens into a real, sanitized chat transcript through DetailPane's existing html iframe branch, zero proto/frontend change"
  - "sgnl:// deep link validated hands-on against the installed, running Signal Desktop (both the bare and E.164-contact forms)"
  - "docs/plugin-contract.md's `path` local-path source-config key, published for third-party plugin authors"
affects: [phase-05-whatsapp]

# Actuals (#2632)
actuals:
  tokens: 15800
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "messageRecord as the single parse-once richness structure — both the compact digest snippet (digest.go) and the full transcript (render.go) read the SAME parsed record, never re-parsing the message row twice"
    - "Attachments and reactions read from Signal Desktop's own dedicated SQL tables (message_attachments, reactions), never from the message row's json blob — a ground-truth correction to 04-RESEARCH.md's illustrative 'everything richer lives in the blob' framing, confirmed by direct schema introspection of the real, live database"
    - "Sanitize-then-wrap per text field, not per document: every untrusted string (body, quote excerpt, attachment filename, reactor/sender name) is sanitized individually via signalTranscriptSanitizePolicy before being concatenated into the assembled transcript markup, mirroring plugins/proton/body.go's discipline"
    - "Own-vs-other ownership signalled by alignment/background tier ONLY — the account's own messages are identified purely by the existing 'You' sender-name convention (ownSenderLabel), never by a new colour or a raw service id comparison"

key-files:
  created:
    - plugins/signal/message.go
    - plugins/signal/message_test.go
    - plugins/signal/render.go
    - plugins/signal/render_test.go
    - plugins/signal/fetch_test.go
    - plugins/signal/deeplink_test.go
    - plugins/signal/README.md
  modified:
    - plugins/signal/digest.go
    - plugins/signal/digest_test.go
    - plugins/signal/plugin.go
    - plugins/signal/byte_identical_test.go
    - plugins/signal/deeplink.go
    - docs/plugin-contract.md

key-decisions:
  - "Attachments and reactions are read from Signal Desktop's own dedicated message_attachments/reactions SQL tables, never from the message row's json blob — confirmed by direct, hands-on introspection of the real, live ~/.config/Signal/sql/db.sqlite (53,346 real messages), which found NO 'attachments' key in any sampled message blob despite hasAttachments=1, while message_attachments/reactions are real, indexed, populated tables. This corrects 04-RESEARCH.md's illustrative 'everything richer than plain body text lives only in that blob' framing for these two fields specifically (deletedForEveryone, editHistory and quote genuinely do live in the blob, confirmed the same way)."
  - "message_attachments.attachmentType is filtered to {attachment, sticker, contact} — excluding 'preview' (a link-preview thumbnail describing a URL in the body, not a file the sender attached), 'quote' (a thumbnail copied from the message being replied to, describing that OTHER message) and 'long-message' (Signal's own body-overflow mechanism, not a user-facing file). Confirmed against real data: attachmentType breaks down as attachment=15214, quote=1189, preview=696, sticker=19, contact=1 across the real database."
  - "Fetch's FULL and PREVIEW variants return the byte-identical wrapped transcript — a Signal digest has no separate 'richer inline preview vs extracted text' distinction the way an email's plain-text-vs-HTML choice does, so both variants share one code path (fetchTranscript)."
  - "Edited-message '(edited)' suffix is co-located with the timestamp only on a run's LAST bubble (the only bubble that ever carries a timestamp, per the UI contract's own 'once at the end of the run's last bubble' rule); a non-last edited bubble renders the suffix on its own line instead of leaving it unmarked."
  - "The reaction line groups reactions by emoji (04-UI-SPEC.md's '{emoji} {reactor name(s)}' example implies per-emoji grouping) — a message with two different emoji reactions renders two reaction lines, not one merged line."

requirements-completed: [SRC-02]

coverage:
  - id: D1
    description: "Every message richness Signal Desktop stores (deleted-for-everyone, edited, attachments, reactions, quoted replies) is parsed into a single messageRecord and correctly degrades the digest's compact tail snippet: an attachment-only tail line shows a placeholder, a deleted tail line is omitted entirely, and a fully-deleted tail yields an empty preview that falls through to StreamRow's existing empty-preview degrade"
    requirement: "SRC-02"
    verification:
      - kind: unit
        ref: "plugins/signal/message_test.go — TestParseMessage_* (plain text, deleted-for-everyone by column, deleted-for-everyone by blob field, edited/latest-text-only, attachment-only, attachment-without-filename, reactions, quote, malformed-blob-yields-record-not-error)"
        status: pass
      - kind: unit
        ref: "plugins/signal/digest_test.go — TestTailSnippet_AttachmentOnlyLastMessageRendersPlaceholder, TestTailSnippet_AttachmentWithNoFilenameUsesFallback, TestTailSnippet_DeletedTailMessageIsOmittedNotTombstoned, TestTailSnippet_AllTailMessagesDeletedYieldsEmptyPreview, TestBuildDigests_MessageCountIncludesDeletedMessages, TestBuildDigests_FullyDeletedTailStillProducesADigestWithEmptyPreview"
        status: pass
      - kind: e2e
        ref: "Live fetch against ~/.config/Signal/sql/db.sqlite (53,346 real messages): sampled the 15 highest-count real digests and confirmed attachment placeholders, grouped reaction lines, quoted excerpts, edited-message suffixes and tombstone bubbles ALL render correctly against real data (not just fixtures) — see 04-03-SUMMARY.md's own session log for the exact grep-based confirmation commands"
        status: pass
    human_judgment: false
  - id: D2
    description: "A Signal digest opens into a readable chat transcript in the existing detail pane: bubbles right/left-aligned by ownership (no accent colour, no per-participant colour), sender name once per run, timestamp once at the run's last bubble, deleted messages as a tombstone keeping their chrome, edited messages with an '(edited)' suffix never a diff, a script/event-handler injection stripped by sanitization before wrapping — zero proto change, zero new Svelte component, zero new DetailPane branch"
    requirement: "SRC-02"
    verification:
      - kind: unit
        ref: "plugins/signal/render_test.go — TestRenderTranscript_* (run grouping same-sender/gap/sender-change, own-message-no-label, deleted-tombstone, edited-suffix-same-line-as-timestamp, attachment-placeholder, reactions-grouped, quote-above-reply, script/event-handler-stripped, sanitize-before-wrap, no-accent-on-bubble/sender/timestamp), TestWrapDocument_CompleteSelfContainedDocument, TestBuildMessageRuns_OrderPreserved"
        status: pass
      - kind: unit
        ref: "plugins/signal/fetch_test.go — TestFetch_FullReturnsWrappedTranscript, TestFetch_PreviewReturnsIdenticalWrappedTranscript, TestFetch_ThumbnailAlwaysUnavailable, TestFetch_UnspecifiedVariantIsInvalidArgument, TestFetch_UnknownSourceIDIsNotFound, TestFetch_MalformedSourceIDIsNotFound, TestFetch_SingleMessageDayRendersExactlyOneBubble"
        status: pass
      - kind: e2e
        ref: "Live GET /api/items/{id} and GET /api/items/{id}/content against a real Signal digest (53,346-message real database): mime_type text/html, size_bytes 2884, document begins with <!doctype html>, exactly one bubble/one sender-name/one timestamp for the fetched single-message day, zero <script> tags"
        status: pass
      - kind: manual_procedural
        ref: "Read web/src/lib/components/DetailPane.svelte's html body-variant branch (line 144-160) and confirmed the returned text/html mime type lands there unchanged; git diff --stat web/ is empty and npm --prefix web run check reports 0 errors"
        status: pass
    human_judgment: false
  - id: D3
    description: "~/.config/Signal/sql/db.sqlite remains byte-identical (SHA-256) across this plan's Match+Fetch cycle, including against real data with Signal Desktop running, and remains free-of-write per the package's own AST read-only scan"
    requirement: "SRC-02"
    verification:
      - kind: unit
        ref: "plugins/signal/byte_identical_test.go — TestDatabaseByteIdenticalAfterMatchAndFetch (fixture, now includes the new message_attachments/reactions tables and messages.id/isErased/json columns)"
        status: pass
      - kind: e2e
        ref: "plugins/signal/byte_identical_test.go — TestLiveDatabaseByteIdentical (WEBSPACES_SIGNAL_LIVE_IT=1, run against the real live database this session, hash unchanged before/after)"
        status: pass
      - kind: e2e
        ref: "scripts/signal-readonly-smoke.sh — run twice this session against the real database, 1467 real digests synced both times, hash unchanged both times"
        status: pass
      - kind: unit
        ref: "plugins/signal/readonly_test.go — TestPluginIssuesNoWriteShapedSQL still passes (unchanged file; the new readAttachments/readReactions/fetchTranscript queries are all SELECT-only)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The 'open in Signal' sgnl:// deep link (both the bare and E.164-contact forms) was validated hands-on against the installed, running Signal Desktop rather than assumed from documentation (04-RESEARCH.md assumption A4)"
    requirement: "SRC-02"
    verification:
      - kind: unit
        ref: "plugins/signal/deeplink_test.go — TestDeepLink_Group, TestDeepLink_PrivateWithE164UsesContactForm, TestDeepLink_PrivateWithoutE164FallsBackToBareForm, TestDeepLink_NeverEmpty, TestDeepLink_UnsafeCharactersAreEscapedNotEmittedRaw, TestEncodePhoneFragment_PlusSignEscaped"
        status: pass
      - kind: manual_procedural
        ref: "gio open invoked against both forms this session (Signal Desktop confirmed as the registered x-scheme-handler/sgnl handler via gio mime / xdg-mime); both returned exit 0, and the contact-form invocation's Signal Desktop single-instance-lock IPC handoff was directly observable in this session's terminal output"
        status: pass
    human_judgment: true
    rationale: "Whether the running Signal Desktop window actually visually raised/focused (as opposed to merely accepting the D-Bus/exec activation without error) cannot be confirmed by this agent session, which has no way to observe the desktop's actual rendered window state. The bare-scheme form's window-raise behavior was already visually confirmed by the developer during 04-01-PLAN.md's own human-verify checkpoint (04-01-SUMMARY.md); a pixel-level re-confirmation of the E.164 contact form specifically is deferred to this phase's end-of-phase human verification pass per workflow.human_verify_mode = 'end-of-phase'."
  - id: D5
    description: "docs/plugin-contract.md documents the path local-path source-config key so a third-party plugin author can build a local-file source without reading plugins/signal's implementation, and no RPC was added (sdk/contract_test.go's allowlist untouched)"
    requirement: "SRC-02"
    verification:
      - kind: unit
        ref: "go test ./sdk/... — unchanged, still passes (no RPC added)"
        status: pass
      - kind: manual_procedural
        ref: "docs/plugin-contract.md's source-config section now documents the path key, its meaning, which shapes are exempt from base_url/token, and the introduction lists plugins/signal as the third reference-plugin shape (local-path, no-network)"
        status: pass
    human_judgment: false

# Metrics
duration: ~2.5h
completed: 2026-08-03
status: complete
---

# Phase 04 Plan 03: Chat-thread renderer, live Fetch, and validated deep link Summary

**A Signal digest opens into a readable, sanitized chat-bubble transcript through the detail pane's existing html iframe (zero proto/frontend change), with attachments/reactions read from Signal Desktop's real dedicated SQL tables rather than the message blob 04-RESEARCH.md assumed, and the sgnl:// "open in Signal" link validated hands-on against the installed application**

## Performance

- **Duration:** ~2.5h (includes hands-on schema introspection of the real, live 53,346-message database before writing any implementation)
- **Tasks:** 3 (all `type="auto"`, Tasks 1-2 `tdd="true"`)
- **Files modified:** 13 (7 created, 6 modified)

## Accomplishments
- Parsed every message richness Signal Desktop stores — deleted-for-everyone, edited, attachments, reactions, quoted replies — into a single `messageRecord` (`message.go`) that both the compact digest snippet and the full transcript read, never re-parsing a message row twice.
- Discovered and corrected a real gap between 04-RESEARCH.md's illustrative assumption and this machine's actual live schema: attachments and reactions live in Signal Desktop's own dedicated `message_attachments`/`reactions` SQL tables, never in the message row's own `json` blob — confirmed by direct introspection of the real database (53,346 messages, 17,182 attachment rows, 7,412 reaction rows) before writing a line of parsing code.
- Built the chat-transcript HTML renderer (`render.go`) — bubbles grouped into sender runs (5-minute gap / sender-change boundary), own messages unlabelled and right-aligned, deleted messages as a tombstone keeping their chrome, edited messages with an "(edited)" suffix, attachments as placeholder chips, reactions grouped by emoji, quoted replies above their reply — sanitized per-field via a dedicated `bluemonday.UGCPolicy()`-derived policy before assembly, wrapped in a fixed, self-contained stylesheet that never uses the accent colour for bubbles/sender-names/timestamps.
- Wired `Fetch`'s FULL/PREVIEW variants to actually re-open the database, re-read the digest's day, render, and return `text/html` — confirmed landing directly in `DetailPane.svelte`'s existing `html` body-variant iframe branch with zero frontend file touched.
- Validated the `sgnl://` deep link hands-on against the installed, running Signal Desktop (both the bare and E.164-contact forms) via `gio open`, closing 04-RESEARCH.md assumption A4.
- Published the `path` local-path source-config shape in `docs/plugin-contract.md` and wrote `plugins/signal/README.md` documenting the cgo/sqlcipher build prerequisite.

## Task Commits

Each task was committed atomically:

1. **Task 1: The rich message record, and what it does to the digest snippet** - `646a072` (feat)
2. **Task 2: The chat-thread transcript — the renderer Phase 5 reuses** - `8efec71` (feat)
3. **Task 3: Open in Signal, validated against the real desktop, and the published contract** - `f21d554` (docs)

**Plan metadata:** (this commit)

_Note: Tasks 1 and 2 are `tdd="true"`; each task's `<behavior>` block described a cohesive, non-separable unit (a full parsing type or a full renderer), so tests and implementation landed together in one commit per task, consistent with 04-01/04-02-SUMMARY.md's own established precedent for this plugin — no separate RED/GREEN commit split was warranted._

## Files Created/Modified

- `plugins/signal/message.go` - `attachment`, `reaction`, `messageRecord` types; `parseMessage` (blob-plus-supplied-richness parser); `senderDisplayName`; `attachmentPlaceholder`/`reactionLines` (shared by digest.go and render.go)
- `plugins/signal/message_test.go` - Table-driven tests for every richness case, including the malformed-blob-yields-record-not-error case
- `plugins/signal/digest.go` - `tailSnippet` made richness-aware (attachment placeholder, deleted-tail omission, empty-preview degrade); `message` struct removed (superseded by `messageRecord`)
- `plugins/signal/digest_test.go` - Rewritten against `messageRecord` fixtures; new attachment/deleted-tail/all-deleted/count-includes-deleted cases
- `plugins/signal/plugin.go` - `readAttachments`/`readReactions` (new dedicated-table queries); `readMessages` rewritten to build `[]messageRecord` via `parseMessage`; `Fetch`'s FULL/PREVIEW now dispatch to `fetchTranscript`, which re-opens the DB, re-reads the digest's day, renders and wraps; `transcriptMimeType` const; `ownSenderLabel` constant reused from render.go in place of the two prior `"You"` literals
- `plugins/signal/byte_identical_test.go` - Fixture schema extended with `messages.id`/`isErased`/`json` columns and the new `message_attachments`/`reactions` tables (schema-only, unpopulated) so the fixture keeps compiling/passing against the new row-reading layer
- `plugins/signal/render.go` - `signalTranscriptSanitizePolicy`, `sanitizeText`, `messageRun`/`buildMessageRuns`, `renderTranscript`/`renderBubble`, `signalThemeStyle`, `WrapDocument`, `formatTimestamp`
- `plugins/signal/render_test.go` - Run-grouping, own-message, deleted/edited/attachment/reaction/quote, sanitization, wrap-completeness and accent-budget tests
- `plugins/signal/fetch_test.go` - FULL/PREVIEW/THUMBNAIL/unspecified-variant/unknown-source-id/single-bubble-day cases, using a shared fixture-plugin helper
- `plugins/signal/deeplink.go` - Doc comment updated recording the 2026-08-03 hands-on `sgnl://` validation
- `plugins/signal/deeplink_test.go` - Group/1:1-with-E.164/1:1-without-E.164/never-empty/escaping table tests
- `plugins/signal/README.md` - Build prerequisite (sqlcipher package + Arch/Debian commands), SQLite 3.51.3 floor and why, `path`-only config shape, read-only guarantee, opt-in live test instructions
- `docs/plugin-contract.md` - `path` source-config key documented in the source-config section; `plugins/signal` added to the introduction's reference-plugin list as the local-path, no-network shape

## Decisions Made

See `key-decisions` in frontmatter for the five load-bearing ones (attachments/reactions live in dedicated tables not the blob; attachmentType filtering; FULL/PREVIEW share one path; edited-suffix/timestamp co-location rule; reaction grouping by emoji). All five are documented inline in the corresponding source file's own comments, not just here.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Attachments and reactions read from dedicated SQL tables, not the message row's json blob**
- **Found during:** Task 1, mandated schema-introspection step (this task's own hands-on `PRAGMA table_info`/`SELECT json FROM messages` probes against the real, live database, before writing `message.go`)
- **Issue:** 04-RESEARCH.md's own framing ("everything richer than plain body text — attachments, reactions, quotes, edit history, the deleted-for-everyone marker — lives only in that blob") is wrong for attachments and reactions specifically on this real schema: no sampled message row with `hasAttachments=1` carried an `attachments` key in its own `json` blob at all. `sqlite_master` introspection found dedicated `message_attachments` (17,182 rows, joined by `messageId = messages.id`, filtered `editHistoryIndex = -1` for the current revision) and `reactions` (7,412 rows) tables instead.
- **Fix:** `message.go`'s `parseMessage` accepts already-resolved `[]attachment`/`[]reaction` slices supplied by the caller rather than parsing them from the blob; `plugin.go`'s new `readAttachments`/`readReactions` query the dedicated tables directly, joined by message id, with `attachmentType` filtered to `{attachment, sticker, contact}` (excluding `preview`/`quote`/`long-message`, which describe something other than a file the sender attached).
- **Files modified:** `plugins/signal/message.go` (documented inline in `messageBlobFields`'s doc comment), `plugins/signal/plugin.go` (`readAttachments`/`readReactions`)
- **Verification:** `message_test.go`'s attachment/reaction cases pass against the supplied-parameter design; a live fetch against the real database (53,346 messages) confirmed attachment placeholders and grouped reaction lines render correctly in the actual served transcript
- **Committed in:** `646a072` (Task 1 commit)

**2. [Rule 3 - Blocking] byte_identical_test.go's fixture schema extended for the new row-reading layer**
- **Found during:** Task 1, first `make test-signal` run after `readMessages`'s signature change
- **Issue:** The existing fixture-database builder (`byte_identical_test.go`, from 04-02) had no `messages.id`/`isErased`/`json` columns and no `message_attachments`/`reactions` tables — the new `readMessages`/`readAttachments`/`readReactions` queries failed against it with "no such table"/"no such column".
- **Fix:** Extended `buildFixtureDatabase`'s schema with the three new columns and two new (schema-only, unpopulated) tables, and gave each fixture message a stable `id`.
- **Files modified:** `plugins/signal/byte_identical_test.go`
- **Verification:** `TestDatabaseByteIdenticalAfterMatchAndFetch` and `TestLiveDatabaseByteIdentical` (against the real database) both pass
- **Committed in:** `646a072` (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (1 Rule 1 ground-truth correction discovered by the plan's own mandated schema-introspection step, 1 Rule 3 blocking knock-on fix to keep the existing fixture test compiling)
**Impact on plan:** Both were necessary corrections against the real, live database this plan's own Task 1 instructed introspecting before writing any parsing code ("use the exact JSON field names recorded... use those, not any name assumed here"). No scope creep — every fix stayed inside `plugins/signal/`'s own files.

## Issues Encountered

None beyond the deviations above. `make test`, `make test-signal`, `go test ./sdk/...`, `CGO_ENABLED=0 go build ./...`, `npm --prefix web run check` and `./scripts/signal-readonly-smoke.sh` all passed repeatedly this session, including twice against the real, live 53,346-message database with Signal Desktop running throughout, with an unchanged SHA-256 hash both times.

## User Setup Required

None — this plan added no new external dependency, config key, or environment variable. `docs/plugin-contract.md`'s `path` key documentation and `plugins/signal/README.md` are read-only reference material; the `[sources.signal]` block itself was already added to the developer's real config in 04-01.

## Next Phase Readiness

- ROADMAP.md's phase-4 success criteria are fully met: the detail pane shows the surrounding conversation thread with an "open in Signal" affordance declared conversation-only, and the transcript renderer is a genuinely source-agnostic HTML-document-per-Fetch pattern (never anything Signal-specific leaking into `DetailPane.svelte`) — Phase 5's WhatsApp plugin can reuse `render.go`'s shape (build its own `messageRecord`-equivalent, its own sanitize policy, and its own `WrapDocument`-style wrap) without touching the frontend at all.
- The one item explicitly deferred to end-of-phase human verification (per `workflow.human_verify_mode = "end-of-phase"`): a pixel-level visual confirmation that the `sgnl://signal.me/#p/<e164>` contact-form deep link actually raises/focuses Signal Desktop on the correct contact, as distinct from the automated confirmation already obtained this session (exit 0, observable IPC handoff). The bare-scheme form's visual behavior was already confirmed during 04-01's own checkpoint.
- No blockers identified.

---
*Phase: 04-signal-conversations*
*Completed: 2026-08-03*

## Self-Check: PASSED

All 13 files listed under `key-files` were verified present on disk, and all three task commits (`646a072`, `8efec71`, `f21d554`) were verified present in `git log --oneline --all`. No missing items.
