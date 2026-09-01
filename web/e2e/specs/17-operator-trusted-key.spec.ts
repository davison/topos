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
// Two more copies of the same binary, signed with the scratch key and NOT
// configured as sources: the picker offers them as new, so the add-source
// interstitial's two consents can be driven for real.
const FRESH_PIN = 'topos-plugin-external-demo-pinonly';
const FRESH_KEY = 'topos-plugin-external-demo-trustkey';
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
		{ name: REUSED, srcPath: `${EXTERNAL_DEMO_BIN_DIR}/${OFFERED}` },
		{ name: FRESH_PIN, srcPath: `${EXTERNAL_DEMO_BIN_DIR}/${OFFERED}` },
		{ name: FRESH_KEY, srcPath: `${EXTERNAL_DEMO_BIN_DIR}/${OFFERED}` }
	],
	signedProvenanceBinaries: [
		{ name: OFFERED, scratchKeyID: SCRATCH_KEY_ID },
		{ name: REUSED, scratchKeyID: SCRATCH_KEY_ID, reusedID: true },
		{ name: FRESH_PIN, scratchKeyID: SCRATCH_KEY_ID },
		{ name: FRESH_KEY, scratchKeyID: SCRATCH_KEY_ID }
	],
	unpinnedExternalBinaries: [OFFERED, REUSED, FRESH_PIN, FRESH_KEY]
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

	// The add-source interstitial (#57 plan): the trial launch reports the
	// offer before the source exists; "pin this binary only" is the
	// default and writes a pin, never a key; choosing the key writes the
	// [[plugins.trusted_keys]] entry, never a pin — and the source lands
	// operator-trusted.
	async function addOffered(page: import('@playwright/test').Page, binary: string, displayName: string, choice: 'pin' | 'key') {
		await page.getByRole('button', { name: 'Add source' }).click();
		// The picker names a plugin type by pluginTypeLabel — the binary name
		// without its prefix, title-cased per dash segment — not by the
		// binary itself; the four copies of the demo binary share a Describe
		// display name, so the label is what tells them apart.
		const label = binary
			.replace('topos-plugin-', '')
			.split('-')
			.map((w) => w.charAt(0).toUpperCase() + w.slice(1))
			.join(' ');
		await expect(page.getByText('Install a new source', { exact: true })).toHaveCount(1);
		const tile = page.getByRole('button', { name: new RegExp(label) }).first();
		await expect(tile).toBeVisible();
		await tile.click();
		const dialog = page.getByRole('dialog');
		await dialog.locator('#conn-display_name').fill(displayName);
		// The connect form's fields are keyed by binary name (plugin-fields.ts);
		// these copies are unknown names, so fill whatever the generic form
		// and the plugin's own declared extras present.
		// An unknown plugin type gets the generic form: the kernel-known keys
		// sit under Advanced options — the demo plugin's fatal-guard requires
		// `path`, exactly the third-party case the generic form exists for.
		await dialog.getByRole('button', { name: 'Advanced options' }).click();
		await dialog.locator('#conn-path').fill(`/tmp/topos-e2e-17-${binary}`);
		if (await dialog.locator('#extra-workspace_id').count()) await dialog.locator('#extra-workspace_id').fill('e2e');
		await dialog.getByRole('button', { name: 'Next' }).click();
		await expect(dialog.getByRole('heading', { name: 'Add an untrusted source' })).toBeVisible();
		const offer = dialog.locator(`fieldset[data-offered-key="${SCRATCH_KEY_ID}"]`);
		await expect(offer, 'the interstitial shows the offer the trial launch reported').toBeVisible();
		await expect(offer.getByText(`Key id: ${SCRATCH_KEY_ID}`)).toBeVisible();
		await expect(offer.getByText(/Fingerprint \(SHA-256\): [0-9a-f]{64}/)).toBeVisible();
		await expect(offer.getByText(/reused id/)).toHaveCount(0);
		await expect(offer.locator('input[value="pin"]'), 'pin this binary only is the default').toBeChecked();
		if (choice === 'key') await offer.locator('input[value="key"]').check();
		await dialog.locator('#untrusted-confirm-typed').fill(binary);
		await dialog.getByRole('button', { name: 'Add untrusted source' }).click();
		await expect(dialog.getByRole('heading', { name: `Match settings for ${WEBSPACE}` })).toBeVisible();
		await dialog.getByRole('button', { name: 'Add source' }).click();
		await expect(page.getByRole('dialog')).toHaveCount(0);
	}

	test('add-source interstitial, offer → pin this binary only: a pin is written and no key', async ({ page, kernel }) => {
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);
		await addOffered(page, FRESH_PIN, 'Pin Only Demo', 'pin');
		const cfg = (await (await fetch(`${kernel.baseURL}/api/config`)).json()) as {
			config: { plugins: { pins?: Record<string, string>; trusted_keys?: { id: string }[] } };
		};
		expect(cfg.config.plugins.pins?.[FRESH_PIN] ?? '').toMatch(/^[0-9a-f]{64}$/);
		expect((cfg.config.plugins.trusted_keys ?? []).map((k) => k.id)).not.toContain(SCRATCH_KEY_ID);
		await expect
			.poll(async () => (await sources(kernel.baseURL)).get('pin-only-demo')?.tier, { timeout: 15_000 })
			.toBe('external');
	});

	test('add-source interstitial, offer → trust the key: the entry is written, no pin, and the source lands operator-trusted', async ({ page, kernel }) => {
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);
		await addOffered(page, FRESH_KEY, 'Trust Key Demo', 'key');
		const cfg = (await (await fetch(`${kernel.baseURL}/api/config`)).json()) as {
			config: { plugins: { pins?: Record<string, string>; trusted_keys?: { id: string; public_key: string }[] } };
		};
		expect(cfg.config.plugins.pins?.[FRESH_KEY], 'the key consent writes no pin').toBeUndefined();
		expect((cfg.config.plugins.trusted_keys ?? []).map((k) => k.id)).toContain(SCRATCH_KEY_ID);
		await expect
			.poll(async () => (await sources(kernel.baseURL)).get('trust-key-demo')?.tier, { timeout: 15_000 })
			.toBe('operator_trusted');
		const row = (await sources(kernel.baseURL)).get('trust-key-demo');
		expect(row?.trusted_key).toBe(SCRATCH_KEY_ID);
		expect(row?.pinned_hash ?? '').toBe('');
	});
});

