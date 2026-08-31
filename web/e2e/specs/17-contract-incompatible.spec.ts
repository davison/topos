// M1-R6/DIST-03 (davison/topos#17): kernel↔plugin incompatibility
// surfaces loudly by name, never as silent source absence — proven at
// the browser layer with a REAL declared mismatch. Two instances of the
// SAME topos-plugin-mock binary share one kernel: "stale" references
// ${WEBSPACES_MOCK_CONTRACT_VERSION} in its own extras, so the value set
// on the kernel process below reaches ITS subprocess (and only its —
// kernel/pluginhost's allowedEnv admits a variable per instance, by
// config reference) and the mock declares contract generation
// "topos.v1" at Describe; the kernel kills it and records
// contract_incompatible. "control" carries no reference, declares the
// sdk's own generation, and syncs normally — one incompatible plugin
// never costs a healthy sibling anything, and never costs the kernel
// its boot (this fixture kernel booting AT ALL is itself an assertion:
// before #17 this exact config was a dead boot).
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import type { FixtureConfigSpec } from '../fixtures/config-builder';

const WEBSPACE = 'contract-gate';
const STALE_DISPLAY = 'Stale Contract';

const configSpec: FixtureConfigSpec = {
	sources: [
		{ id: 'control', plugin: 'topos-plugin-mock', displayName: 'Mock Control' },
		{
			id: 'stale',
			plugin: 'topos-plugin-mock',
			displayName: STALE_DISPLAY,
			// The ${VAR} reference is what admits the env var into this
			// instance's subprocess environment — the key name itself is
			// arbitrary (the mock reads the env var, not this extras value);
			// same pattern as the rendition fixture's spec.
			extras: { contract: '${WEBSPACES_MOCK_CONTRACT_VERSION}' }
		}
	],
	webspaces: [{ name: WEBSPACE, sources: ['control', 'stale'], keywords: ['demo'] }],
	env: { WEBSPACES_MOCK_CONTRACT_VERSION: 'topos.v1' }
};

test.use({ configSpec });

test.describe('17: a contract-generation mismatch is a named launch failure beside a healthy sibling', () => {
	test('destructive chip, both generations named, wire vocabulary, healthy control, named kernel log', async ({
		page,
		kernel
	}) => {
		// Only the control instance ever syncs — "stale" never launches.
		await waitForFirstSync(kernel.baseURL, ['control'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const controlChip = page.getByRole('button', { name: 'Mock Control', exact: true });
		await expect(controlChip).toBeVisible();
		await expect(controlChip.locator('span.size-2')).toHaveClass(/bg-success/);

		const staleChip = page.getByRole('button', { name: STALE_DISPLAY, exact: true });
		await expect(staleChip).toBeVisible();
		await expect(
			staleChip.locator('span.size-2'),
			'expected the contract-incompatible chip to render the DESTRUCTIVE tone, matching the pin-mismatch precedent'
		).toHaveClass(/bg-destructive/);
		// The tooltip (exposed as the accessible description, 14-02's
		// option-b) carries the kernel's own message — which is where BOTH
		// generations are named (M1-R6's "loudly by name").
		await expect(staleChip).toHaveAccessibleDescription(
			/incompatible plugin contract: .*declares contract generation "topos\.v1"; this kernel supports topos\.v2/
		);
		// tier stays trusted (the BINARY is vouched for; its declared
		// contract is what's refused), so no external-tier badge glyph.
		await expect(
			staleChip.locator('svg.lucide-circle-alert'),
			'expected NO trust badge — the binary is trusted; the contract is what is refused'
		).toHaveCount(0);

		// The wire surface: closed vocabulary, never-parsed message text.
		const sourcesRes = await fetch(`${kernel.baseURL}/api/sources`);
		expect(sourcesRes.ok).toBe(true);
		const body = (await sourcesRes.json()) as {
			sources: Array<{
				name: string;
				tier: string;
				reachable: boolean;
				launch_failure?: string;
				last_error?: string;
			}>;
		};
		const stale = body.sources.find((s) => s.name === 'stale');
		expect(stale, 'expected a "stale" entry on GET /api/sources — never silent absence').toBeDefined();
		expect(stale?.launch_failure).toBe('contract_incompatible');
		expect(stale?.reachable).toBe(false);
		expect(stale?.tier).toBe('trusted');
		expect(stale?.last_error).toContain('"topos.v1"');
		expect(stale?.last_error).toContain('topos.v2');

		const control = body.sources.find((s) => s.name === 'control');
		expect(control?.launch_failure ?? '').toBe('');
		expect(control?.tier).toBe('trusted');

		// Log-AND-UI: the refusal is named in the kernel's own output too.
		const logs = kernel.logs();
		expect(logs).toContain('recorded as a named per-instance launch failure');
		expect(logs).toContain('reason=contract_incompatible');
		expect(logs).toContain('instance=stale');
	});
});
