// 09.1-03-PLAN.md Task 3: measures the compact stream/search-result row's
// rendered geometry across all three responsive bands (D-05..D-08), turning
// every must_haves density claim into a Playwright assertion — class-string
// assertions already live in StreamRow.svelte's own literal max-md:
// utilities and marker-overlay.test.ts's structural guards; this spec
// proves the cascade actually resolves the way 09.1-RESEARCH.md's Pattern 3
// predicted, in a real browser. Runs inside make e2e's default chromium
// project via a TEST-scoped viewport override (09.1-01-PLAN.md's own R2
// convention, followed here rather than a new Playwright project) —
// web/playwright.config.ts is unmodified.
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import {
	webspacesWithKeywords,
	type FixtureConfigSpec,
	type FixtureSourceSpec
} from '../fixtures/config-builder';

const WEBSPACE = 'compact-cards';

// mock-01 carries a deliberately long display name — long enough that, for
// an item with no group_label (mock-01:1, "Welcome to the mock source"),
// the compact meta line's single truncating unit overflows a 390px row on
// its own. This is what makes case 5 (long-text truncation) provable
// without a second, artificial fixture — plugins/mock/plugin.go's fixed
// four-item corpus (source ids "1".."4") is otherwise reused unmodified.
const LONG_DISPLAY_NAME = 'Extremely Long Mock Instance Display Name That Overflows A Compact Row';

const sources: FixtureSourceSpec[] = [
	{ id: 'mock-01', plugin: 'topos-plugin-mock', displayName: LONG_DISPLAY_NAME },
	{ id: 'mock-02', plugin: 'topos-plugin-mock', displayName: 'Mock 02' }
];
const attachedIds = sources.map((s) => s.id);
const configSpec: FixtureConfigSpec = {
	sources,
	webspaces: webspacesWithKeywords([WEBSPACE], ['demo'])
};

test.use({ configSpec });

const SEARCH_PLACEHOLDER = 'Search this webspace';

// A minimal, real-shaped GET /api/sources response (docs/api.md's own
// documented shape) reporting mock-01 unreachable — the D-07 stale-dot
// fixture. Registered via page.route (browser-side only); waitForFirstSync
// polls with Node's own global fetch, entirely unaffected by a page-scoped
// route, so real sync landing is always proven BEFORE this route is
// installed for a given test (register-before-navigate discipline, per
// g-08-3-degraded-source-not-outage.spec.ts).
function degradedSourcesBody() {
	return {
		schema_version: 1,
		sources: [
			{
				name: 'mock-01',
				source_type: 'mock',
				display_name: LONG_DISPLAY_NAME,
				plugin: 'topos-plugin-mock',
				reachable: false,
				syncing: false,
				last_status: 'error',
				last_sync_unix: 1785000000,
				last_error: 'e2e-injected 09.1-03 compact-card degraded fixture — connection refused'
			},
			{
				name: 'mock-02',
				source_type: 'mock',
				display_name: 'Mock 02',
				plugin: 'topos-plugin-mock',
				reachable: true,
				syncing: false,
				last_status: 'ok',
				last_sync_unix: 1785000000,
				last_error: ''
			}
		]
	};
}

// The date-marker ruler's overlay container (StreamDateMarkers.svelte) —
// the one element in this route carrying the right-[12px] arbitrary-value
// offset (marker-overlay.test.ts's own "exactly one" structural guard).
// Present in the DOM at every width (+page.svelte gates it on load state,
// not viewport); max-md:hidden is what makes it invisible below 768px, so
// .toBeVisible()/.not.toBeVisible() — never toHaveCount — is the correct
// check here.
const RULER_SELECTOR = '[class*="right-[12px]"]';

test.describe('09.1-03: compact stream/search-result cards (D-05..D-08)', () => {
	test.describe.configure({ mode: 'serial' });

	test.describe('at phone width (390x844)', () => {
		test.use({ viewport: { width: 390, height: 844 }, hasTouch: true });

		test('1: a stream row is 60px tall (D-06)', async ({ page, kernel }) => {
			await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			const row = page.locator('[data-item-id="mock-02:1"]');
			await expect(row).toBeVisible();
			const height = await row.evaluate((el) => el.getBoundingClientRect().height);
			expect(height).toBe(60);
		});

		test('2: the source display name renders as visible TEXT, not only a hover title (D-07)', async ({
			page,
			kernel
		}) => {
			await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			// mock-02:1 ("Welcome to the mock source") has no group_label, so
			// its compact meta text is the display name alone — located by
			// its rendered string (getByText matches textContent, never a
			// title attribute), scoped to this one row so a header chip of
			// the same name elsewhere on the page cannot satisfy this
			// assertion by accident.
			const row = page.locator('[data-item-id="mock-02:1"]');
			await expect(row.getByText('Mock 02', { exact: false })).toBeVisible();
		});

		test('3: the date remains visible in the compact row', async ({ page, kernel }) => {
			await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			const row = page.locator('[data-item-id="mock-02:1"]');
			// formatItemDate's exact rendering (locale-dependent day/month
			// order) is not this case's concern — only that a year renders
			// visibly inside the row.
			await expect(row.getByText(/2024/)).toBeVisible();
		});

		test('4: the thumbnail and tag pills are not visible at compact size (D-06)', async ({ page, kernel }) => {
			await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			// mock-02:4 ("Shopping list") is the one mock item with
			// has_thumbnail=true and two labels ("demo", "errands") — the
			// only item where hiding both is a meaningful check, not a
			// vacuous pass over an item that never had either at any width.
			// Targets the Thumbnail WRAPPER div (max-md:hidden), not the
			// `<img>` inside it directly — the mock plugin never has a real
			// thumbnail rendition to serve (Fetch's THUMBNAIL variant always
			// reports unavailable), so the `<img>`'s own onerror races
			// Thumbnail.svelte's fallback-icon swap; the wrapper is stable
			// across that race, since it hosts whichever branch is current.
			const row = page.locator('[data-item-id="mock-02:4"]');
			await expect(row).toBeVisible();
			await expect(row.locator('div[class*="max-md:hidden"]').first()).not.toBeVisible();
			await expect(row.getByText('errands', { exact: true })).not.toBeVisible();
		});

		test('5: a long source display name truncates while the date stays fully visible (D-07)', async ({
			page,
			kernel
		}) => {
			await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			// mock-01:1 has no group_label either, so its truncating unit is
			// LONG_DISPLAY_NAME alone — already long enough on its own to
			// overflow the row (jointly-truncating behaviour with a group
			// label is the identical mechanism, exercised for the group-label
			// case by mock-01's items 2/3 in the wider corpus; asserting the
			// mechanism once, on the simplest overflowing case, is what
			// actually proves the CSS truncates rather than merely that the
			// text happens to fit).
			const row = page.locator('[data-item-id="mock-01:1"]');
			await expect(row).toBeVisible();
			// span.truncate is unique within a row: the title <p> also
			// carries "truncate", but only the compact name/group-label unit
			// is a <span> with that class (the plugin-icon, date, and stale-
			// dot spans never carry it).
			const nameSpan = row.locator('span.truncate');
			await expect(nameSpan).toContainText(LONG_DISPLAY_NAME);
			const overflow = await nameSpan.evaluate((el) => ({
				scrollWidth: el.scrollWidth,
				clientWidth: el.clientWidth
			}));
			expect(
				overflow.scrollWidth,
				'the long display name must overflow its truncating span at 390px for this case to prove anything'
			).toBeGreaterThan(overflow.clientWidth);
			await expect(row.getByText(/2024/)).toBeVisible();
		});

		test('6: the stale/warning dot survives inside a 60px row for a degraded source (D-07)', async ({
			page,
			kernel
		}) => {
			await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
			await page.route(`${kernel.baseURL}/api/sources`, (route) =>
				route.fulfill({
					status: 200,
					contentType: 'application/json',
					body: JSON.stringify(degradedSourcesBody())
				})
			);
			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			const row = page.locator('[data-item-id="mock-01:1"]');
			await expect(row).toBeVisible();
			const height = await row.evaluate((el) => el.getBoundingClientRect().height);
			expect(height).toBe(60);
			await expect(row.locator('[aria-label*="unreachable"]')).toBeVisible();
		});

		test('7: search-result rows share the compact geometry, with highlighting intact (D-08)', async ({
			page,
			kernel
		}) => {
			await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			await page.getByPlaceholder(SEARCH_PLACEHOLDER).fill('standup');
			const results = page.locator('[data-item-id]');
			await expect(results.first()).toBeVisible();
			const height = await results.first().evaluate((el) => el.getBoundingClientRect().height);
			expect(height).toBe(60);
			await expect(page.locator('.search-highlight').first()).toBeVisible();
		});

		test('8: the date-marker ruler is not visible (D-08)', async ({ page, kernel }) => {
			await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			await expect(page.locator(RULER_SELECTOR)).not.toBeVisible();
		});
	});

	test.describe('at desktop width (1280x900)', () => {
		test.use({ viewport: { width: 1280, height: 900 } });

		test('9: a stream row is 152px tall — desktop density is untouched (D-05)', async ({ page, kernel }) => {
			await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			const row = page.locator('[data-item-id="mock-02:1"]');
			await expect(row).toBeVisible();
			const height = await row.evaluate((el) => el.getBoundingClientRect().height);
			expect(height).toBe(152);
		});

		test('10: the thumbnail and the date-marker ruler are visible at desktop width (D-05)', async ({
			page,
			kernel
		}) => {
			await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			// See case 4's comment: targets the Thumbnail wrapper div, not
			// the `<img>` inside it, since the mock plugin's own thumbnail
			// fetch always fails (no real rendition to serve) and the
			// `<img>`'s onerror fallback swap is a genuine race the wrapper
			// is stable across.
			const row = page.locator('[data-item-id="mock-02:4"]');
			await expect(row.locator('div[class*="max-md:hidden"]').first()).toBeVisible();
			await expect(page.locator(RULER_SELECTOR)).toBeVisible();
		});
	});

	test.describe('at mid-band width (900x900)', () => {
		test.use({ viewport: { width: 900, height: 900 } });

		test('11: a stream row is still 152px tall — compact is a mobile treatment, not a narrow-window one (D-05)', async ({
			page,
			kernel
		}) => {
			await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			const row = page.locator('[data-item-id="mock-02:1"]');
			await expect(row).toBeVisible();
			const height = await row.evaluate((el) => el.getBoundingClientRect().height);
			expect(height).toBe(152);
		});
	});
});
