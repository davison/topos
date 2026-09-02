// M3-R1 (#70/#76): a webspace narrows by date range. The mock's four
// items sit on 2024-01-01..04 (plus the search-only orphan on 01-05), so
// a live range from the pickers narrows the stream immediately, the same
// Save-as-filter affordance promotes it to a persisted chip, and the
// chip's × widens back — the term flow's exact shape, for dates.
import { test, expect } from '../fixtures/kernel';
import { webspacesWithKeywords, type FixtureConfigSpec } from '../fixtures/config-builder';
import { waitForFirstSync } from '../fixtures/kernel';

const MOCK = 'mock-01';
const WEBSPACE = 'holiday';

const configSpec: FixtureConfigSpec = {
	sources: [{ id: MOCK, plugin: 'topos-plugin-mock', displayName: 'Mock One' }],
	webspaces: webspacesWithKeywords([WEBSPACE], ['demo'])
};

test.use({ configSpec });

test.describe('21: date-range narrowing, promotion and removal', () => {
	test.beforeEach(async ({ page, kernel }) => {
		await waitForFirstSync(kernel.baseURL, [MOCK], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);
	});

	test('the pickers narrow live, Save as filter persists the chip, its × widens back', async ({
		page
	}) => {
		const rows = page.getByRole('main').locator('[data-item-id]');
		await expect(rows).toHaveCount(4);

		// Live: 2024-01-02 through 2024-01-03 → items 2 and 3.
		await page.locator('[data-date-from]').fill('2024-01-02');
		await page.locator('[data-date-to]').fill('2024-01-03');
		await expect(rows).toHaveCount(2);

		// The range alone is promotable — the button appears without a query.
		await page.getByRole('button', { name: 'Save as filter' }).click();
		const chip = page.locator('[data-date-filter-chip]');
		await expect(chip).toBeVisible();
		await expect(chip).toContainText('2024');
		await expect(rows).toHaveCount(2);
		// The pickers cleared on promotion — the narrowing is config truth now.
		await expect(page.locator('[data-date-from]')).toHaveValue('');

		// Persisted: a reload still narrows and still shows the chip.
		await page.reload();
		await expect(page.locator('[data-date-filter-chip]')).toBeVisible();
		await expect(rows).toHaveCount(2);

		// The × clears both sides in one write and the stream widens.
		await page.getByRole('button', { name: 'Remove date range' }).click();
		await expect(rows).toHaveCount(4);
		await expect(page.locator('[data-date-filter-chip]')).toHaveCount(0);
	});

	test('an active search narrows under the live range too', async ({ page }) => {
		const rows = page.getByRole('main').locator('[data-item-id]');
		await expect(rows).toHaveCount(4);
		// "standup" matches items 2 and 3; a range covering only day 2
		// leaves one result.
		await page.getByPlaceholder('Search this webspace').fill('standup');
		await expect(rows).toHaveCount(2, { timeout: 15_000 });
		await page.locator('[data-date-from]').fill('2024-01-02');
		await page.locator('[data-date-to]').fill('2024-01-02');
		await expect(rows).toHaveCount(1, { timeout: 15_000 });
	});
});
