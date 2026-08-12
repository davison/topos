---
phase: 10-docs-and-release-readiness
plan: 04
subsystem: infra
tags: [github-milestones, gh-cli, release-engineering, documentation]

# Dependency graph
requires:
  - phase: 10-01
    provides: ".github/workflows/release.yml, .github/workflows/nightly.yml, Makefile build-portable/plugins-portable targets, the recorded Signal-binary-exclusion decision (option-b)"
provides:
  - "scripts/sync-milestones.sh — idempotent gh api wrapper reconciling one GitHub milestone with .planning/'s milestone state, no delete path"
  - "docs/releasing.md — the maintainer-facing page for milestone sync, cutting a release, nightlies, and the Signal-binary decision"
affects: [10-05-readme-install-section]

# Actuals (#2632)
actuals:
  tokens: 2458
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "gh api milestone lookup-by-title-across-all-states before create-vs-patch — the idempotency mechanism this repo's sync scripts use for any GitHub metadata that .planning/ owns and GitHub only mirrors"

key-files:
  created:
    - scripts/sync-milestones.sh
    - docs/releasing.md
  modified: []

key-decisions:
  - "scripts/sync-milestones.sh implements no delete path at all (capability absent, not merely unused) — .planning/ never deletes a milestone either, and a delete would orphan every issue assigned to it"
  - "docs/releasing.md records the Plan 10-01 Task 2 Signal-binary decision (option-b, excluded from published artifacts) as a dated decision with its reason, not as a description of current behavior a later maintainer might casually reverse"

patterns-established:
  - "Milestone sync is a manually-invoked, committed script run at milestone boundaries (paired with /gsd-complete-milestone and /gsd-new-milestone) — deliberately not a scheduled or event-triggered GitHub Action, since milestone boundaries are infrequent human-triggered events in this project's actual workflow"

requirements-completed: [SC-5]

coverage:
  - id: D1
    description: "scripts/sync-milestones.sh is executable, rejects bad input with a named usage line, and reconciles the real pre-existing v1.0 milestone (#1) idempotently — run twice, still exactly one v1.0 milestone, still number 1"
    requirement: "SC-5"
    verification:
      - kind: other
        ref: "live gh api verification this session against davison/topos: no-args exits 1 with a Usage line; 'v1.0 bogus-action' exits 1 naming the received value; two consecutive 'v1.0 open' runs leave milestone count=1 and number=1; grep confirms no DELETE method in the script; git status --porcelain .planning/ empty after running"
        status: pass
    human_judgment: false
  - id: D2
    description: "docs/releasing.md covers milestones, cutting a release, nightlies, and the Signal-binary decision in that order, cross-links CONTRIBUTING.md/docs/plugins//SECURITY.md, and every repository path it references (workflows, script, docs/plugins/signal.md) resolves to a real file"
    requirement: "SC-5"
    verification:
      - kind: other
        ref: "grep/test-based acceptance checks run this session: sync-milestones.sh, checksums.txt, nightly, make signal, STATE.md + 'source of truth', docs/plugins/signal.md + CONTRIBUTING.md + SECURITY.md, v[0-9] + idempotent all present; reference-resolution loop over every .github/workflows/*.yml, scripts/*.sh, docs/*.md path found in the file confirmed each exists on disk"
        status: pass
    human_judgment: false

# Metrics
duration: ~20min
completed: 2026-08-12
status: complete
---

# Phase 10 Plan 4: GitHub Milestone Sync + Release Documentation Summary

**Idempotent `gh api` milestone-sync script (verified live against the real `v1.0` milestone) plus the single maintainer-facing page covering milestone sync, release cuts, nightlies, and the recorded Signal-binary exclusion**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-08-12
- **Completed:** 2026-08-12
- **Tasks:** 2 (both `type="auto"`)
- **Files modified:** 2 (both new)

## Accomplishments
- Added `scripts/sync-milestones.sh`: a committed, idempotent `gh api` wrapper that reconciles one GitHub repository milestone with `.planning/`'s milestone state — lookup-by-title-across-all-states is the idempotency mechanism, and the script implements no delete path at all
- Verified the script live against the real `davison/topos` GitHub API and the real, pre-existing `v1.0` milestone (#1): no-args and bad-action invocations both reject correctly with a named usage/error line; two consecutive `v1.0 open` runs leave exactly one `v1.0` milestone, still numbered 1 — the pre-existing milestone was reconciled, not duplicated
- Added `docs/releasing.md`: the single page a maintainer reads to sync milestones, cut a release, understand what the nightly workflow does, and see why the Signal plugin binary is not a published artifact — records Plan 10-01's Task 2 checkpoint decision (option-b) as a dated decision with its reason, not a re-derivation

## Task Commits

Each task was committed atomically:

1. **Task 1: Idempotent milestone sync script, verified against the real milestone** - `aea5a73` (feat)
2. **Task 2: docs/releasing.md — milestone sync, release cuts, nightlies, and the Signal decision** - `acfce3d` (docs)

**Plan metadata:** this commit (SUMMARY.md) — see below

## Files Created/Modified
- `scripts/sync-milestones.sh` - executable `gh api` wrapper; positional args `<title> <open|close>`, `TOPOS_REPO` env override (default `davison/topos`), echoes resolved repo + operation before any mutation, looks up by title across all states before create-vs-patch, no delete path
- `docs/releasing.md` - four sections in order: Milestones, Cutting a release, Nightlies, The Signal plugin binary; cross-links `CONTRIBUTING.md`, `docs/plugins/`, `SECURITY.md`

## Decisions Made
- No new decisions this plan — `docs/releasing.md` records Plan 10-01's Task 2 decision (Signal binary excluded from published artifacts, option-b) verbatim per that plan's own SUMMARY instruction, rather than re-litigating it.

## Deviations from Plan

None - plan executed exactly as written. Both tasks' `<action>` and `<verify>` blocks were followed as specified; every automated verification step in the plan (script syntax/shebang/pipefail, no-args and bad-action rejection, live idempotency proof against the real `v1.0` milestone, no-delete-path grep, `.planning/` write-nothing check, and every `docs/releasing.md` content/reference-resolution grep) ran and passed.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required. `gh auth status` was already authenticated with `repo`/`workflow` scopes for `davison/topos` (Task 1's `<precondition>`), so no setup step was needed this session.

## Next Phase Readiness

- `scripts/sync-milestones.sh` is committed, executable, and proven idempotent against the real GitHub milestone — ready to be invoked at the next real milestone boundary (`/gsd-complete-milestone` / `/gsd-new-milestone`).
- `docs/releasing.md` is the canonical maintainer-facing release doc; Plan 10-05's repo-wide link guard (run after all Phase 10 worktree branches merge) is the final check that every cross-link this file makes — including `CONTRIBUTING.md`, which does not yet exist in this worktree but is created by a sibling plan in this same phase — resolves once the phase's branches are integrated.
- Nothing in this plan touches `.planning/STATE.md`, `.planning/ROADMAP.md`, or `.planning/REQUIREMENTS.md` — those are updated centrally by the orchestrator after all wave agents complete, per this plan's worktree-mode instructions.

## Self-Check: PASSED

- FOUND: `scripts/sync-milestones.sh`
- FOUND: `docs/releasing.md`
- FOUND commit: `aea5a73` (Task 1)
- FOUND commit: `acfce3d` (Task 2)

---
*Phase: 10-docs-and-release-readiness*
*Completed: 2026-08-12*
