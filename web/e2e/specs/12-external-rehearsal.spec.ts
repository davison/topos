// 12-05-PLAN.md Task 1: criterion 5's rehearsal — a real, full-featured
// source plugin (topos-plugin-filesystem, not a fixture-only proof binary)
// loaded from the external plugins directory, exactly like Phase 11's own
// mockstrict tracer (11-external-tier-badge.spec.ts) but against real
// source work instead of a bare Describe/Match stub. The load-bearing
// fixture choice: this spec deliberately omits topos-plugin-filesystem from
// `pluginBinaries` (the trusted-dir list).
//
// This spec does NOT re-prove Phase 11's own mechanics (the add-flow
// warning, the re-pin flow, the binary-changed state) — those already have
// specs (11-untrusted-add.spec.ts, 11-binary-changed-repin.spec.ts). It
// proves exactly one new thing: a real, full-featured source plugin behaves
// identically on the external path — same discovery, same pin
// verification, same launch, same sync, same badge.
//
// 16-03-PLAN.md Task 2 (gap closure, D-11): the external-tier binary is
// linked under the RENAMED destination `topos-plugin-filesystem-untrusted`
// via `externalBinaryLinks`, rather than under its own name via the
// plain `externalPluginBinaries` this spec originally used. Once trust
// became provenance-driven (Phase 16), `topos-plugin-filesystem` — a
// name the kernel's OWN link-time build manifest covers
// (`MANIFEST_E2E_BINARIES`, Makefile) — resolves TierTrusted from ANY
// directory it is found in, including the external one (success
// criterion 1: "trust is no longer a property of location"), which
// would silently make this spec prove nothing under its original
// fixture (the ORIGINAL header comment's own "Phase 11 D-11 would
// otherwise resolve it as trusted if it were present in both
// directories" concern turns out to apply even to a SINGLE placement,
// once tier stopped being directory-derived at all). The renamed
// destination is the SAME real filesystem plugin binary and behaves
// identically (no argv[0]/filename-dependent logic) — only its name is
// absent from the manifest, restoring the genuine "no evidence, external
// tier" case this spec exists to prove.
import { writeFileSync } from 'node:fs';
import { basename, join } from 'node:path';

import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { mkdtempCorpus } from '../fixtures/corpus';
import type { FixtureConfigSpec } from '../fixtures/config-builder';
import { PLUGIN_BIN_DIR } from '../fixtures/plugin-binaries';

// The same minimal-but-structurally-valid PDF fixture 12-filesystem-tracer
// and 12-filesystem-recursion use — a real %PDF- header through a real
// %%EOF trailer, proving the item that reaches the browser is a genuine PDF
// rendition, not an arbitrary byte blob wearing an application/pdf label.
const MINIMAL_PDF = `%PDF-1.4
1 0 obj
<< /Type /Catalog /Pages 2 0 R >>
endobj
2 0 obj
<< /Type /Pages /Kids [3 0 R] /Count 1 >>
endobj
3 0 obj
<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] /Resources << >> >>
endobj
trailer
<< /Size 4 /Root 1 0 R >>
%%EOF
`;

const MARKDOWN_CONTENT = '# Household notes\n\nSomething to remember.\n';

const PDF_FILENAME = 'invoice.pdf';
const MARKDOWN_FILENAME = 'notes.md';
const FILESYSTEM_BINARY = 'topos-plugin-filesystem-untrusted';
const SOURCE_ID = 'docs-folder';
const DISPLAY_NAME = 'Household Docs (external)';
const WEBSPACE = 'household-external';

// Module-scope temp corpus directory (D-03: state is seeded before kernel
// boot) — its own base name is exactly the D-05 folder-vocabulary label the
// filesystem plugin emits for a top-level file, so the webspace below
// participates via the ordinary keywords fallback with no explicit match
// block, mirroring 12-filesystem-tracer.spec.ts's own convention.
const corpusDir = mkdtempCorpus('topos-e2e-fs-external-');
writeFileSync(join(corpusDir, PDF_FILENAME), MINIMAL_PDF);
writeFileSync(join(corpusDir, MARKDOWN_FILENAME), MARKDOWN_CONTENT);

const configSpec: FixtureConfigSpec = {
	sources: [
		{
			id: SOURCE_ID,
			plugin: FILESYSTEM_BINARY,
			path: corpusDir,
			displayName: DISPLAY_NAME
		}
	],
	webspaces: [{ name: WEBSPACE, keywords: [basename(corpusDir)] }],
	// The load-bearing part (see file header): the filesystem binary is
	// linked ONLY here, under a RENAMED destination the kernel's own
	// link-time build manifest does not cover (Phase 16, D-11) — never
	// in pluginBinaries below, and never under its own name.
	pluginBinaries: [],
	externalBinaryLinks: [
		{ name: FILESYSTEM_BINARY, srcPath: join(PLUGIN_BIN_DIR, 'topos-plugin-filesystem') }
	]
};

test.use({ configSpec });

test.describe('12-05 Task 1: external-tier rehearsal — the real filesystem binary, loaded untrusted', () => {
	test('GET /api/sources reports the instance reachable with tier external', async ({ kernel }) => {
		await waitForFirstSync(kernel.baseURL, [SOURCE_ID], { logs: kernel.logs });

		const res = await fetch(`${kernel.baseURL}/api/sources`);
		expect(res.ok, `sources request failed: ${res.status}`).toBe(true);
		const body = (await res.json()) as {
			sources: Array<{ name: string; tier: string; reachable: boolean; last_error: string }>;
		};

		const byName = new Map(body.sources.map((s) => [s.name, s]));
		const instance = byName.get(SOURCE_ID);

		expect(instance, `expected a "${SOURCE_ID}" entry in GET /api/sources`).toBeTruthy();
		expect(instance?.tier).toBe('external');
		expect(instance?.reachable, `${SOURCE_ID} unreachable: ${instance?.last_error}`).toBe(true);
	});

	test('the webspace stream carries the same item set a trusted-tier corpus would, including the rewritten open-route link', async ({
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [SOURCE_ID], { logs: kernel.logs });

		const streamRes = await fetch(`${kernel.baseURL}/api/webspaces/${WEBSPACE}/stream`);
		expect(streamRes.ok, `stream request failed: ${streamRes.status}`).toBe(true);
		const stream = (await streamRes.json()) as {
			items: Array<{ id: string; source_id: string; link: { url: string } }>;
		};

		expect(
			stream.items,
			`expected exactly two items (pdf + markdown), got: ${JSON.stringify(stream.items)}`
		).toHaveLength(2);

		const bySourceId = new Map(stream.items.map((it) => [it.source_id, it]));
		const pdfItem = bySourceId.get(PDF_FILENAME);
		const mdItem = bySourceId.get(MARKDOWN_FILENAME);
		expect(pdfItem, `expected a "${PDF_FILENAME}" item, got: ${JSON.stringify(stream.items)}`).toBeTruthy();
		expect(mdItem, `expected a "${MARKDOWN_FILENAME}" item, got: ${JSON.stringify(stream.items)}`).toBeTruthy();

		// The plugin -> kernel file://-scheme rewrite works identically on the
		// external path — link.url is the loopback open route, never the
		// plugin's raw file:// deep_link value. Proves syncing is identical
		// across tiers (criterion 5's "loads and syncs identically" half).
		expect(pdfItem?.link.url).toBe(`/api/items/${pdfItem?.id}/open`);
		expect(pdfItem?.link.url.startsWith('file://')).toBe(false);
	});

	test('the source chip carries the untrusted badge, and it is the only chip that does', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [SOURCE_ID], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const chip = page.getByRole('button', { name: DISPLAY_NAME, exact: true });
		await expect(chip).toBeVisible();

		// The trust badge glyph renders as CircleAlert (@lucide/svelte classes
		// it `lucide-circle-alert`), mirroring 11-external-tier-badge.spec.ts's
		// own selector convention.
		const badgeGlyph = 'svg.lucide-circle-alert';
		await expect(
			chip.locator(badgeGlyph),
			'expected the external-tier chip to carry exactly one trust badge'
		).toHaveCount(1);

		// Header-wide total, scoped with Playwright's `:visible` pseudo-class
		// (WebspaceHeader.svelte renders an invisible off-screen measurement
		// clone of every chip for overflow-width calculation) — fails loudly
		// if any OTHER chip also carries a badge, not just vacuously passing
		// on this one instance's scoped assertion.
		await expect(page.locator(`${badgeGlyph}:visible`)).toHaveCount(1);

		// 14-02-PLAN.md Task 2 (14-UI-SPEC.md G1, option-b): the disclosure no
		// longer renders through a native `title` attribute — it is exposed as
		// the button's accessible DESCRIPTION via a visually-hidden sr-only
		// span wired through aria-describedby.
		await expect(chip).toHaveAccessibleDescription(/untrusted external plugin/);
	});
});
