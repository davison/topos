# Roadmap: topos

## Milestones

- ✅ **v1.0 MVP** — Phases 1–10 incl. 07.1, 09.1 (shipped 2026-08-12) — [details](milestones/v1.0-ROADMAP.md)
- 🚧 **v1.1.0 Plugin Ecosystem** — Phases 11–14 (in progress)

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

### 🚧 v1.1.0 Plugin Ecosystem (In Progress)

**Milestone Goal:** Open topos to third-party plugins behind an explicit trust boundary, prove the external path by building a new source out-of-repo, and give users finer control over what lands in a webspace.

- [ ] **Phase 11: External Plugins & the Trust Boundary** - Kernel loads out-of-repo plugin binaries, deriving trust from provenance and marking everything else untrusted, loudly
- [ ] **Phase 12: Filesystem Source** - A local or network folder becomes a source; its binary doubles as the external-path rehearsal before Google Drive
- [ ] **Phase 13: Per-Item Curation & Installable App** - Users get the last word on what's in a webspace, and topos installs as a desktop app
- [ ] **Phase 14: Google Drive Source, Built Out-of-Repo** - A Drive folder as a source, delivered by a plugin developed outside the repo against the published contract alone

## Phase Details

### Phase 11: External Plugins & the Trust Boundary

**Goal**: The user can install and run plugin binaries that topos did not build, and always knows which of their sources come from code the project can't vouch for.
**Depends on**: Nothing (extends the v1.0 kernel's plugin host)
**Requirements**: PLUG-06, PLUG-07, PLUG-08, PLUG-09
**Success Criteria** (what must be TRUE):

  1. User drops a plugin binary into the configured external plugins directory and it appears in the source picker's install catalog alongside in-repo plugins — configurable, launchable, and syncing like any other source.
  2. Adding a source backed by an external plugin shows an explicit warning before the user confirms, and the resulting source carries a persistent untrusted badge everywhere it appears (picker and source chip).
  3. Trust is decided by the kernel from where the binary lives plus a content hash pinned when the source was added; swapping the binary afterwards is caught at the next launch and fails loudly by name instead of inheriting stale trust.
  4. An external plugin receives provider-specific config keys the kernel has never heard of (arbitrary per-instance extras), with no kernel change required to add a new key.
  5. A real binary built outside the in-repo plugin set is discovered, marked untrusted, and synced end to end — the mechanism is proven before any out-of-repo source work starts.

**Plans**: 6 plans (5 waves)
Plans:
**Wave 1**

- [ ] 11-01-PLAN.md — Tracer: two-tier discovery, tier published end to end, untrusted chip badge

**Wave 2** *(blocked on Wave 1 completion)*

- [ ] 11-02-PLAN.md — Kernel trust gate: pin verification, soft per-instance failure, env allowlist, extras wire

**Wave 3** *(blocked on Wave 2 completion)*

- [ ] 11-03-PLAN.md — HTTP trust surface + contract/API/example-config republish
- [ ] 11-04-PLAN.md — Out-of-repo proof binary and its end-to-end gate (success criterion 5)

**Wave 4** *(blocked on Wave 3 completion)*

- [ ] 11-05-PLAN.md — UI: untrusted add flow, picker labels, extras form

**Wave 5** *(blocked on Wave 4 completion)*

- [ ] 11-06-PLAN.md — UI: binary-changed state, re-pin flow, pinned-hash footer

**UI hint**: yes

### Phase 12: Filesystem Source

**Goal**: The user can point topos at a folder — local or on a network mount — and see its documents in the right webspace.
**Depends on**: Phase 11 (for the external-path rehearsal criterion only; the in-repo trusted plugin path is independent of it)
**Requirements**: SRC-04
**Success Criteria** (what must be TRUE):

  1. User adds a folder as a source from the UI, with subfolder recursion on or off, and its documents appear in the matching webspace stream with previews.
  2. Files added, changed, or removed in that folder are reflected in the stream on the next sync — including on NFS/SMB mounts, where OS change notifications never fire.
  3. Every filesystem item deep-links back to the document in the desktop's own file handler, or declares honestly that it can only raise (matching the existing deep-link fidelity vocabulary).
  4. The plugin never writes to the source folder — read-only behavior is enforced by committed guards like every other topos plugin.
  5. The filesystem plugin binary, copied into the external plugins directory, loads and syncs identically while showing the untrusted badge — the external path is proven against a real source plugin before the Google Drive work begins.

**Plans**: TBD

### Phase 13: Per-Item Curation & Installable App

**Goal**: The user has the final say over what a webspace shows, and can install topos as an app rather than a browser tab.
**Depends on**: Nothing (independent of Phases 11–12; sequenced here for even flow)
**Requirements**: KERN-09, KERN-10, UI-13, UI-14
**Success Criteria** (what must be TRUE):

  1. User excludes an individual item from a webspace and it leaves the stream; the exclusion survives re-sync, kernel restart, and an index rebuild, and always outranks whatever the automatic match rules say.
  2. User opens a webspace's excluded-items view, sees exactly what they removed, and un-excludes any of it — the items come back into the stream.
  3. User installs topos from the browser on the desktop (manifest, ServiceWorker, icons) and launches it as a standalone app window against the local kernel.
  4. After the kernel is upgraded, the installed app never serves a stale UI — the user lands on the new build rather than a cached one.
  5. The mobile/LAN install limitation (browsers require a secure context) is documented, along with the recommended user-provided HTTPS workarounds.

**Plans**: TBD
**UI hint**: yes

### Phase 14: Google Drive Source, Built Out-of-Repo

**Goal**: A Google Drive folder becomes a topos source, delivered by a plugin developed in a separate repository against nothing but the published contract and installed through the untrusted external path.
**Depends on**: Phase 11 (hard — external loading and the extras config map must exist); Phase 12 (checkpoint — the external path proven against a real source plugin first)
**Requirements**: SRC-05, SRC-06
**Success Criteria** (what must be TRUE):

  1. User supplies their own Google OAuth client via environment references (no secrets in config), authorizes once, and the source keeps syncing across kernel restarts without re-authorizing.
  2. Documents in a chosen Drive folder appear in the webspace stream with previews — including Workspace-native Docs/Sheets/Slides via export — and every item deep-links to the Drive web UI.
  3. Syncs after the first are incremental (changed items only), not a full folder re-listing.
  4. The plugin lives in its own repository, is installed through the external-plugin path, and carries the untrusted badge in the running app — with any place the published contract or mock fell short written down as a contract gap.

**Plans**: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 11 → 12 → 13 → 14 (decimal insertions, if any, run between their surrounding integers)

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 11. External Plugins & the Trust Boundary | v1.1.0 | 0/TBD | Not started | - |
| 12. Filesystem Source | v1.1.0 | 0/TBD | Not started | - |
| 13. Per-Item Curation & Installable App | v1.1.0 | 0/TBD | Not started | - |
| 14. Google Drive Source, Built Out-of-Repo | v1.1.0 | 0/TBD | Not started | - |

## Backlog

### Phase 999.1: Plugin distribution, dev guide, certification (BACKLOG)

**Goal:** [Captured 2026-08-05; partially consumed by v1.1.0 Phase 11, which delivers external loading + trust marking] What remains: pulling plugins into a local instance by URL (a marketplace eventually), a detailed plugin development guide plus an agent skill for developing new plugins, and a certification/blessing scheme that can promote an external plugin out of untrusted. The long tail of sources this unlocks — X/Facebook/Bluesky, generic IMAP with provider specialisations, OneDrive/Dropbox, Discord, Atom feeds, Slack/IRC — belongs to their own authors once the path exists.
**Requirements:** PLUG-10, PLUG-11, SRC-07, SRC-08 (see REQUIREMENTS.md Future Requirements)
**Plans:** 0 plans

Plans:

- [ ] TBD (promote with /gsd-review-backlog when ready)
