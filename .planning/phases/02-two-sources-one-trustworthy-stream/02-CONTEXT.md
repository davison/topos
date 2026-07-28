# Phase 2: Two Sources, One Trustworthy Stream - Context

**Gathered:** 2026-07-28
**Status:** Ready for planning

<domain>
## Phase Boundary

SilverBullet joins paperless-ngx as the second, structurally different source (SRC-05), and the stream becomes *trustworthy*: user can filter to a single source and back (UI-02), see per-source health — reachable, last sync, last error — in the UI (PLUG-04, UI-06), trigger a manual refresh that neither duplicates items nor stacks concurrent syncs (KERN-04, per-plugin sync coordinator), and every stale/unreachable/deleted item shows an explicit state instead of a blank pane or silent 404 (UI-05). The default-deny per-plugin agent permission model lands in config (AGENT-01) while only two plugins exist. PLUG-05 is validated here: a third party (or a fresh context with no repo access beyond the published `.proto`, contract docs, and reference mock plugin) can build a working plugin from those artifacts alone. No email, no search, no chat sources — those are Phases 3–5.

**Mode note:** The user delegated all Phase 2 gray-area decisions to Claude ("happy to accept agent decisions"). The decisions below are Claude's recommended defaults, recorded as locked so downstream agents don't re-open them; the user retains veto at plan review and UAT.

</domain>

<decisions>
## Implementation Decisions

### SilverBullet in the stream
- **D-01:** One SilverBullet page = one stream item. `source_id` is the page path (without `.md`). Deep link is **exact** fidelity to `{base_url}/{page-path}`.
- **D-02:** Chronological order is driven by the page's **last-modified** timestamp (from the `/.fs` listing metadata) — SilverBullet does not reliably expose creation time. — **Reversibility:** reversible, but re-ordering requires a full re-sync.
- **D-03:** Keyword matching (exact, case-insensitive per Phase 1 D-03) runs against **both** page tags (frontmatter `tags:` and inline `#tag`) **and** page name — matched against the final path segment and the full path, extension stripped. Either match includes the page.
- **D-04:** Stream preview per the hybrid model: page name as title, frontmatter-stripped plaintext snippet, tags rendered as badges (same treatment as paperless tags). Detail pane renders **sanitized rendered markdown** fetched live via `Fetch`; a raw-markdown `ContentVariant` may be offered if cheap, but rendered is the default.

### Refresh & sync cadence
- **D-05:** A background sync scheduler replaces the Phase 1 startup-only trigger: global `sync_interval` in TOML (default `15m`) with optional per-source override; startup sync is retained as the first scheduled run. — **Reversibility:** reversible.
- **D-06:** The per-plugin coordinator is **single-flight**: a refresh request while that plugin is already syncing coalesces into (and reports) the in-flight run — never queued, never parallel. This is the single source of truth for health and sync state that Phases 3–5 plugins inherit. — **Reversibility:** costly — every later plugin builds on the coordinator's semantics.
- **D-07:** Manual refresh is exposed both **per source** (from the health UI) and as **refresh-all** on the webspace header, via a kernel HTTP endpoint (POST) that returns the sync-run status. While in flight, the UI shows a non-blocking syncing indicator (driven by `sync_runs.status = "running"`); the stream re-fetches on completion.

### Health, filter & staleness UI
- **D-08:** Per-source health lives in the **webspace header** as compact status chips (colored dot + source name; green = ok, amber = stale/last error, red = unreachable), with a tooltip showing last-sync relative time and last error, and the per-source refresh action. Reuses the existing badge/tooltip components. Health is a kernel-side merge of the plugin `Health` RPC and `sync_runs` history.
- **D-09:** Source filter is a chip row in the header — "All" plus one chip per source, single-select toggle back to All (matches success criterion "down to a single source and back"). Filter state persists in the URL query (e.g. `?source=silverbullet`) so reloads and deep links preserve it.
- **D-10:** Staleness semantics:
  - **Source unreachable** → that source's stream rows carry a subtle stale indicator; the detail pane still renders the indexed preview with an explicit alert: source unreachable, showing last-synced preview. Never a blank pane.
  - **Item deleted at source** → live fetch returns `content_unavailable`; detail pane shows an explicit "no longer available at source" state over the cached preview. The item disappears from the stream at the next successful sync (the index mirrors source truth); it is never silently 404'd while indexed.
  - Phase 1's rule carries forward: a failed sync must never render as an empty webspace.

### Agent permission config
- **D-11:** TOML grant shape, per source, default-deny by absence:
  ```toml
  [sources.silverbullet.agent]
  read = true      # agent may read this source's items
  handoff = false  # agent may hand actions off via this source's own interfaces
  ```
  Absent block or absent key = `false`. Two independent booleans per AGENT-01.
- **D-12:** "Agent-facing API" becomes a **separate route namespace** (`/agent/v1/...`) mirroring the read API, with grants enforced structurally: ungranted sources are absent from source/health listings, their items are absent from streams, and direct item/content requests return the same error as a nonexistent item (no existence leak). The UI's existing `/api/*` routes are unaffected and remain grant-free for the human user. `docs/api.md`'s Phase-1 statement "there is no separate agent API" must be updated — it described AGENT-02 only and explicitly deferred AGENT-01. `handoff` is recorded and exposed as capability metadata only; actual agent actions are v1.x (AGENT-11). — **Reversibility:** costly — the namespace becomes a published surface agents build against; collapsing it back into `/api/*` later would need an auth mechanism that doesn't exist in v1.

### Claude's Discretion
The user delegated the entire discussion; beyond the locked decisions above, planner/researcher discretion explicitly covers:
- Reference mock plugin design and the PLUG-05 validation exercise (fresh-context build from `.proto` + `docs/plugin-contract.md` + mock alone).
- Exact snippet lengths, health-chip copy, error wording, sanitizer choice for rendered markdown.
- SilverBullet auth handling (`SB_AUTH_TOKEN` bearer via env interpolation, per the locked stack) and confirming the exact `/.fs` metadata shape during planning (roadmap flags this as the one research item).

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope & requirements
- `.planning/ROADMAP.md` — Phase 2 goal, 5 success criteria, and notes (coordinator inherited by Phases 3–5; PLUG-05 validation method; SilverBullet API shape to confirm during planning).
- `.planning/REQUIREMENTS.md` — Phase 2 requirements: KERN-04, PLUG-04, PLUG-05, SRC-05, UI-02, UI-05, UI-06, AGENT-01.
- `.planning/PROJECT.md` — constraints (read-only, privacy, extensibility) and Key Decisions table.
- `.planning/phases/01-first-webspace-end-to-end/01-CONTEXT.md` — locked Phase 1 decisions this phase builds on (TOML config, keyword matching D-02/D-03, env-var secrets).

### Published contracts (Phase 1 output — extend, don't break)
- `docs/plugin-contract.md` — the third-party-facing plugin contract PLUG-05 is validated against.
- `docs/api.md` — HTTP envelope, error-code table, loopback security model; new health/refresh/agent routes must follow it, and its "no separate agent API" line must be revised per D-12.
- `proto/webspaces/v1/` — the `.proto` source of truth; `Health` RPC (reachable, last_sync_unix, last_error) already exists.
- `config.example.toml` — extend with sync intervals and `[sources.*.agent]` grants.

### Technology stack (locked)
- `.claude/CLAUDE.md` — stack decisions incl. SilverBullet `/.fs` HTTP API notes (GET-only, Bearer `SB_AUTH_TOKEN`), hand-rolled thin client, frontmatter parsing approach.

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `proto/webspaces/v1` `Health` RPC — already in the contract; Phase 2 wires it to kernel state and UI rather than defining it.
- `kernel/index/schema.go` — `sync_runs` table (status running/ok/error, error text, item_count) and `webspaces` registry: the coordinator and health merge build directly on these.
- `plugins/paperless/` — the reference plugin shape (client, plugin.go, readonly/outbound-host/fetch tests) the SilverBullet plugin mirrors.
- `web/src/lib/components/` — StreamRow/StreamList/StreamError/StreamEmpty/DetailPane/WebspaceHeader plus shadcn badge, alert, tooltip, button: health chips, filter chips, and stale states compose from these.

### Established Patterns
- Single JSON envelope with `schema_version` + snake_case machine-matchable error codes (`docs/api.md`) — all new routes conform.
- Contract enforcement by test: RPC allowlist (any new RPC fails the build until allowlisted), read-only AST tests, host-pinned egress tests — the SilverBullet client needs its own `outbound_hosts_test.go` equivalent pinned to the configured SilverBullet host.
- Sync-failure branch renders strictly before the empty branch (StreamList) — extend, never regress, when adding stale states.
- Per-item rejection (not whole-batch) on contract violations at sync time.

### Integration Points
- `cmd/webspaces/main.go` — current startup-sync call is replaced by the scheduler/coordinator; `webspaces sync` CLI should route through the coordinator too.
- `kernel/httpapi/routes.go` — mount points for health, refresh, and the `/agent/v1` namespace.
- `kernel/config/` — extend types/loader for sync intervals, SilverBullet source block, and agent grants.
- `kernel/pluginhost/host.go` — plugin discovery/launch; SilverBullet plugin registers alongside paperless.

</code_context>

<specifics>
## Specific Ideas

No specific user references — user delegated this phase's decisions to Claude's recommended defaults (see Mode note in Phase Boundary).

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 2-Two Sources, One Trustworthy Stream*
*Context gathered: 2026-07-28*
