---
phase: 07-webspace-builder-ui
reviewed: 2026-08-09T00:00:00Z
depth: standard
files_reviewed: 26
files_reviewed_list:
  - config.example.toml
  - kernel/config/config.go
  - kernel/config/config_test.go
  - kernel/config/store_test.go
  - kernel/config/types.go
  - kernel/correlate/correlate.go
  - kernel/correlate/correlate_test.go
  - kernel/httpapi/config_test.go
  - kernel/pluginhost/host.go
  - kernel/pluginhost/matchconfig.go
  - kernel/pluginhost/stderr_test.go
  - web/src/lib/components/AddSourceModal.svelte
  - web/src/lib/components/add-source.test.ts
  - web/src/lib/components/ConnectionForm.svelte
  - web/src/lib/components/EditSourceModal.svelte
  - web/src/lib/components/SecretField.svelte
  - web/src/lib/components/WebspaceHeader.svelte
  - web/src/lib/components/webspace-participation.test.ts
  - web/src/lib/config-edit.test.ts
  - web/src/lib/config-edit.ts
  - web/src/lib/participation.test.ts
  - web/src/lib/participation.ts
  - web/src/lib/plugin-fields.test.ts
  - web/src/lib/plugin-fields.ts
  - web/src/routes/+page.svelte
  - web/src/routes/root-empty-state.test.ts
findings:
  critical: 0
  warning: 4
  info: 1
  total: 5
status: issues_found
---

# Phase 07: Code Review Report (gap-closure re-review)

**Reviewed:** 2026-08-09
**Depth:** standard
**Files Reviewed:** 26
**Status:** issues_found

## Summary

This re-review scopes to the 26 files that make up the diff of four gap-closure
plans (07-11 empty-webspace shell, 07-12 config null-serialization + root-route
catch isolation, 07-13 required-field guards + plugin stderr capture, 07-14
remove-source participation fix) layered on top of the already-reviewed 07
baseline. It supersedes the prior 07-REVIEW.md dated 2026-08-08 (96-file
scope), which is no longer applicable to the current tree.

The kernel-side logic (`config.Validate`, `Webspace.IsEmptyShell`,
`correlate.matchFieldsFor`'s D-20 rule, `pluginhost.ValidateMatchConfig`, the
plugin-subprocess stderr tail) is internally consistent and its test coverage
(config_test.go, store_test.go, correlate_test.go, matchconfig.go's callers,
stderr_test.go) exercises the documented edge cases correctly — I traced the
D-20 empty-shell rule end to end (config load → correlate sync → UI) and found
no divergence between the three independent implementations (kernel Go,
`participation.ts`, `config-edit.ts`) of the same participation semantics.
Nothing in the four Go files rises to Critical or Warning.

The defects below are all on the frontend, and share one theme: this exact
gap-closure batch was built specifically to eliminate an `Object.keys(null)`/
null-collection crash class (07-12, G-07-4), and several pieces of *new* code
added in this same batch (07-13, 07-14) reach for a raw property access
instead of the null-tolerant reader pattern the batch itself establishes and
documents as load-bearing elsewhere in the very same files. None of these are
reachable against the kernel binary this phase ships (which now guarantees
non-null collections via `applyDefaults`), which is why they are WARNING
rather than BLOCKER — but the codebase's own doc comments explicitly treat
"SPA newer than kernel" as a real, anticipated scenario worth defending
against, and these spots don't.

## Warnings

### WR-01: ConnectionForm.svelte's advanced-fields loop drops the `required` attribute and secret-field branching present in the primary-fields loop

**File:** `web/src/lib/components/ConnectionForm.svelte:108-120`
**Issue:** The primary-fields loop (lines 60-94) branches on `field.secret` (rendering `SecretField` with `required={field.required}` for a secret field, or a plain `Input` with `required={field.required}` and an asterisk in the label otherwise). The advanced-fields loop does neither: it unconditionally renders a plain `Input` with no `required` prop at all, and its label never appends `' *'` for a required field:
```svelte
{#each advancedFields as field (field.key)}
	<div class="flex flex-col gap-1">
		<label for={`conn-${field.key}`} class="text-[14px] leading-[1.4] text-foreground">
			{field.label}
		</label>
		<Input
			id={`conn-${field.key}`}
			value={fieldValue(field)}
			placeholder={field.placeholder}
			oninput={(e) => setField(field.key, e.currentTarget.value)}
		/>
	</div>
{/each}
```
Today this is silent because `CONNECTION_FIELDS` (plugin-fields.ts) happens to mark every `advanced: true` row (`SYNC_INTERVAL_FIELD`) as `required: false` and never `secret: true` — but that is a property of the current table, not of this component. The very phase that added this file's DOM-level `required` guard was 07-13, closing G-07-5 ("a blank required field must never reach a plugin subprocess" — see the component's own doc comment: "The second, load-bearing line is the submit-time missingRequiredFields() check... The asterisk alone... was purely decorative"). A future plugin type with a required, advanced field (fully permitted by the `ConnectionField` type and the table's own derivation rule) would silently lose both the visual cue and the DOM `required` attribute here, while still being subject to the JS-level `missingRequiredFields` submit guard alone — reintroducing exactly the asymmetry 07-13 was written to close, just for the advanced-fields branch specifically.

This gap is also invisible to the regression test written to guard it: `add-source.test.ts`'s `"the non-secret Input carries a required attribute bound to the field descriptor"` assertion does `extractBetween(connectionFormStripped, '<Input', '/>')`, which returns only the *first* `<Input` occurrence in the file — the primary-fields one. It structurally cannot see the second (advanced-fields) `<Input` block, so a regression here would pass CI silently.
**Fix:**
```svelte
{#each advancedFields as field (field.key)}
	{#if field.secret}
		<SecretField
			label={field.label}
			required={field.required}
			value={unwrapVar(fieldValue(field))}
			placeholder={field.placeholder}
			{envVars}
			oninput={(name) => setField(field.key, wrapVar(name))}
		/>
	{:else}
		<div class="flex flex-col gap-1">
			<label for={`conn-${field.key}`} class="text-[14px] leading-[1.4] text-foreground">
				{field.label}{field.required ? ' *' : ''}
			</label>
			<Input
				id={`conn-${field.key}`}
				value={fieldValue(field)}
				placeholder={field.placeholder}
				required={field.required}
				oninput={(e) => setField(field.key, e.currentTarget.value)}
			/>
		</div>
	{/if}
{/each}
```
And extend `add-source.test.ts` to scope a second assertion at the advanced-fields `<Input>` (or SecretField) specifically, rather than relying on `extractBetween`'s first-match behavior.

### WR-02: config-edit.ts's `removeSourceInstance` bypasses the null-tolerant readers used everywhere else in this file

**File:** `web/src/lib/config-edit.ts:270-285`
**Issue:** Every other webspace-collection read added or touched in this gap-closure batch goes through `webspaceKeywords`/`webspaceSources`/`webspaceMatch` (`participation.ts`) specifically because "GET /api/config genuinely serializes `null` for a hand-written webspace that omits a key... Reading such a field directly is how a TypeError gets raised inside a component and misreported as something else entirely (the mechanism 07-UAT.md G-07-4 documents)" (`participation.ts:17-27`). `removeSourceInstance` does not follow this discipline:
```ts
for (const name of Object.keys(next.webspaces)) {
	const ws = next.webspaces[name];
	if (Object.prototype.hasOwnProperty.call(ws.match, instanceId)) {   // throws if ws.match is null
		const match = { ...ws.match };
		delete match[instanceId];
		ws.match = match;
	}
	if (ws.sources.includes(instanceId)) {                              // throws if ws.sources is null
		ws.sources = ws.sources.filter((s) => s !== instanceId);
	}
}
```
`Object.prototype.hasOwnProperty.call(null, x)` and `null.includes(x)` both throw `TypeError`. This is the Manage Sources modal's instance-delete write path — a webspace whose `match`/`sources` arrived as `null` (a hand-edited `config.toml` reloaded through an older kernel, or any regression in the 07-12 kernel-side normalization this exact defence was built around) would crash the delete flow instead of degrading gracefully.
**Fix:**
```ts
import { webspaceMatch, webspaceSources } from './participation';
...
for (const name of Object.keys(next.webspaces)) {
	const ws = next.webspaces[name];
	const match = webspaceMatch(ws);
	if (Object.prototype.hasOwnProperty.call(match, instanceId)) {
		const nextMatch = { ...match };
		delete nextMatch[instanceId];
		ws.match = nextMatch;
	}
	const sources = webspaceSources(ws);
	if (sources.includes(instanceId)) {
		ws.sources = sources.filter((s) => s !== instanceId);
	}
}
```

### WR-03: `participatingInstances` and `addSourceToWebspace`'s seeding branch read `cfg.sources` unguarded, while every other collection in the same files is null-tolerant

**File:** `web/src/lib/participation.ts:103` (also `web/src/lib/config-edit.ts:159`)
**Issue:** `participatingInstances` — introduced *by this exact gap-closure plan* (07-14, per this file's own header comment: "07-14 extends this module with a participating-instances helper for the chip row") — iterates `Object.keys(cfg.sources)` directly:
```ts
export function participatingInstances(cfg: KernelConfig, webspace: string): Set<string> {
	const ws = cfg.webspaces[webspace];
	const allowlist = webspaceSources(ws);   // null-tolerant
	const keywords = webspaceKeywords(ws);   // null-tolerant
	const match = webspaceMatch(ws);         // null-tolerant
	const result = new Set<string>();
	for (const instance of Object.keys(cfg.sources)) {   // NOT null-tolerant
```
`config-edit.ts`'s `addSourceToWebspace` has the identical pattern at its seeding branch: `sources = Object.keys(cfg.sources);` (line 159). If `cfg.sources` itself ever serializes as `null` — the exact top-level field `07-12-PLAN.md`'s own fix targeted (`TestConfigHandler_ZeroWebspacesSerializesEmptyObjectsNotNull` pins `"sources":null` as the regression this phase closes) — both the add-source picker (`AddSourceModal.svelte`'s `participatingInstances(config, webspace)`) and the chip row (`WebspaceHeader.svelte`'s `participatesIn`) throw `Object.keys(null)` inside a `$derived`, which is the same failure mode that turned a healthy 200 OK kernel into "the topos service didn't respond" in the original G-07-4 report — reintroduced, inside the very module written to prevent it, by new code from the same plan.
**Fix:** Add a null-tolerant `cfg.sources` read (mirroring `webspaceSources`) and use it in both spots:
```ts
// participation.ts
export function configSources(cfg: KernelConfig): Record<string, SourceConfig> {
	return cfg.sources ?? {};
}
...
for (const instance of Object.keys(configSources(cfg))) { ... }
```
```ts
// config-edit.ts
sources = Object.keys(configSources(cfg));
```

### WR-04: `Host.Reconcile` relaunches a plugin subprocess on an Agent-grant-only config change

**File:** `kernel/pluginhost/host.go:183`
**Issue:** `Reconcile` decides whether an instance needs relaunching via `reflect.DeepEqual(p.src, sources[name])` — a full-struct comparison of `config.Source`, which includes the `Agent AgentGrant` field. But `launch()`'s `sourceConfig` payload sent to the subprocess (lines 313-324) never includes `Agent` — the subprocess has no way to observe agent grants at all. Toggling only `read`/`handoff` for a source (with no connection-field change) therefore still fails the `DeepEqual` check and causes `Reconcile` to kill and relaunch that instance's subprocess unnecessarily, producing a brief reachability/health-probe gap for a change that has nothing to do with connectivity.
**Fix:** Compare only the fields that actually affect the subprocess (or a struct with `Agent` zeroed out) when deciding whether to relaunch:
```go
connectionEqual := func(a, b config.Source) bool {
	a.Agent, b.Agent = config.AgentGrant{}, config.AgentGrant{}
	return reflect.DeepEqual(a, b)
}
...
if ok && connectionEqual(p.src, sources[name]) {
	kept[name] = p
	continue
}
```

## Info

### IN-01: `missingRequiredFields` treats an explicit `null` value as present, inconsistent with its own `undefined` handling

**File:** `web/src/lib/plugin-fields.ts:201-212`
**Issue:** The function explicitly special-cases `undefined` ("A value of `undefined` counts as missing") but a `raw` value of `null` falls through the `typeof raw !== 'string'` branch and returns `false` (treated as present/satisfied), even though the doc comment's stated intent ("missing, empty or whitespace-only") reads as though `null` should count as missing too. No current caller ever assigns `null` into `SourceConfig`, so this is not reachable today, but it's an inconsistency in the same defensive function this gap-closure batch (07-13) added specifically to close a required-field gap.
**Fix:** `if (raw === undefined || raw === null) return true;`

---

_Reviewed: 2026-08-09_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
