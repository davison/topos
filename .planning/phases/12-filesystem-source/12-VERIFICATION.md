---
phase: 12-filesystem-source
verified: 2026-08-14T11:05:00Z
status: gaps_found
score: 10/11 must-have truths verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: passed
  previous_score: "7/7"
  gaps_closed:
    - "G-12-1/G-12-3 missing item 1: 'everything from this instance' is now expressible — folderLabels prepends the configured root's own base name to every file at every depth (12-08)"
    - "G-12-1/G-12-3 missing item 2: a zero-match state across a healthy sync is no longer silent — kernel-composed last_notice travels correlate -> sync_runs -> GET /api/sources (12-09)"
    - "G-12-1/G-12-3 missing item 3 (partial): the advisory reaches the browser — healthTone gains a warning branch and SourceChip.svelte gains a tooltip branch naming the advisory; MatchFieldsForm.svelte states match values are exact and wildcards are unsupported (12-10)"
  gaps_remaining: []
  regressions:
    - "12-10's own tooltipText advisory branch introduces a NEW defect (not present before this gap-closure round): it gates only on last_status !== 'error', never on source.reachable/tone, so a currently-unreachable source that still carries a stale successful last_status and a leftover last_notice renders a benign 'synced ... — advisory' tooltip instead of 'unreachable since ...' — masking a real connectivity problem behind a benign advisory. This is CR-01 from the fresh 12-REVIEW.md (gap-closure diff review), independently reconfirmed against current HEAD by this verifier."
  note: "This is the third re-verification of Phase 12. The prior pass (2026-08-14T02:30:00Z) scored the phase 7/7 and passed before UAT (12-UAT.md) surfaced G-12-1/G-12-3: the user's real config (folders = ['*']) could never match, and the resulting empty-but-healthy state was silent at every layer. Gap-closure plans 12-08, 12-09 and 12-10 genuinely close both named missing: items for the ORIGINAL failure mode (match-all is now expressible and documented; a zero-match state is now diagnosed and published as last_notice). This re-verification independently re-ran the backend Go suites, the frontend vitest suites, and the two new e2e specs (chromium; firefox/webkit failed only on a missing local browser-binary install, not a code defect) — all pass. However, a fresh code review of the gap-closure diff (12-REVIEW.md, requested alongside this re-verification) found one Critical, currently-unfixed defect (CR-01) introduced by 12-10's own new tooltip branch, which this verifier independently confirmed by reading the current source. CR-01 is a new instance of the exact failure class this gap-closure round exists to eliminate — a chip surface that reads more reassuring than the truth — just for a different precondition (reachable:false) than the one UAT reported. It also violates 12-10-PLAN.md's own stated must-have ('an unreachable source ... keeps the tone and copy it had before this plan') and its own threat-model mitigation for T-12-46. Recorded as a new gap rather than accepted as non-blocking."
gaps:
  - truth: "12-10's own must-have: 'An advisory never outranks a real problem: a pin mismatch, an unreachable source, a never-synced source and an errored sync all keep the tone and copy they had before this plan.'"
    status: failed
    reason: "SourceChip.svelte's tooltipText derivation gates its new advisory branch on `advisory !== '' && source.last_status !== 'error'` only — it does not check `source.reachable` or the already-computed `tone` value. format.ts's healthTone correctly returns 'destructive' when !source.reachable, ahead of the last_notice check, so the health DOT is unaffected. But the TOOLTIP TEXT is not gated the same way: a source with reachable:false, last_status:'ok' (stale, from before it went unreachable) and a non-empty leftover last_notice hits the advisory branch and renders '{display_name} — synced {relative} — {advisory}', never mentioning the source is currently unreachable — instead of the destructive-tone branch's '{display_name} — unreachable since {relative}'. This is the tooltip — the one place a user reads *why* the dot is red — actively hiding that fact. Confirmed present in the code at HEAD (commit 9083067); no fix has landed since the fresh code review (12-REVIEW.md CR-01) that first found it."
    artifacts:
      - path: "web/src/lib/components/SourceChip.svelte"
        issue: "tooltipText's advisory branch (around line 205) checks `source.last_status !== 'error'` instead of the already-`$derived` `tone` value; `reachable` is never consulted"
    missing:
      - "Gate the advisory branch on `tone === 'success'` (the value SourceChip.svelte already computes via `healthTone(source)`) rather than `last_status !== 'error'`, so the same precedence rule expressed once in healthTone cannot drift from the tooltip's own copy"
      - "A regression test exercising `reachable: false` + `last_status: 'ok'` + non-empty `last_notice` against the actual rendered/derived `tooltipText`, not just against `healthTone`'s tone — match-advisory.test.ts's existing `reachable: false` case (line 91-92) only asserts `healthTone(...) === 'destructive'`; it never exercises which tooltipText branch is chosen for that same input, which is exactly why this defect shipped past a review-driven guard file whose entire purpose was this precedence"
deferred: []
behavior_unverified_items: []
---

# Phase 12: Filesystem Source Verification Report (Re-verification, after 12-08/12-09/12-10 gap-closure waves)

**Phase Goal:** The user can point topos at a folder — local or on a network mount — and see its documents in the right webspace.
**Verified:** 2026-08-14T11:05:00Z
**Status:** gaps_found
**Re-verification:** Yes — after UAT (12-UAT.md) reopened the phase with G-12-1/G-12-3, and gap-closure plans 12-08, 12-09, 12-10 executed and merged. A fresh code review of that gap-closure diff (12-REVIEW.md) was requested alongside this verification and its Critical finding (CR-01) is independently confirmed here.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | User adds a folder as a source from the UI, recursion on/off, documents appear in the matching webspace stream with previews (Roadmap SC1) | ✓ VERIFIED | Unchanged core path from the prior `passed` verification; regression-confirmed by the full `plugins/filesystem` suite passing (`CGO_ENABLED=0 go test ./plugins/filesystem/... -count=1` → ok) and by `12-08`'s label change not perturbing any pre-existing filesystem e2e spec |
| 2 | Files added/changed/removed reflected on next sync, including NFS/SMB mounts (Roadmap SC2) | ✓ VERIFIED | Unchanged, untouched by 12-08/12-09/12-10's `files_modified` lists; `TestWalk_*` suite passes |
| 3 | Every filesystem item deep-links back to the desktop's own file handler, or declares honestly it can only raise (Roadmap SC3) | ✓ VERIFIED | Unchanged, `kernel/httpapi/fsopen.go` untouched by this gap-closure round |
| 4 | The plugin never writes to the source folder — enforced by committed guards (Roadmap SC4) | ✓ VERIFIED | `TestPluginIssuesNoWrite` (AST scan) re-run, passes; file untouched by 12-08/12-09/12-10 |
| 5 | The filesystem binary loads/syncs identically from the external plugins directory, external badge shown (Roadmap SC5) | ✓ VERIFIED | Unchanged, `web/e2e/specs/12-external-rehearsal.spec.ts` untouched by this round |
| — | (Cross-cutting) MUST NOT index/serve/preview/open any file outside the configured source root | ✓ VERIFIED | Unchanged from prior pass; `resolvePath`/`FilesystemOpenHandler` untouched by 12-08/12-09/12-10 |
| 6 | **G-12-1/G-12-3, missing item 1**: "Everything from this instance" is expressible — naming the configured root's own base name in `folders` matches every file that source contributes, at every depth | ✓ VERIFIED | `plugins/filesystem/item.go`'s `folderLabels` now prepends the root base name to every file's label set via a new `dedupeLabels` helper. Independently re-ran: `TestFolderLabels_NestedFileAlsoCarriesTheRootBaseName`, `TestFolderLabels_RootBaseNameEqualToASubfolderSegmentIsNotDuplicated`, `TestFolderLabels_NoLabelNamesADirectoryAboveTheConfiguredRoot`, and the extended `TestFolderLabels_SubdirectoryFileIsContainingDirectoryBaseName` — all PASS. `web/e2e/specs/12-filesystem-root-label-match.spec.ts` exists and PASSES (chromium; see Behavioral Spot-Checks) proving the root-base-name value matches every depth while the user's literal `folders = ['*']` value matches nothing |
| 7 | Docs state match values are exact literals, never globs, on the same page/block that documents this plugin's real doublestar glob keys | ✓ VERIFIED | `grep -n 'never as glob patterns\|everything from this instance' docs/plugins/filesystem.md config.example.toml` finds both phrases in both files; `bash scripts/check-doc-links.sh` referenced in the plan's own gates |
| 8 | **G-12-1/G-12-3, missing item 2**: an explicit match block that matched zero items across a completed healthy sync is named rather than silent | ✓ VERIFIED | `kernel/correlate/correlate.go`'s `zeroMatchNotice` + `WebspaceResult.Notice`, persisted via `kernel/index`'s new `sync_runs.notice` column (schemaVersion 2→3) and `FinishSyncRunWithNotice`, aggregated deterministically in `kernel/syncer/coordinator.go`'s `joinNotices`, published as `last_notice` on `GET /api/sources`. Independently re-ran all 12 new/modified tests across `kernel/correlate`, `kernel/index`, `kernel/syncer`, `kernel/httpapi` — all PASS, including the plugin-agnostic sub-case and the never-masks-a-real-error case |
| 9 | The notice is kernel-composed only — no plugin-provided string ever enters it (A-PLUG-04) | ✓ VERIFIED | `zeroMatchNotice(webspace string, fields map[string][]string)` — signature takes only a webspace name and the resolved field map from configuration; `awk`-scoped grep for `GetItems\|protoItem\|it\.SourceID\|src\.Name` inside the function returns 0 matches |
| 10 | **G-12-1/G-12-3, missing item 3**: the advisory is documented as additive, kernel-owned, not-an-error, at `schema_version` 1 | ✓ VERIFIED | `docs/api.md` line 528 example, line 560 field bullet, line 549/568/574 A-PLUG-04/do-not-parse discipline — all present; `grep -c '"schema_version": 2'` returns 0 |
| 11 | **G-12-1/G-12-3, missing item 3 (browser surface)**: the advisory reaches the source chip as a visible signal, AND never outranks a bigger, real problem (an unreachable source, pin mismatch, never-synced, errored) | ✗ FAILED | The advisory DOES reach the chip: `healthTone` gains a warning branch (`web/src/lib/format.ts:144`), `SourceChip.svelte` gains a tooltip branch, `MatchFieldsForm.svelte` states values are exact/no-wildcards — all confirmed present and passing their own unit tests. But the "never outranks a real problem" half of this same must-have FAILS for the unreachable case: see Gaps below (CR-01) |

**Score:** 10/11 truths verified. Both named `missing:` items from G-12-1/G-12-3's diagnosis (match-all inexpressible; zero-match silence) are genuinely closed at the mechanism level. The one FAILED truth is a narrower, newly-introduced defect in the visibility surface itself (12-10's own tooltip precedence), not a regression of the original reported failure.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `plugins/filesystem/item.go` | `folderLabels` prepending root base name at every depth, `dedupeLabels` | ✓ VERIFIED | Read in full; `dedupeLabels` present once, wired into `folderLabels`'s return path |
| `plugins/filesystem/item_test.go` | New/extended `TestFolderLabels_*` cases | ✓ VERIFIED | 3 new + 1 extended case present and passing |
| `docs/plugins/filesystem.md`, `config.example.toml` | Exact-literal statement + "everything from this instance" recipe | ✓ VERIFIED | Both phrases present in both files |
| `web/e2e/specs/12-filesystem-root-label-match.spec.ts` | Positive/negative match-value proof against a real kernel+plugin | ✓ VERIFIED | File exists; independently re-run (chromium) — PASS |
| `kernel/correlate/correlate.go` | `WebspaceResult.Notice`, `zeroMatchNotice`, `matchFieldsFor`'s `explicit` result | ✓ VERIFIED | Read in full; matches plan's described shape |
| `kernel/index/schema.go`, `store.go` | `sync_runs.notice`, schemaVersion 3, `FinishSyncRunWithNotice` | ✓ VERIFIED | `schemaVersion = 3` present; `FinishSyncRun` delegates to the new sibling |
| `kernel/syncer/coordinator.go` | `joinNotices`, `maxJoinedNotices`, `RunResult.Notice` | ✓ VERIFIED | Sorted, bounded join confirmed by passing `TestSyncOne_NoticesFromSeveralWebspacesJoinSortedAndBounded` |
| `kernel/httpapi/sources.go` | `last_notice` json field on `sourceStatus`, populated in the probe-derived loop | ✓ VERIFIED — with a noted gap (WR-01) | `LastNotice: run.Notice` present in the probe-derived (`healths`) loop (line 180); **absent from the launch-failure loop** (line 185-216) — matches 12-REVIEW.md's WR-01 exactly. Currently non-user-visible because `isPinMismatch`/`launch_failure` precedence already wins in the UI regardless — recorded as a non-blocking Warning, not a gap, per the review's own assessment |
| `docs/api.md` | `last_notice` documented, additive, schema_version unchanged | ✓ VERIFIED | 3+ mentions found in the `GET /api/sources` section |
| `web/src/lib/api.ts` | Optional `last_notice?: string` on `SourceStatus` | ✓ VERIFIED | Present, optional (existing fixtures untouched) |
| `web/src/lib/format.ts` | `healthTone`'s advisory branch, reusing `warning`, no fifth tone | ✓ VERIFIED | Branch present at line 144, ordered after the `!reachable` check (line 142) and the errored-status check — precedence correct at the TONE level |
| `web/src/lib/components/SourceChip.svelte` | Fifth tooltip branch carrying the advisory, gated so a real problem's copy is never displaced | ✗ **STUB PRECEDENCE** | Branch present (line 200-220) but its gate (`advisory !== '' && source.last_status !== 'error'`) does not consult `reachable`/`tone` — see Gaps. The health DOT is unaffected (correct, via `healthTone`); the TOOLTIP TEXT is wrong for the `reachable:false` + `last_status:'ok'` + notice combination |
| `web/src/lib/components/MatchFieldsForm.svelte` | Helper text: values matched exactly, wildcards unsupported | ✓ VERIFIED | Line 63-64: "Values are matched exactly, case-insensitively — wildcards and glob patterns are not supported." |
| `web/src/lib/components/match-advisory.test.ts` | Precedence matrix + structural guards | ✓ VERIFIED (but incomplete coverage) | 38 assertions pass; however its `reachable: false` case (line 91-92) only asserts `healthTone`'s tone, never which `tooltipText` branch is chosen — the exact gap CR-01 lives in was left unguarded by this file's own precedence matrix |
| `web/e2e/specs/12-zero-match-diagnostic.spec.ts` | Browser-level gate over the user's exact failing config | ✓ VERIFIED | File exists; independently re-run (chromium) — PASS (both tests) |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `plugins/filesystem/item.go` (`folderLabels`) | `plugins/filesystem/plugin.go` (`labelMatchesAny`) | label set attached to every Item | ✓ WIRED | `labelMatchesAny`'s `strings.EqualFold` comparison confirmed untouched (exactly 1 occurrence) |
| `kernel/correlate/correlate.go` (`SyncSource`) | `kernel/syncer/coordinator.go` (`syncOne`) | `WebspaceResult.Notice` | ✓ WIRED | `joinNotices` collects every non-empty notice, sorted and bounded |
| `kernel/syncer/coordinator.go` (`syncOne`) | `kernel/index` (`sync_runs.notice`) | `FinishSyncRunWithNotice`, same detached finalize write as status/error | ✓ WIRED | Confirmed via `TestSyncOne_ZeroMatchNoticeLeavesStatusOKAndErrorEmpty` |
| `kernel/index/store.go` (`LatestSyncRunPerSource`) | `kernel/httpapi/sources.go` (`sourceStatusesFrom`) | `run.Notice` → `LastNotice` | ✓ WIRED (probe-derived loop only) | Launch-failure loop omits it (WR-01, non-blocking) |
| `kernel/httpapi/sources.go` (`last_notice`) | `web/src/lib/format.ts` (`healthTone`) | typed `SourcesResponse` field | ✓ WIRED | `TestSources_ZeroMatchNoticeTravelsFromCorrelateToTheSourcesAPI` proves the wire tag; frontend `SourceStatus.last_notice?` mirrors it |
| `web/src/lib/format.ts` (`healthTone`) | `web/src/lib/components/SourceChip.svelte` (dot tone) | `DOT_TONE_CLASS[tone]` | ✓ WIRED, correct precedence | Dot tone is correct in every precedence case, including `reachable:false` |
| `web/src/lib/format.ts` (`healthTone`)/`tone` | `web/src/lib/components/SourceChip.svelte` (`tooltipText`'s advisory branch) | expected: the same `tone` value the dot already computes | ✗ **NOT WIRED — this is CR-01** | `tooltipText`'s advisory branch does NOT read `tone`; it re-derives its own, narrower condition (`last_status !== 'error'`) instead of reusing the precedence `healthTone` already computed one line above it in the same file |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `folderLabels` root-base-name behavior (12-08) | `CGO_ENABLED=0 go test ./plugins/filesystem/... -run 'TestFolderLabels' -count=1 -v` | 5/5 tests PASS | ✓ PASS |
| Zero-match notice detection/aggregation/persistence/publication (12-09) | `go test ./kernel/correlate/ ./kernel/index/ ./kernel/syncer/ ./kernel/httpapi/ -run 'ZeroMatch|Notice' -count=1 -v` | all named tests PASS (11 test functions, incl. subtests) | ✓ PASS |
| Frontend advisory/tooltip/match-hint guards (12-10) | `cd web && npx vitest run src/lib/components/match-advisory.test.ts src/lib/components/match-values-hint.test.ts src/lib/components/source-chip-tooltip.test.ts` | 3 files, 38 tests PASS | ✓ PASS (but see coverage gap noted above — these tests do not catch CR-01) |
| Root-label + zero-match e2e specs, real kernel + real plugin binary | `cd web && npx playwright test e2e/specs/12-filesystem-root-label-match.spec.ts e2e/specs/12-zero-match-diagnostic.spec.ts` | chromium: 3/3 PASS; firefox/webkit: 2 failures — `Executable doesn't exist` (browser binaries not installed in this sandbox, not a code failure) | ✓ PASS (chromium); ⚠ environment-limited on firefox/webkit |
| `docs/plugins/filesystem.md` / `config.example.toml` exact-literal statements | `grep -n 'never as glob patterns\|everything from this instance' docs/plugins/filesystem.md config.example.toml` | both phrases present in both files | ✓ PASS |
| `docs/api.md` `last_notice` documentation | `grep -n 'last_notice' docs/api.md` | 5 occurrences: example JSON, field bullet, A-PLUG-04 sentence, do-not-parse discipline | ✓ PASS |
| **CR-01 reproduction (independent, this verification)** | Read `web/src/lib/components/SourceChip.svelte` lines 189-220 and `web/src/lib/format.ts` lines 132-146 at current HEAD (commit `9083067`) | `tooltipText`'s advisory branch condition is `advisory !== '' && source.last_status !== 'error'` — no `reachable`/`tone` check; `healthTone` itself correctly checks `!source.reachable` first (line 142) | ✗ CONFIRMS CR-01 IS PRESENT AND UNFIXED |
| WR-01 reproduction (independent, this verification) | Read `kernel/httpapi/sources.go` lines 185-216 | launch-failure branch's `sourceStatus{}` literal has no `LastNotice:` line, unlike the probe-derived loop's `LastNotice: run.Notice` (line 180) | ✓ CONFIRMS WR-01 IS PRESENT (non-blocking per review's own precedence analysis, independently re-confirmed here) |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| SRC-04 | 12-01 through 12-10 (all declare `requirements: [SRC-04]`) | User can add a local/network filesystem folder as a source; documents appear with previews and deep links, synced via stat-diff polling | ⚠ SATISFIED AT THE MECHANISM LEVEL, ONE GAP OPEN | The original UAT-reported failure (documents not appearing for the user's real config; the silent healthy-empty state) is genuinely fixed: match-all is expressible (12-08), the zero-match state is diagnosed and published (12-09), and the diagnosis reaches the chip and the match-value form (12-10). The one open gap (CR-01) is narrower than the original failure — it concerns tooltip copy precision for a *different* precondition (`reachable:false`) than what UAT reported — but it is a real, currently-shipped defect in the same "trustworthy health signal" surface this gap-closure round exists to build, and it directly contradicts 12-10-PLAN.md's own must-have. `REQUIREMENTS.md` line 78 still records SRC-04 as "Gaps Found" — this verification's result (`gaps_found`) confirms that mark should NOT yet be updated to Complete |

No orphaned requirements found — SRC-04 is the only requirement mapped to Phase 12, and no plan under this phase claims a requirement ID absent from REQUIREMENTS.md's Phase 12 mapping.

### Anti-Patterns Found

None. No debt markers (`TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`) found in any of the 21 files modified across 12-08/12-09/12-10. CR-01 and WR-01 are logic/precedence defects, not debt markers or stub patterns.

### Related, Non-Blocking Findings (carried from 12-REVIEW.md, independently re-confirmed)

**WR-01 — `sourceStatusesFrom` doesn't populate `LastNotice` for launch-failed (pin-mismatch) merged entries.** Confirmed present at `kernel/httpapi/sources.go:185-216`. Currently masked in the UI because `isPinMismatch`/`launch_failure === 'pin_mismatch'` already takes precedence over the advisory in every consuming surface (both `healthTone` and, once CR-01 is fixed, `tooltipText`), so this is not user-visible today — but it is an inconsistency between what the merge claims to expose from `run` (`LastStatus`/`LastSyncUnix` do come from it) versus what it actually copies (`LastNotice` does not). Recorded as a Warning, not a gap.

**IN-01 — `zeroMatchNotice`'s non-empty-fields guard doesn't catch a field present with an empty value list.** Cosmetic-only per the review's own analysis (an oddly-formatted message, not a functional defect), and its reachability against `config.Validate` is unconfirmed. Recorded as Info, not a gap.

**Stray untracked build binary.** `plugins/filesystem/filesystem`, an untracked ELF binary in the plugin source directory, remains present (`git status --short` at the start of this session). Hygiene note only, unrelated to this phase's commits.

### Human Verification Required

None of the FAILED truth (CR-01) requires human judgment — it is a directly observable code defect (confirmed by reading the current source and its condition logic), not a matter of taste or an ambiguous UX call. It is recorded as a gap, not routed here.

The following items are genuinely **backstop** (`verification: backstop`) truths from 12-08/12-09/12-10's own `must_haves` — they can only be confirmed on the user's own desktop, and this verifier has no access to it. They are informational, not blocking, and do not affect the `gaps_found` status above (which is driven entirely by CR-01):

1. **12-08 backstop: the user's real `~/Documents/Lucid` source, re-saved with the folder's base name as the match value, produces documents in the webspace stream.**
   Expected: after re-saving their real webspace config with the corrected match value, the user's own filesystem documents appear.
   Why human: this is the user's own machine and their own config file; the e2e spec proves the mechanism on a synthetic corpus, but the user's own re-save is theirs to perform.

2. **12-09 backstop: the advisory appears against the user's real `files` source after the next sync, naming webspace `test` and the value they typed.**
   Expected: `GET /api/sources` on the user's own kernel shows a `last_notice` naming their real webspace and their real mistyped value.
   Why human: requires the user's live kernel and their live (still-uncorrected, if they haven't fixed it yet) config.

3. **12-10 backstop: the `files` chip in webspace `test` shows the warning tone and names the zero-matching value on the user's own desktop, and after correcting the value, documents appear with previews and a working desktop-handler open action — including on a network mount.**
   Expected: visually confirmed amber chip with the diagnostic text, then a working preview/open/NFS-mount flow after the fix.
   Why human: previews, the real `xdg-open` handoff and NFS/SMB behavior are explicitly scoped out of the hermetic browser harness by `docs/testing.md`'s own design (matching the same backstop item carried since the original 12-05 tracer).

### Gaps Summary

Both `missing:` items G-12-1/G-12-3's diagnosis named are genuinely closed at the mechanism level: "everything from this instance" is now a typeable, documented, tested match value (12-08); a healthy sync that matched zero items is no longer silent — it is named, kernel-composed, bounded, and published as `last_notice` all the way to `GET /api/sources` (12-09); and that fact now reaches the browser via a warning-tone chip and a form-level hint (12-10). Every backend test this verifier independently re-ran (Go and vitest) passes, and both new e2e specs pass on chromium (firefox/webkit could not be exercised in this sandbox due to missing local browser binaries — an environment limitation, not a code failure).

However, a fresh code review of the gap-closure diff (`12-REVIEW.md`, requested alongside this re-verification) found one Critical, currently-unfixed defect this verifier independently reconfirmed by reading the shipped code: `SourceChip.svelte`'s new advisory tooltip branch (12-10) gates on `last_status !== 'error'` only, never on `source.reachable` or the already-computed `tone` value, so a source that is currently unreachable but still carries a stale `last_status: 'ok'` and a leftover `last_notice` renders a reassuring "synced ... — advisory" tooltip instead of "unreachable since ...". The health dot itself is unaffected (it correctly reads `healthTone`, which checks `reachable` first) — only the tooltip TEXT, the one place a user reads *why* the dot is a given color, is wrong. This directly contradicts 12-10-PLAN.md's own must-have ("an unreachable source ... keeps the tone and copy it had before this plan") and its own threat-model mitigation for T-12-46, and it is precisely the failure class ("a healthy-looking surface hiding a real problem") this entire gap-closure round exists to eliminate — inverted, for a different precondition than the one UAT originally reported. `match-advisory.test.ts`'s own `reachable: false` precedence case (line 91-92) only asserts `healthTone`'s tone, never which tooltip branch is chosen for that input, so the guard file whose stated purpose is exactly this precedence does not catch this regression.

**This looks like a straightforward, low-risk fix** (the plan's own suggested fix — gate on `tone === 'success'` instead of `last_status !== 'error'` — reuses a value already computed one line above the switch), but it is not yet applied at the commit this verification ran against (`9083067`). Recommend a small follow-up gap-closure plan (e.g. `12-11`) applying the fix from `12-REVIEW.md`'s own `CR-01` recommendation and extending `match-advisory.test.ts` to assert the tooltip branch chosen for `reachable:false` + `last_status:'ok'` + non-empty `last_notice`, not just the tone.

---

_Verified: 2026-08-14T11:05:00Z_
_Verifier: Claude (gsd-verifier)_
