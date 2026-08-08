// Unit coverage for plugin-fields.ts's three pure helpers (07-04-PLAN.md
// Task 1): titleCaseField's word-casing, parseMatchValues' blank/all-blank/
// whitespace-around-comma handling, and connectionFieldsFor's known- and
// unknown-plugin-type cases.

import { describe, it, expect } from 'vitest';
import { connectionFieldsFor, parseMatchValues, titleCaseField } from './plugin-fields';

describe('titleCaseField', () => {
	it('converts underscores to spaces and capitalises each word', () => {
		expect(titleCaseField('conversation_names')).toBe('Conversation Names');
	});

	it('leaves a single-word field capitalised', () => {
		expect(titleCaseField('tags')).toBe('Tags');
		expect(titleCaseField('folders')).toBe('Folders');
	});
});

describe('parseMatchValues', () => {
	it('splits a comma-separated string, trimming each part', () => {
		expect(parseMatchValues('a, b, c')).toEqual(['a', 'b', 'c']);
	});

	it('returns an empty array for a blank input', () => {
		expect(parseMatchValues('')).toEqual([]);
	});

	it('returns an empty array for an all-blank input (only commas and whitespace)', () => {
		expect(parseMatchValues(' , , ')).toEqual([]);
	});

	it('trims whitespace around commas', () => {
		expect(parseMatchValues('  boiler  ,  quote  ')).toEqual(['boiler', 'quote']);
	});

	it('drops empty members between two real values', () => {
		expect(parseMatchValues('boiler,,quote')).toEqual(['boiler', 'quote']);
	});
});

describe('connectionFieldsFor', () => {
	it('returns the ordered field set for a known plugin type', () => {
		const fields = connectionFieldsFor('topos-plugin-paperless');
		expect(fields.map((f) => f.key)).toEqual([
			'display_name',
			'base_url',
			'token',
			'api_version',
			'sync_interval'
		]);
	});

	it('marks the expected fields required/secret/advanced', () => {
		const fields = connectionFieldsFor('topos-plugin-paperless');
		expect(fields.find((f) => f.key === 'base_url')?.required).toBe(true);
		expect(fields.find((f) => f.key === 'token')?.secret).toBe(true);
		expect(fields.find((f) => f.key === 'sync_interval')?.advanced).toBe(true);
	});

	it('gives Signal a placeholder local path and no base_url/token fields', () => {
		const fields = connectionFieldsFor('topos-plugin-signal');
		expect(fields.map((f) => f.key)).toEqual(['display_name', 'path', 'sync_interval']);
		expect(fields.find((f) => f.key === 'path')?.placeholder).toBe('~/.config/Signal');
	});

	it('falls back to a minimal field set for an unknown plugin type', () => {
		const fields = connectionFieldsFor('topos-plugin-does-not-exist');
		expect(fields.map((f) => f.key)).toEqual(['display_name', 'sync_interval']);
	});
});
