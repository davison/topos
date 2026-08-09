// This spec is the hermetic successor to `scripts/e2e-smoke.sh`, retired
// by D-10 once the guards below cover its three portable assertions
// end-to-end against a real (mock-shaped, credential-free) kernel instead
// of a live paperless-ngx one:
//   1. the served SPA links a real stylesheet that fetches non-empty and
//      contains the #020617 theme token (the G-01-2 recurrence guard);
//   2. every stream item carries a well-formed `link` shape;
//   3. an unknown webspace answers a typed 404 envelope
//      (`error.code: "webspace_not_found"`).
// Plus one new, cheap guard (4, below) that every other spec's page.goto
// silently depends on: the embedded SPA's client-side routing fallback.
//
// `scripts/e2e-smoke.sh` itself is untouched by this plan — its removal
// belongs to plan 07.1-06, once this file is proven green.
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import type { FixtureConfigSpec } from '../fixtures/config-builder';

const WEBSPACE = 'served';

const configSpec: FixtureConfigSpec = {
	sources: [{ id: 'mock', plugin: 'topos-plugin-mock' }],
	webspaces: [{ name: WEBSPACE, keywords: ['demo'] }]
};

test.use({ configSpec });

// The theme token G-01-2 guards — scripts/assert-stylesheet.sh's own first
// substantive check, carried over verbatim.
const THEME_TOKEN = '#020617';

const KNOWN_FIDELITIES = ['exact', 'anchored', 'conversation-only'];

test.describe('smoke: served SPA — hermetic successor to scripts/e2e-smoke.sh', () => {
	test('the served SPA links a real, theme-token-bearing stylesheet', async ({ request, kernel }) => {
		const htmlResponse = await request.get(`${kernel.baseURL}/`);
		expect(htmlResponse.status(), 'GET / did not return 200').toBe(200);
		const html = await htmlResponse.text();

		// adapter-static emits the stylesheet <link> with either attribute
		// order — match both, exactly like e2e-smoke.sh's own grep did.
		const match =
			html.match(/<link[^>]*rel="stylesheet"[^>]*href="([^"]+\.css)"/) ??
			html.match(/<link[^>]*href="([^"]+\.css)"[^>]*rel="stylesheet"/);
		expect(
			match,
			'served SPA HTML links no stylesheet — the production build may have shipped zero CSS (G-01-2)'
		).not.toBeNull();
		const href = match![1];

		// adapter-static emits relative asset hrefs (./_app/...); normalise
		// to an absolute path before prefixing baseURL, exactly like
		// e2e-smoke.sh's own CSS_PATH normalisation.
		let cssPath = href.replace(/^\./, '');
		if (!cssPath.startsWith('/')) cssPath = `/${cssPath}`;

		const cssResponse = await request.get(`${kernel.baseURL}${cssPath}`);
		expect(cssResponse.status(), `stylesheet linked at ${href} did not fetch successfully`).toBe(200);
		const css = await cssResponse.text();
		expect(css.length, 'served stylesheet is empty').toBeGreaterThan(0);
		expect(css, `served stylesheet does not contain the ${THEME_TOKEN} design token (G-01-2)`).toContain(
			THEME_TOKEN
		);
	});

	test('every stream item has a well-formed link shape', async ({ request, kernel }) => {
		await waitForFirstSync(kernel.baseURL, ['mock'], { logs: kernel.logs });

		const res = await request.get(`${kernel.baseURL}/api/webspaces/${WEBSPACE}/stream`);
		expect(res.status()).toBe(200);
		const body = (await res.json()) as {
			items: Array<{ link: { url: string; fidelity: string } }>;
		};

		expect(body.items.length).toBeGreaterThan(0);
		for (const item of body.items) {
			expect(item.link.url.startsWith('http://localhost/')).toBe(true);
			// Deliberately NOT asserting every item is "exact" — that was
			// true only of the old paperless-only smoke corpus; the mock
			// corpus (plugins/mock/plugin.go) genuinely carries all three
			// declared fidelity values, and asserting a single one here
			// would encode a false invariant.
			expect(KNOWN_FIDELITIES).toContain(item.link.fidelity);
		}
	});

	test('an unknown webspace answers a typed 404 envelope', async ({ request, kernel }) => {
		const res = await request.get(`${kernel.baseURL}/api/webspaces/does-not-exist/stream`);
		expect(res.status()).toBe(404);
		const body = (await res.json()) as { error: { code: string } };
		expect(body.error.code).toBe('webspace_not_found');
	});

	test('a deep client-side route returns 200 HTML — the embedded SPA fallback is wired', async ({
		request,
		kernel
	}) => {
		const res = await request.get(`${kernel.baseURL}/w/${WEBSPACE}`);
		expect(res.status()).toBe(200);
		const contentType = res.headers()['content-type'] ?? '';
		expect(contentType).toContain('text/html');
	});
});
