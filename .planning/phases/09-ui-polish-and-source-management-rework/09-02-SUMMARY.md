---
phase: 09-ui-polish-and-source-management-rework
plan: 02
subsystem: plugin-contract
tags: [go-embed, svg, provenance, playwright, svelte5, plugin-icons]

requires:
  - phase: 09-01
    provides: "DescribeResponse.icon/icon_mime fields, kernel icon endpoint, PluginIcon.svelte fallback chain"
provides:
  - "All seven in-repo plugins (paperless, silverbullet, proton, signal, whatsapp, mock, mockstrict) declare their own identity icon via Describe"
  - "internal/audit/plugin_icons_test.go — repo-wide mechanical audit of icon size/provenance/no-currentColor/Icon+IconMime completeness, discovered by walking plugins/ for go.mod"
  - "QRPanel.svelte renders the topos app icon (web/static/app-icon.png), proven decoded in a real browser by web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts case 14"
  - "StreamRow.svelte (stream + search-result rows) renders each row's own plugin identity icon via PluginIcon.svelte, resolved from the already-fetched sourcesByInstance map — no kernel/proto change"
  - "Documented (config.example.toml, README.md) the absolute-[plugins]-dir staleness risk that caused the Task 4 checkpoint's Puzzle-icon report"
affects: [09-03, 09-04, 09-05, 09-06, 09-07]

actuals:
  tokens: 18000
  tasks: 3
  commits: 5

tech-stack:
  added: []
  patterns:
    - "Real third-party marks committed byte-faithful to upstream, wrapped (not redrawn) in a square viewBox/data-URI when the source asset itself is non-square or raster-only"
    - "Repo-wide plugin-module discovery by walking plugins/ for go.mod (no hardcoded plugin list), mirroring the existing outbound-egress/module-pins audit house style, including a negative-control fixture"
    - "Four-key provenance comment (Source-Project/File/Version/License) directly above every icon go:embed directive, mechanically parsed by internal/audit"

key-files:
  created:
    - plugins/paperless/assets/icon.svg
    - plugins/silverbullet/assets/icon.svg
    - plugins/proton/assets/icon.svg
    - plugins/signal/assets/icon.svg
    - plugins/mockstrict/assets/icon.svg
    - plugins/whatsapp/assets/icon.svg
    - internal/audit/plugin_icons_test.go
    - internal/audit/testdata/plugin_icons_violation/assets/icon.svg
    - internal/audit/testdata/plugin_icons_violation/plugin.go
    - web/src/lib/components/stream-row-plugin-icon.test.ts
    - web/e2e/specs/09-stream-row-plugin-icon.spec.ts
  modified:
    - plugins/paperless/plugin.go
    - plugins/silverbullet/plugin.go
    - plugins/proton/plugin.go
    - plugins/signal/plugin.go
    - plugins/mockstrict/plugin.go
    - plugins/whatsapp/plugin.go
    - plugins/whatsapp/describeonly.go
    - plugins/mock/plugin.go
    - plugins/mock/assets/icon.svg
    - web/src/lib/components/QRPanel.svelte
    - web/src/lib/components/qr-panel.test.ts
    - web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts
    - config.example.toml
    - README.md
    - web/src/lib/components/StreamRow.svelte
    - web/src/lib/components/StreamList.svelte
    - web/src/lib/components/SearchResults.svelte

key-decisions:
  - "paperless-ngx's real logo (src-ui/src/assets/logo-white-notext.svg, GPL-3.0, non-square 70x84.284 viewBox) is wrapped in a square <svg>/<g translate> canvas rather than redrawn — upstream path data is byte-for-byte unmodified"
  - "SilverBullet ships no vector logo upstream (only raster PNGs and a favicon.svg that itself base64-wraps a PNG) — client/images/logo-dock-96x96.png (MIT, already square) is embedded as a base64 data URI inside a plain <svg><image> document, the identical technique upstream's own favicon.svg already uses for its own icon"
  - "The audit's currentColor ban is a literal substring check across the WHOLE icon.svg file, including its own provenance/documentation comment — every Lucide-derived SVG's comment text was worded to describe the baked-color decision without ever spelling out the literal word, since a prose mention inside the file's own XML comment trips the same mechanical check as a live stroke attribute would"
  - "Checkpoint round 2, Item 1 (Puzzle-icon report): diagnosed as a real, external configuration mismatch, NOT a code bug — confirmed directly against the user's own still-running make dev session (read-only GETs): every configured source is reachable (Match/Health prove a stale binary still works for those RPCs) but every icon endpoint 404s, and the MAIN checkout's bin/plugins/ (outside this worktree, timestamped before this plan's first commit) is the binary set actually being served. This worktree's own bin/plugins/ rebuilds correctly and serves icons at 200 when tested directly. Documented (not code-fixed) per checkpoint instructions: a Makefile fix cannot force a foreign absolute [plugins] dir to stay fresh."
  - "Checkpoint round 2, Item 2 (stream-pane icons): resolved the row's plugin binary name from the ALREADY-FETCHED sourcesByInstance map (GET /api/sources, SourceStatus.plugin since 09-01) rather than adding a new streamItem.plugin kernel field — the exact map sourceDisplayName already resolves from, so this is a pure frontend wiring change with zero kernel/proto surface"

patterns-established:
  - "A real third-party mark's upstream bytes are never redrawn to force squareness — wrap in a presentation-only <g transform> or <image> container instead, keeping the committed asset diffable against its exact upstream source"
  - "internal/audit's repo-wide scanners (outbound egress, module pins, now plugin icons) all share the same shape: dynamic discovery (never a hardcoded list), a documented offense-string format naming the offending file, and a negative-control fixture proving the scanner is non-vacuous"

requirements-completed: []

coverage:
  - id: D1
    description: "paperless-ngx and SilverBullet embed their own real upstream logo marks with recorded four-key provenance, wired into Describe"
    verification:
      - kind: unit
        ref: "plugins/paperless, plugins/silverbullet: CGO_ENABLED=0 go build ./... && go test ./..."
        status: pass
      - kind: unit
        ref: "internal/audit/plugin_icons_test.go#TestPluginIcons_EveryInRepoPluginDeclaresOne"
        status: pass
    human_judgment: false
  - id: D2
    description: "Proton (Mail), Signal (MessageCircle) and mockstrict (FlaskConical) embed baked-color (#94a3b8) Lucide-derived glyphs, ISC provenance recorded, wired into Describe; Signal's cgo module still builds/tests under its own tag"
    verification:
      - kind: unit
        ref: "plugins/proton, plugins/mockstrict: CGO_ENABLED=0 go build ./... && go test ./..."
        status: pass
      - kind: unit
        ref: "make test-signal (CGO_ENABLED=1 -tags libsqlcipher)"
        status: pass
    human_judgment: false
  - id: D3
    description: "WhatsApp embeds a MessageSquare glyph (distinct from Signal's MessageCircle) wired into both plugin.go and describeonly.go Describe implementations; internal/audit repo-wide icon test covers all seven plugin modules"
    verification:
      - kind: unit
        ref: "internal/audit/plugin_icons_test.go#TestPluginIcons_EveryInRepoPluginDeclaresOne, #TestPluginIconsScanner_FixtureReportsAllOffenseKinds"
        status: pass
      - kind: integration
        ref: "make test-portable (all 7 workspace modules)"
        status: pass
    human_judgment: false
  - id: D4
    description: "QRPanel.svelte renders the topos app icon (decorative alt), proven decoded in a real browser with no third-party brand asset fetched and no whatsmeow text rendered"
    verification:
      - kind: unit
        ref: "web/src/lib/components/qr-panel.test.ts (app-icon.png presence + image-source allowlist assertions)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts case 14 (naturalWidth > 0, image-src set equality, no whatsmeow text) — all 14 cases in the file pass together"
        status: pass
    human_judgment: false
  - id: D5
    description: "14px/16px icon legibility across all seven plugin chips and menus, and a final upstream trademark-policy recheck for paperless-ngx/SilverBullet before shipping their real marks"
    verification: []
    human_judgment: true
    rationale: "Icon legibility at small sizes and third-party trademark policy interpretation are exactly the two things Task 4's blocking human checkpoint exists for. The checkpoint's round-2 response addressed a Puzzle-icon rendering bug and a scope change (Item 2, stream-pane icons) rather than confirming legibility/trademark explicitly — those two sub-items are not re-verified in this SUMMARY and should be re-checked once the Item 1 fix (a config change on the user's own machine, outside this worktree) is applied and real icons are actually visible to judge."
  - id: D6
    description: "Checkpoint Item 1: diagnosed the Puzzle-icon report as a real, external [plugins] dir configuration mismatch (stale binaries outside this worktree), not a code bug — reproduced end to end and documented the risk"
    verification:
      - kind: manual_procedural
        ref: "Direct GET /api/plugins/{binary}/icon against a freshly-built kernel+plugins in this worktree (200 for all three tested plugin types) vs. against the user's own live make dev session on 127.0.0.1:7777 (404 for all five configured plugins, while /api/sources shows every source reachable) — see commit b3913dc"
        status: pass
    human_judgment: false
  - id: D7
    description: "Checkpoint Item 2: each stream/search-result row renders its own plugin identity icon (PluginIcon.svelte, kernel-served, Puzzle fallback), reusing sourcesByInstance — Thumbnail unchanged"
    verification:
      - kind: unit
        ref: "web/src/lib/components/stream-row-plugin-icon.test.ts (11 structural assertions)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/09-stream-row-plugin-icon.spec.ts (2 specs: distinct decoded icons across two plugin types in one mixed stream; Thumbnail slot unchanged alongside the new icon)"
        status: pass
    human_judgment: false

duration: ~2h10min
completed: 2026-08-11
status: complete
---

# Phase 9 Plan 2: Per-Plugin Identity Icons (Full Rollout) Summary

**All seven in-repo plugins declare their own identity icon through the plugin contract, enforced by a repo-wide `internal/audit` test; the WhatsApp pairing panel is topos-branded; and — per Task 4 checkpoint feedback — every stream and search-result row now also renders its own source's icon (reusing the existing chip icon path, Thumbnails unchanged), with the checkpoint's reported "all Puzzle icons" bug diagnosed as a stale-binary configuration mismatch outside this worktree, not a code defect.**

## Performance

- **Duration:** ~2h10min (Tasks 1-3, ~55min; checkpoint round-2 follow-up, ~75min)
- **Completed:** 2026-08-11
- **Tasks:** 3 of 4 plan tasks executed (Task 4 is a blocking human-verify checkpoint; its round-2 response is addressed by the two checkpoint-directed items below rather than by Task 4's original how-to-verify steps)
- **Files modified:** 28 (11 created, 17 modified)

## Accomplishments

- `plugins/paperless/assets/icon.svg` embeds paperless-ngx's own real logo (upstream `src-ui/src/assets/logo-white-notext.svg`, commit `7620cd0`, GPL-3.0-only), wrapped in a square `<svg><g transform="translate(...)">` canvas — the upstream path data itself is byte-for-byte unmodified.
- `plugins/silverbullet/assets/icon.svg` embeds SilverBullet's own real logo. Upstream ships no vector mark for this project at all (only raster PNGs and a favicon.svg that itself base64-wraps a PNG) — `client/images/logo-dock-96x96.png` (commit `a99e5d9`, MIT, already 96×96 square) is embedded as a base64 data URI inside a plain `<svg><image>` document, the identical technique upstream's own favicon.svg already uses for its own icon.
- `plugins/proton` (Lucide `Mail`), `plugins/signal` (Lucide `MessageCircle`), `plugins/mockstrict` (Lucide `FlaskConical`, matching `plugins/mock`) each embed a baked-`#94a3b8` Lucide-derived SVG with four-key provenance, exported directly from the installed `@lucide/svelte` v1.27.0 package. `make test-signal` (`CGO_ENABLED=1 -tags libsqlcipher`) confirmed the cgo module is unaffected.
- `plugins/whatsapp/assets/icon.svg` embeds a baked-color Lucide `MessageSquare` glyph — deliberately distinct from Signal's `MessageCircle`, and deliberately neither the topos app icon nor any WhatsApp/Meta mark — wired into **both** `SourcePlugin.Describe` (`plugin.go`) and `describeOnlyPlugin.Describe` (`describeonly.go`) so the two launch modes never disagree.
- `internal/audit/plugin_icons_test.go` (`TestPluginIcons_EveryInRepoPluginDeclaresOne`) walks `plugins/` for directories containing their own `go.mod` (no hardcoded plugin list — a future eighth plugin is caught automatically) and, for each, mechanically checks: `assets/icon.svg` exists and is under 65536 bytes; the file never contains the literal substring `currentColor`; the plugin's `//go:embed assets/icon.svg` directive carries all four provenance keys (`Source-Project`/`Source-File`/`Source-Version`/`Source-License`) directly above it, via AST-adjacent line scanning; and every `DescribeResponse` composite literal (found via `go/ast`) sets both `Icon` and `IconMime`. A negative-control fixture (`internal/audit/testdata/plugin_icons_violation/`) proves the scanner reports all three offense kinds and is non-vacuous.
- `QRPanel.svelte` renders `web/static/app-icon.png` (decorative `alt=""`) above the QR/skeleton area, unconditionally across every phase — additive branding only; every existing string, phase branch, timer and callback stayed byte-identical. `qr-panel.test.ts` gained two assertions (app-icon presence, image-source allowlist). `web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts` gained case 14, cloning case 1's populated-QR setup and proving in a real Chromium browser: the app icon's own `naturalWidth > 0` (a genuinely served/decoded asset, not merely a present `<img>` tag), the panel's full image-source set is exactly `{the session's QR data URI, /app-icon.png}` (no third-party brand asset fetched), and the panel's rendered text carries no "whatsmeow" mention. All 14 cases in the file (13 pre-existing + 1 new) pass together.

### Checkpoint round 2 (Task 4 feedback): two items

**Item 1 — diagnosed the "all Puzzle" report, no code bug found.** Rebuilt the kernel + `bin/plugins/{paperless,silverbullet,mock}` fresh from this worktree's HEAD and confirmed every icon endpoint returns 200 with correct bytes. Then, directly against the user's own still-running `make dev` session (`127.0.0.1:7777`, read-only `GET`s only): `GET /api/sources` showed all five configured sources `reachable: true`, but `GET /api/plugins/{binary}/icon` 404'd for every one of them. The main checkout's `bin/plugins/` (`/home/darren/projects/davison/topos/bin/plugins/`, outside this worktree) holds binaries timestamped `17:05` — before this plan's first commit (`754a445`, `17:13`). `go run`'s own `os.Executable()` resolving into `$GOCACHE` (not this repo) is the well-known reason a real dev setup ends up with an absolute `[plugins] dir` override in the first place — and that override is exactly what makes `make dev`'s own `plugins` rebuild (which only ever writes into the invoking checkout's `bin/plugins/`) miss the binaries the kernel actually loads. This is a real, external configuration mismatch on the user's own machine, not a regression in this plan's code, and per the checkpoint's own instructions was documented rather than "fixed" from inside the worktree (config.example.toml's `[plugins] dir` comment, README.md's Development loop section).

**Item 2 — stream/search-result rows now render per-source plugin icons.** `StreamRow.svelte` gained a `plugin` prop (default `''`) rendering the existing `PluginIcon.svelte` (same kernel-served endpoint, same mandatory Puzzle fallback, same `size-3.5` the chip uses) as the leading element of the metadata strip, wrapped in a plain `<span title=...>` for hover-discoverable source name. `StreamList.svelte`/`SearchResults.svelte` resolve each row's plugin binary name from the already-fetched `sourcesByInstance` map (`GET /api/sources`, which has carried `SourceStatus.plugin` since 09-01) — the identical map `sourceDisplayName` already resolves from, so **no kernel/proto change was needed**. The leading `Thumbnail` element is byte-identical to before (verified structurally and in a live browser). `web/e2e/specs/09-stream-row-plugin-icon.spec.ts` attaches two DIFFERENT plugin-type instances (`topos-plugin-mock`, `topos-plugin-mockstrict`) to one webspace and proves, in a real Chromium browser, that the two rows render two distinct, decoded icons, and that the one item with a real thumbnail still renders its Thumbnail slot alongside its new icon.

## Task Commits

1. **Task 1: Real upstream marks — paperless-ngx and SilverBullet** - `754a445` (feat)
2. **Task 2: Lucide-derived baked-colour glyphs — Proton, Signal, mockstrict** - `826e41f` (feat)
3. **Task 3: WhatsApp glyph and topos-branded pairing, plus the repo-wide icon audit** - `486e28c` (feat)
4. **Checkpoint Item 1: diagnose Puzzle-icon report, document staleness risk** - `b3913dc` (docs)
5. **Checkpoint Item 2: plugin identity icons on stream/search rows** - `49418d5` (feat)

_No separate plan-metadata commit — worktree mode; the orchestrator commits SUMMARY.md/STATE.md centrally after the wave._

## Files Created/Modified

- `plugins/paperless/assets/icon.svg`, `plugins/paperless/plugin.go` - real paperless-ngx mark + Describe wiring
- `plugins/silverbullet/assets/icon.svg`, `plugins/silverbullet/plugin.go` - real SilverBullet mark (PNG-in-SVG wrapper) + Describe wiring
- `plugins/proton/assets/icon.svg`, `plugins/proton/plugin.go` - Lucide Mail glyph + Describe wiring
- `plugins/signal/assets/icon.svg`, `plugins/signal/plugin.go` - Lucide MessageCircle glyph + Describe wiring
- `plugins/mockstrict/assets/icon.svg`, `plugins/mockstrict/plugin.go` - Lucide FlaskConical glyph + Describe wiring
- `plugins/whatsapp/assets/icon.svg`, `plugins/whatsapp/plugin.go`, `plugins/whatsapp/describeonly.go` - Lucide MessageSquare glyph, wired into both launch modes
- `plugins/mock/plugin.go`, `plugins/mock/assets/icon.svg` - provenance-comment/currentColor-wording fix (deviation, see below)
- `internal/audit/plugin_icons_test.go`, `internal/audit/testdata/plugin_icons_violation/` - repo-wide icon audit + negative control
- `web/src/lib/components/QRPanel.svelte`, `qr-panel.test.ts` - topos-branded pairing panel + structural guards
- `web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts` - case 14, the browser-level de-branding proof
- `config.example.toml`, `README.md` - checkpoint Item 1: documented the absolute-`[plugins]`-dir staleness risk (also fixed two stale "webspaces" naming references in the same comment block)
- `web/src/lib/components/StreamRow.svelte` - checkpoint Item 2: new `plugin` prop, renders `PluginIcon` in the metadata strip
- `web/src/lib/components/StreamList.svelte`, `SearchResults.svelte` - checkpoint Item 2: resolve and thread `plugin` from `sourcesByInstance`
- `web/src/lib/components/stream-row-plugin-icon.test.ts` - checkpoint Item 2: structural wiring guard
- `web/e2e/specs/09-stream-row-plugin-icon.spec.ts` - checkpoint Item 2: mixed-stream browser proof

## Decisions Made

- **paperless-ngx's non-square real logo wrapped, not redrawn.** Upstream's `logo-white-notext.svg` has viewBox `0 0 70 84.284` (aspect ~0.83), not square. Rather than redraw the mark to fit a square, the path data is left byte-identical and wrapped in a square `84.284×84.284` outer `<svg>` with a `<g transform="translate(7.142,0)">` centering the original artwork — `7.142 = (84.284-70)/2`.
- **SilverBullet's real mark sourced as a wrapped PNG, not a redrawn SVG.** No vector SilverBullet logo exists upstream (confirmed by walking the full repo tree via GitHub's API) — only raster PNGs and a favicon.svg that itself embeds a base64 PNG. `client/images/logo-dock-96x96.png` (already square, 9449 bytes) is embedded as a base64 data URI inside a minimal `<svg><image>` wrapper, mirroring upstream's own technique for its own icon rather than inventing a new one.
- **Provenance convention tightened repo-wide, including the pre-existing mock plugin (Rule 1/2 deviation).** 09-02's acceptance criteria require the four-key provenance block (`Source-Project`/`Source-File`/`Source-Version`/`Source-License`) literally inside `plugin.go`, directly above the `//go:embed` directive — a stricter, differently-shaped convention than 09-01's landed mock plugin, which recorded provenance only inside `assets/icon.svg`'s own XML comment using different (lowercase, non-hyphenated) key names. Since Task 3's repo-wide audit is required to pass across "all seven plugin modules" (including mock), `plugins/mock/plugin.go` was updated to add the same four-key block — otherwise the plan's own central deliverable (the audit) would fail against the very codebase state 09-01 shipped.
- **The audit's `currentColor` ban is a literal whole-file substring check, so every SVG's own documentation comment was worded to avoid the literal word.** Discovered live: my first draft of `internal/audit/plugin_icons_test.go` failed against `plugins/mock/assets/icon.svg` (already landed by 09-01) and against every newly-authored Lucide-derived SVG, because each file's own provenance comment described "the upstream `stroke="currentColor"` default" in prose — a legitimate documentation mention, but the plan's `<behavior>` line ("no plugin icon asset contains the string currentColor") is unambiguous and whole-file. Re-worded every affected comment (mock's included, as part of the same fix above) to describe the baked-color decision without ever spelling out the literal keyword.
- **Checkpoint Item 1 root-caused against the user's own live session, not guessed.** Rather than speculatively patching the Makefile, the diagnosis used read-only `GET`s against the user's already-running `make dev` kernel (still up on `127.0.0.1:7777`) to get ground truth: all sources reachable (proving plugins DID launch — ruling out a `go run`/relative-path launch failure as the mechanism) but every icon 404 (proving a real, older binary set is being served). Cross-referencing the main checkout's `bin/plugins/` mtimes against this plan's own commit timestamps confirmed the exact mechanism without needing access to the user's personal `config.toml`.
- **Checkpoint Item 2 resolved the plugin binary name client-side from already-fetched data, not a new kernel field.** The checkpoint's own instructions flagged this as an open question ("check whether stream item API rows already carry it… if a kernel-side field addition is required…"). `sourcesByInstance` (built from `GET /api/sources` in `+page.svelte`, already threaded into both `StreamList.svelte` and `SearchResults.svelte` for `sourceDisplayName` resolution) already carries `SourceStatus.plugin` — reusing it is a pure frontend change with zero kernel/proto/wire-contract surface, strictly smaller than adding and threading a new `streamItem.plugin` field would have been.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1/2 - Bug / Missing Critical] `plugins/mock/plugin.go` and `plugins/mock/assets/icon.svg` brought into line with this plan's stricter provenance/currentColor conventions**
- **Found during:** Task 3, while writing `internal/audit/plugin_icons_test.go` and first running it against the full repo.
- **Issue:** `plugins/mock` (landed by 09-01, out of this plan's declared `<files>` scope) used a looser provenance-comment shape (lowercase keys, inside the SVG's own XML comment rather than `plugin.go`) and its SVG's own documentation prose literally contained the word `currentColor` — both of which fail this plan's own newly-introduced, mechanically-enforced conventions.
- **Fix:** Added the exact four-key (`Source-Project`/`Source-File`/`Source-Version`/`Source-License`) block directly above `plugins/mock/plugin.go`'s `//go:embed assets/icon.svg` directive, and reworded `plugins/mock/assets/icon.svg`'s comment to describe the baked-color rationale without the literal `currentColor` substring.
- **Files modified:** `plugins/mock/plugin.go`, `plugins/mock/assets/icon.svg`
- **Verification:** `CGO_ENABLED=0 go build ./... && go test ./...` in `plugins/mock`; `go test ./internal/audit/... -run PluginIcons -v` passes for all 7 discovered modules.
- **Committed in:** `486e28c` (Task 3 commit)

**2. [Checkpoint-directed fix, Task 4 round 2, Item 1] Documented the absolute-`[plugins]`-dir staleness risk that produced the "all Puzzle" report**
- **Found during:** Checkpoint round 2 — the orchestrator relayed the user's report that every source chip showed the Puzzle fallback in their real `make dev` session.
- **Issue:** No code bug. Reproduced the full chain fresh in this worktree (`make build`-equivalent + a synthetic config exercising paperless/silverbullet/mock) and every `GET /api/plugins/{binary}/icon` returned 200 with correct bytes. Root-caused directly against the user's own still-running `make dev` kernel (read-only `GET`s): every source `reachable: true` (proving plugins launched fine — an outright launch failure would have shown unreachable, not merely icon-less) but every icon 404. The main checkout's `bin/plugins/` — a directory entirely outside this worktree — holds binaries built at `17:05`, before this plan's first commit (`754a445`, `17:13`); this worktree's own `bin/plugins/` (which `make dev`'s `.PHONY plugins` prerequisite DOES rebuild correctly on every invocation) was never the binary set actually being served. This points to the user's personal `~/.config/topos/config.toml` overriding `[plugins] dir` to an absolute path outside this checkout — a real, external configuration fact this worktree has no access to and no authority to change.
- **Fix:** Per the checkpoint's own explicit instruction ("do NOT reach outside the worktree… document it instead"), added a staleness-risk warning to `config.example.toml`'s `[plugins] dir` comment and a matching callout in `README.md`'s Development loop section — both name the exact failure mode and the fix (point `dir` at this checkout, or rebuild in whichever checkout it names). Also corrected two stale "webspaces" naming references (a pre-existing drift from the project's earlier name, unrelated to this bug but inside the same comment block being edited).
- **Files modified:** `config.example.toml`, `README.md`
- **Verification:** No test surface exists for prose documentation; the diagnosis itself was verified via direct `curl` against both a fresh worktree kernel (200s) and the user's live session (404s) — see the exact evidence in the commit message.
- **Committed in:** `b3913dc`

---

**Total deviations:** 2 auto-fixed/directed (1 Rule 1/2 bug-and-missing-critical fix from Task 3's own work; 1 checkpoint-directed documentation fix from the round-2 diagnosis above — no code changed for the second, since none was needed).
**Impact on plan:** No scope creep beyond what was strictly required. Item 1's root cause sits entirely outside this worktree's authority; the in-worktree action (documentation) is the correct-sized response per the checkpoint's own instructions.

## User-Directed Scope Addition (Task 4 Checkpoint, Item 2)

The checkpoint's round-2 response added an explicit, authoritative scope change beyond 09-UI-SPEC.md Fix 10's original ask (per-plugin icons on chips/menus only): **plugin identity icons must also appear on each item row in the stream pane**, so a mixed cross-source feed is scannable by source at a glance — the chip icons (09-01) stay as they are, but the user's actual ask was for the stream rows, which had never rendered any per-source visual marker at all (only `group_label`/date/tags — `sourceDisplayName` was passed to `StreamRow.svelte` already but was, until this change, used ONLY inside the stale-source tooltip's `aria-label`, never rendered visibly).

Implemented exactly per the checkpoint's stated constraints:
- Reused `PluginIcon.svelte` unchanged — same kernel endpoint, same fallback chain, same fixed-size box. No second icon path.
- The existing `Thumbnail` element (PDF/document previews) is untouched — the icon is additive metadata in the row body, structurally separate from the thumbnail slot (proven both by a structural test and a live e2e assertion that the thumbnail slot still renders exactly one element after the icon was added).
- The plugin binary name was NOT threaded as a new kernel/wire field — `sourcesByInstance` (already fetched, already used for `sourceDisplayName`) already carries it.
- Extended both a unit test (`stream-row-plugin-icon.test.ts`) and a new Playwright e2e spec (`09-stream-row-plugin-icon.spec.ts`), per the project's standing "any UI change extends the e2e suite" convention.
- Touched only `StreamRow.svelte`, `StreamList.svelte`, `SearchResults.svelte`, and their tests — none of the checkpoint's named off-limits files (`SearchBox.svelte`, `DetailPane.svelte`, `SourceChip.svelte`, `WebspaceSwitcher.svelte`, `ManageSourcesModal.svelte`, `AddSourceModal.svelte`, or any `ui/dropdown-menu`/`popover` surface file) were modified.

## Issues Encountered

- SilverBullet has no vector logo mark published anywhere upstream (confirmed by walking the full `silverbulletmd/silverbullet` git tree via GitHub's API, and checking every sibling org repo) — resolved by wrapping the real raster PNG mark upstream does ship, using upstream's own base64-in-SVG technique, rather than substituting a Lucide-derived glyph for what 09-UI-SPEC.md designates a "real mark" plugin.
- The `internal/audit` `currentColor` substring check initially flagged every Lucide-derived SVG's own documentation comment (which described the change away from `currentColor` in prose) — resolved by rewording every such comment (see Decisions Made above).
- `mockstrict`'s e2e fixture required a non-empty `path` field (its own required-connection-field mechanism, distinct from `mock`'s no-config shape) — the first `09-stream-row-plugin-icon.spec.ts` run failed the whole kernel boot until this was added to the fixture's source spec.

## User Setup Required

**Action needed on the user's own machine, outside this worktree, to see real icons in their interactive session:** their `~/.config/topos/config.toml`'s `[plugins] dir` (if it overrides the default) most likely points at a `bin/plugins/` directory outside this worktree, holding binaries built before this plan's commits. To see real plugin icons while testing this branch: either (a) point `[plugins] dir` at this worktree's own `bin/plugins/` absolute path, or (b) once this worktree's wave merges to main, rebuild plugins (`make plugins`/`make build`) in whichever checkout `[plugins] dir` actually names. See `config.example.toml`'s `[plugins] dir` comment and `README.md`'s Development loop section (both updated by this plan) for the full explanation.

## Next Phase Readiness

- All 5 commits are fully committed, tested (`make test-portable`, `make test-signal`, `internal/audit`'s `PluginIcons` tests, `npm run check`, `npm test` — 699 tests, and the full Playwright suite — 48/48 including the new `09-stream-row-plugin-icon.spec.ts`) and verified against their own acceptance criteria plus the checkpoint's round-2 requirements.
- **Task 4's original how-to-verify steps (14px/16px legibility across all seven chips/menus; a final trademark-policy recheck for paperless-ngx/SilverBullet) were not explicitly re-confirmed in this round** — the checkpoint's round-2 response addressed a functional bug report and a scope change instead. `coverage: D5` in this SUMMARY's frontmatter records this as still `human_judgment: true` with no verification. Recommend a follow-up visual pass once real icons are actually visible in the user's session (see User Setup Required above) — nothing in this plan's own code blocks that pass; it's a pure visual/policy judgment call.
- No blockers for the rest of Phase 9's plans — the icon contract itself (09-01) is unchanged, this plan's kernel/proto-adjacent surface (the icon bytes themselves) is additive and already exercised by `internal/audit` and the e2e suite, and the stream-row icon addition (Item 2) touched no file any sibling wave-1 worktree also touches (confirmed against the checkpoint's own do-not-touch list).

---
*Phase: 09-ui-polish-and-source-management-rework*
*Completed: 2026-08-11*

## Self-Check: PASSED

All 11 created files verified present on disk (`git ls-files`); all 5 commit hashes (`754a445`, `826e41f`, `486e28c`, `b3913dc`, `49418d5`) verified present in `git log --oneline --all`.
