<script lang="ts">
	// Renders as a Button accepting `variant` so a destructive confirmation
	// passes variant="destructive" while still resolving every colour
	// through button.svelte's own token-driven buttonVariants — no literal
	// colour lives in this file.
	import { AlertDialog as AlertDialogPrimitive } from "bits-ui";
	import { buttonVariants, type ButtonVariant } from "$lib/components/ui/button/index.js";
	import { cn } from "$lib/utils.js";

	let {
		ref = $bindable(null),
		class: className,
		variant = "default",
		...restProps
	}: AlertDialogPrimitive.ActionProps & { variant?: ButtonVariant } = $props();
</script>

<AlertDialogPrimitive.Action
	bind:ref
	data-slot="alert-dialog-action"
	class={cn(buttonVariants({ variant }), className)}
	{...restProps}
/>
