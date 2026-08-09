<script lang="ts">
	// Step 1's connection-field form, driven entirely by
	// plugin-fields.ts's static connectionFieldsFor table (never a
	// per-plugin-type template) — reused unforked by the two-step
	// "New {plugin type}…" flow AND the chip menu's "Edit connection…"
	// modal (07-04-PLAN.md Tasks 2/3).
	import { Input } from '$lib/components/ui/input/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import SecretField from './SecretField.svelte';
	import { connectionFieldsFor, type ConnectionField } from '$lib/plugin-fields';
	import type { SourceConfig } from '$lib/api';

	let {
		pluginBinary,
		values,
		envVars,
		onchange
	}: {
		pluginBinary: string;
		values: SourceConfig;
		envVars: Record<string, boolean>;
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

	function setField(key: ConnectionField['key'], value: string) {
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
</div>
