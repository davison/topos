---
phase: 07-webspace-builder-ui
reviewed: 2026-08-08T00:00:00Z
depth: standard
files_reviewed: 78
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
  - web/src/lib/last-webspace.test.ts
  - web/src/lib/last-webspace.ts
  - web/src/lib/node-builtins.d.ts
  - web/src/lib/plugin-fields.test.ts
  - web/src/lib/plugin-fields.ts
  - web/src/routes/+page.svelte
  - web/src/routes/w/[webspace]/+page.svelte
findings:
  critical: 1
  warning: 5
  info: 0
  total: 6
status: issues_found
---

# Phase 07: Code Review Report

**Reviewed:** 2026-08-08T00:00:00Z
**Depth:** standard
**Files Reviewed:** 78 (Go kernel + SvelteKit web UI)
**Status:** issues_found

## Summary

This phase adds the webspace-builder UI: config read/write (`GET`/`PUT /api/config`, `POST /api/config/reload`), plugin-type discovery/describe, the "+" add-source picker (one-step existing-instance and two-step new-plugin-type flows), the chip edit menu (edit connection/edit match/remove), the Manage Sources escape hatch (instance/webspace delete, config reload), and the search-promotion permanent-filter feature. The Go side is careful and heavily guarded by AST tests pinning the "config write path never reaches a plugin RPC beyond Describe" invariant, and the D-03/D-05 secret/hash-lock discipline is well tested at both the `config.Store` and HTTP layers.

The most serious defect found is in the frontend: `AddSourceModal.svelte`'s "Save anyway" path (the connection-only save offered after a failed connection test) omits the duplicate-instance-id guard its sibling code path enforces, so editing the display name after a failed test can silently overwrite an existing, unrelated source instance's connection config and agent grants with no confirmation. The remaining findings are functional inconsistencies between the human (`/api/*`) and agent (`/agent/v1/*`) webspace-list endpoints, missing client-side validation that lets a legitimate action fail with a confusing, unrelated kernel error, and a couple of lower-severity robustness gaps.

## Critical Issues

### CR-01: AddSourceModal "Save anyway" can silently overwrite an existing, unrelated source instance

**File:** `web/src/lib/components/AddSourceModal.svelte:242-269` (compare with `handleConnectNext`, lines 204-240)

**Issue:** In the two-step "New {plugin type}…" flow, when Step 1's `describePlugin` trial-launch fails, the modal offers a "Save anyway" button that persists the connection-only instance directly via `saveAnyway()`:

```ts
async function saveAnyway() {
    if (!selectedPluginType || savingConnectionOnly) return;
    const displayName = (connectionValues.display_name ?? '').trim();
    const candidateId = deriveInstanceId(displayName);
    if (candidateId === '') return;

    savingConnectionOnly = true;
    try {
        const nextConfig = upsertSourceInstance(config, candidateId, connectionValues);
        await putConfig({ base_hash: baseHash, config: nextConfig });
        ...
```

Unlike its sibling `handleConnectNext` (Step 1's "Next" submit handler), which explicitly rejects a `candidateId` that collides with an already-configured instance:

```ts
if (config.sources[candidateId]) {
    describeFailed = false;
    connectError = `An instance id "${candidateId}" already exists — choose a different display name.`;
    return;
}
```

`saveAnyway()` performs **no such check**. The user is free to keep editing `connectionValues.display_name` (the `ConnectionForm` remains fully interactive after a failed describe) between the failed "Next" click and the "Save anyway" click. If the edited display name derives to an id that already exists (e.g. the user types the display name of an existing, working instance, or simply reverts an edit), `upsertSourceInstance(config, candidateId, connectionValues)` unconditionally does `next.sources[instanceId] = source`, clobbering that instance's entire `[sources.<id>]` block — including its `base_url`/`token` reference and, critically, its `agent` grants (`connectionValues.agent` is always freshly initialized to `{ read: false, handoff: false }` for the new-instance flow, so an existing instance's `agent.read = true` grant is silently reset to `false`).

This is reachable through ordinary UI interaction (no dev tools needed), requires no confirmation dialog, and the resulting `PUT /api/config` succeeds (`base_hash` still matches, since nothing else changed the file) — so the overwrite is not caught by the D-03 clobber guard either. The victim instance's previously-working connection is silently replaced with the new, **unverified** (describe-failed) one, and its agent-read grant is silently revoked.

**Fix:** Reuse the same collision guard in `saveAnyway()` before calling `upsertSourceInstance`:

```ts
async function saveAnyway() {
    if (!selectedPluginType || savingConnectionOnly) return;
    const displayName = (connectionValues.display_name ?? '').trim();
    const candidateId = deriveInstanceId(displayName);
    if (candidateId === '') return;
    if (config.sources[candidateId]) {
        connectError = `An instance id "${candidateId}" already exists — choose a different display name.`;
        return;
    }
    ...
```

Consider factoring the id-derivation + collision-check into one shared helper so this class of bug can't recur when a third save path is added later.

## Warnings

### WR-01: GET /api/webspaces item_count is not narrowed by the webspace's saved filter, unlike GET /agent/v1/webspaces

**File:** `kernel/httpapi/webspaces.go:36-45` vs `kernel/httpapi/agent.go:121-138,146-181`

**Issue:** `WebspacesHandler` computes each webspace's `item_count` via `store.Webspaces(ctx)` (`kernel/index/store.go:621-646`), which counts every row in `webspace_items` unconditionally — it has no `filterTerms` parameter and never reads `cfg.Webspaces[name].Filter`. `agentWebspacesHandler`, by contrast, computes `item_count` via `agentGrantedItemCount`, which calls `store.StreamItems(ctx, webspaceName, granted, cfg.Webspaces[name].Filter)` — i.e. it **does** narrow by the webspace's saved permanent filter, in addition to the grant restriction.

This directly contradicts the design intent stated in `kernel/config/types.go`'s `Webspace.Filter` doc comment ("the filtered view IS the webspace for every consumer, human and agent alike") and creates an observable inconsistency: for a fully-granted config with an active filter, `GET /api/webspaces` and `GET /agent/v1/webspaces` should report identical `item_count` values per `docs/api.md`'s own mirror table (which documents only a grant-based restriction, not a filter difference) but will not. `GET /api/webspaces/{webspace}/stream` (which is filtered) will also disagree with `GET /api/webspaces`'s own `item_count` for the same webspace. No test in this repo (`live_config_test.go`, `sources_test.go`, etc.) exercises `item_count` together with an active `Filter`, so this went unnoticed. The `item_count` field is not currently rendered anywhere in the web UI, which limits present-day user impact, but it is part of the published HTTP contract any external consumer (including a future UI feature) could rely on.

**Fix:** Either thread `filterTerms` through `index.Store.Webspaces` (mirroring `agentGrantedItemCount`'s approach) and use it from `WebspacesHandler`, or — if `item_count` is deliberately meant to report the raw indexed count regardless of filter — document that divergence explicitly in `docs/api.md` and remove the filter narrowing from `agentGrantedItemCount` so both surfaces agree.

### WR-02: CreateWebspaceModal has no client-side check for an existing webspace name

**File:** `web/src/lib/components/CreateWebspaceModal.svelte:52-76`, `web/src/lib/config-edit.ts:40-44`

**Issue:** `handleSubmit` calls `addWebspace(config, trimmed)` with no check that `trimmed` doesn't already exist in `config.webspaces`. `addWebspace` itself documents that a colliding name is "overwritten with an equally empty entry" and treats that as "a kernel-side load-time validation concern." In practice this means: typing an existing webspace's name into "New webspace" always produces a `PUT /api/config` that zeroes that webspace's `keywords`/`sources`/`match` (and therefore always fails kernel `Validate` with "declares neither a keywords fallback nor any match block", since every already-persisted webspace must have had at least one of those to have been saved in the first place). The save is correctly rejected (no actual data loss occurs), but the user sees a confusing validator error that has nothing to do with "this name is taken" — a poor, avoidable UX regression for a webspace-builder UI whose whole purpose is to shield users from raw config semantics.

**Fix:** Add a simple client-side check in `handleSubmit` (or as a `$derived` disabling the submit button) that flags `trimmed in config.webspaces` before calling `putConfig`, with a clear "A webspace named “X” already exists" message.

### WR-03: Adding a new source instance can retroactively invalidate an unrelated webspace elsewhere in the same config

**File:** `web/src/lib/config-edit.ts:128-145,181-189` (`addSourceToWebspace`, `upsertSourceInstance`), consumed by `AddSourceModal.svelte`'s `submitMatch`/`saveAnyway`

**Issue:** `upsertSourceInstance`/`addSourceToWebspace` only ever mutate the target webspace's own `sources`/`match` entries; they never inspect or adjust any *other* configured webspace. Per `kernel/config/config.go`'s `validateFallbackCoverage`, however, a brand-new source instance automatically "participates" in every *other* webspace whose `sources` allowlist is empty (the default, "every instance participates") — and if that other webspace also has an empty `keywords` fallback (i.e. it relies entirely on explicit `match` blocks, a shape `config.example.toml` itself demonstrates), the newly-added instance now has nothing to resolve its match input from there, and `Config.Validate` will reject the **entire** `PUT /api/config` with an error naming that unrelated webspace.

Concretely: a user adds a brand-new plugin instance from webspace **A**'s "+" picker; if webspace **B** (which the user never touched, may not even be visible in the current view) is match-only with a default (empty) `sources` allowlist, the add to **A** fails with a validator message about **B** — a highly confusing failure mode for a UI built specifically to avoid exposing raw config semantics.

**Fix:** At minimum, surface the kernel's `config_invalid` message clearly (already done generically via `err.message`), but consider either (a) having `addSourceToWebspace` proactively add the new instance's id to *every* affected match-only/no-allowlist webspace's own `sources` exclusion list as part of the same save (making the add strictly additive to the target webspace only), or (b) detecting this case client-side and explaining it in the error copy rather than showing the kernel's raw validator string verbatim.

### WR-04: kernel/config/writer.go's atomic write never fsyncs the containing directory

**File:** `kernel/config/writer.go:56-78`

**Issue:** `WriteCanonical` fsyncs the temp file (`tmp.Sync()`) before `os.Rename(tmpPath, path)`, which is necessary but not sufficient for POSIX crash-durability: without an additional `fsync` on the directory file descriptor after the rename, a power loss or hard kernel crash immediately after `os.Rename` returns can, on some filesystems/mount options (e.g. ext4 without `data=ordered`+journal commit having flushed, or certain non-Linux/network filesystems), leave the directory entry pointing at the old inode, or leave `config.toml` and `config.toml.bak` in an inconsistent pairing. The doc comment's guarantee ("a kernel killed mid-write leaves `config.toml` at its previous content, never truncated or half-written") holds for the ordinary "process killed mid-write" case this code already handles well, but is stronger than what's actually durable across a true power-loss event.

**Fix:** After the `os.Rename` succeeds, open and fsync the parent directory (`dir`) as well — the standard "atomic file replace" pattern. Low priority given the realistic threat model (desktop app, not a distributed system), but worth a one-line fix given how much of this phase's design rests on the write path's durability claims.

### WR-05: Removing the last explicitly-allowlisted source from a webspace silently re-admits every other configured source

**File:** `web/src/lib/config-edit.ts:217-232` (`removeSourceInstance`), acknowledged in its own doc comment (T-07-26) but not surfaced to the user anywhere in the UI

**Issue:** When an instance is deleted via "Manage sources… → Delete" and it was the *last* entry in some webspace's explicit `sources` allowlist, `removeSourceInstance` leaves that webspace's `sources` as `[]`. Per `Webspace.Participates` (`kernel/config/types.go:211-221`), an empty `sources` slice means "every configured instance participates by default" — so if that allowlist had been deliberately restrictive (e.g. explicitly excluding a third, unrelated instance), deleting the last named instance silently re-opens the webspace to every other configured source, including ones the user had intentionally excluded. This is a real, user-visible behavior change with no confirmation or explanation in `ManageSourcesModal`'s destructive-delete `AlertDialog` copy ("This removes {instance} from every webspace and deletes its indexed items. This can't be undone.") — the copy does not mention that a webspace's participation model may flip from restrictive to open-to-all as a side effect.

**Fix:** Either have the delete flow keep a (now-empty-of-real-instances) allowlist from silently reopening participation — e.g. write a sentinel that still excludes everything if that was the prior intent — or, more simply, update the `AlertDialogDescription` copy to warn when this specific case applies ("this webspace's remaining sources will change from an explicit list to 'all configured sources'").

---

_Reviewed: 2026-08-08T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
