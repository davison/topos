---
phase: 06-ui-scalable-source-surface
plan: 01
subsystem: ui
tags: [go, golang.org/x/net/html, bluemonday, svelte, sveltekit, search, sanitization, ui-09]

# Dependency graph
requires:
  - phase: 05-source-instances-per-type-matching
    provides: "kernel/httpapi/rendition.go's D-11 sanitize/wrap/theme pipeline (sanitizeAndWrapRendition, renditionPolicies, stylesheetForShape) that this plan's highlighting step inserts into"
provides:
  - "Kernel-side tree-walk highlighter (highlightTerms + highlightTextNodes + highlightSanitizedFragment) inserted between sanitize and wrap in kernel/httpapi/rendition.go"
  - "The optional ?hl= query parameter on GET /api/items/{id}/content, threaded through renditionHandler in item.go"
  - "Client-side highlighter (highlightTerms + highlightText) in web/src/lib/format.ts implementing the identical term-derivation rule"
  - "DetailPane.svelte searchQuery prop, wired into both the html-branch iframe src (contentUrl) and the shared loadedTextBlock snippet"
  - "10 kernel hardening tests proving the highlighter never touches attributes, tag bytes, or the chat class allowlist, and never re-enters the sanitizer"
  - "15 client-side unit tests covering term derivation, literal metacharacter matching, longest-first tie-breaking, and the segment round-trip invariant"
affects: [06-02, 06-03, ui-verify, docs/api.md]

# Actuals (#2632) — pairs with the plan's `estimate` to calibrate future estimates.
# Same estimateTokens scale (chars/4 over the realized diff), never a harness token count.
actuals:
  tokens: 13566
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Kernel-side tree-walk HTML mutation (parse/walk/render via golang.org/x/net/html) for any text-content transform that must apply to already-sanitized HTML, instead of byte/regex substitution over sanitized bytes"
    - "Shared client/kernel term-derivation rule duplicated deliberately in two languages (kernel/httpapi/rendition.go's highlightTerms and web/src/lib/format.ts's highlightTerms), each doc-commented to point at its sibling so they cannot silently diverge"

key-files:
  created:
    - web/src/lib/components/highlight.test.ts
  modified:
    - kernel/httpapi/rendition.go
    - kernel/httpapi/item.go
    - kernel/httpapi/agent.go
    - kernel/httpapi/rendition_test.go
    - go.mod
    - web/src/lib/api.ts
    - web/src/lib/format.ts
    - web/src/lib/components/DetailPane.svelte
    - web/src/routes/w/[webspace]/+page.svelte
    - docs/api.md

key-decisions:
  - "Task 1 (tracer) was committed and its <verify><automated> suite run to completion; its <human-check> line (make dev + live browser verification) was left embedded rather than synthesized into a mid-flight checkpoint, per .planning/config.json's workflow.human_verify_mode: end-of-phase (#3309) and this plan's own autonomous: true frontmatter — the phase-level verifier harvests it into 06-UAT.md at end-of-phase alongside every other plan's human-check lines."
  - "golang.org/x/net promoted from indirect to direct in go.mod via a hand-edit of the require-block comment (a targeted go get did not itself strip the // indirect marker under this repo's go.work workspace setup, and go mod tidy is explicitly avoided per the 05-04 precedent) — every other entry in go.mod's require block is also marked // indirect regardless of actual directness, a pre-existing repo-wide staleness this plan does not attempt to fix."
  - "Task 2's DetailPane.svelte highlight span uses a scoped <style> block referencing var(--warning)/var(--background) rather than Tailwind bg-warning/text-background utility classes, since the plan's action text called for 'the existing Tailwind theme tokens' and both custom properties are already globally available in the SPA document."

requirements-completed: [UI-09]

coverage:
  - id: D1
    description: "Kernel highlights matched search terms inside html-variant detail-pane content (email/markdown/chat) via a tree-walk <mark> insertion between sanitize and wrap, reached through the SPA's iframe src"
    requirement: "UI-09"
    verification:
      - kind: unit
        ref: "kernel/httpapi/rendition_test.go#TestHighlight_TextNodesOnly_AttributesUntouched"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/rendition_test.go#TestHighlight_ChatClassAllowlistSurvives"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/rendition_test.go#TestHighlight_CaseInsensitiveOriginalCasingPreserved"
        status: pass
    human_judgment: true
    rationale: "Visual amber-highlight confirmation inside the sandboxed iframe across all three real content shapes (email/SilverBullet/Signal) requires make dev plus a live browser check per this plan's own <human-check> line — deferred to end-of-phase UAT per workflow.human_verify_mode: end-of-phase."
  - id: D2
    description: "Client highlights matched search terms inside text/media-variant detail-pane content using the identical term-derivation rule as the kernel"
    requirement: "UI-09"
    verification:
      - kind: unit
        ref: "web/src/lib/components/highlight.test.ts#highlightText > resolves overlapping terms longest-first with no nested or duplicated segments"
        status: pass
      - kind: unit
        ref: "web/src/lib/components/highlight.test.ts#highlightText > the round-trip invariant: concatenating every segment text reproduces the input exactly"
        status: pass
      - kind: unit
        ref: "web/src/lib/components/highlight.test.ts#highlightText > matches a query containing regex metacharacters literally and never throws"
        status: pass
    human_judgment: true
    rationale: "Visual amber-highlight confirmation below a paperless preview box requires make dev plus a live browser check per this plan's own <human-check> line — deferred to end-of-phase UAT per workflow.human_verify_mode: end-of-phase."
  - id: D3
    description: "The sanitizer contract, CSP header, and sandbox directive are provably untouched by the highlighting change"
    requirement: "UI-09"
    verification:
      - kind: unit
        ref: "kernel/httpapi/rendition_test.go#TestHighlight_NoReSanitization"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/rendition_test.go#TestHighlight_EmptyTermsInert"
        status: pass
      - kind: other
        ref: "grep -rq 'allow-same-origin' kernel/httpapi/ (no match)"
        status: pass
    human_judgment: false

# Metrics
duration: ~20min
completed: 2026-08-06
status: complete
---

# Phase 6 Plan 1: Detail-Pane Search Highlighting Summary

**Kernel-side tree-walk `<mark>` insertion (golang.org/x/net/html) for html-variant rendition content, plus an identical client-side term-derivation rule for text/media variants, closing UI-09.**

## Performance

- **Duration:** ~20 min
- **Tasks:** 3
- **Files modified:** 10 modified, 1 created

## Accomplishments

- Kernel-side highlighter: `highlightTerms` (shared term-derivation rule) and `highlightTextNodes` (tree-walk `<mark>` insertion, never byte/regex substitution) in `kernel/httpapi/rendition.go`, inserted strictly between `policy.SanitizeBytes` and the document-wrap step via a new `highlightSanitizedFragment` helper. `sanitizeAndWrapRendition` gains a `terms []string` third parameter; the optional `?hl=` query parameter on `GET /api/items/{id}/content` (`item.go`) feeds it, while `agent.go`'s `/agent/v1` mirror deliberately passes `nil` (no search UI there — 06-RESEARCH.md Open Question 2, resolved).
- Client-side highlighter: `highlightTerms`/`highlightText` in `web/src/lib/format.ts`, implementing the identical rule as the kernel (whitespace-split, lowercase, de-dupe, 2-char floor, 8-term cap, longest-term tie-break) so what the client highlights in the text/media detail-pane variants can never disagree with what the kernel highlights inside the sandboxed iframe.
- `DetailPane.svelte` gains a `searchQuery` prop, threaded into the html-branch iframe `src` via `contentUrl(item.id, searchQuery)` and into the shared `loadedTextBlock` snippet (covering both the text-only and media trailing-text branches) via `highlightText`.
- 10 kernel hardening tests (`TestHighlight*`/`TestHighlightTerms_Derivation`) proving the highlighter touches rendered text only — attributes, tag bytes, the chat-transcript class allowlist, and the sanitizer boundary are all provably unaffected — plus 15 client-side unit tests (term derivation, literal metacharacter matching, longest-first tie-break, the segment-concatenation round-trip invariant).
- `docs/api.md` documents the `?hl=` parameter's semantics, bounds, and its explicit absence from the `/agent/v1` mirror.

## Task Commits

Each task was committed atomically:

1. **Task 1: End-to-end kernel-side highlighting — one search term reaches the sandboxed iframe** (tracer) - `65b0756` (feat)
2. **Task 2: Client-side highlighting for the plain-text and media body variants** - `d5e4956` (feat)
3. **Task 3: Kernel highlighting hardening tests and API documentation** - `5a6ae1f` (test)

_Plan-level metadata commit is made by the wave orchestrator after all worktree agents in this wave complete — not by this executor, per the parallel-execution contract._

## Files Created/Modified

- `kernel/httpapi/rendition.go` - `highlightTerms`, `highlightTextNodes`, `highlightTextNode`, `highlightSanitizedFragment`; `sanitizeAndWrapRendition` gains a `terms []string` parameter; `mark` rule added to `renditionBaseStyle`; restoring `body mark, body mark *` rule added to `renditionEmailReadabilityDelta`
- `kernel/httpapi/item.go` - `renditionHandler` reads the optional `?hl=` query param, derives terms via `highlightTerms`, threads them into `sanitizeAndWrapRendition`
- `kernel/httpapi/agent.go` - `/agent/v1` rendition call site passes `nil` terms with a recorded scope-boundary comment
- `kernel/httpapi/rendition_test.go` - 20 pre-existing call sites updated to the new 3-arg signature; 10 new `TestHighlight*` tests; extended `TestRenditionStylesheetTokensMatchAppCSS` token slice
- `go.mod` - `golang.org/x/net` promoted from indirect to direct (already resolved at v0.53.0)
- `web/src/lib/api.ts` - `contentUrl` gains an optional second `query` parameter, appended as `?hl=`
- `web/src/lib/format.ts` - `highlightTerms`/`highlightText` (client half of the shared rule)
- `web/src/lib/components/DetailPane.svelte` - `searchQuery` prop; html-branch iframe `src` and `loadedTextBlock` both wired to it; scoped `.search-highlight` style rule added
- `web/src/lib/components/highlight.test.ts` - new colocated vitest file, 15 test cases
- `web/src/routes/w/[webspace]/+page.svelte` - passes existing `searchQuery` state down to `<DetailPane>`
- `docs/api.md` - documents `?hl=` on `GET /api/items/{id}/content`, including its absence from `/agent/v1`

## Decisions Made

- Task 1's `<human-check>` line was left embedded in `<verify>` (not synthesized into a mid-flight `checkpoint:human-verify`), per `.planning/config.json`'s `workflow.human_verify_mode: end-of-phase` (#3309) and this plan's `autonomous: true` frontmatter. The phase-level verifier will harvest it into `06-UAT.md` at end-of-phase alongside every other embedded human-check line in this phase.
- `golang.org/x/net`'s `// indirect` marker was removed by direct edit of `go.mod`'s require block, since a targeted `go get golang.org/x/net@v0.53.0` did not itself rewrite the comment under this repo's `go.work` workspace setup, and `go mod tidy` on the root module is explicitly avoided per the Phase 05-04 precedent (it pulls unrelated synthetic transitives). Every other entry in the same require block is also marked `// indirect` regardless of actual directness — a pre-existing repo-wide staleness this plan does not attempt to fix.
- The client-side `.search-highlight` style rule (`DetailPane.svelte`) references `var(--warning)`/`var(--background)` directly in a scoped `<style>` block rather than composing Tailwind `bg-warning`/`text-background` utility classes on the span — both approaches resolve to the same tokens; the scoped-CSS form kept the rule and its doc comment colocated in one place.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Updated 20 pre-existing `sanitizeAndWrapRendition` call sites in `rendition_test.go` to the new 3-arg signature**
- **Found during:** Task 1
- **Issue:** Changing `sanitizeAndWrapRendition`'s signature to accept a third `terms []string` parameter is a compile-time forcing function for every existing caller, including every pre-existing test in `rendition_test.go` (not declared in Task 1's `files` list, but required for `go test ./kernel/httpapi/` to even compile).
- **Fix:** Every existing call site updated to pass `nil` as the third argument; three `t.Fatalf` string literals that happened to contain the substring `sanitizeAndWrapRendition(...)` as error-message text (not real calls) were correctly left untouched after an initial automated pass briefly corrupted them.
- **Files modified:** `kernel/httpapi/rendition_test.go`
- **Verification:** `go test ./kernel/httpapi/` exits 0 with every pre-existing assertion passing unchanged.
- **Committed in:** `65b0756` (Task 1 commit)

**2. [Rule 1 - Bug] Reworded a `rendition.go` CSS comment that embedded a literal `<mark>` string into the served stylesheet**
- **Found during:** Task 3
- **Issue:** A doc comment inside the `renditionBaseStyle` CSS constant literally read `attribute-free <mark> element`, which is served as part of every rendition document's `<style>` block — this text was itself defeating a Task 3 test's own `strings.Contains(got, "<mark")` non-corruption assertion (a false positive, not a real markup leak, but the comment's phrasing was still worth fixing for the same reason `{@html}` was avoided in a Task 2 comment).
- **Fix:** Reworded to "attribute-free mark element" (no angle brackets) — no functional change, comment text only.
- **Files modified:** `kernel/httpapi/rendition.go`
- **Verification:** `TestHighlight_TagBytesUntouched` (rewritten against the chat shape's real class vocabulary, since the original fragment incorrectly assumed the markdown policy allows a bare `class` attribute) now passes for the right reason.
- **Committed in:** `5a6ae1f` (Task 3 commit)

---

**Total deviations:** 2 auto-fixed (1 blocking compile dependency, 1 bug)
**Impact on plan:** Both fixes were necessary for the plan's own stated verification gates to pass truthfully. No scope creep — no behavior outside UI-09's declared surface was touched.

## Issues Encountered

- `web/node_modules` did not exist in this worktree (fresh checkout, never `npm install`ed). Ran `npm ci` against the existing `package-lock.json` — an ordinary project-bootstrap step, not a new-package install, so it did not require the package-legitimacy checkpoint carve-out. `npm test`/`npm run check` were unusable until this ran.
- A bash-tool transformation that added the third `nil` argument to every pre-existing `sanitizeAndWrapRendition` call in `rendition_test.go` briefly corrupted three `t.Fatalf` string-literal error messages that happened to contain the same function-name substring (not real calls). Caught immediately by a follow-up `git diff`/grep pass and corrected before the Task 1 commit; no corrupted state was ever committed.

## Known Stubs

None — every code path this plan touches is a real implementation with real `<verify>` coverage; no placeholder data or hardcoded empty state was introduced.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- UI-09 is closed for every detail-pane body variant (`text`, the trailing text of `media`, and all three `html` content shapes) at the automated-verification layer; the plan's own `<human-check>` lines (Task 1 and Task 2) are embedded in `<verify>` for end-of-phase UAT harvesting per `workflow.human_verify_mode: end-of-phase`, not executed by this run.
- The kernel's D-11 sanitize/wrap/theme pipeline (`sanitizeAndWrapRendition`) now carries a `terms []string` parameter that any future kernel-side rendition-content mutation would need to compose with, not duplicate — plan 06-02/06-03 (if they touch rendition content) should read this plan's doc comments on `rendition.go` before adding a second post-sanitize mutation step.
- `golang.org/x/net`'s `go.mod` entry is the only requirement in the root module's require block without a stale `// indirect` marker; a future phase that runs a real `go mod tidy` sweep should expect the whole block to be rewritten, not just this one line.

## Self-Check: PASSED

- FOUND: `kernel/httpapi/rendition.go`
- FOUND: `web/src/lib/components/highlight.test.ts`
- FOUND: `.planning/phases/06-ui-scalable-source-surface/06-01-SUMMARY.md`
- FOUND: commit `65b0756` (Task 1)
- FOUND: commit `d5e4956` (Task 2)
- FOUND: commit `5a6ae1f` (Task 3)
- FOUND: commit `7436641` (plan metadata)

---
*Phase: 06-ui-scalable-source-surface*
*Completed: 2026-08-06*
