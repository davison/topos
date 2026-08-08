// Static per-plugin-type connection field tables plus the comma-separated
// match-value parser (07-04-PLAN.md Task 1). Connection-field SHAPE is
// static per plugin type by deliberate design, not an oversight: Describe
// (kernel/httpapi/config.go's DescribePluginHandler, 07-02) carries match
// VOCABULARY only — no connection-field schema exists anywhere on the
// wire (07-RESEARCH.md's load-bearing finding) — so this table is the one
// honest place a plugin type's connection fields are declared. A new
// plugin type needs a new row in CONNECTION_FIELDS below.

import type { SourceConfig } from './api';

export interface ConnectionField {
	// key is the SourceConfig field this input writes to.
	key: keyof SourceConfig;
	label: string;
	required: boolean;
	secret: boolean;
	advanced: boolean;
	placeholder?: string;
}

const DISPLAY_NAME_FIELD: ConnectionField = {
	key: 'display_name',
	label: 'Display Name',
	required: false,
	secret: false,
	advanced: false
};

// Every plugin type additionally gets a sync-interval override field,
// marked advanced (07-UI-SPEC.md's collapsed "Advanced options" disclosure).
const SYNC_INTERVAL_FIELD: ConnectionField = {
	key: 'sync_interval',
	label: 'Sync Interval Override',
	required: false,
	secret: false,
	advanced: true,
	placeholder: 'Use webspace default'
};

// Keyed by plugin BINARY name (kernel/pluginhost.DiscoverBinaries' own
// result shape, e.g. "topos-plugin-paperless") — the same string
// GET /api/config/plugin-types returns and SourceConfig.plugin stores.
const CONNECTION_FIELDS: Record<string, ConnectionField[]> = {
	'topos-plugin-paperless': [
		DISPLAY_NAME_FIELD,
		{ key: 'base_url', label: 'Base URL', required: true, secret: false, advanced: false },
		{ key: 'token', label: 'API Token', required: true, secret: true, advanced: false },
		{ key: 'api_version', label: 'API Version', required: false, secret: false, advanced: false },
		SYNC_INTERVAL_FIELD
	],
	'topos-plugin-silverbullet': [
		DISPLAY_NAME_FIELD,
		{ key: 'base_url', label: 'Base URL', required: true, secret: false, advanced: false },
		{ key: 'token', label: 'API Token', required: true, secret: true, advanced: false },
		{
			key: 'ca_cert',
			label: 'CA Certificate Path',
			required: false,
			secret: false,
			advanced: false
		},
		SYNC_INTERVAL_FIELD
	],
	'topos-plugin-proton': [
		DISPLAY_NAME_FIELD,
		{ key: 'base_url', label: 'Base URL', required: true, secret: false, advanced: false },
		{ key: 'username', label: 'Username', required: true, secret: false, advanced: false },
		{
			key: 'token',
			label: 'API Token',
			required: true,
			secret: true,
			advanced: false,
			placeholder: 'used as the IMAP password'
		},
		{
			key: 'webmail_base_url',
			label: 'Webmail Base URL',
			required: false,
			secret: false,
			advanced: false
		},
		SYNC_INTERVAL_FIELD
	],
	'topos-plugin-signal': [
		DISPLAY_NAME_FIELD,
		{
			key: 'path',
			label: 'Local Path',
			required: false,
			secret: false,
			advanced: false,
			placeholder: '~/.config/Signal'
		},
		SYNC_INTERVAL_FIELD
	]
};

/**
 * Returns the ordered ConnectionField descriptors for a plugin type
 * (keyed by plugin binary name). An unrecognised binary — reachable only
 * if a plugin author ships a fifth type before this table is extended —
 * degrades to the minimal Display Name + Sync Interval Override pair
 * rather than throwing or returning nothing, so Step 1 of the two-step
 * modal always has at least a name field to submit.
 */
export function connectionFieldsFor(pluginBinary: string): ConnectionField[] {
	return CONNECTION_FIELDS[pluginBinary] ?? [DISPLAY_NAME_FIELD, SYNC_INTERVAL_FIELD];
}

// Display labels for the "New {plugin type}…" picker row and the Step 1
// modal title (07-UI-SPEC.md's Step 1 table headings) — cosmetic only,
// never used as an identity key (the plugin BINARY name is that key
// everywhere else).
const PLUGIN_TYPE_LABELS: Record<string, string> = {
	'topos-plugin-paperless': 'paperless-ngx',
	'topos-plugin-silverbullet': 'SilverBullet',
	'topos-plugin-proton': 'Proton',
	'topos-plugin-signal': 'Signal'
};

/** Human display label for a plugin binary name, falling back to a title-cased strip of the topos-plugin- prefix for an unrecognised binary. */
export function pluginTypeLabel(pluginBinary: string): string {
	return PLUGIN_TYPE_LABELS[pluginBinary] ?? titleCaseField(pluginBinary.replace(/^topos-plugin-/, ''));
}

/**
 * Turns a vocabulary field name into its label: underscores to spaces,
 * each word capitalised — `conversation_names` becomes
 * `Conversation Names`. Also reused for `pluginTypeLabel`'s fallback
 * branch above (hyphens split the same way as underscores there since
 * `replace` only touches `_` — see that function's own split logic).
 */
export function titleCaseField(name: string): string {
	return name
		.split(/[_-]/)
		.filter((word) => word.length > 0)
		.map((word) => word.charAt(0).toUpperCase() + word.slice(1))
		.join(' ');
}

/**
 * Splits a comma-separated match-field input into a trimmed, non-empty
 * string array — the config's `[]string` match-value shape. An all-blank
 * input (only commas/whitespace) yields an empty array, which callers
 * treat as "omit this field" rather than "match an empty value list" (the
 * kernel's own load-time validation rejects a zero-value field outright).
 */
export function parseMatchValues(input: string): string[] {
	return input
		.split(',')
		.map((value) => value.trim())
		.filter((value) => value.length > 0);
}
