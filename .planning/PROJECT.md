# topos

## What This Is

topos is a locally-run service that pulls together related information from disparate personal data silos — email accounts, Signal and WhatsApp chats, document storage, note-taking apps, wikis, and project directories — into per-topic "webspaces". Instead of visiting each application to piece together the state of a project or topic, you open its webspace and see all related objects in one rich, view-only interface, with deep links back to the original application for anything beyond viewing.

## Core Value

Open one webspace and instantly see and grok all related information across every silo — without visiting each data store individually.

## Current State

**v1.0 MVP shipped 2026-08-12** — 12 phases (10 planned + 2 inserted), 92 plans, 187 tasks, 798 commits in 17 days (2026-07-27 → 2026-08-12).

- All five sources live: paperless-ngx, SilverBullet, Proton/IMAP, Signal, WhatsApp — each a separate go-plugin subprocess behind the published `topos.v1` gRPC contract (contract generation "topos.v2")
- ~42k LOC hand-written Go (kernel + 5 plugins + mock/mockstrict), ~25k LOC Svelte 5/TypeScript SPA, hermetic Playwright e2e suite (42 tests) gating CI alongside Go tests/svelte-check/vitest
- Webspaces are built and edited entirely from the UI (hot-apply config writes); works on mobile widths; first run bootstraps a default config
- Release engineering live: change-gated nightlies, tag-triggered release artifacts (static CGO_ENABLED=0; Signal plugin deliberately excluded, built locally via `make signal`), GitHub milestone mirror script
- Known operational risk: WhatsApp linked-device session can be de-linked/banned by Meta at any time; plugin degrades honestly, captured messages survive
- **v1.1.0 Plugin Ecosystem shipped 2026-08-18** — 4 phases (11–14), 32 plans + 2 gap closures + 5 quick tasks, 309 commits in 6 days. Third-party plugin path proven end-to-end: external trust tier (provenance + pins + manifest gate), Google Drive source built clean-room out-of-repo against the published contract, per-item curation, PWA install. Milestone audit passed (11/11 requirements), phase-14 security audit 26/26 closed. See `.planning/MILESTONES.md` and `milestones/v1.1.0-*`.
- **Next milestone:** not yet defined — run `/gsd-new-milestone`. Seeded candidates: kernel OAuth/secrets services for plugins (requirement-grade todo), backlog 999.1 (distribution/dev guide/certification) and 999.2 (`topos-plugins` repo restructure + trust-model discussion)

## Current Milestone: v1.1.0 Plugin Ecosystem

**Goal:** Open topos to third-party plugins behind an explicit trust boundary, prove the external path by building a new source out-of-repo, and give users finer control over what lands in a webspace.

**Target features:**
- External plugin support — kernel loads out-of-repo plugin binaries; trusted = built from the `davison/topos` repo, everything else marked untrusted with a warning when adding (load + trust marking only; distribution, dev guide, and certification deferred)
- Local/network filesystem plugin — docs in a folder, optionally subfolders; built in-repo as a trusted plugin (the MVP-deferred source)
- Google Drive plugin — docs-in-a-folder over the Drive API; deliberately built *out-of-repo* against the published contract to dogfood the external-plugin mechanism end to end (OneDrive deferred)
- Per-item include/exclude — mark individual stream entries in or out of a webspace; the final tier of the filter-config hierarchy, and the kernel's first user-owned data beyond config
- PWA installability — ServiceWorker + manifest/assets so the app installs on desktop and mobile

**Key context:** Milestone version labels are full semver from now on (`v1.1.0`) so the GSD completion tag directly triggers `release.yml` — no companion tag at close. IMAP-vanilla refactor deferred. Carried-over minor items ride along only when a phase touches their area.

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
- ✓ Webspace builder UI (KERN-08, UI-12): the kernel API gained its first mutating surface (`PUT /api/config` + reload/describe-plugin, content-hash optimistic lock, canonical TOML rewrite, secrets stay environment-only as `${VAR}` references), config changes hot-apply without restart (plugin host/coordinator/scheduler rebuilt in place, removed instances' index rows purged synchronously), and the UI composes webspaces end to end — create/switch/delete webspaces, add/edit/remove named source instances from the chip row, save a live search as a permanent webspace filter — Phase 7 (three UAT rounds; two gap-closure waves; 108-threat STRIDE register audited to threats_open: 0)

- ✓ Pre-1.0 UI polish and source-management rework — plugin-provided identity icons (icons travel in the plugin contract's Describe response; kernel serves them with a MIME allowlist, 64 KiB cap, and CSP-sandboxed route; real paperless-ngx/SilverBullet marks, Lucide-derived glyphs elsewhere, all with audited four-key provenance), favicon + `Disallow: /` robots.txt, popover surface tone, search clear button and media-previewer geometry fixes, per-chip menu (refresh moved in, syncing-guarded), one-click Reload config with re-entry guards, and a two-section source picker (configured instances with location lines + install-a-plugin catalog driven solely by the kernel's plugin-types API) — Phase 9 (27-threat STRIDE register closed to threats_open: 0)

- ✓ WhatsApp plugin (SRC-03) — whatsmeow linked device with its own persistent message store (session + captured messages, both plugin-owned; source stores never touched), one-time QR link via terminal CLI *and* in-app QR panel (kernel link-session endpoint suspends the running instance so link mode gets the store lock), live group-subject/push-name sync (history sync alone carries neither — found by the plan's mandatory real-device spike), conversation-day digests matching group names and saved 1:1 contact names, five named health states that degrade honestly without ever emptying the stream (verified live: `401 logged out from another device`, captured rows survive), wa.me/web.whatsapp.com deep links (no `whatsapp://` handler exists on Linux), read-only + egress AST guards, hermetic Playwright coverage — Phase 8 (verified 5/5 after three gap-closure waves ending in G-08-5's supervisor lock split — a slow plugin relaunch can no longer freeze other sources' routes — confirmed by a human real-device re-link with a second source staying responsive throughout)

- ✓ External plugin support with an enforced trust boundary — kernel discovers and launches out-of-repo plugin binaries from a configured external directory, tier derived from provenance (in-repo dir = trusted, external dir = untrusted); external launches gated on content-hash pins with a soft per-instance failure channel (pin mismatch names the hash change on the chip, two-click re-pin re-establishes trust), scrubbed plugin launch environment, per-instance extras threaded through the published contract, explicit informed consent before adding an untrusted source and a persistent untrusted badge everywhere it appears; `ResolveBinary` confines plugin names to bare filenames (traversal/tier-escape closed, CR-01) — Phase 11 (proven end-to-end with a real out-of-repo binary built by its own module; verified 5/5 after one gap-closure plan)
- ✓ Filesystem source (SRC-04) — a local or network folder becomes a source via the in-repo filesystem plugin: document-scope classification with per-instance `include_glob` widening (Fetch classifies through the instance's own scope, so glob-admitted files get honest metadata-only previews, never a false 404), subfolder recursion as a typed on/off setting, stat-diff polling sync that works on NFS/SMB mounts where OS change notifications never fire, deep links open in the desktop's own file handler through the kernel's fsopen route (resolved-path read/exec discipline, WR-02), read-only behavior enforced by committed guards, full four-key provenance including `source_system`; the same binary copied into the external plugins directory loads and syncs identically under the untrusted badge — the external-path rehearsal before Google Drive — Phase 12 (verified 7/7 after two gap-closure plans)
- ✓ Webspaces defined in a config map matching each source's *native* categorization (IMAP folders/labels, chat group and contact names, paperless-ngx tags, SilverBullet tags/pages) — v1.0 (Phases 1–4 proved keyword matching; Phase 5 upgraded to per-instance typed match blocks with keywords as fallback; Phase 8 closed the last source)
- ✓ Web UI: stream + detail pane — chronological cross-source feed per webspace, filterable by source, inline preview (email body, chat thread, note, document), "open in source" deep link on every item — v1.0 (built out across Phases 1–9.1, including mobile layout and first-run bootstrap)

- ✓ Per-item curation (KERN-09, KERN-10, UI-13, UI-14) — users get the last word on webspace contents: exclude any stream item (single, bulk, or from the detail pane) with a 5s undo toast, marks persist in the kernel index and survive re-sync/restart/index-rebuild while always outranking automatic match rules, an excluded-items view lists exactly what was removed and un-excludes on click; app installs as a PWA on desktop and mobile (ServiceWorker + manifest/assets, update notice via the shared toast layer) — Phase 13 (verified 5/5 after one gap-closure plan closed G-13-1, the cross-webspace undo skeleton strand, with a stale-generation entry guard in `load()` pinned by a RED-first browser spec)

### Active

- [ ] Google Drive source plugin, built out-of-repo against the published contract (dogfoods the external-plugin path)

Deferred candidates (not this milestone): IMAP-vanilla refactor with provider extensions; OneDrive plugin; pull-by-URL distribution, dev guide, certification. Advisory review items (10-REVIEW warnings, 06-REVIEW WR-01) and pending todos ride along only if a phase touches their area.

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
| Build greenfield rather than extend Timelinize/Onyx/Dogsheep | None do identifier-correlated cross-source views; correlation is the core, not a bolt-on | v1.0: ✓ Good — five sources correlated in 17 days; no adopted-codebase constraints fought |
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
| Hermetic browser e2e suite is the regression gate; UI phases extend it as definition of done (07.1 D-11 standing rule) | Manual UAT doesn't survive codebase churn; a Playwright suite driving the shipped binary hermetically (temp config, ephemeral port, mock plugins, zero credentials) makes Phase 7's behavior permanent armor before Phase 8 churns it | Phase 07.1: shipped — 26 specs green locally and on first-ever CI run (GitHub Actions, Chromium, zero retries); smoke script retired; 4 real pre-existing bugs flushed out and fixed |
| Plugin identity icons travel in the plugin contract (Describe response), not a kernel-side lookup table | The kernel holds no table of known plugin types (D-05 discipline); a third-party plugin ships its own mark; kernel enforces the trust boundary (MIME allowlist of svg/png, 64 KiB cap, CSP `default-src 'none'; sandbox` + nosniff on the serving route) | Phase 9: shipped — all seven plugin modules carry audited four-key provenance; `internal/audit` fails the build on a missing key |
| Real upstream logo marks for paperless-ngx/SilverBullet; generic Lucide glyphs for Proton/Signal/WhatsApp | Recognizability where a permissively-usable mark exists; a real WhatsApp/Meta mark would misrepresent an unofficial client (SilverBullet has no vector mark upstream — its square dock PNG is wrapped in an `<svg><image>` data URI, upstream's own technique) | Phase 9: shipped — legibility at 14–16px and brand-policy recheck both confirmed at UAT |
| Signal plugin binary excluded from published release/nightly artifacts (option-b, user-decided at checkpoint) | Every published artifact stays static CGO_ENABLED=0 and runs on any distro; a runner-built cgo binary links Ubuntu's SQLCipher with no execution in CI to catch a mismatch; `make signal` is the already-documented local build | Phase 10: shipped — release path proven live against GitHub API; decision recorded dated in docs/releasing.md |
| `.planning/` is source of truth for milestones; GitHub is a one-way mirror via idempotent `scripts/sync-milestones.sh` (GSD has no native GitHub-milestone sync) | Bidirectional sync makes an unreviewed GitHub edit authoritative; lookup-by-title create-or-patch reconciles the pre-existing v1.0 milestone without duplicating; script has no delete path at all | Phase 10: shipped — verified live twice against real milestone #1, idempotency confirmed |

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
*Last updated: 2026-08-18 after Phase 14 completion (Google Drive source, built out-of-repo)*
