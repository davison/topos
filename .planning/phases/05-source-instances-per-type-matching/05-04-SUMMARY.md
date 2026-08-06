---
phase: 05-source-instances-per-type-matching
plan: 04
subsystem: api
tags: [go, protobuf, bluemonday, sanitization, security-boundary, rendition]

# Dependency graph
requires:
  - phase: 05-source-instances-per-type-matching
    plan: 02
    provides: "Typed match fields and the topos.v2 contract generation this plan's proto edit builds on"
  - phase: 05-source-instances-per-type-matching
    plan: 03
    provides: "Per-instance match config this plan does not touch, but shares the phase's contract-break window with"
provides:
  - "kernel/httpapi/rendition.go — the kernel-owned sanitize/wrap/theme pipeline (sanitizeAndWrapRendition) with three content-shape policies (email, markdown, chat) composed from one shared stylesheet base plus per-shape deltas"
  - "proto ContentShape enum and FetchResponse.content_shape — the wire contract a plugin uses to declare which policy its text/html rendition needs"
  - "pluginhost.FetchResult.ContentShape carrying that field into the kernel domain"
  - "kernel/httpapi renditionHandler/agentRenditionHandler both route every text/html rendition through sanitizeAndWrapRendition and fail closed (unsupported_content_shape) on an unrecognised/unspecified shape"
  - "Three plugins (proton, silverbullet, signal) that return unwrapped, unsanitized content plus a declared shape instead of a wrapped document"
affects: [05-05, 06-ui-scalable-source-surface, 07-webspace-builder-ui]

# Actuals (#2632)
actuals:
  tokens: 34733
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added:
    - "github.com/microcosm-cc/bluemonday v1.0.27 added to the root module (kernel now owns sanitization directly; previously only the plugin modules depended on it)"
  patterns:
    - "Content-shape-keyed sanitize/wrap pipeline: one bluemonday.Policy per toposv1.ContentShape, one shared CSS base plus per-shape delta constants composed by stylesheetForShape, sanitize-then-wrap-never-reverse enforced by sanitizeAndWrapRendition's own structure"
    - "Fail-closed content boundary: a text/html rendition whose declared shape has no policy entry (including the zero value) is refused with a distinct error code and no body — mirrors the existing LinkFidelity/ContentVariant zero-value-is-UNSPECIFIED convention"
    - "Plugin escapes, kernel sanitizes: plugins/signal/render.go HTML-escapes every interpolated text field before assembly (its own structural-integrity guarantee) while the kernel's chat policy performs the actual security sanitization over the assembled fragment — two independent layers with distinct jobs"

key-files:
  created:
    - kernel/httpapi/rendition.go
    - kernel/httpapi/rendition_test.go
  modified:
    - go.mod
    - go.sum
    - proto/topos/v1/plugin.proto
    - sdk/gen/topos/v1/plugin.pb.go
    - sdk/contract_test.go
    - kernel/pluginhost/host.go
    - kernel/httpapi/item.go
    - kernel/httpapi/item_test.go
    - kernel/httpapi/agent.go
    - plugins/proton/body.go
    - plugins/proton/plugin.go
    - plugins/proton/fetch_rendition_test.go
    - plugins/silverbullet/render.go
    - plugins/silverbullet/render_test.go
    - plugins/silverbullet/plugin.go
    - plugins/signal/render.go
    - plugins/signal/render_test.go
    - plugins/signal/plugin.go
    - plugins/signal/fetch_test.go

key-decisions:
  - "Task 1's rendition.go references toposv1.ContentShape, so the proto enum/field and sdk/gen regeneration landed in Task 1's commit rather than Task 2's, under deviation Rule 3 (blocking compile dependency) — Task 1 could not build otherwise. Task 2 still owns FetchResult/item.go/agent.go/plugin wiring and the sdk/contract_test.go zero-value table row, matching its own declared scope."
  - "The email content-shape stylesheet carries forward proton's full 'readability layer' (the body/body* !important neutralizer) even though 05-UI-SPEC.md's Rendition Content Contract table doesn't spell it out — the plan's own must_haves require post-move visual parity with today's per-plugin output, and dropping the neutralizer would be a visible regression for any email whose inline styles previously needed it overridden."
  - "bluemonday's class-attribute Matching regexp is evaluated against a class attribute's ENTIRE value (verified against the vendored library source, not assumed from its docs) — not per space-separated token. chatTranscriptClassTokens is written to accept any sequence of the fixed token set, so a legitimate 'run own'/'bubble other' pair survives while any value containing an unrecognised token is stripped in full (the whole class attribute drops, not just the bad token)."
  - "plugins/signal/fetch_test.go and plugins/proton/fetch_rendition_test.go updated outside this plan's declared files_modified list under Rule 3: both asserted the plugin's Fetch response was a complete wrapped document (doctype prefix, embedded theme tokens) — an assertion the cutover makes false by design. Both now assert the raw fragment plus the declared ContentShape."

patterns-established:
  - "One sanitize/wrap/theme boundary at the content-serving edge, not duplicated per plugin — future plugin authors (the plugin-ecosystem milestone this rides ahead of) declare a content shape and return a fragment; they never author their own stylesheet or sanitizer."

requirements-completed: [KERN-07]

coverage:
  - id: D1
    description: "Plugins return content plus a declared content shape; the kernel's content-serving boundary sanitizes per shape, wraps in one kernel-owned document skeleton, and themes from one kernel-owned stylesheet (D-11)"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "kernel/httpapi/rendition_test.go#TestSanitizeAndWrapRendition_* (structural pipeline), TestEmailShape_*/TestMarkdownShape_*/TestChatShape_* (per-shape), plugins/proton/fetch_rendition_test.go#TestFetch_HTMLOnlyMessageKeepsTheRendition, plugins/silverbullet/fetch_test.go#TestFetch_FullVariant_AvailableTextHTMLWithText, plugins/signal/fetch_test.go#TestFetch_FullReturnsUnwrappedTranscriptFragment"
        status: pass
    human_judgment: false
  - id: D2
    description: "A rendition declaring text/html with no recognised content shape is refused, never served — the kernel fails closed"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "kernel/httpapi/rendition_test.go#TestSanitizeAndWrapRendition_UnrecognisedShapeReturnsErrorNoBytes, kernel/httpapi/item_test.go#TestItemContentHandler_UnrecognisedContentShapeRefusedNoBody"
        status: pass
    human_judgment: false
  - id: D3
    description: "No plugin in the repository emits a full HTML document, a document-wrapping helper, or a hardcoded rendition stylesheet after this plan (D-11)"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "grep -rl 'themeStyle|signalThemeStyle|WrapDocument' plugins --include='*.go' — zero matches, run and confirmed during execution"
        status: pass
    human_judgment: false
  - id: D4
    description: "The kernel-owned rendition stylesheet carries the shared base exactly once, with per-shape deltas layered on it, rather than three independently-authored stylesheets"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "kernel/httpapi/rendition.go — renditionBaseStyle plus renditionProseDelta/renditionEmailImageDelta/renditionMarkdownImageDelta/renditionEmailReadabilityDelta/renditionChatDelta, composed by stylesheetForShape; kernel/httpapi/rendition_test.go#TestRenditionStylesheetTokensMatchAppCSS"
        status: pass
    human_judgment: false
  - id: D5
    description: "The shared hex tokens the rendition stylesheet uses are the same ones web/src/app.css declares, proven by a test that reads app.css"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "kernel/httpapi/rendition_test.go#TestRenditionStylesheetTokensMatchAppCSS"
        status: pass
    human_judgment: false
  - id: D6
    description: "Post-D-11 kernel-owned wrapping renders visually identical to today's per-plugin output for all three content shapes — email, markdown, and chat, including the chat profile's no-accent-on-bubble/sender/timestamp rule"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "kernel/httpapi/rendition_test.go#TestChatShape_NoAccentColourOnBubbleSenderOrTimestamp, TestImagePolicy_EmailHidesMarkdownAllows, TestEmailShape_NeutralizesEmailSuppliedColours"
        status: pass
      human_judgment: true
      rationale: "The CSS token/rule assertions prove byte-for-byte carry-forward of every declared rule, but actual pixel-level visual parity in a real browser was not captured by a screenshot in this run — a human should confirm the rendered iframe still looks identical for a real email/markdown/chat item."
  - id: D7
    description: "The chat profile's tombstone, quote, attachment and reaction rules are carried forward verbatim into the kernel-owned stylesheet"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "kernel/httpapi/rendition_test.go#TestChatShape_TombstoneQuoteAttachmentReactionRulesPresent"
        status: pass
    human_judgment: false
  - id: D8
    description: "The thin, theme-matched scrollbar treatment is reproduced identically in the kernel-owned stylesheet, and the per-plugin render-test scrollbar assertions are relocated to the kernel's own test file rather than dropped"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "kernel/httpapi/rendition_test.go#TestSanitizeAndWrapRendition_InjectsThinThemeMatchedScrollbar"
        status: pass
    human_judgment: false
  - id: D9
    description: "The email profile hides images outright while the markdown profile allows them at full container width"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "kernel/httpapi/rendition_test.go#TestImagePolicy_EmailHidesMarkdownAllows, TestEmailShape_RemoteImagePreservedButHidden"
        status: pass
    human_judgment: false
  - id: D10
    description: "A crafted chat message body cannot forge transcript structure — the producing plugin escapes every interpolated text field, and the kernel's chat policy allows a class attribute only on div within a fixed token set (T-05-17)"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "plugins/signal/render_test.go#TestRenderTranscript_MarkupInMessageBodyRendersAsLiteralEscapedText, kernel/httpapi/rendition_test.go#TestChatShape_ForgedClassOutsideAllowlistIsStripped, TestChatShape_LegitimateStructuralClassesSurvive"
        status: pass
    human_judgment: false

duration: ~20min
completed: 2026-08-06
status: complete
---

# Phase 5 Plan 4: Source Instances & Per-Type Matching Summary

**Rendition presentation moved out of three plugins into one kernel-owned sanitize/wrap/theme boundary (D-11): plugins now declare a `toposv1.ContentShape` and return an unwrapped fragment; `kernel/httpapi/rendition.go`'s `sanitizeAndWrapRendition` is the single place that sanitizes, wraps and themes every text/html rendition, failing closed on an unrecognised shape.**

## Performance

- **Duration:** ~20 min (two commits, back to back)
- **Completed:** 2026-08-06
- **Tasks:** 2
- **Files modified:** 22 (2 created, 20 modified; 1 deleted — `plugins/proton/render_test.go`, fully relocated)

## Accomplishments

- `kernel/httpapi/rendition.go`: three `bluemonday.Policy` values (email, markdown, chat) built once at init, a shared CSS base plus per-shape delta constants (`renditionProseDelta`, `renditionEmailImageDelta`, `renditionMarkdownImageDelta`, `renditionEmailReadabilityDelta`, `renditionChatDelta`), and `sanitizeAndWrapRendition(shape, fragment) ([]byte, error)` — sanitize always runs before wrap, wrapped output is never re-sanitized, an unrecognised/unspecified shape returns an error and zero bytes
- `proto/topos/v1/plugin.proto`: `ContentShape` enum (`CONTENT_SHAPE_UNSPECIFIED = 0`, `..._EMAIL_HTML = 1`, `..._CHAT_TRANSCRIPT = 2`, `..._MARKDOWN_HTML = 3`) and `FetchResponse.content_shape = 8`, regenerated into `sdk/gen/topos/v1/plugin.pb.go`
- `sdk/contract_test.go`'s `TestContractEnumsZeroValueUnspecified` now covers `ContentShape` alongside `LinkFidelity`/`ContentVariant`
- `kernel/pluginhost.FetchResult.ContentShape` carries `FetchResponse.content_shape` into the kernel domain
- `kernel/httpapi/item.go`'s `renditionHandler` and `agent.go`'s `agentRenditionHandler` both branch on `MimeType == "text/html"`: read the fragment, call `sanitizeAndWrapRendition`, write the wrapped document, or refuse with `unsupported_content_shape` (502) and no body on error; every other MIME type keeps its existing `io.Copy` pass-through byte-for-byte
- `plugins/silverbullet`: `RenderSanitized` → `RenderMarkdown` (goldmark conversion only, no bluemonday, no wrap); `Fetch` returns the raw converted fragment with `ContentShape` = markdown
- `plugins/proton`: `body.go` reduced to MIME-part extraction only (`PlainTextPart`/`HTMLPart`/`HasRenderableText`/`Snippet`); `RenderSanitizedEmail`, the email policy, `themeStyle` and `WrapDocument` all deleted; `Fetch`'s HTML fallback returns the raw HTML part with `ContentShape` = email
- `plugins/signal`: `sanitizeText` → `escapeText` (`html.EscapeString`) — the plugin's job is now guaranteeing its own structural markup can't be forged by message content, while the kernel's chat policy performs the actual sanitization over the assembled fragment; `signalTranscriptSanitizePolicy`, `signalThemeStyle` and `WrapDocument` all deleted; `Fetch` returns the raw transcript fragment with `ContentShape` = chat
- `plugins/paperless` and `plugins/mock` checked directly: neither ever serves a `text/html` rendition, so both are unchanged — the zero `ContentShape` value is correct for their PDF/image renditions
- `.planning/todos/pending/2026-08-05-centralize-rendition-theming-in-kernel.md` moved to `.planning/todos/completed/`
- `make test` exits 0 across all six workspace modules (root/kernel, sdk, paperless, silverbullet, proton, mock, signal)

## Task Commits

1. **Task 1: The kernel-owned rendition module — policies, stylesheet, wrap** (tdd) — `d0948b4` (feat)
2. **Task 2: Atomic cutover — plugins return content, the kernel presents it** (tdd) — `fe244d2` (feat)

_Both tasks landed their tests and implementation in a single commit each, consistent with this repo's per-task commit granularity established in prior Phase 5 plans. No separate RED/GREEN commits — `<behavior>`-driven tests were written alongside their implementation within each task, then verified green before committing._

## Files Created/Modified

- `kernel/httpapi/rendition.go` (new) — sanitize/wrap/theme pipeline, three content-shape policies, composed stylesheet
- `kernel/httpapi/rendition_test.go` (new) — relocated + new assertions, `TestRenditionStylesheetTokensMatchAppCSS`
- `go.mod`, `go.sum` — `github.com/microcosm-cc/bluemonday v1.0.27` added to the root module
- `proto/topos/v1/plugin.proto`, `sdk/gen/topos/v1/plugin.pb.go` — `ContentShape` enum, `FetchResponse.content_shape`
- `sdk/contract_test.go` — `ContentShape` row in the zero-value-unspecified table
- `kernel/pluginhost/host.go` — `FetchResult.ContentShape`
- `kernel/httpapi/item.go`, `agent.go` — text/html branch through `sanitizeAndWrapRendition`
- `kernel/httpapi/item_test.go` — rewritten `TestItemContentHandler_TextHTMLRenditionServedWithSecurityHeaders` for the new wrapped-output behavior, new `TestItemContentHandler_UnrecognisedContentShapeRefusedNoBody`
- `plugins/proton/body.go` — MIME-part extraction only; `plugin.go` — `Fetch`'s HTML fallback declares `ContentShape`; `fetch_rendition_test.go` — asserts raw fragment, not a wrapped doc; `render_test.go` — deleted (relocated)
- `plugins/silverbullet/render.go` — `RenderMarkdown`; `plugin.go` — `Fetch` declares `ContentShape`; `render_test.go` — conversion-only assertions kept
- `plugins/signal/render.go` — `escapeText`; `plugin.go` — `Fetch` declares `ContentShape`; `render_test.go` — transcript-structure assertions kept plus a new escaping regression test; `fetch_test.go` — asserts unwrapped fragment plus `ContentShape`

## Decisions Made

- Proto/sdk regeneration landed in Task 1's commit (deviation Rule 3, blocking compile dependency) rather than Task 2's, since `rendition.go`'s policy map is keyed by `toposv1.ContentShape` and Task 1 could not build otherwise. Task 2 still owns the FetchResult/item.go/agent.go/plugin wiring and the contract test's zero-value row, matching its declared scope.
- The email content-shape stylesheet carries forward proton's full readability layer (the `body, body *` `!important` neutralizer) even though 05-UI-SPEC.md's Rendition Content Contract table doesn't spell it out verbatim — the plan's own must_haves require post-move visual parity, and Task 1's action text explicitly named relocating "the one proving an email cannot mark a declaration important" as required coverage.
- bluemonday's class-attribute `Matching` regexp was verified against the vendored library source (not assumed from docs) to match a class attribute's ENTIRE value, not per space-separated token — `chatTranscriptClassTokens` is written accordingly to accept any sequence of the fixed transcript token set, with an out-of-set value dropping the whole attribute rather than being filtered token-by-token.
- `plugins/signal/fetch_test.go` and `plugins/proton/fetch_rendition_test.go` updated beyond this plan's declared `files_modified` list (Rule 3): both previously asserted a fully wrapped document (doctype prefix, embedded theme tokens) in the plugin's own `Fetch` response — an assertion the cutover deliberately makes false. Both now assert the raw fragment plus the declared `ContentShape`.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Proto/sdk regeneration moved into Task 1's commit**
- **Found during:** Task 1 (writing `kernel/httpapi/rendition.go`)
- **Issue:** The plan's Task 1 action text specifies `renditionPolicies` as `map[toposv1.ContentShape]*bluemonday.Policy`, but `ContentShape` didn't exist yet — Task 2's action text is the one that adds it to `proto/topos/v1/plugin.proto`. Task 1 could not build without it.
- **Fix:** Added the `ContentShape` enum and `FetchResponse.content_shape` field to the proto in Task 1, regenerated `sdk/gen/topos/v1/plugin.pb.go` via `make proto` (using `buf`, found at `/opt/go/bin` once added to `PATH`)
- **Files modified:** `proto/topos/v1/plugin.proto`, `sdk/gen/topos/v1/plugin.pb.go` (both also legitimately touched by Task 2's own declared scope for the contract-test row and doc comments)
- **Verification:** `CGO_ENABLED=0 go build ./... && go test ./kernel/httpapi/ -run Rendition -count=1 -v` passed
- **Committed in:** `d0948b4` (Task 1 commit)

**2. [Rule 3 - Blocking] `go mod tidy` avoided in favor of a targeted `go get`**
- **Found during:** Adding the bluemonday dependency to the root module
- **Issue:** Running `go mod tidy` in this Go-workspace repo pulled in unrelated buf/protoreflect transitive packages and added a synthetic pseudo-versioned `require` on the workspace-local `github.com/davison/topos/sdk` module — exactly the failure mode `plugins/proton/go.mod`'s own comment already documents ("`go mod tidy` cannot be run cleanly against this module in isolation because `github.com/davison/topos/sdk` has no published remote")
- **Fix:** Reverted `go.mod`/`go.sum`, then used `go get github.com/microcosm-cc/bluemonday@v1.0.27` alone (no `tidy`), which added exactly the one new dependency and its two transitive deps (`aymerick/douceur`, `gorilla/css`) with no unrelated churn
- **Files modified:** `go.mod`, `go.sum`
- **Verification:** `CGO_ENABLED=0 go build ./...` passed; `git diff go.mod` showed only the intended additions
- **Committed in:** `d0948b4` (Task 1 commit)

**3. [Rule 3 - Blocking] `plugins/signal/fetch_test.go` and `plugins/proton/fetch_rendition_test.go` needed updates outside the declared file list**
- **Found during:** Task 2's own `<verify>` (`make test`)
- **Issue:** Both test files asserted the plugin's `Fetch` response carried a complete wrapped HTML document (doctype prefix, embedded theme hex tokens) — the exact behavior the cutover removes
- **Fix:** Rewrote the relevant assertions to check for the RAW, unwrapped fragment (no doctype) plus the newly-declared `ContentShape` field
- **Files modified:** `plugins/signal/fetch_test.go`, `plugins/proton/fetch_rendition_test.go`
- **Verification:** `make test` exits 0 across all six workspace modules
- **Committed in:** `fe244d2` (Task 2 commit)

**4. [Rule 1 - Bug] Stray literal "themeStyle"/"WrapDocument" strings in a relocated doc comment**
- **Found during:** Running Task 2's own acceptance-criteria greps after the cutover
- **Issue:** `plugins/silverbullet/render_test.go`'s new file-level doc comment referenced "the WrapDocument/themeStyle assertions" in prose, which made `grep -rl 'themeStyle' plugins --include='*.go'` return non-empty even though no code referenced either symbol
- **Fix:** Reworded the comment to describe the same thing without using the literal retired identifier names
- **Files modified:** `plugins/silverbullet/render_test.go`
- **Verification:** `grep -rl 'themeStyle' plugins --include='*.go'` and `grep -rl 'WrapDocument' plugins --include='*.go'` both produce no output
- **Committed in:** `fe244d2` (Task 2 commit)

---

**Total deviations:** 4 auto-fixed (3 Rule 3 blocking, 1 Rule 1 bug)
**Impact on plan:** All four were mechanical fixes required for the plan's own stated `<verify>`/acceptance criteria to actually pass; no scope creep beyond what those requirements already implied.

## Issues Encountered

None beyond the four deviations above.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Plan 05-05 (docs/plugin-contract.md, config.example.toml republishing) can build on this plan's shipped `ContentShape` contract shape without further rendition-boundary rework.
- D6's visual-parity coverage entry is marked `human_judgment: true` — a human should open a real email, markdown page, and Signal chat item through the detail pane and confirm the rendered output still looks identical to before this move, since the CSS-token assertions prove rule carry-forward but not actual pixel rendering.
- Every currently-configured real webspace in this repo (none checked into version control) that serves a `text/html` rendition continues to work unchanged in shape — only the wrapping/sanitizing location moved, not the visible output.

---
*Phase: 05-source-instances-per-type-matching*
*Completed: 2026-08-06*
