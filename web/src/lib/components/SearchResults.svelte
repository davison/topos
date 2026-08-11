<script lang="ts">
	import StreamRow from './StreamRow.svelte';
	import StreamLoadingSkeleton from './StreamLoadingSkeleton.svelte';
	import { searchVariant, searchCopy, noMatchesHeading } from '$lib/format';
	import type { SearchResult, SourceStatus } from '$lib/api';

	// The results region (E2, 03-UI-SPEC.md): renders in place of
	// StreamList inside the same stream pane whenever a query is active, so
	// search introduces no second scroll region. searchVariant (format.ts)
	// is the single, pure, unit-tested decision this component renders
	// from — exactly one branch, never two.
	let {
		query,
		state,
		results,
		selectedId,
		onselect,
		staleSources,
		sourcesByInstance
	}: {
		query: string;
		state: 'idle' | 'loading' | 'error' | 'ready';
		results: SearchResult[];
		selectedId: string | null;
		onselect: (id: string) => void;
		staleSources: Set<string>;
		sourcesByInstance: Map<string, SourceStatus>;
	} = $props();

	let variant = $derived(searchVariant(query, state, results.length));
</script>

{#if variant === 'idle'}
	<!-- Renders nothing — the caller shows the stream instead. -->
{:else if variant === 'loading'}
	<!-- Shared with StreamList's loading branch (WR-04): four skeleton
	     rows at the real row dimensions, so the list doesn't reflow when
	     results arrive. -->
	<StreamLoadingSkeleton />
{:else if variant === 'error'}
	<!-- Inline, non-blocking: replaces only this region. The search box
	     and the rest of the page stay fully interactive (T-03-24 backstop
	     row, E2 error). -->
	<div class="flex h-full flex-col items-center justify-center px-6 py-12 text-center">
		<p class="max-w-md text-[16px] leading-[1.5] text-foreground">
			{searchCopy.errorInline}
		</p>
	</div>
{:else if variant === 'empty'}
	<!-- Zero matches for a non-empty query: distinct copy from the
	     unfiltered stream's own empty state, never "Nothing here yet".
	     The raw query is interpolated as plain text content (Svelte's
	     default text binding), never markup (T-03-21). -->
	<div class="flex h-full flex-col items-center justify-center gap-2 px-6 py-12 text-center">
		<p class="text-[20px] leading-[1.2] font-semibold text-foreground">
			{noMatchesHeading(query)}
		</p>
		<p class="max-w-md text-[16px] leading-[1.5] text-muted-foreground">
			{searchCopy.emptyBody}
		</p>
	</div>
{:else}
	<!-- Populated: one row per result, in exactly the order the API
	     returned — no sort, re-rank, group, or source filter of any kind.
	     Search deliberately spans every source in the webspace. -->
	<div class="flex flex-col gap-3">
		{#each results as result (result.id)}
			<StreamRow
				item={result}
				selected={result.id === selectedId}
				onselect={() => onselect(result.id)}
				stale={staleSources.has(result.source)}
				sourceDisplayName={sourcesByInstance.get(result.source)?.display_name ??
					result.source_display_name}
				plugin={sourcesByInstance.get(result.source)?.plugin ?? ''}
				snippet={result.snippet}
				searchQuery={query}
			/>
		{/each}
	</div>
{/if}
