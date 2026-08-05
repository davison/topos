<script lang="ts">
	import { onMount } from 'svelte';
	import { listWebspaces, type Webspace } from '$lib/api';
	import * as Card from '$lib/components/ui/card/index.js';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';

	let webspaces: Webspace[] = $state([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	onMount(async () => {
		try {
			const res = await listWebspaces();
			webspaces = res.webspaces;
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>topos</title>
</svelte:head>

<main class="mx-auto max-w-3xl px-6 py-12">
	<h1 class="text-[28px] leading-[1.2] font-semibold text-foreground">topos</h1>

	{#if loading}
		<div class="mt-6 flex flex-col gap-3">
			{#each Array(3) as _}
				<Skeleton class="h-20 w-full rounded-lg" />
			{/each}
		</div>
	{:else if error}
		<p class="mt-6 text-[16px] text-muted-foreground">
			Couldn't load this webspace — the topos service didn't respond. Check that it's
			running, then retry.
		</p>
	{:else if webspaces.length === 0}
		<p class="mt-6 text-[16px] text-muted-foreground">
			No webspaces are configured yet. Add one to ~/.config/topos/config.toml.
		</p>
	{:else}
		<div class="mt-6 flex flex-col gap-3">
			{#each webspaces as ws (ws.name)}
				<a href={`/w/${ws.name}`}>
					<Card.Root>
						<Card.Header>
							<Card.Title class="text-[20px] leading-[1.2] font-semibold">{ws.name}</Card.Title>
							<Card.Description class="text-[14px] text-muted-foreground">
								{ws.item_count} {ws.item_count === 1 ? 'item' : 'items'}
							</Card.Description>
						</Card.Header>
					</Card.Root>
				</a>
			{/each}
		</div>
	{/if}
</main>
