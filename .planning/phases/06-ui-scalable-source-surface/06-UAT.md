---
status: testing
phase: 06-ui-scalable-source-surface
source: [06-VERIFICATION.md]
started: 2026-08-06T22:30:00Z
updated: 2026-08-06T22:30:00Z
---

## Current Test

number: 1
name: Search-term highlighting in the rendition iframe
expected: |
  Matched terms render inside a `<mark>` element with an amber background across all three iframe content shapes; with no search query, output is byte-identical to pre-phase (no mark anywhere).
awaiting: user response

## Tests

### 1. Search-term highlighting in the rendition iframe

Run `make dev`, search a webspace for a word known to appear in an email or a SilverBullet note, open that item, and confirm the word is highlighted amber inside the detail pane's rendered iframe document (email HTML, SilverBullet markdown, Signal chat transcript). Clear the search and confirm the highlight disappears.

expected: Matched terms render inside a `<mark>` element with an amber background across all three iframe content shapes; with no search query, output is byte-identical to pre-phase (no mark anywhere).
result: [pending]

### 2. Search-term highlighting in the plain-text/media variant

Search for a word that appears in a paperless document's extracted text, open it, and confirm the word is highlighted amber below the preview box (the plain-text/media detail-pane variant).

expected: The matched word renders in a highlighted span below the document preview.
result: [pending]

### 3. Merged source chip — filter, URL round-trip, hover refresh

With `make dev` running, confirm one chip per configured source appears in a single row; click two chips and confirm the stream narrows to both and the URL carries both names; reload and confirm the selection survives; hover a chip and confirm the refresh icon appears and the health tooltip reads as it did before; click the refresh icon and confirm only that source refreshes and the filter does not change.

expected: One merged chip per instance; multi-select toggling narrows the stream and round-trips through the URL; hover/focus reveals refresh; tooltip copy matches Phase 2; refresh click does not also toggle the filter.
result: [pending]

### 4. Chip-row overflow under live window resize

With a webspace loaded and the chip row rendered, narrow the browser window steadily. Confirm chips move off the row into the '+N' trigger as space runs out and the trigger's count rises; widen the window and confirm they return inline and the trigger's count falls or the trigger disappears; confirm that at no width is a chip clipped at the row's trailing edge with no trigger showing, and that 'Clear filters' and 'Refresh all' stay on the row throughout. Then make one source hidden behind the fold unreachable and confirm the trigger's dot turns the destructive tone at that narrowed width.

expected: Single-line row at any instance count and at any window width, including after a post-load resize; overflow popover reachable in two interactions; worst-of health tone surfaced on the trigger at all times.
result: [pending]

### 5. Deep-link fidelity differentiation

Open a Signal conversation item and confirm its button reads `Show in {source}` with a window icon and an explanatory hover title; open a paperless or email item and confirm its button reads `Open in {source}` with a navigate icon; confirm the small fidelity badge still shows the raw enum value in all three cases.

expected: Two-class icon/verb/title split, badge unchanged (3 raw values).
result: [pending]

### 6. Stream scrollbar date markers

Open a webspace whose stream spans several dates. Confirm thin ticks appear alongside the stream scrollbar, hovering one shows its date, clicking one jumps that date's first row to the top of the pane, the scrollbar thumb itself still drags, and the rows underneath the overlay still click. Then run a search and confirm the ticks disappear while search results are showing.

expected: Clickable, tooltipped date ticks; native scrollbar and row interactivity undisturbed; markers hidden during search.
result: [pending]

## Summary

total: 6
passed: 0
issues: 0
pending: 6
skipped: 0
blocked: 0

## Gaps
