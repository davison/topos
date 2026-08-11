---
status: resolved
trigger: "TODO.md kernel v1.0 item: BUG - not reaping plugin sub-processes on some exit path"
created: 2026-08-11T12:10:00Z
updated: 2026-08-11T14:45:00Z
---

## Symptoms

DATA_START
expected: When the kernel exits — including Ctrl-C on `make dev` — every topos-plugin-* subprocess it launched terminates with it. No live plugin processes survive the kernel.

actual: Orphaned LIVE plugin processes (not zombies) observed still running after the kernel exited. The operator killed them manually and has not seen them re-appear since, so the leak is path-specific or intermittent — TODO.md's own wording is "on some exit path".

errors: none recorded

timeline: Most recent sighting 2026-08-09, early hours. Originally noticed by a prior Claude session during an earlier phase (observation was conversational — grep of git log and .planning for orphan/reap fixes finds only sync_runs-row fixes, so NO committed fix exists; the operator allows it may have been fixed since, but there is no evidence of that). Kernel has since gained: link-session instance suspend/resume (08-03/08-09), config hot-apply host rebuilds (phase 7), and the 08-13 supervisor genMu/pluginhost lock split.

reproduction: |
  Unconfirmed. Operator association: kernel shutdown via Ctrl-C (interrupting `make dev`).
  Candidate exit paths to check, in rough priority order:
  1. SIGINT/SIGTERM on the kernel (does cmd/topos install a signal handler that reaches supervisor/pluginhost Shutdown? does `make dev`'s process wrapping change signal delivery to the kernel vs the vite dev server?)
  2. Kernel death without handler (panic, os.Exit, log.Fatal paths) — go-plugin subprocesses only die on their own if the plugin detects the closed stdio/broker connection
  3. Config hot-apply / Reconcile replacing instances (are old instances' Kill() always reached on every early-return/error branch?)
  4. WhatsApp link-session suspend/resume spawning the link subprocess (is the link-mode child reaped on session timeout/error teardown?)

context: |
  - kernel/pluginhost/host.go:669 `Host.Shutdown()` kills every launched plugin; `Plugin.Kill()` wraps go-plugin `client.Kill()` (waits on client's own wait, host.go:326 comment).
  - hashicorp/go-plugin child processes do NOT automatically exit when the parent dies unless the client kills them or the plugin side watches the RPC connection; an unclean parent exit (signal not handled, SIGKILL, panic without deferred Shutdown) orphans them by design.
  - Relevant recent surface: kernel/supervisor/supervisor.go (Apply/Reconcile/suspend-resume closures), kernel/httpapi/whatsapplink.go (link subprocess), cmd/topos/main.go (signal handling, if any).
  - Constraint: fix must not break `make dev`'s dev-guard smoke cases (`make dev-check`) or the e2e harness's hermetic kernel lifecycle (web/e2e), both of which start/stop the kernel programmatically.
  - This bug is deliberately routed through /gsd-debug instead of a roadmap phase (operator decision 2026-08-11, recorded in STATE.md Roadmap Evolution).
DATA_END

## Current Focus

bug_class: Bohrbug (deterministic given the exit path — the "intermittent" framing is an artifact of an external masking factor, not non-determinism)

hypothesis: |
  Orphaning requires TWO conditions simultaneously (AND-gate):
  (A) CODE: cmd/topos/main.go installs no signal handler at all — `signal.Notify`
      appears nowhere in the Go tree. runServe() ends in a blocking
      `http.ListenAndServe`, so `defer sup.Shutdown()` (kills every
      pluginhost child) and `defer linkStore.Shutdown()` (kills every
      whatsapp-link child) are UNREACHABLE when a signal terminates the
      process with default disposition.
  (B) DEPENDENCY BEHAVIOUR: hashicorp/go-plugin's server side deliberately
      swallows SIGINT (server.go:460-473 "Eat the interrupts"), and has no
      parent-death watchdog — Serve() only returns on ctx.Done() (test mode
      only) or doneCh (client-issued Shutdown). So a terminal Ctrl-C, which
      the tty delivers to the whole foreground process group, kills the
      kernel but is IGNORED by every plugin child.
  Neither alone orphans: with (A) fixed the host reaps them; without (B)
  the group-wide SIGINT would incidentally reap them and mask (A).

  MASKING FACTOR (explains "on some exit path" / non-recurrence): Makefile:209/230
  `trap 'kill 0' INT TERM` (added 6d0e6a8, 2026-08-05) sends SIGTERM — which
  go-plugin does NOT ignore — to the whole process group after Ctrl-C. Since
  no kernel code sets Setpgid/Setsid, children share the kernel's pgid, so
  `make dev` Ctrl-C incidentally reaps them. Exit paths with no such trap
  (direct `./bin/topos serve` + Ctrl-C; `kill <kernel-pid>` as in
  scripts/signal-readonly-smoke.sh:62, which signals the PID not the group)
  still orphan.

test: "DONE — T1..T4 signal matrix against the built kernel + mock plugin (see Evidence 14:05)."
expecting: "DONE — matched the prediction exactly: T1/T2/T4 orphan (ppid=1), T3 reaps, zero shutdown log lines in all four."
next_action: "Apply the fix: install a SIGINT/SIGTERM handler in cmd/topos so the deferred cleanups become reachable, then re-run the matrix as the regression gate."

status_note: "Root cause CONFIRMED. Proceeding to fix_and_verify."

reasoning_checkpoint:
  hypothesis: "Plugin subprocesses are orphaned because cmd/topos installs no signal handler, so SIGINT/SIGTERM kill the kernel with the Go runtime's default disposition and the deferred `linkStore.Shutdown()`/`sup.Shutdown()` in runServe never run; go-plugin children neither die with their parent nor honour SIGINT (they explicitly 'eat the interrupts'), so nothing else reaps them."
  confirming_evidence:
    - "Direct observation: `signal.Notify` has zero occurrences in first-party Go code (whole-repo grep)."
    - "Direct observation: runServe ends in a blocking `http.ListenAndServe` (main.go:201), so the three defers are only reachable via listener error."
    - "Direct observation (T1/T2/T4): plugin child survives with ppid=1 after the kernel dies — a live orphan, matching the reported symptom exactly (not a zombie)."
    - "Direct observation: kernel logged ZERO shutdown/kill/closing lines in all four matrix rows, proving Host.Shutdown never executed even on the row where the child happened to die."
    - "Direct source read of go-plugin@v1.8.0 server.go:460-473 ('Eat the interrupts') plus the absence of any parent-death watchdog in Serve()'s select (server.go:503-525)."
  falsification_test: "After installing the handler, re-run T1/T2/T4. If any row still shows the plugin child alive with ppid=1, or still logs zero shutdown lines, the hypothesis is wrong (cleanup would be failing for some reason other than unreachability)."
  fix_rationale: "The root cause is that the cleanup code is UNREACHABLE, not that it is wrong — Host.Shutdown/Plugin.Kill are already correct and already kill every launched child. So the fix is exactly to make them reachable on the signal paths: catch SIGINT/SIGTERM, unblock the serve loop, and return normally so the existing defers run in their existing (already-correct) order linkStore -> supervisor -> index store. Nothing about the killing logic itself needs to change, which keeps the change minimal and low-risk."
  blind_spots:
    - "SIGKILL, panic, and OOM-kill remain uncoverable by any in-process handler — those paths will still orphan. Deliberately NOT addressed here (see rejected_alternative below)."
    - "Tested only with topos-plugin-mock. The whatsapp link subprocess (linkStore) path is fixed by the same reachability change but is not directly exercised by the regression test."
    - "Repro ran on Linux only; signal semantics differ on Windows (syscall.SIGTERM is not delivered there), so the handler must not assume POSIX-only build."
  rejected_alternative: "Setting SysProcAttr.Pdeathsig=SIGKILL on plugin cmds as defence-in-depth was considered and REJECTED: in Go, Pdeathsig fires when the spawning OS THREAD exits, not the process, and the runtime freely migrates goroutines between threads — so without runtime.LockOSThread it would spuriously kill healthy plugins. That is a worse failure than the bug being fixed."
  candidate_causes:
    - "code: cmd/topos installs no signal handler; runServe blocks in ListenAndServe so deferred cleanup is dead code on signal paths"
    - "dependency/environment: go-plugin's server deliberately ignores SIGINT and has no parent-death watchdog, so children survive a group-wide Ctrl-C"
    - "build/tooling (masking, not causal): Makefile's `trap 'kill 0'` converts make-dev Ctrl-C into a process-group SIGTERM, which go-plugin does NOT ignore — hiding the defect on the most-travelled path since 2026-08-05"
  and_gate: "YES — genuine AND. Orphaning needs (A) the kernel skipping cleanup AND (B) the child not dying on its own. Proven by the T2-vs-T3 differential: same missing handler, same dead kernel, but the child survives SIGINT (B holds) and dies on SIGTERM (B does not hold). Fixing (A) alone is sufficient to close the bug because (A) is the condition topos owns; (B) is upstream-by-design and must be treated as a fixed constraint."

## Evidence

- timestamp: 2026-08-11T12:05:00Z
  checked: "git log --grep orphan/reap (all branches) and .planning grep for a prior fix"
  found: "Only sync_runs-row orphan fixes (f01df96) and unrelated 'orphaned decisions' prose; no process-reaping fix ever committed"
  implication: "The bug is almost certainly still present; absence of re-sightings since Aug 9 reflects the operator killing the stragglers and the trigger path not recurring, not a fix"

- timestamp: 2026-08-11T13:55:00Z
  checked: "grep -rn 'signal.Notify|signal.NotifyContext|os/signal' --include=*.go over the whole repo"
  found: "ZERO occurrences in first-party code. cmd/topos/main.go installs no signal handler of any kind."
  implication: "SIGINT/SIGTERM terminate the kernel with the Go runtime's default disposition. Go does NOT run deferred functions on signal death, so runServe's `defer sup.Shutdown()` (main.go:178) and `defer linkStore.Shutdown()` (main.go:193) are unreachable on every signal path. Condition (A) of the AND-gate is confirmed by static reading."

- timestamp: 2026-08-11T13:56:00Z
  checked: "cmd/topos/main.go runServe() control flow"
  found: "runServe ends `return http.ListenAndServe(listen, router)` (main.go:201) — a call that blocks forever and only returns on a listener error. There is no other exit path from the function in normal operation."
  implication: "The three deferred cleanups are only ever reached if the LISTENER fails. Normal operator shutdown (any signal) can never reach them. This is not an 'early-return branch skips Kill()' bug (symptom hypothesis 3/4) — the entire deferred-cleanup block is dead code under signal termination."

- timestamp: 2026-08-11T13:57:00Z
  checked: "hashicorp/go-plugin@v1.8.0 server.go signal handling and Serve() exit conditions"
  found: "server.go:460-473 `if opts.Test == nil { signal.Notify(ch, os.Interrupt); go func(){ for { <-ch; logger.Trace(\"plugin received interrupt signal, ignoring\") } }() }` — comment reads 'Eat the interrupts.' Serve() then blocks on `select { case <-ctx.Done(): ...; case <-doneCh: }` (server.go:503-525) where ctx is non-background ONLY in test mode. There is no stdin-EOF watch and no parent-death watchdog anywhere in the package."
  implication: "Condition (B) confirmed: plugin children deliberately ignore SIGINT and have NO self-termination path when the parent dies. They exit only when the host issues the Shutdown RPC / Kill. A terminal Ctrl-C therefore cannot incidentally reap them."

- timestamp: 2026-08-11T13:58:00Z
  checked: "grep -rn 'SysProcAttr|Setpgid|Setsid|Pdeathsig' kernel/"
  found: "No occurrences. kernel/pluginhost/host.go:392 `cmd := exec.Command(binPath)` and kernel/httpapi/whatsapplink.go:146 `exec.CommandContext(spawnCtx, binPath, \"-link-json\", ...)` both leave SysProcAttr nil."
  implication: "Children inherit the kernel's process group. So a GROUP-directed signal does reach them — which is what makes the masking factor work, and is why the bug looked intermittent."

- timestamp: 2026-08-11T14:02:00Z
  checked: "FALSE START, recorded so it is not repeated: first repro run launched the kernel as a bash background job (`topos serve &`) from a non-interactive shell and reported 'KERNEL SURVIVED SIGINT'."
  found: "/proc/<pid>/status showed SigIgn=0 and SigCgt=fffffffd7fc1feff (the ordinary Go-runtime catch-all mask), i.e. nothing in topos ignores SIGINT. The survival was a HARNESS ARTIFACT: POSIX requires a non-interactive shell to set SIGINT/SIGQUIT to SIG_IGN for background jobs, and the Go runtime preserves a disposition it inherits as already-ignored."
  implication: "Any repro of this bug MUST reset the SIGINT disposition before exec (harness uses `setsid perl -e '$SIG{INT}=\"DEFAULT\"; exec @ARGV'`). Reading /proc SigIgn/SigCgt is the cheap way to detect the artifact. Do not conclude 'the kernel ignores SIGINT' — it does not."

- timestamp: 2026-08-11T14:05:00Z
  checked: "Signal matrix against the real built kernel + topos-plugin-mock, SIGINT disposition reset to SIG_DFL, kernel in its own session via setsid, plugin child identified by `pgrep -P <kernel>` scoped to the scratchpad plugins dir"
  found: |
    T1  SIGINT  -> kernel PID only  : kernel died, plugin ALIVE ppid=1  ORPHAN
    T2  SIGINT  -> process group    : kernel died, plugin ALIVE ppid=1  ORPHAN  (true Ctrl-C simulation)
    T3  SIGTERM -> process group    : kernel died, plugin REAPED
    T4  SIGTERM -> kernel PID only  : kernel died, plugin ALIVE ppid=1  ORPHAN
    In ALL FOUR cases the kernel logged ZERO shutdown/kill/closing lines.
  implication: |
    Root cause confirmed and fully deterministic — 3 of the 4 exit paths orphan.
    The zero-shutdown-log line in every case proves the deferred cleanups never ran
    even in T3: T3 reaps by ACCIDENT (the child receives the group SIGTERM directly,
    and go-plugin only swallows SIGINT, not SIGTERM), not because the host cleaned up.
    T2-vs-T3 is the decisive differential isolating condition (B).

- timestamp: 2026-08-11T14:06:00Z
  checked: "Which real exit paths map onto which matrix row"
  found: |
    - `make dev` Ctrl-C  -> Makefile:209/230 `trap 'kill 0' INT TERM` fires a
      process-group SIGTERM = T3 = the ONLY non-orphaning row. Added 2026-08-05
      in 6d0e6a8 ("guard make dev against stale-port and dead-kernel footguns"),
      for unrelated port-guard reasons.
    - bare `./bin/topos serve` + Ctrl-C in a terminal -> T2 -> ORPHANS.
    - `kill <kernel-pid>` / pkill / a systemd-style stop -> T4 -> ORPHANS.
      scripts/signal-readonly-smoke.sh:62 `kill "${SERVER_PID:-}"` is exactly T4.
    - web/e2e/fixtures/kernel.ts:239 `process.kill(-pid,'SIGTERM')` is T3 — safe by luck.
  implication: |
    This is the complete explanation of the TODO's 'on some exit path' wording and of
    why the operator stopped seeing orphans after 2026-08-09 without any fix landing:
    the dominant day-to-day exit path (make dev Ctrl-C) was incidentally masked by a
    port-guard commit five days earlier. The defect never went away; it stopped being
    visible on the one path the operator exercises most.

## Eliminated

- hypothesis: "Candidate 3 from the symptom report — config hot-apply / Reconcile replacing instances misses Kill() on some early-return/error branch"
  evidence: "Not the reported defect. The orphan reproduces deterministically with a kernel that never receives a config change (T1/T2/T4 signal the kernel while it sits idle after boot), so a Reconcile branch cannot be the cause of THIS symptom. Reconcile's kill path is separately covered by kernel/pluginhost/reconcile_test.go."
  timestamp: 2026-08-11T14:05:00Z

- hypothesis: "Candidate 4 from the symptom report — the WhatsApp link subprocess is not reaped on session timeout/error teardown"
  evidence: "Not the reported defect, for the same reason: the repro orphans a pluginhost go-plugin child with no link session ever created. linkStore.Shutdown was ALSO unreachable for the identical root cause and is fixed by the same change, so the link path is repaired too — but it was not the mechanism."
  timestamp: 2026-08-11T14:05:00Z

- hypothesis: "The kernel ignores SIGINT (first repro run reported 'KERNEL SURVIVED SIGINT')"
  evidence: "Refuted by /proc/<pid>/status: SigIgn=0000000000000000, SigCgt=fffffffd7fc1feff (ordinary Go-runtime catch-all). The survival was a harness artifact — a non-interactive shell sets SIGINT to SIG_IGN for background jobs and Go preserves an inherited ignore. With the disposition reset before exec, the kernel dies from SIGINT normally."
  timestamp: 2026-08-11T14:02:00Z

- hypothesis: "Plugin children are put in their own process group (which would explain why a group-directed kill misses them)"
  evidence: "grep -rn 'SysProcAttr|Setpgid|Setsid|Pdeathsig' kernel/ returns nothing; both spawn sites leave SysProcAttr nil, so children inherit the kernel's pgid. Confirmed empirically by T3, where a group SIGTERM does reach and kill the child."
  timestamp: 2026-08-11T13:58:00Z

## Resolution

root_cause: |
  Two conditions had to hold together (AND-gate), and both did:

  (A) cmd/topos installed NO signal handler — `signal.Notify` appeared nowhere
      in first-party code. runServe ended in a bare, permanently-blocking
      `http.ListenAndServe`, so SIGINT/SIGTERM terminated the kernel with the
      Go runtime's default disposition. Go does not run deferred functions on
      signal death, which made `defer linkStore.Shutdown()` and
      `defer sup.Shutdown()` — the code that actually kills every plugin
      subprocess — unreachable on every ordinary operator exit path.
      (linkStore.Shutdown's own doc comment already CLAIMED "called once, from
      kernel shutdown (cmd/topos/main.go), so a Ctrl-C never orphans a linking
      process" — a guarantee the code could never actually deliver.)

  (B) hashicorp/go-plugin children do not die with their parent, by design:
      the plugin side explicitly swallows SIGINT (server.go:460-473, "Eat the
      interrupts") and Serve() has no parent-death watchdog. So a terminal
      Ctrl-C, delivered by the tty to the whole foreground process group,
      reached the children and was ignored by them.

  The cleanup code itself was never wrong — it was unreachable. This was NOT
  an early-return branch skipping Kill() (the original hypotheses 3 and 4).

  Why it looked intermittent ("on some exit path"): Makefile:209/230
  `trap 'kill 0' INT TERM`, added 2026-08-05 in 6d0e6a8 for unrelated
  port-guard reasons, converts a `make dev` Ctrl-C into a process-GROUP
  SIGTERM. go-plugin does not swallow SIGTERM, so on that one path the
  children died of their own signal and the defect was invisible — which is
  why the operator stopped seeing orphans after 2026-08-09 without any fix
  landing. Every other path still orphaned.

fix: |
  cmd/topos/main.go — make the existing teardown reachable on signal paths.
  - runServe: run `http.Server.ListenAndServe` in a goroutine and select over
    {listener error, SIGINT/SIGTERM}. On signal: `signal.Reset` first (so a
    second Ctrl-C still hard-kills a kernel the operator believes is hung),
    graceful `srv.Shutdown` bounded by serverShutdownTimeout (10s), then
    return nil so the existing defers run in their existing, already-correct
    order: linkStore.Shutdown -> sup.Shutdown -> store.Close -> cancel.
    A drain failure is logged and NOT returned, so it can never skip the
    plugin teardown.
  - runSync: `signal.NotifyContext` so a Ctrl-C mid-sync cancels RefreshAll
    and reaches the same `defer sup.Shutdown()`. Safe against KB-001 (the
    coordinator already finalises sync_runs on a detached
    context.WithoutCancel, so cancellation cannot strand the row).
  - The boot context's lifecycle is deliberately UNCHANGED: no Shutdown path
    takes a ctx, and supervisor.stopScheduler already self-bounds via its own
    cancel, so nothing needed re-wiring.

  Deliberately NOT done: SysProcAttr.Pdeathsig on plugin cmds. In Go it fires
  when the spawning OS THREAD exits and the runtime migrates goroutines
  between threads, so without runtime.LockOSThread it would spuriously kill
  healthy plugins — a worse failure than the bug. SIGKILL/panic/OOM therefore
  remain uncoverable and still orphan; this is stated in the code comment.

verification:
  oracle_type: specified ("no plugin subprocess survives kernel exit")
  guardrail_verdict: accepted
  signal_1_revert_test: |
    PASS — differential on the same harness. Pre-fix binary: T1/T2/T4 leave the
    child alive at ppid=1; post-fix binary: all four rows reap. Independently
    re-confirmed by stashing the fix and re-running the automated test.
  signal_2_regression_test: |
    PASS — cmd/topos/shutdown_signal_test.go
    TestServeReapsPluginSubprocessesOnShutdownSignal. Mutation-checked against
    the unfixed kernel: 3 of 4 rows FAIL deterministically across 3 runs
    (state "S", reparented), while the "SIGTERM to process group" row PASSES
    even unfixed — proving the 2x2 boundary matrix was necessary and that a
    single-case test written against the `make dev` path would have been green
    against the live bug. Asserts a precondition (a plugin child must exist
    before signalling) so it cannot pass vacuously, and distinguishes a live
    orphan from a zombie, matching the reported symptom exactly.
  signal_3_diff_shape: "PASS — additive (93 insertions, 2 deletions); no deletion-only 'fix'."
  signal_4_existing_gates: |
    PASS — make test-portable (all 8 modules), make dev-check (all 3
    dev-guard cases), make e2e (42/42 chromium specs incl. the hermetic
    kernel lifecycle), go vet, gofmt clean, -race clean.
  signal_5_root_cause_not_symptom: |
    PASS — the fix restores reachability of the existing, already-correct
    teardown rather than adding a second kill path.
  stability: "15 consecutive runs of the regression test: 0 failures."
  manual: "`topos sync` completes normally, logs 'plugin process exited', leaves no plugin process behind."
  human_verify: |
    ACCEPTED 2026-08-11 — operator elected to commit on automated evidence.
    The 4-row signal matrix, the mutation check against the unfixed kernel, and
    the full gate suite (test-portable / dev-check / e2e) were accepted as
    sufficient; no manual reproduction was requested.
    Operator note: an early over-broad pgrep/kill during investigation left the
    live `make dev` kernel plugin-less; operator was told to restart it. Not a
    consequence of the fix.

commit: 799200b

files_changed:
  - "cmd/topos/main.go: signal handling for serve (select over listener/signal + graceful drain) and sync (NotifyContext)"
  - "cmd/topos/shutdown_signal_test.go: NEW — 2x2 {SIGINT,SIGTERM} x {process-group,pid} regression gate"

test_harness_lessons:
  - "A background job started from a NON-INTERACTIVE shell inherits SIGINT=SIG_IGN (POSIX), and the Go runtime preserves an inherited ignore — an early repro wrongly concluded 'the kernel survives SIGINT'. Reset the disposition before exec, and check /proc/<pid>/status SigIgn to detect it."
  - "/proc/<pid>/task/<tid>/children lists that THREAD's children, not the process's. Go fork/execs from an arbitrary OS thread, so reading only task/<pid>/children finds the subprocess or misses it depending on the scheduler. The union over all task dirs is required — this made an early version of the test fail on a random row each run."
  - "exec.Cmd with a strings.Builder for Stdout/Stderr is a data race against exec's copier goroutine and silently loses output on a timeout path; an *os.File is handed straight to the child as an fd and has neither problem."
