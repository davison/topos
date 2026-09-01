<script lang="ts">
	import StreamRow from './StreamRow.svelte';
	import StreamLoadingSkeleton from './StreamLoadingSkeleton.svelte';
	import {
		searchVariant,
		searchCopy,
		noMatchesHeading,
		sourceSearchSummary,
		sourceSearchTone,
		sourceSearchElapsed,
		searchSourcesCopy
	} from '$lib/format';
	import type { SearchResult, SourceStatus, SourceSearchStatus } from '$lib/api';

	// The results region (E2, 03-UI-SPEC.md): renders in place of
	// StreamList inside the same stream pane whenever a query is active, so
	// search introduces no second scroll region. searchVariant (format.ts)
	// is the single, pure, unit-tested decision this component renders
	// from — exactly one branch, never two.
	let {
		query,
		state,
		results,
		sources = null,
		sourcesState = 'idle',
		selectedId,
		onselect,
		staleSources,
		sourcesByInstance
	}: {
		query: string;
		state: 'idle' | 'loading' | 'error' | 'ready';
		results: SearchResult[];
		// The source fan-out's status map and its own lifecycle (M2-R2, #54):
		// pending while the index answer is already on screen and the
		// sources are still answering; ready with the map; error when the
		// second request failed and the index rows stand alone.
		sources?: Record<string, SourceSearchStatus> | null;
		sourcesState?: 'idle' | 'pending' | 'ready' | 'error';
		selectedId: string | null;
		onselect: (id: string) => void;
		staleSources: Set<string>;
		sourcesByInstance: Map<string, SourceStatus>;
	} = $props();

	let variant = $derived(searchVariant(query, state, results.length));
	// Instances in config order where known, so the row is stable across
	// searches; the map's own order for any instance config no longer lists.
	let sourceRows = $derived.by(() => {
		if (!sources) return [] as Array<[string, SourceSearchStatus]>;
		const known = Array.from(sourcesByInstance.keys()).filter((id) => id in sources);
		const rest = Object.keys(sources).filter((id) => !sourcesByInstance.has(id));
		return [...known, ...rest].map((id) => [id, sources[id]] as [string, SourceSearchStatus]);
	});
</script>

<!-- The per-source status row (M2-R2, #54): what each source in the
     webspace did with this search, updating as the fan-out lands — the
     index rows above/below never wait for it. Closed vocabulary from the
     kernel: ok | unsupported | timeout | error. -->
{#snippet sourcesRow()}
	{#if sourcesState !== 'idle'}
		<div
			class="flex flex-wrap items-center gap-x-3 gap-y-1 px-1 text-[14px] leading-[1.4] text-muted-foreground"
			data-search-sources={sourcesState}
			aria-live="polite"
		>
			{#if sourcesState === 'pending'}
				<span>{searchSourcesCopy.pending}</span>
			{:else if sourcesState === 'error'}
				<span class="text-warning">{searchSourcesCopy.failed}</span>
			{:else}
				{#each sourceRows as [id, status] (id)}
					<span
						class="inline-flex items-center gap-1"
						data-search-source={id}
						data-search-source-status={status.status}
						title={status.note || status.error || undefined}
					>
						<span class="text-foreground">{sourcesByInstance.get(id)?.display_name ?? id}</span>
						<span
							class={sourceSearchTone(status.status) === 'warning'
								? 'text-warning'
								: sourceSearchTone(status.status) === 'ok'
									? 'text-success'
									: undefined}>{sourceSearchSummary(status)}</span
						>
						<!-- Elapsed renders for EVERY outcome (PR #61 review round 1)
						     — never only in a tooltip, never displaced by a note. -->
						<span class="tabular-nums" data-search-source-elapsed
							>· {sourceSearchElapsed(status.elapsed_ms)}</span
						>
					</span>
				{/each}
			{/if}
		</div>
	{/if}
{/snippet}

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
	<div class="flex h-full flex-col gap-3">
		{@render sourcesRow()}
		<div class="flex flex-1 flex-col items-center justify-center gap-2 px-6 py-12 text-center">
			<p class="text-[20px] leading-[1.2] font-semibold text-foreground">
				{noMatchesHeading(query)}
			</p>
			<p class="max-w-md text-[16px] leading-[1.5] text-muted-foreground">
				{searchCopy.emptyBody}
			</p>
		</div>
	</div>
{:else}
	<!-- Populated: one row per result, in exactly the order the API
	     returned — no sort, re-rank, group, or source filter of any kind.
	     Search deliberately spans every source in the webspace. -->
	<div class="flex flex-col gap-3">
		{@render sourcesRow()}
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
				matchedIn={result.matched_in}
				unsynced={!result.indexed}
			/>
		{/each}
	</div>
{/if}
