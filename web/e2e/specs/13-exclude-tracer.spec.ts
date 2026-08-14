// 13-01-PLAN.md Task 1: the per-item curation architecture proven end to
// end on the thinnest possible path — one item's Exclude click, through
// the detail pane, the kernel's marks write path, and the shared
// index.Store filter, and back out again as the item disappearing from
// the stream. Follows 12-filesystem-tracer.spec.ts's shape: a hermetic
// kernel fixture, the default topos-plugin-mock reference plugin (no
// pluginBinaries override needed — every mock item carries a "demo"
// label, so the webspace's plain keywords fallback matches all of them),
// asserted via the stream's own rendered rows, never a network intercept.
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import type { FixtureConfigSpec } from '../fixtures/config-builder';

const WEBSPACE = 'exclude-tracer';
const SOURCE_ID = 'mock';

const configSpec: FixtureConfigSpec = {
	sources: [{ id: SOURCE_ID, plugin: 'topos-plugin-mock', displayName: 'Mock Source' }],
	webspaces: [{ name: WEBSPACE, keywords: ['demo'] }]
};

test.use({ configSpec });

test.describe('13-01 Task 1: exclude one item from one webspace — one path only', () => {
	test('clicking Exclude in the detail pane removes the row from the stream', async ({ page, kernel }) => {
		await waitForFirstSync(kernel.baseURL, [SOURCE_ID], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const rows = page.locator('[data-item-id]');
		await expect(rows.first()).toBeVisible();
		const rowCountBefore = await rows.count();
		expect(rowCountBefore).toBeGreaterThan(0);
		const firstRowId = await rows.first().getAttribute('data-item-id');
		expect(firstRowId).toBeTruthy();

		await rows.first().click();

		const excludeButton = page.getByRole('button', { name: 'Exclude from webspace' });
		await expect(excludeButton).toBeVisible();
		await excludeButton.click();

		// The detail pane closes (the excluded item can no longer be shown
		// from the normal stream, D-03) and the row is gone on the next
		// render — asserted against the DOM, never a network intercept.
		await expect(page.locator(`[data-item-id="${firstRowId}"]`)).toHaveCount(0);
		await expect(rows).toHaveCount(rowCountBefore - 1);
	});

	test('the mark is a row in the kernel index, not config — it survives a stream refetch', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [SOURCE_ID], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const rows = page.locator('[data-item-id]');
		await expect(rows.first()).toBeVisible();
		const firstRowId = await rows.first().getAttribute('data-item-id');

		await rows.first().click();
		await page.getByRole('button', { name: 'Exclude from webspace' }).click();
		await expect(page.locator(`[data-item-id="${firstRowId}"]`)).toHaveCount(0);

		// A second, independent GET (not the SPA's own state) proves the
		// exclusion is server-side — a kernel index row, never a
		// client-only filter.
		const streamRes = await fetch(`${kernel.baseURL}/api/webspaces/${WEBSPACE}/stream`);
		expect(streamRes.ok).toBe(true);
		const stream = (await streamRes.json()) as { items: Array<{ id: string }> };
		expect(stream.items.some((it) => it.id === firstRowId)).toBe(false);
	});
});
