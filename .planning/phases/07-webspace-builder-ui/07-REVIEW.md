---
phase: 07-webspace-builder-ui
reviewed: 2026-08-08T00:00:00Z
depth: standard
files_reviewed: 89
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
  info: 0
  total: 3
status: issues_found
---

# Phase 07: Code Review Report

**Reviewed:** 2026-08-08T00:00:00Z
**Depth:** standard
**Files Reviewed:** 89
**Status:** issues_found

## Summary

This is a re-review of Phase 07 (webspace-builder-ui) after gap-closure
plans 07-07 and 07-08 landed to close CR-01 (agent routes held a
boot-time config snapshot) and CR-02 (EditSourceModal kept stale form
state across cancel/reopen) from the prior review. This report
overwrites the prior 07-REVIEW.md, which predates those plans.

**Both prior findings are verified closed and well-tested:**

- **CR-01** — every `/agent/v1` handler in `kernel/httpapi/agent.go` now
  resolves `cfgStore.Expanded()` as the first statement of its own
  request closure (`agentSourcesHandler`, `agentWebspacesHandler`,
  `agentStreamHandler`, `agentItemHandler`, `agentRenditionHandler`), and
  `MountAgentRoutes` itself holds no config value at all. This is backed
  by both a behavioral test
  (`agent_live_config_test.go`'s `TestAgentLiveConfig_*` pair, proving a
  revoke/grant round-trips through the *same* already-constructed router
  with no restart) and a structural AST guard
  (`TestAgentGuard_EveryHandlerResolvesConfigPerRequest`) that pins the
  invariant against a future handler silently reintroducing a
  boot-time snapshot. Confirmed correct by direct reading of `agent.go`.
- **CR-02** — `web/src/routes/w/[webspace]/+page.svelte`'s
  `handleEditClose`/`handleEditSaved` now both route through the single
  `resetEditSession()` function, which nulls `editInstance` and thereby
  destroys the `{#if configResponse && editInstance}`-gated
  `EditSourceModal` subtree outright on every close path — the fix
  Task 1 needed. `EditSourceModal.svelte` additionally seeds its form
  state through two pure, testable helpers
  (`web/src/lib/edit-modal-state.ts`) and adds a second, defensive
  `untrack`-wrapped reset-on-open `$effect` as a belt-and-braces layer.
  Both the route-side and component-side fixes are backed by dedicated
  structural (`edit-modal-reset.test.ts`) and behavioral
  (`edit-modal-state.test.ts`) tests naming the CR-02 regression
  explicitly. Confirmed correct by direct reading of both files.

New findings from this pass, unrelated to CR-01/CR-02, are below: one
correctness bug in `kernel/supervisor/supervisor.go`'s `Apply` that can
leave the coordinator/scheduler pinned to a stale, partially-torn-down
plugin set after a specific failure ordering, and two narrower
front-end race/staleness gaps in the same family as CR-02 (a missing
generation guard, and a defensive reset effect that doesn't fully
deliver its own documented guarantee).

## Critical Issues

### CR-01: `Supervisor.Apply` leaves `coord`/`cfg`/scheduler pinned to stale config when `Host.Reconcile` succeeds but `ValidateMatchConfig` fails

**File:** `kernel/supervisor/supervisor.go:240-300` (specifically the second early-return branch, lines 254-257)

**Issue:** `Apply`'s error-handling is asymmetric across its two
early-return branches, and the asymmetry is unsound given what
`Host.Reconcile` actually does:

```go
if err := s.host.Reconcile(ctx, newCfg.Sources, s.logger); err != nil {
    s.startScheduler(oldCfg)
    return fmt.Errorf("supervisor: apply: %w", err)
}

if err := pluginhost.ValidateMatchConfig(newCfg, s.host); err != nil {
    s.startScheduler(oldCfg)
    return fmt.Errorf("supervisor: apply: %w", err)
}
```

`Host.Reconcile` (`kernel/pluginhost/host.go`) mutates `s.host.plugins`
**in place** on success: any instance whose `config.Source` changed is
relaunched (its old subprocess `Kill()`ed, a new `*Plugin` installed),
and any removed instance is killed and dropped. This mutation is real
and already committed to `s.host` by the time `Reconcile` returns
`nil` — there is no way to "undo" it.

When `Reconcile` succeeds but the immediately-following
`ValidateMatchConfig(newCfg, s.host)` call then fails (a live,
post-launch check — a match block naming a field the just-launched
plugin's real vocabulary doesn't declare, which cannot be caught by
`config.Store.Save`'s own struct-level `Validate` since that runs
before any plugin is launched), the code:

- restarts the scheduler against **`oldCfg`** (the pre-apply source
  set/goroutines), but
- leaves **`s.coord`** untouched — still holding `correlate.Source`
  values (effectively `*pluginhost.Plugin` pointers) captured at the
  *previous* successful `Apply`/`NewSupervisor`, and
- leaves **`s.cfg`** at `oldCfg`.

For any source instance whose connection config actually changed in
this `Apply` call, its *old* `*Plugin` object — the one `s.coord` still
references — has just been `Kill()`ed by `Reconcile`, while `s.host`
now holds a *new* `*Plugin` for that same instance name that `s.coord`
has no reference to at all. The restarted scheduler goroutine for that
source (running against `oldCfg`) will go on calling
`Coordinator.Refresh` → `Match`/`Fetch` against the now-dead old
`*Plugin`'s gRPC client indefinitely, until some *later* `Apply` call
happens to succeed all the way through (rebuilding `s.coord` from
`s.host.Plugins()`). In the meantime that source's syncs fail
continuously with a transport error that gives the operator no signal
connecting it back to the earlier rejected save.

This is reachable through ordinary use: `config.Store.Save` validates
the config document's shape but cannot validate a match block against a
plugin's real, live-learned vocabulary (that check is deliberately
deferred to post-launch, per `ValidateMatchConfig`'s own doc comment) —
so a single `PUT /api/config` that both edits a source's connection
details *and* introduces an invalid match field name for a webspace
will hit exactly this ordering: `Reconcile` succeeds (new connection
launches fine), `ValidateMatchConfig` fails (bad field name), and the
supervisor is left in the inconsistent state described above.

Contrast with the third failure family in the same function
(`DeleteSourceItems`/`DeleteSyncRuns` for a removed instance, lines
263-276): those branches correctly commit `s.coord = newCoordinator(...)`
and `s.cfg = newCfg` before returning their own error, precisely
*because* `Reconcile` (and, transitively, the match-config check) had
already succeeded by that point. The `ValidateMatchConfig` failure
branch is the one place in this function that leaves `s.host` mutated
but `s.coord`/`s.cfg` uncommitted.

**Fix:** Either (a) roll `s.coord`/`s.cfg` forward to `newCfg` in the
`ValidateMatchConfig` failure branch too (since `s.host` already
reflects `newCfg` regardless of what the caller is told), restarting
the scheduler against `newCfg` rather than `oldCfg` so its goroutine
set matches what `s.coord` actually holds:

```go
if err := pluginhost.ValidateMatchConfig(newCfg, s.host); err != nil {
    s.coord = newCoordinator(s.idx, newCfg, s.host)
    s.cfg = newCfg
    s.startScheduler(newCfg)
    return fmt.Errorf("supervisor: apply: %w", err)
}
```

or (b), if leaving the running kernel on the last-known-good match
configuration is the intended behavior on this failure path,
`Host.Reconcile` must be re-invoked (or a symmetric rollback method
added) to relaunch the reconciled instances back against `oldCfg.Sources`
before restarting the scheduler with `oldCfg` — so `s.host`, `s.coord`
and the scheduler generation always agree on which config they're
running. Either fix should be paired with a test exercising exactly
this ordering (`Host.Reconcile` succeeds, `ValidateMatchConfig` fails),
which `supervisor_test.go` does not currently cover — the existing
`TestApply_MidFlightSyncLeavesNoStrandedRunningRow` deliberately makes
`Reconcile` itself fail (a missing binary), never exercising the
downstream `ValidateMatchConfig` branch.

## Warnings

### WR-01: `handleChipEdit`'s match-mode `describePlugin` call has no request-generation guard

**File:** `web/src/routes/w/[webspace]/+page.svelte:166-188`

**Issue:** `handleChipEdit` synchronously reassigns `editInstance`,
`editMode` and `editVocabulary = []` at the top of every call, then —
only for `kind === 'match'` — awaits `describePlugin(...)` before
assigning `editVocabulary = resp.match_vocabulary` and finally
`editOpen = true`. There is no sequence/generation token guarding the
async assignment the way `navGeneration`/`searchRequestSeq` guard every
other async call site in this same file (`load`, `handleSearch`,
`loadConfig`).

If a user opens "Edit match settings…" on one source chip and then,
before that `describePlugin` call resolves, opens "Edit match
settings…" (or "Edit connection…") on a *different* chip, the two
calls' synchronous prefixes race: the second call's
`editInstance`/`editMode` win immediately, but the **first** call's
in-flight `describePlugin` promise can resolve *after* the second call
has already set up its own state (or already opened the modal),
overwriting `editVocabulary` with the first (now-wrong) instance's
vocabulary and re-asserting `editOpen = true` a second time. The
visible effect is the match-settings modal briefly showing (or
persisting) the wrong instance's vocabulary fields, or reopening after
the user closed it.

**Fix:** Capture a local generation/sequence value (mirroring this same
file's `navGeneration` pattern) before the `await`, and guard both the
`editVocabulary` assignment and the final `editOpen = true` on it still
matching the current generation:

```ts
async function handleChipEdit(name: string, kind: 'connection' | 'match' | 'remove') {
  if (kind === 'remove') {
    await handleRemoveSource(name);
    return;
  }
  if (!configResponse) return;
  const gen = ++editGeneration;
  editInstance = name;
  editMode = kind;
  editVocabulary = [];
  if (kind === 'match') {
    try {
      const source = configResponse.config.sources[name];
      const resp = await describePlugin({ plugin: source.plugin, source });
      if (gen !== editGeneration) return; // superseded by a later chip-edit click
      editVocabulary = resp.match_vocabulary;
    } catch {
      // ...
    }
  }
  if (gen !== editGeneration) return;
  editOpen = true;
}
```

### WR-02: `EditSourceModal`'s defensive reset-on-open effect does not actually re-seed `MatchFieldsForm`'s own state, contrary to its doc comment's claim

**File:** `web/src/lib/components/EditSourceModal.svelte:70-94, 178-182`

**Issue:** The reset-on-open `$effect` is documented as "a second,
defensive layer: it re-runs [the seeding helpers] whenever `open`
transitions to true, so a caller that keeps this component mounted
across a close … still cannot resurface a discarded session's typing."
For `mode === 'connection'` this is accurate — `ConnectionForm` is a
fully controlled component (`values={connectionValues}`) with no
internal echo state, so resetting `connectionValues` is sufficient.

For `mode === 'match'`, this claim does not hold. `MatchFieldsForm`
(`web/src/lib/components/MatchFieldsForm.svelte:31-33`) seeds its own
local `text` state **once, at its own mount**, from the initial
`values` prop, and deliberately never re-derives it from `values` on a
later prop change (per that component's own doc comment, this is
correct behavior for the common case: it stops an `onchange` round trip
from fighting the user's typing). `EditSourceModal` renders
`MatchFieldsForm` with no `{#key}` around it, so if `open` is ever
toggled false→true while `EditSourceModal` itself stays mounted (the
exact scenario this effect's doc comment names as the reason for its
own existence), the effect correctly resets the *parent's* `matchBlock`
state — but `MatchFieldsForm`'s own `text` (what the input fields
actually render) does not follow, because `MatchFieldsForm` itself was
never remounted.

Every current caller of `EditSourceModal` (`+page.svelte`,
`ManageSourcesModal.svelte`) happens to *always* destroy and remount
`EditSourceModal` on close (`{#if editInstance}` / `{#if
configResponse && editInstance}` gates, both cleared to `null` before
any subsequent reopen), so this gap is not reachable today — but that
also means the "defensive layer" this effect is documented as providing
is, for match mode specifically, not actually delivered; a future
caller that relies on the doc comment's stated guarantee (rather than
independently re-deriving the same "always destroy on close" discipline
every existing caller happens to follow) would reintroduce a
CR-02-shaped bug one level deeper, inside `MatchFieldsForm`, with no
test catching it — `edit-modal-reset.test.ts`'s guard only checks that
the effect *calls* `seedMatchBlock`/`untrack`, not that the result is
observable through `MatchFieldsForm`.

**Fix:** Either wrap `MatchFieldsForm` in a `{#key instance}` inside
`EditSourceModal` so a genuine re-seed also remounts the child, or
narrow the doc comment's claim to state plainly that the defensive
layer is complete for `connection` mode only and match-mode staleness
protection depends entirely on every caller destroying/remounting
`EditSourceModal` on close (and add a test pinning that caller
discipline, the way `edit-modal-reset.test.ts` already does for the
route).

---

_Reviewed: 2026-08-08T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
