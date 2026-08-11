// Quick task 260811-rqc Task 1's tracer proof, end to end: the header's
// top band renders a branding lockup (app icon + "topos" wordmark +
// tagline) beside the webspace-switcher title, proven in a real browser
// against a real kernel — not inferred from the structural unit guard
// (header-branding.test.ts) alone. Mirrors 09-plugin-icon.spec.ts's
// fixture wiring: a single mock source and one webspace, waitForFirstSync,
// then navigate.
//
// The icon locator is scoped to the page's `banner` region (the <header>
// element's implicit ARIA role — it is not nested inside any sectioning
// element in +page.svelte) so it can never accidentally resolve
// QRPanel.svelte's own copy of the same /app-icon.png asset elsewhere on
// the page.
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import type { FixtureConfigSpec } from '../fixtures/config-builder';

const WEBSPACE = 'branding';

const configSpec: FixtureConfigSpec = {
	sources: [{ id: 'mock', plugin: 'topos-plugin-mock', displayName: 'Mock Source' }],
	webspaces: [{ name: WEBSPACE, keywords: ['demo'] }]
};

test.use({ configSpec });

test.describe('260811-rqc Task 1: header branding lockup renders end to end', () => {
	test('the header banner shows a decoded app icon and both branding texts', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock'], { logs: kernel.logs });

		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const banner = page.getByRole('banner');

		const icon = banner.locator('img[src="/app-icon.png"]');
		await expect(icon).toBeVisible();

		// Prove the image actually DECODED in the browser — naturalWidth is
		// 0 for a broken/never-loaded image, so this is a stronger
		// assertion than merely finding the <img> tag in the DOM.
		await expect
			.poll(async () => icon.evaluate((el: HTMLImageElement) => el.naturalWidth))
			.toBeGreaterThan(0);

		await expect(banner.getByText('topos', { exact: true })).toBeVisible();
		await expect(
			banner.getByText('bringing all your topics to one place', { exact: true })
		).toBeVisible();
	});
});
