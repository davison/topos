<script lang="ts">
	// Surface tokens match popover-content.svelte exactly (same neutral
	// bg-popover/border-border surface, p-1 per 07-UI-SPEC.md's "Styling
	// contract"). The popover token (09-UI-SPEC.md Fix 8) keeps a floating
	// menu visibly lighter than the pane it opens over — the pane surface
	// token is what every pane behind it already uses. max-h-80 +
	// overflow-y-auto is the height cap the E1/E3 overflow backstop truths
	// require: a long menu (many webspaces, many unconfigured plugin
	// types) scrolls inside the menu rather than growing the dropdown past
	// the viewport.
	import { DropdownMenu as DropdownMenuPrimitive } from "bits-ui";
	import { cn } from "$lib/utils.js";
	import type { WithoutChildrenOrChild } from "$lib/utils.js";
	import type { ComponentProps } from "svelte";

	let {
		ref = $bindable(null),
		class: className,
		sideOffset = 6,
		side = "bottom",
		align = "start",
		children,
		portalProps,
		...restProps
	}: DropdownMenuPrimitive.ContentProps & {
		portalProps?: WithoutChildrenOrChild<ComponentProps<typeof DropdownMenuPrimitive.Portal>>;
	} = $props();
</script>

<DropdownMenuPrimitive.Portal {...portalProps}>
	<DropdownMenuPrimitive.Content
		bind:ref
		data-slot="dropdown-menu-content"
		{sideOffset}
		{side}
		{align}
		class={cn(
			"data-open:animate-in data-open:fade-in-0 data-open:zoom-in-95 data-closed:animate-out data-closed:fade-out-0 data-closed:zoom-out-95 data-[side=bottom]:slide-in-from-top-2 data-[side=left]:slide-in-from-right-2 data-[side=right]:slide-in-from-left-2 data-[side=top]:slide-in-from-bottom-2 z-50 max-h-80 min-w-56 origin-(--bits-dropdown-menu-content-transform-origin) overflow-y-auto rounded-md border border-border bg-popover p-1 text-foreground shadow-md outline-none",
			className
		)}
		{...restProps}
	>
		{@render children?.()}
	</DropdownMenuPrimitive.Content>
</DropdownMenuPrimitive.Portal>
