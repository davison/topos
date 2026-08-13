<script lang="ts">
	// TrustBadge overlays a small warning glyph on a plugin identity icon
	// (or any wrapped snippet) whenever that plugin's launch-time
	// provenance tier is 'external' (Phase 11, D-06/D-07/D-08 lineage,
	// 11-UI-SPEC.md E2) — rendering nothing at all for a trusted-tier
	// source, so a trusted chip's markup stays byte-identical to before
	// this phase.
	//
	// Reuses CircleAlert (`@lucide/svelte/icons/circle-alert`) — the exact
	// icon SecretField.svelte already imports and colors text-warning for
	// "this needs the operator's attention" — rather than introducing a
	// new glyph for a conceptually adjacent but visually inconsistent
	// signal. Deliberately not the health dot's plain colored circle: a
	// second small dot next to the existing health dot would read as a
	// second health signal, not a distinct trust signal.
	//
	// Two declared scales share this exact markup shape (11-UI-SPEC.md
	// E2's markup contract), only sizes/surface tokens differ:
	//   - 'chip'   (wraps a size-3.5/14px icon): size-3.5 backdrop, bg-card,
	//              size-2.5 glyph — matches the chip's own bg-card surface.
	//   - 'picker' (wraps a size-4/16px icon): size-4 backdrop, bg-popover,
	//              size-3 glyph — matches PopoverContent's own surface.
	//
	// The backdrop wrapper carries aria-hidden="true": the badge is
	// redundant with the adjacent text label (the picker's "Untrusted"
	// label, landing in a later Phase 11 plan) and the chip's own
	// tooltip — it must never be the ONLY channel conveying "untrusted"
	// to an assistive-tech user.
	import CircleAlert from '@lucide/svelte/icons/circle-alert';
	import type { Snippet } from 'svelte';

	let {
		tier,
		scale,
		children
	}: {
		tier: 'trusted' | 'external';
		scale: 'chip' | 'picker';
		children: Snippet;
	} = $props();

	let backdropSize = $derived(scale === 'chip' ? 'size-3.5' : 'size-4');
	let backdropBg = $derived(scale === 'chip' ? 'bg-card' : 'bg-popover');
	let glyphSize = $derived(scale === 'chip' ? 'size-2.5' : 'size-3');
</script>

<span class="relative inline-flex shrink-0">
	{@render children()}
	{#if tier === 'external'}
		<span
			class={`absolute -bottom-1 -right-1 flex items-center justify-center rounded-full ${backdropSize} ${backdropBg}`}
			aria-hidden="true"
		>
			<CircleAlert class={`${glyphSize} text-warning`} />
		</span>
	{/if}
</span>
