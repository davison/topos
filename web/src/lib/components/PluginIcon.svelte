<script lang="ts">
	import Puzzle from '@lucide/svelte/icons/puzzle';

	// Renders a plugin's own declared identity icon (09-01-PLAN.md Task 3,
	// 09-UI-SPEC.md Fix 10) as a plain <img> sourced from the kernel's
	// per-plugin-binary icon route (see the src below) — the one new
	// frontend icon import this phase adds (Puzzle), and the only
	// rendering path for every plugin icon, real logo or embedded
	// Lucide-derived glyph alike.
	//
	// Mandatory three-step fallback chain — this component's whole
	// contract. Every terminus is either the real icon or Puzzle; an empty
	// box or a broken-image glyph must be unreachable:
	//   1. an empty/unknown plugin binary name -> Puzzle
	//   2. a kernel 404 (undescribed plugin type, or a pre-Phase-9 binary
	//      that returned empty icon bytes) -> the <img>'s own onerror
	//      fires -> Puzzle
	//   3. malformed bytes / a network hiccup -> onerror fires -> Puzzle
	//
	// size is a caller-supplied Tailwind size class (size-3.5 in the
	// source chip, size-4 in the add-source picker and Manage Sources
	// rows) fixing the rendered box's dimensions before the bytes ever
	// load, so no layout shift occurs regardless of which plugin owns the
	// icon. Colour is baked into the served bytes (09-UI-SPEC.md's "baked
	// color, not currentColor" decision) — only the Puzzle fallback, a
	// live Lucide component, carries text-muted-foreground.
	let { plugin, size }: { plugin: string; size: string } = $props();

	// broken flips true on the <img>'s own onerror (404, malformed bytes,
	// a network hiccup) — reset whenever plugin itself changes, so
	// navigating from a broken row to a different plugin's row never
	// inherits the prior row's failure state.
	let broken = $state(false);
	$effect(() => {
		plugin;
		broken = false;
	});

	let showImage = $derived(Boolean(plugin) && !broken);
</script>

{#if showImage}
	<img
		src={`/api/plugins/${plugin}/icon`}
		alt=""
		class={`${size} object-contain`}
		onerror={() => (broken = true)}
	/>
{:else}
	<Puzzle class={`${size} text-muted-foreground`} />
{/if}
