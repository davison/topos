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
affects: [09-03, 09-04, 09-05, 09-06, 09-07]

actuals:
  tokens: 13100
  tasks: 3
  commits: 3

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

key-decisions:
  - "paperless-ngx's real logo (src-ui/src/assets/logo-white-notext.svg, GPL-3.0, non-square 70x84.284 viewBox) is wrapped in a square <svg>/<g translate> canvas rather than redrawn — upstream path data is byte-for-byte unmodified"
  - "SilverBullet ships no vector logo upstream (only raster PNGs and a favicon.svg that itself base64-wraps a PNG) — client/images/logo-dock-96x96.png (MIT, already square) is embedded as a base64 data URI inside a plain <svg><image> document, the identical technique upstream's own favicon.svg already uses for its own icon"
  - "The audit's currentColor ban is a literal substring check across the WHOLE icon.svg file, including its own provenance/documentation comment — every Lucide-derived SVG's comment text was worded to describe the baked-color decision without ever spelling out the literal word, since a prose mention inside the file's own XML comment trips the same mechanical check as a live stroke attribute would"

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
    rationale: "Icon legibility at small sizes and third-party trademark policy interpretation are exactly the two things Task 4's blocking human checkpoint exists for — this executor stopped at that checkpoint per the plan's autonomous: false frontmatter and did not proceed past it."

duration: ~55min
completed: 2026-08-11
status: halted
---

# Phase 9 Plan 2: Per-Plugin Identity Icons (Full Rollout) Summary

**All seven in-repo plugins (paperless-ngx, SilverBullet, Proton, Signal, WhatsApp, mock, mockstrict) now declare their own identity icon through the plugin contract — two real upstream marks, five baked-color Lucide-derived glyphs — enforced by a new repo-wide `internal/audit` test, with the WhatsApp pairing panel topos-branded and proven decoded in a real browser. Halted at Task 4's blocking human checkpoint (icon legibility + trademark recheck).**

## Performance

- **Duration:** ~55min
- **Completed:** 2026-08-11 (Tasks 1-3; Task 4 checkpoint pending)
- **Tasks:** 3 of 4 (Task 4 is a blocking human-verify checkpoint, not executed)
- **Files modified:** 21 (9 created, 12 modified)

## Accomplishments

- `plugins/paperless/assets/icon.svg` embeds paperless-ngx's own real logo (upstream `src-ui/src/assets/logo-white-notext.svg`, commit `7620cd0`, GPL-3.0-only), wrapped in a square `<svg><g transform="translate(...)">` canvas — the upstream path data itself is byte-for-byte unmodified.
- `plugins/silverbullet/assets/icon.svg` embeds SilverBullet's own real logo. Upstream ships no vector mark for this project at all (only raster PNGs and a favicon.svg that itself base64-wraps a PNG) — `client/images/logo-dock-96x96.png` (commit `a99e5d9`, MIT, already 96×96 square) is embedded as a base64 data URI inside a plain `<svg><image>` document, the identical technique upstream's own favicon.svg already uses for its own icon.
- `plugins/proton` (Lucide `Mail`), `plugins/signal` (Lucide `MessageCircle`), `plugins/mockstrict` (Lucide `FlaskConical`, matching `plugins/mock`) each embed a baked-`#94a3b8` Lucide-derived SVG with four-key provenance, exported directly from the installed `@lucide/svelte` v1.27.0 package. `make test-signal` (`CGO_ENABLED=1 -tags libsqlcipher`) confirmed the cgo module is unaffected.
- `plugins/whatsapp/assets/icon.svg` embeds a baked-color Lucide `MessageSquare` glyph — deliberately distinct from Signal's `MessageCircle`, and deliberately neither the topos app icon nor any WhatsApp/Meta mark — wired into **both** `SourcePlugin.Describe` (`plugin.go`) and `describeOnlyPlugin.Describe` (`describeonly.go`) so the two launch modes never disagree.
- `internal/audit/plugin_icons_test.go` (`TestPluginIcons_EveryInRepoPluginDeclaresOne`) walks `plugins/` for directories containing their own `go.mod` (no hardcoded plugin list — a future eighth plugin is caught automatically) and, for each, mechanically checks: `assets/icon.svg` exists and is under 65536 bytes; the file never contains the literal substring `currentColor`; the plugin's `//go:embed assets/icon.svg` directive carries all four provenance keys (`Source-Project`/`Source-File`/`Source-Version`/`Source-License`) directly above it, via AST-adjacent line scanning; and every `DescribeResponse` composite literal (found via `go/ast`) sets both `Icon` and `IconMime`. A negative-control fixture (`internal/audit/testdata/plugin_icons_violation/`) proves the scanner reports all three offense kinds and is non-vacuous.
- `QRPanel.svelte` renders `web/static/app-icon.png` (decorative `alt=""`) above the QR/skeleton area, unconditionally across every phase — additive branding only; every existing string, phase branch, timer and callback stayed byte-identical. `qr-panel.test.ts` gained two assertions (app-icon presence, image-source allowlist). `web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts` gained case 14, cloning case 1's populated-QR setup and proving in a real Chromium browser: the app icon's own `naturalWidth > 0` (a genuinely served/decoded asset, not merely a present `<img>` tag), the panel's full image-source set is exactly `{the session's QR data URI, /app-icon.png}` (no third-party brand asset fetched), and the panel's rendered text carries no "whatsmeow" mention. All 14 cases in the file (13 pre-existing + 1 new) pass together.

## Task Commits

1. **Task 1: Real upstream marks — paperless-ngx and SilverBullet** - `754a445` (feat)
2. **Task 2: Lucide-derived baked-colour glyphs — Proton, Signal, mockstrict** - `826e41f` (feat)
3. **Task 3: WhatsApp glyph and topos-branded pairing, plus the repo-wide icon audit** - `486e28c` (feat)

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

## Decisions Made

- **paperless-ngx's non-square real logo wrapped, not redrawn.** Upstream's `logo-white-notext.svg` has viewBox `0 0 70 84.284` (aspect ~0.83), not square. Rather than redraw the mark to fit a square, the path data is left byte-identical and wrapped in a square `84.284×84.284` outer `<svg>` with a `<g transform="translate(7.142,0)">` centering the original artwork — `7.142 = (84.284-70)/2`.
- **SilverBullet's real mark sourced as a wrapped PNG, not a redrawn SVG.** No vector SilverBullet logo exists upstream (confirmed by walking the full repo tree via GitHub's API) — only raster PNGs and a favicon.svg that itself embeds a base64 PNG. `client/images/logo-dock-96x96.png` (already square, 9449 bytes) is embedded as a base64 data URI inside a minimal `<svg><image>` wrapper, mirroring upstream's own technique for its own icon rather than inventing a new one.
- **Provenance convention tightened repo-wide, including the pre-existing mock plugin (Rule 1/2 deviation).** 09-02's acceptance criteria require the four-key provenance block (`Source-Project`/`Source-File`/`Source-Version`/`Source-License`) literally inside `plugin.go`, directly above the `//go:embed` directive — a stricter, differently-shaped convention than 09-01's landed mock plugin, which recorded provenance only inside `assets/icon.svg`'s own XML comment using different (lowercase, non-hyphenated) key names. Since Task 3's repo-wide audit is required to pass across "all seven plugin modules" (including mock), `plugins/mock/plugin.go` was updated to add the same four-key block — otherwise the plan's own central deliverable (the audit) would fail against the very codebase state 09-01 shipped.
- **The audit's `currentColor` ban is a literal whole-file substring check, so every SVG's own documentation comment was worded to avoid the literal word.** Discovered live: my first draft of `internal/audit/plugin_icons_test.go` failed against `plugins/mock/assets/icon.svg` (already landed by 09-01) and against every newly-authored Lucide-derived SVG, because each file's own provenance comment described "the upstream `stroke="currentColor"` default" in prose — a legitimate documentation mention, but the plan's `<behavior>` line ("no plugin icon asset contains the string currentColor") is unambiguous and whole-file. Re-worded every affected comment (mock's included, as part of the same fix above) to describe the baked-color decision without ever spelling out the literal keyword.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1/2 - Bug / Missing Critical] `plugins/mock/plugin.go` and `plugins/mock/assets/icon.svg` brought into line with this plan's stricter provenance/currentColor conventions**
- **Found during:** Task 3, while writing `internal/audit/plugin_icons_test.go` and first running it against the full repo.
- **Issue:** `plugins/mock` (landed by 09-01, out of this plan's declared `<files>` scope) used a looser provenance-comment shape (lowercase keys, inside the SVG's own XML comment rather than `plugin.go`) and its SVG's own documentation prose literally contained the word `currentColor` — both of which fail this plan's own newly-introduced, mechanically-enforced conventions.
- **Fix:** Added the exact four-key (`Source-Project`/`Source-File`/`Source-Version`/`Source-License`) block directly above `plugins/mock/plugin.go`'s `//go:embed assets/icon.svg` directive, and reworded `plugins/mock/assets/icon.svg`'s comment to describe the baked-color rationale without the literal `currentColor` substring.
- **Files modified:** `plugins/mock/plugin.go`, `plugins/mock/assets/icon.svg`
- **Verification:** `CGO_ENABLED=0 go build ./... && go test ./...` in `plugins/mock`; `go test ./internal/audit/... -run PluginIcons -v` passes for all 7 discovered modules.
- **Committed in:** `486e28c` (Task 3 commit)

---

**Total deviations:** 1 auto-fixed (spanning Rule 1 bug-fix and Rule 2 missing-critical-functionality — the fix was necessary both because the prior state was a latent bug against this plan's new mechanical check, and because without it the plan's own central deliverable, the repo-wide audit, could not pass).
**Impact on plan:** No scope creep beyond what was strictly required for the audit test (this plan's own Task 3 deliverable) to be true across the full plugin set it claims to cover. All other files touched are exactly the plan's declared `<files>` for Tasks 1-3.

## Issues Encountered

- SilverBullet has no vector logo mark published anywhere upstream (confirmed by walking the full `silverbulletmd/silverbullet` git tree via GitHub's API, and checking every sibling org repo) — resolved by wrapping the real raster PNG mark upstream does ship, using upstream's own base64-in-SVG technique, rather than substituting a Lucide-derived glyph for what 09-UI-SPEC.md designates a "real mark" plugin.
- The `internal/audit` `currentColor` substring check initially flagged every Lucide-derived SVG's own documentation comment (which described the change away from `currentColor` in prose) — resolved by rewording every such comment (see Decisions Made above).

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Tasks 1-3 are fully committed, tested (`make test-portable`, `make test-signal`, `internal/audit`, `npm run check`, `npm test`, and the full `uat-08-whatsapp-qr-link.spec.ts` e2e file including the new case 14) and verified against their own acceptance criteria.
- **Task 4 (blocking human checkpoint) was NOT executed** — this plan's frontmatter declares `autonomous: false` specifically because Task 4 requires a human to (a) visually confirm 14px/16px icon legibility across all seven plugin chips/menus, including that Signal and WhatsApp read as distinct by icon alone, and (b) do a final trademark-policy recheck for paperless-ngx and SilverBullet before either real mark ships in a release. Both are explicitly named in the plan as "things a machine genuinely cannot decide." This executor halted at that checkpoint per the standing worktree-mode checkpoint protocol rather than guessing at either judgment call.
- To resume: run `make build`, start the kernel (`make dev` or equivalent), open a webspace with multiple source types configured, and follow Task 4's `<how-to-verify>` steps in `09-02-PLAN.md`.
- No blockers for the rest of Phase 9's plans — the icon contract itself (09-01) is unchanged, and this plan's only kernel/proto-adjacent surface (the icon bytes themselves) is additive and already exercised by `internal/audit` and the e2e suite.

---
*Phase: 09-ui-polish-and-source-management-rework*
*Completed: 2026-08-11 (Tasks 1-3; Task 4 checkpoint pending human verification)*

## Self-Check: PASSED

All 9 created files verified present on disk (`git ls-files`); all 3 task commit hashes (`754a445`, `826e41f`, `486e28c`) verified present in `git log --oneline --all`.
