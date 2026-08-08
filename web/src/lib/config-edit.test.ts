// Unit coverage for config-edit.ts's four pure document-edit helpers
// (07-03-PLAN.md Task 2). Every function's own contract is "returns a NEW
// document, never mutates its input" — proven below by asserting the input
// reference's own JSON serialization is byte-identical before and after
// each call, not merely that the function's declared return type looks
// right.

import { describe, it, expect } from 'vitest';
import {
	cloneConfig,
	addWebspace,
	removeWebspace,
	setWebspaceFilter,
	setMatchBlock,
	addSourceToWebspace,
	removeSourceFromWebspace,
	upsertSourceInstance,
	removeSourceInstance
} from './config-edit';
import type { KernelConfig } from './api';

function fixtureConfig(): KernelConfig {
	return {
		server: { listen: '127.0.0.1:7777' },
		index: { path: '/tmp/index.db' },
		plugins: { dir: '/tmp/plugins' },
		sync: { interval: '5m' },
		sources: {
			paperless: {
				plugin: 'topos-plugin-paperless',
				base_url: 'https://paperless.example',
				token: '${PAPERLESS_TOKEN}',
				agent: { read: true, handoff: false }
			},
			silverbullet: {
				plugin: 'topos-plugin-silverbullet',
				base_url: 'https://sb.example',
				token: '${SB_TOKEN}',
				agent: { read: true, handoff: false }
			}
		},
		webspaces: {
			'house-move': {
				keywords: ['boiler'],
				sources: ['paperless'],
				match: { paperless: { tags: ['house-move'] } },
				filter: ['quote']
			},
			'catch-all': {
				keywords: ['misc'],
				sources: [],
				match: {}
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

	it('never adds a sources allowlist to a webspace that had none (D-14: only addSourceToWebspace seeds one)', () => {
		const cfg = fixtureConfig();
		const next = setWebspaceFilter(cfg, 'catch-all', ['urgent']);
		expect(next.webspaces['catch-all'].sources).toEqual([]);
	});
});

describe('setMatchBlock', () => {
	it('writes a new match block for an instance with no existing block', () => {
		const cfg = fixtureConfig();
		const next = setMatchBlock(cfg, 'catch-all', 'silverbullet', { tags: ['project-x'] });
		expect(next.webspaces['catch-all'].match).toEqual({ silverbullet: { tags: ['project-x'] } });
	});

	it('replaces an existing match block for the same instance', () => {
		const cfg = fixtureConfig();
		const next = setMatchBlock(cfg, 'house-move', 'paperless', { tags: ['renamed'] });
		expect(next.webspaces['house-move'].match).toEqual({ paperless: { tags: ['renamed'] } });
	});

	it('deletes the entry outright when the block has no fields', () => {
		const cfg = fixtureConfig();
		const next = setMatchBlock(cfg, 'house-move', 'paperless', {});
		expect(next.webspaces['house-move'].match).toEqual({});
	});

	it('preserves keywords/sources/filter unrelated to the match write', () => {
		const cfg = fixtureConfig();
		const next = setMatchBlock(cfg, 'house-move', 'paperless', { tags: ['x'] });
		expect(next.webspaces['house-move'].keywords).toEqual(['boiler']);
		expect(next.webspaces['house-move'].sources).toEqual(['paperless']);
		expect(next.webspaces['house-move'].filter).toEqual(['quote']);
	});

	it('creates the webspace with empty defaults when absent from the input', () => {
		const cfg = fixtureConfig();
		const next = setMatchBlock(cfg, 'brand-new', 'paperless', { tags: ['x'] });
		expect(next.webspaces['brand-new']).toEqual({
			keywords: [],
			sources: [],
			match: { paperless: { tags: ['x'] } }
		});
	});

	it('leaves the input document untouched', () => {
		const cfg = fixtureConfig();
		const before = JSON.stringify(cfg);
		setMatchBlock(cfg, 'house-move', 'paperless', { tags: ['x'] });
		expect(JSON.stringify(cfg)).toBe(before);
	});
});

describe('addSourceToWebspace', () => {
	it('appends to an existing allowlist without reordering', () => {
		const cfg = fixtureConfig();
		const next = addSourceToWebspace(cfg, 'house-move', 'silverbullet', { tags: ['x'] });
		expect(next.webspaces['house-move'].sources).toEqual(['paperless', 'silverbullet']);
	});

	it('seeds an allowlist-free webspace with every previously participating instance plus the new one', () => {
		const cfg = fixtureConfig();
		const next = addSourceToWebspace(cfg, 'catch-all', 'silverbullet', { tags: ['x'] });
		expect(next.webspaces['catch-all'].sources).toEqual(['paperless', 'silverbullet']);
	});

	it('writes the match block alongside the allowlist in one call', () => {
		const cfg = fixtureConfig();
		const next = addSourceToWebspace(cfg, 'catch-all', 'silverbullet', { tags: ['project-x'] });
		expect(next.webspaces['catch-all'].match).toEqual({ silverbullet: { tags: ['project-x'] } });
	});

	it('adding an instance already present in the allowlist does not duplicate it', () => {
		const cfg = fixtureConfig();
		const next = addSourceToWebspace(cfg, 'house-move', 'paperless', { tags: ['y'] });
		expect(next.webspaces['house-move'].sources).toEqual(['paperless']);
	});

	it('leaves the input document untouched', () => {
		const cfg = fixtureConfig();
		const before = JSON.stringify(cfg);
		addSourceToWebspace(cfg, 'catch-all', 'silverbullet', { tags: ['x'] });
		expect(JSON.stringify(cfg)).toBe(before);
	});
});

describe('removeSourceFromWebspace', () => {
	it('removes both the match block and the allowlist entry', () => {
		const cfg = fixtureConfig();
		const next = removeSourceFromWebspace(cfg, 'house-move', 'paperless');
		expect(next.webspaces['house-move'].match).toEqual({});
		expect(next.webspaces['house-move'].sources).toEqual([]);
	});

	it('is a no-op (beyond the clone) for a webspace absent from the input', () => {
		const cfg = fixtureConfig();
		const next = removeSourceFromWebspace(cfg, 'does-not-exist', 'paperless');
		expect(next).toEqual(cfg);
	});

	it('leaves the input document untouched', () => {
		const cfg = fixtureConfig();
		const before = JSON.stringify(cfg);
		removeSourceFromWebspace(cfg, 'house-move', 'paperless');
		expect(JSON.stringify(cfg)).toBe(before);
	});
});

describe('upsertSourceInstance', () => {
	it('writes a brand-new instance', () => {
		const cfg = fixtureConfig();
		const next = upsertSourceInstance(cfg, 'proton-home', {
			plugin: 'topos-plugin-proton',
			base_url: 'imaps://mail.example',
			username: 'me@example.com',
			token: '${PROTON_TOKEN}',
			agent: { read: false, handoff: false }
		});
		expect(next.sources['proton-home']).toEqual({
			plugin: 'topos-plugin-proton',
			base_url: 'imaps://mail.example',
			username: 'me@example.com',
			token: '${PROTON_TOKEN}',
			agent: { read: false, handoff: false }
		});
	});

	it('replaces an existing instance wholesale', () => {
		const cfg = fixtureConfig();
		const next = upsertSourceInstance(cfg, 'paperless', {
			plugin: 'topos-plugin-paperless',
			base_url: 'https://paperless.renamed',
			token: '${PAPERLESS_TOKEN_2}',
			agent: { read: true, handoff: false }
		});
		expect(next.sources['paperless'].base_url).toBe('https://paperless.renamed');
		expect(next.sources['paperless'].token).toBe('${PAPERLESS_TOKEN_2}');
	});

	it('leaves every other instance untouched', () => {
		const cfg = fixtureConfig();
		const next = upsertSourceInstance(cfg, 'paperless', {
			plugin: 'topos-plugin-paperless',
			base_url: 'https://paperless.renamed',
			token: '${X}',
			agent: { read: true, handoff: false }
		});
		expect(next.sources['silverbullet']).toEqual(cfg.sources['silverbullet']);
	});

	it('leaves the input document untouched', () => {
		const cfg = fixtureConfig();
		const before = JSON.stringify(cfg);
		upsertSourceInstance(cfg, 'paperless', {
			plugin: 'topos-plugin-paperless',
			base_url: 'https://mutated',
			token: '${X}',
			agent: { read: true, handoff: false }
		});
		expect(JSON.stringify(cfg)).toBe(before);
	});
});

describe('removeSourceInstance', () => {
	it('removes the [sources.<id>] block entirely, leaving every other instance untouched', () => {
		const cfg = fixtureConfig();
		const next = removeSourceInstance(cfg, 'paperless');
		expect(next.sources['paperless']).toBeUndefined();
		expect(next.sources['silverbullet']).toEqual(cfg.sources['silverbullet']);
	});

	it('clears the match block AND the allowlist entry in every webspace referencing it (an instance referenced by two webspaces has both cleared)', () => {
		const cfg = fixtureConfig();
		// Reference "silverbullet" from BOTH fixture webspaces before removing it.
		const withRefs = addSourceToWebspace(
			addSourceToWebspace(cfg, 'house-move', 'silverbullet', { tags: ['x'] }),
			'catch-all',
			'silverbullet',
			{ tags: ['y'] }
		);
		expect(withRefs.webspaces['house-move'].match).toHaveProperty('silverbullet');
		expect(withRefs.webspaces['catch-all'].match).toHaveProperty('silverbullet');

		const next = removeSourceInstance(withRefs, 'silverbullet');
		expect(next.webspaces['house-move'].match).toEqual({ paperless: { tags: ['house-move'] } });
		expect(next.webspaces['house-move'].sources).toEqual(['paperless']);
		expect(next.webspaces['catch-all'].match).toEqual({});
		// catch-all's allowlist was SEEDED by addSourceToWebspace (D-14, from
		// every instance configured at seed time: paperless + silverbullet)
		// before silverbullet was appended — removing silverbullet here
		// leaves that seeded "paperless" entry in place, it does not empty
		// the array back out.
		expect(next.webspaces['catch-all'].sources).toEqual(['paperless']);
	});

	it('leaves a webspace that never referenced the removed instance byte-identical', () => {
		const cfg = fixtureConfig();
		const before = JSON.stringify(cfg.webspaces['catch-all']);
		const next = removeSourceInstance(cfg, 'paperless');
		expect(JSON.stringify(next.webspaces['catch-all'])).toBe(before);
	});

	it('removing an instance no webspace references is a no-op for every webspace document', () => {
		const cfg = fixtureConfig();
		const next = removeSourceInstance(cfg, 'silverbullet');
		expect(next.webspaces).toEqual(cfg.webspaces);
	});

	it('an allowlist left empty by the removal is written as [] — Webspace.Participates treats empty identically to absent (all instances participate), so this is the kernel default, never a dangling reference', () => {
		const cfg = fixtureConfig();
		const next = removeSourceInstance(cfg, 'paperless');
		expect(next.webspaces['house-move'].sources).toEqual([]);
	});

	it('leaves the input document untouched', () => {
		const cfg = fixtureConfig();
		const before = JSON.stringify(cfg);
		removeSourceInstance(cfg, 'paperless');
		expect(JSON.stringify(cfg)).toBe(before);
	});
});
