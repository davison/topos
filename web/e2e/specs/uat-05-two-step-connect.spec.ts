// Ports 07-UAT.md item 5 onto `topos-plugin-mockstrict` (D-05): running
// this against the real Signal plugin would require cgo, SQLCipher and a
// Signal Desktop database, none of which this hermetic harness has.
//
// Verbatim pass condition this spec encodes (07-UAT.md item 5):
//   "+ -> New Signal…: path field arrives pre-filled. Clear it and click
//   Next: missing-field message, zero network requests. Restore it and
//   click Next: Match step loads, finished instance appears as a chip."
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { attachedWebspace, type FixtureConfigSpec, type FixtureSourceSpec } from '../fixtures/config-builder';
import { readConfigToml } from '../fixtures/toml';

const sources: FixtureSourceSpec[] = [
	{ id: 'mock-01', plugin: 'topos-plugin-mock', displayName: 'Mock One' }
];

const configSpec: FixtureConfigSpec = {
	sources,
	webspaces: [attachedWebspace('armor', ['mock-01'], { 'mock-01': { labels: ['demo'] } })],
	pluginBinaries: ['topos-plugin-mock', 'topos-plugin-mockstrict']
};

test.use({ configSpec });

// The exact seeded default plugin-fields.ts's topos-plugin-mockstrict row
// declares (07.1-02-SUMMARY.md) — the value this spec asserts the Corpus
// Path input arrives pre-filled with, and the value it types back in to
// restore the field.
const SEEDED_PATH_DEFAULT = '/tmp/topos-e2e-corpus';

test.describe('07-UAT item 5: two-step "New Mockstrict…" connect flow and the blank-required-field guard', () => {
	test('the required field arrives pre-filled, a blank field blocks with zero requests, and restoring it advances to a real describe-driven Match step', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/armor`);

		// Route handlers registered BEFORE any triggering action, per this
		// plan's own rule — a handler registered after the click it observes
		// would lose the race against the browser's own dispatch and the
		// zero-request assertion below would be vacuous.
		let describeRequestCount = 0;
		let putConfigRequestCount = 0;
		await page.route('**/api/config/describe-plugin', async (route) => {
			describeRequestCount++;
			await route.continue();
		});
		await page.route('**/api/config', async (route) => {
			if (route.request().method() === 'PUT') putConfigRequestCount++;
			await route.continue();
		});

		await page.getByRole('button', { name: 'Add source' }).click();
		await page.getByText('New Mockstrict…').click();

		const dialog = page.getByRole('dialog');

		// --- Step 1/2: the Connect step is showing, "1. Connect" emphasised,
		// and the Corpus Path input's VALUE (not merely its placeholder)
		// equals the seeded default — asserting the value is the whole point:
		// a placeholder cannot be submitted, and that presentation gap is
		// exactly the defect G-07-5 diagnosed.
		await expect(dialog.getByRole('heading', { name: 'Connect Mockstrict' })).toBeVisible();
		await expect(dialog.getByText('1. Connect', { exact: true })).toHaveClass(/font-semibold/);
		await expect(dialog.getByText('2. Match', { exact: true })).not.toHaveClass(/font-semibold/);

		const pathInput = dialog.locator('#conn-path');
		await expect(pathInput).toHaveValue(SEEDED_PATH_DEFAULT);

		// A display name is required for resolveNewInstanceId to derive a
		// collision-free instance id before either "Next" click below can
		// ever advance past the Connect step (instance-id.ts) — entered here,
		// ahead of the blank-path assertion, since it plays no part in that
		// assertion's own missing-field message (missingRequiredFields checks
		// only fields the CONNECTION_FIELDS table marks required, and
		// display_name is not one of them).
		await dialog.locator('#conn-display_name').fill('Corpus One');

		// --- Step 3/4: clear the required field and click Next — assert the
		// missing-field message AND zero describe-plugin/config-write
		// requests together. The zero-request half is the load-bearing
		// assertion (T-07.1-16): the UI's job is to block before the plugin
		// subprocess is ever launched with a blank mandatory field.
		//
		// A single space, not an empty string, is what reaches the app's own
		// missingRequiredFields() JS guard here: the Input's native HTML
		// `required` attribute (ConnectionForm.svelte's own documented "first
		// line of defence") intercepts a genuinely empty value at the browser
		// level BEFORE the form's `submit` event — and therefore this
		// component's onsubmit handler — ever fires, per the HTML
		// constraint-validation algorithm (confirmed empirically: an empty
		// fill() produces zero describe-plugin requests but no visible
		// message, since handleConnectNext() never runs). A whitespace-only
		// value satisfies native `required` (it is non-empty) while still
		// failing missingRequiredFields' own `trim() === ''` check — exactly
		// the gap ConnectionForm.svelte's comment names as the JS guard's
		// load-bearing case, distinct from the native attribute's decorative
		// one, and the only value that reaches this assertion's target code
		// path via a real click rather than the browser's own validation UI.
		await pathInput.fill(' ');
		await dialog.getByRole('button', { name: 'Next' }).click();

		await expect(dialog.getByText('Fill in Corpus Path before continuing.')).toBeVisible();
		expect(describeRequestCount).toBe(0);
		expect(putConfigRequestCount).toBe(0);

		// --- Step 5: restore the seeded value and click Next — the Match
		// step loads, with a field labelled "Tags" (mockstrict's own
		// vocabulary, distinct from mock's "Labels") proving the vocabulary
		// came from a real describe round-trip against the launched plugin
		// rather than a cached or defaulted shape.
		await pathInput.fill(SEEDED_PATH_DEFAULT);
		await dialog.getByRole('button', { name: 'Next' }).click();

		await expect(dialog.getByRole('heading', { name: 'Match settings for armor' })).toBeVisible();
		await expect(dialog.getByText('2. Match', { exact: true })).toHaveClass(/font-semibold/);
		await expect(dialog.getByLabel('Tags')).toBeVisible();
		await expect(dialog.getByText('Labels', { exact: true })).toHaveCount(0);
		expect(describeRequestCount).toBe(1);
		expect(putConfigRequestCount).toBe(0);

		// --- Step 6/7: enter a value the mockstrict corpus actually carries
		// ("strict" — matches s1 and s3, plugins/mockstrict/plugin.go) and
		// submit. The modal closes and a chip for the new instance appears
		// without a page reload.
		await dialog.getByLabel('Tags').fill('strict');
		await dialog.getByRole('button', { name: 'Add source' }).click();

		await expect(page.getByRole('dialog')).toHaveCount(0);
		await expect(page.getByRole('button', { name: 'Edit Corpus One' })).toBeVisible();
		expect(putConfigRequestCount).toBe(1);

		// --- Step 8: the parsed config gains a source table with
		// plugin=topos-plugin-mockstrict and a non-empty path, the webspace's
		// sources allowlist includes it, and its match block carries the
		// "tags" field with the entered value.
		const cfg = readConfigToml(kernel.configPath) as {
			sources: Record<string, { plugin: string; path?: string }>;
			webspaces: Record<
				string,
				{ sources: string[]; match: Record<string, Record<string, string[]>> }
			>;
		};
		const newSource = cfg.sources['corpus-one'];
		expect(newSource).toBeDefined();
		expect(newSource.plugin).toBe('topos-plugin-mockstrict');
		expect(newSource.path).toBeTruthy();
		expect(cfg.webspaces.armor.sources).toContain('corpus-one');
		expect(cfg.webspaces.armor.match['corpus-one']).toEqual({ tags: ['strict'] });

		// --- Step 9: the instance's items appear in the stream once its
		// first sync lands, distinguishable from the mock instance's by
		// title ("Mockstrict fixture: …" vs mock's own item titles). "strict"
		// matches exactly two of the fixed three-item corpus (s1, s3).
		await expect(page.locator('[data-item-id^="corpus-one:"]')).toHaveCount(2, { timeout: 15_000 });
		const titles = await page.locator('[data-item-id^="corpus-one:"]').allTextContents();
		for (const title of titles) {
			expect(title).toContain('Mockstrict fixture:');
		}
	});
});
