# Roadmap: topos

## Milestones

- ✅ **v1.0 MVP** — Phases 1–10 incl. 07.1, 09.1 (shipped 2026-08-12) — [details](milestones/v1.0-ROADMAP.md)
- ✅ **v1.1.0 Plugin Ecosystem** — Phases 11–14 (shipped 2026-08-18) — [details](milestones/v1.1.0-ROADMAP.md)

## Phases

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

## Backlog

### Phase 999.1: Plugin distribution, dev guide, certification (BACKLOG)

**Goal:** [Captured 2026-08-05; partially consumed by v1.1.0 Phase 11, which delivers external loading + trust marking] What remains: pulling plugins into a local instance by URL (a marketplace eventually), a detailed plugin development guide plus an agent skill for developing new plugins, and a certification/blessing scheme that can promote an external plugin out of untrusted. The long tail of sources this unlocks — X/Facebook/Bluesky, generic IMAP with provider specialisations, OneDrive/Dropbox, Discord, Atom feeds, Slack/IRC — belongs to their own authors once the path exists.
**Requirements:** PLUG-10, PLUG-11, SRC-07, SRC-08 (see REQUIREMENTS.md Future Requirements)
**Plans:** 0 plans

Plans:

- [ ] TBD (promote with /gsd-review-backlog when ready)
- [ ] Exempt a fully-`extras` source from the kernel's `base_url`/`token`/`path` config-load requirement, or define a fourth top-level connection-detail key such a source declares instead — found by Phase 14's `topos-plugin-gdrive` clean-room build, whose entire declared configuration lives in `extras` (client_id/client_secret/folder_id) and which the kernel currently refuses to boot without a functionally inert `path` stub (GAP-06, `.planning/phases/14-google-drive-source-built-out-of-repo/14-CONTRACT-GAPS.md`)

### Phase 999.2: Move functional plugins to topos-plugins sibling repo (BACKLOG)

**Goal:** [Captured 2026-08-18, prompted by the Phase 14 clean-room experiment] Restructure the plugin repos: create a new sibling repo `topos-plugins` holding all functional plugins as sub-directories — moving paperless, silverbullet, proton, whatsapp, filesystem, and signal out of this repo, and folding in the gdrive plugin from `topos-plugin-gdrive`. Only the mock plugins remain in the kernel repo, to facilitate documentation and testing. Rationale: the kernel repo's relationship with in-tree plugins is unmaintainable — plugin knowledge leaks across the Makefile, build manifests, and directory layout. **Open question (needs discuss-phase before planning):** the trust model — in-repo plugins currently inherit trust by directory location (see pending todo "Plugin trust tier is directory-location, not provenance"), and moving every functional plugin out-of-repo forces that question: what distinguishes a first-party plugin from `topos-plugins` from any other external binary? Interacts with 999.1 (distribution/certification) and the Phase 14 contract-gap triage (14-05).
**Requirements:** TBD
**Plans:** 0 plans

Plans:

- [ ] TBD (promote with /gsd-review-backlog when ready)
