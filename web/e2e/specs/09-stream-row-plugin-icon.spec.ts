// Checkpoint-directed scope addition (09-02-PLAN.md Task 4 checkpoint
// feedback, Item 2): plugin identity icons must appear on each stream row,
// not only on the source chip, so a mixed cross-source pane is scannable
// by source at a glance. Two source instances of two DIFFERENT plugin
// types (topos-plugin-mock and topos-plugin-mockstrict) are attached to
// one webspace so its stream is genuinely mixed, then this spec proves in
// a real browser: each row's own icon decodes, the two rows' icons are
// distinct (not one shared/default glyph), and the row's existing
// Thumbnail element is unchanged — additive metadata, never a replacement,
// per the checkpoint's own explicit constraint ("the user explicitly likes
// them").
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { attachedWebspace, type FixtureConfigSpec } from '../fixtures/config-builder';

const WEBSPACE = 'mixed-icons';

const configSpec: FixtureConfigSpec = {
	sources: [
		{ id: 'mock', plugin: 'topos-plugin-mock', displayName: 'Mock Source' },
		// mockstrict's own main.go requires a non-empty `path` field (its
		// deliberate required-connection-field fixture mechanism, D-05) —
		// the path is never opened, only checked for non-emptiness.
		{
			id: 'mockstrict',
			plugin: 'topos-plugin-mockstrict',
			path: '/tmp/e2e-mockstrict-unused',
			displayName: 'Mockstrict Source'
		}
	],
	webspaces: [
		attachedWebspace(WEBSPACE, ['mock', 'mockstrict'], {
			mock: { labels: ['demo'] },
			mockstrict: { tags: ['strict'] }
		})
	],
	pluginBinaries: ['topos-plugin-mock', 'topos-plugin-mockstrict']
};

test.use({ configSpec });

test.describe('09-02 checkpoint follow-up: stream rows render per-source plugin icons in a mixed pane', () => {
	test('two rows from two different plugin types each render their own decoded, distinct icon', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock', 'mockstrict'], { logs: kernel.logs });

		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		// mock's fixed item "1" (label "demo") and mockstrict's fixed item
		// "s1" (label "strict") — both plugins' own hardcoded corpora
		// (plugins/mock/plugin.go, plugins/mockstrict/plugin.go).
		const mockRow = page.locator('[data-item-id]', {
			hasText: 'Welcome to the mock source'
		});
		const mockstrictRow = page.locator('[data-item-id]', {
			hasText: 'Mockstrict fixture: alpha record'
		});
		await expect(mockRow).toBeVisible();
		await expect(mockstrictRow).toBeVisible();

		const mockIcon = mockRow.locator('img[src$="/api/plugins/topos-plugin-mock/icon"]');
		const mockstrictIcon = mockstrictRow.locator(
			'img[src$="/api/plugins/topos-plugin-mockstrict/icon"]'
		);
		await expect(mockIcon).toBeVisible();
		await expect(mockstrictIcon).toBeVisible();

		// The standard 09-01's tracer spec set: an image is proven by its
		// own naturalWidth, never by the presence of an <img> tag alone.
		await expect
			.poll(async () => mockIcon.evaluate((el: HTMLImageElement) => el.naturalWidth))
			.toBeGreaterThan(0);
		await expect
			.poll(async () => mockstrictIcon.evaluate((el: HTMLImageElement) => el.naturalWidth))
			.toBeGreaterThan(0);

		// The whole point of this checkpoint follow-up: two rows from two
		// different plugin types render two DIFFERENT icon sources, not one
		// shared/default glyph both rows happen to fall back to.
		const mockSrc = await mockIcon.getAttribute('src');
		const mockstrictSrc = await mockstrictIcon.getAttribute('src');
		expect(mockSrc).not.toEqual(mockstrictSrc);
	});

	test('the row with a real thumbnail (mock item "4") still renders its Thumbnail alongside its own plugin icon', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock', 'mockstrict'], { logs: kernel.logs });

		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		// mock's fixed item "4" ("Shopping list") is the one item in either
		// fixture's corpus with has_thumbnail: true (plugins/mock/plugin.go)
		// — the mock plugin's Fetch never has real thumbnail bytes to
		// serve, so Thumbnail.svelte's own onerror fallback (a generic
		// FileText document glyph) is the correctly-unchanged rendering
		// here; this spec cares only that it renders AT ALL, alongside the
		// row's own new plugin icon, not that a real image byte-decodes.
		const row = page.locator('[data-item-id]', { hasText: 'Shopping list' });
		await expect(row).toBeVisible();

		// The plugin icon is present (additive metadata)...
		const icon = row.locator('img[src$="/api/plugins/topos-plugin-mock/icon"]');
		await expect(icon).toBeVisible();

		// ...and the row's leading Thumbnail slot still renders (either the
		// real <img> or its FileText fallback svg) — never displaced by the
		// new icon. Scoped to the leading slot specifically (:first-child)
		// so this can never accidentally count the plugin icon's own <img>.
		const thumbnailSlot = row.locator(':scope > img:first-child, :scope > div:first-child svg');
		await expect(thumbnailSlot).toHaveCount(1);
	});
});
