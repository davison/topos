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
	import { cn } from '$lib/utils.js';
	import { healthTone, formatRelativeTime, type HealthTone } from '$lib/format';
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
	let {
		source,
		selected,
		onfilter,
		onrefresh,
		onedit
	}: {
		source: SourceStatus;
		selected: boolean;
		onfilter: (name: string) => void;
		onrefresh: (name: string) => void;
		onedit: (name: string, kind: 'connection' | 'match' | 'remove') => void;
	} = $props();

	let tone = $derived(healthTone(source));

	const DOT_TONE_CLASS: Record<HealthTone, string> = {
		success: 'bg-success',
		warning: 'bg-warning',
		destructive: 'bg-destructive',
		unknown: 'bg-muted-foreground'
	};

	// Copywriting Contract (06-UI-SPEC.md): the four Phase 2 tooltip
	// branches carried forward verbatim (D-04, no rewording), plus one new
	// branch this phase adds — while `source.syncing` is true the tooltip
	// reads "{display_name} — syncing…", checked before the four
	// last-known-state branches since a source can be mid-sync regardless
	// of its last recorded outcome. The old inline "Syncing…" text label
	// (SourceHealthChip.svelte) is retired; the spinning refresh icon is
	// now the sole in-place syncing indicator, kept compact at scale.
	let tooltipText = $derived.by(() => {
		if (source.syncing) return `${source.display_name} — syncing…`;
		const relative = formatRelativeTime(source.last_sync_unix);
		switch (tone) {
			case 'success':
				return `${source.display_name} — synced ${relative} ago`;
			case 'warning':
				return `${source.display_name} — last error ${relative} ago: ${source.last_error}`;
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
			<DropdownMenuSeparator />
			<DropdownMenuItem
				class="text-foreground hover:text-destructive focus:text-destructive data-highlighted:text-destructive"
				onSelect={() => onedit(source.name, 'remove')}
			>
				Remove from this webspace
			</DropdownMenuItem>
		</DropdownMenuContent>
	</DropdownMenu>
</div>
