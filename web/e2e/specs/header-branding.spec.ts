// Quick task 260811-rqc: the header's top band renders a branding lockup
// (app icon + "topos" wordmark + tagline) beside the webspace-switcher
// title, proven in a real browser against a real kernel — not inferred
// from the structural unit guard (header-branding.test.ts) alone. Mirrors
// 09-plugin-icon.spec.ts's fixture wiring: mock source(s) and one
// webspace, waitForFirstSync, then navigate.
//
// The icon locator is scoped to the page's `banner` region (the <header>
// element's implicit ARIA role — it is not nested inside any sectioning
// element in +page.svelte) so it can never accidentally resolve
// QRPanel.svelte's own copy of the same /app-icon.png asset elsewhere on
// the page.
//
// Three mock instances (not one) are configured from the start so the
// SAME describe block can also carry Task 2's chip-row non-regression
// proof — Task 1's decode/text assertions are unaffected by source count.
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { mockInstances } from '../fixtures/config-builder';
import type { FixtureConfigSpec } from '../fixtures/config-builder';

const WEBSPACE = 'branding';
const INSTANCE_IDS = ['mock-01', 'mock-02', 'mock-03'];

const configSpec: FixtureConfigSpec = {
	sources: mockInstances(3),
	webspaces: [{ name: WEBSPACE, keywords: ['demo'] }]
};

test.use({ configSpec });

test.describe('260811-rqc: header branding lockup', () => {
	test('Task 1: the header banner shows a decoded app icon and both branding texts', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, INSTANCE_IDS, { logs: kernel.logs });

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

	// Task 2: the assertions the Task 1 tracer deliberately left out — the
	// muted-tone proof and the chip-row non-regression proof — run at a
	// realistic desktop width and then a narrower one, confirming the
	// switcher title truncates rather than the branding being pushed
	// off-screen or overlapping the chip row.
	for (const viewport of [
		{ width: 1440, height: 900, label: 'desktop' },
		{ width: 900, height: 900, label: 'narrow' }
	]) {
		test(`Task 2: at ${viewport.label} viewport (${viewport.width}px), branding is muted and the chip row is un-regressed`, async ({
			page,
			kernel
		}) => {
			await page.setViewportSize({ width: viewport.width, height: viewport.height });
			await waitForFirstSync(kernel.baseURL, INSTANCE_IDS, { logs: kernel.logs });

			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			const banner = page.getByRole('banner');
			const wordmark = banner.getByText('topos', { exact: true });
			const tagline = banner.getByText('bringing all your topics to one place', { exact: true });
			const switcherTitle = banner.locator(`span[title="${WEBSPACE}"]`);

			await expect(wordmark).toBeVisible();
			await expect(tagline).toBeVisible();
			await expect(switcherTitle).toBeVisible();

			// Muted-tone proof: compare COMPUTED colour, not a hardcoded hex
			// literal, so this survives a palette change and only fails if
			// the branding stops being muted relative to the application
			// text colour beside it.
			const [wordmarkColor, taglineColor, switcherColor] = await Promise.all([
				wordmark.evaluate((el) => getComputedStyle(el).color),
				tagline.evaluate((el) => getComputedStyle(el).color),
				switcherTitle.evaluate((el) => getComputedStyle(el).color)
			]);
			expect(
				wordmarkColor,
				'expected the wordmark and tagline to share the same (muted) computed colour'
			).toBe(taglineColor);
			expect(
				wordmarkColor,
				'the branding is rendering at full application text weight and competes with the webspace title — expected its computed colour to differ from the switcher title beside it'
			).not.toBe(switcherColor);

			// Chip-row non-regression proof.
			const firstChip = banner.getByRole('button', { name: 'Mock 01', exact: true });
			const addSourceTrigger = banner.getByRole('button', { name: 'Add source', exact: true });
			const refreshAllButton = banner.getByRole('button', { name: 'Refresh all', exact: true });
			await expect(firstChip).toBeVisible();
			await expect(addSourceTrigger).toBeVisible();
			await expect(refreshAllButton).toBeVisible();

			const icon = banner.locator('img[src="/app-icon.png"]');
			const brandingBlock = icon.locator('xpath=..');
			const chipRow = banner.locator('div.overflow-hidden.flex-nowrap');

			const [brandingBox, chipRowBox, headerBox] = await Promise.all([
				brandingBlock.boundingBox(),
				chipRow.boundingBox(),
				banner.boundingBox()
			]);
			expect(brandingBox, 'expected the branding block to have a bounding box').not.toBeNull();
			expect(chipRowBox, 'expected the chip row to have a bounding box').not.toBeNull();
			expect(headerBox, 'expected the header banner to have a bounding box').not.toBeNull();

			// No intersection: the branding's bottom edge sits at or above
			// the chip row's top edge — separate horizontal bands, so they
			// can never overlap regardless of instance count or viewport
			// width.
			expect(
				brandingBox!.y + brandingBox!.height,
				'the branding block overlaps the chip row — they must occupy separate horizontal bands'
			).toBeLessThanOrEqual(chipRowBox!.y);

			// The chip row's own measured width is still essentially the
			// header's full content width (within a small tolerance for
			// the header's own px-6 horizontal padding, 24px per side) —
			// the behavioural counterpart to the structural placement
			// guard in header-branding.test.ts, catching a future refactor
			// that nests the lockup into the measured row (which would
			// shrink this width).
			const expectedContentWidth = headerBox!.width - 2 * 24;
			expect(
				Math.abs(chipRowBox!.width - expectedContentWidth),
				`expected the chip row's width (${chipRowBox!.width}) to be within a few px of the header's content width (${expectedContentWidth}) — a narrower measured row means visibleChipCount received a shrunk available-width input`
			).toBeLessThan(4);
		});
	}
});
