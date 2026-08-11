// Generalises uat-08-whatsapp-qr-link.spec.ts's own local
// offerWhatsAppPluginType route-interception idiom (quick task
// 260811-r5d) into a shared helper any spec can reach for.
//
// What this helper IS: an override of ONLY the catalog listing
// (GET /api/config/plugin-types), so the "+" picker's Group 2
// ("Install a new source") offers a plugin type the kernel deliberately
// does not — because kernel/pluginhost.ExcludedPluginBinaries (the
// unconditional, kernel-side exclusion) filters `binary` out of the real
// route's response.
//
// What this helper is NOT: every downstream call a spec makes after the
// picker tile is clicked — POST /api/config/describe-plugin, the real
// plugin subprocess launch, and PUT /api/config — still hits the real
// kernel and the real on-disk binary unmocked. DescribePluginHandler and
// config validation both read kernel/pluginhost.DiscoverAllBinaries,
// which the exclusion never touches, so a trial launch and a config write
// against the fixture plugin behave exactly as they would for any real
// plugin type.
//
// This helper exists for fixture plugins the kernel refuses to advertise
// (topos-plugin-mock, topos-plugin-mockstrict) — it is not a way to test
// a plugin type that is genuinely absent from disk; DescribePluginHandler
// would still 404 plugin_binary_not_found for a truly nonexistent binary
// name, injection or not.
import type { Page } from '@playwright/test';

export async function offerPluginType(page: Page, binary: string): Promise<void> {
	await page.route('**/api/config/plugin-types', async (route) => {
		const response = await route.fetch();
		const json = (await response.json()) as { plugin_types: string[] };
		json.plugin_types = [...json.plugin_types, binary].sort();
		await route.fulfill({ response, json });
	});
}
