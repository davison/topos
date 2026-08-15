// 13-06-PLAN.md Task 2: proves the file-drop bypass path (D-12/D-13) is
// closed by real code, not only by kernel/pluginhost's own Go unit tests
// (manifestgate_test.go's TestLaunch_ManifestGate_AbsentFromManifestRefusesNoSubprocess
// already proves the refusal at the launch() level; this spec proves the
// visible half a browser sees).
//
// `bin/plugins-external/topos-plugin-external-demo` — a real binary
// `make e2e`'s own `external-demo` dependency builds — is linked into the
// hermetic kernel's TRUSTED plugin directory under its own name. That
// name never enters `MANIFEST_E2E_BINARIES` (Makefile), so the kernel's
// link-time build manifest has no entry for it at all: the exact
// "someone dropped a binary straight into the trusted directory" shape
// D-12/D-13 close, made concrete without building a single new plugin
// binary for this spec (the acceptance criterion `git diff Makefile` is
// empty for this plan).
import { join } from 'node:path';
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import type { FixtureConfigSpec } from '../fixtures/config-builder';
import { EXTERNAL_DEMO_BIN_DIR } from '../fixtures/plugin-binaries';

const WEBSPACE = 'manifest-gate';
const DROPPED_BINARY = 'topos-plugin-external-demo';
const DROPPED_DISPLAY = 'Dropped Binary';

const configSpec: FixtureConfigSpec = {
	sources: [
		{ id: 'control', plugin: 'topos-plugin-mock', displayName: 'Mock Control' },
		{
			id: 'dropped',
			plugin: DROPPED_BINARY,
			displayName: DROPPED_DISPLAY,
			path: '/tmp/topos-e2e-13-manifest-unverified-dropped'
		}
	],
	webspaces: [{ name: WEBSPACE, sources: ['control', 'dropped'], keywords: ['demo'] }],
	pluginBinaries: ['topos-plugin-mock'],
	trustedBinaryLinks: [
		{ name: DROPPED_BINARY, srcPath: join(EXTERNAL_DEMO_BIN_DIR, DROPPED_BINARY) }
	]
};

test.use({ configSpec });

test.describe('13-06 Task 2: a trusted-directory binary absent from the build manifest refuses to launch', () => {
	test('destructive chip, contract-exact tooltip, no reachable probe, no re-pin action, and the refusal is named in the kernel log', async ({
		page,
		kernel
	}) => {
		// Only wait on the CONTROL instance's first sync — `dropped` never
		// launches at all, so it never reports a landed sync and would
		// stall waitForFirstSync until its own timeout if named here.
		await waitForFirstSync(kernel.baseURL, ['control'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const controlChip = page.getByRole('button', { name: 'Mock Control', exact: true });
		await expect(controlChip).toBeVisible();
		await expect(controlChip.locator('span.size-2')).toHaveClass(/bg-success/);

		const droppedChip = page.getByRole('button', { name: DROPPED_DISPLAY, exact: true });
		await expect(droppedChip).toBeVisible();
		await expect(
			droppedChip.locator('span.size-2'),
			'expected the manifest-unverified chip to render the DESTRUCTIVE tone, matching the pin-mismatch precedent'
		).toHaveClass(/bg-destructive/);
		// 14-02-PLAN.md Task 2 (14-UI-SPEC.md G1, option-b): the contract-exact
		// sentence no longer renders through a native `title` attribute — it is
		// exposed as the button's accessible DESCRIPTION via a visually-hidden
		// sr-only span wired through aria-describedby.
		await expect(droppedChip).toHaveAccessibleDescription(
			`${DROPPED_DISPLAY} — binary not in the trusted build manifest`
		);

		// No new dialog, no new menu item: the chip menu offers exactly
		// its pre-existing entries, with no re-pin ("Trust updated
		// binary…") action — that stays gated on pin_mismatch alone
		// (D-13's "verification never demotes-and-runs" rule).
		await page.getByRole('button', { name: `${DROPPED_DISPLAY} actions` }).click();
		const menu = page.getByRole('menu');
		await expect(menu).toBeVisible();
		await expect(menu.getByRole('menuitem', { name: 'Trust updated binary…' })).toHaveCount(0);
		await page.keyboard.press('Escape');
		await expect(menu).toHaveCount(0);

		// The kernel's own GET /api/sources: never reachable (it never
		// launched), and launch_failure names the exact reason.
		const sourcesRes = await fetch(`${kernel.baseURL}/api/sources`);
		expect(sourcesRes.ok).toBe(true);
		const sourcesBody = (await sourcesRes.json()) as {
			sources: Array<{ name: string; reachable: boolean; launch_failure?: string }>;
		};
		const droppedEntry = sourcesBody.sources.find((s) => s.name === 'dropped');
		expect(droppedEntry, 'expected a "dropped" entry on GET /api/sources').toBeDefined();
		expect(droppedEntry?.reachable, 'expected the manifest-unverified source to be unreachable').toBe(
			false
		);
		expect(droppedEntry?.launch_failure).toBe('manifest_unverified');

		// D-13's log-AND-UI requirement: the refusal is also named in the
		// kernel's own captured output, not only rendered in the browser.
		const logs = kernel.logs();
		expect(
			logs,
			'expected the kernel log to name the manifest-verification refusal (D-12/D-13)'
		).toContain('trusted binary not verified by the build manifest');
		expect(logs, 'expected the refusal log line to name the refused instance').toContain(
			'instance=dropped'
		);
	});
});
