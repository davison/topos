import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { detailBodyVariant } from '$lib/format';
import type { ItemContent } from '$lib/api';

function makeContent(overrides: Partial<ItemContent> = {}): ItemContent {
	return { available: true, unavailable_reason: '', text: '', rendition: null, ...overrides };
}

describe('detailBodyVariant', () => {
	it('is empty for null content', () => {
		expect(detailBodyVariant(null)).toBe('empty');
	});

	it('is empty when content.available is false, regardless of its other fields', () => {
		expect(
			detailBodyVariant(
				makeContent({
					available: false,
					text: 'some text',
					rendition: { mime_type: 'text/html', size_bytes: 10, url: 'https://example.test/x' }
				})
			)
		).toBe('empty');
	});

	it('is html for a text/html rendition (the SilverBullet shape: rendition AND non-empty text)', () => {
		expect(
			detailBodyVariant(
				makeContent({
					text: 'raw markdown source',
					rendition: { mime_type: 'text/html', size_bytes: 200, url: 'https://example.test/rendition' }
				})
			)
		).toBe('html');
	});

	it('is media for an application/pdf rendition (the paperless shape: rendition AND non-empty text)', () => {
		expect(
			detailBodyVariant(
				makeContent({
					text: 'extracted document text',
					rendition: { mime_type: 'application/pdf', size_bytes: 4096, url: 'https://example.test/doc.pdf' }
				})
			)
		).toBe('media');
	});

	it('is media for an image/* rendition', () => {
		expect(
			detailBodyVariant(
				makeContent({
					rendition: { mime_type: 'image/png', size_bytes: 2048, url: 'https://example.test/img.png' }
				})
			)
		).toBe('media');
	});

	it('is text for no rendition and non-empty text — the Proton plain-text-preferred shape', () => {
		expect(detailBodyVariant(makeContent({ text: 'Hello from the plain-text part.' }))).toBe('text');
	});

	it('is empty for no rendition and empty text', () => {
		expect(detailBodyVariant(makeContent({ text: '' }))).toBe('empty');
	});

	it('is empty for no rendition and whitespace-only text', () => {
		expect(detailBodyVariant(makeContent({ text: '   \n\t  ' }))).toBe('empty');
	});

	it('falls through to text for an unrecognised rendition mime type when text is present', () => {
		expect(
			detailBodyVariant(
				makeContent({
					text: 'fallback text',
					rendition: { mime_type: 'application/octet-stream', size_bytes: 1, url: 'https://example.test/x' }
				})
			)
		).toBe('text');
	});

	it('falls through to empty for an unrecognised rendition mime type with no text', () => {
		expect(
			detailBodyVariant(
				makeContent({
					text: '',
					rendition: { mime_type: 'application/octet-stream', size_bytes: 1, url: 'https://example.test/x' }
				})
			)
		).toBe('empty');
	});
});

describe('DetailPane source-scan guard', () => {
	const componentsDir = dirname(fileURLToPath(import.meta.url));
	const source = readFileSync(join(componentsDir, 'DetailPane.svelte'), 'utf-8');

	// The shared detail pane must never learn one source's name — its
	// branch choice is decided entirely by detailBodyVariant/detailPaneState
	// from the SHAPE of the content it is handed. Built from the configured
	// source type names (plugins/*/plugin.go's sourceType consts), checked
	// in both quote styles.
	const forbiddenSourceNames = ['paperless', 'silverbullet', 'proton', 'mock'];

	for (const name of forbiddenSourceNames) {
		it(`contains no quoted literal naming the "${name}" source`, () => {
			const singleQuoted = `'${name}'`;
			const doubleQuoted = `"${name}"`;
			const foundSingle = source.includes(singleQuoted);
			const foundDouble = source.includes(doubleQuoted);
			expect(
				foundSingle || foundDouble,
				`DetailPane.svelte contains a quoted literal naming source "${name}" — the shared pane must decide its branch from content shape alone, never a source identity`
			).toBe(false);
		});
	}
});
