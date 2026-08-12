---
phase: 02-two-sources-one-trustworthy-stream
plan: 05
subsystem: sync-reliability
tags: [go, errgroup, silverbullet, grpc, error-propagation, gap-closure]

# Dependency graph
requires:
  - phase: 02-two-sources-one-trustworthy-stream
    provides: "kernel/correlate.SyncSource's per-(webspace,source_type) replace path and codes.Unavailable-to-error-status wiring (02-01, 02-02) — this plan supplies the one missing link that makes that chain trustworthy for SilverBullet"
provides:
  - "plugins/silverbullet Match now discriminates ErrNotFound (safe skip) from every other page-read failure (propagated as codes.Unavailable), closing the sole BLOCKER gap from 02-VERIFICATION.md truth #6"
  - "plugins/silverbullet/match_test.go: five TestMatch_* regression tests covering the happy path, the deleted-page skip, total read failure, mid-sync auth revocation, and token non-disclosure"
affects: [phase-02-verification, future-plugin-authors-following-the-Match-contract]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "errgroup per-task error discrimination: only errors.Is(err, sentinel) skips; every other error propagates unchanged so g.Wait() observes it — established here for plugins/silverbullet/plugin.go, generalizable to any future plugin's own errgroup-based Match"

key-files:
  created:
    - plugins/silverbullet/match_test.go
  modified:
    - plugins/silverbullet/plugin.go

key-decisions:
  - "Fail-closed policy confirmed as-specified: ANY non-404 read failure fails the whole Match — no tolerance threshold, no retry loop, no partial-result heuristic, because a partially-read sync is indistinguishable from a complete one once persisted"
  - "No kernel change needed — kernel/correlate.SyncSource and kernel/syncer.Coordinator already correctly turn a Match error into a skipped-persist + error-status run (proven by pre-existing TestSyncSource_PartialSourceFailure_HealthySourceItemsPersist and TestRefresh_MatchErrorReturnsErrorStatusNotGoError); the plugin was the only broken link"

requirements-completed: [SRC-05, UI-05]

coverage:
  - id: D1
    description: "Match propagates every non-ErrNotFound page-read failure as codes.Unavailable instead of a silent successful-empty result"
    requirement: SRC-05
    verification:
      - kind: unit
        ref: "plugins/silverbullet/match_test.go#TestMatch_AllPageReadsFail_ReturnsUnavailable"
        status: pass
      - kind: unit
        ref: "plugins/silverbullet/match_test.go#TestMatch_OutageMidSync_AuthFailure_ReturnsUnavailable"
        status: pass
    human_judgment: false
  - id: D2
    description: "A page that 404s between listing and read is still skipped, not failed — the fix does not over-correct the legitimate deleted-page case"
    requirement: SRC-05
    verification:
      - kind: unit
        ref: "plugins/silverbullet/match_test.go#TestMatch_PageDeletedBetweenListingAndRead_SkippedNotFailed"
        status: pass
      - kind: unit
        ref: "plugins/silverbullet/match_test.go#TestMatch_HappyPath_ReturnsOnlyKeywordMatchedPages"
        status: pass
    human_judgment: false
  - id: D3
    description: "The codes.Unavailable error Match returns on an unreachable/unauthorized source never contains the configured bearer token value"
    requirement: SRC-05
    verification:
      - kind: unit
        ref: "plugins/silverbullet/match_test.go#TestMatch_UnavailableError_NeverContainsBearerToken"
        status: pass
    human_judgment: false
  - id: D4
    description: "No SilverBullet row is deleted and no run is recorded ok when a source failure occurs inside Match (downstream guarantee, unchanged code, re-confirmed by full-suite run)"
    requirement: UI-05
    verification:
      - kind: integration
        ref: "make test (root, sdk, plugins/paperless, plugins/silverbullet, plugins/mock all build and pass)"
        status: pass
    human_judgment: false

duration: 5min
completed: 2026-07-29
status: complete
---

# Phase 02 Plan 05: SilverBullet Match Error-Propagation Fix Summary

**Fixed `plugins/silverbullet` Match to return `codes.Unavailable` (not a successful empty result) when any page read fails for a reason other than HTTP 404, closing the phase's single BLOCKER verification gap.**

## Performance

- **Duration:** ~5 min (task execution); full verification suite run took longer in wall-clock but no manual intervention
- **Started:** 2026-07-29T09:23:00Z (approx, session start)
- **Completed:** 2026-07-29T09:26:22Z
- **Tasks:** 2
- **Files modified:** 2 (1 created, 1 modified)

## Accomplishments
- Added `plugins/silverbullet/match_test.go` with five `TestMatch_*` regression tests; three (`AllPageReadsFail`, `OutageMidSync_AuthFailure`, `UnavailableError_NeverContainsBearerToken`) were confirmed failing against the unmodified `Match` before any fix was applied — the required red-before-green evidence.
- Fixed `Match`'s errgroup closure in `plugins/silverbullet/plugin.go` to discriminate `errors.Is(err, ErrNotFound)` (safe skip — page deleted/renamed between listing and read) from every other `ReadFile` error (now propagated so `g.Wait()` returns it and `Match` responds with `codes.Unavailable`).
- Confirmed no kernel-side change was needed: `kernel/correlate.SyncSource` and `kernel/syncer.Coordinator` already correctly skip persistence and record an error status when `Match` fails — verified by the pre-existing `TestSyncSource_PartialSourceFailure_HealthySourceItemsPersist` and `TestRefresh_MatchErrorReturnsErrorStatusNotGoError` tests remaining green.
- Full workspace test suite (`make test`), `make build`, and the frontend sanity suite (`npm run check`, `npm run test`, 44/44 passing) all confirmed green after the fix.

## Task Commits

Each task was committed atomically:

1. **Task 1: Reproduce the gap — regression tests for Match's error-aggregation path** - `692573e` (test)
2. **Task 2: Propagate non-ErrNotFound read failures out of Match's errgroup** - `ae0fa88` (fix)

_Note: this plan required no docs/config-only closing commit beyond the two task commits; the final metadata commit below captures SUMMARY.md/STATE.md/ROADMAP.md/REQUIREMENTS.md._

## Files Created/Modified
- `plugins/silverbullet/match_test.go` - Five `TestMatch_*` regression tests: happy path, deleted-page skip, total read failure, mid-sync auth revocation, token non-disclosure. Introduces `newMatchTestServer`, `matchTestPage`, `matchTestPageOther`, and `pageName` as new package-scope test helpers (no collision with existing `newClientFixtureServer`/`newFetchTestServer`/`fixtureListing`/`fetchTestPage`).
- `plugins/silverbullet/plugin.go` - `Match`'s per-page `g.Go` closure now returns `err` unchanged for every non-`ErrNotFound` `ReadFile` failure instead of unconditionally returning `nil`; rewrote the closure's comment to document both branches, the 404-only-skip rationale, and the destructive-delete consequence this fix prevents.

## Decisions Made
- Kept the fail-closed policy exactly as specified in the plan: no retry loop, no partial-tolerance threshold — any single non-404 read error fails the entire `Match` call. This mirrors the accepted low-severity availability tradeoff recorded in the plan's own threat register (T-02-05-03).
- No new test helper collided with existing package-scope names in `plugins/silverbullet` (`package main` shares symbol space across `client_test.go`, `fetch_test.go`, and the new `match_test.go`) — confirmed by a clean `go vet ./...` pass.

## Deviations from Plan

None — plan executed exactly as written. Both tasks' acceptance criteria were met verbatim:
- Three of five `TestMatch_*` tests failed against the unmodified `plugin.go` (confirmed via `go test ./... -run 'TestMatch_(AllPageReadsFail|OutageMidSync_AuthFailure|UnavailableError)'` exiting non-zero before Task 2).
- `git diff --stat plugins/silverbullet/plugin.go` was empty after Task 1.
- `plugins/silverbullet/go.mod` and `go.sum` are byte-identical to before this plan (confirmed via `git diff --stat` on both files after each task — no output).
- After Task 2, all five `TestMatch_*` tests pass, `go test ./... -race` is clean, `CGO_ENABLED=0 go build ./...` and `go vet ./...` both exit 0, `make test` and `make build` both succeed, and `git diff --name-only` for Task 2 lists only `plugins/silverbullet/plugin.go`.

## Issues Encountered

None. The one internal iteration was self-corrective: the first draft of `match_test.go` included a hand-rolled `itoa` helper to avoid adding an import, then was simplified to use the standard library's `strconv.Itoa` (already implicitly available, adds no module-graph dependency, and is simpler/clearer than a hand-rolled digit-reversal loop) before the file was ever committed.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The phase's sole BLOCKER gap (02-VERIFICATION.md truth #6) is closed. Re-verification should re-check `TestMatch_AllPageReadsFail_ReturnsUnavailable`, `TestMatch_OutageMidSync_AuthFailure_ReturnsUnavailable`, and `Match`'s `errors.Is(err, ErrNotFound)` discrimination directly, per the plan's own re-verification guidance — not SUMMARY prose.
- Deliberately still outstanding and NOT addressed by this plan (per its own scope notes): the two Human Verification items (browser-visual confirmation of the 02-02/02-03/02-04 UI surfaces; a genuinely isolated PLUG-05 re-run) and warnings WR-01 through WR-04 from `02-REVIEW.md` (DetailPane timezone formatter, missing iframe `sandbox` attribute, silently-downgraded `LatestSyncRunPerSource` errors, absent graceful-shutdown signal handling). These remain candidates for a follow-up fix pass, not gaps blocking phase completion.

---
*Phase: 02-two-sources-one-trustworthy-stream*
*Completed: 2026-07-29*

## Self-Check: PASSED

- FOUND: plugins/silverbullet/match_test.go
- FOUND: .planning/phases/02-two-sources-one-trustworthy-stream/02-05-SUMMARY.md
- FOUND: 692573e (Task 1 commit)
- FOUND: ae0fa88 (Task 2 commit)
