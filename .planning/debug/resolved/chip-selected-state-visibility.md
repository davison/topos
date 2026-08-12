---
status: resolved
trigger: "G-06-3: pass with one caveat: when selected there is no visual cue. Previously the chip changed to a contrasting blue shade. Now there is the merest border highlight (or possibly a drop shadow) on the chip, visible only on the left and right rounded edges."
created: 2026-08-07T00:00:00Z
updated: 2026-08-07T00:00:00Z
mode: find_root_cause_only
---

## Current Focus

hypothesis: Selected state uses `bg-accent/10 ring-2 ring-accent`, where `--accent: #1e293b` is
  a deliberately NEUTRAL dark slate identical to `--border`, so both the fill and the ring are
  near-zero contrast against the chip's own `bg-card` + `border-border` unselected treatment;
  additionally the ring's top/bottom edges are clipped by the chip row's `overflow-hidden`.
test: Read SourceChip.svelte selected classes, resolve every token through app.css, compare
  against the deleted SourceFilterChips.svelte pre-Phase-6 treatment, and check the containing
  row for overflow clipping of the ring box-shadow.
expecting: If confirmed, --accent resolves to the same hex as --border and the row is
  overflow-hidden with height == chip height.
next_action: none — diagnosis complete, all three causes confirmed by direct observation
  (compiled CSS, twMerge run against the project's own tailwind-merge, resolved token values,
  measured box geometry). Handoff to plan-phase --gaps for the fix.

reasoning_checkpoint:
  hypothesis: "The selected treatment is invisible because ring-accent/bg-accent/10 resolve to
    #1e293b — byte-identical to the --border colour the chip already paints unselected — and the
    ring's top/bottom edges are additionally clipped by the row's overflow-hidden."
  confirming_evidence:
    - "Compiled CSS: .ring-accent{--tw-ring-color:var(--accent)}; app.css: --accent:#1e293b and
       --border:#1e293b — the same value."
    - "twMerge run on the actual class string drops bg-card, leaving a 10%-alpha slate over an
       identically-coloured #0f172a header surface."
    - ".ring-2 emits an OUTSET box-shadow (--tw-ring-inset empty); the chip is the tallest child
       of an unpadded overflow-hidden row, so its ring exceeds the row's padding box by 2px top
       and bottom."
    - "git show of the deleted SourceFilterChips.svelte confirms the prior treatment was Button
       variant=default = a solid bg-primary #60a5fa fill, matching the user's memory verbatim."
  falsification_test: "Selecting a chip inside the overflow popover (no overflow-hidden ancestor)
    should show a complete unclipped 2px ring on all four sides that is STILL near-invisible. If
    that ring were clearly visible there, colour would not be the primary cause."
  fix_rationale: "n/a — diagnose-only mode; direction recorded in the Fix Direction section."
  blind_spots: "Not verified in a live browser (read-only constraint, parallel agents on the main
    tree) — all geometry is derived from resolved class values and the compiled stylesheet rather
    than from a rendered DevTools measurement. The popover differential above is the cheapest
    live confirmation if wanted."
  candidate_causes:
    - "code (colour): SourceChip.svelte:77 uses the --accent token, which is neutral slate here"
    - "code (layout): WebspaceHeader.svelte:189 overflow-hidden clips the outset ring box-shadow"
    - "spec/config: 06-UI-SPEC.md:45 class literal contradicts 06-UI-SPEC.md:163 Color table"
  and_gate: "yes — colour alone fixed still leaves an asymmetric ring clipped top and bottom;
    clipping alone fixed still leaves an invisible slate-on-slate ring; and leaving the spec
    contradiction in place lets either code fix be reverted by a future spec re-read."

## Symptoms

expected: Selected source chips carry a clearly visible selected-state treatment (previously a
  contrasting blue fill).
actual: Selected chip is visually near-identical to unselected. Only "the merest border
  highlight (or possibly a drop shadow) visible only on the left and right rounded edges."
errors: none
reproduction: 06-UAT.md Test 3 — click a source chip in the webspace header to select it as a filter.
started: Phase 6 plan 06-02, which replaced SourceFilterChips.svelte + SourceHealthChip.svelte
  (both deleted) with the merged SourceChip.svelte. Baseline commit c3dfdf19a7fc.
note: Functional multi-select filtering + URL round-trip both work. This is purely the visual
  affordance.

## Eliminated

- hypothesis: The `selected` prop is not reaching SourceChip / the class is not applied at all.
  evidence: `aria-pressed={selected}` uses the same prop and the UAT reports filtering works and
    round-trips through the URL; the user also observes a faint but real treatment change, so the
    conditional class IS applying. Not a data-flow bug.
  timestamp: 2026-08-07

- hypothesis: Dark-theme token mismatch (selected style authored for a light palette).
  evidence: app.css `.dark` block (lines 128-150) mirrors `:root` verbatim — every token has an
    identical value in both. There is no light palette at all (app.css line 7: "Dark-mode-only —
    no light palette, no toggle"). A theme mismatch is structurally impossible here.
  timestamp: 2026-08-07

## Evidence

- timestamp: 2026-08-07
  checked: web/src/lib/components/SourceChip.svelte:74-79 (the chip wrapper element)
  found: |
    class={cn(
      'group flex shrink-0 items-center gap-1.5 rounded-full border border-border bg-card py-1 pr-1 pl-2.5',
      selected && 'bg-accent/10 ring-2 ring-accent'
    )}
    The ENTIRE selected treatment is two utilities: `bg-accent/10` and `ring-2 ring-accent`.
  implication: Selected-state visibility depends wholly on how far `--accent` diverges from the
    unselected `bg-card` + `border-border` pair.

- timestamp: 2026-08-07
  checked: web/src/app.css token values (:root lines 60-126, .dark lines 128-150)
  found: |
    --accent: #1e293b
    --border: #1e293b     <-- IDENTICAL to --accent
    --card:   #0f172a
    --primary:#60a5fa     (the blue)
    --ring:   #60a5fa     (the blue)
    app.css lines 77-85 carry an explicit comment: "The UI-SPEC reserves the accent BLUE for
    CTAs/links/focus rings/selected-row indicators only ... so --accent stays a neutral surface
    here. The actual blue lives in --primary ... and --ring."
  implication: PRIMARY CAUSE. `ring-accent` paints #1e293b — the exact same colour the chip's
    unselected `border-border` already paints. The ring is not a highlight; it is a same-colour
    thickening of an edge that was already that colour. And `bg-accent/10` composites #1e293b at
    10% alpha over #0f172a -> approx #111927, a delta of under 2/255 per channel from the
    unselected #0f172a. Both halves of the selected treatment are, by construction, invisible.
    The author reached for the token literally named "accent" while app.css documents at length
    that in THIS palette the accent colour is --primary/--ring, not --accent.

- timestamp: 2026-08-07
  checked: git show c3dfdf19a7fc^:web/src/lib/components/SourceFilterChips.svelte (deleted file)
  found: |
    <Button variant={selectedSource === source.name ? 'default' : 'outline'} ... >
    shadcn's `default` Button variant is `bg-primary text-primary-foreground`.
    --primary: #60a5fa -> a saturated blue FILL.
  implication: Confirms the user's memory exactly. The pre-Phase-6 selected treatment was a full
    #60a5fa blue background fill (default variant) vs a transparent bordered outline variant.
    Phase 6 swapped a full-fill primary-blue treatment for a 10%-alpha neutral-slate tint plus a
    ring the same colour as the existing border. The regression is a direct consequence of the
    component rewrite, not of any token change (no token values changed this phase).

- timestamp: 2026-08-07
  checked: web/src/lib/components/WebspaceHeader.svelte:189 (the visible chip row)
  found: |
    <div class="mt-4 flex flex-nowrap items-center gap-2 overflow-hidden" bind:this={rowEl}>
    The row is `overflow-hidden` (required by the 06-04 overflow-measurement design) with no
    vertical padding, and is `items-center` so its content-box height equals the tallest flex
    item — the chip itself.
  implication: SECOND CONTRIBUTING CAUSE, and the one that explains the specific "left and right
    rounded edges only" wording. Tailwind's `ring-*` is an OUTSET box-shadow drawn outside the
    element's border box. `overflow: hidden` on an ancestor clips descendant box-shadow painting
    to the ancestor's padding box. Because row height == chip height exactly, the ring's top and
    bottom 2px fall outside the row and are clipped away, while the left/right 2px sit inside the
    8px `gap-2` between chips and survive. On a `rounded-full` pill the surviving fragments are
    precisely the two semicircular end caps — verbatim what the user reported seeing.

- timestamp: 2026-08-07
  checked: web/src/lib/components/WebspaceHeader.svelte:218-231 (overflow popover copy of the chip)
  found: |
    Inside PopoverContent the chips render in `flex flex-col gap-2` with NO overflow-hidden
    ancestor, so the ring is NOT clipped there.
  implication: Differential prediction that separates the two causes: a chip selected inside the
    overflow popover should show a complete, unclipped 2px ring on all four sides — yet still be
    near-invisible, because the ring colour still equals the border colour. Confirming this
    proves colour (not clipping) is the primary cause and clipping is secondary.

- timestamp: 2026-08-07
  checked: Compiled Tailwind output, web/.svelte-kit/output/client/_app/immutable/assets/0.B6kmLTnW.css
  found: |
    .ring-accent{--tw-ring-color:var(--accent)}
    .bg-accent\/10{background-color:color-mix(in oklab, var(--accent) 10%, transparent)}
    .ring-2{--tw-ring-shadow:var(--tw-ring-inset,) 0 0 0 calc(2px + var(--tw-ring-offset-width))
            var(--tw-ring-color,currentcolor); box-shadow:...var(--tw-ring-shadow)...}
  implication: Direct confirmation of both mechanisms, not inference. The ring resolves to
    var(--accent) = #1e293b, and `--tw-ring-inset` is empty so the ring is an OUTSET box-shadow —
    exactly the kind of painting an ancestor's `overflow: hidden` clips.

- timestamp: 2026-08-07
  checked: twMerge resolution of the chip's actual class string (run against the project's own
    tailwind-merge 3.6.0)
  found: |
    SELECTED   -> ...rounded-full border border-border py-1 pr-1 pl-2.5 bg-accent/10 ring-2 ring-accent
    UNSELECTED -> ...rounded-full border border-border bg-card py-1 pr-1 pl-2.5
    `bg-card` is REMOVED from the selected chip — bg-accent/10 wins the bg-color conflict group.
  implication: The selected chip is not "bg-card plus a tint"; it loses its opaque fill entirely
    and becomes a ~10%-alpha slate wash over whatever is behind it. What is behind it is the
    header, which is itself `bg-card` (#0f172a) — the same colour the chip just gave up. Net
    rendered delta from unselected: roughly #0f172a -> #11192c, about 2/255 per channel.

- timestamp: 2026-08-07
  checked: .planning/phases/06-ui-scalable-source-surface/06-UI-SPEC.md lines 45 and 163
  found: |
    Line 45 (component spec):  "the chip gets a 2px accent ring (`ring-2 ring-accent`) plus
                                `bg-accent/10`"
    Line 163 (Color table):    "| Accent (10%) | `#60a5fa` | ... **the filtered/selected source
                                chip's ring** ... |"
    #60a5fa is --primary / --ring in this palette. --accent is #1e293b.
  implication: ORIGIN CAUSE (spec category, distinct from the two code causes). The UI-SPEC
    contradicts itself: its Color table assigns #60a5fa to the selected chip's ring, while its
    component prose spells the class as `ring-accent`, which cannot produce #60a5fa in this
    codebase. app.css lines 77-85 documents this exact trap in advance ("--accent stays a neutral
    surface here ... The actual blue lives in --primary ... and --ring"). 06-02-PLAN.md line 167
    then carried the ambiguity forward as prose ("a two-pixel accent ring plus a ten-percent
    accent background tint"), and the implementer resolved "accent" to the literally-named token.
    The implementation is faithful to the spec's letter and violates the spec's own colour table.

- timestamp: 2026-08-07
  checked: web/src/lib/components/ui/button/button.svelte size variants, vs every child of the row
  found: |
    Button size icon = `size-8` (32px), but SourceChip passes class="size-11" which twMerge
    promotes to 44px. Chip height = 44px + py-1 x2 = 52px.
    Overflow trigger = h-11 = 44px. Trailing Clear filters / Refresh all = size="sm" = h-7 = 28px.
  implication: The chip (52px) is the TALLEST child of the row, so the `items-center` row's
    content-box height equals the chip height exactly and the row has no vertical padding. The
    chip therefore spans y=0..52 with its ring painting y=-2..54 — the 2px above and below fall
    outside the row's padding box and are clipped. The left/right 2px sit inside the 8px `gap-2`
    and survive. Geometry confirms the reported "left and right rounded edges only" precisely.

- timestamp: 2026-08-07
  checked: grep for tests asserting the chip's selected classes across web/src
  found: No test file references SourceChip at all; `ring-accent`/`bg-accent` appear in exactly
    two places repo-wide — SourceChip.svelte:77 and 06-UI-SPEC.md:45.
  implication: No automated gate could have caught this. It is a pure rendered-contrast defect
    with no behavioural signature — `aria-pressed` is correct, filtering is correct, the class
    string is correct per spec. Only a human looking at the screen (or a contrast assertion /
    visual-regression snapshot) can detect it. Also means the fix is fully localized: one class
    string in one component, plus the spec line that produced it.

## Resolution

root_cause: |
  Three contributing causes across two categories. AND-gate: YES — fixing any one alone leaves a
  still-broken affordance, and the symptom's exact reported SHAPE requires (1) and (2) together.

  (1) COLOUR — primary. SourceChip.svelte:77 expresses the selected state as
      `bg-accent/10 ring-2 ring-accent`. In this project's palette `--accent` is #1e293b, a
      deliberately neutral dark slate that is byte-identical to `--border` (#1e293b), which the
      chip already paints when UNSELECTED. So the ring adds no new colour, and `bg-accent/10`
      composites to ~#111927 over the chip's #0f172a `bg-card` — under 2/255 per channel of
      change. app.css lines 77-85 explicitly document that in this palette the accent BLUE for
      "selected-row indicators" lives in `--primary`/`--ring` (#60a5fa), NOT in `--accent`. The
      component used the misleadingly-named token.

  (2) CLIPPING — secondary, explains the observed "left and right edges only" shape.
      `ring-*` renders as an outset box-shadow; the chip's parent row
      (WebspaceHeader.svelte:189) is `overflow-hidden` with content-box height equal to the chip
      height, so the ring's top/bottom 2px are clipped and only the left/right pill caps (which
      sit inside the 8px inter-chip gap) survive.

  (3) SPEC CONTRADICTION — origin, category: spec/config, not code.
      06-UI-SPEC.md:163 (Color table) assigns #60a5fa to "the filtered/selected source chip's
      ring", but 06-UI-SPEC.md:45 (component prose) spells the class as `ring-2 ring-accent`,
      which cannot yield #60a5fa here. 06-02-PLAN.md:167 carried the ambiguity forward as
      "accent ring / accent tint". The implementation faithfully followed the class literal and
      thereby violated the Color table. Fixing only the component without fixing UI-SPEC.md:45
      leaves the contradiction live to reintroduce the bug.

  Regression origin: the pre-Phase-6 SourceFilterChips.svelte used shadcn Button
  `variant="default"` when selected = a full `bg-primary` (#60a5fa) blue fill. No token values
  changed in Phase 6; the loss is entirely from the 06-02 component rewrite choosing a new,
  much weaker selected treatment.

  Why not caught: no gate existed for this class of defect. No test references SourceChip; the
  markup, ARIA and behaviour are all correct, and the class string matches the written spec —
  the defect exists only in rendered contrast. Code review would have had to resolve
  --accent -> #1e293b -> equals --border by hand to see it.
fix: "not applied — diagnose-only mode (goal: find_root_cause_only). Handoff to plan-phase --gaps."
verification: "n/a — diagnose-only mode"
files_changed: []

## Fix Direction (for plan-phase --gaps, not applied here)

Three things must change together; the first two are the fix, the third stops it recurring.

1. Colour — SourceChip.svelte:77. Replace the `accent` token with the palette's real accent blue
   (#60a5fa), which is `--primary` / `--ring`, i.e. `ring-ring` or `ring-primary` rather than
   `ring-accent`. Note `bg-accent/10` -> `bg-primary/10` is still only a ~10% wash; the user
   explicitly remembers and asks for a "contrasting blue shade", so a solid `bg-primary` fill
   (matching the retired Button `variant="default"` treatment) is the closer match to both the
   prior behaviour and the report.
   CAUTION if choosing a solid fill: the chip's label span hardcodes `text-foreground` (#f1f5f9)
   at SourceChip.svelte:96, which is near-white and would be unreadable on #60a5fa. A fill
   approach must thread `selected` into that span (`text-primary-foreground`, #0f172a) and
   re-check the health-dot tones against the blue ground.

2. Clipping — the ring must survive WebspaceHeader.svelte:189's `overflow-hidden`, which is
   load-bearing for the 06-04 overflow measurement and should not simply be removed. Options, in
   rough order of least disruption: use `ring-inset` (inset box-shadow paints inside the border
   box and is never clipped); or use a thicker `border-primary` on the chip instead of a ring
   (borders are inside the element box); or give the row `py-1` with a compensating `-my-1` so
   the ring has room. A solid background fill per (1) sidesteps this cause entirely — backgrounds
   paint inside the border box.

3. Spec — fix 06-UI-SPEC.md:45 so its class literal agrees with its own Color table at line 163.
   Otherwise the contradiction survives the code fix and any future re-read of the spec
   reintroduces `ring-accent`. Consider also a short note near app.css lines 77-85 pointing at
   this incident, since that comment already anticipated exactly this confusion.

Suggested recurrence guard: a component test on SourceChip asserting the selected chip's
resolved class string contains a primary/ring-token class and NOT `ring-accent`/`bg-accent`, or a
visual-regression snapshot of the selected chip. A plain behavioural test cannot catch this class.
