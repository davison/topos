---
phase: 08-whatsapp-conversations-managed-risk
plan: 07
subsystem: ui
tags: [svelte, sveltekit, playwright, whatsapp, gap-closure]

# Dependency graph
requires:
  - phase: 08-whatsapp-conversations-managed-risk
    provides: QRPanel.svelte's fixed-cadence poll and pairing progress state (plan 08-05), the plugin's pairing_accepted/already_linked wire emission and kernel stderr capture (plan 08-06), 08-UI-SPEC.md Amendment 2's locked declined-link notice copy (plan 08-05)
provides:
  - AddSourceModal.svelte's linkNotice — a neutral, non-Alert notice shown on the connect step after declining the QR panel, naming the Re-link recovery route
  - add-source.test.ts structural guards pinning the notice's state wiring (set on cancel, cleared on reset/reselect, never a failure signal) and exact copy
  - Three hermetic Playwright cases (9, 10, 11) closing G-08-1's structural e2e gap: qr->poll->pairing_accepted->paired at a realistic 60s first-code expiry, no-Cancel-during-pairing, and already-linked recovery
affects: []

# Actuals (#2632)
actuals:
  tokens: 3655
  tasks: 2
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Playwright poll-sequence scripting reused unchanged across new cases: scriptLinkSession's polls array is consumed in order with the last entry repeating once exhausted, letting a case 'hold' a non-terminal state for its whole duration by omitting a terminal entry"

key-files:
  created: []
  modified:
    - web/src/lib/components/AddSourceModal.svelte
    - web/src/lib/components/add-source.test.ts
    - web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts

key-decisions:
  - "linkNotice rendered as a plain muted <p> gated by its own {#if linkNotice} block, deliberately outside the existing connectError Alert and never the destructive variant — the exact structural property e2e case 8's E5 evidence (no failure alert, no Save anyway) depends on"
  - "TDD task 1 committed as two commits (test RED, feat GREEN) rather than one combined commit, matching this phase's established convention from plans 08-05/08-06 — corrected after an initial combined commit was reset and split before Task 2 began"
  - "Case 9's 60-second expires_in_seconds is the load-bearing fixture value (whatsmeow's real first-code validity window), not an arbitrary number — a smaller value would pass against the pre-fix defective cadence and prove nothing; explicit 15s assertion timeouts make a cadence regression fail loudly instead of passing slowly"

requirements-completed: [SRC-03]

coverage:
  - id: D1
    description: "Declining the QR panel leaves the user told, in neutral language, that the source can be saved now and linked later — never presented as a failed connection test"
    requirement: SRC-03
    verification:
      - kind: unit
        ref: "web/src/lib/components/add-source.test.ts#declined-link notice (Amendment 2, G-08-1): neutral, not a failure"
        status: pass
    human_judgment: false
  - id: D2
    description: "A link session that starts against a store already holding a linked device (already_linked) carries the Add-Source flow forward to the match step without ever showing a QR code"
    requirement: SRC-03
    verification:
      - kind: e2e
        ref: "web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts#11. already-linked recovery: a completed pairing is picked up, not stranded (08-UAT.md G-08-1)"
        status: pass
    human_judgment: false
  - id: D3
    description: "The suite fails if the panel's poll cadence is ever re-tied to a QR code's validity window — the qr -> poll -> pairing_accepted -> paired walk runs at a realistic 60-second first-code expiry and completes in seconds"
    requirement: SRC-03
    verification:
      - kind: e2e
        ref: "web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts#9. qr then paired via the poll loop, at a realistic first-code expiry (08-UAT.md G-08-1)"
        status: pass
    human_judgment: false
  - id: D4
    description: "No cancel affordance is reachable while a session sits in its post-pair progress (pairing_accepted) state"
    requirement: SRC-03
    verification:
      - kind: e2e
        ref: "web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts#10. no cancel affordance during the post-pair window (08-UAT.md G-08-1)"
        status: pass
    human_judgment: false
  - id: D5
    description: "The real-device WhatsApp link flow is re-tested by a human against the shipped fix, and UAT test 1's reported symptom ('modal remains on screen with the refresh counter dwindling') no longer reproduces"
    verification: []
    human_judgment: true
    rationale: "Requires a live WhatsApp account and a real kernel run — inherently outside this automated execution environment. Per workflow.human_verify_mode = end-of-phase (.planning/config.json), this plan's own <human-check> is not a mid-flight halt; it is deferred to the phase-level verifier's end-of-phase UAT consolidation (08-UAT.md)."

# Metrics
duration: ~30min
completed: 2026-08-10
status: complete
---

# Phase 08 Plan 07: Declined-Link Notice and G-08-1 Regression Armor Summary

**A neutral, non-Alert notice tells a user who declines the QR panel that their pairing options are still open, and three new hermetic Playwright cases give the qr→poll→pairing_accepted→paired sequence — the only one a real device produces — its first automated coverage at a realistic 60-second first-code expiry, closing G-08-1's remaining structural test gap.**

## Performance

- **Duration:** ~30 min
- **Tasks:** 2
- **Files modified:** 3

## Accomplishments

- `AddSourceModal.svelte` gained `linkNotice` ($state, empty-string default): `handleLinkCancelled` sets it to 08-UI-SPEC.md Amendment 2's exact frozen copy (`"Not linked yet — you can save this source now and link later from its menu (Re-link…)."`) alongside the existing return to the connect step, touching neither `describeFailed` nor `connectError` — declining a link opportunity is never treated as a failed connection test. Cleared in `resetFlowState` and `selectPluginType` so a later plugin-type selection never inherits a stale notice. Rendered as a plain muted paragraph, deliberately outside any `Alert` and never the destructive variant.
- `add-source.test.ts` gained a new describe block with 7 structural guards pinning every piece of that wiring: the state assignment, the untouched failure flags, the clear-on-reset/clear-on-reselect behavior, the outside-any-Alert rendering, and the exact copy.
- `uat-08-whatsapp-qr-link.spec.ts` gained cases 9, 10 and 11, appended to the existing `08-04` describe block using only already-existing helpers (no helper modified):
  - **Case 9** walks `qr → pairing_accepted → paired` via the poll loop with `expires_in_seconds: 60` (whatsmeow's real first-code validity window) — the exact sequence UAT test 1 reported failing, previously unexercised because the suite's own success case scripted `paired` directly into the START response. Completes in ~7s thanks to the panel's fixed 2s poll cadence (08-05); explicit 15s assertion timeouts.
  - **Case 10** proves no Cancel control is reachable while the panel holds in the `pairing_accepted` state (no terminal entry scripted; the poll helper repeats the last entry), then closes via Escape to exercise the same unmount-cancel path as case 6.
  - **Case 11** proves an `already_linked` session never renders a QR image and carries the flow forward to the match step — the recovery path for a pairing completed on a prior attempt.

## Task Commits

Each task was committed atomically:

1. **Task 1: Declining the QR panel leaves a neutral, actionable notice** - `a2d215d` (test, RED) + `01bce84` (feat, GREEN)
2. **Task 2: Playwright cases for qr→paired-via-poll, pairing-state cancel gate, and already-linked recovery** - `fc32bc0` (test)

_Task 1 is `tdd="true"`: RED (`a2d215d`) added the 7 new structural guards and confirmed they failed against the pre-existing component (linkNotice did not exist); GREEN (`01bce84`) implemented the fix and all 44 add-source.test.ts assertions pass. Task 2 is a pure e2e-spec addition with no production code change — the fix it proves against (QRPanel's fixed cadence and pairing states) was already shipped in plans 08-05/08-06._

## Files Created/Modified

- `web/src/lib/components/AddSourceModal.svelte` - `linkNotice` state, set in `handleLinkCancelled`, cleared in `resetFlowState`/`selectPluginType`, rendered as a plain muted paragraph in the connect branch
- `web/src/lib/components/add-source.test.ts` - New `declined-link notice (Amendment 2, G-08-1)` describe block, 7 assertions
- `web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts` - Cases 9, 10, 11 appended to the existing `08-04` describe block

## Decisions Made

- The notice renders inside its own `{#if linkNotice}` guard, structurally separate from the `connectError` `Alert` block — pinned by a test asserting the containing markup carries no `variant="destructive"` and no `<Alert` at all, so e2e case 8's E5 evidence keeps meaning what it says.
- Task 1's TDD commits were split into RED (`a2d215d`) and GREEN (`01bce84`) after an initial combined commit was created and then reset — matching plans 08-05/08-06's established two-commit convention for `tdd="true"` tasks in this phase, rather than deviating from it.
- Case 9's `expires_in_seconds: 60` is the load-bearing fixture value, deliberately not shortened for test speed — a smaller value would make the case pass against the pre-fix defective cadence and prove nothing about the actual defect (G-08-1's root cause).

## Deviations from Plan

None — plan executed exactly as written. The Task 1 commit-splitting (see Decisions Made) was a self-correction during execution to match this phase's own established TDD commit convention, not a deviation from the plan's instructions (the plan does not itself mandate commit granularity beyond `tdd="true"`).

## Issues Encountered

- `web/node_modules` was absent in this worktree (git worktrees don't carry `node_modules`, same as plans 08-05/08-06). Symlinked it from the main repo's `web/node_modules` after confirming `package-lock.json` is byte-identical (matching md5sum) between the two — read-only use, no lockfile modification, no `npm install`/`npm ci` run.
- `make e2e`'s own recipe begins with `npm --prefix web ci`, which would have deleted and reinstalled `node_modules` (removing the read-only symlink) — ran the recipe's remaining steps manually instead (`npm run build`, the two Go plugin builds, `CGO_ENABLED=0 go build -o bin/topos ./cmd/topos`, then `npx playwright test`) so the symlinked `node_modules` stayed untouched, with equivalent verification coverage (the whatsapp-qr-link spec ran 11/11, full suite 37/37).
- Building the SvelteKit SPA before rebuilding `bin/topos` matters: `kernel/webui/build/.gitkeep` gets overwritten by `npm run build`'s output and must be restored afterward (`git checkout -- kernel/webui/build/.gitkeep`) since it's tracked as a placeholder for the gitignored build directory — restored before this plan's commits so `git diff --stat` shows only the three files in `files_modified`.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- G-08-1's e2e coverage gap is closed: the qr→poll→pairing_accepted→paired sequence (the only one a real device produces) now has automated regression armor at a realistic 60-second first-code expiry, and the panel's no-cancel-during-pairing and already-linked-recovery behaviors are both proven hermetically.
- The Add-Source flow now tells a user who declines the QR panel exactly what their options are, closing the second half of G-08-1's reported symptom (a genuinely successful pairing being silently discarded with no recovery signal).
- **Outstanding: this plan's `<human-check>` (D5) is deferred, not skipped** — `workflow.human_verify_mode = end-of-phase` means the real-device re-test of UAT test 1 is consolidated into the phase-level verifier's end-of-phase UAT pass rather than gating this plan's own completion. The phase is NOT fully closed on G-08-1 until that human re-test confirms the reported symptom no longer reproduces on a real device; if it still reproduces, the finding returns to `.planning/debug/whatsapp-qr-link-no-success.md` rather than being recorded as a pass.
- No other blockers.

---
*Phase: 08-whatsapp-conversations-managed-risk*
*Completed: 2026-08-10*

## Self-Check: PASSED

- FOUND: .planning/phases/08-whatsapp-conversations-managed-risk/08-07-SUMMARY.md
- FOUND: web/src/lib/components/AddSourceModal.svelte
- FOUND: web/src/lib/components/add-source.test.ts
- FOUND: web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts
- FOUND: a2d215d (Task 1 RED commit)
- FOUND: 01bce84 (Task 1 GREEN commit)
- FOUND: fc32bc0 (Task 2 commit)
