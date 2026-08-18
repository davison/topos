---
phase: 12-filesystem-source
plan: 03
subsystem: source-plugin
tags: [go-plugin, filesystem, filepath.WalkDir, symlink-policy, ast-guard, e2e]

requires:
  - phase: 12-filesystem-source
    provides: "12-01's file://-scheme deep-link convention and kernel-mediated open route; 12-02's document-scope classifier, extras-driven glob resolution, and preview-shape Fetch dispatch this plan's walk() feeds into"

provides:
  - "kernel/config.Source.Recursive — a typed, omitempty boolean published in config.toml and the config API's JSON shape, end to end into the plugin launch envelope (WEBSPACES_SOURCE_CONFIG's top-level recursive key, always emitted, never folded into extras)"
  - "plugins/filesystem/walk.go — recursion-aware, symlink-safe, permission-tolerant filepath.WalkDir traversal returning the complete current item set every call, with a 25,000-item per-sync cap"
  - "plugins/filesystem/readonly_test.go — committed AST guard (PLUG-02) failing the build on any os-package write selector, with negative-control fixtures"
  - "Honest filesystem Health: os.ReadDir distinguishes a permission-denied root from an empty-but-readable one — os.Stat alone could not (T-12-16)"

affects: [12-04, 12-05]

actuals:
  tokens: 16400
  tasks: 3
  commits: 4

tech-stack:
  added: []
  patterns:
    - "Typed config.Source boolean, omitempty on both toml/json tags, threaded through pluginhost's NAMED sourceConfigEnvelope as an always-emitted top-level scalar (never omitempty on the envelope side — false is a meaningful present value there, distinct from 'unset')"
    - "filepath.WalkDir-based tree traversal that never auto-follows a symlinked directory (WalkDir's own Lstat semantics close the ancestor-symlink-loop class structurally, with no cycle-detection code needed) — the reusable shape for any future local-path plugin needing recursion"
    - "Health probes via the SAME read op the real workload needs (os.ReadDir, not os.Stat) so a directory that exists-but-denies-entry is distinguishable from one that is empty — os.Stat alone cannot make this distinction on Linux, since stat(2) only needs search permission on the PARENT, not the target"

key-files:
  created:
    - plugins/filesystem/walk.go
    - plugins/filesystem/walk_test.go
    - plugins/filesystem/health_test.go
    - plugins/filesystem/readonly_test.go
    - web/e2e/specs/12-filesystem-recursion.spec.ts
  modified:
    - kernel/config/types.go
    - kernel/config/config_test.go
    - kernel/config/store_test.go
    - kernel/pluginhost/host.go
    - kernel/pluginhost/env_test.go
    - kernel/pluginhost/extras_test.go
    - plugins/filesystem/main.go
    - plugins/filesystem/plugin.go
    - plugins/filesystem/item.go
    - plugins/filesystem/fetch_test.go
    - plugins/filesystem/item_test.go
    - web/e2e/fixtures/config-builder.ts

key-decisions:
  - "NewSourcePlugin's signature gained a third recursive bool parameter in Task 1 (not deferred to Task 2), because main.go — the only file that can pass config.Source.Recursive into the plugin — is not in Task 2's file list at all. Without this, Task 2 would have had nothing to read the config value from."
  - "Health switched from os.Stat to os.ReadDir: Stat alone cannot detect a permission-denied root on Linux (stat(2) only needs search permission on the parent directory, not on the target), which would have silently collapsed 'unreadable' and 'reachable' into the same health state — exactly the T-12-16 distinction this task exists to guarantee."
  - "Symlinked-directory non-descent is implemented structurally (filepath.WalkDir's own Lstat semantics never auto-follow a symlink) rather than via cycle detection — an ancestor-pointing symlink is provably safe by construction, not merely tested against one corpus shape."

requirements-completed: [SRC-04]

coverage:
  - id: D1
    description: "The recursive config key is typed end to end: a TOML source block declaring it decodes correctly, a canonical rewrite never introduces it into a source that never declared it, and it reaches the plugin subprocess as a top-level envelope boolean outside extras"
    requirement: SRC-04
    verification:
      - kind: unit
        ref: "kernel/config/config_test.go#TestLoad_RecursiveTrueDecodesTrueOmittedDecodesFalse"
        status: pass
      - kind: unit
        ref: "kernel/config/store_test.go#TestStore_Save_RecursiveKeyOmittedWhenNeverDeclaredPreservedWhenTrue"
        status: pass
      - kind: unit
        ref: "kernel/pluginhost/env_test.go#TestSourceConfigEnvelope_RecursiveKeyPresentOutsideExtras"
        status: pass
    human_judgment: false

  - id: D2
    description: "Recursion toggles between top-level-only and full-depth walking, each source_id forward-slash relative, proven against a real kernel and a real plugin binary — add and remove both reflected with no watcher dependency"
    requirement: SRC-04
    verification:
      - kind: unit
        ref: "plugins/filesystem/walk_test.go#TestWalk_NonRecursiveOnlyTopLevelFilesBecomeItems"
        status: pass
      - kind: unit
        ref: "plugins/filesystem/walk_test.go#TestWalk_RecursiveFilesAtEveryDepthBecomeItems"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/12-filesystem-recursion.spec.ts — both tests, chromium project"
        status: pass
    human_judgment: false

  - id: D3
    description: "A nested file carries per-segment folder labels plus the cumulative relative path (D-05), so a webspace match block can name either a subfolder or a full relative path"
    requirement: SRC-04
    verification:
      - kind: unit
        ref: "plugins/filesystem/walk_test.go (folder-labelled items exercised via the recursion e2e spec's nested-subfolder keyword match)"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/12-filesystem-recursion.spec.ts — nested item matched via the 'receipts' folder keyword"
        status: pass
    human_judgment: false

  - id: D4
    description: "A real awkward tree — symlink loop, permission-denied subtree, dot-directory — walks to completion and returns exactly the documents an operator would expect"
    requirement: SRC-04
    verification:
      - kind: unit
        ref: "plugins/filesystem/walk_test.go#TestWalk_SymlinkPointingAtAncestorCompletesWithoutHanging"
        status: pass
      - kind: unit
        ref: "plugins/filesystem/walk_test.go#TestWalk_PermissionDeniedSubdirectoryIsSkippedWalkCompletes"
        status: pass
      - kind: unit
        ref: "plugins/filesystem/walk_test.go#TestWalk_FileInsideDotDirectorySkippedByDefault"
        status: pass
      - kind: unit
        ref: "plugins/filesystem/walk_test.go#TestWalk_IncludeGlobBringsBackDotFileAndDotDirectoryContents"
        status: pass
    human_judgment: false

  - id: D5
    description: "A cancelled context or a fatal root-read error returns an error rather than a partial set; a tree exceeding the per-sync item cap fails naming the cap and exclude_glob"
    requirement: SRC-04
    verification:
      - kind: unit
        ref: "plugins/filesystem/walk_test.go#TestWalk_CancelledContextAbortsWithErrorAndNoPartialSet"
        status: pass
      - kind: unit
        ref: "plugins/filesystem/walk_test.go#TestWalk_NonExistentRootReturnsErrorNotEmptyResult"
        status: pass
      - kind: unit
        ref: "plugins/filesystem/walk_test.go#TestWalk_UnreadableRootReturnsErrorNotEmptyResult"
        status: pass
      - kind: unit
        ref: "plugins/filesystem/walk_test.go#TestWalk_ExceedingItemCapReturnsErrorNamingCapAndExcludeGlob"
        status: pass
    human_judgment: false

  - id: D6
    description: "Health distinguishes a readable-and-empty root from an unreachable one, with the OS error named as the cause"
    requirement: SRC-04
    verification:
      - kind: unit
        ref: "plugins/filesystem/health_test.go#TestHealth_EmptyFolderAndMissingMountAreDistinguishable"
        status: pass
      - kind: unit
        ref: "plugins/filesystem/health_test.go#TestHealth_UnreadableRootReportsUnreachableWithOSErrorCause"
        status: pass
    human_judgment: false

  - id: D7
    description: "No non-test Go file in the filesystem plugin package references any os-package write selector — the guard is a committed AST scan with its own negative-control fixtures"
    requirement: SRC-04
    verification:
      - kind: unit
        ref: "plugins/filesystem/readonly_test.go#TestPluginIssuesNoWrite (incl. both negative-control fixture assertions)"
        status: pass
      - kind: unit
        ref: "go test ./internal/audit/ -count=1 (repo-wide egress audit unaffected — this plugin needs no allowlist entry, no outbound requests at all)"
        status: pass
    human_judgment: false

duration: ~45min
completed: 2026-08-13
status: complete
---

# Phase 12 Plan 03: The Folder as a Real Folder Summary

**Subfolder recursion is now a typed on/off config.toml key threaded through the launch envelope into a symlink-safe, permission-tolerant `filepath.WalkDir` traversal that returns the complete current item set every sync, with honest health and a committed read-only guard.**

## Performance

- **Duration:** ~45 min
- **Tasks:** 3
- **Files modified:** 17 (5 created, 12 modified)

## Accomplishments

- `kernel/config.Source.Recursive` (typed, `omitempty` on both `toml`/`json` tags) is now a published config key, threaded unmodified through `pluginhost.sourceConfigEnvelope` as an always-emitted top-level boolean (never folded into `extras`, never `omitempty` on the envelope side — false is a meaningful present value there) into `plugins/filesystem`'s `sourceConfig`, and finally into `SourcePlugin`'s own `recursive` field.
- `plugins/filesystem/walk.go`: the tree traversal moved out of the 12-01/12-02 tracer's inline `os.ReadDir` into a real `filepath.WalkDir`-based walker. Recursion toggles top-level-only vs. every-depth; a dot-prefixed file or directory is skipped by default unless an `include_glob` pattern explicitly names it; a symlinked directory is never descended into (WalkDir's own Lstat semantics make this structural, not cycle-detected — an ancestor-pointing symlink is provably safe); a symlinked regular file is included with its resolved real path re-validated against the root (T-12-12); a per-entry permission error skips that subtree and the walk completes; a root-read failure or a cancelled context aborts with an error, never a partial set; and a 25,000-item cap fails naming `exclude_glob`.
- `plugins/filesystem/item.go`'s `folderLabels` now emits, for a nested file, one label per containing-directory segment plus the cumulative relative path (D-05) — a webspace match block can name either a bare subfolder or a full relative path.
- `Health` switched from `os.Stat` to `os.ReadDir`: `os.Stat` alone cannot distinguish a permission-denied root from an empty-but-readable one on Linux (stat(2) needs search permission on the *parent*, not the target), which would have silently collapsed the T-12-16 distinction this task exists to guarantee.
- `plugins/filesystem/readonly_test.go`: a committed AST guard (PLUG-02) scoped to this package's own directory, flagging any `os.<selector>` call naming a write operation, with two negative-control fixtures proving the scan isn't vacuous.
- `web/e2e/specs/12-filesystem-recursion.spec.ts`: two source instances (one `recursive = true`, one not) over the same corpus prove the nested item appears only through the recursive instance; deleting the file and triggering a re-sync through the existing refresh route removes it — criterion 2's add/remove claim, driven end to end against a real kernel and a real plugin binary.

## Task Commits

1. **Task 1: The recursive toggle, typed from config.toml to the plugin subprocess** — `377dfb7` (feat)
2. **Task 2 (RED): failing tests for recursion-aware walk and honest health** — `9d38b06` (test)
3. **Task 2 (GREEN): recursion-aware walk, symlink/hidden policy, honest health** — `0974edc` (feat)
4. **Task 3: The committed read-only guard** — `18ef371` (test)

**Plan metadata:** this SUMMARY's own commit (pending, see below)

## Files Created/Modified

- `kernel/config/types.go` — `Source.Recursive bool`, `toml:"recursive,omitempty" json:"recursive,omitempty"`
- `kernel/config/config_test.go`, `store_test.go` — decode true/omitted-false, Validate acceptance, canonical-rewrite omit/preserve round trip
- `kernel/pluginhost/host.go` — `sourceConfigEnvelope.Recursive` (always-emitted, outside `extras`), wired into `launch`
- `kernel/pluginhost/env_test.go` — envelope-document assertions for the new key
- `kernel/pluginhost/extras_test.go` — fixed `TestSourceConfigEnvelope_TopLevelKeyNamesUnchanged` (see Deviations)
- `plugins/filesystem/main.go` — decodes `recursive` from `WEBSPACES_SOURCE_CONFIG`
- `plugins/filesystem/plugin.go` — `NewSourcePlugin`'s new `recursive` param; `Match` delegates to `walk()`; `toItem` takes any-depth `sourceID`; `Health` reworked
- `plugins/filesystem/item.go` — `folderLabels` extended for nested paths
- `plugins/filesystem/fetch_test.go`, `item_test.go` — updated `NewSourcePlugin` call sites
- `plugins/filesystem/walk.go`, `walk_test.go` — the new traversal and its test suite
- `plugins/filesystem/health_test.go` — the honest-degradation test suite
- `plugins/filesystem/readonly_test.go` — the committed AST write-guard
- `web/e2e/fixtures/config-builder.ts` — `FixtureSourceSpec.recursive?: boolean`, emitted only when true
- `web/e2e/specs/12-filesystem-recursion.spec.ts` — the new e2e spec

## Decisions Made

- **`NewSourcePlugin`'s signature change landed in Task 1, not Task 2:** Task 1's action text says main.go should "pass [recursive] into the constructed plugin implementation," but `main.go` is absent from Task 2's file list entirely — Task 2 could never have received the config value otherwise. Wiring the field through in Task 1 (with all existing call sites updated to pass `false`, preserving current behavior) is what let Task 2 consume `p.recursive` from a stable, already-tested seam.
- **Health probes via `os.ReadDir`, not `os.Stat`:** see key-decisions above — this is a correctness requirement (T-12-16), not a style choice.
- **Symlinked-directory non-descent is structural:** `filepath.WalkDir` uses Lstat semantics for every walked entry and never auto-follows a symlink, so a symlinked directory simply never gets a chance to be traversed — no cycle-detection bookkeeping was written, and the ancestor-pointing-symlink test proves this by construction rather than by exhausting a search-depth budget.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Fixed `TestSourceConfigEnvelope_TopLevelKeyNamesUnchanged` after adding a non-string envelope field**
- **Found during:** Task 1, running `go test ./kernel/pluginhost/...` after adding `sourceConfigEnvelope.Recursive`
- **Issue:** The existing test decoded the marshalled envelope into `map[string]string`. Once the envelope carried a non-string scalar (`Recursive`, a `bool`), `json.Unmarshal` failed with `cannot unmarshal bool into Go value of type string` — an unrelated pre-existing test broke as a direct, mechanical consequence of this task's own change.
- **Fix:** Changed the decode target to `map[string]any`; the string-keyed assertions below it are unaffected.
- **Files modified:** `kernel/pluginhost/extras_test.go`
- **Verification:** `go test ./kernel/pluginhost/... -count=1` passes in full.
- **Committed in:** `377dfb7` (Task 1 commit)

**2. [Rule 3 - Blocking issue] `NewSourcePlugin`'s signature extended beyond Task 1's declared file list**
- **Found during:** Task 1, implementing "pass it into the constructed plugin implementation"
- **Issue:** `plugins/filesystem/plugin.go` is not listed in Task 1's `<files>`, but `NewSourcePlugin` (declared there) is the only place that could receive `cfg.Recursive` from `main.go` — without touching it, the decoded config value would have had nowhere to go, and Task 2 (which doesn't touch `main.go`) would have had no way to read it either.
- **Fix:** Added a `recursive bool` parameter to `NewSourcePlugin` and a matching `SourcePlugin.recursive` field; updated all 13 existing call sites (`fetch_test.go` x11, `item_test.go` x2) to pass `false`, preserving current non-recursive test behavior unchanged.
- **Files modified:** `plugins/filesystem/plugin.go`, `plugins/filesystem/fetch_test.go`, `plugins/filesystem/item_test.go`
- **Verification:** `cd plugins/filesystem && go test ./... -count=1` passes in full.
- **Committed in:** `377dfb7` (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (1 Rule 1 — a pre-existing test broken by this task's own field addition; 1 Rule 3 — a blocking gap in the plan's own task/file split, necessary for Task 2 to be implementable at all)
**Impact on plan:** Both were necessary for correctness and for the plan's own two-task split to actually work end to end. No scope creep.

## Issues Encountered

- Two stray `plugins/filesystem/filesystem` binaries were produced by `go build ./...` invocations with no `-o` flag inside the plugin's own directory during Tasks 1–3 — deleted before each commit, same as 12-01/12-02's own notes.
- `npm run build` (invoked by `make e2e`) again overwrote the gitignored `kernel/webui/build/.gitkeep` placeholder — restored via `git checkout -- kernel/webui/build/.gitkeep` before committing Task 2, same as 12-01's note.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- The typed `recursive` config key, the recursion-aware walk, honest health, and the read-only guard are all real, tested machinery — later plans in this phase (connection-form UI, docs, external-tier rehearsal) can build directly on top without touching this plan's own files again.
- `plugins/filesystem` now has full parity with the other in-repo plugins' committed-guard convention (`readonly_test.go`) — 12-RESEARCH.md's recommended project structure item that 12-02's summary flagged as still missing is now closed.
- D6 (the narrow-layout "Couldn't open" overflow backstop, carried forward from 12-01/12-02) remains genuinely unverified — untouched by this plan's scope.

## Self-Check: PASSED

- FOUND: `plugins/filesystem/walk.go`, `plugins/filesystem/walk_test.go`, `plugins/filesystem/health_test.go`, `plugins/filesystem/readonly_test.go`, `web/e2e/specs/12-filesystem-recursion.spec.ts`
- FOUND commits: `377dfb7`, `9d38b06`, `0974edc`, `18ef371` (all present in `git log --oneline`)

---
*Phase: 12-filesystem-source*
*Completed: 2026-08-13*
