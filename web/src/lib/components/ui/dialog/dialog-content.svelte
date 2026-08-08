<script lang="ts">
	// Content surface tokens per 07-UI-SPEC.md "Styling contract": bg-card
	// with border-border at rounded-lg / p-6 (the lg spacing step). Width is
	// deliberately NOT baked in here — every caller passes its own max-w-md
	// (single-purpose modals) or max-w-lg (two-step / manage-sources) via
	// `class`, per the UI-SPEC's per-modal width table.
	import { Dialog as DialogPrimitive } from "bits-ui";
	import { cn } from "$lib/utils.js";
	import type { WithoutChildrenOrChild } from "$lib/utils.js";
	import type { ComponentProps } from "svelte";

	let {
		ref = $bindable(null),
		class: className,
		children,
		portalProps,
		...restProps
	}: DialogPrimitive.ContentProps & {
		portalProps?: WithoutChildrenOrChild<ComponentProps<typeof DialogPrimitive.Portal>>;
	} = $props();
</script>

<DialogPrimitive.Portal {...portalProps}>
	<DialogPrimitive.Overlay
		data-slot="dialog-overlay"
		class="data-open:animate-in data-open:fade-in-0 data-closed:animate-out data-closed:fade-out-0 fixed inset-0 z-50 bg-background/80"
	/>
	<DialogPrimitive.Content
		bind:ref
		data-slot="dialog-content"
		class={cn(
			"data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95 data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95 fixed top-1/2 left-1/2 z-50 w-full -translate-x-1/2 -translate-y-1/2 rounded-lg border border-border bg-card p-6 text-foreground shadow-md outline-none",
			className
		)}
		{...restProps}
	>
		{@render children?.()}
	</DialogPrimitive.Content>
</DialogPrimitive.Portal>
