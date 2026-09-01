<script lang="ts">
	import { observeResize } from '$lib/resize-observer';
	import SourceChip from './SourceChip.svelte';
	import SearchBox from './SearchBox.svelte';
	import FilterChip from './FilterChip.svelte';
	import WebspaceSwitcher from './WebspaceSwitcher.svelte';
	import AddSourceModal from './AddSourceModal.svelte';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Popover, PopoverTrigger, PopoverContent } from '$lib/components/ui/popover/index.js';
	import { Alert, AlertDescription } from '$lib/components/ui/alert/index.js';
	import {
		shouldShowSourceRows,
		worstHealthTone,
		visibleChipCount,
		type HealthTone
	} from '$lib/format';
	import { participatesIn } from '$lib/participation';
	import type { KernelConfig, SourceStatus } from '$lib/api';
	import Ellipsis from '@lucide/svelte/icons/ellipsis';
	import X from '@lucide/svelte/icons/x';
	import { cn } from '$lib/utils.js';

	// The webspace title renders synchronously from the route param, same
	// as Phase 1. The chip row below it DOES have its own async load/error
	// state (GET /api/sources, owned by the caller) — unlike Phase 1's
	// title-only header, a non-critical sources failure must never block
	// or blank the primary stream view, so the row renders only once
	// sourcesState is 'ready' and at least one source is configured
	// (shouldShowSourceRows, 02-UI-SPEC.md E1/E5, unchanged this phase).
	let {
		webspace,
		webspaces,
		oncreatewebspace,
		onreload,
		reloadBusy,
		reloadError,
		onmanagesources,
		sources,
		sourcesState,
		selectedSources,
		onfilter,
		onclearfilters,
		onrefresh,
		onrefreshall,
		searchQuery,
		onsearch,
		filters,
		filterBusy,
		filterError,
		unknownConfigKeys,
		onsavefilter,
		filterBySource,
		onremovesourcefilter,
		onremovefilter,
		config,
		baseHash,
		pluginTypes,
		pluginTypeTiers,
		envVars,
		onsourceadded,
		onedit,
		collapsed = false,
		excludedCount,
		view,
		ontoggleview
	}: {
		webspace: string;
		// webspaces backs the WebspaceSwitcher's menu list (D-10) — the
		// full set of configured webspace names in the kernel's own
		// deterministic GET /api/config order, owned and fetched by the
		// caller (the webspace route already fetches config for the
		// filter-write path, 07-01-PLAN.md).
		webspaces: string[];
		oncreatewebspace: () => void;
		// onreload/reloadBusy/reloadError (09-06-PLAN.md Task 2,
		// 09-UI-SPEC.md Fix 7): the relocated "Reload config" action. The
		// first two are forwarded to WebspaceSwitcher exactly as
		// oncreatewebspace/onmanagesources already are — this component owns
		// no reload logic of its own, only threads it through. reloadError,
		// when non-null, renders a destructive Alert in the header region
		// using the identical treatment as filterError below — kept as a
		// DISTINCT prop rather than merged into filterError's own slot: a
		// filter write failing and a config reload failing are different
		// events with different recovery, and conflating them would make
		// the copy ambiguous about which action actually failed. The route
		// owns the reload call itself (mirroring how it already owns
		// oncreatewebspace/onmanagesources) and clears reloadError on the
		// next successful action, matching how filterError is cleared today.
		onreload: () => void;
		reloadBusy: boolean;
		reloadError: string | null;
		onmanagesources: () => void;
		sources: SourceStatus[];
		sourcesState: 'loading' | 'error' | 'ready';
		selectedSources: Set<string>;
		onfilter: (name: string) => void;
		onclearfilters: () => void;
		onrefresh: (name: string) => void;
		onrefreshall: () => void;
		searchQuery: string;
		onsearch: (q: string) => void;
		// Search-promotion permanent filter stack (D-16-D-19): filters is
		// the webspace's currently-saved `filter` array (stored order,
		// never re-sorted — UI-12 ordering edge), rendered as a row of
		// FilterChip below the source-chip row. filterBusy disables both
		// the "Save as filter" button and every chip's remove control
		// while a PUT /api/config write for a filter change is in flight.
		// filterError, when non-null, renders a destructive Alert below
		// the chip rows (D-03/D-09 error surfacing) — the caller owns
		// clearing it on the next successful action.
		filters: string[];
		filterBusy: boolean;
		filterError: string | null;
		// unknownConfigKeys is GET /api/config's `unknown_keys` list
		// (deviation, Rule 2: discovered live during the Task 1 tracer
		// checkpoint) — a non-empty list means config.toml already
		// carries a hand-authored key the kernel's Config struct doesn't
		// model, which blocks EVERY save through this UI outright (the
		// Store.Save unknown-key guard refuses to write, D-01's
		// lossless-rewrite prohibition). Rendered proactively, independent
		// of any save attempt, so that state is never silently discovered
		// only after a click appears to do nothing.
		unknownConfigKeys: string[];
		onsavefilter: () => void;
		// filterBySource (M2-R3, #55): the current webspace's per-instance
		// filter map — renders one labelled chip per (instance, term) in the
		// filter row and the funnel marker on the instance's own SourceChip.
		filterBySource: Record<string, string[]>;
		onremovesourcefilter: (instance: string, term: string) => void;
		onremovefilter: (term: string) => void;
		// Add-source picker/flows (D-11, 07-04-PLAN.md): config/baseHash are
		// the same last GET/PUT /api/config snapshot the filter-write path
		// already threads through; null while that fetch is still in
		// flight, which the trigger below tolerates by simply not rendering
		// yet — it appears the moment config resolves, same timing the rest
		// of this row already depends on. pluginTypes is
		// GET /api/config/plugin-types' own result, fetched by the route
		// alongside config. onsourceadded fires after any add-source save
		// completes, so the caller can refresh config/sources/stream state.
		config: KernelConfig | null;
		baseHash: string;
		pluginTypes: string[];
		// pluginTypeTiers (Phase 11, PLUG-06/08): GET /api/config/plugin-types'
		// own plugin_type_tiers lookup table, fetched alongside pluginTypes
		// by the caller and threaded straight through to AddSourceModal —
		// this component owns no tier logic of its own.
		pluginTypeTiers: Record<string, string>;
		// envVars is the last GET/PUT /api/config response's own env_vars
		// presence map (D-15's secret-field set/unset badge source) — see
		// SecretField.svelte's doc comment for why this, not a per-keystroke
		// lookup, is what the badge reads.
		envVars: Record<string, boolean>;
		onsourceadded: () => void;
		// onedit (D-12, Task 3) is threaded to every real SourceChip below —
		// including the clones inside the overflow popover, so a chip there
		// behaves identically to one in the row — but explicitly NOT to the
		// invisible measurement clones, which keep a no-op handler so a
		// measurement clone can never dispatch an edit.
		//
		// 11-06-PLAN.md Task 1 widens the kind union with 'trust-update'
		// (E4) — this type annotation must stay in lockstep with
		// SourceChip.svelte's own onedit prop type, since this component's
		// onedit is a distinct, separately-typed prop passed straight
		// through, not inferred from the child.
		onedit: (
			name: string,
			kind:
				| 'connection'
				| 'match'
				| 'relink'
				| 'remove'
				| 'trust-update'
				| 'trust-key'
				| 'untrust-key'
				| 'filter'
		) => void;
		// collapsed (checkpoint deviation, 09.1-01-PLAN.md issue 2): the
		// caller's own scroll-driven decision, below 1024px only — the
		// route owns the scroll listener (on the stream pane's own scroll
		// container, never the window) since scroll direction is
		// meaningless above 1024px, where this header never collapses.
		// Every collapse effect below is itself scoped under `max-lg:`, so
		// an unexpected `true` at desktop width is still a no-op — belt
		// and braces alongside the caller's own width-gated listener.
		collapsed?: boolean;
		// excludedCount/view/ontoggleview (13-UI-SPEC.md E4, KERN-10): the
		// excluded-items view toggle, rendered in this component's own
		// existing trailing reserved group (alongside Refresh all) —
		// excludedCount sources from the SAME stream response that drives
		// the stream itself (StreamResponse.excluded_count), never a
		// separate round trip. view is the caller's own current bucket
		// ('included' | 'excluded'); ontoggleview flips it. The toggle is
		// absent entirely (not disabled) when excludedCount is 0 (D-06).
		excludedCount: number;
		view: 'included' | 'excluded';
		ontoggleview: () => void;
	} = $props();

	let showSourceRows = $derived(shouldShowSourceRows(sourcesState, sources));

	// The chip row's CONTENT is scoped to this webspace's actual
	// participants (07-14-PLAN.md Task 3, closes 07-UAT.md G-07-6's second
	// half): `sources` (GET /api/sources) is kernel-wide and unfiltered by
	// design — one entry per configured instance, regardless of which
	// webspace(s) it belongs to. `participatesIn` (the shared predicate
	// also used by AddSourceModal's picker, so the two surfaces can never
	// disagree) narrows it. `config` can still be null on the very first
	// render — loadConfig and loadSources are independent, unsequenced
	// fetches in the caller — so every configured source is shown until it
	// resolves, rather than none, so the row never flashes empty before
	// narrowing a moment later.
	//
	// Deliberately NOT applied to `showSourceRows` above: that gate answers
	// "does this installation have any configured source instances at
	// all", a different question from "which of them belong to this
	// webspace" — see the `{#if showSourceRows}` block below for the full
	// rationale (it also gates the "+" add-source trigger, which must stay
	// reachable for a webspace with zero participants).
	let participatingSources = $derived(
		config ? sources.filter((source) => participatesIn(config, webspace, source.name)) : sources
	);

	// "Save as filter" (D-19): shown only when the trimmed search query is
	// non-empty AND is not already an active filter term — a query
	// byte-identical to an active term offers no affordance, so the
	// persisted filter list can never contain a duplicate (UI-12 adjacency
	// edge). A query that is merely a prefix or superstring of an active
	// term is untouched by this check and still offers the affordance.
	let trimmedQuery = $derived(searchQuery.trim());
	let showSaveAsFilter = $derived(trimmedQuery !== '' && !filters.includes(trimmedQuery));

	// The row below renders with `gap-2` (line ~175) — Tailwind's `gap-2` is
	// 0.5rem, 8px at the default 16px root font size. Kept as a named
	// constant (rather than re-derived from computed style) so the overflow
	// math in `visibleChipCount` and the row's actual class stay visibly in
	// sync; if the Tailwind gap scale on the row ever changes, update both
	// together (CR-01).
	const CHIP_ROW_GAP_PX = 8;

	// Minimum width the row must be able to seat before visibleChipCount's
	// G-14-2 floor forces one chip inline (14-06-PLAN.md Task 2). This is
	// the chip's non-text furniture, verified live in the browser at 375px
	// (not guessed — probe: row 327px, trailing 87px, add-source 44px,
	// overflow trigger 73px, leaving a 99px overflow budget): health dot
	// 8px + its two 6px gaps, trust-badged plugin icon 14px, button padding
	// 10px left + 6px right, wrapper right padding 4px, the 32px fixed-size
	// actions trigger, and the wrapper's 1px borders = 88px. A chip at
	// exactly this width still shows its dot, icon, ellipsized name and
	// menu — a functional hover target, not a sliver — and in the common
	// forced-single-chip case the overflow trigger disappears entirely, so
	// the chip springs well past this minimum. Kept as a named constant
	// (rather than re-derived) for the same reason as CHIP_ROW_GAP_PX
	// above; the e2e spec 14-chip-health-narrow-viewport.spec.ts pins its
	// own copy — change both together.
	const MIN_INLINE_CHIP_PX = 88;

	const DOT_TONE_CLASS: Record<HealthTone, string> = {
		success: 'bg-success',
		warning: 'bg-warning',
		destructive: 'bg-destructive',
		unknown: 'bg-muted-foreground'
	};

	// --- Overflow measurement (UI-07 "Scaling to 10+ instances") ---
	//
	// Four DOM regions cooperate:
	//  - `rowEl`: the visible, clipped row — its clientWidth is the
	//    available width.
	//  - `measureEl`: an off-screen (position:absolute, invisible), unclipped
	//    clone of every chip, rendered purely so each chip's NATURAL
	//    (unconstrained) offsetWidth can be read — the visible row itself
	//    never reveals a hidden chip's true width once `overflow-hidden`
	//    clips it.
	//  - `trailingEl`: the always-rendered Clear filters/Refresh all group,
	//    already live in the visible row (Clear filters only appears when
	//    the selection is non-empty, so this group's own width already
	//    reflects that live gating — no separate hidden clone needed).
	//  - `overflowTriggerMeasureEl`: a hidden clone of the overflow trigger
	//    itself, needed because the real trigger only renders once overflow
	//    is already known — a chicken-and-egg case a hidden clone resolves.
	//
	// Attachment (see the $effect below, wired through
	// web/src/lib/resize-observer.ts) is driven by the four element
	// references themselves: reading them synchronously inside the effect
	// body registers them as its dependencies, so the attachment (re)runs
	// whenever a ref binds or rebinds — including the very first render,
	// when every ref is still unbound (the sources request is still in
	// flight and the row's render gate is false) and nothing is
	// constructed to watch. That is what keeps the row's overflow
	// computation correct for the lifetime of the component, not only at
	// initial load. The measurement callback itself only writes a value
	// when it actually changed, and never schedules a synchronous
	// re-measure from inside itself — that combination is what keeps this
	// bounded rather than a resize-then-measure feedback loop (T-06-10).
	let rowEl: HTMLDivElement | undefined = $state();
	let measureEl: HTMLDivElement | undefined = $state();
	let trailingEl: HTMLDivElement | undefined = $state();
	let overflowTriggerMeasureEl: HTMLButtonElement | undefined = $state();
	// addSourceWrapperEl wraps AddSourceModal's own trigger button
	// (D-11) — measured directly (it always renders once `config` is
	// available, unlike the overflow trigger, so no hidden clone is
	// needed) and folded into the reserved-width budget below so
	// visibleChipCount never lets a chip overlap it at tight widths.
	let addSourceWrapperEl: HTMLDivElement | undefined = $state();

	let availableWidth = $state(0);
	let chipWidths = $state<number[]>([]);
	let reservedWidth = $state(0);
	let overflowTriggerWidth = $state(0);
	let addSourceWidth = $state(0);

	function widthsEqual(a: number[], b: number[]): boolean {
		return a.length === b.length && a.every((value, i) => value === b[i]);
	}

	function measure() {
		if (rowEl) {
			const next = rowEl.clientWidth;
			if (next !== availableWidth) availableWidth = next;
		}
		if (measureEl) {
			const widths = Array.from(measureEl.children).map(
				(el) => (el as HTMLElement).offsetWidth
			);
			if (!widthsEqual(widths, chipWidths)) chipWidths = widths;
		}
		if (trailingEl) {
			const next = trailingEl.offsetWidth;
			if (next !== reservedWidth) reservedWidth = next;
		}
		if (overflowTriggerMeasureEl) {
			const next = overflowTriggerMeasureEl.offsetWidth;
			if (next !== overflowTriggerWidth) overflowTriggerWidth = next;
		}
		if (addSourceWrapperEl) {
			const next = addSourceWrapperEl.offsetWidth;
			if (next !== addSourceWidth) addSourceWidth = next;
		}
	}

	// Reading all four refs here — not just rowEl/measureEl/trailingEl —
	// is what pins IN-01 closed: the overflow-trigger measurement clone is
	// now observed directly, rather than only refreshed incidentally when
	// the sources-keyed effect below happens to fire. No explicit initial
	// measure is needed: observing an element runs its callback once
	// immediately, and the sources-keyed effect already schedules a
	// deferred measure on the same mount/update transition.
	$effect(() =>
		observeResize(
			[rowEl, measureEl, trailingEl, overflowTriggerMeasureEl, addSourceWrapperEl],
			measure
		)
	);

	// Re-measure whenever the FILTERED source list itself changes shape (a
	// different participant count, a renamed display_name that changes a
	// chip's natural width, or a webspace switch that changes which
	// instances participate) — reading `participatingSources` here (not
	// the raw `sources` prop) is what makes this effect re-run, and is
	// deliberately the same filtered list the visible slice and the
	// measurement clones below consume, so a re-measure is never triggered
	// by (or skipped for) a change to a non-participating instance the row
	// never rendered. The actual DOM read is deferred to a microtask so it
	// runs after Svelte has applied the resulting DOM update. `config` is
	// read too so the add-source trigger's own arrival (it renders only
	// once config resolves) also triggers a re-measure.
	$effect(() => {
		participatingSources;
		selectedSources;
		config;
		queueMicrotask(measure);
	});

	// The add-source trigger's own width (plus the one extra gap between
	// it and the trailing group, per CR-01's per-gap accounting) is folded
	// into the reserved budget here, at the call site, rather than by
	// widening visibleChipCount's own parameter list — that function's
	// signature and its existing tests stay untouched; a new
	// always-present reserved element is, from that pure function's own
	// point of view, indistinguishable from a wider trailing group.
	let combinedReservedWidth = $derived(
		reservedWidth + addSourceWidth + (addSourceWidth > 0 ? CHIP_ROW_GAP_PX : 0)
	);

	// The MIN_INLINE_CHIP_PX floor stays bounded and non-oscillating
	// (T-06-10's property, preserved): the forced chip shrinks purely in
	// CSS layout (the `shrinkable` prop below), so the NATURAL widths in
	// `chipWidths` — read from the never-shrinkable measurement clones —
	// are unchanged by it, and a re-measure cannot feed the shrink back
	// into this count.
	let visibleCount = $derived(
		visibleChipCount(
			chipWidths,
			availableWidth,
			combinedReservedWidth,
			overflowTriggerWidth,
			CHIP_ROW_GAP_PX,
			MIN_INLINE_CHIP_PX
		)
	);
	let visibleSources = $derived(participatingSources.slice(0, visibleCount));
	let hiddenSources = $derived(participatingSources.slice(visibleCount));
	let hasOverflow = $derived(hiddenSources.length > 0);
	let overflowTone = $derived(worstHealthTone(hiddenSources));
</script>

<header class="shrink-0 border-b border-border bg-card px-6 py-6 {collapsed ? 'max-lg:py-3' : ''}">
	<!--
	  D-10: the static <h1> title becomes a webspace switcher — same
	  Display role (28px/600/1.2), same truncate+title treatment on the
	  trigger — so it still reads as a heading at rest, but now opens a
	  drop-down listing every configured webspace plus the "+ New
	  webspace" / "Manage sources…" escape hatches (D-13).
	-->
	<div class="flex items-start justify-between gap-4">
		<div class="min-w-0">
			<WebspaceSwitcher
				{webspace}
				{webspaces}
				oncreate={oncreatewebspace}
				{onreload}
				{reloadBusy}
				onmanage={onmanagesources}
			/>
		</div>

		<!--
		  Header branding lockup (quick task 260811-rqc): app icon + "topos"
		  wordmark + tagline, right-aligned beside the webspace-switcher
		  title. shrink-0 so the switcher's own truncate (min-w-0 on its
		  sibling column above) gives way before this block is ever
		  squeezed. A SIBLING of — and rendered entirely BEFORE — the
		  measured chip row below (bind:this={rowEl}): nesting it inside
		  that row would silently shrink visibleChipCount's
		  available-width input with no visible error (T-rqc-02).

		  09.1-04-PLAN.md Task 1: max-md:hidden below 768px. This is purely
		  decorative and carries no affordance, which is why it is the one
		  element this phase drops outright at narrow widths rather than
		  relocating — the width it reclaims goes to the webspace switcher
		  title. The chip overflow popover needs no narrow-width special
		  case: the existing ResizeObserver-driven measurement is already
		  width-driven at every width.
		-->
		<div class="flex shrink-0 items-center gap-2 max-md:hidden">
			<img src="/app-icon.png" alt="" class="size-10 shrink-0 rounded-md" />
			<div class="flex flex-col text-muted-foreground">
				<span class="text-[20px] leading-[1.2] font-semibold">topos</span>
				<span class="text-[12px] leading-[1.4]">bringing all your topics to one place</span>
			</div>
		</div>
	</div>

	<!--
	  Reload-config failure (Fix 7, relocated): the same modal-less
	  destructive-Alert pattern as the filterError Alert below — a config
	  reload failing is a different event from a filter write failing
	  (distinct props, see the prop-block comment above), but shares the
	  identical header-region treatment.
	-->
	{#if reloadError}
		<Alert variant="destructive" class="mt-3">
			<AlertDescription>{reloadError}</AlertDescription>
		</Alert>
	{/if}

	<!--
	  config.toml has a hand-authored key the kernel doesn't model (a stray
	  table left over from a prior manual edit or migration). Store.Save's
	  unknown-key guard refuses EVERY save while this is true, anywhere in
	  the file — not only for this webspace — so this renders unconditionally
	  ahead of any save attempt, rather than only surfacing once a save is
	  clicked and rejected (the surprise a silent block would otherwise be).
	-->
	{#if unknownConfigKeys.length > 0}
		<Alert variant="destructive" class="mt-3">
			<AlertDescription>
				config.toml has keys topos doesn't recognise ({unknownConfigKeys.join(', ')}) — fix these
				by hand before any change made here can be saved.
			</AlertDescription>
		</Alert>
	{/if}

	<!--
	  Checkpoint fix (09.1-01-PLAN.md issue 2): everything below the
	  switcher/branding row collapses away below 1024px while `collapsed`
	  is true — the chip row, the search box, the saved-filter chips, and
	  a filter write's own error alert. `overflow-hidden` on this wrapper
	  is what actually prevents the collapsing content from being
	  interactable/focusable while visually collapsed, not merely
	  visually hidden.

	  Deliberately an instant snap (no CSS transition on max-height/
	  opacity), not a stylistic choice: collapsing/expanding changes
	  <main>'s available height, which changes the stream pane's own
	  clientHeight, which the browser responds to by resetting that same
	  element's scrollTop — confirmed live, independent of whether the
	  change is instant or animated. +page.svelte's setHeaderCollapsed
	  restores the captured pre-toggle scrollTop across two animation
	  frames immediately after the snap; an ACTIVE multi-frame CSS
	  transition re-triggers that same browser reset on every intermediate
	  frame, arriving after the one-shot restore and silently overwriting
	  it (reproduced live: the stream settled tens of pixels away from
	  where it was scrolled). A smoothly animated collapse is a real,
	  worthwhile follow-up, but needs the restore to track the transition
	  itself (e.g. a rAF loop bounded by `transitionend`), not a single
	  fire-and-forget correction — out of scope for this checkpoint fix.
	-->
	<div
		class="max-lg:overflow-hidden {collapsed ? 'max-lg:max-h-0 max-lg:opacity-0' : 'max-lg:max-h-[480px] max-lg:opacity-100'}"
	>
		{#if showSourceRows}
			<!--
			  D-01: one merged chip per configured instance replaces Phase 2's
		  two rows. Chips render in config-declared instance order — never
		  sorted or reordered by health state, so a chip's position stays a
		  stable target across health changes. `flex-nowrap` plus the
		  measured overflow popover below keep this a single line at any
		  instance count.
		-->
		<div class="mt-4 flex flex-nowrap items-center gap-2 overflow-hidden" bind:this={rowEl}>
			<!--
			  shrinkable (G-14-2): only the VISIBLE row's chips may give way —
			  a floor-forced chip truncates its display name harder rather
			  than pushing the reserved trailing controls out of the clipped
			  row. The overflow-popover clones are laid out in a column
			  (shrinking would compress height, not width) and the invisible
			  measurement clones exist to report NATURAL width — neither is
			  marked.
			-->
			{#each visibleSources as source (source.name)}
				<SourceChip
					{source}
					selected={selectedSources.has(source.name)}
					{onfilter}
					{onrefresh}
					{onedit}
					busy={filterBusy}
					filtered={(filterBySource[source.name] ?? []).length > 0}
					shrinkable
				/>
			{/each}

			{#if hasOverflow}
				<Popover>
					<PopoverTrigger>
						{#snippet child({ props })}
							<button
								{...props}
								type="button"
								aria-label={`${hiddenSources.length} more sources`}
								class="flex h-11 shrink-0 items-center gap-1.5 rounded-full border border-border bg-card px-2.5 text-muted-foreground"
							>
								<Ellipsis class="size-4" aria-hidden="true" />
								<span class="text-[14px] leading-[1.4]">+{hiddenSources.length}</span>
								<span
									class={cn('size-2 shrink-0 rounded-full', DOT_TONE_CLASS[overflowTone])}
									aria-hidden="true"
								></span>
							</button>
						{/snippet}
					</PopoverTrigger>
					<PopoverContent>
						<p class="mb-2 text-[14px] font-medium text-foreground">More sources</p>
						<div class="flex flex-col gap-2">
							{#each hiddenSources as source (source.name)}
								<SourceChip
									{source}
									selected={selectedSources.has(source.name)}
									{onfilter}
									{onrefresh}
									{onedit}
									busy={filterBusy}
									filtered={(filterBySource[source.name] ?? []).length > 0}
								/>
							{/each}
						</div>
					</PopoverContent>
				</Popover>
			{/if}

			<!--
			  D-11: the "+" add-source trigger — a sibling inside the chip
			  row, after the last visible chip / overflow trigger and before
			  the ml-auto reserved trailing group, so it is never pushed into
			  the overflow popover and always reachable in one tab stop.
			  Wrapped for direct width measurement (see addSourceWrapperEl
			  above); renders only once `config` has resolved (AddSourceModal
			  needs it to compute the picker's available-instance list).
			-->
			{#if config}
				<div bind:this={addSourceWrapperEl} class="shrink-0">
					<AddSourceModal
						{webspace}
						{config}
						{baseHash}
						{pluginTypes}
						{pluginTypeTiers}
						{envVars}
						onsaved={onsourceadded}
					/>
				</div>
			{/if}

			<!--
			  Reserved trailing space: Clear filters (when the selection is
			  non-empty) and Refresh all — measured via `trailingEl` and
			  never themselves pushed into the overflow popover, so both stay
			  reachable in one tab stop from the chip row regardless of
			  instance count.
			-->
			<div class="ml-auto flex shrink-0 items-center gap-2" bind:this={trailingEl}>
				{#if selectedSources.size > 0}
					<Button variant="ghost" size="sm" onclick={onclearfilters}>
						<X class="size-4" aria-hidden="true" />
						Clear filters
					</Button>
				{/if}
				<!-- Excluded-items view toggle (13-UI-SPEC.md E4, D-06):
				     absent entirely — not disabled/greyed — while
				     excludedCount is 0, so a webspace with no exclusions
				     looks exactly like it did before this phase. Pressed
				     state mirrors the existing selected-chip accent
				     treatment (SourceChip.svelte), never a new toggle
				     style. Lives in this reserved trailing group, never
				     inside either measured chip row above, so it can
				     never participate in or perturb their overflow
				     measurement. -->
				{#if excludedCount > 0}
					<Button
						variant="outline"
						size="sm"
						aria-pressed={view === 'excluded'}
						onclick={ontoggleview}
						class={cn(view === 'excluded' && 'border-primary text-primary')}
					>
						Excluded ({excludedCount})
					</Button>
				{/if}
				<Button variant="outline" size="sm" onclick={onrefreshall}>Refresh all</Button>
			</div>
		</div>

		<!--
		  Off-screen measurement clones: never visible to the user
		  (position:absolute + invisible + pointer-events-none), used only
		  to read each chip's and the overflow trigger's natural
		  (unconstrained) width so visibleChipCount has real numbers to
		  compute against — the visible row's own `overflow-hidden` clips a
		  hidden chip's true width before it can ever be measured there.

		  Checkpoint fix (09.1-01-PLAN.md, issue 1): both clones below are
		  `position: absolute` with no `position: relative` ancestor
		  anywhere in this app, so their containing block was the initial
		  containing block, not this component — an ancestor's own
		  `overflow-hidden` (e.g. one added directly to the <header> root)
		  has NO clipping effect on them, confirmed live. This wrapper is
		  BOTH `relative` (so it becomes their real containing block) AND
		  `overflow-hidden h-0` (so their full natural, unclipped width —
		  the whole point of a measurement clone — never contributes to
		  document.documentElement.scrollWidth). Neither offsetWidth read
		  below is affected: intrinsic sizing is computed independently of
		  whether an ancestor visually clips the result.
		-->
		<div class="relative h-0 overflow-hidden">
			<div
				class="invisible absolute flex items-center gap-2 whitespace-nowrap"
				aria-hidden="true"
				bind:this={measureEl}
			>
				{#each participatingSources as source (source.name)}
					<SourceChip
						{source}
						selected={selectedSources.has(source.name)}
						onfilter={() => {}}
						onrefresh={() => {}}
						onedit={() => {}}
					/>
				{/each}
			</div>
			<button
				type="button"
				tabindex="-1"
				class="invisible absolute flex h-11 shrink-0 items-center gap-1.5 rounded-full border border-border bg-card px-2.5"
				aria-hidden="true"
				bind:this={overflowTriggerMeasureEl}
			>
				<Ellipsis class="size-4" />
				<span class="text-[14px] leading-[1.4]">+{participatingSources.length}</span>
				<span class="size-2 shrink-0 rounded-full"></span>
			</button>
		</div>
	{/if}

	<!--
	  The search box renders whenever the webspace does — unlike the row
	  above, it is NOT gated behind shouldShowSourceRows: searching the
	  local index doesn't depend on any source being reachable, so a
	  sourceless webspace or a sources-request failure never hides it.
	  "Save as filter" sits immediately trailing it in the same row
	  (07-UI-SPEC.md "Save-as-filter affordance").
	-->
	<div class="mt-3 flex items-center gap-2">
		<SearchBox query={searchQuery} onquery={onsearch} />
		{#if showSaveAsFilter}
			<Button variant="ghost" size="sm" disabled={filterBusy} onclick={onsavefilter}>
				Save as filter
			</Button>
		{/if}
	</div>

	<!--
	  Filter chip row (D-19): its own line below the source-chip row and
	  the search box — never merged into the source-chip row's
	  flex-nowrap overflow measurement, since permanent filters and
	  per-source health/filter are two different concepts sharing this
	  header. Gated on filters.length so it is ABSENT (not an
	  empty-styled row) with zero active filters.
	-->
	{#if filters.length > 0 || Object.keys(filterBySource).length > 0}
		<div class="mt-3 flex flex-wrap items-center gap-2">
			{#each filters as term (term)}
				<FilterChip {term} disabled={filterBusy} onremove={onremovefilter} />
			{/each}
			<!-- Per-source filter chips (M2-R3, #55): labelled with whose rows
			     they narrow, removable independently, after the global chips
			     in a stable instance-then-term order. -->
			{#each Object.keys(filterBySource).sort() as instance (instance)}
				{#each filterBySource[instance] as term (instance + ':' + term)}
					<FilterChip
						{term}
						{instance}
						disabled={filterBusy}
						onremove={(t) => onremovesourcefilter(instance, t)}
					/>
				{/each}
			{/each}
		</div>
	{/if}

		{#if filterError}
			<Alert variant="destructive" class="mt-3">
				<AlertDescription>{filterError}</AlertDescription>
			</Alert>
		{/if}
	</div>
</header>
