// 12-01-PLAN.md Task 2: the whole filesystem-source path end to end on one
// thin slice — a single PDF sitting in a configured folder becomes a
// stream item, and its "Open in …" action reaches the kernel-mediated open
// route (POST /api/items/{id}/open, D-06) that resolves the real file path
// from the index alone. Proves the file://-scheme deep-link convention
// plus its kernel-side rewrite (kernel/httpapi/stream.go's
// resolveStreamLinkURL) and the OpenInSource local-exec branch
// (12-UI-SPEC.md F2) together, against a real kernel and a real
// topos-plugin-filesystem binary. Does NOT assert on xdg-open's own
// behaviour — the exec is covered deterministically by the stub-opener Go
// tests (kernel/httpapi/fsopen_test.go).
import { writeFileSync } from 'node:fs';
import { basename, join } from 'node:path';

import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { mkdtempCorpus } from '../fixtures/corpus';
import type { FixtureConfigSpec } from '../fixtures/config-builder';

// A minimal but structurally valid single-page PDF — a real %PDF- header,
// one Catalog/Pages/Page object chain, and a %%EOF trailer — proving the
// item that reaches the browser is a genuine PDF rendition, not an
// arbitrary byte blob wearing an application/pdf label.
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
const SOURCE_ID = 'docs-folder';
const DISPLAY_NAME = 'Household Docs';
const WEBSPACE = 'household';

// Module-scope temp corpus directory (D-03: state is seeded before kernel
// boot) — its own base name is exactly the D-05 folder-vocabulary label
// the filesystem plugin emits for a top-level file, so a webspace whose
// keywords name it participates via the ordinary keywords fallback with no
// explicit match block.
const corpusDir = mkdtempCorpus('topos-e2e-fs-');
writeFileSync(join(corpusDir, PDF_FILENAME), MINIMAL_PDF);

const configSpec: FixtureConfigSpec = {
	sources: [
		{
			id: SOURCE_ID,
			plugin: 'topos-plugin-filesystem',
			path: corpusDir,
			displayName: DISPLAY_NAME
		}
	],
	webspaces: [{ name: WEBSPACE, keywords: [basename(corpusDir)] }],
	pluginBinaries: ['topos-plugin-filesystem']
};

test.use({ configSpec });

test.describe('12-01 Task 2: filesystem source tracer — one PDF, end to end', () => {
	test('the stream carries exactly one item, served through the kernel open route, with a PDF rendition', async ({
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [SOURCE_ID], { logs: kernel.logs });

		const streamRes = await fetch(`${kernel.baseURL}/api/webspaces/${WEBSPACE}/stream`);
		expect(streamRes.ok, `stream request failed: ${streamRes.status}`).toBe(true);
		const stream = (await streamRes.json()) as {
			items: Array<{ id: string; source_id: string; link: { url: string } }>;
		};

		expect(stream.items, `expected exactly one item, got: ${JSON.stringify(stream.items)}`).toHaveLength(
			1
		);
		const streamItem = stream.items[0];
		expect(streamItem.source_id).toBe(PDF_FILENAME);

		// The plugin -> kernel rewrite end to end: link.url is the loopback
		// open route for this item's own id, never the plugin's raw file://
		// deep_link value.
		expect(streamItem.link.url).toBe(`/api/items/${streamItem.id}/open`);
		expect(streamItem.link.url.startsWith('file://')).toBe(false);

		const itemRes = await fetch(`${kernel.baseURL}/api/items/${encodeURIComponent(streamItem.id)}`);
		expect(itemRes.ok, `item detail request failed: ${itemRes.status}`).toBe(true);
		const detail = (await itemRes.json()) as {
			content: { rendition: { mime_type: string } | null };
		};
		expect(detail.content.rendition, 'expected a rendition on the item detail response').not.toBeNull();
		expect(detail.content.rendition?.mime_type).toBe('application/pdf');
	});

	test('selecting the row renders an Open control that is a button, not an anchor', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [SOURCE_ID], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const row = page.locator('[data-item-id]').first();
		await expect(row).toBeVisible();
		await row.click();

		const openControl = page.getByRole('button', { name: `Open in ${DISPLAY_NAME}` });
		await expect(openControl).toBeVisible();
		const tagName = await openControl.evaluate((el) => el.tagName);
		expect(tagName).toBe('BUTTON');

		// The old anchor-navigation affordance must not also be present for
		// this item — a local-exec link never renders as a real <a href>.
		await expect(page.getByRole('link', { name: `Open in ${DISPLAY_NAME}` })).toHaveCount(0);
	});
});
