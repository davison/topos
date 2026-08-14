import { describe, it, expect } from 'vitest';
import { markPhrase } from './toast';

// Asserts the pluralization decision against 13-UI-SPEC.md's Copywriting
// Contract literal strings — count 1 is singular, every other count
// (including 0) is plural, for both verbs.
describe('markPhrase', () => {
	it('Excluded, count 0 -> "Excluded 0 items"', () => {
		expect(markPhrase('Excluded', 0)).toBe('Excluded 0 items');
	});

	it('Excluded, count 1 -> "Excluded 1 item"', () => {
		expect(markPhrase('Excluded', 1)).toBe('Excluded 1 item');
	});

	it('Excluded, count 2 -> "Excluded 2 items"', () => {
		expect(markPhrase('Excluded', 2)).toBe('Excluded 2 items');
	});

	it('Included, count 0 -> "Included 0 items"', () => {
		expect(markPhrase('Included', 0)).toBe('Included 0 items');
	});

	it('Included, count 1 -> "Included 1 item"', () => {
		expect(markPhrase('Included', 1)).toBe('Included 1 item');
	});

	it('Included, count 2 -> "Included 2 items"', () => {
		expect(markPhrase('Included', 2)).toBe('Included 2 items');
	});
});
