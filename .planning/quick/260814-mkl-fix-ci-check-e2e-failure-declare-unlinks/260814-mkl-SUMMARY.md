---
phase: quick-260814-mkl
plan: 01
subsystem: testing
tags: [typescript, ambient-types, e2e, playwright, node-fs]

requires: []
provides:
  - "unlinkSync ambient declaration in web/e2e/e2e-builtins.d.ts's declare module 'node:fs' block"
affects: [e2e, ci]

actuals:
  tokens: 330
  tasks: 1
  commits: 1

tech-stack:
  added: []
  patterns:
    - "e2e ambient shim: one narrowly-typed declaration per call site, with a provenance comment naming the originating plan"

key-files:
  created: []
  modified:
    - web/e2e/e2e-builtins.d.ts

key-decisions:
  - "Declared unlinkSync(path: string): void — matching the single call shape at web/e2e/specs/12-filesystem-recursion.spec.ts:126 exactly, no overloads or PathLike/URL variants."
  - "Did not install @types/node — kept the fix inside the existing hand-rolled ambient shim, preserving the package-legitimacy gate scoped in 07.1-01-SUMMARY.md."

patterns-established: []

requirements-completed: [QUICK-260814-mkl]

coverage:
  - id: D1
    description: "npm --prefix web run check:e2e (CI's failing job) now exits 0 with no diagnostics"
    requirement: QUICK-260814-mkl
    verification:
      - kind: other
        ref: "npm --prefix web run check:e2e"
        status: pass
    human_judgment: false

duration: 5min
completed: 2026-08-14
status: complete
---

# Quick Task 260814-mkl: Fix CI check:e2e failure Summary

**Added a single `unlinkSync(path: string): void` declaration to the e2e tree's hand-rolled `node:fs` ambient shim, unblocking `check:e2e`'s TS2305 failure without installing `@types/node` or touching the spec it serves.**

## Performance

- **Duration:** ~5 min
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments
- `npm --prefix web run check:e2e` now exits 0 (previously failed with `TS2305: Module '"node:fs"' has no exported member 'unlinkSync'` at `e2e/specs/12-filesystem-recursion.spec.ts(14,42)`)
- `web/e2e/e2e-builtins.d.ts`'s `declare module 'node:fs'` block now declares `unlinkSync`, matching the file's established "narrow declaration per call site, with provenance comment" discipline

## Task Commits

Each task was committed atomically:

1. **Task 1: Declare unlinkSync in the e2e node:fs ambient shim** - `66d4012` (fix)

## Files Created/Modified
- `web/e2e/e2e-builtins.d.ts` - added `export function unlinkSync(path: string): void;` plus a provenance comment inside the `declare module 'node:fs'` block

## Decisions Made
- Signature narrowed to exactly the one observed call shape (single string arg, discarded void return) — no `PathLike`/`URL` union, no options bag, no second overload.
- No package installed; `web/package.json` and `web/package-lock.json` are untouched.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
None.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- CI's `check:e2e` job is green again; no follow-on work required.
- The ambient shim file's ordering/style is preserved for future phase-scoped additions to follow the same pattern.

## Self-Check: PASSED
- FOUND: web/e2e/e2e-builtins.d.ts (unlinkSync declaration present)
- FOUND: 66d4012 (commit exists in git log)
- `npm --prefix web run check:e2e` exits 0
- `git diff --name-only -- web/` returns exactly `web/e2e/e2e-builtins.d.ts`

---
*Quick task: 260814-mkl*
*Completed: 2026-08-14*
