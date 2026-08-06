---
phase: 06-ui-scalable-source-surface
verified: 2026-08-06T22:30:00Z
status: gaps_found
score: 3/4 roadmap success criteria fully verified (1 partial)
behavior_unverified: 0
overrides_applied: 0
gaps:
  - truth: "Success Criterion 1 — the design remains usable at 10+ instances (overflow, grouping, or collapse; no unbounded chip rows), sustained across the component's lifetime"
    status: partial
    reason: "The chip row's overflow computation is correct exactly once — at the moment `sources` first populates from GET /api/sources — and then goes silently stale for the rest of the session. The ResizeObserver that WebspaceHeader.svelte's own doc comment says 'watches rowEl and measureEl ... and trailingEl' never actually observes anything: it is constructed inside `onMount`, which fires once, synchronously, right after the component's first mount. At that instant `sourcesState` is still `'loading'`, `showSourceRows` is `false`, and `rowEl`/`measureEl`/`trailingEl` (only bound via `bind:this` inside the `{#if showSourceRows}` block) are all `undefined` — so every `if (xEl) observer.observe(xEl)` guard fails and `observer.observe()` is never called on anything, for the lifetime of the component. No code later re-attaches the observer once the elements are actually bound. Independently confirmed by reading `web/src/lib/components/WebspaceHeader.svelte:83-125` line by line (not merely trusting SUMMARY/REVIEW claims); the phase's own 06-REVIEW.md documents the identical defect as WR-01/IN-01. Practical consequence: narrowing the browser window (the exact human-check the plan itself recommends as a stand-in for configuring 10+ real instances — 06-02-PLAN.md Task 2's `<human-check>`) will not recompute `visibleChipCount`, so chips can end up clipped by the row's `overflow-hidden` without the '+N' overflow trigger ever appearing or updating — the same failure mode the phase's own prohibition (\"MUST NOT let a source's failure state become invisible\") was written to prevent, now reachable via a plain window resize rather than the overflow popover."
    artifacts:
      - path: "web/src/lib/components/WebspaceHeader.svelte"
        issue: "onMount (lines 118-125) is the only place ResizeObserver.observe() is ever called, and it runs before the observed elements exist. overflowTriggerMeasureEl is never added to the observer's watch list at all (IN-01), even after the timing issue is fixed."
    missing:
      - "Move ResizeObserver construction/attachment into a ref-driven `$effect` (or equivalent re-attachment on ref change) so it re-observes once rowEl/measureEl/trailingEl/overflowTriggerMeasureEl actually exist, and continues to respond to browser-window resizes and chip-width changes for the rest of the session — not just the one measurement pass the `sources`-keyed `$effect` happens to trigger at initial load."
deferred: []
human_verification:
  - test: "Run `make dev`, search a webspace for a word known to appear in an email or a SilverBullet note, open that item, and confirm the word is highlighted amber inside the detail pane's rendered iframe document (email HTML, SilverBullet markdown, Signal chat transcript). Clear the search and confirm the highlight disappears."
    expected: "Matched terms render inside a `<mark>` element with an amber background across all three iframe content shapes; with no search query, output is byte-identical to pre-phase (no mark anywhere)."
    why_human: "The rendition iframe is served under a CSP with no same-origin token (opaque origin) — the SPA cannot script into it, so the only way to confirm the highlight actually renders visually is a live browser check. Kernel-side tree-mutation and sanitizer-boundary logic is proven by 10 passing Go tests; visual confirmation is the remaining gap."
  - test: "Search for a word that appears in a paperless document's extracted text, open it, and confirm the word is highlighted amber below the preview box (the plain-text/media detail-pane variant)."
    expected: "The matched word renders in a highlighted span below the document preview."
    why_human: "Client-side rendering in a live browser; the underlying `highlightText` segmentation logic is proven by 15 passing unit tests including the round-trip and metacharacter-literal invariants."
  - test: "With `make dev` running, confirm one chip per configured source appears in a single row; click two chips and confirm the stream narrows to both and the URL carries both names; reload and confirm the selection survives; hover a chip and confirm the refresh icon appears and the health tooltip reads as it did before; click the refresh icon and confirm only that source refreshes and the filter does not change."
    expected: "One merged chip per instance; multi-select toggling narrows the stream and round-trips through the URL; hover/focus reveals refresh; tooltip copy matches Phase 2; refresh click does not also toggle the filter."
    why_human: "Hover/focus-reveal, hover tooltip rendering, and the multi-click interaction sequence require a live browser session. The underlying filter-resolution, toggle, and serialization logic is proven by 41 passing unit tests in sources.test.ts."
  - test: "Temporarily configure ten or more source instances (or narrow the browser window hard against the existing set) and confirm the chip row stays exactly one line high, an overflow trigger appears showing the hidden count, opening it lists the hidden sources as full chips that filter/refresh identically, and the refresh-all/clear-filters controls stay on the row. Then make one hidden source unreachable and confirm the overflow trigger's dot turns destructive."
    expected: "Single-line row at any instance count; overflow popover reachable in two interactions; worst-of health tone surfaced on the trigger."
    why_human: "Real-DOM measurement and popover interaction require a live browser. NOTE: per the gap recorded above, the 'narrow the browser window' variant of this check is expected to fail post-initial-load, since the ResizeObserver never actually re-measures on resize — this human check will likely surface the WR-01 defect directly if performed against an already-loaded page."
  - test: "Open a Signal conversation item and confirm its button reads `Show in {source}` with a window icon and an explanatory hover title; open a paperless or email item and confirm its button reads `Open in {source}` with a navigate icon; confirm the small fidelity badge still shows the raw enum value in all three cases."
    expected: "Two-class icon/verb/title split, badge unchanged (3 raw values)."
    why_human: "Visual icon rendering and title-attribute hover behavior in a live browser; the `fidelityAffordance` mapping itself is proven by 8 passing unit tests."
  - test: "Open a webspace whose stream spans several dates. Confirm thin ticks appear alongside the stream scrollbar, hovering one shows its date, clicking one jumps that date's first row to the top of the pane, the scrollbar thumb itself still drags, and the rows underneath the overlay still click. Then run a search and confirm the ticks disappear while search results are showing."
    expected: "Clickable, tooltipped date ticks; native scrollbar and row interactivity undisturbed; markers hidden during search."
    why_human: "Pointer-event pass-through, click-to-scroll behavior, and overlay-vs-native-scrollbar interaction require a live browser session. The `dateMarkers` derivation itself (position formula, adaptive thinning, 24px floor, UTC boundary) is proven by 15 passing unit tests."
---

# Phase 6: UI Scalable Source Surface Verification Report

**Phase Goal:** The webspace header presents many source instances without duplication — health and filtering combined into one affordance per source — and the accumulated UI polish lands: deep-link fidelity differentiation, search-term highlighting in the detail pane, and themed scrollbars with date markers
**Verified:** 2026-08-06T22:30:00Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| # | Truth (Roadmap Success Criterion) | Status | Evidence |
|---|---|---|---|
| 1 | Each source instance appears exactly once in the header — a single chip combining health, filter toggle and refresh — usable at 10+ instances (overflow/grouping/collapse; no unbounded rows) | ⚠️ PARTIAL | `SourceChip.svelte` merges health dot + name + `aria-pressed` filter toggle + hover/focus-revealed refresh (confirmed by direct read); `SourceHealthChip.svelte`/`SourceFilterChips.svelte` confirmed absent from the tree; `WebspaceHeader.svelte` uses `flex-nowrap overflow-hidden` plus a measured overflow popover (`visibleChipCount`, `worstHealthTone`) — correct at initial load. **Gap:** the `ResizeObserver` wired to keep this correct across the session never actually observes any element (see Gaps below) — confirmed by direct code read, not merely the code-review report. |
| 2 | Items whose "open in source" link can only raise the source app's window are visually differentiated from links that navigate to the item | ✓ VERIFIED | `fidelityAffordance` (format.ts) — `conversation-only` → `windowOnly:true`/`AppWindow` icon/"Show in …"; all other + unrecognised values → navigating/`ArrowUpRight`/"Open in …". `OpenInSource.svelte` renders icon→label→badge in fixed order, badge (`formatFidelity`) unchanged (3 raw values preserved). 8 unit tests pass including the unrecognised-value and empty-string fallback. |
| 3 | After an in-webspace search, matched terms are highlighted in the detail pane's rendered content across text/HTML/chat-transcript variants; stream unaffected | ✓ VERIFIED | Kernel: `highlightTerms`/`highlightTextNodes`/`highlightSanitizedFragment` (rendition.go) insert `<mark>` via `html.ParseFragment`/`html.Render` tree mutation strictly between sanitize and wrap; `?hl=` wired in `item.go`'s `renditionHandler`; `agent.go` passes `nil` (byte-identical `/agent/v1` output, by design). Client: `highlightText`/`highlightTerms` (format.ts) implement an identical rule for the text/media variants, wired into `DetailPane.svelte`'s `loadedTextBlock` and the html-branch `contentUrl(item.id, searchQuery)`. 10 kernel tests + 15 client tests pass, including no-re-sanitization, chat-class-allowlist survival, multi-byte safety, CSP/sandbox-unchanged assertions, and the element-boundary backstop. |
| 4 | Scrollbars app-wide are thin and theme-matched; the stream pane's scrollbar carries date markers reflecting visible chronology | ✓ VERIFIED | `web/src/app.css`'s scrollbar tokens (`--scrollbar-thumb`, `scrollbar-width: thin`, `::-webkit-scrollbar*`) confirmed byte-unchanged since before this phase (`git diff --quiet` against the phase's base commit). `dateMarkers` (format.ts) derives adaptively-thinned (day→week→month), 24px-floored markers from in-memory stream items; `StreamDateMarkers.svelte` renders a `pointer-events-none` overlay with `pointer-events-auto` tick hit-areas reusing the existing scrollbar-thumb tokens (no new colour), gated to render only over the stream (not search results, not the rendition iframe, not the overflow popover) via `+page.svelte`'s conditional. 15 unit tests pass covering the floor invariant, order fidelity and the UTC day boundary. |

**Score:** 3/4 roadmap success criteria fully verified; 1 partially verified (initial-load case correct, session-lifetime resize case broken)

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `web/src/lib/components/SourceChip.svelte` | merged per-instance affordance | ✓ VERIFIED | 118 lines; health dot, truncated name w/ `title`, `aria-pressed` toggle, hover/focus-revealed refresh w/ `stopPropagation`, tooltip incl. syncing branch |
| `web/src/lib/components/SourceHealthChip.svelte`, `SourceFilterChips.svelte` | deleted | ✓ VERIFIED | confirmed absent from working tree |
| `web/src/lib/components/ui/popover/{index.ts,popover.svelte,popover-trigger.svelte,popover-content.svelte}` | local wrapper over installed bits-ui | ✓ VERIFIED | present; `git diff --quiet -- web/package.json web/package-lock.json` succeeds — no new dependency |
| `web/src/lib/format.ts` | filter/tone/fit/fidelity/marker helpers | ✓ VERIFIED | `resolveSourceFilters`, `toggleSourceFilter`, `serializeSourceFilters`, `worstHealthTone`, `visibleChipCount`, `fidelityAffordance`, `dateMarkers`, `highlightTerms`, `highlightText` all present and exported |
| `web/src/lib/components/WebspaceHeader.svelte` | measured single-line row + overflow | ⚠️ ORPHANED WIRING | Row/overflow render correctly on initial load; `ResizeObserver` construct is present (satisfies grep-based acceptance criteria) but its `.observe()` calls never fire against real elements — dead wiring, see Gaps |
| `kernel/httpapi/rendition.go` | highlight tree-walk, mark stylesheet rules | ✓ VERIFIED | `highlightTerms`, `highlightTextNodes`, `highlightSanitizedFragment`; `mark`/`body mark` rules present w/ `#fbbf24`/`#020617` |
| `kernel/httpapi/item.go` | `?hl=` wiring | ✓ VERIFIED | reads `r.URL.Query().Get("hl")`, threads through `highlightTerms` into `sanitizeAndWrapRendition` |
| `kernel/httpapi/agent.go` | unhighlighted `/agent/v1` mirror | ✓ VERIFIED | passes `nil` terms with recorded scope-boundary comment |
| `web/src/lib/components/OpenInSource.svelte` | fidelity-differentiated affordance | ✓ VERIFIED | icon→label→badge order, `fidelityAffordance` + `formatFidelity` both called |
| `web/src/lib/components/StreamDateMarkers.svelte` | date-tick overlay | ✓ VERIFIED | `pointer-events-none` container, `pointer-events-auto` ticks, scrollbar-thumb token reuse |
| `web/src/lib/components/highlight.test.ts`, `fidelity.test.ts`, `markers.test.ts`, `sources.test.ts` | unit coverage | ✓ VERIFIED | all pass (97 tests across these 4 files alone; 173/173 across the whole `web` suite) |
| `docs/api.md` | `?hl=` documentation | ✓ VERIFIED | documents the parameter's bounds and its absence from `/agent/v1` |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `DetailPane.svelte` | `api.ts` | `contentUrl(item.id, searchQuery)` in html branch | ✓ WIRED | confirmed at line 181; media-branch image/PDF calls correctly left unmodified |
| `item.go` renditionHandler | `rendition.go` | `sanitizeAndWrapRendition(shape, fragment, terms)` | ✓ WIRED | confirmed |
| `+page.svelte` | `DetailPane.svelte` | `searchQuery` prop | ✓ WIRED | confirmed |
| `WebspaceHeader.svelte` | `SourceChip.svelte` | one chip per instance, inline + popover | ✓ WIRED | confirmed, same component reused unforked in both places |
| `+page.svelte` URL query | `format.ts` | `resolveSourceFilters` | ✓ WIRED | confirmed |
| `StreamList.svelte` | `format.ts` | `filterItemsBySource` | ✓ WIRED | confirmed, `Set<string>` signature |
| `OpenInSource.svelte` | `format.ts` | `fidelityAffordance` | ✓ WIRED | confirmed |
| `+page.svelte` | `StreamDateMarkers.svelte` | overlay as sibling of scroll region, gated | ✓ WIRED | confirmed, gated behind `!searchQuery.trim() && loadState === 'ready' && response` |
| `StreamDateMarkers.svelte` | `StreamRow.svelte` | `data-item-id` scroll target | ✓ WIRED | confirmed on both sides |
| `WebspaceHeader.svelte`'s `ResizeObserver` | `rowEl`/`measureEl`/`trailingEl`/`overflowTriggerMeasureEl` | `.observe()` calls in `onMount` | ✗ NOT_WIRED | `onMount` fires before any of these elements exist (they're bound only inside `{#if showSourceRows}`, which is false at `sourcesState: 'loading'`); every `if (xEl) observer.observe(xEl)` guard fails permanently; nothing re-attaches later |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| UI-07 | 06-02 | Header presents each source instance exactly once, one affordance, scales past 10 instances | ⚠️ PARTIAL | Chip merge, multi-select filtering and initial-load overflow are all solid and unit-tested; the "stays usable ... as instance count grows" half of the requirement is undermined by the dead `ResizeObserver` (see gap) |
| UI-08 | 06-03 | Deep-link fidelity visually differentiated | ✓ SATISFIED | `fidelityAffordance` + `OpenInSource.svelte`, unit-tested |
| UI-09 | 06-01 | Search-term highlighting in detail pane across content variants | ✓ SATISFIED | kernel + client highlighters, extensively unit-tested, sanitizer/CSP boundary provably untouched |
| UI-11 | 06-03 | Thin theme-matched scrollbars app-wide + stream date markers | ✓ SATISFIED | base scrollbar theming confirmed byte-unchanged; `dateMarkers`/`StreamDateMarkers.svelte` implemented and unit-tested |

No orphaned requirements — REQUIREMENTS.md's traceability table maps exactly UI-07, UI-08, UI-09, UI-11 to Phase 6, matching all 4 plan frontmatter `requirements:` declarations (06-01: UI-09; 06-02: UI-07; 06-03: UI-08, UI-11).

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `web/src/lib/components/WebspaceHeader.svelte` | 118-125 | Dead `ResizeObserver` wiring (elements unbound at `onMount` time) | 🛑 Blocker | Overflow computation goes stale after initial load; a resized window can silently clip a chip without showing it in the overflow popover — the exact silent-invisibility failure mode the phase's own prohibition rules out |
| `web/src/lib/components/WebspaceHeader.svelte` | 112-124 | `overflowTriggerMeasureEl` read by `measure()` but never added to the (non-functional) observer's watch list | ℹ️ Info | Same root cause as above; secondary once the observer attachment is fixed |
| `kernel/httpapi/rendition.go` / `web/src/lib/format.ts` | `highlightTerms` (~line 307 / ~490) | Term-count and minimum-length are bounded, but individual term *length* is not | ℹ️ Info | `docs/06-REVIEW.md` WR-02 — a very long single-token query is a self-inflicted slow-response risk only, on a loopback/unauthenticated route; low severity, accepted disposition already recorded in the phase's own threat register |
| `kernel/httpapi/item_test.go` / `agent_test.go` | — | No handler-level (HTTP-request) test asserts `?hl=` wiring or its absence on `/agent/v1` | ℹ️ Info | `06-REVIEW.md` IN-02 — the pure functions are thoroughly tested; the actual route wiring is untested at the handler boundary |

No `TBD`/`FIXME`/`XXX` debt markers found in any file this phase touched.

## Gaps Summary

Three of the four roadmap success criteria (UI-08, UI-09, UI-11) are fully, independently verified against the codebase — not just against SUMMARY.md's claims — with passing automated test suites (`go test ./...`, `make test`, `npm test` at 173/173, `npm run check` at 0 errors) and direct code reads confirming the sanitizer/CSP/sandbox/scrollbar-theming invariants are untouched.

Success Criterion 1 (the merged source chip, header deduplication) is real and well-built for the common case — the two-row header genuinely became one row of merged chips, multi-select filtering works and is thoroughly unit-tested, and the overflow popover computes correctly the moment source data first loads. But the mechanism specifically built to keep that computation correct as the environment changes — a `ResizeObserver` — is inert: it is constructed in `onMount` before the DOM elements it's supposed to watch exist, and nothing ever re-attaches it once they do. This is not a code-review suspicion; it was independently re-derived by reading `WebspaceHeader.svelte` line by line. The result is a header that looks and behaves correctly immediately after page load, then quietly stops adapting to window resizes or chip-width changes for the rest of the session — a real regression risk against the phase's own explicit prohibition that a source's degraded/failing state must never become invisible.

This is a scoped, well-understood, single-file fix (move `ResizeObserver` construction into a ref-driven `$effect`, per 06-REVIEW.md's WR-01 suggested fix) — not a design failure — but it is a functional gap that should be closed, not waived, before Phase 6 is considered fully done.

---

_Verified: 2026-08-06T22:30:00Z_
_Verifier: Claude (gsd-verifier)_
