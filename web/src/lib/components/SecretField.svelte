<script lang="ts">
	// The env-var-name-only secret field contract (D-15, T-07-19): this
	// component's only content, ever, is an environment variable NAME —
	// never a secret value. A plain text input (never `type="password"`,
	// never autofill-eligible) plus a live set/unset badge sourced from
	// the kernel's own already-fetched env_vars presence map
	// (GET/PUT /api/config's `env_vars: Record<string, boolean>` — a
	// boolean per name, never a value, D-05).
	//
	// Deviation from 07-UI-SPEC.md's literal "debounced live lookup"
	// wording: the kernel's env_vars map (kernel/httpapi/config.go's
	// envVarsIn) only ever reports on variable names already referenced
	// somewhere in the PERSISTED config document — there is no endpoint
	// that checks an arbitrary, not-yet-saved name, and this plan's own
	// files_modified scope carries no kernel/ files to add one. This
	// component instead reads directly from the `envVars` prop the caller
	// already holds (no additional network round trip needed or
	// possible) — for an existing instance's stored `${VAR}` reference
	// this is exactly accurate; for a brand-new, never-yet-saved variable
	// name it conservatively renders "Not set" until the instance is
	// actually saved (D-15's "informational, never a submit blocker"
	// framing already treats this badge as advisory, not authoritative).
	import { Input } from '$lib/components/ui/input/index.js';
	import CircleCheck from '@lucide/svelte/icons/circle-check';
	import CircleAlert from '@lucide/svelte/icons/circle-alert';

	let {
		label,
		value,
		placeholder,
		required = false,
		envVars,
		oninput
	}: {
		label: string;
		// value is the environment variable NAME (bare, no `${}` wrapper —
		// the caller extracts/re-wraps against the stored `${VAR}` form).
		value: string;
		placeholder?: string;
		required?: boolean;
		envVars: Record<string, boolean>;
		oninput: (name: string) => void;
	} = $props();

	let trimmed = $derived(value.trim());
	let isSet = $derived(trimmed !== '' && envVars[trimmed] === true);
	let isUnset = $derived(trimmed !== '' && envVars[trimmed] !== true);
</script>

<div class="flex flex-col gap-1">
	<label for={`secret-${label}`} class="text-[14px] leading-[1.4] text-foreground">
		{label}{required ? ' *' : ''}
	</label>
	<Input
		id={`secret-${label}`}
		type="text"
		autocomplete="off"
		{placeholder}
		{value}
		oninput={(e) => oninput(e.currentTarget.value)}
	/>
	{#if trimmed !== ''}
		<div class="flex items-center gap-1.5">
			{#if isSet}
				<CircleCheck class="size-3.5 shrink-0 text-success" aria-hidden="true" />
				<span class="text-[14px] leading-[1.4] text-success">Set</span>
			{:else if isUnset}
				<CircleAlert class="size-3.5 shrink-0 text-warning" aria-hidden="true" />
				<span class="text-[14px] leading-[1.4] text-warning">
					Not set — add it to .env and restart before this source can connect.
				</span>
			{/if}
		</div>
	{/if}
</div>
