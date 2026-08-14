---
phase: 12-filesystem-source
verified: 2026-08-14T01:15:00Z
status: gaps_found
score: 6/7 must-have truths verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: "4/6"
  gaps_closed:
    - "Every filesystem item deep-links back to the document in the desktop's own file handler, or declares honestly that it can only raise (CR-01: xdg-open child no longer killed by request-context cancellation)"
    - "MUST NOT index, serve, preview, or open any file outside the configured source root (CR-02: Fetch and Open now re-validate containment against filepath.EvalSymlinks-resolved paths, not lexical-only)"
  gaps_remaining: []
  regressions: []
  note: "Both previously-recorded gaps (G-12-1/CR-01, G-12-2/CR-02) are independently confirmed closed by re-reading the code and re-running the cited tests. However, a FRESH code review pass (12-REVIEW.md, re-run after 12-06's gap closure, status: issues_found) found one new, unrelated Critical defect that this verification independently reproduced live: Fetch's classification path ignores the include_glob scope that legitimately admitted an item at Match time, so a file synced into the stream only because of include_glob (with an extension outside the default allowlist) 404s when opened instead of returning the documented honest metadata-only preview. This is not a regression introduced by 12-06 (12-06 never touched scope.go, classify.go or fetchByKind's classification call) — it is a pre-existing defect in the original 12-02-PLAN.md Fetch dispatch that the fresh review surfaced and that no prior verification pass caught."
gaps:
  - truth: "A file admitted to a filesystem source's index only because include_glob widened scope past the default extension allowlist is fetched as an honest metadata-only preview when opened — never a false 'not found' for a file that is present on disk and was legitimately synced (12-02-PLAN.md D-03; docs/plugins/filesystem.md's documented resolution-order behavior)"
    status: failed
    reason: "plugins/filesystem/fetch.go's fetchByKind calls the bare package-level classify(sourceID) directly instead of building a *scope from p.extras and classifying through scope.includes — the same path Match/walk already use. classify only ever consults the fixed extensionTable and has no knowledge of include_glob/exclude_glob, so it cannot reproduce scope.includes' 'unknown extension admitted by glob -> metadata-only' branch (scope_test.go's TestScope_UnknownExtensionIncludedByGlobIsMetadataOnly proves this branch exists and is intentional). For any item indexed only because include_glob widened past the default allowlist, classify returns ok=false and fetchByKind answers codes.NotFound -- surfaced by the kernel as 404 item_not_found on both GET /api/items/{id} and GET /api/items/{id}/content -- for a file the UI legitimately lists in the stream. This is CR-01 in the freshly re-run 12-REVIEW.md (Critical, unresolved as of the latest commit b2d8180, 'docs(12): add code review report' -- no fix commit follows it). Independently reproduced live in this verification pass: a temp Go test built a SourcePlugin with include_glob=\"**/*.zip\", confirmed archive.zip appears in Match's results, then called Fetch and observed 'rpc error: code = NotFound desc = filesystem: item \"archive.zip\" not found' -- the false-404 the review describes, not a hypothetical."
    artifacts:
      - path: "plugins/filesystem/fetch.go"
        issue: "fetchByKind (lines 81-93) classifies via the bare classify(sourceID) call, which has no knowledge of p.extras' include_glob/exclude_glob and cannot reproduce scope.includes' unknown-extension-admitted-by-glob branch"
    missing:
      - "Build a *scope from p.extras inside fetchByKind (via newScope(p.extras), mirroring how Match/walk already construct one) and classify through scope.includes instead of the bare classify() call, per 12-REVIEW.md CR-01's suggested fix -- included=false should still map to NotFound for a source_id that is on disk but genuinely outside the instance's current scope (e.g. scope narrowed since the item was indexed), matching today's behavior for that case"
      - "A regression test that Fetches an item whose only qualification is include_glob against an extension outside the default allowlist, asserting Available:false with the metadata-only unavailable_reason instead of a NotFound gRPC error -- fetch_test.go's existing fixtures only cover extensions already in extensionTable plus a genuinely-missing file, and no test exercises Fetch against a NewSourcePlugin instance with a non-default include_glob for an unrecognized extension"
deferred: []
---

# Phase 12: Filesystem Source Verification Report (Re-verification)

**Phase Goal:** The user can point topos at a folder — local or on a network mount — and see its documents in the right webspace.
**Verified:** 2026-08-14T01:15:00Z
**Status:** gaps_found
**Re-verification:** Yes — after gap-closure plan 12-06 executed (closing G-12-1/CR-01 and G-12-2/CR-02, plus the carried-forward WR-01 symlinked-root warning); this pass also incorporates a freshly re-run code review (12-REVIEW.md) that surfaced one new, previously-uncaught Critical defect.

## Goal Achievement

### Observable Truths (mapped to the 5 roadmap success criteria, the cross-cutting hard prohibition, and one new truth surfaced by the fresh review)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | User adds a folder as a source from the UI, recursion on/off, documents appear in the matching webspace stream with previews | ✓ VERIFIED | Unchanged since prior pass: `web/src/lib/plugin-fields.ts` connection row, `ConnectionForm.svelte` checkbox branch, `web/e2e/specs/12-filesystem-add-source.spec.ts`, `web/e2e/specs/12-filesystem-recursion.spec.ts`. Preview pipeline for default-allowlist extensions (pdf/png/jpeg/gif/webp/md/txt/docx/svg) confirmed working by passing `plugins/filesystem` unit tests. (Caveat for a non-default subset covered by truth #7 below.) |
| 2 | Files added/changed/removed are reflected on next sync, including on NFS/SMB mounts with no OS notification dependency | ✓ VERIFIED | `plugins/filesystem/walk.go` full re-walk unchanged in design; 12-06 additionally fixed `walk`'s handling of a symlinked configured root (WR-01) so it is no longer silently under-traversed. `go test ./plugins/filesystem/... -count=1` passes, including all pre-existing `TestWalk_*` cases plus the new `TestWalk_InTreeSymlinkUnderASymlinkedRootIsStillIncluded`. |
| 3 | Every filesystem item deep-links back to the desktop's own file handler, or declares honestly it can only raise | ✓ VERIFIED (was FAILED in prior pass — CR-01 now closed) | `kernel/httpapi/fsopen.go`'s `newXDGOpener` now takes a blank-identifier context parameter and builds the child with plain `exec.Command("xdg-open", path)` — structurally impossible to bind to a caller's context (confirmed by reading the current source). `FilesystemOpenHandler` hands the opener `context.WithoutCancel(ctx)`. `grep -c 'WithoutCancel' kernel/httpapi/fsopen.go` = 1. `go test ./kernel/httpapi/ -run 'TestFilesystemOpen|TestNewXDGOpener' -count=1` passes, including `TestFilesystemOpen_OpenerContextIsDetachedFromTheRequestContext` (behavioural) and `TestNewXDGOpener_ChildIsNotBoundToACallerContext` (AST-structural). |
| 4 | The plugin never writes to the source folder — enforced by committed guards | ✓ VERIFIED | Unchanged: `plugins/filesystem/readonly_test.go`'s committed AST scan still passes; untouched by 12-06 or the fresh review. |
| 5 | The filesystem binary loads/syncs identically from the external plugins directory, showing the untrusted badge, before Google Drive work begins | ✓ VERIFIED | Unchanged: `web/e2e/specs/12-external-rehearsal.spec.ts` exists and is untouched by 12-06 or the fresh review's findings. |
| — | (Cross-cutting) MUST NOT index/serve/preview/open any file outside the configured source root | ✓ VERIFIED (was FAILED in prior pass — CR-02 now closed) | `plugins/filesystem/item.go`'s `resolvePath` and `kernel/httpapi/fsopen.go`'s inline check both now call `filepath.EvalSymlinks` on the joined path and the configured root (via a hand-duplicated `resolveRoot` helper in each module) and compare the RESOLVED pair, failing closed on resolution error and mapping a vanished file to `NotFound`/`item_not_found` distinctly from a genuine containment escape (`InvalidArgument`/`invalid_path`). `grep -c EvalSymlinks` reports 3 in `item.go`, 4 in `fsopen.go`, 1 in `walk.go`. `TestFetch_SymlinkSwappedAfterIndexingIsRefusedBeforeAnyBytesAreServed` and `TestFilesystemOpen_SymlinkSwappedAfterIndexingAnswersInvalidPathAndNeverOpens` both pass. The fresh code review independently re-verified both fixes hold (12-REVIEW.md's "Summary" section). |
| 7 (new) | A file admitted to the index only via `include_glob` widening past the default extension allowlist is fetched as an honest metadata-only preview, never a false "not found" | ✗ FAILED | **New Critical finding, fresh 12-REVIEW.md.** `fetchByKind` (`plugins/filesystem/fetch.go:81-93`) classifies via the bare `classify(sourceID)`, which has no knowledge of `p.extras`' `include_glob`/`exclude_glob` and cannot reproduce `scope.includes`' "unknown extension admitted by glob → metadata-only" branch that `Match`/`walk` already implement (`scope_test.go`'s `TestScope_UnknownExtensionIncludedByGlobIsMetadataOnly` proves the branch is intentional). Independently reproduced live in this verification pass (see Behavioral Spot-Checks). See gap below. |

**Score:** 6/7 truths verified — the two previously-failed truths (CR-01, CR-02) are now closed; one new truth, surfaced by the freshly re-run code review and independently reproduced here, fails.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `plugins/filesystem/plugin.go` | SourcePlugin Describe/Match/Fetch/Health | ✓ VERIFIED | Present, substantive, `topos.v2` contract version confirmed |
| `plugins/filesystem/item.go` | relative-path source_id, folder labels, file:// deep link, symlink-resolving containment | ✓ VERIFIED (was ⚠️ PRESENT BUT UNSAFE) | `resolveRoot` + `resolvePath` now resolve symlinks before the containment comparison (CR-02 closed) |
| `plugins/filesystem/fetch.go` | per-preview-kind dispatch honoring the scope that admitted the item | ✗ STUB-EQUIVALENT (classification path bypasses scope) | `fetchByKind` classifies via the bare `classify()` helper, not `scope.includes` — see gap |
| `kernel/httpapi/fsopen.go` | loopback open route, symlink-resolving containment, detached opener lifetime | ✓ VERIFIED (was ⚠️ PRESENT BUT UNSAFE / BROKEN) | Containment resolves symlinks (CR-02 closed); opener context detached via `context.WithoutCancel`, `newXDGOpener` structurally cannot bind to a caller context (CR-01 closed) |
| `plugins/filesystem/walk.go` | recursion-aware, symlink-safe, permission-tolerant walk, symlinked-root-safe | ✓ VERIFIED (WR-01 closed) | Now resolves the configured root once and walks from the resolved root; in-tree symlink comparison uses the resolved root |
| `plugins/filesystem/readonly_test.go` | committed AST guard | ✓ VERIFIED | Present, substantive, passing |
| `web/src/lib/components/ui/checkbox/checkbox.svelte` | shadcn-svelte Checkbox primitive | ✓ VERIFIED | Unchanged, present, wired into ConnectionForm |
| `docs/plugins/filesystem.md`, `docs/api.md`, `docs/plugin-contract.md` | operator + contract docs | ✓ VERIFIED | All three brought back into agreement with the shipped symlink-containment code by 12-06; `docs/plugin-contract.md:824-858`'s republished guarantee is now true of the shipped code for the CR-01/CR-02 class of defect (it does not cover the new Fetch/scope defect, which is a different code path) |

### Key Link Verification

| From | To | Via | Status | Details |
|------|----|----|--------|---------|
| `plugins/filesystem/item.go` | `kernel/httpapi/stream.go` | `file://` deep_link rewritten to loopback open route | ✓ WIRED | Unchanged, `TestStreamLink*` passing |
| `kernel/httpapi/routes.go` | `kernel/httpapi/fsopen.go` | `POST /api/items/{id}/open` registration | ✓ WIRED | Unchanged, single registration confirmed |
| `web/src/lib/components/OpenInSource.svelte` | `kernel/httpapi/fsopen.go` | fetch POST against same-origin `/api/` link | ✓ WIRED | Unchanged |
| `web/src/lib/plugin-fields.ts` | `web/src/lib/components/ConnectionForm.svelte` | field descriptor `kind` selects render branch | ✓ WIRED | Unchanged |
| `plugins/filesystem/plugin.go` (Match) | `plugins/filesystem/scope.go` (`scope.includes`) | `walk.go` builds one `*scope` per Match call and classifies through it | ✓ WIRED | Confirmed by reading `walk.go` |
| `plugins/filesystem/fetch.go` (`fetchByKind`) | `plugins/filesystem/scope.go` (`scope.includes`) | **expected but absent** | ✗ NOT WIRED | `fetchByKind` calls the bare `classify()` instead — this IS the gap |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Plugin + kernel unit suites (post gap-closure) | `CGO_ENABLED=0 go test ./plugins/filesystem/... -count=1` and `go test ./kernel/httpapi/... -run 'TestFilesystemOpen\|TestNewXDGOpener\|TestStreamLink' -count=1` | both `ok` | ✓ PASS |
| Symlink-resolving guards present | `grep -c EvalSymlinks plugins/filesystem/item.go kernel/httpapi/fsopen.go plugins/filesystem/walk.go` | 3 / 4 / 1 | ✓ PASS |
| Opener context detachment present | `grep -c WithoutCancel kernel/httpapi/fsopen.go` | 1 | ✓ PASS |
| `internal/audit` contract suite | `go test ./internal/audit/ -count=1` | ok | ✓ PASS |
| Full repo build | `go build ./...` | clean, exit 0 | ✓ PASS |
| **Live reproduction of the new Fetch/scope defect** | A temporary test (`plugins/filesystem/zzprobe_test.go`, written for this verification, removed afterward, working tree left clean) built `NewSourcePlugin(root, map[string]string{"include_glob": "**/*.zip"}, false)`, wrote `archive.zip`, called `Match` (confirmed the item appears in results), then called `Fetch` | `Match` includes `archive.zip`; `Fetch` returns `rpc error: code = NotFound desc = filesystem: item "archive.zip" not found` instead of `Available:false` with the metadata-only reason | ✗ FAIL — confirms 12-REVIEW.md's CR-01 (fresh review) is real, not theoretical |

No full e2e browser run was executed (would require booting a real kernel/browser harness); the e2e spec files were confirmed to exist and are unaffected by either the gap-closure work or the new finding, consistent with the prior verification pass's scope.

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|--------------|--------|----------|
| SRC-04 | 12-01 through 12-06 (all declare `requirements: [SRC-04]`) | User can add a local/network filesystem folder as a source; documents appear with previews and deep links, synced via stat-diff polling | ⚠️ PARTIALLY SATISFIED | The deep-link and containment clauses that failed the previous verification pass are now genuinely fixed (CR-01, CR-02, WR-01 all confirmed closed by reading the code and re-running the cited tests). However, the "documents appear... with previews" clause is still not fully satisfied: a legitimately-synced file admitted only via `include_glob` past the default extension allowlist 404s instead of showing an honest metadata-only preview. `REQUIREMENTS.md` correctly still records SRC-04 as unchecked / "Gaps Found" (line 20, line 78) — this mark should remain until the new gap closes; it was NOT prematurely reverted to Complete in this pass. |

No orphaned requirements found — SRC-04 is the only requirement mapped to Phase 12.

### Anti-Patterns Found

None. No debt markers (`TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`) found in any non-test file touched by this phase (including the files 12-06 modified). `go build ./...` is clean. The temporary probe test file created for this verification's live reproduction was removed before finishing; `git status --short` confirms no stray files remain from this verification pass.

### Related, Non-Blocking Findings (carried forward from the freshly re-run 12-REVIEW.md, not scored as gaps)

**WR-01 (new, Warning — distinct from the WR-01 12-06 already closed):** `plugins/filesystem/plugin.go`'s `toItem` omits the `source_system` provenance key that `docs/plugin-contract.md`'s "Provenance" section documents as plugin-populated, and that every sibling plugin (paperless, silverbullet, signal) sets. `kernel/httpapi/stream.go` copies `Provenance` verbatim and does not synthesize this key on a plugin's behalf, so every filesystem-sourced item's API response is silently missing `provenance.source_system`. Confirmed by reading `plugin.go`'s `toItem` function (no `source_system` key present) and confirming `internal/audit` does not currently check for it (`go test ./internal/audit/ -count=1` passes regardless). No phase must-have or roadmap success criterion explicitly names this field, so it does not change the overall status, but it is a genuine, silent contract regression that should be closed.

**WR-02 (new, Warning):** Both `resolvePath` and `FilesystemOpenHandler` correctly validate containment against the `filepath.EvalSymlinks`-resolved path, but then perform the actual read/exec against the original lexical path rather than the already-resolved one — leaving a narrow, single-request TOCTOU window between validation and the syscall (distinct from and narrower than the cross-request window CR-02 closed). The review itself frames this as a lower-priority hardening item, comparable to the already-accepted T-12-23 residual risk, not a fresh exploitable path this phase introduced.

**IN-01 (new, Info):** `plugins/filesystem/main.go`'s package doc comment is stale, still describing a pre-recursion tracer-era state that recursion (12-03-PLAN.md) has since superseded. Cosmetic only.

### Human Verification Required

None. All findings in this pass (the two closed gaps, the new gap, and the three non-blocking findings) are demonstrable directly from source and by running Go tests, including one live reproduction — none require exercising a live desktop environment or browser.

### Gaps Summary

Plan 12-06 genuinely closed both gaps this verifier previously recorded (CR-01: the kernel's `xdg-open` open action is no longer tied to the HTTP request's lifetime; CR-02: the containment re-validation on the Fetch and Open routes now resolves symlinks before comparing, rather than comparing lexically), plus the carried-forward WR-01 warning about symlinked configured roots. All of this is independently confirmed by reading the current source and re-running the specific tests that prove each property, not merely by trusting 12-06-SUMMARY.md's claims.

However, this phase's own code-review gate re-ran after that gap closure (per the standard workflow) and found one new, unrelated Critical defect: `fetchByKind`'s classification step ignores the `include_glob`/`exclude_glob` scope that `Match` and `walk` already honor, so a file that legitimately appears in the stream *only* because `include_glob` widened past the default extension allowlist throws a false "not found" when opened, instead of the documented honest "preview not supported for this file type" response. This verification independently reproduced the defect live (not merely re-stating the review's prose) by building a `SourcePlugin` with `include_glob: "**/*.zip"`, confirming `Match` includes the file, then calling `Fetch` and observing the `NotFound` error the review describes.

This is not a regression caused by 12-06 — 12-06's diff never touched `scope.go`, `classify.go`, or `fetchByKind`'s classification call — it is a pre-existing defect in the original Fetch dispatch (12-02-PLAN.md) that neither the initial verification pass nor 12-06's gap-closure work happened to exercise. Because it breaks the phase's own documented behavior for a feature this phase built (`include_glob` scope widening, D-03), and produces user-visible dishonest behavior (a false 404 for a file the UI itself lists), it must be closed before Phase 12 can be marked complete.

The two additional Warning-level findings (missing `source_system` provenance key; a narrower single-request TOCTOU window between symlink resolution and the actual read/exec) and one Info-level finding (a stale doc comment) do not block this verification's status but should be addressed in the same closure pass for efficiency, since they were found by the same review cycle.

---

_Verified: 2026-08-14T01:15:00Z_
_Verifier: Claude (gsd-verifier)_
