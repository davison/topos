---
phase: 06-ui-scalable-source-surface
reviewed: 2026-08-06T00:00:00Z
depth: standard
files_reviewed: 26
files_reviewed_list:
  - docs/api.md
  - go.mod
  - kernel/httpapi/agent.go
  - kernel/httpapi/item.go
  - kernel/httpapi/rendition.go
  - kernel/httpapi/rendition_test.go
  - web/src/lib/api.ts
  - web/src/lib/components/DetailPane.svelte
  - web/src/lib/components/fidelity.test.ts
  - web/src/lib/components/highlight.test.ts
  - web/src/lib/components/markers.test.ts
  - web/src/lib/components/OpenInSource.svelte
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
  critical: 1
  warning: 3
  info: 1
  total: 5
status: issues_found
---

# Phase 06: Code Review Report

**Reviewed:** 2026-08-06T00:00:00Z
**Depth:** standard
**Files Reviewed:** 26
**Status:** issues_found

## Summary

This is a fresh full-phase review, superseding an earlier partial review of
this phase directory. That earlier review's single recorded defect (a dead
`ResizeObserver` in `WebspaceHeader.svelte` that never observed anything)
has since been fixed by the 06-04 gap-closure wave (`resize-observer.ts` +
its extraction, verified against `resize-observer.test.ts`'s source-scan
guard) — confirmed not to have regressed.

The kernel-side work (`kernel/httpapi/rendition.go`, `item.go`, `agent.go`)
remains careful and well-defended: sanitize-then-wrap ordering is
preserved, highlighting is a tree mutation via `golang.org/x/net/html`
(never string concatenation), the `/agent/v1` rendition mirror
deliberately passes `nil` terms so it never highlights, and the extensive
`rendition_test.go` suite directly exercises the XSS-relevant invariants.
I traced the sanitize → highlight → wrap pipeline end to end and found no
injection path.

The one substantive **new** defect is a correctness bug in this phase's
own headline deliverable: the chip-row overflow computation
(`visibleChipCount` in `format.ts`, wired up in `WebspaceHeader.svelte`)
never accounts for the `gap-2` (8px) spacing the real flex row applies
between chips — so in exactly the "10+ sources" scenario Phase 06 exists
to solve, the computed "how many chips fit" count is optimistic and the
row can silently clip chips past its `overflow-hidden` edge, with no way
to reach them through the "+N" popover. This was not caught by either the
earlier partial review or the 06-04 gap-closure wave, since both focused
on the (now-fixed) dead-observer defect rather than the underlying width
arithmetic.

Three further items round out the findings: a docs/implementation
mismatch where `?hl=` highlighting is applied to the `/thumbnail` route
despite `docs/api.md`'s explicit "content route only" claim; an unbounded
per-term length in both the kernel and client term-derivation helpers,
despite the docstring's "bounded-work controls for threat T-06-03" claim
(carried forward from the earlier review — still unaddressed); and a
pre-existing (not introduced this phase, but present in reviewed files)
gap where a source already syncing before the page loads never triggers
client-side polling, so its spinner can go stale indefinitely.

## Critical Issues

### CR-01: Chip-row overflow math ignores the real layout's inter-chip gap, so chips can silently clip past the visible row

**File:** `web/src/lib/format.ts:274-293` (`visibleChipCount`), wired up in `web/src/lib/components/WebspaceHeader.svelte:94-153` (esp. lines 104-122, 146-148) against a row that renders with `gap-2` (`WebspaceHeader.svelte:175`)

**Issue:** `visibleChipCount` computes how many chips fit by summing only
`chipWidths` (each chip's own `offsetWidth`) against `availableWidth -
reservedWidth`, and — once it decides not everything fits — against
`budget - overflowTriggerWidth`. It never adds the 8px `gap-2` spacing that
`rowEl` (`WebspaceHeader.svelte:175`, `class="... gap-2 overflow-hidden"`)
actually inserts between every adjacent flex child: each rendered chip,
the overflow-trigger button, and the trailing "Clear filters / Refresh
all" group.

`06-UI-SPEC.md` even names this the "chip row gap" spacing token (`sm`,
8px, line 129), so the gap is a deliberate part of the visual design the
width math was supposed to reserve for — it just never does.

The practical effect: with N visible chips there are `N-1` real 8px gaps
between them, plus one more gap before the overflow trigger (when
present) and one more before the trailing group — none of which
`visibleChipCount` charges against the budget. As the configured source
count grows (the exact "UI-07: usable at 10+ instances" scenario this
phase's headline feature targets), the accumulated unaccounted gap width
(`(N-1) * 8px` and more) can easily exceed the margin between the
optimistic total and the real `clientWidth`, causing the row to actually
overflow `overflow-hidden` in the browser — meaning one or more of the
"visible" chips gets clipped at the trailing edge, invisible and
unreachable, while the overflow popover (computed as if those chips were
still on-screen) never lists them either. A source's filter/refresh chip
can become completely inaccessible from the UI.

This is not caught by the existing unit tests (`sources.test.ts`'s
`visibleChipCount` suite) because those tests exercise the pure function
in isolation with hand-picked widths that happen to not need any gap
margin — they never assert against the real `gap-2` DOM behavior, which
only exists once `WebspaceHeader.svelte` is actually mounted (the vitest
suite runs `environment: 'node'`, with no real layout to catch this).

**Fix:** Thread the row's real gap width through the pure function and
charge it for every adjacent pair — including the boundary into the
overflow trigger and the trailing group — e.g.:

```ts
export function visibleChipCount(
	chipWidths: number[],
	availableWidth: number,
	reservedWidth: number,
	overflowTriggerWidth: number,
	gapWidth: number // NEW — the row's real inter-item gap (8px / gap-2)
): number {
	const budget = availableWidth - reservedWidth;
	const gapsAmong = (n: number) => (n > 0 ? (n - 1) * gapWidth : 0);
	// +1 trailing gap: the space between the last visible item (chip or
	// trigger) and the reserved trailing group.
	const total = chipWidths.reduce((sum, w) => sum + w, 0) + gapsAmong(chipWidths.length) + (chipWidths.length > 0 ? gapWidth : 0);
	if (total <= budget) return chipWidths.length;

	// One more gap for the trigger itself, plus the gap before it.
	const overflowBudget = budget - overflowTriggerWidth - gapWidth * 2;
	let used = 0;
	let count = 0;
	for (const width of chipWidths) {
		const candidateGap = count > 0 ? gapWidth : 0;
		if (used + candidateGap + width > overflowBudget) break;
		used += candidateGap + width;
		count += 1;
	}
	return count;
}
```

and pass a `GAP_PX = 8` (or a value read from the computed style of
`rowEl`, if drift-proofing against a future Tailwind gap-scale change
matters) constant from `WebspaceHeader.svelte`'s call site at line 147.
Add a regression test asserting that a chip count/width combination which
exactly fits *without* gaps (like the existing `[10, 10, 10]` at
`availableWidth: 30` case) now correctly reports fewer visible chips once
gap width is non-zero.

## Warnings

### WR-01: `?hl=` highlighting is applied to `/thumbnail` too, contradicting docs/api.md's "content route only" claim

**File:** `kernel/httpapi/item.go:153-218` (`renditionHandler`, shared verbatim by `ItemContentHandler` and `ItemThumbnailHandler`); contradicts `docs/api.md:362`

**Issue:** `docs/api.md` states plainly: "**`?hl=` (UI-09, optional, `GET
/api/items/{id}/content` only)**". But `renditionHandler` — the single
function both `ItemContentHandler` (`/content`) and `ItemThumbnailHandler`
(`/thumbnail`) delegate to — reads `r.URL.Query().Get("hl")`
unconditionally whenever the resolved rendition's MIME type is
`text/html` (`item.go:210`), with no branch on `variant`. If any plugin
ever returns a `text/html` thumbnail rendition (nothing in the allowlist
or contract prevents this — `text/html` is a globally allowed MIME type,
not restricted to the "full"/"preview" variants), a request to
`GET /api/items/{id}/thumbnail?hl=needle` would silently apply
highlighting, in direct contradiction of the documented contract.

**Fix:** Gate the `hl` read on `variant == toposv1.ContentVariant_CONTENT_VARIANT_PREVIEW`
inside `renditionHandler`, or split the two handlers so only
`ItemContentHandler`'s call site ever derives `terms`:

```go
var terms []string
if variant == toposv1.ContentVariant_CONTENT_VARIANT_PREVIEW {
	terms = highlightTerms(r.URL.Query().Get("hl"))
}
```

Add a test asserting a `text/html` thumbnail response is never
highlighted regardless of `?hl=`.

### WR-02: Individual highlight terms have no upper length bound, despite the docstring's "bounded-work controls" claim

**File:** `kernel/httpapi/rendition.go:298-326` (`highlightTerms`); `web/src/lib/format.ts:490-511` (`highlightTerms`)

**Issue:** Both implementations cap the *number* of derived terms at 8 and
drop any term shorter than 2 runes/characters, and the Go docstring
explicitly frames this as "the bounded-work controls for threat T-06-03."
Neither implementation bounds an individual term's *maximum* length,
though — a caller can supply `?hl=` (or a client-side search query) with
a single very long unbroken "word" (no whitespace), and it survives
`strings.Fields`/`.split(/\s+/)` as one arbitrarily long term. The
existing length-guard in `highlightTextNode`
(`if i+len(termRunes) > n { continue }`) prevents this from becoming a
meaningful CPU/DoS issue against the current implementation (a term
longer than the remaining document text is skipped in O(1) per position),
so the practical risk today is low — but the docstring's framing of "the"
bounded-work control is inaccurate as written, and a future refactor that
removes or weakens that length guard would silently reopen the gap the
docstring claims is already closed.

**Fix:** Either cap term length explicitly (e.g., drop or truncate any
field over some reasonable bound, such as 64 runes/characters, alongside
the existing `<2` drop) in both `highlightTerms` implementations, or
narrow the docstring's claim to describe what's actually bounded (count
and minimum length only) so a future reader doesn't rely on a guarantee
that isn't there.

### WR-03: A source already syncing when the page loads never starts client-side polling, so its spinner can go stale indefinitely

**File:** `web/src/routes/w/[webspace]/+page.svelte:106-133` (`loadSources`/`ensurePolling`), `:209-216` (mount/webspace-change effect)

**Issue:** `ensurePolling()` — the function that starts the 2-second
`GET /api/sources` poll loop and stops it once nothing is syncing — is
only ever called from `handleRefreshSource` and `handleRefreshAll`
(lines 136, 149), i.e. only when *this browser tab* initiates a refresh.
The initial-mount/webspace-change effect (lines 209-216) calls
`loadSources()` directly, with no check of whether the response it gets
back already reports a source `syncing: true` (e.g. the background
scheduler, `topos sync` CLI, or another browser tab kicked one off).
`loadSources()` itself (lines 106-115) also never calls `ensurePolling()`.
The result: if a source happens to be mid-sync when the page is opened or
reloaded, `SourceChip.svelte`'s spinning refresh icon
(`source.syncing && 'animate-spin'`) renders correctly on that first
paint, but nothing schedules the next poll — the spinner will keep
spinning even after the sync actually finishes, until the user manually
triggers a refresh on some source (which then, incidentally, starts
polling and picks up the stale state on its next tick).

Note this pattern predates this phase (it was already present in Phase
2's `+page.svelte`, commit `85cc931`), but the file is in this phase's
review scope and the defect is real and observable in the current code.

**Fix:** Call `ensurePolling()` from `loadSources()` itself whenever the
freshly-loaded response contains any `syncing: true` entry, so any
already-in-flight sync — regardless of who started it — is picked up:

```ts
async function loadSources() {
	try {
		const res = await getSources();
		sources = res.sources;
		sourcesState = 'ready';
		if (sources.some((s) => s.syncing)) ensurePolling();
	} catch {
		sources = [];
		sourcesState = 'error';
	}
}
```

## Info

### IN-01: `getJSON`/`postJSON` in `web/src/lib/api.ts` are near-duplicate implementations

**File:** `web/src/lib/api.ts:131-171`

**Issue:** `getJSON` and `postJSON` share an identical body (error-envelope
parse, `ApiError` construction, generic-fallback message) differing only
in the `fetch` call's method. This is pre-existing (predates this phase),
but it's a DRY violation that risks the two error-handling paths drifting
out of sync if one is edited without the other — worth flagging while
this file is in scope.

**Fix:** Factor the shared response-handling logic into one helper, e.g.
`async function request<T>(path: string, init?: RequestInit): Promise<T>`,
with `getJSON`/`postJSON` as thin wrappers over it.

---

_Reviewed: 2026-08-06T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
