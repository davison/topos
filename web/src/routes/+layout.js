// Phase 1 is a pure SPA embedded in the Go kernel binary: no SvelteKit
// server ever runs against this build, so server-side rendering and
// prerendering are both meaningless (and prerendering would try to crawl
// routes against a kernel API that isn't running at build time).
export const ssr = false;
export const prerender = false;
