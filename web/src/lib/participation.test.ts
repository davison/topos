// Unit coverage for participation.ts's null-tolerant readers and shell
// discriminator (07-11-PLAN.md Task 2). isEmptyWebspaceShell's three
// "any one non-empty means not a shell" cases are asserted independently
// so a two-condition implementation cannot pass.

import { describe, it, expect } from 'vitest';
import {
	webspaceKeywords,
	webspaceSources,
	webspaceMatch,
	isEmptyWebspaceShell,
	participatingInstances,
	participatesIn
} from './participation';
import type { KernelConfig, WebspaceConfig } from './api';

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

// --- participatingInstances / participatesIn (07-14-PLAN.md Task 1,
// closes 07-UAT.md G-07-6's second half) ---
//
// A THREE-instance fixture is required throughout: a two-instance fixture
// cannot distinguish "some but not all participate" from either the
// all-participate or the empty-set outcome, both of which a wrong (e.g.
// allowlist-gate-only) implementation would also produce.

function threeInstanceConfig(webspaces: Record<string, WebspaceConfig>): KernelConfig {
	const source = (plugin: string) => ({ plugin, agent: { read: true, handoff: false } });
	return {
		server: { listen: '127.0.0.1:7777' },
		index: { path: '/tmp/index.db' },
		plugins: { dir: '/tmp/plugins' },
		sync: { interval: '5m' },
		sources: {
			a: source('topos-plugin-a'),
			b: source('topos-plugin-b'),
			c: source('topos-plugin-c')
		},
		webspaces
	};
}

describe('participatingInstances / participatesIn', () => {
	// Every case below is collected here, then Both functions are asserted
	// against the SAME table in the loop that follows — participatesIn is
	// never independently duplicated case-by-case, so the two cannot
	// diverge without a test failing.
	const cases: { name: string; cfg: KernelConfig; webspace: string; expected: string[] }[] = [];

	cases.push({
		name:
			'explicit non-empty allowlist with match blocks: named instances participate — chips for a/b only, add-picker offers c',
		cfg: threeInstanceConfig({
			cars: {
				keywords: ['fallback-should-not-matter'],
				sources: ['a', 'b'],
				match: { a: { tags: ['x'] }, b: { tags: ['y'] } }
			}
		}),
		webspace: 'cars',
		expected: ['a', 'b']
	});

	cases.push({
		name:
			'empty allowlist plus non-empty keywords fallback: every configured instance participates — chips for a, b and c',
		cfg: threeInstanceConfig({
			cars: { keywords: ['fallback'], sources: [], match: {} }
		}),
		webspace: 'cars',
		expected: ['a', 'b', 'c']
	});

	cases.push({
		name:
			'empty allowlist, no keywords, match blocks for two of three: exactly those two participate — the third has no allowlist exclusion and no match input, so the kernel would sync nothing for it',
		cfg: threeInstanceConfig({
			cars: { keywords: [], sources: [], match: { a: { tags: ['x'] }, b: { tags: ['y'] } } }
		}),
		webspace: 'cars',
		expected: ['a', 'b']
	});

	cases.push({
		name: 'an 07-11 empty shell: no chips, and the add-picker offers every instance',
		cfg: threeInstanceConfig({ cars: { keywords: [], sources: [], match: {} } }),
		webspace: 'cars',
		expected: []
	});

	cases.push({
		name: 'a webspace absent from the config: no chips',
		cfg: threeInstanceConfig({}),
		webspace: 'does-not-exist',
		expected: []
	});

	cases.push({
		name:
			'a webspace arriving with null keywords/sources/match (wire shape older than the TS type): no chips, no exception',
		cfg: threeInstanceConfig({
			cars: { keywords: null, sources: null, match: null } as unknown as WebspaceConfig
		}),
		webspace: 'cars',
		expected: []
	});

	cases.push({
		name:
			'an allowlist naming an unconfigured instance: the phantom id never appears — the set is always a subset of the configured instances',
		cfg: threeInstanceConfig({
			cars: { keywords: [], sources: ['a', 'ghost'], match: { a: { tags: ['x'] } } }
		}),
		webspace: 'cars',
		expected: ['a']
	});

	for (const { name, cfg, webspace, expected } of cases) {
		it(name, () => {
			const set = participatingInstances(cfg, webspace);
			expect([...set].sort()).toEqual([...expected].sort());
			// The set is always a subset of the configured instances — the
			// assertion that fails if participatingInstances ever iterates the
			// allowlist instead of cfg.sources.
			for (const instance of set) {
				expect(Object.keys(cfg.sources)).toContain(instance);
			}
		});
	}

	it('participatesIn agrees with participatingInstances across every case above, for every configured instance', () => {
		for (const { cfg, webspace } of cases) {
			const set = participatingInstances(cfg, webspace);
			for (const instance of Object.keys(cfg.sources)) {
				expect(participatesIn(cfg, webspace, instance)).toBe(set.has(instance));
			}
		}
	});

	it('does not throw for a webspace whose collections all arrive as null', () => {
		const cfg = threeInstanceConfig({
			cars: { keywords: null, sources: null, match: null } as unknown as WebspaceConfig
		});
		expect(() => participatingInstances(cfg, 'cars')).not.toThrow();
		expect(() => participatesIn(cfg, 'cars', 'a')).not.toThrow();
	});
});
