// 14-04-PLAN.md Task 1: hermetic browser proof of the Google Drive plugin
// (topos-plugin-gdrive) on the untrusted external path — everything this
// phase's two success criteria (SRC-05/SRC-06) let a browser driver prove
// WITHOUT a real Google account. The binary this spec drives is not built
// by this repository, ever: it is the sole out-of-repo deliverable of the
// separate `davison/topos-plugin-gdrive` clean-room project (D-08,
// `.planning/phases/14-google-drive-source-built-out-of-repo/
// 14-PLUGIN-PRD.md` is the sole hand-off document into it; see
// `14-03-SUMMARY.md` for the built-binary hand-off evidence this task
// consumes).
//
// The load-bearing fixture choice, mirroring 12-external-rehearsal.spec.ts's
// own header exactly: `pluginBinaries: []` (the trusted directory stays
// legitimately empty, D-09) and the Drive binary is named ONLY in
// `externalPluginBinaries`, resolved from `externalPluginBinariesSrcDir` —
// never the default `PLUGIN_BIN_DIR` (`bin/plugins`), since this binary was
// never built into that directory and never should be. A binary present in
// BOTH directories resolves as trusted (Phase 11 D-11) — that would make
// this spec prove nothing.
//
// The source instance is configured deliberately credential-less: its two
// secret extras (`client_id`/`client_secret`) reference the environment
// variables `GDRIVE_CLIENT_ID`/`GDRIVE_CLIENT_SECRET`, which the fixture
// kernel's own explicit environment allowlist (web/e2e/fixtures/kernel.ts's
// `launchKernel`) deliberately never sets — `kernel/config/config.go`'s
// `expandEnv` therefore resolves both references to the empty string before
// the plugin subprocess ever sees them. This produces the plugin's "never
// authorized — no token file found" health state without a Google account,
// a network call, or a real OAuth flow: the one Drive behaviour a hermetic
// harness can prove, and the specific one this file's second test pins.
//
// Skipped, loudly, whenever TOPOS_GDRIVE_BIN is unset or does not name an
// existing file — see resolveDriveBin below. A silent skip (an empty file,
// or a file whose tests all vacuously pass with nothing configured) would
// be worse than no spec at all: a developer or CI run without the sibling
// checkout built would see nothing wrong.
import { existsSync } from 'node:fs';
import { basename, dirname } from 'node:path';

import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { webspacesWithKeywords, type FixtureConfigSpec } from '../fixtures/config-builder';

type DriveBinResolution = { missing: true; reason: string } | { missing: false; path: string };

// resolveDriveBin reads TOPOS_GDRIVE_BIN once, at module scope, so a
// missing/nonexistent binary is discovered before any fixture (in
// particular the worker-scoped `kernel` fixture, which would otherwise
// throw deep inside linkPluginBinaries/hashPluginBinary) is ever touched —
// test.skip below stops Playwright from instantiating that fixture at all
// when this returns `missing: true`.
function resolveDriveBin(): DriveBinResolution {
	const raw = process.env.TOPOS_GDRIVE_BIN;
	if (raw === undefined || raw === '') {
		return {
			missing: true,
			reason: 'TOPOS_GDRIVE_BIN is not set'
		};
	}
	if (!existsSync(raw)) {
		return {
			missing: true,
			reason: `TOPOS_GDRIVE_BIN ("${raw}") does not point to an existing file`
		};
	}
	return { missing: false, path: raw };
}

const driveBin = resolveDriveBin();

const SKIP_REASON = driveBin.missing
	? `${driveBin.reason} — this spec proves the real, out-of-repo topos-plugin-gdrive ` +
		'binary (built entirely by the separate topos-plugin-gdrive clean-room checkout, ' +
		'see .planning/phases/14-google-drive-source-built-out-of-repo/14-03-SUMMARY.md) on ' +
		'the untrusted external path, and has nothing to prove without that binary. Export ' +
		'TOPOS_GDRIVE_BIN to its absolute path (the sibling checkout\'s built binary) and run ' +
		'`make gdrive-external-rehearsal`, or set the TOPOS_GDRIVE_BIN make variable directly.'
	: '';

// DRIVE_BINARY/DRIVE_SRC_DIR are only ever read by tests below, all of
// which are skipped whenever driveBin.missing is true — the placeholder
// values in that branch are never used to link, hash, or launch anything.
const DRIVE_BINARY = driveBin.missing ? 'topos-plugin-gdrive' : basename(driveBin.path);
const DRIVE_SRC_DIR = driveBin.missing ? '' : dirname(driveBin.path);

const SOURCE_ID = 'gdrive-external';
const DISPLAY_NAME = 'Google Drive (external)';
const WEBSPACE = 'gdrive-external-rehearsal';

// The exact "never authorized" sentence from 14-UI-SPEC.md's G4 table /
// 14-PLUGIN-PRD.md's Health States table — copied verbatim, byte for byte
// (including the em dash), so `grep` for this distinctive clause across
// both files finds the identical text (14-04-PLAN.md Task 1 acceptance
// criterion).
const NOT_AUTHORIZED_SENTENCE =
	'Not authorized — run "topos-plugin-gdrive auth" in a terminal, then use this source\'s "Refresh now".';

// Credential-less on purpose — see file header. folder_id is a plain
// literal (never a secret), syntactically plausible but never validated by
// Describe (which must be side-effect-free per the PRD's Design Guidance).
//
// The webspace carries a non-empty `keywords` fallback (any literal — it
// never needs to actually match anything, since the credential-less source
// never returns items to match against) rather than a D-20 empty shell:
// kernel/correlate/correlate.go's `matchFieldsFor`/D-20 has-match-input
// rule means a source with NEITHER an explicit `match` block NOR a
// non-empty `keywords` fallback does not participate in that webspace at
// all, and `SyncSource` never calls `Match` for a non-participating pair —
// with a lone empty-shell webspace, the plugin's Match RPC would never be
// invoked, so the not-authorized sentence this file's second test asserts
// on would never have anywhere to come from, and the chip this file's
// fourth/fifth tests assert on would never render as a header row either
// (web/src/lib/participation.ts's identical client-side mirror gates the
// chip on the same participation rule).
const configSpec: FixtureConfigSpec = driveBin.missing
	? { sources: [], webspaces: [] }
	: {
			sources: [
				{
					id: SOURCE_ID,
					plugin: DRIVE_BINARY,
					displayName: DISPLAY_NAME,
					extras: {
						client_id: '${GDRIVE_CLIENT_ID}',
						client_secret: '${GDRIVE_CLIENT_SECRET}',
						folder_id: '1a2B3cD4EfGhIjKlmNoPQRstuVwxYZ'
					}
				}
			],
			webspaces: webspacesWithKeywords([WEBSPACE], ['gdrive-e2e-rehearsal']),
			// The load-bearing part (see file header): the Drive binary is named
			// ONLY here, never in pluginBinaries below.
			pluginBinaries: [],
			externalPluginBinaries: [DRIVE_BINARY],
			externalPluginBinariesSrcDir: DRIVE_SRC_DIR
		};

test.use({ configSpec });

test.describe('14-04 Task 1: Google Drive plugin, external tier, no Google account', () => {
	test.skip(driveBin.missing, SKIP_REASON);

	test('GET /api/sources reports the instance with the external tier', async ({ kernel }) => {
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
	});

	test('the instance is unreachable and its last error is the exact not-authorized sentence — never a healthy but empty stream', async ({
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [SOURCE_ID], { logs: kernel.logs });

		const res = await fetch(`${kernel.baseURL}/api/sources`);
		expect(res.ok, `sources request failed: ${res.status}`).toBe(true);
		const body = (await res.json()) as {
			sources: Array<{ name: string; reachable: boolean; last_error: string }>;
		};

		const instance = body.sources.find((s) => s.name === SOURCE_ID);
		expect(instance, `expected a "${SOURCE_ID}" entry in GET /api/sources`).toBeTruthy();
		expect(instance?.reachable, 'expected the credential-less Drive source to be unreachable').toBe(false);
		// .toContain, not .toBe (matching 13-manifest-unverified.spec.ts's own
		// "contract-exact" convention): kernel/correlate.SyncSource wraps every
		// Match error in generic, kernel-composed context ('match against
		// source "…": rpc error: code = Unavailable desc = …') before it lands
		// in sync_runs.error — that wrapping is uniform host behavior for any
		// plugin's Match failure, never something a plugin controls. The exact
		// PRD/UI-SPEC sentence must still appear byte-for-byte inside it,
		// which is what this assertion pins — never a paraphrase, never a
		// loosely-matching regex fragment.
		expect(instance?.last_error).toContain(NOT_AUTHORIZED_SENTENCE);
	});

	test('POST /api/config/describe-plugin reports all three declared extras and the folders match vocabulary, with no credential required', async ({
		kernel
	}) => {
		// Deliberately an EMPTY source (no extras filled in at all) — this is
		// the Describe path proving itself: the response is rendered by
		// generic host code with nothing in this repository naming this
		// plugin (14-04-PLAN.md Task 1 acceptance criterion), and it must
		// answer without credentials because the host trial-launches Describe
		// before an operator has finished typing (14-PLUGIN-PRD.md's Design
		// Guidance: "Describe must be side-effect-free and must never
		// validate credentials or call Drive").
		const res = await fetch(`${kernel.baseURL}/api/config/describe-plugin`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ plugin: DRIVE_BINARY, source: { plugin: DRIVE_BINARY } })
		});
		expect(res.ok, `describe-plugin failed: ${res.status}`).toBe(true);
		const body = (await res.json()) as {
			match_vocabulary: string[];
			extras: Array<{ key: string; label: string; required: boolean; secret: boolean; placeholder?: string }>;
		};

		expect(body.match_vocabulary).toContain('folders');

		const byKey = new Map(body.extras.map((f) => [f.key, f]));
		const clientId = byKey.get('client_id');
		const clientSecret = byKey.get('client_secret');
		const folderId = byKey.get('folder_id');

		expect(clientId, 'expected a client_id extras field').toBeTruthy();
		expect(clientSecret, 'expected a client_secret extras field').toBeTruthy();
		expect(folderId, 'expected a folder_id extras field').toBeTruthy();

		expect(clientId?.secret, 'client_id must be marked secret').toBe(true);
		expect(clientSecret?.secret, 'client_secret must be marked secret').toBe(true);
		expect(folderId?.secret, 'folder_id must NOT be marked secret').toBe(false);
	});

	test('the chip on the webspace page carries exactly one trust badge, and it is the only visible badge on the page', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [SOURCE_ID], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const chip = page.getByRole('button', { name: DISPLAY_NAME, exact: true });
		await expect(chip).toBeVisible();

		// The trust badge glyph renders as CircleAlert (@lucide/svelte classes
		// it `lucide-circle-alert`), mirroring 11-external-tier-badge.spec.ts's
		// and 12-external-rehearsal.spec.ts's own selector convention.
		const badgeGlyph = 'svg.lucide-circle-alert';
		await expect(
			chip.locator(badgeGlyph),
			'expected the external-tier chip to carry exactly one trust badge'
		).toHaveCount(1);

		// Header-wide total, scoped with Playwright's `:visible` pseudo-class
		// (WebspaceHeader.svelte renders an invisible off-screen measurement
		// clone of every chip for overflow-width calculation) — fails loudly
		// if any OTHER chip also carries a badge.
		await expect(page.locator(`${badgeGlyph}:visible`)).toHaveCount(1);
	});

	test('the chip exposes the untrusted-external-plugin clause as its accessible description', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [SOURCE_ID], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const chip = page.getByRole('button', { name: DISPLAY_NAME, exact: true });

		// 14-02-PLAN.md Task 2 (14-UI-SPEC.md G1, option-b): the disclosure no
		// longer renders through a native `title` attribute — it is exposed as
		// the button's accessible DESCRIPTION via a visually-hidden sr-only
		// span wired through aria-describedby. Repointed onto the same surface
		// 12-external-rehearsal.spec.ts's own assertion was repointed to
		// (14-02-SUMMARY.md), not a third convention.
		await expect(chip).toHaveAccessibleDescription(/untrusted external plugin/);
	});
});
