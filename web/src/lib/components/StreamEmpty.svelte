<script lang="ts">
	// The approved empty state (02-UI-SPEC.md Copywriting Contract):
	// renders only when the stream response resolved successfully, the
	// sync did not fail, and no items matched the webspace's keywords —
	// deliberately distinct from, and checked strictly after, the
	// sync-failure state in StreamList.svelte, so a failed sync can never
	// be mistaken for a genuinely empty topic.
	//
	// When `displayName` is supplied, this renders the filtered-empty
	// variant instead (D-09): distinct copy that points the user back to
	// "All" rather than implying the whole webspace is empty — the
	// difference between "you are looking at a slice" and "there is
	// nothing here".
	let { displayName = null }: { displayName?: string | null } = $props();
</script>

<div class="flex h-full flex-col items-center justify-center gap-2 px-6 py-12 text-center">
	{#if displayName}
		<p class="text-[20px] leading-[1.2] font-semibold text-foreground">No {displayName} items yet</p>
		<p class="max-w-md text-[16px] leading-[1.5] text-muted-foreground">
			No {displayName} items match this webspace's keywords yet. Switch back to All, or check your
			tags.
		</p>
	{:else}
		<p class="text-[20px] leading-[1.2] font-semibold text-foreground">Nothing here yet</p>
		<p class="max-w-md text-[16px] leading-[1.5] text-muted-foreground">
			No items match this webspace's keywords yet. Check your tags, or wait for the next sync.
		</p>
	{/if}
</div>
