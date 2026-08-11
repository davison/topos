---
phase: 08-whatsapp-conversations-managed-risk
reviewed: 2026-08-11T00:36:50Z
depth: standard
files_reviewed: 12
files_reviewed_list:
  - docs/testing.md
  - kernel/supervisor/readiness_test.go
  - kernel/syncer/scheduler.go
  - kernel/syncer/scheduler_test.go
  - plugins/mock/main.go
  - plugins/mock/plugin.go
  - plugins/mock/plugin_test.go
  - plugins/mock/readiness.go
  - plugins/whatsapp/connect.go
  - plugins/whatsapp/connect_test.go
  - plugins/whatsapp/health.go
  - plugins/whatsapp/health_test.go
findings:
  critical: 0
  warning: 2
  info: 1
  total: 3
status: issues_found
---

# Phase 08: Code Review Report

**Reviewed:** 2026-08-11T00:36:50Z
**Depth:** standard
**Files Reviewed:** 12
**Status:** issues_found

## Summary

This is the G-08-4 gap-closure wave: 08-11 makes `healthStateConnecting`
the WhatsApp plugin's zero value and adds a bounded serve-mode login wait
before the go-plugin handshake completes; 08-12 adds a bounded
first-refresh retry to the kernel scheduler and an opt-in
launch-readiness fixture to the mock plugin, plus a hermetic supervisor
test proving the two work together.

I read all twelve files, diffed each against the stated base commit to
separate genuinely new code from pre-existing code, cross-checked the
new code against callers/callees outside the diff (`plugin.go`,
`eventhandler.go`, `pairwait.go`, `main.go`, `kernel/pluginhost/host.go`,
`cmd/topos/main.go`), inspected the vendored `go-plugin` and `whatsmeow`
source to verify specific claims made in code comments (go-plugin's
default `StartTimeout`, whatsmeow's `AddEventHandler`/`RemoveEventHandler`
pair), and ran the full test suite for the affected packages with the
race detector (`go test -race`), including the real-subprocess hermetic
supervisor test. Everything passes, no data race was detected, and the
zero-value/retry logic is well covered by both unit and structural
(AST-based) tests. I did not find a defect that rises to Critical
(security/data-loss/crash) under this review's severity rubric.

I did find one real, provable regression to kernel/plugin boot latency
that the diff's own reasoning does not fully account for (WR-01), plus a
smaller resource-cleanup gap specific to the new long-lived serve-mode
code path (WR-02) that a look at the sibling short-lived `-link` CLI flow
makes clear is a genuine omission, not an intentional convention. One
maintainability nit (IN-01) rounds out the findings.

## Warnings

### WR-01: The new serve-mode login wait can block the ENTIRE kernel's HTTP startup, not just the WhatsApp source

**File:** `plugins/whatsapp/connect.go:168-170` (the `loginWaiter.wait(serveLoginTimeout)` call added by this diff)

**Issue:** `startBackgroundClient`'s already-paired success path now blocks
for up to `serveLoginTimeout` (15s) waiting for a real `*events.Connected`
before returning. This wait runs *before* `goplugin.Serve()` is ever
called in `plugins/whatsapp/main.go`, which means the go-plugin handshake
line the parent process is blocking on doesn't get written to stdout
until the wait resolves (success, a definitive failure event, or the 15s
timeout) — exactly what the code comment itself acknowledges: "every
second spent here is a second the kernel's `pluginhost.launch` is blocked
on the handshake completing."

What the comment does not account for is *what else* is blocked on that
same call. `kernel/pluginhost/host.go`'s `Discover` (used at boot) and
`Reconcile` (used by hot-apply) launch every configured source
**sequentially**, in a single `for name, src := range sources { launch(...) }`
loop (host.go:137-144, 191-200) — not one goroutine per source. And
`cmd/topos/main.go` does not call `http.ListenAndServe` until
`supervisor.NewSupervisor` — which calls `pluginhost.Discover` synchronously
— returns. So on **every normal kernel restart** with an already-linked
WhatsApp source configured (the steady-state case, not an edge case), the
entire kernel's HTTP server — and therefore every *other*, unrelated
source's own reachability, and the web UI itself — cannot become
available until WhatsApp's login either completes or times out. Because
Go map iteration order is randomized, whether WhatsApp's wait lands first
or last in the launch sequence (and therefore how much of the up-to-15s
delay other sources inherit) is non-deterministic run to run.

Before this diff, `startBackgroundClient`'s already-paired success path
returned immediately after dispatching `client.Connect()` (a non-blocking
dial), so this was not a bottleneck. Checked every sibling plugin's
`main.go` (`paperless`, `silverbullet`, `proton`, `signal`) — none of them
perform any blocking network/login round trip before `goplugin.Serve()`;
this diff introduces the only such synchronous, multi-second,
network-dependent wait in the whole launch path.

This is a real, provable regression to boot/hot-apply latency, not a
"performance" tuning concern (Big-O, N+1 queries) that this review's scope
excludes — it's a missing concurrency boundary that lets one source's slow
login gate the whole kernel's and every sibling source's availability, on
a path (`cmd/topos/main.go`'s boot sequence) that previously had no such
dependency.

**Fix:** Pick one:
- Launch each configured source concurrently in `Discover`/`Reconcile`
  (e.g. an `errgroup.Group` per source, preserving the existing
  "every launch this call needed has now succeeded — only now kill/commit"
  ordering guarantee) so one slow source can never delay a sibling's
  launch or the HTTP listener; or
- Move the login wait off the synchronous startup path entirely — start
  `startBackgroundClient`'s dial, return immediately once
  `healthStateConnecting` is recorded, and let 08-12's own
  first-refresh retry (already built for exactly this "handshake
  completed before Match is truly ready" gap) absorb the residual window
  instead of blocking the handshake on it; or, at minimum,
- Shorten `serveLoginTimeout` substantially and lean more on the
  kernel-side retry, since 08-12 already covers the failure mode this
  wait exists for.

### WR-02: `loginWaiter`'s event handler is never removed from the long-lived serve-mode client

**File:** `plugins/whatsapp/connect.go:130-131, 168-170`

**Issue:** `client.AddEventHandler(loginWaiter.handleEvent)` returns a
`uint32` handler ID (`whatsmeow.Client.AddEventHandler`, which pairs with
`Client.RemoveEventHandler(id)` — confirmed in the vendored whatsmeow
source), but the return value is discarded here, and no
`RemoveEventHandler` call follows `loginWaiter.wait(...)` once it
returns. Compare with `link.go`'s identical registration
(`client.AddEventHandler(loginWaiter.handleEvent)` at link.go:109): there
it's harmless because the `-link` CLI flow's client is short-lived and
disconnected/exited shortly after. `connect.go`'s serve-mode client is
different — it's the plugin's persistent connection for the entire
subprocess lifetime (per `plugin.go`'s own doc comment), which is
typically days or weeks of uptime. Once `wait()` returns, this handler
becomes permanent dead weight: it is invoked (and does nothing, since
`pairLoginWaiter.signal`'s `sync.Once` already fired) for every single
whatsmeow event dispatched for the rest of the process's life, and the
`loginWaiter` (its buffered channel and `sync.Once`) is never eligible
for garbage collection.

Not a correctness bug — `signal`'s `sync.Once` plus a buffered channel of
size 1 makes the extra calls safe (no panic, no leak of unbounded memory)
— but it's a straightforward, low-risk cleanup the whatsmeow API was
clearly designed to support, and copying the short-lived `-link` flow's
registration pattern into the long-lived serve-mode path without also
copying (or adding) a matching teardown is a real omission introduced by
this diff.

**Fix:**
```go
handlerID := client.AddEventHandler(loginWaiter.handleEvent)
// ...
if err := loginWaiter.wait(serveLoginTimeout); err != nil {
    fmt.Fprintf(p.logOut, "%s: serve-mode startup: %v\n", pluginName, err)
}
client.RemoveEventHandler(handlerID)
```

## Info

### IN-01: The hermetic readiness test's timing comments are coupled to a production constant with no assertion enforcing the link

**File:** `kernel/supervisor/readiness_test.go:56-59`

**Issue:** The test's comment states "the default retry schedule's first
delay is 2s and the readiness window is only 0.7s, so the retry should
supersede the launch-window error well inside 15s" — this is accurate
today (`kernel/syncer/scheduler.go`'s `defaultFirstRefreshRetryDelays =
[]time.Duration{2 * time.Second, 5 * time.Second}`), and the test itself
(generous 15s deadline) won't break if that constant is tuned later. But
the comment's *specific* numeric claim ("first delay is 2s") has no
assertion tying it to the actual constant — a future change to
`defaultFirstRefreshRetryDelays` would silently leave this comment
describing stale behaviour while the test keeps passing. Low-value to fix
on its own, but worth a note since it's the kind of drift that makes a
"why does this test use a 15s deadline" question harder to answer
correctly later.

**Fix:** Either reference `defaultFirstRefreshRetryDelays` directly in
the comment via a `//go:generate`-free code comment pointer (already
partially done — the comment names the file), or accept this as
acceptable drift risk given the deadline itself is generous enough not to
require tightening.

---

_Reviewed: 2026-08-11T00:36:50Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
