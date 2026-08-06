---
phase: 06-ui-scalable-source-surface
reviewed: 2026-08-06T00:00:00Z
depth: standard
files_reviewed: 24
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
  - web/src/routes/w/[webspace]/+page.svelte
findings:
  critical: 0
  warning: 2
  info: 2
  total: 4
status: issues_found
---

# Phase 06: Code Review Report

**Reviewed:** 2026-08-06T00:00:00Z
**Depth:** standard
**Files Reviewed:** 24
**Status:** issues_found

## Summary

The kernel-side work (`kernel/httpapi/rendition.go`, `item.go`, `agent.go`) is careful and well-defended: sanitize-then-wrap ordering is preserved, highlighting is a tree mutation via `golang.org/x/net/html` (never string concatenation), the `/agent/v1` rendition mirror deliberately passes `nil` terms so it never highlights, the CSP header string is byte-identical across both namespaces, and the extensive `rendition_test.go` suite directly exercises the XSS-relevant invariants (script stripping, `javascript:` href stripping, no re-sanitization of the highlighter's own insertions, multi-byte safety, attribute/tag-byte non-interference, and the element-boundary backstop). I traced the sanitize → highlight → wrap pipeline end to end and found no injection path: matched text is only ever inserted as `html.Node` tree nodes rendered back out through the encoder, never spliced into raw bytes.

The client-side highlighting (`format.ts`, `DetailPane.svelte`) mirrors the kernel's term-derivation rule and renders exclusively through Svelte's default text binding (no `{@html}`), so no new XSS surface is introduced there either.

The one substantive defect found is functional, not security: `WebspaceHeader.svelte`'s new measured-overflow feature (the phase's own headline "Scaling to 10+ instances" deliverable) wires up a `ResizeObserver` that never actually observes anything, because the observed elements don't exist yet at the time `onMount` runs (they're gated behind an async `sourcesState === 'ready'` condition). The feature still "works" on first paint (a `$effect` keyed on `sources`/`selectedSources` does one correct measurement pass), but silently stops responding to window resizes for the rest of the session. Two smaller items round out the findings: an unbounded per-term length in the UI-09 highlighter (the document explicitly frames the 8-term/2-character bounds as the "bounded-work control for threat T-06-03," but never bounds an individual term's length), and a test-coverage gap at the HTTP-handler boundary for `?hl=`.

## Warnings

### WR-01: Chip-overflow `ResizeObserver` never observes anything — window resize breaks the new measured-overflow row

**File:** `web/src/lib/components/WebspaceHeader.svelte:118-125`

**Issue:** The overflow-measurement elements (`rowEl`, `measureEl`, `trailingEl`) are only bound (via `bind:this`) inside the `{#if showSourceRows}` block, and `showSourceRows` is `false` on first render because `sourcesState` starts as `'loading'` (see `web/src/routes/w/[webspace]/+page.svelte:67`, `sourcesState: ... = $state('loading')`) and only flips to `'ready'` after the async `GET /api/sources` call resolves. `onMount` — which both takes the first measurement and constructs the `ResizeObserver` — runs once, synchronously after the component's *initial* mount, i.e. before that async resolution ever happens:

```js
onMount(() => {
    measure();
    const observer = new ResizeObserver(() => measure());
    if (rowEl) observer.observe(rowEl);        // rowEl is undefined here
    if (measureEl) observer.observe(measureEl); // measureEl is undefined here
    if (trailingEl) observer.observe(trailingEl); // trailingEl is undefined here
    return () => observer.disconnect();
});
```

At the moment `onMount` runs, `showSourceRows` is still `false`, so none of `rowEl`/`measureEl`/`trailingEl` exist yet — every `if (xEl)` guard is false, so `observer.observe(...)` is never called for anything. The `ResizeObserver` instance is created but permanently idle for the lifetime of the component; nothing re-registers it once the elements are later bound.

The feature isn't completely broken — the `$effect` at line 132-136 (`sources; selectedSources; queueMicrotask(measure);`) does trigger one correct `measure()` pass the first time `sources` populates (i.e. right after the `sourcesState → 'ready'` transition), so the initial layout is usually correct. But because the `ResizeObserver` was never actually attached, **any subsequent browser-window resize, sidebar toggle, or other layout change that doesn't also change `sources`/`selectedSources` never re-triggers `measure()`.** `visibleCount`/`visibleSources`/`hiddenSources` then stay frozen at whatever the row's width happened to be at load time — chips can end up visually clipped by `overflow-hidden` on `rowEl` without the "+N" overflow trigger ever appearing (or appearing with a stale count), for the entire remainder of the session. This is exactly the resize-adaptiveness the surrounding doc comment (lines 76-82) describes as the intended behavior ("A ResizeObserver watches `rowEl` and `measureEl`... and `trailingEl`").

**Fix:** Attach the observer reactively to whichever elements are currently bound, rather than once in `onMount`. For example, move observer setup into an `$effect` keyed on the refs themselves so it re-runs (and re-observes) whenever they change identity:

```js
$effect(() => {
    if (!rowEl || !measureEl || !trailingEl) return;
    const observer = new ResizeObserver(() => measure());
    observer.observe(rowEl);
    observer.observe(measureEl);
    observer.observe(trailingEl);
    measure();
    return () => observer.disconnect();
});
```

(Drop the `onMount` block entirely, or keep it only for the very first `measure()` call.) This also incidentally fixes `overflowTriggerMeasureEl`, which today is read inside `measure()` but was never added to the observer's watch list at all (see IN-01 below).



### WR-02: `highlightTerms` bounds term *count* and minimum length, but not maximum term length

**File:** `kernel/httpapi/rendition.go:298-326`, `web/src/lib/format.ts:490-511`

**Issue:** `rendition.go`'s own doc comment frames the whitespace-split/lowercase/de-duplicate/drop-sub-2-char/cap-at-8 rule as "the bounded-work controls for threat T-06-03" (line 305). That framing is only half true: the 8-term cap and the 2-character minimum bound the *number* of terms and filter out noise, but neither implementation bounds an individual term's *length*. `strings.Fields`/`query.split(/\s+/)` split only on whitespace, so a caller can supply `?hl=` (or a pasted search-box query) containing a single very long whitespace-free token — e.g. tens of thousands of characters — and it survives as one term, unchanged, all the way into `highlightTextNode`'s per-text-node rune-by-rune scan (`kernel/httpapi/rendition.go:405-457`), which is re-run for every text node in the document for every request. Because this is a loopback-only, unauthenticated API (per `docs/api.md`'s own threat model, any local process can already reach this route), the practical impact is limited to a self-inflicted slow response rather than a cross-user DoS — but it's a real gap relative to the stated "bounded-work" invariant, and the same gap exists identically in the client's `highlightTerms` (`web/src/lib/format.ts:498-511`).

**Fix:** Add a maximum rune-length cap per term (e.g. drop or truncate any field longer than some fixed bound, such as 64 runes) in both `highlightTerms` implementations, keeping the two in step the same way the 8-term cap and 2-character minimum already are.

## Info

### IN-01: `overflowTriggerMeasureEl` is read by `measure()` but never registered with the `ResizeObserver`

**File:** `web/src/lib/components/WebspaceHeader.svelte:112-115, 118-124`

**Issue:** `measure()` reads `overflowTriggerMeasureEl.offsetWidth` (lines 112-115), but unlike `rowEl`/`measureEl`/`trailingEl`, `overflowTriggerMeasureEl` is never passed to `observer.observe(...)` in `onMount`. Its width only changes when the digit count of `sources.length` changes (the hidden clone renders `+{sources.length}`), which today happens to coincide with a `sources` array identity change and therefore does get picked up by the `$effect` at line 132-136 — but this is incidental, not by design, and would silently stop working if the trigger's rendered content ever depended on anything else.

**Fix:** Once WR-01 is fixed by moving observer setup into a ref-driven `$effect`, add `observer.observe(overflowTriggerMeasureEl)` alongside the other three elements for consistency, rather than relying on the `sources` effect as an implicit substitute.

### IN-02: `?hl=` / search-term highlighting has no test coverage at the HTTP-handler boundary

**File:** `kernel/httpapi/item.go:210`, `kernel/httpapi/agent.go:350`

**Issue:** `kernel/httpapi/rendition_test.go` thoroughly tests the pure functions `sanitizeAndWrapRendition`/`highlightTerms`/`highlightTextNodes` directly (passing `terms` in as an explicit argument), but neither `kernel/httpapi/item_test.go` nor `kernel/httpapi/agent_test.go` was touched in this phase (confirmed via `git diff` against the phase's diff base — zero changes to either file). This means the actual wiring — `terms := highlightTerms(r.URL.Query().Get("hl"))` in `renditionHandler`, and the fact that `agentRenditionHandler` never reads the query string at all — is unverified by any test that issues a real `GET .../content?hl=...` request through the router. A regression here (e.g. someone accidentally reading `hl` in `agentRenditionHandler`, or mis-wiring the query key) would not be caught by the existing suite.

**Fix:** Add a handler-level test (in `item_test.go`) that issues `GET /api/items/{id}/content?hl=needle` against a `text/html` fixture and asserts the response body contains `<mark>needle</mark>`, plus a companion test on `agent_test.go` asserting `GET /agent/v1/items/{id}/content?hl=needle` (for a granted item) never highlights, to close the gap between the unit-tested pure function and the route that actually calls it.

---

_Reviewed: 2026-08-06T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
