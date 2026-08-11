---
status: resolved
trigger: "Intermittent failure of TestApply_MidFlightSyncLeavesNoStrandedRunningRow under -race after phase 08 gap-closure wave 7 (plans 08-13/08-14, gap G-08-5) merged"
created: 2026-08-11T09:20:00Z
updated: 2026-08-11T11:30:00Z
---

## Symptoms

DATA_START
expected: TestApply_MidFlightSyncLeavesNoStrandedRunningRow (kernel/supervisor/supervisor_test.go:247) passes consistently under the race detector. The test asserts that Apply's stopScheduler call cancels the OLD scheduler generation's context and waits for its Run to fully return BEFORE Apply returns, so a mid-flight sync is always finalised (status "ok"/"error"/"cancelled" with a FinishedUnix timestamp), never left stranded at status "running".

actual: Intermittent failure (~30% of iterations): the mid-flight sync run row is still {Status:running FinishedUnix:0} at the moment Apply returns. Fails under full-package -race contention; passes in isolation (-run TestApply_MidFlightSyncLeavesNoStrandedRunningRow -count=3 -race → 3/3 pass).

errors: |
  --- FAIL: TestApply_MidFlightSyncLeavesNoStrandedRunningRow (0.03s)
      supervisor_test.go:294: expected the mid-flight sync to be finalised (not left at status "running") by the time Apply returns, got: {Source:slow StartedUnix:1786439550 FinishedUnix:0 Status:running Error: ItemCount:0}
      supervisor_test.go:297: expected the mid-flight sync run to carry a finished time, got: {Source:slow StartedUnix:1786439550 FinishedUnix:0 Status:running Error: ItemCount:0}

timeline: First observed 2026-08-11 during phase 08 plan 08-15's automated gate run, immediately after plans 08-13 and 08-14 (gap G-08-5, mutex-hold regression closure) merged to main at commit ae286a2. Plan 08-13 split kernel/supervisor/supervisor.go's supervisor lock into a new genMu sync.RWMutex (Host()/Coordinator() reader-only, Reconcile launches with no lock held) and added an internal RWMutex + snapshot() to kernel/pluginhost/host.go. The failing test's guarantee is exactly the concurrency contract that refactor touched. Not known to fail before this wave (the same gate was green in prior waves).

reproduction: |
  cd /home/darren/projects/davison/topos
  CGO_ENABLED=1 go test ./kernel/supervisor/ -count=10 -race
  # → 3/10 iterations FAIL with the assertion above (reproduced 2026-08-11).
  # Full-suite form (how the gate found it): CGO_ENABLED=1 go test ./kernel/... -count=1 -race
  # Note: -race requires CGO_ENABLED=1 (toolchain constraint; the plan's literal CGO_ENABLED=0 form is invalid).

context: |
  - Recent commits that changed the implicated code: ab732f6 "feat(08-13): kernel structurally guarantees a slow plugin relaunch cannot freeze other sources" (genMu split in kernel/supervisor/supervisor.go; Host.mu + snapshot() in kernel/pluginhost/host.go; new kernel/supervisor/launchlatency_test.go), and 08-14's plugins/whatsapp/connect.go change (login wait moved to a goroutine — plugin-side, less likely implicated in a kernel/supervisor unit test).
  - Constraint from plan 08-13's objective: a slow plugin relaunch must NOT freeze other sources' routes — any fix must preserve that (kernel/supervisor/launchlatency_test.go TestResume_SlowRelaunchDoesNotFreezeOtherSources pins it).
  - This finding blocks phase 08 plan 08-15 (final gate): gates must be green before the real-device human check runs.
  - Test doc comment (supervisor_test.go ~line 247): "Apply's stopScheduler call must cancel the OLD scheduler generation's context and wait for its Run to fully return BEFORE anything else happens, so a mid-flight sync is always finalised... regardless of whether Reconcile itself goes on to succeed."
DATA_END

## Current Focus

bug_class: "Heisenbug (transient, contention-dependent scheduling) — but with a Bohrbug core: the underlying defect is a deterministic test-oracle error whose OBSERVABILITY is timing-dependent. No production concurrency defect exists."

reasoning_checkpoint:
  hypothesis: "The mid-flight sync IS correctly finalised before Apply returns. The test fails because it asserts through idx.LatestSyncRunPerSource, a MAX(id)-per-source aggregate, while Apply's pre-Reconcile failure branch has ALREADY started a new scheduler generation (startScheduler(oldCfg)) whose immediate first refresh inserts a SECOND sync_runs row for the same source at status running. When that second row's INSERT wins the race against the test's read, the aggregate returns the new in-flight run instead of the mid-flight one, and the assertion inspects the wrong row."
  confirming_evidence:
    - "Direct row dump immediately after Apply returned (probe test, 8/8 iterations): count=2 rows for slow — id=1 status=error finished=<set> (the mid-flight run, correctly finalised) and id=2 status=running finished=NULL (a NEW run)."
    - "LatestSyncRunPerSource returned exactly the id=2 row: {Status:running FinishedUnix:0} — the same value the failing assertion reports."
    - "kernel/index/store.go:777 — WHERE id IN (SELECT MAX(id) FROM sync_runs GROUP BY source): latest-row semantics, cannot address an earlier run."
    - "kernel/supervisor/supervisor.go:667 — Apply's pre-Reconcile failure branch calls s.startScheduler(oldCfg) BEFORE returning; syncer.Scheduler.Run (scheduler.go:74-101) fires every configured source's first refresh immediately, and syncOne (coordinator.go:141) inserts the running row before Match is ever called."
    - "Ordering is deterministic in the code path: syncOne finalises synchronously on a detached context (coordinator.go:191-194) -> refreshAndLog -> firstRefresh -> runSource -> wg.Done -> Run returns -> close(done) -> stopScheduler's <-s.done unblocks -> Apply returns. The finalize strictly happens-before Apply returns."
    - "Both rows carry the same started_unix (same wall-clock second), so no existing field on index.SyncRun can distinguish them."
  falsification_test: "If the hypothesis were wrong, the row dump at the moment of failure would show a SINGLE row for slow left at status running (a genuinely stranded mid-flight run). It showed two rows, the first finalised."
  fix_rationale: "The production ordering guarantee holds and needs no change. The fix corrects the test's oracle so it addresses the specific run that was in flight when Apply was called — the FIRST sync_runs row for the source, deterministically id-ordered and inserted before Apply was ever entered — instead of a latest-row aggregate that a legitimate subsequent run can supersede. This makes the assertion deterministic without weakening it: a genuinely stranded mid-flight run still fails it."
  blind_spots:
    - "Adds a read method to kernel/index (SyncRunsForSource) that has no production caller yet — justified as the per-source history primitive, but it is new API surface introduced by a test need."
    - "Sibling tests in this package may leak the same forever-blocked generation-2 goroutine; only this test is being repaired."
  candidate_causes:
    - "code (test): oracle uses a latest-row aggregate to assert about a specific earlier run — CONFIRMED"
    - "code (production): Apply's failure-branch scheduler restart inserts a second running row — CONFIRMED as a contributing condition, but correct and intended behaviour, not a defect"
    - "data (fixture): blockingSource.Match blocks on ctx.Done() and is shared across generations, so generation 2's run stays running indefinitely and leaks a goroutine — CONFIRMED contributing condition and the amplifier"
    - "environment: goroutine scheduling under -race/-count decides whether the second INSERT beats the read — trigger, not cause"
  and_gate: "yes — the failure requires BOTH (a) the oracle reading a latest-row aggregate AND (b) a second, indefinitely-running sync_runs row existing for the same source. Either alone is benign: with a precise oracle the extra row is irrelevant, and with a fixture whose second Match returned promptly the aggregate would show a finalised row."

test: "Rewrite the assertion to address the mid-flight run directly (first row for the source, oldest-first), then stress the repaired test at -count=50 -race in isolation and at full-package -count=5 -race."
expecting: "Zero failures across every iteration, versus a ~25%/iteration failure rate before the change. Negative control: reverting only the production-side ordering (not applicable here) is replaced by asserting the repaired test still fails if the mid-flight run is left running."
next_action: "NONE — session resolved. Rename applied, all gates re-run green, session archived to .planning/debug/resolved/ and knowledge base updated."

checkpoint_response:
  received: 2026-08-11T11:05:00Z
  fix_approval: "APPROVED — accept the staged fix (test-oracle repair + t.Cleanup(s.Shutdown) + index tests)"
  api_change_requested: "Rename SyncRunsForSource to a ForTesting-suffixed variant (SyncRunsForSourceForTesting), following config.NewStoreForTesting. Closes the blind spot recorded above: new production API surface introduced by a test need is now explicitly marked test-only."

## Evidence

- timestamp: 2026-08-11T09:10:00Z
  checked: "Gate run: CGO_ENABLED=1 go test ./kernel/... -count=1 -race (executor agent, plan 08-15 Task 1)"
  found: "TestApply_MidFlightSyncLeavesNoStrandedRunningRow failed once; isolated -run with -count=3 passed 3/3; immediate full-suite rerun passed"
  implication: "Failure is contention-dependent, not deterministic"

- timestamp: 2026-08-11T09:18:00Z
  checked: "Orchestrator stress run: CGO_ENABLED=1 go test ./kernel/supervisor/ -count=10 -race"
  found: "3/10 iterations failed with identical assertion (row left Status:running, FinishedUnix:0)"
  implication: "Reliably reproducible at package level ~30% per iteration; genuine residual race in the 08-13 refactor area, not an environmental flake"

- timestamp: 2026-08-11T10:20:00Z
  checked: "Knowledge base (.planning/debug/knowledge-base.md) semantic match on symptoms"
  found: "KB-001 (stranded running row from a cancelled finalise) and KB-002 (any-row vs latest-row query semantics) both keyword-match. KB-001's fix shape (context.WithoutCancel detached finalise) is ALREADY present at kernel/syncer/coordinator.go:191. KB-002's lesson — a latest-row aggregate answers a different question than a per-run one — is the live lead."
  implication: "Treat KB-002's class as the first hypothesis to test: the assertion may be reading the wrong ROW rather than observing a stranded write"

- timestamp: 2026-08-11T10:25:00Z
  checked: "Isolated stress at HEAD (ae286a2): CGO_ENABLED=1 go test ./kernel/supervisor/ -run TestApply_MidFlightSyncLeavesNoStrandedRunningRow -count=20 -race"
  found: "5/20 iterations FAIL — the failure reproduces in isolation once the iteration count is raised (the earlier -count=3 isolated pass was simply under-sampled)"
  implication: "No full-package contention is needed; per-iteration failure rate is ~25%. Cheap, fast reproduction available for verification."

- timestamp: 2026-08-11T10:35:00Z
  checked: "Temporary probe test (kernel/supervisor/zz_probe_test.go, since deleted) replicating the test exactly but dumping every sync_runs row via a raw sqlite read immediately after Apply returned; 8 iterations under -race"
  found: |
    Identical result 8/8:
      count=2
      id=1 source=slow started=<T> finished={<T> true}  status=error
      id=2 source=slow started=<T> finished={0 false}   status=running
    LatestSyncRunPerSource[slow] = {Status:running FinishedUnix:0} — i.e. exactly the id=2 row.
  implication: |
    DECISIVE. The mid-flight run (id=1) IS finalised before Apply returns — the production ordering
    guarantee (stopScheduler waits for the old generation's Run to return, and syncOne's detached
    finalise is synchronous within it) holds exactly as documented. Row id=2 is a NEW run started by
    Apply's own pre-Reconcile failure branch (supervisor.go:667 s.startScheduler(oldCfg)), whose
    Scheduler.Run fires an immediate first refresh for "slow"; syncOne inserts its running row before
    Match is called, and blockingSource.Match then blocks forever on the uncancelled generation-2
    context so the row never finalises. LatestSyncRunPerSource (MAX(id) GROUP BY source,
    kernel/index/store.go:777) therefore returns id=2. The test is inspecting the wrong row.

- timestamp: 2026-08-11T10:40:00Z
  checked: "Same test at the pre-wave-7 baseline via git worktree at 98f8723 (= ab732f6^, before the 08-13 genMu split AND before 08-14): -count=20 -race"
  found: "Fails identically (2/20 in that run) with byte-identical assertion output"
  implication: "08-13/08-14 did NOT introduce this. The defect is latent in the test as written and predates phase 08's wave 7 entirely."

- timestamp: 2026-08-11T10:45:00Z
  checked: "git log --diff-filter=A -S'TestApply_MidFlightSyncLeavesNoStrandedRunningRow'"
  found: "Test introduced 2026-08-08 in f25c4ab (feat(07-02): apply on save). The latest-row assertion has been wrong since the day it was written."
  implication: "Amplification, not introduction, explains the timeline: each failing/passing iteration leaks one generation-2 goroutine blocked forever in Match plus an open index DB, so contention — and the failure rate — climbs with -count and with full-package parallelism. That is why the gate only started tripping recently."

- timestamp: 2026-08-11T10:50:00Z
  checked: "index.SyncRun struct (kernel/index/store.go:674) and index.Open (store.go:35) for an existing way to address one specific run"
  found: "SyncRun carries no id; both rows share the same started_unix (same wall-clock second) so no field distinguishes them. index.Open sets no WAL/busy_timeout, so a second sqlite connection from the test is not a safe read path. Precedent for a test-facing production helper exists: config.NewStoreForTesting."
  implication: "A precise oracle needs a small oldest-first per-source read on index.Store; a raw second connection from the supervisor test is rejected as SQLITE_BUSY-prone."

## Eliminated

- hypothesis: "08-13's genMu reader/writer lock split broke the synchronous ordering between stopScheduler's wait-for-Run-return and Apply's return path, so the old generation's finalisation write can be skipped or overlapped"
  evidence: "The row dump shows the mid-flight run (id=1) finalised with a finished_unix and status=error in 8/8 observations taken immediately after Apply returned — the ordering holds. Independently, the same failure reproduces at 98f8723, which predates the genMu split entirely."
  timestamp: 2026-08-11T10:40:00Z

- hypothesis: "KB-001 recurrence — the mid-flight sync's finalising UPDATE is being rejected with context canceled, stranding the row"
  evidence: "kernel/syncer/coordinator.go:191 already finalises on context.WithTimeout(context.WithoutCancel(ctx), finishRunTimeout), and the dump shows the row genuinely finalised. No stranded write exists."
  timestamp: 2026-08-11T10:35:00Z

- hypothesis: "Environmental flake / race-detector artifact rather than a real defect"
  evidence: "Deterministically explained by a two-row sequence reproduced 8/8 with instrumentation, and reproducible at a fixed ~25%/iteration rate across two different commits."
  timestamp: 2026-08-11T10:25:00Z

## Resolution

root_cause: |
  Two conditions that only fail together (AND-gate):

  (1) TEST ORACLE (primary): TestApply_MidFlightSyncLeavesNoStrandedRunningRow asserts through
      idx.LatestSyncRunPerSource, whose SQL is `WHERE id IN (SELECT MAX(id) FROM sync_runs GROUP BY
      source)` — a latest-row aggregate. It answers "what is this source's CURRENT run?", not "was
      the run that was in flight when Apply was called finalised?", which is the invariant the test
      exists to pin. Any later run for the same source silently supersedes the run under test.

  (2) A SECOND, INDEFINITELY-RUNNING ROW: Apply's pre-Reconcile failure branch
      (kernel/supervisor/supervisor.go:667) correctly restarts the OLD generation via
      startScheduler(oldCfg) BEFORE returning; syncer.Scheduler.Run fires every configured source's
      first refresh immediately, and Coordinator.syncOne inserts that run's "running" row before
      Match is ever called. The test's shared blockingSource fixture then blocks forever on the new,
      uncancelled generation's context, so the row never finalises and also leaks a goroutine.

  When (2)'s INSERT wins the scheduling race against the test's read — ~25% of iterations, rising
  with -count and full-package parallelism because each leaked goroutine adds contention — (1)
  returns that new running row and the assertion reports it as a stranded mid-flight sync.

  The production invariant under test is NOT violated: stopScheduler cancels the old generation and
  blocks on <-s.done until Scheduler.Run returns, and Coordinator.syncOne's detached finalise
  (context.WithoutCancel, coordinator.go:191) completes synchronously inside that wait — so the
  mid-flight run is always finalised strictly before Apply returns. Confirmed by direct row dump.
fix: |
  Corrected the test's oracle so it addresses the specific run under test; production behaviour
  unchanged (no production defect was found).

  1. kernel/index/store.go — new Store.SyncRunsForSourceForTesting(ctx, source): every sync_runs row
     for one instance id, ORDER BY id (monotonic in insertion order), oldest first. The per-run read
     the latest-row aggregates could not express. Documented against KB-002's distinction.
     NAMED FOR TEST-ONLY SURFACE at the user's direction on the human-verify checkpoint, following
     the existing config.NewStoreForTesting precedent: no production code path needs a source's full
     run history today, so the ForTesting suffix keeps it out of the supported read API until a real
     production caller appears. This closes the blind spot recorded in the reasoning checkpoint
     ("adds a read method that has no production caller yet"). The doc comment states the test-only
     status and retains the KB-002 per-run-vs-latest-row rationale in full.
  2. kernel/supervisor/supervisor_test.go — TestApply_MidFlightSyncLeavesNoStrandedRunningRow now
     asserts on runs[0] from SyncRunsForSourceForTesting (deterministically the run that was in
     flight when Apply was called — blocker.entered already proved it had started before Apply)
     instead of LatestSyncRunPerSource["slow"]. Doc comment records why the source's LATEST run is
     the wrong row here, so the aggregate is not reintroduced.
  3. kernel/supervisor/supervisor_test.go — added t.Cleanup(s.Shutdown), stopping the generation
     Apply restarts. That removes the leaked forever-blocked Match goroutine and open index handle
     per iteration, which was the amplifier that made the flake worsen with -count and under
     full-package parallelism.
  4. kernel/index/store_test.go — two tests for the new reader: ordering/scoping with the empty and
     singleton boundaries, and the earlier-finished-plus-later-running fixture that is the ONLY
     arrangement where SyncRunsForSourceForTesting and LatestSyncRunPerSource disagree (KB-002's
     stale-row lesson) — without it the two readers are substitutable undetected. Test function
     names were renamed alongside the method so no identifier refers to a name that no longer
     exists.

verification: |
  oracle_type: derived (contract) — the assertion is derived from Apply's documented ordering
  contract (stopScheduler cancels and awaits the old generation's Run before Apply returns), not
  from an implicit crash/no-error signal.

  guardrail_verdict: accepted

  signal 1 — original repro now passes:
    CGO_ENABLED=1 go test ./kernel/supervisor/ -run TestApply_MidFlightSyncLeavesNoStrandedRunningRow
      -count=50 -race  -> ok, 0 failures (pre-fix rate was 5/20 = ~25%/iteration)
    CGO_ENABLED=1 go test ./kernel/supervisor/ -count=10 -race  -> ok 72.198s, 0 failures
      (this is the exact command that previously produced 3/10)

  signal 2 — bug returns without the fix: YES. The pre-fix assertion fails at HEAD (5/20) and at the
    pre-wave-7 baseline 98f8723 (2/20), byte-identical output.

  signal 3 — mutation at the fix site (does the repaired test still bite?):
    M1: kernel/syncer/coordinator.go finishCtx built from ctx instead of context.WithoutCancel(ctx)
        (reverting the KB-001 detached finalise) -> KILLED, 10/10 iterations FAIL.
    M2: kernel/supervisor/supervisor.go stopScheduler's `<-s.done` wait removed
        (the ordering guarantee itself) -> KILLED, 5/10 iterations FAIL.
    Both mutants reverted; git status confirms clean.

  signal 4 — not a deletion-only diff: +201/-7 across four files; the oracle is replaced with a
    stricter one and new coverage is added.

  signal 5 — regression:
    CGO_ENABLED=1 go test ./kernel/... -count=1 -race -> all packages ok
    make test-portable -> all modules ok (kernel, sdk, paperless, silverbullet, proton, mock,
      mockstrict, whatsapp)
    go vet ./kernel/... -> clean; gofmt -l on all changed files -> clean
    golangci-lint not installed in this environment (not run)

  POST-RENAME RE-RUN (2026-08-11T11:30Z, after the checkpoint-requested
  SyncRunsForSource -> SyncRunsForSourceForTesting rename). The rename is a pure identifier change
  with no behavioural delta, so the guardrail verdict above stands; these gates confirm it:
    CGO_ENABLED=1 go test ./kernel/index/... ./kernel/supervisor/... -count=1 -race
      -> ok (index 2.110s, supervisor 8.604s)
    CGO_ENABLED=1 go test ./kernel/supervisor/ -count=10 -race -> ok 72.369s, 0 failures
      (again the exact command that previously produced 3/10)
    CGO_ENABLED=1 go test ./kernel/supervisor/ -run TestApply_MidFlightSyncLeavesNoStrandedRunningRow
      -count=50 -race -> ok, 0 failures
    CGO_ENABLED=1 go test ./kernel/... -count=1 -race -> all packages ok
    CGO_ENABLED=1 go test ./kernel/index/ -run TestSyncRunsForSourceForTesting -v -count=1 -race
      -> both renamed tests confirmed RUN and PASS (guards against a rename silently skipping them,
      which a package-level "ok" would not have caught)
    go vet ./kernel/... -> clean; gofmt -l on the three changed files -> clean

files_changed:
  - kernel/index/store.go
  - kernel/index/store_test.go
  - kernel/supervisor/supervisor_test.go
