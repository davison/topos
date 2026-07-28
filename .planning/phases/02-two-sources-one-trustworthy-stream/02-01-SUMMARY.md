---
phase: 02-two-sources-one-trustworthy-stream
plan: 01
subsystem: sync-engine
tags: [go-plugin, sqlite, goldmark, bluemonday, silverbullet, tls, sveltekit]

requires:
  - phase: 01-first-webspace-end-to-end
    provides: kernel + plugin architecture (go-plugin/gRPC contract), hybrid data model index, stream + detail pane UI, paperless-ngx as the reference plugin
provides:
  - "plugins/silverbullet: a second, structurally different source plugin (wiki pages vs. documents), Describe/Match/Fetch/Health complete"
  - "kernel/index.Store.ReplaceWebspaceSourceItems: per-(webspace, source_type) persistence, replacing the whole-webspace ReplaceWebspaceItems"
  - "kernel/correlate.Engine.SyncSource/SyncAll: source-major sync loop — one source's Match failure never discards a healthy source's items"
  - "kernel/httpapi allowedRenditionTypes + text/html rendition support for rendered markdown"
  - "web/src/lib/api.ts sourceDisplayName helper — parameterized source-specific UI copy (DetailPane failure alert, OpenInSource button label)"
  - "optional per-source ca_cert config for self-signed-TLS source instances"
affects: [02-02, 02-03, 02-04, kernel/sync (coordinator, later plan in this phase)]

tech-stack:
  added:
    - "github.com/adrg/frontmatter v0.2.0 (YAML frontmatter parsing)"
    - "golang.org/x/sync v0.22.0 (errgroup, bounded Match concurrency)"
    - "github.com/yuin/goldmark v1.8.5 (markdown -> HTML, safe-by-default)"
    - "github.com/microcosm-cc/bluemonday v1.0.27 (HTML sanitization, UGCPolicy)"
  patterns:
    - "Source-major sync loop (SyncSource per source, iterated by SyncAll) replaces webspace-major — the identity a sync operates on is (webspace, source_type), not webspace"
    - "Source-scoped index replace (ReplaceWebspaceSourceItems) — delete predicate is (webspace_name, source_type), never a whole-webspace delete"
    - "Rendered-markdown-as-iframe: a plugin sanitizes server-side (goldmark + bluemonday), the kernel allowlists the MIME type and serves it under the existing sandboxed-iframe CSP — the same pattern PDF renditions already used"
    - "sourceDisplayName(source_type) local mapping in web/src/lib/api.ts — parameterizes any UI copy that used to hardcode one source's name"

key-files:
  created:
    - plugins/silverbullet/{go.mod,go.sum,main.go,client.go,plugin.go,frontmatter.go,render.go}
    - plugins/silverbullet/{client_test.go,frontmatter_test.go,render_test.go,fetch_test.go,outbound_hosts_test.go}
  modified:
    - kernel/index/store.go (ReplaceWebspaceSourceItems)
    - kernel/correlate/correlate.go (SyncSource, SyncAll, WebspaceResult.SourceType)
    - kernel/httpapi/item.go (text/html rendition allowlist)
    - kernel/config/{types.go,config.go} (optional ca_cert field + ~-expansion)
    - kernel/pluginhost/host.go (ca_cert threaded into plugin subprocess env)
    - internal/audit/outbound_hosts_test.go (sanctionedEgressFile -> sanctionedEgressFiles)
    - cmd/webspaces/main.go (runSync per-(webspace,source) printing)
    - web/src/lib/components/{DetailPane.svelte,OpenInSource.svelte}, web/src/lib/api.ts
    - config.example.toml, docs/api.md, go.work, go.work.sum, Makefile, .gitignore

key-decisions:
  - "Sync identity promoted from 'webspace' to '(webspace, source_type)' — a Match failure on one source never discards, delays, or rolls back a healthy sibling source's items (fixes the Phase-1-only-correct-for-one-source bug RESEARCH.md flagged as the Critical Architecture Finding)"
  - "ReplaceWebspaceItems deleted outright (not kept as a compatibility wrapper) in favor of ReplaceWebspaceSourceItems — leaving a whole-webspace write path in place would be exactly the bug Phases 3-5 inherit"
  - "Added an optional ca_cert config field (kernel/config.Source) and threaded it through pluginhost + NewClient/NewSourcePlugin (now 3-arg) to pin a self-signed CA — discovered live against the user's real SilverBullet instance, not anticipated by the plan's 2-arg constructor sketch"
  - "Client always sends X-Sync-Mode: true and treats a text/html response from /.fs as an error rather than attempting to parse it — this SilverBullet v2 instance answers unrecognized/malformed requests with its SPA HTML shell (200 or 307), not a 4xx"
  - "Fixed a hardcoded 'paperless-ngx didn't respond' / 'Open in paperless-ngx' UI-copy bug (RESEARCH.md Pitfall 3) discovered by live user verification mid-plan — added sourceDisplayName() to parameterize both DetailPane's failure alert and OpenInSource's button label"

patterns-established:
  - "Source plugin Fetch must re-derive its own on-source path from source_id when the two differ (D-01 strips the .md extension for source_id; the real file always has it) — a bug caught only by live verification, now covered by fetch_test.go's fixture server rejecting any other path"
  - "A text/html rendition that IS the item's content (not a preview alongside separate text, like SilverBullet's rendered markdown) gets its own full-pane-body layout branch, checked before the shared small-preview-box branch other rendition types share"
  - "A plugin that returns text/html for direct iframe rendering must wrap its sanitized fragment in a minimal self-contained document (WrapDocument) carrying a fixed, hardcoded stylesheet matching the app's theme tokens — an iframe document has no access to the SPA's own CSS, and unstyled HTML renders unreadably against the app's dark background"

requirements-completed: [SRC-05]
# KERN-04 ("Sync scheduler with a per-plugin coordinator; user can trigger
# manual refresh") is NOT fully complete by this plan — only its
# persistence-correctness prerequisite (source-major sync, never discarding
# a healthy source's items) landed here. The scheduler/coordinator/manual-
# refresh apparatus itself (RESEARCH.md kernel/sync/coordinator.go, D-06,
# D-07) is scoped to a later plan in this phase. Left pending in
# REQUIREMENTS.md rather than marked complete.

coverage:
  - id: D1
    description: "A SilverBullet page whose frontmatter/inline tags or page name match a webspace keyword appears interleaved with paperless-ngx documents in one chronological stream, with an exact deep link"
    requirement: SRC-05
    verification:
      - kind: unit
        ref: "plugins/silverbullet/frontmatter_test.go#TestMatchesKeyword_Table"
        status: pass
      - kind: integration
        ref: "live sync against the user's real SilverBullet + paperless-ngx instances: GET /api/webspaces/house-move/stream returned 52 items (35 paperless + 17 silverbullet), interleaved by timestamp"
        status: pass
    human_judgment: true
    rationale: "Visual interleaving and deep-link correctness in the actual browser UI were confirmed directly by the user mid-plan (tracer checkpoint); recorded here as evidence, not re-asked."
  - id: D2
    description: "A sync cycle where one source's Match fails still persists every other source's freshly matched items — the failing source's previous rows survive untouched"
    requirement: KERN-04
    verification:
      - kind: unit
        ref: "kernel/correlate/correlate_test.go#TestSyncAll_PartialSourceFailure_HealthySourceItemsPersist"
        status: pass
      - kind: unit
        ref: "kernel/index/store_test.go#TestReplaceWebspaceSourceItems_OtherSourceRowsUntouched"
        status: pass
      - kind: integration
        ref: "live sync with SB_URL pointed at an unreachable host: paperless still synced 35 items; a subsequent serve confirmed all 17 silverbullet rows from the prior successful sync were still present"
        status: pass
    human_judgment: false
  - id: D3
    description: "Opening a SilverBullet item renders its page content as sanitized HTML inside the same sandboxed rendition iframe pattern the PDF rendition already uses, filling the pane's body and matching the app's dark theme"
    verification:
      - kind: unit
        ref: "plugins/silverbullet/render_test.go#TestRenderSanitized_StripsRawScriptElement"
        status: pass
      - kind: unit
        ref: "plugins/silverbullet/render_test.go#TestWrapDocument_InjectsThemeStyleAndPreservesFragment"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/item_test.go#TestItemContentHandler_TextHTMLRenditionServedWithSecurityHeaders"
        status: pass
      - kind: integration
        ref: "live GET /api/items/{silverbullet id}/content returned 200, Content-Type text/html, sanitized rendered HTML body wrapped in the theme-styled document"
        status: pass
    human_judgment: true
    rationale: "First tracer-gate UAT pass (screenshot at /home/darren/Pictures/.clip.png) found the rendered pane cramped into the small preview box with near-unreadable dark-on-dark text; both are now fixed (fix(02-01) commits 36555ec, 26e24b6) and verified at the API/markup level, but final visual confirmation of the fix is a human re-verification step this executor cannot self-approve — no browser tooling was available in this session to take a confirming screenshot."
  - id: D4
    description: "SilverBullet plugin's read-only and host-pinning guarantees are enforced by committed tests, not convention"
    verification:
      - kind: unit
        ref: "plugins/silverbullet/outbound_hosts_test.go#TestAllowHost_PredicateTable"
        status: pass
      - kind: unit
        ref: "plugins/paperless/readonly_test.go#TestPluginsIssueOnlyGetRequests (covers plugins/silverbullet automatically)"
        status: pass
      - kind: unit
        ref: "internal/audit/outbound_hosts_test.go#TestNoForeignEgressOutsideSanctionedClient"
        status: pass
    human_judgment: false

duration: 88min
completed: 2026-07-28
status: complete
---

# Phase 2 Plan 1: SilverBullet Joins Paperless in One Cross-Source Stream Summary

**SilverBullet wiki pages and paperless-ngx documents now interleave in one chronological stream via a source-major sync engine (`ReplaceWebspaceSourceItems`/`SyncSource`) that survives a sibling source's failure, with SilverBullet page content rendered as goldmark+bluemonday-sanitized HTML in the existing sandboxed iframe.**

## Performance

- **Duration:** 88 min (active implementation, from live-credential resume through the two post-tracer-gate UAT fixes)
- **Completed:** 2026-07-28
- **Tasks:** 3 completed, plus 2 post-tracer-gate UAT fix commits
- **Files modified:** 34 (excluding `.planning/`) — the 2 fix commits touched files already in this set (`DetailPane.svelte`, `plugin.go`, `render.go`, `render_test.go`), no new files

## Accomplishments

- Promoted the sync identity from `webspace` to `(webspace, source_type)`: `kernel/index.Store.ReplaceWebspaceSourceItems` replaces `ReplaceWebspaceItems` outright, and `kernel/correlate.Engine.SyncSource`/`SyncAll` persist each source's contribution independently — fixing the partial-source-failure bug that Phase 1's one-source-only design left latent.
- Built `plugins/silverbullet` as a complete, independent Go module: a host-pinned + TLS-CA-pinned `/.fs` HTTP client, frontmatter/inline-tag extraction and D-03 keyword matching, and the full `Describe`/`Match`/`Fetch`/`Health` plugin contract.
- SilverBullet page content renders as sanitized HTML (goldmark + `bluemonday.UGCPolicy()`) inside the same sandboxed `<iframe>` pattern the PDF rendition already used — `kernel/httpapi`'s MIME allowlist and `docs/api.md` both updated.
- Verified everything against the user's real, live SilverBullet instance (not just fixtures): resolved the `/.fs` envelope shape, discovered and fixed three live-only failure modes (TLS self-signed cert, `X-Sync-Mode` header requirement, trailing-slash `//.fs` bug), and proved the partial-source-failure fix by pointing `SB_URL` at an unreachable host mid-session and confirming paperless's 35 items stayed fresh while silverbullet's 17 previously-synced items survived untouched.

## Task Commits

1. **Task 1: A SilverBullet page appears in the stream beside a paperless document** - `f1bec9a` (feat)
2. **Task 2: Opening a SilverBullet page renders its content, sanitized, in the detail pane** - `955028a` (feat)
3. **Task 3: Prove the SilverBullet plugin's read-only and host-pinning guarantees mechanically** - `5615ecf` (test)

_Note: this plan's `tdd="true"` tasks were executed as single atomic commits per the `type="tracer"`/`type="auto"` commit protocol (real implementation + real tests + real `<verify>` together), not as separate RED/GREEN commits — the task's own tracer-type instructions ("execute and commit exactly like `type="auto"`") take precedence over the generic per-task TDD split for a cross-cutting change touching 14+ files across three Go modules._

### Post-tracer-gate UAT fixes

The tracer feedback gate's human verification (stream interleaving and deep links) passed; the detail-pane rendering did not, on two counts (screenshot evidence at `/home/darren/Pictures/.clip.png`). Both were fixed as separate `fix(02-01)` commits and are recorded as deviations 5 and 6 below:

4. **fix: SilverBullet rendered content fills the detail pane body** - `36555ec` (fix)
5. **fix: inject app-matching dark theme into rendered SilverBullet HTML** - `26e24b6` (fix)

## Files Created/Modified

- `plugins/silverbullet/client.go` - host-pinned, TLS-CA-pinnable, GET-only `/.fs` HTTP client
- `plugins/silverbullet/frontmatter.go` - frontmatter+inline tag extraction, D-03 keyword matching, snippet truncation
- `plugins/silverbullet/plugin.go` - `Describe`/`Match`/`Fetch`/`Health` adapter
- `plugins/silverbullet/render.go` - goldmark render + bluemonday sanitize
- `plugins/silverbullet/main.go` - subprocess entrypoint
- `plugins/silverbullet/{client_test.go,frontmatter_test.go,render_test.go,fetch_test.go,outbound_hosts_test.go}` - full test coverage (read-only AST scan covered automatically by `plugins/paperless/readonly_test.go`)
- `kernel/index/store.go` - `ReplaceWebspaceSourceItems` (replaces `ReplaceWebspaceItems`)
- `kernel/correlate/correlate.go` - `SyncSource`, restructured `SyncAll`, `WebspaceResult.SourceType`
- `kernel/httpapi/item.go` - `text/html` added to `allowedRenditionTypes`
- `kernel/config/{types.go,config.go}` - optional `ca_cert` source field + `~` expansion
- `kernel/pluginhost/host.go` - `ca_cert` threaded into the plugin subprocess's `WEBSPACES_SOURCE_CONFIG`
- `internal/audit/outbound_hosts_test.go` - `sanctionedEgressFile` widened to `sanctionedEgressFiles`
- `cmd/webspaces/main.go` - `runSync` prints one line per `(webspace, source)`
- `web/src/lib/api.ts` - `sourceDisplayName()` helper
- `web/src/lib/components/DetailPane.svelte` - `text/html` iframe branch, parameterized failure copy, raw-text suppressed for `text/html` renditions
- `web/src/lib/components/OpenInSource.svelte` - parameterized button label
- `config.example.toml`, `docs/api.md`, `go.work`, `go.work.sum`, `Makefile`, `.gitignore` - wiring/docs

## Decisions Made

- **Sync identity promotion is the load-bearing change this plan makes** — every later plugin (Phases 3-5) inherits `ReplaceWebspaceSourceItems`/`SyncSource`, not the old whole-webspace path, which was deleted rather than kept as a compatibility shim.
- **`ca_cert` is a generic per-source config field**, not SilverBullet-specific — any future LAN source behind a self-signed cert can use the same mechanism.
- **`sourceDisplayName()` is a small local mapping**, not a call to a not-yet-built `GET /api/sources` endpoint — that endpoint (with a live, plugin-reported `display_name`) is explicitly later-plan scope (RESEARCH.md "Health merge"); this is the minimal fix for the correctness bug found now.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical functionality] TLS CA pinning for a self-signed SilverBullet instance**
- **Found during:** Task 1 Step 0 (live verification against the real instance)
- **Issue:** The plan's `NewClient(baseURL, token string) *Client` / `NewSourcePlugin(baseURL, token string) *SourcePlugin` signatures had no way to trust a self-signed certificate; Go's default TLS verification cannot connect to the user's real instance at all.
- **Fix:** Added a third `caCertPath string` parameter to both constructors, an optional `ca_cert` field on `kernel/config.Source` (with `~` expansion, mirroring the index-path pattern), and threading through `kernel/pluginhost/host.go`'s subprocess env.
- **Files modified:** `plugins/silverbullet/{client.go,plugin.go,main.go}`, `kernel/config/{types.go,config.go}`, `kernel/pluginhost/host.go`, `config.example.toml`
- **Verification:** Live `curl --cacert` and Go client requests against `https://notes.davisononline.home` both succeed; `~/.config/webspaces/silverbullet-ca.pem` verified as the correct pinned CA.
- **Committed in:** `f1bec9a`

**2. [Rule 1 - Bug] `X-Sync-Mode: true` header requirement and base-URL trailing-slash normalization**
- **Found during:** Task 1 Step 0
- **Issue:** This SilverBullet v2 instance answers `/.fs` requests without `X-Sync-Mode: true` (or with a doubled `//.fs` from an un-normalized trailing base_url slash) with its SPA HTML shell (200/307) instead of JSON — silently un-parseable, not a clean error.
- **Fix:** `client.go` always sends the header, normalizes the base URL via `strings.TrimRight(baseURL, "/")`, and treats any `text/html` response as an error rather than attempting to decode it.
- **Files modified:** `plugins/silverbullet/client.go`
- **Verification:** `TestClient_NormalizesTrailingSlashInBaseURL`, live `curl` reproduction of both failure modes recorded during Step 0.
- **Committed in:** `f1bec9a`

**3. [Rule 1 - Bug] `Fetch` didn't re-append the `.md` extension `Match` had stripped**
- **Found during:** Task 2 live verification (`GET /api/items/{silverbullet id}` returned `item_not_found` for every real item)
- **Issue:** `source_id` is the page path *without* `.md` (D-01), but the actual file on the instance always carries it; `fetchFull` was calling `ReadFile` on the bare path, which always 404s.
- **Fix:** `fetchFull` now appends `.md` before calling `ReadFile`.
- **Files modified:** `plugins/silverbullet/plugin.go`, `plugins/silverbullet/fetch_test.go` (fixture server now only answers the correctly-suffixed path, so this is a committed regression test)
- **Verification:** Live `GET /api/items/{id}` and `/content` now both succeed against the real instance.
- **Committed in:** `955028a`

**4. [Rule 1 - Bug] Hardcoded "paperless-ngx" copy in the detail pane and open-in-source button**
- **Found during:** Task 2, reported live by the user after visually verifying the tracer checkpoint
- **Issue:** `DetailPane.svelte`'s failure alert and `OpenInSource.svelte`'s button both hardcoded "paperless-ngx" regardless of the item's actual source (RESEARCH.md's own flagged "Pitfall 3") — a SilverBullet item's failure state showed the wrong source name, and (before Task 2's Fetch fix) every SilverBullet item hit that failure state.
- **Fix:** Added `sourceDisplayName(sourceType)` to `web/src/lib/api.ts` (local mapping: `paperless` -> "paperless-ngx", `silverbullet` -> "SilverBullet", fallback to the raw `source_type`); parameterized both components.
- **Files modified:** `web/src/lib/api.ts`, `web/src/lib/components/{DetailPane.svelte,OpenInSource.svelte}`
- **Verification:** `npm run check` (0 errors), visual re-check after rebuild showed correct source-specific copy.
- **Committed in:** `955028a`

**5. [Rule 1 - Bug] Rendered SilverBullet content confined to the small PDF-preview box**
- **Found during:** Tracer feedback gate human UAT (post-Task-3), screenshot evidence at `/home/darren/Pictures/.clip.png`
- **Issue:** `DetailPane.svelte`'s `text/html` branch was nested inside the same `h-72` fixed-height box paperless's PDF thumbnail uses, with an empty raw-text area below it. For SilverBullet the rendered content IS the item's content, not a preview alongside separately-shown text — it needs the pane's full remaining body.
- **Fix:** Gave the `text/html` branch its own top-level layout path (checked before the shared PDF/image/text branch), occupying `min-h-0 flex-1` (the pane's full remaining width/height) instead of the small box. Content still scrolls inside the iframe's own document (UI-SPEC requirement, unchanged) — only the outer container's sizing changed.
- **Files modified:** `web/src/lib/components/DetailPane.svelte`
- **Verification:** `npm run check` (0 errors); no `{@html}` directive introduced (`! grep -F '@html'` still exits 0); live API response unchanged (layout-only fix).
- **Committed in:** `36555ec`

**6. [Rule 1 - Bug] Unstyled rendered HTML unreadable against the app's dark theme**
- **Found during:** Tracer feedback gate human UAT (post-Task-3), same screenshot evidence
- **Issue:** The sanitized HTML fragment `Fetch` returned carried no CSS of its own; rendered inside the iframe it inherited the browser default (near-black text) over the app's dark (`#0f172a`/`#020617`) background — close to unreadable.
- **Fix:** New `plugins/silverbullet/render.go` `WrapDocument(sanitizedFragment)` wraps `RenderSanitized`'s output in a minimal, self-contained HTML document with a fixed, hardcoded `<style>` block matching `web/src/app.css`'s theme tokens (`--card` background, `--foreground` text, `--primary` links, `--muted`/`--muted-foreground` for code/quotes). No external stylesheet fetch, no `@import`, no `url()`. Runs strictly *after* `bluemonday.SanitizeBytes` and is never re-passed through the sanitizer — the stylesheet is a Go string literal, never derived from page content, so it cannot reintroduce any stripped XSS surface.
- **Files modified:** `plugins/silverbullet/render.go`, `plugins/silverbullet/plugin.go` (`fetchFull` calls `WrapDocument`), `plugins/silverbullet/render_test.go` (two new committed regression tests)
- **Verification:** `go test ./...` (silverbullet module); live `GET /api/items/{id}/content` confirmed the full styled document is returned, sanitized fragment unchanged in `<body>`.
- **Committed in:** `26e24b6`

---

**Total deviations:** 6 auto-fixed (1 Rule 2, 5 Rule 1)
**Impact on plan:** All six were necessary for correctness against the real deployment this plan is explicitly scoped to verify against (Task 1 Step 0's live-instance check, the plan's own human-check step, and the tracer feedback gate's human UAT). No scope creep beyond what live verification demanded — the `ca_cert`/constructor-signature change is the only one that touches a declared interface, and it's additive (existing 2-arg call sites elsewhere in the plan's own action text still compile as written, since the plan itself never called these constructors from outside `main.go`/tests). Deviations 5-6 are UI-only fixes (layout container + a server-generated stylesheet); no plugin contract, index schema, or sync-engine behavior changed.

## Issues Encountered

None beyond the four deviations above — all were found and fixed within the same session via live verification against the user's real SilverBullet instance, which the plan's own Task 1 Step 0 and human-check step already called for.

## User Setup Required

None beyond what the plan's `user_setup` block already specified (`SILVERBULLET_URL`/`SB_AUTH_TOKEN` — resolved live as `SB_URL`/`SB_AUTH_TOKEN` in this deployment's actual `.env`). Additionally, for this specific self-signed-TLS deployment: the pinned CA certificate now lives at `~/.config/webspaces/silverbullet-ca.pem`, and `~/.config/webspaces/config.toml` has a `[sources.silverbullet]` block (`base_url`, `token`, `ca_cert`) alongside the existing `[sources.paperless]` block. Both are outside the git repo (personal deployment config) and were wired live during this plan's execution to support the human-check step.

## Threat Flags

| Flag | File | Description |
|------|------|-------------|
| threat_flag: new-trust-input | `plugins/silverbullet/client.go`, `kernel/config/types.go` | The new `ca_cert` config field reads an admin-configured local file path (never user/network input) and loads it into the HTTP client's `tls.Config.RootCAs`, replacing (not appending to) the system trust store for that one client. This is a deliberate, narrow addition (pin one CA, never `InsecureSkipVerify`) not present in the plan's original `<threat_model>`. Risk is low: the path comes from the same trusted local config file that already carries the plaintext-adjacent `${SB_AUTH_TOKEN}` reference, and an unreadable/unparsable file falls back to the system trust store rather than failing open or panicking. Not itself a mitigation of an existing threat ID — flagged for completeness since it's new surface, not because it weakens an existing guarantee. |

## Next Phase Readiness

- `kernel/correlate.Engine.SyncSource`/`kernel/index.Store.ReplaceWebspaceSourceItems` are the write path every later-phase plugin (IMAP, Signal, WhatsApp) now inherits — no further architectural change needed for a third/fourth/fifth source to persist correctly.
- `plugins/silverbullet` is a complete reference for the "second source" shape (as `plugins/paperless` was for the first), useful for 02-02/02-03/02-04's coordinator/health/agent-API work and for the later PLUG-05 mock-plugin validation exercise.
- The `GET /api/sources` health-merge endpoint (RESEARCH.md, a later plan in this phase) can replace `sourceDisplayName()`'s local mapping with a live, plugin-reported `display_name` without changing `DetailPane.svelte`'s call site shape.
- **Outstanding before this plan's tracer gate is fully signed off:** the two detail-pane fixes (`36555ec`, `26e24b6`) need a human visual re-verification pass — they were verified at the API/markup level in this session but no browser tooling was available to capture a confirming screenshot. Port 7777 was left free and both plugin subprocesses cleaned up at the end of this session.
- No other blockers identified for 02-02 (refresh/health/coordinator) or 02-03/02-04 (filter UI, agent permissions).

---
*Phase: 02-two-sources-one-trustworthy-stream*
*Completed: 2026-07-28*

## Self-Check: PASSED

All claimed files exist on disk and all five commits (`f1bec9a`, `955028a`, `5615ecf`, `36555ec`, `26e24b6`) are present in `git log`.
