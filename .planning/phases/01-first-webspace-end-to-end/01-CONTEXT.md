# Phase 1: First Webspace, End to End - Context

**Gathered:** 2026-07-27
**Status:** Ready for planning

<domain>
## Phase Boundary

The walking skeleton, proven by one real source: user defines a webspace in config, starts the service, opens the webspace in a browser, and sees their paperless-ngx documents for that topic as one chronological stream with inline previews, a live-fetched detail pane, and exact deep links back to paperless-ngx. Lands the normalized Item schema, local index store (SQLite), webspace config loader, sync-time correlation engine, plugin host (`hashicorp/go-plugin` gRPC subprocess), published read-only `.proto` contract, kernel HTTP API (agent-consumable per AGENT-02), and the SvelteKit SPA shell — all behind the paperless-ngx plugin only. No second source, no filters, no health UI, no search (those are Phases 2–3).

</domain>

<decisions>
## Implementation Decisions

### Webspace config shape
- **D-01:** Config format is **TOML**. — **Reversibility:** costly — every downstream artifact (config loader, docs, the user's hand-maintained file, examples in plugin contract docs) assumes the format; switching later means a migration of the user's config and all documentation.
- **D-02:** Each webspace defines a **list of keywords** (e.g. `["house-move", "House"]`). **All keywords are matched in all plugins** against each source's native categorization (paperless tags, IMAP folders/labels, chat group names, SilverBullet tags/pages). There is **no per-source override syntax** — a silo that names things differently is handled by adding its variant to the shared keyword list. — **Reversibility:** costly — the keyword-list shape is baked into the plugin contract's match request and every plugin's matching code; Phases 2–5 plugins all build on it.
- **D-03:** Keyword matching is **exact, case-insensitive**: `house` matches tag `House` but not `Household`. No substring or prefix matching — variants are listed explicitly. Deterministic, no false positives (consistent with the v1 config-mapped correlation decision in PROJECT.md).
- **D-04:** **One config file** at `~/.config/webspaces/config.toml` containing kernel settings, source connections, and webspace definitions. **Secrets are not stored in the file** — tokens (e.g. paperless-ngx API token) come from environment variables, with `${VAR}` interpolation supported in the config file.

### Claude's Discretion
The user explicitly left these areas to Claude during research/planning:
- **Stream & detail presentation** — what a document item shows in the stream (title/snippet/tags/thumbnail), which timestamp drives chronological order, and what the detail pane renders for a paperless document (extracted text vs embedded PDF preview).
- **Sync trigger for Phase 1** — the full scheduler/coordinator is Phase 2 (KERN-04); Phase 1 needs a minimal trigger (startup sync, simple interval, or manual). Also what is stored as the local "preview" for a document.
- **Running the service** — command shape (e.g. single `webspaces serve`), plugins directory location, listen port, dev vs prod serving of the SPA.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Phase scope & requirements
- `.planning/ROADMAP.md` — Phase 1 goal, success criteria (5), and notes: contract must be sketched against two structurally different sources (documents AND chat) before it is written, even though only paperless-ngx ships this phase.
- `.planning/REQUIREMENTS.md` — Phase 1 requirements: KERN-01, KERN-02, KERN-03, PLUG-01, PLUG-02, PLUG-03, SRC-04, UI-01, UI-03, UI-04, AGENT-02.
- `.planning/PROJECT.md` — constraints (read-only, privacy, extensibility) and Key Decisions table.

### Technology stack (locked)
- `.claude/CLAUDE.md` — full researched stack: Go kernel, `modernc.org/sqlite` (pure Go, FTS5) for the index, `hashicorp/go-plugin` gRPC subprocess plugins with `buf`-managed `.proto` as the published contract, `go-chi/chi` HTTP router, SvelteKit 2 / Svelte 5 SPA via `adapter-static` embedded with `go:embed`, hand-rolled paperless-ngx REST client (~4 endpoints, token auth). Includes "What NOT to Use" list.

No other external specs or ADRs exist yet — this is the first phase of a greenfield repo.

</canonical_refs>

<code_context>
## Existing Code Insights

Greenfield repository — no source code exists yet (only `.planning/` and `.claude/`). No reusable assets, patterns, or integration points to carry in. Phase 1 establishes the patterns every later phase inherits: repo layout (kernel vs per-plugin modules — note the tech stack requires the Signal plugin's cgo to stay out of the kernel module), the Item schema, the `.proto` contract, and the UI component conventions.

</code_context>

<specifics>
## Specific Ideas

- Config example the user gave: a "house-move" webspace with keywords `"house-move, House"` — both keywords checked against paperless tags AND (later) IMAP folders alike. Use this as the canonical example in config docs and sample config.

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope.

</deferred>

---

*Phase: 1-First Webspace, End to End*
*Context gathered: 2026-07-27*
