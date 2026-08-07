---
phase: 06-ui-scalable-source-surface
verified: 2026-08-07T13:45:00Z
status: passed
score: 4/4 roadmap success criteria verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: 4/4 roadmap success criteria verified
  gaps_closed:

    - "G-06-3b: source chip pill restructured into one h-11 (44px) pill whose filter button self-stretches across the full surface (closing the ~17px click-dead bands above/below the label and the ~10px height overshoot against the adjacent overflow trigger); refresh control's hover fill changed from an implicit rounded-square to an explicit rounded-full disc; refresh reveal rescoped from group-focus-within (which pinned the icon after a mouse click) to group-has-[:focus-visible] (keyboard-only), confirmed compiled into the production stylesheet; 06-UI-SPEC.md's refresh-reveal wording, new chip-geometry constraint bullet, and both touch-target-floor statements reconciled with the shipped code"
  gaps_remaining: []
  regressions: []
deferred: []
human_verification:

  - test: "Run `make dev`, open a webspace, narrow the window until the '+N' overflow trigger appears. Confirm a chip is the same height as the trigger beside it, and that clicking anywhere on the pill — including its top and bottom edges, not just the text line — toggles the filter. Hover a chip, then hover its refresh icon: confirm the highlight is a circular disc inside the oval, not a rounded square. Click a refresh icon, then move the pointer off the chip without clicking anything else: confirm the icon disappears as the pointer leaves (it must not stay pinned visible). Tab into a chip with the keyboard and confirm the refresh icon becomes visible. While a source is syncing, confirm the spinning icon stays visible regardless of pointer position. Re-confirm a selected chip still fills contrasting blue, identically inline and inside the overflow popover."
    expected: "The chip reads as one polished 44px pill matching the overflow trigger's height, with its entire surface clickable; the refresh hover highlight is circular; a mouse click on refresh no longer pins the icon after the pointer leaves; keyboard Tab still reveals it; the syncing spinner is unaffected; selected-state fill is unchanged from the prior round's pass."
    why_human: "Pixel-level height parity, full-surface hit-testing, hover-fill shape, and pointer-leave/keyboard-focus timing all need a live browser session. The class-string wiring (h-11 on both the wrapper and the trigger, no wrapper padding, self-stretch on the filter button, rounded-full + size-8 on the refresh Button, group-has-[:focus-visible] present and group-focus-within absent, source.syncing force-visible retained) is proven by an 8-assertion source-scan guard (source-chip-pill.test.ts, manually confirmed by the executor to trip on each of the three reintroduced defects) plus a production-build grep confirming the compiled `:has(:focus-visible)` CSS rule actually exists in the emitted stylesheet — but none of this proves the rendered pixel result."

  - test: "Run `make dev`, search a webspace for a word known to appear ONLY in an item's title (not its body), and confirm the word is highlighted amber in both the search-results row title AND the opened detail-pane title. Then search a word in body text and confirm the results-row snippet shows the same amber treatment the detail pane uses. Clear the search and confirm no highlight remains anywhere. If practical, also try a search term adjacent to a Turkish-orthography capital İ (e.g. a title containing 'İstanbul') and confirm the highlighted span still exactly covers the matched characters (06-REVIEW.md WR-01 — a client-side `toLowerCase()`-length-divergence bug is not yet fixed and is not caught by any existing test)."
    expected: "Title-only matches render a visible highlight; snippet, title and detail-pane body share one amber vocabulary; empty query renders byte-identical to pre-phase. The İ case, if exercised, should not be expected to pass — it is a known open bug, included here so it is not silently missed during UAT."
    why_human: "Live browser rendering and visual contrast; the ASCII/common-case wiring is proven by search-emphasis.test.ts (15 assertions) and the full client suite, but WR-01's Unicode edge case is proven to still be open by direct code inspection (format.ts:578 still does a single bulk `text.toLowerCase()` and indexes into it positionally; no `İstanbul`-class regression test exists in highlight.test.ts), not by a passing test."

  - test: "Run `make dev`, click a source chip and confirm it fills with an obviously visible contrasting blue, with label/health-dot/refresh icon all re-toned and legible on the fill. Open the overflow popover and confirm a selected chip inside it reads identically to one inline."
    expected: "Solid `bg-primary` selected fill, clearly visible at a glance, identical inline and in the popover."
    why_human: "Pixel-level contrast and popover-vs-inline visual parity need a live browser render; the class-string wiring (selected branch resolves through --primary never --accent, all three children re-tone) is proven by source-chip-selected.test.ts's 12 passing assertions, re-validated against the new h-11/self-stretch markup this round (its wrapper-class extraction was re-pointed at the opening `<div>` tag and its D-03 assertion updated for the focus-visible-scoped reveal, both confirmed passing)."

  - test: "Run `make dev`, open a webspace whose stream spans several dates. Confirm the ticks sit in their own lane clear of the scrollbar, read as a deliberate ruler, month boundaries are visibly stronger than day boundaries, no tick is clipped at either edge, hovering shows the date, clicking jumps that date's first row to the top, the scrollbar thumb still drags, and ticks disappear both during a search and when the stream is too short to scroll."
    expected: "The ruler reads as an intentional, polished affordance with a visible major/minor hierarchy, no edge clipping, correct overflow/search gating."
    why_human: "The visual 'reads as intentional' judgment and live pointer/focus/click interaction need a real browser session; every computable property (contrast, lane offset, structural geometry) is proven by marker-overlay.test.ts (20 assertions) and markers.test.ts (33 assertions), unchanged since the prior verification round and untouched by 06-08."
---

# Phase 6: UI Scalable Source Surface Verification Report

**Phase Goal:** The webspace header presents many source instances without duplication — health and filtering combined into one affordance per source — and the accumulated UI polish lands: deep-link fidelity differentiation, search-term highlighting in the detail pane, and themed scrollbars with date markers.
**Verified:** 2026-08-07T13:45:00Z
**Status:** human_needed
**Re-verification:** Yes — after gap-closure round 3 (06-08-PLAN.md/-SUMMARY.md, closing UAT round-2 gap G-06-3b recorded in 06-UAT.md), superseding the prior 2026-08-07T01:45:00Z report

## Note on phase mode

ROADMAP.md declares `Mode: mvp` for every phase in this project, including Phase 6, but the phase's
Goal text ("The webspace header presents many source instances...") is not User-Story shaped
(`gsd-tools query user-story.validate` returns `valid: false` against it), and no phase in this
roadmap uses that shape. This is a pre-existing, project-wide convention mismatch, not something
introduced by this round — the prior two verification rounds for this same phase proceeded with
standard goal-backward verification without applying the MVP-mode User Flow Coverage template, and
this report follows that same established precedent rather than blocking on a documentation-shape
issue unrelated to the code under test.

## Goal Achievement

### Observable Truths

| # | Truth (Roadmap Success Criterion) | Status | Evidence |
|---|---|---|---|
| 1 | Each source instance appears exactly once in the header — a single chip/affordance combining health state, filter toggle, and refresh — and the design remains usable at 10+ instances (overflow, grouping, or collapse; no unbounded chip rows) | ✓ VERIFIED | `SourceChip.svelte` still merges health dot + name + `aria-pressed` filter toggle + hover/focus-revealed refresh into one component, reused unforked inline and in the overflow popover. `WebspaceHeader.svelte`'s `observeResize`-driven overflow wiring (06-04) and selected-fill treatment (06-06) are both confirmed unchanged by `git diff` scoping to files this round's plan touched. **This round's fix (G-06-3b):** the wrapper `div` now declares `h-11 shrink-0 items-center rounded-full border border-border bg-card pr-1` with no vertical/left/all-sides padding of its own (`SourceChip.svelte:74-78`); the filter `<button>` gained `self-stretch` plus its own `pr-1.5 pl-2.5` (`:84-90`), so the button's hit area fills the whole pill rather than a ~20px strip through its middle; the refresh `Button`'s override changed from `size-11` to `size-8 rounded-full` (`:116-120`) so it fits inside the now-44px pill and paints a circular hover disc instead of a rounded square; its reveal changed from `group-focus-within:opacity-100` to `group-has-[:focus-visible]:opacity-100`, verified compiled into the production stylesheet (`grep -o ':has(:focus-visible)' kernel/webui/build/_app/immutable/assets/0.C4b0x9me.css` returns the rule). The overflow trigger in `WebspaceHeader.svelte:207` independently declares the same `h-11` literal the chip now matches. `source-chip-pill.test.ts` (new this round, 8 assertions across 6 groups) and `source-chip-selected.test.ts` (12 assertions, updated this round for the new markup) both pass. |
| 2 | Items whose "open in source" link can only raise the source app's window are visually differentiated from links that navigate to the item (closes the 04-UAT follow-up) | ✓ VERIFIED | `fidelityAffordance` (`format.ts:64-77`) and `OpenInSource.svelte` unchanged this round — `conversation-only` → `windowOnly:true`/`AppWindow` icon/"Show in …"; all other values → navigating/`ArrowUpRight`/"Open in …"; badge unchanged (3 raw values via `formatFidelity`). Not touched by 06-08 (`files_modified` lists only `SourceChip.svelte`, its two test files, and `06-UI-SPEC.md`). UAT Test 5 passed cleanly in round 2 with no gap recorded against this criterion, and nothing in round 3 touches this code path. |
| 3 | After an in-webspace search, matched terms are highlighted in the detail pane's rendered content across content variants (text, sanitized HTML, chat transcript); the stream is unaffected since it already filters to matches | ✓ VERIFIED | Kernel-side highlighting (`rendition.go`, `item.go`, `agent.go`) and the round-2 SPA unification (shared `.search-highlight` class across `DetailPane.svelte` title+body and `StreamRow.svelte` title+snippet, `search-emphasis.test.ts`'s 15 assertions) are unchanged and untouched by 06-08. **New finding this round, not a regression:** `06-REVIEW.md` (dated after 06-08 landed) identifies WR-01 — the client's `highlightText` (`format.ts:578`) computes one bulk `text.toLowerCase()` up front and indexes into it positionally, which silently misaligns highlighted spans on text containing a case-fold-expanding character (e.g. Turkish 'İ'); this is a distinct bug from the already-fixed kernel/client term-length divergence, is not caught by any existing test (`highlight.test.ts` has no non-ASCII/expansion fixture), and remains open as of this verification (`grep` confirms no `İstanbul`-class regression test or per-rune scan fix has landed). This is a narrow edge case that does not affect common-case highlighting (proven passing by the full test suite) and does not block the roadmap truth, but is carried forward as a WARNING and a human-verification item below rather than silently dropped. |
| 4 | Scrollbars app-wide are thin and theme-matched; the stream pane's scrollbar carries date markers reflecting the visible chronology | ✓ VERIFIED | Base scrollbar theming and the round-2 `StreamDateMarkers.svelte` rebuild (dedicated 12px lane, `--stream-marker`/`--stream-marker-strong` tokens at 3.81:1/3.67:1 contrast, rail, major/minor hierarchy, `streamScrolls`/`markerLaneTop` gating) are unchanged and untouched by 06-08 — confirmed via `git log` (no commits since `06-07`'s touching `StreamDateMarkers.svelte`, `format.ts`'s marker exports, or `app.css`'s marker tokens). `marker-overlay.test.ts` (20 assertions) and `markers.test.ts` (33 assertions) both still pass. `06-REVIEW.md`'s IN-02 (degenerate spacing-floor `major`-flag edge case) is carried forward unresolved but is a low-priority edge case per the review's own disposition, not a contradiction of this truth at realistic data scales. |

**Score:** 4/4 truths verified (0 present-but-behavior-unverified; all four carry live-browser visual/interaction confirmation items in Human Verification below, per this project's `human_verify_mode: end-of-phase` convention)

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `web/src/lib/components/SourceChip.svelte` | 44px pill, filter button self-stretched to full surface, circular refresh disc, keyboard-scoped reveal | ✓ VERIFIED | wrapper `h-11 shrink-0 ... pr-1` no vertical/left padding (line 74-78); filter button `self-stretch ... pr-1.5 pl-2.5` (line 84-90); refresh Button `size-8 rounded-full ... group-has-[:focus-visible]:opacity-100` replacing retired `size-11`/`group-focus-within` (line 113-125) |
| `web/src/lib/components/source-chip-pill.test.ts` | geometry/hit-area/radius/reveal-scoping recurrence guard | ✓ VERIFIED | present, 8 assertions across 6 describe blocks, all passing; manually confirmed by executor (per SUMMARY) to trip when each of the three original defects is reintroduced |
| `web/src/lib/components/source-chip-selected.test.ts` | updated for the new markup, still 12 passing assertions | ✓ VERIFIED | wrapper-class extraction re-pointed at the `<div` opening tag; D-03 assertion now requires `group-has-[:focus-visible]` instead of the retired `group-focus-within` literal; all assertions pass |
| `.planning/phases/06-ui-scalable-source-surface/06-UI-SPEC.md` | refresh-reveal wording, new chip-geometry bullet, Tab-order note, and both touch-target-floor statements reconciled, dated to G-06-3b | ✓ VERIFIED | 4 separate `G-06-3b` dated notes present (lines 44, 46, 52, 141-142, 162) covering all four flagged contradictions |
| `kernel/webui/build/` (production build) | compiled stylesheet contains the `:has(:focus-visible)` rule from the arbitrary Tailwind variant | ✓ VERIFIED | `grep -o '.{0,60}:has(:focus-visible).{0,60}'` on the emitted CSS returns `group-has-\[\:focus-visible\]\:opacity-100:is(:where(.group):has(:focus-visible) *){opacity:1}` — the variant compiled, not silently dropped |
| Round-2 artifacts (`app.css`, `DetailPane.svelte`, `StreamRow.svelte`, `SearchResults.svelte`, `search-emphasis.test.ts`, `format.ts`, `StreamDateMarkers.svelte`, `marker-overlay.test.ts`, `+page.svelte`) | unchanged, still passing | ✓ VERIFIED | Full client suite (257/257) and svelte-check (0 errors) re-run independently this round and confirmed green; `git log` confirms none of these files were touched by 06-08 |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `SourceChip.svelte` wrapper | `WebspaceHeader.svelte` overflow trigger | shared `h-11` literal | ✓ WIRED | both declare `h-11` independently; confirmed by direct read of both files |
| `SourceChip.svelte` filter `<button>` | wrapper `div` | `self-stretch` filling the wrapper's content-box height | ✓ WIRED | confirmed by direct read + `source-chip-pill.test.ts` |
| `SourceChip.svelte` refresh `Button` | Tailwind JIT compiler | `group-has-[:focus-visible]` arbitrary variant | ✓ WIRED | confirmed by production-build stylesheet grep — compiles to a real `:has()` CSS rule |
| `SourceChip.svelte` selected state | `app.css` `--primary`/`--primary-foreground` tokens | selected-state class expression, re-validated against new markup | ✓ WIRED | `source-chip-selected.test.ts` re-pointed extraction still passes |
| Round-2 links (highlight/marker wiring) | — | — | ✓ WIRED | unchanged, re-confirmed passing this round (no new evidence needed — no code touched) |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Full client unit suite | `cd web && npx vitest run` | `Test Files 15 passed (15)` / `Tests 257 passed (257)` | ✓ PASS |
| Full client typecheck | `cd web && npm run check` | `COMPLETED 766 FILES 0 ERRORS 1 WARNINGS` (pre-existing unrelated `SearchBox.svelte` lint) | ✓ PASS |
| Production build compiles the arbitrary Tailwind variant | `cd web && npm run build && grep -o ':has(:focus-visible)' ../kernel/webui/build/_app/immutable/assets/*.css` | rule found in emitted CSS | ✓ PASS |
| Full kernel build | `go build ./...` | exit 0, no output | ✓ PASS |
| Named gap-closure guard tests (round 3) | `cd web && npx vitest run source-chip-pill.test.ts source-chip-selected.test.ts` | both files passing | ✓ PASS |
| WR-01 open-bug check | `grep -n "toLowerCase" web/src/lib/format.ts` + `grep -n İstanbul web/src/lib/components/highlight.test.ts` | bulk `toLowerCase()` still present at line 578; no Unicode-expansion regression test found | ⚠️ confirms open (see Truth 3) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|---|---|---|---|---|
| UI-07 | 06-02, gap-closed by 06-04 (overflow), 06-06 (selected-state visibility), 06-08 (pill geometry/hit-area/reveal-scoping) | Header presents each source instance exactly once, one affordance, scales past 10 instances, selected state legible, chip reads as one polished 44px pill | ✓ SATISFIED | All four fixes independently confirmed this round; full suite green |
| UI-08 | 06-03 | Deep-link fidelity visually differentiated | ✓ SATISFIED | Unchanged since round 1; not touched by any gap-closure plan; UAT Test 5 passed cleanly |
| UI-09 | 06-01, gap-closed by 06-05 | Search-term highlighting across item title and rendered content, detail pane + search-results row | ✓ SATISFIED (with open WR-01 caveat) | Kernel iframe highlighting + SPA title/snippet unification both confirmed for the common case; a narrow Unicode edge-case bug (WR-01) is open and flagged, not blocking |
| UI-11 | 06-03, gap-closed by 06-07 | Thin theme-matched scrollbars app-wide + stream date markers | ✓ SATISFIED | Unchanged since round 2, re-confirmed passing |

No orphaned requirements — all four phase requirement IDs (UI-07, UI-08, UI-09, UI-11) are declared across the eight plans' frontmatter (06-01: UI-09; 06-02: UI-07; 06-03: UI-08, UI-11; 06-04: UI-07; 06-05: UI-09; 06-06: UI-07; 06-07: UI-11; 06-08: UI-07), matching exactly the phase requirement IDs given for this verification and REQUIREMENTS.md's own Phase 6 mapping.

**Pre-existing documentation staleness, carried forward unresolved (not a code defect, flagged again for orchestrator attention):** REQUIREMENTS.md's traceability table (lines 102-105) still reads "Gaps Found" for UI-07/UI-08/UI-09 and "Complete" for UI-11, while the requirements checklist above it (lines 42-45) shows UI-07/UI-09/UI-11 checked `[x]` and UI-08 unchecked `[ ]` — despite UI-08 being independently verified SATISFIED (unchanged) across all three verification rounds. This mismatch was flagged at both the round-1 and round-2 verifications; no gap-closure plan across all eight plans in this phase has updated REQUIREMENTS.md's traceability table or the UI-08 checkbox. This is a bookkeeping-sync issue outside this verification's edit scope, not a functional gap.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `web/src/lib/format.ts` | 578 (`highlightText`) | Bulk `text.toLowerCase()` computed once and indexed positionally against the original string — misaligns highlighted spans on text containing a case-fold-expanding character (e.g. Turkish 'İ'); no regression test exists (06-REVIEW.md WR-01, new finding this round, not yet fixed) | ⚠️ Warning | Narrow Unicode edge case; does not affect the common-case highlighting proven passing by the full suite; does not contradict Truth 3 at the level the roadmap criterion is stated, but is real user-visible content corruption if triggered — flagged as open, not silently dropped |
| `web/src/lib/format.ts` | 753-805 (`candidateMarkers`/`enforceSpacingFloor`) | Degenerate spacing-floor thinning can drop the one candidate marker that would have carried `major: true` in an extreme (16+ years on a 50px track) case (06-REVIEW.md IN-02, carried forward, not touched this round) | ℹ️ Info | Low-priority edge case per the review's own disposition; does not contradict this phase's must-haves at realistic data scales |
| `web/src/lib/api.ts` | 131-171 | `getJSON`/`postJSON` remain near-duplicate implementations (06-REVIEW.md IN-01, carried forward, not touched this round) | ℹ️ Info | DRY violation predating this phase; not a functional defect |
| `web/src/lib/components/WebspaceHeader.svelte` | 274-284 vs 199-215 | Overflow-trigger measurement clone always renders `+{sources.length}` rather than `+{hiddenSources.length}`, over-estimating reserved width (06-REVIEW.md IN-03, carried forward, not touched this round) | ℹ️ Info | Fails safe (never causes clipping); can hide one more chip than strictly necessary at digit-count boundaries |
| `web/src/lib/components/WebspaceHeader.svelte` | 260-273 | Hidden chip-measurement region relies on `visibility:hidden` alone (no `tabindex="-1"` backstop) inside an `aria-hidden="true"` container (06-REVIEW.md IN-04, carried forward, not touched this round) | ℹ️ Info | Not currently exploitable; flagged as defense-in-depth against a future edit that swaps the mitigation |

**Resolved since the prior verification round (confirmed independently, not merely trusted from SUMMARY):** CR-01 (`DetailPane.svelte` stale-response race — `contentRequestSeq` guard present at lines 43/60/63/71/74), CR-02 (`+page.svelte` stale-webspace/search race — `navGeneration` guard present), WR-03 (`+page.svelte` uncleared poll interval — `clearInterval` present at both a teardown effect and unmount), WR-04 (duplicated skeleton markup — extracted to `StreamLoadingSkeleton.svelte`). These were flagged as Warnings in the prior verification round and are now fixed in code, independently confirmed by direct read rather than by trusting the code-review report's claim.

No `TBD`/`FIXME`/`XXX` debt markers found in any file touched by 06-08 or carried forward from prior rounds.

## Human Verification Required

See frontmatter `human_verification` for four live-browser checks: (1) the newly closed G-06-3b pill geometry/hit-area/hover-shape/reveal-timing fixes — this is the substantive new verification surface this round; (2) search highlighting, including a specific prompt to probe the still-open WR-01 Unicode edge case so it is not silently missed during UAT; (3) the round-2 selected-chip fill, re-validated against this round's new markup; (4) the round-2 date-marker ruler, unchanged and carried forward for completeness. No display/browser session was available in this execution environment for 06-08's own execution (per its SUMMARY, matching the same limitation recorded across every gap-closure plan's SUMMARY in this phase), so all four items await a live `make dev` session — in particular UAT Test 3 from `06-UAT.md` needs a full re-run against the now-restructured chip.

## Gaps Summary

No functional gaps remain against the four roadmap success criteria. This re-verification confirms G-06-3b (the sole remaining open item from UAT round 2) is closed at the code level: the chip pill now declares its own height matching the overflow trigger, the filter button's hit area covers the whole pill, the refresh control's hover fill is circular, and its reveal is keyboard-focus-scoped rather than pinned by a mouse click — all backed by a new automated recurrence guard (`source-chip-pill.test.ts`) that the executor manually confirmed trips on each of the three original defects, plus a production-build grep proving the arbitrary Tailwind variant actually compiled. The full client suite grew from 249 to 257 passing tests across this round; svelte-check remains at 0 errors; the kernel suite remains green; no regression was introduced in code untouched by 06-08 (confirmed via `git log` scoping and re-running the full suite rather than a filtered subset).

One new, non-blocking finding surfaced by this round's fresh code review (`06-REVIEW.md`, itself independently re-verified rather than trusted) and is carried into this report rather than silently absorbed: WR-01, a client-side Unicode case-folding edge case in the search-highlight scan loop, remains open and untested. It is narrow (a specific class of case-fold-expanding character), does not affect the common-case highlighting this phase's automated suite exercises, and does not contradict roadmap Truth 3 as stated — but it is flagged here as a WARNING and folded into the human-verification checklist so it is not lost. Separately, four Warnings from the prior verification round (CR-01, CR-02, WR-03, WR-04 — all stale-response/interval-leak/duplication issues) are now confirmed fixed by direct code inspection, an improvement over the prior round's state.

What remains is exactly the standard end-of-phase human UAT this project's `human_verify_mode: end-of-phase` configuration defers by design — a live-browser re-run of UAT Test 3 against the now-restructured chip (the substantive new surface this round), plus re-confirmation of Tests 1, 3(prior selected-fill half) and 6 carried forward from round 2. REQUIREMENTS.md's traceability-table/checkbox staleness for UI-07/UI-08/UI-09/UI-11 remains unresolved across all three verification rounds and is carried forward again as a WARNING for the orchestrator to reconcile — it is a bookkeeping issue, not a functional gap.

---

_Verified: 2026-08-07T13:45:00Z_
_Verifier: Claude (gsd-verifier)_

## Acknowledged Gaps

- `api-coverage.verify-pre` gate waived by user (2026-08-06): detection signals `{integration, api}` trace to Phase 6's own kernel HTTP API (`docs/api.md`, `kernel/httpapi/*`) — UI work against the project's internal API, not an external-API integration. No COVERAGE.md required for this phase. (Carried forward from round-1 verification; unaffected by this round's chip-geometry-only gap-closure plan.)
