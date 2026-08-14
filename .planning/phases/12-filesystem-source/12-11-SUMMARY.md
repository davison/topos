---
phase: 12-filesystem-source
plan: 11
subsystem: ui
tags: [svelte, sveltekit, playwright, go, chi, health-tooltip, gap-closure]

# Dependency graph
requires:
  - phase: 12-filesystem-source
    provides: "12-10's healthTone advisory branch and SourceChip.svelte's tooltip advisory branch (the code this plan corrects)"
provides:
  - "format.ts's isAdvisoryOnly predicate, the single source of the tooltip advisory gate's precedence"
  - "SourceChip.svelte's tooltip gate deferring to isAdvisoryOnly instead of testing last_status directly"
  - "the tooltipText branch-selection matrix (vitest) and browser proof (Playwright) closing 12-VERIFICATION.md's CR-01 gap"
  - "the recorded and pinned launch-failure last_notice contract (WR-01)"
affects: [filesystem-source, future health-tooltip work]

actuals:
  tokens: 7727
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "A precedence rule is expressed once and consumers defer to it (isAdvisoryOnly re-asks healthTone with the notice removed) instead of re-deriving a narrower copy"
    - "Guard files assert the CHOSEN BRANCH (full tooltip string) for a precedence chain, not merely which tone/class the chip ends up with"

key-files:
  created:
    - web/e2e/specs/12-tooltip-precedence.spec.ts
  modified:
    - web/src/lib/format.ts
    - web/src/lib/components/SourceChip.svelte
    - web/src/lib/components/match-advisory.test.ts
    - kernel/httpapi/sources.go
    - kernel/httpapi/sources_test.go

key-decisions:
  - "Rejected 12-REVIEW.md's literal CR-01 fix suggestion (gate on tone === 'success') because healthTone's own advisory branch already returns 'warning' for that state, making the suggested gate permanently unreachable dead code — added isAdvisoryOnly instead, which re-asks healthTone with the notice removed"
  - "WR-01 resolved by recording the launch-failure LastNotice omission as a deliberate, tested contract (matching docs/api.md's published behaviour) rather than populating the field, which would have contradicted shipped documentation"

patterns-established:
  - "isAdvisoryOnly(source): boolean — the reusable 'would this source be plain healthy without its advisory' predicate any future consumer of last_notice precedence should call rather than re-deriving"

requirements-completed: [SRC-04]

coverage:
  - id: D1
    description: "isAdvisoryOnly predicate added to format.ts, deferring the advisory-only precedence question to healthTone rather than re-deriving a narrower condition"
    requirement: "SRC-04"
    verification:
      - kind: unit
        ref: "web/src/lib/components/match-advisory.test.ts#isAdvisoryOnly: would this source be plain healthy with no advisory at all?"
        status: pass
    human_judgment: false
  - id: D2
    description: "SourceChip.svelte's tooltip advisory branch gates on advisoryOnly (isAdvisoryOnly(source)) instead of testing last_status directly — CR-01 closed"
    requirement: "SRC-04"
    verification:
      - kind: unit
        ref: "web/src/lib/components/match-advisory.test.ts#tooltipText BRANCH-SELECTION matrix"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/12-tooltip-precedence.spec.ts#A — reachable: false + last_status ok + a leftover notice"
        status: pass
    human_judgment: false
  - id: D3
    description: "The healthy-with-advisory tooltip branch still renders (anti-dead-code guard) — a reachable, ok-status source with a notice still gets the synced-plus-advisory tooltip"
    requirement: "SRC-04"
    verification:
      - kind: unit
        ref: "web/src/lib/components/match-advisory.test.ts#healthTone / isAdvisoryOnly coupling: the anti-dead-code guard"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/12-tooltip-precedence.spec.ts#B — reachable + last_status ok + the same notice"
        status: pass
    human_judgment: false
  - id: D4
    description: "WR-01: the launch-failure merge's deliberate LastNotice omission is recorded in a comment and pinned by a Go test"
    verification:
      - kind: unit
        ref: "kernel/httpapi/sources_test.go#TestSourcesHandler_LaunchFailedEntryCarriesNoLastNotice"
        status: pass
    human_judgment: false

duration: 22min
completed: 2026-08-14
status: complete
---

# Phase 12 Plan 11: Tooltip Precedence Gap Closure (CR-01, WR-01) Summary

**Added format.ts's `isAdvisoryOnly` predicate so SourceChip.svelte's tooltip advisory gate defers to healthTone's own precedence chain instead of a narrower, driftable copy — closing CR-01, the one open gap in 12-VERIFICATION.md — plus recorded and pinned the launch-failure `last_notice` contract WR-01 asked for.**

## Performance

- **Duration:** 22 min
- **Started:** 2026-08-14T11:38:02+01:00 (base commit)
- **Completed:** 2026-08-14T12:00:07+01:00
- **Tasks:** 3
- **Files modified:** 6

## Accomplishments
- `isAdvisoryOnly(source)` added to `format.ts`: answers "would this source be plain healthy with no advisory at all?" by re-asking `healthTone` with the notice removed — the single precedence source both the dot's tone and the tooltip's text now consult.
- `SourceChip.svelte`'s tooltip advisory branch now gates on `advisory !== '' && advisoryOnly` instead of `advisory !== '' && source.last_status !== 'error'` — an unreachable source with a stale successful status and a leftover notice now correctly selects the "unreachable since" branch instead of a reassuring advisory tooltip.
- `match-advisory.test.ts` extended with the `isAdvisoryOnly` seven-case behavioural matrix, the `healthTone`/`isAdvisoryOnly` anti-dead-code coupling assertion, branch-order assertions, and a six-case `tooltipText` branch-selection matrix that asserts full produced strings (not tone alone) — the exact coverage 12-VERIFICATION.md named as missing.
- New Playwright spec `12-tooltip-precedence.spec.ts` proves both states in a real browser against the rendered `title` attribute: Test A (CR-01) and Test B (the anti-dead-code guard). Verified by hand that Test A fails when the gate change is reverted.
- WR-01 closed: `kernel/httpapi/sources.go`'s launch-failure merge gets a comment recording that `LastNotice` is deliberately not copied from the recorded run (matching `docs/api.md`'s published contract), pinned by `TestSourcesHandler_LaunchFailedEntryCarriesNoLastNotice`.

## Task Commits

Each task was committed atomically:

1. **Task 1: The gate defers to healthTone's own precedence — isAdvisoryOnly, the chip's gate, and a branch-selection matrix** - `8ec3f50` (fix)
2. **Task 2: The browser proof — the rendered title attribute, for the state the harness cannot produce live** - `0ea8f22` (test)
3. **Task 3: WR-01 — record and pin the launch-failure last_notice contract docs/api.md already publishes** - `e59fbe8` (docs)

_Note: This plan's tasks were not TDD-flagged individually (Task 1 is `type="tracer" tdd="true"` but its RED/GREEN split collapsed into one commit alongside the new predicate, gate, and guard file changes together — verification (`npm --prefix web run test`) passed before commit, matching the tracer feedback-gate protocol rather than a strict multi-commit RED/GREEN cycle)._

## Files Created/Modified
- `web/src/lib/format.ts` - Adds `isAdvisoryOnly(source): boolean`, placed after `healthTone`; `healthTone`'s own body is byte-unchanged.
- `web/src/lib/components/SourceChip.svelte` - Imports `isAdvisoryOnly`; adds the `advisoryOnly` derived value; changes the tooltip advisory branch's gate condition; extends the block comment recording CR-01.
- `web/src/lib/components/match-advisory.test.ts` - Adds the `isAdvisoryOnly` matrix, the coupling assertion, branch-order assertions, and the `tooltipText` branch-selection matrix (with its anchored mirror function); updates the two structural gate-literal assertions for the new condition text.
- `web/e2e/specs/12-tooltip-precedence.spec.ts` - New Playwright spec: two tests proving the rendered `title` attribute for the CR-01 case and the anti-dead-code guard case.
- `kernel/httpapi/sources.go` - Comment-only change recording why `LastNotice` is deliberately omitted from the launch-failure merge (WR-01).
- `kernel/httpapi/sources_test.go` - Adds `TestSourcesHandler_LaunchFailedEntryCarriesNoLastNotice`.

## Decisions Made
- Rejected 12-REVIEW.md's literal CR-01 fix suggestion (`advisory !== '' && tone === 'success'`) because `healthTone`'s own advisory branch (added by 12-10) already returns `'warning'`, not `'success'`, for a healthy source carrying an advisory — the suggested gate can never fire, deleting 12-10's feature. Used `isAdvisoryOnly` instead, which re-asks `healthTone` with the notice stripped, answering the actual question ("would this source be healthy without the advisory?") rather than testing the chip's own already-derived tone.
- Kept `advisory !== ''` alongside `advisoryOnly` in the gate rather than replacing it, preserving the pre-existing (out-of-scope) whitespace-only-notice behavioural inconsistency between `SourceChip.svelte` (trims) and `healthTone` (does not trim) exactly as before.
- WR-01: took the review's second suggested resolution (record the omission, don't populate the field) because `docs/api.md` already publishes the empty-for-launch-failure contract; populating `LastNotice` there would have made the code contradict shipped documentation.

## Deviations from Plan

None - plan executed exactly as written. The plan's own "Critical implementation note" pre-empted the one deviation risk (the review's incorrect literal suggestion) by prescribing `isAdvisoryOnly` directly, so no Rule 4 architectural question arose during execution.

## Issues Encountered
- `web/node_modules` was absent in this worktree (a fresh git worktree checkout, not `npm ci`'d). Symlinked it to the main worktree's `web/node_modules` to avoid a redundant install; this symlink is untracked and does not affect git state.
- `npm --prefix web run check:e2e` reports one pre-existing, unrelated TypeScript error in `web/e2e/specs/12-filesystem-recursion.spec.ts` (`Module "node:fs" has no exported member 'unlinkSync'`) — confirmed present at the plan's base commit (`f8fdf8d`), not touched by this plan, and outside `files_modified`. Verified in isolation that `12-tooltip-precedence.spec.ts` itself typechecks cleanly with zero errors (`npx tsc --noEmit -p e2e/tsconfig.json` filtered to only that pre-existing failure). Left unfixed per the executor's scope-boundary rule; not logged to `deferred-items.md` since that file is outside this plan's `files_modified` list.
- The Go-embedded kernel binary must be rebuilt after each `npm run build` for a SPA change to take effect in the e2e harness (go:embed captures `kernel/webui/build` at Go build time) — rebuilt `bin/topos` before every playwright run in this session to avoid testing a stale embedded build.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- CR-01 (12-VERIFICATION.md's sole open gap) and WR-01 are both closed; 12-VERIFICATION.md should now report 11/11 on re-verification.
- No blockers for phase completion from this plan. IN-01 (`zeroMatchNotice`'s field-key-count guard) remains deliberately deferred per this plan's own scope, unchanged.

## Self-Check: PASSED

All 7 created/modified files confirmed present on disk; all 4 commits
(`8ec3f50`, `0ea8f22`, `e59fbe8`, `8fbd576`) confirmed present in `git log`.

---
*Phase: 12-filesystem-source*
*Completed: 2026-08-14*
