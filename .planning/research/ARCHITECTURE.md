# Architecture Research

**Domain:** Local-first personal cross-source data aggregation ("kernel + plugin" correlation service)
**Researched:** 2026-07-27
**Confidence:** MEDIUM-HIGH (component-boundary patterns HIGH — cross-checked against three independent shipped systems; Webspaces-specific specifics like exact paperless-ngx UI routes are MEDIUM)

> Note on tooling: this run's `gsd-tools query research-plan` / `classify-confidence` seams were not available in the installed gsd-tools version (`Unknown command: research-plan`). Research proceeded via direct `WebSearch`/`WebFetch` against primary sources (project wikis, official docs, GitHub READMEs). Confidence tiers below are assigned manually using the same hierarchy: official docs/source = HIGH, secondary community writeups = MEDIUM, single unverified blog = LOW.

## Standard Architecture

### System Overview

Three independent, actively-maintained systems converge on the *same* shape for "pull heterogeneous personal/enterprise data into one place, browse from a single UI, defer mutation to the source": **Timelinize** (personal timeline archive, Go, single-user desktop), **Onyx/Danswer** (enterprise connector-based search, Python, multi-tenant server), and **Home Assistant** (device/service integration hub, Python, single-node). Webspaces sits architecturally closest to Timelinize (single-user, local-first, deep-link-back) but should borrow Onyx's connector *state-machine* (load vs. poll) and Home Assistant's *coordinator* pattern for the sync scheduler.

```
┌───────────────────────────────────────────────────────────────────────┐
│                              Web UI (client)                          │
│   Webspace picker → Stream (cross-source feed) → Detail pane          │
│   "open in source" deep link · filter by source/type · search box     │
└───────────────────────────┬─────────────────────────────────────────--┘
                             │ HTTP (JSON) — kernel's own API, not plugin APIs
┌───────────────────────────▼───────────────────────────────────────────┐
│                              KERNEL                                   │
│  ┌─────────────┐ ┌────────────────┐ ┌──────────────┐ ┌─────────────┐  │
│  │  Webspace   │ │  Correlation   │ │     Sync     │ │  HTTP API   │  │
│  │  Config     │ │    Engine      │ │  Scheduler   │ │  (for UI)   │  │
│  │  (keyword   │ │  (match items  │ │ (per-plugin  │ │  read-only, │  │
│  │  map)       │ │  → webspace)   │ │ refresh loop)│ │  paginated  │  │
│  └──────┬──────┘ └───────┬────────┘ └──────┬───────┘ └──────┬──────┘  │
│         └────────────────┴──────────────────┴────────────────         │
│                                  │                                     │
│                         ┌────────▼─────────┐                          │
│                         │   Plugin Host    │  ← loads/manages plugins │
│                         │  (registry +     │    via the Plugin        │
│                         │   lifecycle)     │    Contract (below)      │
│                         └────────┬─────────┘                          │
│                                  │                                     │
│                         ┌────────▼─────────┐                          │
│                         │   Index Store    │  ← normalized Item rows  │
│                         │ (metadata+       │    + sync cursors +      │
│                         │  preview only)   │    webspace assignments  │
│                         └──────────────────┘                          │
└───────────────────────────┬─────────────────────────────────────────--┘
                             │ Plugin Contract (discover → enumerate →
                             │  metadata → live-fetch → deep-link)
      ┌──────────────┬───────────────┬───────────────┬────────────────┐
      ▼              ▼               ▼               ▼                ▼
┌──────────┐  ┌─────────────┐ ┌─────────────┐ ┌──────────────┐ ┌─────────────┐
│  Email   │  │   Signal    │ │  WhatsApp   │  │ paperless-ngx│ │ SilverBullet│
│  plugin  │  │   plugin    │ │   plugin    │  │   plugin     │ │   plugin    │
│ (IMAP,   │  │ (read-only  │ │ (read-only  │  │ (REST API,  │ │ (HTTP API,  │
│  live)   │  │ local SQL-  │ │ local store,│  │  live LAN)   │ │  live LAN)  │
│          │  │ Cipher DB)  │ │ live local) │  │              │ │             │
└────┬─────┘  └──────┬──────┘ └──────┬──────┘  └──────┬───────┘ └──────┬──────┘
     │               │               │                │                │
     ▼               ▼               ▼                ▼                ▼
Proton Bridge   Signal Desktop  WhatsApp local  paperless-ngx     SilverBullet
(IMAP, LAN)     DB (same host)  store (same     REST API (LAN)   HTTP API (LAN)
                                host)
```

### Component Responsibilities

| Component | Responsibility | Typical Implementation |
|-----------|----------------|------------------------|
| Webspace Config | Declarative map: webspace name → keyword → per-source native-category match rule (IMAP folder/label, chat group name, paperless-ngx tag, SilverBullet tag/page) | Single config file (YAML/TOML/JSON) hot-reloadable; validated against plugin-declared capabilities at load time |
| Plugin Host | Discovers, registers, and manages plugin lifecycle (init with credentials, health check, teardown); enforces the Plugin Contract; isolates plugin failures | In-process registry keyed by plugin ID (Go/Node: interface implementations; if out-of-process, gRPC/JSON-RPC over stdio or Unix socket per plugin, à la Terraform/Home Assistant custom-component model) |
| Sync Scheduler | Drives each plugin's enumerate step on an interval or manual trigger; tracks last-synced cursor per (plugin, webspace); backs off on error | Per-plugin async job/coroutine; analogous to HA's `DataUpdateCoordinator` (one coordinated poll shared by consumers) and Onyx's poll/load connector jobs |
| Correlation Engine | Applies the webspace's keyword rule against each plugin's enumerated items; assigns matched items to the webspace; is the *only* place cross-source logic lives (v1: deterministic, no ranking/AI) | Pure function per (webspace, plugin, item-metadata) → boolean/webspace-id; runs at sync time, not query time, so the index already has webspace assignment persisted |
| Index Store | Persists normalized `Item` rows (metadata + preview, not full content), sync cursors/state, webspace assignments | Single embedded DB (SQLite is the natural choice here — matches Timelinize's own choice for the same problem shape: local-first, single-writer, easy backup) |
| HTTP API | Read-only, paginated API the Web UI consumes; never talks to plugin/source APIs directly — always goes through the kernel | REST or simple RPC; endpoints: list webspaces, stream items for a webspace (cursor-paginated, filterable by source/type/date), fetch single item detail (triggers live-fetch), trigger manual sync |
| Web UI | Webspace picker, chronological cross-source stream, detail pane with inline preview, source filter, "open in source" deep link | SPA or server-rendered; talks only to kernel's HTTP API |
| Plugins (one per source) | Implement the Plugin Contract: discover identifiers, enumerate items matching a keyword rule, return metadata/preview, live-fetch full content, produce a deep link | One package/module per source; some wrap a live remote API (email/paperless-ngx/SilverBullet), others read a local file/DB directly (Signal/WhatsApp) — see Plugin Contract below for why this distinction matters |

## Recommended Project Structure

```
webspaces/
├── kernel/
│   ├── config/                # webspace config loader + schema validation
│   │   └── webspaces.yaml
│   ├── correlate/              # correlation engine (pure matching logic)
│   ├── sync/                   # scheduler, per-plugin job runner, backoff
│   ├── store/                  # index store: schema, migrations, queries
│   │   └── migrations/
│   ├── plugins/                # plugin host: registry, contract types, lifecycle
│   │   └── contract.go|.ts     # the interface every plugin implements
│   └── api/                    # HTTP API for the UI (read-only, paginated)
├── plugins/
│   ├── email-imap/
│   ├── signal/
│   ├── whatsapp/
│   ├── paperless-ngx/
│   └── silverbullet/
├── ui/
│   ├── stream/                 # cross-source chronological feed
│   ├── detail/                 # per-item detail pane, source-specific renderers
│   └── webspace-picker/
└── docs/
    └── PLUGIN-CONTRACT.md      # the stable, documented contract (per constraint: third-party pluggability)
```

### Structure Rationale

- **kernel/plugins/contract.\*** is deliberately isolated from any individual plugin — it's the one file/package third-party plugin authors need to read. Keeping it out of any specific plugin's directory signals "this is the interface, not an implementation."
- **plugins/** as siblings, each a self-contained package with its own credentials/config schema — mirrors Home Assistant's `custom_components/<domain>/` and Onyx's `backend/onyx/connectors/<name>/` layout. New sources are added by dropping in a new sibling directory, never by touching the kernel.
- **kernel/store/** owns the *only* schema for normalized items — no plugin writes its own tables; this keeps the correlation engine source-agnostic.
- **ui/detail/** has source-specific renderers (email body vs. chat thread vs. note vs. PDF preview) but they all consume the same normalized `Item` + a "detail payload" blob from live-fetch — rendering variance is presentation-layer only, not a kernel concern.

## Architectural Patterns

### Pattern 1: Hybrid local-index + live-fetch (borrowed from Timelinize's local-archive model, inverted)

**What:** The Index Store never holds full content — only metadata + a short preview needed to render the stream and support search/filter. When a user opens an item's detail pane, the kernel calls the *same plugin's* live-fetch method to pull full content directly from the source at that moment.
**When to use:** Whenever content is (a) large, (b) already durably stored at the source, and (c) subject to source-side edits/deletions you want reflected live (email read-state, chat edits, doc revisions). This is essentially the *opposite* of Timelinize (which duplicates everything into its own archive) and closer to Onyx's "index metadata, but the answer engine can also cite/open the live doc" — chosen deliberately per Webspaces' view-only, no-duplication requirement.
**Trade-offs:** Pro: no storage bloat, no staleness of full content, respects "read-only, no mutation" constraint trivially (nothing is copied that could drift). Con: opening an item is only as fast as the slowest source (IMAP round-trip, LAN latency); requires every plugin to support two speeds of access (cheap enumerate, potentially expensive fetch); local-DB sources (Signal/WhatsApp) get this "for free" since re-reading a local SQLite file is fast — the two speeds effectively collapse to one for those two plugins.

**Example (contract shape, Go/TS-flavored pseudocode):**
```typescript
interface SourcePlugin {
  id(): string;                                   // stable plugin identifier
  discoverIdentifiers(): Promise<Identifier[]>;    // e.g. IMAP folders, chat group names, tags
  enumerate(rule: KeywordRule, since?: Cursor): Promise<ItemMetadata[]>; // cheap, batched
  fetchFull(ref: ItemRef): Promise<ItemContent>;   // expensive, called only on open
  deepLink(ref: ItemRef): string;                  // URL/URI back to the source app
}
```

### Pattern 2: Config-declared correlation, resolved at sync time not query time (deterministic v1)

**What:** Each webspace declares a keyword and, per source, how that keyword maps onto the source's *native* categorization (IMAP label name, chat group name, paperless-ngx tag, SilverBullet tag/page). The correlation engine evaluates this mapping once per sync cycle per plugin and writes the webspace assignment onto the Item row in the Index Store.
**When to use:** V1 explicitly rejects AI/context-relative correlation (per PROJECT.md) — this pattern gives zero false positives and is trivially explainable/debuggable ("why is this here? because it's tagged X"). It is the deliberate seam where a v2 "AI-inferred correlation" layer can later be inserted, alongside (not replacing) the deterministic map.
**Trade-offs:** Pro: fast reads (assignment precomputed), explainable, no runtime cost per query. Con: doesn't catch items that *should* semantically belong but aren't natively tagged/named that way — an explicit, accepted limitation per Out of Scope.

### Pattern 3: Coordinator-per-plugin sync scheduling (borrowed from Home Assistant's DataUpdateCoordinator)

**What:** One coordinator object per plugin instance owns polling cadence, dedupes concurrent refresh requests, and pushes updated state to the Index Store; downstream consumers (correlation engine, UI "last synced" indicator) subscribe rather than each polling independently.
**When to use:** Prevents redundant API/DB calls when multiple webspaces reference the same plugin (e.g., three webspaces each need the email plugin's Inbox folder polled — one coordinator serves all three, not three separate pollers). Also gives one place to implement backoff/error surfacing per source.
**Trade-offs:** Pro: single source of truth for "is this plugin healthy / when did it last sync," easy to expose in the UI. Con: a slow/misbehaving plugin's coordinator can lag all webspaces depending on it unless isolated per-plugin (which it is, by design — one coordinator per plugin, not global).

## Data Flow

### Sync Flow (background, scheduled)

```
Sync Scheduler (interval tick, per plugin)
    ↓
Plugin Host invokes plugin.enumerate(keywordRule, sinceCursor)
    ↓
Plugin talks to its source:
    - live remote sources (IMAP, paperless-ngx, SilverBullet): network call over LAN
    - local sources (Signal, WhatsApp): direct read of local DB/store on same host
    ↓
Plugin returns ItemMetadata[] (normalized shape, see Normalized Item Schema)
    ↓
Correlation Engine matches metadata against webspace keyword rules
    ↓
Index Store: upsert Item rows (metadata + preview + webspace assignment + cursor)
    ↓
UI "last synced" indicator updates (kernel exposes coordinator state via HTTP API)
```

### Request Flow (user opens the UI)

```
[User opens webspace]
    ↓
UI → kernel HTTP API: GET /webspaces/{id}/stream?cursor=&source=&type=
    ↓
Kernel → Index Store: paginated query, ordered by timestamp, filtered
    ↓
Kernel → UI: normalized Item[] (metadata + preview only — fast, no live-fetch)
    ↓
[User clicks an item]
    ↓
UI → kernel HTTP API: GET /items/{id}/detail
    ↓
Kernel → Plugin Host → plugin.fetchFull(ref) → source (live)
    ↓
Kernel → UI: full content payload + deepLink(ref)
    ↓
UI renders source-specific detail view + "Open in [source]" link (opens external app/browser)
```

### Key Data Flows

1. **Metadata sync (cheap, scheduled):** plugin → correlation engine → index store. Runs continuously in background; this is what makes the stream feel instant.
2. **Full-content fetch (expensive, on-demand):** kernel → plugin → source, triggered only by user action (opening detail pane). Never cached beyond the current view — re-fetched each open, satisfying "no duplication, no staleness."
3. **Deep-link resolution (cheap, computed):** plugin → kernel → UI, a pure function of the item's source-native identifier; no network call needed since it's just URL construction (though correctness depends on knowing each source's UI URL scheme — see Integration Points below, some sources like Signal/WhatsApp cannot deep-link to a specific message, only to the app/thread).

## Scaling Considerations

This is explicitly a single-user, single-machine, personal tool (Out of Scope: cloud/multi-user). "Scale" here means *data volume and source count*, not concurrent users.

| Scale | Architecture Adjustments |
|-------|--------------------------|
| 5 sources, single mailbox+chats, months of history | Current design as-is: SQLite index, in-process plugins, synchronous per-plugin sync loop |
| 5-15 sources (later filesystem/other plugins added), years of history | Index Store needs indexes on (webspace_id, timestamp) and (source, native_id) for dedup; sync scheduler needs per-plugin concurrency (don't block IMAP sync on a slow paperless-ngx call) |
| Very large mailboxes/chat histories (10k+ items per source) | Cursor-based incremental enumerate (not full re-scan) becomes mandatory, not optional — mirrors Onyx's `PollConnector` (time-range incremental) vs `LoadConnector` (full bulk, only for first sync) distinction |

### Scaling Priorities

1. **First bottleneck:** full re-enumeration on every sync tick (re-scanning entire IMAP folders or chat DBs each cycle). Fix: every plugin must support an incremental mode keyed by a cursor (UIDVALIDITY/UID for IMAP, rowid/timestamp for local SQLite reads, `modified` timestamp for paperless-ngx/SilverBullet APIs) — do this from day one per plugin, not retrofitted later, since retrofitting a stateful cursor into a plugin contract after three plugins exist is exactly the kind of rewrite this research is meant to prevent.
2. **Second bottleneck:** live-fetch latency for remote LAN sources when a user rapidly clicks through many items. Fix: kernel-side short-lived (session-only, not persisted) in-memory cache of the last N fetched items is acceptable and doesn't violate "no duplication into the index" since it never touches the Index Store.

## Anti-Patterns

### Anti-Pattern 1: Letting plugins write directly to a shared "documents" table with source-specific columns

**What people do:** Add nullable columns to one big table per source's quirks (e.g., `email_subject`, `chat_sender`, `doc_tags`) as new plugins are added.
**Why it's wrong:** Every new source plugin requires a kernel schema migration, defeating the "third-party can add a plugin later" constraint and coupling the kernel to source specifics it should never know about.
**Instead:** One normalized `Item` schema (see below) with a `type`-discriminated `metadata` JSON blob for source-specific fields. The kernel only ever reads the normalized fields for stream/filter/correlate; source-specific fields are opaque to the kernel and only interpreted by the matching plugin and the UI's source-specific renderer.

### Anti-Pattern 2: Correlation engine calling out to live plugin APIs at query time

**What people do:** Compute "does this item match webspace X" on every UI request by asking the plugin live (e.g., re-checking IMAP folder membership on each page load).
**Why it's wrong:** Makes the stream only as fast as the slowest, most-remote plugin; reintroduces exactly the "visit each app" latency problem this project exists to remove.
**Instead:** Correlation is resolved once at sync time and persisted as a webspace assignment on the Item row (Pattern 2 above). Query time only ever reads the Index Store.

### Anti-Pattern 3: Treating "local DB read" plugins (Signal/WhatsApp) the same as "remote API" plugins for locking/concurrency

**What people do:** Assume all plugins are safe to poll concurrently and aggressively, since remote APIs tolerate it.
**Why it's wrong:** Signal Desktop's `db.sqlite` is a live SQLCipher database the Signal Desktop app itself may hold open; concurrent/careless access risks lock contention or (worse, given the read-only constraint) accidental writes if the wrong driver/mode is used.
**Instead:** Local-DB plugins must open in explicit read-only mode (e.g., SQLite URI `?mode=ro` / `immutable=1`, or copy-then-read from a snapshot if the driver doesn't support safe concurrent read-only access) and treat "source app might be running and holding the file" as a first-class condition, not an edge case. This is a plugin-contract-level requirement, not something to leave to each plugin author's discretion — the contract should document the expected access mode explicitly for file/DB-backed plugins.

### Anti-Pattern 4: One-size-fits-all plugin capability assumption

**What people do:** Design the plugin contract assuming every plugin can equally "enumerate by native category," "search full text," and "live-fetch cheaply."
**Why it's wrong:** Signal/WhatsApp local reads are cheap for *both* enumerate and full-fetch (same DB), while IMAP/paperless-ngx/SilverBullet are cheap to enumerate (metadata endpoints) but comparatively expensive to full-fetch (body/attachment/page content) over LAN. Baking in one cost model breaks UI assumptions (e.g., prefetching full content "for free").
**Instead:** Plugin contract should let each plugin declare capability/cost hints (e.g., `fetchCost: "local" | "remote"`), which the kernel/UI can use to decide things like prefetch-on-hover behavior.

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| Proton Mail Bridge (IMAP) | Standard IMAP client (folders = Proton labels), live network calls, poll via UID/UIDVALIDITY cursor | Bridge binds to 127.0.0.1 by default and uses a self-signed cert — per PROJECT.md this must be reconfigured to listen on LAN or tunnelled; plugin must accept a custom/self-signed cert config, not assume public CA trust. No universal deep-link to open one specific message in a webmail client by Message-ID; deep link options are a `mailto:` (opens compose, not view) or a link into Proton webmail's search-by-subject/date if that's reachable — flag as a real limitation, not solved by architecture alone. |
| Signal Desktop local DB | Direct SQLCipher-encrypted SQLite read, read-only mode, key sourced from local `config.json` (same as Signal Desktop app does) | No official read API. No deep-link scheme exists for jumping to a specific message; deep link is best-effort "open Signal Desktop" (app-level, not thread/message-level). This is the highest-uncertainty plugin (undocumented DB schema, subject to change on Signal Desktop upgrades) — flag for deeper phase-specific research and a schema-version detection/fallback in the plugin. |
| WhatsApp local store | Local read of desktop/linked-device store (mechanism less standardized than Signal's; may require a library such as `whatsmeow`-style local session parsing) | Confirm concretely in a dedicated research/spike phase before committing to implementation — PROJECT.md itself flags this as the riskiest area. Same deep-link limitation as Signal (no message-level deep link). |
| paperless-ngx REST API | `GET /api/documents/?tags__id=<tag>` (or similar filter) for enumerate, `GET /api/documents/<id>/` + `/metadata/` for full fetch, `GET /api/documents/<id>/preview/` for inline preview | Live LAN call, straightforward token-authenticated REST; UI deep link is the app's own document-detail SPA route (commonly `/documents/<id>/details` — verify exact route against the deployed version at implementation time, MEDIUM confidence here since it wasn't directly confirmed from source). |
| SilverBullet HTTP API | `GET /index.json` for enumerate (full file listing + metadata, filter by tag client-side or via space-script query), `GET /.fs/<path>` for full page content | Live LAN call; deep link is simply `http://<silverbullet-host>/<page-name>` since SilverBullet serves pages directly by name — one of the cleanest deep-link stories of the five sources. |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| UI ↔ Kernel HTTP API | HTTP/JSON, read-only, paginated | UI never talks to plugins or source systems directly — this boundary is what keeps "view-only, no mutation" enforceable in one place (the kernel simply never exposes a write endpoint). |
| Kernel ↔ Plugin Host | In-process function calls (if plugins are compiled/linked in) or local IPC (Unix socket/stdio JSON-RPC) if plugins run as separate processes | Given "third parties can add plugins later" constraint, favor a process-isolation boundary (like Home Assistant's custom component loading, or a gRPC-per-plugin model) over tight in-process coupling, so a misbehaving/crashing plugin (most likely candidate: Signal/WhatsApp local DB parsing) cannot take down the kernel or other plugins. Trade-off: added serialization overhead is negligible at this data scale. |
| Plugin Host ↔ Index Store | Plugins never write to the Index Store directly — they return data to the kernel, which normalizes and writes it | Preserves the single-schema-owner rule (Anti-Pattern 1) and lets the kernel enforce the read-only guarantee even if a plugin's internal logic is buggy. |
| Correlation Engine ↔ Webspace Config | Config is read-only input to correlation; correlation never mutates config | Config changes (adding/editing a webspace) should trigger re-correlation of already-synced items, not just apply going forward — otherwise editing a webspace definition silently orphans historical matches. |
| Sync Scheduler ↔ Plugin Host | Scheduler owns *when*; Plugin Host owns *how* | Scheduler should be source-agnostic (doesn't know IMAP from SQLite) — it only knows "this plugin's coordinator is due for a tick" and per-plugin backoff/error state. |

## Suggested Build Order

Dependency-driven, not feature-driven — each step is unblocked by the one before it and produces something independently testable:

1. **Normalized Item schema + Index Store** (no plugins needed yet). Design the schema first since every other component depends on its shape; validate it against all five sources' known metadata on paper before writing plugin code.
2. **Plugin Contract (interface only) + Plugin Host skeleton**, proven with one trivial fake/mock plugin (e.g., a static-fixture plugin returning canned items). This lets the kernel's correlation, sync, and API layers be built and tested without depending on any real, flaky external system (IMAP/Signal/etc.) yet.
3. **Webspace Config + Correlation Engine**, tested against the mock plugin. Correlation logic is pure and source-agnostic — get it right before real sources introduce noise.
4. **Sync Scheduler (coordinator-per-plugin)**, still against the mock plugin, proving the incremental-cursor model end-to-end.
5. **HTTP API (read-only) + minimal UI (stream only, no detail pane)**, against the mock plugin — proves the full request flow before any real source complexity.
6. **First real plugin: paperless-ngx or SilverBullet** (pick whichever is lower-risk — both are simple authenticated REST/HTTP APIs over LAN with no local-file complications) to validate the live-fetch + deep-link parts of the contract against a real, well-documented system before tackling the two hard ones.
7. **Email (IMAP) plugin** — introduces folder/label-based enumerate, UID cursor tracking, and the harder deep-link problem (no clean message-level deep link) — do this once the contract has already proven itself on an easier source.
8. **Signal plugin** — the highest-risk plugin (undocumented local DB, encryption key handling, read-only access safety); tackle only after the contract, sync model, and read-only-access pattern (Anti-Pattern 3) are already proven elsewhere, so any local-DB-specific problems are isolated to this one component.
9. **WhatsApp plugin** — same risk class as Signal, likely benefits from patterns/lessons learned building the Signal plugin (both are "read a live local store the source app also has open" problems); do last.
10. **Detail pane with source-specific renderers + full deep-link wiring across all five sources** — layered on top once every plugin's `fetchFull`/`deepLink` are implemented; this is a UI-only step that doesn't block or get blocked by later plugin work.

This ordering front-loads the kernel/contract (steps 1-5) behind a mock plugin specifically so that plugin risk (steps 6-9, especially 8-9) is isolated and doesn't block or corrupt validation of the kernel itself — matching how Home Assistant, Onyx, and Timelinize all separate a stable core from a wide, independently-evolving set of source integrations.

## Sources

- [timelinize/timelinize (GitHub)](https://github.com/timelinize/timelinize) — HIGH
- [Timelinize Wiki: Data Sources](https://github.com/timelinize/timelinize/wiki/Data-Sources) — HIGH (FileImporter `Recognize`/`FileImport` contract, checkpointed imports)
- [Timelinize Wiki: Schema](https://github.com/timelinize/timelinize/wiki/Schema) — HIGH (Item/Entity/Relationship/attribute model, cross-source dedup via identifying attributes)
- [Timelinize Wiki: Develop](https://github.com/timelinize/timelinize/wiki/Develop) — MEDIUM (build-only content, limited architecture detail)
- [onyx-dot-app/onyx connectors README](https://github.com/onyx-dot-app/onyx/blob/main/backend/onyx/connectors/README.md) — HIGH (LoadConnector/PollConnector/SlimConnector contract)
- [Onyx Documentation: Connectors](https://docs.onyx.app/overview/core_features/connectors) — MEDIUM
- [Home Assistant Developer Docs: Devices and Services](https://developers.home-assistant.io/docs/architecture/devices-and-services/) — HIGH (integration/config-entry/entity-platform/registry boundaries)
- [Home Assistant Developer Docs: Fetching Data (DataUpdateCoordinator)](https://developers.home-assistant.io/docs/integration_fetching_data/) — HIGH
- [Home Assistant core: update_coordinator.py](https://github.com/home-assistant/core/blob/dev/homeassistant/helpers/update_coordinator.py) — HIGH (source of truth for coordinator pattern)
- [Paperless-ngx REST API docs](https://docs.paperless-ngx.com/api/) — HIGH (`/api/documents/<id>/`, `/metadata/`, `/preview/` endpoints)
- [SilverBullet HTTP API](https://silverbullet.md/HTTP%20API) — HIGH (`GET /.fs/*`, `GET /index.json`, page-by-name URL serving)
- Signal Desktop local DB (SQLCipher, key in `config.json`, forked `better-sqlite3`) — MEDIUM (cross-referenced across multiple independent forensics/community writeups, no single official source since it's undocumented by Signal)
- WhatsApp local store access — LOW (no authoritative source found in this pass; PROJECT.md itself correctly flags this as the riskiest, least-validated area — recommend a dedicated spike/research phase before committing to a design)

---
*Architecture research for: local-first personal cross-source data aggregation (kernel + plugin)*
*Researched: 2026-07-27*
