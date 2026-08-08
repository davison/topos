<script lang="ts">
	// Body role (16px/400/1.5), text-muted-foreground — every AlertDialog
	// body copy in this phase (07-UI-SPEC.md Destructive Confirmation
	// Contract) maps onto this role.
	import { AlertDialog as AlertDialogPrimitive } from "bits-ui";
	import { cn } from "$lib/utils.js";

	let {
		ref = $bindable(null),
		class: className,
		...restProps
	}: AlertDialogPrimitive.DescriptionProps = $props();
</script>

<AlertDialogPrimitive.Description
	bind:ref
	data-slot="alert-dialog-description"
	class={cn("text-[16px] leading-[1.5] text-muted-foreground", className)}
	{...restProps}
/>
