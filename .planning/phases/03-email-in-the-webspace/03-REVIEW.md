---
phase: 03-email-in-the-webspace
reviewed: 2026-07-31T00:00:00Z
depth: standard
files_reviewed: 30
files_reviewed_list:
  - docs/api.md
  - kernel/config/types.go
  - kernel/httpapi/routes.go
  - kernel/httpapi/search.go
  - kernel/httpapi/search_test.go
  - kernel/index/schema.go
  - kernel/index/store.go
  - kernel/index/store_test.go
  - kernel/pluginhost/host.go
  - plugins/proton/body.go
  - plugins/proton/client.go
  - plugins/proton/go.mod
  - plugins/proton/go.sum
  - plugins/proton/imap_transcript_test.go
  - plugins/proton/item_test.go
  - plugins/proton/live_bridge_test.go
  - plugins/proton/main.go
  - plugins/proton/outbound_hosts_test.go
  - plugins/proton/plugin.go
  - plugins/proton/readonly_test.go
  - plugins/proton/render_test.go
  - web/src/lib/api.ts
  - web/src/lib/components/DetailPane.svelte
  - web/src/lib/components/SearchBox.svelte
  - web/src/lib/components/SearchResults.svelte
  - web/src/lib/components/StreamRow.svelte
  - web/src/lib/components/ui/input/index.ts
  - web/src/lib/components/ui/input/input.svelte
  - web/src/lib/components/WebspaceHeader.svelte
  - web/src/lib/format.test.ts
  - web/src/lib/format.ts
  - web/src/routes/w/[webspace]/+page.svelte
findings:
  critical: 1
  warning: 1
  info: 3
  total: 5
status: issues_found
---

# Phase 03: Code Review Report

**Reviewed:** 2026-07-31T00:00:00Z
**Depth:** standard
**Files Reviewed:** 30
**Status:** issues_found

## Summary

The bulk of this phase (search/FTS5 in `kernel/index`, the search HTTP route,
and the search UI) is careful, well-tested, and matches its own
documentation (`docs/api.md`) closely — parameterized SQL throughout,
correct bm25 ordering, defensive handling of malformed FTS5 input, and a
thorough `parseSnippet`/`searchVariant` test matrix on the frontend.

The Proton Mail plugin's read-only/TLS-pinning/sanitization controls
(`client.go`, `body.go`) are also solid and are backed by real wire-level
and sanitizer tests.

However, cross-referencing `plugins/proton/plugin.go` against its actual
caller (`kernel/correlate/correlate.go`) surfaces a critical, load-bearing
data-loss bug: the plugin's `mailboxCache` — the only mechanism `Fetch`
uses to resolve a `source_id` back to an IMAP mailbox — is fully replaced,
not merged, on every `Match` call, and `Match` is called once **per
configured webspace** within a single sync cycle for this source. Any
webspace other than the last one processed in that cycle loses the ability
to fetch its own emails' bodies until (by chance) it is again the
last-processed webspace. This will reproduce on any deployment with two or
more webspaces that both match Proton Mail items — which is the project's
core multi-webspace use case, not an edge case.

A smaller, real logic bug was also found in `DetailPane.svelte`, which
re-implements date formatting locally instead of using the already-tested,
UTC-pinned `formatItemDate` from `format.ts`, reintroducing the exact
"wrong calendar day in a negative-UTC-offset timezone" class of bug that
`format.ts`'s own doc comments and tests explicitly exist to prevent.

## Critical Issues

### CR-01: Proton plugin's mailbox cache is replaced (not merged) per `Match` call, breaking `Fetch` for every webspace but the last one synced

**File:** `plugins/proton/plugin.go:64-75, 141-148, 197-204, 381-385`

**Issue:**

`SourcePlugin.mailboxCache` is the *only* state `Fetch`/`fetchFull` uses to
resolve a `source_id` to the IMAP mailbox it must `EXAMINE`
(`plugins/proton/plugin.go:391-396`, used at `plugins/proton/plugin.go:424-427`).
It is populated by `setMailboxCache`, which unconditionally **overwrites**
the entire map:

```go
// plugins/proton/plugin.go:381-385
func (p *SourcePlugin) setMailboxCache(cache map[string]string) {
	p.mailboxMu.Lock()
	defer p.mailboxMu.Unlock()
	p.mailboxCache = cache
}
```

`Match` calls this once per invocation — both on the "no mailbox matched
this webspace's keywords" path:

```go
// plugins/proton/plugin.go:141-148
if len(matchedMailboxes) == 0 {
	// No mailbox leaf name matches this webspace's keywords: a
	// successful, empty sync — never an error, and never a wipe of a
	// sibling source's already-indexed rows for this webspace (the
	// caller, kernel/correlate, replaces only THIS source's rows).
	p.setMailboxCache(map[string]string{})
	return &webspacesv1.MatchResponse{}, nil
}
```

and on the success path:

```go
// plugins/proton/plugin.go:197-204
newCache := make(map[string]string, len(byMessageID))
items := make([]*webspacesv1.Item, 0, len(byMessageID))
for msgID, m := range byMessageID {
	sourceID := encodeSourceID(msgID)
	newCache[sourceID] = m.mailbox
	items = append(items, p.toItem(sourceID, m))
}
p.setMailboxCache(newCache)
```

The doc comment on the field (`plugins/proton/plugin.go:64-72`) reasons
this is safe because plugin subprocesses live for the kernel's whole
lifetime and a sync always runs before the UI can show an item — but that
reasoning only covers the "fresh process, before first sync" case. It does
not account for the actual call pattern: `kernel/correlate/correlate.go`'s
`SyncSource` calls this source's `Match` **once per configured webspace**,
sequentially, within a single sync cycle:

```go
// kernel/correlate/correlate.go:77-113 (excerpt)
func (e *Engine) SyncSource(ctx context.Context, src Source) (results []WebspaceResult, rejections string) {
	...
	for name, ws := range e.Config.Webspaces {
		resp, err := src.Match(ctx, ws.Keywords)
		...
	}
	...
}
```

Because `Match` for webspace B's keywords only re-discovers messages that
live in mailboxes matching webspace B's keywords, its `newCache` never
contains entries for messages that only matched webspace A's keywords.
`setMailboxCache(newCache)` then **replaces** the whole map, silently
dropping every `source_id → mailbox` entry that webspace A's earlier
`Match` call in the same loop had just inserted.

The comment at lines 142-145 explicitly reasons about *not* wiping "a
sibling source's already-indexed rows for this webspace" (the
`webspace_items` index rows, correctly scoped per `(webspace, source_type)`
by `ReplaceWebspaceSourceItems`) — but this is a different invariant from
the one actually broken here: the in-memory `mailboxCache` is shared
across **every webspace this plugin instance serves**, and nothing scopes
it per webspace.

**Concrete impact:** with two or more webspaces configured that both match
Proton Mail messages (the project's stated multi-webspace use case), after
every sync cycle only the webspace that happened to be last in Go's
(randomized) `map[string]Webspace` iteration order has working `Fetch`
calls. Opening any item belonging to any other webspace in the detail pane
returns `codes.NotFound` from `fetchFull`
(`plugins/proton/plugin.go:424-427`) — surfaced to the user as `source_id
%q is not known — the index has not been synced since this plugin
started`, even though the index has been synced and the item is visible in
the stream. Because Go's map iteration order is randomized per range, which
webspace "wins" can even change between consecutive sync cycles.

**Fix:** merge into the cache instead of replacing it, and stop resetting
it to an empty map on the "no mailboxes matched" path (that path legitimately
has nothing new to contribute for *this* webspace, but must not erase what
other webspaces already contributed in the same sync cycle):

```go
// merge, don't replace
func (p *SourcePlugin) mergeMailboxCache(entries map[string]string) {
	p.mailboxMu.Lock()
	defer p.mailboxMu.Unlock()
	for id, mbox := range entries {
		p.mailboxCache[id] = mbox
	}
}
```

Call `p.mergeMailboxCache(newCache)` instead of `p.setMailboxCache(newCache)`
at line 204, and delete the `p.setMailboxCache(map[string]string{})` call at
line 146 entirely (a webspace matching zero mailboxes has nothing to merge
and must not touch entries owned by other webspaces). If stale entries
(messages that no longer match any webspace) are a concern, expire them
based on a full-cycle "seen this sync round" set rather than clearing
per-webspace, since `Fetch`'s own UID re-resolution already handles a
truly-deleted message safely.

## Warnings

### WR-01: `DetailPane.svelte` re-implements date formatting without pinning UTC, reintroducing the timezone off-by-one-day bug `format.ts` exists to prevent

**File:** `web/src/lib/components/DetailPane.svelte:33-40` (used at line 77)

**Issue:** `web/src/lib/format.ts` deliberately pins `formatItemDate` to
`timeZone: 'UTC'`, with an explicit doc comment and a dedicated test
(`web/src/lib/format.test.ts:37-56`) proving that without this pin, a
viewer in a negative-UTC-offset timezone (e.g. `America/Los_Angeles`) would
see the previous calendar day for a date-only source timestamp.
`DetailPane.svelte` does not use that shared, tested function — it defines
its own local `formatDate` that omits the UTC pin entirely:

```svelte
<!-- web/src/lib/components/DetailPane.svelte:33-40 -->
function formatDate(unix: number): string {
	if (!unix) return '';
	return new Date(unix * 1000).toLocaleDateString(undefined, {
		year: 'numeric',
		month: 'short',
		day: 'numeric'
	});
}
```

This means the same item can show two different dates in the same UI: the
stream row (`StreamRow.svelte:83`, via the shared `formatItemDate`) renders
the correct UTC calendar day, while the detail pane header
(`DetailPane.svelte:77`) can render the day before it, for any viewer west
of UTC, for date-only timestamps such as paperless-ngx's `created` field.

**Fix:** delete the local `formatDate` and import/use `formatItemDate` from
`$lib/format`, exactly as `StreamRow.svelte` already does:

```svelte
import { formatItemDate, detailPaneState } from '$lib/format';
...
<span>{formatItemDate(item.timestamp_unix)}</span>
```

## Info

### IN-01: Dead sentinel errors with misleading doc comments in the Proton plugin

**File:** `plugins/proton/client.go:22-28, 39-44`

**Issue:** `ErrNotFound` and `ErrNoMessageID` are exported, documented as
"returned when..." a specific condition occurs, but neither is ever
returned or wrapped anywhere in `plugins/proton/*.go` (grep confirms zero
non-declaration references). `plugin.go`'s actual no-Message-Id skip path
uses a plain counter (`skippedNoMessageID++`) and a log line instead of
`ErrNoMessageID`; `fetchFull`'s not-found paths use `status.Errorf(codes.NotFound, ...)`
directly instead of `ErrNotFound`. The doc comments describe behavior that
does not exist in the code, which will mislead a future reader.

**Fix:** either wire these sentinels into the paths their doc comments
describe, or remove them if the counter/log-line and direct
`status.Errorf` approach is the intended design.

### IN-02: `Snippet` helper in `body.go` is unused

**File:** `plugins/proton/body.go:112-121`

**Issue:** `Snippet` is exported and documented as mirroring the
SilverBullet plugin's preview-truncation helper, but nothing in
`plugins/proton` (production code or tests) calls it — `toItem` leaves
`Preview` permanently empty by design (`plugins/proton/plugin.go:356`), so
there is currently no caller for a preview-truncation helper.

**Fix:** remove it until there's an actual call site, or wire it in if a
future change starts populating `Preview`.

### IN-03: `getJSON`/`postJSON` in `api.ts` are near-duplicate implementations

**File:** `web/src/lib/api.ts:121-137, 145-161`

**Issue:** `getJSON` and `postJSON` are identical except for the `fetch`
call's method and the absence of a generic error path difference — the
entire error-envelope-parsing block (lines 124-134 and 148-158) is
duplicated verbatim. This is a maintainability risk: a future fix to the
error-handling logic (e.g. handling a new failure shape) has to be applied
in two places, and the two have already begun to drift in comments ("mirrors
getJSON's error-envelope handling exactly").

**Fix:** factor the shared logic into one helper, e.g.:

```ts
async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const res = await fetch(path, init);
	if (!res.ok) {
		let envelope: ApiErrorEnvelope | undefined;
		try {
			envelope = (await res.json()) as ApiErrorEnvelope;
		} catch {
			/* not the error envelope */
		}
		if (envelope?.error) throw new ApiError(envelope.error.code, envelope.error.message);
		throw new ApiError('http_error', `Request to ${path} failed with status ${res.status}`);
	}
	return (await res.json()) as T;
}
const getJSON = <T>(path: string) => request<T>(path);
const postJSON = <T>(path: string) => request<T>(path, { method: 'POST' });
```

---

_Reviewed: 2026-07-31T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
