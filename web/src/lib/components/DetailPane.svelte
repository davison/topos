<script lang="ts">
	import { getItem, contentUrl, ApiError, type StreamItem, type ItemContent } from '$lib/api';
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';
	import { Alert, AlertTitle, AlertDescription, AlertAction } from '$lib/components/ui/alert/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import OpenInSource from '$lib/components/OpenInSource.svelte';
	import FileText from '@lucide/svelte/icons/file-text';
	import AlertTriangle from '@lucide/svelte/icons/alert-triangle';
	import { detailPaneState, detailBodyVariant, formatItemDate, highlightText } from '$lib/format';

	// item is the stream row already held in memory — the header below
	// renders from it synchronously, before getItem(id) resolves (stage
	// one of the two-stage render this pane implements). displayName and
	// sourceReachable are supplied by the caller (+page.svelte, sourced
	// from the live GET /api/sources response) — this pane is shared
	// across every source, so neither its copy nor its unreachable/error
	// branch choice may hardcode or guess at one source's identity.
	// searchQuery (UI-09) is the active in-webspace search string, also
	// supplied by +page.svelte — threaded into the html-branch iframe src
	// via contentUrl so the kernel can highlight matched terms server-side
	// (the sandboxed iframe is an opaque origin, so this is the only
	// channel into that document; see rendition.go's own doc comment).
	let {
		item,
		displayName,
		sourceReachable,
		searchQuery
	}: {
		item: StreamItem;
		displayName: string;
		sourceReachable: boolean;
		searchQuery: string;
	} = $props();

	let content: ItemContent | null = $state(null);
	let loadingContent = $state(true);
	let fetchErrorCode: string | null = $state(null);

	// Guards against a stale-response race: if the user selects item A
	// (slow fetch), then item B (fast fetch) before A resolves, A's later
	// resolution must not clobber B's already-rendered content. Mirrors the
	// searchRequestSeq pattern in +page.svelte's handleSearch.
	let contentRequestSeq = 0;

	// The one place that decides which of the four mutually exclusive
	// states this pane shows (D-10) — see format.ts for the full
	// precedence rule and staleness.test.ts for its unit tests.
	let paneState = $derived(detailPaneState(content, fetchErrorCode, sourceReachable));

	// The one place that decides which of the body region's four mutually
	// exclusive branches to render, once paneState has resolved to
	// 'loaded' — see format.ts for the full precedence rule and
	// detail-body.test.ts for its unit tests.
	let bodyVariant = $derived(detailBodyVariant(content));

	async function loadContent(id: string) {
		loadingContent = true;
		fetchErrorCode = null;
		content = null;
		const seq = ++contentRequestSeq;
		try {
			const detail = await getItem(id);
			if (seq !== contentRequestSeq) return; // a newer selection has since superseded this one
			content = detail.content;
		} catch (err) {
			// Any live-fetch failure (source unreachable, network,
			// unexpected shape) lands here; detailPaneState maps it to the
			// unreachable or generic-error branch below based on this
			// item's source's live reachability, not the specific error
			// code — the deep link stays usable either way.
			if (seq !== contentRequestSeq) return;
			fetchErrorCode = err instanceof ApiError ? err.code : 'unknown_error';
		} finally {
			if (seq === contentRequestSeq) loadingContent = false;
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
		<!-- UI-09 (G-06-1): the title is FTS-indexed alongside preview/body
		     (kernel/index/schema.go), so a title-only match is a routine
		     search outcome — the header must highlight it exactly like the
		     body does below, through the same shared .search-highlight
		     class (app.css), or the user gets no visible explanation for
		     why the item surfaced. -->
		<h2 class="text-[20px] leading-[1.2] font-semibold text-foreground">
			{#each highlightText(item.title, searchQuery) as segment, i (i)}
				<span class={segment.match ? 'search-highlight' : undefined}>{segment.text}</span>
			{/each}
		</h2>
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
		<OpenInSource link={item.link} {displayName} />
	</header>

	<!-- The one physical rendering of loaded extracted text — shared by
	     the media branch (below a fixed-height preview box) and the
	     text-only branch (alone, taking the pane's full remaining
	     height), so the typography can never drift between the two.
	     UI-09: each segment comes from highlightText (format.ts), the
	     client half of the shared kernel/client term-derivation rule —
	     matched segments carry the .search-highlight class (declared once,
	     globally, in app.css — shared with the title above and with
	     StreamRow.svelte's title/snippet), unmatched segments carry no
	     class. Highlighting changes colour
	     only, never size or weight: highlighted text still inherits this
	     block's own type role. Segment text is rendered through Svelte's
	     default text binding — never via a raw-HTML directive — so no
	     escaping is needed here: highlightText itself never returns
	     markup. -->
	{#snippet loadedTextBlock()}
		<div
			class="min-h-0 flex-1 overflow-y-auto text-[16px] leading-[1.5] whitespace-pre-wrap text-foreground"
		>
			{#each highlightText(content?.text ?? '', searchQuery) as segment, i (i)}
				<span class={segment.match ? 'search-highlight' : undefined}>{segment.text}</span>
			{/each}
		</div>
	{/snippet}

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
	{:else if bodyVariant === 'html'}
		<!-- Rendered markdown (SilverBullet, D-04) or a sanitized HTML
		     email rendition (Proton, 03-09) IS the item's content, not a
		     preview thumbnail alongside separate extracted text — unlike
		     the media branch below, it occupies the pane's full remaining
		     body (min-h-0 flex-1), never the small fixed-height preview
		     box. Content still scrolls inside the iframe's own document
		     and never pushes this pane's own layout (UI-SPEC). The
		     sanitized HTML is served through the kernel's own hardened,
		     sandboxed rendition route and rendered inside this iframe —
		     never injected into the SPA document via Svelte's raw-HTML
		     directive, which would discard the sandbox boundary this
		     iframe provides for free (RESEARCH.md's explicit
		     anti-pattern). -->
		<div class="min-h-0 flex-1 overflow-hidden rounded-lg border border-border bg-card">
			<iframe title={item.title} src={contentUrl(item.id, searchQuery)} class="h-full w-full"></iframe>
		</div>
	{:else if bodyVariant === 'media'}
		<div class="flex min-h-0 flex-1 flex-col gap-6">
			<div class="h-72 shrink-0 overflow-hidden rounded-lg border border-border bg-card">
				{#if content?.rendition?.mime_type === 'application/pdf'}
					<iframe title={item.title} src={contentUrl(item.id)} class="h-full w-full"></iframe>
				{:else}
					<img src={contentUrl(item.id)} alt={item.title} class="h-full w-full object-contain" />
				{/if}
			</div>
			{#if content?.text}
				{@render loadedTextBlock()}
			{/if}
		</div>
	{:else if bodyVariant === 'text'}
		<!-- Text is the whole content — no rendition was offered for this
		     item (03-09: a Proton email whose plugin declined to emit one
		     because the plain-text part was already renderable), so the
		     text block takes the pane's full remaining height, with no
		     fixed-height preview box announcing an absent preview above
		     it. Renders the SAME loadedTextBlock snippet the media branch
		     uses above, so the typography can never drift between the two
		     surfaces that show extracted text. -->
		{@render loadedTextBlock()}
	{:else}
		<!-- Nothing at all to show — reuses the placeholder icon and copy
		     verbatim from the former media-branch placeholder case
		     (03-UI-SPEC.md Copywriting Contract: this phase's detail-pane
		     copy is unchanged from Phase 1/2). Given the pane's remaining
		     body rather than a fixed-height box, matching the other
		     branches. -->
		<div class="flex min-h-0 flex-1 flex-col items-center justify-center gap-2 text-muted-foreground">
			<FileText class="size-10" />
			<p class="text-[14px]">No preview available</p>
		</div>
	{/if}
</div>
