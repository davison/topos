# Phase 12: Filesystem Source - Research

**Researched:** 2026-08-13
**Domain:** Local/network filesystem crawling as a topos source plugin; kernel-mediated native-app deep links; trusted/external plugin parity rehearsal
**Confidence:** HIGH (architecture, contract mechanics, in-repo precedent) / MEDIUM (external glob library choice, NFS/SMB polling specifics) / LOW (nothing load-bearing — one open design fork flagged explicitly below)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Stable item identity**
- **D-01:** The per-source stable ID is the **path relative to the source folder**. Deterministic on every mount type (NFS/SMB fileid/inode stability is not guaranteed, and copy/restore/rsync churns inodes anyway). — Reversibility: costly — index rows, Phase 13 exclusions, and dedup all key on this ID; changing the scheme later forces an index rebuild for existing filesystem sources.
- **D-02:** A rename or move is honestly a **remove + add** (old item disappears, new item appears). No content-hash rename detection — the existing sync-integrity machinery already handles removals cleanly, and hash-assisted re-identification would cost a full read of every changed file plus a corner-case matrix.

**Document scope**
- **D-03:** Default scope is a **document-ish allowlist** (PDF, office formats, markdown, plain text, images). Per-instance **extras keys widen or narrow it** (e.g. include/exclude globs) — the escape hatch for unusual folders, and a real exercise of Phase 11's extras machinery (D-12/D-13/D-15 from 11-CONTEXT.md) on an in-repo plugin.

**Previews**
- **D-04:** Previews **reuse the existing Fetch bytes+MIME pipeline** — no new kernel or UI rendering work. PDFs and images return raw bytes with their MIME type and render inline via the existing media previewer (paperless precedent); markdown returns rendered content through the kernel rendition boundary (SilverBullet shape precedent); plain text returns as text; office formats (docx/xlsx/…) get metadata + deep link only — browsers can't render them natively, and server-side conversion is explicitly out of scope this phase.

**Webspace matching**
- **D-05:** The plugin's declared match vocabulary is **folder paths** — the filesystem's native categorization, mirroring how email folders/labels work. A webspace's match block lists subfolder paths/names; the keywords fallback matches folder names the same way. One instance can serve many webspaces. No filename-token match field.

**Deep links**
- **D-06:** Clicking a filesystem item triggers a **kernel-mediated open**: a small, loopback-only kernel endpoint runs `xdg-open` on the item's real path, so the document opens in the desktop's own handler — the full criterion-3 experience, declared at navigating fidelity. The endpoint MUST be constrained to paths of currently-indexed items (never an arbitrary caller-supplied path) and treated with the same care as the kernel's other exec surface (the WhatsApp link spawner precedent shows the shape, but this is new machinery, not reuse).

### Claude's Discretion
- Change-detection signal within stat-diff polling (mtime+size is the obvious default), sync cadence, and large-tree scan strategy.
- Exact allowlist contents (which extensions per category) and the extras key names/glob syntax for scope overrides.
- Hidden-file/dot-directory and in-tree symlink handling defaults.
- Read-only guard specifics (same committed-guard pattern as every other plugin).
- External-rehearsal test setup (Phase 11 D-11 already settled tier-collision semantics; the fixture/harness approach is the planner's call, with `testdata/external-plugin` and the Phase 11 e2e fixtures as precedent).
- Whether "open containing folder" is offered as a secondary affordance alongside the primary open action.

### Deferred Ideas (OUT OF SCOPE)
- Server-side office-document conversion (LibreOffice headless or similar) for inline docx/xlsx previews — new heavyweight machinery; revisit if metadata + deep link proves insufficient.
- "Signal schema-version verify-and-accept tooling" — reviewed, unrelated to this phase, stays pending for a Signal-plugin phase.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SRC-04 | User can add a local or network filesystem folder (optional subfolders) as a source; documents appear in webspace streams with previews and deep links, synced via stat-diff polling that works on network mounts | Full architecture below: plugin shape (Match/Fetch/Health), stat-diff scan strategy, preview pipeline reuse, kernel-mediated deep-link endpoint, external-tier rehearsal fixture pattern |
</phase_requirements>

## Summary

The filesystem source is the sixth source plugin and the first that reads
directly from an arbitrary local/network path with no source-system API
at all — closer in shape to `plugins/signal` (a local-path source with no
`base_url`/`token`) than to `plugins/paperless`/`plugins/silverbullet`
(REST clients). Its entire job is: on every `Match` call, walk the
configured folder (respecting a recursion setting and an extras-driven
include/exclude glob scope), stat each candidate file, and return one
`Item` per matching file with `source_id` set to the path relative to the
folder root (`D-01`). Because `kernel/correlate.Engine.SyncSource` calls
`Store.ReplaceWebspaceSourceItems` with the *complete* item set every
sync (verified: `kernel/correlate/correlate.go:127`), a full re-walk on
every `Match` call is not an optimization choice — it is how the
existing sync-integrity machinery already makes "stat-diff" and "remove
detection" fall out for free (`D-02`): a file that no longer stat()s
simply isn't in this sync's returned set, and the kernel's own replace
semantics deletes its row. There is no OS filesystem-watcher dependency
anywhere in this design (`SRC-04`'s NFS/SMB requirement), which is
correct — `inotify`/`fanotify`/kqueue simply do not fire for changes made
by a *different* NFS/SMB client, so polling is the only correct
mechanism on a network mount, not merely a fallback.

Two pieces of new, real machinery are needed beyond "a sixth plugin
package": (1) a document-scope classifier (extension allowlist +
extras-driven glob widening/narrowing, `D-03`) that decides which files
become items and which of three preview shapes each gets (raw
bytes+MIME for PDF/image, kernel-rendered markdown, plain text, or
metadata-only), and (2) a brand-new, loopback-only kernel HTTP endpoint
that execs `xdg-open` against a real filesystem path resolved
server-side from an already-indexed item id — never a client-supplied
path (`D-06`). Neither of these has a direct in-repo precedent to copy
verbatim; the closest analog for the exec surface is
`kernel/httpapi/whatsapplink.go`'s link-subprocess spawner (a raw
`exec.CommandContext` outside the go-plugin gRPC boundary, resolved only
against a `pluginhost.DiscoverAllBinaries`-validated path) — the same
discipline (resolve identity from trusted server-side state, never from
the request body) applies here, but the target binary (`xdg-open`) and
lifecycle (fire-and-forget, no session/poll/cancel state machine) are
much simpler than a WhatsApp link session.

One item Fetch-pipeline gap was found by reading the kernel's own MIME
allowlist directly: `kernel/httpapi/item.go`'s `allowedRenditionTypes`
(lines 37-51) currently allows `application/pdf`, `image/png`,
`image/jpeg`, `image/gif`, `image/webp`, and `text/html` only —
**`text/plain` is absent**. `D-04`'s "plain text returns as text" preview
path will 415 (`unsupported_rendition_type`) against the live kernel
until this map gains a `text/plain: true` entry; this is a required
kernel-side one-line change, not an assumption, and the planner must
scope a task for it explicitly.

**Primary recommendation:** Build `plugins/filesystem` as a local-path
source (own `go.mod`, `CGO_ENABLED=0`, no cgo) modeled directly on
`plugins/signal`'s `Path`-only configuration shape and `plugins/silverbullet`'s
Fetch/preview branching, add `bmatcuk/doublestar/v4` for glob-based
extras scope overrides (stdlib `filepath.Match` has no `**` support),
add the missing `text/plain` MIME entry to `kernel/httpapi/item.go`, and
build the kernel-mediated open endpoint as new, narrowly-scoped machinery
resolved exclusively from the index (never from a plugin-declared
`deep_link` URL string or a request-supplied path).

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Folder walk + stat-diff scan | Plugin subprocess (`plugins/filesystem`) | — | Contract's `Match` RPC is the only place a plugin computes its item set; no kernel involvement per-file |
| Document-scope classification (allowlist + extras globs) | Plugin subprocess | Kernel (`config.Source.Extras`, generic passthrough) | Kernel never interprets extras values (D-12); the plugin owns all scope logic, kernel only carries the strings |
| Preview bytes/text generation | Plugin subprocess (`Fetch` RPC) | Kernel (sanitize/wrap/theme for `text/html`, MIME allowlist enforcement) | Hybrid data model: plugin supplies raw content, kernel is the sole sanitize/serve boundary (`kernel/httpapi/rendition.go`, `item.go`) |
| Deep-link "open" action | Kernel (new loopback HTTP route) | Plugin (declares intent only) | D-06 requires path resolution to happen server-side from indexed state, never from a plugin- or client-supplied string — same discipline as the existing `whatsapplink.go` exec surface |
| Item identity / removal detection | Kernel (`kernel/correlate`, `kernel/index`) | Plugin (declares `source_id` per D-01) | `Store.ReplaceWebspaceSourceItems`'s full-replace semantics already implement "file no longer present == item removed"; the plugin does no diffing of its own |
| Trust tier / external rehearsal | Kernel (`kernel/pluginhost`, directory-tier resolution) | — | Unchanged from Phase 11: tier is derived purely from which directory the binary resolves from at launch, never from the plugin |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| Go stdlib `path/filepath`, `os`, `io/fs` | Go 1.25 (repo floor, verified: `go.mod:3`) | Directory walk (`filepath.WalkDir`), stat calls, path joins | Zero new dependency for the core scan loop; every other plugin in this repo already uses only stdlib for local I/O (`plugins/signal`) |
| `github.com/bmatcuk/doublestar/v4` | v4.10.0 (latest tagged, confirmed via `go list -m -versions`, module proxy — `[VERIFIED: proxy.golang.org]`) | `**`-capable glob matching for the extras include/exclude scope override (D-03) | Go's stdlib `filepath.Match`/`filepath.Glob` do not support `**` (recursive-directory glob) at all — a real gap for "exclude everything under `**/node_modules/**`"-style overrides an operator will expect. `doublestar` is the de facto standard fill for this gap (`[ASSUMED]` — chosen from training knowledge + a WebSearch confirming it is still the actively maintained v4 line; not independently checked against a `SLOP`/`SUS` legitimacy scanner because the project's package-legitimacy tooling in this environment supports npm/PyPI/crates ecosystems only, not Go modules — treat as `[SUS]`-equivalent and gate its addition behind a `checkpoint:human-verify` task before first use) |
| `github.com/yuin/goldmark` | Already vendored (`plugins/silverbullet/go.sum`; reuse via a second `go.mod` require, not a new dependency in spirit) | Markdown → HTML fragment conversion for `.md` preview (D-04, "SilverBullet shape precedent") | Verified in-repo precedent: `plugins/silverbullet/render.go:22` (`var mdConverter = goldmark.New()`) — safe defaults (no raw-HTML passthrough, no dangerous URL schemes), kernel-side `bluemonday` sanitizer is the second layer per D-11 |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Go stdlib `mime` (`mime.TypeByExtension`) | stdlib | Extension → MIME mapping for the classifier | `[ASSUMED]` — training knowledge: `mime.TypeByExtension` on Linux augments its built-in table from `/etc/mime.types` and can vary slightly by distro for less-common extensions; the built-in table reliably covers `.pdf`/`.png`/`.jpg`/`.jpeg`/`.gif`/`.webp` (already kernel-allowlisted). **Recommend a small hand-rolled `map[string]string` for the fixed extension set this plugin's default allowlist covers** rather than trusting the OS table, so behavior doesn't drift across the user's own machine vs. CI vs. a future install — mirrors the "closed, explicit allowlist" spirit already used for `kernel/httpapi/item.go`'s `allowedRenditionTypes`. |
| `os/exec` (stdlib) | stdlib | Kernel-side `xdg-open <path>` invocation for the deep-link endpoint | New kernel machinery, not a plugin dependency — see Architecture Patterns below |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `doublestar/v4` for glob scope | Hand-rolled recursive-glob matcher | More code to test and maintain for a well-solved problem (`Don't Hand-Roll`, below) — reasonable only if the legitimacy checkpoint blocks the dependency |
| Full re-walk every `Match` call | A plugin-side persisted mtime cache (skip unchanged subtrees) | Full re-walk is simpler and matches the existing "Match returns the complete current set" contract exactly; a persisted cache adds a second source of truth for "is this item still there" that must itself never drift from the filesystem's own truth — premature for v1 unless a large-tree performance problem is actually observed |
| `xdg-open` exec via a new kernel route | A `file://` link opened directly by the browser | Browsers sandbox/block `file://` navigation from an `http://` origin inconsistently across engines and OS; D-06 explicitly locks the kernel-mediated `xdg-open` approach for this reason |

**Installation:**
```bash
mkdir plugins/filesystem && cd plugins/filesystem
go mod init github.com/davison/topos/plugins/filesystem
go get github.com/bmatcuk/doublestar/v4@v4.10.0
cd /home/darren/projects/davison/topos
go work use ./plugins/filesystem
```

**Version verification:** `go list -m -versions github.com/bmatcuk/doublestar/v4` was run this session against the live Go module proxy and returned versions through `v4.10.0` as the latest tag `[VERIFIED: proxy.golang.org]`. This confirms the module exists and is actively tagged; it does **not** independently confirm supply-chain trustworthiness (see Package Legitimacy Audit below).

## Package Legitimacy Audit

This phase's only new external dependency is a Go module. The project's
package-legitimacy tooling (`gsd_run query package-legitimacy check`) is
scoped to `npm`/`pypi`/`crates` ecosystems in this environment and does
not cover Go modules — so no automated `OK`/`SUS`/`SLOP` verdict could be
obtained this session. Treat the entry below as `[SUS]`-equivalent by
default per the protocol's fallback rule.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| `github.com/bmatcuk/doublestar/v4` | Go module proxy (proxy.golang.org) | Long-lived project (v1 predates v4; v4 line itself has 20+ tagged releases through v4.10.0) — `[ASSUMED]`, not independently dated this session | Not obtainable from the Go proxy (no download-count API); widely imported by well-known tools (`golangci-lint`, `helm`, others) per training knowledge — `[ASSUMED]` | `github.com/bmatcuk/doublestar` (confirmed via WebSearch result URLs) | Unverified (no automated Go-ecosystem check available) | **Flagged — planner must add a `checkpoint:human-verify` task before `go get`-ing this dependency**, confirming the module path and a pinned version against the GitHub repo directly |

**Packages removed due to `[SLOP]` verdict:** none — no automated scan ran.
**Packages flagged as suspicious `[SUS]`:** `github.com/bmatcuk/doublestar/v4` (tooling gap, not a specific red flag on the package itself — see disposition above).

## Architecture Patterns

### System Architecture Diagram

```
   Sync cycle (kernel/syncer.Coordinator, unchanged)
        │
        ▼
   kernel/correlate.Engine.SyncSource(ctx, filesystemSource)
        │  match_fields["folders"] = configured folder-path values (D-05)
        ▼
   plugins/filesystem  Match RPC
        │
        ├─ resolve scope: WEBSPACES_SOURCE_CONFIG.path (root) +
        │    extras.include_glob / extras.exclude_glob (D-03) +
        │    default document-ish extension allowlist
        │
        ├─ filepath.WalkDir(root) ── respecting recursive on/off ──┐
        │                                                          │
        │      for each candidate file:                           │
        │        stat (mtime, size) ── "stat-diff" signal ─────────┤
        │        classify extension → preview kind                │
        │        derive folder-path label for match comparison ────┤
        │        build Item{ source_id: relative path (D-01),      │
        │                     labels: [folder path],                │
        │                     fidelity: EXACT or ANCHORED,          │
        │                     deep_link: kernel open-route token }  │
        │                                                          │
        ▼                                                          │
   MatchResponse{ items }  ── the COMPLETE current file set ───────┘
        │
        ▼
   kernel/correlate: validateCorrelatedItem (fidelity + deep_link non-empty)
        │
        ▼
   Store.ReplaceWebspaceSourceItems(webspace, "filesystem", items)
        │   full replace — a file gone from this Match call's set is
        │   deleted from the index automatically (D-02: rename = remove+add)
        ▼
   Item stream (kernel/httpapi/stream.go) ── DetailPane ── OpenInSource.svelte
        │
        ├─ preview click → GET /api/items/{id}/content
        │      → pluginhost.Fetch(source, source_id, PREVIEW)
        │      → plugins/filesystem Fetch RPC re-reads the file fresh
        │        (PDF/image: raw bytes+MIME; .md: goldmark → HTML fragment,
        │         CONTENT_SHAPE_MARKDOWN_HTML; .txt: text only;
        │         office formats: available=false, metadata + deep link only)
        │      → kernel/httpapi/rendition.go sanitizes/wraps text/html,
        │        enforces allowedRenditionTypes (needs text/plain added)
        │
        └─ "Open in Source" click → NEW kernel loopback route
               resolves item id → index row → (source config Path root
               + item source_id) → validated real path
               → os/exec xdg-open <path>   (never a client-supplied path)
```

### Recommended Project Structure
```
plugins/filesystem/
├── go.mod                  # own module, CGO_ENABLED=0, like signal/silverbullet/paperless siblings
├── main.go                 # reads WEBSPACES_SOURCE_CONFIG, constructs SourcePlugin, goplugin.Serve
├── plugin.go                # Describe/Match/Fetch/Health — sdk.SourcePlugin implementation
├── scope.go                 # extension allowlist + doublestar include/exclude glob resolution (D-03)
├── walk.go                  # filepath.WalkDir wrapper: recursion toggle, hidden/symlink policy, stat-diff item building
├── classify.go               # extension → {preview kind, mime type} mapping (hand-rolled, not mime.TypeByExtension)
├── render.go                 # goldmark markdown → HTML fragment (mirrors plugins/silverbullet/render.go)
├── deeplink.go               # builds the item's deep_link value/token consumed by the new kernel open route
├── readonly_test.go           # AST scan: fails build on any os.WriteFile/os.Remove/os.Create/os.Rename/os.Mkdir/etc. reference (mirrors plugins/signal/readonly_test.go's SQL-write scan idiom)
└── *_test.go                  # unit tests per file above

kernel/httpapi/
├── item.go                    # existing — add "text/plain": true to allowedRenditionTypes (line ~50)
└── fsopen.go                   # NEW — loopback-only xdg-open route, resolved from index state only
```

### Pattern 1: Local-path source, `Path`-only config (no `base_url`/`token`)
**What:** `config.Source.Path` is populated instead of `BaseURL`/`Token`; `kernel/config/config.go`'s `Validate` already branches on this (verified: `kernel/config/config.go:339` — `if strings.TrimSpace(src.Path) == "" { ... require base_url+token ... }`), so a filesystem source declaring `path` alone passes validation with zero kernel changes to the validation branch itself.
**When to use:** Always, for this plugin — this is the exact shape `plugins/signal` already established.
**Example:**
```toml
[sources.docs-folder]
plugin = "topos-plugin-filesystem"
path = "/mnt/nas/household-docs"   # "~" expansion is the PLUGIN's job, not the kernel's — see plugins/signal/README.md precedent

[sources.docs-folder.extras]
include_glob = "**/*.pdf,**/*.md"   # single string value only (D-13) — plugin splits on a chosen delimiter
exclude_glob = "**/node_modules/**,**/.git/**"
```
(Source: `plugins/signal/README.md:50-58`, `kernel/config/types.go:129-143`, both read this session.)

### Pattern 2: Preview-kind branching in `Fetch` (mirrors `plugins/paperless/plugin.go`)
**What:** `Fetch`'s `switch req.GetVariant()` dispatches to a per-preview-kind handler, exactly like paperless's `fetchFull`/`fetchRendition` split (verified: `plugins/paperless/plugin.go:164-233`).
**When to use:** For every `Item` this plugin returns — the preview kind (PDF/image raw bytes, markdown→HTML, plain text, metadata-only) is decided once by `classify.go` at `Match` time and re-derived identically at `Fetch` time (never cached between the two calls, matching the "Fetch re-fetches fresh from source" hybrid-model rule the contract states for every plugin).
**Example:**
```go
// Source: plugins/paperless/plugin.go:164-180 (structure), adapted
func (p *SourcePlugin) Fetch(ctx context.Context, req *toposv1.FetchRequest) (*toposv1.FetchResponse, error) {
	relPath := req.GetSourceId() // D-01: source_id IS the relative path
	switch classifyKind(relPath) {
	case kindPDFOrImage:
		return p.fetchRawBytes(relPath)
	case kindMarkdown:
		return p.fetchRenderedMarkdown(relPath) // goldmark -> CONTENT_SHAPE_MARKDOWN_HTML
	case kindPlainText:
		return p.fetchPlainText(relPath)
	case kindMetadataOnly: // office formats
		return &toposv1.FetchResponse{Available: false, UnavailableReason: "preview not supported for this file type; open in source"}, nil
	default:
		return nil, status.Errorf(codes.NotFound, "filesystem: item %q not found", relPath)
	}
}
```

### Pattern 3: The kernel-mediated deep-link "open" endpoint (NEW machinery — no direct copy-paste precedent)
**What:** A loopback-only HTTP route, e.g. `POST /api/items/{id}/open`, that: (1) resolves `{id}` to an index row exactly like `ItemHandler`/`renditionHandler` already do (verified pattern: `kernel/httpapi/item.go:95-136`, `store.GetItem(ctx, id)`), (2) looks up that item's owning source's configured `Path` from `cfgStore.Expanded().Sources[it.Source].Path`, (3) joins it with the item's `SourceID` (the relative path, D-01), (4) validates the resulting absolute path is still lexically inside the source's `Path` root (defense against a `../`-crafted `source_id` somehow reaching this point), and (5) execs `xdg-open <path>` via `os/exec.CommandContext`, fire-and-forget, logging failure but never blocking the HTTP response on the child process's own exit.
**When to use:** Exactly this one new route. This is the "small, loopback-only kernel endpoint" D-06 names.
**Design note — resolving the identity problem:** a plugin subprocess is not told the kernel's own listen address (`[server].listen`, default `127.0.0.1:7777`) anywhere in the current launch environment (verified: the allowlist in `docs/plugin-contract.md`'s "The launch environment" section carries no such field, and `kernel/config/types.go`'s `Config.Server.Listen` is never threaded into `WEBSPACES_SOURCE_CONFIG` — read this session, no such wiring exists in `kernel/config/types.go`). This means the plugin **cannot itself construct** `http://127.0.0.1:7777/api/items/{source}:{source_id}/open` as its `deep_link` value — it doesn't know its own source instance id either (contract: "A plugin never learns, asserts, or needs its own instance identity"). **Recommended resolution:** the plugin sets `deep_link` to a `file://` URI built from its own knowledge of the real absolute path (it has both the configured root and the relative `source_id`), e.g. `file:///mnt/nas/household-docs/receipts/2026-invoice.pdf`. The kernel's item-serving layer (`kernel/httpapi/stream.go:202`, which today echoes `it.DeepLink` verbatim into the JSON `Link.url` field) is extended with a small, source-agnostic rewrite: any item whose `DeepLink` carries the `file://` scheme has its served `Link.url` rewritten to the new loopback open route (`/api/items/{id}/open`) instead of the raw `file://` value — keyed off the URL *scheme*, not the plugin's `source_type`, so this stays consistent with the "no built-in table of known plugin types" discipline (`docs/plugin-contract.md`, D-05 lineage) rather than special-casing `source_type == "filesystem"`. `validateCorrelatedItem` (verified: `kernel/correlate/correlate.go:230-238`) only checks non-empty, so a `file://` value passes unmodified today. **This is the phase's one genuinely new architectural decision — flag it to the user/planner explicitly rather than treating it as settled**, since CONTEXT.md's D-06 describes the endpoint's behavior and security constraint but not this specific plugin-to-kernel signaling mechanism.
**Example (illustrative, not verified against existing code — this route does not exist yet):**
```go
// NEW FILE: kernel/httpapi/fsopen.go — modeled on whatsapplink.go's
// "resolve identity server-side, never trust the request" discipline
// (kernel/httpapi/whatsapplink.go:701-705's binPath comment) but far
// simpler: no session, no poll, no kill — one exec, fire-and-forget.
func FilesystemOpenHandler(store *index.Store, cfgStore *config.Store, logger hclog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := itemIDParam(r)
		it, ok, err := store.GetItem(r.Context(), id)
		if err != nil || !ok {
			WriteError(w, http.StatusNotFound, "item_not_found", "item not found")
			return
		}
		src, ok := cfgStore.Expanded().Sources[it.Source]
		if !ok || src.Path == "" {
			WriteError(w, http.StatusNotFound, "item_not_found", "source has no local path configured")
			return
		}
		root, err := filepath.Abs(src.Path)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "internal_error", err.Error())
			return
		}
		full := filepath.Join(root, it.SourceID) // it.SourceID is the D-01 relative path, from the INDEX, never the request
		if !strings.HasPrefix(full, root+string(filepath.Separator)) {
			WriteError(w, http.StatusBadRequest, "invalid_path", "resolved path escapes source root")
			return
		}
		cmd := exec.Command("xdg-open", full)
		if err := cmd.Start(); err != nil {
			WriteError(w, http.StatusBadGateway, "open_failed", err.Error())
			return
		}
		go func() { _ = cmd.Wait() }() // never block the response on the opener's own exit
		WriteJSON(w, http.StatusOK, map[string]bool{"opened": true})
	}
}
```

### Anti-Patterns to Avoid
- **Trusting `mime.TypeByExtension` for the closed default allowlist:** stdlib behavior varies by OS/distro `/etc/mime.types` presence — use an explicit `map[string]string` matching exactly the extensions this plugin's document-ish allowlist declares, mirroring the kernel's own closed-allowlist discipline in `kernel/httpapi/item.go`.
- **Letting a plugin-supplied path reach `xdg-open` unresolved:** the new open route must resolve the real filesystem path from the **index** (server-side, trusted) using the item id, never accept a path string in the request body or query string — this is the single sharpest security surface this phase introduces (mirrors T-01-10's rendition-handler discipline and T-08-06's "directory listing is the authority" discipline for `WhatsAppLinkStartHandler`).
- **Reading full file content on every `Match` call to build the preview snippet:** for large trees this defeats the point of a lightweight stat-diff scan. `Item.preview` may legitimately be left `""` at Match time for this source (the contract states `preview` is optional) — full content/preview generation belongs in `Fetch`, called only on item-open, exactly like every other plugin already does.
- **Special-casing `source_type == "filesystem"` anywhere in kernel code:** the whole contract's discipline is "the kernel holds no built-in table of known plugin types" (D-05 lineage) — the `file://`-scheme deep-link rewrite recommended above is deliberately keyed off the URL scheme, not the plugin's declared identity, so a future third-party local-path plugin gets the same treatment for free.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Recursive glob matching (`**/*.pdf`, `**/node_modules/**`) | A custom path-segment matcher | `bmatcuk/doublestar/v4` | Correct `**` semantics (arbitrary depth, `..` handling, `{a,b}` alternation) is a well-solved, easy-to-get-subtly-wrong problem; stdlib's `filepath.Match` explicitly does not support `**` |
| Markdown → HTML conversion | A hand-rolled markdown parser | `goldmark` (already vendored via `plugins/silverbullet`) | Markdown parsing has a huge edge-case surface (tables, footnotes, embedded HTML); goldmark is already the project's chosen, audited-for-safe-defaults converter |
| Directory tree walking with symlink/hidden-file policy | Manual `os.ReadDir` recursion | stdlib `filepath.WalkDir` | `WalkDir` already handles `io/fs.SkipDir`, error propagation per entry, and lazy `os.Lstat`-then-`Stat` semantics correctly — reimplementing it risks silently mishandling symlink loops or permission-denied subtrees |

**Key insight:** every "don't hand-roll" item above is a correctness trap with a large blast radius (a broken glob silently excludes or includes the wrong files across an entire tree; a broken walker can loop forever on a symlink cycle or skip permission-denied subtrees silently) — none of them is performance-sensitive enough to justify a bespoke implementation for a single-user desktop tool.

## Common Pitfalls

### Pitfall 1: `text/plain` is not in the kernel's rendition MIME allowlist
**What goes wrong:** D-04's "plain text returns as text" preview path serves a 415 `unsupported_rendition_type` from `GET /api/items/{id}/content` even though the plugin's `Fetch` succeeds.
**Why it happens:** `kernel/httpapi/item.go`'s `allowedRenditionTypes` map (verified, lines 37-51: `"application/pdf": true, "image/png": true, "image/jpeg": true, "image/gif": true, "image/webp": true, "text/html": true`) has no `text/plain` entry — this predates the filesystem plugin, since no prior source ever needed a plain-text rendition (paperless returns PDF; silverbullet/proton/signal all render as `text/html`).
**How to avoid:** Add `"text/plain": true` to that map as an explicit task in this phase's plan — it's a one-line, low-risk kernel change, but it must not be missed or discovered only during UAT.
**Warning signs:** A `.txt` item's detail pane shows extracted text (from the `GET /api/items/{id}` route's `content.text` field, which has no MIME allowlist) but its content/preview route 415s — the two routes have different guard rules, which is easy to miss when eyeballing only the item-detail JSON.

### Pitfall 2: The plugin cannot construct its own kernel-loopback deep link
**What goes wrong:** A naive first implementation tries to set `deep_link` to `http://127.0.0.1:7777/api/items/.../open`, hardcoding the default listen address — breaks the moment an operator changes `[server].listen`, and the plugin has no way to learn its own source instance id to build the full `{source}:{source_id}` composite anyway (contract: "A plugin never learns... its own instance identity").
**Why it happens:** The launch environment allowlist (`docs/plugin-contract.md`, verified this session) deliberately does not include the kernel's own base URL — nothing in the current contract threads it through.
**How to avoid:** Use the `file://`-scheme-plus-kernel-side-rewrite design in Architecture Pattern 3 above, or (the heavier alternative) extend the contract to pass the kernel's own base URL as a new built-in `WEBSPACES_SOURCE_CONFIG` field — a genuine contract change with third-party-plugin-author implications, likely overkill for this one feature. Flag this design fork explicitly at plan time; it is not resolved by CONTEXT.md.
**Warning signs:** `deep_link` values that hardcode a port number, or a `TODO` guessing at "the kernel will fix this up somehow."

### Pitfall 3: `filepath.WalkDir` on a symlink cycle or a permission-denied subtree
**What goes wrong:** An in-tree symlink pointing back at an ancestor directory causes infinite recursion if followed naively; a permission-denied subdirectory (common on a shared NFS/SMB mount with mixed ACLs) can abort the entire walk if the `WalkDirFunc`'s error isn't handled per-entry.
**Why it happens:** `filepath.WalkDir` does NOT follow symlinks by default (it uses `Lstat` semantics for the walked entries themselves) — but a plugin that explicitly `os.Stat`s a symlink target to decide "is this a regular file" (necessary to classify `.pdf`/`.md`/etc.) can be tricked into an infinite loop if it also recurses into symlinked directories. A single `os.ReadDir` permission error on one subtree, if returned as a hard `error` from the `WalkDirFunc` instead of `filepath.SkipDir`, aborts the entire remaining walk.
**How to avoid:** Do not recurse into symlinked directories by default (CONTEXT.md's discretion item explicitly calls out "in-tree symlink handling defaults" as the planner's call — recommend: symlinked files are followed for classification purposes, symlinked directories are NOT descended into, matching the common `rsync --safe-links`-style default); on a per-entry walk error, log and `return filepath.SkipDir` (or continue) rather than aborting the sync for the whole source.
**Warning signs:** A sync that never completes on a real NAS mount, or a sync that reports zero items despite the folder clearly containing matching files (one early permission error silently killed the rest of the walk).

### Pitfall 4: NFS/SMB mtime granularity and clock skew defeat naive stat-diff
**What goes wrong:** Some SMB/CIFS mounts report mtime at 2-second granularity (a legacy FAT-era convention some implementations still carry), and NFS client-side attribute caching can serve a stale mtime for several seconds after a remote write. A stat-diff signal built purely on "did mtime change since last observed value" can miss or delay detecting a genuine change within that window.
**Why it happens:** This is inherent to the network filesystem protocols themselves, not a bug in any particular Go code — `[ASSUMED]`, based on general systems knowledge of SMB/NFS attribute semantics, not verified against a specific mount in this session.
**How to avoid:** Since this plugin performs a full re-walk with a full `Match` response every sync cycle (not an incremental diff persisted between cycles — see Architecture, above), the practical mitigation is simply "the next scheduled sync interval will catch it" — there's no persisted "last known mtime" state to go stale in the first place. Document the sync interval as the real freshness bound for network mounts, and don't design a plugin-side incremental cache that could itself get stuck on a stale mtime.
**Warning signs:** A user reports "I changed the file five minutes ago and it's still showing the old preview" — check whether the sync interval, not the stat-diff logic itself, explains the delay.

### Pitfall 5: Office-format MIME types are not in the kernel's rendition allowlist (by design)
**What goes wrong:** A naive implementation might try to return `application/vnd.openxmlformats-officedocument.wordprocessingml.document` bytes from `Fetch` "just in case the browser can render it" — it can't, and the kernel's `allowedRenditionTypes` allowlist would 415 it anyway even if it tried.
**Why it happens:** D-04 explicitly scopes office formats to "metadata + deep link only" this phase.
**How to avoid:** For office-format files, `Fetch` should return `available: false` with a clear `unavailable_reason` (matching the paperless precedent for "no previewable rendition," verified: `plugins/paperless/plugin.go:156`, `noRenditionReason`) and never attempt to set `mime_type`/`data` at all.
**Warning signs:** A PR that adds new entries to `kernel/httpapi/item.go`'s `allowedRenditionTypes` for office MIME types — that's out of scope for this phase per the Deferred Ideas list.

## Code Examples

### `Describe` — folder-path match vocabulary (D-05)
```go
// Modeled on plugins/paperless/plugin.go:67-76 and the contract's stated
// match_vocabulary shape (docs/plugin-contract.md: the four in-repo
// examples are ["folders"] (proton), ["tags"] (paperless), ["tags",
// "pages"] (silverbullet), ["conversations"] (signal) — "folders" for
// this plugin mirrors proton's exactly, since both use the source's own
// native directory/folder categorization).
var matchVocabulary = []string{"folders"}

func (p *SourcePlugin) Describe(_ context.Context, _ *toposv1.DescribeRequest) (*toposv1.DescribeResponse, error) {
	return &toposv1.DescribeResponse{
		SourceType:      "filesystem",
		DisplayName:     "Filesystem folder",
		ContractVersion: "topos.v2",
		MatchVocabulary: matchVocabulary,
		Extras: []*toposv1.ExtrasField{
			{Key: "include_glob", Label: "Include glob (comma-separated)", Required: false, Secret: false, Placeholder: "**/*.pdf,**/*.md"},
			{Key: "exclude_glob", Label: "Exclude glob (comma-separated)", Required: false, Secret: false, Placeholder: "**/node_modules/**"},
		},
	}, nil
}
```
(Structure verified against `plugins/paperless/plugin.go:67-76`; `ExtrasField` shape verified against `proto/topos/v1/plugin.proto:75-101` and `testdata/external-plugin/plugin.go:105-121`'s worked example.)

### Read-only guard — filesystem-write AST scan (mirrors the signal SQL-write scan idiom)
```go
// plugins/filesystem/readonly_test.go — models plugins/signal/readonly_test.go's
// AST-walk-not-text-match idiom (verified: plugins/signal/readonly_test.go:36-60),
// adapted from database/sql write-selector names to os-package write-selector names.
var disallowedOSSelectors = map[string]bool{
	"WriteFile": true, "Remove": true, "RemoveAll": true, "Create": true,
	"OpenFile": true, "Rename": true, "Mkdir": true, "MkdirAll": true,
	"Chmod": true, "Chown": true, "Truncate": true, "Symlink": true, "Link": true,
}
// walk this package's own non-test .go files (mirrors plugins/signal's
// filepath.WalkDir(".", ...) scope, since — unlike the HTTP scan, which
// walks the whole plugins/ tree — this plugin's write surface is package-local)
```

### Full/preview/thumbnail `ContentVariant` dispatch
```go
// Mirrors plugins/paperless/plugin.go:164-180's switch shape exactly —
// same three-branch dispatch, different per-branch bodies.
func (p *SourcePlugin) Fetch(ctx context.Context, req *toposv1.FetchRequest) (*toposv1.FetchResponse, error) {
	switch req.GetVariant() {
	case toposv1.ContentVariant_CONTENT_VARIANT_FULL:
		return p.fetchFull(ctx, req.GetSourceId())
	case toposv1.ContentVariant_CONTENT_VARIANT_PREVIEW:
		return p.fetchPreview(ctx, req.GetSourceId())
	case toposv1.ContentVariant_CONTENT_VARIANT_THUMBNAIL:
		return &toposv1.FetchResponse{Available: false, UnavailableReason: "no thumbnail rendition"}, nil
	default:
		return nil, status.Error(codes.InvalidArgument, "filesystem: unspecified content variant")
	}
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| OS filesystem watchers (`inotify`/`fanotify`) for change detection | Interval-based stat-diff polling | N/A — this is a per-mount-type constraint, not a version-driven change | Network mounts (NFS/SMB) never deliver local `inotify` events for a remote client's writes; polling is the only correct mechanism regardless of how new or old the watcher API is |
| `filepath.Walk` | `filepath.WalkDir` (`io/fs.WalkDirFunc`, Go 1.16+) | Go 1.16 (2021) | `WalkDir` avoids an extra `os.Lstat` per entry that `Walk`'s `os.FileInfo`-based callback required — faster for large trees, and this repo's floor is Go 1.25 so there is no reason to use the older API |

**Deprecated/outdated:**
- `filepath.Walk`: superseded by `filepath.WalkDir` since Go 1.16; use `WalkDir` for any new tree-walking code in this repo.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `bmatcuk/doublestar/v4` is the right glob library and is itself trustworthy | Standard Stack, Package Legitimacy Audit | No automated Go-ecosystem legitimacy check was available this session; mitigated by a mandatory `checkpoint:human-verify` task before first `go get` |
| A2 | `mime.TypeByExtension` varies by OS/distro for less-common extensions | Standard Stack, Anti-Patterns | If wrong (i.e., stdlib behavior is actually fully deterministic across all target platforms), the recommendation to hand-roll a MIME map is merely extra caution, not a correctness bug — low risk either way |
| A3 | Recursion on/off is best modeled as a new typed `config.Source` boolean field rather than an extras string key | Architecture Patterns (implicit in Recommended Project Structure) | CONTEXT.md does not lock this — it is absent from both the Decisions and the Discretion list verbatim, though the phase's own success-criterion wording ("subfolder recursion on or off" as a UI-level toggle) suggests a typed boolean fits better than a string extras value. If the planner instead threads it through extras (string "true"/"false"), the add-source form would render a text input instead of a checkbox — a worse UX for a binary choice, but functionally workable. **Flag this explicitly for planner/user confirmation** — it is not resolved by CONTEXT.md's decisions. |
| A4 | SMB/NFS mtime granularity and attribute-cache staleness are real, relevant risks for this phase | Common Pitfalls (Pitfall 4) | General systems knowledge, not verified against a specific NFS/SMB mount in this session; if wrong, Pitfall 4's mitigation (rely on sync interval, don't build a plugin-side incremental cache) is still sound advice regardless |
| A5 | `os.Environ()`-based launch env has no kernel-base-URL field a plugin could read | Architecture Pattern 3, Pitfall 2 | Verified this session by reading `docs/plugin-contract.md`'s full "The launch environment" section and `kernel/config/types.go` in full — no such field exists today. Low risk of being wrong. |

**If this table is empty:** N/A — see entries above.

## Open Questions

1. **How does the plugin signal "this item needs the kernel-mediated open route" without knowing the kernel's own address or its own instance id?**
   - What we know: The contract deliberately withholds both pieces of information from a plugin subprocess (verified this session against `docs/plugin-contract.md` and `kernel/config/types.go`). `kernel/httpapi/stream.go:202` echoes `Item.DeepLink` verbatim today, with no existing transform hook.
   - What's unclear: Whether the planner should (a) introduce a `file://`-scheme convention the kernel rewrites at serve time (recommended above, keyed off URL scheme not source_type), (b) extend the contract with a new built-in "kernel base URL" field threaded into `WEBSPACES_SOURCE_CONFIG` (heavier, third-party-plugin-facing contract change), or (c) some other mechanism.
   - Recommendation: Option (a) — smallest blast radius, no contract version bump, stays consistent with "no built-in table of plugin types" since the trigger is a URL scheme, not a hardcoded `source_type` string. Surface this as a locked decision at plan time, since it is genuinely new architecture CONTEXT.md's D-06 named the *what* but not the *how* for.

2. **Should "subfolder recursion on/off" be a new typed `config.Source` field or an extras string key?**
   - What we know: D-03 explicitly scopes extras to include/exclude glob overrides; recursion is not mentioned there. The phase's own success criterion 1 describes it as a UI-level on/off toggle.
   - What's unclear: CONTEXT.md's Discretion list does not explicitly assign this decision to either bucket.
   - Recommendation: A new typed `Recursive bool` field on `config.Source` (default `false`, matching Go's zero value and the conservative "don't silently traverse a huge tree" default) — gives the add-source UI a real checkbox rather than a text input for a binary choice, and keeps extras scoped purely to D-03's glob-override use case as CONTEXT.md describes it.

3. **What extension set constitutes the default "document-ish" allowlist?**
   - What we know: D-03 names four categories — PDF, office formats, markdown, plain text, images — and explicitly defers the exact extension list to Claude's Discretion.
   - What's unclear: The precise extension list (e.g., does "office formats" include legacy `.doc`/`.xls`, or only the modern `.docx`/`.xlsx` Open XML forms? Does "images" include `.svg`, `.bmp`, `.tiff` — none of which are in the kernel's current rendition MIME allowlist?).
   - Recommendation: Scope the default image allowlist to exactly the MIME types the kernel already renders inline (`png`, `jpeg`/`jpg`, `gif`, `webp`) so every allowlisted image gets a real preview with zero kernel changes; treat any other image extension (`.svg`, `.bmp`, `.tiff`, `.heic`) as metadata-only unless/until the kernel's `allowedRenditionTypes` is deliberately widened. For office formats, include both legacy and OOXML extensions (`.doc`, `.docx`, `.xls`, `.xlsx`, `.ppt`, `.pptx`, `.odt`, `.ods`, `.odp`) since all get identical metadata-only treatment regardless of exact format — no reason to be narrower there.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| `xdg-open` (or platform equivalent) | The kernel-mediated open endpoint (D-06) | Not probed this session — assumed present on any standard Linux desktop per `xdg-utils`, which is a near-universal desktop dependency; the project is explicitly Linux-anchored (`.claude/CLAUDE.md`, Phase 11 D-09's "Windows: ... without committing to Windows support") | — | If absent, `os/exec.Command("xdg-open", ...).Start()` fails at exec time with a clear error — the endpoint should surface this as a named, honest failure (`open_failed`), never a silent no-op |
| `bmatcuk/doublestar/v4` (Go module) | Extras-driven glob scope override (D-03) | Resolvable via the Go module proxy — confirmed this session (`go list -m -versions`) | v4.10.0 latest tagged | None needed if present; if the legitimacy checkpoint blocks it, fall back to stdlib `filepath.Match` with a documented "no `**` support" limitation for the extras glob feature only (the default extension allowlist itself needs no glob library at all) |
| A local or network-mounted test folder (NFS/SMB) for criterion 2's rehearsal | UAT / e2e coverage of the network-mount stat-diff claim | Not probed this session — likely requires either a real NAS/SMB share reachable from the dev machine, or a locally-mounted loopback NFS/SMB server for CI-safe testing | — | If no real network mount is available in CI, the e2e/UAT plan should still assert correctness against a plain local directory (the polling logic is mount-type-agnostic by construction — it never calls a filesystem-watcher API) and treat the NFS/SMB-specific claim as a manual verification note, consistent with `docs/testing.md`'s "extend the Playwright suite as definition of done" convention wherever a browser can drive the check |

**Missing dependencies with no fallback:** none identified — every dependency above has a documented fallback or degrades honestly.

**Missing dependencies with fallback:** `xdg-open` presence unverified (assumed standard); `doublestar` gated behind a legitimacy checkpoint with a stdlib fallback for the narrower glob feature only.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | No | This phase adds no auth surface; the new open route inherits the kernel's existing loopback-only, unauthenticated-by-design boundary (verified: `kernel/httpapi/agent.go:5` describes "same unauthenticated loopback boundary" as the project's existing posture; `cmd/topos/main.go:440-447` — loopback-bind-is-the-security-default, verified this session) |
| V4 Access Control | Yes | The open route must resolve the target path exclusively from server-side index state keyed by an already-synced item id — never a request-supplied path or a plugin-declared arbitrary string (D-06's explicit constraint) |
| V5 Input Validation | Yes | Path-join + prefix-check against the resolved source root (defense-in-depth against a crafted `source_id`/relative path escaping via `../` segments), even though `source_id` originates from the plugin's own trusted walk, not directly from an HTTP request |
| V12 File and Resources | Yes | The plugin itself must never write to the source folder (existing read-only-by-construction contract discipline, PLUG-02) — enforced mechanically via a package-local AST scan for filesystem-write selectors, mirroring `plugins/signal/readonly_test.go`'s SQL-write scan idiom |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| Path traversal via a crafted relative `source_id` reaching the `xdg-open` exec call | Tampering / Elevation of Privilege | Resolve the path from the plugin's own walk-derived `source_id` stored in the trusted local index (never from an HTTP request body/query), then re-validate the joined absolute path is still lexically prefixed by the configured source root before exec — matches the recommended `fsopen.go` example above |
| A malicious or buggy plugin writing into the watched folder | Tampering | Mechanical AST scan (build-time test) blocking any `os` package write-selector reference in the plugin's own non-test source, mirroring the existing HTTP-verb scan (`plugins/paperless/readonly_test.go`) and SQL-verb scan (`plugins/signal/readonly_test.go`) idioms already enforced repo-wide |
| Arbitrary command execution via the new `xdg-open` exec surface if the binary name or arguments were ever influenced by request input | Elevation of Privilege | The binary name (`"xdg-open"`) is a fixed string literal, never derived from configuration or a request; the sole argument is the server-resolved, root-validated absolute path — mirrors the existing `whatsapplink.go` discipline of resolving `binPath` only from `pluginhost.DiscoverAllBinaries`, never from request-supplied data (verified: `kernel/httpapi/routes.go:701-705`'s comment on `binPath`) |
| Symlink-based scope escape (a symlink inside the watched folder pointing outside its root) | Information Disclosure | Per the recommended default in Pitfall 3, do not descend into symlinked directories; for symlinked *files*, the resolved real path (via `filepath.EvalSymlinks` or an equivalent check) should still be validated against the source root before it is ever exec'd via the open route — flag as a planner task, since D-06's "constrained to paths of currently-indexed items" already implies this but the walk-time symlink policy interacts with it |

## Sources

### Primary (HIGH confidence)
- `docs/plugin-contract.md` (read in full this session) — the published `topos.v1`/`topos.v2` contract: RPC semantics, `Item` fields, `LinkFidelity`/`ContentVariant`/`ContentShape` enums, launch-environment allowlist, extras shape
- `proto/topos/v1/plugin.proto` (read in full this session) — wire truth, line-verified for every enum/message quoted above
- `kernel/config/types.go` (read in full this session) — `Source`/`Webspace`/`PluginsConfig` struct shapes, `Path`/`Extras` field doc comments
- `kernel/config/config.go:328-382` (read this session) — `Validate`'s local-path-source branch (`src.Path` requirement logic)
- `kernel/correlate/correlate.go` (read in full this session) — `SyncSource`'s full-replace persistence semantics (`Store.ReplaceWebspaceSourceItems`), `validateCorrelatedItem`'s exact validation rules
- `kernel/httpapi/item.go` (read in full this session) — `allowedRenditionTypes` MIME allowlist (missing `text/plain`, verified), `ItemHandler`/`renditionHandler` request-time Fetch-call pattern
- `kernel/httpapi/stream.go:202` (read this session) — confirms `DeepLink` is echoed verbatim into `Link.url` today, no existing transform
- `kernel/httpapi/whatsapplink.go`, `routes.go:79-136` (read in full this session) — the closest existing precedent for a kernel-side raw-subprocess exec surface outside the go-plugin gRPC boundary
- `plugins/paperless/plugin.go`, `readonly_test.go` (read in full this session) — Fetch/ContentVariant dispatch pattern, HTTP-verb AST-scan idiom
- `plugins/silverbullet/render.go` (read in full this session) — goldmark markdown-rendering precedent
- `plugins/signal/plugin.go`, `README.md`, `readonly_test.go` (read this session) — local-path (`Path`-only) source config precedent, SQL-write AST-scan idiom
- `web/src/lib/components/OpenInSource.svelte`, `web/src/lib/format.ts` (read this session) — existing deep-link click affordance (`<a href target=_blank>`), `fidelityAffordance` two-class split
- `web/e2e/fixtures/plugin-binaries.ts`, `config-builder.ts` (read this session) — external-tier rehearsal fixture mechanics (`externalPluginBinaries`, `externalPluginBinariesSrcDir`, pin-hash writing)
- `testdata/external-plugin/plugin.go` (read in full this session) — Phase 11's out-of-repo proof-plugin shape, directly reusable as the rehearsal pattern for this phase's criterion 5
- `.planning/phases/11-external-plugins-the-trust-boundary/11-CONTEXT.md` (read in full this session) — locked Phase 11 decisions this phase inherits (D-09 through D-15)
- `.planning/phases/12-filesystem-source/12-CONTEXT.md` (read in full this session) — this phase's locked decisions (D-01 through D-06)
- `go.work`, `go.mod` (read this session) — Go 1.25 floor, existing workspace module list

### Secondary (MEDIUM confidence)
- WebSearch: "bmatcuk/doublestar go glob library current version github" — confirms v4 is the current, actively maintained major version line, standard `**` glob support
- `go list -m -versions github.com/bmatcuk/doublestar/v4` (run this session against the live Go module proxy) — confirms tagged versions through v4.10.0 exist and are resolvable

### Tertiary (LOW confidence)
- SMB/NFS mtime-granularity and attribute-cache-staleness claims (Pitfall 4) — general systems knowledge, not independently verified against a specific mount this session, marked `[ASSUMED]`
- `mime.TypeByExtension`'s distro-dependent behavior claim — general Go stdlib knowledge, not independently verified this session, marked `[ASSUMED]`
- `xdg-open` presence assumption — not probed on the actual target environment this session

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH for stdlib usage and in-repo-precedented libraries (goldmark); MEDIUM for `doublestar` (real but not legitimacy-scanned)
- Architecture: HIGH for everything traceable to a read, line-cited kernel/plugin file; MEDIUM for the deep-link signaling mechanism (Pattern 3/Open Question 1), which is genuinely new and not yet a locked decision
- Pitfalls: HIGH for the `text/plain` MIME-allowlist gap and the plugin-can't-know-its-own-address gap (both verified by direct file reads); LOW/ASSUMED for the NFS/SMB mtime-granularity claim

**Research date:** 2026-08-13
**Valid until:** 30 days (stable, in-repo-precedent-heavy domain; re-check `doublestar` version and `xdg-utils` availability if this research is reused past that window)
