---
phase: 06-ui-scalable-source-surface
reviewed: 2026-08-07T00:00:00Z
depth: standard
files_reviewed: 33
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
  - web/src/lib/components/source-chip-pill.test.ts
  - web/src/lib/components/source-chip-selected.test.ts
  - web/src/lib/components/SourceChip.svelte
  - web/src/lib/components/sources.test.ts
  - web/src/lib/components/staleness.test.ts
  - web/src/lib/components/StreamDateMarkers.svelte
  - web/src/lib/components/StreamList.svelte
  - web/src/lib/components/StreamLoadingSkeleton.svelte
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
  critical: 0
  warning: 1
  info: 4
  total: 5
status: issues_found
---

# Phase 06: Code Review Report

**Reviewed:** 2026-08-07
**Depth:** standard
**Files Reviewed:** 33
**Status:** issues_found

## Summary

This is a fresh, full re-review of the phase in its current, post-gap-closure
state — three plans (06-06 selected-chip colour, 06-07 date-marker ruler
rebuild, 06-08 source-chip pill polish) landed after the prior 06-REVIEW.md
was written. I re-verified rather than assumed: `go build ./...`,
`go vet ./...`, `go test ./kernel/httpapi/...`, the full vitest suite (257
tests / 15 files), `npm run build`, and `npx svelte-check` were all run
clean during this review, with zero errors/warnings on any file in scope.

**All four prior findings that carried a Critical/Warning severity are
independently confirmed fixed in the current code**, matching their
corresponding commits:
- CR-01 (DetailPane stale-response race): `DetailPane.svelte` now has a
  `contentRequestSeq` guard, checked after every `await`, exactly as
  prescribed.
- CR-02 / WR-02 (cross-webspace/-search stale races): `+page.svelte` now has
  a `navGeneration` counter, bumped once per webspace-keyed `$effect` run and
  checked in both `load()` and `handleSearch()`.
- WR-03 (uncleared poll interval): `+page.svelte` now has an `$effect`
  returning a `clearInterval` teardown.
- WR-04 (duplicated skeleton markup): extracted into
  `StreamLoadingSkeleton.svelte`, used by both `StreamList.svelte` and
  `SearchResults.svelte`.
- The prior WR-01 (kernel/client `highlightTerms` rune-vs-UTF16-unit count
  divergence) is also confirmed fixed: the client's `highlightTerms` now
  counts via `[...term].length` (code points), matching the kernel's
  `utf8.RuneCountInString`.

I traced the client/kernel UI-09 highlighting contract end-to-end again
(`kernel/httpapi/rendition.go`'s `highlightTerms`/`highlightTextNode` vs.
`web/src/lib/format.ts`'s `highlightTerms`/`highlightText`), since the two
are explicitly documented as a "must never diverge" pair, and this time
found a *different*, still-open divergence in the client's `highlightText`
matching logic (not the already-fixed term-derivation function) — see WR-01
below. I also re-traced the `WebspaceHeader.svelte` overflow-measurement
pipeline, the `SourceChip.svelte` pill geometry/reveal fixes from 06-08, the
`StreamDateMarkers.svelte`/`dateMarkers` gutter-lane rebuild from 06-07, and
the agent-grant filtering in `agent.go` against their own documented
invariants; all held up. Two Info items from the prior review that are
**not** yet addressed are carried forward below (`api.ts` duplication,
`dateMarkers`'s degenerate major-flag edge case), plus two new Info items
found this pass.

## Warnings

### WR-01: Client-side `highlightText` can misalign matches on Unicode text where `toLowerCase()` changes string length

**File:** `web/src/lib/format.ts:569-604` (`highlightText`)

**Issue:** `highlightText` computes `const lower = text.toLowerCase();` once,
up front, and then scans/slices using the *same* numeric index `i` into both
`lower` (for matching) and the original `text` (for the substrings pushed
into `segments`):

```ts
const lower = text.toLowerCase();
...
while (i < text.length) {
  let matchLen = 0;
  for (const term of sorted) {
    if (lower.startsWith(term, i)) { matchLen = term.length; break; }
  }
  ...
  segments.push({ text: text.slice(i, i + matchLen), match: true });
  ...
}
```

This assumes `lower.length === text.length` and that every code unit of
`text` maps 1:1 to the same-offset code unit of `lower`. That assumption is
false for a real, non-exotic character: `'İ'` (U+0130, LATIN CAPITAL LETTER
I WITH DOT ABOVE — the capital form used in "İstanbul" and any other
Turkish-orthography proper noun that could plausibly appear in a synced
email/chat/document title) lowercases in JS to a **two-code-unit** string
(`'i'` + a combining dot-above), one UTF-16 unit longer than the
single-unit input. The moment a document's title/preview/body text contains
this character, `lower` becomes one code unit longer than `text` from that
point on, and every subsequent `lower.startsWith(term, i)` check is
evaluated against the wrong offset relative to `text[i]` — a match can be
silently missed, or (worse) reported at an index whose corresponding
`text.slice(i, i+matchLen)` no longer contains the matched characters,
producing a visibly wrong highlight (the wrong substring boxed in amber) for
everything after the divergent character. This is user-visible content
corruption in the search-highlight UI, not merely a missed highlight.

This is a genuine regression risk specifically because the **kernel-side**
counterpart this file's own doc comment says must "stay in step, term-for-
term and tie-break-for-tie-break" (`kernel/httpapi/rendition.go`'s
`highlightTextNode`) was written *specifically* to avoid this exact class of
bug — its own comment reads:

> "case-folding can change a string's byte length for some Unicode text, and
> comparing rune-by-rune against already-lowercased terms sidesteps that
> entirely, so a multi-byte rune adjacent to (or inside) a match is never
> split or corrupted"

The Go implementation achieves this by comparing rune-by-rune with
`unicode.ToLower` per rune (never bulk-transforming the whole string), which
sidesteps length divergence because Go's `unicode.ToLower` is a strict 1:1
rune mapping with no multi-rune expansion. The client's `highlightText` does
the opposite: a single bulk `String.prototype.toLowerCase()` call over the
whole string, which — unlike Go's `unicode.ToLower` — implements Unicode's
context-independent special casing (including the İ expansion) and can
change the string's length. `highlight.test.ts` has no test for this case —
every fixture in that file is plain ASCII or simple accented text — so this
divergence would not be caught by the existing suite, and it is a distinct
bug from the already-fixed term-length-counting WR-01 in the prior review
(this one is in the *scan/match* loop over the source text, not in the
*term-derivation* function).

**Fix:** Scan `text` at the code-point level rather than relying on a
single length-preserving bulk transform staying aligned with the original —
e.g. lower-case each candidate window individually at match time (mirroring
the kernel's per-rune comparison) instead of pre-computing one bulk-
lowercased string and indexing into it positionally. At minimum, add a
regression test using `'İstanbul'` (or another word containing a
case-fold-expanding character) to `highlight.test.ts`, mirroring
`kernel/httpapi/rendition_test.go`'s `TestHighlight_MultiByteSafety`, so this
class of bug is caught mechanically going forward.

## Info

### IN-01: `getJSON`/`postJSON` in `web/src/lib/api.ts` remain near-duplicate implementations

**File:** `web/src/lib/api.ts:131-171`

**Issue:** Carried forward from the prior review, still present and
unaddressed. `getJSON` and `postJSON` share an identical body (error-envelope
parse, `ApiError` construction, generic-fallback message), differing only in
the `fetch` call's method:

```ts
async function getJSON<T>(path: string): Promise<T> { ... }
async function postJSON<T>(path: string): Promise<T> { ... }
```

A DRY violation that risks the two error-handling paths drifting out of sync
if one is edited without the other — predates this phase but is still
present in the reviewed file.

**Fix:** Factor the shared response-handling logic into one helper, e.g.
`async function request<T>(path: string, init?: RequestInit): Promise<T>`,
with `getJSON`/`postJSON` as thin wrappers over it.

### IN-02: `dateMarkers`'s degenerate spacing-floor thinning can silently drop the one marker that would have carried `major: true`

**File:** `web/src/lib/format.ts:753-805` (`candidateMarkers`,
`enforceSpacingFloor`)

**Issue:** Carried forward from the prior review; the 06-07 date-marker
ruler rebuild did not touch this logic. `major` is computed once per
candidate, inside `candidateMarkers`, relative to the immediately preceding
*candidate* — sound at that stage because "kept" and "pushed" coincide there.
But `enforceSpacingFloor` (the degenerate-case backstop for when even
month-granularity candidates violate the 24px spacing floor) filters
candidates without ever recomputing `major`:

```ts
function enforceSpacingFloor(markers: DateMarker[]): DateMarker[] {
	if (markers.length === 0) return markers;
	const kept: DateMarker[] = [markers[0]];
	for (let i = 1; i < markers.length; i += 1) {
		const candidate = markers[i];
		const last = kept[kept.length - 1];
		if (candidate.topPx - last.topPx >= MARKER_SPACING_FLOOR_PX) {
			kept.push(candidate);   // `candidate.major` carried through unchanged
		}
	}
	return kept;
}
```

In the extreme degenerate case this function exists for (e.g. many years of
monthly candidates compressed onto a short track), a candidate whose `major`
flag was correctly `true` (because it is the actual year-boundary month) can
be dropped by the spacing floor, while a later, non-boundary month survives
with `major: false` — so the rendered ruler can show no major/year tick at
all for a year that did occur in the data. `markers.test.ts`'s degenerate-
case coverage only asserts each surviving marker's `major` is a boolean and
the first kept marker is `true`; it does not assert semantic correctness of
major placement after thinning, so this would not be caught by the current
suite.

**Fix:** Low priority given how extreme the triggering scenario is (many
years of history compressed into a very short track). If closed, `major`
should be recomputed against the previously *kept* marker's coarser key
inside `enforceSpacingFloor`, rather than carried through unchanged from
`candidateMarkers`.

### IN-03: Overflow-trigger measurement clone systematically overestimates its own reserved width

**File:** `web/src/lib/components/WebspaceHeader.svelte:274-284` (hidden
`overflowTriggerMeasureEl` clone) vs. `:199-215` (real, rendered trigger)

**Issue:** The hidden measurement clone used to size the overflow trigger
for `visibleChipCount`'s budget always renders `+{sources.length}` (the
**total** configured source count), while the real, visible trigger renders
`+{hiddenSources.length}` (only the sources actually pushed into overflow,
which is always `<= sources.length` and typically much smaller once any
chips are visible at all). Since `hiddenSources.length` never has more
digits than `sources.length`, the measured clone's text is never narrower
than the real trigger's eventual text, so `overflowTriggerWidth` is always
an over-estimate, never an under-estimate — this cannot cause visual
clipping, but it does mean `visibleChipCount`'s overflow math is
unnecessarily conservative at digit-count boundaries (e.g. 10 configured
sources with only 1 pushed to overflow: measured as `"+10"`, rendered as
`"+1"`), and can hide one more chip than strictly necessary in that case.

**Fix:** Not required for correctness — this fails safe. If exact-fit
precision is ever wanted in practice, note that the clone can't know
`hiddenSources.length` before it's rendered (the same chicken-and-egg
problem the clone exists to solve in the first place), so closing this
would need a second measurement pass rather than a one-line change; leaving
as-is with a short comment noting the tradeoff is a reasonable option too.

### IN-04: Hidden chip-measurement region nests focusable elements inside an `aria-hidden="true"` container without an explicit `tabindex="-1"` backstop

**File:** `web/src/lib/components/WebspaceHeader.svelte:260-273`

**Issue:** The off-screen chip-measurement clone (`measureEl`'s parent
`<div>`) is marked `aria-hidden="true"` and relies solely on the `invisible`
Tailwind utility (`visibility: hidden`) to keep its rendered `SourceChip`
instances — each containing a real `<button>` filter control and a real
`<Button>` refresh control — out of the tab order. `visibility: hidden`
does reliably remove an element from the natural tab order in all current
browsers, so this is not exploitable in normal keyboard use today. However,
`aria-hidden="true"` wrapping focusable descendants is a well-known
accessibility anti-pattern (flagged by axe-core's `aria-hidden-focus` rule):
it is not defense-in-depth against a future change that swaps `invisible`
for something that doesn't remove focusability (e.g. `opacity-0` alone,
which the *visible* refresh-button reveal in `SourceChip.svelte` itself
relies on — a plausible copy-paste mistake if this region is ever edited by
someone matching that nearby pattern).

**Fix:** Not urgent given the current `invisible` mitigation, but consider
adding `tabindex="-1"` to each measurement clone's interactive elements (or
otherwise structurally guaranteeing they can never gain real interactive
children). The sibling `overflowTriggerMeasureEl` clone two elements below
already does this correctly (explicit `tabindex="-1"` *and* `invisible`) —
the chip clone region could match that same belt-and-braces pattern for
consistency.

---

_Reviewed: 2026-08-07_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
