---
phase: 08-whatsapp-conversations-managed-risk
plan: 03
subsystem: api
tags: [whatsapp, whatsmeow, qr-pairing, http-api, subprocess, go-plugin, rsc.io-qr]

requires:
  - phase: 08-whatsapp-conversations-managed-risk (Plan 08-01)
    provides: "The whatsmeow-backed plugins/whatsapp module — the ASCII -link terminal QR flow (link.go/pairwait.go) this plan's refactor shares a core with, and the store-lock (storelock.go) this plan's kernel-side second layer (SuspendInstance) backs up"
provides:
  - "An amended 08-UI-SPEC.md: the QR panel's full component contract (sizing, copy, state coverage, entry points) and the widened Groups/Contacts match-field table, both dated and pointer-superseded from the sections they replace"
  - "The audited rsc.io/qr encoder, pinned in plugins/whatsapp/go.mod only — the kernel binary carries no QR dependency"
  - "plugins/whatsapp: a -link-json machine-readable link mode (runLinkJSON) sharing one QR-channel/store-lock core (runLinkCore) with the unchanged ASCII -link mode, emitting newline-delimited qr/paired/error/timeout JSON events with the raw pairing payload never leaving the process as text"
  - "kernel/supervisor.Supervisor.SuspendInstance: stops one named running instance for a caller-held duration, resuming it by re-reading the live config at resume time"
  - "kernel/httpapi: POST/GET/DELETE /api/config/whatsapp-link — a raw-subprocess link-session surface outside the go-plugin gRPC handshake, with a deadline-reaped in-memory session store and a concurrent-session cap"
affects: [08-04-whatsapp-in-app-pairing-ui]

actuals:
  tokens: 27325
  tasks: 3
  commits: 7

tech-stack:
  added: ["rsc.io/qr v0.2.0 (plugins/whatsapp module only — zero transitive deps, direct PNG output)"]
  patterns:
    - "Shared link-flow core, two emitters: runLinkCore (plugins/whatsapp/link.go) drives one QR-channel/store-lock sequence; a linkEmitter interface (code/alreadyLinked/pairingAccepted/loggedIn) lets asciiLinkEmitter (terminal text, byte-identical to Plan 08-01) and jsonLinkEmitter (newline-delimited JSON) present identical lifecycle events without duplicating the whatsmeow-driving logic"
    - "Raw-subprocess kernel surface outside go-plugin: kernel/httpapi/whatsapplink.go execs the discovered plugin binary directly (exec.CommandContext, -link-json -path) rather than through the SourcePlugin gRPC contract — the established DescribePluginType trial-launch precedent extended to a long-lived, streaming-output subprocess instead of a one-shot RPC call"
    - "Suspend/resume as a narrow Host.Reconcile diff: Supervisor.SuspendInstance reuses Host.Reconcile (the same launch/kill discipline Apply already uses) against the current source map minus one name, rather than inventing a second plugin-lifecycle mechanism; resume re-reads config fresh at call time, not a value snapshotted at suspend time"
    - "In-memory session store with deadline reaper + concurrent cap: linkSessionStore mirrors no existing pattern in this codebase (first kernel surface holding live subprocess handles across multiple HTTP requests) — a background goroutine reaps sessions past a fixed deadline, and register() rejects beyond a small constant so an abandoned browser tab cannot accumulate subprocesses"

key-files:
  created:
    - kernel/httpapi/whatsapplink.go
    - kernel/httpapi/whatsapplink_test.go
    - kernel/supervisor/suspend_test.go
    - plugins/whatsapp/link_test.go
  modified:
    - .planning/phases/08-whatsapp-conversations-managed-risk/08-UI-SPEC.md
    - .planning/phases/08-whatsapp-conversations-managed-risk/08-RESEARCH.md
    - plugins/whatsapp/go.mod
    - plugins/whatsapp/link.go
    - plugins/whatsapp/main.go
    - kernel/supervisor/supervisor.go
    - kernel/httpapi/routes.go
    - cmd/topos/main.go
    - docs/api.md
    - kernel/httpapi/agent_test.go
    - kernel/httpapi/contract_test.go
    - kernel/httpapi/live_config_test.go
    - kernel/httpapi/agent_live_config_test.go
    - kernel/httpapi/config_test.go

key-decisions:
  - "QR encoder: rsc.io/qr v0.2.0 selected on the first candidate check — real tagged versions (v0.1.0/v0.2.0), zero transitive dependencies, direct (*qr.Code).PNG() []byte output, and Russ Cox (rsc.io) maintainer identity; no fallback to skip2/go-qrcode or yeqown/go-qrcode was needed"
  - "linkEvent wire vocabulary: kind ∈ {qr, paired, error, timeout}; error carries {code, message} with code ∈ {store_in_use, link_failed} — the kernel maps store_in_use to the distinct whatsapp_store_in_use API code and everything else to the generic link_failed"
  - "Router's return type widened to (chi.Router, *linkSessionStore) — not explicitly specified by the plan's 'thread a Suspender parameter beside applier' text, but required so cmd/topos/main.go can defer the store's Shutdown() and satisfy the must_have that kernel shutdown terminates every live link subprocess (Rule 3, documented as a deviation below)"
  - "Poll-time plugin-side link failures (store_in_use/link_failed) are reported as HTTP 200 with a state:\"error\" body, not the shared error envelope — the poll itself succeeded; the underlying link attempt is what failed. Only handler-level failures (bad request, unknown plugin/session, subprocess-start failure, session cap) use the shared error envelope"
  - "linkSessionDeadline set to 5 minutes (generous over whatsmeow's own ~20-60s QR rotation plus link.go's own 60s post-pair-login wait) with a 15s reaper tick and a 4-session concurrent cap — all Claude's-discretion constants per the plan's own transport/protocol note"

requirements-completed: [SRC-03]

coverage:
  - id: D1
    description: "08-UI-SPEC.md carries a dated 'Amendment — In-App QR Panel (D-01/D-02/D-03) and the Widened Match-Field Table (D-05)' section with dated supersession pointers from the two sections it replaces, and an intact Checker Sign-Off block recording the amendment's provenance"
    verification:
      - kind: other
        ref: "grep -c 'Amendment — In-App QR Panel' 08-UI-SPEC.md (4 occurrences: heading + 2 pointers + sign-off note); manual read confirms QR sizing/copy/state-coverage/entry-point/match-table content"
        status: pass
    human_judgment: false
  - id: D2
    description: "The QR-to-PNG encoder is audited via the manual Go-ecosystem legitimacy protocol before being added to go.mod; the verdict is recorded as a dated go.mod comment and a replaced row in 08-RESEARCH.md's Package Legitimacy Audit table; the root go.mod is unchanged"
    verification:
      - kind: unit
        ref: "go list -m -versions rsc.io/qr; go list -m -json rsc.io/qr@latest (both run live, 2026-08-10); git diff --stat go.mod shows no change"
        status: pass
    human_judgment: false
  - id: D3
    description: "plugins/whatsapp gains a -link-json machine-readable link mode sharing one QR-channel core with the unchanged ASCII -link mode; every linkEvent kind marshals to one newline-free JSON line; a qr event's png_data_uri starts with the PNG data-URI prefix with a positive expires_in_seconds; the raw QR payload never appears in any emitted event or diagnostic log line; a store-lock failure emits a distinguishably-coded error event; -link and -link-json are mutually exclusive"
    verification:
      - kind: unit
        ref: "plugins/whatsapp: go test ./... -run 'TestLinkEvent|TestLinkJSON|TestLinkASCII|TestValidateLinkFlags' -v (all pass); CGO_ENABLED=0 go vet ./... clean"
        status: pass
    human_judgment: false
  - id: D4
    description: "kernel/httpapi exposes POST/GET/DELETE /api/config/whatsapp-link: the discovered-binary allowlist check runs before any subprocess is spawned (proved via an injected spawner fake recording zero invocations); poll returns the latest event and 404s on retirement/unknown ids; cancel terminates the subprocess; a session past its deadline is reaped; a start naming an existing instance suspends it before spawning and resumes it on any terminal state including error; SuspendInstance on an absent name is a no-op; the store-in-use failure maps to a code distinct from the generic link_failed; sdk/contract_test.go's four-RPC allowlist is unchanged; docs/api.md documents all three routes and their error codes"
    verification:
      - kind: unit
        ref: "go test ./kernel/httpapi/... ./kernel/supervisor/... -run 'WhatsAppLink|Suspend' -v (all pass); cd sdk && go test ./... -run TestContract (pass); go build ./... && make test-portable (full workspace, all 8 modules pass)"
        status: pass
    human_judgment: false

duration: ~1h50min
completed: 2026-08-10
status: complete
---

# Phase 8 Plan 3: WhatsApp In-App Pairing (Kernel Half) Summary

**Kernel-side D-01 build: an amended UI-SPEC for the QR panel, the audited rsc.io/qr encoder confined to the plugin module, a `-link-json` machine-readable link mode sharing one QR-channel core with Plan 08-01's unchanged ASCII flow, and a raw-subprocess `POST/GET/DELETE /api/config/whatsapp-link` session surface that suspends/resumes a running instance around the link — all outside the locked four-RPC plugin contract.**

## Performance

- **Duration:** ~1h50min
- **Started:** 2026-08-10 (continuation of Phase 8 execution)
- **Completed:** 2026-08-10
- **Tasks:** 3 of 3 complete
- **Files modified:** 18 (4 new, 14 modified — 4 of the 14 modifications are Rule 3 compile-dependency fixes to existing `Router(...)` call sites and route-allowlist AST guards outside this plan's declared `files_modified`)

## Accomplishments

- `08-UI-SPEC.md` amended with the QR panel's full, concrete component contract (192px image sizing, instruction/countdown copy, five-state coverage, two entry points) and the widened `Groups`/`Contacts` match-field table — dated, pointer-superseded from the two sections it replaces, Checker Sign-Off intact with a provenance line
- `rsc.io/qr` v0.2.0 audited live via the manual Go-ecosystem protocol (real tagged versions, zero transitive deps, direct PNG output, Russ Cox maintainer identity) and pinned in `plugins/whatsapp/go.mod` only — verified the root `go.mod` carries no QR dependency
- `plugins/whatsapp/link.go` refactored into a shared `runLinkCore` driven by a `linkEmitter` interface: `asciiLinkEmitter` reproduces Plan 08-01's terminal flow byte-for-byte (pinned by `TestLinkASCII`); `jsonLinkEmitter` emits newline-delimited `qr`/`paired`/`error`/`timeout` JSON events, with the raw QR pairing payload proven absent from every emitted event and diagnostic log line
- `kernel/supervisor.Supervisor.SuspendInstance` added: stops one named running instance via the same `Host.Reconcile` discipline `Apply` already uses, returns a resume closure that re-reads live config at call time; an absent name is a deliberate no-op for the Add-Source flow
- `kernel/httpapi/whatsapplink.go` added: the full link-session HTTP surface (start/poll/cancel), spawning the plugin binary in `-link-json` mode as a raw `exec.CommandContext` subprocess validated against `pluginhost.DiscoverAllBinaries` strictly before execution, with an in-memory session store (deadline reaper, 4-session cap, kernel-shutdown cleanup)
- All automated verification passes: `go build ./...`, `make test-portable` (full 8-module workspace), `cd sdk && go test ./... -run TestContract` (four-RPC allowlist unchanged), and every task's own named test subset

## Task Commits

1. **Task 1: Amend the UI contract for the QR panel, and audit the QR encoder before adding it**
   - `8a6679b` (docs) — UI-SPEC amendment, rsc.io/qr audit recorded in go.mod and 08-RESEARCH.md
2. **Task 2: Give the plugin a machine-readable link mode the kernel can relay**
   - `27f8ec9` (test) — RED: failing tests for linkEvent marshaling, QR payload shape, payload-leak guard, error codes, ASCII instruction line, flag mutual exclusivity
   - `f4da8f8` (feat) — GREEN: `runLinkCore`/`linkEmitter`/`asciiLinkEmitter`/`jsonLinkEmitter`, `-link-json` flag
3. **Task 3: Expose the link session over HTTP, suspending any running instance for its duration**
   - `3f49a16` (test) — RED: failing tests for `Supervisor.SuspendInstance` (no-op on absent name, stop-then-resume)
   - `7830954` (feat) — GREEN: `SuspendInstance` implementation
   - `c93367c` (test) — RED: failing tests for the whatsapp-link HTTP session surface
   - `62b2db0` (feat) — GREEN: `whatsapplink.go`, `routes.go`/`main.go` wiring, `docs/api.md`, and the Rule 3 compile-dependency fixes

**Plan metadata:** this commit (docs: finalize SUMMARY — plan complete)

_Note: `tdd="true"` was applied to Tasks 2 and 3 — each followed strict RED/GREEN discipline (a compile-failing test commit followed by the implementation that turns it green), verified by physically removing the new implementation file and re-running `go vet`/`go test` to confirm the RED failure before restoring it, for both the plugin-side (Task 2) and kernel-side (Task 3) halves. No REFACTOR commit was needed for either task — the GREEN implementation was already the intended shape._

## Files Created/Modified

- `.planning/phases/08-whatsapp-conversations-managed-risk/08-UI-SPEC.md` — Amendment section (QR panel contract, widened match-field table), dated supersession pointers
- `.planning/phases/08-whatsapp-conversations-managed-risk/08-RESEARCH.md` — rsc.io/qr audit verdict replacing the UNVERIFIED placeholder row
- `plugins/whatsapp/go.mod` — `require rsc.io/qr v0.2.0` with a dated audit comment
- `plugins/whatsapp/link.go` — `runLinkCore`, `linkEmitter`, `asciiLinkEmitter`, `jsonLinkEmitter`, `linkEvent`/`linkEventKind`, `newQRLinkEvent`, `runLinkJSON` (rewritten; `runLinkCLI` now a one-line wrapper over the shared core)
- `plugins/whatsapp/link_test.go` — RED/GREEN tests for the above
- `plugins/whatsapp/main.go` — `-link-json` flag, `validateLinkFlags`
- `kernel/supervisor/supervisor.go` — `SuspendInstance`
- `kernel/supervisor/suspend_test.go` — RED/GREEN tests for `SuspendInstance`
- `kernel/httpapi/whatsapplink.go` — `Suspender`, `linkSpawner`/`execLinkSpawner`, `linkSession`/`linkSessionStore`, `WhatsAppLinkStartHandler`/`PollHandler`/`CancelHandler`
- `kernel/httpapi/whatsapplink_test.go` — RED/GREEN tests for the above
- `kernel/httpapi/routes.go` — `Suspender` parameter, three new route registrations, `Router` now returns `(chi.Router, *linkSessionStore)`
- `cmd/topos/main.go` — passes `sup` as the new `Suspender` argument, `defer linkStore.Shutdown()`
- `docs/api.md` — the three whatsapp-link routes, poll-until-terminal contract, three new error codes
- `kernel/httpapi/agent_test.go`, `contract_test.go`, `live_config_test.go`, `agent_live_config_test.go` — updated `Router(...)` call sites for the new parameter/return shape (Rule 3)
- `kernel/httpapi/config_test.go` — `TestRoutesGuard_NonGetRoutesScopedToConfig`'s allowlist extended with the two new mutating routes (Rule 3)

## Decisions Made

See `key-decisions` in frontmatter above. Summary: `rsc.io/qr` passed the audit on the first candidate; the `linkEvent` wire vocabulary and its two error codes; `Router`'s return type widened beyond the plan's literal "thread a Suspender parameter" text so kernel-shutdown cleanup is actually wireable; poll-time plugin failures use `200 state:"error"` rather than the shared error envelope; and the session store's deadline/reaper/cap constants.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Recorded the rsc.io/qr audit verdict in 08-RESEARCH.md**
- **Found during:** Task 1
- **Issue:** `08-RESEARCH.md` is not in Task 1's declared `files_modified`, but the plan's own action text and acceptance criteria explicitly require replacing the UNVERIFIED placeholder row in its Package Legitimacy Audit table
- **Fix:** Added the audit verdict as a new table row, preserving the file's existing structure
- **Files modified:** `.planning/phases/08-whatsapp-conversations-managed-risk/08-RESEARCH.md`
- **Committed in:** `8a6679b`

**2. [Rule 3 - Blocking] Widened `Router`'s return type to `(chi.Router, *linkSessionStore)`**
- **Found during:** Task 3
- **Issue:** The plan's must_haves require "the link endpoint never leaves an orphaned link subprocess: cancelling the session, the session expiring, and the kernel shutting down each terminate the subprocess" — but `Router`'s only other change the plan names is threading a `Suspender` parameter beside `applier`. With no other seam to reach `cmd/topos/main.go`'s shutdown sequence, the kernel-shutdown guarantee had no wiring path
- **Fix:** `Router` now also returns the constructed `*linkSessionStore`; `cmd/topos/main.go` captures it and defers `Shutdown()`, mirroring how `sup.Shutdown()` already kills every pluginhost-launched subprocess on kernel exit
- **Files modified:** `kernel/httpapi/routes.go`, `cmd/topos/main.go`
- **Committed in:** `62b2db0`

**3. [Rule 3 - Blocking] Updated four existing `Router(...)` call sites and two AST route-allowlist guards outside this plan's declared files**
- **Found during:** Task 3
- **Issue:** Adding a `Suspender` parameter and a second return value to `Router` is a breaking signature change; `kernel/httpapi/agent_test.go`, `contract_test.go`, `live_config_test.go`, and `agent_live_config_test.go` all called the old 8-argument, single-return signature directly and would not compile. Separately, `config_test.go`'s `TestRoutesGuard_NonGetRoutesScopedToConfig` and `contract_test.go`'s `TestContract_MutatingRoutesAreConfigScoped` are deliberate AST allowlists of every non-GET route `routes.go` may register — both failed once the two new mutating routes were registered, exactly as their own doc comments say they are designed to
- **Fix:** Updated all four call sites to pass `&fakeSuspender{}` and capture the second return value (discarded where unused); extended both allowlists with `{Post, /api/config/whatsapp-link}` and `{Delete, /api/config/whatsapp-link/{session}}`
- **Files modified:** `kernel/httpapi/agent_test.go`, `contract_test.go`, `live_config_test.go`, `agent_live_config_test.go`, `config_test.go`
- **Committed in:** `62b2db0`

---

**Total deviations:** 3 auto-fixed (1 missing-critical documentation gap, 2 blocking compile/guard dependencies)
**Impact on plan:** All three necessary for correctness or to satisfy the plan's own must_haves/acceptance criteria. No scope creep — every fix was a direct, mechanical consequence of Task 3's own design, not new functionality.

## Issues Encountered

None beyond the deviations above. `plugins/whatsapp`'s real whatsmeow-driven connection/event-handling glue (the QR-channel loop, `Client.Connect()`) has no live-network unit test — as Plan 08-01's own summary noted, this is inherent to the domain (no real WhatsApp session available in this execution environment), not a shortcut. This plan's own new logic (`linkEvent` marshaling, QR-image rendering, the emitter presentation split, the HTTP session lifecycle, suspend/resume) is fully unit-tested via injected fakes, matching the pattern Plan 08-01 already established for its own pure-function seams.

## User Setup Required

None. This plan adds no new external service, credential, or environment variable — the QR encoder is a Go module dependency, and the link-session HTTP surface is served on the kernel's existing loopback listener.

## Next Phase Readiness

- **Ready for Plan 08-04** (the frontend QR panel): `POST/GET/DELETE /api/config/whatsapp-link`'s exact request/response shapes are documented in `docs/api.md` and pinned by `kernel/httpapi/whatsapplink_test.go`; `08-UI-SPEC.md`'s amendment gives the component contract (sizing, copy, state coverage, entry points) to build against directly.
- **No open blockers.** All three tasks' automated verification passes; `sdk/contract_test.go`'s four-RPC allowlist is unchanged, confirming the link-mode surface stayed entirely outside the locked plugin contract as designed.

---
*Phase: 08-whatsapp-conversations-managed-risk*
*Completed: 2026-08-10*
