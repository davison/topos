---
status: resolved
trigger: "pass with caveat: the styling for the ticks is poor, when I first saw them I thought they were CSS artifacts from a broken stylesheet or a faulty page render"
gap_id: G-06-6
created: 2026-08-07T00:20:00Z
updated: 2026-08-07T00:52:00Z
---

## Current Focus

hypothesis: CONFIRMED (multi-cause). The ticks read as render artifacts because of a conjunction of three independent defects: (a) the overlay is inset 2px from the pane edge instead of clearing the ~11px native scrollbar, so every tick is bisected by the scrollbar edge; (b) the rest-state tone is within 1.35:1 of `--border`, i.e. visually identical to a stray hairline fragment; (c) the ticks carry no rail, label, or hierarchy that would group them into a system. Two further defects guarantee at least one obviously-broken tick on every render.
test: Static analysis of the rendered geometry (Tailwind v4 spacing scale, `scrollbar-width: thin`), computed contrast ratios against every surface the ticks sit on, and diff of implementation against 06-UI-SPEC.md's stated placement.
expecting: A concrete, quantified list a fix planner can design a coherent treatment from.
next_action: Hand off to plan-phase --gaps. No fix applied (diagnose-only mode).

bug_class: Bohrbug — fully deterministic, reproduces on every render with a multi-date stream. Not timing- or environment-dependent.

reasoning_checkpoint:
  hypothesis: "The tick treatment fails to read as an affordance because its geometry places it half-on/half-off the native scrollbar, its rest colour is indistinguishable from --border, and no grouping form (rail/label/hierarchy) exists to bind the dashes into a system."
  confirming_evidence:
    - "Measured geometry: tick horizontal extent [R-16px, R-4px]; native scrollbar occupies [R-11px, R]. 7 of the tick's 12px sit on top of the scrollbar, 5px stick out to its left."
    - "Computed contrast: rest tick #353d4f vs --background = 1.86:1; vs --border #1e293b = 1.35:1."
    - "Component source: tick is a bare `span.h-0.5.w-3` — no rail, no label, no major/minor distinction."
    - "06-UI-SPEC.md line 111 says 'immediately left of the native scrollbar track'; component comment says 'immediately inside'."
  falsification_test: "If the overlay's right offset were >= the native scrollbar width AND the rest tone were >= 3:1 against the pane surface AND a grouping rail existed, and the ticks still read as artifacts, this hypothesis is wrong."
  fix_rationale: "Each cause is independently sufficient to break the 'intentional affordance' reading; the fix must address geometry, tone, and form together. Addressing only one leaves the artifact impression intact."
  blind_spots: "No live browser measurement was possible (read-only, no display). Geometry is derived arithmetically from the Tailwind spacing scale and the CSS scrollbar width rather than measured from a running page. Chromium's exact `scrollbar-width: thin` pixel value (~11px) is platform-dependent, but any value > 4px produces the overlap."
  candidate_causes:
    - "code (geometry): overlay right-inset of 2px does not clear the native scrollbar"
    - "code (derivation): dateMarkers has no scroll-overflow guard, so ticks render with no scrollbar present"
    - "config (design tokens): --scrollbar-thumb at 35% alpha is below the 3:1 non-text UI floor and collides with --border"
    - "design/spec: no rail, label, or visual hierarchy specified for the tick set"
    - "process: 06-03-SUMMARY.md records the overlay's visual treatment as never confirmed in a live browser"
  and_gate: "YES — the reported perception requires the conjunction. Correct tone with wrong geometry still shows bars bisected by the scrollbar; correct geometry with border-tone hairlines and no rail still reads as stray stylesheet debris. Root cause is recorded as a set."

## Symptoms

expected: The stream scrollbar's date ticks read as an intentional, polished UI affordance. Functionally everything works — ticks render, tooltips show, click-to-jump works, ticks hide during search.
actual: The ticks look like CSS artifacts from a broken stylesheet or a faulty page render.
errors: none
reproduction: Test 6 in 06-UAT.md — open a webspace whose stream spans several dates; look at the ticks alongside the stream scrollbar.
started: Discovered during Phase 6 UAT (2026-08-07). Built by plan 06-03 (StreamDateMarkers.svelte), commit `899504f`.

## Eliminated

- hypothesis: "Tooltip open-delay is too long, so the user never saw the date explanation before forming the artifact impression."
  evidence: web/src/lib/components/ui/tooltip/tooltip-provider.svelte defaults `delayDuration = 0` — tooltips appear instantly on hover. Not a contributing cause.
  timestamp: 2026-08-07T00:44:00Z

- hypothesis: "The tick colour differs between light and dark themes, so one theme renders it invisibly."
  evidence: app.css `.dark` mirrors `:root` verbatim (comment at lines 118-121 states this explicitly, and the palette values at 61-110 vs 129-149 are identical). Single effective theme — no theme-dependent divergence.
  timestamp: 2026-08-07T00:47:00Z

- hypothesis: "Markers are positioned wrongly relative to the items they represent (chronology misrepresentation, the plan's stated MUST-NOT)."
  evidence: `candidateMarkers` uses `topPx = (index / items.length) * trackHeightPx` over the in-order, non-virtualised, fixed-height-row list — index-proportional positioning is correct for a track-mapped overlay. Chronology is faithful. (Separate, lesser thumb-travel mismatch noted in Evidence, but it is not a chronology error.)
  timestamp: 2026-08-07T00:48:00Z

## Evidence

- timestamp: 2026-08-07T00:20:00Z
  checked: web/src/lib/components/StreamDateMarkers.svelte (full file)
  found: Overlay container is `div.pointer-events-none.absolute.inset-y-0.right-0.5.w-4`. Each tick is a `button.absolute.right-0.h-4.w-4.-translate-y-1/2` containing a bare `span.h-0.5.w-3.rounded-full.bg-[var(--scrollbar-thumb)]`. No rail/track element, no date label, no month/year vs day distinction, no `cursor-pointer`, and `focus-visible:outline-none` with only a colour swap as the focus signal.
  implication: The entire visual vocabulary of the affordance is a single 12x2px dash repeated at irregular offsets. Nothing binds the dashes into a set.

- timestamp: 2026-08-07T00:28:00Z
  checked: web/src/app.css lines 152-200 (scrollbar rules) + Tailwind v4 spacing scale (`--spacing` confirmed un-overridden at app.css:46-47)
  found: `:root { scrollbar-width: thin; }` is inherited by the stream scroll div; the webkit fallback declares `::-webkit-scrollbar { width: 10px }`. Chromium's `thin` renders ~11px. Tailwind v4 base `--spacing: 0.25rem` gives `right-0.5` = 2px, `w-4` = 16px, `w-3` = 12px, `h-0.5` = 2px.
  implication: With the pane's right edge at R — overlay spans [R-18, R-2]; the 12px tick, centred in the 16px button, spans [R-16, R-4]; the native scrollbar occupies [R-11, R]. **7 of each tick's 12px are painted on top of the scrollbar; 5px stick out to its left.** Every tick is bisected by the scrollbar's left edge.

- timestamp: 2026-08-07T00:33:00Z
  checked: Alpha compositing of the tick over the scrollbar thumb
  found: Both the thumb and the tick use the identical token `--scrollbar-thumb` = `color-mix(in srgb, var(--muted-foreground) 35%, transparent)`. Where the tick overlaps the thumb, two 35%-alpha layers composite to ~58% — where it overlaps the transparent track, it shows at 35%.
  implication: Each tick renders as **two different tones across its own 12px width**, and the boundary between those tones *moves as the user scrolls* (the thumb slides under the tick row). A bar that changes tone mid-length and shifts appearance during scroll is the textbook signature of a broken render, not a control.

- timestamp: 2026-08-07T00:38:00Z
  checked: Computed WCAG contrast (script in scratchpad) for the tick against every surface it sits on
  found: |
    rest  #353d4f vs --background #020617 = 1.86:1
    rest  #3e485c vs --card       #0f172a = 1.94:1
    rest         vs --border      #1e293b = 1.35:1 (background) / 1.59:1 (card)
    hover #5a6478 vs --background          = 3.39:1
    hover #5f6b7f vs --card                = 3.31:1
  implication: The rest state is far below WCAG 1.4.11's 3:1 floor for non-text UI components, and — the decisive number — it is only **1.35:1 away from `--border`**, the app's own hairline colour. A 2px bar in near-exactly the border tone is *literally indistinguishable from a stray fragment of a border rule*. This is the most direct mechanical explanation of "CSS artifacts from a broken stylesheet". Only on hover does the tick cross 3:1 — so the affordance is invisible until the pointer is already on it.

- timestamp: 2026-08-07T00:41:00Z
  checked: `candidateMarkers` (format.ts:666-685) + the `-translate-y-1/2` on the tick button + parent overflow context (`main.flex.min-h-0.flex-1.gap-8.px-6.py-8`, no `overflow-hidden`; wrapper `div.relative`, no `overflow-hidden`)
  found: The first candidate is **always** emitted (`lastKey` starts `null`, so index 0 always pushes) with `topPx = (0 / N) * H = 0`. The button applies `-translate-y-1/2`, centring it on y=0.
  implication: **There is always a tick centred exactly on the stream pane's top edge, half of it rendered above the pane, spilling unclipped into `main`'s 32px top padding.** A detached 2px dash floating in the gutter above the first row, on every single render. This alone reads as a rendering fault.

- timestamp: 2026-08-07T00:45:00Z
  checked: `dateMarkers` guard clause (format.ts:737-753)
  found: The only guards are `items.length < 2`, `trackHeightPx <= 0`, and all-items-same-day. There is **no check that the stream actually overflows** (no `scrollHeight > clientHeight` test).
  implication: When a multi-date stream fits the pane without scrolling, **there is no scrollbar at all** — yet the ticks still render. The user then sees a column of faint dashes at the right edge of the stream rows, decorating nothing. That is the purest possible "artifact" presentation, and it is a likely first impression on a short webspace.

- timestamp: 2026-08-07T00:49:00Z
  checked: Thumb-travel vs tick-position mapping
  found: Ticks map item fraction `f` to `f * trackHeight`. The scrollbar thumb's top edge maps the same `f` to `f * (trackHeight - thumbHeight)`. For a short stream (e.g. 30 rows in an 800px pane, thumb ~500px), a tick at f=0.9 sits at 720px while the thumb's top never travels past 300px.
  implication: Ticks systematically occupy track regions the thumb never reaches, and drift out of alignment with the thumb as you scroll. This actively contradicts the "these belong to the scrollbar" reading. Not a chronology error (the ticks still correctly describe the item order), but it breaks the visual association with the scrollbar.

- timestamp: 2026-08-07T00:51:00Z
  checked: 06-UI-SPEC.md lines 111 and 115-117 vs the component's own header comment (StreamDateMarkers.svelte lines 48-57)
  found: |
    SPEC:  "a separate absolutely-positioned overlay sitting immediately LEFT OF
            the stream pane's native scrollbar track"
    IMPL:  "inset just enough to sit immediately INSIDE the native scrollbar track
            (app.css's ::-webkit-scrollbar is 10px wide, unchanged by this file)"
  implication: **Direct spec divergence, acknowledged in the implementation's own comment.** The implementer read the 10px scrollbar width, then chose a 2px inset — which cannot clear a 10px scrollbar. To sit left of it the overlay needs a right offset of at least the scrollbar width (`right-2.5`/10px minimum, `right-3`/12px with margin), not `right-0.5`/2px. The spec's other visual instructions (2px tick, `--scrollbar-thumb` at rest / `--scrollbar-thumb-hover` on hover) were followed literally — so the spec itself under-specifies: it never called for a rail, labels, hierarchy, or a minimum contrast, and its "reuse the thumb token, no new colour" constraint is what forced the 1.86:1 rest tone.

- timestamp: 2026-08-07T00:52:00Z
  checked: 06-03-SUMMARY.md lines 60-67 and 147
  found: The plan's own `<human-check>` verification for tick appearance/positioning was recorded as `human_judgment: true` and explicitly **never exercised** — "no live browser session was available... a follow-up UAT pass against the running app should confirm both".
  implication: The overlay's visual treatment was shipped without anyone ever looking at it. UAT was the first observation. This is why an arithmetically-obvious 2px-vs-10px error survived to the user.

- timestamp: 2026-08-07T00:53:00Z
  checked: Pointer-event footprint of the tick hit areas over the scrollbar
  found: Each `pointer-events-auto` button spans [R-18, R-2] x 16px tall — covering ~9px of the ~11px-wide scrollbar across a 16px band at every marker position.
  implication: Secondary observation only. UAT confirmed the thumb still drags, so this is not the reported defect — but it constrains the fix: enlarging the tick hit area for discoverability must not extend further over the scrollbar, which is another reason the whole overlay should move left of the scrollbar rather than grow in place.

## Resolution

root_cause: |
  Three conjoint causes (AND-gate fired — no single one alone produces the reported impression):

  1. GEOMETRY — The overlay is inset only 2px (`right-0.5`) from the pane's right
     edge, but the native scrollbar is ~11px wide (`scrollbar-width: thin`; 10px
     webkit fallback). Each 12px tick therefore spans [R-16, R-4] and is bisected
     by the scrollbar's left edge: 5px outside it, 7px painted on top of it. Because
     tick and thumb share the same 35%-alpha token, the on-thumb half composites to
     a different tone than the off-thumb half, and that tonal boundary slides along
     the tick as the user scrolls. This directly contradicts 06-UI-SPEC.md line 111
     ("immediately LEFT OF the native scrollbar track"); the component's own comment
     documents the divergence ("immediately INSIDE").

  2. TONE — The rest-state tick is `--scrollbar-thumb` (`--muted-foreground` at 35%
     alpha), computing to #353d4f: 1.86:1 against the pane background (below WCAG
     1.4.11's 3:1 non-text floor) and only 1.35:1 away from `--border` (#1e293b).
     A 2px bar rendered in the app's own hairline-border tone is by definition
     indistinguishable from a stray fragment of a border rule.

  3. FORM — The tick set has no grouping vocabulary whatsoever: no rail or track
     backing the ticks, no date labels, no major/minor hierarchy (month/year vs day),
     and irregular vertical spacing (24px floor, otherwise index-proportional). Nothing
     on screen binds the dashes into a system or explains them; the only explanation
     is a tooltip gated behind hovering a 16px band, on an element that shows no
     pointer cursor (`cursor-pointer` appears nowhere in the codebase).

  Two further defects guarantee at least one visibly-broken tick on every render:

  4. A marker at `topPx = 0` is ALWAYS emitted (first candidate always kept), and the
     button applies `-translate-y-1/2` — so a tick is permanently half-rendered above
     the pane's top edge, spilling unclipped into `main`'s padding.

  5. `dateMarkers` has no scroll-overflow guard, so on a multi-date stream that fits
     the pane the ticks render with no scrollbar present at all — dashes decorating
     nothing.

  Process cause: 06-03-SUMMARY.md records the overlay's visual verification as
  `human_judgment: true` and never exercised ("no live browser session was
  available"). Nobody looked at it before UAT.

fix: [not applied — diagnose-only mode; handed to plan-phase --gaps]
verification: [n/a]
files_changed: []
