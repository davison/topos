// Minimal ambient type declarations for the Node.js builtins and runtime
// globals the Playwright fixture/spec tree (web/e2e/) uses. No @types/node
// package is installed in this project — Task 1's package-legitimacy gate
// scoped its approval to exactly @playwright/test, playwright and
// smol-toml (see 07.1-01-SUMMARY.md); adding a fourth, unapproved package
// mid-task would bypass that gate outright. This mirrors
// web/src/lib/node-builtins.d.ts's own established "narrow declarations
// for exactly what's imported, nothing more" discipline (Phase 03-06),
// scoped to this separate e2e/tsconfig.json root rather than the
// SvelteKit app's.
declare namespace NodeJS {
	// Opaque handle type — callers never read fields off it, only pass it
	// straight back into clearTimeout/clearInterval.
	interface Timeout {}
}

// Opaque handle type (Phase 11, 11-01-PLAN.md Task 3's hashPluginBinary):
// node:fs.readFileSync's no-encoding overload returns this, and
// node:crypto's Hash.update accepts it — callers never read fields off it
// directly, only pass the raw bytes straight through to the hasher.
//
// 11-06-PLAN.md Task 3 (the binary-changed/re-pin spec) additionally needs
// the Buffer NAMESPACE's own static constructors — appending a tampered
// trailing byte to a real binary's bytes needs Buffer.from/Buffer.concat,
// narrowed to exactly that pair, matching this file's own established
// "exactly what's imported, nothing more" discipline.
interface Buffer {}
declare var Buffer: {
	from(data: number[]): Buffer;
	concat(list: Buffer[]): Buffer;
};

declare var process: {
	env: Record<string, string | undefined>;
	platform: string;
	kill(pid: number, signal?: string | number): boolean;
	// 'exit' only (fixtures/corpus.ts): the one Node hook whose lifetime is
	// the process, which is the lifetime a spec's module-scope temp corpus
	// actually has. Deliberately not widened to the full event union —
	// handlers for 'exit' must be synchronous, and narrowing the event name
	// here keeps that constraint visible at the declaration.
	on(event: 'exit', listener: () => void): void;
};

declare function setTimeout(handler: () => void, timeoutMs?: number): NodeJS.Timeout;
declare function clearTimeout(handle: NodeJS.Timeout): void;
declare function setInterval(handler: () => void, timeoutMs?: number): NodeJS.Timeout;
declare function clearInterval(handle: NodeJS.Timeout): void;

// Node 18+ ships a global fetch — narrowed here to exactly the shape this
// fixture tree reads off a response (status/ok/json()).
declare function fetch(input: string, init?: { method?: string }): Promise<{
	ok: boolean;
	status: number;
	json(): Promise<unknown>;
}>;

declare var console: {
	log(...args: unknown[]): void;
	error(...args: unknown[]): void;
};

declare module 'node:fs' {
	export function mkdtempSync(prefix: string): string;
	export function mkdirSync(path: string, options?: { recursive?: boolean }): string | undefined;
	export function rmSync(path: string, options?: { recursive?: boolean; force?: boolean }): void;
	export function existsSync(path: string): boolean;
	export function symlinkSync(target: string, path: string): void;
	// Two overloads, matching node:fs's real shape: no encoding argument
	// returns the raw bytes (Buffer) — hashPluginBinary's own read, since
	// hashing a plugin binary's text-decoded bytes would corrupt them; an
	// encoding argument returns a decoded string, every pre-existing
	// caller's shape.
	export function readFileSync(path: string): Buffer;
	export function readFileSync(path: string, encoding: string): string;
	// Two overloads, matching node:fs's real shape: a string caller (every
	// pre-existing writer in this tree, e.g. config-builder.ts's
	// writeConfig) and a raw-bytes caller (11-06-PLAN.md Task 3's tampered-
	// binary write, which must never round-trip through a text encoding).
	export function writeFileSync(path: string, data: string, encoding?: string): void;
	export function writeFileSync(path: string, data: Buffer): void;
	// The tampered-binary write in 13-manifest-unverified.spec.ts passes
	// Buffer data WITH a { mode } options object; without this overload the
	// call falls into the string overload above and fails TS2345 (#9).
	export function writeFileSync(path: string, data: Buffer, options: { mode: number }): void;
	// chmodSync (11-06-PLAN.md Task 3): restores the executable bit the
	// tampered-binary write above does not itself preserve (writeFileSync
	// creates a fresh file at the default umask-governed mode, not the
	// original symlink target's 0o755).
	export function chmodSync(path: string, mode: number): void;
	export function readdirSync(path: string): string[];
	// unlinkSync (12-03-PLAN.md Task 2, the filesystem-recursion spec):
	// removes the single nested file so the next sync can prove the item
	// drops out of the stream. The block's existing rmSync is a
	// directory-shaped recursive removal, not the right tool for deleting
	// one leaf file, hence this distinct member.
	export function unlinkSync(path: string): void;
}

declare module 'node:path' {
	export function join(...segments: string[]): string;
	export function resolve(...segments: string[]): string;
	export function dirname(path: string): string;
	export function isAbsolute(path: string): boolean;
	// basename (12-01-PLAN.md Task 2, the filesystem-tracer spec): the
	// corpus temp directory's own base name is the D-05 folder-vocabulary
	// label the filesystem plugin emits for a top-level file.
	export function basename(path: string): string;
}

declare module 'node:os' {
	export function tmpdir(): string;
}

declare module 'node:url' {
	export function fileURLToPath(url: string): string;
}

// hashPluginBinary's only dependency (Phase 11, 11-01-PLAN.md Task 3) —
// narrowed to exactly the createHash('sha256').update(bytes).digest('hex')
// chain that function uses, mirroring this file's own established
// discipline.
declare module 'node:crypto' {
	interface Hash {
		update(data: Buffer): Hash;
		digest(encoding: string): string;
	}
	export function createHash(algorithm: string): Hash;
}

declare module 'node:net' {
	export interface AddressInfo {
		port: number;
		address: string;
		family: string;
	}
	export interface Server {
		listen(port: number, host: string, callback: () => void): Server;
		address(): AddressInfo | string | null;
		close(callback?: (err?: Error) => void): Server;
		on(event: string, listener: (err: Error) => void): Server;
	}
	export function createServer(): Server;
}

declare module 'node:child_process' {
	export interface ChildProcessOutputStream {
		on(event: 'data', listener: (chunk: { toString(): string }) => void): void;
	}
	export interface ChildProcess {
		pid?: number;
		exitCode: number | null;
		signalCode: string | null;
		stdout: ChildProcessOutputStream | null;
		stderr: ChildProcessOutputStream | null;
		once(event: 'exit', listener: () => void): void;
	}
	export interface SpawnOptions {
		detached?: boolean;
		env?: Record<string, string | undefined>;
	}
	export function spawn(command: string, args: string[], options?: SpawnOptions): ChildProcess;
	// Phase 16's fixture signing (plugin-binaries.ts signProvenanceFixture)
	// shells out to topos-provenance synchronously with { encoding: 'utf-8' }
	// and reads status/stdout/stderr — the only spawnSync shape this tree
	// uses (#9).
	export interface SpawnSyncOptions {
		encoding: 'utf-8';
	}
	export interface SpawnSyncReturns {
		status: number | null;
		stdout: string;
		stderr: string;
	}
	export function spawnSync(
		command: string,
		args: string[],
		options: SpawnSyncOptions
	): SpawnSyncReturns;
}

// --- Browser-context Window augmentation ---------------------------------
// `window`/`document`/`MutationObserver`/`Node`/`Element` are already
// available from TypeScript's own default-lib set (confirmed empirically:
// this tsconfig's `target: "es2022"` pulls in the DOM lib by default even
// though no `"lib"` option is declared here, and no @types/node is
// installed) — referenced only from inside `page.addInitScript()`/
// `page.evaluate()` callback bodies (07.1-04-PLAN.md Task 2's
// no-skeleton-flash MutationObserver), which execute in the BROWSER, not
// this Node-side fixture/spec process, but whose source text TypeScript
// still type-checks as part of this same compilation unit. The only gap is
// the spec<->browser-context bridge fields those callbacks attach to
// `window` (e.g. `__armSkeletonObserver`) — the real `Window` interface
// naturally has no knowledge of those, so they are added here via
// interface merging (the standard, supported way to extend `Window`)
// rather than by redeclaring `window` itself, which would conflict with
// lib.dom.d.ts's own declaration.
// This file has no top-level import/export, so it stays a global SCRIPT
// (not a module) — a plain top-level `interface Window` here merges
// directly with lib.dom.d.ts's own global `Window` interface with no
// `declare global {}` wrapper needed (that wrapper is only required inside
// a file that TypeScript otherwise treats as a module).
interface Window {
	__skeletonArmed?: boolean;
	__skeletonInsertionCount?: number;
	__armSkeletonObserver?: () => void;
	__disarmSkeletonObserver?: () => void;
}
