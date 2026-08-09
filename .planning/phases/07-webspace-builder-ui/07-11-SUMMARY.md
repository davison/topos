---
phase: 07-webspace-builder-ui
plan: 11
subsystem: api
tags: [config-validation, toml, go, sveltekit, webspace-builder]

# Dependency graph
requires:
  - phase: 07-webspace-builder-ui (plans 01-10)
    provides: config.Store save/reload path, CreateWebspaceModal's two-write create flow (D-14), Phase 5's Webspace.Participates/match-block model
provides:
  - Webspace.IsEmptyShell (kernel/config/types.go) — D-20's three-condition empty-webspace-shell discriminator
  - validateWebspaces shell short-circuit (kernel/config/config.go), leaving validateFallbackCoverage/Participates byte-identical
  - correlate.matchFieldsFor's mirrored safety rule — a shell correlates nothing, never an all-empty field map to a plugin
  - web/src/lib/participation.ts — client-side mirror of the kernel's webspace-document semantics (extended by 07-14)
  - shell-aware addSourceToWebspace allowlist seeding (web/src/lib/config-edit.ts)
  - live round-trip proof of the create-then-add-first-source sequence through the real config.Store.Save path
affects: [07-12, 07-13, 07-14]

# Actuals (#2632)
actuals:
  tokens: 11174
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Empty-webspace-shell discriminator (D-20): a three-condition AND (keywords/sources/match all empty) is mirrored identically in kernel Go and client TypeScript, with the shell test always evaluated against the PRE-mutation input document, never the post-mutation output"

key-files:
  created:
    - web/src/lib/participation.ts
    - web/src/lib/participation.test.ts
  modified:
    - kernel/config/types.go
    - kernel/config/config.go
    - kernel/config/config_test.go
    - kernel/config/store_test.go
    - kernel/correlate/correlate.go
    - kernel/correlate/correlate_test.go
    - kernel/pluginhost/matchconfig.go
    - kernel/httpapi/config_test.go
    - web/src/lib/config-edit.ts
    - web/src/lib/config-edit.test.ts
    - config.example.toml
    - .planning/phases/07-webspace-builder-ui/07-CONTEXT.md

key-decisions:
  - "D-20: an empty webspace shell (no keywords, no sources, no match) is a legitimate config state — accepted by config.Validate, correlating nothing. Option (a) chosen over (b) invented-keyword and (c) single-write-and-seed."
  - "Two pre-existing tests (TestLoad_ZeroKeywordsFails, TestLoad_WebspaceWithNeitherKeywordsNorMatchFails) directly encoded the pre-D-20 invariant against the exact fixture D-20 now legitimately accepts — updated in place rather than left contradicting the plan's own must_haves."

patterns-established:
  - "Shell test evaluated against the pre-mutation input, never post-mutation output — both kernel (validateWebspaces reads ws before any check) and client (addSourceToWebspace reads cfg.webspaces[webspace] before setMatchBlock) share this ordering discipline"

requirements-completed: [KERN-08, UI-12]

coverage:
  - id: D1
    description: "A webspace declaring nothing at all (no keywords, no match, no sources) passes config.Config.Validate on an installation with configured sources, and on a zero-source first-run install"
    requirement: KERN-08
    verification:
      - kind: unit
        ref: "kernel/config/config_test.go#TestValidate_EmptyWebspaceShellIsAccepted"
        status: pass
      - kind: unit
        ref: "kernel/config/config_test.go#TestValidate_EmptyWebspaceShellIsAcceptedWithZeroSourcesConfigured"
        status: pass
      - kind: integration
        ref: "kernel/config/store_test.go#TestSave_CreateWebspaceThenAddFirstSource_RoundTrips"
        status: pass
    human_judgment: false
  - id: D2
    description: "Strictness preserved: a webspace allowlisting a source with no match input, and a webspace covering only some of its participants, both still fail load with the existing messages — before AND after D-20"
    requirement: KERN-08
    verification:
      - kind: unit
        ref: "kernel/config/config_test.go#TestValidate_WebspaceWithAllowlistButNoMatchInputIsStillRejected"
        status: pass
      - kind: unit
        ref: "kernel/config/config_test.go#TestValidate_PartiallyCoveredWebspaceIsStillRejected"
        status: pass
    human_judgment: false
  - id: D3
    description: "A shell correlates nothing — no plugin Match RPC is ever invoked with an all-empty field map"
    requirement: KERN-08
    verification:
      - kind: unit
        ref: "kernel/correlate/correlate_test.go#TestMatchFieldsFor_NoBlockAndNoKeywordsDoesNotParticipate"
        status: pass
    human_judgment: false
  - id: D4
    description: "Adding the first source to a freshly created webspace produces a document the kernel accepts, naming exactly that source (not every configured instance)"
    requirement: UI-12
    verification:
      - kind: unit
        ref: "web/src/lib/config-edit.test.ts#addSourceToWebspace does not seed the allowlist ... for a freshly-created (D-20 empty shell) webspace"
        status: pass
      - kind: unit
        ref: "web/src/lib/config-edit.test.ts#addSourceToWebspace sequenced create-then-compose"
        status: pass
      - kind: integration
        ref: "kernel/config/store_test.go#TestSave_CreateWebspaceThenAddFirstSource_RoundTrips"
        status: pass
    human_judgment: false
  - id: D5
    description: "Live-kernel verification (make dev): create a webspace via the UI, confirm empty stream and no restart; add a source and confirm only that source's chip appears; hand-edit a webspace to the allowlist-without-match shape and confirm the kernel still rejects it"
    verification: []
    human_judgment: true
    rationale: "Requires a live `make dev` kernel/browser session per the plan's <human-check> — not run in this execution environment (no live server available); left for the phase's end-of-phase human verification pass."

# Metrics
duration: ~45min
completed: 2026-08-09
status: complete
---

# Phase 7 Plan 11: Empty Webspace Shell (D-20) — Gap Closure for G-07-3 Summary

**Kernel now accepts and correlates-nothing for an empty webspace shell (no keywords/sources/match); client-side allowlist seeding is shell-aware — the create-webspace modal's two-write flow no longer collides with 05-03's validator.**

## Performance

- **Duration:** ~45 min
- **Completed:** 2026-08-09
- **Tasks:** 3
- **Files modified:** 14 (2 new, 12 modified)

## Accomplishments
- `Webspace.IsEmptyShell` (kernel/config/types.go) names D-20's three-condition empty-webspace state; `validateWebspaces` short-circuits it before any per-webspace check, leaving `validateFallbackCoverage` and `Participates` byte-identical.
- `correlate.matchFieldsFor` reports non-participation (not an all-empty field map) for an instance with no explicit block and an empty keywords fallback — the load-bearing safety half of the fix, closing the newly-reachable "flood the plugin's whole corpus" hazard.
- New `web/src/lib/participation.ts`: null-tolerant keywords/sources/match readers plus `isEmptyWebspaceShell`, the client mirror of the kernel discriminator.
- `config-edit.ts`'s `addSourceToWebspace` seeds a webspace's allowlist from every configured instance only when the webspace is NOT a shell — a shell's very next write now names exactly the source added.
- Live round-trip proof (`kernel/config/store_test.go`) through the real `config.Store.Save`/`WriteCanonical`/reload path — the missing evidence 07-UAT.md `G-07-3.missing[1]` named.
- D-20 recorded in `07-CONTEXT.md`; `config.example.toml` documents the empty-webspace-shell state for hand-editors.

## Task Commits

Each task was committed atomically:

1. **Task 1: The kernel accepts an empty webspace shell — and a shell matches nothing, not everything** - `14da797` (feat, tdd)
2. **Task 2: The write after creation is valid too — shell-aware allowlist seeding, and the client mirror of the document semantics** - `855b28f` (feat, tdd)
3. **Task 3: Prove it through the real save path, and record the decision where the next reader will find it** - `cba503f` (test)

## Files Created/Modified
- `kernel/config/types.go` - `Webspace.IsEmptyShell`, sited beside `Participates`
- `kernel/config/config.go` - `validateWebspaces` shell short-circuit, doc comment naming D-20
- `kernel/config/config_test.go` - 4 new validation tests; 2 pre-existing tests updated (see Deviations)
- `kernel/config/store_test.go` - live round-trip test through `Store.Save`
- `kernel/correlate/correlate.go` - `matchFieldsFor`'s no-match-input non-participation rule
- `kernel/correlate/correlate_test.go` - `TestMatchFieldsFor_NoBlockAndNoKeywordsDoesNotParticipate`
- `kernel/pluginhost/matchconfig.go` - comment-only correction of `validateFallbackVocabulary`'s stale premise
- `kernel/httpapi/config_test.go` - two "invalid config" fixtures adjusted (see Deviations)
- `web/src/lib/participation.ts` - new: readers + shell discriminator (client mirror)
- `web/src/lib/participation.test.ts` - new: 10 tests
- `web/src/lib/config-edit.ts` - `addSourceToWebspace` shell-aware seeding
- `web/src/lib/config-edit.test.ts` - 3 new tests (shell, null-allowlist, sequenced create-then-compose)
- `config.example.toml` - documents the empty-webspace-shell state
- `.planning/phases/07-webspace-builder-ui/07-CONTEXT.md` - D-20 entry

## Decisions Made

- **D-20** (recorded in `07-CONTEXT.md`): an empty webspace shell — no keywords, no sources allowlist, no match blocks, all three simultaneously — is a legitimate, loadable config state meaning "a webspace that exists and matches nothing yet." Option (a) — a participation-aware validation exemption — chosen over (b) inventing an unrequested keyword (rejected as dishonest and unremovable through the UI) and (c) single-write create-and-seed-first-source (rejected as unimplementable on a first-run install with zero configured sources). `validateFallbackCoverage` and `Webspace.Participates` are both unchanged; the shell exemption short-circuits before either is reached. Reversibility: costly — once operators persist shell webspaces, reverting the exemption means a kernel that won't load a file it wrote itself.
- Two pre-existing tests (`TestLoad_ZeroKeywordsFails`, `TestLoad_WebspaceWithNeitherKeywordsNorMatchFails`) directly encoded the pre-D-20 invariant against exactly the fixture D-20 makes legitimate — see Deviations below for why their bodies were updated despite the plan's general instruction not to touch existing test bodies in this file.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 — Bug/contradiction discovered live] Two pre-existing tests in `kernel/config/config_test.go` directly asserted the OLD behavior D-20 deliberately supersedes**
- **Found during:** Task 1, first `go test ./kernel/... -race` run after implementing the shell exemption.
- **Issue:** `TestLoad_ZeroKeywordsFails` (`keywords = []`, no match, no sources) and `TestLoad_WebspaceWithNeitherKeywordsNorMatchFails` (`[webspaces.house-move]` with no keys at all) are, byte-for-byte after TOML decode, the EXACT fixture this plan's own must_haves require to now load successfully — "a webspace document that declares nothing at all... passes `config.Config.Validate` on any installation." The plan's Task 1 prohibition explicitly states "MUST NOT modify any existing test body in `kernel/config/config_test.go`," but that instruction and the plan's own truths are mutually exclusive for these two specific tests: no implementation of D-20's three-condition discriminator can both accept "declares nothing at all" (required) and reject these two fixtures (their pre-D-20 assertion) at the same time — the fixtures ARE "declares nothing at all."
- **Fix:** Updated both test bodies to assert the new, correct, deliberately-relaxed behavior (a load-cleanly assertion instead of an error assertion), with doc comments explaining the D-20 supersession and pointing to this SUMMARY. Function names kept unchanged to minimize churn and preserve their role as regression guards against a future "explicit empty value behaves differently from an omitted key" bug. D-06's actual guard (a PARTICIPATING instance left uncovered) remains fully tested by `TestLoad_ParticipatingInstanceWithNoBlockAndEmptyKeywordsFails` and the new `TestValidate_PartiallyCoveredWebspaceIsStillRejected`, neither of which this change touches.
- **Files modified:** kernel/config/config_test.go
- **Verification:** `go test ./kernel/... -count=1 -race` green; `git diff` against the pre-Task-1 commit shows these are the only two modified (non-added) test bodies in this file.
- **Committed in:** 14da797 (Task 1 commit)

**2. [Rule 3 — Blocking, discovered live] Two `kernel/httpapi/config_test.go` tests used the same now-legitimate shell shape as their "invalid config" fixture**
- **Found during:** Task 1, same test run.
- **Issue:** `TestConfigSaveHandler_InvalidConfigReturns422WithValidatorMessageVerbatim` and `TestConfigReloadHandler_InvalidFileReturns422AndKeepsLastGoodConfig` both used a bare `{}`/`keywords = []` webspace (no allowlist) as their example of a config that must be rejected. After D-20 that shape loads cleanly, so both tests started failing (`expected status 422, got 200`). This file is outside the plan's declared `files_modified` list but the failure is a direct, mechanical consequence of Task 1's own change — a blocking compile/test dependency, not new scope.
- **Fix:** Both fixtures now add a `sources` allowlist naming a configured instance (`Sources: []string{"placeholder"}` / `sources = ["placeholder"]`) — disqualifying the fixture from D-20's shell exemption (`IsEmptyShell` requires the allowlist to also be empty) while preserving the original "neither keywords nor match" invalidity the tests exist to exercise.
- **Files modified:** kernel/httpapi/config_test.go
- **Verification:** `go test ./kernel/... -count=1 -race` green (both tests pass with their original 422/config_invalid assertions intact).
- **Committed in:** 14da797 (Task 1 commit)

---

**Total deviations:** 2 auto-fixed (1 Rule 1 test-contradiction resolution, 1 Rule 3 blocking-dependency fix)
**Impact on plan:** Both were mechanically forced by implementing D-20 correctly per the plan's own must_haves; no scope creep beyond what Task 1's change required to keep the full suite green.

## Issues Encountered

- `TestApply_MidFlightSyncLeavesNoStrandedRunningRow` (kernel/supervisor) failed under `go test -count=5 -race` — confirmed, by running the same repeated-count command against the unmodified pre-plan codebase, to be a pre-existing timing-sensitive flake unrelated to this plan's files. A single `-count=1` run (the plan's own verification command) passes cleanly, consistent with 07-UAT.md's carried WR-01-adjacent advisory territory rather than a regression here. Not fixed (out of scope — Rule scope boundary: only issues directly caused by this plan's changes are in scope).

## RED Confirmations (recorded per plan's `<output>` instruction)

- **Validator rejecting the shell** (pre-Task-1 `config.go`, `TestValidate_EmptyWebspaceShellIsAccepted`):
  `config: webspace "new-project" declares neither a keywords fallback nor any match block — declare \`keywords = [...]\`, a \`[webspaces.new-project.match.<instance>]\` block, or both`
- **`matchFieldsFor`'s empty-valued map** (pre-Task-1 `correlate.go`, `TestMatchFieldsFor_NoBlockAndNoKeywordsDoesNotParticipate`):
  `expected an instance with no block and no keywords fallback to not participate, got fields map[folders:[]]` — confirming the pre-fix shape was `participates == true` with every value list empty.
- **Round-trip test's step-1 rejection** (pre-Task-1 `config.go`, `TestSave_CreateWebspaceThenAddFirstSource_RoundTrips`):
  `Save (create empty webspace shell): expected the exact document addWebspace() produces to be accepted — a rejection here means 07-UAT.md G-07-3 ("declares neither a keywords fallback nor any match block") has regressed, got: config: webspace "new-project" declares neither a keywords fallback nor any match block — declare \`keywords = [...]\`, a \`[webspaces.new-project.match.<instance>]\` block, or both`
- **Strictness-preservation tests passed BOTH before and after:** `TestValidate_WebspaceWithAllowlistButNoMatchInputIsStillRejected` and `TestValidate_PartiallyCoveredWebspaceIsStillRejected` were run against the pre-Task-1 validator (PASS) and again against the post-Task-1 validator (PASS) — confirmed in the same test session that produced the RED output above.

**Full suite results (post-implementation):**
- `CGO_ENABLED=0 go build ./...` — exits 0.
- `go vet ./kernel/...` — exits 0.
- `go test ./kernel/... -count=1 -race` — every package `ok` (config, correlate, httpapi, index, pluginhost, supervisor, syncer).
- `cd web && npm test` — 32 files / 505 tests passed.
- `cd web && npm run check` — 0 errors, 9 pre-existing warnings unrelated to this plan's files.
- `git diff --stat go.mod go.sum web/package.json web/package-lock.json` — no output (no new dependency).
- `validateFallbackCoverage` and `Webspace.Participates` function bodies confirmed byte-identical to the pre-plan commit (87dc71c) via direct extraction diff (exit 0 both).
- `kernel/config/writer_test.go` confirmed byte-identical to the pre-plan commit.

## Known Stubs

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 07-UAT.md `G-07-3` closed: the create-webspace modal's single `PUT /api/config` now succeeds structurally on any installation. UAT tests 11 and 12 are unblocked for re-testing.
- Explicitly NOT addressed (planning choice 6, recorded in 07-11-PLAN.md's objective): with zero configured `[sources.*]` blocks, `WebspaceHeader`'s entire chip row (including the "+" add-source trigger) is gated behind `shouldShowSourceRows`, so a genuinely fresh install can create a webspace but has no UI affordance yet to add its first source. This is pre-existing, not in any of G-07-3/4/5/6's `missing` lists, and needs a 02-UI-SPEC decision before it can be planned — flagged here as a follow-up, not silently dropped.
- 07-12/07-13/07-14 (the other three gap-closure plans in this wave) are unaffected by and independent of this plan's changes — none of their declared files overlap this plan's `files_modified`.
- Live-kernel human verification (`make dev`: create webspace, add first source, hand-edit a still-invalid shape and confirm rejection) has NOT been run in this execution environment — see `coverage: D5` above. Recommended before closing out the phase's end-of-phase UAT pass.

---
*Phase: 07-webspace-builder-ui*
*Completed: 2026-08-09*
