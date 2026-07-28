---
phase: 01-first-webspace-end-to-end
reviewed: 2026-07-28T01:18:47Z
depth: standard
files_reviewed: 41
files_reviewed_list:
  - cmd/webspaces/main.go
  - docs/api.md
  - docs/plugin-contract.md
  - kernel/config/config.go
  - kernel/correlate/correlate.go
  - kernel/correlate/correlate_test.go
  - kernel/httpapi/contract_test.go
  - kernel/httpapi/item.go
  - kernel/httpapi/item_test.go
  - kernel/httpapi/routes.go
  - kernel/httpapi/stream.go
  - kernel/httpapi/webspaces.go
  - kernel/index/schema.go
  - kernel/index/store.go
  - kernel/index/store_test.go
  - kernel/item/item.go
  - kernel/pluginhost/host.go
  - kernel/webui/embed.go
  - plugins/paperless/client.go
  - plugins/paperless/fetch_test.go
  - plugins/paperless/main.go
  - plugins/paperless/plugin.go
  - plugins/paperless/readonly_test.go
  - proto/webspaces/v1/plugin.proto
  - scripts/e2e-smoke.sh
  - sdk/contract_test.go
  - sdk/shared.go
  - web/package.json
  - web/src/app.css
  - web/src/lib/api.ts
  - web/src/lib/components/DetailPane.svelte
  - web/src/lib/components/OpenInSource.svelte
  - web/src/lib/components/StreamEmpty.svelte
  - web/src/lib/components/StreamError.svelte
  - web/src/lib/components/StreamList.svelte
  - web/src/lib/components/StreamRow.svelte
  - web/src/lib/components/Thumbnail.svelte
  - web/src/lib/components/WebspaceHeader.svelte
  - web/src/lib/format.test.ts
  - web/src/lib/format.ts
  - web/src/routes/+layout.svelte
  - web/src/routes/+page.svelte
  - web/src/routes/w/[webspace]/+page.svelte
  - web/svelte.config.js
  - web/vite.config.ts
findings:
  critical: 0
  warning: 7
  info: 5
  total: 12
status: issues_found
---

# Phase 01: Code Review Report

**Reviewed:** 2026-07-28T01:18:47Z
**Depth:** standard
**Files Reviewed:** 41 (approx. — svelte.config.js/vite.config.ts/package.json/proto reviewed for consistency, not counted as independent logic units)
**Status:** issues_found

## Summary

Reviewed the full walking-skeleton slice: kernel HTTP API, correlation/sync engine, SQLite index, the go-plugin host, the reference paperless-ngx plugin, the gRPC contract + SDK, and the SvelteKit UI. The read-only guarantee (PLUG-02) is mechanically enforced (AST scan + proto allowlist test) and verified — no mutating HTTP verb exists anywhere under `plugins/`. No hardcoded secrets, no SQL injection (all queries are parameterized), no XSS surface (no `{@html}`/`innerHTML` usage; Svelte's default text interpolation is used throughout), and the rendition MIME allowlist + CSP/`nosniff` header set on `/content` and `/thumbnail` is correctly implemented and tested.

No Critical-severity findings. Several Warning-level correctness and robustness gaps were found, concentrated in three areas: (1) a UI date-formatting bug in `DetailPane.svelte` that silently reintroduces the exact UTC-boundary bug `format.ts` was written to prevent, (2) error-path categorization/observability gaps in the kernel (`pluginhost.Fetch`'s "no plugin registered" case, and `runServe`'s silent discard of per-webspace sync failures), and (3) missing/inconsistent validation and error handling around plugin config and date parsing. Info-level items are code duplication and minor UX/robustness nits.

## Warnings

### WR-01: `DetailPane.svelte` reintroduces the UTC date-boundary bug `format.ts` was built to prevent

**File:** `web/src/lib/components/DetailPane.svelte:18-25`
**Issue:** `web/src/lib/format.ts` has a dedicated, tested helper (`formatItemDate`) that pins date formatting to `timeZone: 'UTC'` specifically because paperless-ngx's `created` field is a date-only value stored as midnight UTC — formatting it in the browser's local timezone shows the wrong calendar day for any negative-offset viewer (documented in `format.ts`'s own header comment and covered by a dedicated test in `format.test.ts`). `DetailPane.svelte` does not use that shared helper; it declares its own local `formatDate` that calls `toLocaleDateString` with no `timeZone` option, silently falling back to the browser's local zone:
```ts
function formatDate(unix: number): string {
    if (!unix) return '';
    return new Date(unix * 1000).toLocaleDateString(undefined, {
        year: 'numeric',
        month: 'short',
        day: 'numeric'
    });
}
```
`StreamRow.svelte` correctly imports and uses `formatItemDate` from `$lib/format`. `DetailPane.svelte` is the one place in the UI that reintroduces the exact bug the shared helper exists to prevent — a document dated e.g. `2024-01-01` (midnight UTC) will render as `Dec 31, 2023` in the detail pane for any user west of UTC, while the same item's row in the stream list correctly shows `1 Jan 2024`.
**Fix:** Delete the local `formatDate` and import the shared helper:
```ts
import { formatItemDate } from '$lib/format';
// ...
<span>{formatItemDate(item.timestamp_unix)}</span>
```

### WR-02: `pluginhost.Host.Fetch` miscategorizes "no plugin registered" as `item_not_found` instead of `source_unavailable`

**File:** `kernel/pluginhost/host.go:189-193`
**Issue:**
```go
p := h.bySourceType(sourceType)
if p == nil {
    return FetchResult{}, fmt.Errorf("%w: no plugin registered for source type %q", ErrItemNotFound, sourceType)
}
```
This path is only reached after `kernel/httpapi/item.go` has already confirmed the item exists in the local index (`store.GetItem` returned `ok=true`). If the owning plugin process has since crashed, exited, or was removed from config, the item genuinely *does* exist — the failure is that its source is unreachable, not that the item is missing. Per `docs/api.md`'s own definition, `source_unavailable` (502) is "The live Fetch call to the owning plugin failed — the source system was unreachable or errored," which is exactly this case. Returning `item_not_found` (404) here is misleading to a client/agent: a 404 implies "this id doesn't exist," when the correct signal is "this id's source is currently unreachable."
**Fix:**
```go
p := h.bySourceType(sourceType)
if p == nil {
    return FetchResult{}, fmt.Errorf("%w: no plugin registered for source type %q", ErrSourceUnavailable, sourceType)
}
```

### WR-03: `runServe`'s background startup sync silently discards per-webspace failures

**File:** `cmd/webspaces/main.go:162-166`
**Issue:**
```go
go func() {
    if _, err := engine.SyncAll(ctx); err != nil {
        logger.Error("startup sync failed", "error", err)
    }
}()
```
`engine.SyncAll` only returns a non-nil top-level `error` for a `RecordSyncRun` write failure. A per-webspace failure (e.g. because one source's `Match` RPC errored) is instead reported via that webspace's `WebspaceResult.Err` in the returned `results` slice — which is discarded here (`_`). The CLI path (`runSync`, lines 137-143) does print each `WebspaceResult.Err` to stdout, but the server's automatic startup sync goroutine has no equivalent logging, so a webspace that fails to sync at server startup produces **no log line at all** — the operator has no way to discover it short of noticing the webspace is missing/stale via the UI and manually running `webspaces sync` from a terminal.
**Fix:**
```go
go func() {
    results, err := engine.SyncAll(ctx)
    if err != nil {
        logger.Error("startup sync failed", "error", err)
        return
    }
    for _, r := range results {
        if r.Err != nil {
            logger.Error("startup sync: webspace failed", "webspace", r.Webspace, "error", r.Err)
        }
    }
}()
```

### WR-04: Config validation doesn't catch an empty or invalid plugin binary name

**File:** `kernel/config/config.go:115-122`, `kernel/pluginhost/host.go:99-103`
**Issue:** `Validate` checks `src.BaseURL` and `src.Token` are non-empty for every configured source but never checks `src.Plugin` (the plugin binary filename used to build `binPath` in `pluginhost.launch`). If `Plugin` is empty, `filepath.Join(pluginsDir, "")` resolves to `pluginsDir` itself — a directory — and the existence check:
```go
binPath := filepath.Join(pluginsDir, src.Plugin)
if _, err := os.Stat(binPath); err != nil {
    return nil, fmt.Errorf("plugin binary %s not found: %w", binPath, err)
}
```
succeeds (`os.Stat` on a directory returns no error), so this clear, named error is skipped entirely. The subsequent `exec.Command(binPath)` attempt to launch a directory as an executable then fails deep inside `go-plugin`'s handshake with an opaque, low-level error (e.g. a permission/exec-format error) instead of the clear "source %q has empty plugin name" message the rest of `Validate` provides for `base_url`/`token`.
**Fix:** Add a `Plugin` non-empty check to `Validate`, and harden the existence check to reject non-regular-files:
```go
// in Validate:
if strings.TrimSpace(src.Plugin) == "" {
    return fmt.Errorf("config: source %q has empty plugin binary name", name)
}
// in pluginhost.launch:
info, err := os.Stat(binPath)
if err != nil {
    return nil, fmt.Errorf("plugin binary %s not found: %w", binPath, err)
}
if info.IsDir() {
    return nil, fmt.Errorf("plugin binary %s is a directory, not an executable", binPath)
}
```

### WR-05: Inconsistent error handling between the `created` and `added` document fields

**File:** `plugins/paperless/client.go:283-307`
**Issue:** `toDocument` treats the two date fields inconsistently:
```go
created, err := time.Parse("2006-01-02", d.Created)
if err != nil {
    if t, err2 := time.Parse(time.RFC3339, d.Created); err2 == nil {
        created = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
    } else {
        return Document{}, fmt.Errorf("parse created %q: %w", d.Created, err)
    }
}

added, err := time.Parse(time.RFC3339, d.Added)
if err != nil {
    added = time.Unix(0, 0).UTC()
}
```
A malformed `created` value is a hard error that propagates up through `toDocument` → `ListDocuments` → `Match`, turning into a `codes.Unavailable` gRPC error that fails the *entire* sync for that webspace (every other, well-formed document in that page/response is dropped too — see `correlate.SyncAll`'s all-or-nothing-per-webspace persistence). A malformed `added` value, by contrast, is silently swallowed and defaulted to the Unix epoch with no error and no log line — the document silently sorts as if received in 1970 (worst-case tie-break position), with no diagnostic trail for why. Neither behavior is clearly "more correct" than the other, but the asymmetry itself is the bug: one malformed field takes down an entire sync, the other is invisibly absorbed.
**Fix:** Apply the same policy to both fields — either both fail loudly (surfacing a clear per-document parse error that `correlate.go`'s existing per-item rejection path, not a whole-sync failure, could handle), or both degrade gracefully with a logged warning naming the document id and the bad value.

### WR-06: No HTTP server timeouts, and no per-request deadline on plugin `Fetch` calls

**File:** `cmd/webspaces/main.go:176`, `kernel/httpapi/item.go:93,154`
**Issue:** `runServe` starts the listener via the bare `http.ListenAndServe(listen, router)`, which uses `http.Server`'s zero-value defaults — no `ReadTimeout`, `WriteTimeout`, or `IdleTimeout`. Separately, `ItemHandler`/`renditionHandler` pass the inbound request's context straight through to `fetcher.Fetch` with no additional deadline applied. If a plugin subprocess hangs (e.g. the source system stops responding mid-request, or the plugin process itself deadlocks), the request-time `Fetch` call blocks indefinitely, holding the HTTP connection (and the underlying goroutine) open forever. Even for a loopback-only, single-user service, a hung or misbehaving plugin can accumulate stuck connections/goroutines over the life of a long-running `serve` process.
**Fix:**
```go
srv := &http.Server{
    Addr:         listen,
    Handler:      router,
    ReadTimeout:  10 * time.Second,
    WriteTimeout: 60 * time.Second, // renditions can be large; keep generous
    IdleTimeout:  120 * time.Second,
}
return srv.ListenAndServe()
```
and wrap the `Fetch` call sites with a bounded context, e.g. `ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second); defer cancel()`.

### WR-07: Landing page shows misleading, wrong-context error copy and discards the actual error message

**File:** `web/src/routes/+page.svelte:9,16,36-40`
**Issue:** On fetch failure, the caught error is stored (`error = e instanceof Error ? e.message : String(e)`) but the stored value is never rendered — it's only used as a truthiness check (`{:else if error}`). The copy shown instead is hardcoded and contextually wrong for this page:
```svelte
{:else if error}
    <p class="mt-6 text-[16px] text-muted-foreground">
        Couldn't load this webspace — the webspaces service didn't respond. Check that it's
        running, then retry.
    </p>
```
This is the `/` landing page that lists *every* configured webspace — there is no single "this webspace" being loaded here (that copy appears to be copy-pasted from `StreamError.svelte`, which *is* scoped to one webspace). It's also missing the retry affordance every other error state in this app provides (`StreamError.svelte` and `DetailPane.svelte`'s failure state both render a "Retry" button; this page does not).
**Fix:** Correct the copy to describe the actual page ("Couldn't load your webspaces"), and either render the captured `error` string or remove the unused assignment; add a retry control consistent with `StreamError.svelte`'s pattern.

## Info

### IN-01: Duplicate fidelity-label map in `OpenInSource.svelte`

**File:** `web/src/lib/components/OpenInSource.svelte:11-15`
**Issue:** Re-declares the same three-entry `{exact, anchored, conversation-only}` map that `formatFidelity`/`FIDELITY_LABELS` in `web/src/lib/format.ts` already provides (and unit-tests). Two independent copies of the same lookup table will silently drift if the fidelity enum's display strings are ever changed in one place and not the other.
**Fix:** `import { formatFidelity } from '$lib/format';` and use `formatFidelity(link.fidelity)` instead of the local `fidelityLabel` map.

### IN-02: Unmatched `/api/*` routes fall through to the SPA's HTML 200 response instead of the documented JSON error envelope

**File:** `kernel/httpapi/routes.go:51-59`
**Issue:** `spaHandler` (registered as `r.NotFound(...)`) serves `200.html` for *any* unmatched path, including a mistyped, deprecated, or future-version path under `/api/`. Per `docs/api.md`, "Every error response, on every route, uses the identical shared envelope" — but an unmatched `/api/...` path returns `200` with an HTML body, not a JSON `404` envelope, which could confuse a programmatic agent client that expects AGENT-02's uniform JSON contract on every request under `/api/`.
**Fix:** Explicitly register a JSON 404 handler for the `/api/*` prefix (e.g. `r.Route("/api", func(r chi.Router) { ... r.NotFound(jsonNotFound) })`) so only genuinely non-API paths reach the SPA fallback.

### IN-03: Hardcoded, predictable `/tmp` path in the e2e smoke script

**File:** `scripts/e2e-smoke.sh:79`
**Issue:** `CODE="$(curl -sS -o /tmp/webspaces-404-body.json ...)"` uses a fixed, predictable filename in the shared `/tmp` directory rather than `mktemp`. On a genuinely multi-user machine this is a minor TOCTOU/symlink-race hygiene issue; low severity for a local dev/CI-only smoke test, but easy to avoid.
**Fix:** `BODY_FILE="$(mktemp)"; trap '... rm -f "$BODY_FILE"' EXIT` and use `$BODY_FILE` in place of the hardcoded path.

### IN-04: `has_thumbnail` is unconditionally `true` for every paperless-ngx document

**File:** `plugins/paperless/plugin.go:104`
**Issue:** `toItem` sets `HasThumbnail: true` for every document regardless of whether paperless-ngx can actually produce a thumbnail for that file type. The UI degrades gracefully (`Thumbnail.svelte` falls back to a generic icon on image load failure), so this isn't a correctness bug, but it does mean every stream row for an unthumbnailable document triggers a doomed `/api/items/{id}/thumbnail` request that predictably 404s as `content_unavailable`.
**Fix:** If cheaply knowable (e.g. by file mimetype already present on the document), set `HasThumbnail` based on whether a thumbnail rendition is actually expected to succeed, per the field's documented contract in `docs/plugin-contract.md` ("lets the UI decide whether to render a thumbnail slot without an extra round-trip").

### IN-05: Duplicated request-building code between `getJSON` and `Document`

**File:** `plugins/paperless/client.go:208-236, 309-341`
**Issue:** `Client.Document` and `Client.getJSON` both build a `GET` request, set the same `Authorization`/`Accept` headers, and handle the same status-code branching, but `Document` doesn't call through `getJSON` (it needs to distinguish `404` specially, which `getJSON` doesn't support). The header-setting and error-wrapping logic is duplicated rather than shared.
**Fix:** Extract a small `c.doGET(ctx, path, query) (*http.Response, error)` helper that both `getJSON` and `Document` call, keeping the 404-vs-decode branching separate per caller.

---

_Reviewed: 2026-07-28T01:18:47Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
