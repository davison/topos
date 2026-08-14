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

	Checkpoint fix (13-04-PLAN.md Task 4, defect 2): --normal-border was
	set to the raw --border token (#1e293b) — the LIGHTEST step of
	app.css's own 4-step dark elevation staircase, but still too close in
	lightness to --popover (#172033, the toast's own background) to read
	as a visible edge, and svelte-sonner's own hardcoded
	box-shadow (0px 4px 12px rgba(0,0,0,.1), see node_modules/
	svelte-sonner/dist/Toaster.svelte) is a BLACK shadow — contributing
	nothing against an already near-black page background (--background:
	#020617). Confirmed via a real browser render (13-pwa-manifest-sw.spec.ts's
	toast-contrast case) that the toast was legible up close but did not
	visually "pop" — exactly the reported "close to invisible" complaint.
	Fixed with two changes, both derived from EXISTING tokens via
	color-mix() (the same idiom app.css's own --scrollbar-thumb/
	--stream-marker tokens already use for a visible-but-neutral mark
	against this same dark background) — no new hex colour introduced,
	and the toast stays informational/neutral-toned (never accent or
	destructive), per 13-UI-SPEC.md E3.3's own constraint:
	  1. --normal-border strengthens to a genuine mid-grey ring
	     (--muted-foreground at 50% opacity) instead of the barely-lighter
	     --border token.
	  2. toastOptions.style overrides svelte-sonner's hardcoded
	     black box-shadow (not exposed as a CSS variable) with a soft
	     --muted-foreground-derived outer ring plus a darker, larger-radius
	     drop shadow for real depth against the page.
-->
<Sonner
	theme="dark"
	position="bottom-right"
	class="toaster group"
	style="--normal-bg: var(--color-popover); --normal-text: var(--color-popover-foreground); --normal-border: color-mix(in srgb, var(--color-muted-foreground) 50%, transparent); --description-color: var(--color-muted-foreground);"
	toastOptions={{
		style:
			'box-shadow: 0 0 0 1px color-mix(in srgb, var(--color-muted-foreground) 30%, transparent), 0 12px 32px color-mix(in srgb, black 60%, transparent);'
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
