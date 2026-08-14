---
phase: 12-filesystem-source
plan: 07
subsystem: api
tags: [go, glob-scope, symlink-containment, provenance, filesystem-plugin, security]

# Dependency graph
requires:
  - phase: 12-filesystem-source (plans 01-06)
    provides: filesystem plugin (walk, item, fetch, scope), kernel open route (fsopen.go), plugin-contract docs, symlink-resolving containment
provides:
  - Fetch classifies through the instance's own scope (newScope(p.extras) + scope.includes), not the bare classify() helper — closes the CR-01 (fresh review) false-404 for include_glob-admitted items
  - Reads and execs target the filepath.EvalSymlinks-resolved path the containment check approved, at both plugin Fetch and the kernel open route, through a single opened handle per fetched file — closes WR-02
  - provenance.source_system published on every filesystem item — closes WR-01 (fresh review)
  - plugins/filesystem/main.go's package doc corrected to state recursion has shipped since 12-03 — closes IN-01
affects: [12-filesystem-source verification, 14-google-drive (inherits the corrected plugin-contract guarantee and provenance/resolved-path conventions)]

# Actuals (#2632)
actuals:
  tokens: 11300
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Fetch and Match/walk share exactly one scope.includes classification call, built fresh per-call from newScope(p.extras) — no second, weaker classification rule may exist anywhere in the plugin"
    - "resolvePath returns TWO paths for one file: the lexical identity path (D-01, what item identity/index/deep-link key on) and the resolved real path (what every read/exec targets) — a caller that validates one path and uses another is exactly the bug this signature widening prevents"
    - "openForFetch opens once and stats through the same handle, so the size-check and the read observe the same file — no second os.Open call can race the first"

key-files:
  created: []
  modified:
    - plugins/filesystem/fetch.go
    - plugins/filesystem/fetch_test.go
    - plugins/filesystem/item.go
    - plugins/filesystem/item_test.go
    - plugins/filesystem/plugin.go
    - plugins/filesystem/main.go
    - kernel/httpapi/fsopen.go
    - kernel/httpapi/fsopen_test.go
    - docs/plugins/filesystem.md
    - docs/api.md
    - docs/plugin-contract.md
    - web/e2e/specs/12-include-glob-metadata-preview.spec.ts

key-decisions:
  - "A malformed operator glob at Fetch time maps to codes.Unavailable, not codes.Internal (the fresh review's own suggested patch) — Match already maps this exact error class to codes.Unavailable, and the kernel maps every non-NotFound Fetch error to the same 502 source_unavailable regardless, so one malformed pattern produces one class of failure at both entry points. Documented inline so a future reader does not revert it to Internal."
  - "resolvePath's widened signature returns empty strings for BOTH results on any error, not just the lexical one — a caller that ignores the error cannot accidentally read a half-validated resolved path either."
  - "The residual final-component TOCTOU window (between filepath.EvalSymlinks returning and the os.Open/exec call) is documented as 'narrows but does not eliminate' in both docs/api.md and docs/plugin-contract.md, using that exact phrase, so no third-party plugin author builds on a stronger guarantee than the code provides. Fully closing it needs openat/O_NOFOLLOW, out of scope at ASVS L1 for this single-user loopback tool (T-12-29, same framing as the already-accepted T-12-23)."

patterns-established:
  - "One shared classification path for both Match's index-membership decision and Fetch's serve decision — a second, independently-maintained rule for the same question is exactly how this gap was introduced in the first place (12-02-PLAN.md's original Fetch dispatch)."

requirements-completed: [SRC-04]

coverage:
  - id: D1
    description: "A file admitted to the index only because include_glob widened past the default extension allowlist previews honestly as metadata-only (never a false NotFound) on both CONTENT_VARIANT_FULL and CONTENT_VARIANT_PREVIEW"
    requirement: "SRC-04"
    verification:
      - kind: unit
        ref: "plugins/filesystem/fetch_test.go#TestFetch_IncludeGlobAdmittedUnknownExtensionIsMetadataOnlyNotNotFound"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/12-include-glob-metadata-preview.spec.ts"
        status: pass
    human_judgment: false
  - id: D2
    description: "A file genuinely outside scope (excluded by exclude_glob, or an unrecognized extension with no include_glob) still answers codes.NotFound — the honesty fix widens nothing"
    requirement: "SRC-04"
    verification:
      - kind: unit
        ref: "plugins/filesystem/fetch_test.go#TestFetch_ExcludedByGlobIsStillNotFound"
        status: pass
      - kind: unit
        ref: "plugins/filesystem/fetch_test.go#TestFetch_UnknownExtensionWithNoIncludeGlobIsStillNotFound"
        status: pass
    human_judgment: false
  - id: D3
    description: "A malformed operator glob at Fetch time names the offending pattern and fails with codes.Unavailable, matching Match's own answer for the identical pattern"
    requirement: "SRC-04"
    verification:
      - kind: unit
        ref: "plugins/filesystem/fetch_test.go#TestFetch_MalformedGlobPatternSurfacesTheOffendingPattern"
        status: pass
    human_judgment: false
  - id: D4
    description: "Every fetched file's bytes are read from the filepath.EvalSymlinks-resolved path the containment check approved, through a single opened handle that also decided the size — not a second, separately-opened path"
    requirement: "SRC-04"
    verification:
      - kind: unit
        ref: "plugins/filesystem/item_test.go#TestResolvePath_ReturnsTheLexicalIdentityPathAndTheResolvedRealPath"
        status: pass
      - kind: unit
        ref: "plugins/filesystem/fetch_test.go#TestFetch_SymlinkedRootStillServesAnInRootFile"
        status: pass
    human_judgment: false
  - id: D5
    description: "The kernel's desktop-open route execs the same resolved path its containment check approved, not the lexical one"
    requirement: "SRC-04"
    verification:
      - kind: unit
        ref: "kernel/httpapi/fsopen_test.go#TestFilesystemOpen_OpenerReceivesTheSymlinkResolvedPath"
        status: pass
    human_judgment: false
  - id: D6
    description: "Every filesystem item's provenance carries all five plugin-populated keys the contract documents, including source_system, reaching a real API client end to end"
    requirement: "SRC-04"
    verification:
      - kind: unit
        ref: "plugins/filesystem/item_test.go#TestMatch_ItemProvenanceCarriesTheFivePluginOwnedKeys"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/12-include-glob-metadata-preview.spec.ts"
        status: pass
    human_judgment: false
  - id: D7
    description: "docs/plugins/filesystem.md, docs/api.md, docs/plugin-contract.md describe the resolved-path discipline and its residual TOCTOU window honestly, matching the shipped code"
    human_judgment: true
    rationale: "Prose accuracy is a documentation-review judgment call, not something a unit test can assert; grep checks in the plan's acceptance criteria confirm the required phrases are present, but agreement with the code's actual behavior needs a human or reviewer read"

duration: ~35min
completed: 2026-08-14
status: complete
---

# Phase 12 Plan 07: Gap Closure — Scope-Aware Fetch, Resolved-Path I/O, Provenance Key Summary

**Fetch now classifies through the same `scope.includes` call Match/walk already use instead of a second, weaker rule — closing the false 404 an `include_glob`-admitted `.zip` produced — plus resolved-path read/exec discipline at both I/O sites, the missing `source_system` provenance key, and a corrected package doc comment.**

## Performance

- **Duration:** ~35 min
- **Started:** 2026-08-14T00:56:00Z
- **Completed:** 2026-08-14T01:12:00Z
- **Tasks:** 3
- **Files modified:** 12

## Accomplishments

- **The gap 12-VERIFICATION.md recorded is closed.** `fetchByKind` now builds one `*scope` from `p.extras` via `newScope` — the identical construction `Match` performs — and classifies through `scope.includes(sourceID)` instead of the package-level `classify()` helper, which had no knowledge of `include_glob`/`exclude_glob` and could not reproduce the "unknown extension admitted by glob → metadata-only" branch. A `.zip` (or any extension outside the default allowlist) admitted only by `include_glob` now previews honestly (`Available:false`, `metadataOnlyReason`) on both `CONTENT_VARIANT_FULL` and `CONTENT_VARIANT_PREVIEW`, reproducing and closing the verifier's exact live repro. A malformed operator glob answers `codes.Unavailable`, naming the offending pattern — mirroring `Match`'s own answer for the identical pattern, deliberately not `codes.Internal` as the fresh review's own suggested patch proposed, since the kernel maps every non-`NotFound` Fetch error to the same `502 source_unavailable` regardless.
- **The honesty fix widens nothing.** A file matching `exclude_glob`, or an unrecognized extension with no `include_glob` declared, still answers `codes.NotFound` — proved by two new regression tests reproducing today's (correct) behavior for those cases.
- **WR-02 closed: reads and execs now target the resolved path the containment check approved, not the lexical one it validated.** `resolvePath`'s signature widened to return both the lexical identity path (D-01, unchanged — what item identity/index/`fileDeepLink` key on) and the resolved real path. `fetchByKind` and the three per-kind rendition helpers now read through the resolved path via a new `openForFetch` helper that opens once and stats through that same handle — closing the gap between "how big is this" and "read these bytes," which previously could observe two different files via separate `os.Stat`/`os.ReadFile`/`os.Open` calls. `kernel/httpapi/fsopen.go`'s `FilesystemOpenHandler` execs the resolved path instead of the lexical `full` it validated, keeping the existing `context.WithoutCancel` detachment (CR-01) untouched.
- **The residual TOCTOU window is documented honestly, not overclaimed.** Both `docs/api.md` and `docs/plugin-contract.md` now state, using the exact phrase, that this discipline "narrows but does not eliminate" the race between resolution and the syscall — the intermediate-component-swap class is closed, but a final-component swap between `EvalSymlinks` returning and `os.Open`/exec remains possible without descriptor-based (`openat`/`O_NOFOLLOW`) traversal, which is out of scope at ASVS L1 for this project.
- **WR-01 closed: every filesystem item now publishes `provenance.source_system`.** `toItem` sets it to `p.root` — the filesystem analog of paperless/silverbullet's `p.baseURL` and signal's `p.configDir` — completing the five plugin-populated provenance keys `docs/plugin-contract.md` documents. Proved by a new unit test and by an e2e assertion that the key reaches a real API client through both the stream and `GET /api/items/{id}`, not merely that the plugin sets it in memory.
- **IN-01 closed.** `plugins/filesystem/main.go`'s package doc comment no longer describes recursion as future work — it names `12-03-PLAN.md` as the plan that shipped it, with `walk.go` as its consumer.

## Task Commits

Each task was committed atomically:

1. **Task 1: Fetch classifies through the instance's own scope, so an include_glob-only item previews honestly instead of 404ing** - `5e59edc` (feat)
2. **Task 2: Read and exec the same resolved path the containment check approved (WR-02)** - `5d8db83` (fix)
3. **Task 3: Publish the missing source_system provenance key (WR-01) and correct the stale package doc (IN-01)** - `7729f50` (fix)

**Plan metadata:** commit follows this SUMMARY

## Files Created/Modified

- `plugins/filesystem/fetch.go` - `fetchByKind` classifies through `newScope(p.extras).includes(sourceID)`; new `openForFetch` supersedes `statForFetch`; the three rendition helpers read through the resolved path via one opened handle
- `plugins/filesystem/fetch_test.go` - Five new tests: include_glob-admitted metadata-only, excluded-still-not-found, unknown-extension-still-not-found, malformed-glob-names-itself, symlinked-root still serves
- `plugins/filesystem/item.go` - `resolvePath` widened to `(full, resolved string, err error)`
- `plugins/filesystem/item_test.go` - Existing `TestResolvePath_*` cases updated for the third result; new `TestResolvePath_ReturnsTheLexicalIdentityPathAndTheResolvedRealPath` and `TestMatch_ItemProvenanceCarriesTheFivePluginOwnedKeys`
- `plugins/filesystem/plugin.go` - `toItem` adds `provenance["source_system"] = p.root`
- `plugins/filesystem/main.go` - Package doc corrected to name 12-03-PLAN.md and state recursion has shipped
- `kernel/httpapi/fsopen.go` - Hands `opener` the resolved path instead of the lexical one
- `kernel/httpapi/fsopen_test.go` - New `TestFilesystemOpen_OpenerReceivesTheSymlinkResolvedPath`; two existing tests' expectations computed through `filepath.EvalSymlinks` via a new `resolvedJoinForTest` helper
- `docs/plugins/filesystem.md` - "Resolution order" section states classification re-runs at Fetch time
- `docs/api.md` - Open-route resolution paragraph names the resolved-path exec and the "narrows but does not eliminate" residual
- `docs/plugin-contract.md` - `file://` convention paragraph extended with the same two points, in the contract's own voice
- `web/e2e/specs/12-include-glob-metadata-preview.spec.ts` - New spec: an `include_glob`-only `.zip` item is listed in the stream, answers an honest unavailable preview (not `item_not_found`) on both the detail and content routes, and its `provenance.source_system` reaches the client

## Decisions Made

See `key-decisions` in the frontmatter above — no decisions beyond what the plan itself specified were required; all three were named choices the plan's action text called for explicitly (the `codes.Unavailable` departure from the review's own suggested `codes.Internal`, the empty-string-on-error discipline for both `resolvePath` results, and the exact "narrows but does not eliminate" phrasing).

## Deviations from Plan

None - plan executed exactly as written. The one adjustment worth noting is not a deviation from the plan's intent but a sequencing correction within Task 1: the e2e spec's `provenance.source_system` assertion (which the plan assigns to Task 3) was initially added while writing Task 1's spec and had to be removed until Task 3 actually populated that key — caught immediately by running the spec against Task 1's code alone, before committing, so no incorrect assertion ever reached a commit.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All three findings the freshly re-run `12-REVIEW.md` raised (the new CR-01 classification gap, WR-02 resolved-path discipline, WR-01 missing provenance key) plus the cosmetic IN-01 stale doc comment are closed, each with regression coverage that did not exist before this plan.
- `REQUIREMENTS.md`'s SRC-04 mark and `ROADMAP.md`'s Phase 12 status are intentionally left untouched, per this plan's non-goals — the verifier owns those marks and should re-run against this closure.
- `make test-portable`, `go test ./internal/audit/ -count=1`, and the three named e2e specs (`12-include-glob-metadata-preview`, `12-filesystem-tracer`, `12-filesystem-recursion`) all pass against a freshly built kernel and plugin binary.

## Self-Check: PASSED
