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
const AGENT_WEBSPACE = 'holiday-agent';

const configSpec: FixtureConfigSpec = {
	sources: [{ id: MOCK, plugin: 'topos-plugin-mock', displayName: 'Mock One', agent: { read: true } }],
	webspaces: webspacesWithKeywords([WEBSPACE, AGENT_WEBSPACE], ['demo'])
};

test.use({ configSpec });

test.describe('21: date-range narrowing, promotion and removal', () => {
	test.beforeEach(async ({ page, kernel }) => {
		await waitForFirstSync(kernel.baseURL, [MOCK], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);
		// Hermetic per test whatever the kernel fixture's scope: a sibling
		// test's PROMOTED range would otherwise leak in — and since
		// promotion intersects with the saved range, an inherited narrower
		// range would silently narrow this test's own save. The check asks
		// the CONFIG API (the chip renders asynchronously — counting it
		// right after goto races the fetch and misses saved state).
		const cfg = await (await page.request.get(`${kernel.baseURL}/api/config`)).json();
		const saved = cfg.config.webspaces[WEBSPACE] ?? {};
		if (saved.date_from || saved.date_to) {
			const staleChip = page.locator('[data-date-filter-chip]');
			await expect(staleChip).toBeVisible();
			await page.getByRole('button', { name: 'Remove date range' }).click();
			await expect(staleChip).toHaveCount(0);
		}
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


	test('promotion persists the intersection with a saved range — never a widening', async ({
		page,
		kernel
	}) => {
		const rows = page.getByRole('main').locator('[data-item-id]');
		// Save 01-02..01-03 first.
		await page.locator('[data-date-from]').fill('2024-01-02');
		await page.locator('[data-date-to]').fill('2024-01-03');
		await page.getByRole('button', { name: 'Save as filter' }).click();
		await expect(page.locator('[data-date-filter-chip]')).toBeVisible();
		// Live: only a one-sided from INSIDE the saved range, then promote.
		await page.locator('[data-date-from]').fill('2024-01-03');
		await expect(rows).toHaveCount(1);
		await page.getByRole('button', { name: 'Save as filter' }).click();
		await expect(rows).toHaveCount(1);
		// The persisted range is the intersection: 01-03..01-03 — the saved
		// `to` survived the one-sided live promotion (PR #79 round 1).
		const cfg = await (await page.request.get(`${kernel.baseURL}/api/config`)).json();
		expect(cfg.config.webspaces[WEBSPACE].date_from).toBe('2024-01-03');
		expect(cfg.config.webspaces[WEBSPACE].date_to).toBe('2024-01-03');
	});

	test('the agent mirror honours the saved range', async ({ page, kernel }) => {
		// Its own webspace: no sibling test promotes state under this name,
		// whatever the worker scheduling.
		await page.goto(`${kernel.baseURL}/w/${AGENT_WEBSPACE}`);
		await page.locator('[data-date-from]').fill('2024-01-02');
		await page.locator('[data-date-to]').fill('2024-01-03');
		await page.getByRole('button', { name: 'Save as filter' }).click();
		await expect(page.locator('[data-date-filter-chip]')).toBeVisible();
		const resp = await page.request.get(
			`${kernel.baseURL}/agent/v1/webspaces/${AGENT_WEBSPACE}/stream`
		);
		expect(resp.ok()).toBeTruthy();
		expect((await resp.json()).items).toHaveLength(2);
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
