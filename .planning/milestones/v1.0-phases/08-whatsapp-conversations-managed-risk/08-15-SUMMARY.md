---
phase: 08-whatsapp-conversations-managed-risk
plan: 15
subsystem: testing
tags: [gates, race-detector, playwright, real-device, human-verify, whatsapp]

# Dependency graph
requires:
  - phase: 08-whatsapp-conversations-managed-risk
    provides: "08-13's supervisor reader/mutation-lock split (Host()/Coordinator() off genMu, internally-synchronised pluginhost.Host) and 08-14's WhatsApp login-wait-off-the-launch-path fix, both closing gap G-08-5"
provides:
  - "All four repository gates (test-portable, kernel race suite, dev-check, Playwright e2e) confirmed green against the merged 08-13+08-14 tree, with the e2e spec count recorded unchanged (42 tests / 16 spec files) as the falsification check for both fix plans' 'no new UI-visible change' claim"
  - "A human-verified real-device WhatsApp re-link against a live account, confirming phase success criterion 4 ('every other source is unaffected') during the exact completion window that previously froze the kernel"
  - "The real-device item carried forward from the previous verification cycle (08-UAT.md) discharged: re-linked webspace loads healthy, no outage banner, no stale pairing instruction"
  - "Gap G-08-5's human half and gap G-08-4's human half both closed"
affects: [08-verification, 08-uat, phase-08-close-out]

# Actuals (#2632)
actuals:
  tokens: 0
  tasks: 2
  commits: 1

# Tech tracking
tech-stack:
  added: []
  patterns: []

key-files:
  created: []
  modified: []

key-decisions:
  - "Task 1's four gates were not re-run in this continuation session — they were run to completion by the prior executor against this exact base commit (e1144bf) earlier the same day, and are recorded here verbatim. A cheap targeted sanity check (CGO_ENABLED=1 go test ./kernel/supervisor/... -race, the package most directly touched by 08-13) was re-run in this session and passed, confirming the tree had not drifted before the SUMMARY was written."
  - "The Task 2 checkpoint (human-verify, gate=blocking) was not re-run — the human's response was already received and is transcribed verbatim into this SUMMARY per the resume instructions, not re-solicited."

patterns-established: []

requirements-completed: [SRC-03]

coverage:
  - id: D1
    description: "All four automated repository gates (make test-portable, kernel race suite, make dev-check, make e2e) are green against the merged 08-13+08-14 tree"
    requirement: SRC-03
    verification:
      - kind: unit
        ref: "make test-portable (13 Go modules, all ok)"
        status: pass
      - kind: integration
        ref: "CGO_ENABLED=1 go test ./kernel/... -count=1 -race (7 kernel packages, zero DATA RACE)"
        status: pass
      - kind: other
        ref: "make dev-check (three dev-guard smoke cases)"
        status: pass
      - kind: e2e
        ref: "make e2e (42 tests, 16 spec files, chromium, 14.1s, zero failures)"
        status: pass
    human_judgment: false
  - id: D2
    description: "A real WhatsApp re-link against a live account leaves every other configured source responsive (item-open, health chip, manual refresh) during the exact completion window that previously froze the kernel, discharging phase success criterion 4 and gap G-08-5's human half"
    verification: []
    human_judgment: true
    rationale: "No harness in this repo can drive a live QR pairing against a real WhatsApp account; docs/testing.md records this as one of three permanently manual items. Human response received and transcribed verbatim below."
  - id: D3
    description: "The real-device item carried forward from the previous verification cycle (08-UAT.md) is discharged: a genuine re-link's resulting webspace loads and settles healthy without an outage banner or the stale pairing instruction, closing gap G-08-4's human half"
    verification: []
    human_judgment: true
    rationale: "Same live-account constraint as D2 — requires a human watching a real re-link complete. Human response received and transcribed verbatim below."

duration: ~1min (continuation session; sanity check only, no gate re-run)
completed: 2026-08-11
status: complete
---

# Phase 08 Plan 15: Real-device re-link gate Summary

**All four repository gates confirmed green (test-portable, kernel race suite, dev-check, Playwright at 42 tests / 16 specs unchanged), and a human-verified real WhatsApp re-link confirmed every other source stays responsive during the completion window that previously froze the kernel — closing gap G-08-5's and G-08-4's human halves and discharging the real-device item carried forward from the previous verification cycle.**

## Performance

- **Duration:** ~1 min in this continuation session (sanity re-check only; the four gates themselves were run to completion by the prior executor against this same base commit)
- **Started:** 2026-08-11 (continuation session, after human checkpoint response received)
- **Completed:** 2026-08-11
- **Tasks:** 2 (Task 1: automated gates; Task 2: checkpoint:human-verify)
- **Files modified:** 0 (this plan modifies no source file — it is a gate, not a fix)

## Accomplishments

- Confirmed all four repository gates green against the merged 08-13+08-14 tree.
- Obtained and recorded a human's verbatim confirmation of a real WhatsApp re-link against a live account, with a second unrelated source staying responsive throughout — the one observation no automated harness in this repo can make.
- Discharged the real-device item carried forward across two prior verification cycles (08-UAT.md).
- Recorded, for the historical record, the first-attempt gate failure (a test-oracle defect, not a production bug) and its resolution.

## Task 1: Automated gates (executed by prior executor, this session, base e1144bf)

Run in the order specified by the plan, against the working tree with both 08-13 and 08-14 merged:

1. **`make test-portable`** — exit 0. All 13 Go modules report `ok`, zero failures.
2. **Race gate.** The plan's literal `CGO_ENABLED=0 go test ./kernel/... -count=1 -race` is a toolchain contradiction — Go's `-race` flag itself requires cgo to build the race-instrumented binary, independent of whether the kernel's own dependency set uses cgo (it does not). Ran `CGO_ENABLED=1 go test ./kernel/... -count=1 -race` instead — exit 0, all 7 kernel packages report `ok`, zero `DATA RACE`. Additionally stress-ran the previously-flaky `TestApply_MidFlightSyncLeavesNoStrandedRunningRow` in isolation with `-count=10 -race`: 10/10 PASS.
3. **`make dev-check`** — exit 0; all three dev-guard smoke cases pass.
4. **`make e2e`** — exit 0. **42 tests passed across 16 spec files**, chromium, 14.1s, zero failures, zero flakes. Spec-file count is unchanged from before this wave (the last spec-touching commit is 672fba2, from plan 08-10, predating both 08-13 and 08-14) — this falsifiably confirms both fix plans' recorded "no new Playwright spec needed" decision, since neither plan changed a component, route, response shape, or user-visible copy.

**Result:** all four gates green. `git status --porcelain` clean apart from planning artifacts — this plan modified no source file.

### First-attempt history (for the record)

An earlier run of this gate plan on 2026-08-11 halted at gate 2 when `TestApply_MidFlightSyncLeavesNoStrandedRunningRow` failed 3/10 under `-count=10 -race`. Root cause: the test asserted through the `MAX(id)` `LatestSyncRunPerSource` aggregate while `Apply`'s failure branch legitimately inserts a second running row — a test-oracle defect present since phase 07 (commit f25c4ab), not a production bug; the 08-13 `genMu` refactor was investigated and exonerated. Fixed in commit 998a9ab (oracle retargeted via a new `SyncRunsForSourceForTesting` reader plus `t.Cleanup(s.Shutdown)`); full debug record at `.planning/debug/resolved/apply-midflight-sync-race.md`, knowledge-base entry recorded against commit e1144bf. Gates were then re-run green, as recorded above.

## Task 2: Real-device re-link (checkpoint:human-verify, gate=blocking)

**Human response, received 2026-08-11, verbatim:**

> "aspproved - everything works as stated in steps 1-5, including no stray plugin process"

This confirms, per the checkpoint's five numbered steps in `08-15-PLAN.md`:

1. **Boot latency (WR-01).** Kernel start with an already-linked WhatsApp source configured feels normal — no multi-second stall before the SPA is reachable.
2. **The main check — every other source is unaffected (criterion 4).** During the exact window a live re-link's pairing was accepted and completed, the second tab's item-open, health chip, and manual refresh for an unrelated source all answered promptly — no visible hang.
3. **The carried-forward item.** The re-linked WhatsApp webspace loaded normally, with no outage banner and no stale pairing instruction for the now-paired device; the source settled to healthy with its stream populated within a few seconds.
4. **Restart.** Kernel restart reconnected the WhatsApp source with no second QR prompt; the other source was reachable immediately.
5. **Repeat (idempotency).** A second re-link in the same session left exactly one WhatsApp source chip and no stray `topos-plugin-whatsapp` process.

**Disposition:** approved — a full pass, no reproduced symptom. Per the plan's own prohibition list and threat T-08-44, this is recorded as a pass because the human explicitly confirmed all five steps behaved as expected; no failure was reported that would require routing to a new gap instead.

This discharges:
- Phase success criterion 4's "every other source is unaffected" clause.
- The real-device item carried forward from the previous verification cycle (`08-UAT.md`).
- Gap G-08-5's human half (the observational counterpart to Task 1's automated gates).
- Gap G-08-4's human half (the stale-pairing-instruction / outage-banner regression check).

**Privacy check (T-08-43):** the recorded response above contains no message content, contact or group name, phone number, or QR payload — only a pass/fail-style confirmation of the five documented steps, consistent with the checkpoint's own instructions on what may be reported.

## Task Commits

This plan modifies no source file; Task 1 and Task 2 are gate-execution and human-verification tasks with no per-task code commit. The only commit produced by this plan is the metadata commit carrying this SUMMARY.

**Plan metadata:** (this commit) `docs(08-15): complete real-device gate — four gates green, human re-link check approved`

## Files Created/Modified
None. This plan modifies no source file — it is a gate, not a fix, per its own objective and prohibitions.

## Decisions Made
- Did not re-run Task 1's four gates in this continuation session; they were run to completion by the prior executor against this exact base commit (e1144bf) earlier the same day, and a targeted sanity check (`CGO_ENABLED=1 go test ./kernel/supervisor/... -count=1 -race`, the package most directly touched by 08-13) was re-run instead and passed, confirming no drift before this SUMMARY was written.
- Did not re-run Task 2's human checkpoint — the human's response was already received; it is transcribed verbatim above per the resume instructions rather than re-solicited.

## Deviations from Plan

None in this continuation session — plan executed exactly as specified for finalization (verify prior state cheaply, write and commit the SUMMARY, defer STATE.md/ROADMAP.md updates to the orchestrator).

### Auto-fixed Issues (carried forward from Task 1's original execution, for completeness)

**1. [Rule 3 - Blocking] `-race` requires `CGO_ENABLED=1`, not `=0` as the plan's Task 1 `<verify>` literally states**
- **Found during:** Task 1, gate 2
- **Issue:** The plan's `<verify>` block specifies `CGO_ENABLED=0 go test ./kernel/... -count=1 -race`. Running the `-race` leg with `CGO_ENABLED=0` fails immediately: `go: -race requires cgo; enable cgo by setting CGO_ENABLED=1` — a Go toolchain constraint, not a project or plan error.
- **Fix:** Ran the race leg with `CGO_ENABLED=1` instead, as documented in 08-14-SUMMARY.md's precedent for the same toolchain constraint. Kernel module's own dependency set is unchanged by this (no go.mod/go.sum diff).
- **Files modified:** none (verification-command adjustment only)
- **Verification:** `CGO_ENABLED=1 go test ./kernel/... -count=1 -race` passes, zero DATA RACE.
- **Committed in:** n/a (no code change)

**2. [Rule 1 - Bug] Flaky test oracle in `TestApply_MidFlightSyncLeavesNoStrandedRunningRow`**
- **Found during:** Task 1, gate 2 (first attempt, stress run)
- **Issue:** Test failed 3/10 under `-count=10 -race` due to a test-oracle defect (asserting through a `MAX(id)` aggregate that legitimately admits a second running row on `Apply`'s failure branch), pre-existing since phase 07 — not a production bug and not caused by 08-13's `genMu` refactor.
- **Fix:** Retargeted the oracle to a new `SyncRunsForSourceForTesting` reader and added `t.Cleanup(s.Shutdown)`, in commit 998a9ab (separate from this plan; already landed in this plan's base commit e1144bf).
- **Files modified:** (landed prior to this plan; see `.planning/debug/resolved/apply-midflight-sync-race.md`)
- **Verification:** 10/10 PASS on isolated stress re-run after the fix.
- **Committed in:** 998a9ab (prior to this plan's base commit)

---

**Total deviations:** 2 auto-fixed, both carried forward from the original Task 1 execution (1 blocking toolchain constraint, 1 bug fix already landed in the base commit). No deviation occurred in this finalization session itself.
**Impact on plan:** No scope creep; this plan touches no source file in either the original execution or this finalization.

## Issues Encountered
None beyond the toolchain constraint and pre-existing test-oracle defect documented above, both resolved before this SUMMARY was written.

## User Setup Required
None beyond the plan's own `user_setup` block (phone with the WhatsApp account to hand, one other healthy non-WhatsApp source configured) — already satisfied by the human who ran and approved Task 2's checkpoint.

## Next Phase Readiness
- Gap G-08-5 (kernel/plugin launch-latency freeze) is fully closed — both its automated proof (08-13, 08-14, and this plan's Task 1 gates) and its human observational half (this plan's Task 2).
- Gap G-08-4 (stale pairing instruction / false outage state after re-link) is fully closed — its human half discharged in Task 2.
- The real-device item carried forward from the previous verification cycle (`08-UAT.md`) is discharged, not carried forward a third time.
- All four repository gates are green with the e2e spec count confirmed unchanged, closing out this gap-closure wave's verification loop.
- This SUMMARY does not update STATE.md, ROADMAP.md, or REQUIREMENTS.md — per this session's explicit instruction, the orchestrator owns those writes.

## Self-Check: PASSED

Verified before commit:
- `.planning/phases/08-whatsapp-conversations-managed-risk/08-15-PLAN.md` exists and was read.
- `.planning/phases/08-whatsapp-conversations-managed-risk/08-14-SUMMARY.md` exists and was read (format precedent, toolchain-constraint precedent).
- Base commit e1144bf971e940c8a4ef45c096c361a3e2fdcf5c confirmed via `git log --oneline -5` (chain 998a9ab..e1144bf present).
- Sanity check `CGO_ENABLED=1 go test ./kernel/supervisor/... -count=1 -race` re-run in this session: exit 0, `ok`.
- `git status --porcelain` clean before writing this SUMMARY — no source file modified.
- This SUMMARY reviewed against T-08-43's disclosure list before commit: no message content, contact/group name, phone number, or QR payload present.

---
*Phase: 08-whatsapp-conversations-managed-risk*
*Completed: 2026-08-11*
