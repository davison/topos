// The tracer slice (07.1-01-PLAN.md Task 2): one path through every layer
// this phase touches — Makefile build -> generated temp config -> spawned
// kernel binary -> embedded SPA -> Chromium -> stream assertion. Proves
// the whole harness architecture on one thin path before nine spec files
// are written on top of it.
import { test, expect } from '../fixtures/kernel';
import type { FixtureConfigSpec } from '../fixtures/config-builder';

const configSpec: FixtureConfigSpec = {
	sources: [{ id: 'mock', plugin: 'topos-plugin-mock' }],
	webspaces: [{ name: 'tracer', keywords: ['demo'] }]
};

test.use({ configSpec });

// Inline poll for the boot-time sync landing on the "mock" instance —
// "kernel is listening" and "the first sync landed" are two different
// events (kernel/syncer/scheduler.go fires each source's first refresh
// immediately at boot, so the race is real). Task 3 extracts this into
// fixtures/kernel.ts's exported `waitForFirstSync`, shared by every
// downstream spec.
async function waitForMockSyncToLand(baseURL: string): Promise<void> {
	const deadline = Date.now() + 30_000;
	while (Date.now() < deadline) {
		const res = await fetch(`${baseURL}/api/sources`);
		if (res.ok) {
			const body = (await res.json()) as {
				sources?: Array<{ name: string; syncing: boolean; last_status: string }>;
			};
			const mock = (body.sources ?? []).find((s) => s.name === 'mock');
			if (mock && mock.syncing === false && mock.last_status !== '') return;
		}
		await new Promise<void>((resolve) => setTimeout(resolve, 200));
	}
	throw new Error('tracer: "mock" source never reported a landed first sync within 30s');
}

test.describe('tracer: a hermetic kernel serves seeded mock items to a real browser', () => {
	test('the mock corpus renders in the tracer webspace stream', async ({ page, kernel }) => {
		await waitForMockSyncToLand(kernel.baseURL);

		await page.goto(`${kernel.baseURL}/w/tracer`);

		// The webspace switcher trigger renders the webspace name (D-10) —
		// this is the SPA's replacement for a static <h1> title.
		await expect(page.getByRole('button', { name: 'tracer' })).toBeVisible();

		// The mock plugin's fixed four-item corpus (plugins/mock/plugin.go)
		// all carry the "demo" label, so every one of them lands in this
		// webspace via the keywords fallback.
		await expect(page.getByText('Welcome to the mock source')).toBeVisible();
		await expect(page.getByText('Standup: Tuesday')).toBeVisible();
		await expect(page.getByText('Standup: Wednesday')).toBeVisible();
		await expect(page.getByText('Shopping list')).toBeVisible();
	});
});
