// Ports 07-UAT.md item 1 (G-07-1) into a permanent regression gate.
//
// Verbatim pass condition this spec encodes (07-UAT.md item 1):
//   "make dev; + New webspace → name → submit: modal closes, navigates to
//   /w/<name>, stream shows "Nothing here yet" immediately with no error
//   state and no Retry click (round-2 test 1 re-run against the fix). Then
//   an unconfigured name typed into the address bar renders the
//   not-configured copy naming it — not the outage copy — with the
//   switcher still usable."
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { mockInstances, webspacesWithKeywords, type FixtureConfigSpec } from '../fixtures/config-builder';
import { readConfigToml } from '../fixtures/toml';

const configSpec: FixtureConfigSpec = {
	sources: mockInstances(1),
	webspaces: webspacesWithKeywords(['existing'], ['demo'])
};

test.use({ configSpec });

test.describe('07-UAT item 1 (G-07-1): create-webspace lands empty; an unconfigured name is not an outage', () => {
	test('creating a webspace from the switcher lands on an empty stream, never an error state', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });

		// --- Part A: create lands on an empty stream, not an error --------
		await page.goto(`${kernel.baseURL}/w/existing`);

		await page.getByRole('button', { name: 'existing' }).click();
		await page.getByRole('menuitem', { name: '+ New webspace' }).click();

		await expect(page.getByRole('heading', { name: 'New webspace' })).toBeVisible();
		await page.getByLabel('Name').fill('fresh-project');
		await page.getByRole('dialog').getByRole('button', { name: 'Create webspace' }).click();

		// The modal closes.
		await expect(page.getByRole('heading', { name: 'New webspace' })).not.toBeVisible();

		// Navigates to the new webspace's route.
		await expect(page).toHaveURL(/\/w\/fresh-project$/);

		// The stream shows the unfiltered empty copy immediately.
		await expect(page.getByText('Nothing here yet')).toBeVisible();

		// Assert the ABSENCE of the outage state directly, in the same check
		// window as the empty-copy assertion above — G-07-1 was precisely
		// the case where an error state rendered instead of the empty one,
		// and only a direct negative assertion catches its return.
		await expect(page.getByRole('button', { name: 'Retry' })).toHaveCount(0);
		await expect(page.getByText("Couldn't load this webspace")).toHaveCount(0);

		// The canonical config-effect: the new webspace's table exists.
		const afterCreate = readConfigToml(kernel.configPath);
		const webspacesAfterCreate = afterCreate.webspaces as Record<string, unknown>;
		expect(Object.prototype.hasOwnProperty.call(webspacesAfterCreate, 'fresh-project')).toBe(true);

		// --- Part B: an unconfigured name is not an outage -----------------
		await page.goto(`${kernel.baseURL}/w/totally-unconfigured`);

		await expect(page.getByText("That webspace isn't configured")).toBeVisible();
		await expect(page.getByText(`"totally-unconfigured" isn't in your config`)).toBeVisible();

		// Same negative-assertion discipline as Part A: the not-configured
		// state must never be confused with the outage state.
		await expect(page.getByRole('button', { name: 'Retry' })).toHaveCount(0);
		await expect(page.getByText("Couldn't load this webspace")).toHaveCount(0);

		// The switcher — mounted above every stream state — is the
		// recovery affordance StreamMissing deliberately relies on instead
		// of a Retry button. Prove it actually works: open it, pick
		// "existing", and confirm that webspace's stream renders (its
		// boot-time sync already landed above).
		await page.getByRole('button', { name: 'totally-unconfigured' }).click();
		await page.getByRole('menuitem', { name: 'existing', exact: true }).click();

		await expect(page).toHaveURL(/\/w\/existing$/);
		await expect(page.getByText('Welcome to the mock source')).toBeVisible();
	});
});
