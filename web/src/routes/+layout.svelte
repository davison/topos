<script lang="ts">
	import '../app.css';
	import favicon from '$lib/assets/favicon.svg';
	import { page } from '$app/state';
	import WebspaceHeader from '$lib/components/WebspaceHeader.svelte';

	let { children } = $props();

	// Only the /w/[webspace] route has a `webspace` param — the landing
	// page ("/") renders its own heading and has no use for this header,
	// so it stays outside the fixed-height two-pane wrapper below.
	let webspace = $derived(page.params.webspace);
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

{#if webspace}
	<div class="flex h-screen flex-col">
		<WebspaceHeader {webspace} />
		<div class="min-h-0 flex-1">
			{@render children()}
		</div>
	</div>
{:else}
	{@render children()}
{/if}
