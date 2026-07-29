---
status: complete
phase: 02-two-sources-one-trustworthy-stream
source: [02-VERIFICATION.md]
started: 2026-07-29T10:50:00Z
updated: 2026-07-29T18:40:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Visual confirmation of health chips, filter chips, and staleness states in a real browser
expected: Two health chips with correct colors; unreachable tooltip with relative time and full error; amber stale markers on SilverBullet rows only; cached detail pane with unreachable alert (never blank); source filter chips update the list and `source` query parameter, surviving reload — all matching 02-UI-SPEC.md.
result: issue
reported: "everything passes as described, however the tooltip when hovering over the health chip is only about 10px wide and so none of the text is readable. There is a similar issue on the index page too where something clickable exists below the title 'webspaces' but is also only a few pixels wide and displaying no text. Clicking it takes you to /w/house-move. Other than the 2 styling issues, looks perfect"
severity: major

### 2. PLUG-05 third-party self-sufficiency from a genuinely fresh/isolated context
expected: A truly isolated agent (real dispatched subagent, or human with no prior exposure to this repository) builds a SourcePlugin using only proto/webspaces/v1/plugin.proto, docs/plugin-contract.md, the sdk module, and plugins/mock — no access to plugins/paperless or plugins/silverbullet — producing a clean build with zero or few gaps, corroborating 02-04's in-session two-gap-then-zero-gap result.
result: pass
source: automated
evidence: "Fresh general-purpose subagent (no session context, isolation rules barring all non-contract paths) built a 'flatfile' SourcePlugin from only the four allowed inputs: go build/go vet clean, 13/13 tests pass. Verdict: self-sufficient with minor gaps (5) — all interpretive (free-text Match semantics for a tag-less source, file:// deep links, mtime timestamps), none a missing contract fact. Betters 02-04's in-session two-gap result."

### 3. Final visual confirmation that G-02-1's fix renders correctly in a real browser
expected: Index page column at normal width with full-width webspace cards (name and item count visible and clickable); health-chip tooltip on /w/house-move readable in full, including untruncated error text on an unreachable source; StreamEmpty's filtered-empty variant and StreamError copy at readable paragraph width where reachable. Widths 20rem/48rem/28rem per 02-UI-SPEC.md; no recurrence of the collapse reported in test 1. (Fix verified at CSS-mechanism level in 02-VERIFICATION.md — this is the final pixel-render confirmation.)
result: pass

## Summary

total: 3
passed: 2
issues: 1
pending: 0
skipped: 0
blocked: 0

## Gaps

- gap_id: G-02-1
  truth: "Hovering a source health chip shows a readable tooltip (relative time + full untruncated error), and the index page's webspace link renders its name as a normally-sized clickable element"
  status: resolved
  resolved_by: 02-06-PLAN.md
  resolved_at: 2026-07-29
  reason: "User reported: tooltip when hovering the health chip is only ~10px wide, none of the text readable; similar issue on the index page — a clickable element below the 'webspaces' title is a few pixels wide with no visible text (links to /w/house-move). All other Test 1 checks passed."
  severity: major
  test: 1
  root_cause: "web/src/app.css lines 45-52 place the UI-SPEC spacing tokens (--spacing-xs: 4px ... --spacing-3xl: 64px) inside the Tailwind v4 @theme inline block; --spacing-<key> theme entries shadow the default --container-<key> scale, so the built CSS compiles .max-w-xs{max-width:4px}, .max-w-md{max-width:16px}, .max-w-3xl{max-width:64px}. Tooltip (w-fit max-w-xs) collapses to ~10px; index page's <main class='mx-auto max-w-3xl'> collapses to 64px and Card's overflow-hidden clips the webspace link to a few px. Latent victims: StreamEmpty.svelte / StreamError.svelte (max-w-md = 16px), not rendered during UAT."
  artifacts:
    - path: "web/src/app.css"
      issue: "seven --spacing-<named> tokens in @theme inline shadow Tailwind v4's container scale (used by zero utilities — pure documentation in a live namespace)"
    - path: "web/src/lib/components/ui/tooltip/tooltip-content.svelte"
      issue: "victim via w-fit max-w-xs (symptom 1: ~10px tooltip)"
    - path: "web/src/routes/+page.svelte"
      issue: "victim via max-w-3xl on <main> (symptom 2: collapsed index column, clipped webspace link)"
    - path: "web/src/lib/components/StreamEmpty.svelte"
      issue: "latent victim via max-w-md (16px) — not rendered during UAT"
    - path: "web/src/lib/components/StreamError.svelte"
      issue: "latent victim via max-w-md (16px) — not rendered during UAT"
  missing:
    - "Remove/relocate the seven --spacing-<named> entries out of the @theme block (plain :root custom properties or renamed namespace); no consumers reference them as utilities"
    - "Rebuild and assert built CSS resolves .max-w-3xl to 48rem, .max-w-xs to 20rem, .max-w-md to 28rem"
    - "Extend e2e-smoke.sh stylesheet assertion to reject collapsed max-width values (recurrence guard)"
    - "Visually re-check tooltip, index link, StreamEmpty and StreamError after fix"
  debug_session: .planning/debug/resolved/collapsed-tooltip-and-index-link.md
  resolution: "Closed by plan 02-06 (commits 3a6d7ec, c34c5e1): removed the seven --spacing-<key> entries from web/src/app.css's @theme inline block. Re-verified in 02-VERIFICATION.md (2026-07-29T18:00Z) against the freshly built CSS — max-w-xs/md/3xl resolve to 20rem/28rem/48rem via var(--container-*). Recurrence guard: scripts/assert-stylesheet.sh, wired into scripts/e2e-smoke.sh. Final browser visual pass tracked as test 3."
