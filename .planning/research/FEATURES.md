# Feature Research

**Domain:** Local-first, single-user, view-only personal data aggregation ("unified topic view" over email, chat, documents, notes)
**Researched:** 2026-07-27
**Confidence:** MEDIUM (web search only, no MCP research providers configured for this environment; findings cross-checked across 2+ independent sources per claim where possible — see Sources)

## Feature Landscape

The closest comparable products fall into three families, none of which do exactly what Webspaces does:

- **Personal archive tools** (Timelinize, successor to mholt/timeliner): import everything into a local unified timeline, correlate people/places across sources via an "entity" model with multiple identifiers (email, phone, username), and expose Timeline/Map/Conversations/Gallery views. Closest analog to the identifier-correlation idea, but it **imports and owns a full copy of all content** rather than indexing metadata and fetching live.
- **Personal data warehouse tools** (Dogsheep/Datasette): per-source `*-to-sqlite` importers dump each silo into its own SQLite table; `dogsheep-beta` layers one faceted full-text search index on top. No cross-source identifier correlation — it's "search everything," not "correlate by topic."
- **Enterprise connector-search platforms** (Onyx/Danswer, and adjacent Khoj): 40+ connectors syncing metadata + content into a central index with incremental sync and access control, then a chat/RAG interface answers questions with citations back to source. Built for multi-user orgs; correlation is done implicitly by an LLM/embedding at query time, not by a deterministic per-topic keyword map.

Webspaces' differentiator against all three: **deterministic, config-driven, per-topic correlation using each source's own native categorization** (IMAP folder/label, chat group name, paperless-ngx tag, SilverBullet tag), combined with a **hybrid index+live-fetch model** that avoids both the staleness/duplication of full-import tools and the "black box" feel of AI-inferred correlation.

### Table Stakes (Users Expect These)

Features users assume exist. Missing these = product feels incomplete.

| Feature | Why Expected | Complexity | Notes |
|---------|--------------|------------|-------|
| Webspace config (keyword → native category map per source) | It's the entire mechanism of the product; without it there's no correlation, just a generic multi-source browser | MEDIUM | Config file or config UI; must map to each plugin's native taxonomy concept (folder, group, tag, directory) — this is inherently plugin-specific, not one generic field |
| Cross-source chronological stream per webspace | Core Value in PROJECT.md is "open one webspace, see everything" — a feed is the minimum viable expression of that | MEDIUM | Needs a common sortable timestamp across heterogeneous item types (email date, message timestamp, doc created/modified, note last-edited) |
| Per-source filter within the stream | Every unified-view product surveyed (Onyx, unified inbox tools) offers this; without it a busy webspace becomes noise | LOW | Filter chips/toggles over the already-indexed metadata; no new data needed |
| Inline preview per content type | "See without leaving" is the whole value prop; a feed of bare titles with no preview forces users back into the source anyway | MEDIUM-HIGH | Each content type needs its own renderer: email body (sanitized HTML/plaintext), chat thread excerpt, note markdown render, document thumbnail/first-page. Highest per-plugin surface area item. |
| "Open in source" deep link on every item | Explicit PROJECT.md requirement; also the safety valve that lets the product stay view-only — without it, view-only becomes a dead end | MEDIUM | Capability varies sharply by source — see Deep-Link Mechanics below. Must be treated as a per-plugin contract obligation, not a generic feature. |
| Sync status / freshness indicator (last synced, per source) | Every reviewed connector platform (Onyx, DataHub-style sync UIs) surfaces this; users of a personal tool with flaky home-network sources (LAN bridge, desktop DB reads) will distrust results with no visible freshness signal | LOW-MEDIUM | Minimum: last-synced timestamp + idle/syncing/error state per plugin, non-blocking indicator, hover for exact time. See UI Pattern below. |
| Plugin/connector list with health status | With 5 plugins from day one (IMAP, Signal, WhatsApp, paperless-ngx, SilverBullet), the user needs to see "what's connected and is it working" without reading logs | LOW-MEDIUM | Simple admin/settings screen: plugin name, enabled/disabled, last sync, last error. Does not need per-plugin credential UI if config stays file-based for v1. |
| Read-only guarantee (no accidental mutation) | Explicit constraint in PROJECT.md ("Plugins must never mutate source data stores"); a single accidental write to the Signal/WhatsApp DB could corrupt the user's real chat history | MEDIUM (as a design discipline, not a UI feature) | Enforced at the plugin contract level (e.g., open source DBs read-only / copy-then-read for SQLCipher DBs) rather than in the UI |
| Search within a webspace | Once a webspace has more than a handful of items, browse-only stops scaling; every comparable product (Dogsheep-beta, Onyx, unified inboxes) treats in-scope search as baseline | MEDIUM | Can run over the local metadata/preview index (titles, snippets, tags) without needing full-text of live content |

### Differentiators (Competitive Advantage)

Features that set the product apart. Not required, but valuable.

| Feature | Value Proposition | Complexity | Notes |
|---------|-------------------|------------|-------|
| Native-taxonomy keyword correlation (vs. full-text/semantic search) | Deterministic and explainable — "this item is in the webspace because it's tagged/foldered/grouped X," not "the AI thinks it's related." Zero false positives by construction. This is the one thing none of Timelinize/Dogsheep/Onyx/Khoj do. | MEDIUM | Requires each plugin to expose its native categorization primitive as a first-class, queryable field — a real constraint on the plugin contract, not just a UI nicety |
| Hybrid local index + live fetch (no full content mirroring) | Avoids Timelinize/Dogsheep's tradeoff (own a stale, ever-growing copy of everything) and Onyx's tradeoff (full re-index pipeline, access-control sync) while still being fast to browse | HIGH | The index must carry enough metadata/preview to render the stream and preview pane without a live fetch, but defer full content to open-in-source or an on-demand live fetch; getting the "how much to cache" boundary right is the hardest design decision in the project |
| "Why is this here" provenance on every item | Shows the exact matched keyword + native field (folder/tag/group name) that pulled the item into the webspace — turns the correlation model from implicit to inspectable, which builds trust that full-text/AI tools don't get for free | LOW-MEDIUM | Cheap once the correlation match is computed — just surface the match reason already known at index time |
| Honest, source-appropriate deep-link fidelity | Rather than a single generic "open" button, each plugin declares what precision it can offer (exact document, exact page/message where possible, or best-effort "open the conversation, here's the timestamp to look for") — see Deep-Link Mechanics below | MEDIUM | Sets accurate user expectations instead of the false promise of parity that a generic "open in source" implies |
| Cross-webspace search | Search across *all* configured webspaces at once, not just the one currently open — useful once a user has several active topics and isn't sure which webspace something landed in | MEDIUM | Depends on Search-within-a-webspace and a shared index across webspaces; natural v1.x extension, not required for MVP |
| Documented, stable plugin contract for third-party plugins | Explicit PROJECT.md goal ("addable later, including by other people"); mirrors how Timelinize/Dogsheep/Onyx are all architected as kernel+connectors, validating the shape, but none of them publish a contract aimed at a *personal-tool* third-party audience the way this project intends | HIGH | The contract itself (sync interface, native-category exposure, deep-link builder, preview renderer registration) is a deliverable, not just internal plumbing |

### Anti-Features (Commonly Requested, Often Problematic)

Features that seem good but create problems.

| Feature | Why Requested | Why Problematic | Alternative |
|---------|---------------|------------------|-------------|
| Write/reply/edit in any source (compose email, reply to chat, edit note) | "While I'm looking at it, let me just reply" is a natural UX pull once content is visible | Explicitly out of scope in PROJECT.md; also multiplies the plugin contract's surface area and risk (mutation of chat DBs is exactly what's forbidden) enormously for marginal convenience | Deep link into the real source app for any action beyond viewing |
| AI/agent-inferred, semantic, or context-relative correlation in v1 | Khoj/Onyx both do this and it looks more "magic"; tempting to leapfrog the keyword map | Explicitly deferred in PROJECT.md; undermines the deterministic, no-false-positive value of v1, and couples a correctness-critical MVP to LLM behavior before the plumbing is proven | Ship the configured keyword map first; layer AI-assisted/context-relative search on top once matching, indexing, and deep-linking are solid |
| Full content mirroring / permanent local copy of everything (Timelinize/Dogsheep style) | Feels safer ("my data, all in one place") and enables full-text search without live fetches | Directly conflicts with the chosen hybrid model; grows storage unboundedly, goes stale, and duplicates data that's already reachable live over LAN/local DB; for chat DBs in particular, re-copying full history is unnecessary risk for zero benefit | Metadata + preview in the local index; full content fetched live on open (already the chosen design) |
| Multi-user support / access control / roles | Onyx's admin/curator/basic model looks like "table stakes" because it's the most visible enterprise product studied | This is a single-user personal tool per PROJECT.md; role/permission machinery is pure overhead with no user to serve | None needed — skip entirely |
| Full chat-client UI parity (bubbles, reactions, read receipts, media grid) inside the preview pane | The chat data is right there in the DB, so "just render it properly" feels achievable | Re-implements Signal/WhatsApp's own UI, which PROJECT.md explicitly excludes ("replicating source-app features"); also chases a moving target as those apps evolve | Lightweight thread excerpt (last N messages, sender, timestamp) sufficient to identify the item; full fidelity stays in the real app via deep link |
| A single generic "deep link" abstraction presented identically for every source | Simpler mental model, simpler UI component | Sources are not equally addressable — paperless-ngx and SilverBullet support exact-item deep links, chat apps (Signal/WhatsApp) only support opening a conversation, not jumping to one message (confirmed: `sgnl://` / `signal.me` and `wa.me` / `whatsapp://send` both open a chat with a contact, no message-level target exists) — presenting false parity misleads the user | Per-plugin-declared deep-link fidelity (exact item / exact-with-anchor / conversation-only-plus-manual-locate), rendered with a visibly different affordance where fidelity is lower |
| Unify multiple accounts of the *same* content type (e.g., merge several email inboxes into one) | This is what commercial "unified inbox" products (Canary Mail, Mailbird) actually do, and it's easy to conflate with this project's goal | Solves a different problem (account consolidation) than topic/keyword correlation across *different* content types; not a stated need in PROJECT.md and would dilute the webspace concept | If multiple accounts per source type are ever needed, treat each as a separate plugin instance feeding the same webspace correlation, not a new feature |

## Feature Dependencies

```
Webspace config (keyword -> native category map)
    └──requires──> Plugin contract exposing native categorization per source

Cross-source stream
    └──requires──> Webspace config
    └──requires──> At least one connected plugin
    └──requires──> Hybrid local index (metadata + preview)

Inline preview per content type
    └──requires──> Content-type-specific renderer per plugin
    └──enhances──> Cross-source stream

"Open in source" deep link
    └──requires──> Plugin-declared deep-link builder (per-item, fidelity-aware)
    └──conflicts with──> treating deep-link fidelity as uniform across sources

Search within a webspace
    └──requires──> Hybrid local index
    └──enhances──> Cross-source stream (turns browse into browse+search)

Cross-webspace search (v1.x/v2)
    └──requires──> Search within a webspace
    └──requires──> Shared index across all configured webspaces

Sync status / freshness indicator
    └──requires──> Plugin reports last-sync timestamp + error state to kernel

Plugin/connector management UI
    └──requires──> Plugin discovery/registration mechanism in kernel
    └──enhances──> Sync status indicator (surfaces it per plugin)

"Why is this here" provenance
    └──requires──> Correlation match reason captured at index time (cheap add-on to Webspace config matching)

AI/agent-inferred correlation (v2)
    └──requires──> Webspace config + hybrid index + deep-link mechanics all proven in v1
    └──conflicts with──> shipping deterministic keyword matching as the trusted v1 baseline
```

### Dependency Notes

- **Cross-source stream requires the hybrid local index, not just plugin connectivity:** browsing needs to be fast and uniform across heterogeneous sources (a LAN paperless call is not the same latency as a local Signal DB read); the index is what makes the stream feel like one product instead of five slow round-trips.
- **Deep link requires a per-plugin fidelity-aware builder, not a generic URL field:** paperless-ngx and SilverBullet can target an exact item; IMAP can target an exact message only if the destination mail client supports a Message-ID URI scheme (inconsistent across clients); Signal/WhatsApp can only open the conversation. This must be modeled explicitly in the plugin contract (e.g., a `deepLinkFidelity: exact | anchored | conversation` field) so the UI can render an honest affordance rather than a broken promise.
- **Sync status enhances trust but isn't required for the very first vertical slice:** a single-plugin walking skeleton can ship without it, but it becomes necessary the moment more than one plugin exists and any of them can silently fail (LAN bridge down, Signal DB locked, etc.) — which is true from the second MVP plugin onward.
- **AI/agent-inferred correlation conflicts with the v1 trust model:** PROJECT.md is explicit that this is deferred; introducing it early would blur the "deterministic, explainable, no false positives" guarantee that's the actual differentiator of v1.

## MVP Definition

### Launch With (v1)

Minimum viable product — matches PROJECT.md's "Active" requirements almost exactly.

- [ ] Webspace config map (keyword ↔ native categorization per source) — the mechanism itself
- [ ] Kernel + plugin architecture with a documented contract — needed from day one since 5 plugins ship at once
- [ ] Email (IMAP via Proton Bridge), Signal, WhatsApp, paperless-ngx, SilverBullet plugins — the user's actual silos
- [ ] Hybrid data model (index metadata/preview locally, fetch full content live on open) — core architectural bet, must be validated early
- [ ] Web UI: cross-source stream + detail pane, filterable by source, inline preview per content type, "open in source" deep link on every item — the smallest complete expression of Core Value

### Add After Validation (v1.x)

Features to add once core is working and the user has lived with it.

- [ ] Search within a webspace — once webspaces have enough items that scrolling stops working
- [ ] Sync status / freshness indicators per plugin — once more than one plugin exists and any can silently go stale (true almost immediately given 5 plugins)
- [ ] Plugin management/health UI — once the user needs to diagnose "why isn't X showing up" without reading logs
- [ ] "Why is this here" provenance display — cheap addition once correlation matching is stable

### Future Consideration (v2+)

Features to defer until the deterministic v1 is proven.

- [ ] Cross-webspace / global search — defer until multiple webspaces are in active daily use and "which webspace was that in" becomes a real friction point
- [ ] AI/agent-inferred, context-relative correlation — explicitly deferred per PROJECT.md; layer on top of proven plumbing, not instead of it
- [ ] Local filesystem / project-directory plugin — explicitly deferred per PROJECT.md
- [ ] Additional third-party plugins via the published contract — depends on the contract itself being stable and documented, which is a v1 deliverable but the plugins built against it are naturally later

## Feature Prioritization Matrix

| Feature | User Value | Implementation Cost | Priority |
|---------|------------|---------------------|----------|
| Webspace config map | HIGH | MEDIUM | P1 |
| Kernel + plugin contract | HIGH | HIGH | P1 |
| Cross-source stream + detail pane | HIGH | MEDIUM | P1 |
| Inline preview per content type | HIGH | HIGH | P1 |
| Open-in-source deep link | HIGH | MEDIUM | P1 |
| Hybrid index + live fetch | HIGH | HIGH | P1 |
| Search within a webspace | MEDIUM-HIGH | MEDIUM | P2 |
| Sync status / freshness indicator | MEDIUM | LOW-MEDIUM | P2 |
| Plugin management/health UI | MEDIUM | LOW-MEDIUM | P2 |
| "Why is this here" provenance | MEDIUM | LOW | P2 |
| Cross-webspace search | MEDIUM | MEDIUM | P3 |
| AI/agent-inferred correlation | HIGH (long-term) | HIGH | P3 |
| Filesystem/project-directory plugin | MEDIUM | MEDIUM | P3 |

**Priority key:**
- P1: Must have for launch
- P2: Should have, add when possible
- P3: Nice to have, future consideration

## Deep-Link Mechanics Per Source (feeds the "Open in Source" feature)

| Source | Best available fidelity | Mechanism | Confidence |
|--------|--------------------------|-----------|------------|
| paperless-ngx | Exact document | `https://<host>/documents/<id>/details` — stable numeric DB primary key, safe to persist at index time | MEDIUM |
| SilverBullet | Exact page (and header anchor) | App is served as a web app; a page is directly reachable at `http://<host>:<port>/<PageName>` (URL-encoded); internal `[[page#header]]` link syntax confirms header-level anchors are addressable | MEDIUM |
| IMAP email | Exact message, client-dependent | Requires a client-specific URI scheme plus percent-encoded `Message-ID` (e.g. iOS Mail `message:<id>`); the message must already be present/downloaded in that specific client; no universal cross-client scheme exists, and Proton's own webmail deep-link format was not independently verified in this pass (flagged as a gap) | LOW-MEDIUM |
| Signal | Conversation only, not a specific message | `sgnl://` URI scheme (desktop) or `signal.me/#p/<phone>` open a chat with a contact; no public message-level deep-link target exists | MEDIUM |
| WhatsApp | Conversation only, not a specific message | `wa.me/<number>` / `whatsapp://send?phone=` open a chat with a contact (falls back to WhatsApp Web/download page if not logged in); no message-level deep-link target exists | MEDIUM |

**Design implication:** the plugin contract should carry an explicit deep-link fidelity level per item (exact / anchored / conversation-only) so the UI can render a different, honest affordance for chat items ("open conversation — look for message at [timestamp]") versus document/note items ("open exact item"). Treating all five sources as equally deep-linkable would overpromise on two of the five MVP plugins.

## Sync Status / Freshness UI Pattern (feeds the sync indicator feature)

Cross-domain research (data connector platforms, ingestion tooling, sync UX guides) converges on the same shape, applicable directly to Webspaces:

- Per-source (not just global) status: idle / syncing / error — with plugin identity visible (which of the 5 sources)
- Last-synced timestamp: relative display ("5 min ago") with exact time on hover
- Non-blocking, low-emphasis placement (small badge/icon in a toolbar or per-plugin row in a settings screen), not a modal or banner for the normal case
- Errors are the exception to "low emphasis" — they need a visibly distinct state and a retry affordance, since a silently-failed plugin (e.g., LAN bridge unreachable, Signal DB locked by the desktop app) is indistinguishable from "nothing new happened" without one

## Competitor Feature Analysis

| Feature | Timelinize | Onyx/Danswer | Dogsheep | Khoj | Our Approach |
|---------|--------------|--------------|----------|------|--------------|
| Cross-source correlation | Entity model: links same person across sources by shared identifier (email/phone/username) | Implicit via LLM retrieval over all indexed content; no explicit identifier linking | None — separate tables per source, unified only by a shared search index | Implicit via semantic/vector similarity | Deterministic, explicit: configured keyword matched against each source's *native* categorization (folder/tag/group) |
| Content storage | Full local copy of everything (owns the data permanently) | Indexes content + metadata centrally; access-control synced | Full local copy per source (SQLite exports) | Vector index of documents; some full-text | Hybrid: metadata + preview only in local index; full content fetched live on open |
| Search scope | Search within the unified timeline | Unified conversational search/chat across all 40+ connectors | Faceted search across all imported sources (`dogsheep-beta`) | Semantic search + chat over knowledge base | v1: search within a webspace; v1.x/v2: cross-webspace search |
| Deep link back to source | Not a primary design goal (data is imported, timeline *is* the source of truth) | Citations link back to originals for verification | Not a focus — data already exported/owned locally | Some source citation in chat answers | First-class, per-item, fidelity-aware "open in source" — core requirement, not an afterthought |
| Multi-user / access control | No (personal tool) | Yes — roles (admin/curator/basic), access-control sync | No (personal tool) | Supports both personal and enterprise/cloud modes | No — explicitly single-user by design |
| Write-back to source | No | No (search/chat product, not an editor) | No | Chat can act as an agent in some configs | No — explicitly view-only by design |
| Plugin/connector extensibility | Built-in data sources, not documented for third parties in the same way | Documented connector framework, 40+ built-in | Individual `*-to-sqlite` tools, loosely coupled, community-extensible | Some integration surface (Obsidian plugin, REST/MCP) | Documented plugin contract, explicit goal of third-party extensibility |

## Sources

- [Timelinize homepage](https://timelinize.com/) — entity model, views
- [Timelinize GitHub](https://github.com/timelinize/timelinize)
- [Timelinize data sources docs](https://timelinize.com/docs/data-sources/media)
- [Dogsheep GitHub org / homepage](https://dogsheep.github.io/)
- [Simon Willison — Personal Data Warehouses](https://simonwillison.net/2020/Nov/14/personal-data-warehouses/)
- [Talk Python To Me — Personal search engine with datasette and dogsheep](https://talkpython.fm/episodes/show/299/personal-search-engine-with-datasette-and-dogsheep)
- [Onyx documentation — RAG and Search](https://docs.onyx.app/overview/core_features/internal_search)
- [Onyx GitHub](https://github.com/onyx-dot-app/onyx)
- [Khoj — Obsidian plugin listing](https://community.obsidian.md/plugins/khoj)
- [Khoj explained — HoangYell](https://hoangyell.com/khoj-explained/)
- [Canary Mail unified inbox](https://canarymail.io/features/unified-inbox)
- [NSHipster — Message-ID and Mail.app Deep Linking](https://nshipster.com/message-id/)
- [paperless-ngx REST API docs](https://docs.paperless-ngx.com/api/)
- [paperless-ngx GitHub discussion — Easily get and locate documents by ID](https://github.com/paperless-ngx/paperless-ngx/discussions/8555)
- [SilverBullet — Link documentation](https://silverbullet.md/Link)
- [SilverBullet — Tags, Links, and Backlinks (DeepWiki)](https://deepwiki.com/silverbulletmd/silverbullet/6.4-tags-links-and-backlinks)
- [Signal's newish URI scheme](https://shkspr.mobi/blog/2023/02/signals-newish-uri-scheme/)
- [signal.me URLs — Signal Wiki](https://signal.miraheze.org/wiki/Signal.me_URLs)
- [AppsFlyer — WhatsApp deep links](https://www.appsflyer.com/blog/deep-linking/whatsapp-deep-link/)
- [DataHub — Sync Status docs](https://docs.datahub.com/docs/sync-status)
- [Fivetran — connectors sync overview](https://fivetran.com/docs/core-concepts/syncoverview)
- Note: Proton Mail webmail's own deep-link URL format for a specific message was not independently verified in this research pass (search unavailable) — flagged as an open gap for phase-specific research if/when the email plugin needs to deep-link into Proton's own webmail rather than a generic IMAP client.

---
*Feature research for: local-first personal cross-source data aggregation ("Webspaces")*
*Researched: 2026-07-27*
