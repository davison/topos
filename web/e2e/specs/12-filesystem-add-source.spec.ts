// 12-04-PLAN.md Task 3: success criterion 1, driven entirely from the real
// UI against a real booted kernel and a real topos-plugin-filesystem
// binary. No filesystem source exists in the starting config — the
// picker's Group 2 catalog, the Connect step's Local Path/Include
// subfolders fields (12-UI-SPEC.md F1), the missing-required-field guard,
// and the Match step are all exercised exactly as an operator would use
// them, ending with the checkbox's value having actually reached the
// launched plugin subprocess: only a document one level BELOW the
// configured root reaches the stream, and only because "Include
// subfolders" was ticked before saving.
import { mkdtempSync, mkdirSync, rmSync, writeFileSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';

import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import type { FixtureConfigSpec } from '../fixtures/config-builder';

// A minimal but structurally valid single-page PDF — same fixture body
// 12-01/12-03's specs already use.
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
const NESTED_SUBFOLDER = 'archive';
const NESTED_FILENAME = 'old-invoice.pdf';
const NESTED_SOURCE_ID = `${NESTED_SUBFOLDER}/${NESTED_FILENAME}`;

const WEBSPACE = 'household-archive';
const DISPLAY_NAME = 'Household Docs';
// deriveInstanceId's slug for DISPLAY_NAME (web/src/lib/instance-id.ts):
// trim, lowercase, non-alphanumeric runs collapsed to a single "-".
const INSTANCE_ID = 'household-docs';

// Module-scope temp corpus directory (D-03: state is seeded before kernel
// boot) — a top-level document plus one nested one level down, under a
// subfolder whose own name is exactly the D-05 folder-vocabulary label the
// filesystem plugin emits for it. The webspace's own keywords name that
// subfolder alone, so the TOP-LEVEL document never participates — only the
// nested one does, and only through the recursive walk the UI's checkbox
// turns on during this spec.
const corpusDir = mkdtempSync(join(tmpdir(), 'topos-e2e-fs-add-source-'));
writeFileSync(join(corpusDir, TOP_LEVEL_FILENAME), MINIMAL_PDF);
mkdirSync(join(corpusDir, NESTED_SUBFOLDER));
writeFileSync(join(corpusDir, NESTED_SUBFOLDER, NESTED_FILENAME), MINIMAL_PDF);

// No filesystem source in the starting config — criterion 1 requires the
// source to be created THROUGH the UI, not seeded. A single pre-existing
// topos-plugin-mock instance is seeded instead, for a reason that has
// nothing to do with the filesystem plugin itself: WebspaceHeader.svelte's
// chip row (and the "+" Add-source trigger living inside it) only renders
// once at least one source is configured SYSTEM-WIDE
// (shouldShowSourceRows, web/src/lib/format.ts) — a genuinely zero-source
// install has no chip row to click "+" from at all. mock-01 is never
// attached to this webspace's own match block and its fixed corpus items
// carry no "archive"-labelled item (plugins/mock/plugin_test.go's fixture
// data), so it contributes nothing to WEBSPACE's stream either way.
const configSpec: FixtureConfigSpec = {
	sources: [{ id: 'mock-01', plugin: 'topos-plugin-mock', displayName: 'Mock One' }],
	webspaces: [{ name: WEBSPACE, keywords: [NESTED_SUBFOLDER] }],
	pluginBinaries: ['topos-plugin-mock', 'topos-plugin-filesystem']
};

test.use({ configSpec });

test.afterAll(() => {
	rmSync(corpusDir, { recursive: true, force: true });
});

test.describe('12-04 Task 3: adding a folder as a source through the real UI, end to end', () => {
	test('an empty path is blocked, the checkbox starts unchecked, and saving with subfolders included brings the nested document into the stream', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		await page.getByRole('button', { name: 'Add source' }).click();
		await page.getByRole('button', { name: 'Filesystem folder', exact: true }).click();

		const dialog = page.getByRole('dialog');
		await expect(dialog.getByRole('heading', { name: 'Connect Filesystem folder' })).toBeVisible();

		// --- populated/empty (F1): Local Path renders empty with its
		// placeholder, the checkbox renders unchecked.
		const pathInput = dialog.locator('#conn-path');
		await expect(pathInput).toHaveValue('');
		await expect(pathInput).toHaveAttribute(
			'placeholder',
			'e.g. /home/you/Documents or /mnt/nas/shared-docs'
		);

		const recursiveCheckbox = dialog.getByRole('checkbox', { name: 'Include subfolders' });
		await expect(recursiveCheckbox).not.toBeChecked();

		// --- error (F1): a whitespace-only path passes the native `required`
		// attribute but fails ConnectionForm's own missingRequiredFields
		// check — the same technique 07.1's uat-05 spec uses to reach the
		// JS-level guard rather than the browser's own (non-assertable)
		// validation UI.
		await pathInput.fill(' ');
		await dialog.getByRole('button', { name: 'Next' }).click();
		await expect(dialog.getByText('Fill in Local Path before continuing.')).toBeVisible();

		// --- fill in the real values and tick "Include subfolders" —
		// partial (F1): the checkbox and the path field validate
		// independently, no cross-field consequence either way.
		await dialog.locator('#conn-display_name').fill(DISPLAY_NAME);
		await pathInput.fill(corpusDir);
		await recursiveCheckbox.click();
		await expect(recursiveCheckbox).toBeChecked();

		await dialog.getByRole('button', { name: 'Next' }).click();
		await expect(
			dialog.getByRole('heading', { name: `Match settings for ${WEBSPACE}` })
		).toBeVisible();

		// Leave the Match step's fields blank — the webspace's own keywords
		// fallback (naming the nested subfolder) is what brings the nested
		// document into the stream, so a populated stream here proves the
		// checkbox's value actually reached the plugin, not merely the form.
		await dialog.getByRole('button', { name: 'Add source' }).click();

		await expect(page.getByRole('dialog')).toHaveCount(0);

		await waitForFirstSync(kernel.baseURL, [INSTANCE_ID], { logs: kernel.logs });

		const streamRes = await fetch(`${kernel.baseURL}/api/webspaces/${WEBSPACE}/stream`);
		expect(streamRes.ok, `stream request failed: ${streamRes.status}`).toBe(true);
		const stream = (await streamRes.json()) as {
			items: Array<{ source: string; source_id: string }>;
		};

		const nestedItem = stream.items.find(
			(it) => it.source === INSTANCE_ID && it.source_id === NESTED_SOURCE_ID
		);
		expect(
			nestedItem,
			`expected the nested document to reach the stream via the checkbox-enabled recursive walk, got: ${JSON.stringify(stream.items)}`
		).toBeTruthy();

		const topLevelItem = stream.items.find(
			(it) => it.source === INSTANCE_ID && it.source_id === TOP_LEVEL_FILENAME
		);
		expect(
			topLevelItem,
			'the top-level document does not carry the "archive" folder label, so it must not appear in this webspace'
		).toBeUndefined();
	});
});
