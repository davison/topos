---
phase: 08-whatsapp-conversations-managed-risk
plan: 01
subsystem: source-plugin
tags: [whatsmeow, whatsapp, sqlite, go-plugin, chat-transcript, linked-device]

requires:
  - phase: 04-signal-conversations
    provides: "The chat-transcript rendering contract (CONTENT_SHAPE_CHAT_TRANSCRIPT policy) and the digest/render/deeplink pattern this plugin ports near-verbatim"
  - phase: 05-source-instances-and-per-type-matching
    provides: "The per-instance match_fields contract (topos.v2) this plugin's Match implements"
provides:
  - "plugins/whatsapp: a working, pure-Go, cgo-free WhatsApp source plugin skeleton — link, capture, store, match (groups only), digest, transcript render, deep link — end-to-end code complete and unit-tested, NOT yet verified against a real linked device"
affects: [08-02-whatsapp-1-1-and-health-taxonomy, 08-03-whatsapp-in-app-pairing]

actuals:
  tokens: 20000
  tasks: 2
  commits: 1

tech-stack:
  added: ["go.mau.fi/whatsmeow@v0.0.0-20260806224404-e277b766ab33", "modernc.org/sqlite v1.54.0 (plugin-local)", "github.com/mdp/qrterminal/v3 v3.2.1"]
  patterns:
    - "Persistent-connection plugin: unlike every other plugin in this repo (open-and-close per RPC), this plugin holds a whatsmeow Client.Connect() and an always-open *sql.DB for its entire process lifetime; Match/Fetch read the local store fresh every call, never a live whatsmeow call"
    - "chat_jid-first identity (assumption-delta promote decision): message store primary key, digest grouping key, and source_id encoding are all keyed on chat_jid from the outset, with is_group as one variant field, not a parallel identity path"
    - "Store-lock mutual exclusion: storelock.go's exclusive advisory LOCK_EX|LOCK_NB lock enforces link-mode / serve-mode never holding whatsmeow's sqlstore open concurrently"

key-files:
  created:
    - plugins/whatsapp/main.go
    - plugins/whatsapp/link.go
    - plugins/whatsapp/storelock.go
    - plugins/whatsapp/connect.go
    - plugins/whatsapp/eventhandler.go
    - plugins/whatsapp/messagestore.go
    - plugins/whatsapp/plugin.go
    - plugins/whatsapp/digest.go
    - plugins/whatsapp/match.go
    - plugins/whatsapp/render.go
    - plugins/whatsapp/deeplink.go
    - plugins/whatsapp/go.mod
  modified:
    - go.work
    - Makefile
    - config.example.toml
    - .gitignore

key-decisions:
  - "Task 1 checkpoint APPROVED (user response, this session): pin go.mau.fi/whatsmeow at v0.0.0-20260806224404-e277b766ab33 — verified live on 2026-08-10 via `go list -m -json go.mau.fi/whatsmeow@latest` (identical to 08-RESEARCH.md's own snapshot), pinned commit's go.mod dependency tree confirmed 100% cgo-free"
  - "buildMessageRuns determines run ownership from messageRecord.IsFromMe directly (WhatsApp's own store gives this natively) rather than Signal's derived-from-display-name-string convention — a minor correctness improvement over the ported pattern, not a Signal-parity break"
  - "Deletion/edit handling implemented via whatsmeow ProtocolMessage REVOKE/MESSAGE_EDIT events (messagestore.MarkDeleted/MarkEdited) — not explicitly named in 08-01-PLAN.md's action text but required for the schema's own is_deleted/is_edited columns to ever be set by anything"
  - "Added .gitignore entries for plugins/whatsapp/whatsapp and plugins/mockstrict/mockstrict (the latter a pre-existing gap, not introduced by this plan) — both are stray `go build ./...`-without--o binaries the existing stray-binary block already covers for every other plugin"

requirements-completed: []

coverage:
  - id: D1
    description: "plugins/whatsapp module builds pure-Go (CGO_ENABLED=0), joins go.work, and its own test suite (storelock, messagestore, digest/render edge cases) passes"
    verification:
      - kind: unit
        ref: "plugins/whatsapp: go test ./... -run 'TestStoreLock|TestDigest|TestMessageStore' -v"
        status: pass
      - kind: integration
        ref: "make test-portable (full workspace, whatsapp module included)"
        status: pass
    human_judgment: false
  - id: D2
    description: "A real WhatsApp account links via the terminal QR flow, the linked session survives a kernel restart, a matching group's digest appears in a real webspace stream, and opening it renders the Phase 4 chat transcript"
    verification: []
    human_judgment: true
    rationale: "Requires a real WhatsApp account with an active phone to scan the pairing QR code (this task's own stated precondition) — no such device is available to this automated execution environment. Code is written and unit-tested but genuinely unverified against a live WhatsApp session."

duration: ~55min
completed: 2026-08-10
status: halted
---

# Phase 8 Plan 1: WhatsApp Conversations (Managed Risk) — Tracer Summary

**Pure-Go plugins/whatsapp source plugin (whatsmeow-backed): terminal QR linking, persistent-connection message capture into its own SQLite store, group-name matching, chat-day digests, and Phase-4-reused chat-transcript rendering — code-complete and unit-tested, but NOT YET verified against a real linked WhatsApp account.**

## Performance

- **Duration:** ~55 min (this continuation session; Task 1's original checkpoint pause is not counted)
- **Started:** 2026-08-10T12:00Z (approx, continuation agent resume)
- **Completed:** 2026-08-10T12:49Z
- **Tasks:** 1 of 3 fully complete (Task 1 approved by user), Task 2 code-complete/automated-verify passed but its human-check step not run, Task 3 not started
- **Files modified:** 21 (17 new plugins/whatsapp files, go.work, Makefile, config.example.toml, .gitignore)

## Accomplishments

- Task 1's blocking-human package-legitimacy checkpoint resolved: user approved pinning `go.mau.fi/whatsmeow@v0.0.0-20260806224404-e277b766ab33`
- Full `plugins/whatsapp` module written end-to-end: CLI link mode (ASCII QR via qrterminal), whatsmeow session-store connection with a persistent background client, this plugin's own separate `messages.db` (WAL, busy-timeout), an exclusive store-lock enforcing link-mode/serve-mode mutual exclusion, group-name matching, chat-day digest assembly, and chat-transcript rendering reused near-verbatim from `plugins/signal/render.go`
- `sourceIDForDigest`/`decodeSourceID` proven to round-trip identically for a group JID and a 1:1 JID (the assumption-delta's own advisory contract test), even though 1:1 matching itself is out of this plan's scope
- All automated verification from the plan's own `<verify>` block passes: `make test-portable` (whatsapp module included), `CGO_ENABLED=0 go build ./plugins/whatsapp`, and `go test ./... -run 'TestStoreLock|TestDigest|TestMessageStore'`

## Task Commits

1. **Task 1: Package legitimacy checkpoint** — resolved by user approval (no code commit; approval recorded above)
2. **Task 2: End-to-end tracer (link → connect → capture → match → digest → transcript)** — `f659c20` (feat) — automated verify passed; human-check verify (real device linking) and the tracer feedback gate are BLOCKED pending human action (see Checkpoint below)

**Plan metadata:** this commit (docs: SUMMARY + checkpoint)

_Note: this is a `tdd="true"` tracer task — RED/GREEN discipline was followed for the pure-function/store layer (storelock, messagestore, digest/render edge cases) inside the single Task 2 commit, since the task's own action text describes writing tests first for pure helpers with no network dependency; the whatsmeow-dependent connection/event-handling code (connect.go, eventhandler.go, link.go, plugin.go's Match/Fetch/Health) has no unit tests because it cannot be exercised without a live WhatsApp session — this is inherent to the task, not a shortcut._

## Files Created/Modified

- `plugins/whatsapp/main.go` — WEBSPACES_SOURCE_CONFIG bootstrap + `-link`/`-path` flag branch
- `plugins/whatsapp/link.go` — one-shot terminal QR link flow (`runLinkCLI`)
- `plugins/whatsapp/storelock.go` — exclusive advisory lock (`acquireStoreLock`/`ErrStoreInUse`)
- `plugins/whatsapp/connect.go` — whatsmeow sqlstore open, device lookup, persistent `Client.Connect()`, `pluginLogger` (stderr-only, WARN floor)
- `plugins/whatsapp/eventhandler.go` — background `handleEvent`: message capture, history-sync replay, group-name refresh, REVOKE/MESSAGE_EDIT handling
- `plugins/whatsapp/messagestore.go` — this plugin's own SQLite store (`chats`/`messages` tables), WAL + busy-timeout, idempotent `Append`
- `plugins/whatsapp/plugin.go` — `SourcePlugin` (Describe/Match/Fetch/Health), health-state flag, empty-success-vs-error discipline
- `plugins/whatsapp/digest.go` — chat-day digest assembly, `sourceIDForDigest`/`decodeSourceID`, `localDay`/`localDayKey`, `tailSnippet`/`Snippet`
- `plugins/whatsapp/match.go` — `matchesAnyKeyword`, `candidateNames` (group-subject-only), `eligibleChats`
- `plugins/whatsapp/render.go` — `buildMessageRuns`, `renderTranscript`/`renderBubble`, `escapeText`
- `plugins/whatsapp/deeplink.go` — bare `whatsapp://` scheme (NOT yet hands-on verified — Task 3's job)
- `plugins/whatsapp/go.mod` — whatsmeow pin + dated audit comment
- `go.work` — added `./plugins/whatsapp` member
- `Makefile` — `plugins:`/`test-portable:` targets extended
- `config.example.toml` — new `[sources.whatsapp]` block
- `.gitignore` — stray-binary entries for `plugins/whatsapp/whatsapp` and `plugins/mockstrict/mockstrict`

## Decisions Made

See `key-decisions` in frontmatter above. Summary: Task 1's pin approved as-is; `IsFromMe`-based run ownership (a robustness improvement over the ported Signal pattern); delete/edit handling added via whatsmeow's own ProtocolMessage taxonomy (Rule 2 — the schema's own columns would otherwise never be set by anything); two `.gitignore` entries added for stray binaries.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Implemented message delete/edit handling via whatsmeow ProtocolMessage events**
- **Found during:** Task 2 (eventhandler.go)
- **Issue:** The plan's action text describes the `messages` table's `is_deleted`/`is_edited` columns but does not explicitly spell out the whatsmeow event-handling logic that sets them (a REVOKE protocol message marks a target message deleted; a MESSAGE_EDIT protocol message updates a target message's body and marks it edited)
- **Fix:** Added `eventhandler.go` handling for `waE2E.ProtocolMessage_REVOKE` and `waE2E.ProtocolMessage_MESSAGE_EDIT`, calling `messageStore.MarkDeleted`/`MarkEdited` by the target message's own ID
- **Files modified:** plugins/whatsapp/eventhandler.go, plugins/whatsapp/messagestore.go
- **Verification:** Compiles and passes `go vet`; not exercised by a live event (no automated test — requires a real WhatsApp session to trigger a real revoke/edit)
- **Committed in:** `f659c20`

**2. [Rule 3 - Blocking] Added missing .gitignore entries for stray plugin binaries**
- **Found during:** Task 2, post-build cleanup
- **Issue:** `go build ./plugins/whatsapp/...` without `-o` (and the pre-existing `make test-portable` mockstrict target) leaves a binary named after the package in the repo root/module directory; the existing `.gitignore` stray-binary block covered every other plugin but not `plugins/whatsapp` (new) or `plugins/mockstrict` (a pre-existing gap, unrelated to this plan, discovered incidentally)
- **Fix:** Added both paths to the existing stray-binary `.gitignore` block
- **Files modified:** .gitignore
- **Verification:** `git status --short` clean after `rm -f whatsapp plugins/mockstrict/mockstrict`
- **Committed in:** `f659c20`

---

**Total deviations:** 2 auto-fixed (1 missing critical, 1 blocking)
**Impact on plan:** Both necessary for correctness/hygiene. No scope creep.

## Issues Encountered

- **whatsmeow API surface required live research, not training-data recall:** the exact shape of `sqlstore.New`'s dialect string (confirmed `"sqlite"` — modernc.org/sqlite's own registered driver name — works via `go.mau.fi/util/dbutil.ParseDialect`'s `strings.HasPrefix(engine, "sqlite")` prefix match), `ParseWebMessage`'s signature, `ProtocolMessage_MESSAGE_EDIT`'s numeric value, and `ownSenderLabel`-equivalent fields (`types.MessageSource.IsFromMe`/`IsGroup`) were all confirmed by reading the actual vendored source in the Go module cache before writing any code, per the analysis-paralysis guard's action-over-reading discipline — resolved without a checkpoint.
- **No real WhatsApp account is available to this automated execution environment.** Task 2's own `<precondition>` names this a hard prerequisite with no fallback; its `<verify>` block's `<human-check>` half (real terminal linking, real group digest appearing in a real stream) could not run. This is the sole reason this plan halts here rather than completing.

## User Setup Required

None yet in the code sense (no external service credentials) — but Task 2's human-check verify and all of Task 3 require the user to physically scan a QR code with a real, active WhatsApp phone. See Checkpoint below for the exact commands.

## Next Phase Readiness — CHECKPOINT REACHED

**Type:** human-verify (Task 2's own `<human-check>` step) leading directly into **human-action** (Task 3's mandatory hands-on spike)
**Gate:** blocking
**Plan:** 08-01
**Progress:** Task 1 approved; Task 2 code-complete + automated-verify passed; Task 2 human-check and Task 3 NOT run (no real WhatsApp account available to this agent)

### Completed Tasks

| Task | Name | Commit | Files |
| ---- | ---- | ------ | ----- |
| 1 | Package legitimacy checkpoint | (user approval, no commit) | plugins/whatsapp/go.mod (pin + audit comment) |
| 2 | End-to-end tracer — code + automated verify | `f659c20` | plugins/whatsapp/*, go.work, Makefile, config.example.toml, .gitignore |

### Current Task

**Task 2 (human-check half) / Task 3 (mandatory spike):** blocked
**Status:** awaiting human action — a real WhatsApp account and phone are required
**Blocked by:** Precondition not met — "A real WhatsApp account with an active phone is available to scan the pairing QR code" (Task 2's own stated precondition; no fallback exists per 08-RESEARCH.md Environment Availability)

### Checkpoint Details

Run these commands yourself (no CLI step here can be automated further):

```bash
mkdir -p ~/.local/share/topos/whatsapp
CGO_ENABLED=0 go build -o bin/plugins/topos-plugin-whatsapp ./plugins/whatsapp
bin/plugins/topos-plugin-whatsapp -link -path ~/.local/share/topos/whatsapp
```

1. Scan the rendered ASCII QR code with your phone (WhatsApp > Linked devices > Link a device). Confirm the process exits 0 with "Linked successfully."
2. Add a `[sources.whatsapp]` block (already present in `config.example.toml` — copy into your real `config.toml`) plus a webspace `match` block naming a real group you're in.
3. Run `make dev`. Confirm the WhatsApp source chip appears healthy, a digest row for that group appears in the stream within a sync cycle, and opening it renders the Phase 4 chat transcript.
4. Restart the kernel (`make dev` down/up) and confirm it reconnects with no second QR scan (Task 2's must_haves criterion 1).
5. Then work through Task 3's four-part spike (backfill volume, deep-link scheme — `gio open 'whatsapp://'` — link stability across a restart and airplane-mode, and the de-link event taxonomy) exactly as `08-01-PLAN.md` Task 3 describes, and record all four answers.

### Awaiting

The user's real-device verification of Task 2's human-check step, followed by Task 3's four spike answers (backfill row count + date range + DB size; deep-link scheme result; restart/airplane-mode observations; de-link event name + whether captured rows survived) plus an "approved" or corrective response. `08-01-PLAN.md`'s `<output>` block requires both to be recorded verbatim in this SUMMARY before the plan can be marked complete — Plans 08-02 and 08-03 consume them directly (Task 3's deep-link answer, in particular, may require correcting `deeplink.go`).

---
*Phase: 08-whatsapp-conversations-managed-risk*
*Completed: 2026-08-10 (halted — see Checkpoint above)*
