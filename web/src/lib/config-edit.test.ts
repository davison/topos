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
	removeSourceInstance,
	setTrustedKey,
	removeTrustedKey,
	setSourceFilterTerms,
	splitFilterInput,
	setWebspaceDateRange,
	renameWebspace,
} from './config-edit';
import { isEmptyWebspaceShell } from './participation';
import type { KernelConfig, WebspaceConfig } from './api';

// Three-instance fixture for removeSourceFromWebspace's seeding cases
// below (07-14-PLAN.md Task 2, closes 07-UAT.md G-07-6's first half): a
// two-instance fixture cannot distinguish "the other one remains" from
// "the allowlist widened to everyone" or "the allowlist stayed empty" — a
// three-instance config makes each outcome distinguishable. `docs` is a
// second, untouched webspace present in every case, asserted unchanged.
function carsConfig(ws: WebspaceConfig): KernelConfig {
	const instance = (plugin: string) => ({
		plugin,
		agent: { read: true, handoff: false }
	});
	return {
		server: { listen: '127.0.0.1:7777' },
		index: { path: '/tmp/index.db' },
		plugins: { dir: '/tmp/plugins' },
		sync: { interval: '5m' },
		sources: {
			a: instance('topos-plugin-a'),
			b: instance('topos-plugin-b'),
			c: instance('topos-plugin-c')
		},
		webspaces: {
			cars: ws,
			docs: {
				keywords: ['paperwork'],
				sources: ['a'],
				match: { a: { tags: ['docs'] } }
			}
		}
	};
}

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
		expect(next.webspaces['new-project']).toEqual({
			keywords: [],
			sources: [],
			match: {}
		});
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
		const next = setMatchBlock(cfg, 'catch-all', 'silverbullet', {
			tags: ['project-x']
		});
		expect(next.webspaces['catch-all'].match).toEqual({
			silverbullet: { tags: ['project-x'] }
		});
	});

	it('replaces an existing match block for the same instance', () => {
		const cfg = fixtureConfig();
		const next = setMatchBlock(cfg, 'house-move', 'paperless', {
			tags: ['renamed']
		});
		expect(next.webspaces['house-move'].match).toEqual({
			paperless: { tags: ['renamed'] }
		});
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
		const next = addSourceToWebspace(cfg, 'house-move', 'silverbullet', {
			tags: ['x']
		});
		expect(next.webspaces['house-move'].sources).toEqual(['paperless', 'silverbullet']);
	});

	it('seeds an allowlist-free webspace with every previously participating instance plus the new one', () => {
		const cfg = fixtureConfig();
		const next = addSourceToWebspace(cfg, 'catch-all', 'silverbullet', {
			tags: ['x']
		});
		expect(next.webspaces['catch-all'].sources).toEqual(['paperless', 'silverbullet']);
	});

	it('writes the match block alongside the allowlist in one call', () => {
		const cfg = fixtureConfig();
		const next = addSourceToWebspace(cfg, 'catch-all', 'silverbullet', {
			tags: ['project-x']
		});
		expect(next.webspaces['catch-all'].match).toEqual({
			silverbullet: { tags: ['project-x'] }
		});
	});

	it('adding an instance already present in the allowlist does not duplicate it', () => {
		const cfg = fixtureConfig();
		const next = addSourceToWebspace(cfg, 'house-move', 'paperless', {
			tags: ['y']
		});
		expect(next.webspaces['house-move'].sources).toEqual(['paperless']);
	});

	it('leaves the input document untouched', () => {
		const cfg = fixtureConfig();
		const before = JSON.stringify(cfg);
		addSourceToWebspace(cfg, 'catch-all', 'silverbullet', { tags: ['x'] });
		expect(JSON.stringify(cfg)).toBe(before);
	});

	// --- D-20 shell-aware seeding (07-11-PLAN.md Task 2, closes 07-UAT.md
	// G-07-3): a webspace created by addWebspace has no participants to
	// preserve — seeding it from every configured instance would silently
	// drag every OTHER source into a webspace the user just created, and
	// produce a document the kernel rejects. ---

	it('does not seed the allowlist with every configured instance for a freshly-created (D-20 empty shell) webspace', () => {
		const withShell = addWebspace(fixtureConfig(), 'new-project');
		// A third configured instance exists in this cfg beyond paperless —
		// the assertion below fails if the seeding branch is still
		// unconditional.
		const withThird = upsertSourceInstance(withShell, 'proton-work', {
			plugin: 'topos-plugin-proton',
			base_url: 'https://proton.example',
			token: '${PROTON_TOKEN}',
			agent: { read: true, handoff: false }
		});
		const next = addSourceToWebspace(withThird, 'new-project', 'paperless', {
			tags: ['x']
		});
		expect(next.webspaces['new-project'].sources).toEqual(['paperless']);
	});

	it('seeds every configured instance without throwing when a hand-written webspace arrives with a null sources allowlist', () => {
		const cfg = fixtureConfig();
		const withNullSources = cloneConfig(cfg);
		(
			withNullSources.webspaces['catch-all'] as unknown as {
				sources: string[] | null;
			}
		).sources = null;
		const next = addSourceToWebspace(withNullSources, 'catch-all', 'silverbullet', { tags: ['x'] });
		expect(next.webspaces['catch-all'].sources).toEqual(['paperless', 'silverbullet']);
	});

	it('sequenced create-then-compose: addWebspace then addSourceToWebspace twice yields exactly those two instances in add order', () => {
		const created = addWebspace(fixtureConfig(), 'brand-new');
		const withFirst = addSourceToWebspace(created, 'brand-new', 'paperless', {
			tags: ['a']
		});
		const withSecond = addSourceToWebspace(withFirst, 'brand-new', 'silverbullet', { tags: ['b'] });
		expect(withSecond.webspaces['brand-new'].sources).toEqual(['paperless', 'silverbullet']);
		expect(withSecond.webspaces['brand-new'].match).toEqual({
			paperless: { tags: ['a'] },
			silverbullet: { tags: ['b'] }
		});
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

	// --- Seeding before filtering (07-14-PLAN.md Task 2, closes
	// 07-UAT.md G-07-6's first half). The `house-move` fixture above starts
	// from an explicit non-empty allowlist and is exactly why the reported
	// case was never caught — every case below exercises the empty/null
	// all-participate starting shape instead, using the three-instance
	// `carsConfig` fixture so "the other two remain" is distinguishable
	// from both "unchanged" and "widened to everyone". ---

	it('the reported G-07-6 case: an all-participate webspace (no explicit allowlist) loses exactly the removed instance, not nothing — the write returns 200, the removed instance keeps participating via the all-participate default, and the chip never disappears if this seed is missing', () => {
		const cfg = carsConfig({
			keywords: [],
			sources: [],
			match: { a: { tags: ['x'] }, b: { tags: ['y'] }, c: { tags: ['z'] } }
		});
		const next = removeSourceFromWebspace(cfg, 'cars', 'b');
		expect([...next.webspaces['cars'].sources].sort()).toEqual(['a', 'c']);
		expect(next.webspaces['cars'].match).toEqual({
			a: { tags: ['x'] },
			c: { tags: ['z'] }
		});
	});

	it('seeds every configured instance without throwing when the allowlist arrives as null', () => {
		const cfg = carsConfig({
			keywords: [],
			sources: null,
			match: { a: { tags: ['x'] }, b: { tags: ['y'] } }
		} as unknown as WebspaceConfig);
		const next = removeSourceFromWebspace(cfg, 'cars', 'b');
		expect([...next.webspaces['cars'].sources].sort()).toEqual(['a', 'c']);
	});

	it('preserves relative order when narrowing an already-explicit allowlist', () => {
		const cfg = carsConfig({
			keywords: [],
			sources: ['c', 'a', 'b'],
			match: { a: { tags: ['x'] }, b: { tags: ['y'] }, c: { tags: ['z'] } }
		});
		const next = removeSourceFromWebspace(cfg, 'cars', 'a');
		expect(next.webspaces['cars'].sources).toEqual(['c', 'b']);
	});

	// Pinned known boundary (07-14-PLAN.md planning choice 5): removing the
	// LAST named entry from an explicit allowlist leaves an empty
	// allowlist, which the config format can only encode as "all
	// participate" — for a webspace that still declares a keywords
	// fallback, the remaining configured instances rejoin. This is the
	// existing encoding's only representable outcome (there is no
	// "explicit allowlist of zero, still narrowed" shape distinct from the
	// default) — a known boundary, pinned here, not a redesign target for
	// this gap-closure plan.
	it('pinned boundary: removing the last named entry from an explicit allowlist reverts to all-participate for a webspace with a keywords fallback (planning choice 5)', () => {
		const cfg = carsConfig({
			keywords: ['fallback'],
			sources: ['a'],
			match: { a: { tags: ['x'] } }
		});
		const next = removeSourceFromWebspace(cfg, 'cars', 'a');
		expect(next.webspaces['cars'].sources).toEqual([]);
	});

	it('the same boundary without a keywords fallback yields a D-20 empty shell — invisible on the UI-built path, since such a webspace has no participants at all', () => {
		const cfg = carsConfig({
			keywords: [],
			sources: ['a'],
			match: { a: { tags: ['x'] } }
		});
		const next = removeSourceFromWebspace(cfg, 'cars', 'a');
		expect(isEmptyWebspaceShell(next.webspaces['cars'])).toBe(true);
	});

	it('removing an instance absent from both the allowlist and the match map is a no-op beyond the clone — no accidental widening', () => {
		const cfg = carsConfig({
			keywords: [],
			sources: ['a', 'b'],
			match: { a: { tags: ['x'] }, b: { tags: ['y'] } }
		});
		const next = removeSourceFromWebspace(cfg, 'cars', 'c');
		expect(next.webspaces['cars'].sources).toEqual(['a', 'b']);
		expect(next.webspaces['cars'].match).toEqual({
			a: { tags: ['x'] },
			b: { tags: ['y'] }
		});
	});

	it('removing the same instance twice in sequence yields equal results (idempotent)', () => {
		const cfg = carsConfig({
			keywords: [],
			sources: [],
			match: { a: { tags: ['x'] }, b: { tags: ['y'] }, c: { tags: ['z'] } }
		});
		const once = removeSourceFromWebspace(cfg, 'cars', 'b');
		const twice = removeSourceFromWebspace(once, 'cars', 'b');
		expect(twice).toEqual(once);
	});

	it("leaves the instance's own [sources.<id>] block present and unchanged", () => {
		const cfg = carsConfig({
			keywords: [],
			sources: [],
			match: { a: { tags: ['x'] }, b: { tags: ['y'] }, c: { tags: ['z'] } }
		});
		const next = removeSourceFromWebspace(cfg, 'cars', 'b');
		expect(next.sources['b']).toEqual(cfg.sources['b']);
	});

	it('leaves every other webspace in the document unchanged', () => {
		const cfg = carsConfig({
			keywords: [],
			sources: [],
			match: { a: { tags: ['x'] }, b: { tags: ['y'] }, c: { tags: ['z'] } }
		});
		const next = removeSourceFromWebspace(cfg, 'cars', 'b');
		expect(next.webspaces['docs']).toEqual(cfg.webspaces['docs']);
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
		expect(next.webspaces['house-move'].match).toEqual({
			paperless: { tags: ['house-move'] }
		});
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

describe('setTrustedKey / removeTrustedKey (M2-R4)', () => {
	const base: KernelConfig = {
		server: { listen: '127.0.0.1:7777' },
		index: { path: 'x' },
		plugins: { dir: 'plugins', pins: { 'topos-plugin-x': 'a'.repeat(64) } },
		sync: { interval: '15m' },
		sources: {},
		webspaces: {}
	};
	const offer = {
		id: 'acme-2026a',
		fingerprint: 'ff'.repeat(32),
		public_key: 'AAAA'
	};
	it('adds the entry from the offer, stamping trusted_at, and leaves pins alone', () => {
		const next = setTrustedKey(base, offer, 'Acme', new Date('2026-09-02T10:00:00Z'));
		expect(next.plugins.trusted_keys).toEqual([
			{
				id: 'acme-2026a',
				public_key: 'AAAA',
				trusted_at: '2026-09-02T10:00:00.000Z',
				note: 'Acme'
			}
		]);
		expect(next.plugins.pins).toEqual(base.plugins.pins);
		expect(base.plugins.trusted_keys, 'must not mutate the input').toBeUndefined();
	});
	it('replaces an entry with the same id rather than duplicating it', () => {
		const once = setTrustedKey(base, offer);
		const twice = setTrustedKey(once, { ...offer, public_key: 'BBBB' });
		expect(twice.plugins.trusted_keys).toHaveLength(1);
		expect(twice.plugins.trusted_keys?.[0].public_key).toBe('BBBB');
	});
	it('removeTrustedKey drops the entry, and the table when it was the last', () => {
		const once = setTrustedKey(base, offer);
		const gone = removeTrustedKey(once, 'acme-2026a');
		expect('trusted_keys' in gone.plugins).toBe(false);
		const other = setTrustedKey(once, { ...offer, id: 'other' });
		expect(removeTrustedKey(other, 'acme-2026a').plugins.trusted_keys?.map((k) => k.id)).toEqual([
			'other'
		]);
	});
});

describe('setSourceFilterTerms / splitFilterInput (M2-R3, #55)', () => {
	const base = () =>
		({
			server: { listen: '' },
			paths: {},
			sources: { 'mock-01': { plugin: 'topos-plugin-mock' }, 'mock-02': { plugin: 'topos-plugin-mock' } },
			webspaces: {
				house: { keywords: ['demo'], sources: [], match: {}, filter: ['boiler'] }
			}
		}) as unknown as Parameters<typeof setSourceFilterTerms>[0];

	it('sets, replaces and removes an instance entry, preserving filter', () => {
		let cfg = setSourceFilterTerms(base(), 'house', 'mock-01', ['quote']);
		expect(cfg.webspaces.house.filter_by_source).toEqual({ 'mock-01': ['quote'] });
		expect(cfg.webspaces.house.filter).toEqual(['boiler']);
		cfg = setSourceFilterTerms(cfg, 'house', 'mock-01', []);
		expect(cfg.webspaces.house.filter_by_source).toBeUndefined();
	});

	it('setWebspaceFilter preserves filter_by_source', () => {
		let cfg = setSourceFilterTerms(base(), 'house', 'mock-01', ['quote']);
		cfg = setWebspaceFilter(cfg, 'house', ['boiler', 'van']);
		expect(cfg.webspaces.house.filter_by_source).toEqual({ 'mock-01': ['quote'] });
	});

	it('removing the source drops its entry', () => {
		let cfg = setSourceFilterTerms(base(), 'house', 'mock-01', ['quote']);
		cfg = removeSourceFromWebspace(cfg, 'house', 'mock-01');
		expect(cfg.webspaces.house.filter_by_source).toBeUndefined();
		cfg = setSourceFilterTerms(base(), 'house', 'mock-01', ['quote']);
		cfg = removeSourceInstance(cfg, 'mock-01');
		expect(cfg.webspaces.house.filter_by_source).toBeUndefined();
	});

	it('splits instance:term tokens and keeps the rest as one global term', () => {
		expect(splitFilterInput('mock-01:quote boiler van mock-01:2026', ['mock-01', 'mock-02'])).toEqual({
			global: 'boiler van',
			bySource: { 'mock-01': ['quote', '2026'] }
		});
		// unknown prefix and bare colon stay global, never dropped
		expect(splitFilterInput('ghost:x :y plain', ['mock-01'])).toEqual({
			global: 'ghost:x :y plain',
			bySource: {}
		});
	});
});

describe('setWebspaceDateRange (M3-R1, #70)', () => {
	const base = () =>
		({
			server: { listen: '' },
			paths: {},
			sources: { 'mock-01': { plugin: 'topos-plugin-mock' } },
			webspaces: {
				holiday: { keywords: ['demo'], sources: [], match: {}, filter: ['boiler'] }
			}
		}) as unknown as Parameters<typeof setWebspaceDateRange>[0];

	it('sets, clears one side, and removes both — preserving filter', () => {
		let cfg = setWebspaceDateRange(base(), 'holiday', '2026-03-01', '2026-03-31');
		expect(cfg.webspaces.holiday.date_from).toBe('2026-03-01');
		expect(cfg.webspaces.holiday.date_to).toBe('2026-03-31');
		expect(cfg.webspaces.holiday.filter).toEqual(['boiler']);
		cfg = setWebspaceDateRange(cfg, 'holiday', '2026-03-01', '');
		expect(cfg.webspaces.holiday.date_to).toBeUndefined();
		cfg = setWebspaceDateRange(cfg, 'holiday', '', '');
		expect(cfg.webspaces.holiday.date_from).toBeUndefined();
		expect(cfg.webspaces.holiday.date_to).toBeUndefined();
	});

	it('setWebspaceFilter and setSourceFilterTerms preserve the range', () => {
		let cfg = setWebspaceDateRange(base(), 'holiday', '2026-03-01', '2026-03-31');
		cfg = setWebspaceFilter(cfg, 'holiday', ['boiler', 'van']);
		expect(cfg.webspaces.holiday.date_from).toBe('2026-03-01');
		cfg = setSourceFilterTerms(cfg, 'holiday', 'mock-01', ['quote']);
		expect(cfg.webspaces.holiday.date_to).toBe('2026-03-31');
	});
});

describe('renameWebspace (M3-R2, #77)', () => {
	const base = () =>
		({
			server: { listen: '' },
			paths: {},
			sources: { 'mock-01': { plugin: 'topos-plugin-mock' } },
			webspaces: {
				old: { keywords: ['demo'], sources: [], match: {}, filter: ['boiler'] },
				other: { keywords: ['x'], sources: [], match: {} }
			}
		}) as unknown as Parameters<typeof renameWebspace>[0];

	it('carries the body byte-identical under the new key', () => {
		const cfg = renameWebspace(base(), 'old', 'new');
		expect(cfg.webspaces.new).toEqual(base().webspaces.old);
		expect(cfg.webspaces.old).toBeUndefined();
	});
	it('refuses collisions, unknown names and no-ops by returning the input', () => {
		const b = base();
		expect(renameWebspace(b, 'old', 'other')).toBe(b);
		expect(renameWebspace(b, 'ghost', 'new')).toBe(b);
		expect(renameWebspace(b, 'old', 'old')).toBe(b);
	});
});
