---
phase: 05-source-instances-per-type-matching
verified: "2026-08-06T15:25:00Z"
status: passed
score: 4/4 must-haves verified
behavior_unverified: 0
overrides_applied: 0
process_note: >
human_verification:

  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]
  - [object Object]

---

# Phase 5: Source Instances & Per-Type Matching Verification Report

**Phase Goal:** Sources become named instances — the same plugin type can be configured multiple times under user-chosen display names — and each instance declares matching config appropriate to its plugin type, replacing the single shared keyword list.
**Verified:** 2026-08-06T15:25:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth (ROADMAP Success Criterion) | Status | Evidence |
|---|---|---|---|
| 1 | User can configure two instances of the same plugin type with distinct display names and see them as separate sources in the stream, source filter, and health UI | ✓ VERIFIED (mechanics) | `TestTwoInstancesOfOnePluginType_StayDistinct` (kernel/syncer, PASS), `TestAgent_TwoInstancesOfOnePluginType_UngrantedNeverLeaks` (kernel/httpapi, PASS), `sources.test.ts` "resolves exactly one of two instances sharing a plugin kind" (PASS), `staleness.test.ts` "marks only the unreachable instance stale when two instances share one source_type" (PASS), `config.example.toml` worked two-instance example (`home-email`/`work-email`, both `topos-plugin-proton`). Live: `GET /api/sources` on a freshly-built binary against the operator's migrated config returns 4 distinct, correctly-named, sorted instances. Visual/pixel confirmation in an actual browser is a separate open human-verification item (below). |
| 2 | Each source instance carries its own matching configuration, typed to its plugin (IMAP folders/labels, document tags, chat conversation/group names, wiki tags/pages), replacing the single shared per-webspace keyword list; all five existing sources migrate to the new shape | ✓ VERIFIED | All 5 in-repo plugins (`mock`, `paperless`, `silverbullet`, `proton`, `signal`) declare `matchVocabulary` and read `match_fields` — confirmed via `grep -c 'matchVocabulary'` on each `plugin.go` (≥1 each) and by reading `plugins/{paperless,silverbullet,proton,signal}/plugin.go`. `kernel/config/types.go` has `MatchBlock`/`Webspace{Keywords,Sources,Match}`; `kernel/correlate/correlate.go#matchFieldsFor` implements allowlist→block→fallback. All go workspace tests pass (`make test`, exit 0, 6 modules). Live migrated config declares `[webspaces.house-move.match.<instance>]` blocks for every real instance. |
| 3 | Source identity throughout the kernel — index rows, sync runs, agent grants, HTTP API, and UI display — is the named instance, never the bare plugin type; existing webspace data migrates or re-syncs cleanly with no orphaned rows | ✓ VERIFIED | `item.FromProto(source, sourceType, …)`, `item.ID(source, sourceID)` (`kernel/item/item.go`), `sync_runs.source`/`items.source` columns with `schemaVersion = 2`-gated rebuild (`kernel/index/schema.go`), `granted[it.Source]` grant filtering (`kernel/httpapi/agent.go`, `SourceTypesByName` fully removed), `streamItem.source`/`source_display_name` (`kernel/httpapi/stream.go`). De-allowlisted-instance row clearing proven by `TestSyncSource_DeallowlistedInstanceRowsCleared` (PASS). Live: I ran `./bin/topos sync` against the operator's real migrated config on a freshly built binary — paperless 37, proton 44, signal 110, silverbullet 16 items, identical counts to 05-05's own live verification, no orphaned-row or validation error. Live `GET /api/webspaces/house-move/stream` returns items with `id` in `{source}:{source_id}` form. |
| 4 | The contract change is published: `docs/plugin-contract.md`, `proto/topos/v1/` (ROADMAP text says `proto/webspaces/v1/` — a stale path in ROADMAP.md unrelated to this phase's execution; the actual module is `proto/topos/v1/`), `config.example.toml`, and the mock plugin all reflect per-instance match config, and the standing contract tests still pass | ✓ VERIFIED | `docs/plugin-contract.md` contains `match_vocabulary` (6×), `content_shape` (8×), and the corrected `{source}:{source_id}` id scheme. `docs/api.md` contains `unsupported_content_shape` (2×) and `{source}:{source_id}` (2×). `config.example.toml` contains `display_name`, `sources =`, `[webspaces.` … `.match.` blocks, and all four field names (`tags`, `pages`, `folders`, `conversations`). `TestContractRPCAllowlist`, `TestContractEnumsZeroValueUnspecified`, `TestContractDeclaresMatchVocabulary` all PASS (`sdk/contract_test.go`). `make test` exits 0 across all 6 workspace modules; `npm run test` (101/101) and `npm run check` (0 errors) both pass. |

**Score:** 4/4 truths verified (0 present-behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `kernel/item/item.go` | `Item.Source`, `ID(source, sourceID)`, `FromProto(source, sourceType, …)` | ✓ VERIFIED | Present, substantive, wired (called from `kernel/correlate/correlate.go:114`) |
| `kernel/index/schema.go` | `items.source`/`sync_runs.source` columns, `schemaVersion` | ✓ VERIFIED | `const schemaVersion = 2` present; rebuild path in `kernel/index/store.go` |
| `kernel/config/types.go` | `Source.DisplayName`, `MatchBlock`, `Webspace{Keywords,Sources,Match}` | ✓ VERIFIED | All present with `toml:` struct tags |
| `kernel/pluginhost/host.go` | instance-keyed `Fetch`, `Plugin.DisplayName`/`PluginDisplayName` | ✓ VERIFIED | `byInstance` lookup present, `SourceTypesByName` fully removed |
| `kernel/pluginhost/matchconfig.go` | `ValidateMatchConfig` | ✓ VERIFIED | Present, wired into `cmd/topos/main.go`'s `setup()` |
| `kernel/httpapi/stream.go` | `streamItem.source`/`source_display_name` | ✓ VERIFIED | Present, live-confirmed via curl (`GET /api/webspaces/house-move/stream`) |
| `kernel/httpapi/rendition.go` | `sanitizeAndWrapRendition`, `renditionPolicies`, `renditionBaseStyle` | ✓ VERIFIED | Present; live-confirmed via curl (`GET /api/items/{id}/content` returns kernel-composed `<!doctype html>` wrapper) |
| `proto/topos/v1/plugin.proto` | `StringList`, `MatchRequest.match_fields`, `DescribeResponse.match_vocabulary`, `ContentShape` enum | ✓ VERIFIED | All present via grep and `sdk/contract_test.go` |
| `docs/plugin-contract.md` | republished contract | ✓ VERIFIED | Contains `match_vocabulary`, `content_shape`, corrected id scheme |
| `config.example.toml` | new shape documentation | ✓ VERIFIED | All required keys present, structurally validated (`go test ./kernel/config/`) |
| `.planning/phases/.../migrated-config.toml` | operator's migrated config | ✓ VERIFIED | Present, no literal secrets, installed at `~/.config/topos/config.toml` with `.pre-phase05.bak` |

### Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `kernel/correlate/correlate.go` | `kernel/item/item.go` | `item.FromProto(src.Name(), src.SourceType(), …)` | ✓ WIRED | Confirmed at line 114 |
| `kernel/httpapi/agent.go` | `kernel/config/config.go` | `granted[it.Source]` via `grantedSources(cfg)` | ✓ WIRED | Confirmed 6 call sites |
| `web/src/lib/components/SourceFilterChips.svelte` | `web/src/lib/api.ts` | `source.name` filter key, `source.display_name` label | ✓ WIRED | Confirmed, plus `title=` truncation attribute present |
| `cmd/topos/main.go` | `kernel/pluginhost/matchconfig.go` | `ValidateMatchConfig(` in `setup()` | ✓ WIRED | Confirmed |
| `kernel/httpapi/item.go` / `agent.go` | `kernel/httpapi/rendition.go` | `sanitizeAndWrapRendition(` | ✓ WIRED | Confirmed, both files; live-confirmed via curl |

### Behavioral Spot-Checks (live, run this session)

| Behavior | Command | Result | Status |
|---|---|---|---|
| Two named tracer tests for instance isolation | `go test ./kernel/syncer/ -run TestTwoInstancesOfOnePluginType_StayDistinct -v` | PASS | ✓ PASS |
| Agent grant isolation between two same-type instances | `go test ./kernel/httpapi/ -run TestAgent_TwoInstancesOfOnePluginType_UngrantedNeverLeaks -v` | PASS | ✓ PASS |
| Deterministic vocabulary-error ordering | `go test ./kernel/pluginhost/ -run TestValidateMatchConfig -count=20` | PASS (20/20) | ✓ PASS |
| Full workspace build + test | `CGO_ENABLED=0 go build ./... && make test` | 0 exit, all 6 modules ok | ✓ PASS |
| Web unit tests | `npm --prefix web run test -- --run` | 101/101 passed | ✓ PASS |
| Web type/svelte-check | `npm --prefix web run check` | 0 errors, 1 unrelated pre-existing warning (SearchBox.svelte) | ✓ PASS |
| Live sync against operator's real migrated config, freshly built binary | `./bin/topos sync` (isolated copy of index + config) | paperless 37, proton 44, signal 110, silverbullet 16 — matches 05-05's own reported counts, no validation error | ✓ PASS |
| Live `GET /api/sources` against migrated config | curl, ephemeral serve on port 7799 | 4 distinct instances, sorted by id, each `reachable: true`, resolved `display_name` | ✓ PASS |
| Live stream item shape | curl `GET /api/webspaces/house-move/stream?limit=3` | `id` in `{source}:{source_id}` form, `source`/`source_type`/`source_display_name` all present | ✓ PASS |
| Live rendition wrap | curl `GET /api/items/{id}/content` | 200, kernel-composed `<!doctype html><style>...` document | ✓ PASS |
| Debt-marker scan (TBD/FIXME/XXX) across all phase-modified Go/TS/Svelte files | `grep -n -E "TBD\|FIXME\|XXX"` | 0 matches | ✓ PASS |
| Stub-pattern scan (TODO/HACK/PLACEHOLDER, not-implemented strings) | `grep -n -E "TODO\|HACK\|PLACEHOLDER"` etc. | 0 matches | ✓ PASS |
| No plugin references retired rendition helpers | `grep -rl 'themeStyle\|signalThemeStyle\|WrapDocument' plugins --include='*.go'` | 0 matches | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|---|---|---|---|---|
| KERN-06 | 05-01, 05-03, 05-05 | Sources are named instances, usable throughout UI/API | ✓ SATISFIED | Truths 1 and 3 above; REQUIREMENTS.md marks Complete |
| KERN-07 | 05-02, 05-03, 05-04, 05-05 | Matching config declared per source instance, typed to plugin | ✓ SATISFIED | Truths 2 and 4 above; REQUIREMENTS.md marks Complete |

No orphaned requirements: REQUIREMENTS.md maps only KERN-06 and KERN-07 to Phase 5, and both appear in at least one plan's `requirements:` frontmatter.

### Anti-Patterns Found

None. Debt-marker and stub-pattern scans across every Go/TS/Svelte file this phase's 5 plans modified returned zero matches. No hardcoded empty-data stubs, no orphaned wiring, no placeholder returns found in any of the files inspected.

### Live Operational Note (not a phase code defect, but worth flagging)

While testing, I found the user's currently-running `make dev` session (bound to `127.0.0.1:7777`, process started **before** this phase's code was committed today) now returns `{"error":{"code":"internal_error","message":"index: latest sync run per source: SQL logic error: no such column: source_type (1)"}}` from `GET /api/sources`. This is the pre-existing binary (built from pre-Phase-5 source, still running via `go run`) trying to query the OLD `sync_runs.source_type` column name against an index that has since been rebuilt to schema v2 (`sync_runs.source`) by later work this phase. This is **exactly** the scenario 05-05-SUMMARY.md's "User Setup Required" section already flagged: that live session needs a normal restart (`make dev` stop/re-run) to pick up the migrated config and current binary. A fresh build (`make build` + `./bin/topos sync`/`serve`, which I ran against an isolated copy of the same config/index) works correctly with no errors. Recommend restarting the live `make dev` session.

### Human Verification Required

5 items — see YAML frontmatter `human_verification` for full detail. Summary:

1. **Visual confirmation of two-instance chips/health UI** (WINDOWS.md open item #4) — the single most important outstanding item; substituted with curl/API checks in 05-05, never opened in a real browser.
2. **Rendition pixel parity** (email/markdown/chat) after the D-11 kernel-owned rendition move — CSS-rule carry-forward is test-proven, actual browser rendering is not.
3. **Interrupted schema-rebuild recoverability** — structural (single-transaction) guarantee, not exercised by a kill-mid-rebuild test.
4. **Stale-binary handshake fail-fast** — `ProtocolVersion` bump is structurally correct and contract-pinned, but no live stale-binary launch was exercised.
5. **Subprocess teardown on `ValidateMatchConfig` failure** — mirrors two pre-existing error paths by inspection, not exercised against a real launched subprocess.

Items 3-5 were explicitly self-flagged as `human_judgment: true` by the executing agent in 05-01/05-02/05-03's own coverage tables; items 1-2 are the still-open WINDOWS.md ledger entry and 05-04's own D6 flag.

### Gaps Summary

No blocking gaps. All four ROADMAP Success Criteria have strong, independently-reproduced evidence (I re-ran the named tests, the full 6-module `make test`, the web test/check suite, and a live sync+serve against the operator's real migrated config on a freshly built binary — all green, matching the numbers the SUMMARYs themselves reported). Both requirement IDs (KERN-06, KERN-07) are satisfied with no orphaned requirements. No debt markers, stub patterns, or broken wiring found in any phase-modified file.

The phase is held at `human_needed` rather than `passed` solely because of the outstanding visual/human-judgment items already self-identified by the executing agents across all five plans and tracked in `.planning/WINDOWS.md` (open item #4). None of these are evidence of incorrect behavior — they are simply checks that require a human eye (real browser rendering) or a live-fault-injection scenario (kill-mid-rebuild, stale-binary launch, subprocess-teardown-on-failure) that this session, like the execution sessions before it, could not perform.

---

_Verified: 2026-08-06T15:25:00Z_
_Verifier: Claude (gsd-verifier)_
