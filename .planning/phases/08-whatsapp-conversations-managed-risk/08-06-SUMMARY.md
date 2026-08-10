---
phase: 08-whatsapp-conversations-managed-risk
plan: 06
subsystem: api
tags: [go, whatsapp, whatsmeow, os-exec, hclog, gap-closure]

# Dependency graph
requires:
  - phase: 08-whatsapp-conversations-managed-risk
    provides: QRPanel.svelte's tolerance for pairing_accepted/already_linked and docs/api.md's wire contract for both (plan 08-05), the whatsapp-link HTTP surface (plan 08-03), 08-UAT.md's G-08-1 diagnosis
provides:
  - plugins/whatsapp/link.go emits pairing_accepted and already_linked non-terminal wire events on stdout (device-id-free), in addition to their existing stderr diagnostics
  - kernel/httpapi/whatsapplink.go's newExecLinkSpawner(logger) constructor, capturing the link subprocess's stderr into a named hclog sublogger instead of discarding it to /dev/null
  - First automated test coverage of the production exec spawner (kernel/httpapi/whatsapplink_exec_test.go): streaming, argv, stderr capture, trailing-partial flush, environment inheritance, kill
  - Kernel-side proof that both progress states are non-terminal end to end (poll stays 200, no resume, no diagnostic fields leak into the response body) behind the pre-existing exactly-once terminal contract
affects: [08-07]

# Actuals (#2632)
actuals:
  tokens: 8220
  tasks: 3
  commits: 5

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Non-*os.File io.Writer assigned to cmd.Stderr so os/exec runs its own copy goroutine and cmd.Wait() waits for it — never cmd.StderrPipe() read concurrently with Wait"
    - "Line-oriented capture writer: buffer to next newline, emit each complete line through a named sublogger, retain trailing partial line, flush once after Wait() returns (race-free by construction, not by locking discipline alone)"
    - "linkSpawner as a constructor (newExecLinkSpawner(logger)) rather than a free function, so production wiring can thread a caller-supplied dependency through while test doubles (fakeSpawner) stay untouched"

key-files:
  created:
    - kernel/httpapi/whatsapplink_exec_test.go
  modified:
    - plugins/whatsapp/link.go
    - plugins/whatsapp/link_test.go
    - kernel/httpapi/whatsapplink.go
    - kernel/httpapi/whatsapplink_test.go
    - kernel/httpapi/routes.go

key-decisions:
  - "linkEventKind's doc comment now states which three of six kinds are terminal, naming kernel/httpapi/whatsapplink.go's isTerminalKind as the hand-maintained mirror that must not drift — per the plan's own instruction, not an added scope item"
  - "stderrLineLogger's buffer cap fixed at 64 KiB (T-08-17) — a value, not specified by the plan, chosen to match the existing stdout scanner's own 64 KiB starting buffer size for a consistent order-of-magnitude across both streams"
  - "Task 3 test names renamed at authoring time from the initially-drafted TestJSONLinkEmitter_* to TestLinkJSON_* so the plan's own fixed verify command (-run 'TestLink') actually picks them up — a mechanical fix, not a deviation, since no plan text mandated the literal draft names"

requirements-completed: [SRC-03]

coverage:
  - id: D1
    description: "plugins/whatsapp/link.go's jsonLinkEmitter emits pairing_accepted and already_linked as single-key, device-id-free events on stdout, with existing stderr diagnostics unchanged"
    requirement: SRC-03
    verification:
      - kind: unit
        ref: "plugins/whatsapp/link_test.go#TestLinkEvent_ProgressEventsCarryOnlyKind"
        status: pass
      - kind: unit
        ref: "plugins/whatsapp/link_test.go#TestLinkJSON_PairingAcceptedEmitsProgressEvent"
        status: pass
      - kind: unit
        ref: "plugins/whatsapp/link_test.go#TestLinkJSON_AlreadyLinkedEmitsProgressEventWithoutDeviceID"
        status: pass
      - kind: unit
        ref: "plugins/whatsapp/link_test.go#TestLinkEvent_MarshalsToOneLine"
        status: pass
    human_judgment: false
  - id: D2
    description: "The kernel captures the link subprocess's stderr into its own logger (stderrLineLogger, cmd.Stderr) instead of discarding it — including a trailing line with no newline"
    requirement: SRC-03
    verification:
      - kind: unit
        ref: "kernel/httpapi/whatsapplink_exec_test.go#TestExecLinkSpawner_CapturesStderrIntoLogger"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/whatsapplink_exec_test.go#TestExecLinkSpawner_FlushesTrailingPartialStderrLine"
        status: pass
    human_judgment: false
  - id: D3
    description: "The production spawner (newExecLinkSpawner) has its first automated coverage: real subprocess streaming, argv construction, environment inheritance, and kill — not just the pre-existing fakeSpawner harness"
    requirement: SRC-03
    verification:
      - kind: unit
        ref: "kernel/httpapi/whatsapplink_exec_test.go#TestExecLinkSpawner_StreamsLinesInOrderAndExitsClean"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/whatsapplink_exec_test.go#TestExecLinkSpawner_PassesLinkModeFlagAndDataPath"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/whatsapplink_exec_test.go#TestExecLinkSpawner_InheritsEnvironment"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/whatsapplink_exec_test.go#TestExecLinkSpawner_KillTerminatesLongRunningSubprocess"
        status: pass
    human_judgment: false
  - id: D4
    description: "Both progress states are non-terminal end to end through the kernel session: a poll stays 200 (not 404), carries no diagnostic fields, and does not resume a suspended instance; the pre-existing exactly-once terminal contract behind paired/error/timeout is unchanged"
    requirement: SRC-03
    verification:
      - kind: unit
        ref: "kernel/httpapi/whatsapplink_test.go#TestWhatsAppLink_ProgressStatesAreNonTerminal"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/whatsapplink_test.go#TestIsTerminalKind_ProgressKindsAreNonTerminal"
        status: pass
    human_judgment: false
  - id: D5
    description: "make test-portable passes end to end across every workspace module, plugins/whatsapp's read-only and no-egress AST scans remain green, and no go.mod/go.sum changed"
    verification:
      - kind: other
        ref: "make test-portable (all modules ok); plugins/whatsapp TestReadOnly_NoSendCapableClientSelector and TestOutboundHosts_NoSelfConstructedHTTPClientOrUnlistedHostLiteral pass; git diff --stat over go.mod/go.sum is empty"
        status: pass
    human_judgment: false

# Metrics
duration: ~35min
completed: 2026-08-10
status: complete
---

# Phase 08 Plan 06: WhatsApp Link Post-Pair Progress Events and Kernel Stderr Capture Summary

**The plugin now announces `pairing_accepted`/`already_linked` on the wire (device-id-free), the kernel captures the link subprocess's stderr through a new `newExecLinkSpawner(logger)` instead of discarding it to `/dev/null`, and the production spawner has its first automated test coverage — closing the producer half of G-08-1's AND-gate.**

## Performance

- **Duration:** ~35 min
- **Started:** 2026-08-10
- **Completed:** 2026-08-10
- **Tasks:** 3
- **Files modified:** 6 (1 created, 5 modified)

## Accomplishments

- `plugins/whatsapp/link.go` gained two new non-terminal `linkEventKind` values (`pairing_accepted`, `already_linked`) with constructor helpers; `jsonLinkEmitter.pairingAccepted()`/`.alreadyLinked()` now write the corresponding single-key event to stdout in addition to their existing stderr diagnostic — the device id stays stderr-only by design (it embeds the user's own phone number)
- `kernel/httpapi/whatsapplink.go`'s `execLinkSpawner` free function became `newExecLinkSpawner(logger hclog.Logger) linkSpawner`, a constructor whose closure assigns a new line-oriented `stderrLineLogger` to `cmd.Stderr` — every line the subprocess writes to stderr now reaches a named `link-subprocess` hclog sublogger instead of `/dev/null`, with a 64 KiB retention cap (T-08-17) and a trailing-partial-line flush after `cmd.Wait()` returns
- `cmd.Env = nil`'s doc comment corrected: it means the child inherits this process's environment (the true, documented os/exec behavior), not the opposite the prior comment claimed — referencing 08-UAT.md's G-08-1 as where that false claim was found
- `kernel/httpapi/whatsapplink_exec_test.go` (new file) drives the real spawner against executable `/bin/sh` fixtures: streaming, argv, stderr capture, trailing-partial flush, environment inheritance, and kill — the first automated coverage of this process boundary, which the pre-existing `fakeSpawner` harness structurally cannot provide
- `kernel/httpapi/whatsapplink_test.go` gained a table-driven test proving both progress kinds are non-terminal end to end (200 not 404, no diagnostic fields, no premature resume) and a guard locking `isTerminalKind`'s allowlist against ever including a progress kind

## Task Commits

Each task was committed atomically (TDD RED/GREEN pairs):

1. **Task 1: The plugin announces post-pair progress on the wire** - `bcd157c` (test, RED) + `1c254a5` (feat, GREEN)
2. **Task 2: The kernel captures the link subprocess's stderr, and the production spawner gets its first test** - `3696d96` (test, RED) + `eb0c6e3` (feat, GREEN)
3. **Task 3: Progress states are non-terminal end to end through the kernel session** - `fdae11e` (test only — no production code change, per the plan's own prediction that `isTerminalKind`'s closed allowlist already made this hold)

_All three tasks are `tdd="true"`. Task 3 has no GREEN commit because none was needed: the plan explicitly states the expected outcome is these assertions passing against the already-shipped kernel, and they did on first run._

## Files Created/Modified

- `plugins/whatsapp/link.go` - Two new non-terminal `linkEventKind` values and constructors; `jsonLinkEmitter.pairingAccepted()`/`.alreadyLinked()` now emit to stdout in addition to stderr
- `plugins/whatsapp/link_test.go` - New guards for the progress events' single-key shape and the device-id stdout/stderr split
- `kernel/httpapi/whatsapplink.go` - `execLinkSpawner` → `newExecLinkSpawner(logger)`; new `stderrLineLogger` type capturing subprocess stderr; corrected `cmd.Env` comment
- `kernel/httpapi/whatsapplink_exec_test.go` - New file: first automated coverage of the real spawner (streaming, argv, stderr, env, kill)
- `kernel/httpapi/whatsapplink_test.go` - New table-driven progress-state test and `isTerminalKind` allowlist guard
- `kernel/httpapi/routes.go` - Single call site updated to `newExecLinkSpawner(logger)`

## Decisions Made

- `stderrLineLogger`'s retention cap fixed at 64 KiB, matching the existing stdout scanner's own starting buffer size for consistency across both captured streams — the plan specified the behavior (bound the buffer) but not a literal value
- Two Task-2 test names were fixed at authoring time so they actually match the plan's own literal `-run 'TestLink'` verify command (renamed from an initial `TestJSONLinkEmitter_*` draft, which does not contain the substring `TestLink`, to `TestLinkJSON_*`) — caught before commit, not a runtime deviation
- Task 3 produced test-only commits with no accompanying feat commit, exactly matching the plan's own stated expectation that these assertions would pass unchanged against the already-shipped kernel

## Deviations from Plan

None - plan executed exactly as written. Task 3's "no production code change" is the plan's own explicitly predicted outcome, not a deviation from it.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The producer half of G-08-1's AND-gate is closed: the plugin now emits `pairing_accepted`/`already_linked` on the wire, and plan 08-05's consumer-side tolerance (QRPanel.svelte, already shipped) means the panel renders the new Pairing progress phase the moment these events arrive.
- The kernel no longer discards the link subprocess's stderr — the next failure in this path will leave a captured diagnostic trail via the kernel's own logger, closing the observability gap that made the original failing UAT run read `errors: None`.
- The kernel↔plugin process boundary (the real `os/exec` spawner) has automated coverage for the first time.
- Plan 08-07 owns the genuine end-to-end Playwright proof (`qr -> poll -> pairing_accepted -> paired`) against this now-complete producer+consumer pair, plus the Add-Source declined-link notice implementation whose copy was locked in plan 08-05's `08-UI-SPEC.md` Amendment 2.
- No blockers.

---
*Phase: 08-whatsapp-conversations-managed-risk*
*Completed: 2026-08-10*
