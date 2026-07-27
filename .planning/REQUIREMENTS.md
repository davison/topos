# Requirements — Webspaces

Defined 2026-07-27. REQ-IDs are stable references; traceability is filled by the roadmap.

## v1 Requirements

### Kernel & Correlation

- [x] **KERN-01**: User can define webspaces in config: each has a keyword mapped to the native categorization of each source (IMAP folders/labels, chat group names, paperless-ngx tags, SilverBullet tags/pages)
- [x] **KERN-02**: Correlation is resolved at sync time and persisted in the local index (never computed at query time)
- [ ] **KERN-03**: Hybrid store: metadata + preview indexed locally; full content fetched live from the source when an item is opened
- [ ] **KERN-04**: Sync scheduler with a per-plugin coordinator (dedups refreshes, tracks health); user can trigger manual refresh
- [ ] **KERN-05**: User can full-text search within a webspace (FTS5 over indexed metadata/previews)

### Plugin Architecture

- [x] **PLUG-01**: Plugins run as isolated subprocesses under a documented, versioned gRPC contract (published `.proto`)
- [x] **PLUG-02**: Contract is read-only by construction — no source-mutating methods exist
- [ ] **PLUG-03**: Every item declares deep-link fidelity: exact / anchored / conversation-only
- [ ] **PLUG-04**: Plugins report health (reachable, last sync, last error) to the kernel
- [ ] **PLUG-05**: A third party can build a plugin from the contract docs + reference mock plugin alone

### Source Plugins

- [ ] **SRC-01**: Email plugin (IMAP) works against Proton Mail Bridge (self-signed cert handling); uses `BODY.PEEK` so mail is never marked read; matches webspace keyword against folders/labels; dedups by Message-ID
- [ ] **SRC-02**: Signal plugin reads Signal Desktop DB strictly read-only (`mode=ro`); extracts key via OS keyring (backend-detected); detects schema version and fails loudly on unknown
- [ ] **SRC-03**: WhatsApp plugin runs as a whatsmeow linked device with its own persistent message store; degrades gracefully on de-link/ban; matches on group names
- [x] **SRC-04**: paperless-ngx plugin via REST API; matches on tags; exact deep links to documents
- [ ] **SRC-05**: SilverBullet plugin; matches on tags/pages; exact deep links to pages

### UI

- [x] **UI-01**: Web UI shows a webspace as a chronological cross-source stream with a detail pane
- [ ] **UI-02**: User can filter the stream by source
- [ ] **UI-03**: Items render inline previews appropriate to type (email body, chat thread, note, document)
- [ ] **UI-04**: Every item has an "open in source" affordance matching its declared link fidelity
- [ ] **UI-05**: Stale or source-unavailable items show an explicit state (not silent 404s)
- [ ] **UI-06**: Sync status and plugin health are visible in the UI

### Agent Readiness (provisions only — the agent itself is v1.x)

- [ ] **AGENT-01**: Per-plugin permission model, default-deny, defined in config: separately grants (a) agent read access to a source's items and (b) action hand-off via that source's own interfaces
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

Mapped by roadmap 2026-07-27. All 23 v1 requirements map to exactly one phase — no orphans, no duplicates.

| Requirement | Phase | Status |
|-------------|-------|--------|
| KERN-01 | Phase 1 | Complete |
| KERN-02 | Phase 1 | Complete |
| KERN-03 | Phase 1 | Pending |
| KERN-04 | Phase 2 | Pending |
| KERN-05 | Phase 3 | Pending |
| PLUG-01 | Phase 1 | Complete |
| PLUG-02 | Phase 1 | Complete |
| PLUG-03 | Phase 1 | Pending |
| PLUG-04 | Phase 2 | Pending |
| PLUG-05 | Phase 2 | Pending |
| SRC-01 | Phase 3 | Pending |
| SRC-02 | Phase 4 | Pending |
| SRC-03 | Phase 5 | Pending |
| SRC-04 | Phase 1 | Complete |
| SRC-05 | Phase 2 | Pending |
| UI-01 | Phase 1 | Complete |
| UI-02 | Phase 2 | Pending |
| UI-03 | Phase 1 | Pending |
| UI-04 | Phase 1 | Pending |
| UI-05 | Phase 2 | Pending |
| UI-06 | Phase 2 | Pending |
| AGENT-01 | Phase 2 | Pending |
| AGENT-02 | Phase 1 | Complete |

Deferred requirements (AGENT-10, AGENT-11, AGENT-12, UI-10) are v1.x and intentionally unmapped.
