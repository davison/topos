// 13-VERIFICATION.md gap-closure regression spec for 13-REVIEW.md WR-01: the
// exclude/include undo toast's reversal callback read the route's LIVE
// webspace binding at click time instead of the webspace the toast was
// CREATED in — so switching webspaces inside the toast's 5000ms window and
// then clicking the still-visible Undo silently wrote the reversal against
// the wrong webspace (a misdirected `remove` matched zero rows; a
// misdirected `add` created a real, user-invisible exclusion elsewhere).
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import type { FixtureConfigSpec } from '../fixtures/config-builder';

const SOURCE_ID = 'mock';

// Six webspaces, one pair per test — see the comment on switchWebspace
// below for why each test owns its own pair rather than sharing one: the
// `kernel` fixture is worker-scoped while `fullyParallel` splits a file
// across jobs, so tests in this file may or may not share a kernel
// process, and this spec's assertions are absolute (`excluded_count`
// equals an exact integer), not relative — a shared kernel must never be
// able to leak a mark from one test into another's assertion.
const WS_A1 = 'undo-nav-a1';
const WS_B1 = 'undo-nav-b1';
const WS_A2 = 'undo-nav-a2';
const WS_B2 = 'undo-nav-b2';
const WS_A3 = 'undo-nav-a3';
const WS_B3 = 'undo-nav-b3';

const configSpec: FixtureConfigSpec = {
	sources: [{ id: SOURCE_ID, plugin: 'topos-plugin-mock', displayName: 'Mock Source' }],
	webspaces: [WS_A1, WS_B1, WS_A2, WS_B2, WS_A3, WS_B3].map((name) => ({
		name,
		keywords: ['demo']
	}))
};

test.use({ configSpec });

// switchWebspace drives a real WebspaceSwitcher navigation from `from` to
// `to`. The final assertion — the trigger now reading `to` — is the
// load-bearing one: it proves page.params.webspace, and therefore the
// route's `webspace` $derived binding (the exact value WR-01's defect
// misread), has actually changed. A bare waitForURL would not prove that;
// the binding is what the defect actually misread, not the URL.
async function switchWebspace(page: import('@playwright/test').Page, from: string, to: string) {
	await page.getByRole('button', { name: from, exact: true }).click();
	await page.getByRole('menuitem', { name: to, exact: true }).click();
	await expect(page.getByRole('button', { name: to, exact: true })).toBeVisible();
}

// readStream fetches a webspace's stream directly from the kernel — the
// proof surface for every assertion in this file, because the SPA
// deliberately renders no visible signal for a reversal issued after a
// navigation (the snapshotted `gen` is stale post-navigation, so load(gen)
// no-ops by design).
async function readStream(
	baseURL: string,
	webspace: string
): Promise<{ items: Array<{ id: string }>; excluded_count: number }> {
	const res = await fetch(`${baseURL}/api/webspaces/${webspace}/stream`);
	expect(res.ok).toBe(true);
	return (await res.json()) as { items: Array<{ id: string }>; excluded_count: number };
}

test.describe('13-07 Task 1/2: the undo toast targets the webspace it was created in, not the one navigated to', () => {
	test('single-item exclude in A, switch to B, Undo — the exclusion is reversed in A, not B', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [SOURCE_ID], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WS_A1}`);

		const rows = page.locator('[data-item-id]');
		await expect(rows.first()).toBeVisible();
		const rowCountBefore = await rows.count();
		const itemId = await rows.first().getAttribute('data-item-id');
		expect(itemId).toBeTruthy();

		await rows.first().click();
		await page.getByRole('button', { name: 'Exclude from webspace' }).click();
		await expect(page.locator(`[data-item-id="${itemId}"]`)).toHaveCount(0);
		await expect(rows).toHaveCount(rowCountBefore - 1);
		// Copywriting Contract (13-UI-SPEC.md): singular, exact.
		await expect(page.getByText('Excluded 1 item', { exact: true })).toBeVisible();

		// The whole span between the exclude click above and the Undo click
		// below must fit inside the toast's real 5000ms lifetime — keep the
		// awaited steps to the minimum listed here, never lengthen the toast
		// duration to buy room (13-UI-SPEC.md E3.1 contract behaviour).
		await switchWebspace(page, WS_A1, WS_B1);

		// exact: true is load-bearing here, not stylistic: the fixture's own
		// webspace names (undo-nav-*) contain "undo" as a case-insensitive
		// substring, so a non-exact name match against the WebspaceSwitcher
		// trigger button (accessible name = the current webspace name) would
		// ambiguously resolve to two elements.
		const undoButton = page.getByRole('button', { name: 'Undo', exact: true });
		await expect(undoButton).toBeVisible();
		await undoButton.click();

		// The SPA renders no signal for this Undo (the snapshotted `gen` is
		// stale post-navigation, so load(gen) no-ops by design) — poll the
		// kernel directly instead of asserting on rendered rows.
		await expect
			.poll(
				async () => {
					const stream = await readStream(kernel.baseURL, WS_A1);
					return { hasItem: stream.items.some((it) => it.id === itemId), excludedCount: stream.excluded_count };
				},
				{ timeout: 10_000 }
			)
			.toEqual({ hasItem: true, excludedCount: 0 });

		const streamB = await readStream(kernel.baseURL, WS_B1);
		expect(streamB.excluded_count).toBe(0);
	});

	test('bulk exclude in A, switch to B, Undo — every excluded id is restored in A', async ({ page, kernel }) => {
		await waitForFirstSync(kernel.baseURL, [SOURCE_ID], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WS_A2}`);

		const rows = page.locator('[data-item-id]');
		await expect(rows.first()).toBeVisible();

		await rows.nth(0).click({ modifiers: ['Control'] });
		await rows.nth(1).click({ modifiers: ['Control'] });
		await expect(page.getByText('2 selected', { exact: true })).toBeVisible();

		const excludedIds = (await Promise.all([
			rows.nth(0).getAttribute('data-item-id'),
			rows.nth(1).getAttribute('data-item-id')
		])) as string[];

		await page.getByRole('button', { name: 'Exclude', exact: true }).click();
		await expect(page.getByText('Excluded 2 items', { exact: true })).toBeVisible();
		for (const id of excludedIds) {
			await expect(page.locator(`[data-item-id="${id}"]`)).toHaveCount(0);
		}

		await switchWebspace(page, WS_A2, WS_B2);

		const undoButton = page.getByRole('button', { name: 'Undo', exact: true });
		await expect(undoButton).toBeVisible();
		await undoButton.click();

		await expect
			.poll(
				async () => {
					const stream = await readStream(kernel.baseURL, WS_A2);
					const idsPresent = new Set(stream.items.map((it) => it.id));
					return {
						allPresent: excludedIds.every((id) => idsPresent.has(id)),
						excludedCount: stream.excluded_count
					};
				},
				{ timeout: 10_000 }
			)
			.toEqual({ allPresent: true, excludedCount: 0 });

		const streamB = await readStream(kernel.baseURL, WS_B2);
		expect(streamB.excluded_count).toBe(0);
	});

	test('detail-pane include in A, switch to B, Undo — A is re-excluded and B gains no mark', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [SOURCE_ID], { logs: kernel.logs });

		// Seed the precondition through the API rather than the UI (docs/
		// testing.md: "Seed state through the fixture, not the UI") — this
		// also keeps exactly one toast alive at a time, so the Undo locator
		// can never hit two toasts at once. This is the same write the UI's
		// Exclude button issues, taken directly so the test's subject is
		// the include-then-undo path and nothing else.
		const seedStream = await readStream(kernel.baseURL, WS_A3);
		const itemId = seedStream.items[0].id;
		const seedRes = await fetch(`${kernel.baseURL}/api/webspaces/${WS_A3}/marks`, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ kind: 'excluded', action: 'add', item_ids: [itemId] })
		});
		expect(seedRes.ok).toBe(true);

		await page.goto(`${kernel.baseURL}/w/${WS_A3}`);

		await page.getByRole('button', { name: 'Excluded (1)', exact: true }).click();
		const rows = page.locator('[data-item-id]');
		await expect(rows).toHaveCount(1);
		await expect(page.locator(`[data-item-id="${itemId}"]`)).toBeVisible();

		await rows.first().click();
		await page.getByRole('button', { name: 'Include in webspace' }).click();
		await expect(page.getByText('Included 1 item', { exact: true })).toBeVisible();

		await switchWebspace(page, WS_A3, WS_B3);

		const undoButton = page.getByRole('button', { name: 'Undo', exact: true });
		await expect(undoButton).toBeVisible();
		await undoButton.click();

		// The sharper half of this test: pre-fix, this exact sequence wrote
		// a real, user-invisible exclusion into a webspace the user had
		// merely navigated to (a misdirected `add`, the corrupting
		// direction — it does not merely no-op like a misdirected
		// `remove`).
		await expect
			.poll(
				async () => {
					const stream = await readStream(kernel.baseURL, WS_A3);
					return {
						hasItem: stream.items.some((it) => it.id === itemId),
						excludedCount: stream.excluded_count
					};
				},
				{ timeout: 10_000 }
			)
			.toEqual({ hasItem: false, excludedCount: 1 });

		const streamB = await readStream(kernel.baseURL, WS_B3);
		expect(streamB.excluded_count).toBe(0);
		expect(streamB.items.some((it) => it.id === itemId)).toBe(true);
	});
});
