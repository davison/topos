// Pure config-document edit helpers shared by every builder surface this
// phase (and 07-04/07-05) add — the webspace switcher's "+ New webspace",
// the webspace route's Save-as-filter/remove-filter writes (07-01's inline
// mutation moves here, 07-03-PLAN.md Task 2), and every later source-add/
// source-edit modal. Each function takes a KernelConfig value and returns
// a NEW document; the input is never mutated in place, so a caller can
// safely hold the previous value across the async putConfig() round trip
// until a real response lands.
//
// cloneConfig deep-clones via a JSON round trip, not structuredClone:
// 07-01-PLAN.md's tracer checkpoint (commit d8125cf) found that
// structuredClone unconditionally throws DataCloneError on a Svelte 5
// $state reactive Proxy, in every engine. This file is a plain .ts module
// (not .svelte.ts), so Svelte's own $state.snapshot() rune compiler macro
// isn't available here either — JSON.stringify/parse reads every property
// through a reactive Proxy exactly as an ordinary property access would
// (safe outside a reactive $effect; it just doesn't track), producing a
// fully plain, non-reactive document regardless of whether the caller
// passed a $state value or an already-plain object.

import type { KernelConfig } from './api';

/**
 * Deep-clones a KernelConfig document. Safe against a Svelte 5 reactive
 * Proxy input (see file header) — the returned value is always a plain,
 * non-reactive object structurally identical to the input.
 */
export function cloneConfig(cfg: KernelConfig): KernelConfig {
	return JSON.parse(JSON.stringify(cfg)) as KernelConfig;
}

/**
 * Returns a new document with an empty `[webspaces.<name>]` entry added —
 * no `sources` allowlist yet (D-14: the explicit allowlist is written only
 * once a source is actually added to the webspace via the "+" chip-row
 * trigger, a later plan). A `name` that already exists is overwritten with
 * an equally empty entry — a real collision is a kernel-side load-time
 * validation concern, not this pure helper's job to police.
 */
export function addWebspace(cfg: KernelConfig, name: string): KernelConfig {
	const next = cloneConfig(cfg);
	next.webspaces[name] = { keywords: [], sources: [], match: {} };
	return next;
}

/**
 * Returns a new document with the named webspace removed entirely. A name
 * absent from the input is a no-op beyond the clone itself.
 */
export function removeWebspace(cfg: KernelConfig, name: string): KernelConfig {
	const next = cloneConfig(cfg);
	delete next.webspaces[name];
	return next;
}

/**
 * Returns a new document with the named webspace's `filter` stack replaced
 * by `terms` — the single write path every filter save/remove goes through
 * (07-01-PLAN.md Task 2's "one place a webspace's config is edited" rule,
 * moved here from the webspace route's own inline mutation). Every other
 * field on the webspace (keywords/sources/match) is preserved unchanged; a
 * webspace absent from the input is created with empty defaults before the
 * filter is applied, so a caller never needs to special-case "webspace not
 * yet present in this snapshot."
 */
export function setWebspaceFilter(cfg: KernelConfig, name: string, terms: string[]): KernelConfig {
	const next = cloneConfig(cfg);
	const existing = next.webspaces[name];
	next.webspaces[name] = {
		keywords: existing?.keywords ?? [],
		sources: existing?.sources ?? [],
		match: existing?.match ?? {},
		filter: terms
	};
	return next;
}
