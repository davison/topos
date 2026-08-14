// 12-07-PLAN.md Task 1: closes the gap 12-VERIFICATION.md recorded — a file
// admitted to a filesystem source's index only because include_glob widened
// scope past the default extension allowlist must be fetched as an honest
// metadata-only preview, never a false "not found," on BOTH the kernel's
// detail route (GET /api/items/{id}, CONTENT_VARIANT_FULL) and its content
// route (GET /api/items/{id}/content, CONTENT_VARIANT_PREVIEW). Reproduces
// the verifier's live repro: include_glob="**/*.zip" plus an archive.zip.
//
// This is a plugin/kernel contract defect, not a DOM rendering regression —
// asserted at the kernel API level only. No browser-DOM assertion is added
// here; docs/testing.md already scopes real xdg-open desktop-handler
// behaviour out of this hermetic harness, and the "open in source" affordance
// for an unavailable rendition is unaffected by this gap.
import { mkdtempSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { basename, join } from 'node:path';

import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import type { FixtureConfigSpec } from '../fixtures/config-builder';

const ZIP_FILENAME = 'archive.zip';
const SOURCE_ID = 'docs-folder';
const WEBSPACE = 'household';

// Module-scope temp corpus dir (state seeded before kernel boot) containing
// exactly one file whose extension (.zip) is outside the default extension
// allowlist — the plugin never parses its bytes, so arbitrary content is
// fine. Its own base name is the D-05 folder-vocabulary label a top-level
// file carries, so the webspace's keywords name it directly.
const corpusDir = mkdtempSync(join(tmpdir(), 'topos-e2e-fs-includeglob-'));
writeFileSync(join(corpusDir, ZIP_FILENAME), 'arbitrary bytes, never parsed');

const configSpec: FixtureConfigSpec = {
	sources: [
		{
			id: SOURCE_ID,
			plugin: 'topos-plugin-filesystem',
			path: corpusDir,
			extras: { include_glob: '**/*.zip' }
		}
	],
	webspaces: [{ name: WEBSPACE, keywords: [basename(corpusDir)] }],
	pluginBinaries: ['topos-plugin-filesystem']
};

test.use({ configSpec });

test.afterAll(() => {
	rmSync(corpusDir, { recursive: true, force: true });
});

test.describe('12-07 Task 1: include_glob-admitted unknown extension previews honestly, never 404s', () => {
	test('the item appears in the stream and answers an honest unavailable preview, not item_not_found', async ({
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [SOURCE_ID], { logs: kernel.logs });

		const streamRes = await fetch(`${kernel.baseURL}/api/webspaces/${WEBSPACE}/stream`);
		expect(streamRes.ok, `stream request failed: ${streamRes.status}`).toBe(true);
		const stream = (await streamRes.json()) as {
			items: Array<{ id: string; source_id: string; provenance?: Record<string, string> }>;
		};

		expect(stream.items, `expected exactly one item, got: ${JSON.stringify(stream.items)}`).toHaveLength(
			1
		);
		const streamItem = stream.items[0];
		expect(streamItem.source_id).toBe(ZIP_FILENAME);

		// GET /api/items/{id} (CONTENT_VARIANT_FULL) — under the pre-fix code
		// this was a false 404 item_not_found for a file the stream itself
		// just listed.
		const itemRes = await fetch(`${kernel.baseURL}/api/items/${encodeURIComponent(streamItem.id)}`);
		expect(itemRes.ok, `expected 200, got ${itemRes.status}`).toBe(true);
		const detail = (await itemRes.json()) as {
			content: { available: boolean; unavailable_reason?: string };
		};
		expect(detail.content.available).toBe(false);
		expect(detail.content.unavailable_reason).toBeTruthy();

		// GET /api/items/{id}/content (CONTENT_VARIANT_PREVIEW) — the honest
		// "no rendition" answer is content_unavailable, specifically NOT
		// item_not_found, which is what the false-404 produced.
		const contentRes = await fetch(
			`${kernel.baseURL}/api/items/${encodeURIComponent(streamItem.id)}/content`
		);
		expect(contentRes.status).toBe(404);
		const contentBody = (await contentRes.json()) as { error: { code: string } };
		expect(contentBody.error.code).toBe('content_unavailable');
		expect(contentBody.error.code).not.toBe('item_not_found');
	});
});
