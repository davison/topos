<script lang="ts">
	// Label role (14px/400/1.4) per 07-UI-SPEC.md Typography. Hover/focus
	// uses the neutral --muted surface convention shared with every other
	// menu in the app (Color table: "not used for dropdown menu item hover
	// states") — never the accent/primary tone, which stays reserved for
	// CTAs, focus rings and the selected stream row.
	import { DropdownMenu as DropdownMenuPrimitive } from "bits-ui";
	import { cn } from "$lib/utils.js";

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: DropdownMenuPrimitive.ItemProps = $props();
</script>

<DropdownMenuPrimitive.Item
	bind:ref
	data-slot="dropdown-menu-item"
	class={cn(
		"relative flex cursor-default items-center gap-2 rounded-sm px-2 py-1.5 text-[14px] leading-[1.4] text-foreground outline-none select-none data-highlighted:bg-muted data-disabled:pointer-events-none data-disabled:opacity-50 [&_svg]:size-4 [&_svg]:shrink-0 [&_svg]:text-muted-foreground",
		className
	)}
	{...restProps}
>
	{@render children?.()}
</DropdownMenuPrimitive.Item>
