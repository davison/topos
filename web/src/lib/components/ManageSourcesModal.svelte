<script lang="ts">
	import {
		DialogFooter
	} from '$lib/components/ui/dialog/index.js';
	import { Input } from '$lib/components/ui/input/index.js';
	// D-13's deliberately minimal escape hatch — the ONE place in the app
	// that deletes a source instance or a webspace outright. Opened only
	// from WebspaceSwitcher's "Manage sources…" item (07-05-PLAN.md Task
	// 1). The config-reload affordance this modal used to also hold has
	// relocated to WebspaceSwitcher's own menu root, owned by the webspace
	// route (09-06-PLAN.md Task 2, 09-UI-SPEC.md Fix 7) — this file no
	// longer imports reloadConfig or renders any reload control; one entry
	// point for that action, not two.
	//
	// Holds its own local (localConfig/localHash) snapshot, seeded from the
	// incoming config/baseHash props whenever the modal transitions open
	// (CreateWebspaceModal's own reset-on-open pattern) and updated
	// directly from each successful putConfig/getConfig response — never
	// waiting on the parent route's own onchanged-triggered refresh to land
	// before the NEXT delete in this same modal session can proceed with a
	// fresh base_hash. onchanged() is still called after every success so
	// the parent's chip row/stream/header state also refreshes once that
	// fetch resolves.
	import { goto } from '$app/navigation';
	import {
		Dialog,
		DialogContent,
		DialogHeader,
		DialogTitle
	} from '$lib/components/ui/dialog/index.js';
	import {
		AlertDialog,
		AlertDialogContent,
		AlertDialogTitle,
		AlertDialogDescription,
		AlertDialogAction,
		AlertDialogCancel
	} from '$lib/components/ui/alert-dialog/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Alert, AlertDescription } from '$lib/components/ui/alert/index.js';
	import EditSourceModal from './EditSourceModal.svelte';
	import Pencil from '@lucide/svelte/icons/pencil';
	import Trash2 from '@lucide/svelte/icons/trash-2';
	import PluginIcon from '$lib/components/PluginIcon.svelte';
	import { removeSourceInstance, removeWebspace, renameWebspace } from '$lib/config-edit';
	import {
		putConfig,
		getConfig,
		ApiError,
		CONFIG_CONFLICT_MESSAGE,
		type KernelConfig
	} from '$lib/api';

	let {
		open,
		config,
		baseHash,
		envVars,
		currentWebspace,
		onclose,
		onchanged
	}: {
		open: boolean;
		config: KernelConfig;
		baseHash: string;
		envVars: Record<string, boolean>;
		// currentWebspace serves two purposes: (1) threaded through to
		// EditSourceModal's required `webspace` prop when this modal's
		// "Edit" opens it in 'connection' mode — that mode never actually
		// reads `webspace` (only 'match' mode does, and Manage Sources never
		// opens 'match' mode here), so that use is a type-shape
		// accommodation only; (2) compared against a just-deleted
		// webspace's own name below, so a delete of the webspace the user
		// is currently looking at navigates away rather than leaving them
		// on a route the kernel no longer knows.
		currentWebspace: string;
		onclose: () => void;
		onchanged: () => void;
	} = $props();

	let localConfig = $state<KernelConfig>(config);
	let localHash = $state(baseHash);
	let instanceDeleteTarget = $state<string | null>(null);
	let webspaceDeleteTarget = $state<string | null>(null);
	let webspaceRenameTarget = $state<string | null>(null);
	let renameValue = $state('');
	let editInstance = $state<string | null>(null);
	let deleting = $state(false);
	let deleteError = $state<string | null>(null);

	// Reset local state whenever the modal transitions to open, so a second
	// open after a prior session never shows a stale error or a stale
	// snapshot from before the parent's own config/baseHash last changed —
	// same discipline as CreateWebspaceModal's own reset-on-open effect.
	$effect(() => {
		if (open) {
			localConfig = config;
			localHash = baseHash;
			instanceDeleteTarget = null;
			webspaceDeleteTarget = null;
			editInstance = null;
			deleting = false;
			deleteError = null;
		}
	});

	let instanceIds = $derived(Object.keys(localConfig.sources));
	let webspaceNames = $derived(Object.keys(localConfig.webspaces));

	function displayNameFor(id: string): string {
		return localConfig.sources[id]?.display_name ?? id;
	}

	function handleOpenChange(next: boolean) {
		if (!next) onclose();
	}

	async function confirmDeleteInstance() {
		if (!instanceDeleteTarget || deleting) return;
		deleting = true;
		deleteError = null;
		try {
			const nextConfig = removeSourceInstance(localConfig, instanceDeleteTarget);
			const res = await putConfig({ base_hash: localHash, config: nextConfig });
			localConfig = res.config;
			localHash = res.hash;
			instanceDeleteTarget = null;
			onchanged();
		} catch (err) {
			deleteError =
				err instanceof ApiError && err.code === 'config_changed_on_disk'
					? CONFIG_CONFLICT_MESSAGE
					: err instanceof ApiError
						? err.message
						: 'Something went wrong deleting this source — check the browser console and try again.';
		} finally {
			deleting = false;
		}
	}


	// The rename write (M3-R2, #77): one putConfig carrying the key change
	// alone — the kernel's rename detection pairs on the byte-identical
	// body and migrates the index rows, so items and exclusion marks
	// survive. Renaming the webspace the user is on navigates to the new
	// URL, the delete flow's own discipline.
	let renameError = $state<string | null>(null);
	async function confirmRenameWebspace() {
		const from = webspaceRenameTarget;
		const to = renameValue.trim();
		if (!from || deleting || to === '') return;
		if (to !== from && to in localConfig.webspaces) {
			renameError = `A webspace named "${to}" already exists.`;
			return;
		}
		if (to === from) {
			webspaceRenameTarget = null;
			return;
		}
		deleting = true;
		renameError = null;
		try {
			const nextConfig = renameWebspace(localConfig, from, to);
			const res = await putConfig({ base_hash: localHash, config: nextConfig });
			localConfig = res.config;
			localHash = res.hash;
			webspaceRenameTarget = null;
			onchanged();
			if (from === currentWebspace) {
				await goto(`/w/${encodeURIComponent(to)}`);
			}
		} catch (err) {
			renameError =
				err instanceof ApiError && err.code === 'config_changed_on_disk'
					? CONFIG_CONFLICT_MESSAGE
					: err instanceof ApiError
						? err.message
						: 'Something went wrong renaming this webspace — check the browser console and try again.';
		} finally {
			deleting = false;
		}
	}

	async function confirmDeleteWebspace() {
		if (!webspaceDeleteTarget || deleting) return;
		const deletedName = webspaceDeleteTarget;
		deleting = true;
		deleteError = null;
		try {
			const nextConfig = removeWebspace(localConfig, deletedName);
			const res = await putConfig({ base_hash: localHash, config: nextConfig });
			localConfig = res.config;
			localHash = res.hash;
			webspaceDeleteTarget = null;
			onchanged();
			// Deleting the webspace the user is currently looking at must
			// never leave them on a route the kernel no longer knows —
			// navigate to another remaining webspace, or to the root
			// route's own zero-webspaces empty state (07-03-PLAN.md) when
			// none remain.
			if (deletedName === currentWebspace) {
				const remaining = Object.keys(res.config.webspaces);
				if (remaining.length > 0) {
					await goto(`/w/${encodeURIComponent(remaining[0])}`);
				} else {
					await goto('/');
				}
			}
		} catch (err) {
			deleteError =
				err instanceof ApiError && err.code === 'config_changed_on_disk'
					? CONFIG_CONFLICT_MESSAGE
					: err instanceof ApiError
						? err.message
						: 'Something went wrong deleting this webspace — check the browser console and try again.';
		} finally {
			deleting = false;
		}
	}

	// EditSourceModal (the chip menu's own "Edit connection…" modal, reused
	// unforked here) calls its own putConfig internally and hands back no
	// response body — re-fetch here so a delete immediately following an
	// edit, within the same Manage Sources session, still carries a fresh
	// base_hash rather than racing the parent's own onchanged-triggered
	// refresh.
	async function handleEditSaved() {
		editInstance = null;
		try {
			const res = await getConfig();
			localConfig = res.config;
			localHash = res.hash;
		} catch {
			// Best-effort refresh only — onchanged() below still fires the
			// parent's own getConfig(), which is the authoritative fallback
			// if this one failed.
		}
		onchanged();
	}
</script>

<Dialog {open} onOpenChange={handleOpenChange}>
	<DialogContent class="max-w-lg">
		<DialogHeader>
			<DialogTitle>Manage sources</DialogTitle>
		</DialogHeader>

		<div class="flex flex-col gap-1">
			<h3 class="text-[14px] leading-[1.4] font-semibold text-foreground">Source instances</h3>
			<div class="max-h-64 divide-y divide-border overflow-y-auto">
				{#each instanceIds as id (id)}
					{@const source = localConfig.sources[id]}
					<div class="flex items-center justify-between gap-2 py-2">
						<div class="flex min-w-0 items-center gap-2">
							<!-- 09-UI-SPEC.md Fix 10 sizing rule: size-4 (16px),
							     matching the Pencil/Trash2 sizing already in this
							     row. shrink-0 so the icon survives a long
							     truncating display name. -->
							<PluginIcon plugin={source.plugin} size="size-4 shrink-0" />
							<div class="min-w-0">
								<p
									class="truncate text-[14px] leading-[1.4] text-foreground"
									title={displayNameFor(id)}
								>
									{displayNameFor(id)}
								</p>
								<p class="text-[14px] leading-[1.4] text-muted-foreground">{source.plugin}</p>
							</div>
						</div>
						<div class="flex shrink-0 items-center gap-1">
							<Button variant="ghost" size="sm" onclick={() => (editInstance = id)}>
								<Pencil class="size-3.5" aria-hidden="true" />
								Edit
							</Button>
							<Button
								variant="ghost"
								size="sm"
								class="text-destructive"
								onclick={() => (instanceDeleteTarget = id)}
							>
								<Trash2 class="size-3.5" aria-hidden="true" />
								Delete
							</Button>
						</div>
					</div>
				{/each}
			</div>
		</div>

		<div class="mt-4 flex flex-col gap-1">
			<h3 class="text-[14px] leading-[1.4] font-semibold text-foreground">Webspaces</h3>
			<div class="max-h-64 divide-y divide-border overflow-y-auto">
				{#each webspaceNames as name (name)}
					<div class="flex items-center justify-between gap-2 py-2">
						<p class="truncate text-[14px] leading-[1.4] text-foreground" title={name}>
							{name}
						</p>
						<div class="flex shrink-0 items-center gap-1">
							<!-- M3-R2 (#77): the rename affordance the capture's
							     fallback names — beside the delete, never in-place
							     in the header (it collides with the switcher). -->
							<Button
								variant="ghost"
								size="sm"
								aria-label={`Rename webspace ${name}`}
								onclick={() => {
									webspaceRenameTarget = name;
									renameValue = name;
								}}
							>
								<Pencil class="size-3.5" aria-hidden="true" />
								Rename
							</Button>
							<Button
								variant="ghost"
								size="sm"
								class="text-destructive"
								onclick={() => (webspaceDeleteTarget = name)}
							>
								<Trash2 class="size-3.5" aria-hidden="true" />
								Delete
							</Button>
						</div>
					</div>
				{/each}
			</div>
		</div>

		{#if deleteError}
			<Alert variant="destructive" class="mt-4">
				<AlertDescription>{deleteError}</AlertDescription>
			</Alert>
		{/if}
	</DialogContent>
</Dialog>

<!--
  Destructive Confirmation Contract (07-UI-SPEC.md): AlertDialog, no
  type-to-confirm field — a single deliberate click plus the named,
  specific consequence in the body copy. This is the ONLY place in the
  app an instance can be deleted outright (D-12/D-13) — the chip menu
  offers "Remove from this webspace" only, never full deletion.
-->
<AlertDialog
	open={instanceDeleteTarget !== null}
	onOpenChange={(next) => {
		if (!next) instanceDeleteTarget = null;
	}}
>
	<AlertDialogContent>
		<AlertDialogTitle
			>Delete {instanceDeleteTarget ? displayNameFor(instanceDeleteTarget) : ''}?</AlertDialogTitle
		>
		<AlertDialogDescription>
			This removes {instanceDeleteTarget
				? displayNameFor(instanceDeleteTarget)
				: ''} from every webspace and deletes its indexed items. This can't be undone.
		</AlertDialogDescription>
		<div class="mt-4 flex justify-end gap-2">
			<AlertDialogCancel>Cancel</AlertDialogCancel>
			<AlertDialogAction variant="destructive" disabled={deleting} onclick={confirmDeleteInstance}>
				Delete
			</AlertDialogAction>
		</div>
	</AlertDialogContent>
</AlertDialog>

<AlertDialog
	open={webspaceDeleteTarget !== null}
	onOpenChange={(next) => {
		if (!next) webspaceDeleteTarget = null;
	}}
>
	<AlertDialogContent>
		<AlertDialogTitle>Delete {webspaceDeleteTarget}?</AlertDialogTitle>
		<AlertDialogDescription>
			This removes the webspace and its filters. Source instances and other webspaces are
			unaffected.
		</AlertDialogDescription>
		<div class="mt-4 flex justify-end gap-2">
			<AlertDialogCancel>Cancel</AlertDialogCancel>
			<AlertDialogAction variant="destructive" disabled={deleting} onclick={confirmDeleteWebspace}>
				Delete
			</AlertDialogAction>
		</div>
	</AlertDialogContent>
</AlertDialog>

{#if editInstance}
	{#key editInstance}
		<EditSourceModal
			open={true}
			mode="connection"
			instance={editInstance}
			webspace={currentWebspace}
			config={localConfig}
			baseHash={localHash}
			{envVars}
			vocabulary={[]}
			extrasFields={[]}
			onclose={() => (editInstance = null)}
			onsaved={handleEditSaved}
		/>
	{/key}
{/if}

{#if webspaceRenameTarget}
	<Dialog open={true} onOpenChange={(o: boolean) => { if (!o) webspaceRenameTarget = null; }}>
		<DialogContent class="max-w-md" data-rename-webspace-dialog={webspaceRenameTarget}>
			<DialogHeader>
				<DialogTitle>Rename {webspaceRenameTarget}</DialogTitle>
			</DialogHeader>
			<p class="text-[14px] leading-[1.4] text-muted-foreground">
				Everything the webspace holds — its sources, filters, items and exclusions — follows the
				new name, and its address becomes /w/{renameValue.trim() || '…'}.
			</p>
			<Input
				bind:value={renameValue}
				aria-label="New webspace name"
				data-rename-webspace-input
				disabled={deleting}
			/>
			{#if renameError}
				<p class="text-[14px] leading-[1.4] text-destructive">{renameError}</p>
			{/if}
			<DialogFooter>
				<Button variant="ghost" onclick={() => (webspaceRenameTarget = null)} disabled={deleting}
					>Cancel</Button
				>
				<Button
					onclick={confirmRenameWebspace}
					disabled={deleting || renameValue.trim() === ''}
					data-rename-webspace-save>Rename</Button
				>
			</DialogFooter>
		</DialogContent>
	</Dialog>
{/if}
