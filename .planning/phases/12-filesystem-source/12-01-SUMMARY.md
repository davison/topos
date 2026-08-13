---
phase: 12-filesystem-source
plan: 01
subsystem: source-plugin
tags: [go-plugin, grpc, filesystem, xdg-open, svelte5, deep-link, e2e]

requires:
  - phase: 11-external-plugins-the-trust-boundary
    provides: two-tier plugin discovery/launch, extras machinery, e2e fixture harness the filesystem tracer's e2e spec reuses unmodified

provides:
  - "plugins/filesystem — a real, minimal-but-production-quality go-plugin source: Describe/Match/Fetch/Health over a configured local folder, top-level *.pdf files only"
  - "The plugin->kernel deep-link signalling convention (file:// scheme + kernel-side scheme-keyed rewrite, Task 1 checkpoint option-a) — a published-contract-level pattern every later local-path plugin (incl. Phase 14's out-of-repo Google Drive plugin) can now read and build against"
  - "kernel/httpapi/fsopen.go — POST /api/items/{id}/open, the kernel's second raw-exec HTTP surface (after whatsapplink.go), resolving its target exclusively from index state + configuration"
  - "kernel/httpapi/stream.go's resolveStreamLinkURL — the file://-scheme serve-time rewrite, source_type-agnostic by construction"
  - "OpenInSource.svelte's local-exec branch (12-UI-SPEC.md F2) — the first non-anchor-navigation Open-in-Source interaction in the app"

affects: [12-02, 12-03, 12-04, 12-05, 14-google-drive-plugin]

actuals:
  tokens: 18700
  tasks: 1
  commits: 1

tech-stack:
  added: []
  patterns:
    - "Local-path source plugin (Path-only sourceConfig, no base_url/token) — plugins/signal's shape, now a second precedent"
    - "Kernel-mediated exec surface resolved exclusively from index state + config, never the request — whatsapplink.go's discipline, now a second precedent (fsopen.go)"
    - "Serve-time deep-link rewrite keyed on URL scheme, never source_type — the reusable convention for any future local-path plugin"
    - "Svelte component behavior tests via comment-stripped source scanning (this repo's house convention; no jsdom/@testing-library/svelte harness exists)"

key-files:
  created:
    - plugins/filesystem/go.mod
    - plugins/filesystem/main.go
    - plugins/filesystem/plugin.go
    - plugins/filesystem/item.go
    - plugins/filesystem/item_test.go
    - plugins/filesystem/assets/icon.svg
    - kernel/httpapi/fsopen.go
    - kernel/httpapi/fsopen_test.go
    - web/src/lib/components/open-in-source-local-exec.test.ts
    - web/e2e/specs/12-filesystem-tracer.spec.ts
  modified:
    - go.work
    - Makefile
    - kernel/httpapi/stream.go
    - kernel/httpapi/stream_test.go
    - kernel/httpapi/routes.go
    - kernel/httpapi/config_test.go
    - kernel/httpapi/contract_test.go
    - web/src/lib/components/OpenInSource.svelte
    - web/e2e/e2e-builtins.d.ts

key-decisions:
  - "Task 1 checkpoint resolved as option-a (user-selected): the filesystem plugin emits an honest file:// deep_link built from its own root + relative source_id; the kernel rewrites it to /api/items/{id}/open at serve time, keyed on the file:// scheme alone, never source_type. No contract version bump."

patterns-established:
  - "Serve-time deep-link scheme rewrite (kernel/httpapi/stream.go's resolveStreamLinkURL): any future local-path plugin gets kernel-mediated-open treatment for free by emitting a file:// deep_link — zero kernel code change required per plugin type."
  - "OpenInSource.svelte's isLocalExecLink branch: any future link.url shaped as a same-origin /api/ path gets the fetch-based local-exec interaction automatically — zero per-plugin-type frontend branching."

requirements-completed: [SRC-04]

coverage:
  - id: D1
    description: "A folder configured as a filesystem source produces one stream item per top-level *.pdf file, with a D-01 relative-path source_id, LINK_FIDELITY_EXACT, an empty preview, and a file:// deep_link — no file body read at Match time"
    requirement: SRC-04
    verification:
      - kind: unit
        ref: "plugins/filesystem/item_test.go#TestMatch_TopLevelPDFYieldsExactFidelityAndEmptyPreview"
        status: pass
      - kind: unit
        ref: "plugins/filesystem/item_test.go#TestRelPathSourceID_SubdirectoryFileYieldsForwardSlashRelativePath"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/12-filesystem-tracer.spec.ts — 'the stream carries exactly one item, served through the kernel open route, with a PDF rendition'"
        status: pass
    human_judgment: false

  - id: D2
    description: "The kernel rewrites a file://-scheme deep_link to the loopback open route at serve time, keyed on the URL scheme alone — every other item's deep_link is echoed unchanged"
    requirement: SRC-04
    verification:
      - kind: unit
        ref: "kernel/httpapi/stream_test.go#TestStreamLink_FileSchemeDeepLinkIsRewrittenToTheOpenRoute"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/stream_test.go#TestStreamLink_NonFileSchemeDeepLinkIsEchoedUnchanged"
        status: pass
      - kind: integration
        ref: "kernel/httpapi/stream_test.go#TestStreamHandler_FileSchemeItemServesRewrittenLinkEndToEnd"
        status: pass
    human_judgment: false

  - id: D3
    description: "POST /api/items/{id}/open resolves the exec target exclusively from the indexed item's source_id + the source's configured Path, expands a leading ~, refuses a path escape (invalid_path), refuses a non-file:// item and an unknown item (item_not_found), and surfaces an opener failure as open_failed with the opener's own message"
    requirement: SRC-04
    verification:
      - kind: unit
        ref: "kernel/httpapi/fsopen_test.go#TestFilesystemOpen_HappyPathOpensTheJoinedAbsolutePath"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/fsopen_test.go#TestFilesystemOpen_TildeInConfiguredPathIsExpandedBeforeTheJoin"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/fsopen_test.go#TestFilesystemOpen_PathEscapeAnswersInvalidPath"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/fsopen_test.go#TestFilesystemOpen_UnknownItemAnswersItemNotFound"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/fsopen_test.go#TestFilesystemOpen_NonFileSchemeDeepLinkAnswersItemNotFound"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/fsopen_test.go#TestFilesystemOpen_OpenerErrorAnswersOpenFailed"
        status: pass
    human_judgment: false

  - id: D4
    description: "OpenInSource renders a real <button> (not <a>) for a same-origin /api/ link.url in both presentations, issues exactly one POST on click, changes nothing visible on success, swaps to the destructive 'Couldn't open' label/title for 2.5s on failure (kernel detail verbatim, or the fixed fallback), and is never given a disabled attribute"
    requirement: SRC-04
    verification:
      - kind: unit
        ref: "web/src/lib/components/open-in-source-local-exec.test.ts (22 assertions covering branch selection, the single POST, the failure swap/revert, title/aria-label sourcing, and the absence of disabled)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/12-filesystem-tracer.spec.ts — 'selecting the row renders an Open control that is a button, not an anchor'"
        status: pass
    human_judgment: false

  - id: D5
    description: "The whole path proven against a real kernel and a real topos-plugin-filesystem binary: one PDF in a configured folder becomes a stream item whose Open control POSTs to /api/items/{id}/open"
    requirement: SRC-04
    verification:
      - kind: e2e
        ref: "web/e2e/specs/12-filesystem-tracer.spec.ts (both tests, chromium project, make e2e)"
        status: pass
    human_judgment: false

  - id: D6
    description: "The 'Couldn't open' swap string never overflows a narrow layout where the at-rest label did not (mobile takeover combination) — 12-UI-SPEC.md's own documented backstop, unverified by any visual check"
    verification: []
    human_judgment: true
    rationale: "12-UI-SPEC.md itself scopes this as a backstop truth ('unverified, no visual check has been run against the mobile takeover combination') rather than an explicit one — carried forward unresolved, not newly introduced by this plan's execution."

duration: ~50min
completed: 2026-08-13
status: complete
---

# Phase 12 Plan 01: Filesystem Source Tracer Summary

**A single PDF in a configured folder now flows end to end — filesystem plugin → kernel stream → kernel-mediated `xdg-open` route → a real `<button>` in the browser — proving the `file://`-scheme deep-link convention and the loopback open route the rest of Phase 12 builds on.**

## Performance

- **Duration:** ~50 min (continuation from Task 2 onward; Task 1 was the already-resolved decision checkpoint)
- **Tasks:** 1 (Task 2 — the tracer; Task 1 was a `checkpoint:decision` resolved by the user before this continuation started, with nothing to commit for it)
- **Files modified:** 20 (11 created, 9 modified)

## Accomplishments

- New `plugins/filesystem` go-plugin module: `Describe`/`Match`/`Fetch`/`Health` over a configured local folder — top-level `*.pdf` files only for this tracer, no recursion, no extras, no file body read at `Match` time.
- Locked and implemented the Task 1 checkpoint's mechanism (option-a): the plugin emits an honest `file://` deep link; `kernel/httpapi/stream.go`'s new `resolveStreamLinkURL` rewrites it to `/api/items/{id}/open` at serve time, keyed on the URL scheme alone — never `source_type` — so a future third-party local-path plugin gets identical treatment with zero kernel change.
- New `kernel/httpapi/fsopen.go`: `POST /api/items/{id}/open`, the kernel's second raw-exec HTTP surface, resolving the `xdg-open` target exclusively from the indexed item's own `source_id` plus its source's configured `Path` — refusing a `../`-style escape with `invalid_path` and a non-filesystem item with `item_not_found`, never trusting anything from the request itself.
- `OpenInSource.svelte` gained a local-exec branch (12-UI-SPEC.md F2): a same-origin `/api/` `link.url` now renders a real `<button>` that POSTs instead of navigating, with the 2.5s "Couldn't open" destructive-tone swap-then-revert on failure.
- Hermetic end-to-end proof (`web/e2e/specs/12-filesystem-tracer.spec.ts`) against a real booted kernel and a real `topos-plugin-filesystem` binary — no assertion on `xdg-open`'s own behavior (that's the stub-opener Go tests' job).

## Task Commits

Task 1 (`checkpoint:decision`) had no implementation work — nothing to commit; its outcome (option-a) is recorded above and implemented by Task 2's commit.

1. **Task 2: End-to-end "a PDF in a folder is in my stream and opens in my desktop"** — `afbc679` (feat)

**Plan metadata:** this SUMMARY's own commit (pending, see below)

## Files Created/Modified

- `plugins/filesystem/go.mod`, `go.sum`, `main.go`, `plugin.go`, `item.go`, `item_test.go`, `assets/icon.svg` — the new plugin module
- `go.work` — added `./plugins/filesystem` to the workspace `use` block
- `Makefile` — `topos-plugin-filesystem` added to `plugins`, `plugins-portable`, `test-portable`, and the `e2e` fixture-binary block
- `kernel/httpapi/fsopen.go`, `fsopen_test.go` — the new loopback open route + its 6-case test suite (happy path, `~` expansion, path escape, unknown item, non-`file://` item, opener error)
- `kernel/httpapi/stream.go`, `stream_test.go` — `resolveStreamLinkURL` + 3 new tests
- `kernel/httpapi/routes.go` — `POST /api/items/{id}/open` registered on `/api` only
- `kernel/httpapi/config_test.go`, `contract_test.go` — the two non-GET-route allowlist guards updated for the new intentional mutating route (deviation, see below)
- `web/src/lib/components/OpenInSource.svelte` — the local-exec branch (F2)
- `web/src/lib/components/open-in-source-local-exec.test.ts` — 22 source-scan assertions covering the new branch's full behavior contract
- `web/e2e/e2e-builtins.d.ts` — added the `basename` ambient declaration the new spec's corpus-directory-label derivation needs
- `web/e2e/specs/12-filesystem-tracer.spec.ts` — the hermetic end-to-end spec

## Decisions Made

- **Task 1 checkpoint outcome (option-a, user-selected):** the filesystem plugin never learns the kernel's own base URL or its own source instance id (both deliberately withheld by the contract) — it builds a `file://` URI over the real absolute path it already knows (its configured root + the relative `source_id`), and the kernel rewrites that to the loopback open route at serve time, keyed on the URL *scheme*, never `source_type`. No contract version bump, no new launch-environment field. This is now the convention Phase 14's out-of-repo Google Drive plugin (and any future local-path plugin) reads and builds against.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Updated the two non-GET-route allowlist guards to include the new intentional mutating route**
- **Found during:** Task 2, running `go test ./kernel/httpapi/...` after adding `POST /api/items/{id}/open`
- **Issue:** `TestRoutesGuard_NonGetRoutesScopedToConfig` (config_test.go) and `TestContract_MutatingRoutesAreConfigScoped` (contract_test.go) are AST-level guards that fail the build on ANY unreviewed non-GET route addition to `routes.go` — both failed as designed the moment the new route landed.
- **Fix:** Added `{"Post", "/api/items/{id}/open"}` to both guards' explicit allowlists, with a comment naming this plan's threat-model rows (T-12-01 through T-12-06) as the route's own review trail — exactly the "deliberate, reviewed decision" both guards' own failure messages ask for.
- **Files modified:** `kernel/httpapi/config_test.go`, `kernel/httpapi/contract_test.go`
- **Verification:** `go test ./kernel/httpapi/...` passes in full (0 failures) after the update.
- **Committed in:** `afbc679` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (Rule 1 — a build-time guard test correctly caught the new intentional mutating route and needed its allowlist updated; not a bug in the new code itself)
**Impact on plan:** Necessary maintenance of an existing guard test, not scope creep — the guard's own design intent (force a reviewed decision on any new mutating route) was honored, not bypassed.

## Issues Encountered

- `npm ci` had not been run in this worktree (no `node_modules`) — required before any `npm run test`/`check`/`e2e` step could execute; resolved by running it once at the start of frontend verification.
- `npm run build` inside `make e2e` overwrote the gitignored `kernel/webui/build/.gitkeep` placeholder with real build output as a side effect — restored via `git checkout -- kernel/webui/build/.gitkeep` before committing, so no build artifact was accidentally staged.
- A stray `plugins/filesystem/filesystem` binary was produced by an earlier `go build ./...` invocation with no `-o` flag inside the plugin's own directory — deleted before committing.

## User Setup Required

None — no external service configuration required. `xdg-open` presence on the target desktop is assumed (per 12-RESEARCH.md's Environment Availability table) and was not probed this session; the open route's own tests are stubbed against an `Opener` seam specifically so this gate never depends on `xdg-open` being installed on the CI runner.

## Next Phase Readiness

- The `file://`-scheme deep-link convention and the kernel-mediated open route are now real, tested machinery — 12-02 through 12-05 (extension allowlist, extras-driven scope, subfolder recursion, markdown/plain-text/office preview shapes, symlink/hidden-file policy, the connection-form UI, docs, and the external-tier rehearsal) can build directly on top without touching this plan's own files again.
- `kernel/httpapi/item.go`'s `allowedRenditionTypes` map still lacks a `text/plain` entry (12-RESEARCH.md Pitfall 1) — out of this tracer's PDF-only scope; a later plan in this phase must add it before the plain-text preview path can work.
- The `bmatcuk/doublestar/v4` glob dependency 12-RESEARCH.md flags for the extras-driven scope-widening feature was not needed by this tracer at all (PDF-only, no globs) — its `checkpoint:human-verify` legitimacy gate remains a later plan's task, not this one's.
- D6 (the narrow-layout overflow backstop on the "Couldn't open" swap string) remains genuinely unverified — inherited from 12-UI-SPEC.md's own documented backstop, not a gap this plan introduced or was asked to close.

---
*Phase: 12-filesystem-source*
*Completed: 2026-08-13*
