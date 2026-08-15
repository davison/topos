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
// A4/B4 (13-08-PLAN.md, G-13-1 gap closure): the fourth pair, added
// specifically for the empty-second-webspace reproduction — B4 is seeded
// with `keywords: []` (never omitted; see uat-06/uat-07's identical
// rationale for why an explicit empty array, not an absent field, is the
// established way this harness expresses a genuinely empty webspace) so
// it participates in zero of the mock source's items and renders the
// `Nothing here yet` empty-state copy, exactly like 13-UAT.md test 1's
// reported condition.
const WS_A4 = 'undo-nav-a4';
const WS_B4 = 'undo-nav-b4';

const configSpec: FixtureConfigSpec = {
	sources: [{ id: SOURCE_ID, plugin: 'topos-plugin-mock', displayName: 'Mock Source' }],
	webspaces: [
		...[WS_A1, WS_B1, WS_A2, WS_B2, WS_A3, WS_B3].map((name) => ({
			name,
			keywords: ['demo']
		})),
		{ name: WS_A4, keywords: ['demo'] },
		{ name: WS_B4, keywords: [] }
	]
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

// clickUndo (13-08-PLAN.md, G-13-1 gap closure): registers the wait for
// the reversal's mark POST BEFORE clicking Undo, so every assertion the
// caller makes AFTER this resolves is sequenced strictly after the
// response has reached the PAGE — the exact point after which onUndo's
// continuation (the `load(gen)` call whose pre-fix side effect this spec
// now pins) runs. Without this ordering, an absence-of-skeleton assertion
// could evaluate before the browser had even received its POST response
// and pass green against the unfixed source, which is the exact failure
// mode this whole plan exists to stop repeating. `markWebspace` is the
// webspace the reversal's mark write actually targets (the toast's own
// snapshotted `ws`, captured at mark time — WR-01) — NOT necessarily the
// webspace currently showing on screen.
async function clickUndo(
	page: import('@playwright/test').Page,
	markWebspace: string
): Promise<void> {
	const responsePromise = page.waitForResponse(
		(res) =>
			res.url().endsWith(`/api/webspaces/${markWebspace}/marks`) && res.request().method() === 'POST'
	);
	const undoButton = page.getByRole('button', { name: 'Undo', exact: true });
	await expect(undoButton).toBeVisible();
	await undoButton.click();
	await responsePromise;
}

// streamSkeletonLocator (13-08-PLAN.md, G-13-1 gap closure): a CSS
// selector by necessity, not by preference — the skeleton is decoration
// with no accessible name (StreamLoadingSkeleton.svelte), so its ABSENCE
// has no user-facing locator. `data-slot="skeleton"` is an attribute the
// shared shadcn Skeleton component already ships
// (web/src/lib/components/ui/skeleton/skeleton.svelte), never one added
// for this test — docs/testing.md's "Prefer user-facing locators" rule
// and 13-07-PLAN.md's standing prohibition are both about ADDING
// attributes to shipped components. Kept as the belt-and-braces companion
// to the primary getByText assertion below, never the sole gate.
function streamSkeletonLocator(page: import('@playwright/test').Page) {
	return page.locator('[data-slot="skeleton"].stream-row-surface');
}

// readStream fetches a webspace's stream directly from the kernel — the
// proof surface for webspace A's reversal in every test below. A's own
// stream is never re-fetched by the SPA after a cross-webspace navigation
// (nothing in +page.svelte re-issues a getStream call for a webspace that
// is no longer current), so there is no rendered A-side signal to assert
// on until the user navigates back — the kernel is A's proof surface for
// that reason, not because a stale-generation load() silently discards
// its own effects. Webspace B's rendered stream, by contrast, IS directly
// assertable: load()'s entry guard (web/src/routes/w/[webspace]/
// +page.svelte) makes a call with an already-stale generation a true
// no-op, so B's stream can never be driven into a loading state by the
// reversal — every test below now asserts on B's rendered rows (or, for
// the fourth test's genuinely empty B4, its rendered empty-state copy)
// directly, alongside this kernel-side read.
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

		// clickUndo's markWebspace is WS_A1 (the toast's own snapshotted
		// `ws` — exact: true, load-bearing, matters here too: the fixture's
		// own webspace names (undo-nav-*) contain "undo" as a
		// case-insensitive substring, so a non-exact name match against the
		// WebspaceSwitcher trigger button would ambiguously resolve to two
		// elements).
		await clickUndo(page, WS_A1);

		// A's own stream is never re-fetched by the SPA after this
		// cross-webspace navigation — poll the kernel directly instead of
		// asserting on rendered rows (see readStream's own doc comment
		// above for why A and B differ here).
		await expect
			.poll(
				async () => {
					const stream = await readStream(kernel.baseURL, WS_A1);
					return { hasItem: stream.items.some((it) => it.id === itemId), excludedCount: stream.excluded_count };
				},
				{ timeout: 10_000 }
			)
			.toEqual({ hasItem: true, excludedCount: 0 });

		// B's RENDERED stream, untouched by A's reversal (13-08-PLAN.md
		// Task 2, G-13-1): asserted against server truth — streamB.items
		// is nonzero here (unlike Task 1's empty B4, this proves the
		// assertion isn't vacuously passing on an empty list) — plus zero
		// stream skeletons. StreamList.svelte renders the skeleton branch
		// INSTEAD of any row, so a rendered row set at the expected count
		// is itself positive proof loadState never flipped to 'loading'.
		const streamB = await readStream(kernel.baseURL, WS_B1);
		expect(streamB.excluded_count).toBe(0);
		expect(streamB.items.length).toBeGreaterThan(0);
		await expect(page.locator('[data-item-id]')).toHaveCount(streamB.items.length);
		await expect(streamSkeletonLocator(page)).toHaveCount(0);
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

		await clickUndo(page, WS_A2);

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

		// B's RENDERED stream, untouched by A's reversal (13-08-PLAN.md
		// Task 2, G-13-1) — same discipline as the single-item test above.
		const streamB = await readStream(kernel.baseURL, WS_B2);
		expect(streamB.excluded_count).toBe(0);
		expect(streamB.items.length).toBeGreaterThan(0);
		await expect(page.locator('[data-item-id]')).toHaveCount(streamB.items.length);
		await expect(streamSkeletonLocator(page)).toHaveCount(0);
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

		await clickUndo(page, WS_A3);

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

		// B's RENDERED stream, untouched by A's reversal (13-08-PLAN.md
		// Task 2, G-13-1) — same discipline as the two tests above.
		const streamB = await readStream(kernel.baseURL, WS_B3);
		expect(streamB.excluded_count).toBe(0);
		expect(streamB.items.some((it) => it.id === itemId)).toBe(true);
		expect(streamB.items.length).toBeGreaterThan(0);
		await expect(page.locator('[data-item-id]')).toHaveCount(streamB.items.length);
		await expect(streamSkeletonLocator(page)).toHaveCount(0);
	});

	// 13-08-PLAN.md Task 1 (G-13-1 gap closure): the exact UAT reproduction
	// (13-UAT.md test 1) — clicking Undo after switching to an EMPTY
	// second webspace strands four permanent loading skeletons there. The
	// three tests above prove the reversal lands correctly (13-07's
	// scope); this test proves the navigated-to webspace's RENDERED stream
	// is left untouched by that reversal — the half 13-UAT.md's gap named
	// and the existing spec deliberately skipped.
	test('exclude in A4, switch to EMPTY B4, Undo — B4 renders no stranded skeleton (G-13-1)', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [SOURCE_ID], { logs: kernel.logs });

		// Precondition, asserted from the kernel BEFORE any UI work (never
		// in-window — the awaited span between Exclude and Undo below must
		// stay inside the toast's real 5000ms lifetime): B4 is genuinely
		// empty, the exact condition that made the injected skeleton rows
		// conspicuous in the reported UAT reproduction.
		const seedStreamB = await readStream(kernel.baseURL, WS_B4);
		expect(seedStreamB.items.length).toBe(0);

		await page.goto(`${kernel.baseURL}/w/${WS_A4}`);

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

		// Keep every awaited step between the Exclude click above and the
		// Undo click below to the minimum the file's existing comment
		// demands — the whole span must still fit inside the toast's real
		// 5000ms lifetime.
		await switchWebspace(page, WS_A4, WS_B4);

		// markWebspace is WS_A4 (the toast's own snapshotted `ws`), not the
		// webspace now on screen (WS_B4) — that is exactly WR-01's fix
		// this spec's first three tests already pin.
		await clickUndo(page, WS_A4);

		// B4's RENDERED stream — the assertion 13-UAT.md's gap named and
		// the existing three tests deliberately skipped. StreamList.svelte
		// checks the skeleton branch strictly ahead of every
		// response-derived branch, so the empty-state copy being visible
		// is itself positive proof `loadState !== 'loading'`.
		await expect(page.getByText('Nothing here yet')).toBeVisible();
		await expect(streamSkeletonLocator(page)).toHaveCount(0);

		// A4 from the kernel — the unchanged proof surface and unchanged
		// expectation: the item is listed again and excluded_count is 0.
		await expect
			.poll(
				async () => {
					const stream = await readStream(kernel.baseURL, WS_A4);
					return { hasItem: stream.items.some((it) => it.id === itemId), excludedCount: stream.excluded_count };
				},
				{ timeout: 10_000 }
			)
			.toEqual({ hasItem: true, excludedCount: 0 });

		// Navigate BACK to A4 and assert the restored item renders as a
		// row — the reversal is user-visible on return, not merely
		// kernel-true (KERN-09 Success Criterion 1's "trivially
		// reversible" contract, end to end through the browser).
		await switchWebspace(page, WS_B4, WS_A4);
		await expect(page.locator(`[data-item-id="${itemId}"]`)).toBeVisible();
	});
});
