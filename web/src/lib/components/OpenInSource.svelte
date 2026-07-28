<script lang="ts">
	import { Button } from '$lib/components/ui/button/index.js';
	import { Badge } from '$lib/components/ui/badge/index.js';
	import { sourceDisplayName, type Link } from '$lib/api';

	// sourceType parameterizes the button label (RESEARCH.md Pitfall 3):
	// this component is shared across every source, so the label must
	// never hardcode one source's name.
	let { link, sourceType }: { link: Link; sourceType: string } = $props();

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
	<Button href={link.url} target="_blank" rel="noopener noreferrer" class="min-h-11">
		Open in {sourceDisplayName(sourceType)}
	</Button>
	<Badge variant="secondary">{fidelityLabel[link.fidelity] ?? link.fidelity}</Badge>
</div>
