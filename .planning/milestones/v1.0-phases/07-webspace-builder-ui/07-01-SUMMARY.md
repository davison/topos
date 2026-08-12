---
phase: 07-webspace-builder-ui
plan: 01
subsystem: api
tags: [go, sqlite, fts5, svelte5, toml, config, search]

# Dependency graph
requires:
  - phase: 06-ui-scalable-source-surface
    provides: WebspaceHeader.svelte's chip-row layout, search-box wiring, and the per-page load()/navGeneration race-guard pattern this plan's filter state reuses
provides:
  - "config.Store: swappable raw/expanded config holder with a content-hash optimistic lock (D-03), backing the kernel's first mutating HTTP surface"
  - "PUT /api/config: validate -> hash-check -> unknown-key-check -> canonical write -> hot-swap, with config.toml.bak and atomic rename"
  - "A webspace's filter [] terms narrow GET /api/webspaces/{ws}/stream, .../search, and GET /agent/v1/webspaces/{ws}/stream identically via the shared BuildMatchQuery"
  - "Search AND-combines a live query with the saved filter stack rather than escaping it (D-18)"
  - "FilterChip.svelte + WebspaceHeader's Save-as-filter affordance: stackable, removable, disableable while a write is in flight"
affects: [07-02, 07-03, 07-04, 07-05]

# Actuals (#2632) — pairs with the plan's estimate to calibrate future estimates.
actuals:
  tokens: 41553
  tasks: 3
  commits: 6

tech-stack:
  added: []
  patterns:
    - "config.Store: atomic.Pointer[Config] pair (raw/expanded) plus a hash-guarded Save — the seam every future config-writing plan in this phase reuses"
    - "BuildMatchQuery(filterTerms, liveQuery): the single shared FTS5 MATCH composer StreamItems, Search and the agent stream mirror all go through, so the filtered view can never disagree with itself across consumers"
    - "AST-based structural guards (go/parser + go/ast) over routes.go and config.go, proving a closed non-GET route set and a plugin-call-free config handler file — a future PR that widens either fails the suite outright"

key-files:
  created:
    - kernel/config/store.go
    - kernel/config/writer.go
    - kernel/config/store_test.go
    - kernel/config/writer_test.go
    - kernel/httpapi/config.go
    - kernel/httpapi/config_test.go
    - web/src/lib/components/FilterChip.svelte
    - web/src/lib/components/filter-chip.test.ts
  modified:
    - kernel/config/types.go
    - kernel/config/config.go
    - kernel/httpapi/routes.go
    - kernel/httpapi/stream.go
    - kernel/httpapi/search.go
    - kernel/httpapi/agent.go
    - kernel/index/store.go
    - cmd/topos/main.go
    - web/src/lib/api.ts
    - web/src/lib/components/WebspaceHeader.svelte
    - web/src/routes/w/[webspace]/+page.svelte
    - docs/api.md
    - config.example.toml

key-decisions:
  - "assumption-delta: the running configuration is promoted to the primary noun (config.Store), demoting the boot-time *config.Config pointer to a per-request Store.Expanded() snapshot — locked at Task 1's tracer checkpoint"
  - "WebspacesHandler/ItemHandler/SourceRefreshHandler keep a boot-time cfg snapshot as accepted debt for this plan only (07-02 Task 2 fills the gap); StreamHandler/SearchHandler/agentStreamHandler read cfgStore.Expanded() fresh per request so a saved filter narrows the very next request with no restart"
  - "Store.Search's empty-result short circuit now depends on BOTH filterTerms and rawQuery sanitizing to nothing (Task 2) — a filter-only call (empty live query, non-empty saved filter) still queries and ranks by relevance rather than returning early"
  - "$state.snapshot() (not structuredClone) for cloning the fetched config document before mutation — Svelte 5's reactive Proxy unconditionally rejects structuredClone in every engine (tracer-gate live-repro fix, d8125cf)"

patterns-established:
  - "Two AST guards (TestRoutesGuard_NonGetRoutesScopedToConfig, TestConfigHandlersGuard_NoPluginCallOtherThanDescribe) pin a structural security boundary at compile-test time rather than only in prose — both manually verified to fail loudly against a deliberately introduced violation, then reverted, before being trusted as the committed guard"
  - "Secret-round-trip tests prove the raw/expanded split by asserting Store.Expanded() DOES hold the resolved value (not just that Store.Raw()/disk don't) — a test that skipped the positive assertion could pass vacuously against a build where expansion silently broke"

requirements-completed: [KERN-08, UI-12]

coverage:
  - id: D1
    description: "Save-as-filter tracer: click Save as filter, stream narrows, term persists to config.toml with .bak, chip renders/removes, no kernel restart — human-verified live in Task 1's tracer gate"
    requirement: "UI-12"
    verification:
      - kind: manual_procedural
        ref: "Task 1 tracer feedback gate — human returned 'approved' after live save/persist/chip/remove/narrow-widen verification"
        status: pass
      - kind: unit
        ref: "kernel/config/store_test.go#TestStore_Save_NoUnknownKeysSucceeds"
        status: pass
    human_judgment: false
  - id: D2
    description: "A webspace's filter[] narrows the stream, search, and agent-stream surfaces identically via BuildMatchQuery, with no trailing prefix-match on saved terms"
    requirement: "KERN-08"
    verification:
      - kind: unit
        ref: "kernel/index/store_test.go#TestStreamItems_TwoTermFilterRequiresBoth"
        status: pass
      - kind: unit
        ref: "kernel/index/store_test.go#TestStreamItems_SingleTermFilterNarrowsToMatchingRowsChronological"
        status: pass
      - kind: unit
        ref: "kernel/index/store_test.go#TestStreamItems_EmptyFilterReturnsIdenticalRowsAndOrder"
        status: pass
    human_judgment: false
  - id: D3
    description: "A live search AND-combines with the saved filter stack rather than replacing it"
    requirement: "KERN-08"
    verification:
      - kind: unit
        ref: "kernel/index/store_test.go#TestSearch_FilterTermsANDCombineWithLiveQuery"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/search_test.go#TestSearchHandler_ReadsWebspaceFilterFromLiveConfig"
        status: pass
    human_judgment: false
  - id: D4
    description: "FilterChip visually distinct from SourceChip (rounded-md vs rounded-full), absent (not empty-styled) at zero filters, remove control labeled, both controls disableable while a write is in flight"
    requirement: "UI-12"
    verification:
      - kind: unit
        ref: "web/src/lib/components/filter-chip.test.ts"
        status: pass
    human_judgment: false
  - id: D5
    description: "docs/api.md documents GET/PUT /api/config (request/response shape, failure modes, raw-form guarantee) and the four new error codes; config.example.toml documents the filter key"
    verification: []
    human_judgment: true
    rationale: "Prose documentation accuracy is a judgment call beyond what a compiling-TOML sanity check (performed manually this session, not committed as a test) or a grep-based guard can certify"
  - id: D6
    description: "Persistence invariants D-01/D-03/D-04/D-05/D-09 plus the lossless-rewrite prohibition each have a test that goes red when the guarantee breaks — including two manually verified against a deliberately introduced violation"
    requirement: "KERN-08"
    verification:
      - kind: unit
        ref: "kernel/config/store_test.go#TestStore_Save_SecretRoundTrip_NeverPersistsResolvedValue"
        status: pass
      - kind: unit
        ref: "kernel/config/store_test.go#TestStore_Save_ClobberGuard_StaleHashRejectedFileUnchanged"
        status: pass
      - kind: unit
        ref: "kernel/config/writer_test.go#TestWriteCanonical_BackupHoldsPreviousContentAndOverwritesOnSecondWrite"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/config_test.go#TestConfigSaveHandler_InvalidConfigReturns422WithValidatorMessageVerbatim"
        status: pass
    human_judgment: false
  - id: D7
    description: "Config-scoped mutation boundary (success criterion 4): routes.go registers no non-GET route beyond PUT /api/config and the two pre-existing refresh routes; config.go references no plugin Match/Fetch call"
    requirement: "KERN-08"
    verification:
      - kind: unit
        ref: "kernel/httpapi/config_test.go#TestRoutesGuard_NonGetRoutesScopedToConfig"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/config_test.go#TestConfigHandlersGuard_NoPluginCallOtherThanDescribe"
        status: pass
    human_judgment: false

duration: ~52min (across three tasks, including the pause for Task 1's human-verify checkpoint)
completed: 2026-08-08
status: complete
---

# Phase 7 Plan 1: Search-Promotion Permanent Filters Summary

**Search-to-filter promotion through the kernel's first mutating config surface — a hash-guarded raw/expanded config.Store, canonical TOML writer with rolling backup, and a shared BuildMatchQuery narrowing every read surface (stream, search, agent) identically, with the filter stack AND-combining a live search rather than replacing it.**

## Performance

- **Duration:** ~52 min total plan span (three tasks; includes the human-verify pause between Task 1's tracer commit and its approval)
- **Started:** 2026-08-07T23:04:38Z (Task 1's first commit)
- **Completed:** 2026-08-07T23:56:59Z (Task 3's last commit)
- **Tasks:** 3
- **Files modified:** 31 (13 created, 18 modified) across the whole plan

## Accomplishments

- The kernel's first mutating HTTP surface (`PUT /api/config`) ships with every hardening invariant this phase's threat register requires: an optimistic content-hash clobber guard (D-03), a lossless-rewrite refusal for hand-authored keys the `Config` struct doesn't model (D-01), and a D-09 validation dry run that reuses the exact same `Validate` a hand-edited file must pass at load time.
- A webspace's saved `filter` stack narrows `GET /api/webspaces/{ws}/stream`, `.../search`, and `GET /agent/v1/webspaces/{ws}/stream` **identically** through one shared `BuildMatchQuery` — and a live search now AND-combines with that stack instead of escaping it, so a further search always refines within the saved filter (D-18).
- Every one of the phase's locked invariants — D-01, D-03, D-04, D-05, D-09, D-16, D-18, plus the lossless-rewrite and config-scoped-mutation boundaries — now has a test that goes red when the guarantee breaks. Three of those tests were manually verified to actually go red against a deliberately introduced violation (a swapped `WriteCanonical` argument, an added `Delete` route, an added `Fetch` reference), then reverted, before being trusted as the committed guard.
- `docs/api.md` and `config.example.toml` document the new surface for a hand-editing operator: the raw-form guarantee, the four new error codes, and the `filter` key's own worked example.

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end "save this search as a permanent filter" tracer** — `08e17aa` (feat), with two live-checkpoint fixes folded in: `2b6cbe7` (fix — unknown-keys Alert), `d8125cf` (fix — `$state.snapshot()` clone bug). Human-verified and approved live before Task 2 began.
2. **Task 2: The filter stack is stackable, removable, and refines live searches** — `6f846aa` (feat)
3. **Task 3: Pin the persistence invariants** — `1b83cbb` (test)

**Plan metadata:** (this commit)

## Files Created/Modified

- `kernel/config/store.go` - `Store`: swappable raw/expanded config holder, `Save`'s validate→hash-check→unknown-key-check→write→swap sequence
- `kernel/config/writer.go` - `WriteCanonical`: canonical TOML, `CanonicalHeader`, rolling `.bak`, atomic temp+rename
- `kernel/config/types.go` - `Webspace.Filter []string`; `json` tags mirroring every `toml` tag; `omitempty` on `Source.BaseURL/Token/APIVersion`
- `kernel/config/config.go` - `LoadRaw`/`UnknownKeys`, the dual-parse (expanded + raw) load path
- `kernel/config/store_test.go` / `writer_test.go` - D-01/D-03/D-04/D-05 invariant tests (new)
- `kernel/httpapi/config.go` - `ConfigHandler`/`ConfigSaveHandler`, the raw-only response envelope, env-var presence scan
- `kernel/httpapi/config_test.go` - HTTP-layer invariant tests, D-09 validator-message test, the two AST structural guards (new)
- `kernel/httpapi/routes.go` - `Router` takes `*config.Store`; mounts `GET`/`PUT /api/config`
- `kernel/httpapi/stream.go`, `kernel/httpapi/agent.go` - `StreamItems`/agent stream pass the webspace's `Filter`, reading `cfgStore.Expanded()` fresh per request
- `kernel/httpapi/search.go` - `Store.Search` takes `filterTerms`; `SearchHandler` reads the live config store
- `kernel/index/store.go` - `BuildMatchQuery`, `StreamItems`'s filter-aware query, `Search`'s AND-combined MATCH
- `cmd/topos/main.go` - boots a `*config.Store` via `config.NewStore`
- `web/src/lib/api.ts` - `getConfig`/`putConfig`, `KernelConfig`/`ConfigResponse`/`ConfigSaveRequest` types
- `web/src/lib/components/FilterChip.svelte` - the `rounded-md` filter chip (new)
- `web/src/lib/components/filter-chip.test.ts` - source-scan guard for D-19 distinctness, the empty-row gate, aria-label, disabled bindings (new)
- `web/src/lib/components/WebspaceHeader.svelte`, `web/src/routes/w/[webspace]/+page.svelte` - Save-as-filter affordance, chip row, `writeFilter`'s single write path
- `docs/api.md` - `GET`/`PUT /api/config` sections, four new error-code rows, corrected "What is not here yet"
- `config.example.toml` - `filter` key documentation, commented worked example

## Decisions Made

- **assumption-delta (locked at Task 1's checkpoint):** the running configuration is promoted to the primary noun — `config.Store` becomes authoritative, with a boot-time `*config.Config` demoted to a per-request `Store.Expanded()` snapshot. `WebspacesHandler`/`ItemHandler`/`SourceRefreshHandler` keep a boot-time snapshot as accepted debt for this plan alone (07-02 Task 2 fills the gap); `StreamHandler`/`SearchHandler`/`agentStreamHandler` read fresh per request so a saved filter narrows the very next request with no kernel restart.
- **`Store.Search`'s empty-result short circuit** now depends on both `filterTerms` and `rawQuery` sanitizing to nothing (Task 2) — a filter-only call still queries and ranks by relevance rather than returning early, which is what lets a future filter-only search view exist without a second code path.
- **`$state.snapshot()` over `structuredClone`** for cloning the fetched config document before mutation — a live tracer-checkpoint bug found that Svelte 5's reactive Proxy unconditionally rejects `structuredClone` in every engine, which was silently killing every save click (fixed in `d8125cf`).

## Deviations from Plan

### Auto-fixed Issues (folded into Task 1's tracer commits, listed here for completeness)

**1. [Rule 2 - Missing Critical] `GET /api/config`'s `unknown_keys` was computed server-side but never surfaced to the operator**
- **Found during:** Task 1 tracer human-verify checkpoint
- **Issue:** A stray hand-authored config key blocks every save (by design — the lossless-rewrite guard), but the UI gave no indication why "Save as filter" silently failed until the click was attempted.
- **Fix:** `unknown_keys` now renders as a persistent destructive `Alert` in `WebspaceHeader.svelte` on page load, independent of any save attempt.
- **Files modified:** `web/src/lib/components/WebspaceHeader.svelte`, `web/src/routes/w/[webspace]/+page.svelte`, plus a source-scan test.
- **Committed in:** `2b6cbe7`

**2. [Rule 1 - Bug] `writeFilter` cloned a Svelte 5 `$state` Proxy with `structuredClone`, which always throws**
- **Found during:** Task 1 tracer human-verify checkpoint (live repro — every save click died silently)
- **Issue:** `structuredClone` unconditionally rejects a Proxy in every engine; this threw before the code's own `try` block, surfacing only as an unhandled promise rejection in the browser console.
- **Fix:** Switched to `$state.snapshot()`; both silent early-exits in the save path now surface a visible `filterError` message; non-`ApiError` failures no longer mislabeled as hash conflicts.
- **Files modified:** `web/src/routes/w/[webspace]/+page.svelte`, plus 8 new tests (`save-filter-clone.test.ts`).
- **Committed in:** `d8125cf`

**3. [Rule 3 - Blocking, environmental, not code] Operator's live `~/.config/topos/config.toml` had a stale Phase-5 `[webspaces.cars.<instance>]` table missing its `.match.` segment**
- **Found during:** Task 1 tracer human-verify checkpoint
- **Fix:** Repaired by hand; a backup was taken at `~/.config/topos/config.toml.pre-07-01-fix.bak` before a successful UI save canonically rewrote the file (now carrying the managed header).
- **Files modified:** none (operator's local config only, outside the repo).

---

**Total deviations:** 3 auto-fixed during Task 1's human-verify checkpoint (1 missing critical, 1 bug, 1 environmental repair). None during Task 2 or Task 3 — both executed exactly as planned.
**Impact on plan:** All three were necessary for the tracer's own done criteria ("no kernel restart", "a user can click Save as filter and see the stream narrow") to actually hold on the operator's real environment. No scope creep.

## Known Stubs

None — no hardcoded empty values, placeholder text, or unwired data sources were introduced by this plan.

## Issues Encountered

None beyond the three items already documented above as tracer-gate deviations. Task 2 and Task 3 completed with zero test failures on first correct implementation, aside from two self-caught test-authoring mistakes fixed before commit: an `sed`/`perl` regex that mangled a `context.Background()` call inside a bulk signature-update (caught by `go build`, fixed manually) and an incorrect non-nil-slice assumption in a hostile-filter-term test (caught by the test itself, the assertion removed as not part of `StreamItems`' actual contract).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The `PUT /api/config` seam (`config.Store.Save`) is exactly what every later plan in this phase (hot-apply, the webspace switcher, the source picker, the manage-sources escape hatch) writes through — proven end-to-end and invariant-tested here first.
- 07-02 Task 2 must thread `cfgStore` through `WebspacesHandler`/`ItemHandler`/`SourceRefreshHandler` (currently a boot-time snapshot, deliberately accepted as this plan's own debt) and add the companion invariant test asserting no handler constructor captures a `*config.Config` at `Router` construction time.
- No blockers identified for 07-02 through 07-05.

---
*Phase: 07-webspace-builder-ui*
*Completed: 2026-08-08*

## Self-Check: PASSED
