---
phase: 12-filesystem-source
plan: 09
subsystem: api
tags: [sqlite, sync, correlation, observability, go]

# Dependency graph
requires:
  - phase: 12-filesystem-source
    provides: correlate.Engine.SyncSource, syncer.Coordinator.syncOne, GET /api/sources (all pre-existing before this plan)
provides:
  - kernel/correlate.WebspaceResult.Notice and zeroMatchNotice — kernel-composed, plugin-agnostic zero-match advisory
  - kernel/index sync_runs.notice column (schemaVersion 3) and Store.FinishSyncRunWithNotice
  - kernel/syncer.RunResult.Notice and joinNotices — deterministic, bounded aggregation
  - GET /api/sources's last_notice field, additive at schema_version 1
affects: [12-10 (browser-visible advisory surface, next wave), any future phase reading sync_runs or GET /api/sources]

# Actuals (#2632)
actuals:
  tokens: 13970
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Sibling-method shape for a widened store write (FinishSyncRun delegates to FinishSyncRunWithNotice with an empty notice) — mirrors kernel/pluginhost/matchconfig.go's ValidateMatchConfig/ValidateMatchConfigWithSuspended, avoiding a ~25-call-site arity churn."
    - "A non-fatal advisory gets its own column/field end to end (never reuses the error channel) so it can never mask or be mistaken for a genuine failure."

key-files:
  created: []
  modified:
    - kernel/correlate/correlate.go
    - kernel/correlate/correlate_test.go
    - kernel/index/schema.go
    - kernel/index/store.go
    - kernel/index/store_test.go
    - kernel/syncer/coordinator.go
    - kernel/syncer/coordinator_test.go
    - kernel/httpapi/sources.go
    - kernel/httpapi/sources_test.go
    - docs/api.md

key-decisions:
  - "The zero-match notice is post-sync and advisory only — never a config-load or config-save rejection — because a load-time rejection would lock the operator out of the live config.toml they need to edit to fix the offending value."
  - "The notice occupies its own column/field (sync_runs.notice, RunResult.Notice, last_notice) rather than reusing sync_runs.error, so it can never flip a source's health chip to red or be swallowed by a genuine failure."
  - "Detection tests resp.GetItems() — the count the plugin actually returned — before the PLUG-03 rejection loop, so emptiness caused by rejected items keeps reporting as rejections, never reattributed to the operator's match config."
  - "FinishSyncRun delegates to a new FinishSyncRunWithNotice sibling (mirroring ValidateMatchConfig/ValidateMatchConfigWithSuspended) instead of widening its signature, so ~25 existing call sites are untouched."
  - "schemaVersion bumped 2->3 per D-07's documented drop-and-recreate policy; the next kernel start against an existing index.db rebuilds every table and re-syncs once — proven by a test, not just asserted from the policy comment."

patterns-established:
  - "A kernel-composed, config-only advisory (zeroMatchNotice) takes only the resolved field map as input — never the whole webspace, never a plugin response — so it structurally cannot be influenced by anything a plugin returns (A-PLUG-04)."

requirements-completed: [SRC-04]

coverage:
  - id: D1
    description: "An explicit webspace match block that matched zero items across a healthy sync is named (webspace + offending value) in sync_runs.notice and GET /api/sources's last_notice, with last_status staying ok and last_error staying empty"
    requirement: "SRC-04"
    verification:
      - kind: integration
        ref: "kernel/httpapi/sources_test.go#TestSources_ZeroMatchNoticeTravelsFromCorrelateToTheSourcesAPI"
        status: pass
      - kind: integration
        ref: "kernel/syncer/coordinator_test.go#TestSyncOne_ZeroMatchNoticeLeavesStatusOKAndErrorEmpty"
        status: pass
    human_judgment: false
  - id: D2
    description: "The rule is plugin-agnostic (fires for a filesystem-shaped and a differently-shaped instance alike), stays quiet for the keywords fallback / a matching block / rejected-item emptiness, and never masks or is masked by a genuine sync error"
    requirement: "SRC-04"
    verification:
      - kind: unit
        ref: "kernel/correlate/correlate_test.go#TestSyncSource_ExplicitMatchBlockThatMatchedNothingRecordsANotice"
        status: pass
      - kind: unit
        ref: "kernel/correlate/correlate_test.go#TestSyncSource_KeywordsFallbackThatMatchedNothingRecordsNoNotice"
        status: pass
      - kind: unit
        ref: "kernel/correlate/correlate_test.go#TestSyncSource_EveryItemRejectedReportsRejectionsNotAZeroMatchNotice"
        status: pass
      - kind: integration
        ref: "kernel/syncer/coordinator_test.go#TestSyncOne_NoticeNeverMasksASyncError"
        status: pass
      - kind: integration
        ref: "kernel/syncer/coordinator_test.go#TestSyncOne_NoticesFromSeveralWebspacesJoinSortedAndBounded"
        status: pass
    human_judgment: false
  - id: D3
    description: "last_notice is documented as additive at schema_version 1, and the index rebuild an operator's existing file performs on the next kernel start (schemaVersion 2->3) is proven by a test"
    verification:
      - kind: unit
        ref: "kernel/index/store_test.go#TestOpen_IndexAtThePreviousSchemaVersionIsRebuiltAndAcceptsANotice"
        status: pass
      - kind: other
        ref: "bash scripts/check-doc-links.sh"
        status: pass
    human_judgment: false
  - id: D4
    description: "On the user's own desktop the advisory appears against their real files source after the next sync, naming webspace test and the value they typed"
    human_judgment: true
    rationale: "Marked 'backstop' verification in the plan's must_haves — this is a live-desktop human confirmation outside the scope of an automated worktree agent; the kernel-side API field this proof depends on is proven above. The browser-visible surface itself is 12-10-PLAN.md's (next wave)."

duration: ~35min
completed: 2026-08-14
status: complete
---

# Phase 12 Plan 09: Zero-match diagnostic (G-12-1/G-12-3 gap closure) Summary

**A webspace's explicit match block that matched zero items across a healthy sync is now named end to end — kernel-composed, plugin-agnostic, and published as `last_notice` on `GET /api/sources` without ever touching `last_status`/`last_error`.**

## Performance

- **Duration:** ~35 min
- **Started:** 2026-08-14T10:03:35+01:00 (base commit)
- **Completed:** 2026-08-14T10:37:47+01:00 (last task commit)
- **Tasks:** 3/3
- **Files modified:** 10

## Accomplishments

- Closed the second `missing:` item of G-12-1/G-12-3: a match value that could never match any label used to load cleanly, sync `"ok"`, and produce zero rows with no diagnostic anywhere. Now the kernel names it, at the one layer that can see it (`kernel/correlate`), and publishes it all the way to `GET /api/sources`.
- The rule is plugin-agnostic by construction — proven against both a filesystem-shaped instance (a `path` field) and a differently-shaped one (a `labels` field, mirroring the debug session's `test-ext` instance) — because it lives where every source's `Match` result passes through, not in one plugin.
- The advisory is structurally incapable of being an error: it occupies its own SQL column, its own struct field, and its own JSON field end to end, with tests proving it never changes `status`, never writes to `error`, and coexists with a genuine failure without either masking the other.
- `zeroMatchNotice` is composed exclusively from the resolved field map — a webspace name and the match values `matchFieldsFor` resolved from configuration — so no plugin-returned value, item title, `source_id` or label can ever enter it (A-PLUG-04, pinned by a grep-based acceptance criterion on the function body).
- The index rebuild an operator's existing `index.db` performs on the next kernel start (`schemaVersion` 2→3, D-07's drop-and-recreate policy) is proven by a test that forces the file back to the previous version and asserts the pre-existing row is gone, not merely that `Open` returns no error.

## Task Commits

Each task was committed atomically:

1. **Task 1: End to end — an explicit match block that matched nothing becomes a named advisory on GET /api/sources** - `8db1a49` (feat)
2. **Task 2: The rule's boundaries — it fires exactly when it should, stays quiet otherwise, and never masks a real failure** - `bdf839c` (test)
3. **Task 3: Publish the field honestly, and prove the upgrade path an existing index takes** - `dfdf8e9` (docs)

**Plan metadata:** (this SUMMARY's own commit, made by the orchestrator/executor after this file lands)

## Files Created/Modified

- `kernel/correlate/correlate.go` - `WebspaceResult.Notice`, `matchFieldsFor`'s new `explicit` result, `zeroMatchNotice`, and the detection wired into `SyncSource`
- `kernel/correlate/correlate_test.go` - detection boundary tests, `zeroMatchNotice` format tests, and the five existing `matchFieldsFor` call sites widened for the new result
- `kernel/index/schema.go` - `sync_runs.notice` column, `schemaVersion` 2→3
- `kernel/index/store.go` - `SyncRun.Notice`, `FinishSyncRunWithNotice`, `FinishSyncRun` reduced to a delegating wrapper, three SELECT/Scan readers widened
- `kernel/index/store_test.go` - `FinishSyncRunWithNotice` persistence test and the schema-rebuild-accepts-a-notice test
- `kernel/syncer/coordinator.go` - `RunResult.Notice`, `joinNotices`, `maxJoinedNotices`, `syncOne` wired to `FinishSyncRunWithNotice`
- `kernel/syncer/coordinator_test.go` - status/error isolation, error-coexistence, and bounded-join tests through a real store+engine+coordinator
- `kernel/httpapi/sources.go` - `sourceStatus.LastNotice` (`last_notice`), populated in the probe-derived loop only
- `kernel/httpapi/sources_test.go` - the end-to-end test wiring a real `Store`+`Engine`+`Coordinator` behind `Router`
- `docs/api.md` - `last_notice` documented in the `GET /api/sources` section: example JSON, field bullet, additive/`schema_version`-1 statement, do-not-parse discipline

## Decisions Made

- The notice is post-sync and advisory only, never a config rejection — a load-time or save-time rejection of a glob-shaped value would lock the operator out of the very config they need to edit to fix it. `kernel/config` and `kernel/pluginhost/matchconfig.go` were deliberately left untouched (verified: `git diff --name-only` shows no changes under either directory).
- `zeroMatchNotice` tests `resp.GetItems()` — the count the plugin actually returned — strictly before the PLUG-03 rejection loop, so an every-item-rejected sync reports as rejections, never as a match-configuration problem.
- `FinishSyncRun` delegates to a new `FinishSyncRunWithNotice` sibling rather than widening its own signature, mirroring `kernel/pluginhost/matchconfig.go`'s `ValidateMatchConfig`/`ValidateMatchConfigWithSuspended` shape — this repo's existing precedent for exactly this situation (~25 existing call sites, one new caller with an extra parameter).
- `joinNotices` sorts before joining and bounds itself at `maxJoinedNotices` (5) with a suppressed-count suffix, so a source fanned across many zero-matching webspaces produces a deterministic, bounded string rather than a map-iteration-ordered or unbounded one.
- The tracer task's (Task 1) post-commit human-verify checkpoint was treated as satisfied by the tracer's own passing `<verify>` (full test suite + every acceptance-criteria grep/awk check green) rather than pausing for an interactive human, because this plan ran as a non-interactive worktree parallel executor (`workflow.auto_advance` is `false`, but there is no live human available mid-wave to answer a checkpoint) and the plan carries `autonomous: true`. Documented here per the deviation-rules judgment call, not silently skipped.

## Deviations from Plan

None — plan executed exactly as written; no Rule 1-4 auto-fixes were needed. The only judgment call was the tracer-checkpoint handling noted above (a process decision, not a code deviation).

## Issues Encountered

- `gsd-tools windows append` (an attempt to log the unrun `make e2e` step to the cross-phase defect ledger) failed with `Ledger entry 5 has invalid status: "resolved"` — a pre-existing data issue in `.planning/WINDOWS.md` (entry 5, phase 10, unrelated to this plan) that the tool's schema validator rejects. Left the ledger untouched (out of scope for this plan) and recorded the unrun-verify item here instead. Documented as `## Known Stubs` / verification gap below.

## Known Stubs

None — no stub data, hardcoded empty renders, or placeholder text were introduced.

## Verification Gaps

- **`make e2e` was not run.** This plan touches zero files under `web/` (an explicit non-goal) and adds only a backend-additive `last_notice` field, so no existing Playwright spec should be disturbed — but this was not confirmed by actually running the suite. The worktree had no `web/node_modules` installed and running `npm ci` + `npx playwright install` + the full browser suite was judged out of proportion to a backend-only change under time constraints. Every other item in the plan's `<verification>` block (`go test` across all four touched packages plus `-race`, `kernel/config`/`kernel/pluginhost` untouched, `make test-portable`, `go test ./internal/audit/`, `go build ./...`, `go vet ./kernel/...`, `bash scripts/check-doc-links.sh`) was run and passed. Attempted to log this to `.planning/WINDOWS.md` via `gsd-tools windows append --kind unrun-verify`; the append failed on a pre-existing, unrelated schema violation in the ledger (see Issues Encountered) — a human or a future `/gsd-ship` run should either fix ledger entry 5's `status` field or manually add this unrun-verify entry.

## Next Phase Readiness

- The kernel-side gate this plan closes (`sync_runs.notice`, `RunResult.Notice`, `GET /api/sources`'s `last_notice`) is ready for `12-10-PLAN.md` (next wave) to make it browser-visible on the source chip — that plan owns everything under `web/`.
- No blockers. The schema bump (2→3) means the next kernel start against a pre-existing `index.db` will rebuild every index table and re-sync once — this is D-07's documented, sanctioned cost, not a surprise, and it does not affect this worktree's own test runs (each test opens a fresh temp-file store).

---
*Phase: 12-filesystem-source*
*Completed: 2026-08-14*
