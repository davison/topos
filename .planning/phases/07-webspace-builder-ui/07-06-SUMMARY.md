---
phase: 07-webspace-builder-ui
plan: 06
subsystem: ui
tags: [svelte, config-edit, security, gap-closure]

requires:
  - phase: 07-webspace-builder-ui (07-04, 07-05)
    provides: AddSourceModal.svelte's two-step new-instance flow, config-edit.ts's upsertSourceInstance, the shared save-state error contract (CONFIG_CONFLICT_MESSAGE)
provides:
  - "web/src/lib/instance-id.ts: deriveInstanceId + resolveNewInstanceId, the single instance-id derivation and collision-check site"
  - "Both AddSourceModal write paths (handleConnectNext, saveAnyway) route through resolveNewInstanceId before any config write"
  - "A structural test suite that fails if a third new-instance write path skips the guard, a second derivation site reappears, or the retry affordance is hidden on rejection"
affects: [07-verification, 07-review]

actuals:
  tokens: 4299
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "Shared pure guard module for a security-relevant write precondition, called from every write path (not duplicated inline per caller)"
    - "Structural source-scan invariant tests (comment-stripped + extractBetween + ordering/count assertions) to pin a fix against silent regression by a future third call site"

key-files:
  created:
    - web/src/lib/instance-id.ts
    - web/src/lib/instance-id.test.ts
  modified:
    - web/src/lib/components/AddSourceModal.svelte
    - web/src/lib/components/add-source.test.ts

key-decisions:
  - "resolveNewInstanceId returns a discriminated InstanceIdResult ({ ok: true; id } | { ok: false; reason; message }) rather than throwing — both call sites already branch on a boolean-ish outcome and set connectError from a message string, so a plain return value fits the existing control flow with no new exception path"
  - "saveAnyway's rejection branch does NOT clear describeFailed — a collision must leave the Save anyway action on screen so the user can correct the display name and retry without re-running the failed connection test (preserves the deliberate asymmetry with handleConnectNext, which does clear it since a Next-time validation rejection isn't a connection failure)"

patterns-established:
  - "Pattern: a security-relevant guard used by 2+ call sites in one component gets its own pure module (not a local function each caller half-remembers to call), plus a structural test proving every call site is wired to it and ordered correctly relative to the write it guards."

requirements-completed: [KERN-08, UI-12]

coverage:
  - id: D1
    description: "resolveNewInstanceId rejects a display name colliding with an existing config.sources id, with the exact message the Next path already used"
    requirement: KERN-08
    verification:
      - kind: unit
        ref: "web/src/lib/instance-id.test.ts#resolveNewInstanceId > rejects a display name that derives to an id already present in config.sources (CR-01 core scenario)"
        status: pass
      - kind: unit
        ref: "web/src/lib/instance-id.test.ts#resolveNewInstanceId > CR-01 regression: resolving the existing victim instance's OWN stored display name never returns ok"
        status: pass
    human_judgment: false
  - id: D2
    description: "saveAnyway calls resolveNewInstanceId before upsertSourceInstance, with a return between them, so a colliding name can never reach the write"
    requirement: UI-12
    verification:
      - kind: unit
        ref: "web/src/lib/components/add-source.test.ts#saveAnyway: CR-01 regression — resolveNewInstanceId guards every write > calls resolveNewInstanceId( before upsertSourceInstance(, with a return between them (CR-01)"
        status: pass
    human_judgment: false
  - id: D3
    description: "A rejected saveAnyway leaves the Save anyway action on screen (describeFailed untouched) so the user can retry with a corrected name without re-running the failed connection test"
    requirement: UI-12
    verification:
      - kind: unit
        ref: "web/src/lib/components/add-source.test.ts#invariant: every new-instance write path routes through the one shared guard > saveAnyway never clears the describe-failed flag"
        status: pass
    human_judgment: false
  - id: D4
    description: "Structural invariant: a third new-instance write path added later must extend the guard or the suite fails (single derivation site, exactly 2 upsertSourceInstance( call sites, sole guarded newInstanceId assignment)"
    verification:
      - kind: unit
        ref: "web/src/lib/components/add-source.test.ts#invariant: every new-instance write path routes through the one shared guard"
        status: pass
    human_judgment: false
  - id: D5
    description: "Live confirmation in the browser that Save anyway refuses a colliding display name and leaves the victim instance's connection/agent grants byte-identical"
    human_judgment: true
    rationale: "Requires a live kernel via make dev, real network-tab inspection (no PUT /api/config issued), and byte comparison of config.toml before/after — this is 07-VERIFICATION.md's deferred human-check, folded into the pending end-of-phase make dev walkthrough (07-05-PLAN.md's own human-check), not executable from this session."

duration: ~15min
completed: 2026-08-08
status: complete
---

# Phase 07 Plan 06: Close CR-01 — shared instance-id collision guard Summary

**Extracted the AddSourceModal instance-id derivation/collision check into `web/src/lib/instance-id.ts` and wired both `handleConnectNext` and `saveAnyway` through it, closing 07-REVIEW.md's CR-01 (saveAnyway previously skipped the guard entirely, letting a colliding display name silently overwrite another instance's connection and agent grants).**

## Performance

- **Duration:** ~15 min
- **Tasks:** 2
- **Files modified:** 4

## Accomplishments
- New `web/src/lib/instance-id.ts` exports `deriveInstanceId`, `resolveNewInstanceId`, and the `InstanceIdResult` discriminated type — the single derivation and collision-check site every new-instance write path must call before reaching `upsertSourceInstance`
- `saveAnyway` now calls `resolveNewInstanceId` first and returns on a not-ok result without writing anything or clearing `describeFailed`, preserving the retry affordance
- `handleConnectNext` refactored to call the same shared helper instead of its own inline blank/collision checks — byte-identical user-facing messages preserved
- 8 new behavioural unit tests in `instance-id.test.ts`, including a named CR-01 regression case (resolving the victim instance's own stored display name never returns `ok`)
- A structural invariant test suite in `add-source.test.ts` (Task 2) that fails if: a local `deriveInstanceId` reappears, a third `upsertSourceInstance(` call site is added without the guard, or a rejection clears the retry-affordance flag

## Task Commits

1. **Task 1: Close CR-01 — one shared id-resolution guard both AddSourceModal write paths call** - `54ba33c` (fix)
2. **Task 2: Pin the invariant so a third new-instance write path cannot reintroduce CR-01** - `2582b47` (test)

_No plan-metadata commit yet — this SUMMARY/STATE/ROADMAP update is committed separately per the executor's final_commit step._

## Files Created/Modified
- `web/src/lib/instance-id.ts` - New pure module: `deriveInstanceId` (moved verbatim from the component) and `resolveNewInstanceId` (blank/collision guard against a `KernelConfig`)
- `web/src/lib/instance-id.test.ts` - Derivation cases, resolution cases, the CR-01 named regression, and a never-mutates-input assertion
- `web/src/lib/components/AddSourceModal.svelte` - Local `deriveInstanceId` removed; `handleConnectNext` and `saveAnyway` both call `resolveNewInstanceId` from `$lib/instance-id`
- `web/src/lib/components/add-source.test.ts` - Task 1's ordering regression proof (`resolveNewInstanceId(` before `upsertSourceInstance(` with a `return` between) plus Task 2's full invariant describe block

## Decisions Made
- `resolveNewInstanceId` returns a discriminated result rather than throwing — fits the existing `connectError`/message-string control flow at both call sites with no new exception path to catch
- `saveAnyway`'s rejection branch deliberately does not touch `describeFailed` (unlike `handleConnectNext`, which does) — the asymmetry is intentional and is itself pinned by a Task 2 test

## Deviations from Plan

None — plan executed exactly as written. One test-writing correction made during self-verification (not a deviation from the PLAN.md's action text, a bug in my own first draft of the invariant test): the initial `newInstanceId` non-null-assignment count regex also matched the `let newInstanceId = $state<...>(null)` declaration line, over-counting to 3. Fixed by filtering out `$state`-declaration and plain-`null`-reset statements before counting, verified against `npm run test` going from 1 failing to 469/469 passing.

## Issues Encountered
None beyond the self-corrected test regex above.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- 07-VERIFICATION.md's single failed truth (the CR-01 clobber path) is now closed at the code level; the remaining item is the live `make dev` walkthrough already scheduled by 07-05-PLAN.md's own `<human-check>`, which this plan's own `<human-check>` extends with the exact collision scenario to exercise
- `CGO_ENABLED=0 go build ./...` and `go test ./kernel/... -count=1` both pass — confirmed no Go-side regression, as expected since this plan touches no kernel file
- Phase 07 (webspace-builder-ui) has no further open gaps from 07-REVIEW.md's Critical findings; five WARNING findings (WR-01..WR-05) remain deliberately deferred, unchanged by this plan

---
*Phase: 07-webspace-builder-ui*
*Completed: 2026-08-08*
