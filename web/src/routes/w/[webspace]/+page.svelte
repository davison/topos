<script lang="ts">
	import { onMount } from 'svelte';
	import { page } from '$app/state';
	import { getStream, type StreamItem } from '$lib/api';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';

	// The [webspace] dynamic segment always matches for this route, so
	// page.params.webspace is never actually undefined at runtime — the
	// fallback only satisfies the type checker.
	let webspace = $derived(page.params.webspace ?? '');
	let items: StreamItem[] = $state([]);
	let loading = $state(true);
	let error = $state<string | null>(null);

	function formatDate(unix: number): string {
		if (!unix) return '';
		return new Date(unix * 1000).toLocaleDateString(undefined, {
			year: 'numeric',
			month: 'short',
			day: 'numeric'
		});
	}

	onMount(async () => {
		try {
			const res = await getStream(webspace);
			items = res.items;
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>{webspace} — webspaces</title>
</svelte:head>

<header class="border-b border-border bg-card px-6 py-4">
	<h1 class="truncate text-[28px] leading-[1.2] font-semibold text-foreground" title={webspace}>
		{webspace}
	</h1>
</header>

<main class="mx-auto max-w-3xl px-6 py-8">
	{#if loading}
		<div class="flex flex-col gap-3">
			{#each Array(4) as _}
				<Skeleton class="h-16 w-full rounded-lg" />
			{/each}
		</div>
	{:else if error}
		<p class="text-[16px] text-muted-foreground">
			The webspaces service didn't respond — check that it's running, then retry.
		</p>
	{:else if items.length === 0}
		<div class="py-12 text-center">
			<p class="text-[20px] font-semibold text-foreground">Nothing here yet</p>
			<p class="mt-2 text-[16px] text-muted-foreground">
				No paperless-ngx documents match this webspace's keywords yet. Check your tags, or wait
				for the next sync.
			</p>
		</div>
	{:else}
		<ul class="flex flex-col divide-y divide-border">
			{#each items as item (item.id)}
				<li class="py-4">
					<a href={item.link.url} target="_blank" rel="noreferrer" class="block">
						<p class="text-[20px] leading-[1.2] font-semibold text-foreground">{item.title}</p>
						<p class="mt-1 text-[14px] text-muted-foreground">
							{formatDate(item.timestamp_unix)}
						</p>
					</a>
				</li>
			{/each}
		</ul>
	{/if}
</main>
