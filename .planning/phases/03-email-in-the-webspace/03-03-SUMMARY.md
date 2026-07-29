---
phase: 03-email-in-the-webspace
plan: 03
subsystem: search
tags: [sqlite, fts5, bm25, go-chi, kernel-httpapi]

requires:
  - phase: 01-first-webspace-end-to-end
    provides: "kernel/index's items rowid table, deliberately shaped since Phase 1 for an external-content FTS5 addition with no migration"
  - phase: 01-first-webspace-end-to-end
    provides: "kernel/httpapi's store-only handler pattern (StreamHandler) that search.go's SearchHandler mirrors for the same structural plugin-isolation guarantee"
provides:
  - "items_fts external-content FTS5 virtual table + items_ai/items_ad/items_au sync triggers over items(title, preview)"
  - "Store.Open first-creation backfill so a pre-existing index file (items synced before this schema addition) becomes searchable without a re-sync"
  - "Store.Search(ctx, webspaceName, rawQuery) — webspace-scoped, bm25-ranked, snippet-carrying full-text search"
  - "ftsQuery raw-text sanitizer — no user search-box input can ever produce an FTS5 syntax error"
  - "GET /api/webspaces/{webspace}/search?q= route, documented in docs/api.md"
affects: [03-04-search-ui, search, kernel-index, kernel-httpapi]

tech-stack:
  added: []
  patterns:
    - "External-content FTS5 table (content='items', content_rowid='rowid') kept in sync by AFTER INSERT/UPDATE/DELETE triggers, with a first-creation 'rebuild' backfill in Store.Open for pre-existing index files"
    - "Store-only handler signature (SearchHandler(store *index.Store) http.HandlerFunc) as the structural guarantee a route can never reach a plugin — same pattern as StreamHandler"
    - "Literal-phrase FTS5 query sanitization (quote each whitespace-delimited field, AND-join, prefix-match the last term) so raw user text never reaches MATCH unescaped"

key-files:
  created:
    - kernel/httpapi/search.go
    - kernel/httpapi/search_test.go
  modified:
    - kernel/index/schema.go
    - kernel/index/store.go
    - kernel/index/store_test.go
    - kernel/httpapi/routes.go
    - docs/api.md

key-decisions:
  - "Bound the snippet's STX/ETX delimiter characters as query parameters (not string-concatenated into the SQL) so the entire search query stays a single backtick-quoted constant with zero + concatenation, matching this plan's own acceptance criteria"
  - "A residual FTS5-syntax error from the query itself degrades to an empty result set (never propagates as a 500) — the sanitizer makes this unreachable in principle, but Search treats it as a defense-in-depth backstop rather than trusting the sanitizer alone"

patterns-established:
  - "Any future full-text-search-adjacent route (agent-facing or otherwise) should reuse Store.Search and ftsQuery rather than re-deriving FTS5 query construction"

requirements-completed: [KERN-05]

coverage:
  - id: D1
    description: "GET /api/webspaces/{webspace}/search?q= returns webspace-scoped, bm25-ranked, snippet-carrying results, structurally unable to reach a plugin"
    requirement: "KERN-05"
    verification:
      - kind: unit
        ref: "kernel/httpapi/search_test.go#TestSearchHandler_OneMatchFlattenedShapeWithSnippet"
        status: pass
      - kind: unit
        ref: "kernel/index/store_test.go#TestSearch_ScopedToOneWebspaceOnly"
        status: pass
      - kind: unit
        ref: "kernel/index/store_test.go#TestSearch_TitleMatchRanksAbovePreviewMatch"
        status: pass
    human_judgment: false
  - id: D2
    description: "Missing/empty/whitespace/quote-only q returns 200 with empty results (never an error); a q matching nothing returns 200 empty; unknown webspace returns 404 webspace_not_found"
    requirement: "KERN-05"
    verification:
      - kind: unit
        ref: "kernel/httpapi/search_test.go#TestSearchHandler_AbsentQReturns200EmptyResults"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/search_test.go#TestSearchHandler_SingleQuoteQReturns200NotServerError"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/search_test.go#TestSearchHandler_UnknownWebspace404"
        status: pass
    human_judgment: false
  - id: D3
    description: "An index file that already holds items synced before this schema addition backfills its FTS index on first Open, so pre-Phase-3 content is searchable without a manual re-sync"
    requirement: "KERN-05"
    verification:
      - kind: unit
        ref: "kernel/index/store_test.go#TestBackfill_ReopeningAPreexistingIndexFindsItsItems"
        status: pass
    human_judgment: false
  - id: D4
    description: "docs/api.md documents the search route completely (q semantics, result shape, snippet delimiter contract, 50-result cap, bm25 ordering, webspace scoping, no /agent/v1 mirror) and no longer claims search is unimplemented"
    verification:
      - kind: other
        ref: "grep -q '### `GET /api/webspaces/{webspace}/search`' docs/api.md && grep -q 'snippet' docs/api.md && ! grep -Fq 'the first one with enough volume to need it' docs/api.md"
        status: pass
    human_judgment: false

duration: 7min
completed: 2026-07-29
status: complete
---

# Phase 3 Plan 3: Full-text search within a webspace Summary

**SQLite FTS5 external-content search over `items(title, preview)`, exposed as `GET /api/webspaces/{webspace}/search?q=` with bm25 ranking, highlighted snippets, and a first-creation backfill for pre-existing index files.**

## Performance

- **Duration:** ~7 min
- **Completed:** 2026-07-29
- **Tasks:** 2
- **Files modified:** 6 (2 created, 4 modified)

## Accomplishments

- `items_fts` external-content FTS5 virtual table plus `items_ai`/`items_ad`/`items_au` sync triggers, appended to `kernel/index/schema.go`'s existing idempotent schema const — no migration, exactly as the schema's own Phase 1 comment anticipated
- `Store.Open`'s first-creation backfill: checks `sqlite_master` for `items_fts` before applying the schema, and runs the fts5 `'rebuild'` command afterward if it didn't already exist — so paperless and SilverBullet items synced in Phases 1–2 become searchable the moment this schema addition lands, with no re-sync required
- `Store.Search(ctx, webspaceName, rawQuery)` — joins `items_fts` → `items` → `webspace_items`, scoped to one webspace, ordered `bm25(items_fts) ASC` (bm25 is more-negative-is-better), capped at 50 rows, returning `SearchResult{Item, Snippet, Rank}`
- `ftsQuery(raw)` — the FTS5 query sanitizer: splits on whitespace, strips embedded quotes, drops empty fields, wraps each surviving field as a literal phrase, joins with implicit AND, and prefix-matches the final term — so no user search-box text can ever reach `MATCH` unescaped
- `SearchHandler(store *index.Store) http.HandlerFunc` in the new `kernel/httpapi/search.go`, same store-only signature as `StreamHandler` (structurally unable to import `kernel/pluginhost`), registered as `GET /api/webspaces/{webspace}/search` in `routes.go`
- `docs/api.md` fully documents the new route and removes the stale "search not yet implemented" bullet

## Task Commits

1. **Task 1: A webspace is searchable end to end — FTS5 index through a kernel route** - `69ad70c` (feat)
2. **Task 2: Publish the search contract in docs/api.md** - `fe46993` (docs)

_No separate plan-metadata commit was made in this worktree — STATE.md/ROADMAP.md updates are owned by the orchestrator after all wave agents complete, per this plan's parallel-execution instructions._

## Files Created/Modified

- `kernel/index/schema.go` - Added `items_fts` FTS5 virtual table + 3 sync triggers, appended to the existing schema const
- `kernel/index/store.go` - Added `Store.Open`'s FTS5 backfill, `SnippetOpen`/`SnippetClose` constants, `SearchResult`, `ftsQuery`, `Search`
- `kernel/index/store_test.go` - Added 7 new tests covering matching, no-match, webspace scoping, ranking, snippet delimiters, empty/whitespace/quote queries, and the backfill
- `kernel/httpapi/search.go` - New: `SearchHandler`, `searchResult`, `searchResponse`, `toSearchResult`
- `kernel/httpapi/search_test.go` - New: 7 handler tests (404, absent/empty/whitespace/quote `q`, zero matches, one match with flattened JSON shape)
- `kernel/httpapi/routes.go` - Registered `r.Get("/api/webspaces/{webspace}/search", SearchHandler(store))`
- `docs/api.md` - New search route section, `Ordering guarantee` cross-reference, `/agent/v1` no-mirror note, `webspace_not_found` error-code row extended, stale "not yet implemented" bullet removed

## Decisions Made

- Bound the snippet's STX/ETX delimiters as SQL query parameters (positions 1–2 in `searchQuery`, ahead of the webspace name and MATCH query) rather than building the query string with any `+` concatenation, so `searchQuery` stays a single backtick-quoted constant end to end — required by this plan's own acceptance criteria ("the query is a single backtick-quoted constant with no `+` concatenation building any part of it").
- `Search` treats a residual FTS5-syntax error from the underlying query as "no results" (empty slice, nil error) rather than propagating it, as a defense-in-depth backstop — `ftsQuery`'s phrase-quoting makes this branch unreachable in principle, but the plan explicitly calls for the degrade-not-500 behavior regardless of whether the sanitizer is airtight.

## Deviations from Plan

None — plan executed exactly as written. Both the FTS5 schema/query design and the handler shape came directly from `03-RESEARCH.md`'s Pattern 3 (verified end-to-end against this repo's pinned `modernc.org/sqlite` dependency in the research session) and `03-PATTERNS.md`'s `stream.go`/`StreamItems` analogs.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required. This plan is independent of the email plugin/Proton Bridge work in the other Phase 3 plans; it searches whatever is already indexed from Phases 1–2 (paperless-ngx, SilverBullet).

## Next Phase Readiness

- `Store.Search` and `GET /api/webspaces/{webspace}/search` are ready for 03-04's search UI (`SearchBox.svelte`, `api.ts`'s `searchWebspace()`) to consume directly — the JSON response shape (flattened stream-item fields + `snippet`) was designed specifically so the SPA can reuse its existing stream-row rendering component.
- No blockers for 03-04 or the email plugin plans; this plan's index-layer change is additive and backward-compatible with every item already synced by paperless-ngx/SilverBullet.

---
*Phase: 03-email-in-the-webspace*
*Completed: 2026-07-29*
