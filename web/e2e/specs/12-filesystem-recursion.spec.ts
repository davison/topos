// 12-03-PLAN.md Task 2: criterion 2's add/change/remove claim, driven end
// to end against a real kernel and a real topos-plugin-filesystem binary.
// Two source instances point at the SAME corpus directory — one
// `recursive = true`, one not (config-builder.ts's Task 1 addition) — and
// a webspace matches the nested subfolder's own name via the ordinary
// keywords fallback (D-05's folder-vocabulary labels, extended by Task 2
// to include a nested file's containing-directory segment names). The
// recursive instance contributes the nested item; the non-recursive one
// never even walks past the root's own top level, so it cannot. Deleting
// the nested file and triggering a re-sync through the existing refresh
// route proves the item leaves the stream — no OS filesystem-watcher
// dependency anywhere in this design, and the kernel's own full-replace
// persistence is what makes the removal observable.
import { mkdtempSync, mkdirSync, rmSync, unlinkSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

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

const RECURSIVE_ID = 'docs-recursive';
const FLAT_ID = 'docs-flat';
const WEBSPACE = 'household-receipts';

// Module-scope temp corpus directory (D-03: state is seeded before kernel
// boot) — a top-level document plus one nested one level down, under a
// subfolder whose own name (`receipts`) is exactly the D-05 folder label
// Task 2's folderLabels extension emits for it.
const corpusDir = mkdtempSync(join(tmpdir(), 'topos-e2e-fs-recursion-'));
writeFileSync(join(corpusDir, TOP_LEVEL_FILENAME), MINIMAL_PDF);
mkdirSync(join(corpusDir, NESTED_SUBFOLDER));
const nestedFilePath = join(corpusDir, NESTED_SUBFOLDER, NESTED_FILENAME);
writeFileSync(nestedFilePath, MINIMAL_PDF);

const configSpec: FixtureConfigSpec = {
	sources: [
		{
			id: RECURSIVE_ID,
			plugin: 'topos-plugin-filesystem',
			path: corpusDir,
			recursive: true,
			displayName: 'Household Docs (recursive)'
		},
		{
			id: FLAT_ID,
			plugin: 'topos-plugin-filesystem',
			path: corpusDir,
			recursive: false,
			displayName: 'Household Docs (flat)'
		}
	],
	// The keywords fallback fans across every declared vocabulary field
	// (here, just "folders") — matching the nested subfolder's own name
	// keeps this webspace narrow to exactly the item this spec cares
	// about, on both instances.
	webspaces: [{ name: WEBSPACE, keywords: [NESTED_SUBFOLDER] }],
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

async function fetchStreamItems(baseURL: string): Promise<StreamItem[]> {
	const res = await fetch(`${baseURL}/api/webspaces/${WEBSPACE}/stream`);
	expect(res.ok, `stream request failed: ${res.status}`).toBe(true);
	const body = (await res.json()) as { items: StreamItem[] };
	return body.items;
}

test.describe('12-03 Task 2: recursion toggle — add and remove reflected end to end', () => {
	test('the recursive instance contributes the nested item; the non-recursive instance does not', async ({
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [RECURSIVE_ID, FLAT_ID], { logs: kernel.logs });

		const items = await fetchStreamItems(kernel.baseURL);

		const recursiveItem = items.find((it) => it.source === RECURSIVE_ID && it.source_id === NESTED_SOURCE_ID);
		expect(recursiveItem, `expected the recursive instance to contribute ${NESTED_SOURCE_ID}, got: ${JSON.stringify(items)}`).toBeTruthy();

		const flatNestedItem = items.find((it) => it.source === FLAT_ID && it.source_id === NESTED_SOURCE_ID);
		expect(
			flatNestedItem,
			`expected the non-recursive instance to NEVER contribute a nested item, got: ${JSON.stringify(items)}`
		).toBeUndefined();
	});

	test('deleting the nested file and re-syncing removes it from the stream — no watcher, just the next full-replace', async ({
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [RECURSIVE_ID, FLAT_ID], { logs: kernel.logs });

		const before = await fetchStreamItems(kernel.baseURL);
		const beforeItem = before.find((it) => it.source === RECURSIVE_ID && it.source_id === NESTED_SOURCE_ID);
		expect(beforeItem, `expected the nested item to be present before deletion, got: ${JSON.stringify(before)}`).toBeTruthy();

		unlinkSync(nestedFilePath);

		const refreshRes = await fetch(`${kernel.baseURL}/api/sources/${RECURSIVE_ID}/refresh`, { method: 'POST' });
		expect(refreshRes.ok, `refresh request failed: ${refreshRes.status}`).toBe(true);

		const after = await fetchStreamItems(kernel.baseURL);
		const afterItem = after.find((it) => it.source === RECURSIVE_ID && it.source_id === NESTED_SOURCE_ID);
		expect(
			afterItem,
			`expected the deleted item to be absent after re-sync, got: ${JSON.stringify(after)}`
		).toBeUndefined();
	});
});
