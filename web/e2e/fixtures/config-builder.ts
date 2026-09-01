// Builds and writes the temp config.toml a fixture's kernel boots against
// (D-03: state is seeded by generating this file BEFORE kernel boot, never
// by driving the UI to build preconditions). buildConfig is the single
// writer of index.path/plugins.dir (T-07.1-01's mitigation) — every
// downstream helper in this file goes through it, so the absolute-path
// guard below can never be bypassed by a helper that forgets to apply it.
import { isAbsolute, join } from 'node:path';
import { writeFileSync } from 'node:fs';
import { stringify } from 'smol-toml';
import { hashPluginBinary, hashPluginBinaryAtPath } from './plugin-binaries';

/** One [sources.<id>] block, mirroring kernel/config.Source's toml tags. */
export interface FixtureSourceSpec {
	id: string;
	plugin: string;
	path?: string;
	baseUrl?: string;
	token?: string;
	displayName?: string;
	// recursive mirrors kernel/config.Source.Recursive (12-03-PLAN.md
	// Task 1) — emitted into the source's TOML entry only when true,
	// mirroring how extras below is emitted only when non-empty, so a
	// fixture naming nothing here stays byte-identical to before this
	// field existed.
	recursive?: boolean;
	agent?: { read?: boolean; handoff?: boolean };
	// extras mirrors kernel/config.Source.Extras (D-12/D-13, Phase 11) —
	// this instance's opaque per-plugin passthrough map. Emitted as the
	// source's `extras` sub-table only when non-empty, so a fixture
	// naming nothing here is byte-identical to before this field existed.
	extras?: Record<string, string>;
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
	// pluginBinaries: forwarded by the kernel fixture to linkPluginBinaries
	// (plugin-binaries.ts) — defaults to ['topos-plugin-mock'] so a spec
	// that only needs the reference mock plugin names nothing extra. A
	// spec needing the Plan 07.1-02 e2e plugin (or any future plugin type
	// this harness grows) names it here without editing this fixture file.
	pluginBinaries?: string[];
	// externalPluginBinaries (Phase 11, 11-01-PLAN.md Task 3): binary
	// names the kernel fixture links into the SECOND, external plugins
	// directory (`plugins.external_dir`) rather than the trusted one —
	// the two-tier sibling of pluginBinaries above. Every name listed
	// here is also written into buildConfig's `[plugins.pins]` table
	// (hashed via hashPluginBinary), so an external-tier fixture source
	// stays launchable once pin enforcement lands in a later Phase 11
	// plan. Empty/undefined by default — a fixture naming nothing here
	// still gets a (legitimately empty) external directory, per D-09's
	// "missing external directory is a legitimate empty tier" rule.
	externalPluginBinaries?: string[];
	// externalPluginBinariesSrcDir (Phase 11, 11-05-PLAN.md Task 3): the
	// directory externalPluginBinaries' names are resolved from for BOTH
	// the symlink source (kernel.ts's linkPluginBinaries call) AND the pin
	// hash below — defaults to PLUGIN_BIN_DIR (bin/plugins), matching
	// every existing fixture's assumption that an "external" fixture
	// binary is simply a trusted-dir binary symlinked into the second
	// directory (e.g. topos-plugin-mockstrict). A spec proving a
	// GENUINELY out-of-repo binary (built by `make external-demo` into
	// `bin/plugins-external/`, never `bin/plugins/`) sets this to
	// plugin-binaries.ts's EXTERNAL_DEMO_BIN_DIR so both the symlink
	// source and the pin hash resolve against the real out-of-repo build
	// artifact rather than throwing (no bin/plugins copy exists for it).
	externalPluginBinariesSrcDir?: string;
	// trustedBinaryLinks (13-06-PLAN.md Task 2): additional binaries linked
	// into the TRUSTED directory alongside pluginBinaries above, but each
	// entry names its OWN destination name and its OWN arbitrary absolute
	// source path — the general, per-entry sibling of
	// externalPluginBinariesSrcDir's list-wide srcDir override. This is
	// the mechanism a spec proving the file-drop bypass path (D-12/D-13)
	// uses to link a real out-of-repo binary (built by `make e2e`'s
	// `external-demo` dependency into `bin/plugins-external/`, never
	// `bin/plugins/`) into the hermetic kernel's trusted directory under a
	// name the link-time build manifest never covers — proving the
	// manifest gate refuses it by name rather than by directory location
	// alone. Empty/undefined by default, so a fixture naming nothing here
	// is byte-identical to before this field existed. Never hashed into
	// [plugins.pins] — that table is for the external tier only.
	trustedBinaryLinks?: { name: string; srcPath: string }[];
	// externalBinaryLinks (16-03-PLAN.md Task 2, gap closure): the EXTERNAL
	// directory's own sibling of trustedBinaryLinks above — each entry
	// symlinks an arbitrary source binary into `plugins.external_dir`
	// under its OWN caller-chosen destination name, which need not match
	// the source file's own basename. This exists because
	// externalPluginBinaries (above) always resolves `srcDir/name` to
	// `destDir/name` — the SAME name both sides — which is exactly wrong
	// once trust became provenance-driven (Phase 16, D-11): a binary NAME
	// covered by the kernel's own link-time build manifest (e.g.
	// `topos-plugin-mockstrict` — `MANIFEST_E2E_BINARIES`,
	// Makefile) now resolves TierTrusted from
	// ANY directory it is found in, including the external one, since
	// tier is a pure function of provenance rather than of location
	// (success criterion 1: "trust is no longer a property of
	// location"). A fixture that genuinely needs to prove "this real
	// binary, loaded from the external directory, resolves external
	// tier" must therefore place it under a name the manifest does NOT
	// cover — this field is that seam, mirroring
	// kernel/pluginhost/manifestgate_test.go's own
	// buildRenamedMockPluginDirForManifestGate precedent at the Go level.
	// Never hashed into [plugins.pins] — that table is keyed by the same
	// renamed destination name a fixture using this field must pin
	// against, so it is the fixture's own responsibility, not
	// buildConfig's, exactly like trustedBinaryLinks above.
	externalBinaryLinks?: { name: string; srcPath: string }[];
	// signedProvenanceBinaries (16-05-PLAN.md Task 2, D-01/D-05/D-07 e2e
	// proof of location-independent trust): each entry names a binary —
	// already linked into the EXTERNAL plugin directory via
	// externalBinaryLinks/externalPluginBinaries above — that gets its
	// OWN signed release manifest (`.provenance.json`/`.provenance.sig`)
	// written beside it, via the real `topos-provenance` CLI
	// (plugin-binaries.ts's signProvenanceFixture), through the link-time
	// `-X ...provenanceKeysExtra` key seam `make e2e` injects (Makefile,
	// D-12). ONE binary per manifest — never batched into a single
	// shared manifest — so a fixture proving the signature-revoked case
	// (`removeSignature: true`) can delete exactly ONE binary's own
	// evidence file without also revoking a sibling entry's trust,
	// letting one kernel boot carry both the positive and negative case
	// side by side (Playwright forbids varying a worker-scoped fixture's
	// option — configSpec — within one spec file, so both cases must
	// share one boot). Crucially, an entry's name must NOT also receive a
	// `[plugins.pins]` entry — see the pin loops below — because its
	// trust comes from provenance, and a pin would mask which mechanism
	// actually granted the launch. Empty/undefined by default, so a
	// fixture naming nothing here is byte-identical to before this field
	// existed.
	// scratchKeyID (M2-R4): sign this entry with a freshly generated key the
	// kernel does not accept, so its binary carries an OFFER instead of
	// evidence; the same id across entries shares one key. reusedID: sign
	// with the scratch key but NAME the accepted fixture key id, to drive
	// the reused-id warning.
	signedProvenanceBinaries?: {
		name: string;
		removeSignature?: boolean;
		scratchKeyID?: string;
		reusedID?: boolean;
	}[];
	// unpinnedExternalBinaries: external binaries that must NOT get a pin
	// written — they launch pin-mismatched, carrying whatever offer their
	// signature yields (M2-R4).
	unpinnedExternalBinaries?: string[];
	// env: extra environment variables layered onto the spawned KERNEL
	// process's fixed allowlist (kernel.ts's launchKernel) — never
	// replacing it. This is HALF of how a spec reaches a mock-plugin
	// fixture env var (e.g. WEBSPACES_MOCK_RENDITION, docs/testing.md):
	// the kernel builds each plugin subprocess's environment from a
	// strict allowlist (kernel/pluginhost's allowedEnv, D-14 — never
	// os.Environ() wholesale), so the value set here reaches the plugin
	// only when that instance's own source config ALSO references
	// ${THE_VAR} (config.EnvRefNames' per-instance allowlist) — see the
	// rendition spec's extras block for the pattern. Empty/undefined by
	// default, so a spec naming nothing extra here is byte-identical to
	// before this field existed.
	env?: Record<string, string>;
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
export function buildConfig(
	spec: FixtureConfigSpec,
	opts: BuildConfigOptions
): Record<string, unknown> {
	if (!isAbsolute(opts.tmpDir)) {
		throw new Error(
			`config-builder: opts.tmpDir must be an absolute path, got ${JSON.stringify(opts.tmpDir)}`
		);
	}

	const indexPath = join(opts.tmpDir, 'index', 'index.db');
	const pluginsDir = join(opts.tmpDir, 'plugins');
	// externalPluginsDir (Phase 11, 11-01-PLAN.md Task 3): the second,
	// untrusted plugin directory, computed here under the identical
	// single-writer/absolute-path discipline as index.path/plugins.dir —
	// so a two-tier fixture never needs a second config-writing seam.
	const externalPluginsDir = join(opts.tmpDir, 'plugins-external');

	if (!isAbsolute(indexPath) || !isAbsolute(pluginsDir) || !isAbsolute(externalPluginsDir)) {
		// Unreachable given the isAbsolute(opts.tmpDir) check above and
		// node:path.join's own contract (joining onto an absolute path
		// always stays absolute) — kept as a second, independent guard
		// rather than trusting that invariant silently, since this is
		// exactly the property T-07.1-01 exists to pin.
		throw new Error(
			'config-builder: computed index.path/plugins.dir/plugins.external_dir is not absolute'
		);
	}

	const sources: Record<string, unknown> = {};
	for (const src of spec.sources ?? []) {
		sources[src.id] = buildSourceEntry(src);
	}

	const webspaces: Record<string, unknown> = {};
	for (const ws of spec.webspaces ?? []) {
		webspaces[ws.name] = buildWebspaceEntry(ws);
	}

	// pins (Phase 11, D-01/D-02): every name in spec.externalPluginBinaries
	// gets an entry here, hashed via hashPluginBinary against the real
	// build artifact — the identical bytes linkPluginBinaries symlinks
	// into the fixture's external directory, so the pin always matches
	// what the kernel will actually hash at launch time. Written here
	// (rather than by a downstream helper) for the same single-writer
	// reason index.path/plugins.dir already are — this is the ONE place
	// [plugins] gets assembled.
	// signedProvenanceNames (16-05-PLAN.md Task 2): names whose trust
	// comes from a signed provenance manifest, never from a pin — the pin
	// loops below must skip every name in this set, so a fixture cannot
	// accidentally mask "which mechanism granted the launch" by writing
	// both a pin AND a signed manifest for the same name.
	const signedProvenanceNames = new Set(
		(spec.signedProvenanceBinaries ?? []).filter((e) => !e.scratchKeyID).map((e) => e.name)
	);
	for (const name of spec.unpinnedExternalBinaries ?? []) signedProvenanceNames.add(name);

	const pins: Record<string, string> = {};
	for (const name of spec.externalPluginBinaries ?? []) {
		if (signedProvenanceNames.has(name)) continue;
		pins[name] = hashPluginBinary(name, spec.externalPluginBinariesSrcDir);
	}
	// externalBinaryLinks (16-03-PLAN.md Task 2, gap closure): each entry's
	// OWN destination name is pinned against the OWN source path's bytes —
	// the renamed-destination sibling of the loop above, since
	// hashPluginBinary's `srcDir + name` join does not apply to an
	// arbitrary, independently-named source path.
	for (const link of spec.externalBinaryLinks ?? []) {
		if (signedProvenanceNames.has(link.name)) continue;
		pins[link.name] = hashPluginBinaryAtPath(link.srcPath);
	}

	return {
		server: { listen: `127.0.0.1:${opts.port}` },
		index: { path: indexPath },
		plugins: { dir: pluginsDir, external_dir: externalPluginsDir, pins },
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
	// recursive (12-03-PLAN.md Task 1): emitted only when true, mirroring
	// extras' non-empty-only emission below — a fixture leaving it
	// undefined (or explicitly false) writes no `recursive` key at all.
	if (src.recursive === true) entry.recursive = true;
	if (src.agent !== undefined) {
		entry.agent = {
			read: src.agent.read ?? false,
			handoff: src.agent.handoff ?? false
		};
	}
	// extras (D-12/D-13, Phase 11): emitted only when non-empty, mirroring
	// every other optional sub-table on this entry.
	if (src.extras !== undefined && Object.keys(src.extras).length > 0) {
		entry.extras = src.extras;
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

// --- Downstream-spec builder helpers (07.1-01-PLAN.md Task 3) -----------
//
// These cover every fixture shape D-03 names (07.1-CONTEXT.md): N mock
// instances, N webspaces sharing a keywords fallback, a D-20 empty shell,
// and a webspace with an explicit sources allowlist + per-instance match
// blocks. Every one of them returns plain FixtureSourceSpec/
// FixtureWebspaceSpec values for the caller to spread into its own
// FixtureConfigSpec — buildConfig stays the ONE place index.path/
// plugins.dir get written, so a helper can never bypass that guard
// (T-07.1-01).

/**
 * mockInstances returns n topos-plugin-mock source specs with
 * deterministic ids ("mock-01" .. "mock-NN") and display names — the
 * fixture shape UAT item 7's "15+ instances" scaling spec needs.
 */
export function mockInstances(n: number): FixtureSourceSpec[] {
	const specs: FixtureSourceSpec[] = [];
	for (let i = 1; i <= n; i++) {
		const suffix = String(i).padStart(2, '0');
		specs.push({
			id: `mock-${suffix}`,
			plugin: 'topos-plugin-mock',
			displayName: `Mock ${suffix}`
		});
	}
	return specs;
}

/**
 * webspacesWithKeywords returns one FixtureWebspaceSpec per name in
 * `names`, each sharing the same `keywords` fallback list.
 */
export function webspacesWithKeywords(names: string[], keywords: string[]): FixtureWebspaceSpec[] {
	return names.map((name) => ({ name, keywords }));
}

/**
 * emptyShellWebspace returns a webspace declaring none of keywords,
 * sources or match — D-20's "a webspace that exists and matches nothing
 * yet" (07-11-PLAN.md, kernel/config.Webspace.IsEmptyShell), the exact
 * document web/src/lib/config-edit.ts's addWebspace() PUTs as the
 * create-webspace modal's first write.
 */
export function emptyShellWebspace(name: string): FixtureWebspaceSpec {
	return { name };
}

/**
 * attachedWebspace returns a webspace whose `sources` allowlist names
 * exactly `instanceIds` and whose `match` map carries a per-instance
 * block — the D-02 explicit-match-block shape, for the uat-03/06/07
 * fixtures that need a webspace pre-attached to specific instances rather
 * than relying on the keywords fallback.
 */
export function attachedWebspace(
	name: string,
	instanceIds: string[],
	match: Record<string, Record<string, string[]>>
): FixtureWebspaceSpec {
	return { name, sources: instanceIds, match };
}
