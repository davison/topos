# Genesis: the GSD era

Written 2026-08-31 at the migration to CodeCrew
([#7](https://github.com/davison/topos/issues/7), the first task of
[#6](https://github.com/davison/topos/issues/6)). This record predates the
protocol it now lives under: everything before it was built with GSD
(Get Shit Done), whose complete working state — roadmaps, requirements,
phase plans, verification records, retrospectives — is frozen under
[`.planning/`](../../.planning/) as the archive this page maps. Nothing
there was rewritten for the migration; it is linked, not converted.

## What topos is

A locally-run service that pulls together related information from
disparate personal data silos — email, Signal and WhatsApp chats, document
storage, note-taking apps, wikis, project directories — into per-topic
"webspaces": open one and see all related objects across every silo in a
rich, view-only interface, with deep links back to the original
application. ([PROJECT.md](../../.planning/PROJECT.md) keeps the full
statement.)

## The ladder

| GSD milestone | Phases | Shipped | Record |
|---|---|---|---|
| v1.0 MVP | 1–10 (+07.1, 09.1 inserted) | 2026-08-12 | [v1.0-ROADMAP](../../.planning/milestones/v1.0-ROADMAP.md) |
| v1.1.0 Plugin Ecosystem | 11–14 | 2026-08-18 | [v1.1.0-ROADMAP](../../.planning/milestones/v1.1.0-ROADMAP.md) |
| v1.2.0 Dev/Prod Separation | 15 | 2026-08-19 | [v1.2.0-ROADMAP](../../.planning/milestones/v1.2.0-ROADMAP.md) |
| v1.3.0 Plugin Repo Split — first third | 16 | phase complete 2026-08-20 | [phase 16](../../.planning/phases/16-provenance-based-plugin-trust/) |

- **v1.0 MVP** — the kernel spine with a published go-plugin/gRPC contract
  and five real sources (paperless-ngx, SilverBullet, Proton/IMAP, Signal,
  WhatsApp), webspaces built and edited entirely from the UI. 12 phases,
  92 plans, 798 commits in 17 days; ~42k LOC Go, ~25k LOC Svelte.
- **v1.1.0 Plugin Ecosystem** — third-party plugins behind an explicit
  trust boundary, proven end-to-end by a Google Drive source built
  clean-room out-of-repo against the published contract alone; per-item
  curation; PWA install. 4 phases, 309 commits in 6 days.
- **v1.2.0 Dev/Prod Separation** — topos became an installed application:
  `make install` from checksum-verified release artifacts, mechanical
  dev/prod isolation (topos-devguard, `make isolation-check`), data-safe
  uninstall. 1 phase, 42 commits in 2 days.
- **Phase 16, Provenance-Based Plugin Trust** — trust derives from
  ed25519-signed release provenance rather than directory location; the
  escalation paths (config edit, file drop, binary shadowing) are closed
  by committed tests; the `topos-plugins` sibling repo exists with its
  signing workflow and one real signed release. Completed under GSD as the
  first third of v1.3.0; its remainder is this hub's first CodeCrew
  milestone ([#6](https://github.com/davison/topos/issues/6)), which ships
  the v1.3.0 tag.

Full delivered/stats/closeout paragraphs per milestone:
[MILESTONES.md](../../.planning/MILESTONES.md).

## The migration decision

Recorded 2026-08-31, decided by the operator:

- **davison/topos is its own hub**, never a spoke of another project's
  ([radiusred/gh-codecrew](https://github.com/radiusred/gh-codecrew) is
  the framework's hub and hosts only its own development).
- **Solo routing**: all five roles are held by the operator (`~` in
  `.codecrew.yml`); no GitHub Apps. Seats may be staffed later — the
  routing table is the seam.
- **The reviewer seat runs on a Codex harness**: reviews are performed by
  a fresh headless Codex session loaded with the reviewer contract
  (`gh codecrew roles show reviewer`) for de-correlated judgment, posting
  under the operator's authority; merges use
  `task finish --operator-confirm`.
- **Two numbering schemes, no collision**: CodeCrew milestones count
  M1, M2, … from this migration; the GSD era counted version-named
  milestones over continuous phases 1–16. CodeCrew requirement IDs look
  like `M1-R4`; GSD's look like `REPO-01` and are cited in requirement
  prose for traceability. Release tags stay semver and are shared by both
  eras — the next tag after v1.2.0 is v1.3.0, cut at the first CodeCrew
  milestone's close.
- **`.planning/` is frozen** as the GSD archive; the CodeCrew record
  (issues, this directory, `ROADMAP.md`) takes over from here. The GSD
  toolchain is not used in this repository beyond reading the archive.

## Carried seeds

GSD's deferred requirements were re-filed as backlog captures at the
migration so they live on the record rather than in an archive file:
[#1](https://github.com/davison/topos/issues/1) PLUG-CERT ·
[#2](https://github.com/davison/topos/issues/2) PULL-UI ·
[#3](https://github.com/davison/topos/issues/3) PLUG-MKT ·
[#4](https://github.com/davison/topos/issues/4) KERN-OAUTH ·
[#5](https://github.com/davison/topos/issues/5) Signal schema tooling.
