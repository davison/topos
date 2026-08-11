---
phase: quick-260811-rqc
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - web/src/lib/components/WebspaceHeader.svelte
  - web/src/lib/components/header-branding.test.ts
  - web/e2e/specs/header-branding.spec.ts
autonomous: false
requirements: [QUICK-260811-rqc]
user_setup: []

estimate:
  tokens: 45000
  raw_tokens: 35000
  tasks: 3
  confidence: low

must_haves:
  truths:
    - "On every webspace route the header's top band renders a branding lockup on the RIGHT: the app icon, the wordmark `topos` beside it, and the tagline `bringing all your topics to one place` on its own line directly beneath the wordmark."
    - "The icon is a real, DECODED image in the browser (naturalWidth > 0), served from the already-shipped `/app-icon.png` static asset — not a broken `<img>` that merely exists in the DOM."
    - "Both the wordmark and the tagline render in the muted-foreground token, visibly a different computed colour from the application's default text colour rendered right beside them (the webspace-switcher title in the same band)."
    - "The chip row's overflow measurement is provably untouched: the branding block is a sibling ABOVE the measured chip row, never a child of it, so `rowEl.clientWidth` still spans the header's full content width and `visibleChipCount` receives byte-identical inputs to before this change."
    - "The `+` add-source trigger and the `Refresh all` button stay visible, on-screen and non-overlapped with the branding block at realistic desktop viewport widths."
    - "`cd web && npm run check` reports 0 errors, `npm test` is green (including every pre-existing chip-geometry guard), and `make e2e` is green."
  artifacts:
    - "web/src/lib/components/WebspaceHeader.svelte — the title band becomes a two-column flex row (switcher left, branding lockup right); the chip row below it is structurally unchanged"
    - "web/src/lib/components/header-branding.test.ts — comment-stripped structural guard: lockup composition, muted token on both texts, and the sibling-not-child placement relative to the measured chip row"
    - "web/e2e/specs/header-branding.spec.ts — browser proof: icon decodes, both texts render, muted tone differs from application text colour, chip row and its affordances un-regressed"
  key_links:
    - "branding block ↔ chip row (`bind:this={rowEl}`): SIBLING, never nested. This is the single connection whose breakage silently corrupts `visibleChipCount`'s available-width input with no visible error — guarded structurally in Task 1 and behaviourally in Task 2."
    - "branding `<img>` ↔ `/app-icon.png` static asset already embedded via the SvelteKit build and served by the kernel (the same path `+layout.svelte`'s favicon link and `QRPanel.svelte` already consume)."
    - "wordmark + tagline spans ↔ the `--muted-foreground` design token, applied via `text-muted-foreground` — the locked colour requirement."
---

<objective>
Put the topos identity inside the application UI: an app-icon + wordmark +
tagline lockup in the top-right of the webspace header, in muted text.

Purpose: the app icon added in Phase 9 currently appears only in the browser
tab. The product has no visible identity in its own interface.

Output: a branding lockup in `WebspaceHeader.svelte`, a structural unit guard,
and a Playwright spec (per the standing 07.1 D-11 rule that a UI change extends
the e2e suite as part of its definition of done).
</objective>

<execution_context>
@$HOME/.claude/gsd-core/workflows/execute-plan.md
@$HOME/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@.planning/todos/pending/2026-08-11-header-branding-app-icon-topos-wordmark-and-tagline.md

@web/src/lib/components/WebspaceHeader.svelte
@web/src/lib/components/source-chip-pill.test.ts
@web/e2e/specs/09-plugin-icon.spec.ts
</context>

<interfaces>

**Locked layout requirement (the user's own words, treat as non-negotiable):**
app icon in the TOP RIGHT of the header; next to it the word `topos` in a
large-ish font; underneath `topos` the tagline
`bringing all your topics to one place` in smaller text; both texts muted
relative to the application text colour.

**Existing facts this plan builds on (already verified in the codebase — do not
re-derive):**

- `web/src/lib/components/WebspaceHeader.svelte` line ~329: the root is
  `<header class="shrink-0 border-b border-border bg-card px-6 py-6">`, and
  `<WebspaceSwitcher .../>` is currently a bare, full-width direct child — the
  header's top band.
- The chip row is a SEPARATE, later sibling: the `<div class="mt-4 flex
  flex-nowrap items-center gap-2 overflow-hidden" bind:this={rowEl}>` block. Its
  `clientWidth` is the sole available-width input to `visibleChipCount`
  (`measure()`, line ~245).
- `--muted-foreground: #94a3b8` and `--foreground: #f1f5f9` (`web/src/app.css`),
  exposed to Tailwind as `text-muted-foreground` / `text-foreground`
  respectively.
- The webspace-switcher trigger's title span is
  `text-[28px] leading-[1.2] font-semibold text-foreground` with
  `max-w-[min(80vw,32rem)] truncate` — this is the "application text colour"
  the branding must read as muted against, sitting in the same band.
- `QRPanel.svelte` line 329 is the house precedent for this exact asset:
  `<img src="/app-icon.png" alt="" class="size-10 rounded-md" />` — decorative
  empty alt, because adjacent copy already states what it is.
- `AddSourceModal`'s picker trigger carries `aria-label="Add source"` and is the
  only element with that accessible name while the modal is closed.
- Unit-test house pattern (`source-chip-pill.test.ts`): vitest under
  `environment: 'node'` with NO component-mount harness — assertions are
  comment-stripped source scans with `extractBetween`-style region scoping, a
  found-non-empty-source guard first, and one consequence-describing message per
  assertion.
- E2E house pattern (`09-plugin-icon.spec.ts`): `test.use({ configSpec })` from
  `../fixtures/kernel`, `waitForFirstSync(kernel.baseURL, [ids], { logs })`,
  then `page.goto(\`${kernel.baseURL}/w/${WEBSPACE}\`)`. A decoded image is
  proven with `expect.poll(() => icon.evaluate((el: HTMLImageElement) =>
  el.naturalWidth)).toBeGreaterThan(0)`.
- A single spec runs via `make e2e E2E_ARGS=specs/header-branding.spec.ts`.

</interfaces>

<tasks>

<task type="tracer" tdd="true">
  <name>Task 1: Branding lockup in the header's top band, wired end to end</name>
  <files>web/src/lib/components/WebspaceHeader.svelte, web/src/lib/components/header-branding.test.ts, web/e2e/specs/header-branding.spec.ts</files>
  <read_first>
    web/src/lib/components/WebspaceHeader.svelte (lines 329-345 — the header root and the bare WebspaceSwitcher child; lines 376-395 — the chip row's opening tag with bind:this={rowEl}),
    web/src/lib/components/source-chip-pill.test.ts (lines 48-70 — the comment-strip / read-source / region-scope helper preamble to mirror),
    web/e2e/specs/09-plugin-icon.spec.ts (the whole file — fixture wiring and the naturalWidth decode proof to mirror)
  </read_first>
  <behavior>
    Structural guard (header-branding.test.ts), all against COMMENT-STRIPPED
    WebspaceHeader source so no prose in the component can satisfy or trip a
    check:
    - Test 1: the source contains an `<img>` whose src is `/app-icon.png` and
      whose alt is the empty string (decorative — the wordmark beside it names
      the product).
    - Test 2: the literal wordmark text `topos` and the literal tagline text
      `bringing all your topics to one place` both appear in the markup.
    - Test 3: the branding region carries `text-muted-foreground`, and does NOT
      carry the application's default text-colour utility anywhere inside it —
      region-scoped to the extracted lockup block, so the switcher's own use of
      that utility elsewhere in the file cannot mask a regression.
    - Test 4 (the load-bearing one): the index of the `/app-icon.png` `<img>` in
      the comment-stripped source is strictly LESS than the index of
      `bind:this={rowEl}`, i.e. the lockup is emitted before — and therefore
      outside — the measured chip row. Message must name the consequence: a
      lockup nested inside the measured row would silently shrink
      `visibleChipCount`'s available width.
    - Test 5: the wordmark's font-size class is strictly smaller than the
      switcher title's `text-[28px]`, so the branding never out-shouts the
      webspace title it sits beside; and the tagline's font-size class is
      strictly smaller than the wordmark's.

    E2E tracer proof (header-branding.spec.ts, ONE test this task):
    - the header's `img[src="/app-icon.png"]` is visible AND decodes
      (naturalWidth > 0), and the texts `topos` and
      `bringing all your topics to one place` are both visible on a webspace
      route served by a real kernel.
  </behavior>
  <action>
Write the structural tests in header-branding.test.ts FIRST and watch them fail,
then implement, then write the single e2e tracer test.

Implementation in WebspaceHeader.svelte: replace the bare `<WebspaceSwitcher
.../>` child with a two-column flex band — `flex items-start justify-between
gap-4` — whose first column is a `min-w-0` wrapper holding the UNCHANGED
`<WebspaceSwitcher {...}/>` (min-w-0 is what lets its existing `truncate` actually
shrink inside a flex row), and whose second column is the new branding lockup
marked `shrink-0` so the switcher title truncates before the branding is
squeezed.

The lockup: a horizontal flex row, `items-center`, small gap. Left of the row is
the icon — reuse QRPanel's exact treatment for this asset (`src="/app-icon.png"`,
empty alt, `size-10 rounded-md`, plus `shrink-0`). Right of the icon is a
vertical flex column holding two spans: the wordmark `topos` at
`text-[20px] leading-[1.2] font-semibold`, and beneath it the tagline
`bringing all your topics to one place` at `text-[12px] leading-[1.4]`. Put
`text-muted-foreground` on both text spans (or on the column that owns them —
either satisfies the guard, pick one and be consistent). Do NOT apply the
application's default text-colour utility anywhere inside the lockup, and do not
name that utility in a code comment inside the lockup either.

Weight discipline: the wordmark uses the same semibold the switcher title
already uses; the tagline stays at the default weight. Introduce no third weight.

CRITICAL — do not touch the chip row. The new band is a sibling that closes
BEFORE `<div ... bind:this={rowEl}>` opens. Nothing about `rowEl`, `measureEl`,
`trailingEl`, `addSourceWrapperEl`, `measure()`, `CHIP_ROW_GAP_PX`,
`combinedReservedWidth` or `visibleChipCount` changes — the chip row remains a
full-width block-level child of `<header>`, so its `clientWidth` is byte-identical
to before. If you find yourself editing any of those names, stop: the lockup has
been placed in the wrong parent.

Interpretation note to record in the SUMMARY: `next to it the word topos` is
implemented as the conventional lockup order — icon first, then wordmark —
with the whole block right-aligned in the header band, which is what
`app icon displayed in the top right` describes. Task 3 confirms this reading
visually.

For the e2e file, mirror 09-plugin-icon.spec.ts's fixture wiring exactly: a
single mock source and one webspace in the configSpec, `waitForFirstSync`, then
navigate. Scope the icon locator to the page's `banner`/header region so it can
never accidentally resolve QRPanel's copy of the same asset.
  </action>
  <verify>
    <automated>cd web &amp;&amp; npx vitest run src/lib/components/header-branding.test.ts &amp;&amp; npm run check</automated>
    <automated>cd /home/darren/projects/davison/topos &amp;&amp; make e2e E2E_ARGS=specs/header-branding.spec.ts</automated>
  </verify>
  <done>All five structural guards pass; `npm run check` reports 0 errors; the e2e tracer proves the icon decodes and both texts render in a real browser against a real kernel.</done>
</task>

<task type="auto">
  <name>Task 2: Prove the muted tone and the chip-row non-regression in the browser</name>
  <files>web/e2e/specs/header-branding.spec.ts</files>
  <read_first>web/e2e/specs/header-branding.spec.ts (Task 1's output), web/src/lib/components/WebspaceHeader.svelte (lines 376-472 — the chip row, its + trigger and its trailing Refresh all group, for accurate locators)</read_first>
  <action>
Extend the spec from Task 1 with the assertions the tracer deliberately left
out. Keep them in the same file, as additional tests in the same describe block.

Muted-tone proof: read the computed `color` of the wordmark span and of the
tagline span, and the computed `color` of the webspace-switcher title span
sitting in the same band. Assert the two branding colours are equal to each
other and STRICTLY DIFFERENT from the switcher title's. Compare computed values
rather than hardcoding a hex literal — the assertion should survive a palette
change and only fail if the branding stops being muted. Give the failure message
the consequence: the branding is rendering at full application text weight and
competes with the webspace title.

Chip-row non-regression proof: configure the fixture with ENOUGH source
instances that the chip row is genuinely populated (three or more), then assert
(a) at least one source chip is visible, (b) the `Add source` trigger is visible,
(c) the `Refresh all` button is visible, and (d) no bounding-box intersection
between the branding block and the chip row — compute both `boundingBox()`
values and assert the branding's bottom edge is at or above the chip row's top
edge, i.e. they occupy separate horizontal bands and cannot overlap. Assert the
chip row's own measured width is still essentially the header's full content
width (within a small tolerance for the header's horizontal padding) — this is
the behavioural counterpart to Task 1's structural placement guard and is what
would catch a future refactor that nests the lockup into the measured row.

Run the assertions at a realistic desktop viewport, and additionally at a
narrower one, to confirm the switcher title truncates rather than the branding
being pushed off-screen or overlapping the chip row.
  </action>
  <verify>
    <automated>cd /home/darren/projects/davison/topos &amp;&amp; make e2e E2E_ARGS=specs/header-branding.spec.ts</automated>
    <automated>cd web &amp;&amp; npm run check &amp;&amp; npm test</automated>
  </verify>
  <done>The spec proves both branding texts render in a colour distinct from the application text colour beside them; with 3+ sources configured the chip row, its `+` trigger and `Refresh all` all stay visible and non-overlapping with the branding at both viewport widths; the full `npm test` suite (including every pre-existing chip-geometry guard) is green.</done>
</task>

<task type="checkpoint:human-verify" gate="blocking">
  <name>Task 3: Confirm the branding lockup reads right</name>
  <what-built>The header branding lockup: app icon, `topos` wordmark, and the tagline `bringing all your topics to one place`, right-aligned in the header's top band beside the webspace-switcher title, both texts in the muted-foreground token. Structural unit guards and a Playwright spec ship with it.</what-built>
  <how-to-verify>
Run `make dev` and open the webspace UI in a browser (the dev server URL make
dev prints).

Confirm, in order:
1. The app icon and the two lines of text sit at the TOP RIGHT of the header,
   level with the webspace name on the left.
2. The lockup order reads correctly to you — icon first, then `topos` with the
   tagline beneath it. If you actually wanted the wordmark to the LEFT of the
   icon (a literal reading of "icon in the top right, text next to it"), say so
   and it will be flipped.
3. `topos` is clearly larger than the tagline, and clearly smaller than the
   webspace name on the left.
4. Both branding lines look dimmer than the webspace name — muted, not
   full-strength text.
5. Resize the browser window narrower and wider. The source chips, the `+`
   button and `Refresh all` must never overlap or be pushed under the branding;
   the webspace name should truncate with an ellipsis before anything collides.
  </how-to-verify>
  <resume-signal>Type "approved", or describe what looks wrong (arrangement, size, tone, or collision).</resume-signal>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| *(none new)* | This change adds no input parsing, no network call, no new dependency, and no new route. It renders a static, already-shipped, already-served local asset plus two hardcoded string literals. |

## STRIDE Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Plan |
|-----------|----------|-----------|----------|-------------|-----------------|
| T-rqc-01 | Information disclosure | `WebspaceHeader.svelte` branding lockup | low | accept | The lockup renders only compile-time constants (the product name and tagline) and a static asset already public at `/app-icon.png` — no webspace, source, config or item data reaches it. No user-controlled string is interpolated. |
| T-rqc-02 | Denial of service | chip-row overflow measurement | low | mitigate | A branding block nested inside the measured chip row would shrink the row's available width and could collapse every chip into the overflow popover — a self-inflicted usability failure, not an attack. Mitigated by Task 1's structural placement guard and Task 2's bounding-box/width assertions. |
| T-rqc-SC | Tampering | package installs | n/a | accept | No npm/pip/cargo install is performed by this plan — zero new dependencies. |
</threat_model>

<verification>
- `cd web && npm run check` — 0 errors.
- `cd web && npm test` — green, including `source-chip-pill.test.ts`,
  `source-chip-selected.test.ts`, `pane-layout.test.ts` and every other
  pre-existing geometry guard, unmodified.
- `make e2e` — the full suite green, not only the new spec.
- `git diff` on `WebspaceHeader.svelte` touches ONLY the top band: no change to
  `measure()`, `visibleChipCount`, `CHIP_ROW_GAP_PX`, `combinedReservedWidth`,
  or any of the five bound element refs.
</verification>

<success_criteria>
- The header renders the icon + `topos` + tagline lockup, top right, in muted
  text, on every webspace route.
- The icon decodes in a real browser (naturalWidth > 0), proven by Playwright.
- The chip row's overflow measurement provably receives the same available width
  as before, proven both structurally (source placement) and behaviourally
  (bounding boxes + measured width).
- All three gates green: `npm run check`, `npm test`, `make e2e`.
- The human-verify checkpoint returned "approved".
</success_criteria>

<output>
Create `.planning/quick/260811-rqc-header-branding-app-icon-topos-wordmark-/260811-rqc-SUMMARY.md` when done.
</output>
