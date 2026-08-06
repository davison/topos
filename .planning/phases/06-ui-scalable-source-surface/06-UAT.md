---
status: complete
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
  artifacts: []  # Filled by diagnosis
  missing: []    # Filled by diagnosis
- gap_id: G-06-3
  truth: "Selected chips carry a clearly visible selected-state treatment (functional multi-select filtering itself works and round-trips through the URL)."
  status: failed
  reason: "User reported: pass with one caveat: when selected there is no visual cue. Previously the chip changed to a contrasting blue shade. Now there is the merest border highlight (or possibly a drop shadow) on the chip, visible only on the left and right rounded edges."
  severity: minor
  test: 3
  artifacts: []  # Filled by diagnosis
  missing: []    # Filled by diagnosis
- gap_id: G-06-6
  truth: "Scrollbar date ticks read as an intentional, polished UI affordance (functionally the ticks, tooltips, click-to-jump, and search-hiding all work)."
  status: failed
  reason: "User reported: pass with caveat: the styling for the ticks is poor, when I first saw them I thought they were CSS artifacts from a broken stylesheet or a faulty page render"
  severity: cosmetic
  test: 6
  artifacts: []  # Filled by diagnosis
  missing: []    # Filled by diagnosis
