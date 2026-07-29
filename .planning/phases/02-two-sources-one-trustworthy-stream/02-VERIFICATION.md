---
phase: 02-two-sources-one-trustworthy-stream
verified: 2026-07-29T18:00:00Z
status: human_needed
score: 11/11 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: 6/6
  gaps_closed:
    - "G-02-1: web/src/app.css's seven --spacing-<key> theme entries shadowed Tailwind v4's default container scale, collapsing max-w-xs (health-chip tooltip), max-w-3xl (index page column/card link) and max-w-md (StreamEmpty/StreamError) to 4px/64px/16px instead of their intended 20rem/48rem/28rem"
  gaps_remaining: []
  regressions: []
---

# Phase 2: Two Sources, One Trustworthy Stream Verification Report

**Phase Goal:** User sees two structurally different sources interleaved in one webspace and can filter, refresh, and tell at a glance whether what they are looking at is complete and current

**Verified:** 2026-07-29T18:00:00Z
**Status:** human_needed
**Re-verification:** Yes — after gap-closure plan 02-06 (gap_ids: [G-02-1], commits `3a6d7ec`, `c34c5e1`, docs commit `ee4e2ce`)

## Re-Verification Summary

The prior `02-VERIFICATION.md` (2026-07-29T10:45:00Z) scored 6/6 truths VERIFIED but was already `human_needed` on two carried-forward items. A subsequent end-of-phase UAT session (`02-UAT.md`) exercised those two items in a real browser/fresh-agent context:

- **Test 1** (visual confirmation): **issue** — the user reported the health-chip tooltip renders as an unreadable ~10px sliver, and the index page has a similarly collapsed, text-less, but clickable webspace link. Filed as **G-02-1**, root-caused (`.planning/debug/collapsed-tooltip-and-index-link.md`) to seven `--spacing-<key>` entries inside `web/src/app.css`'s Tailwind v4 `@theme inline` block shadowing the default `--container-*` scale that every named `max-w-*` utility resolves from.
- **Test 2** (PLUG-05 fresh-context self-sufficiency): **pass** — a genuinely isolated subagent built a `SourcePlugin` from only the four allowed contract inputs, 13/13 tests passing, 5 minor interpretive gaps (no missing contract fact). This resolves the second carried-forward human-verification item from the prior report; it does not need to be re-checked here.

Gap-closure plan **02-06** closed G-02-1. This re-verification re-checks the fix directly against the built codebase — not against `02-06-SUMMARY.md` prose — and re-confirms the rest of the phase has zero regression.

### Direct evidence gathered this session

1. **Read `web/src/app.css` in full.** The seven `--spacing-xs/sm/md/lg/xl/2xl/3xl` entries are gone from the `@theme inline` block; only color, radius, and font entries remain, plus a comment recording the shadowing mechanism and the numeric-utility mapping so the mistake isn't reintroduced. Confirmed via `grep -n -- "--spacing-" web/src/app.css` — zero named-key declarations remain (only the explanatory comment references the string).
2. **Rebuilt the SPA from scratch** (`npm --prefix web run build`) and inspected the actual emitted CSS asset (`kernel/webui/build/_app/immutable/assets/*.css`), not SUMMARY claims:
   ```
   .max-w-3xl{max-width:var(--container-3xl)}   --container-3xl:48rem
   .max-w-md{max-width:var(--container-md)}     --container-md:28rem
   .max-w-xs{max-width:var(--container-xs)}     --container-xs:20rem
   ```
   No `.max-w-(xs|sm|md|lg|xl|[0-9]xl){max-width:[0-9]+px}` pattern found anywhere in the built file (the exact collapsed-pixel signature of G-02-1 is now absent).
3. **Traced every consumer of the affected utilities** directly in source: `web/src/lib/components/ui/tooltip/tooltip-content.svelte` (`max-w-xs`), `web/src/routes/+page.svelte` (`max-w-3xl` on `<main>`), `web/src/lib/components/StreamEmpty.svelte` (`max-w-md`, both the default and D-09 filtered-empty variant), `web/src/lib/components/StreamError.svelte` (`max-w-md`) — all four consumers exist, are unmodified by this plan, and now resolve through the corrected container-scale values traced in step 2.
4. **Ran `scripts/assert-stylesheet.sh` against the freshly built stylesheet** (no argument, this session): exit 0, `OK: stylesheet ... passes G-01-2 and G-02-1 guards`.
5. **Ran the plan's own adversarial fixtures directly** (not trusted from SUMMARY): a synthetic collapsed-pixel CSS fixture (`.max-w-3xl{max-width:64px}`) was correctly **rejected** (non-zero exit); a synthetic `var(--container-*)`-based fixture including Tailwind's legitimate `max-w-px{max-width:1px}` utility was correctly **accepted** (exit 0) — proving the guard discriminates the failure mode without over-triggering on the deliberately excluded `px` key.
6. **Confirmed wiring**: `grep -n "assert-stylesheet.sh" scripts/e2e-smoke.sh` shows the smoke test's stylesheet stage now delegates to the shared helper (`"$SCRIPT_DIR/assert-stylesheet.sh" "$CSS_TMP"`), replacing its prior inline non-empty/token checks, while `bash -n` on both scripts confirms clean syntax.
7. **Ran the frontend test/typecheck suites directly this session**: `npm --prefix web run test` — 3 suites, 44/44 passing (unchanged); `npm --prefix web run check` — 0 errors, 0 warnings, 738 files.
8. **Confirmed change-scope discipline directly**: `git diff --stat 3a6d7ec^..c34c5e1 -- . ':!.planning'` touches exactly three files: `web/src/app.css`, `scripts/assert-stylesheet.sh` (new, 86 lines), `scripts/e2e-smoke.sh` (9 lines changed). No Svelte component, no Go file, and no `web/package.json` changed — the fix is scoped exactly to what the plan declared.
9. **Confirmed commits exist**: `3a6d7ec` (fix, Task 1) and `c34c5e1` (feat, Task 2) are both present in `git log`, plus `ee4e2ce` (docs, plan completion) and a subsequent code-review commit.
10. **Confirmed the prohibition (`must_haves.prohibitions`, verification: test)** — "MUST NOT declare named keys in Tailwind v4's spacing theme namespace" — is satisfied: zero `--spacing-<key>` declarations remain in `app.css`'s `@theme inline` block, confirmed by direct grep this session.

**Conclusion: G-02-1 is closed.** The fix is exactly the one-group deletion the gap-closure plan specified, proven by inspecting the actual built stylesheet bytes (not SUMMARY prose), running the plan's own adversarial pass/fail fixtures, and confirming zero regression in the frontend test/typecheck suites and zero scope creep beyond the three declared files.

## Goal Achievement

### Observable Truths — Phase-level (ROADMAP Success Criteria + core value proposition)

Unchanged since the prior verification; re-confirmed this session as a regression check (all supporting files outside `web/src/app.css`/`scripts/*` are untouched by 02-06, per the change-scope diff in evidence item 8).

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SilverBullet pages matching the webspace keyword by tag or page name appear interleaved with paperless documents in one chronological stream, each with an exact deep link to its page | VERIFIED (regression-checked) | Untouched by 02-06; prior session's live-API and unit-test evidence stands. |
| 2 | User can filter the stream down to a single source and back to all sources | VERIFIED (regression-checked) | `SourceFilterChips.svelte` untouched by 02-06; 44/44 vitest tests re-run this session, still pass. |
| 3 | User can see per-source health in the UI — reachable, last sync time, last error — and trigger a manual refresh that brings in new items without duplicating existing ones or stacking concurrent syncs | VERIFIED (regression-checked) | `kernel/syncer/coordinator.go` untouched by 02-06 (Go files unaffected by this CSS-only gap closure). |
| 4 | An item whose source is unreachable or has been deleted shows an explicit unavailable/stale state in the stream and detail pane, never a blank pane or a silent 404 | VERIFIED (regression-checked) | `detailPaneState`/`StreamRow.svelte` untouched; `staleness.test.ts` re-confirmed passing this session via `npm run test`. |
| 5 | Config grants agent read access and action hand-off separately, per plugin, defaulting to deny — a plugin with no grant is absent from agent-facing API responses | VERIFIED (regression-checked) | `kernel/config/types.go`, `kernel/httpapi/agent.go` untouched by 02-06. |
| 6 | A SilverBullet sync that experiences a genuine outage or auth failure mid-Match is recorded as an error, not silently reported as a successful sync with zero items, and does not silently delete previously-synced items | VERIFIED (regression-checked) | `plugins/silverbullet/plugin.go` untouched by 02-06 (out of its file scope); the 02-05 fix stands. |

### Observable Truths — 02-06 gap closure (G-02-1), from PLAN frontmatter `must_haves.truths`

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 7 | Hovering (or focusing) a source health chip shows a tooltip bubble wide enough to read its whole sentence, not a ~10px sliver | VERIFIED (mechanism proven; pixel render unconfirmed this session) | `tooltip-content.svelte`'s `max-w-xs` now resolves to `var(--container-xs)` = 20rem in the actual built CSS (read directly, not from SUMMARY). CSS is deterministic — the class-to-pixel mapping fully determines the rendered width, with no further runtime logic — but no browser/screenshot tool was available in this verification session to capture the rendered DOM. See Human Verification #1. |
| 8 | The index page renders its column at normal page width and each webspace as a full-width card whose name is visible and clickable | VERIFIED (mechanism proven; pixel render unconfirmed this session) | `+page.svelte`'s `<main class="mx-auto max-w-3xl px-6">` now resolves `max-w-3xl` to `var(--container-3xl)` = 48rem in the built CSS (read directly). Same caveat as #7 — see Human Verification #1. |
| 9 | The built application stylesheet resolves named max-width utilities from Tailwind v4's default container scale (`max-w-xs`=20rem, `max-w-md`=28rem, `max-w-3xl`=48rem), not hand-authored pixel values | VERIFIED | Read directly from the freshly rebuilt CSS this session (evidence item 2); `scripts/assert-stylesheet.sh` (run directly, exit 0) codifies and confirms this. |
| 10 | `StreamEmpty` (including the D-09 filtered-empty variant) and `StreamError` render their copy blocks at readable paragraph width | VERIFIED (mechanism proven; pixel render unconfirmed this session) | Both components' `max-w-md` classes traced directly in source (evidence item 3); both resolve to `var(--container-md)`=28rem in the built CSS. Never rendered in the original UAT session (latent victims) — still not visually exercised. See Human Verification #1. |
| 11 | A future re-introduction of a theme entry that shadows the container scale is caught by an automated stylesheet assertion before it reaches a browser | VERIFIED | `scripts/assert-stylesheet.sh` exists, is executable, passes against the real build, and — run directly this session with the plan's own adversarial fixtures — correctly rejects a collapsed-pixel fixture and correctly accepts a `var()`-based fixture containing the legitimate `max-w-px` exemption. `scripts/e2e-smoke.sh` delegates to it (confirmed by grep), so the guard also runs against the live-served stylesheet, not just the local build. |

**Score:** 11/11 truths verified (0 failed; 0 present-but-behaviorally-unproven in the state-transition sense — truths 7/8/10 are proven at the CSS-mechanism level with pixel-deterministic evidence, and are separately flagged for a final human visual pass per the project's `human_verify_mode: end-of-phase` policy and the standing "visual appearance always needs human" rule)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `web/src/app.css` | `@theme inline` carries only color/radius/font entries — no named spacing-scale keys shadowing the container scale | VERIFIED | Read directly this session; zero `--spacing-<key>` declarations remain (only an explanatory comment). |
| `scripts/assert-stylesheet.sh` | Reusable, standalone CSS assertion covering G-01-2 and G-02-1 guards | VERIFIED | Exists, executable, 87 lines (`min_lines: 45` satisfied), contains `max-width`; ran directly this session against the real build (PASS) and both adversarial fixtures (PASS/PASS as expected). |
| `scripts/e2e-smoke.sh` | Stylesheet stage delegates to the shared helper | VERIFIED | Contains `assert-stylesheet.sh`; `bash -n` clean; prior inline non-empty/token checks removed, replaced by a single delegated call. |
| All 19 other artifacts from the prior 02-VERIFICATION.md | Unchanged | VERIFIED (regression, confirmed via diff scope, not re-read byte-for-byte) | `git diff --stat 3a6d7ec^..c34c5e1 -- . ':!.planning'` confirms zero changes to any file outside the three named above. |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `web/src/app.css` | `web/src/lib/components/ui/tooltip/tooltip-content.svelte` | `max-w-xs`'s resolved value is decided entirely by which theme namespace `app.css` populates | VERIFIED | Confirmed directly: theme fix in place, `tooltip-content.svelte`'s class list unchanged (`w-fit max-w-xs`), built CSS resolves `max-w-xs` to `var(--container-xs)`=20rem. |
| `web/src/app.css` | `web/src/routes/+page.svelte` | `<main>`'s `max-w-3xl` cap, and `Card.Root`'s `overflow-hidden` clipping once that cap collapses | VERIFIED | Confirmed directly: `+page.svelte` unchanged, `max-w-3xl` now resolves to 48rem in the built CSS — the collapse mechanism is gone. |
| `scripts/e2e-smoke.sh` | `scripts/assert-stylesheet.sh` | the smoke test fetches the served stylesheet and hands the file to the helper instead of running its own inline greps | VERIFIED | `grep -n "assert-stylesheet.sh" scripts/e2e-smoke.sh` shows the delegated call at line 78, replacing the two prior inline checks. |
| All other key links from the prior 02-VERIFICATION.md | Unchanged | VERIFIED (regression) | No code touched by 02-06 affects any other link; diff scope confirms this. |

### Data-Flow Trace (Level 4)

Not applicable in the usual dynamic-data sense — this gap closure is a static CSS theme fix, not a data-fetching path. The equivalent "data flow" here is the CSS cascade: `app.css`'s `@theme inline` block → Tailwind's generated utility classes → the four consuming components' rendered widths. Traced directly (evidence items 2–3) and confirmed to resolve correctly end to end in the actual built artifact.

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Freshly built stylesheet resolves the three named container widths correctly | `npm --prefix web run build` then grep the emitted CSS (run directly this session) | `--container-xs:20rem`, `--container-md:28rem`, `--container-3xl:48rem`; zero collapsed-pixel matches | PASS |
| `assert-stylesheet.sh` passes against the real build | `./scripts/assert-stylesheet.sh` (no arg, run directly) | `OK: stylesheet ... passes G-01-2 and G-02-1 guards` | PASS |
| `assert-stylesheet.sh` rejects a collapsed-pixel fixture | `./scripts/assert-stylesheet.sh "$BAD"` where `$BAD` contains `.max-w-3xl{max-width:64px}` | non-zero exit | PASS |
| `assert-stylesheet.sh` accepts a correct `var()`-based fixture with the `max-w-px` exemption | `./scripts/assert-stylesheet.sh "$GOOD"` | exit 0 | PASS |
| `e2e-smoke.sh` delegates its stylesheet stage to the helper | `grep -n "assert-stylesheet.sh" scripts/e2e-smoke.sh` | line 78: `"$SCRIPT_DIR/assert-stylesheet.sh" "$CSS_TMP"` | PASS |
| Both scripts remain syntactically valid | `bash -n scripts/assert-stylesheet.sh && bash -n scripts/e2e-smoke.sh` | clean | PASS |
| Frontend unit/type suites, no regression | `npm --prefix web run test` / `npm --prefix web run check` | 44/44 tests pass; 0 type errors/warnings across 738 files | PASS |
| Change-scope discipline: only the three intended files changed | `git diff --stat 3a6d7ec^..c34c5e1 -- . ':!.planning'` | `web/src/app.css`, `scripts/assert-stylesheet.sh` (new), `scripts/e2e-smoke.sh` only | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| SRC-05 | 02-01, 02-05 | SilverBullet plugin; tag/page matching; exact deep links; failure-mode integrity | SATISFIED | Unchanged by 02-06; prior session's evidence stands. |
| KERN-04 | 02-01, 02-02 | Sync scheduler + coordinator; dedups refreshes; manual refresh | SATISFIED | Unchanged by 02-06. |
| PLUG-04 | 02-02 | Plugins report health (reachable, last sync, last error) | SATISFIED | Unchanged by 02-06. The reason the health chip's diagnostic detail was previously *unreadable* (G-02-1) was a pure CSS layout defect, not a data/wiring defect — `last_error`/`last_sync` were already correctly sourced and rendered, just visually clipped. That defect is now closed. |
| PLUG-05 | 02-04 | Third party can build a plugin from contract + mock alone | SATISFIED | `02-UAT.md` test 2 (source: automated, a genuinely isolated fresh subagent): clean build, 13/13 tests pass, 5 minor interpretive gaps, no missing contract fact — resolves the prior report's open human-verification item on this requirement. |
| UI-02 | 02-03 | Filter by source | SATISFIED | Unchanged by 02-06; `02-UAT.md` test 1 confirms filter chips work correctly in a real browser ("everything passes as described" apart from the two G-02-1 symptoms). |
| UI-05 | 02-03, 02-05, 02-06 | Stale/unavailable explicit states, never silent 404; sync-run integrity; health-chip diagnostic must be legible | SATISFIED | The G-02-1 defect that made the health chip's own diagnostic tooltip unreadable — directly undermining "tell at a glance" — is closed at the CSS-mechanism level (evidence items 2–3); final pixel confirmation is the one open Human Verification item. |
| UI-06 | 02-03, 02-06 | Sync status and plugin health visible | SATISFIED | Same as UI-05 — the visibility mechanism (tooltip width) is fixed; pixel confirmation pending. |
| AGENT-01 | 02-04 | Per-plugin permission model, default-deny | SATISFIED | Unchanged by 02-06. |

**No orphaned requirements**: all 8 IDs are exactly the union of every plan's `requirements:` frontmatter field (including 02-06's `[UI-05, UI-06]`) and match `REQUIREMENTS.md`'s Phase 2 traceability rows in both directions.

**Note (documentation lag, not an implementation gap):** `.planning/REQUIREMENTS.md`'s traceability table still marks all 8 of these IDs `Gaps Found` as of this verification — it was last written when the phase was at an earlier gaps state and has not been updated to reflect either the 02-05 or 02-06 gap closures. This is a stale-tracking-document issue, not evidence the underlying requirements are unmet; recommend updating `REQUIREMENTS.md` alongside or after this verification is filed.

### Anti-Patterns Found

No `TBD`/`FIXME`/`XXX` debt markers in any of the three files 02-06 touched (`web/src/app.css`, `scripts/assert-stylesheet.sh`, `scripts/e2e-smoke.sh`) — confirmed by direct grep this session (the one `XXXXXX` match in `e2e-smoke.sh` is `mktemp`'s template placeholder, not a debt marker).

Carried forward unchanged from the prior verification (not touched by 02-06's file scope, confirmed via the change-scope diff):

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `web/src/lib/components/DetailPane.svelte` | 33-40, 72 | Local `formatDate` reintroduces the UTC timezone bug (WR-01) | Warning | Unchanged; narrow (date display only). |
| `web/src/lib/components/DetailPane.svelte` | 143, 149 | `<iframe>` elements carry no `sandbox` attribute (WR-02) | Warning | Unchanged; latent, not currently exploitable (CSP already sandboxes). |
| `kernel/httpapi/stream.go`, `webspaces.go`, `agent.go` | various | `LatestSyncRunPerSource` errors silently downgraded to "unknown" (WR-03) | Warning | Unchanged; low likelihood, narrow blast radius. |
| `cmd/webspaces/main.go` | `runServe` | No OS signal handling for graceful shutdown (WR-04) | Warning | Unchanged; operational hygiene. |
| `web/src/routes/w/[webspace]/+page.svelte` | — | Polling interval not cleared on component destroy (WR-05) | Warning | Unchanged; resource-cleanup hygiene, not a phase-goal blocker. |

**New minor finding (informational, not a gate item):** `02-06-SUMMARY.md`'s "Issues Encountered" section states the unrelated `e2e-smoke.sh` failure (`link.url` not matching `$PAPERLESS_URL/documents/` prefix) was "Logged to the phase's deferred-items tracking" — direct inspection of `.planning/phases/02-two-sources-one-trustworthy-stream/deferred-items.md` this session shows it was **not** actually added there. This is a minor SUMMARY-accuracy discrepancy, not a phase-goal blocker: the underlying issue is pre-existing (predates 02-06, outside its declared `files_modified`), environment-dependent (a `PAPERLESS_URL`/configured-source-URL mismatch in this execution environment, not a code defect in the deep-link construction verified under truth #1), and does not touch any of this phase's must-haves. Flagged here so it isn't lost, since the SUMMARY's own claim of where it was logged doesn't hold up.

## Human Verification Required

### 1. Final visual confirmation that G-02-1's fix renders correctly in a real browser

**Test:** Run `make build`, then `./bin/webspaces serve`, and open `http://127.0.0.1:7777/` in a browser. Confirm: (1) the index page renders a normal-width column and each webspace is a full-width card showing its name (e.g. "house-move") and item count — not a few-pixels-wide clickable sliver; (2) on `/w/house-move`, hovering a source health chip shows a tooltip bubble wide enough to read the entire sentence, including the full error text on an unreachable source; (3) if a filtered-empty state is reachable via a source filter chip, its copy renders as a readable paragraph rather than a ~16px column; (4) a genuinely errored stream (`StreamError.svelte`) also renders at readable width if reachable.

**Expected:** All four surfaces render at their intended widths (20rem/48rem/28rem), matching `02-UI-SPEC.md`'s Copywriting Contract and Design System, with no recurrence of the ~10px/~64px/~16px collapse reported in `02-UAT.md`.

**Why human:** No browser/screenshot automation tooling (Playwright, Chromium, etc.) was available in this verification session, mirroring the gap-closure execution session's own constraint. The fix is proven at the CSS-mechanism level with pixel-deterministic evidence — the built stylesheet's `max-w-xs`/`max-w-md`/`max-w-3xl` rules now resolve to exactly the values a browser will apply (20rem/28rem/48rem, confirmed by direct inspection of the compiled asset) — but actual rendered pixels in a live browser session have not been re-captured since the fix, and visual appearance always requires a final human look per this project's `human_verify_mode: end-of-phase` policy.

## Gaps Summary

No gaps remain. Gap **G-02-1** — `web/src/app.css`'s seven named `--spacing-<key>` theme entries shadowing Tailwind v4's default container scale, collapsing the health-chip tooltip (`max-w-xs`), the index page column/card link (`max-w-3xl`), and `StreamEmpty`/`StreamError`'s copy width (`max-w-md`) to 4px/64px/16px — is closed. Verified this session by rebuilding the SPA from scratch and reading the actual emitted CSS bytes (not SUMMARY claims), confirming all three named widths now resolve to 20rem/48rem/28rem via `var(--container-*)`, running the plan's own adversarial pass/fail fixtures against `scripts/assert-stylesheet.sh` directly, confirming the e2e smoke test now delegates to that guard, and confirming the change's blast radius was exactly the three files the gap-closure plan declared (`git diff --stat`), so the rest of the phase carries zero regression risk (re-confirmed via a fresh `npm run test`/`npm run check` pass).

One human-verification item remains, narrower than the prior report's two: a final real-browser visual confirmation that the fix renders correctly (the CSS-mechanism evidence is pixel-deterministic and strong, but pixel rendering itself is not something grep/build inspection can substitute for). The second item from the prior report — PLUG-05's fresh-context self-sufficiency exercise — is now resolved (`02-UAT.md` test 2: pass, via a genuinely isolated subagent). Five non-blocking warnings (WR-01 through WR-05) remain open for a future fix pass; none block the phase goal. One minor SUMMARY-accuracy discrepancy (a claimed deferred-items log entry that isn't actually present) is flagged for awareness but does not affect the phase goal or this verification's outcome.

---

_Verified: 2026-07-29T18:00:00Z_
_Verifier: Claude (gsd-verifier)_
