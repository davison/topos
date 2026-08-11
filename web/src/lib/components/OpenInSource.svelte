<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import ArrowUpRight from '@lucide/svelte/icons/arrow-up-right';
	import AppWindow from '@lucide/svelte/icons/app-window';
	import { fidelityAffordance, formatFidelity } from '$lib/format';
	import type { Link } from '$lib/api';

	// displayName parameterizes the button label (RESEARCH.md Pitfall 3,
	// generalized to source instance identity by D-08/D-09): this
	// component is shared across every source instance, so the label must
	// never hardcode one source's name or fall back to the plugin kind
	// (source_type) — the caller (DetailPane.svelte) resolves the correct
	// instance display name and passes it straight through.
	//
	// iconOnly (09.1-01-PLAN.md Task 2, D-04): the mobile takeover's slim
	// bar has room for an icon only, not the label/badge row below. Copy
	// is never dropped from the accessible name — only from view — so
	// aria-label carries affordance.label whenever the visible span is
	// omitted.
	let {
		link,
		displayName,
		iconOnly = false
	}: { link: Link; displayName: string; iconOnly?: boolean } = $props();

	// The two-class icon/verb/title split (UI-08) — see fidelityAffordance's
	// (format.ts) own doc comment for why this stays a two-class split
	// alongside the Badge below, which keeps the raw three-value enum for
	// the power-user detail.
	let affordance = $derived(fidelityAffordance(link.fidelity, displayName));
</script>

{#if iconOnly}
	<Button
		href={link.url}
		target="_blank"
		rel="noopener noreferrer"
		variant="ghost"
		class="size-11 rounded-md"
		title={affordance.title}
		aria-label={affordance.label}
	>
		{#if affordance.windowOnly}
			<AppWindow class="size-4 shrink-0" />
		{:else}
			<ArrowUpRight class="size-4 shrink-0" />
		{/if}
	</Button>
{:else}
	<div class="flex items-center gap-2">
		<Button
			href={link.url}
			target="_blank"
			rel="noopener noreferrer"
			class="min-h-11 max-w-64"
			title={affordance.title}
		>
			{#if affordance.windowOnly}
				<AppWindow class="size-4 shrink-0" />
			{:else}
				<ArrowUpRight class="size-4 shrink-0" />
			{/if}
			<span class="truncate">{affordance.label}</span>
		</Button>
		<Badge variant="secondary">{formatFidelity(link.fidelity)}</Badge>
	</div>
{/if}
