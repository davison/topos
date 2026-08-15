---
gsd_state_version: 1.0
milestone: v1.1.0
milestone_name: Plugin Ecosystem
current_phase: 14
current_phase_name: google-drive-source-built-out-of-repo
status: executing
stopped_at: Phase 14 UI-SPEC approved
last_updated: "2026-08-15T22:26:26.327Z"
last_activity: 2026-08-15
last_activity_desc: Phase 13 complete, transitioned to Phase 14
progress:
  total_phases: 4
  completed_phases: 3
  total_plans: 31
  completed_plans: 26
  percent: 75
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-15 after Phase 13 completion)

**Core value:** Open one webspace and instantly see and grok all related information across every silo — without visiting each data store individually.
**Current focus:** Phase 14 — google-drive-source-built-out-of-repo

## Current Position

Phase: 14 (google-drive-source-built-out-of-repo) — EXECUTING
Plan: 1 of 5
Status: Executing Phase 14
Progress: [████████████████████] 26/26 plans (100%) — 3/4 phases complete
Last activity: 2026-08-15 — Phase 14 execution started

## Performance Metrics

**Velocity:**

- Total plans completed: 118
- Average duration: —
- Total execution time: —

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 | 6 | - | - |
| 02 | 6 | - | - |
| 03 | 10 | - | - |
| 04 | 4 | - | - |
| 05 | 5 | - | - |
| 06 | 8 | - | - |
| 07 | 16 | - | - |
| 07.1 | 6 | - | - |
| 08 | 15 | - | - |
| 09 | 7 | - | - |
| 09.1 | 4 | - | - |
| 10 | 5 | - | - |
| 11 | 7 | - | - |
| 12 | 11 | - | - |
| 13 | 8 | - | - |

**Recent Trend:**

- Last 5 plans: —
- Trend: —

*Updated after each plan completion*
| Phase 01 P01 | 2h44m | 3 tasks | 90 files |
**Per-Plan Metrics:**

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| Phase 01 P02 | 40min | 2 tasks | 17 files |
| Phase 01 P03 | 25min | 2 tasks | 14 files |
| Phase 01 P04 | 47min | 2 tasks | 11 files |
| Phase 01 P05 | 4min | 3 tasks | 2 files |
| Phase 01 P06 | 15min | 3 tasks | 6 files |
| Phase 02 P01 | 68min | 3 tasks | 34 files |
| Phase 02 P02 | 35min | 3 tasks | 22 files |
| Phase 02 P03 | 55min | 3 tasks | 15 files |
| Phase 02 P04 | 50min | 3 tasks | 21 files |
| Phase 02 P05 | 5min | 2 tasks | 2 files |
| Phase 02 P06 | 15min | 2 tasks | 3 files |
| Phase 03 P05 | 21min | 3 tasks | 4 files |
| Phase 03 P06 | 20min | 2 tasks | 6 files |
| Phase 04 P01 | 3h | 2 tasks | 20 files |
| Phase 04 P02 | 2h | 3 tasks | 18 files |
| Phase 04 P03 | ~2.5h | 3 tasks | 13 files |
| Phase 260805-kt3 P01 | ~20min | 1 tasks | 3 files |
| Phase 05 P01 | 40min | 3 tasks | 35 files |
| Phase 05 P02 | ~2h | 3 tasks | 22 files |
| Phase 05 P03 | 55min | 2 tasks | 10 files |
| Phase 05 P04 | ~20min | 2 tasks | 22 files |
| Phase 05 P05 | 35min | 3 tasks | 5 files |
| Phase 06 P02 | ~20min | 3 tasks | 13 files |
| Phase 06 P03 | 20min | 3 tasks | 7 files |
| Phase 06 P04 | ~12min | 1 tasks | 3 files |
| Phase 06 P05 | ~15min | 3 tasks | 8 files |
| Phase 06 P06 | ~15min | 2 tasks | 4 files |
| Phase 06 P07 | 32min | 3 tasks | 7 files |
| Phase 06 P08 | 13min | 2 tasks | 4 files |
| Phase 07 P01 | 52min | 3 tasks | 31 files |
| Phase 07 P02 | ~2h | 3 tasks | 23 files |
| Phase 07 P03 | ~8min | 3 tasks | 32 files |
| Phase 07 P04 | ~1h | 3 tasks | 16 files |
| Phase 07 P05 | ~16 min | 3 tasks | 17 files |
| Phase 07 P06 | ~15min | 2 tasks | 4 files |
| Phase 07 P07 | 22min | 2 tasks | 2 files |
| Phase 07 P08 | ~20min | 2 tasks | 5 files |
| Phase 07 P09 | 17min | 2 tasks | 2 files |
| Phase 07 P10 | ~15min | 2 tasks | 2 files |
| Phase 07 P11 | 45min | 3 tasks | 14 files |
| Phase 07 P13 | ~18min | 3 tasks | 9 files |
| Phase 07 P12 | 10min | 2 tasks | 5 files |
| Phase 07 P14 | 8min | 3 tasks | 7 files |
| Phase 07 P15 | ~35min | 2 tasks | 13 files |
| Phase 07 P16 | ~55min | 3 tasks | 6 files |
| Phase 12 P07 | ~35min | 3 tasks | 12 files |

## Accumulated Context

### Roadmap Evolution

v1.0's evolution history (phase insertions 5–7, 07.1, 09.1; the WhatsApp shift to Phase 8; Phases 9–10 additions) is archived in `milestones/v1.0-ROADMAP.md` and summarized in MILESTONES.md.

- 2026-08-12 — v1.1.0 roadmap created: 4 phases (11-14), numbering continued from v1.0's Phase 10. Coarse granularity compressed the research summary's five suggested phases into four — per-item marks and PWA merged into a single web-app phase (Phase 13) since both are independent, UI-surface work. One hard dependency only: Phase 11 (external loading) → Phase 14 (Google Drive). Phase 12's filesystem plugin binary carries the external-path rehearsal criterion, the research-recommended checkpoint before Phase 14's OAuth/API work.

### Decisions

v1.0 decision log archived: full table in PROJECT.md Key Decisions; per-plan decision detail in milestones/v1.0-phases/*/[0-9]*-SUMMARY.md.

- [Phase 11] Trust tier derived solely from which directory a binary resolves from at launch; never from plugin self-description or config.
- [Phase 11] External-tier launches gated on SHA-256 content pins recomputed from disk before every exec; pin mismatch is a soft per-instance failure (chip shows named cause, two-click re-pin; kernel boot and unrelated saves unaffected).
- [Phase 11] Plugin subprocess environment built from an explicit allowlist + only the instance's own ${VAR} references — kernel environment never inherited wholesale.
- [Phase 11] Source.Plugin confined to bare filenames end-to-end (ResolveBinary first-statement guard + config.Validate twin) after CR-01 gap closure.
- [Phase 12] Fetch classifies through the instance's own scope (newScope(p.extras).includes) instead of a second, weaker classify() rule — closes the false 404 for an include_glob-admitted item outside the default extension allowlist.
- [Phase 12] A malformed operator glob at Fetch time maps to codes.Unavailable (not codes.Internal), matching Match's own answer for the identical pattern class.
- [Phase 12] Fetch and the kernel open route read/exec the filepath.EvalSymlinks-resolved path the containment check approved, not the lexical path it validated; the residual final-component TOCTOU window is documented as 'narrows but does not eliminate.'
- [Phase 12] Match values are exact literals, never globs; "everything from this instance" is expressed via the root's own base name, and a zero-match state across a healthy sync surfaces as a kernel-composed `last_notice` advisory (never plugin text).
- [Phase 12] UI surfaces consult one precedence chain: the chip tooltip defers to `isAdvisoryOnly` in format.ts (re-asks `healthTone` with the notice stripped) rather than re-deriving its own gate — CR-01's fix pattern for any future health-adjacent surface.
- [Phase 12] Launch-failed source entries deliberately carry no `last_notice` (matches docs/api.md contract; pinned by Go test, WR-01).
- [Phase 13] Per-item marks live in the kernel index as user-owned data: they survive re-sync, restart, and index rebuild, and always outrank automatic match rules.
- [Phase 13] Undo toast closures act on the webspace where the toast was created, never the currently-viewed one (13-07 gap closure).
- [Phase 13] `load()` rejects stale navigation generations before any side effect — the `loadState = 'loading'` write sits behind the generation guard, so a stale-gen call is a true no-op (13-08 / G-13-1; pinned by a RED-first browser spec asserting on the rendered stream).
- [Phase 13] PWA never-stale update flow: ServiceWorker update surfaces through the shared toast layer; the installed app always lands on the new build after a kernel upgrade.

### Pending Todos

[From .planning/todos/pending/ — ideas captured during sessions]

- 2026-08-05 — Signal schema-version verify-and-accept tooling (minor, tooling)
- 2026-08-05 — Centralize rendition theming (and sanitization) in the kernel content boundary (major, api; schedule with Phase 5 contract work or ecosystem milestone)
- 2026-08-13 — Plugin trust tier is directory-location, not provenance (major, security; Phase 11 debt — consent flow bypassable via config edit, file drop, or D-11 shadowing; candidate own hardening phase)

### Blockers/Concerns

- ✅ [Phase 10, RESOLVED — risk accepted 2026-08-12] UF-10-01 (10-SECURITY.md): `docs/ss/1.png` and `2.png` show some of the operator's own personal data on the public repo. Operator reviewed the screenshots at creation time, confirms they were composed carefully and nothing of concern is exposed, and explicitly accepts the (permanent) exposure. No remediation planned; do not re-flag.
- ⚠️ [Pre-existing, surfaced during Phase 10] Flaky CI: `kernel/httpapi` `TestExecLinkSpawner_StreamsLinesInOrderAndExitsClean` + `_KillTerminatesLongRunningSubprocess` fail intermittently on GitHub runners (subprocess yields no lines / kill doesn't terminate), pass locally. Route to `/gsd-debug "flaky ExecLinkSpawner tests in kernel/httpapi on CI"`.
- ⚠️ [Phase 10, advisory] 10-REVIEW.md warnings open: web/README.md non-link references, duplicated release-asset lists across workflows, test-portable CGO_ENABLED scoping. Fix via `/gsd-code-review 10 --fix`.
- ⚠️ [Phase 6] 06-REVIEW.md WR-01 still open (advisory): client-side `highlightText` in `web/src/lib/format.ts` bulk-lowercases then indexes positionally — highlight spans mis-position after case-fold-expanding characters (e.g. İ). Narrow, untested edge case; fix via `/gsd-code-review 6 --fix`.
- ⚠️ [Phase 8, ongoing operational risk] WhatsApp linked-device route can still be de-linked or banned by Meta at any time; plugin degrades honestly (named health states, captured rows survive) but there is no recovery beyond re-linking.

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260729-p2n | create a wrapper script that exposes the env vars in .env to the webspaces binary | 2026-07-29 | 7becca1 | [260729-p2n-create-a-wrapper-script-that-exposes-the](./quick/260729-p2n-create-a-wrapper-script-that-exposes-the/) |
| 260805-irt | fix pane flex: stream pane fixed width, detail pane flexes on viewport resize | 2026-08-05 | 4e51006 | [260805-irt-fix-pane-flex-stream-pane-fixed-width-de](./quick/260805-irt-fix-pane-flex-stream-pane-fixed-width-de/) |
| 260805-j98 | style scrollbars: thin, theme-matched app-wide (incl. rendition iframes) | 2026-08-05 | 2604de1 | [260805-j98-style-scrollbars-thin-theme-matched-app-](./quick/260805-j98-style-scrollbars-thin-theme-matched-app-/) |
| 260805-kt3 | narrow Proton deep-link search with sender+date criteria (live-approved) | 2026-08-05 | 1fb1fa6 | [260805-kt3-narrow-proton-deep-link-search-for-gener](./quick/260805-kt3-narrow-proton-deep-link-search-for-gener/) |
| 260805-lry | accept Signal schema 1740 after live read-set verification (source recovered) | 2026-08-05 | 9f000c3 | [260805-lry-accept-signal-desktop-schema-version-174](./quick/260805-lry-accept-signal-desktop-schema-version-174/) |
| 260805-o5d | harden make dev: plugins rebuilt as prerequisite, fail loudly on port squat/dead kernel | 2026-08-05 | 6d0e6a8 | [260805-o5d-harden-make-dev-rebuild-plugin-binaries-](./quick/260805-o5d-harden-make-dev-rebuild-plugin-binaries-/) |
| 260806-0v2 | rename application webspaces → topos (contract spine, build surface, paths, live install; checkpoint closed 2026-08-12 via sustained live use) | 2026-08-06 | 21cf179, 1117b71 | [260806-0v2-rename-application-webspaces-to-topos-ac](./quick/260806-0v2-rename-application-webspaces-to-topos-ac/) |
| 260806-f1 | (fast) refresh README: 8-phase roadmap, five-plugin layout, seven workspace modules | 2026-08-06 | cd8ba20 | — |
| 260806-f2 | (fast) expose Vite dev server (:5173) to the tailscale network via make dev | 2026-08-06 | 54d30b4 | — |
| 260811-r5d | exclude topos-plugin-mockstrict from the picker catalog (kernel exclusion + e2e injection helper + docs correction) | 2026-08-11 | 160853a | [260811-r5d-exclude-topos-plugin-mockstrict-from-the](./quick/260811-r5d-exclude-topos-plugin-mockstrict-from-the/) |
| 260811-rqc | header branding lockup: app icon + topos wordmark + tagline, top-right of header, muted (human-verified live) | 2026-08-11 | 55e133e | [260811-rqc-header-branding-app-icon-topos-wordmark-](./quick/260811-rqc-header-branding-app-icon-topos-wordmark-/) |
| 11 | (fast) fix webspace page title: '{webspace} — topos' replaces pre-rename '— webspaces' (+ smoke e2e assertion) | 2026-08-11 | ce9f985 | — |
| 260814-mkl | Fix CI check:e2e failure: declare unlinkSync in e2e node:fs ambient shim | 2026-08-14 | 66d4012 | [260814-mkl-fix-ci-check-e2e-failure-declare-unlinks](./quick/260814-mkl-fix-ci-check-e2e-failure-declare-unlinks/) |

## Deferred Items

Items acknowledged and carried forward from previous milestone close:

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| todo | signal-schema-version-verify-and-accept-tooling (minor, tooling) | pending | v1.0 close, 2026-08-12 |

## Session Continuity

Last session: 2026-08-15T17:33:14.109Z
Stopped at: Phase 14 UI-SPEC approved
Resume file: /home/darren/projects/davison/topos/.planning/phases/14-google-drive-source-built-out-of-repo/14-UI-SPEC.md

## Operator Next Steps

- Plan Phase 14 with /gsd-discuss-phase 14 (or /gsd-plan-phase 14 directly)
- Open design questions to settle during phase discuss/spec (from research/SUMMARY.md "Gaps to Address"):
  - Phase 14: Google OAuth credential distribution (bring-your-own is the requirement's stated position — confirm at spec time)
