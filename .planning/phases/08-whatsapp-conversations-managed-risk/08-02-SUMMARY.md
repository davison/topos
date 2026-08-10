---
phase: 08-whatsapp-conversations-managed-risk
plan: 02
subsystem: source-plugin
tags: [whatsmeow, whatsapp, health-taxonomy, matching, ast-scan, go]

requires:
  - phase: 08-whatsapp-conversations-managed-risk
    provides: "Plan 08-01's whatsmeow pin, persistent-connection plugin architecture, chat_jid-first message store, and all four Task 3 real-device spike answers (de-link surfaces as LoggedOut/ConnectFailure reason 401, local rows survive de-link, deep links are wa.me/web.whatsapp.com not whatsapp://, backfill volume)"
  - phase: 04-signal-conversations
    provides: "D-06 anti-injection precedent (never a contact's self-chosen push/profile name as a match candidate) and openGuarded's named-error-per-cause shape this plan's health taxonomy mirrors"
provides:
  - "plugins/whatsapp/health.go: a six-state healthState taxonomy (notLinked/linked/delinked/banned/expired/streamReplaced) with five distinct, honest, non-data-loss-implying last_error messages, and healthStateFromLogoutReason translating whatsmeow's ConnectFailureReason codes into the correct named cause"
  - "plugins/whatsapp/plugin.go Match: the health guard now runs BEFORE the zero-keywords early return, so every non-healthy state returns codes.Unavailable, never an empty success, even with no keywords in either field"
  - "1:1 chat matching on the user's own saved address-book contact name only (chats.contact_name, D-05/D-06/D-07) — widened two-field match vocabulary (groups, contacts) rendered by the existing Phase 7 Match-Fields Form with zero new frontend code"
  - "plugins/whatsapp/readonly_test.go and outbound_hosts_test.go: non-vacuous, negative-controlled AST scans locking the read-only/no-egress boundary, both passing under make test-portable"
affects: [08-03-whatsapp-in-app-pairing, 08-04]

actuals:
  tokens: 18673
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "healthState taxonomy: a mutex-guarded (state healthState, detail string) pair on SourcePlugin, replacing a single healthy/lastError flag — state.Message() is a fixed per-cause template (health.go's healthMessages map) that a test suite can assert is distinct per cause; detail carries only dynamic, non-templated context (e.g. a TemporaryBan reason code) appended in parentheses, never substituted for the template"
    - "Match's health guard runs strictly BEFORE the zero-keywords early return — the ordering itself is the correctness property T-08-05 depends on, not just the branch content"
    - "Two-field disjoint matching: eligibleChats takes two separate keyword lists and dispatches each chat to exactly one of them by its own IsGroup flag — a value typed into the wrong field can never cross-match, by construction rather than by a runtime guard"
    - "Additive idempotent SQLite migration via PRAGMA table_info column-existence guard (messagestore.go's columnExists/migrateAddContactNameColumn) — the pattern for widening this plugin's own local schema without a versioned migration framework"
    - "AST-scan enforcement for a plugin whose danger surface is a Go client API, not SQL text: disallowedClientSelectors is a plain selector-name set derived by grepping the pinned dependency's own exported method surface from the module cache, not from memory, with a comment recording the exact derivation command"

key-files:
  created:
    - plugins/whatsapp/health.go
    - plugins/whatsapp/health_test.go
    - plugins/whatsapp/delink_test.go
    - plugins/whatsapp/match_test.go
    - plugins/whatsapp/readonly_test.go
    - plugins/whatsapp/outbound_hosts_test.go
  modified:
    - plugins/whatsapp/plugin.go
    - plugins/whatsapp/eventhandler.go
    - plugins/whatsapp/connect.go
    - plugins/whatsapp/match.go
    - plugins/whatsapp/messagestore.go

key-decisions:
  - "healthStateFromLogoutReason's three mappings are graded by evidence: 401 (LoggedOut) -> delinked is EMPIRICALLY CONFIRMED by Plan 08-01's real-device spike; 403 (MainDeviceGone, whatsmeow's own comment: aka LOCKED) -> expired and 406 (UnknownLogout, whatsmeow's own comment: aka BANNED) -> banned are INFERRED from whatsmeow's own source comments, not observed live — recorded explicitly in health.go's doc comment so a future spike round can upgrade the inferred mappings to confirmed ones"
  - "connect.go's boot-time branch where an ALREADY-paired device's first Connect() dial fails transiently is reported as healthStateNotLinked (not one of the three new named causes) with the real error carried in detail — none of de-link/ban/expiry honestly describe a transient network hiccup on a device that IS still paired, and whatsmeow's own EnableAutoReconnect self-heals it via a later *events.Connected with zero further code; documented in connect.go's own comment as a deliberate, non-obvious choice within the plan's fixed six-constant taxonomy"
  - "Added *events.ConnectFailure handling (Rule 2 — missing critical functionality): the plan's action text names LoggedOut/TemporaryBan explicitly but not whatsmeow's own generic unrecognised-connect-failure event; without a handler for it, an unrecognised failure would leave healthState stuck at whatever it was before (potentially a stale 'linked' read), violating the 'never silently healthy' requirement in spirit. Mapped to healthStateDelinked per that same requirement."
  - "deeplink.go required NO code change for Task 1: the task's action text ('do not adopt a wa.me HTTPS link under any outcome') was written before Plan 08-01's real-device spike ran; that spike (recorded in 08-01-SUMMARY.md, and restated verbatim in this plan's own orchestrator prompt) empirically confirmed wa.me/web.whatsapp.com as WhatsApp's own documented click-to-chat API and the ONLY working scheme (no WhatsApp Linux client exists to register a bare whatsapp:// URI) — 08-01 already implemented and dated this correction. Kept the already-correct, already-spike-verified implementation rather than reverting to a confirmed-non-functional scheme; treated the plan's own stale pre-spike guidance as superseded by the later, more authoritative evidence it was written to be conditioned on."

requirements-completed: [SRC-03]

coverage:
  - id: D1
    description: "Five named health states (not-linked, linked, de-linked, banned, session-expired, plus stream-replaced) each surface a distinct, honest last_error that never implies data loss"
    requirement: "SRC-03"
    verification:
      - kind: unit
        ref: "plugins/whatsapp/health_test.go#TestHealthState_MessagesNonEmptyAndDistinct"
        status: pass
      - kind: unit
        ref: "plugins/whatsapp/health_test.go#TestHealthState_MessagesNeverImplyDataLoss"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every non-healthy state makes Match return codes.Unavailable, never an empty success, even with zero keywords — the criterion-4 kernel/correlate distinction"
    requirement: "SRC-03"
    verification:
      - kind: unit
        ref: "plugins/whatsapp/delink_test.go#TestDelink_MatchReturnsUnavailableForEveryNonHealthyState"
        status: pass
      - kind: unit
        ref: "plugins/whatsapp/delink_test.go#TestDelink_HealthyEmptyMatchIsSuccessNotError"
        status: pass
    human_judgment: false
  - id: D3
    description: "No failure event (LoggedOut/TemporaryBan/ConnectFailure/StreamReplaced) deletes, truncates, or empties this plugin's own message store"
    requirement: "SRC-03"
    verification:
      - kind: unit
        ref: "plugins/whatsapp/delink_test.go#TestDelinkPreservesStore"
        status: pass
    human_judgment: false
  - id: D4
    description: "1:1 chats match on the user's own saved address-book contact name only; unsaved contacts and remote-supplied push names are never matchable (D-05/D-06/D-07)"
    requirement: "SRC-03"
    verification:
      - kind: unit
        ref: "plugins/whatsapp/match_test.go#TestEligible_ContactMatchesOnlyOnContactsField"
        status: pass
      - kind: unit
        ref: "plugins/whatsapp/match_test.go#TestEligible_PushNameIsNeverACandidate"
        status: pass
      - kind: unit
        ref: "plugins/whatsapp/match_test.go#TestEligible_UnsavedContactNeverMatchesIncludingOwnPhoneNumber"
        status: pass
    human_judgment: false
  - id: D5
    description: "A store created by Plan 08-01 (no contact_name column) opens successfully and preserves its existing rows after the additive migration"
    verification:
      - kind: unit
        ref: "plugins/whatsapp/match_test.go#TestContactNameMigration"
        status: pass
    human_judgment: false
  - id: D6
    description: "Non-vacuous, negative-controlled AST scans enforce the read-only boundary (no send-capable whatsmeow Client selector) and the no-egress boundary (no self-constructed net/http client, no non-allowlisted host literal)"
    verification:
      - kind: unit
        ref: "plugins/whatsapp/readonly_test.go#TestReadOnly_NoSendCapableClientSelector"
        status: pass
      - kind: unit
        ref: "plugins/whatsapp/outbound_hosts_test.go#TestOutboundHosts_NoSelfConstructedHTTPClientOrUnlistedHostLiteral"
        status: pass
      - kind: integration
        ref: "make test-portable"
        status: pass
    human_judgment: false

duration: ~25min
completed: 2026-08-10
status: complete
---

# Phase 8 Plan 2: WhatsApp 1:1 Matching & Health Taxonomy Summary

**Five named WhatsApp health states (not-linked/de-linked/banned/expired/stream-replaced) that never empty the stream on failure, widened matching to 1:1 chats on saved contact names only, and two non-vacuous AST scans locking the read-only and no-egress boundaries.**

## Performance

- **Duration:** ~25 min (commit-timestamp span; research/reading time before the first commit is not reflected)
- **Started:** 2026-08-10T13:33Z (approx, base commit)
- **Completed:** 2026-08-10T13:54Z
- **Tasks:** 3 of 3 complete
- **Files modified:** 11 (6 created, 5 modified)

## Accomplishments

- Replaced Plan 08-01's single non-healthy flag with a six-state named taxonomy (`healthState` in `health.go`): `notLinked`/`linked`/`delinked`/`banned`/`expired`/`streamReplaced`, each with its own fixed, honest `last_error` template that never states or implies previously captured messages were lost
- `healthStateFromLogoutReason` translates whatsmeow's `ConnectFailureReason` codes into the correct cause — 401 (empirically confirmed by Plan 08-01's real spike) maps to de-linked; 403/406 map to expired/banned respectively, based on whatsmeow's own source comments ("aka LOCKED"/"aka BANNED"), explicitly flagged as inferred rather than spike-confirmed
- `Match`'s health guard now runs strictly before the zero-keywords early return — a de-linked plugin asked with no keywords still returns `codes.Unavailable`, never a silent empty success, closing the exact gap `kernel/correlate` depends on to decide whether to wipe previously-synced rows
- Widened matching to 1:1 chats: a new `chats.contact_name` column (additive, idempotent migration), populated exclusively from whatsmeow's own local contact store's `FullName`/`FirstName` — never `PushName`/`BusinessName` — via a new `resolveContactName` helper called on every 1:1 message and on live `*events.Contact` updates
- `matchVocabulary` widened from `["groups"]` to `["groups", "contacts"]`; `eligibleChats` now takes two disjoint keyword lists, so a value typed into the wrong field can never cross-match a chat of the other kind
- Two new AST-scan test files (`readonly_test.go`, `outbound_hosts_test.go`), each with a non-vacuous negative control, enforce: no send-capable/mutating/presence-broadcasting whatsmeow `Client` selector is ever referenced outside test files, and no self-constructed `net/http` client or non-allowlisted absolute-URL literal exists in the package (the two already-audited wa.me/web.whatsapp.com deep-link literals are the sole, exact-match exception)
- `make test-portable` passes end to end across the whole workspace

## Task Commits

1. **Task 1: Name the four failure causes and prove none of them empties the stream** - `2a4425c` (feat)
2. **Task 2: Widen matching to 1:1 chats on saved contact names only (D-05/D-06/D-07)** - `51fdbec` (feat)
3. **Task 3: Lock the read-only boundary with AST scans and negative controls** - `d81cf84` (test)

**Plan metadata:** this commit (docs: finalize SUMMARY — plan complete)

## Files Created/Modified

- `plugins/whatsapp/health.go` (new) — `healthState` taxonomy, `healthMessages` template map, `healthStateFromLogoutReason`
- `plugins/whatsapp/health_test.go` (new) — per-state distinctness, non-empty, no-data-loss-copy, and `Health` RPC branching tests
- `plugins/whatsapp/delink_test.go` (new) — the criterion-4 regression (`Match` returns `codes.Unavailable` for every non-healthy state, a healthy empty match is a real success) and `TestDelinkPreservesStore`
- `plugins/whatsapp/plugin.go` — `SourcePlugin`'s `healthy`/`lastError` fields replaced with `state`/`detail`; `setHealthState`/`healthState`/`currentMessage` accessor trio; `Match`'s health guard reordered above the zero-keywords return; `matchVocabulary` widened; `Match` reads both `groups`/`contacts` fields; `toItem` threads the matched chat's real `IsGroup` into `conversationDeepLink` instead of a hardcoded `true`
- `plugins/whatsapp/eventhandler.go` — routes `LoggedOut`/`TemporaryBan`/`ConnectFailure`/`StreamReplaced` to named states; new `*events.Contact` handler and `resolveContactName`; 1:1 message handling now upserts `contact_name`
- `plugins/whatsapp/connect.go` — `device.ID == nil` and a transient boot-time `Connect()` failure both report `healthStateNotLinked` (with the real error in `detail` for the latter)
- `plugins/whatsapp/match.go` — `candidateNames`/`eligibleChats` rewritten for the two-field, `IsGroup`-dispatched disjoint matching
- `plugins/whatsapp/messagestore.go` — `chats.contact_name` column, `columnExists`/`migrateAddContactNameColumn`, `UpsertContactName`, `Chats()` now selects the new column
- `plugins/whatsapp/match_test.go` (new) — field-disjointness, anti-injection, unsaved-contact non-matchability, exact-case-insensitive matching, no-duplicate union, and `TestContactNameMigration`
- `plugins/whatsapp/readonly_test.go` (new) — `disallowedClientSelectors` AST scan with two negative controls
- `plugins/whatsapp/outbound_hosts_test.go` (new) — `allowedDeepLinkURLLiterals`-gated AST scan with two negative controls (a third-party host, and a lookalike-host bypass attempt)

## Decisions Made

See `key-decisions` in frontmatter above. Summary: the two inferred (vs. empirically confirmed) logout-reason mappings are explicitly flagged in code for future re-verification; the boot-time transient-connect-failure branch (a genuine gap in the plan's fixed six-state taxonomy) is reported as `healthStateNotLinked` with the real error preserved in `detail`, relying on whatsmeow's own auto-reconnect to self-heal; `*events.ConnectFailure` handling was added beyond the plan's literal text to close a "never silently healthy" gap; `deeplink.go` was deliberately left unchanged because the plan's own anti-wa.me instruction predates and is superseded by Plan 08-01's real-device spike confirmation of exactly that scheme.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added `*events.ConnectFailure` handling**
- **Found during:** Task 1 (eventhandler.go)
- **Issue:** The plan's action text names `LoggedOut` and a "temporary-ban event" explicitly, but whatsmeow also dispatches a distinct `*events.ConnectFailure` for connect failures it does not otherwise recognise (its own `connectionevents.go` fallback path) — left unhandled, an unrecognised failure would leave `healthState` at whatever it was before, potentially reporting stale "linked" health after a real failure, contradicting the task's own "never silently healthy" requirement
- **Fix:** Added a `case *events.ConnectFailure` branch mapping to `healthStateDelinked`, capturing the reason code and message as `detail`
- **Files modified:** plugins/whatsapp/eventhandler.go
- **Committed in:** `2a4425c`

**2. [Rule 1 - Bug, non-code] Kept `deeplink.go`'s already-correct wa.me/web.whatsapp.com implementation instead of following the plan's stale anti-wa.me instruction**
- **Found during:** Task 1 (read_first review of `deeplink.go` and `08-01-SUMMARY.md`)
- **Issue:** This plan's action text says "Do not adopt a wa.me HTTPS link under any outcome — the reasons in Plan 08-01 still hold," but Plan 08-01's own real-device spike (which this very plan's orchestrator prompt restates verbatim) empirically found no WhatsApp Linux client exists to register a bare `whatsapp://` scheme, and corrected `deeplink.go` to wa.me/web.whatsapp.com — WhatsApp's own documented click-to-chat API — before this plan began. The instruction was written before that spike ran and was not updated afterward.
- **Fix:** No code change. Left `deeplink.go` as Plan 08-01 shipped it (dated, spike-verified comment already in place); documented the conflict here rather than reverting to a confirmed-non-functional scheme.
- **Files modified:** none
- **Committed in:** n/a (no-op; documented for audit trail)

---

**Total deviations:** 2 (1 auto-fixed missing-critical-functionality addition, 1 documented no-op where the plan's own literal text was superseded by already-verified prior-plan evidence)
**Impact on plan:** Both necessary for correctness/honesty of the health taxonomy. No scope creep — no files outside the plan's declared `files_modified` list were touched.

## Issues Encountered

- whatsmeow's exact `ConnectFailureReason` → named-cause mapping required live research against the pinned module's own source (not training-data recall): confirmed via the module cache at `go.mau.fi/whatsmeow@v0.0.0-20260806224404-e277b766ab33`'s `types/events/events.go` and `connectionevents.go` — in particular that 403/406's Go constant names ("MainDeviceGone"/"UnknownLogout") are misleading relative to what whatsmeow's own comments say the WhatsApp web client actually calls them ("LOCKED"/"BANNED"), and that `*events.TemporaryBan` is a fully separate event type from `*events.LoggedOut`, not a `LoggedOut` sub-case.
- The disallowed-selector set for `readonly_test.go` required enumerating the pinned whatsmeow commit's entire exported `*Client` method surface (128 methods) via `grep` against the module cache and manually categorising each as read-only, local-only-configuration, or send/mutate/broadcast — including catching `DangerousInternals` ("allows access to all unexported methods in Client"), which is not a `Send`/`Set`-prefixed name and would otherwise have been easy to miss.

## User Setup Required

None — no external service configuration required. This plan is code-only against the plugin already linked and verified live in Plan 08-01.

## Next Phase Readiness

- **Ready for Plan 08-03** (in-app pairing UX): the health taxonomy's `healthStateNotLinked` state and its message text ("Use this source's chip menu... or run this plugin binary's -link flag") already reference the D-03/D-04 chip "Re-link…" entry and CLI fallback Plan 08-03 implements — no further health.go changes anticipated.
- **Ready for Plan 08-04**: the two-field `matchVocabulary` (`groups`, `contacts`) is now the stable Describe() contract; `08-UI-SPEC.md`'s match-field table amendment (noted as required in `08-CONTEXT.md`) should reference this vocabulary going forward.
- **No open blockers.** All three tasks' acceptance criteria pass; `make test-portable` is green.
- **Known gap, not a defect:** `connect.go`'s boot-time transient-Connect()-failure branch is not one of the plan's five/six named causes by design (see key-decisions) — if a future round wants a dedicated "connecting" state distinct from `healthStateNotLinked`, that is a new taxonomy member, not a bug in this plan's own scope.

---
*Phase: 08-whatsapp-conversations-managed-risk*
*Completed: 2026-08-10*
