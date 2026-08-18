---
status: awaiting_human_verify
trigger: "CI flake (~50% of runs) in make test-portable: TestServeReapsPluginSubprocessesOnShutdownSignal/SIGINT_to_kernel_pid — plugin subprocess survived kernel exit in state S, reparented to ppid 1; kernel produced no output"
created: 2026-08-18T00:00:00Z
updated: 2026-08-18T02:00:00Z
---

## Current Focus

bug_class: Heisenbug (timing-dependent startup-ordering race; probability scales with scheduler starvation — 2-core CI runner vs 12-core dev box)

known_pattern_candidate: "KB-004 — defer behind a blocking call is dead code on every signal path (plugin-procs-orphaned-on-exit). This recurrence looks like the STARTUP-WINDOW residue of that same class: the fix installed signal.Notify, but installed it AFTER the plugin subprocesses are spawned."

hypothesis: |
  H1 (primary): In runServe (cmd/topos/main.go), signal.Notify(sigCh, ...) at line 489
  executes AFTER supervisor.NewSupervisor at line 436 — which launches the plugin
  subprocesses during construction. There is therefore a real window (plugin child
  exists, handler not yet installed) where SIGINT/SIGTERM to the kernel pid kills the
  kernel with the Go runtime's default disposition: no defers run, no output emitted,
  plugin orphaned. The test's precondition loop polls /proc every 50ms and signals the
  INSTANT a child appears — on a starved 2-core CI runner the spawn→Notify window
  (go-plugin handshake + boot RPCs + scheduler start + Router construction) is long
  enough that the signal frequently lands inside it.

  Why only the "SIGINT to kernel pid" row fails: in the group-SIGINT row the CHILD is
  also young enough to die of the group SIGINT (go-plugin's "eat the interrupts"
  handler is itself not yet installed in a just-spawned child), so no orphan survives
  to assert on; group-SIGTERM always kills the child; SIGTERM-to-pid (row 4) shares
  the window but runs as the 4th launch when everything is warm/faster.

  Note: runSync is NOT exposed — it registers signal.NotifyContext at its very top,
  BEFORE NewSupervisor. runServe is the only path with the inversion.

test: |
  1. Read kernel/supervisor to confirm NewSupervisor spawns plugins synchronously and
     what (if anything) it logs at Info during a healthy boot — must be consistent
     with "kernel produced no output".
  2. git log 799200b..HEAD -- cmd/topos kernel/supervisor kernel/pluginhost to see
     whether the ordering changed or the window merely widened.
  3. Reproduce locally: pin to 1-2 CPUs (taskset) and run
     go test ./cmd/topos -run 'TestServeReapsPluginSubprocessesOnShutdownSignal/SIGINT_to_kernel_pid' -count=20
     Expect failures at a material rate with the exact CI signature (state "S",
     ppid 1, kernel produced no output). Also capture kernel exit status in a probe:
     default-disposition death = "signal: interrupt", graceful = exit 0.

expecting: |
  If H1 is true: failing iterations show empty kernel log + orphan, and the kernel's
  wait status is death-by-SIGINT (not exit 0). If instead the kernel log contains
  "shutdown signal received" on failing runs, H1 is FALSE (handler was installed and
  teardown itself is broken — investigate supervisor Shutdown instead).

next_action: "DONE — fix applied and guardrail-accepted (see Resolution). Awaiting human verification: operator to confirm make test-portable stays green across CI runs on the real 2-vCPU runner."

reasoning_checkpoint:
  hypothesis: "runServe installs signal.Notify AFTER supervisor.NewSupervisor has spawned the plugin subprocesses, so a SIGINT/SIGTERM landing in the spawn->Notify window kills the kernel with the Go runtime's default disposition: no defers run, no output is emitted, and any plugin child that got its go-plugin handshake line written before the kernel's pipes closed survives as a live orphan (state S, ppid 1). The test signals at the first observable child, which on a starved 2-vCPU CI runner frequently lands inside that window."
  confirming_evidence:
    - "Direct observation (probe, 1ms poll, 15 iterations): 15/15 kernels died with wait status 'signal: interrupt' (default disposition), 0 log bytes, no 'shutdown signal received' line — the handler was demonstrably NOT installed at the moment a plugin child first exists."
    - "Direct observation (same probe, iteration 08): child 1561576 survived in state S, ppid 1 — byte-for-byte the CI failure signature, including '<kernel produced no output>'."
    - "Differential (probe, 500ms delay after child appears): 5/5 kernels exited 0, logged 'shutdown signal received', all children reaped — teardown itself is intact; only its reachability during the startup window is broken."
    - "Static reading: signal.Notify at main.go:489 vs NewSupervisor (spawns plugins) at main.go:436; runSync by contrast registers NotifyContext at its top (main.go:354) BEFORE NewSupervisor, so the sync path has no such window."
    - "git show 0010698 (the original reap fix): its runServe hunk starts at old-line 197 — the end of the function — so the Notify-after-spawn ordering has existed since the fix landed. Latent, not introduced by 14-01 (which only added flag parsing before runServe)."
  falsification_test: "If failing CI runs showed 'shutdown signal received' in the kernel log, or the kernel's wait status were exit 0 while a child survived, the hypothesis is wrong (teardown itself would be broken). Probe shows the opposite on every window-hit iteration."
  fix_rationale: "The teardown and the signal handler are both correct; the handler is installed too late — after children who cannot outlive-proof themselves already exist. Moving registration to the top of runServe (before setup/NewSupervisor, mirroring runSync's existing order) closes the window: a signal arriving mid-boot is buffered in sigCh, boot completes, the select drains it immediately and the existing defers reap everything. No child can exist before the handler does."
  blind_spots:
    - "SIGKILL/panic/OOM during boot still orphan — unchanged, documented limit from the original fix."
    - "A signal during a HUNG boot is swallowed until boot completes or fails (go-plugin's own 1-minute start timeout bounds the plugin-launch part). Same trade-off runSync has always had with NotifyContext-at-top."
    - "The orphan-vs-reaped secondary race (whether the child survives its handshake write to the dead kernel's pipe) explains why CI failure is ~50% rather than 100%, and why only the pid-directed SIGINT row was reported: in group rows the young child usually dies of the group signal itself. Not separately load-tested per row."
  candidate_causes:
    - "code (CONFIRMED): startup-ordering window — plugin spawn precedes signal.Notify in runServe"
    - "environment (contributing, not root): 2-vCPU shared CI runner stretches the spawn->Notify window across the test's 50ms poll granularity; a 12-core idle desktop completes boot within one poll interval, hiding the window (20/20 green even pinned to 1 CPU at 50ms poll; probe at 1ms poll hits it 15/15)"
    - "config/data: ruled out — same config/fixtures pass deterministically when the signal arrives post-boot"
  and_gate: "YES — the ORPHAN needs (A) kernel dies without running teardown (signal in the window) AND (B) the specific child survives parent death (won its handshake-write race; go-plugin has no parent-death watchdog). (A) is the condition topos owns and is what the fix closes; (B) is upstream-by-design, same as the original session's AND-gate."

## Symptoms
<!-- prefilled by orchestrator; IMMUTABLE -->

DATA_START
expected: On SIGINT to the kernel pid, the kernel's signal path (runServe select over SIGINT/SIGTERM -> graceful srv.Shutdown -> deferred linkStore.Shutdown -> sup.Shutdown) reaps every plugin subprocess before exit; the test asserts no plugin outlives the kernel.
actual: Plugin subprocess 6819 survived kernel exit in state "S", reparented to ppid 1 — orphaned. Notably "kernel produced no output" — the kernel process emitted nothing before exiting.
errors: none beyond the test assertion at shutdown_signal_test.go:355
reproduction: CI (GitHub Actions, CGO_ENABLED=0, make test-portable) fails ~50% of runs; presumably reproducible locally under repetition/load
timeline: Recurrence — original orphaned-plugins bug fixed in resolved/plugin-procs-orphaned-on-exit.md (799200b); resolved/ci-flaky-test-portable.md separately fixed a whatsapplink spawner pipe race (4c524a7). Phase 14 plan 14-01 (eef30c4) modified cmd/topos/main.go (resolveConfigPath / --config / TOPOS_CONFIG).
DATA_END

## Eliminated

## Evidence

- timestamp: 2026-08-18T00:00:00Z
  checked: "Phase 0 knowledge-base match"
  found: "KB-004 matches on [orphaned, reparented ppid 1, plugin subprocess survived, signal, defer unreachable]. Root cause was: no signal handler at all -> defers dead on signal paths. Fix was: signal.Notify + select in runServe (799200b). Why not caught: no gate exercised signal paths. Recurrence guard: this very test (the 2x2 matrix). KB-003/KB-005/KB-007 all warn the recurrence may be LATENT rather than introduced by the most recent commit — verify against baseline at high count before blaming 14-01."
  implication: "The recurrence guard is firing, which means either the guarded contract re-broke or the guard exposes a residual window the original fix never covered. Treat KB-004's fix shape as the thing to re-inspect: WHERE is signal.Notify relative to the plugin spawn?"

- timestamp: 2026-08-18T00:05:00Z
  checked: "cmd/topos/main.go current runServe ordering (worktree @ 8b0ce8a)"
  found: "Boot order: setup (line 400) -> ReconcileInterruptedSyncRuns (412) -> supervisor.NewSupervisor (436, spawns plugin subprocesses) -> defer sup.Shutdown (440) -> httpapi.Router (455) -> defer linkStore.Shutdown (456) -> http.Server construction (486) -> signal.Notify (488-489) -> ListenAndServe goroutine (493) -> logger.Info(\"topos serving\") (501) -> select (503). runSync by contrast calls signal.NotifyContext FIRST (line 354), before setup/NewSupervisor."
  implication: "A window exists in runServe where a plugin child is alive but SIGINT/SIGTERM still has default disposition. The test's precondition gate (poll /proc for a child every 50ms, signal immediately) is precisely shaped to land inside this window on a slow runner. 'Kernel produced no output' is consistent: the only unconditional pre-Notify Info log would come from the supervisor (to check) and 'topos serving' is post-Notify."

- timestamp: 2026-08-18T00:05:00Z
  checked: "shutdown_signal_test.go harness (full read)"
  found: "Harness is sound per KB-004's three trap list: signals via syscall.Kill from the test process (no shell SIG_IGN inheritance), child discovery via union over /proc/<pid>/task/*/children, output captured to *os.File. Precondition waits ONLY for a child to exist (line 317-327) — it does NOT wait for the kernel to be serving/listening. Then signals immediately."
  implication: "The harness deliberately signals at the earliest observable moment of plugin life. That is a legitimate real-world timing (service-manager stop during startup), so if the kernel orphans in this window it is a PRODUCT hole, not a harness bug."

- timestamp: 2026-08-18T00:20:00Z
  checked: "git log 0010698..HEAD -- cmd/topos kernel/supervisor kernel/pluginhost + git show 0010698"
  found: "The original reap fix placed signal.Notify at the END of runServe (hunk @@ -197 — after NewSupervisor) from day one. Since then phases 11-13 added post-spawn boot work (describe RPCs carrying tier/hash/icons, launch-gate interplay), and 14-01 only added pre-runServe flag parsing. Nothing ever moved Notify."
  implication: "The window is LATENT since the original fix, not introduced by 14-01 — exactly the KB-003/KB-005/KB-007 'diff-independent means latent' pattern. Its width grew with boot work and CI starvation until the test's 50ms poll started landing inside it."

- timestamp: 2026-08-18T00:30:00Z
  checked: "Local stress: go test -run .../SIGINT_to_kernel_pid -count=20, unpinned, then pinned taskset -c 0,1, then taskset -c 0 (50ms precondition poll throughout)"
  found: "60/60 PASS, iterations 0.05-0.06s. Even one pinned CPU of an idle desktop completes the whole spawn->Notify window inside a single 50ms poll interval."
  implication: "The harness's 50ms poll granularity cannot reach the window on this machine — reproduction needs the signal delivered at the earliest instant of child existence, not more repetitions (KB-003's under-sampling lesson, inverted: the sampling GRID, not the count, was the limiter)."

- timestamp: 2026-08-18T00:45:00Z
  checked: "sigprobe (scratchpad): starts the test-built kernel, polls /proc children at 1ms, SIGINTs the kernel pid at first child sighting, records wait status + log bytes + child post-mortem. 15 iterations at delay=0; 5 iterations at delay=500ms"
  found: |
    delay=0:  15/15 kernel[signal: interrupt], logBytes=0, handlerLog=false.
              14/15 children died anyway (young child loses its handshake-write
              race against the dead kernel's closing pipes); 1/15 ORPHAN
              child state=S ppid=1 — the exact CI signature.
    delay=500ms: 5/5 kernel[exit status 0], handlerLog=true, all REAPED.
  implication: "Root cause CONFIRMED with a controlled differential. The kernel ALWAYS dies unhandled when signalled at first-child-instant (the window is deterministic, not probabilistic); the test's ~50% CI failure rate is the SECONDARY race (child survival) stacked on the CI-only window hit. Teardown is intact; only its reachability during startup is broken. This is a PRODUCT hole: a service-manager SIGTERM or operator Ctrl-C during kernel startup (plugins spawned, handler not yet installed) orphans plugins in production, test aside."

- timestamp: 2026-08-18T01:30:00Z
  checked: "Post-fix behaviour of the GROUP-directed rows at the new 1ms earliest-instant timing"
  found: "Group SIGINT/SIGTERM at first-child-instant can kill the INFANT plugin child (go-plugin's own 'eat the interrupts' handler is not installed yet either); the kernel then survives the signal, its describe RPC to the dead child fails, NewSupervisor returns an error, and runServe exits 1 via fatal() WITH the deferred teardown having run — no orphan, but a non-zero exit. The first draft of the hardened oracle (exit MUST be 0) failed these rows."
  implication: "The precise product contract is 'the kernel must never die BY the shutdown signal' (WaitStatus.Signaled()), not 'must exit 0'. A launch-failure exit(1) after a group signal shoots the infant child is legitimate: teardown ran, nothing leaked. Oracle refined accordingly; the orphan assertions remain the leak guard for the non-zero-exit path."

## Resolution

root_cause: |
  Startup-ordering window in runServe (cmd/topos/main.go), latent since the
  original reap fix (0010698): signal.Notify was registered at the BOTTOM of
  runServe — after supervisor.NewSupervisor had already spawned every plugin
  subprocess. In the window (plugins alive, handler not yet installed) a
  SIGINT/SIGTERM killed the kernel with the Go runtime's default disposition:
  no output, no defers, no teardown. The orphan then materialises when the
  child ALSO survives its parent's death — it does whenever its go-plugin
  handshake line was written before the kernel's pipe read-ends closed
  (otherwise the infant child dies of SIGPIPE/EPIPE on that write) — the
  secondary race that made the CI failure ~50% rather than 100%.

  The regression test signals at the first observable plugin child. On a
  starved 2-vCPU CI runner the widened boot (phases 11-13 added post-spawn
  describe RPCs carrying tier/hash/icons) stretches the window across the
  test's 50ms poll granularity; a fast dev box completes boot inside one
  poll interval, which is why the flake was CI-only. NOT introduced by
  14-01 (pre-runServe flag parsing only) — diff-independent and latent,
  the KB-003/KB-005/KB-007 pattern again.

  This is a PRODUCT hole, not a harness race: a service-manager SIGTERM or
  operator Ctrl-C landing during kernel startup (a legitimate real-world
  timing) orphaned plugins in production. The harness's earliest-instant
  signal is a valid schedule, so the product was fixed, not the gate.

fix: |
  cmd/topos/main.go — move the sigCh registration (make chan / signal.Notify
  / defer signal.Stop) from the bottom of runServe to its TOP, before setup()
  and supervisor.NewSupervisor, mirroring the register-before-spawn ordering
  runSync has always had via NotifyContext. A signal arriving mid-boot is
  held in the buffered sigCh until the select at the bottom of the function,
  so boot completes and the ordinary graceful teardown (existing defers, in
  their existing order) reaps every subprocess. No plugin can now exist
  before the handler does. No other logic changed.

  cmd/topos/shutdown_signal_test.go — two hardenings so the guard bites on
  every machine, not only on starved CI:
  (1) precondition poll tightened 50ms -> 1ms, so the signal is delivered at
      the earliest instant of plugin life (the window timing) on any host;
  (2) new mechanism oracle: the kernel's wait status must not be Signaled()
      — death BY the shutdown signal is default disposition, i.e. teardown
      skipped — deterministic where the orphan check needs the secondary
      child-survival race. A non-zero EXIT is deliberately accepted (group
      rows can kill the infant child, failing the launch; teardown still
      runs; orphan assertions still guard leaks).

verification:
  oracle_type: |
    specified — "the kernel must handle SIGINT/SIGTERM gracefully (never die
    by default disposition) and no plugin subprocess may outlive it";
    boundary neighbours covered by the existing 2x2 matrix rows plus the
    earliest-instant timing (the window boundary) and the post-boot timing
    (probe differential).
  guardrail_verdict: accepted
  signal_1_revert_test: |
    PASS — with the fix reverted (git checkout HEAD -- cmd/topos/main.go)
    the hardened test fails 12/12 subtests across 3 runs, every one via
    "kernel was killed BY <sig> ... (default disposition — teardown
    skipped)", with 6 additional live-orphan hits (state R, ppid 1,
    "<kernel produced no output>" — the exact CI signature). Restoring the
    fix returns it to green.
  signal_2_regression_test: |
    PASS — the hardened gate is deterministic against the unfixed kernel on
    a fast 16-core desktop (12/12), where the pre-hardening gate was 60/60
    green even pinned to one CPU: the 50ms sampling grid, not the iteration
    count, was what hid the window locally. Mutation-checked with the FINAL
    oracle shape.
  signal_3_diff_shape: "PASS — additive: registration moved earlier + comments; test gains an oracle and a tighter poll; no logic deleted."
  signal_4_existing_gates: |
    PASS — make test-portable fully green (root module + sdk + all seven
    portable plugin modules); go vet clean; gofmt clean.
  signal_5_root_cause_not_symptom: |
    PASS — the fix closes the ordering window itself. Explicitly rejected
    symptom-shaped alternatives: making the test wait for "topos serving"
    before signalling (would mask the real production hole), and sleeps/
    retries (would re-hide the window behind timing).
  stability: |
    Pre-fix observed rates: CI ~50%/run at 50ms poll; local hardened
    harness 12/12 subtest failures (100%); sigprobe at earliest-instant
    timing: 15/15 kernels died by raw SIGINT (window hit deterministic),
    1/15 produced the live orphan (secondary race).
    Post-fix: 40 full-matrix test runs (20 unpinned + 20 pinned to one
    CPU) = 160 subtests, 0 failures; sigprobe same timing: 15/15 exit
    status 0, "shutdown signal received" logged, all children reaped.
  human_verify: PENDING — awaiting operator confirmation (green CI runs of make test-portable on the real 2-vCPU runner is the environment this flake lived in).

files_changed:
  - "cmd/topos/main.go: sigCh registration moved to the top of runServe (before any plugin spawn); pointer comment left at the old site"
  - "cmd/topos/shutdown_signal_test.go: 1ms precondition poll + not-Signaled() wait-status oracle; header documents the second act"
