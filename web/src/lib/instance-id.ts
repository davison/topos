// The single instance-id derivation and collision-check site every
// new-instance write path in AddSourceModal.svelte must call before it
// reaches upsertSourceInstance (config-edit.ts).
//
// Why this module exists (07-REVIEW.md CR-01): saveAnyway() omitted the
// duplicate-instance-id collision guard its sibling handleConnectNext
// enforced. upsertSourceInstance replaces a [sources.<id>] block wholesale
// (`next.sources[instanceId] = source`) — so an unguarded call from any
// caller silently overwrites another instance's base_url/token reference
// AND resets its agent.read/agent.handoff grants to false, since the
// new-instance flow always initialises `agent` fresh. It is reachable
// through ordinary UI interaction (typing a display name that happens to
// derive to an existing instance id), needs no confirmation, and passes
// D-03's hash lock because nothing else touched the file.
//
// Every path that writes a brand-new [sources.<id>] block must resolve its
// candidate id through resolveNewInstanceId first. Only what the kernel
// structurally cannot express is rejected here — a blank id, or one already
// present in config.sources — everything else, including display-name
// uniqueness, is left to the kernel's own load-time validator, so there is
// one rule set.

import type { KernelConfig } from './api';

/**
 * Discriminated result of resolveNewInstanceId: either a resolved,
 * collision-free candidate id, or a reason the caller must not proceed to
 * write, paired with the exact user-facing message to render.
 */
export type InstanceIdResult =
	| { ok: true; id: string }
	| { ok: false; reason: 'blank' | 'collision'; message: string };

/**
 * Turns a typed display name into a candidate [sources.<id>] map key:
 * trimmed, lowercased, every run of non-[a-z0-9] characters collapsed to a
 * single hyphen, leading/trailing hyphens stripped. Pure — never touches a
 * config document.
 */
export function deriveInstanceId(displayName: string): string {
	return displayName
		.trim()
		.toLowerCase()
		.replace(/[^a-z0-9]+/g, '-')
		.replace(/^-+|-+$/g, '');
}

/**
 * Derives a candidate instance id from `displayName` and checks it against
 * `cfg.sources` before any caller is allowed to write. Returns, in order:
 * the blank result when the derived id is the empty string; the collision
 * result when `cfg.sources[candidateId]` is already present; otherwise the
 * ok result carrying the derived id. Reads `cfg`, never mutates it.
 */
export function resolveNewInstanceId(cfg: KernelConfig, displayName: string): InstanceIdResult {
	const candidateId = deriveInstanceId(displayName);
	if (candidateId === '') {
		return { ok: false, reason: 'blank', message: 'Enter a display name so this instance has an id.' };
	}
	if (cfg.sources[candidateId]) {
		return {
			ok: false,
			reason: 'collision',
			message: `An instance id "${candidateId}" already exists — choose a different display name.`
		};
	}
	return { ok: true, id: candidateId };
}
