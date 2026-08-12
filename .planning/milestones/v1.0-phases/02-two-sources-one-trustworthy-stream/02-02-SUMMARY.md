---
phase: 02-two-sources-one-trustworthy-stream
plan: 02
subsystem: sync-engine
tags: [singleflight, ticker, health-check, coordinator, go-plugin]

requires:
  - phase: 02-two-sources-one-trustworthy-stream
    provides: "02-01: source-major sync engine (correlate.Engine.SyncSource/SyncAll, index.Store.ReplaceWebspaceSourceItems) that this plan's coordinator wraps"
provides:
  - "kernel/syncer (new package): Coordinator (single-flight per-source sync, two-phase sync_runs write) and Scheduler (one ticker goroutine per configured source, immediate first run)"
  - "kernel/index.Store: StartSyncRun/FinishSyncRun (two-phase write), LatestSyncRunPerSource, SyncingSourceTypes — replaces the single-write RecordSyncRun"
  - "kernel/config: [sync] interval (default 15m) + per-source sync_interval override, SyncIntervalFor, load-time validation"
  - "kernel/pluginhost: Plugin.DisplayName(), Host.ProbeSources (concurrent live Health RPC probe)"
  - "kernel/httpapi: GET /api/sources, POST /api/sources/{name}/refresh, POST /api/sync, aggregateSyncStatus (multi-source sync object)"
affects: [02-03, 02-04, kernel/sync (later plugins inherit the coordinator for free by being a configured source)]

tech-stack:
  added:
    - "golang.org/x/sync v0.22.0 (singleflight.Group) — added to the root module; already present as an indirect dep via plugins/silverbullet"
  patterns:
    - "Two-phase sync_runs write (StartSyncRun then FinishSyncRun on the same row) — the running/finished state a live 'is this syncing right now' read depends on"
    - "Single-flight-per-source coordinator (golang.org/x/sync/singleflight.Group.Do keyed by source name) as the ONE entry point every caller of a sync must go through — scheduler, manual refresh routes, and the CLI all call Coordinator.Refresh/RefreshAll, never the correlation engine directly"
    - "Health merge: live reachability from a per-request plugin Health RPC probe, sync history (last_status/last_sync_unix/last_error) exclusively from the kernel's own sync_runs table — a plugin's self-reported history is never trusted"
    - "aggregateSyncStatus: error > running > ok > zero-value precedence across every configured source's latest run, so a webspace with one silently-broken source can never report an unqualified ok"

key-files:
  created:
    - kernel/syncer/{coordinator.go,coordinator_test.go,scheduler.go,scheduler_test.go}
    - kernel/httpapi/{sources.go,sources_test.go}
  modified:
    - kernel/index/store.go (StartSyncRun/FinishSyncRun/LatestSyncRunPerSource/SyncingSourceTypes; RecordSyncRun removed)
    - kernel/index/store_test.go
    - kernel/correlate/correlate.go (SyncSource returns rejections string, no longer writes sync_runs; SyncAll deleted)
    - kernel/correlate/correlate_test.go
    - kernel/config/{types.go,config.go,config_test.go} ([sync] interval, per-source sync_interval, SyncIntervalFor)
    - kernel/pluginhost/host.go (DisplayName, ProbeSources)
    - kernel/httpapi/{routes.go,stream.go,webspaces.go,contract_test.go}
    - cmd/webspaces/main.go (coordinator + scheduler wiring, CLI routed through RefreshAll)
    - config.example.toml, docs/api.md, go.mod, go.sum

key-decisions:
  - "kernel/syncer package name (not kernel/sync as 02-PATTERNS.md's analog search suggested) — the package needs both golang.org/x/sync/singleflight and the standard library's own sync (sync.WaitGroup in scheduler.go), and a package literally named sync would force every later concurrent-plugin file (IMAP IDLE, WhatsApp WebSocket in Phases 3-5) to alias one import or the other"
  - "correlate.Engine.SyncSource's signature changed to (results []WebspaceResult, rejections string) rather than staying []WebspaceResult as the plan's <interfaces> 'Existing after 02-01' block literally showed — the coordinator needs the aggregated per-item rejection message to record on the sync_runs row it now owns, and the plan's own Task 1 action text explicitly specifies this return shape; treated the action text as authoritative over the summary block, matching how 02-01-SUMMARY.md told this executor to prefer live/detailed instructions over stale sketches"
  - "correlate.Engine.NowFunc/now() removed — dead code once sync_runs recording moved to the coordinator (StartSyncRun/FinishSyncRun use SQLite's own unixepoch(), not an injected clock)"
  - "kernel/syncer.Scheduler.Coordinator is the concrete *Coordinator type per the plan's locked <interfaces> block (not an interface) — scheduler_test.go drives a real Coordinator wrapping an in-memory fake correlate.Source instead of substituting the Coordinator itself"
  - "SourcesHandler's last_status/last_sync_unix/last_error come exclusively from LatestSyncRunPerSource, never from the live probe's own error text — a source reported as reachable:false with a non-empty last_error in tests is backed by a previously-recorded sync_runs failure, consistent with A-PLUG-04's 'the kernel is the single source of truth for sync history'"

requirements-completed: [KERN-04, PLUG-04]

coverage:
  - id: D1
    description: "A refresh request for a source already syncing coalesces into the in-flight run instead of running twice or queuing"
    requirement: KERN-04
    verification:
      - kind: unit
        ref: "kernel/syncer/coordinator_test.go#TestRefresh_ConcurrentCallsCoalesceIntoOneSyncCycle (run with -race)"
        status: pass
    human_judgment: false
  - id: D2
    description: "The kernel syncs every configured source immediately at startup and then on a repeating [sync] interval (default 15m, per-source override)"
    requirement: KERN-04
    verification:
      - kind: unit
        ref: "kernel/syncer/scheduler_test.go#TestScheduler_Run_ImmediateFirstRunThenTicks, #TestScheduler_Run_TwoSourcesTickIndependently"
        status: pass
      - kind: integration
        ref: "live `webspaces serve` startup log: both paperless and silverbullet completed a scheduled sync run within 0.3s of process start, before one interval elapsed"
        status: pass
    human_judgment: false
  - id: D3
    description: "GET /api/sources reports per-source name, source_type, display_name, live reachable, syncing, last_status, last_sync_unix, last_error"
    requirement: PLUG-04
    verification:
      - kind: unit
        ref: "kernel/httpapi/sources_test.go#TestSourcesHandler_ReturnsBothSourcesSortedByName, #TestSourcesHandler_NoRecordedRunReportsNeutralUnknownState, #TestSourcesHandler_RunningRowReportsSyncingTrue, #TestSourcesHandler_ProbeFailureForOnePluginDoesNotFailWholeResponse"
        status: pass
      - kind: integration
        ref: "live GET /api/sources against the user's real paperless-ngx + SilverBullet instances returned both sources reachable:true with correct plugin-reported display_name (\"paperless-ngx\", \"SilverBullet\")"
        status: pass
    human_judgment: false
  - id: D4
    description: "POST /api/sources/{name}/refresh and POST /api/sync trigger a sync through the coordinator and return run status; an unconfigured name returns 404 source_not_found without enumerating configured names"
    requirement: KERN-04
    verification:
      - kind: unit
        ref: "kernel/httpapi/sources_test.go#TestSourceRefreshHandler_ValidNameReturns200WithRunStatus, #TestSourceRefreshHandler_UnconfiguredNameReturns404WithoutEnumerating, #TestSyncRefreshHandler_ReturnsOneStatusPerSource"
        status: pass
      - kind: integration
        ref: "live: two successive POST /api/sources/silverbullet/refresh calls both reported item_count 17 (no duplication); POST /api/sources/does-not-exist/refresh returned 404 source_not_found"
        status: pass
    human_judgment: false
  - id: D5
    description: "The stream/webspace-list sync object aggregates every configured source: error if any source's latest run errored, so a webspace with a silently-failing source never looks merely empty"
    requirement: KERN-04
    verification:
      - kind: unit
        ref: "kernel/httpapi/sources_test.go#TestAggregateSyncStatus_ErrorTakesPrecedenceOverOK, #TestAggregateSyncStatus_RunningTakesPrecedenceOverOK, #TestStreamHandler_SyncStatusErrorWhenOneOfTwoSourcesFailed"
        status: pass
    human_judgment: false
  - id: D6
    description: "Every sync in the system — scheduler, per-source refresh, refresh-all, and the CLI — reaches a source's sync through the one Coordinator entry point; no caller reaches the correlation engine or a plugin directly"
    requirement: KERN-04
    verification:
      - kind: unit
        ref: "shell assertion (Task 1 verify): non-test importer set of kernel/correlate is exactly cmd/webspaces/main.go, kernel/correlate/correlate.go, kernel/syncer/coordinator.go"
        status: pass
    human_judgment: false

duration: 35min
completed: 2026-07-28
status: complete
---

# Phase 2 Plan 2: One Coordinator, Every Sync Summary

**A `kernel/syncer` package (`Coordinator` + `Scheduler`) makes every sync in the system — background ticks, manual per-source/all-source refresh, and the `webspaces sync` CLI — go through one single-flight entry point, with a two-phase `sync_runs` write and a kernel-side health merge (`GET /api/sources`) that never trusts a plugin's self-reported sync history.**

## Performance

- **Duration:** ~35 min (active implementation across 3 tasks, plus live verification against the user's real paperless-ngx + SilverBullet instances)
- **Completed:** 2026-07-28
- **Tasks:** 3 completed
- **Files modified:** 21 (8 new, 13 modified), excluding `.planning/`

## Accomplishments

- `kernel/index.Store` gained a two-phase `sync_runs` write (`StartSyncRun`/`FinishSyncRun`) plus `LatestSyncRunPerSource`/`SyncingSourceTypes`, replacing the old single-write `RecordSyncRun` — a sync is now visible mid-flight, not just after it finishes.
- `kernel/syncer.Coordinator` wraps `correlate.Engine.SyncSource` in `golang.org/x/sync/singleflight`, proven by a `-race` test that two concurrent same-source refreshes drive exactly one underlying `Match` call.
- `kernel/syncer.Scheduler` runs one ticker goroutine per configured source (immediate first run, then `[sync] interval` or a per-source `sync_interval` override), replacing Phase 1's one-shot startup goroutine.
- `cmd/webspaces/main.go`'s `runSync` and `runServe` both route exclusively through the coordinator — the correlation engine's sync methods have no other caller.
- `GET /api/sources` merges a live per-plugin `Health` RPC probe (`kernel/pluginhost.Host.ProbeSources`) with the kernel's own recorded sync history; `POST /api/sources/{name}/refresh` and `POST /api/sync` dispatch through the identical coordinator the scheduler uses.
- The `sync` object on the stream and webspace-list envelopes is now an aggregate (`aggregateSyncStatus`) across every configured source, so a webspace whose only failing source returned zero items can never report an unqualified `ok`.
- Verified live against the user's real instances: startup synced both sources within 0.3s (before the 15-minute interval), `GET /api/sources` reported correct plugin-reported `display_name`s, and two successive manual refreshes of silverbullet left the item count at 17 (no duplication).

## Task Commits

1. **Task 1: One sync per source at a time, recorded while it is happening** - `c3efac9` (feat)
2. **Task 2: The kernel syncs on a schedule, and the CLI goes through the coordinator too** - `1c91f6e` (feat)
3. **Task 3: The API reports per-source health and accepts a manual refresh** - `0c577b1` (feat)

_Note: this plan's `tdd="true"` tasks were executed as single atomic commits per the `type="auto"` commit protocol (real implementation + real tests + real `<verify>` together), consistent with how 02-01-PLAN.md's tracer task was executed and documented — each task's own behavior/verify blocks were satisfied within its one commit rather than split into separate RED/GREEN commits._

## Files Created/Modified

- `kernel/syncer/coordinator.go` - `Coordinator`, `RunResult`, `ErrUnknownSource`; single-flight `Refresh`/`RefreshAll`
- `kernel/syncer/coordinator_test.go` - single-flight race proof, rejection-propagation, unknown-source, refresh-all partial-failure, no-duplicate-on-repeat tests
- `kernel/syncer/scheduler.go` - `Scheduler.Run`, one ticker goroutine per source
- `kernel/syncer/scheduler_test.go` - immediate-first-run, clean-exit-on-cancel, independent-per-source-ticking tests
- `kernel/index/store.go` - `StartSyncRun`, `FinishSyncRun`, `LatestSyncRunPerSource`, `SyncingSourceTypes`; `RecordSyncRun` removed, `LatestSyncRun` kept (still used nowhere post-Task-3 but retained per the plan's own instruction)
- `kernel/index/store_test.go` - two-phase write, per-source latest-run, syncing-set tests
- `kernel/correlate/correlate.go` - `SyncSource` returns `(results, rejections string)`, no longer writes `sync_runs`; `SyncAll` and `NowFunc`/`now()` removed
- `kernel/correlate/correlate_test.go` - adapted to per-source-call shape (no more `SyncAll`)
- `kernel/config/{types.go,config.go}` - `SyncConfig`, `DefaultSyncInterval`, `Source.SyncInterval`, `SyncIntervalFor`, interval validation
- `kernel/config/config_test.go` - interval default/override/validation tests
- `kernel/pluginhost/host.go` - `Plugin.DisplayName()`, `SourceHealth`, `Host.ProbeSources` (concurrent probe)
- `kernel/httpapi/sources.go` - `HealthProber`, `Refresher`, `SourcesHandler`, `SourceRefreshHandler`, `SyncRefreshHandler`, `aggregateSyncStatus`
- `kernel/httpapi/sources_test.go` - full handler + aggregate coverage
- `kernel/httpapi/{routes.go,stream.go,webspaces.go}` - `Router` widened; both single-`LatestSyncRun` call sites replaced with `aggregateSyncStatus`
- `kernel/httpapi/contract_test.go` - updated `Router(...)` call sites for the widened signature
- `cmd/webspaces/main.go` - `newCoordinator` helper; `runSync`/`runServe` route through it exclusively
- `config.example.toml` - documents `[sync] interval` and per-source `sync_interval`
- `docs/api.md` - documents `/api/sources`, the two refresh routes, `source_not_found`, and the aggregate `sync` object's new semantics
- `go.mod`, `go.sum` - `golang.org/x/sync v0.22.0` added to the root module

## Decisions Made

- **`kernel/syncer`, not `kernel/sync`** — deliberately diverges from `02-PATTERNS.md`'s suggested analog because the package genuinely needs both `golang.org/x/sync/singleflight` and the standard library's `sync` (for `scheduler.go`'s `sync.WaitGroup`); a package literally named `sync` would force an import alias in this file and in every later concurrent-plugin file Phases 3-5 add (IMAP IDLE, WhatsApp's WebSocket).
- **`SyncSource` returns `(results, rejections string)`** rather than staying `[]WebspaceResult`-only as the plan's `<interfaces>` "Existing after 02-01" block literally showed. Task 1's own action text explicitly specifies this return shape (the coordinator needs the aggregated rejection message to write into `sync_runs.error`), and that specific, detailed instruction was treated as authoritative over the necessarily-abbreviated interfaces summary — the same reconciliation principle this plan's own `files_to_read` note applied to 02-01-SUMMARY.md.
- **`Engine.NowFunc`/`now()` removed** — dead code once `sync_runs` timestamps moved to the coordinator's `StartSyncRun`/`FinishSyncRun`, which use SQLite's own `unixepoch()` rather than an injected clock.
- **`Scheduler.Coordinator` stayed the concrete `*Coordinator` type**, per the plan's locked `<interfaces>` block, rather than an interface — `scheduler_test.go` drives a real `Coordinator` wrapping an in-memory fake `correlate.Source` instead of substituting the `Coordinator` itself.
- **`SourcesHandler`'s `last_error` is sourced exclusively from `sync_runs`**, never from the live probe's own error text — a source shown `reachable:false` with a non-empty `last_error` in this plan's tests is backed by a previously-recorded `sync_runs` failure, consistent with A-PLUG-04's "the kernel is the single source of truth for sync history."

## Deviations from Plan

### Auto-fixed Issues

None beyond the interfaces-block reconciliation and dead-code removal documented above under Decisions Made — both are Rule 1-adjacent corrections needed for the plan's own Task 1 action text and acceptance criteria to be internally satisfiable, not new functionality.

---

**Total deviations:** 0 requiring separate deviation entries (the SyncSource signature and NowFunc removal are documented as decisions, since they resolve an internal inconsistency in the plan's own text rather than fixing a bug or adding missing functionality)
**Impact on plan:** No scope creep. Every file touched was already listed in the plan's `files_modified`.

## Issues Encountered

None — the plan's own task ordering (persistence primitives, then scheduler/CLI wiring, then HTTP surface) meant each task's `go build ./...` succeeded before moving to the next.

## User Setup Required

None beyond what 02-01-PLAN.md's `user_setup` already covered (`SB_URL`/`SB_AUTH_TOKEN`, `~/.config/webspaces/config.toml`). This plan added no new required config — `[sync] interval` and `sync_interval` are both optional with documented defaults.

## Next Phase Readiness

- `kernel/syncer.Coordinator`/`Scheduler` are the sync entry point every later-phase plugin (IMAP, Signal, WhatsApp) inherits for free by being a configured source — scheduling, single-flight dedup, and health reporting all come from `config.Sources` + `pluginhost.Discover`, with no per-plugin scheduler code needed.
- `GET /api/sources`'s exact JSON shape (documented above and in `docs/api.md`) is what 02-03-PLAN.md's UI health chips and refresh buttons should be written against: `{ "schema_version": 1, "sources": [{ "name", "source_type", "display_name", "reachable", "syncing", "last_status", "last_sync_unix", "last_error" }] }`.
- The `Router` signature is now `Router(store, cfg, fetcher, prober, refresher)` — any future route addition should follow the same narrow-interface pattern (`HealthProber`, `Refresher`) rather than taking concrete `*pluginhost.Host`/`*syncer.Coordinator` params, to keep handlers testable with fakes.
- `web/src/lib/api.ts`'s `sourceDisplayName()` local mapping (added in 02-01) can now be replaced with the live, plugin-reported `display_name` from `GET /api/sources` — flagged in 02-01-SUMMARY.md as pending exactly this endpoint, now available for 02-03.
- No blockers identified for 02-03 (filter UI, health chips, refresh buttons) or 02-04.

---
*Phase: 02-two-sources-one-trustworthy-stream*
*Completed: 2026-07-28*

## Self-Check: PASSED

All 22 claimed files exist on disk and all three task commits (`c3efac9`, `1c91f6e`, `0c577b1`) are present in `git log`.
