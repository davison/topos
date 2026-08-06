// highlight.test.ts covers the client half of UI-09's search-term
// highlighting rule — highlightTerms (term derivation) and highlightText
// (segmentation) in $lib/format. The kernel half of the same rule
// (kernel/httpapi/rendition.go's highlightTerms/highlightTextNodes) has
// its own coverage in kernel/httpapi/rendition_test.go (Task 3); this
// file's job is to prove the client's independent implementation of the
// identical rule behaves correctly on its own terms.
import { describe, it, expect } from 'vitest';
import { highlightTerms, highlightText } from '$lib/format';

describe('highlightTerms', () => {
	it('drops terms shorter than 2 characters', () => {
		expect(highlightTerms('a bb c dd')).toEqual(['bb', 'dd']);
	});

	it('caps the result at the first 8 terms', () => {
		const query = Array.from({ length: 12 }, (_, i) => `term${i}`).join(' ');
		expect(highlightTerms(query)).toHaveLength(8);
	});

	it('de-duplicates repeated terms, keeping the first occurrence order', () => {
		expect(highlightTerms('hello world hello')).toEqual(['hello', 'world']);
	});

	it('lowercases every derived term', () => {
		expect(highlightTerms('HELLO World')).toEqual(['hello', 'world']);
	});

	it('returns an empty array for an empty query', () => {
		expect(highlightTerms('')).toEqual([]);
	});

	it('returns an empty array for a whitespace-only query', () => {
		expect(highlightTerms('   \t\n  ')).toEqual([]);
	});

	it('returns an empty array when every field is dropped (all sub-2-character)', () => {
		expect(highlightTerms('a b c')).toEqual([]);
	});
});

describe('highlightText', () => {
	it('returns a single unmatched segment for an empty query', () => {
		const segments = highlightText('hello world', '');
		expect(segments).toEqual([{ text: 'hello world', match: false }]);
	});

	it('returns an empty array for empty text', () => {
		expect(highlightText('', 'hello')).toEqual([]);
	});

	it('matches case-insensitively while preserving the source text original casing', () => {
		const segments = highlightText('Hello World', 'hello');
		expect(segments).toEqual([
			{ text: 'Hello', match: true },
			{ text: ' World', match: false }
		]);
	});

	it('matches a query containing regex metacharacters literally and never throws', () => {
		expect(() => highlightText('price: $5.00 (on sale)', '$5.00 (on')).not.toThrow();
		const segments = highlightText('price: $5.00 (on sale)', '$5.00 (on');
		expect(segments.some((s) => s.match)).toBe(true);
	});

	it('a zero-match query renders no highlights', () => {
		const segments = highlightText('hello world', 'xyz');
		expect(segments).toEqual([{ text: 'hello world', match: false }]);
	});

	it('resolves overlapping terms longest-first with no nested or duplicated segments', () => {
		// "cat" and "category" both start at the same position; "category"
		// must win as the longer term, producing one matched segment, not
		// a matched "cat" followed by an unmatched/duplicated "egory".
		const segments = highlightText('category theory', 'cat category');
		expect(segments[0]).toEqual({ text: 'category', match: true });
		expect(segments.some((s) => s.text === 'cat' && s.match)).toBe(false);
	});

	it('the round-trip invariant: concatenating every segment text reproduces the input exactly', () => {
		const cases: Array<[string, string]> = [
			['hello world, hello universe', 'hello'],
			['no matches here', 'xyz'],
			['', 'hello'],
			['CaSe MiXeD text', 'case mixed'],
			['a b c hello world', 'hello world']
		];
		for (const [text, query] of cases) {
			const segments = highlightText(text, query);
			expect(segments.map((s) => s.text).join('')).toBe(text);
		}
	});

	it('never emits a matched segment for a term shorter than 2 characters', () => {
		const segments = highlightText('a b c hello', 'a b hello');
		expect(segments.some((s) => s.match && s.text.length < 2)).toBe(false);
	});
});
