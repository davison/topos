---
phase: 12-filesystem-source
plan: 08
subsystem: source-plugins
tags: [filesystem, match-vocabulary, D-05, D-04, docs, e2e]

# Dependency graph
requires:
  - phase: 12-filesystem-source
    provides: "folderLabels (12-03), the filesystem plugin's Match/labelMatchesAny exact-match discipline (12-01..12-07), the UAT diagnosis in .planning/debug/filesystem-items-missing-from-stream.md"
provides:
  - "folderLabels emits the configured root's own base name for every file at every depth (top-level and nested), so one webspace folders value expresses \"everything from this instance\" for a recursive source"
  - "dedupeLabels — order-preserving, case-insensitive, first-occurrence-wins dedupe for the label set"
  - "docs/plugins/filesystem.md and config.example.toml both state that match values are exact literals, never glob patterns, with the worked \"everything from this instance\" recipe"
  - "web/e2e/specs/12-filesystem-root-label-match.spec.ts — the user's exact reported failure (folders = ['*']) and its fix, both pinned end to end against a real kernel and a real topos-plugin-filesystem subprocess"
affects: ["12-09-filesystem-source (zero-match diagnostic — the second missing: item of G-12-1/G-12-3)", "12-10-filesystem-source (web/src/ advisory surface)"]

# Actuals (#2632)
actuals:
  tokens: 5237
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Label-widening-at-the-producer: a match-vocabulary gap is closed by widening what a plugin REPORTS (folderLabels), never by teaching the comparator (labelMatchesAny) new semantics — keeps the exact-match discipline (D-04) a true cross-plugin invariant"
    - "Order-preserving, case-insensitive, first-occurrence-wins dedupe (dedupeLabels) for a label/value set that may legitimately collide"

key-files:
  created:
    - web/e2e/specs/12-filesystem-root-label-match.spec.ts
  modified:
    - plugins/filesystem/item.go
    - plugins/filesystem/item_test.go
    - docs/plugins/filesystem.md
    - config.example.toml

key-decisions:
  - "The root base name is prepended, not appended — folderLabels' contract is now root-base-name-first, then per-segment labels, then the cumulative relative path, so the label a match block most likely wants (the whole-instance one) is always first"
  - "dedupeLabels lives beside folderLabels as a package-local helper rather than a generic slice utility — the plan explicitly scoped this to the label-set collision case, not a general dedupe library"
  - "The e2e spec asserts entirely at the kernel API level (fetch against kernel.baseURL), never the DOM — the defect this plan closes is in match-value semantics, not rendering, matching 12-filesystem-tracer.spec.ts's own idiom"

patterns-established:
  - "A gap-closure plan citing its own gap ids (G-12-1, G-12-3) and plan id (12-08-PLAN.md) inside the doc comment it rewrites, so the comment carries its own provenance"

requirements-completed: [SRC-04]

coverage:
  - id: D1
    description: "A webspace's folders match block naming a filesystem source's configured root base name matches every file that instance contributes, at every depth — top-level and nested alike — closing the first missing: item of G-12-1/G-12-3"
    requirement: "SRC-04"
    verification:
      - kind: unit
        ref: "plugins/filesystem/item_test.go#TestFolderLabels_NestedFileAlsoCarriesTheRootBaseName"
        status: pass
      - kind: unit
        ref: "plugins/filesystem/item_test.go#TestFolderLabels_RootBaseNameEqualToASubfolderSegmentIsNotDuplicated"
        status: pass
      - kind: unit
        ref: "plugins/filesystem/item_test.go#TestFolderLabels_NoLabelNamesADirectoryAboveTheConfiguredRoot"
        status: pass
      - kind: unit
        ref: "plugins/filesystem/item_test.go#TestFolderLabels_SubdirectoryFileIsContainingDirectoryBaseName (extended)"
        status: pass
      - kind: unit
        ref: "plugins/filesystem/item_test.go#TestFolderLabels_TopLevelFileIsRootBaseName (unmodified regression guard)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/12-filesystem-root-label-match.spec.ts#the root base name matches every file at every depth; the glob-shaped value matches nothing"
        status: pass
    human_judgment: false
  - id: D2
    description: "docs/plugins/filesystem.md and config.example.toml both state match values are compared as exact literals, never glob patterns, and give the worked everything-from-this-instance recipe on the same page/block that documents this plugin's real doublestar glob keys"
    requirement: "SRC-04"
    verification:
      - kind: other
        ref: "bash scripts/check-doc-links.sh"
        status: pass
      - kind: other
        ref: "grep -c 'never as glob patterns'/'everything from this instance' in docs/plugins/filesystem.md and config.example.toml (all >=1)"
        status: pass
      - kind: integration
        ref: "go test ./kernel/config/ -count=1 (config.example.toml stays parseable; every added line is a comment)"
        status: pass
    human_judgment: false
  - id: D3
    description: "The user's exact reported failure (folders = ['*'] over a recursive filesystem source) and its fix are both pinned as one committed e2e gate against a real kernel and a real topos-plugin-filesystem binary, including the healthy-sync-with-empty-stream coexistence"
    requirement: "SRC-04"
    verification:
      - kind: e2e
        ref: "web/e2e/specs/12-filesystem-root-label-match.spec.ts"
        status: pass
      - kind: e2e
        ref: "make e2e (full suite, 116 specs including the four other filesystem specs and this new one)"
        status: pass
    human_judgment: false

duration: ~35min
completed: 2026-08-14
status: complete
---

# Phase 12 Plan 08: Root-label match-value gap closure Summary

**`folderLabels` now prepends the configured root's own base name to every file at every depth (deduped via a new `dedupeLabels` helper), making "everything from this instance" a typeable `folders` value for the first time; both operator-facing docs now say match values are exact literals, never globs; and one new e2e spec pins the user's exact `folders = ['*']` failure alongside its fix against a real kernel.**

## Performance

- **Duration:** ~35 min
- **Completed:** 2026-08-14
- **Tasks:** 3
- **Files modified:** 5 (1 created, 4 modified)

## Accomplishments
- Closed the first `missing:` item of G-12-1/G-12-3: a recursive filesystem source's operator can now type one `folders` value (the configured folder's own base name) and match every file that source contributes, at any depth — previously structurally inexpressible, since nested files never carried the root base name.
- Added `dedupeLabels`, an order-preserving, case-insensitive, first-occurrence-wins dedupe, so a subfolder sharing its root's own name reports one label rather than two.
- Rewrote the "Folder-vocabulary match values (`folders`)" section of `docs/plugins/filesystem.md` and added matching commentary to `config.example.toml` in both places an operator reads it, stating plainly that match values are compared as exact literals — **never as glob patterns** — on the same page/block that documents this plugin's real `include_glob`/`exclude_glob` doublestar keys.
- Committed `web/e2e/specs/12-filesystem-root-label-match.spec.ts`, reproducing the user's real config (`folders = ['*']`) as a negative case alongside the root-base-name value as its fix, plus the healthy-sync-with-empty-stream coexistence assertion that made the original failure hard to diagnose.

## Task Commits

Each task was committed atomically:

1. **Task 1: One value covers a whole instance — the root base name labels every file, at every depth** - `6dae168` (feat)
2. **Task 2: Say it on the page that documents globs — match values are exact, and here is how to match everything** - `5c6117e` (docs)
3. **Task 3: The user's mistake and its fix, both proven against a real kernel and a real plugin binary** - `de61379` (test)

**Plan metadata:** SUMMARY.md commit (this commit) will follow, per worktree-mode execution.

## Files Created/Modified
- `plugins/filesystem/item.go` - `folderLabels` rewritten to prepend the configured root's own base name to every file's label set, at every depth; new `dedupeLabels` package-local helper
- `plugins/filesystem/item_test.go` - extended `TestFolderLabels_SubdirectoryFileIsContainingDirectoryBaseName` to assert the full exact ordered set; added `TestFolderLabels_NestedFileAlsoCarriesTheRootBaseName`, `TestFolderLabels_RootBaseNameEqualToASubfolderSegmentIsNotDuplicated`, `TestFolderLabels_NoLabelNamesADirectoryAboveTheConfiguredRoot`
- `docs/plugins/filesystem.md` - "Folder-vocabulary match values (`folders`)" section rewritten: what every file reports, the everything-from-this-instance recipe, and the never-as-glob-patterns statement linked to `docs/plugin-contract.md`'s own match rule
- `config.example.toml` - matching commentary added to `[sources.filesystem]`'s block and the generic webspace-matching commentary's closing exact-match paragraph; every added line is a comment, no key/value changed
- `web/e2e/specs/12-filesystem-root-label-match.spec.ts` (new) - two webspaces, one matching the root base name (positive case), one matching a bare asterisk byte-identical to the user's real config (negative case), against a real kernel and plugin subprocess

## Decisions Made
- The root base name is prepended FIRST in the label set (root base name, then per-segment labels, then the cumulative relative path) rather than appended, so the whole-instance label is always the first and most discoverable value.
- `dedupeLabels` stays package-local rather than becoming a generic slice utility — scoped exactly to the label-set collision case the plan named.
- The new e2e spec's assertions are entirely API-level (`fetch` against `kernel.baseURL`), matching `12-filesystem-tracer.spec.ts`'s own idiom, since the defect being pinned is in match-value semantics rather than DOM rendering.

## Deviations from Plan

None - plan executed exactly as written. All three tasks, their `<behavior>`/`<action>` specifications, and every listed acceptance criterion were implemented and verified without needing an auto-fix, an architectural change, or a scope adjustment.

## Issues Encountered
None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness
- 12-09-PLAN.md (the zero-match diagnostic, the SECOND `missing:` item of G-12-1/G-12-3) runs in this same wave against entirely different files (`docs/api.md` and kernel-side diagnostics) — this plan's non-goals explicitly deferred that surface and asserted nothing about a `last_notice`/advisory field, so no conflict.
- 12-10-PLAN.md (the `web/src/` advisory UI surface) runs next wave, also untouched by this plan.
- Full verification suite green: `plugins/filesystem` package tests (including the three new + one extended `TestFolderLabels_*` cases), `go vet`, `make test-portable` (all Go modules), `go test ./internal/audit/`, `bash scripts/check-doc-links.sh`, and the full `make e2e` (116/116 specs, including all four pre-existing filesystem specs plus this plan's new one) all pass with no regressions.

---
*Phase: 12-filesystem-source*
*Completed: 2026-08-14*
