<script lang="ts">
	import SourceChip from './SourceChip.svelte';
	import SearchBox from './SearchBox.svelte';
	import { Button } from '$lib/components/ui/button/index.js';
	import { shouldShowSourceRows } from '$lib/format';
	import type { SourceStatus } from '$lib/api';

	// The webspace title renders synchronously from the route param, same
	// as Phase 1. The chip row below it DOES have its own async load/error
	// state (GET /api/sources, owned by the caller) — unlike Phase 1's
	// title-only header, a non-critical sources failure must never block
	// or blank the primary stream view, so the row renders only once
	// sourcesState is 'ready' and at least one source is configured
	// (shouldShowSourceRows, 02-UI-SPEC.md E1/E5, unchanged this phase).
	let {
		webspace,
		sources,
		sourcesState,
		selectedSources,
		onfilter,
		onrefresh,
		onrefreshall,
		searchQuery,
		onsearch
	}: {
		webspace: string;
		sources: SourceStatus[];
		sourcesState: 'loading' | 'error' | 'ready';
		selectedSources: Set<string>;
		onfilter: (name: string) => void;
		onrefresh: (name: string) => void;
		onrefreshall: () => void;
		searchQuery: string;
		onsearch: (q: string) => void;
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
		<!--
		  D-01: one merged chip per configured instance replaces Phase 2's
		  two rows (health + filter). Chips render in config-declared
		  instance order — never sorted or reordered by health state, so a
		  chip's position stays a stable target across health changes.
		  `flex-nowrap` keeps this a single line at any instance count; Task
		  2 adds the measured overflow popover that keeps it that way past
		  the point every chip's natural width no longer fits.
		-->
		<div class="mt-4 flex flex-nowrap items-center gap-2 overflow-hidden">
			{#each sources as source (source.name)}
				<SourceChip
					{source}
					selected={selectedSources.has(source.name)}
					onfilter={(name) => onfilter(name)}
					{onrefresh}
				/>
			{/each}
			<Button variant="outline" size="sm" class="shrink-0" onclick={onrefreshall}
				>Refresh all</Button
			>
		</div>
	{/if}

	<!--
	  The search box renders whenever the webspace does — unlike the row
	  above, it is NOT gated behind shouldShowSourceRows: searching the
	  local index doesn't depend on any source being reachable, so a
	  sourceless webspace or a sources-request failure never hides it.
	-->
	<div class="mt-3">
		<SearchBox query={searchQuery} onquery={onsearch} />
	</div>
</header>
