# Roadmap: topos

## Milestones

- ✅ **v1.0 MVP** — Phases 1–10 incl. 07.1, 09.1 (shipped 2026-08-12) — [details](milestones/v1.0-ROADMAP.md)
- ✅ **v1.1.0 Plugin Ecosystem** — Phases 11–14 (shipped 2026-08-18) — [details](milestones/v1.1.0-ROADMAP.md)
- 🚧 **v1.2.0 Dev/Prod Separation** — Phase 15 (started 2026-08-18)

## Milestone v1.2.0: Dev/Prod Separation

**Goal:** The operator runs topos daily from installed release artifacts while
developing the next milestone from the checkout — the two instances can never
clash on port, config, or state.

**Shape:** One phase, at the user's explicit direction. Install and isolation
are not separable deliverables here: an installed instance is only *proven*
safe at the moment a dev instance runs beside it without either noticing. Coarse
granularity agrees — splitting install from isolation would produce a phase
whose success criteria could not be observed until the next phase landed.

## Phases

- [ ] **Phase 15: Installed Instance & Dev Isolation** - `make install` / `install-signal` / `uninstall` from published release artifacts, plus full dev-side port, config, and state isolation from the checkout

<details>
<summary>✅ v1.0 MVP (Phases 1–10) — SHIPPED 2026-08-12</summary>

- [x] Phase 1: First Webspace, End to End (6/6 plans) — completed 2026-07-28
- [x] Phase 2: Two Sources, One Trustworthy Stream (6/6 plans) — completed 2026-07-29
- [x] Phase 3: Email in the Webspace (10/10 plans) — completed 2026-08-02
- [x] Phase 4: Signal Conversations (4/4 plans) — completed 2026-08-04
- [x] Phase 5: Source Instances & Per-Type Matching (5/5 plans) — completed 2026-08-06
- [x] Phase 6: UI — Scalable Source Surface (8/8 plans) — completed 2026-08-07
- [x] Phase 7: Webspace Builder UI (16/16 plans) — completed 2026-08-09
- [x] Phase 07.1: Browser E2E Harness (6/6 plans, INSERTED) — completed 2026-08-10
- [x] Phase 8: WhatsApp Conversations (Managed Risk) (15/15 plans) — completed 2026-08-11
- [x] Phase 9: UI Polish and Source Management Rework (7/7 plans) — completed 2026-08-11
- [x] Phase 09.1: 1.0 Release UX Essentials — mobile layout + first-run bootstrap (4/4 plans, INSERTED) — completed 2026-08-12
- [x] Phase 10: Docs and Release Readiness (5/5 plans) — completed 2026-08-12

Full phase details, success criteria, and plan lists: [milestones/v1.0-ROADMAP.md](milestones/v1.0-ROADMAP.md)

</details>

<details>
<summary>✅ v1.1.0 Plugin Ecosystem (Phases 11–14) — SHIPPED 2026-08-18</summary>

- [x] Phase 11: External Plugins & the Trust Boundary (7/7 plans) — completed 2026-08-13
- [x] Phase 12: Filesystem Source (11/11 plans) — completed 2026-08-14
- [x] Phase 13: Per-Item Curation & Installable App (8/8 plans) — completed 2026-08-15
- [x] Phase 14: Google Drive Source, Built Out-of-Repo (6/6 plans incl. gap closure) — completed 2026-08-18

Full phase details, success criteria, and plan lists: [milestones/v1.1.0-ROADMAP.md](milestones/v1.1.0-ROADMAP.md)

</details>

## Phase Details

### Phase 15: Installed Instance & Dev Isolation

**Goal**: The operator installs topos from published GitHub release artifacts and runs it daily from PATH, while the checkout's dev loop runs alongside it on its own port, config, and state — neither instance can see or disturb the other's data.
**Depends on**: Nothing new (builds directly on v1.1.0's 14-01 `--config`/`TOPOS_CONFIG` precursor and Phase 10's tag-triggered release artifacts)
**Requirements**: INST-01, INST-02, INST-03, INST-04, INST-05, ISOL-01, ISOL-02, ISOL-03
**Success Criteria** (what must be TRUE):

  1. `make install` with no argument installs the **latest** GitHub release's published artifacts, and `make install <version>` installs that tag's — kernel to `$PREFIX/bin`, plugins to `$PREFIX/lib/topos/plugins`, with `PREFIX` configurable and defaulting to `/usr/local`. Install is download-and-copy only: it completes on a machine with no Go toolchain. (INST-01, INST-02)
  2. The operator starts the installed instance by typing `topos` from PATH: it serves on port 7777, reads config and writes state in the existing home/XDG locations unchanged, and discovers its plugins from the installed plugins directory — the operator's live instance migrates onto installed artifacts without touching their config or index. (INST-03)
  3. `make install-signal` is an explicit opt-in that builds the cgo Signal plugin locally into the installed *external* plugins directory, and the installed instance picks it up via the one-time consent-and-pin flow — the base install path stays toolchain-free because the Signal binary is deliberately excluded from published releases. Trusted-dir placement is refused by Phase 13's build-manifest verification; the external-tier path is the operator-confirmed resolution (2026-08-18). (INST-04)
  4. `make uninstall` removes exactly what was installed — prefix binaries and plugins — and leaves the operator's config, kernel index, and plugin stores completely untouched, verifiably so. (INST-05)
  5. The installed instance and a dev run from the checkout run **simultaneously** with neither noticing the other: the dev run binds a non-7777 port and keeps every writable artifact (dev config, kernel index, all plugin stores including a separate WhatsApp link for real-source dev runs) in .gitignored per-checkout or `/tmp` paths, so a dev run or a test run never reads or writes the home/XDG locations the installed instance owns. (ISOL-01, ISOL-02, ISOL-03)

**Plans**: 3/5 plans executed

Plans:
**Wave 1**

- [x] 15-01-PLAN.md — Installed layout end to end: `make install` for an explicit tag, checksum-verified, and the kernel's installed-layout plugin resolution (tracer)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 15-02-PLAN.md — Latest-release resolution, `make uninstall`, and the filesystem plugin added to the published asset set

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 15-03-PLAN.md — `make install-signal` / `make uninstall-signal`, and a behavioural proof that the base install needs no toolchain

**Wave 4** *(blocked on Wave 3 completion)*

- [ ] 15-04-PLAN.md — `cmd/topos-devguard` isolation refusal, the dev loop's move to port 7778, and the per-checkout plugin-store convention

**Wave 5** *(blocked on Wave 4 completion)*

- [ ] 15-05-PLAN.md — The committed side-by-side simultaneity proof, the gate documentation, and the migration runbook

**Notes**:

- Install consumes *published release artifacts*, never a from-source build at a tag (explicit Out of Scope in REQUIREMENTS.md). The Signal plugin is the single exception, and only via `make install-signal`.
- No service management this milestone — run mode is manual start from PATH.
- Criterion 5 closes the pending todo "Dev/test servers must not use the production port 7777" (2026-08-05, minor/kernel), whose natural home the 14-01 config split already established.
- Expected to be Makefile and kernel-path work rather than UI work, so the standing "UI phases extend the Playwright e2e suite" rule (07.1 D-11) does not bind here — but the isolation guarantees in criteria 4 and 5 are exactly the kind that decay silently, so pin them with committed tests wherever practical (e.g. an install/uninstall smoke against a temp `PREFIX`, and an assertion that a dev/test run leaves the home/XDG paths byte-unchanged) rather than leaving them as remembered manual checks.
- Natural end-to-end proof for UAT: the operator's own live instance migrates to `make install`-ed artifacts, and a dev instance is started beside it.

## Progress

**Execution Order:**
Phases execute in numeric order: 15 (decimal insertions, if any, run between their surrounding integers)

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 15. Installed Instance & Dev Isolation | v1.2.0 | 3/5 | In Progress|  |

## Backlog

### Phase 999.1: Plugin distribution, dev guide, certification (BACKLOG)

**Goal:** [Captured 2026-08-05; partially consumed by v1.1.0 Phase 11, which delivers external loading + trust marking] What remains: pulling plugins into a local instance by URL (a marketplace eventually), a detailed plugin development guide plus an agent skill for developing new plugins, and a certification/blessing scheme that can promote an external plugin out of untrusted. The long tail of sources this unlocks — X/Facebook/Bluesky, generic IMAP with provider specialisations, OneDrive/Dropbox, Discord, Atom feeds, Slack/IRC — belongs to their own authors once the path exists.
**Requirements:** PLUG-10, PLUG-11, SRC-07, SRC-08 (see REQUIREMENTS.md Future Requirements)
**Plans:** 0 plans

Plans:

- [ ] TBD (promote with /gsd-review-backlog when ready)
- [ ] Exempt a fully-`extras` source from the kernel's `base_url`/`token`/`path` config-load requirement, or define a fourth top-level connection-detail key such a source declares instead — found by Phase 14's `topos-plugin-gdrive` clean-room build, whose entire declared configuration lives in `extras` (client_id/client_secret/folder_id) and which the kernel currently refuses to boot without a functionally inert `path` stub (GAP-06, `.planning/milestones/v1.1.0-phases/14-google-drive-source-built-out-of-repo/14-CONTRACT-GAPS.md`)

### Phase 999.2: Move functional plugins to topos-plugins sibling repo (BACKLOG)

**Goal:** [Captured 2026-08-18, prompted by the Phase 14 clean-room experiment] Restructure the plugin repos: create a new sibling repo `topos-plugins` holding all functional plugins as sub-directories — moving paperless, silverbullet, proton, whatsapp, filesystem, and signal out of this repo, and folding in the gdrive plugin from `topos-plugin-gdrive`. Only the mock plugins remain in the kernel repo, to facilitate documentation and testing. Rationale: the kernel repo's relationship with in-tree plugins is unmaintainable — plugin knowledge leaks across the Makefile, build manifests, and directory layout. **Open question (needs discuss-phase before planning):** the trust model — in-repo plugins currently inherit trust by directory location (see pending todo "Plugin trust tier is directory-location, not provenance"), and moving every functional plugin out-of-repo forces that question: what distinguishes a first-party plugin from `topos-plugins` from any other external binary? Interacts with 999.1 (distribution/certification) and the Phase 14 contract-gap triage (14-05).
**Requirements:** TBD
**Plans:** 0 plans

Plans:

- [ ] TBD (promote with /gsd-review-backlog when ready)
