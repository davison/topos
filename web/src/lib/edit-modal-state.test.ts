// Unit coverage for edit-modal-state.ts's seeding helpers (07-08-PLAN.md
// Task 2, closing 07-REVIEW.md CR-02). House pattern (matches
// instance-id.test.ts / config-edit.test.ts): real fixtures, behavioural
// assertions, one consequence-describing message per assertion, and a
// named regression case tied to the review finding.

import { describe, it, expect } from 'vitest';
import { seedConnectionValues, seedMatchBlock } from './edit-modal-state';
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
				match: { 'home-email': { labels: ['house-move', 'quotes'] } }
			},
			'no-match-map': {
				keywords: [],
				sources: []
				// match deliberately omitted — the "webspace with no match map
				// at all" fixture case.
			} as unknown as KernelConfig['webspaces'][string]
		}
	};
}

describe('seedConnectionValues', () => {
	it('returns a known instance\'s stored connection values, including both agent grants and the verbatim token reference', () => {
		const cfg = fixtureConfig();
		const result = seedConnectionValues(cfg, 'home-email');
		expect(result).toEqual({
			plugin: 'topos-plugin-proton',
			base_url: 'imaps://mail.example',
			token: '${HOME_EMAIL_TOKEN}',
			agent: { read: true, handoff: true },
			display_name: 'Home Email'
		});
	});

	it('returns an empty-plugin default with both agent grants false for an instance absent from config', () => {
		const cfg = fixtureConfig();
		const result = seedConnectionValues(cfg, 'does-not-exist');
		expect(result).toEqual({ plugin: '', agent: { read: false, handoff: false } });
	});

	it('does not mutate the passed config', () => {
		const cfg = fixtureConfig();
		const before = JSON.stringify(cfg);
		seedConnectionValues(cfg, 'home-email');
		seedConnectionValues(cfg, 'does-not-exist');
		expect(JSON.stringify(cfg)).toBe(before);
	});
});

describe('seedMatchBlock', () => {
	it('returns the stored match block for an instance with a match entry', () => {
		const cfg = fixtureConfig();
		const result = seedMatchBlock(cfg, 'house-move', 'home-email');
		expect(result).toEqual({ labels: ['house-move', 'quotes'] });
	});

	it('returns an empty block for an instance with no entry in an existing match map', () => {
		const cfg = fixtureConfig();
		const result = seedMatchBlock(cfg, 'house-move', 'some-other-instance');
		expect(result).toEqual({});
	});

	it('returns an empty block for a webspace with no match map at all', () => {
		const cfg = fixtureConfig();
		const result = seedMatchBlock(cfg, 'no-match-map', 'home-email');
		expect(result).toEqual({});
	});

	it('returns an empty block for a webspace absent from config entirely', () => {
		const cfg = fixtureConfig();
		const result = seedMatchBlock(cfg, 'does-not-exist-webspace', 'home-email');
		expect(result).toEqual({});
	});

	it('does not mutate the passed config', () => {
		const cfg = fixtureConfig();
		const before = JSON.stringify(cfg);
		seedMatchBlock(cfg, 'house-move', 'home-email');
		seedMatchBlock(cfg, 'no-match-map', 'home-email');
		seedMatchBlock(cfg, 'does-not-exist-webspace', 'home-email');
		expect(JSON.stringify(cfg)).toBe(before);
	});
});

describe('CR-02 regression: a re-seed has no memory of a discarded session', () => {
	it('seedConnectionValues: mutating a first result and re-seeding from the same unchanged config yields a value matching the pre-mutation snapshot', () => {
		const cfg = fixtureConfig();
		const first = seedConnectionValues(cfg, 'home-email');
		const snapshotBeforeMutation = JSON.parse(JSON.stringify(first));

		// Mutate the returned value the way an in-progress edit session
		// does — the exact kind of typing a user then Cancels.
		first.base_url = 'https://wrong-typed-and-discarded.example';
		first.display_name = 'Discarded Draft Name';
		first.agent.read = false;

		const second = seedConnectionValues(cfg, 'home-email');
		expect(
			second,
			'CR-02: a reopened edit modal re-seeds from config and therefore cannot show, or save, a value from a session the user abandoned'
		).toEqual(snapshotBeforeMutation);
	});

	it('seedMatchBlock: mutating a first result (including a term added to a match key) and re-seeding yields a value matching the pre-mutation snapshot', () => {
		const cfg = fixtureConfig();
		const first = seedMatchBlock(cfg, 'house-move', 'home-email');
		const snapshotBeforeMutation = JSON.parse(JSON.stringify(first));

		// Mutate the returned value the way an in-progress match-settings
		// edit session does — adding a term the user then Cancels.
		first.labels.push('discarded-term');
		first.newKey = ['also-discarded'];

		const second = seedMatchBlock(cfg, 'house-move', 'home-email');
		expect(
			second,
			'CR-02: a reopened match-settings modal re-seeds from config and therefore cannot show, or save, a discarded match term'
		).toEqual(snapshotBeforeMutation);
	});
});
