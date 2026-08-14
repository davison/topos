// 13-03-PLAN.md Task 3: the multi-select + bulk-exclude round trip, driven
// entirely from the real UI against a real booted kernel — ctrl/shift-click
// selection, the floating action bar's live count, the D-01 rule that only
// Esc and the bar's own Clear empty the selection (a plain click on an
// unrelated row must NOT), and Exclude + Undo. Reuses 13-01's tracer fixture
// shape (one kernel per spec file, the default topos-plugin-mock reference
// plugin — every mock item carries a "demo" label, so the webspace's plain
// keywords fallback matches all four of them).
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import type { FixtureConfigSpec } from '../fixtures/config-builder';

const WEBSPACE = 'bulk-exclude';
const SOURCE_ID = 'mock';

const configSpec: FixtureConfigSpec = {
	sources: [{ id: SOURCE_ID, plugin: 'topos-plugin-mock', displayName: 'Mock Source' }],
	webspaces: [{ name: WEBSPACE, keywords: ['demo'] }]
};

test.use({ configSpec });

test.describe('13-03 Task 3: multi-select and the floating bulk-action bar', () => {
	test('ctrl/shift-click select a range, a plain click never clears it, Esc clears, and Exclude + Undo round-trip', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [SOURCE_ID], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const rows = page.locator('[data-item-id]');
		await expect(rows.first()).toBeVisible();
		// The mock plugin's fixed corpus is exactly four items, all
		// "demo"-labelled — see plugins/mock/plugin.go's mockItems doc
		// comment.
		await expect(rows).toHaveCount(4);

		// (1) empty (D-01): no action bar at rest.
		await expect(page.getByText(/\d+ selected/)).toHaveCount(0);

		// (2) ctrl-click row 1: the bar appears reading "1 selected".
		await rows.nth(0).click({ modifiers: ['Control'] });
		await expect(page.getByText('1 selected', { exact: true })).toBeVisible();

		// (3) shift-click row 3: the bar reads "3 selected" — the
		// contiguous range from row 1 through row 3, inclusive.
		await rows.nth(2).click({ modifiers: ['Shift'] });
		await expect(page.getByText('3 selected', { exact: true })).toBeVisible();

		// (4) a plain click on row 4 (outside the selected range) opens the
		// detail pane and does NOT clear the existing bulk selection — a
		// plain click is not a clearing action (D-01).
		await rows.nth(3).click();
		await expect(page.getByRole('button', { name: 'Exclude from webspace' })).toBeVisible();
		await expect(page.getByText('3 selected', { exact: true })).toBeVisible();

		// (5) Esc empties the selection; the bar disappears.
		await page.keyboard.press('Escape');
		await expect(page.getByText(/\d+ selected/)).toHaveCount(0);

		// (6) re-select two rows, click Exclude, and assert both rows leave
		// the stream and the toast reads "Excluded 2 items".
		await rows.nth(0).click({ modifiers: ['Control'] });
		await rows.nth(1).click({ modifiers: ['Control'] });
		await expect(page.getByText('2 selected', { exact: true })).toBeVisible();

		const excludedIds = (await Promise.all([
			rows.nth(0).getAttribute('data-item-id'),
			rows.nth(1).getAttribute('data-item-id')
		])) as string[];

		await page.getByRole('button', { name: 'Exclude', exact: true }).click();

		await expect(page.getByText('Excluded 2 items', { exact: true })).toBeVisible();
		await expect(rows).toHaveCount(2);
		for (const id of excludedIds) {
			await expect(page.locator(`[data-item-id="${id}"]`)).toHaveCount(0);
		}

		// (7) click the toast's Undo and assert both rows return.
		await page.getByRole('button', { name: 'Undo' }).click();

		await expect(rows).toHaveCount(4);
		for (const id of excludedIds) {
			await expect(page.locator(`[data-item-id="${id}"]`)).toHaveCount(1);
		}
	});
});
