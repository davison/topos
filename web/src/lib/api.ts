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

/** GET /api/webspaces */
export function listWebspaces(): Promise<WebspacesResponse> {
	return getJSON<WebspacesResponse>('/api/webspaces');
}

/** GET /api/webspaces/{webspace}/stream */
export function getStream(webspace: string): Promise<StreamResponse> {
	return getJSON<StreamResponse>(`/api/webspaces/${encodeURIComponent(webspace)}/stream`);
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
	silverbullet: 'SilverBullet'
};

/** Human-friendly display name for a source_type. */
export function sourceDisplayName(sourceType: string): string {
	return SOURCE_DISPLAY_NAMES[sourceType] ?? sourceType;
}
