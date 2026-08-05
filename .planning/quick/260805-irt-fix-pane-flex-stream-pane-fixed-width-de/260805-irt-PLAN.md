---
phase: 260805-irt
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - web/src/lib/pane-layout.test.ts
  - web/src/routes/w/[webspace]/+page.svelte
autonomous: false
requirements: [QUICK-260805-irt]

estimate:
  tokens: 25000
  raw_tokens: 18000
  tasks: 3
  confidence: low

must_haves:
  truths:
    - "With an item selected, widening or narrowing the browser window changes the detail pane's width while the stream pane's width stays constant."
    - "With no item selected, the stream pane still fills the full content width of <main>, exactly as it does today."
    - "The stream pane and the detail pane still scroll independently; scrolling one never moves the other."
    - "Stream rows still truncate/clamp rather than overflow horizontally at the narrower fixed stream width."
  artifacts:
    - "web/src/lib/pane-layout.test.ts — region-scoped source-scan guard asserting the fixed/flex inversion"
    - "web/src/routes/w/[webspace]/+page.svelte — stream wrapper fixed, detail wrapper flexing"
  key_links:
    - "The <main class=\"flex ...\"> row in web/src/routes/w/[webspace]/+page.svelte is the ONLY place pane sizing is assigned — neither StreamList.svelte nor DetailPane.svelte sets its own outer width."
    - "The stream wrapper's fixed-vs-flex class must be driven by the same `selectedItem` value that gates whether the detail pane renders at all, so the two can never disagree."
---

<objective>
Invert which pane absorbs viewport width changes in the webspace two-pane layout. Today the stream (list) pane is `flex-1` and the detail pane is pinned at `w-[480px] shrink-0`, so growing the window grows the list and leaves the reading surface fixed. Reverse it: the stream pane takes the fixed width, the detail pane flexes.

Purpose: The detail pane is the reading surface (extracted text, PDF/image embeds, chat transcripts). Extra viewport width belongs to the content being read, not to a list of truncated one-line titles.

Output: A source-scan invariant test plus a class-level change to the single `<main>` flex row that owns pane sizing.
</objective>

<execution_context>
@$HOME/.claude/gsd-core/workflows/execute-plan.md
@$HOME/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/PROJECT.md
@.planning/STATE.md

@web/src/routes/w/[webspace]/+page.svelte
@web/src/lib/components/date-format.test.ts
</context>

<interfaces>
**Current state of the two panes** (`web/src/routes/w/[webspace]/+page.svelte`, inside `<main class="flex min-h-0 flex-1 gap-8 px-6 py-8">`):

- Stream wrapper (line ~210), holds `SearchResults` or `StreamList`:
  `min-h-0 min-w-0 flex-1 overflow-x-hidden overflow-y-auto`
- Detail wrapper (line ~236), rendered only inside `{#if selectedItem}`, holds `DetailPane`:
  `flex w-[480px] shrink-0 flex-col overflow-hidden border-l border-border pl-8`

**Height chain (do not disturb):** `+layout.svelte` `h-screen flex-col` → `min-h-0 flex-1` → page `flex h-full min-h-0 flex-col` → `WebspaceHeader` (`shrink-0`) + `<main class="flex min-h-0 flex-1 ...">`. Both panes rely on `min-h-0` for their independent scroll regions.

**Test harness facts:**
- `vitest` runs in `environment: 'node'`, `include: ['src/**/*.test.ts']` (`web/vite.config.ts`). There is no jsdom / component-mount harness, and jsdom would not help here — jsdom performs no layout, so flexbox widths are never computed in it.
- The project's established pattern for asserting on markup is a Node source-scan test: see `web/src/lib/components/date-format.test.ts`, which reads `.svelte` files with `readFileSync` from `node:fs` and asserts on their text.
- Node built-in types come from the hand-written `web/src/lib/node-builtins.d.ts` ambient declarations — `readFileSync`, `readdirSync`, `dirname`, `join`, `fileURLToPath` are already declared. **Add no new dependency**; if a needed built-in is not declared there, extend that file rather than installing `@types/node`.
- Gates: `npm --prefix web run test` (vitest) and `npm --prefix web run check` (svelte-check, must stay at 0 errors).
</interfaces>

<tasks>

<task type="auto">
  <name>Task 1: Add a failing region-scoped guard for the pane fixed/flex inversion</name>
  <files>web/src/lib/pane-layout.test.ts</files>
  <precondition>`web/src/routes/w/[webspace]/+page.svelte` exists and its `<main>` row contains exactly one class attribute with `overflow-x-hidden` (stream wrapper) and exactly one with `border-l` (detail wrapper).</precondition>
  <behavior>
    Reads `../routes/w/[webspace]/+page.svelte` relative to the test file and asserts, against the *extracted class attribute of each pane wrapper* (never the whole file):
    - Test 1 (loud-failure guard): the route source read is non-empty — a wrong path resolution must fail here, not make every later assertion vacuously pass.
    - Test 2: exactly one class attribute contains `overflow-x-hidden`, and exactly one contains `border-l`.
    - Test 3: the detail wrapper's class attribute does NOT match a fixed pixel-width utility (regex `/w-\[\d+px\]/`) and does NOT contain a no-shrink utility.
    - Test 4: the detail wrapper's class attribute contains the grow utility and a zero-min-width utility.
    - Test 5: the stream wrapper's class attribute DOES match `/w-\[\d+px\]/` and DOES contain the no-shrink utility.
    At the pre-fix source state, Tests 3 and 5 must both fail (RED).
  </behavior>
  <action>
Create the new test file following the exact shape of `web/src/lib/components/date-format.test.ts` (vitest `describe`/`it`, `readFileSync` from `node:fs`, `dirname`/`join` from `node:path`, `fileURLToPath` from `node:url`, resolving the directory from `import.meta.url`).

Resolve the route file as `join(dirname(fileURLToPath(import.meta.url)), '..', 'routes', 'w', '[webspace]', '+page.svelte')` and read it once at module scope into a constant.

Write a helper that, given a marker substring, collects every `class="..."` attribute value in the route source via a global regex over `class="([^"]*)"`, filters to those whose value contains the marker, asserts that exactly one matched, and returns that single attribute value. Every subsequent assertion runs against that returned string, NOT against the whole file — this scoping is what makes the guard immune to any prose an author later writes in a Svelte comment elsewhere in the file. Note that rationale in a comment inside the helper.

Use the marker `overflow-x-hidden` for the stream wrapper and `border-l` for the detail wrapper (each is unique to its own pane wrapper in this file today; the exactly-one assertion in the helper is what enforces that it stays true).

Define one shared regex constant for a fixed pixel width utility — an arbitrary-value width bracket containing digits followed by `px`. Assert the inversion against the two extracted strings per the behavior block above, using the growth utility `flex-1`, the no-shrink utility `shrink-0`, and the zero-min-width utility `min-w-0`.

Give every `expect` a failure message that states the layout invariant in plain language (e.g. that the reading pane must absorb viewport width while the list pane stays constant), matching the explanatory-message style used throughout `date-format.test.ts`.

Do not modify the route file in this task — the failing run is the point.
  </action>
  <verify>
    <automated>cd web &amp;&amp; npm run test 2>&amp;1 | tail -30   # MUST show pane-layout.test.ts failing (RED) with 2 failed assertions</automated>
  </verify>
  <done>`web/src/lib/pane-layout.test.ts` exists, runs under the existing vitest config with no new dependency installed, and fails against the current (un-inverted) route source with the detail-wrapper fixed-width assertion and the stream-wrapper fixed-width assertion both reported as failures.</done>
</task>

<task type="auto">
  <name>Task 2: Invert the pane sizing so the stream is fixed and the detail pane flexes</name>
  <files>web/src/routes/w/[webspace]/+page.svelte</files>
  <action>
Edit only the two pane wrapper `<div>` elements inside `<main>`. Change no script logic, no props, no component internals, and nothing in `StreamList.svelte` or `DetailPane.svelte`.

**Detail wrapper** (the one carrying `border-l border-border pl-8`): remove the fixed width and the no-shrink utility; add `flex-1` and `min-w-0`. Keep `flex`, `flex-col`, `overflow-hidden`, and the border/padding utilities exactly as they are. `min-w-0` is load-bearing, not cosmetic: as a growing flex child, wide intrinsic content (a PDF `<embed>`, a long unbroken token in `whitespace-pre-wrap` text) would otherwise set a floor on the pane's width and push the layout.

**Stream wrapper** (the one carrying `overflow-x-hidden`): make its sizing conditional on `selectedItem`, the same value that already gates whether the detail wrapper renders. When an item is selected it takes the fixed width `w-[480px]` plus `shrink-0`; when nothing is selected it keeps `flex-1`. Express this as a Svelte class interpolation inside the attribute, e.g. a ternary on `selectedItem` yielding one of the two utility pairs, with `min-h-0 min-w-0 overflow-x-hidden overflow-y-auto` remaining unconditional.

The conditional is required, not optional: making the stream unconditionally fixed would regress the single-pane state, stranding a narrow column beside a large blank region on a wide display whenever no item is open. Today the single-pane state fills the full content width and must continue to.

Reuse the existing `480px` constant rather than inventing a new number — it is the width this layout already sanctions for a pane, simply moved from the detail side to the stream side. Do not add a theme token or a named container key for it; `web/src/app.css` documents (with a recorded past regression) that named keys in the spacing namespace shadow the container scale.

Update the explanatory comment already sitting above the stream wrapper so it describes the sizing relationship as well as the scroll relationship. Keep its existing statements about independent scroll regions and about `SearchResults` rendering in place of `StreamList` inside the same pane.
  </action>
  <verify>
    <automated>cd web &amp;&amp; npm run test &amp;&amp; npm run check &amp;&amp; npm run build</automated>
  </verify>
  <done>`pane-layout.test.ts` passes (GREEN) alongside all pre-existing vitest suites; `svelte-check` reports 0 errors; the production build succeeds and the emitted stylesheet still generates the `w-[480px]`, `shrink-0`, `flex-1`, and `min-w-0` utilities now referenced from inside a class interpolation.</done>
</task>

<task type="checkpoint:human-verify" gate="blocking">
  <name>Task 3: Confirm the resize behavior in a real browser</name>
  <what-built>The stream/detail flex relationship is inverted: the stream pane now holds a fixed 480px width whenever an item is open, and the detail pane absorbs all viewport width changes. A source-scan test guards the invariant, but no automated gate can prove real flexbox behavior — vitest runs in Node, and jsdom performs no layout, so computed pane widths do not exist outside a real browser. This resize check is the only way to actually exercise it.</what-built>
  <how-to-verify>
1. Run `make dev` from the repo root (starts the kernel on 127.0.0.1:7777 and opens the Vite dev server).
2. Open a webspace route (`/w/<name>`) that has items in its stream.
3. **Two-pane resize (the fix):** click a stream item so the detail pane opens. Drag the window edge to widen and then narrow the browser, or open devtools and drag the viewport divider.
   - Expect: the stream pane's width does NOT change. All the extra (or lost) width goes to the detail pane on the right.
   - Before this change the opposite happened — the list grew and the detail pane stayed pinned.
4. **Single-pane regression check:** press nothing / deselect so no item is open (reload the route, or navigate to another webspace).
   - Expect: the stream fills the full content width edge to edge, with no narrow-column-plus-blank-space layout. Now resize the window — the stream should flex to fill.
5. **Independent scroll (preserved):** with the detail pane open on an item that has long extracted text or a PDF/image preview, scroll the stream list.
   - Expect: the detail pane's own scroll position does not move, and vice versa.
6. **Stream row integrity at the narrower fixed width:** with the detail pane open, check a row with a long title, a group label, a date, and tag pills.
   - Expect: the title truncates to one line, the snippet clamps to 2 lines, and nothing overflows horizontally or forces a horizontal scrollbar in the stream pane.
7. **Detail pane at a wide viewport:** maximize the window with an item open.
   - Expect: embeds and text fill the wider pane without the pane pushing past the window edge or introducing a horizontal scrollbar on the page.
  </how-to-verify>
  <resume-signal>Type "approved" if all seven checks behave as described, or describe which step misbehaved and what you saw instead.</resume-signal>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|----------|-------------|
| *(none newly crossed)* | This change touches only CSS utility classes on two wrapper elements in an already-rendered authenticated-local route. It introduces no new input, no new network call, no new data path, and no new dependency. |

## STRIDE Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Plan |
|-----------|----------|-----------|----------|-------------|-----------------|
| T-260805-irt-01 | Denial of Service | `web/src/routes/w/[webspace]/+page.svelte` two-pane layout | low | mitigate | A layout regression that strands the stream in a narrow column (or lets intrinsic detail-pane content push the row wider than the viewport) makes the primary view unusable — availability of the UI itself. Mitigated by the `min-w-0` requirement on the growing detail pane (Task 2), the conditional that preserves the full-width single-pane state, and human-verify steps 4-7. |
| T-260805-irt-02 | Tampering | package manager installs | low | accept | No package is installed by this plan — the source-scan test runs on the existing vitest + hand-written `node-builtins.d.ts` setup. Task 1 explicitly forbids adding a dependency, so the supply-chain surface is unchanged and no legitimacy checkpoint is required. |
</threat_model>

<verification>
- `npm --prefix web run test` — full vitest suite green, including the new `pane-layout.test.ts` and the pre-existing `format`, `date-format`, `detail-body`, `sources`, and `staleness` suites.
- `npm --prefix web run check` — svelte-check at 0 errors (the standing project bar).
- `npm --prefix web run build` — production build succeeds with the conditional class interpolation.
- Human-verify checkpoint — seven-step manual resize walkthrough covering the fix, the single-pane regression, independent scroll, row truncation at the narrower width, and wide-viewport detail rendering.
</verification>

<success_criteria>
- With an item open, viewport width changes are absorbed entirely by the detail pane; the stream pane's rendered width is constant.
- With no item open, the stream pane still fills the full content width and still flexes with the viewport.
- Independent scroll regions, source-filter chips, search-in-place-of-stream rendering, and stream row truncation/clamping all behave exactly as before.
- `pane-layout.test.ts` fails on the pre-fix source and passes on the post-fix source, and is scoped to the two pane class attributes so a future comment cannot silently satisfy or invalidate it.
- No new npm dependency; `svelte-check` still reports 0 errors.
</success_criteria>

<output>
Create `.planning/quick/260805-irt-fix-pane-flex-stream-pane-fixed-width-de/260805-irt-SUMMARY.md` when done
</output>
