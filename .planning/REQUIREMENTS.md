# Requirements: topos — Milestone v1.2.0 Dev/Prod Separation

**Defined:** 2026-08-18
**Core Value:** Open one webspace and instantly see and grok all related information across every silo — without visiting each data store individually.
**Milestone goal:** The operator runs topos daily from installed release artifacts while developing the next milestone from the checkout — the two instances can never clash on port, config, or state.

## v1.2.0 Requirements

Requirements for this milestone. Each maps to roadmap phases.

### Install

- [ ] **INST-01**: User can run `make install [version]` to download that tag's GitHub release artifacts and install the kernel to `$PREFIX/bin` and plugins to `$PREFIX/lib/topos/plugins` (PREFIX configurable, default `/usr/local`)
- [ ] **INST-02**: User can run `make install` with no version argument and get the latest GitHub release
- [ ] **INST-03**: User can start the installed instance as `topos` from PATH; it uses home/XDG config and state locations and discovers plugins from the installed plugins dir
- [ ] **INST-04**: User can run `make install-signal` to build the cgo Signal plugin locally into the installed plugins dir (base install stays toolchain-free: download + copy only)
- [ ] **INST-05**: User can run `make uninstall` to remove installed binaries/plugins; config and state are never touched

### Dev isolation

- [ ] **ISOL-01**: Dev runs from the checkout keep all writable state (config, kernel index, plugin stores) in .gitignored per-checkout or `/tmp` locations — never the home/XDG locations the installed instance uses
- [ ] **ISOL-02**: Dev/test servers bind a non-7777 port; the installed instance keeps 7777
- [ ] **ISOL-03**: The installed instance and a dev instance run simultaneously without clashing on port, config, or state

## Future Requirements

Deferred to future milestones. Tracked but not in current roadmap.

- Kernel OAuth/secrets services for plugins (requirement-grade todo from the gdrive clean-room build)
- Pull-by-URL plugin distribution, dev guide, certification (backlog 999.1)
- `topos-plugins` sibling-repo restructure + trust-model discussion (backlog 999.2)
- IMAP-vanilla refactor with provider extensions
- OneDrive plugin

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| systemd/service management for the installed instance | Manual start from PATH is the chosen run mode this milestone |
| Building install artifacts from source at a tag | Install consumes published GitHub release artifacts; only the Signal plugin builds locally (cgo, excluded from releases by design) |
| Signal plugin in the base `make install` | Keeps base install toolchain-free; `make install-signal` is the explicit opt-in |
| Packaging (AUR, deb, etc.) | `make install` from the checkout is sufficient for a single-operator tool |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| INST-01 | — | Pending |
| INST-02 | — | Pending |
| INST-03 | — | Pending |
| INST-04 | — | Pending |
| INST-05 | — | Pending |
| ISOL-01 | — | Pending |
| ISOL-02 | — | Pending |
| ISOL-03 | — | Pending |

**Coverage:**
- v1.2.0 requirements: 8 total
- Mapped to phases: 0 (roadmap pending)
- Unmapped: 8 ⚠️

---
*Requirements defined: 2026-08-18*
*Last updated: 2026-08-18 after initial definition*
