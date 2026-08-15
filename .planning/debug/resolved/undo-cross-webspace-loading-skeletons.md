---
status: resolved
trigger: "G-13-1 (undo-cross-webspace-loading-skeletons): Clicking the Undo toast button after excluding an item and switching to a different (empty) webspace injects 4 glowing loading-skeleton cards into the second webspace's stream. They never resolve and only clear on a full page reload or clicking Refresh all."
created: 2026-08-15T14:30:00Z
updated: 2026-08-15T15:10:00Z
---

## Current Focus

hypothesis: CONFIRMED — load(gen) called with a stale generation from the undo closure flips loadState to 'loading' BEFORE the generation check, and the post-await generation check then guarantees loadState can never return to 'ready'
test: Complete static trace of the state machine — every observable symptom (4 cards, never resolves, cleared by Refresh-all or reload, toast window 5000ms) matched to a specific line
next_action: Return ROOT CAUSE FOUND (goal: find_root_cause_only — no fix applied)
bug_class: Bohrbug (fully deterministic given the UAT reproduction steps)

reasoning_checkpoint:
  hypothesis: "Undo's onUndo closure calls load(gen) with the navGeneration captured at exclusion time; after a webspace switch that gen is stale, so load() sets loadState='loading' synchronously (before any staleness check), then the post-await `gen !== navGeneration` early-return skips both loadState='ready' and the error path — stranding the stream permanently in the loading state, which StreamList renders as exactly 4 skeleton rows."
  confirming_evidence:
    - "StreamLoadingSkeleton.svelte line 13: `{#each Array(4)}` — exactly 4 skeleton rows, matching '4 glowing cards' verbatim"
    - "StreamList.svelte line 85: skeleton renders purely off `state === 'loading'` — loadState is the only input"
    - "+page.svelte line 876: `if (!quiet) loadState = 'loading'` runs before the await; line 879: `if (gen !== navGeneration) return` runs after — one-way transition for any stale-gen call; catch path (line 900) has the same early return"
    - "onUndo closures pass the captured gen: lines 132-135 (handleExclude), 155-158 (handleInclude), 236-239 (handleBulkPrimary) — introduced by commit 405141d (13-07 WR-01 gap closure)"
    - "toast.ts line 72: duration 5000 — matches the reproduction window"
    - "handleRefreshAll line 1106 calls load(navGeneration) with the LIVE generation — completes normally, sets loadState='ready' — exactly matching 'clears on Refresh all'; full reload remounts and the webspace-keyed effect loads with a matching gen — matching the other reported recovery path"
    - "e2e spec 13-undo-across-webspace-switch.spec.ts lines 52-53 & 99-101 state the false assumption verbatim: 'load(gen) no-ops by design … poll the kernel directly instead of asserting on rendered rows' — the spec never inspects webspace B's rendered stream after the Undo click, which is why the e2e gate passed while human UAT caught it"
  falsification_test: "If loadState were re-set to 'ready' anywhere on the stale-gen path, or if the skeleton rendered from anything other than loadState, the symptom could not persist until Refresh-all. Neither is true: both exits of load() behind the gen check return without touching loadState, and StreamList's skeleton branch reads only `state`."
  fix_rationale: "Root cause is load()'s side-effect-before-guard ordering, not the undo write (which correctly targets the captured webspace and is proven by the passing kernel-side e2e assertions). Making a stale-generation load() a true no-op closes the whole class for every caller."
  blind_spots: "Not executed live in a browser (diagnose-only mode); trace is static. Mitigated by: every one of the five observable symptoms (count=4, glow=skeleton, persistence, both recovery paths, 5000ms window) maps 1:1 to a specific line, and the e2e spec's comment independently documents the exact false assumption."
  candidate_causes:
    - "code: load() writes loadState='loading' before the staleness guard (CONFIRMED, contributing condition 1)"
    - "code: 13-07 WR-01 onUndo closures deliberately pass a stale gen, assuming no-op semantics (CONFIRMED, contributing condition 2)"
    - "data: second webspace being empty (ELIMINATED — visibility amplifier only, see Eliminated)"
    - "environment: browser/timing (ELIMINATED — deterministic; the 5000ms toast window is a precondition of the repro, not a defect)"
  and_gate: "yes — the failure requires BOTH a caller passing a stale generation to non-quiet load() AND load() mutating loadState before checking staleness. Either condition alone is harmless; fixing the ordering in load() closes the class for all callers."

## Symptoms

expected: Undo across a webspace switch reverses the exclusion in the ORIGINAL webspace (the one where the item was excluded), without corrupting the currently-viewed webspace's stream. No loading skeletons should appear in the second webspace, which was empty.
actual: "the toast and button are still reachable and the button can be clicked (pass) but when clicking the button it shows 4 glowing (loading) cards in the stream of the 2nd webspace which requires a reload or 'Refresh all' to clear. Note that the 2nd webspace was empty, this could be relevant"
errors: None reported
reproduction: Exclude an item in webspace A, use WebspaceSwitcher to navigate to webspace B (empty), click Undo on the still-visible toast within its 5000ms lifetime.
started: Discovered during Phase 13 UAT (per-item curation / installable app phase); regression introduced by commit 405141d (13-07 WR-01 gap closure)

## Eliminated

- hypothesis: The second webspace being empty is causal
  evidence: StreamList.svelte line 85 renders the skeleton on `state === 'loading'` regardless of response content — a non-empty webspace B would suffer identical stuck skeletons (replacing its rendered rows). Emptiness only made the 4 injected cards conspicuous.
  timestamp: 2026-08-15T14:42:00Z
- hypothesis: The undo write itself targets the wrong webspace (data corruption in B)
  evidence: setItemMarks(ws, …) uses the webspace captured at mark time (the very thing 405141d fixed); the kernel-side reversal is covered by the passing e2e assertions in 13-undo-across-webspace-switch.spec.ts (excluded_count polls against both A and B); UAT itself reported the reversal half as pass.
  timestamp: 2026-08-15T14:42:00Z
- hypothesis: A network/API failure on the undo path produces the stuck state
  evidence: No errors reported; a rejected onUndo fires markFailureToast ("Couldn't include … — try again."), which the user did not see; and load()'s catch path has the same stale-gen early return anyway, so the failure mode is identical with or without a network error.
  timestamp: 2026-08-15T14:42:00Z

## Evidence

- timestamp: 2026-08-15T14:30:00Z
  checked: .planning/debug/knowledge-base.md (KB-001..KB-007)
  found: All entries are Go kernel lifecycle / process / Playwright-harness defect classes; none match a frontend stream-state/skeleton corruption bug
  implication: No known-pattern shortcut; fresh investigation of web UI undo/stream code
- timestamp: 2026-08-15T14:35:00Z
  checked: web/src/routes/w/[webspace]/+page.svelte — handleExclude/handleInclude/handleBulkPrimary and load()
  found: All three onUndo closures call `await load(gen)` with the navGeneration captured at mark time (lines 132-135, 155-158, 236-239). load() (lines 874-912) sets `loadState = 'loading'` at line 876, synchronously, before the await at line 878; the staleness guard `if (gen !== navGeneration) return` at line 879 (and again at line 900 in catch) runs only AFTER the await and skips every path that could set loadState back to 'ready'/'error'.
  implication: Calling load() with a stale gen is a one-way transition into the loading state — precisely a permanent skeleton
- timestamp: 2026-08-15T14:37:00Z
  checked: web/src/lib/components/StreamLoadingSkeleton.svelte and StreamList.svelte
  found: Skeleton is `Array(4)` rows of `Skeleton class="stream-row-surface …"` (glowing cards); StreamList renders it purely on `state === 'loading'` (line 85), before any response-derived branch
  implication: "4 glowing cards" maps 1:1 to loadState being stuck at 'loading'; response content (empty or not) is irrelevant
- timestamp: 2026-08-15T14:38:00Z
  checked: web/src/lib/toast.ts markSuccessToast
  found: duration 5000 (line 72); Undo action calls onUndo; the toaster lives in +layout.svelte so the toast survives the webspace navigation
  implication: The UAT reproduction window (click Undo after switching, within 5000ms) is exactly the toast contract
- timestamp: 2026-08-15T14:40:00Z
  checked: git log -L on handleExclude
  found: Commit 405141d "feat(13-07): snapshot webspace/navGeneration in mark handlers (WR-01)" changed `load(navGeneration)` to `load(gen)` inside onUndo while (correctly) capturing ws for setItemMarks
  implication: The WR-01 gap closure fixed the wrong-webspace write but introduced this regression — the captured gen was passed to a function that is not safe to call with a stale generation
- timestamp: 2026-08-15T14:43:00Z
  checked: web/e2e/specs/13-undo-across-webspace-switch.spec.ts
  found: Comments at lines 52-53 and 99-101 assert "the snapshotted `gen` is stale post-navigation, so load(gen) no-ops by design — poll the kernel directly instead of asserting on rendered rows". The spec never asserts on webspace B's rendered stream after the Undo click.
  implication: The false "no-op by design" assumption is documented verbatim in the regression spec itself; the spec verified the data-layer half of WR-01 and deliberately skipped the view-layer half — which is why `make e2e` passed while human UAT failed
- timestamp: 2026-08-15T14:44:00Z
  checked: Recovery paths named in the UAT report
  found: handleRefreshAll (line 1106) and a full page reload both invoke load with the CURRENT navGeneration, which completes normally and sets loadState='ready'
  implication: Both reported recovery paths ("reload or 'Refresh all'") are exactly the two paths that call load() with a non-stale generation — closing the evidence loop

## Resolution

root_cause: "In web/src/routes/w/[webspace]/+page.svelte, load(gen) mutates loadState to 'loading' (line 876) BEFORE the stale-generation guard, which only runs after the await (line 879). The 13-07/WR-01 gap closure (commit 405141d) made the three undo closures (handleExclude line 134, handleInclude line 157, handleBulkPrimary line 238) call load(gen) with the generation captured at mark time — deliberately stale after a webspace switch, on the documented-but-false assumption that a stale-gen load() 'no-ops by design'. It does not no-op: it flips the currently-viewed webspace's stream into the loading state (StreamList renders StreamLoadingSkeleton's exactly-4 rows on state==='loading') and the post-await guard then skips every exit that could set loadState back to 'ready' or 'error', stranding the skeletons until something calls load() with the live generation ('Refresh all') or the page is reloaded."
fix: ""
verification: ""
files_changed: []
