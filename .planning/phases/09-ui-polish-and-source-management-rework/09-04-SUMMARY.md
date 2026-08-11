---
phase: 09-ui-polish-and-source-management-rework
plan: 04
subsystem: ui
tags: [svelte5, tailwind, css-transforms, css-float, go-plugin, playwright, e2e]

requires:
  - phase: 09-01
    provides: "plugins/mock/plugin.go's icon wiring (Describe RPC, SourcePlugin struct shape) — this plan's Fetch/main.go edits land in the same files"
provides:
  - "SearchBox.svelte's clear button centred via a non-transformed flex track (absolute inset-y-0 right-0 flex items-center), decoupled from ui/button/button.svelte's shared press affordance"
  - "DetailPane.svelte's media branch: bounded, aspect-locked box (w-full max-w-sm aspect-[3/4]) that floats left with flowing text when content.text is present, centres unfloated otherwise"
  - "plugins/mock's WEBSPACES_MOCK_RENDITION fixture — an opt-in, item-scoped fake PNG rendition making DetailPane's media branch reachable by the hermetic e2e harness"
  - "FixtureConfigSpec.env (web/e2e/fixtures/config-builder.ts/kernel.ts) — the first general-purpose mechanism for an e2e spec to set extra env vars on its own kernel process"
affects: [09-05, 09-06, 09-07]

actuals:
  tokens: 12150
  tasks: 3
  commits: 3

tech-stack:
  added: []
  patterns:
    - "Positioning-technique decoupling over affordance special-casing: when two CSS utilities write the same custom property, replace the colliding technique rather than disabling the shared one for a single control"
    - "Conditional CSS float (float-left, gated on data presence) as the mechanism for 'box beside text when there is text, centred block otherwise' — no JS layout logic, no clearfix element (the parent's own overflow-y-auto already establishes the block-formatting context)"
    - "Mock-plugin fixture shape (readiness.go/renditionfixture.go): a dedicated file, a WEBSPACES_-prefixed env var read via an injected getenv parameter (never direct os.Getenv), off-by-default, wired into the SourcePlugin struct via a with*() chainable setter called from main.go"
    - "FixtureConfigSpec.env: a spec-level escape hatch to reach a kernel-inherited env var, layered onto (never replacing) kernel.ts's fixed spawn-time allowlist"

key-files:
  created:
    - web/src/lib/components/search-box-clear.test.ts
    - web/src/lib/components/detail-media-box.test.ts
    - plugins/mock/renditionfixture.go
    - plugins/mock/renditionfixture_test.go
    - plugins/mock/assets/fixture-rendition.png
    - web/e2e/specs/09-search-clear-and-previewer.spec.ts
  modified:
    - web/src/lib/components/SearchBox.svelte
    - web/src/lib/components/DetailPane.svelte
    - plugins/mock/plugin.go
    - plugins/mock/main.go
    - docs/testing.md
    - web/e2e/fixtures/config-builder.ts
    - web/e2e/fixtures/kernel.ts

key-decisions:
  - "e2e clear-button assertion uses a bounded tolerance (<=2px), not literal pixel equality — the shared Button press affordance's own ~1px translate is deliberately preserved (Fix 2's contract decouples the collision, it does not zero the affordance out for one control), so a strict-equality assertion would itself be wrong per the plan's own Task 1 instruction"
  - "Fixture rendition attached to mock item '1' specifically (not a new item) — it already carries non-empty extracted text, letting one item exercise both the float and the flowing-text UI consideration at once"
  - "Added FixtureConfigSpec.env (Rule 3, blocking): no existing mechanism let an e2e spec set WEBSPACES_MOCK_RENDITION on its own kernel process — kernel.ts's spawn env was a fixed, unparameterized allowlist. Layered on top of (never replacing) the existing PATH/HOME/XDG_* allowlist so T-07.1-05's isolation guarantee is unchanged."

patterns-established:
  - "A plugin fixture scoped to ONE designated item id, proven both by the fixture-off byte-identical test and an explicit 'every other item unaffected' test — the shape any future mock-plugin fixture needing per-item rather than plugin-wide behavior should follow"

requirements-completed: []

coverage:
  - id: D1
    description: "SearchBox.svelte's clear button holds an identical (within the shared press affordance's own tiny tolerance) vertical position across rest/hover/active/focus-visible — the transform-collision jump is gone, and ui/button/button.svelte is untouched"
    verification:
      - kind: unit
        ref: "web/src/lib/components/search-box-clear.test.ts"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/09-search-clear-and-previewer.spec.ts > the clear button does not jump under a held mousedown"
        status: pass
    human_judgment: false
  - id: D2
    description: "DetailPane.svelte's media branch is bounded at a 3:4 aspect ratio (max-w-sm), applied uniformly to the PDF iframe and img fallback, floats left with flowing text when text is present, and centres unfloated when it is not"
    verification:
      - kind: unit
        ref: "web/src/lib/components/detail-media-box.test.ts"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/09-search-clear-and-previewer.spec.ts > the media preview box is bounded at a 3:4 aspect ratio, and extracted text flows beside it"
        status: pass
    human_judgment: false
  - id: D3
    description: "plugins/mock's WEBSPACES_MOCK_RENDITION fixture makes the media branch reachable by the hermetic harness, scoped to one item, byte-identical when off, image/png confirmed already allowlisted"
    verification:
      - kind: unit
        ref: "plugins/mock/renditionfixture_test.go (4 mock <behavior> cases incl. fixture-off byte-identical)"
        status: pass
      - kind: integration
        ref: "make test-portable (all 8 Go modules)"
        status: pass
    human_judgment: false

duration: ~35min
completed: 2026-08-11
status: complete
---

# Phase 9 Plan 4: Search Clear Button & Media Previewer Geometry Fixes Summary

**Decoupled the search clear button's centring from the shared button press affordance (fixing the ~22px mousedown jump), bounded the PDF/image previewer to a 3:4 aspect-locked box that floats extracted text beside it, and gave the mock plugin an opt-in rendition fixture so both fixes are proven end-to-end in a real Playwright browser.**

## Performance

- **Duration:** ~35min
- **Completed:** 2026-08-11
- **Tasks:** 3
- **Files modified:** 13 (6 created, 7 modified)

## Accomplishments

- `SearchBox.svelte`'s clear `Button` is now centred by an `absolute inset-y-0 right-0 flex items-center` wrapper instead of `top-1/2 -translate-y-1/2` — the old technique wrote the same `--tw-translate-y` CSS custom property as `ui/button/button.svelte`'s shared `active:not-aria-[haspopup]:translate-y-px` press affordance, so on mousedown the press value replaced the centring value outright and the button snapped by roughly half its own 44px height. `ui/button/button.svelte` is provably untouched (`git diff --stat` empty), guarded by `search-box-clear.test.ts`'s positive assertion that it still declares its press affordance.
- `DetailPane.svelte`'s `bodyVariant === 'media'` branch replaced its fixed `h-72`/unbounded-width box with `w-full max-w-sm aspect-[3/4]`, applied uniformly to both the PDF `<iframe>` and `<img>` fallback. When `content?.text` is present, the box gets `float-left mr-6 mb-4` inside the existing `overflow-y-auto` scroll container (which already establishes the block-formatting context a float needs — no clearfix) and the text renders as a plain flowing block with typography classes kept byte-identical to the shared `loadedTextBlock` snippet. With no text, the box centres (`mx-auto`) instead of floating.
- `plugins/mock/renditionfixture.go` adds `WEBSPACES_MOCK_RENDITION`, off by default, mirroring `readiness.go`'s established shape exactly. When set, mock item `"1"` alone gains a fake embedded PNG rendition (`FULL` gains a rendition descriptor, `PREVIEW`/`THUMBNAIL` return the embedded bytes); every other item and the fixture-off case are proven byte-identical to the pre-fixture behaviour by `renditionfixture_test.go`.
- Added `FixtureConfigSpec.env` (layered onto `kernel.ts`'s fixed spawn-time env allowlist, never replacing it) so `web/e2e/specs/09-search-clear-and-previewer.spec.ts` can set `WEBSPACES_MOCK_RENDITION=1` on its own hermetic kernel — this is the general mechanism any future spec needing a mock-plugin fixture env var will reuse.
- `09-search-clear-and-previewer.spec.ts` proves both fixes in a real Chromium browser: the clear button's bounding box stays within a small bounded tolerance under a held mousedown (not literal zero-pixel equality — the shared press affordance's own ~1px translate is deliberately preserved); the previewer's rendered box is width-capped at 384px with a 0.75 (3:4) aspect ratio within tolerance; and the extracted text block's top sits above the preview box's bottom, proving the text flows beside the float.

## Task Commits

1. **Task 1: Decouple the search clear button's centring from the shared press affordance** - `957a80e` (feat)
2. **Task 2: Bound the media previewer to an aspect ratio and let text flow around it** - `5c1d699` (feat)
3. **Task 3: Make the media branch browser-drivable, and prove both fixes in a real browser** - `5bc3893` (feat)

_No separate plan-metadata commit — worktree mode; the orchestrator commits SUMMARY.md/STATE.md centrally after the wave._

## Files Created/Modified

- `web/src/lib/components/SearchBox.svelte` - clear button centring technique changed to a non-transformed flex track
- `web/src/lib/components/search-box-clear.test.ts` - source-scan guard over the centring technique and the app-wide press affordance's continued presence
- `web/src/lib/components/DetailPane.svelte` - media branch reworked to a bounded, aspect-locked, conditionally-floated box
- `web/src/lib/components/detail-media-box.test.ts` - source-scan guard over sizing, conditional float, and the flowing text block
- `plugins/mock/renditionfixture.go`, `renditionfixture_test.go` - the `WEBSPACES_MOCK_RENDITION` fixture and its four `<behavior>`-case tests
- `plugins/mock/assets/fixture-rendition.png` - a 313-byte embedded square PNG fixture asset
- `plugins/mock/plugin.go` - `SourcePlugin.renditionFixture` field, `withRenditionFixture` setter, `Fetch` wired to the fixture scoped to one item id
- `plugins/mock/main.go` - reads `WEBSPACES_MOCK_RENDITION` via `os.Getenv`, chains `withRenditionFixture` onto the plugin construction
- `docs/testing.md` - new `WEBSPACES_MOCK_RENDITION` subsection alongside the two existing `WEBSPACES_MOCK_*` fixtures
- `web/e2e/fixtures/config-builder.ts` - `FixtureConfigSpec.env` field
- `web/e2e/fixtures/kernel.ts` - `configSpec.env` layered onto the spawned kernel's env allowlist
- `web/e2e/specs/09-search-clear-and-previewer.spec.ts` - the real-browser proof of both fixes

## Decisions Made

- **e2e clear-button assertion uses a bounded tolerance (`<=2px`), not literal pixel equality.** The plan's own `<behavior>` list (and 09-UI-SPEC.md's must-have "pressing it does not move it") reads as literal zero-movement, but Task 1's own `<action>` text explicitly requires the shared press affordance's ordinary ~1px translate to survive unchanged ("applies exactly the 1px press it was designed to") and explicitly forbids special-casing it off for this one control. A strict-equality e2e assertion would therefore either fail against a correctly-implemented fix or require reintroducing the exact anti-pattern the plan warns against. Chose a tight bound (2px) that comfortably distinguishes the fix (ordinary ~1px press) from the original defect (~22px, half the button's own height) — the honest reading of "identical" given the plan's own explicit press-affordance-preservation instruction.
- **Fixture rendition attached to mock item `"1"` (not a new item).** It already carries non-empty extracted text (`mockFullText["1"]`), letting one item exercise both Fix 9 UI considerations the e2e spec needs (float beside text, and — via any other item — the unfloated no-rendition case) without adding a fifth fixed item to the mock corpus.
- **Added `FixtureConfigSpec.env` (Rule 3 — blocking compile/test dependency).** No existing mechanism let an e2e spec reach a mock-plugin fixture env var on its own kernel process; `kernel.ts`'s spawn env was a fixed, unparameterized allowlist (`PATH`/`HOME`/`XDG_*` only). Layered `configSpec.env` on top of (never replacing) that allowlist, so T-07.1-05's isolation guarantee (no operator env leaks into the test kernel) is structurally unchanged — the new field can only ADD keys the fixture author explicitly names, never override the hermetic identity keys.

## Deviations from Plan

### Auto-fixed Issues

None beyond the decisions above, which are documented as explicit interpretive choices rather than bugs — the plan's own text (Task 1's `<action>`) directly resolves the ambiguity in the `<behavior>`/must-haves wording, so this is a faithful reading, not a deviation from stated intent.

**1. [Rule 3 - Blocking] Added `FixtureConfigSpec.env` to reach the mock plugin's fixture env var**
- **Found during:** Task 3, writing the e2e spec
- **Issue:** Task 3's own `<behavior>` requires "with the rendition fixture enabled for this spec file's kernel" — no existing fixture infrastructure supported passing an arbitrary env var to the spawned kernel process.
- **Fix:** Added `FixtureConfigSpec.env?: Record<string, string>` (config-builder.ts), spread into `kernel.ts`'s `launchKernel`'s `spawn(...)` env object, positioned so the fixed `PATH`/`HOME`/`XDG_*` keys are set AFTER the spread and cannot be overridden by a spec.
- **Files modified:** `web/e2e/fixtures/config-builder.ts`, `web/e2e/fixtures/kernel.ts`
- **Verification:** `npm run check:e2e` (0 errors), the new spec's own successful run against a kernel with `WEBSPACES_MOCK_RENDITION=1` set.
- **Committed in:** `5bc3893` (Task 3 commit)

**Total deviations:** 1 auto-fixed (Rule 3, blocking infrastructure gap) plus two explicit interpretive decisions recorded above (Decisions Made) resolving wording tension inside the plan itself, not disagreements with it.
**Impact on plan:** No scope creep — the env pass-through is strictly load-bearing for Task 3's own stated requirement, and the tolerance-based assertion is the literal application of Task 1's own instruction to preserve the shared press affordance.

## Issues Encountered

- The first e2e run against a freshly-built `bin/topos` returned `404 page not found` for every route — the kernel binary was built BEFORE `npm run build` regenerated the `go:embed`'d `kernel/webui/build/` SPA output. Rebuilding the kernel binary AFTER the web build resolved it. Not a code defect; a local build-order mistake during manual verification (this repo's own `make e2e` target already orders these correctly: web build, then kernel build).
- `npm run build` overwrote/removed the gitignored `kernel/webui/build/.gitkeep` placeholder; restored it via `git show` before committing, since it's the tracked marker keeping that otherwise-empty gitignored directory present in a fresh checkout.

## User Setup Required

None — no external service configuration required. `npm install` was needed once at session start (`web/node_modules` was absent), a one-time environment setup cost, not a plan defect.

## Next Phase Readiness

- Both fixes are permanent Playwright gates (`web/e2e/specs/09-search-clear-and-previewer.spec.ts`), per the repo's standing rule that UI-touching phases extend the e2e suite as part of their definition of done.
- `FixtureConfigSpec.env` is now available to any future e2e spec needing a mock-plugin (or future plugin) fixture env var — no further fixture-infrastructure work needed for that class of need.
- No blockers for 09-05/09-06/09-07.

---
*Phase: 09-ui-polish-and-source-management-rework*
*Completed: 2026-08-11*

## Self-Check: PASSED

All 6 created files verified present on disk; all 3 task commit hashes (`957a80e`, `5c1d699`, `5bc3893`) verified present in `git log --oneline --all`.
