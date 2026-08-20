# Roadmap: topos

## Milestones

- ✅ **v1.0 MVP** — Phases 1–10 incl. 07.1, 09.1 (shipped 2026-08-12) — [details](milestones/v1.0-ROADMAP.md)
- ✅ **v1.1.0 Plugin Ecosystem** — Phases 11–14 (shipped 2026-08-18) — [details](milestones/v1.1.0-ROADMAP.md)
- ✅ **v1.2.0 Dev/Prod Separation** — Phase 15 (shipped 2026-08-19) — [details](milestones/v1.2.0-ROADMAP.md)
- 🚧 **v1.3.0 Plugin Repo Split** — Phases 16–18 (started 2026-08-19)

## Milestone v1.3.0: Plugin Repo Split

**Goal:** Every functional plugin moves out of the kernel repo into a sibling
`topos-plugins` repo with its own releases, backed by a real first-party trust
mechanism — so the kernel↔plugin boundary the operator's own sources cross is
the same one a third-party author crosses.

**Shape:** Three phases, ordered by a hard dependency the milestone cannot
dodge. Today's trusted tier is *directory location* — in-repo plugins inherit
trust from where they sit plus a link-time build manifest. Moving all seven
functional plugins out of the repo destroys that basis. So provenance-based
trust must exist in the kernel **before** the move, not after it: Phase 16
builds and proves the verifier while the in-repo plugins are still there to
fall back on, Phase 17 performs the move and stands up the sibling repo's
release pipeline against that verifier, and Phase 18 spends the resulting
machinery on the third-party path (install from a URL, and a guide that walks
an author to a plugin worth installing).

Coarse granularity holds Phase 17 together as one phase despite its nine
requirements: a half-moved plugin set is not a deliverable, and the operator's
live instance must come out the other side with every source still syncing.

## Phases

- [ ] **Phase 16: Provenance-Based Plugin Trust** - The kernel derives trust from verifiable artifact provenance rather than directory location, closing the standing escalation paths
- [ ] **Phase 17: The Plugin Repo Split** - All seven functional plugins move to the `topos-plugins` sibling repo with its own CI, releases, and install path; the kernel repo keeps contract + mocks only
- [ ] **Phase 18: Pull-by-URL Install & the Plugin Developer Guide** - One CLI command takes a plugin from URL to running source in the tier its provenance earns, documented for third-party authors

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

<details>
<summary>✅ v1.2.0 Dev/Prod Separation (Phase 15) — SHIPPED 2026-08-19</summary>

- [x] Phase 15: Installed Instance & Dev Isolation (5/5 plans + 1 inline gap-closure round) — completed 2026-08-19

Full phase details, success criteria, and plan lists: [milestones/v1.2.0-ROADMAP.md](milestones/v1.2.0-ROADMAP.md)

</details>

## Phase Details

### Phase 16: Provenance-Based Plugin Trust

**Goal**: The kernel decides a plugin's trust tier from verifiable provenance carried by the artifact itself, so a first-party plugin earns trust wherever it lives on disk — and no config edit, file drop, or shadow binary can forge it.
**Depends on**: Nothing new (builds on Phase 11's external tier + SHA-256 content pins, and Phase 15's verify-before-place install discipline)
**Requirements**: TRUST-01, TRUST-02, TRUST-03, TRUST-04
**Success Criteria** (what must be TRUE):

  1. A plugin binary carrying valid first-party provenance verifies as **trusted regardless of which directory it resolves from** — the same binary with its provenance absent, stripped, or tampered does not, and says so by name. Trust is no longer a property of location. (TRUST-01)
  2. The verifier is proven against an artifact produced by the **same signing/attestation path `topos-plugins` will use in Phase 17** — not only a hand-crafted test double — and needs no entry in the kernel's link-time build manifest to accept it. (TRUST-02)
  3. An unsigned third-party binary behaves exactly as it does today: consent interstitial, content pin, persistent untrusted badge, two-click re-pin on hash change. The fallback path is unchanged and its existing e2e coverage still passes. (TRUST-03)
  4. Every escalation path named in the standing security todo is closed by a **committed test that fails if its gate is removed**: trust cannot be granted by editing config, by dropping a file into the trusted directory, or by shadowing a trusted plugin name with a different binary. (TRUST-04)
  5. The operator can see *why* a plugin holds the tier it holds — a verification failure names the cause on the source chip and in logs, and never silently downgrades a plugin the operator believes is trusted.

**Plans**: 5/5 plans executed

Plans:
**Wave 1**

- [x] 16-01-PLAN.md — Tracer: ed25519-signed release-manifest format, verifier, embedded key set, two-arm launch gate, and the topos-provenance CLI

**Wave 2** *(blocked on Wave 1 completion)*

- [x] 16-02-PLAN.md — Tier derives from provenance, not from Dirs; both plugin directories become pure search paths (D-11)

**Wave 3** *(blocked on Wave 2 completion)*

- [x] 16-03-PLAN.md — TRUST-04: committed escalation tests (config edit, file drop, shadowing) plus their browser-visible half

**Wave 4** *(blocked on Wave 3 completion)*

- [x] 16-04-PLAN.md — TRUST-02: the topos-plugins sibling repo, its signing workflow, one real signed release, and the embedded key

**Wave 5** *(blocked on Wave 4 completion)*

- [x] 16-05-PLAN.md — Install-time verification, signed-provenance e2e coverage, and the canonical trust-model documentation

**Notes**:

- **Discuss-phase is mandatory before planning** — the backlog entry (999.2) flags the trust model as the open design question of this milestone: what actually distinguishes a first-party `topos-plugins` binary from any other external binary? Sigstore/cosign keyless attestation, a kernel-embedded public key, GitHub artifact attestations, and a signed checksums manifest are all live candidates with very different operational costs for a single-operator desktop tool. Settle that in discussion, not in a plan.
- This phase deliberately lands **before** the move, while the in-repo functional plugins still exist. That keeps the operator's live instance working throughout and gives the verifier a real before/after comparison — the trusted-manifest path must keep working until Phase 17 retires it.
- Criteria 1–4 map directly onto the pending todo *"Plugin trust tier is directory-location, not provenance"* (2026-08-13, major/security). Closing that todo is a deliverable of this phase, not a side effect.
- Tier derivation is visible in the UI (untrusted badge, consent interstitial, chip health). No new UI surface is designed here, but existing Playwright specs assert on those surfaces — per the standing rule (07.1 D-11), any UI-visible change to tier surfacing extends the e2e suite as part of definition of done.

### Phase 17: The Plugin Repo Split

**Goal**: All seven functional plugins live, build, and release from the sibling `topos-plugins` repo; the kernel repo carries the contract and its mocks and nothing else; the operator installs kernel and plugins separately and ends up with the same working instance.
**Depends on**: Phase 16 (provenance verification must exist before the move destroys the directory-based trust basis)
**Requirements**: REPO-01, REPO-02, REPO-03, REPO-04, REPO-05, DIST-01, DIST-02, DIST-03, DIST-04
**Success Criteria** (what must be TRUE):

  1. **The split is done and the kernel repo is plugin-agnostic**: `topos-plugins` holds paperless, silverbullet, proton, whatsapp, filesystem, signal, and the folded-in gdrive as subdirectories, each building and running from that checkout (`topos-plugin-gdrive` superseded); the kernel repo retains only mock/mockstrict, with no Makefile target, build manifest entry, or directory-layout assumption naming a functional plugin. (REPO-01, REPO-02, REPO-04)
  2. **Kernel CI is green on mocks alone**: Go tests, the Playwright e2e suite, and the dev-check gates all pass in a kernel checkout with no functional plugin present anywhere on the machine. (REPO-03)
  3. **The dev loop spans two checkouts**: the developer runs kernel and `topos-plugins` side by side and locally built plugins load into the dev kernel — without breaking Phase 15's dev/installed isolation guarantees (`make isolation-check` still passes). (REPO-05)
  4. **topos-plugins releases and installs on its own**: its own CI/release pipeline publishes versioned, checksummed, provenance-carrying artifacts; a `make install`-grade flow installs and upgrades plugins without touching the kernel and the kernel without touching plugins; the installed kernel resolves those artifacts as first-party trusted via Phase 16's verification; and Signal's local cgo build still works from the `topos-plugins` checkout, landing in the external tier via consent-and-pin with the base install still toolchain-free. (DIST-01, DIST-02, DIST-04)
  5. **Mismatch fails by name**: a kernel↔plugin contract or handshake version mismatch names both the version the kernel requires and the version the plugin offers, on the chip and in logs — never presenting as a source that silently vanished. (DIST-03)

**Plans**: TBD (set by `/gsd-plan-phase 17`)

Plans:

- [ ] TBD

**Notes**:

- The single largest phase of the milestone (nine requirements) and deliberately so: a half-moved plugin set is not a deliverable. Expect a plan-heavy phase sequenced so the operator's live instance never loses a source — the safe order is *sibling repo stands up and releases* → *installed instance migrates onto those artifacts* → *kernel repo deletes its copies*, never the reverse.
- Signal is the one plugin that cannot ship as a published artifact (cgo/SQLCipher — the Phase 10 option-b decision stands). DIST-04 is the "nothing got worse" criterion: the local build path just moves checkouts.
- The e2e suite becomes the kernel repo's *only* plugin-driven coverage after this phase. Criterion 2 is therefore load-bearing for the whole repo's future testability, not a formality — treat any spec that quietly depends on a functional plugin as a defect to fix, not to skip.
- Watch for GAP-06 (fully-`extras` source config refused by the kernel's `base_url`/`token`/`path` requirement) during the gdrive fold-in. It rides along **only if** the fold-in forces it; otherwise it stays in backlog 999.1.
- `docs/testing.md`, `docs/install.md`, and `docs/releasing.md` all describe a world with in-repo plugins. They are part of the move, not a follow-up.

### Phase 18: Pull-by-URL Install & the Plugin Developer Guide

**Goal**: A third-party author can take a plugin from a URL to a running source in one command, and can get from the published contract and mocks to that plugin by following a guide that matches the restructured repo.
**Depends on**: Phase 17 (needs real published `topos-plugins` artifacts to pull, and the restructured kernel repo to document)
**Requirements**: PULL-01, PULL-02, DOCS-01
**Success Criteria** (what must be TRUE):

  1. One CLI command takes a plugin URL and ends with that plugin installed in **the trust tier its provenance earns** — valid first-party provenance lands trusted, anything else lands untrusted through the unchanged consent-and-pin flow. No flag can override the tier. (PULL-01)
  2. A failed download, checksum, or provenance check **aborts with a named reason and leaves nothing placed** — both plugin directories are byte-identical to their state before the attempt, proven by a committed test, not by inspection. (PULL-02)
  3. A developer following the plugin development guide gets from the published contract + mock plugins to a working, installable out-of-repo plugin, with the install step being the same pull-by-URL command from criterion 1. (DOCS-01)
  4. The guide is validated the way Phase 14's clean-room build validated the contract: followed end to end without prior knowledge of the codebase, with the gaps that surface **closed in the guide**, not merely recorded.

**Plans**: TBD (set by `/gsd-plan-phase 18`)

Plans:

- [ ] TBD

**Notes**:

- CLI-first by explicit milestone decision. The in-app install-from-URL flow (`PULL-UI`) stays in backlog 999.1 — this phase adds no UI surface, so the e2e suite extends only if a kernel-side change alters an existing asserted behavior.
- Pull-by-URL is a thin front end over machinery this milestone already built: Phase 16's verifier decides the tier, Phase 15's stage → verify → place discipline decides the mechanics. Resist re-implementing either.
- Criterion 4 is the phase's real gate. A dev guide nobody has followed is a hypothesis; Phase 14 proved the fresh-context walkthrough is what converts it into a documented path.
- DOCS-01 closes the dev-guide half of backlog 999.1. Certification/blessing (`PLUG-CERT`) and marketplace/discovery (`PLUG-MKT`) remain deferred.

## Progress

**Execution Order:**
Phases execute in numeric order: 16 → 17 → 18 (decimal insertions, if any, run between their surrounding integers)

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 16. Provenance-Based Plugin Trust | v1.3.0 | 5/5 | In Progress|  |
| 17. The Plugin Repo Split | v1.3.0 | 0/TBD | Not started | - |
| 18. Pull-by-URL Install & the Plugin Developer Guide | v1.3.0 | 0/TBD | Not started | - |

## Backlog

### Phase 999.1: Plugin distribution, dev guide, certification (BACKLOG)

**Goal:** [Captured 2026-08-05; partially consumed by v1.1.0 Phase 11, which delivers external loading + trust marking; further consumed by v1.3.0 Phase 18, which delivers pull-by-URL (CLI) and the plugin development guide] What remains: a certification/blessing scheme that can promote an external plugin out of untrusted (`PLUG-CERT`), the in-app install-from-URL flow in the source picker (`PULL-UI`), an agent skill for developing new plugins, and a discovery/marketplace surface (`PLUG-MKT`). The long tail of sources this unlocks — X/Facebook/Bluesky, generic IMAP with provider specialisations, OneDrive/Dropbox, Discord, Atom feeds, Slack/IRC — belongs to their own authors once the path exists.
**Requirements:** PLUG-CERT, PULL-UI, PLUG-MKT (see REQUIREMENTS.md Future Requirements). Originally PLUG-10, PLUG-11, SRC-07, SRC-08.
**Plans:** 0 plans

Plans:

- [ ] TBD (promote with /gsd-review-backlog when ready)
- [ ] Exempt a fully-`extras` source from the kernel's `base_url`/`token`/`path` config-load requirement, or define a fourth top-level connection-detail key such a source declares instead — found by Phase 14's `topos-plugin-gdrive` clean-room build, whose entire declared configuration lives in `extras` (client_id/client_secret/folder_id) and which the kernel currently refuses to boot without a functionally inert `path` stub (GAP-06, `.planning/milestones/v1.1.0-phases/14-google-drive-source-built-out-of-repo/14-CONTRACT-GAPS.md`). **Rides along with Phase 17 only if the gdrive fold-in forces it**; otherwise it stays here.

### Phase 999.2: Move functional plugins to topos-plugins sibling repo (PROMOTED → v1.3.0)

**Status:** ✅ Promoted into milestone v1.3.0 (2026-08-19) as Phase 16 (Provenance-Based Plugin Trust) + Phase 17 (The Plugin Repo Split). The flagged open question — the trust model — became Phase 16's own phase, with discuss-phase mandated before planning. No work remains in this backlog entry; it is retained for provenance of the idea.

**Goal:** [Captured 2026-08-18, prompted by the Phase 14 clean-room experiment] Restructure the plugin repos: create a new sibling repo `topos-plugins` holding all functional plugins as sub-directories — moving paperless, silverbullet, proton, whatsapp, filesystem, and signal out of this repo, and folding in the gdrive plugin from `topos-plugin-gdrive`. Only the mock plugins remain in the kernel repo, to facilitate documentation and testing. Rationale: the kernel repo's relationship with in-tree plugins is unmaintainable — plugin knowledge leaks across the Makefile, build manifests, and directory layout. **Open question (needs discuss-phase before planning):** the trust model — in-repo plugins currently inherit trust by directory location (see pending todo "Plugin trust tier is directory-location, not provenance"), and moving every functional plugin out-of-repo forces that question: what distinguishes a first-party plugin from `topos-plugins` from any other external binary? Interacts with 999.1 (distribution/certification) and the Phase 14 contract-gap triage (14-05).
**Requirements:** REPO-01..05, TRUST-01..04, DIST-01..04 (v1.3.0)
**Plans:** promoted — see Phase 16 and Phase 17 above
