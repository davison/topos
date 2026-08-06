<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import type { SourceStatus } from '$lib/api';

	// D-09: All plus one chip per source INSTANCE, single-select, wrapping
	// with the header row rather than scrolling. Presentational only — the
	// caller owns the current selection and the URL round-trip (A-UI-02).
	// Keyed on source.name (the instance id, D-08), never source_type: two
	// instances of one plugin type render as two independent chips with no
	// client-side disambiguation, because load-time uniqueness (D-09)
	// already guarantees their display_names are distinct.
	let {
		sources,
		selectedSource,
		onfilter
	}: {
		sources: SourceStatus[];
		selectedSource: string | null;
		onfilter: (source: string | null) => void;
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
			variant={selectedSource === source.name ? 'default' : 'outline'}
			size="sm"
			aria-pressed={selectedSource === source.name}
			onclick={() => onfilter(source.name)}
			class="max-w-40"
			title={source.display_name}
		>
			<span class="truncate">{source.display_name}</span>
		</Button>
	{/each}
</div>
