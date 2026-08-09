# Requirements — topos

Defined 2026-07-27. REQ-IDs are stable references; traceability is filled by the roadmap.

## v1 Requirements

### Kernel & Correlation

- [x] **KERN-01**: User can define webspaces in config: each has a keyword mapped to the native categorization of each source (IMAP folders/labels, chat group names, paperless-ngx tags, SilverBullet tags/pages)
- [x] **KERN-02**: Correlation is resolved at sync time and persisted in the local index (never computed at query time)
- [x] **KERN-03**: Hybrid store: metadata + preview indexed locally; full content fetched live from the source when an item is opened
- [x] **KERN-04**: Sync scheduler with a per-plugin coordinator (dedups refreshes, tracks health); user can trigger manual refresh
- [x] **KERN-05**: User can full-text search within a webspace (FTS5 over indexed metadata/previews)
- [x] **KERN-06**: Sources are named instances — the same plugin type can be configured multiple times, each under a user-chosen display name (e.g. "Home email" / "Work email") used throughout the UI and API *(added 2026-08-05)*
- [x] **KERN-07**: Matching config is declared per source instance and typed to its plugin (IMAP folders/labels, document tags, chat conversation/group names, wiki tags/pages), replacing the single shared per-webspace keyword list *(added 2026-08-05; supersedes the KERN-01 shared-keyword shape — KERN-01 remains satisfied via migration)*
- [x] **KERN-08**: Webspace and source-instance configuration is editable through the kernel API (non-secret fields only; secrets stay environment-only), while hand-editing the config file remains supported *(added 2026-08-05)*

### Plugin Architecture

- [x] **PLUG-01**: Plugins run as isolated subprocesses under a documented, versioned gRPC contract (published `.proto`)
- [x] **PLUG-02**: Contract is read-only by construction — no source-mutating methods exist
- [x] **PLUG-03**: Every item declares deep-link fidelity: exact / anchored / conversation-only
- [x] **PLUG-04**: Plugins report health (reachable, last sync, last error) to the kernel
- [x] **PLUG-05**: A third party can build a plugin from the contract docs + reference mock plugin alone

### Source Plugins

- [x] **SRC-01**: Email plugin (IMAP) works against Proton Mail Bridge (self-signed cert handling); uses `BODY.PEEK` so mail is never marked read; matches webspace keyword against folders/labels; dedups by Message-ID
- [x] **SRC-02**: Signal plugin reads Signal Desktop DB strictly read-only (`mode=ro`); extracts key via OS keyring (backend-detected); detects schema version and fails loudly on unknown
- [ ] **SRC-03**: WhatsApp plugin runs as a whatsmeow linked device with its own persistent message store; degrades gracefully on de-link/ban; matches on group names
- [x] **SRC-04**: paperless-ngx plugin via REST API; matches on tags; exact deep links to documents
- [x] **SRC-05**: SilverBullet plugin; matches on tags/pages; exact deep links to pages

### UI

- [x] **UI-01**: Web UI shows a webspace as a chronological cross-source stream with a detail pane
- [x] **UI-02**: User can filter the stream by source
- [x] **UI-03**: Items render inline previews appropriate to type (email body, chat thread, note, document)
- [x] **UI-04**: Every item has an "open in source" affordance matching its declared link fidelity
- [x] **UI-05**: Stale or source-unavailable items show an explicit state (not silent 404s)
- [x] **UI-06**: Sync status and plugin health are visible in the UI
- [x] **UI-07**: The header presents each source instance exactly once — one affordance combining health, filter toggle, and refresh — and stays usable as the instance count grows *(added 2026-08-05)*
- [x] **UI-08**: "Open in source" affordances visually differentiate links that navigate to the item from links that can only raise the source app's window *(added 2026-08-05; from 04-UAT follow-up)*
- [x] **UI-09**: After an in-webspace search, matched terms are highlighted in the item title and the rendered content, in both the detail pane and the search-results row *(added 2026-08-05; surface wording disambiguated 2026-08-07 per G-06-1)*
- [x] **UI-11**: Scrollbars are thin and theme-matched app-wide; the stream scrollbar carries date markers *(added 2026-08-05; UI-10 was already assigned to the v1.x sync-status polish item)*
- [x] **UI-12**: Webspace builder UI — pick plugin types, configure named instances, save the set as a webspace, and promote a live search into a permanent webspace filter refinable by further search *(added 2026-08-05)*

### Agent Readiness (provisions only — the agent itself is v1.x)

- [x] **AGENT-01**: Per-plugin permission model, default-deny, defined in config: separately grants (a) agent read access to a source's items and (b) action hand-off via that source's own interfaces
- [x] **AGENT-02**: Item schema and kernel HTTP API are designed for programmatic/agent consumption (stable IDs, structured content, machine-readable provenance)

## v1.x Requirements (deferred — v1 architecture must be ready for them)

- **AGENT-10**: Agent chat scoped to a webspace's objects as context
- **AGENT-11**: Agent-requested interactions ("draft an email reply", "update the wiki note") via hand-off to source apps/APIs — only for plugins the user pre-authorized (AGENT-01)
- **AGENT-12**: Pluggable LLM backend (OpenAI-compatible abstraction; locally hosted models preferred/first-class)
- **UI-10**: Sync-status detail views and provenance display polish

## v2

- Cross-webspace search
- AI-inferred correlation (context-relative searches beyond configured keywords)

## Out of Scope

- Direct writes through the plugin contract — excluded permanently; agent actions go via source interfaces under explicit per-plugin permission
- Replicating source-app functionality (composing, replying, editing inside webspaces) — view + hand-off only
- Cloud/SaaS deployment, multi-user — personal, locally-run tool
- Additional source plugins (filesystem, etc.) — deferred until after v1; plugin contract makes them additive

## Traceability

Mapped by roadmap 2026-07-27; extended 2026-08-05 (restructure: Phases 5–7 inserted, WhatsApp shifted to Phase 8, 8 requirements added). All 31 v1 requirements map to exactly one phase — no orphans, no duplicates.

| Requirement | Phase | Status |
|-------------|-------|--------|
| KERN-01 | Phase 1 | Complete |
| KERN-02 | Phase 1 | Complete |
| KERN-03 | Phase 1 | Complete |
| KERN-04 | Phase 2 | Gaps Found |
| KERN-05 | Phase 3 | Complete |
| KERN-06 | Phase 5 | Complete |
| KERN-07 | Phase 5 | Complete |
| KERN-08 | Phase 7 | Gaps Found |
| PLUG-01 | Phase 1 | Complete |
| PLUG-02 | Phase 1 | Complete |
| PLUG-03 | Phase 1 | Complete |
| PLUG-04 | Phase 2 | Gaps Found |
| PLUG-05 | Phase 2 | Gaps Found |
| SRC-01 | Phase 3 | Complete |
| SRC-02 | Phase 4 | Complete |
| SRC-03 | Phase 8 | Pending |
| SRC-04 | Phase 1 | Complete |
| SRC-05 | Phase 2 | Gaps Found |
| UI-01 | Phase 1 | Complete |
| UI-02 | Phase 2 | Gaps Found |
| UI-03 | Phase 1 | Complete |
| UI-04 | Phase 1 | Complete |
| UI-05 | Phase 2 | Gaps Found |
| UI-06 | Phase 2 | Gaps Found |
| UI-07 | Phase 6 | Complete |
| UI-08 | Phase 6 | Complete |
| UI-09 | Phase 6 | Complete |
| UI-11 | Phase 6 | Complete |
| UI-12 | Phase 7 | Gaps Found |
| AGENT-01 | Phase 2 | Gaps Found |
| AGENT-02 | Phase 1 | Complete |
| AGENT-10 | — | Deferred (v1.x) |
| AGENT-11 | — | Deferred (v1.x) |
| AGENT-12 | — | Deferred (v1.x) |
| UI-10 | — | Deferred (v1.x) |

Deferred requirements (AGENT-10, AGENT-11, AGENT-12, UI-10) are v1.x and intentionally unmapped to any v1 phase; listed above so traceability scans account for every REQ-ID.
