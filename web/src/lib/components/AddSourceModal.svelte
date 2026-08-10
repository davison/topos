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
	import ConnectionForm from './ConnectionForm.svelte';
	import QRPanel from './QRPanel.svelte';
	import Plus from '@lucide/svelte/icons/plus';
	import {
		pluginTypeLabel,
		defaultConnectionValues,
		missingRequiredFields,
		missingRequiredFieldsMessage,
		WHATSAPP_PLUGIN_BINARY
	} from '$lib/plugin-fields';
	import { addSourceToWebspace, upsertSourceInstance } from '$lib/config-edit';
	import { participatingInstances } from '$lib/participation';
	import { resolveNewInstanceId } from '$lib/instance-id';
	import {
		describePlugin,
		putConfig,
		ApiError,
		CONFIG_CONFLICT_MESSAGE,
		type KernelConfig,
		type SourceConfig
	} from '$lib/api';

	let {
		webspace,
		config,
		baseHash,
		pluginTypes,
		envVars,
		onsaved
	}: {
		webspace: string;
		config: KernelConfig;
		baseHash: string;
		// pluginTypes is every discovered-but-not-necessarily-configured
		// plugin binary name (GET /api/config/plugin-types), excluding the
		// mock reference fixture — the picker's "New {plugin type}…" rows.
		pluginTypes: string[];
		// envVars is the last GET/PUT /api/config response's own env_vars
		// presence map — see SecretField.svelte's doc comment for why this
		// (not a per-keystroke lookup) is what the secret badge reads.
		envVars: Record<string, boolean>;
		// onsaved fires after any flow below completes a successful save —
		// the caller (the webspace route, via WebspaceHeader) refreshes
		// getConfig()/loadSources()/load() so the new chip and its items
		// appear without a page reload.
		onsaved: () => void;
	} = $props();

	let pickerOpen = $state(false);

	// Instances already participating in this webspace are never offered
	// again — `participatingInstances` (web/src/lib/participation.ts) is
	// the shared client-side mirror of the kernel's effective participation
	// (the allowlist gate, Phase 5 D-03, AND 07-11's D-20 has-match-input
	// rule), replacing this component's own former inline participant-set
	// derivation (07-14-PLAN.md Task 3, closes 07-UAT.md G-07-6's second
	// half). This predicate is now shared with the header's chip row
	// (WebspaceHeader.svelte), so the picker's "not yet in this webspace"
	// list and the chips it adds to can never disagree with each other.
	let webspaceParticipants = $derived(participatingInstances(config, webspace));
	let availableInstances = $derived(
		Object.keys(config.sources).filter((id) => !webspaceParticipants.has(id))
	);
	let pickerEmpty = $derived(availableInstances.length === 0 && pluginTypes.length === 0);

	// --- Shared save-in-flight state (D-09/D-03's shared error contract) ---
	// 'link' (08-04-PLAN.md Task 1, D-01/D-02) is the WhatsApp-only branch
	// between 'connect' and 'match': entered instead of 'match' when the
	// selected plugin type is WhatsApp and Step 1's trial launch succeeds
	// — QRPanel renders inline below the already-entered connection
	// fields, in this SAME Step 1 dialog, never a new one.
	let step = $state<'existing' | 'connect' | 'link' | 'match' | 'connect-saved' | null>(null);
	let selectedInstance = $state<string | null>(null);
	let existingVocabulary = $state<string[]>([]);
	let matchBlock = $state<Record<string, string[]>>({});
	let loadingVocabulary = $state(false);
	let saving = $state(false);
	let error = $state<string | null>(null);

	// --- Two-step new-instance flow state ---
	let selectedPluginType = $state<string | null>(null);
	let connectionValues = $state<SourceConfig>({ plugin: '', agent: { read: false, handoff: false } });
	let describing = $state(false);
	let describeFailed = $state(false);
	let connectError = $state<string | null>(null);
	let savingConnectionOnly = $state(false);
	let savedAnywayMessage = $state('');
	let newInstanceId = $state<string | null>(null);
	let newVocabulary = $state<string[]>([]);

	function resetFlowState() {
		step = null;
		selectedInstance = null;
		existingVocabulary = [];
		matchBlock = {};
		saving = false;
		error = null;
		selectedPluginType = null;
		connectionValues = { plugin: '', agent: { read: false, handoff: false } };
		describing = false;
		describeFailed = false;
		connectError = null;
		savingConnectionOnly = false;
		savedAnywayMessage = '';
		newInstanceId = null;
		newVocabulary = [];
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

	// selectPluginType opens Step 1 ("Connect {Plugin Type}") of the
	// two-step new-instance flow — nothing is persisted until Step 1's
	// Describe call succeeds (advancing to Step 2) or "Save anyway" is
	// used after a Describe failure.
	function selectPluginType(plugin: string) {
		pickerOpen = false;
		selectedPluginType = plugin;
		// defaultConnectionValues seeds a REAL, editable value for any field
		// that declares one (today, only Signal's path) — distinct from a
		// placeholder, which a user cannot submit and which is the specific
		// presentation that made leaving Signal's mandatory path untouched
		// the natural action (07-13-PLAN.md's G-07-5 diagnosis). The agent
		// grants shape is added on top exactly as it was before; that has
		// nothing to do with the connection-field table.
		connectionValues = {
			...defaultConnectionValues(plugin),
			agent: { read: false, handoff: false }
		} as SourceConfig;
		describeFailed = false;
		connectError = null;
		step = 'connect';
	}

	function handleExistingOpenChange(next: boolean) {
		if (!next) resetFlowState();
	}

	function handleConnectOpenChange(next: boolean) {
		if (!next) resetFlowState();
	}

	function backToConnect() {
		step = 'connect';
		error = null;
	}

	// handleLinkPaired (D-02's Add-Source success transition): the QR
	// panel's own paired callback advances straight to the existing
	// 'match' step, reusing the vocabulary handleConnectNext's Describe
	// call already returned — no second describe round-trip.
	function handleLinkPaired() {
		step = 'match';
	}

	// handleLinkCancelled (D-02): cancelling out of the link step returns
	// to 'connect' with every typed connection value intact —
	// connectionValues is untouched here, and the existing "saved but not
	// yet linked" outcome (Save-anyway from Step 2, or simply moving on)
	// remains a valid, supported path.
	function handleLinkCancelled() {
		step = 'connect';
	}

	// handleConnectNext trial-launches the plugin against Step 1's
	// just-typed, not-yet-persisted connection fields (describePlugin,
	// 07-02) — writing nothing. On success it advances to Step 2 with the
	// returned match vocabulary; on failure (including Proton's own
	// base-url-scheme rejection, which surfaces through this exact path
	// with the plugin's own message — not a special case to code around)
	// it renders the fixed failure copy plus the kernel's own error text
	// and reveals "Save anyway".
	async function handleConnectNext(event: SubmitEvent) {
		event.preventDefault();
		if (!selectedPluginType || describing) return;

		// A blank required field must never reach describePlugin — that call
		// launches the plugin subprocess with these exact values, and the
		// plugin's own pre-goplugin.Serve guard fatals on a blank mandatory
		// field before the handshake, producing an unexplained-until-07-13
		// error (07-UAT.md G-07-5). This check runs before the collision
		// guard below so a missing-field message never gets shadowed by an
		// id check that would otherwise pass.
		const missing = missingRequiredFields(selectedPluginType, connectionValues);
		if (missing.length > 0) {
			describeFailed = false;
			connectError = missingRequiredFieldsMessage(missing);
			return;
		}

		const displayName = (connectionValues.display_name ?? '').trim();
		const idResult = resolveNewInstanceId(config, displayName);
		if (!idResult.ok) {
			describeFailed = false;
			connectError = idResult.message;
			return;
		}

		describing = true;
		describeFailed = false;
		connectError = null;
		try {
			const resp = await describePlugin({ plugin: selectedPluginType, source: connectionValues });
			newInstanceId = idResult.id;
			newVocabulary = resp.match_vocabulary;
			matchBlock = {};
			// D-01/D-02: Describe carries only static match vocabulary and
			// succeeds identically whether or not a WhatsApp device is
			// paired — there is no field on this response that reports
			// linked status, so this trial-launch success is itself the
			// only signal Step 1 has. For WhatsApp, that success routes to
			// the 'link' step (the QR opportunity) rather than straight to
			// 'match'; the load-bearing property is what this branch does
			// NOT do — describeFailed stays false and the Save-anyway
			// control (gated on it) stays hidden, so an unpaired instance
			// is never treated as a trial-launch failure.
			step = selectedPluginType === WHATSAPP_PLUGIN_BINARY ? 'link' : 'match';
		} catch (err) {
			describeFailed = true;
			const detail =
				err instanceof ApiError
					? err.message
					: 'check the browser console and try again.';
			connectError = `Couldn't verify this connection. ${detail}`;
		} finally {
			describing = false;
		}
	}

	// saveAnyway persists the connection-only instance immediately after a
	// Describe failure — the one deliberate connection-only write in this
	// flow, closing the modal and surfacing a follow-up notice naming
	// where to add match settings once the source can connect.
	async function saveAnyway() {
		if (!selectedPluginType || savingConnectionOnly) return;

		// Save anyway means "persist despite a failed connection test", never
		// "persist an instance the plugin cannot start" — a connection-only
		// instance missing a mandatory field would fail every subsequent
		// hot-apply reconcile. Same guard as handleConnectNext, same message,
		// before saveAnyway's own collision guard.
		const missing = missingRequiredFields(selectedPluginType, connectionValues);
		if (missing.length > 0) {
			connectError = missingRequiredFieldsMessage(missing);
			return;
		}

		const displayName = (connectionValues.display_name ?? '').trim();
		const idResult = resolveNewInstanceId(config, displayName);
		if (!idResult.ok) {
			connectError = idResult.message;
			return;
		}

		connectError = null;
		savingConnectionOnly = true;
		try {
			const nextConfig = upsertSourceInstance(config, idResult.id, connectionValues);
			await putConfig({ base_hash: baseHash, config: nextConfig });
			savedAnywayMessage = `Saved. Add match settings from ${displayName || idResult.id}'s menu once it can connect.`;
			step = 'connect-saved';
			onsaved();
		} catch (err) {
			connectError =
				err instanceof ApiError && err.code === 'config_changed_on_disk'
					? CONFIG_CONFLICT_MESSAGE
					: err instanceof ApiError
						? err.message
						: 'Something went wrong saving this connection — check the browser console and try again.';
		} finally {
			savingConnectionOnly = false;
		}
	}

	// submitMatch performs ONE save writing the source block, the match
	// block and the extended sources allowlist together — two sequential
	// saves would leave a configured-but-unparticipating instance if the
	// second failed, and would burn the base hash.
	async function submitMatch(event: SubmitEvent) {
		event.preventDefault();
		if (!newInstanceId || !selectedPluginType || saving) return;
		saving = true;
		error = null;
		try {
			const withSource = upsertSourceInstance(config, newInstanceId, connectionValues);
			const nextConfig = addSourceToWebspace(withSource, webspace, newInstanceId, matchBlock);
			await putConfig({ base_hash: baseHash, config: nextConfig });
			resetFlowState();
			onsaved();
		} catch (err) {
			error =
				err instanceof ApiError && err.code === 'config_changed_on_disk'
					? CONFIG_CONFLICT_MESSAGE
					: err instanceof ApiError
						? err.message
						: 'Something went wrong adding this source — check the browser console and try again.';
		} finally {
			saving = false;
		}
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
					? CONFIG_CONFLICT_MESSAGE
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

<!--
  Two-step "New {plugin type}…" flow (D-11): Step 1 connects a new
  instance (ConnectionForm, static per-plugin-type fields), Step 2 asks
  only for match settings, driven by whatever vocabulary Step 1's
  describePlugin call returned — no field names are hardcoded per plugin
  type here. A distinct Dialog from the one-step existing-instance modal
  above (max-w-lg, wider — Step 1 carries more fields), gated on every
  step this flow can be in, including the post-"Save anyway" confirmation.
-->
<Dialog
	open={step === 'connect' || step === 'link' || step === 'match' || step === 'connect-saved'}
	onOpenChange={handleConnectOpenChange}
>
	<DialogContent class="max-w-lg">
		{#if step === 'connect-saved'}
			<DialogHeader>
				<DialogTitle
					>Connect {selectedPluginType ? pluginTypeLabel(selectedPluginType) : ''}</DialogTitle
				>
			</DialogHeader>
			<p class="text-[14px] leading-[1.4] text-foreground">{savedAnywayMessage}</p>
			<DialogFooter>
				<Button type="button" onclick={resetFlowState}>Done</Button>
			</DialogFooter>
		{:else}
			<div class="flex items-center gap-1.5 text-[14px] leading-[1.4] text-muted-foreground">
				<span class={step === 'connect' ? 'font-semibold text-foreground' : ''}>1. Connect</span>
				<span aria-hidden="true">/</span>
				<span class={step === 'match' ? 'font-semibold text-foreground' : ''}>2. Match</span>
			</div>

			{#if step === 'connect'}
				<DialogHeader>
					<DialogTitle
						>Connect {selectedPluginType ? pluginTypeLabel(selectedPluginType) : ''}</DialogTitle
					>
				</DialogHeader>
				<form onsubmit={handleConnectNext}>
					<ConnectionForm
						pluginBinary={selectedPluginType ?? ''}
						values={connectionValues}
						{envVars}
						onchange={(next) => (connectionValues = next)}
					/>

					{#if connectError}
						<Alert variant="destructive" class="mt-4">
							<AlertDescription>{connectError}</AlertDescription>
						</Alert>
					{/if}

					<DialogFooter>
						<Button type="button" variant="ghost" onclick={resetFlowState}>Cancel</Button>
						{#if describeFailed}
							<Button
								type="button"
								variant="ghost"
								disabled={savingConnectionOnly}
								onclick={saveAnyway}
							>
								Save anyway
							</Button>
						{/if}
						<Button type="submit" disabled={describing}>Next</Button>
					</DialogFooter>
				</form>
			{:else if step === 'link'}
				<!--
				  D-01/D-02: the QR panel renders inline, INSIDE this same
				  Step 1 dialog, below the already-entered connection
				  fields — never a new Dialog (08-UI-SPEC.md's Amendment:
				  "the panel renders inline below the connection fields ...
				  inside the existing two-step modal's DialogContent — no
				  width change to that dialog"). ConnectionForm stays
				  visible (and editable) above it so the user can see what
				  they entered while scanning; QRPanel owns its own Cancel
				  control, so no separate dialog footer is rendered here.
				-->
				<DialogHeader>
					<DialogTitle
						>Connect {selectedPluginType ? pluginTypeLabel(selectedPluginType) : ''}</DialogTitle
					>
				</DialogHeader>
				<div class="flex flex-col gap-4">
					<ConnectionForm
						pluginBinary={selectedPluginType ?? ''}
						values={connectionValues}
						{envVars}
						onchange={(next) => (connectionValues = next)}
					/>
					<QRPanel
						plugin={selectedPluginType ?? ''}
						path={connectionValues.path ?? ''}
						onpaired={handleLinkPaired}
						oncancelled={handleLinkCancelled}
					/>
				</div>
			{:else if step === 'match'}
				<DialogHeader>
					<DialogTitle>Match settings for {webspace}</DialogTitle>
				</DialogHeader>
				<form onsubmit={submitMatch}>
					<MatchFieldsForm
						vocabulary={newVocabulary}
						values={matchBlock}
						onchange={(block) => (matchBlock = block)}
					/>

					{#if error}
						<Alert variant="destructive" class="mt-4">
							<AlertDescription>{error}</AlertDescription>
						</Alert>
					{/if}

					<DialogFooter>
						<Button type="button" variant="ghost" onclick={backToConnect}>Back</Button>
						<Button type="submit" disabled={saving}>Add source</Button>
					</DialogFooter>
				</form>
			{/if}
		{/if}
	</DialogContent>
</Dialog>
