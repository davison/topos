# Requirements: topos — v1.3.0 Plugin Repo Split

**Defined:** 2026-08-19
**Core Value:** Open one webspace and instantly see and grok all related information across every silo — without visiting each data store individually.

## v1.3.0 Requirements

Requirements for this milestone. Each maps to roadmap phases.

### Repo Restructure (REPO)

- [ ] **REPO-01**: All six functional plugins (paperless, silverbullet, proton, whatsapp, filesystem, signal) build and run from the `topos-plugins` sibling repo as subdirectories
- [ ] **REPO-02**: The gdrive plugin is folded into `topos-plugins` from `topos-plugin-gdrive`, superseding that repo
- [ ] **REPO-03**: The kernel repo retains only mock/mockstrict plugins, and kernel CI (Go tests, e2e suite, dev-check gates) passes with no functional plugin present
- [ ] **REPO-04**: The kernel repo carries no per-plugin knowledge — Makefile targets, build manifests, and layout assumptions about functional plugins are gone
- [ ] **REPO-05**: Developer can run the dev loop with kernel and topos-plugins checkouts side by side, with locally built plugins loading into the dev kernel

### First-Party Trust (TRUST)

- [x] **TRUST-01**: Plugin trust derives from verifiable provenance (signed/attested release artifacts), not from which directory a binary resolves from
- [x] **TRUST-02**: A topos-plugins release binary verifies as first-party trusted on an installed instance, despite living outside the kernel repo and its link-time manifest
- [x] **TRUST-03**: An unsigned external binary still installs through the existing consent-and-pin flow, marked untrusted (fallback unchanged)
- [x] **TRUST-04**: Trust cannot be escalated via config edit, file drop, or binary shadowing — the "trust tier is directory-location" todo's attack paths are closed by committed tests

### Independent Distribution (DIST)

- [ ] **DIST-01**: topos-plugins has its own CI/release pipeline publishing versioned, checksummed, provenance-carrying artifacts
- [ ] **DIST-02**: User can install and update plugins independently of the kernel (and vice versa) — a `make install`-grade flow for topos-plugins artifacts
- [ ] **DIST-03**: Kernel↔plugin incompatibility surfaces loudly by name (contract/handshake version), never as silent source absence
- [ ] **DIST-04**: The Signal plugin's local cgo build path survives the move — buildable from the topos-plugins checkout into the external tier via consent-and-pin, as today

### Pull-by-URL Install (PULL)

- [ ] **PULL-01**: User can install a plugin from a URL with one CLI command — download, verify, and land in the trust tier its provenance earns
- [ ] **PULL-02**: A failed verification aborts loudly and leaves nothing placed

### Developer Guide (DOCS)

- [ ] **DOCS-01**: A plugin development guide walks a third-party author from published contract + mocks to a working, installable out-of-repo plugin

## Future Requirements

Deferred to later milestones. Tracked but not in current roadmap.

### Plugin Ecosystem (999.1 remainder)

- **PLUG-CERT**: Certification/blessing scheme that can promote an external plugin out of untrusted
- **PULL-UI**: In-app install-from-URL flow in the source picker (kernel downloads, verifies, and lands a plugin from the running instance)
- **PLUG-MKT**: Plugin discovery/marketplace surface

### Kernel Services

- **KERN-OAUTH**: OAuth connectivity and secrets management abstracted into the kernel for all plugins (requirement-grade todo from the gdrive clean-room build)

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Certification/blessing scheme | 999.1 remainder — trust promotion path deferred until first-party trust mechanism has settled |
| In-app install-from-URL UI | CLI-first decided for this milestone; UI flow layers on later |
| Marketplace/discovery | Needs distribution + certification groundwork first |
| Kernel OAuth/secrets services | Separate requirement-grade candidate; orthogonal to the repo split |
| GAP-06 (fully-`extras` source config exemption) | Rides along only if the gdrive fold-in forces it; not a milestone commitment |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| REPO-01 | Phase 17 | Pending |
| REPO-02 | Phase 17 | Pending |
| REPO-03 | Phase 17 | Pending |
| REPO-04 | Phase 17 | Pending |
| REPO-05 | Phase 17 | Pending |
| TRUST-01 | Phase 16 | Complete |
| TRUST-02 | Phase 16 | Complete |
| TRUST-03 | Phase 16 | Complete |
| TRUST-04 | Phase 16 | Complete |
| DIST-01 | Phase 17 | Pending |
| DIST-02 | Phase 17 | Pending |
| DIST-03 | Phase 17 | Pending |
| DIST-04 | Phase 17 | Pending |
| PULL-01 | Phase 18 | Pending |
| PULL-02 | Phase 18 | Pending |
| DOCS-01 | Phase 18 | Pending |

**Coverage:**
- v1.3.0 requirements: 16 total
- Mapped to phases: 16 ✓
- Unmapped: 0

**By phase:**

| Phase | Requirements | Count |
|-------|--------------|-------|
| 16. Provenance-Based Plugin Trust | TRUST-01, TRUST-02, TRUST-03, TRUST-04 | 4 |
| 17. The Plugin Repo Split | REPO-01..05, DIST-01, DIST-02, DIST-03, DIST-04 | 9 |
| 18. Pull-by-URL Install & the Plugin Developer Guide | PULL-01, PULL-02, DOCS-01 | 3 |

---
*Requirements defined: 2026-08-19*
*Last updated: 2026-08-19 after roadmap creation (16/16 mapped to Phases 16–18)*
