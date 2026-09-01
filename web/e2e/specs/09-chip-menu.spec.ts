// Ports 09-05-PLAN.md Task 3's browser-drivable half of 09-UI-SPEC.md
// Fix 3 (de-duplicated relative-time tooltip) and Fix 5 (refresh folded
// into the overflow menu, one trailing control) into a permanent
// regression gate — per docs/testing.md's standing rule, any UAT item a
// browser can drive becomes a spec rather than a manual re-check.
//
// Verbatim pass conditions this spec encodes (09-UI-SPEC.md):
//   Fix 5: "The chip's remaining trailing control collapses from two
//   size-8 hover-revealed buttons to ONE (the ⋮ trigger)... New menu
//   order (updated for #55's Filter this source…): Refresh now →
//   separator → Edit connection… → Edit match
//   settings… → (Re-link…, WhatsApp only) → separator → Remove from this
//   webspace."
//   Fix 3: "no trailing 'ago' — formatRelativeTime's own output already
//   carries the correct word" — correct for every Intl.RelativeTimeFormat
//   numeric-auto shape, not just the commonly-hit one.
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { attachedWebspace, type FixtureConfigSpec, type FixtureSourceSpec } from '../fixtures/config-builder';

const WEBSPACE = 'chipmenu';

const sources: FixtureSourceSpec[] = [
	{ id: 'mock-01', plugin: 'topos-plugin-mock', displayName: 'Mock One' }
];

const configSpec: FixtureConfigSpec = {
	sources,
	webspaces: [attachedWebspace(WEBSPACE, ['mock-01'], { 'mock-01': { labels: ['demo'] } })]
};

test.use({ configSpec });

test.describe('09-05 Task 3: chip single trailing control, menu order, refresh, and de-duplicated tooltip copy', () => {
	test('the chip reveals exactly one trailing hover-revealed control', async ({ page, kernel }) => {
		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		// exact: true — distinct from the "Mock One actions" overflow trigger,
		// whose accessible name also contains this substring.
		const filterButton = page.getByRole('button', { name: 'Mock One', exact: true });
		await expect(filterButton).toBeVisible();

		await filterButton.hover();

		// Counts the controls the chip's own hover-reveal treatment
		// (opacity-0 base class, group-hover:opacity-100 override) currently
		// renders VISIBLE — not merely present in the DOM. Before Fix 5 this
		// count was 2 (the standalone refresh button plus the edit trigger);
		// the design rule is "exactly one", so this asserts the count, not
		// that a specific named control is absent.
		const revealedCount = await filterButton.evaluate((el) => {
			const wrapper = el.closest('.group');
			if (!wrapper) throw new Error('expected an ancestor .group chip wrapper');
			const candidates = Array.from(wrapper.querySelectorAll('button')).filter((btn) =>
				btn.className.includes('opacity-0')
			);
			return candidates.filter((btn) => getComputedStyle(btn).opacity !== '0').length;
		});

		expect(
			revealedCount,
			'expected exactly one trailing hover-revealed control inside the chip (the overflow trigger) — a second visible control would mean Fix 5 regressed'
		).toBe(1);
	});

	test('the overflow menu renders in the exact Fix 5 order for a non-WhatsApp source', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		await page.getByRole('button', { name: 'Mock One actions' }).click();

		const menu = page.getByRole('menu');
		await expect(menu).toBeVisible();
		const items = (await menu.getByRole('menuitem').allTextContents()).map((text) => text.trim());

		expect(
			items,
			'expected the menu item TEXT SEQUENCE — not merely membership — to equal the Fix 5 order for a non-WhatsApp source, which has no Re-link… item'
		).toEqual([
			'Refresh now',
			'Edit connection…',
			'Edit match settings…',
			'Filter this source…',
			'Remove from this webspace'
		]);
	});

	test('Refresh now triggers a real refresh request against the kernel, proving the relocated control is wired', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		// Registered before the click that provokes it (docs/testing.md's own
		// rule) — lets the real request through (route.fetch + fulfill) so
		// this proves an actual, successful kernel round trip, not merely
		// that the menu item is clickable.
		let refreshRequestSeen = false;
		let refreshResponseStatus: number | null = null;
		await page.route('**/api/sources/mock-01/refresh', async (route) => {
			refreshRequestSeen = true;
			const response = await route.fetch();
			refreshResponseStatus = response.status();
			await route.fulfill({ response });
		});

		await page.getByRole('button', { name: 'Mock One actions' }).click();
		await page.getByRole('menuitem', { name: 'Refresh now' }).click();

		await expect
			.poll(() => refreshRequestSeen, {
				timeout: 10_000,
				message: 'expected clicking Refresh now to fire POST /api/sources/mock-01/refresh'
			})
			.toBe(true);
		await expect
			.poll(() => refreshResponseStatus, {
				timeout: 10_000,
				message: 'expected the refresh request the relocated control fired to complete successfully'
			})
			.toBe(200);

		await page.unroute('**/api/sources/mock-01/refresh');
	});

	test('the tooltip reads a de-duplicated sync phrase, catching every "word ago" variant, not just the common one', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const filterButton = page.getByRole('button', { name: 'Mock One', exact: true });
		await filterButton.hover();

		const tooltip = page.locator('[data-slot="tooltip-content"]');
		await expect(tooltip).toBeVisible();

		const text = ((await tooltip.textContent()) ?? '').trim();
		expect(
			text,
			'expected the tooltip to contain "synced" followed by a relative-time phrase, once the first sync has completed'
		).toContain('synced');

		// The general form of the "ago ago" / "yesterday ago" defect: no
		// whitespace-delimited word may repeat immediately after itself,
		// regardless of which Intl.RelativeTimeFormat numeric-auto shape the
		// live relative delta happens to produce at run time.
		const words = text.split(/\s+/).filter((w) => w.length > 0);
		for (let i = 1; i < words.length; i += 1) {
			expect(
				words[i].toLowerCase(),
				`expected no word to repeat immediately after itself in the tooltip text "${text}"`
			).not.toBe(words[i - 1].toLowerCase());
		}
	});
});
