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
	// required mirrors that plugin's own pre-goplugin.Serve fatal guard —
	// see the CONNECTION_FIELDS comment below for the derivation rule and
	// the four source files it is read from. A field is required here if
	// and only if the plugin itself exits fatally when it is empty; a
	// field the plugin merely defaults (paperless's api_version) or never
	// checks (ca_cert everywhere) stays optional.
	required: boolean;
	secret: boolean;
	advanced: boolean;
	placeholder?: string;
	// defaultValue is a value that is genuinely correct on a STANDARD
	// install, seeded into the form as a real, user-editable value — never
	// merely display-only grey text the way `placeholder` is. This is
	// distinct from placeholder specifically because a placeholder cannot
	// be submitted (07-13-PLAN.md's G-07-5 diagnosis: this exact gap made
	// leaving Signal's mandatory path untouched the natural operator
	// action). A field whose correct value is INSTALLATION-SPECIFIC must
	// declare a placeholder and no default — Proton's webmail_base_url is
	// the worked example: its account index varies per user, so a wrong
	// seeded default would silently produce broken deep links, a worse
	// failure than a visible empty required field now that empty required
	// fields are actually visible (the missingRequiredFields guard below).
	defaultValue?: string;
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
//
// DERIVATION RULE (07-13-PLAN.md Task 1, closing 07-UAT.md G-07-5): every
// `required: true` below is copied from that plugin's own pre-
// `goplugin.Serve` fatal guard — the four files that are the one true
// source of what each plugin cannot start without:
//   plugins/paperless/main.go    (base_url, token fatal; api_version defaults)
//   plugins/silverbullet/main.go (base_url, token fatal; ca_cert has no guard)
//   plugins/proton/main.go       (base_url, username, token, webmail_base_url fatal; ca_cert has no guard)
//   plugins/signal/main.go       (path fatal — the field this gap was filed against)
// A new plugin type needs its own row derived the same way, by reading
// its main.go guards — never by copying a sibling row's shape.
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
			// required: true mirrors plugins/proton/main.go's
			// `cfg.WebmailBaseURL == ""` fatal guard (line ~56) — the
			// SAME optional-in-UI/mandatory-in-plugin mismatch G-07-5
			// found in Signal's path, confirmed by controlled experiment
			// in .planning/debug/signal-trial-launch-handshake.md
			// (12:08:30 evidence entry). NO defaultValue: the webmail
			// account index varies per user, so seeding a value here
			// would silently produce broken deep links rather than the
			// visible empty-required-field message this plan adds.
			key: 'webmail_base_url',
			label: 'Webmail Base URL',
			required: true,
			secret: false,
			advanced: false
		},
		SYNC_INTERVAL_FIELD
	],
	'topos-plugin-signal': [
		DISPLAY_NAME_FIELD,
		{
			// required: true mirrors plugins/signal/main.go's
			// `cfg.Path == ""` fatal guard (line ~47) — the exact defect
			// G-07-5 diagnosed. defaultValue is seeded (not just a
			// placeholder) because ~/.config/Signal is genuinely correct
			// on a standard Linux Signal Desktop install — see the
			// ConnectionField.defaultValue doc comment for why this is
			// the one field in the whole table allowed one.
			key: 'path',
			label: 'Local Path',
			required: true,
			secret: false,
			advanced: false,
			placeholder: '~/.config/Signal',
			defaultValue: '~/.config/Signal'
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

/**
 * Builds the initial connectionValues for a freshly-picked plugin type:
 * the plugin binary under the `plugin` key, plus every field that
 * declares a `defaultValue` — a real, editable seeded value, never mere
 * placeholder text. For a plugin type with no defaults (every type except
 * Signal today) this returns just `{ plugin: pluginBinary }`. Callers
 * (AddSourceModal's plugin-type selection) are responsible for adding the
 * `agent` grants shape on top, since that has nothing to do with this
 * table.
 */
export function defaultConnectionValues(pluginBinary: string): Partial<SourceConfig> {
	const values: Partial<SourceConfig> = { plugin: pluginBinary };
	for (const field of connectionFieldsFor(pluginBinary)) {
		if (field.defaultValue !== undefined) {
			(values as Record<string, string>)[field.key] = field.defaultValue;
		}
	}
	return values;
}

/**
 * Returns, in table order, the descriptor for every required field whose
 * current value in `values` is missing, empty or whitespace-only. A value
 * of `undefined` (the field was never set) counts as missing. When the
 * value IS present but is not a string (only `agent` today, never a
 * `required` field) the descriptor names something that is not a text
 * field, and it is not this helper's job to judge it — treated as
 * present.
 *
 * A secret field is satisfied by the presence of an environment variable
 * NAME alone, never by that variable actually being set (D-15): this
 * helper only ever reads `values[field.key]`, the stored `${VAR}`
 * reference string, and never consults the envVars presence map. An
 * unset-but-named variable must still save with SecretField's own
 * warning badge, not be blocked here as a second, stricter gate.
 *
 * A plugin binary with no CONNECTION_FIELDS row degrades (via
 * connectionFieldsFor) to Display Name + Sync Interval Override, neither
 * of which is required — so this always returns an empty list for an
 * unknown plugin type, never blocking a plugin type the table knows
 * nothing about.
 */
export function missingRequiredFields(
	pluginBinary: string,
	values: SourceConfig
): ConnectionField[] {
	return connectionFieldsFor(pluginBinary).filter((field) => {
		if (!field.required) return false;
		const raw = values[field.key];
		if (raw === undefined) return true;
		if (typeof raw !== 'string') return false;
		return raw.trim() === '';
	});
}

/**
 * Renders missingRequiredFields' result as one instruction sentence
 * naming the missing fields by their human labels — the single exported
 * function every submit site (Connect step, Save anyway, Edit
 * connection…) calls, so all three produce identical copy.
 */
export function missingRequiredFieldsMessage(fields: ConnectionField[]): string {
	const labels = fields.map((f) => f.label);
	const joined =
		labels.length <= 1
			? labels.join('')
			: `${labels.slice(0, -1).join(', ')} and ${labels[labels.length - 1]}`;
	return `Fill in ${joined} before continuing.`;
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
