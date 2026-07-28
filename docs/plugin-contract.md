# The Webspaces plugin contract

This document is the published, third-party-facing contract for a
webspaces **source plugin**: a subprocess that reads one personal data
silo (paperless-ngx, an IMAP mailbox, a Signal database, a SilverBullet
space, ...) and hands normalized items back to the kernel. It is written
for a reader with no access to this repository beyond four things:

- this file,
- `proto/webspaces/v1/plugin.proto` (the wire contract),
- the `sdk` Go module (`github.com/davison/webspaces/sdk`), and
- `plugins/mock` — a complete, working reference plugin built from
  exactly these four inputs and nothing else (`PLUG-05`; see "Build your
  first plugin", below).

If those four are all you have, you should be able to write a working
plugin — `plugins/mock` is the proof: it was built and validated against
this document with no access to any real-source plugin implementation.

**Other implementations (aside, not required reading):** two fuller,
real-source examples also ship in this repository —
`plugins/paperless` (a REST API source) and `plugins/silverbullet` (an
HTTP-with-frontmatter source) — useful once you're past "Build your first
plugin" and want to see how a plugin structures a real HTTP client, but
neither is needed to understand or apply anything in this document.

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
- Every plugin shipped in this repository is additionally checked by a Go
  AST scan that walks every file under `plugins/` and fails the build if
  any file constructs a non-`GET` HTTP request (`http.MethodPost`,
  `http.NewRequest(http.MethodDelete, ...)`, and so on). A third-party
  plugin outside this repository doesn't get that specific scan for free,
  but it inherits the same shape from the contract itself: there is no RPC
  that could carry a write, so there is nothing for a plugin's own
  outbound requests to trigger beyond reads. If your plugin's source
  system exposes a read-only API token or credential, prefer that over a
  read/write one — the contract structurally prevents this kernel from
  ever asking your plugin to write, but a well-scoped credential is a
  second, independent line of defense at your source system's own
  boundary.

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

**A plugin with nothing to configure reads the variable and does nothing
with it.** Not every source needs connection details at all — a source
that has no external system to reach (like `plugins/mock`) simply never
requires `WEBSPACES_SOURCE_CONFIG` to be set:

```go
// plugins/mock/main.go — read it if present (forward-compatible with an
// operator setting an empty [sources.mock] config block), but never fail
// startup for its absence, unlike a plugin with real required keys.
_ = os.Getenv("WEBSPACES_SOURCE_CONFIG")
```

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

**Worked example** — `plugins/mock`'s `Match` (the full file is
`plugins/mock/plugin.go`) has a fixed, in-memory item set instead of a
real source system to query, but the matching rule itself is identical to
what a real plugin must implement: every item whose `Labels` contains any
keyword, compared exactly and case-insensitively:

```go
func (p *SourcePlugin) Match(_ context.Context, req *webspacesv1.MatchRequest) (*webspacesv1.MatchResponse, error) {
	keywords := req.GetKeywords()
	var items []*webspacesv1.Item
	for _, it := range mockItems {
		if labelsMatchAnyKeyword(it.GetLabels(), keywords) {
			items = append(items, it)
		}
	}
	return &webspacesv1.MatchResponse{Items: items}, nil
}

func labelsMatchAnyKeyword(labels, keywords []string) bool {
	for _, label := range labels {
		for _, kw := range keywords {
			if strings.EqualFold(label, kw) { // exact, case-insensitive
				return true
			}
		}
	}
	return false
}
```

A real plugin's `Match` typically has one more step before this: resolving
each keyword against the source system's own categorization API (an HTTP
call to look up a tag by name, an IMAP `LIST` to find a matching folder,
...) before it can even ask "which items carry this categorization" — the
mock skips that step because its "categorization" (`Labels`) is already
in memory. Whatever that resolution step looks like for your source,
return a gRPC `codes.Unavailable` status (not a partial, silently-empty
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

A lightweight reachability probe, called live on every request to
`GET /api/sources` / `GET /agent/v1/sources` (`PLUG-04`) — never cached,
so implement it as a cheap operation (a lightweight list/ping call, not a
full resync). Return `reachable: false` with `last_error` set for any
failure to reach the source system; never return a gRPC error from
`Health` itself.

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

| Field | Required? | Meaning |
|---|---|---|
| `source_id` | **Required** | Stable within your plugin — the kernel derives its own global id as `"{source_type}:{source_id}"`. Never reuse a `source_id` for two different underlying objects. |
| `source_type` | **Required** | Must exactly match what your `Describe` RPC reports — the kernel doesn't trust a value here that disagrees with `Describe`. |
| `title` | **Required** (may be a placeholder string, never truly empty) | Short, human-readable. |
| `preview` | Optional — may be `""` | A bounded snippet (hundreds of characters, not the full document/message) — the local index stores this, never full content, per the hybrid data model. |
| `timestamp_unix` | **Required** | The primary sort key across the whole stream — real-world event time (when a document was created, a message sent). |
| `secondary_timestamp_unix` | Optional — may be `0` | The tie-break sort key when two items share `timestamp_unix` — typically an ingestion or receipt time. This exists because a date-only source (a `created` field with only day granularity, for example) still needs a deterministic same-day order; use the more precise field you have (an `added`/`received` timestamp, if your source has one) here. |
| `fidelity` | **Required**, and must not be `LINK_FIDELITY_UNSPECIFIED` | See `LinkFidelity`, below — the kernel rejects (at sync time) any item with an unspecified fidelity; that one item is skipped and logged, the rest of that sync's valid items still persist. |
| `deep_link` | **Required**, must not be `""` | An absolute URL back to the source system for this exact item — also rejected at sync time if empty, with the same skip-and-log behavior as an unspecified fidelity. |
| `labels` | Optional — may be empty | The source's own native categorization strings (tag names, folder names) — informational, not used for matching (matching happens inside your `Match` implementation, before this message is built). |
| `provenance` | **Required** — populate the five plugin-owned keys (see "Provenance", below) | Machine-readable provenance the kernel HTTP API republishes verbatim to agents (AGENT-02). |
| `group_id` / `group_label` | Optional — leave both `""` for a source with no thread concept | For sources with a natural thread/conversation concept (a chat, a mail thread): a stable id and human label for that group. |
| `has_thumbnail` | Optional — defaults to `false` | Whether a `CONTENT_VARIANT_THUMBNAIL` fetch is expected to succeed — lets the UI decide whether to render a thumbnail slot without an extra round-trip. |

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

## Build your first plugin

This walkthrough goes from an empty directory to a plugin the kernel
launches and calls successfully, using nothing beyond the four inputs
listed at the top of this document. `plugins/mock` is the worked example
throughout — every step below names the exact file in that module where
the step lives.

**1. Create a new Go module under `plugins/`.**

```
mkdir plugins/yourplugin && cd plugins/yourplugin
go mod init github.com/davison/webspaces/plugins/yourplugin
```

(Substitute your own module path if you're building outside this
repository entirely — nothing about the contract requires your plugin to
live in this repo.)

**2. Add your module to the Go workspace, if building inside this repo.**

`go.work` at the repository root lists every module `go build`/`go test`
resolve across in one pass — see the top-level `use (...)` block, which
`plugins/mock`'s entry (`./plugins/mock`) mirrors. Add your own module's
path there the same way, or run `go work use ./plugins/yourplugin`.

**3. Depend on the `sdk` module** (see "Depending on the SDK", above) and
implement `sdk.SourcePlugin`'s four methods — `Describe`, `Match`,
`Fetch`, `Health` — on a type of your own (`plugins/mock/plugin.go`'s
`SourcePlugin` struct and its four methods are the complete worked
example; start there and adapt).

**4. Write your `main` package** (`plugins/mock/main.go` is the complete
worked example): read `WEBSPACES_SOURCE_CONFIG` if your plugin needs
connection details (see "Configuration", above — and note a plugin with
nothing to configure, like the mock, simply doesn't require it), construct
your `SourcePlugin` implementation, and call `plugin.Serve` with
`sdk.Handshake`, your implementation registered under the `"source"` key,
and `sdk.GRPCServer` (see "Depending on the SDK", above, for the exact
shape).

**5. Build it.**

```
CGO_ENABLED=0 go build -o bin/plugins/webspaces-plugin-yourplugin ./plugins/yourplugin
```

**6. Configure the kernel to launch it** — add a
`[sources.<name>]` block to your `config.toml` (see `config.example.toml`
for the exact shape, including its commented-out `[sources.mock]`
example) naming your binary's filename as `plugin`, plus whatever
connection-detail keys your plugin's own `main.go` reads out of
`WEBSPACES_SOURCE_CONFIG`. Add a `[webspaces.<name>]` block with at least
one keyword that matches something your `Match` implementation returns,
so a sync actually produces items to see.

**7. Run it.** `webspaces sync` (a one-shot sync) or `webspaces serve`
(sync-on-schedule plus the HTTP API) both launch every configured plugin,
call `Describe` immediately after the handshake, and then drive `Match`
at sync time and `Fetch`/`Health` at request time, exactly per "RPC
semantics", above. If your plugin fails to launch, the kernel's own
startup log names which configured source failed and why — the handshake
and `Describe` call both happen before any sync work starts, so a
misconfigured plugin fails fast rather than silently producing zero
items.

**8. Write tests against the behavior list, not the implementation** —
`plugins/mock/plugin_test.go` is the complete worked example: it asserts
`Describe`'s identity fields, `Match`'s exact-case-insensitive rule and
its zero-items-on-no-match behavior, that every returned `Item` carries a
non-`UNSPECIFIED` fidelity and a non-empty `deep_link` (the same
correlation-boundary check the kernel itself enforces at sync time — see
`fidelity`/`deep_link` in the `Item` table, above), `Fetch`'s
not-found-maps-to-`codes.NotFound` behavior, and `Health`'s always-true
shape for a source with nothing to be unreachable from. Adapt the same
assertions against your own plugin's real behavior.

This is the exact process `plugins/mock` itself was built and validated
through (`PLUG-05`) — a fresh agent context, given only this document,
`plugin.proto`, the `sdk` module, and `plugins/mock` as inputs, produced a
second working plugin from these steps alone. See 02-04-SUMMARY.md for
that validation exercise's record (inputs given, gaps found and closed,
and the honestly-stated limits of that approximation).

## What this document does not cover

- The kernel HTTP JSON API that a browser or an agent consumes — see
  `docs/api.md`, including its `/agent/v1/*` namespace and the per-source
  `agent.read`/`agent.handoff` grants (`AGENT-01`) that gate it. Nothing
  in this document — the plugin contract itself — is grant-aware; grants
  are a kernel-side, config-driven concern applied after your plugin's
  items reach the index, not something a plugin implements or checks.
- Agent-initiated actions (`AGENT-11`, e.g. "draft an email reply") — the
  whole contract above is read-only end to end; action hand-off, if it
  ever lands, is a v1.x concern layered on top of the RPCs above, not a
  change to them.
