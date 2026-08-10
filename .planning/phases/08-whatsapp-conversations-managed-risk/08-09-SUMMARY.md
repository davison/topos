---
phase: 08-whatsapp-conversations-managed-risk
plan: 09
subsystem: sync-engine
tags: [go, grpc-go, go-plugin, supervisor, generation-contract, whatsapp-link]

# Dependency graph
requires:
  - phase: 08-whatsapp-conversations-managed-risk
    provides: SuspendInstance seam (D-03 Re-link), Apply's commitGeneration contract, the WhatsApp link session HTTP routes
provides:
  - A source instance suspended for a WhatsApp link session and then resumed syncs successfully on its very next refresh, through a real launched-then-killed-then-relaunched plugin subprocess, with no config save and no kernel restart
  - Every background sync the kernel dispatches (Apply's eager resync included) is bound to its own scheduler generation's context and wait group, so stopScheduler is never blocked by an uncancellable goroutine
  - Every WhatsApp link lifecycle call site (start's suspend, both its failure-path resumes, poll's terminal resume, cancel's resume) runs on a detached context.Background() that outlives the HTTP request
  - A refresh issued during a suspension window returns syncer.ErrUnknownSource before any sync_runs row is started, so a kernel lifecycle artifact is never pinned to a source's health surface as a failed sync
affects: [08-10-plan (owns the stream response's sync-status/copy half of G-08-3), sources.go's now-reachable 404 doc comment]

# Actuals (#2632)
actuals:
  tokens: 9252
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Any mutation of the launched plugin set (not only Apply) must go through stopScheduler -> Host.Reconcile -> commitGeneration in that exact order, so s.host/s.coord/s.cfg/the running generation always reflect one and the same generation"
    - "Every background sync dispatched outside Scheduler.Run's own goroutine set must derive its context from the generation's genCtx and register on its genWG before starting, so stopScheduler's wait bounds all work belonging to the generation it tears down"
    - "Every HTTP handler call that outlives, or can mutate state that outlives, the request runs on context.Background(), never r.Context()"

key-files:
  created: []
  modified:
    - kernel/supervisor/supervisor.go
    - kernel/supervisor/suspend_test.go
    - kernel/supervisor/supervisor_test.go
    - kernel/httpapi/whatsapplink.go
    - kernel/httpapi/whatsapplink_test.go

key-decisions:
  - "SuspendInstance and its resume closure now perform the exact stopScheduler -> Reconcile -> commitGeneration sequence Apply already uses, rather than giving syncer.Coordinator a live by-name resolution seam through pluginhost.Host (rejected per the plan's own prohibition: Host.plugins is an unsynchronised slice Reconcile mutates in place)"
  - "A suspend/resume Reconcile failure restarts the OLD (pre-suspend) generation via startScheduler, mirroring Apply's own pre-Reconcile failure branch — the kernel is never left with no scheduler running at all"
  - "Apply's eager-resync dispatch reads coord/genCtx/genWG together, right after commitGeneration, so dispatched goroutines belong to the generation that will own their cancellation, not a stale one captured earlier"

patterns-established:
  - "Pattern: any lifecycle mutation of the launched plugin set is a generation change, not a narrow side-mutation — this is now the second call site (after Apply) to prove the discipline out"

requirements-completed: [SRC-03]

coverage:
  - id: D1
    description: "A source instance suspended for a WhatsApp link session and then resumed syncs successfully on its next refresh, with no config save and no restart"
    requirement: "SRC-03"
    verification:
      - kind: unit
        ref: "kernel/supervisor/suspend_test.go#TestSuspendInstance_ResumedInstanceStillSyncs"
        status: pass
    human_judgment: false
  - id: D2
    description: "No sync the kernel dispatches (including Apply's eager resync) can outlive or block the generation that dispatched it"
    requirement: "SRC-03"
    verification:
      - kind: unit
        ref: "kernel/supervisor/supervisor_test.go#TestApply_EagerResyncDoesNotOutliveItsGeneration"
        status: pass
    human_judgment: false
  - id: D3
    description: "No suspension ever writes an errored sync run for the instance it suspended (a lifecycle artifact never pins a source's health surface)"
    requirement: "SRC-03"
    verification:
      - kind: unit
        ref: "kernel/supervisor/suspend_test.go#TestSuspendInstance_SuspendedWindowRecordsNoErroredRun"
        status: pass
    human_judgment: false
  - id: D4
    description: "A browser that disappears mid-session cannot abort the relaunch/generation rebuild — every lifecycle call site hands the supervisor a detached context"
    requirement: "SRC-03"
    verification:
      - kind: unit
        ref: "kernel/httpapi/whatsapplink_test.go#TestWhatsAppLink_SuspendAndResumeRunOnDetachedContexts"
        status: pass
    human_judgment: false
  - id: D5
    description: "Live-device UAT re-run of 08-UAT.md test 3 (opening the webspace after a real WhatsApp re-link no longer surfaces the grpc closing-connection error)"
    human_judgment: true
    rationale: "This plan replaces the original live repro with a hermetic Go test driving the identical path against real launched mock plugin subprocesses, per the plan's own assumptions section. The user-visible confirmation against a real device belongs to the phase's UAT re-run, not to this plan's automated verification."

# Metrics
duration: 55min
completed: 2026-08-11
status: complete
---

# Phase 08 Plan 09: Kernel-side generation-correct suspend/resume (G-08-3) Summary

**`supervisor.SuspendInstance` and its resume closure are now full generation changes (stopScheduler → Reconcile → commitGeneration), Apply's eager-resync goroutines are bound to their own generation's context/wait-group, and all five WhatsApp link lifecycle call sites run on `context.Background()` — closing the kernel half of 08-UAT.md's G-08-3 (`rpc error: code = Canceled desc = grpc: the client connection is closing` after a re-link).**

## Performance

- **Duration:** 55 min (continuation agent portion; total plan duration spans two agent sessions)
- **Started:** 2026-08-11T00:03:00Z (Task 1, prior agent)
- **Completed:** 2026-08-11T01:15:00Z
- **Tasks:** 3 completed
- **Files modified:** 5

## Accomplishments
- `SuspendInstance` and its resume closure rebuild the `syncer.Coordinator` over the live plugin handle on every suspend and resume, instead of leaving the coordinator dispatching against a killed go-plugin gRPC client until the next config save or kernel restart
- Every dispatched background sync (Apply's eager resync included) is now bound to its owning generation's cancellable context and wait group, so `stopScheduler` — now reachable from the WhatsApp link-start HTTP request path — can never block indefinitely
- All five lifecycle call sites in `whatsapplink.go` (start's suspend, its two failure-path resumes, poll's terminal resume, cancel's resume) hand the supervisor a detached `context.Background()`, matching the file's existing reaper/Shutdown discipline
- A refresh issued while an instance is suspended returns `syncer.ErrUnknownSource` before any `sync_runs` row is started, proven never to pin a source's health chip to a lifecycle artifact
- Four new/extended regression tests, each proven RED against its pre-fix shape before being left green

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end — a suspended-then-resumed instance still syncs** - `287a4bc` (fix)
2. **Task 2: Every dispatched sync is cancellable by the generation that owns it** - `5eae640` (fix)
3. **Task 3: Nothing a link session does is ever pinned to a source's health** - `b59aaa3` (test)

**Plan metadata:** (this commit, docs)

## Files Created/Modified
- `kernel/supervisor/supervisor.go` - `SuspendInstance`/resume closure now perform stopScheduler → Reconcile → commitGeneration; two new generation-scoped fields (`genCtx`, `genWG`); Apply's eager-resync dispatch derives its context/wait-group registration from the just-committed generation
- `kernel/supervisor/suspend_test.go` - `TestSuspendInstance_ResumedInstanceStillSyncs` (Task 1) and `TestSuspendInstance_SuspendedWindowRecordsNoErroredRun` (Task 3, Guard 1)
- `kernel/supervisor/supervisor_test.go` - `TestApply_EagerResyncDoesNotOutliveItsGeneration` (Task 2); `blockingSource` extended with an `exited` channel and guarding `sync.Once`
- `kernel/httpapi/whatsapplink.go` - all five lifecycle call sites detached to `context.Background()`; doc comment at the spawn site restated positively
- `kernel/httpapi/whatsapplink_test.go` - `TestWhatsAppLink_SuspendAndResumeRunOnDetachedContexts` (Task 3, Guard 2); `fakeSuspender` extended to record the context of every suspend/resume call

## Decisions Made
- Took the generation-contract branch (`stopScheduler` → `Reconcile` → `commitGeneration`), not a live by-name resolution seam through `pluginhost.Host` — the plan's own prohibition rules the latter out (`Host.plugins` is an unsynchronised slice `Reconcile` mutates in place; resolving through it from scheduler goroutines would trade a stale-handle bug for a data race)
- A suspend/resume `Reconcile` failure restarts the OLD generation (mirrors `Apply`'s own pre-Reconcile failure branch) — never leaves the kernel with no scheduler running
- Apply's eager-resync dispatch re-reads `coord`/`genCtx`/`genWG` immediately after `commitGeneration`, not before — those fields belong to the NEW generation `commitGeneration` just installed

## Deviations from Plan

### Auto-fixed / Documented Issues

**1. [Rule 3 - Blocking] Task 2's literal `<verify>` command is self-contradictory**
- **Found during:** Task 2 verification (continuation agent)
- **Issue:** The plan's `<verify>` for Task 2 reads `CGO_ENABLED=0 go test ./kernel/supervisor/... -count=1 -race` — Go's race detector requires cgo, so `CGO_ENABLED=0` with `-race` fails immediately with `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1`, never actually running the race check the task exists to prove.
- **Fix:** Ran the intended check with `CGO_ENABLED=1 go test ./kernel/supervisor/... -count=1 -race` (gcc is present in this environment). No race detected.
- **Files modified:** None (verification-only).
- **Verification:** `ok github.com/davison/topos/kernel/supervisor 2.434s`, no race report.
- **Committed in:** N/A (pre-existing commit 5eae640; no code change needed).

**2. [Rule 3 - Blocking] Task 3's uncommitted state on resume was Guard 1, not Task 2 verification**
- **Found during:** Continuation start
- **Issue:** The prior agent's interruption note said "verify this Task 2 state" but the actual uncommitted diff in `suspend_test.go` was `TestSuspendInstance_SuspendedWindowRecordsNoErroredRun`, whose own doc comment identifies it as "Task 3's Guard 1 (08-09-PLAN.md)". Task 2's own files (`supervisor.go`, `supervisor_test.go`) were already fully committed at `5eae640` and needed only verification, not further edits.
- **Fix:** Verified Task 2's commit independently (build + race test, see deviation 1), then continued Task 3 from the correct point: completed Guard 1 (already present, unverified) and wrote Guard 2 (`TestWhatsAppLink_SuspendAndResumeRunOnDetachedContexts`) from scratch, extending `fakeSuspender` to record contexts.
- **Files modified:** `kernel/httpapi/whatsapplink_test.go` (Guard 2, new), `kernel/supervisor/suspend_test.go` (Guard 1, already present).
- **Verification:** Both guards pass; both proven RED (see below).
- **Committed in:** `b59aaa3`.

**3. [Rule 3 - Blocking / documented, not fixed] `npm --prefix web run test -- --run` cannot run in this worktree**
- **Found during:** Plan-level verification
- **Issue:** The plan's own `<verification>` requires `npm --prefix web run test -- --run` to still pass untouched. `web/node_modules` is not installed in this worktree (`sh: line 1: vitest: command not found`) — a pre-existing environment condition, not caused by this plan (no web file is in `files_modified` or was touched).
- **Fix:** Not fixed — installing frontend dependencies is outside this plan's scope (Go-only `files_modified`) and outside the deviation rules' auto-fix authority for an unrelated toolchain. Documented here per the plan's own instruction: "any failure here is an unrelated regression and must be reported, not absorbed."
- **Files modified:** None.
- **Verification:** N/A — command could not run.
- **Committed in:** N/A.

**4. [Documented, not fixed] Pre-existing flaky test under `-race`, unrelated to this plan's files**
- **Found during:** Full-suite `-race` sanity pass after Task 3
- **Issue:** `TestApply_MidFlightSyncLeavesNoStrandedRunningRow` (in `supervisor_test.go`, untouched by this plan — confirmed via `git diff 5eae640 HEAD -- kernel/supervisor/supervisor_test.go` showing zero diff for that hunk) fails intermittently under `-race` (observed 1-in-5 in a local repeat run). The test's own doc comment already documents this as "a pre-existing race independent of 07-09's fix (reproduces identically against unmodified supervisor.go under -race), never previously guarded against" — i.e., a known, pre-existing, out-of-scope flake this plan's `must_haves.prohibitions` explicitly forbids widening scope to fix (the Reconcile-failure branch's restart "is never touched by this plan").
- **Fix:** Not fixed — explicitly out of scope per the plan's own prohibitions. Flagged here for phase-level tracking.
- **Files modified:** None.
- **Verification:** Reproduced via 5 repeated runs of `CGO_ENABLED=1 go test ./kernel/supervisor/... -race -run TestApply_MidFlightSyncLeavesNoStrandedRunningRow -count=1 -v`; 1 failure, 4 passes.
- **Committed in:** N/A.

---

**Total deviations:** 4 (2 verification-command/process corrections, 2 documented-but-not-fixed pre-existing/out-of-scope conditions). **Impact on plan:** None affect the plan's own correctness or scope — all four are either verification-methodology corrections or explicitly out-of-scope pre-existing conditions surfaced for visibility.

## RED Demonstrations (quoted verbatim, all three tasks)

Each was reproduced by the continuation agent (the interrupting agent's own RED runs, if performed, were not recorded) by temporarily reverting the fix, running the test, capturing the failure, then restoring byte-identical (`git diff --stat` clean in every case) and re-running green.

**Task 1 — `TestSuspendInstance_ResumedInstanceStillSyncs`** (both `s.commitGeneration(s.cfg)` calls in `SuspendInstance`/resume temporarily removed):
```
suspend_test.go:290: expected Status "ok" for a resumed instance's next refresh, got "error" (error: "match against source \"suspend-me\": rpc error: code = Canceled desc = grpc: the client connection is closing") — a non-ok status here means the coordinator is still dispatching against the subprocess SuspendInstance already killed
```
This is the exact error string from `08-UAT.md`'s original G-08-3 report.

**Task 2 — `TestApply_EagerResyncDoesNotOutliveItsGeneration`** (the test's own dispatch mirror temporarily reverted to a detached, untracked `context.Background()` instead of `genCtx`/`genWG`):
```
supervisor_test.go:388: expected the dispatched resync's Match call to have returned (its context cancelled) by the time Shutdown completed
```
(Note: `Shutdown` itself returned promptly here because the untracked goroutine was never registered on `genWG` — the failure surfaces as the `blocker.exited` assertion, not the 2-second deadline, since an unregistered goroutine cannot block a wait it was never added to. This is itself informative: it demonstrates the goroutine truly was untracked, which is precisely the defect being guarded against.)

**Task 3 Guard 1 — `TestSuspendInstance_SuspendedWindowRecordsNoErroredRun`** (`SuspendInstance`'s `commitGeneration` call temporarily removed):
```
suspend_test.go:374: expected Refresh during the suspension window to return syncer.ErrUnknownSource, got: <nil>
```

**Task 3 Guard 2 — `TestWhatsAppLink_SuspendAndResumeRunOnDetachedContexts`** (all three of start's suspend call, poll's terminal resume, and cancel's resume temporarily reverted to `r.Context()`):
```
whatsapplink_test.go:595: expected the context handed to SuspendInstance to be detached from the request (no error), got: context canceled — a handler forwarding r.Context() would hand over an already-cancelled context here
whatsapplink_test.go:629: expected the context handed to resume from the poll handler to be detached from the request (no error), got: context canceled — a handler forwarding r.Context() would hand over an already-cancelled context here
whatsapplink_test.go:656: expected the context handed to resume from the cancel handler to be detached from the request (no error), got: context canceled — a handler forwarding r.Context() would hand over an already-cancelled context here
```

## Issues Encountered

The prior (interrupted) agent's narration mislabeled its own uncommitted work as "Task 2 verification" when it was actually the beginning of Task 3's Guard 1 test. Resolved by reading the uncommitted diff directly (its own doc comment self-identified as "Task 3's Guard 1"), verifying Task 2's actual commit independently, and continuing Task 3 correctly. See Deviation 2, above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The kernel half of `08-UAT.md` G-08-3 is closed: a re-link's `SuspendInstance`/resume path is generation-correct, background syncs are cancellable, and lifecycle calls are detached from the HTTP request.
- `08-10-PLAN.md` (already planned, same wave) owns the remaining UAT missing item: scoping/degrading the stream response's sync-status copy — the UI half of G-08-3.
- `kernel/httpapi/sources.go`'s doc comment claiming the suspension-window 404 branch is unreachable is now false (a suspended instance's refresh reaches it) — flagged in this plan's `<assumptions>` as belonging to `08-10-PLAN.md`, which already owns that file in this wave.
- The pre-existing `-race` flake in `TestApply_MidFlightSyncLeavesNoStrandedRunningRow` (Deviation 4) remains unfixed and out of scope; worth a dedicated follow-up if `-race` runs in CI need to be reliably green.

---
*Phase: 08-whatsapp-conversations-managed-risk*
*Completed: 2026-08-11*
