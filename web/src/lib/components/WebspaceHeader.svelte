<script lang="ts">
	import SourceHealthChip from './SourceHealthChip.svelte';
	import SourceFilterChips from './SourceFilterChips.svelte';
	import { Button } from '$lib/components/ui/button/index.js';
	import { shouldShowSourceRows } from '$lib/format';
	import type { SourceStatus } from '$lib/api';

	// The webspace title renders synchronously from the route param, same
	// as Phase 1. The health-chip and source-filter rows below it DO have
	// their own async load/error state (GET /api/sources, owned by the
	// caller) — unlike Phase 1's title-only header, a non-critical sources
	// failure must never block or blank the primary stream view, so both
	// rows render only once sourcesState is 'ready' and at least one
	// source is configured (shouldShowSourceRows, 02-UI-SPEC.md E1/E5).
	let {
		webspace,
		sources,
		sourcesState,
		selectedSource,
		onfilter,
		onrefresh,
		onrefreshall
	}: {
		webspace: string;
		sources: SourceStatus[];
		sourcesState: 'loading' | 'error' | 'ready';
		selectedSource: string | null;
		onfilter: (sourceType: string | null) => void;
		onrefresh: (name: string) => void;
		onrefreshall: () => void;
	} = $props();

	let showSourceRows = $derived(shouldShowSourceRows(sourcesState, sources));
</script>

<header class="shrink-0 border-b border-border bg-card px-6 py-6">
	<!--
	  Display role (28px/600/1.2). Webspace names are user-defined
	  config.toml keys of arbitrary length, so this truncates to one line
	  with an ellipsis; the title attribute keeps the full name reachable
	  on hover regardless of how long it is.
	-->
	<h1 class="truncate text-[28px] leading-[1.2] font-semibold text-foreground" title={webspace}>
		{webspace}
	</h1>

	{#if showSourceRows}
		<div class="mt-4 flex flex-wrap items-center gap-2">
			{#each sources as source (source.name)}
				<SourceHealthChip {source} {onrefresh} />
			{/each}
			<Button variant="outline" size="sm" onclick={onrefreshall}>Refresh all</Button>
		</div>
		<div class="mt-3">
			<SourceFilterChips {sources} {selectedSource} {onfilter} />
		</div>
	{/if}
</header>
