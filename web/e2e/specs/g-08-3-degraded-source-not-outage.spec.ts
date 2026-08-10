// Ports 08-UAT.md test 3 into a permanent regression gate (G-08-3): a
// per-source sync failure must degrade that source, never fake a total
// outage.
//
// Verbatim expectation this spec encodes (08-UAT.md test 3):
//   "After linking a WhatsApp source, opening a webspace loads the
//   stream; if the whatsapp plugin connection is unavailable mid-restart,
//   the source degrades (health error surfaced) without failing the
//   whole webspace load."
//
// Interception strategy and why it is hermetic: this harness's plugin
// directory is a closed, cgo-free set (docs/testing.md, "Harness
// architecture") — none of the fixture plugins can be made to fail their
// own Match on demand, so there is no live path to a genuinely failed
// sync inside this harness. What IS testable, and what the reported
// defect actually is, is entirely how the SPA presents a stream response
// it is handed: a 200 body carrying `sync.status: 'error'` alongside
// either zero items or one. Each case below scripts that response (or
// aborts the request outright) at the route layer instead of provoking a
// real failing sync — deliberate, not a shortcut around an available real
// path. The kernel half of this gap (scoping the sync aggregate to a
// webspace's participating sources) is proven separately, against the
// real handlers, by the Go tests in kernel/httpapi/stream_test.go; this
// file proves only the SPA's presentation layer.
//
// Route interception is registered BEFORE navigation in every case, per
// this harness's own "register before the triggering action" discipline
// (docs/testing.md, "Writing a new spec") — see uat-04-zero-webspace-
// vs-outage.spec.ts for the closest analog (distinguishing "nothing
// configured" from "nothing answering") and why it does not stop the
// kernel to test a genuine outage; the abort-the-route technique here is
// the same one, applied one level lower (a single route, not the whole
// kernel process).
import { test, expect } from '../fixtures/kernel';
import { mockInstances, webspacesWithKeywords, type FixtureConfigSpec } from '../fixtures/config-builder';

const configSpec: FixtureConfigSpec = {
	sources: mockInstances(1),
	webspaces: webspacesWithKeywords(['g-08-3'], ['demo'])
};

test.use({ configSpec });

// A scripted error string that could not occur by accident — every
// assertion below checks for THIS text specifically, never merely "some
// error text is present," so the fix's own passing cannot be confused
// with a component that hardcodes a message.
const DISTINCTIVE_ERROR = 'mock-01: e2e-injected G-08-3 sync failure — connection refused';

const OUTAGE_SENTENCE = "The topos service didn't respond — check that it's running, then retry.";
const OUTAGE_TITLE = "Couldn't load this webspace";
const DEGRADED_TITLE = "A source couldn't sync";

function streamEnvelope(sync: { status: string; error: string }, items: unknown[]) {
	return {
		schema_version: 1,
		webspace: 'g-08-3',
		sync: { status: sync.status, finished_unix: 1785000000, error: sync.error },
		items
	};
}

test.describe('08-UAT test 3 (G-08-3): a per-source sync failure degrades its webspace instead of faking an outage', () => {
	test('a per-source sync failure with zero items degrades, and never claims the service is down', async ({
		page,
		kernel
	}) => {
		await page.route(`${kernel.baseURL}/api/webspaces/g-08-3/stream`, (route) =>
			route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify(streamEnvelope({ status: 'error', error: DISTINCTIVE_ERROR }, []))
			})
		);

		await page.goto(`${kernel.baseURL}/w/g-08-3`);

		// Wait for the alert region to settle before asserting its contents —
		// role="alert" is common to both StreamError and StreamSyncDegraded
		// (both reuse the same Alert primitive), so this wait is valid on
		// both sides of the fix and rules out a race against the initial
		// loading skeleton. Ordered so the pre-fix RED failure lands on the
		// sentence-presence assertion, not a timeout: the sync-failure branch
		// currently routes through StreamError, whose fixed outage sentence
		// is what a real user sees and what this fix exists to remove from
		// this path.
		await expect(page.getByRole('alert')).toBeVisible();
		await expect(page.getByText(OUTAGE_SENTENCE)).toHaveCount(0);
		await expect(page.getByText(DEGRADED_TITLE)).toBeVisible();
		await expect(page.getByText(DISTINCTIVE_ERROR)).toBeVisible();
	});

	test('a genuine fetch failure still renders the outage state, and never the degraded state', async ({
		page,
		kernel
	}) => {
		await page.route(`${kernel.baseURL}/api/webspaces/g-08-3/stream`, (route) => route.abort());

		await page.goto(`${kernel.baseURL}/w/g-08-3`);

		await expect(page.getByText(OUTAGE_TITLE)).toBeVisible();
		await expect(page.getByText(OUTAGE_SENTENCE)).toBeVisible();
		await expect(page.getByText(DEGRADED_TITLE)).toHaveCount(0);
	});

	test('a sync failure alongside items renders the stream, not either error state', async ({ page, kernel }) => {
		const item = {
			id: 'mock-01:g-08-3-adjacency',
			source: 'mock-01',
			source_type: 'mock',
			source_display_name: 'Mock 01',
			source_id: 'g-08-3-adjacency',
			title: 'G-08-3 adjacency item',
			preview: '',
			timestamp_unix: 1785000000,
			secondary_timestamp_unix: 0,
			labels: [],
			group_id: '',
			group_label: '',
			link: { url: 'https://example.lan/g-08-3-adjacency', fidelity: 'exact' },
			provenance: {}
		};
		await page.route(`${kernel.baseURL}/api/webspaces/g-08-3/stream`, (route) =>
			route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify(streamEnvelope({ status: 'error', error: DISTINCTIVE_ERROR }, [item]))
			})
		);

		await page.goto(`${kernel.baseURL}/w/g-08-3`);

		await expect(page.getByText('G-08-3 adjacency item')).toBeVisible();
		await expect(page.getByText(DEGRADED_TITLE)).toHaveCount(0);
		await expect(page.getByText(OUTAGE_SENTENCE)).toHaveCount(0);
	});
});
