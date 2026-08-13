// 11-05-PLAN.md Task 3: the untrusted add journey and the extras round
// trip, end to end against a real kernel and the real, genuinely
// out-of-repo topos-plugin-external-demo binary (11-04-PLAN.md's proof
// plugin, linked from bin/plugins-external — never bin/plugins). Proves,
// through the browser rather than a remembered manual check (this
// project's own testing convention):
//
//   - the picker's untrusted label appears on the external-tier catalog
//     tile and not on a trusted-tier Group 1 row (E2/E3, D-07)
//   - the untrusted-confirm interstitial's disabled-until-exact-match gate
//     (E1, D-05) and its kernel-derived binary/hash/env-disclosure facts
//   - confirming writes the pin in the SAME save the match step already
//     issues, using the EXACT hash string the dialog itself displayed —
//     never a value this spec computed independently (T-11-25)
//   - the declared "Workspace ID" extras field (topos-plugin-external-demo
//     ACTUALLY declares this key as required+non-secret, per its own
//     Describe response — testdata/external-plugin/plugin.go) round-trips
//     into `[sources.<id>.extras]` and reaches the plugin process
//     unmodified, observable in the synced stream (PLUG-09)
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import type { FixtureConfigSpec, FixtureSourceSpec } from '../fixtures/config-builder';
import { EXTERNAL_DEMO_BIN_DIR } from '../fixtures/plugin-binaries';

const WEBSPACE = 'proof';
const EXTERNAL_BINARY = 'topos-plugin-external-demo';

// mock-control participates in the webspace (keeping it a non-shell
// explicit allowlist); mock-01 deliberately does NOT — it is exactly what
// Group 1 ("Add to this webspace") offers, giving this spec a trusted-tier
// row to contrast the external-tier catalog tile against, in the SAME
// picker open. topos-plugin-mock is excluded from Group 2's catalog
// (kernel/pluginhost.ExcludedPluginBinaries) — the "trusted mock row" this
// spec compares against the untrusted label is necessarily a Group 1 row,
// not a Group 2 tile.
const sources: FixtureSourceSpec[] = [
	{ id: 'mock-control', plugin: 'topos-plugin-mock', displayName: 'Mock Control' },
	{ id: 'mock-01', plugin: 'topos-plugin-mock', displayName: 'Mock One' }
];

// keywords fallback (D-01, per-instance match blocks optional): both
// mock-control (unused by this spec's own assertions) and the demo
// instance this spec adds via the UI rely on this SAME webspace-level
// fallback rather than an explicit per-instance match block — the demo
// plugin's own fixed item label is "external-demo-proof"
// (testdata/external-plugin/plugin.go's itemLabel), matched
// case-insensitively and exactly.
const configSpec: FixtureConfigSpec = {
	sources,
	webspaces: [{ name: WEBSPACE, sources: ['mock-control'], keywords: ['external-demo-proof'] }],
	pluginBinaries: ['topos-plugin-mock'],
	externalPluginBinaries: [EXTERNAL_BINARY],
	externalPluginBinariesSrcDir: EXTERNAL_DEMO_BIN_DIR
};

test.use({ configSpec });

test.describe('11-05 Task 3: untrusted add journey and the extras passthrough, end to end', () => {
	test('picker labels, the confirm-step gate, the written pin, and the extras item all prove out against the real out-of-repo binary', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock-control', 'mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		// --- 1. Picker: the external-tier catalog tile carries the
		// UNTRUSTED label; the trusted-tier Group 1 row does not. Exactly
		// two group headers — no third, untrusted-only section (D-07).
		await page.getByRole('button', { name: 'Add source' }).click();

		await expect(page.getByText('Add to this webspace', { exact: true })).toHaveCount(1);
		await expect(page.getByText('Install a new source', { exact: true })).toHaveCount(1);

		const mockRow = page.getByRole('button', { name: /Mock One/ });
		await expect(mockRow).toBeVisible();
		await expect(
			mockRow,
			'expected the trusted-tier Group 1 row to carry no Untrusted label'
		).not.toContainText('Untrusted');

		const externalTile = page.getByRole('button', { name: /External Demo/ });
		await expect(externalTile).toBeVisible();
		await expect(
			externalTile,
			'expected the external-tier Group 2 tile to carry the Untrusted label'
		).toContainText('Untrusted');

		// --- 2. Select the external plugin, fill the connect step's one
		// required field, and click Next.
		await externalTile.click();

		const dialog = page.getByRole('dialog');
		await expect(dialog.getByRole('heading', { name: 'Connect External Demo' })).toBeVisible();
		await dialog.locator('#conn-display_name').fill('Demo Proof');
		await dialog.locator('#conn-path').fill('/tmp/topos-e2e-11-external-demo-instance');
		await dialog.getByRole('button', { name: 'Next' }).click();

		// --- 3. The confirm step (E1): binary name, a 64-hex-char pinned
		// hash, and the zero-referenced-vars env-disclosure branch (neither
		// the path nor the display name is a ${VAR} reference). The
		// primary action is disabled until the typed value exactly matches
		// the binary name.
		await expect(dialog.getByRole('heading', { name: 'Add an untrusted source' })).toBeVisible();
		await expect(dialog.getByText(`Binary: ${EXTERNAL_BINARY}`)).toBeVisible();

		const hashLine = dialog.getByText(/Pinned hash \(SHA-256\):/);
		await expect(hashLine).toBeVisible();
		const hashLineText = (await hashLine.textContent()) ?? '';
		const hashMatch = hashLineText.match(/([0-9a-f]{64})/);
		expect(hashMatch, `expected a 64-hex-char hash in "${hashLineText}"`).not.toBeNull();
		const dialogHash = hashMatch![1];

		await expect(
			dialog.getByText(
				"topos will hand this plugin only the standard PATH/HOME/locale environment — this source's configuration references no environment variables."
			)
		).toBeVisible();

		const confirmInput = dialog.locator('#untrusted-confirm-typed');
		const addUntrustedButton = dialog.getByRole('button', { name: 'Add untrusted source' });
		await expect(addUntrustedButton).toBeDisabled();

		await confirmInput.fill('topos-plugin-wrong-name');
		await expect(
			addUntrustedButton,
			'expected the primary action to stay disabled for an incorrect typed value'
		).toBeDisabled();

		await confirmInput.fill(EXTERNAL_BINARY);
		await expect(
			addUntrustedButton,
			'expected the primary action to enable once the typed value exactly matches the binary name'
		).toBeEnabled();

		// --- 4. Before confirming, navigate back to the connect step
		// (E1 interaction: cancelling preserves every typed connection
		// value) and add the extras value there — topos-plugin-external-demo
		// declares "workspace_id" as a REQUIRED, non-secret extras key
		// (testdata/external-plugin/plugin.go's Describe response), so it
		// renders as a labeled declared field, not a free-form row.
		await dialog.getByRole('button', { name: 'Cancel' }).click();
		await expect(dialog.getByRole('heading', { name: 'Connect External Demo' })).toBeVisible();
		await expect(
			dialog.locator('#conn-path'),
			'expected cancelling the confirm step to preserve the already-typed connect-step values'
		).toHaveValue('/tmp/topos-e2e-11-external-demo-instance');

		const workspaceIdInput = dialog.locator('#extra-workspace_id');
		await expect(workspaceIdInput).toBeVisible();
		await workspaceIdInput.fill('acme-42');

		await dialog.getByRole('button', { name: 'Next' }).click();
		await expect(dialog.getByRole('heading', { name: 'Add an untrusted source' })).toBeVisible();
		await expect(addUntrustedButton).toBeDisabled();
		await confirmInput.fill(EXTERNAL_BINARY);
		await expect(addUntrustedButton).toBeEnabled();

		// --- 5. Confirm, complete the match step (relying on the webspace's
		// own keywords fallback — no per-instance match field filled), and
		// save.
		await addUntrustedButton.click();
		await expect(dialog.getByRole('heading', { name: `Match settings for ${WEBSPACE}` })).toBeVisible();
		await dialog.getByRole('button', { name: 'Add source' }).click();

		await expect(page.getByRole('dialog')).toHaveCount(0);

		// --- The new chip appears carrying the trust badge.
		const demoChip = page.getByRole('button', { name: 'Demo Proof', exact: true });
		await expect(demoChip).toBeVisible();
		await expect(
			demoChip.locator('svg.lucide-circle-alert:visible'),
			'expected the newly-added external-tier chip to carry the trust badge'
		).toHaveCount(1);

		// --- GET /api/config: the pin equals the EXACT hash the dialog
		// displayed (never merely a non-empty string — a vacuous assertion
		// would pass even if the client had invented its own value), and
		// the source's extras table carries the typed key.
		const configRes = await fetch(`${kernel.baseURL}/api/config`);
		expect(configRes.ok).toBe(true);
		const configBody = (await configRes.json()) as {
			config: {
				plugins: { pins?: Record<string, string> };
				sources: Record<string, { extras?: Record<string, string> }>;
			};
		};
		expect(configBody.config.plugins.pins?.[EXTERNAL_BINARY]).toBe(dialogHash);
		expect(configBody.config.sources['demo-proof']?.extras?.workspace_id).toBe('acme-42');

		// --- After the first sync, the stream carries the proof binary's
		// own observational item reporting exactly what it received —
		// PLUG-09's passthrough proven by observation, not assertion.
		await waitForFirstSync(kernel.baseURL, ['mock-control', 'mock-01', 'demo-proof'], {
			logs: kernel.logs
		});

		const extrasItem = page.locator('[data-item-id^="demo-proof:"]', {
			hasText: 'extras workspace_id=acme-42'
		});
		await expect(
			extrasItem,
			'expected the demo-proof instance to sync an item reporting the extras key/value it was launched with'
		).toHaveCount(1, { timeout: 15_000 });
	});
});
