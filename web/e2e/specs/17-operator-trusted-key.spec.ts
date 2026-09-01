// 17-operator-trusted-key.spec.ts — M2-R4 (davison/topos#49, #56, #57):
// operator-trusted developer keys, end to end against a real binary.
//
// The external-demo plugin's release is signed with a SCRATCH key the
// kernel does not accept (generated per worker by keygenScratchKey — never
// the e2e fixture key the build links in), and its source is configured
// WITHOUT a pin. So at boot the kernel refuses it pin-mismatched, carrying
// the OFFER the signature yields. From there:
//
//   1. the chip menu offers "Trust signing key…"; the dialog names the key
//      and fingerprint the kernel reported; confirming writes
//      [[plugins.trusted_keys]] through the one config write, the apply
//      relaunches the plugin at operator_trusted — no pin, the key named,
//      the badge's key glyph — and its items sync;
//   2. "Stop trusting key…" withdraws it: the next launch is external again,
//      pin-mismatched, with the same offer back on the menu;
//   3. a release whose signature NAMES the accepted fixture key id but
//      carries different bytes is offered with the reused-id warning.
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import type { FixtureConfigSpec } from '../fixtures/config-builder';
import { EXTERNAL_DEMO_BIN_DIR } from '../fixtures/plugin-binaries';

const WEBSPACE = 'operator-trusted-key';
const OFFERED = 'topos-plugin-external-demo';
const REUSED = 'topos-plugin-external-demo-reused';
const SCRATCH_KEY_ID = 'acme-e2e';

const configSpec: FixtureConfigSpec = {
	sources: [
		{ id: 'control', plugin: 'topos-plugin-mock', displayName: 'Mock Control' },
		{
			id: 'offered',
			plugin: OFFERED,
			path: '/tmp/topos-e2e-17-offered-unused',
			displayName: 'Offered Demo',
			extras: { workspace_id: 'e2e' }
		},
		{
			id: 'reused',
			plugin: REUSED,
			path: '/tmp/topos-e2e-17-reused-unused',
			displayName: 'Reused Id Demo',
			extras: { workspace_id: 'e2e' }
		}
	],
	webspaces: [{ name: WEBSPACE, keywords: ['external-demo-proof', 'demo', 'fixture'] }],
	pluginBinaries: ['topos-plugin-mock'],
	externalBinaryLinks: [
		{ name: OFFERED, srcPath: `${EXTERNAL_DEMO_BIN_DIR}/${OFFERED}` },
		{ name: REUSED, srcPath: `${EXTERNAL_DEMO_BIN_DIR}/${OFFERED}` }
	],
	signedProvenanceBinaries: [
		{ name: OFFERED, scratchKeyID: SCRATCH_KEY_ID },
		{ name: REUSED, scratchKeyID: SCRATCH_KEY_ID, reusedID: true }
	],
	unpinnedExternalBinaries: [OFFERED, REUSED]
};
test.use({ configSpec });

interface SourceRow {
	name: string;
	tier: string;
	trusted_key?: string;
	offered_key?: {
		id: string;
		fingerprint: string;
		public_key: string;
		reused?: boolean;
	};
	launch_failure?: string;
	pinned_hash?: string;
	reachable: boolean;
}

async function sources(baseURL: string): Promise<Map<string, SourceRow>> {
	const res = await fetch(`${baseURL}/api/sources`);
	expect(res.ok).toBe(true);
	const body = (await res.json()) as { sources: SourceRow[] };
	return new Map(body.sources.map((s) => [s.name, s]));
}

test.describe('M2-R4: a developer key the operator trusts', () => {
	test('offer → trust the key: the plugin relaunches operator-trusted, unpinned, badged as yours; stop trusting returns it to external with the offer', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['control'], { logs: kernel.logs });

		// --- the kernel's facts before any consent ---
		let rows = await sources(kernel.baseURL);
		const offered = rows.get('offered');
		expect(offered, 'expected an "offered" entry').toBeTruthy();
		expect(offered?.tier).toBe('external');
		expect(offered?.launch_failure, 'unpinned external: refused at launch').toBe('pin_mismatch');
		expect(offered?.offered_key?.id, 'the scratch key is offered by id').toBe(SCRATCH_KEY_ID);
		expect(offered?.offered_key?.fingerprint ?? '').toMatch(/^[0-9a-f]{64}$/);
		expect(offered?.offered_key?.reused ?? false).toBe(false);
		const fingerprint = offered!.offered_key!.fingerprint;

		// --- the chip menu offers the key; the dialog shows the kernel's facts ---
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);
		const chip = page.getByRole('button', { name: /Offered Demo/ }).first();
		await expect(chip).toBeVisible();
		await page.getByRole('button', { name: 'Offered Demo actions' }).click();
		const menu = page.getByRole('menu');
		await expect(menu.getByRole('menuitem', { name: 'Trust signing key…' })).toBeVisible();
		await menu.getByRole('menuitem', { name: 'Trust signing key…' }).click();
		const dialog = page.locator('[data-trust-key-dialog="trust"]');
		await expect(dialog.getByRole('heading', { name: 'Trust this signing key?' })).toBeVisible();
		await expect(dialog.getByText(`Key id: ${SCRATCH_KEY_ID}`)).toBeVisible();
		await expect(dialog.getByText(`Fingerprint (SHA-256): ${fingerprint}`)).toBeVisible();
		await expect(dialog.getByText(/reused id/)).toHaveCount(0);
		await dialog.locator('#trust-key-note').fill('e2e scratch key');
		await dialog.getByRole('button', { name: 'Trust this key' }).click();
		await expect(dialog).toHaveCount(0);

		// --- the apply relaunched it on the operator's word ---
		await waitForFirstSync(kernel.baseURL, ['offered'], { logs: kernel.logs });
		rows = await sources(kernel.baseURL);
		const trusted = rows.get('offered');
		expect(trusted?.tier, 'trusted by the operator, not by topos').toBe('operator_trusted');
		expect(trusted?.trusted_key).toBe(SCRATCH_KEY_ID);
		expect(trusted?.pinned_hash ?? '', 'the evidence is the signature: no pin').toBe('');
		expect(trusted?.launch_failure ?? '').toBe('');
		expect(trusted?.reachable, `offered unreachable: ${JSON.stringify(trusted)}`).toBe(true);
		const cfg = (await (await fetch(`${kernel.baseURL}/api/config`)).json()) as {
			config: {
				plugins: {
					trusted_keys?: { id: string; public_key: string; note?: string }[];
				};
			};
		};
		expect(cfg.config.plugins.trusted_keys?.map((k) => k.id)).toEqual([SCRATCH_KEY_ID]);
		expect(cfg.config.plugins.trusted_keys?.[0].note).toBe('e2e scratch key');
		await page.reload();
		const chipAfter = page.getByRole('button', { name: /Offered Demo/ }).first();
		await expect(chipAfter.locator('[data-trust="operator"]')).toHaveCount(1);
		await expect(chipAfter.locator('svg.lucide-circle-alert')).toHaveCount(0);
		await page.getByRole('button', { name: 'Offered Demo actions' }).click();
		const menu2 = page.getByRole('menu');
		await expect(menu2.locator(`[data-trusted-key="${SCRATCH_KEY_ID}"]`)).toBeVisible();
		await expect(menu2.getByRole('menuitem', { name: 'Trust signing key…' })).toHaveCount(0);

		// --- stop trusting: back to external, offer back on the menu ---
		await menu2.getByRole('menuitem', { name: 'Stop trusting key…' }).click();
		const untrust = page.locator('[data-trust-key-dialog="untrust"]');
		await expect(untrust.getByRole('heading', { name: 'Stop trusting this key?' })).toBeVisible();
		await untrust.getByRole('button', { name: /Stop trusting/ }).click();
		await expect(untrust).toHaveCount(0);
		await expect
			.poll(async () => (await sources(kernel.baseURL)).get('offered')?.tier, {
				timeout: 15_000
			})
			.toBe('external');
		rows = await sources(kernel.baseURL);
		const again = rows.get('offered');
		expect(again?.launch_failure).toBe('pin_mismatch');
		expect(again?.offered_key?.id).toBe(SCRATCH_KEY_ID);
		const cfg2 = (await (await fetch(`${kernel.baseURL}/api/config`)).json()) as {
			config: { plugins: { trusted_keys?: unknown[] } };
		};
		expect(cfg2.config.plugins.trusted_keys ?? []).toEqual([]);
	});

	test('a signature naming the accepted key id with different bytes is offered as a reused id, with the warning', async ({
		page,
		kernel
	}) => {
		const rows = await sources(kernel.baseURL);
		const reused = rows.get('reused');
		expect(reused?.tier).toBe('external');
		expect(reused?.offered_key?.reused, 'the accepted id with other bytes is a reused id').toBe(
			true
		);
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);
		await page.getByRole('button', { name: 'Reused Id Demo actions' }).click();
		await page.getByRole('menu').getByRole('menuitem', { name: 'Trust signing key…' }).click();
		const dialog = page.locator('[data-trust-key-dialog="trust"]');
		await expect(dialog.getByText(/reused id/)).toBeVisible();
		await dialog.getByRole('button', { name: 'Cancel' }).click();
		await expect(dialog).toHaveCount(0);
	});
});
