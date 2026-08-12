# Phase 2: Two Sources, One Trustworthy Stream - Research

**Researched:** 2026-07-28
**Domain:** Go kernel sync/health/permission architecture + SilverBullet HTTP source plugin + Svelte UI (filter/health/staleness)
**Confidence:** MEDIUM-HIGH (kernel/Go findings HIGH — grounded directly in this repo's Phase 1 code; SilverBullet HTTP API findings MEDIUM — no official OpenAPI spec exists, cross-checked across three independent sources)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**SilverBullet in the stream**
- **D-01:** One SilverBullet page = one stream item. `source_id` is the page path (without `.md`). Deep link is **exact** fidelity to `{base_url}/{page-path}`.
- **D-02:** Chronological order is driven by the page's **last-modified** timestamp (from the `/.fs` listing metadata) — SilverBullet does not reliably expose creation time. — Reversible, but re-ordering requires a full re-sync.
- **D-03:** Keyword matching (exact, case-insensitive per Phase 1 D-03) runs against **both** page tags (frontmatter `tags:` and inline `#tag`) **and** page name — matched against the final path segment and the full path, extension stripped. Either match includes the page.
- **D-04:** Stream preview per the hybrid model: page name as title, frontmatter-stripped plaintext snippet, tags rendered as badges (same treatment as paperless tags). Detail pane renders **sanitized rendered markdown** fetched live via `Fetch`; a raw-markdown `ContentVariant` may be offered if cheap, but rendered is the default.

**Refresh & sync cadence**
- **D-05:** A background sync scheduler replaces the Phase 1 startup-only trigger: global `sync_interval` in TOML (default `15m`) with optional per-source override; startup sync is retained as the first scheduled run. — Reversible.
- **D-06:** The per-plugin coordinator is **single-flight**: a refresh request while that plugin is already syncing coalesces into (and reports) the in-flight run — never queued, never parallel. This is the single source of truth for health and sync state that Phases 3–5 plugins inherit. — Costly to reverse: every later plugin builds on the coordinator's semantics.
- **D-07:** Manual refresh is exposed both **per source** (from the health UI) and as **refresh-all** on the webspace header, via a kernel HTTP endpoint (POST) that returns the sync-run status. While in flight, the UI shows a non-blocking syncing indicator (driven by `sync_runs.status = "running"`); the stream re-fetches on completion.

**Health, filter & staleness UI**
- **D-08:** Per-source health lives in the **webspace header** as compact status chips (colored dot + source name; green = ok, amber = stale/last error, red = unreachable), with a tooltip showing last-sync relative time and last error, and the per-source refresh action. Reuses the existing badge/tooltip components. Health is a kernel-side merge of the plugin `Health` RPC and `sync_runs` history.
- **D-09:** Source filter is a chip row in the header — "All" plus one chip per source, single-select toggle back to All. Filter state persists in the URL query (e.g. `?source=silverbullet`) so reloads and deep links preserve it.
- **D-10:** Staleness semantics:
  - **Source unreachable** → that source's stream rows carry a subtle stale indicator; the detail pane still renders the indexed preview with an explicit alert: source unreachable, showing last-synced preview. Never a blank pane.
  - **Item deleted at source** → live fetch returns `content_unavailable`; detail pane shows an explicit "no longer available at source" state over the cached preview. The item disappears from the stream at the next successful sync; it is never silently 404'd while indexed.
  - Phase 1's rule carries forward: a failed sync must never render as an empty webspace.

**Agent permission config**
- **D-11:** TOML grant shape, per source, default-deny by absence:
  ```toml
  [sources.silverbullet.agent]
  read = true
  handoff = false
  ```
  Absent block or absent key = `false`.
- **D-12:** "Agent-facing API" becomes a **separate route namespace** (`/agent/v1/...`) mirroring the read API, with grants enforced structurally: ungranted sources are absent from source/health listings, their items are absent from streams, and direct item/content requests return the same error as a nonexistent item (no existence leak). The UI's existing `/api/*` routes are unaffected and remain grant-free for the human user. `docs/api.md`'s Phase-1 statement "there is no separate agent API" must be updated. `handoff` is recorded and exposed as capability metadata only; actual agent actions are v1.x (AGENT-11). — Costly to reverse: the namespace becomes a published surface agents build against.

### Claude's Discretion
- Reference mock plugin design and the PLUG-05 validation exercise (fresh-context build from `.proto` + `docs/plugin-contract.md` + mock alone).
- Exact snippet lengths, health-chip copy, error wording, sanitizer choice for rendered markdown.
- SilverBullet auth handling (`SB_AUTH_TOKEN` bearer via env interpolation) and confirming the exact `/.fs` metadata shape during planning.

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| KERN-04 | Sync scheduler with per-plugin coordinator (dedups refreshes, tracks health); manual refresh | See "Critical Architecture Finding" + "Standard Stack" (singleflight, ticker) + Code Examples |
| PLUG-04 | Plugins report health (reachable, last sync, last error) to the kernel | `Health` RPC already exists (Phase 1) — see Architecture Patterns "Health merge" |
| PLUG-05 | Third party can build a plugin from contract docs + reference mock plugin alone | See Architecture Patterns "PLUG-05 validation via isolated mock plugin" |
| SRC-05 | SilverBullet plugin; matches on tags/pages; exact deep links | See SilverBullet HTTP API findings, Code Examples, Common Pitfalls |
| UI-02 | Filter stream by source | See Architecture Patterns "Client-side source filter" |
| UI-05 | Stale/unavailable items show explicit state | See D-10 + Common Pitfalls "Partial-source-failure persistence bug" |
| UI-06 | Sync status and plugin health visible in UI | See D-08 + Architecture Patterns "Health merge" |
| AGENT-01 | Per-plugin permission model, default-deny | See Architecture Patterns "Agent namespace and grant filtering" + Security Domain |
</phase_requirements>

## Summary

Phase 2 is less about new external technology and more about a structural correction to code Phase 1 built for exactly one source. The single biggest research finding is that **Phase 1's sync/persist path cannot survive a second source without a fix**: `correlate.Engine.SyncAll` skips persisting a webspace's items entirely if *any* configured source's `Match` call errors, and `ReplaceWebspaceItems` always replaces the *whole* item set for a webspace regardless of which source triggered the sync. With one source these two facts happened to look correct (a failing sync just left the old item set stale-but-intact). With two sources they are actively wrong: a healthy source's freshly-matched items get thrown away whenever the other source is unreachable, and there is no way to refresh "just SilverBullet" without also re-running (and being blocked by) paperless. This is the concrete shape KERN-04's "per-plugin coordinator" must take, and it must be fixed as part of this phase, not layered on top of the existing bug.

The second-largest finding is that SilverBullet's HTTP API has **no server-side tag or name filter** — unlike paperless-ngx's `tags__id__in` query, `GET /.fs` returns a flat JSON array of every file's metadata (`name`, `created`, `lastModified`, `contentType`, `size`, `perm`) and nothing else; matching by tag requires the plugin to read every candidate page's raw body (frontmatter + inline `#tags`) itself. This is a real, accepted-for-MVP performance tradeoff (bounded-concurrency fetch of the whole space on every sync), not a bug to design around this phase.

Everything else is standard, well-trodden Go: `golang.org/x/sync/singleflight` for the per-plugin single-flight coordinator, a plain `time.Ticker`-based scheduler (not `robfig/cron`, whose spec-parsing power this project doesn't need and whose last tag is over five years old), `goldmark` + `bluemonday` for sanitized markdown-to-HTML rendering served through the existing PDF-iframe UI pattern, and `adrg/frontmatter` (already the documented stack choice) for tag extraction.

**Primary recommendation:** Fix the sync/persist path to operate per-`(webspace, source_type)` before adding SilverBullet — build the coordinator as the thing that calls this corrected path, not a wrapper around the existing one.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| SilverBullet source plugin (SRC-05) | Plugin subprocess (Backend) | — | Mirrors `plugins/paperless`: its own Go module, gRPC subprocess, no kernel awareness of its transport |
| Sync scheduler + per-plugin coordinator (KERN-04) | Kernel/Backend | — | Owns timing, single-flight, and the corrected per-source persistence path; no UI or plugin-side involvement |
| Health reporting (PLUG-04) | Plugin subprocess (source) + Kernel (merge) | Frontend (display only) | Plugin's `Health` RPC supplies live reachability; kernel merges with its own `sync_runs` history — the plugin never sees or reports the merged view |
| Health chips + tooltip (UI-06, D-08) | Frontend | Kernel (new `GET /api/sources` route) | Pure display of kernel-merged data; no client-side merge logic |
| Source filter (UI-02, D-09) | Frontend | — | Filters the already-fetched stream response client-side by `source_type`; URL query param owns persisted state. No new backend query parameter needed — the whole webspace's items are already one response |
| Staleness / unavailable states (UI-05, D-10) | Frontend (render) | Kernel (per-source health + `content_unavailable` semantics) | Kernel supplies the signals (`sync.status`, per-source health, `content_unavailable`); frontend owns how they render |
| Agent permission grants (AGENT-01) | Kernel/Backend | Config | New `/agent/v1` route namespace + config-driven filtering; no plugin or frontend involvement — plugins don't know grants exist |
| Rendered-markdown sanitization (D-04) | Plugin subprocess (SilverBullet) | Kernel (MIME allowlist extension) | Sanitize at the source of the untrusted content (the plugin), not in the kernel or frontend — matches the existing "kernel never re-parses source-specific formats" boundary |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `golang.org/x/sync/singleflight` | v0.22.0 [VERIFIED: proxy.golang.org, checked 2026-07-28] | Per-plugin single-flight coordinator (D-06) | Official Go extended-stdlib package; `Group.Do(key, fn) (v, err, shared)` is exactly the "coalesce concurrent identical work" primitive D-06 describes — no reason to hand-roll a mutex+map version of this |
| `github.com/yuin/goldmark` | v1.8.5 [VERIFIED: proxy.golang.org, checked 2026-07-28] | Markdown → HTML rendering (D-04) inside the SilverBullet plugin | The de facto standard pure-Go CommonMark renderer (used by Hugo); does not render raw HTML or dangerous URL schemes by default |
| `github.com/microcosm-cc/bluemonday` | v1.0.27 [VERIFIED: proxy.golang.org, checked 2026-07-28] | HTML sanitization of goldmark's output before it ever reaches the kernel/UI | OWASP-Java-Sanitizer-inspired; `UGCPolicy()` is the documented pairing for "markdown conversions" per its own README — defense in depth alongside goldmark's own safe defaults |
| `github.com/adrg/frontmatter` | v0.2.0 [VERIFIED: proxy.golang.org, checked 2026-07-28] | Extract YAML frontmatter (incl. `tags:`) from SilverBullet page bodies (D-03) | Already the documented stack choice (`.claude/CLAUDE.md` Supporting Libraries) — `frontmatter.Parse(io.Reader, &struct) (rest []byte, err error)` |
| `time.Ticker` (stdlib) | Go 1.25 stdlib | Background sync scheduler (D-05) | See "Alternatives Considered" — `robfig/cron`'s full cron-spec parser is unneeded power for "every N minutes, optional per-source override"; a ticker loop matches this project's established preference for hand-rolled-over-heavy-dependency (TOML+`os.Expand` over Viper, hand-rolled REST clients over SDKs) |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/robfig/cron/v3` | v3.0.1 [VERIFIED: proxy.golang.org] | Alternative scheduler | Only if a future phase needs real cron-spec scheduling (e.g. "sync IMAP only during working hours") — not needed for D-05's flat interval requirement |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `time.Ticker` scheduler | `robfig/cron/v3` | Cron gives arbitrary schedule expressions and jitter-free wall-clock alignment; costs an extra dependency and a spec-parsing failure mode for a requirement that's just "every `sync_interval`" — last tagged release is Jan 2020, so it's stable but not actively evolving |
| `goldmark` + `bluemonday` | Render nothing server-side; ship raw markdown text and let the SPA render it client-side (e.g. `marked` + `dompurify` in JS) | Moves the sanitization boundary into the browser bundle and duplicates it per-source (email/chat will also need rendering later); D-04 explicitly asks for the *plugin* to hand back rendered content, keeping the kernel/UI source-format-agnostic — server-side rendering is the locked default |
| `adrg/frontmatter` | Hand-rolled `strings.Cut` on `---` delimiters + `gopkg.in/yaml.v3` unmarshal | Viable and even simpler for YAML-only frontmatter, but `adrg/frontmatter` is already the documented stack choice — don't re-litigate it here |

**Installation:**
```bash
go get golang.org/x/sync@v0.22.0
go get github.com/yuin/goldmark@v1.8.5
go get github.com/microcosm-cc/bluemonday@v1.0.27
go get github.com/adrg/frontmatter@v0.2.0
```

**Version verification:** All four versions above were confirmed live against `proxy.golang.org`'s module version list (`curl https://proxy.golang.org/<module>/@v/list`) on 2026-07-28 — this is the Go module proxy itself, not a training-data guess or a WebSearch summary.

## Package Legitimacy Audit

> The `gsd-tools query package-legitimacy check` seam only supports `npm|pypi|crates` ecosystems; this phase's new dependencies are all Go modules, so legitimacy was assessed manually against each module's publish history on `proxy.golang.org` (a long, continuous version history with no re-published/yanked-then-reused version numbers is the Go-ecosystem equivalent signal to npm download counts/age).

| Package | Registry | Age / History | Downstream Usage | Source Repo | Verdict | Disposition |
|---------|----------|----------------|-------------------|--------------|---------|-------------|
| `golang.org/x/sync` | Go module proxy | Official `golang.org/x/*` extended-stdlib module; version history back to v0.1.0 | Imported by a large fraction of the Go ecosystem | github.com/golang/sync | OK | Approved |
| `github.com/yuin/goldmark` | Go module proxy | 60+ published versions, active up to v1.8.5 (published day of this research) | Standard dependency of Hugo and many Go static-site/markdown tools | github.com/yuin/goldmark | OK | Approved |
| `github.com/microcosm-cc/bluemonday` | Go module proxy | Continuous version history from v1.0.1 through v1.0.27 (2024) | Widely used Go HTML sanitizer, referenced by OWASP-adjacent tooling comparisons | github.com/microcosm-cc/bluemonday | OK | Approved |
| `github.com/adrg/frontmatter` | Go module proxy | v0.1.0 → v0.2.0 (2020); already this project's documented stack choice per `.claude/CLAUDE.md` | Small but stable, single-purpose library | github.com/adrg/frontmatter | OK | Approved (pre-approved by project stack decision, not a new discovery this phase) |
| `github.com/robfig/cron/v3` | Go module proxy | v3.0.0-rc1 → v3.0.1 (2020); de facto standard Go cron library, referenced in the project's own "Alternatives Considered" above | Not adopted this phase (alternative only) | github.com/robfig/cron | OK (not used) | Not installed — documented as an alternative only |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none
**Postinstall scripts:** Go modules have no npm-style postinstall hook mechanism; N/A.

## Critical Architecture Finding: partial-source-failure persistence bug

This is the load-bearing finding for KERN-04/UI-05/D-10 and must shape the plan's task structure, so it is called out ahead of the general pattern catalogue below.

**What's broken today** (`kernel/correlate/correlate.go`, `SyncAll`):

```go
for name, ws := range e.Config.Webspaces {
    var items []item.Item
    var firstErr error
    for _, src := range e.Sources {
        resp, err := src.Match(ctx, ws.Keywords)
        if err != nil {
            firstErr = ...   // records the error but the loop continues
            continue         // to try the OTHER sources
        }
        // ... items = append(items, matched items from this source)
    }
    if firstErr != nil {
        results = append(results, WebspaceResult{Webspace: name, Err: firstErr})
        continue   // <-- Store.ReplaceWebspaceItems is never called at all
    }
    e.Store.ReplaceWebspaceItems(ctx, name, items)  // replaces the WHOLE webspace's items
}
```

With one source (Phase 1), "any source fails" and "the only source fails" are the same event, and skipping `ReplaceWebspaceItems` entirely was the *correct* behavior — it left the previous sync's item set stale-but-intact (satisfying "a failed sync must never render as an empty webspace"). With two sources this silently breaks in two ways:

1. **A working source's fresh items are discarded.** If SilverBullet is unreachable and paperless is fine, `firstErr` is still non-nil, the whole `continue` fires, and paperless's newly-matched items for that sync cycle are thrown away — the webspace shows *stale paperless data* even though paperless itself is perfectly healthy. This directly contradicts D-10 ("Source unreachable → **that source's** rows carry a stale indicator", implying the other source's rows do not).
2. **There is no way to refresh one source independently.** `ReplaceWebspaceItems(ctx, webspaceName, items)` deletes *every* `webspace_items` row for that webspace and reinserts the full merged set — it has no concept of "only touch this source's rows." A per-source manual refresh (D-07) cannot be built on top of this method without wiping the other source's already-correct rows during the window between delete and reinsert.

**Recommended fix** (shape, not full implementation — planner's call on exact package layout):

- Restructure the sync loop from *webspace-major, all-sources-required* to *source-major, independent-per-source persistence*: for each source, run `Match` against every webspace's keywords, and persist that source's contribution to each webspace **independently** of whether any other source succeeded or failed this cycle.
- Add a source-scoped replace method to `kernel/index/store.go`, e.g.:
  ```go
  // ReplaceWebspaceSourceItems replaces only the rows in webspace_items for
  // (webspaceName, sourceType) — it must never delete another source's rows
  // for the same webspace.
  func (s *Store) ReplaceWebspaceSourceItems(ctx context.Context, webspaceName, sourceType string, items []item.Item) error
  ```
  implemented with a source-scoped delete before reinsert, in one transaction:
  ```sql
  DELETE FROM webspace_items
  WHERE webspace_name = ?
    AND item_id IN (SELECT id FROM items WHERE source_type = ?)
  ```
- `webspaces.synced_unix` (the "is this webspace known at all" registry row used by `WebspaceExists`) should be marked on **any** source's first successful contribution, not require every configured source to have synced at least once — otherwise a webspace with a working paperless source and a not-yet-configured/still-erroring SilverBullet source would incorrectly 404 as "never synced."
- The existing whole-webspace `ReplaceWebspaceItems` either goes away entirely in favor of the source-scoped version, or is kept only as a Phase-1-compatibility helper that calls the new one once per distinct `source_type` present in its input — the planner should pick whichever is less code, but the *coordinator* must call the source-scoped path.
- `sync_runs` already anticipates this: its `status` column comment lists `"running" | "ok" | "error"` (Phase 1 schema, unused "running" value) — the coordinator should insert a `status="running"` row when a source's sync starts and update that same row to `"ok"`/`"error"` with `finished_unix` set on completion, rather than only ever inserting a single post-hoc row as Phase 1's `RecordSyncRun` does today. This is what makes D-07's "non-blocking syncing indicator" and D-08's health merge possible without new schema.

## Architecture Patterns

### System Architecture Diagram

```
                         ┌─────────────────────────────────────────────┐
                         │  Scheduler (time.Ticker, per configured     │
                         │  sync_interval, one goroutine per source)   │
                         └───────────────────┬───────────────────────-┘
                                             │ calls
                                             ▼
  HTTP POST /api/sources/{name}/refresh ──▶ ┌────────────────────────────┐
  HTTP POST /api/sync (refresh-all)     ──▶ │  Coordinator.Refresh(name) │
                                             │  singleflight.Group.Do(   │
                                             │    key=name, fn=syncOne)  │
                                             └───────────┬────────────────┘
                                                         │ per source, per webspace
                                                         ▼
                                          ┌───────────────────────────────┐
                                          │ Plugin.Match(keywords)         │
                                          │ (unchanged gRPC call, per-src) │
                                          └───────────────┬───────────────┘
                                                         │ items + err
                                                         ▼
                                     ┌─────────────────────────────────────┐
                                     │ index.ReplaceWebspaceSourceItems(    │
                                     │   webspace, source_type, items)      │
                                     │ index.StartSyncRun / FinishSyncRun   │
                                     └───────────────────┬───────────────────┘
                                                         │
      GET /api/webspaces/{ws}/stream (unchanged) ◀───────┘ (pure index read)
      GET /api/sources (NEW: health merge)         ◀── Plugin.Health(ctx) [live] + sync_runs [history]
      GET /agent/v1/... (NEW: grant-filtered mirror of /api/*)

  Browser SPA:
    WebspaceHeader ── health chips (GET /api/sources) + filter chips (client-side, URL ?source=)
    StreamList ── unchanged sync-failure/empty/populated branches, now per-item stale badge
    DetailPane ── unchanged two-stage render; unavailable/stale alert branch added (D-10)
```

### Client-side source filter (UI-02, D-09)

**What:** The SPA already fetches the *entire* webspace's item list in one `GET /api/webspaces/{ws}/stream` call (Phase 1). D-09's "All ↔ single source" filter does **not** need a new backend query parameter — it's a pure client-side `Array.prototype.filter(item => item.source_type === selected)` over the response already in memory, with `selected` read from/written to the URL's `?source=` query param via SvelteKit's `page.url.searchParams`.

**When to use:** Any time the full dataset needed to answer a UI question is already in one response — filtering client-side avoids a round trip and keeps `kernel/httpapi/stream.go`'s "pure index read, no new params" contract simple.

**Example:**
```typescript
// web/src/routes/w/[webspace]/+page.svelte — reading/writing the filter
import { page } from '$app/state';
import { goto } from '$app/navigation';

let selectedSource = $derived(page.url.searchParams.get('source'));
let visibleItems = $derived(
  selectedSource
    ? response?.items.filter((i) => i.source_type === selectedSource)
    : response?.items
);

function setSourceFilter(sourceType: string | null) {
  const url = new URL(page.url);
  if (sourceType) url.searchParams.set('source', sourceType);
  else url.searchParams.delete('source');
  goto(url, { replaceState: true, keepFocus: true, noScroll: true });
}
```

### Health merge (PLUG-04, UI-06, D-08)

**What:** A new `GET /api/sources` kernel route returns, per configured source: `name` (config key), `source_type`/`display_name` (from `Describe`, already cached at plugin-launch time in `pluginhost.Plugin`), `reachable` (a **live** call to that plugin's `Health` RPC — the same "lightweight reachability probe" pattern paperless's own `Health()` already implements via `p.client.AllTags(ctx)`), and `last_sync_unix`/`last_error` (from the kernel's **own** `sync_runs` table, not the plugin's self-reported `HealthResponse.last_sync_unix`/`last_error` — the kernel is the single source of truth per D-06, and a plugin's own Health RPC is a reachability probe only).

**When to use:** Any time UI-06 or D-08 needs "is source X ok right now" (chip color) vs. "when did source X last successfully sync, and what broke last time" (tooltip detail) — these are two different data sources merged into one response shape, not one query.

**Example response shape:**
```json
{
  "schema_version": 1,
  "sources": [
    { "name": "paperless", "source_type": "paperless", "display_name": "paperless-ngx",
      "reachable": true, "syncing": false, "last_sync_unix": 1785000000, "last_error": "" },
    { "name": "silverbullet", "source_type": "silverbullet", "display_name": "SilverBullet",
      "reachable": false, "syncing": false, "last_sync_unix": 1784900000, "last_error": "dial tcp: connection refused" }
  ]
}
```

### PLUG-05 validation via isolated mock plugin

**What:** Build a small, separate `plugins/mock/` — an in-memory `sdk.SourcePlugin` implementation with zero real network dependency (a fixed, hand-written `[]*webspacesv1.Item` slice, deterministic `Health`) — as the concrete artifact for PLUG-05's validation exercise, distinct from the real (and necessarily more complex) SilverBullet plugin. The validation itself is a process step for the planner to schedule: hand a fresh context/agent exactly three inputs — `proto/webspaces/v1/plugin.proto`, `docs/plugin-contract.md`, and the `sdk` module — with **no** read access to `plugins/paperless/` or `plugins/silverbullet/`, and have it build a working plugin from those alone. `plugins/mock/` can either be the *output* of that exercise (kept in-repo as the proof artifact) or a small pre-built fixture the planner writes first and re-derives independently — either way, keep it separate from the real SilverBullet plugin so a gap in the contract docs surfaces as "the mock plugin builder got stuck," not "the SilverBullet plugin has a bug."

**When to use:** This is a documentation/contract-completeness test, not a feature — don't fold it into SRC-05's task list; give it its own plan task with its own pass/fail criterion ("a build with no access to `plugins/paperless` or `plugins/silverbullet` produces a working `SourcePlugin` implementation").

### Recommended Project Structure

```
plugins/
├── paperless/           # existing (Phase 1)
├── silverbullet/         # NEW — own go.mod, own module in go.work
│   ├── main.go           # plugin.Serve entrypoint (mirrors paperless/main.go)
│   ├── client.go          # hand-rolled GET-only /.fs HTTP client
│   ├── plugin.go          # sdk.SourcePlugin impl: Describe/Match/Fetch/Health
│   ├── frontmatter.go     # tag/name extraction (adrg/frontmatter + inline #tag scan)
│   ├── render.go          # goldmark + bluemonday sanitized-HTML rendering
│   ├── client_test.go
│   ├── outbound_hosts_test.go   # host-pinned egress test, mirrors paperless's
│   └── readonly_test.go         # AST walk: only GET requests, mirrors paperless's
└── mock/                 # NEW — PLUG-05 validation artifact, own go.mod
    ├── main.go
    └── plugin.go          # trivial fixed-item SourcePlugin, no network calls

kernel/
├── sync/                 # NEW package — scheduler + coordinator
│   ├── coordinator.go     # singleflight.Group keyed by source name; wraps corrected persist path
│   └── scheduler.go       # time.Ticker per source, calls coordinator.Refresh(name)
├── correlate/             # MODIFIED — source-major loop, calls index.ReplaceWebspaceSourceItems
├── index/                 # MODIFIED — new ReplaceWebspaceSourceItems, StartSyncRun/FinishSyncRun
├── httpapi/
│   ├── sources.go          # NEW — GET /api/sources, POST /api/sources/{name}/refresh, POST /api/sync
│   └── agent/              # NEW — /agent/v1/* handlers wrapping the same store/fetcher with grant filtering
└── config/                 # MODIFIED — [sync] interval, per-source sync_interval override, [sources.*.agent]
```

### Rendered-markdown-as-iframe pattern (D-04, MIME allowlist)

**What:** `kernel/httpapi/item.go`'s `/content` and `/thumbnail` routes currently enforce a **fixed MIME allowlist** (`application/pdf`, `image/png`, `image/jpeg`, `image/gif`, `image/webp` — see `docs/api.md`) before writing any byte, and every accepted response already carries `Content-Security-Policy: default-src 'none'; object-src 'none'; sandbox`. `DetailPane.svelte` already renders `application/pdf` content inside an `<iframe src={contentUrl(item.id)}>` rather than injecting it into the page DOM. **Reuse this exact pattern for SilverBullet's rendered markdown**: the plugin's `Fetch` sanitizes with `bluemonday` and returns `mime_type: "text/html"`; the kernel's MIME allowlist is extended to include `text/html`; the SPA adds one more `{:else if content.rendition?.mime_type === 'text/html'}` branch rendering the same `<iframe src={contentUrl(item.id)}>`. The CSP `sandbox` directive on that response is what makes the iframe boundary meaningful even though bluemonday already sanitized server-side — defense in depth, not redundant.

**Anti-pattern to avoid:** Do **not** inject the sanitized HTML into the page via Svelte's `{@html}` — that renders same-origin, same-CSP-context as the rest of the SPA, discarding the sandboxing the existing `/content` route's response headers already provide for free via the iframe boundary.

### PDF-style two-field Fetch response, applied to markdown

**Example** (mirrors `plugins/paperless/plugin.go`'s `fetchFull`):
```go
// plugins/silverbullet/plugin.go
func (p *SourcePlugin) fetchFull(ctx context.Context, path string) (*webspacesv1.FetchResponse, error) {
	raw, err := p.client.ReadFile(ctx, path) // GET /.fs/{path}
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "silverbullet: page %q not found", path)
		}
		return nil, status.Errorf(codes.Unavailable, "silverbullet: read %q: %v", path, err)
	}

	rest, tags := extractFrontmatterAndTags(raw) // adrg/frontmatter + inline #tag scan

	var buf bytes.Buffer
	if err := goldmark.Convert(rest, &buf); err != nil {
		return nil, status.Errorf(codes.Internal, "silverbullet: render %q: %v", path, err)
	}
	sanitized := bluemondayPolicy.SanitizeBytes(buf.Bytes())

	return &webspacesv1.FetchResponse{
		Available: true,
		MimeType:  "text/html",
		SizeBytes: int64(len(sanitized)),
		Text:      string(rest), // raw markdown, for a future raw-ContentVariant (D-04, optional)
		Data:      sanitized,
		Provenance: map[string]string{"source_type": sourceType, "source_id": path},
	}, nil
}
```

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Coalescing concurrent refresh requests for the same source (D-06) | A hand-rolled `map[string]*sync.Mutex` + in-flight tracking struct | `golang.org/x/sync/singleflight` | Exactly this problem, from the Go team, with `Do`/`DoChan`/`Forget` already covering the "coalesce, don't queue" and "allow a future call to bypass a forgotten key" cases D-06 and D-07 need |
| Markdown → sanitized HTML | A hand-rolled regex-based tag stripper | `goldmark` (safe-by-default rendering) + `bluemonday.UGCPolicy()` | XSS-in-a-personal-notes-app is a real risk (a note synced from an untrusted shared space, or copy-pasted HTML) — hand-rolled sanitization is the single most common source of sanitizer bypass bugs; both libraries are widely used specifically for this pairing |
| YAML frontmatter parsing | Hand-rolled `strings.Split` on `---` plus manual YAML field extraction | `adrg/frontmatter` (already the documented stack choice) | Already decided; re-deriving it here would contradict `.claude/CLAUDE.md` |
| Per-source, per-webspace persistence | A second parallel "SilverBullet-only" table/path bolted onto the existing webspace-major `ReplaceWebspaceItems` | Refactor to the source-scoped `ReplaceWebspaceSourceItems` (see Critical Architecture Finding) | Bolting on a parallel path for "the new source" duplicates the correctness bug for the next three sources (Phases 3–5) that inherit this coordinator — fix the general case once |

**Key insight:** Every "don't hand-roll" item above already has a locked answer either in this project's existing code (source-scoped persistence, sanitization pairing) or its documented stack (`adrg/frontmatter`) — the risk this phase carries is re-deriving a *different*, incompatible answer under time pressure, not lacking an answer.

## Common Pitfalls

### Pitfall 1: Partial-source-failure persistence bug (see Critical Architecture Finding, above)
**What goes wrong:** A healthy source's items get silently discarded whenever a *different* source fails during the same sync cycle, and a per-source manual refresh can't be built without risking clobbering the other source's rows.
**Why it happens:** Phase 1's `correlate.Engine.SyncAll` and `index.ReplaceWebspaceItems` were both written correctly for exactly one source and never revisited for N sources.
**How to avoid:** Restructure to source-major persistence (`ReplaceWebspaceSourceItems`) before wiring up the coordinator — see above for the concrete shape.
**Warning signs:** A UAT scenario where SilverBullet is deliberately taken offline should show paperless items still fresh and only SilverBullet-sourced rows/health marked stale; if the whole webspace goes stale, this bug wasn't fixed.

### Pitfall 2: SilverBullet has no server-side tag filter
**What goes wrong:** A plugin author reflexively looks for a `?tag=` or `?filter=` query parameter on `GET /.fs` (as paperless-ngx has for `tags__id__in`) and doesn't find one, then either gives up on tag matching or builds something fragile against an undocumented endpoint.
**Why it happens:** SilverBullet's HTTP surface is deliberately minimal — "mostly just to sync data," per its own community discussion — with no query/filter/search API at all as of this research.
**How to avoid:** Fetch the full `GET /.fs` listing once per sync, filter to markdown pages client-side (by `contentType`/`.md` suffix, excluding SilverBullet's own library/plug files), then fetch each candidate page's raw body via `GET /.fs/{path}` to extract frontmatter tags + inline `#tags` for matching. Bound the concurrency of these per-page fetches (e.g. a small worker pool / `errgroup.SetLimit`) so a large space doesn't open hundreds of simultaneous connections.
**Warning signs:** Sync time scaling linearly (or worse) with total space size rather than webspace-matched item count — expected and accepted for this phase's MVP scope, but worth a one-line note in the plan for future optimization (e.g. skip re-fetching a page whose `lastModified` hasn't changed since the previous sync).

### Pitfall 3: Hardcoded source name in shared UI copy
**What goes wrong:** `DetailPane.svelte`'s content-load-failure branch currently reads literally `"paperless-ngx didn't respond. It may be offline — try again, or open it directly in paperless-ngx."` — a SilverBullet failure would show this exact wrong copy verbatim.
**Why it happens:** Phase 1 had exactly one source, so the copy was (reasonably) written source-specific rather than parameterized.
**How to avoid:** Parameterize this string with the item's `source_type`/a display-name lookup (e.g. `"${displayName} didn't respond..."`), sourced from the same `GET /api/sources` display-name data the health chips use.
**Warning signs:** Any string literal containing "paperless" outside `plugins/paperless/` after this phase is a signal something wasn't generalized.

### Pitfall 4: MIME allowlist silently 415s the new rendered-markdown content
**What goes wrong:** `kernel/httpapi/item.go`'s `/content` route rejects any MIME type outside its fixed allowlist with `415 unsupported_rendition_type` — today that allowlist is `{application/pdf, image/png, image/jpeg, image/gif, image/webp}`, which does **not** include `text/html`. Shipping the SilverBullet plugin without updating this allowlist means every detail-pane open 415s.
**Why it happens:** The allowlist was sized for Phase 1's one source (documents/images only); it's easy to forget it needs revisiting for a source whose primary rendition type is different.
**How to avoid:** Add `text/html` to the allowlist explicitly, and update `docs/api.md`'s documented allowlist (a published-contract change, same category as the D-12 "no separate agent API" line that must be revised).
**Warning signs:** UAT step "open a SilverBullet page in the detail pane" returns a 415 instead of rendered content.

### Pitfall 5: `sync_runs.status = "running"` needs an update path, not just inserts
**What goes wrong:** `index.RecordSyncRun` today only ever `INSERT`s a single row per sync, with both `started_unix` and `finished_unix` already known — there's no method to insert a `"running"` row up front and later update it to `"ok"`/`"error"`. Building D-07's "non-blocking syncing indicator" against `sync_runs.status = "running"` requires this two-phase write.
**Why it happens:** Phase 1's schema comment already anticipated `"running"` as a valid status value, but the one method that writes to this table was never split into start/finish halves because Phase 1 only ever synced once at startup with no concurrent readers watching for "is it syncing right now."
**How to avoid:** Add `StartSyncRun(ctx, sourceType) (runID int64, err error)` (inserts with `finished_unix` NULL, `status="running"`) and `FinishSyncRun(ctx, runID, status, error, itemCount)` (updates the same row), and have `GET /api/sources`'s `syncing` field derive from `SELECT 1 FROM sync_runs WHERE source_type = ? AND status = 'running'`.
**Warning signs:** The "syncing" spinner never appears (because nothing ever writes a running row) or never clears (because nothing updates it).

## Code Examples

### Reading and filtering SilverBullet's file listing

```go
// plugins/silverbullet/client.go — GET /.fs returns []FileMeta.
// Field names confirmed against the SilverBullet Go server module's
// FileMeta struct (pkg.go.dev/github.com/silverbulletmd/silverbullet/server,
// pre-release Go rewrite) and the HTTP API doc's per-file response headers
// (X-Last-Modified, X-Created, X-Permission) — both agree on the same
// field set. [CITED: silverbullet.md/HTTP%20API, pkg.go.dev]
type FileMeta struct {
	Name         string `json:"name"`
	Created      int64  `json:"created"`      // Unix milliseconds
	LastModified int64  `json:"lastModified"` // Unix milliseconds — D-02's primary sort key
	ContentType  string `json:"contentType"`
	Size         int64  `json:"size"`
	Perm         string `json:"perm"` // "ro" | "rw"
}

func (c *Client) ListFiles(ctx context.Context) ([]FileMeta, error) {
	// GET {base_url}/.fs, Authorization: Bearer <SB_AUTH_TOKEN>
	// returns a flat JSON array — no query params, no pagination.
}

func isPage(f FileMeta) bool {
	return strings.HasSuffix(f.Name, ".md") && !strings.HasPrefix(f.Name, "_")
	// SilverBullet stores library/plug files under leading-underscore paths
	// (e.g. "_plug/*") — exclude these from webspace matching.
}
```

### Timestamp mapping (D-02)

```go
// FileMeta already carries both LastModified and Created — use the same
// primary/secondary pattern the paperless plugin uses (Created/Added),
// just with the roles reversed: LastModified is the primary sort key
// (D-02 requires it), Created is the secondary tie-break for same-
// last-modified pages (rare, but the schema's tie-break column expects a
// value regardless).
item.TimestampUnix = f.LastModified / 1000
item.SecondaryTimestampUnix = f.Created / 1000
```

### Deep link (D-01)

```go
sourceID := strings.TrimSuffix(f.Name, ".md")
deepLink := fmt.Sprintf("%s/%s", strings.TrimRight(baseURL, "/"), sourceID)
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| SilverBullet server was a single Deno/TypeScript process (`http_server.ts`) | A Go rewrite of the server exists as of a Feb 2026 pre-release module (`github.com/silverbulletmd/silverbullet/server`) | Confirmed via `pkg.go.dev`, published Feb 13 2026 [CITED: pkg.go.dev] | Doesn't change this plugin's design (it talks HTTP either way), but confirms the `/.fs` `FileMeta` shape is a stable, cross-implementation contract rather than an implementation detail of one server version — safe to build against |

**Deprecated/outdated:** None identified specific to this phase's scope.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `GET /.fs` returns a flat JSON *array* of `FileMeta` objects (not an object keyed by path, not paginated) | SilverBullet HTTP API findings, Code Examples | If it's actually keyed/paginated, the plugin's listing parse and the "fetch everything once per sync" cost model both need adjusting — low-cost to verify: one `curl` against a real instance during plan-time or Task 1 of implementation |
| A2 | There is no hidden/undocumented tag or name query parameter on `GET /.fs` or a search endpoint | Common Pitfalls #2 | If one exists, the "fetch every page body" cost (Pitfall 2) could be avoided entirely — worth a quick check against the user's actual SilverBullet instance/version before committing to the bounded-worker-pool full-space-read design |
| A3 | The user's SilverBullet instance exposes `SB_AUTH_TOKEN` bearer-token auth (not session-cookie-only) | User Constraints (Claude's Discretion), `.claude/CLAUDE.md` | If the instance requires session-cookie auth instead, the plugin needs a login flow rather than a static bearer token — should be confirmed against the user's actual instance during planning, per the roadmap's own note that this needs confirming |

**If this table is empty:** N/A — see above three assumptions, all flagged for a cheap, direct check against the user's real SilverBullet instance early in plan execution (Task 1 spike-style check, not a separate research pass).

## Open Questions

1. **Exact shape of `GET /.fs`'s top-level JSON envelope**
   - What we know: it's "a full listing... in JSON format" containing `FileMeta`-shaped entries per file (confirmed field names via the Go server module and the per-file HTTP header names, which agree).
   - What's unclear: whether the top-level value is a bare array or wrapped in an object (e.g. `{"files": [...]}`).
   - Recommendation: the plan's first SilverBullet-plugin task should include a real `curl` against the user's own instance (base URL + `SB_AUTH_TOKEN` from their actual config) as its first verification step, before writing the parsing code — this is a five-minute check that fully resolves A1.

2. **Whether `/agent/v1` needs its own error-envelope/schema-version namespace or reuses `docs/api.md`'s existing shapes verbatim**
   - What we know: D-12 says the namespace "mirrors the read API" structurally.
   - What's unclear: whether `schema_version` on `/agent/v1/*` responses is the same counter as `/api/*` or an independently-versioned one (mattering if `/api/*` and `/agent/v1/*` ever need to evolve their shapes independently).
   - Recommendation: reuse the exact same envelope and `schema_version` counter unless the planner identifies a concrete reason to split them — simpler, and nothing in this phase's scope requires independent versioning.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | All kernel/plugin work | ✓ | go1.26.5 (per `go version` in this environment; project targets 1.25+) | — |
| SilverBullet instance (LAN-reachable, per project constraints) | SRC-05 end-to-end testing | Not verified in this research session — no network path to the user's home-server instance from here | — | Plan must include a manual/human-verify checkpoint confirming reachability + `SB_AUTH_TOKEN` before the plugin can be UAT'd against the real instance |
| `proxy.golang.org` (module version verification) | Standard Stack version checks | ✓ | — | — |

**Missing dependencies with no fallback:** none blocking planning; the SilverBullet instance itself must be confirmed reachable before execution/UAT (already flagged as a `.planning` note: "confirm the exact `/.fs` metadata shape during planning").

**Missing dependencies with fallback:** none.

## Security Domain

### Applicable ASVS Categories (Level 1)

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | No — unchanged from Phase 1 | This API has no authentication in v1 (loopback-only trust boundary, documented in `docs/api.md`); `/agent/v1` inherits the same boundary — it is authorization-shaped (grants), not a new authentication mechanism. Document this explicitly so AGENT-01 isn't mistaken for an auth control. |
| V3 Session Management | No | No sessions anywhere in this API |
| V4 Access Control | Yes — central to this phase | AGENT-01's default-deny per-source grant, enforced structurally (ungranted sources absent from listings; ungranted direct item access returns the identical `item_not_found` shape as a genuinely nonexistent item — no existence leak) |
| V5 Input Validation | Yes | New path params (`/api/sources/{name}/refresh`'s `{name}`) must be validated against the configured source-name set before dispatch — an unrecognized name returns a 404 in the same shape as other "not found" routes, never a stack trace or a different error family that leaks which names *are* valid |
| V6 Cryptography | No new surface | `SB_AUTH_TOKEN` handled exactly like paperless's existing token: `${VAR}` env-reference only in TOML, never logged, never written back to disk — reuse the existing pattern, don't re-derive it |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| Stored XSS via a SilverBullet page containing crafted HTML/script, rendered into the detail pane | Tampering / Elevation of Privilege | goldmark's safe-by-default rendering (no raw HTML, no dangerous URL schemes) + `bluemonday.UGCPolicy()` sanitization pass + the existing `Content-Security-Policy: ...sandbox` header on the `/content` route + rendering inside an `<iframe>` (three independent layers, not one) |
| Source/item existence leak via differing error responses for "unauthorized" vs "doesn't exist" on `/agent/v1` | Information Disclosure | Identical error code, message shape, and HTTP status for both cases (D-12) — never a distinct `access_denied`-style code that would let an agent enumerate which sources exist but are ungranted |
| SSRF / cross-host redirect confusion in the new SilverBullet HTTP client | Tampering / Spoofing | Mirror `plugins/paperless`'s `outbound_hosts_test.go` pattern exactly: an `allowHost` predicate pinned to the configured base host (permitting case-insensitive host match, loopback, and `localhost`), enforced via a custom `CheckRedirect`, with an explicit test proving a cross-host redirect is refused and a same-host redirect is still followed |
| Credential leakage via plugin subprocess logs | Information Disclosure | `SB_AUTH_TOKEN` (like paperless's token) must never appear in a log line at any level — log "token configured" / "missing environment variable X" only, per the existing documented plugin-contract logging rule |
| Sync-race double-run despite single-flight | Denial of Service (self-inflicted) / Repudiation of health data | Both the scheduler's ticker and the manual-refresh HTTP handler must call the *same* `Coordinator.Refresh(name)` entry point, which internally wraps the actual sync in `singleflight.Group.Do(name, ...)` — never let either caller bypass the coordinator and call `correlate`/plugin code directly, or the single-flight guarantee has a hole |

## Sources

### Primary (HIGH confidence)
- This repository's own Phase 1 code: `kernel/correlate/correlate.go`, `kernel/index/schema.go`, `kernel/index/store.go`, `kernel/pluginhost/host.go`, `kernel/httpapi/routes.go`, `proto/webspaces/v1/plugin.proto`, `plugins/paperless/*`, `docs/api.md`, `docs/plugin-contract.md`, `web/src/lib/components/*`, `web/src/lib/api.ts` — read directly this session, 2026-07-28
- `proxy.golang.org` module version lists for `golang.org/x/sync`, `github.com/yuin/goldmark`, `github.com/microcosm-cc/bluemonday`, `github.com/adrg/frontmatter`, `github.com/robfig/cron/v3`, `github.com/pelletier/go-toml/v2` — fetched directly via `curl`, 2026-07-28
- `pkg.go.dev/golang.org/x/sync/singleflight` — API surface fetched directly, 2026-07-28

### Secondary (MEDIUM confidence)
- `https://silverbullet.md/HTTP%20API` — `/.fs` and `/.fs/{path}` behavior, `SB_AUTH_TOKEN` bearer auth, per-file response headers (fetched directly)
- `https://silverbullet.md/Frontmatter` — frontmatter `tags:` formats (fetched directly)
- `pkg.go.dev/github.com/silverbulletmd/silverbullet/server` — `FileMeta` struct field names, cross-checked against the HTTP API doc's per-file headers (both agree) — pre-release Go server rewrite, Feb 2026
- `https://community.silverbullet.md/t/how-to-search-via-http-api/3198` — confirms no search/tag-filter endpoint exists (community discussion, cross-checked against the absence of any such endpoint in the official HTTP API doc)
- `pkg.go.dev/github.com/microcosm-cc/bluemonday`, `pkg.go.dev/github.com/yuin/goldmark`, `pkg.go.dev/github.com/adrg/frontmatter`, `pkg.go.dev/github.com/robfig/cron/v3` — API surfaces fetched directly

### Tertiary (LOW confidence)
- General WebSearch summaries on OWASP ASVS v5's chapter numbering (no specific clause number for "default-deny, no existence leak" was found in a single authoritative source this session — the pattern itself is a standard, well-known access-control control, but cite the general ASVS Access Control chapter, not a specific v5 clause number, until independently confirmed)

## Metadata

**Confidence breakdown:**
- Standard stack (Go libraries): HIGH — every version verified directly against the Go module proxy, not training data
- SilverBullet HTTP API shape: MEDIUM — no official OpenAPI/JSON-schema spec exists; cross-checked across the official docs site, a pre-release Go server module's struct definition, and per-file HTTP response headers, all of which agree, but the top-level listing envelope shape (array vs. wrapped object) remains an assumption (A1) pending a direct check against the user's real instance
- Architecture (sync/persist fix, health merge, agent namespace): HIGH — grounded directly in reading this repository's own Phase 1 source code, not external research
- Pitfalls: HIGH — five of five pitfalls are demonstrated directly against this repo's existing code (not hypothetical), with file/line-level specificity

**Research date:** 2026-07-28
**Valid until:** 30 days for the Go library recommendations (stable ecosystem); 7 days for the SilverBullet HTTP API specifics if the plan defers the direct-instance check (A1/A2) — resolve those against the real instance rather than re-researching if more than a week passes before implementation
