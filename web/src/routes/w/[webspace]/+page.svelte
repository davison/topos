<script lang="ts">
	import { untrack } from 'svelte';
	import { page } from '$app/state';
	import { goto, pushState } from '$app/navigation';
	import {
		getStream,
		getSources,
		refreshSource,
		refreshAll,
		searchWebspace,
		getConfig,
		putConfig,
		reloadConfig,
		listPluginTypes,
		describePlugin,
		setItemMarks,
		ApiError,
		CONFIG_CONFLICT_MESSAGE,
		type StreamResponse,
		type SourceStatus,
		type SearchResult,
		type SourceSearchStatus,
		type ConfigResponse,
		type ExtrasFieldDecl
	} from '$lib/api';
	import {
		resolveSourceFilters,
		toggleSourceFilter,
		serializeSourceFilters,
		staleSources,
		filterItemsBySource
	} from '$lib/format';
	import { setWebspaceFilter, removeSourceFromWebspace } from '$lib/config-edit';
	import { markSuccessToast, markFailureToast } from '$lib/toast';
	import { toggleSelection, selectRange, clearSelection } from '$lib/selection';
	import WebspaceHeader from '$lib/components/WebspaceHeader.svelte';
	import StreamList from '$lib/components/StreamList.svelte';
	import StreamDateMarkers from '$lib/components/StreamDateMarkers.svelte';
	import SearchResults from '$lib/components/SearchResults.svelte';
	import DetailPane from '$lib/components/DetailPane.svelte';
	import CreateWebspaceModal from '$lib/components/CreateWebspaceModal.svelte';
	import EditSourceModal from '$lib/components/EditSourceModal.svelte';
	import RelinkModal from '$lib/components/RelinkModal.svelte';
	import TrustUpdateDialog from '$lib/components/TrustUpdateDialog.svelte';
	import TrustKeyDialog from '$lib/components/TrustKeyDialog.svelte';
	import ManageSourcesModal from '$lib/components/ManageSourcesModal.svelte';
	import { writeLastWebspace } from '$lib/last-webspace';

	// The [webspace] dynamic segment always matches for this route, so
	// page.params.webspace is never actually undefined at runtime — the
	// fallback only satisfies the type checker.
	let webspace = $derived(page.params.webspace ?? '');

	let response = $state<StreamResponse | null>(null);
	// view (13-UI-SPEC.md E4, KERN-10): which stream bucket is currently
	// showing — 'included' (the normal stream, the default) or 'excluded'
	// (the excluded-items view toggle). Threaded into load()'s own
	// getStream call and reset to 'included' in the per-webspace effect
	// below, alongside every other transient per-webspace UI state.
	let view = $state<'included' | 'excluded'>('included');
	// Named loadState, not `state` — a local variable literally named
	// `state` collides with the `$state()` rune's auto-subscription
	// parsing (Svelte tries to treat `$state(...)` as a store
	// subscription to a variable named `state`, causing a "used before
	// its declaration" compiler error).
	let loadState: 'loading' | 'error' | 'not-found' | 'ready' = $state('loading');
	let selectedId = $state<string | null>(null);

	// Mobile takeover history (D-01/D-03/D-04, 09.1-01-PLAN.md Task 1):
	// takeoverActive mirrors whether the current history entry carries the
	// shallow-routing marker pushed below. selectItem/closeDetail are the
	// only two call sites that touch it — both the visible back control
	// and the phone's back button converge on closeDetail's
	// `history.back()` path (planner_resolutions R3), so there is exactly
	// one popstate transition, not two.
	let takeoverActive = $derived(page.state.itemOpen === true);

	function selectItem(id: string) {
		selectedId = id;
		// Guarded for a non-browser environment even though this app is
		// SPA-only: page.state is empty before the client router mounts,
		// so nothing here may assume a window exists yet. The
		// `!takeoverActive` guard is "one history entry per takeover
		// session, not per item" — selecting a second item from inside an
		// already-open takeover reassigns the selection without stacking
		// a second history entry.
		if (
			typeof window !== 'undefined' &&
			window.matchMedia('(max-width: 767px)').matches &&
			!takeoverActive
		) {
			pushState('', { itemOpen: true });
		}
	}

	function closeDetail() {
		if (takeoverActive) {
			history.back();
		} else {
			selectedId = null;
		}
	}

	// markBusy/handleExclude/handleInclude (KERN-09/KERN-10, 13-01-PLAN.md
	// Tasks 1 and 3): the detail-pane single-item exclude/include write,
	// the exact mirror of each other. markBusy disables the control for
	// the duration of the request (prevents a double-fire); on success
	// the detail pane closes (an excluded item can no longer be shown
	// from the normal stream, D-03 — and today's DetailPane render site
	// only ever wires onexclude, never oninclude, since the excluded
	// view a later plan adds is the only place `excluded` is ever true)
	// and the stream refetches so the change is visible on the very next
	// render; the success/failure toast fires via the shared toast.ts
	// helpers, so this and the later bulk action-bar path can never drift
	// in wording (E3.1). ws/gen (13-07-PLAN.md, WR-01 gap closure) are
	// captured at entry because markSuccessToast's Undo action can fire up
	// to 5000ms later, by which time a WebspaceSwitcher navigation may
	// have changed the route's derived webspace/navGeneration bindings —
	// the same discipline writeFilter and handleSearch already apply for
	// their own, shorter, delay windows. Passing the snapshotted `gen` to
	// `load` from a callback firing this late is safe BECAUSE of load's
	// own entry guard (13-08-PLAN.md, G-13-1 gap closure, below) — a
	// generation that is already stale by the time onUndo fires makes
	// `load` a true no-op rather than stranding the navigated-to
	// webspace's stream in a permanent loading state.
	let markBusy = $state(false);

	async function handleExclude(id: string) {
		const ws = webspace;
		const gen = navGeneration;
		markBusy = true;
		try {
			await setItemMarks(ws, 'add', [id]);
			closeDetail();
			await load(gen);
			markSuccessToast({
				verb: 'Excluded',
				count: 1,
				onUndo: async () => {
					await setItemMarks(ws, 'remove', [id]);
					await load(gen);
				}
			});
		} catch {
			markFailureToast({ verb: 'exclude', count: 1 });
		} finally {
			markBusy = false;
		}
	}

	async function handleInclude(id: string) {
		const ws = webspace;
		const gen = navGeneration;
		markBusy = true;
		try {
			await setItemMarks(ws, 'remove', [id]);
			closeDetail();
			await load(gen);
			markSuccessToast({
				verb: 'Included',
				count: 1,
				onUndo: async () => {
					await setItemMarks(ws, 'add', [id]);
					await load(gen);
				}
			});
		} catch {
			markFailureToast({ verb: 'include', count: 1 });
		} finally {
			markBusy = false;
		}
	}

	// Bulk selection (13-UI-SPEC.md E1/E2, KERN-09/KERN-10's desktop-standard
	// multi-select) — bulkSelection is the ONLY mode indicator (SelectionActionBar
	// renders whenever it's non-empty; a separate mode flag would be a second
	// source of truth, per this plan's own key_links). bulkAnchor tracks the
	// most recently ctrl-clicked row, so a subsequent shift-click ranges from
	// there — reset to null only by handleBulkClear and the per-webspace
	// effect below, never implicitly by a plain click (D-01: a plain click on
	// an unrelated row must never clear an existing bulk selection).
	let bulkSelection = $state(new Set<string>());
	let bulkAnchor = $state<string | null>(null);
	let bulkBusy = $state(false);
	let bulkModeActive = $derived(bulkSelection.size > 0);

	// currentBulkAnchor resolves 13-UI-SPEC.md E1's full shift-click anchor
	// fallback chain: the most recently ctrl-clicked row, else the
	// currently-open item (no ctrl-click has happened yet this session),
	// else the first rendered row (neither exists) — never null while any
	// row is rendered, so a shift-click as the very first bulk-select
	// action still resolves to a sensible range instead of a single-id
	// fallback inside selectRange.
	function currentBulkAnchor(): string | null {
		if (bulkAnchor !== null) return bulkAnchor;
		if (selectedId !== null) return selectedId;
		return visibleStreamItems.length > 0 ? visibleStreamItems[0].id : null;
	}

	function handleBulkToggle(id: string, mode: 'toggle' | 'range') {
		if (mode === 'toggle') {
			bulkSelection = toggleSelection(bulkSelection, id);
			bulkAnchor = id;
			return;
		}
		const orderedIds = visibleStreamItems.map((item) => item.id);
		bulkSelection = selectRange(orderedIds, currentBulkAnchor(), id);
		bulkAnchor = id;
	}

	function handleBulkClear() {
		bulkSelection = clearSelection();
		bulkAnchor = null;
	}

	// handleBulkPrimary (D-02: no confirm dialog) mirrors handleExclude/
	// handleInclude above at N items instead of 1 — same busy-disable
	// discipline, same setItemMarks call, same shared toast helpers (so
	// the single-item and bulk paths can never drift in wording). action
	// is view-driven (13-UI-SPEC.md E2/E4): 'add' (Exclude) in the normal
	// stream, 'remove' (Include) while viewing the excluded bucket — the
	// same view value StreamList's own primaryLabel prop reads, so the
	// button's label and the write it fires can never disagree.
	async function handleBulkPrimary() {
		const ws = webspace;
		const gen = navGeneration;
		const ids = [...bulkSelection];
		const count = ids.length;
		if (count === 0) return;
		const excludedView = view === 'excluded';
		const action = excludedView ? 'remove' : 'add';
		const undoAction = excludedView ? 'add' : 'remove';
		const verb: 'Excluded' | 'Included' = excludedView ? 'Included' : 'Excluded';
		const failureVerb: 'exclude' | 'include' = excludedView ? 'include' : 'exclude';
		bulkBusy = true;
		try {
			await setItemMarks(ws, action, ids);
			handleBulkClear();
			await load(gen);
			markSuccessToast({
				verb,
				count,
				onUndo: async () => {
					await setItemMarks(ws, undoAction, ids);
					await load(gen);
				}
			});
		} catch {
			markFailureToast({ verb: failureVerb, count });
		} finally {
			bulkBusy = false;
		}
	}

	// handleToggleView (13-UI-SPEC.md E4): flips the stream's own bucket,
	// clearing both the bulk selection and the open item (an item open
	// from one bucket has no meaning once the OTHER bucket is showing —
	// mirrors the per-webspace effect's own reset discipline) before
	// refetching through the same load() path every other webspace fetch
	// uses, so the toggle inherits the identical loading/error treatment.
	async function handleToggleView() {
		view = view === 'excluded' ? 'included' : 'excluded';
		handleBulkClear();
		selectedId = null;
		await load(navGeneration);
	}

	// Esc clears the bulk selection (D-01: Esc and the action bar's own
	// Clear button are the only two paths that empty the selection) —
	// registered globally rather than scoped to a row/bar element, since
	// the action bar itself may not have focus when the user presses Esc.
	$effect(() => {
		function handleGlobalKeydown(event: KeyboardEvent) {
			if (event.key === 'Escape' && bulkSelection.size > 0) {
				handleBulkClear();
			}
		}
		window.addEventListener('keydown', handleGlobalKeydown);
		return () => window.removeEventListener('keydown', handleGlobalKeydown);
	});

	// Deliberately a plain `let`, NOT `$state` — writing it inside the
	// effect below must not retrigger that same effect. This open-to-
	// closed edge detector is what keeps a closed takeover and a
	// deselected stream from becoming two states that can drift apart. It
	// must NOT clear the selection on a plain "not open" read: at desktop
	// width `itemOpen` is always absent while a selection is legitimately
	// set.
	let wasTakeoverOpen = false;
	$effect(() => {
		const isOpen = page.state.itemOpen === true;
		if (wasTakeoverOpen && !isOpen) {
			selectedId = null;
		}
		wasTakeoverOpen = isOpen;
	});

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
	// The source fan-out's half of a search (M2-R2, #54): progressive as two
	// requests — the index answer lands first and is shown at once; the
	// scope=all answer replaces it (the kernel merges by stable id) and
	// brings the per-source status map. A slow source can only ever delay
	// the second answer.
	let searchSources = $state<Record<string, SourceSearchStatus> | null>(null);
	let searchSourcesState: 'idle' | 'pending' | 'ready' | 'error' = $state('idle');
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

	// pluginTypeTiers (Phase 11, PLUG-06/08): GET /api/config/plugin-types'
	// own plugin_type_tiers lookup table, fetched alongside pluginTypes
	// above and held here for the same lifetime — defaults to an empty
	// object on failure, exactly like pluginTypes defaults to an empty
	// array (loadPluginTypes below). Threaded through WebspaceHeader to
	// AddSourceModal so its picker rows and untrusted-confirm step can key
	// off tier without a second network call.
	let pluginTypeTiers = $state<Record<string, string>>({});

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
	//
	// CR-02 (07-REVIEW.md, closed by 07-08-PLAN.md): the render site below
	// keys EditSourceModal on `${editInstance}-${editMode}`, so reopening
	// the SAME source in the SAME mode produces an identical key and Svelte
	// will not remount the component — only dropping editInstance makes the
	// enclosing `{#if configResponse && editInstance}` guard false, which
	// destroys the modal subtree outright. A destroyed subtree always
	// remounts fresh on the next open, regardless of what the `{#key}`
	// expression evaluates to; that is the whole mechanism. Without this,
	// the modal's form state — seeded once at mount — would survive from a
	// previous, abandoned edit session (anything typed and then Cancelled),
	// and a later save on the reopened session would write that stale
	// value straight over the real connection config. resetEditSession
	// below is the ONE place this session is cleared — every close path
	// (Cancel, the dialog's own outside-click/Escape close, and a
	// successful save) routes through it.
	let editOpen = $state(false);
	let editMode = $state<'connection' | 'match'>('connection');
	let editInstance = $state<string | null>(null);
	let editVocabulary = $state<string[]>([]);
	// editExtrasFields (Phase 11, PLUG-09, D-15): this instance's plugin's
	// declared expected extras keys, resolved via the SAME describePlugin
	// substitute editVocabulary already uses (GET /api/sources carries
	// neither field) — fetched for BOTH 'connection' and 'match' modes now
	// (widened from editVocabulary's 'match'-only fetch), since the extras
	// form (E6) renders inside ConnectionForm, reached only from
	// 'connection' mode. Safe to pass [] when the fetch hasn't resolved
	// yet or failed: the free-form editor still shows an instance's
	// already-saved extras values either way.
	let editExtrasFields = $state<ExtrasFieldDecl[]>([]);

	// relinkInstance (D-03, 08-04-PLAN.md Task 2) is tracked in its OWN
	// state value, deliberately never overloading editInstance/editMode —
	// a Re-link session opens a structurally different modal (RelinkModal,
	// no describePlugin call, no stale-response guard) and must never
	// collide with an in-progress Edit connection…/Edit match settings…
	// session over the same shared state, the exact class of bug the
	// stale-response guard above exists to prevent for the describe race.
	let relinkInstance = $state<string | null>(null);

	// trustUpdateInstance (11-06-PLAN.md Task 2, 11-UI-SPEC.md E4) mirrors
	// relinkInstance's own shape exactly — its own state slot, never
	// overloaded onto editInstance/editMode, since TrustUpdateDialog is
	// another structurally different per-chip modal with no describePlugin
	// call of its own. Resolved to a real SourceStatus via trustUpdateSource
	// (declared below, alongside sourcesByInstance — GET /api/sources' own
	// live pinned_hash/current_hash facts; TrustUpdateDialog never derives
	// or computes a hash itself).
	let trustUpdateInstance = $state<string | null>(null);
	// trustKeyInstance/trustKeyMode (M2-R4): the one key-consent session —
	// trust an offered key, or stop trusting the key that vouches.
	let trustKeyInstance = $state<string | null>(null);
	let trustKeyMode = $state<'trust' | 'untrust'>('trust');

	// The single edit-session reset site (mirrors AddSourceModal.svelte's
	// own single resetFlowState) — handleEditClose and handleEditSaved both
	// call this and do no clearing of their own. Clearing editInstance
	// unmounts EditSourceModal immediately, with no exit transition; that
	// is intentional and matches ManageSourcesModal's own edit-modal close
	// behavior — do not add a delay or transition-end hook here, since that
	// would reopen a window where the stale component is still mounted.
	function resetEditSession() {
		editOpen = false;
		editInstance = null;
		editMode = 'connection';
		editVocabulary = [];
		editExtrasFields = [];
	}

	async function handleChipEdit(
		name: string,
		kind:
			| 'connection'
			| 'match'
			| 'relink'
			| 'remove'
			| 'trust-update'
			| 'trust-key'
			| 'untrust-key'
	) {
		if (kind === 'trust-key' || kind === 'untrust-key') {
			// M2-R4 (davison/topos#49): the key dialogs mirror the re-pin
			// session — one instance, resolved live to its SourceStatus, and
			// a single putConfig on confirm.
			trustKeyInstance = name;
			trustKeyMode = kind === 'trust-key' ? 'trust' : 'untrust';
			return;
		}
		if (kind === 'remove') {
			await handleRemoveSource(name);
			return;
		}
		if (kind === 'relink') {
			// Branches before the describe path below on purpose — Re-link
			// opens a different modal and needs no describePlugin call,
			// and therefore no stale-response guard.
			relinkInstance = name;
			return;
		}
		if (kind === 'trust-update') {
			// Branches before the describe path below, exactly like
			// 'relink' above (11-06-PLAN.md Task 2, E4) — re-establishing
			// trust needs no describePlugin call either, and therefore no
			// stale-response guard.
			trustUpdateInstance = name;
			return;
		}
		if (!configResponse) return;
		editInstance = name;
		editMode = kind;
		editVocabulary = [];
		editExtrasFields = [];
		if (kind === 'match') {
			try {
				const source = configResponse.config.sources[name];
				const resp = await describePlugin({ plugin: source.plugin, source });
				// Stale-response guard (07.1-05-PLAN.md, closes 07-UAT.md
				// item 10's "carried advisory" — the chip-edit describePlugin
				// race). editVocabulary/editInstance/editMode are single
				// shared $state values: without this check, a SLOWER click's
				// describePlugin response resolving AFTER a FASTER, later
				// click has already opened a different chip's edit session
				// would land here and overwrite that session's already-
				// showing vocabulary — reverting the visibly-open modal back
				// to the first chip's fields even though the user is now
				// looking at the second chip's. Discarding a response whose
				// own (name, kind) no longer matches the CURRENT edit
				// session is what makes "the second click's state always
				// wins" true both immediately and after the slower response
				// lands, not merely at the moment of the second click.
				if (editInstance !== name || editMode !== kind) return;
				editVocabulary = resp.match_vocabulary;
				// ?? [] guards a describe response that omits `extras`
				// entirely — editExtrasFields must never become anything
				// but an array, since extrasKeyError reads it unconditionally.
				editExtrasFields = resp.extras ?? [];
			} catch {
				// Match settings can still be viewed/edited against whatever
				// vocabulary resolved (possibly none) — a describe failure
				// here is not fatal, it just means the form renders no
				// fields until the instance can connect again. Same
				// staleness guard as the success path above — a failure
				// resolving after the session has moved on must not force
				// editOpen on the now-current (different) session either.
				if (editInstance !== name || editMode !== kind) return;
			}
			editOpen = true;
			return;
		}
		// 'connection' mode opens immediately, unchanged from before Phase
		// 11 (no describePlugin call ever blocked this open) — the extras
		// declarations fetch below is fire-and-forget and must never delay
		// this modal's own open transition. ConnectionForm's own effect
		// (keyed on extrasFields' identity) picks up the response whenever
		// it lands; the same staleness guard as the match branch above
		// applies inside loadEditExtrasFields.
		editOpen = true;
		void loadEditExtrasFields(name, kind);
	}

	// loadEditExtrasFields (Phase 11, PLUG-09, D-15): the 'connection' mode
	// half of the widening above, split into its own function purely so it
	// can run without blocking editOpen — 'connection' mode has always
	// opened synchronously and this must not change that.
	async function loadEditExtrasFields(name: string, kind: 'connection' | 'match') {
		if (!configResponse) return;
		try {
			const source = configResponse.config.sources[name];
			const resp = await describePlugin({ plugin: source.plugin, source });
			if (editInstance !== name || editMode !== kind) return;
			editExtrasFields = resp.extras ?? [];
		} catch {
			if (editInstance !== name || editMode !== kind) return;
			// Silent — the free-form editor still shows an instance's
			// already-saved extras values even with zero declarations known.
		}
	}

	function handleEditClose() {
		resetEditSession();
	}

	async function handleEditSaved() {
		resetEditSession();
		await Promise.all([loadConfig(navGeneration), loadSources(), load(navGeneration)]);
	}

	// handleRelinkClose/handleRelinked (D-03): relinkInstance is the ONE
	// place a Re-link session is cleared — RelinkModal's own onclose fires
	// on every close path (Escape, outside click, paired, and an explicit
	// cancel from inside QRPanel), and onrelinked fires additionally once
	// paired, so the caller refreshes the same source-health state a
	// config change already refreshes elsewhere in this file (D-07's
	// eager reconcile) — the chip's own health dot updates in place, no
	// page reload.
	function handleRelinkClose() {
		relinkInstance = null;
	}

	async function handleRelinked() {
		await Promise.all([loadConfig(navGeneration), loadSources(), load(navGeneration)]);
	}

	// handleTrustUpdateClose/handleTrustUpdateSaved (11-06-PLAN.md Task 2,
	// E4): trustUpdateInstance is the ONE place a re-pin session is
	// cleared, mirroring handleRelinkClose/handleRelinked exactly.
	// TrustUpdateDialog itself calls onclose() on Cancel/outside-click/
	// Escape and onsaved() (which also clears the instance) once the
	// single putConfig write succeeds — refreshing config (the new pin),
	// sources (the chip's own health/hash fields) and the stream (D-07's
	// eager reconcile) together, so the chip recovers without a kernel
	// restart.
	function handleTrustKeyClose() {
		trustKeyInstance = null;
	}
	async function handleTrustKeySaved() {
		trustKeyInstance = null;
		await Promise.all([loadConfig(navGeneration), loadSources(), load(navGeneration)]);
	}
	function handleTrustUpdateClose() {
		trustUpdateInstance = null;
	}

	async function handleTrustUpdateSaved() {
		trustUpdateInstance = null;
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

	// Reload-config state (09-06-PLAN.md Task 2, 09-UI-SPEC.md Fix 7): the
	// route now owns the reload call outright, mirroring how it already
	// owns oncreatewebspace (createOpen)/onmanagesources (manageOpen) above
	// — ManageSourcesModal.svelte no longer holds its own reloading/
	// reloadError state or Reload-config control; this is the one
	// remaining entry point (WebspaceSwitcher's menu-root item). reloadBusy
	// guards re-entry here (a second, route-owned guard alongside the menu
	// item's own disabled={reloadBusy} — T-09-20 requires both, since the
	// action tears down and relaunches every plugin subprocess).
	let reloadBusy = $state(false);
	let reloadError = $state<string | null>(null);

	async function handleReload() {
		if (reloadBusy) return;
		reloadBusy = true;
		try {
			const res = await reloadConfig();
			// Adopt the returned config/hash into the SAME state
			// getConfig()/putConfig() already populate, so a hand-edited
			// config.toml becomes visible without a page reload — then
			// refresh sources/the stream through the identical paths every
			// other config-changing handler already uses (D-07's eager
			// reconcile, same shape as handleSourceAdded/
			// handleManageSourcesChanged above).
			configResponse = res;
			reloadError = null;
			await Promise.all([loadSources(), load(navGeneration)]);
		} catch (err) {
			// Copywriting Contract (09-UI-SPEC.md, "Reload-config failure"):
			// the kernel's verbatim message followed by the fixed
			// reassurance sentence — reloadConfig() itself leaves the
			// kernel's previously running configuration completely
			// untouched on failure (api.ts's own doc comment), so this is
			// purely a user-facing report, not a rollback the UI performs.
			const detail = err instanceof ApiError ? err.message : 'check the browser console and try again';
			reloadError = `${detail}. The previous configuration is still running.`;
		} finally {
			reloadBusy = false;
		}
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
	// trustUpdateSource (11-06-PLAN.md Task 2, E4): resolves trustUpdateInstance
	// to its live SourceStatus via the same sourcesByInstance map every other
	// per-instance lookup already uses — declared here, after
	// sourcesByInstance, so its own $derived reads an already-initialized
	// value rather than one still in its declaration's temporal dead zone.
	let trustKeySource = $derived(
		trustKeyInstance ? (sourcesByInstance.get(trustKeyInstance) ?? null) : null
	);
	let trustUpdateSource = $derived(
		trustUpdateInstance ? (sourcesByInstance.get(trustUpdateInstance) ?? null) : null
	);

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

	// Auto-collapsing header (checkpoint deviation, 09.1-01-PLAN.md issue
	// 2, approved live during the human-verify checkpoint — not in the
	// plan's original must_haves): below 1024px, WebspaceHeader collapses
	// to a slim bar on scroll-down and re-inflates the instant scroll-up
	// begins, the standard mobile pattern. Hooked to the STREAM's own
	// scroll container (streamScrollEl, bound above) — this app has no
	// window-level scrolling (the stream and detail panes each own an
	// independent internal scroll region), so a window scroll listener
	// would never fire. DetailPane's own internal scroll (the reading
	// surface, at 768-1024px side-by-side) is a deliberately out-of-scope
	// follow-up: DetailPane exposes no scroll-container ref today, and
	// wiring one is a larger change than this checkpoint fix warrants —
	// recorded in 09.1-01-SUMMARY.md, not silently dropped.
	let headerCollapsed = $state(false);
	// Plain `let`, not $state: written from inside handleStreamScroll on
	// every scroll event, and must never itself be a reactive trigger —
	// only headerCollapsed (derived FROM this comparison) is the signal
	// WebspaceHeader reacts to.
	let lastStreamScrollTop = 0;
	// Below this scrollTop, the header always stays expanded — matches
	// most mobile browser chrome's own behaviour (pull only starts
	// collapsing after the content has genuinely scrolled away from the
	// top) and avoids a jittery collapse from a 1-2px rubber-band bounce
	// at rest.
	const HEADER_COLLAPSE_SCROLL_THRESHOLD = 24;
	// Matches WebspaceHeader's own max-lg:duration-200 collapse
	// transition exactly.
	const HEADER_COLLAPSE_TRANSITION_MS = 200;
	// A re-entrancy guard, not merely a nicety: toggling headerCollapsed
	// changes WebspaceHeader's rendered height, which changes <main>'s
	// available height, which changes THIS SAME scroll container's own
	// clientHeight mid-transition — confirmed live to fire spurious
	// native 'scroll' events on this exact element as a pure reflow side
	// effect (no user input at all), with scrollTop transiently snapping
	// toward 0. Without this guard, that transient read straight into the
	// "near top -> expand" rule below, re-expanding the header, growing
	// clientHeight back, and repeating indefinitely — a self-triggering
	// collapse/expand loop that also visibly corrupted the stream's own
	// scroll position. Suppressing scroll handling for the duration of
	// our OWN transition breaks the loop at its source; any genuine user
	// scroll during that ~200ms window is simply picked up by the next
	// event once the guard clears.
	let suppressScrollHandling = false;
	let suppressScrollHandlingTimer: ReturnType<typeof setTimeout> | null = null;

	function setHeaderCollapsed(next: boolean) {
		if (headerCollapsed === next) return;
		// Captured BEFORE the toggle: confirmed live that collapsing or
		// expanding the header — which changes <main>'s (and therefore
		// this SAME scroll container's) own clientHeight — makes the
		// browser itself reset scrollTop to 0 as a direct consequence of
		// that reflow, independent of any CSS transition (reproduced
		// with the collapse transition removed entirely). This is not a
		// bug our own scroll-direction logic can prevent by being more
		// careful; it is the browser's own resize-driven scroll
		// invalidation. Restoring the captured value after the reflow
		// settles (below) is what actually preserves D-01's guarantee
		// through a header collapse/expand, not merely through the
		// takeover open/close this plan's Task 1 already covers.
		const preToggleScrollTop = streamScrollEl?.scrollTop ?? 0;
		headerCollapsed = next;
		// Two nested rAFs: the first lets the browser complete the
		// layout pass this class change triggers (and, per the above,
		// its own scrollTop reset within that same pass); the second
		// re-asserts our captured value AFTER that reset has already
		// happened, so this write wins the race instead of being
		// silently overwritten by it.
		requestAnimationFrame(() => {
			requestAnimationFrame(() => {
				if (streamScrollEl) streamScrollEl.scrollTop = preToggleScrollTop;
			});
		});
		suppressScrollHandling = true;
		if (suppressScrollHandlingTimer !== null) clearTimeout(suppressScrollHandlingTimer);
		suppressScrollHandlingTimer = setTimeout(() => {
			suppressScrollHandling = false;
			suppressScrollHandlingTimer = null;
			if (streamScrollEl) {
				// Never stay collapsed once the reclaimed space means
				// there is nothing left to scroll — a short stream (few
				// items) can legitimately fit entirely once the header
				// shrinks, and with nothing left to scroll, no further
				// 'scroll' event will EVER fire to correct a stuck
				// collapsed state (confirmed live: without this check,
				// collapsing could permanently pin the header collapsed
				// the instant it made the stream fully fit). Collapsing
				// exists to reveal more content, not to hide the header
				// when there's no more content to reveal.
				if (streamScrollEl.scrollHeight <= streamScrollEl.clientHeight) {
					headerCollapsed = false;
				}
				// Re-baseline against wherever the reflow actually left
				// the scroll position, so the FIRST post-guard event
				// compares against reality rather than the
				// pre-transition value.
				lastStreamScrollTop = streamScrollEl.scrollTop;
			}
		}, HEADER_COLLAPSE_TRANSITION_MS);
	}

	function handleStreamScroll() {
		if (!streamScrollEl || suppressScrollHandling) return;
		const top = streamScrollEl.scrollTop;
		if (top <= HEADER_COLLAPSE_SCROLL_THRESHOLD) {
			setHeaderCollapsed(false);
		} else if (top > lastStreamScrollTop) {
			setHeaderCollapsed(true);
		} else if (top < lastStreamScrollTop) {
			// Re-inflate the instant an upward scroll begins — no
			// threshold, no debounce, matching the user's own request
			// ("re-inflate as soon as a scroll up event begins").
			setHeaderCollapsed(false);
		}
		lastStreamScrollTop = top;
	}

	// quiet (07-16-PLAN.md Task 3, closing 07-UAT.md G-07-7's residual
	// case): defaults to false for every existing call site below — the
	// webspace-keyed effect, Retry, the save paths, the refresh handlers
	// and the remove handler — none of which change behaviour. Only
	// ensurePolling's sync-completion refetch passes quiet: true.
	//
	// A background sync (the eager resync a config apply dispatches, or a
	// source's own regular scheduled tick) is exactly where a
	// de-participated pair's rows get purged when the synchronous purge in
	// Supervisor.Apply could not reach them (an eager resync that failed at
	// the time of the save). Until now nothing refetched the stream when
	// that background sync finished, so a stale view could persist until a
	// manual refresh; it is also a general freshness win for a sync that
	// ADDS items. Quiet mode is what makes firing this on every sync
	// completion safe: it skips the loading assignment below, so a
	// background refetch can never replace an already-rendered stream with
	// the loading skeleton (StreamList.svelte renders that skeleton purely
	// off `state === 'loading'`), and on failure it returns without
	// touching either response or loadState, so a failed background
	// refresh leaves whatever is on screen exactly as it was — strictly
	// better than today's behaviour of never refreshing at all, never
	// worse.
	//
	// Entry guard (13-08-PLAN.md, G-13-1 gap closure): a `gen` that is
	// already stale AT ENTRY — before the `quiet` destructuring, before
	// the `loadState = 'loading'` write — now makes `load` a true no-op:
	// no state write, no request. This is what makes it safe for a
	// DEFERRED callback (the undo toast's `onUndo`, which snapshots its
	// generation at mark time per WR-01 above) to call `load` with a
	// generation the user has since navigated away from. Before this
	// guard existed, the ordering was backwards: `loadState = 'loading'`
	// was written FIRST, and only then did the post-await guard below
	// check staleness — but that guard's only job is to discard a STALE
	// RESPONSE, so it returns past every exit that could ever restore
	// `'ready'` or `'error'`. A stale-generation call was therefore a
	// one-way transition into a permanently-rendered skeleton (four rows,
	// StreamLoadingSkeleton.svelte) until a page reload or "Refresh all"
	// forced a fresh `load`. The guard sits at the very top of the
	// function precisely so it holds for EVERY caller, present and
	// future, not only the three `onUndo` closures that exposed it. Both
	// post-await guards below are unchanged — they cover a DIFFERENT
	// race (a navigation that happens DURING an in-flight fetch, which
	// this entry check cannot see, since `gen` was still current when
	// the fetch started).
	async function load(gen: number, options?: { quiet?: boolean }) {
		if (gen !== navGeneration) return; // already stale at entry: no state write, no request
		const quiet = options?.quiet ?? false;
		if (!quiet) loadState = 'loading';
		try {
			const res = await getStream(webspace, view);
			if (gen !== navGeneration) return; // a newer webspace navigation has since superseded this one
			// Auto-flip (13-UI-SPEC.md E4): the excluded view emptied while
			// it was showing (the last item was un-excluded, undone, or
			// pruned elsewhere) — flip back to the normal stream and
			// refetch exactly once. Guarded structurally, not by a
			// counter: `view` is reset to 'included' BEFORE the recursive
			// call, so this branch can never be true on that recursive
			// call — the recursion terminates after at most one extra
			// fetch, never a loop.
			if (view === 'excluded' && res.items.length === 0) {
				view = 'included';
				await load(gen, options);
				return;
			}
			response = res;
			loadState = 'ready';
		} catch (err) {
			// The generation check runs first, exactly as it did before this
			// change (07-15-PLAN.md Task 2) — a stale response from a
			// superseded webspace navigation is still discarded before either
			// state below is ever set.
			if (gen !== navGeneration) return;
			if (quiet) return; // leave whatever is on screen untouched
			response = null;
			// A typed webspace_not_found answer from a healthy kernel is
			// classified apart from every other failure (no envelope, a
			// different code, a network failure): only the latter renders the
			// service-unreachable copy. This catch still wraps only the
			// request above — no processing is added here (07-12's Task 2
			// lesson: a catch that wraps more than the request itself turns
			// any downstream bug into a false outage report).
			loadState = err instanceof ApiError && err.code === 'webspace_not_found' ? 'not-found' : 'error';
		}
	}

	// Fetched once (not webspace-scoped, not re-fetched on navigation) —
	// the discovered plugin binary set is boot-time filesystem state.
	async function loadPluginTypes() {
		try {
			const res = await listPluginTypes();
			pluginTypes = res.plugin_types;
			pluginTypeTiers = res.plugin_type_tiers;
		} catch {
			pluginTypes = [];
			pluginTypeTiers = {};
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
	//
	// 07-16-PLAN.md Task 3: the stop branch — the tick where syncing falls
	// to false — now ALSO refetches the stream, quietly, once. This is
	// where a completed background sync (an eager resync a config apply
	// dispatched, or a source's own regular tick) is what the stream has
	// been waiting to reflect; the poll already fires regardless of who
	// started the sync (WR-03, above `loadSources`'s own comment), and this
	// extends that same "pick it up regardless of origin" property to the
	// stream, not only the chips. `gen` is captured before the tick's
	// first await (`loadSources()`), exactly like every other call path in
	// this file (`writeFilter`, `handleSearch`) — a webspace navigation
	// mid-tick still discards the stale refetch via load()'s own
	// generation check.
	let pollHandle: ReturnType<typeof setInterval> | null = null;
	function ensurePolling() {
		if (pollHandle !== null) return;
		pollHandle = setInterval(async () => {
			const gen = navGeneration;
			await loadSources();
			if (!sources.some((s) => s.syncing) && pollHandle !== null) {
				clearInterval(pollHandle);
				pollHandle = null;
				await load(gen, { quiet: true });
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
		//
		// state: page.state (09.1-01-PLAN.md backstop truth) — SvelteKit's
		// goto defaults an omitted `state` option to `{}`, which would
		// silently replace the current history entry's itemOpen flag with
		// nothing and close the mobile takeover as a side effect of
		// toggling a filter chip. Threading the CURRENT page.state through
		// keeps replaceState a pure URL-query update, exactly as its own
		// comment above already claims.
		goto(`${url.pathname}${url.search}`, {
			replaceState: true,
			keepFocus: true,
			noScroll: true,
			state: page.state
		});
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
			searchSources = null;
			searchSourcesState = 'idle';
			return;
		}

		searchState = 'loading';
		searchSources = null;
		searchSourcesState = 'pending';
		const seq = ++searchRequestSeq;
		const gen = navGeneration; // captured now so a webspace nav mid-flight also invalidates this response
		const live = () => seq === searchRequestSeq && gen === navGeneration;

		// Both requests leave together; whichever lands first is shown. The
		// index answer never overwrites a fuller scope=all answer that beat
		// it, and the scope=all answer always supersedes the index one.
		let allLanded = false;
		const indexHalf = searchWebspace(webspace, query, 'index').then(
			(res) => {
				if (!live() || allLanded) return;
				searchResults = res.results;
				searchState = 'ready';
			},
			() => {
				/* the scope=all answer, or its failure, decides */
			}
		);
		try {
			const res = await searchWebspace(webspace, query, 'all');
			if (!live()) return;
			allLanded = true;
			searchResults = res.results;
			searchSources = res.sources ?? {};
			searchState = 'ready';
			searchSourcesState = 'ready';
		} catch {
			if (!live()) return;
			allLanded = true;
			await indexHalf;
			if (!live()) return;
			if (searchState === 'ready') {
				// The index answered; only the fan-out failed — say so, keep the rows.
				searchSourcesState = 'error';
			} else {
				searchResults = [];
				searchState = 'error';
				searchSourcesState = 'idle';
			}
		}
	}

	// Re-fetch (and drop any stale selection/search) whenever the webspace
	// route param changes. writeLastWebspace records this visit (D-10,
	// 07-03-PLAN.md Task 3) so the root route's redirect always lands here
	// next time — every visit updates the memory, not only navigation via
	// the switcher.
	//
	// 13-03-PLAN.md Task 2 deviation (Rule 1 — bug): this effect's ONLY
	// intended reactive dependency is `webspace` (its own doc comment
	// above says so) — but `++navGeneration` both READS and WRITES
	// navGeneration inside the effect body, which makes Svelte's
	// dependency tracker treat navGeneration as a tracked source too.
	// Every OTHER call site that reads navGeneration (load(), handleBulk-
	// Primary(), handleToggleView(), …) does so from inside an event
	// handler, never inside a tracked reactive context, so those reads
	// alone can't explain a re-run — but confirmed live (13-excluded-
	// view.spec.ts) that this effect DOES re-run 1-2 extra times shortly
	// after a view-toggle interaction, each time resetting `view` back to
	// 'included' and re-issuing a stale-view fetch that clobbers the
	// freshly-toggled 'excluded' one moments after it lands. Wrapping
	// every read/write below in untrack() makes `webspace` the ONLY
	// tracked read, matching the doc comment's own stated intent, and
	// closes the clobber (verified: the toggle round trip is stable
	// across repeated runs after this fix).
	$effect(() => {
		const ws = webspace;
		untrack(() => {
			const gen = ++navGeneration;
			selectedId = null;
			searchQuery = '';
			searchState = 'idle';
			searchResults = [];
			searchSources = null;
			searchSourcesState = 'idle';
			filterError = null;
			headerCollapsed = false;
			lastStreamScrollTop = 0;
			// Bulk selection is per-webspace, transient UI state — never
			// persisted, never meaningful across a webspace switch.
			bulkSelection = clearSelection();
			bulkAnchor = null;
			// view (13-UI-SPEC.md E4): every fresh webspace visit starts
			// on the normal stream — the excluded view is never sticky
			// across a switch.
			view = 'included';
			if (suppressScrollHandlingTimer !== null) {
				clearTimeout(suppressScrollHandlingTimer);
				suppressScrollHandlingTimer = null;
			}
			suppressScrollHandling = false;
			writeLastWebspace(ws);
			load(gen);
			loadSources();
			loadConfig(gen);
		});
	});
</script>

<svelte:head>
	<title>{webspace} — topos</title>
</svelte:head>

<div class="flex h-full min-h-0 flex-col">
	<!-- D-04 (09.1-01-PLAN.md Task 1): max-md:invisible conceals the app
	     header while the mobile takeover is up. shrink-0 mirrors
	     WebspaceHeader's own root class so the surrounding flex column
	     behaves identically whether this wrapper is here or not. -->
	<div class="shrink-0 {selectedItem ? 'max-md:invisible' : ''}">
		<WebspaceHeader
			{webspace}
			webspaces={webspaceNames}
			oncreatewebspace={() => (createOpen = true)}
			onreload={handleReload}
			{reloadBusy}
			{reloadError}
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
			{pluginTypeTiers}
			envVars={configResponse?.env_vars ?? {}}
			onsourceadded={handleSourceAdded}
			onedit={handleChipEdit}
			collapsed={headerCollapsed}
			excludedCount={response?.excluded_count ?? 0}
			{view}
			ontoggleview={handleToggleView}
		/>
	</div>

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
				extrasFields={editExtrasFields}
				onclose={handleEditClose}
				onsaved={handleEditSaved}
			/>
		{/key}
	{/if}

	{#if configResponse && relinkInstance}
		{#key relinkInstance}
			<RelinkModal
				open={true}
				instance={relinkInstance}
				config={configResponse.config}
				onclose={handleRelinkClose}
				onrelinked={handleRelinked}
			/>
		{/key}
	{/if}

	{#if configResponse && trustKeyInstance && trustKeySource}
		{#key trustKeyInstance + trustKeyMode}
			<TrustKeyDialog
				open={true}
				mode={trustKeyMode}
				source={trustKeySource}
				config={configResponse.config}
				baseHash={configResponse.hash}
				onclose={handleTrustKeyClose}
				onsaved={handleTrustKeySaved}
			/>
		{/key}
	{/if}

	{#if configResponse && trustUpdateInstance && trustUpdateSource}
		{#key trustUpdateInstance}
			<TrustUpdateDialog
				open={true}
				source={trustUpdateSource}
				config={configResponse.config}
				baseHash={configResponse.hash}
				onclose={handleTrustUpdateClose}
				onsaved={handleTrustUpdateSaved}
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
		  absorbs viewport width changes (flex-1) at every band. The stream
		  pane's width, when an item is selected, now follows three bands
		  driven by the same `selectedItem` value that gates the detail
		  pane's rendering, so the panes can never disagree (D-02,
		  09.1-01-PLAN.md):
		    - below 768px: `max-md:invisible max-md:flex-1` -- the stream
		      keeps its layout box (so its scroll offset survives, D-01;
		      see planner_resolutions R1 -- `invisible`, never `hidden`)
		      but is visually concealed behind the detail pane's takeover;
		    - 768-1024px: `md:w-[clamp(240px,30vw,400px)] md:shrink-0` --
		      a proportional mid-band (RESEARCH Open Question 2);
		    - 1024px and up: `lg:` fixed 480 pixels -- today's exact fixed
		      width, byte-identical to the pre-phase layout.
		  With nothing selected there is no sibling to size against, so the
		  stream pane falls back to flex-1 at every band.

		  Wrapped in a `relative` container (UI-11) so StreamDateMarkers
		  can render as an absolutely-positioned sibling of the actual
		  scroll region below, pinned to the pane's full height rather
		  than scrolling along with its content. The scroll div's own
		  overflow/scroll classes are unchanged; the conditional width
		  moves to this wrapper since it -- not the scroll div -- is now
		  the flex child `main` sizes.

		  Positioning caveat (UI-SPEC): `fixed inset-0` on the detail
		  wrapper below resolves against the viewport only while no
		  ancestor establishes a containing block via transform, filter, or
		  perspective. None does today. A future phase adding one to an
		  ancestor of this route must give the takeover its own isolation
		  or move it to a portal.
		-->
		<div
			class="relative min-h-0 min-w-0 {selectedItem
				? 'max-md:invisible max-md:flex-1 md:w-[clamp(240px,30vw,400px)] md:shrink-0 lg:w-[480px]'
				: 'flex-1'}"
		>
			<!-- pr-6 (24px) below reserves a right gutter for
			     StreamDateMarkers' own lane (UI-11 gap closure G-06-6).
			     Load-bearing, not cosmetic: without it the marker lane
			     sits on top of the row list, which alternates between
			     the card surface and the background in the gaps
			     between rows -- the ruler would composite against two
			     different tones banding down its length, the same
			     "two tones across its own width" defect this gap
			     closure exists to fix. Do not reclaim this padding at
			     768px and above.

			     max-md:pr-0 (D-08): below 768px the ruler itself is not
			     rendered (StreamDateMarkers' own max-md:hidden) -- there
			     is no lane left to clear, so the gutter is reclaimed for
			     the compact row content instead of sitting idle. -->
			<div
				bind:this={streamScrollEl}
				bind:clientHeight={streamScrollHeight}
				onscroll={handleStreamScroll}
				class="h-full min-h-0 min-w-0 overflow-x-hidden overflow-y-auto pr-6 max-md:pr-0 {selectedItem
					? 'max-md:invisible max-md:flex-1 md:w-[clamp(240px,30vw,400px)] md:shrink-0 lg:w-[480px]'
					: 'flex-1'}"
			>
				{#if searchQuery.trim()}
					<SearchResults
						query={searchQuery}
						state={searchState}
						results={searchResults}
						sources={searchSources}
						sourcesState={searchSourcesState}
						{selectedId}
						onselect={selectItem}
						staleSources={staleInstances}
						{sourcesByInstance}
					/>
				{:else}
					<StreamList
						state={loadState}
						{response}
						{webspace}
						{selectedId}
						onselect={selectItem}
						onretry={() => load(navGeneration)}
						staleSources={staleInstances}
						{selectedSources}
						{sourcesByInstance}
						bulkSelected={bulkSelection}
						{bulkModeActive}
						onbulktoggle={handleBulkToggle}
						primaryLabel={view === 'excluded' ? 'Include' : 'Exclude'}
						{bulkBusy}
						onbulkprimary={handleBulkPrimary}
						onbulkclear={handleBulkClear}
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
			<div
				class="fixed inset-0 z-20 flex flex-col overflow-hidden bg-background md:static md:inset-auto md:z-auto md:min-w-0 md:flex-1 md:border-l md:border-border md:pl-8"
			>
				<DetailPane
					item={selectedItem}
					displayName={sourcesByInstance.get(selectedItem.source)?.display_name ??
						selectedItem.source_display_name ??
						selectedItem.source}
					sourceReachable={sourcesByInstance.get(selectedItem.source)?.reachable ?? true}
					{searchQuery}
					onback={closeDetail}
					onexclude={() => handleExclude(selectedItem.id)}
					oninclude={() => handleInclude(selectedItem.id)}
					{markBusy}
					excluded={view === 'excluded'}
				/>
			</div>
		{/if}
	</main>
</div>
