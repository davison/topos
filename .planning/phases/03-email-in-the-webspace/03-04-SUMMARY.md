---
phase: 03-email-in-the-webspace
plan: 04
subsystem: ui
tags: [svelte, sveltekit, shadcn-svelte, search, vitest]

requires:
  - phase: 03-email-in-the-webspace
    provides: "03-03's GET /api/webspaces/{webspace}/search?q= route — bm25-ranked, snippet-carrying, webspace-scoped, capped at 50, STX/ETX snippet delimiters (kernel/index/store.go's SnippetOpen/SnippetClose)"
  - phase: 03-email-in-the-webspace
    provides: "03-01's StreamRow.svelte sender (group_label) rendering and SOURCE_DISPLAY_NAMES pattern in api.ts"
provides:
  - "SearchBox.svelte — debounced (300ms) search input mounted in WebspaceHeader, icon-only 'Clear search' control meeting the 44px touch-target floor, never disabled, no spinner of its own"
  - "SearchResults.svelte — renders idle/loading/error/empty/populated from the pure searchVariant(query, state, resultCount) decision, reusing StreamList's skeleton rows and StreamRow's row shape verbatim"
  - "api.ts: SearchResult/SearchResponse types, searchWebspace(), SNIPPET_OPEN/SNIPPET_CLOSE constants, a proton entry in SOURCE_DISPLAY_NAMES"
  - "format.ts: parseSnippet, searchVariant, searchCopy, noMatchesHeading — the pure, unit-tested decisions the search templates render from"
  - "StreamRow.svelte's new optional snippet prop, rendering parseSnippet's segments in the existing semibold weight in place of item.preview"
  - "+page.svelte's search wiring: searchQuery/searchState/searchResults state, a monotonically increasing request-sequence guard, and selectedItem resolving against the active result set"
affects: [email-plugin-ui, search, webspace-header]

tech-stack:
  added: ["shadcn-svelte input registry block (web/src/lib/components/ui/input)"]
  patterns:
    - "Pure decision function per template branch (searchVariant, parseSnippet) asserted in format.test.ts — no component-testing library in this project, so every branch a template picks between lives in format.ts"
    - "Locked UI-SPEC copy centralized as a frozen object (searchCopy) plus one interpolation helper (noMatchesHeading), consumed by components rather than duplicated as inline literals"
    - "Monotonically increasing request-sequence number captured before an await and compared after, to discard a stale response race in as-you-type search"

key-files:
  created:
    - web/src/lib/components/SearchBox.svelte
    - web/src/lib/components/SearchResults.svelte
    - web/src/lib/components/ui/input/input.svelte
    - web/src/lib/components/ui/input/index.ts
  modified:
    - web/src/lib/api.ts
    - web/src/lib/format.ts
    - web/src/lib/format.test.ts
    - web/src/lib/components/StreamRow.svelte
    - web/src/lib/components/WebspaceHeader.svelte
    - web/src/routes/w/[webspace]/+page.svelte

key-decisions:
  - "SearchBox keeps a local, uncommitted inputValue separate from the caller's query prop so typing feels instant while the debounce (300ms) gates the actual request; an $effect re-syncs inputValue whenever query changes externally (the webspace-change reset to ''), since the initializer alone only seeds the value once and would otherwise leave stale text visible after navigating to a different webspace"
  - "Search state (searchQuery/searchState/searchResults) lives in route component state, not the URL — the UI-SPEC specifies no URL persistence for it, unlike the existing ?source= filter param"
  - "parseSnippet validates strict open/close delimiter alternation up front; any malformed shape (unpaired delimiter, two opens in a row) degrades the ENTIRE snippet to one plain-text segment with delimiters stripped, rather than partially parsing — simpler and safer than attempting to salvage a malformed run"

patterns-established:
  - "A search-adjacent UI element reuses StreamRow verbatim by passing an optional snippet prop rather than introducing a parallel row component"

requirements-completed: [KERN-05]

coverage:
  - id: D1
    description: "A user can type a query into the webspace header's search box and get back ranked, clickable matches spanning every source; clicking a result opens the same detail pane a stream row opens"
    requirement: "KERN-05"
    verification:
      - kind: unit
        ref: "web/src/lib/format.test.ts#searchVariant returns populated for a non-empty query whose completed request returned at least one result"
        status: pass
      - kind: unit
        ref: "web/src/lib/format.test.ts#searchVariant state matrix (one assertion per row)"
        status: pass
    human_judgment: true
    rationale: "Cross-source ranked results, snippet highlighting, and detail-pane opening on click require visual/interactive confirmation in a running browser (the plan's own human-check step) beyond what a unit test over pure decision functions can prove."
  - id: D2
    description: "Every search state (idle, loading, error, empty, populated, partial snippet) renders its specified treatment with copy matching the UI-SPEC exactly"
    requirement: "KERN-05"
    verification:
      - kind: unit
        ref: "web/src/lib/format.test.ts#searchCopy matches the UI-SPEC Copywriting Contract character for character"
        status: pass
      - kind: unit
        ref: "web/src/lib/format.test.ts#noMatchesHeading interpolates the query verbatim between straight double quotes"
        status: pass
      - kind: unit
        ref: "web/src/lib/format.test.ts#parseSnippet (all 10 cases, including leading/trailing/whole-string/elision-adjacent/multi-byte)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Search adds no new interaction model: no second scroll region, no client-side re-ranking, no new row component, no raw-HTML rendering anywhere"
    requirement: "KERN-05"
    verification:
      - kind: other
        ref: "grep -c '{@html' web/src/lib/components/SearchResults.svelte and StreamRow.svelte both return 0; grep -c '.sort(' and 'filterItemsBySource' in SearchResults.svelte both return 0"
        status: pass
    human_judgment: false

duration: ~20min
completed: 2026-07-31
status: complete
---

# Phase 3 Plan 4: Search a webspace from the browser Summary

**A debounced search box in the webspace header and a results region that reuses StreamRow verbatim — cross-source, bm25-ranked, snippet-highlighted matches with zero new interaction model.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-07-31
- **Tasks:** 2
- **Files modified:** 10 (4 created, 6 modified)

## Accomplishments

- `SearchBox.svelte`: debounced (300ms) search input mounted in `WebspaceHeader` at the same row-tier as the source filter chips, rendered unconditionally (never gated behind `shouldShowSourceRows`, since searching the index doesn't depend on any source being reachable); icon-only "Clear search" control meeting the 44px touch-target floor via the same `size-11` `Button` pattern `SourceHealthChip`'s refresh control already uses; the input is never disabled and never carries a spinner of its own
- `SearchResults.svelte`: renders exactly one of idle/loading/error/empty/populated from `searchVariant`, reusing `StreamList`'s four-skeleton-row loading treatment and `StreamRow`'s exact row shape for populated results, with results rendered in the API's own bm25-rank order (no client sort, re-rank, or source filter)
- `api.ts`: `SearchResult` (extends `StreamItem` so it renders through `StreamRow` unchanged), `SearchResponse`, `searchWebspace()` against `GET /api/webspaces/{webspace}/search?q=`, `SNIPPET_OPEN`/`SNIPPET_CLOSE` matching `kernel/index/store.go`'s `SnippetOpen`/`SnippetClose` exactly, and a `proton` → `Proton Mail` entry in `SOURCE_DISPLAY_NAMES`
- `format.ts`: `parseSnippet` (STX/ETX-delimited snippet → ordered plain/matched segments, degrading any malformed delimiter run to one plain-text segment with delimiters stripped rather than throwing or leaking a control character), `searchVariant` (the single pure decision `SearchResults` renders from), `searchCopy` (frozen object holding the four UI-SPEC-locked strings) and `noMatchesHeading` (verbatim query interpolation) — all unit-tested, including edge cases for leading/trailing/whole-string/elision-adjacent matches and a surrogate-pair character adjacent to a delimiter
- `StreamRow.svelte`: new optional `snippet` prop — present and non-empty, renders `parseSnippet`'s segments (matched text in the existing semibold weight, never a new color) in place of `item.preview`; absent, behavior is byte-identical to Phase 1/2; present-but-empty, the preview region is omitted entirely (same degrade the existing `item.preview` guard already applies)
- `+page.svelte`: `searchQuery`/`searchState`/`searchResults` component state (deliberately not URL-persisted, per UI-SPEC), a monotonically increasing `searchRequestSeq` guard so a slower earlier response can never overwrite a faster later one, `SearchResults` rendered in place of `StreamList` inside the same stream pane whenever the trimmed query is non-empty (no second scroll region), and `selectedItem` now also resolves against the active result set so clicking a search result opens the same detail pane a stream row does
- Generated the shadcn-svelte `input` registry block — the single new primitive this phase introduces

## Task Commits

Each task was committed atomically:

1. **Task 1: Search a webspace from the browser and open a result** - `4619cf2` (feat)
2. **Task 2: Cover the search state matrix and prove the copy is exact** - `dd3ad2f` (test)

_No separate plan-metadata commit was made in this worktree — STATE.md/ROADMAP.md updates are owned by the orchestrator after all wave agents complete, per this plan's parallel-execution instructions._

## Files Created/Modified

- `web/src/lib/components/SearchBox.svelte` - Debounced search input with icon and clear control
- `web/src/lib/components/SearchResults.svelte` - Results list rendering loading/error/empty/populated states
- `web/src/lib/components/ui/input/input.svelte`, `web/src/lib/components/ui/input/index.ts` - Generated shadcn-svelte Input primitive
- `web/src/lib/api.ts` - `SearchResult`, `SearchResponse`, `searchWebspace()`, `SNIPPET_OPEN`/`SNIPPET_CLOSE`, `proton` display name
- `web/src/lib/format.ts` - `parseSnippet`, `searchVariant`, `searchCopy`, `noMatchesHeading`
- `web/src/lib/format.test.ts` - 24 new tests covering `parseSnippet`'s 10 cases, `searchVariant`'s full state matrix, and character-for-character copy assertions
- `web/src/lib/components/StreamRow.svelte` - New optional `snippet` prop rendering `parseSnippet`'s segments
- `web/src/lib/components/WebspaceHeader.svelte` - Mounts `SearchBox` unconditionally at the filter-chip row-tier
- `web/src/routes/w/[webspace]/+page.svelte` - Search state, request-sequence guard, `SearchResults`/`StreamList` toggle, extended `selectedItem` lookup

## Decisions Made

- `SearchBox` keeps a local `inputValue` separate from the caller's `query` prop for instant typing feedback, with an `$effect` re-syncing it whenever `query` changes externally — needed because the `$state(query)` initializer alone only seeds the value once, which would otherwise leave stale query text visible in the box after the route's webspace-change effect resets `searchQuery` to `''` on navigating to a different webspace.
- `parseSnippet` validates delimiter alternation up front (must strictly alternate open/close starting with open) and degrades the *entire* snippet to one plain-text segment (delimiters stripped) on any malformed shape, rather than attempting a partial parse — simpler to reason about and impossible to leak a delimiter character from, satisfying the plan's "never throw, never leak a delimiter" requirement.
- Task 1 shipped `SearchBox`/`SearchResults` with the four locked strings as inline literals (needed for that task's own acceptance grep); Task 2 introduced `searchCopy`/`noMatchesHeading` as their one definition and refactored both components to render from it, per the plan's own task split.

## Deviations from Plan

None — plan executed exactly as written. The `web/node_modules` directory didn't exist yet in this worktree; ran `npm install` (already-declared `package.json`/`package-lock.json`, no version changes) before `npx shadcn-svelte@latest add input` so the CLI's internal `svelte-kit sync` step could run.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required. This plan is client-side only and consumes the search route 03-03 already shipped.

## Next Phase Readiness

- Search UI is fully wired end to end against the live kernel search route; no blockers for any later phase.
- `docs/api.md`'s search route documentation (shipped in 03-03) and this plan's `api.ts`/`format.ts` additions are now the complete, consistent contract for any future search-adjacent UI work (e.g., a future agent-facing search surface, if ever added).

## Self-Check: PASSED

All files created/modified confirmed present on disk (`web/src/lib/api.ts`,
`web/src/lib/format.ts`, `web/src/lib/format.test.ts`,
`web/src/lib/components/SearchBox.svelte`,
`web/src/lib/components/SearchResults.svelte`,
`web/src/lib/components/StreamRow.svelte`,
`web/src/lib/components/WebspaceHeader.svelte`,
`web/src/routes/w/[webspace]/+page.svelte`,
`web/src/lib/components/ui/input/input.svelte`,
`web/src/lib/components/ui/input/index.ts`, this SUMMARY.md). Both task
commits confirmed present in `git log --oneline --all` (`4619cf2`, `dd3ad2f`).
`npm --prefix web run test` (69 tests) and `npm --prefix web run check`
(0 errors) both pass; `npm --prefix web run build && ./scripts/assert-stylesheet.sh`
exits 0.

---
*Phase: 03-email-in-the-webspace*
*Completed: 2026-07-31*
