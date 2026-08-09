// Builds and writes the temp config.toml a fixture's kernel boots against
// (D-03: state is seeded by generating this file BEFORE kernel boot, never
// by driving the UI to build preconditions). buildConfig is the single
// writer of index.path/plugins.dir (T-07.1-01's mitigation) — every
// downstream helper in this file goes through it, so the absolute-path
// guard below can never be bypassed by a helper that forgets to apply it.
import { isAbsolute, join } from 'node:path';
import { writeFileSync } from 'node:fs';
import { stringify } from 'smol-toml';

/** One [sources.<id>] block, mirroring kernel/config.Source's toml tags. */
export interface FixtureSourceSpec {
	id: string;
	plugin: string;
	path?: string;
	baseUrl?: string;
	token?: string;
	displayName?: string;
	agent?: { read?: boolean; handoff?: boolean };
}

/** One [webspaces.<name>] block, mirroring kernel/config.Webspace's toml tags. */
export interface FixtureWebspaceSpec {
	name: string;
	keywords?: string[];
	sources?: string[];
	match?: Record<string, Record<string, string[]>>;
	filter?: string[];
}

/** The full document buildConfig assembles into a kernel/config.Config shape. */
export interface FixtureConfigSpec {
	sources?: FixtureSourceSpec[];
	webspaces?: FixtureWebspaceSpec[];
	syncInterval?: string;
}

export interface BuildConfigOptions {
	/** The ephemeral port [server].listen binds — see kernel.ts's allocateEphemeralPort. */
	port: number;
	/** The fixture's own temp directory. MUST be absolute. */
	tmpDir: string;
}

// PLACEHOLDER_CONNECTION_VALUE mirrors config.example.toml's own documented
// workaround for a genuinely config-free plugin (the commented-out
// [sources.mock] block's own base_url/token lines): kernel/config.Validate
// requires EVERY configured source to declare either base_url+token or
// path, even one whose plugin itself ignores both (topos-plugin-mock).
// "Any non-empty placeholder value works; the mock plugin ignores both."
const PLACEHOLDER_CONNECTION_VALUE = 'e2e-fixture-unused';

/**
 * buildConfig returns a plain object mirroring kernel/config.Config's toml
 * tags (server, index, plugins, sync, sources, webspaces). index.path and
 * plugins.dir are ALWAYS computed here from opts.tmpDir — never read from
 * the spec, never omitted, never relative — because
 * kernel/config.applyDefaults silently substitutes the operator's real
 * ~/.local/share/topos/index.db and real bin/plugins set for an omitted
 * key (07.1-RESEARCH.md Pitfall 1, T-07.1-01). opts.tmpDir itself must be
 * absolute; buildConfig throws rather than silently producing a relative
 * or empty path — this is what makes "a fixture whose spec omits
 * index.path" structurally impossible rather than merely undocumented.
 */
export function buildConfig(spec: FixtureConfigSpec, opts: BuildConfigOptions): Record<string, unknown> {
	if (!isAbsolute(opts.tmpDir)) {
		throw new Error(
			`config-builder: opts.tmpDir must be an absolute path, got ${JSON.stringify(opts.tmpDir)}`
		);
	}

	const indexPath = join(opts.tmpDir, 'index', 'index.db');
	const pluginsDir = join(opts.tmpDir, 'plugins');

	if (!isAbsolute(indexPath) || !isAbsolute(pluginsDir)) {
		// Unreachable given the isAbsolute(opts.tmpDir) check above and
		// node:path.join's own contract (joining onto an absolute path
		// always stays absolute) — kept as a second, independent guard
		// rather than trusting that invariant silently, since this is
		// exactly the property T-07.1-01 exists to pin.
		throw new Error('config-builder: computed index.path/plugins.dir is not absolute');
	}

	const sources: Record<string, unknown> = {};
	for (const src of spec.sources ?? []) {
		sources[src.id] = buildSourceEntry(src);
	}

	const webspaces: Record<string, unknown> = {};
	for (const ws of spec.webspaces ?? []) {
		webspaces[ws.name] = buildWebspaceEntry(ws);
	}

	return {
		server: { listen: `127.0.0.1:${opts.port}` },
		index: { path: indexPath },
		plugins: { dir: pluginsDir },
		// sync.interval defaults to "1h" (not kernel/config's own "15m"
		// default) so the boot-time refresh happens exactly once per
		// fixture lifetime and no background resync races a mid-spec
		// assertion (07.1-01-PLAN.md Task 2 action).
		sync: { interval: spec.syncInterval ?? '1h' },
		sources,
		webspaces
	};
}

function buildSourceEntry(src: FixtureSourceSpec): Record<string, unknown> {
	const entry: Record<string, unknown> = { plugin: src.plugin };

	const hasPath = src.path !== undefined && src.path !== '';
	const hasBaseUrl = src.baseUrl !== undefined && src.baseUrl !== '';
	const hasToken = src.token !== undefined && src.token !== '';

	if (hasPath) {
		entry.path = src.path;
	} else if (hasBaseUrl || hasToken) {
		entry.base_url = src.baseUrl ?? '';
		entry.token = src.token ?? '';
	} else {
		entry.base_url = PLACEHOLDER_CONNECTION_VALUE;
		entry.token = PLACEHOLDER_CONNECTION_VALUE;
	}

	if (src.displayName !== undefined) entry.display_name = src.displayName;
	if (src.agent !== undefined) {
		entry.agent = { read: src.agent.read ?? false, handoff: src.agent.handoff ?? false };
	}

	return entry;
}

function buildWebspaceEntry(ws: FixtureWebspaceSpec): Record<string, unknown> {
	const entry: Record<string, unknown> = {};
	// Only the keys the spec actually declared are emitted — so a D-20
	// empty shell (no keywords, no sources, no match) stays genuinely
	// empty on disk rather than gaining an accidental `keywords = []`.
	if (ws.keywords !== undefined) entry.keywords = ws.keywords;
	if (ws.sources !== undefined) entry.sources = ws.sources;
	if (ws.match !== undefined) entry.match = ws.match;
	if (ws.filter !== undefined) entry.filter = ws.filter;
	return entry;
}

/** writeConfig serialises doc with smol-toml's stringify and writes config.toml inside configDir. */
export function writeConfig(configDir: string, doc: Record<string, unknown>): void {
	writeFileSync(join(configDir, 'config.toml'), stringify(doc), 'utf-8');
}
