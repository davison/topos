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

import type { WebspaceConfig } from './api';

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
