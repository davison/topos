// 09.1-01-PLAN.md Task 3: the mobile detail takeover, end to end. Every
// case below is one of this plan's own `must_haves` truths made
// executable (D-01..D-04, plus the UI-SPEC E1/E3 resolutions), run inside
// `make e2e`'s default chromium project via a TEST-scoped viewport
// override (planner_resolutions R2) rather than a new Playwright project —
// `web/playwright.config.ts` is unmodified by this spec.
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { mockInstances, webspacesWithKeywords, type FixtureConfigSpec } from '../fixtures/config-builder';

const WEBSPACE = 'mobile-takeover';
const INSTANCE_DISPLAY_NAME = 'Mock 01';

// One mock instance (four fixed items, every one tagged "demo" — see
// plugins/mock/plugin.go's mockItems) is enough surface for every case
// below; a webspace matching on the shared "demo" keyword picks up all
// four without an explicit per-instance match block.
const sources = mockInstances(1);
const attachedIds = sources.map((s) => s.id);
const configSpec: FixtureConfigSpec = {
	sources,
	webspaces: webspacesWithKeywords([WEBSPACE], ['demo'])
};

test.use({ configSpec });

// Unique to the stream's own scroll region (+page.svelte): no other
// element in this route carries BOTH overflow-y-auto and
// overflow-x-hidden at once — see pane-layout.test.ts's own
// findUniqueClassAttr for the same "exactly one marker" discipline,
// applied here as a CSS locator rather than a source-scan regex since
// this is a real rendered page, not source text.
const STREAM_SCROLL_SELECTOR = '.overflow-y-auto.overflow-x-hidden';

// The takeover slim bar's title <p> is the one element immediately
// following the back button in the DOM (DetailPane.svelte's bar markup:
// Button, then the title <p>, then OpenInSource) — a general-sibling CSS
// selector reaches it precisely, without depending on ancestor divs or
// duplicate title text elsewhere on the page (the header's own h2 carries
// the identical title but is display:none below md).
const BAR_TITLE_SELECTOR = '[aria-label="Back to stream"] ~ p';

test.describe('09.1-01: mobile detail takeover (D-01..D-04)', () => {
	test.describe.configure({ mode: 'serial' });

	test.describe('at phone width (390x844)', () => {
		test.use({ viewport: { width: 390, height: 844 }, hasTouch: true });

		test('1: takeover opens full-viewport and the header gives way', async ({ page, kernel }) => {
			await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			const firstRow = page.locator('[data-item-id]').first();
			await expect(firstRow).toBeVisible();
			await firstRow.click();

			// The takeover's own chrome is up...
			await expect(page.getByRole('button', { name: 'Back to stream' })).toBeVisible();
			// ...and both concealed surfaces (D-04) are genuinely not
			// visible — not merely absent from view, but visibility:hidden
			// (the stream row) / display:none (the header), which
			// toBeVisible()'s underlying isVisible() check treats
			// identically to "not visible".
			await expect(firstRow).not.toBeVisible();
			await expect(page.getByRole('button', { name: WEBSPACE })).not.toBeVisible();
		});

		test('2: browser back closes it and stays in the app', async ({ page, kernel }) => {
			await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			await page.locator('[data-item-id]').first().click();
			await expect(page.getByRole('button', { name: 'Back to stream' })).toBeVisible();

			await page.goBack();

			await expect(page.locator('[data-item-id]').first()).toBeVisible();
			await expect(page.getByRole('button', { name: 'Back to stream' })).toHaveCount(0);
			await expect(page).toHaveURL(new RegExp(`/w/${WEBSPACE}$`));
		});

		test('3: the visible back control behaves identically to the browser back button', async ({
			page,
			kernel
		}) => {
			await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			await page.locator('[data-item-id]').first().click();
			await page.getByRole('button', { name: 'Back to stream' }).click();

			await expect(page.locator('[data-item-id]').first()).toBeVisible();
			await expect(page.getByRole('button', { name: 'Back to stream' })).toHaveCount(0);
			await expect(page).toHaveURL(new RegExp(`/w/${WEBSPACE}$`));
		});

		test('4: the stream keeps its scroll position across an open/close round trip (D-01)', async ({
			page,
			kernel
		}) => {
			await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			const scrollBox = page.locator(STREAM_SCROLL_SELECTOR);
			await scrollBox.evaluate((el) => {
				el.scrollTop = el.scrollHeight - el.clientHeight;
			});
			const scrolledOffset = await scrollBox.evaluate((el) => el.scrollTop);
			expect(
				scrolledOffset,
				'the fixture stream must be tall enough to scroll at 390x844 for this case to prove anything'
			).toBeGreaterThan(0);

			// Open the LAST row (the one now nearest the bottom of the
			// scrolled view) rather than .first(), since .first() would be
			// scrolled out of view at this offset.
			await page.locator('[data-item-id]').last().click();
			await expect(page.getByRole('button', { name: 'Back to stream' })).toBeVisible();

			await page.goBack();
			await expect(page.getByRole('button', { name: 'Back to stream' })).toHaveCount(0);

			const restoredOffset = await scrollBox.evaluate((el) => el.scrollTop);
			expect(
				restoredOffset,
				'max-md:invisible (not max-md:hidden) must keep the scroll box alive so its offset survives the takeover — see planner_resolutions R1'
			).toBe(scrolledOffset);
		});

		test('5: one history entry per takeover session, not per item', async ({ page, kernel }) => {
			await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			const rows = page.locator('[data-item-id]');
			const firstTitle = (await rows.first().locator('p').first().innerText()).trim();
			const secondTitle = (await rows.nth(1).locator('p').first().innerText()).trim();
			expect(secondTitle, 'the fixture needs two distinctly-titled rows for this case to prove anything').not.toBe(
				firstTitle
			);

			await rows.first().click();
			const barTitle = page.locator(BAR_TITLE_SELECTOR);
			await expect(barTitle).toHaveText(firstTitle);

			// Reselecting a second item from inside the already-open
			// takeover must reassign the selection without pushing a
			// second history entry. The row is concealed (visibility:
			// hidden) by design while the takeover is up, and it now sits
			// BEHIND the detail pane's own fixed inset-0 z-20 overlay — a
			// real click (even with { force: true }, which only skips
			// Playwright's own actionability checks, not the browser's
			// real hit-testing) would land on the overlay, not the row.
			// dispatchEvent bypasses hit-testing entirely by firing the
			// event straight at the target node, which is what actually
			// exercises the underlying selectItem/pushState guard this
			// case is about.
			await rows.nth(1).dispatchEvent('click');
			// Proves the reassignment genuinely happened, not merely that
			// the dispatchEvent call didn't throw.
			await expect(barTitle).toHaveText(secondTitle);
			await expect(page.getByRole('button', { name: 'Back to stream' })).toBeVisible();

			// A SINGLE back press must land straight on the plain stream —
			// if the second selection had pushed its own history entry,
			// this one press would instead land back on the first item's
			// still-open takeover.
			await page.goBack();
			await expect(rows.first()).toBeVisible();
			await expect(page.getByRole('button', { name: 'Back to stream' })).toHaveCount(0);
		});

		test('6: toggling a source filter does not close the takeover (backstop)', async ({ page, kernel }) => {
			await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			// Open the takeover BEFORE touching the filter — with nothing
			// filtered yet, the lone chip is still inline in the header's
			// row (selecting a filter later reveals a "Clear filters"
			// control that reflows the row; toggling first, while the
			// chip is guaranteed inline, keeps this case about the
			// takeover/history interaction, not the header's own
			// independent overflow-into-a-popover behaviour).
			await page.locator('[data-item-id]').first().click();
			await expect(page.getByRole('button', { name: 'Back to stream' })).toBeVisible();

			// The filter chip lives inside the header, concealed
			// (visibility: hidden) while the takeover is up by design
			// (D-04) — a real mobile user cannot reach it either, and it
			// now sits behind the detail pane's own fixed inset-0 z-20
			// overlay too. Neither getByRole (Chromium's accessibility
			// tree excludes a visibility:hidden subtree outright) nor a
			// real/{force:true} click (real hit-testing at those screen
			// coordinates would land on the overlay) can reach it —
			// dispatchEvent fires the event straight at the target node,
			// bypassing both problems, and is scoped to the real chip row
			// (not its off-screen aria-hidden measurement clone,
			// WebspaceHeader.svelte's own `measureEl`).
			const chipRow = page.locator('.mt-4.flex.flex-nowrap.items-center.gap-2.overflow-hidden');
			await chipRow.locator('button').filter({ hasText: INSTANCE_DISPLAY_NAME }).dispatchEvent('click');

			// The CORE UI-SPEC assertion this case exists for (see
			// "Composing with existing URL state"): a replaceState-driven
			// filter write must not disturb page.state and must
			// therefore never accidentally close the takeover — see
			// +page.svelte's navigateFilters, which now threads
			// state: page.state through its replaceState goto for
			// exactly this reason (09.1-01-PLAN.md deviation: this was a
			// live bug, found and fixed while writing this case).
			await expect(page.getByRole('button', { name: 'Back to stream' })).toBeVisible();
			expect(page.url()).toContain('sources=');

			await page.goBack();
			await expect(page.getByRole('button', { name: 'Back to stream' })).toHaveCount(0);
			// The filter write landed on the takeover's OWN history entry
			// (replaceState never creates a second entry — see the
			// existing "toggling a chip repeatedly doesn't fill the
			// back-button history" comment on navigateFilters), so a
			// single back press discards the takeover and the filter
			// together, landing on the pre-takeover, pre-filter entry —
			// never a partially-reverted state.
			expect(page.url()).not.toContain('sources=');
		});
	});

	test.describe('at desktop width (1280x900)', () => {
		test.use({ viewport: { width: 1280, height: 900 } });

		test('7: layout is unchanged from the pre-phase desktop layout (D-02)', async ({ page, kernel }) => {
			await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			await page.locator('[data-item-id]').first().click();

			await expect(page.getByRole('button', { name: 'Back to stream' })).toHaveCount(0);
			const streamBox = page.locator(STREAM_SCROLL_SELECTOR);
			const width = await streamBox.evaluate((el) => el.getBoundingClientRect().width);
			expect(width).toBe(480);
		});
	});

	test.describe('at mid-band width (900x900)', () => {
		test.use({ viewport: { width: 900, height: 900 } });

		test('8: the stream pane shrinks proportionally, never to the fixed 480px (D-02)', async ({
			page,
			kernel
		}) => {
			await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			await page.locator('[data-item-id]').first().click();

			const streamBox = page.locator(STREAM_SCROLL_SELECTOR);
			const width = await streamBox.evaluate((el) => el.getBoundingClientRect().width);
			expect(width).toBeGreaterThan(240);
			expect(width).toBeLessThan(480);
			await expect(streamBox).toBeVisible();
		});
	});
});
