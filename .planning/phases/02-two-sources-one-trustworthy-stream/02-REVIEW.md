---
phase: 02-two-sources-one-trustworthy-stream
reviewed: 2026-07-29T00:00:00Z
depth: standard
files_reviewed: 31
files_reviewed_list:
  - cmd/webspaces/main.go
  - docs/api.md
  - docs/plugin-contract.md
  - internal/audit/outbound_hosts_test.go
  - kernel/correlate/correlate.go
  - kernel/correlate/correlate_test.go
  - kernel/httpapi/agent.go
  - kernel/httpapi/agent_test.go
  - kernel/httpapi/item.go
  - kernel/index/store.go
  - kernel/index/store_test.go
  - kernel/pluginhost/host.go
  - plugins/silverbullet/match_test.go
  - plugins/silverbullet/plugin.go
  - scripts/assert-stylesheet.sh
  - scripts/e2e-smoke.sh
  - web/src/app.css
  - web/src/lib/api.ts
  - web/src/lib/components/DetailPane.svelte
  - web/src/lib/components/SourceFilterChips.svelte
  - web/src/lib/components/SourceHealthChip.svelte
  - web/src/lib/components/sources.test.ts
  - web/src/lib/components/staleness.test.ts
  - web/src/lib/components/StreamEmpty.svelte
  - web/src/lib/components/StreamList.svelte
  - web/src/lib/components/StreamRow.svelte
  - web/src/lib/components/WebspaceHeader.svelte
  - web/src/lib/format.test.ts
  - web/src/lib/format.ts
  - web/src/routes/+layout.svelte
  - web/src/routes/w/[webspace]/+page.svelte
findings:
  critical: 1
  warning: 9
  info: 5
  total: 15
status: issues_found
---

# Phase 02: Code Review Report (re-review after gap closure 02-06)

**Reviewed:** 2026-07-29T00:00:00Z
**Depth:** standard
**Files Reviewed:** 31
**Status:** issues_found

## Summary

This is a re-review of the full phase-02 file set after gap-closure plan
02-06 landed (commits 3a6d7ec, c34c5e1: removal of the shadowing
`--spacing-*` theme entries from `web/src/app.css` that collapsed named
`max-w-*` utilities, the new `scripts/assert-stylesheet.sh` guard, and
`scripts/e2e-smoke.sh`'s delegation to it). **That gap (G-02-1) is closed
correctly**: `app.css` no longer declares named `--spacing-<key>` entries
in its `@theme inline` block, and `assert-stylesheet.sh` mechanically
verifies both the `#020617` design-token presence (G-01-2) and that
`max-w-xs`/`md`/`3xl` resolve to their rem values rather than collapsing
to raw pixels — the previous review's underlying concern is fixed, and
the new script is a clean, well-documented extraction with no behavior
regression in `e2e-smoke.sh`.

This pass reviewed the entire listed file set at standard depth (not just
the gap-closure diff), re-verifying every still-open item from the prior
`02-REVIEW.md` against the current code and looking for defects
independently. Four of the prior review's five warnings and all four of
its Info items are **still present and unaddressed** (confirmed against
the current file contents below) — this phase's brief for this round did
not ask for their remediation, so they are carried forward rather than
dropped. This pass additionally found one new **Critical**-severity
defect: `DetailPane.svelte` has no guard against out-of-order async
responses, so rapidly selecting stream rows can display one item's
fetched content underneath a different item's header — a direct hit
against this phase's own "trustworthy stream" framing. Several new
warnings were also found: an analogous (lower-severity) race in the
page-level webspace loader, a CLI exit code that never reflects sync
failure, and two real gaps in the outbound-egress audit test's
enforcement mechanism (it can be defeated by import-aliasing `net/http`
or by building a foreign URL from more than one string literal, the
latter directly contradicting the test's own doc comment).

## Critical Issues

### CR-01: DetailPane.svelte has no guard against out-of-order async responses — can display the wrong item's content

**File:** `web/src/lib/components/DetailPane.svelte:42-64`
**Issue:** `loadContent(id)` is invoked from a `$effect` every time
`item.id` changes (line 63). It performs `await getItem(id)` and then
unconditionally assigns the result to the component-wide
`content`/`fetchErrorCode` state — there is no check that `id` still
matches the currently-selected `item.id` by the time the awaited call
resolves:

```ts
async function loadContent(id: string) {
	loadingContent = true;
	fetchErrorCode = null;
	content = null;
	try {
		const detail = await getItem(id);
		content = detail.content;
	} catch (err) {
		fetchErrorCode = err instanceof ApiError ? err.code : 'unknown_error';
	} finally {
		loadingContent = false;
	}
}

$effect(() => {
	loadContent(item.id);
});
```

If a user selects item A (triggering a `getItem('A')` call that's slow
for any reason — a degraded source, a large rendition, network jitter)
and then quickly selects item B before A's request resolves, B's request
can resolve first; A's late response then overwrites `content`, so the
pane shows B's header (title, labels, date — rendered synchronously from
the `item` prop, stage one of this component's own documented two-stage
render) with A's fetched body/rendition underneath it. This is exactly
the class of bug this phase's own "trustworthy stream" framing exists to
prevent: item metadata paired with a different item's content, with no
visual indication anything is wrong. There is no cancellation, no
`AbortController`, and no post-await staleness check anywhere in this
function.
**Fix:**
```ts
async function loadContent(id: string) {
	loadingContent = true;
	fetchErrorCode = null;
	content = null;
	try {
		const detail = await getItem(id);
		if (id !== item.id) return; // superseded by a newer selection
		content = detail.content;
	} catch (err) {
		if (id !== item.id) return;
		fetchErrorCode = err instanceof ApiError ? err.code : 'unknown_error';
	} finally {
		if (id === item.id) loadingContent = false;
	}
}
```
(An `AbortController` threaded through `getItem`/`fetch` would be the
more thorough fix — this minimal id-guard is sufficient to stop the wrong
data from ever being displayed.)

## Warnings

### WR-01: DetailPane.svelte's own date formatter reintroduces the timezone bug format.ts was built to fix

**File:** `web/src/lib/components/DetailPane.svelte:33-40` (used at line 72)
**Issue:** `web/src/lib/format.ts`'s `formatItemDate` deliberately pins
date formatting to UTC — paperless-ngx's `created` field is a date-only
value at midnight UTC, and formatting it in the viewer's local timezone
shows the wrong calendar day for anyone west of UTC (`format.test.ts`'s
"pins the calendar day to UTC..." test exists specifically to guard
this). `StreamRow.svelte` correctly imports and uses `formatItemDate`.
`DetailPane.svelte` still defines and uses its own local `formatDate`
that does **not** pin to UTC:
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
For the exact midnight-UTC boundary case `format.test.ts` guards against,
a user in `America/Los_Angeles` sees the stream row (via `formatItemDate`)
and the detail pane (via this local `formatDate`) disagree on the
calendar day for the same item, in the same view.
**Fix:** Delete the local `formatDate` and import `formatItemDate` from
`$lib/format`, exactly as `StreamRow.svelte` does.

### WR-02: Detail-pane iframes still lack the `sandbox` attribute the codebase's own comments claim exists

**File:** `web/src/lib/components/DetailPane.svelte:143,149`; referenced from `kernel/httpapi/item.go:190-196`
**Issue:** `kernel/httpapi/item.go`'s `renditionHandler` comment (and
`docs/api.md`'s rendition section) describe two independent layers
blocking script execution in a rendered `text/html` document: the
`Content-Security-Policy: ...; sandbox` response header, **and** "the
embedding `<iframe>`'s own sandbox attribute in DetailPane.svelte".
Neither iframe in `DetailPane.svelte` carries a `sandbox` attribute:
```svelte
<iframe title={item.title} src={contentUrl(item.id)} class="h-full w-full"></iframe>
```
(both the `text/html` branch at line 143 and the PDF-rendition branch at
line 149). The response-level CSP `sandbox` directive does apply
sandboxing to the framed document independent of the parent `<iframe>`
element's own attribute, so this is not currently an exploitable bypass —
but the comment in `item.go` asserts a second, independent client-side
defense-in-depth layer that does not exist in the code. A future change
that widens or accidentally drops the CSP header (e.g. while iterating on
`style-src`, as already happened once per that file's own history) would
then remove *all* iframe sandboxing, not just the "redundant" layer the
comment implies still remains.
**Fix:** Either add `sandbox=""` (or an explicit minimal token list) to
both iframe elements in `DetailPane.svelte` to match what the comment
describes, or correct the comment in `kernel/httpapi/item.go` (and
`docs/api.md`) to stop claiming a client-side defense layer that isn't
implemented.

### WR-03: agentStreamHandler silently swallows a sync-run-lookup error that its sibling handler surfaces as a 500

**File:** `kernel/httpapi/agent.go:232-234`
**Issue:** `agentWebspacesHandler` (lines 156-160) treats a
`store.LatestSyncRunPerSource` failure as a hard error and writes
`internal_error`/500. `agentStreamHandler`, calling the exact same store
method, instead does:
```go
if runs, err := store.LatestSyncRunPerSource(ctx); err == nil {
	resp.Sync = aggregateSyncStatus(filterRunsByGrant(runs, granted))
}
```
— on error, it silently leaves `resp.Sync` at its zero value and still
returns `200 OK` with the (correct) items but a bogus, unset `sync`
object indistinguishable from "no source has ever synced". This is an
inconsistent error-handling policy between two handlers in the same file
solving the same problem, and it hides a real index-layer failure (disk
I/O error, poisoned connection) from the caller.
**Fix:**
```go
runs, err := store.LatestSyncRunPerSource(ctx)
if err != nil {
	WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
	return
}
resp.Sync = aggregateSyncStatus(filterRunsByGrant(runs, granted))
```

### WR-04: No OS signal handling for graceful shutdown in cmd/webspaces/main.go

**File:** `cmd/webspaces/main.go:155-185` (`runServe`)
**Issue:** `runServe` wires up `defer host.Shutdown()`, `defer
store.Close()`, and `defer cancel()`, but `main.go` never installs a
`signal.Notify`/`os.Signal` handler (no `os/signal` or `syscall` import
anywhere in the file), and the bare `http.ListenAndServe(listen, router)`
call only returns (running those deferred calls) on a listener-level
error. A normal `Ctrl+C` (`SIGINT`) or a process manager's `SIGTERM`
terminates the process immediately via the Go runtime's default signal
disposition, without running any deferred function. Concretely, on a
normal shutdown: the scheduler's background goroutines are killed
abruptly rather than via `ctx.Done()`; `store.Close()` (which checkpoints
the SQLite WAL) never runs; and `host.Shutdown()` (`p.Kill()` on every
launched plugin subprocess) never runs from the kernel's own shutdown
path.
**Fix:**
```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
```
and use that `ctx` for both the scheduler and an `http.Server` whose
`Shutdown(ctx)` is called on signal (replacing the bare
`http.ListenAndServe` call), so a normal `Ctrl+C` exercises the same
clean-shutdown path the existing deferred calls already assume exists.

### WR-05: +page.svelte's stream/source loads have the same out-of-order-response race as CR-01

**File:** `web/src/routes/w/[webspace]/+page.svelte:46-55, 125-129`
**Issue:** `load()` and `loadSources()` are invoked from the `$effect` at
line 125 whenever `webspace` changes, and both unconditionally assign
`response`/`sources` on resolution with no check that the in-flight
request still corresponds to the currently-active `webspace`. Rapidly
switching between two webspaces (back/forward navigation, or a quick
link-click sequence) can let an older webspace's slower response resolve
after a newer one's faster response, leaving the page displaying one
webspace's title alongside a different webspace's stream items.
**Fix:** Track the in-flight webspace and ignore stale resolutions:
```ts
async function load() {
	const requested = webspace;
	loadState = 'loading';
	try {
		const res = await getStream(requested);
		if (requested !== webspace) return;
		response = res;
		loadState = 'ready';
	} catch {
		if (requested !== webspace) return;
		response = null;
		loadState = 'error';
	}
}
```
Apply the same pattern to `loadSources()`.

### WR-06: +page.svelte's sync-status poll interval is never cleared on component teardown

**File:** `web/src/routes/w/[webspace]/+page.svelte:74-84`
**Issue:** `ensurePolling()` starts a `setInterval` (`pollHandle`) that
calls `loadSources()` every 2s until no source is syncing, but there is
no `onDestroy` (or equivalent teardown) that clears `pollHandle` when the
component itself is destroyed — e.g. the user navigates away from
`/w/[webspace]` to another route entirely while a sync is still
in-flight. If that happens, the interval keeps firing indefinitely,
issuing background `GET /api/sources` requests and mutating state no
longer rendered by anything, until the polled source(s) coincidentally
stop syncing (which may never happen if a source is genuinely stuck).
**Fix:**
```ts
import { onDestroy } from 'svelte';
// ...
onDestroy(() => {
	if (pollHandle !== null) {
		clearInterval(pollHandle);
		pollHandle = null;
	}
});
```

### WR-07: `webspaces sync`'s exit code never reflects a sync failure

**File:** `cmd/webspaces/main.go:131-153`
**Issue:** `runSync()` iterates `coord.RefreshAll(ctx)`'s results and
prints `"%s: error: %s"` for any source whose `Status == "error"`, but
always `return nil` afterward regardless of whether any (or every)
source failed. `main()` only calls `fatal(err)` (which sets a non-zero
exit code) when `runSync()` itself returns a non-nil error — so
`webspaces sync` exits `0` even when every configured source failed to
sync. This defeats any cron/CI/automation caller that checks the process
exit code to detect a failed sync (the operator has to parse stdout
instead), and is inconsistent with the emphasis this phase places on
surfacing per-source failure faithfully (`sync.status`, `/agent/v1`
parity, etc.) through every other surface.
**Fix:**
```go
func runSync() error {
	// ...
	results := coord.RefreshAll(ctx)

	var failed bool
	for _, r := range results {
		if r.Status == "error" {
			fmt.Printf("%s: error: %s\n", r.Source, r.Error)
			failed = true
			continue
		}
		fmt.Printf("%s: %d items\n", r.Source, r.ItemCount)
	}
	if failed {
		return fmt.Errorf("one or more sources failed to sync")
	}
	return nil
}
```

### WR-08: The outbound-egress audit test can be defeated by import-aliasing net/http

**File:** `internal/audit/outbound_hosts_test.go:193-217`
**Issue:** The AST scan identifies outbound HTTP construction purely by
matching the literal identifier spelling `"http"` on a `SelectorExpr`'s
`X` (`pkgIdent.Name == "http"`), never by resolving which import path
that identifier is bound to (the file is parsed with
`parser.SkipObjectResolution`, so import resolution isn't even
performed). A file that imports `net/http` under any other local name —
`import nethttp "net/http"` and then `nethttp.Get(...)` — produces a
`SelectorExpr` whose base ident is named `"nethttp"`, which never matches
`outboundHTTPIdents`/`outboundHTTPTypes`, so the call is invisible to this
enforcement mechanism entirely. This is the mechanical control the
project relies on to guarantee "the kernel...must have no egress at all"
and that only sanctioned plugin client files talk to the network — a
one-line import rename silently defeats it.
**Fix:** Walk each file's `*ast.ImportSpec`s first, build a map from
local identifier (explicit alias, or the default `http` when unaliased)
to import path, and only flag a `SelectorExpr`/`CompositeLit` whose base
identifier resolves to import path `"net/http"` — not one that merely
happens to be spelled `"http"`.

### WR-09: The outbound-egress audit test's URL-literal scan is defeated by string concatenation, contrary to its own doc comment

**File:** `internal/audit/outbound_hosts_test.go:83-92, 180-192`
**Issue:** The function doc comment claims "a comment or a string built
by concatenation cannot trip or defeat this check," but
`scanFileForForeignEgress` only inspects individual `*ast.BasicLit`
string nodes against `foreignURLHost`, which requires the *whole* literal
to start with a scheme-and-authority prefix
(`schemeAuthority.MatchString(val)`) and have a non-empty parsed
`Hostname()`. A URL built from two or more literal pieces — e.g.
`"http://" + "evil.example.com/" + path`, or
`fmt.Sprintf("http://%s/api", "evil.example.com")` — is invisible to this
scan: `"http://"` alone parses to an empty hostname (not flagged, per
`foreignURLHost`'s own `h == ""` early return) and the remaining pieces
never match the scheme-prefix regex at all. This directly contradicts the
doc comment's stated guarantee and is a real gap in a security-enforcing
test (the same class of gap as WR-08 — both live in the mechanism meant
to guarantee "no personal content leaves the user's machine" per
PROJECT.md's Constraints).
**Fix:** At minimum, correct the doc comment to state the actual
limitation (single-literal foreign URLs only); consider additionally
flagging any `*ast.BinaryExpr` with `token.ADD` where either operand is a
string literal containing a scheme prefix, and any `fmt.Sprintf`/
`net/url.URL{}` construction whose format/base string contains a foreign
scheme prefix, to narrow (not fully close) this gap.

## Info

### IN-01: correlate.Engine.Sources is a dead struct field

**File:** `kernel/correlate/correlate.go:33-42`
**Issue:** `Engine.Sources []Source` is declared and documented
("retained for callers that build an Engine alongside a
kernel/syncer.Coordinator... but is no longer read by anything in this
package"), yet every `&Engine{...}` construction site in the repository
(`cmd/webspaces/main.go`, `kernel/correlate/correlate_test.go`) sets only
`Store` and `Config` — `Sources` is never assigned, and the comment
itself confirms it's never read. A dead exported field on a small,
actively-used struct invites a future caller to populate it under the
mistaken belief it does something.
**Fix:** Remove the field (and its doc comment) unless a concrete
near-term caller is identified.

### IN-02: /agent/v1 grant filtering is keyed by source_type, not the config name that carries the grant

**File:** `kernel/httpapi/agent.go:40-54`
**Issue:** `grantedSourceTypes` resolves each `agent.read = true` config
name to a `source_type` via `prober.SourceTypesByName()`, and every
downstream filter checks membership by `source_type` alone
(`granted[it.SourceType]`), never by config name. Nothing in the plugin
contract or `pluginhost.Discover` prevents two different
`[sources.<name>]` entries from launching plugins that both report the
same `Describe`-reported `source_type`. In that (config-error-only, not
externally-reachable) scenario, granting `agent.read = true` on one
config entry would also expose the other's items through `/agent/v1`,
even if that second entry's own `[sources.<name>.agent]` block says
`read = false`.
**Fix:** Either document that `source_type` must be unique across
configured sources for per-source grant isolation to hold, or key the
granted set (and carry it through the filtering) by config name rather
than `source_type` alone.

### IN-03: Rendition-serving logic is duplicated verbatim between item.go and agent.go

**File:** `kernel/httpapi/item.go:147-213`, `kernel/httpapi/agent.go:302-345`
**Issue:** `agentRenditionHandler` is a near line-for-line copy of
`renditionHandler` — same MIME allowlist check, same hardened
five-header response set, same streaming logic — differing only in the
grant check and the not-found response shape. This is called out in
`agent.go`'s own comment as a deliberate scope decision, but it means the
security-relevant CSP header string
(`"default-src 'none'; style-src 'unsafe-inline'; object-src 'none';
sandbox"`) now exists as two independent literals that must be kept in
sync by hand — a future change to one (e.g. tightening the CSP further)
is one easy oversight away from silently not applying to the other
namespace.
**Fix:** Factor the shared header-writing/allowlist/streaming logic into
one unexported helper both handlers call, parameterized only by the
not-found response and URL prefix, next time either file is touched.

### IN-04: docs/plugin-contract.md's WEBSPACES_SOURCE_CONFIG example omits the ca_cert key the kernel actually sends

**File:** `docs/plugin-contract.md:155-165`; `kernel/pluginhost/host.go:114-119`
**Issue:** The contract doc's worked example of the JSON a plugin
receives via `WEBSPACES_SOURCE_CONFIG` shows only three keys:
```json
{ "base_url": "https://paperless.example.lan", "token": "abc123...", "api_version": "10" }
```
`pluginhost.launch`, however, always marshals a fourth key:
```go
sourceConfig, err := json.Marshal(map[string]string{
	"base_url":    src.BaseURL,
	"token":       src.Token,
	"api_version": src.APIVersion,
	"ca_cert":     src.CACert,
})
```
A third-party plugin author following this document exactly (per its own
stated goal — "you should be able to write a working plugin" from this
document alone) would not learn that `ca_cert` is always present (as `""`
when unset) in the JSON blob their plugin receives.
**Fix:** Add `ca_cert` to the documented example JSON and a one-line note
on when it's non-empty (custom CA for a self-signed/internal TLS
endpoint).

### IN-05: getJSON/postJSON in api.ts are near-duplicate implementations

**File:** `web/src/lib/api.ts:99-139`
**Issue:** `getJSON` and `postJSON` share identical error-envelope
parsing/fallback logic (lines 101-113 vs 125-137), differing only in the
`fetch` call's method. This is a maintenance hazard: a future fix to the
error-handling behavior (e.g. adding a status-code-specific branch) is
easy to apply to one function and forget in the other.
**Fix:** Extract the shared response-handling logic into one helper, e.g.
`async function request<T>(path: string, init?: RequestInit): Promise<T>`,
and have both `getJSON`/`postJSON` call it with the appropriate `init`.

---

_Reviewed: 2026-07-29T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
