# Roadmap: Webspaces

## Overview

Webspaces is built as a series of vertical slices: every phase ends with the user able to open a browser, pick a webspace, and see more of their life in it than they could before. Phase 1 delivers the whole spine — config, index, plugin contract, HTTP API, stream, detail pane — proven by one real low-risk source (paperless-ngx) rather than a mock. Phase 2 adds a structurally different second source (SilverBullet), which is what actually proves the plugin contract is source-agnostic, plus the operability surface (filter, health, refresh, staleness, agent permission gating) that makes a multi-source view trustworthy. Phases 3–5 then add one source per phase in strict ascending order of integration risk: IMAP email, Signal, and finally WhatsApp. The risky sources come last on purpose — by the time WhatsApp's linked-device session is attempted, the system is already useful and complete without it, so a ban or de-link degrades one plugin rather than sinking the milestone.

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: First Webspace, End to End** - Kernel spine, plugin contract, and web UI proven by real paperless-ngx documents in a browsable stream
- [ ] **Phase 2: Two Sources, One Trustworthy Stream** - SilverBullet joins the stream; filter, health, refresh, staleness states, and default-deny agent permissions
- [ ] **Phase 3: Email in the Webspace** - Proton/IMAP mail appears in the stream, never marked read, with full-text search across the webspace
- [ ] **Phase 4: Signal Conversations** - Signal Desktop history in the stream, read strictly read-only with keyring and schema-version safety
- [ ] **Phase 5: WhatsApp Conversations (Managed Risk)** - Linked-device WhatsApp history in the stream, degrading gracefully on de-link or ban

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

**Plans**: 4 plans (3 waves)
Plans:
**Wave 1**

- [x] 01-01-PLAN.md — Walking skeleton tracer: config → plugin subprocess → SQLite index → HTTP API → browser, against real paperless-ngx

**Wave 2** *(blocked on Wave 1 completion)*

- [ ] 01-02-PLAN.md — Detail pane: live content fetch through the plugin, hardened content routes, open-in-source with declared fidelity

**Wave 3** *(blocked on Wave 2 completion)*

- [ ] 01-03-PLAN.md — Stream presentation and every UI state, including a failed sync never masquerading as an empty webspace
- [ ] 01-04-PLAN.md — Published plugin + HTTP contracts, contract-conformance tests enforcing read-only and the agent envelope, operator docs

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

**Plans**: TBD
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

**Plans**: TBD

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

**Plans**: TBD

Notes:

- Single requirement, but a full vertical slice with a mandatory spike — kept separate from Phase 5 so chat access is proven on the tractable source before the unpredictable one.
- Introduces the chat-thread renderer that Phase 5 reuses; extends the existing UI contract rather than defining a new one.
- Research: MEDIUM-HIGH. Spike before planning: Signal Desktop DB schema, `safeStorage` keyring backend extraction tested hands-on against the user's actual Arch/DE setup, schema-version detection, and SQLCipher/SQLite version stability (pin SQLite ≥ 3.51.3; never VACUUM or checkpoint).

### Phase 5: WhatsApp Conversations (Managed Risk)

**Goal**: User's WhatsApp groups for a topic appear in the webspace stream via a linked-device session, and everything else keeps working when that session breaks
**Mode:** mvp
**Depends on**: Phase 4
**Requirements**: SRC-03
**Success Criteria** (what must be TRUE):

  1. User can link webspaces as a WhatsApp device by scanning a QR code, and the session survives service restarts without re-linking
  2. Messages from WhatsApp groups whose names match the webspace keyword appear in the stream alongside every other source, using the chat rendering established in Phase 4
  3. The plugin persists its own message store, so conversations captured while it was running stay browsable regardless of what the WhatsApp desktop app retains
  4. De-link, ban, or session expiry surfaces as an explicit plugin-health error in the UI while previously captured messages remain browsable and every other source is unaffected

**Plans**: TBD

Notes:

- Deliberately last. WhatsApp has no official personal-use API and the linked-device route can be de-linked or banned without warning; sequencing it after every other source means v1 is already useful if this plugin has to be dropped or shipped as best-effort.
- The official WhatsApp Desktop app is a thin mirror, not a durable store — this plugin is an active linked-device client with its own event-stream persistence, not a file reader.
- Research: HIGH — do not plan on assumptions. Spike must answer: linking stability and ban-risk patterns, how much history backfills on first link, event-stream persistence architecture, and de-link/re-link recovery.

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4 → 5

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. First Webspace, End to End | 1/4 | In Progress|  |
| 2. Two Sources, One Trustworthy Stream | 0/TBD | Not started | - |
| 3. Email in the Webspace | 0/TBD | Not started | - |
| 4. Signal Conversations | 0/TBD | Not started | - |
| 5. WhatsApp Conversations (Managed Risk) | 0/TBD | Not started | - |

## Requirement Coverage

All 23 v1 requirements are mapped to exactly one phase.

| Requirement | Phase |
|-------------|-------|
| KERN-01 | Phase 1 |
| KERN-02 | Phase 1 |
| KERN-03 | Phase 1 |
| KERN-04 | Phase 2 |
| KERN-05 | Phase 3 |
| PLUG-01 | Phase 1 |
| PLUG-02 | Phase 1 |
| PLUG-03 | Phase 1 |
| PLUG-04 | Phase 2 |
| PLUG-05 | Phase 2 |
| SRC-01 | Phase 3 |
| SRC-02 | Phase 4 |
| SRC-03 | Phase 5 |
| SRC-04 | Phase 1 |
| SRC-05 | Phase 2 |
| UI-01 | Phase 1 |
| UI-02 | Phase 2 |
| UI-03 | Phase 1 |
| UI-04 | Phase 1 |
| UI-05 | Phase 2 |
| UI-06 | Phase 2 |
| AGENT-01 | Phase 2 |
| AGENT-02 | Phase 1 |
