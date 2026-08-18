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
// 390px, under the 14-06 rule (G-14-2): ONE chip is guaranteed inline
// whenever the overflow budget can seat a minimum-width chip
// (MIN_INLINE_CHIP_PX), and everything past that first chip relegates to
// the "+N more sources" popover. The guarantee is budget-gated, not
// unconditional: when a selection makes "Clear filters" appear, the
// widened reserved trailing group drops the overflow budget below the
// minimum and even a single participating chip legitimately relegates —
// which is exactly the state case 3 drives. SINGLE_WEBSPACE (one
// participating chip) is what makes cases 1-5, 7-9 possible at all — a
// genuinely visible-at-rest, tappable trigger with no overflow in the
// way; MULTI_WEBSPACE (three participating chips: the first held inline
// by the floor, the other two deferred) is case 6's own proof that the
// overflow popover still surfaces every chip a narrow viewport cannot
// fit inline.
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

		// 14-02-PLAN.md Task 2 (14-UI-SPEC.md G1, approved design contract)
		// removes the native `title` this test originally required as the
		// touch degrade for chip health detail (09.1-04-PLAN.md R2, RESEARCH
		// Pitfall 2 — "chip health detail is otherwise unreachable without
		// hover"). G1 replaces it with an aria-describedby-wired sr-only span,
		// exposed only to assistive technology, NOT to a plain touch tap. This
		// is a KNOWN, DELIBERATE regression — G1 and its Task 1 checkpoint did
		// not discuss touch-only reachability — recorded in 14-02-SUMMARY.md
		// and the cross-phase WINDOWS.md ledger for follow-up: a touch user on
		// a source chip below 768px who is not running a screen reader can no
		// longer reach the health sentence at all.
		test('5: chip health detail is reachable only to assistive technology now, not via a plain touch tap (14-02-PLAN.md G1 supersedes 09.1-04-PLAN.md R2)', async ({
			page,
			kernel
		}) => {
			await waitForFirstSync(kernel.baseURL, allIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${SINGLE_WEBSPACE}`);

			const filterButton = page.getByRole('button', { name: 'Mock 01', exact: true });

			const title = await filterButton.getAttribute('title');
			expect(
				title,
				'expected NO native title on the filter button — 14-UI-SPEC.md G1 removes it in favour of an aria-describedby-wired description exposed only to assistive tech'
			).toBeNull();

			// The one reachability guarantee G1 DOES preserve: a screen reader
			// still exposes the health sentence as the button's accessible
			// description.
			await expect(filterButton).toHaveAccessibleDescription(/Mock 01/);
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
			// Under the 14-06 rule (G-14-2), the FIRST participating
			// instance is held inline by the minimum-chip floor at this
			// width; everything past it — including Mock 03 — is guaranteed
			// to have overflowed. Asserting on Mock 03 (never Mock 01, the
			// floor's own chip) proves the popover genuinely surfaces a
			// chip that did not fit inline, not merely that it opens.
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
