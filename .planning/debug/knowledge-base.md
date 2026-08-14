---
status: complete
type: knowledge-base
note: Permanent reference catalogue of defect classes, not a debug session — status set only so audit-open's scanner (open = status not in [resolved, complete]) doesn't flag it.
---

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

---

## KB-005 — `os/exec` closes the pipe it handed you, underneath your reader

**Discovered:** 2026-08-12 · [ci-flaky-test-portable](resolved/ci-flaky-test-portable.md) · fix `4c524a7`
**Class:** subprocess I/O ownership · **Severity:** total, silent loss of a subprocess's output — ~48%/iteration under a starved scheduler

### The pattern

The obvious way to stream a subprocess's output while also collecting its exit status is the
*documented-incorrect* way:

```go
stdout, _ := cmd.StdoutPipe()
cmd.Start()
go func() {                                  // reader
    scanner := bufio.NewScanner(stdout)
    for scanner.Scan() { lines <- scanner.Bytes() }
}()
go func() { done <- cmd.Wait() }()           // races the reader
```

`StdoutPipe`'s own doc says it outright: *"Wait will close the pipe after seeing the command exit,
so it is incorrect to call Wait before all reads from the pipe have completed."* When the `Wait`
goroutine wins, the reader's first `Read` returns `os.ErrClosed`, `Scan()` returns false without
ever yielding a line, and the channel closes empty.

What makes this so easy to ship is that **there is no correct ordering available**. "Finish all
reads, then Wait" is exactly what streaming cannot promise — you don't know when output ends until
you've hit EOF, and the whole point is to hand lines onward as they arrive. So the fix is not to
sequence the two goroutines; it is to take the pipe out of os/exec's ownership:

```go
pr, pw, _ := os.Pipe()
cmd.Stdout = pw                              // an *os.File: handed to the child as an fd
cmd.Start()
pw.Close()                                   // parent's copy — without this, no EOF, ever
// read pr freely; Wait now has nothing to do with it
```

What `cmd.Stdout` is set to changes os/exec's behaviour completely, and only one row is safe to read
concurrently with `Wait`:

| `cmd.Stdout` is… | What os/exec does | Who closes your read end |
|---|---|---|
| an `*os.File` **you** own | passes the fd straight to the child — no pipe, no copy goroutine | **you do** — no interaction with `Wait` |
| any other `io.Writer` | makes a pipe + copy goroutine; `Wait` blocks until the copy finishes | os/exec (see KB-006 for how that blocks) |
| the `cmd.StdoutPipe()` return | makes a pipe, hands you the read end | **os/exec, inside `Wait`** — the trap |

### The tell

**Zero lines, not truncated lines.** A mid-stream truncation race yields partial data; this one
yields *nothing at all*, because the close lands before the first `Read` rather than during. With a
buffered hand-off you may instead see exactly one line — whatever a send could hold in flight. So
"got `[]`, want 3 lines" or "got exactly 1" from a process that certainly wrote more is the
signature, and an elapsed time of `0.00s` confirms nobody waited on anything.

Then check `scanner.Err()`, which is the single observation that identifies this class:

- `os.ErrClosed` (`"read |0: file already closed"`) → the pipe was closed under you. This.
- `nil` / EOF → the child genuinely wrote nothing; go look at exec-side causes instead.

In this session that one check is what separated "Wait closed the pipe" from "the fixture never
ran" — 194/194 empty iterations reported `ErrClosed`, which killed the `ETXTBSY`/failed-exec
hypothesis outright.

### Why it hid

The window is not a narrow instant — for a fast-exiting child it is *the entire run*, so this is
less a race than a coin flip weighted by core count and load. A 12-core dev box almost never lost
it; a 2-core loaded GitHub runner lost it constantly. Forced (`GOMAXPROCS=1`, `taskset -c 0`) it
reproduced at 48.5%.

More uncomfortably, **the codebase had already found this bug and not fixed it**. A comment at the
adjacent stderr site named `cmd.StderrPipe()` read concurrently with `Wait` as *"the same
incorrect-usage pattern already flagged on the stdout side"* — and declined to introduce a second
instance of it. The stdout side it was pointing at still had the first one. This is KB-004's lesson
in a new costume: prose that identifies a hazard is not a fix for that hazard. When you find a
comment describing a bug elsewhere in the file, go and check that the bug described was actually
removed.

### Gate gap this exposed

`go test -count=1` on a developer machine cannot see this class — same under-sampling that let
KB-003 through. The gate that *did* catch it was CI, which is to say: the flake was the gate, and it
was read as CI being unreliable rather than as CI reporting a real defect. Docs-only commits failing
was treated as proof the failure was noise; it was actually proof the failure was **diff-independent
and latent**, which is a much stronger signal than a failure correlated with a change.

**Recurrence guard now in place:**
`TestExecLinkSpawner_DeliversBufferedOutputWhenDrainedAfterSubprocessExit` — it drains *only after*
the subprocess has fully exited, which converts the probabilistic race into a deterministic one
(`Wait` is guaranteed to have had its chance first). Mutation-checked: fails 10/10 against the
unfixed spawner with the predicted signature, passes with the fix.

### Where to watch for it in this codebase

- Any `StdoutPipe`/`StderrPipe` call site. After this fix there are **none** left in non-test code —
  keep it that way; adding one back is the whole bug.
- `kernel/pluginhost/host.go` builds an `exec.Cmd` but hands it to go-plugin, which owns the I/O
  itself (and deliberately leaves `cmd.Stderr` unset for that reason). Don't "helpfully" attach
  pipes to a command someone else is going to run.
- Anywhere a test or tool wires `cmd.Stdout` to a `strings.Builder`/`bytes.Buffer` — that is the
  middle row of the table above, and KB-004 already recorded it as a data race against the copier.

### Checklist for review

- [ ] Is `cmd.Wait()` reachable concurrently with any read of a pipe os/exec created? If yes, this
      is the bug, regardless of how unlikely the window looks.
- [ ] Is `cmd.Stdout`/`cmd.Stderr` an `*os.File`? If not, os/exec owns a pipe and a goroutine on
      your behalf — know which row of the table you are in.
- [ ] Own an `os.Pipe`? Is the **parent's** copy of the write end closed after `Start`? Forgetting
      it means EOF never arrives and the reader hangs forever instead of failing fast.
- [ ] Does a failing streaming test report *zero* items rather than partial ones? Check
      `scanner.Err()`/read error for `os.ErrClosed` before assuming the producer wrote nothing.
- [ ] Does a comment in this file describe a bug "already flagged elsewhere"? Go verify that
      elsewhere was fixed.

---

## KB-006 — A forked grandchild inherits the pipe, so `Wait` outlives the process you killed

**Discovered:** 2026-08-12 · [ci-flaky-test-portable](resolved/ci-flaky-test-portable.md) · fix `4c524a7`
**Class:** process lifecycle / inherited descriptors · **Severity:** unbounded block on the cancel path, holding an exclusive resource

### The pattern

This is KB-005's mirror image, and the two must be read together: KB-005 is os/exec closing a pipe
**too early**, KB-006 is a pipe that will not close **at all**. Both reduce to the same question —
*who holds this file descriptor* — and fixing one does nothing for the other.

Killing a subprocess does not close the pipes you gave it. Descriptors survive `fork`, so every
process the child spawned holds its own copy of the write end. Meanwhile:

- `exec.CommandContext`'s cancel signals **exactly one pid**, and `SysProcAttr` is deliberately nil
  at this site so link subprocesses stay in the kernel's process group (KB-004).
- When `cmd.Stderr` is a non-`*os.File`, `cmd.Wait` waits for os/exec's copy goroutine to reach EOF.
- That EOF requires **every** holder of the write end to close it — including the orphan.

So `Wait` blocks for the *orphan's* lifetime rather than the child's. Here that parked `Wait`,
`done`, and `linkSession.consume`'s terminal event for a `sleep 30`'s full 30 seconds, while the
orphan kept holding the whatsmeow store that T-08-07 requires be held exclusively. The kill
"succeeded" and the session was still stuck.

### The lever, and the two that were rejected

| Lever | Effect | Verdict |
|---|---|---|
| `cmd.WaitDelay` | after the process is gone, force-close the pipes and return within a bound | **chosen** |
| `SysProcAttr.Setpgid` + `kill(-pgid)` | kills the whole tree, orphan included | **rejected** — removes link subprocesses from the kernel's process group and regresses KB-004's signal-reaping fix |
| `SysProcAttr.Pdeathsig` | child dies with parent | already rejected in KB-004 (parent-thread-scoped, and wrong layer) |

**Choosing the delay is itself a hazard.** `WaitDelay` must never elapse on a *clean* exit, or it
converts a nil exit error into `ErrWaitDelay` and you have traded a rare flake for a new one in
whatever asserts a clean shutdown. 1s here is ~1000x the observed clean-exit drain and still 3x
inside the test's 3s bound — verified by stability runs, not by argument.

**And it invalidates an assumption you may have written down elsewhere.** "`Wait` returned,
therefore os/exec's copy goroutine has finished, therefore nothing can be writing concurrently" is
true *only* off the `WaitDelay` path. On it, `Wait` force-closes and returns without awaiting the
copier, so a final `Write` can still be in flight. A `flush()` here carried exactly that reasoning
in a comment; it was already mutex-guarded so it stayed correct, but its stated *justification* had
become false. When you add `WaitDelay`, grep for any safety argument resting on "Wait has returned".

### The tell

Two failures in one CI run with **different elapsed-time signatures**: `0.00s` on a clean-exit test
(KB-005) and exactly `3.00s` — the timeout bound, to the centisecond — on a kill test. An elapsed
time equal to the bound means something waited out the clock rather than failing; an elapsed time of
zero means nothing waited at all. *Those are not the same bug*, and treating them as one is how a
single-cause fix ships and the gate stays flaky. Fixing only KB-005 would have left the kill failure
in place and made the correct fix look like a failed one.

### Gate gap this exposed

No test covered "subprocess forks a child, **then** gets killed". The existing kill test ran
`sleep 30` through `sh -c`, which produces a grandchild only when `/bin/sh` decides to fork — bash
does, dash often `exec`s directly and doesn't. The coverage was therefore accidental and
**shell-dependent**, which is also why it reproduced on the dev box (bash) and behaved differently
in CI (dash). A fixture whose behaviour depends on which `/bin/sh` is installed is not a test of
anything.

**Recurrence guard now in place:**
`TestExecLinkSpawner_KillYieldsDoneWhenSubprocessLeftAForkedChild` — forks the grandchild
*deliberately* rather than hoping the shell obliges, and reaps it via a recorded pid (measured: 0
stray processes across 5 runs). Mutation-checked at 10/10 against the unfixed spawner.

### Where to watch for it in this codebase

- Every spawn site that cancels or kills: `kernel/httpapi/whatsapplink.go` and
  `kernel/pluginhost/host.go`, both intentionally `SysProcAttr`-nil per KB-004 — which means both
  are structurally exposed to orphans by design, and must bound their waits instead.
- Anywhere a reader is expected to see EOF *because* a process was killed. If anything could have
  forked, close your own read end on cancellation rather than waiting for an EOF that has no
  obligation to arrive. That watcher must exit when the read completes, or it becomes the per-spawn
  goroutine leak KB-003 warns amplifies later flakes.

### Checklist for review

- [ ] Can the subprocess fork? (A shell, a wrapper script, anything spawning helpers — assume yes.)
      Then `CommandContext`'s kill does not end the tree.
- [ ] Is `cmd.WaitDelay` set at any spawn site that can be cancelled? Unset means "wait forever for
      whoever inherited these fds".
- [ ] Is the delay comfortably above the clean-exit drain **and** below any caller's timeout bound?
      Both ends matter — too tight invents a new flake on the healthy path.
- [ ] Does anything rely on "`Wait` returned, so the copiers are done"? `WaitDelay` breaks that.
- [ ] On cancel, does the reader get released by closing your own descriptor, rather than by hoping
      for EOF?
- [ ] Two tests failing in one run — do their elapsed times tell the same story? A `0.00s` and a
      `bound.00s` are two mechanisms, not one flaky package.

---

## KB-007 — `test.afterAll` destroys per JOB what module scope creates per PROCESS

**Discovered:** 2026-08-14 · [e2e-fs-recursion-ci-flake](resolved/e2e-fs-recursion-ci-flake.md) · fix ships with this entry
**Class:** test lifecycle / setup–teardown scope mismatch · **Severity:** CI-only hard failure, three sibling specs passing *vacuously*, and an unconditional temp-dir leak on every run — all from one cause

### The pattern

**Setup scope and teardown scope must be the same unit.** State is created in a spec file's module
scope and destroyed in a test hook:

```ts
// module scope — runs ONCE PER NODE PROCESS
const corpusDir = mkdtempSync(join(tmpdir(), 'topos-e2e-fs-recursion-'));
writeFileSync(join(corpusDir, 'receipts/nested-invoice.pdf'), …);

test.afterAll(() => rmSync(corpusDir, { recursive: true, force: true }));  // runs ONCE PER JOB
```

This reads as symmetric and reviews as symmetric. It is not. Module scope is bound to the **Node
process**; `test.afterAll` is bound to a **Playwright job**. A process can run many jobs, so the
teardown fires while the setup that produced the state has already been permanently consumed —
Node's module cache guarantees the module body will not run a second time in that process.

Three facts from `playwright@1.62.1` make that gap the *default* path rather than an unlucky one:

| Mechanism | Source | Consequence |
|---|---|---|
| `parallelWithHooksGroupSize = Math.ceil(tests / workers)` | `runner/index.js` `createTestGroups` | a 2-test file becomes **two jobs** at `workers>=2`, and **one job** at `workers==1` |
| `if (!nextSuites.has(suite)) await _runAfterAllHooksForSuite(...)` | `worker/workerProcessEntry.js` | `afterAll` fires at the end of **every** job, not at the end of the file |
| `findIndex(w => w.worker.hash() === job.workerHash)` before falling back to a fresh slot | `Dispatcher._scheduleJob` | the next job is **preferentially** handed back to the same worker — warm module cache, no re-seed |

So job 1 runs, `afterAll` deletes the corpus, and the dispatcher's *preferred* choice is to run job 2
in that same process — against a directory that no longer exists. This is not a race. There is no
timing window and nothing to lose; it is a scheduling arrangement, which is why the repro
(`--workers=2` over a slower single-test file plus this one) reproduced on the **first** attempt with
the CI trace's artifact directory name matching character for character.

### Why it hid — worker count changes CORRECTNESS here, not speed

A 12-core dev box resolves `workers: '50%'` to 6, so the two jobs land in two *different* worker
processes. Each re-runs module scope, each owns its own corpus, each owns its own kernel — the defect
is structurally unreachable. A 4-vCPU CI runner resolves to 2, and co-location becomes the
dispatcher's preferred outcome.

Note the direction, because it inverts the usual intuition: `--workers=1` also *passes*, since
`ceil(2/1) = 2` yields a single group and `afterAll` cannot fire between the two tests. The failure
exists only in the middle of the range. **Worker count is not a performance knob for a spec written
this way — it selects whether the file's teardown runs between its own tests.** Any reasoning of the
form "fewer workers is just slower, more workers is just faster" is invalid the moment a file mixes
module-scope state with `beforeAll`/`afterAll`.

This is the same shape as **KB-003** and **KB-005**: a latent defect, invisible at a dev box's core
count, surfaced by a starved CI runner, and misattributed to the most recent unrelated commit — here
quick task `260814-mkl`, which added an ambient type declaration and nothing else. KB-005's line
applies verbatim: a failure that is *diff-independent* is latent, not introduced, and that is a
stronger signal than a failure correlated with a change.

### The masking factor — and the three specs that passed vacuously

The `kernel` fixture is declared `{ scope: 'worker' }` (`web/e2e/fixtures/kernel.ts`), a deliberate
choice under D-02 (one kernel per spec file). So when job 2 reused the worker, it also reused a live
kernel whose **index still held the item job 1 had synced**. Hence the failure's misleading shape:

- line 124 — "the nested item is present in the stream" — **PASSED**, served from a stale index
  describing files that had been deleted from disk;
- line 126 — `unlinkSync(nestedFilePath)` — failed with `ENOENT`.

The first observable symptom was therefore a filesystem call one line *after* the assertion that
should have caught it. An assertion passing off cached state is not evidence the state exists.

The corollary is the worse half of this entry. Four phase-12 filesystem specs split into ≥2 jobs at
`workers>=2`, but only `12-filesystem-recursion` ever touched the corpus **from disk** in a later
job. `12-zero-match-diagnostic`, `12-filesystem-tracer` and `12-external-rehearsal` only read through
the kernel — so on a co-located run they were asserting against a corpus that no longer existed, and
they **passed**. Three green specs proving nothing, and one red one. *The loud instance was the only
reason the silent ones were found;* a fix scoped to the file that failed would have left the other
three passing vacuously forever.

> When a scheduling-dependent teardown bug surfaces in one file, enumerate every file sharing the
> pattern and classify each by **what it reads**. Files that read only through a surviving cache
> don't fail — they stop testing.

### The second presentation: the loader process, and the tell that serial-mode was the symptom fix

`/tmp` held **72** leaked corpora — roughly one per spec file per suite run, *including* the
single-test files that are structurally immune to job splitting. A `--list` invocation (collection
only, runs no test) took the count 13 → 14.

Playwright loads every spec file in a **collection/loader process** to build the test tree. Module
scope executes there. No test runs there, so no hook runs there, so that corpus is orphaned
**unconditionally — on every run, including green ones**. Same root cause, second presentation:
setup bound to process lifetime, teardown bound to a test hook.

This is what settled the fix. `describe.configure({ mode: 'serial' })` would have collapsed the file
back to one job, closed the `ENOENT`, turned CI green — **and left the loader leak completely
untouched**, because serial mode changes job splitting and says nothing about which process runs
module scope. A fix that resolves one presentation of a cause while a second presentation of the same
cause carries on unchanged is a symptom fix, and the surviving presentation is how you know.

### The fix shape

Move the teardown to the scope the setup actually has. `process.on('exit')` is the only Node hook
whose lifetime is the process:

```ts
// web/e2e/fixtures/corpus.ts
export function mkdtempCorpus(prefix: string): string {
  const dir = mkdtempSync(join(tmpdir(), prefix));
  registerCleanupOnce();                     // one process.on('exit') for all corpora
  created.push(dir);
  return dir;
}
```

Every phase-12 filesystem spec routes through it and **every `test.afterAll` `rmSync` is deleted**.
What each spec proves is unchanged, no parallelism is surrendered, and D-02 (worker-scoped kernel),
D-03 (corpus seeded before kernel boot) and D-15 (no retries, no serial mode) are all untouched —
deliberately. Only the scope mismatch is ours; the dispatcher's co-location and the worker-scoped
kernel are correct, intentional behaviours and were left exactly as they were.

Empirically confirmed rather than assumed — that `process.on('exit')` fires inside a Playwright
worker was the fix's stated blind spot. A full 122-test run at `--workers=2` added **zero** corpora
(74 before, 74 after), and the collection probe that previously leaked one per invocation now leaks
none. It fires in worker and loader processes alike.

### Gate gap this exposed

Every gate was structurally blind. `make e2e` on a dev box runs at 6 workers, where the arrangement
cannot occur; `--workers=1`, the instinctive "make it deterministic" debugging move, is *also* in the
passing region. The failing region is only the middle, so no local invocation of the existing gate
could reach it. As in KB-003 and KB-005, CI was the gate that caught it, and the flake was initially
read as CI being unreliable rather than CI reporting a real defect.

Worse, no *runtime* assertion can cover this class reliably: the failing arrangement is
scheduling-dependent, and the three sibling instances fail silently by passing. A test cannot catch a
bug whose signature is "a different test stops meaning anything."

**Recurrence guard now in place:** `web/e2e/specs/spec-hygiene.spec.ts` — a **structural** spec (boots
no kernel) that fails if any spec file calls `mkdtempSync` directly, or removes a directory inside a
`test.afterAll` body. It covers both halves of the contract, which are the two adjacent ways to
reintroduce the defect, and extracts hook bodies by balanced-paren scan rather than a regex a
parenthesised body would defeat. Mutation-checked 2/2 against the pre-fix spec, naming the offending
file in both messages, so it cannot pass vacuously. The rule is also written into `docs/testing.md`
under "Writing a new spec" — a structural gate plus the doc that explains it, because the gate alone
tells you *that* you broke a rule, not why it exists.

### Where to watch for it in this codebase

- **Any new spec doing work at module scope.** In this harness that is not an anti-pattern to remove
  — D-03 requires corpus state to exist before the kernel boots, which is before any hook runs. The
  requirement is that its teardown be process-scoped to match.
- **`web/e2e/fixtures/kernel.ts`** — `configSpec` and `kernel` are both `scope: 'worker'`. Anything
  that outlives a job can serve stale data into the next one; assertions that read only through the
  kernel cannot detect that their fixtures were destroyed.
- **Any spec file that grows from 1 test to 2.** `12-filesystem-add-source`,
  `12-filesystem-root-label-match` and `12-include-glob-metadata-preview` were immune only because
  `ceil(1/n) = 1` gives them a single job. That immunity evaporates the day someone adds a second
  test, with no edit to the setup code that would prompt a reviewer to look.
- **`web/playwright.config.ts`** — `fullyParallel: true` and the default `workers: '50%'` are the
  preconditions. Changing either changes which specs are reachable by this class.
- **Non-e2e analogues:** any Go `TestMain`/package-level fixture torn down in a `t.Cleanup`, or any
  singleton initialised at import and released per-test.

### Checklist for review

- [ ] For every piece of state a test uses: **what unit creates it, and what unit destroys it?** If
      those two names differ (module/process vs suite/job/test), that is the bug, whatever the
      current worker count makes visible.
- [ ] Is teardown attached to a hook while setup sits at module scope? Node runs a module body once
      per process; no Playwright hook has process lifetime.
- [ ] Does the spec file have ≥2 tests *and* a `beforeAll`/`afterAll` *and* run under
      `fullyParallel`? Then it splits into multiple jobs and `afterAll` runs between its own tests.
- [ ] Would `--workers=1` and a high worker count both pass? Then a passing run proves nothing about
      the middle of the range — reproduce at `--workers=2` before concluding anything.
- [ ] Does an assertion that "passed" read through a long-lived cache or worker-scoped fixture? It
      may be describing state that has already been deleted.
- [ ] When one file in a family fails: which sibling files share the pattern, and do they only read
      through a surviving fixture? Those are passing vacuously — enumerate them before fixing.
- [ ] Does the proposed fix close **every** presentation of the cause? A leak, a slow drift, or a
      loader-process side effect that survives your fix means you fixed the symptom.
- [ ] Before blaming the newest commit for a CI-only failure: does the defect reproduce from the
      test's own structure with no reference to that diff? (KB-003, KB-005 — same trap, third time.)
