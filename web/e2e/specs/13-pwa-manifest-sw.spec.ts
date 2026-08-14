// 13-04-PLAN.md Task 2: browser proof for UI-13's served-surface
// requirements, against a real built-and-embedded kernel — per the
// standing Phase 07.1 rule (docs/testing.md "## Standing rule") that any
// UAT item a browser can drive becomes a permanent spec, not a manual
// check.
//
// Covers, per the plan's <behavior> block:
//   1. GET /manifest.webmanifest returns 200 with a Content-Type header
//      whose media type is application/manifest+json (the Go-side half of
//      13-RESEARCH.md Pitfall 4 — cmd/topos/mime_test.go proves the
//      registration call itself; this proves it takes effect on the real
//      HTTP response).
//   2. The served document carries a manifest <link> element.
//   3. A ServiceWorker registers successfully in the page, scoped to the
//      site root.
//   4. After a page load and a stream fetch, every entry in the browser's
//      Cache Storage has a request URL outside the /api/ path prefix — the
//      behavioural proof that no API response is ever precached or
//      runtime-cached (13-RESEARCH.md Pitfall 5).
//
// Zero arbitrary waits (D-15's zero-retries policy, per the plan's own
// instruction): every assertion below either polls a real condition
// (waitForFirstSync) or awaits a native promise with no fixed timeout
// (navigator.serviceWorker.ready).
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import type { FixtureConfigSpec } from '../fixtures/config-builder';

const WEBSPACE = 'pwa-shell';

const configSpec: FixtureConfigSpec = {
	sources: [{ id: 'mock', plugin: 'topos-plugin-mock' }],
	webspaces: [{ name: WEBSPACE, keywords: ['demo'] }]
};

test.use({ configSpec });

test.describe('13-04: PWA manifest, service-worker registration, and API-free Cache Storage', () => {
	test('GET /manifest.webmanifest returns 200 with Content-Type application/manifest+json', async ({
		request,
		kernel
	}) => {
		const res = await request.get(`${kernel.baseURL}/manifest.webmanifest`);
		expect(res.status()).toBe(200);

		const contentType = res.headers()['content-type'] ?? '';
		// Compare only the media type (strip any charset parameter) — the
		// exact header string isn't the contract, the media type browsers
		// read for installability decisions is (cmd/topos/mime_test.go pins
		// the Go-side registration this response header depends on).
		const mediaType = contentType.split(';')[0].trim();
		expect(mediaType).toBe('application/manifest+json');
	});

	test('the served document carries a manifest link element', async ({ page, kernel }) => {
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const manifestHref = await page.locator('link[rel="manifest"]').getAttribute('href');
		expect(manifestHref, 'expected the document to carry a <link rel="manifest">').not.toBeNull();

		const resolvedPathname = new URL(manifestHref!, kernel.baseURL).pathname;
		expect(resolvedPathname).toBe('/manifest.webmanifest');
	});

	test('a ServiceWorker registers successfully, scoped to the site root', async ({ page, kernel }) => {
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		// navigator.serviceWorker.ready resolves once an active worker is
		// associated with this page's scope — a real condition-based await,
		// not a fixed-duration wait.
		const scope = await page.evaluate(async () => {
			const registration = await navigator.serviceWorker.ready;
			return registration.scope;
		});

		expect(new URL(scope).pathname).toBe('/');
	});

	test('a window focus event triggers a ServiceWorker update check (checkpoint defect 1)', async ({
		page,
		kernel
	}) => {
		// Patches ServiceWorkerRegistration.prototype.update BEFORE any
		// navigation/registration happens, so the count below reflects
		// every call scheduleUpdateChecks (web/src/lib/pwa-update.ts)
		// makes, including the eventual periodic one it schedules. This
		// proves the checkpoint-B defect 1 fix wires a REAL update check
		// to a REAL window event, in a REAL browser — the unit tests in
		// pwa-update.test.ts already prove the wiring logic itself
		// against fakes; this proves the fakes match reality.
		await page.addInitScript(() => {
			(window as unknown as { __updateCalls: number }).__updateCalls = 0;
			const originalUpdate = ServiceWorkerRegistration.prototype.update;
			ServiceWorkerRegistration.prototype.update = function (
				this: ServiceWorkerRegistration,
				...args: []
			) {
				(window as unknown as { __updateCalls: number }).__updateCalls++;
				return originalUpdate.apply(this, args);
			};
		});

		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);
		await page.evaluate(async () => {
			await navigator.serviceWorker.ready;
		});

		const before = await page.evaluate(
			() => (window as unknown as { __updateCalls: number }).__updateCalls
		);

		await page.evaluate(() => window.dispatchEvent(new Event('focus')));

		await expect
			.poll(async () => page.evaluate(() => (window as unknown as { __updateCalls: number }).__updateCalls))
			.toBeGreaterThan(before);
	});

	test('after a page load and a stream fetch, no Cache Storage entry has a /api/ request URL', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock'], { logs: kernel.logs });

		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		// Confirm the stream's own /api fetch actually happened (proving this
		// assertion isn't vacuously true because nothing hit the API yet),
		// then wait for the ServiceWorker to be active — the SW's install-time
		// precache write is what this test is proving does NOT contain any
		// /api/ entry.
		await page.waitForResponse((res) => new URL(res.url()).pathname.startsWith('/api/') && res.status() === 200);
		await page.evaluate(async () => {
			await navigator.serviceWorker.ready;
		});

		const cachedApiUrls = await page.evaluate(async () => {
			const cacheNames = await caches.keys();
			const urls: string[] = [];
			for (const name of cacheNames) {
				const cache = await caches.open(name);
				const requests = await cache.keys();
				for (const req of requests) {
					if (new URL(req.url).pathname.startsWith('/api/')) urls.push(req.url);
				}
			}
			return urls;
		});

		expect(
			cachedApiUrls,
			`expected zero Cache Storage entries under /api/, found: ${JSON.stringify(cachedApiUrls)}`
		).toEqual([]);
	});

	test('a toast renders with a genuinely high-contrast background against the page (checkpoint defect 2, round 2)', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		// Fires the shared markSuccessToast helper — the same untyped
		// toast() call and --normal-* styling pwaUpdatedToast() uses, so
		// this exercises the exact surface the checkpoint's "close to
		// invisible" / "background itself needs contrast" reports were
		// about, without needing a real kernel upgrade to trigger the
		// PWA-specific toast.
		const rows = page.locator('[data-item-id]');
		await expect(rows.first()).toBeVisible();
		await rows.first().click();
		await page.getByRole('button', { name: 'Exclude from webspace' }).click();

		const toast = page.locator('[data-sonner-toast]').first();
		await expect(toast).toBeVisible();

		const styles = await toast.evaluate((el) => {
			const cs = getComputedStyle(el);
			return { background: cs.backgroundColor, borderColor: cs.borderColor, boxShadow: cs.boxShadow };
		});
		const bodyBg = await page.evaluate(() => getComputedStyle(document.body).backgroundColor);

		expect(
			styles.borderColor,
			'expected a distinct border color from the toast fill — a border indistinguishable from its own background gives no visible edge'
		).not.toBe(styles.background);
		expect(
			styles.boxShadow,
			'expected a real box-shadow, not the "none" a fully-transparent/absent shadow would report'
		).not.toBe('none');

		// Round 1 (a same-family dark surface, one --popover step lighter
		// than the page) only asserted "not equal" — technically distinct,
		// but re-verification reported it still read as "close to
		// invisible". Round 2 reuses SourceChip's own tooltip treatment
		// (tooltip-content.svelte's bg-foreground/text-background full
		// light/dark inversion) specifically because it is the one
		// chip-adjacent floating surface that is NOT just another step of
		// the same dark staircase — so this asserts the STRONGER claim a
		// mere inequality can't: a real WCAG-style luminance gap, not just
		// a different RGB triple.
		function relativeLuminance(rgb: string): number {
			const match = rgb.match(/\d+(\.\d+)?/g);
			if (!match) throw new Error(`could not parse computed color: ${rgb}`);
			const [r, g, b] = match.slice(0, 3).map(Number);
			const toLinear = (c: number) => {
				const s = c / 255;
				return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4);
			};
			return 0.2126 * toLinear(r) + 0.7152 * toLinear(g) + 0.0722 * toLinear(b);
		}

		const toastLuminance = relativeLuminance(styles.background);
		const pageLuminance = relativeLuminance(bodyBg);
		const lighter = Math.max(toastLuminance, pageLuminance) + 0.05;
		const darker = Math.min(toastLuminance, pageLuminance) + 0.05;
		const contrastRatio = lighter / darker;

		expect(
			contrastRatio,
			`expected a WCAG-AA-grade contrast ratio (>= 4.5) between the toast background (${styles.background}, luminance ${toastLuminance.toFixed(3)}) and the page background (${bodyBg}, luminance ${pageLuminance.toFixed(3)}) — got ${contrastRatio.toFixed(2)}`
		).toBeGreaterThanOrEqual(4.5);
	});
});
