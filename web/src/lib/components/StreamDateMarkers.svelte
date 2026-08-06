<script lang="ts">
	import {
		Tooltip,
		TooltipContent,
		TooltipProvider,
		TooltipTrigger
	} from '$lib/components/ui/tooltip/index.js';
	import { dateMarkers, formatItemDate } from '$lib/format';
	import type { StreamItem } from '$lib/api';

	// items is the same filtered, in-order item list the stream pane is
	// currently rendering (UI-11) — this component never re-fetches or
	// re-sorts. trackHeightPx is the stream scroll region's own client
	// height, bound by the caller. scrollContainer is that same scroll
	// region's element reference, used only to resolve a tick's target row
	// by its data-item-id attribute (StreamRow.svelte) when clicked.
	let {
		items,
		trackHeightPx,
		scrollContainer
	}: {
		items: StreamItem[];
		trackHeightPx: number;
		scrollContainer: HTMLElement | null;
	} = $props();

	// The single pure derivation this overlay renders from — see format.ts
	// for the full adaptive-thinning rule and markers.test.ts for its unit
	// tests. Empty items, a non-positive track height, or a single-date
	// stream all resolve to zero markers here, with no branching needed in
	// this template.
	let markers = $derived(dateMarkers(items, trackHeightPx));

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
  Absolutely-positioned overlay pinned to the stream pane's trailing edge,
  inset just enough to sit immediately inside the native scrollbar track
  (app.css's ::-webkit-scrollbar is 10px wide, unchanged by this file),
  spanning the pane's full height. pointer-events-none on this container is
  the mechanical half of the "never degrade the primary scrollbar" rule —
  only each tick's own hit area (pointer-events-auto below) opts back into
  interactivity, so the native thumb stays grabbable and the stream rows
  underneath stay clickable across the overlay's full span.
-->
<div class="pointer-events-none absolute inset-y-0 right-0.5 w-4" aria-hidden={markers.length === 0}>
	{#each markers as marker (marker.itemId)}
		<TooltipProvider>
			<Tooltip>
				<TooltipTrigger>
					{#snippet child({ props })}
						<button
							{...props}
							type="button"
							class="group pointer-events-auto absolute right-0 flex h-4 w-4 -translate-y-1/2 items-center justify-center rounded-sm focus-visible:outline-none"
							style={`top: ${marker.topPx}px;`}
							aria-label={formatItemDate(marker.timestampUnix)}
							onclick={() => scrollToMarker(marker.itemId)}
						>
							<!--
							  The tick itself: a 2px-tall horizontal bar reusing the
							  existing scrollbar thumb tokens at rest and on
							  hover/focus (app.css, quick task 260805-j98) — no new
							  colour value is introduced by this phase.
							-->
							<span
								class="h-0.5 w-3 rounded-full bg-[var(--scrollbar-thumb)] group-hover:bg-[var(--scrollbar-thumb-hover)] group-focus-visible:bg-[var(--scrollbar-thumb-hover)]"
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
