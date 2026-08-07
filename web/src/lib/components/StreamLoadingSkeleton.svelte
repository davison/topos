<script lang="ts">
	import { Skeleton } from '$lib/components/ui/skeleton/index.js';

	// WR-04: shared loading-skeleton surface for StreamList.svelte and
	// SearchResults.svelte — both render four skeleton rows at the real row
	// dimensions (.stream-row-surface, app.css) so the list doesn't reflow
	// when data/results arrive. Extracted so a future change to the skeleton
	// count/dimensions is made in exactly one place, mirroring the
	// `.search-highlight` shared-surface pattern (G-06-1).
</script>

<div class="flex flex-col gap-3">
	{#each Array(4) as _, i (i)}
		<Skeleton class="stream-row-surface w-full rounded-lg" />
	{/each}
</div>
