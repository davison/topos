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

- [x] **Phase 11: External Plugins & the Trust Boundary** - Kernel loads out-of-repo plugin binaries, deriving trust from provenance and marking everything else untrusted, loudly (completed 2026-08-13)
- [x] **Phase 12: Filesystem Source** - A local or network folder becomes a source; its binary doubles as the external-path rehearsal before Google Drive (UAT gap closure in progress — G-12-1/G-12-3) (completed 2026-08-14)
- [x] **Phase 13: Per-Item Curation & Installable App** - Users get the last word on what's in a webspace, and topos installs as a desktop app (completed 2026-08-15)
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

**Plans**: 7/7 plans executed (5 waves + 1 gap-closure wave)
Plans:
**Wave 1**

- [x] 11-01-PLAN.md — Tracer: two-tier discovery, tier published end to end, untrusted chip badge

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 11-02-PLAN.md — Kernel trust gate: pin verification, soft per-instance failure, env allowlist, extras wire

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 11-03-PLAN.md — HTTP trust surface + contract/API/example-config republish
- [x] 11-04-PLAN.md — Out-of-repo proof binary and its end-to-end gate (success criterion 5)

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 11-05-PLAN.md — UI: untrusted add flow, picker labels, extras form

**Wave 5** *(blocked on Wave 4 completion)*

- [x] 11-06-PLAN.md — UI: binary-changed state, re-pin flow, pinned-hash footer

**Gap closure** *(from 11-VERIFICATION.md — G-11-1, criterion 3 / PLUG-07)*

- [x] 11-07-PLAN.md — Confine ResolveBinary and validate Source.Plugin's shape: close the CR-01 traversal/trust-tier bypass

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

**Plans**: 11/11 plans executed (4 waves + 4 gap-closure waves)
Plans:
**Wave 1**

- [x] 12-01-PLAN.md — Tracer: a PDF in a folder reaches the stream and opens through the kernel-mediated open route

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 12-02-PLAN.md — Document scope, extras-driven globs, and the four preview shapes

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 12-03-PLAN.md — Recursion end to end, symlink/permission-safe walk, honest health, read-only guard

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 12-04-PLAN.md — UI: add a folder source from the picker (checkbox field kind, filesystem row)
- [x] 12-05-PLAN.md — External-tier rehearsal against a real plugin, plus the docs republish

**Gap closure** *(from 12-VERIFICATION.md — G-12-1 / criterion 3, G-12-2 / source-root containment; SRC-04)*

- [x] 12-06-PLAN.md — Symlink-resolving containment at the Fetch and Open sites, an xdg-open child that outlives the request, and the resolved-root walk fix

**Gap closure 2** *(from the 2026-08-14 re-verification — G-12-3: Fetch ignored the include_glob scope that admitted the item; SRC-04)*

- [x] 12-07-PLAN.md — Fetch classifies through the instance's own scope (honest metadata-only preview, no false 404), resolved-path reads/exec, the missing source_system provenance key

**Gap closure 3** *(from 12-UAT.md — G-12-1 / G-12-3: a webspace match value of `folders = ['*']` can never match, and the resulting zero-match state was silent; SRC-04)*

**Wave 1**

- [x] 12-08-PLAN.md — "Everything from this instance" made expressible (root base name labels every file at every depth) plus docs/config that state match values are never globs
- [x] 12-09-PLAN.md — The zero-match diagnostic: an explicit match block that matched nothing is named on the sync run and published as `last_notice`

**Wave 2** *(blocked on 12-09)*

- [x] 12-10-PLAN.md — The advisory becomes visible: warning-tone source chip with the kernel's text, exact-match hint in the shared match form, browser gate over the user's exact config

**Gap closure 4** *(from the 2026-08-14T11:05 re-verification + 12-REVIEW.md — CR-01: 12-10's own tooltip advisory branch gates on `last_status` alone, so an unreachable source with a stale ok status and a leftover notice reads as a benign advisory; SRC-04)*

**Wave 1**

- [x] 12-11-PLAN.md — The tooltip gate defers to healthTone's precedence (`isAdvisoryOnly`), a branch-selection matrix replaces tone-only coverage, and the launch-failure `last_notice` contract is recorded and pinned

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

**Plans**: 8/8 plans executed (13-07 closes 13-VERIFICATION.md; 13-08 closes 13-UAT.md G-13-1)

Plans:
**Wave 1**

- [x] 13-01-PLAN.md — Tracer: item_marks table, mark write/read API, shared exclusion filter, toast layer (wave 1)
- [x] 13-05-PLAN.md — Trust hardening: link-time build-provenance manifest, reordered build, refuse-to-load launch gate (wave 1)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 13-02-PLAN.md — Excluded-bucket view param, excluded_count, orphan prune sweep, API docs (wave 2)
- [x] 13-04-PLAN.md — PWA manifest/icons/ServiceWorker, never-stale update flow, UI-14 docs (wave 2)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 13-03-PLAN.md — Multi-select, floating action bar, excluded-view toggle, include mirror, e2e specs (wave 3)

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 13-06-PLAN.md — Trust hardening UI states, browser specs, republished plugin contract and Signal guidance (wave 4)

**Wave 5** *(gap closure — blocked on Wave 4 completion)*

- [x] 13-07-PLAN.md — Undo toast targets the toast-creation webspace, not the current one; cross-webspace-switch regression spec (wave 5)

**Wave 6** *(gap closure — blocked on Wave 5 completion)*

- [x] 13-08-PLAN.md — Stale-generation load() entry guard (no stranded skeletons after a cross-webspace undo); rendered-stream assertions added to the undo spec (wave 6)

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

**Plans**: 3/5 plans executed

Plans:
**Wave 1**

- [x] 14-01-PLAN.md — Split the dev/test config path from the operator's production config (folded todo; UAT precondition)
- [x] 14-02-PLAN.md — Suppress the native tooltips duplicating the source-chip popover, and bring the Playwright suite onto the chip's new locator surface (folded todo)

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 14-03-PLAN.md — Author the plugin PRD and bootstrap the out-of-repo clean room, then hand off to its own GSD project

**Wave 3** *(blocked on Wave 2 completion)*

- [ ] 14-04-PLAN.md — Prove the Drive plugin on the untrusted external path: binary-gated browser spec plus the recorded live UAT

**Wave 4** *(blocked on Wave 3 completion)*

- [ ] 14-05-PLAN.md — Triage the contract-gap log back into the published contract and the backlog

## Progress

**Execution Order:**
Phases execute in numeric order: 11 → 12 → 13 → 14 (decimal insertions, if any, run between their surrounding integers)

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 11. External Plugins & the Trust Boundary | v1.1.0 | 7/7 | Complete    | 2026-08-13 |
| 12. Filesystem Source | v1.1.0 | 11/11 | Complete    | 2026-08-14 |
| 13. Per-Item Curation & Installable App | v1.1.0 | 8/8 | Complete    | 2026-08-15 |
| 14. Google Drive Source, Built Out-of-Repo | v1.1.0 | 3/5 | In Progress|  |

## Backlog

### Phase 999.1: Plugin distribution, dev guide, certification (BACKLOG)

**Goal:** [Captured 2026-08-05; partially consumed by v1.1.0 Phase 11, which delivers external loading + trust marking] What remains: pulling plugins into a local instance by URL (a marketplace eventually), a detailed plugin development guide plus an agent skill for developing new plugins, and a certification/blessing scheme that can promote an external plugin out of untrusted. The long tail of sources this unlocks — X/Facebook/Bluesky, generic IMAP with provider specialisations, OneDrive/Dropbox, Discord, Atom feeds, Slack/IRC — belongs to their own authors once the path exists.
**Requirements:** PLUG-10, PLUG-11, SRC-07, SRC-08 (see REQUIREMENTS.md Future Requirements)
**Plans:** 0 plans

Plans:

- [ ] TBD (promote with /gsd-review-backlog when ready)

### Phase 999.2: Move functional plugins to topos-plugins sibling repo (BACKLOG)

**Goal:** [Captured 2026-08-18, prompted by the Phase 14 clean-room experiment] Restructure the plugin repos: create a new sibling repo `topos-plugins` holding all functional plugins as sub-directories — moving paperless, silverbullet, proton, whatsapp, filesystem, and signal out of this repo, and folding in the gdrive plugin from `topos-plugin-gdrive`. Only the mock plugins remain in the kernel repo, to facilitate documentation and testing. Rationale: the kernel repo's relationship with in-tree plugins is unmaintainable — plugin knowledge leaks across the Makefile, build manifests, and directory layout. **Open question (needs discuss-phase before planning):** the trust model — in-repo plugins currently inherit trust by directory location (see pending todo "Plugin trust tier is directory-location, not provenance"), and moving every functional plugin out-of-repo forces that question: what distinguishes a first-party plugin from `topos-plugins` from any other external binary? Interacts with 999.1 (distribution/certification) and the Phase 14 contract-gap triage (14-05).
**Requirements:** TBD
**Plans:** 0 plans

Plans:

- [ ] TBD (promote with /gsd-review-backlog when ready)
