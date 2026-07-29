---
status: diagnosed
trigger: "G-02-1: the tooltip when hovering over the health chip is only about 10px wide and so none of the text is readable. There is a similar issue on the index page too where something clickable exists below the title 'webspaces' but is also only a few pixels wide and displaying no text. Clicking it takes you to /w/house-move. Other than the 2 styling issues, looks perfect"
created: 2026-07-29T12:30:00Z
updated: 2026-07-29T12:55:00Z
---

## Current Focus
<!-- OVERWRITE on each update - reflects NOW -->

hypothesis: "CONFIRMED: app.css @theme defines a project spacing scale as --spacing-xs/sm/md/lg/xl/2xl/3xl (4px…64px). In Tailwind v4, entries in the --spacing-* theme namespace become named keys for sizing utilities, shadowing the default --container-* scale that named max-w-* sizes resolve from. The built CSS proves it: .max-w-xs{max-width:4px}, .max-w-md{max-width:16px}, .max-w-3xl{max-width:64px} (should be 320px/448px/768px)"
test: "complete — built-CSS inspection, source grep, primitive reads, git archaeology"
expecting: "n/a — diagnosis complete"
next_action: "Return ROOT CAUSE FOUND to orchestrator (goal: find_root_cause_only — no fix applied)"

bug_class: Bohrbug (deterministic — values baked into the built stylesheet, identical every load)

known_pattern_candidate: "stream-ui-unstyled (G-01-2) — prior total-CSS-absence bug; its blind_spots field explicitly flagged residual Tailwind theme/coverage risk post-fix. This bug is that residual: it was latent since 01-01 but invisible while ZERO CSS shipped; the 01-05 import fix made the collision reachable"

reasoning_checkpoint:
  hypothesis: "The hand-authored --spacing-<named> tokens in app.css's @theme collide with Tailwind v4's spacing namespace; named sizing-utility keys (max-w-xs/md/3xl) resolve to those tiny spacing values instead of the default --container-* scale, collapsing every element that uses a named max-w-*"
  confirming_evidence:
    - "Built CSS (kernel/webui/build/_app/immutable/assets/0.DGXW7fvo.css, fresh Jul 29 10:38, UAT started 10:50): .max-w-xs{max-width:4px}, .max-w-md{max-width:16px}, .max-w-3xl{max-width:64px} — exactly the app.css token values (--spacing-xs:4px, --spacing-md:16px, --spacing-3xl:64px), inlined because the block is @theme inline"
    - "TooltipContent class list contains 'w-fit max-w-xs' → tooltip bubble capped at 4px max-width → ~10px visible bubble, text unreadable (symptom 1)"
    - "Index +page.svelte: <main class='mx-auto max-w-3xl px-6 py-12'> → main capped at 64px; px-6 leaves a ~16px content box. The h1 'webspaces' is a single word that visibly overflows (title readable), but Card.Root has overflow-hidden, so the card clips its own title text → a few-px-wide clickable <a> with no visible text that navigates to /w/house-move (symptom 2)"
    - "Everything else looked perfect because no other named-key sizing utility exists in the source: exhaustive grep finds only 5 named-size usages, all max-w-* (tooltip-content, +page.svelte, StreamEmpty ×2, StreamError). Numeric spacing utilities (px-3, gap-1.5, size-2…) resolve via the intact base --spacing:.25rem multiplier"
    - "The seven --spacing-<named> tokens are used NOWHERE as utilities (no p-md/gap-lg/etc. anywhere in web/src) — they are pure liability, added as documentation of the UI-SPEC spacing scale"
  falsification_test: "If the collision were not the cause, the built CSS would show .max-w-xs{max-width:var(--container-xs)} (or 20rem) and the collapse would need a DOM/runtime explanation. Direct observation refutes this: the built rules carry the literal spacing-token pixel values."
  fix_rationale: "n/a — diagnose-only mode. Fix direction: remove the --spacing-<named> entries from the @theme namespace (delete, or relocate as plain :root custom properties / rename outside --spacing-*), so named max-w-* keys fall back to the default --container-* scale; rebuild and assert .max-w-3xl{max-width:48rem} in the built CSS"
  blind_spots: "Not verified by rebuild (diagnose-only). Tooltip visible-width arithmetic (~10px vs 4px + border-box padding floor) not pixel-verified in a browser — immaterial to the mechanism. StreamEmpty/StreamError max-w-md collapse (latent same-bug states) inferred from built CSS, not yet rendered in any UAT session."
  candidate_causes:
    - "code (authored CSS/theme): --spacing-* namespace collision in app.css @theme (CONFIRMED)"
    - "config/build: stale embedded build in kernel/webui/build (ELIMINATED — assets timestamped Jul 29 10:38 same morning as UAT 10:50, and the CSS contains Phase-02 tooltip classes, so it reflects current source)"
    - "code (components): shadcn primitives failing to render children snippets → genuinely empty elements (ELIMINATED — card.svelte/card-header/card-title/tooltip-content all {@render children?.()} correctly; the index link demonstrably carries its href, and the collision fully explains both symptoms without an empty-render)"
    - "environment (browser): UA/dark-mode quirk (ELIMINATED — the wrong values are baked into the shipped stylesheet; deterministic across environments)"
  and_gate: "no — one root cause explains both reported symptoms. Card.Root's overflow-hidden is a second CONDITION shaping symptom 2's appearance (why the card shows no text while the sibling h1 overflows readably), but it is intended shadcn behavior, not a defect; absent the max-width collapse it is harmless."

## Symptoms
<!-- Written during gathering, then IMMUTABLE -->

expected: "Hovering a source health chip shows a readable tooltip containing a relative last-sync time and the full untruncated last error. The index page ('webspaces' title) lists each webspace as a normally-sized clickable link showing its name."
actual: "Tooltip on health chip hover is ~10px wide, none of the text readable. Index page: a clickable element below the 'webspaces' title is a few pixels wide, displaying no text; clicking navigates to /w/house-move. All other UI in the same session correct — colors, stale markers, filter chips, detail pane."
errors: "None reported (no console errors mentioned)"
reproduction: "Test 1 in .planning/phases/02-two-sources-one-trustworthy-stream/02-UAT.md — bin/webspaces serve (127.0.0.1:7777, embedded production build), open / and /w/house-move, hover the SilverBullet health chip"
started: "Discovered during Phase 02 UAT (2026-07-29). Tooltip is Phase 02 (SourceHealthChip.svelte, plan 02-03); index webspace link predates it (Phase 01). Identical symptom → prefer shared root cause."

## Eliminated
<!-- APPEND only - prevents re-investigating -->

- hypothesis: "Shared component/primitive fails to render its children snippet, producing genuinely empty elements"
  evidence: "card.svelte, card-header.svelte, card-title.svelte, card-description.svelte, tooltip-content.svelte all render {@render children?.()} correctly; +page.svelte passes {ws.name} into Card.Title and SourceHealthChip passes {tooltipText} into TooltipContent. Content reaches the DOM — the collapse is a CSS max-width cap (+ overflow-hidden clipping on the card), not missing content."
  timestamp: 2026-07-29T12:45:00Z

- hypothesis: "Stale embedded build — kernel/webui/build predates the Phase 02 source"
  evidence: "Build assets timestamped Jul 29 10:38 (UAT started 10:50 same day). The built CSS contains the Phase-02 tooltip-content classes (origin-(--bits-tooltip-content-transform-origin)) and the card classes — it reflects current source."
  timestamp: 2026-07-29T12:47:00Z

- hypothesis: "Tailwind v4 source scanning misses the ui/ primitive files, so their utilities are absent from the built CSS"
  evidence: "The utilities are PRESENT in the built CSS (.w-fit, .max-w-xs, card-header's has-data-[slot=card-action] rules, container:card-header/inline-size). The problem is the RESOLVED VALUE of named max-w keys, not missing rules."
  timestamp: 2026-07-29T12:48:00Z

## Evidence
<!-- APPEND only - facts discovered -->

- timestamp: 2026-07-29T12:30:00Z
  checked: ".planning/debug/stream-ui-unstyled.md (prior diagnosed session, G-01-2) + STATE.md decisions"
  found: "Prior bug: app.css never imported → zero CSS in build. Fixed in 01-05. That session's blind_spots warned residual theme/coverage risk. Also relevant: shadcn registry retired baseColor/style; all colors hand-authored in app.css from UI-SPEC tokens."
  implication: "Knowledge-base match — check theme/built-CSS first."

- timestamp: 2026-07-29T12:40:00Z
  checked: "web/src/lib/components/SourceHealthChip.svelte + web/src/routes/+page.svelte"
  found: "Tooltip content is shadcn TooltipContent receiving derived tooltipText. Index link is <a href=/w/{name}> wrapping shadcn Card.Root > Card.Header > Card.Title({ws.name}). Both collapsed elements are shadcn primitives; chip label itself (plain spans) renders fine."
  implication: "Both symptoms flow through shadcn primitive class lists — inspect those classes."

- timestamp: 2026-07-29T12:44:00Z
  checked: "ui/card/*.svelte and ui/tooltip/*.svelte primitive sources"
  found: "All render children correctly. TooltipContent class list: 'inline-flex … px-3 py-1.5 … w-fit max-w-xs …'. Card.Root: '… overflow-hidden rounded-xl py-(--card-spacing) … flex flex-col'. Card.Header: grid + @container/card-header."
  implication: "Key suspects: w-fit max-w-xs (tooltip) and whatever constrains the index page width; card overflow-hidden explains text clipping if width collapses."

- timestamp: 2026-07-29T12:47:00Z
  checked: "kernel/webui/build/_app/immutable/assets/0.DGXW7fvo.css (34KB, Jul 29 10:38) — rule bodies"
  found: ".max-w-xs{max-width:4px} .max-w-md{max-width:16px} .max-w-3xl{max-width:64px}. .w-fit{width:fit-content} correct. --spacing:.25rem present (numeric utilities healthy). --container-xs never defined/referenced (0 occurrences)."
  implication: "SMOKING GUN: named max-w keys resolved against app.css's --spacing-<named> tokens (4px/16px/64px), not the default --container-* scale (20rem/28rem/48rem). Values are inlined because the block is @theme inline."

- timestamp: 2026-07-29T12:50:00Z
  checked: "web/src/app.css full read"
  found: "@theme inline block (lines 15-53) includes '/* Spacing scale (4px multiples) per 01-UI-SPEC.md */ --spacing-xs:4px; --spacing-sm:8px; --spacing-md:16px; --spacing-lg:24px; --spacing-xl:32px; --spacing-2xl:48px; --spacing-3xl:64px'."
  implication: "The colliding tokens are hand-authored UI-SPEC documentation placed into Tailwind's live --spacing-* theme namespace."

- timestamp: 2026-07-29T12:52:00Z
  checked: "Exhaustive grep of web/src for named-key sizing utilities and for intended token usages (p-md, gap-lg, etc.)"
  found: "Only 5 named-size usages exist, ALL max-w-*: tooltip-content.svelte (max-w-xs), routes/+page.svelte (max-w-3xl on <main>), StreamEmpty.svelte ×2 + StreamError.svelte (max-w-md). ZERO usages of the spacing tokens as utilities anywhere."
  implication: "Explains the precise blast radius: only the tooltip and index page visibly broke; StreamEmpty/StreamError are latent same-bug states (16px max-width) not yet rendered in UAT. The tokens themselves are dead weight — removable without touching any consumer."

- timestamp: 2026-07-29T12:54:00Z
  checked: "git log --follow web/src/app.css; git log -S 'spacing-xs'"
  found: "Tokens introduced in da15f94 (plan 01-01 scaffold) — the same commit that omitted the app.css import (G-01-2). Latent through Phase 01 because no CSS shipped at all until the 01-05 fix; index page evidently first eyeballed during Phase 02 UAT."
  implication: "Timeline coherent: one Phase-01 scaffold commit seeded both UAT styling gaps; this one became observable only after the first was fixed."

## Resolution
<!-- OVERWRITE as understanding evolves -->

root_cause: "web/src/app.css places the 01-UI-SPEC 'Spacing Scale' tokens (--spacing-xs:4px, --spacing-sm:8px, --spacing-md:16px, --spacing-lg:24px, --spacing-xl:32px, --spacing-2xl:48px, --spacing-3xl:64px) inside the Tailwind v4 @theme inline block. In Tailwind v4, --spacing-<key> theme entries create named keys for the sizing-utility namespace and SHADOW the default --container-<key> scale that named max-width sizes resolve from. Consequently the built stylesheet compiles .max-w-xs to max-width:4px (default: 20rem), .max-w-md to 16px (default: 28rem), and .max-w-3xl to 64px (default: 48rem). Symptom 1: shadcn TooltipContent's own class list is 'w-fit max-w-xs', so the health-chip tooltip bubble is capped at 4px — a ~10px sliver with unreadable text. Symptom 2: the index page's <main> uses 'mx-auto max-w-3xl px-6', capping the whole page column at 64px with a ~16px content box; the single-word h1 'webspaces' visibly overflows and stays readable, but the webspace link's Card.Root has overflow-hidden, so the card clips its title to a few-px-wide, text-less — yet clickable — element linking to /w/house-move. Latent additional victims: StreamEmpty and StreamError use max-w-md (now 16px) but haven't been rendered in UAT yet. Present since scaffold commit da15f94 (plan 01-01); masked until the 01-05 app.css-import fix made any CSS ship at all. The seven tokens are referenced by zero utilities in the codebase — pure documentation placed in a live namespace."
fix: ""
verification: ""
files_changed: []
