---
phase: 02-two-sources-one-trustworthy-stream
reviewed: 2026-07-29T00:00:00Z
depth: standard
files_reviewed: 47
files_reviewed_list:
  - cmd/webspaces/main.go
  - config.example.toml
  - docs/api.md
  - docs/plugin-contract.md
  - .gitignore
  - go.mod
  - go.sum
  - go.work
  - go.work.sum
  - internal/audit/outbound_hosts_test.go
  - kernel/config/config.go
  - kernel/config/config_test.go
  - kernel/config/types.go
  - kernel/correlate/correlate.go
  - kernel/correlate/correlate_test.go
  - kernel/httpapi/agent.go
  - kernel/httpapi/agent_test.go
  - kernel/httpapi/contract_test.go
  - kernel/httpapi/item.go
  - kernel/httpapi/item_test.go
  - kernel/httpapi/routes.go
  - kernel/httpapi/sources.go
  - kernel/httpapi/sources_test.go
  - kernel/httpapi/stream.go
  - kernel/httpapi/stream_test.go
  - kernel/httpapi/webspaces.go
  - kernel/index/store.go
  - kernel/index/store_test.go
  - kernel/pluginhost/host.go
  - kernel/syncer/coordinator.go
  - kernel/syncer/coordinator_test.go
  - kernel/syncer/scheduler.go
  - kernel/syncer/scheduler_test.go
  - Makefile
  - plugins/mock/go.mod
  - plugins/mock/main.go
  - plugins/mock/plugin.go
  - plugins/mock/plugin_test.go
  - plugins/silverbullet/client.go
  - plugins/silverbullet/client_test.go
  - plugins/silverbullet/fetch_test.go
  - plugins/silverbullet/frontmatter.go
  - plugins/silverbullet/frontmatter_test.go
  - plugins/silverbullet/go.mod
  - plugins/silverbullet/go.sum
  - plugins/silverbullet/main.go
  - plugins/silverbullet/outbound_hosts_test.go
  - plugins/silverbullet/plugin.go
  - plugins/silverbullet/render.go
  - plugins/silverbullet/render_test.go
  - web/src/app.css
  - web/src/lib/api.ts
  - web/src/lib/components/DetailPane.svelte
  - web/src/lib/components/OpenInSource.svelte
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
  warning: 4
  info: 3
  total: 8
status: issues_found
---

# Phase 02: Code Review Report

**Reviewed:** 2026-07-29T00:00:00Z
**Depth:** standard
**Files Reviewed:** 47
**Status:** issues_found

## Summary

This phase adds the SilverBullet source plugin, promotes sync identity from
"webspace" to "(webspace, source_type)", adds the `/agent/v1` grant-filtered
namespace, and adds source-health/filter UI. The kernel-side sync
architecture (source-scoped `ReplaceWebspaceSourceItems`, the two-phase
`sync_runs` write, the `singleflight`-coordinated `Coordinator`, the
default-deny agent grant model) is well tested and internally consistent —
the partial-source-failure regression this phase was explicitly built to
fix is genuinely fixed and well covered by tests.

However, one genuine BLOCKER was found in the SilverBullet plugin's own
`Match` implementation: it silently swallows *every* per-page read failure
— including a systemic outage or auth failure partway through a sync — and
treats it identically to "page not found," which contradicts the plugin
contract's own explicit requirement ("return `codes.Unavailable`... not a
partial, silently-empty result... when the source system cannot be
reached") and can silently wipe a webspace's previously-synced SilverBullet
items. The dead `g.Wait()` error-check immediately below it shows this was
an implementation oversight, not a deliberate design choice — that branch
can never fire given how the goroutine body is written.

Several warnings were found in the frontend and the HTTP layer: a
date-formatting inconsistency between `DetailPane.svelte` and the rest of
the UI that reintroduces a timezone bug the codebase's own comments say was
deliberately fixed elsewhere; a documented-but-absent `iframe sandbox`
attribute; and inconsistent silent-error-swallowing for sync-status lookups
across handlers in the same package. None of these rise to data loss or a
security bypass on their own, but they are real, provable defects.

## Critical Issues

### CR-01: SilverBullet `Match` silently treats every page-read failure as "no match," violating the contract's own Unavailable-on-outage requirement and risking silent data loss

**File:** `plugins/silverbullet/plugin.go:94-120`

**Issue:** Inside `Match`'s bounded worker pool, every `ReadFile` failure —
regardless of cause — is swallowed and treated as "this page just doesn't
match":

```go
g.Go(func() error {
    raw, err := p.client.ReadFile(gctx, f.Name)
    if err != nil {
        // A single unreadable page (e.g. deleted between listing
        // and read, or a transient error) must not fail the whole
        // sync — it simply never matches, same as a page with no
        // matching tag/name would.
        return nil
    }
    ...
    return nil
})
```

The comment's justification ("deleted between listing and read") only
covers one failure mode. In practice `ReadFile` also fails for a dropped
connection, a revoked/expired bearer token, a TLS failure, or the
SilverBullet instance going down entirely *after* `ListFiles` already
succeeded. All of these are indistinguishable here from "page not present"
and are silently discarded — the goroutine always returns `nil`.

This directly contradicts `docs/plugin-contract.md`'s Match section:

> return a gRPC `codes.Unavailable` status (not a partial, silently-empty
> result) when the source system cannot be reached — the kernel records
> this per-source in that sync run's status... rather than treating "the
> source is down" the same as "nothing matched."

Concretely: if every page read fails after a successful listing (e.g. the
instance drops mid-sync, or the token is revoked), `Match` returns a
*successful* `MatchResponse` with zero items. `correlate.SyncSource` then
calls `Store.ReplaceWebspaceSourceItems` with an empty item slice, which
**deletes every previously-synced SilverBullet row for every webspace** and
records the sync as `"ok"` with `item_count: 0` — not as an error. This is
exactly the kind of silent, unrecorded failure the rest of this phase (the
source-scoped `ReplaceWebspaceSourceItems`, the two-phase `sync_runs`
write, the aggregate `sync.status` in the HTTP API) was built to prevent,
reintroduced one layer down inside this specific plugin.

The dead code immediately below confirms this is an oversight rather than
an intentional simplification — `g.Wait()` can never return a non-nil
error given the goroutine body above always returns `nil`, so this branch
is unreachable:

```go
if err := g.Wait(); err != nil {
    return nil, status.Errorf(codes.Unavailable, "silverbullet: match: %v", err)
}
```

There is no test in `plugins/silverbullet/` covering `Match`'s behavior
under partial or total read failure (the existing suite covers `client.go`,
`frontmatter.go`, `render.go` and `Fetch`, but not `Match`'s own
error-aggregation logic) — this gap in coverage is presumably why the bug
shipped.

**Fix:** Distinguish "page genuinely not found" (safe to skip) from a
transport/auth failure (must fail the sync). `ReadFile` already
distinguishes these via `ErrNotFound` vs. any other error — propagate the
latter:

```go
g.Go(func() error {
    raw, err := p.client.ReadFile(gctx, f.Name)
    if err != nil {
        if errors.Is(err, ErrNotFound) {
            // Deleted between listing and read — not a sync failure.
            return nil
        }
        return err // transport/auth/etc — must fail the whole Match.
    }
    ...
    return nil
})
...
if err := g.Wait(); err != nil {
    return nil, status.Errorf(codes.Unavailable, "silverbullet: match: %v", err)
}
```

Add a test that makes every page read (after a successful listing) fail
with a non-404 error and asserts `Match` returns `codes.Unavailable`
instead of an empty, successful `MatchResponse`.

## Warnings

### WR-01: `DetailPane.svelte`'s own date formatter reintroduces the timezone bug `format.ts` was built to fix

**File:** `web/src/lib/components/DetailPane.svelte:33-40` (used at line 72)

**Issue:** `web/src/lib/format.ts` implements `formatItemDate` specifically
to pin date formatting to UTC, with an extensive comment (and a dedicated
regression test in `format.test.ts`) explaining why: paperless-ngx's
`created` field is a date-only value at midnight UTC, and formatting it in
the browser's local timezone would show the wrong calendar day for any
user west of UTC. `StreamRow.svelte` correctly uses this shared helper
(`import { formatItemDate } from '$lib/format'`).

`DetailPane.svelte`, however, defines and uses its own local `formatDate`
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
a user in `America/Los_Angeles` would now see the stream row (using
`formatItemDate`) and the detail pane (using this local `formatDate`)
disagree on the calendar day for the same item.

**Fix:** Delete the local `formatDate` and import `formatItemDate` from
`$lib/format` instead, exactly as `StreamRow.svelte` does:

```ts
import { formatItemDate, detailPaneState } from '$lib/format';
// ...
<span>{formatItemDate(item.timestamp_unix)}</span>
```

### WR-02: Detail-pane iframes lack the `sandbox` attribute the codebase's own security comments claim exists

**File:** `web/src/lib/components/DetailPane.svelte:143,149`; referenced from `kernel/httpapi/item.go:190-196`

**Issue:** `kernel/httpapi/item.go`'s `renditionHandler` comment describes
two independent layers blocking script execution in a rendered
`text/html` document: the `Content-Security-Policy: ...; sandbox` response
header, **and** "the embedding `<iframe>`'s own sandbox attribute in
DetailPane.svelte". The actual iframe elements in `DetailPane.svelte`,
however, carry no `sandbox` attribute at all:

```svelte
<iframe title={item.title} src={contentUrl(item.id)} class="h-full w-full"></iframe>
```
(both the `text/html` branch and the PDF-rendition branch)

The CSP `sandbox` directive on the HTTP response does apply sandboxing to
the framed document regardless of the parent `<iframe>` element's own
attribute, so this is not currently an exploitable bypass — but the
comment in `item.go` asserts a second, independent defense-in-depth layer
that does not exist in the code. This is exactly the kind of mismatch that
causes a future change (e.g. someone widening the CSP for a legitimate
reason) to silently remove *all* iframe sandboxing rather than the
"redundant" layer the comment implies still remains.

**Fix:** Either add `sandbox=""` (or an explicit minimal token list, e.g.
`sandbox="allow-same-origin"` only if strictly required) to both iframe
elements in `DetailPane.svelte` to match what the comment describes, or
correct the comment in `kernel/httpapi/item.go` to stop claiming a
client-side defense layer that isn't implemented.

### WR-03: Sync-run lookup errors are silently swallowed in three of four handlers, masking a real store failure as "never synced"

**File:** `kernel/httpapi/stream.go:79-81`, `kernel/httpapi/webspaces.go:36-39`, `kernel/httpapi/agent.go:232-234`

**Issue:** `StreamHandler`, `WebspacesHandler`, and `agentStreamHandler`
all call `store.LatestSyncRunPerSource(ctx)` and, if it errors, silently
fall through with a zero-value `syncStatus` rather than surfacing the
error:

```go
// stream.go
if runs, err := store.LatestSyncRunPerSource(ctx); err == nil {
    resp.Sync = aggregateSyncStatus(runs)
}
```

```go
// webspaces.go
var lastSync syncStatus
if runs, err := store.LatestSyncRunPerSource(ctx); err == nil {
    lastSync = aggregateSyncStatus(runs)
}
```

```go
// agent.go, agentStreamHandler
if runs, err := store.LatestSyncRunPerSource(ctx); err == nil {
    resp.Sync = aggregateSyncStatus(filterRunsByGrant(runs, granted))
}
```

A genuine SQLite failure here (disk I/O error, corrupted index, etc.) is
indistinguishable from "no source has ever synced" to the client — the
response is still `200` with `sync.status: ""`, the documented "unknown"
neutral state, rather than a `500 internal_error`. This is inconsistent
with the sibling handler in the same file, `agentWebspacesHandler`, which
correctly propagates this exact error:

```go
// agent.go, agentWebspacesHandler
runs, err := store.LatestSyncRunPerSource(ctx)
if err != nil {
    WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
    return
}
```

**Fix:** Make the three swallowing call sites match `agentWebspacesHandler`'s
pattern — return `500 internal_error` on a genuine store failure instead of
silently downgrading it to the neutral "never synced" UI state.

### WR-04: No OS signal handling for graceful shutdown in `cmd/webspaces/main.go`

**File:** `cmd/webspaces/main.go:155-185` (`runServe`)

**Issue:** `runServe` wires up `defer host.Shutdown()`, `defer
store.Close()`, and `defer cancel()` (to stop the scheduler's goroutines),
but nothing in `main.go` ever installs a `signal.Notify`/`os.Signal`
handler. `http.ListenAndServe` only returns (letting those deferred calls
run) on a listener-level error (e.g. the port is already in use) — a
normal `Ctrl+C` (`SIGINT`) or `SIGTERM` from a process manager terminates
the Go process immediately via the runtime's default signal disposition,
without running any deferred function. In practice this means:

- The scheduler's background goroutines are killed abruptly rather than
  via `ctx.Done()`.
- `store.Close()` (which checkpoints the SQLite WAL) never runs on a normal
  shutdown.
- `host.Shutdown()` (`p.Kill()` on every launched plugin) never runs on the
  kernel's own shutdown path — plugin subprocess cleanup ends up relying
  entirely on `hashicorp/go-plugin`'s own dead-parent detection inside each
  plugin subprocess, rather than an explicit, logged shutdown from the
  kernel side.

**Fix:** Install a signal handler in `runServe` that cancels `ctx` (and
therefore triggers the existing deferred cleanup) on `SIGINT`/`SIGTERM`:

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
defer stop()
```

and use that `ctx` for both the scheduler and `http.Server.Shutdown(ctx)`
(replacing the bare `http.ListenAndServe` call with an `http.Server` whose
`Shutdown` is called on signal) so a normal `Ctrl+C` exercises the same
clean-shutdown path the deferred calls already assume exists.

## Info

### IN-01: `Config.Validate` requires `base_url`/`token` for every source, even a genuinely configless one

**File:** `kernel/config/config.go:182-188`

**Issue:** `Validate` unconditionally rejects a source config with an
empty `base_url` or `token`, regardless of whether that source's plugin
actually needs either. `config.example.toml`'s own commented-out
`[sources.mock]` block documents this as a known, unaddressed gap: an
operator wanting to run the reference `mock` plugin (which explicitly reads
`WEBSPACES_SOURCE_CONFIG` only if present, and needs neither key) must set
two meaningless placeholder values (`base_url = "unused"`, `token =
"unused"`) purely to satisfy this kernel-level constraint. This phase adds
a second real, configured-required source (SilverBullet) but does not
revisit this constraint despite the mock plugin (`PLUG-05`, this phase's
own reference-plugin deliverable) being the concrete counter-example that
needs it relaxed.

**Fix:** Either make `base_url`/`token` validation opt-in per plugin (e.g.
a small `RequiresConnectionConfig bool` a plugin's own `Describe`/config
convention could signal), or explicitly document in `config.example.toml`
that this is deferred to a specific future phase rather than "pre-existing"
indefinitely.

### IN-02: `/agent/v1` grant filtering is keyed by `source_type`, not the config name that carries the grant

**File:** `kernel/httpapi/agent.go:40-54`

**Issue:** `grantedSourceTypes` builds its granted set by resolving each
`agent.read = true` config name to a `source_type` via
`prober.SourceTypesByName()`. Grant filtering downstream then checks
membership by `source_type` alone (`granted[it.SourceType]`), not by
config name. If two different `[sources.<name>]` entries happen to launch
plugins that report the *same* `Describe`-reported `source_type` (nothing
in the plugin contract or the kernel's `Discover` structurally prevents
this — `source_type` is entirely plugin-controlled), granting `agent.read
= true` on one of them would also expose the other's items through
`/agent/v1`, even if that second source's own `[sources.<name>.agent]`
block says `read = false`. This is a narrow, config-error-only scenario
(not reachable by an external attacker), but it's a real gap in the
per-source grant isolation this phase's `AGENT-01` design otherwise goes
to considerable lengths to guarantee.

**Fix:** Either document that `source_type` must be unique across
configured sources for the agent grant model's isolation to hold, or key
the granted set by config name and carry the config name (not just
`source_type`) through to the item/stream filtering paths.

### IN-03: Rendition-serving logic (CSP headers, allowlist check, streaming) is duplicated verbatim between `item.go` and `agent.go`

**File:** `kernel/httpapi/item.go:147-213`, `kernel/httpapi/agent.go:302-345`

**Issue:** `agentRenditionHandler` in `agent.go` is a near-line-for-line
copy of `renditionHandler` in `item.go` — same MIME allowlist check, same
five-header hardened response, same streaming logic — differing only in
the grant check and the not-found response shape. The code comment
explains this was a deliberate scope decision (`item.go` was out of this
plan's `files_modified` list), but it does mean the security-relevant CSP
header string (`"default-src 'none'; style-src 'unsafe-inline'; object-src
'none'; sandbox"`) now exists as two independent literals that must be
kept in sync by hand; a future change to one (e.g. tightening the CSP) is
one easy oversight away from silently not applying to the other namespace.

**Fix:** Factor the shared header-writing/allowlist/streaming logic into
one unexported helper both `renditionHandler` and `agentRenditionHandler`
call, parameterized only by the not-found response and URL prefix, the
next time either file is touched.

---

_Reviewed: 2026-07-29T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
