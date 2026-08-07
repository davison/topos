---
phase: 06-ui-scalable-source-surface
fixed_at: 2026-08-07T09:33:00Z
review_path: .planning/phases/06-ui-scalable-source-surface/06-REVIEW.md
iteration: 1
findings_in_scope: 6
fixed: 6
skipped: 0
status: all_fixed
---

# Phase 06: Code Review Fix Report

**Fixed at:** 2026-08-07T09:33:00Z
**Source review:** .planning/phases/06-ui-scalable-source-surface/06-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope: 6 (fix_scope: critical_warning — CR-01, CR-02, WR-01, WR-02, WR-03, WR-04; IN-01 and IN-02 excluded)
- Fixed: 6
- Skipped: 0

This is a fresh, full re-review round. The current `06-REVIEW.md` is a fresh,
full re-review that superseded a prior mid-phase review (whose findings were
independently confirmed already fixed). This report supersedes the earlier
`06-REVIEW-FIX.md` on disk (which addressed that prior round's four findings)
— its content is not merged or carried forward.

**Verification environment:** All fixes were applied and verified in an
isolated git worktree (`workflow.use_worktrees` was unset in
`.planning/config.json`, defaulting to `true`), then fast-forwarded onto
`main`. `npm run check` (svelte-check, 0 errors, 1 pre-existing unrelated
warning in `SearchBox.svelte`) and `npx vitest run` (14 files / 249 tests,
all passed) were run after every commit from inside that worktree, with
`web/node_modules` symlinked from the main checkout to avoid a slow
reinstall — no worktree-local dependency state affects these results, so
they are reproducible from `web/` in the main checkout post-merge. No Go
code was changed (WR-01 was fixed client-side only, per the review's
recommended minimal fix), so no Go build/test pass was required.

## Fixed Issues

### CR-01: DetailPane content fetch has no stale-response guard — rapid item switching can show the wrong item's content

**Files modified:** `web/src/lib/components/DetailPane.svelte`
**Commit:** 73da5cd
**Applied fix:** Added a `contentRequestSeq` counter (mirroring the existing
`searchRequestSeq` pattern in `+page.svelte`), incremented once per
`loadContent(id)` call and captured into a local `seq` before the `await`.
Both the success and error branches now check `seq !== contentRequestSeq`
and bail out before writing `content`/`fetchErrorCode`, and `loadingContent`
is only cleared in `finally` when `seq` is still current — so a slower,
superseded fetch can no longer overwrite what's on screen for a newer
selection.

### CR-02: `+page.svelte`'s stream fetch has no stale-webspace guard — rapid webspace navigation can show the wrong webspace's stream

**Files modified:** `web/src/routes/w/[webspace]/+page.svelte`
**Commit:** 22d681f
**Applied fix:** Added a `navGeneration` counter, bumped once per run of the
webspace-keyed `$effect` and threaded into `load(gen)` as an explicit
parameter. `load` now checks `gen !== navGeneration` after its `await` in
both the success and error branches before writing `response`/`loadState`.
Updated all three call sites (`handleRefreshSource`, `handleRefreshAll`, the
`StreamList` `onretry` callback) to pass the current `navGeneration` so a
same-webspace retry/refresh still resolves correctly. Committed together
with WR-02 and WR-03 since all three touch the same `$effect`/`navGeneration`
mechanism in the same file.

### WR-01: Client and kernel `highlightTerms` diverge on multi-byte (astral) characters, contradicting the documented "must never diverge" invariant

**Files modified:** `web/src/lib/format.ts`
**Commit:** 227b41d
**Applied fix:** Changed the client's `highlightTerms` to count Unicode code
points via `[...term].length` instead of `term.length` (UTF-16 code units)
for both the `<2` and `>64` bounds, matching the kernel's
`utf8.RuneCountInString` in `kernel/httpapi/rendition.go`. No kernel-side
change was needed — the review's recommended minimal fix (client-side only)
was applied as-is.

### WR-02: `+page.svelte`'s `handleSearch` stale-response guard doesn't cover cross-webspace navigation

**Files modified:** `web/src/routes/w/[webspace]/+page.svelte`
**Commit:** 22d681f
**Applied fix:** Folded into the same `navGeneration` mechanism added for
CR-02: `handleSearch` now captures `gen = navGeneration` before its `await`
and checks both `seq !== searchRequestSeq || gen !== navGeneration` in the
success and error branches before writing `searchResults`/`searchState`, so
a search issued against a since-navigated-away-from webspace can no longer
overwrite the current webspace's search state.

### WR-03: `ensurePolling`'s `setInterval` is never cleared on component teardown

**Files modified:** `web/src/routes/w/[webspace]/+page.svelte`
**Commit:** 22d681f
**Applied fix:** Added a dedicated `$effect(() => { return () => { if
(pollHandle !== null) clearInterval(pollHandle); }; });` teardown effect, as
recommended, so the poll interval is cleared if the component is destroyed
while a sync is still in flight.

### WR-04: Duplicated loading-skeleton markup between `StreamList.svelte` and `SearchResults.svelte`

**Files modified:** `web/src/lib/components/StreamLoadingSkeleton.svelte` (new), `web/src/lib/components/StreamList.svelte`, `web/src/lib/components/SearchResults.svelte`
**Commit:** 5cc393d
**Applied fix:** Extracted the byte-identical four-skeleton-row markup into a
new `StreamLoadingSkeleton.svelte` component (sibling to the existing
`StreamEmpty.svelte`/`StreamError.svelte` pattern) and replaced both call
sites' inline markup with `<StreamLoadingSkeleton />`.

## Skipped Issues

None — all in-scope findings were fixed.

---

_Fixed: 2026-08-07T09:33:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
