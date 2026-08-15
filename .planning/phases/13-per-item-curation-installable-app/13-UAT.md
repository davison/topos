---
status: resolved
phase: 13-per-item-curation-installable-app
source: [13-VERIFICATION.md]
started: 2026-08-15T01:15:00Z
updated: 2026-08-15T14:45:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Real human timing feasibility of the undo-across-webspace-switch window
expected: A human can comfortably complete exclude → WebspaceSwitcher switch → click Undo inside the toast's real 5000ms window with normal reaction time; the Undo button is still visible and clickable when reached, and the undo correctly reverses the exclusion in the ORIGINAL webspace (not the one navigated to).
result: issue
reported: "the toast and button are still reachable and the button can be clicked (pass) but when clicking the button it shows 4 glowing (loading) cards in the stream of the 2nd webspace which requires a reload or 'Refresh all' to clear. Note that the 2nd webspace was empty, this could be relevant"
severity: major

## Summary

total: 1
passed: 0
issues: 1
pending: 0
skipped: 0
blocked: 0

## Gaps

- gap_id: G-13-1
  truth: "Clicking Undo after switching webspaces reverses the exclusion in the original webspace without corrupting the currently-viewed webspace's stream; no stale loading skeletons appear in the second webspace"
  status: resolved
  reason: "User reported: the toast and button are still reachable and the button can be clicked (pass) but when clicking the button it shows 4 glowing (loading) cards in the stream of the 2nd webspace which requires a reload or 'Refresh all' to clear. Note that the 2nd webspace was empty, this could be relevant"
  severity: major
  test: 1
  root_cause: "load() in web/src/routes/w/[webspace]/+page.svelte sets loadState = 'loading' synchronously BEFORE its stale-generation guard (guard only runs after the awaited getStream). The 13-07/WR-01 gap closure (commit 405141d) made the three onUndo closures (handleExclude, handleInclude, handleBulkPrimary) call load(gen) with the generation captured at mark time — deliberately stale after a webspace switch — on the false assumption that a stale-gen load() no-ops. It doesn't: it flips the currently-viewed webspace into the loading state, then the post-await guard (gen !== navGeneration) returns early and skips every exit that would restore loadState to 'ready'/'error', stranding the 4 skeleton cards (Array(4) in StreamLoadingSkeleton.svelte) permanently."
  artifacts:
    - path: "web/src/routes/w/[webspace]/+page.svelte"
      issue: "load() performs its loadState = 'loading' side effect (line ~876) before the staleness guard (~line 900); three onUndo closures pass a captured, potentially-stale generation (lines ~132-135, ~155-158, ~236-239)"
    - path: "web/e2e/specs/13-undo-across-webspace-switch.spec.ts"
      issue: "Encodes the false 'load(gen) no-ops by design' assumption and polls the kernel instead of asserting on webspace B's rendered stream after the Undo click — the coverage gap that let this ship"
  missing:
    - "Add a true no-op guard at the top of load(): if (gen !== navGeneration) return; BEFORE the loadState = 'loading' write — closes the class for every caller"
    - "Optionally switch the three onUndo closures to load(gen, { quiet: true }) so same-webspace undo refreshes without a skeleton flash and stale-gen undo never touches loadState"
    - "Extend 13-undo-across-webspace-switch.spec.ts to assert on webspace B's rendered stream after the Undo click (row set unchanged, no skeleton visible)"
  debug_session: ".planning/debug/undo-cross-webspace-loading-skeletons.md"
