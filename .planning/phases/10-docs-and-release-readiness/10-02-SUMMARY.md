---
phase: 10-docs-and-release-readiness
plan: 02
subsystem: docs
tags: [markdown, plugin-docs, operator-docs]

# Dependency graph
requires: []
provides:
  - "docs/plugins/_template.md — the canonical section shape for a plugin operator page"
  - "docs/plugins/README.md — index of the five real-source plugin pages"
  - "docs/plugins/{paperless,silverbullet,proton,signal,whatsapp}.md — one operator page per real plugin"
affects: [10-05, docs-restructure]

# Actuals (#2632)
actuals:
  tokens: 3569
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Plugin operator pages: fixed four-heading shape (Install Requirements, Configuration, Gotchas, Security & Privacy Notes) plus a `Match vocabulary:` line, all derived from _template.md"
    - "Link-don't-duplicate: every plugin page points at config.example.toml's fully-commented reference block instead of reproducing it, enforced by a per-file <120-line ceiling"

key-files:
  created:
    - docs/plugins/_template.md
    - docs/plugins/README.md
    - docs/plugins/signal.md
    - docs/plugins/paperless.md
    - docs/plugins/silverbullet.md
    - docs/plugins/proton.md
    - docs/plugins/whatsapp.md
  modified: []

key-decisions:
  - "Match vocabulary lines copied verbatim from each plugin's matchVocabulary source declaration (conversations / tags / tags+pages / folders / groups+contacts) rather than paraphrased, so the page cannot silently drift from what Describe actually returns"
  - "Read-only/egress claims on each page name the actual test function (e.g. TestPluginsIssueOnlyGetRequests, TestAllowHost_PredicateTable, TestPluginIssuesNoIMAPMutatingCommands) rather than asserting read-only/egress-restricted generically"

patterns-established:
  - "A new plugin's operator page starts by copying docs/plugins/_template.md; docs/plugins/README.md states this rule and excludes the mock/mockstrict fixture plugins by name"

requirements-completed: [SC-3]

coverage:
  - id: D1
    description: "docs/plugins/_template.md establishes the four-heading shape (Install Requirements, Configuration, Gotchas, Security & Privacy Notes) every plugin page derives from"
    requirement: "SC-3"
    verification:
      - kind: other
        ref: "grep -qF heading assertions over docs/plugins/_template.md (run during Task 1 execution)"
        status: pass
    human_judgment: false
  - id: D2
    description: "docs/plugins/README.md indexes all five plugin pages, names _template.md as the starting point for a new page, and excludes mock/mockstrict"
    requirement: "SC-3"
    verification:
      - kind: other
        ref: "dangling-link scan over docs/plugins/README.md (all five link targets resolve)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Five plugin operator pages (signal, paperless, silverbullet, proton, whatsapp) each carry the canonical shape, their real declared match vocabulary, and a pointer to config.example.toml instead of a copy of it"
    requirement: "SC-3"
    verification:
      - kind: other
        ref: "per-file heading/vocabulary/reference-pointer/line-count assertions (Tasks 1-3 <verify> blocks, all passed)"
        status: pass
    human_judgment: false
  - id: D4
    description: "No credential-bearing line across docs/plugins/ is a literal; silverbullet.md/proton.md contain no certificate-verification-bypass language"
    requirement: "SC-3"
    verification:
      - kind: other
        ref: "grep -hE literal-credential scan + grep -icE insecure|skip-verify scan over docs/plugins/*.md"
        status: pass
    human_judgment: false

# Metrics
duration: ~20min
completed: 2026-08-12
status: complete
---

# Phase 10 Plan 02: Plugin Operator Documentation Summary

**Created `docs/plugins/` — a template plus one operator-facing page for each of the five real source plugins (paperless-ngx, SilverBullet, Proton Mail, Signal, WhatsApp), each pointing at `config.example.toml` and `docs/plugin-contract.md` rather than duplicating them.**

## Performance

- **Duration:** ~20 min
- **Completed:** 2026-08-12
- **Tasks:** 3
- **Files modified:** 7 (all new)

## Accomplishments
- `docs/plugins/_template.md` — the canonical fill-in-the-blanks shape (Install Requirements, Configuration, Gotchas, Security & Privacy Notes, plus a `Match vocabulary:` line) that every plugin page is derived from.
- `docs/plugins/README.md` — an index stating the operator/author audience split against `docs/plugin-contract.md`, linking all five pages, and naming `_template.md` as the starting point for a new plugin page (with `mock`/`mockstrict` explicitly excluded).
- Five plugin pages (`signal.md`, `paperless.md`, `silverbullet.md`, `proton.md`, `whatsapp.md`), each under 120 lines, each stating its plugin's real declared match vocabulary and its real operator-visible failure modes — `proton.md`'s Bridge loopback/self-signed-cert/scheme-match gotchas and `whatsapp.md`'s de-link/ban risk carry the most substance, as the plan intended.

## Task Commits

Each task was committed atomically:

1. **Task 1: The template, the index, and the page it was reverse-engineered from** - `2c3a89a` (docs)
2. **Task 2: paperless-ngx and SilverBullet pages** - `6ee81bc` (docs)
3. **Task 3: Proton Mail and WhatsApp pages** - `2dac71b` (docs)

_Note: this plan's own metadata commit (SUMMARY.md) follows this commit, per worktree execution rules — STATE.md/ROADMAP.md are updated by the orchestrator after the wave merges._

## Files Created/Modified
- `docs/plugins/_template.md` - fill-in-the-blanks shape for a new plugin page
- `docs/plugins/README.md` - index of the five plugin pages, audience split, template rule
- `docs/plugins/signal.md` - Signal operator page: cgo/sqlcipher prerequisite, 3.51.3 SQLite floor, no-credentials key resolution
- `docs/plugins/paperless.md` - paperless-ngx operator page: token auth, api_version negotiated at sync time
- `docs/plugins/silverbullet.md` - SilverBullet operator page: optional ca_cert for self-signed TLS
- `docs/plugins/proton.md` - Proton Mail operator page: Bridge scheme rule, required ca_cert, Bridge-scoped password
- `docs/plugins/whatsapp.md` - WhatsApp operator page: one-time out-of-band linking step, path isolation, de-link/ban risk

## Decisions Made
- Match vocabulary lines copied verbatim from each plugin's `matchVocabulary` source declaration (found via `grep -n matchVocabulary plugins/*/plugin.go`) rather than paraphrased.
- Read-only/egress security notes name the actual enforcing test function per plugin (`TestPluginsIssueOnlyGetRequests` shared across paperless/silverbullet via the AST scan at `plugins/paperless/readonly_test.go`; `TestPluginIssuesNoIMAPMutatingCommands` for proton; `TestReadOnly_NoSendCapableClientSelector` for WhatsApp; `byte_identical_test.go`/`readonly_test.go` for signal) rather than a generic "read-only, trust us" claim.

## Deviations from Plan

None — plan executed exactly as written. Task 1's own `<verify>` block includes a dangling-link check over `docs/plugins/README.md` that only resolves once all five pages exist (i.e., after Task 3); all three tasks' content was authored together before running any `<verify>` block, then each task's own `<verify>` was run and confirmed passing, then commits were split by each task's declared `<files>` list exactly as specified.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
`docs/plugins/` is ready for Plan 10-05's doc-link guard (`scripts/check-doc-links.sh`) to validate against. The five canonical section headings and the `Match vocabulary:` line convention are now established for any future plugin page.

---
*Phase: 10-docs-and-release-readiness*
*Completed: 2026-08-12*

## Self-Check: PASSED

All 7 created docs/plugins/ files and the SUMMARY.md itself found on disk; all 3 task commits (`2c3a89a`, `6ee81bc`, `2dac71b`) plus the SUMMARY commit found in git log.
