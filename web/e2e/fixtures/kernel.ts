// The kernel fixture: boots the shipped `bin/topos serve` binary against a
// freshly generated temp config.toml on an OS-assigned ephemeral port, with
// an explicit environment allowlist that keeps it off the operator's real
// config/index/plugins (T-07.1-05) — the same hermetic-boot mechanics
// scripts/dev-guard-smoke.sh established for `make dev`'s own guard,
// ported to TypeScript. One kernel per spec FILE (D-02): see the
// `configSpec` worker-option design note below.
import { test as base, expect } from '@playwright/test';
import type { ChildProcess } from 'node:child_process';
import { spawn } from 'node:child_process';
import { mkdirSync, rmSync } from 'node:fs';
import { mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { createServer } from 'node:net';
import { join } from 'node:path';

import { KERNEL_BIN, linkPluginBinaries } from './plugin-binaries';
import { buildConfig, writeConfig, type FixtureConfigSpec } from './config-builder';

export { expect };

export interface KernelFixture {
	baseURL: string;
	configPath: string;
	tmpDir: string;
	/** Terminates the kernel process group and removes tmpDir. Idempotent. */
	stop(): Promise<void>;
	/** The kernel child process's captured stdout+stderr, bounded, for failure messages. */
	logs(): string;
}

const READY_TIMEOUT_MS = 30_000;
const TEARDOWN_TIMEOUT_MS = 10_000;
const POLL_INTERVAL_MS = 200;
const LOG_BUFFER_MAX_CHARS = 64 * 1024;

function sleep(ms: number): Promise<void> {
	return new Promise((resolve) => setTimeout(resolve, ms));
}

function setsEqual(a: Set<string>, b: Set<string>): boolean {
	if (a.size !== b.size) return false;
	for (const v of a) if (!b.has(v)) return false;
	return true;
}

// A bounded, front-discard in-memory buffer for the kernel child's own
// stdout/stderr — mirrors kernel/pluginhost.launch's stderrTail capture
// (bounded, front-discard), so a fixture failure can show the kernel's own
// last output as evidence rather than a bare timeout with no context.
class BoundedLogBuffer {
	private chunks: string[] = [];
	private size = 0;

	append(chunk: string): void {
		this.chunks.push(chunk);
		this.size += chunk.length;
		while (this.size > LOG_BUFFER_MAX_CHARS && this.chunks.length > 1) {
			const removed = this.chunks.shift();
			if (removed) this.size -= removed.length;
		}
	}

	toString(): string {
		return this.chunks.join('');
	}
}

/**
 * allocateEphemeralPort binds 127.0.0.1:0, reads the OS-assigned port, and
 * releases it — the same technique dev-guard-smoke.sh's free_port() uses
 * (python's socket.bind(("127.0.0.1", 0))). Never a fixed port with a
 * retry loop.
 */
export async function allocateEphemeralPort(): Promise<number> {
	return new Promise((resolve, reject) => {
		const server = createServer();
		server.on('error', reject);
		server.listen(0, '127.0.0.1', () => {
			const address = server.address();
			if (address === null || typeof address === 'string') {
				server.close();
				reject(new Error('allocateEphemeralPort: could not resolve an assigned port'));
				return;
			}
			const port = address.port;
			server.close((err) => {
				if (err) reject(err);
				else resolve(port);
			});
		});
	});
}

interface LaunchedKernel {
	baseURL: string;
	configPath: string;
	tmpDir: string;
	child: ChildProcess;
	logBuffer: BoundedLogBuffer;
	stopped: boolean;
}

async function waitForKernelReady(
	baseURL: string,
	expectedWebspaces: Set<string>,
	child: ChildProcess,
	logBuffer: BoundedLogBuffer
): Promise<void> {
	const deadline = Date.now() + READY_TIMEOUT_MS;
	let lastError: unknown = null;

	while (Date.now() < deadline) {
		if (child.exitCode !== null || child.signalCode !== null) {
			throw new Error(
				`kernel fixture: kernel process exited before becoming ready (exitCode=${child.exitCode}, signal=${child.signalCode})\n--- captured kernel output ---\n${logBuffer.toString()}`
			);
		}

		let res: { ok: boolean; status: number; json(): Promise<unknown> } | null = null;
		try {
			res = await fetch(`${baseURL}/api/webspaces`);
		} catch (err) {
			lastError = err;
		}

		if (res !== null) {
			if (!res.ok) {
				lastError = new Error(`unexpected status ${res.status}`);
			} else {
				const body = (await res.json()) as { webspaces?: Array<{ name: string }> };
				const actual = new Set((body.webspaces ?? []).map((w) => w.name));
				if (!setsEqual(actual, expectedWebspaces)) {
					// A mismatch means the probe reached some OTHER kernel
					// (a stale listener on a recycled port) — fail
					// immediately rather than retrying into a vacuous pass
					// against it (07.1-CONTEXT.md's key_links "readiness
					// poll" note).
					throw new Error(
						`kernel fixture: readiness probe reached a kernel reporting an unexpected webspace set — expected [${[...expectedWebspaces].join(', ')}], got [${[...actual].join(', ')}]. This usually means the probe reached some OTHER kernel.\n--- captured kernel output ---\n${logBuffer.toString()}`
					);
				}
				return;
			}
		}

		await sleep(POLL_INTERVAL_MS);
	}

	throw new Error(
		`kernel fixture: kernel never became ready within ${READY_TIMEOUT_MS}ms at ${baseURL}/api/webspaces (last error: ${String(lastError)})\n--- captured kernel output ---\n${logBuffer.toString()}`
	);
}

const SYNC_TIMEOUT_MS = 30_000;

export interface WaitForFirstSyncOptions {
	/** Deadline in ms — defaults to 30s. */
	timeoutMs?: number;
	/** A KernelFixture (or any object with a `logs()` method) to include captured kernel output in a timeout error. */
	logs?: () => string;
}

/**
 * waitForFirstSync polls GET /api/sources until every instance named in
 * `instanceIds` reports `syncing: false` AND a non-empty `last_status`,
 * bounded by opts.timeoutMs (default 30s). "Kernel is listening"
 * (waitForKernelReady, above) and "the first sync landed" are two
 * different events — kernel/syncer/scheduler.go's Scheduler.Run fires
 * each configured source's first refresh immediately at boot, so the race
 * between a spec's first assertion and that refresh landing is real, not
 * theoretical; skipping this gate is what turns into an intermittent
 * empty-stream flake. Fails loud with the captured kernel logs (when
 * opts.logs is supplied) and the last response body on exhaustion.
 */
export async function waitForFirstSync(
	baseURL: string,
	instanceIds: string[],
	opts: WaitForFirstSyncOptions = {}
): Promise<void> {
	const timeoutMs = opts.timeoutMs ?? SYNC_TIMEOUT_MS;
	const deadline = Date.now() + timeoutMs;
	let lastBody: unknown = null;
	let lastError: unknown = null;

	while (Date.now() < deadline) {
		try {
			const res = await fetch(`${baseURL}/api/sources`);
			if (res.ok) {
				const body = (await res.json()) as {
					sources?: Array<{ name: string; syncing: boolean; last_status: string }>;
				};
				lastBody = body;
				const byName = new Map((body.sources ?? []).map((s) => [s.name, s]));
				const allLanded = instanceIds.every((id) => {
					const s = byName.get(id);
					return s !== undefined && s.syncing === false && s.last_status !== '';
				});
				if (allLanded) return;
			} else {
				lastError = new Error(`unexpected status ${res.status}`);
			}
		} catch (err) {
			lastError = err;
		}
		await sleep(POLL_INTERVAL_MS);
	}

	throw new Error(
		`waitForFirstSync: source(s) [${instanceIds.join(', ')}] did not report a landed first sync within ${timeoutMs}ms.\nlast response body: ${JSON.stringify(lastBody)}\nlast error: ${String(lastError)}\n--- captured kernel output ---\n${opts.logs ? opts.logs() : '(no logs supplied)'}`
	);
}

function waitForExit(child: ChildProcess, timeoutMs: number): Promise<boolean> {
	return new Promise((resolve) => {
		if (child.exitCode !== null || child.signalCode !== null) {
			resolve(true);
			return;
		}
		const timer = setTimeout(() => resolve(false), timeoutMs);
		child.once('exit', () => {
			clearTimeout(timer);
			resolve(true);
		});
	});
}

async function terminateProcessGroup(child: ChildProcess): Promise<void> {
	if (child.exitCode !== null || child.signalCode !== null) return;
	const pid = child.pid;
	if (pid === undefined) return;

	const exited = waitForExit(child, TEARDOWN_TIMEOUT_MS);
	try {
		// `detached: true` at spawn puts the kernel in its own process
		// group, so -pid signals the whole group — the kernel's own
		// launched plugin subprocesses get reaped too, not just the
		// kernel itself.
		process.kill(-pid, 'SIGTERM');
	} catch {
		// process group may already be gone
	}
	const exitedInTime = await exited;
	if (!exitedInTime) {
		try {
			process.kill(-pid, 'SIGKILL');
		} catch {
			// ignore
		}
		await waitForExit(child, TEARDOWN_TIMEOUT_MS);
	}
}

async function stopLaunched(launched: LaunchedKernel): Promise<void> {
	if (launched.stopped) return;
	launched.stopped = true;
	await terminateProcessGroup(launched.child);
	rmSync(launched.tmpDir, { recursive: true, force: true });
}

async function launchKernel(configSpec: FixtureConfigSpec): Promise<LaunchedKernel> {
	const port = await allocateEphemeralPort();
	const tmpDir = mkdtempSync(join(tmpdir(), 'topos-e2e-'));

	const configDir = join(tmpDir, 'topos');
	const indexDir = join(tmpDir, 'index');
	const pluginsDirPath = join(tmpDir, 'plugins');
	// externalPluginsDirPath (Phase 11, 11-01-PLAN.md Task 3): the second,
	// untrusted plugin directory buildConfig always writes into
	// `plugins.external_dir` — created unconditionally, alongside the
	// existing trusted directory, so an empty externalPluginBinaries list
	// still produces an existing-but-empty directory (the legitimate
	// empty tier, D-09), never a missing one.
	const externalPluginsDirPath = join(tmpDir, 'plugins-external');
	const shareDir = join(tmpDir, 'share');
	const cacheDir = join(tmpDir, 'cache');
	for (const dir of [configDir, indexDir, pluginsDirPath, externalPluginsDirPath, shareDir, cacheDir]) {
		mkdirSync(dir, { recursive: true });
	}

	const doc = buildConfig(configSpec, { port, tmpDir });
	writeConfig(configDir, doc);
	const configPath = join(configDir, 'config.toml');

	linkPluginBinaries(pluginsDirPath, configSpec.pluginBinaries ?? ['topos-plugin-mock']);
	linkPluginBinaries(externalPluginsDirPath, configSpec.externalPluginBinaries ?? []);

	const logBuffer = new BoundedLogBuffer();
	const child = spawn(KERNEL_BIN, ['serve'], {
		detached: true,
		// Explicit allowlist, never a spread of process.env (T-07.1-05):
		// nothing else is inherited — in particular the developer's own
		// XDG_CONFIG_HOME must never reach the child, or the test kernel
		// would load the operator's real config.toml. configSpec.env is
		// layered on TOP of this fixed allowlist (never replacing it) —
		// the one sanctioned way a spec reaches a mock-plugin fixture env
		// var (e.g. WEBSPACES_MOCK_RENDITION, docs/testing.md); it cannot
		// override PATH/HOME/XDG_* since those keys are set again below.
		env: {
			...configSpec.env,
			PATH: process.env.PATH,
			HOME: tmpDir,
			XDG_CONFIG_HOME: tmpDir,
			XDG_DATA_HOME: shareDir,
			XDG_CACHE_HOME: cacheDir
		}
	});
	child.stdout?.on('data', (chunk) => logBuffer.append(chunk.toString()));
	child.stderr?.on('data', (chunk) => logBuffer.append(chunk.toString()));

	const baseURL = `http://127.0.0.1:${port}`;
	const expectedWebspaces = new Set((configSpec.webspaces ?? []).map((w) => w.name));

	await waitForKernelReady(baseURL, expectedWebspaces, child, logBuffer);

	return { baseURL, configPath, tmpDir, child, logBuffer, stopped: false };
}

// --- Playwright fixture wiring ------------------------------------------
//
// D-02 fixture scoping decision: `kernel` is a WORKER-scoped fixture keyed
// on the `configSpec` worker-option. Playwright creates a fresh
// worker-fixture instance whenever an option value it depends on differs
// (the same mechanism behind the well-known "storageState per project"
// pattern) — so even though Playwright may reuse one OS worker process
// across multiple spec files, a file that calls `test.use({ configSpec })`
// with its own distinct object at the top of the file gets its own fresh
// `kernel` instance, and the previous file's kernel is torn down first.
// This gives an exact one-kernel-per-distinct-configSpec-value guarantee
// (in practice, one per spec file, since each file declares its own
// config per D-03) without needing to tune Playwright's `workers` count to
// the spec-file count — which wouldn't scale as 07.1-02..07.1-05 add more
// files. See 07.1-01-SUMMARY.md for the full write-up of why this was
// chosen over the plan's other offered variant (test-scoped fixture +
// `test.describe.configure({ mode: 'serial' })`).
type WorkerOptions = {
	configSpec: FixtureConfigSpec;
	kernel: KernelFixture;
};

export const test = base.extend<object, WorkerOptions>({
	configSpec: [{}, { option: true, scope: 'worker' }],
	kernel: [
		async ({ configSpec }, use) => {
			const launched = await launchKernel(configSpec);
			const fixture: KernelFixture = {
				baseURL: launched.baseURL,
				configPath: launched.configPath,
				tmpDir: launched.tmpDir,
				stop: () => stopLaunched(launched),
				logs: () => launched.logBuffer.toString()
			};
			try {
				await use(fixture);
			} finally {
				await stopLaunched(launched);
			}
		},
		{ scope: 'worker' }
	]
});
