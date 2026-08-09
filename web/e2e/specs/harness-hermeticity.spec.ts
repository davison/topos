// The guard that makes T-07.1-01 and T-07.1-02 testable rather than merely
// intended: proves the harness itself never touches a real path, never
// launches a real (non-mock) plugin binary, and that the kernel a spec
// talks to is genuinely the one the fixture spawned.
import { readdirSync } from 'node:fs';
import { isAbsolute, join } from 'node:path';

import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { buildConfig, mockInstances, webspacesWithKeywords, type FixtureConfigSpec } from '../fixtures/config-builder';
import { readConfigToml } from '../fixtures/toml';

const configSpec: FixtureConfigSpec = {
	sources: mockInstances(1),
	webspaces: webspacesWithKeywords(['hermetic'], ['demo'])
};

test.use({ configSpec });

// Playwright forbids overriding a worker-scoped option (`configSpec`) from
// inside a nested describe block — it can only be set top-level in the
// file (or the config file). Every test in this file therefore shares the
// ONE kernel instance `configSpec` above produces; serial mode pins their
// execution to declaration order within a single worker, so the stop()
// test (deliberately declared LAST) can safely end that shared kernel's
// life without any earlier assertion racing it.
test.describe.configure({ mode: 'serial' });

test('buildConfig throws rather than emitting a config with a relative index.path/plugins.dir', () => {
	// A fixture whose spec "omits index.path" is structurally impossible:
	// no FixtureConfigSpec field even names it — buildConfig always
	// computes it from opts.tmpDir. The only way this guard is ever
	// exercised is opts.tmpDir itself not being absolute, so that's what
	// this proves, with a thrown error rather than a silently-relative
	// path landing in the generated file.
	expect(() => buildConfig({}, { port: 0, tmpDir: 'relative/not-absolute' })).toThrow(/absolute/);
});

test('config.toml parses, and index.path/plugins.dir are both absolute and resolve inside tmpDir', async ({
	kernel
}) => {
	const doc = readConfigToml(kernel.configPath);
	const index = doc.index as { path: string };
	const plugins = doc.plugins as { dir: string };

	expect(isAbsolute(index.path)).toBe(true);
	expect(isAbsolute(plugins.dir)).toBe(true);
	expect(index.path.startsWith(kernel.tmpDir)).toBe(true);
	expect(plugins.dir.startsWith(kernel.tmpDir)).toBe(true);
});

test('the temp plugins directory holds exactly the requested binary set — no real plugin binaries', async ({
	kernel
}) => {
	const entries = readdirSync(join(kernel.tmpDir, 'plugins')).sort();

	expect(entries).toEqual(['topos-plugin-mock']);
	expect(entries.length).toBe(1);

	const forbidden = [
		'topos-plugin-paperless',
		'topos-plugin-silverbullet',
		'topos-plugin-proton',
		'topos-plugin-signal'
	];
	for (const name of forbidden) {
		expect(entries.includes(name)).toBe(false);
	}
});

test('the kernel answers with exactly the seeded webspace set', async ({ kernel }) => {
	await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });

	const res = await fetch(`${kernel.baseURL}/api/webspaces`);
	expect(res.ok).toBe(true);
	const body = (await res.json()) as { webspaces: Array<{ name: string }> };
	const names = body.webspaces.map((w) => w.name).sort();

	expect(names).toEqual(['hermetic']);
});

test("the developer's own XDG_CONFIG_HOME, if set, is not what the kernel read from", async ({ kernel }) => {
	const developerXdg = process.env.XDG_CONFIG_HOME;
	if (developerXdg !== undefined && developerXdg !== '') {
		expect(kernel.tmpDir).not.toBe(developerXdg);
		expect(kernel.configPath.startsWith(developerXdg)).toBe(false);
	}
	// True unconditionally: the kernel's configPath always lives inside
	// this fixture's own tmpDir, never anywhere the developer's real
	// session happens to point XDG_CONFIG_HOME.
	expect(kernel.configPath.startsWith(kernel.tmpDir)).toBe(true);
});

// Declared LAST (serial mode above pins it there): stop()'s own assertion
// ends the shared kernel this file's earlier tests depend on.
test('stop() makes the kernel stop answering, and is idempotent', async ({ kernel }) => {
	const baseURL = kernel.baseURL;

	await kernel.stop();

	let threw = false;
	try {
		await fetch(`${baseURL}/api/webspaces`);
	} catch {
		threw = true;
	}
	expect(threw).toBe(true);

	// A second stop() call must not throw or hang — the fixture's own
	// automatic post-use teardown calls stop() again unconditionally.
	await kernel.stop();
});
