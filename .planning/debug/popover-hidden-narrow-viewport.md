---
status: diagnosed
trigger: "G-14-2 popover-hidden-narrow-viewport — The source chip's health popover is not shown at all on narrow viewports; it only appears when the viewport is wide enough to accommodate it."
created: 2026-08-18T16:30:00Z
updated: 2026-08-18T17:50:00Z
---

## Current Focus
<!-- OVERWRITE on each update - reflects NOW -->

bug_class: Bohrbug (deterministic) — CONFIRMED via cross-engine live reproduction
hypothesis: CONFIRMED (H1a) — the popover's hover trigger (the chip) is removed from the visible row by the chip-row overflow relegation at narrow widths; the tooltip machinery itself is sound at every width
test: complete — two Playwright experiments (Chromium + Firefox), screenshots
expecting: —
next_action: return ROOT CAUSE FOUND diagnosis to caller (goal: find_root_cause_only — plan-phase --gaps owns the fix)

reasoning_checkpoint:
  hypothesis: "The health popover cannot appear at narrow viewport widths because its trigger — the SourceChip — is relegated out of the visible row by visibleChipCount once chip natural width + reserved trailing controls exceed the row budget; with a realistic long-named chip (button capped at max-w-48/192px, whole chip ~230px) the row renders ZERO chips at ≤~400px, leaving nothing to hover"
  confirming_evidence:
    - "Experiment 2 (Chromium): chipInRow=true and tooltip fully visible at 440px; chipInRow=false + only '+N' trigger at 400px and 375px — matches the user's reported ~375-400px threshold exactly"
    - "Screenshot diag-live-resize-400.png: row contains only '+1' pill, add button, 'Refresh all' — no chip, so no hover target"
    - "Experiments 1+2: the styled tooltip renders visible, wrapped at 320px, correctly flipped/shifted at EVERY width down to 375px whenever a trigger exists — including on the clone inside the '+N' overflow popover (screenshot diag-popover-chip-w375.png shows it painting legibly above the popover)"
    - "Firefox run: byte-identical behavior at every width — not engine-specific"
  falsification_test: "If a chip visible in the row at ≤400px failed to show the tooltip on hover, this hypothesis would be wrong — tested directly (short-named 106px chip stays in row at 375px and its tooltip shows fine), so hiding is strictly a function of trigger relegation, not viewport width"
  fix_rationale: "N/A — goal is find_root_cause_only; fix direction handed to plan-phase --gaps"
  blind_spots: "User's live source count unknown (more chips move the relegation threshold wider than 400px — consistent with, not contradicting, the report); WebKit untested (two engines agreeing + mechanism being pure layout math makes engine-specificity implausible); did not reproduce the user's exact live gdrive setup"
  candidate_causes:
    - "code: visibleChipCount relegation removes the tooltip's trigger from the row (CONFIRMED primary)"
    - "data: long display_name (external gdrive-style source) widens the chip to ~230px, guaranteeing relegation at mobile widths (CONFIRMED contributor)"
    - "code/UX: the only remaining narrow-width path to health detail — open '+N', hover the clone — works but is undiscoverable; nothing signals that a relegated chip's health lives there (CONFIRMED second gate)"
    - "environment: user's browser engine hides the tooltip (ELIMINATED — Chromium and Firefox identical)"
  and_gate: "yes — the experienced failure needs BOTH (1) trigger relegation out of the row (code+data) AND (2) the '+N' overflow path not being taken (UX discoverability); condition 2 alone decides whether the health sentence is merely hidden-behind-one-click or 'not shown at all'"

## Symptoms
<!-- Written during gathering, then IMMUTABLE -->

expected: With a chip whose health sentence is long (e.g. an external untrusted plugin with a stale sync time), the popover/described text remains legible and usefully laid out on a narrow (mobile-width, ~375-400px) viewport.
actual: "the popover is not shown at all unless the viewport is wide enough to accommodate it" (verbatim user report, browser window resized narrow on desktop)
errors: None reported
reproduction: Test 2 in 14-UAT.md — run the UI, narrow the viewport below ~400px, hover/focus a source chip with a long health sentence
started: Discovered during UAT of phase 14. Popover rendered by SourceChip.svelte via bits-ui tooltip; 14-02 (commit 95ee38a) removed native title attributes and added visually-hidden aria-describedby description. Unknown whether 14-02 introduced or merely exposed pre-existing narrow-viewport hiding.

## Eliminated
<!-- APPEND only - prevents re-investigating -->

## Evidence
<!-- APPEND only - facts discovered -->

- timestamp: 2026-08-18T16:30:00Z
  checked: .planning/debug/knowledge-base.md for matching prior patterns
  found: No entries match a UI/tooltip/viewport symptom (KB-001 is a Go context-cancellation two-phase-write pattern; other entries unrelated to frontend floating UI)
  implication: No known-pattern shortcut; proceed with fresh investigation

- timestamp: 2026-08-18T16:31:00Z
  checked: web/src/lib/components/SourceChip.svelte (full read)
  found: Chip renders bits-ui Tooltip (TooltipProvider > Tooltip > TooltipTrigger child-snippet button + TooltipContent{tooltipText}). No media queries or width gating in this file. tooltipText can be very long (display name + synced-relative + advisory + " — untrusted external plugin" appends). aria-describedby span (sr-only) duplicates the text.
  implication: The hiding mechanism is not in SourceChip itself — it must live in the tooltip wrapper components (web/src/lib/components/ui/tooltip/), bits-ui's floating positioning layer, or global CSS

- timestamp: 2026-08-18T16:35:00Z
  checked: web/src/lib/components/ui/tooltip/*.svelte (all 5 wrapper files) + bits-ui 2.18.1 dist (use-floating-layer.svelte.js, tooltip-content.svelte defaults)
  found: Local wrappers are thin passthroughs; TooltipContent applies `w-fit max-w-xs` (320px cap) and no collision props. bits-ui defaults — avoidCollisions=true, hideWhenDetached=false, collisionPadding=0. The hide/referenceHidden middleware (which sets visibility:hidden+pointer-events:none on the floating wrapper) only runs when hideWhenDetached=true, which is NOT the case here. Floating wrapper always gets `minWidth: max-content`.
  implication: The library's collision behavior (flip/shift) repositions but never hides. Whatever hides the tooltip on narrow viewports is NOT the default hide middleware. Suspects narrow to (a) WebspaceHeader's chip-row overflow machinery (chip moved into overflow popover or clipped), (b) something about the trigger/hover state at narrow widths, (c) a positioning result that lands the content offscreen. Need live reproduction to differentiate.

- timestamp: 2026-08-18T16:36:00Z
  checked: WebspaceHeader.svelte structural grep (overflow/measure/responsive classes)
  found: Chip row is `flex flex-nowrap items-center gap-2 overflow-hidden` (line 482). Chips that don't fit are moved into an overflow popover (hiddenSources); a `h-0 overflow-hidden` measurement clone row (line 614) renders a second SourceChip instance per source. Title block is `max-md:hidden` below 768px.
  implication: On narrow viewports a long-named chip may not be in the visible row at all (only inside overflow popover), and every source ALWAYS has a hidden measurement clone whose own TooltipTrigger exists in clipped DOM. Need to determine which chip instance the user actually hovered and what its tooltip DOM did.

- timestamp: 2026-08-18T16:45:00Z
  checked: git show acd472d (the 14-02 commit, "suppress native tooltips on SourceChip (option-b)")
  found: The commit ONLY removes two title attributes (chip button + truncated name span) and adds the sr-only description span + aria-describedby. Nothing viewport-dependent touched; the styled bits-ui Tooltip markup is unchanged.
  implication: 14-02 did not introduce viewport-dependent hiding. It removed the browser-native title tooltip, which previously showed regardless of any styled-popover behavior — so whatever narrow-viewport behavior exists in the styled tooltip predates 14-02 and is now EXPOSED (no native fallback masks it).

- timestamp: 2026-08-18T16:50:00Z
  checked: WebspaceHeader.svelte (full chip-row + overflow machinery, lines 131-660) and format.ts visibleChipCount (lines 382-410)
  found: Chip row is overflow-hidden flex-nowrap; chips that don't fit the measured budget are NOT rendered in the row — they render only inside the "+N more sources" overflow Popover. visibleChipCount subtracts reserved trailing controls (Refresh all, Excluded toggle, add-source "+") and floors at 0, so at narrow widths ALL chips can be relegated to the overflow popover. Also: title/switcher block is max-md:hidden below 768px.
  implication: H1 candidate confirmed as structurally possible — on a ~375-400px viewport the chip may simply not exist in the visible row, so there is nothing to hover; "wide enough to accommodate it" would then describe the chip re-entering the row as width grows. Still need live reproduction to confirm which of H1/H2 matches the user's observation, and to test the popover-clone chip's tooltip.

- timestamp: 2026-08-18T16:55:00Z
  checked: 14-UI-SPEC.md line 230 + 14-04-PLAN.md backstop item
  found: The backstop truth was about G4 long messages in "a narrow mobile-takeover chip tooltip"; spec expected the chip hover tooltip to stay terse and long text to reach only the StreamSyncDegraded banner. UAT Test 2 checked the popover legibility for a long health sentence at mobile width.
  implication: Test 2 exercised the hover tooltip at narrow desktop-resized width — the exact surface never visually verified during the phase.

- timestamp: 2026-08-18T17:20:00Z
  checked: "Experiment 1: hermetic kernel + Playwright, short-named chip ('Mock 01', 106px wide), long health sentence via intercepted GET /api/sources (external tier + stale sync + notice), widths 1280/900/800/700/640/560/500/440/400/375"
  found: "At EVERY width incl. 375px: chip stayed in the row, hover opened the tooltip, wrapper+content both visibility:visible, opacity 1, rect x=0 y=127 w=320 h=76 (wrapped to 4 lines within max-w-xs, flipped below the chip, shifted to the left edge). Overflow '+N' trigger never appeared."
  implication: "The styled tooltip itself handles narrow viewports CORRECTLY (flip+shift+320px wrap). Naive H2 eliminated. The user's report cannot be about an in-row chip with a short name — the differentiator must be chip width/count: a realistic long display name (max-w-48 button caps at 192px; whole chip ~230px) or multiple chips pushes visibleChipCount to 0 at ~375-400px, removing the chip from the row entirely."

- timestamp: 2026-08-18T17:35:00Z
  checked: "Experiment 2 (Chromium): same harness, display_name = 'Google Drive Personal Archive' (button hits its max-w-48/192px cap; whole chip ~230px incl. menu button), widths 1280/700/560/500/440/400/375, plus a live-resize (1280→400 without reload) pass"
  found: "440px: chip in row, tooltip visible (320x76, wrapped). 400px and 375px: chipInRow=FALSE, only the '+N' overflow trigger renders — the row shows '+1' pill + add button + 'Refresh all' and NO chip (screenshot diag-live-resize-400.png). Opening '+N' and hovering the clone: tooltip renders visible and legible ABOVE the popover (screenshot diag-popover-chip-w375.png). Live-resize pass reproduces the same relegation."
  implication: "The reported symptom is reproduced as TRIGGER DISAPPEARANCE, not popover failure: at ≤~400px there is no chip to hover, so no popover can appear; widen past ~440px and the chip re-enters the row and the popover shows — precisely 'not shown at all unless the viewport is wide enough to accommodate it'. The threshold moves wider with more chips or wider reserved controls (the user's live webspace likely crossed it well above 400px)."

- timestamp: 2026-08-18T17:45:00Z
  checked: "Experiment 2 rerun under Firefox (playwright --project=firefox)"
  found: "Identical results at every width: chip in row through 440px, relegated at 400/375, tooltip visible whenever a trigger exists, popover-clone tooltip works"
  implication: "Not engine-specific — pure layout-budget math (visibleChipCount), deterministic in any browser"

- timestamp: 2026-08-18T17:48:00Z
  checked: "Historical scope: does 14-02 (acd472d) bear on this mechanism?"
  found: "Pre-14-02 the native title lived on the SAME chip button that is relegated out of the row — at narrow widths it was equally unreachable. The relegation machinery dates to the Phase 6 chip-row overflow design (UI-07) with 09.1-01 measurement fixes."
  implication: "14-02 neither introduced nor exposed this. The behavior is pre-existing; phase 14's UAT Test 2 (the UI-SPEC's single 'verification: backstop' item) is simply the first time anyone visually checked the chip health surface at mobile width."

## Eliminated
<!-- consolidated -->

- hypothesis: "H2-naive: the bits-ui floating tooltip is hidden/unplaceable for a VISIBLE in-row chip at narrow viewport widths (collision/hide middleware, max-width overflow, offscreen transform)"
  evidence: "Experiment 1 — tooltip fully visible, wrapped at 320px, correctly flipped below the chip and shifted into the viewport at all widths down to 375px for an in-row chip"
  timestamp: 2026-08-18T17:20:00Z

- hypothesis: "H1b: the tooltip fails specifically for the chip clone inside the '+N' overflow popover (z-order/hover interplay with PopoverContent)"
  evidence: "Experiment 2 — hovering the popover clone opens the tooltip, visible and painted above the popover, in both Chromium and Firefox (screenshot diag-popover-chip-w375.png)"
  timestamp: 2026-08-18T17:40:00Z

- hypothesis: "Engine-specific rendering failure in the user's browser"
  evidence: "Chromium and Firefox produce byte-identical diagnostics at every width; mechanism is pure layout math"
  timestamp: 2026-08-18T17:45:00Z

- hypothesis: "14-02's native-title removal introduced or exposed the narrow-viewport behavior"
  evidence: "The removed title sat on the same relegated button — pre-14-02 behavior at narrow width was identical (no surface at all); relegation predates phase 14 (Phase 6 UI-07 overflow design)"
  timestamp: 2026-08-18T17:48:00Z

## Resolution
<!-- OVERWRITE as understanding evolves -->

root_cause: "The chip health popover is unreachable on narrow viewports because its hover trigger — the SourceChip itself — is removed from the visible chip row by WebspaceHeader's overflow relegation (visibleChipCount, web/src/lib/format.ts:382) once the chip's natural width (~230px for a long display name: max-w-48 caps the button at 192px + ~38px menu control) plus reserved trailing controls (add-source '+', 'Refresh all', optional 'Excluded') exceeds the row budget; at ≤~400px with one long-named chip (wider thresholds with more chips) the row renders ZERO chips — only the '+N' pill — so there is nothing to hover and no popover can appear. AND-gate second condition: the sole remaining narrow-width path (open '+N', hover the clone inside the popover — which WORKS, verified cross-engine) is undiscoverable, so in practice the health detail is 'not shown at all'. The tooltip rendering/positioning machinery is NOT at fault: it wraps at max-w-xs (320px), flips and shifts correctly, and stays fully visible down to 375px whenever a trigger exists, in Chromium and Firefox alike. Pre-existing behavior of the Phase 6 chip-row overflow design; 14-02 neither introduced nor exposed it — phase 14's UAT was simply the first narrow-viewport visual check of this surface."
fix: "N/A — diagnose-only session (goal: find_root_cause_only); fix owned by plan-phase --gaps"
verification: "Root cause verified by controlled reproduction at the exact reported threshold (~400px), a live-resize pass, disproof of every competing hypothesis, and cross-engine agreement — see Evidence 17:20/17:35/17:45"
files_changed: []
