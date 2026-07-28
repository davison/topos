<script lang="ts">
	// Renders synchronously from route state — the webspace name comes
	// straight from the URL param (config.toml key), so this header has
	// no async load state and no error state of its own (see
	// 01-UI-SPEC.md UI Considerations: nav/top-header loading + error
	// rows are dismissed for exactly this reason).
	let { webspace }: { webspace: string } = $props();
</script>

<header class="shrink-0 border-b border-border bg-card px-6 py-6">
	<!--
	  Display role (28px/600/1.2). Webspace names are user-defined
	  config.toml keys of arbitrary length, so this truncates to one line
	  with an ellipsis; the title attribute keeps the full name reachable
	  on hover regardless of how long it is.
	-->
	<h1 class="truncate text-[28px] leading-[1.2] font-semibold text-foreground" title={webspace}>
		{webspace}
	</h1>
</header>
