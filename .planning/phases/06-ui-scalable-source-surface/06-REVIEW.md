---
phase: 06-ui-scalable-source-surface
reviewed: 2026-08-07T00:00:00Z
depth: standard
files_reviewed: 32
files_reviewed_list:
  - docs/api.md
  - go.mod
  - kernel/httpapi/agent.go
  - kernel/httpapi/item.go
  - kernel/httpapi/item_test.go
  - kernel/httpapi/rendition.go
  - kernel/httpapi/rendition_test.go
  - web/src/app.css
  - web/src/lib/api.ts
  - web/src/lib/components/DetailPane.svelte
  - web/src/lib/components/fidelity.test.ts
  - web/src/lib/components/highlight.test.ts
  - web/src/lib/components/marker-overlay.test.ts
  - web/src/lib/components/markers.test.ts
  - web/src/lib/components/OpenInSource.svelte
  - web/src/lib/components/search-emphasis.test.ts
  - web/src/lib/components/SearchResults.svelte
  - web/src/lib/components/source-chip-selected.test.ts
  - web/src/lib/components/SourceChip.svelte
  - web/src/lib/components/sources.test.ts
  - web/src/lib/components/staleness.test.ts
  - web/src/lib/components/StreamDateMarkers.svelte
  - web/src/lib/components/StreamList.svelte
  - web/src/lib/components/StreamRow.svelte
  - web/src/lib/components/ui/popover/index.ts
  - web/src/lib/components/ui/popover/popover-content.svelte
  - web/src/lib/components/ui/popover/popover.svelte
  - web/src/lib/components/ui/popover/popover-trigger.svelte
  - web/src/lib/components/WebspaceHeader.svelte
  - web/src/lib/format.ts
  - web/src/lib/resize-observer.test.ts
  - web/src/lib/resize-observer.ts
  - web/src/routes/w/[webspace]/+page.svelte
findings:
  critical: 2
  warning: 4
  info: 2
  total: 8
status: issues_found
---

# Phase 06: Code Review Report

**Reviewed:** 2026-08-07
**Depth:** standard
**Files Reviewed:** 32
**Status:** issues_found

## Summary

This is a fresh, full re-review of the phase — the earlier mid-phase
06-REVIEW.md (chip-row gap math, `?hl=` leaking onto `/thumbnail`, unbounded
term length, stale-syncing-spinner-on-load) is superseded; all four of its
findings are independently confirmed fixed in the current code (`visibleChipCount`
now threads a real `gapWidth`, `renditionHandler` now gates `hl` on the
`PREVIEW` variant with a regression test, both `highlightTerms` implementations
now cap term length at 64, and `loadSources()` now calls `ensurePolling()` when
it observes an already-syncing source) and are not repeated below.

The kernel-side UI-09 highlighting work (`kernel/httpapi/rendition.go`,
`item.go`, `agent.go`) is solid: fail-closed content-shape handling, bounded
term derivation, tree-mutation-only highlighting, and thorough test coverage
including multi-byte safety and non-re-sanitization guarantees. The
gap-closure frontend work (`SourceChip.svelte`, `StreamDateMarkers.svelte`, the
shared `.search-highlight` class, the `visibleChipCount` overflow math,
`dateMarkers`/`markerLaneTop`) is also well tested, including several
structural "source-scan" guard tests specifically designed to catch the exact
class of gap that slipped through the prior UAT round (G-06-1, G-06-3, G-06-6).

The two blockers below are both stale-response races, and neither is
unit-tested: they are the same class of bug the codebase's own `handleSearch`
sequence-number guard was explicitly built to prevent elsewhere — they exist
in the two places (`DetailPane.svelte`'s content fetch, `+page.svelte`'s stream
fetch) that don't yet have that guard, plus a related gap in the existing
search guard itself. A user who clicks through items or webspaces quickly
enough (a genuinely reachable interaction, not a contrived edge case) can end
up looking at content that doesn't belong to what's currently selected or
displayed, with no visual indication anything is wrong.

## Critical Issues

### CR-01: DetailPane content fetch has no stale-response guard — rapid item switching can show the wrong item's content

**File:** `web/src/lib/components/DetailPane.svelte:50-72`

**Issue:** `loadContent(id)` is invoked from `$effect(() => { loadContent(item.id); });`
every time the selected item changes. `loadContent` unconditionally assigns
`content = detail.content` (or sets `fetchErrorCode`) when its `getItem(id)`
promise resolves, with no check that `id` is still the currently selected item:

```svelte
async function loadContent(id: string) {
	loadingContent = true;
	fetchErrorCode = null;
	content = null;
	try {
		const detail = await getItem(id);
		content = detail.content;   // <- no guard that `id` is still selected
	} catch (err) {
		fetchErrorCode = err instanceof ApiError ? err.code : 'unknown_error';
	} finally {
		loadingContent = false;
	}
}

$effect(() => {
	loadContent(item.id);
});
```

If a user selects item A (slow plugin `Fetch`, e.g. a temporarily loaded-down
source) and then quickly selects item B (fast `Fetch`), B's fetch resolves
first and correctly renders. When A's slower fetch later resolves, its
`content = detail.content` (A's data) overwrites what's on screen — the header
(title, labels, date, `OpenInSource` link) still correctly shows item B
(rendered synchronously from the `item` prop, stage one of this component's own
documented two-stage render), but the body now shows item A's PDF/text/HTML
rendition. There is no visual indicator that this happened. This is exactly the
race class `+page.svelte`'s `handleSearch` already guards against with its own
`searchRequestSeq` counter (`web/src/routes/w/[webspace]/+page.svelte:190-214`)
— that pattern is simply missing here.

**Fix:** Add the same sequence-number (or `AbortController`) guard already used
for search:

```svelte
let contentRequestSeq = 0;

async function loadContent(id: string) {
	loadingContent = true;
	fetchErrorCode = null;
	content = null;
	const seq = ++contentRequestSeq;
	try {
		const detail = await getItem(id);
		if (seq !== contentRequestSeq) return; // a newer selection has since superseded this one
		content = detail.content;
	} catch (err) {
		if (seq !== contentRequestSeq) return;
		fetchErrorCode = err instanceof ApiError ? err.code : 'unknown_error';
	} finally {
		if (seq === contentRequestSeq) loadingContent = false;
	}
}
```

### CR-02: `+page.svelte`'s stream fetch has no stale-webspace guard — rapid webspace navigation can show the wrong webspace's stream

**File:** `web/src/routes/w/[webspace]/+page.svelte:95-104, 218-225`

**Issue:** SvelteKit reuses this page component instance across
`/w/A` → `/w/B` navigation (the route matches the same file; only
`page.params.webspace` changes) — this is exactly why the file already resets
`selectedId`/`searchQuery`/`searchResults` in the webspace-keyed `$effect`. But
`load()` never checks, after its `await`, that the webspace it was called for
is still the current one:

```svelte
async function load() {
	loadState = 'loading';
	try {
		response = await getStream(webspace);   // `webspace` captured at call time
		loadState = 'ready';                    // written unconditionally on resolve
	} catch {
		response = null;
		loadState = 'error';
	}
}

$effect(() => {
	selectedId = null;
	searchQuery = '';
	searchState = 'idle';
	searchResults = [];
	load();
	loadSources();
});
```

If a user navigates from webspace A (with a slow in-flight `getStream(A)`
call) to webspace B, and A's request resolves after the navigation, `response`
is silently overwritten with webspace A's items while the URL, page title
(`<svelte:head><title>{webspace} — webspaces</title></svelte:head>`) and header
all show webspace B — the stream pane renders items that don't belong to the
webspace the user is now looking at, with `StreamDateMarkers`/`selectedItem`
resolution following right along with the wrong data.

**Fix:** Track a generation/sequence number for webspace navigation (mirroring
the existing `searchRequestSeq` pattern) and check it after the `await` before
writing `response`/`loadState`:

```svelte
let navGeneration = 0;

async function load(gen: number) {
	loadState = 'loading';
	try {
		const res = await getStream(webspace);
		if (gen !== navGeneration) return;
		response = res;
		loadState = 'ready';
	} catch {
		if (gen !== navGeneration) return;
		response = null;
		loadState = 'error';
	}
}

$effect(() => {
	const gen = ++navGeneration;
	selectedId = null;
	searchQuery = '';
	searchState = 'idle';
	searchResults = [];
	load(gen);
	loadSources();
});
```

See WR-03 below for the related gap in `handleSearch`'s own guard, which
should be folded into the same `navGeneration` check.

## Warnings

### WR-01: Client and kernel `highlightTerms` diverge on multi-byte (astral) characters, contradicting the documented "must never diverge" invariant

**File:** `web/src/lib/format.ts:513-537` vs `kernel/httpapi/rendition.go:298-340`

**Issue:** Both files document that their term-derivation rules "must stay in
step, term-for-term" so the client's plain-text/media highlighting never
disagrees with the kernel's iframe-rendered highlighting for the same query.
The `<2`/`>64` length checks, however, count different units:

- Kernel: `n := utf8.RuneCountInString(f); if n < 2 || n > highlightTermMaxRunes`
  (Unicode code points).
- Client: `if (term.length < 2 || term.length > HIGHLIGHT_TERM_MAX_LENGTH)`
  (UTF-16 code units).

For any search term containing astral-plane characters (emoji, many CJK
Extension B/C characters, mathematical alphanumeric symbols, etc.), a JS
string's `.length` is larger than its Unicode code-point count (each astral
character is a 2-unit surrogate pair). Concretely:
- A single-emoji term ("😀", 1 code point) is dropped by the kernel (`< 2`
  runes) but kept by the client (`.length === 2`) — the kernel-rendered
  document (SilverBullet/Proton/Signal renditions) won't highlight it, the
  SPA's own title/body/snippet text will.
- A 40-emoji term (40 code points, well under the 64-rune cap) is kept by the
  kernel but dropped by the client (`.length === 80 > 64`).

**Fix:** Count Unicode code points on the client too, e.g. via
`[...term].length` (spreads by code point) instead of `term.length`, for both
the `<2` and `>64` comparisons in `highlightTerms`.

### WR-02: `+page.svelte`'s `handleSearch` stale-response guard doesn't cover cross-webspace navigation

**File:** `web/src/routes/w/[webspace]/+page.svelte:190-214`

**Issue:** `searchRequestSeq` correctly prevents an older search *within the
same webspace* from overwriting a newer one, but nothing invalidates it when
the webspace itself changes — the webspace-keyed `$effect`
(`+page.svelte:218-225`) resets `searchQuery`/`searchState`/`searchResults` but
never bumps `searchRequestSeq`. If a search issued against webspace A is still
in flight when the user navigates to webspace B, and A's `searchWebspace`
response resolves after the navigation, its `seq` still equals the unchanged
`searchRequestSeq`, so `searchResults` (and `searchState`) get overwritten with
webspace A's search results while the page now shows webspace B.

**Fix:** Fold this into CR-02's proposed `navGeneration` guard — check both
`seq === searchRequestSeq` and `gen === navGeneration` (captured at the start
of `handleSearch`) before writing `searchResults`/`searchState`.

### WR-03: `ensurePolling`'s `setInterval` is never cleared on component teardown

**File:** `web/src/routes/w/[webspace]/+page.svelte:132-142`

**Issue:** `ensurePolling()` starts a `setInterval` that clears itself once no
source is syncing, but there is no `$effect` cleanup / `onDestroy` that clears
`pollHandle` if the component itself is destroyed (e.g. the user navigates away
from the `/w/[webspace]` route tree entirely) while a sync is still in flight.
The interval keeps firing `loadSources()` (and reassigning `sources`) against a
torn-down component's state until the in-flight sync finishes on its own —
wasted network/CPU work and, on a stricter runtime, a source of "state updated
after destroy" warnings.

**Fix:** Register the interval's teardown in an effect, e.g.:

```svelte
$effect(() => {
	return () => {
		if (pollHandle !== null) clearInterval(pollHandle);
	};
});
```

### WR-04: Duplicated loading-skeleton markup between `StreamList.svelte` and `SearchResults.svelte`

**File:** `web/src/lib/components/StreamList.svelte:51-61`, `web/src/lib/components/SearchResults.svelte:35-43`

**Issue:** Both components render byte-identical loading skeletons:

```svelte
<div class="flex flex-col gap-3">
	{#each Array(4) as _, i (i)}
		<Skeleton class="stream-row-surface w-full rounded-lg" />
	{/each}
</div>
```

This is the same "shared surface, declared twice" pattern the phase's own
`.search-highlight` class relocation (G-06-1) exists to prevent for the match
treatment — a future change to the skeleton count/dimensions has to be made in
two places and can silently drift.

**Fix:** Extract a small `StreamLoadingSkeleton.svelte` (or a shared snippet)
and use it from both call sites.

## Info

### IN-01: `getJSON`/`postJSON` in `web/src/lib/api.ts` remain near-duplicate implementations

**File:** `web/src/lib/api.ts:131-171`

**Issue:** `getJSON` and `postJSON` share an identical body (error-envelope
parse, `ApiError` construction, generic-fallback message), differing only in
the `fetch` call's method. This predates this phase but is still present in
the reviewed file: a DRY violation that risks the two error-handling paths
drifting out of sync if one is edited without the other.

**Fix:** Factor the shared response-handling logic into one helper, e.g.
`async function request<T>(path: string, init?: RequestInit): Promise<T>`,
with `getJSON`/`postJSON` as thin wrappers over it.

### IN-02: `dateMarkers`'s degenerate spacing-floor thinning can silently drop the one marker that would have carried `major: true`

**File:** `web/src/lib/format.ts:746-838`

**Issue:** `major` is computed once per candidate, in `candidateMarkers`,
relative to the immediately preceding *candidate* (not the previously *kept*
marker). This is documented as sound because, at the `candidateMarkers` stage,
"kept" and "pushed" coincide. Once `enforceSpacingFloor` further thins the
month-level candidate list in the extreme degenerate case (e.g. 200 monthly
candidates on a 50px track, `markers.test.ts`'s own fixture), a candidate whose
`major` flag was correctly `true` (because it is the actual year-boundary
month) can be dropped by the spacing floor, while a later, non-boundary month
survives with `major: false` — so the final rendered ruler can, in this
extreme case, show no major/year tick at all for a year that did occur in the
data. `markers.test.ts`'s own degenerate-case test ("carries the major flag
through the degenerate spacing-floor thinning untouched") only asserts each
surviving marker's `major` is a boolean and the first is `true` — it doesn't
assert the semantic correctness of major placement post-thinning, so this
wouldn't be caught by the current suite.

**Fix:** Low priority given how extreme the triggering scenario is (16+ years
of history compressed into a 50px track). If it's worth closing, `major`
should be recomputed against the previously *kept* marker inside
`enforceSpacingFloor` rather than carried through from `candidateMarkers`
unchanged.

---

_Reviewed: 2026-08-07_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
