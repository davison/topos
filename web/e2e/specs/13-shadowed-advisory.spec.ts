// 13-06-PLAN.md Task 2: proves the D-14 shadowing advisory is visible in
// the browser, not only logged (kernel/pluginhost/manifestgate_test.go's
// TestLaunch_ManifestGate_TrustedShadowingExternalCarriesAdvisory already
// proves the kernel-side fact; this spec proves the chip half).
//
// `bin/plugins/topos-plugin-mock` is linked into BOTH the hermetic
// trusted directory (pluginBinaries, the default every other fixture
// already uses) AND the hermetic external directory (externalPluginBinaries,
// with a real pin recorded against it — the same mechanism
// 11-external-tier-badge.spec.ts already exercises) under the identical
// name. The trusted copy wins the launch (the shadow rule is unchanged)
// and verifies against the build manifest (topos-plugin-mock IS in
// MANIFEST_E2E_BINARIES), so the instance launches and syncs normally —
// its chip should read the WARNING tone (an ambiguity advisory), never
// destructive.
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import type { FixtureConfigSpec } from '../fixtures/config-builder';

const WEBSPACE = 'shadow-advisory';
const SHADOWED_DISPLAY = 'Shadowed Mock';

const configSpec: FixtureConfigSpec = {
	sources: [{ id: 'shadowed', plugin: 'topos-plugin-mock', displayName: SHADOWED_DISPLAY }],
	webspaces: [{ name: WEBSPACE, sources: ['shadowed'], keywords: ['demo'] }],
	pluginBinaries: ['topos-plugin-mock'],
	externalPluginBinaries: ['topos-plugin-mock']
};

test.use({ configSpec });

test.describe('13-06 Task 2: a trusted binary shadowing a same-named pinned external plugin carries a visible advisory', () => {
	test('warning-tone chip (never destructive), contract-exact tooltip, source still launches and syncs', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['shadowed'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const chip = page.getByRole('button', { name: SHADOWED_DISPLAY, exact: true });
		await expect(chip).toBeVisible();
		await expect(
			chip.locator('span.size-2'),
			'expected the shadowed chip to render the WARNING tone'
		).toHaveClass(/bg-warning/);
		await expect(
			chip.locator('span.size-2'),
			'expected the shadowed chip to NEVER render the destructive tone — the pinned external plugin the operator consented to may still be the one running'
		).not.toHaveClass(/bg-destructive/);
		await expect(chip).toHaveAttribute(
			'title',
			`${SHADOWED_DISPLAY} — a same-named trusted-directory binary is shadowing this pinned plugin`
		);

		// The kernel's own GET /api/sources: the instance DID launch and
		// synced normally — launch_advisory is a fact about an entry that
		// launched, never a failure.
		const sourcesRes = await fetch(`${kernel.baseURL}/api/sources`);
		expect(sourcesRes.ok).toBe(true);
		const sourcesBody = (await sourcesRes.json()) as {
			sources: Array<{
				name: string;
				reachable: boolean;
				tier: string;
				launch_advisory?: string;
				launch_failure?: string;
			}>;
		};
		const entry = sourcesBody.sources.find((s) => s.name === 'shadowed');
		expect(entry, 'expected a "shadowed" entry on GET /api/sources').toBeDefined();
		expect(entry?.reachable, 'expected the trusted copy to have launched and be reachable').toBe(
			true
		);
		expect(entry?.tier).toBe('trusted');
		expect(entry?.launch_advisory).toBe('shadowed');
		expect(entry?.launch_failure ?? '').toBe('');

		// The source actually synced its corpus — a degraded-advisory
		// source is not an outage.
		await expect(page.getByText('Welcome to the mock source')).toBeVisible();
	});
});
