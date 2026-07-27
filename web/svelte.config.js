import adapter from '@sveltejs/adapter-static';
import { vitePreprocess } from '@sveltejs/vite-plugin-svelte';

/** @type {import('@sveltejs/kit').Config} */
const config = {
	preprocess: vitePreprocess(),

	kit: {
		// Static SPA build embedded into the Go kernel binary via
		// kernel/webui/embed.go's `//go:embed all:build`. The build output
		// lands inside the Go module so `go build` sees it without any
		// extra copy step.
		//
		// fallback is deliberately NOT the more intuitive default name
		// (a file named "index" + the html extension) — that name
		// collides with prerendered output per current SvelteKit /
		// adapter-static guidance. `200.html` is the fallback page that
		// makes client-side routes (e.g. /w/house-move) survive a direct
		// browser reload once served by the kernel's catch-all handler.
		adapter: adapter({
			pages: '../kernel/webui/build',
			assets: '../kernel/webui/build',
			fallback: '200.html',
			precompress: false,
			strict: true
		})
	}
};

export default config;
