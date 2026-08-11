<script lang="ts">
	// Replaces the static <h1>{webspace}</h1> title (D-10): the header's
	// composition entry point. Trigger reads exactly as the heading it
	// replaces (same Display role, same truncate+title treatment) until
	// hovered/focused; the menu lists every configured webspace plus the
	// three static items 09-UI-SPEC.md Fix 7 permits (superseding D-13's
	// original two-item rule — see the widened test assertion in
	// webspace-switcher.test.ts) and nothing else.
	import { goto } from '$app/navigation';
	import {
		DropdownMenu,
		DropdownMenuTrigger,
		DropdownMenuContent,
		DropdownMenuItem,
		DropdownMenuSeparator
	} from '$lib/components/ui/dropdown-menu/index.js';
	import ChevronDown from '@lucide/svelte/icons/chevron-down';
	import Plus from '@lucide/svelte/icons/plus';
	import RotateCw from '@lucide/svelte/icons/rotate-cw';
	import { cn } from '$lib/utils.js';

	let {
		webspace,
		webspaces,
		oncreate,
		onreload,
		reloadBusy = false,
		onmanage
	}: {
		webspace: string;
		// webspaces is rendered in the exact order the kernel returns from
		// GET /api/config (deterministic map serialization order) — never
		// re-sorted here. This supersedes the UI-SPEC's literal
		// "config-declared order" wording: a Go map cannot preserve TOML
		// declaration order, so the stability guarantee that actually
		// matters (never re-sorted per render, never reordered by state) is
		// what this component honours instead.
		webspaces: string[];
		oncreate: () => void;
		// onreload/reloadBusy (09-06-PLAN.md Task 1, 09-UI-SPEC.md Fix 7):
		// the relocated `Reload config` action, now a single-click item at
		// the menu root rather than buried in ManageSourcesModal's footer.
		// reloadBusy disables the item for the duration of the request —
		// the same shared in-flight pattern every other write in the app
		// uses. The route owns the actual reload call (Task 2); this
		// component only renders the item and forwards the click.
		onreload: () => void;
		reloadBusy?: boolean;
		onmanage: () => void;
	} = $props();

	function handleSelect(name: string) {
		if (name === webspace) return;
		goto(`/w/${encodeURIComponent(name)}`);
	}
</script>

<DropdownMenu>
	<DropdownMenuTrigger>
		{#snippet child({ props })}
			<button
				{...props}
				type="button"
				class="-mx-1 flex items-center gap-1.5 rounded-lg px-1 hover:bg-muted"
			>
				<span
					class="max-w-[min(80vw,32rem)] truncate text-[28px] leading-[1.2] font-semibold text-foreground"
					title={webspace}
				>
					{webspace}
				</span>
				<ChevronDown class="size-4 shrink-0 text-muted-foreground" aria-hidden="true" />
			</button>
		{/snippet}
	</DropdownMenuTrigger>
	<DropdownMenuContent>
		{#each webspaces as name (name)}
			{@const isCurrent = name === webspace}
			<DropdownMenuItem
				aria-current={isCurrent ? 'true' : undefined}
				onSelect={() => handleSelect(name)}
			>
				<span class={cn('whitespace-normal break-words', isCurrent && 'font-semibold')}>
					{name}
				</span>
			</DropdownMenuItem>
		{/each}
		<DropdownMenuSeparator />
		<DropdownMenuItem onSelect={oncreate}>
			<Plus class="size-4" aria-hidden="true" />
			New webspace
		</DropdownMenuItem>
		<DropdownMenuItem disabled={reloadBusy} onSelect={onreload}>
			<RotateCw class="size-4" aria-hidden="true" />
			Reload config
		</DropdownMenuItem>
		<DropdownMenuSeparator />
		<DropdownMenuItem onSelect={onmanage}>Manage sources…</DropdownMenuItem>
	</DropdownMenuContent>
</DropdownMenu>
