---
phase: 02-two-sources-one-trustworthy-stream
plan: 03
subsystem: ui
tags: [sveltekit, svelte5, tailwind, shadcn-svelte, health-chips, source-filter, staleness]

requires:
  - phase: 02-two-sources-one-trustworthy-stream
    provides: "02-02: GET /api/sources, POST /api/sources/{name}/refresh, POST /api/sync — the kernel-side health merge and coordinator-backed refresh routes this plan's UI is written against"
  - phase: 02-two-sources-one-trustworthy-stream
    provides: "02-01: SilverBullet as the second source, ItemContent.available/unavailable_reason, the text/html iframe rendition pattern DetailPane already used"
provides:
  - "web/src/lib/components/SourceHealthChip.svelte + SourceFilterChips.svelte: per-source status chips and a single-select source filter row, composed from Phase 1's Badge/Tooltip/Button/Alert primitives, zero new shadcn blocks"
  - "web/src/lib/format.ts pure helpers: formatRelativeTime, healthTone, syncingSourceTypes, shouldShowSourceRows, resolveSourceFilter, filterItemsBySource, streamVariant, staleSourceTypes, detailPaneState — every branch a template picks between is unit-testable without mounting a component"
  - "web/src/lib/api.ts: SourceStatus/SourcesResponse/RefreshResult types + getSources/refreshSource/refreshAll/postJSON, against the live kernel/httpapi/sources.go shape"
  - "WebspaceHeader.svelte, StreamList.svelte, StreamRow.svelte, StreamEmpty.svelte, DetailPane.svelte, +page.svelte generalized to serve N sources: URL-persisted source filter, stale-row markers, unreachable/deleted detail-pane states, no hardcoded source name anywhere in this plan's files"
affects: [03, 04, kernel/sync (no kernel change this plan — pure frontend), any later source plugin (paperless/silverbullet's UI treatment now generalizes automatically to a third source once it's configured)]

tech-stack:
  added: []
  patterns:
    - "Pure decision helpers in format.ts (streamVariant, detailPaneState, healthTone, resolveSourceFilter) — every branch a Svelte template picks between is a plain function first, unit-tested, then rendered from; keeps 'exactly one state, never two, never none' assertable without a component-test harness"
    - "URL-as-state for the source filter (page.url.searchParams + goto(..., {replaceState:true, keepFocus:true, noScroll:true})) — client-side narrowing of an already-fetched response, no new backend parameter (A-UI-02)"
    - "Cached-preview-over-alert for staleness: the unreachable/deleted detail-pane branches render the StreamItem's own preview/thumbnail_url (already in memory from the stream fetch) under a non-destructive Alert, never a re-fetch and never a blank pane"
    - "bits-ui Tooltip's {#snippet child({props})} render-prop pattern to make a non-button element (a status dot) the tooltip trigger, avoiding a nested <button>-in-<button> when the stale marker/chip sits inside an already-interactive row"

key-files:
  created:
    - web/src/lib/components/SourceHealthChip.svelte
    - web/src/lib/components/SourceFilterChips.svelte
    - web/src/lib/components/sources.test.ts
    - web/src/lib/components/staleness.test.ts
  modified:
    - web/src/lib/api.ts (SourceStatus/SourcesResponse/RefreshResult/SourceRefreshResponse/SyncRefreshResponse, getSources/refreshSource/refreshAll/postJSON)
    - web/src/lib/format.ts (formatRelativeTime, healthTone, syncingSourceTypes, shouldShowSourceRows, resolveSourceFilter, filterItemsBySource, streamVariant, staleSourceTypes, detailPaneState)
    - web/src/lib/format.test.ts (formatRelativeTime + healthTone tables)
    - web/src/app.css (--success/--warning tokens, --color-success/--color-warning Tailwind mappings)
    - web/src/lib/components/WebspaceHeader.svelte (health-chip row + filter row + Refresh all)
    - web/src/lib/components/StreamList.svelte (streamVariant-driven rendering, staleness + display-name threading)
    - web/src/lib/components/StreamRow.svelte (stale marker)
    - web/src/lib/components/StreamEmpty.svelte (filtered-empty copy + generalized unfiltered copy)
    - web/src/lib/components/DetailPane.svelte (displayName/sourceReachable props, unreachable/deleted branches)
    - web/src/routes/w/[webspace]/+page.svelte (sources fetch/poll, filter state, refresh handlers, now also renders WebspaceHeader)
    - web/src/routes/+layout.svelte (WebspaceHeader removed — moved into +page.svelte)

key-decisions:
  - "RefreshResult's field shape follows the live kernel/httpapi/sources.go + docs/api.md exactly (name/source_type/status/item_count/error/coalesced/finished_unix, response envelopes {source: RefreshResult} and {sources: RefreshResult[]}), not PLAN.md's <interfaces> sketch (which named the field 'source', included a 'started_unix' the kernel never sends, and used 'result'/'results' as the wrapper keys) — the plan's own note designates 02-02-SUMMARY.md and the live kernel as authoritative over that sketch"
  - "WebspaceHeader moved from +layout.svelte into +page.svelte (not in PLAN.md's files_modified, added as a Rule 3 blocking-issue fix) — the header's new props (sources, sourcesState, selectedSource, refresh/filter handlers) are all owned by the page's per-webspace state, and SvelteKit's {@render children()} can't hand props back from a page to its parent layout; the header had to move to whichever component actually owns that state"
  - "healthTone checks last_status === '' (never synced) before checking reachable — a source that has never completed a sync renders neutral/unknown even if the live probe reports it currently unreachable, per docs/api.md's explicit 'render this as a neutral indicator, never as a green ok' framing for the never-synced case"
  - "The 'unknown' (never-synced) health-chip tooltip has no row in 02-UI-SPEC.md's Copywriting Contract (which only covers reachable/stale/unreachable) — added '{display_name} — not yet synced' as the minimal necessary copy so the tooltip is never empty (Rule 2)"
  - "'Cached preview' for the unreachable/deleted detail-pane states is the StreamItem's own preview/thumbnail_url (already in memory from the stream response), not a stashed previous ItemContent — the hybrid data model only ever stores metadata+preview locally, so there is no separate 'last good live content' to fall back to"
  - "streamVariant (format.ts) is the single pure decision StreamList renders from, folding 'sync-failure beats any empty state' and 'filtered vs unfiltered empty copy' into one unit-tested function rather than two separate inline $derived booleans"

requirements-completed: [UI-02, UI-05, UI-06]

coverage:
  - id: D1
    description: "The webspace header shows one health chip per configured source (dot + display name), tooltip with relative last-sync time and last error, and per-source + refresh-all controls that show a non-blocking syncing state"
    requirement: UI-06
    verification:
      - kind: unit
        ref: "web/src/lib/format.test.ts — healthTone/formatRelativeTime tables; web/src/lib/components/sources.test.ts — healthTone full matrix, syncingSourceTypes, shouldShowSourceRows"
        status: pass
      - kind: integration
        ref: "live GET /api/sources against the user's real instances returned both sources reachable:true with correct display_name; POST /api/sources/silverbullet/refresh returned 200 with item_count 17; POST /api/sources/does-not-exist/refresh returned 404"
        status: pass
      - kind: other
        ref: "grep of kernel/webui/build/_app/immutable/nodes/*.js after `make build` confirms the compiled SPA bundle contains 'Refresh all', 'currently unreachable', 'No longer available at', 'Switch back to All'"
        status: pass
    human_judgment: true
    rationale: "Visual chip layout, tooltip appearance/wrapping, and the spinning-icon syncing state were not screenshotted — no browser tooling was available in this session (same limitation noted in 02-01-SUMMARY.md). API-level and bundle-content evidence above confirms the feature is wired end-to-end; a human should confirm the rendered chips visually before/at ship."
  - id: D2
    description: "User can filter the stream to a single source and back to All; the choice persists in the URL across reload and deep links; an unrecognised filter value degrades to the unfiltered stream"
    requirement: UI-02
    verification:
      - kind: unit
        ref: "web/src/lib/components/sources.test.ts — resolveSourceFilter, filterItemsBySource, streamVariant (incl. sync-failure-over-empty-while-filtered, empty vs empty-filtered)"
        status: pass
    human_judgment: true
    rationale: "Click-through of the filter chips, URL query round-trip on reload, and the resulting visual list narrowing were not exercised in a browser in this session — covered at the pure-logic level by the tests above; a human should click through the chips at least once before ship."
  - id: D3
    description: "An item whose source is unreachable shows a stale marker in the stream and a non-destructive alert over its cached preview in the detail pane; an item confirmed gone at the source shows a distinct no-longer-available alert; both keep the cached title/date/labels/open-in-source affordance, never a blank pane"
    requirement: UI-05
    verification:
      - kind: unit
        ref: "web/src/lib/components/staleness.test.ts — detailPaneState's full matrix incl. deleted-over-unreachable precedence, staleSourceTypes derivation, order-preservation under filtering"
        status: pass
      - kind: integration
        ref: "live GET /api/items/{silverbullet id} returned content.available:true (loaded branch); both sources reachable:true during this session so the unreachable/deleted branches could not be exercised end-to-end against the live instance without deliberately stopping a service, which this session's live_environment_facts asked to avoid disrupting"
        status: pass
    human_judgment: true
    rationale: "The unreachable and deleted visual states (amber stream markers, the non-destructive alert overlay) were not exercised against a deliberately-stopped source in this session — proven correct at the pure-decision-function level (staleness.test.ts) but a human should verify the visual overlay by stopping SilverBullet, per the plan's own human-check step."
  - id: D4
    description: "No user-visible copy in this plan's components names a specific source system literally — every such string interpolates the caller-supplied display name"
    requirement: UI-05
    verification:
      - kind: other
        ref: "grep -rn 'paperless-ngx' across DetailPane.svelte, StreamEmpty.svelte, StreamRow.svelte, SourceHealthChip.svelte, SourceFilterChips.svelte, WebspaceHeader.svelte — zero matches; grep -rn '@html' DetailPane.svelte — zero matches (no raw-HTML directive introduced)"
        status: pass
    human_judgment: false

duration: ~55min
completed: 2026-07-28
status: complete
---

# Phase 2 Plan 3: The Stream You Can Trust Summary

**Per-source health chips (reachable/syncing/last-sync/last-error + refresh), a URL-persisted single-select source filter, and explicit stale/no-longer-available states in both the stream and the detail pane — composed entirely from Phase 1's installed shadcn-svelte primitives, zero new registry blocks.**

## Performance

- **Duration:** ~55 min (extensive prior-plan reconciliation reading, staged implementation, live verification against the user's real paperless-ngx + SilverBullet instances)
- **Completed:** 2026-07-28
- **Tasks:** 3 completed
- **Files modified:** 15 (4 new, 11 modified), excluding `.planning/`

## Accomplishments

- `SourceHealthChip.svelte` + `SourceFilterChips.svelte`: a per-source status dot/name/tooltip/refresh chip and an All-plus-one-per-source filter row, both composed from `Badge`/`Tooltip`/`Button`/`Alert` — no new shadcn-svelte block installed (confirmed: `git status` shows no change under `web/src/lib/components/ui/`).
- `format.ts` gained nine new pure, unit-tested decision helpers (`healthTone`, `formatRelativeTime`, `syncingSourceTypes`, `shouldShowSourceRows`, `resolveSourceFilter`, `filterItemsBySource`, `streamVariant`, `staleSourceTypes`, `detailPaneState`) — every branch `WebspaceHeader`, `StreamList`, and `DetailPane` pick between is a plain function first, so "exactly one state, never two, never none" is assertable without a component-test harness.
- `+page.svelte` fetches and polls `GET /api/sources` (stopping the poll once nothing is syncing), owns the URL-persisted source filter, and wires per-source/refresh-all handlers through `POST /api/sources/{name}/refresh` and `POST /api/sync` — all against the live shapes shipped in 02-02.
- `StreamList.svelte`'s sync-failure-before-empty ordering (Phase 1's core invariant) now flows through the single `streamVariant` decision, extended with the filtered-vs-unfiltered empty copy, without weakening the "a filter can never mask a sync failure" guarantee.
- `DetailPane.svelte` generalizes from Phase 1's single-source, hardcoded-copy pane to a caller-supplied `displayName`/`sourceReachable` contract with two new non-destructive alert branches (source unreachable, item no-longer-available) rendered over the item's own cached preview — never a blank pane.
- Verified live against the user's real instances: `GET /api/sources` returned both sources `reachable:true` with correct plugin-reported `display_name`; `POST /api/sources/silverbullet/refresh` returned `200` with `item_count: 17`; an unconfigured refresh returned `404`; the compiled SPA bundle (post-`make build`) contains the new Copywriting Contract strings; `grep` confirms zero hardcoded "paperless-ngx" strings and zero `@html` directives in any file this plan touched.

## Task Commits

1. **Task 1: The header tells you which sources exist, whether they are up, and lets you refresh them** - `85cc931` (feat)
2. **Task 2: Filter the stream to one source and back, and say so when the filtered view is empty** - `cd64f5c` (feat)
3. **Task 3: Stale and unavailable items say so, in the stream and in the detail pane** - `03760de` (feat)

_Note on commit granularity: `format.ts`'s full set of nine pure helpers, and `WebspaceHeader.svelte`/`+page.svelte`'s full prop contracts, landed together in the Task 1 commit rather than growing incrementally per task, because `StreamList.svelte`'s and `DetailPane.svelte`'s declared Svelte prop types are closed object literals — `+page.svelte` cannot pass `StreamList` a `staleSourceTypes` prop it doesn't yet declare without `svelte-check` failing, and vice versa. The three files are load-bearing on each other's final prop shape. `npm run check`/`test`/`build` were run once against the plan's complete, final tree (all pass) rather than after each individual task's slice, consistent with how 02-01-SUMMARY.md and 02-02-SUMMARY.md documented the same tradeoff for their own cross-cutting, tightly-coupled changes. Each commit's message and file set otherwise tracks its task's own `<files>` list and `<action>` text as closely as the coupling allows.

## Files Created/Modified

- `web/src/lib/components/SourceHealthChip.svelte` - per-source dot/name/tooltip/refresh chip
- `web/src/lib/components/SourceFilterChips.svelte` - All + one chip per source, single-select
- `web/src/lib/components/sources.test.ts` - healthTone/formatRelativeTime/syncingSourceTypes/shouldShowSourceRows/resolveSourceFilter/filterItemsBySource/streamVariant tests
- `web/src/lib/components/staleness.test.ts` - detailPaneState/staleSourceTypes/order-preservation tests
- `web/src/lib/api.ts` - SourceStatus/SourcesResponse/RefreshResult(+response envelopes), getSources/refreshSource/refreshAll/postJSON
- `web/src/lib/format.ts` - the nine new pure helpers listed above
- `web/src/lib/format.test.ts` - formatRelativeTime + healthTone tables
- `web/src/app.css` - `--success`/`--warning` tokens + `--color-success`/`--color-warning` Tailwind mappings
- `web/src/lib/components/WebspaceHeader.svelte` - health-chip row, filter row, Refresh all
- `web/src/lib/components/StreamList.svelte` - streamVariant-driven rendering, stale/display-name threading
- `web/src/lib/components/StreamRow.svelte` - stale marker (warning-tone dot + tooltip)
- `web/src/lib/components/StreamEmpty.svelte` - filtered-empty copy + generalized unfiltered copy
- `web/src/lib/components/DetailPane.svelte` - displayName/sourceReachable props, unreachable/deleted branches
- `web/src/routes/w/[webspace]/+page.svelte` - sources fetch/poll, filter state, refresh handlers, now renders WebspaceHeader
- `web/src/routes/+layout.svelte` - WebspaceHeader removed (moved into +page.svelte)

## Decisions Made

See `key-decisions` in the frontmatter above for the full list with rationale. Summary:
- `RefreshResult`'s TypeScript shape follows the live kernel code/docs, not PLAN.md's `<interfaces>` sketch (field names, response envelope keys, and a `started_unix` the kernel doesn't send all differ).
- `WebspaceHeader` moved from `+layout.svelte` into `+page.svelte` — a structural fix required because the header's new props are owned by page-level state a layout can't receive back from its own children.
- `healthTone` treats "never synced" as taking precedence over live reachability, per docs/api.md's explicit framing.
- The never-synced tooltip copy and the "cached preview = StreamItem.preview/thumbnail_url" interpretation both fill gaps 02-UI-SPEC.md's Copywriting Contract left implicit.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] WebspaceHeader moved from `+layout.svelte` into `+page.svelte`**
- **Found during:** Task 1, while wiring `WebspaceHeader`'s new interactive props
- **Issue:** `+layout.svelte` renders `WebspaceHeader` as a sibling of `{@render children()}` (i.e. `+page.svelte`); the header's new props (`sources`, `sourcesState`, `selectedSource`, the refresh/filter handlers) are all owned by per-webspace state that only `+page.svelte` computes. SvelteKit has no mechanism for a page to pass props back up into the layout that renders it — the header could not receive this state from where it was mounted.
- **Fix:** Removed `WebspaceHeader`'s render from `+layout.svelte` (kept the `flex h-screen flex-col` / `min-h-0 flex-1` wrapper unchanged) and moved the header render into `+page.svelte`'s own template, which now owns the full flex-column layout for its route.
- **Files modified:** `web/src/routes/+layout.svelte`, `web/src/routes/w/[webspace]/+page.svelte`
- **Verification:** `npm run check`/`test`/`build` all pass; live `curl` of `/w/house-move` returned `200`.
- **Committed in:** `85cc931` (Task 1 commit)

**2. [Rule 1 - Bug] `StreamEmpty.svelte`'s unfiltered body copy still hardcoded "paperless-ngx documents"**
- **Found during:** Task 2, while adding the filtered-empty variant
- **Issue:** Phase 1's unfiltered empty-state body read "No paperless-ngx documents match this webspace's keywords yet..." — a leftover hardcoded source name (the same Pitfall 3 category 02-01 already fixed elsewhere) that 02-UI-SPEC.md's Copywriting Contract generalizes to "No items match this webspace's keywords yet...".
- **Fix:** Updated the unfiltered body to the UI-SPEC's exact generalized string, alongside adding the new filtered-empty variant.
- **Files modified:** `web/src/lib/components/StreamEmpty.svelte`
- **Verification:** `grep -rn 'paperless-ngx' web/src/lib/components/StreamEmpty.svelte` returns no match.
- **Committed in:** `cd64f5c` (Task 2 commit)

**3. [Rule 2 - Missing critical functionality] Tooltip copy for the never-synced ("unknown") health-chip state**
- **Found during:** Task 1, implementing `SourceHealthChip`'s tooltip
- **Issue:** 02-UI-SPEC.md's Copywriting Contract only defines tooltip copy for the reachable/stale/unreachable tones — a source that has never completed a sync (`last_status: ''`) has no contracted tooltip string, and a hover-only diagnostic surface cannot render empty text.
- **Fix:** Added `"{display_name} — not yet synced"` as the minimal, sensible copy for that tone.
- **Files modified:** `web/src/lib/components/SourceHealthChip.svelte`
- **Verification:** `healthTone` test matrix confirms the unknown tone is reachable via `last_status: ''`; visual tooltip text was not screenshotted (see coverage D1's `human_judgment` rationale).
- **Committed in:** `85cc931` (Task 1 commit)

---

**Total deviations:** 3 auto-fixed (1 Rule 3, 1 Rule 1, 1 Rule 2)
**Impact on plan:** All three were necessary for the plan's own acceptance criteria and truths to be satisfiable — the layout restructure is required for the header to receive its declared props at all, the empty-state fix closes a gap the plan's own must-have truth ("no user-visible copy names a specific source system...") already covered, and the tooltip copy fills a genuine hole in the Copywriting Contract. No scope creep beyond what the plan's own files list already covered (the layout file is the one addition, and it's a two-line removal, not new functionality).

## Issues Encountered

None beyond the three deviations above. The main practical challenge was reconciling `PLAN.md`'s `<interfaces>` sketch for the refresh endpoints against the actually-shipped `kernel/httpapi/sources.go`/`docs/api.md` shape from 02-02 — resolved by treating the live kernel code and docs as authoritative, per the plan's own instruction and this phase's established precedent (02-01/02-02 SUMMARYs did the same for their own interface sketches).

## User Setup Required

None beyond what 02-01-PLAN.md's `user_setup` and 02-02's sync-interval config already covered. This plan added no new required configuration — it is a pure frontend change against endpoints 02-02 already shipped.

## Next Phase Readiness

- `format.ts`'s pure helpers (`healthTone`, `streamVariant`, `detailPaneState`, `staleSourceTypes`) are the general-purpose decision layer any future source (Phase 3's IMAP plugin onward) inherits for free — a third configured source automatically gets a health chip, a filter chip, and correct stale-marker/detail-pane behavior with zero UI code changes, since every one of these helpers operates on the `SourceStatus[]`/`StreamItem[]` shape, never a specific source name.
- **Outstanding for a human before this plan is fully signed off:** no browser tooling was available in this session (same limitation as 02-01), so the plan's four `<human-check>` verification steps (chip colors/tooltip text, filter click-through + URL persistence, stale marker + unreachable alert with a deliberately-stopped source, deleted-item alert) were not visually confirmed — only proven at the API/unit-test/bundle-content level (see `coverage` block above for exactly what was and wasn't exercised). Port 7777 was left free and both plugin subprocesses were cleaned up at the end of this session.
- No blockers identified for 02-04 or Phase 3.

---
*Phase: 02-two-sources-one-trustworthy-stream*
*Completed: 2026-07-28*

## Self-Check: PASSED

All 4 claimed new files exist on disk and all three task commits (`85cc931`, `cd64f5c`, `03760de`) are present in `git log`.
