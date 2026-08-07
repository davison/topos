# topos

## What This Is

topos is a locally-run service that pulls together related information from disparate personal data silos — email accounts, Signal and WhatsApp chats, document storage, note-taking apps, wikis, and project directories — into per-topic "webspaces". Instead of visiting each application to piece together the state of a project or topic, you open its webspace and see all related objects in one rich, view-only interface, with deep links back to the original application for anything beyond viewing.

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
- ✓ Signal plugin (SRC-02) — reads Signal Desktop's SQLCipher database strictly read-only (`mode=ro`, byte-identical after full sync, proven with Signal running), key unwrapped via runtime-detected keyring backend, unrecognised schema versions fail loudly by name, conversation-day digests in the stream with thread detail pane, deep links validated live (1:1 contact-form navigates via literal-'+' E.164; groups raise-only — Signal ships no group-navigation route) — Phase 4
- ✓ Named source instances (KERN-06): the config map key is the kernel's source identity everywhere (index rows, sync runs, agent grants, HTTP API, UI display); the same plugin type configures multiple times under distinct display names, with a schema-version-gated index rebuild for existing data — Phase 5
- ✓ Per-instance typed matching (KERN-07): plugin-declared `match_vocabulary` on the wire (`map<string, StringList> match_fields`, `keywords` retired, contract generation "topos.v2", handshake v1→v2 fail-fast), per-instance `match` blocks with participation allowlist and fail-loud-by-name startup validation, webspace `keywords` as fallback; all five plugins migrated and the operator's live config hand-migrated — Phase 5
- ✓ Kernel-owned rendition boundary: plugins return content plus a declared `ContentShape`; the kernel sanitizes, wraps, and themes at the CSP boundary (three per-plugin theme copies deleted) — Phase 5
- ✓ Scalable source surface (UI-07/08/09/11): each source instance appears exactly once in the header as a single merged chip (health + filter toggle + hover-revealed refresh in one 44px pill, overflow popover at high instance counts), deep-link affordances differentiate raise-only from navigating links (closes the 04-UAT follow-up), search terms highlight across stream rows, result titles/snippets and the detail pane (including sanitized renditions, injected kernel-side as HTML tree nodes), and the stream scrollbar carries a date-marker ruler with click-to-jump — Phase 6 (three UAT gap-closure rounds to land the chip polish)

### Active

- [ ] Define webspaces in a config map: each webspace matches against the *native* categorization of each source (IMAP folders/labels, chat group names, paperless-ngx tags, SilverBullet tags/pages, directory names) *(Phases 1–4 proved keyword matching for all four shipped sources; Phase 5 upgraded the shape to per-instance typed match blocks with keywords as fallback; WhatsApp pending)*
- [ ] WhatsApp plugin — reads WhatsApp desktop/linked-device local store on the same machine
- [ ] Web UI: stream + detail pane — chronological cross-source feed per webspace, filterable by source, inline preview (email body, chat thread, note, document), "open in source" deep link on every item *(Phases 3–4: email body preview, in-webspace search, and Signal chat-thread preview shipped; Phase 6 closed the 04-UAT follow-up — raise-only vs navigating deep links are now visually differentiated; WhatsApp previews pending)*

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
| Chat access via desktop app local databases | Least infra of the workaround options; no cloud, no bridges to run | Phase 4: validated for Signal — SQLCipher DB read `mode=ro`, byte-identical after sync even with Signal running; keyring backend detected at runtime |
| Deploy to desktop, reach server services over LAN | Chat DBs live on the desktop; bridge/paperless/SilverBullet are network-reachable anyway | Phase 4: validated — Signal Desktop's local DB read in place on the desktop |
| MVP sources: IMAP email, Signal, WhatsApp, paperless-ngx, SilverBullet | The user's actual silos; filesystem and others deferred | Phase 4: four of five shipped (paperless-ngx, SilverBullet, Proton/IMAP, Signal) |
| Signal deep links: E.164 allowlist, emitted verbatim; groups raise-only | Signal Desktop's validator demands a literal '+' (percent-encoding is rejected); its route table has no group-navigation route, so group links can only raise the window — conversation-only fidelity is the honest declaration | Phase 4: confirmed live (1:1 navigates; group raise-only diagnosed as upstream-hard, journal-verified) |
| Email readability decided in the plugin, never the shared pane | A UI-side "prefer text over rendition" rule would have flipped SilverBullet's rendered markdown to raw; the producing plugin returns plain text alone when a usable text/plain part exists | Phase 3: shipped — DetailPane stays source-agnostic, branches only on content shape |
| Proton deep link = All Mail subject-search URL, declared ANCHORED | Proton webmail addresses custom labels by internal id (not name) and offers no Message-ID→webmail-id mapping; a search link is the only addressable form | Phase 3: confirmed working live |
| Source identity = config-map instance key, split from plugin type (D-08) | Two instances of one plugin binary must never share identity in index rows, grants, or UI; `source_type` stays purely "which plugin kind" | Phase 5: shipped — leak-tested (two instances stay distinct incl. agent grants) |
| Match contract: generic plugin-declared field map, proto package stays topos.v1, handshake v1→v2 (option-a) | Field names never fixed in the proto — kernel holds no table of known plugin types (D-05); stale binaries fail at handshake, not confusingly at first Match; avoids a full topos.v2 package-move churn | Phase 5: shipped — user-locked at checkpoint; contract republished in docs/plugin-contract.md |
| Rendition sanitize/wrap/theme moved into the kernel (D-11) | Sanitization must sit inside the trust boundary once plugins can be third-party; one theme edit instead of three plugin copies | Phase 5: shipped — plugins return content + declared ContentShape; UAT confirmed pixel parity |
| Search-term highlighting injected kernel-side as HTML tree nodes after sanitization | Marking text via `x/net/html` parse/walk/render into text nodes only — never string substitution over sanitized HTML — keeps the sanitizer output as the trust anchor | Phase 6: shipped — attributes/tags proven byte-untouched, multi-byte runes survive |
| One merged chip per source instance (D-01): health + filter + refresh in a single affordance | Header must stay usable at 10+ instances without duplicated per-source controls | Phase 6: shipped — took three UAT gap-closure rounds (selected-state fill, pill geometry/reveal) to read as one polished control |

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
*Last updated: 2026-08-07 after Phase 6*
