<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import {
		Tooltip,
		TooltipContent,
		TooltipProvider,
		TooltipTrigger
	} from '$lib/components/ui/tooltip/index.js';
	import {
		DropdownMenu,
		DropdownMenuTrigger,
		DropdownMenuContent,
		DropdownMenuItem,
		DropdownMenuSeparator
	} from '$lib/components/ui/dropdown-menu/index.js';
	import RefreshCw from '@lucide/svelte/icons/refresh-cw';
	import EllipsisVertical from '@lucide/svelte/icons/ellipsis-vertical';
	import Pencil from '@lucide/svelte/icons/pencil';
	import QrCode from '@lucide/svelte/icons/qr-code';
	import PluginIcon from '$lib/components/PluginIcon.svelte';
	import { cn } from '$lib/utils.js';
	import { healthTone, formatRelativeTime, type HealthTone } from '$lib/format';
	import { WHATSAPP_SOURCE_TYPE } from '$lib/plugin-fields';
	import type { SourceStatus } from '$lib/api';

	// The single merged per-instance affordance (D-01): health dot,
	// display name, a real-button filter toggle, and a hover/focus-revealed
	// refresh control — replacing the retired SourceHealthChip.svelte +
	// SourceFilterChips.svelte pair. This exact component renders both
	// inline in WebspaceHeader's chip row and inside the overflow popover
	// (06-UI-SPEC.md "Header Redesign") — reusing it unforked in both
	// places is what keeps truncation and per-instance isolation identical
	// regardless of where a given chip is currently visible.
	//
	// D-12 (07-04-PLAN.md Task 3) adds a third control: an edit menu
	// trailing the refresh button, offering Edit connection…/Edit match
	// settings…/Remove from this webspace via `onedit`. A measurement
	// clone (WebspaceHeader.svelte's invisible `measureEl` row) passes a
	// no-op `onedit` — see this file's own guard, chip-edit-menu.test.ts.
	//
	// D-03 (08-04-PLAN.md Task 2) widens the menu with a fourth entry,
	// "Re-link…", offered only when `source.source_type` is WhatsApp's own
	// Describe-reported kind — every other plugin type has nothing to
	// re-link, and an inert menu entry is worse than an absent one.
	//
	// 09-01-PLAN.md Task 3 (09-UI-SPEC.md Fix 10) adds the plugin's own
	// identity icon (PluginIcon, kernel-served, mandatory Puzzle fallback)
	// between the health dot and the display name — chip now reads
	// [dot][icon][name]. Only this one addition; the tooltip copy and the
	// trailing-control rework belong to 09-05.
	let {
		source,
		selected,
		onfilter,
		onrefresh,
		onedit,
		busy = false
	}: {
		source: SourceStatus;
		selected: boolean;
		onfilter: (name: string) => void;
		onrefresh: (name: string) => void;
		onedit: (name: string, kind: 'connection' | 'match' | 'relink' | 'remove') => void;
		// busy (07-05-PLAN.md Task 2, the shared save/reload state pattern's
		// in-flight rule — E6 "the initiating control disables in flight")
		// disables ONLY the "Remove from this webspace" item below: it is
		// the one write this menu can trigger directly, sharing the route's
		// own filterBusy flag with the modal-less filter-save/-remove path
		// (both write through the identical putConfig seam). Edit
		// connection…/Edit match settings… merely open a modal — opening
		// one while an unrelated write is in flight is harmless, so neither
		// is gated on this flag.
		busy?: boolean;
	} = $props();

	let tone = $derived(healthTone(source));

	// isWhatsApp gates the Re-link… menu entry (D-03) — keyed on
	// source_type, the Describe-reported plugin KIND GET /api/sources
	// actually exposes, never on a plugin binary name this component has
	// no other reason to know.
	let isWhatsApp = $derived(source.source_type === WHATSAPP_SOURCE_TYPE);

	const DOT_TONE_CLASS: Record<HealthTone, string> = {
		success: 'bg-success',
		warning: 'bg-warning',
		destructive: 'bg-destructive',
		unknown: 'bg-muted-foreground'
	};

	// Copywriting Contract (06-UI-SPEC.md, revised 09-UI-SPEC.md Fix 3): the
	// four Phase 2 tooltip branches carried forward verbatim (D-04, no
	// rewording), plus one new branch this phase adds — while
	// `source.syncing` is true the tooltip reads "{display_name} —
	// syncing…", checked before the four last-known-state branches since a
	// source can be mid-sync regardless of its last recorded outcome. The
	// old inline "Syncing…" text label (SourceHealthChip.svelte) is
	// retired; the spinning refresh icon is now the sole in-place syncing
	// indicator, kept compact at scale.
	//
	// Fix 3: `formatRelativeTime` (Intl.RelativeTimeFormat, numeric:
	// 'auto') already returns a complete phrase — "5 minutes ago" for a
	// numeric delta, but also "yesterday", "last week" and "now" for its
	// special-cased deltas. The success/warning branches use `${relative}`
	// verbatim with NO appended word — appending " ago" was wrong in every
	// case, not just the numeric-delta ones ("synced yesterday ago" was a
	// latent instance of the identical bug).
	let tooltipText = $derived.by(() => {
		if (source.syncing) return `${source.display_name} — syncing…`;
		const relative = formatRelativeTime(source.last_sync_unix);
		switch (tone) {
			case 'success':
				return `${source.display_name} — synced ${relative}`;
			case 'warning':
				return `${source.display_name} — last error ${relative}: ${source.last_error}`;
			case 'destructive':
				return `${source.display_name} — unreachable since ${relative}`;
			default:
				return `${source.display_name} — not yet synced`;
		}
	});

	// stopPropagation so a refresh click never also toggles the chip's own
	// filter state — the two controls are siblings, not nested.
	function handleRefreshClick(event: MouseEvent) {
		event.stopPropagation();
		onrefresh(source.name);
	}

	// stopPropagation before anything else — this is the D-12 versus Phase
	// 6 D-01 collision, and the single most important line in this
	// component's edit-menu control: opening the menu must never also
	// toggle the chip's filter state. bits-ui's own trigger props are
	// still invoked afterward (props.onclick?.(event)) so the menu's own
	// interaction handling (its VoiceOver click-detail-0 case; real mouse
	// opens are driven by the trigger's own onpointerdown, untouched here)
	// keeps working.
	function handleEditClick(event: MouseEvent, triggerOnClick?: (e: MouseEvent) => void) {
		event.stopPropagation();
		triggerOnClick?.(event);
	}
</script>

<div
	class={cn(
		'group flex h-11 shrink-0 items-center rounded-full border border-border bg-card pr-1',
		selected && 'border-primary bg-primary'
	)}
>
	<TooltipProvider>
		<Tooltip>
			<TooltipTrigger>
				{#snippet child({ props })}
					<button
						{...props}
						type="button"
						aria-pressed={selected}
						onclick={() => onfilter(source.name)}
						class="flex max-w-48 items-center gap-1.5 self-stretch rounded-full pr-1.5 pl-2.5"
					>
						<span
							class={cn(
								'size-2 shrink-0 rounded-full',
								DOT_TONE_CLASS[tone],
								selected && 'ring-1 ring-primary-foreground'
							)}
							aria-hidden="true"
						></span>
						<PluginIcon plugin={source.plugin} size="size-3.5 shrink-0" />
						<span
							class={cn(
								'truncate text-[14px] leading-[1.4]',
								selected ? 'text-primary-foreground' : 'text-foreground'
							)}
							title={source.display_name}>{source.display_name}</span
						>
					</button>
				{/snippet}
			</TooltipTrigger>
			<TooltipContent>{tooltipText}</TooltipContent>
		</Tooltip>
	</TooltipProvider>

	<Button
		variant="ghost"
		size="icon"
		class={cn(
			'size-8 rounded-full opacity-0 transition-opacity group-hover:opacity-100 group-has-[:focus-visible]:opacity-100',
			source.syncing && 'opacity-100',
			selected && 'text-primary-foreground hover:bg-primary-foreground/20 hover:text-primary-foreground'
		)}
		aria-label={`Refresh ${source.display_name}`}
		onclick={handleRefreshClick}
	>
		<RefreshCw class={cn('size-4', source.syncing && 'animate-spin')} />
	</Button>

	<DropdownMenu>
		<DropdownMenuTrigger>
			{#snippet child({ props })}
				<Button
					{...props}
					variant="ghost"
					size="icon"
					class={cn(
						'size-8 rounded-full opacity-0 transition-opacity group-hover:opacity-100 group-has-[:focus-visible]:opacity-100',
						selected &&
							'text-primary-foreground hover:bg-primary-foreground/20 hover:text-primary-foreground'
					)}
					aria-label={`Edit ${source.display_name}`}
					onclick={(event: MouseEvent) =>
						handleEditClick(event, (props as { onclick?: (e: MouseEvent) => void }).onclick)}
				>
					<EllipsisVertical class="size-4" />
				</Button>
			{/snippet}
		</DropdownMenuTrigger>
		<DropdownMenuContent>
			<DropdownMenuItem onSelect={() => onedit(source.name, 'connection')}>
				<Pencil aria-hidden="true" />
				Edit connection…
			</DropdownMenuItem>
			<DropdownMenuItem onSelect={() => onedit(source.name, 'match')}>
				<Pencil aria-hidden="true" />
				Edit match settings…
			</DropdownMenuItem>
			{#if isWhatsApp}
				<DropdownMenuItem onSelect={() => onedit(source.name, 'relink')}>
					<QrCode aria-hidden="true" />
					Re-link…
				</DropdownMenuItem>
			{/if}
			<DropdownMenuSeparator />
			<DropdownMenuItem
				class="text-foreground hover:text-destructive focus:text-destructive data-highlighted:text-destructive"
				disabled={busy}
				onSelect={() => onedit(source.name, 'remove')}
			>
				Remove from this webspace
			</DropdownMenuItem>
		</DropdownMenuContent>
	</DropdownMenu>
</div>
