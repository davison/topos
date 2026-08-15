---
phase: 13-per-item-curation-installable-app
reviewed: 2026-08-15T00:00:00Z
depth: standard
files_reviewed: 3
files_reviewed_list:
  - web/src/routes/w/[webspace]/+page.svelte
  - web/e2e/specs/13-undo-across-webspace-switch.spec.ts
  - docs/testing.md
findings:
  critical: 0
  warning: 1
  info: 1
  total: 2
status: issues_found
---

# Phase 13: Code Review Report (13-08 gap-closure re-review)

**Reviewed:** 2026-08-15T00:00:00Z
**Depth:** standard
**Files Reviewed:** 3
**Status:** issues_found

**Scope note:** This is a re-review of the delta introduced by gap-closure
plan 13-08 (`git diff ceca9dd83e4f2e1909766787d24b8574de9259fa..HEAD`, commits
4f67d32/d44fd68/bcf09dc), which closes G-13-1 (a stale-generation `load()`
call could strand the navigated-to webspace's stream in a permanent loading
skeleton). This review supersedes only the parts of the prior 13-REVIEW.md
that this delta touches; WR-02 from that prior review was re-verified
against the current source (not assumed) and is carried forward below
because it remains unfixed in the code as it stands today — 13-08's scope
was G-13-1 only, and nothing in this delta touches the `closeDetail()`/
`handleBulkClear()` call sites WR-02 is about.

## Summary

The core fix — an entry guard at the top of `load()`
(`web/src/routes/w/[webspace]/+page.svelte:902`, `if (gen !== navGeneration)
return;`) — is correct and verified by tracing every call site in the file:

- It only ever actually triggers (returns early) for the three deferred
  `onUndo` closures' `load(gen)` calls and, incidentally, for
  `ensurePolling`'s quiet poll-completion `load(gen, { quiet: true })` call
  if a webspace switch happened mid-tick — both are exactly the deferred-
  callback cases the comment says it targets.
- Every other call site in the file passes `navGeneration` itself, or a `gen`
  already proven equal to the live `navGeneration` by an earlier guard in the
  same function, so the new guard is a structural no-op for those paths —
  confirmed no regression to any of the ~15 other `load(...)` call sites.
- The two existing post-await guards inside `load()` are untouched and still
  correctly cover the separate in-flight-fetch race, exactly as the code
  comment claims.

The new fourth spec (`exclude in A4, switch to EMPTY B4, Undo`) correctly
exercises the fix: `StreamList.svelte`'s `{#if state === 'loading'}` branch
is checked ahead of every other branch (`web/src/lib/components/
StreamList.svelte:85`), so a rendered "Nothing here yet" (`StreamEmpty.svelte:25`)
is legitimate proof `loadState` never flipped to `'loading'`; the
`streamSkeletonLocator` CSS selector (`[data-slot="skeleton"].stream-row-surface`)
only ever matches `StreamLoadingSkeleton.svelte`'s own rows (`StreamRow.svelte`'s
row surface carries the `.stream-row-surface` class but never
`data-slot="skeleton"`), so it cannot produce a false positive against a
populated stream. `clickUndo`'s `page.waitForResponse` registration before
the click, and its `endsWith('/api/webspaces/${markWebspace}/marks')`
matcher, are consistent with `setItemMarks`'s actual request URL
(`web/src/lib/api.ts:553`). The `docs/testing.md` prose update accurately
describes both the fix and the two-layer (kernel-read for A, rendered-DOM
for B) assertion strategy the spec now uses.

No new defects were introduced by this delta. One warning from the prior
review remains open (re-verified against current line numbers, not carried
forward blindly) and one new minor info-level observation was found on the
same code path during this delta's review.

## Warnings

### WR-01: `closeDetail()`/`handleBulkClear()` still run unconditionally after a navigation, ungated by `navGeneration` (carried forward, re-verified, still unfixed)

**File:** `web/src/routes/w/[webspace]/+page.svelte:132` (`handleExclude`),
`:155` (`handleInclude`), `:236` (`handleBulkPrimary`)

**Issue:** The prior review (WR-02 in the superseded 13-REVIEW.md) identified
that `closeDetail()`/`handleBulkClear()` run unconditionally immediately
after the *initial* (non-undo) `setItemMarks` write's `await`, with no
`gen === navGeneration` guard — unlike `load(gen)` in the same functions,
which is correctly guarded (and, after this delta, guarded even more
robustly at entry). 13-08-PLAN.md's scope was G-13-1 (the `load()` entry
guard) only; it did not touch these three call sites, and re-reading the
current source confirms they are still unconditional:

```
132:  closeDetail();       // handleExclude — no gen check
155:  closeDetail();       // handleInclude — no gen check
236:  handleBulkClear();   // handleBulkPrimary — no gen check
```

If a user triggers an exclude/include/bulk-exclude, then navigates to a
different webspace and opens a different item (or makes a new bulk
selection) in the new webspace before the *original* `setItemMarks` request
resolves (a narrow window — normally single-digit milliseconds against the
local kernel, not the 5000ms undo window WR-01/G-13-1 cover), the original
handler's `closeDetail()`/`handleBulkClear()` fires against now-current
state and spuriously deselects the newly opened item or clears the newly
made selection in the *destination* webspace. No data is written to the
wrong webspace (`ws` still correctly targets the origin webspace for the
mark write itself) — this is a UI-only state clobber, not a data-integrity
bug — but it is a real, unintended cross-webspace side effect that the
`navGeneration` discipline this file otherwise applies consistently (see the
extensive `load()`/`writeFilter()`/`handleSearch()` generation-guard comments
throughout the same file) does not yet cover here.

The same shared-state gap also applies, at lower severity, to `markBusy`/
`bulkBusy`: both are set `true` synchronously at handler entry and cleared
unconditionally in each handler's `finally` block. If webspace B's own
detail pane or action bar happens to be rendered when a still-in-flight
webspace-A handler's `finally` fires, B's busy-disabled control flips
enabled/disabled as a side effect of A's request settling — narrower still
than the `closeDetail`/`handleBulkClear` case, but the same missing
`gen === navGeneration` discipline.

**Fix:** Gate the two side effects (and, ideally, the busy-state writes) the
same way `load()`'s own entry guard now does:

```ts
async function handleExclude(id: string) {
	const ws = webspace;
	const gen = navGeneration;
	markBusy = true;
	try {
		await setItemMarks(ws, 'add', [id]);
		if (gen === navGeneration) closeDetail();
		await load(gen);
		...
	} finally {
		if (gen === navGeneration) markBusy = false;
	}
}
```

and correspondingly in `handleInclude` (`closeDetail()`) and
`handleBulkPrimary` (`handleBulkClear()`, `bulkBusy = false`).

## Info

### IN-01: `load()`'s new entry-guard comment block is long enough to bury the guard's one line of actual logic

**File:** `web/src/routes/w/[webspace]/+page.svelte:879-900`

**Issue:** The comment preceding `load()` documenting the G-13-1 fix (~22
lines) is accurate and useful for future readers tracing the bug's history,
but it sits directly above a function whose own doc comment already runs to
~35 lines (the pre-existing `quiet` explanation). The two comment blocks
together are now longer than the function body they describe, making the
single load-bearing line (`if (gen !== navGeneration) return;`) easy to
skim past on a future edit. This matches the file's established heavy-
comment convention (evident throughout — e.g. the `navGeneration` doc
comment at line ~682, the `handleStreamScroll` block at ~769-839) so it is
not a deviation from house style, just a note that a future trim pass could
consider consolidating the two `load()` comment blocks into one narrative
rather than two stacked ones.

**Fix:** Optional. If touched again, consider merging the `quiet` and
entry-guard comment blocks into a single ordered explanation (state machine:
entry guard → quiet branch → success → auto-flip → error) rather than two
separately-dated additions.

---

_Reviewed: 2026-08-15T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
