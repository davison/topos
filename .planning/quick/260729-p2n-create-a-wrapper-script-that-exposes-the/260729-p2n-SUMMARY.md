---
phase: quick-260729-p2n
plan: 01
subsystem: infra
tags: [bash, scripts, dotenv, dev-tooling]

# Dependency graph
requires:
  - phase: 01-first-webspace-end-to-end
    provides: kernel/config.Load's ${VAR}-style env expansion and bin/webspaces build target
provides:
  - "scripts/run-with-env.sh: sources+exports repo-root .env, execs bin/webspaces with args"
  - "scripts/check-run-with-env.sh: committed credential-free behavioural check for the wrapper"
  - "README.md documents the wrapper and all four .env keys (paperless + SilverBullet)"
affects: [uat, phase-03-email, phase-04-signal, phase-05-whatsapp]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "set -a / . <envfile> / set +a idiom (from scripts/e2e-smoke.sh) reused for exporting .env into a wrapper's exec'd child process"
    - "Credential-free behavioural check pattern: fixture .env with sentinel values + fake binary that reports presence-only (${VAR:+yes}), never the value itself"

key-files:
  created:
    - scripts/run-with-env.sh
    - scripts/check-run-with-env.sh
  modified:
    - README.md

key-decisions:
  - "No re-validation of which .env keys are present in the wrapper — kernel/config.Load already fails with a precise missing-variable error; duplicating that policy would drift"
  - "exec (not a backgrounded run) so signals/Ctrl-C reach bin/webspaces directly during a long-running serve"

patterns-established:
  - "Wrapper scripts follow the same shebang/set -euo pipefail/SCRIPT_DIR+REPO_ROOT/cd convention as scripts/e2e-smoke.sh and scripts/assert-stylesheet.sh"

requirements-completed: [QUICK-260729-p2n]

coverage:
  - id: D1
    description: "scripts/run-with-env.sh exports the repo-root .env and execs bin/webspaces with arguments passed through verbatim and in order"
    requirement: "QUICK-260729-p2n"
    verification:
      - kind: integration
        ref: "scripts/check-run-with-env.sh (ARGS: and *_SET:yes assertions)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Wrapper never emits a credential value on any code path (happy path, zero-arg, missing-binary)"
    requirement: "QUICK-260729-p2n"
    verification:
      - kind: integration
        ref: "scripts/check-run-with-env.sh (SENTINELVALUE9f3a absence assertions across all three invocations)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Missing/unbuilt bin/webspaces produces an actionable error naming make build; zero-argument invocation prints usage and exits non-zero"
    requirement: "QUICK-260729-p2n"
    verification:
      - kind: integration
        ref: "scripts/check-run-with-env.sh (missing-binary and zero-argument assertions)"
        status: pass
    human_judgment: false
  - id: D4
    description: "From a fresh shell with nothing exported, ./scripts/run-with-env.sh serve starts the kernel with all four keys reachable by the kernel and its plugin subprocesses against the real .env"
    human_judgment: true
    rationale: "Requires the operator's real repo-root .env with live paperless-ngx and SilverBullet credentials, plus a running kernel and reachable LAN sources — not something the automated fixture check can prove without touching real credentials, per the plan's own verification section (manual UAT confirmation)."

duration: 12min
completed: 2026-07-29
status: complete
---

# Quick Task 260729-p2n: run-with-env.sh Wrapper Summary

**Bash wrapper that sources and exports the repo-root `.env` before exec'ing `bin/webspaces`, plus a credential-free behavioural check proving pass-through, export, and zero credential leakage.**

## Performance

- **Duration:** 12 min
- **Started:** 2026-07-29T16:57:00Z
- **Completed:** 2026-07-29T17:09:07Z
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments
- `scripts/run-with-env.sh` sources the gitignored `.env` with the `set -a` / `.` / `set +a` idiom (matching `scripts/e2e-smoke.sh`), so `PAPERLESS_URL`/`PAPERLESS_TOKEN`/`SB_URL`/`SB_AUTH_TOKEN` are exported (not merely assigned) before `bin/webspaces` — and every go-plugin subprocess it spawns — reads them.
- `scripts/check-run-with-env.sh` proves all four required behaviours (argument pass-through and order, export reaching the child, zero credential leakage, actionable failure messages for zero-args and a missing binary) using a fixture `.env` with a sentinel value and a fake binary — no real credentials touched.
- README.md's "Build and run" section now documents the wrapper next to the direct-binary commands, and the "Configure" section names all four `.env` keys (the SilverBullet pair was previously undocumented, stale since Phase 02 shipped that source).

## Task Commits

Each task was committed atomically:

1. **Task 1: Add scripts/run-with-env.sh and its behavioural check** - `7becca1` (feat)
2. **Task 2: Document the wrapper in README** - `c7ca1fd` (docs)

_Note: docs commit for this SUMMARY/STATE update is made separately by the orchestrator._

## Files Created/Modified
- `scripts/run-with-env.sh` - Sources+exports repo-root `.env`, verifies `bin/webspaces` exists and is executable, `exec`s it with all arguments verbatim
- `scripts/check-run-with-env.sh` - Committed credential-free behavioural check (fixture `.env` + fake binary) asserting pass-through, export, no-leak, and both failure modes
- `README.md` - "Build and run" section documents the wrapper; "Configure" section lists all four `.env` keys

## Decisions Made
- No re-validation of which `.env` keys are present in the wrapper — `kernel/config.Load` already fails with a precise missing-variable error; duplicating that policy here would drift and could wrongly reject a config that only uses some of the four keys.
- Used `exec` (not a backgrounded run) so signals/`Ctrl-C` reach `bin/webspaces` directly during a long-running `serve`, matching the plan's explicit requirement.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required. This plan only wires existing `.env` credentials into a wrapper; it introduces no new secrets or accounts.

## Next Phase Readiness
- `./scripts/run-with-env.sh serve` is available for UAT sessions and future phase work (email/Signal/WhatsApp) that will add more `.env` keys — the wrapper requires no changes for additional keys since it exports whatever the `.env` file defines.
- Manual UAT confirmation (operator, per plan's `<verification>`) is the one item this executor could not perform: running `./scripts/run-with-env.sh serve` from a fresh shell against the real `.env` and confirming both paperless and SilverBullet sources are reachable at `http://127.0.0.1:7777/`. Tracked as coverage item D4 (`human_judgment: true`) above.

---
*Phase: quick-260729-p2n*
*Completed: 2026-07-29*

## Self-Check: PASSED

- FOUND: scripts/run-with-env.sh
- FOUND: scripts/check-run-with-env.sh
- FOUND: .planning/quick/260729-p2n-create-a-wrapper-script-that-exposes-the/260729-p2n-SUMMARY.md
- FOUND: 7becca1 (Task 1 commit)
- FOUND: c7ca1fd (Task 2 commit)
