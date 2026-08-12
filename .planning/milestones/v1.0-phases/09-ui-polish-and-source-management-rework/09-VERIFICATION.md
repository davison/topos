---
phase: 09-ui-polish-and-source-management-rework
verified: 2026-08-11T17:14:56Z
status: passed
score: 6/6 must-haves verified
behavior_unverified: 0
overrides_applied: 0
human_verification:

  - test: "With the kernel's `[plugins] dir` pointed at freshly-built plugin binaries from this codebase (not a stale external bin/plugins/), open a webspace with at least one instance of each plugin type (paperless, SilverBullet, Proton/email, Signal, WhatsApp, mock) configured, and view the chip row, the '+' picker, the Manage Sources rows, and the stream/search row icons at their shipped sizes (14px chip/stream, 16px picker/Manage-Sources)."
    expected: "Every plugin's icon is legible and visually distinguishable from the others at 14-16px on the dark palette — the real paperless-ngx and SilverBullet marks read as their upstream logos, and the Lucide-derived Mail/MessageCircle/MessageSquare/FlaskConical glyphs read as recognizably distinct shapes, not a blur."
    why_human: "Small-size glyph legibility is a visual-perception judgment call that cannot be verified by source inspection or automated tests; this is exactly the check Task 4 of 09-02-PLAN.md's blocking human-verify checkpoint exists for. The user's first attempt was foiled by a local, external configuration issue (a stale `[plugins] dir` serving pre-icon-contract binaries, diagnosed and documented in README.md/config.example.toml by 09-02) rather than completed — see 09-02-SUMMARY.md's `coverage: D5` (`human_judgment: true`, no verification recorded)."

  - test: "Before shipping, recheck paperless-ngx's and SilverBullet's current trademark/brand-usage policies (not just their code license) for any restriction on embedding their logo mark in third-party software."
    expected: "Confirmation that no separate trademark policy prohibits topos from embedding these upstream logo assets (GPL-3.0 / MIT code license alone does not settle trademark use), or a decision to swap either mark for a generic glyph if a restrictive policy is found."
    why_human: "Trademark-policy interpretation is a legal/policy judgment, not something derivable from the repository. 09-UI-SPEC.md's own Fix 10 table flags this explicitly ('no separate restrictive brand policy is known to exist ... executor should recheck before shipping'), and 09-02-SUMMARY.md's D5 records this recheck as not yet performed."
---

# Phase 9: UI Polish and Source Management Rework Verification Report

**Phase Goal:** The v1.0 UI is release-clean — the small rough edges catalogued in TODO.md are gone, every plugin is visually identifiable by its own icon, and adding sources follows a redesigned model that separates "install a plugin" from "configure an instance" (ROADMAP.md Phase 9).
**Verified:** 2026-08-11T17:14:56Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

Merged from ROADMAP.md's 5 Success Criteria plus each plan's `must_haves.truths` (all seven plans declare `requirements: []`; no plan narrowed the roadmap's SC scope).

| # | Truth (ROADMAP Success Criterion) | Status | Evidence |
|---|---|---|---|
| 1 | SC1 — All eight named fixes land: favicon, search-clear no-jump, "ago" dedup, single "+", chip refresh in 3-dot menu, valid robots.txt, "Reload Config" at dropdown root, dropdown menus visually distinct from panes | ✓ VERIFIED | Each fix independently confirmed in source (see "Required Artifacts" and "Key Link Verification" below) and proven live in a real Chromium browser by `09-static-assets-and-surfaces.spec.ts` (3/3), `09-search-clear-and-previewer.spec.ts` (2/2), `09-chip-menu.spec.ts` (4/4), `09-switcher-reload.spec.ts` (4/4) — all re-run by this verification, all green. |
| 2 | SC2 — PDF/media previewer flexes to a bounded size with a sensible aspect ratio, text flows around it | ✓ VERIFIED | `DetailPane.svelte:219-220`: `w-full max-w-sm aspect-[3/4]`, conditionally `float-left mr-6 mb-4` when `content?.text` present, else `mx-auto`. Proven live by `09-search-clear-and-previewer.spec.ts > the media preview box is bounded at a 3:4 aspect ratio, and extracted text flows beside it` (re-run, pass). |
| 3 | SC3 — Each plugin (paperless, SilverBullet, email, Signal, WhatsApp, mock) ships its own recognizable icon used consistently in chips/menus; WhatsApp is branded as topos, not whatsmeow | ✓ VERIFIED (technical/wiring) — legibility + trademark recheck outstanding, see Human Verification | All seven in-repo plugin binaries (`paperless`, `silverbullet`, `proton`, `signal`, `whatsapp`, `mock`, `mockstrict`) declare an icon via `DescribeResponse.icon`/`icon_mime`, enforced by `internal/audit/plugin_icons_test.go` (re-run, pass). WhatsApp's glyph (`MessageSquare`) is deliberately distinct from Signal's (`MessageCircle`) and is neither the topos app icon nor any WhatsApp/Meta mark (confirmed by reading `plugins/whatsapp/assets/icon.svg`'s own provenance comment). `QRPanel.svelte:329` renders `/app-icon.png` in the pairing UI. All confirmed live by `09-plugin-icon.spec.ts` (3/3) and `uat-08-whatsapp-qr-link.spec.ts` case 14 (re-run, pass). **Icon legibility at 14-16px and a final trademark-policy recheck for the real paperless-ngx/SilverBullet marks remain unverified** — this is 09-02-PLAN.md's own Task 4 blocking human-verify checkpoint, explicitly recorded as unresolved in 09-02-SUMMARY.md (`coverage: D5`, `human_judgment: true`, no verification). |
| 4 | SC4 — The "+" menu is reworked: instances show configured URL/directory, adding a new source type is a visually separated plugin-catalog group | ✓ VERIFIED | `AddSourceModal.svelte`: Group 1 ("Add to this webspace", instance rows showing `source.base_url \|\| source.path \|\| pluginTypeLabel(source.plugin)`) and Group 2 ("Install a new source", bordered catalog tiles) confirmed in source (lines 462-505). Proven live by `09-picker-groups.spec.ts` (5/5, re-run, pass), including the instance-row-shows-location and catalog-tile-visually-distinct assertions. |
| 5 | SC5 — Playwright specs cover the browser-drivable changes per the standing Phase 07.1 rule | ✓ VERIFIED | Seven new specs added this phase (`09-plugin-icon`, `09-stream-row-plugin-icon`, `09-static-assets-and-surfaces`, `09-search-clear-and-previewer`, `09-chip-menu`, `09-switcher-reload`, `09-picker-groups`) — 23 test cases total, all re-run for this verification and all passing (23/23, 5.1s). |
| 6 | User-directed scope addition (09-02 Task 4 checkpoint, item 2): plugin identity icons appear on stream/search-result rows, thumbnails preserved | ✓ VERIFIED | `StreamRow.svelte:114` renders `<PluginIcon {plugin} size="size-3.5" />`; `StreamList.svelte:96` and `SearchResults.svelte:75` resolve `plugin` from the already-fetched `sourcesByInstance` map. Structurally guarded by `stream-row-plugin-icon.test.ts` (unit, pass) and proven live by `09-stream-row-plugin-icon.spec.ts` (2/2, re-run, pass — two distinct decoded icons across two plugin types in one mixed stream, and the Thumbnail slot unchanged alongside the new icon). |

**Score:** 6/6 truths verified (technical/structural); 2 human-visual/legal judgment items outstanding (see Human Verification below — these gate the phase's own blocking checkpoint, not a code gap).

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `proto/topos/v1/plugin.proto` | `DescribeResponse.icon = 5`, `icon_mime = 6`, additive | ✓ VERIFIED | Present, wire-compatible per comment at lines 41-60; `sdk/contract_test.go` guards additivity. |
| `kernel/httpapi/pluginicon.go` | `GET /api/plugins/{plugin}/icon`, cached/hardened, 404-as-normal | ✓ VERIFIED | Present, wired into `routes.go`; `go test ./kernel/...` green. |
| `web/src/lib/components/PluginIcon.svelte` | Mandatory 3-step fallback chain (`<img>` → onerror → Puzzle) | ✓ VERIFIED | Present, matches spec exactly (read in full); used by `SourceChip.svelte`, `StreamRow.svelte`, `ManageSourcesModal.svelte`, `AddSourceModal.svelte`. |
| `plugins/{paperless,silverbullet,proton,signal,whatsapp,mock,mockstrict}/assets/icon.svg` | One embedded icon per in-repo plugin, provenance-documented | ✓ VERIFIED | All 7 present; `internal/audit/plugin_icons_test.go#TestPluginIcons_EveryInRepoPluginDeclaresOne` re-run, pass. |
| `web/static/robots.txt` + `kernel/webui/build/robots.txt` twin | Full disallow | ✓ VERIFIED | Both present and byte-identical (`Disallow: /`), confirmed by direct read. |
| `--popover: #172033` in `web/src/app.css` | New elevation step, distinct from `--card`/`--border` | ✓ VERIFIED | Present at both `:root`/`.dark` declaration sites; `dropdown-menu-content.svelte`/`popover-content.svelte` use `bg-popover`. |
| `web/src/lib/components/AddSourceModal.svelte` | Two-group picker (`w-80`) | ✓ VERIFIED | Present, Group 1/Group 2 structure confirmed in source. |
| `web/e2e/specs/09-*.spec.ts` (7 files) | Browser-drivable coverage for every fix | ✓ VERIFIED | All 7 present, all 23 cases pass in a real Chromium run performed by this verification. |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `SourceStatus.plugin` (kernel `/api/sources`) | `PluginIcon` `src` attribute | binary-name key | ✓ WIRED | `kernel/httpapi/sources.go` populates `Plugin`; `SourceChip.svelte`/`StreamRow.svelte` consume `source.plugin`. |
| plugin's `Describe` RPC | `GET /api/plugins/{binary}/icon` | `pluginhost.Plugin.iconBytes/iconMIME` captured at existing launch-time `Describe` call, no new RPC | ✓ WIRED | Confirmed in `kernel/pluginhost/host.go`; `icon_test.go` and `pluginicon_test.go` both green. |
| `sourcesByInstance` map (already fetched for `sourceDisplayName`) | `StreamRow`/`SearchResults` `plugin` prop | frontend-only reuse, no new kernel field | ✓ WIRED | `StreamList.svelte:96`, `SearchResults.svelte:75` both resolve `.plugin` from the same map. |
| `web/static/robots.txt` | `kernel/webui/build/robots.txt` | `npm run build` | ✓ WIRED | Rebuilt during this verification; both files byte-identical. |
| `app.css --popover` token | `bg-popover` utility | Tailwind v4 `@theme inline --color-popover` | ✓ WIRED | Confirmed both menu components use `bg-popover`, not `bg-card`; `Dialog`/`AlertDialog` correctly still `bg-card` (out of scope per Fix 8). |
| `AddSourceModal` picker rows | `PluginIcon` (Group 1) / `Puzzle` fallback (Group 2) | `source.plugin` vs. never-Described plugin type | ✓ WIRED | Confirmed in source; e2e proves the visual distinction between an instance row and a catalog tile. |

### Behavioral Spot-Checks / Full Test Runs

| Behavior | Command | Result | Status |
|---|---|---|---|
| Go build (whole module) | `go build ./...` | clean | ✓ PASS |
| Go tests (kernel/internal/sdk) | `go test ./kernel/... ./internal/... ./sdk/...` | all packages `ok` | ✓ PASS |
| All 7 first-party plugin builds | `CGO_ENABLED=0 go build ./...` per plugin dir | clean (all 6 non-signal + mock/mockstrict) | ✓ PASS |
| Frontend unit tests | `npm test -- --run` | 770/770 passed, 45 files | ✓ PASS |
| Frontend type/svelte-check | `npm run check` | 0 errors, 9 pre-existing warnings (unrelated to phase 9) | ✓ PASS |
| Phase-9 e2e specs (real Chromium) | `npx playwright test --project=chromium specs/09-*.spec.ts` | 23/23 passed | ✓ PASS |
| Regression e2e specs touched by phase-9 locator fixes | `npx playwright test --project=chromium specs/uat-{01,02,03,05,06,08,10}-*.spec.ts specs/mockstrict-discovery.spec.ts` | 23/23 passed | ✓ PASS |
| Plugin-icon repo-wide audit | `go test ./internal/audit/... -run PluginIcons -v` | 2/2 passed | ✓ PASS |

All commands above were re-run independently by this verification agent (not merely cited from SUMMARY.md), against the current worktree HEAD (`5ded7a6`).

### Requirements Coverage

All seven plans (`09-01` through `09-07`) declare `requirements: []` in their PLAN.md frontmatter, confirmed by direct grep. ROADMAP.md's own Phase 9 entry states `**Requirements**: none (polish backlog from TODO.md v1.0 sections)`. Cross-referencing `.planning/REQUIREMENTS.md`'s full requirement table confirms **no requirement ID maps to Phase 9** (grep for "Phase 9" in REQUIREMENTS.md returns no matches) — no orphaned requirements exist for this phase. This is the expected, correct state, not a gap.

### Anti-Patterns Found

No debt markers (`TBD`/`FIXME`/`XXX`) or warning-level markers (`TODO`/`HACK`/`PLACEHOLDER`) found in any phase-9-touched file (checked across all key components and kernel/plugin files this phase modified).

09-REVIEW.md (code review, run prior to this verification) found **0 critical, 2 warnings, 1 info** — advisory only, none block the phase goal:

| File | Finding | Severity | Impact |
|---|---|---|---|
| `internal/audit/plugin_icons_test.go:19,51-56` | Icon-size audit uses `>=` where the kernel enforcement it claims to mirror (`kernel/pluginhost/host.go`) uses `>` — an exactly-65536-byte icon would be wrongly rejected by the audit though the kernel would accept it | ⚠️ Warning (dormant — no committed icon is near this boundary; largest is ~12.7KB) | Will incorrectly block a future plugin author shipping a legally-sized icon; does not affect any currently-shipped icon or phase-9 truth |
| `web/src/lib/components/SourceChip.svelte:104-105` (comment) vs. `:202-210` (behavior) | Folding the standalone refresh button into the overflow menu (Fix 5) removed the chip's only *at-rest* (unopened, unhovered) visual signal that a source is mid-sync; a nearby doc comment still incorrectly claims an "in-place" indicator exists | ⚠️ Warning (real UX regression, but not a violation of any locked must-have or 09-UI-SPEC.md contract line — Fix 5's own spec only requires the spinning icon inside the opened menu, which is present and tested) | Does not block SC1's "chip refresh lives in the 3-dot menu" truth, which is satisfied; noted for a future phase, not this one |
| `kernel/httpapi/pluginicon.go:88` | CSP omits the (functionally redundant) `object-src 'none'` token the sibling rendition route sets | ℹ️ Info (no security gap — `object-src` already falls back to `default-src 'none'`) | Cosmetic consistency only |

### Human Verification Required

1. **Plugin icon legibility at shipped sizes (14px chip/stream, 16px picker/Manage-Sources)**
   **Test:** With the kernel's `[plugins] dir` pointed at freshly-built binaries from this codebase (not a stale external `bin/plugins/`), configure at least one instance of each plugin type and view the chip row, the "+" picker, Manage Sources, and stream/search rows.
   **Expected:** Every plugin's icon reads as legible and distinguishable from the others at its shipped size — the real paperless-ngx/SilverBullet marks read as their upstream logos, and the four Lucide-derived glyphs (Mail/MessageCircle/MessageSquare/FlaskConical) read as distinct shapes.
   **Why human:** Small-glyph visual legibility cannot be verified by source inspection or an automated test. This is 09-02-PLAN.md's own Task 4 blocking checkpoint — the user's first attempt was foiled by an external, local configuration issue (a stale `[plugins] dir` serving pre-icon-contract binaries, root-caused and documented in README.md/config.example.toml by 09-02's checkpoint response) rather than actually completed. `09-02-SUMMARY.md`'s `coverage: D5` explicitly records `human_judgment: true` with an empty `verification: []` array.

2. **Trademark-policy recheck for the real paperless-ngx and SilverBullet logo marks**
   **Test:** Confirm whether paperless-ngx (`paperless-ngx/paperless-ngx`, GPL-3.0) or SilverBullet (`silverbulletmd/silverbullet`, MIT) publish any separate trademark/brand-usage policy restricting third-party embedding of their logo, beyond their code license.
   **Expected:** Either a confirmation that no such restriction exists (as 09-UI-SPEC.md's research found at spec time), or a decision to swap either mark for a generic glyph if one is found.
   **Why human:** Trademark-policy interpretation is a legal/policy judgment outside what a codebase read can settle. 09-UI-SPEC.md's own Fix 10 table explicitly flags this as unresolved ("no separate restrictive brand policy is known to exist... executor should recheck before shipping"), and `09-02-SUMMARY.md`'s D5 confirms this recheck was not performed.

### Gaps Summary

No code-level gaps found. Every ROADMAP.md Phase 9 Success Criterion (1-5) is technically implemented, wired, and covered by passing unit and real-browser end-to-end tests re-run independently by this verification. Every artifact and key link the plans declared exists and functions as specified. The two outstanding items are exactly the ones the phase's own plan (09-02-PLAN.md Task 4) flagged as requiring a human — visual icon legibility at small sizes and a trademark-policy recheck for two real third-party logo marks — and the phase's own SUMMARY.md is honest that these were not completed (the user's first attempt was blocked by an unrelated local configuration issue, now documented and fixable). This routes the phase to `human_needed`, not `gaps_found`: nothing here requires new code, only a visual/legal confirmation once the user's local `[plugins] dir` points at freshly-built binaries.

---

_Verified: 2026-08-11T17:14:56Z_
_Verifier: Claude (gsd-verifier)_
