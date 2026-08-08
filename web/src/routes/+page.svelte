<script lang="ts">
	// Redirect-only root route (D-10, 07-03-PLAN.md Task 3). The standalone
	// card-list home page is retired — composition happens entirely in the
	// header now (WebspaceSwitcher, CreateWebspaceModal), so "/" has exactly
	// one job: send the visitor to a webspace, or — with none configured —
	// show them how to create the first one.
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { getConfig, type ConfigResponse } from '$lib/api';
	import { readLastWebspace, writeLastWebspace, resolveRedirectTarget } from '$lib/last-webspace';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import CreateWebspaceModal from '$lib/components/CreateWebspaceModal.svelte';

	// 'loading' while the redirect decision is being made (renders the
	// existing Skeleton treatment, unchanged from the retired page, so a
	// slow kernel never flashes the empty state on its way to a real
	// target); 'error' when the config request itself fails (existing
	// kernel-unreachable copy, unchanged); 'empty' when the kernel reports
	// zero webspaces — the only state this component actually renders
	// content for, since every other outcome navigates away via goto
	// before ever reaching here.
	let phase: 'loading' | 'error' | 'empty' = $state('loading');
	let configResponse = $state<ConfigResponse | null>(null);
	let createOpen = $state(false);

	onMount(async () => {
		try {
			const res = await getConfig();
			configResponse = res;
			const webspaceNames = Object.keys(res.config.webspaces);
			const target = resolveRedirectTarget(webspaceNames, readLastWebspace());
			if (target !== null) {
				// replaceState so the redirect itself never sits in the back
				// history — pressing back from the destination webspace must
				// not bounce through this route again.
				await goto(`/w/${encodeURIComponent(target)}`, { replaceState: true });
				return;
			}
			phase = 'empty';
		} catch {
			phase = 'error';
		}
	});

	async function handleCreated(name: string) {
		createOpen = false;
		writeLastWebspace(name);
		await goto(`/w/${encodeURIComponent(name)}`, { replaceState: true });
	}
</script>

<svelte:head>
	<title>topos</title>
</svelte:head>

{#if phase === 'loading'}
	<main class="mx-auto max-w-3xl px-6 py-12">
		<h1 class="text-[28px] leading-[1.2] font-semibold text-foreground">topos</h1>
		<div class="mt-6 flex flex-col gap-3">
			{#each Array(3) as _}
				<Skeleton class="h-20 w-full rounded-lg" />
			{/each}
		</div>
	</main>
{:else if phase === 'error'}
	<main class="mx-auto max-w-3xl px-6 py-12">
		<h1 class="text-[28px] leading-[1.2] font-semibold text-foreground">topos</h1>
		<p class="mt-6 text-[16px] text-muted-foreground">
			Couldn't load this webspace — the topos service didn't respond. Check that it's running,
			then retry.
		</p>
	</main>
{:else}
	<!--
	  Zero-webspaces empty state (D-10, UI-12 empty edge): full-page,
	  centred, replaces the entire header+stream region — never a blank
	  page, never a redirect loop (T-07-18, resolveRedirectTarget only ever
	  names a webspace the kernel's own response reported).
	-->
	<main class="flex min-h-screen flex-col items-center justify-center px-6 text-center">
		<h1 class="text-[20px] leading-[1.2] font-semibold text-foreground">No webspaces yet</h1>
		<p class="mt-2 max-w-md text-[16px] leading-[1.5] text-muted-foreground">
			A webspace pulls related items from your sources into one view. Create one to get
			started.
		</p>
		<Button class="mt-6" onclick={() => (createOpen = true)}>Create webspace</Button>
	</main>

	{#if configResponse}
		<CreateWebspaceModal
			open={createOpen}
			config={configResponse.config}
			baseHash={configResponse.hash}
			onclose={() => (createOpen = false)}
			oncreated={handleCreated}
		/>
	{/if}
{/if}
