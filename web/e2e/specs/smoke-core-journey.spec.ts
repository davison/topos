// Core-journey smoke set (D-09): the four things that must never break at
// all — the stream renders every seeded item, the detail pane opens with
// real content, and (smoke-search-filter.spec.ts) search finds/highlights
// and source filter chips filter. This file covers the first two. Seeded
// with TWO topos-plugin-mock instances so a regression that silently
// merges two instances into one shows up as a wrong row count, not merely
// an empty stream — the mock corpus (plugins/mock/plugin.go) is four items
// all carrying the "demo" label, so both instances contribute all four
// under the webspace's keywords fallback: 2 x 4 = 8 rows.
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { webspacesWithKeywords, type FixtureConfigSpec } from '../fixtures/config-builder';

const MOCK_ONE = 'mock-01';
const MOCK_TWO = 'mock-02';

const MOCK_TITLES = [
	'Welcome to the mock source',
	'Standup: Tuesday',
	'Standup: Wednesday',
	'Shopping list'
];

// mockFullText["1"] verbatim (plugins/mock/plugin.go) — a distinctive
// substring proving the DETAIL PANE'S OWN extracted-text fetch landed, not
// a paraphrase and not the stream's cached item.preview (a different
// string entirely).
const ITEM_1_FULL_TEXT_SUBSTRING = "mock source plugin's full extracted text for item 1";

const configSpec: FixtureConfigSpec = {
	sources: [
		{ id: MOCK_ONE, plugin: 'topos-plugin-mock', displayName: 'Mock One' },
		{ id: MOCK_TWO, plugin: 'topos-plugin-mock', displayName: 'Mock Two' }
	],
	webspaces: webspacesWithKeywords(['smoke'], ['demo'])
};

test.use({ configSpec });

test.describe('smoke: core journey — stream renders, detail pane opens with content', () => {
	test('the stream renders both instances, and the detail pane loads real content for a clicked row', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [MOCK_ONE, MOCK_TWO], { logs: kernel.logs });

		await page.goto(`${kernel.baseURL}/w/smoke`);

		// 1. The webspace switcher trigger renders the webspace name, and
		// the document title follows the "{webspace} — topos" format.
		await expect(page.getByRole('button', { name: 'smoke' })).toBeVisible();
		await expect(page).toHaveTitle('smoke — topos');

		// 2. Exactly eight rows — the mock corpus is four items and both
		// instances contribute independently under the kernel's
		// {source}:{source_id} stable-ID scheme. An exact count (not a
		// non-empty check) is what catches a regression that silently
		// merges the two instances into one.
		const rows = page.getByRole('main').locator('[data-item-id]');
		await expect(rows).toHaveCount(8);

		// 3. All four mock titles are visible (each appears twice — once
		// per instance — so `.first()` avoids a strict-mode violation).
		for (const title of MOCK_TITLES) {
			await expect(page.getByText(title, { exact: true }).first()).toBeVisible();
		}

		// 4. Clicking the row titled "Welcome to the mock source" selects
		// it (aria-pressed becomes true) and the detail pane heading shows
		// that same title.
		const item1Row = page
			.getByRole('button')
			.filter({ hasText: 'Welcome to the mock source' })
			.first();
		await item1Row.click();
		await expect(item1Row).toHaveAttribute('aria-pressed', 'true');
		await expect(page.getByRole('heading', { name: 'Welcome to the mock source' })).toBeVisible();

		// 5. The detail pane renders the item's own live-fetched extracted
		// text — a distinctive substring of mockFullText["1"], read from
		// plugins/mock/plugin.go, not a paraphrase.
		await expect(page.getByText(ITEM_1_FULL_TEXT_SUBSTRING)).toBeVisible();

		// 6. The mock offers no rendition for any variant, but DOES return
		// extracted text for the FULL variant — a normal "available: true,
		// no rendition" outcome (docs/plugin-contract.md's Fetch section),
		// not a failure. The detail pane's own Alert component always
		// carries role="alert" (web/src/lib/components/ui/alert/alert.svelte)
		// regardless of variant (destructive error vs. advisory
		// deleted/unreachable), so asserting zero of them proves neither
		// branch fired — the pane is showing the loaded text, not an error
		// or "no longer available" treatment.
		await expect(page.getByRole('alert')).toHaveCount(0);

		// 7. Selecting a second row moves the detail pane to that row's
		// title and clears the first row's pressed state.
		const item2Row = page.getByRole('button').filter({ hasText: 'Standup: Tuesday' }).first();
		await item2Row.click();
		await expect(item2Row).toHaveAttribute('aria-pressed', 'true');
		await expect(item1Row).toHaveAttribute('aria-pressed', 'false');
		await expect(page.getByRole('heading', { name: 'Standup: Tuesday' })).toBeVisible();
	});
});
