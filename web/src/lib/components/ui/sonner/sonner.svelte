<script lang="ts">
	import { Toaster as Sonner, type ToasterProps as SonnerProps } from 'svelte-sonner';
	import Loader2Icon from '@lucide/svelte/icons/loader-2';
	import CircleCheckIcon from '@lucide/svelte/icons/circle-check';
	import OctagonXIcon from '@lucide/svelte/icons/octagon-x';
	import InfoIcon from '@lucide/svelte/icons/info';
	import TriangleAlertIcon from '@lucide/svelte/icons/triangle-alert';

	let { ...restProps }: SonnerProps = $props();
</script>

<!--
	13-UI-SPEC.md E3: this app is dark-mode-only with no light/dark toggle
	(unchanged since Phase 1) — theme="dark" is hardcoded rather than
	pulling in the shadcn-svelte registry default's `mode-watcher`
	dependency, which exists solely to track a theme toggle this app does
	not have. Themed against this app's own CSS-variable tokens
	(--color-popover/--color-popover-foreground/--color-border/
	--color-foreground/--color-muted-foreground, app.css) rather than the
	library's own default light palette. position="bottom-right" per
	13-UI-SPEC.md E3 ("least likely to occlude the stream or detail
	pane").

	Checkpoint fix (13-04-PLAN.md Task 4, defect 2, two rounds):

	Round 1 tried --normal-bg: var(--color-popover) (the same neutral
	elevation-staircase surface dropdown-menu-content.svelte/
	popover-content.svelte already use for the chip menus) with a
	strengthened border + shadow. User re-verification: still not enough
	— "the background itself, not just the border, needs to contrast with
	the main scheme," modeled on "the popovers on the source chips."

	Round 2 (this one): the chip-adjacent surface that actually reads as
	the OPPOSITE end of this app's tonal range — not just one more
	dark-staircase step lighter — is tooltip-content.svelte's own
	SourceChip hover tooltip, styled `bg-foreground text-background`: a
	full light/dark INVERSION (near-white fill, near-black text), with no
	border and no shadow at all — the swap itself is the entire
	affordance, and it is unmistakably visible against this app's
	near-black page background in a way another same-family dark tone
	(--popover, --card, --secondary — all four of app.css's own dark
	staircase steps) structurally cannot be. This toast reuses that exact
	token pair (still no new hex colour: --foreground/--background are
	both pre-existing app.css tokens) and keeps a single dark drop-shadow
	for depth, which — unlike round 1's shadow problem — now has real
	visual purpose: a LIGHT surface casting a dark shadow onto a dark page
	reads as genuinely elevated, where a dark-on-dark shadow could not.
	Still never destructive/accent-toned (13-UI-SPEC.md E3.3) — this is a
	brightness inversion, not a hue change.
-->
<Sonner
	theme="dark"
	position="bottom-right"
	class="toaster group"
	style="--normal-bg: var(--color-foreground); --normal-text: var(--color-background); --normal-border: var(--color-border); --description-color: var(--color-muted-foreground);"
	toastOptions={{
		style: 'box-shadow: 0 12px 32px color-mix(in srgb, black 60%, transparent);'
	}}
	{...restProps}
>
	{#snippet loadingIcon()}
		<Loader2Icon class="size-4 animate-spin" />
	{/snippet}
	{#snippet successIcon()}
		<CircleCheckIcon class="size-4" />
	{/snippet}
	{#snippet errorIcon()}
		<OctagonXIcon class="size-4" />
	{/snippet}
	{#snippet infoIcon()}
		<InfoIcon class="size-4" />
	{/snippet}
	{#snippet warningIcon()}
		<TriangleAlertIcon class="size-4" />
	{/snippet}
</Sonner>
