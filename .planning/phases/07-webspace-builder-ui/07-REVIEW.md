---
phase: 07-webspace-builder-ui
reviewed: 2026-08-08T00:00:00Z
depth: standard
files_reviewed: 96
files_reviewed_list:
  - cmd/topos/main.go
  - config.example.toml
  - docs/api.md
  - go.mod
  - kernel/config/config.go
  - kernel/config/store.go
  - kernel/config/store_test.go
  - kernel/config/types.go
  - kernel/config/writer.go
  - kernel/config/writer_test.go
  - kernel/correlate/correlate_test.go
  - kernel/httpapi/agent.go
  - kernel/httpapi/agent_live_config_test.go
  - kernel/httpapi/agent_test.go
  - kernel/httpapi/config.go
  - kernel/httpapi/config_test.go
  - kernel/httpapi/contract_test.go
  - kernel/httpapi/item.go
  - kernel/httpapi/item_test.go
  - kernel/httpapi/live_config_test.go
  - kernel/httpapi/routes.go
  - kernel/httpapi/search.go
  - kernel/httpapi/search_test.go
  - kernel/httpapi/sources.go
  - kernel/httpapi/sources_test.go
  - kernel/httpapi/stream.go
  - kernel/httpapi/stream_test.go
  - kernel/httpapi/webspaces.go
  - kernel/index/store.go
  - kernel/index/store_test.go
  - kernel/pluginhost/describe_test.go
  - kernel/pluginhost/discover_binaries.go
  - kernel/pluginhost/discover_binaries_test.go
  - kernel/pluginhost/host.go
  - kernel/pluginhost/reconcile_test.go
  - kernel/supervisor/supervisor.go
  - kernel/supervisor/supervisor_test.go
  - kernel/syncer/coordinator_test.go
  - kernel/syncer/scheduler.go
  - README.md
  - web/src/lib/api.ts
  - web/src/lib/components/AddSourceModal.svelte
  - web/src/lib/components/add-source.test.ts
  - web/src/lib/components/chip-edit-menu.test.ts
  - web/src/lib/components/ConnectionForm.svelte
  - web/src/lib/components/CreateWebspaceModal.svelte
  - web/src/lib/components/edit-modal-reset.test.ts
  - web/src/lib/components/EditSourceModal.svelte
  - web/src/lib/components/FilterChip.svelte
  - web/src/lib/components/filter-chip.test.ts
  - web/src/lib/components/ManageSourcesModal.svelte
  - web/src/lib/components/manage-sources.test.ts
  - web/src/lib/components/MatchFieldsForm.svelte
  - web/src/lib/components/save-filter-clone.test.ts
  - web/src/lib/components/save-state.test.ts
  - web/src/lib/components/SecretField.svelte
  - web/src/lib/components/secret-field.test.ts
  - web/src/lib/components/SourceChip.svelte
  - web/src/lib/components/ui/alert-dialog/alert-dialog-action.svelte
  - web/src/lib/components/ui/alert-dialog/alert-dialog-cancel.svelte
  - web/src/lib/components/ui/alert-dialog/alert-dialog-content.svelte
  - web/src/lib/components/ui/alert-dialog/alert-dialog-description.svelte
  - web/src/lib/components/ui/alert-dialog/alert-dialog.svelte
  - web/src/lib/components/ui/alert-dialog/alert-dialog-title.svelte
  - web/src/lib/components/ui/alert-dialog/index.ts
  - web/src/lib/components/ui/dialog/dialog-content.svelte
  - web/src/lib/components/ui/dialog/dialog-footer.svelte
  - web/src/lib/components/ui/dialog/dialog-header.svelte
  - web/src/lib/components/ui/dialog/dialog.svelte
  - web/src/lib/components/ui/dialog/dialog-title.svelte
  - web/src/lib/components/ui/dialog/dialog-trigger.svelte
  - web/src/lib/components/ui/dialog/index.ts
  - web/src/lib/components/ui/dropdown-menu/dropdown-menu-content.svelte
  - web/src/lib/components/ui/dropdown-menu/dropdown-menu-item.svelte
  - web/src/lib/components/ui/dropdown-menu/dropdown-menu-separator.svelte
  - web/src/lib/components/ui/dropdown-menu/dropdown-menu.svelte
  - web/src/lib/components/ui/dropdown-menu/dropdown-menu-trigger.svelte
  - web/src/lib/components/ui/dropdown-menu/index.ts
  - web/src/lib/components/ui/overlay-primitives.test.ts
  - web/src/lib/components/unknown-config-keys.test.ts
  - web/src/lib/components/WebspaceHeader.svelte
  - web/src/lib/components/WebspaceSwitcher.svelte
  - web/src/lib/components/webspace-switcher.test.ts
  - web/src/lib/config-edit.test.ts
  - web/src/lib/config-edit.ts
  - web/src/lib/edit-modal-state.test.ts
  - web/src/lib/edit-modal-state.ts
  - web/src/lib/instance-id.test.ts
  - web/src/lib/instance-id.ts
  - web/src/lib/last-webspace.test.ts
  - web/src/lib/last-webspace.ts
  - web/src/lib/node-builtins.d.ts
  - web/src/lib/plugin-fields.test.ts
  - web/src/lib/plugin-fields.ts
  - web/src/routes/+page.svelte
  - web/src/routes/w/[webspace]/+page.svelte
findings:
  critical: 1
  warning: 2
  info: 1
  total: 4
status: issues_found
---

# Phase 07: Code Review Report (re-review after 07-09 gap closure)

**Reviewed:** 2026-08-08T00:00:00Z
**Depth:** standard
**Files Reviewed:** 96
**Status:** issues_found

## Summary

This is a re-review after 07-09's gap closure of the previously-filed CR-01
(`supervisor.Apply` leaving a stale coordinator dispatching against a
killed plugin host after a post-Reconcile failure). I traced
`commitGeneration` and every one of `Apply`'s error branches line by line
against `supervisor_test.go`'s four tests, plus `pluginhost.Host.Reconcile`,
`config.Store.Save`/`Reload`, and the live-config-resolution discipline the
`/api/*` and `/agent/v1/*` handlers now share.

**The specific defect named in the prior review (gaps[0]) is fixed.**
`commitGeneration` is now the single site every post-Reconcile failure
branch (`ValidateMatchConfig` failure, both D-07 index-cleanup failures)
funnels through, and it performs coordinator-rebuild → `s.cfg` update →
scheduler-restart in the doc-comment-specified order, so `s.host`,
`s.coord`, `s.cfg` and the running scheduler generation are always drawn
from one config generation on every exit path. `supervisor_test.go`'s
`TestApply_ValidateMatchConfigFailsAfterReconcile_CoordinatorTracksRelaunchedPlugin`
and `TestApply_RejectedSaveIsIdempotent_SecondApplyDoesNotRelaunchSubprocesses`
exercise this convincingly.

However, tracing the D-07 removed-instance cleanup loop against the same
`Apply` control flow surfaced a **new, closely related defect in the same
function**: the cleanup loop that deletes a removed source instance's
index rows and sync history is skipped entirely when the post-Reconcile
`ValidateMatchConfig` check fails, and can also abort partway through when
it has multiple instances to clean up — and in both cases there is no
retry path, ever, because `s.cfg` has already been committed to the new
generation by the time the failure is observed. This is the same root
cause class as the original gaps[0] (a post-Reconcile failure branch that
doesn't fully carry out the state-repair `commitGeneration` was built to
guarantee), just on the D-07 cleanup seam instead of the coordinator seam.
I classify it as a BLOCKER because it silently and permanently violates a
documented, tested invariant (T-07-13) with a real (if narrower) reach path
through the hand-edit-plus-reload flow this phase explicitly supports.

The rest of the phase — the config write/read path (`kernel/config`), the
live-config-resolution discipline across `/api/*` and `/agent/v1/*`
(`kernel/httpapi`), the builder UI's optimistic-update/error-contract
pattern across every modal, and the D-16/D-17/D-18 filter-narrowing
plumbing in `kernel/index` — is careful, internally consistent, and well
covered by targeted regression tests for the specific defects the phase's
own commit history already found and fixed (CR-01/CR-02 in the frontend,
gaps[0] in the supervisor). Two lower-severity issues are noted below.

## Critical Issues

### CR-01: `supervisor.Apply`'s D-07 cleanup is skipped or partially applied on a post-Reconcile failure, permanently orphaning a removed instance's index rows

**File:** `kernel/supervisor/supervisor.go:307-377`

**Issue:** `Apply`'s control flow is:

```go
if err := s.host.Reconcile(ctx, newCfg.Sources, s.logger); err != nil {
    s.startScheduler(oldCfg)
    return err                                   // pre-commit: OK, unchanged
}

if err := pluginhost.ValidateMatchConfig(newCfg, s.host); err != nil {
    s.commitGeneration(newCfg)
    return err                                   // <-- returns here, BEFORE
}                                                  //     the loop below ever runs

for _, name := range removedInstances(oldCfg, newCfg) {
    if err := s.idx.DeleteSourceItems(ctx, name); err != nil {
        s.commitGeneration(newCfg)
        return err                                // <-- returns here, stopping
    }                                              //     the loop mid-iteration
    if err := s.idx.DeleteSyncRuns(ctx, name); err != nil {
        s.commitGeneration(newCfg)
        return err                                // <-- same, after items but
    }                                              //     before sync_runs for `name`
}

s.commitGeneration(newCfg)
```

Two related ways this strands data permanently:

1. **Skipped entirely on a `ValidateMatchConfig` failure.** If a single
   `Apply` call both removes instance `X` from config *and* introduces an
   unrelated match-vocabulary violation for some other still-configured
   instance `Y` (a config shape `config.Validate` cannot itself reject,
   since the vocabulary check only runs post-launch against a live
   plugin — see `ValidateMatchConfig`'s own doc comment), the function
   returns from the `ValidateMatchConfig` branch before the D-07 loop is
   ever reached. `X`'s subprocess has already been killed by `Reconcile`
   (which runs and commits *before* this check), but `X`'s `items` rows
   and `sync_runs` history are never deleted.

2. **Partial iteration on a mid-loop `DeleteSourceItems`/`DeleteSyncRuns`
   failure.** If `removedInstances(oldCfg, newCfg)` names more than one
   instance and an early one fails to clean up (a SQL-level error), the
   loop returns immediately — every instance later in the sorted name
   order is never attempted.

In both cases, `s.commitGeneration(newCfg)` runs on the failing branch,
which sets `s.cfg = newCfg`. On the *next* `Apply` call, `oldCfg` is now
`newCfg` — which already lacks the stranded instance(s) — so
`removedInstances(oldCfg, newCfg2)` can never again compute that instance
as "removed": it isn't present in either side of the diff from that point
forward. There is no retry path; the orphaned `items`/`sync_runs` rows for
that instance persist in the index forever, or until a re-added instance
under the same `[sources.<id>]` key inherits them as phantom history —
exactly the outcome `DeleteSourceItems`/`DeleteSyncRuns`'s own doc
comments (`kernel/index/store.go:291-316`) and T-07-13 name as the
guarantee this cleanup exists to provide.

This is reachable through the documented, supported hand-edit +
`POST /api/config/reload` flow (`ManageSourcesModal.svelte`'s "Reload
config" button): a single hand-edited `config.toml` that both removes a
`[sources.<id>]` block and adds/typos a
`[webspaces.<name>.match.<other-instance>]` field outside that instance's
declared vocabulary hits this exact combination in one `Apply` call.
`supervisor_test.go` has no test covering a removed instance combined with
a post-Reconcile failure (grepped for `removedInstances`/
`DeleteSourceItems`/`DeleteSyncRuns` — no hits outside `supervisor.go`
itself and `kernel/index`), so this gap is not caught by the existing
regression suite the 07-09 gap closure otherwise relies on. Confirmed via
`git show f25c4ab:kernel/supervisor/supervisor.go` that this ordering
(vocabulary check before the D-07 loop) predates the 07-09 fix — it is not
a regression 07-09 introduced, but it is the same class of defect 07-09 set
out to close on the coordinator seam, left unclosed on this adjacent seam.

**Fix:** Run the D-07 cleanup loop unconditionally on every post-Reconcile
path — i.e., move it ahead of (or independent of) the `ValidateMatchConfig`
check, since `Reconcile` has already committed by the time either check
runs and the removed instances are already gone from the host regardless
of whether the vocabulary check subsequently rejects the save. Additionally,
make the loop itself not abort on a single instance's cleanup failure —
collect per-instance errors and continue to the next name, so one SQL
failure doesn't strand every other removed instance in the same batch:

```go
if err := s.host.Reconcile(ctx, newCfg.Sources, s.logger); err != nil {
    s.startScheduler(oldCfg)
    return fmt.Errorf("supervisor: apply: %w", err)
}

// D-07 cleanup must run for every post-Reconcile branch, and must not
// abort partway — Reconcile has already committed regardless of what
// ValidateMatchConfig or a later failure decides, so every removed
// instance's data must be cleared or the loss becomes permanent (see
// commitGeneration's own doc comment on why there is no retry path once
// s.cfg advances).
var cleanupErrs []error
for _, name := range removedInstances(oldCfg, newCfg) {
    if err := s.idx.DeleteSourceItems(ctx, name); err != nil {
        cleanupErrs = append(cleanupErrs, fmt.Errorf("delete items for removed source %q: %w", name, err))
        continue
    }
    if err := s.idx.DeleteSyncRuns(ctx, name); err != nil {
        cleanupErrs = append(cleanupErrs, fmt.Errorf("delete sync history for removed source %q: %w", name, err))
    }
}

matchErr := pluginhost.ValidateMatchConfig(newCfg, s.host)

s.commitGeneration(newCfg)

if err := errors.Join(append(cleanupErrs, matchErr)...); err != nil {
    return fmt.Errorf("supervisor: apply: %w", err)
}
```

## Warnings

### WR-01: `WebspacesHandler`'s per-webspace `last_sync` is actually a global aggregate across every configured source

**File:** `kernel/httpapi/webspaces.go:36-64`

**Issue:** `runs, err := store.LatestSyncRunPerSource(ctx)` and
`aggregateSyncStatus(runs)` are computed **once**, outside the `for _, name
:= range names` loop, and the resulting single `lastSync` value is then
assigned to every `webspaceSummary.LastSync` field. A webspace whose own
participating sources are all healthy will report `"error"` if *any other*
unrelated, differently-scoped source elsewhere in the config has a failing
latest run — the aggregate is never narrowed to the sources that actually
participate in that particular webspace (contrast with `StreamHandler`,
`SearchHandler` and `agentStreamHandler`, which all correctly narrow via
the webspace's own participating/granted set).

This predates this phase (confirmed via `git diff 5329270..HEAD --
kernel/httpapi/webspaces.go`, which only threads `cfgStore` through for
live-read purposes and does not touch this aggregation), so it is not a
regression introduced by this phase's changes — but it was directly
adjacent code this phase touched and is worth surfacing since `GET
/api/webspaces`'s `last_sync` field feeds `WebspaceSwitcher`/dashboard-style
UI this phase built on top of.

**Fix:** Compute `runs` once (as now, for efficiency) but filter it per
webspace to the sources actually participating in that webspace (via
`config.Webspace.Participates`) before calling `aggregateSyncStatus`,
mirroring `agentWebspacesHandler`'s existing `filterRunsByGrant` pattern
but keyed on participation instead of grant:

```go
for _, name := range names {
    ws := cfg.Webspaces[name]
    participating := map[string]index.SyncRun{}
    for source, run := range runs {
        if ws.Participates(source) {
            participating[source] = run
        }
    }
    resp.Webspaces = append(resp.Webspaces, webspaceSummary{
        Name:      name,
        Keywords:  keywordsOrEmpty(ws.Keywords),
        ItemCount: counts[name],
        LastSync:  aggregateSyncStatus(participating),
    })
}
```

### WR-02: `resolveNewInstanceId` derives instance ids from unauthenticated, un-normalized display-name text with no length/reserved-word bound

**File:** `web/src/lib/instance-id.ts:40-46`

**Issue:** `deriveInstanceId` lowercases, collapses non-`[a-z0-9]` runs to
a single hyphen, and strips leading/trailing hyphens — but places no upper
bound on the resulting string's length (a very long pasted display name
produces an equally long `[sources.<id>]` TOML table key) and does not
guard against deriving to a name that collides with a reserved/structural
concept (e.g. a display name of `"__trial__"` — the exact literal
`DescribePluginType` (`kernel/pluginhost/host.go:340`) uses as its
fixed, non-persisted trial-launch instance name). A collision here is
almost certainly harmless in practice (the trial-launch path never touches
persisted config), but it's the kind of unvalidated derived-identifier
path where "harmless today" is an easy invariant to break silently later
(e.g. if a future change starts keying anything off that literal). This is
purely a robustness/defense-in-depth observation, not a demonstrated
exploit — `cfg.Sources[candidateId]` collision-checking already covers the
one case that matters today (an actual instance-id clash).

**Fix:** Consider capping `deriveInstanceId`'s output length (config keys
this long are already awkward for `config.toml` hand-editing, which this
project explicitly supports as a first-class path) and/or reserving the
`__trial__` literal (and any other kernel-internal sentinel instance
names) alongside the existing `cfg.sources[candidateId]` collision check
in `resolveNewInstanceId`.

## Info

### IN-01: `Store.WriteCanonical`'s temp file inherits `os.CreateTemp`'s default `0o600` implicitly rather than declaring it

**File:** `kernel/config/writer.go:57-60`

**Issue:** `os.CreateTemp(dir, ".config-*.toml")` relies on the stdlib's
own default file mode (`0o600`) to keep a config file that may carry
`${VAR}`-referenced-but-still-sensitive-looking content non-world-readable
during the brief window before the atomic rename. This is already correct
behavior (matches the explicit `0o600` used two lines above for the
`.bak` file), but it's implicit rather than stated — a future refactor
that swaps `os.CreateTemp` for a different temp-file helper (or changes
the process umask assumption) could silently regress file permissions
with no test catching it, since nothing here asserts the temp file's mode.

**Fix:** No behavior change needed; consider a comment noting the
reliance on `os.CreateTemp`'s `0o600` default (matching this file's own
practice of documenting every other security-relevant decision inline),
or a defensive `tmp.Chmod(0o600)` immediately after creation for
robustness against a future Go stdlib default change.

---

_Reviewed: 2026-08-08T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
