// Ports 07-UAT.md item 10 into a permanent regression gate.
//
// Verbatim pass condition this spec encodes (07-UAT.md item 10):
//   "Open "Edit match settings…" on one chip, then before its vocabulary
//   loads, open an edit modal on a different chip: the modal never shows
//   or reverts to the first chip's vocabulary/open state — the second
//   click's state always wins."
//
// This item was carried through Phase 7's UAT as a non-blocking advisory
// (07-UAT.md test 10, "may skip") — never fixed, only deferred. Writing
// this spec against the shipped code found a real, reproducible failure:
// the SLOWER (first) click's describePlugin response, when it finally
// resolves, overwrote the already-open SECOND chip's vocabulary — because
// editVocabulary/editInstance/editMode are single shared $state values in
// web/src/routes/w/[webspace]/+page.svelte's handleChipEdit, and nothing
// checked whether a resolving response still belonged to the CURRENT edit
// session before writing it. Fixed inline in this plan (Rule 1 — see
// SUMMARY.md's Deviations section): handleChipEdit now discards a
// response whose own (name, kind) no longer matches editInstance/editMode
// by the time it resolves.
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { attachedWebspace, type FixtureConfigSpec, type FixtureSourceSpec } from '../fixtures/config-builder';

const sources: FixtureSourceSpec[] = [
	{ id: 'mock-01', plugin: 'topos-plugin-mock', displayName: 'Alpha Mock' },
	{
		id: 'strict-01',
		plugin: 'topos-plugin-mockstrict',
		path: '/tmp/topos-e2e-corpus',
		displayName: 'Beta Strict'
	}
];

const configSpec: FixtureConfigSpec = {
	sources,
	webspaces: [
		attachedWebspace('armor', ['mock-01', 'strict-01'], {
			'mock-01': { labels: ['demo'] },
			'strict-01': { tags: ['strict'] }
		})
	],
	pluginBinaries: ['topos-plugin-mock', 'topos-plugin-mockstrict']
};

test.use({ configSpec });

// How long the mock-backed chip's describe-plugin response is held before
// being allowed to continue — long enough that the test's own assertions
// (querying the DOM) reliably run before it resolves, without depending on
// exact timing.
const DELAY_MS = 1500;

test.describe('07-UAT item 10: chip-edit describePlugin race — the second click always wins', () => {
	test('the mockstrict chip wins immediately and stays won after the delayed mock response lands, with the mechanism proven to still work normally', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock-01', 'strict-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/armor`);

		// Registered BEFORE any triggering click — a handler registered
		// afterwards loses the race against the browser's own dispatch, and
		// this spec would then pass while proving nothing (the exact
		// vacuous shape this item's advisory status made tempting).
		// `route.fulfill` has no built-in delay parameter, so the delay is
		// an explicit awaited timer inside the handler. Identified by
		// inspecting the request's own posted body for the plugin name —
		// only the mock-backed chip's request is delayed; the mockstrict
		// chip's own request continues immediately.
		await page.route('**/api/config/describe-plugin', async (route) => {
			const body = route.request().postDataJSON() as { plugin: string };
			if (body.plugin === 'topos-plugin-mock') {
				await new Promise((resolve) => setTimeout(resolve, DELAY_MS));
			}
			await route.continue();
		});

		// --- Steps 1-3: choose "Edit match settings…" on the mock-backed
		// chip first — its describe request is now delayed.
		await page.getByRole('button', { name: 'Alpha Mock actions' }).click();
		await page.getByRole('menuitem', { name: 'Edit match settings…' }).click();

		// --- Step 4: before that resolves, choose "Edit match settings…"
		// on the mockstrict-backed chip.
		await page.getByRole('button', { name: 'Beta Strict actions' }).click();
		await page.getByRole('menuitem', { name: 'Edit match settings…' }).click();

		// --- Step 5: the modal that opens shows the mockstrict vocabulary
		// — a field labelled "Tags" — and no field labelled "Labels".
		const dialog = page.getByRole('dialog');
		await expect(dialog).toBeVisible();
		await expect(dialog.getByLabel('Tags')).toBeVisible();
		await expect(dialog.getByLabel('Labels')).toHaveCount(0);

		// --- Step 6: wait past the first request's delay and assert AGAIN
		// that the visible field is still "Tags" and still not "Labels" —
		// the second click's state must not merely win initially, it must
		// not be reverted when the slower response finally lands. This is
		// the item's real content; the first assertion alone would pass
		// against the very defect it guards.
		await page.waitForTimeout(DELAY_MS + 500);
		await expect(dialog.getByLabel('Tags')).toBeVisible();
		await expect(dialog.getByLabel('Labels')).toHaveCount(0);

		// --- Step 7: the modal was never observed carrying both
		// vocabularies at once (structurally impossible here — MatchFieldsForm
		// renders one field set for whatever `vocabulary` currently holds —
		// but pinned as an explicit assertion of the mutually-exclusive
		// outcome), and closing/reopening the mock-backed chip's edit modal
		// (with the delay removed) correctly shows "Labels" — proving the
		// spec is not passing because one side always wins unconditionally.
		await expect(dialog.getByLabel('Tags')).toBeVisible();
		await expect(dialog.getByLabel('Labels')).toHaveCount(0);

		await page.getByRole('button', { name: 'Cancel' }).click();
		await expect(dialog).toHaveCount(0);
		await page.unroute('**/api/config/describe-plugin');

		await page.getByRole('button', { name: 'Alpha Mock actions' }).click();
		await page.getByRole('menuitem', { name: 'Edit match settings…' }).click();
		await expect(dialog.getByLabel('Labels')).toBeVisible();
		await expect(dialog.getByLabel('Tags')).toHaveCount(0);
	});
});
