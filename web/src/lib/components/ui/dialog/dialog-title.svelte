<script lang="ts">
	// Heading role (20px/600/1.2) — every modal title in this phase maps
	// onto this role (07-UI-SPEC.md Typography).
	import { Dialog as DialogPrimitive } from "bits-ui";
	import { cn } from "$lib/utils.js";

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: DialogPrimitive.TitleProps = $props();
</script>

<DialogPrimitive.Title
	bind:ref
	data-slot="dialog-title"
	class={cn("text-[20px] leading-[1.2] font-semibold text-foreground", className)}
	{...restProps}
/>
