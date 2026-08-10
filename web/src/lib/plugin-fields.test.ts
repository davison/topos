// Unit coverage for plugin-fields.ts's three pure helpers (07-04-PLAN.md
// Task 1): titleCaseField's word-casing, parseMatchValues' blank/all-blank/
// whitespace-around-comma handling, and connectionFieldsFor's known- and
// unknown-plugin-type cases.
//
// 07-13-PLAN.md Task 1 extends this file with the derived-required-flags
// coverage that closed 07-UAT.md G-07-5: the required flags in
// CONNECTION_FIELDS below are NOT chosen by inspection of this table —
// they are derived from each plugin's own pre-`goplugin.Serve` fatal
// guard, read directly from plugins/paperless/main.go,
// plugins/silverbullet/main.go, plugins/proton/main.go and
// plugins/signal/main.go. A future plugin type's required set must be
// re-derived the same way, not copied from a sibling row.

import { describe, it, expect } from 'vitest';
import {
	connectionFieldsFor,
	defaultConnectionValues,
	missingRequiredFields,
	missingRequiredFieldsMessage,
	parseMatchValues,
	pluginTypeLabel,
	titleCaseField
} from './plugin-fields';

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

	// 08-04-PLAN.md Task 1: the WhatsApp row, same shape as Signal's own
	// local-path-only row.
	it('gives WhatsApp a required, pre-seeded local path and no base_url/token fields', () => {
		const fields = connectionFieldsFor('topos-plugin-whatsapp');
		expect(fields.map((f) => f.key)).toEqual(['display_name', 'path', 'sync_interval']);
		const pathField = fields.find((f) => f.key === 'path');
		expect(pathField?.required).toBe(true);
		expect(pathField?.placeholder).toBe('~/.local/share/topos/whatsapp');
		expect(pathField?.defaultValue).toBe('~/.local/share/topos/whatsapp');
	});

	it('falls back to a minimal field set for an unknown plugin type', () => {
		const fields = connectionFieldsFor('topos-plugin-does-not-exist');
		expect(fields.map((f) => f.key)).toEqual(['display_name', 'sync_interval']);
	});

	// 07.1-02-PLAN.md Task 2: the browser E2E harness's hermetic
	// stand-in for Signal's required-field flow (D-05/D-06).
	it('returns exactly three descriptors in order for topos-plugin-mockstrict: display name, a required path field, sync interval', () => {
		const fields = connectionFieldsFor('topos-plugin-mockstrict');
		expect(fields.map((f) => f.key)).toEqual(['display_name', 'path', 'sync_interval']);
		const pathField = fields.find((f) => f.key === 'path');
		expect(pathField?.required).toBe(true);
		expect(pathField?.secret).toBe(false);
		expect(pathField?.advanced).toBe(false);
	});
});

// Table-truth tests (07-13-PLAN.md Task 1): each plugin's required set is
// asserted against a hand-derived list read directly from that plugin's
// own pre-Serve fatal guards, never against a copy of CONNECTION_FIELDS
// itself — a table that agreed with itself would prove nothing. A failure
// here means CONNECTION_FIELDS has drifted from the plugin file named in
// the failure message.
describe('table truth: required flags match each plugin binary\'s own pre-Serve fatal guards', () => {
	function requiredKeys(pluginBinary: string): string[] {
		return connectionFieldsFor(pluginBinary)
			.filter((f) => f.required)
			.map((f) => f.key);
	}

	it('paperless requires exactly base_url and token (plugins/paperless/main.go fatals on both, defaults api_version instead of fatalling)', () => {
		expect(requiredKeys('topos-plugin-paperless')).toEqual(['base_url', 'token']);
	});

	it('silverbullet requires exactly base_url and token (plugins/silverbullet/main.go fatals on both; ca_cert has no guard)', () => {
		expect(requiredKeys('topos-plugin-silverbullet')).toEqual(['base_url', 'token']);
	});

	it('proton requires exactly base_url, username, token and webmail_base_url (plugins/proton/main.go fatals on all four; ca_cert has no guard)', () => {
		expect(requiredKeys('topos-plugin-proton')).toEqual([
			'base_url',
			'username',
			'token',
			'webmail_base_url'
		]);
	});

	it('signal requires exactly path (plugins/signal/main.go fatals on cfg.Path == "")', () => {
		expect(requiredKeys('topos-plugin-signal')).toEqual(['path']);
	});

	// 08-04-PLAN.md Task 1: mirrors plugins/whatsapp/main.go's
	// `cfg.Path == ""` fatal guard, read directly from that file (see
	// plugin-fields.ts's own DERIVATION RULE comment on the whatsapp row).
	it('whatsapp requires exactly path (plugins/whatsapp/main.go fatals on cfg.Path == "")', () => {
		expect(requiredKeys('topos-plugin-whatsapp')).toEqual(['path']);
	});
});

describe('exactly two fields in the whole table declare a default value (Signal and WhatsApp)', () => {
	it('walks every plugin binary rather than checking Signal alone, so a later default added without a recorded rationale fails this suite', () => {
		const allPluginBinaries = [
			'topos-plugin-paperless',
			'topos-plugin-silverbullet',
			'topos-plugin-proton',
			'topos-plugin-signal',
			'topos-plugin-whatsapp'
		];
		const fieldsWithDefaults = allPluginBinaries.flatMap((binary) =>
			connectionFieldsFor(binary).filter((f) => f.defaultValue !== undefined)
		);
		expect(fieldsWithDefaults.length).toBe(2);
		for (const field of fieldsWithDefaults) {
			expect(field.key).toBe('path');
		}
		expect(fieldsWithDefaults.map((f) => f.defaultValue).sort()).toEqual(
			['~/.config/Signal', '~/.local/share/topos/whatsapp'].sort()
		);
	});
});

describe('defaultConnectionValues', () => {
	it('returns the plugin binary plus every field declaring a default value, for Signal', () => {
		expect(defaultConnectionValues('topos-plugin-signal')).toEqual({
			plugin: 'topos-plugin-signal',
			path: '~/.config/Signal'
		});
	});

	it('returns just the plugin key for a plugin type with no defaults', () => {
		expect(defaultConnectionValues('topos-plugin-paperless')).toEqual({
			plugin: 'topos-plugin-paperless'
		});
		expect(defaultConnectionValues('topos-plugin-proton')).toEqual({
			plugin: 'topos-plugin-proton'
		});
	});

	it('returns the plugin binary plus the seeded corpus path, for topos-plugin-mockstrict', () => {
		expect(defaultConnectionValues('topos-plugin-mockstrict')).toEqual({
			plugin: 'topos-plugin-mockstrict',
			path: '/tmp/topos-e2e-corpus'
		});
	});

	it('returns the plugin binary plus the seeded local path, for topos-plugin-whatsapp', () => {
		expect(defaultConnectionValues('topos-plugin-whatsapp')).toEqual({
			plugin: 'topos-plugin-whatsapp',
			path: '~/.local/share/topos/whatsapp'
		});
	});
});

describe('missingRequiredFields', () => {
	it('returns descriptors for every required field whose value is absent, empty or whitespace-only, in table order', () => {
		const missing = missingRequiredFields('topos-plugin-paperless', { plugin: 'topos-plugin-paperless', agent: { read: false, handoff: false } });
		expect(missing.map((f) => f.key)).toEqual(['base_url', 'token']);
	});

	it('reports a whitespace-only value as missing', () => {
		const missing = missingRequiredFields('topos-plugin-paperless', {
			plugin: 'topos-plugin-paperless',
			base_url: '   ',
			token: 'real-token',
			agent: { read: false, handoff: false }
		});
		expect(missing.map((f) => f.key)).toEqual(['base_url']);
	});

	it('returns an empty list when every required field is filled', () => {
		const missing = missingRequiredFields('topos-plugin-signal', {
			plugin: 'topos-plugin-signal',
			path: '~/.config/Signal',
			agent: { read: false, handoff: false }
		});
		expect(missing).toEqual([]);
	});

	it('ignores optional fields entirely, even when they are blank', () => {
		const missing = missingRequiredFields('topos-plugin-paperless', {
			plugin: 'topos-plugin-paperless',
			base_url: 'https://paperless.example',
			token: 'real-token',
			api_version: '',
			agent: { read: false, handoff: false }
		});
		expect(missing).toEqual([]);
	});

	it('returns an empty list for a plugin binary with no table row (the degrade set contains no required field)', () => {
		const missing = missingRequiredFields('topos-plugin-does-not-exist', {
			plugin: 'topos-plugin-does-not-exist',
			agent: { read: false, handoff: false }
		});
		expect(missing).toEqual([]);
	});

	it('reports a secret field holding a variable name for a variable that is NOT set as present, not missing — D-15 requires an unset variable to still save with a warning', () => {
		const missing = missingRequiredFields('topos-plugin-paperless', {
			plugin: 'topos-plugin-paperless',
			base_url: 'https://paperless.example',
			token: '${SOME_VAR_NOT_ACTUALLY_SET}',
			agent: { read: false, handoff: false }
		});
		expect(missing).toEqual([]);
	});

	it('returns one descriptor labelled Corpus Path for topos-plugin-mockstrict when path is blank', () => {
		const missing = missingRequiredFields('topos-plugin-mockstrict', {
			plugin: 'topos-plugin-mockstrict',
			path: '',
			agent: { read: false, handoff: false }
		});
		expect(missing.map((f) => f.label)).toEqual(['Corpus Path']);
	});

	it('returns an empty list for topos-plugin-mockstrict when path holds any non-whitespace string', () => {
		const missing = missingRequiredFields('topos-plugin-mockstrict', {
			plugin: 'topos-plugin-mockstrict',
			path: '/tmp/topos-e2e-corpus',
			agent: { read: false, handoff: false }
		});
		expect(missing).toEqual([]);
	});
});

describe('missingRequiredFieldsMessage', () => {
	it('names a single missing field by its label', () => {
		const fields = connectionFieldsFor('topos-plugin-signal').filter((f) => f.key === 'path');
		expect(missingRequiredFieldsMessage(fields)).toBe('Fill in Local Path before continuing.');
	});

	it('names several missing fields by their labels', () => {
		const fields = connectionFieldsFor('topos-plugin-paperless').filter(
			(f) => f.key === 'base_url' || f.key === 'token'
		);
		expect(missingRequiredFieldsMessage(fields)).toBe('Fill in Base URL and API Token before continuing.');
	});

	it('renders the topos-plugin-mockstrict path field as Corpus Path', () => {
		const fields = connectionFieldsFor('topos-plugin-mockstrict').filter((f) => f.key === 'path');
		expect(missingRequiredFieldsMessage(fields)).toBe('Fill in Corpus Path before continuing.');
	});
});

describe('pluginTypeLabel', () => {
	it('falls back to a title-cased prefix strip for topos-plugin-mockstrict — no PLUGIN_TYPE_LABELS entry is added for it', () => {
		expect(pluginTypeLabel('topos-plugin-mockstrict')).toBe('Mockstrict');
	});

	it('resolves topos-plugin-whatsapp to "WhatsApp"', () => {
		expect(pluginTypeLabel('topos-plugin-whatsapp')).toBe('WhatsApp');
	});
});
