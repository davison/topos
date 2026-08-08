<script lang="ts">
	// Plain layout div — every modal in this phase renders exactly two
	// footer buttons (Cancel + a primary submit), right-aligned.
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
	data-slot="dialog-footer"
	class={cn("mt-6 flex items-center justify-end gap-2", className)}
	{...restProps}
>
	{@render children?.()}
</div>
