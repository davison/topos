<script lang="ts">
	import { getItem, contentUrl, sourceDisplayName, type StreamItem, type ItemContent } from '$lib/api';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
	import { Alert, AlertTitle, AlertDescription, AlertAction } from '$lib/components/ui/alert/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import OpenInSource from '$lib/components/OpenInSource.svelte';
	import FileText from '@lucide/svelte/icons/file-text';

	// item is the stream row already held in memory — the header below
	// renders from it synchronously, before getItem(id) resolves (stage
	// one of the two-stage render this pane implements).
	let { item }: { item: StreamItem } = $props();

	let content: ItemContent | null = $state(null);
	let loadingContent = $state(true);
	let loadFailed = $state(false);

	// displayName parameterizes the source-specific copy below
	// (RESEARCH.md Pitfall 3) — this pane is shared across every source,
	// so neither the failure copy nor OpenInSource's button label may
	// hardcode one source's name.
	let displayName = $derived(sourceDisplayName(item.source_type));

	function formatDate(unix: number): string {
		if (!unix) return '';
		return new Date(unix * 1000).toLocaleDateString(undefined, {
			year: 'numeric',
			month: 'short',
			day: 'numeric'
		});
	}

	async function loadContent(id: string) {
		loadingContent = true;
		loadFailed = false;
		content = null;
		try {
			const detail = await getItem(id);
			content = detail.content;
		} catch {
			// The approved error copy is generic ("didn't respond") and
			// does not distinguish error codes — any failure (source
			// unavailable, network, unexpected shape) surfaces the same
			// state, with the deep link still usable.
			loadFailed = true;
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
		<div class="flex flex-wrap items-center gap-2 text-[14px] text-muted-foreground">
			<span>{formatDate(item.timestamp_unix)}</span>
			{#each item.labels as label (label)}
				<span class="rounded-full bg-secondary px-2 py-0.5 text-secondary-foreground">{label}</span>
			{/each}
		</div>
		<OpenInSource link={item.link} sourceType={item.source_type} />
	</header>

	<!-- Stage two: live-fetched preview + extracted text. -->
	{#if loadingContent}
		<div class="flex min-h-0 flex-1 flex-col gap-6">
			<Skeleton class="h-72 w-full shrink-0 rounded-lg" />
			<Skeleton class="w-full flex-1 rounded-lg" />
		</div>
	{:else if loadFailed}
		<Alert variant="destructive">
			<AlertTitle>Couldn't load this document</AlertTitle>
			<AlertDescription>
				{displayName} didn't respond. It may be offline — try again, or open it directly in
				{displayName}.
			</AlertDescription>
			<AlertAction>
				<Button variant="outline" size="sm" onclick={() => loadContent(item.id)}>Retry</Button>
			</AlertAction>
		</Alert>
	{:else if content}
		<div class="flex min-h-0 flex-1 flex-col gap-6">
			<div class="h-72 shrink-0 overflow-hidden rounded-lg border border-border bg-card">
				{#if content.available && content.rendition?.mime_type === 'application/pdf'}
					<iframe title={item.title} src={contentUrl(item.id)} class="h-full w-full"></iframe>
				{:else if content.available && content.rendition?.mime_type === 'text/html'}
					<!-- Sanitized rendered markdown (SilverBullet, D-04): served
					     through the kernel's own hardened, sandboxed rendition
					     route and rendered inside this iframe — never injected
					     into the SPA document via Svelte's raw-HTML directive,
					     which would discard the sandbox boundary this iframe
					     provides for free (RESEARCH.md's explicit anti-pattern). -->
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
			{#if content.text && content.rendition?.mime_type !== 'text/html'}
				<div
					class="min-h-0 flex-1 overflow-y-auto text-[16px] leading-[1.5] whitespace-pre-wrap text-foreground"
				>
					{content.text}
				</div>
			{/if}
		</div>
	{/if}
</div>
