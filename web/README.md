# web

This is the SvelteKit SPA embedded into the topos kernel binary at build
time. It is not run or deployed standalone — its production build is
produced by the repository's `make build` target and embedded by the
kernel rather than served by a Node process.

- **`CONTRIBUTING.md`** — the dev loop, build, and test commands for
  working on this package.
- **`docs/testing.md`** — the testing map covering this package's own
  `svelte-check`, unit, and Playwright gates.
