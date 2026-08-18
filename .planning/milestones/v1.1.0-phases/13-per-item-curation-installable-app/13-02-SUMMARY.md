---
phase: 13-per-item-curation-installable-app
plan: 02
subsystem: curation
tags: [sqlite, chi, go]

requires:
  - phase: 13-01
    provides: "item_marks table, SetItemMarks/ClearItemMarks/CountItemMarks, markFilterClause composed into StreamItems/Search, POST /api/webspaces/{webspace}/marks"
provides:
  - "index.MarkView (ViewIncluded/ViewExcluded) widening Store.StreamItems — the one place the IN/NOT IN branch is chosen, with the zero value defaulting safely to ViewIncluded"
  - "?view= query parameter on GET /api/webspaces/{webspace}/stream, parsed AFTER the webspaceIsKnown gate, rejecting anything but included/excluded with 400 invalid_request"
  - "streamResponse.excluded_count, populated on every stream request in both views from CountItemMarks"
  - "pruneItemMarksTx — the orphan mark sweep inside ReplaceWebspaceSourceItems' existing transaction, scoped to (webspace, source) and structurally unreachable on a failed sync or an index rebuild"
  - "three-surface exclusion parity proven by test: stream, search, and the /agent/v1 stream mirror agree on the partition"
  - "docs/api.md: the marks route, ?view=, excluded_count, and the mark-lifecycle contract published"
affects: [13-03, 13-04, 13-05, 13-06]

actuals:
  tokens: 18077
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "StreamItems' view axis is a typed MarkView (not a bare string) whose zero value resolves to the safe default (ViewIncluded) at the SQL-branch-selection site — an unset argument can never accidentally surface the excluded bucket, mirroring the codebase's existing 'unset means the pre-existing default' discipline"
    - "The orphan prune sweep mirrors the existing source-scoped webspace_items delete's exact subquery shape (IN (SELECT id FROM items WHERE source = ?)) rather than inventing a second scoping idiom — one ownership-attribution pattern for the whole file"

key-files:
  created: []
  modified:
    - kernel/index/store.go
    - kernel/index/marks_test.go
    - kernel/index/store_test.go
    - kernel/httpapi/stream.go
    - kernel/httpapi/stream_test.go
    - kernel/httpapi/agent.go
    - kernel/httpapi/agent_test.go
    - kernel/correlate/correlate_test.go
    - kernel/supervisor/readiness_test.go
    - kernel/supervisor/supervisor_test.go
    - kernel/syncer/coordinator_test.go
    - docs/api.md

key-decisions:
  - "StreamItems' view parameter defaults to ViewIncluded for both the empty string AND any value other than ViewExcluded — a single normalization point (streamMarkFilterClause) rather than requiring every caller to pass a valid MarkView explicitly"
  - "The prune sweep's empty-kept-id branch (de-allowlist call, items=nil) omits the NOT IN clause entirely rather than emitting 'NOT IN ()' (invalid SQL) — the resulting 'delete every mark for this source in this webspace' behavior is PD-02's decided consequence, named at the branch by comment"
  - "The interrupted-sync atomicity test uses a second raw sql.DB connection to the same file holding a BEGIN IMMEDIATE write lock, forcing the store's own transaction to fail with SQLITE_BUSY — a genuine cross-connection lock-contention proof rather than a context-cancellation shortcut that wouldn't exercise the same failure path"

patterns-established:
  - "MarkView's zero-value-safe design: any new StreamItems caller that forgets to pass a view explicitly gets ViewIncluded, never the excluded bucket — the safe failure direction is baked into the type rather than relying on every caller remembering"

requirements-completed: [KERN-09, KERN-10]

coverage:
  - id: D1
    description: "GET /api/webspaces/{webspace}/stream?view=excluded returns exactly the items carrying an excluded mark for that webspace, in the same chronological order the normal stream uses, and the two views are exact complements of the unfiltered item set"
    requirement: KERN-10
    verification:
      - kind: unit
        ref: "kernel/index/marks_test.go#TestStreamItems_ViewExcludedReturnsExactlyMarkedItems"
        status: pass
      - kind: unit
        ref: "kernel/index/marks_test.go#TestStreamItems_IncludedAndExcludedViewsAreComplements"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/stream_test.go#TestStreamHandler_ViewExcludedReturnsOnlyMarkedItems"
        status: pass
    human_judgment: false
  - id: D2
    description: "A view value other than included/excluded is rejected 400 invalid_request naming both allowed values; an unknown webspace still answers 404 regardless of view; a zero-mark webspace's excluded view answers 200 with an empty array and excluded_count 0"
    requirement: KERN-10
    verification:
      - kind: unit
        ref: "kernel/httpapi/stream_test.go#TestStreamHandler_ViewBogusReturns400InvalidRequest"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/stream_test.go#TestStreamHandler_ViewExcludedUnknownWebspaceReturns404"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/stream_test.go#TestStreamHandler_ViewExcludedZeroMarksReturns200EmptyArrayWithZeroCount"
        status: pass
    human_judgment: false
  - id: D3
    description: "The stream response carries excluded_count in both views, populated fresh on every request"
    requirement: KERN-10
    verification:
      - kind: unit
        ref: "kernel/httpapi/stream_test.go#TestStreamHandler_ViewExcludedReturnsOnlyMarkedItems"
        status: pass
    human_judgment: false
  - id: D4
    description: "When a healthy sync of a (webspace, source) pair no longer reports an item, that item's mark for that webspace is swept in the same transaction as the item's own webspace_items row — scoped so it never touches a sibling source's or a sibling webspace's marks"
    requirement: KERN-09
    verification:
      - kind: unit
        ref: "kernel/index/marks_test.go#TestReplaceWebspaceSourceItems_OmittedExcludedItemIsPruned"
        status: pass
      - kind: unit
        ref: "kernel/index/marks_test.go#TestReplaceWebspaceSourceItems_SweepNeverTouchesSiblingSourceMarks"
        status: pass
      - kind: unit
        ref: "kernel/index/marks_test.go#TestReplaceWebspaceSourceItems_SweepNeverTouchesOtherWebspaceMarks"
        status: pass
      - kind: unit
        ref: "kernel/index/marks_test.go#TestReplaceWebspaceSourceItems_ResyncedExcludedItemKeepsItsMark"
        status: pass
    human_judgment: false
  - id: D5
    description: "A failed sync never prunes a mark (structurally unreachable, not merely skipped by a runtime check); an index rebuild never prunes a mark; an interrupted sync leaves both items and marks exactly as they were; a de-allowlist call clears that source's marks for that webspace (PD-02); a pruned-then-restored item re-enters the stream unexcluded"
    requirement: KERN-09
    verification:
      - kind: unit
        ref: "kernel/correlate/correlate_test.go#TestSyncSource_MatchFailureNeverPrunesAMark"
        status: pass
      - kind: unit
        ref: "kernel/index/marks_test.go#TestReplaceWebspaceSourceItems_InterruptedSyncLeavesItemsAndMarksUnchanged"
        status: pass
      - kind: unit
        ref: "kernel/index/marks_test.go#TestReplaceWebspaceSourceItems_DeallowlistClearsThatSourcesMarks"
        status: pass
      - kind: unit
        ref: "kernel/index/marks_test.go#TestReplaceWebspaceSourceItems_ReappearingItemIsUnexcluded"
        status: pass
    human_judgment: false
  - id: D6
    description: "The /agent/v1 stream mirror and in-webspace search return the identical excluded/included partition the human stream does, proven against all three surfaces in one test"
    requirement: KERN-09
    verification:
      - kind: unit
        ref: "kernel/httpapi/agent_test.go#TestThreeSurfaceParity_ExcludedItemAgreesAcrossStreamSearchAndAgentMirror"
        status: pass
    human_judgment: false
  - id: D7
    description: "The marks route, ?view= parameter, excluded_count field, and the mark-lifecycle contract (survival/removal cases, de-allowlist and rename consequences) are published in docs/api.md"
    verification:
      - kind: other
        ref: "make docs-check"
        status: pass
    human_judgment: false

duration: ~40min
completed: 2026-08-14
status: complete
---

# Phase 13 Plan 02: Excluded Bucket, Orphan Prune Sweep & Cross-Surface Contract Summary

**`?view=excluded` on the stream route with a live `excluded_count`, a healthy-sync-only orphan prune sweep inside `ReplaceWebspaceSourceItems`' existing transaction, and a three-surface (stream/search/agent) exclusion-parity test backed by the published `docs/api.md` contract.**

## Performance

- **Duration:** ~40 min
- **Completed:** 2026-08-14
- **Tasks:** 3
- **Files modified:** 12 (0 created, 12 modified across three commits)

## Accomplishments

- `index.MarkView` (`ViewIncluded`/`ViewExcluded`) widens `Store.StreamItems` at the one place the mark filter's IN/NOT IN branch is already chosen — the zero value resolves safely to `ViewIncluded`, so no existing or future caller can accidentally surface the excluded bucket by omission
- `GET /api/webspaces/{webspace}/stream?view=` parses the query parameter AFTER the `webspaceIsKnown` existence gate (unknown webspace still 404s regardless of `view`), rejects anything but `included`/`excluded` with `400 invalid_request`, and populates the new `excluded_count` field on every request in both views
- `pruneItemMarksTx` — the orphan mark sweep — runs inside `ReplaceWebspaceSourceItems`' existing transaction, immediately after the reinsert loop, scoped by the kernel's own `(webspaceName, source)` parameters (mirroring the existing source-scoped `webspace_items` delete's subquery shape) so it can never cross into a sibling source's or sibling webspace's marks; a failed `Match` never reaches this method at all, and an index rebuild drops item rows before any sync runs, so both paths are structurally unreachable
- The de-allowlist branch (`items=nil`) clears the source's marks for that webspace entirely (PD-02), named at the branch by comment alongside D-09
- A three-surface parity test proves stream, search, and the `/agent/v1` stream mirror agree on exactly which items are excluded, against one temp index and one config
- `docs/api.md` gains the `POST /api/webspaces/{webspace}/marks` route section, the `?view=` parameter and `excluded_count` field on the stream route, a search-side PD-06 note, and a mark-lifecycle paragraph naming all four survival/removal cases plus the de-allowlist and rename consequences

## Task Commits

Each task was committed atomically:

1. **Task 1: The excluded bucket — ?view= on the stream route and excluded_count in the response** - `ea91596` (feat)
2. **Task 2: Orphan prune sweep inside the existing sync transaction** - `c0efad1` (feat)
3. **Task 3: Cross-surface parity tests and the published API contract** - `c27b4d3` (docs)

## Files Created/Modified

- `kernel/index/store.go` - `MarkView`/`ViewIncluded`/`ViewExcluded`, `streamMarkFilterClause`, widened `StreamItems`, `pruneItemMarksTx`, and its call site inside `ReplaceWebspaceSourceItems`
- `kernel/index/marks_test.go` - view-complement/zero-mark tests (Task 1) and all seven prune-sweep behavior tests including the lock-contention interrupted-sync proof (Task 2)
- `kernel/index/store_test.go` - mechanical `StreamItems` call-site widening (view argument) across all 17 pre-existing call sites
- `kernel/httpapi/stream.go` - `parseStreamView`, `?view=` wiring, `streamResponse.ExcludedCount`
- `kernel/httpapi/stream_test.go` - handler tests for every Task 1 acceptance criterion
- `kernel/httpapi/agent.go` - both `StreamItems` call sites pass `index.ViewIncluded` explicitly, with a comment naming why
- `kernel/httpapi/agent_test.go` - the three-surface parity test (Task 3)
- `kernel/correlate/correlate_test.go` - `TestSyncSource_MatchFailureNeverPrunesAMark` (Task 2's structural D-10 proof) plus mechanical call-site widening
- `kernel/supervisor/readiness_test.go`, `kernel/supervisor/supervisor_test.go`, `kernel/syncer/coordinator_test.go` - mechanical `StreamItems` call-site widening outside this plan's declared file scope (see Deviations)
- `docs/api.md` - the marks route section, `?view=`/`excluded_count` on the stream route, search's PD-06 note, the agent mirror's no-view-of-its-own note, the extended `invalid_request` error-code row, and the mark-lifecycle paragraph

## Decisions Made

- `StreamItems`' view parameter normalizes to `ViewIncluded` for both the zero value and any unrecognized value at one site (`streamMarkFilterClause`) rather than requiring every call site to pass a validated `MarkView` — the HTTP layer still rejects a bad `?view=` with 400 before it ever reaches the store, but the store itself fails safe regardless.
- The prune sweep's empty-kept-id branch omits the `NOT IN (...)` clause entirely (an empty SQL `IN` list is a syntax error) rather than special-casing it — the resulting "delete every mark for this source in this webspace" behavior is PD-02's own decided consequence, and the code names PD-02/D-09 explicitly at that branch so a future reader doesn't mistake it for an accidental byproduct of empty-slice handling.
- The interrupted-sync atomicity test opens a genuine second `sql.DB` connection to the same on-disk file and holds a `BEGIN IMMEDIATE` write lock, forcing the store's own transaction to fail with `SQLITE_BUSY` — chosen over a context-cancellation shortcut because it exercises the actual cross-connection lock-contention failure mode a real interrupted sync would hit, not merely "the call never started."

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Widened every `StreamItems` call site in the whole workspace, including files outside this task's declared scope**
- **Found during:** Task 1
- **Issue:** `StreamItems`' signature widened from `(ctx, webspaceName, filterTerms)` to `(ctx, webspaceName, filterTerms, view)` per the plan's own instruction — this broke compilation at every existing call site, including three test files never listed in Task 1's `files_modified` (`kernel/supervisor/readiness_test.go`, `kernel/supervisor/supervisor_test.go`, `kernel/syncer/coordinator_test.go`) plus `kernel/index/store_test.go`, which the plan did list.
- **Fix:** Added `index.ViewIncluded` (or the unqualified `ViewIncluded` inside package `index`) at all 17 store-package call sites and every call site in the three out-of-scope test files, preserving each test's pre-existing behavior byte-for-byte. `kernel/supervisor/readiness_test.go` additionally needed a new `index` package import.
- **Files modified:** kernel/index/store_test.go, kernel/supervisor/readiness_test.go, kernel/supervisor/supervisor_test.go, kernel/syncer/coordinator_test.go
- **Verification:** `go build ./...` and `go vet ./...` clean; `go test ./...` (full workspace) and `make test-portable` both pass.
- **Committed in:** ea91596 (Task 1 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking, mechanical signature-widening propagation)
**Impact on plan:** Necessary consequence of the plan's own deliberate `StreamItems` signature change (13-02-PLAN.md Task 1's explicit instruction). No behavior change to any of the affected tests — every widened call site passes `ViewIncluded`, reproducing pre-Phase-13 output exactly. No scope creep.

## Issues Encountered

None — `go build`/`go vet`/`go test ./...` and `make test-portable`/`make docs-check` all passed clean on first full run after each task, with the SQLite write-lock-contention test (Task 2) verified stable across repeated runs.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The excluded bucket is fully readable (`?view=excluded`, `excluded_count`) and the orphan prune sweep is proven correct against every adjacent failure mode (sibling source, sibling webspace, failed sync, index rebuild, interrupted transaction) — 13-03+ can build the multi-select/action-bar and excluded-view-toggle UI directly against this contract with no further kernel-side curation work expected.
- `docs/api.md` is the published source of truth for `?view=`, `excluded_count`, and the marks route's full lifecycle — later plans documenting UI behavior can cite it rather than re-deriving the rules.
- No blockers. `go vet ./kernel/...`, `go test ./...` (workspace-wide), `make test-portable`, and `make docs-check` all pass clean.

---
*Phase: 13-per-item-curation-installable-app*
*Completed: 2026-08-14*

## Self-Check: PASSED

- FOUND: kernel/index/store.go, kernel/httpapi/stream.go, kernel/httpapi/agent.go, docs/api.md, kernel/httpapi/agent_test.go, kernel/index/marks_test.go, kernel/correlate/correlate_test.go
- FOUND: commits ea91596, c0efad1, c27b4d3 in `git log --oneline --all`
