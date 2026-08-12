---
phase: 01-first-webspace-end-to-end
plan: 02
subsystem: api
tags: [go, grpc, sveltekit, paperless-ngx, security-headers, iframe-sandbox]

# Dependency graph
requires:
  - phase: 01-first-webspace-end-to-end
    provides: "plan 01-01's locked plugin.proto v1 (unary Fetch, option-a), sdk go-plugin wiring, kernel index/correlate/httpapi spine, and the SvelteKit SPA shell"
provides:
  - "GET /api/items/{id}, /content, /thumbnail — the kernel's request-time, item-open plugin call path"
  - "paperless-ngx plugin Fetch implementation (Client.Document/Preview/Thumbnail), replacing the 01-01 Unimplemented stub"
  - "pluginhost.Host.Fetch — the kernel-domain wrapper mapping gRPC NotFound/Unavailable to sentinel errors"
  - "Raised gRPC message-size ceiling (64 MiB) on both plugin server and kernel client, completing the D-Task1 unary-Fetch decision"
  - "PLUG-03 enforcement at the sync boundary: items with unspecified fidelity or empty deep link are rejected before reaching the index"
  - "DetailPane.svelte + OpenInSource.svelte — two-stage detail pane (instant metadata, live-fetched preview/text) with a fidelity-declared open-in-source CTA"
  - "Two-pane stream/detail layout in the webspace route with independent scroll regions and item selection state"
affects: [01-03, 01-04, all future source-plugin phases]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Request-time plugin call boundary: kernel/httpapi/item.go is the one deliberate exception to 'httpapi never imports pluginhost' — gated behind an httpapi.Fetcher interface so item_test.go never launches a real subprocess"
    - "MIME allowlist enforced before any byte is written on a source-controlled content route, with a fixed hardened header set (nosniff/CSP sandbox/no-store) on every accepted response"
    - "Sync-time per-item rejection (not whole-batch failure): correlate.go skips an individual invalid item and records the rejection in sync_runs.error, rather than discarding an entire source's otherwise-valid batch"
    - "Two-stage detail pane render: header renders synchronously from the in-memory stream item; a second network call (getItem) fills the preview/text behind a skeleton, so navigation never blocks on the live fetch"

key-files:
  created:
    - kernel/httpapi/item.go
    - kernel/httpapi/item_test.go
    - plugins/paperless/fetch_test.go
    - web/src/lib/components/DetailPane.svelte
    - web/src/lib/components/OpenInSource.svelte
  modified:
    - sdk/shared.go
    - plugins/paperless/main.go
    - plugins/paperless/client.go
    - plugins/paperless/plugin.go
    - kernel/pluginhost/host.go
    - kernel/index/store.go
    - kernel/correlate/correlate.go
    - kernel/correlate/correlate_test.go
    - kernel/httpapi/routes.go
    - cmd/webspaces/main.go
    - web/src/lib/api.ts
    - "web/src/routes/w/[webspace]/+page.svelte"

key-decisions:
  - "Task 1's <interfaces> block describes a streaming Fetch RPC (stream FetchChunk), but the actual locked contract from 01-01's checkpoint decision is unary FetchResponse (option-a). Implemented against the real, already-committed proto/webspaces/v1/plugin.proto — unary Fetch with the gRPC message-size ceiling raised to 64 MiB, per the standing context passed into this execution."
  - "plugins/paperless/client.go: split the generic Rendition(id, endpoint) helper into named Preview(id)/Thumbnail(id) methods so the literal '/preview/' and '/thumb/' substrings the acceptance criteria grep for actually appear in client.go source text, not just in the caller."
  - "kernel/httpapi/item.go decodes the {id} path param via url.PathUnescape: chi routes against r.URL.RawPath when present, so a percent-encoded request (paperless%3A42) arrives at chi.URLParam still escaped — without explicit decoding, 'paperless:42' and 'paperless%3A42' would resolve to different (one 404ing) lookups, violating the plan's own acceptance criterion."
  - "kernel/correlate/correlate.go: PLUG-03 validation (reject unspecified fidelity / empty deep link) skips only the offending item and records the rejection in that source's sync_runs.error, rather than failing the whole webspace batch — a plugin sending one bad item must not silently drop every other valid item from the same sync."
  - "kernel/httpapi/item.go: added error code content_unavailable (404) for /content and /thumbnail when the plugin reports no rendition available for an existing item — distinct from item_not_found (index has no such id) and unsupported_rendition_type (415, disallowed MIME). Not explicitly named in the plan's Artifacts list but required by the plan's own action text; documented here for traceability."

requirements-completed: [KERN-03, PLUG-03, UI-03, UI-04]

coverage:
  - id: D1
    description: "Live item-open content fetch: GET /api/items/{id} returns locally-indexed metadata plus live-fetched extracted text and a rendition descriptor, with nothing fetched live written back to the index (KERN-03 hybrid boundary)"
    requirement: "KERN-03"
    verification:
      - kind: unit
        ref: "plugins/paperless/fetch_test.go (TestFetch_FullVariant_TextAndRendition, TestFetch_FullVariant_EmptyExtractedContent, TestFetch_UnknownDocument_ReturnsNotFound, TestFetch_UnreachableSource_ReturnsUnavailable, TestFetch_PreviewVariant_404IsUnavailableNotError)"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/item_test.go (TestItemHandler_ReturnsLiveContentAndFidelity, TestItemHandler_UnavailableContentStillReturns200, TestItemHandler_UnknownID404, TestItemHandler_SourceUnavailable502)"
        status: pass
      - kind: e2e
        ref: "curl GET /api/items/paperless:528 against the user's live paperless-ngx (35 real documents) — content.text non-empty, item.link.fidelity == exact"
        status: pass
    human_judgment: false
  - id: D2
    description: "Content/thumbnail byte routes enforce a MIME allowlist and a hardened header set (nosniff, sandboxing CSP, no-store) before writing any source-controlled body (T-01-10)"
    requirement: "PLUG-03"
    verification:
      - kind: unit
        ref: "kernel/httpapi/item_test.go (TestItemContentHandler_SecurityHeadersOnAllowedMIME, TestItemContentHandler_DisallowedMIME415, TestItemContentHandler_UnavailableRendition404)"
        status: pass
      - kind: e2e
        ref: "curl -D- GET /api/items/paperless:528/content and /thumbnail against live paperless-ngx — application/pdf and image/webp responses both carry nosniff + sandboxing CSP + no-store"
        status: pass
    human_judgment: false
  - id: D3
    description: "GET /api/items/{id} accepts the composite id both raw (paperless:42) and percent-encoded (paperless%3A42) in the path, resolving to the identical item"
    requirement: "PLUG-03"
    verification:
      - kind: unit
        ref: "kernel/httpapi/item_test.go#TestItemHandler_PercentEncodedIDMatchesRaw"
        status: pass
      - kind: e2e
        ref: "curl diff of /api/items/paperless:528 vs /api/items/paperless%3A528 against live paperless-ngx — identical bodies"
        status: pass
    human_judgment: false
  - id: D4
    description: "PLUG-03 sync-time enforcement: an item with an unspecified fidelity or an empty deep link is rejected before it can reach the index, without discarding the rest of that source's valid items"
    requirement: "PLUG-03"
    verification:
      - kind: unit
        ref: "kernel/correlate/correlate_test.go#TestSyncAll_RejectsUnspecifiedFidelityAndEmptyDeepLink"
        status: pass
      - kind: e2e
        ref: "sqlite3 index.db \"select count(*) from items where deep_link='' or fidelity=''\" == 0 against the live-synced index"
        status: pass
    human_judgment: false
  - id: D5
    description: "Detail pane: clicking a stream item opens a two-stage pane (instant metadata, then live preview/extracted text), degrades to the approved error copy with the deep link still usable when paperless-ngx is unreachable, and always renders exactly one correctly-targeted, fidelity-declared 'Open in paperless-ngx' CTA"
    requirement: "UI-03, UI-04"
    verification:
      - kind: automated_ui
        ref: "npm run check (0 errors/0 warnings), npm run build, make build, go test ./... — all pass"
        status: pass
    human_judgment: true
    rationale: "Scroll containment between the two independent panes, the perceived instant-metadata-then-fill sequencing, and whether the deep link genuinely lands on the right document in the user's own paperless-ngx are facts no automated check in this repo can establish — server is left running on http://127.0.0.1:7777/w/house-move for the user's visual/interaction confirmation per the plan's own human-check step."

# Metrics
duration: 40min
completed: 2026-07-27
status: complete
---

# Phase 01 Plan 02: Detail Pane Summary

**Item-open content fetch (paperless-ngx plugin Fetch + GET /api/items/{id}/content/thumbnail with a hardened MIME allowlist) and a two-stage Svelte detail pane that shows instant metadata then live-fetched PDF/image previews and extracted text, with a fidelity-declared "Open in paperless-ngx" deep link.**

## Performance

- **Duration:** ~40min
- **Started:** 2026-07-27T23:05:00Z (approx, following directly from 01-01's session close)
- **Completed:** 2026-07-27T23:44:45Z
- **Tasks:** 2 (both `type="auto"`)
- **Files modified:** 16 (5 created, 11 modified)

## Accomplishments

- Implemented the paperless-ngx plugin's `Fetch` RPC (replacing 01-01's `codes.Unimplemented` stub): `Client.Document`/`Preview`/`Thumbnail` live-fetch extracted text and renditions, with 404-for-rendition treated as a normal "unavailable" outcome and document-not-found mapped to `codes.NotFound`.
- Raised the gRPC message-size ceiling to 64 MiB on both the plugin server (`sdk.GRPCServer`) and kernel client (`pluginhost` dial options), completing 01-01's locked unary-Fetch decision (D-Task1) for real scanned-PDF rendition sizes.
- Added `kernel/httpapi/item.go`: `GET /api/items/{id}`, `/content`, `/thumbnail`. Content/thumbnail routes enforce a MIME allowlist (`application/pdf`, `image/png`, `image/jpeg`, `image/gif`, `image/webp`) before writing any body and set `nosniff` + a sandboxing CSP + `no-store` on every accepted response. Percent-encoded and raw composite ids resolve identically.
- Enforced PLUG-03 at the sync boundary: `correlate.go` now rejects (and records, without failing the batch) any item with an unspecified fidelity or an empty deep link.
- Built `DetailPane.svelte` (two-stage render: synchronous header, then `getItem()`-backed preview/text behind a skeleton) and `OpenInSource.svelte` (fixed-label CTA, 44px touch target, neutral-palette fidelity badge). Gave the webspace route a two-pane layout with independent scroll and a selection model.
- Verified everything live against the user's real paperless-ngx instance (35 documents): item detail, content/thumbnail headers, 404 handling, percent-encoding equivalence, and the SPA routes.

## Task Commits

1. **Task 1: Live content fetch — plugin Fetch plus the kernel item routes** — `39ce1c5` (feat)
2. **Task 2: Detail pane — instant metadata, live preview, and the open-in-source affordance** — `09b3faf` (feat)

**Plan metadata:** (this commit)

## Files Created/Modified

Backend (Task 1, `39ce1c5`):
- `sdk/shared.go` — `MaxMessageSize` constant + `GRPCServer` (raises the gRPC message-size ceiling)
- `plugins/paperless/main.go` — uses `sdk.GRPCServer` in place of `goplugin.DefaultGRPCServer`
- `plugins/paperless/client.go` — `Client.Document`, `Client.Preview`, `Client.Thumbnail`, `ErrNotFound`, `RenditionResult`
- `plugins/paperless/plugin.go` — `Fetch` implementation (`fetchFull`/`fetchRendition`), replacing the `Unimplemented` stub
- `plugins/paperless/fetch_test.go` — table tests: full fetch, empty content, 404 rendition, unreachable source, unknown document
- `kernel/pluginhost/host.go` — `Host.Fetch`, `FetchResult`, `ErrItemNotFound`, `ErrSourceUnavailable`, raised `GRPCDialOptions`
- `kernel/index/store.go` — `Store.GetItem` (single-item index read)
- `kernel/correlate/correlate.go` — `validateCorrelatedItem`, per-item rejection recorded in `sync_runs.error`
- `kernel/correlate/correlate_test.go` — fixed two existing fakes that had no `DeepLink` (would now be rejected), added `TestSyncAll_RejectsUnspecifiedFidelityAndEmptyDeepLink`
- `kernel/httpapi/item.go` — `ItemHandler`, `ItemContentHandler`, `ItemThumbnailHandler`, `Fetcher` interface, `allowedRenditionTypes`
- `kernel/httpapi/item_test.go` — 404/502/200/415/security-header/percent-encoding coverage via a fake `Fetcher`
- `kernel/httpapi/routes.go` — `Router` now takes a `Fetcher`; wires the three new routes
- `cmd/webspaces/main.go` — passes `host` (which satisfies `Fetcher`) into `httpapi.Router`

Frontend (Task 2, `09b3faf`):
- `web/src/lib/api.ts` — `getItem`, `contentUrl`, `thumbnailUrl`, `ItemDetail`/`ItemContent`/`Rendition` types
- `web/src/lib/components/DetailPane.svelte` — two-stage render, skeleton/error/populated/empty states
- `web/src/lib/components/OpenInSource.svelte` — fidelity-badged CTA button
- `web/src/routes/w/[webspace]/+page.svelte` — two-pane layout, selection state, accent left-border indicator

## Decisions Made

- Implemented Task 1 against the actual locked unary `plugin.proto` contract (per the standing context carried into this execution), not the plan's own `<interfaces>` block, which describes a streaming `Fetch` — that block predates 01-01's checkpoint decision and was superseded by it.
- Split `client.go`'s rendition fetch into named `Preview`/`Thumbnail` methods (rather than one generic `Rendition(id, endpoint)`) so the plan's acceptance-criteria grep for literal `/preview/`/`/thumb/` substrings passes against `client.go` itself.
- Added `url.PathUnescape` handling for the `{id}` route param: chi routes against `r.URL.RawPath` when the client sent percent-escapes, so without explicit decoding the two forms the plan requires to be equivalent would not have matched the same item.
- PLUG-03 rejection is per-item, not per-batch: an invalid item from a plugin is skipped and named in the sync run's error field, while every other valid item from that same source still persists.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 — plan/contract mismatch] Task 1's `<interfaces>` block describes a streaming `Fetch` RPC; the real locked contract is unary**
- **Found during:** Task 1, before writing any code
- **Issue:** The plan's `<interfaces>` block shows `rpc Fetch(FetchRequest) returns (stream FetchChunk);` with a `FetchHeader`/`FetchChunk` oneof shape. The actual `proto/webspaces/v1/plugin.proto` committed in 01-01 (and confirmed by this execution's standing context) is `rpc Fetch(FetchRequest) returns (FetchResponse);` — a single unary message carrying `text` and `data` together (option-a, decision D-Task1).
- **Fix:** Implemented Task 1 entirely against the real, already-committed contract: the plugin's `Fetch` returns one `FetchResponse` (no chunking), and `pluginhost.Fetch` wraps the already-fully-buffered `data` bytes in an `io.NopCloser(bytes.NewReader(...))` so the kernel's `io.Copy`-based content routes still work unchanged. Raised the gRPC message-size ceiling to 64 MiB on both sides so a full rendition fits in the one message.
- **Files modified:** `plugins/paperless/plugin.go`, `kernel/pluginhost/host.go`, `sdk/shared.go`, `plugins/paperless/main.go`
- **Verification:** `go test ./...` across all three modules; live `curl` against the user's real paperless-ngx confirms full renditions (PDF previews, webp thumbnails) round-trip correctly
- **Committed in:** `39ce1c5`

**2. [Rule 1 — bug] Percent-encoded item id resolved to a different (404ing) lookup than the raw form**
- **Found during:** Task 1, writing `TestItemHandler_PercentEncodedIDMatchesRaw`
- **Issue:** chi routes against `r.URL.RawPath` when the incoming request path contains percent-escapes, so `chi.URLParam(r, "id")` returned the literal `paperless%3A42` (still escaped) rather than the decoded `paperless:42` — causing the percent-encoded request to 404 against the index while the raw form succeeded, directly violating this plan's own acceptance criterion.
- **Fix:** Added `itemIDParam(r)` which decodes via `url.PathUnescape` before any store lookup.
- **Files modified:** `kernel/httpapi/item.go`
- **Verification:** `TestItemHandler_PercentEncodedIDMatchesRaw` passes; live curl diff of both forms returns identical bodies
- **Committed in:** `39ce1c5`

**3. [Rule 1 — bug] Acceptance-criteria grep for literal `/preview/`/`/thumb/` in `client.go` failed against the initial generic `Rendition(id, endpoint)` design**
- **Found during:** Task 1, running the plan's own acceptance-criteria greps
- **Issue:** `plugins/paperless/client.go` contains /preview/ and /thumb/` is a literal-substring check; the first implementation built those paths via `fmt.Sprintf("/api/documents/%d/%s/", id, endpoint)` with `endpoint` passed in from `plugin.go`, so the literal strings only existed in the caller, not in `client.go`.
- **Fix:** Replaced the generic method with named `Preview(ctx, id)` and `Thumbnail(ctx, id)` methods that embed the literal `"/api/documents/%d/preview/"` and `"/api/documents/%d/thumb/"` format strings directly in `client.go`.
- **Files modified:** `plugins/paperless/client.go`, `plugins/paperless/plugin.go`
- **Verification:** `grep -o "/preview/" plugins/paperless/client.go` and `grep -o "/thumb/"` both match; `go test ./...` still green
- **Committed in:** `39ce1c5`

**4. [Rule 1 — bug] My own PLUG-03 correlation validation broke two pre-existing green tests**
- **Found during:** Task 1, running `go test ./...` after adding `validateCorrelatedItem`
- **Issue:** `TestSyncAll_PersistsMatchedItems` and `TestSyncAll_KeywordOrderDoesNotAffectResult` used fake items with no `DeepLink` set, which the new validation now correctly rejects — these tests were never asserting anything about deep links before, so the omission had been harmless until this change made it load-bearing.
- **Fix:** Added a non-empty `DeepLink` to every fake item in those two tests.
- **Files modified:** `kernel/correlate/correlate_test.go`
- **Verification:** `go test ./kernel/correlate/...` green; new `TestSyncAll_RejectsUnspecifiedFidelityAndEmptyDeepLink` added alongside
- **Committed in:** `39ce1c5`

**5. [Rule 2 — missing critical functionality] Added `content_unavailable` error code**
- **Found during:** Task 1, implementing `/content` and `/thumbnail`
- **Issue:** The plan's Artifacts list names `item_not_found`, `source_unavailable`, and `unsupported_rendition_type` as the error codes this plan introduces, but doesn't name a code for "the item exists but the plugin reports no rendition available for this variant" — a real, reachable state (e.g. a document type paperless-ngx cannot thumbnail).
- **Fix:** Added `content_unavailable` (404) for exactly that case, kept distinct from `item_not_found` (the id doesn't exist in the index at all).
- **Files modified:** `kernel/httpapi/item.go`
- **Verification:** `TestItemContentHandler_UnavailableRendition404`
- **Committed in:** `39ce1c5`

---

**Total deviations:** 5 auto-fixed (1 plan/contract precedence, 3 bugs found via the plan's own acceptance criteria/tests, 1 missing error code). None required an architectural decision (Rule 4) — all resolved within the plan's existing boundaries and the already-locked 01-01 contract.
**Impact on plan:** No scope creep. Every deviation was either required to honor the already-locked unary-Fetch decision from 01-01, or was caught and fixed by the plan's own stated acceptance criteria and tests before completion.

## Issues Encountered

- A stale `webspaces serve` + `webspaces-plugin-paperless` process pair from the 01-01 session (PIDs 439639/439646) was still running on port 7777 at the start of this session; killed before running `./scripts/e2e-smoke.sh`. A second orphaned plugin subprocess (461849) was found after the smoke script's own trap killed only the parent kernel process; also killed before starting the final verification server.
- `.planning/config.json` (an `_auto_chain_active` field addition) and the tracked `kernel/webui/build/.gitkeep` placeholder (deleted by the npm build's output-directory cleanup, since the directory now holds real build artifacts) both show as locally modified/deleted in `git status`, but neither is caused by this plan's own task actions — left unstaged and undocumented as an out-of-scope pre-existing/build-tooling artifact per the deviation rules' scope boundary.

## User Setup Required

None beyond what 01-01 already established (`PAPERLESS_URL`/`PAPERLESS_TOKEN` in `.env`, already configured and verified working).

## Next Phase Readiness

- `make build && ./bin/webspaces sync && ./bin/webspaces serve` produces a single binary serving the full stream-to-detail-pane flow against the user's real paperless-ngx data (35 documents).
- A fresh `webspaces serve` (with `webspaces-plugin-paperless` as its child) is running on `http://127.0.0.1:7777/w/house-move` for the user to complete this plan's human-check step: click a document row and confirm (a) instant metadata + skeleton-then-fill sequencing and independent scroll containment between the two panes, (b) the "Open in paperless-ngx" CTA lands on exactly that document, and (c) stopping paperless-ngx produces the approved error copy with the header/CTA still usable.
- The detail-pane surface (row thumbnails, tag-pill polish, rich stream-row rendering) and any remaining stream-view refinements are deliberately out of scope here, owned by plans 01-03 and 01-04 per the phase's plan boundaries.

---
*Phase: 01-first-webspace-end-to-end*
*Completed: 2026-07-27*

## Self-Check: PASSED

All 18 files referenced above (backend: item.go, item_test.go, fetch_test.go, shared.go, main.go, client.go, plugin.go, host.go, store.go, correlate.go, correlate_test.go, routes.go, main.go; frontend: DetailPane.svelte, OpenInSource.svelte, api.ts, +page.svelte; this SUMMARY) confirmed present on disk. Both referenced commits (39ce1c5, 09b3faf) confirmed present in `git log --all`.
