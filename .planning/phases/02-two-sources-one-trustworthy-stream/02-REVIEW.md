---
phase: 02-two-sources-one-trustworthy-stream
reviewed: 2026-07-29T00:00:00Z
depth: standard
files_reviewed: 28
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
  critical: 0
  warning: 5
  info: 4
  total: 9
status: issues_found
---

# Phase 02: Code Review Report

**Reviewed:** 2026-07-29T00:00:00Z
**Depth:** standard
**Files Reviewed:** 28
**Status:** issues_found

## Summary

This is a re-review of the file set from the prior `02-REVIEW.md`, after gap-
closure plan `02-05` fixed that review's `CR-01` (SilverBullet `Match`
swallowing per-page read failures). This review replaces the prior report.

**CR-01 verified fixed.** `plugins/silverbullet/plugin.go`'s `Match` now
correctly distinguishes `ErrNotFound` (safe to skip — the page was deleted
between listing and read) from every other `ReadFile` error (propagated so
`errgroup` cancels and `Match` returns `codes.Unavailable`), exactly per
`docs/plugin-contract.md`'s Match section. `plugins/silverbullet/match_test.go`
now covers both the previously-untested total-outage case
(`TestMatch_AllPageReadsFail_ReturnsUnavailable`) and a mid-sync partial
failure (`TestMatch_OutageMidSync_AuthFailure_ReturnsUnavailable`), plus a
regression test that the error message never leaks the bearer token. No
remaining defect found in this path.

No new Critical-severity issues were found in this pass. Four of the prior
review's warnings were re-evaluated against the current code and all four are
**still present** (the phase deliberately left them unaddressed, per this
review's brief): a timezone-formatting inconsistency in `DetailPane.svelte`,
a documented-but-absent iframe `sandbox` attribute, a silently-swallowed
sync-status-lookup error in the `/agent/v1` stream handler, and the absence of
OS signal handling for graceful kernel shutdown. One new warning was found in
this pass (an uncleaned polling interval in the webspace page component), plus
several Info-level maintainability items.

## Critical Issues

None found in this pass.

## Warnings

### WR-01: `DetailPane.svelte`'s own date formatter still reintroduces the timezone bug `format.ts` was built to fix

**File:** `web/src/lib/components/DetailPane.svelte:33-40` (used at line 72)

**Issue:** `web/src/lib/format.ts`'s `formatItemDate` pins date formatting to
UTC deliberately — its own comment and `format.test.ts`'s
`'pins the calendar day to UTC even for a timestamp a negative-offset zone
would render as the previous day'` test explain why: paperless-ngx's
`created` field is a date-only value at midnight UTC, and formatting it in
the viewer's local timezone shows the wrong calendar day for anyone west of
UTC. `StreamRow.svelte` correctly imports and uses this shared helper.

`DetailPane.svelte` still defines and uses its own local `formatDate` that
does **not** pin to UTC:

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

For the exact midnight-UTC boundary case `format.test.ts` guards against, a
user in `America/Los_Angeles` sees the stream row (via `formatItemDate`) and
the detail pane (via this local `formatDate`) disagree on the calendar day
for the same item, in the same view.

**Fix:** Delete the local `formatDate` and import `formatItemDate` from
`$lib/format`, exactly as `StreamRow.svelte` does:

```ts
import { formatItemDate, detailPaneState } from '$lib/format';
// ...
<span>{formatItemDate(item.timestamp_unix)}</span>
```

### WR-02: Detail-pane iframes still lack the `sandbox` attribute the codebase's own comments claim exists

**File:** `web/src/lib/components/DetailPane.svelte:143,149`; referenced from `kernel/httpapi/item.go:190-196`

**Issue:** `kernel/httpapi/item.go`'s `renditionHandler` comment (and
`docs/api.md`'s rendition section) describe two independent layers blocking
script execution in a rendered `text/html` document: the
`Content-Security-Policy: ...; sandbox` response header, **and** "the
embedding `<iframe>`'s own sandbox attribute in DetailPane.svelte". Neither
iframe in `DetailPane.svelte` carries a `sandbox` attribute:

```svelte
<iframe title={item.title} src={contentUrl(item.id)} class="h-full w-full"></iframe>
```
(both the `text/html` branch at line 143 and the PDF-rendition branch at line 149)

The CSP `sandbox` directive on the HTTP response does apply sandboxing to the
framed document independent of the parent `<iframe>` element's own attribute,
so this is not currently an exploitable bypass — but the comment in `item.go`
asserts a second, independent client-side defense-in-depth layer that does
not exist in the code. A future change that widens or accidentally drops the
CSP header (e.g. while iterating on `style-src`, as already happened once per
that file's own history) would then remove *all* iframe sandboxing, not just
the "redundant" layer the comment implies still remains.

**Fix:** Either add `sandbox=""` (or an explicit minimal token list) to both
iframe elements in `DetailPane.svelte` to match what the comment describes,
or correct the comment in `kernel/httpapi/item.go` (and `docs/api.md`) to
stop claiming a client-side defense layer that isn't implemented.

### WR-03: `agentStreamHandler` still silently swallows a `LatestSyncRunPerSource` failure, masking a real store error as "never synced"

**File:** `kernel/httpapi/agent.go:230-234`

**Issue:**

```go
if runs, err := store.LatestSyncRunPerSource(ctx); err == nil {
	resp.Sync = aggregateSyncStatus(filterRunsByGrant(runs, granted))
}
```

A genuine SQLite failure here (disk I/O error, corrupted index, a poisoned
connection) is indistinguishable to the client from "no source has ever
synced" — the response is still `200` with `sync.status: ""`, the documented
neutral "unknown" state, rather than a `500 internal_error`. This is
inconsistent with the sibling handler in the same file,
`agentWebspacesHandler` (lines 156-160), which correctly propagates this
exact error:

```go
runs, err := store.LatestSyncRunPerSource(ctx)
if err != nil {
	WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
	return
}
```

(This file set does not include `kernel/httpapi/stream.go` /
`kernel/httpapi/webspaces.go` this round, so this review cannot confirm
whether the equivalent pattern the prior review flagged there has changed;
the instance in `agent.go`, which is in scope, is unchanged.)

**Fix:** Make `agentStreamHandler` match `agentWebspacesHandler`'s pattern —
return `500 internal_error` on a genuine store failure instead of silently
downgrading it to the neutral "never synced" UI state.

### WR-04: Still no OS signal handling for graceful shutdown in `cmd/webspaces/main.go`

**File:** `cmd/webspaces/main.go:155-185` (`runServe`)

**Issue:** `runServe` wires up `defer host.Shutdown()`, `defer store.Close()`,
and `defer cancel()`, but `main.go` never installs a `signal.Notify`/
`os.Signal` handler, and `http.ListenAndServe` only returns (running those
deferred calls) on a listener-level error. A normal `Ctrl+C` (`SIGINT`) or a
process manager's `SIGTERM` terminates the process immediately via the Go
runtime's default signal disposition, without running any deferred function.
Concretely, on a normal shutdown:

- The scheduler's background goroutines are killed abruptly rather than via
  `ctx.Done()`.
- `store.Close()` (which checkpoints the SQLite WAL) never runs.
- `host.Shutdown()` (`p.Kill()` on every launched plugin subprocess) never
  runs from the kernel's own shutdown path — plugin cleanup relies entirely
  on `hashicorp/go-plugin`'s own dead-parent detection inside each plugin
  subprocess rather than an explicit, logged kernel-side shutdown.

**Fix:**

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
```

and use that `ctx` for both the scheduler and an `http.Server` whose
`Shutdown(ctx)` is called on signal (replacing the bare
`http.ListenAndServe` call), so a normal `Ctrl+C` exercises the same
clean-shutdown path the existing deferred calls already assume exists.

### WR-05 (new): The webspace page's sync-status polling interval is never cleared when the component is destroyed

**File:** `web/src/routes/w/[webspace]/+page.svelte:74-84`

**Issue:** `ensurePolling` starts a `setInterval` that polls
`GET /api/sources` every 2s while any source is syncing, and clears itself
only once no source reports `syncing`:

```ts
let pollHandle: ReturnType<typeof setInterval> | null = null;
function ensurePolling() {
	if (pollHandle !== null) return;
	pollHandle = setInterval(async () => {
		await loadSources();
		if (!sources.some((s) => s.syncing) && pollHandle !== null) {
			clearInterval(pollHandle);
			pollHandle = null;
		}
	}, 2000);
}
```

There is no `onDestroy` (or equivalent teardown) that clears `pollHandle`
when this component instance is torn down. If a user triggers a refresh
(starting the poll loop) and then navigates away to a route that destroys
this component before the in-flight sync finishes — e.g. back to `/` or to
an entirely different part of the app, if one is added later — the interval
keeps firing indefinitely: it continues issuing `GET /api/sources` requests
and mutating `sources`/`sourcesState` on a component instance nothing is
rendering from any more, until the syncing source(s) eventually finish (which
may never happen if that source is genuinely stuck). This is a real resource
leak, not just a style nit — the loop has no lifecycle-bound upper bound
independent of the sync it's polling for.

**Fix:** Clear the interval on component teardown regardless of sync state:

```ts
import { onDestroy } from 'svelte';
// ...
onDestroy(() => {
	if (pollHandle !== null) clearInterval(pollHandle);
});
```

## Info

### IN-01 (new): `correlate.Engine.Sources` is a dead struct field

**File:** `kernel/correlate/correlate.go:38-42`

**Issue:** `Engine.Sources []Source` is declared and documented ("retained
for callers that build an Engine alongside a kernel/syncer.Coordinator...
but is no longer read by anything in this package"), but a repo-wide search
shows every `&Engine{...}` construction site (`cmd/webspaces/main.go`,
`kernel/correlate/correlate_test.go`, `kernel/syncer/coordinator_test.go`,
`kernel/syncer/scheduler_test.go`) sets only `Store` and `Config` — `Sources`
is never assigned, and the comment itself confirms it's never read. It is
fully dead, not merely unused-by-this-package.

**Fix:** Remove the field (and its doc comment) unless a concrete near-term
caller is identified; a dead exported field on a small, actively-read struct
invites a future caller to populate it under the mistaken belief it does
something.

### IN-02: `/agent/v1` grant filtering is keyed by `source_type`, not the config name that carries the grant

**File:** `kernel/httpapi/agent.go:40-54`

**Issue:** `grantedSourceTypes` resolves each `agent.read = true` config name
to a `source_type` via `prober.SourceTypesByName()`, and every downstream
filter checks membership by `source_type` alone (`granted[it.SourceType]`),
never by config name. Nothing in the plugin contract or `pluginhost.Discover`
prevents two different `[sources.<name>]` entries from launching plugins
that report the same `Describe`-reported `source_type`. In that
(admittedly config-error-only, not externally-reachable) scenario, granting
`agent.read = true` on one config entry would also expose the other's items
through `/agent/v1`, even if that second entry's own
`[sources.<name>.agent]` block says `read = false`.

**Fix:** Either document that `source_type` must be unique across configured
sources for per-source grant isolation to hold, or key the granted set (and
carry through the filtering) by config name rather than `source_type` alone.

### IN-03: Rendition-serving logic is duplicated verbatim between `item.go` and `agent.go`

**File:** `kernel/httpapi/item.go:147-213`, `kernel/httpapi/agent.go:302-345`

**Issue:** `agentRenditionHandler` is a near line-for-line copy of
`renditionHandler` — same MIME allowlist check, same five-header hardened
response set, same streaming logic — differing only in the grant check and
the not-found response shape. This is called out in `agent.go`'s own comment
as a deliberate scope decision, but it means the security-relevant CSP header
string (`"default-src 'none'; style-src 'unsafe-inline'; object-src 'none';
sandbox"`) now exists as two independent literals that must be kept in sync
by hand — a future change to one (e.g. tightening the CSP further) is one
easy oversight away from silently not applying to the other namespace.

**Fix:** Factor the shared header-writing/allowlist/streaming logic into one
unexported helper both handlers call, parameterized only by the not-found
response and URL prefix, next time either file is touched.

### IN-04 (new): `docs/plugin-contract.md`'s `WEBSPACES_SOURCE_CONFIG` example omits the `ca_cert` key the kernel actually sends

**File:** `docs/plugin-contract.md:155-165`; `kernel/pluginhost/host.go:114-119`

**Issue:** The contract doc's worked example of the JSON a plugin receives
via `WEBSPACES_SOURCE_CONFIG` shows only three keys:

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

**Fix:** Add `ca_cert` to the documented example JSON and a one-line note on
when it's non-empty (custom CA for a self-signed/internal TLS endpoint).

---

_Reviewed: 2026-07-29T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
