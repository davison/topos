<script lang="ts">
	import { observeResize } from '$lib/resize-observer';
	import SourceChip from './SourceChip.svelte';
	import SearchBox from './SearchBox.svelte';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Popover, PopoverTrigger, PopoverContent } from '$lib/components/ui/popover/index.js';
	import {
		shouldShowSourceRows,
		worstHealthTone,
		visibleChipCount,
		type HealthTone
	} from '$lib/format';
	import type { SourceStatus } from '$lib/api';
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
		sources,
		sourcesState,
		selectedSources,
		onfilter,
		onclearfilters,
		onrefresh,
		onrefreshall,
		searchQuery,
		onsearch
	}: {
		webspace: string;
		sources: SourceStatus[];
		sourcesState: 'loading' | 'error' | 'ready';
		selectedSources: Set<string>;
		onfilter: (name: string) => void;
		onclearfilters: () => void;
		onrefresh: (name: string) => void;
		onrefreshall: () => void;
		searchQuery: string;
		onsearch: (q: string) => void;
	} = $props();

	let showSourceRows = $derived(shouldShowSourceRows(sourcesState, sources));

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

	let availableWidth = $state(0);
	let chipWidths = $state<number[]>([]);
	let reservedWidth = $state(0);
	let overflowTriggerWidth = $state(0);

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
	}

	// Reading all four refs here — not just rowEl/measureEl/trailingEl —
	// is what pins IN-01 closed: the overflow-trigger measurement clone is
	// now observed directly, rather than only refreshed incidentally when
	// the sources-keyed effect below happens to fire. No explicit initial
	// measure is needed: observing an element runs its callback once
	// immediately, and the sources-keyed effect already schedules a
	// deferred measure on the same mount/update transition.
	$effect(() =>
		observeResize([rowEl, measureEl, trailingEl, overflowTriggerMeasureEl], measure)
	);

	// Re-measure whenever the source list itself changes shape (a
	// different instance count, or a renamed display_name that changes a
	// chip's natural width) — reading `sources` here is what makes this
	// effect re-run; the actual DOM read is deferred to a microtask so it
	// runs after Svelte has applied the resulting DOM update.
	$effect(() => {
		sources;
		selectedSources;
		queueMicrotask(measure);
	});

	let visibleCount = $derived(
		visibleChipCount(chipWidths, availableWidth, reservedWidth, overflowTriggerWidth)
	);
	let visibleSources = $derived(sources.slice(0, visibleCount));
	let hiddenSources = $derived(sources.slice(visibleCount));
	let hasOverflow = $derived(hiddenSources.length > 0);
	let overflowTone = $derived(worstHealthTone(hiddenSources));
</script>

<header class="shrink-0 border-b border-border bg-card px-6 py-6">
	<!--
	  Display role (28px/600/1.2). Webspace names are user-defined
	  config.toml keys of arbitrary length, so this truncates to one line
	  with an ellipsis; the title attribute keeps the full name reachable
	  on hover regardless of how long it is.
	-->
	<h1 class="truncate text-[28px] leading-[1.2] font-semibold text-foreground" title={webspace}>
		{webspace}
	</h1>

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
			{#each visibleSources as source (source.name)}
				<SourceChip
					{source}
					selected={selectedSources.has(source.name)}
					{onfilter}
					{onrefresh}
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
								/>
							{/each}
						</div>
					</PopoverContent>
				</Popover>
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
		-->
		<div
			class="invisible absolute flex items-center gap-2 whitespace-nowrap"
			aria-hidden="true"
			bind:this={measureEl}
		>
			{#each sources as source (source.name)}
				<SourceChip
					{source}
					selected={selectedSources.has(source.name)}
					onfilter={() => {}}
					onrefresh={() => {}}
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
			<span class="text-[14px] leading-[1.4]">+{sources.length}</span>
			<span class="size-2 shrink-0 rounded-full"></span>
		</button>
	{/if}

	<!--
	  The search box renders whenever the webspace does — unlike the row
	  above, it is NOT gated behind shouldShowSourceRows: searching the
	  local index doesn't depend on any source being reachable, so a
	  sourceless webspace or a sources-request failure never hides it.
	-->
	<div class="mt-3">
		<SearchBox query={searchQuery} onquery={onsearch} />
	</div>
</header>
