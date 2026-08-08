// The single seeding site for EditSourceModal.svelte's form state
// (07-08-PLAN.md Task 2, closing 07-REVIEW.md CR-02 alongside Task 1's
// route-side reset).
//
// Why this module exists: CR-02 found that EditSourceModal's mount-time
// seeding was never re-run, because its caller (the webspace route) never
// actually dropped the component across a close — Task 1 fixes that
// caller-side bug directly, but the modal itself held no defensive layer
// of its own. Pulling the seeding expressions out into pure, exported
// functions is what makes them callable from two places (the component's
// own `$state` initialisers AND a defensive reset-on-open effect) without
// duplicating the fallback logic, and what makes the CR-02 regression
// provable behaviourally, with no component-mount test harness.
//
// Both functions return a FRESH object on every call — never the object
// held by the passed config document. This is a deliberate change from the
// modal's previous inline seeding, which handed back the live config
// object directly. The returned value becomes mutable component `$state`;
// aliasing it would let a form edit reach into the config document the
// browser holds and be carried into some later, unrelated save without
// ever passing through a save path — the same silent-corruption family
// CR-01 and CR-02 both belong to. It is also what makes re-seeding
// meaningful: a second call must have no memory of a prior call's caller
// having mutated its returned value.

import type { KernelConfig, SourceConfig } from './api';

/**
 * Returns the connection values to seed EditSourceModal's 'connection' mode
 * form with, for `instance` in `config`. Falls back to an empty-plugin
 * source with both agent grants false when the instance is not present in
 * `config.sources` — the same fallback the modal used inline before this
 * module existed. Always returns a fresh object (including a fresh
 * `agent` object when a stored source is found) — never the object stored
 * in `config` itself. Pure: never mutates `config`.
 */
export function seedConnectionValues(config: KernelConfig, instance: string): SourceConfig {
	const stored = config.sources[instance];
	if (!stored) {
		return { plugin: '', agent: { read: false, handoff: false } };
	}
	return { ...stored, agent: { ...stored.agent } };
}

/**
 * Returns the match block to seed EditSourceModal's 'match' mode form with,
 * for `instance` within `webspace` in `config`. Falls back to an empty
 * object when the webspace, its match map, or that instance's entry within
 * it is absent — the same fallback the modal used inline before this
 * module existed. Always returns a fresh object whose per-key value arrays
 * are also fresh arrays — never the object (or arrays) stored in `config`
 * itself. Pure: never mutates `config`.
 */
export function seedMatchBlock(
	config: KernelConfig,
	webspace: string,
	instance: string
): Record<string, string[]> {
	const stored = config.webspaces[webspace]?.match?.[instance];
	if (!stored) {
		return {};
	}
	const fresh: Record<string, string[]> = {};
	for (const key of Object.keys(stored)) {
		fresh[key] = [...stored[key]];
	}
	return fresh;
}
