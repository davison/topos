---
phase: 06-ui-scalable-source-surface
verified: 2026-08-06T22:26:20Z
status: human_needed
score: 4/4 roadmap success criteria verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 3/4 roadmap success criteria fully verified (1 partial)
  gaps_closed:
    - "Success Criterion 1 — the design remains usable at 10+ instances (overflow, grouping, or collapse; no unbounded chip rows), sustained across the component's lifetime"
  gaps_remaining: []
  regressions: []
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
  - test: "With a webspace loaded and the chip row rendered, narrow the browser window steadily. Confirm chips move off the row into the '+N' trigger as space runs out and the trigger's count rises; widen the window and confirm they return inline and the trigger's count falls or the trigger disappears; confirm that at no width is a chip clipped at the row's trailing edge with no trigger showing, and that 'Clear filters' and 'Refresh all' stay on the row throughout. Then make one source hidden behind the fold unreachable and confirm the trigger's dot turns the destructive tone at that narrowed width."
    expected: "Single-line row at any instance count and at any window width, including after a post-load resize; overflow popover reachable in two interactions; worst-of health tone surfaced on the trigger at all times."
    why_human: "Real-DOM measurement, ResizeObserver firing, and popover interaction require a live browser. Code-level fix verified: `observeResize` (web/src/lib/resize-observer.ts) is now wired through a ref-driven `$effect` in WebspaceHeader.svelte, proven by 6 behavioral unit tests of the attachment/re-attachment semantics plus a 5-case comment-stripped source-scan guard proving the component wiring shape (all 4 measured elements named, teardown returned, no dead mount-hook, no second construction site). This is the gap 06-VERIFICATION.md previously recorded as PARTIAL; the mechanism is now demonstrably correct, but a live window-resize click-through has not yet been performed."
  - test: "Open a Signal conversation item and confirm its button reads `Show in {source}` with a window icon and an explanatory hover title; open a paperless or email item and confirm its button reads `Open in {source}` with a navigate icon; confirm the small fidelity badge still shows the raw enum value in all three cases."
    expected: "Two-class icon/verb/title split, badge unchanged (3 raw values)."
    why_human: "Visual icon rendering and title-attribute hover behavior in a live browser; the `fidelityAffordance` mapping itself is proven by 8 passing unit tests."
  - test: "Open a webspace whose stream spans several dates. Confirm thin ticks appear alongside the stream scrollbar, hovering one shows its date, clicking one jumps that date's first row to the top of the pane, the scrollbar thumb itself still drags, and the rows underneath the overlay still click. Then run a search and confirm the ticks disappear while search results are showing."
    expected: "Clickable, tooltipped date ticks; native scrollbar and row interactivity undisturbed; markers hidden during search."
    why_human: "Pointer-event pass-through, click-to-scroll behavior, and overlay-vs-native-scrollbar interaction require a live browser session. The `dateMarkers` derivation itself (position formula, adaptive thinning, 24px floor, UTC boundary) is proven by 15 passing unit tests."
---

# Phase 6: UI Scalable Source Surface Verification Report

**Phase Goal:** The webspace header presents many source instances without duplication — health and filtering combined into one affordance per source — and the accumulated UI polish lands: deep-link fidelity differentiation, search-term highlighting in the detail pane, and themed scrollbars with date markers
**Verified:** 2026-08-06T22:26:20Z
**Status:** human_needed
**Re-verification:** Yes — after gap closure (06-04-PLAN.md / 06-04-SUMMARY.md, this session)

## Goal Achievement

### Observable Truths

| # | Truth (Roadmap Success Criterion) | Status | Evidence |
|---|---|---|---|
| 1 | Each source instance appears exactly once in the header — a single chip combining health, filter toggle and refresh — usable at 10+ instances (overflow/grouping/collapse; no unbounded rows), sustained across the component's lifetime | ✓ VERIFIED | `SourceChip.svelte` merges health dot + name + `aria-pressed` filter toggle + hover/focus-revealed refresh (confirmed by direct read, unchanged since prior verification); `SourceHealthChip.svelte`/`SourceFilterChips.svelte` confirmed absent from the tree. Overflow: `WebspaceHeader.svelte` now attaches its `ResizeObserver` through `observeResize` (`web/src/lib/resize-observer.ts`) inside a ref-driven `$effect` that reads all four measured elements (`rowEl`, `measureEl`, `trailingEl`, `overflowTriggerMeasureEl`) synchronously — confirmed by direct line-by-line read (not SUMMARY claims). The prior defect (observer constructed in a one-shot `onMount` before the elements existed, so every `observe()` call was skipped forever) is gone: the mount-hook block and its `onMount` import are fully removed. `observeResize` itself is behaviorally proven by 6 unit tests (bind/unbind/callback-routing/idempotent-teardown/re-attachment-after-teardown), and the component's wiring shape is proven by a 5-case comment-stripped source-scan guard (not a raw grep — the prior defect specifically satisfied a raw grep while being dead) asserting the effect returns `observeResize(...)`'s result and names all four elements. |
| 2 | Items whose "open in source" link can only raise the source app's window are visually differentiated from links that navigate to the item | ✓ VERIFIED | `fidelityAffordance` (format.ts) — `conversation-only` → `windowOnly:true`/`AppWindow` icon/"Show in …"; all other + unrecognised values → navigating/`ArrowUpRight`/"Open in …". `OpenInSource.svelte` renders icon→label→badge in fixed order, badge (`formatFidelity`) unchanged (3 raw values preserved). 8 unit tests pass including the unrecognised-value and empty-string fallback. Unchanged since prior verification; re-confirmed present and unbroken by this session's file-scoped diff. |
| 3 | After an in-webspace search, matched terms are highlighted in the detail pane's rendered content across text/HTML/chat-transcript variants; stream unaffected | ✓ VERIFIED | Kernel: `highlightTerms`/`highlightTextNodes`/`highlightSanitizedFragment` (rendition.go) insert `<mark>` via `html.ParseFragment`/`html.Render` tree mutation strictly between sanitize and wrap; `?hl=` wired in `item.go`'s `renditionHandler` (`r.URL.Query().Get("hl")` → `highlightTerms` → `sanitizeAndWrapRendition`); `agent.go` passes `nil` (confirmed at line 350 — byte-identical `/agent/v1` output, by design, with a recorded scope-boundary comment). Client: `highlightText`/`highlightTerms` (format.ts) implement an identical rule for the text/media variants, wired into `DetailPane.svelte`. `go test ./kernel/...` passes (all packages ok); client suite passes. Unchanged since prior verification. |
| 4 | Scrollbars app-wide are thin and theme-matched; the stream pane's scrollbar carries date markers reflecting visible chronology | ✓ VERIFIED | `web/src/app.css`'s scrollbar tokens (`--scrollbar-thumb`, `scrollbar-width: thin`, `::-webkit-scrollbar*`) re-confirmed present and unchanged this session (`git diff --quiet -- web/src/app.css` succeeds). `dateMarkers` (format.ts) derives adaptively-thinned, 24px-floored markers; `StreamDateMarkers.svelte` renders a `pointer-events-none` overlay with `pointer-events-auto` tick hit-areas. Unchanged since prior verification. |

**Score:** 4/4 roadmap success criteria verified (previously 3/4 fully verified, 1 partial — the partial is now closed)

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `web/src/lib/components/SourceChip.svelte` | merged per-instance affordance | ✓ VERIFIED | unchanged since prior verification; re-confirmed present |
| `web/src/lib/components/SourceHealthChip.svelte`, `SourceFilterChips.svelte` | deleted | ✓ VERIFIED | confirmed absent from working tree |
| `web/src/lib/components/ui/popover/*` | local wrapper over installed bits-ui | ✓ VERIFIED | present; no new dependency (lockfile diff clean) |
| `web/src/lib/format.ts` | filter/tone/fit/fidelity/marker helpers | ✓ VERIFIED | all exports confirmed present (`resolveSourceFilters`, `worstHealthTone`, `visibleChipCount`, `fidelityAffordance`, `dateMarkers`, `highlightTerms`, `highlightText`) |
| `web/src/lib/components/WebspaceHeader.svelte` | measured single-line row + working overflow, session-lifetime | ✓ VERIFIED | Row/overflow render correctly at initial load (unchanged); `ResizeObserver` attachment now moved into a ref-driven `$effect` calling `observeResize([rowEl, measureEl, trailingEl, overflowTriggerMeasureEl], measure)` — confirmed by direct read of lines 89–133; dead mount-hook block and its `onMount` import fully removed |
| `web/src/lib/resize-observer.ts` (new, this gap-closure plan) | injectable, testable attachment helper | ✓ VERIFIED | 63 lines; exports `observeResize`, `ResizeObserverLike`, `CreateResizeObserver`; is the sole non-test file under `web/src/` constructing `new ResizeObserver` (confirmed: `grep -rl 'new ResizeObserver' --exclude='*.test.ts' src/` returns exactly this one path) |
| `web/src/lib/resize-observer.test.ts` (new) | behavioral + structural coverage of the attachment fix | ✓ VERIFIED | 295 lines, 11 test cases across two describe blocks (6 behavioral on `observeResize`, 5 structural comment-stripped source-scan guard on the component); all pass |
| `kernel/httpapi/rendition.go` | highlight tree-walk, mark stylesheet rules | ✓ VERIFIED | unchanged since prior verification; re-confirmed |
| `kernel/httpapi/item.go` | `?hl=` wiring | ✓ VERIFIED | unchanged since prior verification; re-confirmed |
| `kernel/httpapi/agent.go` | unhighlighted `/agent/v1` mirror | ✓ VERIFIED | unchanged since prior verification; re-confirmed passes `nil` terms |
| `web/src/lib/components/OpenInSource.svelte` | fidelity-differentiated affordance | ✓ VERIFIED | unchanged since prior verification; re-confirmed |
| `web/src/lib/components/StreamDateMarkers.svelte` | date-tick overlay | ✓ VERIFIED | unchanged since prior verification; re-confirmed |
| Unit test files (`highlight.test.ts`, `fidelity.test.ts`, `markers.test.ts`, `sources.test.ts`, `resize-observer.test.ts`) | unit coverage | ✓ VERIFIED | `npm test` — 11 test files, 184/184 passing (up from 173/173 pre-gap-closure); `npm run check` — 0 errors |
| `docs/api.md` | `?hl=` documentation | ✓ VERIFIED | unchanged since prior verification; re-confirmed |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `DetailPane.svelte` | `api.ts` | `contentUrl(item.id, searchQuery)` in html branch | ✓ WIRED | unchanged, re-confirmed |
| `item.go` renditionHandler | `rendition.go` | `sanitizeAndWrapRendition(shape, fragment, terms)` | ✓ WIRED | unchanged, re-confirmed |
| `+page.svelte` | `DetailPane.svelte` | `searchQuery` prop | ✓ WIRED | unchanged, re-confirmed |
| `WebspaceHeader.svelte` | `SourceChip.svelte` | one chip per instance, inline + popover | ✓ WIRED | unchanged, re-confirmed |
| `+page.svelte` URL query | `format.ts` | `resolveSourceFilters` | ✓ WIRED | unchanged, re-confirmed |
| `StreamList.svelte` | `format.ts` | `filterItemsBySource` | ✓ WIRED | unchanged, re-confirmed |
| `OpenInSource.svelte` | `format.ts` | `fidelityAffordance` | ✓ WIRED | unchanged, re-confirmed |
| `+page.svelte` | `StreamDateMarkers.svelte` | overlay as sibling of scroll region, gated | ✓ WIRED | unchanged, re-confirmed |
| `StreamDateMarkers.svelte` | `StreamRow.svelte` | `data-item-id` scroll target | ✓ WIRED | unchanged, re-confirmed |
| `WebspaceHeader.svelte`'s `$effect` | `web/src/lib/resize-observer.ts`'s `observeResize` | ref-driven effect call, teardown propagated as cleanup | ✓ WIRED | **Fixed this session.** Confirmed by direct read (WebspaceHeader.svelte:131-133: `$effect(() => observeResize([rowEl, measureEl, trailingEl, overflowTriggerMeasureEl], measure))`) and by the source-scan guard's 5 passing assertions (non-empty stripped script; effect wraps `observeResize(...)` and returns its result — an expression-body arrow, satisfying the guard's accepted-shapes check; call names all four elements; no `onMount` reference remains; no second `new ResizeObserver` construction site in the component; markup still binds all four via `bind:this`) |
| `observeResize` | `rowEl`/`measureEl`/`trailingEl`/`overflowTriggerMeasureEl` | `.observe()` per bound target, none for an unbound one | ✓ WIRED | **Fixed this session.** Previously `NOT_WIRED` (`onMount` fired before any of these elements existed). Now: reading the four `$state` refs synchronously inside the `$effect` body registers them as reactive dependencies, so the effect re-runs (and `observeResize` re-attaches) each time a ref binds or rebinds — this is standard, documented Svelte 5 rune dependency-tracking behavior, not bespoke logic. The re-attachment path itself (teardown → new elements → re-observe) is directly exercised by `resize-observer.test.ts`'s "re-observes a fresh set of elements when called again after teardown" case |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| `observeResize` constructs nothing when every target is unbound (the exact failure state of the pre-fix code) | `cd web && npx vitest run -t "constructs no observer at all when every target is still unbound"` (subsumed by full run below) | pass | ✓ PASS |
| `observeResize` re-attaches after a teardown with a new element set | same file, "re-observes a fresh set of elements when called again after teardown" | pass | ✓ PASS |
| Full client suite | `cd web && npm test` (run once, full output captured) | `Test Files 11 passed (11)` / `Tests 184 passed (184)` | ✓ PASS |
| Full client typecheck | `cd web && npm run check` | `COMPLETED 761 FILES 0 ERRORS 1 WARNINGS` (warning is pre-existing, in `SearchBox.svelte`, unrelated to this phase) | ✓ PASS |
| Full kernel suite | `go build ./... && go test ./kernel/...` | all packages `ok` | ✓ PASS |
| Tree-wide uniqueness gate: only one production file constructs a real observer | `cd web && test "$(grep -rl 'new ResizeObserver' --exclude='*.test.ts' src/)" = "src/lib/resize-observer.ts"` | exit 0 | ✓ PASS |
| Lockfile unchanged | `git diff --quiet -- web/package.json web/package-lock.json` | exit 0 | ✓ PASS |
| Scope check: only the 3 files declared in the gap-closure plan changed since the last verification | `git diff --stat` between last-verification commit and HEAD, restricted to `web/`, `kernel/`, `docs/api.md` | `WebspaceHeader.svelte`, `resize-observer.ts`, `resize-observer.test.ts` only | ✓ PASS — no unrelated scope creep |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| UI-07 | 06-02, closed by 06-04 | Header presents each source instance exactly once, one affordance, scales past 10 instances | ✓ SATISFIED | Chip merge, multi-select filtering and overflow popover computation are correct both at initial load and across the session (post-resize) — the dead `ResizeObserver` wiring that previously undermined the session-lifetime half is fixed and behaviorally tested |
| UI-08 | 06-03 | Deep-link fidelity visually differentiated | ✓ SATISFIED | `fidelityAffordance` + `OpenInSource.svelte`, unit-tested; unchanged since prior verification |
| UI-09 | 06-01 | Search-term highlighting in detail pane across content variants | ✓ SATISFIED | kernel + client highlighters, extensively unit-tested; unchanged since prior verification |
| UI-11 | 06-03 | Thin theme-matched scrollbars app-wide + stream date markers | ✓ SATISFIED | base scrollbar theming confirmed byte-unchanged; `dateMarkers`/`StreamDateMarkers.svelte` implemented and unit-tested; unchanged since prior verification |

No orphaned requirements — REQUIREMENTS.md's traceability table maps exactly UI-07, UI-08, UI-09, UI-11 to Phase 6, matching all plan frontmatter `requirements:` declarations (06-01: UI-09; 06-02: UI-07; 06-03: UI-08, UI-11; 06-04 gap-closure: UI-07, re-declared for the same requirement it closes, not a new one).

Note: REQUIREMENTS.md's own checkbox/traceability bookkeeping (lines 42, 99-105) is currently stale relative to this finding — UI-07's checkbox reads `[x]` while its traceability-table row still reads "Gaps Found," and UI-08/09/11 both read `[ ]`/"Gaps Found." This is a documentation-sync artifact from the phase's git history (a prior "revert premature Complete requirements" commit), not a code defect; it is not part of this verification's scope to edit, but is flagged here since the orchestrator will need to reconcile it once this phase's status is accepted.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `kernel/httpapi/rendition.go` / `web/src/lib/format.ts` | `highlightTerms` (~line 307 / ~490) | Term-count and minimum-length are bounded, but individual term *length* is not | ℹ️ Info | `06-REVIEW.md` WR-02 — self-inflicted slow-response risk only, on a loopback/unauthenticated route; accepted disposition already recorded in the phase's threat register. Carried forward, unchanged, out of scope for the gap-closure plan |
| `kernel/httpapi/item_test.go` / `agent_test.go` | — | No handler-level (HTTP-request) test asserts `?hl=` wiring or its absence on `/agent/v1` | ℹ️ Info | `06-REVIEW.md` IN-02 — carried forward, unchanged, explicitly out of scope for the gap-closure plan |

No `TBD`/`FIXME`/`XXX` debt markers found in any file this phase (including the gap-closure plan) touched. The previously-recorded blocker (dead `ResizeObserver` wiring, `WebspaceHeader.svelte:118-125`) is resolved — the mount-hook block cited in the prior report's anti-pattern table no longer exists in the file.

## Gaps Summary

All four roadmap success criteria are now fully, independently verified against the codebase — not just against SUMMARY.md's claims. This is a re-verification following gap-closure plan 06-04, which addressed the single gap recorded in the prior VERIFICATION.md (2026-08-06T22:30:00Z, `gaps_found`): the chip row's `ResizeObserver` was constructed in a one-shot `onMount` hook that fired before any of the four measured DOM elements existed, so every guarded `observe()` call was silently skipped for the component's entire lifetime, and the overflow computation went stale immediately after the one correct measurement pass at initial load.

The fix extracts attachment into an injectable `web/src/lib/resize-observer.ts` module (`observeResize`) and rewires `WebspaceHeader.svelte` to call it from inside a ref-driven `$effect` that reads all four measured element refs (`rowEl`, `measureEl`, `trailingEl`, `overflowTriggerMeasureEl`) synchronously. Because these are Svelte 5 `$state` variables, reading them inside the effect body registers them as reactive dependencies — the effect now genuinely re-runs (and re-attaches the observer) whenever a ref binds or rebinds, closing both the original timing bug and IN-01 (the overflow-trigger measurement clone, previously never observed at all).

This fix is proven at three independent levels: (1) 6 behavioral unit tests of `observeResize` itself, directly exercising the bind/unbind/callback-routing/idempotent-teardown/re-attachment-after-teardown semantics that constitute the state transition; (2) a 5-case comment-stripped source-scan guard proving the component's wiring shape (not a raw grep — the prior defect specifically satisfied a raw-grep-style check while being non-functional, and this plan's own test explicitly guards against a repeat by asserting against comment-stripped source and requiring the effect's teardown be the attachment's actual return value); and (3) the full client (184/184) and kernel test suites, `npm run check` (0 errors), the tree-wide uniqueness gate (the helper is the sole production site constructing a real observer), and a lockfile-unchanged gate, all passing. A scope check confirms only the three files declared in the gap-closure plan changed since the prior verification — no unrelated regression risk was introduced.

What remains is exactly what remained after the prior verification for the phase's other three success criteria: live-browser human confirmation of visual/interactive behavior that a node-environment test cannot see (post-load window-resize chip reflow, search-term highlight rendering inside the CSP-opaque rendition iframe, fidelity icon/title rendering, and date-marker click/hover/pointer-passthrough behavior). None of these are code gaps — they are the standard end-of-phase UAT this project's `human_verify_mode: end-of-phase` configuration defers by design, harvested here from each plan's `<human-check>` blocks.

---

_Verified: 2026-08-06T22:26:20Z_
_Verifier: Claude (gsd-verifier)_

## Acknowledged Gaps

- `api-coverage.verify-pre` gate waived by user (2026-08-06): detection signals `{integration, api}` trace to Phase 6's own kernel HTTP API (`docs/api.md`, `kernel/httpapi/*`) — UI work against the project's internal API, not an external-API integration. No COVERAGE.md required for this phase.
