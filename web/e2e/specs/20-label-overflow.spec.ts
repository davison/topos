// M3-R3 (#63/#75): the stream row declares its overflow instead of
// clipping it. Mock item 4 carries five labels (three deliberately long)
// — at a desktop viewport the pills must clamp to a +N pill whose title
// names the hidden labels, nothing may paint below the meta strip's box,
// and a row whose labels fit shows no marker at all.
import { test, expect } from '../fixtures/kernel';
import { webspacesWithKeywords, type FixtureConfigSpec } from '../fixtures/config-builder';
import { waitForFirstSync } from '../fixtures/kernel';

const MOCK = 'mock-01';
const WEBSPACE = 'overflow';

const configSpec: FixtureConfigSpec = {
	sources: [{ id: MOCK, plugin: 'topos-plugin-mock', displayName: 'Mock One' }],
	webspaces: webspacesWithKeywords([WEBSPACE], ['demo'])
};

test.use({ configSpec, viewport: { width: 1100, height: 800 } });

test.describe('20: label pills clamp and declare, never clip', () => {
	test.beforeEach(async ({ page, kernel }) => {
		await waitForFirstSync(kernel.baseURL, [MOCK], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);
	});

	test('the long-tailed row shows a +N pill naming the hidden labels', async ({ page }) => {
		const row = page
			.getByRole('main')
			.locator('[data-item-id]', { hasText: 'Shopping list' });
		await expect(row).toHaveCount(1);
		const marker = row.locator('[data-label-overflow]');
		await expect(marker).toBeVisible();
		const hiddenCount = Number(await marker.getAttribute('data-label-overflow'));
		expect(hiddenCount).toBeGreaterThanOrEqual(1);
		await expect(marker).toHaveText(`+${hiddenCount}`);
		// The title declares exactly what is hidden.
		const title = (await marker.getAttribute('title')) ?? '';
		expect(title.length).toBeGreaterThan(0);
		expect(title.split(', ').length).toBe(hiddenCount);
	});

	test('nothing paints below the meta strip — the clip zone is empty by construction', async ({
		page
	}) => {
		const row = page
			.getByRole('main')
			.locator('[data-item-id]', { hasText: 'Shopping list' });
		const strip = row.locator('.stream-row-meta');
		const stripBox = await strip.boundingBox();
		expect(stripBox).not.toBeNull();
		// Every badge inside the strip sits fully inside the strip's box —
		// no second line, no half-clipped pill (the reported symptom).
		const badges = strip.locator('span, [data-label-overflow]');
		const n = await badges.count();
		for (let i = 0; i < n; i++) {
			const b = await badges.nth(i).boundingBox();
			if (!b) continue; // hidden elements have no box — that is the honest state
			expect(b.y + b.height).toBeLessThanOrEqual(stripBox!.y + stripBox!.height + 1);
		}
	});

	test('at a narrow desktop pane the declaration survives horizontally too', async ({ page }) => {
		// PR #78 round 1: the selected-item layout squeezes the stream to
		// its 240–400px pane — the +N pill and the date must stay fully
		// inside the row horizontally; only the label pills may give way.
		await page.setViewportSize({ width: 900, height: 800 });
		const row = page
			.getByRole('main')
			.locator('[data-item-id]', { hasText: 'Shopping list' });
		await row.click(); // opens the detail pane, narrowing the stream
		const marker = row.locator('[data-label-overflow]');
		await expect(marker).toBeVisible();
		const rowBox = await row.boundingBox();
		const markerBox = await marker.boundingBox();
		expect(rowBox && markerBox).toBeTruthy();
		expect(markerBox!.x).toBeGreaterThanOrEqual(rowBox!.x - 1);
		expect(markerBox!.x + markerBox!.width).toBeLessThanOrEqual(rowBox!.x + rowBox!.width + 1);
		expect(markerBox!.y).toBeGreaterThanOrEqual(rowBox!.y - 1);
		expect(markerBox!.y + markerBox!.height).toBeLessThanOrEqual(rowBox!.y + rowBox!.height + 1);
		// Every VISIBLE pill's box sits inside the row too (PR #78 round
		// 2): a pill gives way by shrink+ellipsis, never by a half-cut
		// crop at the region edge.
		const pills = row.locator('.stream-row-meta [data-slot="badge"]:not([data-label-overflow])');
		const n = await pills.count();
		for (let i = 0; i < n; i++) {
			const b = await pills.nth(i).boundingBox();
			if (!b) continue;
			expect(b.x + b.width).toBeLessThanOrEqual(rowBox!.x + rowBox!.width + 1);
		}
	});

	test("at QA's 768-840 band the declaration still survives (#82)", async ({ page }) => {
		// The pane clamp bottoms out here (~216px row, ~73px strip): the
		// date collapses entirely before the declaration gives an inch.
		await page.setViewportSize({ width: 800, height: 800 });
		const row = page
			.getByRole('main')
			.locator('[data-item-id]', { hasText: 'Shopping list' });
		await row.click();
		const marker = row.locator('[data-label-overflow]');
		await expect(marker).toBeVisible();
		const rowBox = await row.boundingBox();
		const markerBox = await marker.boundingBox();
		expect(rowBox && markerBox).toBeTruthy();
		expect(markerBox!.x).toBeGreaterThanOrEqual(rowBox!.x - 1);
		expect(markerBox!.x + markerBox!.width).toBeLessThanOrEqual(rowBox!.x + rowBox!.width + 1);
		expect(markerBox!.y).toBeGreaterThanOrEqual(rowBox!.y - 1);
		expect(markerBox!.y + markerBox!.height).toBeLessThanOrEqual(rowBox!.y + rowBox!.height + 1);
	});

	test('a row whose labels fit shows no overflow marker', async ({ page }) => {
		const row = page
			.getByRole('main')
			.locator('[data-item-id]', { hasText: 'Welcome to the mock source' });
		await expect(row).toHaveCount(1);
		await expect(row.locator('[data-label-overflow]')).toHaveCount(0);
	});
});
