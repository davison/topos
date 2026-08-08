<script lang="ts">
	// The "+" chip-row affordance (D-11): the picker plus the one-step
	// (existing instance) and two-step (new plugin-type instance) add
	// flows, all in one self-contained component.
	//
	// Architecture note (CONTEXT.md's "Claude's Discretion: picker
	// presentation"): this component is mounted DIRECTLY INLINE inside
	// WebspaceHeader.svelte's chip row, rather than at the route level
	// with an externally-toggled `open` boolean the way CreateWebspaceModal
	// is. bits-ui's Popover.Root/Trigger/Content must live in one
	// component subtree — the picker's Popover trigger has to be the
	// physical `+` button the chip row measures for overflow accounting
	// (WebspaceHeader.svelte's visibleChipCount math), so splitting the
	// trigger into WebspaceHeader and the picker's own content into a
	// separately-mounted modal is not composable. The two Dialog flows
	// below are unaffected by this placement — DialogContent already
	// portals to <body> regardless of where its component sits in the
	// tree (ui/dialog/dialog-content.svelte).
	import { Popover, PopoverTrigger, PopoverContent } from '$lib/components/ui/popover/index.js';
	import {
		Dialog,
		DialogContent,
		DialogHeader,
		DialogTitle,
		DialogFooter
	} from '$lib/components/ui/dialog/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import { Alert, AlertDescription } from '$lib/components/ui/alert/index.js';
	import MatchFieldsForm from './MatchFieldsForm.svelte';
	import Plus from '@lucide/svelte/icons/plus';
	import { pluginTypeLabel } from '$lib/plugin-fields';
	import { addSourceToWebspace } from '$lib/config-edit';
	import { describePlugin, putConfig, ApiError, type KernelConfig } from '$lib/api';

	let {
		webspace,
		config,
		baseHash,
		pluginTypes,
		onsaved
	}: {
		webspace: string;
		config: KernelConfig;
		baseHash: string;
		// pluginTypes is every discovered-but-not-necessarily-configured
		// plugin binary name (GET /api/config/plugin-types), excluding the
		// mock reference fixture — the picker's "New {plugin type}…" rows.
		pluginTypes: string[];
		// onsaved fires after any flow below completes a successful save —
		// the caller (the webspace route, via WebspaceHeader) refreshes
		// getConfig()/loadSources()/load() so the new chip and its items
		// appear without a page reload.
		onsaved: () => void;
	} = $props();

	let pickerOpen = $state(false);

	// Instances already participating in this webspace are never offered
	// again — an empty `sources` allowlist means every configured instance
	// participates by default (Phase 5 D-03's Webspace.Participates rule,
	// mirrored here).
	let participatingSet = $derived.by(() => {
		const ws = config.webspaces[webspace];
		const sources = ws?.sources ?? [];
		return new Set(sources.length > 0 ? sources : Object.keys(config.sources));
	});
	let availableInstances = $derived(
		Object.keys(config.sources).filter((id) => !participatingSet.has(id))
	);
	let pickerEmpty = $derived(availableInstances.length === 0 && pluginTypes.length === 0);

	// --- Shared save-in-flight state (D-09/D-03's shared error contract) ---
	let step = $state<'existing' | null>(null);
	let selectedInstance = $state<string | null>(null);
	let existingVocabulary = $state<string[]>([]);
	let matchBlock = $state<Record<string, string[]>>({});
	let loadingVocabulary = $state(false);
	let saving = $state(false);
	let error = $state<string | null>(null);

	function resetFlowState() {
		step = null;
		selectedInstance = null;
		existingVocabulary = [];
		matchBlock = {};
		saving = false;
		error = null;
	}

	// selectExisting reads the instance's declared match vocabulary via
	// describePlugin, trial-launched against that instance's OWN
	// already-stored (raw, ${VAR}-reference) connection config. This is a
	// deliberate substitute for reading match_vocabulary off the
	// already-fetched GET /api/sources payload: that response
	// (kernel/httpapi/sources.go's sourceStatus) does not carry the
	// vocabulary today, and this plan's own files_modified scope carries
	// no kernel/ files to add it there. describePlugin (07-02) already
	// exists for exactly this "learn a plugin's declared vocabulary
	// without persisting anything" job — an already-configured instance's
	// stored Source trial-launches identically to a not-yet-configured
	// one.
	async function selectExisting(instanceId: string) {
		pickerOpen = false;
		loadingVocabulary = true;
		error = null;
		try {
			const source = config.sources[instanceId];
			const resp = await describePlugin({ plugin: source.plugin, source });
			existingVocabulary = resp.match_vocabulary;
			selectedInstance = instanceId;
			matchBlock = {};
			step = 'existing';
		} catch (err) {
			error =
				err instanceof ApiError
					? err.message
					: `Could not load match fields for ${instanceId} — check the browser console and try again.`;
		} finally {
			loadingVocabulary = false;
		}
	}

	// selectPluginType is wired by the picker below (Task 2 fills in the
	// two-step Connect/Match flow this opens).
	function selectPluginType(plugin: string) {
		pickerOpen = false;
		void plugin;
	}

	function handleExistingOpenChange(next: boolean) {
		if (!next) resetFlowState();
	}

	async function submitExisting(event: SubmitEvent) {
		event.preventDefault();
		if (!selectedInstance || saving) return;
		saving = true;
		error = null;
		try {
			const nextConfig = addSourceToWebspace(config, webspace, selectedInstance, matchBlock);
			await putConfig({ base_hash: baseHash, config: nextConfig });
			resetFlowState();
			onsaved();
		} catch (err) {
			// D-09/D-03's shared error contract: the modal stays open with
			// entered values intact on rejection.
			error =
				err instanceof ApiError && err.code === 'config_changed_on_disk'
					? 'Config changed on disk — review and retry.'
					: err instanceof ApiError
						? err.message
						: 'Something went wrong adding this source — check the browser console and try again.';
		} finally {
			saving = false;
		}
	}

	let selectedDisplayName = $derived(
		selectedInstance ? (config.sources[selectedInstance]?.display_name ?? selectedInstance) : ''
	);
</script>

<Popover bind:open={pickerOpen}>
	<PopoverTrigger>
		{#snippet child({ props })}
			<button
				{...props}
				type="button"
				aria-label="Add source"
				disabled={loadingVocabulary}
				class="flex size-11 shrink-0 items-center justify-center rounded-full border border-dashed border-border text-muted-foreground hover:border-primary hover:text-primary"
			>
				<Plus class="size-5" aria-hidden="true" />
			</button>
		{/snippet}
	</PopoverTrigger>
	<PopoverContent class="max-h-80 overflow-y-auto">
		{#if pickerEmpty}
			<p class="text-[14px] leading-[1.4] text-muted-foreground">
				All available sources are already in this webspace.
			</p>
		{:else}
			<div class="flex flex-col gap-1">
				{#each availableInstances as instanceId (instanceId)}
					{@const source = config.sources[instanceId]}
					<button
						type="button"
						class="flex flex-col items-start rounded-sm px-2 py-1.5 text-left hover:bg-muted"
						onclick={() => selectExisting(instanceId)}
					>
						<span class="text-[14px] leading-[1.4] text-foreground">
							{source.display_name ?? instanceId}
						</span>
						<span class="text-[14px] leading-[1.4] text-muted-foreground">
							{source.plugin}
						</span>
					</button>
				{/each}

				{#if availableInstances.length > 0 && pluginTypes.length > 0}
					<div class="-mx-1 my-1 h-px bg-border"></div>
				{/if}

				{#each pluginTypes as plugin (plugin)}
					<button
						type="button"
						class="flex items-center gap-1.5 rounded-sm px-2 py-1.5 text-left text-[14px] leading-[1.4] text-foreground hover:bg-muted"
						onclick={() => selectPluginType(plugin)}
					>
						<Plus class="size-3.5 shrink-0 text-muted-foreground" aria-hidden="true" />
						New {pluginTypeLabel(plugin)}…
					</button>
				{/each}
			</div>
		{/if}
	</PopoverContent>
</Popover>

<Dialog open={step === 'existing'} onOpenChange={handleExistingOpenChange}>
	<DialogContent class="max-w-md">
		<DialogHeader>
			<DialogTitle>Add {selectedDisplayName} to {webspace}</DialogTitle>
		</DialogHeader>
		<form onsubmit={submitExisting}>
			<MatchFieldsForm
				vocabulary={existingVocabulary}
				values={matchBlock}
				onchange={(block) => (matchBlock = block)}
			/>

			{#if error}
				<Alert variant="destructive" class="mt-4">
					<AlertDescription>{error}</AlertDescription>
				</Alert>
			{/if}

			<DialogFooter>
				<Button type="button" variant="ghost" onclick={resetFlowState}>Cancel</Button>
				<Button type="submit" disabled={saving}>Add source</Button>
			</DialogFooter>
		</form>
	</DialogContent>
</Dialog>
