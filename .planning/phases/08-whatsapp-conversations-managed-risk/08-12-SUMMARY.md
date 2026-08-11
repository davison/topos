---
phase: 08-whatsapp-conversations-managed-risk
plan: 12
subsystem: sync-scheduler
tags: [go, scheduler, plugin-readiness, sync_runs, mock-plugin, hermetic-testing, go-plugin]

# Dependency graph
requires:
  - phase: 08-11
    provides: "The plugin's own honest health state and a handshake that means ready — this plan's kernel-side and fixture-side fix for G-08-4's other two legs"
provides:
  - "A bounded, context-cancellable first-refresh retry on kernel/syncer.Scheduler (FirstRefreshRetryDelays, firstRefresh) that supersedes a generation's errored first-sync row within seconds instead of the sync interval"
  - "An opt-in, default-off launch-readiness window on the reference plugins/mock (WEBSPACES_MOCK_READY_AFTER_MS) giving the launch-readiness failure class its first fixture"
  - "kernel/supervisor/readiness_test.go's hermetic gate over a real mock subprocess, proven to fail when the retry is removed"
affects: [08-UAT, syncer, pluginhost, mock-plugin, testing-docs]

# Actuals (#2632)
actuals:
  tokens: 8700
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Superseding retry, not a readiness probe: a generation's first Coordinator.Refresh is retried on a bounded backoff schedule so a later successful sync_runs row supersedes an earlier errored one via LatestSyncRunPerSource's MAX(id)-per-source selection — no new sync_runs status invented"
    - "Fixture-only opt-in env var on a reference plugin, parsed via an injectable getenv func for unit-testability, guarding only the RPCs a real not-yet-ready plugin would also fail (Match/Health), never Describe (which pluginhost.launch treats as launch-critical)"

key-files:
  created:
    - plugins/mock/readiness.go
    - kernel/supervisor/readiness_test.go
  modified:
    - kernel/syncer/scheduler.go
    - kernel/syncer/scheduler_test.go
    - plugins/mock/plugin.go
    - plugins/mock/main.go
    - plugins/mock/plugin_test.go
    - docs/testing.md

key-decisions:
  - "A connecting-state Match failure still records an errored sync_runs row; the scheduler supersedes it with a later successful row rather than suppressing or special-casing the plugin's self-reported state — decided explicitly in the plan, implemented unchanged"
  - "FirstRefreshRetryDelays nil means 'use the package default two-delay schedule' (what supervisor.startScheduler produces, untouched); an explicitly empty non-nil slice means 'no retries at all' — tests set a compressed schedule to avoid sleeping out production delays"
  - "The mock's readiness window guards Match and Health only, never Describe or Fetch — Describe is launch-critical (pluginhost.launch fails the whole launch on a Describe error) and Fetch is never on the sync path this gap concerns"

patterns-established:
  - "A Scheduler-level, test-overridable delay schedule (nil = default, empty = disabled) is the house pattern for bounding a background retry loop without a sleep-heavy test suite"

requirements-completed: [SRC-03]

coverage:
  - id: D1
    description: "Scheduler retries a generation's first refresh on a bounded, context-cancellable backoff schedule; a recovering source's latest sync run supersedes the earlier error, a permanently broken source still ends on an error after exactly 1+len(delays) attempts, and ticker/manual refreshes are untouched"
    requirement: "SRC-03"
    verification:
      - kind: unit
        ref: "kernel/syncer/scheduler_test.go#TestScheduler_FirstRefreshRetriesUntilTheSourceIsReady"
        status: pass
      - kind: unit
        ref: "kernel/syncer/scheduler_test.go#TestScheduler_FirstRefreshGivesUpAndLeavesTheErrorRecorded"
        status: pass
      - kind: unit
        ref: "kernel/syncer/scheduler_test.go#TestScheduler_FirstRefreshRetryStopsOnContextCancel"
        status: pass
    human_judgment: false
  - id: D2
    description: "plugins/mock gains an opt-in, default-off launch-readiness window (WEBSPACES_MOCK_READY_AFTER_MS) that refuses Match/Health for a configured window while Describe keeps answering, and fails startup loudly on a malformed value"
    requirement: "SRC-03"
    verification:
      - kind: unit
        ref: "plugins/mock/plugin_test.go#TestReadinessWindowFromEnv"
        status: pass
      - kind: unit
        ref: "plugins/mock/plugin_test.go#TestMatch_ReadinessWindowInFutureRefusesMatchButDescribeStillSucceeds"
        status: pass
      - kind: unit
        ref: "plugins/mock/plugin_test.go#TestMatch_ReadinessWindowElapsedReturnsNormalItemSet"
        status: pass
    human_judgment: false
  - id: D3
    description: "A hermetic end-to-end gate drives a real mock subprocess through a launch-readiness window and proves the source ends up synced (ok latest run + persisted items) rather than pinned on an error — demonstrated to fail when the scheduler's retry is reverted"
    requirement: "SRC-03"
    verification:
      - kind: integration
        ref: "kernel/supervisor/readiness_test.go#TestBoot_FirstRefreshSurvivesAPluginLaunchReadinessWindow"
        status: pass
    human_judgment: false

duration: 6min
completed: 2026-08-11
status: complete
---

# Phase 08 Plan 12: Kernel launch-readiness retry + mock fixture (G-08-4) Summary

**Scheduler retries a generation's first refresh on a bounded 2s/5s backoff so a launch-window Match failure is superseded within seconds instead of pinning the source for the 15-minute sync interval, and plugins/mock gains an opt-in `WEBSPACES_MOCK_READY_AFTER_MS` window giving this failure class its first hermetic, real-subprocess gate.**

## Performance

- **Duration:** ~6 min (task execution; excludes file-reading/context-gathering overhead)
- **Started:** 2026-08-11T00:17:09Z
- **Completed:** 2026-08-11T00:23:04Z
- **Tasks:** 3/3
- **Files modified:** 8 (2 created, 6 modified)

## Accomplishments

- `kernel/syncer.Scheduler` gained a bounded, context-cancellable first-refresh retry (`FirstRefreshRetryDelays`, `firstRefresh`) — a generation's immediate first refresh is retried on failure until it succeeds or the schedule is exhausted; ticker refreshes and every other `Coordinator.Refresh` caller (manual-refresh HTTP route, CLI) are byte-identical to before.
- `plugins/mock` gained an opt-in, default-off launch-readiness window (`WEBSPACES_MOCK_READY_AFTER_MS`) documented as a test fixture (not a contract feature) in its own file, `plugins/mock/readiness.go` — Match/Health refuse while the window is open, Describe never gated.
- `kernel/supervisor/readiness_test.go`'s `TestBoot_FirstRefreshSurvivesAPluginLaunchReadinessWindow` is the first hermetic gate in this repo for G-08-4's whole failure class: it boots a real supervisor against a real mock subprocess refusing `Match` for 700ms and proves the source ends up with an `ok` latest sync run and persisted, streamable items.
- Ran the plan's mandated negative control by hand: reverted `runSource` to call `refreshAndLog` directly (undoing Task 1's fix), reran the new gate, and confirmed it fails with the launch-window error pinned as the latest sync run — then restored the fix and reconfirmed green. This is recorded evidence the gate is real, not vacuous.

## Task Commits

Each task was committed atomically:

1. **Task 1: The scheduler retries a generation's first refresh instead of pinning it for 15 minutes** - `5f694dc` (feat)
2. **Task 2: The reference mock plugin gains an opt-in launch-readiness window** - `05ff94f` (feat)
3. **Task 3: An end-to-end gate that drives a real mock subprocess through a launch-readiness window** - `4f3f245` (test)

**Plan metadata:** pending (this commit)

## Files Created/Modified

- `kernel/syncer/scheduler.go` - `FirstRefreshRetryDelays` field, `defaultFirstRefreshRetryDelays`, `firstRefresh` retry loop, `refreshAndLog` now returns success bool
- `kernel/syncer/scheduler_test.go` - `flakySource` fake, three new tests (retry-until-ready, bounded give-up, context-cancel-during-backoff)
- `plugins/mock/readiness.go` - new: `readyAfterEnvVar`, `notReadyMessage`, `readinessWindow`, `readinessWindowFromEnv`
- `plugins/mock/plugin.go` - `ready *readinessWindow` field, `withReadinessWindow` setter, Match/Health guards, Describe/Fetch left unguarded with explanatory comments
- `plugins/mock/main.go` - parses the env var before `goplugin.Serve`, fails loudly on a bad value
- `plugins/mock/plugin_test.go` - table test over `readinessWindowFromEnv` plus Match/Describe behaviour inside and after the window
- `kernel/supervisor/readiness_test.go` - new: `TestBoot_FirstRefreshSurvivesAPluginLaunchReadinessWindow`
- `docs/testing.md` - new "`WEBSPACES_MOCK_READY_AFTER_MS`" subsection under "The two mock-shaped plugins", dated entry in "What changed"

## Decisions Made

- G-08-4's `missing[2]` decided exactly as the plan specified: a connecting-state Match failure still records an errored `sync_runs` row (no new status invented, no schema/API/SPA ripple); the scheduler supersedes it with a later successful row via `LatestSyncRunPerSource`'s existing `MAX(id)`-per-source selection.
- `FirstRefreshRetryDelays` semantics: `nil` = package default (`2s, 5s`), non-nil-empty = no retries — chosen so `supervisor.startScheduler` (left untouched by this plan) gets the default automatically while tests can compress the schedule without a sleep-heavy suite.
- The mock's readiness guard sits on Match and Health only; Describe is deliberately never gated (pluginhost.launch calls it immediately post-handshake and treats an error there as a launch failure — a guarded Describe would make the fixture's own scenario unreachable) and Fetch is unguarded since it's never on the sync path.

## Deviations from Plan

None - plan executed exactly as written. All three tasks' `<action>` and `<verify>` blocks were followed as specified, including the exact `FirstRefreshRetryDelays` nil/empty-slice semantics, the three-named-edit constraint on `plugin.go`, and the negative-control instruction in `<verification>`.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required. `WEBSPACES_MOCK_READY_AFTER_MS` is off by default and never set by `make e2e` or any real installation.

## Next Phase Readiness

- G-08-4 is fully closed across all three legs (08-11's plugin honesty fix, this plan's kernel retry + fixture). The kernel is now tolerant of the launch-readiness window every subprocess plugin inherently has, and this failure class has permanent hermetic armor via `kernel/supervisor/readiness_test.go`.
- No blockers. This plan's scope was deliberately backend-only (no frontend files touched, per the plan's own scope note) — the Playwright suite is unaffected and needs no update.

---
*Phase: 08-whatsapp-conversations-managed-risk*
*Completed: 2026-08-11*
