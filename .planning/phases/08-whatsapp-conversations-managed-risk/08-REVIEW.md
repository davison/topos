---
phase: 08-whatsapp-conversations-managed-risk
reviewed: 2026-08-10T14:40:46Z
depth: standard
files_reviewed: 44
files_reviewed_list:
  - .gitignore
  - Makefile
  - cmd/topos/main.go
  - config.example.toml
  - docs/api.md
  - docs/testing.md
  - go.work
  - internal/audit/outbound_hosts_test.go
  - kernel/httpapi/agent_live_config_test.go
  - kernel/httpapi/agent_test.go
  - kernel/httpapi/config_test.go
  - kernel/httpapi/contract_test.go
  - kernel/httpapi/live_config_test.go
  - kernel/httpapi/routes.go
  - kernel/httpapi/whatsapplink.go
  - kernel/httpapi/whatsapplink_test.go
  - kernel/supervisor/supervisor.go
  - kernel/supervisor/suspend_test.go
  - plugins/whatsapp/connect.go
  - plugins/whatsapp/connect_test.go
  - plugins/whatsapp/deeplink.go
  - plugins/whatsapp/deeplink_test.go
  - plugins/whatsapp/delink_test.go
  - plugins/whatsapp/digest.go
  - plugins/whatsapp/digest_test.go
  - plugins/whatsapp/eventhandler.go
  - plugins/whatsapp/go.mod
  - plugins/whatsapp/groupsync.go
  - plugins/whatsapp/groupsync_test.go
  - plugins/whatsapp/health.go
  - plugins/whatsapp/health_test.go
  - plugins/whatsapp/link.go
  - plugins/whatsapp/link_test.go
  - plugins/whatsapp/main.go
  - plugins/whatsapp/match.go
  - plugins/whatsapp/match_test.go
  - plugins/whatsapp/messagestore.go
  - plugins/whatsapp/messagestore_test.go
  - plugins/whatsapp/outbound_hosts_test.go
  - plugins/whatsapp/pairwait.go
  - plugins/whatsapp/pairwait_test.go
  - plugins/whatsapp/plugin.go
  - plugins/whatsapp/pushnames.go
  - plugins/whatsapp/pushnames_test.go
  - plugins/whatsapp/readonly_test.go
  - plugins/whatsapp/render.go
  - plugins/whatsapp/storelock.go
  - plugins/whatsapp/storelock_test.go
  - web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts
  - web/src/lib/api.ts
  - web/src/lib/components/AddSourceModal.svelte
  - web/src/lib/components/QRPanel.svelte
  - web/src/lib/components/RelinkModal.svelte
  - web/src/lib/components/SourceChip.svelte
  - web/src/lib/components/WebspaceHeader.svelte
  - web/src/lib/components/add-source.test.ts
  - web/src/lib/components/chip-edit-menu.test.ts
  - web/src/lib/components/qr-panel.test.ts
  - web/src/lib/components/relink.test.ts
  - web/src/lib/plugin-fields.test.ts
  - web/src/lib/plugin-fields.ts
  - web/src/routes/w/[webspace]/+page.svelte
findings:
  critical: 1
  warning: 2
  info: 1
  total: 4
status: issues_found
---

# Phase 08: Code Review Report

**Reviewed:** 2026-08-10T14:40:46Z
**Depth:** standard
**Files Reviewed:** 44 (source and test files under the phase's declared scope; `plugins/whatsapp/link_test.go` was listed in `<required_reading>` but the file present on disk is `pairwait_test.go` — both were read)
**Status:** issues_found

## Summary

This phase adds a WhatsApp source plugin (whatsmeow-backed), a kernel
`whatsapp-link` HTTP session surface for in-app QR pairing, and the
frontend flows that drive it. The read-only/no-send guarantees are well
enforced by AST-scanned tests (`readonly_test.go`, `outbound_hosts_test.go`),
the health-state taxonomy is carefully designed to never imply data loss,
and the store-lock/session-lifecycle code in
`kernel/httpapi/whatsapplink.go` is generally careful about ordering,
cleanup, and idempotency.

However, tracing the `describePlugin` call path used by two existing,
pre-Phase-8 UI flows (add an already-configured instance to a second
webspace; "Edit match settings…" on a chip) against WhatsApp's
plugin-architecture-specific exclusive store lock (`storelock.go`)
surfaces a genuine functional break: those flows always trial-launch a
**second** `topos-plugin-whatsapp` subprocess against the same data
directory the real, already-running instance holds an exclusive,
non-blocking `flock` on — the second launch always loses the race and
`Describe` always fails for a running WhatsApp instance. This is
unreachable from the e2e suite (which never builds a real WhatsApp
binary and intercepts these routes), so it would only surface against a
real deployment. Two further concurrency/robustness gaps are worth
tightening. See findings below.

## Critical Issues

### CR-01: `describePlugin`'s trial-launch always collides with a running WhatsApp instance's exclusive store lock, breaking "Edit match settings…" and "add existing instance to a second webspace" for WhatsApp

**File:** `web/src/routes/w/[webspace]/+page.svelte:192-221` (chip-menu "Edit match settings…"), `web/src/lib/components/AddSourceModal.svelte:163-182` (`selectExisting`, the "+ picker → existing instance" one-step flow)
**Also implicated:** `plugins/whatsapp/storelock.go:33-53` (`acquireStoreLock`, non-blocking exclusive `flock`), `plugins/whatsapp/connect.go:74-79` (`startBackgroundClient` acquires the lock unconditionally before doing anything else), `plugins/whatsapp/main.go:90-113` (the non-`-link`/`-link-json` path — the one `POST /api/config/describe-plugin` trial-launch uses — always reaches `NewSourcePlugin` → `startBackgroundClient` → `acquireStoreLock`)

**Issue:**

`POST /api/config/describe-plugin` (docs/api.md) works by spawning a
**brand-new** `topos-plugin-whatsapp` subprocess against the submitted
connection fields (`{plugin, source}`) and calling its `Describe` RPC,
then killing it. Two pre-existing UI flows (both untouched by this
phase, both reused for WhatsApp per `AddSourceModal.svelte`'s own
comment: *"an already-configured instance's stored Source trial-launches
identically to a not-yet-configured one"*) call this route against an
**already-configured, already-running** instance's stored `Source` in
order to learn its match vocabulary before rendering the match-fields
form:

- `AddSourceModal.svelte`'s `selectExisting()` — used when a WhatsApp
  instance already participates in webspace A and the operator opens
  the "+" picker in webspace B and picks that same instance to add it
  there too.
- `+page.svelte`'s `handleChipEdit(name, 'match')` — used by every
  source chip's "Edit match settings…" menu entry, including WhatsApp's
  (`SourceChip.svelte` renders this entry unconditionally for every
  source type, per `chip-edit-menu.test.ts`'s own "exactly four items"
  assertion — there is no WhatsApp exclusion).

Both call `describePlugin({ plugin: source.plugin, source })` where
`source` is `config.sources[instance]` — the **same** `path` the
already-running, pluginhost-launched instance for that name is using
right now (WhatsApp is unique among this repo's plugins in holding a
persistent connection for its entire process lifetime — `plugin.go`'s
own doc comment — so this instance is *always* running whenever the
kernel is up and the source is configured).

The kernel's trial-launch path spawns the plugin binary the same way a
real launch does (`WEBSPACES_SOURCE_CONFIG` env var, no special "trial"
flag exists anywhere in `plugins/whatsapp/main.go`), so the new
subprocess reaches `NewSourcePlugin(ctx, dataDir)` →
`startBackgroundClient` → `acquireStoreLock(p.dir)` unconditionally,
**before** it can ever answer a `Describe` RPC. `acquireStoreLock` takes
a non-blocking exclusive `flock` (`LOCK_EX|LOCK_NB`) and returns
`ErrStoreInUse` immediately if the lock is already held — which it
always is, held by the real running instance. `NewSourcePlugin` then
returns that error, `main.go`'s `fatal()` exits the process before
`goplugin.Serve` is ever reached, and the go-plugin handshake never
completes — so `POST /api/config/describe-plugin` **always** fails with
`502 plugin_describe_failed` for an already-configured, running
WhatsApp instance.

Net effect: for any WhatsApp source that is currently linked and
running (the normal, intended state once Phase 8's own QR-pairing flow
succeeds), an operator can never use "Edit match settings…" on that
chip — the modal opens but silently renders an empty match-fields form
(the `catch` branch in `handleChipEdit` swallows the failure and leaves
`editVocabulary = []`, with no error surfaced to the user, so it reads
as "there are no fields," not as a failure) — and can never add that
same instance to a second webspace via the "+" picker's one-step
existing-instance flow (which *does* surface the `plugin_describe_failed`
error, but offers no recovery). This is core, expected functionality
for a phase whose whole purpose is WhatsApp group/contact matching, and
it is unreachable from `make e2e` (the harness never builds a real
`topos-plugin-whatsapp` binary and intercepts every `describe-plugin`
call in `uat-08-whatsapp-qr-link.spec.ts`), so it will only be
discovered against a real deployment.

**Fix:** Either (a) have `DescribePluginHandler`/the plugin host special-case
an instance that is already launched and running — read the vocabulary
the already-completed `Describe` call at launch time already returned
(pluginhost presumably calls `Describe` once per launched instance for
`source_type` resolution already) rather than trial-launching a second
process, or (b) have the WhatsApp plugin's trial-launch path skip
`acquireStoreLock`/`startBackgroundClient` entirely when only `Describe`
is being requested (a "describe-only" mode analogous to `-link`/
`-link-json`, since `Describe` needs no live connection or store access
at all — it returns static constants). Whichever fix is chosen, extend
`uat-08-whatsapp-qr-link.spec.ts` (or a Go-level test against a real
built `topos-plugin-whatsapp` binary) to cover "Edit match settings…"
against an already-linked instance, since this is exactly the gap that
let the defect ship unnoticed.

## Warnings

### WR-01: WhatsApp link-session concurrency cap is enforced after the subprocess is already spawned

**File:** `kernel/httpapi/whatsapplink.go:472-496`
**Issue:** `WhatsAppLinkStartHandler` calls `spawner(context.Background(), binPath, req.Path)` (line 472) — which, in production, execs the plugin binary, opens its two databases and takes the exclusive store lock — **before** calling `store.register(sess)` (line 488), which is the only place `maxConcurrentLinkSessions` (4) is enforced. So the cap only ever limits how many sessions can be *held open*, not how many subprocesses can be *started concurrently*: N simultaneous `POST /api/config/whatsapp-link` requests (e.g. a client issuing repeated rapid retries, or several browser tabs) will spawn N subprocesses regardless of N, with only the first 4 surviving `register()` — the rest are killed immediately after paying the full cost of a process spawn, a SQLite open, and a `flock` acquire/release. This weakens (without fully defeating) the "a stuck or abandoned browser tab cannot accumulate unbounded subprocesses" guarantee the comment above `maxConcurrentLinkSessions` describes, and — because two of these could legitimately be racing for the *same* data directory's `flock` — makes the transient `whatsapp_store_in_use` error reachable even from this kernel's own over-eager spawning, not only from a genuinely conflicting external process.
**Fix:** Reserve a slot (e.g. an atomic counter or buffered-channel semaphore sized `maxConcurrentLinkSessions`) before calling `spawner`, and release it in the same paths that currently call `store.register`'s failure branch / session retirement — so the cap is enforced before any subprocess is started, matching the ordering discipline `WhatsAppLinkStartHandler`'s own doc comment already applies to the plugin-binary allowlist check ("directory listing, never a caller-supplied name, is the authority over what may be launched … BEFORE anything is executed").

### WR-02: A concurrent, unrelated config save can fail while a WhatsApp re-link session is suspending an instance

**File:** `kernel/supervisor/supervisor.go:233-269` (`SuspendInstance`), `kernel/supervisor/supervisor.go:430-519` (`Apply`)
**Issue:** `SuspendInstance` reconciles the host down to every source *except* the named one, but never removes that source from `s.cfg.Sources` — it only mutates the launched process set, not the config-of-record. If a config save/reload lands on **any other route** (e.g. an unrelated webspace filter edit, or a totally unrelated source's connection edit) while a WhatsApp re-link session is in flight (up to 5 minutes, `linkSessionDeadline`), `Apply`'s `s.host.Reconcile(ctx, newCfg.Sources, s.logger)` call will see the suspended instance still present in `newCfg.Sources` (nothing about the suspension is visible to `Apply`) and attempt to relaunch it — racing the live `-link-json` subprocess for the same `whatsapp.lock`. That relaunch attempt fails (`ErrStoreInUse`, surfaced as a generic launch failure through `pluginhost.Host.Reconcile`), which makes `Apply`'s pre-Reconcile-commit failure branch fire: the *entire, otherwise-valid* unrelated config save is rejected as `500 apply_failed`, and the running kernel's `s.cfg` is left pointing at the old generation while `config.toml` on disk already reflects the new one (a state divergence this code path already documents as a known, if rare, condition for other causes — this phase adds a new, easily-triggered way to reach it: simply having a WhatsApp Re-link… dialog open in one browser tab while saving anything else in another).
**Fix:** Have `Apply` (or `Host.Reconcile`) skip relaunching a source that is currently suspended (e.g. by having `Supervisor` track a small "suspended instance names" set that `Apply` subtracts from `newCfg.Sources` before calling `Reconcile`, restoring it once the caller's own resume closure runs), or have `SuspendInstance` hold `s.mu` for its entire duration rather than just across the initial `Reconcile` call (trading the current "an Apply can interleave with a suspension" behavior for "an Apply blocks until the link session ends," which is bounded by `linkSessionDeadline` and arguably the more surprising alternative to a user, but at least fails safe rather than surprisingly).

## Info

### IN-01: `storeLock.Release` silently drops the file-close error when unlock also fails

**File:** `plugins/whatsapp/storelock.go:60-67`
**Issue:** When `syscall.Flock(..., LOCK_UN)` fails, `Release` returns that error and never reports `l.f.Close()`'s own return value (only `closeErr` is returned on the *unlock-succeeded* path). In practice an `flock(LOCK_UN)` failure is rare and the fd is closed either way (best-effort), so this is not a correctness bug — the OS will drop the lock at process exit regardless (as the function's own doc comment notes) — but a caller diagnosing an unusual `Release` failure loses the close error entirely.
**Fix:** `return errors.Join(unlockErr, closeErr)` instead of the early return, so a genuine double-fault is reported instead of discarding one half.

---

_Reviewed: 2026-08-10T14:40:46Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
