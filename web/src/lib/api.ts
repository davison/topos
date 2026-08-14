// The single typed client for the kernel's JSON contract
// (kernel/httpapi/*.go). Every fetch here is a relative /api/... path —
// nothing in the SPA calls a source (e.g. paperless-ngx) directly; every
// source request goes through the kernel.

export interface SyncStatus {
	status: string;
	finished_unix: number;
	error: string;
}

export interface Webspace {
	name: string;
	keywords: string[];
	item_count: number;
	last_sync: SyncStatus;
}

export interface WebspacesResponse {
	schema_version: number;
	webspaces: Webspace[];
}

export interface Link {
	url: string;
	fidelity: string;
}

export interface Provenance {
	source_type?: string;
	source_system?: string;
	source_id?: string;
	plugin?: string;
	contract_version?: string;
	synced_at_unix?: string;
	[key: string]: string | undefined;
}

export interface StreamItem {
	id: string;
	// source is the source INSTANCE id ([sources.<id>] config key, D-08) —
	// the identity key for filtering, staleness and grant lookups. Two
	// items sharing one source_type (plugin kind) always have distinct
	// source values when synced through two configured instances.
	source: string;
	source_type: string;
	// source_display_name is the resolved, operator-authored label for
	// this item's source instance (D-09): its configured display_name, or
	// the instance id itself when omitted — the kernel never emits an
	// empty source_display_name.
	source_display_name: string;
	source_id: string;
	title: string;
	preview: string;
	timestamp_unix: number;
	secondary_timestamp_unix: number;
	labels: string[];
	group_id: string;
	group_label: string;
	link: Link;
	thumbnail_url?: string;
	provenance: Provenance;
}

export interface StreamResponse {
	schema_version: number;
	webspace: string;
	sync: SyncStatus;
	items: StreamItem[];
	// excluded_count (13-03-PLAN.md Task 2, KERN-10) is the webspace's LIVE
	// total excluded-item count, populated on EVERY stream request in
	// BOTH views (kernel/httpapi/stream.go's streamResponse.ExcludedCount)
	// — this is the one field the excluded-items view toggle (E4) reads;
	// no separate count-only round trip.
	excluded_count: number;
}

// SearchResult extends StreamItem so it is directly usable anywhere a
// StreamItem is expected — this is what lets StreamRow render a search
// result unchanged, with `snippet` swapped in for the preview region.
export interface SearchResult extends StreamItem {
	snippet: string;
}

export interface SearchResponse {
	schema_version: number;
	webspace: string;
	query: string;
	results: SearchResult[];
}

// SNIPPET_OPEN/SNIPPET_CLOSE mirror kernel/index/store.go's SnippetOpen/
// SnippetClose constants exactly (03-03-SUMMARY.md) — the ASCII STX/ETX
// control characters the kernel wraps a matched term between in a
// search result's `snippet` field. These characters cannot occur in real
// subject lines or preview text, so a consumer can split on them safely.
export const SNIPPET_OPEN = '\u0002';
export const SNIPPET_CLOSE = '\u0003';

export interface Rendition {
	mime_type: string;
	size_bytes: number;
	url: string;
}

export interface ItemContent {
	available: boolean;
	unavailable_reason: string;
	text: string;
	rendition: Rendition | null;
}

export interface ItemDetail {
	schema_version: number;
	item: StreamItem;
	content: ItemContent;
}

export interface ApiErrorEnvelope {
	schema_version: number;
	error: {
		code: string;
		message: string;
	};
}

export class ApiError extends Error {
	code: string;

	constructor(code: string, message: string) {
		super(message);
		this.name = 'ApiError';
		this.code = code;
	}
}

// CONFIG_CONFLICT_MESSAGE (07-05-PLAN.md Task 2, D-03) is the ONE fixed
// copy every config-writing surface in the app shows on a
// config_changed_on_disk rejection — this is the single place that
// literal string is spelled out; every caller imports this constant
// rather than re-typing the string, so save-state.test.ts's
// exactly-one-occurrence guard is a fact about the source tree, not a
// convention every modal has to remember to follow.
export const CONFIG_CONFLICT_MESSAGE = 'Config changed on disk — review and retry.';

async function getJSON<T>(path: string): Promise<T> {
	const res = await fetch(path);
	if (!res.ok) {
		let envelope: ApiErrorEnvelope | undefined;
		try {
			envelope = (await res.json()) as ApiErrorEnvelope;
		} catch {
			// response body wasn't the error envelope (e.g. the kernel is
			// entirely unreachable) — fall through to a generic error below.
		}
		if (envelope?.error) {
			throw new ApiError(envelope.error.code, envelope.error.message);
		}
		throw new ApiError('http_error', `Request to ${path} failed with status ${res.status}`);
	}
	return (await res.json()) as T;
}

/**
 * postJSON mirrors getJSON's error-envelope handling exactly (same
 * ApiError construction, same fallback for a non-envelope body) for the
 * kernel's POST routes. body is optional — most POST routes here (refresh,
 * sync) carry no request body — but when present it is JSON-stringified
 * with a Content-Type header, the shape PUT /api/config also needs
 * (07-01-PLAN.md Task 1).
 */
async function postJSON<T>(path: string, body?: unknown): Promise<T> {
	const init: RequestInit = { method: 'POST' };
	if (body !== undefined) {
		init.headers = { 'Content-Type': 'application/json' };
		init.body = JSON.stringify(body);
	}
	const res = await fetch(path, init);
	if (!res.ok) {
		let envelope: ApiErrorEnvelope | undefined;
		try {
			envelope = (await res.json()) as ApiErrorEnvelope;
		} catch {
			// response body wasn't the error envelope (e.g. the kernel is
			// entirely unreachable) — fall through to a generic error below.
		}
		if (envelope?.error) {
			throw new ApiError(envelope.error.code, envelope.error.message);
		}
		throw new ApiError('http_error', `Request to ${path} failed with status ${res.status}`);
	}
	return (await res.json()) as T;
}

/**
 * putJSON mirrors getJSON/postJSON's error-envelope handling exactly, for
 * the kernel's one PUT route (PUT /api/config, the kernel's first mutating
 * HTTP surface, 07-01-PLAN.md Task 1). Unlike postJSON's optional body,
 * every PUT caller has one — there is currently exactly one PUT route and
 * it always carries a request document.
 */
async function putJSON<T>(path: string, body: unknown): Promise<T> {
	const res = await fetch(path, {
		method: 'PUT',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(body)
	});
	if (!res.ok) {
		let envelope: ApiErrorEnvelope | undefined;
		try {
			envelope = (await res.json()) as ApiErrorEnvelope;
		} catch {
			// response body wasn't the error envelope (e.g. the kernel is
			// entirely unreachable) — fall through to a generic error below.
		}
		if (envelope?.error) {
			throw new ApiError(envelope.error.code, envelope.error.message);
		}
		throw new ApiError('http_error', `Request to ${path} failed with status ${res.status}`);
	}
	return (await res.json()) as T;
}

/** GET /api/webspaces */
export function listWebspaces(): Promise<WebspacesResponse> {
	return getJSON<WebspacesResponse>('/api/webspaces');
}

/**
 * GET /api/webspaces/{webspace}/stream, optionally widened to the
 * excluded bucket (13-03-PLAN.md Task 2, KERN-10). `view` defaults to
 * omitted — a call with no second argument (every pre-Phase-13 call site)
 * produces a byte-identical request to before this parameter existed;
 * only `view === 'excluded'` appends `?view=excluded`. The kernel itself
 * treats an omitted/`'included'` value identically (docs/api.md), so
 * `'included'` is never appended even when passed explicitly.
 */
export function getStream(
	webspace: string,
	view?: 'included' | 'excluded'
): Promise<StreamResponse> {
	const path = `/api/webspaces/${encodeURIComponent(webspace)}/stream`;
	return getJSON<StreamResponse>(view === 'excluded' ? `${path}?view=excluded` : path);
}

/** GET /api/webspaces/{webspace}/search?q= */
export function searchWebspace(webspace: string, query: string): Promise<SearchResponse> {
	return getJSON<SearchResponse>(
		`/api/webspaces/${encodeURIComponent(webspace)}/search?q=${encodeURIComponent(query)}`
	);
}

/** GET /api/items/{id} */
export function getItem(id: string): Promise<ItemDetail> {
	return getJSON<ItemDetail>(`/api/items/${encodeURIComponent(id)}`);
}

/**
 * Relative kernel path for GET /api/items/{id}/content. When query is a
 * non-empty (trimmed) search string, appends it as an encoded `hl` query
 * parameter — the channel the kernel-side highlighter (UI-09,
 * kernel/httpapi/rendition.go's highlightTerms) reads to derive the
 * highlight term set. An empty/whitespace-only or omitted query returns
 * today's path unchanged, byte-identical to the pre-UI-09 URL.
 */
export function contentUrl(id: string, query?: string): string {
	const base = `/api/items/${encodeURIComponent(id)}/content`;
	const trimmed = query?.trim();
	if (!trimmed) return base;
	return `${base}?hl=${encodeURIComponent(trimmed)}`;
}

/** Relative kernel path for GET /api/items/{id}/thumbnail. */
export function thumbnailUrl(id: string): string {
	return `/api/items/${encodeURIComponent(id)}/thumbnail`;
}

// --- Source health / filter / refresh (02-02's kernel/httpapi/sources.go) ---

export interface SourceStatus {
	name: string; // source INSTANCE id (the [sources.<id>] config key, D-08) — matches StreamItem.source, filter/staleness/grant identity
	source_type: string; // plugin kind (Describe-reported), matches StreamItem.source_type — descriptive only, never identity
	display_name: string; // this instance's resolved display name (D-09): configured display_name, or `name` itself when omitted
	plugin: string; // plugin BINARY name (e.g. "topos-plugin-mock") — the key PluginIcon.svelte addresses GET /api/plugins/{plugin}/icon with (09-01-PLAN.md Task 3)
	// tier is this instance's launch-time provenance (Phase 11, PLUG-06/07)
	// — published by kernel/httpapi/sources.go's sourceStatus.Tier, derived
	// EXCLUSIVELY from which configured directory the launched binary
	// resolved from (kernel/pluginhost.ResolveBinary), never from anything
	// the plugin itself declares. TrustBadge and SourceChip's tooltip
	// render off this field alone (T-11-01).
	tier: 'trusted' | 'external';
	// pinned_hash/current_hash/launch_failure are declared here now (the
	// complete Phase 11 wire surface, so no later plan re-edits this file)
	// but not yet published by the kernel — that lands with pin
	// verification in a later Phase 11 plan (D-01/D-02/D-03). pinned_hash
	// is the SHA-256 pinned in [plugins.pins] for this instance's binary
	// (external tier only, D-04); current_hash is the hash actually on
	// disk at last launch attempt; launch_failure carries 'pin_mismatch'
	// when the two disagree, driving the "binary changed" health state
	// and the chip menu's re-pin action.
	pinned_hash?: string;
	current_hash?: string;
	launch_failure?: '' | 'pin_mismatch';
	reachable: boolean;
	syncing: boolean;
	last_status: '' | 'running' | 'ok' | 'error'; // '' = never run = unknown
	last_sync_unix: number;
	last_error: string;
	// last_notice (12-09-PLAN.md, published on GET /api/sources) is a
	// non-fatal, human-readable advisory the KERNEL recorded about the
	// LAST COMPLETED sync — today, that a webspace's explicit match block
	// matched none of this source's items. A non-empty value does NOT
	// imply failure: it coexists with last_status: 'ok' and an empty
	// last_error, which is exactly the "healthy but contributing nothing"
	// shape this field exists to surface (12-10-PLAN.md). Per docs/api.md
	// a client must never parse or branch on its text — render it and
	// nothing more. Optional, for the same reason launch_failure is:
	// existing fixtures build this object literally.
	last_notice?: string;
}

export interface SourcesResponse {
	schema_version: number;
	sources: SourceStatus[];
}

// RefreshResult mirrors one entry of the kernel's runStatus JSON shape
// (kernel/httpapi/sources.go) exactly, per docs/api.md — this diverges
// from PLAN.md's <interfaces> sketch (which named the field "source" and
// included a "started_unix" the kernel does not send); the live kernel
// code and docs/api.md are authoritative over that sketch, consistent
// with how 02-01/02-02 reconciled their own interface sketches.
export interface RefreshResult {
	name: string;
	source_type: string;
	status: 'ok' | 'error';
	item_count: number;
	error: string;
	coalesced: boolean;
	finished_unix: number;
}

export interface SourceRefreshResponse {
	schema_version: number;
	source: RefreshResult;
}

export interface SyncRefreshResponse {
	schema_version: number;
	sources: RefreshResult[];
}

/** GET /api/sources */
export function getSources(): Promise<SourcesResponse> {
	return getJSON<SourcesResponse>('/api/sources');
}

/** POST /api/sources/{name}/refresh */
export function refreshSource(name: string): Promise<SourceRefreshResponse> {
	return postJSON<SourceRefreshResponse>(`/api/sources/${encodeURIComponent(name)}/refresh`);
}

/** POST /api/sync */
export function refreshAll(): Promise<SyncRefreshResponse> {
	return postJSON<SyncRefreshResponse>('/api/sync');
}

// --- Config read/write (07-01's kernel/httpapi/config.go, D-16/D-17/D-18) ---
//
// Every field below mirrors kernel/config/types.go's json tags exactly —
// this is the RAW (pre-expansion) config document: `${VAR}` secret
// references travel verbatim in both directions, never a resolved value
// (D-05). The wire shape is deliberately the same struct shape TOML
// decodes into on the kernel side, so a round trip through this client
// never needs a translation layer.

export interface AgentGrantConfig {
	read: boolean;
	handoff: boolean;
}

export interface SourceConfig {
	plugin: string;
	base_url?: string;
	token?: string;
	api_version?: string;
	username?: string;
	webmail_base_url?: string;
	ca_cert?: string;
	sync_interval?: string;
	path?: string;
	// recursive is the frontend half of the typed config.Source.Recursive
	// bool (12-03-PLAN.md Task 1) — the filesystem plugin's "Include
	// subfolders" checkbox (12-UI-SPEC.md F1) writes here. Mirrors that
	// field's own omit-when-false wire behaviour (`omitempty` on both
	// toml/json tags in kernel/config/types.go): absent or false both mean
	// "don't recurse," the conservative default.
	recursive?: boolean;
	agent: AgentGrantConfig;
	display_name?: string;
	// extras is this instance's opaque, per-plugin passthrough map (D-12,
	// D-13) — kernel/config/types.go's Source.Extras mirrored verbatim:
	// string values only, round-tripped through the canonical TOML rewrite
	// without kernel interpretation. Present only when non-empty
	// (`omitempty` on the wire, matching every other optional field on
	// this interface).
	extras?: Record<string, string>;
}

export interface WebspaceConfig {
	keywords: string[];
	sources: string[];
	match: Record<string, Record<string, string[]>>;
	// filter is the promoted-search permanent filter stack (D-16/D-17/
	// D-18): each entry an AND-ed FTS term, appended by "Save as filter"
	// and removed independently, in stored array order (UI-12 ordering
	// edge). Optional/absent means no active filter.
	filter?: string[];
}

export interface KernelConfig {
	server: { listen: string };
	index: { path: string };
	// plugins.external_dir/pins mirror kernel/config/types.go's
	// PluginsConfig.ExternalDir/Pins (Phase 11, D-09/D-01/D-02): the
	// second, untrusted plugin directory and the per-external-binary
	// SHA-256 pin map. Both optional/absent-when-empty on the wire,
	// matching the Go struct's own `omitempty` tags.
	plugins: { dir: string; external_dir?: string; pins?: Record<string, string> };
	sync: { interval: string };
	sources: Record<string, SourceConfig>;
	webspaces: Record<string, WebspaceConfig>;
}

export interface ConfigResponse {
	schema_version: number;
	// hash is the base_hash a subsequent putConfig call must echo back
	// (D-03's optimistic content-hash lock) — a save whose base_hash no
	// longer matches the file on disk is rejected with
	// config_changed_on_disk rather than silently overwriting a
	// concurrent hand-edit.
	hash: string;
	config: KernelConfig;
	// env_vars reports, per ${VAR}/$VAR reference found anywhere in
	// config, whether that variable is currently set in the kernel
	// process's own environment — a boolean only, never the value (D-05,
	// D-15's set/unset secret badge).
	env_vars: Record<string, boolean>;
	unknown_keys: string[];
}

export interface ConfigSaveRequest {
	base_hash: string;
	config: KernelConfig;
}

/** GET /api/config */
export function getConfig(): Promise<ConfigResponse> {
	return getJSON<ConfigResponse>('/api/config');
}

/** PUT /api/config */
export function putConfig(req: ConfigSaveRequest): Promise<ConfigResponse> {
	return putJSON<ConfigResponse>('/api/config', req);
}

/**
 * POST /api/config/reload (D-08) — re-reads config.toml from disk through
 * the identical validate-then-apply path a save uses; the only way a
 * hand-edited file reaches the running kernel, since there is deliberately
 * no file watcher. Takes no request body. On success, returns the
 * identical ConfigResponse shape GET/PUT /api/config return. On failure
 * (422 config_invalid), the kernel's previously running configuration is
 * left completely untouched — see ManageSourcesModal.svelte's own
 * handling of the rejection.
 */
export function reloadConfig(): Promise<ConfigResponse> {
	return postJSON<ConfigResponse>('/api/config/reload');
}

// --- Plugin-type discovery / Describe (07-02's kernel/httpapi/config.go,
// D-11's "+" chip picker) ---

export interface PluginTypesResponse {
	schema_version: number;
	// plugin_types is every discovered plugin BINARY name (e.g.
	// "topos-plugin-paperless"), excluding the mock reference fixture —
	// the "New {plugin type}…" picker row list.
	plugin_types: string[];
	// plugin_type_tiers is an ADDITIVE sibling (Phase 11, PLUG-06/07): a
	// tier lookup table spanning EVERY discovered binary in BOTH
	// directories, keyed by binary name — kernel/httpapi/config.go's
	// pluginTypesResponse.PluginTypeTiers mirrored verbatim. Deliberately
	// wider than plugin_types (it also covers excluded fixture names,
	// since it is a lookup table for names a caller already holds, never
	// a second catalog to browse) — see that Go type's own doc comment.
	// No schema_version bump accompanies this field.
	plugin_type_tiers: Record<string, 'trusted' | 'external'>;
}

/** GET /api/config/plugin-types */
export function listPluginTypes(): Promise<PluginTypesResponse> {
	return getJSON<PluginTypesResponse>('/api/config/plugin-types');
}

// --- Per-item marks (KERN-09/KERN-10, 13-01's kernel/httpapi/marks.go) ---

/** MarkAction mirrors marksRequest.action on the wire: "add" excludes, "remove" includes. */
export type MarkAction = 'add' | 'remove';

export interface MarksResponse {
	schema_version: number;
	webspace: string;
	kind: string;
	action: MarkAction;
	changed: number;
	// excluded_count is the webspace's LIVE total after this write — the
	// same count the excluded-items view toggle renders (13-UI-SPEC.md
	// E4), so a caller never has to track a running total itself.
	excluded_count: number;
}

/**
 * POST /api/webspaces/{webspace}/marks. kind is always 'excluded' — the
 * only mark kind this app writes (KERN-09/KERN-10) — so callers only ever
 * choose the action and the item id(s).
 */
export function setItemMarks(
	webspace: string,
	action: MarkAction,
	itemIds: string[]
): Promise<MarksResponse> {
	return postJSON<MarksResponse>(`/api/webspaces/${encodeURIComponent(webspace)}/marks`, {
		kind: 'excluded',
		action,
		item_ids: itemIds
	});
}

/**
 * deleteJSON mirrors getJSON/postJSON/putJSON's error-envelope handling
 * exactly, for the kernel's one DELETE route
 * (DELETE /api/config/whatsapp-link/{session}, 08-03-PLAN.md Task 3).
 */
async function deleteJSON<T>(path: string): Promise<T> {
	const res = await fetch(path, { method: 'DELETE' });
	if (!res.ok) {
		let envelope: ApiErrorEnvelope | undefined;
		try {
			envelope = (await res.json()) as ApiErrorEnvelope;
		} catch {
			// response body wasn't the error envelope (e.g. the kernel is
			// entirely unreachable) — fall through to a generic error below.
		}
		if (envelope?.error) {
			throw new ApiError(envelope.error.code, envelope.error.message);
		}
		throw new ApiError('http_error', `Request to ${path} failed with status ${res.status}`);
	}
	return (await res.json()) as T;
}

// --- WhatsApp in-app QR link session (08-03's kernel/httpapi/whatsapplink.go,
// D-01 — docs/api.md's "POST /api/config/whatsapp-link, GET
// /api/config/whatsapp-link/{session}, DELETE
// /api/config/whatsapp-link/{session}" section is authoritative for every
// field name below) ---

// WhatsAppLinkState mirrors docs/api.md's seven wire values exactly, plus
// the DELETE route's own "cancelled" terminal value — never an eighth,
// locally-invented state. 'pairing_accepted' and 'already_linked' are
// non-terminal (G-08-1): a consumer must keep polling after observing
// either, exactly like 'pending' and 'qr'.
export type WhatsAppLinkState =
	| 'pending'
	| 'qr'
	| 'pairing_accepted'
	| 'already_linked'
	| 'paired'
	| 'error'
	| 'timeout'
	| 'cancelled';

export interface WhatsAppLinkSession {
	schema_version: number;
	session: string;
	state: WhatsAppLinkState;
	// Present only when state === 'qr'.
	png_data_uri?: string;
	expires_in_seconds?: number;
	// Present only when state === 'error'.
	code?: string;
	message?: string;
}

export interface StartWhatsAppLinkRequest {
	// plugin MUST be a member of a prior listPluginTypes() result — the
	// kernel re-checks this itself (T-08-06) and never trusts this value
	// blindly, identical discipline to describePlugin's own plugin field.
	plugin: string;
	// path is the WhatsApp instance's own data directory (the same value
	// as [sources.whatsapp].path).
	path: string;
	// instance is present for the Re-link… flow — an already-configured
	// instance name the kernel suspends for the session's duration — and
	// absent for the Add-Source flow, where nothing is configured yet to
	// suspend.
	instance?: string;
}

/** POST /api/config/whatsapp-link */
export function startWhatsAppLink(req: StartWhatsAppLinkRequest): Promise<WhatsAppLinkSession> {
	return postJSON<WhatsAppLinkSession>('/api/config/whatsapp-link', req);
}

/** GET /api/config/whatsapp-link/{session} */
export function pollWhatsAppLink(session: string): Promise<WhatsAppLinkSession> {
	return getJSON<WhatsAppLinkSession>(`/api/config/whatsapp-link/${encodeURIComponent(session)}`);
}

/** DELETE /api/config/whatsapp-link/{session} */
export function cancelWhatsAppLink(session: string): Promise<WhatsAppLinkSession> {
	return deleteJSON<WhatsAppLinkSession>(`/api/config/whatsapp-link/${encodeURIComponent(session)}`);
}

export interface DescribePluginRequest {
	// plugin is the binary name — must be a member of a prior
	// listPluginTypes() result; the kernel re-checks this itself
	// (T-07-09) and never trusts this value blindly.
	plugin: string;
	// source carries the connection fields typed into step 1, or an
	// already-configured instance's own stored Source (the one-step
	// existing-instance add flow reuses this same RPC to learn that
	// instance's declared match vocabulary — see AddSourceModal.svelte).
	// Nothing here is persisted by this call.
	source: SourceConfig;
}

// ExtrasFieldDecl is one entry of a later Phase 11 plan's
// DescribePluginResponse.extras (D-15): the plugin contract's optional
// declaration of an expected extras key, so the add-source form can
// render a labeled input instead of relying entirely on the free-form
// key/value editor. `secret` mirrors SecretField's own treatment (a
// declared secret-ish key renders through SecretField, identical
// treatment to a secret connection field); `placeholder` is
// display-only — a declared default is NEVER auto-filled into the bound
// value (D-14: a malicious plugin cannot get its suggested default
// silently saved).
export interface ExtrasFieldDecl {
	key: string;
	label: string;
	required: boolean;
	secret: boolean;
	placeholder: string;
}

export interface DescribePluginResponse {
	schema_version: number;
	source_type: string;
	plugin_display_name: string;
	match_vocabulary: string[];
	// tier/binary_hash/env_var_names/extras are declared here now (the
	// complete Phase 11 wire surface, so no later plan re-edits this
	// file) — DescribePluginHandler gains the code that POPULATES them in
	// a later Phase 11 plan alongside pin verification and the extras
	// form. tier is kernel-derived provenance (T-11-01), never
	// plugin-asserted; binary_hash is computed kernel-side from the
	// RESOLVED binary's bytes, never supplied by the plugin process;
	// env_var_names are NAMES only, never values — the same env_vars
	// boolean-only discipline ConfigResponse already applies (D-05),
	// extended to the trial-launch describe response so the
	// untrusted-confirm interstitial (E1) can disclose which variables
	// this source's own configuration references.
	tier: 'trusted' | 'external';
	binary_hash: string;
	env_var_names: string[];
	extras: ExtrasFieldDecl[];
}

/** POST /api/config/describe-plugin */
export function describePlugin(req: DescribePluginRequest): Promise<DescribePluginResponse> {
	return postJSON<DescribePluginResponse>('/api/config/describe-plugin', req);
}
