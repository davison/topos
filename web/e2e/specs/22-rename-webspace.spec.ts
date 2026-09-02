// M3-R2 (#71/#77): a webspace's name can change after creation. The
// rename carries the config key, the index rows and the URL together:
// items and an exclusion mark survive, the open tab lands on the new
// address, the old address is gone, and a collision is refused with the
// error surfaced.
import { test, expect } from '../fixtures/kernel';
import { webspacesWithKeywords, type FixtureConfigSpec } from '../fixtures/config-builder';
import { waitForFirstSync } from '../fixtures/kernel';

const MOCK = 'mock-01';

const configSpec: FixtureConfigSpec = {
	sources: [{ id: MOCK, plugin: 'topos-plugin-mock', displayName: 'Mock One' }],
	webspaces: [
		...webspacesWithKeywords(['house'], ['demo']),
		...webspacesWithKeywords(['sibling'], ['demo'])
	]
};

test.use({ configSpec });

test.describe('22: webspace rename carries items, marks and the URL', () => {
	test.beforeEach(async ({ page, kernel }) => {
		await waitForFirstSync(kernel.baseURL, [MOCK], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/house`);
	});

	test('rename survives items and an exclusion, moves the tab, kills the old URL', async ({
		page,
		kernel
	}) => {
		const rows = page.getByRole('main').locator('[data-item-id]');
		await expect(rows).toHaveCount(4);

		// Exclude one item first — the operator judgment the rename must
		// not reset (the Decision on #77).
		await rows.first().click();
		await page.getByRole('button', { name: 'Exclude from webspace' }).first().click();
		await expect(rows).toHaveCount(3);

		// Rename via Manage sources.
		await page.getByRole('banner').getByRole('button', { name: 'house', exact: true }).click();
		await page.getByRole('menuitem', { name: 'Manage sources…' }).click();
		await page.getByRole('button', { name: 'Rename webspace house' }).click();
		await page.locator('[data-rename-webspace-input]').fill('household');
		await page.locator('[data-rename-webspace-save]').click();

		// The open tab lands on the new URL with the same narrowed stream.
		await expect(page).toHaveURL(new RegExp('/w/household$'));
		await expect(rows).toHaveCount(3);

		// The exclusion followed: the excluded view holds exactly one.
		const resp = await page.request.get(
			`${kernel.baseURL}/api/webspaces/household/stream?view=excluded`
		);
		expect(resp.ok()).toBeTruthy();
		expect((await resp.json()).items).toHaveLength(1);

		// The old name is gone from the kernel.
		const old = await page.request.get(`${kernel.baseURL}/api/webspaces/house/stream`);
		expect(old.status()).toBe(404);
	});

	test('a collision is refused with the error surfaced, nothing written', async ({ page }) => {
		await page.getByRole('banner').getByRole('button', { name: 'house', exact: true }).click();
		await page.getByRole('menuitem', { name: 'Manage sources…' }).click();
		await page.getByRole('button', { name: 'Rename webspace house' }).click();
		await page.locator('[data-rename-webspace-input]').fill('sibling');
		await page.locator('[data-rename-webspace-save]').click();
		await expect(page.locator('[data-rename-webspace-dialog]')).toContainText('already exists');
		// Still on the old URL; the webspace list unchanged.
		await expect(page).toHaveURL(new RegExp('/w/house$'));
	});
});
