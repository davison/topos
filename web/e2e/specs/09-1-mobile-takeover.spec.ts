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

// FOUR mock instances (16 fixed items total, every one tagged "demo" — see
// plugins/mock/plugin.go's mockItems) — checkpoint fix (issue 2): the
// header-auto-collapse feature reclaims real layout height when it
// collapses, which fewer items are no longer guaranteed to overflow once
// that height is reclaimed (case 4's own scroll-preservation assertion
// needs a genuine, persistent overflow to prove anything). A webspace
// matching on the shared "demo" keyword picks up all sixteen without an
// explicit per-instance match block.
//
// 09.1-03-PLAN.md deviation (Rule 3 — blocking, found live running this
// file as this plan's own required regression check): bumped from TWO
// instances (eight items) to FOUR (sixteen items) after 09.1-03 shipped
// the compact 60px stream row below 768px (D-06). At 390x844 the original
// eight items summed to 480px of content against a measured ~586px
// scrollable area — no overflow at all, so case 4's own "the fixture
// stream must be tall enough to scroll" guard failed outright (not a
// flake — a real, reproducible zero). Desktop's unchanged 152px rows never
// had this problem; only the max-md: 768px band did. Sixteen items (960px)
// clears the measured area with comfortable margin, including once the
// header-collapse feature (case 10) reclaims further height.
const sources = mockInstances(4);
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
			// A modest, FIXED offset — deliberately not the exact maximum.
			// The checkpoint's header-auto-collapse feature (issue 2)
			// reclaims real layout height when it collapses, which
			// legitimately SHRINKS the maximum scrollable range (more of
			// the tail content becomes visible without scrolling) — the
			// "maximum" is a moving target once the header can resize the
			// very viewport being scrolled, so pinning this case to it
			// would be testing an edge that shifts out from under itself
			// for a real, correct reason, not a regression. A fixed
			// mid-list offset, comfortably far from either the top or the
			// (now variable) bottom, proves the same D-01 guarantee
			// without depending on that boundary.
			await scrollBox.evaluate((el) => {
				el.scrollTop = 200;
			});
			const scrolledOffset = await scrollBox.evaluate((el) => el.scrollTop);
			expect(
				scrolledOffset,
				'the fixture stream must be tall enough to scroll at 390x844 for this case to prove anything'
			).toBeGreaterThan(0);

			// Click whichever row is ACTUALLY centred in the current
			// viewport, found via elementFromPoint — not .first()/.last(),
			// both of which risk sitting outside the scrolled view and
			// triggering Playwright's own scroll-into-view actionability
			// step before the click, which would silently move scrollTop
			// out from under this assertion before the takeover even
			// opens.
			const visibleItemId = await page.evaluate((selector) => {
				const box = document.querySelector(selector);
				if (!box) return null;
				const rect = box.getBoundingClientRect();
				const el = document.elementFromPoint(rect.left + rect.width / 2, rect.top + rect.height / 2);
				return el?.closest('[data-item-id]')?.getAttribute('data-item-id') ?? null;
			}, STREAM_SCROLL_SELECTOR);
			expect(visibleItemId, 'expected a row to be visible at the scrolled offset').not.toBeNull();
			await page.locator(`[data-item-id="${visibleItemId}"]`).click();
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

			// .first()/.last() (not .first()/.nth(1)): with 2+ mock
			// instances, the SAME fixed item (identical title AND
			// timestamp) exists once per instance, so adjacent rows can
			// legitimately tie on timestamp and sort next to each other —
			// .first() and .last() sit at opposite ends of the sorted
			// stream and are guaranteed distinct titles (item 1's "Welcome
			// to the mock source" vs item 4's "Shopping list", per
			// plugins/mock/plugin.go's fixed corpus).
			const rows = page.locator('[data-item-id]');
			const firstTitle = (await rows.first().locator('p').first().innerText()).trim();
			const secondTitle = (await rows.last().locator('p').first().innerText()).trim();
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
			await rows.last().dispatchEvent('click');
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

			await page.locator('[data-item-id]').first().click();
			await expect(page.getByRole('button', { name: 'Back to stream' })).toBeVisible();

			// The filter chip lives inside the header, concealed
			// (visibility: hidden) while the takeover is up by design
			// (D-04) — a real mobile user cannot reach it either, and it
			// now sits behind the detail pane's own fixed inset-0 z-20
			// overlay too. With two mock instances (fixture bumped for
			// case 4's own needs — see the module comment), neither chip
			// fits in the row's inline slot at 390px even before any
			// filter is applied — both live behind the "N more sources"
			// overflow trigger. getByRole cannot reach either the trigger
			// or a chip (Chromium's accessibility tree excludes a
			// visibility:hidden subtree outright); an attribute-selector
			// CSS locator can, since it queries the DOM directly, not the
			// accessibility tree. dispatchEvent opens the popover the
			// same way (bypassing the overlay's real hit-testing that a
			// real/{force:true} click would fail); PopoverContent then
			// portals to document.body, so the chip inside it is a
			// normal, visible, real-clickable element once open.
			await page.locator('button[aria-label*="more sources"]').dispatchEvent('click');
			const popoverChip = page
				.locator('[data-slot="popover-content"] button[aria-pressed]')
				.filter({ hasText: INSTANCE_DISPLAY_NAME });
			await popoverChip.click();

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

		test('9: the page never overflows horizontally, in either the stream or the takeover state (checkpoint fix, issue 1)', async ({
			page,
			kernel
		}) => {
			await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			// Root cause: WebspaceHeader.svelte's off-screen chip-width
			// measurement clone (`measureEl`) is `position: absolute` with
			// no `position: relative` ancestor anywhere in this app, so
			// its containing block was the initial containing block, not
			// a nearby DOM ancestor — its full, deliberately-unclipped
			// natural width leaked into document.documentElement.
			// scrollWidth. Two mock instances (this file's own fixture)
			// reproduce it reliably; a single instance's one chip does
			// not always exceed 390px on its own.
			const streamMetrics = await page.evaluate(() => ({
				scrollWidth: document.documentElement.scrollWidth,
				clientWidth: document.documentElement.clientWidth
			}));
			expect(streamMetrics.scrollWidth, 'stream state must not overflow horizontally').toBeLessThanOrEqual(
				streamMetrics.clientWidth
			);

			await page.locator('[data-item-id]').first().click();
			await expect(page.getByRole('button', { name: 'Back to stream' })).toBeVisible();

			const takeoverMetrics = await page.evaluate(() => ({
				scrollWidth: document.documentElement.scrollWidth,
				clientWidth: document.documentElement.clientWidth
			}));
			expect(
				takeoverMetrics.scrollWidth,
				'takeover state must not overflow horizontally — the fixed inset-0 overlay sizes itself against the layout viewport, so a stream-state overflow bleeds into real takeover content, not merely empty space'
			).toBeLessThanOrEqual(takeoverMetrics.clientWidth);

			// html/body's own overflow-x: hidden (app.css, the same
			// checkpoint fix) is the backstop: even if some future
			// element reintroduces oversized content, a real scroll
			// gesture must never be able to move the page horizontally.
			await page.mouse.move(195, 400);
			await page.mouse.wheel(300, 0);
			await page.waitForTimeout(100);
			const scrollXAfterWheel = await page.evaluate(() => window.scrollX);
			expect(scrollXAfterWheel, 'a horizontal wheel gesture must never scroll the page').toBe(0);
		});

		test('10: the header collapses on scroll down and re-inflates the instant scroll-up begins (checkpoint fix, issue 2)', async ({
			page,
			kernel
		}) => {
			await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			const header = page.locator('header').first();
			const scrollBox = page.locator(STREAM_SCROLL_SELECTOR);

			const expandedHeight = await header.evaluate((el) => el.getBoundingClientRect().height);

			await scrollBox.evaluate((el) => {
				el.scrollTop = 200;
				el.dispatchEvent(new Event('scroll'));
			});
			await expect
				.poll(() => header.evaluate((el) => el.getBoundingClientRect().height))
				.toBeLessThan(expandedHeight);
			const collapsedHeight = await header.evaluate((el) => el.getBoundingClientRect().height);

			// Wait out +page.svelte's own post-collapse re-entrancy guard
			// (HEADER_COLLAPSE_TRANSITION_MS, 200ms) before dispatching the
			// next scroll event — a synthetic 'scroll' event fired inside
			// that window is deliberately ignored (it exists to suppress
			// the browser's OWN reflow-driven scroll events, not to add
			// artificial latency to a real user's next gesture; a real
			// scroll-up gesture arriving that quickly after a collapse is
			// not a realistic input pattern this guard needs to serve).
			await page.waitForTimeout(250);

			await scrollBox.evaluate((el) => {
				el.scrollTop = 150;
				el.dispatchEvent(new Event('scroll'));
			});
			await expect
				.poll(() => header.evaluate((el) => el.getBoundingClientRect().height))
				.toBe(expandedHeight);

			expect(
				collapsedHeight,
				'the collapsed header must be meaningfully shorter, not merely a rounding difference'
			).toBeLessThan(expandedHeight - 40);
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

	// Checkpoint fixes (issues 1 and 2): the human-verify report's own
	// reproduction viewport — a phone rotated to landscape, still well
	// below the 1024px desktop breakpoint, where the side-by-side layout
	// applies (D-02) but the full header previously left "virtually no
	// stream or details pane content visible".
	test.describe('at landscape-phone width (844x390)', () => {
		test.use({ viewport: { width: 844, height: 390 } });

		test('11: no horizontal overflow, and the header reclaims real height on scroll (checkpoint fixes 1 and 2)', async ({
			page,
			kernel
		}) => {
			await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

			const metrics = await page.evaluate(() => ({
				scrollWidth: document.documentElement.scrollWidth,
				clientWidth: document.documentElement.clientWidth
			}));
			expect(metrics.scrollWidth).toBeLessThanOrEqual(metrics.clientWidth);

			// Side-by-side at this width (D-02) — an item is already
			// selected here, distinct from the phone-portrait describe
			// block above (which is deliberately mobile-takeover-only).
			await page.locator('[data-item-id]').first().click();
			await expect(page.locator(STREAM_SCROLL_SELECTOR)).toBeVisible();

			const header = page.locator('header').first();
			const expandedHeight = await header.evaluate((el) => el.getBoundingClientRect().height);
			// At a 390px-tall viewport, an uncollapsed header eating a
			// third or more of it is exactly the "virtually no stream or
			// details pane content visible" complaint this fix answers.
			expect(expandedHeight).toBeLessThan(390);

			const scrollBox = page.locator(STREAM_SCROLL_SELECTOR);
			await scrollBox.evaluate((el) => {
				el.scrollTop = 150;
				el.dispatchEvent(new Event('scroll'));
			});
			await expect
				.poll(() => header.evaluate((el) => el.getBoundingClientRect().height))
				.toBeLessThan(expandedHeight - 40);
		});
	});
});
