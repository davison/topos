// Static per-plugin-type connection field tables plus the comma-separated
// match-value parser (07-04-PLAN.md Task 1). Connection-field SHAPE is
// static per plugin type by deliberate design, not an oversight: Describe
// (kernel/httpapi/config.go's DescribePluginHandler, 07-02) carries match
// VOCABULARY only — no connection-field schema exists anywhere on the
// wire (07-RESEARCH.md's load-bearing finding) — so this table is the one
// honest place a plugin type's connection fields are declared. A new
// plugin type needs a new row in CONNECTION_FIELDS below.

import type { SourceConfig, ExtrasFieldDecl } from './api';

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
	],
	'topos-plugin-whatsapp': [
		DISPLAY_NAME_FIELD,
		{
			// required: true mirrors plugins/whatsapp/main.go's
			// `cfg.Path == ""` fatal guard — the same shape as Signal's own
			// path field (WEBSPACES_SOURCE_CONFIG's decoded path is fatal
			// when empty). defaultValue is seeded (not just a placeholder)
			// for the same reason Signal's is: config.example.toml's
			// [sources.whatsapp] path default
			// (~/.local/share/topos/whatsapp) is genuinely correct on a
			// standard install, and 08-UI-SPEC.md's Amendment section
			// names this exact value as the Add-Source Step 1 field's
			// default (superseding this document's own earlier
			// ~/.config/topos/whatsapp placeholder).
			key: 'path',
			label: 'Local Path',
			required: true,
			secret: false,
			advanced: false,
			placeholder: '~/.local/share/topos/whatsapp',
			defaultValue: '~/.local/share/topos/whatsapp'
		},
		SYNC_INTERVAL_FIELD
	],
	// topos-plugin-mockstrict (07.1-02-PLAN.md D-05/D-06): the browser E2E
	// harness's hermetic, cgo-free stand-in for Signal's required-field
	// flow. Three things about this row, in the order a reader needs them:
	//
	// 1. required: true is derived from plugins/mockstrict/main.go's own
	//    empty-path fatal guard exactly as the DERIVATION RULE above
	//    demands — not copied from the Signal row's shape.
	// 2. defaultValue is seeded (not merely a placeholder) specifically so
	//    the browser E2E suite can exercise the pre-fill / clear / blocked
	//    / restore loop UAT item 5 describes — a placeholder cannot
	//    express that loop because a placeholder is not a submittable
	//    value (see the ConnectionField.defaultValue doc comment above).
	// 3. This row is inert in a real installation: the topos-plugin-
	//    mockstrict binary is built only by `make e2e`, never by
	//    `make build` or `make plugins`, so GET /api/config/plugin-types
	//    never returns it outside the harness and this row is never
	//    reachable in an operator's own picker.
	'topos-plugin-mockstrict': [
		DISPLAY_NAME_FIELD,
		{
			key: 'path',
			label: 'Corpus Path',
			required: true,
			secret: false,
			advanced: false,
			placeholder: '/tmp/topos-e2e-corpus',
			defaultValue: '/tmp/topos-e2e-corpus'
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
	'topos-plugin-signal': 'Signal',
	'topos-plugin-whatsapp': 'WhatsApp'
};

// WHATSAPP_PLUGIN_BINARY is the one canonical spelling of the WhatsApp
// plugin's binary name — used wherever a component needs to branch on
// "is the selected/described plugin type WhatsApp" (AddSourceModal.svelte's
// link-step branch, RelinkModal.svelte's default plugin prop) rather than
// re-typing the literal string per call site.
export const WHATSAPP_PLUGIN_BINARY = 'topos-plugin-whatsapp';

// WHATSAPP_SOURCE_TYPE mirrors plugins/whatsapp/plugin.go's sourceType
// constant ("whatsapp") — the Describe-reported plugin KIND
// SourceStatus.source_type carries (docs/api.md's GET /api/sources).
// SourceChip.svelte's "Re-link…" menu entry (D-03) is gated on this value,
// never on WHATSAPP_PLUGIN_BINARY — source_type is the field GET
// /api/sources actually exposes.
export const WHATSAPP_SOURCE_TYPE = 'whatsapp';

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

// --- Phase 11 trust-boundary helpers (PLUG-06/08, D-06/D-07) ---

/**
 * Returns whether `pluginBinary` resolved from the external (untrusted)
 * plugin directory at launch time, per GET /api/config/plugin-types'
 * `plugin_type_tiers` lookup table (11-01-PLAN.md). A binary absent from
 * `tiers` (not yet discovered, or the table failed to load) is treated as
 * trusted — the conservative default, since an unknown binary cannot be
 * selected from the picker in the first place.
 */
export function isExternalTier(tiers: Record<string, string>, pluginBinary: string): boolean {
	return tiers[pluginBinary] === 'external';
}

/** The picker's fixed "Untrusted" label copy (11-UI-SPEC.md E3), shared by both picker groups and the untrusted-add test suite so the string exists in exactly one place. */
export const UNTRUSTED_LABEL = 'Untrusted';

// --- Phase 11 extras form helpers (PLUG-09, D-12/D-13/D-15, 11-UI-SPEC.md E6) ---

/** One row of ConnectionForm's free-form (undeclared-key) extras editor — local UI-editing state only, never itself the wire shape (rowsToExtras composes the final `Record<string, string>` SourceConfig.extras carries). */
export interface ExtrasRow {
	key: string;
	value: string;
}

/**
 * Returns the free-form editor's initial rows for a saved (or in-progress)
 * extras map: every key NOT among `declared`'s own keys, in stable
 * alphabetical order (so re-deriving after a fresh declarations array
 * arrives never reshuffles rows the operator is looking at). A saved key
 * that a plugin's CURRENT Describe response no longer declares still
 * surfaces here as a free-form row — a value can never become invisible
 * (D-15's "older kernel + newer plugin still workable" framing).
 */
export function extrasToRows(
	extras: Record<string, string> | undefined,
	declared: ExtrasFieldDecl[]
): ExtrasRow[] {
	const declaredKeys = new Set(declared.map((field) => field.key));
	return Object.entries(extras ?? {})
		.filter(([key]) => !declaredKeys.has(key))
		.sort(([a], [b]) => a.localeCompare(b))
		.map(([key, value]) => ({ key, value }));
}

/**
 * Composes the final `extras` map a save writes: every declared field's
 * OWN bound value (non-blank only — a blank non-required declared field is
 * simply omitted, matching every other optional field's tolerance for
 * being left empty) plus every free-form row whose key is non-blank once
 * trimmed. A row's raw (untrimmed) key is never written — only the
 * trimmed form. Caller is responsible for rejecting an empty/duplicate key
 * BEFORE calling this (extrasKeyError, below) — this function does not
 * itself validate, so a duplicate key here silently keeps the LAST
 * occurrence, matching plain JS object-literal semantics.
 */
export function rowsToExtras(
	declaredValues: Record<string, string>,
	rows: ExtrasRow[]
): Record<string, string> {
	const result: Record<string, string> = {};
	for (const [key, value] of Object.entries(declaredValues)) {
		if (value.trim() === '') continue;
		result[key] = value;
	}
	for (const row of rows) {
		const key = row.key.trim();
		if (key === '') continue;
		result[key] = row.value;
	}
	return result;
}

/**
 * Returns the fixed copy `Every extra field needs a unique key.` when any
 * free-form row's key is empty/whitespace-only, duplicates another
 * free-form row's key, or duplicates a declared field's key — null
 * otherwise. Checked at the same submit-time point every caller already
 * runs missingRequiredFields, never on every keystroke.
 */
export function extrasKeyError(declared: ExtrasFieldDecl[], rows: ExtrasRow[]): string | null {
	const declaredKeys = new Set(declared.map((field) => field.key.trim()));
	const seenRowKeys = new Set<string>();
	for (const row of rows) {
		const key = row.key.trim();
		if (key === '' || declaredKeys.has(key) || seenRowKeys.has(key)) {
			return 'Every extra field needs a unique key.';
		}
		seenRowKeys.add(key);
	}
	return null;
}
