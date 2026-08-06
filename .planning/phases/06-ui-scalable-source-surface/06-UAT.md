---
status: diagnosed
phase: 06-ui-scalable-source-surface
source: [06-VERIFICATION.md]
started: 2026-08-06T22:30:00Z
updated: 2026-08-07T00:15:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Search-term highlighting in the rendition iframe

Run `make dev`, search a webspace for a word known to appear in an email or a SilverBullet note, open that item, and confirm the word is highlighted amber inside the detail pane's rendered iframe document (email HTML, SilverBullet markdown, Signal chat transcript). Clear the search and confirm the highlight disappears.

expected: Matched terms render inside a `<mark>` element with an amber background across all three iframe content shapes; with no search query, output is byte-identical to pre-phase (no mark anywhere).
result: issue
reported: "In the detail pane title, no search term highlight is shown. An amber highlight inside a mark element shows in the detail pane body. In the stream pane, the search term shows as <span class=\"font-semibold\"/> (barely visible)."
severity: major

### 2. Search-term highlighting in the plain-text/media variant

Search for a word that appears in a paperless document's extracted text, open it, and confirm the word is highlighted amber below the preview box (the plain-text/media detail-pane variant).

expected: The matched word renders in a highlighted span below the document preview.
result: pass

### 3. Merged source chip — filter, URL round-trip, hover refresh

With `make dev` running, confirm one chip per configured source appears in a single row; click two chips and confirm the stream narrows to both and the URL carries both names; reload and confirm the selection survives; hover a chip and confirm the refresh icon appears and the health tooltip reads as it did before; click the refresh icon and confirm only that source refreshes and the filter does not change.

expected: One merged chip per instance; multi-select toggling narrows the stream and round-trips through the URL; hover/focus reveals refresh; tooltip copy matches Phase 2; refresh click does not also toggle the filter.
result: issue
reported: "pass with one caveat: when selected there is no visual cue. Previously the chip changed to a contrasting blue shade. Now there is the merest border highlight (or possibly a drop shadow) on the chip, visible only on the left and right rounded edges."
severity: minor

### 4. Chip-row overflow under live window resize

With a webspace loaded and the chip row rendered, narrow the browser window steadily. Confirm chips move off the row into the '+N' trigger as space runs out and the trigger's count rises; widen the window and confirm they return inline and the trigger's count falls or the trigger disappears; confirm that at no width is a chip clipped at the row's trailing edge with no trigger showing, and that 'Clear filters' and 'Refresh all' stay on the row throughout. Then make one source hidden behind the fold unreachable and confirm the trigger's dot turns the destructive tone at that narrowed width.

expected: Single-line row at any instance count and at any window width, including after a post-load resize; overflow popover reachable in two interactions; worst-of health tone surfaced on the trigger at all times.
result: pass

### 5. Deep-link fidelity differentiation

Open a Signal conversation item and confirm its button reads `Show in {source}` with a window icon and an explanatory hover title; open a paperless or email item and confirm its button reads `Open in {source}` with a navigate icon; confirm the small fidelity badge still shows the raw enum value in all three cases.

expected: Two-class icon/verb/title split, badge unchanged (3 raw values).
result: pass

### 6. Stream scrollbar date markers

Open a webspace whose stream spans several dates. Confirm thin ticks appear alongside the stream scrollbar, hovering one shows its date, clicking one jumps that date's first row to the top of the pane, the scrollbar thumb itself still drags, and the rows underneath the overlay still click. Then run a search and confirm the ticks disappear while search results are showing.

expected: Clickable, tooltipped date ticks; native scrollbar and row interactivity undisturbed; markers hidden during search.
result: issue
reported: "pass with caveat: the styling for the ticks is poor, when I first saw them I thought they were CSS artifacts from a broken stylesheet or a faulty page render"
severity: cosmetic

## Summary

total: 6
passed: 3
issues: 3
pending: 0
skipped: 0
blocked: 0

## Gaps

- gap_id: G-06-1
  truth: "Matched terms render inside a `<mark>` element with an amber background across all three iframe content shapes; with no search query, output is byte-identical to pre-phase (no mark anywhere)."
  status: failed
  reason: "User reported: In the detail pane title, no search term highlight is shown. An amber highlight inside a mark element shows in the detail pane body. In the stream pane, the search term shows as <span class=\"font-semibold\"/> (barely visible)."
  severity: major
  test: 1
  root_cause: "Two independent causes. RC-1: detail-pane title was never in UI-09's implemented scope — DetailPane.svelte:78 renders {item.title} as a plain text binding while highlightText is applied only to the body block (06-01-PLAN.md scoped UI-09 to detailBodyVariant branches; REQUIREMENTS.md:44 wording 'rendered content' was read as body-only). Amplifier: FTS5 indexes title AND preview (kernel/index/schema.go:86-96), so a title-only match renders ZERO highlight anywhere. RC-2: StreamRow.svelte:121 font-semibold is Phase 3 code implementing a recorded design contract (03-UI-SPEC.md:69 — semibold, no highlight colour, to keep the two-weight typography rule); Phase 6 introduced a stronger amber vocabulary and never reconciled the two."
  artifacts:
    - path: "web/src/lib/components/DetailPane.svelte"
      issue: "line 78 title unhighlighted; reusable .search-highlight class already exists at lines 227-232"
    - path: "web/src/lib/components/StreamRow.svelte"
      issue: "line 121 font-semibold snippet treatment (pre-existing Phase 3 contract); line 68 title also unhighlighted"
  missing:
    - "Apply highlightText to detail-pane title reusing .search-highlight"
    - "Unify kernel <mark>, .search-highlight, and stream snippet treatment on one shared token (explicit decision to overturn 03-UI-SPEC.md:69 contract)"
    - "Test asserting a title-only match renders a visible highlight somewhere"
  debug_session: .planning/debug/search-highlight-title-and-stream.md
- gap_id: G-06-3
  truth: "Selected chips carry a clearly visible selected-state treatment (functional multi-select filtering itself works and round-trips through the URL)."
  status: failed
  reason: "User reported: pass with one caveat: when selected there is no visual cue. Previously the chip changed to a contrasting blue shade. Now there is the merest border highlight (or possibly a drop shadow) on the chip, visible only on the left and right rounded edges."
  severity: minor
  test: 3
  root_cause: "Selected classes use the wrong token: SourceChip.svelte:77 'bg-accent/10 ring-2 ring-accent' — but --accent (#1e293b) is byte-identical to --border, so the ring adds no colour and the 10% wash changes the fill by <2/255 per channel (app.css:77-85 warns the real blue lives in --primary/--ring). Additionally the outset ring is clipped top/bottom by WebspaceHeader.svelte:189's overflow-hidden row (chip is the tallest child), leaving only the semicircular end-cap fragments the user saw. Origin: 06-UI-SPEC.md:45 spells 'ring-2 ring-accent' contradicting its own colour table (line 163: #60a5fa). Pre-Phase-6 treatment was a solid bg-primary (#60a5fa) fill via variant='default'."
  artifacts:
    - path: "web/src/lib/components/SourceChip.svelte"
      issue: "line 77 uses neutral accent token instead of --primary/--ring blue; line 96 hardcodes text-foreground (unreadable on a primary fill)"
    - path: "web/src/lib/components/WebspaceHeader.svelte"
      issue: "line 189 overflow-hidden clips outset ring top/bottom (load-bearing for 06-04 measurement — do not remove)"
    - path: ".planning/phases/06-ui-scalable-source-surface/06-UI-SPEC.md"
      issue: "line 45 class spec contradicts line 163 colour table"
  missing:
    - "Solid bg-primary selected fill (matches prior treatment) or ring-inset with ring-primary"
    - "Thread selected into label span for text-primary-foreground; re-check health-dot tones on fill"
    - "Fix 06-UI-SPEC.md:45 to match the colour table"
    - "Class-string assertion or visual snapshot as recurrence guard"
  debug_session: .planning/debug/chip-selected-state-visibility.md
- gap_id: G-06-6
  truth: "Scrollbar date ticks read as an intentional, polished UI affordance (functionally the ticks, tooltips, click-to-jump, and search-hiding all work)."
  status: failed
  reason: "User reported: pass with caveat: the styling for the ticks is poor, when I first saw them I thought they were CSS artifacts from a broken stylesheet or a faulty page render"
  severity: cosmetic
  test: 6
  root_cause: "Three conjoint causes plus two guaranteed-broken renders. (1) Geometry: overlay inset right-0.5 (2px) puts 7px of each 12px tick ON the ~11px scrollbar — tick and thumb share the same 35%-alpha token so each tick renders two tones that shift while scrolling. (2) Tone: rest tick colour #353d4f is 1.86:1 vs background (below the 3:1 non-text floor) and 1.35:1 from --border — reads as a stray border fragment. (3) Form: no rail, no labels, no major/minor hierarchy, no cursor-pointer, no focus ring. Plus: a tick always renders half-clipped at topPx=0 (candidateMarkers always emits index 0, -translate-y-1/2, no overflow-hidden), and ticks render even when the stream doesn't scroll (no scrollHeight>clientHeight guard). Implementation also diverged from 06-UI-SPEC.md:111 ('left of the scrollbar track'). 06-03-SUMMARY recorded the visual check as human_judgment, never exercised."
  artifacts:
    - path: "web/src/lib/components/StreamDateMarkers.svelte"
      issue: "right-0.5 inset overlaps scrollbar; h-0.5 w-3 tick in thumb token; -translate-y-1/2 top clipping; no rail/label/cursor/focus treatment"
    - path: "web/src/lib/format.ts"
      issue: "dateMarkers (737-753) missing scroll-overflow guard; candidateMarkers (666-685) always emits a marker at topPx=0"
    - path: "web/src/app.css"
      issue: "--scrollbar-thumb 35% alpha is the low-contrast tone source"
    - path: ".planning/phases/06-ui-scalable-source-surface/06-UI-SPEC.md"
      issue: "lines 111/115-117 diverged and under-specified (mandates thumb-token reuse forcing 1.86:1)"
  missing:
    - "Move overlay clear of the scrollbar (right offset >= scrollbar width) into its own lane"
    - "Dedicated --stream-marker token at >=3:1 rest contrast (amend spec's no-new-colour constraint)"
    - "Grouping form: faint rail + major/minor tick hierarchy (and consider labels at major boundaries)"
    - "Clamp/drop the topPx=0 marker; gate markers on actual scroll overflow"
    - "cursor-pointer and real focus-visible ring on tick buttons"
    - "Amend 06-UI-SPEC.md marker section alongside the code"
  debug_session: .planning/debug/date-marker-tick-styling.md
