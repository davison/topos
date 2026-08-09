// The tracer slice (07.1-01-PLAN.md Task 2): one path through every layer
// this phase touches — Makefile build -> generated temp config -> spawned
// kernel binary -> embedded SPA -> Chromium -> stream assertion. Proves
// the whole harness architecture on one thin path before nine spec files
// are written on top of it.
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import type { FixtureConfigSpec } from '../fixtures/config-builder';

const configSpec: FixtureConfigSpec = {
	sources: [{ id: 'mock', plugin: 'topos-plugin-mock' }],
	webspaces: [{ name: 'tracer', keywords: ['demo'] }]
};

test.use({ configSpec });

test.describe('tracer: a hermetic kernel serves seeded mock items to a real browser', () => {
	test('the mock corpus renders in the tracer webspace stream', async ({ page, kernel }) => {
		await waitForFirstSync(kernel.baseURL, ['mock'], { logs: kernel.logs });

		await page.goto(`${kernel.baseURL}/w/tracer`);

		// The webspace switcher trigger renders the webspace name (D-10) —
		// this is the SPA's replacement for a static <h1> title.
		await expect(page.getByRole('button', { name: 'tracer' })).toBeVisible();

		// The mock plugin's fixed four-item corpus (plugins/mock/plugin.go)
		// all carry the "demo" label, so every one of them lands in this
		// webspace via the keywords fallback.
		await expect(page.getByText('Welcome to the mock source')).toBeVisible();
		await expect(page.getByText('Standup: Tuesday')).toBeVisible();
		await expect(page.getByText('Standup: Wednesday')).toBeVisible();
		await expect(page.getByText('Shopping list')).toBeVisible();
	});
});
