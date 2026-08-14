<script lang="ts">
	// The vocabulary-driven match form shared by the "+" picker's one-step
	// and two-step add flows and the chip menu's "Edit match settings…"
	// modal (07-04-PLAN.md, D-11). One labelled text input per declared
	// vocabulary field, comma-separated values mapped to the config's
	// []string shape — the form structurally cannot emit an empty value
	// list (parseMatchValues drops a blank field from the emitted block
	// entirely, never as `field: []`, which the kernel's own load-time
	// validation rejects).
	//
	// `text` is seeded ONCE at mount from the initial `values` prop, not
	// reactively re-derived from it on every change — callers key this
	// component (e.g. {#key instance}) so a genuinely different
	// instance/vocabulary always mounts a fresh MatchFieldsForm. A
	// reactive re-seed would otherwise fight the user: onchange's own
	// round trip updates the parent's `values` object, and re-deriving
	// `text` from that would wipe whatever the user is mid-typing.
	import { Input } from '$lib/components/ui/input/index.js';
	import { titleCaseField, parseMatchValues } from '$lib/plugin-fields';

	let {
		vocabulary,
		values,
		onchange
	}: {
		vocabulary: string[];
		values: Record<string, string[]>;
		onchange: (block: Record<string, string[]>) => void;
	} = $props();

	let text = $state<Record<string, string>>(
		Object.fromEntries(vocabulary.map((field) => [field, (values[field] ?? []).join(', ')]))
	);

	function emit() {
		const block: Record<string, string[]> = {};
		for (const field of vocabulary) {
			const parsed = parseMatchValues(text[field] ?? '');
			if (parsed.length > 0) block[field] = parsed;
		}
		onchange(block);
	}

	function handleInput(field: string, value: string) {
		text[field] = value;
		emit();
	}
</script>

{#if vocabulary.length > 0}
	<div class="flex flex-col gap-4">
		{#each vocabulary as field (field)}
			<div class="flex flex-col gap-1">
				<label for={`match-field-${field}`} class="text-[14px] leading-[1.4] text-foreground">
					{titleCaseField(field)}
				</label>
				<Input
					id={`match-field-${field}`}
					value={text[field] ?? ''}
					oninput={(e) => handleInput(field, e.currentTarget.value)}
				/>
				<p class="text-[14px] leading-[1.4] text-muted-foreground">
					Comma-separated. Matches if any value is present. Values are matched exactly,
					case-insensitively — wildcards and glob patterns are not supported.
				</p>
			</div>
		{/each}
	</div>
{/if}
