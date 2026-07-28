<script lang="ts">
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
	import StreamRow from './StreamRow.svelte';
	import StreamEmpty from './StreamEmpty.svelte';
	import StreamError from './StreamError.svelte';
	import type { StreamResponse } from '$lib/api';

	let {
		state,
		response,
		selectedId,
		onselect,
		onretry
	}: {
		state: 'loading' | 'error' | 'ready';
		response: StreamResponse | null;
		selectedId: string | null;
		onselect: (id: string) => void;
		onretry: () => void;
	} = $props();

	// Sync-failure check: MUST be evaluated, and rendered, before the
	// empty-state check below. A webspace whose sync run recorded a
	// failure and returned zero items must show the failure state, never
	// "Nothing here yet" — a silently under-reported webspace is the
	// worst possible failure for this product (see PLAN.md prohibitions).
	// This ordering is the entire point of this component.
	let syncFailed = $derived(
		response !== null && response.sync.status === 'error' && response.items.length === 0
	);
	let isEmpty = $derived(response !== null && !syncFailed && response.items.length === 0);
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
{:else if syncFailed && response}
	<StreamError {onretry} syncError={response.sync.error} />
{:else if isEmpty}
	<StreamEmpty />
{:else if response}
	<!-- Populated: one row per item, in exactly the order the API
	     returned — no sort, re-sort, re-group or filter of any kind. -->
	<div class="flex flex-col gap-3">
		{#each response.items as item (item.id)}
			<StreamRow {item} selected={item.id === selectedId} onselect={() => onselect(item.id)} />
		{/each}
	</div>
{/if}
