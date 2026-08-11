// 09-06-PLAN.md Task 3's browser proof for the relocated "Reload config"
// action (09-UI-SPEC.md Fix 7): the switcher's three-item menu order, a
// real reload that makes a hand-edited config.toml visible without a page
// reload, a failed reload's modal-less header Alert (no dialog), and the
// absence of a second entry point inside Manage sources….
//
// Each test below asserts by NAME/ORDER rather than by fixed count or
// position, deliberately robust to another test in this file (running in
// the same worker, per D-02's worker-scoped-kernel-per-configSpec fixture)
// having already hand-edited config.toml and added a webspace — see the
// menu-structure test's own comment for why.
import { writeFileSync } from 'node:fs';
import { stringify } from 'smol-toml';
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { mockInstances, webspacesWithKeywords, type FixtureConfigSpec } from '../fixtures/config-builder';
import { readConfigToml } from '../fixtures/toml';

const configSpec: FixtureConfigSpec = {
	sources: mockInstances(1),
	webspaces: webspacesWithKeywords(['existing'], ['demo'])
};

test.use({ configSpec });

test.describe('09-06 Task 3: switcher-root Reload config — menu order, a real reload, its modal-less failure, and no second entry point', () => {
	test('menu structure: New webspace, Reload config, Manage sources… in that order, after the webspace list', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/existing`);

		await page.getByRole('button', { name: 'existing' }).click();
		const menu = page.locator('[data-slot="dropdown-menu-content"]');
		await expect(menu).toBeVisible();

		// Asserted by ORDER (indexOf comparisons), not by a fixed slice
		// offset past the webspace list — another test in this file may
		// already have hand-edited config.toml to add a webspace before
		// this one runs (both share one worker-scoped kernel per D-02),
		// which would shift a position-based offset but never the static
		// trio's own relative order. allTextContents() returns each
		// element's raw (un-normalised) textContent — unlike the
		// accessible-name computation Playwright's role locator itself
		// uses, it does NOT collapse the surrounding template whitespace
		// each multi-line DropdownMenuItem's markup leaves around its icon
		// and label — so every entry is normalised before comparison.
		const rawTexts = await menu.getByRole('menuitem').allTextContents();
		const texts = rawTexts.map((t) => t.trim().replace(/\s+/g, ' '));
		const createIndex = texts.indexOf('New webspace');
		const reloadIndex = texts.indexOf('Reload config');
		const manageIndex = texts.indexOf('Manage sources…');
		expect(createIndex, 'expected to find "New webspace"').toBeGreaterThanOrEqual(0);
		expect(reloadIndex, 'expected "Reload config" after "New webspace"').toBeGreaterThan(createIndex);
		expect(manageIndex, 'expected "Manage sources…" after "Reload config"').toBeGreaterThan(
			reloadIndex
		);
		// Manage sources… is the LAST item — no fourth static action trails it.
		expect(texts[texts.length - 1]).toBe('Manage sources…');

		await page.keyboard.press('Escape');
		await expect(menu).toHaveCount(0);
	});

	test('reload works: a hand-edited config.toml becomes visible without a page reload, and the menu closes', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/existing`);

		// The stronger of the two assertion shapes the plan offers: rather
		// than merely observing that a request reached the kernel, hand-edit
		// config.toml directly (bypassing the running kernel entirely, the
		// exact scenario "Reload config" exists for) and prove the change
		// becomes visible in the UI after clicking it — with no page reload
		// anywhere in this test.
		const doc = readConfigToml(kernel.configPath);
		const webspaces = doc.webspaces as Record<string, unknown>;
		webspaces['reload-proof'] = { keywords: ['demo'] };
		writeFileSync(kernel.configPath, stringify(doc), 'utf-8');

		const trigger = page.getByRole('button', { name: 'existing' });
		await trigger.click();
		const menu = page.locator('[data-slot="dropdown-menu-content"]');
		await expect(menu).toBeVisible();

		await page.getByRole('menuitem', { name: 'Reload config' }).click();

		// The menu closes on select — the same default DropdownMenuItem
		// behavior every other item in this menu already exhibits.
		await expect(menu).toHaveCount(0);

		// The new webspace is now in the switcher's own list — proof the
		// reload actually re-read config.toml and the UI reflects it live.
		await trigger.click();
		await expect(page.getByRole('menuitem', { name: 'reload-proof' })).toBeVisible();
	});

	test('reload failure surfaces without a modal: a destructive header Alert, no dialog', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/existing`);

		await page.route('**/api/config/reload', async (route) => {
			await route.fulfill({
				status: 422,
				contentType: 'application/json',
				body: JSON.stringify({
					schema_version: 1,
					error: { code: 'config_invalid', message: 'e2e-injected reload failure' }
				})
			});
		});

		await page.getByRole('button', { name: 'existing' }).click();
		await page.getByRole('menuitem', { name: 'Reload config' }).click();

		// The fixed reassurance sentence (Copywriting Contract), carrying
		// the kernel's verbatim message ahead of it, rendered as a
		// destructive Alert in the header region — the same modal-less
		// pattern chip-remove and filter writes already use.
		await expect(
			page.getByText('e2e-injected reload failure. The previous configuration is still running.')
		).toBeVisible();

		// No dialog opened — the failure is the header's own Alert, never a
		// second surface.
		await expect(page.getByRole('dialog')).toHaveCount(0);

		await page.unroute('**/api/config/reload');
	});

	test('no second entry point: Manage sources… offers no reload control', async ({ page, kernel }) => {
		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/existing`);

		await page.getByRole('button', { name: 'existing' }).click();
		await page.getByRole('menuitem', { name: 'Manage sources…' }).click();

		const dialog = page.getByRole('dialog');
		await expect(dialog).toBeVisible();
		await expect(dialog.getByText('Reload config')).toHaveCount(0);
		await expect(dialog.getByRole('button', { name: 'Reload config' })).toHaveCount(0);

		await page.keyboard.press('Escape');
		await expect(dialog).toHaveCount(0);
	});
});
