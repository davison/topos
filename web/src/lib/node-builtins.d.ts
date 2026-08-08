// Minimal ambient type declarations for the three Node.js built-in module
// functions date-format.test.ts uses to read component source files off
// disk. No @types/node package is installed in this project (it is not a
// runtime dependency of the SvelteKit app, only of the vitest test
// runner's own Node process, which needs no type information to execute)
// and this plan's threat model explicitly records that no dependency is
// added to close this gap — see plugins/proton's mirror-image go.mod/
// go.sum guard. These narrow declarations satisfy `svelte-check` for the
// exact functions the test imports, nothing more.
declare module 'node:fs' {
	export function readFileSync(path: string, encoding: string): string;
	export function readdirSync(path: string): string[];
	// statSync's return type is narrowed to exactly the one member
	// save-state.test.ts's recursive directory walk actually reads
	// (isDirectory()) — same "nothing more than the exact functions/shapes
	// the test imports" discipline the rest of this file follows.
	export function statSync(path: string): { isDirectory(): boolean };
}

declare module 'node:path' {
	export function dirname(path: string): string;
	export function join(...segments: string[]): string;
	export function relative(from: string, to: string): string;
}

declare module 'node:url' {
	export function fileURLToPath(url: string): string;
}
