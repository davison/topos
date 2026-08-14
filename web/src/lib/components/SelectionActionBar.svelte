<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { Separator } from '$lib/components/ui/separator/index.js';

	// The floating bulk-action bar (13-UI-SPEC.md E2). Rendered by
	// StreamList.svelte inside the stream pane's own scroll container
	// (sticky bottom-4) whenever the selection is non-empty — appearing and
	// disappearing is the entire selection-mode signal, so this component
	// renders nothing at count 0 rather than a disabled/greyed variant.
	//
	// primaryLabel is driven by the caller's own view state (normal stream
	// vs. the excluded bucket, 13-03-PLAN.md Task 2) — this component holds
	// no view logic of its own, only renders whichever label it's handed.
	// busy disables both buttons for the duration of an in-flight bulk
	// write (prevents a double-fire double-submit); the caller decides when
	// to flip it back, exactly like DetailPane.svelte's own markBusy.
	let {
		count,
		primaryLabel,
		busy,
		onprimary,
		onclear
	}: {
		count: number;
		primaryLabel: 'Exclude' | 'Include';
		busy: boolean;
		onprimary: () => void;
		onclear: () => void;
	} = $props();
</script>

{#if count > 0}
	<div
		class="sticky bottom-4 flex items-center gap-3 rounded-lg border border-border bg-popover px-4 py-3 shadow-lg"
	>
		<span class="text-[14px] leading-[1.4] text-foreground">{count} selected</span>
		<Separator orientation="vertical" class="h-4" />
		<Button size="sm" disabled={busy} onclick={onprimary}>{primaryLabel}</Button>
		<Button variant="ghost" size="sm" disabled={busy} onclick={onclear}>Clear</Button>
	</div>
{/if}
