<script lang="ts">
	import { Alert, AlertTitle, AlertDescription, AlertAction } from '$lib/components/ui/alert/index.js';
	import { Button } from '$lib/components/ui/button/index.js';

	// The approved stream-load error state (01-UI-SPEC.md Copywriting
	// Contract, narrowed by 08-UI-SPEC.md Amendment 3 / 08-UAT.md G-08-3).
	// Now has exactly ONE cause: the stream fetch itself failed and the
	// kernel is genuinely unreachable. It previously also rendered for a
	// second, distinct cause — a per-source sync failure with zero items
	// — via a `syncError` prop; that cause now has its own honest state,
	// StreamSyncDegraded.svelte, which names the failing source instead
	// of this component's fixed copy falsely claiming the whole service
	// is down. See StreamSyncDegraded.svelte's own doc comment for the
	// full distinction between every stream state.
	let { onretry }: { onretry: () => void } = $props();
</script>

<div class="flex h-full flex-col items-center justify-center px-6 py-12">
	<Alert variant="destructive" class="max-w-md">
		<AlertTitle>Couldn't load this webspace</AlertTitle>
		<AlertDescription>
			<p>The topos service didn't respond — check that it's running, then retry.</p>
		</AlertDescription>
		<AlertAction>
			<Button variant="outline" size="sm" onclick={onretry}>Retry</Button>
		</AlertAction>
	</Alert>
</div>
