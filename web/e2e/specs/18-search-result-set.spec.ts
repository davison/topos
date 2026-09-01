// M2-R2, #54: the result set says where each hit matched, marks a hit the
// index has not synced, carries a per-source status row that lands after
// the index answer (a slow source never blocks the fast ones), and the
// detail pane labels its highlight as find-in-page. Driven by the mock's
// own fixtures: WEBSPACES_MOCK_SEARCH_DELAY_MS (plugins/mock/
// searchfixture.go) reaches only the instance whose extras reference it,
// and topos-plugin-mockstrict implements no Search at all.
import { test, expect } from '../fixtures/kernel';
import { webspacesWithKeywords, type FixtureConfigSpec } from '../fixtures/config-builder';
import { waitForFirstSync } from '../fixtures/kernel';

const FAST = 'mock-01';
const SLOW = 'mock-02';
const STRICT = 'strict-01';
const WEBSPACE = 'results';
const SEARCH_PLACEHOLDER = 'Search this webspace';
// The slow instance sleeps past pluginhost.SearchBudget (five seconds),
// so its outcome is "timeout" and the scope=all answer takes ~5s.
const SLOW_DELAY_MS = '7000';
const SETTLE = { timeout: 20_000 };

const configSpec: FixtureConfigSpec = {
	sources: [
		{ id: FAST, plugin: 'topos-plugin-mock', displayName: 'Fast Mock' },
		{
			id: SLOW,
			plugin: 'topos-plugin-mock',
			displayName: 'Slow Mock',
			extras: { search_delay: '${WEBSPACES_MOCK_SEARCH_DELAY_MS}' }
		},
		{ id: STRICT, plugin: 'topos-plugin-mockstrict', displayName: 'Strict', path: '/strict-never-opened' }
	],
	webspaces: webspacesWithKeywords([WEBSPACE], ['demo']),
	pluginBinaries: ['topos-plugin-mock', 'topos-plugin-mockstrict'],
	env: { WEBSPACES_MOCK_SEARCH_DELAY_MS: SLOW_DELAY_MS }
};

test.use({ configSpec });

test.describe('18: the result set that says where it matched, arriving progressively', () => {
	test.beforeEach(async ({ page, kernel }) => {
		await waitForFirstSync(kernel.baseURL, [FAST, SLOW], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);
	});

	test('a body-only hit arrives tagged, and the status row names every source', async ({ page }) => {
		const rows = page.getByRole('main').locator('[data-item-id]');
		// "mocked" lives only in item 4's body — the index never sees it.
		await page.getByPlaceholder(SEARCH_PLACEHOLDER).fill('mocked');
		const status = page.locator('[data-search-sources]');
		await expect(status).toHaveAttribute('data-search-sources', 'ready', SETTLE);
		await expect(rows).toHaveCount(1);
		const row = rows.first();
		await expect(row).toContainText('Shopping list');
		await expect(row.locator('[data-matched-in="body"]')).toHaveText(/Body/);
		await expect(row.locator('[data-unsynced]')).toHaveCount(0);

		const source = (id: string) => status.locator(`[data-search-source="${id}"]`);
		await expect(source(FAST)).toHaveAttribute('data-search-source-status', 'ok');
		await expect(source(FAST)).toContainText('Fast Mock');
		await expect(source(FAST)).toContainText('1 hit');
		await expect(source(STRICT)).toHaveAttribute('data-search-source-status', 'unsupported');
		await expect(source(STRICT)).toContainText('no content search');
		await expect(source(SLOW)).toHaveAttribute('data-search-source-status', 'timeout');
		await expect(source(SLOW)).toContainText('timed out');
	});

	test('the index answer is on screen while the sources are still answering', async ({ page }) => {
		const rows = page.getByRole('main').locator('[data-item-id]');
		await page.getByPlaceholder(SEARCH_PLACEHOLDER).fill('standup');
		// Items 2 and 3, once per mock instance, from the index — well
		// inside the slow source's budget.
		await expect(rows).toHaveCount(4);
		await expect(page.locator('[data-search-sources="pending"]')).toBeVisible();
		await expect(page.locator('[data-search-sources="pending"]')).toContainText('Searching sources');
		// Then the fan-out lands; the rows stay (both found by index and
		// source: tagged with both).
		await expect(page.locator('[data-search-sources="ready"]')).toBeVisible(SETTLE);
		await expect(rows).toHaveCount(4);
		await expect(rows.first().locator('[data-matched-in]')).toHaveText(/Title|Preview|Body/);
	});

	test('a hit the index has not synced renders from the source and is marked', async ({ page }) => {
		const rows = page.getByRole('main').locator('[data-item-id]');
		await page.getByPlaceholder(SEARCH_PLACEHOLDER).fill('orphaned');
		await expect(rows).toHaveCount(1, SETTLE);
		await expect(rows.first()).toContainText('A note beyond the index');
		await expect(rows.first().locator('[data-unsynced]')).toHaveText(/Not yet synced/);
		await expect(rows.first().locator('[data-matched-in="body"]')).toBeVisible();
	});

	test('the detail pane labels its highlight as find-in-page', async ({ page }) => {
		const rows = page.getByRole('main').locator('[data-item-id]');
		await page.getByPlaceholder(SEARCH_PLACEHOLDER).fill('standup');
		await expect(rows).toHaveCount(4);
		await rows.first().click();
		const label = page.locator('[data-find-in-page]');
		await expect(label).toBeVisible();
		await expect(label).toContainText('Find in page');
		await expect(label).toContainText('standup');
	});
});
