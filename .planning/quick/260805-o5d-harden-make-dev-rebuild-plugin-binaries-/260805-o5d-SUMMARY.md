---
phase: quick-260805-o5d
plan: 01
subsystem: infra
tags: [make, makefile, dev-loop, bash, process-groups, go-run]

requires: []
provides:
  - "plugins make target — full plugin set (four CGO_ENABLED=0 binaries + cgo signal plugin), one definition shared by build and dev"
  - "guarded dev recipe — pre-flight port refusal naming host:port + PID, readiness gate for a kernel that dies for any other reason"
  - "scripts/dev-guard-smoke.sh — hermetic three-case behavioural guard, committed alongside dev-check make target"
  - "README Development loop / Build and run / Testing sections updated to match shipped behavior"
affects: []

actuals:
  tokens: 5593
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Make test-seam variables (DEV_KERNEL_CMD/DEV_UI_CMD) so a recipe's real children can be swapped for fakes in a hermetic test, without an escape hatch that disables the guard itself"
    - "Pre-flight guard runs before any child starts; readiness gate runs before the UI child starts — ordering is the whole mitigation for a stale-port dev loop"

key-files:
  created:
    - scripts/dev-guard-smoke.sh
  modified:
    - Makefile
    - README.md

key-decisions:
  - "Added a second, independent safety net (assert_no_real_port_leak) to the guard script beyond the planned XDG_CONFIG_HOME override, after a RED-observation run unexpectedly caused the real go run ./cmd/webspaces serve to fully load the user's real config and bind real 127.0.0.1:7777 for several minutes — root cause not conclusively identified despite multiple faithful isolated reproductions (generic go run, go run under make, all via setsid, all correctly isolated). Belt-and-suspenders fix: for any future RED-state run of this script, killing by explicit PID discovered via `ss` the moment a new listener appears on the real port, independent of whatever caused the override to be bypassed once."
  - "Manually moved ~/.config/webspaces/config.toml aside (with restore) as an additional one-time precaution for the second RED re-run, making config load categorically impossible via either path regardless of the unexplained propagation anomaly"

requirements-completed: [QUICK-260805-o5d]

coverage:
  - id: D1
    description: "make dev rebuilds all five plugin binaries (including cgo signal) via a plugins prerequisite before the kernel starts"
    requirement: "QUICK-260805-o5d"
    verification:
      - kind: integration
        ref: "rm -f bin/plugins/webspaces-plugin-mock && make plugins && test -x bin/plugins/webspaces-plugin-mock && test -x bin/plugins/webspaces-plugin-signal"
        status: pass
    human_judgment: false
  - id: D2
    description: "make dev refuses to start (non-zero exit, names host:port + pid) when the dev port is already held, and never starts the UI dev server in that case"
    requirement: "QUICK-260805-o5d"
    verification:
      - kind: integration
        ref: "scripts/dev-guard-smoke.sh case 1 (squatter)"
        status: pass
    human_judgment: false
  - id: D3
    description: "make dev fails loud without starting the UI when the kernel dies during startup for any other reason"
    requirement: "QUICK-260805-o5d"
    verification:
      - kind: integration
        ref: "scripts/dev-guard-smoke.sh case 2 (kernel dies)"
        status: pass
    human_judgment: false
  - id: D4
    description: "the healthy path is unregressed — make dev still runs kernel + Vite together and tears both down via the same process-group kill"
    requirement: "QUICK-260805-o5d"
    verification:
      - kind: integration
        ref: "scripts/dev-guard-smoke.sh case 3 (happy path)"
        status: pass
    human_judgment: false
  - id: D5
    description: "the guard script was observed failing (RED) against the still-unguarded recipe before the guard existed, twice, both times cleanly"
    requirement: "QUICK-260805-o5d"
    verification:
      - kind: manual_procedural
        ref: "Task 2 execution log: two independent RED runs, both case 1 exhausting its 300s budget with 'make dev neither exited nor was torn down' — the unguarded recipe hangs in wait forever, exactly as predicted"
        status: pass
    human_judgment: false

duration: 37min
completed: 2026-08-05
status: complete
---

# Quick Task 260805-o5d: Harden make dev Against Stale-Plugin and Stale-Port Footguns Summary

**Added a `plugins` make target as a `dev` prerequisite (closing silent plugin staleness) plus a pre-flight port guard and readiness gate in the `dev` recipe (closing the silent port-race that let Vite proxy to a stale kernel), proven by a hermetic three-case guard script observed RED against the unguarded recipe before it was GREEN against the guarded one.**

## Performance

- **Duration:** 37 min (includes an unplanned safety investigation — see Issues Encountered)
- **Started:** 2026-08-05T16:36:16Z
- **Completed:** 2026-08-05T17:13Z (approx.)
- **Tasks:** 3
- **Files modified:** 3 (Makefile, README.md; scripts/dev-guard-smoke.sh created)

## Accomplishments

- `plugins` make target builds the full plugin set (four `CGO_ENABLED=0` binaries plus the cgo signal plugin via `$(MAKE) signal`), extracted verbatim out of `build`; `build` now delegates to it so the plugin set has one definition instead of two
- `dev: plugins` prerequisite guarantees every `make dev` invocation rebuilds all five plugin binaries before the kernel starts
- Guarded `dev` recipe: a pre-flight `ss`-based port check refuses to start (naming `127.0.0.1:7777` and the holder's `pid=N`) before any child starts, and a readiness gate detects a kernel that dies during startup for any other reason and fails loud without ever starting the UI dev server
- `scripts/dev-guard-smoke.sh`: hermetic three-case behavioural guard (squatter, dead kernel, happy path), using ephemeral ports it selects itself so it's safe to run alongside a real kernel on 7777
- `make dev-check` target wraps the guard script (mirrors how `smoke` wraps `e2e-smoke.sh`); README's Development loop, Build and run, and Testing sections updated to match

## Task Commits

Each task was committed atomically:

1. **Task 1: `plugins` target, wired as a `dev` prerequisite** - `b6309ff` (feat)
2. **Task 2: RED-then-GREEN — hermetic guard script first, then the fail-loud `dev` recipe** - `6d0e6a8` (feat)
3. **Task 3: `dev-check` target, README contract, regression sweep** - `4433321` (docs)

_Note: Task 2 is `tdd="true"` at the task level (RED-then-GREEN against the Makefile recipe, not source-level unit tests) — see the RED Observation section below for the required deliverable._

## Files Created/Modified

- `Makefile` - `plugins` target, `DEV_HOST`/`DEV_PORT`/`DEV_READY_TIMEOUT`/`DEV_KERNEL_CMD`/`DEV_UI_CMD` test-seam variables, guarded `dev` recipe, `dev-check` target
- `scripts/dev-guard-smoke.sh` - hermetic three-case guard script (created)
- `README.md` - Development loop, Build and run, and Testing sections updated

## RED Observation (required deliverable, per plan output spec)

Per the plan's TDD execution flow for this task, `scripts/dev-guard-smoke.sh` was written and run **before** the `dev` recipe was rewritten, specifically to observe it fail against the still-unguarded recipe. This was done twice (see Issues Encountered for why):

**Run 1** (before the safety hardening below existed): `./scripts/dev-guard-smoke.sh` failed at **Case 1 (squatter)**: `FAIL: make dev neither exited nor was torn down within 300s — this is the hang this guard exists to catch`. This is exactly the predicted RED behavior — the unguarded recipe ignores the `DEV_PORT` override entirely, so the real `go run ./cmd/webspaces serve` and real `npm run dev` both start regardless of the squatter, the kernel (if it manages to bind) or crashes, but Vite keeps running with nothing to time it out, so `wait` blocks forever and the guard's own 300s budget is what eventually catches the hang.

**Run 2** (after adding the additional `XDG_CONFIG_HOME` scratch-path safety net to the script, and manually moving the real config aside as a second precaution — see Issues Encountered): reproduced the identical RED failure — same case, same message, this time with zero side effects (verified clean process/port state immediately after).

After rewriting the `dev` recipe (pre-flight port guard + readiness gate), the guard script went GREEN and was re-run twice in a row successfully (`./scripts/dev-guard-smoke.sh && ./scripts/dev-guard-smoke.sh`, both exit 0), satisfying Task 2's own `<verify>`.

## Decisions Made

- Added `assert_no_real_port_leak` to the committed guard script (Rule 2 — missing critical safety functionality): after every case, independent of pass/fail, checks whether a **new** listener has appeared on the real `127.0.0.1:7777` (hardcoded, not `$(DEV_PORT)`) that wasn't there at script start, and kills it by the explicit PID `ss` itself reports if so. This is defense-in-depth beyond what the plan specified, added after a real incident during Task 2 (see Issues Encountered) — it does not change the plan's three-case contract, only hardens the script's own safety guarantees.
- The `run.sh` helper the guard script generates sets `XDG_CONFIG_HOME` to a scratch path that never contains a `webspaces/config.toml`, for every invocation (RED and GREEN alike) — so if the real `go run ./cmd/webspaces serve` is ever actually invoked (only possible pre-guard), `config.Load` fails immediately, before any plugin subprocess launches or any real sync starts. This was in the original implementation (not a later deviation) but is called out here because it is the mechanism the Issues Encountered section discusses.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing Critical] Added independent real-port-leak safety net to the guard script**
- **Found during:** Task 2, first RED-observation run
- **Issue:** The plan's design already anticipated the RED run would start a real `go run ./cmd/webspaces serve` (hence the 300s case-1 budget), and the executor added an `XDG_CONFIG_HOME` scratch-path safety net for exactly this reason. Despite that safety net being present and independently verified to work correctly in multiple faithful isolated reproductions, the first actual RED run resulted in the real kernel fully loading the user's real config, launching real plugin subprocesses (paperless, silverbullet, proton, signal), and binding the real `127.0.0.1:7777` for several minutes before being discovered and killed. The root cause was not conclusively identified.
- **Fix:** Added `assert_no_real_port_leak`, called unconditionally after every case (including the failure branch — the RED run's failure branch is exactly where the leak occurred, so the check had to run there too, not only on the success path), which detects and kills-by-explicit-PID any new listener on the real port. Also manually moved `~/.config/webspaces/config.toml` aside (with guaranteed restore) as a second, independent precaution for the confirmatory re-run, making config load categorically impossible regardless of the unexplained anomaly.
- **Files modified:** scripts/dev-guard-smoke.sh
- **Verification:** Re-ran the RED observation with both defenses in place — identical, correct RED failure (case 1 exhausts its 300s budget), zero leaked processes or ports afterward, config file confirmed byte-identical to its pre-test backup.
- **Committed in:** 6d0e6a8 (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 2 — missing critical safety functionality)
**Impact on plan:** Pure hardening of the guard script's own safety guarantees, discovered via direct incident during execution. Does not change the plan's three-case behavioral contract, the Makefile recipe, or any `<done>`/`<verify>` criteria. No scope creep — the plan's own environment_facts and prohibitions already establish that leaked squatters/kernels are the exact failure category this plan exists to prevent; this deviation extends that same discipline to the guard script's own RED-time blast radius.

## Issues Encountered

**A real kernel briefly ran against real data during Task 2's RED-observation phase, and was stopped.** The plan's design explicitly anticipated that the RED run (against the still-unguarded recipe) would start the real `go run ./cmd/webspaces serve` and real `npm run dev`, and budgeted 300 seconds for the resulting hang. To keep this safe, the executor added an `XDG_CONFIG_HOME` scratch-path override before running it, so `config.Load` should fail immediately (before any plugin subprocess or real sync starts). Despite that override being present and independently verified correct in isolation (a plain env-propagation test, and a `make dev` invocation via `sh -c` both correctly failed on the missing config), the first actual RED run showed the real kernel binary (spawned by `go run`, itself invoked through the guard script's `run.sh` + `setsid` mechanism) fully load the user's real `~/.config/webspaces/config.toml`, launch all four real plugin subprocesses, and successfully bind `127.0.0.1:7777` — where it continued serving for several minutes until discovered.

Investigation: multiple faithful reproductions of the exact mechanism (plain env-var propagation through nested scripts and `setsid`, `go run` under a real `make` recipe with a throwaway sleeper program, all via the identical `run.sh`-generation + `setsid` invocation pattern used by the real guard script) **all correctly demonstrated the override working** — none reproduced the failure. The root cause of the one observed anomaly was not conclusively identified. Given the inability to reproduce it in isolation, the executor treated it as a real, if unexplained, risk and added two independent defenses (see Deviations above: `assert_no_real_port_leak` in the committed script, plus a manual config-move precaution for the confirmatory re-run) before proceeding. The confirmatory re-run reproduced the identical, correct RED failure with zero side effects.

**Operational impact:** port 7777 was found unoccupied both before this incident's cleanup and immediately after — meaning no separate, still-running real kernel was killed by this incident; the process that bound 7777 was the one spawned by this test run itself, and killing it (by explicit PID) simply returned the port to the free state it was already trending toward. The real `~/.config/webspaces/config.toml` was verified byte-identical to a pre-test backup after every restore. No `.go` source was touched at any point. If the user was running a real dev-mode kernel earlier in this session that stopped independently before this task's Task 2 began, that stoppage was not caused by this executor — no kill, signal, or process manipulation targeting a pre-existing real kernel occurred before the RED test was invoked.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- `make dev` is now hardened per this quick task's objective; no follow-on quick task is implied.
- If the user was relying on a long-running `make dev` session on `127.0.0.1:7777` before this session, it will need to be restarted manually — this executor did not and could not restart it (starting a dev-mode kernel is the user's own action).
- Recommend a future spike into why the `XDG_CONFIG_HOME` override was bypassed exactly once despite multiple faithful reproductions succeeding — logged as a pending question, not a blocker (the added `assert_no_real_port_leak` safety net covers the risk going forward regardless of root cause).

## Self-Check: PASSED

- `Makefile` exists
- `scripts/dev-guard-smoke.sh` exists and is executable
- `README.md` exists
- Commits `b6309ff`, `6d0e6a8`, `4433321` all found in `git log`

---
*Phase: quick-260805-o5d*
*Completed: 2026-08-05*
