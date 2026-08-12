---
phase: 10-docs-and-release-readiness
plan: 05
subsystem: docs
tags: [readme, contributing, ci, doc-link-guard, release-engineering]

# Dependency graph
requires:
  - phase: 10-01
    provides: "Signal-exclusion decision (option-b), release/nightly workflows and their published artifact set"
  - phase: 10-02
    provides: "docs/plugins/ per-plugin pages, docs/plugin-contract.md link targets"
  - phase: 10-03
    provides: "SECURITY.md, web/README.md, docs/ss/README.md's four-slot indexed screenshot convention"
  - phase: 10-04
    provides: "docs/releasing.md"
provides:
  - "CONTRIBUTING.md — repository layout, dev loop, testing gates, build/release targets, cross-links"
  - "README.md rewritten for a new user: screenshots, current status, install (prebuilt + from-source), configure, run, credits"
  - "scripts/check-doc-links.sh — repo-wide relative-markdown-link resolution guard, wired into make docs-check and CI"
affects: []

actuals:
  tokens: 7511
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "docs-check runs first in ci.yml's test job — cheapest gate, fails fast, before any npm/go step"
    - "Indexed-screenshot exception carved into the link guard by convention (docs/ss/N.png), not a special-cased skip list"

key-files:
  created:
    - CONTRIBUTING.md
    - scripts/check-doc-links.sh
  modified:
    - README.md
    - Makefile
    - .github/workflows/ci.yml

key-decisions:
  - "Removed the dead ./scripts/run-with-env.sh reference from README's Configure/Build sections entirely (script was deleted in ae8c1b3, quick task 260805-o5d/cleanup) rather than inventing a replacement wrapper — plain export/source instructions are sufficient and don't point at a file that doesn't exist"
  - "Module count in CONTRIBUTING.md's Repository layout corrected to nine workspace modules (root, sdk, paperless, silverbullet, proton, signal, whatsapp, mock, mockstrict) — the pre-existing README text was stale at seven/five plugins"
  - "check-doc-links.sh's indexed-screenshot exception matches on docs/ss/*.png with a numeric basename (regex ^[0-9]+\\.png$), not a hardcoded 1-4 list, so a future fifth screenshot slot doesn't require a script edit"

patterns-established:
  - "Doc-link guard extraction: grep -noE for [text](target)/![alt](target), sed to isolate line:target, skip http(s)/mailto/pure-fragment, strip trailing #fragment, resolve relative to the containing file's own directory"

requirements-completed: [SC-1]

coverage:
  - id: D1
    description: "CONTRIBUTING.md carries repository layout, dev loop, and testing content with full operational detail (port guard rationale, absolute-plugins-dir staleness trap, test/test-portable cgo split, DEV_* overrides, Playwright standing convention), removed from README.md rather than duplicated"
    requirement: "SC-1"
    verification:
      - kind: other
        ref: "grep checks for DEV_PORT/DEV_HOST/DEV_READY_TIMEOUT/make test/make e2e/make dev-check/go.work in CONTRIBUTING.md (all present) and their absence from README.md (all absent) — run live this session"
        status: pass
      - kind: other
        ref: "make test-portable — full pass, 9 modules, run live after Task 1"
        status: pass
    human_judgment: false
  - id: D2
    description: "README.md rewritten for a new user: four indexed screenshot slots before any status prose, current (11-phase, five-source) status replacing stale eight-phase/Phase-8-upcoming text, an Install section naming real prebuilt artifacts and the Signal-exclusion decision, a corrected Configure section with no dead script reference, the security-boundary paragraph preserved verbatim in substance, cross-links to every other Phase 10 doc, and a Credits section citing Claude and openGSD with a live-verified URL"
    requirement: "SC-1"
    verification:
      - kind: other
        ref: "Every automated <verify> grep in 10-05-PLAN.md Task 2 run live and passing: screenshot indices, cross-links, security-boundary phrasing (127.0.0.1/loopback + no authentication — reworded to keep the phrase on one grep-able line), Claude/openGSD credits, checksums.txt + whatsapp mentions, zero stale eight-phase/Phase-8-upcoming hits, zero dangling scripts/*.sh references"
        status: pass
      - kind: other
        ref: "curl -sfSL against https://github.com/open-gsd/gsd-core (200) and https://github.com/davison/topos/releases/latest (200) — run live this session"
        status: pass
    human_judgment: false
  - id: D3
    description: "scripts/check-doc-links.sh exists, is executable, passes bash -n, opens with an explanatory header documenting the external-URL exclusion and the docs/ss/ indexed-screenshot exception; make docs-check passes against the repo as this phase leaves it (35 links, 18 files); the guard genuinely fails on an injected broken link (proven live, then reverted); it reports every failure with file:line, not just the first; it prints a files/links-checked summary on success; Makefile's docs-check target is in .PHONY; ci.yml runs make docs-check as a plain run: step with zero new uses: actions (count unchanged at 5), placed before every npm/go step"
    requirement: "SC-1"
    verification:
      - kind: other
        ref: "bash -n scripts/check-doc-links.sh; make docs-check (pass, 35 links/18 files); live negative-control injection into docs/plugins/README.md caught and reported by file:line, then git checkout -- reverted; grep -c 'uses:' .github/workflows/ci.yml == 5 — all run live this session"
        status: pass
      - kind: other
        ref: "make test-portable — full pass after Task 3, confirming the doc-only change didn't touch source"
        status: pass
    human_judgment: false

duration: ~35min
completed: 2026-08-12
status: complete
---

# Phase 10 Plan 5: README/CONTRIBUTING Split and Doc-Link Guard Summary

**Split the README in two (new-user README.md, contributor CONTRIBUTING.md), rewrote README.md around four screenshot slots and the project's real current state, and wired a repo-wide relative-link guard into `make docs-check` and CI so a broken doc link now fails a push.**

## Performance

- **Duration:** ~35 min
- **Started:** 2026-08-12
- **Completed:** 2026-08-12
- **Tasks:** 3
- **Files modified:** 5 (2 created, 3 modified)

## Accomplishments

- Created `CONTRIBUTING.md`, moving the Repository layout, Development loop, and Testing sections out of `README.md` with their operational detail intact (dev port guard rationale, absolute-`[plugins] dir` staleness trap, the `test`/`test-portable` cgo split, all three `DEV_*` overrides, the Playwright standing convention) and adding a Building and releasing section pointing at `docs/releasing.md`.
- Rewrote `README.md` end to end for a new visitor: four indexed screenshot slots (`docs/ss/1.png`-`4.png`, matching `docs/ss/README.md`'s locked descriptions) immediately after the opening paragraphs, a current Status section (11 phases complete, all five sources shipping including WhatsApp) replacing the stale eight-phase/"Phase 8 coming next" text, a new Install section documenting both the prebuilt-download path (naming the real release artifacts and `checksums.txt`, and stating the Signal-binary exclusion per Plan 10-01's option-b decision) and the from-source path, a corrected Configure section (real env vars for all four network-backed sources, the dead `./scripts/run-with-env.sh` reference removed since that script was deleted in `ae8c1b3`), the security-boundary paragraph preserved verbatim in substance, cross-links to every other doc this phase produced, and a Credits section for Claude/Anthropic and openGSD with the URL verified live (200).
- Added `scripts/check-doc-links.sh`: enumerates the repo-root, `docs/`, `web/`, and `plugins/` markdown files (excluding `.planning/` and `node_modules`), extracts every markdown link/image target, skips external URLs/mailto/pure-fragment anchors, resolves the rest relative to the containing file's own directory, and fails — reporting every broken target's file and line, not just the first — with one deliberate exception for `docs/ss/*.png` indexed screenshots that haven't been dropped in yet. Wired into a new `docs-check` Makefile target and as the first, fail-fast step of `ci.yml`'s `test` job (a plain `run:` step — the workflow's `uses:` action count stays at 5).
- Verified the guard genuinely fails: injected a broken link into `docs/plugins/README.md`, confirmed `make docs-check` exits non-zero and names the exact file/line, then reverted the injection. All 35 real links across 18 tracked markdown files resolve clean.

## Task Commits

Each task was committed atomically:

1. **Task 1: CONTRIBUTING.md — move the contributor half out of the README** - `6ef378e` (feat)
2. **Task 2: Rewrite README.md for a new user** - `7c561b2` (feat)
3. **Task 3: Doc-link guard, wired into the gate set and into CI** - `75f3b5c` (feat)

**Plan metadata:** this commit (SUMMARY.md — worktree mode, orchestrator owns STATE.md/ROADMAP.md)

## Files Created/Modified

- `CONTRIBUTING.md` - New: repository layout, dev loop, testing gates, build/release targets, cross-links — the contributor-facing half of the old README
- `README.md` - Rewritten: screenshots, current status, install (prebuilt + from source), configure, run, where-to-look-next, credits
- `scripts/check-doc-links.sh` - New: repo-wide relative-markdown-link resolution guard
- `Makefile` - Added `docs-check` target (and to `.PHONY`)
- `.github/workflows/ci.yml` - Added `make docs-check` as the first step of the `test` job

## Decisions Made

- Removed the dead `./scripts/run-with-env.sh` reference outright rather than inventing a replacement wrapper — the script was deleted in commit `ae8c1b3` (part of an earlier working-tree cleanup), and plain `export`/`source .env` instructions are sufficient without pointing at a file that no longer exists.
- Corrected the module count in `CONTRIBUTING.md`'s Repository layout to nine workspace modules (root, `sdk`, and seven plugins — `paperless`, `silverbullet`, `proton`, `signal`, `whatsapp`, `mock`, `mockstrict`) — the pre-existing README text was stale at "seven modules"/five named plugins from before WhatsApp (Phase 8) and the `mockstrict` fixture (Phase 07.1) existed.
- `check-doc-links.sh`'s indexed-screenshot exception matches any `docs/ss/*.png` with a purely numeric basename (`^[0-9]+\.png$`), not a hardcoded four-item list — a future fifth screenshot slot needs no script change.

## Deviations from Plan

None — plan executed as written. The only judgment call was the exact wording fix in the Run section's security-boundary paragraph (rewrapping so "no authentication" stays on one physical line rather than splitting across the markdown line-wrap) so the plan's own line-based grep verification could actually see the phrase; the paragraph's substance is unchanged from the pre-existing README text.

## Issues Encountered

None.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- This is the last plan in Phase 10 (wave 3, depends on all four prior plans). With 10-01 through 10-04, the phase goal is met end to end: a new user can understand, install, and configure topos from the docs alone, contributor material has its own home, security reporting has a home, every plugin has a consistent one-page doc, GitHub milestones have a committed sync mechanism, and CI publishes change-gated nightlies and release artifacts — with every link between those documents machine-verified.
- Open item carried from 10-01 (not this plan's scope): `.github/workflows/nightly.yml`'s live two-dispatch change-gate proof is still pending human/orchestrator action once this worktree branch is integrated with `origin/main` (tracked in `.planning/WINDOWS.md` entry #6).
- No blockers for phase completion from this plan.

## Self-Check: PASSED

- FOUND: `CONTRIBUTING.md`
- FOUND: `README.md`
- FOUND: `scripts/check-doc-links.sh`
- FOUND: `Makefile` (docs-check target + .PHONY)
- FOUND: `.github/workflows/ci.yml` (make docs-check step)
- FOUND commit: `6ef378e` (Task 1)
- FOUND commit: `7c561b2` (Task 2)
- FOUND commit: `75f3b5c` (Task 3)

---
*Phase: 10-docs-and-release-readiness*
*Completed: 2026-08-12*
