import { sveltekit } from '@sveltejs/kit/vite';
import tailwindcss from '@tailwindcss/vite';
import { defineConfig } from 'vite';

export default defineConfig({
	// sveltekit() is called with no arguments so SvelteKit loads its
	// configuration from svelte.config.js (the adapter-static setup) rather
	// than from options passed here — passing options here would silently
	// ignore svelte.config.js.
	plugins: [tailwindcss(), sveltekit()],
	server: {
		// `make dev` runs the kernel separately on 127.0.0.1:7777 and the
		// SvelteKit dev server proxies /api to it, so the SPA is never
		// embedded during development.
		proxy: {
			'/api': {
				target: 'http://127.0.0.1:7777',
				changeOrigin: true
			}
		}
	}
});
