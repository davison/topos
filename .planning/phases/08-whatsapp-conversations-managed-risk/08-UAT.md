---
status: diagnosed
phase: 08-whatsapp-conversations-managed-risk
source: [08-VERIFICATION.md]
started: 2026-08-10T20:45:00Z
updated: 2026-08-10T21:22:01Z
supersedes: previous 08-UAT.md cycle (19 tests, 18 passed, 1 issue → G-08-1, closed by plans 08-05/08-06/08-07 and re-verified after 08-08)
---

## Current Test

[testing complete]

## Tests

### 1. Real-device re-test of the QR pairing flow (G-08-1 fix)
expected: The panel leaves the QR state within a few seconds of the phone reporting success ("Scan accepted — completing login…", then the linked confirmation, then Step 2); it must not sit on a code with a ticking countdown. Restart the kernel and confirm it reconnects with no second QR.
context: Carried forward from the prior verification. Requires a live WhatsApp account and a real kernel run — inherently outside the automated environment.
result: pass
note: "User also observed an error in the webspace after pairing — details pending, may belong to test 2 or a separate issue"

### 2. Real-kernel confirmation of the CR-01 fix (in-flight teardown releases real resources)
expected: "`make dev`, open the SPA, start Add Source → New WhatsApp… with a display name and the seeded local path, click Next, then press Escape immediately — inside the window where the panel still shows its skeleton and no QR has appeared. The kernel log shows the link session cancelled within about a second, not reaped roughly five minutes later. Repeat five times in quick succession, then start one more link attempt and let it run: that final attempt must reach the QR state normally, not be rejected for exceeding the 4-slot concurrent link-session cap."
context: 08-08-PLAN.md Task 1's deferred <human-check> (`verification: backstop` truth). The e2e harness intercepts the link HTTP routes and never spawns a real topos-plugin-whatsapp subprocess, so no automated test can observe a real subprocess exiting or a suspended source instance resuming.
result: pass

### 3. Webspace stream loads after WhatsApp pairing (degradation guarantee)
expected: After linking a WhatsApp source, opening a webspace loads the stream; if the whatsapp plugin connection is unavailable mid-restart, the source degrades (health error surfaced) without failing the whole webspace load.
context: Reported by the user immediately after test 1's successful pairing — not part of the scripted tests, captured as its own issue.
result: issue
reported: "Couldn't load this webspace / The topos service didn't respond — check that it's running, then retry. Kernel error: whatsapp: match against source \"whatsapp\": rpc error: code = Canceled desc = grpc: the client connection is closing"
severity: major

## Summary

total: 3
passed: 2
issues: 1
pending: 0
skipped: 0
blocked: 0

## Gaps

- gap_id: G-08-3
  truth: "Opening a webspace after linking a WhatsApp source loads the stream; a whatsapp plugin whose gRPC connection is closing (e.g. mid-restart after pairing) degrades as an unavailable source rather than failing the entire webspace load"
  status: failed
  reason: "User reported: Couldn't load this webspace / The topos service didn't respond — check that it's running, then retry. Kernel error: whatsapp: match against source \"whatsapp\": rpc error: code = Canceled desc = grpc: the client connection is closing"
  severity: major
  test: 3
  root_cause: "AND-gate of two defects: (1) supervisor.SuspendInstance (used by the WhatsApp link session on the named-instance/Re-link path) kills the instance's go-plugin client via Host.Reconcile → Plugin.Kill() (local gRPC ClientConn.Close) but never rebuilds the syncer.Coordinator or restarts the scheduler generation, and its resume closure relaunches the instance in the host only — the Coordinator captured its *pluginhost.Plugin handles at construction and has no update seam, so every subsequent Match (scheduled tick, POST refresh, eager resync) goes through the killed client and yields grpc ErrClientConnClosing until the next config save or restart; a re-link never saves config (RelinkModal.onrelinked only refetches health), so the error is pinned indefinitely. (2) aggregateSyncStatus folds ANY configured source's latest errored run into the single global sync object of every webspace's 200 stream response, and the SPA's streamVariant maps sync.status==='error' && items.length===0 to a full-page StreamError whose Phase-1 copy falsely claims 'The topos service didn't respond' — the kernel actually answered 200; per-source degradation is structurally violated for any zero-item webspace when any source's latest sync failed."
  artifacts:
    - path: "kernel/supervisor/supervisor.go"
      issue: "SuspendInstance + resume closure never rebuild coordinator/restart scheduler (~245-295); eager-resync goroutines untracked (~595-611)"
    - path: "kernel/syncer/coordinator.go"
      issue: "sources captured once at construction; no live by-name resolution seam"
    - path: "kernel/httpapi/whatsapplink.go"
      issue: "poll-handler resume runs under r.Context() (reaper/Shutdown correctly use Background)"
    - path: "kernel/httpapi/sources.go"
      issue: "aggregateSyncStatus: global any-source error aggregation embedded per-webspace"
    - path: "kernel/httpapi/stream.go"
      issue: "stream response carries global sync status not scoped to the webspace's participating sources"
    - path: "web/src/lib/format.ts"
      issue: "streamVariant escalates sync.status==='error' && items.length===0 to full-page StreamError"
    - path: "web/src/lib/components/StreamError.svelte"
      issue: "fixed 'The topos service didn't respond' copy shown for per-source sync errors, not just genuine fetch failures"
  missing:
    - "Make suspend/resume a generation change: stop scheduler + commitGeneration (minus/plus the instance) in both suspend and resume, or give the Coordinator live by-name source resolution through the host"
    - "Run poll-handler resume on a detached context, not r.Context()"
    - "Track/scope Apply's eager-resync goroutines so stopScheduler cannot block on them"
    - "Regression test: drive a Coordinator sync through a suspended-then-resumed instance (the suspend_test.go blind spot)"
    - "Scope the stream response's sync status to the webspace's participating sources and/or degrade per-source sync failure to chip-level health error + empty stream; reserve the 'didn't respond' copy for genuine fetch failures"
  debug_session: .planning/debug/whatsapp-grpc-closing-fails-webspace.md
