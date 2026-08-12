# Stack Research — v1.1.0 "Plugin Ecosystem"

**Domain:** Additions to an existing Go kernel + gRPC-subprocess plugin architecture + SvelteKit SPA — external/untrusted plugin loading, a filesystem source plugin, an out-of-repo Google Drive source plugin, user-owned per-item marks, and PWA installability
**Researched:** 2026-08-12
**Confidence:** HIGH (Google API/OAuth libraries, fsnotify limitations, PWA manifest requirements, SQLite schema pattern — grounded directly in this repo's own code) / MEDIUM (binary trust-marking approach — no single "correct" answer, this is a judgment call) / LOW (mobile PWA installability over LAN without HTTPS — genuine open constraint, flagged below, not solved)

**Scope note:** This file covers only the five v1.1.0 target features. The v1.0 stack (Go 1.25, `hashicorp/go-plugin` gRPC subprocess plugins, `modernc.org/sqlite`+FTS5, `go-imap` v1, `whatsmeow`, SQLCipher via cgo, SvelteKit 2/Svelte 5 + `adapter-static`) is unchanged and not re-litigated here — see the milestone's carried-over `.planning/PROJECT.md` for that context. Versions below were checked against pkg.go.dev/npm on 2026-08-12 and cross-referenced against this repo's actual `go.mod`/`web/package.json` (Go 1.25.0, Vite 8, SvelteKit 2.63, `hashicorp/go-plugin` v1.8.0) so recommendations match what's already pinned, not a generic greenfield pick.

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| `google.golang.org/api/drive/v3` | v0.292.0 (google.golang.org/api, published Aug 2026) | Google Drive REST client for the out-of-repo Drive plugin | The official, Google-maintained Go client — auto-generated from the Drive v3 discovery doc, in maintenance mode (stable API surface, security/bugfixes only, no churn) which is exactly what a "docs in a folder" read-only integration needs. `drive.NewService(ctx, option.WithTokenSource(...))` plus `Files.List` with a `'<folderID>' in parents` query and `Files.Get(...).Download()` covers the whole feature (list folder contents, optional recursion via repeated `Files.List` per discovered sub-folder id, fetch content on item-open) with zero hand-rolled HTTP/JSON. |
| `golang.org/x/oauth2` + `golang.org/x/oauth2/google` | v0.36.0 (published Feb 2026) | OAuth 2.0 desktop/installed-app flow for the Drive plugin | This is the standard low-level OAuth2 client every Google Go example builds on (`google.golang.org/api`'s own `option.WithTokenSource` expects an `oauth2.TokenSource`). For a desktop app, use the **loopback IP redirect** flow (`http://127.0.0.1:<ephemeral-port>/callback`) — Google's currently-supported "installed app" pattern (out-of-band/copy-paste `urn:ietf:wg:oauth:2.0:oob` was deprecated and removed by Google in 2022) — not a webview or an embedded client secret treated as confidential. `oauth2.Config.AuthCodeURL` + a short-lived local `http.Server` on `127.0.0.1:0` to catch the redirect is the whole flow; `Config.Exchange` then yields a `*oauth2.Token` with `RefreshToken` set (request `access_type=offline` + `prompt=consent` on first authorization to guarantee a refresh token is issued). `oauth2.Config.TokenSource(ctx, token)` auto-refreshes on expiry — no manual refresh-token exchange code needed. |
| `github.com/zalando/go-keyring` | v0.2.8 (published Mar 2026) | Store the Drive OAuth refresh token in the OS secret store, not a plaintext file | Google's own guidance and OWASP-aligned OAuth best practice is unambiguous: refresh tokens for desktop apps belong in the OS's secure credential store, never a plain file (even with restrictive permissions) or app-local JSON. On Linux this is the freedesktop Secret Service (GNOME Keyring/KWallet via D-Bus) — **the same backend this repo's Signal plugin already unwraps a key from** (`plugins/signal/secretservice.go`), but that code is a hand-rolled, topos-internal D-Bus client not exposed to third parties. Since the Drive plugin is being built **out-of-repo** specifically to dogfood the external-plugin path, it cannot import internal topos packages — it needs a self-contained, standalone dependency. `zalando/go-keyring` is the de facto standard for this in the Go ecosystem: single API (`keyring.Set/Get/Delete`), cross-platform (Secret Service on Linux, Keychain on macOS, Credential Manager on Windows — relevant if the Drive plugin is ever run on a non-Linux desktop), actively maintained. This also validates that a third-party plugin author *can* get correct secret handling without needing anything topos-specific — good signal for the contract's real-world usability. |
| `debug/buildinfo` (Go stdlib) | Go 1.25 (already pinned) | Read embedded module path + VCS revision from an external plugin binary at load time, to compute the trusted/untrusted marking | See "Binary trust marking" discussion below — this is the pragmatic, zero-new-dependency mechanism for this milestone's "trusted = built from the `davison/topos` repo" requirement. |
| `vite-plugin-pwa` + `@vite-pwa/sveltekit` | `vite-plugin-pwa` v1.3.0, `@vite-pwa/sveltekit` v1.1.0 (npm, checked 2026-08-12; peer deps confirm Vite 8 support) | Generate the ServiceWorker + `manifest.webmanifest` as part of the existing `vite build` | `@vite-pwa/sveltekit` is the official SvelteKit integration wrapper around `vite-plugin-pwa` (itself the de facto standard PWA tooling for the Vite ecosystem, built on Google's `workbox-build`/`workbox-window`). It plugs into the *existing* `web/` build (SvelteKit 2.63 + `adapter-static` SPA mode, already on Vite 8) with no new build pipeline: `vite build` emits `sw.js` + `manifest.webmanifest` alongside the prerendered `index.html`/assets that `go:embed` already picks up wholesale in `cmd/topos`. No kernel-side code changes needed beyond MIME-type correctness (see integration note below). |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/fsnotify/fsnotify` | v1.10.1 (Go 1.23+ required — already satisfied) | Optional low-latency change detection for the filesystem plugin, **local mounts only** | Use as an *accelerator* layered on top of the scheduled poll (see below), never as the sole detection mechanism — see "Filesystem watching" discussion. Already the standard, actively-maintained cross-platform Go fsnotify wrapper (inotify/kqueue/ReadDirectoryChangesW); no reason to reach for anything else for the local-mount case. |
| `filepath.WalkDir` + `os.Stat` (Go stdlib) | Go 1.25 (already pinned) | Baseline directory scan for the filesystem plugin — the mechanism that actually has to work for network mounts | See "Filesystem watching" discussion below. No third-party dependency — this is the load-bearing mechanism, fsnotify is the optional enhancement. |
| `mime.AddExtensionType` (Go stdlib, called once at kernel startup) | Go 1.25 (already pinned) | Force-register `.webmanifest` → `application/manifest+json` and confirm `.js`/service-worker MIME types before the kernel's static file handler serves the embedded PWA assets | See "PWA" integration note below — Go's `mime` package reads `/etc/mime.types` on Linux, which frequently lacks a `.webmanifest` entry, so relying on `mime.TypeByExtension` unmodified risks serving the manifest with a wrong/empty Content-Type and failing Chrome's installability check silently. |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| None new | — | This milestone needs no new dev-tool additions — `make dev`/`air`, `golangci-lint`, the existing Playwright e2e harness, and `web/`'s existing `vite`/`svelte-check`/`vitest` toolchain all already cover the new surface area (PWA install flow becomes a new Playwright spec per this repo's own testing convention — see `docs/testing.md` — not a new tool). |

## Binary trust marking — recommendation and rationale

**Recommendation: `debug/buildinfo.ReadFile(path)` (stdlib, zero new dependency) — do NOT adopt `goreleaser` or `sigstore/cosign` for this milestone.**

This repo already ships its own hand-rolled release pipeline (`.github/workflows/release.yml`): plain `sha256sum` over the build outputs into a single `checksums.txt`, published as a GitHub Release asset via `gh release create`, with **no `goreleaser` and no signing** anywhere in the toolchain today. That's a deliberate, already-proven pragmatic choice for a single-maintainer project, and the milestone's own framing of "trusted" here is a **UX warning label, not a cryptographic security boundary** — the requirements doc explicitly defers "distribution, dev guide, and certification" to later. Introducing `goreleaser` (a new build-orchestration layer) and `cosign`/Sigstore keyless signing (a new CI OIDC dependency + verification step + transparency-log lookup) for a warning label would be materially more infrastructure than the feature calls for, mirroring this project's own existing "What NOT to Use" pattern (e.g., rejecting Extism/WASM sandboxing because the actual threat model didn't need it yet).

What to build instead, using only stdlib:
- Go automatically embeds VCS metadata (`vcs.revision`, `vcs.time`, `vcs.modified`) into any binary built in module mode from a git checkout (since Go 1.18) — no build flag needed beyond having git available at build time (already true for this repo's `make build`/CI).
- `debug/buildinfo.ReadFile(pluginPath)` reads that embedded info back out of **any** binary on disk, including one the kernel didn't build — exactly the operation needed when a user points the kernel at an external plugin binary.
- The kernel's trust check: `info.Main.Path == "github.com/davison/topos"` (or a specific in-repo plugin's known import path) **and** `vcs.modified == "false"` is a *provenance signal*, not proof — be explicit about this in the UI copy ("built from source claiming to be davison/topos, unmodified working tree") rather than implying a cryptographic guarantee, because a determined actor could hand-craft a binary with fabricated buildinfo strings. That honesty is consistent with the milestone explicitly deferring "certification."
- For the common, non-adversarial case this milestone actually targets — the maintainer's own release-published binaries plus a user's own local `make plugins` build — this is sufficient and reuses tooling the repo already has zero of today (no new external services, no key management for a single maintainer to lose).
- Revisit `cosign` keyless signing (GitHub Actions OIDC-based, no private key custody) specifically if/when a real third-party plugin distribution channel materializes and the trust boundary needs to become a genuine security control rather than a warning label — that's the same "cross this bridge later" posture the original stack research already applied to `sqlite-vec`.

## Filesystem watching — fsnotify vs polling for network mounts

**Recommendation: `filepath.WalkDir` + mtime/size stat-diff as the load-bearing sync mechanism; `fsnotify` as an opt-in, local-mount-only accelerator. Do not adopt `radovskyb/watcher`.**

Confirmed directly from kernel-level docs (LWN, `fsnotify` upstream discussion): **NFS and SMB/CIFS provide no OS-level change-notification hook on Linux** — `inotify` has no way to ask a network filesystem server to push notifications, so `fsnotify`-based watching on a network-mounted folder will silently miss changes (it doesn't error — it just never fires). Since this plugin's whole premise is "docs in a folder, optional subfolders" and the milestone context explicitly calls out network mounts as in scope, a watch-only design would be broken on exactly the mounts most likely to be used for a shared documents folder.

This repo's existing architecture already makes the right answer straightforward: every other source plugin is driven by the kernel's scheduled-sync + manual-refresh model (per-source health chips, staleness states, manual refresh — all shipped in Phase 2/6), not a push model. The filesystem plugin should match that: on each `Sync()` invocation (scheduled interval or manual refresh), do a full `filepath.WalkDir` over the configured root (respecting the optional-subfolders setting), stat each file, and diff mtime+size against the last-synced state already needed for incremental indexing. This works identically and correctly on local and network mounts, needs no new dependency, and fits the existing sync cadence other plugins already use.

Layer `fsnotify` on top **only** as a way to shrink the effective latency between scheduled polls for local paths (trigger an out-of-cycle `Sync()` on a local fsnotify event) — never as the only detection path, and gate it off (or silently no-op) when the configured root resolves to a non-local filesystem (detectable via `syscall.Statfs`'s filesystem-type magic number on Linux, or more simply: default watch mode to off and document that it's unsupported on NFS/SMB roots, since heuristically detecting every possible network filesystem type is unnecessary complexity for what's fundamentally a nice-to-have latency optimization).

Do not add `radovskyb/watcher` (a pure-polling fsnotify alternative) — it would be redundant with the stat-diff walk you already need for correctness, and the project has a large number of stale open issues/PRs with no clear recent maintenance signal, which doesn't clear this repo's own bar for taking on a dependency (contrast with e.g. `mutecomm/go-sqlcipher`, which the original stack research accepted low velocity for only because the on-disk format it wraps is itself stable — that reasoning doesn't transfer to a file-watching library).

## Per-item state storage — SQLite schema pattern for marks that survive index rebuilds

**Recommendation: a new `item_marks` table, keyed by the same stable `"{source}:{source_id}"` string `items.id` already uses, with no `REFERENCES`/`ON DELETE CASCADE` to `items`, and explicitly excluded from `rebuildOnSchemaChange`'s drop list. No new library.**

This is a schema-design decision, not a new-technology one — but it's the one piece of this milestone with a real footgun, grounded directly in this repo's own `kernel/index/schema.go` and `kernel/index/store.go`:

- `items.id` is already a **deterministic, source-derived natural key** (`"{source}:{source_id}"`), not a surrogate autoincrement id — so the same real-world item gets the same `items.id` again after any re-sync, including a full rebuild.
- `Store.Open`'s `rebuildOnSchemaChange` (triggered by a `schemaVersion` bump) does a hard `DROP TABLE` of `items_fts`, `webspace_items`, `webspaces`, `sync_runs`, and `items` — by an **explicit named list**, not "drop everything in the file." That's the exact mechanism to exploit correctly: a new table simply needs to never appear in that list.
- `webspace_items` (the closest existing precedent) deliberately uses `item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE` — correct for *that* table, because webspace membership genuinely should evaporate when an item is gone. `item_marks` must do the opposite: it's user-authored state that should **outlive** a schema-version rebuild of the derived index, per the milestone's own framing ("the kernel's first user-owned data beyond config").

Concretely: `CREATE TABLE IF NOT EXISTS item_marks (item_id TEXT PRIMARY KEY, state TEXT NOT NULL, marked_at INTEGER NOT NULL)` — no foreign key, indexed by the same natural key format as `items.id` — added to the idempotent `schema` block but **not** added to `rebuildOnSchemaChange`'s `DROP TABLE` list. Query-time, `LEFT JOIN item_marks ON item_marks.item_id = items.id` annotates stream rows; a mark for an item that hasn't been (re-)synced yet simply doesn't join to anything until the matching `items.id` reappears, which is the correct and desired behavior since `items.id` is stable across resyncs of the same underlying source item. This is the standard SQLite pattern for "durable annotation table decoupled from a regenerable content table" (same shape as e.g. a read-it-later app keying "starred" state off a stable URL rather than a re-crawled content-cache row id) — no ORM or migration library needed given the existing hand-written-SQL convention this repo already uses.

## PWA — ServiceWorker strategy and integration with the go:embed SPA

**Recommendation: `@vite-pwa/sveltekit` generating a Workbox-based ServiceWorker at build time; kernel-side changes limited to correct MIME-type serving. Flag the mobile-over-LAN installability gap explicitly rather than assume it's solved by adding the manifest.**

- `@vite-pwa/sveltekit` is a thin, official wrapper that hooks SvelteKit's `adapter-static` build output (already how this repo ships its SPA) and emits `sw.js`/`workbox-*.js`/`manifest.webmanifest` into the same static output directory `go:embed` already slurps wholesale — this milestone needs **no new embed path or Go build step**, just picking up the new files that land in the existing embedded directory. Use `generateSW` strategy (Workbox-managed precache manifest of the built SPA assets), not `injectManifest` (hand-written service worker) — this app has no offline-data-sync requirement (it's explicitly read-only/live-fetch-on-open per the hybrid data model), so Workbox's default precache-and-serve-app-shell strategy is sufficient; a custom service worker would be unjustified complexity.
- Required manifest fields for installability (Chrome/Android criteria, checked against MDN + Chrome DevTools docs as of 2026): `name`, `short_name`, `icons` including a 512×512 entry, `start_url`, `display: "standalone"`. Also include a 512×512 **maskable** icon (separate `purpose: "maskable"` entry, content kept inside the central 80% safe zone) to avoid the white-box/clipped look Android's adaptive-icon system otherwise produces — `@vite-pwa/assets-generator` (a `vite-plugin-pwa` peer tool) can derive the maskable variant from a single source SVG/PNG. For iOS Safari (which ignores the manifest's icon list and has no automatic install prompt — "Add to Home Screen" is manual), also add `<link rel="apple-touch-icon" href="...">` in the SPA's `app.html`.
- **Kernel-side integration gotcha**: Go's `mime` package on Linux resolves extensions via `/etc/mime.types`, which commonly has no `.webmanifest` entry — call `mime.AddExtensionType(".webmanifest", "application/manifest+json")` once at kernel startup (before the static file handler is wired) so the manifest isn't served with a missing/incorrect Content-Type, which silently fails Chrome's installability check with no obvious error surfaced to the user. Double-check `sw.js` is served as `application/javascript` and, since `go:embed` + the existing static handler already serve from the SPA's root path, no `Service-Worker-Allowed` header override is needed (the default scope is the directory the script is served from, which is already root here).
- **Open constraint, not solved by any library choice — flag for the roadmap**: this kernel binds to `127.0.0.1` by default and warns operators who reconfigure it to a LAN address (`isLoopback()` check in `cmd/topos/main.go`, a deliberate Phase 1 security default). Chrome/Android's installability check requires a **secure context**: HTTPS, or specifically `localhost`/`127.0.0.1`/`[::1]` — a plain LAN IP (e.g. `192.168.x.x`) reached from a phone does **not** qualify, even with a self-signed cert accepted in-browser, and mobile Safari has no PWA install prompt at all regardless of origin. So "installable on desktop and mobile" as literally stated has two different stories: **desktop install over `localhost` works out of the box** once the manifest/SW ship; **mobile install over LAN requires the user to front the kernel with their own HTTPS reverse proxy or tunnel** (e.g., Caddy with a local CA, or Tailscale Serve) — that's infrastructure outside topos itself, consistent with this project's existing "no cloud/SaaS, LAN-reachable services the user already runs" posture, not something to build into the kernel. Recommend the roadmap phase for PWA explicitly scope "desktop install" as the guaranteed deliverable and treat "mobile install" as conditional on the user's own network setup, documented rather than engineered around.

## Installation

```bash
# Go (Drive plugin — built out-of-repo against the published contract,
# so these are new dependencies of THAT module, not the kernel's go.mod)
go get google.golang.org/api/drive/v3@v0.292.0
go get golang.org/x/oauth2@v0.36.0
go get golang.org/x/oauth2/google
go get github.com/zalando/go-keyring@v0.2.8

# Go (filesystem plugin — in-repo, add fsnotify only if the local-mount
# accelerator is built in this milestone rather than deferred)
go get github.com/fsnotify/fsnotify@v1.10.1

# Frontend (web/) — PWA build tooling
npm install -D vite-plugin-pwa@1.3.0 @vite-pwa/sveltekit@1.1.0
```

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|--------------|-------------|--------------------------|
| `debug/buildinfo` (stdlib) for trust marking | `goreleaser` + `sigstore/cosign` keyless signing | Once a real third-party plugin distribution/certification story exists and "trusted" needs to be a genuine cryptographic guarantee rather than a warning label — explicitly deferred by this milestone's own scope ("certification deferred"). |
| `filepath.WalkDir` stat-diff (load-bearing) + `fsnotify` (local accelerator) | `fsnotify` alone as the sole detection mechanism | Never, for this project — the filesystem plugin's stated scope includes network mounts, where `fsnotify` silently doesn't fire at all (confirmed kernel-level Linux limitation, not a library bug). |
| `filepath.WalkDir` stat-diff + `fsnotify` | `radovskyb/watcher` (pure-polling library) | If you specifically want a single unified watch API instead of hand-rolled polling *and* are comfortable with its unclear current maintenance signal — the stat-diff you need for network-mount correctness already gives you polling for free, so this adds a dependency without adding capability here. |
| `zalando/go-keyring` for the Drive plugin's token storage | This repo's own hand-rolled `plugins/signal/secretservice.go` D-Bus client | Only if the Drive plugin were being built in-repo (it deliberately isn't — dogfooding the external path means it can't depend on internal topos packages). |
| `@vite-pwa/sveltekit` (`generateSW` strategy) | Hand-rolled `sw.js` (`injectManifest` strategy or fully custom) | If the app later needs genuine offline data access (e.g., serving cached webspace content with no kernel reachable) — out of scope today given the hybrid data model's live-fetch-on-open design; a custom service worker would fight the "read-only, always-fetch-from-plugin" architecture rather than serve it. |

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|--------------|
| `goreleaser` / `sigstore/cosign` for this milestone's trust marking | Materially more CI/signing infrastructure than a UX warning label needs; this repo's own release pipeline is already a deliberately minimal hand-rolled `sha256sum` + `gh release create`, with no signing today | `debug/buildinfo.ReadFile` (stdlib) reading module path + VCS revision/modified flag |
| Relying on `fsnotify` alone for the filesystem plugin | Silently non-functional on NFS/SMB/CIFS mounts — no error, just missed changes, which is worse than an obvious failure | Scheduled `filepath.WalkDir` stat-diff as the load-bearing mechanism (matches this repo's existing scheduled-sync/manual-refresh model for every other source), `fsnotify` only as a local-mount latency accelerator |
| `radovskyb/watcher` | Redundant given the stat-diff walk already required for network-mount correctness; unclear current maintenance (many long-open issues/PRs, no confirmed recent release activity) | `filepath.WalkDir` + `os.Stat` (stdlib) |
| A `REFERENCES items(id) ON DELETE CASCADE` foreign key on the new per-item marks table | Would tie user-owned data to the same drop/recreate rebuild path `items` already goes through on a schema-version bump (`rebuildOnSchemaChange` in `kernel/index/store.go`), destroying the exact data this milestone calls out as needing to survive rebuilds | A standalone `item_marks` table with no FK, keyed by the same natural `items.id` string format, explicitly excluded from the rebuild's `DROP TABLE` list |
| Assuming the manifest + ServiceWorker alone make the app "installable on mobile" | Chrome/Android's secure-context check does not extend to LAN IPs the way it does to `localhost`/`127.0.0.1`; the kernel deliberately binds to loopback by default (`isLoopback()` in `cmd/topos/main.go`) and warns on LAN exposure — mobile install over LAN needs the user's own HTTPS front-end, which is out of scope for the kernel itself | Ship the PWA assets and treat desktop install (over `localhost`) as the guaranteed deliverable; document mobile install as conditional on the user fronting topos with their own HTTPS reverse proxy/tunnel |
| Plain-file (even permission-restricted) storage of the Drive OAuth refresh token | Google's own OAuth best-practice guidance and general OWASP-aligned desktop-app guidance are explicit that refresh tokens belong in OS-native secure storage, not app-local files, regardless of file permissions | `zalando/go-keyring` (Secret Service on Linux — the same backend class this repo already trusts for Signal's key) |
| The deprecated OOB (`urn:ietf:wg:oauth:2.0:oob`) "copy-paste the code" installed-app OAuth flow | Google removed OOB support in 2022; any tutorial still showing it is stale and will fail against current Google OAuth endpoints | The loopback-IP-redirect flow (`http://127.0.0.1:<ephemeral-port>/...`), Google's current supported pattern for installed/desktop apps |

## Version Compatibility

| Package A | Compatible With | Notes |
|-----------|------------------|-------|
| `google.golang.org/api@v0.292.0` | `golang.org/x/oauth2@v0.36.0`, Go 1.25 (this repo's pinned toolchain) | `option.WithTokenSource(oauth2Config.TokenSource(ctx, token))` is the glue between the two — no version coupling beyond both being reasonably current. |
| `fsnotify@v1.10.1` | Go 1.23+ (this repo is on 1.25 — satisfied) | Pure enhancement layer per the recommendation above; no hard coupling to anything else in the stack. |
| `vite-plugin-pwa@1.3.0` / `@vite-pwa/sveltekit@1.1.0` | Vite `^3.1.0 \|\| ^4.0.0 \|\| ^5.0.0 \|\| ^6.0.0 \|\| ^7.0.0 \|\| ^8.0.0` (peer dep, confirmed via `npm view`), SvelteKit 2.x, `@sveltejs/adapter-static` | This repo is already on Vite 8 (`web/package.json`) and SvelteKit 2.63 — both within the plugin's supported peer range; confirmed no upgrade needed elsewhere in `web/` to adopt PWA tooling. |
| `zalando/go-keyring@v0.2.8` | Linux: requires a running Secret Service provider (GNOME Keyring or KWallet via its Secret Service shim) on the session D-Bus | Same runtime dependency class this repo already accepted for the Signal plugin (Phase 4) — if no keyring daemon is running, both fail the same way; no new operational risk class introduced. |

## Sources

- pkg.go.dev: `google.golang.org/api` (v0.292.0, published Aug 4 2026) — HIGH
- pkg.go.dev: `golang.org/x/oauth2` (v0.36.0, published Feb 11 2026) — HIGH
- pkg.go.dev: `github.com/fsnotify/fsnotify` (v1.10.1, published May 4 2026, requires Go 1.23+) — HIGH
- pkg.go.dev: `github.com/zalando/go-keyring` (v0.2.8, published Mar 23 2026) — HIGH
- npm registry (`npm view`): `vite-plugin-pwa@1.3.0`, `@vite-pwa/sveltekit@1.1.0`, peer-dependency range confirming Vite 8 support — HIGH
- LWN.net, "Change notifications for network filesystems" + `fsnotify`/kernel mailing-list discussion of NFS/SMB/CIFS notification gaps — HIGH (kernel-level technical limitation, cross-confirmed across multiple independent sources)
- Google for Developers, OAuth 2.0 best practices + installed-app flow documentation (loopback redirect, OOB deprecation) — HIGH (official)
- MDN, "Making PWAs installable" + Chrome for Developers, Lighthouse installable-manifest criteria — HIGH (official/vendor docs)
- This repository's own code as primary source: `kernel/index/schema.go`, `kernel/index/store.go` (schema-rebuild mechanics), `cmd/topos/main.go` (loopback-binding default), `.github/workflows/release.yml` (existing hand-rolled checksums-only release pipeline), `plugins/signal/secretservice.go` (existing Secret Service D-Bus precedent) — HIGH (ground truth)
- `goreleaser`/`cosign` documentation and community sign-binaries guides (GoReleaser docs, carlosbecker.com) — MEDIUM, used to characterize the alternative that was deliberately **not** recommended, not as a basis for the chosen approach
- GitHub: `radovskyb/watcher` — LOW confidence on current maintenance status (unable to confirm recent release/commit activity); factored into the explicit non-recommendation above

---
*Stack research for: topos v1.1.0 "Plugin Ecosystem" milestone*
*Researched: 2026-08-12*
