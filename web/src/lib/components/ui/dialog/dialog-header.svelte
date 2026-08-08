<script lang="ts">
	// Plain layout div (no bits-ui primitive) — matches shadcn's own
	// DialogHeader shape, which is pure spacing/typography wiring around
	// whatever DialogTitle (and, optionally, caller-supplied helper text)
	// is rendered inside it.
	import type { HTMLAttributes } from "svelte/elements";
	import { cn, type WithElementRef } from "$lib/utils.js";

	let {
		ref = $bindable(null),
		class: className,
		children,
		...restProps
	}: WithElementRef<HTMLAttributes<HTMLDivElement>> = $props();
</script>

<div
	bind:this={ref}
	data-slot="dialog-header"
	class={cn("flex flex-col gap-1.5", className)}
	{...restProps}
>
	{@render children?.()}
</div>
