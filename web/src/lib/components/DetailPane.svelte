<script lang="ts">
	import { getItem, contentUrl, ApiError, type StreamItem, type ItemContent } from '$lib/api';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
	import { Alert, AlertTitle, AlertDescription, AlertAction } from '$lib/components/ui/alert/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import OpenInSource from '$lib/components/OpenInSource.svelte';
	import FileText from '@lucide/svelte/icons/file-text';
	import AlertTriangle from '@lucide/svelte/icons/alert-triangle';
	import { detailPaneState, formatItemDate } from '$lib/format';

	// item is the stream row already held in memory — the header below
	// renders from it synchronously, before getItem(id) resolves (stage
	// one of the two-stage render this pane implements). displayName and
	// sourceReachable are supplied by the caller (+page.svelte, sourced
	// from the live GET /api/sources response) — this pane is shared
	// across every source, so neither its copy nor its unreachable/error
	// branch choice may hardcode or guess at one source's identity.
	let {
		item,
		displayName,
		sourceReachable
	}: { item: StreamItem; displayName: string; sourceReachable: boolean } = $props();

	let content: ItemContent | null = $state(null);
	let loadingContent = $state(true);
	let fetchErrorCode: string | null = $state(null);

	// The one place that decides which of the four mutually exclusive
	// states this pane shows (D-10) — see format.ts for the full
	// precedence rule and staleness.test.ts for its unit tests.
	let paneState = $derived(detailPaneState(content, fetchErrorCode, sourceReachable));

	async function loadContent(id: string) {
		loadingContent = true;
		fetchErrorCode = null;
		content = null;
		try {
			const detail = await getItem(id);
			content = detail.content;
		} catch (err) {
			// Any live-fetch failure (source unreachable, network,
			// unexpected shape) lands here; detailPaneState maps it to the
			// unreachable or generic-error branch below based on this
			// item's source's live reachability, not the specific error
			// code — the deep link stays usable either way.
			fetchErrorCode = err instanceof ApiError ? err.code : 'unknown_error';
		} finally {
			loadingContent = false;
		}
	}

	// Re-fetch live content whenever the selected item changes.
	$effect(() => {
		loadContent(item.id);
	});
</script>

<div class="flex h-full min-h-0 flex-col gap-6">
	<!-- Stage one: instant metadata, synchronous — never waits on a network call. -->
	<header class="flex shrink-0 flex-col gap-2">
		<h2 class="text-[20px] leading-[1.2] font-semibold text-foreground">{item.title}</h2>
		{#if item.group_label}
			<p class="truncate text-[14px] leading-[1.4] text-muted-foreground" title={item.group_label}>
				{item.group_label}
			</p>
		{/if}
		<div class="flex flex-wrap items-center gap-2 text-[14px] text-muted-foreground">
			<span>{formatItemDate(item.timestamp_unix)}</span>
			{#each item.labels as label (label)}
				<span class="rounded-full bg-secondary px-2 py-0.5 text-secondary-foreground">{label}</span>
			{/each}
		</div>
		<OpenInSource link={item.link} sourceType={item.source_type} />
	</header>

	<!-- Stage two: live-fetched preview + extracted text, or one of the
	     three failure states below — never a blank pane (D-10). -->
	{#if loadingContent}
		<div class="flex min-h-0 flex-1 flex-col gap-6">
			<Skeleton class="h-72 w-full shrink-0 rounded-lg" />
			<Skeleton class="w-full flex-1 rounded-lg" />
		</div>
	{:else if paneState === 'unreachable' || paneState === 'deleted'}
		<!-- D-10: the source is unreachable, or the item is confirmed gone
		     at the source — either way the item itself is still viewable,
		     so this is a non-destructive alert layered over the cached
		     preview (item.preview / item.thumbnail_url — already in
		     memory from the stream, not a re-fetch), never a replacement
		     for the pane. -->
		<div class="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto">
			<Alert>
				<AlertTriangle />
				<AlertTitle>
					{paneState === 'deleted' ? 'No longer available' : 'Source unreachable'}
				</AlertTitle>
				<AlertDescription>
					{#if paneState === 'deleted'}
						No longer available at {displayName} — showing the last synced version.
					{:else}
						This source is currently unreachable — showing the last synced version.
					{/if}
				</AlertDescription>
			</Alert>
			{#if item.thumbnail_url}
				<div class="h-72 shrink-0 overflow-hidden rounded-lg border border-border bg-card">
					<img src={item.thumbnail_url} alt={item.title} class="h-full w-full object-contain" />
				</div>
			{/if}
			{#if item.preview}
				<p class="text-[16px] leading-[1.5] whitespace-pre-wrap text-foreground">
					{item.preview}
				</p>
			{/if}
		</div>
	{:else if paneState === 'error'}
		<Alert variant="destructive">
			<AlertTitle>Couldn't load this item</AlertTitle>
			<AlertDescription>
				{displayName} didn't respond. It may be offline — try again, or open it directly in
				{displayName}.
			</AlertDescription>
			<AlertAction>
				<Button variant="outline" size="sm" onclick={() => loadContent(item.id)}>Retry</Button>
			</AlertAction>
		</Alert>
	{:else if content && content.available && content.rendition?.mime_type === 'text/html'}
		<!-- Rendered markdown (SilverBullet, D-04) IS the item's content, not
		     a preview thumbnail alongside separate extracted text — unlike
		     the PDF/image branch below, it occupies the pane's full
		     remaining body (min-h-0 flex-1), never the small fixed-height
		     preview box. Content still scrolls inside the iframe's own
		     document and never pushes this pane's own layout (UI-SPEC). The
		     sanitized HTML is served through the kernel's own hardened,
		     sandboxed rendition route and rendered inside this iframe —
		     never injected into the SPA document via Svelte's raw-HTML
		     directive, which would discard the sandbox boundary this iframe
		     provides for free (RESEARCH.md's explicit anti-pattern). -->
		<div class="min-h-0 flex-1 overflow-hidden rounded-lg border border-border bg-card">
			<iframe title={item.title} src={contentUrl(item.id)} class="h-full w-full"></iframe>
		</div>
	{:else if content}
		<div class="flex min-h-0 flex-1 flex-col gap-6">
			<div class="h-72 shrink-0 overflow-hidden rounded-lg border border-border bg-card">
				{#if content.available && content.rendition?.mime_type === 'application/pdf'}
					<iframe title={item.title} src={contentUrl(item.id)} class="h-full w-full"></iframe>
				{:else if content.available && content.rendition?.mime_type.startsWith('image/')}
					<img src={contentUrl(item.id)} alt={item.title} class="h-full w-full object-contain" />
				{:else}
					<div class="flex h-full flex-col items-center justify-center gap-2 text-muted-foreground">
						<FileText class="size-10" />
						<p class="text-[14px]">No preview available</p>
					</div>
				{/if}
			</div>
			{#if content.text}
				<div
					class="min-h-0 flex-1 overflow-y-auto text-[16px] leading-[1.5] whitespace-pre-wrap text-foreground"
				>
					{content.text}
				</div>
			{/if}
		</div>
	{/if}
</div>
