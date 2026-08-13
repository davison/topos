<script lang="ts">
	// The chip menu's "Trust updated binary…" re-pin confirmation dialog
	// (11-06-PLAN.md Task 2, 11-UI-SPEC.md E4, D-03). Same prop shape
	// RelinkModal.svelte already establishes for a per-chip dialog opened
	// from SourceChip's own menu (open/instance-derived-content/onclose),
	// plus the save-in-flight/destructive-Alert/CONFIG_CONFLICT_MESSAGE
	// convention EditSourceModal.svelte's own submit handlers already use.
	//
	// Unlike EditSourceModal, this dialog holds no editable form state at
	// all — confirming is a single, un-parameterized write. The route
	// mounts this component keyed on the instance id (mirroring
	// EditSourceModal's own `{#key}` remount discipline), so `saving`/
	// `error` start fresh on every open without needing a defensive
	// reset-on-open effect.
	import {
		Dialog,
		DialogContent,
		DialogHeader,
		DialogTitle,
		DialogFooter
	} from '$lib/components/ui/dialog/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Alert, AlertDescription } from '$lib/components/ui/alert/index.js';
	import { setPluginPin } from '$lib/config-edit';
	import { shortHash } from '$lib/format';
	import {
		putConfig,
		ApiError,
		CONFIG_CONFLICT_MESSAGE,
		type KernelConfig,
		type SourceStatus
	} from '$lib/api';

	let {
		open,
		source,
		config,
		baseHash,
		onclose,
		onsaved
	}: {
		open: boolean;
		// source is the instance's live SourceStatus (GET /api/sources) —
		// carries plugin (the binary name setPluginPin keys on, D-02),
		// pinned_hash (the previously-trusted hash, possibly absent) and
		// current_hash (the kernel-published hash actually on disk, the
		// exact value this dialog writes back — the client never computes
		// its own hash, T-11-30).
		source: SourceStatus;
		config: KernelConfig;
		baseHash: string;
		onclose: () => void;
		onsaved: () => void;
	} = $props();

	let saving = $state(false);
	let error = $state<string | null>(null);

	function handleOpenChange(next: boolean) {
		if (!next) onclose();
	}

	async function confirmTrustUpdate() {
		if (saving) return;
		saving = true;
		error = null;
		try {
			// setPluginPin only ever echoes back a kernel-computed hash — it
			// never derives one itself (T-11-25); source.current_hash is
			// GET /api/sources' own re-verified-at-launch value, the same
			// fact this dialog's own "Currently on disk" line already
			// showed. Pins are per BINARY (D-02): this one write repairs
			// every instance backed by source.plugin in the same save.
			const nextConfig = setPluginPin(config, source.plugin, source.current_hash ?? '');
			await putConfig({ base_hash: baseHash, config: nextConfig });
			onsaved();
		} catch (err) {
			error =
				err instanceof ApiError && err.code === 'config_changed_on_disk'
					? CONFIG_CONFLICT_MESSAGE
					: err instanceof ApiError
						? err.message
						: 'Something went wrong trusting this binary — check the browser console and try again.';
		} finally {
			saving = false;
		}
	}
</script>

<Dialog {open} onOpenChange={handleOpenChange}>
	<DialogContent class="max-w-md">
		<DialogHeader>
			<DialogTitle>Binary changed</DialogTitle>
		</DialogHeader>
		<p class="text-[16px] leading-[1.5] text-foreground">
			{source.plugin} no longer matches the hash topos pinned when you added it. This can mean the
			binary was rebuilt, or that something else replaced it — topos can't tell which. Only continue
			if you trust this change.
		</p>
		<div class="rounded-md border border-border bg-card p-3">
			<p class="font-mono text-[14px] leading-[1.4]">
				Previously pinned: {source.pinned_hash ? shortHash(source.pinned_hash) : 'not pinned'}
			</p>
			<p class="break-all font-mono text-[14px] leading-[1.4]">
				Currently on disk: {source.current_hash ?? ''}
			</p>
		</div>

		{#if error}
			<Alert variant="destructive">
				<AlertDescription>{error}</AlertDescription>
			</Alert>
		{/if}

		<DialogFooter>
			<Button type="button" variant="ghost" onclick={onclose}>Cancel</Button>
			<Button type="button" disabled={saving} onclick={confirmTrustUpdate}>
				Trust updated binary
			</Button>
		</DialogFooter>
	</DialogContent>
</Dialog>
