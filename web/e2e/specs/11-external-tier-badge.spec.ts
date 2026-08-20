// Phase 11's own tracer proof, end to end (11-01-PLAN.md Task 3): a real
// kernel booted by the hermetic harness discovers one plugin from EACH
// tier — topos-plugin-mock (trusted, PLUGIN_BIN_DIR) and mockstrict
// (external, a SECOND fixture directory this plan's config-builder/
// kernel fixture changes introduce) — syncs both, and the browser shows
// the untrusted trust badge on exactly the external-tier chip. Proves
// PLUG-06 (external discovery/launch) and the chip half of PLUG-08
// (trust badge) together, on one thin path, exactly as this whole plan's
// <objective> states.
//
// 16-03-PLAN.md Task 2 (gap closure, D-11): the external-tier participant
// is linked under the RENAMED destination `topos-plugin-mockstrict-untrusted`
// via `externalBinaryLinks`, rather than under its own name via the
// plain `externalPluginBinaries` this spec originally used. Once trust
// became provenance-driven (Phase 16), `topos-plugin-mockstrict` — a
// name the kernel's OWN link-time build manifest covers
// (`MANIFEST_E2E_BINARIES`, Makefile) — resolves TierTrusted from ANY
// directory, including the external one (success criterion 1: "trust is
// no longer a property of location"), which would make this spec prove
// nothing under its original fixture. The renamed destination is the
// SAME real mockstrict binary and behaves identically (its own Go
// binary carries no argv[0]/filename-dependent logic) — only its name is
// absent from the manifest, restoring the genuine "no evidence, external
// tier" case this spec exists to prove.
import { join } from 'node:path';
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import type { FixtureConfigSpec } from '../fixtures/config-builder';
import { PLUGIN_BIN_DIR } from '../fixtures/plugin-binaries';

const WEBSPACE = 'trust-boundary';
const MOCKSTRICT_EXTERNAL_BINARY = 'topos-plugin-mockstrict-untrusted';

// keywords ['demo', 'fixture'] deliberately spans both corpora's declared
// vocabulary field: mock's own field is "labels" (its corpus carries
// "demo"), mockstrict's own field is "tags" (its corpus carries
// "fixture" on two of its three items) — the D-01 keywords fallback
// applies the SAME list against whichever field each participating
// instance declares, so one webspace matches both without an explicit
// per-instance match block.
const configSpec: FixtureConfigSpec = {
	sources: [
		{ id: 'mock', plugin: 'topos-plugin-mock', displayName: 'Mock Source' },
		{
			id: 'mockstrict',
			plugin: MOCKSTRICT_EXTERNAL_BINARY,
			path: '/tmp/e2e-11-mockstrict-unused',
			displayName: 'Mockstrict Corpus'
		}
	],
	webspaces: [{ name: WEBSPACE, keywords: ['demo', 'fixture'] }],
	pluginBinaries: ['topos-plugin-mock'],
	externalBinaryLinks: [
		{ name: MOCKSTRICT_EXTERNAL_BINARY, srcPath: join(PLUGIN_BIN_DIR, 'topos-plugin-mockstrict') }
	]
};

test.use({ configSpec });

test.describe('11-01 Task 3: two-tier discovery end to end — one trusted, one external, exactly one badged chip', () => {
	test('GET /api/sources reports tier "trusted" for the mock instance and "external" for the mockstrict instance', async ({
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock', 'mockstrict'], { logs: kernel.logs });

		const res = await fetch(`${kernel.baseURL}/api/sources`);
		expect(res.ok).toBe(true);
		const body = (await res.json()) as {
			sources: Array<{ name: string; tier: string; reachable: boolean; last_error: string }>;
		};

		const byName = new Map(body.sources.map((s) => [s.name, s]));
		const mock = byName.get('mock');
		const mockstrict = byName.get('mockstrict');

		expect(mock, 'expected a "mock" entry in GET /api/sources').toBeTruthy();
		expect(mockstrict, 'expected a "mockstrict" entry in GET /api/sources').toBeTruthy();
		expect(mock?.tier).toBe('trusted');
		expect(mockstrict?.tier).toBe('external');

		// Neither instance reports a launch failure — both binaries were
		// discovered and launched successfully from their own tier.
		expect(mock?.reachable, `mock unreachable: ${mock?.last_error}`).toBe(true);
		expect(mockstrict?.reachable, `mockstrict unreachable: ${mockstrict?.last_error}`).toBe(true);
	});

	test('exactly one chip renders the trust badge — the external-tier mockstrict chip, never the trusted mock chip', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock', 'mockstrict'], { logs: kernel.logs });

		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		// exact: true, mirroring 09-plugin-icon.spec.ts's own convention — the
		// chip's filter button, distinct from its sibling "{name} actions"
		// overflow-trigger control, whose accessible name also contains this
		// substring.
		const mockChip = page.getByRole('button', { name: 'Mock Source', exact: true });
		const mockstrictChip = page.getByRole('button', { name: 'Mockstrict Corpus', exact: true });
		await expect(mockChip).toBeVisible();
		await expect(mockstrictChip).toBeVisible();

		// The trust badge glyph renders as CircleAlert, which @lucide/svelte's
		// shared Icon.svelte always classes `lucide-{name}` — `lucide-circle-alert`
		// is therefore a stable selector for the badge itself, independent of
		// its aria-hidden="true" backdrop wrapper (which getByRole locators
		// above correctly ignore for accessible-name purposes).
		const badgeGlyph = 'svg.lucide-circle-alert';

		await expect(
			mockstrictChip.locator(badgeGlyph),
			'expected exactly one trust badge on the mockstrict (external-tier) chip'
		).toHaveCount(1);
		await expect(
			mockChip.locator(badgeGlyph),
			'expected NO trust badge on the mock (trusted-tier) chip — a trusted chip must render byte-identically to before this phase'
		).toHaveCount(0);

		// Fails loudly (not vacuously) if the VISIBLE badge count across the
		// whole header row is anything other than exactly one — names which
		// instance carries it via the two scoped assertions above, and this
		// assertion independently pins the header-wide total. Scoped with
		// Playwright's own `:visible` pseudo-class: WebspaceHeader.svelte
		// renders an invisible (`visibility: hidden`) off-screen measurement
		// clone of every chip for overflow-width calculation (09.1-01-PLAN.md),
		// so an unscoped page-wide count would double every real badge.
		await expect(page.locator(`${badgeGlyph}:visible`)).toHaveCount(1);
	});

	test('the mockstrict chip health description discloses "untrusted external plugin"; the mock chip health description does not', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock', 'mockstrict'], { logs: kernel.logs });

		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const mockChip = page.getByRole('button', { name: 'Mock Source', exact: true });
		const mockstrictChip = page.getByRole('button', { name: 'Mockstrict Corpus', exact: true });

		// 14-02-PLAN.md Task 2 (14-UI-SPEC.md G1, option-b): the disclosure no
		// longer renders through a native `title` attribute — it is exposed as
		// the button's accessible DESCRIPTION via a visually-hidden sr-only
		// span wired through aria-describedby.
		await expect(mockstrictChip).toHaveAccessibleDescription(/untrusted external plugin/);
		await expect(mockChip).not.toHaveAccessibleDescription(/untrusted external plugin/);
	});
});
