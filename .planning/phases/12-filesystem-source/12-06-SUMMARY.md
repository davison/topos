---
phase: 12-filesystem-source
plan: 06
subsystem: api
tags: [go, symlink-containment, context-lifecycle, filesystem-plugin, security]

# Dependency graph
requires:
  - phase: 12-filesystem-source (plans 01-05)
    provides: filesystem plugin (walk, item, fetch), kernel open route (fsopen.go), plugin-contract docs
provides:
  - Symlink-resolving containment at both load-bearing sites (plugin Fetch, kernel open route) — closes CR-02
  - xdg-open child process detached from the HTTP request's own context — closes CR-01
  - Symlinked/bind-mounted configured root no longer silently drops in-tree symlinked files — closes WR-01
  - docs/plugin-contract.md, docs/api.md, docs/plugins/filesystem.md brought back into agreement with shipped code
affects: [12-filesystem-source verification, 14-google-drive (inherits the plugin-contract guarantee)]

# Actuals (#2632)
actuals:
  tokens: 11627
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "resolveRoot hand-kept twin (plugins/filesystem/item.go and kernel/httpapi/fsopen.go): resolve the configured root once with filepath.EvalSymlinks, fall back to filepath.Clean on failure, compare resolved paths against the resolved root — never the lexical one"
    - "errors.Is(err, fs.ErrNotExist) on a %w-wrapped filepath.EvalSymlinks error distinguishes a vanished file (honest NotFound/item_not_found) from a genuine containment escape (InvalidArgument/invalid_path)"
    - "context.WithoutCancel at the one call site that starts a detached child process, paired with a production Opener whose context parameter is structurally the blank identifier"
    - "go/ast structural test (parser.ParseFile + ast.Inspect) scoped to a single named source file, asserting on AST shape rather than text, for a property a behavioural test cannot reach without execing a real binary"

key-files:
  created: []
  modified:
    - plugins/filesystem/item.go
    - plugins/filesystem/item_test.go
    - plugins/filesystem/fetch.go
    - plugins/filesystem/fetch_test.go
    - plugins/filesystem/walk.go
    - plugins/filesystem/walk_test.go
    - kernel/httpapi/fsopen.go
    - kernel/httpapi/fsopen_test.go
    - docs/plugin-contract.md
    - docs/api.md
    - docs/plugins/filesystem.md

key-decisions:
  - "filepath.WalkDir uses Lstat semantics on its OWN root argument, not just on entries it discovers — a symlinked configured root was never descended into at all (WalkDir sees a non-directory at the top and stops), not merely under-inclusive on its symlinked children as WR-01's original framing suggested. walk() now resolves the root once and walks FROM the resolved root, with relPathSourceID computed against that same resolved base, so the fix actually traverses instead of only fixing a containment comparison that was unreachable code for a symlinked root"
  - "resolveRoot is duplicated by hand in both plugins/filesystem/item.go and kernel/httpapi/fsopen.go (separate Go modules, plugin cannot be imported by kernel) — identical shape, cross-referencing doc comments, matching the existing resolvePath/fsopen.go lexical-guard duplication discipline already established in the codebase"
  - "resolvePath and the kernel's inline check return the LEXICAL joined path on success, never the resolved one — callers, the index, fileDeepLink and the opener all key on the lexical path under the configured root; only the containment COMPARISON uses resolved values"

patterns-established:
  - "Fail-closed symlink resolution at every site that touches bytes or execs a program (Fetch, Open, and now Match/walk), decided against the EvalSymlinks-resolved target compared to the EvalSymlinks-resolved configured root — never the lexical path alone"

requirements-completed: [SRC-04]

coverage:
  - id: D1
    description: "A file swapped for an outward symlink after indexing is refused by the plugin's Fetch path (byte-serving site) with no byte of the symlink target ever served"
    requirement: "SRC-04"
    verification:
      - kind: unit
        ref: "plugins/filesystem/fetch_test.go#TestFetch_SymlinkSwappedAfterIndexingIsRefusedBeforeAnyBytesAreServed"
        status: pass
    human_judgment: false
  - id: D2
    description: "The same post-index symlink swap is refused by POST /api/items/{id}/open with invalid_path (400), and the opener is never called"
    requirement: "SRC-04"
    verification:
      - kind: unit
        ref: "kernel/httpapi/fsopen_test.go#TestFilesystemOpen_SymlinkSwappedAfterIndexingAnswersInvalidPathAndNeverOpens"
        status: pass
    human_judgment: false
  - id: D3
    description: "A file that vanished between sync and request is reported honestly as NotFound/item_not_found at both sites, not as a containment violation"
    requirement: "SRC-04"
    verification:
      - kind: unit
        ref: "plugins/filesystem/fetch_test.go#TestFetch_MissingFileIsNotFoundGRPCError"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/fsopen_test.go#TestFilesystemOpen_VanishedFileAnswersItemNotFoundAndNeverOpens"
        status: pass
    human_judgment: false
  - id: D4
    description: "The context handed to the opener is detached from the HTTP request's own context, and the production opener structurally cannot bind its child to a caller context"
    requirement: "SRC-04"
    verification:
      - kind: unit
        ref: "kernel/httpapi/fsopen_test.go#TestFilesystemOpen_OpenerContextIsDetachedFromTheRequestContext"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/fsopen_test.go#TestNewXDGOpener_ChildIsNotBoundToACallerContext"
        status: pass
    human_judgment: false
  - id: D5
    description: "A symlinked or bind-mounted configured root still contributes its in-tree symlinked files instead of silently dropping every one of them"
    requirement: "SRC-04"
    verification:
      - kind: unit
        ref: "plugins/filesystem/walk_test.go#TestWalk_InTreeSymlinkUnderASymlinkedRootIsStillIncluded"
        status: pass
    human_judgment: false
  - id: D6
    description: "docs/plugin-contract.md, docs/api.md and docs/plugins/filesystem.md describe the symlink-resolving containment discipline the shipped code actually enforces"
    human_judgment: true
    rationale: "Prose accuracy is a documentation-review judgment call, not something a unit test can assert; grep checks in the plan's acceptance criteria confirm the mechanism is named, but agreement with the code's actual behavior needs a human or reviewer read"

duration: ~45min
completed: 2026-08-14
status: complete
---

# Phase 12 Plan 06: Symlink Containment and Opener Lifetime Gap Closure Summary

**Closed CR-01 (xdg-open child killed by request-context cancellation), CR-02 (lexical-only symlink containment at Fetch/Open), and WR-01 (symlinked root silently dropping in-tree files) from 12-VERIFICATION.md, with eight new regression tests and three docs corrected to match the shipped code.**

## Performance

- **Duration:** ~45 min
- **Started:** 2026-08-13T23:28:00Z
- **Completed:** 2026-08-14T00:13:18Z
- **Tasks:** 3
- **Files modified:** 11

## Accomplishments

- **CR-02 closed at both load-bearing sites.** `plugins/filesystem/item.go`'s `resolvePath` (reached from `fetch.go`'s `fetchByKind`) and `kernel/httpapi/fsopen.go`'s inline containment check now re-validate containment against the `filepath.EvalSymlinks`-resolved joined path and resolved configured root, after the existing lexical `..`-segment check. A file indexed legitimately and later swapped on disk for a symlink pointing outside the root is refused before any byte is read or any process is exec'd — proved by `TestFetch_SymlinkSwappedAfterIndexingIsRefusedBeforeAnyBytesAreServed` (which asserts the secret target's bytes appear nowhere in the error) and `TestFilesystemOpen_SymlinkSwappedAfterIndexingAnswersInvalidPathAndNeverOpens`.
- **A vanished file is now reported honestly.** `errors.Is(err, fs.ErrNotExist)` on the wrapped `EvalSymlinks` error distinguishes a file that simply no longer exists (answers `codes.NotFound` / `item_not_found`) from a genuine containment escape (`codes.InvalidArgument` / `invalid_path`) — no new HTTP error code introduced.
- **CR-01 closed.** `newXDGOpener`'s returned closure now takes the blank identifier for its context parameter and builds the child with the plain `exec.Command` form — structurally impossible to bind to a caller's context. `FilesystemOpenHandler` hands the opener `context.WithoutCancel(ctx)`, so cancelling the request's own context after the handler returns (which `net/http` does essentially synchronously) leaves the opener's context uncancelled. Proved behaviourally (`TestFilesystemOpen_OpenerContextIsDetachedFromTheRequestContext`) and structurally via a `go/ast` scan of `fsopen.go` alone (`TestNewXDGOpener_ChildIsNotBoundToACallerContext`).
- **WR-01 closed, with a deeper root cause found and fixed.** The plan's literal instruction (fix the in-tree symlink comparison to use a resolved root) turned out to be necessary but not sufficient: `filepath.WalkDir` applies `Lstat` semantics to its own root argument, so a symlinked configured root was never descended into at all — not merely under-inclusive on its symlinked children. `walk()` now resolves the root once before entering `WalkDir` and walks *from* the resolved root, computing `relPathSourceID` against that same resolved base throughout, so a symlinked/bind-mounted root (the `~/Documents` → `~/dotfiles/Documents` pattern) is actually traversed and its in-tree symlinked files are included.
- **Docs brought back into agreement with the code.** `docs/plugin-contract.md`'s published re-resolution guarantee, `docs/api.md`'s open-route section and error-code table, and `docs/plugins/filesystem.md`'s symlink policy section all now name the symlink-resolution mechanism and describe the fail-closed rule the shipped code enforces.

## Task Commits

Each task was committed atomically:

1. **Task 1: Resolve symlinks before containment at BOTH load-bearing sites** - `0367cc9` (fix)
2. **Task 2: Detach the opened child process from the HTTP request's lifetime** - `2d9df52` (fix)
3. **Task 3: Resolve the walk's own root (WR-01), and bring the three published docs back into agreement with the code** - `906e0f0` (fix)

**Plan metadata:** commit follows this SUMMARY

## Files Created/Modified

- `plugins/filesystem/item.go` - Added `resolveRoot`; `resolvePath` now re-validates containment against the resolved joined path and resolved root, wrapping resolution errors with `%w`
- `plugins/filesystem/item_test.go` - New symlink-swap and symlinked-root regression tests; fixed `TestResolvePath_JoinsRootAndSourceID`'s fixture to exist on disk
- `plugins/filesystem/fetch.go` - `fetchByKind` maps a vanished-file resolution error to `codes.NotFound`, every other resolution failure to `codes.InvalidArgument`
- `plugins/filesystem/fetch_test.go` - New `TestFetch_SymlinkSwappedAfterIndexingIsRefusedBeforeAnyBytesAreServed`
- `plugins/filesystem/walk.go` - Resolves the configured root once before walking; walks from and computes relative source IDs against the resolved root; in-tree symlink comparison uses resolved root
- `plugins/filesystem/walk_test.go` - New `TestWalk_InTreeSymlinkUnderASymlinkedRootIsStillIncluded`
- `kernel/httpapi/fsopen.go` - Added `resolveRoot` twin; inline containment check re-validates against resolved values; `newXDGOpener`'s closure takes a blank context parameter and uses plain `exec.Command`; handler hands the opener `context.WithoutCancel(ctx)`
- `kernel/httpapi/fsopen_test.go` - `recordingOpener` gains `calledCtx`; fixed three existing tests' fixtures to exist on disk; four new tests (symlink swap, vanished file, context detachment, AST structural check)
- `docs/plugin-contract.md` - Reworded the `file://` convention's re-resolution guarantee to name `EvalSymlinks` and the fail-closed rule; added a sentence pointing plugin authors at `resolvePath` as the `Fetch`-side reference
- `docs/api.md` - Extended the open route's resolution paragraph and the `item_not_found`/`invalid_path` error-table rows
- `docs/plugins/filesystem.md` - Extended "Symlink and dot-file policy" with the Fetch/Open re-check and symlinked-root support

## Decisions Made

- **WalkDir's own Lstat-on-root behavior required walking from the resolved root, not just comparing against it.** The plan's Task 3 read_first/action text described the fix purely as "change the in-tree symlink containment comparison to use a resolved root." Empirical verification (a small standalone `filepath.WalkDir` probe against a symlinked directory) showed this alone does not work: `WalkDir` calls `os.Lstat` on its own root argument, and if that root is a symlink to a directory, the walk callback fires once for the root itself (reporting non-directory) and never descends — the containment-comparison fix would have been dead code for the very case WR-01 describes. The actual fix resolves the root once and passes the RESOLVED path to `filepath.WalkDir`, with `relPathSourceID` computed against that same resolved base so `source_id` values stay correct forward-slash relative paths (Rule 1 auto-fix: a bug in the intended fix's reachability, not a scope change).
- **`resolveRoot` is duplicated by hand in both modules** rather than shared, matching the existing `resolvePath`/`fsopen.go` lexical-guard precedent already in the codebase (the plugin and kernel are separate Go modules).

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed three existing kernel/httpapi tests and one existing plugin test whose fixtures were never materialized on disk**
- **Found during:** Task 1
- **Issue:** `TestFilesystemOpen_HappyPathOpensTheJoinedAbsolutePath`, `TestFilesystemOpen_OpenerErrorAnswersOpenFailed`, `TestFilesystemOpen_TildeInConfiguredPathIsExpandedBeforeTheJoin` (kernel) and `TestResolvePath_JoinsRootAndSourceID` (plugin) all called the resolution path against a `source_id`/path that had no corresponding file on disk. Once fail-closed `filepath.EvalSymlinks` resolution reaches these tests, `EvalSymlinks` fails for a non-existent path and the tests fail — not a regression in behavior, but a fixture gap the plan's own Task 1 step 5 explicitly called out for the three kernel tests (the plugin-side test was an oversight the plan didn't separately name).
- **Fix:** Wrote a real fixture file (or, for the tilde test, a real `os.MkdirTemp`-created directory under the user's home with `t.Cleanup` registered for removal) before each test's assertions run. Every existing assertion was kept exactly as written — this is fixture correction, never assertion loosening.
- **Files modified:** `kernel/httpapi/fsopen_test.go`, `plugins/filesystem/item_test.go`
- **Verification:** All four tests pass; `go test ./... -count=1` clean in both modules.
- **Committed in:** `0367cc9` (Task 1 commit)

**2. [Rule 1 - Bug] `walk()` needed to walk from the resolved root, not merely compare against it**
- **Found during:** Task 3
- **Issue:** As described above under "Decisions Made" — `filepath.WalkDir`'s own Lstat semantics on its root argument meant the plan's literally-described fix (change the containment comparison only) would leave a symlinked configured root completely untraversed, not merely under-inclusive.
- **Fix:** `walk()` resolves the configured root once (`resolveRoot`) before calling `filepath.WalkDir`, walks from that resolved path, and computes `relPathSourceID` against the same resolved base throughout the callback.
- **Files modified:** `plugins/filesystem/walk.go`
- **Verification:** `TestWalk_InTreeSymlinkUnderASymlinkedRootIsStillIncluded` passes; all 15 pre-existing `TestWalk_*` cases still pass, including `TestWalk_SymlinkToFileOutsideRootIsExcluded`.
- **Committed in:** `906e0f0` (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (both Rule 1 - bug fixes necessary to make the plan's stated fail-closed behavior actually reachable/correct)
**Impact on plan:** No scope creep — both deviations were required for the plan's own stated `<done>` and `<acceptance_criteria>` to hold, and both are documented in-line in the corresponding commits' code comments.

## Issues Encountered

None beyond the two deviations documented above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All three gaps from `12-VERIFICATION.md` (G-12-1, G-12-2) plus the carried-forward WR-01 warning are closed with regression tests that structurally could not exist before this plan (symlink-swap fixtures, request-context-cancellation proof, AST-scoped structural proof).
- `make test-portable` (all workspace modules), `go test ./internal/audit/`, `go build ./...` and `go vet ./kernel/httpapi/` all pass clean.
- `docs/plugin-contract.md:824-858`'s published guarantee is now true of the shipped code — Phase 14's out-of-repo Google Drive plugin author can build against it safely.
- No e2e spec was added or changed, per the plan's explicit scope note: both closed findings are Go process-lifecycle and Go path-resolution bugs, and `docs/testing.md` already scopes real `xdg-open` desktop-handler behavior out of the hermetic browser harness by design.
- REQUIREMENTS.md, STATE.md and ROADMAP.md are intentionally NOT updated by this plan (worktree/wave execution) — the orchestrator owns those writes after all wave agents complete.

---
*Phase: 12-filesystem-source*
*Completed: 2026-08-14*
