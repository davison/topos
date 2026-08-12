# Roadmap: topos

## Milestones

- ✅ **v1.0 MVP** — Phases 1–10 incl. 07.1, 09.1 (shipped 2026-08-12) — [details](milestones/v1.0-ROADMAP.md)
- ⏳ **Next milestone** — not yet defined (run `/gsd-new-milestone`)

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

## Backlog

### Phase 999.1: Plugin ecosystem — out-of-repo plugins, distribution, certification (BACKLOG)

**Goal:** [Captured 2026-08-05 for future planning — likely its own milestone] Plugins move out of the main repo into separate repos: the dozens of plugins needed to make webspaces useful to others (X/Facebook/Bluesky, generic IMAP with Proton/Gmail/Outlook specialisations, GDrive/OneDrive/Dropbox, Discord, Atom feeds, Slack/IRC, and more) would be developed by the people who use those sources and can test them. Needs: a system for pulling plugins into a locally configured instance (by URL configuration first, a marketplace eventually), a detailed plugin development guide plus an agent skill for developing new plugins, and a certification/blessing scheme for external plugins. Depends on Phase 5's contract stabilisation (named instances + per-type matching) — the contract must settle before external authors build against it.
**Requirements:** TBD
**Plans:** 0 plans

Plans:

- [ ] TBD (promote with /gsd-review-backlog when ready)
