<script lang="ts">
	import {
		Tooltip,
		TooltipContent,
		TooltipProvider,
		TooltipTrigger
	} from '$lib/components/ui/tooltip/index.js';
	import { dateMarkers, formatItemDate, markerLaneTop, streamScrolls } from '$lib/format';
	import { observeResize } from '$lib/resize-observer';
	import type { StreamItem } from '$lib/api';

	// items is the same filtered, in-order item list the stream pane is
	// currently rendering (UI-11) — this component never re-fetches or
	// re-sorts. trackHeightPx is the stream scroll region's own client
	// height, bound by the caller. scrollContainer is that same scroll
	// region's element reference, used both to measure the region's full
	// content height (below) and to resolve a tick's target row by its
	// data-item-id attribute (StreamRow.svelte) when clicked.
	let {
		items,
		trackHeightPx,
		scrollContainer
	}: {
		items: StreamItem[];
		trackHeightPx: number;
		scrollContainer: HTMLElement | null;
	} = $props();

	// The scroll region's full content height (`scrollHeight`, as distinct
	// from `trackHeightPx`'s visible `clientHeight`) — the other half of
	// the scroll-overflow gate below (UI-11 gap closure G-06-6, defect 5):
	// a multi-date stream that fits its pane without scrolling has no
	// scrollbar, so the ruler must render nothing rather than decorate an
	// absent one. `measure` writes the value only when it actually
	// changed and is never scheduled synchronously from inside itself;
	// the observer is attached in a ref-driven $effect returning
	// observeResize's idempotent teardown — the same bounded pattern
	// WebspaceHeader.svelte's own chip-row measurement already proves
	// (T-06-18).
	let contentHeight = $state(0);

	function measure() {
		if (!scrollContainer) return;
		const next = scrollContainer.scrollHeight;
		if (next !== contentHeight) contentHeight = next;
	}

	$effect(() => observeResize([scrollContainer], measure));

	// Re-measure whenever the visible item list or the track's own client
	// height changes shape — reading both here is what makes this effect
	// re-run; the actual DOM read is deferred to a microtask so it runs
	// after Svelte has applied the resulting DOM update, mirroring the
	// deferral WebspaceHeader.svelte uses for its own sources-keyed
	// re-measure.
	$effect(() => {
		items;
		trackHeightPx;
		queueMicrotask(measure);
	});

	// The single pure derivation this overlay renders from — see format.ts
	// for the full adaptive-thinning and major/minor rules, and
	// markers.test.ts for their unit tests. Zero markers unless the scroll
	// region actually overflows its track (streamScrolls) — this closes
	// defect 5, a non-scrolling stream decorating nothing.
	let markers = $derived(
		streamScrolls(contentHeight, trackHeightPx) ? dateMarkers(items, trackHeightPx) : []
	);

	// Resolves a marker's target row inside scrollContainer by its
	// item-id data attribute and scrolls it to the top of the pane. A
	// missing container or an item that has since left the list (e.g. a
	// filter changed between render and click) is a no-op, never a throw.
	function scrollToMarker(itemId: string) {
		if (!scrollContainer) return;
		const target = scrollContainer.querySelector<HTMLElement>(
			`[data-item-id="${CSS.escape(itemId)}"]`
		);
		if (!target) return;
		target.scrollIntoView({ block: 'start' });
	}
</script>

<!--
  A dedicated ruler lane, clear of the native scrollbar (UI-11 gap closure
  G-06-6). The container sits right-[12px] in from the stream pane's
  trailing edge -- covering both the ~11px `scrollbar-width: thin`
  Chromium renders and the 10px ::-webkit-scrollbar fallback (app.css),
  with a pixel of clearance -- and is 12px wide, occupying the gutter
  +page.svelte's scroll region reserves (`pr-6`) rather than the native
  scrollbar's own track. This replaces the retired "immediately inside the
  scrollbar track" placement (a 2px inset), which bisected every tick
  across the scrollbar's edge -- the mechanical cause of G-06-6.
  pointer-events-none on this container is the mechanical half of the
  "never degrade the primary scrollbar" rule -- only each tick's own hit
  area (pointer-events-auto below) opts back into interactivity, so the
  native thumb stays grabbable and the stream rows underneath stay
  clickable across the overlay's full span.
-->
<div
	class="pointer-events-none absolute inset-y-0 right-[12px] w-[12px]"
	aria-hidden={markers.length === 0}
>
	{#if markers.length > 0}
		<!--
		  The rail: a faint hairline in the app's own --border token,
		  pinned to the lane's trailing edge and spanning its full height,
		  grouping the ticks into a ruler rather than leaving them as
		  disconnected dashes. Deliberately not held to the 3:1 non-text
		  floor below -- it is a grouping affordance, not a control, so
		  that floor applies to the ticks, not to this line. Rendered only
		  when at least one marker exists, so a non-scrolling or
		  single-date stream shows nothing at all.
		-->
		<div
			class="pointer-events-none absolute inset-y-0 right-0 w-px bg-border"
			aria-hidden="true"
		></div>
	{/if}
	{#each markers as marker (marker.itemId)}
		<TooltipProvider>
			<Tooltip>
				<TooltipTrigger>
					{#snippet child({ props })}
						<button
							{...props}
							type="button"
							class="group pointer-events-auto absolute right-0 flex h-4 w-[20px] -translate-y-1/2 cursor-pointer items-center justify-end rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-ring"
							style={`top: ${markerLaneTop(marker.topPx, trackHeightPx)}px;`}
							aria-label={formatItemDate(marker.timestampUnix)}
							onclick={() => scrollToMarker(marker.itemId)}
						>
							<!--
							  The tick itself, anchored to the rail and
							  extending left from it: two grades driven by
							  the `major` flag (format.ts) -- a minor tick
							  (~7px) paints the dedicated --stream-marker
							  token at rest; a major tick spans the lane's
							  full 12px and paints --stream-marker-strong
							  outright, no hover state needed. Both are
							  marker-only tokens (app.css) -- never the
							  scrollbar thumb tokens, whose reuse here is
							  what produced G-06-6's 1.86:1 rest contrast.
							  A minor tick swaps to the strong tone on
							  hover/focus, alongside the focus ring above.
							-->
							<span
								class={marker.major
									? 'h-0.5 w-[12px] rounded-full bg-[var(--stream-marker-strong)]'
									: 'h-0.5 w-[7px] rounded-full bg-[var(--stream-marker)] group-hover:bg-[var(--stream-marker-strong)] group-focus-visible:bg-[var(--stream-marker-strong)]'}
							></span>
						</button>
					{/snippet}
				</TooltipTrigger>
				<TooltipContent>
					{formatItemDate(marker.timestampUnix)}
				</TooltipContent>
			</Tooltip>
		</TooltipProvider>
	{/each}
</div>
