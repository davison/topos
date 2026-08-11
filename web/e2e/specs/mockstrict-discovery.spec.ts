// Permanent gate for kernel/pluginhost.ExcludedPluginBinaries' mockstrict
// entry, in both directions (quick task 260811-r5d, superseding this
// file's original 07.1-02-PLAN.md Task 3 "prove mockstrict is offered"
// intent — that intent is now inverted, deliberately, by 260811-r5d's own
// plan): the fixture plugins directory holds exactly the requested closed
// set, the kernel's plugin-type discovery excludes BOTH the mock and
// mockstrict reference fixtures unconditionally, and the SPA's "+" chip
// picker renders no Mockstrict catalog tile — while an already-configured
// mockstrict instance is completely unaffected, still offered as a Group 1
// "Add to this webspace" row and still describable via a real subprocess
// round trip.
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
//
// A second, configured-but-unattached topos-plugin-mockstrict instance
// (09-stream-row-plugin-icon.spec.ts's non-empty-path fixture shape,
// mockstrict's own main.go requiring it) is the load-bearing fixture for
// this file's own inversion: it proves the exclusion is a catalog-listing
// policy only, never a ban on an already-configured instance.
const configSpec: FixtureConfigSpec = {
	sources: [
		...mockInstances(1),
		{
			id: 'mockstrict-01',
			plugin: 'topos-plugin-mockstrict',
			path: '/tmp/e2e-mockstrict-unused',
			displayName: 'Mockstrict Corpus'
		}
	],
	webspaces: [emptyShellWebspace('discovery')],
	pluginBinaries: ['topos-plugin-mock', 'topos-plugin-mockstrict']
};

test.use({ configSpec });

test.describe('mockstrict plugin-type exclusion (quick task 260811-r5d)', () => {
	test('the temp plugins directory holds exactly the requested binary set — mock and mockstrict, nothing else', async ({
		kernel
	}) => {
		const entries = readdirSync(join(kernel.tmpDir, 'plugins')).sort();
		expect(entries).toEqual(['topos-plugin-mock', 'topos-plugin-mockstrict']);
	});

	test('GET /api/config/plugin-types excludes BOTH topos-plugin-mock and topos-plugin-mockstrict', async ({
		kernel
	}) => {
		const res = await fetch(`${kernel.baseURL}/api/config/plugin-types`);
		expect(res.ok).toBe(true);
		const body = (await res.json()) as { plugin_types: string[] };

		// Inverted from this file's pre-260811-r5d assertion (which required
		// mockstrict to be INCLUDED) — the inversion is the point of this
		// change: kernel/pluginhost.ExcludedPluginBinaries now excludes
		// mockstrict unconditionally, exactly like the pre-existing mock
		// exclusion, so both fixture binaries are absent from the catalog a
		// real operator's picker reads.
		expect(body.plugin_types).not.toContain('topos-plugin-mock');
		expect(body.plugin_types).not.toContain('topos-plugin-mockstrict');
	});

	test('the picker offers no Mockstrict catalog tile, while the configured mockstrict instance still appears as a Group 1 row', async ({
		page,
		kernel
	}) => {
		// Only mock-01 (Group 1's chip-row gate) and mockstrict-01 (the
		// instance under test) are configured; neither is attached to the
		// empty-shell "discovery" webspace, but waitForFirstSync polls the
		// unfiltered GET /api/sources — attachment plays no part in whether a
		// source's first sync lands.
		await waitForFirstSync(kernel.baseURL, ['mock-01', 'mockstrict-01'], { logs: kernel.logs });

		await page.goto(`${kernel.baseURL}/w/discovery`);

		await page.getByRole('button', { name: 'Add source' }).click();

		// This pairing is the load-bearing assertion of the whole file: the
		// exclusion is a catalog policy, never a ban on the instance.
		await expect(page.getByRole('button', { name: 'Mockstrict', exact: true })).toHaveCount(0);
		await expect(page.getByRole('button', { name: /Mockstrict Corpus/ })).toBeVisible();
	});

	test('describe-plugin for the configured mockstrict instance returns 200 with its tags match vocabulary', async ({
		kernel
	}) => {
		// Direct regression gate for the 404 failure 07.1-04 hit when this
		// same exclusion mechanism was last widened without also confirming
		// DiscoverAllBinaries (and therefore DescribePluginHandler) stayed
		// unfiltered. Request/response shapes read from
		// kernel/httpapi/config.go's describePluginRequest/describePluginResponse
		// rather than guessed.
		const res = await fetch(`${kernel.baseURL}/api/config/describe-plugin`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({
				plugin: 'topos-plugin-mockstrict',
				source: { plugin: 'topos-plugin-mockstrict', path: '/tmp/e2e-mockstrict-unused' }
			})
		});
		expect(res.ok).toBe(true);
		expect(res.status).toBe(200);
		const body = (await res.json()) as { match_vocabulary: string[] };
		expect(body.match_vocabulary).toContain('tags');
	});
});
