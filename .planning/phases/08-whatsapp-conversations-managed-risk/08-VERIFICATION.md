---
phase: 08-whatsapp-conversations-managed-risk
verified: 2026-08-11T02:15:00Z
status: gaps_found
score: 4/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: 5/5
  gaps_closed:
    - "G-08-4 plugin half (root cause 1): plugins/whatsapp/health.go now declares healthStateConnecting FIRST in the iota block (the plugin's Go zero value), so a never-explicitly-assigned *SourcePlugin can no longer report the false 'Not linked — pair this device' message for an already-paired, actively-connecting device. Confirmed by direct read of health.go/connect.go at HEAD and by live re-run of TestConnectingState_IsTheZeroValue and TestConnectingState_MatchMessageIsNotThePairingInstruction."
    - "G-08-4 plugin half (root cause 1, belt-and-braces): connect.go's startBackgroundClient now explicitly calls p.setHealthState(healthStateConnecting, \"\") before dialing, and registers a pairLoginWaiter (reusing the link flow's own primitive) before client.Connect(), blocking up to serveLoginTimeout (15s) for a real *events.Connected before returning. Confirmed by source read and by the AST ordering guard TestStartBackgroundClient_SuccessPathSetsConnectingAndWaitsForLogin, re-run live and passing."
    - "G-08-4 kernel half (root causes 2 and 3): kernel/syncer/scheduler.go's Scheduler now retries a generation's immediate first refresh on a bounded 2s/5s backoff (firstRefresh) when it errors, so a launch-window Match failure is superseded by a later successful sync_runs row within seconds instead of staying pinned for the 15-minute default interval. Ticker/manual/HTTP refreshes are untouched. Confirmed by source read and by live re-run of TestScheduler_FirstRefreshRetriesUntilTheSourceIsReady, TestScheduler_FirstRefreshGivesUpAndLeavesTheErrorRecorded, TestScheduler_FirstRefreshRetryStopsOnContextCancel, plus the three pre-existing scheduler tests (unmodified, still pass)."
    - "G-08-4 fixture gap (missing[3]): plugins/mock gained an opt-in, default-off WEBSPACES_MOCK_READY_AFTER_MS launch-readiness window (guarding Match/Health only, never Describe), giving this failure class its first real-subprocess fixture. Confirmed by source read and live re-run of plugins/mock's full test suite."
    - "First hermetic, real-subprocess gate for this failure class: kernel/supervisor/readiness_test.go's TestBoot_FirstRefreshSurvivesAPluginLaunchReadinessWindow boots a real supervisor against a real topos-plugin-mock subprocess refusing Match for 700ms and asserts the source ends with an ok latest sync run AND persisted, streamable items. Re-run live in this session: PASS (2.45s). I independently re-ran the plan's mandated negative control myself (not trusting the SUMMARY's claim): reverted runSource to call refreshAndLog directly (undoing the retry), reran the same test, confirmed it FAILS with the launch-window error pinned as the latest sync run, then restored the fix and reconfirmed the file is back to a clean git diff and the suite is green again."
  gaps_remaining:
    - "Real-device re-test of the resume/re-link flow (08-UAT.md's still-open item from the prior human_needed verification) has still not been performed — carried forward, now secondary to the new gap below."
  regressions:
    - "NEW, found independently in this re-verification (not reported by either gap-closure plan's SUMMARY, and only partially characterized by the phase's own fresh code review as a boot-time-only concern): plugins/whatsapp/connect.go's new bounded login wait (serveLoginTimeout, 15s), introduced by 08-11 to fix G-08-4, is invoked from inside kernel/supervisor.Supervisor's SuspendInstance resume closure (kernel/supervisor/supervisor.go:380-401) WHILE THAT CLOSURE HOLDS s.mu for the closure's entire duration. s.mu is the same mutex Host()/Coordinator() take (supervisor.go:182-193), and Fetch/ProbeSources/Refresh/RefreshAll (supervisor.go:205-237) — which back every item-open, health-probe and manual-refresh HTTP route for EVERY source, not just WhatsApp — all resolve through Host()/Coordinator() on every call. Because the resume closure is invoked synchronously inside WhatsAppLinkPollHandler on the terminal poll of a real re-link (kernel/httpapi/whatsapplink.go:721-734) and inside WhatsAppLinkCancelHandler (line 753-759), the practical effect is: at the exact moment a real WhatsApp re-link completes — the scenario the whole G-08-3/G-08-4 effort exists to make safe — every other source's item-fetch, health-probe, and manual-refresh routes across the entire kernel are frozen for up to 15 seconds. This directly contradicts phase success criterion 4's 'every other source is unaffected' clause. A narrower version of the same underlying defect (kernel BOOT delayed by up to 15s when an already-linked WhatsApp source exists, before any lock contention is even possible since the HTTP server isn't listening yet) was independently found by the phase's own mandated fresh code review of this exact wave (08-REVIEW.md, committed e748545, findings WR-01/WR-02) but was never fixed — there is no 08-REVIEW-FIX.md covering this review pass, and no commit after e748545 touches connect.go, supervisor.go, or host.go."
---

# Phase 8: WhatsApp Conversations (Managed Risk) Verification Report

**Phase Goal:** User's WhatsApp groups for a topic appear in the webspace stream via a linked-device session, and everything else keeps working when that session breaks
**Verified:** 2026-08-11T02:15:00Z
**Status:** gaps_found
**Re-verification:** Yes — after gap-closure plans 08-11 (plugin: `healthStateConnecting` zero-value fix + serve-mode login wait) and 08-12 (kernel: bounded first-refresh retry + mock launch-readiness fixture + hermetic supervisor gate), which close `08-UAT.md`'s gap G-08-4 (a freshly launched, already-paired WhatsApp plugin misreported "Not linked" and the source stayed dead until the 15-minute sync interval), found by the previous UAT cycle after the prior `08-VERIFICATION.md` (`status: human_needed`, 5/5).

## Context

The prior verification (`status: human_needed`, 5/5) found all automated evidence for the phase's five observable truths in place but routed to `human_needed` pending a real-device UAT re-run of the G-08-3 fix. That UAT re-run (`08-UAT.md`) confirmed G-08-3's presentation-layer fix works live, but surfaced a NEW issue: a freshly-launched, already-paired WhatsApp instance still reported "Not linked" immediately after a real pairing. This became gap G-08-4, diagnosed as an AND-gate of three defects (plugin-side dishonest zero-value health state, kernel-side missing launch-readiness gate, and a missing fixture to catch this failure class at all).

Plan 08-11 closed the plugin-side defect (a new `healthStateConnecting` zero value, plus an explicit connecting-state assignment and a bounded login wait before the go-plugin handshake completes). Plan 08-12 closed the kernel-side defect (a bounded, generation-scoped first-refresh retry) and the fixture gap (`plugins/mock`'s opt-in launch-readiness window plus a hermetic real-subprocess gate). Both plans' own SUMMARYs report clean, complete closure with no deviations.

**This re-verification independently re-reads all touched code (not on SUMMARY's word), re-runs every relevant test live in this session — including re-running the plan's own mandated negative control by hand rather than trusting its recorded outcome — and finds G-08-4 genuinely closed by all three legs.** However, tracing the actual call graph of 08-11's new blocking wait (a step none of the SUMMARYs, and only half of the phase's own fresh code review, characterized fully) surfaced a real, unaddressed regression: the wait is invoked from inside a code path that holds the kernel's central supervisor mutex for its entire duration, freezing every other source's item-fetch/health-probe/refresh routes for up to 15 seconds at the exact moment a real WhatsApp re-link completes — the one scenario success criterion 4 explicitly promises stays unaffected. This is why the report routes to `gaps_found` rather than a clean `passed` or a re-affirmed `human_needed`.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | User can link webspaces as a WhatsApp device by scanning a QR code, and the session survives service restarts without re-linking | ✓ VERIFIED (regression-free; see Gaps Summary for a related but separate boot-latency note) | `link.go`, `pairwait.go`, `plugin.go`'s link-flow surface untouched by 08-11/08-12. Full `plugins/whatsapp` suite re-run live: `ok github.com/davison/topos/plugins/whatsapp 0.020s`. |
| 2 | Messages from WhatsApp groups whose names match the webspace's matching config appear in the stream alongside every other source, using the Phase-4 chat rendering | ✓ VERIFIED (regression-free) | `render.go`, `digest.go`, `plugin.go`'s `Match`/`Eligible` logic untouched by 08-11/08-12; full whatsapp-plugin suite passes live, including `TestDelink_MatchReturnsUnavailableForEveryNonHealthyState` extended to cover the new sixth state. |
| 3 | The plugin persists its own message store, so conversations captured while it was running stay browsable regardless of what the WhatsApp desktop app retains | ✓ VERIFIED (regression-free) | `messagestore.go` untouched by 08-11/08-12; whatsapp-plugin suite passes live. |
| 4 | De-link, ban, or session expiry surfaces as an explicit plugin-health error in the UI while previously captured messages remain browsable and **every other source is unaffected** | ✗ FAILED — new regression introduced by this wave's own fix | See "Regression" in the Gaps Summary and the frontmatter `re_verification.regressions` entry. `kernel/supervisor/supervisor.go`'s `SuspendInstance` resume closure (lines 380-401) holds `s.mu` for the entire duration of `s.host.Reconcile(...)`, which now transitively blocks up to `serveLoginTimeout` (15s, `plugins/whatsapp/connect.go`) launching a re-linked WhatsApp instance. `Host()`/`Coordinator()` (supervisor.go:182-193) — which `Fetch`/`ProbeSources`/`Refresh`/`RefreshAll` (supervisor.go:205-237) all resolve through on every call — take the same `s.mu`. `Fetch` backs `ItemHandler`/`ItemContentHandler`/`ItemThumbnailHandler` (`kernel/httpapi/item.go`) for every source. The resume closure is invoked synchronously inside the real re-link's terminal poll (`kernel/httpapi/whatsapplink.go:721-734`) and cancel handler (`:753-759`). Net effect, confirmed by direct code read: at the exact moment a real re-link completes, every other source's item-open/health-probe/manual-refresh routes freeze kernel-wide for up to 15s. |
| 5 | G-08-4 itself is closed: a freshly launched, already-paired instance reports an honest transient state (never the false "Not linked" message) and the source syncs within seconds rather than staying dead for the 15-minute interval | ✓ VERIFIED by hermetic evidence, independently re-run and negative-controlled in this session | See re_verification.gaps_closed above for the full evidence chain, including my own from-scratch re-run of the plan's negative control (not trusting the recorded SUMMARY claim). |

**Score:** 4/5 truths verified by direct, live-re-run codebase evidence. Truth #4 fails on a newly introduced, independently discovered regression (not reported by either 08-11/08-12 SUMMARY) that violates the phase's own "every other source is unaffected" language during the exact re-link scenario this wave was built to make safe.

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `plugins/whatsapp/health.go` | `healthStateConnecting` declared first in the iota block (the zero value); distinct, non-data-loss-implying message template | ✓ VERIFIED | Read in full at HEAD; matches 08-11-PLAN's stated shape exactly. |
| `plugins/whatsapp/health_test.go` | `healthStateConnecting` in `nonHealthyStates`; `TestConnectingState_IsTheZeroValue`; `TestConnectingState_MatchMessageIsNotThePairingInstruction` | ✓ VERIFIED | Both new tests present, re-run live, pass; all pre-existing per-state regressions (`TestHealthState_HealthyExactlyOne`, `TestHealthState_MessagesNonEmptyAndDistinct`, `TestHealth_ReachableFalseWithLastErrorPerState`) still pass with the sixth state included. |
| `plugins/whatsapp/connect.go` | `serveLoginTimeout` constant; explicit connecting-state assignment before dial; `pairLoginWaiter` registered before `Connect()`, awaited after | ✓ VERIFIED (functionally correct in isolation; see Truth #4 for the cross-package consequence) | Read in full at HEAD; ordering matches 08-11-PLAN exactly, including the doc comment's own acknowledgement that "every second spent here is a second the kernel's pluginhost.launch is blocked on the handshake completing" — a caveat the plan flagged but whose blast radius (the supervisor-wide mutex) was not traced. |
| `plugins/whatsapp/connect_test.go` | AST structural guard over `startBackgroundClient`'s call ordering | ✓ VERIFIED | `TestStartBackgroundClient_SuccessPathSetsConnectingAndWaitsForLogin` present, re-run live, passes. |
| `.planning/phases/08-whatsapp-conversations-managed-risk/08-UI-SPEC.md` | Sixth taxonomy row for the connecting cause | ✓ VERIFIED | Row present at line 61, dated note at line 64 naming G-08-4/Plan 08-11. |
| `kernel/syncer/scheduler.go` | `FirstRefreshRetryDelays` field; `defaultFirstRefreshRetryDelays`; `firstRefresh` retry loop; `refreshAndLog` returns success bool | ✓ VERIFIED | Read in full at HEAD; matches 08-12-PLAN's stated shape exactly. |
| `kernel/syncer/scheduler_test.go` | `flakySource`; three new retry tests; three pre-existing tests unmodified | ✓ VERIFIED | All six tests re-run live, pass (1.554s total). |
| `plugins/mock/readiness.go` | `readyAfterEnvVar`, `notReadyMessage`, `readinessWindow`, `readinessWindowFromEnv` (nil-receiver-ready, injectable getenv) | ✓ VERIFIED | New file present, matches plan exactly; explicit "TEST FIXTURE, not contract" doc comment present. |
| `plugins/mock/plugin.go`, `plugins/mock/main.go` | `ready *readinessWindow` field; guard on Match/Health only, never Describe; env parsed before `goplugin.Serve` | ✓ VERIFIED | Read in full; three surgical edits exactly as specified; `main.go` fails loudly (`os.Exit(1)`) on a malformed value. |
| `plugins/mock/plugin_test.go` | Table test over `readinessWindowFromEnv`; Match/Describe-inside/after-window tests | ✓ VERIFIED | Present, re-run live, pass (`TestReadinessWindowFromEnv` 6 subtests, plus the two Match/window tests). |
| `kernel/supervisor/readiness_test.go` | End-to-end gate over a real mock subprocess through a launch-readiness window | ✓ VERIFIED | Present, re-run live: PASS (2.45s). Negative control independently re-verified by me in this session (see re_verification notes) — fails with the exact G-08-4-shaped error message when the retry fix is reverted, confirming the gate is real. |
| `docs/testing.md` | `WEBSPACES_MOCK_READY_AFTER_MS` documented under "The two mock-shaped plugins"; dated "What changed" entry | ✓ VERIFIED | Both present, read in full, match plan's stated content. |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `connect.go`'s pre-dial `setHealthState(healthStateConnecting, "")` | `plugin.go`'s `currentMessage()` → `Match`'s `codes.Unavailable` message text | direct call chain | ✓ WIRED | Confirmed by source read and `TestConnectingState_MatchMessageIsNotThePairingInstruction`. |
| `pairLoginWaiter` registered on the same client as `p.handleEvent`, before `Connect()` | `healthStateLinked` assignment landing before `wait()` returns | whatsmeow's synchronous, in-registration-order event dispatch | ✓ WIRED | Confirmed by source read (ordering comment explicit); no automated test can exercise this without a live server — correctly left as an AST ordering guard rather than a false claim of behavioral coverage. |
| `Scheduler.runSource`'s immediate first refresh | `firstRefresh`'s bounded retry loop → `Coordinator.Refresh` → a later `sync_runs` row | `LatestSyncRunPerSource`'s `MAX(id)`-per-source selection | ✓ WIRED | Confirmed by source read and by `TestScheduler_FirstRefreshRetriesUntilTheSourceIsReady` asserting `status == "ok"` becomes latest. |
| `WEBSPACES_MOCK_READY_AFTER_MS` set on the kernel process | the mock subprocess's own env | `pluginhost.launch`'s `append(os.Environ(), ...)` | ✓ WIRED | Confirmed by source read and by `TestBoot_FirstRefreshSurvivesAPluginLaunchReadinessWindow` (re-run live, PASS) driving a real subprocess through the window via `t.Setenv`. |
| **NEW FINDING:** `SuspendInstance`'s resume closure's `s.mu.Lock()` | `Host()`/`Coordinator()` (both take `s.mu`) → `Fetch`/`ProbeSources`/`Refresh`/`RefreshAll` for **every** source | shared mutex, held across the now-potentially-15s `Host.Reconcile` call | ⚠️ WIRED-BUT-HARMFUL | Confirmed by direct source read: `supervisor.go:380-401` (resume closure), `:182-193` (`Host`/`Coordinator`), `:205-237` (`Fetch`/`ProbeSources`/`Refresh`/`RefreshAll`), `kernel/httpapi/item.go` (`Fetch` backs every item-open route), `kernel/httpapi/whatsapplink.go:721-734,753-759` (resume invoked synchronously from the poll/cancel handlers). This link was not intended by either plan and is the direct cause of Truth #4's failure. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Full Go workspace build | `CGO_ENABLED=0 go build ./...` | clean | ✓ PASS |
| Full kernel test suite | `CGO_ENABLED=0 go test ./kernel/... -count=1` | all packages `ok` | ✓ PASS |
| Targeted G-08-4 whatsapp-plugin tests (health/connect/delink) | `go test ./plugins/whatsapp/... -run '...' -v` | all pass | ✓ PASS |
| Full whatsapp plugin suite | `go test ./plugins/whatsapp/... -count=1` | `ok` (0.020s) | ✓ PASS |
| Full scheduler suite (6 tests incl. 3 new + 3 pre-existing unmodified) | `go test ./kernel/syncer/... -run TestScheduler -v` | all pass | ✓ PASS |
| Full mock plugin suite | `cd plugins/mock && go test ./... -v` | all pass, incl. `TestReadinessWindowFromEnv` (6 subtests) | ✓ PASS |
| Real-subprocess hermetic gate | `go test ./kernel/supervisor/... -run TestBoot_FirstRefreshSurvivesAPluginLaunchReadinessWindow -v` | PASS (2.45s) | ✓ PASS |
| **Negative control (re-run independently by verifier, not trusting SUMMARY)** | Reverted `runSource` to call `refreshAndLog` directly, re-ran the same gate | **FAILS** as expected: `"G-08-4: mock-01 never reached an \"ok\" latest sync run... error=\"...WEBSPACES_MOCK_READY_AFTER_MS launch-readiness window not yet elapsed\""`, then file restored and confirmed `git diff` clean | ✓ PASS (gate proven non-vacuous) |
| Full workspace `make test-portable` (13 modules) | `make test-portable` | all `ok` | ✓ PASS |
| Debt-marker scan on all 13 files touched by 08-11/08-12 | `grep -n -E "TBD\|FIXME\|XXX\|TODO\|HACK\|PLACEHOLDER"` per file | no matches | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes declared by this phase's plans or ROADMAP success criteria. Step 7c: SKIPPED (no declared probes).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| SRC-03 | 08-01 … 08-12 | WhatsApp plugin runs as a whatsmeow linked device with its own persistent message store; degrades gracefully on de-link/ban; matches on group names | ✗ BLOCKED | Three of four success criteria have complete, regression-free automated evidence, and G-08-4 (the plugin/kernel/fixture launch-readiness defect) is genuinely closed. But criterion 4's "every other source is unaffected" clause is now violated by a regression this same gap-closure wave introduced (see Truth #4). `.planning/REQUIREMENTS.md` line 30 correctly still shows `[ ]` — accurate, not an omission. |

No orphaned requirements — SRC-03 is the only ID `.planning/ROADMAP.md` maps to Phase 8, and all twelve plans (including 08-11, 08-12) declare `requirements: [SRC-03]`.

### Anti-Patterns Found

No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers in any of the 13 files touched by 08-11/08-12.

The phase's own fresh code review of this exact wave (`08-REVIEW.md`, committed `e748545`, status `issues_found`, 0 critical / 2 warning / 1 info) independently found a narrower characterization of the same underlying regression, plus one unrelated minor finding — I confirmed both are present and unaddressed:

| File | Line(s) | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `plugins/whatsapp/connect.go` | `loginWaiter.wait(serveLoginTimeout)` call (~168-170) | Review's WR-01: characterizes the new blocking wait as delaying kernel BOOT (and hot-apply) by up to 15s per already-linked WhatsApp source, since `pluginhost.Discover`/`Reconcile` launch sources sequentially and `cmd/topos/main.go` doesn't call `http.ListenAndServe` until `NewSupervisor` returns | ⚠️ Warning (review's own rubric) — **this verification's own tracing found the more severe live-mutex manifestation reported as Truth #4's failure above; the review's characterization is real but incomplete** | Confirmed present and unfixed: no commit after `e748545` touches `connect.go`, `supervisor.go`, or `host.go`. No `08-REVIEW-FIX.md` exists for this review pass (the existing `08-REVIEW-FIX.md` in this directory is dated `2026-08-10T16:40:00Z`, addressing a different, earlier review of the 08-09/08-10 wave — confirmed by reading its own content, which fixes `CR-01`/an earlier-numbered `WR-01`/`WR-02` unrelated to this one). |
| `plugins/whatsapp/connect.go` | `client.AddEventHandler(loginWaiter.handleEvent)` (~130-131) | Review's WR-02: the login waiter's event handler is never removed (`RemoveEventHandler`) from the long-lived serve-mode client after `wait()` returns — permanent, harmless-but-wasteful dead weight for the process's remaining lifetime | ⚠️ Warning (review's own rubric) | Confirmed present and unfixed. Low severity — not a correctness bug (the review's own assessment, which this verification agrees with), just an omitted cleanup call. Not counted as a blocking gap on its own. |
| `kernel/supervisor/readiness_test.go` | comment (~56-59) | Review's IN-01: the test's comment states specific numeric timing facts about `defaultFirstRefreshRetryDelays` with no assertion tying the comment to the actual constant, risking silent comment drift | ℹ️ Info | Confirmed present. Cosmetic, non-blocking. |

### Human Verification Required

1. **Real-device re-test of the resume/re-link flow, after the mutex-holding regression (Truth #4) is fixed.**
   **Test:** `make dev`, re-link a real WhatsApp account via the QR flow, and — while that re-link is completing — try to open an item, check a health chip, or manually refresh a *different* source in a separate browser tab/webspace.
   **Expected (once fixed):** The other source's request completes normally and promptly; it must not visibly hang for several seconds waiting on the WhatsApp re-link's own subprocess relaunch.
   **Why human:** Confirms the fix's real-world latency characteristics (this verification's Go-level evidence is definitive for the mechanism, but the felt UX impact of a multi-second freeze is worth a live confirmation once addressed) and requires a live WhatsApp account.

2. **Carried forward from the prior verification cycle: real-device confirmation that a genuine WhatsApp re-link's resulting webspace loads and stays healthy (the original G-08-3/G-08-4 scenario, `08-UAT.md` test 3/its successor).**
   **Test:** Pair or re-link a real WhatsApp account via `make dev`'s QR flow; open the webspace immediately after.
   **Expected:** Loads normally; if the source's very first post-relaunch sync is still in its retry window, the stream loads emptily/populated rather than an outage banner, and settles to `ok` within a few seconds per 08-12's retry schedule.
   **Why human:** Requires a live WhatsApp account and a real kernel run — still not performed in this repo's artifacts (`08-UAT.md` is unchanged since the pre-08-11/08-12 diagnosis).

### Gaps Summary

**Blocking gap (new, found independently in this re-verification):** `plugins/whatsapp/connect.go`'s new bounded login wait (`serveLoginTimeout`, 15s — added by 08-11 to correctly fix G-08-4's plugin-side dishonesty) is reached from inside `kernel/supervisor.Supervisor`'s `SuspendInstance` resume closure while that closure holds `s.mu` for its entire duration. `s.mu` is the same mutex `Host()`/`Coordinator()` take, and every source's item-fetch, health-probe, and manual-refresh HTTP routes resolve through `Host()`/`Coordinator()` on every call. Because the resume closure runs synchronously inside the real re-link flow's terminal poll and cancel handlers, the practical, user-visible effect is: **at the exact moment a real WhatsApp re-link completes, every other configured source (email, paperless, Signal, SilverBullet, ...) becomes unable to serve item-opens, health probes, or manual refreshes for up to 15 seconds.** This directly contradicts phase success criterion 4's "every other source is unaffected" language, for the specific scenario (recovering from a broken WhatsApp session) that criterion exists to cover.

A narrower version of this same defect — kernel BOOT delayed up to 15s when an already-linked WhatsApp source exists (no lock contention possible there since the HTTP server isn't listening yet, but still a real availability regression on every restart) — was independently found by this exact wave's own mandated fresh code review (`08-REVIEW.md`, `e748545`, findings WR-01/WR-02) and left unfixed: there is no `08-REVIEW-FIX.md` for this review pass, and no commit after `e748545` touches any of the implicated files.

**What is genuinely and solidly closed:** G-08-4 itself — the exact defect the user's real-device UAT reported (a freshly relaunched, already-paired instance falsely reporting "Not linked") — is closed across all three of its diagnosed legs (plugin zero-value/explicit-state fix, kernel first-refresh retry, mock fixture), independently re-verified in this session with every relevant test suite re-run live and the plan's own negative control re-executed by hand (not merely re-read from the SUMMARY). Truths 1-3 and 5 hold with no regressions.

**What remains before this phase can be `passed`:** (a) fix the mutex-holding regression — the code review's own suggested fix directions (move the login wait off the synchronous launch path and lean on 08-12's retry instead, or launch sources concurrently, or shorten `serveLoginTimeout`) are a reasonable starting point, but the resume-closure's specific mutex-holds-across-Reconcile shape (distinct from the review's boot-only framing) needs its own accounting in whatever fix is chosen; (b) the still-outstanding real-device confirmation carried from the prior verification cycle.
