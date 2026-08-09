<script lang="ts">
	// The not-configured stream state (07-15-PLAN.md, closes 07-UAT.md
	// G-07-1's client half): distinct from its two siblings —
	// StreamEmpty.svelte renders when a configured webspace exists and
	// holds nothing yet, StreamError.svelte renders when the kernel could
	// not be reached at all, and this component renders when the kernel
	// gave a healthy, definitive answer that {webspace} simply is not in
	// its running configuration. Nothing is broken in that third case (the
	// kernel is up and answered correctly and promptly), so the markup
	// mirrors StreamEmpty's centred neutral treatment, never
	// StreamError's destructive Alert (planning choice 4).
	//
	// Carries no Retry control: after 07-15-PLAN.md's kernel-side fix a
	// webspace_not_found answer is definitive, not transient — retrying an
	// unconfigured name fails identically forever, so offering Retry here
	// would reproduce the exact "click Retry, hope" interaction this state
	// exists to remove (planning choice 5). The webspace switcher in the
	// header, which stays mounted above every stream state, is the
	// recovery affordance instead.
	let { webspace }: { webspace: string } = $props();
</script>

<div class="flex h-full flex-col items-center justify-center gap-2 px-6 py-12 text-center">
	<p class="text-[20px] leading-[1.2] font-semibold text-foreground">
		That webspace isn't configured
	</p>
	<p class="max-w-md text-[16px] leading-[1.5] text-muted-foreground">
		"{webspace}" isn't in your config — it may have been renamed or removed. Pick one from the
		switcher above.
	</p>
</div>
