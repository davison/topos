// 09-07-PLAN.md Task 3 (09-UI-SPEC.md Fix 11): proves the "+" picker's
// two-group rework against a real kernel — two headed groups, an instance
// row that identifies itself by configured location rather than the raw
// plugin binary name, a catalog tile visually distinct from an instance
// row (via computed style, not class strings), long-value truncation with
// a native title, and that neither pre-existing add flow regressed.
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { attachedWebspace, type FixtureConfigSpec, type FixtureSourceSpec } from '../fixtures/config-builder';
import { offerPluginType } from '../fixtures/plugin-types';

// Two configured-but-unattached mock-01/mock-02 instances of the SAME
// plugin binary (topos-plugin-mock), each with a distinct base_url — the
// only thing that would have distinguished them under the shipped code was
// display_name; base_url is the value Fix 11 makes visible instead. A
// third instance (mock-03) carries a deliberately overlong base_url for
// the truncation/title case. None of the three participate in `armor`
// (only mock-01 does — attachedWebspace below only names it), so mock-02
// and mock-03 are exactly what Group 1 offers.
const LONG_LOCATION =
	'https://a-very-long-subdomain-name-that-will-not-fit-in-the-picker-row.example.com/some/deep/path/segment';

// kernel/config.Validate requires every configured source to declare
// either base_url+token or path (config-builder.ts's own
// PLACEHOLDER_CONNECTION_VALUE comment) — topos-plugin-mock itself ignores
// both, but a base_url with no token still fails that gate, so a
// placeholder token rides along wherever base_url is set below.
const FIXTURE_TOKEN = 'e2e-fixture-unused';

const sources: FixtureSourceSpec[] = [
	{
		id: 'mock-01',
		plugin: 'topos-plugin-mock',
		displayName: 'Mock One',
		baseUrl: 'https://mock-one.example.com',
		token: FIXTURE_TOKEN
	},
	{
		id: 'mock-02',
		plugin: 'topos-plugin-mock',
		displayName: 'Mock Two',
		baseUrl: 'https://mock-two.example.com',
		token: FIXTURE_TOKEN
	},
	{
		id: 'mock-03',
		plugin: 'topos-plugin-mock',
		displayName: 'Mock Three',
		baseUrl: LONG_LOCATION,
		token: FIXTURE_TOKEN
	}
];

const configSpec: FixtureConfigSpec = {
	sources,
	webspaces: [attachedWebspace('armor', ['mock-01'], { 'mock-01': { labels: ['demo'] } })],
	// topos-plugin-mockstrict is discovered but never configured with an
	// instance — the shape Group 2's catalog tile renders (Fix 11's honest
	// "never Described, always Puzzle" case). The kernel no longer
	// advertises it as an installable catalog type (quick task 260811-r5d,
	// kernel/pluginhost.ExcludedPluginBinaries), so the three tests below
	// that need Group 2 populated (its "Install a new source" heading
	// and/or the Mockstrict tile itself) restore it via offerPluginType
	// (../fixtures/plugin-types) route injection — this fixture's own
	// Group 2 subject is restored, not deleted. The remaining two tests
	// (the location-vs-plugin-name row test and the truncation test) only
	// ever assert on Group 1 and are unaffected.
	pluginBinaries: ['topos-plugin-mock', 'topos-plugin-mockstrict']
};

test.use({ configSpec });

test.describe('09-07: two-group "+" picker (Fix 11)', () => {
	test('two headed groups render with their exact copy', async ({ page, kernel }) => {
		await waitForFirstSync(kernel.baseURL, ['mock-01', 'mock-02', 'mock-03'], { logs: kernel.logs });
		// The kernel excludes topos-plugin-mockstrict from the catalog
		// (quick task 260811-r5d) — restore Group 2 via injection so its own
		// "Install a new source" heading has a tile to render for.
		await offerPluginType(page, 'topos-plugin-mockstrict');
		await page.goto(`${kernel.baseURL}/w/armor`);

		await page.getByRole('button', { name: 'Add source' }).click();

		await expect(page.getByText('Add to this webspace', { exact: true })).toBeVisible();
		await expect(page.getByText('Install a new source', { exact: true })).toBeVisible();
	});

	test('an instance row carries its configured location, not the plugin binary name', async ({ page, kernel }) => {
		await waitForFirstSync(kernel.baseURL, ['mock-01', 'mock-02', 'mock-03'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/armor`);

		await page.getByRole('button', { name: 'Add source' }).click();

		const mockTwoRow = page.getByRole('button', { name: /Mock Two/ });
		const rowText = await mockTwoRow.textContent();

		// This is the assertion that would have failed against the shipped
		// code — the row used to show the raw plugin binary name
		// (topos-plugin-mock), identical across mock-02 and mock-03, never
		// their configured location.
		expect(rowText, "expected Mock Two's row to show its own configured base_url").toContain(
			'https://mock-two.example.com'
		);
		expect(
			rowText,
			"expected Mock Two's row to never show the raw plugin binary name"
		).not.toContain('topos-plugin-mock');
	});

	test('a catalog tile is visually distinct from an instance row via computed style', async ({ page, kernel }) => {
		await waitForFirstSync(kernel.baseURL, ['mock-01', 'mock-02', 'mock-03'], { logs: kernel.logs });
		// The kernel excludes topos-plugin-mockstrict from the catalog
		// (quick task 260811-r5d) — restore the Group 2 tile via injection.
		await offerPluginType(page, 'topos-plugin-mockstrict');
		await page.goto(`${kernel.baseURL}/w/armor`);

		await page.getByRole('button', { name: 'Add source' }).click();

		const instanceRow = page.getByRole('button', { name: /Mock Two/ });
		const catalogTile = page.getByRole('button', { name: 'Mockstrict', exact: true });
		await expect(catalogTile).toBeVisible();

		const instanceRowBorderWidth = await instanceRow.evaluate(
			(el) => getComputedStyle(el).borderWidth
		);
		const catalogTileBorderWidth = await catalogTile.evaluate(
			(el) => getComputedStyle(el).borderWidth
		);

		// Read resolved computed values, not class strings — a class present
		// but not applied (e.g. shadowed by a later utility) could not pass
		// this assertion.
		expect(
			catalogTileBorderWidth,
			'expected the catalog tile to carry a real, non-zero border'
		).not.toBe('0px');
		expect(
			instanceRowBorderWidth,
			'expected a plain Group 1 row to carry no border at rest — the whole point of the tile being visibly heavier chrome'
		).not.toBe(catalogTileBorderWidth);
	});

	test('a long location value truncates with a native title carrying the full value', async ({ page, kernel }) => {
		await waitForFirstSync(kernel.baseURL, ['mock-01', 'mock-02', 'mock-03'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/armor`);

		await page.getByRole('button', { name: 'Add source' }).click();

		const mockThreeRow = page.getByRole('button', { name: /Mock Three/ });
		const locationEl = mockThreeRow.locator('[title]');
		await expect(locationEl).toHaveCount(1);
		await expect(locationEl).toHaveAttribute('title', LONG_LOCATION);
	});

	test('both existing add flows still open their correct dialog — instance row and catalog tile', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock-01', 'mock-02', 'mock-03'], { logs: kernel.logs });
		// The kernel excludes topos-plugin-mockstrict from the catalog
		// (quick task 260811-r5d) — restore the Group 2 tile via injection.
		await offerPluginType(page, 'topos-plugin-mockstrict');
		await page.goto(`${kernel.baseURL}/w/armor`);

		// --- One-step existing-instance flow, entered via a Group 1 row.
		await page.getByRole('button', { name: 'Add source' }).click();
		await page.getByRole('button', { name: /Mock Two/ }).click();

		const existingDialog = page.getByRole('dialog');
		await expect(existingDialog.getByRole('heading', { name: 'Add Mock Two to armor' })).toBeVisible();
		await existingDialog.getByRole('button', { name: 'Cancel' }).click();
		await expect(page.getByRole('dialog')).toHaveCount(0);

		// --- Two-step new-plugin-type flow, entered via a Group 2 tile.
		await page.getByRole('button', { name: 'Add source' }).click();
		await page.getByRole('button', { name: 'Mockstrict', exact: true }).click();

		const connectDialog = page.getByRole('dialog');
		await expect(connectDialog.getByRole('heading', { name: 'Connect Mockstrict' })).toBeVisible();
	});
});
