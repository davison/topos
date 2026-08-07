<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import Filter from '@lucide/svelte/icons/filter';
	import X from '@lucide/svelte/icons/x';
	import { cn } from '$lib/utils.js';

	// One permanent-filter chip (D-16-D-19): a saved search term, stacked
	// AND with every other active filter, removable independently. Shape
	// is `rounded-md` — deliberately NOT SourceChip's `rounded-full` — the
	// single geometric difference that makes a filter chip legible as "a
	// different kind of thing" at a glance (07-UI-SPEC.md "Filter chip
	// row"), without inventing a new colour.
	let {
		term,
		disabled,
		onremove
	}: {
		term: string;
		disabled: boolean;
		onremove: (term: string) => void;
	} = $props();

	// stopPropagation first, matching SourceChip's refresh-click discipline
	// (handleRefreshClick) — a secondary control nested inside a chip must
	// never also trigger the chip's own semantics, even though this chip
	// (unlike SourceChip) has no click-toggle behavior of its own to guard
	// against; keeping the same isolation convention here is deliberate
	// consistency, not defensive-for-no-reason.
	function handleRemoveClick(event: MouseEvent) {
		event.stopPropagation();
		onremove(term);
	}
</script>

<div
	class={cn(
		'flex h-8 shrink-0 items-center gap-1 rounded-md border border-border bg-muted pr-1 pl-2.5'
	)}
>
	<Filter class="size-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
	<span class="truncate text-[14px] leading-[1.4] text-foreground">"{term}"</span>
	<Button
		variant="ghost"
		size="icon"
		class="size-8 shrink-0 rounded-md"
		aria-label={`Remove filter ${term}`}
		{disabled}
		onclick={handleRemoveClick}
	>
		<X class="size-4" />
	</Button>
</div>
