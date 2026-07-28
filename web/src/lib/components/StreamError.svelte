<script lang="ts">
	import { Alert, AlertTitle, AlertDescription, AlertAction } from '$lib/components/ui/alert/index.js';
	import { Button } from '$lib/components/ui/button/index.js';

	// The approved stream-load error state (01-UI-SPEC.md Copywriting
	// Contract). Rendered for two distinct causes, both mapped to the
	// same copy and control:
	//   1. The stream fetch itself failed (kernel unreachable).
	//   2. The fetch succeeded but the recorded sync run failed and
	//      returned zero items — the sync-failure branch in
	//      StreamList.svelte, which must never fall through to the
	//      empty state instead. When triggered this way, the caller
	//      passes `syncError` so the user can see what actually failed;
	//      this is deliberately the only sync detail Phase 1 surfaces.
	let { onretry, syncError = '' }: { onretry: () => void; syncError?: string } = $props();
</script>

<div class="flex h-full flex-col items-center justify-center px-6 py-12">
	<Alert variant="destructive" class="max-w-md">
		<AlertTitle>Couldn't load this webspace</AlertTitle>
		<AlertDescription>
			<p>The webspaces service didn't respond — check that it's running, then retry.</p>
			{#if syncError}
				<p class="mt-2 text-[14px] leading-[1.4]">{syncError}</p>
			{/if}
		</AlertDescription>
		<AlertAction>
			<Button variant="outline" size="sm" onclick={onretry}>Retry</Button>
		</AlertAction>
	</Alert>
</div>
