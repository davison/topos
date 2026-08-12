# Architecture Research: v1.1.0 "Plugin Ecosystem" Integration

**Domain:** Integration of 5 new features into topos's existing Go-kernel +
gRPC-plugin + SvelteKit-SPA architecture
**Researched:** 2026-08-12
**Confidence:** HIGH (grounded directly in the current codebase — `kernel/pluginhost`,
`kernel/httpapi`, `kernel/index`, `kernel/config`, `internal/audit`,
`docs/plugin-contract.md`, `docs/api.md` — not external ecosystem survey)

This file answers one question only: **how do the five v1.1.0 features
attach to the architecture that already exists**, file by file, table by
table. It is deliberately not a greenfield "standard architecture for this
domain" document — topos already has a standard architecture; this is the
integration diff against it.

## System Overview

```
┌───────────────────────────────────────────────────────────────────────┐
│  SvelteKit SPA (web/src) — embedded via kernel/webui go:embed          │
│  ┌────────────────┐  ┌───────────────────┐  ┌───────────────────────┐ │
│  │ AddSourceModal  │  │ StreamItem row     │  │ NEW: manifest.json /  │ │
│  │  + trust badge  │  │  + mark toggle     │  │  service-worker.js    │ │
│  │  (NEW)          │  │  (NEW)             │  │  (NEW, PWA)           │ │
│  └────────┬────────┘  └─────────┬──────────┘  └───────────┬───────────┘ │
└───────────┼─────────────────────┼──────────────────────────┼───────────┘
            │ PUT /api/config     │ PUT /api/webspaces/{ws}   │ served as
            │ GET plugin-types    │  /items/{id}/mark (NEW)   │ static assets
            ▼                     ▼                            ▼
┌───────────────────────────────────────────────────────────────────────┐
│  kernel/httpapi                                                        │
│  ┌───────────────┐ ┌────────────────┐ ┌───────────────────────────┐  │
│  │ config.go      │ │ stream.go /     │ │ NEW: marks.go              │  │
│  │  +trust field  │ │ search.go       │ │  PUT/DELETE mark handler   │  │
│  │  in plugin-    │ │  +mark-aware    │ │                             │  │
│  │  types resp.   │ │  query          │ │                             │  │
│  └───────┬────────┘ └────────┬────────┘ └──────────────┬──────────────┘ │
└──────────┼───────────────────┼──────────────────────────┼──────────────┘
           │                   │                            │
           ▼                   ▼                            ▼
┌────────────────────┐ ┌──────────────────┐  ┌────────────────────────┐
│ kernel/pluginhost    │ │ kernel/index       │  │ kernel/index            │
│  discover_binaries.go│ │  Store.StreamItems │  │  NEW: item_marks table  │
│  +external_dir scan  │ │  (query reshaped)  │  │  (exempt from D-07      │
│  +trust-by-origin     │ │                    │  │  rebuild-drop)          │
└──────────┬───────────┘ └────────────────────┘  └─────────────────────────┘
           │ launches subprocess over go-plugin/gRPC (topos.v1 proto, handshake v2)
           ▼
┌───────────────────────────────────────────────────────────────────────┐
│  Plugin subprocesses (native binaries, full OS access — no sandbox)    │
│  ┌───────────┐┌────────────┐┌─────────┐┌────────┐┌──────────────────┐ │
│  │ paperless ││silverbullet││ proton  ││ signal ││ whatsapp          │ │
│  │(in-repo,  ││(in-repo,   ││(in-repo)││(cgo,   ││(in-repo)          │ │
│  │ trusted)  ││ trusted)   ││trusted) ││trusted)││trusted)           │ │
│  └───────────┘└────────────┘└─────────┘└────────┘└──────────────────┘ │
│  ┌────────────────────────┐   ┌──────────────────────────────────────┐│
│  │ NEW: filesystem          │   │ NEW: gdrive (built OUT-OF-REPO,       ││
│  │ (in-repo, trusted,       │   │ separate git repo/module, imports     ││
│  │  local-path source like  │   │ github.com/davison/topos/sdk,         ││
│  │  plugins/signal)         │   │ dropped in external_dir → untrusted)  ││
│  └────────────────────────┘   └──────────────────────────────────────┘│
└───────────────────────────────────────────────────────────────────────┘
```

Every box labeled `NEW` is additive. Nothing in the existing RPC surface
(`Describe`/`Match`/`Fetch`/`Health`), handshake, or `topos.v1` proto
package changes — this milestone does not need a `topos.v3` contract
generation. That is itself a load-bearing finding: it constrains *how*
external loading and the filesystem/GDrive plugins can be built (they must
work within the current `ProtocolVersion: 2` / `contract_version:
"topos.v2"` contract, not extend it).

## (a) Where trust marking lives — and why self-declared trust is a non-starter

**The kernel is the only party that can assert trust, and it can only
assert it from something structural, not from anything a plugin process
says about itself.** This mirrors the existing discipline verbatim: `desc
:= impl.Describe(...)` already documents "a plugin's identity is never
trusted from its filename or its config key, only from what `Describe`
reports" — but `Describe` itself is *plugin-authored output*. A malicious
or just poorly-written external plugin can set `source_type: "paperless"`
and `display_name: "paperless-ngx"` in its own `Describe` response with
zero code changes to the kernel needed to make that lie succeed. Trust
cannot be a `Describe` field for the same reason identity isn't allowed to
be a caller-supplied field anywhere else in this contract.

**Recommended mechanism: directory-tier provenance, computed at discovery
time, never persisted as editable state.**

- `[plugins] dir` (existing key, default `"plugins"`) keeps its current
  meaning: the directory `make plugins`/`make build`/the release pipeline
  populates with binaries built from this repository's own source tree.
  Everything discovered here is **trusted**.
- Add `[plugins] external_dir` (new key, e.g. default
  `"~/.local/share/topos/external-plugins"`, empty/absent = feature
  disabled — an operator who never sets this key sees no behavior change
  at all). Everything discovered here is **untrusted**, unconditionally,
  regardless of binary name — an external binary literally named
  `topos-plugin-paperless` is still untrusted, because the trust signal is
  "which directory did the kernel find this in," never the name.
- `kernel/pluginhost/discover_binaries.go`'s `DiscoverAllBinaries` becomes
  two calls (one per directory) whose results are tagged with an origin
  enum (`OriginBundled` / `OriginExternal`) rather than merged into one
  flat list. `DiscoverBinaries` (the UI-policy view already documented as
  distinct from `DiscoverAllBinaries`'s security-relevant view) inherits
  the same split.
- Trust is **not** written into `config.toml`. If it were a `config.Source`
  field (e.g. `trusted = true`), a hand-edit of the file — which the
  project's own config philosophy explicitly supports as a first-class
  path, not a UI-only one — could silently flip an untrusted plugin to
  trusted with no warning ever shown again. Keeping trust as a live,
  recomputed-at-launch fact (which directory does `src.Plugin` currently
  resolve from) means the warning is honest every single time a plugin
  from `external_dir` launches, including after a config hand-edit,
  a `Reconcile` hot-apply, and a kernel restart — there is no persisted
  "I already accepted this" flag to go stale or be copied between
  machines.
- `pluginhost.Plugin` gains an `Origin` field (parallel to the existing
  `sourceType`/`pluginName`/`iconBytes` shape) set once at `launch()` from
  which directory `binPath` resolved under, surfaced by `Host.Plugins()`/
  `ProbeSources()` exactly like `Icon()` is today.

**Why not a stronger mechanism (checksums, code-signing, a published
trust list)?** PROJECT.md's milestone scope is explicit: *"load + trust
marking only; distribution, dev guide, and certification deferred."*
Directory-tier provenance is the correct **v1.1.0** answer because it is
the only mechanism that requires zero new infrastructure (no signing key
to manage, no checksum manifest to publish and keep in sync with release
artifacts, no trust-list file to fetch) while still being **kernel-derived,
not plugin-asserted** — it satisfies "kernel must determine provenance"
honestly, at the cost of not being cryptographically strong (an operator
who manually copies a binary from `external_dir` into `dir` defeats it —
this is explicitly accepted, not hidden: see (c) below, and matches the
already-published sentence in `docs/plugin-contract.md`, *"only run plugin
binaries you built yourself or whose source you trust."*). Certification/
signing is the natural v1.2+ layer once a real third-party plugin author
shows up (same "reconsider when adoption pressure exists" pattern this
project already used for `go-plugin` vs. plain HTTP/JSON).

**Rejected alternative: a hardcoded plugin-name allowlist** ("trusted
names are paperless/silverbullet/proton/signal/whatsapp/filesystem").
Directly contradicts D-05's already-established discipline — "the kernel
holds no built-in table of known plugin types" — extended from match
vocabulary to plugin *type* discovery in `discover_binaries.go`'s own doc
comment. Don't reintroduce that table for trust.

## (b) Source picker / config UI surfacing

The two-section picker (`AddSourceModal.svelte`, Phase 9) already
separates *configured instances* from an *install-a-plugin catalog* driven
by `GET /api/config/plugin-types`. That response needs one additive field
per entry:

```json
{ "schema_version": 2, "plugin_types": [
  { "binary": "topos-plugin-paperless", "trusted": true },
  { "binary": "topos-plugin-gdrive", "trusted": false }
]}
```

(bumping `plugin-types`' own `schema_version`, not the plugin
contract's — this is a kernel HTTP API change, orthogonal to the
`topos.v1`/`topos.v2` proto/handshake versioning discussed above).

UI touchpoints, extending existing components rather than adding new ones:

- **Catalog list (`AddSourceModal.svelte`'s "New `<plugin type>`…" rows):**
  an untrusted entry gets a visible badge (e.g. an amber "Untrusted"
  pill next to the plugin's icon — reuse `PluginIcon.svelte`, which
  already falls back gracefully when no icon is available, since an
  external plugin's self-declared icon is exactly as untrustworthy as its
  self-declared identity — same MIME-allowlist/64 KiB-cap capture path
  already enforced in `pluginhost.captureIcon` covers it either way, no
  new work needed there).
- **Add-flow confirmation:** before `POST /api/config/describe-plugin`'s
  result is accepted into the two-step modal's step 2, an untrusted trial
  launch shows an interstitial warning — reuse the existing `Alert`/
  `AlertDescription` component already imported into `AddSourceModal.svelte`
  — with text drawn directly from `docs/plugin-contract.md`'s own
  established language: *"This plugin was not built from the
  `davison/topos` repository. A plugin binary has the same access to this
  machine as topos itself — only continue if you trust its source."* This
  is a confirm-to-proceed step, not a hard block (the milestone's "load +
  trust marking" scope implies untrusted plugins are loadable, just
  clearly labeled — a hard block would make the GDrive dogfooding goal
  itself impossible without a separate unblock mechanism).
- **`ManageSourcesModal.svelte` / per-instance chip:** an already-configured
  instance launched from `external_dir` keeps showing the same badge
  persistently (not just at add-time) — consistent with "recomputed at
  launch, never a one-time-accepted flag" from (a). `SourceChip.svelte`'s
  existing per-source tooltip (health/diagnostic) gains one more line for
  this.
- **`GET /api/sources`** (existing per-instance health endpoint) gains the
  same `trusted` boolean per row, sourced from `pluginhost.SourceHealth`
  (parallel to how `Plugin` field already threads through for icon
  lookups) — this is what powers the persistent chip badge above.

## (c) Read-only / egress AST guarantees do not, and cannot, extend to external plugins — replace with honest labeling + the structural RPC boundary

Verified directly: `internal/audit`'s two guard tests
(`TestNoForeignEgressOutsideSanctionedClient`,
`TestNoModuleDeclaresAKnownVulnerablePin`) both `filepath.WalkDir(repoRoot,
...)` where `repoRoot = "../.."` — **a filesystem walk of this repository's
own source tree.** An external plugin's source code, by construction, does
not live in this tree (it may not even be Go, per the contract's own "or,
if the source plugin's language support ever expands, a separate binary
speaking the same gRPC contract" clause) and the kernel never has access
to it at build time or runtime — only the compiled binary. There is no
version of these AST scans that can run against a binary you don't have
source for. This is not a gap to close this milestone; it is a structural
boundary to document honestly, in three layers:

1. **What still holds, unconditionally, for every plugin regardless of
   trust:** the `SourcePlugin` gRPC service exposes exactly four RPCs —
   `Describe`, `Match`, `Fetch`, `Health` — none of which can write. This
   is enforced by `sdk/contract_test.go`'s RPC-allowlist test, which *is*
   still a real guarantee for an external plugin, because it constrains
   what the **kernel can ask any plugin to do**, not what the plugin's own
   process does independently. An external plugin cannot be sent a write
   RPC by this kernel, ever, no matter how untrusted — that boundary is
   contract-shaped, not trust-shaped, and doesn't weaken here.
2. **What stops holding the moment a plugin is external:** whether the
   plugin's *own implementation* only issues read (`GET`) HTTP calls to
   its source system, whether it logs a credential, whether it talks to
   any host beyond its declared source — all of that is exactly what
   `outbound_hosts_test.go`'s AST scan currently proves for the five
   in-repo plugins, and none of it is provable for a binary from
   `external_dir`. `docs/plugin-contract.md` already states this
   precisely for the general "no containment" case ("`hashicorp/go-plugin`
   is a transport, not a sandbox... Installing a third-party plugin is
   therefore the same trust decision as installing the kernel binary
   itself") — this milestone's job is to make that existing sentence
   *actionable* in the UI (the badge/warning in (b)) rather than only
   readable in a markdown file a user may never open.
3. **What replaces the AST guarantee for an external plugin, concretely:**
   nothing mechanical does. The mitigation stack is (i) the RPC-boundary
   guarantee in point 1, which is real and unconditional; (ii) honest,
   persistent UI labeling from (b), so the trust decision is visible every
   time, not just at add-time; (iii) documentation — extend
   `docs/plugin-contract.md`'s existing "no containment" section with an
   explicit pointer to this milestone's `external_dir` mechanism once it
   ships, so the contract document and the shipped behavior describe the
   same thing; (iv) explicitly **not** attempting process sandboxing
   (seccomp/namespaces/restricted env) this milestone — that would be a
   materially larger scope change (the same WASM-vs-subprocess trade-off
   this project already rejected for the in-repo case in PROJECT.md's
   "What NOT to Use" table, now revisited for a case where sandboxing
   would actually earn its keep, but is still out of this milestone's
   explicitly deferred scope alongside "distribution, dev guide, and
   certification").

## (d) Filesystem plugin — fits the existing local-path pattern exactly

`plugins/signal` is already this contract's reference implementation of
"a local-path source, no network endpoint at all" — `docs/plugin-
contract.md` names it explicitly as the shape to follow. The filesystem
plugin is a structurally simpler instance of the *same* shape (no cgo, no
SQLCipher, no keyring unwrap step):

- **Config:** `{"path": "/home/user/Documents/project-x"}` via the
  existing `Source.Path` field and `WEBSPACES_SOURCE_CONFIG`'s existing
  `"path"` key — no `config.Source` struct change needed. "Optionally
  subfolders" (from the milestone scope) is a plugin-side recursion flag —
  either a second config key (would need one new `Source` field, e.g.
  `Recursive bool` — small, additive, `omitempty`-safe) or, more in
  keeping with "the exact key set is source-specific" language already in
  the contract, folded into a structured `path` value the plugin parses
  itself. Recommend the dedicated boolean field: it is visible in
  `config.example.toml` and the config UI's connection form without the
  plugin inventing its own micro-syntax, and it's one line in
  `kernel/pluginhost/host.go`'s existing fixed-key JSON marshal (see (e)
  for why this fixed-key marshal itself needs to change anyway for GDrive).
- **`Describe`:** `source_type: "filesystem"`, a `match_vocabulary` of
  something like `["folders"]` (top-level subfolder names as the native
  categorization — directly parallel to Proton's `["folders"]`) or
  `["tags"]` if frontmatter-less plain files instead key on a
  naming convention; this is a plugin-design decision for the phase that
  builds it, not an architecture constraint.
- **`Match` / sync-time indexing:** walks the configured path (respecting
  the recursion flag), and for each file builds an `Item` with `title` =
  filename, `labels` = the containing subfolder path segments,
  `timestamp_unix` = file mtime, `preview` = a bounded text extract **only
  for formats cheap to extract from without a heavyweight dependency**
  (plain text/markdown read directly; anything else — PDF, docx — either
  ships with no preview text in v1.1 or the phase adds a text-extraction
  library, which is a scope decision, not an architecture one).
  `deep_link`: a `file://` URI is the only "source system" a local file
  has — declare `LINK_FIDELITY_EXACT` (it opens exactly this object, via
  the OS's own file-URI handler) even though, unlike every other current
  source, the "source system" here is the OS itself, not a remote app.
- **`Fetch` / "what fetch-live means for a local file":** identical
  *contractually* to every other plugin — `Fetch` is called only at
  request time, re-reads the file from disk fresh on every call (no
  caching layer, matching the hybrid model's "content fetched live from
  source on open" exactly), and returns `text` for extractable formats or
  `data`+`mime_type` for opaque binary formats (mirroring
  `plugins/paperless`'s PDF/image `Fetch` path: binary renditions bypass
  the kernel's HTML sanitize/wrap pipeline entirely — `content_shape` stays
  `CONTENT_SHAPE_UNSPECIFIED`, correctly, since `mime_type` won't be
  `"text/html"`). The only genuinely new wrinkle versus every existing
  plugin: a *local* file can disappear or change between `Match`-time
  indexing and `Fetch`-time open (a remote API call for a deleted
  paperless document already returns 404 the same way) — the plugin
  should map a missing file to `codes.NotFound` exactly like every other
  plugin does for a deleted item, no new kernel-side handling required.
- **Build wiring:** in-repo, trusted, joins the existing `plugins-
  portable`/`plugins`/`build`/`build-portable` Makefile targets exactly
  like `paperless`/`silverbullet`/`proton`/`whatsapp` (no cgo needed, so it
  does **not** need `signal`'s special isolated target) — `mkdir -p
  bin/plugins && go build -o bin/plugins/topos-plugin-filesystem
  ./plugins/filesystem`, added to `go.work`'s `use` block.

## (e) Google Drive plugin — out-of-repo module, dogfooding the external path end to end

**Repo layout:** a **separate git repository** (not a subdirectory of
`davison/topos`) — this is the entire point of "dogfoods the
external-plugin mechanism end to end." Its own `go.mod`:

```go
module github.com/<author>/topos-plugin-gdrive

go 1.25

require github.com/davison/topos/sdk v0.x.y
```

`github.com/davison/topos/sdk` already resolves as a standalone Go module
today — `sdk/go.mod`'s own module path is `github.com/davison/topos/sdk`,
and it's a subdirectory-rooted module inside the public `davison/topos`
repo, which Go's module resolution already supports without any change on
the topos side (`go get github.com/davison/topos/sdk@<tag-or-commit>`
works exactly as documented in `docs/plugin-contract.md`'s "Build your
first plugin" walkthrough — that walkthrough's step 1 already says "or, if
building outside this repo entirely — nothing about the contract requires
your plugin to live in this repo"). **No new publishing infrastructure is
required for this milestone** — the SDK is already externally consumable;
the GDrive plugin is the first thing that actually proves it by consuming
it that way. Recommend tagging `sdk/vX.Y.Z` releases going forward (Go
module proxy resolution wants real semver tags on the module's own path
prefix) as a small, cheap addition alongside this phase, not a
prerequisite blocking it (an untagged `@latest`/pseudo-version works fine
for dogfooding).

**Contract dependency:** the GDrive plugin implements `sdk.SourcePlugin`'s
four methods exactly like `plugins/mock` does, imports
`toposv1 "github.com/davison/topos/sdk/gen/topos/v1"` for the generated
proto types, and registers under handshake `sdk.Handshake` — **it consumes
the same generated Go stubs the SDK module vendors, not the raw
`.proto` file** (the SDK module is the documented "stable Go-native
surface"; the `.proto` itself is the source of truth but an external Go
plugin author, per the contract's own framing, works through the SDK, not
`buf generate` themselves, unless they need a non-Go implementation).

**OAuth token storage — this is a genuine new architectural surface, not
just "give GDrive a token field":**

- Google Drive OAuth2 needs more than the existing `Source.Token` bearer-
  token field models: a client ID/secret (or a pre-authorized refresh
  token), and critically, a **refresh token that the plugin itself must
  persist and rotate** — unlike every current plugin (`paperless`/
  `silverbullet`/`proton` all use a static, operator-supplied credential
  that never changes at runtime), GDrive's live-usable credential
  (the access token) expires hourly and is refreshed by the plugin process
  itself using the refresh token. **This is new: it is the first plugin
  whose runtime needs to persist its own state beyond what
  `WEBSPACES_SOURCE_CONFIG` hands it at launch** — but there's already a
  precedent for exactly this shape: `plugins/whatsapp` already owns a
  persistent, plugin-side store for its session (explicitly "session +
  captured messages, both plugin-owned; source stores never touched").
  GDrive's refresh-token persistence should follow that same precedent —
  a plugin-owned, plugin-managed file (e.g.
  `~/.local/share/topos/gdrive/<instance>/token.json`), **not** a new
  kernel-side secret store, and **not** re-submitting the rotated token
  back to the kernel/config.toml on every refresh (config.toml's whole
  design principle — "secrets stay environment-only as `${VAR}`
  references" — assumes secrets are static and operator-managed; a token
  that rotates hourly cannot live there without breaking that
  principle). The one-time OAuth *authorization* flow (get an initial
  refresh token) should follow WhatsApp's own in-app pairing precedent
  loosely — but WhatsApp's flow is entirely plugin-owned via a
  kernel-mediated link-session endpoint; GDrive's is a standard OAuth2
  redirect/consent flow, which is a materially different UX shape (a
  browser redirect to accounts.google.com, not a QR code) and is worth
  scoping as its own design question in the phase that builds this, not
  assumed to be "the same as WhatsApp."
- **This surfaces a required kernel-side change that predates GDrive and
  actually blocks it:** `kernel/pluginhost/host.go`'s `launch()` marshals
  `WEBSPACES_SOURCE_CONFIG` from a **fixed, hardcoded set of `config.Source`
  fields** (`base_url`, `token`, `api_version`, `ca_cert`, `username`,
  `webmail_base_url`, `path`). An out-of-repo plugin the kernel has no
  compile-time knowledge of (GDrive needing `client_id`/`client_secret`/
  `folder_id`, or any future third-party plugin needing its own arbitrary
  keys) has **no way to receive connection details the kernel doesn't
  already know to marshal** — this directly contradicts the contract's own
  claim that "a plugin defines and documents whatever keys it needs."
  That claim is currently only true for the five keys already hardcoded.
  **Fix required, and it belongs in the external-plugin-loading phase, not
  the GDrive phase**: add a generic `Extra map[string]string` (or
  similarly named) field to `config.Source`, merged into the
  `WEBSPACES_SOURCE_CONFIG` JSON alongside the fixed keys, round-tripped
  through TOML the same way `Match`/`Agent` blocks already are. Every
  in-repo plugin keeps using the fixed named keys unchanged (no migration
  needed); only a plugin needing keys the kernel doesn't model by name —
  GDrive's `client_id`/`client_secret`/`folder_id` — reads them out of the
  generic map instead. This is the actual load-bearing prerequisite for
  "prove the external path... built out-of-repo against the published
  contract" being true in more than name — without it, "the published
  contract" silently means "the six fields topos happens to hardcode
  today," not what `docs/plugin-contract.md` documents.
- **Build wiring:** out-of-repo means no `go.work` entry, no Makefile
  target in `davison/topos` at all — the GDrive plugin repo has its own
  `go build -o topos-plugin-gdrive .`, and the *operator* copies the
  resulting binary into `external_dir` from (a). This is exactly the path
  the trust-marking mechanism needs to exist for before this plugin can be
  meaningfully dogfooded — see Build Order, below.

## (f) Per-item include/exclude — the kernel's first user-owned data, and the first schema exception to D-07

**Schema:** a new table, additive to `kernel/index/schema.go`:

```sql
CREATE TABLE IF NOT EXISTS item_marks (
  webspace_name TEXT NOT NULL,
  item_id       TEXT NOT NULL,   -- global "{source}:{source_id}" id (docs/api.md's stable-ID scheme)
  state         TEXT NOT NULL,   -- 'include' | 'exclude'
  marked_unix   INTEGER NOT NULL,
  PRIMARY KEY (webspace_name, item_id)
);
```

Two deliberate departures from `webspace_items`'s existing shape, both
necessary because this is user data, not synced/derived data:

1. **No `REFERENCES items(id) ON DELETE CASCADE`, and no foreign-key
   constraint against `items` at all.** `webspace_items` can safely
   cascade-delete because it is 100% re-derivable from the next sync
   (D-07's own stated rationale for every existing table). A mark is the
   opposite — it must **outlive** the specific item row's own churn (a
   resync that upserts the same `item_id` again, or a schema-version
   rebuild that drops and recreates `items` entirely) and silently
   re-attach the moment a matching `item_id` reappears. A hard FK would
   force marks to be deleted and lost exactly when they're most likely to
   matter (across a rebuild).
2. **Excluded from `rebuildOnSchemaChange`'s `DROP TABLE IF EXISTS` list**
   (currently: `items_fts`, `webspace_items`, `webspaces`, `sync_runs`,
   `items`). This is the one line that actually makes the "user-owned
   data beyond config" property real — everything else in that function
   is intentionally destroyed and rebuilt from a fresh sync on a schema
   bump (`schema.go`'s own doc comment: "every row here is re-derivable
   from a fresh sync, so there is nothing worth migrating"). `item_marks`
   is the first table for which that sentence is **false**, and the code
   needs a comment saying so explicitly, at the exact point future
   maintainers will be tempted to "clean up" the drop list. If `item_marks`
   itself ever needs a genuinely breaking shape change later, it needs its
   own additive-only discipline (`ALTER TABLE ... ADD COLUMN`, never a
   blanket drop) — don't reuse the single global `PRAGMA user_version`
   counter for both concerns, since it currently means "drop and
   recreate," which is exactly what must never happen to this table.

**Interaction with instance renaming (D-08):** no special-case code
needed. Renaming a `[sources.<id>]` key already creates a brand-new
instance identity whose items carry a different id prefix — old marks,
keyed to the old prefix, simply stop matching anything and go inert,
identical to how old sync history and old index rows already behave on a
rename. Consistent with existing discipline, not a new edge case.

**Filter application order — this is the one place the milestone
description's "final tier of the filter hierarchy" needs to be made
concrete, and it has a real SQL-shape decision buried in it:**

The existing hierarchy is: per-instance typed `match` blocks → webspace
`keywords` fallback (both resolved once, at *sync* time, into
`webspace_items` rows) → the promoted-search `Filter` FTS-term stack
(applied at *read* time, narrowing `StreamItems`/`Search`/the agent
stream identically, per D-16's "the filtered view IS the webspace for
every consumer" rule). Per-item marks are also a *read-time* concern (a
mark doesn't change what `Match` returns or what's in the index — it
changes what's **shown**), so they sit alongside the `Filter` stack in
that same read-time layer, and must be applied in this precedence, outer
to inner:

1. **Per-source agent grants (`AGENT-01`) still dominate everything**,
   unconditionally — an `exclude`/`include` mark on an item from a source
   with `agent.read = false` must never let that item appear in an agent
   stream. Marks operate *within* the set of sources already visible to
   the calling context, never as a bypass of that boundary. (This needs
   an explicit test: "include-marked item from an unread-granted source
   is still absent from `/agent/v1` streams.")
2. **Sync-time membership** (`webspace_items`, from match/keywords) is the
   base set.
3. **`exclude`** removes an item from that base set.
4. **`include`** adds an item **not otherwise in the base set** back in —
   and this is the concrete SQL wrinkle: `StreamItems`'s current query
   (verified directly in `kernel/index/store.go`) is anchored
   `FROM items JOIN webspace_items ON webspace_items.item_id = items.id
   WHERE webspace_items.webspace_name = ?` — an item absent from
   `webspace_items` is structurally invisible to this query no matter what
   else is added. Supporting `include` for an item **outside** the synced
   match set requires re-anchoring the query on `items` with a `LEFT JOIN
   webspace_items` (or a `UNION`), which is a real, non-trivial rewrite of
   the primary stream/search query shape — not a one-line `WHERE` addition
   — and should be scoped and estimated as such in the phase plan, not
   assumed to be "just add a join."
5. **The promoted-search `Filter` stack** narrows on top of the result of
   3–4, exactly as it narrows the current base set today.

**Open question this research surfaces rather than resolves (needs a
planning-time decision, not an architecture-time one): where does a user
*find* an item to mark `include` if it's outside the webspace's current
match set?** Every existing read surface (`GET .../stream`,
`GET .../search`) is already scoped to `webspace_items` — there is no
existing "browse everything this source synced, matched or not" view. Two
honest resolutions, either of which is a reasonable phase-planning
decision but the roadmap should pick one deliberately:

- **(i) Narrow scope:** `include` is really "un-exclude" — a toggle that
  only ever restores an item this same webspace previously excluded
  (state transitions `unmarked → exclude → unmarked`, `include` never
  needed as a distinct third state for an item that never matched at all).
  This needs no new browse surface and is the cheaper, safer v1.1 scope.
- **(ii) Full scope:** `include` genuinely means "pull in an item this
  webspace's match config would never have selected," which requires a
  new browse surface (e.g., "search this source's full synced index,
  unscoped by webspace membership") before it's reachable at all — a
  materially bigger feature than the milestone's one-line description
  suggests.

Given the milestone phrase "mark individual **stream entries**" (not "mark
any item"), (i) reads as the more literal, lower-risk interpretation — flag
this explicitly for `/gsd-new-milestone`'s requirements/roadmap step rather
than let a phase plan silently assume the bigger scope.

**API surface (additive, mirrors existing config-mutation patterns):**

```
PUT /api/webspaces/{webspace}/items/{id}/mark   { "state": "exclude" }
DELETE /api/webspaces/{webspace}/items/{id}/mark   (clears back to unmarked)
```

Existing `GET .../stream` / `GET .../search` response items should carry
their current mark state (`"mark": "exclude" | null`) so the SPA can render
the toggle affordance without a second round-trip per row — this is the
same "the response already carries what the UI needs to render" discipline
already used for icons/health/fidelity elsewhere in the API.

**UI touchpoints:** a per-row action (menu entry or hover affordance on
the stream row component under `web/src/routes/w/[webspace]/`) — "Exclude
from this webspace" / "Include" toggle, optimistic-update against the
already-loaded stream list, matching the existing pattern used for the
per-chip refresh/menu actions in `SourceChip.svelte`.

## (g) PWA installability

This is the most self-contained of the five features — it touches
`web/static/`, `web/svelte.config.js`, and `kernel/webui/embed.go`'s
existing `go:embed all:build` boundary, and nothing else in the kernel.

- **Manifest:** `web/static/manifest.webmanifest` (or `manifest.json`) —
  a static file `adapter-static` already copies into the build output
  unchanged (same mechanism that already ships `web/static/robots.txt` and
  `web/static/app-icon.png` today). Needs `name`, `short_name`, `start_url`
  (`/`, or a specific webspace if one is remembered — v1.1 scope should
  probably keep this simple: `/`), `display: "standalone"`, `icons` (an
  array of sized PNGs — `web/static/app-icon.png` already exists at
  1024×1024, source material for generating the standard PWA size set:
  typically 192×192 and 512×512 minimum, plus a maskable variant is
  recommended by current install-prompt heuristics), and `theme_color`/
  `background_color` matching the SPA's existing theme tokens.
- **ServiceWorker:** `web/src/service-worker.js` (SvelteKit's own
  convention — `@sveltejs/kit` auto-detects this file and bundles it,
  which works cleanly with `adapter-static`'s SPA-fallback mode already
  configured) or a hand-rolled `web/static/service-worker.js` registered
  manually from `+layout.svelte` if the SvelteKit-native path conflicts
  with the existing `fallback: '200.html'` SPA-routing setup (verify
  during implementation — this is the one integration risk in this
  feature, not architectural, just needs a spike/verification step in the
  phase that builds it).
- **Cache strategy vs. the live API — this is the actual design decision,
  not the manifest/SW plumbing:** topos's core value is "instant metadata
  from index, live preview fill via plugin `Fetch`" — i.e., the app is
  explicitly *not* meant to serve stale content confidently. The
  ServiceWorker should:
  - **Cache-first, long-TTL** for the SPA's own static assets (JS/CSS/
    fonts/icons) — this is the standard, safe PWA shell-caching pattern
    and matches how `GET /api/plugins/{plugin}/icon` already declares
    itself (`Cache-Control: public, max-age=31536000, immutable`) — the
    same "static, versioned by build" reasoning applies to the SPA bundle
    itself.
  - **Network-only (never cache) for every `/api/*` route.** Item content,
    stream data, search results, and config are all either live-fetched
    from a source plugin or reflect the current index/config state — this
    project's entire hybrid-data-model argument ("fast browsing... without
    staleness of content") is undermined if a ServiceWorker starts
    silently serving a stale `GET /api/webspaces/{ws}/stream` response
    while offline and calling it a feature. Recommend explicitly **not**
    attempting an "offline stream" experience this milestone — installable
    (a home-screen icon, a standalone window, works when the kernel is
    reachable) is the stated scope; "usable with the kernel unreachable"
    is a different, larger feature this milestone doesn't ask for. A
    ServiceWorker that intercepts `/api/*` only to immediately
    network-fetch (no cache fallback) keeps the installability win without
    quietly promising offline reads the architecture (loopback-only, no
    auth, single-user, always-needs-the-kernel-process) was never built to
    honor.
- **`kernel/webui/embed.go` needs no changes at all** — `manifest.webmanifest`
  and `service-worker.js` land in `web/static/`/`web/src/`, `adapter-static`
  copies/bundles them into `kernel/webui/build/` exactly like every other
  static asset already does, and `//go:embed all:build` already picks up
  anything present there. This feature is entirely a `web/` change from
  the kernel's point of view.
- **One kernel-side check worth adding, not structural but easy to miss:**
  a ServiceWorker requires being served over a secure context — `https:`
  or `localhost`/`127.0.0.1`. The kernel's existing default
  (`listen = "127.0.0.1:7777"`) already satisfies this for the common
  case; the existing startup warning for a non-loopback `listen` value
  ("this exposes the API beyond this machine") is the natural place to
  also note "and installability requires HTTPS at that point," since a
  LAN-bound deployment without TLS would silently lose the PWA install
  prompt with no error anywhere — worth a one-line addition to that
  existing warning, not a new mechanism.

## Suggested Build Order

Ordered by hard dependency, not by the milestone's own listed order
(which groups external-plugin loading first, correctly, but the reasoning
matters for sequencing the rest):

1. **External plugin loading + trust marking (a, b, c).** Nothing else in
   this milestone can be *proven* without this landing first: the GDrive
   plugin (e) has nowhere honest to be placed until `external_dir` and its
   trust badge exist, and the milestone's own stated purpose for GDrive
   ("dogfoods the external-plugin mechanism end to end") is meaningless
   before this phase ships. This phase should also land the
   `config.Source.Extra` generic-config-map fix identified in (e) — it's
   logically part of "what does an external plugin need to actually be
   configurable," not GDrive-specific, and landing it here means the
   GDrive phase consumes a stable, already-tested mechanism rather than
   co-developing it under time pressure in an out-of-repo module where
   kernel-side test coverage doesn't reach.
2. **Filesystem plugin (d).** No dependency on (1) at all — it's an
   in-repo, trusted plugin using the existing local-path pattern
   unchanged. Sequencing it second (not first) is a *value* decision, not
   a dependency one: it's lower-risk, proves nothing new architecturally
   (Signal already proved the local-path shape), and could equally run in
   parallel with (1) if the team has capacity — call this out explicitly
   as parallelizable in the roadmap rather than strictly serial.
3. **Google Drive plugin, out-of-repo (e).** Hard-depends on (1)
   (external loading must exist to load it at all; the `Extra` config map
   must exist for it to receive OAuth credentials). Also benefits from (2)
   existing first only insofar as (2) will have exercised the
   local-path/no-network-endpoint-shaped parts of the contract that GDrive
   partially shares (folder-of-documents framing) — not a hard dependency,
   but useful prior art for whoever builds GDrive to reference.
4. **Per-item include/exclude (f).** Fully independent of (1)/(2)/(3) —
   it's a stream/index/UI feature with zero plugin-contract surface. Can
   run in parallel with any of the above. The one internal dependency is
   sequencing *within* this feature: the `item_marks` schema-exemption
   change (excluding it from `rebuildOnSchemaChange`'s drop list) should
   land and be tested in isolation before the query-anchor rewrite in
   `StreamItems`/`Search` (the `include`-support re-anchor identified
   above), since the rewrite is the riskier, more invasive change and
   benefits from the schema/API/basic-exclude path already being solid
   underneath it. Recommend splitting this into two phases or two
   sub-plans: (4a) schema + `exclude` only [lower risk, ships value
   alone] → (4b) `include` + the query re-anchor [higher risk, only
   pursued once the open scope question above is resolved].
5. **PWA installability (g).** Fully independent of everything else in
   this milestone — pure `web/` + static-asset work with no kernel-side
   coupling beyond the one-line HTTPS-warning addition. Can run anytime,
   including first, if the team wants an early low-risk win, or last as a
   wrap-up phase. No reason to gate it on anything above.

**Net dependency shape:** `(1) → (3)`; everything else — `(2)`, `(4)`,
`(5)` — is independent of `(1)` and of each other, and independent of
`(3)`. Only the external-plugin-loading → GDrive edge is a genuine hard
sequencing constraint; the roadmap should feel free to interleave or
parallelize the remaining four phases based on team capacity rather than
treating the milestone's five bullet points as a strict list.

## Sources

- `/home/darren/projects/davison/topos/.planning/PROJECT.md` — HIGH (project's own source of truth)
- `/home/darren/projects/davison/topos/docs/plugin-contract.md` — HIGH (published contract, read in full)
- `/home/darren/projects/davison/topos/docs/api.md` (§§ plugin-types, describe-plugin, plugin icon, config) — HIGH (read directly)
- `/home/darren/projects/davison/topos/kernel/pluginhost/{host.go,discover_binaries.go}` — HIGH (read directly, current discovery/launch/trust-adjacent code)
- `/home/darren/projects/davison/topos/kernel/index/{schema.go,store.go}` — HIGH (read directly, current rebuild-on-schema-change and StreamItems query shape)
- `/home/darren/projects/davison/topos/kernel/config/types.go` — HIGH (read directly, `Source` struct's current fixed-field shape)
- `/home/darren/projects/davison/topos/kernel/httpapi/rendition.go`, `plugins/paperless/plugin.go` — HIGH (read directly, binary-vs-text/html rendition handling precedent for the filesystem plugin)
- `/home/darren/projects/davison/topos/internal/audit/{outbound_hosts_test.go,module_pins_test.go}` — HIGH (read directly, confirms AST scan scope is `repoRoot` filesystem walk only)
- `/home/darren/projects/davison/topos/web/src/lib/components/AddSourceModal.svelte`, `web/svelte.config.js`, `kernel/webui/embed.go`, `go.work`, `sdk/go.mod` — HIGH (read directly, current SPA/build/module-boundary shape)

---
*Architecture research for: topos v1.1.0 "Plugin Ecosystem" milestone integration*
*Researched: 2026-08-12*
