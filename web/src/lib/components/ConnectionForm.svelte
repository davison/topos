<script lang="ts">
	// Step 1's connection-field form, driven entirely by
	// plugin-fields.ts's static connectionFieldsFor table (never a
	// per-plugin-type template) — reused unforked by the two-step
	// "New {plugin type}…" flow AND the chip menu's "Edit connection…"
	// modal (07-04-PLAN.md Tasks 2/3). Phase 11 (PLUG-09, D-12/D-13/D-15,
	// 11-UI-SPEC.md E6) extends this same, still-unforked component with
	// the provider-specific extras section, appended after the existing
	// "Advanced options" disclosure.
	import { untrack } from 'svelte';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Checkbox } from '$lib/components/ui/checkbox/index.js';
	import SecretField from './SecretField.svelte';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import Plus from '@lucide/svelte/icons/plus';
	import {
		connectionFieldsFor,
		extrasToRows,
		rowsToExtras,
		type ConnectionField,
		type ExtrasRow
	} from '$lib/plugin-fields';
	import type { SourceConfig, ExtrasFieldDecl } from '$lib/api';

	let {
		pluginBinary,
		values,
		envVars,
		extrasFields = [],
		extrasRows = $bindable([]),
		onchange
	}: {
		pluginBinary: string;
		values: SourceConfig;
		envVars: Record<string, boolean>;
		// extrasFields (D-15) is the plugin's Describe-declared expected
		// extras keys, learned by the caller via a trial describe call —
		// [] when not yet known or the plugin declares none, which still
		// leaves the free-form editor below fully usable.
		extrasFields?: ExtrasFieldDecl[];
		// extrasRows is the free-form (undeclared-key) editor's own row
		// list — bindable so the caller (AddSourceModal/EditSourceModal)
		// can run extrasKeyError against the EXACT rows this component
		// renders, at its own submit-time validation point, without a
		// second, possibly-divergent copy of this list.
		extrasRows?: ExtrasRow[];
		onchange: (values: SourceConfig) => void;
	} = $props();

	let fields = $derived(connectionFieldsFor(pluginBinary));
	let primaryFields = $derived(fields.filter((f) => !f.advanced));
	let advancedFields = $derived(fields.filter((f) => f.advanced));
	let advancedOpen = $state(false);

	function fieldValue(field: ConnectionField): string {
		const raw = values[field.key];
		return typeof raw === 'string' ? raw : '';
	}

	// boolFieldValue reads a checkbox-kind field's current value, coercing
	// an unset or non-boolean stored value to false (unchecked) — never
	// throws, never renders an indeterminate state. Mirrors
	// ConnectionField.defaultValue's own doc comment: a checkbox's initial
	// state is never sourced from defaultValue, only from the value
	// actually stored (or its safe false fallback).
	function boolFieldValue(field: ConnectionField): boolean {
		const raw = values[field.key];
		return typeof raw === 'boolean' ? raw : false;
	}

	function setField(key: ConnectionField['key'], value: string | boolean) {
		onchange({ ...values, [key]: value });
	}

	// A stored ${VAR} (or bare $VAR) reference's bare variable name —
	// SecretField only ever sees/edits the bare name; wrapVar below
	// restores the `${}` form on write, matching D-05's "never expanded,
	// never unwrapped in the file" requirement (this is a DISPLAY-layer
	// unwrap only, never touching the persisted string's own shape).
	const VAR_PATTERN = /^\$\{([A-Za-z_][A-Za-z0-9_]*)\}$|^\$([A-Za-z_][A-Za-z0-9_]*)$/;

	function unwrapVar(raw: string): string {
		if (!raw) return '';
		const match = VAR_PATTERN.exec(raw);
		if (!match) return raw;
		return match[1] ?? match[2] ?? '';
	}

	function wrapVar(name: string): string {
		const trimmed = name.trim();
		return trimmed === '' ? '' : `\${${trimmed}}`;
	}

	// --- Extras (E6, D-12/D-13/D-14/D-15) ---
	//
	// declaredExtrasValues holds each declared field's OWN bound value,
	// separate from extrasRows (the free-form, undeclared-key rows) —
	// never merged into one list, since a declared field renders as a
	// fixed labeled input while a free-form row can be added/removed.
	// Neither is EVER seeded from a declaration's `placeholder` (D-14): an
	// instance's own already-SAVED extras value is real data and pre-fills
	// here exactly like an existing connection field would; a
	// plugin-suggested default never does — it stays display-only, bound
	// only to the Input's placeholder attribute below.
	let declaredExtrasValues = $state<Record<string, string>>(
		Object.fromEntries(extrasFields.map((field) => [field.key, values.extras?.[field.key] ?? '']))
	);

	// Re-syncs both extras pieces whenever a NEW declarations array arrives
	// (extrasFields' identity changes — a fresh best-effort or real
	// describe response) — re-derived from values.extras, which already
	// carries every row/declared edit made so far via commitExtras below
	// (each edit calls onchange immediately), so a value typed before this
	// fires is never lost. Tracks extrasFields alone; every other read is
	// untracked so a keystroke inside THIS form (which reassigns `values`
	// via onchange) can never re-trigger this effect and wipe an
	// in-progress row.
	$effect(() => {
		const fields = extrasFields;
		untrack(() => {
			declaredExtrasValues = Object.fromEntries(
				fields.map((field) => [field.key, values.extras?.[field.key] ?? ''])
			);
			extrasRows = extrasToRows(values.extras, fields);
		});
	});

	function commitExtras() {
		const composed = rowsToExtras(declaredExtrasValues, extrasRows);
		onchange({ ...values, extras: Object.keys(composed).length > 0 ? composed : undefined });
	}

	function setDeclaredExtra(key: string, value: string) {
		declaredExtrasValues = { ...declaredExtrasValues, [key]: value };
		commitExtras();
	}

	function setExtraRowKey(index: number, key: string) {
		extrasRows = extrasRows.map((row, i) => (i === index ? { ...row, key } : row));
		commitExtras();
	}

	function setExtraRowValue(index: number, value: string) {
		extrasRows = extrasRows.map((row, i) => (i === index ? { ...row, value } : row));
		commitExtras();
	}

	function addExtraRow() {
		extrasRows = [...extrasRows, { key: '', value: '' }];
	}

	function removeExtraRow(index: number) {
		extrasRows = extrasRows.filter((_, i) => i !== index);
		commitExtras();
	}
</script>

<div class="flex flex-col gap-4">
	{#each primaryFields as field (field.key)}
		{#if field.secret}
			<!-- First line of defence: the DOM required attribute (forwarded
			     into SecretField below, and directly on the plain Input in the
			     else branch). The second, load-bearing line is the submit-time
			     missingRequiredFields() check each of this component's three
			     consumers (AddSourceModal's Connect step and Save anyway,
			     EditSourceModal's Edit connection…) runs itself — this
			     component cannot own that check because each consumer issues
			     its own request from its own submit handler, so a guard living
			     only here could never stop any of them. The asterisk alone
			     (still rendered below) was purely decorative. -->
			<SecretField
				label={field.label}
				required={field.required}
				value={unwrapVar(fieldValue(field))}
				placeholder={field.placeholder}
				{envVars}
				oninput={(name) => setField(field.key, wrapVar(name))}
			/>
		{:else if field.kind === 'checkbox'}
			<label
				for={`conn-${field.key}`}
				class="flex min-h-11 items-center gap-2 text-[14px] leading-[1.4] text-foreground"
			>
				<Checkbox
					id={`conn-${field.key}`}
					checked={boolFieldValue(field)}
					onCheckedChange={(v) => setField(field.key, v)}
				/>
				{field.label}
			</label>
			{#if field.helperText}
				<p class="text-[14px] leading-[1.4] text-muted-foreground">{field.helperText}</p>
			{/if}
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

	{#if advancedFields.length > 0}
		<div>
			<Button
				type="button"
				variant="ghost"
				size="sm"
				onclick={() => (advancedOpen = !advancedOpen)}
			>
				Advanced options
			</Button>
			{#if advancedOpen}
				<div class="mt-2 flex flex-col gap-4">
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
				</div>
			{/if}
		</div>
	{/if}

	<!--
	  E6 — Extras Form: Declared Fields + Free-Form Editor (D-12/D-13/D-14/
	  D-15). "Additional fields" and the free-form row editor render
	  unconditionally, regardless of declared-field count — the free-form
	  editor is how a user adds provider-specific values even for a plugin
	  type that declares none; "Provider-specific settings" itself is
	  gated on there being at least one declaration.
	-->
	<div class="flex flex-col gap-2">
		{#if extrasFields.length > 0}
			<p
				class="text-[14px] leading-[1.4] font-medium tracking-wide text-muted-foreground uppercase"
			>
				Provider-specific settings
			</p>
			{#each extrasFields as field (field.key)}
				{#if field.secret}
					<SecretField
						label={field.label}
						required={field.required}
						value={unwrapVar(declaredExtrasValues[field.key] ?? '')}
						placeholder={field.placeholder}
						{envVars}
						oninput={(name) => setDeclaredExtra(field.key, wrapVar(name))}
					/>
				{:else}
					<div class="flex flex-col gap-1">
						<label for={`extra-${field.key}`} class="text-[14px] leading-[1.4] text-foreground">
							{field.label}{field.required ? ' *' : ''}
						</label>
						<Input
							id={`extra-${field.key}`}
							value={declaredExtrasValues[field.key] ?? ''}
							placeholder={field.placeholder}
							required={field.required}
							oninput={(e) => setDeclaredExtra(field.key, e.currentTarget.value)}
						/>
					</div>
				{/if}
			{/each}
		{/if}

		<p class="text-[14px] leading-[1.4] font-medium tracking-wide text-muted-foreground uppercase">
			Additional fields
		</p>
		{#each extrasRows as row, index (index)}
			<div class="flex items-center gap-2">
				<Input
					class="w-1/3"
					placeholder="Key"
					value={row.key}
					oninput={(e) => setExtraRowKey(index, e.currentTarget.value)}
				/>
				<Input
					class="flex-1"
					placeholder="Value"
					value={row.value}
					oninput={(e) => setExtraRowValue(index, e.currentTarget.value)}
				/>
				<Button
					type="button"
					variant="ghost"
					size="icon"
					class="size-8"
					aria-label="Remove field"
					onclick={() => removeExtraRow(index)}
				>
					<Trash2 class="size-3.5" aria-hidden="true" />
				</Button>
			</div>
		{/each}
		<Button type="button" variant="ghost" size="sm" onclick={addExtraRow}>
			<Plus class="size-4" aria-hidden="true" />
			Add field
		</Button>
	</div>
</div>
