// 11-06-PLAN.md Task 3: swap a real, genuinely out-of-repo plugin binary
// mid-run, see the kernel catch it and refuse the launch by name, prove
// it's caught (and recovered from) through a real browser session rather
// than only in Go (kernel/supervisor/externalproof_test.go already proves
// the kernel-side refusal; this spec proves the visible half — ROADMAP
// success criterion 3, PLUG-07/PLUG-08, 11-UI-SPEC.md E4).
//
// Two sources participate in one webspace: mock-control (trusted, the
// control) and demo-instance (topos-plugin-external-demo, 11-04-PLAN.md's
// out-of-repo proof plugin, linked from bin/plugins-external — never
// bin/plugins — with a correct pin computed at fixture-build time). After
// the first sync both are healthy. The spec then:
//
//   1. Replaces ONLY the fixture's own external-directory entry with
//      tampered bytes (the shared build output at bin/plugins-external/ is
//      never touched — it stays a symlink source read from, never written
//      to).
//   2. Triggers a real config write through the browser (the chip's own
//      Edit connection… flow, changing only the display name) and asserts
//      the save SUCCEEDS — a pin mismatch on one instance must never
//      reject an unrelated save (T-11-33).
//   3. Asserts the external chip alone shows the destructive tone and the
//      binary-changed tooltip, that the control chip stays healthy, and
//      that the control source's own item still streams — a degraded
//      source is not an outage (g-08-3's own precedent, one level up the
//      trust boundary).
//   4. Opens the chip's menu, asserts Trust updated binary… is the FIRST
//      item and Refresh now is disabled.
//   5. Opens the re-pin dialog, asserts the previously-pinned and
//      currently-on-disk hashes are the EXACT values this spec
//      independently computed (never a vacuous non-empty check) and that
//      they differ, confirms, and asserts the persisted pin equals the new
//      on-disk hash and the chip leaves the failure state.
import { createHash } from 'node:crypto';
import { chmodSync, readFileSync, rmSync, writeFileSync } from 'node:fs';
import { join } from 'node:path';
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { attachedWebspace, type FixtureConfigSpec, type FixtureSourceSpec } from '../fixtures/config-builder';
import { EXTERNAL_DEMO_BIN_DIR } from '../fixtures/plugin-binaries';

const WEBSPACE = 'repin-proof';
const EXTERNAL_BINARY = 'topos-plugin-external-demo';
const RENAMED_DISPLAY_NAME = 'Demo Proof Renamed';

const sources: FixtureSourceSpec[] = [
	{ id: 'mock-control', plugin: 'topos-plugin-mock', displayName: 'Mock Control' },
	{
		id: 'demo-instance',
		plugin: EXTERNAL_BINARY,
		displayName: 'Demo Proof',
		path: '/tmp/topos-e2e-11-repin-demo-instance',
		// workspace_id (11-04-PLAN.md's own Describe response) is a
		// REQUIRED declared extras field — seeded here so Task 3's
		// Edit connection… save (step 3) isn't blocked by this spec's own
		// unrelated required-field validation, which has nothing to do
		// with the pin-mismatch behavior this spec actually proves.
		extras: { workspace_id: 'acme-42' }
	}
];

const configSpec: FixtureConfigSpec = {
	sources,
	webspaces: [
		attachedWebspace(WEBSPACE, ['mock-control', 'demo-instance'], {
			'mock-control': { labels: ['demo'] },
			'demo-instance': { labels: ['external-demo-proof'] }
		})
	],
	pluginBinaries: ['topos-plugin-mock'],
	externalPluginBinaries: [EXTERNAL_BINARY],
	externalPluginBinariesSrcDir: EXTERNAL_DEMO_BIN_DIR
};

test.use({ configSpec });

test.describe('11-06 Task 3: swap a real binary mid-run, see it caught, re-pin, recover', () => {
	test('a tampered external binary refuses to launch by name, an unrelated save still succeeds, and re-pinning recovers the chip', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock-control', 'demo-instance'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const mockChip = page.getByRole('button', { name: 'Mock Control', exact: true });
		const demoChip = page.getByRole('button', { name: 'Demo Proof', exact: true });

		// --- 1. Both chips healthy; the external chip carries the trust
		// badge (its own tier alone, independent of the pin state this spec
		// is about to break).
		await expect(mockChip).toBeVisible();
		await expect(demoChip).toBeVisible();
		await expect(mockChip.locator('span.size-2')).toHaveClass(/bg-success/);
		await expect(demoChip.locator('span.size-2')).toHaveClass(/bg-success/);
		await expect(
			demoChip.locator('svg.lucide-circle-alert:visible'),
			'expected the external-tier chip to carry the trust badge before this spec touches anything'
		).toHaveCount(1);

		// --- 2. Replace ONLY the fixture's own external-directory entry —
		// never bin/plugins-external/ itself, the shared build output every
		// other worker/spec reads from. Remove the symlink first: writing
		// through a live symlink without removing it would mutate the
		// target file the symlink points at, which is exactly the shared
		// artifact this spec must never touch.
		const originalBytes = readFileSync(join(EXTERNAL_DEMO_BIN_DIR, EXTERNAL_BINARY));
		const tamperedBytes = Buffer.concat([originalBytes, Buffer.from([0])]);
		const originalHash = createHash('sha256').update(originalBytes).digest('hex');
		const tamperedHash = createHash('sha256').update(tamperedBytes).digest('hex');
		expect(
			originalHash,
			'sanity: the tampered copy must hash differently from the original, or this spec would prove nothing'
		).not.toBe(tamperedHash);

		const externalDir = join(kernel.tmpDir, 'plugins-external');
		const linkPath = join(externalDir, EXTERNAL_BINARY);
		rmSync(linkPath, { force: true });
		writeFileSync(linkPath, tamperedBytes);
		chmodSync(linkPath, 0o755);

		// --- 3. Trigger a relaunch through a REAL config write, via the
		// chip's own Edit connection… flow — changing only the display
		// name. Assert the save SUCCEEDS: a pin mismatch on demo-instance
		// must never reject this unrelated write (T-11-33).
		// The chip's "…actions" trigger is a SIBLING of the filter button
		// above (SourceChip.svelte's own [filter-button][actions-trigger]
		// markup shape), not a descendant — located at the page level, same
		// as 09-chip-menu.spec.ts's own precedent.
		await page.getByRole('button', { name: 'Demo Proof actions' }).click();
		await page.getByRole('menuitem', { name: 'Edit connection…' }).click();

		const editDialog = page.getByRole('dialog');
		await expect(editDialog.getByRole('heading', { name: 'Edit connection' })).toBeVisible();
		const displayNameInput = editDialog.locator('#conn-display_name');
		await displayNameInput.fill(RENAMED_DISPLAY_NAME);
		await editDialog.getByRole('button', { name: 'Save changes' }).click();

		await expect(
			page.getByRole('dialog'),
			'expected the Edit connection… save to succeed and close the dialog despite the tampered binary'
		).toHaveCount(0);
		await expect(page.getByText('Config changed on disk')).toHaveCount(0);

		// --- 4. The external chip (now under its renamed display name)
		// shows the destructive tone and the binary-changed tooltip,
		// together (E4's "populated (chip coupling)" contract) — the
		// control chip stays healthy, and the control source's own item
		// still streams (a degraded source is not an outage).
		const renamedChip = page.getByRole('button', { name: RENAMED_DISPLAY_NAME, exact: true });
		await expect(renamedChip).toBeVisible();
		await expect(renamedChip.locator('span.size-2')).toHaveClass(/bg-destructive/, {
			timeout: 15_000
		});
		await expect(renamedChip).toHaveAttribute(
			'title',
			`${RENAMED_DISPLAY_NAME} — binary changed since it was trusted`
		);

		await expect(mockChip).toBeVisible();
		await expect(mockChip.locator('span.size-2')).toHaveClass(/bg-success/);
		await expect(page.getByText('Welcome to the mock source')).toBeVisible();

		// --- 5. Open the chip menu: Trust updated binary… is the FIRST
		// item, Refresh now is disabled (there is nothing running to
		// refresh).
		await page.getByRole('button', { name: `${RENAMED_DISPLAY_NAME} actions` }).click();
		const menu = page.getByRole('menu');
		await expect(menu).toBeVisible();
		const menuItems = (await menu.getByRole('menuitem').allTextContents()).map((t) => t.trim());
		expect(
			menuItems[0],
			'expected "Trust updated binary…" to be the first menu item while the mismatch signal is set'
		).toBe('Trust updated binary…');
		await expect(menu.getByRole('menuitem', { name: 'Refresh now' })).toBeDisabled();

		// --- 6. Open the re-pin dialog: both hashes render as the EXACT
		// values this spec independently computed (never a vacuous
		// non-empty check), and they differ.
		await menu.getByRole('menuitem', { name: 'Trust updated binary…' }).click();
		const repinDialog = page.getByRole('dialog');
		await expect(repinDialog.getByRole('heading', { name: 'Binary changed' })).toBeVisible();

		const shortOriginal = `${originalHash.slice(0, 12)}…`;
		await expect(
			repinDialog.getByText(`Previously pinned: ${shortOriginal}`),
			'expected the previously-pinned line to show the short form of the ORIGINAL hash this fixture pinned at build time'
		).toBeVisible();
		await expect(
			repinDialog.getByText(`Currently on disk: ${tamperedHash}`),
			'expected the currently-on-disk line to show the FULL tampered hash — the exact bytes now on disk'
		).toBeVisible();

		// --- 7. Confirm: the save succeeds, the persisted pin equals the
		// NEW on-disk hash, and the chip returns to a non-failure state
		// without a kernel restart.
		await repinDialog.getByRole('button', { name: 'Trust updated binary' }).click();
		await expect(page.getByRole('dialog')).toHaveCount(0);

		const configRes = await fetch(`${kernel.baseURL}/api/config`);
		expect(configRes.ok).toBe(true);
		const configBody = (await configRes.json()) as {
			config: { plugins: { pins?: Record<string, string> } };
		};
		expect(
			configBody.config.plugins.pins?.[EXTERNAL_BINARY],
			'expected the persisted pin to equal the exact hash now on disk, echoed back verbatim (T-11-30)'
		).toBe(tamperedHash);

		await expect(renamedChip.locator('span.size-2')).toHaveClass(/bg-success/, { timeout: 15_000 });
		await expect(
			renamedChip,
			'expected the recovered chip\'s tooltip to no longer carry the binary-changed wording'
		).not.toHaveAttribute('title', /binary changed/);

		await page.getByRole('button', { name: `${RENAMED_DISPLAY_NAME} actions` }).click();
		const recoveredMenu = page.getByRole('menu');
		await expect(recoveredMenu.getByRole('menuitem', { name: 'Trust updated binary…' })).toHaveCount(0);
		await expect(recoveredMenu.getByRole('menuitem', { name: 'Refresh now' })).toBeEnabled();
	});
});
