// Structural guard closing 07-REVIEW.md CR-02 (07-08-PLAN.md gap closure):
// `handleEditClose`/`handleEditSaved` in the webspace route only flipped the
// edit modal's open flag; they never cleared `editInstance`/`editMode`.
// Because EditSourceModal is rendered inside `{#key
// `${editInstance}-${editMode}`}`, reopening the same source in the same
// mode produced an identical key, Svelte never remounted the component, and
// its form `$state` — seeded once at mount — survived from the previous,
// abandoned session (discarded typing included). A later save on the
// reopened session would then write that stale value straight over the
// real connection config.
//
// No component-mount harness exists here (web/vite.config.ts's test block:
// environment: 'node' — see pane-layout.test.ts's own header comment for
// the same reasoning), so this file is the structural half of the proof:
// it scans the route's and EditSourceModal.svelte's own source text off
// disk. The behavioural half — proving the seeding helpers themselves have
// no memory of a discarded session — lives in edit-modal-state.test.ts
// (07-08-PLAN.md Task 2).
//
// House pattern (matches chip-edit-menu.test.ts / pane-layout.test.ts):
// comment-stripped source scanning, `extractBetween` scoping, a
// found-non-empty-source guard first, and one consequence-describing
// message per assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));

// Resolved relative to this test file: up two directories (out of
// components/, out of lib/, into src/), then into the route tree — the
// same resolution pane-layout.test.ts uses, adjusted for this file living
// one directory deeper.
const routePath = join(here, '..', '..', 'routes', 'w', '[webspace]', '+page.svelte');
const editModalPath = join(here, 'EditSourceModal.svelte');
const manageSourcesPath = join(here, 'ManageSourcesModal.svelte');

const rawRoute = readFileSync(routePath, 'utf-8');
const rawEditModal = readFileSync(editModalPath, 'utf-8');
const rawManageSources = readFileSync(manageSourcesPath, 'utf-8');

function stripComments(source: string): string {
	return source
		.replace(/<!--[\s\S]*?-->/g, ' ')
		.replace(/\/\*[\s\S]*?\*\//g, ' ')
		.replace(/\/\/.*$/gm, ' ');
}

function extractBetween(source: string, startMarker: string, endMarker: string): string {
	const startIndex = source.indexOf(startMarker);
	expect(startIndex, `expected to find "${startMarker}" in the scanned source`).toBeGreaterThanOrEqual(0);
	const endIndex = source.indexOf(endMarker, startIndex);
	expect(endIndex, `expected to find "${endMarker}" after "${startMarker}"`).toBeGreaterThan(startIndex);
	return source.slice(startIndex, endIndex + endMarker.length);
}

const strippedRoute = stripComments(rawRoute);
const strippedEditModal = stripComments(rawEditModal);
const strippedManageSources = stripComments(rawManageSources);

describe('edit-modal-reset guard: found non-empty comment-stripped sources', () => {
	it('+page.svelte', () => {
		expect(strippedRoute.length).toBeGreaterThan(0);
	});
	it('EditSourceModal.svelte', () => {
		expect(strippedEditModal.length).toBeGreaterThan(0);
	});
	it('ManageSourcesModal.svelte', () => {
		expect(strippedManageSources.length).toBeGreaterThan(0);
	});
});

const resetEditSessionBody = extractBetween(strippedRoute, 'function resetEditSession() {', '\n\t}');
const handleEditCloseBody = extractBetween(strippedRoute, 'function handleEditClose() {', '\n\t}');
const handleEditSavedBody = extractBetween(strippedRoute, 'async function handleEditSaved() {', '\n\t}');

describe('CR-02: closing the edit modal ends the edit session (route-side)', () => {
	it('handleEditClose calls the shared resetEditSession', () => {
		expect(
			handleEditCloseBody.includes('resetEditSession()'),
			'CR-02: handleEditClose must call resetEditSession() — a close that only flips the open flag leaves a mounted component holding a previous session\'s $state'
		).toBe(true);
	});

	it('handleEditSaved calls resetEditSession() strictly before its awaited reload', () => {
		const resetIndex = handleEditSavedBody.indexOf('resetEditSession()');
		const awaitIndex = handleEditSavedBody.indexOf('await Promise.all(');
		expect(resetIndex, 'expected handleEditSaved to call resetEditSession()').toBeGreaterThanOrEqual(0);
		expect(awaitIndex, 'expected handleEditSaved to await a Promise.all(...) reload').toBeGreaterThanOrEqual(0);
		expect(
			resetIndex < awaitIndex,
			'CR-02: resetEditSession() must run BEFORE the awaited reload triple, so no intermediate render can show the modal against a half-refreshed config'
		).toBe(true);
	});

	it('resetEditSession clears editInstance to null', () => {
		expect(
			resetEditSessionBody.includes('editInstance = null'),
			'CR-02: resetEditSession must clear editInstance to null — that is what makes the {#if configResponse && editInstance} guard destroy the modal subtree on every close'
		).toBe(true);
	});

	it('resetEditSession also resets editMode to \'connection\' and editVocabulary to an empty array', () => {
		expect(
			resetEditSessionBody.includes("editMode = 'connection'"),
			'expected resetEditSession to reset editMode back to its declared initial value \'connection\''
		).toBe(true);
		expect(
			resetEditSessionBody.includes('editVocabulary = []'),
			'expected resetEditSession to reset editVocabulary back to an empty array'
		).toBe(true);
	});

	it('there is exactly one editInstance = null assignment across the whole route file, and it lives inside resetEditSession', () => {
		const occurrences = strippedRoute.match(/editInstance = null/g) ?? [];
		expect(
			occurrences.length,
			'CR-02: a second, divergent editInstance = null reset site must never appear — resetEditSession is the ONE place this session is cleared'
		).toBe(1);
	});
});

describe('CR-02: the other entry point (Manage sources…) still clears its own edit session', () => {
	it('ManageSourcesModal.svelte still nulls editInstance in its edit modal\'s close handler', () => {
		expect(
			strippedManageSources.includes('onclose={() => (editInstance = null)}'),
			'expected ManageSourcesModal.svelte to keep clearing editInstance in EditSourceModal\'s onclose — the chip ⋮ menu route is not the only entry point into this component, and D-13 requires both to be correct'
		).toBe(true);
	});
});

describe('CR-02: EditSourceModal has exactly one seeding site plus an untracked reset-on-open effect', () => {
	it('imports untrack from svelte and both helpers from $lib/edit-modal-state', () => {
		expect(
			strippedEditModal.includes("import { untrack } from 'svelte';"),
			'expected EditSourceModal.svelte to import untrack from svelte for the reset-on-open effect'
		).toBe(true);
		expect(
			strippedEditModal.includes(
				"import { seedConnectionValues, seedMatchBlock } from '$lib/edit-modal-state';"
			),
			'expected EditSourceModal.svelte to import both seeding helpers from $lib/edit-modal-state'
		).toBe(true);
	});

	it('each helper name appears exactly twice — once in a $state initialiser, once in the reset-on-open effect', () => {
		const seedConnectionCalls = strippedEditModal.match(/seedConnectionValues\(/g) ?? [];
		const seedMatchCalls = strippedEditModal.match(/seedMatchBlock\(/g) ?? [];
		expect(
			seedConnectionCalls.length,
			'expected exactly 2 call sites for seedConnectionValues( — a third seeding site must never appear unnoticed, and the two existing sites (initialiser + effect) must never drift apart'
		).toBe(2);
		expect(
			seedMatchCalls.length,
			'expected exactly 2 call sites for seedMatchBlock( — a third seeding site must never appear unnoticed, and the two existing sites (initialiser + effect) must never drift apart'
		).toBe(2);
	});

	it('the reset-on-open effect wraps its seeding calls in untrack, with untrack( appearing before the first helper call', () => {
		const effectBody = extractBetween(strippedEditModal, '$effect(() => {', '\n\t});');
		const untrackIndex = effectBody.indexOf('untrack(');
		const firstHelperIndex = effectBody.indexOf('seedConnectionValues(');
		expect(untrackIndex, 'expected the reset-on-open effect to call untrack(...)').toBeGreaterThanOrEqual(0);
		expect(
			firstHelperIndex,
			'expected the reset-on-open effect to call seedConnectionValues(...)'
		).toBeGreaterThanOrEqual(0);
		expect(
			untrackIndex < firstHelperIndex,
			'a tracked re-seed would wipe the user\'s in-progress typing on any parent config refresh — the seeding calls must sit inside the untracked region, not before it'
		).toBe(true);
	});

	it('contains zero occurrences of the old inline seeding expressions it used to hold', () => {
		expect(
			strippedEditModal.includes('config.sources[instance] ??'),
			'expected the old inline connectionValues seeding expression to be gone — edit-modal-state.ts is now the only seeding site'
		).toBe(false);
		expect(
			strippedEditModal.includes('config.webspaces[webspace]?.match?.[instance]'),
			'expected the old inline matchBlock seeding expression to be gone — edit-modal-state.ts is now the only seeding site'
		).toBe(false);
	});
});
