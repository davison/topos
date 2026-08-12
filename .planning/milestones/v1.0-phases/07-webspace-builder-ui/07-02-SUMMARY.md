---
phase: 07-webspace-builder-ui
plan: 02
subsystem: api
tags: [go, hot-reload, config, pluginhost, syncer, supervisor, http]

# Dependency graph
requires:
  - phase: 07-webspace-builder-ui
    provides: "07-01's config.Store (raw/expanded split, content-hash lock, canonical writer, PUT /api/config) — this plan's Apply seam swaps in behind that same Save/Reload pair"
provides:
  - "kernel/supervisor.Supervisor: the apply seam that rebuilds the plugin host, sync coordinator and background scheduler in place after every config.Store.Save/Reload — no kernel restart (D-06)"
  - "pluginhost.Host.Reconcile: incremental add/change/remove diff against a launched plugin set, killing only what one apply call itself launched on failure (T-07-11)"
  - "POST /api/config/reload (D-08): the hand-edit pickup path, same validate-then-apply sequence a save uses, never dies on a bad file"
  - "Every httpapi handler (WebspacesHandler/ItemHandler/SourceRefreshHandler/StreamHandler/SearchHandler) reads cfgStore fresh per request — no boot-time *config.Config snapshot survives anywhere in routes.go"
  - "pluginhost.DiscoverBinaries + DescribePluginType: GET /api/config/plugin-types and POST /api/config/describe-plugin, the kernel half of the two-step 'New <plugin type>...' modal (D-11), trial-launch-then-kill, persists nothing"
affects: [07-03, 07-04, 07-05]

# Actuals (#2632) — pairs with the plan's estimate to calibrate future estimates.
actuals:
  tokens: 29943
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "kernel/supervisor.Supervisor: a single mutex-guarded owner of (host, coordinator, scheduler generation), with Apply as the one seam a config.Store swap hands its result to — full-rebuild-of-scheduler-per-apply chosen over a per-source diff (07-RESEARCH.md's accepted MVP tradeoff), while pluginhost.Host.Reconcile diffs incrementally so an unrelated source's subprocess is never disturbed"
    - "Supervisor itself satisfies Fetcher/HealthProber/Refresher by delegating to Host()/Coordinator() fresh on every call — never a *Host/*Coordinator pointer captured once at Router construction, since Apply replaces the *syncer.Coordinator wholesale on every successful reconcile (no in-place source-set mutation exists on Coordinator)"
    - "Live-config-per-request: every httpapi handler resolves cfgStore.Expanded() as the first statement of its own returned closure — routes.go holds zero local *config.Config values, enforced by an AST guard scanning for the identifier 'cfg'"
    - "Trial-launch-then-kill for Describe (DescribePluginType): reuses pluginhost's unexported launch() verbatim under a throwaway instance name, writes nothing, registers nothing on any Host — pinned by an AST guard over its own function body reaching no RPC beyond Describe"

key-files:
  created:
    - kernel/supervisor/supervisor.go
    - kernel/supervisor/supervisor_test.go
    - kernel/pluginhost/discover_binaries.go
    - kernel/pluginhost/discover_binaries_test.go
    - kernel/pluginhost/describe_test.go
    - kernel/pluginhost/reconcile_test.go
    - kernel/httpapi/live_config_test.go
  modified:
    - kernel/pluginhost/host.go
    - kernel/index/store.go
    - kernel/index/store_test.go
    - kernel/syncer/scheduler.go
    - kernel/httpapi/config.go
    - kernel/httpapi/config_test.go
    - kernel/httpapi/routes.go
    - kernel/httpapi/webspaces.go
    - kernel/httpapi/item.go
    - kernel/httpapi/sources.go
    - kernel/httpapi/contract_test.go
    - kernel/httpapi/agent_test.go
    - kernel/httpapi/item_test.go
    - kernel/httpapi/sources_test.go
    - cmd/topos/main.go
    - docs/api.md

key-decisions:
  - "In-flight sync handling during apply (07-RESEARCH.md Open Question 2): cancel the OLD scheduler generation's context and BLOCK until its Run call fully returns before Host.Reconcile runs — never drain/wait for the sync itself; Coordinator.syncOne's pre-existing detached sync_runs finalize (kernel/syncer/coordinator.go, proven since Phase 2) records the interrupted run cleanly, identical to kernel shutdown"
  - "Rule 1 bug found during Task 1 implementation, not in the plan's own text: Apply replaces the *syncer.Coordinator wholesale on every successful reconcile, so passing sup.Host()/sup.Coordinator() (captured once) into httpapi.Router would freeze the refresher at whatever coordinator existed at kernel-start — a source added by any later apply would be permanently unreachable through POST /api/sources/{name}/refresh and POST /api/sync. Fixed by making Supervisor itself satisfy Fetcher/HealthProber/Refresher, delegating to Host()/Coordinator() fresh on every call; cmd/topos/main.go passes sup itself into Router, never sup.Host()/sup.Coordinator()"
  - "A failed Reconcile/ValidateMatchConfig during Apply restarts the scheduler against the OLD (pre-apply) host/coordinator/cfg triple, not the new one — keeps the sync subsystem self-consistent (never scheduling against instances the host doesn't have) even though cfgStore's GET /api/config already reflects the new document; the plan's own 'indeterminate state, a reload can recover' framing is satisfied by GET /api/config alone changing, not by the sync subsystem silently drifting further"
  - "Match-only edit resync (a webspace/match block changed, connection config did not): dispatches an explicit async Coordinator.Refresh per unchanged instance when cfg.Webspaces differs old-to-new, on top of (not instead of) the full scheduler restart's own immediate-first-refresh — redundant with the restart in practice (Coordinator.Refresh's single-flight coalesces the two into one sync) but implemented literally per the plan's own action text rather than relying solely on the restart's incidental coverage"
  - "07-RESEARCH.md assumption A2 confirmed for all four non-Signal-verified plugin types by reading each main(): paperless/silverbullet/proton/signal all only check field PRESENCE before goplugin.Serve; none opens a live network/database connection first. Proton's NewClient is the one eager constructor and only validates the base_url scheme (imap/imaps) — a real Step 1 constraint, not a defect, recorded verbatim in docs/api.md's describe-plugin section"
  - "07-RESEARCH.md Open Question 1 decided: topos-plugin-mock is excluded from DiscoverBinaries' UI-facing result (ExcludedPluginBinaries), never offered by the '+' chip picker, though still a legitimate config.toml entry for anyone who hand-edits one in"

requirements-completed: [KERN-08]

coverage:
  - id: D1
    description: "A config save reconfigures the running kernel in place: an added instance's plugin is launched and gets an eager first sync, a removed instance's plugin is killed and its index rows/sync history deleted, an unchanged instance's subprocess is never restarted — all within the same PUT /api/config request (D-06/D-07)"
    requirement: "KERN-08"
    verification:
      - kind: unit
        ref: "kernel/pluginhost/reconcile_test.go#TestReconcile_UnchangedSourceMapKeepsSamePluginPointers"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/reconcile_test.go#TestReconcile_RemovedInstanceIsKilledAndDropped"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/reconcile_test.go#TestReconcile_LaunchFailureLeavesPreviouslyRunningSetIntact"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/reconcile_test.go#TestReconcile_ChangedConnectionConfigRelaunches"
        status: pass
      - kind: integration
        ref: "kernel/supervisor/supervisor_test.go#TestApply_RemovedInstance_PluginGoneAndIndexRowsGone"
        status: pass
      - kind: unit
        ref: "kernel/index/store_test.go#TestDeleteSourceItems_RemovesOnlyThatInstancesRowsEverywhere"
        status: pass
    human_judgment: false
  - id: D2
    description: "An apply landing mid-sync cancels that sync's scheduler context and leaves no sync_runs row stranded at status running"
    requirement: "KERN-08"
    verification:
      - kind: integration
        ref: "kernel/supervisor/supervisor_test.go#TestApply_MidFlightSyncLeavesNoStrandedRunningRow"
        status: pass
    human_judgment: false
  - id: D3
    description: "POST /api/config/reload re-reads config.toml through the same validate-then-apply path a save uses; an invalid reload leaves the last-good configuration running and returns 422 with the loader's own message, never kills the kernel"
    requirement: "KERN-08"
    verification:
      - kind: unit
        ref: "kernel/httpapi/config_test.go#TestConfigReloadHandler_ValidHandEditAppliesAndReturnsNewDocument"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/config_test.go#TestConfigReloadHandler_InvalidFileReturns422AndKeepsLastGoodConfig"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/config_test.go#TestConfigReloadHandler_ApplyFailureReturns500ApplyFailed"
        status: pass
    human_judgment: false
  - id: D4
    description: "Every HTTP handler resolves live config at request time — no handler in routes.go serves a value captured once at Router construction (the assumption-delta promote decision, Pitfall 1)"
    requirement: "KERN-08"
    verification:
      - kind: unit
        ref: "kernel/httpapi/live_config_test.go#TestLiveConfig_WebspacesHandler_KeywordsChangeVisibleWithoutRestart"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/live_config_test.go#TestLiveConfig_ItemHandler_DisplayNameChangeVisibleWithoutRestart"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/live_config_test.go#TestLiveConfig_SourceRefreshHandler_NewlyAddedSourceIsImmediatelyRefreshable"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/config_test.go#TestRoutesGuard_NoLocalConfigSnapshot"
        status: pass
    human_judgment: false
  - id: D5
    description: "The kernel reports which plugin types are installed (excluding the mock fixture) and, for a not-yet-configured type, its Describe-declared match vocabulary from just-submitted connection fields — writing nothing and registering nothing"
    requirement: "KERN-08"
    verification:
      - kind: unit
        ref: "kernel/pluginhost/discover_binaries_test.go#TestDiscoverBinaries_ExcludesMockBinary"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/config_test.go#TestDescribePluginHandler_UnknownBinaryReturns404WithoutExecuting"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/config_test.go#TestDescribePluginHandler_RealPaperlessBinary_ReturnsMatchVocabulary"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/config_test.go#TestDescribePluginTypeGuard_ReachesNoRPCBeyondDescribe"
        status: pass
    human_judgment: false
  - id: D6
    description: "docs/api.md documents POST /api/config/reload, GET /api/config/plugin-types, POST /api/config/describe-plugin and the three new error codes (apply_failed, plugin_binary_not_found, plugin_describe_failed)"
    verification: []
    human_judgment: true
    rationale: "Prose documentation accuracy is a judgment call beyond what a compiling-example or grep-based check can certify"

duration: ~2h (three tasks, no checkpoints — autonomous plan)
completed: 2026-08-08
status: complete
---

# Phase 7 Plan 2: Config Hot-Apply, Reload, and Plugin-Type Discovery Summary

**The kernel's config-change loop closes end to end: a `PUT /api/config` save now relaunches, kills, or leaves untouched every affected plugin subprocess in the same request (`kernel/supervisor.Supervisor.Apply`), `POST /api/config/reload` picks up hand-edits without ever risking the running kernel, every HTTP handler reads config live per request, and `GET /api/config/plugin-types` / `POST /api/config/describe-plugin` give the future "+" chip picker the two facts it can't compute client-side.**

## Performance

- **Duration:** ~2h across three tasks (autonomous execution, no checkpoints)
- **Started:** 2026-08-08 (session start)
- **Completed:** 2026-08-08T01:30:30+01:00 (Task 3's commit)
- **Tasks:** 3
- **Files touched:** 23 (7 created, 16 modified)

## Accomplishments

- `kernel/supervisor.Supervisor` is the apply seam 07-01 deliberately left unbuilt: `NewSupervisor` performs the kernel's boot sequence once (Discover, `ValidateMatchConfig`, build the coordinator, start the scheduler); `Apply` repeats the relevant part of it after every successful `config.Store.Save`/`Reload` — cancel-and-drain the old scheduler generation, `Host.Reconcile` the plugin set, delete removed instances' index rows/sync history, rebuild the coordinator, start a fresh scheduler generation whose own "immediate first refresh" design gives D-07's eager reconcile for free.
- `pluginhost.Host.Reconcile` diffs incrementally: an unchanged instance's `*Plugin` pointer survives byte-identical, a changed-connection-config instance is killed and relaunched, a removed instance is killed and dropped, and a launch failure for one added instance kills only what that call itself launched — the previously running set is never disturbed (T-07-11), all proven against the repo's own real mock plugin subprocess, not hand-built fakes with a nil client.
- `POST /api/config/reload` (D-08) closes the hand-edit loop: same validate-then-apply sequence a save uses, and `config.Store.Reload`'s pre-existing load-into-locals-then-swap discipline (already correct from 07-01) means an invalid file leaves the previously running configuration completely untouched — proven by writing a syntactically-valid-but-semantically-invalid file, reloading, and confirming a subsequent `GET /api/config` is byte-for-byte the pre-reload document.
- Every httpapi handler now resolves `cfgStore.Expanded()` fresh per request — `WebspacesHandler`, `ItemHandler` and `SourceRefreshHandler` were the last three holdouts from 07-01's tracer, and `routes.go` no longer declares a local `*config.Config` value anywhere, enforced by an AST guard scanning for the identifier `cfg`.
- `pluginhost.DiscoverBinaries` + `DescribePluginType` (trial-launch, `Describe`, kill, before anything is persisted) back two new routes, `GET /api/config/plugin-types` and `POST /api/config/describe-plugin` — the requested binary must be a member of `DiscoverBinaries`' own result set before anything executes (T-07-09), and `DescribePluginType`'s own function body is pinned by an AST guard to reach no RPC beyond `Describe` (T-07-10, PLUG-02).
- 07-RESEARCH.md's deferred assumption A2 is resolved for all four non-Signal-verified plugin types by reading each `main()`: paperless, silverbullet, proton and signal all defer live connectivity checks past process startup — only field *presence* is required, never a working connection. Proton's `NewClient` is the one eager constructor, and it only validates the `base_url` scheme (`imap`/`imaps`), never dials.

## Task Commits

Each task was committed atomically:

1. **Task 1: Apply on save — plugin host, coordinator and scheduler rebuilt in place** — `f25c4ab` (feat)
2. **Task 2: Reload a hand-edited file, and make every handler read live config** — `f4c2e98` (feat)
3. **Task 3: Report configurable plugin types and their match vocabulary before anything is persisted** — `894ab20` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified

- `kernel/supervisor/supervisor.go` (new) - `Supervisor`, `NewSupervisor`, `Apply`, `Shutdown`, `Host`/`Coordinator` accessors, and the `Fetch`/`ProbeSources`/`Refresh`/`RefreshAll` delegating methods that satisfy `httpapi.Fetcher`/`HealthProber`/`Refresher`
- `kernel/supervisor/supervisor_test.go` (new) - Apply-level integration tests: removed-instance cleanup against a real mock plugin, mid-flight-sync cancellation against a synthetic blocking source
- `kernel/pluginhost/host.go` - `Host.Reconcile`, `DescribeInfo`, `DescribePluginType`; `Plugin` gained a `src config.Source` field for Reconcile's diff; `Host` gained a `pluginsDir` field
- `kernel/pluginhost/reconcile_test.go` (new) - `Reconcile`'s four behavior guarantees, against a real built mock-plugin subprocess (required: `Plugin.Kill()` panics on a hand-built value with a nil client)
- `kernel/pluginhost/discover_binaries.go` (new) - `DiscoverBinaries`, `PluginBinaryPrefix`, `ExcludedPluginBinaries`
- `kernel/pluginhost/discover_binaries_test.go` (new) - sorted/mock-free/missing-directory behavior
- `kernel/pluginhost/describe_test.go` (new) - error-wrap shape (no subprocess) plus a real-plugin happy path
- `kernel/index/store.go` - `DeleteSourceItems`, `DeleteSyncRuns`
- `kernel/index/store_test.go` - cross-webspace/FTS deletion proof, no-op-for-unknown-source, sync-history scoping
- `kernel/syncer/scheduler.go` - doc-comment only: names the restart-a-fresh-Scheduler pattern `Supervisor.Apply` uses
- `kernel/httpapi/config.go` - `Applier` interface; `ConfigSaveHandler` now calls `Apply`; `ConfigReloadHandler`, `PluginTypesHandler`, `DescribePluginHandler` and their request/response types
- `kernel/httpapi/config_test.go` - reload success/invalid/apply-failure tests, plugin-types/describe-plugin tests (including a real built paperless plugin), the no-local-cfg-snapshot AST guard, the DescribePluginType-reaches-no-RPC AST guard, and the relaxed (pluginhost import now expected) plugin-call guard
- `kernel/httpapi/routes.go` - `Router` gains `applier`, `pluginsDir`, `logger` params; mounts `/api/config/reload`, `/api/config/plugin-types`, `/api/config/describe-plugin`; `WebspacesHandler`/`ItemHandler`/`SourceRefreshHandler` now take `cfgStore`
- `kernel/httpapi/webspaces.go`, `item.go`, `sources.go` - `cfg := cfgStore.Expanded()` moved inside each handler's closure
- `kernel/httpapi/live_config_test.go` (new) - the promote-decision invariant proven against the real `Router`: a keywords/display-name/newly-added-source change is visible on the next request to the same already-constructed router
- `kernel/httpapi/contract_test.go`, `agent_test.go`, `item_test.go`, `sources_test.go` - updated `Router(...)`/handler-constructor call sites for the new `Applier`/`*config.Store`/`pluginsDir`/`logger` parameters; `fakeApplier` test double added
- `cmd/topos/main.go` - `setup` no longer builds the plugin host; `runServe`/`runSync` both build a `*supervisor.Supervisor` and pass it (not `sup.Host()`/`sup.Coordinator()`) into `Router`
- `docs/api.md` - `POST /api/config/reload`, `GET /api/config/plugin-types`, `POST /api/config/describe-plugin` sections; `apply_failed`, `plugin_binary_not_found`, `plugin_describe_failed` error-code rows

## Decisions Made

- **In-flight sync during apply:** cancel the OLD scheduler generation and BLOCK until it fully returns before `Reconcile` runs, relying on `Coordinator.syncOne`'s pre-existing detached finalize for the interrupted sync's own outcome — never a drain/wait on the sync itself.
- **Rule 1 bug, found during implementation (not in the plan text):** `Apply` replaces the `*syncer.Coordinator` wholesale on every successful reconcile — passing `sup.Coordinator()` (captured once) into `Router` would have frozen manual refresh against an increasingly stale source set. Fixed by making `Supervisor` itself satisfy `Fetcher`/`HealthProber`/`Refresher`, resolving current state per call.
- **Failed-apply scheduler restart:** restarts against the OLD (pre-apply) host/coordinator/cfg triple, not the new one, keeping the sync subsystem self-consistent even though `GET /api/config` already reflects the new document — matches the plan's own "indeterminate state, a reload can recover" framing without letting the sync subsystem itself drift further on a failure.
- **Match-only-edit resync** is dispatched explicitly (a redundant-but-harmless single-flight-coalesced `Coordinator.Refresh`) on top of the full scheduler restart's own coverage, per the plan's literal action text.
- **07-RESEARCH.md assumption A2** confirmed for paperless/silverbullet/proton/signal; Proton's `NewClient` base_url-scheme check recorded as a real Step 1 constraint.
- **07-RESEARCH.md Open Question 1** decided: `topos-plugin-mock` excluded from `DiscoverBinaries`' UI-facing result, though still a legitimate hand-edited `config.toml` entry.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `Router`'s refresher would have frozen at kernel-start's coordinator**
- **Found during:** Task 1, while wiring `cmd/topos/main.go` to pass a fetcher/prober/refresher into `httpapi.Router`
- **Issue:** `Supervisor.Apply` replaces `s.coord` with a brand-new `*syncer.Coordinator` on every successful reconcile (no in-place source-set mutation exists on `Coordinator`). Passing `sup.Coordinator()` — evaluated once at `Router` construction — would mean every manual refresh (`POST /api/sources/{name}/refresh`, `POST /api/sync`) kept dispatching against the coordinator's source set as it existed at kernel boot, forever. A source added by ANY later apply would be permanently unreachable through those two routes, silently.
- **Fix:** `Supervisor` itself now implements `Fetch`/`ProbeSources`/`Refresh`/`RefreshAll`, each delegating to `Host()`/`Coordinator()` resolved fresh on every call. `cmd/topos/main.go` passes `sup` itself into `Router`, never `sup.Host()`/`sup.Coordinator()`.
- **Files modified:** `kernel/supervisor/supervisor.go`, `cmd/topos/main.go`
- **Verification:** `kernel/supervisor/supervisor_test.go#TestApply_RemovedInstance_PluginGoneAndIndexRowsGone` exercises `sup.Host()` after an `Apply` that replaced the coordinator; the fix is structural (delegation, not a snapshot) so no dedicated regression test was needed beyond the existing supervisor suite passing.
- **Committed in:** `f25c4ab` (Task 1 commit)

**2. [Rule 3 - Blocking compile dependency] Existing AST guard forbade `config.go` importing `pluginhost`**
- **Found during:** Task 3, wiring `DiscoverBinaries`/`DescribePluginType` into `kernel/httpapi/config.go`
- **Issue:** 07-01's `TestConfigHandlersGuard_NoPluginCallOtherThanDescribe` asserted `config.go` imports no package whose path contains `pluginhost` — a guard written before this plan's own Task 3 made that import necessary.
- **Fix:** Relaxed the guard to allow the `pluginhost` import while still forbidding a direct `/sdk/` import and any `Fetch`/`Match` selector reference in `config.go`'s own source (the actual boundary that matters); added a new, separate AST guard (`TestDescribePluginTypeGuard_ReachesNoRPCBeyondDescribe`) proving `DescribePluginType`'s own function body in `host.go` reaches no RPC beyond `Describe`.
- **Files modified:** `kernel/httpapi/config_test.go`
- **Verification:** both guards pass; the relaxed guard still fails if `config.go` ever imports `/sdk/` or references `Fetch`/`Match` directly.
- **Committed in:** `894ab20` (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (1 bug, 1 blocking compile dependency). Both necessary for correctness; no scope creep.

## Known Stubs

None — every route added in this plan is a real, working implementation exercised by its own tests; no hardcoded empty values or placeholder text were introduced.

## Issues Encountered

- `go list -m -f {{.Dir}}` under this repo's `go.work` (which joins six modules) returns one line per module when called without an explicit module argument — every test-time plugin-binary-build helper (pluginhost's, supervisor's, httpapi's) had to name `github.com/davison/topos` explicitly to resolve the repo root reliably. Fixed on first test run, not left as a flaky helper.
- No live `make dev` session was available in this execution environment to perform the plan's own `<verification>` section's manual "hand-edit config.toml, call the real reload route, confirm GET /api/sources with no restart" step against a genuinely running kernel. The equivalent behavior is proven at the HTTP-handler layer instead, against the real `chi.Router` and a real `config.Store` backed by a temp file (`kernel/httpapi/config_test.go`'s reload tests, `kernel/httpapi/live_config_test.go`), which exercises the identical code path a live kernel would run — the gap is the live-process/live-plugin-subprocess integration, not the logic.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The apply seam (`kernel/supervisor.Supervisor.Apply`), the reload route, and live-per-request config reads are exactly what 07-03's builder UI (webspace switcher, "+" chip picker, source config modals) needs underneath it — every save the UI makes will reach the running kernel immediately, and the picker's plugin-type list / match-vocabulary form now have a real kernel-side data source (`GET /api/config/plugin-types`, `POST /api/config/describe-plugin`).
- `docs/api.md` documents all three new routes and the three new error codes for the UI work to build against.
- No blockers identified for 07-03 through 07-05.

---
*Phase: 07-webspace-builder-ui*
*Completed: 2026-08-08*

## Self-Check: PASSED
