<script lang="ts">
	// Same secondary-surface styling contract as dialog-content.svelte
	// (bg-card/border-border, rounded-lg/p-6) — an AlertDialog is a Dialog
	// variant that cannot be dismissed by an outside click/Escape (bits-ui's
	// own AlertDialog primitive enforces that), so every destructive
	// confirmation in this phase renders through this file rather than the
	// plain Dialog.
	import { AlertDialog as AlertDialogPrimitive } from "bits-ui";
	import { cn } from "$lib/utils.js";
	import type { WithoutChildrenOrChild } from "$lib/utils.js";
	import type { ComponentProps } from "svelte";

	let {
		ref = $bindable(null),
		class: className,
		children,
		portalProps,
		...restProps
	}: AlertDialogPrimitive.ContentProps & {
		portalProps?: WithoutChildrenOrChild<ComponentProps<typeof AlertDialogPrimitive.Portal>>;
	} = $props();
</script>

<AlertDialogPrimitive.Portal {...portalProps}>
	<AlertDialogPrimitive.Overlay
		data-slot="alert-dialog-overlay"
		class="data-open:animate-in data-open:fade-in-0 data-closed:animate-out data-closed:fade-out-0 fixed inset-0 z-50 bg-background/80"
	/>
	<AlertDialogPrimitive.Content
		bind:ref
		data-slot="alert-dialog-content"
		class={cn(
			"data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95 data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95 fixed top-1/2 left-1/2 z-50 w-full max-w-md -translate-x-1/2 -translate-y-1/2 rounded-lg border border-border bg-card p-6 text-foreground shadow-md outline-none",
			className
		)}
		{...restProps}
	>
		{@render children?.()}
	</AlertDialogPrimitive.Content>
</AlertDialogPrimitive.Portal>
