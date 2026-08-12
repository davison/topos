---
status: complete
phase: 08-whatsapp-conversations-managed-risk
source: [08-VERIFICATION.md]
started: 2026-08-11T00:35:00Z
updated: 2026-08-12T00:00:00Z
supersedes: previous 08-UAT.md cycle (3 tests, 2 passed, 1 issue → G-08-3, diagnosed in .planning/debug/whatsapp-grpc-closing-fails-webspace.md, closed in code by plans 08-09/08-10)
---

## Current Test

[testing complete]

## Tests

### 1. Real-device re-test of G-08-3's fix — webspace stream loads after WhatsApp pairing
expected: Pair or re-link a real WhatsApp account via `make dev`, then open its webspace immediately after; the stream loads normally (no "Couldn't load this webspace" / grpc-closing error). An unavailable whatsapp source degrades per-source (StreamSyncDegraded / chip health error), previously captured messages stay browsable, and an unrelated failing source never affects this webspace.
context: The one item automation cannot cover — the hermetic Go tests drive a mock plugin subprocess and the Playwright specs script the HTTP routes, but no test drives a real topos-plugin-whatsapp subprocess through an actual pair/re-link/degrade cycle. Both 08-09-PLAN.md and 08-10-PLAN.md explicitly defer G-08-3's real-device confirmation to this re-run.
result: issue
reported: "after login and opening the webspace: A source couldn't sync / Nothing to show here yet. Your other sources are unaffected — check the source chips above, then retry. / whatsapp: match against source \"whatsapp\": rpc error: code = Unavailable desc = whatsapp: Not linked — pair this device with WhatsApp to start syncing. Use this source's chip menu (\"Re-link…\") or run this plugin binary's -link flag."
severity: major
note: "G-08-3's presentation half is CONFIRMED FIXED — the webspace loaded with the per-source StreamSyncDegraded notice instead of the old full-page fake outage, and the copy correctly says other sources are unaffected. The new issue is upstream: immediately after a successful real-device pairing, the whatsapp plugin still reports Not linked, so the fresh session was not picked up by the running instance."

## Summary

total: 1
passed: 0
issues: 1
pending: 0
skipped: 0
blocked: 0

## Gaps

- gap_id: G-08-4   # next free id in this phase's gap history — G-08-1/G-08-3 are already claimed by executed gap-closure plans' frontmatter, so reusing the test-number formula (G-08-1) would make reconciliation instantly mark this new gap resolved
  truth: "Immediately after a successful real-device WhatsApp pairing, the running whatsapp source instance uses the fresh session: the source syncs (or reports a transient connecting state) rather than staying pinned on the pre-pairing 'Not linked' health error"
  status: resolved
  resolution: "Closed by plans 08-13/08-14/08-15: healthStateConnecting added as the Go zero value (72ed516), serve mode registers a pairLoginWaiter and sets connecting before dialing (a539156/6daf391), Scheduler.firstRefresh bounded retry supersedes a Match landing in the login window (d959706), hermetic subprocess gate (71121cf); real-device re-link human-approved with four repository gates green in 08-15 (db5e0b6). Marked complete 2026-08-12 during v1.0 milestone close sweep."
  reason: "User reported: after login and opening the webspace: A source couldn't sync / Nothing to show here yet. Your other sources are unaffected — check the source chips above, then retry. / whatsapp: match against source \"whatsapp\": rpc error: code = Unavailable desc = whatsapp: Not linked — pair this device with WhatsApp to start syncing. Use this source's chip menu (\"Re-link…\") or run this plugin binary's -link flag."
  severity: major
  test: 1
  root_cause: "AND-gate of two defects, empirically confirmed (a zero-value &SourcePlugin{} reproduces the reported gRPC message byte-for-byte). (1) Plugin: plugins/whatsapp has no 'linked, connecting' health state and startBackgroundClient's SUCCESSFUL path (device.ID != nil, client.Connect() == nil) sets no health state at all — p.state stays at the Go zero value, which health.go defines as healthStateNotLinked; healthStateLinked is assigned in exactly one place, eventhandler.go's *events.Connected case, which arrives only after a further server round trip. (2) Kernel: no plugin-readiness gate exists between subprocess launch and first Match — commitGeneration → startScheduler → runSource refreshes immediately, and main.go completes the go-plugin handshake at the first instant of the not-yet-Connected window, so the kernel's Match deterministically beats the WhatsApp login round trip (Bohrbug, not flake). The errored sync_runs row is then pinned for DefaultSyncInterval=15m and the stream banner renders it (banner reads persisted sync run, not live health). Same window exists at kernel boot and every Apply; 08-09's eager first refresh is the newly-introduced trigger. plugins/mock's Match is unconditionally ready, so no automated gate could catch this class."
  artifacts:
    - path: "plugins/whatsapp/connect.go"
      issue: "successful startBackgroundClient path (~128) returns without any setHealthState — leaves zero-value NotLinked"
    - path: "plugins/whatsapp/health.go"
      issue: "taxonomy has no connecting/logging-in state; NotLinked is iota==0 so it doubles as the uninitialised value"
    - path: "plugins/whatsapp/eventhandler.go"
      issue: "line 28 *events.Connected is the only transition to healthy"
    - path: "kernel/supervisor/supervisor.go"
      issue: "commitGeneration/resume/NewSupervisor start the scheduler with no plugin-readiness gate"
    - path: "kernel/syncer/scheduler.go"
      issue: "runSource (73-74) fires the first refresh immediately, before its ticker"
    - path: "kernel/httpapi/stream.go"
      issue: "banner (95-97) reads the persisted errored sync run, not live health — pinned 15m by DefaultSyncInterval"
    - path: "plugins/mock/plugin.go"
      issue: "Match (201-210) unconditionally ready — no fixture can surface a launch-readiness window"
  missing:
    - "Add a distinct healthStateConnecting ('Linked — connecting to WhatsApp…') and set it explicitly on connect.go's success path so NotLinked stops doubling as the uninitialised zero value"
    - "Close the readiness race: reuse pairwait.go's pairLoginWaiter in startBackgroundClient (bounded wait for *events.Connected, fall through to connecting on timeout) so the go-plugin handshake genuinely means ready"
    - "Decide whether a Match failure in the connecting state should persist an errored sync_runs row at all, given the 15m interval pins the banner"
    - "Give plugins/mock an opt-in 'not ready for N ms after launch' mode so this failure class has a fixture"
  debug_session: .planning/debug/resolved/whatsapp-paired-session-not-picked-up.md
  context: "G-08-3's presentation half is confirmed fixed (per-source StreamSyncDegraded, no fake outage). This gap is the kernel/plugin seam: the freshly paired session IS read correctly from the store — the instance just reports the wrong health state during its connect window and the kernel queries it before it is ready."
