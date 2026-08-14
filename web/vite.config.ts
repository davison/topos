import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vitest/config';
import { SvelteKitPWA } from '@vite-pwa/sveltekit';

export default defineConfig({
	// sveltekit() is called with no arguments so SvelteKit loads its
	// configuration from svelte.config.js (the adapter-static setup) rather
	// than from options passed here — passing options here would silently
	// ignore svelte.config.js.
	plugins: [
		tailwindcss(),
		sveltekit(),
		// 13-04-PLAN.md Task 1 (UI-13): PWA installability. kit.adapterFallback
		// + kit.spa are the load-bearing settings for this repo specifically —
		// adapter-static writes 200.html (this repo's deliberately non-default
		// SPA fallback name, see svelte.config.js) AFTER this plugin runs, so
		// without them the fallback page has no precache revision and a
		// deep-link reload can serve stale markup even after the JS bundle
		// updates (13-RESEARCH.md Pitfall 3). `spa: true` makes the plugin
		// derive that revision from `.svelte-kit/output/client/_app/
		// version.json` instead, which changes on every kernel rebuild.
		SvelteKitPWA({
			strategies: 'generateSW',
			registerType: 'autoUpdate',
			kit: {
				adapterFallback: '200.html',
				spa: true
			},
			manifest: {
				name: 'topos',
				short_name: 'topos',
				description: 'Bringing all your topics to one place.',
				display: 'standalone',
				start_url: '/',
				// Both colours are this app's own --background token
				// (web/src/app.css) — the launch splash must never flash an
				// unstyled light background (13-UI-SPEC.md E7).
				theme_color: '#020617',
				background_color: '#020617'
			},
			// Icons are generated (never hand-authored) from the existing
			// web/static/app-icon.png via pwa-assets.config.ts, so the
			// manifest's icons array reflects the generated set rather than a
			// duplicated, driftable literal.
			pwaAssets: {
				config: true
			},
			workbox: {
				// Precache the app shell only — HTML/JS/CSS/icons/manifest.
				// Deliberately no runtimeCaching entry anywhere in this object:
				// an entry matching the /api/ prefix would silently reintroduce
				// the offline API caching this milestone's Requirements
				// document rules out (13-RESEARCH.md Pitfall 5).
				globPatterns: ['**/*.{js,css,html,ico,png,svg,webp,webmanifest}'],
				navigateFallback: '/200.html',
				navigateFallbackDenylist: [/^\/api\//]
			}
		})
	],
	server: {
		// `make dev` runs the kernel separately on 127.0.0.1:7777 and the
		// SvelteKit dev server proxies /api to it, so the SPA is never
		// embedded during development.
		proxy: {
			'/api': {
				target: 'http://127.0.0.1:7777',
				changeOrigin: true
			}
		},
		// `make dev` passes --host so the dev server is reachable over the
		// tailscale network; the leading dot allowlists any MagicDNS name
		// (e.g. hepburn-ts.<tailnet>.ts.net). IP-literal access is always
		// permitted by Vite and needs no entry here.
		allowedHosts: ['.ts.net']
	},
	test: {
		// Plan 01-03 only unit-tests plain TS (web/src/lib/format.ts) — no
		// Svelte component test harness (jsdom, @testing-library/svelte) is
		// needed yet, so the environment stays 'node' until a future plan
		// actually needs to mount a component.
		environment: 'node',
		include: ['src/**/*.test.ts']
	}
});
