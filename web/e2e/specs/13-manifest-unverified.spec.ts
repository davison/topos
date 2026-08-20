// 13-06-PLAN.md Task 2 (D-12/D-13) originally proved the file-drop bypass
// path this spec's own name still recalls. Repointed by 16-03-PLAN.md
// Task 2 (D-11): under provenance-based trust, a binary sitting in the
// trusted directory with NO evidence at all is now the FILE-DROP case
// (see the sibling spec `16-file-drop-external-tier.spec.ts`), not a
// manifest-unverified refusal — location alone confers nothing. The case
// this spec still genuinely covers is the one D-11 left untouched: a
// binary whose provenance says one thing and whose bytes say another.
//
// The fixture now links a TAMPERED copy of the real `topos-plugin-mock`
// binary (one byte appended, so its digest differs) into the hermetic
// kernel's trusted directory under the name `topos-plugin-mock` — a name
// `MANIFEST_E2E_BINARIES` (Makefile) DOES cover, so the kernel's
// link-time build manifest positively vouches for it with a digest that
// no longer matches what's on disk. The "control" instance is
// `topos-plugin-mockstrict` (also a `MANIFEST_E2E_BINARIES` name) rather
// than `topos-plugin-mock`, so both the tampered and the healthy binary
// can coexist as distinct on-disk files under this one trusted directory.
//
// Every assertion below is unchanged in shape from before this repoint —
// destructive chip, contract-exact tooltip, absence of a reachable probe,
// absence of a re-pin action, and the named refusal in the kernel log —
// because that surface is still exactly right for a binary whose
// provenance says one thing and whose bytes say another
// (kernel/pluginhost/escalation_test.go's own
// "digest mismatch under a legitimately-named manifest entry" case pins
// the identical property at the Go level).
import { readFileSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import type { FixtureConfigSpec } from '../fixtures/config-builder';
import { mkdtempCorpus } from '../fixtures/corpus';
import { PLUGIN_BIN_DIR } from '../fixtures/plugin-binaries';

const WEBSPACE = 'manifest-gate';
const TAMPERED_BINARY = 'topos-plugin-mock';
const TAMPERED_DISPLAY = 'Tampered Provenance';

// Built ONCE at module load: the real topos-plugin-mock bytes plus one
// appended byte, so its SHA-256 differs from what the kernel's link-time
// build manifest vouches for under this exact name (D-05: names are
// bound to hashes). Written into its own mkdtempCorpus-managed scratch
// directory (never node:fs's own temp-dir primitive directly, per
// spec-hygiene.spec.ts's mechanical rule) under a name distinct from
// `topos-plugin-mock` itself; only the SYMLINK destination name (below, via
// trustedBinaryLinks) needs to read `topos-plugin-mock`.
const tamperedScratchDir = mkdtempCorpus('topos-e2e-13-manifest-unverified-');
const originalMockBytes = readFileSync(join(PLUGIN_BIN_DIR, 'topos-plugin-mock'));
const tamperedMockBytes = Buffer.concat([originalMockBytes, Buffer.from([0])]);
const tamperedMockPath = join(tamperedScratchDir, 'topos-plugin-mock-tampered');
writeFileSync(tamperedMockPath, tamperedMockBytes, { mode: 0o755 });

const configSpec: FixtureConfigSpec = {
	sources: [
		{
			id: 'control',
			plugin: 'topos-plugin-mockstrict',
			displayName: 'Mock Control',
			path: '/tmp/topos-e2e-13-manifest-unverified-control'
		},
		{ id: 'tampered', plugin: TAMPERED_BINARY, displayName: TAMPERED_DISPLAY }
	],
	webspaces: [{ name: WEBSPACE, sources: ['control', 'tampered'], keywords: ['demo'] }],
	pluginBinaries: ['topos-plugin-mockstrict'],
	trustedBinaryLinks: [{ name: TAMPERED_BINARY, srcPath: tamperedMockPath }]
};

test.use({ configSpec });

test.describe('16-03 Task 2: a trusted-directory binary whose bytes no longer match a legitimately-named manifest entry refuses to launch', () => {
	test('destructive chip, contract-exact tooltip, no reachable probe, no re-pin action, and the refusal is named in the kernel log', async ({
		page,
		kernel
	}) => {
		// Only wait on the CONTROL instance's first sync — `tampered` never
		// launches at all, so it never reports a landed sync and would
		// stall waitForFirstSync until its own timeout if named here.
		await waitForFirstSync(kernel.baseURL, ['control'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const controlChip = page.getByRole('button', { name: 'Mock Control', exact: true });
		await expect(controlChip).toBeVisible();
		await expect(controlChip.locator('span.size-2')).toHaveClass(/bg-success/);

		const tamperedChip = page.getByRole('button', { name: TAMPERED_DISPLAY, exact: true });
		await expect(tamperedChip).toBeVisible();
		await expect(
			tamperedChip.locator('span.size-2'),
			'expected the manifest-unverified chip to render the DESTRUCTIVE tone, matching the pin-mismatch precedent'
		).toHaveClass(/bg-destructive/);
		// 14-02-PLAN.md Task 2 (14-UI-SPEC.md G1, option-b): the contract-exact
		// sentence no longer renders through a native `title` attribute — it is
		// exposed as the button's accessible DESCRIPTION via a visually-hidden
		// sr-only span wired through aria-describedby.
		await expect(tamperedChip).toHaveAccessibleDescription(
			`${TAMPERED_DISPLAY} — binary not in the trusted build manifest`
		);

		// No new dialog, no new menu item: the chip menu offers exactly
		// its pre-existing entries, with no re-pin ("Trust updated
		// binary…") action — that stays gated on pin_mismatch alone
		// (D-13's "verification never demotes-and-runs" rule).
		await page.getByRole('button', { name: `${TAMPERED_DISPLAY} actions` }).click();
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
		const tamperedEntry = sourcesBody.sources.find((s) => s.name === 'tampered');
		expect(tamperedEntry, 'expected a "tampered" entry on GET /api/sources').toBeDefined();
		expect(
			tamperedEntry?.reachable,
			'expected the manifest-unverified source to be unreachable'
		).toBe(false);
		expect(tamperedEntry?.launch_failure).toBe('manifest_unverified');

		// D-13's log-AND-UI requirement: the refusal is also named in the
		// kernel's own captured output, not only rendered in the browser.
		const logs = kernel.logs();
		expect(
			logs,
			'expected the kernel log to name the manifest-verification refusal (D-12/D-13)'
		).toContain('trusted binary not verified by the build manifest');
		expect(logs, 'expected the refusal log line to name the refused instance').toContain(
			'instance=tampered'
		);
	});
});
