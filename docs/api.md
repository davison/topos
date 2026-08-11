# The topos kernel HTTP API

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
      "source": "paperless",
      "source_type": "paperless",
      "source_display_name": "paperless-ngx",
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
        "plugin": "topos-plugin-paperless",
        "contract_version": "topos.v1",
        "synced_at_unix": "1785000000"
      }
    }
  ]
}
```

**`source` is the source INSTANCE id** — the `[sources.<id>]` config map
key this item was synced through (`D-08`). It is the identity key
everywhere identity matters: it prefixes every item `id` (see "The
stable-ID scheme", below), keys every sync-run record, and is what
`/agent/v1` grants are checked against. **`source_type`** is retained
unchanged alongside it as the *plugin kind* the owning plugin reported via
its `Describe` RPC — purely descriptive provenance, never an identity key.
Two source instances configured against the same plugin binary (e.g. two
Proton accounts, "home-email" and "work-email") always share one
`source_type` but always have distinct `source` values, distinct item id
namespaces, and distinct sync history — they never merge (`D-10`).
**`source_display_name`** is that instance's resolved, operator-authored
label: its configured `[sources.<id>] display_name`, or the instance id
itself when that key is omitted (`D-09`) — the kernel never emits an empty
`source_display_name`.

A webspace is **known** — and this route (and `GET /agent/v1/webspaces/
{webspace}/stream`) answers `200`, with `"items": []` when nothing has
matched yet — the instant it is named in the running configuration (the
`[webspaces.<name>]` block a `PUT /api/config` save just wrote), with no
dependency on whether or when a sync has actually run against it. A
webspace present only in the local index (its `[webspaces.*]` block was
later removed from the config while previously-synced rows survive) is
also known, by the same rule. Only a name in **neither** the running
config **nor** the index returns `404 webspace_not_found` — this is the
one place `404` and "empty" mean genuinely different things, and this API
never conflates them. Before `07-15-PLAN.md` this route answered from
sync history alone, which meant a webspace just created through the UI
(config-known, index-unknown until its first sync completed) could 404
transiently, and — on an install with zero configured sources at all,
where no sync ever runs — permanently; that gap is closed.

**The `sync` object is an aggregate across the webspace's PARTICIPATING
sources** (`KERN-04`, narrowed by `08-UAT.md` G-08-3), not a single
most-recent-run and not every configured source — a behavior change from
Phase 1 (single most-recently-recorded run kernel-wide) and again from
the aggregate's original Phase 2 shape (every configured source,
regardless of whether it feeds this webspace). "Participating" is decided
by the identical allowlist-and-match-input rule the sync path itself
applies (`correlate.ParticipatesIn`): a source instance counts only when
it is still configured AND either named in the webspace's `sources`
allowlist implicitly (an empty allowlist admits every configured
instance) or explicitly, AND has actual match input for this webspace (an
explicit `match` block naming it, or a non-empty `keywords` fallback). A
source excluded by the webspace's own `sources` allowlist, or one that
was removed from `[sources.*]` entirely while its sync history survives,
never contributes to this webspace's `sync` object — even if that same
source's failure is real and currently affecting a DIFFERENT webspace it
does feed.

`status` is `"error"` if *any* participating source's latest run errored,
else `"running"` if any participating source is still mid-sync, else
`"ok"` if at least one participating source's run has ever completed,
else the zero value (`""`) if no participating source has ever synced —
including the case where the webspace has no participating sources at
all (e.g. a webspace known only from surviving index rows, with no
`[webspaces.*]` block left in config). `finished_unix` is the newest
`finished_unix` across every participating source's latest run. `error`
joins each failing participating source's message, prefixed with its
source INSTANCE id (never the plugin kind — two instances of one plugin
type report independently), in sorted source order (so it's
deterministic) — e.g. `"work-email: dial tcp: connection refused"`. This
is what stops a two-source webspace whose only failing PARTICIPATING
source returned nothing from rendering as merely empty: before the
participation scope, a webspace with one healthy source and one silently
broken one — even a source that didn't feed this webspace at all — could
have that unrelated failure reported here just because it was the most
recently recorded run anywhere in the kernel. `GET /api/webspaces`'s
`last_sync` field is now computed PER WEBSPACE by this identical rule
(each webspace's own participating sources, not one shared kernel-wide
value), and the `/agent/v1` mirrors below compose this same participation
scoping with grant filtering.

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
      "source": "paperless",
      "source_type": "paperless",
      "source_display_name": "paperless-ngx",
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
same unknown name — including the config-known-before-first-sync case:
a webspace named in the running config but with no synced items yet
answers `200` here too, never `404`.

This route has **no `/agent/v1` mirror** in this version — see "The
`/agent/v1` namespace", below, for why.

### `GET /api/items/{id}`

`{id}` is the stable composite id (`{source}:{source_id}`, e.g.
`paperless:528` — see "The stable-ID scheme", below), accepted both raw and
percent-encoded (`paperless%3A528`) — both resolve to the same item.
Returns the item's indexed metadata (identical shape to a stream row) plus
**one live `Fetch(FULL)` call** to the owning plugin for extracted text
and a rendition descriptor. This is the one route family with
request-time source-system latency; everything else on this API is an
index read. The plugin call is made against the item's `source` (its
source INSTANCE id), never its `source_type` — this is what lets two
configured instances of one plugin binary resolve to the correct running
subprocess each.

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

`text/html` renditions (produced today by the Proton, SilverBullet, and
Signal plugins — an email body, a rendered wiki page, and a chat
transcript, respectively) are sanitized, wrapped and themed by the
**kernel**, not by the producing plugin (D-11). A plugin's `Fetch`
response carries the bare content fragment plus a declared
`content_shape` (`CONTENT_SHAPE_EMAIL_HTML`, `CONTENT_SHAPE_MARKDOWN_HTML`,
or `CONTENT_SHAPE_CHAT_TRANSCRIPT` — see `docs/plugin-contract.md`); this
route looks up that shape's `bluemonday.Policy`, sanitizes the fragment
with it, and wraps the sanitized result in one kernel-owned, self-contained
HTML document (a fixed doctype/head/style skeleton, composed once per
shape from a shared CSS base plus that shape's own delta rules) before
writing any byte. This centralizes what used to be three near-identical
per-plugin sanitize policies and theme stylesheets into one place, so a
theme change is a one-file edit and sanitization stays inside the trust
boundary even once plugins are third-party. Sanitization always runs
strictly **before** wrapping, and the wrapped output is never re-sanitized
— the injected stylesheet is fixed Go source, never derived from the
fragment, so it cannot reintroduce anything the policy just stripped.

A `text/html` rendition whose declared `content_shape` is unrecognised —
including the zero value, `CONTENT_SHAPE_UNSPECIFIED`, which a plugin
built against the pre-Phase-5 contract could never have set correctly —
is refused outright: `502 unsupported_content_shape`, with **no body
written**. The kernel fails closed here rather than ever guessing a
policy or serving an unsanitized document from its own origin. This is
the one rendition-serving failure mode that isn't a plugin-reachability
or MIME-allowlist problem; see the error-code table below.

The kernel's own allowlist-plus-CSP mechanism above (in particular the
`sandbox` CSP directive, which strips script execution, form submission,
and top-level navigation from the iframe an embedding client renders this
content inside) remains a second, independent layer of defense on top of
the sanitize/wrap boundary — not a replacement for it.

`style-src 'unsafe-inline'` exists specifically so a `text/html`
rendition's own inline `<style>` block is actually applied inside the
embedding iframe — `default-src 'none'` with no `style-src` override
otherwise blocks a document from styling itself at all, which shipped as
a live bug (unstyled rendered markdown read as near-black text against
the app's dark theme) before this directive was added. This does not
weaken script blocking: `default-src 'none'` (with no `script-src`
override) and the `sandbox` directive together still deny all script
execution regardless of `style-src`. It's safe specifically because the
only inline style any rendition document can ever carry is the fixed
stylesheet **the kernel's own `sanitizeAndWrapRendition` injects strictly
after its bluemonday policy runs** over the plugin-supplied fragment — the
policy strips any `<style>` element or `style` attribute that originated
from plugin/source content before that trusted stylesheet is ever
appended, so a hostile or malformed source document cannot smuggle a
stylesheet through this directive.

**`?hl=` (UI-09, optional, `GET /api/items/{id}/content` only):** carries
the user's raw in-webspace search query. The kernel whitespace-splits it,
lowercases each field, de-duplicates, drops any field shorter than 2
characters, and caps the result at the first 8 terms — the same
literal-term derivation `GET /api/webspaces/{webspace}/search` itself uses
internally (whitespace-split, no stemming). For a `text/html` rendition,
each derived term is matched case-insensitively as a literal substring
inside the already-sanitized document's text content and wrapped in a
bare `<mark>` element — a tree mutation via `golang.org/x/net/html`
(parse/walk/render), never a byte- or pattern-level substitution over the
sanitized bytes, so it cannot corrupt markup, alter an attribute value, or
reintroduce anything the sanitizer already stripped. The highlighted
output is never re-sanitized: sanitization runs exactly once, strictly
before this step. `?hl=` does not change the sanitizer policy, the
`Content-Security-Policy` header, or how any non-`text/html` rendition
(PDF, image) is served — an absent or empty `?hl=` produces a response
byte-identical to the pre-UI-09 output. **The `/agent/v1` rendition
mirror does not accept `?hl=`** — the agent surface has no search UI, so
`GET /agent/v1/items/{id}/content` always serves an unhighlighted
document regardless of any query string supplied.

### `GET /api/sources`

One entry per configured source **instance** (`name`, the `[sources.<id>]`
config map key — `D-08`): its resolved `display_name` (`D-09`), the
plugin-reported `source_type` (the plugin kind, shared by every instance
of that plugin binary), a **live** reachability probe result, whether it
is currently syncing, and the kernel's own recorded sync history for it
(`PLUG-04`). Sorted by instance id — so two instances of one plugin type
always appear in the same deterministic order, run to run. Two
`[sources.*]` entries whose `plugin` value is identical (e.g. two Proton
accounts) always produce **two distinct entries** here, never one merged
row.

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
`topos sync` CLI command all dedupe against each other via the same
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

### `GET /api/config`

The kernel's live configuration document — the **RAW, pre-expansion**
form, never the runtime-expanded one — plus its content hash and the
set-or-not status of every `${VAR}`/`$VAR` secret reference it contains.
This is the first read route on this API whose response can change without
any sync running: a `PUT /api/config` (below) or a hand-edit of
`config.toml` followed by the kernel's own reload both change what the
very next `GET /api/config` returns.

```
$ curl -s http://127.0.0.1:7777/api/config | jq
{
  "schema_version": 1,
  "hash": "3f2a9c1e...",
  "config": {
    "server": { "listen": "127.0.0.1:7777" },
    "sources": {
      "paperless": { "plugin": "paperless", "base_url": "https://paperless.example.lan", "token": "${PAPERLESS_TOKEN}" }
    },
    "webspaces": {
      "house-move": { "keywords": ["house and home"], "filter": ["boiler"] }
    }
  },
  "env_vars": { "PAPERLESS_TOKEN": true },
  "unknown_keys": []
}
```

**`config` is always the raw, unexpanded document.** A `${VAR}` or `$VAR`
reference inside a `token`, `base_url`, or any other field is returned
**verbatim** — never resolved to the secret value it stands in for. The
expanded, secret-bearing runtime config the kernel actually uses to reach
a source system is held only in server-side memory and is never the value
handed to this response (`D-05`); this route is structurally incapable of
leaking a resolved secret because it never even holds a reference to the
expanded form.

**`hash`** is the hex-encoded content hash of the on-disk file this
response was built from — echo it back as `base_hash` on the next
`PUT /api/config` (below). Two `GET /api/config` calls with no intervening
write return byte-identical bodies, including this hash.

**`env_vars`** reports, per `${VAR}`/`$VAR` name referenced anywhere in
`config`, whether that variable is currently **set** in the kernel
process's own environment — a boolean only, never the value (`D-05` again:
even a set/unset signal stops short of the secret itself).

**`unknown_keys`** lists any TOML key or table `config.toml` carries that
the kernel's `Config` struct does not model (e.g. a stray table left over
from a hand-edit or a prior migration) — sorted, empty array when none. A
non-empty `unknown_keys` list blocks **every** `PUT /api/config` outright
(see `config_has_unknown_keys` below), not only a save that touches the
unrecognised key: the kernel refuses to write a canonical rewrite that
would silently drop content it doesn't understand (`D-01`'s
lossless-rewrite prohibition).

### `PUT /api/config`

The kernel's **first mutating HTTP surface** (success criterion 4),
scoped strictly to configuration — no other route on this API accepts a
request body that changes persisted state. Saves the given config
document to `config.toml` and hot-swaps the kernel's running
configuration in the same call: a filter saved through this route narrows
the very next `GET /api/webspaces/{webspace}/stream` (or `/search`, or the
`/agent/v1` mirror) request with **no kernel restart**.

**Request body:**

```json
{
  "base_hash": "3f2a9c1e...",
  "config": { "...": "the full raw config document, as returned by GET /api/config, with your edits applied" }
}
```

`base_hash` MUST be the `hash` value from the `GET /api/config` (or a
previous `PUT /api/config`) response your edit started from — this is the
`D-03` optimistic-lock clobber guard: the kernel re-reads `config.toml`
from disk at save time and rejects the write if its current hash no
longer matches `base_hash` (see `config_changed_on_disk`, below).

**On success:** `200` with the identical `configResponse` shape
`GET /api/config` returns — the post-save state IS the next `GET`'s state,
so there is no separate "save result" shape to learn. Before writing
anything, the kernel validates the would-be config through the exact same
`(*config.Config).Validate` a hand-edited file must pass at load time
(`D-09`) — reusing the loader's own validator, never a second rule set —
and writes `config.toml.bak` (overwriting any previous backup, `D-04`)
immediately before the new content, via a same-directory temp-file-plus-
rename so the write is atomic (`D-01`): a kernel killed mid-write leaves
`config.toml` at its previous content, never truncated or half-written.

```
$ curl -s -X PUT http://127.0.0.1:7777/api/config \
    -H 'Content-Type: application/json' \
    -d '{"base_hash":"3f2a9c1e...","config":{"...":"edited document"}}' | jq
{ "schema_version": 1, "hash": "9b7e04aa...", "config": { "...": "..." }, "env_vars": { "...": "..." }, "unknown_keys": [] }
```

**Failure modes**, each mapped to the shared error envelope (see the
error-code table below): a malformed request body (`invalid_request`,
`400`), a stale `base_hash` (`config_changed_on_disk`, `409` — the write
is rejected and `config.toml` is left exactly as the out-of-band change
left it, no partial write), an on-disk file carrying keys the `Config`
struct doesn't model (`config_has_unknown_keys`, `409` — refuses rather
than silently dropping them), or a config that fails validation
(`config_invalid`, `422`, carrying the validator's own message verbatim —
no UI-authored paraphrase). Every rejected save writes nothing; the
kernel's own running configuration is untouched until a save actually
succeeds.

**Webspace `filter` (`D-16`/`D-17`/`D-18`):** the mechanism this route
exists to serve for v1 — a webspace's `filter` array (an AND-ed list of
exact, case-sensitive-stripped terms) narrows
`GET /api/webspaces/{webspace}/stream`, `GET /api/webspaces/{webspace}/search`
and `GET /agent/v1/webspaces/{webspace}/stream` **identically**: the
filtered view IS the webspace for every consumer, human and agent alike.
A live search AND-combines with the saved filter stack rather than
replacing it — a further search always refines within the saved filter.
See `config.example.toml` for the field's own worked documentation.

### `POST /api/config/reload`

Re-reads `config.toml` from disk through the **identical** validate-then-
apply path a `PUT /api/config` save uses (`D-08`) — the only way a
hand-edited config file reaches the running kernel, since this API has no
file watcher. Takes no request body.

**On success:** `200` with the identical `configResponse` shape
`GET`/`PUT /api/config` return — the just-reloaded document, its new hash,
`env_vars` and `unknown_keys`.

**On failure:** `422 config_invalid`, carrying the loader's own error
message verbatim, and the kernel's previously running configuration is
left **completely untouched** — the same file-content hash, the same raw
and expanded documents, the same launched plugins. A bad hand-edit can
never kill the kernel by way of this route; a subsequent `GET /api/config`
after a failed reload returns exactly what it would have returned before
the reload was attempted.

```
$ curl -s -X POST http://127.0.0.1:7777/api/config/reload | jq
{ "schema_version": 1, "hash": "9b7e04aa...", "config": { "...": "..." }, "env_vars": { "...": "..." }, "unknown_keys": [] }
```

An apply failure after a successful reload (the file was valid, but the
running kernel could not fully reconcile plugin subprocesses against it)
maps to `500 apply_failed` — see the error-code table below; retry the
same route once the underlying cause (e.g. a plugin binary temporarily
missing) is fixed.

### `GET /api/config/plugin-types`

The plugin binaries actually installed in the kernel's configured plugins
directory — the kernel side of the webspace builder's "+" chip picker's
"New <plugin type>…" list (`D-11`). Only the kernel process can see this
directory on the desktop machine's filesystem; there is no built-in table
of known plugin types anywhere (`docs/plugin-contract.md`'s own `D-05`
discipline, extended here to discovery).

```
$ curl -s http://127.0.0.1:7777/api/config/plugin-types | jq
{ "schema_version": 1, "plugin_types": ["topos-plugin-paperless", "topos-plugin-proton", "topos-plugin-silverbullet"] }
```

`plugin_types` is sorted and never includes `topos-plugin-mock` — the
developer/reference fixture PLUG-05's third-party-implementer proof
builds against, deliberately excluded from the picker so a real
deployment can never accidentally enable a fixed set of fake demo items.

### `POST /api/config/describe-plugin`

Trial-launches a plugin binary against **just-submitted, not-yet-
persisted** connection fields, calls its read-only `Describe` RPC, and
kills the subprocess before this route returns — the kernel half of the
webspace builder's two-step "New <plugin type>…" modal (`D-11` step 1 ->
step 2): step 1 collects connection fields; this route answers what match
fields step 2's form should offer, **before anything is written to
`config.toml`**.

**Request body:**

```json
{
  "plugin": "topos-plugin-paperless",
  "source": { "base_url": "https://paperless.example.lan", "token": "unverified-but-present" }
}
```

`plugin` MUST be a member of `GET /api/config/plugin-types`'s own result
set — a request naming any other value is refused `404
plugin_binary_not_found` **before anything is executed**: directory
listing, never a caller-supplied name, is the authority over what may be
launched (`T-07-09`). `source`'s connection fields need only be
**present**, not working — every plugin type defers live connectivity
past its own startup, so a placeholder token or an unreachable
`base_url` still reaches `Describe` (verified against all four non-Signal
plugin types; see `07-02-SUMMARY.md`).

**On success:** `200` with the plugin kind's own facts — never anything
from the request body echoed back:

```
$ curl -s -X POST http://127.0.0.1:7777/api/config/describe-plugin \
    -H 'Content-Type: application/json' \
    -d '{"plugin":"topos-plugin-paperless","source":{"base_url":"https://paperless.example.lan","token":"unverified"}}' | jq
{ "schema_version": 1, "source_type": "paperless", "plugin_display_name": "paperless-ngx", "match_vocabulary": ["tags"] }
```

**This route persists nothing, registers nothing, and reaches no RPC
beyond `Describe`.** No `[sources.*]` block is added to `config.toml` by
this call, the trial-launched subprocess is never added to the running
kernel's plugin host, and `pluginhost.DescribePluginType`'s own body is
pinned by an AST test to reach no `Match`/`Fetch` call (`T-07-10`,
`PLUG-02`) — the trial-launch path can never become a general
plugin-invocation surface for request-supplied input.

**Failure modes:** `404 plugin_binary_not_found` (above), or `502
plugin_describe_failed` when the trial launch or the `Describe` call
itself fails (e.g. a malformed `base_url` scheme) — carrying the kernel's
own error text so the modal can show it beside its "Save anyway" fallback.

### `GET /api/plugins/{plugin}/icon`

A plugin **binary**'s own declared identity icon (`09-01-PLAN.md` Task 2,
`09-UI-SPEC.md` Fix 10) — keyed by `{plugin}` = the plugin binary name
(`source.plugin`, e.g. `topos-plugin-paperless`; already present on every
`/api/sources` row), never by `source_type` or an instance name. This is a
deliberate `promote`: `source_type`, like the icon itself, is only learned
once a plugin binary has actually been launched and `Describe`d — a
per-instance route (`/api/sources/{name}/icon`) would have implied the
icon is per-instance data when it is not (two instances of one binary have
byte-identical icons).

```
$ curl -s -D- http://127.0.0.1:7777/api/plugins/topos-plugin-mock/icon -o /dev/null
HTTP/1.1 200 OK
Content-Type: image/svg+xml
Cache-Control: public, max-age=31536000, immutable
ETag: "…"
X-Content-Type-Options: nosniff
Content-Disposition: inline
Content-Security-Policy: default-src 'none'; style-src 'unsafe-inline'; sandbox
```

The bytes and mime come straight from the plugin's own `Describe` response
(`DescribeResponse.icon`/`icon_mime`), captured at the same launch-time
`Describe` call the kernel already makes — this route issues **no new
RPC** and reaches no plugin at request time. `Cache-Control` is
long-lived and `immutable`: icon bytes are static for a given binary
build. A conditional request whose `If-None-Match` matches the current
`ETag` gets `304` with no body.

The kernel enforces its own mime allowlist (`image/svg+xml` or
`image/png`) and a 64KB size ceiling at capture time — a plugin cannot
choose the served `Content-Type` outside that set, and an oversized icon
is dropped, never truncated, and never fails the plugin's launch.

**`404 icon_unavailable` is the routine, expected case for any plugin
binary the kernel has never successfully `Describe`d** — no configured
instance of it has ever launched, every launch attempt so far failed
before reaching `Describe`, or it's a pre-Phase-9 binary that never set
these fields. This is never a client-visible error state: the SPA's
`PluginIcon.svelte` always falls back to a generic glyph in this case. A
`{plugin}` value containing a path separator or a `..` segment also 404s,
before the lookup (an exact-match over an in-memory map, never the
filesystem) is even attempted.

### `POST /api/config/whatsapp-link`, `GET /api/config/whatsapp-link/{session}`, `DELETE /api/config/whatsapp-link/{session}`

The in-app WhatsApp QR-pairing surface (`D-01`, `08-03-PLAN.md`): starts,
polls, and cancels a **link session** — a raw subprocess running the
WhatsApp plugin binary in its machine-readable link mode
(`topos-plugin-whatsapp -link-json -path <dir>`, `plugins/whatsapp/
link.go`'s `runLinkJSON`), spawned entirely **outside the `go-plugin` gRPC
handshake**. This is deliberately **not** a `SourcePlugin` RPC —
`docs/plugin-contract.md`'s locked four-RPC allowlist (`Describe`, `Match`,
`Fetch`, `Health`, "no fifth RPC may ever be added") is unaffected by this
route's existence, since nothing here talks to a launched plugin's gRPC
service at all.

Short-poll, not SSE: the browser calls `POST` once to start a session,
then polls `GET .../whatsapp-link/{session}` on a fixed short interval of
its own (a couple of seconds), independent of any code's validity window,
until the session reaches a terminal state, and may `DELETE` to cancel
early. `expires_in_seconds` drives only the countdown the panel displays,
never the poll cadence itself. `08-UAT.md`'s `G-08-1` is why this
distinction is called out explicitly: tying the interval to
`expires_in_seconds` left a terminal event the kernel had already recorded
undelivered to the browser for up to a full 60-second first-code window.

**`POST /api/config/whatsapp-link` request body:**

```json
{
  "plugin": "topos-plugin-whatsapp",
  "path": "~/.local/share/topos/whatsapp",
  "instance": "my-whatsapp"
}
```

`plugin` MUST be a member of `GET /api/config/plugin-types`'s own
discovered-binary set (`pluginhost.DiscoverAllBinaries`) — refused `404
plugin_binary_not_found` **before anything is executed**, identical
authority rule to `POST /api/config/describe-plugin` (`T-08-06`). `path`
is the WhatsApp instance's data directory (the same value as
`[sources.whatsapp].path`) and must be non-empty. `instance` is
**optional**: present for the "Re-link…" chip-menu flow (an
already-configured instance name to suspend for the session's duration —
see below), absent for the Add-Source flow (nothing configured yet to
suspend).

When `instance` is non-empty, the kernel calls `Supervisor.SuspendInstance`
**before** spawning the link subprocess: it stops that instance's own
running plugin process for the session's duration, so the link subprocess
and the regular pluginhost-launched instance never hold WhatsApp's
`sqlstore` session file open at the same time. The suspended instance is
automatically **resumed** the moment the session reaches any terminal
state (whether that is `paired`, `error`, `timeout`, cancellation, or
deadline expiry) — never left suspended past the session's own lifetime.
Naming an instance not currently running is a deliberate no-op (e.g. the
Add-Source flow's own not-yet-saved instance needs no special-casing).

**On success:** `200` with a session id and its current state:

```json
{ "schema_version": 1, "session": "5f1e...c2", "state": "pending" }
```

**`GET /api/config/whatsapp-link/{session}`** returns the **latest**
event the link subprocess has emitted, polled until `state` is one of the
three terminal values below:

```json
{ "schema_version": 1, "session": "5f1e...c2", "state": "qr", "png_data_uri": "data:image/png;base64,...", "expires_in_seconds": 18 }
```

| `state` | Meaning | Extra fields |
|---|---|---|
| `pending` | Session started; no event has arrived from the subprocess yet. | — |
| `qr` | A rotating pairing code is ready to display. | `png_data_uri` (a `data:image/png;base64,...` URI — never the raw pairing payload itself, which is a live credential and never leaves the plugin subprocess as text), `expires_in_seconds` (the real whatsmeow-reported validity window for this specific code, driving the browser's own countdown) |
| `pairing_accepted` | **Non-terminal.** The phone accepted the scan; the plugin is completing the post-pair login handshake. Polling continues. | — |
| `already_linked` | **Non-terminal.** The store already held a linked device when the session started; the plugin is reconnecting to confirm that session is genuinely usable. Polling continues. | — |
| `paired` | The device linked successfully. | — |
| `error` | The link attempt failed. | `code` (`whatsapp_store_in_use` or `link_failed`, see the error-code table below), `message` |
| `timeout` | The QR channel closed without a scan (the code(s) expired). | — |

Neither `pairing_accepted` nor `already_linked` carries a device
identifier: the linked device's JID embeds the user's own phone number
and never crosses this wire (the plugin keeps it in its own stderr
diagnostic only).

**Terminal states are delivered exactly once.** The moment a poll observes
`paired`, `error`, or `timeout`, the kernel retires the session (killing
the subprocess if it hasn't already exited and running the suspended
instance's resume, if any) as part of answering that same request. A
second poll for the same `session` id after a terminal state was already
observed returns `404 link_session_not_found` — the same code an unknown
or already-cancelled/expired session id returns. `pairing_accepted` and
`already_linked` are explicitly **not** terminal — observing either
leaves the session live and pollable, and retires nothing.

The link subprocess's own stderr is captured by the kernel and re-emitted
through the kernel's own logger; it never appears in any HTTP response
body on this route.

**`DELETE /api/config/whatsapp-link/{session}`** cancels an in-progress
session early: kills the subprocess, resumes any suspended instance, and
retires the session.

```json
{ "schema_version": 1, "session": "5f1e...c2", "state": "cancelled" }
```

**Session limits:** a kernel process holds at most a small, fixed number
of concurrent link sessions (`link_failed`, `429`, if exceeded — a stuck
or abandoned browser tab cannot accumulate unbounded subprocesses). A
session left unpolled past its own deadline is terminated by a background
reaper — the same `link_session_not_found` a manual cancel produces. Every
live session's subprocess is also terminated on kernel shutdown, so a
Ctrl-C never orphans a linking process holding WhatsApp's session-store
lock.

**Failure modes:** `400 invalid_request` (malformed body or empty `path`),
`404 plugin_binary_not_found` (above), `404 link_session_not_found` (an
unknown, already-retired, reaped, or already-cancelled session id), `429
link_failed` (the concurrent-session cap), or `502 link_failed` if the
subprocess itself fails to start. A `whatsapp_store_in_use`/`link_failed`
**poll-time** error (the plugin subprocess itself failed, e.g. because its
own store lock was already held) is reported as a `200` response with
`"state": "error"`, not an HTTP-level failure — the poll succeeded; the
underlying link attempt is what failed.

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

**Grants are per source INSTANCE, never per plugin kind (`D-08`, `D-10`).**
`[sources.<name>.agent]` is keyed on the instance id — the same
`[sources.<name>]` map key that carries the grant everywhere else. Two
instances of one plugin type (e.g. `home-email` and `work-email`, both
running the Proton plugin) can be granted independently: granting
`home-email` alone never admits `work-email`'s items, sources entry, or
sync history, even though both share one `source_type`. There is no way
to grant "every instance of a plugin kind" in one declaration — each
instance's `agent.read` is set explicitly.

| Route | Mirrors | Restriction |
|---|---|---|
| `GET /agent/v1/sources` | `GET /api/sources` | Ungranted sources are omitted entirely; each entry gains a `capabilities: {read, handoff}` object. |
| `GET /agent/v1/webspaces` | `GET /api/webspaces` | `item_count` and `last_sync` are computed over granted sources only; `last_sync` additionally composes the same per-webspace participation scoping the `/api` route now applies (see "The `sync` object is an aggregate..." above) — grant filtering never widens what participation alone would report. |
| `GET /agent/v1/webspaces/{webspace}/stream` | `GET /api/webspaces/{webspace}/stream` | `items` and `sync` are restricted to granted sources; ordering is otherwise identical to the `/api` stream with ungranted rows removed, never reordered. `sync` composes grant filtering with the identical participation scoping the `/api` route applies. |
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

Every item's `id` is `"{source}:{source_id}"` — `source` is the source
INSTANCE id, the `[sources.<id>]` config map key the item was synced
through (`D-08`; e.g. `"paperless"`, or `"home-email"`/`"work-email"` for
two Proton instances), and `source_id` is the owning plugin's own stable
local identifier for the object (e.g. `"528"`, a paperless-ngx document
id). Identity comes only from the operator's own config map key — never
from anything a plugin process asserts, a binary filename, or launch
order. `source_type`, the plugin kind learned from `Describe`, is retained
as descriptive provenance but never appears in the id and is never used to
key identity anywhere in the kernel.

- **Stable across syncs**: re-syncing never changes an existing item's id;
  the same source object always upserts to the same row.
- **Unique across source instances**: two different source instances can
  use overlapping `source_id` values (a paperless-ngx document `"1"` and
  an IMAP message `"1"` — or even two instances of the *same* plugin type,
  each with their own `source_id "1"`) without colliding, because the
  `source` (instance id) prefix disambiguates them. Two instances of one
  plugin type never merge: their items occupy distinct id namespaces,
  their sync history forms independent per-instance series, and an
  `agent.read` grant on one instance never admits the other's items
  (`D-10`).
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
| `source_type` | Matches the item's own top-level `source_type` — the plugin kind (`Describe`-reported), never the id prefix (that's `source`, the instance id — see "The stable-ID scheme"). |
| `source_system` | The specific source instance's connection endpoint (e.g. a paperless-ngx base URL) — distinguishes "which paperless-ngx" if you configure more than one instance; the item's top-level `source` field is the config-authored instance id for the same purpose. |
| `source_id` | Matches the item's own `source_id`. |
| `plugin` | The plugin binary name that produced this item (e.g. `"topos-plugin-paperless"`). |
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
| `webspace_not_found` | 404 | `GET /api/webspaces/{webspace}/stream`, `GET /api/webspaces/{webspace}/search`, `GET /agent/v1/webspaces/{webspace}/stream` | `{webspace}` is in neither the running configuration nor the local index. A configured-but-never-synced webspace is NOT this case — it answers `200` with `"items": []` (or `"results": []`) instead. |
| `item_not_found` | 404 | `GET /api/items/{id}` and its `/content`, `/thumbnail` children | `{id}` does not exist in the local index (or, for `Fetch`-level failures, the plugin itself reports the source object no longer exists). On `/agent/v1/*`, also covers an `{id}` whose source exists but is ungranted — deliberately the same code as a genuinely nonexistent id. |
| `source_unavailable` | 502 | `GET /api/items/{id}` and its `/content`, `/thumbnail` children | The live `Fetch` call to the owning plugin failed — the source system was unreachable or errored. |
| `unsupported_rendition_type` | 415 | `GET /api/items/{id}/content`, `/thumbnail` | The plugin reported a rendition MIME type outside the fixed allowlist; the kernel refuses to serve it. |
| `unsupported_content_shape` | 502 | `GET /api/items/{id}/content` | The rendition's `mime_type` is `text/html` but its plugin-declared `content_shape` is unrecognised or unspecified (`CONTENT_SHAPE_UNSPECIFIED`) — the kernel's sanitize/wrap boundary (D-11) refuses to guess a policy and writes no body. |
| `content_unavailable` | 404 | `GET /api/items/{id}/content`, `/thumbnail` | The item exists and the plugin was reachable, but no rendition is available for this specific variant (distinct from `item_not_found`: the item is real, this rendition just doesn't exist). |
| `source_not_found` | 404 | `POST /api/sources/{name}/refresh` | `{name}` does not match any configured `[sources.<name>]` entry. The message never enumerates which names do exist. |
| `invalid_request` | 400 | `PUT /api/config` | The request body is not valid JSON, or is missing the `config` field. |
| `config_changed_on_disk` | 409 | `PUT /api/config` | `base_hash` no longer matches `config.toml`'s current on-disk hash — someone else (another browser tab, a hand-edit, a `topos` CLI run) saved since you last read it. Nothing is written; re-`GET /api/config` and retry. |
| `config_has_unknown_keys` | 409 | `PUT /api/config` | `config.toml` carries a TOML key or table the `Config` struct doesn't model. The kernel refuses to write a canonical rewrite that would silently drop it — the message names the offending key(s); fix them by hand before any UI save can succeed. |
| `config_invalid` | 422 | `PUT /api/config`, `POST /api/config/reload` | The submitted (or reloaded) config fails the same `(*config.Config).Validate` a hand-edited file must pass at load time. The message is the validator's own error string, verbatim. On a failed reload, the previously running configuration is left completely untouched. |
| `apply_failed` | 500 | `PUT /api/config`, `POST /api/config/reload` | The file was written/reloaded and is now the kernel's config-of-record, but the running kernel (plugin host, coordinator, scheduler) could not fully reconcile against it — never a silent `200`. Retry `POST /api/config/reload`. |
| `plugin_binary_not_found` | 404 | `POST /api/config/describe-plugin`, `POST /api/config/whatsapp-link` | The named `plugin` is not a member of `GET /api/config/plugin-types`'s own result set (or, for the link route, `pluginhost.DiscoverAllBinaries`'s result set) — refused before anything is executed. |
| `plugin_describe_failed` | 502 | `POST /api/config/describe-plugin` | The trial launch or its `Describe` call failed (e.g. a malformed `base_url` scheme) — the kernel's own error text, so the modal can show it beside its "Save anyway" fallback. |
| `link_session_not_found` | 404 | `GET /api/config/whatsapp-link/{session}`, `DELETE /api/config/whatsapp-link/{session}` | `{session}` is unknown, was already retired after a terminal poll, was reaped past its deadline, or was already cancelled. |
| `whatsapp_store_in_use` | — (200, `state: "error"`) | `GET /api/config/whatsapp-link/{session}` | The link subprocess's own store lock was already held by another `topos-plugin-whatsapp` process — the independent second layer behind `SuspendInstance` (`T-08-07`). Not an HTTP-level failure; see the whatsapp-link route's own "Failure modes" note above. |
| `link_failed` | 429 (session cap) / 502 (subprocess start failure) / — (200, `state: "error"`, any other link-subprocess failure) | `POST`, `GET /api/config/whatsapp-link*` | A generic link failure — the concurrent-session cap, a subprocess that failed to start, or any plugin-reported error other than a store-lock conflict. |
| `internal_error` | 500 | any route | An unexpected kernel-side failure (e.g. the local index file itself is unreadable) — not a source or plugin problem. |

## What is not here yet

This API is read-only over **source data** end to end for the whole of
v1 — `PUT /api/config` (above) is the one deliberate exception, a
mutating surface scoped strictly to the kernel's own configuration, never
to anything a plugin's source system holds. Not yet present, and not
planned for this phase:

- **Agent-initiated actions** (`AGENT-11`, e.g. "draft an email reply") —
  no route on either namespace performs an action against a *source*
  system in v1; action hand-off is a v1.x concern layered on top of the
  `/agent/v1` permission model documented above.
