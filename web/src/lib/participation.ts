// Client-side mirror of the kernel's own webspace-document semantics —
// kernel/config/types.go's Webspace.IsEmptyShell (D-20, 07-11-PLAN.md) and
// its Participates rule — kept in ONE place so config-edit.ts,
// AddSourceModal.svelte and WebspaceHeader.svelte can never drift from
// each other or from the kernel. Any change to either kernel function must
// be reflected here in the same commit.
//
// 07-14 extends this module with a participating-instances helper for the
// chip row (replacing AddSourceModal.svelte's own inline
// `participatingSet`); this plan (07-11) adds only the null-tolerant
// readers below and the shell discriminator they build.

import type { KernelConfig, WebspaceConfig } from './api';

/**
 * Null-tolerant readers for a webspace's keywords/sources/match. They
 * exist because the Go Webspace struct's collection fields carry no
 * `omitempty` (kernel/config/types.go) — GET /api/config genuinely
 * serializes `null` for a hand-written webspace that omits a key, even
 * though KernelConfig's TypeScript type declares keywords/sources/match as
 * always-present arrays/objects. Reading such a field directly is how a
 * TypeError gets raised inside a component and misreported as something
 * else entirely (the mechanism 07-UAT.md G-07-4 documents on the root
 * route). 07-12 additionally normalizes this kernel-side; these readers
 * stay regardless as defence in depth. `ws` also accepts `undefined` — a
 * webspace not present in this config snapshot at all reads the same as
 * one present with every collection empty.
 */
export function webspaceKeywords(ws: WebspaceConfig | undefined): string[] {
	return ws?.keywords ?? [];
}

export function webspaceSources(ws: WebspaceConfig | undefined): string[] {
	return ws?.sources ?? [];
}

export function webspaceMatch(
	ws: WebspaceConfig | undefined
): Record<string, Record<string, string[]>> {
	return ws?.match ?? {};
}

/**
 * The client-side mirror of kernel/config/types.go's
 * `Webspace.IsEmptyShell` (D-20, 07-11-PLAN.md, closes 07-UAT.md G-07-3):
 * true when all three of keywords/sources/match are empty — the state
 * web/src/lib/config-edit.ts's `addWebspace()` PUTs as the create-webspace
 * modal's first write, and the state a webspace remains in until its
 * first source is added. `undefined` counts as a shell too — a webspace
 * not present in this snapshot has nothing to match.
 *
 * All three conditions are required — a webspace naming instances in
 * `sources` while declaring no match input is NOT a shell; that is the
 * operator-typo shape ("allowlisted a source, told it nothing to match")
 * the kernel's own validator still rejects loudly. `filter` is
 * deliberately not considered: a permanent filter narrows a stream at
 * query time (D-16/D-17/D-18) and cannot itself make a webspace match
 * anything, so a webspace carrying only a filter and nothing else is
 * still a shell for matching purposes.
 */
export function isEmptyWebspaceShell(ws: WebspaceConfig | undefined): boolean {
	return (
		webspaceKeywords(ws).length === 0 &&
		webspaceSources(ws).length === 0 &&
		Object.keys(webspaceMatch(ws)).length === 0
	);
}

/**
 * The client-side mirror of two kernel functions that must be read
 * together: `kernel/config/types.go`'s `Webspace.Participates` (the
 * allowlist gate — an empty allowlist means every configured instance
 * participates, a non-empty one means exactly the named instances; this is
 * Phase 5 D-03's default) AND `kernel/correlate/correlate.go`'s
 * `matchFieldsFor` (07-11's D-20 has-match-input rule — an instance with no
 * explicit `match` block and an empty `keywords` fallback does not
 * participate, even if the allowlist gate would otherwise admit it). Both
 * rules must hold for an instance to appear in the returned set; this
 * conjunction is deliberately exactly what the kernel would sync, so a
 * chip on screen always corresponds to a source the kernel would really
 * place items into.
 *
 * Iterates the CONFIGURED instances (`cfg.sources`), never the webspace's
 * own allowlist — an allowlist naming an instance that is not configured
 * can therefore never leak a phantom id into the result; the returned set
 * is always a subset of `Object.keys(cfg.sources)`.
 *
 * This replaces `AddSourceModal.svelte`'s inline `participatingSet`, which
 * implemented the allowlist gate only — correct before D-20, incomplete
 * after it, since a D-20 empty shell would otherwise report every
 * configured instance as a participant. A shared implementation is why
 * that indirection is worth it: `config-edit.ts`, `AddSourceModal.svelte`
 * and `WebspaceHeader.svelte` read one answer instead of three that can
 * drift.
 */
export function participatingInstances(cfg: KernelConfig, webspace: string): Set<string> {
	const ws = cfg.webspaces[webspace];
	const allowlist = webspaceSources(ws);
	const keywords = webspaceKeywords(ws);
	const match = webspaceMatch(ws);

	const result = new Set<string>();
	for (const instance of Object.keys(cfg.sources)) {
		const allowlistGate = allowlist.length === 0 || allowlist.includes(instance);
		if (!allowlistGate) continue;

		const hasMatchInput =
			Object.prototype.hasOwnProperty.call(match, instance) || keywords.length > 0;
		if (!hasMatchInput) continue;

		result.add(instance);
	}
	return result;
}

/**
 * The single-instance form of `participatingInstances`, expressed in terms
 * of that same set so the two functions cannot drift from one another. See
 * `participatingInstances`'s doc comment for the two kernel rules this
 * mirrors.
 */
export function participatesIn(cfg: KernelConfig, webspace: string, instance: string): boolean {
	return participatingInstances(cfg, webspace).has(instance);
}
