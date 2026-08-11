// Ports 07-UAT.md item 6 into a permanent regression gate.
//
// Verbatim pass condition this spec encodes (07-UAT.md item 6):
//   "Remove from this webspace: chip disappears immediately with no
//   reload, config.toml narrows correctly, other webspaces untouched.
//   "+" picker re-offers the removed instance; re-adding restores its
//   chip and items."
//
// Distinct from uat-02-remove-source-items.spec.ts (07.1-04): that spec
// covers a single-webspace remove/re-add and the background-sync-silence
// clauses G-07-7 adds. This spec's own value is the "other webspaces
// untouched" clause — untestable without a second webspace sharing an
// instance, so `beta` exists purely to make that clause assertable.
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { attachedWebspace, type FixtureConfigSpec, type FixtureSourceSpec } from '../fixtures/config-builder';
import { readConfigToml } from '../fixtures/toml';

const sources: FixtureSourceSpec[] = [
	{ id: 'mock-01', plugin: 'topos-plugin-mock', displayName: 'Mock One' },
	{ id: 'mock-02', plugin: 'topos-plugin-mock', displayName: 'Mock Two' }
];

// Both webspaces are seeded with an explicit `keywords: []` here (never
// left undefined) — matching what kernel/config's canonical writer ALWAYS
// emits for every webspace's Keywords field on ANY save, since it carries
// no `omitempty` tag (07.1-04-SUMMARY.md's own verified finding). Without
// this, the baseline capture below (read before any save has ever
// happened) would see beta's `keywords` key genuinely absent, and the
// FIRST save anywhere in the document — even one that never touches
// beta's participants — would then normalize it to a present `[]`, a
// byte-level diff that is the writer's own well-understood, accepted
// behavior, not the "other webspaces untouched" defect this spec exists
// to catch. Seeding it up front keeps the deep-equality baseline aligned
// with the writer's real steady state, so a failure here can only mean a
// genuine over-reach into beta's own participant data.
const configSpec: FixtureConfigSpec = {
	sources,
	webspaces: [
		{
			...attachedWebspace('alpha', ['mock-01', 'mock-02'], {
				'mock-01': { labels: ['demo'] },
				'mock-02': { labels: ['demo'] }
			}),
			keywords: []
		},
		{
			...attachedWebspace('beta', ['mock-01', 'mock-02'], {
				'mock-01': { labels: ['demo'] },
				'mock-02': { labels: ['demo'] }
			}),
			keywords: []
		}
	]
};

test.use({ configSpec });

interface ParsedConfig {
	webspaces: Record<
		string,
		{ sources: string[]; match: Record<string, Record<string, string[]>>; keywords?: string[] }
	>;
}

test.describe('07-UAT item 6: remove-chip round-trip, with other webspaces provably untouched', () => {
	test("removing and re-adding a chip on one webspace narrows/restores correctly, and never touches a sibling webspace sharing the same instance", async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, ['mock-01', 'mock-02'], { logs: kernel.logs });

		// --- Step 1: full parsed-config baseline before any action.
		const baseline = readConfigToml(kernel.configPath) as unknown as ParsedConfig;
		const betaBaseline = baseline.webspaces.beta;

		await page.goto(`${kernel.baseURL}/w/alpha`);
		await expect(page.getByRole('button', { name: 'Mock One actions' })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Mock Two actions' })).toBeVisible();

		// --- Step 2: remove Mock Two from alpha through its chip menu — the
		// chip disappears with no page reload.
		await page.getByRole('button', { name: 'Mock Two actions' }).click();
		await page.getByRole('menuitem', { name: 'Remove from this webspace' }).click();

		await expect(page.getByRole('button', { name: 'Mock Two actions' })).toHaveCount(0);
		await expect(page.getByRole('button', { name: 'Mock One actions' })).toBeVisible();

		// --- Steps 3/4: alpha's sources array names only mock-01, alpha's
		// match table has only a mock-01 key, in the exact shape
		// removeSourceFromWebspace actually writes (a present, narrowed
		// array — never an omitted key), and beta's entire table is deeply
		// equal to the baseline capture. Deep equality (not a spot check) is
		// the defect class this guards: a writer that rewrites more of the
		// document than it was asked to.
		const afterRemove = readConfigToml(kernel.configPath) as unknown as ParsedConfig;
		expect(afterRemove.webspaces.alpha.sources).toEqual(['mock-01']);
		expect(Object.keys(afterRemove.webspaces.alpha.match)).toEqual(['mock-01']);
		expect(afterRemove.webspaces.beta).toEqual(betaBaseline);

		// --- Step 5: the add-source picker on alpha re-offers Mock Two — the
		// picker excludes only current participants.
		await page.getByRole('button', { name: 'Add source' }).click();
		await expect(page.getByRole('button', { name: /Mock Two/ })).toBeVisible();

		// --- Step 6: re-add it, complete the match step, submit.
		await page.getByRole('button', { name: /Mock Two/ }).click();
		const matchDialog = page.getByRole('dialog');
		await expect(matchDialog.getByRole('heading', { name: 'Add Mock Two to alpha' })).toBeVisible();
		await matchDialog.getByLabel('Labels').fill('demo');
		await matchDialog.getByRole('button', { name: 'Add source' }).click();

		// --- Step 7: the chip returns and, once the resync lands, its items
		// are back in the stream — with no page reload at any point.
		await expect(page.getByRole('button', { name: 'Mock Two actions' })).toBeVisible();
		await expect(page.locator('[data-item-id^="mock-02:"]')).toHaveCount(4, { timeout: 15_000 });

		// --- Step 8: alpha is back to two participants, and beta is STILL
		// deeply equal to the baseline.
		const afterReadd = readConfigToml(kernel.configPath) as unknown as ParsedConfig;
		expect([...afterReadd.webspaces.alpha.sources].sort()).toEqual(['mock-01', 'mock-02']);
		expect(Object.keys(afterReadd.webspaces.alpha.match).sort()).toEqual(['mock-01', 'mock-02']);
		expect(afterReadd.webspaces.beta).toEqual(betaBaseline);

		// --- Step 9: navigate to beta and assert both chips render there
		// and its stream is unaffected — the on-disk equality proven in the
		// browser too, with no reload anywhere in this spec.
		await page.goto(`${kernel.baseURL}/w/beta`);
		await expect(page.getByRole('button', { name: 'Mock One actions' })).toBeVisible();
		await expect(page.getByRole('button', { name: 'Mock Two actions' })).toBeVisible();
		await expect(page.locator('[data-item-id]')).toHaveCount(8);
	});
});
