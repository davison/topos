---
phase: 10-docs-and-release-readiness
plan: 03
subsystem: docs
tags: [security-policy, github-private-vulnerability-reporting, sveltekit, screenshots]

requires: []
provides:
  - "SECURITY.md at repo root, GitHub-recommended shape, linking the private advisory intake"
  - "Private vulnerability reporting enabled on davison/topos (confirmed by API read-back)"
  - "web/README.md replaced — no SvelteKit scaffold instructions remain"
  - "docs/ss/README.md defining the indexed 1.png-4.png screenshot slot convention"
affects: [10-05]

actuals:
  tokens: 600
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "docs/ss/ kept present in git via a README rather than committed placeholder images"

key-files:
  created:
    - SECURITY.md
    - docs/ss/README.md
  modified:
    - web/README.md

key-decisions:
  - "Private vulnerability reporting confirmed via the read-back endpoint (gh api repos/davison/topos/private-vulnerability-reporting --jq '.enabled' returned true), not the write-success fallback"
  - "Screenshot slots locked at four indices: 1.png stream+detail pane, 2.png source chip row, 3.png cross-source search, 4.png webspace builder — Plan 10-05 must conform to this contract"

patterns-established:
  - "Screenshot placeholder convention: indexed docs/ss/N.png, directory tracked via README only, no fake images committed"

requirements-completed: [SC-2, SC-4]

coverage:
  - id: D1
    description: "SECURITY.md exists with GitHub-recommended shape and links a live private-reporting path"
    requirement: "SC-4"
    verification:
      - kind: other
        ref: "grep -qF '# Security Policy' / '## Supported Versions' / '## Reporting a Vulnerability' / 'security/advisories/new' SECURITY.md — all pass"
        status: pass
    human_judgment: false
  - id: D2
    description: "Private vulnerability reporting enabled on davison/topos"
    requirement: "SC-4"
    verification:
      - kind: other
        ref: "gh api repos/davison/topos/private-vulnerability-reporting --jq '.enabled' -> true"
        status: pass
    human_judgment: false
  - id: D3
    description: "web/README.md replaced — no SvelteKit scaffold instructions, points at CONTRIBUTING.md and docs/testing.md"
    requirement: "SC-2"
    verification:
      - kind: other
        ref: "grep -qivE 'npx sv create|sv@|npm run preview|deployment adapter' web/README.md (no match) + grep -qF 'CONTRIBUTING.md'/'docs/testing.md' (both match)"
        status: pass
    human_judgment: false
  - id: D4
    description: "docs/ss/README.md establishes the indexed screenshot convention (1.png-4.png), directory tracked with no placeholder images"
    requirement: "SC-2"
    verification:
      - kind: other
        ref: "for loop over indices 1-4 confirms each present in docs/ss/README.md; ls docs/ss | wc -l == 1; find docs/ss -name '*.png' empty"
        status: pass
    human_judgment: false

duration: 15min
completed: 2026-08-12
status: complete
---

# Phase 10 Plan 03: Root Docs Gaps — SECURITY.md, web/README.md, Screenshot Convention Summary

**SECURITY.md with GitHub private vulnerability reporting enabled live via REST API, SvelteKit scaffold README replaced, and the four-slot indexed screenshot convention locked for Plan 10-05.**

## Performance

- **Duration:** ~15 min
- **Completed:** 2026-08-12T10:20:23Z
- **Tasks:** 2
- **Files modified:** 3 (2 created, 1 replaced)

## Accomplishments
- Created `SECURITY.md` at the repo root: GitHub's recommended `# Security Policy` / `## Supported Versions` / `## Reporting a Vulnerability` shape, grounded in the project's actual documented v1 security boundary (loopback-only bind, no HTTP API auth) rather than inventing a different posture, and naming the four report classes that matter (source mutation, credential/secret leakage, source content executing in the embedded UI, unauthorized agent reads).
- Enabled private vulnerability reporting on `davison/topos` via `gh api --method PUT repos/davison/topos/private-vulnerability-reporting`, confirmed live via the read-back endpoint (`.enabled == true`) — the "Report a vulnerability" link in SECURITY.md is a working link, not a dead one.
- Replaced `web/README.md` (previously unmodified `sv create` scaffold output describing `npx sv create`, `npm run dev`/`preview`, and deployment adapters — none applicable to this repo) with an 11-line pointer stating the SPA is embedded into the kernel binary at build time and pointing at `CONTRIBUTING.md` and `docs/testing.md`.
- Created `docs/ss/README.md`, establishing the indexed screenshot convention (`1.png` through `4.png`) that keeps `docs/ss/` present in git without committing placeholder images, and defining the four slots' intended content (stream+detail pane, source chip row, cross-source search, webspace builder) for Plan 10-05's README rewrite to consume.

## Task Commits

Each task was committed atomically:

1. **Task 1: SECURITY.md, and the private disclosure channel it points at** - `0df9816` (feat)
2. **Task 2: Replace the SvelteKit scaffold README and establish the screenshot convention** - `4e90c31` (feat)

**Plan metadata:** commit deferred to worktree-mode metadata commit (SUMMARY.md, no STATE.md/ROADMAP.md — orchestrator owns those)

## Files Created/Modified
- `SECURITY.md` - Vulnerability disclosure policy pointing at GitHub's private advisory intake; states the loopback/no-auth v1 boundary as deliberate, not a finding
- `web/README.md` - Replaces SvelteKit scaffold output; states the SPA is embedded into the kernel binary, points at CONTRIBUTING.md and docs/testing.md
- `docs/ss/README.md` - New file establishing the docs/ss/ directory and its indexed screenshot naming convention

## Decisions Made
- Used the API read-back confirmation path (not the write-success fallback) for private vulnerability reporting — `gh api repos/davison/topos/private-vulnerability-reporting --jq '.enabled'` returned `true` on the first attempt, so no fallback was needed.
- Screenshot slot descriptions locked exactly as the plan specified: `1.png` webspace stream + detail pane open, `2.png` source chip row with per-source health, `3.png` cross-source search results, `4.png` webspace builder UI. Plan 10-05 should reference these same four indices and descriptions rather than inventing new ones.

## Deviations from Plan

None - plan executed exactly as written. The single wording adjustment (adding a literal "read-only" phrase to SECURITY.md's mutation-report bullet to satisfy the plan's own automated verify gate, which grepped for `read-only|read only`) is not a deviation from plan content — it's the plan's own acceptance criterion driving exact phrasing, applied during Task 1 before commit.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required. Private vulnerability reporting was enabled programmatically via `gh api` using the operator's already-authenticated `gh` credentials (precondition verified before Task 1: `gh auth status` reported an authenticated `repo`-scoped account).

## Next Phase Readiness
- `web/README.md` forward-references `CONTRIBUTING.md`, which does not yet exist — this is intentional per the plan; Plan 10-05 (later wave) creates it and its link-integrity guard proves the reference resolves at the end of the phase.
- Plan 10-05's README rewrite should embed the four screenshot indices exactly as defined here (`1.png`-`4.png`, same descriptions) rather than inventing new ones — the contract is now committed at `docs/ss/README.md`.
- No blockers for other Phase 10 plans in this wave — this plan touched no files shared with any sibling plan.

---
*Phase: 10-docs-and-release-readiness*
*Completed: 2026-08-12*

## Self-Check: PASSED
