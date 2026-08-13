// 09-04-PLAN.md Task 3: proves both geometry fixes in a real browser —
// assertions that would have failed against the shipped (pre-fix) code.
//
// - Fix 2 (SearchBox.svelte): the clear button's bounding box must not
//   jump on mousedown the way the shipped defect did (roughly half the
//   button's own 44px height, ~22px, the "15px jump" the roadmap named).
//   The clear button's own shared Button press affordance
//   (ui/button/button.svelte's `active:not-aria-[haspopup]:translate-y-px`)
//   is deliberately UNCHANGED and still applies its ordinary ~1px press —
//   Fix 2's contract is decoupling the collision, not zeroing the shared
//   affordance out for this one control (09-04-PLAN.md Task 1's <action>:
//   "applies exactly the 1px press it was designed to"). So this spec
//   asserts the vertical delta stays within a small, bounded tolerance
//   that comfortably distinguishes "ordinary 1px press" from "the ~22px
//   defect", not literal zero-pixel equality.
// - Fix 9 (DetailPane.svelte): the media previewer renders bounded at a
//   3:4 aspect ratio (width <= 384px) instead of the shipped
//   wide-and-short h-72 box, with extracted text flowing beside the float
//   rather than stacking below it.
//
// Fix 9's proof needs a mock item that actually HAS a rendition — the
// mock plugin never offers one by design (plugins/mock/plugin.go's
// noRenditionReason), so this spec enables the mock's rendition fixture
// (WEBSPACES_MOCK_RENDITION, plugins/mock/renditionfixture.go,
// docs/testing.md) for its own kernel, which attaches a fake PNG
// rendition to item "1" ("Welcome to the mock source") alone — the same
// item smoke-core-journey.spec.ts already proves carries non-empty
// extracted text, so one item exercises both the float and the flowing
// text at once.
//
// 11-02-PLAN.md Task 2 (D-14): WEBSPACES_MOCK_RENDITION alone on the
// kernel process is no longer enough to reach the plugin subprocess —
// kernel/pluginhost.launch's exec.Cmd now carries
// goplugin.ClientConfig.SkipHostEnv:true, so a plugin subprocess receives
// only the documented allowlist plus the values behind ${VAR} references
// this instance's own raw config declares. The mock source's own
// `extras.rendition` entry below supplies that reference, travelling the
// same documented, reference-driven path every other Phase 11 fixture
// migration in this repo now uses.
import { test, expect, waitForFirstSync } from '../fixtures/kernel';
import { webspacesWithKeywords, type FixtureConfigSpec } from '../fixtures/config-builder';

const MOCK_ID = 'mock';
const ITEM_1_TITLE = 'Welcome to the mock source';
// mockFullText["1"] verbatim (plugins/mock/plugin.go) — the same
// distinctive substring smoke-core-journey.spec.ts asserts, proving the
// live-fetched extracted text (not the stream's cached preview) landed.
const ITEM_1_FULL_TEXT_SUBSTRING = "mock source plugin's full extracted text for item 1";
const CLEAR_LABEL = 'Clear search';
const SEARCH_PLACEHOLDER = 'Search this webspace';

const configSpec: FixtureConfigSpec = {
	sources: [
		{
			id: MOCK_ID,
			plugin: 'topos-plugin-mock',
			displayName: 'Mock Source',
			// extras.rendition references ${WEBSPACES_MOCK_RENDITION} so the
			// value set on the kernel process below actually reaches this
			// instance's plugin subprocess (11-02-PLAN.md Task 2, D-14) — the
			// key name itself is arbitrary; only the ${VAR} reference matters.
			extras: { rendition: '${WEBSPACES_MOCK_RENDITION}' }
		}
	],
	webspaces: webspacesWithKeywords(['previewer'], ['demo']),
	env: { WEBSPACES_MOCK_RENDITION: '1' }
};

test.use({ configSpec });

test.describe('09-04 Task 3: search clear button stability and previewer geometry', () => {
	test('the clear button does not jump under a held mousedown', async ({ page, kernel }) => {
		await waitForFirstSync(kernel.baseURL, [MOCK_ID], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/previewer`);

		await page.getByPlaceholder(SEARCH_PLACEHOLDER).fill('demo');
		const clearButton = page.getByRole('button', { name: CLEAR_LABEL });
		await expect(clearButton).toBeVisible();

		const restBox = await clearButton.boundingBox();
		expect(restBox, 'expected the clear button to have a measurable bounding box at rest').not.toBeNull();

		// Move the mouse over the button's centre and hold the button down —
		// a genuine mousedown, so the browser's own :active pseudo-class
		// (and therefore the shared Button press affordance) actually
		// applies, not a synthetic dispatched event.
		await page.mouse.move(
			restBox!.x + restBox!.width / 2,
			restBox!.y + restBox!.height / 2
		);
		await page.mouse.down();
		const activeBox = await clearButton.boundingBox();
		// Release afterwards regardless of assertion outcome, leaving the
		// page in a sane state.
		await page.mouse.up();

		expect(
			activeBox,
			'expected the clear button to still have a measurable bounding box while pressed'
		).not.toBeNull();

		// The shipped defect moved the button by roughly half its own 44px
		// height (~22px) — the centring transform being wiped out entirely
		// by the press transform. The fix's own shared press affordance
		// still applies its ordinary ~1px translate, so the bound here is
		// tight enough to catch the original defect by a wide margin while
		// tolerating the deliberately-preserved, tiny app-wide press.
		const verticalDelta = Math.abs(activeBox!.y - restBox!.y);
		expect(
			verticalDelta,
			`expected the clear button's vertical position to stay within 2px under a held mousedown (rest y=${restBox!.y}, active y=${activeBox!.y}) — a larger delta means the centring transform collided with the press transform again`
		).toBeLessThanOrEqual(2);
	});

	test('the media preview box is bounded at a 3:4 aspect ratio, and extracted text flows beside it', async ({
		page,
		kernel
	}) => {
		await waitForFirstSync(kernel.baseURL, [MOCK_ID], { logs: kernel.logs });
		await page.goto(`${kernel.baseURL}/w/previewer`);

		const item1Row = page.getByRole('button').filter({ hasText: ITEM_1_TITLE }).first();
		await item1Row.click();
		await expect(page.getByRole('heading', { name: ITEM_1_TITLE })).toBeVisible();

		// The fixture rendition is image/png (never application/pdf), so
		// DetailPane.svelte's media branch renders the <img> fallback —
		// alt is the item's own title (decorative content aside, this is
		// what makes the element addressable without a new test hook).
		const previewImg = page.locator(`img[alt="${ITEM_1_TITLE}"]`);
		await expect(previewImg).toBeVisible();
		// Prove the image actually decoded, not just present in the DOM.
		await expect
			.poll(async () => previewImg.evaluate((el: HTMLImageElement) => el.naturalWidth))
			.toBeGreaterThan(0);

		// The aspect-locked BOX is the img's immediate parent — object-contain
		// on a non-3:4-shaped source image would letterbox inside the box,
		// so the box (not the img itself) is what must carry the 3:4 ratio.
		const previewBox = previewImg.locator('xpath=..');
		const boxRect = await previewBox.boundingBox();
		expect(boxRect, 'expected the aspect-locked preview box to have a measurable bounding box').not.toBeNull();

		expect(
			boxRect!.width,
			`expected the preview box's rendered width to be at most 384px (max-w-sm), got ${boxRect!.width}`
		).toBeLessThanOrEqual(384.5);

		const ratio = boxRect!.width / boxRect!.height;
		expect(
			ratio,
			`expected the preview box's rendered width/height ratio to be 0.75 (3:4) within tolerance, got ${ratio}`
		).toBeGreaterThanOrEqual(0.73);
		expect(ratio).toBeLessThanOrEqual(0.77);

		// The extracted text block's own bounding box top sits above the
		// preview box's bottom edge — true only if the text is wrapping in
		// the line-box space beside the float, not stacked in a separate
		// flex row below it (the shipped defect's layout).
		const textBlock = page.getByText(ITEM_1_FULL_TEXT_SUBSTRING);
		await expect(textBlock).toBeVisible();
		const textRect = await textBlock.boundingBox();
		expect(textRect, 'expected the extracted text block to have a measurable bounding box').not.toBeNull();

		const previewBottom = boxRect!.y + boxRect!.height;
		expect(
			textRect!.y,
			`expected the extracted text block's top (${textRect!.y}) to sit above the preview box's bottom (${previewBottom}) — proving the text flows beside the float rather than starting below it`
		).toBeLessThan(previewBottom);
	});
});
