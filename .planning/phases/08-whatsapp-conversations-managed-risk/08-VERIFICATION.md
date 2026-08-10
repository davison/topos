---
phase: 08-whatsapp-conversations-managed-risk
verified: 2026-08-10T14:46:42Z
status: gaps_found
score: 4/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
gaps:
  - truth: "A running (linked) WhatsApp instance's match configuration (which groups/contacts it matches) can be viewed and edited through the existing 'Edit match settings…' chip menu entry, and an already-configured WhatsApp instance can be added to a second webspace via the '+' picker's existing-instance flow — both reused, unmodified Phase 7 UI machinery this phase's own plan text relies on ('the widened two-field vocabulary renders through the existing Match-Fields Form with no new form code')"
    status: failed
    reason: "POST /api/config/describe-plugin always trial-launches a brand-new topos-plugin-whatsapp subprocess against the stored Source config. plugins/whatsapp/main.go's non-link code path unconditionally calls NewSourcePlugin -> startBackgroundClient -> acquireStoreLock (storelock.go, LOCK_EX|LOCK_NB, non-blocking). WhatsApp is the only plugin in this repo that holds a persistent connection for its entire process lifetime, so the real, already-running instance for that source name always already holds this lock. The trial-launch subprocess's acquireStoreLock call therefore always returns ErrStoreInUse, main.go's fatal() exits before goplugin.Serve, the go-plugin handshake never completes, and describePlugin fails with 502 plugin_describe_failed for any WhatsApp instance that is currently linked and running (the normal, intended post-pairing state). 'Edit match settings…' swallows this failure silently (the catch branch in +page.svelte's handleChipEdit leaves editVocabulary = [] with no error surfaced — the modal opens showing zero fields, reading as 'no fields' rather than 'failed'); AddSourceModal.svelte's selectExisting() (the '+ picker -> add an already-configured instance to a second webspace' path) does surface the plugin_describe_failed error but offers no recovery. Confirmed live in the current codebase: plugins/whatsapp/main.go (no describe-only mode exists), plugins/whatsapp/plugin.go:111-121 (NewSourcePlugin unconditionally starts the background client), plugins/whatsapp/connect.go:74-79 (startBackgroundClient acquires the lock before anything else), web/src/routes/w/[webspace]/+page.svelte:189-221 (the swallowed-failure match-edit path), web/src/lib/components/AddSourceModal.svelte:163-182 (selectExisting). Originally identified as CR-01 in 08-REVIEW.md (Critical, issues_found) and remains unfixed at the current commit (b796bf3) — no code change addresses it in any of the four plans."
    artifacts:
      - path: "plugins/whatsapp/main.go"
        issue: "No 'describe-only' mode analogous to -link/-link-json exists; the default (serve) code path always reaches NewSourcePlugin, which always acquires the exclusive store lock before Describe can ever answer, colliding with an already-running instance of the same source"
      - path: "web/src/routes/w/[webspace]/+page.svelte"
        issue: "handleChipEdit's 'match' branch swallows a describePlugin failure into an empty editVocabulary with no user-facing error, so the failure reads as 'no match fields' rather than as a failure — SourceChip.svelte renders 'Edit match settings…' unconditionally for every source type including WhatsApp, with no gate excluding it"
      - path: "web/src/lib/components/AddSourceModal.svelte"
        issue: "selectExisting() (the one-step 'add an existing instance to a second webspace' flow) surfaces plugin_describe_failed for a running WhatsApp instance with no recovery path"
    missing:
      - "Either: DescribePluginHandler/pluginhost reuse the vocabulary already learned from the Describe call pluginhost performs once at launch time for a running instance, instead of trial-launching a second subprocess; or: the WhatsApp plugin's trial-launch path gains a describe-only mode that answers Describe without acquiring the store lock or starting the background client (Describe returns only static constants and needs no live connection or store access)"
      - "A regression test (e2e or Go-level, against a real built topos-plugin-whatsapp binary) covering 'Edit match settings…' against an already-linked WhatsApp instance — this is exactly the gap that let the defect ship unnoticed, since uat-08-whatsapp-qr-link.spec.ts intercepts every describe-plugin call and the hermetic e2e harness never builds a real WhatsApp binary"
---

# Phase 8: WhatsApp Conversations (Managed Risk) Verification Report

**Phase Goal:** User's WhatsApp groups for a topic appear in the webspace stream via a linked-device session, and everything else keeps working when that session breaks
**Verified:** 2026-08-10T14:46:42Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

Roadmap Success Criteria (`.planning/ROADMAP.md`, Phase 8) plus the phase-level must-have this gap concerns:

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | User can link WhatsApp as a device by scanning a QR code, and the session survives service restarts without re-linking | ✓ VERIFIED | CLI `-link` flow (Plan 08-01) and in-app QR panel (Plans 08-03/08-04) both implemented and tested. Live-verified on a real account across three checkpoint rounds (08-01-SUMMARY.md): a real QR scan links a real account; a kernel stop+restart reconnects with NO second QR scan; `/api/sources` reported `reachable:true`; 616 previously-captured message rows were intact post-restart. `plugins/whatsapp/pairwait.go`'s `pairLoginWaiter` proven by unit test (`TestPairLoginWaiter_*`, 5 cases, all pass) to wait for a genuine post-pair login before disconnecting, both fresh-pair and reconnect paths. |
| 2 | Messages from WhatsApp groups whose names match the webspace's matching config appear in the stream alongside every other source, using the Phase-4 chat rendering | ✓ VERIFIED | Live-verified: a real group's day digest ("Camberley — 1 message") served through `/api/webspaces/{ws}/stream` during Plan 08-01's Task 3 spike. `plugins/whatsapp/plugin.go`'s `Match`/`toItem` builds `CONTENT_SHAPE_CHAT_TRANSCRIPT` items exactly as Phase 4's Signal plugin does; `render.go`/`fetchTranscript` route the same rendering policy. Matching widened to groups AND 1:1 saved-contact-name chats (D-05/D-06/D-07), unit-tested (`TestEligible_*`, `TestMatch_ExactCaseInsensitiveOnly`, all pass). See Gap 1 below for a related, narrower failure in *editing* that matching config for an already-linked instance via the UI — the initial setup and live-matching path itself is proven working. |
| 3 | The plugin persists its own message store, so conversations captured while it was running stay browsable regardless of what the WhatsApp desktop app retains | ✓ VERIFIED | `messagestore.go` opens a second, separate SQLite file (`messages.db`, WAL + busy-timeout) distinct from whatsmeow's own `whatsmeow.db` session store — confirmed live: two distinct files under the configured path, 180 KB `messages.db` holding 616 real messages across 134 chats spanning nearly 8 years of history. `Append` is idempotent (`ON CONFLICT DO UPDATE`), unit-tested (`TestMessageStore_AppendIdempotent`, `TestMessageStore_ChatIsolationAndOrdering`). `Fetch`/`fetchTranscript` reads only this local store, never a live whatsmeow call. |
| 4 | De-link, ban, or session expiry surfaces as an explicit plugin-health error in the UI while previously captured messages remain browsable and every other source is unaffected | ✓ VERIFIED | `health.go` declares six named states (`notLinked`/`linked`/`delinked`/`banned`/`expired`/`streamReplaced`), each with a distinct, honest message that never implies data loss (`TestHealthState_MessagesNonEmptyAndDistinct`, `TestHealthState_MessagesNeverImplyDataLoss`, both pass). `Match`'s health guard runs BEFORE the zero-keywords early return, so every non-healthy state returns `codes.Unavailable` rather than an empty success — the exact distinction `kernel/correlate` needs to preserve previously-synced rows (`TestDelink_MatchReturnsUnavailableForEveryNonHealthyState`, `TestDelink_HealthyEmptyMatchIsSuccessNotError`, both pass). `TestDelinkPreservesStore` drives all six failure-causing events through `handleEvent` and asserts the message-store row count is unchanged for every one — a genuine behavioral proof, not presence-only. Live-verified: a real de-link (`LoggedOut`/401) left `messages.db` untouched. Note: WR-02 in 08-REVIEW.md identifies a narrow, unrelated-config-save race during an ACTIVE re-link session that could reject another source's unrelated save — a real but narrow warning-grade gap, not part of this criterion's ordinary de-link/ban/expiry path (Re-link is user-initiated, not part of "session breaks"). |
| 5 (phase-derived, from Plan 08-02's own must_have) | A running WhatsApp instance's match configuration can be viewed/edited through the existing "Edit match settings…" UI, and an existing instance can be added to a second webspace — both reused, unmodified Phase 7 machinery this phase's plans explicitly rely on | ✗ FAILED | See `gaps` in frontmatter — CR-01 (08-REVIEW.md), confirmed live in the current codebase and unfixed. |

**Score:** 4/5 truths verified (1 failed — see Gaps Summary)

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `plugins/whatsapp/messagestore.go` | Plugin's own message-content SQLite store | ✓ VERIFIED | Exists, substantive, wired; `chat_jid`-keyed schema, WAL, idempotent `Append`; unit-tested |
| `plugins/whatsapp/connect.go` | sqlstore open, device lookup, persistent `Client.Connect` | ✓ VERIFIED | Exists, substantive, wired; live-verified against a real account |
| `plugins/whatsapp/link.go` | CLI + machine-readable link modes sharing one core | ✓ VERIFIED | `runLinkCore`/`asciiLinkEmitter`/`jsonLinkEmitter`; both modes unit-tested (`TestLinkASCII`, `TestLinkEvent_*`, `TestLinkJSON_*`) |
| `plugins/whatsapp/storelock.go` | Exclusive per-store-path advisory lock | ✓ VERIFIED | `LOCK_EX|LOCK_NB`; `TestStoreLock_SecondAcquireFails`/`TestStoreLock_ReleaseAllowsReacquire` pass. Note: this same mechanism is the root cause of Gap 1 below — the lock does its job correctly, but the kernel's `describePlugin` trial-launch path was never adapted to WhatsApp's persistent-connection architecture |
| `plugins/whatsapp/plugin.go` | Describe/Match/Fetch/Health | ✓ VERIFIED | Health guard ordering, matched-only item conversion, both content variants implemented and tested |
| `plugins/whatsapp/health.go` | Named health-state taxonomy | ✓ VERIFIED | Six states, distinct honest messages, unit-tested |
| `plugins/whatsapp/digest.go` | Conversation-day digest assembly | ✓ VERIFIED | Local-midnight boundary, run-gap threshold, singular/plural, rune-safe truncation — all unit-tested edge cases pass |
| `plugins/whatsapp/render.go` | Chat-transcript structural markup | ✓ VERIFIED | Reused Phase 4 CONTENT_SHAPE_CHAT_TRANSCRIPT policy; live-rendered in real webspace stream |
| `plugins/whatsapp/readonly_test.go` | AST scan, no send-capable client selector | ✓ VERIFIED | Passes with two negative controls proving non-vacuity |
| `plugins/whatsapp/outbound_hosts_test.go` | AST scan, no unlisted egress host | ✓ VERIFIED | Passes with two negative controls (unrelated third-party host, lookalike-host bypass attempt) |
| `kernel/httpapi/whatsapplink.go` | Link-session HTTP surface (start/poll/cancel) | ✓ VERIFIED | Discovered-binary allowlist check before execution, deadline reaper, session cap, suspend/resume around a running instance; unit-tested. WR-01 (concurrency cap enforced after spawn, not before) is a narrow warning, not a functional break |
| `kernel/supervisor/supervisor.go` (`SuspendInstance`) | Stop/resume a named running instance | ✓ VERIFIED | Unit-tested no-op-on-absent-name and stop-then-resume cases. WR-02 (a concurrent unrelated Apply can race an active suspension) is a narrow warning |
| `web/src/lib/components/QRPanel.svelte` | Single QR panel, both entry points | ✓ VERIFIED | All five states rendered, cancels on both explicit Cancel and unmount (`retireSession` single call site), unit- and e2e-tested |
| `web/src/lib/components/RelinkModal.svelte` | Re-link… dialog wrapping QRPanel | ✓ VERIFIED | Small dialog, no connection/match fields, wired to the chip menu's WhatsApp-gated entry |
| `web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts` | Hermetic Playwright coverage of the QR flow | ✓ VERIFIED | 8/8 tests passing; route-intercepted, no real WhatsApp plugin binary in the harness |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `plugins/whatsapp/eventhandler.go` | `plugins/whatsapp/messagestore.go` | background `AddEventHandler` goroutine appends every `events.Message` | ✓ WIRED | Confirmed by code read and `TestDelinkPreservesStore`'s direct exercise of `handleEvent` → store |
| `plugins/whatsapp/plugin.go` | `plugins/whatsapp/messagestore.go` | `Match`/`Fetch` query the local store only | ✓ WIRED | `p.store.Chats()`/`MessagesForChats` calls confirmed in `plugin.go`; no live whatsmeow call in either RPC |
| `Makefile` | `plugins/whatsapp` | `plugins:` target builds `topos-plugin-whatsapp` CGO-free | ✓ WIRED | `make build` confirmed building the binary at `bin/plugins/topos-plugin-whatsapp`, before the cgo `signal` target |
| `kernel/httpapi/whatsapplink.go` | `plugins/whatsapp/link.go` | spawns the discovered binary in `-link-json` mode, reads NDJSON stdout | ✓ WIRED | `linkSpawner`/`execLinkSpawner` construction confirmed; exercised by `kernel/httpapi/whatsapplink_test.go` |
| `kernel/httpapi/whatsapplink.go` | `kernel/supervisor/supervisor.go` | `SuspendInstance` stops/resumes a running instance around a link session | ✓ WIRED | Confirmed by code read and `suspend_test.go`/`whatsapplink_test.go` |
| `web/src/lib/components/QRPanel.svelte` | `web/src/lib/api.ts` | typed start/poll/cancel client functions | ✓ WIRED | `startWhatsAppLink`/`pollWhatsAppLink`/`cancelWhatsAppLink` imported and called |
| `web/src/lib/components/AddSourceModal.svelte` | `web/src/lib/components/QRPanel.svelte` | Step 1's not-linked branch renders the panel inline | ✓ WIRED | Confirmed by code read; e2e cases 1/2/3/8 exercise this live in a browser |
| `web/src/lib/components/SourceChip.svelte` | `web/src/routes/w/[webspace]/+page.svelte` | Re-link… entry raises a `relink` `onedit` kind | ✓ WIRED | Confirmed by code read; e2e case 7 exercises this live in a browser |
| `web/src/routes/w/[webspace]/+page.svelte` (`handleChipEdit`, `'match'` kind) | `describePlugin` → `plugins/whatsapp` trial-launch | Edit match settings for an existing WhatsApp source | ✗ NOT_WIRED (for a running instance) | See Gap 1 — the trial-launch always collides with the running instance's exclusive store lock |

### Data-Flow Trace (Level 4)

| Artifact | Data Variable | Source | Produces Real Data | Status |
| --- | --- | --- | --- | --- |
| `plugins/whatsapp/plugin.go` `Match` → `toItem` | `digests` | `p.store.MessagesForChats` (real SQLite reads, not a fixture/mock) | Yes — live-verified 616 real messages / 134 real chats | ✓ FLOWING |
| `web/src/lib/components/QRPanel.svelte` `qrDataUri` | `session.png_data_uri` | `pollWhatsAppLink` → kernel → plugin's `rsc.io/qr`-rendered PNG data URI | Yes — real kernel round trip, exercised by e2e cases 1/2 | ✓ FLOWING |
| `plugins/whatsapp/health.go` `Message()` | `healthMessages` map | Fixed per-state templates, read by `Health`/`Match` | Real distinct text per state, not a shared placeholder | ✓ FLOWING |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| whatsapp plugin's own test suite | `cd plugins/whatsapp && CGO_ENABLED=0 go test ./... -v` | 51/51 subtests pass, 0 failures | ✓ PASS |
| Full workspace build + test | `make build` then `make test-portable` | Both succeed across all 8 Go modules including `plugins/whatsapp` | ✓ PASS |
| AST read-only/egress scans are non-vacuous | `go test ./... -run 'TestReadOnly|TestOutboundHosts' -v` | Pass, including negative-control fixtures | ✓ PASS |
| Web unit suite | `cd web && npm run test -- --run` | 642/642 tests pass across 38 files | ✓ PASS |
| Web type-check | `cd web && npm run check` | 0 errors, 9 pre-existing warnings unrelated to this phase | ✓ PASS |
| Full Playwright e2e suite | `make e2e` | 34/34 tests pass, including all 8 new WhatsApp QR-flow cases | ✓ PASS |
| CR-01 reproduction (static confirmation, not live-executed against a real WhatsApp binary) | Code read of `plugins/whatsapp/main.go`, `plugin.go`, `connect.go`, `storelock.go`, `+page.svelte`, `AddSourceModal.svelte` | Confirms the described-in-08-REVIEW.md collision path is real and unmitigated at the current commit | ✗ FAIL (confirms Gap 1) |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes are declared by this phase's plans or referenced by ROADMAP success criteria; this phase's verification is carried by its Go/web unit suites and the Playwright e2e suite instead. Step 7c: SKIPPED (no declared probes).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| SRC-03 | 08-01, 08-02, 08-03, 08-04 | WhatsApp plugin runs as a whatsmeow linked device with its own persistent message store; degrades gracefully on de-link/ban; matches on group names | ✓ SATISFIED (with a caveat) | Core capability (link, capture, persist, match, digest, render, degrade honestly) is live-verified end to end against a real account. The one caveat is Gap 1 — editing an already-linked instance's match configuration through the UI is broken, which is a real gap in the *matching config* editing workflow this same requirement description ("matches on group names") implies stays usable post-setup. `.planning/REQUIREMENTS.md` line 30 still shows SRC-03 as `[ ]` Pending / status `Pending` — consistent with verification not yet having passed; do not mark it complete until Gap 1 is closed or an override is recorded. |

No orphaned requirements — SRC-03 is the only ID `.planning/ROADMAP.md` maps to Phase 8, and all four plans declare `requirements: [SRC-03]`.

### Anti-Patterns Found

No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers found in any Go or Svelte/TypeScript file this phase created or modified (`plugins/whatsapp/*.go`, `kernel/httpapi/whatsapplink.go`, `kernel/supervisor/supervisor.go`, `web/src/lib/components/QRPanel.svelte`, `RelinkModal.svelte`, `AddSourceModal.svelte`, `SourceChip.svelte`, `WebspaceHeader.svelte`, `+page.svelte`, `api.ts`, `plugin-fields.ts`). No blocker-grade debt markers.

| File | Line | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| (none found) | — | — | — | — |

### Human Verification Required

None. Gap 1 is programmatically confirmed (a code-level, deterministic collision, not a judgment call), and the roadmap success criteria that ARE true were confirmed by direct code reading plus already-recorded live-device evidence in 08-01-SUMMARY.md (three rounds of real checkpoint feedback, already human-verified during execution). No further human testing is needed to resolve this verification's own status — a closure plan for Gap 1 is the next step.

### Gaps Summary

The phase's headline promise — link a real WhatsApp account by QR, see matching groups' conversations as day digests in the stream, keep browsing previously-captured messages when the session breaks, with every other source unaffected — is genuinely, live-verifiably true. Three of the four ROADMAP success criteria are cleanly verified; the fourth (de-link/ban/expiry degradation) is also verified for its own ordinary path, with only a narrow, separately-flagged warning (WR-02) about an active re-link session racing an unrelated config save.

The one real gap is CR-01 from `08-REVIEW.md`, confirmed still present in the code at the current commit: any UI flow that trial-launches the WhatsApp plugin against an **already-configured, already-running** instance — "Edit match settings…" on the chip, and "add this existing instance to a second webspace" from the "+" picker — always fails, because the trial-launch subprocess collides with the real instance's exclusive store lock. This is structurally invisible to the phase's own e2e suite (which never builds a real `topos-plugin-whatsapp` binary and intercepts every `describe-plugin` call), so it was only caught by code review, not by any automated gate. It affects exactly the steady state a linked WhatsApp source is normally in, and "Edit match settings…" fails silently (empty form, no error) rather than obviously — the worse of the two failure modes described in 08-REVIEW.md.

This does not block the phase's core, most novel, and highest-risk deliverable (the linked-device connection, its own message store, and its honest degradation), all of which are proven live against a real account. It does block a piece of Phase-7-inherited UI functionality this phase's own plans explicitly claim ("the widened two-field vocabulary renders through the existing Match-Fields Form with no new form code") — that claim is only half true: it renders, but only for a not-yet-running instance.

**This looks intentional only in the sense that no plan or SUMMARY claims it was fixed** — 08-REVIEW.md flags it, and no subsequent commit or plan addresses it. If the project wants to accept this as a known, documented limitation to close in a follow-up rather than block the phase, an override can be recorded:

```yaml
overrides:
  - must_have: "A running (linked) WhatsApp instance's match configuration can be viewed and edited through the existing 'Edit match settings…' chip menu entry, and can be added to a second webspace via the existing-instance picker flow"
    reason: "{why this is acceptable to ship as a known gap, and when it will be closed}"
    accepted_by: "{name}"
    accepted_at: "{ISO timestamp}"
```

Absent that, a small closure plan (Fix option (a) or (b) from 08-REVIEW.md's CR-01) should land before the phase and requirement SRC-03 are marked complete.

---

_Verified: 2026-08-10T14:46:42Z_
_Verifier: Claude (gsd-verifier)_
