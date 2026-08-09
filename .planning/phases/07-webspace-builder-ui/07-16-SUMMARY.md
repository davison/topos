---
phase: 07-webspace-builder-ui
plan: 16
subsystem: api
tags: [participation-predicate, config-apply, sync-correctness, go, sveltekit, gap-closure, webspace-builder]

# Dependency graph
requires:
  - phase: 07-webspace-builder-ui (07-11, 07-14, 07-15)
    provides: D-20 empty-shell semantics and the has-match-input rule (07-11), the client-side participatingInstances mirror (07-14), and the config-aware existence gate this plan's route file edits sit alongside (07-15)
provides:
  - "correlate.ParticipatesIn (kernel/correlate/correlate.go) — the single kernel-side participation predicate, called by both the sync path (matchFieldsFor) and the config-apply path (the new purge)"
  - "Supervisor.purgeDeparticipatedWebspaceRows (kernel/supervisor/supervisor.go) — the synchronous local-index purge that clears a de-participated (webspace, source) pair's rows before Apply answers"
  - "load()'s quiet option and ensurePolling's sync-completion refetch (web/src/routes/w/[webspace]/+page.svelte) — a background sync now updates the stream, not only the chips, without ever showing a skeleton over an already-rendered view"
affects: []

actuals:
  tokens: 13059
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "One shared participation predicate (correlate.ParticipatesIn), asked by both the sync path and the config-apply path, mirroring web/src/lib/participation.ts's own client-side conjunction — closes the exact drift class 07-16's must_haves named as the risk (two definitions of participation disagreeing)"
    - "A synchronous, plugin-RPC-free purge (participation diff scoped to the intersection of both configs' webspace and instance names) as the request-path-safe half of D-07's eager reconcile, paired with a deliberately-still-detached eager resync for the half that needs a real plugin round trip"
    - "A quiet load() mode (skip the loading transition, no-op on failure) as the general shape for a background refetch that must never regress an already-rendered view — reusable wherever else a poll-driven refresh needs the same guarantee"

key-files:
  created:
    - web/src/routes/webspace-stream-refresh.test.ts
  modified:
    - kernel/correlate/correlate.go
    - kernel/correlate/correlate_test.go
    - kernel/supervisor/supervisor.go
    - kernel/supervisor/supervisor_test.go
    - web/src/routes/w/[webspace]/+page.svelte

key-decisions:
  - "The purge is synchronous (a pure local index write); the eager resync it sits beside remains detached (go coord.Refresh) — coupling a config save's latency to a plugin round trip was explicitly rejected (T-07-62)."
  - "The purge's diff is scoped to webspace names AND instance ids present in BOTH configs — a webspace deleted from the config, or an instance removed from it entirely, is excluded by construction, not by a runtime check, closing the two failure modes the plan's prohibitions and threat register (T-07-64) name."
  - "matchFieldsFor's first two conditions (the allowlist gate and the has-match-input rule) were extracted behind ParticipatesIn rather than duplicated — its own four rules and their order, and all four pre-existing TestMatchFieldsFor_* test bodies, are unchanged."
  - "The purge's error is joined last (vocabulary error leads, cleanup error next, purge error last) via errors.Join, preserving every pre-07-16 single-fault error message byte-for-byte."
  - "The client's sync-completion refetch is quiet: it skips load()'s loading-state assignment entirely and, on failure, returns after the generation check without touching response or loadState — a failed background refresh is strictly better than never refreshing, never worse."

patterns-established:
  - "correlate.ParticipatesIn / web/src/lib/participation.ts as a matched kernel/client pair — any future change to either must land in the same commit as the other."

requirements-completed: [KERN-08, UI-12]

coverage:
  - id: D1
    description: "One kernel-side participation predicate (ParticipatesIn) is asked by both the sync path (matchFieldsFor) and the config-apply path (the purge); its answer agrees with matchFieldsFor's own second return value across every shape in the plan's behavior table, including nil-collection and D-20-shell edge cases"
    requirement: KERN-08
    verification:
      - kind: unit
        ref: "kernel/correlate/correlate_test.go#TestParticipatesIn_ResolutionShapes (7 sub-cases, each also asserting agreement with matchFieldsFor)"
        status: pass
      - kind: unit
        ref: "kernel/correlate/correlate_test.go#TestMatchFieldsFor_ExplicitBlockReplacesFallback / TestMatchFieldsFor_FallbackFansAcrossTwoFieldVocabulary / TestMatchFieldsFor_DeallowlistedInstanceDoesNotParticipate / TestMatchFieldsFor_NoBlockAndNoKeywordsDoesNotParticipate (all four pre-existing, unmodified bodies)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Narrowing a webspace's allowlist to exclude a still-configured instance clears exactly that (webspace, instance) pair's stream rows synchronously — asserted on the statement immediately after Apply returns, with no sleep or polling — while the same instance's rows in every other participating webspace, and its own items rows, survive untouched"
    requirement: KERN-08
    verification:
      - kind: unit
        ref: "kernel/supervisor/supervisor_test.go#TestApply_PurgesDeparticipatedWebspaceRows_NarrowingClearsOnlyTheFlippedPair"
        status: pass
      - kind: unit
        ref: "kernel/supervisor/supervisor_test.go#TestApply_PurgesDeparticipatedWebspaceRows_LastSourceRemovedLeavesEmptyShellStreamingNothing"
        status: pass
    human_judgment: false
  - id: D3
    description: "The purge never touches a webspace deleted from the config, or an instance removed from it entirely (both excluded by the diff's own scope), and performs no write at all when no pair's participation flipped; a per-pair clear failure is collected and named, never returned early, and joined into Apply's existing error after the cleanup error"
    requirement: KERN-08
    verification:
      - kind: unit
        ref: "kernel/supervisor/supervisor_test.go#TestApply_PurgesDeparticipatedWebspaceRows_DeletedWebspaceRowsUntouched"
        status: pass
      - kind: unit
        ref: "kernel/supervisor/supervisor_test.go#TestApply_PurgesDeparticipatedWebspaceRows_NoOpConfigPerformsNoClear"
        status: pass
      - kind: unit
        ref: "kernel/supervisor/supervisor_test.go#TestApply_PurgesDeparticipatedWebspaceRows_FailureIsJoinedIntoApplyError"
        status: pass
    human_judgment: false
  - id: D4
    description: "The sync-completion poll's stop branch refetches the stream quietly (never blanking an already-rendered view into the loading skeleton, never disturbing the view on a failed background refresh) on the generation captured before the tick's first await; load()'s pre-existing generation check and typed not-found classification (07-15) survive unchanged; handleRemoveSource's single immediate refetch is untouched"
    requirement: UI-12
    verification:
      - kind: unit
        ref: "web/src/routes/webspace-stream-refresh.test.ts (16 assertions across 6 describe blocks)"
        status: pass
    human_judgment: false
  - id: D5
    description: "Live-kernel verification (make dev): removing a source via the chip menu makes the chip and its items disappear together with no manual refresh; a second webspace containing the same instance keeps its items; re-adding the instance restores its chip and, once its sync completes, its items with no manual refresh; a background sync completing on a viewed webspace does not flash a loading skeleton"
    verification: []
    human_judgment: true
    rationale: "Plan's own <human-check> block — requires a live make dev kernel/browser session; not automatable from this execution environment. Deferred to the phase's end-of-phase human-verify pass (workflow.human_verify_mode: end-of-phase), matching 07-15's own D5."

duration: ~55min
completed: 2026-08-09
status: complete
---

# Phase 07 Plan 16: One Participation Predicate; Synchronous Webspace-Row Purge; Quiet Post-Sync Refetch Summary

**Removing a source from a webspace now purges its stream rows in the same synchronous call that answers the config save — via one kernel-side participation predicate shared by the sync path and the apply path — and a background sync's completion now quietly refreshes the stream, not only the source chips, closing 07-UAT.md `G-07-7` on both its immediate and residual cases.**

## Performance

- **Duration:** ~55 min
- **Completed:** 2026-08-09
- **Tasks:** 3/3
- **Files modified:** 6 (1 new, 5 modified)

## Accomplishments

- `kernel/correlate/correlate.go`'s new `ParticipatesIn` extracts `matchFieldsFor`'s first two conditions (Phase 5 D-03's allowlist gate, 07-11's D-20 has-match-input rule) behind one exported, plugin-independent predicate — `matchFieldsFor` now calls it instead of restating either condition, with all four of its documented rules, their order, and all four pre-existing `TestMatchFieldsFor_*` test bodies unchanged.
- `kernel/supervisor/supervisor.go`'s new `purgeDeparticipatedWebspaceRows`, called from `Apply`'s post-Reconcile region (after the D-07 removed-instance cleanup, before the match-vocabulary check, before the single commit site), diffs old-vs-new participation per `(webspace, source)` pair — scoped to names present in BOTH configs on both axes — and clears exactly the pairs that flipped from participating to not, via `ReplaceWebspaceSourceItems(..., nil)`: a pure local index write with no plugin RPC, so it runs synchronously without coupling a save's latency to a plugin's reachability.
- The purge's error joins into `Apply`'s existing `errors.Join` chain last (vocabulary error still leads, cleanup error next, purge error last) — every pre-07-16 single-fault error message stays byte-identical.
- `web/src/routes/w/[webspace]/+page.svelte`'s `load()` gained an optional `quiet` flag: skips the loading-state assignment (so `StreamList.svelte`'s skeleton, which renders purely off `state === 'loading'`, can never replace an already-rendered stream) and, on a quiet failure, returns after the generation check without touching `response` or `loadState`.
- `ensurePolling`'s stop branch (the tick where syncing falls to `false`) now also awaits a quiet `load()` for the generation captured before the tick's first await — so a webspace whose eager resync failed at save time, and whose rows a later scheduled sync purges via `correlate.SyncSource`'s pre-existing non-participation branch, also heals on screen with no manual refresh.
- New `webspace-stream-refresh.test.ts`: a 16-assertion, comment-stripped source-scan guard pinning the stop-branch refetch, its quiet flag, the generation-capture-before-first-await ordering, `load()`'s guarded loading assignment, its preserved catch behavior (07-15's generation check and typed not-found classification), and `handleRemoveSource`'s untouched single refetch.

## Task Commits

Each task was committed atomically:

1. **Task 1: One kernel-side answer to whether a source belongs to a webspace** - `1713c65` (feat, tdd)
2. **Task 2: By the time the save is answered, the index agrees with the config** - `83250bc` (feat, tdd)
3. **Task 3: A finished sync updates the stream, not just the chips** - `4337b76` (feat)

## Files Created/Modified

- `kernel/correlate/correlate.go` - new `ParticipatesIn`; `matchFieldsFor` calls it instead of inlining its first two conditions
- `kernel/correlate/correlate_test.go` - `TestParticipatesIn_ResolutionShapes` (7 sub-cases + per-case agreement assertion against `matchFieldsFor`)
- `kernel/supervisor/supervisor.go` - new `purgeDeparticipatedWebspaceRows`; `Apply` calls it and joins its error; doc comment extended
- `kernel/supervisor/supervisor_test.go` - 5 new tests (core narrowing, last-source/D-20-shell, no-op, deleted-webspace, batched-failure)
- `web/src/routes/w/[webspace]/+page.svelte` - `load()`'s quiet option; `ensurePolling`'s sync-completion refetch
- `web/src/routes/webspace-stream-refresh.test.ts` - new source-scan guard (16 assertions)

## RED Confirmations (recorded per plan's `<output>` requirement)

**Task 1 — the correlate predicate cases, run before `ParticipatesIn` existed:**

```
# github.com/davison/topos/kernel/correlate [github.com/davison/topos/kernel/correlate.test]
kernel/correlate/correlate_test.go:537:14: undefined: ParticipatesIn
FAIL	github.com/davison/topos/kernel/correlate [build failed]
```

A build-failure RED, as expected for an extraction whose tests reference the not-yet-existing symbol. After the extraction, all 7 `TestParticipatesIn_ResolutionShapes` sub-cases passed, and all four pre-existing `TestMatchFieldsFor_*` tests passed with unmodified bodies.

**Task 2 — the core narrowing case, run before the purge existed** (`TestApply_PurgesDeparticipatedWebspaceRows_NarrowingClearsOnlyTheFlippedPair`, against `Apply` with `purgeDeparticipatedWebspaceRows` not yet wired in):

```
--- FAIL: TestApply_PurgesDeparticipatedWebspaceRows_NarrowingClearsOnlyTheFlippedPair (0.46s)
    supervisor_test.go:888: expected ws1 to no longer contain beta's item — beta was excluded by
    the narrowed allowlist and its rows must be purged by the time Apply returns, got: [...alpha:1..4, beta:1]
--- FAIL: TestApply_PurgesDeparticipatedWebspaceRows_LastSourceRemovedLeavesEmptyShellStreamingNothing (0.01s)
    supervisor_test.go:971: expected shell-target to stream nothing once its last participating source was removed, got: [{ID:solo:1 ...}]
--- FAIL: TestApply_PurgesDeparticipatedWebspaceRows_FailureIsJoinedIntoApplyError (0.01s)
    supervisor_test.go:1157: expected Apply to return a non-nil error — both clear calls fail against a closed index store
```

`TestApply_PurgesDeparticipatedWebspaceRows_NoOpConfigPerformsNoClear` and `..._DeletedWebspaceRowsUntouched` passed even pre-purge (a no-op/absent purge trivially satisfies "nothing was cleared") — they became meaningful regression guards once the purge landed, not RED signals in themselves; this is noted rather than hidden. After wiring `purgeDeparticipatedWebspaceRows` into `Apply`, all five tests passed, including under `go test ./kernel/supervisor/ -race` (see also "Issues Encountered" below for a pre-existing, unrelated flake found and ruled out during this verification).

**Task 3 — the client poll guard's stop-branch assertion, run against a temporary, git-diff-confirmed-clean revert of the `await load(gen, { quiet: true });` line:**

```
FAIL  src/routes/webspace-stream-refresh.test.ts > ensurePolling's stop branch refetches the stream,
      quietly, on the captured generation > calls load( inside the interval callback, passing the quiet flag
AssertionError: expected the stop branch to call load(gen, { quiet: true }) ...: expected false to be true
FAIL  ... > the load( call sits after clearInterval, inside the "no longer syncing" branch
AssertionError: expected the quiet stream refetch to be present in the stop branch: expected -1 to be greater than -1
 Test Files  1 failed (1)
      Tests  2 failed | 14 passed (16)
```

The fix was restored immediately after capturing this output; `git diff --stat` of `+page.svelte` before and after the temporary revert/restore was confirmed identical (41 insertions, 2 deletions unchanged). All 16 assertions passed after restoration.

## Verification Results

- `CGO_ENABLED=0 go build ./...` — exit 0
- `go test ./kernel/... -count=1` — every package `ok` (config, correlate, httpapi, index, pluginhost, supervisor, syncer), including all six pre-existing `TestApply_*` tests and all four pre-existing `TestMatchFieldsFor_*` tests with unmodified bodies
- `go test ./kernel/supervisor/ -count=1 -race` — exit 0
- `cd web && npm test` — 36 test files, 601 tests, all pass (up from 07-15's 35/585 baseline: +1 file, +16 tests, exactly this plan's new guard)
- `cd web && npm run check` — 0 errors, the same 9 pre-existing warnings in unrelated files as 07-15's baseline
- `cd web && npm run build` — exit 0
- `git diff --stat go.mod go.sum web/package.json web/package-lock.json` — no output (no dependency added)
- `git diff --stat` against `kernel/httpapi/`, `kernel/index/`, `kernel/syncer/`, `plugins/`, `proto/` — no output (no handler, store method, plugin, or contract changed)
- `grep -c 'correlate.ParticipatesIn(' kernel/supervisor/supervisor.go` — 2 (the old-config and new-config sides of the diff)
- `git diff --stat` across all three task commits — lists exactly the 6 files in the plan's `files_modified`, nothing else
- Read every new supervisor test body: no `sleep`, `time.After`, retry loop, or eventually-helper anywhere — every assertion runs on the statement immediately following `Apply`'s return
- `grep -riE 'TODO|FIXME|XXX|HACK'` over every new/modified file in this plan — no matches

## Decisions Made

- The purge's tests seed fixture items directly via `idx.ReplaceWebspaceSourceItems` (mirroring `TestApply_RemovedInstance_PluginGoneAndIndexRowsGone`'s own established pattern) rather than depending on the real mock plugin's boot-time sync to populate "before" state — discovered live while writing Task 2's tests: calling `Apply` immediately after `NewSupervisor` (as every "no sleep" assertion in this plan requires) cancels the boot scheduler generation's still-in-flight Match RPC via `stopScheduler`'s context cancellation, before it necessarily has time to persist anything, making a boot-sync-dependent "before" state genuinely racy rather than merely slow. Not a plan deviation — the plan named the seeding pattern implicitly via `testFixtureItem` and this is that pattern applied consistently; recorded here because the reasoning (an RPC-cancellation race, not a polling race) is the kind of detail a future reader debugging a flake in this file would need.
- The purge's wrapped error text (`"clear webspace %q source %q: %w"`) reuses `correlate.SyncSource`'s own existing phrasing verbatim for its identical non-participation clear path, rather than inventing new wording — consistent operator-facing vocabulary for the same underlying operation from two call sites.

## Deviations from Plan

None — plan executed exactly as written. All three tasks' `git diff --stat` outputs matched their declared `<files>` lists exactly, every prohibition held (the eager resync stayed detached via `go coord.Refresh`; no HTTP handler, index store method, plugin, or proto file was touched; no existing test body in `correlate_test.go`, `supervisor_test.go`, or `web/src/**/*.test.ts` was modified; no timer/delay/retry was added to `handleRemoveSource` or anywhere else in the route file), and both explicitly-flagged-not-planned items (read-time participation filtering in the stream handler, and orphaned rows for a webspace deleted from the config) were left untouched exactly as planning choices 7 and 8 specify.

## Issues Encountered

- While verifying Task 2 under `-race`, discovered `TestApply_MidFlightSyncLeavesNoStrandedRunningRow` (a pre-existing test, unmodified by this plan) fails intermittently (~2/15 runs) when the full `kernel/supervisor` package runs under `-race` — but never when run in isolation, and at an identical rate against the pre-07-16 commit with this plan's changes fully reverted (2/15 failures on unmodified code, same failure signature: the mid-flight sync's detached `sync_runs` finalize goroutine hasn't completed by the time the test's own immediately-following assertion runs, under the added scheduling latency `-race`'s instrumentation and a growing package test suite introduce). Confirmed pre-existing and unrelated to this plan's changes via a controlled A/B: 25 total runs of the untouched package (10 in one batch, 15 in another) all passed, then a fair 15-run batch on the SAME untouched code reproduced the identical 2/15 failure rate this plan's own changes showed. Not fixed here — out of this gap's declared scope (`kernel/syncer/` and the mid-flight sync handling are untouched by `files_modified`) — but recorded so it is not mistaken for a regression introduced by this plan if seen again in CI.

## Known Stubs

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- 07-UAT.md `G-07-7` is closed on both its immediate case (the purge lands before the save answers) and its residual case (a failed eager resync's rows are cleared by a later scheduled sync, and the client now notices).
- Live-kernel human verification (`make dev`: chip-menu remove disappears the chip and its items together; a second webspace containing the same instance is untouched; re-adding the instance restores its chip and, post-sync, its items with no manual refresh; a background sync completing on a viewed webspace does not flash a loading skeleton) has NOT been run in this execution environment — recorded as coverage item D5, deferred to the phase's end-of-phase human-verify pass per `workflow.human_verify_mode: end-of-phase`, matching 07-15's own D5.
- This was the last plan in Phase 07's 07-UAT.md gap-closure sequence (07-11 through 07-16); ROADMAP/STATE/REQUIREMENTS updates below advance the phase's plan counter accordingly.

---
*Phase: 07-webspace-builder-ui*
*Completed: 2026-08-09*

## Self-Check: PASSED

All created/modified files (kernel/correlate/correlate.go, correlate_test.go, kernel/supervisor/supervisor.go, supervisor_test.go, web/src/routes/w/[webspace]/+page.svelte, web/src/routes/webspace-stream-refresh.test.ts) confirmed present on disk. All three commits (1713c65, 83250bc, 4337b76) confirmed present in `git log`.
