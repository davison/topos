// Unit coverage for participation.ts's null-tolerant readers and shell
// discriminator (07-11-PLAN.md Task 2). isEmptyWebspaceShell's three
// "any one non-empty means not a shell" cases are asserted independently
// so a two-condition implementation cannot pass.

import { describe, it, expect } from 'vitest';
import { webspaceKeywords, webspaceSources, webspaceMatch, isEmptyWebspaceShell } from './participation';
import type { WebspaceConfig } from './api';

describe('webspaceKeywords / webspaceSources / webspaceMatch', () => {
	it('return the empty defaults for an all-empty webspace', () => {
		const ws: WebspaceConfig = { keywords: [], sources: [], match: {} };
		expect(webspaceKeywords(ws)).toEqual([]);
		expect(webspaceSources(ws)).toEqual([]);
		expect(webspaceMatch(ws)).toEqual({});
	});

	it('tolerate a wire document whose collections are null despite the TS type', () => {
		const ws = { keywords: null, sources: null, match: null } as unknown as WebspaceConfig;
		expect(webspaceKeywords(ws)).toEqual([]);
		expect(webspaceSources(ws)).toEqual([]);
		expect(webspaceMatch(ws)).toEqual({});
	});

	it('tolerate an undefined webspace (absent from this config snapshot)', () => {
		expect(webspaceKeywords(undefined)).toEqual([]);
		expect(webspaceSources(undefined)).toEqual([]);
		expect(webspaceMatch(undefined)).toEqual({});
	});

	it('return the webspace real values when present', () => {
		const ws: WebspaceConfig = {
			keywords: ['house'],
			sources: ['paperless'],
			match: { paperless: { tags: ['x'] } }
		};
		expect(webspaceKeywords(ws)).toEqual(['house']);
		expect(webspaceSources(ws)).toEqual(['paperless']);
		expect(webspaceMatch(ws)).toEqual({ paperless: { tags: ['x'] } });
	});
});

describe('isEmptyWebspaceShell', () => {
	it('is true for {keywords: [], sources: [], match: {}} — the literal addWebspace shape', () => {
		expect(isEmptyWebspaceShell({ keywords: [], sources: [], match: {} })).toBe(true);
	});

	it('is true for a webspace whose keywords/sources/match are null on the wire', () => {
		const ws = { keywords: null, sources: null, match: null } as unknown as WebspaceConfig;
		expect(isEmptyWebspaceShell(ws)).toBe(true);
	});

	it('is true for undefined — a webspace not present in this snapshot has nothing to match', () => {
		expect(isEmptyWebspaceShell(undefined)).toBe(true);
	});

	it('is false when keywords alone is non-empty', () => {
		expect(isEmptyWebspaceShell({ keywords: ['house'], sources: [], match: {} })).toBe(false);
	});

	it('is false when sources alone is non-empty (the operator-typo shape stays disqualified)', () => {
		expect(isEmptyWebspaceShell({ keywords: [], sources: ['paperless'], match: {} })).toBe(false);
	});

	it('is false when match alone is non-empty', () => {
		expect(
			isEmptyWebspaceShell({ keywords: [], sources: [], match: { paperless: { tags: ['x'] } } })
		).toBe(false);
	});
});
