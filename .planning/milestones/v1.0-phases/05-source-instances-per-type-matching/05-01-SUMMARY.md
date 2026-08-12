---
phase: 05-source-instances-per-type-matching
plan: 01
subsystem: api
tags: [go, sqlite, svelte5, config, agent-permissions, identity]

# Dependency graph
requires:
  - phase: 02-source-agnostic-kernel
    provides: config.Source, pluginhost.Host, kernel/index schema v1, agent.read grants, GET /api/sources
  - phase: 03-email-and-search
    provides: FTS5 search index, streamItem shape
provides:
  - "Source instance identity (item.Source, config-key-trusted) split from the Describe-learned plugin kind (item.SourceType)"
  - "config.Source.DisplayName (display_name) with case-insensitive uniqueness validation and instance-id default fallback"
  - "index schema_version-gated drop-and-rebuild (items.source, sync_runs.source columns)"
  - "instance-keyed sync_runs series, single-flight keys, and agent.read grants"
  - "kernel/httpapi streamItem.source / source_display_name fields; per-instance /agent/v1 grant filtering"
  - "web UI filter chips, health chips, staleness, and open-in-source CTA keyed on source instance id"
affects: [05-02, 05-03, 05-04, 05-05, 06-ui-scalable-source-surface, 07-webspace-builder-ui]

# Actuals (#2632)
actuals:
  tokens: 35280
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Two-trust-sources split: Source (config-key-trusted, D-08) vs SourceType (Describe-trusted, T-01-07) carried as two distinct fields everywhere, never merged into one parameter"
    - "schema_version-gated drop-and-rebuild migration (D-07): PRAGMA user_version compared at Open, stale non-empty index dropped and recreated in one transaction, no data migration ever written"
    - "Package-level toStreamItem/toStreamItemFor split in kernel/httpapi/stream.go so search.go (out of this plan's file scope) keeps its existing call site while every other caller resolves display names via config.DisplayNameFor"

key-files:
  created: []
  modified:
    - kernel/config/types.go
    - kernel/config/config.go
    - kernel/item/item.go
    - kernel/index/schema.go
    - kernel/index/store.go
    - kernel/correlate/correlate.go
    - kernel/syncer/coordinator.go
    - kernel/pluginhost/host.go
    - kernel/httpapi/stream.go
    - kernel/httpapi/sources.go
    - kernel/httpapi/item.go
    - kernel/httpapi/agent.go
    - kernel/httpapi/routes.go
    - docs/api.md
    - web/src/lib/api.ts
    - web/src/lib/format.ts
    - web/src/lib/components/SourceFilterChips.svelte
    - web/src/lib/components/SourceHealthChip.svelte
    - web/src/lib/components/OpenInSource.svelte
    - web/src/routes/w/[webspace]/+page.svelte

key-decisions:
  - "Deferred SourceTypesByName's removal from pluginhost/host.go into the Task 1 commit (not a separate Task 2 commit as the plan's deprecation note anticipated) — both edits landed in the same continuous session with no intermediate broken-build checkpoint to preserve, so splitting them by commit would have added no real information"
  - "kernel/httpapi/routes.go modified beyond this plan's originally scoped file list — StreamHandler/ItemHandler both gained a cfg parameter to resolve source_display_name, which necessarily changes their call sites in Router() (Rule 3: blocking compile dependency)"
  - "search.go/toSearchResult kept calling the original toStreamItem(it) signature unchanged (out of this plan's file scope); a new toStreamItemFor(it, resolveDisplayName) sibling is what every other caller (stream, item, agent handlers) uses — a search result's source_display_name therefore falls back to the instance id itself rather than any configured override, a documented, narrow scope boundary"
  - "web/src/lib/components/WebspaceHeader.svelte's onfilter prop type parameter renamed from sourceType to source for naming consistency, even though its call signature was unaffected — minor, not a behavior change"

patterns-established:
  - "Instance-id-as-identity split: every kernel layer (item, index, correlate, syncer, pluginhost, httpapi) carries Source alongside the retained SourceType, with the field's meaning ('which config entry' vs 'which plugin binary/kind') documented at each site per 05-RESEARCH.md Pitfall 3"

requirements-completed: [KERN-06]

coverage:
  - id: D1
    description: "Two instances of the same plugin type never merge: distinct item id namespaces, distinct sync_runs series, distinct single-flight keys"
    requirement: "KERN-06"
    verification:
      - kind: unit
        ref: "kernel/syncer/coordinator_test.go#TestTwoInstancesOfOnePluginType_StayDistinct"
        status: pass
    human_judgment: false
  - id: D2
    description: "An agent.read grant on one instance never admits a sibling instance of the same plugin type across any /agent/v1 response (stream, item, sources, webspace item_count)"
    requirement: "KERN-06"
    verification:
      - kind: unit
        ref: "kernel/httpapi/agent_test.go#TestAgent_TwoInstancesOfOnePluginType_UngrantedNeverLeaks"
        status: pass
    human_judgment: false
  - id: D3
    description: "display_name uniqueness (case-insensitive) validated at config load, defaulting to the instance id when omitted"
    requirement: "KERN-06"
    verification:
      - kind: unit
        ref: "kernel/config/config_test.go#TestLoad_DuplicateDisplayNameCaseInsensitiveFailsNamingBothSources"
        status: pass
      - kind: unit
        ref: "kernel/config/config_test.go#TestDisplayNameFor_OmittedDefaultsToInstanceID"
        status: pass
    human_judgment: false
  - id: D4
    description: "Index rebuilds itself (drop-and-resync) on a schema_version mismatch rather than mixing old plugin-kind-keyed rows with new instance-keyed rows"
    requirement: "KERN-06"
    verification:
      - kind: unit
        ref: "kernel/index/store_test.go (existing suite passes against the new schema_version-gated Open path)"
        status: pass
    human_judgment: true
    rationale: "The interrupted-rebuild recoverability truth is marked verification: backstop in the plan (structural guarantee via single-transaction commit, not directly exercised by a kill-mid-rebuild test) — a human should confirm this reasoning holds rather than auto-passing on indirect evidence"
  - id: D5
    description: "UI filter chips, health chips, staleness, and open-in-source CTA key on the source instance id, with long display names truncating to a title attribute"
    requirement: "KERN-06"
    verification:
      - kind: unit
        ref: "web/src/lib/components/sources.test.ts (resolveSourceFilter/filterItemsBySource two-instance cases)"
        status: pass
      - kind: unit
        ref: "web/src/lib/components/staleness.test.ts#staleSources (two-instance case)"
        status: pass
    human_judgment: true
    rationale: "The visual truncate-plus-title rendering and the deep-link CTA copy are UI-SPEC E1 rows best confirmed by a human looking at the rendered chip/CTA, not just the unit-tested data layer"

duration: 40min
completed: 2026-08-06
status: complete
---

# Phase 5 Plan 1: Source Instances & Per-Type Matching Summary

**Split source instance identity (config map key) from the Describe-learned plugin kind across the entire kernel spine, HTTP/agent surface, and UI — two configured instances of one plugin binary now sync, index, and grant independently, with a new `display_name` config field and schema-version-gated index rebuild.**

## Performance

- **Duration:** ~40 min
- **Started:** 2026-08-06T10:25:28Z
- **Completed:** 2026-08-06T11:02:00Z
- **Tasks:** 3
- **Files modified:** 35

## Accomplishments

- `item.Item.Source` (instance id, config-key-trusted) carries identity everywhere — item ids (`{source}:{source_id}`), `sync_runs` rows, single-flight keys, `/agent/v1` grants — while `item.SourceType` (Describe-learned plugin kind) is retained unchanged as descriptive provenance, never an identity key
- `config.Source.DisplayName` (`display_name`) with case-insensitive uniqueness validation at load time and a default fallback to the instance id when omitted
- `kernel/index` gained a `schemaVersion`-gated drop-and-rebuild path (D-07): a stale on-disk `PRAGMA user_version` triggers a single-transaction drop-and-recreate of every schema.go-owned table, never a data migration
- `kernel/httpapi` publishes `source` and `source_display_name` on every item alongside the retained `source_type`; `/agent/v1` grant filtering now keys directly on the instance id via `grantedSources(cfg)`, replacing the old name→plugin-kind indirection
- The Fetch-call bug this phase fixes: `ItemHandler`/`renditionHandler`/agent equivalents now call `fetcher.Fetch(ctx, it.Source, ...)` instead of `it.SourceType` — calling with the plugin kind would have resolved the wrong (or no) launched plugin once two instances of one kind exist
- Web UI (filter chips, health chips, staleness, open-in-source CTA, `+page.svelte`'s `sourcesByInstance` map) all key on the instance id; the hardcoded `SOURCE_DISPLAY_NAMES` fallback map and `sourceDisplayName()` helper are deleted — the only fallback is now the instance id itself, matching the kernel's own D-09 default

## Task Commits

Each task was committed atomically:

1. **Task 1: Two named instances of one plugin type, distinct end to end** - `512a385` (feat)
2. **Task 2: Instance identity on the HTTP and agent surface** - `114a292` (feat)
3. **Task 3: UI reads instance identity and tolerates user-authored display names** - `c7bf32f` (feat)

_No separate TDD RED/GREEN commits: tests were written alongside their implementation within each task's single commit, consistent with this repo's existing per-task (not per-TDD-phase) commit granularity._

## Files Created/Modified

- `kernel/config/types.go`, `kernel/config/config.go` - `Source.DisplayName`, `DisplayNameFor`, uniqueness validation
- `kernel/item/item.go` - `Item.Source`, re-keyed `ID()`/`FromProto()`
- `kernel/index/schema.go`, `kernel/index/store.go` - `items.source`/`sync_runs.source` columns, `schemaVersion`-gated rebuild, `SyncingSources`, instance-keyed `LatestSyncRunPerSource`
- `kernel/correlate/correlate.go`, `kernel/syncer/coordinator.go` - `WebspaceResult.Source`, `RunResult.Source`, sync identity promoted to `(webspace, source)`
- `kernel/pluginhost/host.go` - `byInstance` lookup, `Plugin.DisplayName`/`PluginDisplayName` split, `SourceTypesByName` deleted
- `kernel/httpapi/stream.go`, `item.go`, `agent.go`, `sources.go`, `routes.go` - instance-keyed HTTP/agent surface, `grantedSources`, `toStreamItemFor`
- `docs/api.md` - documents `source`/`source_display_name`, `{source}:{source_id}` id scheme, per-instance agent grants
- `web/src/lib/api.ts`, `format.ts`, `+page.svelte`, `SourceFilterChips.svelte`, `SourceHealthChip.svelte`, `OpenInSource.svelte` - instance-keyed UI, deleted hardcoded display-name map

## Decisions Made

- `kernel/pluginhost/host.go`'s `SourceTypesByName` deletion was completed together with Task 1's `host.go` sweep rather than deferred to a separate Task 2 commit (the plan's own contingency note) — both edits landed in one continuous implementation pass with no intermediate broken-build checkpoint worth preserving in history.
- `kernel/httpapi/routes.go` was modified beyond this plan's declared `files_modified` list: `StreamHandler`/`ItemHandler` both gained a `cfg` parameter (to resolve `source_display_name`), which necessarily changes their construction call sites in `Router()`. Applied under deviation Rule 3 (blocking compile dependency) — the signature change was unavoidable once `toStreamItemFor` needed `cfg.DisplayNameFor`.
- `search.go`'s `toSearchResult` keeps calling the pre-existing `toStreamItem(it)` signature unchanged (that file is explicitly out of this plan's scope) — a new `toStreamItemFor(it, resolveDisplayName)` sibling is what every other caller uses. A search result's `source_display_name` therefore reports the instance id itself rather than any configured override; documented as a narrow, intentional scope boundary in `stream.go`'s doc comments.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `kernel/httpapi/routes.go` required cfg-parameter wiring**
- **Found during:** Task 2 (Instance identity on the HTTP and agent surface)
- **Issue:** `StreamHandler`/`ItemHandler` needed `*config.Config` to resolve `source_display_name` via `toStreamItemFor`, but `routes.go` (not in this plan's declared file list) constructs both handlers in `Router()`
- **Fix:** Updated `Router()`'s two call sites to pass `cfg`; updated the doc comment explaining why this doesn't weaken the KERN-02 plugin-isolation boundary (`cfg` is inert data, never a plugin handle)
- **Files modified:** `kernel/httpapi/routes.go`
- **Verification:** `go build ./...` and `go test ./kernel/httpapi/...` both pass
- **Committed in:** `114a292` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking)
**Impact on plan:** Necessary wiring for Task 2's own stated action (`toStreamItem gains a display-name resolver argument`) to compile; no scope creep beyond what that action already required.

## Issues Encountered

None beyond the routes.go wiring above.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The identity split is complete end to end (config → index → sync → HTTP → agent → UI); Plans 05-02 through 05-05 (per-type matching, vocabulary validation, rendition centralization) build on this instance-keyed foundation without needing further identity rework.
- `config.example.toml` was not updated with a `display_name` example in this plan (not in the declared file list) — worth a small follow-up when a later plan next touches that file, so operators discover the new key from the example config, not only from `docs/api.md`.

---
*Phase: 05-source-instances-per-type-matching*
*Completed: 2026-08-06*

## Self-Check: PASSED

All key files verified present on disk; all three task commit hashes (512a385, 114a292, c7bf32f) verified present in `git log --oneline --all`.
