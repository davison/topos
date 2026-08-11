---
phase: 08-whatsapp-conversations-managed-risk
plan: 14
subsystem: whatsapp-plugin
tags: [whatsmeow, go-plugin, goroutine, event-handler-lifecycle, kernel-startup-latency]

# Dependency graph
requires:
  - phase: 08-whatsapp-conversations-managed-risk
    provides: "plan 08-11's G-08-4 fix (healthStateConnecting zero value, pairLoginWaiter primitive) and plan 08-12's kernel-side bounded first-refresh retry (kernel/syncer/scheduler.go), both of which this plan leans on instead of the synchronous wait it retires"
provides:
  - "startBackgroundClient returns immediately after a successful dial — no plugin in the repo blocks its go-plugin handshake on a live network event"
  - "the serve-mode login waiter's event handler is retired on both dial outcomes (success via a background goroutine's deferred call, failure via an immediate call on the Connect() error branch)"
  - "corrected doc comments in kernel/syncer/scheduler.go and 08-UI-SPEC.md that no longer assert the retired 'handshake absorbs the login round trip' claim"
affects: [08-13, 08-15, kernel-supervisor-launch-latency, whatsapp-plugin-launch-path]

# Actuals (#2632)
actuals:
  tokens: 4973
  tasks: 2
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Dispatch a bounded background wait onto its own goroutine with deferred cleanup, rather than blocking a process-startup path that a supervisor/handshake is waiting on"
    - "AST structural guard updated in place (renamed, not duplicated) when its own house-pattern proof needs a materially different assertion shape"

key-files:
  created: []
  modified:
    - plugins/whatsapp/connect.go
    - plugins/whatsapp/connect_test.go
    - kernel/syncer/scheduler.go
    - .planning/phases/08-whatsapp-conversations-managed-risk/08-UI-SPEC.md

key-decisions:
  - "Superseded 08-11-PLAN.md's must_haves.truths[2] ('the go-plugin handshake genuinely means ready') by design, per this plan's own objective — recorded in a doc comment at the point of change rather than left implicit."
  - "Ran the plan's -race verification with CGO_ENABLED=1 rather than CGO_ENABLED=0 as literally written in <verify>, because Go's -race flag itself requires cgo to compile the race-instrumented binary — a tooling constraint independent of whether the plugin's own dependency set uses cgo (it does not; go.mod/go.sum are byte-identical to HEAD)."

patterns-established:
  - "Login-wait-off-launch-path: a bounded network wait that used to gate a subprocess handshake is now dispatched on a goroutine with deferred handler cleanup, so process startup cost matches every other plugin's."

requirements-completed: [SRC-03]

coverage:
  - id: D1
    description: "startBackgroundClient's serve-mode login wait no longer blocks before goplugin.Serve; it runs on a background goroutine dispatched after a successful dial"
    requirement: SRC-03
    verification:
      - kind: unit
        ref: "plugins/whatsapp/connect_test.go#TestStartBackgroundClient_ConnectingBeforeDialAndLoginWaitOffTheLaunchPath"
        status: pass
    human_judgment: false
  - id: D2
    description: "The login waiter's event handler is removed from the client on both dial outcomes (success and failure), with no accumulated handler on repeated launch/suspend/relaunch"
    requirement: SRC-03
    verification:
      - kind: unit
        ref: "plugins/whatsapp/connect_test.go#TestStartBackgroundClient_ConnectingBeforeDialAndLoginWaitOffTheLaunchPath (assertion d)"
        status: pass
    human_judgment: false
  - id: D3
    description: "kernel/syncer/scheduler.go's defaultFirstRefreshRetryDelays doc comment and 08-UI-SPEC.md's connecting-row note no longer assert that the plugin's own launch absorbs the login round trip"
    verification:
      - kind: unit
        ref: "go test ./kernel/syncer/... -run TestScheduler -v -count=1 (all six tests, comment-only diff confirmed via git diff -U0 grep)"
        status: pass
    human_judgment: false

duration: ~4min (commit-span lower bound; excludes reading/planning time)
completed: 2026-08-11
status: complete
---

# Phase 08 Plan 14: Serve-mode login wait off the launch path Summary

**Moved the WhatsApp plugin's bounded post-dial login wait off `startBackgroundClient`'s synchronous return path onto a background goroutine with deferred handler cleanup, so an already-linked instance's launch costs what every other plugin's launch costs — closing 08-REVIEW.md's WR-01/WR-02 and 08-VERIFICATION.md's G-08-5 — and corrected the two doc comments (`kernel/syncer/scheduler.go`, `08-UI-SPEC.md`) that asserted the now-retired "handshake means ready" behaviour.**

## Performance

- **Duration:** ~4 min between first and last commit (lower bound; excludes file-reading and design time)
- **Started:** 2026-08-11T09:55:49+01:00 (first commit)
- **Completed:** 2026-08-11T09:59:22+01:00 (last commit)
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- `startBackgroundClient` now returns immediately after a successful `client.Connect()` dial — no plugin in this repo blocks its go-plugin handshake on a live network event, closing WR-01/G-08-5.
- The login waiter's `AddEventHandler` return value is captured (`loginWaiterID`) and `RemoveEventHandler` is called on both dial outcomes — immediately on the `Connect()` error branch, and via a deferred call inside the background goroutine on the success branch — closing WR-02.
- The AST structural guard in `connect_test.go` was renamed and rewritten to pin the new shape: the wait must be dispatched inside a `go` statement's function literal (not awaited synchronously), and `RemoveEventHandler` must appear both inside and outside that literal.
- `kernel/syncer/scheduler.go`'s `defaultFirstRefreshRetryDelays` doc comment and `08-UI-SPEC.md`'s connecting-row note were corrected (comment/prose only, no behavior change) to state that the kernel-side bounded first-refresh retry, not the plugin's own launch, now covers the connecting window.
- Two negative controls were run by hand (see below) proving the AST guard is not vacuous.

## Task Commits

Each task was committed atomically, with Task 1 following the TDD RED→GREEN cycle (`tdd="true"`):

1. **Task 1 (RED): pin the new AST guard shape** - `09edf08` (test) — confirmed FAILING against the unmodified `connect.go` (the synchronous wait was still on the launch path).
2. **Task 1 (GREEN): restructure `startBackgroundClient`** - `09e0e71` (feat) — dispatches the wait onto a goroutine, adds `RemoveEventHandler` on both dial outcomes, rewrites `serveLoginTimeout`'s doc comment. Confirmed the AST guard now passes.
3. **Task 2: correct the two falsified doc comments** - `9b70e22` (docs) — `kernel/syncer/scheduler.go` and `08-UI-SPEC.md`, comment/prose only.

_Tracer feedback gate: Task 1 is `type="tracer"`. After committing `09e0e71`, its own `<verify>` (`go build && go vet && go test -race`) was re-run end-to-end and passed before Task 2 began._

## Files Created/Modified
- `plugins/whatsapp/connect.go` - `startBackgroundClient` restructured: login wait moved to a background goroutine with deferred `RemoveEventHandler`; failure branch also calls `RemoveEventHandler`; `serveLoginTimeout`'s doc comment rewritten for its new role (bounds handler lifetime, not launch time); a new comment records the WR-01/G-08-5 trade at the point of change.
- `plugins/whatsapp/connect_test.go` - `TestStartBackgroundClient_SuccessPathSetsConnectingAndWaitsForLogin` renamed to `TestStartBackgroundClient_ConnectingBeforeDialAndLoginWaitOffTheLaunchPath`; assertions (a)/(b) unchanged (connecting-before-dial, waiter-registered-before-dial); assertions (c)/(d) are new (wait must be inside a `go` statement's function literal; `RemoveEventHandler` must appear both inside and outside that literal).
- `kernel/syncer/scheduler.go` - `defaultFirstRefreshRetryDelays`' doc comment corrected; no code, numeric value, or test changed.
- `.planning/phases/08-whatsapp-conversations-managed-risk/08-UI-SPEC.md` - connecting-row note amended additively with a new dated (2026-08-11, gap G-08-5, Plan 08-14) correction line; the original 2026-08-11/Plan 08-11 line is preserved, not rewritten; the taxonomy table itself is untouched.

## Decisions Made
- **Superseded 08-11-PLAN.md's `must_haves.truths[2]`** ("the go-plugin handshake genuinely means ready in the normal case") by design — this was this plan's stated objective, not a discovered deviation. Recorded explicitly in a new comment in `connect.go` at the exact point the trade is made, and in the corrected doc comments in `scheduler.go` and `08-UI-SPEC.md`, so a future reader finds the contradiction resolved rather than latent.
- **Ran `-race` with `CGO_ENABLED=1`** rather than the literal `CGO_ENABLED=0 ... -race` in the plan's `<verify>` block, because Go's race detector itself requires cgo to build the instrumented test binary — this is independent of whether the plugin's own dependency set uses cgo (confirmed unchanged: `git diff go.mod go.sum` is empty across all three commits). All other verify commands (`go build`, `go vet`, plain `go test`, `make test-portable`) ran exactly as written with `CGO_ENABLED=0`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `-race` requires `CGO_ENABLED=1`, not `=0` as the plan's `<verify>` block literally states**
- **Found during:** Task 1 verification
- **Issue:** The plan's `<verify>` command is `CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go vet ./... && CGO_ENABLED=0 go test ./... -count=1 -race`. Running the `-race` leg with `CGO_ENABLED=0` fails immediately with `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1` — a Go toolchain constraint, not a project or plan error.
- **Fix:** Ran `go build`/`go vet`/plain `go test` with `CGO_ENABLED=0` exactly as written, and ran only the `-race` leg with `CGO_ENABLED=1`. Confirmed this changes nothing about the plugin's own dependency footprint: `git diff go.mod go.sum` across all three commits is empty, satisfying the plan's own "kernel binary stays cgo-free... whatsapp plugin module's dependency set is unchanged" prohibition.
- **Files modified:** none (verification-only adjustment)
- **Verification:** `CGO_ENABLED=1 go test ./... -count=1 -race` passes, all tests green including the new/renamed AST guard.
- **Committed in:** n/a (no code change — a verification-command correction, documented here per Rule 3)

---

**Total deviations:** 1 auto-fixed (1 blocking — toolchain constraint, not a code or plan defect)
**Impact on plan:** No scope creep; no file outside `files_modified` touched; `git diff --stat HEAD~3 HEAD` lists exactly the four files the plan named.

## Issues Encountered
None beyond the `-race`/`CGO_ENABLED` toolchain note above.

## Negative Controls (required by plan's `<acceptance_criteria>`)

Both run by hand against the final restructured `connect.go`, then reverted (confirmed via `diff` against a saved backup that the restore matched the intended committed state exactly):

1. **Moved `loginWaiter.wait(serveLoginTimeout)` back onto the synchronous path** (out of the `go` statement, with `RemoveEventHandler` following it inline): re-ran `go test ./... -run TestStartBackgroundClient -v -count=1`. Result: **FAILED** as required —
   ```
   connect_test.go:221: WR-01/G-08-5: wait(serveLoginTimeout) at connect.go:197:12 must be
   dispatched inside a `go` statement's function literal, not awaited synchronously — a
   blocking wait here delays the go-plugin handshake, and through it kernel boot and every
   relaunch
   ```
   Restored; re-ran the guard; passed; `diff` against the pre-mutation backup showed no residual difference.

2. **Deleted the `RemoveEventHandler` call on the `client.Connect()` error branch** (leaving only the one inside the background goroutine): re-ran the guard. Result: **FAILED** as required —
   ```
   connect_test.go:234: WR-02: no RemoveEventHandler call found outside the background `go`
   statement — the client.Connect() error branch must retire the waiter's handler
   immediately, since there is no wait to serve on that branch
   ```
   Restored; re-ran the guard; passed; `diff` against the pre-mutation backup showed no residual difference.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- WR-01 and WR-02 (08-REVIEW.md) are closed; G-08-5 (08-VERIFICATION.md) is closed on the plugin side.
- The two legs that keep G-08-4 closed without the retired synchronous wait — `healthStateConnecting`'s zero value plus its explicit pre-dial assignment, and `kernel/syncer/scheduler.go`'s bounded first-refresh retry — are both provably untouched (unchanged code paths; comment-only diff on the scheduler side, confirmed via `git diff -U0 | grep -vc '^[-+][[:space:]]*//'` returning 0).
- Plan 08-15 (this wave's gate, per the plan's own Playwright scope note) runs next; this plan touches no frontend file and adds no new route or response shape, so the existing e2e suite is expected to pass unchanged.
- Plan 08-13's `kernel/supervisor/launchlatency_test.go` is the behavioural, real-subprocess proof for the cross-source consequence this plan's structural guard cannot itself prove (a slow-launching plugin delaying every other source's launch) — not re-run here, out of this plan's own scope, but named in both the test's doc comment and this summary for traceability.

---
*Phase: 08-whatsapp-conversations-managed-risk*
*Completed: 2026-08-11*
