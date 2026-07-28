# The Webspaces plugin contract

This document is the published, third-party-facing contract for a
webspaces **source plugin**: a subprocess that reads one personal data
silo (paperless-ngx, an IMAP mailbox, a Signal database, a SilverBullet
space, ...) and hands normalized items back to the kernel. It is written
for a reader with no access to this repository beyond three things:

- this file,
- `proto/webspaces/v1/plugin.proto` (the wire contract), and
- the `sdk` Go module (`github.com/davison/webspaces/sdk`).

If those three are all you have, you should be able to write a working
plugin. The one shipping reference implementation, `plugins/paperless`, is
a second, fuller example — but the contract below is complete without it.

## A plugin is read-only by construction

Webspaces never mutates a source. This is not a policy a plugin author is
asked to follow — it is a property of the contract's shape:

- `SourcePlugin`, the gRPC service every plugin implements, declares
  exactly four RPCs: `Describe`, `Match`, `Fetch`, `Health`. None of them
  writes, and no fifth RPC may ever be added to widen that set.
- This is mechanically enforced, not just documented: `sdk/contract_test.go`
  reads `plugin.proto` and asserts its RPC set against a fixed allowlist.
  Adding any RPC — mutating or not — fails that test (and therefore the
  build) until the addition is a deliberate, reviewed widening of the
  allowlist.
- The reference plugin's own HTTP client is additionally checked by
  `plugins/paperless/readonly_test.go`, which walks the Go AST of every
  file under `plugins/` and fails if any file constructs a non-`GET` HTTP
  request. A third-party plugin outside this repository doesn't get that
  specific test for free, but it inherits the same shape: there is no RPC
  in the contract that could carry a write, so there is nothing for a
  plugin's own outbound requests to trigger beyond reads.

A plugin may talk to its source system however it needs to (REST, IMAP,
a local database file, a linked-device WebSocket) — the read-only
guarantee lives at the `SourcePlugin` RPC boundary, not inside a plugin's
own implementation.

## Depending on the SDK

A plugin is a separate Go module (or, if the source plugin's language
support ever expands, a separate binary speaking the same gRPC contract)
that imports `github.com/davison/webspaces/sdk`:

```go
import (
	"github.com/davison/webspaces/sdk"
	webspacesv1 "github.com/davison/webspaces/sdk/gen/webspaces/v1"
)
```

The SDK module is the stable Go-native surface: it re-exports the
handshake config, the plugin map, and a `plugin.GRPCPlugin` adapter that
wires a Go implementation of the `SourcePlugin` interface to the generated
gRPC stubs. You implement `sdk.SourcePlugin`, not the raw generated gRPC
server type:

```go
type SourcePlugin interface {
	Describe(ctx context.Context, req *webspacesv1.DescribeRequest) (*webspacesv1.DescribeResponse, error)
	Match(ctx context.Context, req *webspacesv1.MatchRequest) (*webspacesv1.MatchResponse, error)
	Fetch(ctx context.Context, req *webspacesv1.FetchRequest) (*webspacesv1.FetchResponse, error)
	Health(ctx context.Context, req *webspacesv1.HealthRequest) (*webspacesv1.HealthResponse, error)
}
```

A plugin's `main` package registers that implementation and serves it:

```go
plugin.Serve(&plugin.ServeConfig{
	HandshakeConfig: sdk.Handshake,
	Plugins: map[string]plugin.Plugin{
		"source": &sdk.SourcePluginGRPCPlugin{Impl: &myPlugin{}},
	},
	GRPCServer: sdk.GRPCServer, // raises the gRPC message-size ceiling — see Fetch, below
})
```

## Handshake and the plugin-map key

The kernel and every plugin share one handshake, `sdk.Handshake`:

| Field | Value |
|---|---|
| `ProtocolVersion` | `1` |
| `MagicCookieKey` | `WEBSPACES_PLUGIN` |
| `MagicCookieValue` | `webspaces-source-plugin-v1` |

`ProtocolVersion` is bumped only for a breaking wire-protocol change (not
for an additive contract change — that's what `DescribeResponse`'s
`contract_version` field is for; see Describe, below). A plugin that
serves a different magic cookie or protocol version fails the handshake
outright, before any RPC is attempted.

Every plugin registers its implementation under the plugin-map key
**`"source"`** — this must match exactly, on both the plugin side
(`plugin.ServeConfig.Plugins`) and implicitly on the kernel side (the
kernel always dispenses `"source"`).

## Discovery and launch

The kernel discovers plugins by scanning a plugins directory for binaries,
not from a compile-time list — this is what lets a kernel build ship
without a Signal-plugin's cgo/C-toolchain requirement for a user who
doesn't configure Signal. The directory defaults to `plugins`, resolved
relative to the running `webspaces` executable, and is overridable via the
`[plugins] dir` config key (see `config.example.toml`).

For each configured `[sources.<name>]` entry, the kernel launches
`<plugins-dir>/<plugin-binary-name>` as a subprocess and negotiates the
handshake over gRPC only — `AllowedProtocols` is restricted to
`[]plugin.Protocol{plugin.ProtocolGRPC}`, so a plugin implementing only
the legacy net/rpc transport will fail to connect. Immediately after a
successful handshake, the kernel calls that plugin's `Describe` RPC and
uses the returned `source_type` as the plugin's identity for the rest of
the process's lifetime — **a plugin's identity is never trusted from its
filename or its config key**, only from what `Describe` reports.

## Configuration: `WEBSPACES_SOURCE_CONFIG`

The kernel never passes a plugin's connection details as CLI flags. It
marshals the relevant `[sources.<name>]` config into JSON and sets it as
the `WEBSPACES_SOURCE_CONFIG` environment variable on the launched
subprocess. Today that JSON looks like:

```json
{ "base_url": "https://paperless.example.lan", "token": "abc123...", "api_version": "10" }
```

The exact key set is source-specific (a chat plugin has no `base_url`; a
local-database plugin needs a filesystem path instead) — a plugin defines
and documents whatever keys it needs, and reads them out of this one
environment variable at startup.

A plugin **must fail startup loudly** when a required key is empty (for
example, because the operator's config referenced an unset `${VAR}` that
expanded to `""`) — never start up silently and fail later, mid-`Match`,
with a confusing downstream error. Log the missing key by name (never log
the value of a secret key such as a token) and exit non-zero.

## RPC semantics

### `Describe`

Called once, immediately after the handshake, before any other RPC.
Returns the plugin's identity:

```protobuf
message DescribeResponse {
  string source_type      = 1;  // e.g. "paperless" — the kernel's only
                                  // trusted source of this plugin's identity
  string display_name     = 2;  // e.g. "paperless-ngx" — for UI/logs
  string contract_version = 3;  // e.g. "webspaces.v1"
}
```

`contract_version` is the additive-compatibility signal: a plugin built
against an older but still-compatible revision of this contract can report
that revision here without triggering a handshake-level `ProtocolVersion`
bump.

### `Match`

Called only at sync time, never at request time (item-open). The kernel
passes the **full keyword list of one webspace** — every keyword declared
for that webspace in config, unordered — and the plugin returns every item
whose native categorization in the source system matches **any** of those
keywords.

```protobuf
message MatchRequest  { repeated string keywords = 1; }
message MatchResponse { repeated Item   items    = 1; }
```

Matching must be **exact and case-insensitive** against the source's own
native categorization (a paperless-ngx tag name, an IMAP folder/label, a
chat group name, a SilverBullet page tag) — never a substring or prefix
match. `house` must match a tag literally named `House`, and must **not**
match a tag named `Household`. If a silo names something differently than
the webspace's primary keyword, the fix is adding that variant string to
the webspace's keyword list in config — there is no per-source override
syntax, and a plugin must not invent its own fuzzy-matching behavior to
compensate.

**Worked example** — the reference paperless-ngx plugin resolves each
keyword to zero or more tag IDs via an exact, case-insensitive tag-name
lookup, then fetches every document carrying any of the resolved tag IDs:

```go
func (p *SourcePlugin) Match(ctx context.Context, req *webspacesv1.MatchRequest) (*webspacesv1.MatchResponse, error) {
	tagIDs, err := p.client.ResolveTagIDs(ctx, req.GetKeywords()) // exact, case-insensitive
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "paperless: resolve tag ids: %v", err)
	}
	docs, err := p.client.ListDocuments(ctx, tagIDs) // OR across all resolved tag ids
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "paperless: list documents: %v", err)
	}
	items := make([]*webspacesv1.Item, 0, len(docs))
	for _, d := range docs {
		items = append(items, p.toItem(d))
	}
	return &webspacesv1.MatchResponse{Items: items}, nil
}
```

Return a gRPC `codes.Unavailable` status (not a partial, silently-empty
result) when the source system cannot be reached — the kernel records
this per-source in that sync run's status and surfaces it as
`source_unavailable`-shaped state, rather than treating "the source is
down" the same as "nothing matched."

### `Fetch`

Called only at request time — when a user (or an agent) opens a specific
item — **never** from the sync/`Match` path. This is the live half of the
hybrid data model: `Match` supplies metadata and a bounded preview for the
index; `Fetch` supplies the full extracted text and/or a byte rendition,
fetched fresh from the source on every call.

```protobuf
message FetchRequest { string source_id = 1; ContentVariant variant = 2; }

message FetchResponse {
  bool   available                = 1;
  string unavailable_reason       = 2;
  string mime_type                = 3;  // "" when there is no binary rendition
  int64  size_bytes               = 4;
  string text                     = 5;  // extracted text, may be ""
  bytes  data                     = 6;  // rendition bytes, may be empty
  map<string, string> provenance  = 7;
}
```

`Fetch` is a **single unary RPC**, not a stream: the full rendition's
bytes are returned in one `FetchResponse` message. This was a deliberate
decision (documented in this project's phase history as decision
"D-Task1, option-a") over an initially-sketched streaming alternative —
unary keeps the plugin-author-facing contract simpler, at the cost of
requiring both sides to raise gRPC's default 4 MiB message-size ceiling.
`sdk.GRPCServer` (used in `plugin.ServeConfig.GRPCServer` on the plugin
side) and the kernel's own dial options both raise this to **64 MiB** —
comfortably covering a scanned-PDF preview or thumbnail. A rendition
materially larger than that is expected to fail with a clear gRPC
`ResourceExhausted` error rather than succeed silently truncated; if your
source routinely produces larger renditions, downsize or transcode before
returning them.

`variant` selects what's being requested:

| `ContentVariant` | Meaning |
|---|---|
| `CONTENT_VARIANT_FULL` | Extracted text plus (if available) the primary inline-preview rendition, in one call |
| `CONTENT_VARIANT_PREVIEW` | Just the inline-preview rendition, no text |
| `CONTENT_VARIANT_THUMBNAIL` | Just a small thumbnail rendition, no text |

`available = false` with a populated `unavailable_reason` is a **normal,
expected outcome** — e.g. a document type your source can't render a
preview for — not an error. Return a gRPC error status only for an actual
failure to reach the source system (`codes.Unavailable`) or a source id
that no longer exists (`codes.NotFound`); the kernel maps these to
`source_unavailable` (502) and `item_not_found` (404) respectively on its
own HTTP surface (see `docs/api.md`).

### `Health`

```protobuf
message HealthRequest {}
message HealthResponse { bool reachable = 1; int64 last_sync_unix = 2; string last_error = 3; }
```

A lightweight reachability probe. Phase 1 doesn't yet surface this in the
UI (that's `PLUG-04`, Phase 2) but the RPC exists in the contract now so
every plugin implements it from the start rather than retrofitting it
later.

## The `Item` message

Every item a plugin returns from `Match` is normalized into this shape:

```protobuf
message Item {
  string source_id                = 1;  // stable, plugin-local id
  string source_type              = 2;
  string title                    = 3;
  string preview                  = 4;  // bounded snippet, never full content
  int64  timestamp_unix           = 5;  // primary sort: real-world time
  int64  secondary_timestamp_unix = 6;  // tie-break: ingestion/receipt time
  LinkFidelity fidelity           = 7;
  string deep_link                = 8;
  repeated string labels          = 9;  // native categorization (tag/folder/group names)
  map<string, string> provenance  = 10;
  string group_id                 = 11; // chat thread / mail conversation; "" for documents
  string group_label              = 12;
  bool   has_thumbnail            = 13;
}
```

| Field | Meaning |
|---|---|
| `source_id` | Stable within your plugin — the kernel derives its own global id as `"{source_type}:{source_id}"`. Never reuse a `source_id` for two different underlying objects. |
| `source_type` | Must exactly match what your `Describe` RPC reports — the kernel doesn't trust a value here that disagrees with `Describe`. |
| `title` | Short, human-readable. |
| `preview` | A bounded snippet (hundreds of characters, not the full document/message) — the local index stores this, never full content, per the hybrid data model. |
| `timestamp_unix` | The primary sort key across the whole stream — real-world event time (when a document was created, a message sent). |
| `secondary_timestamp_unix` | The tie-break sort key when two items share `timestamp_unix` — typically an ingestion or receipt time. This exists because a date-only source (paperless-ngx's `created` field, for example, has day granularity, not a timestamp) still needs a deterministic same-day order; use the more precise field you have (e.g. paperless-ngx's full-datetime `added` field) here. |
| `fidelity` | See `LinkFidelity`, below — never omit this; the kernel rejects (at sync time) any item with an unspecified fidelity. |
| `deep_link` | An absolute URL back to the source system for this exact item. Never empty — also rejected at sync time if it is. |
| `labels` | The source's own native categorization strings (tag names, folder names) — informational, not used for matching (matching happens inside your `Match` implementation, before this message is built). |
| `provenance` | Machine-readable provenance the kernel HTTP API republishes verbatim to agents (AGENT-02) — see below for the documented key set. |
| `group_id` / `group_label` | For sources with a natural thread/conversation concept (a chat, a mail thread): a stable id and human label for that group. Leave both `""` for a source with no such concept (a standalone document). |
| `has_thumbnail` | Whether a `CONTENT_VARIANT_THUMBNAIL` fetch is expected to succeed — lets the UI decide whether to render a thumbnail slot without an extra round-trip. |

### Provenance

`provenance` is a `map<string, string>` your plugin populates on every
`Item`. The kernel's published HTTP contract (`docs/api.md`) documents six
provenance keys on every item it serves to a client: `source_type`,
`source_system`, `source_id`, `plugin`, `contract_version`, and
`synced_at_unix`. A plugin is responsible for populating the first five;
`synced_at_unix` is filled in by the kernel's index layer at read time
(never by a plugin — a plugin doesn't know when the kernel will next read
its own persisted row) and will be overwritten if your `Item.provenance`
happens to set it. A reasonable minimum for a plugin to set:

```go
Provenance: map[string]string{
	"source_type":      sourceType,           // matches Describe's source_type
	"source_system":    p.baseURL,             // the source instance this came from
	"source_id":        sourceID,              // matches Item.source_id
	"plugin":            "webspaces-plugin-<yours>",
	"contract_version": contractVersion,       // matches Describe's contract_version
},
```

## `LinkFidelity`

```protobuf
enum LinkFidelity {
  LINK_FIDELITY_UNSPECIFIED       = 0;
  LINK_FIDELITY_EXACT             = 1;
  LINK_FIDELITY_ANCHORED          = 2;
  LINK_FIDELITY_CONVERSATION_ONLY = 3;
}
```

| Value | Meaning |
|---|---|
| `LINK_FIDELITY_EXACT` | `deep_link` opens exactly this object (a paperless-ngx document at its own URL). |
| `LINK_FIDELITY_ANCHORED` | `deep_link` opens the right context but not necessarily scrolled/highlighted to the exact object (e.g. a folder view). |
| `LINK_FIDELITY_CONVERSATION_ONLY` | `deep_link` can only open the surrounding conversation/thread, not the specific message (common for chat sources with no per-message deep-link scheme). |
| `LINK_FIDELITY_UNSPECIFIED` | The zero value — never send this. The kernel's sync-time correlation step rejects any item with an unspecified fidelity before it reaches the index (this specific item is skipped and logged; the rest of that sync's valid items still persist). |

`LinkFidelity` is a three-value enum rather than a boolean deliberately —
the UI and the HTTP API both need the distinction, and a chat source in
particular usually can't offer `EXACT` per-message links.

## `ContentVariant`

```protobuf
enum ContentVariant {
  CONTENT_VARIANT_UNSPECIFIED = 0;
  CONTENT_VARIANT_FULL        = 1;
  CONTENT_VARIANT_PREVIEW     = 2;
  CONTENT_VARIANT_THUMBNAIL   = 3;
}
```

See the `Fetch` section above for the meaning of each non-zero value.
`CONTENT_VARIANT_UNSPECIFIED` is the zero value and is never a valid
request; a plugin receiving it should return an `InvalidArgument` gRPC
error.

## Logging

Plugins log through `hashicorp/go-hclog` (the same logging library the
kernel uses to supervise the subprocess), so plugin and kernel log lines
interleave sanely in one stream rather than needing to be tailed
separately. **A plugin must never log a credential** — an API token, a
decrypted database key, an `Authorization` header value — at any log
level, including debug. Log the *presence* or *name* of a secret
(`"token configured"`, `"missing environment variable X"`), never its
value.

## What this document does not cover

- The kernel HTTP JSON API that a browser or an agent consumes — see
  `docs/api.md`.
- Plugin health surfaced in the UI, and a per-plugin agent permission
  model — both land in Phase 2 (`PLUG-04`, `AGENT-01`).
- A reference "mock" plugin built purely from this document with no access
  to the reference paperless-ngx implementation — planned for Phase 2
  (`PLUG-05`) as the validation step for this contract.
