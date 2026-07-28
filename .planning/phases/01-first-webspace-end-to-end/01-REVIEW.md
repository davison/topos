---
phase: 01-first-webspace-end-to-end
reviewed: 2026-07-28T00:00:00Z
depth: standard
files_reviewed: 46
files_reviewed_list:
  - cmd/webspaces/main.go
  - docs/api.md
  - docs/plugin-contract.md
  - internal/audit/doc.go
  - internal/audit/outbound_hosts_test.go
  - internal/audit/testdata/foreign_host_violation.go.txt
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
  - plugins/paperless/outbound_hosts_test.go
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
  warning: 10
  info: 8
  total: 18
status: issues_found
---

# Phase 01: Code Review Report

**Reviewed:** 2026-07-28T00:00:00Z
**Depth:** standard
**Files Reviewed:** 46
**Status:** issues_found

## Summary

This is a re-review after gap-closure plans 01-05 (SPA stylesheet import + smoke-test hardening) and 01-06 (paperless client outbound host-pinning + repo-wide egress AST audit) landed. Both gap-closure changes were verified directly against the code:

- **01-05**: `web/src/routes/+layout.svelte` now imports `../app.css`, and `scripts/e2e-smoke.sh` asserts a real stylesheet is linked and fetches successfully, and refuses to run against an already-occupied port. Both work as intended; no defects found in either.
- **01-06**: `plugins/paperless/client.go`'s `allowHost`/`DialContext`/`CheckRedirect` host-pinning is sound — verified against `outbound_hosts_test.go`'s redirect, pagination-`next`-URL, and predicate-table tests, all of which exercise real attack shapes (cross-host 302, a `next` URL pointing at a foreign host) rather than trivial cases. `internal/audit/outbound_hosts_test.go`'s AST-based repo-wide scan is a reasonable mechanical backstop, but has a soundness gap of its own (WR-08, below): it matches the *identifier name* `http`, not a resolved import of `net/http`, so an aliased import silently defeats it.

**None of the 12 findings from the previous review (0 critical / 7 warning / 5 info, committed as `87fb36f`) have been fixed.** `git diff 87fb36f..HEAD` touches only `web/src/routes/+layout.svelte`, `scripts/e2e-smoke.sh`, `plugins/paperless/client.go`, and adds `internal/audit/*`; every file the prior findings cite (`DetailPane.svelte`, `kernel/pluginhost/host.go`, `cmd/webspaces/main.go`, `kernel/config/config.go`, `kernel/httpapi/routes.go`, `web/src/routes/+page.svelte`, `web/src/lib/components/OpenInSource.svelte`, `plugins/paperless/plugin.go`) is byte-identical to the previously reviewed version. All 12 are carried forward below (re-verified against current line numbers, and against `plugins/paperless/client.go`'s new line numbers where the file changed), plus 6 new findings from this pass — concentrated, per this re-review's specific instruction, on the newly-added/changed files (`+layout.svelte`, `e2e-smoke.sh`, `client.go`, `outbound_hosts_test.go`, `internal/audit/*`) as well as two adjacent kernel files (`kernel/correlate/correlate.go`, `kernel/httpapi/webspaces.go`) whose interaction surfaced a genuine gap while re-tracing the sync/status data flow.

No Critical/Blocker-severity findings in either pass: no SQL injection (all queries parameterized), no hardcoded secrets, no XSS surface (no `{@html}`/`innerHTML`), and the rendition MIME allowlist + CSP/`nosniff` header set is implemented and tested correctly.

## Warnings

### WR-01: `DetailPane.svelte` reintroduces the UTC date-boundary bug `format.ts` was built to prevent

**File:** `web/src/lib/components/DetailPane.svelte:18-25`
**Issue:** `web/src/lib/format.ts` exports a dedicated, tested helper (`formatItemDate`) that pins date formatting to `timeZone: 'UTC'` specifically because paperless-ngx's `created` field is a date-only value stored as midnight UTC — formatting it in the browser's local timezone shows the wrong calendar day for any negative-offset viewer (documented in `format.ts`'s header comment, covered by a dedicated test in `format.test.ts`). `DetailPane.svelte` does not use that shared helper; it declares its own local `formatDate` that calls `toLocaleDateString` with no `timeZone` option:
```ts
function formatDate(unix: number): string {
	if (!unix) return '';
	return new Date(unix * 1000).toLocaleDateString(undefined, {
		year: 'numeric', month: 'short', day: 'numeric'
	});
}
```
`StreamRow.svelte` correctly imports and uses `formatItemDate` from `$lib/format`. `DetailPane.svelte` is the one place in the UI that reintroduces the exact bug the shared helper exists to prevent — a document dated `2024-01-01` (midnight UTC) renders as `Dec 31, 2023` in the detail pane for any user west of UTC, while the same item's row in the stream list correctly shows `1 Jan 2024`. **Still unfixed** since the prior review.
**Fix:** Delete the local `formatDate` and import the shared helper: `import { formatItemDate } from '$lib/format';` and use `{formatItemDate(item.timestamp_unix)}`.

### WR-02: `pluginhost.Host.Fetch` miscategorizes "no plugin registered" as `item_not_found` instead of `source_unavailable`

**File:** `kernel/pluginhost/host.go:190-193`
**Issue:**
```go
p := h.bySourceType(sourceType)
if p == nil {
	return FetchResult{}, fmt.Errorf("%w: no plugin registered for source type %q", ErrItemNotFound, sourceType)
}
```
This path only fires after `kernel/httpapi/item.go` already confirmed the item exists in the local index (`store.GetItem` returned `ok=true`). If the owning plugin process crashed, exited, or was removed from config, the item genuinely *does* exist — the failure is "source unreachable," not "id doesn't exist." Per `docs/api.md`, `source_unavailable` (502) is defined as exactly this case ("the source system was unreachable or errored"). Returning `item_not_found` (404) here misleads a client/agent into believing the id itself is invalid. **Still unfixed.**
**Fix:** `return FetchResult{}, fmt.Errorf("%w: no plugin registered for source type %q", ErrSourceUnavailable, sourceType)`.

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
`SyncAll` only returns a non-nil top-level error for a `RecordSyncRun` write failure — a per-webspace failure (e.g. one source's `Match` RPC errored) is reported via that webspace's `WebspaceResult.Err` in the returned (and here discarded, `_`) `results` slice. `runSync` (the CLI path, lines 137-143) does print each `WebspaceResult.Err`; the server's automatic startup sync has no equivalent, so a webspace that fails to sync at server startup produces **no log line at all**. **Still unfixed.**
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
**Issue:** `Validate` checks `src.BaseURL` and `src.Token` for every configured source but never checks `src.Plugin` (the binary filename `pluginhost.launch` joins into `binPath`). If `Plugin` is empty, `filepath.Join(pluginsDir, "")` resolves to `pluginsDir` itself — an existing directory — so `os.Stat(binPath)` succeeds and the clear "plugin binary not found" error is skipped entirely. `exec.Command(binPath)` then attempts to launch a directory as an executable, failing deep inside `go-plugin`'s handshake with an opaque low-level error instead of a clean, named config error. **Still unfixed.**
**Fix:** Add to `Validate`: `if strings.TrimSpace(src.Plugin) == "" { return fmt.Errorf("config: source %q has empty plugin binary name", name) }`, and in `pluginhost.launch`, reject a directory explicitly (`info.IsDir()`) rather than relying on `exec.Command` to fail downstream.

### WR-05: Inconsistent error handling between the `created` and `added` document fields

**File:** `plugins/paperless/client.go:370-393`
**Issue:** `toDocument` treats its two date fields inconsistently:
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
A malformed `created` is a hard error that propagates `toDocument` → `ListDocuments` → `Match`, turning into `codes.Unavailable` and failing the *entire webspace's* sync that cycle (per `correlate.SyncAll`'s all-or-nothing-per-webspace persistence, every other well-formed document in that response is dropped too, not just the bad one). A malformed `added` is silently swallowed to the Unix epoch with no error and no log line — the document silently sorts as if received in 1970, with zero diagnostic trail. The asymmetry is the bug: one malformed field takes down an entire sync, the other vanishes invisibly. **Still unfixed** (this file changed for gap G-01-6's host-pinning work, but `toDocument` itself is untouched — only its line numbers shifted).
**Fix:** Apply the same policy to both fields — either both fail loudly in a way `correlate.go`'s existing per-item `validateCorrelatedItem` rejection path can absorb (skip just that document, not the whole sync), or both degrade gracefully with a logged warning naming the document id and the bad value.

### WR-06: No HTTP server timeouts, and no per-request deadline on plugin `Fetch` calls

**File:** `cmd/webspaces/main.go:176`, `kernel/httpapi/item.go:93,154`
**Issue:** `runServe` starts the listener via bare `http.ListenAndServe(listen, router)` — no `ReadTimeout`/`WriteTimeout`/`IdleTimeout`. `ItemHandler`/`renditionHandler` also pass the inbound request's context straight through to `fetcher.Fetch` with no additional deadline. If a plugin subprocess hangs (source system stops responding mid-request, or the plugin deadlocks), the request-time `Fetch` call blocks indefinitely, holding the connection and goroutine open forever. **Still unfixed.**
**Fix:**
```go
srv := &http.Server{Addr: listen, Handler: router, ReadTimeout: 10 * time.Second, WriteTimeout: 60 * time.Second, IdleTimeout: 120 * time.Second}
return srv.ListenAndServe()
```
and wrap `Fetch` call sites with `ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second); defer cancel()`. (Related, same root cause: `runServe` also installs no `SIGINT`/`SIGTERM` handler, so the `defer host.Shutdown()`/`defer store.Close()` cleanup only runs if `http.ListenAndServe` itself returns an error — normal `Ctrl+C`/`systemd stop` termination skips these defers entirely, and the background startup-sync goroutine at line 162 shares `context.Background()` with no cancellation tied to shutdown, so it can still be mid-flight against a store/plugin-set that a fast-failing `ListenAndServe` has already torn down.)

### WR-07: Landing page shows misleading, wrong-context error copy and discards the actual error message

**File:** `web/src/routes/+page.svelte:9,16,36-40`
**Issue:** On fetch failure the caught error is stored (`error = e instanceof Error ? e.message : String(e)`) but only used as a truthiness check, never rendered. The copy shown is hardcoded and contextually wrong — it reads like it was copy-pasted from `StreamError.svelte` (which *is* scoped to one webspace):
```svelte
{:else if error}
	<p class="mt-6 text-[16px] text-muted-foreground">
		Couldn't load this webspace — the webspaces service didn't respond. Check that it's
		running, then retry.
	</p>
```
This is the `/` landing page listing *every* configured webspace — there is no single "this webspace" being loaded. It's also missing the retry affordance every other error state in the app provides. **Still unfixed.**
**Fix:** Correct the copy ("Couldn't load your webspaces"), render or drop the unused `error` string, and add a retry control matching `StreamError.svelte`'s pattern.

### WR-08 (new): Repo-wide egress AST scan matches the identifier name `http`, not a resolved `net/http` import — an import alias silently defeats it

**File:** `internal/audit/outbound_hosts_test.go:51-60, 188-212`; `plugins/paperless/readonly_test.go:31-42, 89-96`
**Issue:** Both AST scanners identify outbound-HTTP usage by checking `pkgIdent.Name == "http"` on a `*ast.SelectorExpr`/`*ast.CompositeLit`:
```go
if pkgIdent, ok := expr.X.(*ast.Ident); ok && pkgIdent.Name == "http" && outboundHTTPIdents[expr.Sel.Name] {
```
This is a textual/name match, not a resolved-import check (the parse uses `parser.SkipObjectResolution`, so no import-binding information is even available to resolve `pkgIdent` back to `"net/http"`). A future plugin author who writes `import nethttp "net/http"` and calls `nethttp.Get(...)` — or wraps `net/http` behind a differently-named local helper package — produces code this "mechanically enforced" gate (per `docs/plugin-contract.md`: "this is mechanically enforced, not just documented") does not detect, in either the repo-wide egress audit or the paperless-specific read-only-verbs audit. The gap is reachable by accident (a routine import-alias to avoid a name collision) as much as by intent — it isn't a hypothetical adversarial-only bypass.
**Fix:** Either resolve imports (drop `parser.SkipObjectResolution` and check the actual import path via `go/types`, or walk `ast.ImportSpec` to map local alias → import path), or, cheaper: additionally flag any import of `"net/http"` under a non-`http` alias as an offense in its own right, forcing every legitimate `net/http` import in the scanned tree to keep the canonical name the scanner already checks for.

### WR-09 (new): `WebspacesHandler` applies one global `last_sync` value to every webspace in the list

**File:** `kernel/httpapi/webspaces.go:36-39, 48-54`
**Issue:**
```go
var lastSync syncStatus
if run, ok, err := store.LatestSyncRun(ctx); err == nil && ok {
	lastSync = syncStatus{Status: run.Status, FinishedUnix: run.FinishedUnix, Error: run.Error}
}
...
for _, name := range names {
	resp.Webspaces = append(resp.Webspaces, webspaceSummary{..., LastSync: lastSync})
}
```
`store.LatestSyncRun` returns the single most-recently-inserted `sync_runs` row across **all sources** (`sync_runs` is keyed by `source_type`, and `correlate.SyncAll` records one row per source per sync cycle, not per webspace). The same `lastSync` value — including its `error` string — is stamped onto every webspace in the `/api/webspaces` response regardless of whether that webspace actually uses the source that failed. With today's single-source (paperless) config this is invisible, but it will silently misattribute a failing source's error to an unrelated, healthy webspace the moment a second source is configured (Phase 2+). Not caught by any current test — `contract_test.go`'s fixtures only ever seed one source.
**Fix:** Either compute a genuinely per-webspace sync status (e.g. worst status among the sources that actually matched into that webspace), or, if a single kernel-wide status is the intended Phase 1 contract, say so explicitly in `docs/api.md` rather than nesting it under each webspace object (which implies a per-webspace value).

### WR-10 (new): `correlate.SyncAll` marks a source's whole sync-run "error" even when it only failed for one of several webspaces

**File:** `kernel/correlate/correlate.go:62-64, 111-122`
**Issue:** `sourceErrors` is a `map[string]error` keyed by source type, written inside the per-webspace loop (`sourceErrors[src.SourceType()] = err`) whenever that source's `Match` call fails for *any* webspace, and never cleared. After the outer webspace loop finishes, the per-source `sync_runs` row is built once, using only this map's final state:
```go
if err, failed := sourceErrors[src.SourceType()]; failed {
	run.Status = "error"
	run.Error = err.Error()
} else if msgs := rejectedItems[src.SourceType()]; len(msgs) > 0 {
	...
}
```
If a source succeeds for webspace A (persisting real items, `sourceItemCounts` incremented) and fails for webspace B, the recorded `sync_runs` row for that source still reports `status: "error"` for the *entire* cycle — even though `item_count` reflects a real, partially-successful sync. A client/agent reading `/api/webspaces`'s `last_sync.status` (compounded by WR-09, which additionally broadcasts this single row to every webspace) sees "error" with no way to tell "this source is completely down" from "this source failed for one specific webspace but is otherwise healthy." `correlate_test.go` doesn't exercise the mixed-outcome case (its error test uses a single webspace).
**Fix:** Track success/failure per (source, webspace) pair, or at minimum record which webspace(s) a source's error applies to in `run.Error` so a reader isn't forced to assume total failure.

## Info

### IN-01: Duplicate fidelity-label map in `OpenInSource.svelte`

**File:** `web/src/lib/components/OpenInSource.svelte:11-15`
**Issue:** Re-declares the same three-entry `{exact, anchored, conversation-only}` map that `formatFidelity`/`FIDELITY_LABELS` in `web/src/lib/format.ts` already provides and unit-tests. Two independent copies will silently drift if the fidelity enum's display strings change in one place and not the other. **Still unfixed.**
**Fix:** `import { formatFidelity } from '$lib/format';` and use `formatFidelity(link.fidelity)`.

### IN-02: Unmatched `/api/*` routes fall through to the SPA's HTML 200 response instead of the documented JSON error envelope

**File:** `kernel/httpapi/routes.go:42, 51-59`
**Issue:** `spaHandler` (registered as `r.NotFound(...)`) serves `200.html` for any unmatched path, including a mistyped/deprecated/future-version path under `/api/`. `docs/api.md` states "every error response, on every route, uses the identical shared envelope" — but an unmatched `/api/...` path returns `200` with an HTML body, not a JSON `404` envelope. **Still unfixed.**
**Fix:** Register an explicit JSON 404 handler scoped to the `/api` prefix so only genuinely non-API paths reach the SPA fallback.

### IN-03: Hardcoded, predictable `/tmp` path in the e2e smoke script

**File:** `scripts/e2e-smoke.sh:114`
**Issue:** `CODE="$(curl -sS -o /tmp/webspaces-404-body.json -w '%{http_code}' ...)"` uses a fixed filename in shared `/tmp` rather than `mktemp` — unlike `CSS_TMP` a few lines earlier (added by gap-closure plan 01-05), which correctly uses `mktemp` and is cleaned up via `trap`. Low severity for a local-only smoke script, but now inconsistent with the pattern the script itself just established a few lines above, and the file is never cleaned up. **Still unfixed.**
**Fix:** `BODY_FILE="$(mktemp)"` and extend the existing `EXIT` trap to `rm -f "$BODY_FILE"` alongside `$CSS_TMP`.

### IN-04: `has_thumbnail` is unconditionally `true` for every paperless-ngx document

**File:** `plugins/paperless/plugin.go:104`
**Issue:** `toItem` sets `HasThumbnail: true` for every document regardless of whether paperless-ngx can actually produce a thumbnail for that file type. The UI degrades gracefully (`Thumbnail.svelte` falls back to a generic icon on load failure), so this isn't a correctness bug, but every stream row for an unthumbnailable document triggers a doomed `/api/items/{id}/thumbnail` request that predictably 404s as `content_unavailable`. **Still unfixed.**
**Fix:** If cheaply knowable (e.g. from the document's mimetype), set `HasThumbnail` based on whether a thumbnail is actually expected to succeed, per the field's documented contract in `docs/plugin-contract.md`.

### IN-05: Duplicated request-building code between `getJSON` and `Document`

**File:** `plugins/paperless/client.go:295-323, 399-428`
**Issue:** `Client.Document` and `Client.getJSON` both build a GET request, set the same `Authorization`/`Accept` headers, and branch on status code, but `Document` doesn't call through `getJSON` (it needs a special 404 branch `getJSON` doesn't support). **Still unfixed** (line numbers shifted from the previous review due to the host-pinning changes above them in the file, but the duplication itself is untouched).
**Fix:** Extract a shared `c.doGET(ctx, path, query) (*http.Response, error)` helper both callers use, keeping the 404-vs-decode branching per caller.

### IN-06 (new): Root landing page doesn't URL-encode the webspace name in its navigation link

**File:** `web/src/routes/+page.svelte:48`
**Issue:** `<a href={`/w/${ws.name}`}>` builds the link directly from the raw config-declared webspace name. Every other place this codebase turns a webspace name into a URL path segment goes through `encodeURIComponent` (`web/src/lib/api.ts`'s `getStream`: `` `/api/webspaces/${encodeURIComponent(webspace)}/stream` ``). `WebspaceHeader.svelte`'s own comment documents that "webspace names are user-defined config.toml keys of arbitrary length" — a name containing `/` would produce a broken link (interpreted as extra path segments rather than a literal character of the name).
**Fix:** `<a href={`/w/${encodeURIComponent(ws.name)}`}>` for consistency with the rest of the codebase's URL-building.

### IN-07 (new): `plugins/paperless/readonly_test.go`'s AST walk includes `_test.go` files, unlike `internal/audit/outbound_hosts_test.go`'s equivalent scan

**File:** `plugins/paperless/readonly_test.go:53-58`
**Issue:** `TestPluginsIssueOnlyGetRequests` walks every `.go` file under `plugins/` with only `if d.IsDir() || !strings.HasSuffix(path, ".go") { return nil }` — no `_test.go` exclusion. `internal/audit/outbound_hosts_test.go`'s repo-wide scan, enforcing the same PLUG-02 read-only principle at a broader scope, explicitly excludes test files (`strings.HasSuffix(path, "_test.go")`) on the stated rationale that "test binaries are never shipped." The two mechanisms enforcing the same invariant apply inconsistent policies; a legitimate future test referencing e.g. `http.MethodPost` while building a negative-control fixture server would fail this build-blocking test even though it ships nothing.
**Fix:** Align the two scanners' file-selection policy — most likely, exclude `_test.go` here too, matching the stated shipped-binaries rationale.

### IN-08 (new): `renditionHandler` treats a legitimately empty (zero-byte) rendition the same as "unavailable"

**File:** `kernel/pluginhost/host.go:208-211`, `kernel/httpapi/item.go:160-163`
**Issue:** `pluginhost.Host.Fetch` only populates `FetchResult.Body` `if len(resp.GetData()) > 0`. `renditionHandler` then treats `result.Body == nil` identically to `!result.Available` (both produce `404 content_unavailable`). A plugin that legitimately returns `Available: true` with a zero-length rendition (e.g. a genuinely empty file) would be reported as unavailable rather than served as an empty, allowlisted-MIME body. Narrow edge case with no current plugin implementation hitting it, but `Body == nil` conflates "no data was sent" with "not available," which `Available` already distinguishes.
**Fix:** If this ever becomes reachable, construct an empty `io.NopCloser` when `Available` is true regardless of data length, so `!result.Available` is the only path that produces `content_unavailable`.

---

_Reviewed: 2026-07-28T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
