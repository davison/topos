<script lang="ts">
	import { page } from '$app/state';
	import { getStream, type StreamResponse } from '$lib/api';
	import StreamList from '$lib/components/StreamList.svelte';
	import DetailPane from '$lib/components/DetailPane.svelte';

	// The [webspace] dynamic segment always matches for this route, so
	// page.params.webspace is never actually undefined at runtime — the
	// fallback only satisfies the type checker.
	let webspace = $derived(page.params.webspace ?? '');

	let response = $state<StreamResponse | null>(null);
	// Named loadState, not `state` — a local variable literally named
	// `state` collides with the `$state()` rune's auto-subscription
	// parsing (Svelte tries to treat `$state(...)` as a store
	// subscription to a variable named `state`, causing a "used before
	// its declaration" compiler error).
	let loadState: 'loading' | 'error' | 'ready' = $state('loading');
	let selectedId = $state<string | null>(null);
	let selectedItem = $derived(response?.items.find((i) => i.id === selectedId) ?? null);

	async function load() {
		loadState = 'loading';
		try {
			response = await getStream(webspace);
			loadState = 'ready';
		} catch {
			response = null;
			loadState = 'error';
		}
	}

	// Re-fetch (and drop any stale selection) whenever the webspace route
	// param changes.
	$effect(() => {
		selectedId = null;
		load();
	});
</script>

<svelte:head>
	<title>{webspace} — webspaces</title>
</svelte:head>

<main class="flex h-full min-h-0 gap-8 px-6 py-8">
	<!--
	  The stream pane owns its own independent scroll region
	  (overflow-y-auto, min-h-0) and never scrolls horizontally
	  (overflow-x-hidden) — scrolling it never moves the detail pane's
	  scroll position, which lives in its own region below.
	-->
	<div class="min-h-0 min-w-0 flex-1 overflow-x-hidden overflow-y-auto">
		<StreamList
			state={loadState}
			{response}
			{selectedId}
			onselect={(id) => (selectedId = id)}
			onretry={load}
		/>
	</div>

	{#if selectedItem}
		<div class="flex w-[480px] shrink-0 flex-col overflow-hidden border-l border-border pl-8">
			<DetailPane item={selectedItem} />
		</div>
	{/if}
</main>
