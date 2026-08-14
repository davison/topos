// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces

// 13-04-PLAN.md Task 2 (UI-13): ambient module declarations for
// `virtual:pwa-register` (and the other framework variants, unused here) —
// vite-plugin-pwa's own recommended reference for a project that imports
// the virtual module directly rather than only via SvelteKitPWA's build-
// time injection.
/// <reference types="vite-plugin-pwa/client" />

declare global {
	namespace App {
		// interface Error {}
		// interface Locals {}
		// interface PageData {}
		interface PageState {
			// Set via `pushState('', { itemOpen: true })` when the narrow-
			// viewport mobile takeover opens (09.1-01-PLAN.md D-03/D-04).
			itemOpen?: boolean;
		}
		// interface Platform {}
	}
}

export {};
