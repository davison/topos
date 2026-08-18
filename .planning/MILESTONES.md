# Milestones

## v1.1.0 Plugin Ecosystem (Shipped: 2026-08-18)

**Delivered:** topos opened to third-party plugins behind an explicit, provenance-derived trust boundary — proven end-to-end by a Google Drive source plugin built in a clean-room repository against the published contract alone and installed through the untrusted external path — plus per-item curation and PWA installability.

**Stats:** 4 phases (11–14), 32 plans + 2 gap-closure waves + 5 quick tasks, 309 commits over 6 days (2026-08-12 → 2026-08-18). 204 files changed, +32,811/−738 LOC (excluding .planning).

**Closeout:** verified — all 4 phases verification-passed, milestone audit passed (11/11 requirements wired, 0 blockers), phase-14 security audit 26/26 threats closed (`threats_open: 0`), e2e suite 145 specs at zero retries. Carried debt tracked in pending todos (kernel OAuth/secrets abstraction — requirement-grade for next milestone; dev-port clash; popover-clone tooltip interception; Signal schema tooling) and two non-blocking review advisories (Phase 13 WR-01 UI-state clobber; Phase 14 Makefile recipe duplication).

**Key accomplishments:**

1. **External plugin trust boundary** — kernel loads out-of-repo binaries from a distinct external tier; trust derived purely from provenance (directory tier + SHA-256 content pin re-verified at every launch, build-manifest gate for the trusted tier, shadowing advisories); explicit type-to-confirm warning interstitial and persistent untrusted badges everywhere a source appears.
2. **Third-party path proven end-to-end** — the Google Drive plugin was developed in a genuine clean-room repository (`github.com/davison/topos-plugin-gdrive`) against the published contract alone, installed through the untrusted external path, hermetically e2e-tested and live-verified against a real Drive (BYO OAuth, incremental sync, Workspace export previews, deep links); its 20 contract gaps were triaged back into the published contract (19 documentation republishes, 1 filed against PLUG-11).
3. **Filesystem source** — local/network folders as a trusted in-repo plugin (glob scoping, zero-match advisories, stat-diff polling on network mounts) that doubled as the external-path rehearsal vehicle before the Drive work.
4. **Per-item curation** — include/exclude marks as user-owned kernel-index data with undo and an excluded-items view; marks survive re-sync/rebuild, outrank match rules, and are source-agnostic by construction.
5. **Installable app** — PWA manifest + ServiceWorker with a never-stale update flow; mobile/LAN secure-context limitation documented with workarounds.
6. **Engineering surface hardened** — dev/prod config isolation (`--config`/`TOPOS_CONFIG`, per-checkout dev config), plugin contract expanded (plugin-private state, extras passthrough guidance), e2e suite grown 42 → 145 specs at zero retries, and the milestone's own out-of-repo exercise seeded the next milestone's backlog (kernel OAuth/secrets services, `topos-plugins` repo restructure).

**Archives:**

- `milestones/v1.1.0-ROADMAP.md` — full phase details (4 phases, all plans)
- `milestones/v1.1.0-REQUIREMENTS.md` — all v1.1.0 requirements with outcomes
- `.planning/v1.1.0-MILESTONE-AUDIT.md` — cross-phase integration audit

## v1.0 MVP (Shipped: 2026-08-12)

**Delivered:** A locally-run personal-data kernel that correlates five real sources — paperless-ngx, SilverBullet, Proton Mail, Signal, and WhatsApp — into per-topic webspaces, browsable and configurable entirely from a web UI, released with docs, CI, and publishing automation.

**Stats:** 12 phases (10 planned + 07.1/09.1 inserted), 92 plans, 241 tasks, 798 commits over 17 days (2026-07-27 → 2026-08-12). ~42k LOC hand-written Go (kernel + 7 plugin modules), ~25k LOC Svelte 5/TypeScript.

**Closeout:** verified — all 12 phases verification-passed, 31/31 v1 requirements complete, 17 debug sessions verified fixed-in-code and closed at the pre-close sweep. Deferred: 1 pending todo (Signal schema-version verify-and-accept tooling — see STATE.md Deferred Items). UF-10-01 (personal data visible in README screenshots) recorded as operator-accepted risk, no remediation planned.

**Key accomplishments:**

1. **Kernel spine with a published plugin contract** — go-plugin/gRPC subprocess isolation, SQLite+FTS5 index, hybrid data model (local metadata/preview, live fetch on open), read-only and egress guarantees enforced by committed AST tests; PLUG-05 proved a third party can build a plugin from the contract + mock alone.
2. **Five real sources, sequenced by ascending risk** — paperless-ngx, SilverBullet, Proton/IMAP (never-marks-read proven four independent ways), Signal (SQLCipher read strictly `mode=ro`, byte-identical with Signal running, runtime keyring detection), and WhatsApp (whatsmeow linked device with its own persistent store, five named health states degrading honestly on de-link/ban).
3. **Named source instances with per-instance typed matching** — the config map key is the kernel's source identity everywhere; plugins declare their match vocabulary on the wire (contract generation topos.v2); kernel owns the rendition sanitize/wrap/theme boundary.
4. **Webspace builder UI** — the kernel's first mutating surface (hash-guarded `PUT /api/config`, canonical TOML rewrite, secrets stay environment-only) with hot apply rebuilding plugin host/coordinator/scheduler in place; webspaces created, composed, and filtered (search-to-filter promotion) entirely from the browser.
5. **A UI that scales and ships** — merged per-instance source chips with overflow, deep-link fidelity differentiation, search highlighting, date-marker ruler, plugin identity icons in the contract, mobile detail takeover below 768px, and first-run bootstrap writing a default config.
6. **Regression armor and release engineering** — hermetic Playwright e2e suite (42 tests) driving the shipped binary in CI alongside Go/svelte-check/vitest gates; change-gated nightlies, tag-triggered release artifacts (static builds; Signal plugin deliberately excluded), GitHub milestone mirror script, new-user README/CONTRIBUTING/SECURITY/per-plugin docs.

**Archives:**

- `milestones/v1.0-ROADMAP.md` — full phase details (12 phases, all plans)
- `milestones/v1.0-REQUIREMENTS.md` — all 31 requirements with outcomes
- `milestones/v1.0-phases/` — complete phase execution history (plans, summaries, verifications, UAT, reviews)

*Per-plan accomplishment detail lives in the archived phase SUMMARY.md files.*
