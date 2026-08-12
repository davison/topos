---
phase: 08-whatsapp-conversations-managed-risk
plan: 08
subsystem: ui
tags: [svelte, playwright, whatsapp-link-session, gap-closure]

# Dependency graph
requires:
  - phase: 08-05
    provides: fixed QR poll cadence, non-terminal pairing_accepted/already_linked states
  - phase: 08-07
    provides: declined-link notice (linkNotice) wiring in AddSourceModal.svelte
provides:
  - QRPanel.svelte releases a link session it created even when torn down before learning the session id (closes CR-01)
  - AddSourceModal.svelte clears the declined-link notice before a fresh trial launch (closes WR-01)
  - e2e spec's case-2 comment corrected to describe the shipped poll cadence (closes IN-01)
  - Two new hermetic Playwright cases (12, 13) and one new structural unit-test guard armoring both fixes
affects: [08-verification, 08-ship]

# Actuals (#2632)
actuals:
  tokens: 4315
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Post-resolution teardown cancel: a promise continuation guarded by `retired` releases a just-learned resource id rather than discarding it, mirroring the existing pre-resolution retireSession guard"

key-files:
  created: []
  modified:
    - web/src/lib/components/QRPanel.svelte
    - web/src/lib/components/qr-panel.test.ts
    - web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts
    - web/src/lib/components/AddSourceModal.svelte
    - web/src/lib/components/add-source.test.ts

key-decisions:
  - "No per-session token/generation counter added to beginSession — that guard is unreachable today (beginSession runs only from onMount and from Retry/Restart, both reachable only from terminal phases a start response has already produced); adding it would widen a deliberately surgical pass, per the plan's own instruction"
  - "Task 1 and Task 3 both touch web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts; committed Task 1's scriptDescribeWhatsApp/case-13 additions only after Task 3 landed, keeping each commit scoped to its own task's declared files_modified"

patterns-established: []

requirements-completed: [SRC-03]

coverage:
  - id: D1
    description: "Closing the Add-Source/Re-link dialog during the in-flight start request cancels the session the kernel already created, instead of orphaning it until the 5-minute reaper (CR-01)"
    requirement: "SRC-03"
    verification:
      - kind: unit
        ref: "web/src/lib/components/qr-panel.test.ts#beginSession cancels a session it created if torn down before learning its id (08-REVIEW.md CR-01)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts#12. teardown during the in-flight start still releases the session (08-REVIEW.md CR-01)"
        status: pass
    human_judgment: true
    rationale: "The hermetic proof (route-layer scripting) cannot observe a real kernel subprocess exiting or a real suspended source instance resuming — Task 1's own <human-check> against a real kernel is deferred per workflow.human_verify_mode = end-of-phase and was not run in this session"
  - id: D2
    description: "A fresh trial-launch failure after declining a link renders exactly one message (the destructive failure alert), never the stale declined-link notice alongside it (WR-01)"
    requirement: "SRC-03"
    verification:
      - kind: unit
        ref: "web/src/lib/components/add-source.test.ts#handleConnectNext clears linkNotice, strictly before its own missingRequiredFields( call, so a stale notice cannot survive the missing-field early return"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts#13. a fresh trial-launch failure renders no stale declined-link notice (08-REVIEW.md WR-01)"
        status: pass
    human_judgment: false
  - id: D3
    description: "Case 2's setup comment describes the shipped fixed POLL_INTERVAL_MS cadence rather than a POLL_FLOOR_MS mechanism G-08-1 already deleted (IN-01)"
    verification:
      - kind: other
        ref: "grep -c POLL_FLOOR_MS web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts -> 0"
        status: pass
    human_judgment: false

duration: ~25min
completed: 2026-08-10
status: complete
---

# Phase 08 Plan 08: WhatsApp Link Session Teardown Race + Declined-Link Notice Gap Closure Summary

**Closed CR-01 (QRPanel now cancels a WhatsApp link session it created even when torn down mid-request), WR-01 (a fresh trial-launch failure no longer co-renders with a stale declined-link notice), and IN-01 (a stale e2e comment), armored by a new structural unit guard and two new hermetic Playwright cases (12, 13).**

## Performance

- **Duration:** ~25 min
- **Completed:** 2026-08-10T19:23:00Z
- **Tasks:** 3
- **Files modified:** 5

## Accomplishments
- `QRPanel.svelte`'s `beginSession` now cancels the link session the moment it learns the id, even if the component was already `retired` (torn down) by the time the start response resolves — closing the window where the kernel's already-spawned subprocess (and, on Re-link, its already-suspended source instance) was orphaned until the 5-minute reaper, or sooner exhausted the 4-slot concurrency cap.
- `qr-panel.test.ts` gained a structural guard scoped to that exact branch, verified RED (all three new assertions failed) against a temporary reversion of the fix, then GREEN again with `QRPanel.svelte` confirmed byte-identical to its prior commit.
- `uat-08-whatsapp-qr-link.spec.ts`'s `LinkScript`/`scriptLinkSession` gained an optional start-response delay, a recorded list of cancelled session ids, and a poll-request counter; new case 12 proves the fix end-to-end against the real built kernel: exactly one cancel, for the exact session id the kernel returned, and zero polls.
- `AddSourceModal.svelte`'s `handleConnectNext` now clears `linkNotice` before its own missing-required-fields early return, so a fresh trial-launch attempt never inherits a prior decline's neutral notice; `add-source.test.ts` asserts the ordering by comparing indices, not just presence.
- New case 13 proves, in the browser, that a declined link followed by a genuine trial-launch failure renders exactly one message (the destructive alert), never the stale notice alongside it.
- Case 2's setup comment corrected to describe the shipped fixed `POLL_INTERVAL_MS` cadence instead of a `POLL_FLOOR_MS` mechanism G-08-1 already deleted.

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end — a dialog closed during the in-flight start releases the session it created** - `aa2fba1` (fix)
2. **Task 2: Structural guard so the discarded-session branch cannot silently return** - `89063c6` (test)
3. **Task 3: A fresh trial launch clears the declined-link notice (WR-01)** - `a02a7a2` (fix)

_Note: Tasks 1 and 3 both modify `web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts` (Task 1 adds case 12 and the shared LinkScript extensions; Task 3 adds case 13 and scriptDescribeWhatsApp's optional fail-after param). Each commit was scoped strictly to its own task's declared changes to that file — Task 3's additions were held back and reapplied only after Task 1's own commit and verification passed at 12/12._

## Files Created/Modified
- `web/src/lib/components/QRPanel.svelte` - `beginSession`'s post-resolution branch now cancels a session learned after teardown, without assigning `sessionId`
- `web/src/lib/components/qr-panel.test.ts` - new describe block guarding that branch structurally (4 assertions), RED-then-GREEN verified
- `web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts` - `LinkScript` extended (startDelayMs, deletedSessionIds, pollCalls), `scriptDescribeWhatsApp` extended (optional failFromCall), case 2's comment corrected, cases 12 and 13 added
- `web/src/lib/components/AddSourceModal.svelte` - `handleConnectNext` clears `linkNotice` before its missing-required-fields early return
- `web/src/lib/components/add-source.test.ts` - new assertion comparing the clear's index against the `missingRequiredFields(` call's index within `handleConnectNext`'s body

## Decisions Made
- Declined to add a per-session token/generation counter to `beginSession` (considered for guarding against a stale start resolving into a newer session) — that path is unreachable today since `beginSession` is only re-entered from `onMount` or from Retry/Restart, both of which are reachable only from terminal phases a start response has already produced. Adding it would have widened this pass beyond the plan's own surgical scope; recorded here rather than in new code, per the plan's own instruction.
- Split the two tasks' edits to the shared e2e spec file cleanly by temporarily reverting Task 3's portion (scriptDescribeWhatsApp's `opts` param and case 13) before Task 1's commit, then reapplying it for Task 3 — keeping each commit's diff scoped exactly to its own task's declared file set, consistent with the atomic-commit-per-task protocol.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None. `npm --prefix web ci` was required at session start (fresh worktree, no `node_modules`) before any `npm --prefix web run …` command would work; this is routine worktree setup, not a plan deviation.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- All three findings from `08-REVIEW.md` (CR-01, WR-01, IN-01) are closed, each with both a structural/unit guard and a hermetic Playwright case.
- Task 1's `<human-check>` (real-kernel observation of the orphaned-subprocess race closing, and that six consecutive open/close cycles don't exhaust the 4-slot concurrency cap) is deferred per `workflow.human_verify_mode = end-of-phase` and was NOT performed in this session — it remains outstanding for end-of-phase human verification.
- Full verification run in this session: `npm --prefix web run test -- --run` (667/667 unit tests), `npm --prefix web run check` (0 errors), `npm --prefix web run check:e2e` (0 errors), `make e2e` full suite (39/39, up from 37 — exactly +2 for cases 12/13), `CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go test ./...` (all green, no Go files touched).
- `git diff --stat` across this plan's three commits touches only the five files declared in `files_modified` — no lockfile, no kernel file, no plugin file.

## Self-Check: PASSED

- FOUND: web/src/lib/components/QRPanel.svelte
- FOUND: web/src/lib/components/qr-panel.test.ts
- FOUND: web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts
- FOUND: web/src/lib/components/AddSourceModal.svelte
- FOUND: web/src/lib/components/add-source.test.ts
- FOUND: .planning/phases/08-whatsapp-conversations-managed-risk/08-08-SUMMARY.md
- FOUND commit: aa2fba1 (Task 1)
- FOUND commit: 89063c6 (Task 2)
- FOUND commit: a02a7a2 (Task 3)
- FOUND commit: 244026a (SUMMARY)

---
*Phase: 08-whatsapp-conversations-managed-risk*
*Completed: 2026-08-10*
