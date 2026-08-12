---
status: resolved
trigger: "In the detail pane title, no search term highlight is shown. An amber highlight inside a mark element shows in the detail pane body. In the stream pane, the search term shows as <span class=\"font-semibold\"/> (barely visible)."
created: 2026-08-07T00:00:00Z
updated: 2026-08-07T00:20:00Z
gap_id: G-06-1
goal: find_root_cause_only
---

## Current Focus

hypothesis: |
  TWO independent root causes (AND-gate fired — see reasoning_checkpoint):
  RC-1 — the detail-pane title was never inside 06-01-PLAN.md's implemented scope for
  UI-09; the plan scoped highlighting to `detailBodyVariant` branches only. This is a
  specification/scope gap, not a coding defect.
  RC-2 — the stream/search-results `font-semibold` match treatment is a deliberate,
  documented Phase 3 (03-04) design decision that Phase 6 introduced a competing amber
  vocabulary alongside without reconciling. Pre-existing, not a Phase 6 regression.

test: (complete — both confirmed by direct code + doc + git-blame evidence)
expecting: (complete)
next_action: Return ROOT CAUSE FOUND to the diagnose-issues workflow. No fix applied
  (goal: find_root_cause_only).

reasoning_checkpoint:
  hypothesis: |
    Two independent causes. RC-1: DetailPane.svelte:78 renders `{item.title}` through a
    plain Svelte text binding because 06-01-PLAN.md never listed the title as a UI-09
    surface — its must_haves.truths and success_criteria enumerate only `bodyVariant`
    branches. RC-2: StreamRow.svelte:121's `font-semibold` was authored in Phase 3
    (commit 4619cf2, 2026-07-31) under an explicit 03-UI-SPEC.md rule choosing weight
    over colour; Phase 6 added an amber `<mark>`/`.search-highlight` vocabulary without
    revisiting it.
  confirming_evidence:
    - "DetailPane.svelte imports highlightText at line 9 and calls it at line 110 ONLY, inside loadedTextBlock; line 78's <h2> uses a raw text binding."
    - "06-01-PLAN.md lines 30-34 (must_haves.truths) and line 318 (success_criteria) name only text/media/html body variants. The word 'title' appears nowhere in the plan's UI-09 scope."
    - "06-01-PLAN.md line 223 instructs Task 2 to 'replace the raw text binding inside the loadedTextBlock snippet' — an exhaustive instruction that excludes the header by construction."
    - "git blame StreamRow.svelte:121 -> commit 4619cf2 'feat(03-04)' dated 2026-07-31 (Phase 3). Phase 6's only StreamRow commit, 899504f, is the date-marker overlay and does not touch this block."
    - "03-UI-SPEC.md:69 and :134 state the semibold choice explicitly and give its rationale ('rather than introducing a highlight color', preserving the 'exactly 2 weights' contract)."
    - "kernel/index/schema.go:86-96 indexes BOTH title and preview into items_fts; store.go:404 uses snippet(items_fts, -1, ...) which auto-selects the best-matching column; store.go:415-417's doc confirms 'indexed title or preview match'."
    - "All 188 web unit tests across 11 files pass — the implementation satisfies its own written spec."
  falsification_test: |
    RC-1 would be falsified if 06-01-PLAN.md, 06-UI-SPEC.md or REQUIREMENTS.md named the
    detail-pane title as a highlight surface (making it an implementation miss rather than
    a scope gap). Grepped all three: it does not.
    RC-2 would be falsified if git blame put StreamRow.svelte:121 inside a Phase 6 commit,
    or if 03-UI-SPEC.md were silent on match styling. Neither holds.
  fix_rationale: |
    Not applying a fix (diagnose-only mode). The correct remediation targets the SCOPE, not
    a bug: extend UI-09's surface set to include the SPA-rendered title, and unify the two
    match-emphasis vocabularies onto one token. Patching DetailPane.svelte:78 alone would
    close the visible half of RC-1 while leaving RC-2's cross-pane inconsistency, and
    would leave StreamRow's own title (line 67-69) unhighlighted too.
  blind_spots: |
    Not verified in a live browser — diagnosis is from source, planning docs and git
    history only (read-only constraint, running in parallel with two sibling agents on the
    main tree). The exact perceived contrast of font-semibold vs the amber mark is taken
    from the user's report rather than measured. Whether the user also wants StreamRow's
    own title line highlighted is an open product question, not a code finding.
  candidate_causes:
    - "code: DetailPane.svelte:78 omits the highlightText call present at line 110 (SURFACE symptom of RC-1, not the cause)"
    - "spec/scope: 06-01-PLAN.md's must_haves + success_criteria scope UI-09 to bodyVariant branches only; UI-09's own wording 'the detail pane's rendered content' is ambiguous about the header (ROOT of RC-1)"
    - "design/config: 03-UI-SPEC.md:69 fixed match emphasis at semibold-weight-not-colour to protect the two-weight typography contract; 06 added an amber token without reconciling (ROOT of RC-2)"
    - "data: items_fts indexes title AND preview, so a title-only match is a first-class search outcome that RC-1 renders entirely unexplained (AMPLIFIER — promotes RC-1 from cosmetic to functional)"
  and_gate: |
    YES — the two sub-symptoms have separate, non-overlapping causes in different
    categories (Phase 6 spec scope vs Phase 3 design decision). Neither causes the other;
    fixing either alone leaves the other user-visible. `root_cause` is therefore a SET of
    two. The data-layer finding (title is FTS-indexed) is not a third cause but the
    amplifier that makes RC-1 a functional gap rather than a cosmetic one.

## Symptoms

expected: |
  After an in-webspace search, matched terms are highlighted in the detail pane's rendered
  content. Test expectation (06-UAT.md Test 1): "Matched terms render inside a `<mark>`
  element with an amber background across all three iframe content shapes; with no search
  query, output is byte-identical to pre-phase."

actual: |
  1. Detail pane TITLE (SPA-rendered, outside the iframe): no search term highlight shown.
  2. Detail pane BODY (iframe): amber highlight inside a <mark> element — WORKS.
  3. Stream pane: the search term renders as <span class="font-semibold"> — barely visible,
     not the amber <mark> treatment.

errors: none reported

reproduction: |
  Test 1 in .planning/phases/06-ui-scalable-source-surface/06-UAT.md — run a search in a
  webspace, open a matched item.

started: Discovered during Phase 6 UAT (2026-08-07).

## Eliminated

- hypothesis: "highlightText is broken or not imported into DetailPane.svelte"
  evidence: |
    It is imported (DetailPane.svelte:9) and correctly invoked (line 110). All 12
    highlight.test.ts cases pass within a fully green 188-test suite. The function is
    sound; it is simply never called on `item.title`.
  timestamp: 2026-08-07T00:12:00Z

- hypothesis: "Phase 6 regressed the stream pane's match styling from amber to semibold"
  evidence: |
    git blame puts StreamRow.svelte:121 at commit 4619cf2 'feat(03-04)' (2026-07-31),
    three phases before Phase 6. Phase 6's only commit touching this file (899504f,
    date-marker overlay) does not modify the snippet block. There was never an amber
    treatment in the stream to regress FROM.
  timestamp: 2026-08-07T00:14:00Z

- hypothesis: "The kernel highlighter fails to reach the title because the title lives outside the iframe"
  evidence: |
    True but not causal. The title is SPA-rendered and would be highlighted client-side by
    highlightText, exactly as the text/media body variants are (DetailPane.svelte:110-112).
    The mechanism exists and is already wired into this very component; only the call is
    absent. The sandbox boundary is irrelevant to the title.
  timestamp: 2026-08-07T00:15:00Z

- hypothesis: "searchQuery is not threaded down to DetailPane, so no surface could highlight"
  evidence: |
    `searchQuery` IS declared as a prop (DetailPane.svelte:23-33) and consumed at both
    line 110 (client highlight) and line 181 (iframe `?hl=`). Threading is correct; the
    body highlight demonstrably works per the user's own report.
  timestamp: 2026-08-07T00:16:00Z

## Evidence

- timestamp: 2026-08-07T00:08:00Z
  checked: web/src/lib/components/DetailPane.svelte
  found: |
    Line 9 imports `highlightText`. Line 110 calls it — inside the `loadedTextBlock`
    snippet only. Line 78 renders the title as `<h2 class="...">{item.title}</h2>`, a
    plain Svelte text binding with no segmentation. Lines 227-232 define
    `.search-highlight { background-color: var(--warning); color: var(--background); ... }`.
  implication: |
    The highlighting machinery is present, correct and already in this component. The
    title is simply not routed through it. A one-line-shaped omission — but see the plan
    evidence below for WHY it was omitted.

- timestamp: 2026-08-07T00:09:00Z
  checked: .planning/phases/06-ui-scalable-source-surface/06-01-PLAN.md
  found: |
    must_haves.truths (lines 30-34) enumerate exactly: the three iframe content shapes,
    the plain-text body, and the media variant's trailing text block. success_criteria
    line 318: "UI-09 is satisfied for every detail-pane body VARIANT: `text`, the trailing
    text of `media`, and all three `html` content shapes." Task 2's action (line 223):
    "replace the raw text binding inside the `loadedTextBlock` snippet". The word "title"
    never appears as a highlight surface anywhere in the plan.
  implication: |
    ROOT of RC-1. The implementation faithfully executed the plan. The defect is in the
    plan's scope derivation, which read UI-09's "rendered content" as the body region and
    never surfaced the title as a candidate surface — so no truth, no acceptance criterion
    and no test ever covered it.

- timestamp: 2026-08-07T00:10:00Z
  checked: .planning/REQUIREMENTS.md:44 (UI-09 wording)
  found: |
    "After an in-webspace search, matched terms are highlighted in the detail pane's
    rendered content".
  implication: |
    The ambiguity that seeded RC-1. "The detail pane's rendered content" admits both the
    planner's narrow reading (the fetched/rendered body) and the user's broad reading
    (everything the pane displays, including its header). The requirement never
    disambiguated, and 06-UI-SPEC.md did not pin it down either.

- timestamp: 2026-08-07T00:11:00Z
  checked: kernel/index/schema.go:86-96 and kernel/index/store.go:392-422
  found: |
    `CREATE VIRTUAL TABLE items_fts USING fts5(title, preview, ...)` — BOTH columns are
    indexed, and all three sync triggers write both. The search SELECT uses
    `snippet(items_fts, -1, ?, ?, '…', 12)`; FTS5's `-1` column argument means
    "auto-select the column with the best match". Store.Search's doc comment (lines
    415-417) states results are items "whose indexed title or preview match rawQuery".
  implication: |
    THE AMPLIFIER — this is what turns RC-1 from cosmetic into functional. A title-only
    match is a first-class, routinely reachable search outcome. For such an item the term
    appears nowhere in the body, so the detail pane renders ZERO highlight on ANY surface,
    and the user is given no visible explanation for why that item surfaced at all. The
    one surface carrying the match evidence is precisely the one never highlighted.

- timestamp: 2026-08-07T00:13:00Z
  checked: git blame + git log on web/src/lib/components/StreamRow.svelte
  found: |
    Line 121 `<span class={segment.match ? 'font-semibold' : undefined}>` blames to commit
    4619cf2 "feat(03-04): search a webspace from the browser and open a result", authored
    2026-07-31. The file's full commit list: 19c87e8 (01-03), 03760de (02-03), 9227e57
    (03-01), 4619cf2 (03-04), 899504f (06-03, date-marker overlay). The Phase 6 commit
    does not touch lines 117-124.
  implication: |
    ROOT of RC-2, and confirmation that sub-symptom 3 is NOT a Phase 6 regression. It is
    Phase 3 behaviour that Phase 6 left untouched — consistent with the phase scope note
    "the stream is unaffected since it already filters to matches."

- timestamp: 2026-08-07T00:14:00Z
  checked: .planning/phases/03-email-in-the-webspace/03-UI-SPEC.md lines 69, 73, 134, 135
  found: |
    Line 69: "search-result matched-term emphasis (E2 below) uses the already-declared
    semibold (600) weight applied inline within body-role (16px/400/1.5) snippet text,
    rather than introducing a highlight color. This keeps the 'exactly 2 weights' contract
    literally true." Line 134: "matched terms render in the existing semibold (600) weight
    ... rather than a new color."
  implication: |
    RC-2 is a DELIBERATE, documented design decision with a stated rationale — not an
    oversight and not dead code. Any remediation must consciously overturn a recorded
    Phase 3 contract (and account for its typography-contract rationale), rather than
    treat line 121 as a bug to patch.

- timestamp: 2026-08-07T00:17:00Z
  checked: web/src/app.css:107,147 · kernel/httpapi/rendition.go:167,233 · DetailPane.svelte:227-232
  found: |
    `--warning: #fbbf24` in both themes. The kernel emits `mark { background: #fbbf24;
    color: #020617; ... }` plus an email-shape `body mark, body mark *` !important
    restoring rule. DetailPane's `.search-highlight` expresses the identical treatment via
    `var(--warning)`/`var(--background)`.
  implication: |
    Phase 6 established a single, coherent amber vocabulary for "this text matched your
    search" across the kernel iframe AND the SPA text body — and then left a third,
    much weaker treatment (semibold weight) live in the adjacent search-results pane. The
    inconsistency the user perceives is real and spans two phases' design contracts.

- timestamp: 2026-08-07T00:18:00Z
  checked: web/src/lib/components/SearchResults.svelte
  found: |
    Line 3 imports StreamRow; lines 72-79 render `<StreamRow ... snippet={result.snippet}>`.
    The unfiltered stream passes no `snippet` prop at all (StreamRow lines 117/125 branch
    on `snippet !== undefined`).
  implication: |
    Confirms the `font-semibold` span only ever appears in SEARCH RESULTS rows, never in
    the plain stream — exactly matching the user's report, and confirming the surface to
    change is StreamRow's snippet branch (shared by SearchResults) rather than the stream.

- timestamp: 2026-08-07T00:19:00Z
  checked: cd web && npx vitest run
  found: "Test Files 11 passed (11) · Tests 188 passed (188)"
  implication: |
    Nothing is broken in the automated sense. The implementation satisfies every assertion
    written for it. This is the signature of a scope/specification gap rather than a coding
    defect — the tests encode the same narrow scope the plan did, so they could never have
    caught either sub-symptom.

## Resolution

root_cause: |
  TWO independent causes (AND-gate fired):

  RC-1 (specification scope gap, Phase 6) — the detail-pane TITLE was never inside UI-09's
  implemented scope. `DetailPane.svelte:78` renders `{item.title}` through a plain Svelte
  text binding while `highlightText` is imported at line 9 and applied at line 110 to the
  body text block only. This faithfully executes 06-01-PLAN.md, whose must_haves.truths
  (lines 30-34), success_criteria (line 318) and Task 2 action text (line 223) all scope
  UI-09 to `detailBodyVariant` branches — the word "title" appears nowhere. The seed is
  UI-09's own ambiguous wording, "the detail pane's rendered content" (REQUIREMENTS.md:44),
  which the planner read as the body region and the user reads as the whole pane.
  AMPLIFIER: `items_fts` indexes BOTH title and preview (kernel/index/schema.go:86-96) and
  `snippet(items_fts, -1, ...)` auto-selects the best-matching column (store.go:404), so a
  TITLE-ONLY match is a first-class search outcome. For such an item the detail pane shows
  zero highlight on any surface — the user gets no explanation for why it surfaced. This
  is what makes RC-1 functional rather than cosmetic.

  RC-2 (cross-phase design-vocabulary drift, pre-existing) — `StreamRow.svelte:121`'s
  `font-semibold` match treatment is Phase 3 code (commit 4619cf2 "feat(03-04)",
  2026-07-31), NOT a Phase 6 regression. It implements a deliberate, documented decision:
  03-UI-SPEC.md:69 and :134 chose the existing semibold weight "rather than introducing a
  highlight color" to preserve the "exactly 2 weights" typography contract. Phase 6 then
  introduced a far stronger amber vocabulary (`#fbbf24`/`--warning`) for the same semantic
  concept in the kernel iframe and the SPA text body, and — per its own scope note, "the
  stream is unaffected since it already filters to matches" — never reconciled the two.
  The result: one concept, two treatments of wildly different salience, in adjacent panes.

fix: (not applied — goal: find_root_cause_only; remediation belongs to plan-phase --gaps)

verification: (n/a — diagnose-only)

files_changed: []

artifacts:
  - path: web/src/lib/components/DetailPane.svelte
    issue: "Line 78 renders {item.title} as a plain text binding; highlightText (imported line 9, used line 110) is never applied to the title. The .search-highlight class at lines 227-232 already exists and is reusable verbatim."
  - path: web/src/lib/components/StreamRow.svelte
    issue: "Line 121 applies 'font-semibold' to matched snippet segments (Phase 3 / commit 4619cf2) instead of the Phase 6 amber treatment. Line 68 also renders {item.title} unhighlighted, so a title-only match is unmarked here too."
  - path: .planning/phases/06-ui-scalable-source-surface/06-01-PLAN.md
    issue: "must_haves.truths (30-34), success_criteria (318) and Task 2 action (223) scope UI-09 to bodyVariant branches only — the title surface was never derived, so no test could cover it."
  - path: .planning/REQUIREMENTS.md
    issue: "UI-09's wording 'the detail pane's rendered content' (line 44) is ambiguous between body-only and whole-pane; this ambiguity seeded the scope gap."
  - path: .planning/phases/03-email-in-the-webspace/03-UI-SPEC.md
    issue: "Lines 69/134 fix match emphasis at semibold-weight-not-colour. Overturning this is a conscious contract change, not a bug patch — the two-weight typography rationale must be addressed."
  - path: kernel/index/schema.go
    issue: "Not defective — but lines 86-96 (items_fts indexes title AND preview) are the reason a title-only match exists and therefore why the unhighlighted title is a functional gap."

missing:
  - "highlightText applied to the detail-pane title (DetailPane.svelte:78), reusing the existing .search-highlight class"
  - "A single shared match-emphasis token/class used by DetailPane, StreamRow and the kernel mark rule, so the amber treatment cannot drift between panes again"
  - "A decision record overturning (or consciously preserving) 03-UI-SPEC.md:69's weight-not-colour rule, since the amber treatment adds a colour the two-weight contract deliberately avoided"
  - "Consideration of StreamRow.svelte:68's own title line, unhighlighted for the same reason as the detail pane's"
  - "A component-level test asserting a title-only match renders a visible highlight somewhere — the current 188-test suite encodes the same narrow scope as the plan and cannot catch this class"
