---
phase: 07-webspace-builder-ui
plan: 09
subsystem: kernel/supervisor
tags: [go, config-hot-apply, goroutine-generation-consistency, gap-closure]

# Dependency graph
requires:
  - phase: 07-webspace-builder-ui (07-02)
    provides: "Supervisor.Apply — the config.Store.Save/Reload hot-apply seam (D-06/D-07) this plan closes a defect in"
provides:
  - "Supervisor.commitGeneration — the single site every branch of Apply that adopts a new config generation goes through, in the one correct order (coordinator, then cfg, then scheduler)"
  - "A post-Reconcile Apply failure (match-vocabulary rejection, or either D-07 index-cleanup step) now adopts the new generation instead of restarting the scheduler against a config the host has already moved past — the coordinator always dispatches against the plugin subprocess that is actually alive"
  - "Behavioural test coverage for the Reconcile-succeeds/ValidateMatchConfig-fails ordering, and for retry idempotency, both proven against real launched plugin subprocesses"
  - "Apply's doc comment states the two-regime contract (pre-Reconcile vs post-Reconcile) the code actually implements"
affects: [08-whatsapp-conversations, any future phase touching kernel/supervisor or kernel/httpapi/config.go]

# Actuals (#2632)
actuals:
  tokens: 5195
  tasks: 2
  commits: 3

tech-stack:
  added: []
  patterns:
    - "commitGeneration(cfg) as the single shared commit site for a Go struct's related-fields invariant (coordinator/config/scheduler-generation) — every branch that adopts a new generation calls one method rather than hand-rolling the same three assignments in a caller-chosen order"

key-files:
  created: []
  modified:
    - kernel/supervisor/supervisor.go
    - kernel/supervisor/supervisor_test.go

key-decisions:
  - "Fix direction: roll forward (adopt the new generation) on every post-Reconcile failure, never roll back — Host.Reconcile provides no undo and a rollback that performs real subprocess launches can itself fail, per the plan's explicit prohibition"
  - "The two D-07 index-cleanup failure branches were found to have the same defect family (starting the scheduler before installing the rebuilt coordinator) — previously unreported, folded into this fix rather than deferred, since Task 1's own acceptance criterion (coordinator installed at exactly one site) is unsatisfiable without also fixing them"
  - "NewSupervisor's boot sequence deliberately keeps its own independent host/coord/cfg/startScheduler sequence rather than being refactored to call commitGeneration — pinned by this plan's own acceptance criterion that s.startScheduler( appears at exactly 3 call sites including NewSupervisor's boot call, which the plan's coordinate-count-1 wording for Task 1 does not account for (see Deviations)"
  - "Hardened the blockingSource test fixture (sync.Once-guarded channel close) after discovering it panics under -race on unmodified pre-07-09 code — a pre-existing, unrelated bug, fixed because it lives in a shared helper (not either name-pinned test's body) and blocks the plan's own mandated -race verification"

patterns-established:
  - "commitGeneration-style shared commit site for enforcing 'N related fields always change together, in one order' invariants across multiple failure branches of one function"

requirements-completed: [KERN-08, UI-12]

coverage:
  - id: D1
    description: "A rejected save at the post-launch match-vocabulary check no longer leaves the coordinator pointing at a killed plugin subprocess — the coordinator, cfg, and scheduler generation are rebuilt to match the host Reconcile already committed"
    requirement: "KERN-08"
    verification:
      - kind: unit
        ref: "kernel/supervisor/supervisor_test.go#TestApply_ValidateMatchConfigFailsAfterReconcile_CoordinatorTracksRelaunchedPlugin"
        status: pass
    human_judgment: false
  - id: D2
    description: "Retrying a rejected save (the documented POST /api/config/reload recovery path) is idempotent — churns no subprocesses on the second apply"
    requirement: "UI-12"
    verification:
      - kind: unit
        ref: "kernel/supervisor/supervisor_test.go#TestApply_RejectedSaveIsIdempotent_SecondApplyDoesNotRelaunchSubprocesses"
        status: pass
    human_judgment: false
  - id: D3
    description: "Live make-dev confirmation: editing a source's connection details and introducing an invalid match field name in the same UI save produces the 500 apply_failed response, and that source continues to sync and report healthy on its next scheduled tick rather than failing continuously until the kernel is restarted"
    verification: []
    human_judgment: true
    rationale: "Requires a running kernel, a live UI save through the webspace builder, and observing a real scheduled sync tick over time — genuinely requires human/live-environment judgment beyond what an automated Go test fixture can assert; deferred to end-of-phase UAT per workflow.human_verify_mode=end-of-phase"

duration: ~17min
completed: 2026-08-08
status: complete
---

# Phase 07 Plan 09: Supervisor.Apply Generation Consistency Summary

**Closed gaps[0]: `Supervisor.Apply` now routes every post-Reconcile failure branch (match-vocabulary rejection, and both D-07 index-cleanup steps) through one shared `commitGeneration` site, so a rejected save never leaves the coordinator dispatching against a plugin subprocess `Host.Reconcile` already killed.**

## Performance

- **Duration:** ~17 min
- **Started:** 2026-08-08T18:52:00Z (approx, from STATE.md's last recorded activity before this plan)
- **Completed:** 2026-08-08T19:09:09Z
- **Tasks:** 2
- **Files modified:** 2 (`kernel/supervisor/supervisor.go`, `kernel/supervisor/supervisor_test.go`)

## Accomplishments

- Added `Supervisor.commitGeneration(cfg)` — the single site every branch of `Apply` that adopts a new config generation goes through, in the load-bearing order (`s.coord` installed before `startScheduler` is called, since `startScheduler` captures `s.coord` into the `syncer.Scheduler` value at call time).
- The `ValidateMatchConfig` failure branch (this **is** gaps[0]) now adopts the new generation via `commitGeneration` instead of restarting the scheduler against the old config — `Host.Reconcile` has already committed its mutation in place by that point and provides no undo, and the config store already swapped to the new config before `Apply` was ever called, so the new generation is the only self-consistent state available. The branch still returns its error unchanged (500 `apply_failed`).
- Found and fixed a **previously unreported second instance of the same defect family**: the two D-07 index-cleanup failure branches (`DeleteSourceItems`, `DeleteSyncRuns`) were calling `s.startScheduler` **before** assigning the rebuilt coordinator, so the scheduler generation they launched captured the *previous* coordinator, not the one installed moments later. Routed through `commitGeneration`, fixing the ordering as a side effect of removing the duplication.
- Added two behavioural tests against real launched mock-plugin subprocesses: the Reconcile-succeeds/ValidateMatchConfig-fails ordering test 07-VERIFICATION.md named as absent, and a retry-idempotency test for the UI-12 edge.
- Rewrote `Apply`'s doc comment to state the two-regime contract the code actually implements (pre-Reconcile: old generation kept; post-Reconcile: new generation adopted, error still returned) — 07-VERIFICATION.md `gaps[0].reason` had cited the old comment as contradicting the code beneath it.
- Hardened a pre-existing, unrelated test fixture bug discovered during verification (see Deviations).

## Task Commits

1. **Task 1 (RED): add failing ordering test** — `d6506d1` (test)
2. **Task 1 (GREEN): route Apply's post-Reconcile failure branches through commitGeneration** — `68c328a` (feat)
3. **Task 2: correct Apply's doc comment** — `99cbfd4` (docs)

**Plan metadata:** (this commit, to follow)

## Files Created/Modified

- `kernel/supervisor/supervisor.go` — added `commitGeneration`; rerouted the `ValidateMatchConfig` failure branch, both D-07 index-cleanup failure branches, and the success path through it; rewrote `Apply`'s doc comment
- `kernel/supervisor/supervisor_test.go` — added `TestApply_ValidateMatchConfigFailsAfterReconcile_CoordinatorTracksRelaunchedPlugin`, `TestApply_RejectedSaveIsIdempotent_SecondApplyDoesNotRelaunchSubprocesses`, `pluginByName` helper; hardened `blockingSource.Match` against a pre-existing double-close race

## Decisions Made

- **Fix direction: roll forward (option (a)), never roll back (option (b))** — five grounds, as specified by the plan and confirmed unchanged during execution:
  1. *Totality*: `newCoordinator` is pure in-memory construction over the host's current plugin list and cannot fail; option (b)'s rollback would require a second `Reconcile` that performs real subprocess launches and can itself fail, leaving no third recourse.
  2. *The store has already swapped*: `config.Store.Save`/`Reload` swap their pointers on success, strictly before the handler calls `Apply` — the new config is already the config-of-record everywhere else in the kernel.
  3. *The documented recovery path only converges under (a)*: `POST /api/config/reload` re-reads the same still-invalid file; under (a) the retry's reconcile is a no-op and fails the same way (proven by `TestApply_RejectedSaveIsIdempotent_SecondApplyDoesNotRelaunchSubprocesses`); under (b) every retry would kill and relaunch the affected subprocesses twice, forever.
  4. *Precedent inside the same function*: the two D-07 index-cleanup branches already rolled forward before this fix — (a) makes all post-Reconcile branches uniform.
  5. *D-08 is satisfied upstream, not here*: the last-good-config guarantee is enforced in `config.Store.Reload`, which validates before swapping; a match-vocabulary rejection is a different class only a live post-launch check can catch, by which point no last-known-good exists at the store level either.
- **The D-07 branch ordering defect was folded into Task 1's commit, not deferred to Task 2 as the plan's task split literally implied.** Task 1's own acceptance criteria requires `grep -c 's.coord = newCoordinator('` to report exactly 1 after Task 1 alone — unsatisfiable while the D-07 branches still hand-roll their own `s.coord = newCoordinator(...)` assignment. Task 1's commit therefore includes routing all four commit-adopting branches (vocabulary, both D-07 steps, success path) through `commitGeneration`; Task 2's commit is a pure doc-comment rewrite (verified via `git diff` to touch comment lines only) plus one added comment on the D-07 loop, matching Task 2's own acceptance criteria (`git diff` shows no changed error strings; `git diff --stat` for that commit lists only `supervisor.go`).
- **`NewSupervisor`'s boot sequence intentionally keeps its own independent `host`/`coord`/`cfg`/`startScheduler` sequence rather than being refactored to call `commitGeneration`.** This is required by the plan's own Task 2 acceptance criterion that `grep -c 's.startScheduler('` reports exactly 3, explicitly naming "NewSupervisor's boot call" as one of the three distinct call sites. As a direct consequence, `grep -c 's.coord = newCoordinator('` reports 2 (once in `commitGeneration`, once in `NewSupervisor`) and `grep -c 's.cfg = '` reports 2 (same split) — not the literal "exactly 1" Task 1's acceptance criteria states. This is a self-inconsistency in the plan's own numeric criteria (the "exactly 1" wording implicitly assumed `NewSupervisor` would also route through `commitGeneration`, which the more specific and more heavily cross-referenced Task 2 criterion explicitly forbids). Resolved in favor of the Task 2 criterion, since it explicitly and unambiguously names the third call site by role.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Hardened `blockingSource.Match` against a pre-existing double-close panic under `-race`**
- **Found during:** Task 1, first `-race` run of the full verification command
- **Issue:** `TestApply_MidFlightSyncLeavesNoStrandedRunningRow`'s `blockingSource` fixture unconditionally called `close(b.entered)` in `Match`. That test's own scenario forces `Host.Reconcile` to fail, hitting the pre-Reconcile failure branch, which (correctly, unchanged by this plan) restarts the scheduler against the old config — and that restarted generation's own immediate first refresh calls `Match` on the SAME shared fixture a second time, panicking on the already-closed channel. **Confirmed via `git stash` to reproduce identically on unmodified pre-07-09 `supervisor.go`, 5/5 runs** — entirely unrelated to gaps[0] and pre-existing since whenever this test was added (07-02).
- **Fix:** Guarded the channel close with `sync.Once` (`closeEnteredOnce`). The second `Match` call now blocks on its own context instead of panicking; that goroutine is a harmless leak for the remainder of the test binary's process.
- **Files modified:** `kernel/supervisor/supervisor_test.go` (only the `blockingSource` type/methods, which sit between — not inside — the two plan-protected test functions; verified via `git diff` that neither `TestApply_MidFlightSyncLeavesNoStrandedRunningRow` nor `TestApply_RemovedInstance_PluginGoneAndIndexRowsGone`'s own bodies changed a single byte)
- **Verification:** `go test ./kernel/supervisor/... -race` no longer panics; ran clean 3+ consecutive times after the fix (previously panicked 5/5 on the unmodified fixture)
- **Committed in:** `68c328a` (Task 1's GREEN commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 — pre-existing bug blocking mandated verification)
**Impact on plan:** Necessary to satisfy the plan's own `-race` verification requirement. No scope creep — confined to a shared test helper, does not touch either name-pinned test's assertions or body.

## Issues Encountered

**A second, separate pre-existing flake was discovered and left unfixed (out of scope).** `TestApply_MidFlightSyncLeavesNoStrandedRunningRow` itself has an intermittent (~1-in-6 to 1-in-8 observed, load-dependent) race between `Coordinator.syncOne`'s detached `sync_runs` finalize write (`kernel/syncer/coordinator.go`) and the test's own synchronous read of `LatestSyncRunPerSource` immediately after `Apply` returns. **Confirmed to reproduce identically on unmodified pre-07-09 `supervisor.go`** (same failure signature: `Status: running`, `FinishedUnix: 0`), so it is unrelated to this plan's fix. It was **not fixed** here because:
1. It lives inside `TestApply_MidFlightSyncLeavesNoStrandedRunningRow`'s own body, which this plan's `must_haves.prohibitions` explicitly forbids editing ("byte-identical to their prior content").
2. A correct fix belongs in `kernel/syncer/coordinator.go` or the test's own wait logic — outside this plan's declared `files_modified` (`kernel/supervisor/supervisor.go`, `kernel/supervisor/supervisor_test.go`'s test-function bodies).

Recorded in `.planning/WINDOWS.md` (entry id 5, kind `deviation`) for cross-phase visibility. All `go test ./kernel/... -count=1 -race` runs performed for this plan's own commits (immediately before each commit) passed clean; the flake was observed only during repeated stress-verification runs afterward, and reproduces at the same rate on the pre-fix baseline.

## Verification Results

- `CGO_ENABLED=0 go build ./...` — exits 0
- `go vet ./kernel/...` — exits 0
- `go test ./kernel/... -count=1 -race` — all packages `ok` (in the runs performed immediately before each commit; a pre-existing, unrelated intermittent flake documented above and in WINDOWS.md)
- `go test ./kernel/supervisor/... -run 'TestApply' -count=1 -v` — all four `TestApply_*` tests pass (`TestApply_RemovedInstance_PluginGoneAndIndexRowsGone`, `TestApply_MidFlightSyncLeavesNoStrandedRunningRow`, `TestApply_ValidateMatchConfigFailsAfterReconcile_CoordinatorTracksRelaunchedPlugin`, `TestApply_RejectedSaveIsIdempotent_SecondApplyDoesNotRelaunchSubprocesses`)
- `git diff --stat go.mod go.sum` — no output (no dependency added)
- `git diff --stat` across this plan's three commits — lists only `kernel/supervisor/supervisor.go` and `kernel/supervisor/supervisor_test.go`
- `grep -riE 'TODO|FIXME|XXX|HACK|PLACEHOLDER' kernel/supervisor/` — no matches
- **Revert-and-confirm (required by acceptance criteria):** temporarily reverted the `ValidateMatchConfig` failure branch back to `s.startScheduler(oldCfg)` (Task 1's pre-fix shape) and re-ran `TestApply_ValidateMatchConfigFailsAfterReconcile_CoordinatorTracksRelaunchedPlugin`. It failed exactly as expected, on the coordinator/cfg/Refresh assertions:
  ```
  supervisor_test.go:402: expected the coordinator to have been rebuilt (a different pointer) even though Apply returned an error
  supervisor_test.go:406: expected the supervisor's own recorded config generation (s.cfg) to be the same pointer cfgStore.Expanded() now returns
  supervisor_test.go:417: gaps[0]: expected Refresh's Status to be "ok" after a rejected apply — ... Got: {Source:changing SourceType:mock Status:error ... Error:match against source "changing": rpc error: code = Canceled desc = grpc: the client connection is closing Coalesced:true ...}
  ```
  Confirms the test is load-bearing, not vacuous. The fix was then restored (verified byte-identical to the committed version via `diff`).

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- 07-VERIFICATION.md `gaps[0]` (the single Critical gap blocking phase completion) is closed. The hot-apply seam (D-06) now keeps `s.host`, `s.coord`, `s.cfg` and the running scheduler generation on one self-consistent config generation across every branch of `Apply`, including every failure path.
- The live `make dev` human-check (D3 in the coverage table above) remains outstanding — a genuine UAT item, not plannable code work, deferred per `workflow.human_verify_mode: end-of-phase`.
- The 30 non-backstop `behavior_unverified_items` and the two Warning-severity findings (WR-01, WR-02) carried in 07-VERIFICATION.md remain open, as scoped — this plan deliberately did not touch `web/src/`.
- One new out-of-scope pre-existing flaky test discovered (`TestApply_MidFlightSyncLeavesNoStrandedRunningRow`'s detached-finalize race) — logged in `.planning/WINDOWS.md`, recommend a follow-up `/gsd-code-review 7 --fix` or dedicated fix pass targeting `kernel/syncer/coordinator.go`'s detached finalize / the test's own wait logic.

---
*Phase: 07-webspace-builder-ui*
*Completed: 2026-08-08*

## Self-Check: PASSED

- FOUND: kernel/supervisor/supervisor.go
- FOUND: kernel/supervisor/supervisor_test.go
- FOUND: .planning/phases/07-webspace-builder-ui/07-09-SUMMARY.md
- FOUND commit: d6506d1
- FOUND commit: 68c328a
- FOUND commit: 99cbfd4
