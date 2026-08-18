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
	import ShieldCheck from '@lucide/svelte/icons/shield-check';
	import Copy from '@lucide/svelte/icons/copy';
	import Check from '@lucide/svelte/icons/check';
	import PluginIcon from '$lib/components/PluginIcon.svelte';
	import TrustBadge from '$lib/components/TrustBadge.svelte';
	import { cn } from '$lib/utils.js';
	import {
		healthTone,
		formatRelativeTime,
		shortHash,
		isAdvisoryOnly,
		type HealthTone
	} from '$lib/format';
	import { WHATSAPP_SOURCE_TYPE } from '$lib/plugin-fields';
	import type { SourceStatus } from '$lib/api';

	// The single merged per-instance affordance (D-01): health dot,
	// display name, a real-button filter toggle, and a hover/focus-revealed
	// overflow menu — replacing the retired SourceHealthChip.svelte +
	// SourceFilterChips.svelte pair. This exact component renders both
	// inline in WebspaceHeader's chip row and inside the overflow popover
	// (06-UI-SPEC.md "Header Redesign") — reusing it unforked in both
	// places is what keeps truncation and per-instance isolation identical
	// regardless of where a given chip is currently visible.
	//
	// D-12 (07-04-PLAN.md Task 3) adds a third control: an edit menu
	// trailing the (then-standalone) refresh button, offering Edit
	// connection…/Edit match settings…/Remove from this webspace via
	// `onedit`. A measurement clone (WebspaceHeader.svelte's invisible
	// `measureEl` row) passes a no-op `onedit` — see this file's own guard,
	// chip-edit-menu.test.ts.
	//
	// D-03 (08-04-PLAN.md Task 2) widens the menu with a fourth entry,
	// "Re-link…", offered only when `source.source_type` is WhatsApp's own
	// Describe-reported kind — every other plugin type has nothing to
	// re-link, and an inert menu entry is worse than an absent one.
	//
	// 09-01-PLAN.md Task 3 (09-UI-SPEC.md Fix 10) adds the plugin's own
	// identity icon (PluginIcon, kernel-served, mandatory Puzzle fallback)
	// between the health dot and the display name — chip now reads
	// [dot][icon][name].
	//
	// 09-05-PLAN.md Task 2 (09-UI-SPEC.md Fix 5) folds the standalone
	// refresh Button into this menu as its first item — the chip now
	// reveals exactly ONE trailing hover/focus-visible control (the ⋮
	// trigger), not two. `onrefresh`/`onedit`'s signatures are unchanged,
	// so WebspaceHeader.svelte needs no edit.
	//
	// 11-01-PLAN.md Task 2 (11-UI-SPEC.md E2) wraps PluginIcon in
	// TrustBadge: an external-tier source's icon gains a small warning
	// glyph overlay, and its tooltip text gains an untrusted clause — a
	// trusted-tier chip's markup and tooltip text are byte-identical to
	// before this phase (D-06).
	//
	// 11-06-PLAN.md Task 1 (11-UI-SPEC.md E4/E5) widens onedit's kind
	// union with 'trust-update' and adds two conditional menu regions: a
	// leading "Trust updated binary…" item (E4) shown only while this
	// source's launch_failure carries the kernel-published pin-mismatch
	// signal, and a static pinned-hash footer (E5) shown for every
	// external-tier source that has a pin — a trusted-tier chip's menu
	// stays byte-identical to before this phase (D-04).
	let {
		source,
		selected,
		onfilter,
		onrefresh,
		onedit,
		busy = false,
		// shrinkable (14-06, G-14-2): for the measured header row ONLY. A
		// floor-forced chip at narrow widths must give way itself — its
		// display name truncating harder — rather than pushing the row's
		// reserved trailing controls (Refresh all, add-source "+") out of
		// the clipped row. Defaults to false so every other render site
		// (overflow-popover clones, measurement clones) keeps today's
		// no-shrink behaviour byte-identically.
		shrinkable = false
	}: {
		source: SourceStatus;
		selected: boolean;
		onfilter: (name: string) => void;
		onrefresh: (name: string) => void;
		onedit: (
			name: string,
			kind: 'connection' | 'match' | 'relink' | 'remove' | 'trust-update'
		) => void;
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
		shrinkable?: boolean;
	} = $props();

	let tone = $derived(healthTone(source));

	// chipDescId (14-02-PLAN.md Task 1/2, option-b): `$props.id()` rather than
	// an id derived from `source.name` — this component renders more than
	// one simultaneous instance for the SAME source (WebspaceHeader.svelte's
	// aria-hidden measurement clone, and the overflow-popover clone, both
	// alongside the visible row chip), so a name-derived id would collide
	// across instances and produce duplicate DOM ids. `$props.id()` is
	// SSR-safe and unique per component instance, which is exactly the
	// guarantee `aria-describedby` needs here.
	const chipDescId = $props.id();

	// isWhatsApp gates the Re-link… menu entry (D-03) — keyed on
	// source_type, the Describe-reported plugin KIND GET /api/sources
	// actually exposes, never on a plugin binary name this component has
	// no other reason to know.
	let isWhatsApp = $derived(source.source_type === WHATSAPP_SOURCE_TYPE);

	// isPinMismatch/isExternal (11-UI-SPEC.md E4/E5) mirror isWhatsApp's own
	// shape — keyed on kernel-published fields (launch_failure/tier), never
	// on a last_error string match (T-11-32's own guard, RESEARCH.md
	// Pitfall 2).
	let isPinMismatch = $derived(source.launch_failure === 'pin_mismatch');
	let isExternal = $derived(source.tier === 'external');

	// isManifestUnverified/isShadowed (13-06-PLAN.md, D-12/D-13/D-14) mirror
	// isPinMismatch's shape exactly — keyed on the kernel-published
	// launch_failure/launch_advisory fields alone, never on a last_error
	// string match, same T-11-32 discipline. isManifestUnverified never
	// gates the chip menu's re-pin action (that stays isPinMismatch-only,
	// D-13's "verification never demotes-and-runs" rule) and gains no new
	// menu item, dialog, or interstitial — the tooltip below is the entire
	// new surface.
	let isManifestUnverified = $derived(source.launch_failure === 'manifest_unverified');
	let isShadowed = $derived(source.launch_advisory === 'shadowed');

	// advisory (12-10-PLAN.md, G-12-1/G-12-3): the trimmed kernel-published
	// last_notice — today, a webspace's explicit match block that matched
	// none of this source's items. Keyed on the kernel-published field
	// alone, never on a string match against last_error (the T-11-32
	// discipline isPinMismatch above already documents, applying verbatim
	// here).
	let advisory = $derived((source.last_notice ?? '').trim());

	// advisoryOnly (12-11-PLAN.md, CR-01 gap closure): defers to format.ts's
	// isAdvisoryOnly rather than re-deriving the tooltip gate's own narrower
	// condition — see that function's docstring for the full precedence
	// reasoning and the tone === 'success' trap it exists to avoid.
	let advisoryOnly = $derived(isAdvisoryOnly(source));

	// hashCopied (E5): a lightweight, silent copy confirmation — the Copy
	// icon swaps to a check for ~1.5s, then reverts. No toast/alert; a
	// clipboard-API failure leaves this false and changes nothing visible,
	// since the full hash remains reachable via the footer's own title
	// attribute.
	let hashCopied = $state(false);

	async function copyPinnedHash() {
		if (!source.pinned_hash) return;
		try {
			await navigator.clipboard.writeText(source.pinned_hash);
			hashCopied = true;
			setTimeout(() => {
				hashCopied = false;
			}, 1500);
		} catch {
			// Silent no-op (E5) — the full hash stays available via this
			// row's own title attribute as a manual-copy fallback.
		}
	}

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
	// Phase 11 (11-UI-SPEC.md E2): trust tier and sync state are
	// orthogonal facts about the same chip — an external-tier source
	// appends " — untrusted external plugin" to WHICHEVER branch above
	// produced the text, rather than replacing it or gaining a fifth
	// branch of its own. The four base strings stay byte-identical for a
	// trusted-tier source (D-06).
	//
	// Phase 11 (11-UI-SPEC.md E4): a binary-changed source gets its OWN
	// branch — checked after the syncing check (still yields to "syncing…"
	// since a mid-relaunch is a more immediate fact) and ahead of the
	// tone switch below (the cause and remedy are different and specific,
	// so this takes priority over the plain destructive/"unreachable"
	// wording). This branch supplies its own complete Copywriting Contract
	// string and is exempt from the trailing "— untrusted external
	// plugin" append below — the exact E4 copy carries no such suffix.
	//
	// 12-10-PLAN.md (G-12-1/G-12-3): a fifth branch, placed after the
	// relative-time constant below and BEFORE the tone switch — a source
	// that synced successfully (never for an errored last_status, so a
	// real error's own copy is never displaced) but carries an advisory
	// gets the display name, the synced-relative phrase, and the
	// kernel's own advisory text, joined by the same em-dash-with-spaces
	// separator every other branch uses. Yields to the syncing and
	// pin-mismatch checks above it — both are more immediate facts than
	// an advisory about an otherwise-successful run. The four switch
	// branches below, the trailing external-tier append, and this
	// component's Copywriting Contract guard (source-chip-tooltip.test.ts)
	// stay byte-identical; an external-tier source with an advisory
	// correctly gets both, the same composition rule Phase 11 established.
	//
	// 12-11-PLAN.md (CR-01 gap closure): the advisory branch's gate now
	// defers to format.ts's `isAdvisoryOnly` (via the `advisoryOnly`
	// derived above) rather than testing `last_status` itself. The old
	// condition consulted neither `source.reachable` nor this file's own
	// computed `tone`, so a source that had gone unreachable while still
	// carrying a stale successful `last_status` and a leftover notice read
	// as a benign "synced … — advisory" tooltip while its own health dot
	// was red. `advisory !== ''` is kept alongside `advisoryOnly` only to
	// preserve today's pre-existing, out-of-scope whitespace-only-notice
	// behaviour (advisory trims to empty here while healthTone reads a
	// whitespace notice as non-empty) — that inconsistency is unchanged by
	// this plan in either direction.
	//
	// 13-06-PLAN.md (D-12/D-13/D-14) adds two more branches to this exact
	// chain:
	//  - `isManifestUnverified` sits in the PROBLEM group, immediately
	//    beside `isPinMismatch` — a trusted-directory binary refused at
	//    launch by the build manifest is exactly the same class of fact
	//    (never launched at all), so it gets its own contract-exact
	//    sentence at the same precedence position.
	//  - `isShadowed` sits in the ADVISORY group, gated on `advisoryOnly`
	//    exactly like the `last_notice` branch immediately above it —
	//    checked AFTER the notice branch so a source carrying both reads
	//    the notice's sentence (today's only specified coexistence order;
	//    the two advisories are not specified to combine). Neither new
	//    branch changes the trailing external-tier append below: a
	//    manifest-unverified or shadowed source is always `tier: "trusted"`
	//    (the manifest gate and the shadow rule are both trusted-tier-only
	//    facts), so that append's condition never fires for either.
	let tooltipText = $derived.by(() => {
		const base = (() => {
			if (source.syncing) return `${source.display_name} — syncing…`;
			if (isPinMismatch) return `${source.display_name} — binary changed since it was trusted`;
			if (isManifestUnverified)
				return `${source.display_name} — binary not in the trusted build manifest`;
			const relative = formatRelativeTime(source.last_sync_unix);
			if (advisory !== '' && advisoryOnly) {
				return `${source.display_name} — synced ${relative} — ${advisory}`;
			}
			if (isShadowed && advisoryOnly) {
				return `${source.display_name} — a same-named trusted-directory binary is shadowing this pinned plugin`;
			}
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
		})();
		return source.tier === 'external' && !isPinMismatch ? `${base} — untrusted external plugin` : base;
	});

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
		'group flex h-11 items-center rounded-full border border-border bg-card pr-1',
		// Mutually exclusive shrink alternatives keyed on the prop — never
		// both emitted (14-06, G-14-2): a shrinkable row chip may give way
		// (min-w-0 lets the truncating name span actually shrink); every
		// other render site keeps the original no-shrink utility.
		shrinkable ? 'min-w-0 shrink' : 'shrink-0',
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
						aria-describedby={chipDescId}
						onclick={() => onfilter(source.name)}
						class="flex min-w-0 max-w-48 items-center gap-1.5 self-stretch rounded-full pr-1.5 pl-2.5"
					>
						<span
							class={cn(
								'size-2 shrink-0 rounded-full',
								DOT_TONE_CLASS[tone],
								selected && 'ring-1 ring-primary-foreground'
							)}
							aria-hidden="true"
						></span>
						<TrustBadge tier={source.tier} scale="chip">
							{#snippet children()}
								<PluginIcon plugin={source.plugin} size="size-3.5 shrink-0" />
							{/snippet}
						</TrustBadge>
						<span
							class={cn(
								'truncate text-[14px] leading-[1.4]',
								selected ? 'text-primary-foreground' : 'text-foreground'
							)}>{source.display_name}</span
						>
					</button>
				{/snippet}
			</TooltipTrigger>
			<TooltipContent>{tooltipText}</TooltipContent>
		</Tooltip>
	</TooltipProvider>
	<!--
	  14-02-PLAN.md Task 1/2 (option-b): the browser-native tooltips that used
	  to duplicate/cover the app's own styled Tooltip popover (the outer
	  button's native-tooltip title, and a second one on the truncated name
	  span) are removed. The health sentence stays reachable as an
	  accessible DESCRIPTION — the role the removed button title played —
	  via this visually-hidden sibling span plus aria-describedby on the
	  button, rather than moving into aria-label (which would replace the
	  button's accessible NAME, today just the display name, with the whole
	  sentence — a real behaviour change ~24 Playwright locators depend on
	  not happening; see 14-02-PLAN.md Task 1's decision record). While the
	  Tooltip is open, bits-ui also sets its own aria-describedby pointing at
	  TooltipContent; this explicit attribute on the DOM element overrides
	  that one, which is harmless because both reference the identical
	  health-sentence expression. The truncated name span's own native
	  tooltip is also removed — its text is unchanged and fully covered by
	  the button's accessible name, so the redundant popover it produced
	  added nothing.
	-->
	<span id={chipDescId} class="sr-only">{tooltipText}</span>

	<DropdownMenu>
		<DropdownMenuTrigger>
			{#snippet child({ props })}
				<Button
					{...props}
					variant="ghost"
					size="icon"
					class={cn(
						'size-8 shrink-0 rounded-full opacity-0 transition-opacity group-hover:opacity-100 group-has-[:focus-visible]:opacity-100 max-md:opacity-100',
						selected &&
							'text-primary-foreground hover:bg-primary-foreground/20 hover:text-primary-foreground'
					)}
					aria-label={`${source.display_name} actions`}
					onclick={(event: MouseEvent) =>
						handleEditClick(event, (props as { onclick?: (e: MouseEvent) => void }).onclick)}
				>
					<EllipsisVertical class="size-4" />
				</Button>
			{/snippet}
		</DropdownMenuTrigger>
		<DropdownMenuContent>
			{#if isPinMismatch}
				<DropdownMenuItem onSelect={() => onedit(source.name, 'trust-update')}>
					<ShieldCheck aria-hidden="true" />
					Trust updated binary…
				</DropdownMenuItem>
			{/if}
			<DropdownMenuItem
				disabled={source.syncing || isPinMismatch}
				onSelect={() => onrefresh(source.name)}
			>
				<RefreshCw class={cn('size-4', source.syncing && 'animate-spin')} aria-hidden="true" />
				Refresh now
			</DropdownMenuItem>
			<DropdownMenuSeparator />
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
			{#if isExternal && source.pinned_hash}
				<DropdownMenuSeparator />
				<div
					class="flex items-center justify-between gap-2 px-2 py-1.5 text-[14px] leading-[1.4] text-muted-foreground"
				>
					<span class="truncate font-mono" title={source.pinned_hash}
						>Pinned: {shortHash(source.pinned_hash)}</span
					>
					<button
						type="button"
						onclick={copyPinnedHash}
						aria-label="Copy pinned hash"
						class="shrink-0 text-muted-foreground hover:text-foreground"
					>
						{#if hashCopied}
							<Check class="size-3.5 text-success" aria-hidden="true" />
						{:else}
							<Copy class="size-3.5" aria-hidden="true" />
						{/if}
					</button>
				</div>
			{/if}
		</DropdownMenuContent>
	</DropdownMenu>
</div>
