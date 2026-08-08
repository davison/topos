<script lang="ts">
	import { page } from '$app/state';
	import { goto } from '$app/navigation';
	import {
		getStream,
		getSources,
		refreshSource,
		refreshAll,
		searchWebspace,
		getConfig,
		putConfig,
		listPluginTypes,
		describePlugin,
		ApiError,
		CONFIG_CONFLICT_MESSAGE,
		type StreamResponse,
		type SourceStatus,
		type SearchResult,
		type ConfigResponse
	} from '$lib/api';
	import {
		resolveSourceFilters,
		toggleSourceFilter,
		serializeSourceFilters,
		staleSources,
		filterItemsBySource
	} from '$lib/format';
	import { setWebspaceFilter, removeSourceFromWebspace } from '$lib/config-edit';
	import WebspaceHeader from '$lib/components/WebspaceHeader.svelte';
	import StreamList from '$lib/components/StreamList.svelte';
	import StreamDateMarkers from '$lib/components/StreamDateMarkers.svelte';
	import SearchResults from '$lib/components/SearchResults.svelte';
	import DetailPane from '$lib/components/DetailPane.svelte';
	import CreateWebspaceModal from '$lib/components/CreateWebspaceModal.svelte';
	import EditSourceModal from '$lib/components/EditSourceModal.svelte';
	import ManageSourcesModal from '$lib/components/ManageSourcesModal.svelte';
	import { writeLastWebspace } from '$lib/last-webspace';

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

	// Search-promotion permanent filter state (D-16-D-19, 07-01-PLAN.md
	// Task 1): configResponse is the last GET/PUT /api/config result —
	// its `hash` is the base_hash the next putConfig call must echo back
	// (D-03), and its `config.webspaces[webspace].filter` array is this
	// webspace's saved permanent filter stack. filterBusy disables the
	// "Save as filter" button and every chip's remove control while a
	// write is in flight; filterError surfaces a save rejection (D-09
	// validation failure or D-03 hash conflict) as a fixed destructive
	// Alert in the header.
	let configResponse = $state<ConfigResponse | null>(null);
	let filterBusy = $state(false);
	let filterError = $state<string | null>(null);
	let filters = $derived(configResponse?.config.webspaces[webspace]?.filter ?? []);

	// pluginTypes backs the "+" add-source picker's "New {plugin type}…"
	// rows (D-11, 07-04-PLAN.md) — every discovered-but-not-necessarily-
	// configured plugin binary, fetched once alongside config and held
	// here rather than re-fetched per webspace visit (the discovered set
	// is boot-time filesystem state, not webspace-scoped). Declared here
	// (ahead of loadPluginTypes' own top-level call below) so that call
	// never reads this binding before its own $state initializer has run.
	let pluginTypes = $state<string[]>([]);

	// unknownConfigKeys surfaces GET /api/config's `unknown_keys` field
	// (already computed by the kernel, previously never read by the UI —
	// deviation, Rule 2: discovered live during the tracer checkpoint that
	// this plan's config.Store.Save guard, by design, refuses EVERY save
	// while config.toml carries a hand-authored key the Config struct
	// doesn't model, anywhere in the file (D-01's lossless-rewrite
	// prohibition) — so a pre-existing stray key silently blocks "Save as
	// filter" with only a post-click Alert as feedback. Surfacing this
	// proactively, before any save is attempted, is what makes that
	// blocked state discoverable rather than looking like the button does
	// nothing.
	let unknownConfigKeys = $derived(configResponse?.unknown_keys ?? []);

	// Webspace switcher / create-webspace state (D-10, 07-03-PLAN.md
	// Task 2). webspaceNames is every configured webspace's key in the
	// kernel's own GET /api/config order (Object.keys on an
	// already-JSON-parsed document preserves the order the response body
	// itself listed them in — the "config-declared order" wording's actual
	// stability guarantee: never re-sorted here, never reordered by any
	// local state). createOpen gates CreateWebspaceModal, shared by both
	// the switcher's "+ New webspace" item and (once 07-03 Task 3 lands)
	// the root route's zero-webspaces empty-state CTA.
	let webspaceNames = $derived(
		configResponse ? Object.keys(configResponse.config.webspaces) : []
	);
	let createOpen = $state(false);

	// handleSourceAdded refreshes every piece of state a successful
	// add-source save could have changed (D-07's eager reconcile): the
	// config document itself (new allowlist/match/source blocks), the
	// source list (a brand-new instance's health chip), and the stream
	// (its first eager sync's items, if any landed already).
	async function handleSourceAdded() {
		await Promise.all([loadConfig(navGeneration), loadSources(), load(navGeneration)]);
	}

	// Chip menu state (D-12, 07-04-PLAN.md Task 3). editVocabulary is
	// resolved via describePlugin against the instance's own stored
	// connection config — the same substitute Task 1's one-step
	// existing-instance flow uses (GET /api/sources carries no
	// match_vocabulary field; see AddSourceModal.svelte's own doc comment
	// for why). Unused (left []) in 'connection' mode.
	let editOpen = $state(false);
	let editMode = $state<'connection' | 'match'>('connection');
	let editInstance = $state<string | null>(null);
	let editVocabulary = $state<string[]>([]);

	async function handleChipEdit(name: string, kind: 'connection' | 'match' | 'remove') {
		if (kind === 'remove') {
			await handleRemoveSource(name);
			return;
		}
		if (!configResponse) return;
		editInstance = name;
		editMode = kind;
		editVocabulary = [];
		if (kind === 'match') {
			try {
				const source = configResponse.config.sources[name];
				const resp = await describePlugin({ plugin: source.plugin, source });
				editVocabulary = resp.match_vocabulary;
			} catch {
				// Match settings can still be viewed/edited against whatever
				// vocabulary resolved (possibly none) — a describe failure
				// here is not fatal, it just means the form renders no
				// fields until the instance can connect again.
			}
		}
		editOpen = true;
	}

	function handleEditClose() {
		editOpen = false;
	}

	async function handleEditSaved() {
		editOpen = false;
		await Promise.all([loadConfig(navGeneration), loadSources(), load(navGeneration)]);
	}

	// handleRemoveSource is a modal-less write (D-12/07-UI-SPEC.md's
	// Destructive Confirmation Contract: reversible in one more click via
	// "+", destroys no data) — reuses filterBusy/filterError, the same
	// disable-while-in-flight state and header error region every other
	// modal-less write (Save as filter / remove filter) already uses.
	async function handleRemoveSource(name: string) {
		if (!configResponse) return;
		filterBusy = true;
		try {
			const nextConfig = removeSourceFromWebspace(configResponse.config, webspace, name);
			const res = await putConfig({ base_hash: configResponse.hash, config: nextConfig });
			configResponse = res;
			filterError = null;
			await Promise.all([loadSources(), load(navGeneration)]);
		} catch (err) {
			filterError =
				err instanceof ApiError && err.code === 'config_changed_on_disk'
					? CONFIG_CONFLICT_MESSAGE
					: err instanceof ApiError
						? err.message
						: 'Something went wrong removing this source — check the browser console and try again.';
			await loadConfig(navGeneration);
		} finally {
			filterBusy = false;
		}
	}

	// Manage Sources modal state (D-13, 07-05-PLAN.md Task 1) — the one
	// escape hatch for instance/webspace deletion and the config-reload
	// affordance. manageOpen replaces the no-op-safe handleManageSources
	// placeholder 07-03/07-04 left in its own webspace-route wiring.
	let manageOpen = $state(false);

	function handleManageSources() {
		manageOpen = true;
	}

	async function handleManageSourcesChanged() {
		// D-07's eager reconcile, same shape as handleSourceAdded above: a
		// delete or a reload inside the modal can change the config
		// document, the source list, and the stream all at once.
		await Promise.all([loadConfig(navGeneration), loadSources(), load(navGeneration)]);
	}

	async function handleWebspaceCreated(name: string) {
		createOpen = false;
		writeLastWebspace(name);
		// Refresh this page's own config snapshot before navigating away —
		// keeps the current webspace's switcher list current even if
		// navigation is ever interrupted, rather than relying solely on the
		// destination route's own webspace-keyed effect to pick up the new
		// entry.
		await loadConfig(navGeneration);
		await goto(`/w/${encodeURIComponent(name)}`);
	}

	// navGeneration guards against a stale-webspace race: SvelteKit reuses
	// this page component instance across /w/A -> /w/B navigation (same
	// route file, only page.params.webspace changes), so a slow in-flight
	// getStream(A)/searchWebspace(A) call can resolve after the user has
	// already navigated to B. Bumped once per webspace-keyed $effect run
	// and checked after every await below (both load() and handleSearch())
	// so a request for a no-longer-current webspace can never overwrite
	// what's on screen.
	let navGeneration = 0;

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

	// The stream pane's scroll region — bound below so StreamDateMarkers
	// (UI-11) can measure the track height and resolve tick clicks against
	// the same element StreamRow.svelte's data-item-id attribute lives in.
	// visibleStreamItems mirrors StreamList.svelte's own internal
	// filterItemsBySource derivation exactly (same inputs, same function)
	// so the markers overlay can never disagree with what the stream
	// itself is rendering.
	let streamScrollEl: HTMLElement | null = $state(null);
	let streamScrollHeight = $state(0);
	let visibleStreamItems = $derived(
		response ? filterItemsBySource(response.items, selectedSources) : []
	);

	async function load(gen: number) {
		loadState = 'loading';
		try {
			const res = await getStream(webspace);
			if (gen !== navGeneration) return; // a newer webspace navigation has since superseded this one
			response = res;
			loadState = 'ready';
		} catch {
			if (gen !== navGeneration) return;
			response = null;
			loadState = 'error';
		}
	}

	// Fetched once (not webspace-scoped, not re-fetched on navigation) —
	// the discovered plugin binary set is boot-time filesystem state.
	async function loadPluginTypes() {
		try {
			const res = await listPluginTypes();
			pluginTypes = res.plugin_types;
		} catch {
			pluginTypes = [];
		}
	}
	loadPluginTypes();

	async function loadConfig(gen: number) {
		try {
			const res = await getConfig();
			if (gen !== navGeneration) return; // a newer webspace navigation has since superseded this one
			configResponse = res;
		} catch {
			if (gen !== navGeneration) return;
			configResponse = null;
		}
	}

	// writeFilter is the one write path both saveFilter and removeFilter go
	// through (07-01-PLAN.md Task 2: "there is no second write path"):
	// mutate only this webspace's filter array via setWebspaceFilter
	// (07-03-PLAN.md Task 2 — the single place a webspace's config is
	// edited, config-edit.ts), putConfig({ base_hash, config }). On success
	// replace configResponse with the response, clear filterError, and
	// (only for a save, never a remove) clear the search box back to empty
	// so it is ready for a further refining search (D-18). On an ApiError,
	// set filterError to the fixed copy for a hash conflict (D-03) or the
	// kernel's own verbatim message otherwise (D-09), then refresh
	// getConfig() so the next attempt starts from current state.
	//
	// Everything below (including the "config not loaded yet" guard) runs
	// inside one try/finally so no exit from this function can be silent —
	// a bug fixed here (07-01 tracer checkpoint, live-dev-session repro)
	// found that `structuredClone(configResponse.config)` threw
	// `DataCloneError` every single time, because `configResponse` is a
	// Svelte 5 `$state` value and `.config` is therefore a deeply-reactive
	// Proxy — `structuredClone` (the DOM/Node structured-clone algorithm)
	// unconditionally rejects any Proxy, in every engine. setWebspaceFilter
	// clones via a JSON round trip internally (config-edit.ts's own
	// cloneConfig), which reads through a reactive Proxy exactly as a plain
	// property access would — this route no longer needs to call
	// `$state.snapshot()` itself.
	async function writeFilter(nextFilters: string[], clearSearchOnSuccess: boolean) {
		const gen = navGeneration;
		filterBusy = true;
		try {
			if (!configResponse) {
				// Reachable in the real world: showSaveAsFilter only reads
				// searchQuery/filters (filters defaults to [] when
				// configResponse is still null), so the button can render
				// and be clicked before the initial getConfig() has
				// resolved. A bare `return` here was the other silent exit
				// this fix closes.
				filterError = 'Config has not finished loading yet — try again in a moment.';
				return;
			}
			const nextConfig = setWebspaceFilter(configResponse.config, webspace, nextFilters);

			const res = await putConfig({ base_hash: configResponse.hash, config: nextConfig });
			if (gen !== navGeneration) return;
			configResponse = res;
			filterError = null;
			if (clearSearchOnSuccess) {
				searchQuery = '';
				searchState = 'idle';
				searchResults = [];
			}
			await load(navGeneration);
		} catch (err) {
			if (gen !== navGeneration) return;
			filterError =
				err instanceof ApiError && err.code === 'config_changed_on_disk'
					? CONFIG_CONFLICT_MESSAGE
					: err instanceof ApiError
						? err.message
						: 'Something went wrong saving the filter — check the browser console and try again.';
			await loadConfig(gen);
		} finally {
			if (gen === navGeneration) filterBusy = false;
		}
	}

	// saveFilter appends the trimmed search query to the end of the
	// existing filter array (never prepends, never dedupes silently — the
	// "Save as filter" affordance's own gating in WebspaceHeader.svelte
	// already prevents offering it for a duplicate term).
	async function saveFilter() {
		const term = searchQuery.trim();
		if (term === '') return;
		await writeFilter([...filters, term], true);
	}

	// removeFilter removes exactly one matching element by value, leaving
	// any others (and their stored order) intact.
	async function removeFilter(term: string) {
		await writeFilter(
			filters.filter((f) => f !== term),
			false
		);
	}

	async function loadSources() {
		try {
			const res = await getSources();
			sources = res.sources;
			sourcesState = 'ready';
			// WR-03: pick up an already-in-flight sync regardless of who
			// started it (the background scheduler, the `topos sync` CLI,
			// or another browser tab) — not only syncs this tab itself
			// kicked off via handleRefreshSource/handleRefreshAll below.
			// Without this, a source that's already syncing when the page
			// loads or the webspace route param changes never schedules a
			// poll, so its spinner can keep spinning after the sync
			// actually finishes.
			if (sources.some((s) => s.syncing)) ensurePolling();
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

	// Clear any in-flight poll interval on component teardown (e.g. the
	// user navigates away from the /w/[webspace] route tree entirely while
	// a sync is still running) — without this the interval keeps firing
	// loadSources() against a torn-down component's state until the
	// in-flight sync finishes on its own.
	$effect(() => {
		return () => {
			if (pollHandle !== null) clearInterval(pollHandle);
		};
	});

	async function handleRefreshSource(name: string) {
		ensurePolling();
		try {
			await refreshSource(name);
		} catch {
			// A failed refresh has no separate toast/dialog — the chip
			// itself is the error surface (D-08) once loadSources() below
			// picks up the newly recorded sync_runs failure.
		} finally {
			await Promise.all([loadSources(), load(navGeneration)]);
		}
	}

	async function handleRefreshAll() {
		ensurePolling();
		try {
			await refreshAll();
		} catch {
			// same as handleRefreshSource above.
		} finally {
			await Promise.all([loadSources(), load(navGeneration)]);
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
		const gen = navGeneration; // captured now so a webspace nav mid-flight also invalidates this response
		try {
			const res = await searchWebspace(webspace, query);
			// a newer search or a webspace navigation has since superseded this one
			if (seq !== searchRequestSeq || gen !== navGeneration) return;
			searchResults = res.results;
			searchState = 'ready';
		} catch {
			if (seq !== searchRequestSeq || gen !== navGeneration) return;
			searchResults = [];
			searchState = 'error';
		}
	}

	// Re-fetch (and drop any stale selection/search) whenever the webspace
	// route param changes. writeLastWebspace records this visit (D-10,
	// 07-03-PLAN.md Task 3) so the root route's redirect always lands here
	// next time — every visit updates the memory, not only navigation via
	// the switcher.
	$effect(() => {
		const gen = ++navGeneration;
		selectedId = null;
		searchQuery = '';
		searchState = 'idle';
		searchResults = [];
		filterError = null;
		writeLastWebspace(webspace);
		load(gen);
		loadSources();
		loadConfig(gen);
	});
</script>

<svelte:head>
	<title>{webspace} — webspaces</title>
</svelte:head>

<div class="flex h-full min-h-0 flex-col">
	<WebspaceHeader
		{webspace}
		webspaces={webspaceNames}
		oncreatewebspace={() => (createOpen = true)}
		onmanagesources={handleManageSources}
		{sources}
		{sourcesState}
		{selectedSources}
		onfilter={toggleFilter}
		onclearfilters={clearFilters}
		onrefresh={handleRefreshSource}
		onrefreshall={handleRefreshAll}
		{searchQuery}
		onsearch={handleSearch}
		{filters}
		{filterBusy}
		{filterError}
		{unknownConfigKeys}
		onsavefilter={saveFilter}
		onremovefilter={removeFilter}
		config={configResponse?.config ?? null}
		baseHash={configResponse?.hash ?? ''}
		{pluginTypes}
		envVars={configResponse?.env_vars ?? {}}
		onsourceadded={handleSourceAdded}
		onedit={handleChipEdit}
	/>

	{#if configResponse}
		<CreateWebspaceModal
			open={createOpen}
			config={configResponse.config}
			baseHash={configResponse.hash}
			onclose={() => (createOpen = false)}
			oncreated={handleWebspaceCreated}
		/>
	{/if}

	{#if configResponse && editInstance}
		{#key `${editInstance}-${editMode}`}
			<EditSourceModal
				open={editOpen}
				mode={editMode}
				instance={editInstance}
				{webspace}
				config={configResponse.config}
				baseHash={configResponse.hash}
				envVars={configResponse.env_vars}
				vocabulary={editVocabulary}
				onclose={handleEditClose}
				onsaved={handleEditSaved}
			/>
		{/key}
	{/if}

	{#if configResponse}
		<ManageSourcesModal
			open={manageOpen}
			config={configResponse.config}
			baseHash={configResponse.hash}
			envVars={configResponse.env_vars}
			currentWebspace={webspace}
			onclose={() => (manageOpen = false)}
			onchanged={handleManageSourcesChanged}
		/>
	{/if}

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

		  Wrapped in a `relative` container (UI-11) so StreamDateMarkers
		  can render as an absolutely-positioned sibling of the actual
		  scroll region below, pinned to the pane's full height rather
		  than scrolling along with its content. The scroll div's own
		  overflow/scroll classes are unchanged; the conditional width
		  moves to this wrapper since it -- not the scroll div -- is now
		  the flex child `main` sizes.
		-->
		<div class="relative min-h-0 min-w-0 {selectedItem ? 'w-[480px] shrink-0' : 'flex-1'}">
			<!-- pr-6 (24px) below reserves a right gutter for
			     StreamDateMarkers' own lane (UI-11 gap closure G-06-6).
			     Load-bearing, not cosmetic: without it the marker lane
			     sits on top of the row list, which alternates between
			     the card surface and the background in the gaps
			     between rows -- the ruler would composite against two
			     different tones banding down its length, the same
			     "two tones across its own width" defect this gap
			     closure exists to fix. Do not reclaim this padding. -->
			<div
				bind:this={streamScrollEl}
				bind:clientHeight={streamScrollHeight}
				class="h-full min-h-0 min-w-0 overflow-x-hidden overflow-y-auto pr-6 {selectedItem
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
						onretry={() => load(navGeneration)}
						staleSources={staleInstances}
						{selectedSources}
						{sourcesByInstance}
					/>
				{/if}
			</div>
			{#if !searchQuery.trim() && loadState === 'ready' && response}
				<!-- Gated behind the same condition that selects the
				     stream over search results, and behind the stream
				     having loaded successfully -- markers never render
				     over search results, a skeleton, an error state (the
				     'error' branch above sets response back to null), or
				     an empty stream (dateMarkers itself returns zero
				     markers for fewer than two items). -->
				<StreamDateMarkers
					items={visibleStreamItems}
					trackHeightPx={streamScrollHeight}
					scrollContainer={streamScrollEl}
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
