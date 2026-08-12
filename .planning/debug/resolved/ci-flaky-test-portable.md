---
status: resolved
trigger: "The CI build on GitHub is flaky: sometimes passing, sometimes failing at the same line in the `make test-portable` step. This happens when no relevant code is touched too (such as docs only commits)"
created: 2026-08-12
updated: 2026-08-12
---

## Symptoms

DATA_START
- **Expected behavior:** CI (`make test-portable` step) passes consistently on every run, especially on commits that touch no Go code (docs-only commits).
- **Actual behavior:** CI is flaky — sometimes passes, sometimes fails at the same line. Confirmed failures on docs-only commits (e.g. "docs(README): add logo", "docs(README): Change logo width to 50px") and code commits alike.
- **Error message (from run 31602076950, 2026-08-12):**
  ```
  --- FAIL: TestExecLinkSpawner_StreamsLinesInOrderAndExitsClean (0.00s)
      whatsapplink_exec_test.go:121: got lines [], want [{"kind":"qr"} {"kind":"pairing_accepted"} {"kind":"paired"}]
  FAIL
  FAIL	github.com/davison/topos/kernel/httpapi	5.140s
  make: *** [Makefile:135: test-portable] Error 1
  ```
- **Timeline:** Multiple failures observed on 2026-08-11 and 2026-08-12 across at least 7 CI runs; interleaved with successes on similar commits (same-content docs commits both pass and fail). Exact first occurrence not yet established.
- **Reproduction:** Non-deterministic. Any push triggering the CI workflow's `make test-portable` step can hit it; docs-only commits confirm it is independent of the diff. Failing test: `TestExecLinkSpawner_StreamsLinesInOrderAndExitsClean` in `kernel/httpapi/whatsapplink_exec_test.go:121` — test reports 0.00s elapsed and got zero output lines where three JSON lines were expected.
- **Recent failed CI run IDs for evidence:** 31602076950, 31600339329, 31600191278, 31598316920, 31587114270, 31496695837, 31445383533
DATA_END

## Current Focus

bug_class: "Heisenbug (transient, non-deterministic, scheduling-sensitive) — SBFL skipped: no failing test locally to build a spectrum from, and a flaky spectrum would poison the ranking"

hypothesis: "newExecLinkSpawner (kernel/httpapi/whatsapplink.go:162-202) violates os/exec's documented StdoutPipe contract: it calls cmd.Wait() in one goroutine CONCURRENTLY with a bufio.Scanner reading cmd.StdoutPipe() in another. os/exec's Wait closes the parent's pipe fds as soon as the child exits, so when the Wait goroutine wins the race against the scanner goroutine's first Read, that Read returns os.ErrClosed ('file already closed'), scanner.Scan() returns false immediately, and lines closes having emitted ZERO lines — producing exactly 'got lines []' in 0.00s. The fixture exits within microseconds, so the window is the whole test."
test: "run TestExecLinkSpawner_StreamsLinesInOrderAndExitsClean at high -count under CPU starvation (GOMAXPROCS=1, taskset to one core) to delay the scanner goroutine's scheduling and let Wait win; also surface scanner.Err() to confirm it is os.ErrClosed rather than a clean EOF"
expecting: "CONFIRMS if the test fails with empty lines under starvation and the scanner error is 'file already closed'. REFUTES if it never fails at count>=500 under GOMAXPROCS=1, or if the scanner error is nil/EOF (which would mean the child genuinely wrote nothing and the cause is exec-side, e.g. ETXTBSY)."
next_action: "DONE — fix applied, all 5 guardrail signals passed, human-verified 2026-08-12 (fix accepted as-is including the 1s linkSubprocessWaitDelay). Session committed, filed as KB-005/KB-006, and archived."

reasoning_checkpoint:
  hypothesis: "TWO independent defects in newExecLinkSpawner (kernel/httpapi/whatsapplink.go), both latent-always and surfaced only by scheduling variance. RC1: cmd.Wait() runs concurrently with a bufio.Scanner reading cmd.StdoutPipe(); os/exec's Wait closes the parent pipe once the child exits, so when Wait wins, the scanner's Read returns os.ErrClosed and lines closes with ZERO lines. RC2: exec.CommandContext's cancel kills only the direct child; a grandchild the shell forked inherits the stderr pipe write end, and because cmd.Stderr is a non-*os.File, Wait blocks on os/exec's stderr copy goroutine until that orphan exits (30s) — so done never fires within the test's 3s bound."
  confirming_evidence:
    - "Standalone repro of the exact spawner shape: 194/400 iterations (48.5%) under GOMAXPROCS=1 yielded 0 lines, and scanner.Err() was 'read |0: file already closed' in 194/194 — os.ErrClosed, not EOF. Proves the pipe was closed under the reader, not that the child wrote nothing."
    - "os/exec StdoutPipe doc states the contract violated verbatim: 'Wait will close the pipe after seeing the command exit... it is thus incorrect to call Wait before all reads from the pipe have completed.'"
    - "RC2 repro: ps showed sleep(183301) as a child of the shell before kill and re-parented to PPID 1 after; cmd.Wait did not return within 5s in 2/2 runs with the exact test fixture. Local /bin/sh is bash and forks the grandchild."
    - "CI run 31598316920 shows BOTH tests failing in the same run (StreamsLinesInOrderAndExitsClean 0.00s, KillTerminatesLongRunningSubprocess 3.00s) — matching the two mechanisms exactly."
    - "GOMAXPROCS=1 taskset -c 0 go test -count=300 reproduced both failures locally; KillTerminates passes 30/30 at normal parallelism, confirming it is a scheduling race (kill vs the shell's fork), not a constant."
  falsification_test: "If the scanner error had been nil/EOF, RC1 would be refuted (the child genuinely wrote nothing — an exec-side cause such as ETXTBSY). If ps had shown no surviving grandchild while Wait still blocked, RC2 would be refuted."
  fix_rationale: "RC1: give the spawner its own os.Pipe for stdout (cmd.Stdout = pw, an *os.File) so os/exec passes the fd straight to the child, creates no copy goroutine, and Wait never closes our read end — removing the ordering constraint entirely rather than papering over it with a sleep or a retry. RC2: set cmd.WaitDelay so Wait force-closes the pipes and returns a bounded time after the process is gone, instead of blocking for the orphan's full lifetime. Both address mechanism, not symptom."
  blind_spots:
    - "WaitDelay must not fire on a CLEAN exit or it would return ErrWaitDelay and create a NEW flake in the nil-error assertion; chose 1s (~1000x the observed copier drain, 3x under the test's 3s bound) — verified by the mutation/stability runs rather than by argument."
    - "Closing the read end on cancel truncates any still-buffered stdout on the kill path. Intentional (caller asked to kill) but it means kill is not a graceful drain."
    - "Not verified on non-bash /bin/sh (CI is dash); the fix is shell-independent by construction, but the grandchild fixture's fork behaviour is not."
  candidate_causes:
    - "code: concurrent cmd.Wait() with StdoutPipe reads (CONFIRMED, RC1)"
    - "code: single-pid kill leaves a grandchild holding the stderr pipe, blocking Wait (CONFIRMED, RC2)"
    - "environment: 2-core loaded GitHub runner vs 12-core dev box — AMPLIFIER only, not a cause; both defects reproduce locally under induced starvation"
    - "config: SysProcAttr left nil (documented decision, KB-004) means no process-group kill — a precondition of RC2, deliberately preserved rather than changed"
  and_gate: "YES — this failure required >1 contributing condition, and the two CI symptoms are NOT one cause. RC1 alone explains the 0.00s empty-lines failure; RC2 alone explains the 3.00s kill failure. Fixing only RC1 would have left the suite still flaky, and the single-symptom evidence in the original report would have made that look like a failed fix. root_cause is therefore a SET."

## Evidence

- timestamp: 2026-08-12
  checked: "gh run list (last 20 runs) + gh run view 31602076950 --log-failed"
  found: "Failures on docs-only commits confirm flake is diff-independent. Failing assertion is whatsapplink_exec_test.go:121: got lines [], want 3 JSON lines (qr, pairing_accepted, paired). Test elapsed 0.00s. Package kernel/httpapi fails in 5.140s. make target test-portable (Makefile:135) exits 1."
  implication: "The test's line-collection path can complete with zero lines — a synchronization gap between spawning the exec subprocess (or fake) and reading its streamed output, not an environment or dependency issue."

- timestamp: 2026-08-12
  checked: "standalone repro (scratchpad/race) replicating newExecLinkSpawner's exact shape: StdoutPipe + scanner goroutine + concurrent cmd.Wait goroutine, 400 iterations under GOMAXPROCS=1 taskset -c 0"
  found: "ok(3 lines)=206 empty=194 partial=0; scanner.Err() on every one of the 194 empty runs was 'read |0: file already closed'"
  implication: "RC1 proven: the pipe is closed by cmd.Wait() under the reader. Not EOF, not an exec/ETXTBSY failure, not a fixture problem. 48.5% per-iteration failure rate under starvation."

- timestamp: 2026-08-12
  checked: "kernel/httpapi/whatsapplink.go:67-73 comment vs the code at 162-202"
  found: "The stderr comment explicitly names 'cmd.StderrPipe() read concurrently with Wait' as 'the same incorrect-usage pattern already flagged on the stdout side' and declines to add a second instance — while the stdout side it refers to still HAS that exact pattern."
  implication: "The defect was known and documented in prose at the adjacent site but never actually removed from the stdout path. A comment asserting a hazard is not a fix (KB-004's own lesson about confident lifecycle prose)."

- timestamp: 2026-08-12
  checked: "standalone repro (scratchpad/kill) with the exact `sleep 30` fixture: ps before kill, cmd.Wait timing after kill, cmd.Stderr as a non-*os.File"
  found: "ps showed sleep(183301) PPID=183299(sh) before kill, re-parented to PPID 1 after. cmd.Wait did NOT return within 5s in 2/2 runs, with and without a trailing `exit 0`. Local /bin/sh -> bash."
  implication: "RC2 proven and independent of RC1: the orphaned grandchild inherits the stderr pipe write end, so os/exec's stderr copy goroutine never reaches EOF and Wait blocks for the orphan's full lifetime. The existing kill test passes only when the SIGKILL beats the shell's fork."

- timestamp: 2026-08-12
  checked: "gh run view 31598316920 --log-failed"
  found: "Both TestExecLinkSpawner_StreamsLinesInOrderAndExitsClean (0.00s) AND TestExecLinkSpawner_KillTerminatesLongRunningSubprocess (3.00s) failed in the same CI run."
  implication: "Confirms two distinct mechanisms rather than one. The originally-reported symptom captured only RC1; a single-cause fix would have left the gate flaky."

- timestamp: 2026-08-12
  checked: "kernel/httpapi/whatsapplink.go:294-317 (linkSession.consume), the production consumer of lines/done"
  found: "consume ranges over lines; with zero lines it falls through to the generic 'link subprocess exited unexpectedly before reporting an outcome' error. It then blocks on <-done."
  implication: "Both defects are PRODUCTION bugs, not test-only artefacts. RC1 destroys a fast-failing plugin's actionable {\"kind\":\"error\"} diagnostic and replaces it with the generic message. RC2 blocks consume on <-done indefinitely after a cancel/reap, while an orphan keeps holding the whatsmeow store that T-08-07 requires be exclusively held."

- timestamp: 2026-08-12
  checked: "knowledge-base.md KB-004 lines 388-397"
  found: "Documents 'Hand it an *os.File — passed straight to the child as an fd, neither problem', and records that whatsapplink.go and pluginhost/host.go 'both intentionally leave SysProcAttr nil so they share the kernel's process group'."
  implication: "Prescribes the RC1 fix shape and rules out the obvious RC2 fix (Setpgid + process-group kill), which would remove the subprocess from the kernel's group and regress KB-004's own signal-reaping fix. cmd.WaitDelay is therefore the correct RC2 lever."

## Eliminated

- hypothesis: "The subprocess never ran or failed to exec (e.g. ETXTBSY on the just-written fixture), so it genuinely produced no output"
  evidence: "scanner.Err() was os.ErrClosed ('file already closed') on 194/194 empty runs, never nil/EOF, and cmd.Start() returned no error. An exec failure would surface at Start and fail the test at 'spawn: %v' instead."
  timestamp: 2026-08-12

- hypothesis: "The test itself is wrong (a test-only synchronisation bug) and production is fine"
  evidence: "The defect is in newExecLinkSpawner, the production spawner wired at routes.go:114; linkSession.consume is exposed to the identical race. The tests are correct — they assert the documented contract."
  timestamp: 2026-08-12

- hypothesis: "CI environment/runner differences (2 cores, dash vs bash, container) are the cause"
  evidence: "Both failures reproduce on the dev box under induced starvation (GOMAXPROCS=1, taskset -c 0). The runner only changes the sampling rate of a latent, always-present race."
  timestamp: 2026-08-12

- hypothesis: "Both CI symptoms share one root cause"
  evidence: "RC1 is a pipe-close-under-reader race on the clean-exit path (0 lines, 0.00s); RC2 is Wait blocking on an orphan-held stderr pipe on the kill path (3.00s timeout). They reproduce independently and need separate fixes."
  timestamp: 2026-08-12

## Resolution

root_cause: >
  TWO independent defects in newExecLinkSpawner (kernel/httpapi/whatsapplink.go), both latent since
  the spawner was written and both surfaced only by scheduler variance — which is why docs-only
  commits failed and why the same commit content both passed and failed.

  RC1 — cmd.Wait() ran concurrently with a bufio.Scanner reading cmd.StdoutPipe(). os/exec owns that
  pipe and closes it inside Wait as soon as the child exits, which os/exec documents as incorrect
  usage ("it is thus incorrect to call Wait before all reads from the pipe have completed"). When the
  Wait goroutine won the race, the scanner's first Read returned os.ErrClosed, Scan() returned false
  having emitted nothing, and the lines channel closed empty — the reported
  'got lines [], want [...]' in 0.00s. Measured at 194/400 iterations (48.5%) under a starved
  scheduler, with scanner.Err() == 'read |0: file already closed' on 194/194.

  RC2 — exec.CommandContext's cancel signals exactly one pid, and SysProcAttr is deliberately nil at
  this site (KB-004: link subprocesses share the kernel's process group). Any process the subprocess
  forks therefore survives the kill still holding the stderr pipe's write end it inherited. Because
  cmd.Stderr is a non-*os.File, cmd.Wait waits for os/exec's stderr copy goroutine to reach EOF, and
  that EOF cannot arrive while the orphan lives — so Wait, done, and linkSession.consume's terminal
  event were all parked for the orphan's full 30s lifetime, failing the kill test at its 3.00s bound.

fix: >
  RC1 — the spawner now creates its own os.Pipe and assigns the write end to cmd.Stdout. os/exec
  passes a caller-supplied *os.File straight to the child as an fd: it creates no pipe, starts no
  copy goroutine, and never closes our read end, so reads and Wait become genuinely independent
  rather than merely ordered (KB-004's "hand it an *os.File"). The parent's copy of the write end is
  closed after Start so EOF still arrives, and a small watcher goroutine closes the read end on
  cancellation — that is what makes lines close promptly on kill, since a scanner blocked on a read
  an orphan holds open would otherwise never see EOF. The watcher exits on scanner completion rather
  than parking on ctx.Done() forever, so it does not become the per-spawn goroutine leak KB-003 warns
  amplifies later flakes.

  RC2 — cmd.WaitDelay = linkSubprocessWaitDelay (1s) bounds Wait once the process itself is gone;
  os/exec force-closes the pipes and returns rather than waiting out an orphan. A process-group kill
  was rejected: Setpgid would remove link subprocesses from the kernel's process group and regress
  KB-004's own signal-reaping fix.

  Comments whose premise WaitDelay invalidates (stderrLineLogger.flush's "no concurrent Write can
  race it") were corrected rather than left asserting a guarantee the code no longer provides —
  KB-004's "a doc comment asserting a lifecycle guarantee is not evidence of one".

verification:
  signal_1_regression_test_bites: >
    PASS (mutation-checked). Two new deterministic tests fail 10/10 against the unfixed spawner with
    exactly the predicted signatures — "got lines [{"kind":"qr"}], want [3 lines]" (the single line an
    unbuffered channel send can hold in flight) and the 3.00s Wait block — and pass with the fix.
    Re-verified after the fixture was changed for process hygiene.
  signal_2_original_repro_fixed: >
    PASS. The exact condition that produced the failures (GOMAXPROCS=1, taskset -c 0) previously gave
    48.5% empty-line failures plus repeated 3.00s kill failures; it now passes 100 consecutive
    iterations of all 8 exec-spawner tests (150 more before the final fixture tweak). Zero failures.
  signal_3_addresses_root_cause_not_symptom: >
    PASS. No sleeps, retries, timeouts-widened, or assertions weakened. Both changes remove the
    mechanism: the pipe can no longer be closed under the reader, and Wait can no longer be held by a
    process the kernel never tracked. Not a deletion-only diff.
  signal_4_revert_restores_bug: >
    PASS. Stashing only kernel/httpapi/whatsapplink.go (tests untouched) reproduces both failures
    deterministically, 10/10.
  signal_5_no_regressions: >
    PASS. make test-portable green across all 8 workspace modules; go test -race ./kernel/httpapi
    -count=2 clean; go vet clean; gofmt clean. Process hygiene verified: the new fixture's forked
    child is reaped via a recorded pid, measured delta of 0 stray processes across 5 runs.
  guardrail_verdict: accepted
  oracle_type: >
    derived (contract) — both new tests assert os/exec's documented pipe-lifetime contract rather
    than a crash or an incidental value, so they pin behaviour the implementation must provide, not
    the shape it currently has. Boundary neighbours covered: zero lines vs all three (RC1's failure
    is precisely a truncation to 0 or 1), and the with-orphan / without-orphan pair for RC2 (the
    pre-existing kill test covers the no-orphan side).

files_changed:
  - "kernel/httpapi/whatsapplink.go: own the stdout pipe instead of cmd.StdoutPipe(); close the read end on cancellation via a watcher that exits on scanner completion; add cmd.WaitDelay = linkSubprocessWaitDelay (1s); correct two comments whose premise WaitDelay changes"
  - "kernel/httpapi/whatsapplink_exec_test.go: add TestExecLinkSpawner_DeliversBufferedOutputWhenDrainedAfterSubprocessExit and TestExecLinkSpawner_KillYieldsDoneWhenSubprocessLeftAForkedChild (deterministic guards for RC1 and RC2), plus drainLinesWithin/asStrings helpers"
