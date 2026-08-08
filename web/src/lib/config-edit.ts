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

import type { KernelConfig, SourceConfig, WebspaceConfig } from './api';

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

/**
 * Returns a new document with `[webspaces.<webspace>.match.<instance>]`
 * set to `block`, or deleted outright when `block` has no fields — an
 * empty block is never written as `{}` on disk, since the kernel's own
 * load-time validation (kernel/config/config.go's validateMatchBlocks)
 * rejects a zero-field match block as a silently-matches-nothing shape. A
 * webspace absent from the input is created with empty defaults first
 * (same "never needs a special case" discipline as setWebspaceFilter
 * above), so a caller never needs to check for its existence first.
 */
export function setMatchBlock(
	cfg: KernelConfig,
	webspace: string,
	instance: string,
	block: Record<string, string[]>
): KernelConfig {
	const next = cloneConfig(cfg);
	const existing = next.webspaces[webspace];
	const ws: WebspaceConfig = {
		keywords: existing?.keywords ?? [],
		sources: existing?.sources ?? [],
		match: { ...(existing?.match ?? {}) },
		...(existing?.filter !== undefined ? { filter: existing.filter } : {})
	};
	if (Object.keys(block).length === 0) {
		delete ws.match[instance];
	} else {
		ws.match[instance] = block;
	}
	next.webspaces[webspace] = ws;
	return next;
}

/**
 * Returns a new document with `instance` added to `webspace`: its match
 * block set via setMatchBlock, and the webspace's `sources` allowlist
 * extended to include it (D-14, the add-source picker's own write path).
 * When the webspace previously had NO allowlist (`sources` empty — Phase
 * 5 D-03's all-instances-participate default), the allowlist is first
 * seeded with every currently configured instance — exactly the
 * participation set "no allowlist" already meant in practice — before
 * appending `instance`, so a webspace the user is now composing in the UI
 * becomes explicit about exactly what it already had and never silently
 * loses a source. A save through setWebspaceFilter (or any other helper
 * in this file) never takes this seeding step — only this function does,
 * and only for the instance it is actively adding (D-14's "never as a
 * side effect of an unrelated save"). When an allowlist already exists,
 * `instance` is appended to its end without reordering the rest; a name
 * already present is a no-op beyond that append check.
 */
export function addSourceToWebspace(
	cfg: KernelConfig,
	webspace: string,
	instance: string,
	block: Record<string, string[]>
): KernelConfig {
	const withMatch = setMatchBlock(cfg, webspace, instance, block);
	const ws = withMatch.webspaces[webspace];
	let sources = ws.sources;
	if (sources.length === 0) {
		sources = Object.keys(cfg.sources);
	}
	if (!sources.includes(instance)) {
		sources = [...sources, instance];
	}
	ws.sources = sources;
	return withMatch;
}

/**
 * Returns a new document with `instance` removed from `webspace`
 * entirely: its match block dropped and its allowlist entry (if present)
 * removed — the chip menu's "Remove from this webspace" write path
 * (07-04-PLAN.md Task 3). The instance's own `[sources.<id>]` block is
 * untouched; only this webspace's participation changes. A webspace
 * absent from the input, or an instance not present in either place, is a
 * no-op beyond the clone.
 */
export function removeSourceFromWebspace(
	cfg: KernelConfig,
	webspace: string,
	instance: string
): KernelConfig {
	const next = cloneConfig(cfg);
	const existing = next.webspaces[webspace];
	if (!existing) return next;
	const match = { ...existing.match };
	delete match[instance];
	next.webspaces[webspace] = {
		...existing,
		match,
		sources: existing.sources.filter((s) => s !== instance)
	};
	return next;
}

/**
 * Returns a new document with `[sources.<instanceId>]` set to `source` —
 * writing a brand-new instance (the two-step "New {plugin type}…" flow's
 * Step 1 "Save anyway" and Step 2 submit paths, 07-04-PLAN.md Task 2) or
 * replacing an existing one wholesale (the chip menu's "Edit
 * connection…" flow, Task 3). Every other instance is untouched.
 */
export function upsertSourceInstance(
	cfg: KernelConfig,
	instanceId: string,
	source: SourceConfig
): KernelConfig {
	const next = cloneConfig(cfg);
	next.sources[instanceId] = source;
	return next;
}
