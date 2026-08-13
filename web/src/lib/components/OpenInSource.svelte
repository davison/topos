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

	// isLocalExecLink (12-UI-SPEC.md F2): a same-origin, relative path
	// beginning with "/api/" is the shape the kernel's file://-scheme
	// deep-link rewrite (kernel/httpapi/stream.go's resolveStreamLinkURL)
	// produces — keyed on URL SHAPE alone, never a plugin type, so a future
	// third-party local-path plugin is covered for free with no additional
	// branching here. Every other link shape (http(s)://, mailto:) falls
	// through to the plain anchor-navigation branch, unchanged.
	let isLocalExecLink = $derived(link.url.startsWith('/api/'));

	// The local-exec failure swap-then-revert mechanic (F2): a rejected
	// fetch or a non-ok response swaps the visible copy to the failure
	// state for FAILURE_REVERT_MS, then reverts. openFailure holds the
	// detail text to show (the kernel's own error message, or the fixed
	// fallback) while non-null, and null at rest.
	const FAILURE_REVERT_MS = 2500;
	const OPEN_FAILURE_LABEL = "Couldn't open";
	const OPEN_FAILURE_FALLBACK_DETAIL = "Couldn't open — file may have moved or been removed.";

	let openFailure = $state<string | null>(null);
	let revertTimer: ReturnType<typeof setTimeout> | undefined;

	// openLocalExecLink issues exactly one POST to link.url. A successful
	// open changes nothing visible — the desktop's own file handler window
	// appearing is the confirmation. A rejected fetch or a non-ok response
	// both trigger the failure swap, preferring the kernel's own error
	// detail (the shared apiError envelope's error.message) and falling
	// back to the fixed copy when the response carries none.
	async function openLocalExecLink() {
		try {
			const res = await fetch(link.url, { method: 'POST' });
			if (res.ok) return;
			let detail = OPEN_FAILURE_FALLBACK_DETAIL;
			try {
				const body = (await res.json()) as { error?: { message?: string } };
				if (body?.error?.message) detail = body.error.message;
			} catch {
				// Non-JSON or empty body — keep the fixed fallback.
			}
			showOpenFailure(detail);
		} catch {
			showOpenFailure(OPEN_FAILURE_FALLBACK_DETAIL);
		}
	}

	function showOpenFailure(detail: string) {
		openFailure = detail;
		clearTimeout(revertTimer);
		revertTimer = setTimeout(() => {
			openFailure = null;
		}, FAILURE_REVERT_MS);
	}
</script>

{#if iconOnly}
	<Button
		href={isLocalExecLink ? undefined : link.url}
		onclick={isLocalExecLink ? openLocalExecLink : undefined}
		target={isLocalExecLink ? undefined : '_blank'}
		rel={isLocalExecLink ? undefined : 'noopener noreferrer'}
		variant="ghost"
		class="size-11 rounded-md"
		title={openFailure ?? affordance.title}
		aria-label={openFailure ?? affordance.label}
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
			href={isLocalExecLink ? undefined : link.url}
			onclick={isLocalExecLink ? openLocalExecLink : undefined}
			target={isLocalExecLink ? undefined : '_blank'}
			rel={isLocalExecLink ? undefined : 'noopener noreferrer'}
			class="min-h-11 max-w-64"
			title={openFailure ?? affordance.title}
		>
			{#if affordance.windowOnly}
				<AppWindow class="size-4 shrink-0" />
			{:else}
				<ArrowUpRight class="size-4 shrink-0" />
			{/if}
			<span class="truncate {openFailure ? 'text-destructive' : ''}"
				>{openFailure ? OPEN_FAILURE_LABEL : affordance.label}</span
			>
		</Button>
		<Badge variant="secondary">{formatFidelity(link.fidelity)}</Badge>
	</div>
{/if}
