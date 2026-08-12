---
phase: 08-whatsapp-conversations-managed-risk
reviewed: 2026-08-11T12:15:26Z
depth: standard
files_reviewed: 14
files_reviewed_list:
  - docs/testing.md
  - kernel/index/store.go
  - kernel/index/store_test.go
  - kernel/pluginhost/host.go
  - kernel/supervisor/launchlatency_test.go
  - kernel/supervisor/readiness_test.go
  - kernel/supervisor/supervisor.go
  - kernel/supervisor/supervisor_test.go
  - kernel/syncer/scheduler.go
  - plugins/mock/main.go
  - plugins/mock/plugin_test.go
  - plugins/mock/readiness.go
  - plugins/whatsapp/connect.go
  - plugins/whatsapp/connect_test.go
findings:
  critical: 0
  warning: 2
  info: 2
  total: 4
status: issues_found
---

# Phase 08: Code Review Report (gap-closure wave 7)

**Reviewed:** 2026-08-11T12:15:26Z
**Depth:** standard
**Files Reviewed:** 14
**Status:** issues_found (no blockers; two warnings, two informational items)

## Summary

This wave (08-13/08-14, plus the 998a9ab debug-session fix) makes three
changes: (1) `kernel/supervisor.Supervisor` splits its single mutation
mutex (`s.mu`) so that `Host()`/`Coordinator()` reads go through a
separate `genMu.RWMutex` instead, so a slow plugin relaunch under
`SuspendInstance`'s resume closure no longer freezes every other source's
health/refresh routes; (2) `kernel/pluginhost.Host` gains its own internal
`sync.RWMutex` around the `plugins` slice, and `Reconcile` now performs its
launch loop with no lock held, taking the write lock only for the
kill-and-commit step; (3) `plugins/whatsapp/connect.go` moves the
serve-mode login wait off the launch path onto a background goroutine, so
a kernel boot/relaunch with an already-linked WhatsApp source no longer
pays up to 15s before `goplugin.Serve` is reached; and (4) `index.Store`
gains a test-only `SyncRunsForSourceForTesting` reader used to fix a
flaky assertion in `TestApply_MidFlightSyncLeavesNoStrandedRunningRow`
that had been reading the wrong row via `LatestSyncRunPerSource`.

I traced the full lock-ordering story across `supervisor.go`/`host.go`
(genMu vs. mu, mu vs. Host's own internal mu) and found it internally
consistent: every writer of `s.host`/`s.coord` holds `s.mu` for the
mutation's whole duration and takes `genMu.Lock()` only for the bare
field assignment, so readers taking `genMu.RLock()` never wait behind a
`Host.Reconcile` call, and the plain (unlocked) reads of `s.coord` inside
`startScheduler`/`stopScheduler`/`commitGeneration` are always either
same-goroutine reads of a value that goroutine just wrote, or reads that
can never be concurrent with a write because `s.mu` already excludes any
other mutator — so they cannot race with `Coordinator()`'s `genMu.RLock()`
reads (read/read is not a race). `pluginhost.Host`'s new internal lock
correctly narrows to protect only the `plugins` field, releasing the lock
across the slow subprocess-launch loop and re-acquiring it only for the
kill+commit region, which is the specific defect G-08-5 targeted. The
WhatsApp `connect.go` goroutine dispatch is structurally sound: the
success and failure branches are mutually exclusive, so
`RemoveEventHandler` is called exactly once regardless of outcome, and
`Match`'s own readiness gate (`healthState.Healthy()`) is driven
independently by the already-registered `p.handleEvent`'s synchronous
`*events.Connected` case, not by the (now purely diagnostic) login-wait
goroutine — so the dispatched goroutine's own 15s timeout has no bearing
on correctness, only on how long a stuck login stays silently
unexplained in the log.

No blocker-level defects found. Two warnings below describe real,
if narrow, risk surfaces this wave leaves in place; two info items note
minor quality/robustness gaps.

## Warnings

### WR-01: WhatsApp's first post-relaunch Match now races a fixed 7s retry budget against a live login round trip that has never been proven against a real degraded network

**File:** `plugins/whatsapp/connect.go:143-202`, `kernel/syncer/scheduler.go:48-66`

**Issue:** Before this wave, `startBackgroundClient` synchronously awaited
`loginWaiter.wait(serveLoginTimeout)` (up to 15s) before returning, which
made the go-plugin handshake itself a soft proxy for "login is likely
done" by the time the kernel's first scheduled `Match` could land. This
wave removes that guarantee outright — the comment at connect.go:138-148
states it plainly: "the go-plugin handshake no longer implies login has
completed." The safety net is now `kernel/syncer/scheduler.go`'s
`defaultFirstRefreshRetryDelays` (`{2s, 5s}`, ~7s of total retry cover),
which relies on `Match` returning `codes.Unavailable` while
`healthState` is still `healthStateConnecting` (not `Healthy()`) and
being retried before the source is pinned on an errored `sync_runs` row.
The scheduler.go comment itself concedes the number is a judgment call
("a WhatsApp login round trip is measured in hundreds of milliseconds"),
and the connect_test.go/launchlatency_test.go suite proves only the
*ordering* (goroutine dispatch, handler removal) via AST structural
guards and a synthetic mock-plugin delay — never a real WhatsApp login
against a slow/lossy network. If a real login takes longer than ~7s
(congested Wi-Fi, VPN, a first-link cold start on a large account), the
source is left on an errored sync run for the rest of the default sync
interval (15 minutes) rather than "within seconds" as the comment at
scheduler.go:56 promises. This exact gap is already tracked in this
phase's planning docs (a "real-device gate" plan is called out as
pending in the wave that produced this file), so it is not undiscovered,
but it is worth restating here as a live, shipped risk rather than a
closed one: the code review found no in-repo evidence this retry budget
has been validated against a real degraded connection.

**Fix:** Either (a) keep this as an explicitly accepted, tracked gap and
make sure the pending real-device UAT gate actually exercises a
throttled/high-latency network before this ships to end users, or (b)
widen `defaultFirstRefreshRetryDelays` with one more attempt further out
(e.g. `{2s, 5s, 10s}`) to buy more margin without reintroducing a
launch-path block, since the retries are already bounded and
context-cancellable.

### WR-02: `pluginhost.Host.Reconcile` has no internal protection against concurrent invocation — correctness depends entirely on a caller convention documented in a different package

**File:** `kernel/pluginhost/host.go:195-265`

**Issue:** The new `h.mu sync.RWMutex` (host.go:129-139) protects
`h.plugins` against a *reader* racing a single in-flight `Reconcile`, but
it does nothing to serialize two *concurrent* `Reconcile` calls against
each other: the `existing`/`kept`/`launched` maps are built from a
snapshot taken under a released `RLock()`, and the final commit
(`h.plugins = next`) is a plain last-writer-wins assignment under
`Lock()`. If two `Reconcile` calls ever ran concurrently, the second
call's `next` slice is built from its own stale `existing`/`kept` view
and would silently clobber whatever the first call had just committed —
including plugin instances the first call launched and the second knew
nothing about — with no compile-time or runtime signal that this
happened (not even a `-race` failure, since every access is technically
lock-protected). The code's own comment (host.go:135-138) says this is
safe *only* because "kernel/supervisor's own mutation lock
(Supervisor.mu) ... every caller of Reconcile in this repo already holds
for the call's whole duration" — i.e., `pluginhost` is a generically
importable package whose most safety-critical exported method's
correctness is guaranteed entirely by an invariant enforced in a sibling
package, with nothing in `pluginhost` itself preventing a future caller
(a second command, a test harness, a future HTTP-triggered reconcile
path) from violating it. This is a pre-existing limitation carried
forward rather than something this wave newly introduced (Host had *no*
internal locking at all before this wave), but the wave's own framing —
"the write lock only for the kill-and-commit region" — reads as if
concurrent-Reconcile safety were now handled, when only reader/single-writer
safety is.

**Fix:** At minimum, tighten the doc comment on `Reconcile` to state
explicitly that concurrent calls are undefined behavior (silent state
loss, not a panic or a race-detector-visible fault) rather than merely
implying it is "excluded" by an external convention. Consider a cheap
internal guard for defense in depth — e.g. a second, dedicated
`reconciling atomic.Bool` (or a plain non-reentrant `sync.Mutex` around
the whole `Reconcile` body, orthogonal to `h.mu`) that returns a named
error instead of silently losing a launch if `Reconcile` is ever called
re-entrantly.

## Info

### IN-01: The new WR-01 background goroutine adds a third concurrent writer to `p.logOut` (`os.Stderr`) with no synchronization

**File:** `plugins/whatsapp/connect.go:197-202`, `plugins/whatsapp/eventhandler.go` (multiple `fmt.Fprintf(p.logOut, ...)` call sites), `plugins/whatsapp/groupsync.go`

**Issue:** `p.logOut` is a plain `io.Writer` (`os.Stderr` in production)
written to via bare `fmt.Fprintf` calls from several independent
goroutines: the whatsmeow event-dispatch goroutine (`eventhandler.go`,
`groupsync.go`), gRPC handler goroutines (`plugin.go`'s `Match`/`Health`
paths), and now this wave's own dispatched login-wait goroutine
(connect.go:197-202). None of these writers coordinate with each other.
This predates this wave (the event-handler and gRPC paths already wrote
concurrently), but the wave adds one more concurrent writer to the same
unsynchronized sink, increasing the odds of two log lines interleaving
into one garbled line under load (a real WhatsApp account with active
groups can fire many events per second). This is a log-hygiene issue,
not a correctness one — no data is lost or corrupted beyond the log
stream's own readability.

**Fix:** Wrap `p.logOut` in a small `sync.Mutex`-guarded writer (or route
every log line through a single `setHealthState`-style helper that holds
`p.mu`) so concurrent `Fprintf` calls can never interleave mid-line.
Low priority given the log stream's own diagnostic (not functional) role.

### IN-02: `index.Store.SyncRunsForSourceForTesting` ships as an unbounded, unenforced-test-only production symbol

**File:** `kernel/index/store.go:800-853`

**Issue:** The method is documented at length as "TEST-ONLY" and named
after the existing `config.NewStoreForTesting` convention, but — like
that predecessor — it lives in a non-`_test.go` file and ships in the
production binary with no compiler, lint, or build-tag enforcement
stopping a real request-path caller from using it. It also runs
`SELECT ... WHERE source = ? ORDER BY id` with no `LIMIT`, which is fine
for today's test fixtures (a handful of rows) but would become an
unbounded read if a caller with a genuinely long-lived instance ever
used it. Since this exactly mirrors an already-established house
convention (`config.NewStoreForTesting`) rather than introducing a new
pattern, this is not a regression — just worth a reviewer's eyes given
the review's specific instruction to check "whether the test-only index
reader is properly fenced": it is fenced by naming and doc comment only,
not by tooling.

**Fix:** No action required to match house convention; if this method's
usage is ever considered for a real (non-test) call site, add a `LIMIT`
and drop the `ForTesting` suffix at that point, per the method's own doc
comment.

---

_Reviewed: 2026-08-11T12:15:26Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
