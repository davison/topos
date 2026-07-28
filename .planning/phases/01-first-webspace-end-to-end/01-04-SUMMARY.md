---
phase: 01-first-webspace-end-to-end
plan: 04
subsystem: testing
tags: [go-ast, contract-testing, httptest, sqlite, documentation, plugin-contract, http-api]

# Dependency graph
requires:
  - phase: 01-first-webspace-end-to-end (plan 01-01, 01-02)
    provides: "proto/webspaces/v1/plugin.proto (locked unary Fetch contract), kernel/httpapi routes and JSON envelope, kernel/index store, plugins/paperless reference implementation"
provides:
  - "sdk/contract_test.go — allowlist assertion pinning plugin.proto's RPC set to exactly Describe/Match/Fetch/Health (PLUG-02)"
  - "plugins/paperless/readonly_test.go — go/ast walk proving no file under plugins/ constructs a non-GET HTTP request"
  - "kernel/httpapi/contract_test.go — pins the agent-facing JSON envelope: schema_version, {source_type}:{source_id} ids, link.fidelity, six-key provenance, 200/[] vs 404 distinction, {error:{code,message}} envelope, byte-identical repeated stream calls (AGENT-02)"
  - "kernel/index/store_test.go — extended with the three-key total-order assertion (timestamp_unix DESC, secondary_timestamp_unix DESC, id ASC) the contract test depends on"
  - "docs/plugin-contract.md — published third-party-facing plugin contract"
  - "docs/api.md — published agent-facing kernel HTTP JSON contract"
  - "config.example.toml — fully-commented reference config with the real house-move/'house and home' example"
  - "README.md — clean-clone-to-browsable-webspace instructions"
affects: ["02 (SilverBullet plugin authored against docs/plugin-contract.md, PLUG-05)", "v1.x (agent consumption of docs/api.md, AGENT-10/11)"]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "RPC-set allowlist test (not a mutation-verb blacklist) as the mechanical enforcement of a contract-shape guarantee"
    - "go/ast-based read-only scan over a package tree, immune to comments/string literals that would defeat a grep-based check"
    - "httptest + temp-file SQLite index as the fixture strategy for HTTP contract tests — zero network dependency"
    - "one-method interface seam (ItemFetcher) introduced purely to make a concrete plugin-host dependency testable"

key-files:
  created:
    - sdk/contract_test.go
    - plugins/paperless/readonly_test.go
    - kernel/httpapi/contract_test.go
    - docs/plugin-contract.md
    - docs/api.md
    - README.md
  modified:
    - kernel/httpapi/item.go
    - kernel/httpapi/routes.go
    - kernel/httpapi/stream.go
    - kernel/index/store.go
    - kernel/index/store_test.go
    - kernel/item/item.go
    - config.example.toml

key-decisions:
  - "Documented Item.SyncedAtUnix as kernel-populated-at-read-time, never plugin-set — the sixth provenance key (synced_at_unix) the contract requires but sync-time plugin provenance never carried on its own"
  - "RPC allowlist test over blacklist: fails the build on ANY new RPC (mutating or not) until deliberately widened, rather than only catching creatively-named mutation verbs"
  - "docs/api.md documents content_unavailable and internal_error error codes in addition to the four the plan's own <interfaces> block named — both exist in the shipped kernel/httpapi code and omitting them would leave the published contract incomplete"
  - "README.md's config-validation instructions use XDG_CONFIG_HOME (the actual, only config-path mechanism cmd/webspaces/main.go implements) rather than a --config flag, since no such flag exists in the shipped binary"

patterns-established:
  - "Contract test failure messages name the requirement ID (PLUG-02, AGENT-02) and point at the plan, so a future contributor who trips the test understands the constraint instead of deleting it"
  - "Published docs are verified against shipped code before commit, not against the plan's own summary of it — every RPC/enum/field/route/error-code/config-key cross-checked file-by-file"

requirements-completed: [AGENT-02, PLUG-02, KERN-01]

coverage:
  - id: D1
    description: "plugin.proto's RPC set is mechanically pinned to an allowlist (Describe, Match, Fetch, Health) — adding a fifth RPC fails the build"
    requirement: "PLUG-02"
    verification:
      - kind: unit
        ref: "sdk/contract_test.go#TestContractRPCAllowlist"
        status: pass
    human_judgment: false
  - id: D2
    description: "A committed Go AST scan proves no file under plugins/ constructs a non-GET HTTP request"
    requirement: "PLUG-02"
    verification:
      - kind: unit
        ref: "plugins/paperless/readonly_test.go#TestPluginsIssueOnlyGetRequests"
        status: pass
    human_judgment: false
  - id: D3
    description: "Agent-facing HTTP envelope (schema_version, stable ids, link.fidelity, six-key provenance, empty-array-not-404, error envelope, byte-identical repeats) is pinned by a committed contract test"
    requirement: "AGENT-02"
    verification:
      - kind: unit
        ref: "kernel/httpapi/contract_test.go#TestContract_StreamEnvelope_IDsLinkAndProvenance"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/contract_test.go#TestContract_EmptyWebspaceReturns200EmptyArrayNotNull"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/contract_test.go#TestContract_StreamCalledTwiceIsByteIdentical"
        status: pass
    human_judgment: false
  - id: D4
    description: "StreamItems total ordering (timestamp_unix DESC, secondary_timestamp_unix DESC, id ASC) is asserted against fixtures that would misorder under SQLite's natural row order"
    verification:
      - kind: unit
        ref: "kernel/index/store_test.go#TestStreamItems_TotalOrderingWithTieBreak"
        status: pass
    human_judgment: false
  - id: D5
    description: "docs/plugin-contract.md and docs/api.md are published, and agree with the shipped code well enough that a reader with only the doc, the .proto, and the SDK module could write a second plugin, or consume the API without repo access"
    requirement: "AGENT-02"
    verification:
      - kind: other
        ref: "manual cross-check: every RPC/enum-value/field in docs/plugin-contract.md matched against proto/webspaces/v1/plugin.proto; every route/error-code/provenance-key in docs/api.md matched against kernel/httpapi/*.go WriteError call sites"
        status: pass
    human_judgment: true
    rationale: "Whether documentation is sufficient for a reader with no repo access is a judgment call the plan's own human-check step calls out explicitly — automated cross-checking confirms no factual drift, but not reader-sufficiency."
  - id: D6
    description: "config.example.toml is a fully-commented reference that runs as written, and README.md takes a clean clone to a browsable webspace"
    requirement: "KERN-01"
    verification:
      - kind: e2e
        ref: "scripts/e2e-smoke.sh (full run against live paperless-ngx, house-move: 35 items, passed)"
        status: pass
      - kind: integration
        ref: "config.example.toml copied into a temp XDG_CONFIG_HOME and run via ./bin/webspaces sync — house-move: 35 items, exit 0"
        status: pass
    human_judgment: true
    rationale: "The plan's own verify block requires a human to literally follow README.md top-to-bottom on this machine as though freshly cloned; automated checks (make build, e2e-smoke.sh, config sync) prove the commands work but not that the prose is unambiguous to a first-time reader."

duration: 47min
completed: 2026-07-28
status: complete
---

# Phase 01 Plan 04: Contract Enforcement and Publication Summary

**Two previously-convention-only guarantees (PLUG-02 read-only, AGENT-02 agent-facing envelope) are now enforced by committed Go tests, and both contracts they protect are published as docs/plugin-contract.md and docs/api.md, verified line-by-line against the shipped code.**

## Performance

- **Duration:** 47 min (01:20 to 02:07, spanning a session continuation — Task 1 was committed in a prior session, this session verified it and completed Task 2)
- **Started:** 2026-07-28T01:20:35+01:00
- **Completed:** 2026-07-28T02:07:24+01:00
- **Tasks:** 2 completed
- **Files modified:** 11 (7 in Task 1, 4 in Task 2)

## Accomplishments

- `sdk/contract_test.go` pins `plugin.proto`'s RPC set to an allowlist of exactly `Describe`, `Match`, `Fetch`, `Health` — any fifth RPC, mutating or not, now fails the build until deliberately allowlisted
- `plugins/paperless/readonly_test.go` walks the Go AST of every file under `plugins/` and fails on any non-GET HTTP request construction, immune to comment/string-literal evasion
- `kernel/httpapi/contract_test.go` pins the entire agent-facing envelope: `schema_version`, `{source_type}:{source_id}` id shape, `link.fidelity`, the six-key `provenance` object, the 200-with-empty-array-vs-404 distinction, the `{error:{code,message}}` envelope, and byte-identical repeated stream responses
- `kernel/index/store_test.go` extended with the full three-key ordering assertion the contract test depends on
- `docs/plugin-contract.md` published: read-only-by-construction guarantee, SDK usage, handshake, discovery/launch, `WEBSPACES_SOURCE_CONFIG`, all four RPCs' semantics, every `Item` field, both enums, logging rules — cross-checked field-for-field against the shipped `.proto`
- `docs/api.md` published: envelope convention, all five routes with real examples, stable-id scheme, ordering guarantee, six provenance keys, and the complete six-entry error-code list (including `content_unavailable` and `internal_error`, which exist in the shipped code but weren't named in the plan's own `<interfaces>` summary)
- `config.example.toml` rewritten as a fully-commented reference, every key cross-checked against `kernel/config/types.go` and `config.go`'s actual `Validate()`, using this deployment's real paperless-ngx tag `"house and home"` as the worked example
- `README.md` written: clean-clone-to-running-webspace instructions, verified end to end by actually running every documented command

## Task Commits

1. **Task 1: Contract conformance tests — enforce read-only and the agent-facing envelope** - `6f9c934` (test) — committed in a prior session
2. **Task 2: Publish the contracts — plugin docs, API docs, example config, README** - `1a1384b` (docs)

_Note: Task 1 was executed and committed in a prior session before this execution resumed; this session verified all of Task 1's acceptance criteria still hold (all three workspace modules' `go test ./...` and `go vet ./...` pass) before proceeding to Task 2._

## Files Created/Modified

- `sdk/contract_test.go` - RPC allowlist + enum zero-value assertions (PLUG-02)
- `plugins/paperless/readonly_test.go` - go/ast read-only scan over `plugins/`
- `kernel/httpapi/contract_test.go` - agent-facing envelope conformance tests (AGENT-02)
- `kernel/httpapi/item.go`, `kernel/httpapi/routes.go`, `kernel/httpapi/stream.go` - `ItemFetcher` test seam, `synced_at_unix` provenance wiring
- `kernel/index/store.go`, `kernel/index/store_test.go` - `SyncedAtUnix` support, extended ordering test
- `kernel/item/item.go` - `Item.SyncedAtUnix` field
- `docs/plugin-contract.md` - published third-party plugin contract
- `docs/api.md` - published agent-facing HTTP API contract
- `config.example.toml` - fully-commented reference config
- `README.md` - build/run instructions for the whole stack

## Decisions Made

- RPC allowlist (not a mutation-verb blacklist) chosen for `sdk/contract_test.go` so any addition — not just an obviously-named mutating RPC — fails the build until deliberate
- `Item.SyncedAtUnix` is populated by the kernel's index layer at read time, never by a plugin, and overwrites anything a plugin's own `provenance` map happens to set for that key — this is the sixth provenance key (`synced_at_unix`) the contract test and this plan's `<interfaces>` block require but the sync-time plugin-populated provenance never carried
- `docs/api.md`'s error-code table documents `content_unavailable` (404, item exists but this rendition variant doesn't) and `internal_error` (500) alongside the four codes the plan's own interface summary named, because both exist in the shipped `kernel/httpapi` code and a published contract that omits real error paths is incomplete
- README's config-validation guidance uses `XDG_CONFIG_HOME` rather than a `--config` flag, because `cmd/webspaces/main.go` implements no such flag — only the `XDG_CONFIG_HOME`/`~/.config/webspaces/config.toml` resolution documented in `configPath()`

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 2 - Missing critical] Documented two additional error codes not named in the plan's `<interfaces>` block**
- **Found during:** Task 2 (writing `docs/api.md`)
- **Issue:** The plan's own `<interfaces>` summary listed only four error codes (`webspace_not_found`, `item_not_found`, `source_unavailable`, `unsupported_rendition_type`), but `kernel/httpapi/item.go` also returns `content_unavailable` (404) and `internal_error` (500) on real code paths. A published contract missing real error paths would mislead an agent parsing `error.code`.
- **Fix:** Added both codes to `docs/api.md`'s error-code table with their HTTP status and meaning, cross-checked against every `WriteError` call site in `kernel/httpapi/*.go`.
- **Files modified:** docs/api.md
- **Verification:** `grep` of all `WriteError` call sites in `kernel/httpapi/` confirms all six codes are now documented; none are missing or extra.
- **Committed in:** 1a1384b (Task 2 commit)

**2. [Rule 1 - Bug] Corrected the plan's assumed `--config` CLI flag**
- **Found during:** Task 2 (verifying README.md's documented run steps)
- **Issue:** The plan's acceptance criteria suggested testing the example config via `./bin/webspaces sync --config <temp path>` "or the documented equivalent." `cmd/webspaces/main.go` implements no `--config` flag at all — only `XDG_CONFIG_HOME`/`~/.config/webspaces/config.toml` resolution via `configPath()`.
- **Fix:** README.md documents only the real mechanism (copy to `~/.config/webspaces/config.toml`); verified the example config is valid by copying it into a temp `XDG_CONFIG_HOME` and running `./bin/webspaces sync` against it directly (`house-move: 35 items`, exit 0) rather than a nonexistent flag.
- **Files modified:** none (README.md already documented the real mechanism correctly; this was a verification-method correction, not a doc fix)
- **Verification:** `XDG_CONFIG_HOME=<tmp> ./bin/webspaces sync` against a copy of `config.example.toml` succeeded (35 items synced)
- **Committed in:** N/A (verification-only; no file changes required)

---

**Total deviations:** 2 (1 missing-critical doc addition, 1 verification-method correction)
**Impact on plan:** Both are corrections that make the published contract match the shipped code more faithfully. No scope creep — no new features, no architectural changes.

## Issues Encountered

- A stray `webspaces-plugin-paperless` subprocess from an earlier verification run was left running (not port-bound, but present) between Bash calls; killed before re-running `make build`/smoke tests. No functional impact — the kernel's own `pluginhost.Host.Shutdown()` correctly kills subprocesses when the parent `webspaces serve`/`sync` process exits normally; the leftover was from a killed *parent* process during earlier interactive debugging, not a defect in the shipped code.
- `kernel/webui/build/.gitkeep` was deleted on disk by `make build` (SvelteKit's `adapter-static` output replaced the placeholder-holding directory). This file is not in this plan's `files_modified` list and the deletion is a pre-existing artifact-management characteristic of the build process, not something this plan's tasks touch — left as an uncommitted working-tree state, not staged into either task commit.

## User Setup Required

None - no external service configuration required beyond `PAPERLESS_URL`/`PAPERLESS_TOKEN`, which were already configured in `.env` from prior phase work.

## Next Phase Readiness

- Phase 1 (First Webspace, End to End) is now fully complete: all four plans (01-01 through 01-04) executed, both PLUG-02 and AGENT-02 are mechanically enforced rather than convention-only, and both third-party-facing contracts (`docs/plugin-contract.md`, `docs/api.md`) are published and verified against the shipped code.
- Phase 2 (SilverBullet, per ROADMAP.md) can author its plugin directly against `docs/plugin-contract.md` — PLUG-05 (validating the contract via a second, structurally different source) is the natural next proof point.
- No blockers introduced by this plan. Existing Phase 3-5 blockers (Proton Bridge LAN exposure, Signal keyring extraction, WhatsApp linking stability) are unchanged and out of this plan's scope.

---
*Phase: 01-first-webspace-end-to-end*
*Completed: 2026-07-28*
