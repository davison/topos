---
phase: 16-provenance-based-plugin-trust
plan: 07
subsystem: docs
tags: [documentation, plugin-trust, provenance, gap-closure]

# Dependency graph
requires:
  - phase: 16-provenance-based-plugin-trust (plan 05)
    provides: docs/plugin-trust.md (canonical trust document) and the shipped D-11 evidence-based EvaluateTrust model
provides:
  - docs/plugin-contract.md's "Trust tiers" section corrected to state the shipped evidence-based trust model
  - Repo-wide sweep of docs/plugin-contract.md removing every residual directory-derived trust claim
affects: [17-plugin-repo-split, plugin-contract-third-party-authors]

# Actuals (#2632)
actuals:
  tokens: 3200
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Trust-touching docs defer to docs/plugin-trust.md as canonical rather than restating the model (D-11 intent)"

key-files:
  created: []
  modified:
    - docs/plugin-contract.md

key-decisions:
  - "Preserved the '## Trust tiers' heading text byte-identical so docs/plugins/signal.md's #trust-tiers anchor and docs/api.md's two references keep resolving"
  - "Deferred to docs/plugin-trust.md for the full model instead of restating the two evidence arms in detail a second time"

patterns-established:
  - "Negative-grep-gated documentation rewrites: acceptance criteria assert both presence of corrected language and absence of every superseded claim by literal string, catching partial corrections"

requirements-completed: [TRUST-01, TRUST-02, TRUST-03, TRUST-04]

coverage:
  - id: D1
    description: "docs/plugin-contract.md's Trust tiers section states both directories are pure search paths and tier is decided per binary by pluginhost.EvaluateTrust from provenance evidence, matching the shipped D-11 model"
    requirement: "TRUST-01"
    verification:
      - kind: other
        ref: "grep -c 'pure search paths' docs/plugin-contract.md -> 1; grep -c 'EvaluateTrust' docs/plugin-contract.md -> 1; grep -cE 'derived exclusively from which directory|the trusted directory always wins|A binary resolved from here is' docs/plugin-contract.md -> 0"
        status: pass
    human_judgment: false
  - id: D2
    description: "The collision paragraph states the shipped rule: evidence-carrying candidate wins, trusted-first order breaks neither/both ties, tamper refusal never falls back to the other copy"
    requirement: "TRUST-04"
    verification:
      - kind: other
        ref: "manual diff read of the rewritten collision paragraph against kernel/pluginhost/discover_binaries.go's resolveBinaryDetailed collision branch (lines 495-527)"
        status: pass
    human_judgment: true
    rationale: "Verifying prose accurately restates code precedence is a judgment call the plan itself required be confirmed by reading the diff and quoting the paragraph (see below), not a mechanical grep"
  - id: D3
    description: "The honest-limits (integrity control, not publisher authentication) and publisher-authentication paragraphs survive the rewrite byte-intact"
    requirement: "TRUST-02"
    verification:
      - kind: other
        ref: "git diff -- docs/plugin-contract.md shows no changes to the 'A link-time manifest match is an integrity control...' / 'Publisher authentication DOES exist...' paragraphs"
        status: pass
    human_judgment: false
  - id: D4
    description: "The external-tier consent-and-pin path still reads as the fully supported way to run an unsigned binary, not a degraded path; the Pinning section (unchanged) and the deferral paragraphs preserve this framing"
    requirement: "TRUST-03"
    verification:
      - kind: other
        ref: "git status --porcelain confirms docs/plugin-contract.md is the only modified file; Pinning section (line 339+) untouched by this plan"
        status: pass
    human_judgment: false
  - id: D5
    description: "Every cross-reference into the Trust tiers section still resolves (docs/plugins/signal.md#trust-tiers anchor, docs/api.md's two references) and make docs-check / scripts/check-doc-links.sh pass"
    verification:
      - kind: other
        ref: "./scripts/check-doc-links.sh -> checked 59 link(s) across 21 file(s) — all resolve; make docs-check exits 0"
        status: pass
    human_judgment: false

duration: ~20min
completed: 2026-08-20
status: complete
---

# Phase 16 Plan 07: Trust tiers documentation gap closure Summary

**Rewrote `docs/plugin-contract.md`'s "Trust tiers" section and swept four other regions in the same file to state the shipped evidence-based (D-11) trust model instead of the pre-Phase-16 directory-derived one, closing 16-VERIFICATION.md gap 2 / 16-REVIEW.md CR-02.**

## Performance

- **Duration:** ~20 min
- **Tasks:** 2
- **Files modified:** 1 (`docs/plugin-contract.md`)

## Accomplishments

- The "Trust tiers" section now states both plugin directories are pure search paths, that tier is decided per binary by `pluginhost.EvaluateTrust` from provenance evidence wherever the binary sits, and closes with a deferral link to `docs/plugin-trust.md` as the authoritative model
- The collision paragraph states the shipped rule accurately: whichever candidate carries valid evidence wins; if neither or both do, the trusted-first search order decides; a tamper refusal never falls back to the other copy
- Swept the four remaining residual regions carrying the same superseded directory-derived-trust premise: the "Discovery and launch" preamble's dangling forward-reference and trust clause, the build-provenance paragraph's per-directory candidacy clause and link-time-only launch description, the bare-filename paragraph's rationale, and the two-instances paragraph's parenthetical
- Every still-true operational detail preserved: default per-OS external directory paths, `cmd/topos-manifest` hashing + `-ldflags -X` linkage, `manifest_unverified` launch failure, the add-source picker's trial-launch gate, verification-never-demotes-and-runs, the `launch_advisory: "shadowed"` fact, the bare-filename rule itself, and the honest-limits ("integrity control, not publisher authentication") and publisher-authentication paragraphs (byte-intact)

## Rewritten collision paragraph (quoted per acceptance criteria)

> **The collision rule.** If a binary of the same filename exists in BOTH directories, the kernel evaluates both candidates and whichever carries valid evidence wins; if neither carries evidence, or both do, the existing trusted-first search order decides which copy launches. A candidate that a manifest positively names with a digest that no longer matches what's on disk is a tamper refusal — that resolves to the refusal itself and never falls back to launching the other copy instead. A binary can never impersonate a trusted plugin merely by choosing a colliding filename in the external directory; only carrying (or lacking) verifiable evidence decides the winner, and every collision is logged by name at the launch-time call site. When the trusted-first tiebreak resolves a collision, `GET /api/sources` also carries a `launch_advisory: "shadowed"` on that instance's own entry (`13-05-PLAN.md`, `D-14`) — a structured, UI-visible fact, not only a log line, so an operator can see that the plugin they separately consented to pin is not the one actually running.

This names all three outcomes the plan's acceptance criteria required: evidence-carrying candidate wins, trusted-first order breaks the neither/both case, and tamper refusal never falls back — matching `resolveBinaryDetailed`'s actual precedence in `kernel/pluginhost/discover_binaries.go` (lines 495-527).

## Task Commits

Each task was committed atomically:

1. **Task 1: Rewrite the "Trust tiers" section to the shipped evidence-based model** - `0409758` (docs)
2. **Task 2: Sweep the residual directory-derived claims elsewhere in the same file** - `908d9e4` (docs)

_Note: this is a tracer-typed task (Task 1) followed by an auto-typed sweep (Task 2); both landed real, complete edits — no throwaway or placeholder content._

## Files Created/Modified

- `docs/plugin-contract.md` - "Trust tiers" section rewritten (heading through collision paragraph); four further regions (Discovery-and-launch preamble, build-provenance paragraph, bare-filename paragraph, two-instances paragraph) corrected to remove the same superseded premise

## Decisions Made

- Kept the `## Trust tiers` heading text byte-identical (not renamed) so `docs/plugins/signal.md`'s `#trust-tiers` anchor and `docs/api.md`'s two references keep resolving, per the plan's explicit constraint
- Chose to defer to `docs/plugin-trust.md` for the full evidence-source detail rather than re-explaining both arms in `plugin-contract.md`, honoring `docs/plugin-trust.md`'s own stated intent that every other trust-touching document links back rather than restating the model
- In the build-provenance paragraph, generalized "trusts NOTHING in the trusted directory" (which was directory-scoped) to "trusts nothing through that [link-time] arm" plus an explicit note that the signed-release-manifest arm can still earn trust independently — this keeps the sentence honest under D-11's two-arm model without contradicting the still-true fact that a bare `go build` carries no link-time manifest

## Deviations from Plan

None - plan executed exactly as written. Line numbers for the heading (206) and Pinning heading (321) matched the plan's read_first citations exactly before any edit, confirming the plan was authored against the current tree.

## Issues Encountered

One self-correction during Task 1: the first draft of the opening sentence wrapped "pure search paths" across a markdown line break ("pure search\npaths"), which failed the `grep -c 'pure search paths'` verification gate (grep does not match across lines). Reworded to keep the phrase on one line; re-ran verification and it passed. No commit was made with the broken wrapping — caught before commit.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 16-VERIFICATION.md gap 2 is closed: `docs/plugin-contract.md` and `docs/plugin-trust.md` now agree — neither asserts a per-directory tier, both describe the same two evidence arms, and only `docs/plugin-trust.md` states the model in full
- `make docs-check` and `scripts/check-doc-links.sh` pass; no source, test, or build recipe file was touched (`git diff --stat -- kernel/ web/src/ web/e2e/ scripts/ Makefile` is empty)
- Two advisory findings from 16-REVIEW.md remain intentionally out of this plan's scope: WR-02 (`scripts/install.sh` marking provenance data files executable) and IN-01 (collision-fallback log wording) — both are recorded as follow-ups for `/gsd-code-review 16 --fix`, not verification gaps

---
*Phase: 16-provenance-based-plugin-trust*
*Completed: 2026-08-20*
