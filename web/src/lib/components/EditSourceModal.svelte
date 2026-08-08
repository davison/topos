<script lang="ts">
	// The chip menu's two edit flows (D-12, 07-04-PLAN.md Task 3): 'connection'
	// mode reuses ConnectionForm (pre-filled, secret field showing the
	// variable NAME it already references, never a value) with a
	// persistent cross-webspace notice above the fields; 'match' mode
	// reuses MatchFieldsForm scoped to THIS webspace's own block only.
	// Both write through the same shared save-in-flight/error pattern
	// every other modal in this phase uses.
	import {
		Dialog,
		DialogContent,
		DialogHeader,
		DialogTitle,
		DialogFooter
	} from '$lib/components/ui/dialog/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Alert, AlertDescription } from '$lib/components/ui/alert/index.js';
	import ConnectionForm from './ConnectionForm.svelte';
	import MatchFieldsForm from './MatchFieldsForm.svelte';
	import { upsertSourceInstance, setMatchBlock } from '$lib/config-edit';
	import { putConfig, ApiError, CONFIG_CONFLICT_MESSAGE, type KernelConfig, type SourceConfig } from '$lib/api';

	let {
		open,
		mode,
		instance,
		webspace,
		config,
		baseHash,
		envVars,
		vocabulary,
		onclose,
		onsaved
	}: {
		open: boolean;
		mode: 'connection' | 'match';
		instance: string;
		webspace: string;
		config: KernelConfig;
		baseHash: string;
		envVars: Record<string, boolean>;
		// vocabulary is this instance's declared match_vocabulary, resolved
		// by the caller (the webspace route, via describePlugin — see that
		// route's handleChipEdit) before opening this modal in 'match' mode.
		// Unused (and safe to pass []) in 'connection' mode.
		vocabulary: string[];
		onclose: () => void;
		onsaved: () => void;
	} = $props();

	let displayName = $derived(config.sources[instance]?.display_name ?? instance);

	// Seeded once from the incoming props — callers key this component
	// (e.g. {#key `${instance}-${mode}`}) so a genuinely different
	// instance/mode always mounts a fresh EditSourceModal, matching
	// MatchFieldsForm's own "seed once, key to remount" discipline (see
	// that component's doc comment for why a reactive re-seed would fight
	// the user's in-progress typing).
	let connectionValues = $state<SourceConfig>(
		config.sources[instance] ?? { plugin: '', agent: { read: false, handoff: false } }
	);
	let matchBlock = $state<Record<string, string[]>>(
		config.webspaces[webspace]?.match?.[instance] ?? {}
	);
	let saving = $state(false);
	let error = $state<string | null>(null);

	function handleOpenChange(next: boolean) {
		if (!next) onclose();
	}

	async function submitConnection(event: SubmitEvent) {
		event.preventDefault();
		if (saving) return;
		saving = true;
		error = null;
		try {
			const nextConfig = upsertSourceInstance(config, instance, connectionValues);
			await putConfig({ base_hash: baseHash, config: nextConfig });
			onsaved();
		} catch (err) {
			error =
				err instanceof ApiError && err.code === 'config_changed_on_disk'
					? CONFIG_CONFLICT_MESSAGE
					: err instanceof ApiError
						? err.message
						: 'Something went wrong saving this connection — check the browser console and try again.';
		} finally {
			saving = false;
		}
	}

	async function submitMatch(event: SubmitEvent) {
		event.preventDefault();
		if (saving) return;
		saving = true;
		error = null;
		try {
			const nextConfig = setMatchBlock(config, webspace, instance, matchBlock);
			await putConfig({ base_hash: baseHash, config: nextConfig });
			onsaved();
		} catch (err) {
			error =
				err instanceof ApiError && err.code === 'config_changed_on_disk'
					? CONFIG_CONFLICT_MESSAGE
					: err instanceof ApiError
						? err.message
						: 'Something went wrong saving match settings — check the browser console and try again.';
		} finally {
			saving = false;
		}
	}
</script>

<Dialog {open} onOpenChange={handleOpenChange}>
	<DialogContent class="max-w-md">
		{#if mode === 'connection'}
			<DialogHeader>
				<DialogTitle>Edit connection</DialogTitle>
			</DialogHeader>
			<div class="mb-4 rounded-md bg-muted p-3">
				<p class="text-[14px] leading-[1.4] text-muted-foreground">
					Changes here apply to every webspace using {displayName}.
				</p>
			</div>
			<form onsubmit={submitConnection}>
				<ConnectionForm
					pluginBinary={connectionValues.plugin}
					values={connectionValues}
					{envVars}
					onchange={(next) => (connectionValues = next)}
				/>

				{#if error}
					<Alert variant="destructive" class="mt-4">
						<AlertDescription>{error}</AlertDescription>
					</Alert>
				{/if}

				<DialogFooter>
					<Button type="button" variant="ghost" onclick={onclose}>Cancel</Button>
					<Button type="submit" disabled={saving}>Save changes</Button>
				</DialogFooter>
			</form>
		{:else}
			<DialogHeader>
				<DialogTitle>Edit match settings</DialogTitle>
			</DialogHeader>
			<form onsubmit={submitMatch}>
				<MatchFieldsForm
					{vocabulary}
					values={matchBlock}
					onchange={(block) => (matchBlock = block)}
				/>

				{#if error}
					<Alert variant="destructive" class="mt-4">
						<AlertDescription>{error}</AlertDescription>
					</Alert>
				{/if}

				<DialogFooter>
					<Button type="button" variant="ghost" onclick={onclose}>Cancel</Button>
					<Button type="submit" disabled={saving}>Save changes</Button>
				</DialogFooter>
			</form>
		{/if}
	</DialogContent>
</Dialog>
