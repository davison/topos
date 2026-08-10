---
phase: 08-whatsapp-conversations-managed-risk
reviewed: 2026-08-11T00:00:00Z
depth: standard
files_reviewed: 17
files_reviewed_list:
  - docs/api.md
  - kernel/httpapi/agent.go
  - kernel/httpapi/agent_test.go
  - kernel/httpapi/sources.go
  - kernel/httpapi/sources_test.go
  - kernel/httpapi/stream.go
  - kernel/httpapi/stream_test.go
  - kernel/httpapi/webspaces.go
  - kernel/httpapi/whatsapplink.go
  - kernel/httpapi/whatsapplink_test.go
  - kernel/supervisor/supervisor.go
  - kernel/supervisor/supervisor_test.go
  - kernel/supervisor/suspend_test.go
  - web/e2e/specs/g-08-3-degraded-source-not-outage.spec.ts
  - web/src/lib/components/stream-degraded.test.ts
  - web/src/lib/components/StreamError.svelte
  - web/src/lib/components/StreamList.svelte
  - web/src/lib/components/StreamSyncDegraded.svelte
findings:
  critical: 0
  warning: 2
  info: 2
  total: 4
status: issues_found
---

# Phase 08: Code Review Report (gap-closure wave, 08-09/08-10, G-08-3)

**Reviewed:** 2026-08-11
**Depth:** standard
**Files Reviewed:** 17
**Status:** issues_found (no blockers; two warnings, two info items)

## Summary

This wave closes UAT gap G-08-3 on two independent fronts:

1. **Kernel-side per-webspace sync scoping** (`filterRunsByParticipation`,
   applied identically in `StreamHandler`, `WebspacesHandler`,
   `agentStreamHandler`, `agentWebspacesHandler`): a webspace's reported
   `sync` object is now scoped to sources that actually
   `correlate.ParticipatesIn` it, so a failing non-participating source can
   no longer make an unrelated webspace look broken. Traced this against
   `correlate.ParticipatesIn`'s actual implementation (allowlist gate +
   has-match-input rule) and it composes correctly, including the
   documented edge case of an index-only, config-unknown webspace
   resolving to zero participants. Composition with `filterRunsByGrant` on
   the `/agent/v1` mirrors is a correct intersection (grant can only
   narrow, never widen, what participation already narrowed). Covered by
   new, well-targeted unit tests in both `sources_test.go`,
   `stream_test.go` and `agent_test.go`; all pass.

2. **`SuspendInstance`/resume now correctly perform a full generation
   change** (`stopScheduler` -> `Host.Reconcile` -> `commitGeneration`,
   mirroring `Apply`'s own sequence) rather than only killing/relaunching
   the subprocess while leaving the coordinator stale — the root cause of
   G-08-3's WhatsApp-specific failure (a resumed instance's syncs kept
   dying with a "grpc: the client connection is closing" error because the
   coordinator still held a handle to the killed subprocess). The new
   `genCtx`/`genWG` generation-scoping on `Apply`'s eager-resync dispatch
   correctly bounds `stopScheduler`'s wait so a suspend/resume triggered
   from the WhatsApp link-start HTTP path can't block forever on an
   untracked background sync. Verified the locking discipline (`s.mu` held
   across the whole suspend/resume/Apply sequence) is race-free for
   `genWG.Add`/`Wait` ordering, and confirmed via `go build`, `go vet`, and
   `go test ./...` (whole repo) that everything compiles and passes,
   including the new
   `TestApply_EagerResyncDoesNotOutliveItsGeneration`,
   `TestSuspendInstance_ResumedInstanceStillSyncs`, and
   `TestSuspendInstance_SuspendedWindowRecordsNoErroredRun` tests.

3. **Frontend split of the degraded-source presentation**
   (`StreamSyncDegraded.svelte` vs `StreamError.svelte`) is clean: the
   `syncError` prop was fully removed from `StreamError` (verified no
   other call site still passes it), `StreamList.svelte`'s branch ordering
   (`sync-failed` still precedes both empty variants) is preserved and
   pinned by a dedicated structural test
   (`stream-degraded.test.ts`), and the new Playwright spec
   (`g-08-3-degraded-source-not-outage.spec.ts`) exercises all three
   relevant states (zero-item sync failure, genuine fetch failure, sync
   failure alongside items) against the real component tree. Frontend
   tests (`vitest run`) pass.

No BLOCKER-level defects were found. The two WARNING findings below are
both about test-coverage gaps on new failure-recovery branches introduced
by this wave, not incorrect behavior observed in the code itself — the
logic in those branches is consistent with the design documented
alongside it, but neither branch is exercised by any test, and this exact
class of bug (a failure branch that forgets to restart the scheduler,
leaving the kernel with no scheduler running at all) is the literal
regression `07-VERIFICATION.md gaps[0]` / `07-REVIEW.md`'s post-07-09 CR-01
already caught once before on `Apply`'s own analogous branches.

## Warnings

### WR-01: `SuspendInstance`'s new pre-Reconcile failure branch has no test coverage

**File:** `kernel/supervisor/supervisor.go:340-359`

**Issue:** This wave adds `s.stopScheduler()` before `Host.Reconcile` in
`SuspendInstance`, and — new, and load-bearing — a call to
`s.startScheduler(s.cfg)` in the failure branch to put the scheduler back:

```go
s.stopScheduler()

if err := s.host.Reconcile(ctx, withoutName, s.logger); err != nil {
    s.startScheduler(s.cfg)
    return nil, fmt.Errorf("supervisor: suspend instance %q: %w", name, err)
}
```

Before this wave, `SuspendInstance` never stopped the scheduler at all, so
this failure branch's `startScheduler` call is entirely new code, and it
is the only thing standing between a failed suspend and a kernel left with
**no scheduler generation running at all** (every configured source
silently stops syncing until the next `Apply`, restart, or another
`SuspendInstance` call happens to fix it as a side effect). No test in
`kernel/supervisor/suspend_test.go` or `supervisor_test.go` forces
`Host.Reconcile` to fail from inside a `SuspendInstance` call — every
existing `SuspendInstance` test uses `buildMockPluginDir` against a
plugin binary that reconciles successfully. The parallel technique already
used elsewhere in this package to force a deterministic `Reconcile`
failure (an empty/nonexistent `pluginsDir` so a launch attempt fails, see
`supervisor_test.go:260`'s comment) would apply here too.

**Fix:** Add a test that forces `Host.Reconcile` to fail inside
`SuspendInstance` (e.g. suspend an instance, then before calling
`resume`, corrupt/replace the plugin binary or otherwise make a
subsequent internal `Reconcile` fail — or, more simply, extract a seam
similar to the existing `blockingSource` fixture) and assert:

```go
resume, err := sup.SuspendInstance(ctx, "suspend-me")
// ... force the *next* Reconcile call (this one, or the resume's) to fail ...
if err == nil {
    t.Fatal("expected SuspendInstance to fail")
}
// Assert the scheduler is still running: a subsequent Refresh of an
// unrelated, still-configured, still-launched source must still work.
if _, err := sup.Refresh(ctx, "leave-alone"); err != nil {
    t.Fatalf("expected the scheduler to still be running after a failed suspend: %v", err)
}
```

### WR-02: resume closure's Reconcile-failure branch is likewise untested

**File:** `kernel/supervisor/supervisor.go:380-400`

**Issue:** The same new pattern (`stopScheduler` -> `Reconcile` ->
`startScheduler`-on-failure `else` `commitGeneration`-on-success) was added
to the `resume` closure `SuspendInstance` returns. On a failed resume,
`name` is deleted from `s.suspended` (deliberately, per the doc comment,
so a later `Apply` is free to retry launching it) but `s.coord` is never
rebuilt — it still reflects the pre-resume, name-less coordinator — while
the scheduler is restarted against `s.cfg`, which *does* still list `name`
as configured. This is a legitimate, if narrow, "stuck until the next
config save" state, and it's exactly the kind of asymmetric-branch defect
this same file's own `Apply` doc comment (`gaps[0]` / CR-01) warns future
readers about. No test drives a resume-time `Reconcile` failure to confirm
(a) the scheduler really does come back up for every *other* still-healthy
source, and (b) a subsequent `Apply` really does relaunch `name` cleanly
from this state.

**Fix:** Same technique as WR-01, applied to the `resume` closure returned
by `SuspendInstance`: force `Host.Reconcile` to fail on the resume call
specifically (suspend succeeds, then the relaunch fails), and assert both
that `leave-alone`'s refresh still works and that a subsequent `Apply`
(with the plugin binary made reconcilable again) successfully relaunches
`name`.

## Info

### IN-01: `WebspacesHandler` silently swallows a `LatestSyncRunPerSource` failure into the neutral "unknown" sync status

**File:** `kernel/httpapi/webspaces.go:47-51`

**Issue:** This wave's refactor makes the error-swallowing more visible but
does not change it:

```go
runs, err := store.LatestSyncRunPerSource(ctx)
if err != nil {
    runs = nil
}
```

A genuine local-index read failure (e.g. a corrupted/locked SQLite file)
is indistinguishable, from the API consumer's point of view, from "nothing
has ever synced" — every webspace's `last_sync` silently renders as the
zero value instead of the response failing with `500 internal_error`.
This predates this wave (the prior code had the identical `if err == nil`
short-circuit) and is unchanged in observable behavior, so it is not a
regression — but since this wave touched these exact lines, it's a fair
place to note the pattern reads more like an oversight than a documented
decision, especially compared to `agentWebspacesHandler`'s sibling code a
few lines away in `agent.go`, which does propagate this same error as
`500 internal_error`. Consider making the two call sites consistent.

**Fix:** Either propagate the error (`WriteError(w,
http.StatusInternalServerError, "internal_error", err.Error()); return`,
matching `agentWebspacesHandler`'s treatment of the identical call), or
add a one-line comment at the swallow site explaining why `/api/webspaces`
specifically prefers a degraded-but-200 response over a hard failure here
(unlike its own agent-namespace mirror).

### IN-02: `TestApply_EagerResyncDoesNotOutliveItsGeneration` doesn't exercise `Apply`'s actual dispatch loop

**File:** `kernel/supervisor/supervisor_test.go:323-394`

**Issue:** The test's own doc comment is candid about this: it manually
constructs a `*Supervisor`, calls `s.startScheduler(cfg)` directly, and
then hand-copies `Apply`'s eager-resync dispatch snippet (`coord :=
s.coord; genCtx := s.genCtx; genWG := s.genWG; genWG.Add(1); go func() {
... }()`) rather than driving it through a real `s.Apply(ctx)` call,
because `pluginhost.Host` has no seam for an in-memory fake source. This
is a reasonable, well-justified workaround given the stated constraint,
but it means a future edit to `Apply`'s actual dispatch loop (e.g.
accidentally reading a stale `genWG` captured before `commitGeneration`,
or reordering the `genWG.Add`/goroutine-launch pair) would not be caught
by this test — only a hand-maintained duplicate of the logic is verified,
not the real call site. `TestApply_MidFlightSyncLeavesNoStrandedRunningRow`
and other `Apply`-driving tests in this file do exercise the real
function, but none of them currently create the specific
long-running-eager-resync-during-a-later-stopScheduler race this test
was written to guard.

**Fix:** No action required if the constraint is accepted as documented.
If a future phase adds any test seam to `pluginhost.Host` for fake
sources, prefer rewriting this test to drive it through a real
`s.Apply(ctx)` call so a regression in the actual dispatch loop (not a
hand-copied replica of it) is what gets caught.

---

_Reviewed: 2026-08-11_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
