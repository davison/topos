<script lang="ts">
	import { Badge } from '$lib/components/ui/badge/index.js';
	import {
		Tooltip,
		TooltipContent,
		TooltipProvider,
		TooltipTrigger
	} from '$lib/components/ui/tooltip/index.js';
	import { Checkbox } from '$lib/components/ui/checkbox/index.js';
	import Thumbnail from './Thumbnail.svelte';
	import PluginIcon from './PluginIcon.svelte';
	import { formatItemDate, parseSnippet, highlightText } from '$lib/format';
	import { cn } from '$lib/utils.js';
	import type { StreamItem } from '$lib/api';

	// `snippet` (03-04, search results): when a non-empty string, replaces
	// the preview region with parseSnippet's segmented rendering instead of
	// item.preview. Absent, this component is byte-identical to Phase 1/2.
	// Present but empty, the preview region is omitted entirely, the same
	// degrade the plain item.preview branch already applies.
	//
	// `searchQuery` (UI-09, G-06-1): the active in-webspace search string,
	// supplied by SearchResults.svelte so a result row can highlight its
	// own title — defaults to the empty string so the unfiltered stream
	// (which never passes it) is byte-identical to before this class was
	// shared. Both the title and the snippet's matched segments render
	// through the shared .search-highlight class (app.css) — the same
	// amber treatment the detail pane and the kernel rendition iframe use,
	// not a bolder font weight (03-UI-SPEC.md's retired weight-only rule,
	// superseded by this plan).
	// bulkSelected/bulkModeActive/onbulktoggle (13-UI-SPEC.md E1, KERN-09/
	// KERN-10): the desktop-only multi-select axis, deliberately orthogonal
	// to selected/onselect above — a row can be simultaneously the open
	// item AND bulk-selected, rendering both signals additively (fill+
	// checkbox vs. left-border accent), never conflicting. onbulktoggle is
	// optional so a caller that has no bulk-select surface at all (e.g.
	// SearchResults.svelte, out of this plan's scope) gets ordinary
	// plain-click behaviour with no extra branch to opt out of — a
	// modifier-click with no handler wired is simply a no-op, never an
	// error and never a fallback to onselect (D-01: only a plain click
	// opens the detail pane).
	let {
		item,
		selected = false,
		onselect,
		stale = false,
		sourceDisplayName = '',
		plugin = '',
		snippet,
		searchQuery = '',
		bulkSelected = false,
		bulkModeActive = false,
		onbulktoggle
	}: {
		item: StreamItem;
		selected?: boolean;
		onselect: () => void;
		stale?: boolean;
		sourceDisplayName?: string;
		// plugin (09-02-PLAN.md Task 4 checkpoint follow-up): the item's
		// source instance's configured plugin binary name (e.g.
		// "topos-plugin-paperless"), resolved by the caller from
		// sourcesByInstance — the same map sourceDisplayName above already
		// resolves from. Renders this row's own plugin identity icon so a
		// mixed, cross-source stream/search-results pane stays scannable by
		// source at a glance, without opening the detail pane. Defaults to
		// '' (PluginIcon's own Puzzle fallback) for any caller that hasn't
		// threaded this prop, and for a historic item whose source instance
		// has since been removed from config (absent from sourcesByInstance).
		plugin?: string;
		snippet?: string;
		searchQuery?: string;
		bulkSelected?: boolean;
		bulkModeActive?: boolean;
		onbulktoggle?: (id: string, mode: 'toggle' | 'range') => void;
	} = $props();

	// The three-way click/keyboard branch (13-UI-SPEC.md E1 "Trigger
	// rule"): shared by both onclick and onkeydown below so Enter/Space on
	// a focused row (native <button> activation) and a real pointer click
	// resolve identically — a KeyboardEvent carries the same
	// ctrlKey/metaKey/shiftKey modifier-state properties a MouseEvent does,
	// so one function serves both without branching on event type.
	function handleActivate(modifiers: { ctrlKey: boolean; metaKey: boolean; shiftKey: boolean }) {
		if (modifiers.ctrlKey || modifiers.metaKey) {
			onbulktoggle?.(item.id, 'toggle');
			return;
		}
		if (modifiers.shiftKey) {
			onbulktoggle?.(item.id, 'range');
			return;
		}
		onselect();
	}

	function handleRowClick(event: MouseEvent) {
		handleActivate(event);
	}

	// Manual Enter/Space activation (13-UI-SPEC.md E1 deviation, Rule 1):
	// the row's root element below is a `<div role="button">`, not a real
	// `<button>` — bits-ui's stock Checkbox (added by this plan) renders
	// its own real `<button role="checkbox">`, and a `<button>` must never
	// contain another interactive `<button>` descendant (invalid HTML;
	// also breaks click-event semantics, since the inner button's click
	// would bubble straight into the outer button's own handler with no
	// way to distinguish "the user clicked the checkbox" from "the user
	// clicked the row"). A native `<button>` fires `click` on Enter/Space
	// automatically; a plain `<div>` does not, so this restores that one
	// piece of behaviour explicitly.
	function handleRowKeydown(event: KeyboardEvent) {
		if (event.key !== 'Enter' && event.key !== ' ') return;
		event.preventDefault();
		handleActivate(event);
	}
</script>

<!--
  Fixed-height row (.stream-row-surface in app.css): a leading 40x52
  thumbnail, a one-line title, a clipped metadata strip (date + tag
  pills, .stream-row-meta) and a two-line preview clamp — never grows
  with tag count or preview length, so a long list scrolls at a
  constant rhythm. 152px at 768px and above, byte-identical to before
  this phase (D-05).

  Below 768px (max-md:, D-05/D-06/D-07): a compact 60px, three-line
  budget — 4px padding (p-1) + 16px title + 4px gap + 14px meta + 4px
  gap + 14px snippet = 60px. No thumbnail, no tag pills (not part of
  D-06's explicit composition); the meta line switches from the
  desktop's flex-wrap badge-clip to a flex-nowrap single truncating
  strip (RESEARCH Pitfall 4 — a different layout strategy, not a
  smaller version of the desktop one).

  Accent color use: the 2px left border below is the ONLY accent
  (--primary/--ring, see app.css) mark in this row, applied only when
  selected — the accent color is otherwise absent from this file. The
  focus-visible ring is the accent's other sanctioned use. Tag pills
  render with the Badge "secondary" variant (neutral palette), the
  metadata text uses the neutral muted-foreground token, and the D-10
  stale marker below uses the dedicated --warning token — never accent.

  13-UI-SPEC.md E1: this row's root is a `<div role="button">`, not a
  `<button>` — see handleRowKeydown's doc comment above for why (the
  leading Checkbox slot below is itself a real interactive <button>, and
  a <button> may never contain another). `group` enables the checkbox
  slot's hover/focus-reveal below, the same discipline SourceChip.svelte's
  own overflow-menu trigger already establishes in this codebase.
  bg-secondary/60 (new when bulkSelected) and the pre-existing
  border-l-primary (when selected) render ADDITIVELY — different visual
  channels (fill vs. border), so a row that is simultaneously open and
  bulk-selected shows both without either one masking the other.
-->
<div
	role="button"
	tabindex="0"
	onclick={handleRowClick}
	onkeydown={handleRowKeydown}
	aria-pressed={selected}
	data-item-id={item.id}
	class={cn(
		'group stream-row-surface flex w-full items-start gap-4 overflow-hidden rounded-lg border border-border bg-card p-4 text-left transition-colors hover:bg-card/80 max-md:h-[60px] max-md:gap-0 max-md:p-1',
		'focus-visible:ring-ring focus-visible:ring-offset-background focus-visible:ring-2 focus-visible:ring-offset-2 focus-visible:outline-none',
		selected && 'border-l-primary border-l-2',
		bulkSelected && 'bg-secondary/60'
	)}
>
	<!-- Leading checkbox slot (13-UI-SPEC.md E1): desktop/pointer-only
	     (max-md:hidden, matching the thumbnail's own breakpoint below),
	     opacity-0 at rest and revealed on row hover/focus or whenever
	     bulkModeActive is true (any item anywhere in the stream is
	     selected) — the same hover/focus-revealed-affordance discipline
	     SourceChip.svelte's own overflow menu already establishes.
	     onclick stopPropagation is load-bearing: without it, a click on
	     the checkbox bubbles into this row's own onclick above (since the
	     checkbox is a nested <button>, not merely decorative markup),
	     which would ALSO fire handleActivate's plain-click branch and open
	     the detail pane on every checkbox click. -->
	<div
		class={cn(
			'flex size-9 shrink-0 items-center justify-center transition-opacity max-md:hidden',
			bulkModeActive
				? 'opacity-100'
				: 'opacity-0 group-hover:opacity-100 group-has-[:focus-visible]:opacity-100'
		)}
	>
		<Checkbox
			checked={bulkSelected}
			onCheckedChange={() => onbulktoggle?.(item.id, 'toggle')}
			onclick={(event: MouseEvent) => event.stopPropagation()}
			aria-label={`Select ${item.title}`}
		/>
	</div>

	<div class="max-md:hidden">
		<Thumbnail {item} />
	</div>

	<div class="min-w-0 flex-1">
		<!-- One-line title, ellipsis-truncated (heading role: 20px/600/1.2).
		     UI-09 (G-06-1): highlighted through the same shared
		     .search-highlight class as the detail-pane title and this
		     row's own snippet below — a title-only match is a routine FTS
		     outcome, so it must be visually explained here too. -->
		<p
			class="truncate text-[20px] leading-[1.2] font-semibold text-foreground max-md:text-[16px] max-md:leading-none"
		>
			{#each highlightText(item.title, searchQuery) as segment, i (i)}
				<span class={segment.match ? 'search-highlight' : undefined}>{segment.text}</span>
			{/each}
		</p>

		<!-- Clipped metadata strip: date + one Badge per tag (label role:
		     14px/400/1.4) at desktop size. Below 768px (max-md:, D-07):
		     switches from flex-wrap badge-clip to a flex-nowrap single
		     truncating strip — icon, then one jointly-truncating
		     name/group-label text unit, then date, then the stale dot
		     last, so the dot is never the element that gets clipped
		     (RESEARCH Pitfall 4 / planner_resolutions R2). -->
		<div
			class="stream-row-meta mt-1 flex flex-wrap items-center gap-2 text-[14px] leading-[1.4] text-muted-foreground max-md:mt-1 max-md:flex-nowrap max-md:gap-1.5 max-md:overflow-hidden max-md:leading-none"
		>
			<!-- Source identity icon (09-02-PLAN.md Task 4 checkpoint
			     follow-up, additive metadata alongside the leading
			     Thumbnail — never a replacement for it): reuses PluginIcon's
			     existing kernel-served fallback chain and decorative
			     alt="" unchanged, at the same size-3.5 the chip uses. A
			     native title gives the source name on hover without a
			     second Tooltip primitive per row. Rendered first so it
			     reads as this row's own source marker, ahead of sender/date. -->
			<span class="shrink-0" title={sourceDisplayName || undefined}>
				<PluginIcon {plugin} size="size-3.5" />
			</span>
			<!-- Compact-only (max-md:, D-07): the source display name,
			     visible as TEXT here for the first time — today it exists
			     only as the icon's hover `title`, useless on a touchscreen
			     with no hover. Combined with the group label into ONE text
			     node inside one truncating span so the two pieces degrade
			     together (jointly truncate) rather than one vanishing while
			     the other stays full-length (planner_resolutions R2). -->
			<span class="hidden min-w-0 flex-1 truncate max-md:block">
				{sourceDisplayName}{item.group_label ? ` · ${item.group_label}` : ''}
			</span>
			<!-- Sender (item.group_label — "chat thread / mail conversation"
			     in the wire contract): plain text, no "From" prefix, never
			     the accent color, omitted entirely when empty so paperless
			     and SilverBullet rows (which never populate this field) are
			     visually unchanged. Rendered as the FIRST entry, before the
			     date, per 03-UI-SPEC.md's E3 resolution. Hidden below
			     768px — the combined span above carries the group label at
			     compact size instead. -->
			{#if item.group_label}
				<span class="shrink-0 max-md:hidden">{item.group_label}</span>
			{/if}
			<span class="shrink-0">{formatItemDate(item.timestamp_unix)}</span>
			{#if stale}
				<!-- D-10: tertiary, per-row proof the affected source's items
				     are still visible (not silently dropped) — subtle, never
				     a banner-level alarm, and never the accent color. -->
				<TooltipProvider>
					<Tooltip>
						<TooltipTrigger>
							{#snippet child({ props })}
								<span
									{...props}
									class="bg-warning size-2 shrink-0 rounded-full"
									aria-label={`${sourceDisplayName} is currently unreachable — this item may be out of date`}
								></span>
							{/snippet}
						</TooltipTrigger>
						<TooltipContent>
							{sourceDisplayName} is currently unreachable — this item may be out of date.
						</TooltipContent>
					</Tooltip>
				</TooltipProvider>
			{/if}
			<!-- Tag pills: not part of D-06's compact composition (thumbnail
			     and tag pills are the two things dropped at compact size) —
			     the full tag list stays visible in the detail pane,
			     unchanged. Hidden as a group below 768px. -->
			<span class="contents max-md:hidden">
				{#each item.labels as label (label)}
					<Badge variant="secondary">{label}</Badge>
				{/each}
			</span>
		</div>

		<!--
		  Two-line preview clamp (body role: 16px/400/1.5). Omitted entirely
		  when the preview is empty — a document with no OCR text degrades
		  to title + metadata only, not an empty or zero-height block. The
		  row keeps its fixed height regardless (.stream-row-surface).

		  Below 768px (max-md:, D-06): single-line clamp at the compact
		  14px/leading-none budget instead of the desktop two-line clamp.
		  Both branches below move together (mt-1 line-clamp-1 text-[14px]
		  leading-none) so a search-result row and a stream row never
		  render at different heights (D-08).
		-->
		{#if snippet !== undefined}
			{#if snippet}
				<p
					class="mt-1 line-clamp-2 text-[16px] leading-[1.5] text-foreground max-md:mt-1 max-md:line-clamp-1 max-md:text-[14px] max-md:leading-none"
				>
					{#each parseSnippet(snippet) as segment, i (i)}
						<span class={segment.match ? 'search-highlight' : undefined}>{segment.text}</span>
					{/each}
				</p>
			{/if}
		{:else if item.preview}
			<p
				class="mt-1 line-clamp-2 text-[16px] leading-[1.5] text-foreground max-md:mt-1 max-md:line-clamp-1 max-md:text-[14px] max-md:leading-none"
			>
				{item.preview}
			</p>
		{/if}
	</div>
</div>
