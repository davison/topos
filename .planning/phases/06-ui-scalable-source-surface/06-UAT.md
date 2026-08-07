---
status: diagnosed
phase: 06-ui-scalable-source-surface
source: [06-VERIFICATION.md]
started: 2026-08-06T22:30:00Z
updated: 2026-08-07T09:00:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Search-term highlighting in the rendition iframe (re-test after G-06-1 fix)

Run `make dev`, search a webspace for a word known to appear in an email or a SilverBullet note, open that item, and confirm the word is highlighted amber inside the detail pane's rendered iframe document (email HTML, SilverBullet markdown, Signal chat transcript) AND in the detail-pane title when it matches there. Search for a word that only appears in an item's title and confirm a visible highlight still appears in both the search-results row and the detail pane. Check the stream/search rows use the same amber treatment (not the old barely-visible bold). Clear the search and confirm all highlights disappear.

expected: One unified amber `.search-highlight` treatment on title, body, and row snippets; title-only matches visibly highlighted; byte-identical rendition output when no query.
result: pass

### 2. Search-term highlighting in the plain-text/media variant

Search for a word that appears in a paperless document's extracted text, open it, and confirm the word is highlighted amber below the preview box (the plain-text/media detail-pane variant).

expected: The matched word renders in a highlighted span below the document preview.
result: pass

### 3. Merged source chip — filter, URL round-trip, hover refresh (re-test after G-06-3 fix)

With `make dev` running, click a chip to select it and confirm the chip fills with an obviously visible contrasting blue (the pre-Phase-6 accent blue), with label, health dot, and refresh icon all re-toned to stay readable on the fill. Open the overflow popover (narrow the window if needed) and confirm a selected chip inside the popover reads identically to one inline. Re-confirm the round-1 passes still hold: multi-select narrows the stream, the URL round-trips, hover reveals refresh, refresh does not toggle the filter.

expected: Solid `bg-primary` selected fill, clearly visible at a glance, identical inline and in the popover; filtering behavior unchanged.
result: issue
reported: "pass with some cosmetic-only issues. 1. The chip is larger (height) than it needs to be, maybe 5px too many above and below the text. This additional space does not trigger the click event on the chip which is counter-intuitive. 2. Hovering over the chip correctly shows the refresh button, moving to hover over the refresh button additionally highlights the button - but the background highlight is a rounded corner square (looks odd inside the more oval chip). 3. Clicking a refresh button causes the refresh icon to remain visible even when not hovering the chip. It is hidden only when clicking any other area of the page"
severity: cosmetic

### 4. Chip-row overflow under live window resize

With a webspace loaded and the chip row rendered, narrow the browser window steadily. Confirm chips move off the row into the '+N' trigger as space runs out and the trigger's count rises; widen the window and confirm they return inline and the trigger's count falls or the trigger disappears; confirm that at no width is a chip clipped at the row's trailing edge with no trigger showing, and that 'Clear filters' and 'Refresh all' stay on the row throughout. Then make one source hidden behind the fold unreachable and confirm the trigger's dot turns the destructive tone at that narrowed width.

expected: Single-line row at any instance count and at any window width, including after a post-load resize; overflow popover reachable in two interactions; worst-of health tone surfaced on the trigger at all times.
result: pass

### 5. Deep-link fidelity differentiation

Open a Signal conversation item and confirm its button reads `Show in {source}` with a window icon and an explanatory hover title; open a paperless or email item and confirm its button reads `Open in {source}` with a navigate icon; confirm the small fidelity badge still shows the raw enum value in all three cases.

expected: Two-class icon/verb/title split, badge unchanged (3 raw values).
result: pass

### 6. Stream scrollbar date markers (re-test after G-06-6 fix)

Open a webspace whose stream spans several dates. Confirm the rebuilt ruler reads as a deliberate navigation affordance — its own lane clear of the scrollbar, a faint vertical rail, and a visible major/minor tick hierarchy — not as CSS artifacts. Confirm no tick is clipped at the pane's top or bottom edge, ticks show a pointer cursor and a focus ring when tabbed to, hovering shows the date, clicking jumps that date's first row to the top, the scrollbar thumb still drags, rows underneath still click, and ticks disappear both during a search and when the stream is too short to scroll.

expected: Intentional ruler read (rail + two-grade ticks in their own lane, clearly visible against the background); no edge clipping; native scrollbar and row interactivity undisturbed; markers hidden during search and when the stream does not scroll.
result: pass

## Summary

total: 6
passed: 5
issues: 1
pending: 0
skipped: 0
blocked: 0

## Gaps

- gap_id: G-06-1
  truth: "Matched terms render inside a `<mark>` element with an amber background across all three iframe content shapes; with no search query, output is byte-identical to pre-phase (no mark anywhere)."
  status: resolved
  resolved_by: 06-05 (unify search-match emphasis — shared .search-highlight class across DetailPane title+body, StreamRow, SearchResults; title-only-match recurrence guard in search-emphasis.test.ts)
  reason: "User reported: In the detail pane title, no search term highlight is shown. An amber highlight inside a mark element shows in the detail pane body. In the stream pane, the search term shows as <span class=\"font-semibold\"/> (barely visible)."
  severity: major
  test: 1
  root_cause: "Two independent causes. RC-1: detail-pane title was never in UI-09's implemented scope — DetailPane.svelte:78 renders {item.title} as a plain text binding while highlightText is applied only to the body block (06-01-PLAN.md scoped UI-09 to detailBodyVariant branches; REQUIREMENTS.md:44 wording 'rendered content' was read as body-only). Amplifier: FTS5 indexes title AND preview (kernel/index/schema.go:86-96), so a title-only match renders ZERO highlight anywhere. RC-2: StreamRow.svelte:121 font-semibold is Phase 3 code implementing a recorded design contract (03-UI-SPEC.md:69 — semibold, no highlight colour, to keep the two-weight typography rule); Phase 6 introduced a stronger amber vocabulary and never reconciled the two."
  debug_session: .planning/debug/search-highlight-title-and-stream.md
- gap_id: G-06-3
  truth: "Selected chips carry a clearly visible selected-state treatment (functional multi-select filtering itself works and round-trips through the URL)."
  status: resolved
  resolved_by: 06-06 (selected chip fills border-primary bg-primary with children re-toned to text-primary-foreground; UI-SPEC anatomy/colour table reconciled; source-chip-selected.test.ts recurrence guard)
  reason: "User reported: pass with one caveat: when selected there is no visual cue. Previously the chip changed to a contrasting blue shade. Now there is the merest border highlight (or possibly a drop shadow) on the chip, visible only on the left and right rounded edges."
  severity: minor
  test: 3
  root_cause: "Selected classes use the wrong token: SourceChip.svelte:77 'bg-accent/10 ring-2 ring-accent' — but --accent (#1e293b) is byte-identical to --border, so the ring adds no colour and the 10% wash changes the fill by <2/255 per channel (app.css:77-85 warns the real blue lives in --primary/--ring). Additionally the outset ring is clipped top/bottom by WebspaceHeader.svelte:189's overflow-hidden row (chip is the tallest child), leaving only the semicircular end-cap fragments the user saw. Origin: 06-UI-SPEC.md:45 spells 'ring-2 ring-accent' contradicting its own colour table (line 163: #60a5fa). Pre-Phase-6 treatment was a solid bg-primary (#60a5fa) fill via variant='default'."
  debug_session: .planning/debug/chip-selected-state-visibility.md
- gap_id: G-06-6
  truth: "Scrollbar date ticks read as an intentional, polished UI affordance (functionally the ticks, tooltips, click-to-jump, and search-hiding all work)."
  status: resolved
  resolved_by: 06-07 (ruler rebuilt in its own 12px lane clear of the scrollbar; dedicated --stream-marker/--stream-marker-strong tokens at 3.81:1/3.67:1 computed contrast; rail + major/minor hierarchy; streamScrolls/markerLaneTop edge-safe gating; cursor-pointer + focus-visible ring; marker-overlay.test.ts computed contrast/geometry guard)
  reason: "User reported: pass with caveat: the styling for the ticks is poor, when I first saw them I thought they were CSS artifacts from a broken stylesheet or a faulty page render"
  severity: cosmetic
  test: 6
  root_cause: "Three conjoint causes plus two guaranteed-broken renders. (1) Geometry: overlay inset right-0.5 (2px) puts 7px of each 12px tick ON the ~11px scrollbar — tick and thumb share the same 35%-alpha token so each tick renders two tones that shift while scrolling. (2) Tone: rest tick colour #353d4f is 1.86:1 vs background (below the 3:1 non-text floor) and 1.35:1 from --border — reads as a stray border fragment. (3) Form: no rail, no labels, no major/minor hierarchy, no cursor-pointer, no focus ring. Plus: a tick always renders half-clipped at topPx=0 (candidateMarkers always emits index 0, -translate-y-1/2, no overflow-hidden), and ticks render even when the stream doesn't scroll (no scrollHeight>clientHeight guard). Implementation also diverged from 06-UI-SPEC.md:111 ('left of the scrollbar track'). 06-03-SUMMARY recorded the visual check as human_judgment, never exercised."
  debug_session: .planning/debug/date-marker-tick-styling.md
- gap_id: G-06-3b
  truth: "Selected/hover chip treatment reads as one polished oval control: chip height fits its text, the whole chip surface is clickable, the hover-revealed refresh button's highlight follows the chip's pill geometry, and the refresh icon hides again when the pointer leaves the chip after a refresh click."
  status: failed
  reason: "User reported: pass with some cosmetic-only issues. 1. The chip is larger (height) than it needs to be, maybe 5px too many above and below the text. This additional space does not trigger the click event on the chip which is counter-intuitive. 2. Hovering over the chip correctly shows the refresh button, moving to hover over the refresh button additionally highlights the button - but the background highlight is a rounded corner square (looks odd inside the more oval chip). 3. Clicking a refresh button causes the refresh icon to remain visible even when not hovering the chip. It is hidden only when clicking any other area of the page"
  severity: cosmetic
  test: 3
  root_cause: "Three co-located causes in SourceChip.svelte, all shipped in 06-02 (7687dd6); 06-06 did not introduce them. (1) Height/dead zone: chip height (~54px) is driven by the 44x44 refresh Button (size-11, per 06-UI-SPEC.md:140/161 touch-target floor) plus outer div py-1, but the filter onclick lives only on the inner label button (~20px tall) — the ~17px bands above/below are visually chip but click-dead, and the chip stands ~10px taller than the adjacent h-11 overflow trigger (WebspaceHeader.svelte:207). (2) Square hover highlight: the refresh Button's class override (SourceChip.svelte:116-120) supplies no border-radius, so buttonVariants' base rounded-lg (button.svelte:7) shapes the ghost hover fill as a 44px rounded square inside the rounded-full pill. (3) Sticky icon: reveal is opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 (per 06-UI-SPEC.md:44/51); a mouse click focuses the button, focus persists after pointer leaves, :focus-within pins opacity at 100, and outline-none + focus-visible-only ring hides why — focus moves only when clicking elsewhere."
  artifacts:
    - path: "web/src/lib/components/SourceChip.svelte"
      issue: "onclick on inner button only while outer div + size-11 refresh Button drive height (1); no radius override on refresh Button (2); group-focus-within:opacity-100 reveal pins icon after mouse click (3)"
    - path: "web/src/lib/components/ui/button/button.svelte"
      issue: "source of rounded-lg default and outline-none/focus-visible-only ring (evidence only, no change needed)"
    - path: ".planning/phases/06-ui-scalable-source-surface/06-UI-SPEC.md"
      issue: "lines 44/51 (:focus-within reveal) and 140/161 (44px floor) must be reconciled with the fix — same doc-code drift pattern as G-06-3"
    - path: "web/src/lib/components/WebspaceHeader.svelte"
      issue: "line 207 h-11 overflow trigger is the height reference the chip should match"
  missing:
    - "Shrink the refresh control (size-7/size-8 visual, rounded-full) and/or move the height driver so the chip lands at h-11; stretch the filter button to fill chip height (self-stretch, vertical padding into the button) so the whole surface is clickable; preserve any kept 44px floor via hit-area extension or invoke the spec's desktop-only exception"
    - "Add rounded-full to the refresh Button's class override"
    - "Replace group-focus-within:opacity-100 with a focus-visible-scoped reveal (focus-visible:opacity-100 or group-has-[:focus-visible]:opacity-100) so keyboard reveal stays but mouse-click focus doesn't pin the icon"
    - "Update 06-UI-SPEC.md:44/51 (and 140/161 floor wording if relaxed) in the same plan so spec and code agree"
  debug_session: .planning/debug/chip-pill-polish.md
