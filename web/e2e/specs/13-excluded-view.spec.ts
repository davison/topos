// 13-03-PLAN.md Task 3: the excluded-items view toggle round trip, driven
// entirely from the real UI — absent at zero exclusions (D-06, never a
// disabled control), counted once an item is excluded, reuses the stream
// surface wholesale (D-05) when flipped, and auto-flips back to the normal
// stream the instant its last item is un-excluded (E4). Reuses 13-01's
// tracer fixture shape.
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import type { FixtureConfigSpec } from '../fixtures/config-builder';

const WEBSPACE = 'excluded-view';
const SOURCE_ID = 'mock';

const configSpec: FixtureConfigSpec = {
	sources: [{ id: SOURCE_ID, plugin: 'topos-plugin-mock', displayName: 'Mock Source' }],
	webspaces: [{ name: WEBSPACE, keywords: ['demo'] }]
};

test.use({ configSpec });

test.describe('13-03 Task 3: the excluded-items view toggle', () => {
	test('the toggle is absent with zero exclusions, appears counted after one, flips the stream, and Include auto-flips back', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [SOURCE_ID], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const rows = page.locator('[data-item-id]');
		await expect(rows.first()).toBeVisible();
		const rowCountBefore = await rows.count();

		// (1) empty (D-06): the toggle element is fully ABSENT, not merely
		// hidden or disabled, before any exclusion exists.
		await expect(page.getByRole('button', { name: /^Excluded \(/ })).toHaveCount(0);

		// (2) exclude one item via the detail-pane control.
		const firstRowId = await rows.first().getAttribute('data-item-id');
		await rows.first().click();
		await page.getByRole('button', { name: 'Exclude from webspace' }).click();
		await expect(page.locator(`[data-item-id="${firstRowId}"]`)).toHaveCount(0);

		// (3) the toggle appears reading "Excluded (1)".
		const toggle = page.getByRole('button', { name: 'Excluded (1)', exact: true });
		await expect(toggle).toBeVisible();

		// (4) clicking it shows the excluded item as the only row, and the
		// bulk primary label reads "Include".
		await toggle.click();
		await expect(rows).toHaveCount(1);
		await expect(page.locator(`[data-item-id="${firstRowId}"]`)).toBeVisible();

		// (5) select it and click Include.
		await rows.first().click({ modifiers: ['Control'] });
		await expect(page.getByText('1 selected', { exact: true })).toBeVisible();
		const includeButton = page.getByRole('button', { name: 'Include', exact: true });
		await expect(includeButton).toBeVisible();
		await includeButton.click();

		// (6) the view auto-flips back to the normal stream, the item is
		// present there, and the toggle is gone (zero exclusions again).
		await expect(rows).toHaveCount(rowCountBefore);
		await expect(page.locator(`[data-item-id="${firstRowId}"]`)).toBeVisible();
		await expect(page.getByRole('button', { name: /^Excluded \(/ })).toHaveCount(0);
	});
});
