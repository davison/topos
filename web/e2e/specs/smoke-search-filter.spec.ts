// Core-journey smoke set (D-09), continued: search returns and highlights,
// source filter chips filter. Same two-mock-instance, one-webspace seeding
// as smoke-core-journey.spec.ts (Task 1) — reused here rather than shared
// via a fixture helper, since D-03 wants each spec file to seed its own
// state up front, not depend on a sibling file's config.
import { test, expect } from '../fixtures/kernel';
import { webspacesWithKeywords, type FixtureConfigSpec } from '../fixtures/config-builder';
import { waitForFirstSync } from '../fixtures/kernel';

const MOCK_ONE = 'mock-01';
const MOCK_TWO = 'mock-02';

const SEARCH_PLACEHOLDER = 'Search this webspace';
const CLEAR_LABEL = 'Clear search';

const configSpec: FixtureConfigSpec = {
	sources: [
		{ id: MOCK_ONE, plugin: 'topos-plugin-mock', displayName: 'Mock One' },
		{ id: MOCK_TWO, plugin: 'topos-plugin-mock', displayName: 'Mock Two' }
	],
	webspaces: webspacesWithKeywords(['smoke'], ['demo'])
};

test.use({ configSpec });

test.describe('smoke: search returns and highlights, source filter chips filter', () => {
	test('search finds, highlights, clears, and reports no-matches distinctly', async ({ page, kernel }) => {
		await waitForFirstSync(kernel.baseURL, [MOCK_ONE, MOCK_TWO], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/smoke`);

		const rows = page.getByRole('main').locator('[data-item-id]');
		await expect(rows).toHaveCount(8); // the unfiltered baseline (Task 1)

		const searchBox = page.getByPlaceholder(SEARCH_PLACEHOLDER);

		// 1. "standup" returns exactly the rows for items 2 and 3, once per
		// instance — four results — and none of item 1 or 4.
		await searchBox.fill('standup');
		await expect(rows).toHaveCount(4);
		await expect(page.getByText('Welcome to the mock source', { exact: true })).toHaveCount(0);
		await expect(page.getByText('Shopping list', { exact: true })).toHaveCount(0);
		// Each search result row renders BOTH its title and its snippet — for
		// these short titles the snippet excerpt is the title text again
		// verbatim, so a per-title paragraph carries two matching elements
		// per row; `.first()` proves presence without depending on that
		// duplication count, which the `rows` (4) assertion above already
		// pins precisely.
		await expect(page.getByText('Standup: Tuesday', { exact: true }).first()).toBeVisible();
		await expect(page.getByText('Standup: Wednesday', { exact: true }).first()).toBeVisible();

		// 2. At least one rendered element carries the shared
		// .search-highlight class (app.css — the 06-05 shared vocabulary
		// promoted out of DetailPane's own component-scoped style block),
		// and its text, case-insensitively, equals the typed term exactly
		// — not merely contains it.
		const highlighted = page.locator('.search-highlight').first();
		await expect(highlighted).toBeVisible();
		await expect(highlighted).toHaveText(/^standup$/i);

		// 3. Activating the clear control restores the full unfiltered
		// stream to its original row count.
		await page.getByRole('button', { name: CLEAR_LABEL }).click();
		await expect(rows).toHaveCount(8);

		// 4. A query that matches nothing renders the no-matches copy the
		// app derives from the query — distinct from both an error state
		// and the plain empty-stream copy.
		await searchBox.fill('zzzznomatch');
		await expect(page.getByText('No matches for "zzzznomatch"', { exact: true })).toBeVisible();
		await expect(
			page.getByText('Try a different word, or clear the search to see the full stream.', {
				exact: true
			})
		).toBeVisible();
		await expect(page.getByText('Nothing here yet', { exact: true })).toHaveCount(0);
		await expect(page.getByRole('alert')).toHaveCount(0);

		// Clear back to the unfiltered stream before the filter-chip
		// assertions below, which assume the baseline 8-row view.
		await page.getByRole('button', { name: CLEAR_LABEL }).click();
		await expect(rows).toHaveCount(8);

		// 5. Both chips render, named "Mock One" and "Mock Two", in
		// config-declared order (GET /api/sources sorts by instance id,
		// and "mock-01" < "mock-02" — the same order the config declared
		// them in).
		const chipOne = page.getByRole('button', { name: 'Mock One', exact: true });
		const chipTwo = page.getByRole('button', { name: 'Mock Two', exact: true });
		await expect(chipOne).toBeVisible();
		await expect(chipTwo).toBeVisible();

		// 6. Clicking the "Mock One" chip sets its aria-pressed to true and
		// narrows the stream to that instance's four rows; "Mock Two"'s
		// chip stays unpressed.
		await chipOne.click();
		await expect(chipOne).toHaveAttribute('aria-pressed', 'true');
		await expect(chipTwo).toHaveAttribute('aria-pressed', 'false');
		await expect(rows).toHaveCount(4);

		// 8. With "Mock One" selected, none of the visible rows attribute
		// themselves to "Mock Two" — read via each row's own
		// data-item-id, which is built from the item's stable
		// {source}:{source_id} id (StreamRow.svelte), so this is an
		// attribution check, not a count that two identical-shaped
		// corpora would leave ambiguous.
		await expect(page.locator('[data-item-id^="mock-02:"]')).toHaveCount(0);
		await expect(page.locator('[data-item-id^="mock-01:"]')).toHaveCount(4);

		// 7. "Clear filters" appears while a chip is selected, and
		// activating it restores both the unpressed state and the full
		// row count.
		const clearFilters = page.getByRole('button', { name: 'Clear filters' });
		await expect(clearFilters).toBeVisible();
		await clearFilters.click();
		await expect(chipOne).toHaveAttribute('aria-pressed', 'false');
		await expect(rows).toHaveCount(8);
	});
});
