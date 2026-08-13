// Hermetic Playwright coverage of the in-app WhatsApp QR pairing flow
// (D-01/D-02/D-03, 08-04-PLAN.md Task 3) — Phase 07.1's standing rule
// (docs/testing.md) applied to this phase's own user-visible promise
// ("link WhatsApp by scanning a QR code").
//
// Hermeticity note (why the link routes and plugin discovery are
// intercepted rather than exercised against a real plugin): the harness's
// plugins directory is a closed set containing only the mock fixture
// binaries (docs/testing.md's "Harness architecture") — topos-plugin-
// whatsapp is deliberately never built by `make e2e`, exactly as the real
// paperless/signal/proton plugins are structurally excluded. So this spec
// seeds config through FixtureConfigSpec for its one real (mock) instance,
// and intercepts:
//   - GET /api/config/plugin-types, so the "+" picker offers "New
//     WhatsApp…" (the picker reads this route's own result set)
//   - POST /api/config/describe-plugin, because the real route would 404
//     plugin_binary_not_found for a binary the harness never built
//   - POST/GET/DELETE /api/config/whatsapp-link, the link session surface
//     itself — the mechanism under test
//   - GET /api/config and GET /api/sources, for the Re-link entry-point
//     case (case 7) only, so a WhatsApp-typed chip exists to click at
//     all — participatesIn() (web/src/lib/participation.ts) requires the
//     instance to be present in config.sources AND the webspace's
//     participation rules, not merely reported by GET /api/sources
//   - PUT /api/config, for case 8's final save ONLY, when (and only when)
//     the posted document names a topos-plugin-whatsapp source. The real
//     route would write the file successfully (config.Store.Save has no
//     plugin-existence check) but then fail apply (500 apply_failed):
//     Supervisor.Apply's Host.Reconcile hard-stats the plugin binary
//     before launching a brand-new instance, and this harness's plugins
//     directory is the same closed set named above — topos-plugin-
//     whatsapp is never in it. That failure is real kernel behaviour
//     against a genuinely un-launchable binary, not anything this plan's
//     own SPA code could fix, and it is an artifact of the harness's
//     structural exclusion, not something a real install (which DOES
//     ship the plugin binary) would ever hit. So this one case writes
//     the posted document to config.toml directly (the same smol-toml
//     writer every fixture helper already uses), proving the SPA sent
//     the correct structural document — the real defect surface this
//     case guards — without requiring a real, launchable WhatsApp binary
//     in the harness.
// Every other assertion (the poll loop, the countdown, the state
// transitions, config writes) proves the SPA's own real behaviour against
// these scripted responses, never a mock of the SPA itself.
import { dirname } from 'node:path';
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import {
	attachedWebspace,
	writeConfig,
	type FixtureConfigSpec,
	type FixtureSourceSpec
} from '../fixtures/config-builder';
import { readConfigToml } from '../fixtures/toml';
import type { Page } from '@playwright/test';

const sources: FixtureSourceSpec[] = [
	{ id: 'mock-01', plugin: 'topos-plugin-mock', displayName: 'Mock One' }
];

const configSpec: FixtureConfigSpec = {
	sources,
	webspaces: [attachedWebspace('armor', ['mock-01'], { 'mock-01': { labels: ['demo'] } })],
	pluginBinaries: ['topos-plugin-mock', 'topos-plugin-mockstrict']
};

test.use({ configSpec });

const WHATSAPP_PLUGIN = 'topos-plugin-whatsapp';
const SEEDED_PATH_DEFAULT = '~/.local/share/topos/whatsapp';

// --- Shared route-interception helpers -----------------------------------

/** Adds topos-plugin-whatsapp to the discovered plugin-types list, so the "+" picker offers "New WhatsApp…". */
async function offerWhatsAppPluginType(page: Page): Promise<void> {
	await page.route('**/api/config/plugin-types', async (route) => {
		const response = await route.fetch();
		const json = (await response.json()) as { plugin_types: string[] };
		json.plugin_types = [...json.plugin_types, WHATSAPP_PLUGIN].sort();
		await route.fulfill({ response, json });
	});
}

/**
 * Scripts a successful describe-plugin response for the WhatsApp trial
 * launch only — every other plugin's request continues unmocked. `opts`
 * is optional so every existing caller (cases 1-12) keeps its original,
 * always-succeeds behaviour unchanged; only case 13 (WR-01) passes
 * `failFromCall` to make the Nth and later WhatsApp describe calls fail.
 */
async function scriptDescribeWhatsApp(page: Page, opts?: { failFromCall: number }): Promise<void> {
	let callCount = 0;
	await page.route('**/api/config/describe-plugin', async (route) => {
		const body = route.request().postDataJSON() as { plugin: string };
		if (body.plugin !== WHATSAPP_PLUGIN) {
			await route.continue();
			return;
		}
		callCount++;
		if (opts && callCount >= opts.failFromCall) {
			await route.fulfill({
				status: 500,
				contentType: 'application/json',
				body: JSON.stringify({
					schema_version: 1,
					error: { code: 'internal_error', message: 'e2e-injected trial-launch failure' }
				})
			});
			return;
		}
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				schema_version: 1,
				source_type: 'whatsapp',
				plugin_display_name: 'WhatsApp',
				match_vocabulary: ['groups', 'contacts']
			})
		});
	});
}

interface LinkResponse {
	status: number;
	body: Record<string, unknown>;
}

interface LinkScript {
	start?: LinkResponse;
	// polls is consumed one entry per GET poll, in order; the last entry
	// repeats once exhausted (a session's terminal state stays terminal).
	polls: LinkResponse[];
	deleteCalls: number;
	// startDelayMs delays the POST route's fulfillment by this many
	// milliseconds before responding — used only by case 12 to create a
	// genuine in-flight window between the request firing and the
	// response resolving. Optional so every other case (1-11) fulfills
	// immediately, unchanged.
	startDelayMs?: number;
	// deletedSessionIds records the session id embedded in each DELETE
	// request's URL, percent-decoded exactly as the client sent it — so a
	// case can assert the id actually cancelled was the one the kernel
	// returned, never a hardcoded default. Optional so cases that only
	// care about the delete count (deleteCalls) need not populate it.
	deletedSessionIds?: string[];
	// pollCalls counts GET poll requests — used by case 12 to assert no
	// poll is ever issued for a session the panel has already abandoned.
	// Optional so cases that don't care leave it unset.
	pollCalls?: number;
}

/**
 * Scripts the three whatsapp-link routes against a fixed, in-order
 * response sequence. Registered as two route patterns (the bare route for
 * POST, and the {session} sub-path for GET/DELETE) — Playwright's `*` glob
 * segment never matches a `/`, so a single "whatsapp-link*" pattern cannot
 * cover both shapes.
 */
async function scriptLinkSession(page: Page, script: LinkScript): Promise<void> {
	await page.route('**/api/config/whatsapp-link', async (route) => {
		if (script.startDelayMs) {
			// The point of the delay (case 12 only): keeps the request
			// genuinely in flight long enough for the test to tear the
			// panel down before this route handler fulfills.
			await new Promise((resolve) => setTimeout(resolve, script.startDelayMs));
		}
		const resp =
			script.start ?? { status: 200, body: { schema_version: 1, session: 'sess-1', state: 'pending' } };
		await route.fulfill({ status: resp.status, contentType: 'application/json', body: JSON.stringify(resp.body) });
	});

	let pollIndex = 0;
	await page.route('**/api/config/whatsapp-link/*', async (route) => {
		if (route.request().method() === 'DELETE') {
			script.deleteCalls++;
			if (script.deletedSessionIds) {
				// The client percent-encodes the id into the DELETE path
				// (web/src/lib/api.ts's cancelWhatsAppLink) — decode the
				// last path segment to recover the exact id it sent.
				const url = route.request().url();
				const rawId = url.slice(url.lastIndexOf('/') + 1);
				script.deletedSessionIds.push(decodeURIComponent(rawId));
			}
			await route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify({ schema_version: 1, session: 'sess-1', state: 'cancelled' })
			});
			return;
		}
		if (script.pollCalls !== undefined) script.pollCalls++;
		const entry = script.polls[Math.min(pollIndex, script.polls.length - 1)];
		pollIndex++;
		await route.fulfill({
			status: entry.status,
			contentType: 'application/json',
			body: JSON.stringify(entry.body)
		});
	});
}

/** Injects a WhatsApp-typed source into GET /api/config and GET /api/sources — case 7's Re-link entry point only. */
async function injectWhatsAppInstance(
	page: Page,
	opts: { id: string; displayName: string; webspace: string }
): Promise<void> {
	await page.route('**/api/config', async (route) => {
		if (route.request().method() !== 'GET') {
			await route.continue();
			return;
		}
		const response = await route.fetch();
		const json = (await response.json()) as {
			config: {
				sources: Record<string, unknown>;
				webspaces: Record<string, { sources?: string[]; match?: Record<string, unknown> }>;
			};
		};
		json.config.sources[opts.id] = {
			plugin: WHATSAPP_PLUGIN,
			path: SEEDED_PATH_DEFAULT,
			display_name: opts.displayName,
			agent: { read: false, handoff: false }
		};
		const ws = json.config.webspaces[opts.webspace];
		ws.sources = [...(ws.sources ?? []), opts.id];
		ws.match = { ...(ws.match ?? {}), [opts.id]: { groups: ['test-group'] } };
		await route.fulfill({ response, json });
	});

	await page.route('**/api/sources', async (route) => {
		const response = await route.fetch();
		const json = (await response.json()) as {
			sources: Array<{ name: string }>;
		};
		json.sources = [
			...json.sources,
			{
				name: opts.id,
				source_type: 'whatsapp',
				display_name: opts.displayName,
				reachable: false,
				syncing: false,
				last_status: '',
				last_sync_unix: 0,
				last_error: ''
			}
		].sort((a, b) => a.name.localeCompare(b.name));
		await route.fulfill({ response, json });
	});
}

/**
 * Case 8 only: intercepts PUT /api/config, and — only for a posted
 * document naming a topos-plugin-whatsapp source — writes it to
 * config.toml directly via the fixture's own smol-toml writer, rather
 * than letting the request reach the real kernel (which would apply-fail
 * launching a plugin binary this harness structurally excludes; see this
 * file's own header comment). Every other PUT /api/config request
 * continues unmocked.
 */
async function interceptWhatsAppSaveOnly(page: Page, configPath: string): Promise<void> {
	await page.route('**/api/config', async (route) => {
		if (route.request().method() !== 'PUT') {
			await route.continue();
			return;
		}
		const body = route.request().postDataJSON() as {
			base_hash: string;
			config: { sources: Record<string, { plugin: string }> };
		};
		const hasWhatsApp = Object.values(body.config.sources).some((s) => s.plugin === WHATSAPP_PLUGIN);
		if (!hasWhatsApp) {
			await route.continue();
			return;
		}
		writeConfig(dirname(configPath), body.config);
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify({
				schema_version: 1,
				hash: 'e2e-faked-hash-after-whatsapp-save',
				config: body.config,
				env_vars: {},
				unknown_keys: []
			})
		});
	});
}

/** Opens the "+" picker, chooses "New WhatsApp…", and fills the required fields — every test starts the connect step this way. */
async function openWhatsAppConnectStep(page: Page, displayName: string): Promise<void> {
	await page.getByRole('button', { name: 'Add source' }).click();
	// 09-07-PLAN.md Fix 11 retired the "New {label}…" catalog-row copy —
	// the picker's Group 2 tile now reads just the plugin type label.
	await page.getByRole('button', { name: 'WhatsApp', exact: true }).click();
	const dialog = page.getByRole('dialog');
	await expect(dialog.getByRole('heading', { name: 'Connect WhatsApp' })).toBeVisible();
	await dialog.locator('#conn-display_name').fill(displayName);
	await expect(dialog.locator('#conn-path')).toHaveValue(SEEDED_PATH_DEFAULT);
	await dialog.getByRole('button', { name: 'Next' }).click();
}

test.describe('08-04: WhatsApp in-app QR pairing flow', () => {
	test('1. Populated QR: the panel shows the QR image with its accessible name, the instruction line, and a countdown line', async ({
		page,
		kernel
	}) => {
		await offerWhatsAppPluginType(page);
		await scriptDescribeWhatsApp(page);
		const qrResponse = {
			status: 200,
			body: {
				schema_version: 1,
				session: 'sess-1',
				state: 'qr',
				png_data_uri: 'data:image/png;base64,AAAA',
				expires_in_seconds: 30
			}
		};
		// start answers 'qr' directly — the panel's own 'pending' ->
		// first-poll round trip (a real, floored delay) is exercised by
		// case 2 below, deliberately; every other case starts already in
		// its target state so its assertions are not incidentally timing-
		// dependent on that floor.
		await scriptLinkSession(page, { start: qrResponse, polls: [qrResponse], deleteCalls: 0 });

		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/armor`);
		await openWhatsAppConnectStep(page, 'My WhatsApp');

		const dialog = page.getByRole('dialog');
		await expect(dialog.getByAltText(/pairing QR code/)).toBeVisible();
		await expect(dialog.getByAltText(/pairing QR code/)).toHaveAttribute(
			'src',
			'data:image/png;base64,AAAA'
		);
		await expect(dialog.getByText('Scan with your phone to link')).toBeVisible();
		await expect(dialog.getByText(/Refreshes in \d:\d{2}/)).toBeVisible();
	});

	test('2. Rotation: the poll route returning a second, different QR payload updates the image without closing the dialog or losing entered values', async ({
		page,
		kernel
	}) => {
		await offerWhatsAppPluginType(page);
		await scriptDescribeWhatsApp(page);
		// start answers the FIRST qr response directly; the one scripted
		// poll answers the SECOND, different qr response, delivered on
		// QRPanel's own fixed POLL_INTERVAL_MS cadence (not tied to
		// expires_in_seconds — G-08-1). This is the one case that
		// deliberately exercises that real cadence delay (a couple of
		// real seconds), since rotation IS the poll mechanism.
		await scriptLinkSession(page, {
			start: {
				status: 200,
				body: {
					schema_version: 1,
					session: 'sess-1',
					state: 'qr',
					png_data_uri: 'data:image/png;base64,FIRST',
					expires_in_seconds: 1
				}
			},
			polls: [
				{
					status: 200,
					body: {
						schema_version: 1,
						session: 'sess-1',
						state: 'qr',
						png_data_uri: 'data:image/png;base64,SECOND',
						expires_in_seconds: 30
					}
				}
			],
			deleteCalls: 0
		});

		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/armor`);
		await openWhatsAppConnectStep(page, 'My WhatsApp');

		const dialog = page.getByRole('dialog');
		await expect(dialog.getByAltText(/pairing QR code/)).toHaveAttribute(
			'src',
			'data:image/png;base64,FIRST'
		);
		// Driven by the scripted second response arriving, not a real-time
		// sleep — Playwright's own polling `expect` waits for the rotation.
		await expect(dialog.getByAltText(/pairing QR code/)).toHaveAttribute(
			'src',
			'data:image/png;base64,SECOND',
			{ timeout: 10_000 }
		);
		await expect(page.getByRole('dialog')).toHaveCount(1);
		await expect(dialog.locator('#conn-display_name')).toHaveValue('My WhatsApp');
	});

	test('3. Scan success: the Add-Source flow advances to the match step and the QR panel is gone', async ({
		page,
		kernel
	}) => {
		await offerWhatsAppPluginType(page);
		await scriptDescribeWhatsApp(page);
		const pairedResponse = { status: 200, body: { schema_version: 1, session: 'sess-1', state: 'paired' } };
		await scriptLinkSession(page, { start: pairedResponse, polls: [pairedResponse], deleteCalls: 0 });

		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/armor`);
		await openWhatsAppConnectStep(page, 'My WhatsApp');

		const dialog = page.getByRole('dialog');
		await expect(dialog.getByRole('heading', { name: 'Match settings for armor' })).toBeVisible();
		await expect(dialog.getByAltText(/pairing QR code/)).toHaveCount(0);
		await expect(dialog.getByLabel('Groups')).toBeVisible();
		await expect(dialog.getByLabel('Contacts')).toBeVisible();
	});

	test('4. Expiry: the panel shows the expired message and offers a restart, and the dialog stays open', async ({
		page,
		kernel
	}) => {
		await offerWhatsAppPluginType(page);
		await scriptDescribeWhatsApp(page);
		const timeoutResponse = { status: 200, body: { schema_version: 1, session: 'sess-1', state: 'timeout' } };
		await scriptLinkSession(page, { start: timeoutResponse, polls: [timeoutResponse], deleteCalls: 0 });

		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/armor`);
		await openWhatsAppConnectStep(page, 'My WhatsApp');

		const dialog = page.getByRole('dialog');
		await expect(
			dialog.getByText('This code expired — start again to get a new one.')
		).toBeVisible();
		await expect(dialog.getByRole('button', { name: 'Restart' })).toBeVisible();
		await expect(page.getByRole('dialog')).toHaveCount(1);
	});

	test('5. Error: the panel renders the kernel\'s own message through the destructive alert', async ({
		page,
		kernel
	}) => {
		await offerWhatsAppPluginType(page);
		await scriptDescribeWhatsApp(page);
		await scriptLinkSession(page, {
			start: {
				status: 200,
				body: {
					schema_version: 1,
					session: 'sess-1',
					state: 'error',
					code: 'whatsapp_store_in_use',
					message: 'Another link session already holds this store — close it first.'
				}
			},
			polls: [
				{
					status: 200,
					body: {
						schema_version: 1,
						session: 'sess-1',
						state: 'error',
						code: 'whatsapp_store_in_use',
						message: 'Another link session already holds this store — close it first.'
					}
				}
			],
			deleteCalls: 0
		});

		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/armor`);
		await openWhatsAppConnectStep(page, 'My WhatsApp');

		const dialog = page.getByRole('dialog');
		await expect(
			dialog.getByText('Another link session already holds this store — close it first.')
		).toBeVisible();
		await expect(dialog.getByRole('button', { name: 'Retry' })).toBeVisible();
	});

	test('6. Cancel releases the session: closing the dialog mid-session issues the cancel request', async ({
		page,
		kernel
	}) => {
		await offerWhatsAppPluginType(page);
		await scriptDescribeWhatsApp(page);
		const qrResponse = {
			status: 200,
			body: {
				schema_version: 1,
				session: 'sess-1',
				state: 'qr',
				png_data_uri: 'data:image/png;base64,AAAA',
				expires_in_seconds: 30
			}
		};
		const script: LinkScript = { start: qrResponse, polls: [qrResponse], deleteCalls: 0 };
		await scriptLinkSession(page, script);

		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/armor`);
		await openWhatsAppConnectStep(page, 'My WhatsApp');

		const dialog = page.getByRole('dialog');
		await expect(dialog.getByAltText(/pairing QR code/)).toBeVisible();

		// Close the dialog via Escape — QRPanel's own onDestroy fires,
		// which must cancel the session (T-08-10). Asserted on the
		// intercepted request having been made, not merely on the UI
		// closing.
		await page.keyboard.press('Escape');
		await expect(page.getByRole('dialog')).toHaveCount(0);
		await expect
			.poll(() => script.deleteCalls, { timeout: 5_000 })
			.toBeGreaterThanOrEqual(1);
	});

	test('7. Re-link entry point: a WhatsApp-typed instance offers Re-link…, opens the same panel, and a non-WhatsApp chip does not offer it', async ({
		page,
		kernel
	}) => {
		await injectWhatsAppInstance(page, { id: 'whatsapp-01', displayName: 'WhatsApp One', webspace: 'armor' });
		const relinkQrResponse = {
			status: 200,
			body: {
				schema_version: 1,
				session: 'sess-1',
				state: 'qr',
				png_data_uri: 'data:image/png;base64,RELINK',
				expires_in_seconds: 30
			}
		};
		await scriptLinkSession(page, { start: relinkQrResponse, polls: [relinkQrResponse], deleteCalls: 0 });

		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/armor`);

		// The non-WhatsApp chip's menu never offers Re-link….
		await expect(page.getByRole('button', { name: 'Mock One actions' })).toBeVisible();
		await page.getByRole('button', { name: 'Mock One actions' }).click();
		await expect(page.getByRole('menuitem', { name: 'Re-link…' })).toHaveCount(0);
		await page.keyboard.press('Escape');

		// The WhatsApp-typed chip's menu offers it, and opening it shows
		// the same QRPanel component.
		await expect(page.getByRole('button', { name: 'WhatsApp One actions' })).toBeVisible();
		await page.getByRole('button', { name: 'WhatsApp One actions' }).click();
		await page.getByRole('menuitem', { name: 'Re-link…' }).click();

		const dialog = page.getByRole('dialog');
		await expect(dialog.getByRole('heading', { name: 'Re-link WhatsApp One' })).toBeVisible();
		await expect(dialog.getByAltText(/pairing QR code/)).toBeVisible();
	});

	test('8. Not linked is not a failure (the E5 evidence): the flow reaches the QR step with no failure alert and no Save-anyway control, and cancelling out still reaches the match step and saves', async ({
		page,
		kernel
	}) => {
		await offerWhatsAppPluginType(page);
		await scriptDescribeWhatsApp(page);
		await interceptWhatsAppSaveOnly(page, kernel.configPath);
		const qrResponse = {
			status: 200,
			body: {
				schema_version: 1,
				session: 'sess-1',
				state: 'qr',
				png_data_uri: 'data:image/png;base64,AAAA',
				expires_in_seconds: 30
			}
		};
		await scriptLinkSession(page, { start: qrResponse, polls: [qrResponse], deleteCalls: 0 });

		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/armor`);
		await openWhatsAppConnectStep(page, 'Second WhatsApp');

		const dialog = page.getByRole('dialog');
		// The E5 evidence: reaching the link step is not a describe
		// failure — no failure alert, no Save anyway control.
		await expect(dialog.getByAltText(/pairing QR code/)).toBeVisible();
		await expect(dialog.getByText("Couldn't verify this connection.")).toHaveCount(0);
		await expect(dialog.getByRole('button', { name: 'Save anyway' })).toHaveCount(0);

		// Cancel out of the QR panel — returns to the connect step with
		// every typed value intact.
		await dialog.getByRole('button', { name: 'Cancel' }).click();
		await expect(dialog.getByRole('heading', { name: 'Connect WhatsApp' })).toBeVisible();
		await expect(dialog.locator('#conn-display_name')).toHaveValue('Second WhatsApp');

		// Clicking Next again — the opportunity was already offered and
		// declined once — reaches the match step directly this time,
		// without re-showing the panel.
		await dialog.getByRole('button', { name: 'Next' }).click();
		await expect(dialog.getByRole('heading', { name: 'Match settings for armor' })).toBeVisible();
		await expect(dialog.getByAltText(/pairing QR code/)).toHaveCount(0);

		await dialog.getByLabel('Groups').fill('demo-group');
		await dialog.getByRole('button', { name: 'Add source' }).click();
		await expect(page.getByRole('dialog')).toHaveCount(0);

		// Config outcome asserted through parsed TOML, never a text match
		// — the instance is saved, fully configured, and the health-dot
		// state (not this test's concern) resolves separately.
		const cfg = readConfigToml(kernel.configPath) as {
			sources: Record<string, { plugin: string; path?: string }>;
			webspaces: Record<
				string,
				{ sources: string[]; match: Record<string, Record<string, string[]>> }
			>;
		};
		const newSource = cfg.sources['second-whatsapp'];
		expect(newSource).toBeDefined();
		expect(newSource.plugin).toBe(WHATSAPP_PLUGIN);
		expect(newSource.path).toBe(SEEDED_PATH_DEFAULT);
		expect(cfg.webspaces.armor.sources).toContain('second-whatsapp');
		expect(cfg.webspaces.armor.match['second-whatsapp']).toEqual({ groups: ['demo-group'] });
	});

	test('9. qr then paired via the poll loop, at a realistic first-code expiry (08-UAT.md G-08-1)', async ({
		page,
		kernel
	}) => {
		await offerWhatsAppPluginType(page);
		await scriptDescribeWhatsApp(page);
		// 60 is the entire point of this case, not an arbitrary fixture
		// value: whatsmeow reports a 60-second validity window for its
		// FIRST code — the one a user actually scans (qrchan.go: `if
		// len(codes) == 6 { timeout = 60 * time.Second }`). Under the
		// pre-fix cadence (G-08-1) the panel's poll interval WAS this
		// value, so the kernel's already-recorded terminal event sat
		// undelivered for up to 60 seconds. QRPanel's fixed 2s
		// POLL_INTERVAL_MS (08-05) means this case still completes in
		// seconds despite the realistic 60-second code.
		const qrResponse = {
			status: 200,
			body: {
				schema_version: 1,
				session: 'sess-1',
				state: 'qr',
				png_data_uri: 'data:image/png;base64,AAAA',
				expires_in_seconds: 60
			}
		};
		const pairingAcceptedResponse = {
			status: 200,
			body: { schema_version: 1, session: 'sess-1', state: 'pairing_accepted' }
		};
		const pairedResponse = { status: 200, body: { schema_version: 1, session: 'sess-1', state: 'paired' } };
		await scriptLinkSession(page, {
			start: qrResponse,
			polls: [qrResponse, pairingAcceptedResponse, pairedResponse],
			deleteCalls: 0
		});

		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/armor`);
		await openWhatsAppConnectStep(page, 'Poll Loop WhatsApp');

		const dialog = page.getByRole('dialog');
		await expect(dialog.getByAltText(/pairing QR code/)).toBeVisible();

		// Explicit timeouts well under the scripted 60s expiry — 15s is
		// generous against a 2s poll cadence and three round trips, and a
		// cadence regression should make this case fail loudly (a
		// timeout), not pass slowly.
		await expect(dialog.getByText('Scan accepted — completing login…')).toBeVisible({
			timeout: 15_000
		});
		await expect(dialog.getByRole('heading', { name: 'Match settings for armor' })).toBeVisible({
			timeout: 15_000
		});
		await expect(dialog.getByAltText(/pairing QR code/)).toHaveCount(0);
	});

	test('10. no cancel affordance during the post-pair window (08-UAT.md G-08-1)', async ({
		page,
		kernel
	}) => {
		await offerWhatsAppPluginType(page);
		await scriptDescribeWhatsApp(page);
		const qrResponse = {
			status: 200,
			body: {
				schema_version: 1,
				session: 'sess-1',
				state: 'qr',
				png_data_uri: 'data:image/png;base64,AAAA',
				expires_in_seconds: 60
			}
		};
		const pairingAcceptedResponse = {
			status: 200,
			body: { schema_version: 1, session: 'sess-1', state: 'pairing_accepted' }
		};
		// No terminal entry — scriptLinkSession's poll helper repeats the
		// last scripted entry once exhausted, so the panel holds in the
		// pairing phase for the whole case.
		const script: LinkScript = {
			start: qrResponse,
			polls: [qrResponse, pairingAcceptedResponse],
			deleteCalls: 0
		};
		await scriptLinkSession(page, script);

		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/armor`);
		await openWhatsAppConnectStep(page, 'No Cancel WhatsApp');

		const dialog = page.getByRole('dialog');
		await expect(dialog.getByText('Scan accepted — completing login…')).toBeVisible({
			timeout: 15_000
		});
		await expect(dialog.getByAltText(/pairing QR code/)).toHaveCount(0);
		// Cancelling here SIGKILLs the link subprocess mid-login-handshake
		// and strands a pairing whatsmeow has already written to disk —
		// the control is deliberately withheld for these few seconds
		// (08-UI-SPEC.md Amendment 2's "No Cancel while pairing").
		await expect(dialog.getByRole('button', { name: 'Cancel' })).toHaveCount(0);
		await expect(page.getByRole('dialog')).toHaveCount(1);

		// Close via Escape so the panel's unmount cancel path runs exactly
		// as case 6 exercises it.
		await page.keyboard.press('Escape');
		await expect(page.getByRole('dialog')).toHaveCount(0);
		await expect
			.poll(() => script.deleteCalls, { timeout: 5_000 })
			.toBeGreaterThanOrEqual(1);
	});

	test('11. already-linked recovery: a completed pairing is picked up, not stranded (08-UAT.md G-08-1)', async ({
		page,
		kernel
	}) => {
		// Stands for: a user who scanned successfully on a previous
		// attempt and closed the dialog left a real device row in
		// whatsmeow.db; re-entering the Add-Source flow against the same
		// path now detects it and carries the flow forward instead of
		// offering a code for a device that is already paired.
		await offerWhatsAppPluginType(page);
		await scriptDescribeWhatsApp(page);
		const alreadyLinkedResponse = {
			status: 200,
			body: { schema_version: 1, session: 'sess-1', state: 'already_linked' }
		};
		const pairedResponse = { status: 200, body: { schema_version: 1, session: 'sess-1', state: 'paired' } };
		await scriptLinkSession(page, {
			start: alreadyLinkedResponse,
			polls: [alreadyLinkedResponse, pairedResponse],
			deleteCalls: 0
		});

		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/armor`);
		await openWhatsAppConnectStep(page, 'Already Linked WhatsApp');

		const dialog = page.getByRole('dialog');
		await expect(dialog.getByText('Already linked — confirming this session…')).toBeVisible({
			timeout: 15_000
		});
		// The absence of any QR image is the whole recovery claim — never
		// offer a code for a device that is already paired.
		await expect(dialog.getByAltText(/pairing QR code/)).toHaveCount(0);
		await expect(dialog.getByRole('heading', { name: 'Match settings for armor' })).toBeVisible({
			timeout: 15_000
		});
	});

	test('12. teardown during the in-flight start still releases the session (08-REVIEW.md CR-01)', async ({
		page,
		kernel
	}) => {
		await offerWhatsAppPluginType(page);
		await scriptDescribeWhatsApp(page);
		// sess-inflight is deliberately not the helper's default sess-1 —
		// asserting on the default would pass against a hardcoded cancel.
		const qrResponse = {
			status: 200,
			body: {
				schema_version: 1,
				session: 'sess-inflight',
				state: 'qr',
				png_data_uri: 'data:image/png;base64,INFLIGHT',
				expires_in_seconds: 30
			}
		};
		const script: LinkScript = {
			start: qrResponse,
			// The point of this case: with no delay the request completes
			// before Escape can land and the case proves nothing.
			startDelayMs: 1500,
			polls: [qrResponse],
			deleteCalls: 0,
			deletedSessionIds: [],
			pollCalls: 0
		};
		await scriptLinkSession(page, script);

		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/armor`);

		// Registered before opening the flow — registering it afterwards
		// races the request that has probably already fired, and would
		// hang.
		const startRequest = page.waitForRequest(
			(req) => req.url().includes('/api/config/whatsapp-link') && req.method() === 'POST'
		);
		await openWhatsAppConnectStep(page, 'In Flight WhatsApp');

		// Observed while the route handler is still sleeping — genuinely
		// in flight, the same window this case's diagnosis (CR-01)
		// describes: the panel is torn down while the start request is
		// still outstanding, so its unmount-time path finds no session id
		// to cancel; the response then arrives for a component that no
		// longer cares, and the subprocess the kernel already spawned
		// (plus, on Re-link, the instance it already suspended) is only
		// released if the panel cancels the id at that moment.
		await startRequest;
		await page.keyboard.press('Escape');
		await expect(page.getByRole('dialog')).toHaveCount(0);

		await expect
			.poll(() => script.deletedSessionIds!.length, { timeout: 5_000 })
			.toBeGreaterThan(0);
		expect(script.deletedSessionIds).toEqual(['sess-inflight']);
		// Exactly one, not "at least one" — at-least-one cannot
		// distinguish the fix from a double-cancel.
		expect(script.deleteCalls).toBe(1);
		expect(script.pollCalls).toBe(0);
	});

	test('13. a fresh trial-launch failure renders no stale declined-link notice (08-REVIEW.md WR-01)', async ({
		page,
		kernel
	}) => {
		await offerWhatsAppPluginType(page);
		// Phase 11 (D-15) added a silent, best-effort describe call at
		// plugin-TYPE SELECTION time (AddSourceModal.svelte's
		// selectPluginType/loadDeclaredExtras, purely to learn declared
		// extras keys) — this is call #1, and it always succeeds here.
		// Call #2 is openWhatsAppConnectStep's own real Next-click trial
		// launch (so the panel appears and can be declined); call #3 — the
		// fresh trial launch after declining — is the one this case makes
		// fail, simulating a genuine connection failure.
		await scriptDescribeWhatsApp(page, { failFromCall: 3 });
		const qrResponse = {
			status: 200,
			body: {
				schema_version: 1,
				session: 'sess-1',
				state: 'qr',
				png_data_uri: 'data:image/png;base64,AAAA',
				expires_in_seconds: 30
			}
		};
		await scriptLinkSession(page, { start: qrResponse, polls: [qrResponse], deleteCalls: 0 });

		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/armor`);
		await openWhatsAppConnectStep(page, 'Notice Then Fail WhatsApp');

		const dialog = page.getByRole('dialog');
		await expect(dialog.getByAltText(/pairing QR code/)).toBeVisible();

		// Decline the link opportunity — the neutral declined-link notice
		// appears on the connect step.
		await dialog.getByRole('button', { name: 'Cancel' }).click();
		await expect(dialog.getByRole('heading', { name: 'Connect WhatsApp' })).toBeVisible();
		await expect(
			dialog.getByText(
				'Not linked yet — you can save this source now and link later from its menu (Re-link…).'
			)
		).toBeVisible();

		// Click Next again — this second trial launch genuinely fails.
		await dialog.getByRole('button', { name: 'Next' }).click();

		// Both halves of the guarantee, at the same moment: the
		// destructive connection-failure alert is visible, and the
		// declined-link notice is gone — never both on screen at once. One
		// message saying the source can be saved and linked later while
		// another says the connection could not be verified is exactly the
		// user-visible contradiction this case forecloses.
		await expect(dialog.getByText("Couldn't verify this connection.")).toBeVisible();
		await expect(
			dialog.getByText(
				'Not linked yet — you can save this source now and link later from its menu (Re-link…).'
			)
		).toHaveCount(0);
	});

	test('14. (09-02) topos-branded pairing panel: the app icon decodes in a real browser and no third-party brand asset is fetched', async ({
		page,
		kernel
	}) => {
		// Clones case 1's exact setup — the populated-QR state is the one
		// browser-drivable surface 09-UI-SPEC.md Fix 10 requires proof
		// for (ROADMAP success criterion 5, the standing Phase 07.1
		// rule): a source scan cannot prove a served asset actually
		// resolves through the kernel's embedded SPA.
		await offerWhatsAppPluginType(page);
		await scriptDescribeWhatsApp(page);
		const qrResponse = {
			status: 200,
			body: {
				schema_version: 1,
				session: 'sess-1',
				state: 'qr',
				png_data_uri: 'data:image/png;base64,AAAA',
				expires_in_seconds: 30
			}
		};
		await scriptLinkSession(page, { start: qrResponse, polls: [qrResponse], deleteCalls: 0 });

		await waitForFirstSync(kernel.baseURL, ['mock-01'], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/armor`);
		await openWhatsAppConnectStep(page, 'Branded WhatsApp');

		const dialog = page.getByRole('dialog');
		await expect(dialog.getByAltText(/pairing QR code/)).toBeVisible();

		// Located by its own src, not by role or accessible name: the
		// app icon is decorative by design (alt=""), so it has no
		// accessible name — an accessible-name query would contradict
		// the markup this same fix writes.
		const appIcon = dialog.locator('img[src="/app-icon.png"]');
		await expect(appIcon).toBeVisible();
		// The standard 09-01's tracer spec set: an image is proven by
		// its own naturalWidth, never by the presence of an <img> tag
		// — the only assertion that separates a genuinely served asset
		// from a broken image icon.
		await expect
			.poll(async () => appIcon.evaluate((img) => (img as HTMLImageElement).naturalWidth))
			.toBeGreaterThan(0);

		// The negative half of the de-branding claim: every image the
		// panel renders in this state is either the session's own QR
		// data URI or the app icon — no third-party brand asset is
		// fetched.
		const imgSrcs = await dialog
			.locator('img')
			.evaluateAll((imgs) => imgs.map((img) => (img as HTMLImageElement).getAttribute('src')));
		expect(new Set(imgSrcs)).toEqual(new Set(['data:image/png;base64,AAAA', '/app-icon.png']));

		// The panel's rendered text carries no whatsmeow mention.
		await expect(dialog).not.toContainText(/whatsmeow/i);
	});
});
