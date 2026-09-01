// Resolves the shipped build artifacts (the kernel binary and its plugin
// binaries) `make e2e` produces, and links exactly the requested subset of
// plugin binaries into a fixture's own temp plugins directory — never a
// directory copy or glob of bin/plugins (D-07: paperless, silverbullet,
// proton and signal must never enter this hermetic harness; T-07.1-02).
import { createHash } from 'node:crypto';
import { spawnSync } from 'node:child_process';
import { existsSync, mkdirSync, readFileSync, rmSync, symlinkSync } from 'node:fs';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';

// This file lives at web/e2e/fixtures/plugin-binaries.ts — three levels
// below the repo root (fixtures -> e2e -> web -> root).
const THIS_FILE = fileURLToPath(import.meta.url);
const REPO_ROOT = resolve(dirname(THIS_FILE), '../../..');

/** Absolute path to the shipped kernel binary `make e2e` builds. */
export const KERNEL_BIN = join(REPO_ROOT, 'bin', 'topos');

/** Absolute path to the directory `make e2e` builds plugin binaries into. */
export const PLUGIN_BIN_DIR = join(REPO_ROOT, 'bin', 'plugins');

/**
 * Absolute path to the directory `make external-demo` builds the
 * genuinely out-of-repo proof binary (topos-plugin-external-demo,
 * ROADMAP success criterion 5) into — never `bin/plugins/`, never built
 * by `make build`/`make plugins`. `make e2e`
 * depends on this target (11-04-PLAN.md), so this directory is populated
 * by the time any spec runs. A spec proving a real out-of-repo binary
 * passes this as FixtureConfigSpec.externalPluginBinariesSrcDir rather
 * than relying on the default PLUGIN_BIN_DIR every other
 * externalPluginBinaries fixture (e.g. topos-plugin-mockstrict) uses —
 * those are trusted-dir binaries symlinked into a second directory; this
 * one has no trusted-dir copy at all.
 */
export const EXTERNAL_DEMO_BIN_DIR = join(REPO_ROOT, 'bin', 'plugins-external');

/**
 * Absolute path to the release-side/operator-side signed-provenance CLI
 * `make e2e` builds (16-05-PLAN.md Task 2) — the SAME binary
 * cmd/topos-provenance/main.go wraps around the kernel's own producers
 * and verifier. signProvenanceFixture below executes this binary to
 * write real .provenance.json/.provenance.sig pairs; this harness never
 * reimplements the manifest or signature scheme in TypeScript.
 */
export const PROVENANCE_BIN = join(REPO_ROOT, 'bin', 'topos-provenance');

/**
 * Absolute path to the ephemeral e2e-only signing private key file
 * `make e2e` generates (via `topos-provenance keygen`) into bin/ —
 * gitignored (bin/ is wholesale-ignored). The matching PUBLIC key is
 * injected into the e2e kernel build's own -X provenanceKeysExtra
 * link-time seam (Makefile), so a manifest signProvenanceFixture writes
 * with this key is exactly what that specific kernel build accepts —
 * this key grants nothing to any other build, checkout build, or
 * installed release.
 */
export const PROVENANCE_KEY_FILE = join(REPO_ROOT, 'bin', 'e2e-fixture.key');

// PROVENANCE_FIXTURE_KEY_ID must match the Makefile's own
// E2E_PROVENANCE_KEY_ID literal exactly — the two are written
// independently (a Make variable and a TypeScript constant cannot share
// one physical source), so a future edit to either must update both.
export const PROVENANCE_FIXTURE_KEY_ID = 'e2e-fixture';

function assertExists(path: string, label: string): void {
	if (!existsSync(path)) {
		throw new Error(
			`e2e fixture: ${label} not found at ${path} — run \`make e2e\` first to build the kernel and plugin binaries before running specs directly (e.g. via \`npx playwright test\`).`
		);
	}
}

/**
 * linkPluginBinaries creates destDir and symlinks exactly the binaries
 * named in `names` from srcDir (defaults to PLUGIN_BIN_DIR) into it — an
 * explicitly passed, closed set. Defaults to `['topos-plugin-mock']`, the
 * only plugin most of this harness needs. Throws a loud error naming the
 * missing path if KERNEL_BIN or any requested binary does not exist.
 *
 * srcDir (Phase 11, 11-01-PLAN.md Task 3) lets a caller link from a
 * DIFFERENT build output directory than PLUGIN_BIN_DIR — the out-of-repo
 * proof binary a later Phase 11 plan builds lives outside `bin/plugins`
 * entirely, and this is the seam that lets a fixture populate a second,
 * external plugins directory from it without duplicating this function.
 */
export function linkPluginBinaries(
	destDir: string,
	names: string[] = ['topos-plugin-mock'],
	srcDir: string = PLUGIN_BIN_DIR
): void {
	assertExists(KERNEL_BIN, 'kernel binary (bin/topos)');
	mkdirSync(destDir, { recursive: true });
	for (const name of names) {
		const src = join(srcDir, name);
		assertExists(src, `plugin binary "${name}"`);
		const dest = join(destDir, name);
		// force-remove any stale entry first — linkPluginBinaries may be
		// called more than once against the same destDir in a future spec
		// shape, and symlinkSync throws EEXIST otherwise.
		rmSync(dest, { force: true });
		symlinkSync(src, dest);
	}
}

/**
 * hashPluginBinary returns the lowercase hex SHA-256 digest of the named
 * binary's bytes, read from srcDir (defaults to PLUGIN_BIN_DIR) — via
 * readFileSync, which follows the symlink linkPluginBinaries creates, so
 * this hashes the identical bytes the kernel's own ResolveBinary/pin
 * verification will hash at launch time (Phase 11, D-01/D-02). Used by
 * config-builder.ts to write a fixture's `[plugins.pins]` table so an
 * external-tier fixture source stays launchable once pin enforcement is
 * turned on in a later Phase 11 plan.
 */
export function hashPluginBinary(name: string, srcDir: string = PLUGIN_BIN_DIR): string {
	const path = join(srcDir, name);
	assertExists(path, `plugin binary "${name}" (for hashing)`);
	return createHash('sha256').update(readFileSync(path)).digest('hex');
}

/**
 * hashPluginBinaryAtPath is hashPluginBinary's sibling for an already-
 * resolved absolute path (16-03-PLAN.md Task 2, gap closure) — the
 * companion `linkPluginBinaryAs`/config-builder.ts's `externalBinaryLinks`
 * already need: both link a source binary under a CALLER-CHOSEN
 * destination name, so the source path itself (never a `srcDir` + `name`
 * join) is what a fixture pinning that renamed destination must hash.
 */
export function hashPluginBinaryAtPath(path: string): string {
	assertExists(path, `plugin binary path "${path}" (for hashing)`);
	return createHash('sha256').update(readFileSync(path)).digest('hex');
}

/**
 * linkPluginBinaryAs symlinks ONE arbitrary source binary (an absolute
 * path — not necessarily from PLUGIN_BIN_DIR or EXTERNAL_DEMO_BIN_DIR)
 * into destDir under a caller-chosen destination NAME, which need not
 * match the source file's own basename (13-06-PLAN.md Task 2). This is
 * the general form linkPluginBinaries above cannot express — that
 * function always resolves srcDir/name to destDir/name, the same name on
 * both sides.
 *
 * The one real use this exists for: proving the file-drop bypass path
 * (D-12/D-13) by linking `bin/plugins-external/topos-plugin-external-demo`
 * — a real binary `make e2e`'s `external-demo` dependency builds into a
 * directory the manifest generator never scans — into the hermetic
 * kernel's TRUSTED plugin directory. Symlink-based, like
 * linkPluginBinaries, so the kernel's own SHA-256 verification hashes the
 * SAME bytes this helper links from, never a copy.
 */
export function linkPluginBinaryAs(destDir: string, destName: string, srcPath: string): void {
	assertExists(KERNEL_BIN, 'kernel binary (bin/topos)');
	assertExists(srcPath, `plugin binary source path "${srcPath}"`);
	mkdirSync(destDir, { recursive: true });
	const dest = join(destDir, destName);
	// force-remove any stale entry first, matching linkPluginBinaries'
	// own idempotency guarantee.
	rmSync(dest, { force: true });
	symlinkSync(srcPath, dest);
}

/** Optional overrides signProvenanceFixture below accepts. `repo` and
 * `version`/`contract` default to fixed values; `tag` defaults to a value
 * DERIVED from `names` (see signProvenanceFixture below) so two calls
 * signing different binaries into the SAME destDir never collide on the
 * manifest's own <repo>-<tag> basename (cmd/topos-provenance's own
 * naming convention). */
export interface SignProvenanceFixtureOptions {
	keyID?: string;
	// keyFile signs with a key OTHER than the e2e fixture key — one the
	// kernel does not accept — to drive the operator-trusted-key flows
	// (M2-R4, davison/topos#49): the offer, the consent, the withdrawal.
	keyFile?: string;
	repo?: string;
	tag?: string;
	version?: string;
	contract?: string;
}

/** The manifest and signature file paths signProvenanceFixture just wrote. */
export interface SignedProvenanceFixture {
	manifestPath: string;
	signaturePath: string;
}

/**
 * signProvenanceFixture signs every binary named in `names` — already
 * linked into destDir by an earlier linkPluginBinaries/linkPluginBinaryAs
 * call — into ONE release manifest, by executing the real PROVENANCE_BIN
 * `sign` subcommand against PROVENANCE_KEY_FILE (16-05-PLAN.md Task 2,
 * D-01/D-05/D-07). This is the ONE way this harness produces a signed
 * manifest: it shells out to the real CLI rather than reimplementing the
 * manifest or signature format in TypeScript, so the fixture and the
 * kernel it tests can never drift apart silently. Returns the exact
 * manifest/signature paths written, so a caller proving the negative
 * case (a missing signature) can delete signaturePath without needing to
 * re-derive cmd/topos-provenance's own <repo>-<tag> naming convention.
 */
export function signProvenanceFixture(
	destDir: string,
	names: string[],
	opts: SignProvenanceFixtureOptions = {}
): SignedProvenanceFixture {
	assertExists(
		PROVENANCE_BIN,
		'topos-provenance CLI (bin/topos-provenance) — run `make e2e` first'
	);
	assertExists(
		PROVENANCE_KEY_FILE,
		'e2e fixture provenance signing key (bin/e2e-fixture.key) — run `make e2e` first'
	);

	const keyID = opts.keyID ?? PROVENANCE_FIXTURE_KEY_ID;
	const repo = opts.repo ?? 'topos-e2e/fixture';
	// Default tag incorporates `names` so two calls signing DIFFERENT
	// binaries into the SAME destDir never collide on the manifest's own
	// <repo>-<tag> basename — each call's manifest/signature pair stays
	// independently addressable (and independently deletable) even when
	// several signed fixtures share one external plugins directory.
	const tag = opts.tag ?? `v0.0.0-e2e-fixture-${names.join('-')}`;
	const version = opts.version ?? '0.0.0-e2e-fixture';
	const contract = opts.contract ?? 'topos.v1';

	const binaryPaths = names.map((name) => {
		const path = join(destDir, name);
		assertExists(
			path,
			`plugin binary "${name}" — must already be linked into ${destDir} before signing`
		);
		return path;
	});

	const result = spawnSync(
		PROVENANCE_BIN,
		[
			'sign',
			'--key-id',
			keyID,
			'--key-file',
			opts.keyFile ?? PROVENANCE_KEY_FILE,
			'--repo',
			repo,
			'--tag',
			tag,
			'--version',
			version,
			'--contract',
			contract,
			'--out-dir',
			destDir,
			...binaryPaths
		],
		{ encoding: 'utf-8' }
	);
	if (result.status !== 0) {
		throw new Error(
			`signProvenanceFixture: \`topos-provenance sign\` exited ${result.status} for [${names.join(', ')}] into ${destDir}\n--- stdout ---\n${result.stdout}\n--- stderr ---\n${result.stderr}`
		);
	}

	const [manifestPath, signaturePath] = result.stdout.trim().split('\n');
	if (!manifestPath || !signaturePath) {
		throw new Error(
			`signProvenanceFixture: could not parse manifest/signature paths from \`topos-provenance sign\` stdout: ${JSON.stringify(result.stdout)}`
		);
	}
	return { manifestPath, signaturePath };
}

/**
 * keygenScratchKey runs `topos-provenance keygen` for a key the kernel does
 * NOT accept — never injected into the build's link-time seam — into
 * outDir, returning the private key file (for signProvenanceFixture's
 * keyFile) and the base64 public key (what the kernel offers back).
 */
export function keygenScratchKey(
	outDir: string,
	keyID: string
): { keyFile: string; publicKey: string } {
	assertExists(
		PROVENANCE_BIN,
		'topos-provenance CLI (bin/topos-provenance) — run `make e2e` first'
	);
	mkdirSync(outDir, { recursive: true });
	const result = spawnSync(PROVENANCE_BIN, ['keygen', '--key-id', keyID, '--out-dir', outDir], {
		encoding: 'utf-8'
	});
	if (result.status !== 0) {
		throw new Error(
			`keygenScratchKey: keygen exited ${result.status}\n${result.stdout}\n${result.stderr}`
		);
	}
	const keyFile = join(outDir, `${keyID}.key`);
	const publicKey = readFileSync(join(outDir, `${keyID}.pub`), 'utf-8').trim();
	return { keyFile, publicKey };
}
