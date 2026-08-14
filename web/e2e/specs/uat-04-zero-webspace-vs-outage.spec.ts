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

	test('a failed config fetch renders the outage copy with a working Retry, and never the zero-webspace empty state', async ({
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

		// 13-04-PLAN.md Task 4 checkpoint-B2 fix: this route's error phase now
		// reuses StreamError.svelte (the same component /w/[webspace]'s own
		// stream-load failure renders) instead of a bespoke, retry-less
		// paragraph — the installed PWA's manifest start_url is '/'
		// (13-UI-SPEC.md E7), so this is the first screen a user launching the
		// installed app with the kernel down actually sees, and UI-SPEC E8
		// requires a working Retry there.
		// StreamError's AlertTitle/AlertDescription render plain <div>s
		// (data-slot="alert-title"/"alert-description"), not heading
		// elements — getByText, not getByRole('heading', ...), matches this
		// component's actual accessible shape.
		await expect(page.getByText("Couldn't load this webspace")).toBeVisible();
		await expect(
			page.getByText("The topos service didn't respond — check that it's running, then retry.")
		).toBeVisible();
		await expect(page.getByRole('heading', { name: 'No webspaces yet' })).toHaveCount(0);

		const retryButton = page.getByRole('button', { name: 'Retry' });
		await expect(retryButton).toBeVisible();

		// Prove Retry is genuinely wired, not decorative: unroute so the NEXT
		// config request succeeds, click Retry, and confirm the route
		// recovers. This file's tests share ONE worker-scoped kernel in
		// serial order (D-02, this file's own header comment) — by this
		// point the earlier test in this file has already created
		// "first-webspace" against that SAME shared kernel, so a
		// successful retry redirects there rather than landing on the
		// zero-webspace empty state (which only the version of this kernel
		// that existed BEFORE that earlier test ever showed).
		await page.unroute('**/api/config');
		await retryButton.click();

		await expect(page).toHaveURL(/\/w\/first-webspace$/);
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
