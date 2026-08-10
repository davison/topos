---
phase: 08-whatsapp-conversations-managed-risk
fixed_at: 2026-08-10T16:40:00Z
review_path: .planning/phases/08-whatsapp-conversations-managed-risk/08-REVIEW.md
iteration: 1
findings_in_scope: 3
fixed: 3
skipped: 0
status: all_fixed
---

# Phase 08: Code Review Fix Report

**Fixed at:** 2026-08-10T16:40:00Z
**Source review:** .planning/phases/08-whatsapp-conversations-managed-risk/08-REVIEW.md
**Iteration:** 1

**Summary:**
- Findings in scope (critical + warning): 3
- Fixed: 3
- Skipped: 0

## Fixed Issues

### CR-01: `describePlugin`'s trial-launch always collides with a running WhatsApp instance's exclusive store lock

**Files modified:** `plugins/whatsapp/main.go`, `plugins/whatsapp/describeonly.go` (new), `kernel/pluginhost/host.go`, `kernel/pluginhost/describe_whatsapp_test.go` (new)
**Commit:** `7cb51d1`
**Applied fix:** Chose fix direction (b) from the review's own two options — a describe-only launch mode — over (a) (reusing a running instance's cached Describe vocabulary from `kernel/httpapi/DescribePluginHandler`), because (a) would have required crossing an existing, explicitly documented architectural boundary: `kernel/httpapi/config.go`'s own `Applier` doc comment states the config write path must never grow a dependency on live plugin process lifecycle beyond the stateless trial-launch it already performs — `DescribePluginHandler` is deliberately given only `pluginsDir`, never a `*pluginhost.Host` reference to currently-launched instances.

Added `WEBSPACES_DESCRIBE_ONLY=1`, an environment variable `kernel/pluginhost.launch` now sets only for `DescribePluginType`'s trial-launch call (a new `describeOnly bool` parameter, `false` for `Discover`/`Reconcile`'s real boot-time/hot-apply launches). `plugins/whatsapp/main.go` checks this variable before requiring `WEBSPACES_SOURCE_CONFIG` at all: when set, it serves a new minimal `describeOnlyPlugin` (`plugins/whatsapp/describeonly.go`) that answers `Describe` from the same fixed package-level constants `SourcePlugin.Describe` already returns, and never calls `NewSourcePlugin`/`acquireStoreLock` — so the trial-launched subprocess never contends with a real running instance's exclusive `storelock.go` flock for the same data directory. `Match`/`Fetch`/`Health` on `describeOnlyPlugin` explicitly return `codes.Unimplemented` rather than silently no-op, since they are never called in this mode (pinned by the existing `TestDescribePluginTypeGuard_ReachesNoRPCBeyondDescribe` AST guard) — this makes a future defect that somehow routed one of them to a describe-only instance fail loudly instead of silently or via a nil-pointer panic.

Other plugin types (Signal, Proton, paperless, SilverBullet) never read `WEBSPACES_DESCRIBE_ONLY`, so this is additive — their trial-launch behavior is byte-identical to before.

Added `kernel/pluginhost/describe_whatsapp_test.go`'s `TestDescribePluginType_WhatsApp_SucceedsWhileARealInstanceHoldsTheStoreLock`: builds the real `topos-plugin-whatsapp` binary, launches a genuine running instance (holding the store lock for the test's duration), then calls `DescribePluginType` against the same binary and data directory while that instance is alive — asserting it succeeds with the correct `source_type`/`plugin_display_name`/`match_vocabulary`, and that the running instance is left completely untouched. Verified to fail against the pre-fix code with the exact `ErrStoreInUse`-wrapped handshake failure the review describes.

### WR-01: WhatsApp link-session concurrency cap is enforced after the subprocess is already spawned

**Files modified:** `kernel/httpapi/whatsapplink.go`, `kernel/httpapi/whatsapplink_test.go`
**Commit:** `9a55f91`
**Applied fix:** Added `linkSessionStore.reserve()`/`release()`, claiming/returning a slot under the store's mutex. `WhatsAppLinkStartHandler` now calls `store.reserve()` immediately after the empty-path check — before `SuspendInstance` and before the subprocess `spawner` call — and calls `store.release()` on every failure path that follows (suspend failure, spawn failure, registration failure). `linkSessionStore.register` no longer re-checks capacity itself; it converts an already-claimed reservation into the live session entry. `TestWhatsAppLinkReaper`'s existing direct `store.register(sess)` call (bypassing `reserve` entirely) continues to work unchanged, since `register` only decrements `reserved` when it is non-zero.

Added `TestWhatsAppLinkStart_CapEnforcedBeforeSpawn`: fills every session slot directly via `register`, then issues one more start request and asserts zero spawner invocations alongside the `429 link_failed` response. Verified to fail against the pre-fix handler (spawner invoked once before the after-the-fact cap check rejected it).

### WR-02: A concurrent, unrelated config save can fail while a WhatsApp re-link session is suspending an instance

**Files modified:** `kernel/supervisor/supervisor.go`, `kernel/pluginhost/matchconfig.go`, `kernel/supervisor/suspend_test.go`
**Commit:** `df7a548`
**Applied fix:** Chose fix direction (a) from the review's own two options — a suspended-instance-aware `Apply` — over (b) (holding `s.mu` for a suspension's entire duration). Investigation showed (b), as literally described, would be substantially worse than the defect it fixes: `Supervisor.mu` also guards `Host()`/`Coordinator()`, which back `Fetch`/`ProbeSources`/`Refresh`/`RefreshAll` — every read-path HTTP handler. Holding it for a suspension's full duration (up to `linkSessionDeadline`, 5 minutes) would freeze the entire kernel's read surface, not just `Apply`, for the whole span of any WhatsApp link/re-link flow.

Added a `suspended map[string]*pluginhost.Plugin` field on `Supervisor`, populated by `SuspendInstance` with the just-killed instance's own `*pluginhost.Plugin` value (its `SourceType`/`PluginDisplayName`/`MatchVocabulary` are plain cached struct fields, safe to read after the subprocess is dead — no live RPC involved) and cleared by the resume closure it returns. `Apply` now excludes every currently-suspended name from what `Host.Reconcile` is asked to launch (so it never tries to relaunch an instance a live link subprocess is contending for), and validates the swapped config via a new `pluginhost.ValidateMatchConfigWithSuspended(cfg, host, suspendedPlugins)` — a sibling of the existing `ValidateMatchConfig` that merges in each suspended instance's cached vocabulary — instead of `ValidateMatchConfig`, which would otherwise reject any webspace the suspended instance participates in as "has no launched plugin" even though nothing about its configuration changed. The end-of-`Apply` eager-resync dispatch loop also skips suspended names. `SuspendInstance`'s existing resume closure already re-reads `s.cfg.Sources` fresh at resume time, so any interleaved save's changes are picked up correctly once the suspending session ends.

Added `TestApply_UnrelatedSaveSucceedsWhileAnInstanceIsSuspended`: suspends one instance, applies a save that is completely unrelated to it (an explicit match block still names the suspended instance; only a second, unrelated instance's connection config changes), and asserts `Apply` succeeds, the suspended instance is left un-relaunched, and it comes back correctly on resume. Verified to fail against the pre-fix `Supervisor` (the suspended instance was incorrectly relaunched mid-Apply, defeating the suspension).

## Skipped Issues

None — all in-scope findings (critical + warning) were fixed. IN-01 (`storeLock.Release` dropping the close error on a double-fault) was out of `fix_scope: critical_warning` and left untouched.

## Verification

Run inside the isolated review-fix worktree (`.claude/worktrees/rf-08-167105-1786375178`, branch `gsd-reviewfix/08-167105`, fast-forwarded into `main` by the cleanup tail):

- `make test-portable` (all `CGO_ENABLED=0` Go modules, including the new/changed `kernel/pluginhost`, `kernel/httpapi`, `kernel/supervisor`, and `plugins/whatsapp` packages): **pass**
- `make test-signal` (the cgo/libsqlcipher Signal plugin module, unaffected by this fix but part of the full `make test` gate): **pass**
- `npm --prefix web test`: **642/642 tests pass** — no `web/` files were touched by any of these three fixes, so no behavior change was expected or observed
- `make build` (full production build: web build, kernel binary, all six plugin binaries including the cgo Signal plugin): **pass**
- `make e2e` was not run — no `web/` files were modified by this fix pass, per the stated constraint

Each fix's own regression test was additionally verified to **fail** against the pre-fix code (via a temporary revert-test-restore cycle) before being committed, confirming each test actually exercises the defect it claims to close, not just the new code path.

---

_Fixed: 2026-08-10T16:40:00Z_
_Fixer: Claude (gsd-code-fixer)_
_Iteration: 1_
