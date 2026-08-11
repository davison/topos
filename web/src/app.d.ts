// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
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
