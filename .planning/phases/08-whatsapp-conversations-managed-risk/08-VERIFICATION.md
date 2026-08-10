---
phase: 08-whatsapp-conversations-managed-risk
verified: 2026-08-10T20:40:00Z
status: human_needed
score: 5/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 4/5
  gaps_closed:
    - "CR-01: QRPanel.svelte's beginSession() now calls cancelWhatsAppLink(session.session) in its retired branch instead of discarding the just-returned session id — orphaned subprocess/suspended-instance window closed"
    - "WR-01: AddSourceModal.svelte's handleConnectNext now clears linkNotice before its own missingRequiredFields( call, so a stale declined-link notice can no longer co-render with a genuine connection-failure alert"
    - "IN-01: the e2e case-2 setup comment no longer references the deleted POLL_FLOOR_MS mechanism"
  gaps_remaining: []
  regressions: []
---

# Phase 8: WhatsApp Conversations (Managed Risk) Verification Report

**Phase Goal:** User's WhatsApp groups for a topic appear in the webspace stream via a linked-device session, and everything else keeps working when that session breaks
**Verified:** 2026-08-10T20:40:00Z
**Status:** human_needed
**Re-verification:** Yes — after gap-closure plan 08-08 (closing 08-REVIEW.md's CR-01, WR-01, IN-01), which the prior 08-VERIFICATION.md (`status: gaps_found`, 4/5) promoted from blocking findings.

## Context

The prior verification (`previous_status: gaps_found`, 4/5) found that `08-REVIEW.md`'s CR-01 — a race in `QRPanel.svelte`'s `beginSession()` where a component torn down while its `POST /api/config/whatsapp-link` was still in flight discarded the just-returned session id instead of cancelling it, orphaning a live subprocess (and, on Re-link, a suspended source instance) for up to five minutes — was real, unresolved, and directly undermined the phase goal's "everything else keeps working" clause. It also flagged two lower-severity findings from the same review, WR-01 (a stale declined-link notice that could co-render with a later connection-failure alert) and IN-01 (a stale comment).

Plan `08-08` closed all three. This re-verification independently re-reads the fixed code, re-runs every test suite live (not on SUMMARY's word), and runs a **fresh code review of the 08-08 diff itself** (`08-REVIEW.md`, committed at `d8b9918`) to check the fix didn't introduce its own new problems. That fresh review found 0 critical findings, 1 warning (WR-02: the terminal `paired`/`error`/`timeout` branches of `applySession` don't clear `sessionId`, so a later unmount/Retry/Restart can issue one redundant, harmless 404-ing cancel — explicitly not a blocker per the review's own analysis), and 1 info (IN-02: a second, non-literal echo of the deleted `POLL_FLOOR_MS` wording survives in case 1's setup comment, missed by the gap-closure's literal grep check). Both are confirmed present and both are cosmetic/non-blocking, not truth failures.

All CR-01/WR-01/IN-01 closures are confirmed correct and independently re-verified below. However, this pass also confirms **two human-verification items remain genuinely outstanding** — one carried forward from the prior verification (real-device re-test of the G-08-1 fix) and one newly required by 08-08-PLAN.md's own Task 1 `<human-check>` (real-kernel confirmation that CR-01's fix actually releases a real subprocess and a real suspended instance, and that six rapid open/close cycles don't exhaust the 4-slot concurrency cap) — because the e2e harness intercepts the WhatsApp link HTTP routes and never spawns a real `topos-plugin-whatsapp` subprocess (see the spec file's own hermeticity note), so no automated evidence in this repo can close that specific claim. Per the decision tree, outstanding human-verification items route this report to `human_needed`, not `passed`.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | User can link webspaces as a WhatsApp device by scanning a QR code, and the session survives service restarts without re-linking | ✓ VERIFIED (regression, unchanged since prior verification) | `link.go`, `connect.go`, `pairwait.go`, `plugin.go` untouched by 08-08. Full `plugins/whatsapp` suite re-run live: `ok github.com/davison/topos/plugins/whatsapp 0.053s`, including `TestPairLoginWaiter_*`, `TestStoreLock_*`. |
| 2 | Messages from WhatsApp groups whose names match the webspace's matching config appear in the stream alongside every other source, using the Phase-4 chat rendering | ✓ VERIFIED (regression, unchanged) | `render.go`, `digest.go`, `plugin.go`'s `Match`/`Eligible` logic untouched by 08-08. `TestMatch_ExactCaseInsensitiveOnly`, `TestEligible_*` re-run live, pass. |
| 3 | The plugin persists its own message store, so conversations captured while it was running stay browsable regardless of what the WhatsApp desktop app retains | ✓ VERIFIED (regression, unchanged) | `messagestore.go` untouched by 08-08. `TestMessageStore_AppendIdempotent`, `TestMessageStore_ChatIsolationAndOrdering` re-run live, pass. |
| 4 | De-link, ban, or session expiry surfaces as an explicit plugin-health error in the UI while previously captured messages remain browsable and every other source is unaffected | ✓ VERIFIED (regression, unchanged) | `health.go` untouched by 08-08; full kernel/httpapi suite re-run live, passes. |
| 5 | Closing the Add-Source/Re-link dialog does not leave an orphaned link subprocess or an orphaned suspended source instance behind (CR-01 — the "everything else keeps working" clause's own resource pool) | ✓ VERIFIED | `QRPanel.svelte`'s `beginSession()` retired-branch now reads `void cancelWhatsAppLink(session.session).catch(() => {});` with no `sessionId` assignment (read in full at HEAD). New structural guard (`qr-panel.test.ts`, 4 assertions) proves the branch's shape by regex against the extracted source. New e2e case 12 (`web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts:771`) delays the start response 1500ms, presses Escape while it's still in flight, and asserts `deletedSessionIds === ['sess-inflight']`, `deleteCalls === 1`, `pollCalls === 0` — re-run live in this session, passes in 2.4s. **Caveat:** this is hermetic route-layer proof only (the harness never spawns a real `topos-plugin-whatsapp` subprocess); the plan's own `verification: backstop` truth and Task 1's `<human-check>` require a real-kernel confirmation that a real subprocess exits and a real suspended instance resumes — that check is still outstanding (see Human Verification Required). |

**Score:** 5/5 truths verified by codebase evidence; 1 of those (#5) has an outstanding real-kernel human-verification requirement that automated evidence cannot close, which is why overall status is `human_needed` rather than `passed`.

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `web/src/lib/components/QRPanel.svelte` | `beginSession`'s post-resolution cancel closing CR-01 | ✓ VERIFIED | Read in full at HEAD; matches 08-08-PLAN's stated shape exactly (`cancelWhatsAppLink(session.session)`, no `sessionId` assignment, `.catch(` swallow) |
| `web/src/lib/components/qr-panel.test.ts` | Structural guard over that branch | ✓ VERIFIED | `describe('beginSession cancels a session it created if torn down before learning its id (08-REVIEW.md CR-01)', ...)` — 4 assertions, all pass live |
| `web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts` | Cases 12 (teardown during in-flight start) and 13 (no stale notice beside a failure alert); corrected case-2 comment | ✓ VERIFIED | Both cases present and pass live; `grep -c 'POLL_FLOOR_MS'` on this file returns 0 |
| `web/src/lib/components/AddSourceModal.svelte` | `handleConnectNext` clears `linkNotice` before its `missingRequiredFields(` call | ✓ VERIFIED | Read in full; clear statement precedes the check at lines 262-286 |
| `web/src/lib/components/add-source.test.ts` | Guard asserting the clear happens, and happens before the missing-field check | ✓ VERIFIED | `handleConnectNext clears linkNotice, strictly before its own missingRequiredFields( call...` — index-comparison assertion, passes live |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `QRPanel.svelte`'s `beginSession()` (in-flight start, torn down before resolution) | `cancelWhatsAppLink` | direct call in the retired branch | ✓ WIRED | Was ✗ NOT WIRED in the prior verification; now confirmed wired by source read, structural guard, and live e2e case 12 |
| `AddSourceModal.svelte`'s `handleConnectNext` | `linkNotice` clear | assignment as first statement after the initial guard, before `missingRequiredFields(` | ✓ WIRED | Was ⚠️ incompletely reset in the prior verification (only cleared by `handleLinkCancelled`/`resetFlowState`/`selectPluginType`, not `handleConnectNext`); now confirmed cleared there too, live e2e case 13 passes |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Full Go workspace build + test | `CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go test ./...` | all packages `ok` | ✓ PASS |
| whatsapp plugin full suite | `CGO_ENABLED=0 go test ./plugins/whatsapp/... -v` | all tests pass | ✓ PASS |
| Web unit suite: qr-panel + add-source (targeted) | `npm run test -- --run src/lib/components/qr-panel.test.ts src/lib/components/add-source.test.ts` | 72/72 pass (up from 67 in the prior verification — +5 new CR-01/WR-01 guards) | ✓ PASS |
| Full web unit suite (single run) | `npm run test -- --run` | 667/667 pass, 38/38 files | ✓ PASS |
| svelte-check | `npm run check` | 838 files, 0 errors, 9 pre-existing warnings unrelated to this phase | ✓ PASS |
| e2e typecheck | `npm run check:e2e` | 0 errors | ✓ PASS |
| Playwright whatsapp-qr-link spec, real build | `make e2e E2E_ARGS="--grep 'whatsapp'"` | **13/13 pass** (up from 11/11 — cases 12, 13 new), including case 12 (2.4s) and case 13 (555ms) | ✓ PASS |
| Debt-marker scan on all 5 files touched by 08-08 | `grep -n -E "TBD\|FIXME\|XXX\|TODO\|HACK\|PLACEHOLDER"` per file | no matches | ✓ PASS |
| Prohibition check: no console/localStorage writes of session/device data in QRPanel.svelte | `grep -n "console\.\|localStorage" web/src/lib/components/QRPanel.svelte` | no matches | ✓ PASS |
| Prohibition check: concurrency cap/reaper deadline unwidened | `grep -n "maxConcurrentLinkSessions\|linkSessionDeadline" kernel/httpapi/*.go` | still `4` and `5 * time.Minute`, unchanged | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes declared by this phase's plans or ROADMAP success criteria. Step 7c: SKIPPED (no declared probes).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| SRC-03 | 08-01 … 08-08 | WhatsApp plugin runs as a whatsmeow linked device with its own persistent message store; degrades gracefully on de-link/ban; matches on group names | ? NEEDS HUMAN | All automated evidence for SRC-03's four success criteria and the CR-01/WR-01/IN-01 closures is in place and passing. The remaining gap between this and full SATISFIED status is the two outstanding real-kernel human-verification items below — `.planning/REQUIREMENTS.md` line 30 correctly still shows `[ ]` pending this verification's routing to `human_needed`, not an omission. |

No orphaned requirements — SRC-03 is the only ID `.planning/ROADMAP.md` maps to Phase 8, and all eight plans (including 08-08) declare `requirements: [SRC-03]`.

### Anti-Patterns Found

No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers in any of the 5 files touched by 08-08 (checked individually, all clean).

The fresh code review of the 08-08 diff (`08-REVIEW.md`, committed `d8b9918`) found two non-blocking issues, confirmed present independently in this session and left unfixed by 08-08 (out of that plan's declared surgical scope):

| File | Line(s) | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `web/src/lib/components/QRPanel.svelte` | `applySession`'s `paired`/`error`/`timeout` cases (~175-191) | None of the three terminal branches clears `sessionId` back to `null`, so `onDestroy`'s unconditional `retireSession()` call re-issues a cancel for an already-kernel-retired session on every successful pairing, Retry, and Restart | ⚠️ Warning | Confirmed by reading the code: the terminal cases set `retired = true` and call `clearTimers()` but never `sessionId = null`. Produces one extra, harmless 404-ing `DELETE` per successful link/retry/restart — no data loss, no security impact, review's own explicit verdict is "not currently... a BLOCKER." Not counted against the score. |
| `web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts` | ~319-324 (case 1 setup comment) | "a real, floored delay" / "that floor" still describes the deleted `POLL_FLOOR_MS` mechanism in different words, missed by the gap-closure's literal `grep -c 'POLL_FLOOR_MS'` check | ℹ️ Info | Confirmed present at those exact lines. Cosmetic only — misleads a future reader, no functional impact. |

Both are recorded here per the mandate to check the codebase directly rather than take either review's word; neither is a truth-blocking finding and neither reopens CR-01/WR-01/IN-01.

### Human Verification Required

1. **Real-device re-test of UAT test 1 (carried forward from the prior verification, still outstanding).**
   **Test:** `make dev`, Add Source → New WhatsApp… with a display name and the seeded local path; scan the QR with a real phone and accept the pairing; observe the panel.
   **Expected:** The panel leaves the QR state within a few seconds of the phone reporting success ("Scan accepted — completing login…", then the linked confirmation, then Step 2); it must not sit on a code with a ticking countdown. Restart the kernel and confirm it reconnects with no second QR.
   **Why human:** Requires a live WhatsApp account and a real kernel run — inherently outside this automated environment.

2. **Real-kernel confirmation of the CR-01 fix (08-08-PLAN.md Task 1's own deferred `<human-check>`, not yet performed).**
   **Test:** `make dev`, open the SPA, start Add Source → New WhatsApp… with a display name and the seeded local path, click Next, then press Escape immediately — inside the window where the panel still shows its skeleton and no QR has appeared. Check the kernel log: the link session should be cancelled within about a second, not reaped roughly five minutes later. Repeat five times in quick succession, then start one more link attempt and let it reach a QR code — that final attempt must reach the QR state normally, not be rejected for exceeding the 4-slot concurrent link-session cap.
   **Why human:** The e2e harness intercepts the WhatsApp link HTTP routes at the network layer and never spawns a real `topos-plugin-whatsapp` subprocess (the spec file's own hermeticity note documents this structural exclusion) — so no automated test in this repo can observe a real subprocess actually exiting or a real suspended source instance actually resuming. This is exactly what the phase's `must_haves.truths` entry marked `verification: backstop` in `08-08-PLAN.md` requires and defers.

### Gaps Summary

No blocking gaps. All three findings the prior verification promoted (CR-01, WR-01, IN-01) are closed, independently confirmed by direct code reading, structural unit guards, and live-rerun Playwright cases in this session — not taken on SUMMARY.md's word. A fresh code review of the 08-08 diff surfaced two further findings (WR-02, IN-02); both are confirmed present and both are non-blocking by their own review's explicit classification (no data loss, no security impact, cosmetic-only for IN-02).

The remaining item preventing a clean `passed` status is not a code defect but a verification-coverage gap: two human-verification items — one carried forward (real-device G-08-1 re-test) and one newly required by 08-08-PLAN's own Task 1 (real-kernel CR-01 confirmation) — are still outstanding because the e2e harness structurally cannot spawn a real WhatsApp plugin subprocess. Both should be performed and their outcomes recorded before this phase is considered fully closed.

---

_Verified: 2026-08-10T20:40:00Z_
_Verifier: Claude (gsd-verifier)_
