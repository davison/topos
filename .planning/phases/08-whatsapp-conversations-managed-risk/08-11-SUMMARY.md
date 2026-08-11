---
phase: 08-whatsapp-conversations-managed-risk
plan: 11
subsystem: api
tags: [whatsmeow, health-state, go-plugin, gap-closure, ast-guard]

requires:
  - phase: 08-whatsapp-conversations-managed-risk
    provides: healthState taxonomy (health.go), pairLoginWaiter primitive (pairwait.go), startBackgroundClient (connect.go) — all from earlier 08-0x plans
provides:
  - "healthStateConnecting: a sixth named health state, declared first in health.go's iota block, replacing healthStateNotLinked as the plugin's Go zero value"
  - "startBackgroundClient's already-paired success path explicitly assigns the connecting state before dialing and blocks on a bounded pairLoginWaiter.wait(serveLoginTimeout) after a successful dial"
  - "regression armor pinning the exact G-08-4 message-identity defect (a zero-value plugin's Match error) so it cannot silently reappear"
affects: [08-12]

actuals:
  tokens: 4400
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Zero-value-as-honest-default: a Go enum's zero value is deliberately chosen to be the least-wrong state (connecting), not repurposed as an implicit 'never assigned' sentinel that happens to alias a real, actionable state"
    - "AST structural guard for call-ordering (readonly_test.go's house pattern) reused to pin a THREE-call ordering constraint (setHealthState -> AddEventHandler -> Connect -> wait) that no behavioral test can exercise without a live WhatsApp server"

key-files:
  created: []
  modified:
    - plugins/whatsapp/health.go
    - plugins/whatsapp/health_test.go
    - plugins/whatsapp/connect.go
    - plugins/whatsapp/connect_test.go
    - .planning/phases/08-whatsapp-conversations-managed-risk/08-UI-SPEC.md

key-decisions:
  - "healthStateConnecting declared FIRST in the iota block (not appended last) so it becomes healthState's Go zero value, per the plan's own must_haves — healthStateNotLinked's doc comment updated to drop its now-false zero-value claim while keeping both of its real meanings"
  - "The connecting-state message offers no recovery action (unlike the other five states' Re-link/-link-flag pointers) — it is transient and self-clearing, so 08-UI-SPEC.md's amendment records that explicitly"
  - "pairLoginWaiter registered AFTER p.handleEvent (both on the same client, before Connect()) so whatsmeow's synchronous in-order event dispatch guarantees p.handleEvent's healthStateLinked assignment lands before the waiter's wait() call returns — no post-wait state assignment was added, to avoid clobbering a same-instant LoggedOut/StreamReplaced"
  - "The wait outcome is never fatal: every path still returns nil so goplugin.Serve is always reached; a non-nil wait error is only logged (one line, pluginName-prefixed, no chat/sender/body content)"

requirements-completed: [SRC-03]

coverage:
  - id: D1
    description: "A zero-value *SourcePlugin (never explicitly assigned) reports the connecting cause, not the not-linked cause, so the go-plugin handshake's earliest window is never mistaken for an unpaired device"
    requirement: SRC-03
    verification:
      - kind: unit
        ref: "plugins/whatsapp/health_test.go#TestConnectingState_IsTheZeroValue"
        status: pass
      - kind: unit
        ref: "plugins/whatsapp/health_test.go#TestConnectingState_MatchMessageIsNotThePairingInstruction"
        status: pass
    human_judgment: false
  - id: D2
    description: "The connecting state's Match error still returns codes.Unavailable (never an empty success), and all five pre-existing non-healthy states still pass their per-state Match/Health regressions with the sixth state included"
    requirement: SRC-03
    verification:
      - kind: unit
        ref: "plugins/whatsapp/delink_test.go#TestDelink_MatchReturnsUnavailableForEveryNonHealthyState"
        status: pass
      - kind: unit
        ref: "plugins/whatsapp/health_test.go#TestHealth_ReachableFalseWithLastErrorPerState"
        status: pass
    human_judgment: false
  - id: D3
    description: "startBackgroundClient's success path assigns healthStateConnecting before dialing, registers the pairLoginWaiter before Connect(), and waits on it (bounded by serveLoginTimeout) after Connect() — pinned by ordering, since no live WhatsApp server is available in this repo's test environment"
    requirement: SRC-03
    verification:
      - kind: unit
        ref: "plugins/whatsapp/connect_test.go#TestStartBackgroundClient_SuccessPathSetsConnectingAndWaitsForLogin"
        status: pass
    human_judgment: false
  - id: D4
    description: "The whole plugins/whatsapp module and the entire workspace build and test green after both changes, including the pre-existing read-only and outbound-hosts AST scans"
    verification:
      - kind: unit
        ref: "make test-portable (full workspace, all 13 modules)"
        status: pass
    human_judgment: false

duration: ~20min
completed: 2026-08-11
status: complete
---

# Phase 08 Plan 11: WhatsApp Connecting-State Zero-Value Fix (G-08-4 Plugin Half) Summary

**Added a sixth `healthStateConnecting` health state as the plugin's honest Go zero value, and made `startBackgroundClient`'s already-paired success path explicitly assign it and block on a bounded login wait before the go-plugin handshake completes — closing gap G-08-4's plugin-side defect where a freshly relaunched, already-paired WhatsApp instance reported the false "Not linked — pair this device" message.**

## Performance

- **Duration:** ~20 min
- **Tasks:** 2
- **Files modified:** 5

## Accomplishments
- `health.go` gained a sixth named state, `healthStateConnecting`, declared first in the iota block so it is the plugin's Go zero value — `healthStateNotLinked` demoted to second position and no longer doubles as "uninitialised"
- `connect.go`'s already-paired success path now explicitly sets the connecting state before dialing, registers a `pairLoginWaiter` (reusing the exact primitive the link flow already proves against a real device) before `client.Connect()`, and blocks on a bounded 15-second wait for a real `*events.Connected` after a successful dial — always falling through to `return nil` so `goplugin.Serve` is never delayed unboundedly
- Regression armor pins the exact byte-level message identity the user reported: a zero-value plugin's `Match` error now carries the connecting template, never the not-linked pairing instruction
- An AST structural guard (mirroring `readonly_test.go`'s house pattern) proves the call ordering inside `startBackgroundClient` that no behavioral test can exercise without a live WhatsApp server
- `08-UI-SPEC.md`'s "New Health-State Taxonomy" table amended with the sixth row, recorded as gap-closure G-08-4, transient and self-clearing (no recovery action, unlike the other five states)

## Task Commits

Each task was committed atomically:

1. **Task 1: A named connecting state that is also the zero value, and the message-identity regression it exists to prevent** - `72ed516` (feat)
2. **Task 2: Serve mode waits for a real login before the handshake says "ready"** - `a539156` (feat)

_Note: worktree mode — this SUMMARY and STATE/ROADMAP updates are the orchestrator's responsibility after merge; no plan-metadata commit was made from this worktree._

## Files Created/Modified
- `plugins/whatsapp/health.go` - `healthStateConnecting` added first in the iota block (the plugin's zero value); `healthStateNotLinked`'s doc comment corrected; new `healthMessages` template with no recovery action
- `plugins/whatsapp/health_test.go` - `healthStateConnecting` added to `nonHealthyStates` (extends every existing per-state regression); `TestConnectingState_IsTheZeroValue` and `TestConnectingState_MatchMessageIsNotThePairingInstruction` added
- `plugins/whatsapp/connect.go` - `serveLoginTimeout` constant added; `startBackgroundClient`'s already-paired branch restructured to assign the connecting state, register the login waiter, and wait after `Connect()`
- `plugins/whatsapp/connect_test.go` - `TestStartBackgroundClient_SuccessPathSetsConnectingAndWaitsForLogin` AST ordering guard added
- `.planning/phases/08-whatsapp-conversations-managed-risk/08-UI-SPEC.md` - taxonomy table amended with the connecting row and a dated gap-closure note

## Decisions Made
- `healthStateConnecting` declared FIRST in the iota block (not appended as a seventh value at the end) so it becomes `healthState`'s Go zero value, per the plan's own must_haves — this is the direct fix for G-08-4's root cause 1
- The connecting-state message deliberately offers no recovery action (unlike D-03/D-04's Re-link/-link-flag pointers on the other five non-healthy templates) — it is transient and self-clearing; 08-UI-SPEC.md's amendment records this explicitly
- `pairLoginWaiter` registered AFTER `p.handleEvent` on the same client, both before `client.Connect()` — whatsmeow dispatches events to handlers synchronously in registration order, so by the time the waiter signals, `p.handleEvent`'s own `*events.Connected` case has already assigned `healthStateLinked`. No post-wait state assignment was added to avoid clobbering a same-instant `LoggedOut`/`StreamReplaced`
- The wait's outcome never affects control flow — every path returns `nil` so `goplugin.Serve` is always reached; a non-nil wait error is logged as one line (no chat name, sender name, message body, or key material) under this package's fixed `pluginName` prefix

## Deviations from Plan

None - plan executed exactly as written. Both tasks' `<action>` sections were followed precisely, including the exact `healthMessages` string for the connecting state and the AST guard's three-call ordering assertions.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Plugin-side half of G-08-4 is closed: the exact reported message can no longer be produced by a paired, connecting instance, pinned by automated tests rather than inspection alone
- 08-02's Match-error-never-empty-success invariant and the pre-existing read-only/outbound-hosts AST scans all still pass (verified via full `make test-portable`)
- Plan 08-12 (hermetic regression gate for this failure CLASS, per this plan's own scope note) is unblocked — this plan touched no frontend file and added no new visual state, consistent with 08-UI-SPEC.md's standing "No new visual states" rule

---
*Phase: 08-whatsapp-conversations-managed-risk*
*Completed: 2026-08-11*
