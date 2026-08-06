---
phase: 06-ui-scalable-source-surface
fixed_at: 2026-08-06T23:37:00Z
review_path: .planning/phases/06-ui-scalable-source-surface/06-REVIEW.md
iteration: 1
findings_in_scope: 4
fixed: 4
skipped: 0
status: all_fixed
---

# Phase 06: Code Review Fix Report

**Fixed at:** 2026-08-06T23:37:00Z
**Source review:** .planning/phases/06-ui-scalable-source-surface/06-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 4 (fix_scope: critical_warning — CR-01, WR-01, WR-02, WR-03; IN-01 excluded)
- Fixed: 4
- Skipped: 0

**Verification environment:** All fixes were applied, built, and tested inside an isolated
git worktree (`workflow.use_worktrees` was unset, defaulting to `true`). Go: `go build ./...`,
`go vet ./...`, and `go test ./...` (all packages, not just the touched ones) all passed at the
end of the run. Frontend: `vitest run` (full suite, 11 files / 188 tests) and `svelte-check`
(761 files, 0 errors) both passed against the touched files after `web/node_modules` was
symlinked in from the main checkout (the worktree has no `node_modules` of its own — the
symlink was removed before handoff and is not part of any commit). These results are
reproducible from the main checkout at `main` after the worktree's commits were
fast-forwarded onto it.

## Fixed Issues

### CR-01: Chip-row overflow math ignores the real layout's inter-chip gap, so chips can silently clip past the visible row

**Files modified:** `web/src/lib/format.ts`, `web/src/lib/components/WebspaceHeader.svelte`, `web/src/lib/components/sources.test.ts`
**Commit:** `310dc70`
**Applied fix:** Added a required `gapWidth` parameter to `visibleChipCount` and charged it for
every adjacent pair of visible items — `N-1` between-chip gaps plus one trailing gap before the
reserved trailing group in the "everything fits" path, and two additional gaps (before/after the
overflow trigger) in the overflow-accumulation path — matching the review's suggested
implementation. `WebspaceHeader.svelte` now defines a `CHIP_ROW_GAP_PX = 8` constant (documented
as tracking the row's real `gap-2` Tailwind class) and passes it at the `visibleChipCount` call
site. Updated all existing `sources.test.ts` calls to pass an explicit `gapWidth` (0 to preserve
prior semantics where the test wasn't about gaps) and added three new regression tests: a
non-zero gap reducing the inline count versus the same layout with no gap, and an exact-boundary
test asserting `N-1` between-chip gaps plus one trailing gap are charged. All 57 tests in
`sources.test.ts` (and 188 across the full frontend suite) pass.

### WR-01: `?hl=` highlighting is applied to `/thumbnail` too, contradicting docs/api.md's "content route only" claim

**Files modified:** `kernel/httpapi/item.go`, `kernel/httpapi/item_test.go`
**Commit:** `342566c`
**Applied fix:** Gated the `hl` query-parameter read inside the shared `renditionHandler` on
`variant == toposv1.ContentVariant_CONTENT_VARIANT_PREVIEW` (the exact form suggested in the
review), so `ItemThumbnailHandler`'s call site (`CONTENT_VARIANT_THUMBNAIL`) never derives
highlight terms regardless of what `?hl=` carries. Added
`TestItemThumbnailHandler_HlParameterNeverHighlights`, a new regression test asserting a
`text/html` thumbnail rendition with a matching `?hl=` term is served with no `<mark>` element
while its own visible text still survives sanitization. `go build ./...`, `go vet
./kernel/httpapi/...`, and `go test ./kernel/httpapi/...` all pass.

### WR-02: Individual highlight terms have no upper length bound, despite the docstring's "bounded-work controls" claim

**Files modified:** `kernel/httpapi/rendition.go`, `kernel/httpapi/rendition_test.go`, `web/src/lib/format.ts`, `web/src/lib/components/highlight.test.ts`
**Commit:** `f5dbfde`
**Applied fix:** Added a `highlightTermMaxRunes = 64` bound to the kernel's `highlightTerms` (Go)
and the mirrored `HIGHLIGHT_TERM_MAX_LENGTH = 64` constant to the client's `highlightTerms` (TS),
dropping any term whose length exceeds the bound alongside the existing `<2` drop — the "explicit
cap" option from the review's Fix section, chosen over narrowing the docstring, so the "bounded-
work controls" claim becomes accurate rather than just re-worded. Updated both docstrings to
describe the new bound. Added matching regression test cases on both sides (`TestHighlightTerms_Derivation`
in Go, two new `it()` blocks in `highlight.test.ts`) asserting a term exactly at 64 characters
survives and one at 65 is dropped. All Go and frontend tests pass.

### WR-03: A source already syncing when the page loads never starts client-side polling, so its spinner can go stale indefinitely

**Files modified:** `web/src/routes/w/[webspace]/+page.svelte`
**Commit:** `cc2ded6`
**Applied fix:** `loadSources()` now calls `ensurePolling()` whenever its freshly-loaded response
contains any `syncing: true` entry — the exact change proposed in the review's Fix section — so
an already-in-flight sync started by the background scheduler, the `topos sync` CLI, or another
browser tab is picked up on initial mount or webspace-route-param change, not only when this tab
itself initiates a refresh via `handleRefreshSource`/`handleRefreshAll`. `ensurePolling` is a
hoisted function declaration defined later in the same script block, so the forward reference is
valid. No existing test infrastructure covers this route file (`web/src/routes/` has no `.test.ts`
files in this project); verified via `svelte-check` (761 files, 0 errors) and a manual read-through
of the diff, which is a minimal 9-line addition with no control-flow changes elsewhere.

## Skipped Issues

None — all in-scope findings were fixed.

---

_Fixed: 2026-08-06T23:37:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
