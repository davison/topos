// 16-05-PLAN.md Task 2: the browser-visible half of success criterion 1
// ("trust is not a property of location") — the half only a signed
// release manifest can prove, complementing the two proofs this phase
// already has:
//   - kernel/pluginhost/tier_test.go's TestResolveBinary_LocationSymmetric
//     (16-02-PLAN.md): the SAME table over BOTH the trusted and the
//     external directory, at the Go level.
//   - web/e2e/specs/16-file-drop-external-tier.spec.ts (16-03-PLAN.md):
//     the browser proves the NEGATIVE case — zero evidence at all, in
//     the TRUSTED directory, earns nothing.
// This spec proves the POSITIVE case, in the EXTERNAL directory: a
// binary that is not named by the kernel's own link-time build manifest
// at all, sitting in the directory that has never conferred trust,
// launches TIER TRUSTED purely because a validly-signed release manifest
// vouches for it — then proves the signature itself is what grants that
// trust, not the file's location or the binary's name, by placing a
// SECOND copy of the same binary type under a second renamed
// destination, signing it too, and then deleting only ITS OWN
// `.provenance.sig` sibling before the kernel boots.
//
// Both binaries link the real `topos-plugin-mockstrict` binary under
// RENAMED destinations — never under its own name — because
// `topos-plugin-mockstrict` IS a MANIFEST_E2E_BINARIES name (Makefile):
// the kernel's link-time build manifest would vouch for it directly,
// making the link-time arm (not the signed one) the thing under test.
// Neither renamed destination carries any link-time evidence at all, so
// each one's ONLY path to TierTrusted is its OWN signed manifest,
// written via the real `topos-provenance` CLI (plugin-binaries.ts's
// signProvenanceFixture — never a hand-rolled TypeScript signer), signed
// with the e2e-only key `make e2e` injects into this kernel build
// through the link-time `-X ...provenanceKeysExtra` seam (D-12) — never
// a runtime-readable trust input.
//
// Both cases share ONE kernel boot: Playwright forbids varying a
// worker-scoped fixture's option (configSpec) via `test.use()` inside a
// `test.describe()` block — the two renamed binaries and their
// independently signed-and-optionally-revoked manifests are what let one
// boot carry both the positive and the negative case side by side.
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import type { FixtureConfigSpec } from '../fixtures/config-builder';
import { PLUGIN_BIN_DIR } from '../fixtures/plugin-binaries';

const WEBSPACE = 'signed-provenance-tier';
const SIGNED_BINARY = 'topos-plugin-mockstrict-signed';
const SIGNED_DISPLAY = 'Signed External Corpus';
const REVOKED_BINARY = 'topos-plugin-mockstrict-revoked';
const REVOKED_DISPLAY = 'Revoked Signature Corpus';
const MOCKSTRICT_SRC_PATH = `${PLUGIN_BIN_DIR}/topos-plugin-mockstrict`;

// keywords ['demo', 'fixture'] spans both corpora's own vocabulary field
// exactly as 11-external-tier-badge.spec.ts's identical fixture does:
// mock's field is "labels" (carries "demo"), mockstrict's field is
// "tags" (carries "fixture" on two of its three items) — one webspace
// matches every instance via the D-01 keywords fallback, no per-instance
// match block needed.
const configSpec: FixtureConfigSpec = {
	sources: [
		{ id: 'control', plugin: 'topos-plugin-mock', displayName: 'Mock Control' },
		{
			id: 'signed',
			plugin: SIGNED_BINARY,
			path: '/tmp/topos-e2e-16-signed-provenance-signed-unused',
			displayName: SIGNED_DISPLAY
		},
		{
			id: 'revoked',
			plugin: REVOKED_BINARY,
			path: '/tmp/topos-e2e-16-signed-provenance-revoked-unused',
			displayName: REVOKED_DISPLAY
		}
	],
	webspaces: [{ name: WEBSPACE, keywords: ['demo', 'fixture'] }],
	pluginBinaries: ['topos-plugin-mock'],
	externalBinaryLinks: [
		{ name: SIGNED_BINARY, srcPath: MOCKSTRICT_SRC_PATH },
		{ name: REVOKED_BINARY, srcPath: MOCKSTRICT_SRC_PATH }
	],
	// Two INDEPENDENT signing calls (one manifest each, kernel.ts) — the
	// only way removeSignature on the second entry can delete exactly
	// its own .provenance.sig without also revoking the first entry's
	// trust. Neither entry writes a [plugins.pins] row (config-builder.ts
	// skips both by name) — provenance is the only mechanism under test.
	signedProvenanceBinaries: [{ name: SIGNED_BINARY }, { name: REVOKED_BINARY, removeSignature: true }]
};

test.use({ configSpec });

test.describe('16-05 Task 2: trust comes from a signed provenance manifest, not from the external directory or the binary name', () => {
	test('a validly signed external-directory binary renders healthy and trusted (no untrusted badge, no re-pin remedy), and its items sync with no pin recorded', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['control', 'signed'], { logs: kernel.logs });

		const res = await fetch(`${kernel.baseURL}/api/sources`);
		expect(res.ok).toBe(true);
		const body = (await res.json()) as {
			sources: Array<{
				name: string;
				tier: string;
				reachable: boolean;
				pinned_hash?: string;
				launch_failure?: string;
				last_error: string;
			}>;
		};
		const byName = new Map(body.sources.map((s) => [s.name, s]));
		const control = byName.get('control');
		const signed = byName.get('signed');

		expect(control, 'expected a "control" entry in GET /api/sources').toBeTruthy();
		expect(signed, 'expected a "signed" entry in GET /api/sources').toBeTruthy();
		expect(control?.tier).toBe('trusted');
		expect(signed?.tier, 'expected the signed external-directory binary to resolve TierTrusted').toBe(
			'trusted'
		);
		expect(control?.reachable, `control unreachable: ${control?.last_error}`).toBe(true);
		expect(signed?.reachable, `signed unreachable: ${signed?.last_error}`).toBe(true);
		expect(signed?.launch_failure ?? '', 'expected no launch_failure on the signed instance').toBe('');
		// A trusted-tier instance is never pinned (docs/api.md) — this is
		// also the direct proof that signedProvenanceBinaries wrote no
		// [plugins.pins] entry for it, only a signed manifest.
		expect(signed?.pinned_hash ?? '', 'expected no pin recorded for the signed instance').toBe('');

		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const controlChip = page.getByRole('button', { name: 'Mock Control', exact: true });
		const signedChip = page.getByRole('button', { name: SIGNED_DISPLAY, exact: true });
		await expect(controlChip).toBeVisible();
		await expect(signedChip).toBeVisible();
		await expect(controlChip.locator('span.size-2')).toHaveClass(/bg-success/);
		await expect(
			signedChip.locator('span.size-2'),
			'expected the signed chip to render the HEALTHY tone — it actually launched and synced'
		).toHaveClass(/bg-success/);

		// The untrusted trust badge (svg.lucide-circle-alert,
		// 11-external-tier-badge.spec.ts's own selector) renders on
		// NEITHER chip: this is success criterion 1 made visible — a
		// validly signed binary in the external directory is
		// indistinguishable from a trusted-directory one, from the
		// operator's own point of view.
		await expect(
			signedChip.locator('svg.lucide-circle-alert'),
			'expected NO trust badge on the signed chip — provenance, not location, granted trust'
		).toHaveCount(0);
		await expect(controlChip.locator('svg.lucide-circle-alert')).toHaveCount(0);

		// No re-pin remedy is offered — that action is gated on
		// pin_mismatch alone (D-13's "verification never demotes-and-runs"
		// rule), and this instance never had a pin to mismatch.
		await page.getByRole('button', { name: `${SIGNED_DISPLAY} actions` }).click();
		const menu = page.getByRole('menu');
		await expect(menu).toBeVisible();
		await expect(menu.getByRole('menuitem', { name: 'Trust updated binary…' })).toHaveCount(0);
		await page.keyboard.press('Escape');
		await expect(menu).toHaveCount(0);

		// The signed source actually launched and synced — its items are
		// in the rendered stream, not merely reported reachable.
		await expect(page.locator('[data-item-id^="signed:"]').first()).toBeVisible();
	});

	test('the same binary, signed into the same directory, with its signature removed before boot, resolves untrusted: badge, no launch, no synced items', async ({
		page,
		kernel
	}) => {
		// The "revoked" instance never launches at all (unpinned external
		// tier once its signature is gone), so only wait on control+signed
		// — naming "revoked" here would stall until its own timeout.
		await waitForFirstSync(kernel.baseURL, ['control', 'signed'], { logs: kernel.logs });

		const res = await fetch(`${kernel.baseURL}/api/sources`);
		expect(res.ok).toBe(true);
		const body = (await res.json()) as {
			sources: Array<{ name: string; tier: string; reachable: boolean; launch_failure?: string }>;
		};
		const revoked = body.sources.find((s) => s.name === 'revoked');
		expect(revoked, 'expected a "revoked" entry in GET /api/sources').toBeTruthy();
		expect(
			revoked?.tier,
			'expected the un-vouched binary to fall to TierExternal — no evidence means no trust (D-11)'
		).toBe('external');
		expect(revoked?.reachable, 'expected the revoked source to be unreachable — it never launched').toBe(
			false
		);
		expect(revoked?.launch_failure).toBe('pin_mismatch');

		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const revokedChip = page.getByRole('button', { name: REVOKED_DISPLAY, exact: true });
		await expect(revokedChip).toBeVisible();
		await expect(
			revokedChip.locator('span.size-2'),
			'expected the DESTRUCTIVE tone — never launched, not a silent success'
		).toHaveClass(/bg-destructive/);
		await expect(
			revokedChip.locator('svg.lucide-circle-alert:visible'),
			'expected the untrusted trust badge once the signature is gone — the SAME binary that (under a sibling name) carries no badge with its own signature present'
		).toHaveCount(1);

		// The consent-required remedy IS available — this is the same
		// never-pinned external-tier state any other unpinned add would
		// present, never a tamper refusal.
		await page.getByRole('button', { name: `${REVOKED_DISPLAY} actions` }).click();
		const menu = page.getByRole('menu');
		await expect(menu).toBeVisible();
		await expect(menu.getByRole('menuitem', { name: 'Trust updated binary…' })).toHaveCount(1);
		await page.keyboard.press('Escape');
		await expect(menu).toHaveCount(0);

		// Nothing in the rendered stream carries an item from the revoked
		// source — it never synced.
		await expect(page.locator('[data-item-id^="revoked:"]')).toHaveCount(0);

		// The still-signed "signed" chip from the sibling test is
		// unaffected by revoking a DIFFERENT binary's signature — each
		// entry's manifest is independent.
		const signedChip = page.getByRole('button', { name: SIGNED_DISPLAY, exact: true });
		await expect(signedChip).toBeVisible();
		await expect(signedChip.locator('span.size-2')).toHaveClass(/bg-success/);
		await expect(signedChip.locator('svg.lucide-circle-alert')).toHaveCount(0);
	});
});
