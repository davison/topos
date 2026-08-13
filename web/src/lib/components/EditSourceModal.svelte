<script lang="ts">
	// The chip menu's two edit flows (D-12, 07-04-PLAN.md Task 3): 'connection'
	// mode reuses ConnectionForm (pre-filled, secret field showing the
	// variable NAME it already references, never a value) with a
	// persistent cross-webspace notice above the fields; 'match' mode
	// reuses MatchFieldsForm scoped to THIS webspace's own block only.
	// Both write through the same shared save-in-flight/error pattern
	// every other modal in this phase uses.
	import { untrack } from 'svelte';
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
	import {
		missingRequiredFields,
		missingRequiredFieldsMessage,
		extrasKeyError,
		type ExtrasRow
	} from '$lib/plugin-fields';
	import { seedConnectionValues, seedMatchBlock } from '$lib/edit-modal-state';
	import {
		putConfig,
		ApiError,
		CONFIG_CONFLICT_MESSAGE,
		type KernelConfig,
		type SourceConfig,
		type ExtrasFieldDecl
	} from '$lib/api';

	let {
		open,
		mode,
		instance,
		webspace,
		config,
		baseHash,
		envVars,
		vocabulary,
		extrasFields,
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
		// extrasFields (Phase 11, PLUG-09, D-15) is this instance's plugin's
		// declared expected extras keys, resolved by the SAME describePlugin
		// call the caller now also makes for 'connection' mode (see the
		// route's handleChipEdit) — reused unforked by ConnectionForm's E6
		// section exactly like the add flow. Safe to pass [] in 'match'
		// mode, or when the describe call failed: the free-form editor
		// still shows an instance's already-saved extras values either way.
		extrasFields: ExtrasFieldDecl[];
		onclose: () => void;
		onsaved: () => void;
	} = $props();

	let displayName = $derived(config.sources[instance]?.display_name ?? instance);

	// Seeded once from the incoming props via the shared edit-modal-state
	// helpers (07-08-PLAN.md Task 2, closing 07-REVIEW.md CR-02) — the
	// route-level caller keys this component (e.g. {#key
	// `${instance}-${mode}`}) so a genuinely different instance/mode always
	// mounts a fresh EditSourceModal, matching MatchFieldsForm's own "seed
	// once, key to remount" discipline. The reset-on-open effect below is a
	// second, defensive layer: it re-runs these same two helpers whenever
	// `open` transitions to true, so a caller that keeps this component
	// mounted across a close (the exact caller-side bug CR-02 found) still
	// cannot resurface a discarded session's typing.
	let connectionValues = $state<SourceConfig>(seedConnectionValues(config, instance));
	let matchBlock = $state<Record<string, string[]>>(seedMatchBlock(config, webspace, instance));
	let saving = $state(false);
	let error = $state<string | null>(null);
	// extrasRows (Phase 11, PLUG-09, D-15): ConnectionForm's own free-form
	// extras row editor state, bound here — same "caller owns it, child
	// mutates it" contract as AddSourceModal's own extrasRows — so
	// submitConnection below can validate the exact rows the form renders.
	let extrasRows = $state<ExtrasRow[]>([]);

	// Defensive reset-on-open (CR-02's second layer, matching
	// CreateWebspaceModal's and ManageSourcesModal's own reset-on-open
	// effects): re-seed both pieces of form state, and clear the error and
	// in-flight flag, whenever `open` becomes true.
	//
	// This effect tracks the `open` flag and NOTHING else. `open` is read
	// first, outside `untrack`, so the effect still re-runs on every open
	// transition; every subsequent read (config/instance/webspace) and
	// assignment is wrapped in `untrack` — imported from `svelte`, not used
	// anywhere else in this codebase before this — so a parent config
	// refresh landing while the modal stays open can NEVER re-run this
	// effect. Without that, a background config reload mid-edit would
	// silently discard the user's in-progress typing — the same class of
	// silent data loss CR-02 itself is, merely inverted. ManageSourcesModal's
	// own reset-on-open effect DOES track its config prop; that is correct
	// for its snapshot-refresh purpose and would be wrong for this one.
	$effect(() => {
		if (!open) return;
		untrack(() => {
			connectionValues = seedConnectionValues(config, instance);
			matchBlock = seedMatchBlock(config, webspace, instance);
			error = null;
			saving = false;
		});
	});

	function handleOpenChange(next: boolean) {
		if (!next) onclose();
	}

	async function submitConnection(event: SubmitEvent) {
		event.preventDefault();
		if (saving) return;

		// Same guard as AddSourceModal's Connect step and Save anyway (07-13-
		// PLAN.md Task 2, planning choice 4): blanking a required field here
		// persists an instance the plugin cannot launch, and the next
		// hot-apply reconcile fails on it. Uses this modal's own error state
		// and rendering — no new error surface.
		const missing = missingRequiredFields(connectionValues.plugin, connectionValues);
		if (missing.length > 0) {
			error = missingRequiredFieldsMessage(missing);
			return;
		}

		// Same submit-time point missingRequiredFields already gates (Phase
		// 11, D-15) — an empty or duplicate extras key must never be
		// persisted either.
		const extrasErr = extrasKeyError(extrasFields, extrasRows);
		if (extrasErr) {
			error = extrasErr;
			return;
		}

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
					{extrasFields}
					bind:extrasRows
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
