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
	source_type: string;
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
 * kernel's POST refresh routes — a failed refresh surfaces the kernel's
 * own error code (e.g. `source_not_found`) rather than a generic message.
 */
async function postJSON<T>(path: string): Promise<T> {
	const res = await fetch(path, { method: 'POST' });
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

/** GET /api/webspaces/{webspace}/stream */
export function getStream(webspace: string): Promise<StreamResponse> {
	return getJSON<StreamResponse>(`/api/webspaces/${encodeURIComponent(webspace)}/stream`);
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

/** Relative kernel path for GET /api/items/{id}/content. */
export function contentUrl(id: string): string {
	return `/api/items/${encodeURIComponent(id)}/content`;
}

/** Relative kernel path for GET /api/items/{id}/thumbnail. */
export function thumbnailUrl(id: string): string {
	return `/api/items/${encodeURIComponent(id)}/thumbnail`;
}

// --- Source health / filter / refresh (02-02's kernel/httpapi/sources.go) ---

export interface SourceStatus {
	name: string; // config key
	source_type: string; // matches StreamItem.source_type
	display_name: string; // e.g. "paperless-ngx", "SilverBullet"
	reachable: boolean;
	syncing: boolean;
	last_status: '' | 'running' | 'ok' | 'error'; // '' = never run = unknown
	last_sync_unix: number;
	last_error: string;
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

// SOURCE_DISPLAY_NAMES is a small local fallback mapping used to
// parameterize source-specific UI copy (RESEARCH.md "Pitfall 3:
// Hardcoded source name in shared UI copy" — DetailPane's failure copy
// and OpenInSource's button label both used to read "paperless-ngx"
// unconditionally, which is wrong for any other source). A live,
// plugin-reported display_name will arrive via a future GET /api/sources
// route (RESEARCH.md "Health merge", a later plan in this phase); this
// mapping is the minimal fix needed now, with a sensible fallback (the
// raw source_type) for any source not yet listed here.
const SOURCE_DISPLAY_NAMES: Record<string, string> = {
	paperless: 'paperless-ngx',
	silverbullet: 'SilverBullet',
	proton: 'Proton Mail',
	signal: 'Signal'
};

/** Human-friendly display name for a source_type. */
export function sourceDisplayName(sourceType: string): string {
	return SOURCE_DISPLAY_NAMES[sourceType] ?? sourceType;
}
