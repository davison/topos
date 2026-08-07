<script lang="ts">
	import StreamRow from './StreamRow.svelte';
	import StreamEmpty from './StreamEmpty.svelte';
	import StreamError from './StreamError.svelte';
	import StreamLoadingSkeleton from './StreamLoadingSkeleton.svelte';
	import { filterItemsBySource, streamVariant } from '$lib/format';
	import type { StreamResponse, SourceStatus } from '$lib/api';

	let {
		state,
		response,
		selectedId,
		onselect,
		onretry,
		staleSources,
		selectedSources,
		sourcesByInstance
	}: {
		state: 'loading' | 'error' | 'ready';
		response: StreamResponse | null;
		selectedId: string | null;
		onselect: (id: string) => void;
		onretry: () => void;
		staleSources: Set<string>;
		selectedSources: Set<string>;
		sourcesByInstance: Map<string, SourceStatus>;
	} = $props();

	// streamVariant (format.ts) is the single, pure, unit-tested decision
	// this component renders from: a failed sync with zero items always
	// wins over any empty state, filtered or not — computed from the
	// response's own aggregate sync status and its UNFILTERED item count,
	// never the filtered subset below, so a filter can never mask a sync
	// failure. This ordering is the entire point of this component (see
	// PLAN.md prohibitions).
	let variant = $derived(response ? streamVariant(response, selectedSources) : null);
	let visibleItems = $derived(
		response ? filterItemsBySource(response.items, selectedSources) : []
	);
	// The filtered-empty copy names the single filtered source's display
	// name only when exactly one member is selected; a multi-select filter
	// (or no filter at all) falls back to the unfiltered empty-state copy
	// path, which takes no name (StreamEmpty.svelte).
	let filteredDisplayName = $derived.by(() => {
		if (selectedSources.size !== 1) return null;
		const [name] = selectedSources;
		return sourcesByInstance.get(name)?.display_name ?? name;
	});
</script>

{#if state === 'loading'}
	<!-- Shown only on the first load of a webspace — later fetches resolve
	     against the already-synced local index and finish before this
	     phase is ever perceptible. -->
	<StreamLoadingSkeleton />
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
