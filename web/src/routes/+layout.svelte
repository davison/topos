<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';

	let { children } = $props();

	// Only the /w/[webspace] route has a `webspace` param — the landing
	// page ("/") renders its own heading and has no use for the fixed
	// h-screen two-pane wrapper below.
	//
	// WebspaceHeader itself moved from here into +page.svelte in Phase 2
	// (02-03-PLAN.md): its new props (sources, sourcesState, the filter
	// selection, the refresh handlers) are all owned by the page's own
	// per-webspace state, and a layout can't receive props back from the
	// page it renders via {@render children()} — the header has to be
	// rendered by whichever component actually owns that state.
	let webspace = $derived(page.params.webspace);
</script>

<svelte:head>
	<link rel="icon" type="image/png" href="/app-icon.png" />
</svelte:head>

{#if webspace}
	<div class="flex h-screen flex-col">
		<div class="min-h-0 flex-1">
			{@render children()}
		</div>
	</div>
{:else}
	{@render children()}
{/if}
