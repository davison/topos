---
phase: 07-webspace-builder-ui
reviewed: 2026-08-08T14:56:06Z
depth: standard
files_reviewed: 74
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
  - web/src/lib/components/ManageSourcesModal.svelte
  - web/src/lib/components/MatchFieldsForm.svelte
  - web/src/lib/components/save-filter-clone.test.ts
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
  critical: 2
  warning: 2
  info: 1
  total: 5
status: issues_found
---

# Phase 07: Code Review Report

**Reviewed:** 2026-08-08T14:56:06Z
**Depth:** standard
**Files Reviewed:** 74 (of 91 files listed — 17 generated `ui/` primitive re-exports and trivial `index.ts` barrels were skimmed, not individually cited)
**Status:** issues_found

## Summary

This is a re-review of Phase 07 (webspace builder UI) after the 07-06 gap-closure plan. The prior CR-01 finding (`saveAnyway()` in `AddSourceModal.svelte` skipping the instance-id collision guard) is **verified fixed**: `web/src/lib/instance-id.ts` is now the single derivation/collision-check site, both `handleConnectNext` and `saveAnyway` call `resolveNewInstanceId` before every `upsertSourceInstance` write, and `add-source.test.ts` pins the invariant (guard-before-write, return-between, no local `deriveInstanceId` reappearing) structurally. Good.

This pass found two new BLOCKER-class issues the prior review didn't cover, both load-bearing:

1. A **security-relevant staleness bug** in the `/agent/v1` route mounting (`kernel/httpapi/agent.go`): four of five agent handlers close over a `*config.Config` snapshot captured once at server boot, not the live `*config.Store`. Revoking (or granting) `agent.read` through the UI's hot-apply config save does **not** take effect on `/agent/v1/sources`, `/agent/v1/webspaces`, `/agent/v1/items/{id}`, or `/agent/v1/items/{id}/content|thumbnail` until the kernel process restarts — directly contradicting this phase's own documented D-06 "hot apply, no restart" guarantee, and specifically undermining the one surface (AGENT-01's default-deny grant model) whose entire job is gating automated access to personal data.

2. A **stale-state bug** in the chip-menu "Edit connection…"/"Edit match settings…" flow (`EditSourceModal.svelte` + `web/src/routes/w/[webspace]/+page.svelte`): the modal's local `$state` (the actual field values, including base_url/token) is seeded once at mount and relies on the caller keying the component to force a remount on reopen. The route's own `handleEditClose`/`handleEditSaved` never clear `editInstance`, so canceling an edit and reopening the *same* source's edit modal shows the previously-typed-but-discarded values, not the current config — a user can silently re-save data they believed they'd canceled.

Two further WARNING-level robustness gaps and one INFO item are listed below.

## Critical Issues

### CR-01: `/agent/v1` routes (all but the stream route) never see a config save/reload — grant revocation has no effect until kernel restart

**File:** `kernel/httpapi/agent.go:391-399` (capture site), corroborated by `kernel/httpapi/routes.go:33-61` and `kernel/httpapi/live_config_test.go` (test coverage gap)

**Issue:** `MountAgentRoutes` is called exactly once, at `Router()` construction time in `runServe()`:

```go
func MountAgentRoutes(r chi.Router, store *index.Store, cfgStore *config.Store, fetcher Fetcher, prober HealthProber) {
	cfg := cfgStore.Expanded()
	r.Get("/agent/v1/sources", agentSourcesHandler(store, cfg, prober))
	r.Get("/agent/v1/webspaces", agentWebspacesHandler(store, cfg, prober))
	r.Get("/agent/v1/webspaces/{webspace}/stream", agentStreamHandler(store, cfgStore, prober))
	r.Get("/agent/v1/items/{id}", agentItemHandler(store, cfg, prober, fetcher))
	r.Get("/agent/v1/items/{id}/content", agentRenditionHandler(store, cfg, prober, fetcher, toposv1.ContentVariant_CONTENT_VARIANT_PREVIEW))
	r.Get("/agent/v1/items/{id}/thumbnail", agentRenditionHandler(store, cfg, prober, fetcher, toposv1.ContentVariant_CONTENT_VARIANT_THUMBNAIL))
}
```

`cfg` is a `*config.Config` pointer resolved **once**, then closed over by `agentSourcesHandler`, `agentWebspacesHandler`, `agentItemHandler` and `agentRenditionHandler` for the lifetime of the process. Only `agentStreamHandler` receives `cfgStore` itself and re-resolves `cfgStore.Expanded()` per request (`agent.go:199`).

Compare this to `kernel/httpapi/webspaces.go:31-33`, `kernel/httpapi/item.go:95-97`, and `kernel/httpapi/sources.go:159-161` (`SourceRefreshHandler`), which all explicitly resolve `cfg := cfgStore.Expanded()` as the *first statement inside the returned closure* — i.e., fresh per request. `routes.go`'s own doc comment (lines 37-44) states this was made true for every `/api/*` handler by 07-02-PLAN.md Task 2, and `live_config_test.go` mechanically proves it for exactly those three handlers. `agent.go`'s own doc comment (lines 384-390) claims the four boot-snapshotted agent handlers get "the same deliberately-temporary boot-snapshot treatment Router gives WebspacesHandler/ItemHandler/SourceRefreshHandler" — but that claim is now false for all three of those `/api/*` handlers, and no equivalent live-config test exists for the agent surface. The gap-closure this phase performed for `/api/*` was never extended to `/agent/v1/*`, except for the one stream route.

**Impact:**
- An operator who revokes a source's `agent.read`/`agent.handoff` grant via `EditSourceModal`/`AddSourceModal` (a hot-apply `PUT /api/config` or `POST /api/config/reload`, both of which call `Applier.Apply` immediately per D-06) will see the change reflected in `/api/*` and in the UI on the very next request — but an already-running agent client hitting `/agent/v1/sources`, `/agent/v1/webspaces`, `/agent/v1/items/{id}`, or the content/thumbnail routes keeps reading the grant set **as it was when the kernel process started**. A revoked grant continues to permit reads of that source's items (including full content via `agentItemHandler`/`agentRenditionHandler`) until the kernel is restarted. This is a live authorization-bypass window on the one surface (AGENT-01) explicitly designed to be default-deny and grant-scoped.
- The inverse also breaks correctness: a newly added source with `agent.read = true` is invisible to `/agent/v1/sources`/`/agent/v1/webspaces` and unreadable via `/agent/v1/items/{id}` until restart, even though the human-facing `/api/*` surface and the UI see it immediately.

**Fix:** Thread `cfgStore` (not a resolved `cfg`) into `agentSourcesHandler`, `agentWebspacesHandler`, `agentItemHandler`, and `agentRenditionHandler`, resolving `cfg := cfgStore.Expanded()` as the first statement inside each returned closure — the identical pattern `agentStreamHandler` and every `/api/*` handler already use:

```go
func agentSourcesHandler(store *index.Store, cfgStore *config.Store, prober HealthProber) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := cfgStore.Expanded()
		ctx := r.Context()
		granted := grantedSources(cfg)
		// ...
	}
}
```
Add a `live_config_test.go`-style regression test asserting a grant revoked via `Store.Save` on the same `*Store`/`Router` stops appearing in `/agent/v1/sources` and starts 404'ing (as `item_not_found`, per T-02-20) on `/agent/v1/items/{id}` on the very next request, with no restart.

---

### CR-02: Canceling/reopening the same source's "Edit connection…"/"Edit match settings…" modal resurfaces discarded, unsaved field values — which can then be silently saved

**File:** `web/src/routes/w/[webspace]/+page.svelte:131-167`, `web/src/lib/components/EditSourceModal.svelte:59-65`

**Issue:** `EditSourceModal`'s form state is seeded exactly once, at mount, directly from props:

```svelte
let connectionValues = $state<SourceConfig>(
	config.sources[instance] ?? { plugin: '', agent: { read: false, handoff: false } }
);
let matchBlock = $state<Record<string, string[]>>(
	config.webspaces[webspace]?.match?.[instance] ?? {}
);
```

The component's own doc comment acknowledges this is deliberate — *"callers key this component ... so a genuinely different instance/mode always mounts a fresh EditSourceModal"* — matching the discipline `MatchFieldsForm.svelte` also documents. That discipline is honored correctly inside `ManageSourcesModal.svelte` (`onclose={() => (editInstance = null)}` at line 366, which drops the `{#if editInstance}` gate entirely and forces a true remount next open), but the **primary** entry point — `SourceChip`'s edit menu, wired through `+page.svelte`'s `handleChipEdit`/`handleEditClose`/`handleEditSaved` — never resets `editInstance`:

```ts
function handleEditClose() {
	editOpen = false;   // editInstance is left set
}

async function handleEditSaved() {
	editOpen = false;   // editInstance is left set here too
	await Promise.all([loadConfig(navGeneration), loadSources(), load(navGeneration)]);
}
```

and the render site only remounts `EditSourceModal` when the `{#key}` value changes:

```svelte
{#if configResponse && editInstance}
	{#key `${editInstance}-${editMode}`}
		<EditSourceModal open={editOpen} mode={editMode} instance={editInstance} ... />
	{/key}
{/if}
```

Because `editInstance`/`editMode` are never cleared, reopening the edit modal for the **same** source in the **same** mode (the common case: open → tweak a field → Cancel → reopen the same source later) produces the identical `{#key}` value, so Svelte does not remount `EditSourceModal` — its `connectionValues`/`matchBlock` `$state` survive untouched from the previous session, including any edits the user typed and then clicked Cancel on. `handleOpenChange`/Cancel only flips `editOpen` to `false` (closing the `Dialog`), it never resets the underlying form state (unlike `AddSourceModal.svelte`'s `resetFlowState()`, which is called on every `onOpenChange(false)`).

**Impact:** A user can type an incorrect `base_url`/token/display name, click Cancel, and later reopen the same source's "Edit connection…" — the incorrect, previously-canceled value is what's shown (not the current config), often indistinguishably from real data since the field looks pre-filled exactly as usual. If they click "Save changes" without noticing (e.g., because they only meant to change one other field), that stale, discarded value is written to `config.toml` via `PUT /api/config`, silently corrupting the real connection config for that source. The same applies to `matchBlock` in `'match'` mode.

**Fix:** Reset `editInstance` (and `editMode`) to `null` in `handleEditClose` (mirroring `ManageSourcesModal.svelte`'s own `onclose={() => (editInstance = null)}`), so every reopen genuinely remounts `EditSourceModal` from current props:

```ts
function handleEditClose() {
	editOpen = false;
	editInstance = null;
}

async function handleEditSaved() {
	editOpen = false;
	editInstance = null;
	await Promise.all([loadConfig(navGeneration), loadSources(), load(navGeneration)]);
}
```
Alternatively/additionally, give `EditSourceModal` its own `$effect(() => { if (open) { connectionValues = ...; matchBlock = ...; } })` reset-on-open, matching `CreateWebspaceModal.svelte`'s and `ManageSourcesModal.svelte`'s own documented "reset local state whenever the modal transitions to open" pattern — this closes the gap even if a future caller makes the same `{#key}`-reset mistake again.

## Warnings

### WR-01: `ManageSourcesModal`'s "Reload config" has no handling for the currently-viewed webspace disappearing from the reloaded file

**File:** `web/src/lib/components/ManageSourcesModal.svelte:174-192`

**Issue:** `confirmDeleteWebspace` explicitly checks whether the deleted webspace was `currentWebspace` and navigates away (`goto('/w/' + remaining[0])` or `goto('/')`) so the user is never left on a route the kernel no longer knows about (lines 149-161). `handleReload`, which can equally cause `currentWebspace` to vanish from the config (a hand-edit that removes or renames the webspace, applied via `POST /api/config/reload`), does no equivalent check:

```ts
async function handleReload() {
	if (reloading) return;
	reloading = true;
	reloadError = null;
	try {
		const res = await reloadConfig();
		localConfig = res.config;
		localHash = res.hash;
		onchanged();
	} catch (err) { ... } finally { reloading = false; }
}
```

**Fix:** After a successful reload, check whether `currentWebspace` is still present in `res.config.webspaces` and navigate away using the same fallback `confirmDeleteWebspace` already implements, for consistency and to avoid leaving the header/stream displaying a webspace the kernel no longer serves.

### WR-02: `SecretField`/`ConnectionForm`'s var-unwrap silently echoes a non-`${VAR}`-shaped stored token verbatim into an editable plaintext field

**File:** `web/src/lib/components/ConnectionForm.svelte:46-51`

**Issue:** `unwrapVar` only strips the `${VAR}`/`$VAR` wrapper when the stored string matches `VAR_PATTERN`; any other shape (e.g., a hand-edited `config.toml` where an operator typed a literal token value instead of a `${VAR}` reference — not structurally prevented by `kernel/config/config.go`'s `Validate`, only a documented convention) is returned unchanged and rendered directly into `SecretField`'s plaintext `<Input>`. `SecretField.svelte`'s own doc comment states its "only content, ever, is an environment variable NAME — never a secret value" (lines 3-8), but this contract is enforced only for the happy-path shape, not defensively. On save, `wrapVar` would then wrap that literal value in `${}`, turning what was a real secret into a broken env-var reference and losing the original credential.

**Fix:** When `unwrapVar` cannot match `VAR_PATTERN`, treat the field as unrecognized/invalid rather than echoing it verbatim — e.g., render a distinct "this field is not a `${VAR}` reference and must be fixed by hand" state instead of populating the plaintext input with what may be a live secret, and avoid silently mangling it into a bogus reference on the next save.

## Info

### IN-01: `agent.go`'s own doc comment is now inaccurate and will mislead future maintainers

**File:** `kernel/httpapi/agent.go:384-390`

**Issue:** The comment above `MountAgentRoutes` claims the four boot-snapshotted agent handlers get "the same deliberately-temporary boot-snapshot treatment Router gives WebspacesHandler/ItemHandler/SourceRefreshHandler" — but per `routes.go`'s own doc comment and the actual code in `webspaces.go`/`item.go`/`sources.go`, none of those three handlers are boot-snapshotted anymore (07-02-PLAN.md Task 2 closed that gap for all of `/api/*`). This comment should be corrected as part of fixing CR-01 above, so it doesn't continue asserting a parity that no longer holds.

**Fix:** Update or remove this comment once CR-01 is fixed; it should describe the live-per-request read the fix introduces, not the historical (now-incorrect) "boot snapshot, matching those three handlers" framing.

---

_Reviewed: 2026-08-08T14:56:06Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
