// 12-11-PLAN.md Task 2: the browser-level proof for CR-01 and its
// anti-dead-code guard — the two states this harness cannot produce live.
//
// Why route interception is the right tool here, not a shortcut: this
// harness has no live path to a source that is SIMULTANEOUSLY unreachable
// (a failed live probe) and carrying both a stale successful sync run and a
// leftover last_notice — the live probe and the recorded run would have to
// disagree, which no fixture plugin in this harness's closed, cgo-free set
// (docs/testing.md, "Harness architecture") can be made to do on demand.
// What IS testable, and what this spec proves, is entirely how the SPA
// presents a GET /api/sources body it is handed — the identical strategy
// g-08-3-degraded-source-not-outage.spec.ts documents and uses for the
// kernel-side sync-status/outage distinction, applied here to the
// SourceChip tooltip's own gate. 12-zero-match-diagnostic.spec.ts is the
// real-kernel counterpart: it drives an actual topos-plugin-filesystem
// subprocess end to end to prove the advisory's OWN path (a healthy,
// reachable source whose match block matched nothing), and deliberately
// stays out of the unreachable-while-advisory combination this spec
// exists for.
//
// Out of scope by design: a real unmounted NFS/SMB share going away under
// a live probe while a stale successful run and a leftover notice are
// still on record is human-verified in UAT, never asserted here
// (docs/testing.md scopes real mounts out of this harness).
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { mockInstances, type FixtureConfigSpec, type FixtureWebspaceSpec } from '../fixtures/config-builder';

const WEBSPACE = 'tooltip-precedence';
const INSTANCE_ID = 'mock-01';
const DISPLAY_NAME = 'Mock 01';

// Both an explicit `sources` allowlist naming the instance AND a non-empty
// `keywords` list are required for participatingInstances (web/src/lib/
// participation.ts) to admit the instance — either alone is not enough for
// a webspace with an explicit sources allowlist, since the allowlist gate
// and the match-input gate are two separate checks the chip's own render
// condition depends on.
const webspace: FixtureWebspaceSpec = {
	name: WEBSPACE,
	sources: [INSTANCE_ID],
	keywords: ['demo']
};

const configSpec: FixtureConfigSpec = {
	sources: mockInstances(1),
	webspaces: [webspace]
};

test.use({ configSpec });

// A distinctive notice that could not appear by accident — the g-08-3
// discipline, so a passing assertion cannot be confused with a component
// that hardcodes copy. Shaped like a real kernel-composed advisory: it
// names the webspace and a glob-shaped match value.
const DISTINCTIVE_NOTICE = `webspace '${WEBSPACE}': match value 'mock-01.folders=*' matched no items`;

// 14-02-PLAN.md Task 3 (option-b): escapes a literal substring for use
// inside a RegExp passed to toHaveAccessibleDescription — mirrors
// 12-zero-match-diagnostic.spec.ts's identical inline escape.
function escapeRegExp(literal: string): string {
	return literal.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}

interface FabricatedSourcesBodyOpts {
	reachable: boolean;
	last_status: 'ok';
	last_notice: string;
}

// A real-shaped GET /api/sources body (docs/api.md's own documented
// shape), modelled on 09-1-compact-cards.spec.ts's degradedSourcesBody().
// Identity fields mirror the fixture instance exactly; reachable/
// last_status/last_notice come from the caller so the same helper serves
// both tests below.
function fabricatedSourcesBody(opts: FabricatedSourcesBodyOpts) {
	return {
		schema_version: 1,
		sources: [
			{
				name: INSTANCE_ID,
				source_type: 'mock',
				display_name: DISPLAY_NAME,
				plugin: 'topos-plugin-mock',
				tier: 'trusted',
				reachable: opts.reachable,
				syncing: false,
				last_status: opts.last_status,
				last_sync_unix: 1785000000,
				last_error: '',
				last_notice: opts.last_notice
			}
		]
	};
}

// 14-02-PLAN.md Task 2 (14-UI-SPEC.md G1, option-b): the chip's health
// sentence no longer renders through a native `title` attribute — it is
// exposed as the button's accessible DESCRIPTION via a visually-hidden
// sr-only span wired through aria-describedby. `toHaveAccessibleDescription`
// reads that computed description the same way assistive tech would,
// making it the correct replacement for the old title-attribute assertions
// below.
test.describe('12-11 Task 2: the rendered accessible description proves the tooltip gate\'s precedence (CR-01)', () => {
	test('A — reachable: false + last_status ok + a leftover notice: the title says unreachable and carries NONE of the notice text', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [INSTANCE_ID], { logs: kernel.logs });

		await page.route(`${kernel.baseURL}/api/sources`, (route) =>
			route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify(
					fabricatedSourcesBody({ reachable: false, last_status: 'ok', last_notice: DISTINCTIVE_NOTICE })
				)
			})
		);

		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const chip = page.getByRole('button', { name: DISPLAY_NAME, exact: true });
		await expect(chip).toBeVisible();

		await expect(chip).toHaveAccessibleDescription(/unreachable since/);
		await expect(
			chip,
			'the unreachable-since health description must carry none of the leftover advisory text'
		).not.toHaveAccessibleDescription(new RegExp(escapeRegExp(DISTINCTIVE_NOTICE)));

		const dot = chip.locator('span.size-2');
		await expect(dot).toHaveClass(/bg-destructive/);
		await expect(dot).not.toHaveClass(/bg-warning/);

		await page.unroute(`${kernel.baseURL}/api/sources`);
	});

	test('B — reachable + last_status ok + the same notice: the title still carries BOTH the synced wording and the notice (the anti-dead-code guard)', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [INSTANCE_ID], { logs: kernel.logs });

		await page.route(`${kernel.baseURL}/api/sources`, (route) =>
			route.fulfill({
				status: 200,
				contentType: 'application/json',
				body: JSON.stringify(
					fabricatedSourcesBody({ reachable: true, last_status: 'ok', last_notice: DISTINCTIVE_NOTICE })
				)
			})
		);

		await page.goto(`${kernel.baseURL}/w/${WEBSPACE}`);

		const chip = page.getByRole('button', { name: DISPLAY_NAME, exact: true });
		await expect(chip).toBeVisible();

		await expect(chip).toHaveAccessibleDescription(/synced/);
		await expect(
			chip,
			'the synced-plus-advisory health description must still carry the notice text — this fails if the gate is ever written against the chip\'s own computed tone (tone === "success"), which would make the advisory branch unreachable'
		).toHaveAccessibleDescription(new RegExp(escapeRegExp(DISTINCTIVE_NOTICE)));

		const dot = chip.locator('span.size-2');
		await expect(dot).toHaveClass(/bg-warning/);
		await expect(dot).not.toHaveClass(/bg-success/);

		await page.unroute(`${kernel.baseURL}/api/sources`);
	});
});
