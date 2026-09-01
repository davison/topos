<script lang="ts">
	// FilterSourceDialog (M2-R3, #55; the design at #50): the chip menu's
	// "Filter this source…" session — one instance's own AND-ed filter
	// terms, edited whole and written with a single putConfig, mirroring
	// TrustKeyDialog's shape exactly (one instance, config+baseHash in,
	// onclose/onsaved out). Clearing the input and saving removes the
	// instance's entry — removal is the same affordance, not a second one.
	import {
		Dialog,
		DialogContent,
		DialogHeader,
		DialogTitle,
		DialogFooter
	} from '$lib/components/ui/dialog/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { setSourceFilterTerms } from '$lib/config-edit';
	import {
		putConfig,
		ApiError,
		CONFIG_CONFLICT_MESSAGE,
		type KernelConfig
	} from '$lib/api';

	let {
		open,
		webspace,
		instance,
		displayName,
		terms,
		config,
		baseHash,
		onclose,
		onsaved
	}: {
		open: boolean;
		webspace: string;
		instance: string;
		displayName: string;
		terms: string[];
		config: KernelConfig;
		baseHash: string;
		onclose: () => void;
		onsaved: () => void;
	} = $props();

	let saving = $state(false);
	let error = $state<string | null>(null);
	let raw = $state(terms.join(' '));

	function handleOpenChange(next: boolean) {
		if (!next) onclose();
	}

	async function confirm() {
		if (saving) return;
		saving = true;
		error = null;
		try {
			const nextTerms = raw.trim() === '' ? [] : raw.trim().split(/\s+/);
			const nextConfig = setSourceFilterTerms(config, webspace, instance, nextTerms);
			await putConfig({ base_hash: baseHash, config: nextConfig });
			onsaved();
		} catch (err) {
			error =
				err instanceof ApiError && err.code === 'config_changed_on_disk'
					? CONFIG_CONFLICT_MESSAGE
					: err instanceof ApiError
						? err.message
						: 'Something went wrong saving this source filter — check the browser console and try again.';
		} finally {
			saving = false;
		}
	}
</script>

<Dialog {open} onOpenChange={handleOpenChange}>
	<DialogContent class="max-w-md" data-filter-source-dialog={instance}>
		<DialogHeader>
			<DialogTitle>Filter {displayName}</DialogTitle>
		</DialogHeader>
		<p class="text-[14px] leading-[1.4] text-muted-foreground">
			Each word below must appear in an item from {displayName} for it to stay in this webspace's
			stream and search — other sources are untouched, and any webspace-wide filter still applies
			on top. Clear the field to stop narrowing this source.
		</p>
		<Input
			bind:value={raw}
			placeholder="e.g. boiler quote"
			aria-label={`Filter terms for ${displayName}`}
			data-filter-source-input
		/>
		{#if error}
			<p class="text-[14px] leading-[1.4] text-destructive">{error}</p>
		{/if}
		<DialogFooter>
			<Button variant="ghost" onclick={onclose} disabled={saving}>Cancel</Button>
			<Button onclick={confirm} disabled={saving} data-filter-source-save>Save</Button>
		</DialogFooter>
	</DialogContent>
</Dialog>
