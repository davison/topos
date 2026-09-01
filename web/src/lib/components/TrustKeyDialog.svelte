<script lang="ts">
	// TrustKeyDialog (M2-R4, davison/topos#49; the design in
	// docs/plugin-trust.md "Operator-trusted keys"): the consent that trusts
	// a developer's signing KEY — every future release that key signs — or
	// withdraws that trust. It mirrors TrustUpdateDialog exactly in shape:
	// every fact shown (key id, fingerprint, reused) is the kernel's, read
	// from the live SourceStatus; the one write is a single putConfig of
	// the whole config with the [[plugins.trusted_keys]] entry added or
	// removed (config-edit's setTrustedKey/removeTrustedKey), never a
	// dedicated endpoint.
	import {
		Dialog,
		DialogContent,
		DialogHeader,
		DialogTitle,
		DialogFooter
	} from '$lib/components/ui/dialog/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Alert, AlertDescription } from '$lib/components/ui/alert/index.js';
	import { setTrustedKey, removeTrustedKey } from '$lib/config-edit';
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
		mode,
		source,
		config,
		baseHash,
		onclose,
		onsaved
	}: {
		open: boolean;
		mode: 'trust' | 'untrust';
		source: SourceStatus;
		config: KernelConfig;
		baseHash: string;
		onclose: () => void;
		onsaved: () => void;
	} = $props();

	let saving = $state(false);
	let error = $state<string | null>(null);
	let note = $state('');

	let offer = $derived(source.offered_key ?? null);
	let keyId = $derived(mode === 'trust' ? (offer?.id ?? '') : (source.trusted_key ?? ''));

	function handleOpenChange(next: boolean) {
		if (!next) onclose();
	}

	async function confirm() {
		if (saving) return;
		if (mode === 'trust' && !offer) return;
		if (mode === 'untrust' && !keyId) return;
		saving = true;
		error = null;
		try {
			const nextConfig =
				mode === 'trust' && offer
					? setTrustedKey(config, offer, note.trim())
					: removeTrustedKey(config, keyId);
			await putConfig({ base_hash: baseHash, config: nextConfig });
			onsaved();
		} catch (err) {
			error =
				err instanceof ApiError && err.code === 'config_changed_on_disk'
					? CONFIG_CONFLICT_MESSAGE
					: err instanceof ApiError
						? err.message
						: 'Something went wrong changing which keys you trust — check the browser console and try again.';
		} finally {
			saving = false;
		}
	}
</script>

<Dialog {open} onOpenChange={handleOpenChange}>
	<DialogContent class="max-w-md" data-trust-key-dialog={mode}>
		{#if mode === 'trust' && offer}
			<DialogHeader>
				<DialogTitle>Trust this signing key?</DialogTitle>
			</DialogHeader>
			<p class="text-[16px] leading-[1.5] text-foreground">
				{source.plugin}'s release is signed by a key topos does not know. The signature verifies
				against the key it carries, so this key really did sign this release — but only you can say
				whose key it is. Trusting it admits <strong>every future release it signs</strong> on this
				instance, unpinned, the way topos's own key does.
			</p>
			<div class="rounded-md border border-border bg-card p-3">
				<p class="font-mono text-[14px] leading-[1.4]">Key id: {offer.id}</p>
				<p class="break-all font-mono text-[14px] leading-[1.4]" title={offer.fingerprint}>
					Fingerprint (SHA-256): {offer.fingerprint}
				</p>
			</div>
			{#if offer.reused}
				<Alert variant="destructive">
					<AlertDescription>
						This key id is already trusted — with a <strong>different</strong> key. A reused id is what
						an impersonation looks like. Do not trust it without checking with the publisher.
					</AlertDescription>
				</Alert>
			{/if}
			<p class="text-[14px] leading-[1.4] text-muted-foreground">
				Check the fingerprint against the one the developer publishes before you continue. Prefer
				to accept only this one binary? Cancel and use the chip's pin instead.
			</p>
			<div class="flex flex-col gap-1">
				<label for="trust-key-note" class="text-[14px] leading-[1.4] text-foreground">
					Note (whose key this is, where you checked it)
				</label>
				<Input
					id="trust-key-note"
					value={note}
					placeholder="e.g. Acme's release key — fingerprint checked on their release page"
					oninput={(e) => (note = e.currentTarget.value)}
				/>
			</div>
		{:else if mode === 'untrust'}
			<DialogHeader>
				<DialogTitle>Stop trusting this key?</DialogTitle>
			</DialogHeader>
			<p class="text-[16px] leading-[1.5] text-foreground">
				{source.plugin} runs because a release signed by key <span class="font-mono">{keyId}</span>
				is trusted by you. Withdrawing that trust returns every plugin it vouches for to the external
				tier at its next launch — each will then need your consent and a pin, as any external binary
				does. Nothing is removed from disk.
			</p>
		{:else}
			<DialogHeader>
				<DialogTitle>No key to act on</DialogTitle>
			</DialogHeader>
			<p class="text-[14px] leading-[1.4] text-muted-foreground">
				This source no longer carries an offer — its status changed since the menu was opened.
			</p>
		{/if}
		{#if error}
			<Alert variant="destructive">
				<AlertDescription>{error}</AlertDescription>
			</Alert>
		{/if}
		<DialogFooter>
			<Button type="button" variant="ghost" onclick={onclose}>Cancel</Button>
			{#if mode === 'trust' && offer}
				<Button type="button" disabled={saving} onclick={confirm}>Trust this key</Button>
			{:else if mode === 'untrust' && keyId}
				<Button type="button" variant="destructive" disabled={saving} onclick={confirm}>
					Stop trusting {shortHash(keyId, 12)}
				</Button>
			{/if}
		</DialogFooter>
	</DialogContent>
</Dialog>
