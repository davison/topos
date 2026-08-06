// Unit tests for fidelityAffordance (UI-08, format.ts) — the two-class
// icon/verb/title mapping that differentiates a raise-window-only link from
// a navigating one, closing the 04-UAT follow-up. Colocated vitest file
// following sources.test.ts's convention.

import { describe, it, expect } from 'vitest';
import { fidelityAffordance, formatFidelity } from '$lib/format';

describe('fidelityAffordance', () => {
	it('exact and anchored produce deeply equal, navigating affordances', () => {
		const exact = fidelityAffordance('exact', 'paperless-ngx');
		const anchored = fidelityAffordance('anchored', 'paperless-ngx');
		expect(exact).toEqual(anchored);
		expect(exact.windowOnly).toBe(false);
	});

	it('exact produces the navigating verb and matching title', () => {
		const affordance = fidelityAffordance('exact', 'paperless-ngx');
		expect(affordance.windowOnly).toBe(false);
		expect(affordance.label).toBe('Open in paperless-ngx');
		expect(affordance.title).toBe('Open in paperless-ngx');
	});

	it('anchored produces the same navigating verb and title as exact', () => {
		const affordance = fidelityAffordance('anchored', 'All Mail');
		expect(affordance.windowOnly).toBe(false);
		expect(affordance.label).toBe('Open in All Mail');
		expect(affordance.title).toBe('Open in All Mail');
	});

	it('conversation-only produces windowOnly true with a distinct verb and title', () => {
		const affordance = fidelityAffordance('conversation-only', 'Signal');
		expect(affordance.windowOnly).toBe(true);
		expect(affordance.label).toBe('Show in Signal');
		expect(affordance.label.startsWith('Show in')).toBe(true);
		expect(affordance.title).toBe(
			'Raise Signal — opens the app/conversation, not this exact message'
		);
	});

	it('an unrecognised value produces the navigating affordance rather than throwing', () => {
		expect(() => fidelityAffordance('nonsense', 'X')).not.toThrow();
		const affordance = fidelityAffordance('nonsense', 'X');
		expect(affordance.windowOnly).toBe(false);
		expect(affordance.label).toBe('Open in X');
		expect(affordance.title).toBe('Open in X');
	});

	it('an empty string value produces the navigating affordance rather than throwing', () => {
		expect(() => fidelityAffordance('', 'X')).not.toThrow();
		const affordance = fidelityAffordance('', 'X');
		expect(affordance.windowOnly).toBe(false);
		expect(affordance.label).toBe('Open in X');
	});

	it('interpolates a long, punctuation-carrying display name verbatim into both label and title', () => {
		const longName = 'Proton Mail (work@example.test) — LAN bridge';

		const navigating = fidelityAffordance('exact', longName);
		expect(navigating.label).toBe(`Open in ${longName}`);
		expect(navigating.title).toBe(`Open in ${longName}`);

		const windowOnly = fidelityAffordance('conversation-only', longName);
		expect(windowOnly.label).toBe(`Show in ${longName}`);
		expect(windowOnly.title).toBe(
			`Raise ${longName} — opens the app/conversation, not this exact message`
		);
	});

	it('does not mutate formatFidelity — all three raw enum values plus its raw fallback are unchanged', () => {
		expect(formatFidelity('exact')).toBe('exact');
		expect(formatFidelity('anchored')).toBe('anchored');
		expect(formatFidelity('conversation-only')).toBe('conversation-only');
		expect(formatFidelity('some-unrecognised-value')).toBe('some-unrecognised-value');
	});
});
