// Resolves the shipped build artifacts (the kernel binary and its plugin
// binaries) `make e2e` produces, and links exactly the requested subset of
// plugin binaries into a fixture's own temp plugins directory — never a
// directory copy or glob of bin/plugins (D-07: paperless, silverbullet,
// proton and signal must never enter this hermetic harness; T-07.1-02).
import { existsSync, mkdirSync, rmSync, symlinkSync } from 'node:fs';
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

function assertExists(path: string, label: string): void {
	if (!existsSync(path)) {
		throw new Error(
			`e2e fixture: ${label} not found at ${path} — run \`make e2e\` first to build the kernel and plugin binaries before running specs directly (e.g. via \`npx playwright test\`).`
		);
	}
}

/**
 * linkPluginBinaries creates destDir and symlinks exactly the binaries
 * named in `names` from PLUGIN_BIN_DIR into it — an explicitly passed,
 * closed set. Defaults to `['topos-plugin-mock']`, the only plugin this
 * phase's harness needs. Throws a loud error naming the missing path if
 * KERNEL_BIN or any requested binary does not exist.
 */
export function linkPluginBinaries(destDir: string, names: string[] = ['topos-plugin-mock']): void {
	assertExists(KERNEL_BIN, 'kernel binary (bin/topos)');
	mkdirSync(destDir, { recursive: true });
	for (const name of names) {
		const src = join(PLUGIN_BIN_DIR, name);
		assertExists(src, `plugin binary "${name}"`);
		const dest = join(destDir, name);
		// force-remove any stale entry first — linkPluginBinaries may be
		// called more than once against the same destDir in a future spec
		// shape, and symlinkSync throws EEXIST otherwise.
		rmSync(dest, { force: true });
		symlinkSync(src, dest);
	}
}
