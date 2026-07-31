<script lang="ts">
	import { onDestroy } from 'svelte';
	import { Input } from '$lib/components/ui/input/index.js';
	import { Button } from '$lib/components/ui/button/index.js';
	import Search from '@lucide/svelte/icons/search';
	import X from '@lucide/svelte/icons/x';

	// The search box (E1, 03-UI-SPEC.md): a persistent header-tier control,
	// debounced as-you-type, never disabled and never carrying a spinner of
	// its own — the loading state lives entirely in SearchResults, which
	// reuses the stream's own skeleton rows. The caller owns `query` (the
	// committed, debounced value that drives the request) and receives
	// every debounce tick through `onquery`.
	let {
		query,
		onquery
	}: {
		query: string;
		onquery: (q: string) => void;
	} = $props();

	// Local, uncommitted input value — kept separate from the caller's
	// `query` prop so typing feels instant while the actual search request
	// (onquery) only fires 300ms after the user pauses.
	let inputValue = $state(query);

	// Re-sync the visible text whenever the caller's `query` changes from
	// outside this component — the only case that happens in practice is
	// the route's webspace-change effect resetting it to '', which must
	// clear the visible box too, not just the caller's own search state.
	// Every other `query` change is this component echoing back a value
	// it already asked for (via onquery, after its own debounce), so this
	// never fights an in-progress keystroke.
	$effect(() => {
		inputValue = query;
	});

	let debounceHandle: ReturnType<typeof setTimeout> | null = null;

	function handleInput(value: string) {
		inputValue = value;
		if (debounceHandle !== null) clearTimeout(debounceHandle);
		debounceHandle = setTimeout(() => {
			debounceHandle = null;
			onquery(value);
		}, 300);
	}

	function handleClear() {
		if (debounceHandle !== null) {
			clearTimeout(debounceHandle);
			debounceHandle = null;
		}
		inputValue = '';
		// Clearing fires immediately, with no debounce delay.
		onquery('');
	}

	onDestroy(() => {
		if (debounceHandle !== null) clearTimeout(debounceHandle);
	});
</script>

<!--
  The input is never disabled and never renders a spinner of its own, in
  any state (T-03-24's debounce/sequence-guard live in the route, not
  here). The only accent-coloured mark on this control is the standard
  focus-visible ring, inherited from the shadcn Input/Button primitives.
-->
<div class="relative w-full max-w-sm">
	<Search
		class="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground"
		aria-hidden="true"
	/>
	<Input
		type="text"
		placeholder="Search this webspace"
		value={inputValue}
		oninput={(e) => handleInput(e.currentTarget.value)}
		class="pr-9 pl-8"
	/>
	{#if inputValue}
		<Button
			type="button"
			variant="ghost"
			size="icon"
			class="absolute top-1/2 right-0 size-11 -translate-y-1/2"
			aria-label="Clear search"
			onclick={handleClear}
		>
			<X class="size-4" />
		</Button>
	{/if}
</div>
