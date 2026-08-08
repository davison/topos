<script lang="ts">
	// Single-field create-webspace dialog (07-UI-SPEC.md "Create Webspace
	// modal"), reused unchanged by both entry points: the switcher's
	// "+ New webspace" item and the root route's zero-webspaces empty-state
	// CTA (07-03-PLAN.md Task 3).
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
	import { addWebspace } from '$lib/config-edit';
	import { putConfig, ApiError, CONFIG_CONFLICT_MESSAGE, type KernelConfig } from '$lib/api';

	let {
		open,
		config,
		baseHash,
		onclose,
		oncreated
	}: {
		open: boolean;
		config: KernelConfig;
		baseHash: string;
		onclose: () => void;
		oncreated: (name: string) => void;
	} = $props();

	let name = $state('');
	let saving = $state(false);
	let error = $state<string | null>(null);

	// Reset local form state whenever the modal transitions to open, so a
	// second open after a prior create/cancel never shows stale input or a
	// stale error from the last time it was open.
	$effect(() => {
		if (open) {
			name = '';
			saving = false;
			error = null;
		}
	});

	function handleOpenChange(next: boolean) {
		if (!next) onclose();
	}

	async function handleSubmit(event: SubmitEvent) {
		event.preventDefault();
		const trimmed = name.trim();
		if (trimmed === '' || saving) return;

		saving = true;
		error = null;
		try {
			const nextConfig = addWebspace(config, trimmed);
			await putConfig({ base_hash: baseHash, config: nextConfig });
			oncreated(trimmed);
		} catch (err) {
			// The typed name (`name`) is deliberately left untouched here —
			// D-09/D-03's shared error contract requires the modal to stay
			// open with entered values intact on rejection.
			error =
				err instanceof ApiError && err.code === 'config_changed_on_disk'
					? CONFIG_CONFLICT_MESSAGE
					: err instanceof ApiError
						? err.message
						: 'Something went wrong creating the webspace — check the browser console and try again.';
		} finally {
			saving = false;
		}
	}
</script>

<Dialog {open} onOpenChange={handleOpenChange}>
	<DialogContent class="max-w-md">
		<DialogHeader>
			<DialogTitle>New webspace</DialogTitle>
		</DialogHeader>
		<form onsubmit={handleSubmit}>
			<div class="flex flex-col gap-1">
				<label for="new-webspace-name" class="text-[14px] leading-[1.4] text-foreground">
					Name
				</label>
				<Input id="new-webspace-name" bind:value={name} disabled={saving} autofocus />
				<p class="text-[14px] leading-[1.4] text-muted-foreground">
					Used in the URL and the switcher above — lowercase, no spaces recommended.
				</p>
			</div>

			{#if error}
				<Alert variant="destructive" class="mt-4">
					<AlertDescription>{error}</AlertDescription>
				</Alert>
			{/if}

			<DialogFooter>
				<Button type="button" variant="ghost" onclick={onclose}>Cancel</Button>
				<Button type="submit" disabled={saving || name.trim() === ''}>Create webspace</Button>
			</DialogFooter>
		</form>
	</DialogContent>
</Dialog>
