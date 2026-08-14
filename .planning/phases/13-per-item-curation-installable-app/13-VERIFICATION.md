---
phase: 13-per-item-curation-installable-app
verified: 2026-08-14T22:47:31Z
status: gaps_found
score: 4/5 roadmap truths verified cleanly (1 partial)
behavior_unverified: 0
overrides_applied: 0
gaps:
  - truth: "Excluding or including fires an undo toast whose Undo action re-issues the exact mirror write for the same item ids (D-02/D-08, UI-SPEC E3.1) — part of Success Criteria 1 and 2's 'trivially reversible' contract"
    status: partial
    reason: >
      web/src/routes/w/[webspace]/+page.svelte's handleExclude, handleInclude, and
      handleBulkPrimary build each toast's onUndo callback as a closure that reads
      the reactive `webspace` (and `navGeneration`) bindings AT THE TIME THE USER
      CLICKS UNDO, not at the time the toast is created. The toast has a 5000ms
      duration, and the app supports switching webspaces via WebspaceSwitcher
      mid-session, so a user who excludes an item, navigates to a different
      webspace within that window, and then clicks the still-visible "Undo"
      button will silently write the reversal against the WRONG webspace.
      item_marks is keyed (webspace_name, item_id, kind), so the write typically
      matches zero rows in the new webspace (changed: 0); the original exclusion
      in the source webspace is never undone, no error is surfaced, and the toast
      dismisses as if the undo succeeded. This is a real, reproducible defect —
      already identified and documented in 13-REVIEW.md (WR-01), with a concrete
      fix (snapshot `webspace`/`navGeneration` into local consts when the toast is
      created, mirroring every other delayed write path in the same file) — but
      it was left unfixed at phase close.
    artifacts:
      - path: "web/src/routes/w/[webspace]/+page.svelte"
        issue: "handleExclude (:116-135), handleInclude (:137-156), and handleBulkPrimary (:208-235) read `webspace`/`navGeneration` reactively inside their onUndo closures instead of snapshotting them at toast-creation time"
    missing:
      - "Snapshot `const ws = webspace` and `const gen = navGeneration` at the top of handleExclude/handleInclude/handleBulkPrimary, and use `ws`/`gen` (not the reactive bindings) inside each onUndo callback and its own setItemMarks/load calls — exactly the pattern 13-REVIEW.md WR-01 already specifies."
      - "A regression test (unit or e2e) that excludes an item, switches webspace, then fires Undo, and asserts the original webspace's exclusion is actually reversed (or, short of a fix, that a failure is surfaced to the user rather than silently swallowed)."
---

# Phase 13: Per-Item Curation & Installable App Verification Report

**Phase Goal:** The user has the final say over what a webspace shows, and can install topos as an app rather than a browser tab.
**Verified:** 2026-08-14T22:47:31Z
**Status:** gaps_found
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (Roadmap Success Criteria — the contract)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | User excludes an individual item from a webspace and it leaves the stream; the exclusion survives re-sync, kernel restart, and an index rebuild, and always outranks whatever the automatic match rules say. | ⚠️ PARTIAL | Core exclude mechanism VERIFIED: `item_marks` table (`kernel/index/schema.go:91-100`) carries no FK to `items(id)` and is deliberately absent from `rebuildOnSchemaChange`'s drop list, so it survives an index rebuild by construction. `POST /api/webspaces/{ws}/marks` (`kernel/httpapi/marks.go`) writes via `store.SetItemMarks`, bound-parameter SQL (`kernel/index/store.go:406-473`). `StreamItems`/`Search`/agent-mirror all apply `streamMarkFilterClause` (`kernel/index/store.go:358-401`) so exclusion is enforced once, structurally, not per-handler. Backed by direct unit tests: `TestSetItemMarks_IdempotentInsert`, `TestStreamItems_OmitsExcludedItemForItsOwnWebspaceOnly`, `TestDeleteSourceItems_MarkSurvives`, `TestItemMarks_SurviveIndexRebuild`, `TestSetItemMarks_MarkForUnindexedItemOutranksLaterMatch` (`kernel/index/marks_test.go`). **Gap:** the toast "Undo" convenience action that fires immediately after an exclude/include is demonstrably broken across a webspace switch — see gap entry above (`13-REVIEW.md` WR-01, unfixed). |
| 2 | User opens a webspace's excluded-items view, sees exactly what they removed, and un-excludes any of it — the items come back into the stream. | ⚠️ PARTIAL | Direct un-exclude VERIFIED: `?view=excluded` on the stream route (`kernel/httpapi/stream.go`, `parseStreamView`) returns exactly the marked set via `markFilterClauseExcluded`; `excluded_count` rides every stream response. `WebspaceHeader.svelte` renders the `Excluded ({count})` toggle only when `excludedCount > 0` (D-06). The excluded view reuses `StreamList`/`StreamRow`/`DetailPane` wholesale; `DetailPane`'s Include control (only reachable while `excluded=true`) fires `handleInclude`, mirroring `handleExclude`. Backed by `TestStreamItems_ViewExcludedReturnsExactlyMarkedItems`, `TestStreamItems_IncludedAndExcludedViewsAreComplements`, `TestStreamItems_ZeroMarksExcludedViewReturnsEmpty`, `TestClearItemMarks_UnmarkedItemIsNoOp`, and e2e `13-excluded-view.spec.ts`. **Same gap as #1** applies to this criterion's toast-Undo convenience path (bulk include-then-undo). |
| 3 | User installs topos from the browser on the desktop (manifest, ServiceWorker, icons) and launches it as a standalone app window against the local kernel. | ✓ VERIFIED | `web/vite.config.ts` wires `SvelteKitPWA` (`generateSW`/`autoUpdate`, manifest with `standalone` display, app's own background/theme tokens, `pwaAssets` generated from `web/static/app-icon.png`). `cmd/topos/main.go` registers `.webmanifest` → `application/manifest+json` via `mime.AddExtensionType` before serving, independent of the host mime DB. Confirmed by the **human-verify checkpoint** in 13-04-PLAN.md Task 4 step A, approved by the user per 13-04-SUMMARY.md, plus `web/e2e/specs/13-pwa-manifest-sw.spec.ts` (manifest/SW/Cache-Storage assertions). |
| 4 | After the kernel is upgraded, the installed app never serves a stale UI — the user lands on the new build rather than a cached one. | ✓ VERIFIED | `kit.adapterFallback: '200.html'` + `kit.spa: true` gives the SPA fallback a real precache revision (13-RESEARCH Pitfall 3). Workbox `globPatterns` covers the app shell only; `navigateFallbackDenylist: [/^\/api\//]` and no `runtimeCaching` entry keep `/api` traffic live (never precached — Out-of-Scope requirement honored). `registerType: 'autoUpdate'` + `web/src/lib/pwa-update.ts`/`pwa-update.test.ts` drive the reload and the informational toast (`web/src/lib/toast.ts:119`, exact copy `topos updated to the latest version.`). Confirmed by the human-verify checkpoint step B — approved only after two fix rounds (root-route Retry button, automatic update-check-on-focus detection, toast contrast) per 13-04-SUMMARY.md; the checkpoint explicitly re-verified the deep-linked-route case (Pitfall 3's specific risk). |
| 5 | The mobile/LAN install limitation (browsers require a secure context) is documented, along with the recommended user-provided HTTPS workarounds. | ✓ VERIFIED | `README.md:115-147` documents the secure-context requirement, that LAN HTTP install is blocked while the page itself still works over LAN, and points to user-provided HTTPS workarounds; no kernel HTTPS mode is implied (matches `UI-15` being explicitly deferred). Confirmed by human-verify checkpoint step D. |

**Score:** 4/5 roadmap truths cleanly verified; truth 1 and truth 2 share one identified, unfixed correctness gap (Undo-after-navigation) documented above.

### Supporting Must-Haves (from PLAN frontmatter, spot-verified)

The six plans declare dozens of granular `must_haves.truths`. Representative coverage, verified directly against code and tests (not exhaustively enumerated here):

| Must-have (paraphrased) | Plan | Status | Evidence |
|---|---|---|---|
| Exclude idempotent, no duplicate row, `changed=0` on repeat | 13-01 | ✓ VERIFIED | `TestSetItemMarks_IdempotentInsert`; `INSERT OR IGNORE` on the `(webspace_name, item_id, kind)` PK |
| Empty/absent `item_ids` → 400 `invalid_request` naming the field | 13-01 | ✓ VERIFIED | `kernel/httpapi/marks.go:92-95` |
| Un-exclude on unmarked item → 200 `changed=0`, never 404 | 13-01/13-02 | ✓ VERIFIED | `TestClearItemMarks_UnmarkedItemIsNoOp` |
| Mark filter never reorders the stream | 13-01 | ✓ VERIFIED | `TestStreamItems_ExcludedItemOrderingPreservedAmongSurvivors` |
| Exclusion applies identically to stream, in-webspace search, `/agent/v1` mirror | 13-01/13-02 | ✓ VERIFIED | `TestSearch_OmitsExcludedItemForItsOwnWebspaceOnly`; `kernel/httpapi/agent_test.go`; shared `streamMarkFilterClause` used by all three call sites |
| Healthy-sync-only prune sweep scoped to `(webspace, source)`, unreachable on a failed sync | 13-02 | ✓ VERIFIED | `TestReplaceWebspaceSourceItems_OmittedExcludedItemIsPruned`, `_SweepNeverTouchesSiblingSourceMarks`, `_SweepNeverTouchesOtherWebspaceMarks`, `_InterruptedSyncLeavesItemsAndMarksUnchanged` |
| `?view=` invalid value → 400 naming allowed values; empty-marks excluded view → 200 empty, not 404 | 13-02 | ✓ VERIFIED | `parseStreamView`; `TestStreamItems_ZeroMarksExcludedViewReturnsEmpty` |
| Ctrl/Cmd-click toggles bulk selection without opening detail pane; shift-click replaces range; plain click doesn't clear selection | 13-03 | ✓ VERIFIED | `StreamRow.svelte:79-95` `handleActivate`; `web/src/lib/selection.ts` `toggleSelection`/`selectRange` + `selection.test.ts` |
| Action bar absent at 0, buttons disable in-flight | 13-03 | ✓ VERIFIED | `SelectionActionBar.svelte:9,38-39`; `selection-action-bar.test.ts` |
| Excluded toggle absent (not disabled) at count 0, auto-flips back at 0 | 13-03 | ✓ VERIFIED | `WebspaceHeader.svelte:578`; `handleToggleView` reset path in `+page.svelte` |
| PWA precaches shell only, never `/api` | 13-04 | ✓ VERIFIED | `vite.config.ts` `globPatterns`/`navigateFallbackDenylist` |
| Kernel-down still renders app's own error state, not a browser error page | 13-04 | ✓ VERIFIED | Human-verify checkpoint step C (fixed a missing Retry button, then approved) |
| Trust-tier status derives from link-time build-provenance manifest, not directory | 13-05 | ✓ VERIFIED | `kernel/pluginhost/manifest.go` (`buildManifest`, `-ldflags -X`), `VerifyTrustedBinary`; `kernel/pluginhost/manifestgate_test.go` |
| Unverified trusted-dir binary refuses to launch, named failure `manifest_unverified`, no silent directory fallback | 13-05 | ✓ VERIFIED | `kernel/pluginhost/host.go:67,144-176,888-930`; e2e `13-manifest-unverified.spec.ts` |
| Shadowing advisory surfaced as structured per-source field, never suppresses a real failure | 13-05/13-06 | ✓ VERIFIED | `SourceHealth.LaunchAdvisory`; `SourceChip.svelte` `isShadowed` (warning tone, never destructive); `source-chip-tooltip.test.ts`; e2e `13-shadowed-advisory.spec.ts` |
| Manifest-unverified chip: destructive tone, contract-exact tooltip, no re-pin action | 13-06 | ✓ VERIFIED | `SourceChip.svelte` `isManifestUnverified`; `match-advisory.test.ts` precedence matrix |

Two `verification: backstop` truths (13-01, 13-03, 13-04 — undo-copy-out-of-context, narrow-desktop-width fit, multi-window-update independence) are explicitly non-inferable from code inspection alone and are correctly left unverified by design; none surfaced as a problem during the human-verify checkpoints that did exercise adjacent behavior.

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `kernel/index/schema.go` | `item_marks` table, no FK, absent from rebuild-drop list | ✓ VERIFIED | Present, documented with load-bearing comment |
| `kernel/index/store.go` | `SetItemMarks`/`ClearItemMarks`/`CountItemMarks`/`pruneItemMarksTx`/`MarkView`/`StreamItems` | ✓ VERIFIED | All present, bound-parameter SQL throughout |
| `kernel/httpapi/marks.go` | `POST /api/webspaces/{ws}/marks` | ✓ VERIFIED | Full validation chain, wired to `store.SetItemMarks`/`ClearItemMarks` |
| `kernel/httpapi/stream.go` | `?view=` param, `excluded_count` | ✓ VERIFIED | `parseStreamView`, threaded into `StreamItems` |
| `web/src/lib/toast.ts` | Undo/success/failure/update toast helpers | ✓ VERIFIED — **but wired with the reactive-closure bug** | See gap above |
| `web/src/lib/selection.ts` | Pure bulk-selection helpers | ✓ VERIFIED | `toggleSelection`, `selectRange`, `clearSelection`, tested |
| `web/src/lib/components/SelectionActionBar.svelte` | Multi-select action bar | ✓ VERIFIED | Zero/one/many rendering, disable-in-flight |
| `web/vite.config.ts` | `SvelteKitPWA` config | ✓ VERIFIED | Shell-only precache, manifest, autoUpdate |
| `cmd/topos/main.go` | manifest MIME registration | ✓ VERIFIED | `registerManifestMimeType` |
| `kernel/pluginhost/manifest.go` | Link-time build-provenance manifest | ✓ VERIFIED | `buildManifest`, `VerifyTrustedBinary`, `ParseManifest` |
| `cmd/topos-manifest/main.go` | Manifest generator | ✓ VERIFIED | Wired into `Makefile` build/dev/e2e recipes |
| `docs/api.md` | Marks + `?view=` endpoint docs | ✓ VERIFIED | Documented with curl examples and error table entries |
| `docs/plugin-contract.md`, `docs/plugins/signal.md` | Trust-authority republish | ✓ VERIFIED | Manifest named as trust authority; hashing framed as integrity, not authentication |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `correlate.SyncSource`'s healthy-pair call | `ReplaceWebspaceSourceItems` prune sweep | Call-site scoping | ✓ WIRED | `pruneItemMarksTx` runs only inside the sync's own transaction |
| Stream route `?view=` | `StreamItems`'s IN/NOT IN branch | `index.MarkView` | ✓ WIRED | `parseStreamView` → `streamMarkFilterClause` |
| `excluded_count` in stream response | `WebspaceHeader`'s toggle visibility | Prop threading | ✓ WIRED | `WebspaceHeader.svelte:578` `{#if excludedCount > 0}` |
| `StreamRow` click-handler branch | Route's selection state | `handleBulkToggle` | ✓ WIRED | `+page.svelte:184-193` |
| `SvelteKitPWA` `kit.adapterFallback`+`kit.spa` | `200.html` precache revision | Build config | ✓ WIRED | Confirmed live via human checkpoint (deep-link reload case) |
| `mime.AddExtensionType` | `http.FileServer` Content-Type | Startup hook | ✓ WIRED | `cmd/topos/main.go:178` |
| `virtual:pwa-register` autoUpdate callback | Update toast | `pwa-update.ts` | ✓ WIRED | Confirmed live via human checkpoint, after adding automatic update-check-on-focus |
| `sourceStatus.launch_advisory` | `SourceChip` dot tone/tooltip | `format.ts` precedence chain | ✓ WIRED | Single chain, no parallel gate (per WR review's own confirmation) |
| Toast `onUndo` closure | `setItemMarks(webspace, ...)` | Reactive `webspace` binding | ⚠️ **MISWIRED under navigation** | Reads `webspace`/`navGeneration` at click-time, not toast-creation time — see gap |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Go test suite (index/httpapi/pluginhost/syncer/supervisor/correlate/cmd) | `go test ./kernel/index/... ./kernel/httpapi/... ./kernel/pluginhost/... ./kernel/syncer/... ./kernel/supervisor/... ./kernel/correlate/... ./cmd/topos/...` | all `ok` | ✓ PASS |
| Full Go build | `go build ./...` | clean | ✓ PASS |
| Web unit test suite | `npx vitest run` (web/) | 58 files / 1078 tests passed | ✓ PASS |
| Named mark-idempotency/prune/rebuild-survival tests exist and pass | enumerated in `kernel/index/marks_test.go` | 18 relevant tests present | ✓ PASS |
| Phase-modified files free of debt markers (TBD/FIXME/XXX/TODO/HACK/PLACEHOLDER) | `grep` across key changed files | none found | ✓ PASS |

Full `make e2e` (135 Playwright specs) was not re-run standalone in this verification pass — accepted on the strength of the phase's own documented pass plus the two independently-reproduced human-verify checkpoint rounds; the Go and web unit suites and a fresh `go build ./...` were re-run directly as the automated corroboration.

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|---|---|---|---|---|
| KERN-09 | 13-01, 13-02, 13-03 | Exclude an item; survives re-sync/rebuild/restart; outranks match rules | ✓ SATISFIED (with an open, unfixed convenience-path defect — see gap) | Schema, store, handler, tests as above |
| KERN-10 | 13-01, 13-02, 13-03 | View and un-exclude excluded items | ✓ SATISFIED (same open defect on the Undo convenience path) | `?view=excluded`, excluded-view UI, Include control |
| UI-13 | 13-04 | PWA install + never-stale update flow | ✓ SATISFIED | Human-verify checkpoint approved (two fix rounds) |
| UI-14 | 13-04 | Mobile/LAN limitation documented | ✓ SATISFIED | README section, checkpoint step D |
| PLUG-07 (extra, not a phase-13 requirement ID but claimed by 13-05/13-06) | 13-05, 13-06 | Trust derives from provenance, never plugin self-declaration | ✓ SATISFIED — hardens Phase 11's already-Complete requirement with the link-time manifest gate; no regression to Phase 11 external-tier behavior found | `manifest.go`, `manifestgate_test.go`, e2e `13-manifest-unverified.spec.ts`/`13-shadowed-advisory.spec.ts` |

No orphaned requirements: REQUIREMENTS.md maps exactly KERN-09, KERN-10, UI-13, UI-14 to Phase 13, and all four are claimed by at least one plan's `requirements` field.

**Note:** `.planning/REQUIREMENTS.md` still shows KERN-09/KERN-10 as unchecked `[ ]` with traceability status "Pending", while UI-13/UI-14 are checked `[x]`/"Complete" — this file has not yet been updated to reflect Phase 13's actual completion state for the per-item-curation half. Not itself a code gap, but should be reconciled when this phase closes.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `web/src/routes/w/[webspace]/+page.svelte` | 116-235 | Reactive-state closure captured at async-callback-fire-time instead of creation-time (WR-01) | 🛑 Blocker for a clean pass (narrow real-world trigger, silent failure, already identified+unfixed) | Undo silently targets the wrong webspace after a webspace switch; no error surfaced |
| `kernel/httpapi/marks.go` | 78-100 | No `http.MaxBytesReader` before JSON decode; the 1000-id cap is enforced only after the full body is materialized (WR-02) | ⚠️ Warning | Doc comment overstates the DoS mitigation as fully closed; pre-existing pattern across the package, low practical risk on a loopback-only kernel |
| `kernel/pluginhost/manifest.go` / `discover_binaries.go` | 83-117 / 326-335 | `validatePluginBinaryName` doesn't reject `,`/`=`, the manifest wire-format's own separators (WR-03) | ℹ️ Info | Only reachable via a hand-renamed binary filename; fails closed (empty manifest) rather than open, but with no operator-facing diagnostic |
| `web/src/routes/w/[webspace]/+page.svelte` | 254-262 | Global unscoped keydown listener for Esc-clears-selection (IN-02) | ℹ️ Info | Documented as intentional in code review; no functional issue |

All four items were originally surfaced by the phase's own committed `13-REVIEW.md` (0 critical / 3 warning / 2 info) and independently re-confirmed by direct code reading during this verification. WR-02/WR-03/IN-02 are advisory and don't block phase completion. WR-01 is elevated to a phase gap here because it demonstrably breaks a stated `must_haves` truth under a realistic, supported usage pattern (switching webspaces), with no user-facing error to compensate.

### Human Verification Required

None outstanding — the phase's blocking human-verify checkpoint (13-04-PLAN.md Task 4: real PWA install, real kernel upgrade, real kernel-down launch, README read-through) was already completed and approved by the user after two fix rounds, per 13-04-SUMMARY.md. No further human verification items were identified by this pass.

### Gaps Summary

Four of five roadmap success criteria are cleanly and directly verified against code, tests, and (for the two criteria that need a real browser/real kernel restart) an already-completed, approved human-verify checkpoint. The kernel-side exclude/include mechanism (marks table, prune sweep, view filtering, stream/search/agent parity) is solid, well-tested, and matches its `must_haves` truths precisely — this is the strongest part of the phase.

The one real gap is narrow but genuine: the toast "Undo" convenience action for exclude/include (single-item and bulk) captures the target webspace reactively rather than as a snapshot, so clicking "Undo" after switching webspaces silently writes the reversal against the wrong webspace and never surfaces an error. This was already found and precisely diagnosed by the phase's own code review (`13-REVIEW.md` WR-01) — including the exact fix — but was left unresolved when the phase was marked complete. It does not block the core exclude/un-exclude mechanism (which works correctly via direct clicks in the excluded view), only the toast's "Undo" shortcut across a specific, realistic navigation sequence.

Recommended resolution: either (a) apply the already-specified snapshot fix (a small, well-scoped change touching three functions in one file, with a regression test), or (b) explicitly accept this as a known limitation via a VERIFICATION.md override if the team judges the narrow trigger condition acceptable for this milestone.

---

_Verified: 2026-08-14T22:47:31Z_
_Verifier: Claude (gsd-verifier)_
