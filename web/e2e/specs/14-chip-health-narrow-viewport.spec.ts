// 14-06-PLAN.md Task 1 (gap G-14-2, 14-UAT.md Test 2): chip health detail
// must be REACHABLE at a mobile-width viewport. The debug session
// (.planning/debug/popover-hidden-narrow-viewport.md) proved the popover
// machinery was never the defect — it wraps at 320px, flips and shifts
// correctly down to 375px whenever a trigger exists, cross-engine. The
// failure is trigger disappearance: visibleChipCount floors at zero, so at
// ≤~400px a long-named chip is relegated wholesale into the "+N" pill and
// the row contains NO chip to hover. A regression here will therefore
// present as a missing TRIGGER (case 1's locator), not a missing popover.
//
// Reproduction parameters mirror the debug session's Experiment 2 exactly:
// display name "Google Drive Personal Archive" (button capped at max-w-48/
// 192px, whole chip ~230px), relegated at 400/375, in-row again at 440.
// Health state is fabricated via GET /api/sources route interception —
// 12-tooltip-precedence.spec.ts's established pattern for states the
// hermetic fixture plugins cannot produce on demand (external tier +
// unreachable + stale sync, the exact UAT Test 2 scenario).
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import {
	attachedWebspace,
	mockInstances,
	type FixtureConfigSpec,
	type FixtureSourceSpec
} from '../fixtures/config-builder';

const LONG_WEBSPACE = 'chip-health-long';
const MULTI_WEBSPACE = 'chip-health-multi';
const LONG_ID = 'gdrive-long';
const LONG_NAME = 'Google Drive Personal Archive';

// Must stay in step with WebspaceHeader.svelte's MIN_INLINE_CHIP_PX (the
// named constant Task 2 introduces beside CHIP_ROW_GAP_PX — 88px, the
// chip's live-measured non-text furniture). Case 3 below asserts the
// forced chip's rendered box is at least this wide, so a drift between
// the two constants fails loudly here rather than silently rendering a
// broken sliver.
const MIN_INLINE_CHIP_PX = 88;

const mocks = mockInstances(6);
const longSource: FixtureSourceSpec = {
	id: LONG_ID,
	plugin: 'topos-plugin-mock',
	displayName: LONG_NAME
};

// Both an explicit `sources` allowlist AND a per-instance match block are
// required for the chip to render at all — the allowlist gate and the
// match-input gate are two separate checks (12-tooltip-precedence.spec.ts).
const configSpec: FixtureConfigSpec = {
	sources: [...mocks, longSource],
	webspaces: [
		attachedWebspace(LONG_WEBSPACE, [LONG_ID], { [LONG_ID]: { labels: ['demo'] } }),
		attachedWebspace(
			MULTI_WEBSPACE,
			mocks.map((m) => m.id),
			Object.fromEntries(mocks.map((m) => [m.id, { labels: ['demo'] }]))
		)
	]
};

test.use({ configSpec });

const ALL_IDS = [...mocks.map((m) => m.id), LONG_ID];

// 12-zero-match-diagnostic.spec.ts's house helper for regex-matching a
// literal sentence inside toHaveAccessibleDescription.
function escapeRegExp(literal: string): string {
	return literal.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

// A real-shaped GET /api/sources body (docs/api.md), identity fields
// mirroring the long-named fixture instance exactly. External tier +
// unreachable + stale sync + a deliberately long last_error: the composed
// health sentence ends in the untrusted-external-plugin clause and is long
// enough that legibility at 375px requires wrapping (case 2's subject).
function fabricatedLongSourceBody() {
	return {
		schema_version: 1,
		sources: [
			{
				name: LONG_ID,
				source_type: 'mock',
				display_name: LONG_NAME,
				plugin: 'topos-plugin-mock',
				tier: 'external',
				reachable: false,
				syncing: false,
				last_status: 'ok',
				last_sync_unix: 1785000000,
				last_error:
					'dial tcp 192.0.2.17:443: connect: connection timed out after repeated retries against the remote endpoint',
				last_notice: ''
			}
		]
	};
}

async function interceptLongSource(
	page: import('@playwright/test').Page,
	baseURL: string
): Promise<void> {
	await page.route(`${baseURL}/api/sources`, (route) =>
		route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify(fabricatedLongSourceBody())
		})
	);
}

test.describe('14-06: chip health detail reachable at a narrow viewport (G-14-2)', () => {
	test.describe('at phone width (375x812)', () => {
		test.use({ viewport: { width: 375, height: 812 } });

		test('1: the long-named chip stays in the row and its health popover opens on hover', async ({
			page,
			kernel
		}) => {
			await waitForFirstSync(kernel.baseURL, ALL_IDS, { logs: kernel.logs });
			await interceptLongSource(page, kernel.baseURL);
			await page.goto(`${kernel.baseURL}/w/${LONG_WEBSPACE}`);

			// Page-level locators, per house pattern (09-1-*, 12-*): the header
			// element is nested inside <main>, so it exposes no banner role.
			const chip = page.getByRole('button', { name: LONG_NAME, exact: true });
			await expect(
				chip,
				'G-14-2: the chip must be IN THE ROW at 375px — its absence (only a "+N" pill) is the diagnosed trigger disappearance'
			).toBeVisible();

			// The chip must be the row's own, not a clone inside an open
			// popover — no popover content is open at rest.
			await expect(page.locator('[data-slot="popover-content"]')).toHaveCount(0);

			await chip.hover();
			const tooltip = page.locator('[data-slot="tooltip-content"]').filter({ visible: true });
			await expect(tooltip, 'hovering the in-row chip must open the health popover').toBeVisible();
			await expect(tooltip).toContainText(LONG_NAME);
			await expect(
				tooltip,
				'the sentence must carry the untrusted-external-plugin clause — the UAT scenario is an external untrusted plugin with a stale sync'
			).toContainText('untrusted external plugin');

			await page.unroute(`${kernel.baseURL}/api/sources`);
		});

		test('2: the opened popover is legible — inside the viewport, capped at 320px, wrapped', async ({
			page,
			kernel
		}) => {
			await waitForFirstSync(kernel.baseURL, ALL_IDS, { logs: kernel.logs });
			await interceptLongSource(page, kernel.baseURL);
			await page.goto(`${kernel.baseURL}/w/${LONG_WEBSPACE}`);

			const chip = page.getByRole('button', { name: LONG_NAME, exact: true });
			await expect(chip).toBeVisible();
			await chip.hover();

			const tooltip = page.locator('[data-slot="tooltip-content"]').filter({ visible: true });
			await expect(tooltip).toBeVisible();
			const box = await tooltip.boundingBox();
			expect(box, 'the visible tooltip must have a bounding box').not.toBeNull();
			if (box === null) return;

			expect(box.x, `popover left edge at ${box.x}px must not be clipped left of the viewport`).toBeGreaterThanOrEqual(0);
			expect(
				box.x + box.width,
				`popover right edge at ${box.x + box.width}px must stay inside the 375px viewport`
			).toBeLessThanOrEqual(375);
			expect(box.width, `popover width ${box.width}px must respect the max-w-xs (320px) cap`).toBeLessThanOrEqual(320);
			// A single text line renders ~19-24px tall (debug session: the
			// same sentence wrapped to 4 lines at 76px). Anything over 30px
			// proves the sentence WRAPPED rather than being clipped to one
			// overflowing line.
			expect(
				box.height,
				`popover height ${box.height}px must exceed a single text line — the long sentence must WRAP, not clip`
			).toBeGreaterThan(30);

			await page.unroute(`${kernel.baseURL}/api/sources`);
		});

		test('3: forcing the chip inline trades away no other affordance', async ({ page, kernel }) => {
			await waitForFirstSync(kernel.baseURL, ALL_IDS, { logs: kernel.logs });
			await interceptLongSource(page, kernel.baseURL);
			await page.goto(`${kernel.baseURL}/w/${LONG_WEBSPACE}`);

			// Page-level locators, per house pattern (09-1-*, 12-*): the header
			// element is nested inside <main>, so it exposes no banner role.
			const chip = page.getByRole('button', { name: LONG_NAME, exact: true });
			await expect(chip).toBeVisible();

			await expect(
				page.getByRole('button', { name: 'Refresh all' }),
				'"Refresh all" must survive the forced inline chip at 375px'
			).toBeVisible();
			await expect(
				page.getByRole('button', { name: 'Add source' }),
				'the add-source "+" trigger must survive the forced inline chip at 375px'
			).toBeVisible();

			// The chip wrapper (the button's parent group) must be at least
			// the pinned minimum — a narrower box means the floor fired
			// without the budget to seat a legible chip.
			const chipBox = await chip.boundingBox();
			expect(chipBox, 'the in-row chip must have a bounding box').not.toBeNull();
			if (chipBox === null) return;
			expect(
				chipBox.width,
				`the forced chip's rendered width ${chipBox.width}px must be at least MIN_INLINE_CHIP_PX (${MIN_INLINE_CHIP_PX}px, in step with WebspaceHeader.svelte)`
			).toBeGreaterThanOrEqual(MIN_INLINE_CHIP_PX);

			await page.unroute(`${kernel.baseURL}/api/sources`);
		});

		test('4: exactly ONE chip is forced with six instances — overflow still owns the rest', async ({
			page,
			kernel
		}) => {
			await waitForFirstSync(kernel.baseURL, ALL_IDS, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${MULTI_WEBSPACE}`);

			// Anchored pattern: matches the chip filter buttons ("Mock 01")
			// but not the "... actions" dropdown triggers. The invisible
			// measurement clones are excluded from getByRole automatically —
			// their container is aria-hidden and visibility-hidden.
			const chips = page.getByRole('button', { name: /^Mock \d{2}$/ });
			await expect(
				chips,
				'exactly one chip must be forced inline at 375px — zero re-opens G-14-2, more than one breaks the Phase 6 overflow design (UI-07)'
			).toHaveCount(1);

			await expect(
				page.getByRole('button', { name: '5 more sources' }),
				'the remaining five instances must still relegate to the overflow popover'
			).toBeVisible();
		});

		test('5: the locked option-b accessible-description surface is intact', async ({
			page,
			kernel
		}) => {
			await waitForFirstSync(kernel.baseURL, ALL_IDS, { logs: kernel.logs });
			await interceptLongSource(page, kernel.baseURL);
			await page.goto(`${kernel.baseURL}/w/${LONG_WEBSPACE}`);

			const chip = page.getByRole('button', { name: LONG_NAME, exact: true });
			await expect(chip).toBeVisible();

			// 14-02-SUMMARY.md's option-b record: the health sentence is the
			// button's accessible DESCRIPTION (sr-only span + aria-describedby),
			// never its name, and no browser-native tooltip attribute returns.
			await expect(chip).toHaveAccessibleDescription(
				new RegExp(escapeRegExp('untrusted external plugin'))
			);
			await expect(chip).not.toHaveAttribute('title', /./);

			await page.unroute(`${kernel.baseURL}/api/sources`);
		});
	});

	test.describe('desktop parity (1280x900)', () => {
		test.use({ viewport: { width: 1280, height: 900 } });

		test('6: at desktop width the floor changes nothing — multiple chips render inline', async ({
			page,
			kernel
		}) => {
			await waitForFirstSync(kernel.baseURL, ALL_IDS, { logs: kernel.logs });
			await page.goto(`${kernel.baseURL}/w/${MULTI_WEBSPACE}`);

			const chips = page.getByRole('button', { name: /^Mock \d{2}$/ });
			// .count() does not auto-wait — anchor on the first chip becoming
			// visible before counting, then poll the count itself.
			await expect(chips.first()).toBeVisible();
			await expect
				.poll(() => chips.count(), {
					message:
						'at 1280px the six-instance row must show more than one chip inline — the floor only ever raises a zero'
				})
				.toBeGreaterThan(1);
		});
	});
});
