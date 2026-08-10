<script lang="ts">
	import { Alert, AlertTitle, AlertDescription, AlertAction } from '$lib/components/ui/alert/index.js';
	import { Button } from '$lib/components/ui/button/index.js';

	// The degraded stream state (08-UI-SPEC.md Amendment 3, closes
	// 08-UAT.md G-08-3): renders when the kernel answered 200 but the
	// webspace's participating-source sync aggregate is "error" and the
	// stream returned zero items — one source failed to sync, not the
	// kernel itself.
	//
	// Distinguished explicitly from each sibling stream state:
	//   - StreamEmpty: a healthy webspace holding nothing yet — no sync
	//     failure at all.
	//   - StreamMissing: the kernel gave a healthy, definitive answer that
	//     the webspace simply is not configured — nothing failed to sync
	//     because nothing is trying to.
	//   - StreamError: the kernel could not be reached at all (the stream
	//     fetch itself failed). This is this component's ONE remaining
	//     cause after this split — the kernel answered 200 here, and
	//     claiming otherwise sent a user to restart a service that was
	//     running fine, which is exactly what made 08-UAT.md's G-08-3
	//     report a major-severity defect rather than a cosmetic one.
	//
	// The recorded sync error is rendered verbatim (the kernel's own
	// sync_runs text, A-PLUG-04 — never a plugin's self-reported health).
	// The body copy deliberately points at the source chips above rather
	// than re-deriving per-source health here: that surface already
	// carries each source's own tone, last sync time and last_error
	// (UI-05/UI-06), so this state's job is to explain why the stream is
	// empty and hand the user to the surface that already diagnoses it —
	// not to grow a second per-source health surface.
	let { onretry, syncError }: { onretry: () => void; syncError: string } = $props();
</script>

<div class="flex h-full flex-col items-center justify-center px-6 py-12">
	<Alert variant="default" class="max-w-md">
		<AlertTitle>A source couldn't sync</AlertTitle>
		<AlertDescription>
			<p>
				Nothing to show here yet. Your other sources are unaffected — check the source chips
				above, then retry.
			</p>
			{#if syncError}
				<p class="mt-2 text-[14px] leading-[1.4]">{syncError}</p>
			{/if}
		</AlertDescription>
		<AlertAction>
			<Button variant="outline" size="sm" onclick={onretry}>Retry</Button>
		</AlertAction>
	</Alert>
</div>
