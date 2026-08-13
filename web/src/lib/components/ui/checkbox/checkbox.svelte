<script lang="ts">
	// shadcn-svelte official registry `checkbox` block, installed the same
	// way every other web/src/lib/components/ui/* wrapper in this tree was
	// (12-04-PLAN.md Task 1) — unmodified stock recipe, no colour override.
	// The stock unchecked-border and checked-fill mapping onto the existing
	// --border/--primary tokens (see web/src/app.css) is correct as shipped.
	import { Checkbox as CheckboxPrimitive } from 'bits-ui';
	import Check from '@lucide/svelte/icons/check';
	import Minus from '@lucide/svelte/icons/minus';
	import { cn } from '$lib/utils.js';

	let {
		ref = $bindable(null),
		checked = $bindable(false),
		indeterminate = $bindable(false),
		class: className,
		...restProps
	}: CheckboxPrimitive.RootProps = $props();
</script>

<CheckboxPrimitive.Root
	bind:ref
	bind:checked
	bind:indeterminate
	data-slot="checkbox"
	class={cn(
		'peer size-4 shrink-0 rounded-[4px] border border-input shadow-xs outline-none transition-shadow focus-visible:ring-[3px] focus-visible:ring-ring/50 disabled:cursor-not-allowed disabled:opacity-50 data-[state=checked]:border-primary data-[state=checked]:bg-primary data-[state=checked]:text-primary-foreground',
		className
	)}
	{...restProps}
>
	{#snippet children({ checked, indeterminate })}
		<div
			data-slot="checkbox-indicator"
			class="flex items-center justify-center text-current transition-none"
		>
			{#if indeterminate}
				<Minus class="size-3.5" />
			{:else}
				<Check class={cn('size-3.5', !checked && 'text-transparent')} />
			{/if}
		</div>
	{/snippet}
</CheckboxPrimitive.Root>
