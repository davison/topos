<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import type { SourceStatus } from '$lib/api';

	// D-09: All plus one chip per source, single-select, wrapping with the
	// header row rather than scrolling. Presentational only — the caller
	// owns the current selection and the URL round-trip (A-UI-02).
	let {
		sources,
		selectedSource,
		onfilter
	}: {
		sources: SourceStatus[];
		selectedSource: string | null;
		onfilter: (sourceType: string | null) => void;
	} = $props();
</script>

<div class="flex flex-wrap items-center gap-2">
	<Button
		variant={selectedSource === null ? 'default' : 'outline'}
		size="sm"
		aria-pressed={selectedSource === null}
		onclick={() => onfilter(null)}
	>
		All
	</Button>
	{#each sources as source (source.name)}
		<Button
			variant={selectedSource === source.source_type ? 'default' : 'outline'}
			size="sm"
			aria-pressed={selectedSource === source.source_type}
			onclick={() => onfilter(source.source_type)}
		>
			{source.display_name}
		</Button>
	{/each}
</div>
