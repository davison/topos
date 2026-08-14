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
-->
<Sonner
	theme="dark"
	position="bottom-right"
	class="toaster group"
	style="--normal-bg: var(--color-popover); --normal-text: var(--color-popover-foreground); --normal-border: var(--color-border); --description-color: var(--color-muted-foreground);"
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
