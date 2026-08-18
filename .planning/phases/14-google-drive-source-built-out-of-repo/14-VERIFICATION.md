---
phase: 14-google-drive-source-built-out-of-repo
verified: 2026-08-18T13:10:00Z
status: human_needed
score: 9/9 must-haves verified
behavior_unverified: 0
overrides_applied: 0
human_verification:
  - test: "Confirm whether the touch-only (<768px, no screen reader) loss of chip health-detail access is an acceptable regression, or route a follow-up fix (e.g. a tap-to-reveal affordance, or a media-query-scoped title fallback below 768px)."
    expected: "An explicit accept/reject decision recorded (e.g. a WINDOWS.md ledger entry, a new todo, or a phase assignment), rather than the current state: a documented-but-undecided regression flagged by 14-02-SUMMARY.md (coverage D4, human_judgment: true) and by 14-REVIEW.md (WR-01), with the intended WINDOWS.md ledger entry never actually written (gsd-tools' `windows` subcommand was unavailable when 14-02 tried)."
    why_human: "This is a product/UX tradeoff — losing a previously-shipped touch accessibility guarantee (09.1-04-PLAN.md R2) — that no automated check can validate as 'correct' or 'incorrect'; a human must decide whether the tradeoff is acceptable as shipped."
  - test: "At the shortest supported mobile-takeover viewport, trigger a Drive source in the folder-inaccessible or rate-limited health state and visually confirm the health sentence renders legibly inside the chip's tooltip without being clipped or overflowing."
    expected: "The long health-state sentence wraps or truncates gracefully in the mobile-takeover popover, matching every other source's health-sentence rendering at that viewport."
    why_human: "14-04-PLAN.md itself declares this a `verification: backstop` truth — the UI-SPEC's own single unconfirmed item, explicitly never visually checked with these exact string lengths at the shortest supported viewport. A backstop truth abstains to human review by design; it was never claimed VERIFIED by any plan or summary."
---

# Phase 14: Google Drive Source, Built Out-of-Repo Verification Report

**Phase Goal:** A Drive folder as a source, delivered by a plugin developed outside the repo against the published contract alone.
**Verified:** 2026-08-18T13:10:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

Must-haves merged from ROADMAP.md's four Success Criteria (the roadmap contract) plus each plan's frontmatter `must_haves.truths`, deduplicated. SC1–SC3 are proven only by a real Google account and are therefore evidenced by the recorded, operator-run live UAT (`14-LIVE-UAT.md`) rather than by this session — per this task's own instruction, that operator attestation is treated as real evidence (a completed `checkpoint:human-action`/`human-check` gate, explicitly approved), not as independently machine-verified fact re-witnessed by this verifier.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SC1 — Operator supplies own OAuth client via env refs (no secrets in config), authorizes once, source keeps syncing across kernel restarts with no re-auth | ✓ VERIFIED (human attestation) | `14-LIVE-UAT.md` Results Table row 1, run 2026-08-18, operator's verbatim report "everything passes"; the resolved plan gate `checkpoint` for Task 2 of 14-04-PLAN.md; the auth mechanism (`--config`/env-ref only, never a literal secret in config) is structurally confirmed by `14-PLUGIN-PRD.md`'s Locked Decisions and by `grep`-verified absence of any credential value in `14-LIVE-UAT.md`. Evidence quality: blanket report, no specific witnessed detail (date/log/screenshot) recorded — noted explicitly in the source document itself. |
| 2 | SC2 — Documents in the chosen folder appear in the stream with previews (incl. Workspace Docs/Sheets/Slides via export); every item deep-links to the Drive web UI | ✓ VERIFIED (human attestation) | `14-LIVE-UAT.md` Results Table row 2, same run/report. Same evidence-quality caveat as row 1. |
| 3 | SC3 — Syncs after the first are incremental, not a full folder re-listing | ✓ VERIFIED (human attestation) | `14-LIVE-UAT.md` Results Table row 3, same run/report. Same evidence-quality caveat as row 1. |
| 4 | SC4 — Plugin lives in its own repository, installed through the external-plugin path, carries the untrusted badge, and any contract/mock shortfall is written down as a contract gap | ✓ VERIFIED | Hermetic, machine-run: `make gdrive-external-rehearsal TOPOS_GDRIVE_BIN=.../topos-plugin-gdrive` — 5/5 tests pass against the real, out-of-repo binary (re-run live by this verifier, not just trusted from SUMMARY). `grep -rn 'topos-plugin-gdrive' kernel/ web/src/ internal/ cmd/` returns nothing. `/home/darren/projects/davison/topos-plugin-gdrive` confirmed a real, separate git repo with 34 non-vendored `.go` files, pushed to `github.com/davison/topos-plugin-gdrive`, `main` in sync with `origin/main`. `14-CONTRACT-GAPS.md` (this repo) confirms 20/20 gap ids imported and triaged; `docs/plugin-contract.md` confirms the "Plugin-private state" section and 8 other sections republished. |
| 5 | The Drive plugin loads from external plugins dir, marked external tier, untrusted badge — no copy in a trusted dir | ✓ VERIFIED | `14-gdrive-external-rehearsal.spec.ts` test 1 (tier=external) and test 4 (chip carries exactly one trust badge) — both re-run live, pass. Spec sets `pluginBinaries: []`. |
| 6 | Add-source flow renders the plugin's three declared extras + folders match vocabulary from Describe alone, no in-repo row for it | ✓ VERIFIED | `14-gdrive-external-rehearsal.spec.ts` test 2 (`POST /api/config/describe-plugin`) — re-run live, passes; `grep` confirms zero in-repo source references the plugin name. |
| 7 | An unauthorized Drive source reports itself unreachable with the named not-authorized sentence, never healthy-but-empty | ✓ VERIFIED | `14-gdrive-external-rehearsal.spec.ts` test 3 — re-run live, passes (`.toContain` on the exact PRD/UI-SPEC sentence). |
| 8 | Chip exposes the untrusted-external-plugin clause via the accessible-description surface (14-02's option-b) | ✓ VERIFIED | `14-gdrive-external-rehearsal.spec.ts` test 5 — re-run live, passes; `source-chip-tooltip.test.ts` (20/20 pass, re-run live) pins `aria-describedby`/`sr-only` structurally. |
| 9 | The untrusted-confirm interstitial names GDRIVE_CLIENT_ID and GDRIVE_CLIENT_SECRET by name | ✓ VERIFIED (human attestation) | `14-LIVE-UAT.md` Install step 11 + Results row 4, operator-attested; not covered by any hermetic spec (the interstitial's dynamic disclosure text was not asserted in `14-gdrive-external-rehearsal.spec.ts`). |

**Score:** 9/9 truths verified (0 present-but-behavior-unverified). Two additional items — a UX-tradeoff decision and a UI-SPEC-declared backstop truth — are open and routed to Human Verification below; neither is a FAILED must-have.

### Plan-Level Must-Haves (14-01 through 14-05)

All plan-level `must_haves.truths`, `artifacts`, and `key_links` were checked directly against the codebase (not accepted from SUMMARY claims) and passed:

| Plan | Must-have area | Verification performed |
|------|----------------|-------------------------|
| 14-01 | `--config`/`TOPOS_CONFIG` precedence; per-checkout `config.dev.toml` | `go build` + ran the built binary against a nonexistent `--config` path — error names that path, not the XDG default; `go test ./cmd/topos/...` passes; `Makefile` confirmed to contain `DEV_CONFIG`, `dev-config` target (create-only, `sed 's|@CHECKOUT@|...'`), `dev: plugins dev-config`; `config.dev.example.toml` confirmed tracked with `@CHECKOUT@` placeholders and no `[sources]`; `.gitignore` confirmed to ignore `/config.dev.toml`; `docs/testing.md` confirmed to contain the "real config and the dev config" section naming `TOPOS_CONFIG`. |
| 14-02 | Native tooltip suppression; accessible description preserved | `grep` confirms zero `title={tooltipText}`/`title={source.display_name}` on `SourceChip.svelte`, exactly one `title={source.pinned_hash}` (untouched footer span); `aria-describedby={chipDescId}` + `sr-only` span present; `npm run test -- source-chip-tooltip` (20/20 pass, re-run live); `npm run check` (svelte-check, 0 errors, re-run live). |
| 14-03 | PRD as sole hand-off; clean-room repo bootstrap; hand-off checkpoint resolved | `14-PLUGIN-PRD.md` confirmed to exist, contain `GDRIVE_CLIENT_SECRET`, and cite no `kernel/`/`internal/` path (re-checked live); sibling repo confirmed on disk as a real, separate, pushed git repository with a real, built ELF binary that prints the expected `hashicorp/go-plugin` subprocess-safety message when run directly (re-run live); `CONTRACT-GAPS.md` in the sibling repo confirmed to contain 20 `### GAP-` entries (re-counted live). |
| 14-04 | Hermetic external-tier proof; live UAT script + run | See rows 4–9 above. |
| 14-05 | Gap-log triage; contract republish; backlog filing | `14-CONTRACT-GAPS.md` confirmed 20 `### GAP-` entries, both seeded `GAP-01`/`GAP-02` present; `docs/plugin-contract.md` confirmed a "Plugin-private state" heading plus 3 other cross-references; `.planning/ROADMAP.md` confirmed a new backlog line under Phase 999.1 citing `GAP-06`; `.planning/REQUIREMENTS.md` confirmed a new evidence line under `PLUG-11` citing all 20 gap ids; `make docs-check` passes (re-run live). |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `cmd/topos/main.go` | `--config`/`TOPOS_CONFIG` precedence chain | ✓ VERIFIED | `resolveConfigPath`/`parseConfigFlag` present and wired into `serve`/`sync`; live-tested |
| `cmd/topos/configpath_test.go` | Precedence table test | ✓ VERIFIED | `go test ./cmd/topos/...` passes |
| `config.dev.example.toml` | Tracked per-checkout template | ✓ VERIFIED | Tracked, contains `@CHECKOUT@`, no `[sources]` |
| `Makefile` | `DEV_CONFIG`, `dev-config`, `gdrive-external-rehearsal` | ✓ VERIFIED | All present and functional (live-run) |
| `docs/testing.md` | Dev-config split + Drive spec docs | ✓ VERIFIED | Both sections present |
| `web/src/lib/components/SourceChip.svelte` | Native-tooltip attrs removed, `aria-describedby` added | ✓ VERIFIED | Confirmed by grep + passing test suite |
| `web/src/lib/components/source-chip-tooltip.test.ts` | Structural suppression assertions | ✓ VERIFIED | 20/20 pass |
| `.planning/phases/.../14-PLUGIN-PRD.md` | Sole hand-off document | ✓ VERIFIED | Present, all required content confirmed |
| `/home/darren/projects/davison/topos-plugin-gdrive/` (sibling repo, read-only checked) | Clean-room repo, PRD copy, gap log, built binary | ✓ VERIFIED | Real repo, pushed, real binary, 20-entry gap log |
| `web/e2e/specs/14-gdrive-external-rehearsal.spec.ts` | Hermetic external-tier proof | ✓ VERIFIED | 5/5 pass against real binary (re-run live) |
| `.planning/phases/.../14-LIVE-UAT.md` | Live UAT script + results | ✓ VERIFIED (present + filled) | Present, filled in as operator-attested (evidence-quality caveat noted throughout) |
| `.planning/phases/.../14-CONTRACT-GAPS.md` | Imported + triaged gap log | ✓ VERIFIED | 20/20 entries, all dispositioned |
| `docs/plugin-contract.md` | Republished documentation-fixable gaps | ✓ VERIFIED | "Plugin-private state" section + 8 other sections |
| `.planning/ROADMAP.md` | Backlog item for contract-change gap | ✓ VERIFIED | GAP-06 filed under Phase 999.1 |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `Makefile` | `cmd/topos/main.go` | `DEV_KERNEL_CMD` passes `--config $(DEV_CONFIG)` | ✓ WIRED | Confirmed in Makefile source |
| `web/e2e/specs/14-gdrive-external-rehearsal.spec.ts` | `web/e2e/fixtures/plugin-binaries.ts` | `externalPluginBinariesSrcDir` points outside `bin/plugins` | ✓ WIRED | Confirmed by live test run — binary loaded external-tier only |
| `web/e2e/specs/14-gdrive-external-rehearsal.spec.ts` | `web/src/lib/components/SourceChip.svelte` | Untrusted-badge + accessible-description assertions | ✓ WIRED | Test 4/5 pass live |
| `14-CONTRACT-GAPS.md` | `docs/plugin-contract.md` | Triage rows name landing sections | ✓ WIRED | Cross-checked ("private state" section exists as named) |
| `14-CONTRACT-GAPS.md` | `.planning/ROADMAP.md` | Contract-change row names backlog item | ✓ WIRED | `GAP-06` present in both files |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Config-path resolver rejects nonexistent `--config` path, names it (not XDG default) | `go build -o /tmp/topos-cfgcheck ./cmd/topos && /tmp/topos-cfgcheck serve --config /nonexistent/...` | Exit 1, error names the given path | ✓ PASS |
| `cmd/topos` test suite | `go test ./cmd/topos/...` | ok | ✓ PASS |
| Svelte type/lint check | `npm --prefix web run check` | 0 errors | ✓ PASS |
| Chip tooltip suppression unit tests | `npm --prefix web run test -- source-chip-tooltip` | 20/20 pass | ✓ PASS |
| Drive external-tier hermetic spec against the REAL out-of-repo binary | `make gdrive-external-rehearsal TOPOS_GDRIVE_BIN=/home/darren/projects/davison/topos-plugin-gdrive/topos-plugin-gdrive` | 5/5 pass | ✓ PASS |
| No in-repo source names the plugin | `grep -rn 'topos-plugin-gdrive' kernel/ web/src/ internal/ cmd/` | empty | ✓ PASS |
| Sibling repo pushed and in sync | `git -C topos-plugin-gdrive status --porcelain=v1 --branch` | `## main...origin/main`, no divergence | ✓ PASS |
| Docs link check | `make docs-check` | 40 links resolve | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|-----------------|--------------|--------|----------|
| SRC-05 | 14-01, 14-03, 14-04 | BYO Google OAuth, authorize once + restart survival, incremental sync, Workspace-export previews, deep links | ✓ SATISFIED (human-attested for the live-only behaviors) | `14-LIVE-UAT.md` results table (operator-attested); `14-01`'s config split structurally supports the dev-loop precondition |
| SRC-06 | 14-02, 14-03, 14-04, 14-05 | Plugin developed out-of-repo against the published contract, installed through the external path, proving a third party can ship a working untrusted plugin | ✓ SATISFIED | Live-verified: real sibling repo, real binary, 5/5 hermetic e2e tests, 20-entry gap log fully triaged and republished |

No orphaned requirements: both SRC-05 and SRC-06 appear in REQUIREMENTS.md's phase-14 mapping and are each claimed by at least one plan's `requirements:` frontmatter. REQUIREMENTS.md's checkboxes for SRC-05/SRC-06 are currently unchecked ("Pending") — expected, since the orchestrator marks them complete only after verification passes; not a gap.

### Anti-Patterns Found

Scanned all files touched by this phase (per SUMMARY key-files) for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`/stub-return patterns. No debt markers or stubs found in any phase-14-modified source file. The three `TBD` hits in `.planning/ROADMAP.md` are pre-existing Phase 999.1/999.2 backlog placeholders, confirmed via `git show` on 14-05's own commit to be untouched by this phase's diff (only a new bullet line was added). No blockers.

Two advisory findings carried from `14-REVIEW.md` (status: `issues_found`, 0 critical, 2 warnings, 4 info — not independently re-litigated here, but cross-checked against the codebase):

- **WR-01** (warning): the touch-only (<768px, no screen reader) accessibility regression from removing the chip's native `title` attribute is real, documented (`14-02-SUMMARY.md` coverage D4, `human_judgment: true`), but never brought to an explicit accept/reject decision, and its intended `WINDOWS.md` ledger entry was never actually written (the `gsd-tools windows` subcommand was unavailable when 14-02 tried). Routed to Human Verification below.
- **WR-02** (warning): `gdrive-external-rehearsal`'s Makefile target duplicates `e2e`'s fixture-build steps rather than sharing a common prerequisite — a maintainability/drift-risk nit, not a functional defect (confirmed live: the duplicated build steps did in fact produce working fixtures). Not blocking; not re-routed to human review since it is a code-quality concern already surfaced by REVIEW.md, not a phase-goal gap.

### Human Verification Required

### 1. Touch-only accessibility regression — accept or route a follow-up

**Test:** Review `14-02-SUMMARY.md`'s "Known Regression" section and `14-REVIEW.md`'s WR-01. Decide whether shipping with no touch-only (sub-768px, non-screen-reader) path to a source chip's health detail is acceptable as-is, or whether a follow-up (tap-to-reveal popover, conditional `title` below 768px, etc.) should be filed.
**Expected:** An explicit, recorded decision — a `WINDOWS.md` ledger entry, a new todo, or an assigned follow-up phase — rather than the current open-ended state.
**Why human:** Product/UX tradeoff; no automated check can determine "acceptable" here, and the plan's own coverage item (D4) already asked for this decision but it was never actually recorded anywhere actionable.

### 2. Long health-message legibility in the mobile-takeover chip tooltip

**Test:** At the shortest supported viewport, reach the folder-inaccessible or rate-limited health state on a Drive source and visually confirm the sentence renders legibly in the chip's mobile-takeover tooltip.
**Expected:** The sentence wraps/truncates the same way every other source's health sentence does at that viewport — no clipping, no overflow.
**Why human:** Declared a `verification: backstop` truth by 14-04-PLAN.md itself — the UI-SPEC's own single unconfirmed item, explicitly never checked with these exact string lengths at this viewport. This is not a gap introduced by this verification; it was correctly never claimed VERIFIED by any plan or SUMMARY.

### Gaps Summary

No FAILED must-haves. Every artifact, key link, and roadmap Success Criterion has direct codebase or recorded-attestation evidence, most of it re-run live by this verification session rather than trusted from SUMMARY claims (config-path resolver, svelte-check, chip tooltip unit tests, and — most materially — the 5-test hermetic Playwright spec against the real, out-of-repo `topos-plugin-gdrive` binary, plus a read-only check of the sibling repository's real git history, push state, and built binary).

The phase is not blocked, but two items need a human decision before this phase can be considered fully closed: (1) an explicit accept/reject on the documented touch-accessibility regression, which has been openly flagged since 14-02 but never formally resolved or ledgered, and (2) the UI-SPEC's own self-declared backstop truth about long health-message legibility on narrow viewports, which no plan or summary ever claimed to have checked.
