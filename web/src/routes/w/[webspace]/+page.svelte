<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import {
		getStream,
		getSources,
		refreshSource,
		refreshAll,
		searchWebspace,
		type StreamResponse,
		type SourceStatus,
		type SearchResult
	} from '$lib/api';
	import { resolveSourceFilters, toggleSourceFilter, serializeSourceFilters, staleSources } from '$lib/format';
	import WebspaceHeader from '$lib/components/WebspaceHeader.svelte';
	import StreamList from '$lib/components/StreamList.svelte';
	import SearchResults from '$lib/components/SearchResults.svelte';
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

	// Search state (KERN-05 browser half, 03-04): kept in component state
	// rather than the URL — the UI-SPEC specifies no URL persistence for
	// it, unlike the `?source=` filter param below. `searchRequestSeq` is
	// a monotonically increasing sequence number captured before each
	// await and compared after, so a slower earlier request can never
	// overwrite a faster later one (T-03-25) — reachable in normal use
	// with a 300ms debounce and as-you-type searching, not a theoretical
	// edge.
	let searchQuery = $state('');
	let searchState: 'idle' | 'loading' | 'error' | 'ready' = $state('idle');
	let searchResults = $state<SearchResult[]>([]);
	let searchRequestSeq = 0;

	let selectedItem = $derived(
		response?.items.find((i) => i.id === selectedId) ??
			searchResults.find((r) => r.id === selectedId) ??
			null
	);

	// Sources: fetched (and polled while any source is syncing)
	// independently of the stream, so a sources-request failure can never
	// block or blank the primary stream view (02-UI-SPEC.md E1 error row).
	// sourcesByInstance is keyed by SourceStatus.name (the source INSTANCE
	// id, D-08) — never source_type, so two instances of one plugin type
	// resolve to two independent map entries rather than colliding on one.
	let sources = $state<SourceStatus[]>([]);
	let sourcesState: 'loading' | 'error' | 'ready' = $state('loading');
	let sourcesByInstance = $derived(new Map(sources.map((s) => [s.name, s])));
	let staleInstances = $derived(staleSources(sources));

	// Filter selection lives in the URL query so a reload or a shared
	// deep link restores the same multi-source view (D-02, superseding
	// Phase 2's single-select D-09); an unrecognised member degrades that
	// member alone rather than the whole selection (T-02-17's multi-value
	// form, 06-RESEARCH.md Pitfall 4). `?sources=` (plural) replaces
	// Phase 2's `?source=` outright — no backward-compatible read of the
	// old singular key (06-02-PLAN.md decision).
	let selectedSources = $derived(
		resolveSourceFilters(page.url.searchParams.get('sources'), sources)
	);

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

	async function loadSources() {
		try {
			const res = await getSources();
			sources = res.sources;
			sourcesState = 'ready';
		} catch {
			sources = [];
			sourcesState = 'error';
		}
	}

	// While any source reports syncing, poll GET /api/sources on a modest
	// interval and stop as soon as none is syncing — the non-blocking
	// "in progress" indicator D-07 specifies; the stream itself is never
	// blocked while this runs (T-02-18: an index read plus a lightweight
	// per-plugin probe, bounded and local, stopping the moment nothing is
	// syncing).
	let pollHandle: ReturnType<typeof setInterval> | null = null;
	function ensurePolling() {
		if (pollHandle !== null) return;
		pollHandle = setInterval(async () => {
			await loadSources();
			if (!sources.some((s) => s.syncing) && pollHandle !== null) {
				clearInterval(pollHandle);
				pollHandle = null;
			}
		}, 2000);
	}

	async function handleRefreshSource(name: string) {
		ensurePolling();
		try {
			await refreshSource(name);
		} catch {
			// A failed refresh has no separate toast/dialog — the chip
			// itself is the error surface (D-08) once loadSources() below
			// picks up the newly recorded sync_runs failure.
		} finally {
			await Promise.all([loadSources(), load()]);
		}
	}

	async function handleRefreshAll() {
		ensurePolling();
		try {
			await refreshAll();
		} catch {
			// same as handleRefreshSource above.
		} finally {
			await Promise.all([loadSources(), load()]);
		}
	}

	function navigateFilters(next: Set<string>) {
		const url = new URL(page.url);
		const serialized = serializeSourceFilters(next);
		if (serialized) {
			url.searchParams.set('sources', serialized);
		} else {
			url.searchParams.delete('sources');
		}
		// replaceState (not a new history entry) so toggling a chip
		// repeatedly doesn't fill the back-button history; keepFocus/
		// noScroll so selecting a chip never steals focus or scrolls.
		goto(`${url.pathname}${url.search}`, { replaceState: true, keepFocus: true, noScroll: true });
	}

	function toggleFilter(name: string) {
		navigateFilters(toggleSourceFilter(selectedSources, name));
	}

	function clearFilters() {
		navigateFilters(new Set());
	}

	async function handleSearch(query: string) {
		searchQuery = query;

		// An empty or whitespace-only query resets to idle and clears the
		// results without issuing a request — clearing the box always
		// returns the untouched stream.
		if (query.trim() === '') {
			searchState = 'idle';
			searchResults = [];
			return;
		}

		searchState = 'loading';
		const seq = ++searchRequestSeq;
		try {
			const res = await searchWebspace(webspace, query);
			if (seq !== searchRequestSeq) return; // a newer request has since superseded this one
			searchResults = res.results;
			searchState = 'ready';
		} catch {
			if (seq !== searchRequestSeq) return;
			searchResults = [];
			searchState = 'error';
		}
	}

	// Re-fetch (and drop any stale selection/search) whenever the webspace
	// route param changes.
	$effect(() => {
		selectedId = null;
		searchQuery = '';
		searchState = 'idle';
		searchResults = [];
		load();
		loadSources();
	});
</script>

<svelte:head>
	<title>{webspace} — webspaces</title>
</svelte:head>

<div class="flex h-full min-h-0 flex-col">
	<WebspaceHeader
		{webspace}
		{sources}
		{sourcesState}
		{selectedSources}
		onfilter={toggleFilter}
		onclearfilters={clearFilters}
		onrefresh={handleRefreshSource}
		onrefreshall={handleRefreshAll}
		{searchQuery}
		onsearch={handleSearch}
	/>

	<main class="flex min-h-0 flex-1 gap-8 px-6 py-8">
		<!--
		  The stream pane owns its own independent scroll region
		  (overflow-y-auto, min-h-0) and never scrolls horizontally
		  (overflow-x-hidden) — scrolling it never moves the detail pane's
		  scroll position, which lives in its own region below. SearchResults
		  renders in place of StreamList whenever the trimmed query is
		  non-empty, inside this same pane, so search introduces no second
		  scroll region; the source-filter chips keep governing the stream,
		  which returns exactly as it was when the query is cleared.

		  Sizing: the detail pane (below) is the reading surface, so it
		  absorbs viewport width changes (flex-1). The stream pane holds a
		  fixed width (w-[480px], the same width the detail pane used to
		  own) whenever an item is selected and the detail pane is open --
		  driven by the same `selectedItem` value that gates the detail
		  pane's rendering, so the two can never disagree. With nothing
		  selected there is no sibling to size against, so the stream pane
		  falls back to flex-1 and keeps filling the full content width.
		-->
		<div
			class="min-h-0 min-w-0 overflow-x-hidden overflow-y-auto {selectedItem
				? 'w-[480px] shrink-0'
				: 'flex-1'}"
		>
			{#if searchQuery.trim()}
				<SearchResults
					query={searchQuery}
					state={searchState}
					results={searchResults}
					{selectedId}
					onselect={(id) => (selectedId = id)}
					staleSources={staleInstances}
					{sourcesByInstance}
				/>
			{:else}
				<StreamList
					state={loadState}
					{response}
					{selectedId}
					onselect={(id) => (selectedId = id)}
					onretry={load}
					staleSources={staleInstances}
					{selectedSources}
					{sourcesByInstance}
				/>
			{/if}
		</div>

		{#if selectedItem}
			<div class="flex min-w-0 flex-1 flex-col overflow-hidden border-l border-border pl-8">
				<DetailPane
					item={selectedItem}
					displayName={sourcesByInstance.get(selectedItem.source)?.display_name ??
						selectedItem.source_display_name ??
						selectedItem.source}
					sourceReachable={sourcesByInstance.get(selectedItem.source)?.reachable ?? true}
					{searchQuery}
				/>
			</div>
		{/if}
	</main>
</div>
