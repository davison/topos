// Unit coverage for instance-id.ts's derivation and collision guard
// (07-06-PLAN.md Task 1, closing 07-REVIEW.md CR-01). House pattern
// (matches config-edit.test.ts / plugin-fields.test.ts): real fixtures,
// behavioural assertions, one consequence-describing message per
// assertion.

import { describe, it, expect } from 'vitest';
import { deriveInstanceId, resolveNewInstanceId } from './instance-id';
import type { KernelConfig } from './api';

function fixtureConfig(): KernelConfig {
	return {
		server: { listen: '127.0.0.1:7777' },
		index: { path: '/tmp/index.db' },
		plugins: { dir: '/tmp/plugins' },
		sync: { interval: '5m' },
		sources: {
			'home-email': {
				plugin: 'topos-plugin-proton',
				base_url: 'imaps://mail.example',
				token: '${HOME_EMAIL_TOKEN}',
				agent: { read: true, handoff: true },
				display_name: 'Home Email'
			}
		},
		webspaces: {
			'house-move': {
				keywords: ['boiler'],
				sources: [],
				match: {}
			}
		}
	};
}

describe('deriveInstanceId', () => {
	it('lowercases and hyphenates a simple display name', () => {
		expect(deriveInstanceId('Home Email')).toBe('home-email');
	});

	it('collapses runs of whitespace/punctuation to a single hyphen and trims edges', () => {
		expect(deriveInstanceId('  Work / Notes!!  ')).toBe('work-notes');
	});

	it('lowercases an all-caps input', () => {
		expect(deriveInstanceId('SIGNAL')).toBe('signal');
	});

	it('yields the empty string for an input of only hyphens', () => {
		expect(deriveInstanceId('---')).toBe('');
	});

	it('yields the empty string for an input of only punctuation', () => {
		expect(deriveInstanceId('!!! ///')).toBe('');
	});

	it('yields the empty string for the empty string', () => {
		expect(deriveInstanceId('')).toBe('');
	});
});

describe('resolveNewInstanceId', () => {
	it('rejects a display name that derives to an id already present in config.sources (CR-01 core scenario)', () => {
		const cfg = fixtureConfig();
		const result = resolveNewInstanceId(cfg, 'Home Email');
		expect(result).toEqual({
			ok: false,
			reason: 'collision',
			message: 'An instance id "home-email" already exists — choose a different display name.'
		});
	});

	it(
		'CR-01 regression: resolving the existing victim instance\'s OWN stored display name never returns ok — ' +
			'a caller reaching upsertSourceInstance with this id would silently clobber the victim\'s connection ' +
			'and reset its agent.read/agent.handoff grants to false (07-REVIEW.md CR-01)',
		() => {
			const cfg = fixtureConfig();
			const victimDisplayName = cfg.sources['home-email'].display_name!;
			const result = resolveNewInstanceId(cfg, victimDisplayName);
			expect(result.ok, 'expected resolving an existing instance\'s own display name to be rejected, not ok').toBe(
				false
			);
		}
	);

	it('rejects a blank display name with the exact message the Next path renders', () => {
		const cfg = fixtureConfig();
		const result = resolveNewInstanceId(cfg, '');
		expect(result).toEqual({
			ok: false,
			reason: 'blank',
			message: 'Enter a display name so this instance has an id.'
		});
	});

	it('rejects a whitespace-only display name as blank', () => {
		const cfg = fixtureConfig();
		const result = resolveNewInstanceId(cfg, '   ');
		expect(result.ok).toBe(false);
		expect((result as { reason: string }).reason).toBe('blank');
	});

	it('rejects a punctuation-only display name as blank', () => {
		const cfg = fixtureConfig();
		const result = resolveNewInstanceId(cfg, '!!!');
		expect(result.ok).toBe(false);
		expect((result as { reason: string }).reason).toBe('blank');
	});

	it('resolves a collision-free display name to its derived id', () => {
		const cfg = fixtureConfig();
		const result = resolveNewInstanceId(cfg, 'Work Email');
		expect(result).toEqual({ ok: true, id: 'work-email' });
	});

	it('never mutates the passed config document', () => {
		const cfg = fixtureConfig();
		const before = JSON.stringify(cfg);
		resolveNewInstanceId(cfg, 'Home Email');
		resolveNewInstanceId(cfg, 'Work Email');
		resolveNewInstanceId(cfg, '');
		expect(JSON.stringify(cfg)).toBe(before);
	});
});
