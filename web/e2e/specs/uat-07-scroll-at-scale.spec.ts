// Ports 07-UAT.md item 7 into a permanent regression gate.
//
// Verbatim pass condition this spec encodes (07-UAT.md item 7):
//   "With 15+ webspaces and 15+ instances, the switcher, the "+" picker,
//   and Manage sources… all scroll internally (fixed max-height) rather
//   than growing past the viewport."
//
// Every assertion below is a GEOMETRY assertion (scrollHeight vs
// clientHeight, and a bounding-box bottom edge against the viewport
// height) — never a Tailwind class-name assertion. A class assertion
// would still pass even after a later layout change made the element
// render off-screen anyway; the geometry is what this item is actually
// about.
import type { Locator } from '@playwright/test';
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import {
	attachedWebspace,
	mockInstances,
	type FixtureConfigSpec,
	type FixtureWebspaceSpec
} from '../fixtures/config-builder';

const INSTANCE_COUNT = 16;
const WEBSPACE_COUNT = 16;
const CURRENT_WEBSPACE = 'ws-01';

// 16 mock instances total (D-07's mock-only harness). Only the first 8 are
// attached to the CURRENT webspace's own chip row/allowlist — enough on
// their own to overflow the chip row into its "+N more sources" trigger at
// the fixed 800x600 viewport below — leaving the remaining 8 unattached
// (and therefore "available") so the add-source "+" picker's own popover
// has enough rows to overflow too. Attaching all 16 to one webspace would
// leave the picker with nothing available to show (every instance already
// participating), which would make its own overflow assertion vacuous.
const allSources = mockInstances(INSTANCE_COUNT);
const attachedIds = allSources.slice(0, 8).map((s) => s.id);

const attachedMatch: Record<string, Record<string, string[]>> = {};
for (const id of attachedIds) attachedMatch[id] = { labels: ['demo'] };

// The other 15 webspaces exist purely to give the SWITCHER and the Manage
// Sources modal's own webspace list enough rows to overflow — each is a
// D-20 empty shell (keywords/sources/match all explicitly empty, never
// omitted — see uat-06's identical rationale for why `keywords: []` is
// seeded rather than left undefined: it matches what the canonical writer
// always emits on any save, so this fixture's own on-disk shape is never
// itself the thing under test here).
const webspaces: FixtureWebspaceSpec[] = [];
for (let i = 1; i <= WEBSPACE_COUNT; i++) {
	const name = `ws-${String(i).padStart(2, '0')}`;
	webspaces.push(
		name === CURRENT_WEBSPACE
			? { ...attachedWebspace(name, attachedIds, attachedMatch), keywords: [] }
			: { name, keywords: [] }
	);
}

const configSpec: FixtureConfigSpec = { sources: allSources, webspaces };

// An explicit, modest, fixed viewport (800x600) — chosen so the geometry
// assertions below are deterministic rather than dependent on the test
// runner's own default window size, which Playwright does not guarantee
// across environments.
test.use({ configSpec, viewport: { width: 800, height: 600 } });

interface OverflowMetrics {
	scrollHeight: number;
	clientHeight: number;
	bottom: number;
}

async function readOverflowMetrics(locator: Locator): Promise<OverflowMetrics> {
	return locator.evaluate((el) => ({
		scrollHeight: el.scrollHeight,
		clientHeight: el.clientHeight,
		bottom: el.getBoundingClientRect().bottom
	}));
}

test.describe('07-UAT item 7: switcher, add-source picker, and Manage sources all scroll internally at 15+ webspaces/instances', () => {
	test('every scrollable overlay stays internally scrollable and on-screen, and the chip row stays one line', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, attachedIds, { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${CURRENT_WEBSPACE}`);

		// --- 1: the webspace switcher's content is internally scrollable
		// (scrollHeight > clientHeight) AND stays on screen (bounding-box
		// bottom <= viewport height). Neither property alone proves the
		// other: internal scrollability alone does not prove the content
		// stayed on screen, and staying on screen alone does not prove the
		// content was not simply truncated.
		await page.getByRole('button', { name: CURRENT_WEBSPACE }).click();
		const switcherContent = page.locator('[data-slot="dropdown-menu-content"]');
		await expect(switcherContent).toBeVisible();
		const switcherMetrics = await readOverflowMetrics(switcherContent);
		expect(switcherMetrics.scrollHeight).toBeGreaterThan(switcherMetrics.clientHeight);
		expect(switcherMetrics.bottom).toBeLessThanOrEqual(600);

		// --- 2: "Manage sources…" is still reachable inside the scrolled
		// menu — an unreachable escape hatch is the actual user harm this
		// item guards against, not merely "the menu has a scrollbar".
		const manageItem = page.getByRole('menuitem', { name: 'Manage sources…' });
		await manageItem.scrollIntoViewIfNeeded();
		await expect(manageItem).toBeVisible();
		await page.keyboard.press('Escape');
		await expect(switcherContent).toHaveCount(0);

		// --- 3: the add-source picker's own content carries the same two
		// properties.
		await page.getByRole('button', { name: 'Add source' }).click();
		const popoverContent = page.locator('[data-slot="popover-content"]');
		await expect(popoverContent).toBeVisible();
		const popoverMetrics = await readOverflowMetrics(popoverContent);
		expect(popoverMetrics.scrollHeight).toBeGreaterThan(popoverMetrics.clientHeight);
		expect(popoverMetrics.bottom).toBeLessThanOrEqual(600);
		await page.keyboard.press('Escape');
		await expect(popoverContent).toHaveCount(0);

		// --- 4: Manage sources… — both its source-instances list and its
		// webspaces list — carry the same two properties, each on its own
		// independently-scrollable region.
		await page.getByRole('button', { name: CURRENT_WEBSPACE }).click();
		await page.getByRole('menuitem', { name: 'Manage sources…' }).click();
		const manageDialog = page.getByRole('dialog');
		await expect(manageDialog).toBeVisible();
		const manageRegions = manageDialog.locator('.overflow-y-auto');
		await expect(manageRegions).toHaveCount(2);
		for (let i = 0; i < 2; i++) {
			const metrics = await readOverflowMetrics(manageRegions.nth(i));
			expect(metrics.scrollHeight).toBeGreaterThan(metrics.clientHeight);
			expect(metrics.bottom).toBeLessThanOrEqual(600);
		}
		await page.keyboard.press('Escape');
		await expect(manageDialog).toHaveCount(0);

		// --- 5: the chip row itself has not grown past one line at 8
		// attached instances — the overflow trigger showing a hidden-source
		// count is present, and the row's own bounding-box height stays
		// within one chip row's height (SourceChip's own `h-11`, 44px).
		const overflowTrigger = page.getByRole('button', { name: /more sources/ });
		await expect(overflowTrigger).toBeVisible();
		const chipRow = page.locator('.flex.flex-nowrap.items-center.gap-2.overflow-hidden').first();
		const rowHeight = await chipRow.evaluate((el) => el.getBoundingClientRect().height);
		expect(rowHeight).toBeLessThanOrEqual(48);
	});
});
