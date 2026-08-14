# Phase 13: Per-Item Curation & Installable App - Research

**Researched:** 2026-08-14
**Domain:** Local SQLite curation-mark storage/filtering, SvelteKit PWA installability (adapter-static + kernel-embedded serving), and Go build-time trust-provenance manifests
**Confidence:** HIGH

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Exclude interaction**
- **D-01:** The stream gets desktop-standard **multi-select** — ctrl-click toggles, shift-click extends a range — while plain click keeps opening the detail pane. A **floating action bar** appears while the selection is non-empty ("N selected — Exclude · Clear"); Esc or Clear empties it. The detail pane additionally carries a single-item exclude button for the currently open item.
- **D-02:** Exclusion (single or bulk) is **instant with an undo toast** — no confirm dialogs ever; the excluded-items view is the durable reversal path.
- **D-03:** Exclusion removes the item from **every webspace surface**: stream, in-webspace FTS search results, date-marker ruler, and any counts. One consistent rule — exclusion is the final filter tier applied wherever webspace items are read.
- **D-04:** Chat conversation-day digests exclude **per digest row** (that conversation, that day). "This conversation never belongs here" is match-rules territory, not per-item curation — no conversation-wide exclusion mechanism.

**Excluded-items view**
- **D-05:** The view is a **stream filter toggle**: a chip/toggle in the webspace chrome (e.g. "Excluded (7)") flips the stream itself to show the excluded bucket, reusing StreamRow, the detail pane, and the multi-select machinery. No modal, no separate route.
- **D-06:** The toggle appears **only when the webspace's excluded count > 0** (count shown). A webspace with no exclusions looks exactly like today.
- **D-07:** The excluded view orders **chronologically like the stream** (same date markers) — it is literally the same stream showing the other bucket.
- **D-08:** Un-exclude is the **exact mirror of exclude**: multi-select + action bar offering "Include", single-item include button in the detail pane, instant with undo toast.

**Orphaned exclusions**
- **D-09:** **Silent auto-prune**: when a **healthy** sync no longer reports an excluded item, the mark is swept with the index row. The excluded view only ever shows items that would otherwise be in the stream. A reappearing item (e.g. restored file) re-enters the stream unexcluded.
- **D-10:** Prune fires **only on a healthy sync that omits the item** — a failed sync (unreachable mount, plugin error) must never count as "item gone", and an **index rebuild must never prune marks** (rebuild wipes item rows before re-sync completes; marks must survive per KERN-09). This is the never-silently-empty precedent applied to marks.
- **D-11:** Rename/move losing an exclusion (new stable ID under Phase 12's remove+add identity model) is **accepted** as the honest consequence — no rename-tracking for marks.

**Trust-tier hardening (folded todo)**
- **D-12:** Trusted-tier status derives from a **build-provenance manifest**: a set of first-party plugin identities/hashes embedded into the kernel at link time (kernel and plugins are rebuilt together by `make build`/`make dev`, so dev rebuilds stay false-alarm-free). A trusted-dir binary that doesn't verify against the manifest gets **no trusted-tier treatment regardless of directory** — closing the config-edit, file-drop, and shadowing bypass paths. — **Reversibility:** costly — the manifest becomes the trust authority the UI, docs, and release process describe; retiring it means re-deriving trust from directory location again and re-opening the three bypass paths.
- **D-13:** **Hash-verification failure refuses to load, loudly** (supersedes the demote-and-run idea discussed en route; matches the operator's TODO.md intent): the binary does not launch; the affected source chip shows a named error state (log AND UI) explaining why; the explicit consent + pin flow (external-tier add / re-accept) is the only path to running it. Unverified code never executes. This harmonizes with Phase 11's shipped external-tier behavior, where a pin mismatch already fails the launch per-instance with a named state and a two-click re-pin.
- **D-14:** The **D-11 shadowing event** (same-named trusted-dir binary shadowing a pinned external plugin) is surfaced in the **UI** (health state / advisory on affected sources), not just a kernel log line.
- **D-15:** The **locally-built Signal plugin** (release artifacts exclude it; `make signal` builds it locally, so a release kernel's manifest can never contain its hash) is handled **honestly via consent + pin**: it refuses to load until the user explicitly accepts it through the external-tier consent flow, then runs with a pinned hash and the untrusted badge. **The plugin documentation must clearly explain why** (cgo local build ≠ release manifest). Source-builders running `make build` (kernel + plugins together) are unaffected.
- **D-16:** The **two plugin directories stay** this phase. The manifest is the trust *authority*; directories remain the shipped install conveniences. Collapsing to a single directory waits for the distribution work (PLUG-10), where a managed plugin directory becomes the natural shape.

### Claude's Discretion
- **PWA mechanics end to end**: tooling choice (vite-plugin-pwa / @vite-pwa/sveltekit vs hand-rolled SW), manifest contents, icon set derivation from the existing `web/static/app-icon.png`, ServiceWorker update strategy satisfying the never-stale criterion (the operator deliberately left the update UX — silent reload vs prompt — to Claude), kernel MIME-type/serving adjustments, and what the installed window shows when the kernel isn't running.
- Marks storage mechanics: where the marks live (separate SQLite file vs rebuild-exempt table in the index DB), schema, and the API shape for exclude/include endpoints — constrained only by D-09/D-10 survival semantics and "not config TOML" (Phase 11 D-01 framing: marks are the kernel's first user-owned data beyond config).
- Undo-toast duration/stacking, selection keyboard shortcuts beyond ctrl/shift-click, action-bar copy.
- Manifest generation mechanics (how `make build` embeds the hash set, hash algorithm, manifest format) and the exact name/copy of the refuse-to-load error state.
- Whether the excluded-count toggle lives with the filter chips or elsewhere in the webspace chrome.

### Deferred Ideas (OUT OF SCOPE)
- **Plugin distribution** (TODO.md §Plugin Trust System): fork-based development with a `publish-plugin` GHA building binary + hash artifact, pull-by-repo-URL install with hash-verified fetch, single managed plugin directory, and PR-merge trust promotion — defers to backlog Phase 999.1 (PLUG-10/11). The backlog entry should absorb TODO.md's notes; Phase 13's manifest + refuse-to-load verification is the foundation it builds on.
- **Single plugin directory** (collapse of the Phase 11 two-tier layout) — revisit alongside PLUG-10 distribution (D-16).
- **Conversation-wide exclusion** for chat digests — match-rules/config territory if ever wanted; explicitly not a mark type (D-04).
- **Plugin picker search/filter box** (TODO.md vNext) — noted, stays vNext.
- Reviewed but not folded: "Signal schema-version verify-and-accept tooling" (2026-08-05) — unrelated to curation, PWA, or trust hardening; stays pending for a phase that touches the Signal plugin.
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| KERN-09 | User can exclude an individual stream item from a webspace; the exclusion survives re-syncs and index rebuilds and always outranks automatic match rules | Architecture Pattern 1 (rebuild-exempt, no-FK `item_marks` table) + Pattern 2 (prune-sweep hooked into `ReplaceWebspaceSourceItems`'s existing transaction, which already only fires on healthy per-(webspace,source) syncs) + Pitfall 1 (no schemaVersion bump needed/wanted for the new table) |
| KERN-10 | User can view a webspace's excluded items and un-exclude them (mark reversal; pulling never-matched items in via a Browse RPC is explicitly out of scope) | Architecture Pattern 3 (shared mark-aware SQL filter, `IN`/`NOT IN` toggle for the two views) + Architectural Responsibility Map row "Excluded-items view toggle" (query-param toggle on the existing stream route, per D-05) |
| UI-13 | App installs as a PWA on desktop (manifest, ServiceWorker, icons) with an update flow that never serves a stale UI against an upgraded kernel | Standard Stack (`vite-plugin-pwa`/`@vite-pwa/sveltekit`/`@vite-pwa/assets-generator`, all version/compatibility-verified against this repo's pinned Vite 8/SvelteKit 2) + Pitfall 3 (adapter-static fallback-revisioning caveat) + Pitfall 4 (`.webmanifest` MIME type) + Pitfall 5 (don't runtime-cache `/api/*`) |
| UI-14 | Mobile/LAN install limitation (secure-context requirement) is documented, with the recommended user-provided HTTPS workarounds | Code Examples "Secure-context behavior" (MDN-cited confirmation that localhost/127.0.0.1 is a secure context but LAN-IP-over-HTTP is not) + cross-reference to the kernel's existing `isLoopback()` warning (`cmd/topos/main.go`) as the network-layer counterpart of this browser-layer limitation |
</phase_requirements>

## Summary

Phase 13 is three independently-shippable strands riding the same existing architecture, not three new subsystems. Per-item curation (KERN-09/KERN-10) adds one new SQLite table read/write pattern that must sit *outside* the index's existing schema-rebuild blast radius (`kernel/index/schema.go` / `store.go`) and hook into the *exact* point in the sync pipeline that already, for free, satisfies "healthy sync only" (`kernel/correlate/correlate.go`'s `SyncSource`, which never calls `ReplaceWebspaceSourceItems` for a (webspace, source) pair unless that pair's Match call actually succeeded). PWA installability (UI-13/UI-14) is a `web/`-only addition (`vite-plugin-pwa` + `@vite-pwa/sveltekit`, both compatible with the pinned Vite 8 / SvelteKit 2 versions) plus one or two small kernel serving fixes (MIME type for `.webmanifest`); the desktop install path already satisfies the secure-context requirement today because the kernel's default `127.0.0.1` listener is a browser-recognized secure context, and the LAN/mobile limitation UI-14 asks to document is the *browser-side* mirror of a network-layer warning the kernel already logs (`cmd/topos/main.go`'s `isLoopback` check). Trust-tier hardening (the folded D-12–D-16 todo) is the strand with the sharpest concrete gotcha: **the Makefile currently builds the kernel binary *before* the plugin binaries** in both `build` and `build-portable` (the actual release recipe), which is backwards for a link-time-embedded plugin-hash manifest — this ordering must be fixed as a prerequisite, not a detail, of implementing D-12.

**Primary recommendation:** Implement all three strands by extending existing, already-load-bearing patterns rather than inventing new ones — a rebuild-exempt `item_marks`-style table hooked into `ReplaceWebspaceSourceItems`'s existing transaction for curation; `vite-plugin-pwa`'s `generateSW` strategy with `registerType: 'autoUpdate'` scoped to the app shell only (never `/api/*`) for the PWA; and a generated-Go-source plugin-hash manifest (reusing `pluginhost.HashBinary` and the existing `LaunchFailurePinMismatch`-style soft-fail pattern) built into a *reordered* Makefile for trust hardening.

## Project Constraints (from CLAUDE.md)

Directives from `.claude/CLAUDE.md` that apply to this phase's implementation, extracted so the planner can verify compliance:

- **Read-only by design (v1.0 constraint, unchanged):** "Write-back to any source" is out of scope project-wide. Marks are kernel-owned data, not a write to a source — the new exclude/include endpoints must write only to `kernel/index`'s own SQLite file and must never route through a plugin's `Fetch`/`Match`/`Health` RPCs (the contract has no write RPC to begin with, per `docs/plugin-contract.md`).
- **All data stays local; no personal content leaves the user's machine:** PWA icons/manifest/ServiceWorker must be generated at build time and served by the kernel's own embedded static file handler (`kernel/webui/embed.go` + `spaHandler`) — no third-party CDN, font host, or external asset reference in the generated manifest.
- **Never store full source content in the local index** (explicit "What NOT to Use" entry): the new `item_marks` table must store only the mark itself (item id, webspace, kind, timestamp) — never a copy of the item's title/preview/content, which already lives in `items`.
- **Deployment: runs on the user's desktop machine, required for local access to Signal/WhatsApp desktop databases:** unchanged by this phase; reinforces why the PWA's desktop-install path is the primary target and LAN/mobile is explicitly a documented limitation rather than a feature to build around (UI-14).
- **Extensibility: plugin contract must be stable and documented enough for third-party source plugins:** the trust-tier hardening strand (D-12–D-16) changes *how* trust is derived but must not change the `SourcePlugin` gRPC contract itself (`Describe`/`Match`/`Fetch`/`Health`) — this is purely a kernel-side launch-gate change, not a contract change.
- **Testing convention:** "Any phase that touches the UI extends the Playwright e2e suite as part of its definition of done; any UAT item a browser can drive becomes a spec (`web/e2e/specs/`) rather than staying a manual check." This phase's UI-heavy criteria (multi-select + undo, excluded view, refuse-to-load error states, and whatever PWA manifest/SW presence is browser-drivable) must land as new specs following the existing `13-*.spec.ts` naming convention (see `web/e2e/specs/` for the `NN-description.spec.ts` pattern already in use, e.g. `11-binary-changed-repin.spec.ts`, `12-filesystem-add-source.spec.ts`), not as manual-only checks.
- **Git/testing preferences (global, `~/.claude/CLAUDE.md`):** TDD preferred but not dogmatic — tests should prove requirements and guard regressions, not chase coverage; linear history with rebase merging, no unnecessary merge commits.

## Architectural Responsibility Map

topos is a single-binary desktop deployment (Go kernel + embedded SvelteKit SPA, no SSR tier, no CDN) — "Frontend Server (SSR)" and "CDN/Static" from the standard tier vocabulary collapse into the kernel's own embedded-static-file serving (`kernel/webui/embed.go` + `kernel/httpapi/routes.go`'s `spaHandler`). The table below uses the closest-fit tier for each capability and annotates where the standard vocabulary doesn't map 1:1 onto this architecture.

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Exclude/include a mark (write) | API / Backend (`kernel/index.Store`, new HTTP routes) | — | Marks are kernel-owned user data outside config (Phase 11 D-01 framing, reaffirmed by CONTEXT.md); the write must be transactional against the same index the sync engine writes to |
| Mark-aware stream/search filtering (read) | API / Backend (`kernel/index.Store.StreamItems`/`Search`, `kernel/httpapi/stream.go`+`search.go`) | Database/Storage (SQL join) | D-03 requires exclusion applied at every read surface (stream, FTS search, agent mirror) — a single shared filter point in the SQL, not a client-side post-filter, is the only way this can never disagree with itself |
| Orphan-prune sweep | API / Backend (`kernel/index.Store.ReplaceWebspaceSourceItems`) | — | Must run inside the exact same transaction as the sync write it is scoped to, at (webspace, source) granularity — see Architecture Patterns below |
| Multi-select, action bar, undo toast | Browser / Client (Svelte components) | — | Pure client-side interaction state; no new server concept beyond the exclude/include calls it triggers |
| Excluded-items view toggle | Browser / Client (stream view-mode state) | API / Backend (`?view=` query param on existing stream route) | D-05 explicitly reuses the existing stream surface — this is a query-time toggle, not a new page/route |
| PWA manifest, icons, ServiceWorker | Browser / Client (build-time Vite plugin output) | Kernel static serving (embed.go/spaHandler) | Generated entirely at `web/` build time; the kernel's only job is serving the generated files with correct MIME types and no defeating cache headers |
| SW update / never-stale enforcement | Browser / Client (ServiceWorker + Workbox precache) | — | Enforced entirely client-side once a new SW activates; no kernel version-check endpoint is required for `autoUpdate` |
| Trust manifest generation | Build tooling (Makefile + a generator step) | — | Must run after plugin binaries exist, before the kernel compiles — a build-time, not run-time, concern |
| Trust manifest verification (launch gate) | API / Backend (`kernel/pluginhost.ResolveBinary`) | — | Extends the existing launch-time provenance check; never re-derived from anything the plugin reports (unchanged discipline) |
| Refuse-to-load / shadowing UI states | Browser / Client (SourceChip/TrustBadge) | API / Backend (`kernel/httpapi/sources.go` LaunchFailure vocabulary) | Follows the existing `pin_mismatch` named-state pattern exactly |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `vite-plugin-pwa` | 1.3.0 [CITED: npm registry, `npm view vite-plugin-pwa version`/`peerDependencies`, checked this session] | Generates the manifest + ServiceWorker via Workbox, driven by Vite's own build | De facto standard Vite PWA tooling; peerDependencies (`vite: "^3.1.0 \|\| ... \|\| ^8.0.0"`) explicitly covers this repo's pinned `vite@^8.0.16` |
| `@vite-pwa/sveltekit` | 1.1.0 [CITED: npm registry, checked this session] | SvelteKit-specific wiring (adapter detection, `virtual:pwa-register` helpers) on top of `vite-plugin-pwa` | peerDependencies (`@sveltejs/kit: "^1.3.1 \|\| ^2.0.1"`) covers this repo's pinned `@sveltejs/kit@^2.63.0` |
| `workbox-build` / `workbox-window` | 7.4.1 [CITED: npm registry, checked this session] | Transitive Workbox engine `vite-plugin-pwa` drives | Pulled in automatically as a `vite-plugin-pwa` peer dependency; not installed directly unless a custom `injectManifest` strategy is chosen |
| `@vite-pwa/assets-generator` | latest [CITED: npm registry, checked this session; package-legitimacy verdict OK] | Derives the required icon set (192/512/maskable) from one source image | `web/static/app-icon.png` is 1024×1024 RGBA [VERIFIED: read via PIL this session] — ample resolution; this tool is the ecosystem-standard companion to `vite-plugin-pwa` for icon generation rather than hand-rolling ImageMagick/sharp scripts |

**Package Legitimacy Audit** (see dedicated section below) — all four verdicts `OK`.

**Installation:**
```bash
npm --prefix web install -D vite-plugin-pwa @vite-pwa/sveltekit @vite-pwa/assets-generator
```

**Version verification:** `npm view vite-plugin-pwa version`, `npm view vite-plugin-pwa peerDependencies`, `npm view @vite-pwa/sveltekit peerDependencies` were run this session against the live npm registry — see Package Legitimacy Audit table for the exact returned signals (weekly downloads, publish dates, repo URLs).

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| Go stdlib `crypto/sha256` (already used) | — | Plugin binary hashing for the manifest | `kernel/pluginhost/binaryhash.go`'s `HashBinary` already implements this exact pattern [VERIFIED: read this session] — reuse it verbatim for manifest generation rather than adding a new hashing helper |
| Go stdlib `mime` | — | Registering `.webmanifest` → `application/manifest+json` if the OS mime database lacks it | `net/http.FileServer` (used by `spaHandler`) infers content-type via `mime.TypeByExtension`; `.webmanifest` is not universally registered in every OS's mime.types |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `vite-plugin-pwa` `generateSW` strategy | `injectManifest` (hand-written service worker) | `generateSW` covers this phase's entire scope (precache app shell, exclude `/api/*`, autoUpdate) with declarative config; `injectManifest` is only worth the extra hand-written SW code if a future phase needs custom runtime caching logic beyond what's in scope here (explicitly out of scope: "Full offline PWA caching") |
| Rebuild-exempt table in the same index.db | A second, separate SQLite file for marks | Same-file is simpler (one `Store`, one transaction spans sync-write + prune-sweep atomically) and is sufficient because "index rebuild" in this codebase concretely and exclusively means `rebuildOnSchemaChange`'s schema-version-triggered drop/recreate [VERIFIED: `kernel/index/store.go` read this session — no other rebuild mechanism exists in the codebase]; a separate file would only be worth the added Store/connection/lifecycle wiring if the durability bar were "survives an operator manually deleting index.db", which nothing in KERN-09/KERN-10 or CONTEXT.md asks for |
| Generated Go source file for the trust manifest | `go:embed` of a JSON/text manifest file | A generated `.go` file (a `map[string]string` const/var) needs no `embed.FS` plumbing and is simpler to wire into `pluginhost.ResolveBinary`'s existing Go code; `go:embed` would only earn its complexity if the manifest needed to be read/parsed at multiple binary-boundary points, which it doesn't here |

## Package Legitimacy Audit

| Package | Registry | Age (last publish) | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| `vite-plugin-pwa` | npm | published 2026-05-05 | ~4.24M/wk | github.com/vite-pwa/vite-plugin-pwa | OK | Approved |
| `@vite-pwa/sveltekit` | npm | published 2025-11-27 | ~141K/wk | github.com/vite-pwa/sveltekit | OK | Approved |
| `workbox-build` | npm | published 2026-05-04 | ~8.84M/wk | github.com/googlechrome/workbox | OK | Approved (transitive) |
| `workbox-window` | npm | published 2026-05-04 | ~8.83M/wk | github.com/googlechrome/workbox | OK | Approved (transitive) |
| `@vite-pwa/assets-generator` | npm | published 2025-10-14 | ~291K/wk | github.com/vite-pwa/assets-generator | OK | Approved |

**Packages removed due to [SLOP] verdict:** none.
**Packages flagged as suspicious [SUS]:** none.

All five packages returned `OK` from `gsd-tools query package-legitimacy check --ecosystem npm` this session (no deprecation flag, no postinstall script, well-known GitHub org repos, high weekly download counts). Package *names* originated from this project's own prior v1.1.0 research (`.planning/research/SUMMARY.md`), not from this session's training-data recall alone, but per the provenance rule they are tagged `[CITED: npm registry]` rather than `[VERIFIED]` — the registry check confirms existence/currency, not authoritative-source publication; no official docs page or Context7 lookup was performed this session to promote them to `[VERIFIED]`.

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────── Browser (installed PWA or tab) ───────────────────────────┐
│                                                                                         │
│  ServiceWorker (Workbox, autoUpdate)                                                   │
│    - precaches app shell (200.html, JS/CSS bundle, icons) — NEVER /api/*               │
│    - on new kernel build → new SW → activates → reload → fresh UI (never stale)        │
│                                                                                         │
│  Svelte SPA                                                                            │
│    StreamList (multi-select, action bar) ──selects──▶ POST /api/webspaces/{ws}/marks/* │
│    DetailPane (single-item exclude/include button)                                     │
│    Excluded-view toggle ──flips view param──▶ GET /api/webspaces/{ws}/stream?view=...  │
└───────────────────────────────────┬─────────────────────────────────────────────────┘
                                     │ HTTP (loopback 127.0.0.1:7777 by default)
┌────────────────────────────────────▼──────────────────────── Go kernel ───────────────┐
│                                                                                         │
│  kernel/httpapi                                                                        │
│    StreamHandler / SearchHandler ──reads──▶ index.Store (mark-aware SQL join)          │
│    new marks routes (exclude/include, bulk) ──writes──▶ index.Store                    │
│    spaHandler ──serves──▶ webui.FS() (embedded SPA build incl. manifest.webmanifest,   │
│                             sw.js, icons)                                              │
│                                                                                         │
│  kernel/correlate.Engine.SyncSource (per webspace × source, healthy pairs only)         │
│    ──▶ index.Store.ReplaceWebspaceSourceItems (existing tx)                            │
│          ├─ upsert items, replace webspace_items (existing)                            │
│          └─ NEW: prune item_marks rows whose item left this (webspace, source) pair    │
│                  (same transaction — never a separate sweep pass)                       │
│                                                                                         │
│  kernel/pluginhost.ResolveBinary (launch gate)                                          │
│    trusted-dir binary ──▶ re-hash (HashBinary) ──▶ compare against generated manifest   │
│                             match → TierTrusted   mismatch/absent → D-13 refuse-to-load │
│    external-dir binary ──▶ existing pin-verify path (unchanged)                         │
│                                                                                         │
│  kernel/index.Store (SQLite, WAL)                                                       │
│    items / webspace_items / webspaces / sync_runs (existing, rebuild-droppable)         │
│    item_marks (NEW — same file, absent from rebuildOnSchemaChange's DROP list)          │
└─────────────────────────────────────────────────────────────────────────────────────┘

┌──────────────────────────── Build time (Makefile) ────────────────────────────────────┐
│  plugins / plugins-portable  (builds bin/plugins/topos-plugin-*)                       │
│         │                                                                              │
│         ▼                                                                              │
│  NEW: generate-manifest  (hashes each binary, writes generated .go source)             │
│         │                                                                              │
│         ▼                                                                              │
│  go build -o bin/topos ./cmd/topos   (kernel now compiles the manifest IN)             │
│                                                                                         │
│  ⚠ current `build`/`build-portable` targets run this in the OPPOSITE order today       │
│    (kernel builds BEFORE plugins) — must be reordered as part of this phase            │
└─────────────────────────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure

No new top-level directories — this phase extends existing packages:

```
kernel/index/
├── schema.go            # add item_marks CREATE TABLE IF NOT EXISTS (no version bump — see Pitfall 1)
├── store.go              # add mark read/write methods + prune-sweep inside ReplaceWebspaceSourceItems
kernel/httpapi/
├── stream.go              # add ?view=excluded handling, mark-aware filtering
├── search.go              # same mark-aware filtering for FTS search
├── marks.go               # NEW — exclude/include route handlers (single + bulk)
kernel/pluginhost/
├── binaryhash.go          # reuse HashBinary as-is
├── manifest.go             # NEW — reads the generated manifest map, exposes a verify(name, path) helper
├── manifest_generated.go   # NEW, git-ignored, regenerated every build — map[string]string binary→hash
├── discover_binaries.go   # ResolveBinary's trusted-tier branch calls the new verify helper
├── host.go                 # add a new LaunchFailure Reason constant beside LaunchFailurePinMismatch
web/src/lib/components/
├── StreamList.svelte       # multi-select, action bar wiring
├── StreamRow.svelte        # ctrl/shift-click selection handling
├── DetailPane.svelte       # single-item exclude/include button
├── WebspaceHeader.svelte   # excluded-count toggle (candidate location — chip row already lives here)
web/
├── vite.config.ts          # add VitePWA()/SvelteKitPWA() plugin
├── svelte.config.js        # unchanged (adapter-static already configured — see Pitfall 3)
Makefile                   # reorder build/build-portable; add generate-manifest step
```

### Pattern 1: Rebuild-exempt, no-FK mark table hooked into the existing sync transaction

**What:** A new table (e.g. `item_marks(webspace_name TEXT, item_id TEXT, kind TEXT, created_unix INTEGER, PRIMARY KEY(webspace_name, item_id))`) added to `schema.go`'s `CREATE TABLE IF NOT EXISTS` block, deliberately **not** added to `rebuildOnSchemaChange`'s manually-enumerated `DROP TABLE` list [VERIFIED: `kernel/index/store.go:119-127`, exact list quoted below], and carrying **no foreign key** to `items(id)`.

**When to use:** This is the only mechanism in this phase that must satisfy three simultaneous survival guarantees (resync, kernel restart, index rebuild) plus one ordering guarantee (always outranks match rules) — the no-FK, rebuild-exempt shape is what makes all four true without special-casing any of them.

**Example (verified against the real file):**
```go
// kernel/index/store.go — the exact enumerated drop list this session read.
// item_marks must NOT be added here.
if itemsTableExists != 0 {
    for _, stmt := range []string{
        `DROP TABLE IF EXISTS items_fts`,
        `DROP TRIGGER IF EXISTS items_ai`,
        `DROP TRIGGER IF EXISTS items_ad`,
        `DROP TRIGGER IF EXISTS items_au`,
        `DROP TABLE IF EXISTS webspace_items`,
        `DROP TABLE IF EXISTS webspaces`,
        `DROP TABLE IF EXISTS sync_runs`,
        `DROP TABLE IF EXISTS items`,
        // item_marks intentionally absent — this IS the "survives an index
        // rebuild" guarantee (D-10), enforced by omission, not a flag.
    } { /* ... */ }
}
```

**Why no FK:** `webspace_items.item_id` DOES carry `REFERENCES items(id) ON DELETE CASCADE` [VERIFIED: `kernel/index/schema.go:49`, `item_id TEXT NOT NULL REFERENCES items(id) ON DELETE CASCADE`]. If `item_marks` copied that shape, any `DELETE FROM items` — which `DeleteSourceItems` already performs on source removal [VERIFIED: `kernel/index/store.go:298-303`] — would silently cascade-delete marks *before* D-09/D-10's own explicit healthy-sync-only prune rule ever got a chance to apply, defeating the "always outranks... survives re-sync" guarantee for the narrower case of a removed-then-re-added source. A no-FK table, joined by plain `item_id` string equality at read time, makes the *only* deletion path the explicit sweep this phase writes.

### Pattern 2: Orphan-prune hooks into `ReplaceWebspaceSourceItems`'s existing transaction — for free, this already satisfies "healthy sync only"

**What:** `kernel/correlate/correlate.go`'s `SyncSource` [VERIFIED: read this session, lines 97-167] calls `Store.ReplaceWebspaceSourceItems` for a (webspace, source) pair in exactly two cases: (a) the source is de-allowlisted for that webspace (clears with `items=nil`, line 109), or (b) `src.Match(ctx, fields)` **succeeded** (line 157, only reached after the `if err != nil { ...; continue }` guard at line 119-123 has already skipped past any Match failure). **A failed Match for this (webspace, source) pair never reaches `ReplaceWebspaceSourceItems` at all.**

**When to use:** This is the exact code-level guarantee D-10 asks for ("prune fires only on a healthy sync that omits the item... a failed sync must never count as item gone"). Adding the prune-sweep *inside* `ReplaceWebspaceSourceItems`'s existing transaction — scoped to the same `webspaceName`/`source` parameters it already has — satisfies D-10 with no new plumbing to distinguish "healthy" from "failed" at the call site; that distinction already exists structurally.

**Example (the exact existing delete this phase's prune-sweep should mirror):**
```go
// kernel/index/store.go:208-213 — the existing scoped delete pattern
// (VERBATIM as currently written) that the new prune statement should
// copy the shape of, scoped identically by source via a subquery on items:
if _, err := tx.ExecContext(ctx, `
DELETE FROM webspace_items
WHERE webspace_name = ?
  AND item_id IN (SELECT id FROM items WHERE source = ?)
`, webspaceName, source); err != nil { /* ... */ }

// NEW prune statement (same shape, targets item_marks, restricted to items
// NOT in the just-synced set — bind the new item ID list or use a
// temp/anti-join against the freshly-inserted webspace_items rows):
// DELETE FROM item_marks
// WHERE webspace_name = ?
//   AND item_id IN (SELECT id FROM items WHERE source = ?)
//   AND item_id NOT IN (<the ids just re-inserted into webspace_items>)
```

**Open edge case to decide explicitly at plan time (not silently inherited):** the de-allowlist branch (`items=nil`, correlate.go:109) also reaches `ReplaceWebspaceSourceItems`. If the prune-sweep is unconditional, de-allowlisting a source from a webspace would also silently clear that source's marks for the webspace — arguably consistent with the "no orphaned rows" precedent `webspace_items` already follows, but CONTEXT.md's D-09/D-10 discussion frames "healthy sync that no longer reports the item", not "operator removed the source from participation". Flag this as a named decision for the plan, not an accidental side effect of the implementation shape.

### Pattern 3: Mark-aware read filtering — one shared SQL join point

**What:** `StreamItems` and `Search` both already funnel through `BuildMatchQuery` for the webspace's permanent-filter/live-query FTS narrowing [VERIFIED: `kernel/index/store.go:333-437`]. Mark-aware filtering (D-03: exclusion applies to stream, search, date markers, and counts uniformly) should be added as a second, independent `WHERE item_id NOT IN (SELECT item_id FROM item_marks WHERE webspace_name = ?)` clause (or `IN` for the excluded view) applied at the *same* two call sites, so a filter stack and an exclusion mark compose correctly rather than one silently overriding the other.

**When to use:** Every webspace item read — this is the "final filter tier" CONTEXT.md's phase boundary names explicitly ("this is the final selection criteria in the hierarchy of filter configs", `TODO.md` §kernel v1.1).

**Also check:** the `/agent/v1` webspace stream mirror. `kernel/httpapi/stream.go`'s own doc comment states "search and the agent stream mirror all call this [`webspaceIsKnown`]" — confirm during planning whether the agent-facing stream route calls `StreamItems` (and therefore inherits mark-filtering automatically) or has its own query path; if the latter, it needs the identical filter added explicitly. Marking this an **open verification item**, not assumed either way, since this session did not open `kernel/httpapi/agent.go`.

### Pattern 4: Build-time trust manifest — generated Go source, not `go:embed`

**What:** A new Makefile step, run after `plugins`/`plugins-portable` build the real binaries and before `go build -o bin/topos`, that hashes each `bin/plugins/topos-plugin-*` file with the exact same algorithm `pluginhost.HashBinary` already uses [VERIFIED: `kernel/pluginhost/binaryhash.go:26-38`, `sha256.New()` + `io.Copy` + `hex.EncodeToString`] and writes a generated `.go` file (e.g. `kernel/pluginhost/manifest_generated.go`, git-ignored) declaring a `map[string]string` of binary name → hex digest. `pluginhost.ResolveBinary`'s trusted-tier branch [VERIFIED: `kernel/pluginhost/discover_binaries.go:368-398`] re-hashes the on-disk binary and compares against this map; a match keeps `TierTrusted`, a mismatch or absent entry routes to D-13's refuse-to-load path via a **new** `LaunchFailure.Reason` constant declared beside the existing one:

```go
// kernel/pluginhost/host.go:53-57 — the exact existing pattern to extend,
// not replace:
const LaunchFailurePinMismatch = "pin_mismatch"
// NEW, same shape:
// const LaunchFailureManifestUnverified = "manifest_unverified"
```

**When to use:** This is the concrete implementation of D-12/D-13 — "derive trusted-tier status from a build-provenance manifest... a binary that doesn't verify against the manifest gets no trusted-tier treatment regardless of directory... refuses to load, loudly."

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| ServiceWorker precache + update lifecycle | A hand-written `sw.js` with manual `cache.addAll`/`fetch` interception | `vite-plugin-pwa`'s `generateSW` strategy | Workbox (which `generateSW` drives) already solves precache-manifest revisioning, update-on-new-build detection, and the "never stale" activation lifecycle — hand-rolling risks exactly the staleness bug this phase's success criterion 4 exists to prevent |
| PWA icon set derivation (192/512/maskable, safe-zone padding) | Manual ImageMagick/sharp resize scripts | `@vite-pwa/assets-generator` | Purpose-built for this exact task, integrates directly with `vite-plugin-pwa`'s manifest config, correctly handles maskable safe-zone padding (a common hand-rolled mistake) |
| Plugin binary hashing | A second hashing helper for manifest generation | `pluginhost.HashBinary` (existing) | The project already has a stated "one-hashing-convention discipline" [VERIFIED: `kernel/pluginhost/binaryhash.go:12-14` doc comment] — a second SHA-256 implementation for the same purpose would violate it |
| Named launch-failure vocabulary | A new ad-hoc string for the manifest-unverified state | Extend `LaunchFailure.Reason`'s existing closed vocabulary (`pin_mismatch` today) | `kernel/httpapi/sources.go`'s own doc comment calls this a "CLOSED-VOCABULARY reason" the SPA gates remedial UI on directly [VERIFIED: `kernel/httpapi/sources.go:81-88`] — a parallel ad-hoc mechanism would fragment that gate |

**Key insight:** every "don't hand-roll" item in this phase already has a load-bearing precedent living in this exact codebase (Workbox for SW lifecycle is the one external-library exception, and even that is the ecosystem standard, not custom). The research risk in this phase is not "which library" but "did you find the existing pattern before writing a parallel one."

## Common Pitfalls

### Pitfall 1: Reflexively bumping `schemaVersion` for the new `item_marks` table
**What goes wrong:** `schema.go`'s doc comment says "Bump this whenever a schema change... makes previously-indexed rows structurally incompatible" — a planner or executor pattern-matching on "we're changing schema.go" might bump `schemaVersion` to add `item_marks`, unintentionally triggering `rebuildOnSchemaChange`'s full drop-and-resync of `items`/`webspace_items`/`webspaces`/`sync_runs`/FTS on every existing user's next kernel start.
**Why it happens:** The doc comment's framing ("bump whenever a schema change...") reads as "any addition", but the actual trigger condition is row-shape incompatibility, not table addition.
**How to avoid:** `db.Exec(schema)` already runs unconditionally on every `Open()` call [VERIFIED: `kernel/index/store.go:61-64`, outside the `rebuildOnSchemaChange` gate] — a new `CREATE TABLE IF NOT EXISTS item_marks (...)` statement added to the `schema` string is picked up on the very next `Open()` for both fresh and existing databases, with **no version bump needed or wanted**.
**Warning signs:** A diff that touches `const schemaVersion = 3` alongside the new table — that's the tell.

### Pitfall 2: Makefile build order is backwards for a link-time manifest
**What goes wrong:** Both `build` (line: `CGO_ENABLED=0 go build -o bin/topos ./cmd/topos` then `$(MAKE) plugins`) and `build-portable` (line: `CGO_ENABLED=0 go build -o bin/topos ./cmd/topos` then `$(MAKE) plugins-portable`) [VERIFIED: `Makefile`, read this session] compile the kernel binary **before** the plugin binaries exist. `build-portable` is the actual recipe `.github/workflows/release.yml` runs [VERIFIED: `release.yml:38`, `run: make build-portable`]. A generated manifest that embeds plugin hashes into the kernel cannot be computed before those binaries exist.
**Why it happens:** The existing order was chosen for an unrelated reason (SPA build must precede kernel build since `go:embed` needs `kernel/webui/build` populated) and nobody needed plugin binaries to exist before the kernel compiled — until now.
**How to avoid:** Insert a manifest-generation step between the plugin build and the kernel build in **both** `build` and `build-portable`, and reorder so plugins build first: `plugins`/`plugins-portable` → generate manifest → `go build -o bin/topos`. `dev` is already correctly ordered (`dev: plugins` runs the prerequisite before the recipe body invokes `go run ./cmd/topos serve`) [VERIFIED: `Makefile` `dev: plugins` line + `DEV_KERNEL_CMD ?= go run ./cmd/topos serve`] — no change needed there beyond ensuring the manifest-generation step also runs as part of (or before) the `dev` recipe.
**Warning signs:** A `make build` that succeeds but produces a kernel whose manifest is empty or stale relative to the plugins it just built.

### Pitfall 3: `@vite-pwa/sveltekit` + `adapter-static`'s non-default fallback filename
**What goes wrong:** [CITED: vite-pwa SvelteKit docs, github.com/vite-pwa/vite-plugin-pwa/blob/main/docs/frameworks/sveltekit.md, checked via WebSearch this session] `adapter-static` generates its SPA fallback page **after** the `@vite-pwa/sveltekit` plugin runs, so the plugin cannot include a content-hash revision for that fallback in its precache manifest automatically — official guidance derives the revision from `.svelte-kit/output/client/_app/version.json` instead. This repo's fallback is named `200.html`, not the library's more commonly-documented default — the exact interaction needs to be verified end to end during implementation, not assumed to "just work" from the docs' generic example.
**Why it happens:** `svelte.config.js`'s `adapter-static` config uses `fallback: '200.html'` deliberately, to avoid colliding with adapter-static's prerendered-output handling of the more intuitive default name [VERIFIED: `web/svelte.config.js:16-22`, read this session] — a detail specific to this repo that generic vite-pwa SvelteKit examples won't reflect.
**How to avoid:** Test the installed-app "kernel upgraded, never stale" criterion specifically against a build that changes `200.html`'s content, not just a build that changes a JS chunk — the fallback page is the one artifact most likely to have a precache-revisioning gap.
**Warning signs:** SW installs and updates JS/CSS correctly but an operator who lands on a client-side route via a direct URL (which serves through `200.html`) sees stale markup.

### Pitfall 4: `.webmanifest` MIME type not registered on every OS
**What goes wrong:** `spaHandler` serves the embedded SPA build (including the generated `manifest.webmanifest`) via `http.FileServer(http.FS(assets))` [VERIFIED: `kernel/httpapi/routes.go:150-158`], which infers `Content-Type` via Go's `mime` package reading the OS's mime.types database. `.webmanifest` → `application/manifest+json` is not universally pre-registered across Linux distros/macOS/Windows.
**Why it happens:** Go's `mime.TypeByExtension` falls back to OS-level mime databases for extensions it doesn't hard-code, and browsers can be strict about a manifest's declared content-type for installability checks.
**How to avoid:** Call `mime.AddExtensionType(".webmanifest", "application/manifest+json")` once at kernel startup (a one-line fix, e.g. in `cmd/topos/main.go` or `kernel/httpapi`'s router construction) rather than relying on the host OS. Verify on the actual dev machine with a quick `curl -I` check before assuming this is or isn't already an issue — this is flagged `[ASSUMED]` pending that verification (see Assumptions Log).
**Warning signs:** Chrome DevTools' "Manifest" panel or `chrome://inspect` reporting a manifest fetch/parse error despite the file being byte-correct.

### Pitfall 5: Workbox `generateSW` accidentally precaching or runtime-caching `/api/*`
**What goes wrong:** `vite-plugin-pwa`'s default `generateSW` glob patterns target the build output directory; without an explicit `navigateFallbackDenylist`/`runtimeCaching` exclusion for `/api/*`, a same-origin API response could get runtime-cached, silently reintroducing the "Full offline PWA caching" this milestone's Requirements doc explicitly rules out ("caching API responses adds stale-data risk for zero benefit... SW scope is install + UI-shell freshness" [VERIFIED: `.planning/REQUIREMENTS.md` Out of Scope table, read this session]).
**Why it happens:** The build-output glob naturally only touches `web/`'s static output, so `/api/*` isn't precached by default — but a developer adding `runtimeCaching` rules for offline resilience (a common PWA tutorial pattern) could inadvertently widen scope beyond what this phase authorizes.
**How to avoid:** Keep the SW config to app-shell precache only; do not add any `runtimeCaching` entry matching `/api/`. If a "kernel not running" friendlier state is wanted (see Open Questions), achieve it by ensuring the app *shell* (HTML/JS/CSS) is precached so the existing `StreamError.svelte` client-side error UI can render — never by caching the API response itself.
**Warning signs:** A stream that shows stale items after the kernel has synced new ones, without a page reload.

### Pitfall 6: Manifest hash list built from the wrong plugin set (Signal / mockstrict confusion)
**What goes wrong:** `plugins`/`plugins-portable` build different binary sets — `plugins` includes `topos-plugin-signal` (via its `$(MAKE) signal` call at the end) [VERIFIED: `Makefile` `plugins:` target], `plugins-portable` does not [VERIFIED: `Makefile` `plugins-portable:` target, six binaries, no signal]. A manifest-generation step that unconditionally globs `bin/plugins/topos-plugin-*` will pick up whatever happens to be sitting there from a prior local build, not necessarily what the current recipe just built (e.g., a stale `topos-plugin-signal` left over from an earlier `make plugins` run, now present during a `make build-portable` run that never re-built it).
**Why it happens:** The manifest generator, if implemented as a blind directory glob rather than driven by the same explicit binary list the Makefile targets already enumerate, can't distinguish "just built by this recipe" from "leftover from an earlier build."
**How to avoid:** Either (a) have the manifest-generation step consume the exact same explicit binary list each Makefile target already hand-enumerates (mirroring the "one hashing convention"/"one module list" discipline the Makefile itself already documents for `test`/`test-portable`), or (b) `rm -rf bin/plugins` at the start of each build recipe before rebuilding, so a stale binary can never survive into the current manifest. `topos-plugin-mockstrict` and `topos-plugin-mock` should be handled deliberately too: `mock` is a legitimate trusted-tier binary that ships and should be in the manifest (it's excluded only from the UI *picker*, per `ExcludedPluginBinaries`, not from launch); `mockstrict` is built only by `make e2e` and must never appear in a real `bin/plugins/` manifest.
**Warning signs:** A release kernel whose manifest names a binary that isn't actually shipped in that release's artifact set, or is missing one that is.

## Code Examples

### Existing hashing helper to reuse for manifest generation
```go
// Source: kernel/pluginhost/binaryhash.go (read in full this session)
func HashBinary(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("pluginhost: hash binary %s: %w", path, err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("pluginhost: hash binary %s: %w", path, err)
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
```

### Existing named-launch-failure pattern to extend for D-13
```go
// Source: kernel/pluginhost/host.go (read in full this session)
const LaunchFailurePinMismatch = "pin_mismatch"

type LaunchFailure struct {
	Instance    string
	Plugin      string
	DisplayName string
	Tier        Tier
	Reason      string
	PinnedHash  string
	CurrentHash string
	Message     string
}
```

### Existing scoped-delete pattern to mirror for the marks prune-sweep
```go
// Source: kernel/index/store.go ReplaceWebspaceSourceItems (read in full this session)
if _, err := tx.ExecContext(ctx, `
DELETE FROM webspace_items
WHERE webspace_name = ?
  AND item_id IN (SELECT id FROM items WHERE source = ?)
`, webspaceName, source); err != nil {
	return fmt.Errorf("index: clear webspace_items for %s/%s: %w", webspaceName, source, err)
}
```

### Secure-context behavior confirming the desktop install path already works
[CITED: MDN "Making PWAs installable", developer.mozilla.org/en-US/docs/Web/Progressive_web_apps/Guides/Making_PWAs_installable, checked via WebSearch this session] — `http://localhost`/`http://127.0.0.1` (any port) is treated as a secure context for both ServiceWorker registration and install eligibility; a LAN IP over plain HTTP (e.g. `http://192.168.x.x:7777`) is not. This maps directly onto the kernel's existing loopback default and its own warning:
```go
// Source: cmd/topos/main.go (read in full this session)
listen := cfg.Server.Listen
if !isLoopback(listen) {
	logger.Warn("kernel HTTP listener is not bound to loopback — this exposes the API beyond this machine", "listen", listen)
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Trust tier = which directory a binary resolved from (Phase 11 shipped shape) | Trust tier = build-provenance manifest (this phase, D-12) | This phase | Closes the config-edit / file-drop / shadowing bypass paths the folded todo documents; directory location becomes an install *convenience*, not the trust *authority* |
| Browser tab only | Installable PWA (manifest + SW + icons) | This phase (UI-13) | Desktop users can launch topos as a standalone app window; no change to the underlying HTTP API surface |

**Deprecated/outdated:** none — this phase adds capability, it does not retire any existing mechanism (the directory-tier discovery machinery from Phase 11 stays; D-16 explicitly keeps the two plugin directories this phase).

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `.webmanifest` is not universally registered in the OS mime database `net/http`'s `FileServer` consults, so an explicit `mime.AddExtensionType` call is needed | Common Pitfalls #4 | Low — a quick `curl -I` check on the actual dev machine during implementation resolves this in minutes either way; if wrong, this is simply a no-op safety line, not a broken feature |
| A2 | The `/agent/v1` webspace stream mirror shares `StreamItems`/mark-filtering automatically, rather than having its own separate query path that would need the exclusion filter added explicitly | Architecture Pattern 3 | Medium — if the agent mirror has an independent SQL path, an agent-driven read (AGENT-01 grants) could leak excluded items even though the human-facing UI correctly hides them; `kernel/httpapi/agent.go` was not opened this session and must be checked before or during planning |
| A3 | The recommended registerType (`autoUpdate`) is the right default for this single-user, read-only desktop app, versus a `prompt`-based update-available toast | Standard Stack / State of the Art | Low — CONTEXT.md explicitly leaves this to Claude's discretion; either choice satisfies UI-13's "never stale" criterion, this is a UX tradeoff not a correctness risk |
| A4 | A de-allowlisted source's marks should be pruned by the same unconditional sweep as a genuinely-vanished item (rather than treated as a distinct, undecided case) | Architecture Pattern 2 | Medium — CONTEXT.md's D-09/D-10 do not explicitly address this edge case; an unreviewed default here could surprise an operator who de-allowlists a source expecting to preserve its marks for a future re-allowlist |

## Open Questions

1. **Does the `/agent/v1` stream mirror need its own explicit mark-filter addition, or does it inherit `StreamItems`'s automatically?**
   - What we know: `kernel/httpapi/stream.go`'s doc comment states the agent stream mirror shares the same `webspaceIsKnown` existence gate as the human-facing stream and search handlers.
   - What's unclear: whether it also shares the *same query method* (`StreamItems`) for its item read, or has a separate handler in `kernel/httpapi/agent.go` (not opened this session) with its own SQL.
   - Recommendation: the planner should open `kernel/httpapi/agent.go` before finalizing the mark-filter task list; if it has an independent path, add the identical filter clause there explicitly rather than assuming inheritance.

2. **Should de-allowlisting a source from a webspace prune that source's marks in the same webspace, or preserve them for a future re-allowlist?**
   - What we know: the existing `webspace_items` precedent clears rows outright on de-allowlist (no orphan preservation); D-09/D-10 discuss only the "item genuinely vanished from a healthy sync" case.
   - What's unclear: whether CONTEXT.md's authors considered de-allowlist-then-re-allowlist as a distinct scenario worth preserving marks through.
   - Recommendation: surface this explicitly as a plan-time decision (not an accidental default of wherever the prune-sweep code happens to sit), defaulting to "prune on de-allowlist too" for consistency with the existing no-orphan precedent unless the plan/discuss step says otherwise.

3. **What exactly should the installed PWA window show when the kernel process is not running?**
   - What we know: CONTEXT.md leaves this to Claude's discretion; no offline API caching is in scope; `StreamError.svelte` already exists as the client-side "kernel didn't respond" UI for a live kernel that returns an error.
   - What's unclear: whether that existing component can be reached at all if the kernel process itself is down (vs. merely erroring) — reaching it requires the app *shell* (HTML/JS/CSS) to be served from the SW precache even when the kernel's HTTP server isn't listening at all, which is a meaningfully different failure mode than a live kernel returning a 5xx.
   - Recommendation: precache the app shell (not API responses) so the existing error-state UI has a chance to render regardless of why the kernel is unreachable; verify this behavior specifically (kernel process killed, not just erroring) during implementation.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Node.js / npm | `web/` build, PWA tooling install | ✓ (existing project dependency, unchanged by this phase) | per `web/package.json` engines (not separately re-verified this session) | — |
| Go toolchain | Kernel/manifest build | ✓ | 1.25.0 [VERIFIED: `go.mod` read this session, `go 1.25.0`] | — |
| System `sqlcipher` (Signal plugin build only) | `make signal` / `make plugins` (not `plugins-portable`) | Not probed this session (pre-existing project dependency, unrelated to this phase's new work) | — | `plugins-portable`/`build-portable` already avoid this dependency; the manifest-generation step should be built to work correctly against either binary set (Pitfall 6) |

No new external service, database, or network dependency is introduced by this phase — the PWA tooling runs entirely at `web/` build time, and the trust manifest is generated entirely at Go build time from files already on disk.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | No new authentication surface; loopback-only, no-auth model unchanged (`docs/api.md` "Loopback-only default, no auth") |
| V3 Session Management | No | No session concept introduced |
| V4 Access Control | Yes | Trust-tier hardening IS an access-control change: build-provenance manifest replaces directory-location as the trust authority (D-12); refuse-to-load on verification failure (D-13) is a fail-closed control |
| V5 Input Validation | Yes | New marks endpoints accept item ids / webspace names from the client — must reuse the existing id-validation discipline (`docs/api.md`'s stable-ID scheme, `filepath.Base`-style bare-name validation already used for plugin binary names in `pluginhost.validatePluginBinaryName`) rather than trusting client-supplied ids/paths unchecked |
| V6 Cryptography | Yes | SHA-256 binary hashing for the manifest is, per the existing codebase's own documented framing, "narrowly an integrity control, not a cryptographic authentication feature" [VERIFIED: `kernel/pluginhost/binaryhash.go:20-25` doc comment] — this phase's manifest inherits that same limitation and must not be described in docs/UI as a stronger guarantee than it is (no signature verification, no publisher authentication) |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Config-edit bypass of trust tier (`plugins.dir` retargeted via `PUT /api/config`) | Elevation of Privilege | Closed by D-12: trust is now derived from the build-time manifest, not directory location, so retargeting `plugins.dir` no longer silently promotes arbitrary binaries to trusted |
| File-drop bypass (dropping a third-party binary into the trusted dir) | Elevation of Privilege | Closed by D-12/D-13: an unverifiable binary (absent from the manifest) refuses to load rather than launching as trusted |
| Shadowing (same-named binary in trusted dir shadows a pinned external plugin) | Spoofing / Elevation of Privilege | D-14: surfaced in the UI (not just a log line) — this phase must extend the existing `ResolveBinary` shadow-log warning [VERIFIED: `kernel/pluginhost/discover_binaries.go:378-386`] with a UI-visible health state |
| TOCTOU on plugin binary swap between add-time and launch-time | Tampering | Already mitigated for external tier (re-verify at every launch, unchanged); this phase extends the identical re-verify-at-every-launch discipline to the trusted tier via the manifest |
| Client-supplied item/webspace id used unsanitized in a mark SQL statement | Tampering / Injection | Use parameterized queries throughout (the existing codebase's universal convention — every example read this session uses `?` placeholders, never string concatenation) |

## Sources

### Primary (HIGH confidence)
- `kernel/index/schema.go`, `kernel/index/store.go` — read in full this session (schema, rebuild-drop list, `ReplaceWebspaceSourceItems`, `StreamItems`, `Search`)
- `kernel/correlate/correlate.go` — read in full this session (`SyncSource`'s per-webspace healthy/failed branching)
- `kernel/httpapi/stream.go`, `kernel/httpapi/routes.go`, `kernel/webui/embed.go` — read in full this session
- `kernel/pluginhost/discover_binaries.go`, `kernel/pluginhost/binaryhash.go`, `kernel/pluginhost/host.go` (partial), `kernel/config/types.go` — read in full/partial this session
- `Makefile`, `.github/workflows/release.yml` — read this session (build order, release recipe)
- `web/package.json`, `web/svelte.config.js`, `web/vite.config.ts`, `web/src/routes/+layout.svelte`, `web/src/lib/components/StreamList.svelte`, `StreamRow.svelte`, `DetailPane.svelte`, `WebspaceHeader.svelte` (partial), `TrustBadge.svelte`/`SourceChip.svelte` (partial) — read this session
- `docs/api.md`, `docs/plugin-contract.md`, `docs/testing.md` (partial) — read this session
- `.planning/phases/13-per-item-curation-installable-app/13-CONTEXT.md`, `.planning/REQUIREMENTS.md`, `.planning/STATE.md`, `.planning/research/SUMMARY.md`, `TODO.md`, `.planning/todos/pending/2026-08-13-plugin-trust-tier-is-directory-location-not-provenance.md` — read in full this session
- npm registry (`npm view`) — checked live this session for `vite-plugin-pwa`, `@vite-pwa/sveltekit`, `workbox-build`, `workbox-window` versions/peerDependencies
- `gsd-tools query package-legitimacy check` — run this session for all five npm packages, all `OK`

### Secondary (MEDIUM confidence)
- MDN "Making PWAs installable" (developer.mozilla.org) and MDN "Secure Contexts" — checked via WebSearch this session, confirms localhost/127.0.0.1 secure-context treatment and LAN-IP-over-HTTP limitation
- vite-pwa SvelteKit framework docs (github.com/vite-pwa/vite-plugin-pwa/blob/main/docs/frameworks/sveltekit.md, github.com/vite-pwa/sveltekit) — checked via WebSearch this session, confirms the `adapter-static` fallback-revisioning caveat

### Tertiary (LOW confidence)
- None relied upon for load-bearing claims in this document.

## Metadata

**Confidence breakdown:**
- Standard stack (PWA libraries): HIGH — versions and peer-dependency compatibility confirmed live against npm registry this session, package legitimacy checked
- Architecture (marks storage/filtering, trust manifest): HIGH — every load-bearing claim traces to a specific file/line read this session, not training-data recall
- Pitfalls: HIGH — the Makefile build-order pitfall and the schema-version pitfall are both directly verified against the actual files, not inferred

**Research date:** 2026-08-14
**Valid until:** 30 days (stable, in-repo-grounded findings) / 7 days for the PWA library version pins specifically, since `vite-plugin-pwa`/`@vite-pwa/sveltekit` publish frequently

---
*Phase: 13-per-item-curation-installable-app*
*Research completed: 2026-08-14*
