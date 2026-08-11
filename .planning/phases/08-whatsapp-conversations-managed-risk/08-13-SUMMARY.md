---
phase: 08-whatsapp-conversations-managed-risk
plan: 13
subsystem: infra
tags: [go, sync.RWMutex, go-plugin, hashicorp-go-plugin, concurrency, data-race, kernel]

# Dependency graph
requires:
  - phase: 08-whatsapp-conversations-managed-risk
    provides: "plugins/mock's opt-in WEBSPACES_MOCK_READY_AFTER_MS launch-readiness fixture and kernel/supervisor/readiness_test.go's t.Setenv-then-boot pattern (08-VERIFICATION.md G-08-4 closure), reused as the shape this plan's own launch-delay fixture and gate follow"
provides:
  - "Supervisor.genMu: a reader-only lock Host()/Coordinator() take instead of s.mu, so the reader path never waits behind a mutation's plugin-subprocess launch"
  - "pluginhost.Host.mu: an internally-synchronised Host safe for concurrent readers against a single in-flight Reconcile, closing a latent pre-existing data race"
  - "plugins/mock's WEBSPACES_MOCK_LAUNCH_DELAY_MS fixture: a controllable slow-handshake plugin launch for hermetic tests"
  - "kernel/supervisor/launchlatency_test.go's TestResume_SlowRelaunchDoesNotFreezeOtherSources: the first hermetic gate for phase success criterion 4's cross-source isolation clause"
affects: [kernel-supervisor, kernel-pluginhost, plugins-mock, docs-testing]

# Actuals (#2632)
actuals:
  tokens: 6965
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Split reader/writer locking: a long-held mutation lock (Supervisor.mu) and a short-held generation-pointer lock (Supervisor.genMu) coexist so a slow operation held under the mutation lock never blocks a fast reader that only needs the current generation's pointer"
    - "Snapshot-then-launch-then-commit locking inside Host.Reconcile: RLock only to snapshot the existing set, release it, perform every subprocess launch with no lock held, then take the write lock only for the kill-and-commit region"

key-files:
  created:
    - kernel/supervisor/launchlatency_test.go
  modified:
    - kernel/supervisor/supervisor.go
    - kernel/pluginhost/host.go
    - plugins/mock/readiness.go
    - plugins/mock/main.go
    - plugins/mock/plugin_test.go
    - kernel/supervisor/readiness_test.go
    - docs/testing.md

key-decisions:
  - "Two separate locks, not one relaxed lock: Supervisor.mu keeps its exact prior scope and duration (still held across Apply/SuspendInstance/resume/Shutdown's whole Reconcile call) — only Host()/Coordinator() move to genMu, so the mutation-serialization guarantee ('never two applies in flight together') is completely unchanged"
  - "Host.Reconcile takes its write lock only for the kill-and-commit region, never across the launch loop — holding it across launches was the exact defect being fixed"
  - "The launch-delay fixture is a sibling of the existing readiness-window fixture, not a replacement: readyAfterEnvVar delays Match/Health AFTER the handshake; launchDelayEnvVar delays the handshake itself, modeling a different failure class (a plugin slow to come up at all, e.g. plugins/whatsapp's serve-mode login wait)"
  - "4000ms delay against a 2s pass/fail threshold in the new gate: with the fix the reader path is sub-millisecond (~2000x margin); without the fix it cannot return before the 4s launch completes (2x margin the other way) — the threshold sits between two orders of magnitude, not on a knife edge"

patterns-established:
  - "A background lifecycle mutation that must run synchronously on a request path (SuspendInstance's resume closure, on the WhatsApp link poll/cancel HTTP routes) must never be the same lock every unrelated reader also takes — split the reader path onto its own lock before that mutation's own slow leg (a real subprocess launch) can grow unboundedly"

requirements-completed: [SRC-03]

coverage:
  - id: D1
    description: "A resume closure relaunching a slow-to-handshake plugin subprocess no longer blocks any other source's health probe or manual refresh (phase success criterion 4)"
    requirement: SRC-03
    verification:
      - kind: unit
        ref: "kernel/supervisor/launchlatency_test.go#TestResume_SlowRelaunchDoesNotFreezeOtherSources"
        status: pass
    human_judgment: false
  - id: D2
    description: "The latent data race between Supervisor.Fetch's reader path and a concurrent Host.Reconcile is closed and proven closed under -race"
    verification:
      - kind: unit
        ref: "go test ./kernel/pluginhost/... ./kernel/supervisor/... -race"
        status: pass
    human_judgment: false
  - id: D3
    description: "The mock plugin's new launch-delay fixture (WEBSPACES_MOCK_LAUNCH_DELAY_MS) has its own unit coverage and is off by default"
    verification:
      - kind: unit
        ref: "plugins/mock/plugin_test.go#TestLaunchDelayFromEnv"
        status: pass
    human_judgment: false
  - id: D4
    description: "08-REVIEW.md's IN-01 comment-drift finding is closed (the readiness_test.go polling-loop comment now references the authoritative retry-delay constant instead of restating a number that can silently drift)"
    verification:
      - kind: unit
        ref: "grep -c defaultFirstRefreshRetryDelays kernel/supervisor/readiness_test.go"
        status: pass
    human_judgment: false

duration: 55min
completed: 2026-08-11
status: complete
---

# Phase 08 Plan 13: Supervisor reader/writer lock split + Host internal synchronization Summary

**Split `Supervisor.Host()`/`Coordinator()` onto a reader-only `genMu` (never `s.mu`) and made `pluginhost.Host` internally thread-safe (`Host.mu` + snapshot-returning reads), closing 08-VERIFICATION.md gap G-08-5 — a slow plugin relaunch no longer freezes every other source's item-fetch, health-probe and manual-refresh routes kernel-wide.**

## Performance

- **Duration:** 55 min
- **Started:** 2026-08-11T08:08Z (approx)
- **Completed:** 2026-08-11T09:03Z
- **Tasks:** 2
- **Files modified:** 7 modified, 1 created

## Accomplishments
- Closed the structural freeze: `Supervisor.SuspendInstance`'s resume closure still holds `s.mu` for its whole duration (including a real `Host.Reconcile` subprocess launch), but `Host()`/`Coordinator()` — and therefore `Fetch`/`ProbeSources`/`Refresh`/`RefreshAll`, which back every source's HTTP routes — now resolve through a separate reader-only `genMu` and never wait on that mutation.
- Closed a latent, pre-existing data race: `Supervisor.Fetch` already released `s.mu` before calling `Host.Fetch` concurrently with a mutating `Reconcile`. `pluginhost.Host` now guards its `plugins` field with its own `sync.RWMutex`, and every read path (`Plugins`, `ProbeSources`, `Fetch`/`byInstance`) reads through a new `snapshot()` helper that returns a defensive copy.
- `Host.Reconcile` now takes its write lock only for the kill-and-commit region — the launch loop for new/changed instances runs with no lock held at all, which is what makes the reader path's independence real rather than theoretical.
- Added the first hermetic, real-subprocess gate for phase success criterion 4's cross-source isolation clause: `TestResume_SlowRelaunchDoesNotFreezeOtherSources`, driving a genuine 4-second-slow mock plugin relaunch and asserting an unrelated source's health probe and manual refresh both return well under 2 seconds.
- Ran the plan's mandated negative control by hand: reverting `Host()`/`Coordinator()` to take `s.mu` made the new test fail (`ProbeSources` took ~3.71s against the 2s threshold); restoring the fix made it pass again (confirmed via `git diff --stat`).
- Closed 08-REVIEW.md's IN-01 comment-drift finding in `readiness_test.go` (comment-only change, no assertion/deadline/`t.Setenv` value touched).
- Documented the new fixture in `docs/testing.md` and added a dated `What changed` entry.

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end — a slow plugin relaunch no longer freezes every other source** - `ab732f6` (feat)
2. **Task 2: The fixture's own unit coverage, its documentation, and IN-01's comment drift** - `2309a30` (test)

_No TDD sub-commits: Task 1 is `type="tracer" tdd="true"` but its single behavior/implementation pair landed as one atomic commit per this plan's own scope (a locking refactor plus one new gate test, not a RED/GREEN split feature)._

## Files Created/Modified
- `kernel/supervisor/supervisor.go` - Added `genMu sync.RWMutex`; `Host()`/`Coordinator()` take `genMu.RLock()` only; every write to `s.host`/`s.coord` (`NewSupervisor`'s two assignments, `commitGeneration`'s coordinator assignment) takes `genMu.Lock()` for the assignment alone
- `kernel/pluginhost/host.go` - Added `Host.mu sync.RWMutex` guarding `plugins`; new `snapshot()` helper; `Reconcile` snapshots under `RLock`, launches with no lock held, commits under `Lock`; `Plugins()`/`ProbeSources()`/`byInstance()` route through `snapshot()`; `Shutdown()` takes `Lock()`
- `plugins/mock/readiness.go` - New `launchDelayEnvVar` const (`WEBSPACES_MOCK_LAUNCH_DELAY_MS`) and `launchDelayFromEnv` parse-or-fail-loud helper, sibling to the existing readiness-window fixture
- `plugins/mock/main.go` - Calls `launchDelayFromEnv(os.Getenv)` before `goplugin.Serve`, sleeping for the parsed duration or exiting non-zero on a malformed value
- `plugins/mock/plugin_test.go` - `TestLaunchDelayFromEnv`, a 6-case table test mirroring `TestReadinessWindowFromEnv`'s shape
- `kernel/supervisor/launchlatency_test.go` (new) - `TestResume_SlowRelaunchDoesNotFreezeOtherSources`, the hermetic cross-source-isolation gate
- `kernel/supervisor/readiness_test.go` - Comment-only fix closing 08-REVIEW.md IN-01
- `docs/testing.md` - New `WEBSPACES_MOCK_LAUNCH_DELAY_MS` subsection and a dated `What changed` entry

## Decisions Made
- Kept `Supervisor.mu`'s scope and duration completely unchanged (still the sole mutation lock for `Apply`/`SuspendInstance`/its resume closure/`Shutdown`) rather than trying to narrow its hold time — the fix is entirely about giving readers a separate, faster path, not about relaxing the mutation guarantee.
- `Host.Reconcile`'s write lock covers only the kill-and-commit region, matching the plan's explicit requirement and preserving `Reconcile`'s existing T-07-11 ordering guarantee (every needed launch succeeds before anything running is torn down) untouched.
- Chose 4000ms/2s as the fixture delay and gate threshold (not a smaller/tighter pair) so the margin is two orders of magnitude in both directions, avoiding a flaky knife-edge assertion — documented inline in the test.

## Deviations from Plan

None - plan executed exactly as written. All acceptance criteria verified: source assertions on `Host()`/`Coordinator()`/`Apply`/`SuspendInstance`/`Shutdown`/`Reconcile`/`Plugins()` all confirmed via grep against the committed code; `git diff --stat` for Task 1 touched exactly the 5 files named in its `<files>`; the negative control was run and its failure recorded above; `docs/plugin-contract.md` is untouched; no `go.mod`/`go.sum` changed in any module.

## Issues Encountered

During verification, an early round of `go build`/`go test` commands was accidentally run against the main repository checkout (`/home/darren/projects/davison/topos`) rather than this worktree (`/home/darren/projects/davison/topos/.claude/worktrees/agent-a3dc9ecd6a13b4cda`) — a `cd` to an absolute path that pointed at the wrong tree. Those runs passed trivially because they were compiling unmodified code, not this plan's changes, and produced no evidence about the actual work. Caught before any commit; all verification was re-run from the correct worktree root (relative paths, no `cd` off the default cwd) with genuine results, including the race-detector runs and the negative control. No code or commit was affected — this was a self-caught verification-tooling mistake, not a deviation in the shipped change itself.

## User Setup Required

None - no external service configuration required. `WEBSPACES_MOCK_LAUNCH_DELAY_MS` is a test-only fixture, never shipped in `make build`/`make plugins`, and off by default.

## Next Phase Readiness

- Phase success criterion 4's cross-source isolation clause now has a hermetic, real-subprocess gate — plan 08-14 (removing the current 15-second boot-latency contributor) and plan 08-15 (running the full Playwright suite as this wave's gate) can proceed on top of this structural fix.
- No blockers. `make test-portable` is green across all eight workspace modules, confirming no other module depended on `Host.Plugins()` returning the live backing slice.

## Self-Check: PASSED

- FOUND: kernel/supervisor/launchlatency_test.go
- FOUND: .planning/phases/08-whatsapp-conversations-managed-risk/08-13-SUMMARY.md
- FOUND commit: ab732f6
- FOUND commit: 2309a30

---
*Phase: 08-whatsapp-conversations-managed-risk*
*Completed: 2026-08-11*
