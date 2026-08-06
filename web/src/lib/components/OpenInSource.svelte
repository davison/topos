<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import type { Link } from '$lib/api';

	// displayName parameterizes the button label (RESEARCH.md Pitfall 3,
	// generalized to source instance identity by D-08/D-09): this
	// component is shared across every source instance, so the label must
	// never hardcode one source's name or fall back to the plugin kind
	// (source_type) — the caller (DetailPane.svelte) resolves the correct
	// instance display name and passes it straight through.
	let { link, displayName }: { link: Link; displayName: string } = $props();

	// Fixed label per enum value (never populated from item/source
	// content) — fidelity is a signal, not stream-ordering input
	// (PLUG-03/ordering).
	const fidelityLabel: Record<string, string> = {
		exact: 'exact',
		anchored: 'anchored',
		'conversation-only': 'conversation-only'
	};
</script>

<div class="flex items-center gap-2">
	<Button
		href={link.url}
		target="_blank"
		rel="noopener noreferrer"
		class="min-h-11 max-w-64"
		title={`Open in ${displayName}`}
	>
		<span class="truncate">Open in {displayName}</span>
	</Button>
	<Badge variant="secondary">{fidelityLabel[link.fidelity] ?? link.fidelity}</Badge>
</div>
