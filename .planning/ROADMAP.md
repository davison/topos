# Roadmap: topos

## Overview

topos is built as a series of vertical slices: every phase ends with the user able to open a browser, pick a webspace, and see more of their life in it than they could before. Phase 1 delivers the whole spine — config, index, plugin contract, HTTP API, stream, detail pane — proven by one real low-risk source (paperless-ngx) rather than a mock. Phase 2 adds a structurally different second source (SilverBullet), which is what actually proves the plugin contract is source-agnostic, plus the operability surface (filter, health, refresh, staleness, agent permission gating) that makes a multi-source view trustworthy. Phases 3–4 then add one source per phase in strict ascending order of integration risk: IMAP email, then Signal. Restructured 2026-08-05: before the final and riskiest source (WhatsApp), three phases of architecture and UI work land first — named source instances with per-plugin-type matching config (Phase 5, reversing the single shared keyword list while only in-repo plugins exist), a scalable source UI surface (Phase 6), and a webspace builder UI (Phase 7). WhatsApp closes the milestone as Phase 8 — by then the system is already useful and complete without it, so a ban or de-link degrades one plugin rather than sinking the milestone.

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: First Webspace, End to End** - Kernel spine, plugin contract, and web UI proven by real paperless-ngx documents in a browsable stream (completed 2026-07-28)
- [x] **Phase 2: Two Sources, One Trustworthy Stream** - SilverBullet joins the stream; filter, health, refresh, staleness states, and default-deny agent permissions (completed 2026-07-29)
- [x] **Phase 3: Email in the Webspace** - Proton/IMAP mail appears in the stream, never marked read, with full-text search across the webspace (completed 2026-07-31)
- [x] **Phase 4: Signal Conversations** - Signal Desktop history in the stream, read strictly read-only with keyring and schema-version safety (completed 2026-08-03)
- [x] **Phase 5: Source Instances & Per-Type Matching** - Sources become named plugin instances, each with matching config typed to its plugin, replacing the single shared keyword list (completed 2026-08-06)
- [x] **Phase 6: UI — Scalable Source Surface** - Header combines health and filtering into one scalable per-source affordance; deep-link fidelity differentiation, detail-pane search highlighting, themed scrollbars with date markers (completed 2026-08-06)
- [x] **Phase 7: Webspace Builder UI** - Configure plugin instances and webspaces from the UI; saved searches become permanent webspace filters (completed 2026-08-09)
- [ ] **Phase 8: WhatsApp Conversations (Managed Risk)** - Linked-device WhatsApp history in the stream, degrading gracefully on de-link or ban (gap closure in progress — 08-REVIEW.md CR-01, orphaned link session)

## Phase Details

### Phase 1: First Webspace, End to End

**Goal**: User can define a webspace, open it in a browser, and see their paperless-ngx documents for that topic as one chronological stream with inline previews and working deep links back to paperless-ngx
**Mode:** mvp
**Depends on**: Nothing (first phase)
**Requirements**: KERN-01, KERN-02, KERN-03, PLUG-01, PLUG-02, PLUG-03, SRC-04, UI-01, UI-03, UI-04, AGENT-02
**Success Criteria** (what must be TRUE):

  1. User can define a webspace in config — a keyword mapped to each source's native categorization — start the service, and open that webspace in a browser
  2. Documents whose paperless-ngx tags match the webspace keyword appear in the stream in chronological order, having been correlated once at sync time and persisted in the local index (not recomputed per request)
  3. Clicking an item opens a detail pane showing the document content fetched live from paperless-ngx, while the stream itself renders from locally indexed metadata and previews
  4. Every item carries an "open in source" affordance that lands on the exact paperless-ngx document, and declares its link fidelity (exact / anchored / conversation-only)
  5. The plugin runs as a separate subprocess over the published `.proto` contract, that contract contains no source-mutating method, and the same items the UI shows are retrievable from the kernel HTTP API with stable IDs, structured content, and machine-readable provenance

**Plans**: 6/6 plans executed
Plans:
**Wave 1**

- [x] 01-01-PLAN.md — Walking skeleton tracer: config → plugin subprocess → SQLite index → HTTP API → browser, against real paperless-ngx

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 01-02-PLAN.md — Detail pane: live content fetch through the plugin, hardened content routes, open-in-source with declared fidelity

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 01-03-PLAN.md — Stream presentation and every UI state, including a failed sync never masquerading as an empty webspace
- [x] 01-04-PLAN.md — Published plugin + HTTP contracts, contract-conformance tests enforcing read-only and the agent envelope, operator docs

**Gap closure** *(from UAT — see 01-UAT.md `## Gaps`)*

- [x] 01-05-PLAN.md — G-01-2: root layout never imported app.css, so the build shipped zero CSS — fix, guard against stylesheet-less builds, hand off for UAT re-run of tests 2–4
- [x] 01-06-PLAN.md — G-01-6: committed outbound-host allowlist — host-pinned paperless client plus a repo-wide egress audit *(runs after 01-05)*

**UI hint**: yes

Notes:

- This is the walking skeleton: normalized Item schema, index store, webspace config loader, correlation engine, plugin host, HTTP API, and SPA shell all land here, but they land behind one *real* source so the slice is user-visible from day one.
- The contract must be sketched against at least two structurally different sources (documents and chat) before it is written, even though only paperless-ngx ships in this phase — Phase 2 is where that pays off.
- Research: MINIMAL. Kernel/plugin-host patterns are established (Timelinize, Home Assistant coordinator, hashicorp/go-plugin). Confirm the exact paperless-ngx document deep-link path during planning.

### Phase 2: Two Sources, One Trustworthy Stream

**Goal**: User sees two structurally different sources interleaved in one webspace and can filter, refresh, and tell at a glance whether what they are looking at is complete and current
**Mode:** mvp
**Depends on**: Phase 1
**Requirements**: KERN-04, PLUG-04, PLUG-05, SRC-05, UI-02, UI-05, UI-06, AGENT-01
**Success Criteria** (what must be TRUE):

  1. SilverBullet pages matching the webspace keyword by tag or page name appear interleaved with paperless documents in one chronological stream, each with an exact deep link to its page
  2. User can filter the stream down to a single source and back to all sources
  3. User can see per-source health in the UI — reachable, last sync time, last error — and trigger a manual refresh that brings in new items without duplicating existing ones or stacking concurrent syncs
  4. An item whose source is unreachable or has been deleted shows an explicit unavailable/stale state in the stream and detail pane, never a blank pane or a silent 404
  5. Config grants agent read access and action hand-off separately, per plugin, defaulting to deny — a plugin with no grant is absent from agent-facing API responses

**Plans**: 6 plans
Plans:
**Wave 1**

- [x] 02-01-PLAN.md — Tracer: SilverBullet joins the stream end to end, plus the source-scoped persistence fix a second source requires

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 02-02-PLAN.md — Single-flight sync coordinator, background scheduler, and the per-source health + manual-refresh HTTP surface

**Wave 3** *(blocked on Wave 2 completion; both plans run in parallel)*

- [x] 02-03-PLAN.md — UI trust surface: health chips, source filter, staleness and unavailable states
- [x] 02-04-PLAN.md — Default-deny agent grants in a `/agent/v1` namespace, plus the reference mock plugin and the PLUG-05 validation exercise

**Gap closure** *(from 02-VERIFICATION.md — 1 blocker, truth #6 failed)*

- [x] 02-05-PLAN.md — SilverBullet `Match` must fail with `codes.Unavailable` on a non-404 read failure instead of returning a silently-empty success that wipes the index

**Gap closure** *(from 02-UAT.md — G-02-1, major)*

- [x] 02-06-PLAN.md — Remove the `app.css` theme entries that shadow Tailwind's container scale (collapsing the health-chip tooltip and the index page column), plus a built-stylesheet recurrence guard

**UI hint**: yes

Notes:

- Two structurally different sources (documents vs. wiki pages) is the real test of PLUG-05; validation is a third party — or a fresh context with no repo access beyond the published `.proto`, contract docs, and reference mock plugin — building a working plugin from those artifacts alone.
- The per-plugin sync coordinator lands here (dedup concurrent refreshes, single source of truth for health), so every plugin added in Phases 3–5 inherits it.
- Agent permissions are established now, while only two plugins exist, so Phases 3–5 add their grants declaratively rather than retrofitting a permission model across five plugins.
- Research: MINIMAL. Coordinator and sync-status patterns are well documented; confirm SilverBullet's HTTP API tag/page query shape during planning.

### Phase 3: Email in the Webspace

**Goal**: User's Proton mail for a topic appears in the webspace stream and stays readable there, and the volume it brings is navigable by searching within the webspace
**Mode:** mvp
**Depends on**: Phase 2
**Requirements**: SRC-01, KERN-05
**Success Criteria** (what must be TRUE):

  1. Emails in Proton folders or labels matching the webspace keyword appear in the stream with sender, subject, and date, and render their body inline in the detail pane
  2. Reading an email inside webspaces never marks it read in Proton — proven by an automated test asserting the `\Seen` flag is unchanged after a full sync and a detail fetch
  3. An email carrying several Proton labels appears exactly once in the stream, deduped by Message-ID
  4. User can type a query inside a webspace and get ranked, clickable matches across every source in that webspace
  5. The plugin reaches Proton Mail Bridge over LAN despite its self-signed certificate, and reports a clear, actionable health error rather than hanging when the bridge is unreachable

**Plans**: 10 plans
Plans:
**Wave 1** *(both plans run in parallel — zero file overlap)*

- [x] 03-01-PLAN.md — Tracer: one real Proton email travels config → IMAP over TLS → EXAMINE/ENVELOPE scan → Message-ID dedup → index → stream row with sender
- [x] 03-03-PLAN.md — Full-text search (KERN-05): FTS5 external-content index with backfill, `Store.Search`, and `GET /api/webspaces/{ws}/search`

**Wave 2** *(both plans run in parallel; blocked on Wave 1)*

- [x] 03-02-PLAN.md — Email bodies render inline (sanitized HTML rendition + detail-pane sender), plus four independent read-only proofs incl. the `\Seen`-unchanged test
- [x] 03-04-PLAN.md — Search UI: search box, ranked results reusing StreamRow with highlighted snippets, and the full state matrix

**Wave 3** *(gap closure from 03-VERIFICATION.md; blocked on Wave 2)*

- [x] 03-05-PLAN.md — Gap closure: the fetched INTERNALDATE reaches `Item.TimestampUnix` so emails carry a date and interleave chronologically, plus the logged empty-Message-Id skip and TLS-comment hygiene

**Wave 4** *(gap closure from the 03-VERIFICATION.md re-verification; blocked on Wave 3)*

- [x] 03-06-PLAN.md — Gap closure: the Proton plugin's `source_id`→mailbox state accumulates across webspaces instead of being replaced per `Match`, so email bodies render for every webspace and not just the last-synced one; plus the detail pane's date comes from the shared UTC-pinned formatter

**Wave 5** *(gap closure from the second 03-VERIFICATION.md re-verification; blocked on Wave 4)*

- [x] 03-07-PLAN.md — Gap closure: `plugins/proton`'s declared `golang.org/x/net` is raised past the CVE-2024-45338 fix boundary (the HTML tokenizer `RenderSanitizedEmail` runs over arbitrary inbound email), with the workspace-selected version and the rendered output both proven unchanged, plus a repo-wide audit test that fails if any of the six workspace modules declares a below-floor dependency again

**Wave 6** *(gap closure from 03-UAT.md gap G-03-1; blocked on Wave 5)*

- [x] 03-08-PLAN.md — Gap closure (code-side half of G-03-1; the `.env` credential correction remains a user action): a rejected Bridge LOGIN whose configured token cannot be a Bridge-generated app password now says so and says where the real one lives, wired once in `client.connect` so it reaches both `HealthResponse.LastError` and the `sync_runs`-fed `last_error` behind the UI's red dot — warning-grade, never blocking a connection attempt — plus the live-Bridge test's misleading username-pointing hint replaced by a reference to that same shared constant

**Wave 7** *(gap closure from 03-UAT.md gap G-03-2, major; blocked on Wave 6)*

- [x] 03-09-PLAN.md — Gap closure: the email detail pane becomes readable — the plugin (never the shared pane, which SilverBullet's both-fields response rules out) returns the plain-text part as the content whenever a message has one, so no rendition is offered for it; the sanitized rendition is kept as the fallback for HTML-only mail and made readable there, with the theme's own colours outranking every email-supplied inline colour and images that can never load under the rendition CSP no longer painting as broken icons — sanitizer allowlist, CSP and every standing security assertion unchanged

**Wave 8** *(gap closure from 03-UAT.md gap G-03-3, minor; blocked on Wave 7 — both plans edit `plugins/proton/plugin.go`)*

- [x] 03-10-PLAN.md — Gap closure: "Open in Proton Mail" stops landing on the bare inbox — the label-name path Proton cannot resolve (custom labels are addressed by internal id, not name) is replaced by a search over the account's All Mail view for the message's subject, built only from configured input, inert against any subject a sender can write, absent when there is no subject, rune-capped, and still honestly declared ANCHORED

**UI hint**: yes

Notes:

- Search lands with email because email is the first source that brings enough volume to make scrolling insufficient.
- Extends the Phase 1/2 UI contract with one new renderer (email body) and one new control (search box) — no new interaction model.
- Research: MEDIUM. A short spike should settle Proton Bridge LAN exposure (tunnel vs. rebind), certificate pinning in the Go IMAP client, and the Proton webmail deep-link format, which was not verified during landscape research.

### Phase 4: Signal Conversations

**Goal**: User's Signal conversations for a topic appear in the webspace stream, read from Signal Desktop's local database without any risk of corrupting or altering it
**Mode:** mvp
**Depends on**: Phase 3
**Requirements**: SRC-02
**Success Criteria** (what must be TRUE):

  1. Messages from Signal conversations or groups whose names match the webspace keyword appear in the stream in chronological order alongside documents, notes, and mail
  2. Detail pane shows the surrounding conversation thread, with an "open in Signal" affordance declared as conversation-only fidelity
  3. Signal Desktop's database is opened strictly read-only (`mode=ro`) and is byte-identical after a full sync, including when Signal Desktop is running at the same time
  4. The decryption key is obtained from whichever keyring backend the user's install actually uses, detected at runtime rather than assumed
  5. An unrecognised Signal database schema version fails loudly, naming the version it found, instead of silently importing nothing

**Plans**: 4 plans

Plans:
**Wave 1**

- [x] 04-01-PLAN.md — Tracer: a real Signal conversation-day digest reaches the stream, read-only, through a local-path source the kernel config now accepts

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 04-02-PLAN.md — Read-only proven by construction, key resolution on either config.json shape, and failures that name themselves in the health chip

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 04-03-PLAN.md — The chat-thread transcript Phase 5 reuses, message richness, validated "open in Signal", and the published local-path contract

**Gap closure** *(from 04-UAT.md — G-04-1, blocker)*

- [x] 04-04-PLAN.md — Emit the E.164 verbatim so Signal Desktop accepts the contact-form deep link, replacing escape-the-plus with allowlist validation

Notes:

- Single requirement, but a full vertical slice with a mandatory spike — kept separate from Phase 5 so chat access is proven on the tractable source before the unpredictable one.
- Introduces the chat-thread renderer that Phase 5 reuses; extends the existing UI contract rather than defining a new one.
- Research: MEDIUM-HIGH. Spike before planning: Signal Desktop DB schema, `safeStorage` keyring backend extraction tested hands-on against the user's actual Arch/DE setup, schema-version detection, and SQLCipher/SQLite version stability (pin SQLite ≥ 3.51.3; never VACUUM or checkpoint).

### Phase 5: Source Instances & Per-Type Matching

**Goal**: Sources become named instances — the same plugin type can be configured multiple times under user-chosen display names — and each instance declares matching config appropriate to its plugin type, replacing the single shared keyword list
**Mode:** mvp
**Depends on**: Phase 4
**Requirements**: KERN-06, KERN-07
**Success Criteria** (what must be TRUE):

  1. User can configure two instances of the same plugin type with distinct display names (e.g. "Home email" and "Work email") and see them as separate sources in the stream, source filter, and health UI
  2. Each source instance carries its own matching configuration, typed to its plugin (IMAP folders/labels, document tags, chat conversation/group names, wiki tags/pages), replacing the single shared per-webspace keyword list; all five existing sources migrate to the new shape
  3. Source identity throughout the kernel — index rows, sync runs, agent grants, HTTP API, and UI display — is the named instance, never the bare plugin type; existing webspace data migrates or re-syncs cleanly with no orphaned rows
  4. The contract change is published: `docs/plugin-contract.md`, `proto/webspaces/v1/`, `config.example.toml`, and the mock plugin all reflect per-instance match config, and the standing contract tests (read-only AST, RPC allowlist, egress pinning) still pass

**Plans**: 5/5 plans executed
Plans:
**Wave 1**

- [x] 05-01-PLAN.md — Named source instances end to end: the config map key becomes the kernel's identity across index, sync, grants, API and UI

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 05-02-PLAN.md — Typed match contract on the wire: plugin-declared vocabulary, typed MatchRequest, five plugins migrated

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 05-03-PLAN.md — Per-instance match blocks, participation allowlist, and loud two-phase validation

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 05-04-PLAN.md — Kernel-owned rendition sanitize/wrap/theme; plugins stop owning presentation

**Wave 5** *(blocked on Wave 4 completion)*

- [x] 05-05-PLAN.md — Publish the contract, rewrite config.example.toml, migrate and verify the live config

Notes:

- Deliberately reverses Phase 1 D-02/D-03 (single shared keyword list, flagged costly-to-reverse at the time): the restructure decision of 2026-08-05 accepted that cost now, while five in-repo plugins exist, rather than after external plugins do.
- This is the foundation for the plugin-ecosystem direction (out-of-repo plugins, pull-by-URL distribution) captured in the backlog — the contract must stabilize on instances + typed matching before external authors build against it.
- Research: LOW-MEDIUM. The moving parts are all in-repo (config shape, index schema, proto contract); the design question is the migration path for existing config and index data.

### Phase 6: UI — Scalable Source Surface

**Goal**: The webspace header presents many source instances without duplication — health and filtering combined into one affordance per source — and the accumulated UI polish lands: deep-link fidelity differentiation, search-term highlighting in the detail pane, and themed scrollbars with date markers
**Mode:** mvp
**Depends on**: Phase 5
**Requirements**: UI-07, UI-08, UI-09, UI-11
**Success Criteria** (what must be TRUE):

  1. Each source instance appears exactly once in the header — a single chip/affordance combining health state, filter toggle, and refresh — and the design remains usable at 10+ instances (overflow, grouping, or collapse; no unbounded chip rows)
  2. Items whose "open in source" link can only raise the source app's window are visually differentiated from links that navigate to the item (closes the 04-UAT follow-up)
  3. After an in-webspace search, matched terms are highlighted in the detail pane's rendered content across content variants (text, sanitized HTML, chat transcript); the stream is unaffected since it already filters to matches
  4. Scrollbars app-wide are thin and theme-matched; the stream pane's scrollbar carries date markers reflecting the visible chronology

**Plans**: 7/8 plans executed

Plans:
**Wave 1**

- [x] 06-01-PLAN.md — Search-term highlighting end-to-end: kernel-side `<mark>` tree-walk for the sandboxed rendition iframe plus client-side highlighting for text bodies (UI-09, phase tracer)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 06-02-PLAN.md — Merged per-instance source chip with multi-select filtering and measured overflow into a popover (UI-07)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 06-03-PLAN.md — Deep-link fidelity differentiation and the stream scrollbar's date-marker overlay (UI-08, UI-11)

**Wave 4** *(gap closure — blocked on Wave 2 completion)*

- [x] 06-04-PLAN.md — Keep the chip row's overflow measurement live for the whole session: ref-driven observer attachment covering all four measured elements (UI-07, closes 06-VERIFICATION.md's single gap)

**Wave 5** *(UAT gap closure — 06-UAT.md G-06-1)*

- [x] 06-05-PLAN.md — One match-emphasis vocabulary across every search surface, extended to the item title (UI-09, closes G-06-1)

**Wave 6** *(UAT gap closure — blocked on Wave 5: shared spec/stylesheet files)*

- [x] 06-06-PLAN.md — Restore a visible selected state on the merged source chip: solid accent fill with re-toned label, dot and refresh (UI-07, closes G-06-3)

**Wave 7** *(UAT gap closure — blocked on Wave 6: shared spec/stylesheet files)*

- [x] 06-07-PLAN.md — Rebuild the stream date-marker overlay as a real ruler: own lane, dedicated tokens at 3:1, rail and major/minor hierarchy, scroll and edge gates (UI-11, closes G-06-6)

**Wave 8** *(UAT gap closure — blocked on Waves 6 and 7: shared `SourceChip.svelte`, guard and spec files)*

- [x] 06-08-PLAN.md — Merged source chip as one polished pill: 44px height parity with the overflow trigger, full-surface filter hit area, circular refresh disc, keyboard-scoped reveal (UI-07, closes G-06-3b)

Notes:

- Depends on Phase 5's instance naming: the header redesign is only worth doing against the final source-identity model.
- Fidelity differentiation builds on `LINK_FIDELITY_*` already declared per item (PLUG-03) — this is UI surfacing, not a contract change.
- Highlighting inside sanitized HTML and transcripts must happen after sanitization without weakening it — treat the sanitizer contract as untouchable.
- Research: LOW. All in-repo UI work; the scrollbar date-marker component is the only novel piece.

### Phase 7: Webspace Builder UI

**Goal**: User can configure sources and webspaces from the UI instead of hand-editing TOML — pick plugin types from a list, configure named instances, save a configured set as a webspace, and promote a live search into the webspace's permanent filter
**Mode:** mvp
**Depends on**: Phase 5, Phase 6
**Requirements**: KERN-08, UI-12
**Success Criteria** (what must be TRUE):

  1. User can create or edit a webspace in the UI: choose from available plugin types, configure instances (non-secret fields only — secrets stay environment-only), and save; the kernel loads the result without a restart or with an explicit reload affordance
  2. A live search within a webspace can be saved as a permanent filter on that webspace; the stream then shows only matching items, and further searches refine within that filter
  3. Hand-editing the config file remains fully supported — the UI write path and the file agree on one persisted form, and a hand-edited file is never clobbered blindly
  4. The config write path is the first mutating surface in the kernel API — it is scoped to configuration only, and the plugin contract's read-only guarantee over source data is untouched

**Plans**: 14/16 plans executed

Plans:
**Wave 1**

- [x] 07-01-PLAN.md — Tracer: save a live search as a permanent filter, end to end through the kernel's first mutating route, canonical config rewrite and filtered index read

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 07-02-PLAN.md — Hot apply: plugin host, coordinator and scheduler rebuilt in place on save; explicit Reload for hand-edits; Describe-before-persist for unconfigured plugin types

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 07-03-PLAN.md — Overlay primitives, the header webspace switcher with webspace creation, and a redirect-only root with a first-run empty state

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 07-04-PLAN.md — The "+" source picker: add a configured instance, connect a new plugin type in two steps, the secret env-var field, and the chip edit menu

**Wave 5** *(blocked on Wave 4 completion)*

- [x] 07-05-PLAN.md — "Manage sources…" escape hatch with both destructive confirmations, the shared save/reload state pattern, and the published config-API contract

**Wave 6** *(gap closure, blocked on Wave 5 completion)*

- [x] 07-06-PLAN.md — Gap closure (07-VERIFICATION.md CR-01): one shared instance-id collision guard both AddSourceModal write paths call, so "Save anyway" can never overwrite an existing source instance

**Wave 7** *(gap closure, blocked on Wave 6 completion — both plans run in parallel, zero file overlap)*

- [x] 07-07-PLAN.md — Gap closure (07-VERIFICATION.md gaps[0], 07-REVIEW.md CR-01/IN-01): every /agent/v1 handler reads the live config per request, so a revoked agent grant is enforced on the next request with no kernel restart
- [x] 07-08-PLAN.md — Gap closure (07-VERIFICATION.md gaps[1], 07-REVIEW.md CR-02): closing the source edit modal ends the edit session, so a Cancelled draft can never resurface and be saved over real config

**Wave 8** *(gap closure, blocked on Wave 7 completion)*

- [x] 07-09-PLAN.md — Gap closure (07-VERIFICATION.md gaps[0], 07-REVIEW.md post-07-07/08 CR-01): a save rejected after Host.Reconcile has already committed adopts the new generation through one shared commit site, so host, coordinator, config and scheduler never disagree and a rejected save cannot silently break a source's sync

**Wave 9** *(gap closure, blocked on Wave 8 completion)*

- [x] 07-10-PLAN.md — Gap closure (07-VERIFICATION.md 2026-08-08 gaps[0], 07-REVIEW.md post-07-09 CR-01): the D-07 removed-instance index cleanup runs to completion on every post-Reconcile path, so a save that removes a source and is then rejected for an unrelated typo can never permanently strand that source's items and sync history

**Wave 10** *(UAT gap closure, blocked on Wave 9 completion — both plans run in parallel, zero file overlap)*

- [x] 07-11-PLAN.md — Gap closure (07-UAT.md G-07-3, blocker): an empty webspace shell is a valid config state (D-20), so "+ New webspace" can actually create one — and a shell correlates nothing rather than everything
- [x] 07-13-PLAN.md — Gap closure (07-UAT.md G-07-5): every connection field a plugin fatals without is required and enforced at submit, and a plugin that dies before the go-plugin handshake surfaces its own stderr line instead of four diagnostics that are all false

**Wave 11** *(UAT gap closure, blocked on Wave 10 completion — both plans run in parallel, zero file overlap)*

- [x] 07-12-PLAN.md — Gap closure (07-UAT.md G-07-4): the config API never serializes null for a collection the SPA iterates, and the root route reaches its "No webspaces yet" empty state instead of blaming the kernel for its own exception
- [x] 07-14-PLAN.md — Gap closure (07-UAT.md G-07-6): "Remove from this webspace" writes a document that really narrows participation, and the chip row renders through one shared participation predicate so the chip actually disappears

**Wave 12** *(UAT round-2 gap closure, blocked on Wave 11 completion)*

- [x] 07-15-PLAN.md — Gap closure (07-UAT.md G-07-1): a webspace exists because it is configured, not because it has been synced — one config-aware existence gate for the stream, search and agent surfaces, and a definitive not-found answer that no longer reads as a service outage

**Wave 13** *(UAT round-2 gap closure, blocked on Wave 12 completion — shares the webspace route file with 07-15)*

- [x] 07-16-PLAN.md — Gap closure (07-UAT.md G-07-7): a save that narrows a webspace purges the de-participated join rows before it answers, so a removed source's items leave the stream with its chip — plus a sync-completion stream refetch as the fallback

Notes:

- First config write path from the UI; wants Phase 5's config shape settled first, hence the dependency.
- "Saveable set as a webspace" and "permanent filter" together turn webspaces from static config into something the user grows interactively — the search-promotion flow is the heart of this phase.
- Research: MEDIUM. TOML round-tripping (preserving comments/unknown keys) vs. a UI-owned config layer needs a decision before planning.

### Phase 07.1: Browser E2E Harness (INSERTED)

**Goal:** Phase 7's UAT items are permanent, automated regression armor — a Playwright suite drives a real browser against the shipped kernel binary serving its embedded SPA, hermetically (temp config, ephemeral port, mock-shaped plugins, zero credentials), and GitHub Actions runs it on every push and PR to main
**Requirements**: No REQUIREMENTS.md ID maps to this phase — it is test infrastructure for behaviour Phase 7 already shipped. `07.1-CONTEXT.md`'s locked decisions D-01…D-15 are the authoritative scope source; each plan's `requirements` frontmatter lists the D-IDs it implements.
**Depends on:** Phase 7
**Plans:** 6/6 plans complete

**Success Criteria** (what must be TRUE):

  1. `make e2e` runs a real Chromium against the built `topos serve` binary and its embedded SPA, with no credentials, no live source and no C toolchain
  2. Each spec file gets its own kernel on its own ephemeral port with its own temp `config.toml` and index DB, and can never reach the operator's real config, index or plugin binaries
  3. 07-UAT.md items 1–7 and 10 exist as specs traceable to their item numbers, plus a core-journey smoke set (stream → detail → search → filter)
  4. GitHub Actions runs Go tests, svelte-check, vitest and the e2e suite on push and PR to main, Chromium only, zero retries, traces and screenshots on failure
  5. `scripts/e2e-smoke.sh` and `make smoke` are retired, their portable guards having reappeared hermetically in the suite first

Plans:
**Wave 1**

- [x] 07.1-01-PLAN.md — Tracer: Playwright harness, hermetic kernel fixture, `make e2e` (wave 1)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 07.1-02-PLAN.md — The `mockstrict` e2e-only plugin and its SPA connection-field row (wave 2)
- [x] 07.1-03-PLAN.md — Core-journey smoke set; absorbs the retiring smoke script's portable guards (wave 2)
- [x] 07.1-04-PLAN.md — UAT items 1–4 as specs (wave 2)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 07.1-05-PLAN.md — UAT items 5, 6, 7 and 10 as specs (wave 3)
- [x] 07.1-06-PLAN.md — First CI workflow, retire `make smoke`, record the standing rule (wave 3)

Notes:

- Inserted urgent, before Phase 8: WhatsApp work churns the stream and detail-pane surfaces this suite guards, so the armor has to exist first.
- UAT items 8–9 (SIGKILL timing windows) stay manual and remain an accepted risk — not browser-automatable, unchanged from Phase 7's verdict.
- **Standing rule (D-11):** from Phase 8 onward, any phase that touches the UI extends the Playwright suite as part of its definition of done, and any browser-automatable UAT item becomes a spec rather than staying a manual check. Plan 07.1-06 records this in `docs/testing.md` and `.claude/CLAUDE.md`.

### Phase 8: WhatsApp Conversations (Managed Risk)

**Goal**: User's WhatsApp groups for a topic appear in the webspace stream via a linked-device session, and everything else keeps working when that session breaks
**Mode:** mvp
**Depends on**: Phase 4, Phase 5
**Requirements**: SRC-03
**Success Criteria** (what must be TRUE):

  1. User can link webspaces as a WhatsApp device by scanning a QR code, and the session survives service restarts without re-linking
  2. Messages from WhatsApp groups whose names match the webspace's matching config appear in the stream alongside every other source, using the chat rendering established in Phase 4
  3. The plugin persists its own message store, so conversations captured while it was running stay browsable regardless of what the WhatsApp desktop app retains
  4. De-link, ban, or session expiry surfaces as an explicit plugin-health error in the UI while previously captured messages remain browsable and every other source is unaffected

**Plans:** 7/8 plans executed (G-08-1 closed; gap closure in progress — `08-REVIEW.md` CR-01 open)

Plans:
**Wave 1**

- [x] 08-01-PLAN.md — Tracer: whatsmeow pin gate, the new pure-Go plugin module, CLI QR link, own message store, and a matching group's day digest in the stream; mandatory hands-on spike

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 08-02-PLAN.md — Managed-risk degradation: five named health states, Match-error-never-empty-success, 1:1 matching on saved contact names (D-05/D-06/D-07), read-only + outbound-host AST scans
- [x] 08-03-PLAN.md — In-app pairing, kernel half: UI-SPEC amendment + QR encoder legitimacy audit, machine-readable link mode, and the short-poll link-session endpoint with instance suspension

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 08-04-PLAN.md — In-app pairing, browser half: QRPanel component, inline Add-Source step, chip Re-link… entry, and hermetic Playwright coverage

**Gap closure — Wave 1** *(08-UAT.md G-08-1: a completed real-device pairing left the panel on a stale QR and connected nothing)*

- [x] 08-05-PLAN.md — Contract + browser half: liveness poll decoupled from QR validity, post-pair progress state, amended docs/api.md and 08-UI-SPEC.md

**Gap closure — Wave 2** *(blocked on 08-05)*

- [x] 08-06-PLAN.md — Plugin + kernel half: `pairing_accepted`/`already_linked` wire events, link-subprocess stderr routed into the kernel log, first automated coverage of `execLinkSpawner`

**Gap closure — Wave 3** *(blocked on 08-05, 08-06)*

- [x] 08-07-PLAN.md — Recovery affordance + regression armor: declined-link notice, the qr→poll→paired Playwright case at a realistic expiry, real-device human re-test

**Gap closure — Wave 4** *(08-VERIFICATION.md's promoted 08-REVIEW.md findings: an orphaned link session survives a dialog closed mid-start)*

- [ ] 08-08-PLAN.md — CR-01/WR-01/IN-01: cancel the link session created while the panel was being torn down, armor it structurally and in the browser, clear the stale declined-link notice

Notes:

- Deliberately last. WhatsApp has no official personal-use API and the linked-device route can be de-linked or banned without warning; sequencing it after every other source means v1 is already useful if this plugin has to be dropped or shipped as best-effort.
- The official WhatsApp Desktop app is a thin mirror, not a durable store — this plugin is an active linked-device client with its own event-stream persistence, not a file reader.
- Shifted from Phase 5 to Phase 8 on 2026-08-05; now depends on Phase 4 (chat renderer) and Phase 5 (per-instance matching contract it must implement).
- Phase 07.1's standing rule (D-11) applies here: any UI work in this phase extends the Playwright e2e suite as part of its definition of done — see `docs/testing.md`.
- Research: HIGH — do not plan on assumptions. Spike must answer: linking stability and ban-risk patterns, how much history backfills on first link, event-stream persistence architecture, and de-link/re-link recovery.

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. First Webspace, End to End | 6/6 | Complete    | 2026-07-28 |
| 2. Two Sources, One Trustworthy Stream | 6/6 | Complete    | 2026-07-29 |
| 3. Email in the Webspace | 10/10 | Complete    | 2026-08-02 |
| 4. Signal Conversations | 4/4 | Complete    | 2026-08-04 |
| 5. Source Instances & Per-Type Matching | 5/5 | Complete    | 2026-08-06 |
| 6. UI — Scalable Source Surface | 8/8 | Complete    | 2026-08-07 |
| 7. Webspace Builder UI | 16/16 | Complete    | 2026-08-09 |
| 8. WhatsApp Conversations (Managed Risk) | 7/8 | In Progress|  |

## Requirement Coverage

All 31 v1 requirements are mapped to exactly one phase (8 added in the 2026-08-05 restructure).

| Requirement | Phase |
|-------------|-------|
| KERN-01 | Phase 1 |
| KERN-02 | Phase 1 |
| KERN-03 | Phase 1 |
| KERN-04 | Phase 2 |
| KERN-05 | Phase 3 |
| KERN-06 | Phase 5 |
| KERN-07 | Phase 5 |
| KERN-08 | Phase 7 |
| PLUG-01 | Phase 1 |
| PLUG-02 | Phase 1 |
| PLUG-03 | Phase 1 |
| PLUG-04 | Phase 2 |
| PLUG-05 | Phase 2 |
| SRC-01 | Phase 3 |
| SRC-02 | Phase 4 |
| SRC-03 | Phase 8 |
| SRC-04 | Phase 1 |
| SRC-05 | Phase 2 |
| UI-01 | Phase 1 |
| UI-02 | Phase 2 |
| UI-03 | Phase 1 |
| UI-04 | Phase 1 |
| UI-05 | Phase 2 |
| UI-06 | Phase 2 |
| UI-07 | Phase 6 |
| UI-08 | Phase 6 |
| UI-09 | Phase 6 |
| UI-11 | Phase 6 |
| UI-12 | Phase 7 |
| AGENT-01 | Phase 2 |
| AGENT-02 | Phase 1 |

## Backlog

### Phase 999.1: Plugin ecosystem — out-of-repo plugins, distribution, certification (BACKLOG)

**Goal:** [Captured 2026-08-05 for future planning — likely its own milestone] Plugins move out of the main repo into separate repos: the dozens of plugins needed to make webspaces useful to others (X/Facebook/Bluesky, generic IMAP with Proton/Gmail/Outlook specialisations, GDrive/OneDrive/Dropbox, Discord, Atom feeds, Slack/IRC, and more) would be developed by the people who use those sources and can test them. Needs: a system for pulling plugins into a locally configured instance (by URL configuration first, a marketplace eventually), a detailed plugin development guide plus an agent skill for developing new plugins, and a certification/blessing scheme for external plugins. Depends on Phase 5's contract stabilisation (named instances + per-type matching) — the contract must settle before external authors build against it.
**Requirements:** TBD
**Plans:** 0 plans

Plans:

- [ ] TBD (promote with /gsd-review-backlog when ready)
