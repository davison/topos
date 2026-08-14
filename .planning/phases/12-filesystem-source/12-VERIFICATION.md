---
phase: 12-filesystem-source
verified: 2026-08-14T02:30:00Z
status: passed
score: 7/7 must-have truths verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: "6/7"
  gaps_closed:
    - "A file admitted to a filesystem source's index only because include_glob widened scope past the default extension allowlist is fetched as an honest metadata-only preview when opened — never a false 'not found' for a file that is present on disk and was legitimately synced (G-12-3 / fresh-review CR-01)"
  gaps_remaining: []
  regressions: []
  note: "Plan 12-07 closed the single recorded gap (fetchByKind now classifies through newScope(p.extras).includes, the same rule Match/walk use, instead of the bare classify() helper) plus three related non-blocking findings from the same review cycle (WR-02 resolved-path read/exec discipline, WR-01 missing provenance.source_system key, IN-01 stale package doc). This verification independently re-ran the exact live repro the prior pass used (include_glob=\"**/*.zip\" + archive.zip via a temporary probe test, removed after use) and confirmed Fetch now returns Available:false with the metadata-only reason instead of codes.NotFound. All 7 named regression tests from 12-07-PLAN.md were run directly and pass, plus the full plugins/filesystem suite, the kernel/httpapi open-route suite, and internal/audit."
gaps: []
deferred: []
---

# Phase 12: Filesystem Source Verification Report (Re-verification, after 12-07 gap closure)

**Phase Goal:** The user can point topos at a folder — local or on a network mount — and see its documents in the right webspace.
**Verified:** 2026-08-14T02:30:00Z
**Status:** passed
**Re-verification:** Yes — after gap-closure plan 12-07 executed (commits `5e59edc`, `5d8db83`, `7729f50`), closing the single gap the prior verification (2026-08-14, `gaps_found`, 6/7) recorded.

## Goal Achievement

### Observable Truths (mapped to the 5 roadmap success criteria, the cross-cutting hard prohibition, and the truth surfaced by the prior pass's fresh review)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | User adds a folder as a source from the UI, recursion on/off, documents appear in the matching webspace stream with previews | ✓ VERIFIED | Unchanged since prior pass: `web/src/lib/plugin-fields.ts` connection row, `ConnectionForm.svelte` checkbox branch, `web/e2e/specs/12-filesystem-add-source.spec.ts`, `web/e2e/specs/12-filesystem-recursion.spec.ts`. Preview pipeline confirmed by `plugins/filesystem` unit suite passing in full (`CGO_ENABLED=0 go test ./plugins/filesystem/... -count=1` → `ok`). The one previously-caveated subset (an `include_glob`-only item outside the default allowlist) is now truth #7 below, and is fixed. |
| 2 | Files added/changed/removed are reflected on next sync, including on NFS/SMB mounts with no OS notification dependency | ✓ VERIFIED | Unchanged: `plugins/filesystem/walk.go`'s full re-walk; untouched by 12-07 (files_modified list confirms `walk.go` was not touched by this plan). `TestWalk_*` suite passes as part of the full package run. |
| 3 | Every filesystem item deep-links back to the desktop's own file handler, or declares honestly it can only raise | ✓ VERIFIED | Unchanged from prior pass (CR-01/12-06 closed then, re-confirmed here): `kernel/httpapi/fsopen.go`'s `newXDGOpener` still builds the child with plain `exec.Command`, opener handed `context.WithoutCancel(ctx)`. 12-07 additionally switched the opener's argument from the lexical `full` to the `resolved` path (WR-02) — read the current source at `kernel/httpapi/fsopen.go:178` (`opener(context.WithoutCancel(ctx), resolved)`) and confirmed `TestFilesystemOpen_OpenerReceivesTheSymlinkResolvedPath` passes. |
| 4 | The plugin never writes to the source folder — enforced by committed guards | ✓ VERIFIED | Unchanged: `plugins/filesystem/readonly_test.go`'s `TestPluginIssuesNoWrite` (AST scan) independently re-run — passes. Untouched by 12-07's `files_modified` list. |
| 5 | The filesystem binary loads/syncs identically from the external plugins directory, showing the untrusted badge, before Google Drive work begins | ✓ VERIFIED | Unchanged: `web/e2e/specs/12-external-rehearsal.spec.ts` exists, untouched by 12-07. |
| — | (Cross-cutting) MUST NOT index/serve/preview/open any file outside the configured source root | ✓ VERIFIED | Unchanged from prior pass (CR-02/12-06 closed then): `resolvePath`/`FilesystemOpenHandler` still compare `filepath.EvalSymlinks`-resolved paths for containment. 12-07 additionally now reads/execs through the SAME resolved path the containment check approved (previously validated the resolved path but read/exec'd the lexical one — WR-02) — `resolvePath`'s signature widened to `(full, resolved string, err error)` (read at `plugins/filesystem/item.go:108-125`), `fetchByKind`/`openForFetch` read `resolved` (`plugins/filesystem/fetch.go:107,130-134`), and `FilesystemOpenHandler` execs `resolved` (`kernel/httpapi/fsopen.go:178`). `TestFetch_SymlinkSwappedAfterIndexingIsRefusedBeforeAnyBytesAreServed`, `TestResolvePath_SymlinkSwapOutsideRootIsRefused`, and `TestFilesystemOpen_SymlinkSwappedAfterIndexingAnswersInvalidPathAndNeverOpens` all still pass — the containment guard was not weakened while switching the I/O target. |
| 7 | A file admitted to the index only via `include_glob` widening past the default extension allowlist is fetched as an honest metadata-only preview, never a false "not found" | ✓ VERIFIED — **gap closed** | `fetchByKind` (`plugins/filesystem/fetch.go:106-138`) now builds `sc := newScope(p.extras)` and classifies via `sc.includes(sourceID)` — the identical construction and rule `Match` uses (`plugin.go:127`) — instead of the bare `classify()` helper. **Independently re-reproduced live in this pass**, not merely re-run from the plan's own tests: a temporary probe test (written for this verification, removed after use, working tree left clean) built `NewSourcePlugin(dir, map[string]string{"include_glob": "**/*.zip"}, false)`, wrote `archive.zip`, confirmed `Match` includes it, then called `Fetch` and observed `available=false reason="preview not supported for this file type; open in source"` — the honest answer, not the previous `codes.NotFound`. All 7 of 12-07-PLAN.md's named new/modified tests were run directly by this verifier and pass: `TestFetch_IncludeGlobAdmittedUnknownExtensionIsMetadataOnlyNotNotFound`, `TestFetch_ExcludedByGlobIsStillNotFound`, `TestFetch_UnknownExtensionWithNoIncludeGlobIsStillNotFound`, `TestFetch_MalformedGlobPatternSurfacesTheOffendingPattern`, `TestFetch_SymlinkedRootStillServesAnInRootFile`, `TestResolvePath_ReturnsTheLexicalIdentityPathAndTheResolvedRealPath`, `TestMatch_ItemProvenanceCarriesTheFivePluginOwnedKeys`. |

**Score:** 7/7 truths verified. The gap the prior pass recorded (truth #7) is closed, independently re-reproduced live, and the two previously-closed cross-cutting fixes (CR-01, CR-02) were re-confirmed unregressed by the same commits that changed which path they read/exec against (WR-02).

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `plugins/filesystem/fetch.go` | `fetchByKind` classifying through `scope.includes`, not the bare `classify()` helper; a shared `openForFetch` for all three renditions | ✓ VERIFIED | Read in full. `newScope(p.extras)` appears once inside `fetchByKind`; `.includes(` appears once; `openForFetch` defined once and used by `fetchBytesRendition`, `fetchMarkdownRendition`, `fetchPlainTextRendition` |
| `plugins/filesystem/item.go` | `resolvePath` returning both lexical and resolved paths | ✓ VERIFIED | Signature is `func resolvePath(root, sourceID string) (full, resolved string, err error)`; both empty on error |
| `plugins/filesystem/plugin.go` | `toItem` setting `provenance.source_system` | ✓ VERIFIED | `"source_system": p.root` present in the `Provenance` map literal |
| `kernel/httpapi/fsopen.go` | Opener handed the resolved path | ✓ VERIFIED | `opener(context.WithoutCancel(ctx), resolved)` — `resolved`, not `full` |
| `plugins/filesystem/main.go` | Corrected package doc naming 12-03 as the plan that shipped recursion | ✓ VERIFIED | `sed -n '1,14p'` shows "recursion is not future work, it has shipped since 12-03" |
| `web/e2e/specs/12-include-glob-metadata-preview.spec.ts` | End-to-end gate over a real kernel + plugin subprocess | ✓ VERIFIED | File exists, read in full: asserts the stream lists the item, `GET /api/items/{id}` returns `available:false`, `GET /api/items/{id}/content` returns `content_unavailable` (not `item_not_found`), and `provenance.source_system` equals the corpus dir |
| `docs/plugins/filesystem.md` | "re-runs at Fetch time" phrase | ✓ VERIFIED | Present at line 64 |
| `docs/api.md`, `docs/plugin-contract.md` | "narrows but does not eliminate" phrase | ✓ VERIFIED | Present in both files |
| `plugins/filesystem/readonly_test.go` | Committed AST guard | ✓ VERIFIED | `TestPluginIssuesNoWrite` passes, untouched by this plan |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `plugins/filesystem/fetch.go` (`fetchByKind`) | `plugins/filesystem/scope.go` (`scope.includes`) | `newScope(p.extras)` built once per Fetch call | ✓ WIRED | **This closes the gap** — confirmed by reading `fetchByKind` and by the live probe reproduction |
| `plugins/filesystem/item.go` (`resolvePath`) | `plugins/filesystem/fetch.go` (per-kind renditions) | resolved real path passed to `openForFetch` | ✓ WIRED | `fetchByKind` discards the lexical `full` (blank identifier) and passes `resolved` to the dispatch switch |
| `kernel/httpapi/fsopen.go` (`FilesystemOpenHandler`) | injected `Opener` | resolved path the containment check approved | ✓ WIRED | Confirmed at the exec call site |
| `plugins/filesystem/plugin.go` (`toItem`) | `kernel/httpapi/stream.go` (`toStreamItemFor`) | provenance republished verbatim | ✓ WIRED | `source_system` now present in the map; kernel copies `Provenance` verbatim (unchanged code path); e2e spec confirms it reaches a client |
| `plugins/filesystem/item.go` | `kernel/httpapi/stream.go` | `file://` deep_link rewritten to loopback open route | ✓ WIRED | Unchanged, `TestStreamLink*` passing |
| `kernel/httpapi/routes.go` | `kernel/httpapi/fsopen.go` | `POST /api/items/{id}/open` registration | ✓ WIRED | Unchanged, untouched by 12-07 |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Fetch closing the recorded gap — 7 named regression tests | `CGO_ENABLED=0 go test ./plugins/filesystem/... -run 'TestFetch_IncludeGlobAdmittedUnknownExtensionIsMetadataOnlyNotNotFound\|TestFetch_ExcludedByGlobIsStillNotFound\|TestFetch_UnknownExtensionWithNoIncludeGlobIsStillNotFound\|TestFetch_MalformedGlobPatternSurfacesTheOffendingPattern\|TestFetch_SymlinkedRootStillServesAnInRootFile\|TestResolvePath_ReturnsTheLexicalIdentityPathAndTheResolvedRealPath\|TestMatch_ItemProvenanceCarriesTheFivePluginOwnedKeys' -count=1 -v` | all 7 RUN, all 7 PASS | ✓ PASS |
| Full plugin package suite (regression safety net) | `CGO_ENABLED=0 go test ./plugins/filesystem/... -count=1` | `ok` | ✓ PASS |
| Kernel open-route + stream-link suite | `go test ./kernel/httpapi/... -run 'TestFilesystemOpen\|TestNewXDGOpener\|TestStreamLink' -count=1 -v` | all 14 cases PASS | ✓ PASS |
| `internal/audit` contract suite | `go test ./internal/audit/... -count=1` | `ok` | ✓ PASS |
| Read-only guard | `go test ./plugins/filesystem/... -run TestPluginIssuesNoWrite -count=1 -v` | PASS | ✓ PASS |
| **Live re-reproduction of the exact prior-pass repro** | Temporary probe test (`plugins/filesystem/zzverifyprobe_test.go`, written for this verification, removed after, `git status --short plugins/filesystem` confirms clean) built `NewSourcePlugin(dir, {"include_glob":"**/*.zip"}, false)`, wrote `archive.zip`, ran `Match` then `Fetch` | `Match` includes `archive.zip`; `Fetch` returns `available=false reason="preview not supported for this file type; open in source"` (no error) | ✓ PASS — confirms the gap is genuinely closed, not merely claimed |
| Doc phrases required by the plan's acceptance criteria | `grep -n 're-runs at Fetch time' docs/plugins/filesystem.md`; `grep -n 'narrows but does not eliminate' docs/api.md docs/plugin-contract.md` | all found | ✓ PASS |
| e2e spec assertions present | Read `web/e2e/specs/12-include-glob-metadata-preview.spec.ts` in full | asserts `content_unavailable` ≠ `item_not_found`, `provenance.source_system` | ✓ PASS |
| `12-07`'s own non-goal: plans 12-01–12-06/REQUIREMENTS.md/ROADMAP.md untouched by this plan's commits | `git diff --stat 5e59edc^..7729f50 -- .planning/REQUIREMENTS.md .planning/ROADMAP.md` | empty diff | ✓ PASS |

Orchestrator-run gates cited (not independently re-run by this verifier, but corroborating): `make build`, `make test` (all Go modules incl. cgo Signal), and the full `make e2e` Playwright suite (115 specs) — all passed in this run, per the task context. This verifier independently re-ran the specific named unit tests and one live probe reproduction rather than relying solely on that claim.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| SRC-04 | 12-01 through 12-07 (all declare `requirements: [SRC-04]`) | User can add a local/network filesystem folder as a source; documents appear with previews and deep links, synced via stat-diff polling | ✓ SATISFIED | All clauses now hold: folder-as-source with recursion toggle (truth 1), stat-diff sync including network mounts (truth 2), deep-link-or-honest-decline (truth 3), read-only enforcement (truth 4), external-plugin-path rehearsal (truth 5), containment (cross-cutting), and — the clause that failed the prior pass — "documents appear... with previews" is now true for the `include_glob`-widened case too (truth 7). `REQUIREMENTS.md` line 78 still records SRC-04 as "Gaps Found" — per this plan's own non-goals, the executor deliberately left that mark for the verifier to update; this verification's result (`passed`) is the input needed to update it, but this verifier does not edit REQUIREMENTS.md itself (that update is the orchestrator's follow-on action). |

No orphaned requirements found — SRC-04 is the only requirement mapped to Phase 12, and no plan under this phase claims a requirement ID absent from REQUIREMENTS.md's Phase 12 mapping.

### Anti-Patterns Found

None. No debt markers (`TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`) in any file 12-07 modified (`fetch.go`, `fetch_test.go`, `item.go`, `item_test.go`, `plugin.go`, `main.go`, `fsopen.go`, `fsopen_test.go`). `go build ./...` was confirmed clean by the orchestrator's gate and this verifier's own `go vet`/test runs raised no errors. The temporary probe test file this verifier created for live reproduction was removed before finishing; `git status --short plugins/filesystem` shows no stray tracked-file changes (only a pre-existing untracked `plugins/filesystem/filesystem` build binary, unrelated to source — a leftover compiled artifact from a `go build`/`make build` run, not part of this plan's diff).

### Related, Non-Blocking Findings

**T-12-29 (accepted, documented honestly).** The residual final-path-component TOCTOU window between `filepath.EvalSymlinks` resolving and the subsequent `os.Open`/`exec` call is not eliminated by this plan — closing it fully would need `openat`/`O_NOFOLLOW` descriptor-based traversal, explicitly out of scope at ASVS L1 for this single-user loopback tool (same framing as the already-accepted T-12-23). `docs/api.md` and `docs/plugin-contract.md` both state this using the exact phrase "narrows but does not eliminate," confirmed present in both files. This is an accepted, honestly-documented residual, not a gap against this phase's goal or roadmap success criteria.

**Stray untracked build binary.** `plugins/filesystem/filesystem` is an untracked, unstripped ELF binary sitting in the plugin source directory (likely left over from a `go build ./...` invocation during the orchestrator's gate run). It is not part of any commit in this plan and does not affect the verification outcome, but is worth a `.gitignore` entry or cleanup pass — noted for hygiene, not scored as a gap.

### Human Verification Required

None. Every finding in this pass — the closed gap, the re-confirmed prior fixes, and the two non-blocking notes above — is demonstrable directly from source, by running the specific named Go tests, and by one live reproduction this verifier constructed independently. No item requires exercising a live desktop environment or browser beyond what the orchestrator's already-passing `make e2e` run covers.

### Gaps Summary

None. Plan 12-07 genuinely closed the single gap the prior verification recorded: `fetchByKind` now classifies through `newScope(p.extras).includes(sourceID)` — the same rule and the same constructed object `Match`/`walk` already use — instead of a second, weaker rule that had no knowledge of `include_glob`/`exclude_glob`. This verifier independently reproduced the exact scenario the prior pass used to prove the bug (`include_glob="**/*.zip"` + `archive.zip`) via a throwaway probe test and confirmed the honest `Available:false` answer now returns instead of `codes.NotFound`. The two cross-cutting containment fixes from 12-06 (CR-01, CR-02) were re-confirmed unregressed even though 12-07 changed which path (`resolved` vs `full`) the same guards read/exec against (WR-02). The three related non-blocking findings from the same review cycle (WR-02 resolved-path discipline, WR-01 missing `source_system` provenance key, IN-01 stale doc comment) are also closed, each with its own new regression test that did not exist before this plan, and this plan's own non-goals (leaving REQUIREMENTS.md, ROADMAP.md, and plans 12-01–12-06 untouched) were independently confirmed via `git diff` across the plan's own commit range.

All five roadmap success criteria, the cross-cutting read-only-source-root prohibition, and the additional truth the fresh code review surfaced are now verified. Phase 12's goal — "The user can point topos at a folder — local or on a network mount — and see its documents in the right webspace" — is achieved in the codebase, not merely claimed in a SUMMARY.

---

_Verified: 2026-08-14T02:30:00Z_
_Verifier: Claude (gsd-verifier)_
