---
status: complete
phase: 06-ui-scalable-source-surface
source: [06-VERIFICATION.md]
started: 2026-08-07T13:55:00Z
updated: 2026-08-07T14:20:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Source chip pill geometry, hover disc, and refresh reveal (G-06-3b re-test)
Run `make dev`, open a webspace, narrow the window until the '+N' overflow trigger appears.
Confirm a chip is the same height as the trigger beside it, and that clicking anywhere on the
pill — including its top and bottom edges, not just the text line — toggles the filter. Hover
a chip, then hover its refresh icon: confirm the highlight is a circular disc inside the oval,
not a rounded square. Click a refresh icon, then move the pointer off the chip without clicking
anything else: confirm the icon disappears as the pointer leaves (it must not stay pinned
visible). Tab into a chip with the keyboard and confirm the refresh icon becomes visible. While
a source is syncing, confirm the spinning icon stays visible regardless of pointer position.
Re-confirm a selected chip still fills contrasting blue, identically inline and inside the
overflow popover.
expected: The chip reads as one polished 44px pill matching the overflow trigger's height, with its entire surface clickable; the refresh hover highlight is circular; a mouse click on refresh no longer pins the icon after the pointer leaves; keyboard Tab still reveals it; the syncing spinner is unaffected; selected-state fill is unchanged from the prior round's pass.
result: pass

### 2. Search-term highlighting, including title-only match and WR-01 Unicode probe
Run `make dev`, search a webspace for a word known to appear ONLY in an item's title (not its
body), and confirm the word is highlighted amber in both the search-results row title AND the
opened detail-pane title. Then search a word in body text and confirm the results-row snippet
shows the same amber treatment the detail pane uses. Clear the search and confirm no highlight
remains anywhere. If practical, also try a search term adjacent to a Turkish-orthography
capital İ (e.g. a title containing 'İstanbul') and confirm the highlighted span still exactly
covers the matched characters (06-REVIEW.md WR-01 — a client-side `toLowerCase()`-length-divergence
bug is not yet fixed and is not caught by any existing test).
expected: Title-only matches render a visible highlight; snippet, title and detail-pane body share one amber vocabulary; empty query renders byte-identical to pre-phase. The İ case, if exercised, should not be expected to pass — it is a known open bug, included here so it is not silently missed during UAT.
result: pass

### 3. Selected chip fill (re-validated against new pill markup)
Run `make dev`, click a source chip and confirm it fills with an obviously visible contrasting
blue, with label/health-dot/refresh icon all re-toned and legible on the fill. Open the overflow
popover and confirm a selected chip inside it reads identically to one inline.
expected: Solid `bg-primary` selected fill, clearly visible at a glance, identical inline and in the popover.
result: pass

### 4. Stream date-marker ruler
Run `make dev`, open a webspace whose stream spans several dates. Confirm the ticks sit in
their own lane clear of the scrollbar, read as a deliberate ruler, month boundaries are visibly
stronger than day boundaries, no tick is clipped at either edge, hovering shows the date,
clicking jumps that date's first row to the top, the scrollbar thumb still drags, and ticks
disappear both during a search and when the stream is too short to scroll.
expected: The ruler reads as an intentional, polished affordance with a visible major/minor hierarchy, no edge clipping, correct overflow/search gating.
result: pass

## Summary

total: 4
passed: 4
issues: 0
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
  status: resolved
  resolved_by: 06-08 (chip wrapper declares h-11 matching WebspaceHeader's overflow trigger with no padding of its own; filter button self-stretch fills the whole pill; refresh Button override size-8 rounded-full paints a circular hover disc; reveal rescoped from group-focus-within to group-has-[:focus-visible] so mouse-click focus no longer pins the icon while keyboard reveal and syncing force-show survive; 06-UI-SPEC.md reveal wording and 44px-floor statements reconciled; source-chip-pill.test.ts recurrence guard proven to trip on each reintroduced defect)
  reason: "User reported: pass with some cosmetic-only issues. 1. The chip is larger (height) than it needs to be, maybe 5px too many above and below the text. This additional space does not trigger the click event on the chip which is counter-intuitive. 2. Hovering over the chip correctly shows the refresh button, moving to hover over the refresh button additionally highlights the button - but the background highlight is a rounded corner square (looks odd inside the more oval chip). 3. Clicking a refresh button causes the refresh icon to remain visible even when not hovering the chip. It is hidden only when clicking any other area of the page"
  severity: cosmetic
  test: 3
  root_cause: "Three co-located causes in SourceChip.svelte, all shipped in 06-02 (7687dd6); 06-06 did not introduce them. (1) Height/dead zone: chip height (~54px) is driven by the 44x44 refresh Button (size-11, per 06-UI-SPEC.md:140/161 touch-target floor) plus outer div py-1, but the filter onclick lives only on the inner label button (~20px tall) — the ~17px bands above/below are visually chip but click-dead, and the chip stands ~10px taller than the adjacent h-11 overflow trigger (WebspaceHeader.svelte:207). (2) Square hover highlight: the refresh Button's class override (SourceChip.svelte:116-120) supplies no border-radius, so buttonVariants' base rounded-lg (button.svelte:7) shapes the ghost hover fill as a 44px rounded square inside the rounded-full pill. (3) Sticky icon: reveal is opacity-0 group-hover:opacity-100 group-focus-within:opacity-100 (per 06-UI-SPEC.md:44/51); a mouse click focuses the button, focus persists after pointer leaves, :focus-within pins opacity at 100, and outline-none + focus-visible-only ring hides why — focus moves only when clicking elsewhere."
  debug_session: .planning/debug/chip-pill-polish.md
