// 16-03-PLAN.md Task 2: the browser-visible half of TRUST-04's file-drop
// escalation path — the sibling case
// kernel/pluginhost/escalation_test.go's own
// TestEscalation_FileDropCannotGrantTrust proves at the Go level. Under
// D-11, a binary dropped straight into the trusted directory with NO
// provenance evidence at all — no link-time manifest entry, no signed
// release manifest beside it — earns nothing from its location. It
// resolves TierExternal exactly like any other unverified binary, and an
// unpinned external-tier launch refuses through the SAME existing
// consent/pin machinery `11-external-tier-badge.spec.ts` and
// `11-untrusted-add.spec.ts` already assert on — never a new UI surface.
//
// `bin/plugins-external/topos-plugin-external-demo` — the real,
// genuinely out-of-repo proof binary `make e2e`'s own `external-demo`
// dependency builds — is linked into the hermetic kernel's TRUSTED
// directory under its own name via `trustedBinaryLinks`, exactly the
// mechanism the old `13-manifest-unverified.spec.ts` used before D-11
// repointed it (see that spec's own header comment). That name never
// enters `MANIFEST_E2E_BINARIES` (Makefile), and no signed release
// manifest is placed beside it either — the exact "someone dropped a
// binary straight into the trusted directory" shape, with no pin
// recorded for it.
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import type { FixtureConfigSpec } from '../fixtures/config-builder';
import { EXTERNAL_DEMO_BIN_DIR } from '../fixtures/plugin-binaries';

const WEBSPACE = 'file-drop-tier';
const DROPPED_BINARY = 'topos-plugin-external-demo';
const DROPPED_DISPLAY = 'Dropped Binary';

const configSpec: FixtureConfigSpec = {
	sources: [
		{ id: 'control', plugin: 'topos-plugin-mock', displayName: 'Mock Control' },
		{
			id: 'dropped',
			plugin: DROPPED_BINARY,
			displayName: DROPPED_DISPLAY,
			path: '/tmp/topos-e2e-16-file-drop-dropped'
		}
	],
	webspaces: [{ name: WEBSPACE, sources: ['control', 'dropped'], keywords: ['demo'] }],
	pluginBinaries: ['topos-plugin-mock'],
	trustedBinaryLinks: [{ name: DROPPED_BINARY, srcPath: `${EXTERNAL_DEMO_BIN_DIR}/${DROPPED_BINARY}` }]
};

test.use({ configSpec });

test.describe('16-03 Task 2: a binary dropped into the trusted directory with zero provenance evidence lands untrusted, not trusted', () => {
	test('control chip healthy, dropped chip untrusted/consent-required (not silently trusted), no synced items, and the refusal is named in the kernel log', async ({
		page,
		kernel
	}) => {
		// Only wait on the CONTROL instance's first sync — `dropped` never
		// launches at all (unpinned external tier), so it never reports a
		// landed sync and would stall waitForFirstSync until its own
		// timeout if named here.
		await waitForFirstSync(kernel.baseURL, ['control'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const controlChip = page.getByRole('button', { name: 'Mock Control', exact: true });
		await expect(controlChip).toBeVisible();
		await expect(controlChip.locator('span.size-2')).toHaveClass(/bg-success/);
		// A healthy trusted-tier chip carries no trust badge — same selector
		// convention `11-external-tier-badge.spec.ts` establishes.
		await expect(controlChip.locator('svg.lucide-circle-alert')).toHaveCount(0);

		const droppedChip = page.getByRole('button', { name: DROPPED_DISPLAY, exact: true });
		await expect(droppedChip).toBeVisible();
		await expect(
			droppedChip.locator('span.size-2'),
			'expected the dropped chip to render the DESTRUCTIVE tone — never launched, not a silent success'
		).toHaveClass(/bg-destructive/);
		// The trust badge (svg.lucide-circle-alert, `11-external-tier-badge.spec.ts`'s
		// own selector) renders for ANY external-tier chip, healthy or not —
		// D-11's location-independence proof: this binary earns external
		// tier from provenance alone, exactly like a genuinely untrusted add
		// via the picker (`11-untrusted-add.spec.ts`), never a trusted-tier
		// silent launch just because it sits in the trusted directory.
		await expect(
			droppedChip.locator('svg.lucide-circle-alert:visible'),
			'expected the dropped chip to carry the untrusted trust badge'
		).toHaveCount(1);
		// 14-02-PLAN.md Task 2 (14-UI-SPEC.md G1, option-b): the contract-exact
		// sentence no longer renders through a native `title` attribute — it is
		// exposed as the button's accessible DESCRIPTION via a visually-hidden
		// sr-only span wired through aria-describedby.
		await expect(droppedChip).toHaveAccessibleDescription(
			`${DROPPED_DISPLAY} — binary changed since it was trusted`
		);

		// The consent-required remedy IS available (unlike the tampered-
		// provenance case): "Trust updated binary…" is offered, because this
		// is the same never-pinned external-tier state any other unpinned
		// add would present, not a tamper refusal.
		await page.getByRole('button', { name: `${DROPPED_DISPLAY} actions` }).click();
		const menu = page.getByRole('menu');
		await expect(menu).toBeVisible();
		await expect(menu.getByRole('menuitem', { name: 'Trust updated binary…' })).toHaveCount(1);
		await page.keyboard.press('Escape');
		await expect(menu).toHaveCount(0);

		// The kernel's own GET /api/sources: never reachable (it never
		// launched), tier is external, and launch_failure names pin_mismatch
		// — never manifest_unverified (D-11: no evidence means external
		// tier, not a trusted-tier refusal).
		const sourcesRes = await fetch(`${kernel.baseURL}/api/sources`);
		expect(sourcesRes.ok).toBe(true);
		const sourcesBody = (await sourcesRes.json()) as {
			sources: Array<{ name: string; tier: string; reachable: boolean; launch_failure?: string }>;
		};
		const droppedEntry = sourcesBody.sources.find((s) => s.name === 'dropped');
		expect(droppedEntry, 'expected a "dropped" entry on GET /api/sources').toBeDefined();
		expect(droppedEntry?.tier).toBe('external');
		expect(droppedEntry?.reachable, 'expected the dropped source to be unreachable').toBe(false);
		expect(droppedEntry?.launch_failure).toBe('pin_mismatch');

		// Nothing in the rendered stream carries an item from the dropped
		// source — it never synced.
		await expect(page.locator('[data-item-id^="dropped:"]')).toHaveCount(0);

		// Success criterion 5: the operator can see WHY — the kernel log
		// names the dropped binary and the reason it was refused.
		const logs = kernel.logs();
		expect(logs, 'expected the kernel log to name the pin-mismatch refusal').toContain(
			'pinned binary hash mismatch'
		);
		expect(logs, 'expected the refusal log line to name the refused instance').toContain(
			'instance=dropped'
		);
	});
});
