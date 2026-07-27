<script lang="ts">
	import { Badge } from '$lib/components/ui/badge/index.js';
	import Thumbnail from './Thumbnail.svelte';
	import { formatItemDate } from '$lib/format';
	import { cn } from '$lib/utils.js';
	import type { StreamItem } from '$lib/api';

	let {
		item,
		selected = false,
		onselect
	}: { item: StreamItem; selected?: boolean; onselect: () => void } = $props();
</script>

<!--
  Fixed-height row (.stream-row-surface in app.css): a leading 40x52
  thumbnail, a one-line title, a clipped metadata strip (date + tag
  pills, .stream-row-meta) and a two-line preview clamp — never grows
  with tag count or preview length, so a long list scrolls at a
  constant rhythm.

  Accent color use: the 2px left border below is the ONLY accent
  (--primary/--ring, see app.css) mark in this row, applied only when
  selected — the accent color is otherwise absent from this file. The
  focus-visible ring is the accent's other sanctioned use. Tag pills
  render with the Badge "secondary" variant (neutral palette) and the
  metadata text uses the neutral muted-foreground token, never accent.
-->
<button
	type="button"
	onclick={onselect}
	aria-pressed={selected}
	class={cn(
		'stream-row-surface flex w-full items-start gap-4 overflow-hidden rounded-lg border border-border bg-card p-4 text-left transition-colors hover:bg-card/80',
		'focus-visible:ring-ring focus-visible:ring-offset-background focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none',
		selected && 'border-l-primary border-l-2'
	)}
>
	<Thumbnail {item} />

	<div class="min-w-0 flex-1">
		<!-- One-line title, ellipsis-truncated (heading role: 20px/600/1.2). -->
		<p class="truncate text-[20px] leading-[1.2] font-semibold text-foreground">
			{item.title}
		</p>

		<!-- Clipped metadata strip: date + one Badge per tag (label role: 14px/400/1.4). -->
		<div class="stream-row-meta mt-1 flex flex-wrap items-center gap-2 text-[14px] leading-[1.4] text-muted-foreground">
			<span class="shrink-0">{formatItemDate(item.timestamp_unix)}</span>
			{#each item.labels as label (label)}
				<Badge variant="secondary">{label}</Badge>
			{/each}
		</div>

		<!--
		  Two-line preview clamp (body role: 16px/400/1.5). Omitted entirely
		  when the preview is empty — a document with no OCR text degrades
		  to title + metadata only, not an empty or zero-height block. The
		  row keeps its fixed height regardless (.stream-row-surface).
		-->
		{#if item.preview}
			<p class="mt-1 line-clamp-2 text-[16px] leading-[1.5] text-foreground">
				{item.preview}
			</p>
		{/if}
	</div>
</button>
