<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import {
		Tooltip,
		TooltipContent,
		TooltipProvider,
		TooltipTrigger
	} from '$lib/components/ui/tooltip/index.js';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';
	import { cn } from '$lib/utils.js';
	import { healthTone, formatRelativeTime, type HealthTone } from '$lib/format';
	import type { SourceStatus } from '$lib/api';

	// A compact per-source status chip: dot + display name (D-08),
	// hover/focus tooltip carrying the relative last-sync time and the
	// last error, and an icon-only refresh control. Composed entirely from
	// installed shadcn primitives (Button, Tooltip) plus a plain styled
	// dot — no new registry block.
	let { source, onrefresh }: { source: SourceStatus; onrefresh: (name: string) => void } = $props();

	let tone = $derived(healthTone(source));

	const DOT_TONE_CLASS: Record<HealthTone, string> = {
		success: 'bg-success',
		warning: 'bg-warning',
		destructive: 'bg-destructive',
		unknown: 'bg-muted-foreground'
	};

	// Copywriting Contract (02-UI-SPEC.md) rows for reachable / stale /
	// unreachable. The "never synced" (unknown) case has no row of its
	// own in the contract — this is the minimal, sensible copy for it
	// (Rule 2: a hover-only diagnostic surface can't render empty text).
	let tooltipText = $derived.by(() => {
		const relative = formatRelativeTime(source.last_sync_unix);
		switch (tone) {
			case 'success':
				return `${source.display_name} — synced ${relative} ago`;
			case 'warning':
				return `${source.display_name} — last error ${relative} ago: ${source.last_error}`;
			case 'destructive':
				return `${source.display_name} — unreachable since ${relative}`;
			default:
				return `${source.display_name} — not yet synced`;
		}
	});
</script>

<div class="flex items-center gap-1.5 rounded-full border border-border bg-card py-1 pr-1 pl-2.5">
	<TooltipProvider>
		<Tooltip>
			<TooltipTrigger>
				{#snippet child({ props })}
					<span {...props} class="flex max-w-48 items-center gap-1.5">
						<span
							class={cn('size-2 shrink-0 rounded-full', DOT_TONE_CLASS[tone])}
							aria-hidden="true"
						></span>
						<span
							class="truncate text-[14px] leading-[1.4] text-foreground"
							title={source.display_name}>{source.display_name}</span
						>
						{#if source.syncing}
							<span class="text-[14px] leading-[1.4] text-muted-foreground">Syncing…</span>
						{/if}
					</span>
				{/snippet}
			</TooltipTrigger>
			<TooltipContent>{tooltipText}</TooltipContent>
		</Tooltip>
	</TooltipProvider>

	<Button
		variant="ghost"
		size="icon"
		class="size-11"
		disabled={source.syncing}
		aria-label={`Refresh ${source.display_name}`}
		onclick={() => onrefresh(source.name)}
	>
		<RefreshCw class={cn('size-4', source.syncing && 'animate-spin')} />
	</Button>
</div>
