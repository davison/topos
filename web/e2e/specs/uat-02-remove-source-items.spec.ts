// Ports 07-UAT.md item 2 (G-07-7) into a permanent regression gate.
//
// Verbatim pass condition this spec encodes (07-UAT.md item 2):
//   "make dev; chip ⋮ menu → Remove from this webspace: chip AND that
//   source's items disappear together with no manual refresh. Re-add the
//   instance via "+": chip returns immediately and its items appear once
//   sync completes, without a manual refresh. A background sync completing
//   on a viewed webspace never flashes the loading skeleton, and a failed
//   background refetch leaves the rendered stream untouched."
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { attachedWebspace, type FixtureConfigSpec, type FixtureSourceSpec } from '../fixtures/config-builder';
import { readConfigToml } from '../fixtures/toml';

const sources: FixtureSourceSpec[] = [
	{ id: 'mock-01', plugin: 'topos-plugin-mock', displayName: 'Mock One' },
	{ id: 'mock-02', plugin: 'topos-plugin-mock', displayName: 'Mock Two' }
];

const configSpec: FixtureConfigSpec = {
	sources,
	webspaces: [
		attachedWebspace('armor', ['mock-01', 'mock-02'], {
			'mock-01': { labels: ['demo'] },
			'mock-02': { labels: ['demo'] }
		})
	]
};

test.use({ configSpec });

// Full mock corpus per instance (plugins/mock/plugin.go's fixed 4-item set,
// every item carrying the "demo" label) — with two instances both matching
// on "demo", the baseline row count is 4 * 2 = 8.
const BASELINE_ROW_COUNT = 8;
const HALVED_ROW_COUNT = 4;

test.describe("07-UAT item 2 (G-07-7): a removed source's items leave with its chip, and background refresh is silent", () => {
	test("remove takes the chip and the items; re-add restores both; background sync is silent; a failed background refetch is inert", async ({
		page,
		kernel
	}) => {
		// Installed BEFORE the first navigation (page.addInitScript runs on
		// every subsequent document) — a MutationObserver that records
		// `[data-slot="skeleton"]` insertions only while "armed", with a
		// resettable count so the spec controls exactly which window is
		// under observation (07.1-04-PLAN.md Task 2, clause 3).
		await page.addInitScript(() => {
			window.__skeletonArmed = false;
			window.__skeletonInsertionCount = 0;

			function isSkeletonNode(node: Node): boolean {
				if (!(node instanceof Element)) return false;
				if (node.matches('[data-slot="skeleton"]')) return true;
				return node.querySelector('[data-slot="skeleton"]') !== null;
			}

			const observer = new MutationObserver((mutations) => {
				if (!window.__skeletonArmed) return;
				for (const mutation of mutations) {
					for (let i = 0; i < mutation.addedNodes.length; i++) {
						if (isSkeletonNode(mutation.addedNodes[i])) {
							window.__skeletonInsertionCount = (window.__skeletonInsertionCount as number) + 1;
						}
					}
				}
			});
			// Observing `document` itself (not `document.documentElement`):
			// page.addInitScript() runs before the document has any content
			// parsed, so `document.documentElement` can still be null at this
			// exact point — passing it to observe() throws
			// "parameter 1 is not of type 'Node'" and silently aborts this
			// whole init script before the arm/disarm bridge functions below
			// are ever assigned. `document` (the Document node itself) always
			// exists from the very start and observing it with
			// `subtree: true` still sees every insertion anywhere in the page.
			observer.observe(document, { childList: true, subtree: true });

			window.__armSkeletonObserver = () => {
				window.__skeletonInsertionCount = 0;
				window.__skeletonArmed = true;
			};
			window.__disarmSkeletonObserver = () => {
				window.__skeletonArmed = false;
			};
		});

		await waitForFirstSync(kernel.baseURL, ['mock-01', 'mock-02'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/armor`);

		// --- Baseline: two chips, full row count from both instances ------
		await expect(page.getByRole('button', { name: 'Mock One actions' })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Mock Two actions' })).toBeVisible();
		await expect(page.locator('[data-item-id]')).toHaveCount(BASELINE_ROW_COUNT);

		// --- Clause 1: remove takes the chip AND the items, no reload -----
		await page.getByRole('button', { name: 'Mock Two actions' }).click();
		await page.getByRole('menuitem', { name: 'Remove from this webspace' }).click();

		await expect(page.getByRole('button', { name: 'Mock Two actions' })).toHaveCount(0);
		await expect(page.locator('[data-item-id]')).toHaveCount(HALVED_ROW_COUNT);
		await expect(page.locator('[data-item-id^="mock-02:"]')).toHaveCount(0);
		await expect(page.locator('[data-item-id^="mock-01:"]')).toHaveCount(HALVED_ROW_COUNT);

		const afterRemove = readConfigToml(kernel.configPath);
		const wsAfterRemove = (
			afterRemove.webspaces as Record<string, { sources: string[]; match: Record<string, unknown> }>
		).armor;
		expect(wsAfterRemove.sources).toEqual(['mock-01']);
		expect(Object.keys(wsAfterRemove.match)).toEqual(['mock-01']);

		// --- Clause 2: re-add restores chip immediately and items once ----
		// sync completes, still with no reload.
		await page.getByRole('button', { name: 'Add source' }).click();
		await page.getByRole('button', { name: /Mock Two/ }).click();

		const matchDialog = page.getByRole('dialog');
		await expect(matchDialog.getByRole('heading', { name: 'Add Mock Two to armor' })).toBeVisible();
		await matchDialog.getByLabel('Labels').fill('demo');
		await matchDialog.getByRole('button', { name: 'Add source' }).click();

		// The chip returns without a page reload.
		await expect(page.getByRole('button', { name: 'Mock Two actions' })).toBeVisible();

		// The items appear once the resync lands — still no manual refresh
		// or reload anywhere in this spec.
		await expect(page.locator('[data-item-id]')).toHaveCount(BASELINE_ROW_COUNT, { timeout: 15_000 });

		const afterReadd = readConfigToml(kernel.configPath);
		const wsAfterReadd = (
			afterReadd.webspaces as Record<string, { sources: string[]; match: Record<string, unknown> }>
		).armor;
		expect([...wsAfterReadd.sources].sort()).toEqual(['mock-01', 'mock-02']);
		expect(Object.keys(wsAfterReadd.match).sort()).toEqual(['mock-01', 'mock-02']);

		// --- Clause 3: a completing background sync never flashes the -----
		// skeleton. Exercised via the chip's own Refresh control
		// (handleRefreshSource, web/src/routes/w/[webspace]/+page.svelte):
		// that handler ALWAYS does TWO things — an immediate, NON-quiet
		// foreground `load()` in its own `finally` block (fetch #1, allowed
		// to flash the skeleton by design — it is directly user-initiated,
		// same carve-out as clause 1's remove and clause 2's add above) AND
		// unconditionally calls ensurePolling(), which arms a 2s-interval
		// poll whose STOP BRANCH performs exactly one QUIET refetch (fetch
		// #2) once syncing next reports false. setInterval's own semantics
		// mean fetch #2 fires only after a full tick elapses, making it a
		// deterministic background-path trigger — discovered live
		// (07.1-04-SUMMARY.md) that arming the observer immediately on
		// click, rather than isolating specifically the window BETWEEN
		// fetch #1 and fetch #2, wrongly attributed fetch #1's own
		// permitted skeleton flash to the background path. Counting fetches
		// (not merely "was the route hit") is what makes this isolation
		// exact rather than timing-dependent.
		let streamFetchCount = 0;
		await page.route('**/api/webspaces/armor/stream', async (route) => {
			streamFetchCount++;
			await route.continue();
		});

		await page.getByRole('button', { name: 'Mock One actions' }).click();
		await page.getByRole('menuitem', { name: 'Refresh now' }).click();
		// Fetch #1: the foreground load — deliberately observed BEFORE
		// arming, so its own permitted skeleton flash is excluded.
		await expect.poll(() => streamFetchCount, { timeout: 15_000 }).toBeGreaterThanOrEqual(1);

		await page.evaluate(() => {
			(window.__armSkeletonObserver as () => void)();
		});

		// Fetch #2: the background poll's own quiet refetch.
		await expect.poll(() => streamFetchCount, { timeout: 15_000 }).toBeGreaterThanOrEqual(2);

		const skeletonInsertionsDuringBackgroundSync = await page.evaluate(
			() => window.__skeletonInsertionCount as number
		);
		expect(skeletonInsertionsDuringBackgroundSync).toBe(0);

		await page.evaluate(() => {
			(window.__disarmSkeletonObserver as () => void)();
		});
		await page.unroute('**/api/webspaces/armor/stream');

		// --- Clause 4: a failed background refetch leaves the rendered ----
		// stream untouched (and never flashes the skeleton either). Same
		// fetch-count isolation as clause 3: fetch #1 (the foreground,
		// user-initiated refresh) is left to SUCCEED normally — a manual
		// refresh's own failure legitimately surfacing an error is a
		// different, already-covered code path, not what this clause
		// guards — only fetch #2 onward (the background poll's own quiet
		// refetch) is made to fail, isolating exactly the case the UAT item
		// names: "a failed BACKGROUND refetch leaves the rendered stream
		// untouched".
		let clause4FetchCount = 0;
		await page.route('**/api/webspaces/armor/stream', async (route) => {
			clause4FetchCount++;
			if (clause4FetchCount === 1) {
				await route.continue();
				return;
			}
			await route.fulfill({
				status: 503,
				contentType: 'application/json',
				body: JSON.stringify({
					schema_version: 1,
					error: { code: 'internal_error', message: 'e2e-injected background-refresh failure' }
				})
			});
		});

		await page.getByRole('button', { name: 'Mock One actions' }).click();
		await page.getByRole('menuitem', { name: 'Refresh now' }).click();
		await expect.poll(() => clause4FetchCount, { timeout: 15_000 }).toBeGreaterThanOrEqual(1);

		await page.evaluate(() => {
			(window.__armSkeletonObserver as () => void)();
		});

		await expect.poll(() => clause4FetchCount, { timeout: 15_000 }).toBeGreaterThanOrEqual(2);

		// The previously rendered rows are still present and unchanged —
		// never fewer (a failed quiet load must never blank the view) and
		// never more (nothing new landed either) — and no outage copy or
		// skeleton appeared during the background (fetch #2) window.
		await expect(page.locator('[data-item-id]')).toHaveCount(BASELINE_ROW_COUNT);
		await expect(page.getByText("Couldn't load this webspace")).toHaveCount(0);

		const skeletonInsertionsDuringFailedRefetch = await page.evaluate(
			() => window.__skeletonInsertionCount as number
		);
		expect(skeletonInsertionsDuringFailedRefetch).toBe(0);

		await page.evaluate(() => {
			(window.__disarmSkeletonObserver as () => void)();
		});

		// Recovery, not merely tolerance: remove the failing handler and
		// confirm a further refresh still updates normally.
		await page.unroute('**/api/webspaces/armor/stream');
		await page.getByRole('button', { name: 'Mock One actions' }).click();
		await page.getByRole('menuitem', { name: 'Refresh now' }).click();
		await expect(page.locator('[data-item-id]')).toHaveCount(BASELINE_ROW_COUNT, { timeout: 15_000 });
		await expect(page.getByText("Couldn't load this webspace")).toHaveCount(0);
	});
});
