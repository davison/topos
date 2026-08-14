// 12-08-PLAN.md Task 3: reproduces the user's exact reported failure
// alongside its fix, against a real kernel and a real
// topos-plugin-filesystem subprocess (G-12-1/G-12-3).
//
// The literal config that produced the failure
// (.planning/debug/filesystem-items-missing-from-stream.md) was
// `[webspaces.test.match.files] folders = ['*']` — the user's own real
// desktop config, iterated from `['**']` — over a recursive filesystem
// source. That value is the NEGATIVE case below: a glob-shaped match
// value is compared as a literal string (D-04) against the labels
// item.go's folderLabels emits, matches none of them, and the source's
// stream is silently empty despite a fully healthy sync. The POSITIVE
// case is the fix Task 1 shipped: naming the configured folder's own base
// name in `folders` now matches every file that instance contributes, at
// every depth (top-level AND nested) — the correct value the user had no
// way to type before this plan.
//
// The assertions are deliberately API-level (fetch against
// kernel.baseURL, following 12-filesystem-tracer.spec.ts's idiom) rather
// than DOM-driven: the defect this spec pins is in match-value semantics,
// not in anything rendered. The `last_status`/`reachable` assertion below
// is the silent-failure shape itself — a healthy source with an empty
// stream genuinely coexist, which is exactly what made this failure hard
// to diagnose in the first place. This spec does NOT assert on any
// advisory or notice field (e.g. a zero-match diagnostic) — that surface
// does not exist in this wave; it is 12-09-PLAN.md's own gap closure and
// would make this spec fail for a reason it is not about.
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
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

const TOP_LEVEL_FILENAME = 'invoice.pdf';
const NESTED_SUBFOLDER = 'receipts';
const NESTED_FILENAME = 'nested-invoice.pdf';
const NESTED_SOURCE_ID = `${NESTED_SUBFOLDER}/${NESTED_FILENAME}`;

const INSTANCE_ID = 'household-docs';
// GLOB_SHAPED_VALUE is byte-identical to the value the user's own real
// config carried (config.toml.bak -> config.toml diff: '**' -> '*').
const GLOB_SHAPED_VALUE = '*';
const MATCH_WEBSPACE = 'match-value-webspace';
const GLOB_WEBSPACE = 'glob-value-webspace';

// Module-scope temp corpus directory (D-03: state is seeded before kernel
// boot) — a top-level document plus one nested one level down, exactly
// the shape the diagnosis's "and_gate" note describes as structurally
// inexpressible before this plan.
const corpusDir = mkdtempSync(join(tmpdir(), 'topos-e2e-fs-root-label-'));
writeFileSync(join(corpusDir, TOP_LEVEL_FILENAME), MINIMAL_PDF);
mkdirSync(join(corpusDir, NESTED_SUBFOLDER));
writeFileSync(join(corpusDir, NESTED_SUBFOLDER, NESTED_FILENAME), MINIMAL_PDF);

// The match value is derived from the corpus dir's own base name at
// runtime — never hard-coded — mirroring folderLabels' own contract
// (item.go, as modified by 12-08-PLAN.md Task 1).
const rootBaseNameMatchValue = basename(corpusDir);

const configSpec: FixtureConfigSpec = {
	sources: [
		{
			id: INSTANCE_ID,
			plugin: 'topos-plugin-filesystem',
			path: corpusDir,
			recursive: true,
			displayName: 'Household Docs'
		}
	],
	// Two webspaces, each with an explicit match block for the ONE
	// instance and no keywords — exactly the config shape the user's
	// failure had ([webspaces.test.match.files], D-02).
	webspaces: [
		{ name: MATCH_WEBSPACE, sources: [INSTANCE_ID], match: { [INSTANCE_ID]: { folders: [rootBaseNameMatchValue] } } },
		{ name: GLOB_WEBSPACE, sources: [INSTANCE_ID], match: { [INSTANCE_ID]: { folders: [GLOB_SHAPED_VALUE] } } }
	],
	pluginBinaries: ['topos-plugin-filesystem']
};

test.use({ configSpec });

test.afterAll(() => {
	rmSync(corpusDir, { recursive: true, force: true });
});

interface StreamItem {
	id: string;
	source: string;
	source_id: string;
}

async function fetchStreamItems(baseURL: string, webspace: string): Promise<StreamItem[]> {
	const res = await fetch(`${baseURL}/api/webspaces/${webspace}/stream`);
	expect(res.ok, `stream request for ${webspace} failed: ${res.status}`).toBe(true);
	const body = (await res.json()) as { items: StreamItem[] };
	return body.items;
}

test.describe('12-08 Task 3: the user\'s reported failure and its fix, both pinned end to end', () => {
	test('the root base name matches every file at every depth; the glob-shaped value matches nothing', async ({
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [INSTANCE_ID], { logs: kernel.logs });

		// POSITIVE case (the fix): naming the configured folder's own base
		// name matches BOTH the top-level and the nested item — one value
		// covering the whole instance at every depth, which is the
		// behaviour that did not exist before this plan.
		const matchItems = await fetchStreamItems(kernel.baseURL, MATCH_WEBSPACE);
		const topLevelItem = matchItems.find(
			(it) => it.source === INSTANCE_ID && it.source_id === TOP_LEVEL_FILENAME
		);
		expect(
			topLevelItem,
			`expected the root-base-name match value to include the top-level file, got: ${JSON.stringify(matchItems)}`
		).toBeTruthy();
		const nestedItem = matchItems.find(
			(it) => it.source === INSTANCE_ID && it.source_id === NESTED_SOURCE_ID
		);
		expect(
			nestedItem,
			`expected the root-base-name match value to include the nested file, got: ${JSON.stringify(matchItems)}`
		).toBeTruthy();

		// NEGATIVE case (the user's exact reported failure): a glob-shaped
		// value is a literal, matches no emitted label, and yields zero
		// items from this instance.
		const globItems = await fetchStreamItems(kernel.baseURL, GLOB_WEBSPACE);
		const globInstanceItems = globItems.filter((it) => it.source === INSTANCE_ID);
		expect(
			globInstanceItems,
			`expected the glob-shaped match value to match nothing from this instance, got: ${JSON.stringify(globInstanceItems)}`
		).toHaveLength(0);

		// The healthy-sync-with-empty-stream coexistence is asserted, not
		// assumed — this is the silent-failure shape that made the original
		// mistake hard to diagnose (the diagnostic side is closed
		// separately, in 12-09-PLAN.md, this same wave).
		const sourcesRes = await fetch(`${kernel.baseURL}/api/sources`);
		expect(sourcesRes.ok, `sources request failed: ${sourcesRes.status}`).toBe(true);
		const sourcesBody = (await sourcesRes.json()) as {
			sources: Array<{ name: string; reachable: boolean; last_status: string }>;
		};
		const instanceStatus = sourcesBody.sources.find((s) => s.name === INSTANCE_ID);
		expect(
			instanceStatus,
			`expected ${INSTANCE_ID} in GET /api/sources, got: ${JSON.stringify(sourcesBody.sources)}`
		).toBeTruthy();
		expect(instanceStatus?.reachable, `expected ${INSTANCE_ID} to be reachable`).toBe(true);
		expect(instanceStatus?.last_status, `expected ${INSTANCE_ID}'s last_status to be "ok"`).toBe('ok');
	});
});
