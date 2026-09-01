// M2-R3, #55: filters that speak per-source. One instance's own filter
// terms narrow ITS rows in the stream while the sibling instance is
// untouched; the chip carries a funnel marker; the labelled chip removes
// one term; and an `instance:term` token in a Save-as-filter save lands
// in the same map. Two mock instances, both synced with the same four
// fixture items.
import { test, expect } from '../fixtures/kernel';
import { webspacesWithKeywords, type FixtureConfigSpec } from '../fixtures/config-builder';
import { waitForFirstSync } from '../fixtures/kernel';

const ONE = 'mock-01';
const TWO = 'mock-02';
const WEBSPACE = 'perfilter';
const SEARCH_PLACEHOLDER = 'Search this webspace';

const configSpec: FixtureConfigSpec = {
	sources: [
		{ id: ONE, plugin: 'topos-plugin-mock', displayName: 'Mock One' },
		{ id: TWO, plugin: 'topos-plugin-mock', displayName: 'Mock Two' }
	],
	webspaces: webspacesWithKeywords([WEBSPACE], ['demo'])
};

test.use({ configSpec });

test.describe('19: per-source filter terms narrow one source and not another', () => {
	test.beforeEach(async ({ page, kernel }) => {
		await waitForFirstSync(kernel.baseURL, [ONE, TWO], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);
	});

	test('the chip-menu dialog sets terms, the funnel and labelled chip appear, removal widens back', async ({
		page
	}) => {
		const rows = page.getByRole('main').locator('[data-item-id]');
		await expect(rows).toHaveCount(8); // four fixture items × two instances

		// Open Mock One's menu → "Filter this source…" → save "standup".
		await page.getByRole('button', { name: 'Mock One actions' }).click();
		await page.getByRole('menuitem', { name: 'Filter this source…' }).click();
		const dialog = page.locator(`[data-filter-source-dialog="${ONE}"]`);
		await expect(dialog).toBeVisible();
		await dialog.locator('[data-filter-source-input]').fill('standup');
		await dialog.locator('[data-filter-source-save]').click();
		await expect(dialog).toHaveCount(0);

		// Mock One narrows to its two standup items; Mock Two keeps four.
		await expect(rows).toHaveCount(6);

		// The funnel marker sits on Mock One's chip alone.
		await expect(
			page.getByRole('button', { name: /Mock One/ }).locator('[data-source-filtered]')
		).toBeVisible();
		await expect(
			page.getByRole('button', { name: /Mock Two/ }).locator('[data-source-filtered]')
		).toHaveCount(0);

		// The labelled chip names whose rows it narrows…
		const chip = page.locator('[data-filter-instance="mock-01"]');
		await expect(chip).toContainText('mock-01');
		await expect(chip).toContainText('standup');

		// …and its × removes exactly that term, widening back out.
		await page.getByRole('button', { name: `Remove ${ONE} filter standup` }).click();
		await expect(rows).toHaveCount(8);
		await expect(page.locator('[data-filter-instance="mock-01"]')).toHaveCount(0);
	});

	test('an instance:term token in Save as filter lands in the per-source map', async ({ page }) => {
		const rows = page.getByRole('main').locator('[data-item-id]');
		await expect(rows).toHaveCount(8);

		await page.getByPlaceholder(SEARCH_PLACEHOLDER).fill(`${ONE}:standup`);
		await page.getByRole('button', { name: 'Save as filter' }).click();

		// The save clears the search and the per-source narrowing applies:
		// six rows, a labelled chip, no global filter chip.
		await expect(rows).toHaveCount(6);
		await expect(page.locator('[data-filter-instance="mock-01"]')).toContainText('standup');
		await expect(page.getByPlaceholder(SEARCH_PLACEHOLDER)).toHaveValue('');

		// Search within the narrowed webspace still works and respects the
		// map: "welcome" (item 1) matches only in Mock Two now.
		await page.getByPlaceholder(SEARCH_PLACEHOLDER).fill('welcome');
		await expect(rows).toHaveCount(1, { timeout: 15_000 });
	});
});
