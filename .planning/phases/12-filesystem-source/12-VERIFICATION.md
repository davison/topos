---
phase: 12-filesystem-source
verified: 2026-08-14T12:15:00Z
status: human_needed
score: 11/11 must-have truths verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: "10/11"
  gaps_closed:
    - "CR-01: SourceChip.svelte's tooltip advisory gate now defers to a new format.ts predicate (isAdvisoryOnly, which re-asks healthTone with the notice removed) instead of testing source.last_status directly. A source that is reachable:false with a stale last_status:'ok' and a leftover last_notice now selects the 'unreachable since …' tooltip branch and carries none of the advisory text — independently reconfirmed by reading the code and by running the new unit matrix and a new e2e spec (both pass)."
    - "WR-01: kernel/httpapi/sources.go's launch-failure merge now carries a comment recording that LastNotice is deliberately not copied from the recorded run (matching docs/api.md's published contract), pinned by a new Go test (TestSourcesHandler_LaunchFailedEntryCarriesNoLastNotice) — independently re-run, passes."
  gaps_remaining: []
  regressions: []
  note: "This is the fourth verification pass on Phase 12 (initial 7/7 passed before UAT; gaps_found 10/11 after UAT reopened it citing G-12-1/G-12-3; gap-closure waves 12-08/12-09/12-10 closed the two UAT-reported missing items but 12-10 introduced a new defect, CR-01, caught by a fresh code review requested alongside that re-verification; this pass verifies gap-closure plan 12-11, which closed both CR-01 and the separately-noted non-blocking WR-01). This verifier independently re-read every changed line (did not trust 12-11-SUMMARY.md's narrative), independently re-ran the full Go test suite (go test ./... — all packages pass), the full frontend vitest suite (999/999 passed, matching the SUMMARY's claim), svelte-check (0 errors, 10 pre-existing unrelated warnings), the three targeted Go tests naming CR-01/WR-01/the pre-existing launch-failure merge test, the full match-advisory.test.ts file (36/36), and — critically — actually EXECUTED the new e2e spec (12-tooltip-precedence.spec.ts, both tests) plus the two previously-passing filesystem e2e specs (12-zero-match-diagnostic.spec.ts, 12-filesystem-root-label-match.spec.ts) rather than accepting the SUMMARY's '120/120 chromium' claim at face value. All pass. git diff --name-only across the plan's commit range lists exactly the six files declared in 12-11-PLAN.md's files_modified and nothing else. The fresh 12-REVIEW.md (delta review requested alongside this verification's predecessor round) independently confirmed both fixes genuine and reported status: clean (0 critical, 0 warning, 1 info — a pre-existing, out-of-scope whitespace-notice edge case, IN-02). No gap remains that is closable by more code: what remains open is a set of backstop truths (declared as verification: backstop in 12-08/12-09/12-10/12-11's own PLAN frontmatter) that require the user's own desktop and real network mounts to confirm — these route to human_needed per the honest-verifier convention rather than being silently marked passed."
deferred: []
behavior_unverified_items: []
human_verification:
  - test: "On the user's real desktop, re-save the actual webspace config so the filesystem source's `folders` match value is the source root's own base name (per 12-08's fix), then trigger a sync."
    expected: "The user's own filesystem documents (previously invisible under the old match-all-inexpressible failure) now appear in the correct webspace stream, with previews."
    why_human: "Requires the user's own machine, their own real folder structure, and their own config file — the e2e specs prove the mechanism against a synthetic corpus but cannot exercise the user's actual data."
  - test: "On the user's real desktop, before correcting the config, check `GET /api/sources` (or the Manage Sources UI) for the `files` source."
    expected: "A `last_notice` naming the real webspace and the real mistyped match value the user originally typed — the zero-match state is diagnosed rather than silent."
    why_human: "Requires the user's live kernel and their live (possibly still-uncorrected) config; the mechanism is proven end-to-end by 12-zero-match-diagnostic.spec.ts against a synthetic corpus."
  - test: "On the user's real desktop: (a) before the fix, confirm the `files` chip in the relevant webspace shows the warning tone with the diagnostic text naming the zero-matching value; (b) after correcting the value, confirm documents appear with previews and a working desktop-handler open action; (c) confirm the same flow works when the source folder is a network (NFS/SMB) mount, not just a local path."
    expected: "Amber/warning chip with correct diagnostic copy pre-fix; full working preview/open/sync flow post-fix; NFS/SMB behaves identically to a local path."
    why_human: "Previews, the real xdg-open handoff, and actual NFS/SMB mount behavior are explicitly scoped out of the hermetic browser harness by docs/testing.md's own design — this is the same backstop item carried since the original 12-05 tracer plan."
  - test: "On the user's real desktop, unmount (or otherwise make unreachable) a network-mounted filesystem source whose last completed sync carried a leftover advisory (e.g. a zero-match notice from before the mount went away), then check the chip."
    expected: "The chip shows the red/destructive dot AND its tooltip reads '{display_name} — unreachable since {relative}' — never the reassuring 'synced … — advisory' text. This is CR-01's real-world counterpart to the now-passing fabricated-response e2e test."
    why_human: "The hermetic harness can only fabricate a `GET /api/sources` response where reachable:false and last_status:'ok' disagree with each other on purpose (12-tooltip-precedence.spec.ts Test A already proves the SPA presents such a body correctly); no fixture plugin can be made to produce that disagreement live, because it requires a real mount actually going away between two real sync attempts."
---

# Phase 12: Filesystem Source Verification Report (Re-verification, after 12-11 gap-closure)

**Phase Goal:** The user can point topos at a folder — local or on a network mount — and see its documents in the right webspace.
**Verified:** 2026-08-14T12:15:00Z
**Status:** human_needed
**Re-verification:** Yes — after gap-closure plan 12-11 executed and merged, closing CR-01 (the sole open gap from the prior `gaps_found` verification) and WR-01 (a previously-noted non-blocking warning).

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | User adds a folder as a source from the UI, recursion on/off, documents appear in the matching webspace stream with previews (Roadmap SC1) | ✓ VERIFIED | Unchanged; regression-confirmed — `go test ./...` (full workspace) passes, including `plugins/filesystem`'s untouched suite |
| 2 | Files added/changed/removed reflected on next sync, including NFS/SMB mounts (Roadmap SC2) | ✓ VERIFIED | Unchanged; untouched by 12-11's `files_modified` list |
| 3 | Every filesystem item deep-links back to the desktop's own file handler, or declares honestly it can only raise (Roadmap SC3) | ✓ VERIFIED | Unchanged; `kernel/httpapi/fsopen.go` untouched by 12-11 |
| 4 | The plugin never writes to the source folder — enforced by committed guards (Roadmap SC4) | ✓ VERIFIED | Unchanged; `TestPluginIssuesNoWrite` file untouched by 12-11 |
| 5 | The filesystem binary loads/syncs identically from the external plugins directory, external badge shown (Roadmap SC5) | ✓ VERIFIED | Unchanged; `web/e2e/specs/12-external-rehearsal.spec.ts` untouched by 12-11 |
| — | (Cross-cutting) MUST NOT index/serve/preview/open any file outside the configured source root | ✓ VERIFIED | Unchanged; `resolvePath`/`FilesystemOpenHandler` untouched by 12-11 |
| 6 | G-12-1/G-12-3, missing item 1: "everything from this instance" is expressible via the root's own base name, at every depth | ✓ VERIFIED | Unchanged from prior pass; `TestFolderLabels_*` suite untouched and still passing |
| 7 | Docs state match values are exact literals, never globs | ✓ VERIFIED | Unchanged; `docs/plugins/filesystem.md`/`config.example.toml` untouched by 12-11 |
| 8 | G-12-1/G-12-3, missing item 2: a zero-match state across a healthy sync is named, not silent | ✓ VERIFIED | Unchanged; `kernel/correlate`/`kernel/index`/`kernel/syncer` untouched by 12-11 (explicit prohibition honored — see Anti-Patterns/Prohibitions below) |
| 9 | The notice is kernel-composed only — no plugin-provided string ever enters it (A-PLUG-04) | ✓ VERIFIED | Unchanged; `zeroMatchNotice` untouched by 12-11 |
| 10 | G-12-1/G-12-3, missing item 3: the advisory is documented as additive, kernel-owned, not-an-error | ✓ VERIFIED | Unchanged; `docs/api.md`'s existing bullets untouched by 12-11's diff (docs already carried the WR-01 contract before this plan) |
| 11 | G-12-1/G-12-3, missing item 3 (browser surface): the advisory reaches the chip AND never outranks a bigger, real problem (unreachable, pin-mismatch, never-synced, errored) | ✓ VERIFIED | **CR-01 closed.** `isAdvisoryOnly(source)` added to `web/src/lib/format.ts`, re-asking `healthTone` with the notice stripped rather than re-deriving a narrower condition; `SourceChip.svelte`'s tooltip advisory gate now reads `advisory !== '' && advisoryOnly` instead of `advisory !== '' && source.last_status !== 'error'`. Independently confirmed at the source: `healthTone`'s own body is byte-unchanged (only an addition after it); the four pre-existing tooltip switch-arm templates are byte-identical (confirmed by reading the file). Independently re-ran `match-advisory.test.ts`'s full file (36/36 pass, including the 7-case `isAdvisoryOnly` matrix and the 6-case `tooltipText` branch-selection matrix asserting full produced strings) and independently EXECUTED the new e2e spec `12-tooltip-precedence.spec.ts` (2/2 pass on chromium) — Test A proves the rendered `title` attribute says "unreachable since …" and contains none of the advisory text for the `reachable:false`+stale-`ok`+leftover-notice state; Test B proves the advisory branch still renders for a genuinely healthy+advisory source (the anti-dead-code guard) |

**Score:** 11/11 truths verified at the mechanism level. Status is `human_needed`, not `passed`, because four backstop truths (declared `verification: backstop` in 12-08/12-09/12-10/12-11's own PLAN frontmatter) require the user's own desktop and real network mounts and cannot be confirmed by this verifier — see Human Verification Required below.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `web/src/lib/format.ts` | `isAdvisoryOnly` predicate, `healthTone` body unchanged | ✓ VERIFIED | Read in full; `isAdvisoryOnly` present exactly once, placed immediately after `healthTone`; `healthTone`'s body (lines 132-146) is identical to the prior verification's own transcription |
| `web/src/lib/components/SourceChip.svelte` | `advisoryOnly` derived value, gate deferring to `isAdvisoryOnly`, four switch-arm templates untouched | ✓ VERIFIED | `isAdvisoryOnly` imported (line 30); `advisoryOnly` derived declared (line 136); gate condition is `advisory !== '' && advisoryOnly` (line 230) |
| `web/src/lib/components/match-advisory.test.ts` | `isAdvisoryOnly` matrix, coupling assertion, branch-selection matrix asserting full strings | ✓ VERIFIED | All present; independently re-run, 36/36 pass |
| `web/e2e/specs/12-tooltip-precedence.spec.ts` | Browser proof of the rendered `title` attribute for both the CR-01 case and the anti-dead-code case | ✓ VERIFIED | File exists (159 lines); independently EXECUTED (not just read) — 2/2 pass on chromium |
| `kernel/httpapi/sources.go` | Comment-only change recording the WR-01 contract, no behavioral change | ✓ VERIFIED | `git diff` of this file across the plan's commit range is comment-only (new lines 218-225); no struct field added, `LastError:` assignment unchanged |
| `kernel/httpapi/sources_test.go` | New Go test pinning the launch-failure `last_notice` contract | ✓ VERIFIED | `TestSourcesHandler_LaunchFailedEntryCarriesNoLastNotice` present; independently re-run, passes; asserts both the raw JSON body and confirms the merge genuinely saw a recorded run (non-vacuous) |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `web/src/lib/format.ts` (`healthTone`) | `web/src/lib/format.ts` (`isAdvisoryOnly`) | re-invokes `healthTone` on a copy with `last_notice` cleared | ✓ WIRED | Confirmed by reading the function body; the coupling test (`healthTone`/`isAdvisoryOnly` "anti-dead-code guard") independently re-run and passes |
| `web/src/lib/format.ts` (`isAdvisoryOnly`) | `web/src/lib/components/SourceChip.svelte` (`tooltipText`'s advisory branch) | the `advisoryOnly` `$derived` value | ✓ WIRED — CR-01 CLOSED | The gate now reads `advisoryOnly` rather than re-deriving a narrower condition; confirmed by both the unit branch-selection matrix and the executed e2e spec |
| `kernel/httpapi/sources.go` (launch-failure merge) | `docs/api.md` (`last_notice` bullet) | deliberate omission, now commented and pinned | ✓ WIRED — WR-01 CLOSED | Comment names `docs/api.md` and the new test; `TestSourcesHandler_LaunchFailedEntryCarriesNoLastNotice` independently re-run, passes |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| `isAdvisoryOnly` + `tooltipText` branch-selection matrix | `npx vitest run src/lib/components/match-advisory.test.ts` (run directly by this verifier) | 36/36 PASS | ✓ PASS |
| CR-01 fix + anti-dead-code guard, real browser | `npx playwright test --project=chromium e2e/specs/12-tooltip-precedence.spec.ts` (run directly by this verifier) | 2/2 PASS | ✓ PASS |
| Regression: previously-passing filesystem e2e specs | `npx playwright test --project=chromium e2e/specs/12-zero-match-diagnostic.spec.ts e2e/specs/12-filesystem-root-label-match.spec.ts` (run directly by this verifier) | 3/3 PASS | ✓ PASS |
| WR-01 + related Go tests | `go test ./kernel/httpapi/... -run 'TestSourcesHandler_LaunchFailedEntryCarriesNoLastNotice|TestSources_ZeroMatchNoticeTravelsFromCorrelateToTheSourcesAPI|TestSourcesHandler_MergesLaunchFailureIntoSourcesResponse' -count=1 -v` (run directly by this verifier) | 3/3 PASS | ✓ PASS |
| Full Go workspace suite (run once, per verifier constraint) | `CGO_ENABLED=0 go test ./...` (run directly by this verifier) | All packages `ok` (cmd/topos, internal/audit, kernel/config, kernel/correlate, kernel/httpapi, kernel/index, kernel/pluginhost, kernel/supervisor, kernel/syncer) | ✓ PASS |
| Full frontend vitest suite (run once, per verifier constraint) | `npm --prefix web run test` (run directly by this verifier) | 999/999 PASS, 54/54 files — matches 12-11-SUMMARY.md's claim, independently confirmed | ✓ PASS |
| svelte-check | `npm --prefix web run check` (run directly by this verifier) | 0 ERRORS, 10 pre-existing warnings (unrelated files: MatchFieldsForm, ConnectionForm, EditSourceModal, ManageSourcesModal, SearchBox — `state_referenced_locally`, none in the 12-11 diff) | ✓ PASS |
| Build | `go build ./...` (run directly by this verifier) | Clean | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| SRC-04 | 12-01 through 12-11 (all declare `requirements: [SRC-04]`) | User can add a local/network filesystem folder as a source; documents appear with previews and deep links, synced via stat-diff polling | ✓ SATISFIED AT THE MECHANISM LEVEL — human confirmation pending | Both gaps from the prior verification (CR-01, and the noted-but-non-blocking WR-01) are now closed and independently reconfirmed. No further mechanism-level gap remains. `REQUIREMENTS.md` line 78 still marks SRC-04 "Gaps Found" — per this phase's own standing prohibition ("the verifier owns those marks"), this verifier is the one to update that mark, and does NOT do so here because status is `human_needed`, not `passed`: the backstop truths in Human Verification Required must be confirmed on the user's own desktop before SRC-04 can be marked Complete |

No orphaned requirements — SRC-04 is the only requirement mapped to Phase 12.

### Anti-Patterns Found

None. No debt markers (`TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`) in any of the 6 files modified by 12-11. `git diff --name-only` across the plan's commit range (`f8fdf8d..8fbd576`) lists exactly the six files declared in `12-11-PLAN.md`'s `files_modified` and nothing else — no scope creep.

### Related, Non-Blocking Findings (carried forward from 12-REVIEW.md's delta review, independently spot-checked)

**IN-02 — whitespace-only `last_notice` would produce a malformed "last error: " tooltip on a healthy sync.** Pre-existing (predates 12-11; the same divergence existed under the old, now-replaced gate), explicitly out-of-scope per 12-11-PLAN.md, and its own code comment documents it. Whether the kernel can ever actually emit a whitespace-only notice is unconfirmed (would require reading `kernel/correlate`, out of this delta's scope). Cosmetic-only, Info-level in the fresh code review. Not a gap.

**IN-01 — `zeroMatchNotice`'s non-empty-fields guard counts field keys rather than non-empty values.** Deferred since the prior verification round, explicitly out of 12-11's scope (`kernel/correlate` is not in its `files_modified`). Not a gap.

**Stray untracked build binary.** `plugins/filesystem/filesystem` remains present as an untracked ELF binary (`git status --short`). Hygiene note only, unrelated to any phase 12 commits.

### Human Verification Required

No FAILED truth requires human judgment — CR-01 and WR-01 are both directly observable code fixes, independently confirmed by reading the current source and by executing (not merely reading about) the relevant test suites. Both are closed.

What remains are four **backstop** (`verification: backstop`) truths, declared in 12-08/12-09/12-10/12-11's own PLAN frontmatter must_haves, that can only be confirmed on the user's real desktop with real data and (for one item) a real network mount. Per the honest-verifier convention, these are not silently marked `passed` — they are recorded below and drive this report's `human_needed` status:

1. **12-08 backstop: the user's real filesystem source, re-saved with the folder's base name as the match value, produces documents in the webspace stream.**
   Expected: after re-saving their real webspace config with the corrected match value, the user's own filesystem documents appear with previews.
   Why human: this is the user's own machine and their own config file.

2. **12-09 backstop: the advisory appears against the user's real filesystem source after the next sync, naming the real webspace and the value they typed.**
   Expected: `GET /api/sources` (or the Manage Sources UI) on the user's own kernel shows a `last_notice` naming their real webspace and the mistyped value, if not yet corrected.
   Why human: requires the user's live kernel and live config.

3. **12-10 backstop: the chip shows the warning tone and names the zero-matching value pre-fix; after correcting it, documents appear with previews and a working desktop-handler open action — including on a network mount.**
   Expected: amber chip with diagnostic text pre-fix; working preview/open/sync flow post-fix; NFS/SMB parity with local paths.
   Why human: previews, the real `xdg-open` handoff, and NFS/SMB behavior are explicitly scoped out of the hermetic browser harness by `docs/testing.md`'s own design.

4. **12-11 backstop: on the user's own desktop, a source that goes unreachable while its last recorded sync carried an advisory shows the red dot AND an "unreachable since" tooltip (CR-01's real-world counterpart).**
   Expected: visually confirmed destructive-tone chip whose tooltip says "unreachable since …" and never the advisory text, after a real mount goes away.
   Why human: only a real mount going away can produce a live probe that disagrees with a stale recorded run in this exact way; the hermetic harness fabricates that disagreement instead (and 12-tooltip-precedence.spec.ts already proves the SPA presents it correctly once fabricated).

### Gaps Summary

No gaps remain. The prior verification's sole blocking gap (CR-01) and its separately-noted non-blocking warning (WR-01) are both genuinely closed by gap-closure plan 12-11, independently reconfirmed here by reading every changed line, by independently re-running the full Go and vitest suites (all pass, matching the SUMMARY's claims), and by independently EXECUTING — not merely trusting — both the new e2e spec and the two previously-passing filesystem e2e specs (all pass on chromium). A fresh code review of the same diff (12-REVIEW.md) reached status `clean` (0 critical, 0 warning) and is independently corroborated.

Status is `human_needed` rather than `passed` solely because four backstop truths, declared by the phase's own gap-closure plans, require confirmation on the user's real desktop (real folders, real config, and for one item, a real network mount going away). These are not new findings — three were already carried in the prior verification's Human Verification Required section as non-blocking informational items; the fourth is 12-11's own new backstop truth, added because its fix concerns exactly the state (a live mount disappearing) that only a real desktop can produce. None of the four represents an unresolved code defect.

---

_Verified: 2026-08-14T12:15:00Z_
_Verifier: Claude (gsd-verifier)_
