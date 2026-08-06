---
phase: 05-source-instances-per-type-matching
plan: 03
subsystem: api
tags: [go, toml, config-validation, plugin-contract, matching]

# Dependency graph
requires:
  - phase: 05-source-instances-per-type-matching
    plan: 01
    provides: "Source instance identity (item.Source, config-key-trusted) split from the Describe-learned plugin kind (item.SourceType)"
  - phase: 05-source-instances-per-type-matching
    plan: 02
    provides: "Typed match fields on the wire: MatchRequest.match_fields (map<string, StringList>), DescribeResponse.match_vocabulary; kernel/correlate.matchFieldsFor's D-01 fallback branch"
provides:
  - "config.MatchBlock and Webspace{Keywords, Sources, Match} — the per-instance typed match shape replacing the single shared keyword list"
  - "config.Validate's sorted-order structural validation of the new webspace shape (shape, unknown instances, dead config, zero/empty fields, fallback coverage)"
  - "kernel/correlate.matchFieldsFor's full D-01/D-02/D-03 resolution chain (allowlist, explicit block, fallback) and SyncSource's de-allowlisted-instance row-clearing path"
  - "kernel/pluginhost.ValidateMatchConfig — the post-launch, pre-sync vocabulary cross-check (D-05), wired into cmd/topos/main.go's setup()"
affects: [05-04, 05-05, 06-ui-scalable-source-surface, 07-webspace-builder-ui]

# Actuals (#2632)
actuals:
  tokens: 10820
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Two-phase config validation: config.Validate stays plugin-independent (structural checks only, runs before any plugin subprocess exists) while pluginhost.ValidateMatchConfig is a second, post-launch phase that needs the live *Host to cross-check declared vocabulary (05-RESEARCH.md Pitfall 1)"
    - "Sorted-order validation iteration: every new validation function (config.validateWebspaces/validateMatchBlocks/validateFallbackCoverage, pluginhost.ValidateMatchConfig) sorts webspace names, instance names, and field names before iterating, so the first reported error is deterministic run to run rather than dependent on Go's randomized map iteration order"
    - "Allowlist-then-block-then-fallback resolution: kernel/correlate.matchFieldsFor now returns (fields, participates) and checks D-03 (allowlist) before D-02 (explicit block) before D-01 (fallback) — a non-participating instance short-circuits before either block or fallback is even considered"

key-files:
  created:
    - kernel/pluginhost/matchconfig.go
    - kernel/pluginhost/matchconfig_test.go
  modified:
    - kernel/config/types.go
    - kernel/config/config.go
    - kernel/config/config_test.go
    - kernel/correlate/correlate.go
    - kernel/correlate/correlate_test.go
    - kernel/httpapi/webspaces.go
    - kernel/httpapi/agent.go
    - cmd/topos/main.go

key-decisions:
  - "05-RESEARCH.md Open Question 1 decided here as the plan specified: an instance excluded by a webspace's sources allowlist but still given an explicit match block in that same webspace is dead config and fails load at Task 1's validateMatchBlocks, naming both the webspace and the instance — not deferred to a warning"
  - "A field's value list itself being empty (zero-length, distinct from a whitespace-only string inside a non-empty list) is treated as a validation failure alongside the plan's explicitly-named 'empty field name' and 'empty/whitespace-only value' cases, since must_haves' framing ('rather than silently matching everything or nothing') covers a field with nothing to match against"
  - "kernel/pluginhost.ValidateMatchConfig also guards a match block or a fallback-relying instance naming a source with no launched plugin, even though pluginhost.Discover launching one subprocess per configured source means this should not occur for a real config — defensive completeness matching the plan's own behavior list ('the check fails rather than passing vacuously'), exercised via matchconfig_test.go's fake-host fixtures rather than a live subprocess mismatch"

patterns-established:
  - "Per-instance match resolution as a three-branch decision (allowlist, explicit block, fallback) fully implemented and unit-tested at the matchFieldsFor level, independent of the plugin-vocabulary cross-check that happens later at kernel startup"

requirements-completed: [KERN-07, KERN-06]

coverage:
  - id: D1
    description: "A webspace declares per-instance match blocks under [webspaces.<ws>.match.<instance>] with typed fields, plus an optional webspace-level keywords fallback applied to any participating instance with no explicit block (D-01), and an explicit block replaces the fallback outright for that instance (D-02)"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "kernel/config/config_test.go#TestLoad_MatchBlockDecodesNestedTOMLShape"
        status: pass
      - kind: unit
        ref: "kernel/correlate/correlate_test.go#TestMatchFieldsFor_ExplicitBlockReplacesFallback, TestMatchFieldsFor_FallbackFansAcrossTwoFieldVocabulary"
        status: pass
    human_judgment: false
  - id: D2
    description: "Every configured instance participates in every webspace by default; an optional webspace-level sources allowlist restricts participation, and a de-allowlisted instance's previously persisted rows are cleared at the next sync rather than orphaned (D-03)"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "kernel/correlate/correlate_test.go#TestMatchFieldsFor_DeallowlistedInstanceDoesNotParticipate, TestSyncSource_DeallowlistedInstanceRowsCleared"
        status: pass
    human_judgment: false
  - id: D3
    description: "A match field name the instance's plugin did not declare fails startup loudly, naming the field, the webspace, the instance, the plugin binary, and the vocabulary that plugin does declare (D-05)"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "kernel/pluginhost/matchconfig_test.go#TestValidateMatchConfig_UnknownFieldFailsNamingEverything"
        status: pass
    human_judgment: false
  - id: D4
    description: "Config validation and the post-launch vocabulary cross-check both iterate webspaces, instances, and field names in sorted order, so the first reported error is deterministic run to run regardless of Go map iteration order (KERN-07 ordering)"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "kernel/pluginhost/matchconfig_test.go#TestValidateMatchConfig_DeterministicOrderingAcrossTwoOffendingFields (count=20)"
        status: pass
    human_judgment: false
  - id: D5
    description: "The kernel rejects a config it cannot make sense of under the new shape (neither keywords nor match, unknown instances, dead config, zero/empty fields, uncovered fallback) with an error naming the offending webspace or instance (D-06)"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "kernel/config/config_test.go (seven new negative-case tests, TestLoad_WebspaceWithNeitherKeywordsNorMatchFails through TestLoad_ParticipatingInstanceWithNoBlockAndEmptyKeywordsFails)"
        status: pass
    human_judgment: false
  - id: D6
    description: "A rejected match config never leaves plugin subprocesses running: cmd/topos/main.go's setup() shuts the host down and closes the store when ValidateMatchConfig fails, and both `topos serve` and `topos sync` inherit the gate through setup"
    requirement: "KERN-07"
    verification: []
    human_judgment: true
    rationale: "The tear-down-on-failure wiring is structural (host.Shutdown()/store.Close() called on the new error path, mirroring the two pre-existing error paths in setup()) and not directly exercised by a test that launches a real subprocess and confirms it exits — a human should confirm this reasoning holds rather than auto-passing on indirect evidence."

duration: ~55min
completed: 2026-08-06
status: complete
---

# Phase 5 Plan 3: Source Instances & Per-Type Matching Summary

**Per-instance typed match blocks, an optional webspace-level keywords fallback, and a sources participation allowlist replace the single shared keyword list in `kernel/config`; `kernel/correlate.matchFieldsFor` resolves allowlist→block→fallback in that order; and a new `kernel/pluginhost.ValidateMatchConfig` cross-checks every match field against each launched plugin's declared vocabulary before any sync runs.**

## Performance

- **Duration:** ~55 min
- **Completed:** 2026-08-06
- **Tasks:** 2
- **Files modified:** 8 modified, 2 created

## Accomplishments

- `config.MatchBlock` (`map[string][]string`) and `Webspace{Keywords, Sources, Match}` replace `Webspace{Keywords}`; `Webspace.Participates(instance)` implements the D-03 allowlist check (empty `Sources` = everyone participates, non-empty = explicit membership only)
- `config.Validate`'s webspace half is rewritten as sorted-order structural checks, independent of any launched plugin: a webspace must declare a non-empty `keywords` fallback, at least one `match` block, or both; every `match`/`sources` entry must name a configured source; a match block for a source excluded by the same webspace's `sources` allowlist is rejected as dead config (05-RESEARCH.md Open Question 1, decided here); a match block must declare at least one field, no empty field name, and no field with zero or empty/whitespace-only values; a participating instance with no block in a webspace whose `keywords` is empty fails, naming both accepted shapes
- `kernel/correlate.matchFieldsFor` now returns `(fields, participates)` and resolves the full chain: allowlist first (a non-participating instance short-circuits to `false`, no Match call), then an explicit `ws.Match[src.Name()]` block returned verbatim (D-02 — never unioned with the fallback), then `ws.Keywords` fanned across the instance's declared vocabulary (D-01, unchanged from 05-02)
- `Engine.SyncSource` calls `Store.ReplaceWebspaceSourceItems(ctx, name, src.Name(), nil)` instead of `Match` for a non-participating instance, clearing its previously persisted rows for that webspace so a de-allowlisted instance never leaves orphaned rows behind (ROADMAP success criterion 3)
- `kernel/httpapi/webspaces.go` and `agent.go` gained a `keywordsOrEmpty` helper so a webspace relying entirely on match blocks (nil `Keywords`) serialises `keywords: []` rather than `null`
- New `kernel/pluginhost/matchconfig.go` implements D-05's second validation phase: `ValidateMatchConfig(cfg, host)` builds an instance-keyed index of launched plugins and, in sorted webspace/instance/field order, rejects an unknown match field (naming the field, the plugin's display name, its `source_type`, and the vocabulary it does declare), a match block or fallback naming an instance with no launched plugin, and a fallback-relying instance whose plugin declared an empty vocabulary
- `cmd/topos/main.go`'s `setup()` calls `ValidateMatchConfig` immediately after `pluginhost.Discover` returns, shutting the host down and closing the store on failure exactly like its two pre-existing error paths — both `runServe` and `runSync` inherit the gate through `setup`
- `make test` exits 0 across all six workspace modules (root/kernel, sdk, paperless, silverbullet, proton, mock, signal)

## Task Commits

Each task was committed atomically:

1. **Task 1: Per-instance match blocks, fallback, and participation allowlist** (tracer) - `79749fc` (feat)
2. **Task 2: Post-launch vocabulary cross-check fails startup by name** - `af1ab88` (feat)

_No separate TDD RED/GREEN commits: tests were written alongside their implementation within each task's single commit, consistent with this repo's existing per-task (not per-TDD-phase) commit granularity. Task 1's tracer feedback gate was satisfied in-run: this executor re-ran Task 1's own `<verify>` commands (`go test ./kernel/config/ ./kernel/correlate/ ./kernel/httpapi/`, `CGO_ENABLED=0 go build ./...`) before starting Task 2, all green._

## Files Created/Modified

- `kernel/config/types.go` - `MatchBlock`, `Webspace{Keywords, Sources, Match}`, `Webspace.Participates`
- `kernel/config/config.go` - `validateWebspaces`/`validateMatchBlocks`/`validateSourcesAllowlist`/`validateFallbackCoverage`, all sorted-order
- `kernel/config/config_test.go` - seven new negative cases plus one positive nested-TOML-decode case
- `kernel/correlate/correlate.go` - `matchFieldsFor(ws, src) (map[string][]string, bool)`, `SyncSource`'s de-allowlisted clear-and-skip branch
- `kernel/correlate/correlate_test.go` - `matchFieldsFor` unit tests (explicit-beats-fallback, two-field fan-out, de-allowlisted) plus a `SyncSource`-level rows-cleared regression test
- `kernel/httpapi/webspaces.go`, `agent.go` - `keywordsOrEmpty` nil→`[]` normalisation
- `kernel/pluginhost/matchconfig.go` (new) - `ValidateMatchConfig`, `validateMatchBlockVocabulary`, `validateFallbackVocabulary`, `joinVocabulary`
- `kernel/pluginhost/matchconfig_test.go` (new) - five cases: unknown field (exact error text), deterministic ordering (count=20), unlaunched instance, empty vocabulary with fallback, passing two-field vocabulary
- `cmd/topos/main.go` - `setup()` calls `ValidateMatchConfig` after `Discover`, tearing down on failure

## Decisions Made

- 05-RESEARCH.md's Open Question 1 (whether a `sources`-excluded instance may still carry a match block) is decided exactly as the plan specified: dead config, load-time error, not a staging feature.
- A field's value list being zero-length (not just containing a whitespace-only string) is treated as a validation failure, reading `must_haves`' "silently matching everything or nothing" framing as covering this case even though the plan's own three named sub-cases (zero fields, empty field name, empty/whitespace-only value) don't spell it out explicitly.
- `ValidateMatchConfig` also guards the "match block/fallback names an unlaunched instance" case even though `pluginhost.Discover` launching one subprocess per configured source means this shouldn't occur for a real config/host pairing — defensive completeness per the plan's own behavior list, verified with fake-plugin-list fixtures (`newTestPlugin`/`newTestHost`) rather than a live subprocess mismatch, exactly as Task 2's action block directed ("use a small fake plugin list rather than launching subprocesses").

## Deviations from Plan

None - plan executed exactly as written. Both tasks' declared `files_modified` lists matched the files actually touched.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Plans 05-04/05-05 (rendition centralization, docs/config.example.toml republishing) can build on this plan's shipped `[webspaces.<ws>.match.<instance>]` shape and `ValidateMatchConfig` gate without further config-loader rework.
- `config.example.toml` still shows only the pre-Phase-5 `keywords`-only shape (05-01-SUMMARY.md already flagged this as a follow-up); a later plan touching that file should add a `match`/`sources` example so operators discover the new shape from the example config, not only from docs.
- Every currently-configured real webspace in this repo (none checked into version control) that relies solely on `keywords` continues to work unchanged — the fallback path is byte-identical to 05-02's shipped behavior.

---
*Phase: 05-source-instances-per-type-matching*
*Completed: 2026-08-06*

## Self-Check: PASSED

All key files verified present on disk; both task commit hashes (`79749fc`, `af1ab88`) verified present in `git log --oneline --all`.
