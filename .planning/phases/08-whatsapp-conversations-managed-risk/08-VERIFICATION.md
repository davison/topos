---
phase: 08-whatsapp-conversations-managed-risk
verified: 2026-08-10T18:40:00Z
status: gaps_found
score: 4/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: 5/5
  gaps_closed:
    - "G-08-1: QR panel poll cadence tied to expires_in_seconds left a completed real-device pairing undelivered for up to 60s and visually indistinguishable from a dead session (plans 08-05/08-06/08-07)"
  gaps_remaining: []
  regressions:
    - "New CR-01 (08-REVIEW.md, post-gap-closure pass): QRPanel.svelte's beginSession() never cancels the link session if the component is torn down while the initial POST /api/config/whatsapp-link is still in flight — an orphaned subprocess (and, for Re-link, an orphaned suspended source instance) for up to 5 minutes. Confirmed still present in the shipped code; no fix commit exists for it."
gaps:
  - truth: "Closing the Add-Source/Re-link dialog does not leave an orphaned link subprocess or an orphaned suspended source instance behind"
    status: failed
    reason: "08-REVIEW.md's CR-01 (generated after gap-closure plans 08-05/08-06/08-07 landed) documents a race in QRPanel.svelte: beginSession() awaits startWhatsAppLink(), and its only guard on an already-torn-down component is `if (retired) return;` — which discards the just-returned session id without ever calling cancelWhatsAppLink. The kernel has already spawned a real subprocess (and, on the Re-link path, already suspended the real source instance) by the time it answers 200, so a user who opens the dialog and closes it again (Escape, backdrop click, any unmount) while that POST is still in flight strands both for up to linkSessionDeadline (5 minutes), or sooner exhausts maxConcurrentLinkSessions (4) and returns 429 to an unrelated later attempt. This directly contradicts the component's own onDestroy comment ('must never leave a subprocess alive holding the WhatsApp store lock') and undermines T-08-07's two-independent-layers mitigation the phase's own threat model claims. Verified independently in this session by reading QRPanel.svelte:219-241 at HEAD — the vulnerable `if (retired) return;` branch (no cancelWhatsAppLink call) is exactly as 08-REVIEW.md describes, and no test in qr-panel.test.ts exercises this path (only the already-mounted cancel/unmount case, T-08-10, is covered)."
    artifacts:
      - path: "web/src/lib/components/QRPanel.svelte"
        issue: "beginSession()'s success branch (`if (retired) return;` at line ~229) discards a just-returned session id with no cancel call when the component was torn down while the start request was in flight"
    missing:
      - "Cancel the session the moment its id is known even if `retired` is already true by the time startWhatsAppLink resolves (08-REVIEW.md's own suggested fix, not yet applied)"
      - "A qr-panel.test.ts guard (or e2e case) exercising unmount-during-in-flight-start, so this regression can't silently return"
---

# Phase 8: WhatsApp Conversations (Managed Risk) Verification Report

**Phase Goal:** User's WhatsApp groups for a topic appear in the webspace stream via a linked-device session, and everything else keeps working when that session breaks
**Verified:** 2026-08-10T18:40:00Z
**Status:** gaps_found
**Re-verification:** Yes — after gap closure for UAT gap G-08-1 (plans 08-05, 08-06, 08-07)

## Context

This re-verification follows a prior full verification (`previous_status: passed`, 5/5, dated earlier the same day) that closed an unrelated code-review gap (CR-01: `describePlugin` trial-launch colliding with a running WhatsApp instance's store lock). Since that verification, `08-UAT.md` ran and found **G-08-1**: a real-device QR pairing succeeded on the phone but the topos QR panel never left the QR screen, and the pairing was discarded on cancel. Root-cause diagnosis identified an AND-gate of three defects (poll cadence tied to a QR code's `expires_in_seconds`, no wire state between `qr` and `paired`, and the kernel discarding the link subprocess's stderr). Plans 08-05 (browser consumer half), 08-06 (plugin/kernel producer half), and 08-07 (recovery affordance + regression armor) executed to close it.

This re-verification independently confirms G-08-1's fix, re-confirms the four roadmap Success Criteria still hold by regression, and — per the mandate to verify the actual codebase rather than SUMMARY claims — ran a full anti-pattern/code-review sweep of the phase's own artifacts. That sweep surfaced a **new, unresolved critical finding** (`08-REVIEW.md`'s CR-01, distinct from the earlier same-named CR-01) that was never fixed. It is confirmed still present in the shipped code below.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | **G-08-1 closed:** a completed real-device pairing reaches the panel within seconds (fixed poll cadence, decoupled from QR validity) and a post-pair progress state prevents the frozen-code-looks-alive illusion | ✓ VERIFIED | `QRPanel.svelte` read in full: `POLL_INTERVAL_MS = 2000`, `schedulePoll()` takes no delay argument, `pairing` phase renders between `qr` and terminal states, countdown restarts only on payload change, `default:` calls `schedulePoll()` rather than hanging. Playwright case 9 (`qr → pairing_accepted → paired` at a realistic 60s first-code expiry) re-run live in this session: **passes in ~7s**, proving the fix mechanically closes the reported symptom rather than merely claiming to. |
| 2 | User can link WhatsApp as a device by scanning a QR code, and the session survives service restarts without re-linking | ✓ VERIFIED (regression — unchanged since prior verification) | `link.go`, `connect.go`, `pairwait.go`, `plugin.go` untouched by the fix commits except `link.go`'s two new non-terminal event constructors (additive). `plugins/whatsapp`'s full test suite re-run live: `ok github.com/davison/topos/plugins/whatsapp 0.053s`. |
| 3 | Messages from WhatsApp groups whose names match the webspace's matching config appear in the stream alongside every other source, using the Phase-4 chat rendering | ✓ VERIFIED (regression — unchanged) | `render.go`, `digest.go`, `plugin.go`'s `Match` logic untouched by gap-closure commits. |
| 4 | The plugin persists its own message store, so conversations captured while it was running stay browsable regardless of what the WhatsApp desktop app retains | ✓ VERIFIED (regression — unchanged) | `messagestore.go` untouched by gap-closure commits; full `./kernel/...` and `./plugins/whatsapp/...` suites re-run live, all pass. |
| 5 | De-link, ban, or session expiry surfaces as an explicit plugin-health error in the UI while previously captured messages remain browsable and every other source is unaffected — **and closing the QR/Re-link dialog never leaves an orphaned subprocess or suspended instance behind** | ✗ FAILED | `health.go` regression-verified unchanged. But the "every other source is unaffected" half of this truth is undermined by the newly-discovered, unresolved CR-01 (see Gaps below): a routine dialog-close during an in-flight link start can strand a suspended source instance and its subprocess for up to 5 minutes, and repeated occurrences exhaust `maxConcurrentLinkSessions`, degrading the WhatsApp linking feature itself for an unrelated later attempt — the opposite of "everything else keeps working." |

**Score:** 4/5 truths verified (the QR-flow fix itself is proven closed; a separate, newly-discovered reliability defect in the same code area is not)

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `docs/api.md` | Amended poll-cadence rule; `pairing_accepted`/`already_linked` non-terminal state rows; stderr-routing note | ✓ VERIFIED | Stale cadence-from-expiry parenthetical removed (grep count 0); both new states present ≥2× each; non-terminal paragraph amended; stderr note present |
| `.planning/phases/08-whatsapp-conversations-managed-risk/08-UI-SPEC.md` | Dated Amendment 2 with supersession pointers into Amendment 1; Checker Sign-Off intact | ✓ VERIFIED | Amendment 2 heading present; two `Superseded in part by Amendment 2` pointers in Amendment 1; `## Checker Sign-Off` present below Amendment 2 |
| `web/src/lib/api.ts` | `WhatsAppLinkState` widened with `pairing_accepted`/`already_linked` | ✓ VERIFIED | Union includes both, doc comment updated |
| `web/src/lib/components/QRPanel.svelte` | Fixed-cadence `schedulePoll()`, `pairing` phase, payload-gated countdown, no-Cancel-during-pairing | ✓ VERIFIED (for G-08-1's specific fix); ✗ carries the separate CR-01 defect (see Gaps) | Read in full; matches plan 08-05's stated behavior exactly for the cadence/progress-state fix |
| `web/src/lib/components/qr-panel.test.ts` | Structural guards for cadence, progress branch, cancel gating, countdown fallback | ✓ VERIFIED | 23 assertions; all pass live (`npm run test -- --run` on this file: pass) |
| `plugins/whatsapp/link.go` | `pairing_accepted`/`already_linked` non-terminal events, device-id-free on stdout | ✓ VERIFIED | Read in full; `linkEventKindPairingAccepted`/`linkEventKindAlreadyLinked`, constructors carry only `Kind`; device id confirmed present only on the stderr diagnostic |
| `plugins/whatsapp/link_test.go` | Guards for the new events' shape and device-id exclusion | ✓ VERIFIED | All `TestLink*` tests re-run live, pass |
| `kernel/httpapi/whatsapplink.go` | `newExecLinkSpawner(logger)` constructor; `stderrLineLogger` capturing subprocess stderr; corrected `cmd.Env` comment; `isTerminalKind` unchanged (still exactly `paired`/`error`/`timeout`) | ✓ VERIFIED | Read in full; `routes.go`'s single call site updated to pass `logger` |
| `kernel/httpapi/whatsapplink_exec_test.go` (new) | First automated coverage of the production spawner: streaming, argv, stderr capture, trailing-flush, env inheritance, kill | ✓ VERIFIED | All 6 `TestExecLinkSpawner_*` cases re-run live, pass |
| `kernel/httpapi/whatsapplink_test.go` | Progress-state non-terminal passthrough coverage | ✓ VERIFIED | `TestWhatsAppLink_ProgressStatesAreNonTerminal`, `TestIsTerminalKind_ProgressKindsAreNonTerminal` re-run live, pass |
| `web/src/lib/components/AddSourceModal.svelte` | Neutral, non-Alert declined-link notice naming the recovery route | ✓ VERIFIED (for its own stated scope); carries the separate, lower-severity WR-01 defect (stale co-render — see below) | `linkNotice` state present, set in `handleLinkCancelled`, cleared in `resetFlowState`/`selectPluginType`, rendered outside any `Alert` |
| `web/src/lib/components/add-source.test.ts` | Guards that the notice is neutral and the cancel path sets no failure state | ✓ VERIFIED | New describe block present; full vitest run confirms pass |
| `web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts` | Three new hermetic cases: qr→poll→paired at realistic expiry, no-cancel-during-pairing, already-linked recovery | ✓ VERIFIED | Cases 9, 10, 11 present; **re-run live against the real built kernel + mock plugins in this session: 11/11 pass**, case 9 completing in ~7s despite its 60s scripted first-code expiry |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `plugins/whatsapp/link.go`'s `pairingAccepted()`/`alreadyLinked()` | `kernel/httpapi/whatsapplink.go`'s `consume` → poll response | stdout event line → `linkEvent` decode → HTTP poll body | ✓ WIRED | `TestWhatsAppLink_ProgressStatesAreNonTerminal` proves a `pairing_accepted`/`already_linked` line the fake subprocess emits reaches a 200 poll response carrying that exact state, non-terminating |
| Kernel poll response `state` | `QRPanel.svelte`'s `applySession` switch | `pollWhatsAppLink` → `applySession(session)` | ✓ WIRED | `pairing_accepted`/`already_linked` cases both set `phase = 'pairing'` and call `schedulePoll()`; live Playwright case 9/11 confirm this renders and progresses |
| Link subprocess stderr | kernel hclog logger | `cmd.Stderr = stderrLineLogger` | ✓ WIRED | `TestExecLinkSpawner_CapturesStderrIntoLogger`/`_FlushesTrailingPartialStderrLine` re-run live, pass — diagnostics reach the sink, never a response body (also pinned by `TestWhatsAppLink_ProgressStatesAreNonTerminal`'s key-set assertion) |
| `AddSourceModal.svelte`'s `handleLinkCancelled` | rendered `linkNotice` paragraph | `$state` set → `{#if linkNotice}` block | ⚠️ WIRED but incompletely reset | Set correctly on cancel; **not cleared by `handleConnectNext`** (08-REVIEW.md WR-01, confirmed: no `linkNotice = ''` inside that function) — a second failed trial-launch after a declined link can render the neutral notice and the destructive connection-failure alert simultaneously. Warning-grade, not a truth-blocking failure, but a real UX defect left unresolved by this gap-closure pass — listed under Anti-Patterns below, not counted against the score. |
| `QRPanel.svelte`'s `beginSession()` (in-flight start) | `cancelWhatsAppLink` | — | ✗ NOT WIRED | See Gaps — no call path exists from "component retired while start request in flight" to `cancelWhatsAppLink`. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Full Go workspace build | `CGO_ENABLED=0 go build ./...` | succeeds | ✓ PASS |
| whatsapp plugin build + full suite | `cd plugins/whatsapp && CGO_ENABLED=0 go build ./... && go test ./...` | builds; `ok ... 0.053s` | ✓ PASS |
| Kernel httpapi: exec spawner + link-session progress-state suite | `CGO_ENABLED=0 go test ./kernel/httpapi/... -run 'TestExecLinkSpawner|TestWhatsAppLink|TestIsTerminalKind' -v` | 15/15 subtests pass | ✓ PASS |
| Full Go workspace test suite (single run) | `CGO_ENABLED=0 go test ./...` | all packages `ok` | ✓ PASS |
| Web unit suite (qr-panel + add-source) | `npm run test -- --run src/lib/components/qr-panel.test.ts src/lib/components/add-source.test.ts` | 67/67 pass | ✓ PASS |
| Full web unit suite (single run) | `npm run test -- --run` | 662/662 pass, 38/38 files | ✓ PASS |
| svelte-check | `npm run check` | 838 files, 0 errors, 9 pre-existing warnings unrelated to this phase | ✓ PASS |
| e2e typecheck | `npm run check:e2e` | 0 errors | ✓ PASS |
| Playwright: whatsapp-qr-link spec, real build (not mocked describe) | build web + kernel + mock plugins, `npx playwright test --project=chromium specs/uat-08-whatsapp-qr-link.spec.ts` | **11/11 pass**; case 9 (`qr→pairing_accepted→paired` at a scripted 60s first-code expiry) completes in ~7s | ✓ PASS — this is the direct, live-in-this-session proof that G-08-1's reported symptom is closed |
| Debt-marker scan across all 14 files touched by 08-05/08-06/08-07 | `grep -n -E "TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER"` per file | no matches in any file | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes declared by this phase's plans or ROADMAP success criteria. Step 7c: SKIPPED (no declared probes).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| SRC-03 | 08-01, 08-02, 08-03, 08-04, 08-05, 08-06, 08-07 | WhatsApp plugin runs as a whatsmeow linked device with its own persistent message store; degrades gracefully on de-link/ban; matches on group names | ⚠️ PARTIALLY SATISFIED | G-08-1's QR-link reliability fix is fully closed and independently confirmed. The "degrades gracefully… everything else keeps working" half of SRC-03's intent is not yet fully met: the newly-discovered CR-01 (unmount-during-in-flight-start orphaning a subprocess and a suspended instance) is a genuine, currently-unresolved regression risk to that same guarantee. `.planning/REQUIREMENTS.md` line 30 still shows `[ ]` Pending — consistent with this verification's outcome (gaps_found), not an omission. |

No orphaned requirements — SRC-03 is the only ID `.planning/ROADMAP.md` maps to Phase 8, and all seven plans (including 08-05/06/07) declare `requirements: [SRC-03]`.

### Anti-Patterns Found

No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers in any of the 14 files touched by the gap-closure plans (checked individually; all clean).

However, the phase's own post-gap-closure code review (`08-REVIEW.md`, generated after 08-05/06/07 landed, scoped to the 13 files those plans touched) found three issues that were **never fixed** — confirmed independently against HEAD in this verification session, not taken on the review's word:

| File | Line(s) | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `web/src/lib/components/QRPanel.svelte` | ~219-241 (`beginSession`) | Unmount-during-in-flight-start race: `if (retired) return;` discards a just-returned session id with no `cancelWhatsAppLink` call | 🛑 Blocker | Orphans a real subprocess (and, on Re-link, a suspended source instance) for up to 5 minutes on an entirely ordinary interaction (open dialog, close it quickly); repeated occurrences exhaust the 4-session concurrency cap, degrading the WhatsApp linking feature for unrelated later attempts — this is the "everything else keeps working" guarantee failing on the linking feature's own resource pool. Directly contradicts the file's own `onDestroy` invariant comment. Promoted to a `gaps` entry above. |
| `web/src/lib/components/AddSourceModal.svelte` | `handleConnectNext` (no `linkNotice = ''`) | Stale declined-link notice can co-render with a later, unrelated connection-failure `Alert` | ⚠️ Warning | Confusing but not data-losing or session-orphaning; a user sees two contradictory messages ("not linked yet, save and link later" alongside "couldn't verify this connection") after declining once and then hitting a real failure on a later attempt. Not counted against the score; recorded here so it is not lost. |
| `web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts` | ~298-302 (case 2 comment) | Comment still references `POLL_FLOOR_MS`, a mechanism this same gap-closure deleted | ℹ️ Info | Cosmetic only — misleads a future reader, no functional impact. |

08-REVIEW-FIX.md on disk (`fixed_at: 2026-08-10T16:40:00Z`) documents fixes for a **different, earlier** CR-01/WR-01/WR-02 set (the `describePlugin` trial-launch store-lock collision, found in the original 44-file review that predates G-08-1's UAT discovery). That fix pass is real, verified, and unrelated to the findings above — `08-REVIEW.md` on disk today is a later, narrower pass (13 files, explicitly stated to replace the prior review) whose own findings have no corresponding fix report at all.

### Human Verification Required

1. **D5 (deferred per `workflow.human_verify_mode = end-of-phase`, plans 08-05 and 08-07): real-device re-test of UAT test 1.**
   **Test:** `make dev`, Add Source → New WhatsApp… with a display name and the seeded local path; scan the QR with a real phone and accept the pairing; observe the panel.
   **Expected:** The panel leaves the QR state within a few seconds of the phone reporting success (shows "Scan accepted — completing login…", then the linked confirmation, then Step 2); it must not sit on a code with a ticking countdown. Restart the kernel and confirm it reconnects with no second QR. Check the kernel log for the link subprocess's own diagnostics (previously silently discarded).
   **Why human:** Requires a live WhatsApp account and a real kernel run — inherently outside this automated environment. This is the check that closes G-08-1 from the user's own perspective; the automated evidence in this report (live Playwright case 9, live unit/integration suites) proves the mechanism is fixed, but only a real device confirms the originally-reported symptom no longer reproduces.
   **Note:** given the new CR-01 gap found in this verification, closing the dialog quickly after a real scan during this re-test should also be tried once, to observe whether the orphaned-subprocess defect reproduces in practice (it is a race window, not a certainty, in normal usage).

### Gaps Summary

**One blocking gap.** G-08-1 itself — the reported symptom ("modal remains on screen with the refresh counter dwindling," no connection from the topos side) — is closed and independently proven closed via a live Playwright run of the exact `qr → pairing_accepted → paired` sequence at a realistic 60-second first-code expiry, plus full regression passes across every touched Go and web test suite. That work is solid.

But this verification's own anti-pattern sweep of the phase's artifacts surfaced `08-REVIEW.md` — a code review generated after 08-05/06/07 landed, scoped exactly to the files those plans touched — documenting an unresolved **Critical** finding (CR-01: unmount-during-in-flight-start race in `QRPanel.svelte`) that has no corresponding fix commit. Independent verification in this session (reading the code at HEAD, checking for a fix commit, checking for a covering test) confirms the defect is real and present today. It sits in the same component this phase's core success criterion 1 (QR linking) depends on, and it directly undermines the "everything else keeps working" clause of the phase goal: a routine, non-adversarial interaction (opening and closing the link dialog) can strand a subprocess and a suspended source instance for up to five minutes, degrading the WhatsApp linking feature itself for unrelated later attempts.

This gap did not exist at the time `08-UAT.md`/G-08-1 was diagnosed — it was introduced into scope (or at minimum, first surfaced) only by the code review that ran after the gap-closure plans shipped, and was never routed through a fix pass. It must be closed (or explicitly overridden with a documented rationale) before Phase 8 can be marked passed.

---

_Verified: 2026-08-10T18:40:00Z_
_Verifier: Claude (gsd-verifier)_
