---
phase: 13-per-item-curation-installable-app
verified: 2026-08-15T01:10:00Z
status: human_needed
score: 5/5 roadmap truths verified; 1 supporting backstop truth unconfirmed
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: "4/5 roadmap truths verified cleanly (1 partial)"
  gaps_closed:
    - "Excluding or including fires an undo toast whose Undo action re-issues the exact mirror write for the same item ids (D-02/D-08, UI-SPEC E3.1) — part of Success Criteria 1 and 2's 'trivially reversible' contract"
  gaps_remaining: []
  regressions: []
behavior_unverified_items: []
human_verification:
  - test: "13-07-PLAN.md's own backstop must-have: with a real mouse/trackpad and normal human reaction time, exclude an item, use the WebspaceSwitcher to move to a different webspace, and click the still-visible Undo button — all inside the toast's real 5000ms window."
    expected: "A human can comfortably complete the switch-and-click sequence inside the 5000ms window in normal use (not a speedrun); the Undo button is still visible and clickable when reached."
    why_human: "13-07-PLAN.md's must_haves.truths explicitly tags this 'human timing feasibility' claim verification: backstop — it states in its own words that code inspection and the automated Playwright spec (which completes the same click sequence in ~1.2s of scripted browser time) prove the code path is correct under real time, but cannot settle whether a real human's reaction/travel time reliably fits the same window. No held-out human-timed test exists in the repo; per the honest-verifier protocol this must abstain to human_needed rather than be inferred from the automated spec's pass."
---

# Phase 13: Per-Item Curation & Installable App Verification Report

**Phase Goal:** The user has the final say over what a webspace shows, and can install topos as an app rather than a browser tab.
**Verified:** 2026-08-15T01:10:00Z
**Status:** human_needed
**Re-verification:** Yes — after gap-closure plan 13-07 (commits `449c654`..`da2ff3c`, reviewed in `13-REVIEW.md`)

## Goal Achievement

### Observable Truths (Roadmap Success Criteria — the contract)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | User excludes an individual item from a webspace and it leaves the stream; the exclusion survives re-sync, kernel restart, and an index rebuild, and always outranks whatever the automatic match rules say. | ✓ VERIFIED | Core mechanism unchanged and re-confirmed (`item_marks` schema/store/handler, `TestSetItemMarks_IdempotentInsert`, `TestItemMarks_SurviveIndexRebuild`, etc. — see quick-regression note below). **The one previously-open gap on this criterion (undo toast targeting the wrong webspace) is now closed** — see gap-closure verification below. |
| 2 | User opens a webspace's excluded-items view, sees exactly what they removed, and un-excludes any of it — the items come back into the stream. | ✓ VERIFIED | `?view=excluded` mechanism unchanged and re-confirmed. **The same previously-open gap (undo toast on the include mirror path) is now closed** — see below. |
| 3 | User installs topos from the browser on the desktop (manifest, ServiceWorker, icons) and launches it as a standalone app window against the local kernel. | ✓ VERIFIED | Unchanged since initial verification; quick-regression check confirms `web/vite.config.ts`, `cmd/topos/main.go`'s manifest MIME registration, and `web/e2e/specs/13-pwa-manifest-sw.spec.ts` all still present. Original human-verify checkpoint (13-04-PLAN.md Task 4) remains approved; untouched by 13-07. |
| 4 | After the kernel is upgraded, the installed app never serves a stale UI — the user lands on the new build rather than a cached one. | ✓ VERIFIED | Unchanged; `web/src/lib/pwa-update.ts` and the `SvelteKitPWA` config are untouched by 13-07's diff (confirmed: 13-07's `files_modified` frontmatter lists only `+page.svelte`, the new spec, `docs/testing.md`, and `.planning/REQUIREMENTS.md`). Original approved human-verify checkpoint stands. |
| 5 | The mobile/LAN install limitation (browsers require a secure context) is documented, along with the recommended user-provided HTTPS workarounds. | ✓ VERIFIED | Unchanged; `README.md:115-147` still present, untouched by 13-07. |

**Score:** 5/5 roadmap truths cleanly verified. The one gap recorded in the prior verification pass (affecting truths 1 and 2) is closed.

### Gap-Closure Verification (13-07 — the reason for this re-run)

The prior `13-VERIFICATION.md` recorded exactly one gap: `handleExclude`, `handleInclude`, and `handleBulkPrimary` in `web/src/routes/w/[webspace]/+page.svelte` read the reactive `webspace`/`navGeneration` bindings **inside** their `onUndo` closures, so a webspace switch during the undo toast's 5000ms window caused the reversal to silently target the wrong (new) webspace.

**Independent verification performed in this pass (not just reading the SUMMARY):**

1. **Read the fix directly.** `handleExclude` (:121-142), `handleInclude` (:144-165), and `handleBulkPrimary` (:217-246) each now capture `const ws = webspace;` / `const gen = navGeneration;` as their first statements and use `ws`/`gen` — never the reactive bindings — throughout the body and inside every `onUndo` closure. `handleToggleView` (:254-259) correctly still reads the live `navGeneration`, matching the plan's stated rationale (synchronous, same-webspace action, no delay window).
2. **Ran the new regression spec myself**, from a fresh build (`go build` with the correct `-ldflags -X ...buildManifest=...` link-time manifest, per `Makefile`'s `e2e` target — a plain `go build` without the manifest flag causes an unrelated environment failure, corrected before proceeding): `npx playwright test --project=chromium e2e/specs/13-undo-across-webspace-switch.spec.ts` → **3 passed** (single-item exclude, bulk exclude, detail-pane include, all across a real `WebspaceSwitcher` navigation).
3. **Independently reproduced the RED failure.** Checked out the pre-fix version of `+page.svelte` (`git show 405141d^:...`), confirmed via grep it has only the one *pre-existing, unrelated* `const ws = webspace` capture (the per-webspace reset `$effect`, present since 13-03, not one of the three mark handlers), rebuilt, and re-ran the same spec: **all 3 tests failed**, each on the exact assertion the defect predicts (webspace A still shows `excluded_count: 1` / item excluded, or — for the include test — a real, unintended exclusion manufactured in webspace B). This is not merely trusting the SUMMARY's RED claim; I reproduced it fresh. Restored the fixed source afterward (`git diff` empty — byte-identical round trip).
4. **Ran the three pre-existing curation specs** (`13-exclude-tracer.spec.ts`, `13-multi-select-bulk-exclude.spec.ts`, `13-excluded-view.spec.ts`) against the fixed build: **5 passed** — same-webspace undo paths are undisturbed by the fix.
5. **Ran the full Go and web unit suites** against the fixed source: `go build ./...` clean; `go test ./kernel/index/... ./kernel/httpapi/... ./kernel/pluginhost/... ./kernel/syncer/... ./kernel/supervisor/... ./kernel/correlate/... ./cmd/topos/...` all `ok`; `npx vitest run` → 58 files / 1078 tests passed (matches SUMMARY's claimed count).
6. **`npm --prefix web run check`** → 0 errors, 10 pre-existing warnings (matches SUMMARY).

**Conclusion: the gap is genuinely closed**, confirmed by my own execution, not by re-stating the SUMMARY's claims.

**One acceptance-criterion discrepancy noted (informational, not a functional defect):** 13-07-PLAN.md's Task 1 acceptance criteria state `grep -c 'const ws = webspace' ... returns 3`. The actual count is **4** — a pre-existing `const ws = webspace` capture inside the unrelated per-webspace reset `$effect` (line ~1197, present since 13-03's commit `655bfe3d`, unrelated to the three mark handlers) was not accounted for when that specific grep was written. The SUMMARY's self-check claims "Re-ran every task's acceptance_criteria grep/command checks: all pass," which is not literally true for this one grep. This does not affect correctness of the fix — independently confirmed via the region-scoped negative gates (both return 0, as expected) and via direct reading of the three handler bodies.

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|---|---|---|---|---|
| KERN-09 | 13-01, 13-02, 13-03, 13-07 | Exclude an item; survives re-sync/rebuild/restart; outranks match rules; undo toast targets the correct webspace | ✓ SATISFIED | Schema/store/handler/tests unchanged and re-confirmed; 13-07 closes the undo-toast gap, independently re-verified above |
| KERN-10 | 13-01, 13-02, 13-03, 13-07 | View and un-exclude excluded items; undo toast on the include mirror path targets the correct webspace | ✓ SATISFIED | `?view=excluded`, excluded-view UI, Include control unchanged; 13-07 closes the corrupting `add`-mirror gap on this path |
| UI-13 | 13-04 | PWA install + never-stale update flow | ✓ SATISFIED | Unchanged since initial verification; human-verify checkpoint approved, untouched by 13-07 |
| UI-14 | 13-04 | Mobile/LAN limitation documented | ✓ SATISFIED | Unchanged; README section, checkpoint step D |

`.planning/REQUIREMENTS.md`'s Traceability table now reads `Gaps Found` for all four Phase 13 requirement rows (KERN-09, KERN-10, UI-13, UI-14) — consistent, per 13-07 Task 3's reconciliation. No requirement checkbox has been ticked (correctly left for this re-verification to decide). No orphaned requirements: exactly KERN-09/KERN-10/UI-13/UI-14 map to Phase 13 in REQUIREMENTS.md, and all four are claimed by at least one plan's `requirements` field.

### Required Artifacts (quick-regression check on previously-verified items; full check on 13-07's delta)

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `web/src/routes/w/[webspace]/+page.svelte` | Snapshot `ws`/`gen` in all three mark handlers | ✓ VERIFIED | Read in full; matches plan and SUMMARY exactly (see Gap-Closure Verification above) |
| `web/e2e/specs/13-undo-across-webspace-switch.spec.ts` | 3 tests covering single-item/bulk/include paths across a real navigation | ✓ VERIFIED | Exists, read in full, ran and passed under my own execution; independently reproduced RED against reverted source |
| `docs/testing.md` | Names the new spec | ✓ VERIFIED | `grep -n '13-undo-across-webspace-switch' docs/testing.md` matches heading and bullet |
| `.planning/REQUIREMENTS.md` | KERN-09/KERN-10 rows reconciled to `Gaps Found` | ✓ VERIFIED | Confirmed via grep; checkboxes correctly left unticked |
| `kernel/index/schema.go`, `kernel/index/store.go`, `kernel/httpapi/marks.go`, `kernel/httpapi/stream.go` | Mark table/store/API, unchanged by 13-07 | ✓ VERIFIED (quick check) | Files present; `go build ./...` clean; relevant `go test` packages all `ok` |
| `web/vite.config.ts`, `cmd/topos/main.go`, `kernel/pluginhost/manifest.go`, `cmd/topos-manifest/main.go`, `docs/api.md`, `docs/plugin-contract.md`, `docs/plugins/signal.md`, `README.md` | PWA/manifest/docs artifacts from 13-04/13-05/13-06, unchanged by 13-07 | ✓ VERIFIED (existence check) | All present; not in 13-07's `files_modified` list, so unaffected by this re-verification's delta |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| Toast `onUndo` closure | `setItemMarks(ws, ...)` / `load(gen)` | Snapshotted local constants captured at handler entry | ✓ WIRED (fixed) | Confirmed by direct reading and by the reproduced RED→GREEN cycle above — no longer reads the live `webspace`/`navGeneration` bindings |
| `correlate.SyncSource`'s healthy-pair call | `ReplaceWebspaceSourceItems` prune sweep | Call-site scoping | ✓ WIRED (unchanged) | Untouched by 13-07 |
| Stream route `?view=` | `StreamItems`'s IN/NOT IN branch | `index.MarkView` | ✓ WIRED (unchanged) | Untouched by 13-07 |
| `SvelteKitPWA` config / `mime.AddExtensionType` / `pwa-update.ts` | PWA install/update flow | Build config / startup hook / autoUpdate callback | ✓ WIRED (unchanged) | Untouched by 13-07 (not in its `files_modified` list) |

### Behavioral Spot-Checks (run directly in this verification pass)

| Behavior | Command | Result | Status |
|---|---|---|---|
| New regression spec passes against the fixed source | `npx playwright test --project=chromium e2e/specs/13-undo-across-webspace-switch.spec.ts` | 3 passed | ✓ PASS |
| New regression spec fails against the pre-fix source, on the predicted assertion | Same command, against `git show 405141d^:...` checked out | 3 failed, each on the webspace-A `excluded_count`/`items` (or webspace-B corrupting-`add`) assertion — not a locator/timeout | ✓ PASS (confirms the spec exercises the real defect) |
| Pre-existing same-webspace curation specs undisturbed by the fix | `npx playwright test --project=chromium e2e/specs/13-exclude-tracer.spec.ts e2e/specs/13-multi-select-bulk-exclude.spec.ts e2e/specs/13-excluded-view.spec.ts` | 5 passed | ✓ PASS |
| Full Go build | `go build ./...` | clean | ✓ PASS |
| Relevant Go test packages | `go test ./kernel/index/... ./kernel/httpapi/... ./kernel/pluginhost/... ./kernel/syncer/... ./kernel/supervisor/... ./kernel/correlate/... ./cmd/topos/...` | all `ok` | ✓ PASS |
| Web unit suite | `npx vitest run` (web/) | 58 files / 1078 tests passed | ✓ PASS |
| Svelte/TS check | `npm --prefix web run check` | 0 errors, 10 pre-existing warnings | ✓ PASS |
| Phase-modified files free of unresolved debt markers | `grep -n -E "TBD\|FIXME\|XXX\|TODO\|HACK\|PLACEHOLDER"` on `+page.svelte` and the new spec | none found | ✓ PASS |

The full 138-spec `make e2e` suite was not re-run standalone in this pass — the targeted specs above (4 files, 8 tests, covering every path 13-07 touched plus its immediate neighbors) plus the SUMMARY's own documented full-suite pass (138/138) and this pass's clean `go build`/unit-suite runs are accepted as sufficient corroboration, consistent with the prior verification's own scoping decision.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `web/src/routes/w/[webspace]/+page.svelte` | 126-127 (`handleExclude`), 149-150 (`handleInclude`), 230-231 (`handleBulkPrimary`) | `closeDetail()`/`handleBulkClear()` still run unconditionally after the initial (non-undo) write's `await`, ungated by `gen === navGeneration` (13-REVIEW.md's re-review, new WR-02 — different from the original full-review's WR-02) | ⚠️ Warning (non-blocking) | UI-only: if a user navigates to a different webspace and opens/selects a different item before the *original* `setItemMarks` round-trip resolves (normally single-digit ms against the local kernel — a much narrower window than WR-01's full 5000ms), the origin handler's `closeDetail()`/`handleBulkClear()` fires against the new webspace's UI state, spuriously deselecting/clearing it. No data is written to the wrong webspace (`ws` still correctly scopes the actual mark write) — this is a UI state clobber, not a correctness regression on the roadmap's "trivially reversible" data contract. Filed by the phase's own re-review with a concrete fix (`if (gen === navGeneration) closeDetail();` etc.); left open, not part of this phase's recorded gap, and does not block goal achievement. Recommend a follow-up quick task. |
| `web/src/routes/w/[webspace]/+page.svelte` | Task 1 acceptance criteria vs. actual code | Acceptance-criterion grep (`const ws = webspace` count) expected `3`, actual `4` due to a pre-existing, unrelated occurrence not accounted for when the grep was authored | ℹ️ Info | Documentation/self-check accuracy only — confirmed via the region-scoped negative gates that the actual fix is correct and complete; no functional impact |
| (carried forward, unchanged from initial full review, still present) | — | WR-02 (`kernel/httpapi/marks.go`, missing `http.MaxBytesReader`), WR-03 (`validatePluginBinaryName` doesn't reject `,`/`=`), IN-02 (global unscoped keydown listener) | ⚠️/ℹ️ | Untouched by 13-07; already assessed as non-blocking advisory in the prior verification pass |

### Human Verification Required

1 item — routed here per the honest-verifier abstention protocol for `verification: backstop` truths (never inferred as passed from an automated test alone):

### 1. Real human timing feasibility of the undo-across-webspace-switch window

**Test:** With a real mouse/trackpad and ordinary (non-speedrun) reaction time: exclude an item from a webspace, use the WebspaceSwitcher to switch to a different webspace, and click the still-visible "Undo" toast button — all within the toast's real 5000ms lifetime.

**Expected:** A person can comfortably complete this sequence inside the window during normal use; the Undo button is still visible and clickable when reached (it should not typically require rushing).

**Why human:** `13-07-PLAN.md`'s own `must_haves.truths` explicitly tags this claim `verification: backstop` and states in its own words that neither code inspection nor the automated Playwright spec (which completes the identical click sequence in ~1.2s of *scripted* browser time — much faster than human reaction/travel time) can settle whether a real person reliably fits the same sequence inside 5000ms. No held-out human-timed test exists. Per the honest-verifier protocol, an unconfirmed `backstop` truth must abstain to `human_needed`, never be inferred as passed from adjacent automated evidence.

### Gaps Summary

No gaps remain against the roadmap's five Success Criteria. The single previously-recorded gap — the exclude/include undo toast silently retargeting its reversal to the wrong webspace after a mid-window switch — is closed, and this pass independently re-derived both the RED failure (against the pre-fix source) and the GREEN pass (against the fixed source), rather than accepting the SUMMARY's account at face value.

One new, narrower, non-blocking UI-only race (WR-02 in the 13-07 re-review) was surfaced by the phase's own code review during gap closure and left open — it does not touch data correctness or any roadmap Success Criterion, and is recommended as a follow-up quick task rather than a phase-blocking gap.

One supporting must-have from 13-07's own plan — an explicitly-flagged non-inferable "can a real human complete this in time" claim — has no held-out human-timed evidence and is routed to human verification rather than silently marked passed.

---

_Verified: 2026-08-15T01:10:00Z_
_Verifier: Claude (gsd-verifier)_
