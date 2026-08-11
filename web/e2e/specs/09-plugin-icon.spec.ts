// 09-01-PLAN.md Task 3's tracer proof, end to end: the mock plugin
// declares an icon via Describe (Task 1's contract fields, Task 2's
// plugins/mock embed), the kernel captures and serves it at
// GET /api/plugins/{plugin}/icon (Task 2), and the source chip renders it
// as a real, decoded <img> through PluginIcon.svelte's kernel-served path
// (Task 3) — proven in a real browser, not inferred from unit tests
// (09-01-PLAN.md's own <verification> list).
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import type { FixtureConfigSpec } from '../fixtures/config-builder';

const WEBSPACE = 'icons';

const configSpec: FixtureConfigSpec = {
	sources: [{ id: 'mock', plugin: 'topos-plugin-mock', displayName: 'Mock Source' }],
	webspaces: [{ name: WEBSPACE, keywords: ['demo'] }]
};

test.use({ configSpec });

test.describe('09-01 Task 3: the mock source chip renders a kernel-served, plugin-declared icon', () => {
	test('the chip contains a decoded <img> sourced from the kernel icon route', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock'], { logs: kernel.logs });

		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		// exact: true — the chip's own filter button, distinct from the
		// sibling "Refresh Mock Source"/"Edit Mock Source" controls and the
		// unrelated stream-row button whose accessible name is an item
		// title, all of which also contain/equal this substring.
		const chip = page.getByRole('button', { name: 'Mock Source', exact: true });
		await expect(chip).toBeVisible();

		const icon = chip.locator('img[src$="/api/plugins/topos-plugin-mock/icon"]');
		await expect(icon).toBeVisible();

		// Prove the image actually DECODED in the browser — naturalWidth is
		// 0 for a broken/never-loaded image, so this is a stronger
		// assertion than merely finding the <img> tag in the DOM.
		await expect
			.poll(async () => icon.evaluate((el: HTMLImageElement) => el.naturalWidth))
			.toBeGreaterThan(0);
	});

	test('GET /api/plugins/topos-plugin-mock/icon returns 200 with an image Content-Type, an ETag, and an immutable Cache-Control', async ({
		request,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock'], { logs: kernel.logs });

		const res = await request.get(`${kernel.baseURL}/api/plugins/topos-plugin-mock/icon`);
		expect(res.status()).toBe(200);

		const headers = res.headers();
		expect(headers['content-type'] ?? '').toContain('image/');
		expect(headers['etag'], 'expected a non-empty ETag').toBeTruthy();
		expect(headers['cache-control'] ?? '').toContain('immutable');
	});

	test('GET /api/plugins/topos-plugin-nonexistent/icon returns 404', async ({ request, kernel }) => {
		const res = await request.get(`${kernel.baseURL}/api/plugins/topos-plugin-nonexistent/icon`);
		expect(res.status()).toBe(404);
	});
});
