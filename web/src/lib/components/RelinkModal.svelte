<script lang="ts">
	// The chip menu's Re-link… entry point (D-03, 08-04-PLAN.md Task 2): a
	// small Dialog wrapping QRPanel (Task 1) — the SAME component the
	// Add-Source flow uses, one component reused unforked from exactly
	// two entry points. Follows EditSourceModal.svelte's own prop and
	// callback convention (open/instance/config/onclose), plus onrelinked
	// in place of onsaved.
	//
	// Deliberately smaller than the Add-Source dialog (max-w-lg): this
	// dialog carries the panel and nothing else — no connection fields,
	// no match fields, no width override on DialogContent — 08-UI-SPEC.md's
	// Amendment: "a new, narrower Dialog/DialogContent sized to the
	// panel's own content".
	import { Dialog, DialogContent, DialogHeader, DialogTitle } from '$lib/components/ui/dialog/index.js';
	import QRPanel from './QRPanel.svelte';
	import { WHATSAPP_PLUGIN_BINARY } from '$lib/plugin-fields';
	import type { KernelConfig } from '$lib/api';

	let {
		open,
		instance,
		config,
		onclose,
		onrelinked
	}: {
		open: boolean;
		// instance is the already-configured WhatsApp instance id being
		// re-linked — its own stored plugin/path/display_name are read
		// from `config` below, never re-typed by the caller.
		instance: string;
		config: KernelConfig;
		onclose: () => void;
		// onrelinked fires once the panel reports paired, so the caller
		// (the webspace route) refreshes source health — the chip's own
		// health dot updates in place, with no page reload (the
		// amendment's Re-link success transition).
		onrelinked: () => void;
	} = $props();

	let source = $derived(config.sources[instance]);
	let displayName = $derived(source?.display_name ?? instance);
	// Falls back to WHATSAPP_PLUGIN_BINARY defensively — every real
	// instance this dialog opens for is already a WhatsApp-typed source
	// (SourceChip.svelte's own source_type gate), so source?.plugin
	// should always resolve; the fallback only guards a not-yet-loaded
	// config snapshot from ever leaving `plugin` empty.
	let plugin = $derived(source?.plugin ?? WHATSAPP_PLUGIN_BINARY);
	let path = $derived(source?.path ?? '');

	function handleOpenChange(next: boolean) {
		if (!next) onclose();
	}

	// The panel's own cancel-on-unmount (T-08-10) terminates the session
	// the instant this dialog closes by any path — Escape, outside click,
	// or this explicit handler — so no separate cancel wiring is needed
	// here.
	function handlePaired() {
		onclose();
		onrelinked();
	}

	function handleCancelled() {
		onclose();
	}
</script>

<Dialog {open} onOpenChange={handleOpenChange}>
	<DialogContent>
		<DialogHeader>
			<DialogTitle>Re-link {displayName}</DialogTitle>
		</DialogHeader>
		<QRPanel {plugin} {path} {instance} onpaired={handlePaired} oncancelled={handleCancelled} />
	</DialogContent>
</Dialog>
