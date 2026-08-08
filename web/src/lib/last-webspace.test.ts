// Unit coverage for last-webspace.ts (07-03-PLAN.md Task 3): the pure
// "remembered, else first, else none" redirect-target resolver, plus the
// storage-throwing case that turns readLastWebspace's own try/catch into
// the `null` the resolver treats as "nothing remembered."
//
// vitest's web/vite.config.ts test block runs environment: 'node' — no
// browser `localStorage` global exists by default here, which already
// exercises readLastWebspace/writeLastWebspace's "inert outside a browser"
// guard for free. The throwing-storage case below installs a minimal fake
// `localStorage` on `globalThis` for the duration of one test, then
// restores the prior value, so this file's tests never leak state to any
// other test file sharing the same process.

import { describe, it, expect, afterEach } from 'vitest';
import {
	LAST_WEBSPACE_KEY,
	readLastWebspace,
	writeLastWebspace,
	resolveRedirectTarget
} from './last-webspace';

describe('resolveRedirectTarget: remembered, else first, else none', () => {
	it('a remembered name still present in webspaces wins', () => {
		expect(resolveRedirectTarget(['alpha', 'beta', 'gamma'], 'beta')).toBe('beta');
	});

	it('a remembered name the kernel no longer reports falls through to the first', () => {
		expect(resolveRedirectTarget(['alpha', 'beta'], 'stale-removed-webspace')).toBe('alpha');
	});

	it('no remembered name (null) falls through to the first', () => {
		expect(resolveRedirectTarget(['alpha', 'beta'], null)).toBe('alpha');
	});

	it('an empty webspace list yields no target regardless of what was remembered', () => {
		expect(resolveRedirectTarget([], 'anything')).toBeNull();
		expect(resolveRedirectTarget([], null)).toBeNull();
	});
});

describe('readLastWebspace / writeLastWebspace: inert outside a browser', () => {
	it('readLastWebspace returns null when localStorage is unavailable', () => {
		// No localStorage global exists in this vitest environment (node) —
		// this exercises the exact "outside a browser" branch.
		expect(readLastWebspace()).toBeNull();
	});

	it('writeLastWebspace does not throw when localStorage is unavailable', () => {
		expect(() => writeLastWebspace('does-not-matter')).not.toThrow();
	});
});

describe('a storage read that throws yields null (and therefore the first via the resolver)', () => {
	const previousDescriptor = Object.getOwnPropertyDescriptor(globalThis, 'localStorage');

	afterEach(() => {
		if (previousDescriptor) {
			Object.defineProperty(globalThis, 'localStorage', previousDescriptor);
		} else {
			// @ts-expect-error -- deleting a test-only shim, restoring the
			// no-localStorage-global baseline this file's other tests rely on.
			delete globalThis.localStorage;
		}
	});

	it('readLastWebspace catches a throwing getItem and returns null, never propagating', () => {
		Object.defineProperty(globalThis, 'localStorage', {
			configurable: true,
			value: {
				getItem() {
					throw new Error('storage disabled (private browsing / quota)');
				},
				setItem() {
					throw new Error('storage disabled (private browsing / quota)');
				}
			}
		});

		expect(() => readLastWebspace()).not.toThrow();
		expect(readLastWebspace()).toBeNull();
		// The resolver treats that null exactly like "nothing remembered" —
		// the end-to-end behaviour a throwing storage read must produce.
		expect(resolveRedirectTarget(['alpha', 'beta'], readLastWebspace())).toBe('alpha');
	});

	it('writeLastWebspace catches a throwing setItem without propagating', () => {
		Object.defineProperty(globalThis, 'localStorage', {
			configurable: true,
			value: {
				getItem() {
					return null;
				},
				setItem() {
					throw new Error('storage disabled (private browsing / quota)');
				}
			}
		});

		expect(() => writeLastWebspace('house-move')).not.toThrow();
	});
});

describe('LAST_WEBSPACE_KEY', () => {
	it('is a stable, non-empty string constant', () => {
		expect(typeof LAST_WEBSPACE_KEY).toBe('string');
		expect(LAST_WEBSPACE_KEY.length).toBeGreaterThan(0);
	});
});
