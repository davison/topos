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
	import { Input } from '$lib/components/ui/input/index.js';
	import { Alert, AlertDescription } from '$lib/components/ui/alert/index.js';
	import MatchFieldsForm from './MatchFieldsForm.svelte';
	import ConnectionForm from './ConnectionForm.svelte';
	import QRPanel from './QRPanel.svelte';
	import PluginIcon from '$lib/components/PluginIcon.svelte';
	import TrustBadge from './TrustBadge.svelte';
	import Plus from '@lucide/svelte/icons/plus';
	import {
		pluginTypeLabel,
		defaultConnectionValues,
		missingRequiredFields,
		missingRequiredFieldsMessage,
		isExternalTier,
		UNTRUSTED_LABEL,
		extrasKeyError,
		WHATSAPP_PLUGIN_BINARY,
		type ExtrasRow
	} from '$lib/plugin-fields';
	import { addSourceToWebspace, upsertSourceInstance, setPluginPin } from '$lib/config-edit';
	import { participatingInstances } from '$lib/participation';
	import { resolveNewInstanceId } from '$lib/instance-id';
	import {
		describePlugin,
		putConfig,
		ApiError,
		CONFIG_CONFLICT_MESSAGE,
		type KernelConfig,
		type SourceConfig,
		type ExtrasFieldDecl
	} from '$lib/api';

	let {
		webspace,
		config,
		baseHash,
		pluginTypes,
		pluginTypeTiers,
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
		// pluginTypeTiers (Phase 11, PLUG-06/08): GET /api/config/plugin-types'
		// own plugin_type_tiers lookup table, spanning every discovered
		// binary in BOTH plugin directories — threaded through
		// WebspaceHeader alongside pluginTypes so this component's picker
		// rows (E2/E3) and the untrusted-confirm step (E1) can key off tier
		// without a second network round trip.
		pluginTypeTiers: Record<string, string>;
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
	let step = $state<
		'existing' | 'connect' | 'link' | 'untrusted-confirm' | 'match' | 'connect-saved' | null
	>(null);
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
	// linkOffered (D-02, 08-04-PLAN.md Task 3's E5 evidence): tracks
	// whether the WhatsApp QR opportunity has already been shown once in
	// THIS modal session. The first successful trial launch for WhatsApp
	// routes to 'link'; a later "Next" click (reached after the panel's
	// own cancelled callback returns to 'connect') routes straight to
	// 'match' instead of re-showing the panel — the opportunity was
	// offered once, cancelling it is a real decision to move on, not a
	// gate that must be satisfied before Step 2 is reachable. This is
	// what makes "cancelling out still reaches the match step and saves"
	// true, per the UI-SPEC amendment's own framing that Step 1 and Step
	// 2 can both succeed while the device stays unpaired.
	let linkOffered = $state(false);

	// linkNotice (D-02, 08-07-PLAN.md Task 1, Amendment 2/G-08-1): cancelling
	// out of the QR panel is a supported, non-failure outcome (Amendment 1's
	// E5 state), but the flow previously said nothing at all about it — a
	// user who had actually completed a pairing on their phone and then
	// closed the dialog was left with no idea that the pairing was real,
	// recoverable, or re-enterable. This holds the neutral notice copy shown
	// on the connect step once a link opportunity has been declined.
	let linkNotice = $state('');

	// --- Untrusted-source confirm step state (Phase 11, PLUG-08, D-05/D-14,
	// 11-UI-SPEC.md E1) ---
	// pendingBinaryHash/pendingEnvVarNames are the kernel-computed facts
	// handleConnectNext's own describe response already returned for an
	// external-tier plugin type — never computed or invented client-side
	// (T-11-25). confirmTyped is the type-to-confirm input's own local
	// value; the primary action stays disabled until it exactly equals
	// selectedPluginType (case-sensitive).
	let pendingBinaryHash = $state('');
	let pendingEnvVarNames = $state<string[]>([]);
	let confirmTyped = $state('');

	// --- Extras (Phase 11, PLUG-09, D-12/D-13/D-15, 11-UI-SPEC.md E6) ---
	// declaredExtras is the plugin's Describe-declared expected extras keys
	// — learned via a best-effort trial describe on plugin-type selection
	// (selectPluginType, below) and refreshed by handleConnectNext's own
	// describe response, never persisted by either call. extrasRows is
	// ConnectionForm's own free-form (undeclared) row editor state, bound
	// here (not owned by ConnectionForm) so extrasKeyError below can
	// validate the SAME rows the form renders, at the same submit-time
	// point missingRequiredFields already runs.
	let declaredExtras = $state<ExtrasFieldDecl[]>([]);
	let extrasRows = $state<ExtrasRow[]>([]);

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
		linkOffered = false;
		linkNotice = '';
		pendingBinaryHash = '';
		pendingEnvVarNames = [];
		confirmTyped = '';
		declaredExtras = [];
		extrasRows = [];
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
		linkNotice = '';
		pendingBinaryHash = '';
		pendingEnvVarNames = [];
		confirmTyped = '';
		declaredExtras = [];
		extrasRows = [];
		step = 'connect';
		// Best-effort, silent trial describe (Task 2, D-15) purely to learn
		// this plugin type's declared extras keys BEFORE the operator has
		// typed anything — the same trial launch handleConnectNext's own
		// Next click already performs (see that function's own describePlugin
		// call), fired one step earlier so the extras form's declared inputs
		// render from the Connect step's very first paint rather than only
		// after Next. The picker row that led here already carried the
		// untrusted label (E3) if this plugin type is external-tier — this
		// call persists nothing, and a failure here is silent: declarations
		// simply stay empty and the free-form editor remains fully usable.
		void loadDeclaredExtras(plugin, connectionValues);
	}

	async function loadDeclaredExtras(plugin: string, source: SourceConfig) {
		try {
			const resp = await describePlugin({ plugin, source });
			// Staleness guard: a slower response landing after the operator
			// has already moved on to a different plugin type (or closed the
			// modal) must never overwrite that later selection's declarations.
			if (selectedPluginType !== plugin) return;
			declaredExtras = resp.extras;
		} catch {
			if (selectedPluginType !== plugin) return;
			declaredExtras = [];
		}
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
	// remains a valid, supported path. Sets linkNotice to the neutral copy
	// locked in 08-UI-SPEC.md's Amendment 2 (08-07-PLAN.md Task 1,
	// G-08-1) — declining a link opportunity is not a failed connection
	// test, so neither describeFailed nor connectError is touched here;
	// the Save-anyway control (gated on describeFailed) must stay hidden.
	function handleLinkCancelled() {
		step = 'connect';
		linkNotice =
			'Not linked yet — you can save this source now and link later from its menu (Re-link…).';
	}

	// cancelUntrustedConfirm (E1 interaction): returns to 'connect' with
	// every typed connection value intact, mirroring handleLinkCancelled's
	// own untouched-connectionValues discipline — only the confirm step's
	// OWN typed-name state is cleared, since re-entering the confirm step
	// later must start from an empty confirmation box again.
	function cancelUntrustedConfirm() {
		step = 'connect';
		confirmTyped = '';
	}

	// confirmUntrusted (D-05, E1): a pure step transition, never a network
	// call — the pin write rides the SAME single putConfig call the Match
	// step's submitMatch already issues (see that function below), so
	// there is nothing here that can itself fail.
	function confirmUntrusted() {
		if (!selectedPluginType || confirmTyped !== selectedPluginType) return;
		step = 'match';
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
		// A fresh trial-launch attempt must not inherit a prior outcome's
		// message (08-REVIEW.md WR-01) — mirrors how selectPluginType
		// already resets this on a fresh plugin-type selection. Cleared
		// here, before the missing-required-fields early return below: a
		// clear placed after that early return would leave the stale
		// notice standing in exactly the branch most likely to render a
		// second, contradictory message.
		linkNotice = '';

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

		// Same submit-time point missingRequiredFields already gates —
		// an empty or duplicate extras key must never reach describePlugin
		// either (D-15's validation guard, Task 2).
		const extrasErr = extrasKeyError(declaredExtras, extrasRows);
		if (extrasErr) {
			describeFailed = false;
			connectError = extrasErr;
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
			// Refresh declarations from this same describe response (Task 2)
			// so returning to Connect via Back still shows them — no second
			// call needed, this response already carries `extras`.
			declaredExtras = resp.extras;
			// D-01/D-02: Describe carries only static match vocabulary and
			// succeeds identically whether or not a WhatsApp device is
			// paired — there is no field on this response that reports
			// linked status, so this trial-launch success is itself the
			// only signal Step 1 has. For WhatsApp, the FIRST such success
			// routes to the 'link' step (the QR opportunity); once that
			// opportunity has been offered and declined (linkOffered),
			// a later success routes straight to 'match' instead of
			// re-showing the panel. The load-bearing property either way
			// is what this branch does NOT do — describeFailed stays
			// false and the Save-anyway control (gated on it) stays
			// hidden, so an unpaired instance is never treated as a
			// trial-launch failure.
			//
			// External-tier plugin types (Phase 11, D-05) route to the new
			// untrusted-confirm step instead of straight to Match — this
			// check runs after the WhatsApp branch so WhatsApp's own
			// (always trusted-tier) link opportunity is unaffected.
			if (selectedPluginType === WHATSAPP_PLUGIN_BINARY && !linkOffered) {
				linkOffered = true;
				step = 'link';
			} else if (isExternalTier(pluginTypeTiers, selectedPluginType)) {
				pendingBinaryHash = resp.binary_hash;
				pendingEnvVarNames = resp.env_var_names;
				confirmTyped = '';
				step = 'untrusted-confirm';
			} else {
				step = 'match';
			}
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

		const extrasErr = extrasKeyError(declaredExtras, extrasRows);
		if (extrasErr) {
			connectError = extrasErr;
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
			const withMatch = addSourceToWebspace(withSource, webspace, newInstanceId, matchBlock);
			// External-tier plugin types (D-01/D-02, E1 interaction): the pin
			// write rides this SAME single putConfig call — no extra network
			// round trip — using pendingBinaryHash, the kernel-computed hash
			// the untrusted-confirm step already displayed and the operator
			// already confirmed against.
			const nextConfig = isExternalTier(pluginTypeTiers, selectedPluginType)
				? setPluginPin(withMatch, selectedPluginType, pendingBinaryHash)
				: withMatch;
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
	<PopoverContent class="w-80 max-h-80 overflow-y-auto">
		{#if pickerEmpty}
			<p class="text-[14px] leading-[1.4] text-muted-foreground">
				All available sources are already in this webspace.
			</p>
		{:else}
			<div class="flex flex-col gap-1">
				{#if availableInstances.length > 0}
					<p
						class="px-2 pt-1 text-[14px] leading-[1.4] font-medium tracking-wide text-muted-foreground uppercase"
					>
						Add to this webspace
					</p>
					{#each availableInstances as instanceId (instanceId)}
						{@const source = config.sources[instanceId]}
						{@const location = source.base_url || source.path || pluginTypeLabel(source.plugin)}
						{@const untrusted = isExternalTier(pluginTypeTiers, source.plugin)}
						<button
							type="button"
							class="flex items-start gap-1.5 rounded-sm px-2 py-1.5 text-left hover:bg-muted"
							onclick={() => selectExisting(instanceId)}
						>
							<TrustBadge tier={untrusted ? 'external' : 'trusted'} scale="picker">
								<PluginIcon plugin={source.plugin} size="size-4 mt-0.5 shrink-0" />
							</TrustBadge>
							<span class="flex min-w-0 flex-col">
								<span class="text-[14px] leading-[1.4] text-foreground">
									{source.display_name ?? instanceId}
								</span>
								<span
									class="truncate text-[14px] leading-[1.4] text-muted-foreground"
									title={location}
								>
									{location}
								</span>
								{#if untrusted}
									<span
										class="text-[14px] leading-[1.4] font-medium tracking-wide text-warning uppercase"
									>
										{UNTRUSTED_LABEL}
									</span>
								{/if}
							</span>
						</button>
					{/each}
				{/if}

				{#if availableInstances.length > 0 && pluginTypes.length > 0}
					<div class="-mx-1 my-1 h-px bg-border"></div>
				{/if}

				{#if pluginTypes.length > 0}
					<p
						class="px-2 pt-1 text-[14px] leading-[1.4] font-medium tracking-wide text-muted-foreground uppercase"
					>
						Install a new source
					</p>
					{#each pluginTypes as plugin (plugin)}
						{@const untrusted = isExternalTier(pluginTypeTiers, plugin)}
						<button
							type="button"
							class="flex items-center justify-between gap-1.5 rounded-md border border-border p-2 text-left text-[14px] leading-[1.4] text-foreground hover:border-primary hover:bg-muted"
							onclick={() => selectPluginType(plugin)}
						>
							<span class="flex items-center gap-1.5">
								<TrustBadge tier={untrusted ? 'external' : 'trusted'} scale="picker">
									<PluginIcon plugin={plugin} size="size-4 shrink-0" />
								</TrustBadge>
								{pluginTypeLabel(plugin)}
							</span>
							{#if untrusted}
								<span
									class="ml-auto text-[14px] leading-[1.4] font-medium tracking-wide text-warning uppercase"
								>
									{UNTRUSTED_LABEL}
								</span>
							{/if}
						</button>
					{/each}
				{/if}
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
	open={step === 'connect' || step === 'link' || step === 'untrusted-confirm' || step === 'match' || step === 'connect-saved'}
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
						extrasFields={declaredExtras}
						bind:extrasRows
						onchange={(next) => (connectionValues = next)}
					/>

					{#if connectError}
						<Alert variant="destructive" class="mt-4">
							<AlertDescription>{connectError}</AlertDescription>
						</Alert>
					{/if}

					{#if linkNotice}
						<!--
						  08-UI-SPEC.md Amendment 2 (G-08-1): deliberately NOT an
						  Alert and NOT the destructive variant — declining to
						  link is not a failed connection test (Amendment 1's E5
						  evidence), and e2e case 8's assertion that reaching and
						  leaving the link step produces no failure alert and no
						  Save-anyway control depends on this staying a plain
						  muted paragraph.
						-->
						<p class="mt-4 text-[14px] leading-[1.4] text-muted-foreground">{linkNotice}</p>
					{/if}

					<DialogFooter>
						<Button type="button" variant="ghost" onclick={resetFlowState}>Cancel</Button>
						<!--
						  Save anyway is never offered for an external-tier plugin
						  type (D-05 lineage, T-11-27): a connection-only save
						  could not be pinned — no successful describe means no
						  kernel-computed hash — so it would create an
						  unstartable source that never showed the untrusted
						  warning.
						-->
						{#if describeFailed && !(selectedPluginType && isExternalTier(pluginTypeTiers, selectedPluginType))}
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
						extrasFields={declaredExtras}
						bind:extrasRows
						onchange={(next) => (connectionValues = next)}
					/>
					<QRPanel
						plugin={selectedPluginType ?? ''}
						path={connectionValues.path ?? ''}
						onpaired={handleLinkPaired}
						oncancelled={handleLinkCancelled}
					/>
				</div>
			{:else if step === 'untrusted-confirm'}
				<!--
				  E1 — Untrusted-Source Confirm Interstitial (D-05, D-14):
				  inserted between a successful trial-launch describe (Step 1)
				  and Step 2 ("Match"), only for an external-tier plugin type.
				  A pure step transition — confirmUntrusted below issues no
				  network call of its own; the pin write rides the SAME
				  putConfig call submitMatch already issues.
				-->
				<DialogHeader>
					<DialogTitle>Add an untrusted source</DialogTitle>
				</DialogHeader>
				<div class="flex flex-col gap-4">
					<p class="text-[14px] leading-[1.5] text-foreground">
						{selectedPluginType} lives outside topos's own plugin directory — this is code topos
						didn't build and can't vouch for. It runs with the same access as any other plugin
						process; topos does not sandbox it.
					</p>
					<div class="rounded-md border border-border bg-card p-3">
						<p class="text-[14px] leading-[1.4]">Binary: {selectedPluginType}</p>
						<p class="font-mono text-[14px] leading-[1.4] break-all">
							Pinned hash (SHA-256): {pendingBinaryHash}
						</p>
					</div>
					{#if pendingEnvVarNames.length === 0}
						<p class="text-[14px] leading-[1.4] text-muted-foreground">
							topos will hand this plugin only the standard PATH/HOME/locale environment — this
							source's configuration references no environment variables.
						</p>
					{:else}
						<p class="text-[14px] leading-[1.4] text-muted-foreground">
							topos will also hand this plugin the standard PATH/HOME/locale environment, plus
							the values behind these variables referenced in this source's own configuration: {pendingEnvVarNames.join(
								', '
							)}.
						</p>
					{/if}
					<div class="flex flex-col gap-1">
						<label
							for="untrusted-confirm-typed"
							class="text-[14px] leading-[1.4] text-foreground"
						>
							Type {selectedPluginType} to confirm
						</label>
						<Input
							id="untrusted-confirm-typed"
							value={confirmTyped}
							placeholder={selectedPluginType ?? ''}
							oninput={(e) => (confirmTyped = e.currentTarget.value)}
						/>
					</div>
				</div>
				<DialogFooter>
					<Button type="button" variant="ghost" onclick={cancelUntrustedConfirm}>Cancel</Button>
					<Button
						type="button"
						disabled={confirmTyped !== selectedPluginType}
						onclick={confirmUntrusted}
					>
						Add untrusted source
					</Button>
				</DialogFooter>
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
