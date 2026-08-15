---
phase: 13-per-item-curation-installable-app
verified: 2026-08-15T15:45:00Z
status: passed
score: 5/5 roadmap truths verified; 8/8 gap-closure (13-08) must-haves verified; 1 backstop truth confirmed via human UAT evidence
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: "5/5 roadmap truths verified; 1 supporting backstop truth unconfirmed"
  gaps_closed:
    - "G-13-1: clicking Undo after switching webspaces stranded four permanent loading skeletons in the navigated-to webspace (found by 13-UAT.md test 1, root-caused to load()'s loadState='loading' write running before its staleness guard)"
  gaps_remaining: []
  regressions: []
behavior_unverified_items: []
human_verification: []
---

# Phase 13: Per-Item Curation & Installable App Verification Report

**Phase Goal:** The user has the final say over what a webspace shows, and can install topos as an app rather than a browser tab.
**Verified:** 2026-08-15T15:45:00Z
**Status:** passed
**Re-verification:** Yes — after gap-closure plan 13-08 (commits `4f67d32`, `d44fd68`, `bcf09dc`), which closed 13-UAT.md's G-13-1

## Context for this run

The prior verification pass (`human_needed`) closed with exactly one open item: a `verification: backstop` truth in 13-07-PLAN.md that could not be settled by code inspection or automated tests — whether a real human can comfortably complete "exclude → switch webspace → click Undo" inside the toast's real 5000ms window. That item was routed to human UAT (`13-UAT.md`). The human tester confirmed the timing/reachability half explicitly ("the toast and button are still reachable and the button can be clicked (pass)") but surfaced a genuine new defect on the same path: clicking Undo left four permanent loading skeletons in the destination webspace, requiring a reload or "Refresh all" to clear. This was recorded as gap **G-13-1** (severity: major).

Gap-closure plan **13-08** fixed the defect (an entry guard at the top of `load()` in `+page.svelte`, closing the class of bug for every caller, not just the three `onUndo` closures that exposed it), added a browser regression spec reproducing the exact UAT condition (empty second webspace), and corrected three sites that had recorded a false "stale-generation `load` is a no-op by design" claim.

This re-verification independently confirms 13-08's fix — not by trusting 13-08-SUMMARY.md, but by reading the code, running the new spec myself, and reproducing both RED (against the pre-fix commit, in an isolated worktree) and GREEN (against the current source).

## Goal Achievement

### Observable Truths (Roadmap Success Criteria — the contract)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | User excludes an individual item from a webspace and it leaves the stream; the exclusion survives re-sync, kernel restart, and an index rebuild, and always outranks whatever the automatic match rules say. | ✓ VERIFIED | Core mechanism unchanged since initial verification (`item_marks` schema/store/handler, `go test ./kernel/index/... ./kernel/httpapi/...` both `ok` in this pass). The previously-open defect on this criterion's reversal path (G-13-1) is now closed — see Gap-Closure Verification below. |
| 2 | User opens a webspace's excluded-items view, sees exactly what they removed, and un-excludes any of it — the items come back into the stream. | ✓ VERIFIED | `?view=excluded` mechanism unchanged. The same defect on the include mirror path (G-13-1) is now closed — see below. |
| 3 | User installs topos from the browser on the desktop (manifest, ServiceWorker, icons) and launches it as a standalone app window against the local kernel. | ✓ VERIFIED | Unchanged since initial verification; not in 13-08's `files_modified`. `13-pwa-manifest-sw.spec.ts` passed in this pass's full `make e2e` run. |
| 4 | After the kernel is upgraded, the installed app never serves a stale UI — the user lands on the new build rather than a cached one. | ✓ VERIFIED | Unchanged; `web/src/lib/pwa-update.ts` untouched by 13-08 (`files_modified` lists only `+page.svelte`, the spec, and `docs/testing.md`). |
| 5 | The mobile/LAN install limitation (browsers require a secure context) is documented, along with the recommended user-provided HTTPS workarounds. | ✓ VERIFIED | Unchanged; `README.md:115-147` still present, untouched by 13-08. |

**Score:** 5/5 roadmap truths cleanly verified.

### Gap-Closure Verification (13-08 — closing G-13-1)

**Independent verification performed in this pass (code read + my own execution, not the SUMMARY's account):**

1. **Read the fix directly.** `load(gen, options)` in `web/src/routes/w/[webspace]/+page.svelte` now opens with `if (gen !== navGeneration) return;` as its literal first statement — before the `quiet` destructuring and before `if (!quiet) loadState = 'loading'`. Both pre-existing post-await guards (the in-flight-fetch race) are byte-for-byte unchanged. Confirmed via `git diff ceca9dd..HEAD -- "web/src/routes/w/[webspace]/+page.svelte"`: the entire diff is +29/-2 lines — one guard line plus doc-comment prose. No other call site in the file was touched.
2. **Ran the new/extended spec myself** from a fresh build (`make e2e E2E_ARGS="e2e/specs/13-undo-across-webspace-switch.spec.ts"`): **4 passed**, including the new 4th test (`exclude in A4, switch to EMPTY B4, Undo — B4 renders no stranded skeleton (G-13-1)`) which reproduces 13-UAT.md's exact reported condition (a genuinely empty second webspace).
3. **Independently reproduced the RED failure**, from scratch, using an isolated git worktree checked out at commit `4f67d32` (the RED commit — new test added, fix not yet applied): ran the identical `make e2e E2E_ARGS=...` command and got **3 passed / 1 failed**, the failure landing exactly on the predicted assertion — `expect(page.getByText('Nothing here yet')).toBeVisible()` timing out on webspace B4, never on any A4 (kernel-side) assertion. This matches 13-08-SUMMARY.md's recorded RED evidence verbatim, but I did not take the SUMMARY's word for it — I reproduced the failure fresh in a disposable worktree, then removed it (`git worktree remove --force`).
4. **Ran the full `make e2e` suite once**, against the fixed source, from the main working tree: **139 passed, 0 failed** — matching 13-08-SUMMARY.md's claimed count, including every same-webspace curation spec (`13-exclude-tracer.spec.ts`, `13-multi-select-bulk-exclude.spec.ts`, `13-excluded-view.spec.ts`) and `spec-hygiene.spec.ts`.
5. **Ran the web unit suite once** (`npm --prefix web run test`): **58 files / 1078 tests passed** — matches SUMMARY.
6. **Ran Go build and the affected packages' tests**: `go build ./...` clean; `go test ./kernel/index/... ./kernel/httpapi/...` both `ok`.
7. **Ran `npm --prefix web run check` and `check:e2e`**: 0 errors both (10 pre-existing, unrelated warnings on `check`, matching SUMMARY).
8. **Verified the false-claim removal**: `grep -rn "no-ops by design" web/e2e/specs docs/testing.md` → no matches. Read `docs/testing.md`'s rewritten catalogue entry for the spec — it now states the no-op property as a direct consequence of `load()`'s entry guard, and describes the two-layer (kernel-for-A, rendered-DOM-for-B) assertion strategy and the empty-second-webspace reproduction.
9. **Verified 13-08-PLAN.md's `must_haves.prohibitions`** were honored, by diffing the actual change: no `quiet: true` was added to any `onUndo` closure (confirmed by reading `handleExclude`/`handleInclude`/`handleBulkPrimary`, unchanged since 13-07 except the surrounding comment); the two post-await guards inside `load` are untouched; `handleToggleView`, `writeFilter`, `handleSearch`, `ensurePolling`, and the webspace-keyed `$effect` are absent from the diff entirely; no `data-testid` attribute was added (the new skeleton locator uses the pre-existing `data-slot="skeleton"` attribute the shared shadcn `Skeleton` component already ships); and no kernel-side assertion in the spec file was weakened or removed (`git diff` shows only click-mechanism refactors — the three `undoButton.click()` sequences were replaced by the shared `clickUndo` helper — and comment-text corrections; no `expect(...)` assertion line beyond the click-related ones was deleted).

**Conclusion: G-13-1 is genuinely closed**, confirmed by my own execution and my own RED/GREEN reproduction — not by re-stating 13-08-SUMMARY.md's claims.

### 13-08 Must-Haves (gap-closure plan frontmatter)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | A stale-generation `load(gen)` call performs NO observable work; the guard is at the top of `load`, holding for every caller. | ✓ VERIFIED | Read directly; confirmed by RED→GREEN reproduction above. |
| 2 | After exclude in A → switch to B inside the toast window → Undo, B's stream renders exactly what it rendered before: no skeleton, row set matches the kernel. | ✓ VERIFIED | 4th spec test, run by me, passed. |
| 3 | The exact UAT reproduction (webspace B EMPTY) is pinned by a browser spec; B shows `Nothing here yet`, never four skeleton rows. | ✓ VERIFIED | `WS_B4` seeded `keywords: []`; test asserts `getByText('Nothing here yet')` visible and `streamSkeletonLocator` count 0. Passed. |
| 4 | Navigating back to A after the cross-webspace Undo shows the restored item as a rendered row — user-visible on return, not merely kernel-true. | ✓ VERIFIED | Spec lines 373-374: `switchWebspace(page, WS_B4, WS_A4)` then `expect(page.locator('[data-item-id="${itemId}"]')).toBeVisible()`. Passed. |
| 5 | 13-07's kernel-side guarantees are unchanged and still asserted (A relists the item, `excluded_count` 0; B gains no mark), across all three write paths. | ✓ VERIFIED | Confirmed unchanged via `git diff`; all 4 tests (covering single-item/bulk/include + the new B4 test) passed. |
| 6 | The post-await generation guards inside `load` remain in place, covering the separate in-flight-fetch race. | ✓ VERIFIED | Read directly — both guards present, byte-identical to pre-fix source. |
| 7 | Every place the repo recorded the false "stale-generation refetch is a no-op by design" claim now states it as a consequence of the entry guard. | ✓ VERIFIED | `grep -rn "no-ops by design" web/e2e/specs docs/testing.md` → 0 matches; `docs/testing.md` and the spec's own comments read the corrected version. |
| 8 | The new B-side assertions are non-vacuous — sequenced after the undo's mark POST response reaches the page, not merely committed by the kernel. | ✓ VERIFIED | `clickUndo` registers `page.waitForResponse` for the `/api/webspaces/{ws}/marks` POST *before* clicking, and awaits it before returning; every B-side assertion in all 4 tests runs after that await. Confirmed by reading the helper and by the fact that the RED reproduction failed on the intended assertion rather than passing vacuously. |

**Score:** 8/8 gap-closure must-haves verified.

### Backstop Truth Resolution (carried over from the previous verification pass)

The prior verification's single open item was 13-07-PLAN.md's `verification: backstop` truth: whether a real human, at normal reaction speed, can complete "exclude → switch webspace → click Undo" inside the toast's real 5000ms window. Per the honest-verifier protocol this cannot be inferred from an automated Playwright run (which completes the sequence in ~1.2s of scripted time) — it required explicit human evidence.

That evidence now exists: `13-UAT.md` test 1 records a real human performing exactly this sequence and reporting **"the toast and button are still reachable and the button can be clicked (pass)"** — a direct, explicit confirmation of the timing/reachability claim, independent of the stranded-skeleton defect (G-13-1) the same test surfaced and which 13-08 has since fixed. This is not an inference from adjacent automated evidence; it is the held-out human-timed test the previous pass required, now on record.

**Status: ✓ VERIFIED (via human UAT evidence, `13-UAT.md` test 1).** No further human verification is required for this truth.

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|---|---|---|---|---|
| KERN-09 | 13-01, 13-02, 13-03, 13-07, 13-08 | Exclude an item; survives re-sync/rebuild/restart; outranks match rules; undo toast targets the correct webspace and never strands the destination webspace's stream | ✓ SATISFIED | Schema/store/handler/tests unchanged and re-confirmed; 13-08 closes G-13-1, independently re-verified above |
| KERN-10 | 13-01, 13-02, 13-03, 13-07, 13-08 | View and un-exclude excluded items; undo toast on the include mirror path targets the correct webspace and never strands the destination webspace's stream | ✓ SATISFIED | `?view=excluded`, excluded-view UI, Include control unchanged; 13-08 closes the corrupting-skeleton gap on this path (detail-pane include test in the same spec) |
| UI-13 | 13-04 | PWA install + never-stale update flow | ✓ SATISFIED | Unchanged since initial verification; untouched by 13-07/13-08; `13-pwa-manifest-sw.spec.ts` passed in this pass's full suite run |
| UI-14 | 13-04 | Mobile/LAN limitation documented | ✓ SATISFIED | Unchanged; README section, checkpoint step D |

`.planning/REQUIREMENTS.md`'s Traceability table currently still reads `Gaps Found` for all four Phase 13 requirement rows (KERN-09, KERN-10, UI-13, UI-14) and no requirement checkbox is ticked — this is stale as of gap closure (13-08's `files_modified` did not include `REQUIREMENTS.md`, correctly leaving the reconciliation for this verification pass to decide). Given this pass's `passed` status, the orchestrator should tick all four checkboxes and update the Traceability rows to `Complete` as part of closing out the phase. No orphaned requirements: exactly KERN-09/KERN-10/UI-13/UI-14 map to Phase 13, and all four are claimed by at least one plan's `requirements` field (13-08 additionally re-declares KERN-09/KERN-10, consistent with its scope).

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `web/src/routes/w/[webspace]/+page.svelte` | Entry guard at top of `load()`, plus corrected mark-handler comments | ✓ VERIFIED | Read in full; diff is +29/-2 lines exactly as claimed; no other call site touched |
| `web/e2e/specs/13-undo-across-webspace-switch.spec.ts` | 4 tests (3 extended + 1 new), all asserting B's rendered stream, sharing `clickUndo` | ✓ VERIFIED | Read in full; ran and passed under my own execution (both targeted and full-suite runs); independently reproduced RED against the pre-fix commit |
| `docs/testing.md` | Spec catalogue entry rewritten to reflect the two-layer strategy and empty-B reproduction, false claim removed | ✓ VERIFIED | `grep` confirms absence of false claim; entry read in full, accurate |
| `.planning/REQUIREMENTS.md` | KERN-09/KERN-10 rows | ⚠️ STALE (informational) | Still reads `Gaps Found`/unticked — expected, out of 13-08's declared scope; flagged above for orchestrator reconciliation, not a phase-blocking gap |
| `kernel/index/schema.go`, `kernel/index/store.go`, `kernel/httpapi/marks.go`, `kernel/httpapi/stream.go` | Mark table/store/API, unchanged by 13-08 | ✓ VERIFIED (quick check) | Files present; `go build ./...` clean; `go test ./kernel/index/... ./kernel/httpapi/...` both `ok` |
| PWA/manifest/docs artifacts from 13-04/13-05/13-06 | Unchanged by 13-07/13-08 | ✓ VERIFIED (existence + full e2e suite pass) | Not in 13-08's `files_modified`; `13-pwa-manifest-sw.spec.ts` passed in this pass's full-suite run |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `load(gen)`'s entry guard | `loadState` | Guard runs before any `loadState` write | ✓ WIRED (fixed) | Confirmed by direct reading and by the reproduced RED→GREEN cycle |
| Toast `onUndo` closure | `load(gen)` with a possibly-stale `gen` | Snapshotted local constants (13-07) now made safe by the entry guard (13-08) | ✓ WIRED | Confirmed unchanged since 13-07, and now provably safe against the stranding defect |
| `clickUndo`'s `page.waitForResponse` | The mark POST reaching the page | Registered before the click, awaited after | ✓ WIRED | Read directly; matches `setItemMarks`'s actual request URL pattern |
| `StreamList.svelte`'s branch order | Skeleton vs. empty-state vs. row rendering | Skeleton branch checked strictly before every response-derived branch | ✓ WIRED (unchanged) | Confirmed via 13-REVIEW.md's re-review and this pass's own passing spec runs |

### Behavioral Spot-Checks (run directly in this verification pass)

| Behavior | Command | Result | Status |
|---|---|---|---|
| New/extended regression spec passes against the fixed source | `make e2e E2E_ARGS="e2e/specs/13-undo-across-webspace-switch.spec.ts"` | 4 passed | ✓ PASS |
| New spec fails against the pre-fix source (isolated worktree at commit `4f67d32`), on the predicted assertion | Same command, against a `git worktree add` checkout of `4f67d32` | 3 passed / 1 failed — the G-13-1 test failed on `getByText('Nothing here yet')` timing out on webspace B4, not on any A4 assertion | ✓ PASS (confirms the spec exercises the real defect) |
| Full Playwright suite | `make e2e` (no `E2E_ARGS`) | 139 passed, 0 failed | ✓ PASS |
| Web unit suite | `npm --prefix web run test` | 58 files / 1078 tests passed | ✓ PASS |
| Svelte/TS check | `npm --prefix web run check` | 0 errors, 10 pre-existing warnings | ✓ PASS |
| e2e TS check | `npm --prefix web run check:e2e` | 0 errors | ✓ PASS |
| Go build | `go build ./...` | clean | ✓ PASS |
| Relevant Go test packages | `go test ./kernel/index/... ./kernel/httpapi/...` | both `ok` | ✓ PASS |
| False-claim removal | `grep -rn "no-ops by design" web/e2e/specs docs/testing.md` | no matches | ✓ PASS |
| Phase-modified files free of unresolved debt markers | `grep -n -E "TBD\|FIXME\|XXX\|TODO\|HACK\|PLACEHOLDER"` on `+page.svelte`, the spec, and `docs/testing.md` | none found | ✓ PASS |

### Probe Execution

Not applicable — this phase's verification gate is the Playwright/vitest/Go test suites above, not a `scripts/*/tests/probe-*.sh` convention. No probes declared in any 13-xx plan or SUMMARY.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `web/src/routes/w/[webspace]/+page.svelte` | 132 (`handleExclude`), 155 (`handleInclude`), 236 (`handleBulkPrimary`) | `closeDetail()`/`handleBulkClear()` (and `markBusy`/`bulkBusy` clears) still run unconditionally after the *initial* (non-undo) write's `await`, ungated by `gen === navGeneration` (13-REVIEW.md's re-review, WR-01, carried forward from the prior review's WR-02) | ⚠️ Warning (non-blocking) | UI-only: if a user navigates to a different webspace and opens/selects a different item before the *original* `setItemMarks` round-trip resolves (single-digit ms against the local kernel — far narrower than the 5000ms undo window G-13-1 covered), the origin handler's `closeDetail()`/`handleBulkClear()` fires against the new webspace's UI state, spuriously deselecting/clearing it. No data is written to the wrong webspace — this is a UI state clobber, not a correctness regression on the roadmap's "trivially reversible" data contract. Explicitly out of 13-08's declared scope (G-13-1 only); left open with a concrete fix proposed in 13-REVIEW.md. Recommend a follow-up quick task; does not block phase goal achievement. |
| — | — | (carried forward, unchanged) WR-02 (`kernel/httpapi/marks.go` missing `http.MaxBytesReader`), WR-03 (`validatePluginBinaryName` doesn't reject `,`/`=`), IN-02 (global unscoped keydown listener) | ⚠️/ℹ️ | Untouched by 13-07/13-08; already assessed as non-blocking advisory in the initial verification pass |

None of the above are debt markers (`TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`) in phase-modified files — confirmed via grep, zero hits.

### Human Verification Required

None. The one item carried over from the prior pass (human timing feasibility of the cross-webspace undo window) is resolved — see "Backstop Truth Resolution" above, evidenced by `13-UAT.md` test 1's explicit human confirmation.

### Gaps Summary

No gaps remain against the roadmap's five Success Criteria or against 13-08's own gap-closure must-haves. G-13-1 — the exclude/include undo toast leaving four permanent loading skeletons in the webspace navigated to during the toast's window — is closed, and this pass independently re-derived both the RED failure (against an isolated worktree checkout of the pre-fix commit) and the GREEN pass (against the current source), rather than accepting 13-08-SUMMARY.md's account at face value. The full 139-test Playwright suite and the 1078-test web unit suite were both run once, directly, in this pass, and both passed.

One narrow, non-blocking UI-only race (WR-01 in 13-REVIEW.md's re-review, formerly WR-02) remains open — it does not touch data correctness or any roadmap Success Criterion, was explicitly out of 13-08's scope, and is recommended as a follow-up quick task rather than a phase-blocking gap.

The `.planning/REQUIREMENTS.md` Traceability table is stale (still reads `Gaps Found` for all four Phase 13 requirements) and should be reconciled to `Complete` as part of closing out this phase, now that this pass returns `passed`.

---

_Verified: 2026-08-15T15:45:00Z_
_Verifier: Claude (gsd-verifier)_
