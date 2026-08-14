<script lang="ts">
	import '../app.css';
	import { page } from '$app/state';
	import { browser } from '$app/environment';
	import { Toaster } from '$lib/components/ui/sonner/index.js';
	import { pwaUpdatedToast } from '$lib/toast';
	import { scheduleUpdateChecks } from '$lib/pwa-update';

	let { children } = $props();

	// Only the /w/[webspace] route has a `webspace` param — the landing
	// page ("/") renders its own heading and has no use for the fixed
	// h-screen two-pane wrapper below.
	//
	// WebspaceHeader itself moved from here into +page.svelte in Phase 2
	// (02-03-PLAN.md): its new props (sources, sourcesState, the filter
	// selection, the refresh handlers) are all owned by the page's own
	// per-webspace state, and a layout can't receive props back from the
	// page it renders via {@render children()} — the header has to be
	// rendered by whichever component actually owns that state.
	let webspace = $derived(page.params.webspace);

	// 13-04-PLAN.md Task 2 (UI-13/UI-SPEC E8): PWA update registration —
	// the ONLY place `virtual:pwa-register` is imported. Guarded behind
	// `browser` so a non-browser evaluation (svelte-check, vitest) is a
	// no-op; the virtual module resolves only inside a real Vite build.
	//
	// registerType: 'autoUpdate' (web/vite.config.ts, PD-08) means
	// vite-plugin-pwa's own registerSW core reloads the page itself with NO
	// user action the moment a new ServiceWorker activates — UNLESS an
	// `onNeedReload` callback is supplied, in which case the callback
	// becomes wholly responsible for reloading (verified against
	// node_modules/vite-plugin-pwa/dist/client/build/svelte.js this
	// session). This app supplies `onNeedReload` to set a one-shot
	// sessionStorage flag immediately before performing the exact same
	// reload the library would have done unprompted — the flag is what
	// lets the NEXT page load (the one immediately after the reload) know
	// to fire pwaUpdatedToast(), since a toast fired here would never be
	// seen: the reload it precedes destroys this document before anything
	// renders.
	const PWA_UPDATED_FLAG = 'topos-pwa-updated';

	if (browser) {
		if (sessionStorage.getItem(PWA_UPDATED_FLAG)) {
			sessionStorage.removeItem(PWA_UPDATED_FLAG);
			pwaUpdatedToast();
		}

		import('virtual:pwa-register')
			.then(({ registerSW }) => {
				registerSW({
					// Register immediately rather than deferring to the window
					// `load` event (workbox-window's own default when
					// `immediate` is unset) — an installed desktop app wants an
					// active ServiceWorker as soon as possible, both for the
					// "kernel not running still renders the shell" guarantee
					// (13-UI-SPEC.md E8) and so the hermetic e2e spec proving
					// registration has no window-load timing to race against.
					immediate: true,
					onNeedReload() {
						sessionStorage.setItem(PWA_UPDATED_FLAG, '1');
						window.location.reload();
					},
					// Checkpoint fix (13-04-PLAN.md Task 4, defect 1): without
					// this, a standalone installed window that never navigates
					// never triggers the browser's own SW update check, so
					// onNeedReload above never had anything to react to —
					// registration.update() (a plain SW-script refetch/compare,
					// never an /api call) is what actually asks the question;
					// this only widens WHEN it gets asked.
					onRegisteredSW(_swScriptUrl, registration) {
						if (!registration) return;
						try {
							scheduleUpdateChecks(registration, {
								windowTarget: window,
								documentTarget: document,
								// Bound to window: a bare `{ setInterval, clearInterval }`
								// detaches these from their required `this` (native
								// timer methods throw "Illegal invocation" when called
								// with any other receiver) — verified via a real
								// browser this session, not assumed.
								timers: {
									setInterval: window.setInterval.bind(window),
									clearInterval: window.clearInterval.bind(window)
								}
							});
						} catch (err) {
							console.error('failed to schedule PWA update checks', err);
						}
					}
				});
			})
			.catch(() => {
				// `virtual:pwa-register` always resolves (a no-op stub under
				// `vite dev`, the real registration module in a production
				// build) — this catch is a defensive backstop only, not an
				// expected path; registration-specific failures are the
				// registerSW() call's own concern (onRegisterError), not this
				// dynamic import's.
			});
	}
</script>

<svelte:head>
	<link rel="icon" type="image/png" href="/app-icon.png" />
</svelte:head>

<!-- Sonner mount (13-UI-SPEC.md E3): the app's first toast primitive,
     mounted exactly ONCE here at the root layout, regardless of route —
     the undo/failure toast (Task 3) and the PWA update notice (a later
     plan) both fire through this single mount. -->
<Toaster />

{#if webspace}
	<div class="flex h-screen flex-col">
		<div class="min-h-0 flex-1">
			{@render children()}
		</div>
	</div>
{:else}
	{@render children()}
{/if}
