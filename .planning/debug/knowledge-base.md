# Debug Knowledge Base

Cross-session patterns worth remembering. Each entry records a defect *class* — not a
one-off bug — so future work can recognise the shape before shipping it again.

Add an entry when a resolved session surfaces a pattern that could plausibly recur in a
different subsystem. Link back to the resolved session for full evidence.

---

## KB-001 — Graceful shutdown cancels the context that also finalises the DB write

**Discovered:** 2026-08-05 · [proton-stuck-syncing](resolved/proton-stuck-syncing.md) · fix `f01df96`
**Class:** two-phase write / lifecycle · **Severity:** silent data loss (no error surfaced to the user)

### The pattern

Code opens a unit of work, records a "started" row, does the work, then writes the outcome —
all under a single `ctx`:

```go
runID, _ := store.StartSyncRun(ctx, sourceType)   // phase 1: "running"
res := doTheWork(ctx)                             // cancelled here on shutdown
store.FinishSyncRun(ctx, runID, res)              // phase 2: NEVER RUNS
```

When that `ctx` is cancelled mid-work — graceful shutdown, an HTTP client disconnecting behind a
manual-trigger endpoint, a parent errgroup failing — `database/sql` rejects the phase-2 UPDATE
outright with `context canceled`. The write is not retried and not queued; it is *skipped*. The
row is stranded in its intermediate state forever.

The insidious part: **the outcome has already happened**. The work either succeeded or failed, and
that fact is real. Cancelling the context that merely *records* it destroys information about
something that already occurred. Cancellation should stop future work, never the recording of past
work.

### Why it hides

- No error reaches the user. The process is shutting down; the log line scrolls past or is lost.
- It only fires on a cancellation that lands *inside* the work window, so it is timing-dependent
  and will not reproduce under normal test runs.
- It preferentially hits the **slowest** component, because that component has the widest window
  to be mid-work at shutdown. In this session only the Proton source was ever affected (IMAP over
  the network to Bridge), which made it look Proton-specific when the defect was in shared code.
  **A bug that appears to affect exactly one plugin may be a shared-path bug filtered by latency.**

### The fix shape

Finalise on a detached context with its own bound:

```go
finishCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), finishRunTimeout)
defer cancel()
store.FinishSyncRun(finishCtx, runID, res)
```

`context.WithoutCancel` (Go 1.21+) keeps values while dropping cancellation. The explicit timeout
stops a detached write from hanging shutdown indefinitely.

Then add the two companions, because the fix above only prevents *new* damage:

1. **Boot-time reconciliation.** A freshly started process has no in-flight work by definition, so
   anything still in an intermediate state at startup was stranded by a previous process. Mark it
   terminal-with-cause (`error` / "interrupted"), don't delete it — the work was genuinely attempted
   and genuinely did not complete, and that is worth knowing. Never let a repair failure block boot.
2. **Bound the blast radius in the read path.** See KB-002 below — it is what turned this from
   transient to permanent.

### Where to watch for it in this codebase

Any two-phase write, i.e. anywhere a "started" marker precedes a separate "finished" write:

- **Phase 8 WhatsApp plugin event-stream persistence** — a long-lived WebSocket whose events are
  checkpointed as they arrive is exactly this shape, and whatsmeow's connection ctx is cancelled on
  shutdown. Highest-risk upcoming instance.
- Any Phase 4-5 plugin that records sync progress or a resume cursor before/after fetching.
- Anything writing a "last successful position" after a long streaming read (large mailbox backfill,
  chat-history import).

### Checklist for review

- [ ] Does any write record something that has *already happened*? If so, it must not be cancellable.
- [ ] Is there a boot-time path that reconciles intermediate states left by a dead process?
- [ ] Is there a test that cancels the context *mid-work* and asserts on the persisted state?
      (Happy-path lifecycle tests will not catch this — see KB-002's note on gate gaps.)

---

## KB-002 — "Any row" vs "latest row" turns a transient glitch into a permanent one

**Discovered:** 2026-08-05 · [proton-stuck-syncing](resolved/proton-stuck-syncing.md) · fix `f01df96`
**Class:** query semantics · **Severity:** stale state pinned indefinitely

### The pattern

A status query asks "is anything in state X?" when the real question is "is this entity's *current*
record in state X?":

```sql
-- wrong: any row, unbounded in time
SELECT DISTINCT source_type FROM sync_runs WHERE status='running' AND finished_unix IS NULL

-- right: each entity's latest row
WHERE id IN (SELECT MAX(id) FROM sync_runs GROUP BY source_type) AND status='running'
```

With any-row semantics a single stale row outvotes **every** subsequent success — in this case one
orphan from 2026-08-04 beat 244 later successful runs. With latest-row semantics the next completed
run naturally supersedes it and the state self-corrects.

"Is this thing happening *right now*" is a property of the entity's current record, not a property
of the entire history table.

### The tell

Two queries over the same table disagreeing produces contradictory UI that looks impossible:
here, a **green health dot** (from `LatestSyncRunPerSource` → `ok`, correct) rendered simultaneously
with **"Syncing..."** (from `SyncingSourceTypes` → any-running, wrong). When a UI shows two states
that cannot both be true, suspect two different queries backing two different fields — not a
rendering bug. The frontend was never at fault in this session.

### Gate gap this exposed

Happy-path lifecycle tests do not catch either KB-001 or KB-002. The codebase already had
`TestStartAndFinishSyncRun` and even `TestSourcesHandler_RunningRowReportsSyncingTrue`, and both
passed throughout. What was missing:

- no test drove the **interrupted** path (cancel mid-work, then assert persisted state)
- no test set up a **stale row plus a later completed row for the same entity**, which is the only
  arrangement where any-row and latest-row semantics differ

When writing tests for a status/lifecycle table, include an aged-stale-row fixture. It is the case
that distinguishes correct from incorrect aggregation, and it is almost never written by default.

---

## KB-003 — A latest-row aggregate is the wrong oracle for a question about one specific run

**Discovered:** 2026-08-11 · [apply-midflight-sync-race](resolved/apply-midflight-sync-race.md) · fix `998a9ab`
**Class:** test oracle / query semantics · **Severity:** flaky gate, ~25%/iteration, misattributed to a recent refactor

### The pattern

This is KB-002's mirror image, and reading KB-002 alone will lead you straight into it. KB-002
established that "is this source syncing *right now*" must use latest-row (`MAX(id) GROUP BY source`)
semantics. The trap: that makes latest-row feel like the *correct* reader generally, so it gets
reached for again when the question has quietly changed shape.

A test asserted "was the sync that was in flight when `Apply` was called finalised?" through
`LatestSyncRunPerSource`. But that aggregate can only ever address the source's *newest* run, and
the code under test legitimately started a newer one before returning — `Apply`'s pre-Reconcile
failure branch restarts the old generation, whose immediate first refresh inserts a second `running`
row for the same source. Whenever that INSERT won the race against the test's read, the oracle
inspected a perfectly healthy new run and reported it as a stranded mid-flight sync.

**The question determines the reader:**

| Question | Correct reader |
|---|---|
| Is this entity in state X *now*? | latest row (`MAX(id) GROUP BY entity`) — KB-002 |
| Was *that specific* unit of work finalised? | that row, addressed directly (here: oldest-first `ORDER BY id`, index 0) |

An aggregate cannot express the second question. If a later record can legitimately supersede the
one you mean, any aggregate over the group is answering a different question than you asked.

### Why it hid — and why it looked like a regression

The failure surfaced immediately after a concurrency refactor (the `genMu` reader/writer split) that
touched *exactly* the ordering contract the test pins. Every circumstantial signal pointed at the new
code. It was wrong by 3 days: the test reproduced identically at the pre-refactor baseline, and
`git log -S` showed the bad assertion had been there since the test was written.

What actually changed was *amplification*. The test leaked one goroutine — parked forever in a
fixture's `Match` on an uncancelled context — plus one open index handle **per iteration**. As the
suite grew and iteration counts rose, contention climbed and the latent race began winning often
enough to trip the gate.

> **A flake that appears the day you touch the relevant code may still predate you.** Before
> bisecting the suspect refactor, run the failing test at the baseline *at a high enough iteration
> count*. An isolated `-count=3` pass is not evidence of anything against a ~25%/iteration failure
> rate — that was the under-sampling that made this look refactor-induced for the first hour.

### The fix shape

1. **Address the specific record.** Add a per-entity, insertion-ordered read (`WHERE source = ?
   ORDER BY id`) so index 0 is deterministically the run that already existed when the operation
   under test began.
2. **Mark test-only surface as such.** No production caller needed a source's full run history, so
   the reader is `SyncRunsForSourceForTesting`, following the existing `config.NewStoreForTesting`
   precedent. A test need should not silently widen the supported API.
3. **Stop leaking the amplifier.** `t.Cleanup(s.Shutdown)` ends the generation the operation under
   test restarts. A test that leaves a live goroutine behind degrades *every* later test in the
   package, and the resulting flakes will be blamed on whatever was committed most recently.

### Gate gap this exposed

`go test -count=1` cannot see this class at all — that is the gate every plan runs. The failure
needs either a raised iteration count or accumulated in-package contention to appear, so it slipped
through every per-plan gate and only surfaced at a phase-final full-suite run, by which point it
looked like fallout from the most recent merge.

**Recurrence guards now in place:** the repaired test itself (mutation-checked — reverting KB-001's
detached finalise kills it 10/10, removing `stopScheduler`'s `<-s.done` wait kills it 5/10, so it
genuinely still pins the production contract); and
`TestSyncRunsForSourceForTesting_EarlierFinishedRunSurvivesALaterRunningOne`, the
earlier-finished-plus-later-running fixture that is the *only* arrangement where the per-run reader
and the latest-row aggregate disagree. Without that fixture the two are substitutable undetected and
the aggregate can quietly creep back in — the same "write the fixture that distinguishes the two
readers" lesson KB-002 closes on.

### Checklist for review

- [ ] Does this assertion name a *specific* unit of work? If a later record for the same entity can
      supersede it, an aggregate cannot express the assertion — address the row directly.
- [ ] Does the code under test legitimately start new work before returning? Then "the latest row"
      and "the row I mean" are different rows.
- [ ] Does this test leave any goroutine, scheduler generation, or DB handle running at return?
- [ ] Before blaming a flake on the newest commit: does it reproduce at the baseline at
      `-count=20`+?
- [ ] New read/write method added only because a test needed it — is it `ForTesting`-suffixed?

---

## KB-004 — `defer` behind a blocking call is dead code on every signal path

**Discovered:** 2026-08-11 · [plugin-procs-orphaned-on-exit](resolved/plugin-procs-orphaned-on-exit.md) · fix `799200b`
**Class:** process lifecycle / unreachable teardown · **Severity:** resource leak (live orphaned subprocesses), invisible on the most-travelled path

### The pattern

Teardown is registered the idiomatic way, then the function parks forever in a blocking call:

```go
defer store.Close()
defer sup.Shutdown()        // kills every plugin subprocess
defer linkStore.Shutdown()  // kills every whatsapp-link subprocess

return http.ListenAndServe(listen, router)   // returns only on listener ERROR
```

This reads as correct and reviews as correct. It is dead code. **Go does not run deferred functions
when a signal terminates the process with its default disposition**, and with no `signal.Notify`
anywhere the only path that reaches those defers is the listener itself failing — i.e. never, during
normal operation. Every ordinary operator exit (Ctrl-C, `kill`, a service-manager stop) skipped the
entire teardown block.

`grep -c signal.Notify` over first-party code returning **zero** in a long-lived daemon is the
one-line tell. It was zero here.

### The AND-gate: unreachable teardown alone is not enough

The leak needs a second, independent condition — **the child must not die on its own**:

| | child dies with parent | child survives parent |
|---|---|---|
| **teardown runs** | clean | clean |
| **teardown skipped** | clean *by accident* | **leak** |

`hashicorp/go-plugin` puts you firmly in the right-hand column *by design*: the plugin server
explicitly swallows SIGINT (`server.go:460-473`, comment "Eat the interrupts") and `Serve()` has no
parent-death watchdog — no stdin-EOF watch, nothing. Its `select` only unblocks on a client-issued
shutdown or a test-mode context. So a terminal Ctrl-C, which the tty delivers to the *whole
foreground process group*, reaches every plugin child and is deliberately ignored by it.

Only condition (A) is ours to fix. (B) is upstream-by-design and must be treated as a fixed
constraint, not a bug to route around.

### Why it hid — an unrelated commit masked the dominant path

The TODO said "on some exit path", and the operator had stopped seeing orphans without any fix
landing. Both were explained by one line of Makefile added five days earlier for *unrelated*
port-guard reasons (`6d0e6a8`):

```make
trap 'kill 0' INT TERM
```

`kill 0` signals the whole process group with **SIGTERM** — which go-plugin does *not* swallow. So
`make dev` + Ctrl-C reaped the children by pure accident, masking the defect on the single path
exercised most. Mapping real exit paths onto the matrix:

| Real path | Row | Result |
|---|---|---|
| `make dev` Ctrl-C (Makefile trap → group SIGTERM) | T3 | reaped **by accident** |
| bare `./bin/topos serve` + Ctrl-C (group SIGINT) | T2 | **orphans** |
| `kill <pid>` / pkill / service stop (pid SIGTERM) | T4 | **orphans** |
| `web/e2e/fixtures/kernel.ts` `process.kill(-pid,'SIGTERM')` | T3 | safe **by luck** |

The kernel logged **zero** shutdown lines in all four rows — including the one that "passed" —
which is what proved T3 was accidental rather than correct. *A path that works because of an
incidental signal is not a path that works.*

> **A resource leak that "went away on its own" did not go away.** Look for something that changed
> the *observation* path — a wrapper script, a trap, a supervisor, a CI runner — before believing a
> defect self-healed with no commit behind it.

### The fix shape

The teardown was never wrong; it was unreachable. So restore reachability and change nothing else:

```go
srv := &http.Server{Addr: listen, Handler: router}
errCh := make(chan error, 1)
go func() { errCh <- srv.ListenAndServe() }()

sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

select {
case err := <-errCh:
    return err
case <-sigCh:
    signal.Reset(os.Interrupt, syscall.SIGTERM) // 2nd Ctrl-C still hard-kills
    ctx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
    defer cancel()
    if err := srv.Shutdown(ctx); err != nil {
        log.Printf("http drain: %v", err)  // logged, NOT returned
    }
    return nil   // <- the existing defers now run, in their existing order
}
```

Three details that matter more than they look:

1. **`signal.Reset` before draining.** Once you catch signals, a hung drain makes the process
   *unkillable by Ctrl-C* — you have converted a leak into a worse UX. Resetting restores default
   disposition so the second Ctrl-C works.
2. **The drain error is logged, never returned.** Returning it would propagate an error out of
   `runServe` and, depending on caller shape, risk skipping the very teardown this fix exists to
   reach. A best-effort HTTP drain must never gate subprocess cleanup.
3. **Defer order was already correct** (linkStore → supervisor → store → cancel). Making them
   reachable required *no* reordering — resist the urge to "improve" ordering in the same change.

Also fixed the one-shot path the same way: `runSync` uses `signal.NotifyContext` so a Ctrl-C
mid-sync cancels `RefreshAll` and still reaches `defer sup.Shutdown()`. Safe against **KB-001**
because the coordinator already finalises `sync_runs` on a detached `context.WithoutCancel` — worth
checking explicitly whenever you newly make a long operation cancellable.

### Rejected: `SysProcAttr.Pdeathsig` as defence-in-depth

Tempting and wrong in Go. `Pdeathsig` fires when the spawning **OS thread** exits, not the process,
and the Go runtime freely migrates goroutines between threads. Without `runtime.LockOSThread` around
the spawn it would kill *healthy* plugins at random — a worse failure than the one being fixed.

SIGKILL, panic, and OOM-kill therefore remain uncoverable by any in-process handler and still
orphan. That is a stated, documented limit, not an oversight.

### Gate gap this exposed

Nothing in the suite ever exercised a signal path. `make test`, `make dev-check`, and `make e2e` all
start and stop the kernel *programmatically* — and the e2e fixture happens to use a group SIGTERM,
landing on the one row that passes even unfixed. The whole gate stack was structurally blind to this.

The sharper lesson is about the shape of the regression test. A single-case test written against the
path the operator actually uses (`make dev` Ctrl-C) **would have been green against the live bug**.
Mutation-checking against the unfixed kernel proved it: 3 of 4 rows fail deterministically over 3
runs, the 4th passes. The 2×2 wasn't thoroughness — it was the minimum that distinguishes fixed from
broken.

> When a bug's trigger is a signal or a lifecycle event, enumerate the **{signal} × {target}**
> matrix. Testing only the path you use tests the path that was already accidentally working.

**Recurrence guards now in place:** `cmd/topos/shutdown_signal_test.go`
`TestServeReapsPluginSubprocessesOnShutdownSignal` — the 2×2 {SIGINT, SIGTERM} × {process-group, pid}
matrix. It asserts a *precondition* (a plugin child must exist before signalling) so it cannot pass
vacuously, and it distinguishes a live orphan from a zombie, matching the reported symptom exactly.
15 consecutive runs, 0 failures.

### Testing signals: three harness traps that each cost a false conclusion

Cache these — they are not specific to this bug, and each one produced a confidently wrong result:

1. **A background job from a non-interactive shell inherits `SIGINT = SIG_IGN`.** POSIX requires it,
   and the Go runtime *preserves* an inherited ignore. The first repro run concluded "the kernel
   survives SIGINT" — it does not. `/proc/<pid>/status` settled it: `SigIgn=0`,
   `SigCgt=fffffffd7fc1feff` (ordinary Go catch-all). Reset the disposition before exec
   (`setsid perl -e '$SIG{INT}="DEFAULT"; exec @ARGV'`) and read `SigIgn` to detect the artifact.
2. **`/proc/<pid>/task/<tid>/children` lists that *thread's* children, not the process's.** Go
   fork/execs from whatever OS thread it happens to be on, so reading only `task/<pid>/children`
   finds the subprocess or misses it depending on the scheduler — an early version of the test failed
   on a *random row each run*. Take the union over all `task/*/children`.
3. **`exec.Cmd` with a `strings.Builder` for `Stdout`/`Stderr` is a data race** against exec's copier
   goroutine, and silently loses output on timeout paths. Hand it an `*os.File` — passed straight to
   the child as an fd, neither problem.

### Where to watch for it in this codebase

- Any new long-lived daemon entrypoint, or any new blocking call at the end of `runServe`.
- Any new subprocess spawn site (`exec.Command` / `exec.CommandContext`) — today `kernel/pluginhost/host.go`
  and `kernel/httpapi/whatsapplink.go`, both intentionally leaving `SysProcAttr` nil so they share
  the kernel's process group.
- Scripts that stop the kernel by **pid** rather than group — `scripts/signal-readonly-smoke.sh`
  (`kill "$SERVER_PID"`) is exactly row T4, the worst-case row, and only became safe *because* of
  this fix.
- A doc comment asserting a lifecycle guarantee is not evidence of one. `linkStore.Shutdown`'s
  comment already claimed "called once, from kernel shutdown … so a Ctrl-C never orphans a linking
  process" — a guarantee the code could not possibly deliver. Grep for confident lifecycle prose and
  verify it.

### Checklist for review

- [ ] Does this daemon call `signal.Notify` / `signal.NotifyContext` at all? (`grep -c` returning 0
      in a process that spawns children is the tell.)
- [ ] Is any `defer` positioned behind a call that blocks indefinitely? Then it is dead code on
      signal paths — Go runs no defers on signal death.
- [ ] Does anything this process spawns die on its own when the parent dies? If not (go-plugin: no),
      skipped teardown means a leak, not just an unclean exit.
- [ ] After catching signals, can a second Ctrl-C still kill a hung process? (`signal.Reset` first.)
- [ ] Does a best-effort drain/flush error return in a way that could skip the *real* cleanup?
- [ ] Is the regression test a **{signal} × {target}** matrix, or a single case on the path you
      happen to use? Mutation-check it against the unfixed binary — if every row fails, the matrix
      may be redundant; if some row passes, that row is why the bug looked intermittent.
