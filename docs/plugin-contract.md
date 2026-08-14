# The topos plugin contract

This document is the published, third-party-facing contract for a
topos **source plugin**: a subprocess that reads one personal data
silo (paperless-ngx, an IMAP mailbox, a Signal database, a SilverBullet
space, ...) and hands normalized items back to the kernel. It is written
for a reader with no access to this repository beyond four things:

- this file,
- `proto/topos/v1/plugin.proto` (the wire contract),
- the `sdk` Go module (`github.com/davison/topos/sdk`), and
- `plugins/mock` — a complete, working reference plugin built from
  exactly these four inputs and nothing else (`PLUG-05`; see "Build your
  first plugin", below).

If those four are all you have, you should be able to write a working
plugin — `plugins/mock` is the proof: it was built and validated against
this document with no access to any real-source plugin implementation.

**Other implementations (aside, not required reading):** three fuller,
real-source examples also ship in this repository —
`plugins/paperless` (a REST API source), `plugins/silverbullet` (an
HTTP-with-frontmatter source), and `plugins/signal` (a **local-path**
source: no network endpoint at all, reads a local Signal Desktop database
file directly) — useful once you're past "Build your first plugin" and
want to see how a plugin structures a real HTTP client or a local-file
source, but none of the three is needed to understand or apply anything
in this document.

## A plugin is read-only by construction

topos never mutates a source. This is not a policy a plugin author is
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

One thing the contract does **not** give you: containment. A plugin is a
regular native binary launched as a subprocess with the full local OS
access of the user who runs the kernel — `hashicorp/go-plugin` is a
transport, not a sandbox. The read-only shape above constrains what the
*kernel* can ask a plugin to do; it does not constrain what a plugin
binary can do on its own. Installing a third-party plugin is therefore
the same trust decision as installing the kernel binary itself: only run
plugin binaries you built yourself or whose source you trust.

## Depending on the SDK

A plugin is a separate Go module (or, if the source plugin's language
support ever expands, a separate binary speaking the same gRPC contract)
that imports `github.com/davison/topos/sdk`:

```go
import (
	"github.com/davison/topos/sdk"
	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)
```

The SDK module is the stable Go-native surface: it re-exports the
handshake config, the plugin map, and a `plugin.GRPCPlugin` adapter that
wires a Go implementation of the `SourcePlugin` interface to the generated
gRPC stubs. You implement `sdk.SourcePlugin`, not the raw generated gRPC
server type:

```go
type SourcePlugin interface {
	Describe(ctx context.Context, req *toposv1.DescribeRequest) (*toposv1.DescribeResponse, error)
	Match(ctx context.Context, req *toposv1.MatchRequest) (*toposv1.MatchResponse, error)
	Fetch(ctx context.Context, req *toposv1.FetchRequest) (*toposv1.FetchResponse, error)
	Health(ctx context.Context, req *toposv1.HealthRequest) (*toposv1.HealthResponse, error)
}
```

A plugin's `main` package registers that implementation and serves it.
Note the import alias below: `goplugin "github.com/hashicorp/go-plugin"`
— **not** the unrelated Go standard-library `plugin` package
(`plugin.Open`, for loading `.so` shared objects), which shares the bare
name `plugin` but has nothing to do with this contract. Every plugin in
this repository (`plugins/mock/main.go` included) uses this exact alias:

```go
import (
	goplugin "github.com/hashicorp/go-plugin"

	"github.com/davison/topos/sdk"
)

goplugin.Serve(&goplugin.ServeConfig{
	HandshakeConfig: sdk.Handshake,
	Plugins: map[string]goplugin.Plugin{
		"source": &sdk.SourcePluginGRPCPlugin{Impl: &myPlugin{}},
	},
	GRPCServer: sdk.GRPCServer, // raises the gRPC message-size ceiling — see Fetch, below
})
```

## Handshake and the plugin-map key

The kernel and every plugin share one handshake, `sdk.Handshake`:

| Field | Value |
|---|---|
| `ProtocolVersion` | `2` |
| `MagicCookieKey` | `TOPOS_PLUGIN` |
| `MagicCookieValue` | `topos-source-plugin-v1` |

`ProtocolVersion` is bumped only for a breaking wire-protocol change (not
for an additive contract change — that's what `DescribeResponse`'s
`contract_version` field is for; see Describe, below). A plugin that
serves a different magic cookie or protocol version fails the handshake
outright, before any RPC is attempted.

`ProtocolVersion` moved from `1` to `2` in this contract generation
because `MatchRequest`'s shape changed from a flat `keywords` list to a
typed `match_fields` map (see Match, below) — a breaking wire change, not
an additive one. This is the deliberate fail-fast for that kind of break:
a plugin binary built against `ProtocolVersion` 1 fails cleanly at the
handshake, before `Describe` or `Match` is ever called, rather than
confusingly at its first `Match` call with an empty or misinterpreted
match map. `DescribeResponse.contract_version` is the complementary,
finer-grained signal: it names the contract *generation* (`"topos.v2"` as
of this break) independently of the proto package path, which stays
`topos.v1` — see Describe, below, for why those two strings are not the
same thing and must not be confused.

Every plugin registers its implementation under the plugin-map key
**`"source"`** — this must match exactly, on both the plugin side
(`plugin.ServeConfig.Plugins`) and implicitly on the kernel side (the
kernel always dispenses `"source"`).

## Discovery and launch

The kernel discovers plugins by scanning a plugins directory for binaries,
not from a compile-time list — this is what lets a kernel build ship
without a Signal-plugin's cgo/C-toolchain requirement for a user who
doesn't configure Signal. As of this contract generation the kernel scans
**two** directories, not one — see "Trust tiers and pinning", below, for
the full shape; the short version is that `[plugins] dir` (default
`plugins`, resolved relative to the running `topos` executable) is
unchanged, and a second, independently configured directory
(`[plugins] external_dir`) is scanned alongside it, with a binary's
directory of origin determining how much the kernel — and, downstream, its
own UI — trusts it.

For each configured `[sources.<name>]` entry, the kernel launches
`<resolved-dir>/<plugin-binary-name>` as a subprocess and negotiates the
handshake over gRPC only — `AllowedProtocols` is restricted to
`[]plugin.Protocol{plugin.ProtocolGRPC}`, so a plugin implementing only
the legacy net/rpc transport will fail to connect. Immediately after a
successful handshake, the kernel calls that plugin's `Describe` RPC and
uses the returned `source_type` as the plugin's identity for the rest of
the process's lifetime — **a plugin's identity is never trusted from its
filename or its config key**, only from what `Describe` reports.

**The kernel may launch the same plugin binary more than once.** Every
`[sources.<id>]` entry in config gets its own subprocess, its own
handshake, and its own `WEBSPACES_SOURCE_CONFIG` (see below) — two
entries pointing at the same `plugin = "topos-plugin-proton"` binary (a
"home-email" instance and a "work-email" instance, say) launch as two
independent subprocesses with two independent connections, sync
histories, and index rows. The `[sources.<id>]` config map key `<id>` is
the **instance identity** the kernel uses everywhere identity matters:
it prefixes every item's stable id, keys every sync-run record, gates
every `/agent/v1` grant, and is what the kernel's HTTP API reports as
`source` on every item and `name` on every `GET /api/sources` entry (see
`docs/api.md`). A plugin **never learns, asserts, or needs its own
instance identity** — it still declares only its `source_type` via
`Describe`, exactly as before this phase, and has no way to observe
which `[sources.<id>]` key the kernel launched it under. Identity lives
entirely on the kernel side of the process boundary.

## Trust tiers

The kernel scans two plugin directories, never one:

- **Trusted** — `[plugins] dir`, default `plugins`, the directory this
  repository's own `make build`/`make dev` populates. A binary resolved
  from here is `pluginhost.TierTrusted`.
- **External** — `[plugins] external_dir`. A binary resolved from here is
  `pluginhost.TierExternal`. Omitted (the common case — most operators
  never set this key), it defaults to a per-OS platform data directory
  with no config required: `$XDG_DATA_HOME/topos/plugins-external`
  (falling back to `~/.local/share/topos/plugins-external` when
  `XDG_DATA_HOME` is unset) on Linux, `~/Library/Application
  Support/topos/plugins-external` on macOS, and
  `%LOCALAPPDATA%\topos\plugins-external` on Windows. The kernel never
  creates this directory itself — an operator who never drops a binary
  there simply has an empty external tier, which is a legitimate,
  unremarkable state, not an error.

**Tier is derived exclusively from which directory a binary resolved
from, launch time — never from anything the plugin itself declares.**
There is no `Describe`-reported "I am trusted" field, no signature, no
allowlist of known-good `source_type` values: the kernel decides tier
purely by asking "which directory did I find this binary's bytes in,"
and that answer is set once at launch and never re-derived from a live
RPC afterward. This is a structural choice, not an oversight — a plugin
process is not a trustworthy witness to its own trustworthiness.

**The shadow rule.** If a binary of the same filename exists in BOTH
directories, the kernel launches the trusted copy and ignores the
external one, logging a named warning identifying the shadowed binary. A
binary can never impersonate a trusted plugin by choosing a colliding
filename in the external directory — the trusted directory always wins,
silently to the running kernel (loudly to its own logs), never the other
way around.

**`[sources.<id>] plugin` must be a bare binary filename.** The value
resolves directly inside one of the two configured directories above and
nowhere else — it must not contain a path separator (either `/` or `\`)
or an `.`/`..` segment, and it must not be empty. A value carrying any of
those shapes fails config load and `PUT /api/config` with an error naming
the offending source, before the config is written to disk. This is what
keeps the premise "trust is decided by the kernel from where the binary
lives" true: a caller-supplied path can never point the kernel at a file
outside `[plugins] dir`/`[plugins] external_dir`, and so can never make a
binary outside either directory launch as if it were `TierTrusted`.

**Two instances, two tiers, no conflict.** Two `[sources.<id>]` entries
naming the same plugin binary always resolve to the same tier (tier is a
property of the binary's resolved path, not of the instance), but two
DIFFERENT plugin binaries can freely sit in different tiers — a
deployment mixing this repository's own trusted paperless/SilverBullet/
Proton/Signal/WhatsApp plugins with a third-party external one is the
expected shape this whole mechanism exists to support.

## Pinning

An external-tier binary is additionally **content-pinned**: the kernel
records the SHA-256 of its bytes in `[plugins.pins]`, keyed by binary
name (e.g. `"topos-plugin-example"`), the first time an operator adds a
source using that binary — the kernel's own confirm-before-trust flow
writes this table; hand-editing it is never required (though a hand-edit
is honored exactly like any other config key). Every subsequent launch of
that binary **re-verifies** the pin: the kernel recomputes the on-disk
SHA-256 immediately before `exec`, and refuses to launch — never
executing the binary at all — if the two hashes disagree.

**Trusted-tier binaries are never pinned.** This repository's own
`make build`/`make dev` rebuilds every trusted binary constantly during
normal development; pinning one would false-alarm on every rebuild for no
security benefit (a trusted-directory binary already IS the operator's
own build). Pinning applies to the external tier exclusively.

**What a pin mismatch looks like operationally.** A pin mismatch is a
SOFT, per-instance failure, never a hard kernel-boot or config-save
failure: every other configured source — trusted or external, pinned or
not — still boots and syncs normally. The one mismatched instance is
simply refused launch, named, with both the pinned and the current hash
recorded; it surfaces as a real, visible entry on `GET /api/sources`
(never a silent omission — see `docs/api.md`'s `launch_failure` field)
carrying a `pin_mismatch` reason the kernel's own UI renders as a
re-pin ("Trust updated binary") action, repairable per source from that
same source's own menu. An external binary with NO recorded pin at all is
treated identically to a mismatch — there is no "unpinned, launch
anyway" state for the external tier.

**A pin proves sameness, not provenance.** A matching SHA-256 tells you
"these are the exact same bytes the kernel saw and you accepted before" —
it says nothing about who built those bytes, whether the source code they
came from is trustworthy, or whether the binary does what its name
implies. There is no signature verification, no publisher identity, no
supply-chain attestation anywhere in this design. If a third-party plugin
author publishes a checksum through their own release process, verify it
yourself, independently of this mechanism, before you first add the
source — the kernel's pin only protects you AFTER that first informed
decision, by detecting any later, unexpected change to the bytes it
already trusted.

## The launch environment

A launched plugin subprocess receives **exactly** the following
environment, and nothing else — it must not expect to inherit anything
from the kernel process's own environment beyond what is listed here:

1. **A fixed, documented desktop-session allowlist**, copied present-only
   (an unset allowlisted variable contributes nothing, never an
   empty-string entry): `PATH`, `HOME`, `LANG`, `LC_ALL`, `LC_CTYPE`,
   `TZ`, `TMPDIR`, `XDG_RUNTIME_DIR`, `DBUS_SESSION_BUS_ADDRESS`. The
   first seven cover the ordinary needs of any subprocess (locating
   binaries, resolving `~`, consistent locale/timezone/scratch-space
   handling); the last two are desktop-session plumbing the Signal
   plugin's Secret Service key retrieval requires — session ADDRESSES,
   not secret values.
2. **The value behind every `${VAR}`/`$VAR` reference THIS instance's own
   raw config actually declares** — scanned from the instance's
   `[sources.<id>]` block (including any `extras` table, below), and
   nothing else. A variable set in the kernel process's own environment
   but referenced nowhere in this instance's own config is never copied
   in, no matter what it happens to be named: the kernel's remaining
   environment (any credential, any unrelated secret sitting in the
   operator's shell) is structurally invisible to a plugin subprocess
   that never referenced it.
3. **`WEBSPACES_SOURCE_CONFIG`** (always) and **`WEBSPACES_DESCRIBE_ONLY=1`**
   (trial launches only) — see "Configuration", below.

Nothing else reaches the subprocess. The kernel never passes its own
`os.Environ()` through wholesale — the allowlist above is enforced at the
process-launch boundary itself (`goplugin.ClientConfig.SkipHostEnv`), not
merely by constructing a restricted `exec.Cmd.Env` and hoping the
transport respects it.

**This is disclosure and least-hand-over, not containment.** Everything
in "A plugin is read-only by construction," above, still applies without
qualification: topos does not sandbox plugins. A plugin binary launched
by the kernel is a regular native process with the full local OS access
of the user who runs the kernel — it can read any file its OS-level
permissions allow, open any socket, and see the desktop session
addresses listed above directly off the filesystem regardless of what
this allowlist withholds. What this section describes is a deliberate,
honest reduction of what the kernel HANDS the subprocess through its own
environment variables — never a claim that the subprocess cannot reach
further on its own. Installing a third-party plugin remains exactly the
same trust decision as installing the kernel binary itself: only run
plugin binaries you built yourself or whose source you trust, tiers and
pins notwithstanding.

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

**The `path` key — a local-path source, no network endpoint at all.** A
source that reads a local file or directory rather than a remote API
declares `path` instead of `base_url`/`token`: the local filesystem
location that source reads from. `plugins/signal` is this repository's
reference implementation of the shape — its config is just

```json
{ "path": "~/.config/Signal" }
```

with no `base_url`, `token`, or any credential at all, because the
"connection detail" a local-path source needs is a filesystem location,
not network coordinates. A source declaring `path` is exempt from the
`base_url`/`token` requirement every other source must satisfy; a source
declaring none of `base_url`, `token`, or `path` still fails config load
— every source must declare at least one recognized connection-detail
shape. `~` in a `path` value is expanded by the plugin itself, not by the
kernel (`kernel/pluginhost` passes the configured string through
unexpanded); see `plugins/signal/README.md` for the fully worked example.

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

**`extras` — a nested object for provider-specific settings the kernel
has no built-in field for.** An operator declares these under
`[sources.<id>.extras]` in `config.toml`:

```toml
[sources.example]
plugin = "topos-plugin-example"
base_url = "${EXAMPLE_URL}"
token = "${EXAMPLE_TOKEN}"

[sources.example.extras]
region = "${EXAMPLE_REGION}"
project_id = "acme-prod"
```

and they reach the subprocess nested inside `WEBSPACES_SOURCE_CONFIG`, as
an `extras` object alongside the top-level keys:

```json
{
  "base_url": "https://example.lan",
  "token": "abc123...",
  "extras": { "region": "eu-west", "project_id": "acme-prod" }
}
```

Every `extras` value is always a **string** — the kernel has no
type-inference over this table, and holds no built-in knowledge of what
any given extras key means (D-12). A `${VAR}`/`$VAR` reference inside an
extras value expands from the environment exactly like `base_url` or
`token` does — the identical `os.Expand` pass, run before your plugin
ever sees the JSON. The `extras` key is **omitted entirely** (never an
empty object) when a source declares no extras at all, so your plugin's
own JSON decode sees "no extras configured" unambiguously rather than an
empty-vs-absent case to special-case.

## RPC semantics

### `Describe`

Called once, immediately after the handshake, before any other RPC.
Returns the plugin's identity.

**The kernel may call `Describe` against your plugin before any source
using it is ever persisted.** The webspace builder's add-source flow
trial-launches your binary — full handshake, one `Describe` call, then
kills the subprocess — against connection fields the operator has typed
but not yet saved, so it can show them the resulting tier, match
vocabulary, and (for the external tier) the binary's own hash before
anything reaches `config.toml`. This is the ONLY way an operator can
learn an external binary's identity/hash before a pin can exist for it
(a `WEBSPACES_DESCRIBE_ONLY=1` marker is set on this launch's own
environment — see "The launch environment", above). A well-behaved
plugin's `Describe` implementation is idempotent and side-effect-free
regardless of whether it is ever called this way or as part of a real,
persisted launch — nothing about this contract distinguishes the two
call sites at the RPC level.

```protobuf
message DescribeResponse {
  string source_type      = 1;  // e.g. "paperless" — the kernel's only
                                  // trusted source of this plugin's identity
  string display_name     = 2;  // e.g. "paperless-ngx" — for UI/logs
  string contract_version = 3;  // e.g. "topos.v2"
  repeated string match_vocabulary = 4;
  bytes  icon              = 5;  // small square SVG or PNG, <= 64KB
  string icon_mime         = 6;  // "image/svg+xml" or "image/png"
  repeated ExtrasField extras = 7;  // OPTIONAL: declared provider-specific
                                      // config.Source.Extras keys (D-15)
}

message ExtrasField {
  string key         = 1;  // exact Extras map key, e.g. "region" — never a label
  string label       = 2;  // human-readable form-input label
  bool   required    = 3;  // advisory only — the kernel never enforces this
  bool   secret      = 4;  // hint: render the form input masked
  string placeholder = 5;  // DISPLAY-ONLY — never pre-filled into a saved value
}
```

`icon`/`icon_mime` (Phase 9, 09-UI-SPEC.md Fix 10) are the plugin's own
declared identity icon, additive fields appended after
`match_vocabulary`. `icon` is a small square SVG or PNG of at most 65536
(64KB) bytes; empty means the plugin declares no icon. `icon_mime` is
either `"image/svg+xml"` or `"image/png"`, and is the empty string if and
only if `icon` is empty — the kernel drops (never truncates) an icon whose
mime is empty, unset, or outside that two-value allowlist, treating it
identically to "no icon declared." The kernel captures both fields at the
same `Describe` call site it already makes (no new RPC), caches them per
launched plugin binary, and serves them at
`GET /api/plugins/{plugin_binary}/icon`. Because this is a proto3
additive change, a plugin built against the pre-Phase-9 contract simply
never sets these two fields — it keeps working completely unchanged, with
no handshake break and no `sdk.Handshake.ProtocolVersion` bump.

`contract_version` is the additive-compatibility signal: a plugin built
against an older but still-compatible revision of this contract can report
that revision here without triggering a handshake-level `ProtocolVersion`
bump. `contract_version` names the contract *generation* (`"topos.v2"`
as of this phase's typed-match-field break), versioned independently of
the proto package path, which stays `topos.v1` — a plugin built against
the pre-Phase-5 contract also reports `"topos.v1"` as its proto package,
so `contract_version`, not the package name, is what a reader compares to
know which `MatchRequest` shape a plugin expects. In practice this
distinction rarely matters to a plugin author: the handshake's
`ProtocolVersion` (see above) is the actual fail-fast for a breaking
change like this one, so a plugin built against the wrong `MatchRequest`
shape never reaches the point of returning `contract_version` at all.

`match_vocabulary` is the field-name vocabulary this plugin's `Match` RPC
reads from `MatchRequest.match_fields` (see Match, below) — declared by
the plugin itself, not looked up in any kernel-side table of known
plugin types (D-05). The kernel validates every operator-configured match
field against this list at startup and **fails startup by name** — naming
the offending field, the webspace, the instance, this plugin's binary,
and the vocabulary it does declare — the moment it finds a config entry
naming a field this plugin didn't declare here. A plugin declaring an
empty `match_vocabulary` can never participate in matching: the kernel
also fails startup if a webspace relies on the keywords fallback (see
Match, below) for an instance whose plugin declared zero fields, since
there is nothing for that fallback to fan into. The four vocabularies
declared by this repository's in-repo plugins — `["folders"]` (proton),
`["tags"]` (paperless), `["tags", "pages"]` (silverbullet),
`["conversations"]` (signal) — are illustrations of the shape, not a
closed set: a future plugin type declares whatever field names make sense
for its own source system's native categorization, with no proto change
required.

`extras` (D-15) is an OPTIONAL, additive declaration of provider-specific
config keys your plugin expects beyond the built-in `base_url`/`token`/
`path`/etc. fields — declaring nothing is fully supported (the kernel's
add-source UI falls back to a free-form key/value editor that still lets
an operator supply any extras key your plugin's own documentation names,
whether or not you declared it here). Each `ExtrasField` names one
`[sources.<id>.extras]` key (`key`, matched verbatim, never a display
label), a human-readable `label` for the kernel's own add-source form,
whether the form should treat it as `required` (advisory only — the
kernel never rejects a saved config missing a "required" extras key,
since your plugin's own requirements are not something
`kernel/config.Validate` enforces), whether the form should render the
input `secret` (masked, matching the treatment `base_url`/`token` already
receive), and an optional `placeholder` hint. **`placeholder` is
display-only and is NEVER pre-filled into a value the kernel then
saves** — this is deliberate (T-11-11): a malicious `Describe` response
suggesting its own default (e.g. `"${SOME_SECRET}"`) can never get
silently persisted into an operator's `config.toml` and later expanded
just because it looked plausible in an empty form field. A field carrying
an empty `key` is dropped entirely by the kernel before it ever reaches
any form — a plugin cannot use this mechanism to inject a nameless field.

### `Match`

Called only at sync time, never at request time (item-open). Unlike the
pre-Phase-5 contract, the kernel does not pass a flat, undifferentiated
keyword list — it passes a **typed field map**, scoped to exactly this one
source instance's own resolved match configuration for the one webspace
being synced:

```protobuf
message StringList { repeated string values = 1; }

message MatchRequest {
  map<string, StringList> match_fields = 2;
}
message MatchResponse { repeated Item items = 1; }
```

`StringList` exists only because proto3 map values cannot themselves be a
`repeated` field — it's a thin wrapper, nothing more. Each key in
`match_fields` is one entry from this plugin's own declared
`match_vocabulary` (see Describe, above); each value is the list of
strings that field must match against. A `MatchRequest` carries **only
this one instance's own fields** — never another instance's match
configuration, even when two instances of the same plugin type are
configured and one webspace matches both differently.

A plugin implements `Match` against three rules:

1. **Read only the keys you declared.** A key present in `match_fields`
   that your plugin did not list in its own `match_vocabulary` must be
   treated as **absent, never as an error** — the kernel already
   validated every configured field name against your declared vocabulary
   at startup (D-05), so a key your plugin doesn't recognize here would
   only occur if a *different* instance's field name happened to collide,
   which your plugin has no business inspecting.
2. **Match exact and case-insensitive, never substring or prefix (D-04).**
   Comparison is against the source's own native categorization (a
   paperless-ngx tag name, an IMAP folder/label, a chat conversation name,
   a SilverBullet page tag): `house` must match a tag literally named
   `House`, and must **not** match a tag named `Household`. There is no
   Unicode normalization — keep your source's spelling and the operator's
   configured spelling consistent. If a silo names something differently
   than the webspace's primary term, the fix is adding that variant
   string to the relevant field's value list in config — there is no
   per-source override syntax, and a plugin must not invent its own
   fuzzy-matching behavior to compensate.
3. **An empty value list for a declared key matches nothing for that
   field, never everything.** A `match_fields["tags"]` entry present with
   zero values (or absent entirely) means "this field contributes no
   matches" — it is not a wildcard.

**Worked example** — `plugins/mock`'s `Match` (the full file is
`plugins/mock/plugin.go`) has a fixed, in-memory item set instead of a
real source system to query, but the matching rule itself is identical to
what a real plugin must implement. The mock declares a one-field
vocabulary, `matchVocabulary = []string{"labels"}`, and reads only that
key:

```go
func (p *SourcePlugin) Match(_ context.Context, req *toposv1.MatchRequest) (*toposv1.MatchResponse, error) {
	keywords := req.GetMatchFields()["labels"].GetValues()
	var items []*toposv1.Item
	for _, it := range mockItems {
		if labelsMatchAnyKeyword(it.GetLabels(), keywords) {
			items = append(items, it)
		}
	}
	return &toposv1.MatchResponse{Items: items}, nil
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

A real plugin with more than one declared field (SilverBullet declares
`["tags", "pages"]`, for example) reads each key independently and unions
the results — a page matches if its tags match any configured `tags`
value OR its page name matches any configured `pages` value; the two
fields are never combined into one comparison.

A real plugin's `Match` typically has one more step before the comparison
above: resolving each value against the source system's own
categorization API (an HTTP call to look up a tag by name, an IMAP `LIST`
to find a matching folder, ...) before it can even ask "which items carry
this categorization" — the mock skips that step because its
"categorization" (`Labels`) is already in memory. Whatever that
resolution step looks like for your source, return a gRPC
`codes.Unavailable` status (not a partial, silently-empty result) when the
source system cannot be reached — the kernel records this per-source in
that sync run's status and surfaces it as `source_unavailable`-shaped
state, rather than treating "the source is down" the same as "nothing
matched."

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
  ContentShape content_shape      = 8;  // REQUIRED whenever mime_type is
                                         // "text/html" — see below
}
```

**`data` for a `text/html` rendition is an unwrapped, unthemed,
unsanitized fragment (D-11).** This is a deliberate move: presentation
used to be each plugin's own job (its own sanitize policy, its own theme
stylesheet, its own document-wrapping helper), and that meant a theme
change touched every plugin and put sanitization outside the trust
boundary once plugins are third-party. As of this contract generation,
the kernel owns the entire sanitize/wrap/theme pipeline at its own
content-serving boundary (`kernel/httpapi/rendition.go`), and a plugin's
job is reduced to two things: return the bare content fragment (no
`<html>`/`<head>`/`<body>` wrapper, no `<style>` block, no inline theme
colors), and declare which of the kernel's rendition profiles that
fragment needs via `content_shape`:

```protobuf
enum ContentShape {
  CONTENT_SHAPE_UNSPECIFIED       = 0;
  CONTENT_SHAPE_EMAIL_HTML        = 1;
  CONTENT_SHAPE_CHAT_TRANSCRIPT   = 2;
  CONTENT_SHAPE_MARKDOWN_HTML     = 3;
}
```

`content_shape` is **required whenever `mime_type` is `"text/html"`** —
the kernel refuses to serve a `text/html` rendition whose `content_shape`
is `CONTENT_SHAPE_UNSPECIFIED` (the zero value fails closed, exactly like
`LinkFidelity` and `ContentVariant`), returning `unsupported_content_shape`
on its own HTTP surface (see `docs/api.md`) rather than ever guessing a
policy or serving an unsanitized document from its own origin. The field
is ignored for every other `mime_type`. A plugin author adding a new kind
of HTML content this contract doesn't yet have a shape for cannot simply
invent one — `CONTENT_SHAPE_UNSPECIFIED` behaves as a load-bearing refusal,
not a permissive default, so a genuinely new shape requires a contract
change (a new enum value plus a matching policy in `rendition.go`), not a
plugin-side workaround.

A plugin **must not** emit a full HTML document (no `<!doctype>`, no
`<html>`/`<head>`/`<body>` tags), and **must not** author its own
stylesheet or embed inline theme colors — both are now the kernel's job,
applied uniformly across every plugin so a theme change is a one-place
edit instead of an N-plugin one. A plugin's only sanitization
responsibility is structural: if your fragment interpolates content the
source system doesn't already guarantee is well-formed markup (message
text into a chat bubble, for example), escape it (`html.EscapeString` or
equivalent) so it can't forge the surrounding structural markup your
plugin itself emits — the kernel's sanitizer is the actual security
boundary, but escaping your own interpolation is still your
responsibility, the same "structural-integrity guarantee" `plugins/signal`
implements for its transcript fragments.

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
| `source_id` | **Required** | Stable within your plugin — the kernel derives its own global id as `"{source}:{source_id}"`, where `source` is the config-authored source **instance** id (the `[sources.<id>]` map key this item synced through), never `source_type` (see "Discovery and launch", above, and `docs/api.md`'s "The stable-ID scheme"). Never reuse a `source_id` for two different underlying objects within your plugin — the instance-id prefix disambiguates across instances, not within one. |
| `source_type` | **Required** | Must exactly match what your `Describe` RPC reports — the kernel doesn't trust a value here that disagrees with `Describe`. Retained as descriptive provenance only; never used to key identity anywhere in the kernel. |
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
	"plugin":            "topos-plugin-<yours>",
	"contract_version": contractVersion,       // matches Describe's contract_version
},
```

### The `file://` local-path deep-link convention

A plugin whose items are local files (a source declaring `path` in its
own configuration — see "Configuration", above) sets `deep_link` to a
`file://` URI over the item's real absolute path, rather than any other
scheme. The kernel rewrites a `file://`-scheme `deep_link` at serve time
into a loopback route, `POST /api/items/{id}/open` (`docs/api.md`), which
execs the desktop's own file-association handler (`xdg-open` on Linux)
against the path it independently resolves from its own index state and
configuration — **never from the plugin-supplied URI itself**, which is
used as a marker only. Every other `deep_link` scheme (`https://`, and
any future scheme) is served unchanged, exactly as the plugin returned
it.

**The trigger is the URL scheme alone, never the plugin's declared
`source_type`.** The kernel holds no table of known plugin types (D-05,
above) and checks nothing beyond `strings.HasPrefix(deep_link, "file://")`
— a third-party local-path plugin gets kernel-mediated, desktop-native
"Open in …" behavior automatically, with zero kernel code change, simply
by emitting an honest `file://` URI. There is no opt-in flag, no
`Describe`-declared capability, and no contract version bump associated
with this convention: it is available to any plugin, in-repo or
out-of-repo, today.

Building the URI is entirely the plugin's own responsibility — join your
configured root with the item's own `source_id` (a forward-slash relative
path, per the `Item` table above) and encode it as `file://` plus the
resulting absolute path. `plugins/filesystem` (`item.go`'s
`fileDeepLink`) is the reference implementation: `"file://" +
filepath.ToSlash(filepath.Join(root, sourceID))`. The kernel's own
re-resolution on the open route re-validates the joined path stays inside
the configured root before ever exec'ing anything, and that re-validation
resolves symlinks: it calls `filepath.EvalSymlinks` on both the joined
path and the configured root, compares the RESOLVED pair, and fails
closed when resolution is impossible — so a file indexed legitimately and
later swapped on disk for a symlink pointing outside the root is refused
rather than followed. This is the same defense-in-depth join-resolve-and-
validate discipline your own plugin must apply when resolving `source_id`
back to a real path for `Fetch` — `plugins/filesystem`'s `resolvePath`
(`item.go`) is the reference implementation for that side too. **The
resolved path is also the path the kernel actually hands to the desktop
handler**, not the lexical join it validated it against — read and exec
the same resolved path you validate, rather than re-walking symlinks
implicitly on the read/exec call itself, so the path your plugin's own
containment check approves and the path it actually opens are always one
and the same.

A path-based check like this one `narrows but does not eliminate` the
race between resolution and the syscall: a single remaining window exists
between `filepath.EvalSymlinks` returning and the following `os.Open`/exec
call, in which the resolved path's own final component could in principle
be swapped again. Fully eliminating it requires descriptor-based traversal
(`openat`/`O_NOFOLLOW`), which topos does not currently do — no plugin or
kernel code in this repo should be built assuming a stronger guarantee
than that.

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

## `ContentShape`

```protobuf
enum ContentShape {
  CONTENT_SHAPE_UNSPECIFIED       = 0;
  CONTENT_SHAPE_EMAIL_HTML        = 1;
  CONTENT_SHAPE_CHAT_TRANSCRIPT   = 2;
  CONTENT_SHAPE_MARKDOWN_HTML     = 3;
}
```

See the `Fetch` section above for the full explanation. In short:
`content_shape` tells the kernel which of its three sanitize/wrap/theme
profiles a `text/html` `FetchResponse.data` fragment needs, and is
required whenever `mime_type` is `"text/html"`. `CONTENT_SHAPE_UNSPECIFIED`
is the zero value and — like `LinkFidelity_LINK_FIDELITY_UNSPECIFIED`
above — is never a valid declaration for `text/html` content: the kernel
refuses to serve it, returning `unsupported_content_shape` rather than
guessing. Currently three plugins in this repository declare a
`content_shape`: `plugins/proton` (`CONTENT_SHAPE_EMAIL_HTML`),
`plugins/silverbullet` (`CONTENT_SHAPE_MARKDOWN_HTML`), and
`plugins/signal` (`CONTENT_SHAPE_CHAT_TRANSCRIPT`) — `plugins/paperless`
and `plugins/mock` never serve a `text/html` rendition at all (paperless
serves PDF/image; mock has no rendition to offer), so the zero value is
correct, unused, for both.

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
go mod init github.com/davison/topos/plugins/yourplugin
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
your `SourcePlugin` implementation, and call `goplugin.Serve` with
`sdk.Handshake`, your implementation registered under the `"source"` key,
and `sdk.GRPCServer` (see "Depending on the SDK", above, for the exact
shape and the `goplugin` import alias).

**5. Build it.**

```
CGO_ENABLED=0 go build -o bin/plugins/topos-plugin-yourplugin ./plugins/yourplugin
```

**6. Configure the kernel to launch it.** The kernel's config file
(`~/.config/topos/config.toml` by default; `config.example.toml` in
this repository is a fully-commented reference for every key) needs two
blocks: one `[sources.<name>]` entry naming your plugin, and one
`[webspaces.<name>]` entry with a keyword your `Match` implementation
will actually return an item for. The minimal shape, self-contained here
so you don't need any file beyond this document to write it:

```toml
[sources.yourplugin]
plugin = "topos-plugin-yourplugin"   # your binary's filename, resolved inside [plugins] dir (default "plugins")
# ... plus whatever connection-detail keys your plugin's own main.go
# reads out of WEBSPACES_SOURCE_CONFIG (see "Configuration", above) — a
# plugin with nothing to configure, like plugins/mock, needs none.

[webspaces.demo]
keywords = ["your-keyword-here"]   # must exactly, case-insensitively match something your Match returns
```

`keywords` here is the webspace-level fallback (D-01): with no explicit
`[webspaces.demo.match.yourplugin]` block, the kernel fans this one list
across every field in your plugin's declared `match_vocabulary` and sends
the result as `match_fields` on every `Match` call for this instance. This
is the minimal shape to get a first plugin running; `config.example.toml`
in this repository is the complete worked reference for the typed,
per-instance `[webspaces.<name>.match.<instance>]` shape, including two
instances of one plugin type and a participation allowlist.

Every dotted-table key here (`[sources.<name>]`, `[webspaces.<name>]`) is
a plain TOML table — nothing plugin-specific about the file format
itself, only the key set under `[sources.<name>]`, which your own plugin
defines and documents (see "Configuration", above).

**7. Run it.** `topos sync` (a one-shot sync) or `topos serve`
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
