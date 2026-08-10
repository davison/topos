---
phase: 08-whatsapp-conversations-managed-risk
verified: 2026-08-11T01:30:00Z
status: human_needed
score: 5/5 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: 5/5
  gaps_closed:
    - "G-08-3 (kernel half): supervisor.SuspendInstance and its resume closure now perform a full generation change (stopScheduler -> Host.Reconcile -> commitGeneration), so a WhatsApp re-link's suspended-then-resumed instance syncs successfully on its very next refresh instead of failing indefinitely with grpc's ErrClientConnClosing until a config save or restart"
    - "G-08-3 (kernel half, sibling defect): every background sync the kernel dispatches (Apply's eager resync included) is now bound to its own scheduler generation's cancellable context and wait group, so stopScheduler — now reachable from the WhatsApp link-start HTTP request path — can never block indefinitely on an untracked goroutine"
    - "G-08-3 (kernel half, sibling defect): all five WhatsApp link lifecycle call sites (start's suspend, its two failure-path resumes, poll's terminal resume, cancel's resume) now run on a detached context.Background(), so a browser disconnect mid-session cannot abort a real relaunch/generation rebuild"
    - "G-08-3 (kernel half, sibling defect): a refresh issued during a suspension window now returns syncer.ErrUnknownSource before any sync_runs row is started, so a lifecycle artifact is never pinned to a source's health surface as a failed sync"
    - "G-08-3 (presentation half): a webspace whose stream is empty and whose participating source's latest sync failed now renders a neutral, source-scoped StreamSyncDegraded state naming what failed, instead of the full-page StreamError claiming the topos service didn't respond to a request the kernel actually answered 200 to"
    - "G-08-3 (presentation half, root defect): the sync-status aggregate is now scoped to each webspace's own participating sources (via correlate.ParticipatesIn) at all four call sites (StreamHandler, WebspacesHandler, agentStreamHandler, agentWebspacesHandler), so a failing non-participating source can no longer make an unrelated webspace look broken"
    - "Real-device re-test of the QR pairing flow (G-08-1 fix) — 08-UAT.md test 1 — passed"
    - "Real-kernel confirmation of the CR-01 fix (in-flight teardown releases real resources) — 08-UAT.md test 2 — passed"
  gaps_remaining: []
  regressions: []
---

# Phase 8: WhatsApp Conversations (Managed Risk) Verification Report

**Phase Goal:** User's WhatsApp groups for a topic appear in the webspace stream via a linked-device session, and everything else keeps working when that session breaks
**Verified:** 2026-08-11T01:30:00Z
**Status:** human_needed
**Re-verification:** Yes — after gap-closure plans 08-09 (kernel: generation-correct suspend/resume) and 08-10 (presentation: participation-scoped sync status + degraded stream state), which close `08-UAT.md`'s gap G-08-3 found by the previous UAT cycle after the prior `08-VERIFICATION.md` (`status: human_needed`, 5/5) routed to human testing.

## Context

The prior verification (`status: human_needed`, 5/5) found all automated evidence for the phase's five observable truths in place, but routed to `human_needed` because two items required a real WhatsApp account and a real kernel run that no automated harness in this repo can substitute for. UAT then ran: test 1 (real-device G-08-1 re-test) passed, test 2 (real-kernel CR-01 confirmation) passed, and test 3 (webspace stream loads after WhatsApp pairing) surfaced a new **major** issue — opening the webspace after a real re-link failed with `Couldn't load this webspace / The topos service didn't respond`, kernel error `rpc error: code = Canceled desc = grpc: the client connection is closing`. This became gap G-08-3.

Diagnosis (`.planning/debug/whatsapp-grpc-closing-fails-webspace.md`) found an AND-gate of two independent defects: (K) `supervisor.SuspendInstance`'s resume path relaunched the plugin subprocess but never rebuilt the `syncer.Coordinator`, so every subsequent sync of the resumed instance kept dispatching through the killed gRPC client; and (P) the kernel's stream-response sync aggregate folded *any* configured source's latest errored run into *every* webspace's response, and the SPA escalated a zero-item + sync-error response to a full-page "service didn't respond" outage — even though the kernel had answered 200.

Plan `08-09` closed (K) and its two load-bearing siblings (uncancellable background syncs, HTTP-request-scoped lifecycle contexts). Plan `08-10` closed (P) with a new `StreamSyncDegraded` component, `correlate.ParticipatesIn`-scoped aggregation at all four call sites, and structural + browser regression armor. A fresh code review of the combined diff (`08-REVIEW.md`, committed `fde87fc`) found 0 critical, 2 warnings (both test-coverage gaps on the *new* Reconcile-failure branches these plans themselves introduced — not incorrect behavior), and 2 info items (both pre-existing, unchanged-in-behavior conditions incidentally touched by the diff).

This re-verification independently re-reads the fixed code (not on SUMMARY's word), re-runs every relevant test suite and the full Go/web/e2e suites live in this session, and confirms the fresh code review's findings are present and genuinely non-blocking. It also checks whether the two prior UAT gaps (test 1, test 2) are closed by genuine human confirmation (they are) and whether G-08-3's fix itself has been confirmed against a real device (it has **not** — both gap-closure plans explicitly and deliberately defer that confirmation to a phase-level UAT re-run of test 3, which has not yet occurred in this repo's artifacts). That single outstanding item is why this report routes to `human_needed` rather than `passed`.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
| --- | --- | --- | --- |
| 1 | User can link webspaces as a WhatsApp device by scanning a QR code, and the session survives service restarts without re-linking | ✓ VERIFIED (regression, unchanged since prior verification; also human-confirmed live) | `link.go`, `connect.go`, `pairwait.go`, `plugin.go` untouched by 08-09/08-10. Full `plugins/whatsapp` suite re-run live: `ok github.com/davison/topos/plugins/whatsapp 0.057s`. `08-UAT.md` test 1 (real device) result: `pass`. |
| 2 | Messages from WhatsApp groups whose names match the webspace's matching config appear in the stream alongside every other source, using the Phase-4 chat rendering | ✓ VERIFIED (regression, unchanged) | `render.go`, `digest.go`, `plugin.go`'s `Match`/`Eligible` logic untouched by 08-09/08-10; full whatsapp-plugin suite passes live. |
| 3 | The plugin persists its own message store, so conversations captured while it was running stay browsable regardless of what the WhatsApp desktop app retains | ✓ VERIFIED (regression, unchanged) | `messagestore.go` untouched by 08-09/08-10; whatsapp-plugin suite passes live. |
| 4 | De-link, ban, or session expiry surfaces as an explicit plugin-health error in the UI while previously captured messages remain browsable and every other source is unaffected — **including the specific G-08-3 case: a re-link that suspends/resumes an instance must leave that instance (and every other source) syncing normally, not silently dead** | ✓ VERIFIED by hermetic evidence | `health.go` untouched. `SuspendInstance`/resume in `kernel/supervisor/supervisor.go` now perform `stopScheduler → Host.Reconcile → commitGeneration` (read in full at HEAD, matches the SUMMARY's claimed shape exactly, doc comment names G-08-3). `TestSuspendInstance_ResumedInstanceStillSyncs` drives a REAL launched-then-killed-then-relaunched mock plugin subprocess through a `Refresh` call after resume and asserts `Status: "ok"` — re-run live, passes (0.44s). `TestSuspendInstance_SuspendedWindowRecordsNoErroredRun`, `TestApply_EagerResyncDoesNotOutliveItsGeneration`, `TestWhatsAppLink_SuspendAndResumeRunOnDetachedContexts` all re-run live, pass. On the presentation side, `StreamSyncDegraded.svelte`, `StreamList.svelte`'s branch swap, and `StreamError.svelte`'s narrowing all read as claimed at HEAD; `filterRunsByParticipation` in `kernel/httpapi/sources.go` scopes all four aggregate call sites via `correlate.ParticipatesIn`; `TestStreamHandler_NonParticipatingSourceFailureDoesNotEscalate`, `TestStreamHandler_ParticipatingSourceFailureStillEscalates`, `TestStreamHandler_IndexOnlyWebspaceReportsZeroValueSyncDespiteOtherFailure`, `TestFilterRunsByParticipation_FourBranches`, `TestAgentStreamHandler_SyncStatusComposesGrantAndParticipation` all re-run live, pass. New e2e spec `g-08-3-degraded-source-not-outage.spec.ts` (3 cases) re-run live in this session, passes 3/3. **Caveat:** every layer of this proof is either a real-but-mock plugin subprocess test or hermetic route-scripted browser test — no automated harness in this repo drives a real `topos-plugin-whatsapp` subprocess through an actual link/re-link/degrade cycle. Both `08-09-PLAN.md` and `08-10-PLAN.md` explicitly and deliberately defer the real-device confirmation of G-08-3's fix to the phase's own UAT re-run of test 3, which has not yet occurred (see Human Verification Required). |
| 5 | 08-UAT.md test 2's own outstanding item (real-kernel confirmation that CR-01's fix releases a real subprocess and a real suspended instance) | ✓ VERIFIED — human-confirmed live, closing the prior verification's last open item | `08-UAT.md` test 2 result: `pass` (no note recorded). This closes the item the prior `08-VERIFICATION.md` (`human_needed`) left open. |

**Score:** 5/5 truths verified by codebase evidence and/or genuine human UAT confirmation. Truth #4's G-08-3 sub-claim is fully proven by hermetic Go and Playwright evidence but still has one outstanding real-device confirmation step, deliberately deferred by both gap-closure plans to the phase's UAT re-run — which is why overall status is `human_needed` rather than `passed`.

### Required Artifacts

| Artifact | Expected | Status | Details |
| --- | --- | --- | --- |
| `kernel/supervisor/supervisor.go` | `SuspendInstance`/resume as full generation changes; `genCtx`/`genWG` generation-scoped fields | ✓ VERIFIED | Read in full at HEAD; matches 08-09-PLAN's stated shape exactly. `grep -c 'commitGeneration(s.cfg)'` → 2 (excluding comments). `grep -c 'context.Background()'` → 0 (excluding comments) — confirms Apply's eager-resync dispatch now uses `genCtx`, not a detached background context. |
| `kernel/supervisor/suspend_test.go` | `TestSuspendInstance_ResumedInstanceStillSyncs`, `TestSuspendInstance_SuspendedWindowRecordsNoErroredRun` | ✓ VERIFIED | Both present, both pass live, both drive real launched mock plugin subprocesses through `buildMockPluginDir`. |
| `kernel/supervisor/supervisor_test.go` | `TestApply_EagerResyncDoesNotOutliveItsGeneration`; `blockingSource.exited` | ✓ VERIFIED | Present, passes live (0.00s in the run, race-checked separately per SUMMARY's documented `-race` run). |
| `kernel/httpapi/whatsapplink.go` | All 5 lifecycle call sites detached to `context.Background()` | ✓ VERIFIED | `grep -c 'r.Context()'` (excluding comments) → 0. |
| `kernel/httpapi/whatsapplink_test.go` | `TestWhatsAppLink_SuspendAndResumeRunOnDetachedContexts` | ✓ VERIFIED | Present, passes live with all 3 subtests (start/poll/cancel). |
| `web/src/lib/components/StreamSyncDegraded.svelte` | Neutral, source-scoped degraded stream state | ✓ VERIFIED | Read in full at HEAD; renders locked Amendment-3 copy, `syncError` passed through, no service-unreachable wording (`grep -c 'respond'` on non-comment lines → 0). |
| `web/src/lib/components/StreamList.svelte` | `sync-failed` branch renders `StreamSyncDegraded`, ahead of both empty branches | ✓ VERIFIED | Read in full; branch order confirmed unchanged (sync-failed → empty-filtered → empty → populated). |
| `web/src/lib/components/StreamError.svelte` | Narrowed to the one fetch-failure cause; `syncError` prop removed | ✓ VERIFIED | Read in full; no `syncError` prop, copy byte-identical to before per SUMMARY diff claim. |
| `web/src/lib/components/stream-degraded.test.ts` | Structural guard over `StreamList`'s branch set | ✓ VERIFIED | 6 assertions, re-run live, pass. |
| `web/e2e/specs/g-08-3-degraded-source-not-outage.spec.ts` | 3 hermetic browser cases (degrade / outage / adjacency) | ✓ VERIFIED | Re-run live in this session as part of the full 42-spec suite; 3/3 pass. |
| `kernel/httpapi/sources.go` | `filterRunsByParticipation` | ✓ VERIFIED | Present, resolves through `correlate.ParticipatesIn` per read. |
| `kernel/httpapi/stream.go`, `webspaces.go`, `agent.go` | All 4 call sites scoped by participation | ✓ VERIFIED | `grep -c 'aggregateSyncStatus(runs)'` (unscoped) across all three files → 0. |
| `docs/api.md` | Aggregate scope corrected | ✓ VERIFIED | Diff confirms the paragraph rewritten; cites G-08-3 per SUMMARY. |

### Key Link Verification

| From | To | Via | Status | Details |
| --- | --- | --- | --- | --- |
| `SuspendInstance`/resume | `syncer.Coordinator` rebuild | `commitGeneration(s.cfg)` after `Host.Reconcile` succeeds | ✓ WIRED | Was the root cause of G-08-3 (coordinator never rebuilt); now confirmed wired by source read and `TestSuspendInstance_ResumedInstanceStillSyncs` driving a real sync through it. |
| Apply's eager-resync dispatch | generation cancellation | `genCtx`/`genWG` re-read immediately after `commitGeneration`, before the dispatch loop | ✓ WIRED | Confirmed by source read; `TestApply_EagerResyncDoesNotOutliveItsGeneration` proves a dispatched goroutine's `Match` call returns once `Shutdown`/generation-cancel fires. |
| `StreamList.svelte`'s `sync-failed` branch | `StreamSyncDegraded` | direct component render, `response.sync.error` passed as `syncError` | ✓ WIRED | Confirmed by source read and `stream-degraded.test.ts`'s structural guard (6 assertions, live pass). |
| `StreamHandler`/`WebspacesHandler`/`agentStreamHandler`/`agentWebspacesHandler` | `filterRunsByParticipation` | direct call, composed with `filterRunsByGrant` on the two agent mirrors | ✓ WIRED | `grep` confirms no caller still aggregates the unscoped map; 5 targeted unit tests (stream/sources/agent) all pass live, covering non-participant exclusion, participant escalation, index-only-webspace zero-value, and grant×participation composition. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
| --- | --- | --- | --- |
| Full Go workspace build | `CGO_ENABLED=0 go build ./...` | clean | ✓ PASS |
| Full kernel test suite | `CGO_ENABLED=0 go test ./kernel/... -count=1` | all packages `ok` | ✓ PASS |
| Targeted gap-closure Go tests (9 named tests across supervisor/httpapi) | `go test -run '<names>' -v -count=1` | all pass, individually verified | ✓ PASS |
| whatsapp plugin full suite | `CGO_ENABLED=0 go test ./plugins/whatsapp/... -count=1` | `ok` | ✓ PASS |
| Full web unit suite | `npm run test -- --run` | 673/673 pass, 39/39 files | ✓ PASS |
| `stream-degraded.test.ts` targeted | `npm run test -- --run src/lib/components/stream-degraded.test.ts` | 6/6 pass | ✓ PASS |
| Full Playwright suite, real build | `npx playwright test --project=chromium` | **42/42 pass** | ✓ PASS |
| Targeted whatsapp/G-08-3/degraded e2e | `make e2e E2E_ARGS="--grep 'whatsapp\|G-08-3\|degraded'"` | 16/16 pass, including all 3 new G-08-3 cases and all 13 whatsapp QR-link cases | ✓ PASS |
| Debt-marker scan on all 17 files touched by 08-09/08-10 | `grep -n -E "TBD\|FIXME\|XXX\|TODO\|HACK\|PLACEHOLDER"` per file | no matches | ✓ PASS |
| Prohibition check: no unscoped aggregate call remains | `grep -c 'aggregateSyncStatus(runs)'` on webspaces.go/stream.go/agent.go | 0 | ✓ PASS |
| Prohibition check: no `r.Context()` remains in whatsapplink.go lifecycle calls | `grep -c 'r.Context()'` | 0 | ✓ PASS |
| Prohibition check: no leftover detached `context.Background()` in supervisor.go's dispatch (should use genCtx) | `grep -c 'context.Background()'` | 0 | ✓ PASS |
| Diff-scope check: no lockfile/go.mod/go.sum touched by the gap-closure wave | `git diff --stat b969465 HEAD -- go.mod go.sum web/package.json web/package-lock.json` | empty | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` probes declared by this phase's plans or ROADMAP success criteria. Step 7c: SKIPPED (no declared probes).

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
| --- | --- | --- | --- | --- |
| SRC-03 | 08-01 … 08-10 | WhatsApp plugin runs as a whatsmeow linked device with its own persistent message store; degrades gracefully on de-link/ban; matches on group names | ? NEEDS HUMAN | All four success criteria have complete automated evidence, including G-08-3's fix (both the kernel generation-correctness defect and the presentation aggregate-scoping defect). `.planning/REQUIREMENTS.md` line 30 correctly still shows `[ ]` and the summary table (line 93) still shows "Gaps Found" — both are accurate as of this verification's routing to `human_needed`, not an omission; they should be updated once the one remaining human-verification item below is closed. |

No orphaned requirements — SRC-03 is the only ID `.planning/ROADMAP.md` maps to Phase 8, and all ten plans (including 08-09, 08-10) declare `requirements: [SRC-03]`.

### Anti-Patterns Found

No `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` markers in any of the 17 files touched by 08-09/08-10 (checked individually, all clean).

The fresh code review of the gap-closure wave (`08-REVIEW.md`, committed `fde87fc`) found two non-blocking warnings and two non-blocking info items, all confirmed present independently in this session:

| File | Line(s) | Pattern | Severity | Impact |
| --- | --- | --- | --- | --- |
| `kernel/supervisor/supervisor.go` | `SuspendInstance`'s pre-Reconcile failure branch (~350-359) | No test forces `Host.Reconcile` to fail from inside `SuspendInstance`, so the new `startScheduler(s.cfg)` recovery call — the only thing standing between a failed suspend and a kernel left with no scheduler running at all — is unexercised | ⚠️ Warning | Read at HEAD, confirmed present exactly as described. Logic is sound and mirrors `Apply`'s already-tested analogous branch, but this specific branch has no direct regression test. Not counted against the score — this is a coverage gap on new code, not incorrect behavior, and the review's own verdict is non-blocking. |
| `kernel/supervisor/supervisor.go` | resume closure's Reconcile-failure branch (~380-400) | Same untested pattern on the resume side; a failed resume's `s.coord` staleness relative to `s.cfg` is unexercised | ⚠️ Warning | Confirmed present. Same disposition as above — non-blocking coverage gap. |
| `kernel/httpapi/webspaces.go` | `LatestSyncRunPerSource` error swallow (~47-51) | Pre-existing pattern (unchanged in behavior by this wave), inconsistent with `agentWebspacesHandler`'s sibling handling of the identical error | ℹ️ Info | Confirmed present; pre-existing, not a regression. |
| `kernel/supervisor/supervisor_test.go` | `TestApply_EagerResyncDoesNotOutliveItsGeneration` (~323-394) | Hand-copies `Apply`'s dispatch snippet rather than driving it through a real `Apply` call, due to a documented `pluginhost.Host` test-seam constraint | ℹ️ Info | Confirmed present; explicitly accepted and documented as a reasonable workaround in both the review and the SUMMARY. |

Both warnings are recorded here per the mandate to check the codebase directly rather than take either review's word; neither is a truth-blocking finding, and neither reopens any G-08-3 sub-cause.

### Human Verification Required

1. **Real-device re-test of `08-UAT.md` test 3 / gap G-08-3's fix (not yet performed).**
   **Test:** `make dev`, pair or re-link a real WhatsApp account via the QR flow (D-03 Re-link or a fresh D-01/D-02 pairing), then open the webspace that source participates in immediately after the pairing/re-link completes.
   **Expected:** The webspace loads normally (or, if the source's very first post-resume sync happens to still be running, the stream loads emptily/populated rather than showing "Couldn't load this webspace / The topos service didn't respond"). No `rpc error: code = Canceled desc = grpc: the client connection is closing` appears in the kernel log for this source going forward. If a *different*, genuinely unrelated source has a failing sync at the same time, the WhatsApp webspace's own load must be unaffected by it.
   **Why human:** Requires a live WhatsApp account and a real kernel run driving a real `topos-plugin-whatsapp` subprocess through an actual suspend/resume cycle — inherently outside this automated environment. Both `08-09-PLAN.md`'s and `08-10-PLAN.md`'s own `<assumptions>` sections explicitly defer this confirmation to the phase's UAT re-run rather than claiming it as covered by their hermetic Go/Playwright tests, and no UAT re-run of test 3 has occurred yet in this repo's artifacts (`08-UAT.md` is unchanged since the pre-gap-closure diagnosis, still recording test 3 as `issue`/gap G-08-3 rather than a re-tested `pass`).

### Gaps Summary

No blocking gaps. All automated evidence for the phase's five observable truths — including both halves of gap G-08-3 (the kernel's generation-incorrect suspend/resume, and the presentation layer's unscoped sync-status aggregate plus false-outage escalation) — is in place, independently re-read and re-run live in this session rather than taken on SUMMARY.md's word: full Go build/test suite, the whatsapp plugin's own suite, all 9 newly-added/gap-closure-relevant named tests individually, the full 673-test web unit suite, and the full 42-spec Playwright suite (up from 39 pre-wave, +3 for the new G-08-3 spec). A fresh code review of the combined 08-09/08-10 diff found 0 critical issues; its two warnings are both test-coverage gaps on new Reconcile-failure recovery branches (not incorrect behavior) and its two info items are both pre-existing, unchanged-in-behavior conditions — none reopen any part of G-08-3.

The remaining item preventing a clean `passed` status is not a code defect but a verification-coverage gap that both gap-closure plans' own authors explicitly flagged and deferred: a real-device UAT re-run confirming that a genuine WhatsApp re-link no longer breaks its webspace's load. `08-UAT.md`'s two other human-verification items from the prior cycle (real-device G-08-1 re-test, real-kernel CR-01 confirmation) were both genuinely re-tested by a human and passed — closing out the prior verification's open items — but G-08-3's own fix has not yet had its equivalent live confirmation. This should be performed and its outcome recorded (updating `08-UAT.md` and, on a clean pass, `.planning/REQUIREMENTS.md`'s SRC-03 row) before this phase is considered fully closed.

---

_Verified: 2026-08-11T01:30:00Z_
_Verifier: Claude (gsd-verifier)_
