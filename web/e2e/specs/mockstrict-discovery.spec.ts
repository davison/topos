// Proves the whole discovery chain D-06/D-07 depend on, end to end
// (07.1-02-PLAN.md Task 3): the fixture plugins directory holds exactly
// the requested closed set, the kernel's plugin-type discovery excludes
// the mock reference fixture but includes mockstrict, and the SPA's "+"
// chip picker renders a Mockstrict catalog tile backed by pluginTypeLabel's
// title-case fallback branch (no PLUGIN_TYPE_LABELS entry exists for it).
import { readdirSync } from 'node:fs';
import { join } from 'node:path';

import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { emptyShellWebspace, mockInstances, type FixtureConfigSpec } from '../fixtures/config-builder';

// One configured mock instance (unrelated to the "discovery" webspace
// itself, never attached to it) is required for WebspaceHeader's chip row
// — and therefore the "+" add-source trigger inside it — to render at
// all: format.ts's shouldShowSourceRows gates the whole row on "does this
// INSTALLATION have any configured source instance", read from the raw,
// unfiltered GET /api/sources response, independent of which webspace is
// open. A D-20 empty-shell "discovery" webspace (no keywords/sources/
// match) still needs no participating source of its own for this spec —
// plugin-type discovery, and the picker rows it renders, come from
// GET /api/config/plugin-types (a directory listing), never from a
// webspace's own participation state.
const configSpec: FixtureConfigSpec = {
	sources: mockInstances(1),
	webspaces: [emptyShellWebspace('discovery')],
	pluginBinaries: ['topos-plugin-mock', 'topos-plugin-mockstrict']
};

test.use({ configSpec });

test.describe('mockstrict plugin-type discovery', () => {
	test('the temp plugins directory holds exactly the requested binary set — mock and mockstrict, nothing else', async ({
		kernel
	}) => {
		const entries = readdirSync(join(kernel.tmpDir, 'plugins')).sort();
		expect(entries).toEqual(['topos-plugin-mock', 'topos-plugin-mockstrict']);
	});

	test('GET /api/config/plugin-types includes topos-plugin-mockstrict and excludes topos-plugin-mock', async ({
		kernel
	}) => {
		const res = await fetch(`${kernel.baseURL}/api/config/plugin-types`);
		expect(res.ok).toBe(true);
		const body = (await res.json()) as { plugin_types: string[] };

		expect(body.plugin_types).toContain('topos-plugin-mockstrict');
		expect(body.plugin_types).not.toContain('topos-plugin-mock');
	});

	test('the add-source picker renders a Mockstrict catalog tile', async ({ page, kernel }) => {
		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });

		await page.goto(`${kernel.baseURL}/w/discovery`);

		await page.getByRole('button', { name: 'Add source' }).click();

		await expect(page.getByRole('button', { name: 'Mockstrict', exact: true })).toBeVisible();
	});
});
