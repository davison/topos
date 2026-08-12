// 09.1-04-PLAN.md Task 2: proves the CONSTRAINT CONTEXT.md sets for narrow-
// width header adaptation — every existing affordance (chip menu, filter
// toggle, refresh, manage-sources) stays reachable on touch, even relocated
// — rather than asserting the Tailwind classes that implement it. Every
// mobile case below reaches its target WITHOUT calling hover(): a spec that
// hovers first would still pass against the pre-fix code, which is exactly
// the gap RESEARCH Pitfall 2 and this plan's Task 1 close.
//
// Follows 09-1-mobile-takeover.spec.ts's own house pattern (TEST-scoped
// viewport override via test.use({ viewport, hasTouch }), inside make e2e's
// default chromium project, web/playwright.config.ts left unmodified) and
// 09-chip-menu.spec.ts's behavioural conventions for locating the chip's
// filter button vs its "{display_name} actions" overflow trigger.
//
// TWO webspaces share one fixture (D-03: still one kernel per file), each
// attaching a different subset of the same three mock instances — verified
// live (not assumed) against WebspaceHeader's visibleChipCount budget at
// 390px: a single participating chip fits inline with room to spare, but
// as soon as a SECOND chip participates the row's reserved trailing space
// (Refresh all + the add-source trigger + the overflow trigger itself)
// leaves less room than even one chip needs, so two or more participating
// instances push ALL of them behind the "+N more sources" popover with
// none left inline. SINGLE_WEBSPACE (one participating chip) is what makes
// cases 1-5, 7-9 possible at all — a genuinely visible-at-rest, tappable
// trigger with no overflow in the way; MULTI_WEBSPACE (three participating
// chips, all deferred) is case 6's own proof that the overflow popover
// still surfaces every chip a narrow viewport cannot fit inline.
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { attachedWebspace, mockInstances, type FixtureConfigSpec } from '../fixtures/config-builder';

const SINGLE_WEBSPACE = 'header-touch-single';
const MULTI_WEBSPACE = 'header-touch-multi';

const sources = mockInstances(3);
const allIds = sources.map((s) => s.id);

const configSpec: FixtureConfigSpec = {
	sources,
	webspaces: [
		attachedWebspace(SINGLE_WEBSPACE, ['mock-01'], { 'mock-01': { labels: ['demo'] } }),
		attachedWebspace(MULTI_WEBSPACE, allIds, {
			'mock-01': { labels: ['demo'] },
			'mock-02': { labels: ['demo'] },
			'mock-03': { labels: ['demo'] }
		})
	]
};

test.use({ configSpec });

test.describe('09.1-04: header touch adaptation (09.1-TOUCH)', () => {
	test.describe.configure({ mode: 'serial' });

	test.describe('at phone width (390x844)', () => {
		test.use({ viewport: { width: 390, height: 844 }, hasTouch: true });

		test('1: chip actions menu opens on a single tap', async ({ page, kernel }) => {
			await waitForFirstSync(kernel.baseURL, allIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${SINGLE_WEBSPACE}`);

			const trigger = page.getByRole('button', { name: 'Mock 01 actions' });
			await expect(trigger).toBeVisible();

			await trigger.click();
			await expect(page.getByRole('menuitem', { name: 'Refresh now' })).toBeVisible();
		});

		test('2: Refresh now is reachable and genuinely wired', async ({ page, kernel }) => {
			await waitForFirstSync(kernel.baseURL, allIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${SINGLE_WEBSPACE}`);

			let refreshRequestSeen = false;
			await page.route('**/api/sources/mock-01/refresh', async (route) => {
				refreshRequestSeen = true;
				await route.continue();
			});

			await page.getByRole('button', { name: 'Mock 01 actions' }).click();
			await page.getByRole('menuitem', { name: 'Refresh now' }).click();

			await expect
				.poll(() => refreshRequestSeen, {
					timeout: 10_000,
					message: 'expected tapping Refresh now to fire POST /api/sources/mock-01/refresh'
				})
				.toBe(true);

			await page.unroute('**/api/sources/mock-01/refresh');
		});

		test('3: filter toggle is reachable and round-trips through the URL', async ({ page, kernel }) => {
			await waitForFirstSync(kernel.baseURL, allIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${SINGLE_WEBSPACE}`);

			// Selecting the chip is itself a real-behaviour discovery: a
			// selected chip makes "Clear filters" appear in the row's
			// reserved trailing group, which (verified live) widens that
			// reservation enough to push even this webspace's ONE
			// participating chip behind the overflow trigger. That is not a
			// silently-disappeared affordance — the chip relocates to the
			// SAME overflow popover case 6 proves reachable — so the second
			// tap (deselecting) goes through the popover rather than the
			// inline row.
			const filterButton = page.getByRole('button', { name: 'Mock 01', exact: true });
			await expect(filterButton).toBeVisible();

			await filterButton.click();
			expect(page.url()).toContain('sources=');

			const overflowTrigger = page.getByRole('button', { name: /more sources/ });
			await expect(overflowTrigger).toBeVisible();
			await overflowTrigger.click();

			const popover = page.locator('[data-slot="popover-content"]');
			const popoverChip = popover.getByRole('button', { name: 'Mock 01', exact: true });
			await expect(popoverChip).toHaveAttribute('aria-pressed', 'true');

			await popoverChip.click();
			expect(page.url()).not.toContain('sources=');
		});

		test('4: manage sources is reachable and opens its modal', async ({ page, kernel }) => {
			await waitForFirstSync(kernel.baseURL, allIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${SINGLE_WEBSPACE}`);

			await page.getByRole('button', { name: SINGLE_WEBSPACE }).click();
			const manageItem = page.getByRole('menuitem', { name: 'Manage sources…' });
			await expect(manageItem).toBeVisible();
			await manageItem.click();

			await expect(page.getByRole('dialog')).toBeVisible();
		});

		test('5: chip health detail is reachable without hover, via a native title', async ({ page, kernel }) => {
			await waitForFirstSync(kernel.baseURL, allIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${SINGLE_WEBSPACE}`);

			const filterButton = page.getByRole('button', { name: 'Mock 01', exact: true });
			const title = await filterButton.getAttribute('title');

			expect(title, 'expected the filter button to carry a non-empty native title').toBeTruthy();
			expect(
				title,
				'expected the native title to carry the same health copy the desktop tooltip shows, naming the source'
			).toContain('Mock 01');
		});

		test('6: chips that do not fit push into the existing overflow popover', async ({ page, kernel }) => {
			await waitForFirstSync(kernel.baseURL, allIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${MULTI_WEBSPACE}`);

			const overflowTrigger = page.getByRole('button', { name: /more sources/ });
			await expect(overflowTrigger).toBeVisible();

			await overflowTrigger.click();
			const popover = page.locator('[data-slot="popover-content"]');
			await expect(popover).toBeVisible();
			await expect(popover.getByText('More sources', { exact: true })).toBeVisible();
			// All three participating instances are guaranteed to have
			// overflowed at this width (verified live: with two or more
			// participating chips, the row's reserved trailing space leaves
			// less room than even one chip needs) — proving the popover
			// genuinely surfaces a chip that did not fit inline, not merely
			// that it opens.
			await expect(popover.getByRole('button', { name: 'Mock 03', exact: true })).toBeVisible();
		});

		test('7: the branding lockup is not visible, while the webspace switcher is', async ({ page, kernel }) => {
			await waitForFirstSync(kernel.baseURL, allIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${SINGLE_WEBSPACE}`);

			const banner = page.getByRole('banner');
			await expect(banner.getByText('topos', { exact: true })).not.toBeVisible();
			await expect(
				banner.getByText('bringing all your topics to one place', { exact: true })
			).not.toBeVisible();
			await expect(page.getByRole('button', { name: SINGLE_WEBSPACE })).toBeVisible();
		});
	});

	test.describe('at desktop width (1280x900)', () => {
		test.use({ viewport: { width: 1280, height: 900 } });

		test('8: the actions trigger still gates on hover, never always-visible at desktop width', async ({
			page,
			kernel
		}) => {
			await waitForFirstSync(kernel.baseURL, allIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${SINGLE_WEBSPACE}`);

			const trigger = page.getByRole('button', { name: 'Mock 01 actions' });
			await expect(trigger).toBeVisible();
			await expect.poll(() => trigger.evaluate((el) => getComputedStyle(el).opacity)).toBe('0');

			await page.getByRole('button', { name: 'Mock 01', exact: true }).hover();
			await expect.poll(() => trigger.evaluate((el) => getComputedStyle(el).opacity)).toBe('1');
		});

		test('9: the branding lockup is visible', async ({ page, kernel }) => {
			await waitForFirstSync(kernel.baseURL, allIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${SINGLE_WEBSPACE}`);

			const banner = page.getByRole('banner');
			await expect(banner.getByText('topos', { exact: true })).toBeVisible();
			await expect(
				banner.getByText('bringing all your topics to one place', { exact: true })
			).toBeVisible();
		});
	});
});
