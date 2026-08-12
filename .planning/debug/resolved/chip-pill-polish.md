---
status: resolved
trigger: "G-06-3b: Source chip (SourceChip.svelte) has three cosmetic/interaction defects from Phase 6 UAT round 2: dead vertical padding (non-clickable), square-ish hover highlight on refresh button inside pill chip, refresh icon stays visible after refresh click until user clicks elsewhere."
created: 2026-08-07T00:00:00Z
updated: 2026-08-07T00:00:00Z
---

## Current Focus

hypothesis: CONFIRMED (all three sub-issues) — see Resolution
test: Direct code inspection of SourceChip.svelte, button.svelte (buttonVariants), utils.ts (cn/twMerge), WebspaceHeader.svelte, 06-UI-SPEC.md, plus git history (26f5d03, 7687dd6)
expecting: n/a — diagnosis complete
next_action: Return ROOT CAUSE FOUND to orchestrator (goal: find_root_cause_only — no fix applied)
bug_class: Bohrbug (deterministic CSS/markup — all three reproduce on every render/interaction)
known_pattern_candidate: none (KB has no UI/CSS entries)

reasoning_checkpoint:
  hypothesis: "Three independent deterministic defects in SourceChip.svelte: (1) chip height is driven by the 44px size-11 refresh button while the filter click handler lives only on the ~20px-tall inner label button, leaving ~17px dead bands; (2) refresh Button inherits buttonVariants' base rounded-lg because the class override never sets a radius, painting a rounded-square hover surface inside a rounded-full pill; (3) group-focus-within:opacity-100 pins the icon visible after a mouse click because clicking focuses the button and focus persists until the user clicks elsewhere."
  confirming_evidence:
    - "SourceChip.svelte:74-126 read directly — outer div (no onclick) carries py-1 and contains size-11 Button; filter onclick is on inner <button> wrapping only dot+label"
    - "button.svelte:7 base includes 'rounded-lg' and 'outline-none' with only focus-visible ring styles; SourceChip's class override (line 116-120) contains no radius class; cn = twMerge so size-11 beats icon-size's size-8 but nothing overrides rounded-lg"
    - "SourceChip.svelte:117 'opacity-0 ... group-hover:opacity-100 group-focus-within:opacity-100' — :focus-within matches persistent mouse-click focus, exactly matching 'hidden only when clicking any other area of the page'"
  falsification_test: "(1) DOM inspection showing the inner button fills chip height would refute the dead-zone claim; (2) computed style showing border-radius:9999px on the refresh button would refute the square-highlight claim; (3) if browsers did not focus buttons on mouse click, focus-within could not explain the sticky icon — but they do, and the reported 'hidden when clicking elsewhere' is the signature of focus, not hover, state"
  fix_rationale: "n/a — diagnose-only; fix direction recorded in Resolution for plan-phase --gaps"
  blind_spots: "Did not run the app or measure rendered pixels; heights are computed from class math (44px button + 8px py-1 + 2px border ≈ 54px chip, ~20px label band). User's '5px too many' is a rough visual estimate; the mechanical dead band is larger (~17px/side). Firefox focuses buttons on click like Chrome on Linux, so issue 3 is cross-browser here; macOS Safari differs but is out of scope (desktop Linux project)."
  candidate_causes:
    - "code: markup structure — click handler on inner button, height-driving sibling on outer div (confirmed, issue 1)"
    - "code: missing radius override on Button class (confirmed, issue 2)"
    - "design-spec (config-of-record): 06-UI-SPEC.md:44/51 explicitly specifies ':focus-within' reveal and lines 140/161 mandate the 44px touch-target floor on the refresh control — the implementation faithfully follows a spec that didn't account for mouse-click focus or for where the 44px floor would push chip geometry (confirmed contributing cause for issues 1 and 3)"
    - "environment: browser focus-on-click behavior — necessary condition for issue 3 but standard/universal on target platform, not a defect"
  and_gate: "yes for issues 1 and 3 — each needs the spec decision (44px floor / :focus-within) AND the implementation shape (handler on inner button only / no focus-visible scoping) to manifest. Issue 2 is single-cause (missing radius override). Fixing code alone suffices, but the spec must be reconciled or the next re-implementation regresses (same doc-code drift pattern as G-06-3)."

## Symptoms

expected: Selected/hover chip treatment reads as one polished oval (pill) control - chip height fits its text with modest padding, the whole visual chip surface triggers the chip's click (filter toggle), the hover-revealed refresh button's highlight follows the chip's pill geometry, and the refresh icon hides again when the pointer leaves the chip after a refresh click.
actual: "pass with some cosmetic-only issues. 1. The chip is larger (height) than it needs to be, maybe 5px too many above and below the text. This additional space does not trigger the click event on the chip which is counter-intuitive. 2. Hovering over the chip correctly shows the refresh button, moving to hover over the refresh button additionally highlights the button - but the background highlight is a rounded corner square (looks odd inside the more oval chip). 3. Clicking a refresh button causes the refresh icon to remain visible even when not hovering the chip. It is hidden only when clicking any other area of the page"
errors: None reported
reproduction: Test 3 in UAT (.planning/phases/06-ui-scalable-source-surface/06-UAT.md) - run `make dev`, open a webspace with source chips in header row, hover a chip, hover its refresh button, click the refresh button, move pointer away.
started: Discovered during UAT round 2, after gap-closure plan 06-06 rebuilt the selected-chip treatment (border-primary bg-primary fill). Prior round's G-06-3 (invisible selected state) resolved; these are new polish issues on same component.

## Eliminated

- hypothesis: Known pattern in knowledge base matches this bug
  evidence: KB-001/KB-002 are Go backend lifecycle/query-semantics patterns; no UI/CSS entries. No match.
  timestamp: 2026-08-07
- hypothesis: 06-06 (commit 26f5d03) introduced these three defects
  evidence: git show 26f5d03 — that commit only changed the selected treatment (bg-accent/10 ring → border-primary bg-primary, child re-toning). 'size-11', 'py-1', 'group-focus-within:opacity-100', and the missing radius override all predate it — present since 06-02 (commit 7687dd6, the original chip merge). Round-2 UAT simply scrutinized the chip more closely once the fill became visible.
  timestamp: 2026-08-07
- hypothesis: Sticky refresh icon is caused by the `source.syncing && 'opacity-100'` class lingering
  evidence: syncing keeps the icon visible only while the sync-run flag is true (intentional per D-03 — spinning icon is the sole in-place syncing indicator). The reported behavior ("hidden only when clicking any other area of the page") is the signature of persistent element focus, not a state flag — focus moves exactly when the user clicks elsewhere. group-focus-within is the match.
  timestamp: 2026-08-07
- hypothesis: TooltipTrigger wrapper interferes with the filter button's hit area
  evidence: SourceChip.svelte:83 uses the child snippet — TooltipTrigger renders no extra wrapping element; props spread directly onto the real <button>. Hit-area geometry is purely the button's own content box.
  timestamp: 2026-08-07

## Evidence

- timestamp: 2026-08-07
  checked: .planning/debug/knowledge-base.md for matching prior patterns
  found: Only KB-001 (context-cancelled two-phase write) and KB-002 (any-row vs latest-row query) — backend Go patterns, unrelated to Svelte UI cosmetics
  implication: No known-pattern shortcut; proceed with direct code inspection
- timestamp: 2026-08-07
  checked: web/src/lib/components/SourceChip.svelte (full read, HEAD f27d2542)
  found: "Outer <div> (line 74-79): 'group flex shrink-0 items-center gap-1.5 rounded-full border border-border bg-card py-1 pr-1 pl-2.5' — NO click handler. Filter onclick (line 88) is on the inner <button> (line 84-106) which wraps only the health dot (size-2) and the label span ('text-[14px] leading-[1.4]' ≈ 19.6px line box). Refresh Button (line 113-125): variant=ghost size=icon with class 'size-11 opacity-0 transition-opacity group-hover:opacity-100 group-focus-within:opacity-100' + syncing/selected conditionals; no radius class anywhere in the override."
  implication: Chip height = tallest child (44px size-11 Button) + py-1 (8px) + border (2px) ≈ 54px, while the only vertically-clickable band is the ~20px inner button — everything above/below is visually chip but interactively dead (issue 1). Icon reveal is bound to hover OR focus-within (issue 3 candidate). No radius override on the Button (issue 2 candidate).
- timestamp: 2026-08-07
  checked: web/src/lib/components/ui/button/button.svelte (buttonVariants)
  found: "base (line 7) includes 'rounded-lg', 'outline-none', and only focus-visible:* ring styles; ghost variant (line 13) = 'hover:bg-muted hover:text-foreground ...'; size icon (line 22) = 'size-8'"
  implication: Issue 2 confirmed — with no radius in SourceChip's override, the 44px hover surface paints bg-muted (or bg-primary-foreground/20 when selected) with rounded-lg corners: a rounded-corner square inside the rounded-full pill. Also explains why the sticky focused button shows no focus ring after a mouse click (outline-none + focus-visible-only ring), making the lingering icon look unexplained.
- timestamp: 2026-08-07
  checked: web/src/lib/utils.ts cn()
  found: cn = twMerge(clsx(...)) — tailwind-merge resolves conflicts in favor of the later class
  implication: 'size-11' in the override genuinely beats size=icon's 'size-8' → refresh button is 44×44px; but 'rounded-full' vs 'rounded-lg' has no such conflict resolution because the override never supplies a radius at all.
- timestamp: 2026-08-07
  checked: .planning/phases/06-ui-scalable-source-surface/06-UI-SPEC.md lines 44, 51, 140, 161
  found: "Line 44 (D-03): refresh control 'opacity-0 at rest, opacity-100 on chip :hover/:focus-within'. Line 51: ':focus-within keeps it visible for keyboard users per D-03's own \"and keyboard focus\" clause'. Lines 140/161: 'the 44px minimum touch-target floor (Phase 1) continues to apply unchanged to the chip row's refresh/overflow/filter controls' — while line 140 simultaneously carves a desktop-only 16px exception for scrollbar ticks on the grounds the project is desktop-only by hard constraint."
  implication: Issues 1 and 3 are spec-faithful implementations of spec decisions that produce the reported behavior. The 44px floor forces the size-11 button that inflates the chip; :focus-within was specified for keyboard reveal but also matches persistent mouse-click focus. The spec must be reconciled alongside any code fix (same doc-code drift lesson as G-06-3).
- timestamp: 2026-08-07
  checked: git log/show for SourceChip.svelte (26f5d03 = 06-06, 7687dd6 = 06-02)
  found: 26f5d03 touched only selected-state classes; size-11, py-1, group-focus-within, and the radius omission all shipped in 7687dd6 (06-02 original merge)
  implication: These are round-1 latent defects surfaced by round-2 scrutiny, not regressions from the G-06-3 fix.
- timestamp: 2026-08-07
  checked: web/src/lib/components/WebspaceHeader.svelte chip row + overflow trigger
  found: "Row (line 189): 'mt-4 flex flex-nowrap items-center gap-2 overflow-hidden'. Overflow trigger (line 207): 'flex h-11 shrink-0 items-center ... rounded-full' = 44px tall."
  implication: The chip (~54px) is ~10px taller than the h-11 overflow trigger sitting next to it and is the tallest child of the overflow-hidden row — corroborates the user's 'maybe 5px too many above and below' (5px/side relative to the 44px trigger). Any height fix should land both elements at the same height.
- timestamp: 2026-08-07
  checked: Browser focus semantics for issue 3 (reasoning, no research needed — standard platform behavior)
  found: On Linux Chrome/Firefox, mousedown on a <button> gives it :focus, which persists after mouseup and after the pointer leaves; :focus-within on an ancestor matches for as long as any descendant holds focus; focus moves only when another focusable/click target is activated
  implication: 'group-focus-within:opacity-100' + click-focus fully explains 'remains visible even when not hovering; hidden only when clicking any other area of the page'. Mechanism confirmed without needing a live repro.

## Resolution

root_cause: "Three co-located causes in SourceChip.svelte, two of them spec-rooted: (1) DEAD OVERSIZED PADDING — the chip's ~54px height is dictated by the 44×44px refresh button ('size-11', implementing 06-UI-SPEC.md:140/161's 44px touch-target floor) plus the outer div's py-1 and borders, but the filter-toggle onclick lives only on the inner label <button> (~20px tall, text-[14px]/1.4 + size-2 dot); the outer div has no handler, so the ~17px bands above/below the label are visually chip but click-dead — and the chip stands ~10px taller than the adjacent h-11 (44px) overflow trigger. (2) SQUARE HOVER HIGHLIGHT — the refresh Button's class override sets no border-radius, so buttonVariants' base 'rounded-lg' (button.svelte:7) shapes the ghost hover fill (hover:bg-muted, or hover:bg-primary-foreground/20 when selected): a 44px rounded-corner square inside the rounded-full pill. (3) STICKY REFRESH ICON — visibility is 'opacity-0 group-hover:opacity-100 group-focus-within:opacity-100' (SourceChip.svelte:117, faithfully implementing 06-UI-SPEC.md:44/51's ':focus-within' reveal); a mouse click focuses the button, focus persists after the pointer leaves, :focus-within keeps opacity at 100, and the base 'outline-none' + focus-visible-only ring means no visible focus indicator explains why — the icon hides only when a click elsewhere moves focus, exactly as reported. All three shipped in 06-02 (7687dd6); 06-06 (26f5d03) did not introduce them."
fix: ""
verification: ""
files_changed: []
