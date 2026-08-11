// Ports 07-UAT.md item 3 into a permanent regression gate.
//
// Verbatim pass condition this spec encodes (07-UAT.md item 3):
//   "+ New webspace → name → submit: navigates to /w/<name> with no
//   restart, config.toml gains [webspaces.<name>], stream is EMPTY.
//   Chip-row "+" then adds exactly the one chosen instance — not every
//   configured instance."
//
// Deviation from the plan's literal must_haves (recorded in full in
// 07.1-04-SUMMARY.md): kernel/config's canonical writer (kernel/config/
// writer.go, toml.Marshal) carries no `omitempty` on Webspace.Keywords/
// Sources/Match, so a UI-created D-20 shell's `keywords`/`sources` keys are
// ALWAYS present on disk (as `[]`), and `match` is always present as an
// empty table — verified directly against the shipped canonical writer,
// not assumed. "Key absence" is therefore not an available signal; this
// spec instead asserts genuine EMPTINESS of all three collections, which is
// the strongest distinguishing signal the shipped writer actually produces
// (and still fails if a shell is ever written with stray content).
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { mockInstances, webspacesWithKeywords, type FixtureConfigSpec } from '../fixtures/config-builder';
import { readConfigToml } from '../fixtures/toml';

const configSpec: FixtureConfigSpec = {
	sources: mockInstances(3),
	webspaces: webspacesWithKeywords(['existing'], ['demo'])
};

test.use({ configSpec });

interface ParsedWebspace {
	keywords?: string[];
	sources?: string[];
	match?: Record<string, unknown>;
}

test.describe('07-UAT item 3: a UI-created webspace is an empty shell, and the chip-row add flow attaches exactly one instance', () => {
	test('creating a webspace produces a genuine D-20 shell, and adding one source from a three-instance picker attaches exactly that one', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock-01', 'mock-02', 'mock-03'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/existing`);

		await page.getByRole('button', { name: 'existing' }).click();
		await page.getByRole('menuitem', { name: '+ New webspace' }).click();
		await page.getByLabel('Name').fill('fresh-shell');
		await page.getByRole('dialog').getByRole('button', { name: 'Create webspace' }).click();

		// Navigates with no page reload; the stream is empty.
		await expect(page).toHaveURL(/\/w\/fresh-shell$/);
		await expect(page.getByText('Nothing here yet')).toBeVisible();

		const afterCreate = readConfigToml(kernel.configPath);
		const shellsById = afterCreate.webspaces as Record<string, ParsedWebspace>;
		expect(shellsById).toHaveProperty('fresh-shell');
		const shell = shellsById['fresh-shell'];

		// Genuine D-20 shell: every collection is EMPTY (see the deviation
		// note above for why emptiness, not key absence, is the assertion).
		expect(shell.keywords ?? []).toEqual([]);
		expect(shell.sources ?? []).toEqual([]);
		expect(Object.keys(shell.match ?? {})).toEqual([]);

		// The picker offers all three configured instances — none
		// participates yet.
		await page.getByRole('button', { name: 'Add source' }).click();
		await expect(page.getByRole('button', { name: /Mock 01/ })).toBeVisible();
		await expect(page.getByRole('button', { name: /Mock 02/ })).toBeVisible();
		await expect(page.getByRole('button', { name: /Mock 03/ })).toBeVisible();

		// Choose exactly one instance.
		await page.getByRole('button', { name: /Mock 02/ }).click();

		const matchDialog = page.getByRole('dialog');
		await expect(matchDialog.getByRole('heading', { name: 'Add Mock 02 to fresh-shell' })).toBeVisible();
		await matchDialog.getByLabel('Labels').fill('demo');
		await matchDialog.getByRole('button', { name: 'Add source' }).click();

		// Exactly one chip renders for the new webspace.
		await expect(page.getByRole('button', { name: 'Mock 02 actions' })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Mock 01 actions' })).toHaveCount(0);
		await expect(page.getByRole('button', { name: 'Mock 03 actions' })).toHaveCount(0);

		// This is the item's real teeth: exactly the CHOSEN instance was
		// attached, not every configured instance.
		const afterAdd = readConfigToml(kernel.configPath);
		const websAfterAdd = afterAdd.webspaces as Record<string, ParsedWebspace>;
		const composed = websAfterAdd['fresh-shell'];
		expect(composed.sources).toEqual(['mock-02']);
		expect(Object.keys(composed.match ?? {})).toEqual(['mock-02']);

		// The other webspace and every instance's own config block are
		// untouched by this add.
		const existingWs = websAfterAdd['existing'] as ParsedWebspace;
		expect(existingWs.keywords).toEqual(['demo']);
		const sourcesById = afterAdd.sources as Record<string, { plugin: string }>;
		expect(sourcesById['mock-01'].plugin).toBe('topos-plugin-mock');
		expect(sourcesById['mock-02'].plugin).toBe('topos-plugin-mock');
		expect(sourcesById['mock-03'].plugin).toBe('topos-plugin-mock');
	});
});
