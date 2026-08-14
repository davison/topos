# Requirements: topos

**Defined:** 2026-08-12
**Milestone:** v1.1.0 Plugin Ecosystem
**Core Value:** Open one webspace and instantly see and grok all related information across every silo — without visiting each data store individually.

## v1.1.0 Requirements

Requirements for this milestone. Each maps to roadmap phases. REQ-ID numbering continues from v1.0 (archived in `milestones/v1.0-REQUIREMENTS.md`).

### Plugin Ecosystem

- [x] **PLUG-06**: Kernel discovers and launches plugin binaries from a configured external plugins directory, distinct from the trusted in-repo plugins directory
- [x] **PLUG-07**: Kernel derives a plugin's trusted/untrusted status from provenance (directory tier, content hash pinned at add-time and re-verified at every launch) — never from anything the plugin declares about itself
- [x] **PLUG-08**: User sees an explicit warning when adding a source from an untrusted plugin, and a persistent untrusted badge wherever that source appears (picker, source chips)
- [x] **PLUG-09**: Plugin host passes arbitrary per-instance config keys through to plugins (generic extra-fields map), so external plugins can receive provider-specific settings without kernel changes

### New Sources

- [x] **SRC-04**: User can add a local or network filesystem folder (optional subfolders) as a source; documents appear in webspace streams with previews and deep links, synced via stat-diff polling that works on network mounts
- [ ] **SRC-05**: User can add a Google Drive folder as a source using their own Google OAuth client (bring-your-own credentials via env refs), with incremental sync, Workspace-doc export previews, and deep links to the Drive web UI
- [ ] **SRC-06**: The Google Drive plugin is developed out-of-repo against the published contract and installed through the external-plugin path — proving a third party can ship a working untrusted plugin end to end

### Per-Item Curation

- [ ] **KERN-09**: User can exclude an individual stream item from a webspace; the exclusion survives re-syncs and index rebuilds and always outranks automatic match rules
- [ ] **KERN-10**: User can view a webspace's excluded items and un-exclude them (mark reversal; pulling never-matched items in via a Browse RPC is explicitly out of scope)

### Web App

- [ ] **UI-13**: App installs as a PWA on desktop (manifest, ServiceWorker, icons) with an update flow that never serves a stale UI against an upgraded kernel
- [ ] **UI-14**: Mobile/LAN install limitation (secure-context requirement) is documented, with the recommended user-provided HTTPS workarounds

## Future Requirements

Deferred to a later milestone. Tracked but not in the current roadmap.

### Plugin Ecosystem

- **PLUG-10**: Pull-by-URL plugin distribution via the source picker's install flow
- **PLUG-11**: Third-party plugin developer guide and certification/trust-promotion path

### New Sources

- **SRC-07**: OneDrive folder source (Microsoft Graph; same shape as Google Drive)
- **SRC-08**: IMAP-vanilla plugin refactor with Proton/GMail/O365 provider extensions

### Per-Item Curation

- **KERN-11**: Include never-matched items into a webspace via a contract-level Browse/List RPC

### Web App

- **UI-15**: Kernel HTTPS listen mode enabling PWA install from LAN/mobile devices

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Plugin sandboxing/containment | go-plugin is a transport, not a sandbox; v1.0 deliberately rejected WASM for capability reasons. Trust marking is honest labeling, documented as such |
| Embedded shared Google OAuth client | "Testing" status expires refresh tokens every 7 days; publishing requires Google verification. Bring-your-own matches the project's env-ref secrets pattern |
| Full offline PWA caching | The backend IS local; caching API responses adds stale-data risk for zero benefit. SW scope is install + UI-shell freshness |
| Plugin marketplace/registry | Single-maintainer project; directory-based discovery suffices until real third-party demand exists |
| Write-back to any source | Read-only by design (v1.0 constraint, unchanged) |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| PLUG-06 | Phase 11 | Complete |
| PLUG-07 | Phase 11 | Complete |
| PLUG-08 | Phase 11 | Complete |
| PLUG-09 | Phase 11 | Complete |
| SRC-04 | Phase 12 | Complete |
| SRC-05 | Phase 14 | Pending |
| SRC-06 | Phase 14 | Pending |
| KERN-09 | Phase 13 | Pending |
| KERN-10 | Phase 13 | Pending |
| UI-13 | Phase 13 | Gaps Found |
| UI-14 | Phase 13 | Gaps Found |

**Coverage:**

- v1.1.0 requirements: 11 total
- Mapped to phases: 11
- Unmapped: 0 ✓

---
*Requirements defined: 2026-08-12*
*Last updated: 2026-08-12 after roadmap creation (Phases 11-14)*
