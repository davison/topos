// 12-10-PLAN.md Task 2: the user's exact reported failure, in a browser,
// against a real kernel and a real topos-plugin-filesystem subprocess
// (G-12-1/G-12-3). This spec reproduces the literal match value from the
// user's real config, as captured in
// .planning/debug/filesystem-items-missing-from-stream.md
// ([webspaces.test.match.files] folders = ['*']): a webspace whose only
// match input for a filesystem instance is an explicit block naming a
// single asterisk in `folders`.
//
// Its subject is the SILENT-FAILURE SHAPE — a healthy, green-reporting
// source contributing zero items to its webspace — rather than the match
// semantics themselves, which 12-filesystem-root-label-match.spec.ts
// already covers (that spec predates this wave's last_notice field and
// deliberately does not assert on it, per its own header comment). This
// spec is the completion of that story: 12-09-PLAN.md made the zero-match
// fact observable at the API layer (last_notice), and this plan (Task 1)
// made it visible on the chip; this spec proves the whole path end to end
// through a real browser session.
//
// A temp corpus dir holds one PDF at its top level — genuinely matchable
// content, so an empty result can only be a matching failure, never an
// empty folder (mirrors 12-filesystem-tracer.spec.ts's own corpus-seeding
// idiom).
//
// What stays outside this harness by design (docs/testing.md): a real
// xdg-open handoff, real previews on the user's own documents, and
// NFS/SMB mount behaviour — all human-verified in UAT, never asserted
// here.
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { basename, join } from 'node:path';

import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import type { FixtureConfigSpec } from '../fixtures/config-builder';

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

const PDF_FILENAME = 'invoice.pdf';
const INSTANCE_ID = 'files';
const DISPLAY_NAME = 'Household Docs';
const WEBSPACE = 'zero-match-diagnostic';
// GLOB_SHAPED_VALUE is byte-identical to the value the user's own real
// config carried (config.toml.bak -> config.toml diff: '**' -> '*'), per
// the debug session's 2026-08-14T09:08 evidence entry.
const GLOB_SHAPED_VALUE = '*';

// Module-scope temp corpus directory (D-03: state is seeded before kernel
// boot).
const corpusDir = mkdtempSync(join(tmpdir(), 'topos-e2e-zero-match-'));
writeFileSync(join(corpusDir, PDF_FILENAME), MINIMAL_PDF);

const configSpec: FixtureConfigSpec = {
	sources: [
		{
			id: INSTANCE_ID,
			plugin: 'topos-plugin-filesystem',
			path: corpusDir,
			displayName: DISPLAY_NAME
		}
	],
	// The exact shape of the user's real config: an explicit sources
	// allowlist plus a per-instance match block naming the asterisk value —
	// no keywords fallback, which deliberately produces no advisory.
	webspaces: [
		{
			name: WEBSPACE,
			sources: [INSTANCE_ID],
			match: { [INSTANCE_ID]: { folders: [GLOB_SHAPED_VALUE] } }
		}
	],
	pluginBinaries: ['topos-plugin-filesystem']
};

test.use({ configSpec });

test.afterAll(() => {
	rmSync(corpusDir, { recursive: true, force: true });
});

interface SourceStatus {
	name: string;
	reachable: boolean;
	last_status: string;
	last_error: string;
	last_notice?: string;
}

test.describe('12-10 Task 2: the user\'s exact config — healthy, green-reporting, contributing nothing, now visible', () => {
	test('GET /api/sources carries a non-empty last_notice alongside last_status ok, and the stream stays empty for this instance', async ({
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [INSTANCE_ID], { logs: kernel.logs });

		const sourcesRes = await fetch(`${kernel.baseURL}/api/sources`);
		expect(sourcesRes.ok, `sources request failed: ${sourcesRes.status}`).toBe(true);
		const sourcesBody = (await sourcesRes.json()) as { sources: SourceStatus[] };
		const instance = sourcesBody.sources.find((s) => s.name === INSTANCE_ID);
		expect(instance, `expected ${INSTANCE_ID} in GET /api/sources, got: ${JSON.stringify(sourcesBody.sources)}`).toBeTruthy();

		// The healthy-while-empty coexistence — the exact shape that made
		// this failure invisible before this plan.
		expect(instance?.reachable, `expected ${INSTANCE_ID} to be reachable`).toBe(true);
		expect(instance?.last_status, `expected ${INSTANCE_ID}'s last_status to be "ok"`).toBe('ok');
		expect(instance?.last_error, `expected ${INSTANCE_ID}'s last_error to be empty`).toBe('');

		expect(
			instance?.last_notice,
			'expected a non-empty last_notice naming the webspace and the asterisk value'
		).toBeTruthy();
		expect(instance?.last_notice).toContain(WEBSPACE);
		expect(instance?.last_notice).toContain(GLOB_SHAPED_VALUE);

		const streamRes = await fetch(`${kernel.baseURL}/api/webspaces/${WEBSPACE}/stream`);
		expect(streamRes.ok, `stream request failed: ${streamRes.status}`).toBe(true);
		const stream = (await streamRes.json()) as { items: Array<{ source: string }> };
		const instanceItems = stream.items.filter((it) => it.source === INSTANCE_ID);
		expect(
			instanceItems,
			`expected zero items from ${INSTANCE_ID}, got: ${JSON.stringify(instanceItems)}`
		).toHaveLength(0);
	});

	test('the chip renders the warning tone and a title carrying the API-published advisory, with no "a source couldn\'t sync" degraded treatment', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [INSTANCE_ID], { logs: kernel.logs });

		const sourcesRes = await fetch(`${kernel.baseURL}/api/sources`);
		const sourcesBody = (await sourcesRes.json()) as { sources: SourceStatus[] };
		const instance = sourcesBody.sources.find((s) => s.name === INSTANCE_ID);
		const notice = instance?.last_notice ?? '';
		expect(notice, 'expected a non-empty last_notice to assert the DOM title against').not.toBe('');
		// A distinctive substring, not the whole sentence — proves the two
		// surfaces agree rather than hard-coding kernel copy a later pass
		// may legitimately improve.
		const distinctiveSubstring = `webspace "${WEBSPACE}"`;
		expect(notice).toContain(distinctiveSubstring);

		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const chip = page.getByRole('button', { name: DISPLAY_NAME, exact: true });
		await expect(chip).toBeVisible();

		// Both directions: the warning tone is present AND the success tone
		// is absent — proving a tone, not merely a class-name sighting.
		await expect(chip.locator('span.size-2')).toHaveClass(/bg-warning/);
		await expect(chip.locator('span.size-2')).not.toHaveClass(/bg-success/);

		await expect(chip).toHaveAttribute('title', new RegExp(distinctiveSubstring.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')));

		// The sync did not fail — claiming otherwise would send the user to
		// restart a healthy service, the exact defect 08-UAT.md's G-08-3
		// recorded. This is the reason 12-08-PLAN.md/12-09-PLAN.md's fix
		// lives on the chip, never on the stream.
		await expect(page.getByText("A source couldn't sync", { exact: true })).toHaveCount(0);
	});
});
