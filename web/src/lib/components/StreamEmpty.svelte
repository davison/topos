<script lang="ts">
	// The approved empty state (01-UI-SPEC.md Copywriting Contract):
	// renders only when the stream response resolved successfully, the
	// sync did not fail, and no items matched the webspace's keywords.
	// This is deliberately distinct from — and checked strictly after —
	// the sync-failure state in StreamList.svelte, so a failed sync can
	// never be mistaken for a genuinely empty topic.
</script>

<div class="flex h-full flex-col items-center justify-center gap-2 px-6 py-12 text-center">
	<p class="text-[20px] leading-[1.2] font-semibold text-foreground">Nothing here yet</p>
	<p class="max-w-md text-[16px] leading-[1.5] text-muted-foreground">
		No paperless-ngx documents match this webspace's keywords yet. Check your tags, or wait for the next sync.
	</p>
</div>
