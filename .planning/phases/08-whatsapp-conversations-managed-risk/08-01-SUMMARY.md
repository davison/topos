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
  tokens: 24350
  tasks: 2
  commits: 6

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
    - plugins/whatsapp/connect_test.go
    - plugins/whatsapp/pairwait.go
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
  - "whatsmeow's own sqlstore requires PRAGMA foreign_keys on at open time — modernc.org/sqlite's DSN pragma syntax (`_pragma=foreign_keys(1)`) differs from the mattn/go-sqlite3-style `_foreign_keys=on` shorthand whatsmeow's own doc comment illustrates; whatsmeowSessionDSN() is the one shared helper both link-mode and serve-mode call"
  - "PairSuccess (and the QR channel's own 'success' event) fires BEFORE the post-pair login handshake completes, per whatsmeow's own doc comment — pairLoginWaiter (pairwait.go) makes -link wait for a genuine *events.Connected before disconnecting, in both the fresh-pairing path and the already-linked/reconnect path (whatsmeow persists Store.ID before PairSuccess dispatches, so a saved device row alone never proves a session actually completed)"

requirements-completed: []

coverage:
  - id: D1
    description: "plugins/whatsapp module builds pure-Go (CGO_ENABLED=0), joins go.work, and its own test suite (storelock, messagestore, digest/render edge cases, DSN/foreign_keys, pairLoginWaiter) passes"
    verification:
      - kind: unit
        ref: "plugins/whatsapp: go test ./... -run 'TestStoreLock|TestDigest|TestMessageStore|TestWhatsmeowSessionDSN|TestPairLoginWaiter' -v (18 tests)"
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

**3. [Rule 1 - Bug] Fixed whatsmeow session-store DSN to actually enable foreign_keys**
- **Found during:** Checkpoint feedback — the user ran `bin/plugins/topos-plugin-whatsapp -link -path ~/.local/share/topos/whatsapp` on their real machine and it failed before showing a QR: `failed to upgrade database: foreign keys are not enabled`
- **Issue:** `connect.go`/`link.go` opened whatsmeow's own sqlstore with `file:<path>?_foreign_keys=on` — the DSN shorthand illustrated in whatsmeow's own doc comment, but that's mattn/go-sqlite3's query-param convention. This plugin uses modernc.org/sqlite (the pure-Go driver, per this project's stack constraints), which silently ignores `_foreign_keys=on` as an unrecognised query param — foreign keys stayed off, and `sqlstore.Container.Upgrade` refuses to run its migrations without them, failing before the QR flow ever starts
- **Fix:** Added `whatsmeowSessionDSN(dbPath)` in `connect.go`, using modernc.org/sqlite's actual DSN pragma syntax: `?_pragma=foreign_keys(1)&_pragma=busy_timeout(10000)` (one `_pragma=<body>` query param per pragma, applied as `PRAGMA <body>` on every new pooled connection). Both `link.go` and `connect.go` now call this one shared helper, so link-mode and serve-mode open whatsmeow's sqlstore identically
- **Files modified:** plugins/whatsapp/connect.go, plugins/whatsapp/link.go, plugins/whatsapp/connect_test.go (new)
- **Verification:** New `TestWhatsmeowSessionDSN_MigrationsRunAgainstRealSQLStore` opens `sqlstore.New` against a real temp-file DB via `whatsmeowSessionDSN` and calls `GetFirstDevice` — reverting the fix locally reproduced the user's exact live error message, confirming the test is a real regression guard, not a vacuous pass. `CGO_ENABLED=0 go build ./plugins/whatsapp` and the full `plugins/whatsapp` test suite (13 tests) pass with the fix in place; `make test-portable` passes for the whole workspace
- **Committed in:** `397e94c`

**4. [Rule 1 - Bug] Fixed premature disconnect in `-link` mode stranding the phone mid post-pair login**
- **Found during:** Second round of checkpoint feedback — the user's QR scan succeeded ("Linked successfully" printed, immediately followed by `Error sending close to websocket: failed to close WebSocket: failed to read frame header: EOF`), but the phone stayed on "Logging in…" with the camera still active
- **Issue:** `runLinkCLI`'s QR-loop `case "success"` returned immediately (firing the deferred `client.Disconnect()`). But `"success"` fires on whatsmeow's `*events.PairSuccess`, whose own doc comment states: "this is generally followed by a websocket reconnection, so you should wait for [`*events.`]Connected before trying to send anything." Disconnecting at `PairSuccess` drops the socket before the phone-side login handshake (app-state/key exchange over a NEW authenticated connection) completes, stranding the phone
- **Fix:** Added `pairwait.go`'s `pairLoginWaiter` — registered as an additional `Client.AddEventHandler` callback *before* `Connect()`, alongside the QR channel's own internal handler, signalling on a genuine `*events.Connected` (success) or `LoggedOut`/`StreamReplaced`/`ConnectFailure` (definitive failure). `link.go`'s `"success"` branch now sets a flag and prints progress instead of returning; after the QR loop exits, it calls `loginWaiter.wait(60s timeout)`, then holds the connection open for a 5s grace window before disconnecting and reporting success
- **Files modified:** plugins/whatsapp/link.go, plugins/whatsapp/pairwait.go (new), plugins/whatsapp/pairwait_test.go (new)
- **Verification:** `pairwait_test.go` unit-tests `pairLoginWaiter` directly with fake whatsmeow events (Connected → success; LoggedOut/StreamReplaced → named failure; no qualifying event → named timeout; a second event after the first is signalled does not block/panic) — no live server needed, since the seam is the event-handler callback itself. Confirmed `connect.go`'s serve-mode path carries no equivalent risk: it never calls `Disconnect()` at all (the connection is held for the plugin subprocess's entire lifetime; the only `Close()` call is `store.Close()`, and only on `startBackgroundClient`'s own construction-failure path, before any connection exists). `CGO_ENABLED=0 go build ./plugins/whatsapp`, the full `plugins/whatsapp` test suite (18 tests), and `make test-portable` all pass
- **Committed in:** `314d5de`

**5. [Rule 1 - Bug] `-link`'s "already linked" branch now reconnects and confirms instead of trusting a saved device row**
- **Found during:** Same investigation as #4 above — checking whatsmeow's own source (`pair.go`'s `handlePair`) showed `cli.Store.Save(ctx)` persists `Store.ID` (making `device.ID` non-nil) *synchronously inside the pairing handshake, before `*events.PairSuccess` is even dispatched* — i.e. strictly before the point #4's bug disconnected. This means the user's interrupted first attempt almost certainly left a fully-saved device row despite the phone never finishing "Logging in…"
- **Issue:** `runLinkCLI`'s original `if device.ID != nil { print "Already linked"; return nil }` branch never attempted to connect at all — a subsequent `-link` run (the exact next step the docs tell a user to take) would have printed a falsely reassuring "Already linked" message and exited, leaving the phone stuck exactly as before with no path to complete the login short of running serve mode separately
- **Fix:** The `device.ID != nil` branch now calls `client.Connect()` and waits on the same `pairLoginWaiter` (60s timeout, 5s grace window) before reporting anything — safe and correct in both cases: an already-fully-linked device reconnects normally (identical to what `connect.go`'s serve-mode path does on every kernel restart), and a half-finished device completes its login handshake here instead of needing a wipe and re-scan
- **Files modified:** plugins/whatsapp/link.go
- **Verification:** `CGO_ENABLED=0 go build ./plugins/whatsapp`, the full `plugins/whatsapp` test suite (18 tests), and `make test-portable` all pass. Not exercised live (requires a real half-linked device to reproduce the exact race) — the fix's correctness rests on `pair.go`'s own source (cited above) plus `connect.go`'s already-live-proven reconnect path being the identical code shape
- **Committed in:** `ae6cf53`

---

**Total deviations:** 5 auto-fixed (1 missing critical, 1 blocking, 3 bugs found via two rounds of checkpoint feedback)
**Impact on plan:** All five necessary for correctness/hygiene. No scope creep.

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

**Two real-device rounds of feedback so far, both now fixed:**
1. `foreign keys are not enabled` at store-open, before any QR ever rendered — fixed by Deviation #3 (`397e94c`).
2. QR scanned successfully, but the plugin disconnected the instant pairing was accepted (before the phone's own post-pair login handshake completed) — phone stuck on "Logging in…", plugin printed a misleading `EOF` warning right after "Linked successfully." Fixed by Deviations #4 and #5 (`314d5de`, `ae6cf53`): `-link` now waits for a genuine `*events.Connected` before disconnecting, in BOTH the fresh-pairing path and the "already linked" path.

**Guidance for the current half-linked state (round 2's interrupted attempt):**

- **What to check on the phone first:** open WhatsApp → Settings → Linked Devices. whatsmeow persists the device's identity (`Store.ID`) to `whatsmeow.db` *synchronously, before the post-pair login handshake even starts* (confirmed by reading `pair.go`'s `handlePair`: `cli.Store.Save(ctx)` runs before `dispatchEvent(&events.PairSuccess{...})`, which is what our old code's premature disconnect fired right after). So the device may or may not already be visible in this list — either is consistent with what happened; if it's NOT listed, WhatsApp's servers likely expired the incomplete session and a fresh QR scan is needed anyway (the rebuilt binary's normal fresh-link path handles this the same as any first link).
- **Do NOT wipe `~/.local/share/topos/whatsapp/` (whatsmeow.db, its `-wal`/`-shm` files, or messages.db) as a first step.** The locally-saved device row from the interrupted attempt is cryptographically complete (the pairing handshake itself, including key exchange, finishes before `Store.Save` — what never finished was the SEPARATE post-pair reconnect-and-login-sync this plugin's fix now waits for). Deviation #5 specifically makes a plain re-run of `-link` reconnect using that saved identity and wait for login to actually complete, rather than short-circuiting with "Already linked" — this is safe whether the device shows up in the phone's Linked Devices list or not.
- **Recommended next step:** rebuild and simply re-run `-link` exactly as before — no extra flags, no manual file cleanup:
  ```bash
  CGO_ENABLED=0 go build -o bin/plugins/topos-plugin-whatsapp ./plugins/whatsapp
  bin/plugins/topos-plugin-whatsapp -link -path ~/.local/share/topos/whatsapp
  ```
  Two possible outcomes, both handled correctly now: (a) if the device is still known to WhatsApp's servers, you'll see "Already linked as `<jid>` — reconnecting to confirm the session is fully established…" followed by "Session confirmed." with NO new QR needed; (b) if the phone-side session expired, WhatsApp will reject the stale reconnect and whatsmeow will need a fresh pairing — if step (a) instead prints a reconnect-failure error, delete `~/.local/share/topos/whatsapp/whatsmeow.db*` (all three files: `whatsmeow.db`, `whatsmeow.db-wal`, `whatsmeow.db-shm` — leave `messages.db` alone, it holds no session state) and re-run `-link` to get a fresh QR. Only do this file cleanup if the reconnect attempt itself reports a failure, not preemptively.
- After a successful link (fresh or reconnect-confirmed), proceed with the original checkpoint steps below.

```bash
mkdir -p ~/.local/share/topos/whatsapp   # already exists if you're resuming; harmless if so
CGO_ENABLED=0 go build -o bin/plugins/topos-plugin-whatsapp ./plugins/whatsapp
bin/plugins/topos-plugin-whatsapp -link -path ~/.local/share/topos/whatsapp
```

1. Confirm the process exits 0 with either "Linked successfully." (fresh pair) or "Session confirmed." (reconnect path) — and, most importantly, confirm the phone's own WhatsApp UI leaves the "Logging in…" screen and shows the device as connected under Linked Devices.
2. Add a `[sources.whatsapp]` block (already present in `config.example.toml` — copy into your real `config.toml`) plus a webspace `match` block naming a real group you're in.
3. Run `make dev`. Confirm the WhatsApp source chip appears healthy, a digest row for that group appears in the stream within a sync cycle, and opening it renders the Phase 4 chat transcript.
4. Restart the kernel (`make dev` down/up) and confirm it reconnects with no second QR scan (Task 2's must_haves criterion 1).
5. Then work through Task 3's four-part spike (backfill volume, deep-link scheme — `gio open 'whatsapp://'` — link stability across a restart and airplane-mode, and the de-link event taxonomy) exactly as `08-01-PLAN.md` Task 3 describes, and record all four answers.

### Awaiting

The user's real-device verification of Task 2's human-check step (now past two rounds of live-bug fixes — DSN pragma syntax, then premature post-pair disconnect in both the fresh-pair and already-linked paths), followed by Task 3's four spike answers (backfill row count + date range + DB size; deep-link scheme result; restart/airplane-mode observations; de-link event name + whether captured rows survived) plus an "approved" or corrective response. `08-01-PLAN.md`'s `<output>` block requires both to be recorded verbatim in this SUMMARY before the plan can be marked complete — Plans 08-02 and 08-03 consume them directly (Task 3's deep-link answer, in particular, may require correcting `deeplink.go`).

---
*Phase: 08-whatsapp-conversations-managed-risk*
*Completed: 2026-08-10 (halted — see Checkpoint above)*
