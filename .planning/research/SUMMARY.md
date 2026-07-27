# Webspaces Research Summary

**Project:** Webspaces (Local-first personal cross-source data aggregation)
**Domain:** Desktop kernel + plugin architecture for unified search/browse across email, chat, documents, notes
**Researched:** 2026-07-27
**Overall Confidence:** MEDIUM-HIGH

## Executive Summary

Webspaces is a well-scoped local-first desktop tool that aggregates heterogeneous personal data sources (email, Signal/WhatsApp, paperless-ngx, SilverBullet) into a deterministic, keyword-driven unified view. The research validates the kernel + plugin architecture as the right approach — three independently-maintained systems (Timelinize, Onyx, Home Assistant) converge on this pattern. **Go is the mandatory choice** for the kernel due to `whatsmeow` being the only mature WhatsApp library; all other stack decisions follow naturally from this anchor.

The hybrid data model (metadata + preview cached locally, full content fetched live on open) is the project's core architectural bet and must be validated early — it trades speed/low-duplication against staleness windows that the UI and backend must handle explicitly. The plugin contract design is the single most critical piece: it must be source-agnostic from day one (sketched against ≥2 structurally different sources before writing code), since third-party extensibility is an explicit goal per PROJECT.md.

**Primary risk:** WhatsApp is the highest-uncertainty plugin — there is no official personal-use API, the official Desktop app doesn't store readable history, and the reverse-engineered workaround (`whatsmeow` linked-device protocol) is subject to ban/de-link without warning. This must be scoped as a known-risk plugin from the start with graceful degradation, not a solved problem. Secondary risks (Signal DB schema churn, Proton Bridge cert/LAN exposure, IMAP read-only safety) are all well-understood and mitigatable with deliberate care.

## Key Findings

### Recommended Stack

**Kernel:** Go 1.23+ is the hard constraint — `whatsmeow` (the only viable WhatsApp library) is Go-only. Go also provides: native cgo access for Signal's SQLCipher, mature IMAP client (`go-imap` v1, stable and battle-tested), first-class concurrency for N long-lived source connections, and single static-binary deployment.

**Storage:** SQLite via `modernc.org/sqlite` (pure Go, no cgo for kernel) with FTS5 for cross-source search. This keeps the kernel's binary a single portable executable — reserve cgo for the Signal plugin only.

**Web UI:** SvelteKit with `@sveltejs/adapter-static` (SPA mode) embedded via `go:embed`. Stream + detail-pane UX (independent scroll, live-updating feed, virtualized lists) is easier to build with client-side state.

**Plugin Architecture:** `hashicorp/go-plugin` (gRPC-over-subprocess) — each plugin is a separate executable. Isolates failures, enables third-party plugins, mirrors Terraform/Vault/Nomad/Home Assistant patterns.

**Core integrations:** Email via `go-imap` v1 (v2 still beta) + Proton Bridge; Signal via `mutecomm/go-sqlcipher/v4` (read-only) + D-Bus keyring; WhatsApp via `go.mau.fi/whatsmeow` (linked-device session); paperless-ngx and SilverBullet via thin REST clients.

### Expected Features

**Launch with (v1):**
- Webspace config (keyword → native category map per source)
- Kernel + plugin architecture with documented contract
- Five plugins (email, Signal, WhatsApp, paperless-ngx, SilverBullet)
- Hybrid data model (index metadata/preview, fetch full content live)
- Web UI: stream + detail pane, source filter, inline preview, deep links

**Add soon after (v1.x):**
- Search within a webspace, sync status indicators, plugin health UI, provenance display

**Defer (v2+):**
- Cross-webspace search, AI correlation, write/reply/edit capability

**Critical constraint:** Deep-link fidelity varies by source — exact links for docs/notes, conversation-only for Signal/WhatsApp, inconsistent for IMAP. Plugin contract must declare fidelity per item.

### Architecture Approach

**Pattern:** Kernel mediates between config-driven correlation engine and process-isolated plugins. Kernel owns: (1) correlating items to webspaces (keyword config matched at sync time, persisted in index), (2) per-plugin sync scheduling (coordinator pattern from Home Assistant), (3) read-only HTTP API. Plugins never write index directly.

**Data flow:**
- Sync: Scheduler → Plugin.enumerate() → Correlation Engine → Index Store
- Browse: UI → Kernel HTTP API → Index Store (fast, precomputed)
- Open detail: UI → Kernel → Plugin.fetchFull() → source (expensive, on-demand)

**Key patterns:**
1. **Hybrid local index + live fetch:** Metadata/preview cached, full content live. Opposite of Timelinize (which duplicates everything).
2. **Config-declared correlation at sync time:** Deterministic, no AI in v1. Avoids query-time bottleneck.
3. **Coordinator per plugin:** Dedupes concurrent refreshes, single source of truth for health tracking.

**Critical build order (dependency-driven):**
1. Normalized Item schema + Index Store
2. Plugin Contract + Plugin Host skeleton (proven with mock plugin)
3. Webspace Config + Correlation Engine
4. Sync Scheduler (full flow end-to-end)
5. HTTP API + minimal UI
6. First real plugin: paperless-ngx or SilverBullet (low-risk REST)
7. Email (IMAP) plugin
8. Signal plugin (highest-risk: undocumented local DB)
9. WhatsApp plugin (same risk class as Signal)
10. Detail pane + deep-link wiring

This front-loads kernel/contract validation behind mock, so real plugin risk doesn't corrupt core validation.

### Critical Pitfalls (Top 5)

1. **WhatsApp file-reader misconception (P1):** Official Desktop app is a thin mirror, not a durable store. **Avoid:** Treat plugin as active linked-device client running `whatsmeow`, persisting its own event stream. Consequence: plugin must support long-running session lifecycle.

2. **WhatsApp account ban / device de-link (P2):** Reverse-engineered protocol violates ToS; device de-linking or account suspension reported in community (no official stats). **Avoid:** Accept as managed risk; use well-maintained library (whatsmeow actively updated mid-2026); isolate plugin so ban degrades gracefully; avoid bulk backfill scraping.

3. **Signal key extraction method changes (P3):** Since 2024, Signal wraps key via Electron's `safeStorage` (GNOME Keyring/KWallet on Linux). Outdated plaintext-key guides fail on current installs. **Avoid:** Detect active `safeStorageBackend` in config.json; branch extraction by keyring type; fail with specific error; test against user's actual OS/DE.

4. **Signal DB schema churn (P4):** Schema changes substantially across app updates with no stability guarantee; parsers fail silently on unknown versions. **Avoid:** Read DB schema version before parsing; fail loudly on unknown versions; keep parsing isolated so branches are additive.

5. **Unsafe concurrent DB access (P5):** Signal/WhatsApp hold databases open while running; writing or triggering checkpoint risks corruption. SQLite itself had WAL-reset corruption bug (3.7.0–3.51.2, fixed March 2026). **Avoid:** Open read-only (SQLite URI `mode=ro`); pin SQLite ≥3.51.3; never VACUUM/checkpoint; use SQLite backup API for copy-then-read.

## Implications for Roadmap

### Suggested Phase Structure

**Phase 1: Kernel Core + Schema + Plugin Architecture**
- Normalized Item schema, Plugin Contract (`.proto` for third-party reference), Plugin Host, Webspace Config loader, Correlation Engine, Sync Scheduler, HTTP API skeleton
- **Rationale:** Dependency-driven — everything follows. De-risks architecture before real plugin complexity.
- **Pitfalls avoided:** P10 (plugin API over-fit to email) — sketch contract against ≥2 sources before writing code; P11 (scope creep) — enforce read-only structurally at contract level
- **Research needed:** MINIMAL — patterns established (Timelinize, Home Assistant, Onyx)

**Phase 2: Mock Plugin + Full Kernel Validation**
- Mock/fixture plugin, minimal UI (stream only), full sync cycle, sync status indicators
- **Rationale:** Prove kernel end-to-end before real plugins introduce complexity
- **Pitfalls avoided:** P9 (staleness handling) — surface explicitly in UI; P8 (Seen-flag mutation) — will be per-plugin in Phase 4+
- **Research needed:** MINIMAL — sync status patterns well-documented

**Phase 3: First Real Plugin (paperless-ngx or SilverBullet)**
- Low-risk REST API, validate live-fetch + deep-link mechanics
- **Rationale:** Lowest-risk well-documented source before Signal/WhatsApp complexity
- **Research needed:** LOW — both APIs well-documented; validate exact deep-link URLs

**Phase 4: Email (IMAP) Plugin**
- Folder/label categorization, Proton Bridge integration, label de-duplication
- **Rationale:** More complex than Phase 3, far safer than Signal/WhatsApp
- **Pitfalls avoided:** P6 (Bridge LAN exposure) — tunnel or pinned cert required; P7 (label duplication) — dedup by Message-ID; P8 (Seen-flag mutation) — use BODY.PEEK, add automated test
- **Research needed:** MEDIUM — Proton Bridge cert pinning in Go IMAP; Proton webmail deep-link format (not verified in original research); recommend 1-day spike

**Phase 5: Signal Plugin**
- SQLCipher read-only access, keyring backend extraction, schema-version detection
- **Rationale:** Highest-risk plugin; benefits from earlier patterns; requires dedicated research
- **Pitfalls avoided:** P3 (keyring churn) — detect backend and branch; P4 (schema churn) — check version, fail loudly; P5 (unsafe access) — explicit mode=ro, verify SQLite ≥3.51.3
- **Research needed:** MEDIUM-HIGH — Require 2-3 day spike: DB schema + keyring backend, hands-on testing on user's Arch setup, sqlcipher version stability

**Phase 6: WhatsApp Plugin**
- Whatsmeow linked-device session, graceful error handling for ban/de-link
- **Rationale:** Highest-uncertainty; benefits from Signal patterns; explicit managed-risk plugin
- **Pitfalls avoided:** P1 (file-reader misconception) — active linked-device, not file reader; P2 (ban/de-link) — documented risk, graceful degradation
- **Research needed:** HIGH — **This is highest-risk area.** Require 3-5 day spike: whatsmeow linking stability, ban-risk patterns, message backfill behavior on first link, event-stream persistence architecture, de-link/re-link recovery. **Do not proceed without answers.**

**Phase 7: Detail Pane + Cross-Source Renderers + Deep-Link Wiring**
- Source-specific renderers, deep-link affordances, provenance display
- **Rationale:** UI layer on top of proven plugins
- **Research needed:** MINIMAL — pattern straightforward once plugins work

**Phase 8: Search + Cross-Webspace Features (v1.x/v2)**
- Full-text search within webspace, cross-webspace search
- **Rationale:** Defer to v1.x — valuable once items accumulate
- **Research needed:** MINIMAL — FTS5 patterns canonical

### Phase Ordering Rationale

1. Kernel first (Phase 1): Every subsequent phase depends on stable contract
2. Mock validation (Phase 2): Proves plumbing before external systems
3. Low-risk REST plugin first (Phase 3): Validates patterns with well-documented API
4. Email (Phase 4): Introduces IMAP complexity, far safer than Signal/WhatsApp
5. Signal (Phase 5): Requires research; benefits from earlier patterns
6. WhatsApp (Phase 6): Highest-uncertainty; requires dedicated spike
7. UI polish (Phase 7): Layered on proven plugins
8. Search (Phase 8): Natural v1.x extension

This minimizes architectural thrash, isolates integration risk, builds confidence incrementally.

### Research Flags

**MUST HAVE spike phases before planning:**
- **Phase 4 (Email/IMAP):** Proton Bridge networking (tunnel vs rebind, cert pinning in Go IMAP). Consider 1-day spike.
- **Phase 5 (Signal):** Signal Desktop DB schema + keyring extraction. Dedicate 2-3 days including hands-on testing on user's Arch setup, schema-version detection, sqlcipher stability.
- **Phase 6 (WhatsApp):** **HIGHEST-RISK AREA.** Require 3-5 day spike: whatsmeow linking stability/ban-risk, message backfill behavior, event-stream persistence, de-link recovery. **Do not proceed on assumptions.**

**Standard patterns (skip research-phase):**
- **Phase 1 (Kernel):** Plugin patterns established (Terraform, Home Assistant)
- **Phase 2 (Mock):** Standard SPA/API patterns
- **Phase 3 (First REST):** APIs well-documented
- **Phase 7 (Detail Pane):** Standard web UI components
- **Phase 8 (Search):** FTS5 patterns canonical

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| **Stack** | HIGH | Go hard constraint validated; SQLite established; go-imap v1 stable; plugin architecture mirrors proven systems |
| **Features** | MEDIUM | MVP clear from PROJECT.md; competitive features validated; deep-link fidelity per source confirmed; user priorities may shift with prototyping |
| **Architecture** | MEDIUM-HIGH | Kernel + coordinator pattern verified across three independent systems (HIGH); hybrid model validated (HIGH); plugin contract design (HIGH confidence in pattern, MEDIUM in exact interface shape) |
| **Pitfalls** | MEDIUM-HIGH | Signal/Proton/IMAP pitfalls (HIGH — official docs); WhatsApp pitfalls (MEDIUM — community consensus, no official Meta data); performance traps (MEDIUM-HIGH — cross-checked) |

**Overall:** MEDIUM-HIGH. Architecture and stack validated; highest uncertainty is WhatsApp (spike needed before Phase 6) and Signal/Proton specifics (spike needed before Phase 4/5). Core patterns proven; execution risk localized to plugin integration.

### Gaps to Address

1. **WhatsApp local store access — HIGH priority:** No official API. Spike must answer: (1) Can whatsmeow link without bans? (2) History backfill quantity? (3) Linked-device session architecture? (4) Recovery strategy? **Do not proceed to Phase 6 without answers.**

2. **Signal Desktop keyring backend — MEDIUM priority:** Keyring churn documented; must test against user's actual Arch setup. Spike should verify schema-version detection and extraction before Phase 5.

3. **Proton Bridge networking — MEDIUM priority:** Go IMAP client cert-pinning mechanism. Spike should confirm tunnel or cert-pinning/firewall is feasible before Phase 4.

4. **Exact deep-link formats — MINOR priority:** Paperless `/documents/<id>/details` vs `/documents/<id>`, Proton Mail webmail format (not verified). Confirm during Phase 3/4 planning.

5. **Hybrid index staleness handling — MEDIUM priority:** Phase 2 should prototype explicit "unavailable at source" UI states so Phase 3+ can verify error path.

## Sources

### Primary (HIGH confidence)
- GitHub: timelinize/timelinize, Timelinize Wiki — Architecture pattern, Item/Entity schema, data-source contract
- GitHub: onyx-dot-app/onyx — Connector load/poll patterns, sync state machine
- Home Assistant Developer Docs: DataUpdateCoordinator — Coordinator pattern
- GitHub: hashicorp/go-plugin — gRPC plugin lifecycle (used by Terraform/Vault/Nomad)
- pkg.go.dev: go.mau.fi/whatsmeow — WhatsApp library, 300+ imports, actively maintained
- GitHub: emersion/go-imap v1 — IMAP client, production-proven, v2 beta confirmed
- SQLite official docs: FTS5 — Full-text search pattern
- Paperless-ngx official docs: REST API — API contract, endpoints
- SilverBullet official docs: HTTP API — Page serving, link handling

### Secondary (MEDIUM confidence)
- GitHub: flathub/org.signal.Signal issues #753/#754 — Signal keyring backend migration
- Migrating Signal Desktop keyring backend — Inane Observations blog — Keyring extraction behavior
- Timelinize Wiki: Data Sources & Develop — Component boundaries, plugin contract
- Proton: Labels in Bridge — Official label duplication documentation
- SQLite: How To Corrupt Database File — Official WAL corruption warnings
- GitHub: wacli — Linked-device architecture reference

### Tertiary (LOW confidence)
- Community blog posts on Proton Bridge LAN rebinding (multiple sources, unofficial)
- WhatsApp ban statistics (Wapisimo blog) — Anecdotal, no official Meta data
- Unified-inbox architecture comparisons (community forums)

---

**Research completed:** 2026-07-27
**Ready for roadmap planning:** Yes (pending spike phases for WhatsApp, Signal, Proton Bridge)

**Next steps for roadmap:**
1. Initiate spike/research phase for WhatsApp before Phase 6 planning (3-5 days)
2. Initiate spike/research phase for Signal keyring/DB before Phase 5 planning (2-3 days)
3. Initiate spike/research phase for Proton Bridge before Phase 4 planning (1 day)
4. Use phase suggestions (1-8 above) as roadmap starting point
5. Run `/gsd-plan-phase` for each suggested phase
