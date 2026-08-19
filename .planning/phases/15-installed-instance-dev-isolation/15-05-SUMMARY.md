---
phase: 15-installed-instance-dev-isolation
plan: 05
subsystem: infra
tags: [isolation, simultaneity, smoke, docs, migration]

requires:
  - phase: 15-installed-instance-dev-isolation
    provides: "15-01's install.sh + install-smoke harness, 15-04's topos-devguard and the 7778 port move"
provides:
  - "make isolation-check (ISOL-03): committed proof that an installed-shaped and a dev-shaped instance run side by side — static port contract, byte-unchanged installed tree across a dev run, concurrent independence with file-set comparison"
  - "scripts/smoke-lib.sh: shared fixture-release builder + free-port helper sourced by both smokes"
  - "docs/testing.md naming all seven gates; CONTRIBUTING.md dev-loop and gate list in step"
  - "docs/install.md migration runbook (INST-03): checkout instance -> installed instance, losing nothing, with troubleshooting"
affects: [milestone-close, uat]

actuals:
  tokens: 11000
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Static contract assertion: read both port defaults from source rather than binding — the gate holds beside a live instance"
    - "File-SET comparison catches stray writes even where legitimate activity changes existing bytes"
    - "Background kernels as direct env-prefixed commands, never via a shell function (a function's $! is the subshell, orphaning the kernel)"

key-files:
  created:
    - scripts/simultaneity-smoke.sh
    - scripts/smoke-lib.sh
  modified:
    - Makefile
    - scripts/install-smoke.sh
    - docs/testing.md
    - CONTRIBUTING.md
    - docs/install.md

key-decisions:
  - "The gate documents seven gates (docs-check included) rather than the plan's estimated six — the acceptance criterion is heading-count == subsection-count, and docs-check was a real, previously-undocumented gate target"
  - "The dev instance's 'real write' is a mark through the marks API (SetItemMarks needs no live item), keeping the write genuine without coupling the gate to mock corpus contents"
  - "The installed-shaped instance resolves its config THROUGH XDG (no --config flag) so the fixture genuinely exercises the installed resolution chain"

patterns-established: []

requirements-completed: [INST-03, ISOL-03]

coverage:
  - id: D1
    description: "Installed and dev instances run simultaneously, each answering only its own webspaces; a dev run leaves the installed tree byte-identical; port defaults differ"
    requirement: ISOL-03
    verification:
      - kind: integration
        ref: "make isolation-check (3 cases: port contract static, byte-unchanged digest manifest, concurrent+file-set) — all pass; real-port baseline re-asserted per case"
        status: pass
  - id: D2
    description: "docs/testing.md names every gate (7) and matches the Makefile's target set; both docs describe port/refusal/escape-hatch identically"
    verification:
      - kind: other
        ref: "heading vs subsection count check (seven == 7); DEV_ISOLATION_BYPASS named in both docs; make docs-check green"
        status: pass
  - id: D3
    description: "An operator can follow docs/install.md from a running checkout instance to a running installed instance without losing config, index, marks, or linked sessions"
    requirement: INST-03
    verification:
      - kind: other
        ref: "migration + troubleshooting sections present; negative pattern check (no delete/move instruction) returns 0; links resolve under docs-check"
        status: pass
      - kind: manual_procedural
        ref: "phase <human-check>: live migration on the operator's machine (steps 1-5 of the runbook + dev alongside + uninstall back-out)"
        status: unknown
    human_judgment: true
    rationale: "The live migration on the operator's real instance and data is the phase's stated UAT — deferred to end-of-phase human check"

duration: 30min
completed: 2026-08-19
---

# Phase 15 Plan 05: Simultaneity gate and migration runbook Summary

`make isolation-check` now pins ISOL-03 as a committed gate — an installed-shaped instance (installed through the real install script, resolving through XDG) and a dev-shaped instance (pre-validated by the real devguard) run side by side with the installed tree proven byte-identical and its file set unchanged, plus the static assertion that the production and dev port defaults differ; the testing map documents all seven gates; and docs/install.md carries the migration runbook that takes an operator from checkout to installed losing nothing.

- Duration: ~30min (2026-08-18T23:48Z → 2026-08-19T00:09Z)
- Tasks: 3
- Files: 7 (2 created, 5 modified)

## Accomplishments

- `scripts/simultaneity-smoke.sh`: three named cases (port contract static — read from `kernel/config/types.go` and the Makefile, never bound; byte-unchanged — recursive digest manifest across a dev sync + mark write with the installed instance stopped; concurrent and independent — both serving on ephemeral ports, each listing exactly its own webspace, installed file-set unchanged across a driven dev write). Real-port safety baseline re-asserted after every case; everything under one mktemp tree with HOME/XDG pointed into it.
- `scripts/smoke-lib.sh` extracted (fixture-release builder + free-port), sourced by both install-smoke (15 cases unregressed) and the new gate — one definition.
- `docs/testing.md`: "The seven gates" with new subsections for `docs-check`, `install-check`, `isolation-check`; the dev/prod port split, mechanical refusal, and two-variable escape hatch documented; per-checkout plugin-store subsection; dated What-changed entry. `CONTRIBUTING.md` carries the same facts as references plus the full gate list.
- `docs/install.md`: migration runbook (preservation rationale, steps, the absolute-`[plugins] dir` case with both remedies, verification list, back-out, dev-alongside hand-off) and troubleshooting (plugins-not-found → absolute-dir case; `manifest_unverified` → Signal section).

## Task Commits

1. Task 1 (gate): `587ec66` feat(15-05) make isolation-check — the committed proof both instances run side by side
2. Task 2 (testing docs): `041a7f9` docs(15-05) bring the testing map and CONTRIBUTING back in step with seven gates
3. Task 3 (runbook): `e2bb8c2` docs(15-05) migration runbook — checkout instance to installed instance, losing nothing

## Deviations from Plan

- **[Rule 1 - Bug] Backgrounded-function orphan in the first gate implementation** — Found during: Task 1 verification. Issue: backgrounding kernels through the `with_env` shell function made `$!` the subshell pid; killing it orphaned the case-2 dev kernel, which still held its port when case 3 rebound it (`bind: address already in use`). Fix: kernels background as direct env-assignment-prefixed commands. Files: scripts/simultaneity-smoke.sh. Verification: `make isolation-check` green. Commit: `587ec66` (fixed before first commit).
- **Gate count: seven, not six.** The plan's objective said "six gates"; the repository actually has seven gate targets once `docs-check` (real, previously undocumented) is counted. The self-consistent acceptance criterion (heading count == subsection count) was followed.

**Total deviations:** 1 auto-fixed bug + 1 documentation-count correction. **Impact:** none — all acceptance criteria pass.

## Verification Results

- `make isolation-check` — all 3 cases pass
- `make install-check` — all 15 cases pass through the extracted lib
- `make dev-check` — all 6 cases pass (Makefile edits unregressed)
- `make docs-check` — green; all cross-links resolve
- Negative pattern check on the runbook (no delete/move/rename of operator data): 0 matches

## Issues Encountered

None beyond the recorded deviation.

## Self-Check: PASSED

- All key-files exist; scripts mode 0755
- 3 task commits present (`git log --grep=15-05`)
- All acceptance criteria re-run green at close-out

## Next Phase Readiness

Phase 15's five plans are all complete. Remaining before milestone close: phase verification, and the phase `<human-check>` UAT items — the live migration (15-05), the live `make install-signal` consent flow (15-03), and the roadmap's end-to-end operator proof.
