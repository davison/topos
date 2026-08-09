// Ports 07-UAT.md item 4 into a permanent regression gate.
//
// Verbatim pass condition this spec encodes (07-UAT.md item 4):
//   "With zero [webspaces.*] blocks, / shows "No webspaces yet" with a
//   working Create webspace CTA. Stop the kernel and reload: the
//   service-unreachable copy renders only for the real outage."
import { test, expect } from '../fixtures/kernel';
import { mockInstances, type FixtureConfigSpec } from '../fixtures/config-builder';
import { readConfigToml } from '../fixtures/toml';

const configSpec: FixtureConfigSpec = {
	sources: mockInstances(1),
	webspaces: []
};

test.use({ configSpec });

// Every test in this file shares the ONE worker-scoped kernel `configSpec`
// above produces (Playwright forbids overriding a worker-scoped option
// from a nested describe — see harness-hermeticity.spec.ts's own doc
// comment for the full rationale). Serial mode pins execution to
// declaration order, so the LAST test's kernel.stop() call safely ends the
// shared kernel only once every earlier test no longer needs it.
test.describe('07-UAT item 4: zero-webspace empty state vs a genuine outage', () => {
	test.describe.configure({ mode: 'serial' });

	test('zero configured webspaces renders the empty state, and its Create webspace CTA is genuinely wired', async ({
		page,
		kernel
	}) => {
		await page.goto(`${kernel.baseURL}/`);

		await expect(page.getByRole('heading', { name: 'No webspaces yet' })).toBeVisible();

		await page.getByRole('button', { name: 'Create webspace' }).click();
		await expect(page.getByRole('heading', { name: 'New webspace' })).toBeVisible();

		await page.getByLabel('Name').fill('first-webspace');
		await page.getByRole('dialog').getByRole('button', { name: 'Create webspace' }).click();

		// Navigates to the new webspace — proving the CTA is genuinely
		// wired, not decorative.
		await expect(page).toHaveURL(/\/w\/first-webspace$/);

		const doc = readConfigToml(kernel.configPath);
		expect(doc.webspaces).toHaveProperty('first-webspace');
	});

	test('a failed config fetch renders the outage copy, and never the zero-webspace empty state', async ({
		page,
		kernel
	}) => {
		// Do NOT stop the kernel and reload here: the kernel also serves the
		// SPA itself, so a stopped kernel cannot serve the page that would
		// render the outage copy at all — the browser would show its own
		// connection-error page instead, which asserts nothing about this
		// app's own error handling. Instead, fail only the config request
		// BEFORE navigating, so the SPA loads normally from the (still
		// running) kernel and its OWN GET /api/config fetch fails — exactly
		// the code path web/src/routes/+page.svelte's outage copy exists
		// for. Registered before navigation, per this harness's own
		// "register before the triggering action" discipline.
		await page.route('**/api/config', (route) => route.abort());

		await page.goto(`${kernel.baseURL}/`);

		await expect(
			page.getByText("Couldn't load this webspace — the topos service didn't respond.")
		).toBeVisible();
		await expect(page.getByRole('heading', { name: 'No webspaces yet' })).toHaveCount(0);

		await page.unroute('**/api/config');
	});

	test('with the kernel stopped entirely, a direct request to the base URL fails to connect at all', async ({
		kernel
	}) => {
		// The complementary fact to the test above: "nothing configured"
		// (a healthy kernel with zero webspaces, or a healthy kernel whose
		// own config fetch failed) is never the same thing as "nothing
		// answering" (no process listening on the port at all). Declared
		// LAST (serial mode above) since this ends the shared kernel every
		// earlier test in this file depends on.
		await kernel.stop();

		let threw = false;
		try {
			await fetch(kernel.baseURL);
		} catch {
			threw = true;
		}
		expect(threw).toBe(true);
	});
});
