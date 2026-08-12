---
phase: 08-whatsapp-conversations-managed-risk
verified: 2026-08-11T13:10:00Z
status: passed
score: 5/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 4/5
  gaps_closed:
    - "G-08-5 (the mutex-hold regression the previous verification cycle found independently): kernel/supervisor/supervisor.go now splits Supervisor.mu (mutation-only) from a new Supervisor.genMu (reader-only, sync.RWMutex). Host() and Coordinator() — which Fetch/ProbeSources/Refresh/RefreshAll (every source's item-open, health-probe and manual-refresh HTTP route) resolve through on every call — take genMu.RLock() only and never touch s.mu. SuspendInstance's resume closure still holds s.mu across the full Host.Reconcile call (mutation serialization unchanged), but readers no longer sit behind it. Confirmed by direct read of supervisor.go at HEAD and by re-running kernel/supervisor/launchlatency_test.go's TestResume_SlowRelaunchDoesNotFreezeOtherSources live in this session (PASS, 4.52s under -race). I independently ran the plan's negative control myself (not trusting the SUMMARY's claim): reverted Host()/Coordinator() to take s.mu, re-ran the same test, confirmed it FAILS ('ProbeSources took 3.71s (>= 2s) ... a health probe of an unrelated source must never block behind a plugin subprocess relaunch'), then restored the fix from a saved backup and reconfirmed the test passes again and the file matches HEAD exactly."
    - "The second, latent defect this fix's own precondition required closing: kernel/pluginhost/host.go's Host gained its own sync.RWMutex (Host.mu) guarding the plugins field, with a snapshot()-returning read path and Reconcile taking the write lock only for the kill-and-commit region (never across the launch loop). Confirmed by direct read and by the full kernel suite passing under -race with a reader and a Reconcile running concurrently."
    - "The plugin-side WR-01/WR-02 findings from 08-REVIEW.md (committed e748545, never fixed at that point) are now closed: plugins/whatsapp/connect.go's serve-mode login wait was moved off startBackgroundClient's synchronous return path onto a background goroutine (WR-01 — an already-linked instance's launch no longer blocks on a live network event before goplugin.Serve), and the login waiter's event handler is now removed on both dial outcomes via a captured loginWaiterID (WR-02). Confirmed by direct read of connect.go at HEAD and by re-running the whole plugins/whatsapp suite live (ok, 1.374s, -race)."
    - "kernel/syncer/scheduler.go's defaultFirstRefreshRetryDelays doc comment and 08-UI-SPEC.md's connecting-row note were corrected so neither still asserts the retired 'handshake means ready' claim the plugin-side fix makes false. Confirmed by direct read."
    - "All four repository gates re-run live and green in this verification session (not merely re-read from 08-15-SUMMARY.md): CGO_ENABLED=0 go build ./... clean; CGO_ENABLED=1 go test ./kernel/... -count=1 -race all ok, zero DATA RACE; make test-portable all 8 modules ok (root+sdk+6 plugins, per the Makefile's test-portable target — note: 08-15-SUMMARY.md's own count of 'all 13 Go modules' does not match the Makefile's 8 cd invocations; a minor SUMMARY self-report inaccuracy, not a gate failure); make dev-check all three cases PASS; make e2e 42 tests passed across 16 spec files in 13.0s, matching 08-15-SUMMARY.md's recorded count exactly."
  gaps_remaining: []
  regressions: []
---

# Phase 8: WhatsApp Conversations (Managed Risk) Verification Report

**Phase Goal:** User's WhatsApp groups for a topic appear in the webspace stream via a linked-device session, and everything else keeps working when that session breaks
**Verified:** 2026-08-11T13:10:00Z
**Status:** passed
**Re-verification:** Yes — after gap-closure wave 7 (plans 08-13, 08-14, 08-15), which close the previous verification cycle's `gaps_found` regression G-08-5 (a slow WhatsApp plugin relaunch holding `Supervisor.mu` across `Host.Reconcile`, freezing every other source's routes for up to 15s during a real re-link — the exact scenario phase success criterion 4 promises stays unaffected).

## Context

The prior `08-VERIFICATION.md` (`status: gaps_found`, 4/5) found G-08-4 (the plugin/kernel launch-readiness AND-gate reported by the first UAT cycle) genuinely closed, but independently discovered a new regression: `SuspendInstance`'s resume closure held `s.mu` — the same mutex `Host()`/`Coordinator()` took — across a real subprocess relaunch that could now last up to 15 seconds (the very login wait added to fix G-08-4). At the exact moment a real WhatsApp re-link completed, every other source's item-open, health-probe and manual-refresh routes froze kernel-wide, directly contradicting success criterion 4.

Plan 08-13 closed the structural half: it split `Supervisor.mu` (mutation-only, unchanged scope) from a new `Supervisor.genMu` (reader-only), so `Host()`/`Coordinator()` never wait behind a mutation's `Reconcile` call regardless of how long a plugin subprocess takes to launch, and it closed a latent, pre-existing data race between `Supervisor.Fetch`'s reader path and a concurrent `Reconcile` by giving `pluginhost.Host` its own internal `sync.RWMutex`. Plan 08-14 closed the root cause the structural fix didn't need to leave in place: it moved the WhatsApp plugin's blocking login wait off the launch path entirely, onto a background goroutine, so an already-linked instance's launch costs what any other plugin's launch costs — closing 08-REVIEW.md's WR-01/WR-02 findings, which had been left open after the prior review pass. Plan 08-15 ran all four repository gates and put a human in front of a real re-link; the human's checkpoint response ("approved — everything works as stated in steps 1-5, including no stray plugin process") discharged both the newly-found mutex regression's felt-latency check and the real-device item carried forward across two prior verification cycles.

**This re-verification independently re-reads every touched file at HEAD (not on SUMMARY's word), re-runs the full kernel suite under `-race`, the whatsapp and mock plugin suites, the specific hermetic gate test, `make test-portable`, `make dev-check`, and `make e2e` live in this session, and additionally re-runs the plan's own negative control by hand** (reverting `Host()`/`Coordinator()` to `s.mu`, confirming the gate fails with the exact predicted symptom, then restoring the fix from a saved backup and confirming it passes again and the file is back to HEAD). All four gates are green, the gate test is proven non-vacuous, and the specific code path that produced the prior regression (`SuspendInstance`'s resume closure, `Host()`/`Coordinator()`, and every route that resolves through them) has been read in full and confirmed to no longer share a lock. The phase goal is achieved.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | User can link webspaces as a WhatsApp device by scanning a QR code, and the session survives service restarts without re-linking | ✓ VERIFIED | `plugins/whatsapp/link.go`, `pairwait.go`, `plugin.go`'s link-flow surface untouched by 08-13/08-14/08-15. Full `plugins/whatsapp` suite re-run live: `ok github.com/davison/topos/plugins/whatsapp 1.374s` (`-race`). Human checkpoint (08-15-SUMMARY.md Task 2, step 4): kernel restart reconnected the WhatsApp source with no second QR prompt. |
| 2 | Messages from WhatsApp groups whose names match the webspace's matching config appear in the stream alongside every other source, using the Phase-4 chat rendering | ✓ VERIFIED | `render.go`, `digest.go`, `plugin.go`'s `Match`/`Eligible` logic untouched by this wave; full whatsapp-plugin suite passes live, including the per-state `Match` regressions. `make e2e`'s `uat-08-whatsapp-qr-link.spec.ts` (13 sub-tests) passes unchanged. |
| 3 | The plugin persists its own message store, so conversations captured while it was running stay browsable regardless of what the WhatsApp desktop app retains | ✓ VERIFIED | `messagestore.go` untouched by this wave; `TestMessageStore_AppendIdempotent`/`TestMessageStore_ChatIsolationAndOrdering` re-run live, pass. |
| 4 | De-link, ban, or session expiry surfaces as an explicit plugin-health error in the UI while previously captured messages remain browsable and **every other source is unaffected** | ✓ VERIFIED — the prior verification cycle's regression is closed | `kernel/supervisor/supervisor.go` now separates `s.mu` (mutation, unchanged scope — still held by `SuspendInstance`'s resume closure across the whole `Host.Reconcile` call) from `s.genMu` (reader-only): `Host()`/`Coordinator()` take `genMu.RLock()` only, so `Fetch`/`ProbeSources`/`Refresh`/`RefreshAll` — which back every source's item-open/health-probe/manual-refresh routes — never wait on a resume closure's subprocess relaunch. Confirmed by direct source read and by `kernel/supervisor/launchlatency_test.go`'s `TestResume_SlowRelaunchDoesNotFreezeOtherSources`, re-run live (PASS, 4.52s, `-race`) and independently negative-controlled by this verifier: reverting `Host()`/`Coordinator()` to `s.mu` makes the same test FAIL with `ProbeSources took 3.71s (>= 2s)`; restoring the fix makes it pass again. Human checkpoint (08-15-SUMMARY.md Task 2, step 2) additionally confirms the felt behavior: during a real re-link's completion window, a second tab's item-open, health chip and manual refresh for an unrelated source all answered promptly with no visible hang. |
| 5 | G-08-4 (the plugin/kernel/fixture launch-readiness AND-gate from the prior UAT cycle) stays closed after this wave's own changes | ✓ VERIFIED | `healthStateConnecting`'s zero-value fix and its explicit pre-dial assignment (`plugins/whatsapp/connect.go`) are untouched by 08-13/08-14. `kernel/syncer/scheduler.go`'s bounded first-refresh retry is untouched in code (comment-only diff, confirmed via the same `git diff -U0 | grep -vc '^[-+][[:space:]]*//'` check the plan mandated). Human checkpoint (08-15-SUMMARY.md Task 2, step 3): the re-linked WhatsApp webspace loaded normally, no outage banner, no stale pairing instruction, settled healthy within seconds. |

**Score:** 5/5 truths verified by direct, live-re-run codebase evidence, independent negative-controlled testing, and a human-approved real-device checkpoint.

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `kernel/supervisor/supervisor.go` | `genMu sync.RWMutex` field; `Host()`/`Coordinator()` take `genMu.RLock()` only; every write to `s.host`/`s.coord` takes `genMu.Lock()` for the assignment alone; `s.mu` keeps its exact prior scope for `Apply`/`SuspendInstance`/resume/`Shutdown` | ✓ VERIFIED | Read in full at HEAD; matches 08-13-PLAN's stated shape exactly (verified line-by-line above). |
| `kernel/pluginhost/host.go` | `Host.mu sync.RWMutex` guarding only `plugins`; `snapshot()` helper; `Reconcile` takes `RLock` only to snapshot, launches with no lock held, takes `Lock()` only for kill-and-commit | ✓ VERIFIED | Read in full at HEAD; matches plan exactly. `Plugins()`/`ProbeSources`/`byInstance` all route through `snapshot()`, returning a defensive copy. |
| `kernel/supervisor/launchlatency_test.go` | `TestResume_SlowRelaunchDoesNotFreezeOtherSources`, the hermetic real-subprocess gate for criterion 4 | ✓ VERIFIED | Present, re-run live: PASS (4.52s, `-race`). Negative control independently re-verified by this verifier (see above) — fails with the exact predicted G-08-5-shaped symptom when the fix is reverted. |
| `plugins/mock/readiness.go` | `launchDelayEnvVar` const, `launchDelayFromEnv` parse-or-fail-loud helper, sibling to the existing readiness-window fixture | ✓ VERIFIED | Present; `TestLaunchDelayFromEnv` (6 subtests) re-run live, passes. |
| `plugins/mock/main.go` | Calls `launchDelayFromEnv(os.Getenv)` before `goplugin.Serve`, sleeping or exiting non-zero on a malformed value | ✓ VERIFIED | Read in full; drives the gate test's controllable slow launch. |
| `plugins/whatsapp/connect.go` | Login wait moved onto a background goroutine dispatched after a successful dial; `loginWaiterID` captured and `RemoveEventHandler` called on both dial outcomes; `serveLoginTimeout`'s doc comment rewritten for its new role | ✓ VERIFIED | Read in full at HEAD; matches 08-14-PLAN exactly (verified above). |
| `plugins/whatsapp/connect_test.go` | `TestStartBackgroundClient_ConnectingBeforeDialAndLoginWaitOffTheLaunchPath`, an AST guard pinning the wait off the launch path and both handler-removal sites | ✓ VERIFIED | Present, re-run live, passes; old test name confirmed absent from the run output. |
| `kernel/syncer/scheduler.go` | `defaultFirstRefreshRetryDelays`' doc comment corrected — no longer claims the plugin's launch absorbs the login round trip | ✓ VERIFIED | Read in full; comment-only diff confirmed. |
| `.planning/phases/08-whatsapp-conversations-managed-risk/08-UI-SPEC.md` | Connecting-row note amended additively, naming plan 08-14 and gap G-08-5 | ✓ VERIFIED | Present; taxonomy table itself untouched. |
| `docs/testing.md` | `WEBSPACES_MOCK_LAUNCH_DELAY_MS` documented; dated `What changed` entries | ✓ VERIFIED | Present, read in full. |
| `08-15-SUMMARY.md` | Recorded output of all four gates plus verbatim human observations | ✓ VERIFIED, AND independently reproduced by this verifier | All four gates re-run live in this session with matching results (`make e2e`: 42 tests / 16 specs, identical to the recorded count). |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `Supervisor.Host()`/`Coordinator()` | `Fetch`/`ProbeSources`/`Refresh`/`RefreshAll` for every source | `genMu.RLock()`, never `s.mu` | ✓ WIRED | Confirmed by direct source read (no reference to `s.mu` in either method body) and by the hermetic gate test, re-run live and independently negative-controlled. |
| `SuspendInstance`'s resume closure's `s.mu.Lock()` | `Host.Reconcile`'s subprocess relaunch | `s.mu`, held for the closure's whole duration (unchanged scope) | ✓ WIRED, no longer harmful | This link is intentional and unchanged from before — what changed is that no *reader* path shares this lock any more, so holding it across a slow relaunch no longer freezes anything else. |
| `Host.Reconcile`'s launch loop | no lock held during subprocess launch | `h.mu.RLock()` released before the `toLaunch` loop; `h.mu.Lock()` taken only for kill-and-commit | ✓ WIRED | Confirmed by source read; the launch loop is not lexically inside any region holding `h.mu.Lock()`. |
| `startBackgroundClient`'s successful dial | `goplugin.Serve` reached immediately | the login wait dispatched via `go func(){...}()`, not awaited | ✓ WIRED | Confirmed by source read: the function returns `nil` immediately after dispatching the goroutine; `loginWaiter.wait(serveLoginTimeout)` only appears inside the `go` statement's literal. |
| `client.AddEventHandler(loginWaiter.handleEvent)`'s returned id | `client.RemoveEventHandler` on both the dial-error branch and the background goroutine's deferred call | `loginWaiterID` | ✓ WIRED | Confirmed by source read: both call sites use the same captured id. |
| `WEBSPACES_MOCK_LAUNCH_DELAY_MS` set on the kernel process | the mock subprocess's own env | `pluginhost.launch`'s `os.Environ()` | ✓ WIRED | Confirmed by source read and by `TestResume_SlowRelaunchDoesNotFreezeOtherSources` driving a real 4-second-slow subprocess through it via `t.Setenv`. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Full Go workspace build | `CGO_ENABLED=0 go build ./...` | clean | ✓ PASS |
| Full kernel test suite under the race detector | `CGO_ENABLED=1 go test ./kernel/... -count=1 -race` | all packages `ok`, zero `DATA RACE` | ✓ PASS |
| Full whatsapp plugin suite under the race detector | `cd plugins/whatsapp && go test ./... -count=1 -race -v` | all pass, `ok` 1.374s | ✓ PASS |
| Full mock plugin suite | `cd plugins/mock && go test ./... -count=1 -v` | all pass, incl. `TestLaunchDelayFromEnv` (6 subtests) and `TestReadinessWindowFromEnv` (6 subtests) | ✓ PASS |
| Cross-source isolation gate | `go test ./kernel/supervisor/... -run TestResume_SlowRelaunchDoesNotFreezeOtherSources -v -count=1 -race` | PASS (4.52s) | ✓ PASS |
| G-08-4 launch-readiness gate | `go test ./kernel/supervisor/... -run TestBoot_FirstRefreshSurvivesAPluginLaunchReadinessWindow -v -count=1 -race` | PASS (2.08s) | ✓ PASS |
| **Negative control (run independently by this verifier, not trusting either SUMMARY)** | Reverted `Host()`/`Coordinator()` to take `s.mu` instead of `genMu`, re-ran the isolation gate | **FAILS** as expected: `ProbeSources took 3.712573773s (>= 2s) ... a health probe of an unrelated source must never block behind a plugin subprocess relaunch`; restored from backup, confirmed `git diff` clean and the test passes again | ✓ PASS (gate proven non-vacuous) |
| Full workspace `make test-portable` | `make test-portable` | all 8 Go modules `ok` (root+sdk+6 plugins; 08-15-SUMMARY.md's count of "13" does not match the Makefile's 8-module test-portable target — flagged as a minor inaccuracy below, not a gate failure) | ✓ PASS |
| Hermetic dev-guard smoke | `make dev-check` | all three cases PASS | ✓ PASS |
| Full Playwright e2e suite | `make e2e` | 42 tests passed across 16 spec files in 13.0s | ✓ PASS (matches 08-15-SUMMARY.md's recorded count exactly) |
| Debt-marker scan on all files touched by this wave | `grep -n -E "TBD\|FIXME\|XXX\|TODO\|HACK\|PLACEHOLDER"` per file | no matches across all 11 touched files | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes declared by this phase's plans or ROADMAP success criteria. Step 7c: SKIPPED (no declared probes).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| SRC-03 | 08-01 … 08-15 | WhatsApp plugin runs as a whatsmeow linked device with its own persistent message store; degrades gracefully on de-link/ban; matches on group names | ✓ SATISFIED | All four success criteria hold with regression-free automated evidence plus a human-approved real-device checkpoint. See Observable Truths 1-5 above. |

No orphaned requirements — SRC-03 is the only ID `.planning/ROADMAP.md` maps to Phase 8, and all fifteen plans (including 08-13/08-14/08-15) declare `requirements: [SRC-03]`.

**Bookkeeping note (not a phase-goal gap):** `.planning/REQUIREMENTS.md` line 30's checkbox currently reads `[x]` for SRC-03, but line 93's traceability table still reads `| SRC-03 | Phase 8 | Gaps Found |`. Git history shows the checkbox was flipped back to `[x]` by commit `c36ff667` ("docs(08-14): complete serve-mode login wait launch-path plan") without the accompanying traceability-table update commit `c42ad3b` had made when it correctly set both to reflect the then-open gap. This verification's own result (`passed`) makes the checkbox's claim now true, but the traceability table's stale "Gaps Found" text should be updated to "Complete" as part of closing out this verification — flagging this so the orchestrator reconciles both lines together rather than leaving the checkbox and the table disagreeing.

### Anti-Patterns Found

No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers in any of the files touched by 08-13/08-14/08-15.

`08-REVIEW.md` (committed `58d14b5`) found 0 critical, 2 warnings, 2 info against this exact wave. Both warnings are accepted, tracked risk rather than blockers:

| File | Finding | Severity | Disposition |
| --- | --- | --- | --- |
| `plugins/whatsapp/connect.go` / `kernel/syncer/scheduler.go` | WR-01: the first post-relaunch `Match` now races a ~7s retry budget (`{2s, 5s}`) against a live login round trip never validated against a real degraded network | ⚠️ Warning (review's own rubric) — accepted, tracked | The review's own suggested mitigation (a human real-device gate) is exactly what plan 08-15's checkpoint performed and the human approved. Genuinely degraded-network behavior (a slow/lossy connection specifically) was not part of that checkpoint's scripted steps — this residual risk is real but narrow, matches the review's own framing, and does not block this phase; it is the kind of risk this verification surfaces rather than silently absorbs. |
| `kernel/pluginhost/host.go` | WR-02: `Host.Reconcile` has no *internal* protection against two concurrent `Reconcile` calls — correctness still depends on `kernel/supervisor`'s external convention (every caller holds `s.mu`) | ⚠️ Warning (review's own rubric) | Pre-existing limitation, not introduced by this wave (Host had no internal locking at all before it). This wave's own `h.mu` closes the reader-vs-single-Reconcile race, which is the concern G-08-5 was about; concurrent-Reconcile safety was never claimed to be newly internal. Not a regression, not a blocker. |
| `plugins/whatsapp/connect.go` | IN-01: the new background goroutine adds a third unsynchronized concurrent writer to `p.logOut` | ℹ️ Info | Log-hygiene only, not correctness; pre-existing pattern (event-handler and gRPC paths already wrote concurrently). |
| `kernel/index/store.go` | IN-02: `SyncRunsForSourceForTesting` is an unbounded, unenforced-test-only production symbol | ℹ️ Info | Matches an already-established house convention (`config.NewStoreForTesting`); not a regression. |

Neither warning is a blocker: both are narrow, previously-existing-in-kind risk surfaces the review itself classifies as `warning`, not `critical`, and neither contradicts any of the five observable truths verified above.

### Human Verification Required

None. Plan 08-15's checkpoint (`type="checkpoint:human-verify" gate="blocking"`) was run against a real WhatsApp account and a second, healthy non-WhatsApp source, covering exactly the observation no harness in this repo can make: whether every other source stays responsive during a live re-link's completion window. The human's verbatim response ("approved - everything works as stated in steps 1-5, including no stray plugin process") is recorded in `08-15-SUMMARY.md` and discharges both the carried-forward real-device item from the previous verification cycle and this cycle's own regression (G-08-5)'s felt-latency check.

### Gaps Summary

None. The regression this verification cycle exists to re-check — `SuspendInstance`'s resume closure holding a lock that also gated every other source's routes — is closed at the structural level (an independent negative control confirms the isolation gate is non-vacuous and catches the exact prior defect) and confirmed at the felt-UX level by a human-approved real-device checkpoint. All five observable truths for phase success criteria 1-4 (plus the still-standing G-08-4 closure) hold with regression-free, independently-reproduced evidence. Two narrow, review-flagged warnings (WR-01's untested-degraded-network retry margin, WR-02's convention-only Reconcile exclusivity) remain as accepted, tracked risk — neither blocks the phase goal, and both are exactly the kind of residual risk this report surfaces rather than silently drops.

One non-blocking bookkeeping item is flagged above: `.planning/REQUIREMENTS.md`'s traceability table (line 93) still reads "Gaps Found" for SRC-03 despite the checkbox (line 30) already reading `[x]` — this verification's `passed` result should be used to update the table's text to "Complete" so the two lines agree.

---

_Verified: 2026-08-11T13:10:00Z_
_Verifier: Claude (gsd-verifier)_
