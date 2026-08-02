# Webspaces

## What This Is

Webspaces is a locally-run service that pulls together related information from disparate personal data silos — email accounts, Signal and WhatsApp chats, document storage, note-taking apps, wikis, and project directories — into per-topic "webspaces". Instead of visiting each application to piece together the state of a project or topic, you open its webspace and see all related objects in one rich, view-only interface, with deep links back to the original application for anything beyond viewing.

## Core Value

Open one webspace and instantly see and grok all related information across every silo — without visiting each data store individually.

## Requirements

### Validated

- ✓ paperless-ngx plugin — via its REST API over LAN — Phase 1 (real documents synced, previewed, deep-linked; read-only and host-pinned egress enforced by committed tests)
- ✓ Hybrid data model: plugins sync metadata + previews into a local index; full content fetched live from the source when an item is opened — Phase 1
- ✓ SilverBullet plugin — wiki/notes content and tags, rendered markdown previews, host-pinned egress, token never logged — Phase 2
- ✓ Kernel + plugin architecture: source connectors are plugins with a documented contract — Phase 2 (source-agnosticism proven by a second, structurally different source; PLUG-05 fresh-context test showed a third party can build a plugin from contract + mock alone)
- ✓ Trustworthy multi-source stream: per-source health chips with diagnostic tooltips, source filter chips, manual refresh, staleness states, per-(webspace, source) sync integrity — Phase 2
- ✓ Default-deny agent permission model (AGENT-01): read-only agent routes gated by per-source grants — Phase 2
- ✓ Email plugin (SRC-01) — Proton Mail Bridge over LAN with self-signed cert, exact-leaf label/folder keyword matching, Message-ID dedup, never-marks-read proven live against the real account, readable detail pane (plain-text preferred, sanitized theme-wrapped HTML fallback), All Mail search deep link — Phase 3
- ✓ Full-text search within a webspace (KERN-05): FTS5 index, ranked cross-source results with highlighted snippets — Phase 3

### Active

- [ ] Define webspaces in a config map: each webspace has a keyword matched against the *native* categorization of each source (IMAP folders/labels, chat group names, paperless-ngx tags, SilverBullet tags/pages, directory names) *(Phases 1–3: proven for paperless-ngx tags, SilverBullet tags/pages, and IMAP labels/folders; chat silos pending)*
- [ ] Signal plugin — reads Signal Desktop's local database on the same machine
- [ ] WhatsApp plugin — reads WhatsApp desktop/linked-device local store on the same machine
- [ ] Web UI: stream + detail pane — chronological cross-source feed per webspace, filterable by source, inline preview (email body, chat thread, note, document), "open in source" deep link on every item *(Phase 3: email body preview and in-webspace search shipped; chat previews pending)*

### Out of Scope

- Write/edit functionality in any source — view-only by design; acting on content happens in the original application via deep links
- Replicating source-app features (composing email, replying to chats, editing notes) — same reason
- AI/agent-inferred correlation (context-relative searches beyond the configured keyword map) — explicitly planned as a later layer, not v1
- Local filesystem / project-directory plugin — valuable but deferred; MVP sources are email, chats, paperless-ngx, SilverBullet
- Cloud/SaaS deployment or multi-user support — this is a personal, locally-run tool scratching one user's itch
- Building on Timelinize, Dogsheep, or Onyx — landscape research (July 2026) found none do identifier-correlated cross-source views; greenfield chosen deliberately

## Context

- **Landscape**: Adjacent open-source projects exist (Timelinize: local timeline archive; Dogsheep/Datasette: personal data warehouse; Onyx/Danswer: connector-based enterprise search) but none correlate objects across sources by a shared identifier with links back to live sources. The kernel+plugin shape mirrors how all three are built, validating the architecture.
- **Deployment target**: The user's desktop machine (Linux/Arch). Chosen so the Signal Desktop and WhatsApp desktop databases are readable locally.
- **Email**: Proton Mail, accessed via Proton Mail Bridge running on a separate home server (already in use by paperless-ngx for attachment ingestion). The user will open firewall access so the desktop can reach the bridge's IMAP endpoint over LAN. Note: the bridge binds to 127.0.0.1 by default and uses a self-signed cert; it must be made to listen on the LAN interface (or be tunnelled). Proton labels/folders are exposed as IMAP folders — good for keyword matching.
- **Remote sources over LAN**: Proton Mail Bridge (IMAP), paperless-ngx (REST API), SilverBullet — all reachable from the desktop.
- **Chat access is the riskiest area**: Signal and WhatsApp have no official personal read APIs. Chosen route is reading the desktop apps' local databases (Signal Desktop's DB is decryptable locally; WhatsApp likely needs a linked-device library such as whatsmeow or an equivalent local-store approach). This drove the deployment-to-desktop decision.
- **Correlation model**: v1 is a deterministic config map — per-webspace keyword looked up in each silo's native categorization. AI/agent-inferred, context-relative search over configured systems is the intended v2 layer.

## Constraints

- **Deployment**: Runs on the user's desktop machine — required for local access to Signal/WhatsApp desktop databases
- **Privacy**: All data stays local; no personal content leaves the user's machines
- **Read-only**: Plugins must never mutate source data stores (especially the chat databases)
- **Extensibility**: Plugin contract must be stable and documented enough for third-party source plugins
- **Dependencies**: Proton Mail Bridge, paperless-ngx, and SilverBullet instances already exist on the home server and are LAN-reachable

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| Build greenfield rather than extend Timelinize/Onyx/Dogsheep | None do identifier-correlated cross-source views; correlation is the core, not a bolt-on | — Pending |
| Kernel + UI with per-source plugins | New sources addable later, including by third parties; mirrors proven connector architectures | Phase 1: shipped — go-plugin/gRPC contract documented and pinned by RPC-allowlist + read-only AST tests |
| Config-mapped keyword correlation for v1; AI inference later | Deterministic, no false positives; magic can layer on top once the plumbing works | Phase 1: works — webspace keyword matched against paperless-ngx tags with per-item rejection on contract violations |
| Hybrid data model (local metadata/preview index, live fetch on open) | Fast browsing and uniform search without full duplication or staleness of content | Phase 1: validated — instant metadata from index, live preview fill via plugin Fetch |
| Chat access via desktop app local databases | Least infra of the workaround options; no cloud, no bridges to run | — Pending |
| Deploy to desktop, reach server services over LAN | Chat DBs live on the desktop; bridge/paperless/SilverBullet are network-reachable anyway | — Pending |
| MVP sources: IMAP email, Signal, WhatsApp, paperless-ngx, SilverBullet | The user's actual silos; filesystem and others deferred | Phase 3: three of five shipped (paperless-ngx, SilverBullet, Proton/IMAP) |
| Email readability decided in the plugin, never the shared pane | A UI-side "prefer text over rendition" rule would have flipped SilverBullet's rendered markdown to raw; the producing plugin returns plain text alone when a usable text/plain part exists | Phase 3: shipped — DetailPane stays source-agnostic, branches only on content shape |
| Proton deep link = All Mail subject-search URL, declared ANCHORED | Proton webmail addresses custom labels by internal id (not name) and offers no Message-ID→webmail-id mapping; a search link is the only addressable form | Phase 3: confirmed working live |

## Evolution

This document evolves at phase transitions and milestone boundaries.

**After each phase transition** (via `/gsd-transition`):
1. Requirements invalidated? → Move to Out of Scope with reason
2. Requirements validated? → Move to Validated with phase reference
3. New requirements emerged? → Add to Active
4. Decisions to log? → Add to Key Decisions
5. "What This Is" still accurate? → Update if drifted

**After each milestone** (via `/gsd-complete-milestone`):
1. Full review of all sections
2. Core Value check — still the right priority?
3. Audit Out of Scope — reasons still valid?
4. Update Context with current state

---
*Last updated: 2026-08-02 after Phase 3*
