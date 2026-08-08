// Root-redirect memory (D-10, 07-03-PLAN.md Task 3): remembers the
// last-visited webspace across visits, and resolves the redirect target
// for `/`. Pulled forward into Task 2's own commit (Rule 3 — blocking
// compile dependency): Task 2's create-webspace flow in
// web/src/routes/w/[webspace]/+page.svelte calls writeLastWebspace so a
// newly created webspace is remembered immediately, before Task 3's root
// route exists to consume it. Task 3 adds no further exports here — only
// its own route/empty-state wiring and this file's unit tests.
//
// Every browser-storage access is guarded so it is inert outside a browser
// (this app is SPA-only, `ssr = false`, but a guard costs nothing and
// keeps this module safely importable from a test file with no DOM) and
// swallows a storage exception rather than throwing — a private-mode
// browser with storage disabled must still redirect, just without memory.

export const LAST_WEBSPACE_KEY = 'topos:last-webspace';

/** Records `name` as the last-visited webspace. Inert outside a browser; swallows a storage exception. */
export function writeLastWebspace(name: string): void {
	if (typeof localStorage === 'undefined') return;
	try {
		localStorage.setItem(LAST_WEBSPACE_KEY, name);
	} catch {
		// Storage disabled/unavailable (private browsing, quota) — the
		// current visit still works, it just won't be remembered next time.
	}
}

/** Reads the last-remembered webspace name, or null if none is recorded, unavailable, or storage throws. */
export function readLastWebspace(): string | null {
	if (typeof localStorage === 'undefined') return null;
	try {
		return localStorage.getItem(LAST_WEBSPACE_KEY);
	} catch {
		return null;
	}
}

/**
 * Pure "remembered, else first, else none" redirect-target resolver: a
 * `remembered` name still present in `webspaces` wins; otherwise the first
 * webspace in the kernel's returned order; `null` when `webspaces` is
 * empty (the zero-webspaces empty state, never a redirect loop — T-07-18).
 * `remembered` is the raw readLastWebspace() result (or null) — this
 * function itself never touches storage, so the "remembered wins" /
 * "stale remembered falls through" / "empty list yields none" /
 * "a storage read that throws yields the first" cases are all testable
 * without a browser (readLastWebspace's own try/catch is what turns a
 * throwing read into the `null` this function treats as "nothing
 * remembered").
 */
export function resolveRedirectTarget(
	webspaces: string[],
	remembered: string | null
): string | null {
	if (remembered !== null && webspaces.includes(remembered)) return remembered;
	return webspaces[0] ?? null;
}
