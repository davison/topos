// 07-04-PLAN.md Task 1's structural guard over AddSourceModal.svelte: the
// "+" trigger's aria-label and dashed-border styling, the picker's
// nothing-left-to-add copy, the popover content's height cap, and the
// one-step (existing-instance) modal's shape — MatchFieldsForm alone, no
// connection fields, a disabled-bindable submit control. Task 2 extends
// this same file (and this same test file) with the two-step new-instance
// flow; every one-step-modal assertion below is scoped via extractBetween
// to the EXISTING-instance Dialog block specifically (located by its
// `open={step === 'existing'}` binding, unique in the file), so these
// assertions keep holding unchanged once that second flow lands alongside
// it.
//
// House pattern (matches webspace-switcher.test.ts / source-chip-pill.test.ts):
// comment-stripped source scanning, a found-non-empty-source guard first,
// and one consequence-describing message per assertion.

import { describe, it, expect } from 'vitest';
import { readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const here = dirname(fileURLToPath(import.meta.url));
const modalPath = join(here, 'AddSourceModal.svelte');
const connectionFormPath = join(here, 'ConnectionForm.svelte');
const editModalPath = join(here, 'EditSourceModal.svelte');

function stripComments(source: string): string {
	return source
		.replace(/<!--[\s\S]*?-->/g, ' ')
		.replace(/\/\*[\s\S]*?\*\//g, ' ')
		.replace(/\/\/.*$/gm, ' ');
}

function extractBetween(source: string, startMarker: string, endMarker: string): string {
	const startIndex = source.indexOf(startMarker);
	expect(
		startIndex,
		`expected to find "${startMarker}" in the scanned source`
	).toBeGreaterThanOrEqual(0);
	const endIndex = source.indexOf(endMarker, startIndex);
	expect(endIndex, `expected to find "${endMarker}" after "${startMarker}"`).toBeGreaterThan(
		startIndex
	);
	return source.slice(startIndex, endIndex + endMarker.length);
}

const raw = readFileSync(modalPath, 'utf-8');
const stripped = stripComments(raw);
const connectionFormStripped = stripComments(readFileSync(connectionFormPath, 'utf-8'));
const editModalStripped = stripComments(readFileSync(editModalPath, 'utf-8'));

describe('add-source guard: found non-empty comment-stripped source', () => {
	it('AddSourceModal.svelte', () => {
		expect(stripped.length).toBeGreaterThan(0);
	});
});

describe('trigger: aria-label and dashed-border styling', () => {
	const triggerBlock = extractBetween(stripped, '<PopoverTrigger>', '</PopoverTrigger>');

	it('carries aria-label="Add source"', () => {
		expect(
			triggerBlock.includes('aria-label="Add source"'),
			'expected the "+" trigger to carry aria-label="Add source"'
		).toBe(true);
	});

	it('carries a dashed-border class', () => {
		expect(
			/border-dashed/.test(triggerBlock),
			'expected the "+" trigger to carry a dashed-border class, the deliberate "this reads as add-new" visual cue'
		).toBe(true);
	});
});

describe('picker: nothing-left-to-add copy', () => {
	it('renders the exact copy', () => {
		expect(
			stripped.includes('All available sources are already in this webspace.'),
			'expected the picker\'s empty branch to render the exact frozen copy'
		).toBe(true);
	});
});

describe('picker content: height-capped and scrollable', () => {
	it('the PopoverContent usage carries both a max-height and an overflow-y-auto class', () => {
		const popoverBlock = extractBetween(stripped, '<PopoverContent', '</PopoverContent>');
		expect(
			/max-h-\d+/.test(popoverBlock),
			'expected the picker popover to carry a max-height class so a long instance list cannot outgrow the viewport'
		).toBe(true);
		expect(
			/overflow-y-auto/.test(popoverBlock),
			'expected the picker popover to carry overflow-y-auto alongside its max-height class'
		).toBe(true);
	});
});

describe('one-step modal: MatchFieldsForm alone, no connection fields', () => {
	const existingDialogBlock = extractBetween(stripped, "open={step === 'existing'}", '</Dialog>');

	it('renders MatchFieldsForm', () => {
		expect(
			existingDialogBlock.includes('<MatchFieldsForm'),
			'expected the existing-instance modal to render MatchFieldsForm'
		).toBe(true);
	});

	it('renders no ConnectionForm (no connection fields — the instance already has them)', () => {
		expect(
			existingDialogBlock.includes('<ConnectionForm'),
			'expected the existing-instance modal to render no connection fields, since the instance already has them'
		).toBe(false);
	});

	it('the submit control carries a disabled binding', () => {
		expect(
			/type="submit"[^>]*disabled=\{/.test(existingDialogBlock),
			'expected the existing-instance modal\'s submit control to bind a disabled state (the shared save-in-flight pattern)'
		).toBe(true);
	});
});

describe('two-step modal: step indicator reads both "1. Connect" and "2. Match"', () => {
	const connectDialogBlock = extractBetween(
		stripped,
		"open={step === 'connect' || step === 'link' || step === 'match' || step === 'connect-saved'}",
		'</Dialog>'
	);

	it('renders "1. Connect"', () => {
		expect(connectDialogBlock.includes('1. Connect')).toBe(true);
	});

	it('renders "2. Match"', () => {
		expect(connectDialogBlock.includes('2. Match')).toBe(true);
	});
});

// 08-04-PLAN.md Task 1 (D-01/D-02): the WhatsApp not-linked branch — a
// trial-launch success for the WhatsApp plugin type routes to the 'link'
// step (the QR opportunity), never treated as a describe failure and never
// revealing Save anyway. QRPanel renders INSIDE this same Step 1 dialog,
// never a newly added one.
describe('WhatsApp not-linked branch (D-01/D-02): link step, never a trial-launch failure', () => {
	const connectDialogBlock = extractBetween(
		stripped,
		"open={step === 'connect' || step === 'link' || step === 'match' || step === 'connect-saved'}",
		'</Dialog>'
	);
	const handleConnectNextBody = extractBetween(
		stripped,
		'async function handleConnectNext(event: SubmitEvent) {',
		'\n\t}'
	);

	it("handleConnectNext's success path routes the WhatsApp plugin type to the 'link' step", () => {
		expect(
			handleConnectNextBody.includes("selectedPluginType === WHATSAPP_PLUGIN_BINARY ? 'link' : 'match'"),
			'expected the trial-launch success branch to route WhatsApp to the link step and every other plugin type to match'
		).toBe(true);
	});

	it('the success path (before the catch block) never sets describeFailed = true — only the catch block does', () => {
		const catchIndex = handleConnectNextBody.indexOf('} catch');
		expect(catchIndex, 'expected handleConnectNext to have a catch block').toBeGreaterThan(0);
		const successBlock = handleConnectNextBody.slice(0, catchIndex);
		expect(
			successBlock.includes('describeFailed = true'),
			'expected the WhatsApp not-linked branch to never set describeFailed — that flag is what reveals Save anyway, and an unpaired WhatsApp instance is not a trial-launch failure'
		).toBe(false);
	});

	it("the 'link' step renders QRPanel inside this same Step 1 dialog block (never a new Dialog)", () => {
		const linkBranch = extractBetween(connectDialogBlock, "{:else if step === 'link'}", '{:else if step === \'match\'}');
		expect(linkBranch.includes('<QRPanel'), 'expected the link step to render QRPanel').toBe(true);
	});

	it('exactly two <Dialog usages exist in this file — the existing-instance modal and the two-step modal — no third Dialog was added for the link step', () => {
		const dialogOpens = stripped.match(/<Dialog\b/g) ?? [];
		expect(
			dialogOpens.length,
			'expected exactly two <Dialog usages (one-step existing-instance modal, two-step new-instance modal) — the link step must render inside the two-step modal, not a new one'
		).toBe(2);
	});

	it("QRPanel's onpaired callback advances to the match step, and oncancelled returns to connect with typed values intact", () => {
		const handleLinkPairedBody = extractBetween(stripped, 'function handleLinkPaired() {', '\n\t}');
		expect(handleLinkPairedBody.includes("step = 'match'")).toBe(true);

		const handleLinkCancelledBody = extractBetween(stripped, 'function handleLinkCancelled() {', '\n\t}');
		expect(handleLinkCancelledBody.includes("step = 'connect'")).toBe(true);
		expect(
			handleLinkCancelledBody.includes('connectionValues ='),
			'expected handleLinkCancelled to leave connectionValues untouched — cancelling the link must preserve every typed connection value'
		).toBe(false);
	});
});

describe('Step 1 failure branch: exact copy plus a Save anyway action', () => {
	it('renders the exact "Couldn\'t verify this connection." copy', () => {
		expect(
			stripped.includes("Couldn't verify this connection."),
			'expected the Step 1 failure branch to render the exact frozen copy'
		).toBe(true);
	});

	it('renders a "Save anyway" action', () => {
		expect(stripped.includes('Save anyway')).toBe(true);
	});

	it('the Save anyway action is gated on the describe-failed flag', () => {
		expect(
			/\{#if describeFailed\}/.test(stripped),
			'expected the Save anyway button to be gated on a describe-failed flag, so it never appears before Step 1 has actually failed'
		).toBe(true);
	});
});

describe('Step 2 submit path: exactly one putConfig call', () => {
	it('submitMatch calls putConfig exactly once', () => {
		const fnBody = extractBetween(
			stripped,
			'async function submitMatch(event: SubmitEvent) {',
			'\n\t}'
		);
		const calls = fnBody.match(/putConfig\(/g) ?? [];
		expect(
			calls.length,
			'expected submitMatch to call putConfig exactly once — two sequential saves could leave a configured-but-unparticipating instance if the second failed'
		).toBe(1);
	});
});

describe('saveAnyway: CR-01 regression — resolveNewInstanceId guards every write', () => {
	const fnBody = extractBetween(stripped, 'async function saveAnyway() {', '\n\t}');

	it('calls resolveNewInstanceId( before upsertSourceInstance(, with a return between them (CR-01)', () => {
		const guardIndex = fnBody.indexOf('resolveNewInstanceId(');
		const writeIndex = fnBody.indexOf('upsertSourceInstance(');
		expect(
			guardIndex,
			'expected saveAnyway to call resolveNewInstanceId( — 07-REVIEW.md CR-01 found this guard missing here'
		).toBeGreaterThanOrEqual(0);
		expect(
			writeIndex,
			'expected saveAnyway to call upsertSourceInstance( at all'
		).toBeGreaterThanOrEqual(0);
		expect(
			guardIndex,
			'expected the resolveNewInstanceId( guard to run strictly before upsertSourceInstance( — otherwise CR-01\'s unguarded overwrite is still reachable'
		).toBeLessThan(writeIndex);
		const between = fnBody.slice(guardIndex, writeIndex);
		expect(
			/\breturn\b/.test(between),
			'expected a return between resolveNewInstanceId( and upsertSourceInstance( so a not-ok result cannot fall through to the write (CR-01)'
		).toBe(true);
	});
});

// 07-06-PLAN.md Task 2: turns Task 1's fix into an invariant the suite
// enforces, so a third new-instance write path added later cannot
// reintroduce CR-01 silently.
describe('invariant: every new-instance write path routes through the one shared guard', () => {
	const saveAnywayBody = extractBetween(stripped, 'async function saveAnyway() {', '\n\t}');
	const submitMatchBody = extractBetween(
		stripped,
		'async function submitMatch(event: SubmitEvent) {',
		'\n\t}'
	);
	const handleConnectNextBody = extractBetween(
		stripped,
		'async function handleConnectNext(event: SubmitEvent) {',
		'\n\t}'
	);

	it('has zero occurrences of a local deriveInstanceId — derivation lives only in instance-id.ts, reached through resolveNewInstanceId', () => {
		expect(
			stripped.includes('deriveInstanceId'),
			'a local derivation reappearing in AddSourceModal.svelte is how CR-01 became possible — a second derivation site can drift from the guarded one'
		).toBe(false);
	});

	it('calls upsertSourceInstance( exactly twice in the whole file, once in saveAnyway and once in submitMatch', () => {
		const totalCalls = stripped.match(/upsertSourceInstance\(/g) ?? [];
		expect(
			totalCalls.length,
			'a third write path is not forbidden, but whoever adds it must extend this guard to cover it — that is the point of pinning the count'
		).toBe(2);

		const inSaveAnyway = saveAnywayBody.match(/upsertSourceInstance\(/g) ?? [];
		const inSubmitMatch = submitMatchBody.match(/upsertSourceInstance\(/g) ?? [];
		expect(inSaveAnyway.length, 'expected saveAnyway to call upsertSourceInstance( exactly once').toBe(1);
		expect(
			inSubmitMatch.length,
			'expected submitMatch to call upsertSourceInstance( exactly once'
		).toBe(1);
	});

	it('saveAnyway reaches upsertSourceInstance( only after a resolveNewInstanceId( call (restated inside the invariant block)', () => {
		const guardIndex = saveAnywayBody.indexOf('resolveNewInstanceId(');
		const writeIndex = saveAnywayBody.indexOf('upsertSourceInstance(');
		expect(guardIndex).toBeGreaterThanOrEqual(0);
		expect(writeIndex).toBeGreaterThan(guardIndex);
	});

	it('submitMatch passes newInstanceId rather than deriving anything locally', () => {
		expect(
			submitMatchBody.includes('upsertSourceInstance(config, newInstanceId, connectionValues)'),
			'expected submitMatch to write only the id the guard already approved (newInstanceId), never a freshly-derived one'
		).toBe(true);
	});

	it('the sole non-null assignment to newInstanceId sits inside handleConnectNext, after that body\'s own resolveNewInstanceId( call', () => {
		// Every "newInstanceId = <expr>;" statement in the file, excluding the
		// $state() declaration itself and the plain `null` reset in
		// resetFlowState — what remains is the one real, guard-approved write.
		const allStatements = stripped.match(/newInstanceId\s*=\s*[^;]+;/g) ?? [];
		const reassignments = allStatements.filter(
			(s) => !s.includes('$state') && !/=\s*null\s*;$/.test(s)
		);
		expect(
			reassignments.length,
			'expected exactly one assignment to newInstanceId that is not the $state() declaration or the null reset in resetFlowState — the only id submitMatch can write is one the guard already approved'
		).toBe(1);

		const guardIndex = handleConnectNextBody.indexOf('resolveNewInstanceId(');
		const assignIndex = handleConnectNextBody.indexOf(reassignments[0]);
		expect(guardIndex).toBeGreaterThanOrEqual(0);
		expect(
			assignIndex,
			'expected the real newInstanceId assignment to live inside handleConnectNext'
		).toBeGreaterThanOrEqual(0);
		expect(
			assignIndex,
			'expected the newInstanceId assignment to happen strictly after the resolveNewInstanceId( call in handleConnectNext'
		).toBeGreaterThan(guardIndex);
	});

	it('saveAnyway never clears the describe-failed flag — doing so on a rejected name would hide Save anyway and force a re-run of the failed connection test', () => {
		expect(
			saveAnywayBody.includes('describeFailed = false'),
			'clearing describeFailed inside saveAnyway would hide the Save anyway button on a rejected display name, forcing the user back through the failed connection test to retry'
		).toBe(false);
	});

	it('handleConnectNext still clears the describe-failed flag on its own validation rejection (the deliberate asymmetry)', () => {
		expect(
			handleConnectNextBody.includes('describeFailed = false'),
			'a validation rejection at Next is not a connection failure and must not reveal Save anyway'
		).toBe(true);
	});
});

// 07-13-PLAN.md Task 2, closing 07-UAT.md G-07-5: a blank required
// connection field must never reach a plugin subprocess. These assertions
// pin the missingRequiredFields( guard ahead of every request-issuing call
// it exists to stop, and pin the default-value seeding that removes the
// trap in the first place.
describe('G-07-5 guard: a blank required field never reaches describePlugin or putConfig', () => {
	const handleConnectNextBody = extractBetween(
		stripped,
		'async function handleConnectNext(event: SubmitEvent) {',
		'\n\t}'
	);
	const saveAnywayBody = extractBetween(stripped, 'async function saveAnyway() {', '\n\t}');

	it('handleConnectNext calls missingRequiredFields( strictly before describePlugin( — a blank mandatory field must never trigger a trial launch', () => {
		const guardIndex = handleConnectNextBody.indexOf('missingRequiredFields(');
		const describeIndex = handleConnectNextBody.indexOf('describePlugin(');
		expect(
			guardIndex,
			'expected handleConnectNext to call missingRequiredFields( — otherwise a blank mandatory field reaches describePlugin, which launches the plugin subprocess with it (07-UAT.md G-07-5)'
		).toBeGreaterThanOrEqual(0);
		expect(describeIndex, 'expected handleConnectNext to call describePlugin( at all').toBeGreaterThanOrEqual(
			0
		);
		expect(
			guardIndex,
			'expected missingRequiredFields( to run strictly before describePlugin( — otherwise the guard cannot stop the request it exists to stop'
		).toBeLessThan(describeIndex);
	});

	it('saveAnyway calls missingRequiredFields( strictly before putConfig( — Save anyway must not persist an instance the plugin can never launch', () => {
		const guardIndex = saveAnywayBody.indexOf('missingRequiredFields(');
		const putIndex = saveAnywayBody.indexOf('putConfig(');
		expect(
			guardIndex,
			'expected saveAnyway to call missingRequiredFields( — otherwise Save anyway can persist an instance missing a mandatory field, which then fails every subsequent hot-apply reconcile'
		).toBeGreaterThanOrEqual(0);
		expect(putIndex, 'expected saveAnyway to call putConfig( at all').toBeGreaterThanOrEqual(0);
		expect(
			guardIndex,
			'expected missingRequiredFields( to run strictly before putConfig( in saveAnyway'
		).toBeLessThan(putIndex);
	});

	it('plugin-type selection builds connectionValues from defaultConnectionValues( — a seeded default is a real editable value, not placeholder text', () => {
		expect(
			stripped.includes('defaultConnectionValues('),
			'expected the plugin-type selection to seed connectionValues from defaultConnectionValues( — the specific presentation (placeholder-only) that made leaving Signal\'s mandatory path untouched the natural action (07-13-PLAN.md G-07-5)'
		).toBe(true);
	});
});

describe('G-07-5 guard: ConnectionForm.svelte marks required fields with the DOM required attribute, not only an asterisk', () => {
	it('found non-empty comment-stripped ConnectionForm.svelte source', () => {
		expect(connectionFormStripped.length).toBeGreaterThan(0);
	});

	it('the non-secret Input carries a required attribute bound to the field descriptor', () => {
		const inputBlock = extractBetween(connectionFormStripped, '<Input', '/>');
		expect(
			/required=\{field\.required\}/.test(inputBlock),
			'expected ConnectionForm\'s plain-field Input to carry required={field.required} — the asterisk alone is decorative and provides no browser-native prevention or correct assistive-technology semantics'
		).toBe(true);
	});

	it('forwards required to SecretField for the secret branch', () => {
		const secretFieldBlock = extractBetween(connectionFormStripped, '<SecretField', '/>');
		expect(
			/required=\{field\.required\}/.test(secretFieldBlock),
			'expected ConnectionForm to forward required={field.required} into SecretField for the secret branch, so both branches mark a required field consistently'
		).toBe(true);
	});
});

describe('G-07-5 guard: EditSourceModal.svelte\'s Edit connection… guards submitConnection the same way', () => {
	it('found non-empty comment-stripped EditSourceModal.svelte source', () => {
		expect(editModalStripped.length).toBeGreaterThan(0);
	});

	it('submitConnection calls missingRequiredFields( strictly before putConfig( — blanking a required field here would persist an instance the plugin can never launch', () => {
		const fnBody = extractBetween(
			editModalStripped,
			'async function submitConnection(event: SubmitEvent) {',
			'\n\t}'
		);
		const guardIndex = fnBody.indexOf('missingRequiredFields(');
		const putIndex = fnBody.indexOf('putConfig(');
		expect(
			guardIndex,
			'expected submitConnection to call missingRequiredFields( — otherwise Edit connection… can persist an instance the plugin cannot start, failing every subsequent hot-apply reconcile'
		).toBeGreaterThanOrEqual(0);
		expect(putIndex, 'expected submitConnection to call putConfig( at all').toBeGreaterThanOrEqual(0);
		expect(
			guardIndex,
			'expected missingRequiredFields( to run strictly before putConfig( in submitConnection'
		).toBeLessThan(putIndex);
	});
});
