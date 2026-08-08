// Unit coverage for config-edit.ts's four pure document-edit helpers
// (07-03-PLAN.md Task 2). Every function's own contract is "returns a NEW
// document, never mutates its input" — proven below by asserting the input
// reference's own JSON serialization is byte-identical before and after
// each call, not merely that the function's declared return type looks
// right.

import { describe, it, expect } from 'vitest';
import { cloneConfig, addWebspace, removeWebspace, setWebspaceFilter } from './config-edit';
import type { KernelConfig } from './api';

function fixtureConfig(): KernelConfig {
	return {
		server: { listen: '127.0.0.1:7777' },
		index: { path: '/tmp/index.db' },
		plugins: { dir: '/tmp/plugins' },
		sync: { interval: '5m' },
		sources: {
			paperless: {
				plugin: 'paperless',
				base_url: 'https://paperless.example',
				token: '${PAPERLESS_TOKEN}',
				agent: { read: true, handoff: false }
			}
		},
		webspaces: {
			'house-move': {
				keywords: ['boiler'],
				sources: ['paperless'],
				match: { paperless: { tags: ['house-move'] } },
				filter: ['quote']
			}
		}
	};
}

describe('cloneConfig', () => {
	it('returns a deep copy, not the same reference', () => {
		const cfg = fixtureConfig();
		const cloned = cloneConfig(cfg);
		expect(cloned).not.toBe(cfg);
		expect(cloned.webspaces).not.toBe(cfg.webspaces);
		expect(cloned.webspaces['house-move']).not.toBe(cfg.webspaces['house-move']);
		expect(cloned).toEqual(cfg);
	});

	it('leaves the input untouched when the clone is mutated', () => {
		const cfg = fixtureConfig();
		const before = JSON.stringify(cfg);
		const cloned = cloneConfig(cfg);
		cloned.webspaces['house-move'].filter = ['mutated'];
		delete cloned.webspaces['house-move'];
		expect(JSON.stringify(cfg)).toBe(before);
	});
});

describe('addWebspace', () => {
	it('adds an empty webspace entry with no sources allowlist yet (D-14)', () => {
		const cfg = fixtureConfig();
		const next = addWebspace(cfg, 'new-project');
		expect(next.webspaces['new-project']).toEqual({ keywords: [], sources: [], match: {} });
		expect(next.webspaces['new-project'].filter).toBeUndefined();
	});

	it('leaves the input document untouched', () => {
		const cfg = fixtureConfig();
		const before = JSON.stringify(cfg);
		addWebspace(cfg, 'new-project');
		expect(JSON.stringify(cfg)).toBe(before);
		expect(cfg.webspaces['new-project']).toBeUndefined();
	});
});

describe('removeWebspace', () => {
	it('removes the named webspace entirely', () => {
		const cfg = fixtureConfig();
		const next = removeWebspace(cfg, 'house-move');
		expect(next.webspaces['house-move']).toBeUndefined();
	});

	it('is a no-op (beyond the clone) for a name absent from the input', () => {
		const cfg = fixtureConfig();
		const next = removeWebspace(cfg, 'does-not-exist');
		expect(next).toEqual(cfg);
	});

	it('leaves the input document untouched', () => {
		const cfg = fixtureConfig();
		const before = JSON.stringify(cfg);
		removeWebspace(cfg, 'house-move');
		expect(JSON.stringify(cfg)).toBe(before);
	});
});

describe('setWebspaceFilter', () => {
	it('replaces the filter stack while preserving keywords/sources/match', () => {
		const cfg = fixtureConfig();
		const next = setWebspaceFilter(cfg, 'house-move', ['boiler-quote', 'urgent']);
		expect(next.webspaces['house-move']).toEqual({
			keywords: ['boiler'],
			sources: ['paperless'],
			match: { paperless: { tags: ['house-move'] } },
			filter: ['boiler-quote', 'urgent']
		});
	});

	it('creates the webspace with empty defaults when absent from the input', () => {
		const cfg = fixtureConfig();
		const next = setWebspaceFilter(cfg, 'brand-new', ['first-term']);
		expect(next.webspaces['brand-new']).toEqual({
			keywords: [],
			sources: [],
			match: {},
			filter: ['first-term']
		});
	});

	it('leaves the input document untouched', () => {
		const cfg = fixtureConfig();
		const before = JSON.stringify(cfg);
		setWebspaceFilter(cfg, 'house-move', ['mutated']);
		expect(JSON.stringify(cfg)).toBe(before);
	});
});
