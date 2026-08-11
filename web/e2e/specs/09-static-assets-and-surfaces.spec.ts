// 09-03-PLAN.md Task 3: browser proof for the plan's three static-asset/
// palette fixes, against a real built-and-embedded kernel — per the standing
// Phase 07.1 rule (docs/testing.md "## Standing rule") that any UAT item a
// browser can drive becomes a permanent spec, not a manual check.
//
// Covers, as three separate test cases:
//   1. Favicon (09-UI-SPEC.md Fix 1) — the served document's resolved icon
//      link points at /app-icon.png, and that path fetches 200/image-png.
//      Asserted against the client-rendered document (page.goto), not the
//      SvelteKit source: this app builds with ssr=false/prerender=false
//      (locked Phase 1 decision, web/src/routes/+layout.js), so the static
//      HTML shell the kernel serves before any JS executes carries no
//      <link rel="icon"> at all — <svelte:head> only populates the real DOM
//      once the client bundle runs. A real browser is the only way to prove
//      the tab icon a user actually sees, which is exactly why this belongs
//      in this suite rather than a static build-output grep.
//   2. robots.txt (Fix 6) — a direct GET returns 200 with a full disallow,
//      proving the embedded twin (kernel/webui/build/robots.txt, go:embed'd
//      from web/static/ at build time) tracks the source tree.
//   3. Menu surface distinctness (Fix 8) — the webspace switcher's open
//      dropdown content renders on a computed background-color distinct
//      from the header pane behind it, and its own computed border-color
//      differs from its own background — the two properties Fix 8 exists to
//      make simultaneously true. Compared via getComputedStyle, never class
//      strings, so this catches a token that is declared but not actually
//      applied.
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import type { FixtureConfigSpec } from '../fixtures/config-builder';

const WEBSPACE = 'surfaces';

const configSpec: FixtureConfigSpec = {
	sources: [{ id: 'mock', plugin: 'topos-plugin-mock' }],
	webspaces: [{ name: WEBSPACE, keywords: ['demo'] }]
};

test.use({ configSpec });

test.describe('09-03: favicon, robots.txt, and floating-menu surface distinctness', () => {
	test('the served document resolves its icon link to /app-icon.png, which fetches 200 image/png', async ({
		page,
		request,
		kernel
	}) => {
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const iconHref = await page.locator('link[rel="icon"]').getAttribute('href');
		expect(iconHref, 'expected the document to carry a <link rel="icon"> once client JS has run').not.toBeNull();
		// The browser may reflect the href attribute back resolved to an
		// absolute URL (observed in Chromium) rather than the literal
		// relative string authored in +layout.svelte — resolve both sides
		// against the kernel's own baseURL and compare pathnames, so this
		// assertion holds regardless of which form the DOM reports.
		const resolvedPathname = new URL(iconHref!, kernel.baseURL).pathname;
		expect(resolvedPathname).toBe('/app-icon.png');

		const iconType = await page.locator('link[rel="icon"]').getAttribute('type');
		expect(iconType).toBe('image/png');

		const iconResponse = await request.get(`${kernel.baseURL}/app-icon.png`);
		expect(iconResponse.status()).toBe(200);
		expect(iconResponse.headers()['content-type'] ?? '').toContain('image/png');
	});

	test('GET /robots.txt returns 200 with a full disallow', async ({ request, kernel }) => {
		const res = await request.get(`${kernel.baseURL}/robots.txt`);
		expect(res.status()).toBe(200);
		const body = await res.text();
		expect(body).toContain('User-agent: *');
		expect(body).toContain('Disallow: /');
	});

	test('an open dropdown menu renders on a surface distinct from the pane behind it, with its own border still legible', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock'], { logs: kernel.logs });

		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const headerPane = page.locator('header');
		await expect(headerPane).toBeVisible();
		const headerBg = await headerPane.evaluate((el) => getComputedStyle(el).backgroundColor);

		// The webspace switcher's trigger reads as the current webspace name
		// (WebspaceSwitcher.svelte) — opening it renders the dropdown over
		// the header pane it lives inside.
		await page.getByRole('button', { name: WEBSPACE, exact: true }).click();

		const menu = page.locator('[data-slot="dropdown-menu-content"]');
		await expect(menu).toBeVisible();

		const menuBg = await menu.evaluate((el) => getComputedStyle(el).backgroundColor);
		const menuBorder = await menu.evaluate((el) => getComputedStyle(el).borderColor);

		expect(
			menuBg,
			`expected the open menu's computed background-color (${menuBg}) to differ from the header pane's (${headerBg}) — the menu must not be camouflaged against the pane it opens over`
		).not.toBe(headerBg);
		expect(
			menuBorder,
			`expected the open menu's own computed border-color (${menuBorder}) to differ from its own computed background-color (${menuBg}) — otherwise the menu's frame vanishes into its own fill`
		).not.toBe(menuBg);
	});
});
