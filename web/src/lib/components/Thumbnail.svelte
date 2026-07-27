<script lang="ts">
	import FileText from '@lucide/svelte/icons/file-text';
	import type { StreamItem } from '$lib/api';

	// Renders a 40x52 leading thumbnail for a stream row. Falls back to a
	// generic document icon — never a broken-image glyph — both when the
	// kernel reports no thumbnail exists at all, and when the `img`'s own
	// fetch fails at request time (unsupported type, transient error).
	let { item }: { item: StreamItem } = $props();

	let hasThumbnail = $derived(Boolean(item.thumbnail_url));
	let broken = $state(false);

	// Reset the broken flag whenever the item changes, so navigating to a
	// different row with a working thumbnail doesn't inherit a prior
	// row's failure state.
	$effect(() => {
		item.id;
		broken = false;
	});
</script>

{#if hasThumbnail && !broken}
	<img
		src={item.thumbnail_url}
		alt=""
		loading="lazy"
		decoding="async"
		class="h-[52px] w-[40px] shrink-0 rounded-sm border border-border object-cover"
		onerror={() => (broken = true)}
	/>
{:else}
	<div
		class="flex h-[52px] w-[40px] shrink-0 items-center justify-center rounded-sm border border-border bg-secondary text-muted-foreground"
	>
		<FileText class="size-5" />
	</div>
{/if}
