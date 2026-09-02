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

import type { KernelConfig, SourceConfig, WebspaceConfig, OfferedKey } from './api';
import { isEmptyWebspaceShell, webspaceSources } from './participation';

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
		...(existing?.filter_by_source !== undefined
			? { filter_by_source: existing.filter_by_source }
			: {}),
		...(existing?.date_from !== undefined ? { date_from: existing.date_from } : {}),
		...(existing?.date_to !== undefined ? { date_to: existing.date_to } : {}),
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
		...(existing?.filter !== undefined ? { filter: existing.filter } : {}),
		...(existing?.date_from !== undefined ? { date_from: existing.date_from } : {}),
		...(existing?.date_to !== undefined ? { date_to: existing.date_to } : {}),
		...(existing?.filter_by_source !== undefined
			? { filter_by_source: existing.filter_by_source }
			: {})
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
 * 5 D-03's all-instances-participate default) AND is not a D-20 empty
 * webspace shell (see below), the allowlist is first seeded with every
 * currently configured instance — exactly the participation set "no
 * allowlist" already meant in practice — before appending `instance`, so
 * a webspace the user is now composing in the UI becomes explicit about
 * exactly what it already had and never silently loses a source. A save
 * through setWebspaceFilter (or any other helper in this file) never
 * takes this seeding step — only this function does, and only for the
 * instance it is actively adding (D-14's "never as a side effect of an
 * unrelated save"). When an allowlist already exists, `instance` is
 * appended to its end without reordering the rest; a name already present
 * is a no-op beyond that append check.
 *
 * Shell case (D-20, 07-11-PLAN.md, closes 07-UAT.md G-07-3): a webspace
 * created by `addWebspace` has no participants to preserve — seeding it
 * from every configured instance there would silently drag every OTHER
 * source into a webspace the user just created and, worse, produce a
 * document the kernel rejects, because `validateFallbackCoverage` would
 * find those dragged-in instances participating with nothing to match. So
 * a shell's allowlist starts empty and receives only the instance being
 * added — D-14's own words, "participation is exactly what was added via
 * +", are literally what this path now writes for a shell.
 *
 * The shell test below is evaluated against `cfg` — this function's own,
 * PRE-setMatchBlock input — rather than the document `setMatchBlock`
 * returns. Evaluating it after would mean the webspace is never a shell
 * by the time the question is asked (setMatchBlock has already written a
 * non-empty `match`), making the shell branch dead code — the one
 * ordering mistake this change invites.
 */
export function addSourceToWebspace(
	cfg: KernelConfig,
	webspace: string,
	instance: string,
	block: Record<string, string[]>
): KernelConfig {
	const wasShell = isEmptyWebspaceShell(cfg.webspaces[webspace]);

	const withMatch = setMatchBlock(cfg, webspace, instance, block);
	const ws = withMatch.webspaces[webspace];
	let sources = webspaceSources(ws);
	if (sources.length === 0 && !wasShell) {
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
 *
 * Seeds the current participant set from `cfg` — this function's own,
 * PRE-mutation input — before filtering: the webspace's existing allowlist
 * when it is non-empty, and otherwise every configured instance
 * (`Object.keys(cfg.sources)`), read through `webspaceSources`'s
 * null-tolerant reader so a hand-written `sources: null` seeds rather than
 * throws. This mirrors `addSourceToWebspace`'s long-standing seeding,
 * which the add direction has always performed and the remove direction
 * never did (07-14-PLAN.md, closes 07-UAT.md `G-07-6`). Filtering the
 * allowlist array directly, without seeding first, was wrong: for a
 * webspace with no explicit allowlist (Phase 5 D-03's all-participate
 * default, `sources: []`), filtering an empty array yields `[]` again —
 * which `kernel/config/types.go`'s `Webspace.Participates` reads as "every
 * configured instance still participates." The write round-trips with a
 * 200 and the removed instance keeps syncing into the webspace; every
 * chip flashes a sync spinner and nothing changes, with no error anywhere
 * to explain it (`G-07-6`'s exact reported symptom).
 *
 * Pinned boundary (D-14): removing the last named entry from an explicit
 * allowlist yields an empty allowlist, which — for a webspace still
 * declaring a `keywords` fallback — means the remaining instances rejoin
 * under the all-participate default. This is the only outcome the config
 * format can represent (there is no meaningful "allowlist of zero, and
 * still narrowed" shape distinct from the all-participate default), and it
 * is invisible on the UI-built path: a webspace with no keywords either is
 * then a D-20 empty shell with no participants at all.
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

	const currentAllowlist = webspaceSources(cfg.webspaces[webspace]);
	const seededAllowlist = currentAllowlist.length > 0 ? currentAllowlist : Object.keys(cfg.sources);

	next.webspaces[webspace] = {
		...existing,
		match,
		sources: seededAllowlist.filter((s) => s !== instance)
	};
	dropSourceFilterEntry(next.webspaces[webspace], instance);
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

/**
 * Returns a new document with `[sources.<instanceId>]` removed entirely,
 * AND every reference to it cleared from every webspace's own document —
 * its `[webspaces.<name>.match.<instanceId>]` block, and its position (if
 * any) in each webspace's `sources` allowlist (07-05-PLAN.md Task 1, the
 * Manage Sources modal's instance-delete flow). Leaving a dangling
 * reference behind would fail the kernel's own load-time
 * validateMatchBlocks/validateSourcesAllowlist checks with a confusing
 * "unknown instance" message — a delete through this UI must never
 * produce that half-applied state, so both references are cleared in the
 * same document this function returns.
 *
 * Decision (T-07-26, recorded per this plan's own instruction to read
 * Webspace.Participates and choose): when removing the instance leaves a
 * webspace's `sources` allowlist EMPTY, this function writes `sources: []`
 * rather than attempting to omit the key. There is no meaningful "drop the
 * key" alternative here — KernelConfig's `sources` field is not optional
 * on the wire (every WebspaceConfig always carries it), so the only two
 * representable shapes are a non-empty array or an empty one, and
 * kernel/config/types.go's own `Participates` treats a zero-length
 * `Sources` slice identically to an absent/omitted one: "every configured
 * instance participates by default." Writing `[]` is therefore not a
 * lesser substitute for omission — it IS the kernel's own encoding of
 * "back to the all-instances-participate default," which is exactly the
 * correct semantics once the one instance that allowlist named is gone.
 */
export function removeSourceInstance(cfg: KernelConfig, instanceId: string): KernelConfig {
	const next = cloneConfig(cfg);
	delete next.sources[instanceId];
	for (const name of Object.keys(next.webspaces)) {
		const ws = next.webspaces[name];
		if (Object.prototype.hasOwnProperty.call(ws.match, instanceId)) {
			const match = { ...ws.match };
			delete match[instanceId];
			ws.match = match;
		}
		if (ws.sources.includes(instanceId)) {
			ws.sources = ws.sources.filter((s) => s !== instanceId);
		}
		dropSourceFilterEntry(ws, instanceId);
	}
	return next;
}


/**
 * Drops `instance`'s entry from ws.filter_by_source in place, removing the
 * key entirely when the map empties — the kernel validates a
 * filter_by_source entry against configured, participating instances, so
 * a webspace losing an instance must lose that instance's filter terms in
 * the same write (M2-R3, #55; the same dangling-reference rule
 * removeSourceInstance already applies to match blocks).
 */
function dropSourceFilterEntry(ws: WebspaceConfig, instance: string): void {
	if (!ws.filter_by_source || !(instance in ws.filter_by_source)) return;
	const map = { ...ws.filter_by_source };
	delete map[instance];
	if (Object.keys(map).length === 0) {
		delete ws.filter_by_source;
	} else {
		ws.filter_by_source = map;
	}
}

/**
 * Returns a new document with `[webspaces.<webspace>.filter_by_source.<instance>]`
 * set to `terms`, or removed outright when `terms` is empty — an empty
 * entry is never written, since the kernel's validateFilterBySource
 * rejects zero-term entries exactly as validateMatchBlocks rejects
 * zero-field blocks (M2-R3, #55). A webspace absent from the input is
 * created with empty defaults first, the same never-needs-a-special-case
 * discipline as setWebspaceFilter above.
 */
export function setSourceFilterTerms(
	cfg: KernelConfig,
	webspace: string,
	instance: string,
	terms: string[]
): KernelConfig {
	const next = cloneConfig(cfg);
	const existing = next.webspaces[webspace];
	const ws: WebspaceConfig = {
		keywords: existing?.keywords ?? [],
		sources: existing?.sources ?? [],
		match: { ...(existing?.match ?? {}) },
		...(existing?.filter !== undefined ? { filter: existing.filter } : {}),
		...(existing?.date_from !== undefined ? { date_from: existing.date_from } : {}),
		...(existing?.date_to !== undefined ? { date_to: existing.date_to } : {}),
		...(existing?.filter_by_source !== undefined
			? { filter_by_source: { ...existing.filter_by_source } }
			: {})
	};
	if (terms.length === 0) {
		dropSourceFilterEntry(ws, instance);
	} else {
		ws.filter_by_source = { ...(ws.filter_by_source ?? {}), [instance]: terms };
	}
	next.webspaces[webspace] = ws;
	return next;
}

/**
 * Splits a Save-as-filter input into the global term and per-instance
 * terms (M2-R3, #55): each whitespace token of the form `<instance>:<term>`
 * whose prefix names one of `instances` goes to that instance's own list;
 * whatever remains is re-joined as ONE global term, preserving the
 * long-standing "the whole query is one filter term" behaviour when no
 * token targets an instance. A token with a colon whose prefix is NOT a
 * configured instance stays in the global term untouched — never silently
 * dropped, never misassigned.
 */
export function splitFilterInput(
	raw: string,
	instances: string[]
): { global: string; bySource: Record<string, string[]> } {
	const bySource: Record<string, string[]> = {};
	const rest: string[] = [];
	for (const token of raw.trim().split(/\s+/)) {
		if (token === '') continue;
		const i = token.indexOf(':');
		const prefix = i > 0 ? token.slice(0, i) : '';
		const term = i > 0 ? token.slice(i + 1) : '';
		if (prefix !== '' && term !== '' && instances.includes(prefix)) {
			(bySource[prefix] ??= []).push(term);
		} else {
			rest.push(token);
		}
	}
	return { global: rest.join(' '), bySource };
}


/**
 * Returns a new document with the webspace's saved date range set (M3-R1,
 * #70): either side empty string clears that side; both empty removes both
 * keys — an absent range is never written as "". Every other field is
 * preserved, the same discipline as setSourceFilterTerms above.
 */
export function setWebspaceDateRange(
	cfg: KernelConfig,
	name: string,
	from: string,
	to: string
): KernelConfig {
	const next = cloneConfig(cfg);
	const existing = next.webspaces[name];
	const ws: WebspaceConfig = {
		keywords: existing?.keywords ?? [],
		sources: existing?.sources ?? [],
		match: { ...(existing?.match ?? {}) },
		...(existing?.filter !== undefined ? { filter: existing.filter } : {}),
		...(existing?.filter_by_source !== undefined
			? { filter_by_source: existing.filter_by_source }
			: {})
	};
	if (from) ws.date_from = from;
	if (to) ws.date_to = to;
	next.webspaces[name] = ws;
	return next;
}

/**
 * Returns a new document with `[plugins.pins]` gaining (or overwriting) an
 * entry for `pluginBinary` set to `hash` — the untrusted-source confirm
 * step's write (Phase 11, D-01/D-02, T-11-25). `hash` MUST be the
 * kernel-computed `binary_hash` a describe response already returned; this
 * function never computes, invents, or otherwise derives a hash of its
 * own — it only ever echoes back a value the caller already holds. Pins
 * are per BINARY, not per source instance (D-02): every existing or
 * future instance of `pluginBinary` shares the one pin this overwrites,
 * so a re-accept (the chip menu's future "Trust updated binary" action)
 * updates every instance at once by construction, never diverging.
 */
export function setPluginPin(cfg: KernelConfig, pluginBinary: string, hash: string): KernelConfig {
	const next = cloneConfig(cfg);
	next.plugins = {
		...next.plugins,
		pins: { ...(next.plugins.pins ?? {}), [pluginBinary]: hash }
	};
	return next;
}

/**
 * setTrustedKey (M2-R4, davison/topos#49): trust an offered signing key —
 * add (or replace, by id) a [[plugins.trusted_keys]] entry from the
 * kernel-reported offer, stamping trusted_at now. Every future release
 * that key signs then runs at the operator_trusted tier, unpinned. Pure,
 * like setPluginPin: the write rides the caller's own putConfig.
 */
export function setTrustedKey(
	cfg: KernelConfig,
	offer: OfferedKey,
	note = '',
	now: Date = new Date()
): KernelConfig {
	const next = cloneConfig(cfg);
	const kept = (next.plugins.trusted_keys ?? []).filter((k) => k.id !== offer.id);
	next.plugins = {
		...next.plugins,
		trusted_keys: [
			...kept,
			{
				id: offer.id,
				public_key: offer.public_key,
				trusted_at: now.toISOString(),
				note
			}
		]
	};
	return next;
}

/**
 * removeTrustedKey: stop trusting a key — its plugins return to the
 * external tier at their next launch, by name, into the consent-and-pin
 * path. Removing the last entry drops the table entirely (omitempty).
 */
export function removeTrustedKey(cfg: KernelConfig, keyId: string): KernelConfig {
	const next = cloneConfig(cfg);
	const kept = (next.plugins.trusted_keys ?? []).filter((k) => k.id !== keyId);
	const { trusted_keys: _dropped, ...rest } = next.plugins;
	next.plugins = kept.length ? { ...rest, trusted_keys: kept } : rest;
	return next;
}
