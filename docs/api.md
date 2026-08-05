# The webspaces kernel HTTP API

This is the complete JSON contract the kernel serves over its loopback
HTTP listener. There are **two route namespaces sharing one JSON
contract**:

- **`/api/*`** — the human/UI-facing surface. The embedded web UI (the
  SvelteKit SPA) consumes exactly this JSON, over exactly these routes.
  It is **grant-free**: every route below documented under `/api/*`
  behaves identically regardless of any `[sources.<name>.agent]`
  configuration (`AGENT-02`).
- **`/agent/v1/*`** — a default-deny, per-source-grant-filtered mirror of
  `/api/*` for an automated caller (`AGENT-01`), documented in its own
  section below. Both namespaces share the identical envelope shape and
  the same `schema_version` counter — there is no second versioning
  scheme, only a narrower view over the same data for the agent surface.

Anything a human can see in the UI, a **granted** agent can retrieve from
the equivalent `/agent/v1/*` endpoint, in the identical structured shape.
An ungranted source is, by design, indistinguishable from a nonexistent
one when viewed through `/agent/v1/*` — see "The `/agent/v1` namespace",
below.

## Loopback-only default, no auth

By default the kernel binds `127.0.0.1:7777` (configurable via `[server]
listen` — see `config.example.toml`). There is **no authentication on
this API in v1, on either namespace**: the security boundary is "this API
is only reachable from processes already running on this machine, as this
user." Binding `listen` to a non-loopback address (e.g. `0.0.0.0` or a LAN
interface) is a deliberate security decision this project has **not**
made — the kernel logs a warning at startup if it detects a non-loopback
bind, but does not refuse to start. Don't expose this port to a LAN or the
internet without adding your own reverse proxy and auth in front of it.

**The `/agent/v1` per-source grant model (`AGENT-01`, below) is an
authorization control layered on top of this boundary, not an
authentication mechanism.** It answers "which sources may an already-local
caller read through this namespace", never "is this caller who it claims
to be" — any process on this machine can already reach `/api/*` (which is
grant-free by design) exactly as freely as `/agent/v1`. Don't mistake a
`[sources.<name>.agent]` grant for a security boundary against another
local process; it isn't one.

## Envelope convention

Every successful response is a JSON object with a top-level
`schema_version` field:

```json
{ "schema_version": 1, "...": "..." }
```

Bump `schema_version` only for a breaking shape change (never for an
additive field). Every error response, on every route, uses the identical
shared envelope:

```json
{ "schema_version": 1, "error": { "code": "item_not_found", "message": "item \"paperless:99999\" was not found in the index" } }
```

`error.code` is a fixed, machine-matchable `snake_case` string (see the
error-code table below); `error.message` is human-readable prose and may
change wording between releases — match on `code`, never on `message`.

## Routes

### `GET /api/webspaces`

Every configured webspace, its keyword list, its current indexed item
count, and the kernel's most recent sync status.

```
$ curl -s http://127.0.0.1:7777/api/webspaces | jq
{
  "schema_version": 1,
  "webspaces": [
    {
      "name": "house-move",
      "keywords": ["house and home"],
      "item_count": 35,
      "last_sync": { "status": "ok", "finished_unix": 1785000000, "error": "" }
    }
  ]
}
```

### `GET /api/webspaces/{webspace}/stream`

Every item correlated into `{webspace}`, in the kernel's total
chronological order (see Ordering, below). This is a pure index read — it
never triggers a live plugin call, so it's always fast regardless of
source-system latency.

```
$ curl -s http://127.0.0.1:7777/api/webspaces/house-move/stream | jq
{
  "schema_version": 1,
  "webspace": "house-move",
  "sync": { "status": "ok", "finished_unix": 1785000000, "error": "" },
  "items": [
    {
      "id": "paperless:528",
      "source_type": "paperless",
      "source_id": "528",
      "title": "Completion statement",
      "preview": "This letter confirms completion has taken place...",
      "timestamp_unix": 1784800000,
      "secondary_timestamp_unix": 1784801234,
      "labels": ["house and home"],
      "group_id": "",
      "group_label": "",
      "link": { "url": "https://paperless.example.lan/documents/528", "fidelity": "exact" },
      "thumbnail_url": "/api/items/paperless:528/thumbnail",
      "provenance": {
        "source_type": "paperless",
        "source_system": "https://paperless.example.lan",
        "source_id": "528",
        "plugin": "webspaces-plugin-paperless",
        "contract_version": "topos.v1",
        "synced_at_unix": "1785000000"
      }
    }
  ]
}
```

A **known** webspace (one that has completed at least one sync, even a
zero-item sync) with no matched items returns `200` and `"items": []` —
never `404` and never a JSON `null` for `items`. An **unconfigured or
never-synced** webspace returns `404 webspace_not_found` — this is the
one place `404` and "empty" mean genuinely different things, and this API
never conflates them.

**The `sync` object is an aggregate across every configured source**
(`KERN-04`), not a single most-recent-run — this is a behavior change
from Phase 1, where it mirrored the single most recently recorded run
across the whole kernel. `status` is `"error"` if *any* configured
source's latest run errored, else `"running"` if any source is still
mid-sync, else `"ok"` if at least one run has ever completed, else the
zero value (`""`) if nothing has ever synced. `finished_unix` is the
newest `finished_unix` across every source's latest run. `error` joins
each failing source's message, prefixed with its source type, in sorted
source order (so it's deterministic) — e.g. `"silverbullet: dial tcp:
connection refused"`. This is what stops a two-source webspace whose only
failing source returned nothing from rendering as merely empty: before
this aggregate, a webspace with one healthy source and one silently
broken one could report `sync.status: "ok"` just because the *other*
source's run happened to be the most recent one recorded anywhere in the
kernel. `GET /api/webspaces`'s `last_sync` field uses the identical
aggregate.

### `GET /api/webspaces/{webspace}/search`

Full-text search over every item correlated into `{webspace}` (`KERN-05`).
Like the stream route, this is a pure index read over the same
`items`/`items_fts` local index — it never triggers a live plugin call, so
it's always fast regardless of source-system latency.

```
$ curl -s "http://127.0.0.1:7777/api/webspaces/house-move/search?q=boiler" | jq
{
  "schema_version": 1,
  "webspace": "house-move",
  "query": "boiler",
  "results": [
    {
      "id": "paperless:528",
      "source_type": "paperless",
      "source_id": "528",
      "title": "Boiler service invoice",
      "preview": "annual boiler service",
      "timestamp_unix": 1784800000,
      "secondary_timestamp_unix": 1784801234,
      "labels": ["house and home"],
      "group_id": "",
      "group_label": "",
      "link": { "url": "https://paperless.example.lan/documents/528", "fidelity": "exact" },
      "thumbnail_url": "/api/items/paperless:528/thumbnail",
      "provenance": { "...": "identical shape to a stream row" },
      "snippet": "annual boiler service"
    }
  ]
}
```

**Query parameter:** `q`. A missing, empty, or whitespace-only `q` returns
`200` with `"query": ""` and `"results": []` — never an error, and the
store is not even queried in this case.

**Query syntax:** the raw `q` text is never handed to the underlying FTS5
`MATCH` operator directly. It's split into whitespace-delimited terms,
each wrapped as a literal phrase and combined with an implicit AND, with
the final term prefix-matched — so a lone double-quote, a leading hyphen,
or any other FTS5-query-syntax character in `q` can never produce a
search-syntax error. A `q` that would be malformed FTS5 syntax degrades to
`200` with an empty `results` array, exactly like a `q` that legitimately
matches nothing — this route never returns a `500` because of what's
typed into a search box.

**Result shape:** every field a stream row carries (see `GET
/api/webspaces/{webspace}/stream`, above), flattened, plus one additional
field:

- `snippet` — a short excerpt around the matched term, with the matched
  term(s) wrapped between two ASCII control characters: **STX** (hex `02`,
  JSON-escaped as `\u0002`) opening and **ETX** (hex `03`, JSON-escaped as
  `\u0003`) closing. These characters cannot occur in real subject lines
  or preview text, so a consumer can split `snippet` on them to apply its
  own highlighting without ever mistaking a delimiter for source content.
  Elided text within the snippet is marked with an ellipsis (`…`).

**Result cap:** results are capped at **50** rows. There is no pagination
in this version — a query matching more than 50 items simply returns the
top 50 by rank.

**Ordering:** results are ordered by **bm25 relevance rank, best match
first** — this is a *different* ordering guarantee from the stream
route's chronological order (see "Ordering guarantee", below, for the
explicit cross-reference).

**Webspace scoping:** results are drawn only from items associated with
`{webspace}` — an item indexed only into a different webspace is never
returned, no matter how well it matches `q`.

**Unknown webspace:** `GET /api/webspaces/{unknown}/search?q=...` returns
`404 webspace_not_found`, identical to the stream route's behavior for the
same unknown name.

This route has **no `/agent/v1` mirror** in this version — see "The
`/agent/v1` namespace", below, for why.

### `GET /api/items/{id}`

`{id}` is the stable composite id (`{source_type}:{source_id}`, e.g.
`paperless:528`), accepted both raw and percent-encoded
(`paperless%3A528`) — both resolve to the same item. Returns the item's
indexed metadata (identical shape to a stream row) plus **one live
`Fetch(FULL)` call** to the owning plugin for extracted text and a
rendition descriptor. This is the one route family with request-time
source-system latency; everything else on this API is an index read.

```
$ curl -s http://127.0.0.1:7777/api/items/paperless:528 | jq
{
  "schema_version": 1,
  "item": { "...": "same shape as a stream row, above" },
  "content": {
    "available": true,
    "unavailable_reason": "",
    "text": "This letter confirms completion has taken place on the sale of...",
    "rendition": { "mime_type": "application/pdf", "size_bytes": 214532, "url": "/api/items/paperless:528/content" }
  }
}
```

`content.available: false` (with `content.rendition: null`) is a normal,
`200`-status outcome — e.g. the source has no previewable rendition for
this item's file type — not an error. A `200` with `content.available:
false` still carries `content.text` if the plugin had extracted text
independent of the rendition.

### `GET /api/items/{id}/content` and `GET /api/items/{id}/thumbnail`

The raw bytes of the preview and thumbnail renditions, respectively,
streamed straight through from the plugin's live `Fetch` call. Both routes
enforce a fixed MIME allowlist (`application/pdf`, `image/png`,
`image/jpeg`, `image/gif`, `image/webp`, `text/html`) before writing any
byte, and set a hardened header set on every accepted response:

```
Content-Type: <allowlisted MIME type>
X-Content-Type-Options: nosniff
Content-Disposition: inline
Content-Security-Policy: default-src 'none'; style-src 'unsafe-inline'; object-src 'none'; sandbox
Cache-Control: private, no-store
```

A plugin-reported MIME type outside the allowlist never reaches a response
header — it's rejected with `415 unsupported_rendition_type` instead. This
matters because these routes serve source-controlled bytes (a paperless-ngx
user's own uploaded PDF, for instance) from the kernel's own origin; the
sandboxing CSP and `nosniff` header are what keep an embedded/rendered
document from executing as if it were same-origin content.

`text/html` renditions (currently produced only by the SilverBullet
plugin, rendering a wiki page's markdown) are sanitized by the *producing
plugin* — via `goldmark` (safe-by-default HTML/URL-scheme rendering) and a
`bluemonday.UGCPolicy()` pass — before the bytes ever reach the kernel.
The kernel does not re-sanitize; it relies on the same allowlist-plus-CSP
mechanism above (in particular the `sandbox` CSP directive, which strips
script execution, form submission, and top-level navigation from the
iframe an embedding client renders this content inside) as the second,
independent layer of defense.

`style-src 'unsafe-inline'` exists specifically so a `text/html`
rendition's own inline `<style>` block is actually applied inside the
embedding iframe — `default-src 'none'` with no `style-src` override
otherwise blocks a document from styling itself at all, which shipped as
a live bug (unstyled rendered markdown read as near-black text against
the app's dark theme) before this directive was added. This does not
weaken script blocking: `default-src 'none'` (with no `script-src`
override) and the `sandbox` directive together still deny all script
execution regardless of `style-src`. It's safe specifically because the
only inline style any rendition document can ever carry is a fixed
string the *producing plugin's own Go source* injects strictly after its
own sanitization pass — SilverBullet's `bluemonday.UGCPolicy()` strips
any `<style>` element or `style` attribute that originated from page
content before that trusted stylesheet is ever appended, so a hostile or
malformed source document cannot smuggle a stylesheet through this
directive.

### `GET /api/sources`

One entry per configured source: its config name, plugin-reported
`source_type` and `display_name`, a **live** reachability probe result,
whether it is currently syncing, and the kernel's own recorded sync
history for it (`PLUG-04`). Sorted by name.

```
$ curl -s http://127.0.0.1:7777/api/sources | jq
{
  "schema_version": 1,
  "sources": [
    {
      "name": "paperless",
      "source_type": "paperless",
      "display_name": "paperless-ngx",
      "reachable": true,
      "syncing": false,
      "last_status": "ok",
      "last_sync_unix": 1785000000,
      "last_error": ""
    },
    {
      "name": "silverbullet",
      "source_type": "silverbullet",
      "display_name": "SilverBullet",
      "reachable": false,
      "syncing": false,
      "last_status": "error",
      "last_sync_unix": 1784900000,
      "last_error": "dial tcp: connection refused"
    }
  ]
}
```

`reachable` is a **live** `Health` RPC probe made at request time, not a
cached value — a source can flip from reachable to unreachable between
two calls with no sync in between. `last_status`, `last_sync_unix` and
`last_error`, by contrast, come exclusively from the kernel's own
recorded sync history — a plugin's self-reported last-sync time and last
error are never trusted for these fields, so a plugin cannot report a
rosier history than the kernel actually recorded and turn its own health
chip green. `last_status: ""` (with `last_sync_unix: 0` and `last_error:
""`) is the neutral "unknown" state for a source that has never completed
a sync — render this as a neutral indicator, never as a green "ok". One
plugin's probe failing never fails the whole response: it becomes that
source's own `reachable: false`, never a `500`.

### `POST /api/sources/{name}/refresh` and `POST /api/sync`

Trigger a manual sync of one configured source, or every configured
source, through the exact same coordinator entry point the background
scheduler uses (`KERN-04`) — a manual refresh, a scheduled tick, and the
`webspaces sync` CLI command all dedupe against each other via the same
single-flight guarantee. A refresh request for a source that is already
syncing **coalesces** into that in-flight run and reports its outcome; it
is never queued behind it and never starts a second concurrent sync for
that source.

```
$ curl -s -X POST http://127.0.0.1:7777/api/sources/silverbullet/refresh | jq
{
  "schema_version": 1,
  "source": {
    "name": "silverbullet",
    "source_type": "silverbullet",
    "status": "ok",
    "item_count": 17,
    "error": "",
    "coalesced": false,
    "finished_unix": 1785000500
  }
}

$ curl -s -X POST http://127.0.0.1:7777/api/sync | jq
{
  "schema_version": 1,
  "sources": [
    { "name": "paperless", "source_type": "paperless", "status": "ok", "item_count": 35, "error": "", "coalesced": false, "finished_unix": 1785000500 },
    { "name": "silverbullet", "source_type": "silverbullet", "status": "ok", "item_count": 17, "error": "", "coalesced": true, "finished_unix": 1785000500 }
  ]
}
```

`POST /api/sources/{name}/refresh` with an unconfigured `{name}` returns
`404 source_not_found` in the standard error envelope — `{name}` is
validated against the configured source set *before* any dispatch, and
the error message names only the value you sent, never which source
names actually exist (this route cannot be used to enumerate configured
sources). `coalesced: true` means this call joined an already-in-flight
sync for that source rather than triggering a fresh one — the reported
outcome is still that run's real result, never a distinct
"already-syncing" rejection.

## The `/agent/v1` namespace (`AGENT-01`)

`/agent/v1/*` mirrors the `/api/*` routes above under **default-deny,
per-source grants** declared in config:

```toml
[sources.silverbullet.agent]
read = true      # this source's items are visible through /agent/v1
handoff = false  # this source's action hand-off capability (metadata only — see below)
```

An absent `[sources.<name>.agent]` block, an absent `read` key, and an
explicit `read = false` are **all identically "deny"** — there is no
`default`/`enabled` key that widens this; the absence of a grant *is* the
deny. `read` and `handoff` are independent: a source with `handoff = true`
and `read = false` is still fully absent from every `/agent/v1/*`
response.

| Route | Mirrors | Restriction |
|---|---|---|
| `GET /agent/v1/sources` | `GET /api/sources` | Ungranted sources are omitted entirely; each entry gains a `capabilities: {read, handoff}` object. |
| `GET /agent/v1/webspaces` | `GET /api/webspaces` | `item_count` and `last_sync` are computed over granted sources only. |
| `GET /agent/v1/webspaces/{webspace}/stream` | `GET /api/webspaces/{webspace}/stream` | `items` and `sync` are restricted to granted sources; ordering is otherwise identical to the `/api` stream with ungranted rows removed, never reordered. |
| `GET /agent/v1/items/{id}` | `GET /api/items/{id}` | An ungranted source's item responds identically to a nonexistent id (see below). |
| `GET /agent/v1/items/{id}/content` | `GET /api/items/{id}/content` | Same restriction as above; no rendition bytes are written for an ungranted item. |
| `GET /agent/v1/items/{id}/thumbnail` | `GET /api/items/{id}/thumbnail` | Same restriction as above. |

```
$ curl -s http://127.0.0.1:7777/agent/v1/sources | jq
{
  "schema_version": 1,
  "sources": [
    {
      "name": "silverbullet",
      "source_type": "silverbullet",
      "display_name": "SilverBullet",
      "reachable": true,
      "syncing": false,
      "last_status": "ok",
      "last_sync_unix": 1785000000,
      "last_error": "",
      "capabilities": { "read": true, "handoff": false }
    }
  ]
}
```

**An ungranted source is indistinguishable from a nonexistent one, by
design (`T-02-20`).** `GET /agent/v1/items/{id}` for an item belonging to
an ungranted source returns a response byte-identical in status code,
error code and message wording to the response for an `{id}` that is not
in the index at all — `404 item_not_found`, same message construction as
`/api/items/{id}`'s not-found case. There is **no distinct error code**
for "exists but ungranted" — a different response shape would be an
enumeration oracle, letting a caller infer which sources exist even
without read access to them, which is exactly what this design prevents.
`GET /agent/v1/webspaces/{webspace}/stream` for a **known** webspace with
zero granted items returns `200` with `"items": []`, never `404` — a
zero-grant config never turns a real webspace into a 404. Zero configured
grants overall means `GET /agent/v1/sources` returns `200` with
`"sources": []`, not an error.

`handoff` is published as **metadata only** in this version of the API —
the `capabilities.handoff` field on `GET /agent/v1/sources`. No route on
either namespace performs an action against a source in v1; agent-
initiated actions (`AGENT-11`, e.g. "draft an email reply") are a v1.x
concern layered on top of this permission model, not present here.

`GET /api/webspaces/{webspace}/search` has **no `/agent/v1` mirror** in
this version. An ungated agent-facing search route would let a caller
search across every source in a webspace regardless of that source's own
`read` grant, bypassing the per-source read grants this namespace exists
to enforce — so search stays `/api`-only until a grant-filtered version
of it is designed.

## The stable-ID scheme

Every item's `id` is `"{source_type}:{source_id}"` — `source_type` is
whatever the owning plugin reported via its `Describe` RPC (e.g.
`"paperless"`), and `source_id` is that plugin's own stable local
identifier for the object (e.g. `"528"`, a paperless-ngx document id).

- **Stable across syncs**: re-syncing never changes an existing item's id;
  the same source object always upserts to the same row.
- **Unique across source types**: two different sources can use
  overlapping `source_id` values (a paperless-ngx document `"1"` and an
  IMAP message `"1"`) without colliding, because the `source_type` prefix
  disambiguates them.
- **Round-trips through percent-encoding**: a client that URL-encodes the
  `:` (producing `paperless%3A528`) gets the identical item back as the
  raw form.

## Ordering guarantee

`GET /api/webspaces/{webspace}/stream` returns items in a **total, stable
order**: `timestamp_unix` descending, then `secondary_timestamp_unix`
descending, then `id` ascending as the final deterministic tie-break. This
order never depends on SQLite's underlying row order, and it never
changes between two calls with no intervening sync — the same request
against the same index state always returns byte-identical JSON.

`GET /api/webspaces/{webspace}/search` uses a **different** ordering
guarantee — bm25 relevance rank, best match first — not the chronological
order above. Don't assume search results share the stream's chronological
ordering; they don't.

## Provenance keys

Every item's `provenance` object carries exactly these six keys:

| Key | Meaning |
|---|---|
| `source_type` | Matches the item's own `source_type` / id prefix. |
| `source_system` | The specific source instance this item came from (e.g. a paperless-ngx base URL) — distinguishes "which paperless-ngx" if you ever configure more than one. |
| `source_id` | Matches the item's own `source_id`. |
| `plugin` | The plugin binary name that produced this item (e.g. `"webspaces-plugin-paperless"`). |
| `contract_version` | The plugin's own `Describe`-reported contract version (e.g. `"topos.v1"`). |
| `synced_at_unix` | When the kernel's index last wrote this row, as a Unix timestamp string — set by the kernel itself at read time, never by a plugin. |

## Error codes

The table below is written against the `/api/*` route names; every code
applies identically to that route's `/agent/v1/*` mirror (e.g.
`item_not_found` on `GET /api/items/{id}` also covers
`GET /agent/v1/items/{id}` — including the ungranted case, which reuses
this exact code rather than a distinct one; see "The `/agent/v1`
namespace", above).

| Code | HTTP status | Route(s) | Meaning |
|---|---|---|---|
| `webspace_not_found` | 404 | `GET /api/webspaces/{webspace}/stream`, `GET /api/webspaces/{webspace}/search` | `{webspace}` is not configured, or has never completed a sync. |
| `item_not_found` | 404 | `GET /api/items/{id}` and its `/content`, `/thumbnail` children | `{id}` does not exist in the local index (or, for `Fetch`-level failures, the plugin itself reports the source object no longer exists). On `/agent/v1/*`, also covers an `{id}` whose source exists but is ungranted — deliberately the same code as a genuinely nonexistent id. |
| `source_unavailable` | 502 | `GET /api/items/{id}` and its `/content`, `/thumbnail` children | The live `Fetch` call to the owning plugin failed — the source system was unreachable or errored. |
| `unsupported_rendition_type` | 415 | `GET /api/items/{id}/content`, `/thumbnail` | The plugin reported a rendition MIME type outside the fixed allowlist; the kernel refuses to serve it. |
| `content_unavailable` | 404 | `GET /api/items/{id}/content`, `/thumbnail` | The item exists and the plugin was reachable, but no rendition is available for this specific variant (distinct from `item_not_found`: the item is real, this rendition just doesn't exist). |
| `source_not_found` | 404 | `POST /api/sources/{name}/refresh` | `{name}` does not match any configured `[sources.<name>]` entry. The message never enumerates which names do exist. |
| `internal_error` | 500 | any route | An unexpected kernel-side failure (e.g. the local index file itself is unreadable) — not a source or plugin problem. |

## What is not here yet

This API is deliberately incomplete for v1. Not yet present, and not
planned for this phase:

- **Agent-initiated actions** (`AGENT-11`, e.g. "draft an email reply") —
  this API is read-only end to end for the whole of v1; action hand-off is
  a v1.x concern layered on top of the `/agent/v1` permission model
  documented above.
