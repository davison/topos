---
phase: 05-source-instances-per-type-matching
plan: 02
subsystem: api
tags: [go, protobuf, grpc, plugin-contract, matching]

# Dependency graph
requires:
  - phase: 05-source-instances-per-type-matching
    plan: 01
    provides: "Source instance identity (item.Source, config-key-trusted) split from the Describe-learned plugin kind (item.SourceType)"
provides:
  - "Typed, plugin-declared match fields on the wire: MatchRequest.match_fields (map<string, StringList>), DescribeResponse.match_vocabulary"
  - "sdk.Handshake.ProtocolVersion 2 — a stale plugin binary fails at handshake, not at first Match"
  - "kernel/correlate.matchFieldsFor — the D-01 fallback resolving a webspace's keywords list into every field of an instance's declared vocabulary"
  - "All five in-repo plugins (mock, paperless, silverbullet, proton, signal) declare a match vocabulary and read typed match_fields"
affects: [05-03, 05-04, 05-05]

# Actuals (#2632)
actuals:
  tokens: 19353
  tasks: 3
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Generic field-name-to-value-list map on the wire (option-a): MatchRequest.match_fields is a map<string, StringList> with no plugin-type-specific field names in the proto — every plugin declares its own vocabulary via DescribeResponse.match_vocabulary, so a future plugin type needs no proto change (D-05)"
    - "Contract generation versioned independently of the proto package path: contract_version is now \"topos.v2\" while the proto package stays \"topos.v1\" — the actual breaking-change fail-fast is sdk.Handshake.ProtocolVersion, not the proto package name"
    - "Per-instance field scoping (T-05-07): correlate.matchFieldsFor is called once per (webspace, source) pair and returns only that one instance's own declared fields — a Match RPC never carries another instance's match configuration"

key-files:
  created: []
  modified:
    - proto/topos/v1/plugin.proto
    - sdk/gen/topos/v1/plugin.pb.go
    - sdk/shared.go
    - sdk/contract_test.go
    - kernel/pluginhost/host.go
    - kernel/correlate/correlate.go
    - kernel/correlate/correlate_test.go
    - kernel/syncer/coordinator_test.go
    - kernel/syncer/scheduler_test.go
    - plugins/mock/plugin.go
    - plugins/mock/plugin_test.go
    - plugins/paperless/plugin.go
    - plugins/paperless/client_test.go
    - plugins/silverbullet/plugin.go
    - plugins/silverbullet/match_test.go
    - plugins/proton/plugin.go
    - plugins/proton/imap_transcript_test.go
    - plugins/proton/fetch_rendition_test.go
    - plugins/proton/mailbox_cache_test.go
    - plugins/proton/item_test.go
    - plugins/proton/live_bridge_test.go
    - plugins/signal/plugin.go
    - plugins/signal/byte_identical_test.go

key-decisions:
  - "Task 1 checkpoint locked option-a exactly as proposed: generic field map, proto package stays topos.v1, sdk.Handshake.ProtocolVersion 1->2, DescribeResponse.contract_version becomes \"topos.v2\" (contract generation, documented as independent of the proto package path), repeated string match_vocabulary = 4 on DescribeResponse, webspace keywords fallback fans into every field of an instance's declared vocabulary"
  - "kernel/syncer/coordinator_test.go and scheduler_test.go's fakeSource/countingSource fixtures updated in Task 2's commit (Rule 3, blocking compile dependency) — outside this plan's declared file list, but go test ./kernel/... would not build otherwise since both types must satisfy correlate.Source's new Match(map[string][]string) signature"
  - "plugins/proton's five test files beyond the plan's declared imap_transcript_test.go (fetch_rendition_test.go, mailbox_cache_test.go, item_test.go, live_bridge_test.go) all switched to a shared foldersMatchReq helper in Task 3's commit (Rule 3) — every one of them builds a MatchRequest and is in the same package, so all five needed the same mechanical update or the module would not compile"
  - "plugins/signal's D-06 contract-change regression test landed in byte_identical_test.go, not message_test.go as the plan's files_modified list named (Rule 3) — the test needs the fixture-database Match() entry point (buildFixtureDatabase, NewSourcePlugin) that only byte_identical_test.go has; message_test.go tests parseMessage in isolation and has no Match-level fixture infrastructure"

patterns-established:
  - "Typed match-field vocabulary: every plugin declares its own field names via Describe's match_vocabulary and reads only its declared keys from MatchRequest.match_fields, ignoring any other key present (D-05) — proven by an ignore-undeclared-key test in every one of the five plugins"

requirements-completed: [KERN-07]

coverage:
  - id: D1
    description: "Every plugin declares its own match-field vocabulary in Describe, and the kernel interprets a match block without any kernel-side table of known plugin types (D-05)"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "plugins/mock/plugin_test.go#TestDescribe_ReturnsMockIdentity, plugins/paperless/client_test.go#TestMatch_ReadsTypedTagsFieldAndIgnoresUndeclaredKey, plugins/proton/imap_transcript_test.go#TestDescribe_DeclaresFoldersVocabulary, plugins/signal/byte_identical_test.go#TestDescribe_DeclaresConversationsVocabulary"
        status: pass
    human_judgment: false
  - id: D2
    description: "Match values are compared exactly and case-insensitively via strings.EqualFold with no Unicode normalization — the typed fields inherit Phase 1 D-03's determinism guarantee unchanged (D-04)"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "plugins/mock/plugin_test.go#TestMatch_NoSubstringMatching, plugins/paperless/client_test.go#TestResolveTagIDs_ExactCaseInsensitive_NoSubstringMatch (unchanged), plugins/silverbullet's tagsMatchAnyKeyword/pageNameMatchesAnyKeyword"
        status: pass
    human_judgment: false
  - id: D3
    description: "A plugin treats a match-field key it did not declare as absent rather than as an error"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "TestMatch_UndeclaredKeyIsIgnored in plugins/mock, plugins/paperless (client_test.go), plugins/silverbullet (match_test.go), plugins/proton (imap_transcript_test.go), plugins/signal (byte_identical_test.go) — one per plugin"
        status: pass
    human_judgment: false
  - id: D4
    description: "A plugin binary built against the previous contract fails at the go-plugin handshake, not confusingly at its first Match call"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "sdk/contract_test.go#TestContractDeclaresMatchVocabulary confirms the reserved keywords clause; sdk.Handshake.ProtocolVersion bumped 1->2 (structural — not directly exercised by a live stale-binary test in this plan)"
        status: pass
      human_judgment: true
      rationale: "The handshake fail-fast itself is go-plugin library behavior triggered by the ProtocolVersion mismatch, not something this plan's own test suite drives end-to-end against a real stale subprocess; a human should confirm this reasoning holds rather than auto-passing on indirect evidence."
  - id: D5
    description: "A webspace with only a keywords list still matches every configured instance exactly as it did before this phase — the fallback is applied to every field in that instance's declared vocabulary (D-01)"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "kernel/correlate/correlate_test.go (all pre-existing SyncSource tests pass unmodified in behavior via matchFieldsFor's single-vocabulary-field fallback)"
        status: pass
    human_judgment: false
  - id: D6
    description: "Signal's 1:1 conversation matching still considers only the nickname and system-contact name, never the profile name and never Note-to-Self, after moving from the shared keyword list to its own conversations field (Phase 4 D-06)"
    requirement: "KERN-07"
    verification:
      - kind: unit
        ref: "plugins/signal/byte_identical_test.go#TestMatch_ProfileNameOnlyAndNoteToSelfNeverMatch_SurvivesContractChange"
        status: pass
    human_judgment: false

duration: ~2h
completed: 2026-08-06
status: complete
---

# Phase 5 Plan 2: Source Instances & Per-Type Matching Summary

**Replaced the shared `MatchRequest{keywords}` list with a typed, plugin-declared `match_fields` map (option-a: generic field-name map, proto package stays `topos.v1`, handshake bumped 1→2) across the published contract, the SDK, the kernel's correlation engine, and all five in-repo plugins — a webspace's `keywords` fallback still reproduces pre-phase behaviour byte for byte.**

## Performance

- **Duration:** ~2h (including a `checkpoint:decision` and a tracer feedback checkpoint, both resolved by the coordinator)
- **Tasks:** 3 (1 decision checkpoint, 2 code tasks)
- **Files modified:** 22
- **Commits:** 2

## Accomplishments

- `proto/topos/v1/plugin.proto`: `MatchRequest.keywords` retired via `reserved 1; reserved "keywords";`, replaced by `map<string, StringList> match_fields`; `DescribeResponse` gains `repeated string match_vocabulary`; `contract_version` bumped to `"topos.v2"` (the contract *generation*, explicitly documented as independent of the unchanged `topos.v1` proto package path)
- `sdk.Handshake.ProtocolVersion` 1→2 — a plugin binary built against the pre-Phase-5 contract now fails cleanly at the go-plugin handshake instead of silently misinterpreting an empty/garbled match map at its first `Match` call
- `sdk/contract_test.go` gains `TestContractDeclaresMatchVocabulary`, pinning both new fields and that `"keywords"` survives only inside a `reserved` clause; the pre-existing `TestContractRPCAllowlist` and `TestContractEnumsZeroValueUnspecified` pass unwidened
- `kernel/correlate.matchFieldsFor` implements the D-01 fallback: a webspace's `keywords` list is fanned into every field of an instance's declared vocabulary (`src.MatchVocabulary()`), scoped to exactly that one instance's own fields per call — never the webspace's whole match configuration (T-05-07)
- `kernel/pluginhost.Plugin` caches `Describe`'s `match_vocabulary` and exposes `MatchVocabulary() []string`; `Plugin.Match` now takes `map[string][]string` and wraps each value slice in a `StringList` for the wire
- All five in-repo plugins declare their own vocabulary and read typed fields, with the comparison logic itself untouched in every case (only the input's provenance changed):
  - `mock`: `["labels"]`
  - `paperless`: `["tags"]` — `client.ResolveTagIDs`'s exact, case-insensitive tag lookup unchanged
  - `silverbullet`: `["tags", "pages"]` — a page matches if its tags match any `tags` value OR its page name matches any `pages` value (unioned, independent fields); `tagsMatchAnyKeyword`/`pageNameMatchesAnyKeyword` split `frontmatter.go`'s previously-combined `MatchesKeyword` check without modifying `frontmatter.go` itself
  - `proton`: `["folders"]` — mailbox-leaf-name matching and the mailbox cache's last-writer-wins merge behaviour unchanged
  - `signal`: `["conversations"]` — `match.go`'s `candidateNames`/`eligibleConversations`/`matchesAnyKeyword` bodies are byte-identical (confirmed via an empty `git diff plugins/signal/match.go`); a new end-to-end regression test proves Phase 4 D-06 (profile-name-only and Note-to-Self conversations never match) survives through the new typed contract entry point, not just at the unit-test layer
- Every plugin gained an "undeclared key is ignored" test and (where a value list is meaningfully testable) an "empty value list matches nothing" test
- `make test` exits 0 across all six workspace modules (root/kernel, sdk, paperless, silverbullet, proton, mock, signal)

## Task Commits

1. **Task 1: Lock the published match contract's shape** — checkpoint:decision, no code commit. User selected **option-a** exactly as proposed.
2. **Task 2: Typed match fields on the wire, proven through the reference plugin** (tracer) — `f65fcd6` (feat) — proto, SDK, `kernel/pluginhost`, `kernel/correlate`, `plugins/mock`, plus the blocking `kernel/syncer` fixture fix (Rule 3)
3. **Task 3: The four real plugins declare their vocabulary and read typed fields** — `edc541d` (feat) — `plugins/paperless`, `plugins/silverbullet`, `plugins/proton`, `plugins/signal` and their test suites

_A tracer feedback checkpoint (`type="tracer"`) was inserted between Task 2 and Task 3 per this repo's execution workflow: the tracer's `<verify>` commands were re-run and confirmed passing before Task 3 began. The user independently re-ran verification and hit the expected intermediate breakage (`plugins/paperless` failing to build against the retired `GetKeywords`) — exactly the state Task 3 existed to resolve._

## Files Created/Modified

- `proto/topos/v1/plugin.proto`, `sdk/gen/topos/v1/plugin.pb.go`, `sdk/shared.go`, `sdk/contract_test.go` — the published contract and its generated stubs
- `kernel/pluginhost/host.go`, `kernel/correlate/correlate.go`, `kernel/correlate/correlate_test.go` — the kernel's field-resolution and RPC-building layer
- `kernel/syncer/coordinator_test.go`, `kernel/syncer/scheduler_test.go` — test fixture signature fix (Rule 3)
- `plugins/mock/plugin.go`, `plugins/mock/plugin_test.go` — reference implementation
- `plugins/paperless/plugin.go`, `plugins/paperless/client_test.go`
- `plugins/silverbullet/plugin.go`, `plugins/silverbullet/match_test.go`
- `plugins/proton/plugin.go`, `plugins/proton/imap_transcript_test.go`, `plugins/proton/fetch_rendition_test.go`, `plugins/proton/mailbox_cache_test.go`, `plugins/proton/item_test.go`, `plugins/proton/live_bridge_test.go`
- `plugins/signal/plugin.go`, `plugins/signal/byte_identical_test.go`

## Decisions Made

- **Task 1 checkpoint:** option-a locked exactly as proposed (see key-decisions in frontmatter for the full shape).
- `kernel/syncer/coordinator_test.go`/`scheduler_test.go` updated in Task 2's commit under deviation Rule 3 — `go test ./kernel/...` (part of Task 2's own `<verify>`) would not build otherwise, since both files' local `fakeSource`/`countingSource` fixtures structurally implement `correlate.Source` and needed the new `Match(map[string][]string)` signature plus a `MatchVocabulary()` method. Neither file was in this plan's declared `files_modified` list.
- `plugins/proton`'s `fetch_rendition_test.go`, `mailbox_cache_test.go`, `item_test.go`, and `live_bridge_test.go` — four files beyond the plan's declared `imap_transcript_test.go` — all switched from `&toposv1.MatchRequest{Keywords: ...}` to a shared `foldersMatchReq` helper under Rule 3: every one of them builds a `MatchRequest` directly and lives in the same Go package as `plugin.go`, so `cd plugins/proton && go build ./...` would not compile with only one file updated.
- `plugins/signal`'s D-06 contract-change regression test (`TestMatch_ProfileNameOnlyAndNoteToSelfNeverMatch_SurvivesContractChange`) landed in `byte_identical_test.go`, not `message_test.go` as the plan's `files_modified` list named, under Rule 3: the test needs `buildFixtureDatabase`/`NewSourcePlugin`-style fixture infrastructure that only `byte_identical_test.go` has; `message_test.go` tests `parseMessage` in isolation with no `Match()`-level entry point.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] `kernel/syncer` test fixtures needed the new `correlate.Source` shape**
- **Found during:** Task 2's own `<verify>` (`go test ./kernel/...`)
- **Issue:** `coordinator_test.go`'s `fakeSource` and `scheduler_test.go`'s `countingSource` structurally implement `correlate.Source` with the old `Match(context.Context, []string)` signature and no `MatchVocabulary()` method
- **Fix:** Added `MatchVocabulary() []string` returning `["keywords"]` and changed `Match`'s parameter to `map[string][]string` on both fixtures
- **Files modified:** `kernel/syncer/coordinator_test.go`, `kernel/syncer/scheduler_test.go`
- **Verification:** `go test ./kernel/... -count=1` passes
- **Committed in:** `f65fcd6` (Task 2 commit)

**2. [Rule 3 - Blocking] Four undeclared proton test files built the old `MatchRequest` shape**
- **Found during:** `cd plugins/proton && go build ./...` while implementing Task 3
- **Issue:** `fetch_rendition_test.go`, `mailbox_cache_test.go`, `item_test.go`, `live_bridge_test.go` all construct `&toposv1.MatchRequest{Keywords: ...}` directly
- **Fix:** Introduced `foldersMatchReq(values []string) *toposv1.MatchRequest` in `imap_transcript_test.go` (the plan's declared file) and mechanically replaced every `MatchRequest{Keywords: ...}` literal across all five proton test files; removed one now-unused `toposv1` import (`item_test.go`)
- **Files modified:** `plugins/proton/fetch_rendition_test.go`, `mailbox_cache_test.go`, `item_test.go`, `live_bridge_test.go`
- **Verification:** `cd plugins/proton && go build ./... && go test ./... -count=1` passes
- **Committed in:** `edc541d` (Task 3 commit)

**3. [Rule 3 - Blocking] Signal's D-06 regression test needed fixture infrastructure not present in `message_test.go`**
- **Found during:** Implementing Task 3's explicit instruction to "add a regression test pinning that a profile name and a Note-to-Self conversation still never match"
- **Issue:** The plan's declared file for this test, `message_test.go`, tests `parseMessage` in isolation and has no `SourcePlugin`/database fixture; the fixture-database `Match()` entry point (`buildFixtureDatabase`, `NewSourcePlugin`, `fixtureKeyHex`) lives entirely in `byte_identical_test.go`
- **Fix:** Added `TestMatch_ProfileNameOnlyAndNoteToSelfNeverMatch_SurvivesContractChange` to `byte_identical_test.go`, building a dedicated fixture with a profile-name-only 1:1 conversation and a Note-to-Self conversation (carrying a nickname that WOULD be a valid candidate if it weren't Note-to-Self), asserting `Match` with both names as keywords returns zero items
- **Files modified:** `plugins/signal/byte_identical_test.go`
- **Verification:** `CGO_ENABLED=1 go test -tags libsqlcipher ./... -run 'Match|Conversation|NoteToSelf|Describe' -count=1 -v` — all pass, including the new test
- **Committed in:** `edc541d` (Task 3 commit)

---

**Total deviations:** 3 auto-fixed (all Rule 3 — blocking compile/task-completion dependencies)
**Impact on plan:** All three deviations were mechanical, same-behavior fixes required for the plan's own stated `<verify>`/`<action>` requirements to actually pass; no scope creep beyond what those requirements already implied.

## Issues Encountered

None beyond the three deviations above. The user's independent verification run (`make test`) hit the expected intermediate breakage between Task 2 and Task 3 (`plugins/paperless` failing on the retired `GetKeywords`) — this was the anticipated state of a tracer-then-expansion plan mid-execution, not a regression, and Task 3 resolved it.

## User Setup Required

None — no external service configuration required.

## Next Phase Readiness

- Plan 05-03 (explicit-block and allowlist match resolution, D-02/D-03) builds directly on `kernel/correlate.matchFieldsFor`, which this plan deliberately scoped to the D-01 fallback branch only, per its own doc comment.
- Plan 05-05 (docs/plugin-contract.md, config.example.toml) will need to republish the new `match_fields`/`match_vocabulary` shape and the `topos.v2` contract-generation string — not yet touched by this plan, which was intentionally scoped to code + tests only.
- Every plugin's `contract_version` now reports `"topos.v2"`; any external tooling or documentation that still checks for `"topos.v1"` will need updating in a later plan.

---
*Phase: 05-source-instances-per-type-matching*
*Completed: 2026-08-06*
