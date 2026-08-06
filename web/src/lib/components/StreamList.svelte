<script lang="ts">
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
	import StreamRow from './StreamRow.svelte';
	import StreamEmpty from './StreamEmpty.svelte';
	import StreamError from './StreamError.svelte';
	import { filterItemsBySource, streamVariant } from '$lib/format';
	import type { StreamResponse, SourceStatus } from '$lib/api';

	let {
		state,
		response,
		selectedId,
		onselect,
		onretry,
		staleSources,
		selectedSource,
		sourcesByInstance
	}: {
		state: 'loading' | 'error' | 'ready';
		response: StreamResponse | null;
		selectedId: string | null;
		onselect: (id: string) => void;
		onretry: () => void;
		staleSources: Set<string>;
		selectedSource: string | null;
		sourcesByInstance: Map<string, SourceStatus>;
	} = $props();

	// streamVariant (format.ts) is the single, pure, unit-tested decision
	// this component renders from: a failed sync with zero items always
	// wins over any empty state, filtered or not — computed from the
	// response's own aggregate sync status and its UNFILTERED item count,
	// never the filtered subset below, so a filter can never mask a sync
	// failure. This ordering is the entire point of this component (see
	// PLAN.md prohibitions).
	let variant = $derived(response ? streamVariant(response, selectedSource) : null);
	let visibleItems = $derived(response ? filterItemsBySource(response.items, selectedSource) : []);
	let filteredDisplayName = $derived(
		selectedSource ? (sourcesByInstance.get(selectedSource)?.display_name ?? selectedSource) : null
	);
</script>

{#if state === 'loading'}
	<!-- Four skeleton rows at the real row dimensions (.stream-row-surface,
	     app.css) so the list doesn't reflow when data arrives. Shown only
	     on the first load of a webspace — later fetches resolve against
	     the already-synced local index and finish before this phase is
	     ever perceptible. -->
	<div class="flex flex-col gap-3">
		{#each Array(4) as _, i (i)}
			<Skeleton class="stream-row-surface w-full rounded-lg" />
		{/each}
	</div>
{:else if state === 'error'}
	<StreamError {onretry} />
{:else if variant === 'sync-failed' && response}
	<StreamError {onretry} syncError={response.sync.error} />
{:else if variant === 'empty-filtered'}
	<StreamEmpty displayName={filteredDisplayName} />
{:else if variant === 'empty'}
	<StreamEmpty />
{:else if response}
	<!-- Populated: one row per item, in exactly the order the API
	     returned — no sort, re-sort, re-group or filter of any kind
	     beyond the source narrowing above. Stale markers are decoration
	     on a row, never an input to ordering. -->
	<div class="flex flex-col gap-3">
		{#each visibleItems as item (item.id)}
			<StreamRow
				{item}
				selected={item.id === selectedId}
				onselect={() => onselect(item.id)}
				stale={staleSources.has(item.source)}
				sourceDisplayName={sourcesByInstance.get(item.source)?.display_name ??
					item.source_display_name}
			/>
		{/each}
	</div>
{/if}
