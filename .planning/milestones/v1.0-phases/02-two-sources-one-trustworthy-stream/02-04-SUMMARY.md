---
phase: 02-two-sources-one-trustworthy-stream
plan: 04
subsystem: api
tags: [authorization, default-deny, agent-api, go-plugin, contract-testing]

requires:
  - phase: 02-two-sources-one-trustworthy-stream
    provides: "02-02: kernel/syncer coordinator as the only sync entry point, pluginhost.Plugin.DisplayName/Host.ProbeSources, kernel/httpapi/sources.go owning /api/sources + refresh routes — this plan's agent routes reuse all three rather than reimplementing them"
provides:
  - "kernel/config.AgentGrant (Read/Handoff, default-deny by Go zero value) and Config.AgentReadGrantedNames()"
  - "kernel/pluginhost.Host.SourceTypesByName() — no-RPC config-name-to-source_type resolution"
  - "kernel/httpapi/agent.go: the /agent/v1 route namespace, a default-deny grant-filtered mirror of /api/*, mounted by Router alongside the human-facing routes"
  - "plugins/mock: a complete, network-free reference SourcePlugin built from proto/webspaces/v1/plugin.proto + docs/plugin-contract.md + the sdk module alone"
  - "docs/plugin-contract.md: a self-sufficiency completeness pass including a new 'Build your first plugin' walkthrough"
affects: [03-*, kernel/httpapi (any future route addition should follow the same narrow-interface pattern), plugins/* (any future third-party plugin build follows the same contract)]

tech-stack:
  added: []
  patterns:
    - "Structural grant filtering: a single grantedSourceTypes(cfg, byName) helper intersects config-declared agent.read grants with the launched-plugin name-to-source_type map; every /agent/v1 handler filters with it after the identical index read/plugin call its /api/* sibling makes, never before and never differently"
    - "No-existence-leak via identical error construction: an ungranted item and a genuinely nonexistent item share one code path (agentItemNotFound) producing byte-identical status/code/message, rather than a distinct 'forbidden'-shaped response"
    - "Reference-plugin-as-contract-proof: plugins/mock is committed as the PLUG-05 deliverable itself, not just an aid — a stranger reads it end to end as documentation, and the contract document is completeness-tested by building a second plugin from it in isolation"

key-files:
  created:
    - kernel/httpapi/agent.go
    - kernel/httpapi/agent_test.go
    - plugins/mock/{go.mod,main.go,plugin.go,plugin_test.go}
    - .planning/phases/02-two-sources-one-trustworthy-stream/deferred-items.md
  modified:
    - kernel/config/{types.go,config.go,config_test.go}
    - kernel/pluginhost/host.go
    - kernel/httpapi/{routes.go,sources.go,sources_test.go}
    - config.example.toml
    - docs/api.md
    - docs/plugin-contract.md
    - go.work
    - Makefile
    - .gitignore

key-decisions:
  - "kernel/httpapi/agent.go lives in package httpapi, not a kernel/httpapi/agent subpackage as 02-PATTERNS.md sketched — a subpackage would need WriteJSON, WriteError, toStreamItem, syncStatus, streamItem, itemContent, rendition, allowedRenditionTypes, writeFetchError and the Fetcher interface from its parent package while the parent mounts the subpackage's routes, which is an import cycle no split avoids without duplicating all of the above"
  - "SourcesHandler's merge logic factored into a new sourceStatusesFrom(ctx, store, prober) helper in sources.go, reused unfiltered by /api/sources and filtered by /agent/v1/sources — avoids two independently-maintained copies of the health/sync-history merge"
  - "agent.go's item/content/thumbnail rendition URLs point at /agent/v1/items/{id}/... (not /api/items/{id}/...) — a deliberate self-consistency choice beyond what the plan's interfaces block specified verbatim, since toStreamItem's thumbnail_url field (reused unmodified, per the plan's own instruction not to touch stream.go) still points at /api/items/.../thumbnail regardless of which namespace served the parent response; not a security concern since /api/* is grant-free by design (T-02-22) and reachable by the same already-local caller either way"
  - "agentGrantedItemCount(ctx, store, webspaceName, granted) reuses StreamItems and filters in Go rather than adding a new per-source-filtered Store query — kernel/index/store.go is not in this plan's files_modified list, and the webspace/agent item counts this plan needs are small enough that an extra index read per webspace per request is not a performance concern at this scale"
  - "config.Validate's unconditional base_url/token requirement for every [sources.<name>] entry (pre-existing, Phase 1) is NOT relaxed for the mock plugin's genuinely-configless case — logged as a deferred item (.planning/phases/02-two-sources-one-trustworthy-stream/deferred-items.md) rather than fixed, since kernel/config/config.go is outside this plan's Task 2 files_modified list and the right fix is a design decision to make when Signal/WhatsApp (Phase 4/5) actually need a configless source"
  - "plugins/mock's DeepLink literals use http://localhost/... rather than a fictitious external hostname (originally https://example.invalid/...) — internal/audit's TestNoForeignEgressOutsideSanctionedClient mechanically fails the build on any non-test, non-loopback absolute URL literal in shipped Go source; a real plugin never hits this because its deep_link is built at runtime from the operator's configured base_url, but the mock has no real base_url to build from"
  - "The PLUG-05 isolation exercise (Task 3) was performed directly by this executor rather than via a dispatched fresh subagent — no Task/subagent-dispatch tool was available in this execution environment. This is a materially weaker approximation than even the plan's own already-flagged A-PLUG-05 limitation (a fresh agent context sharing this project's general Go/gRPC knowledge), since this executor also retained full session memory of plugins/paperless and plugins/silverbullet from earlier in the same session while writing the trial plugin. Recorded honestly below, not papered over."

requirements-completed: [AGENT-01, PLUG-05]

coverage:
  - id: D1
    description: "Config grants agent read access and action hand-off separately per source, defaulting to deny by Go zero value — an absent [agent] block, an absent key, and an explicit false are all identically deny"
    requirement: AGENT-01
    verification:
      - kind: unit
        ref: "kernel/config/config_test.go#TestAgentReadGrantedNames_AbsentEmptyAndExplicitFalseAreAllDenied"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/agent_test.go#TestAgentSourcesHandler_AbsentEmptyAndExplicitFalseAllDeny"
        status: pass
    human_judgment: false
  - id: D2
    description: "read and handoff are independent booleans — a source granted handoff but not read is still absent from every agent listing and its items are still unreadable through the agent namespace"
    requirement: AGENT-01
    verification:
      - kind: unit
        ref: "kernel/config/config_test.go#TestAgentReadGrantedNames_HandoffWithoutReadIsNotReadGranted"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/agent_test.go#TestAgentSourcesHandler_HandoffWithoutReadIsStillFullyAbsent"
        status: pass
    human_judgment: false
  - id: D3
    description: "An ungranted source's item returns a response byte-identical (status, code, message) to a genuinely nonexistent item through /agent/v1/items/{id} and its /content child, so the namespace cannot be used to enumerate withheld sources"
    requirement: AGENT-01
    verification:
      - kind: unit
        ref: "kernel/httpapi/agent_test.go#TestAgentItemHandler_UngrantedItemMatchesNonexistentItemResponse, #TestAgentContentHandler_UngrantedItemMatchesNonexistentAndWritesNoBytes"
        status: pass
      - kind: integration
        ref: "live: curl against a real ungranted SilverBullet item id and a made-up id through /agent/v1/items/{id}, both 404 item_not_found with the id-specific message construction"
        status: pass
    human_judgment: false
  - id: D4
    description: "Every /api/* route's response is unchanged regardless of grant configuration, and zero-grant / known-webspace-zero-items cases return 200 with an empty array rather than an error or 404"
    requirement: AGENT-01
    verification:
      - kind: unit
        ref: "kernel/httpapi/agent_test.go#TestAgentAPIRoutesUnaffected, #TestAgentSourcesHandler_ZeroGrantsReturns200EmptyArray, #TestAgentStreamHandler_KnownWebspaceZeroGrantedItemsReturns200EmptyArray, #TestAgentStreamHandler_UnknownWebspace404, #TestAgentStreamHandler_OrderingMatchesHumanStreamWithUngrantedRemoved, #TestAgentWebspacesHandler_CountsRestrictedToGrantedSources"
        status: pass
    human_judgment: false
  - id: D5
    description: "A reference mock plugin exists as its own Go module with no network dependency, built entirely from the published proto, the contract document, and the sdk module, and the kernel launches and calls it successfully"
    requirement: PLUG-05
    verification:
      - kind: unit
        ref: "plugins/mock/plugin_test.go (9 tests: Describe identity, exact-case-insensitive Match, no-substring-matching, zero-match empty result, fidelity/deep_link validity, group presence, Fetch full/unknown-id, Health)"
        status: pass
      - kind: integration
        ref: "live: webspaces sync/serve against a scratch config with [sources.mock] and no other plugins, launched with WEBSPACES_SOURCE_CONFIG unset — 4 items synced, GET /agent/v1/sources, /agent/v1/webspaces/{ws}/stream, /agent/v1/items/{id} and /api/items/{id} all verified against the real kernel"
        status: pass
    human_judgment: false
  - id: D6
    description: "The published contract is self-sufficient — validated by building a second, independently-shaped plugin from it in isolation (proto + contract doc + sdk + plugins/mock only), with every gap found recorded and closed"
    requirement: PLUG-05
    verification: []
    human_judgment: true
    rationale: "PLUG-05's own pass criterion (02-04-PLAN.md's A-PLUG-05) explicitly frames this as a process validated by qualitative judgment, not a code assertion — the evidence is the gap list and the honestly-stated limits of the approximation, not a pass/fail test. See 'PLUG-05 validation exercise', below, for the full record a human reviewer should read to confirm the exercise was performed as described and its gap-closures are adequate."

duration: 50min
completed: 2026-07-29
status: complete
---

# Phase 2 Plan 4: Default-Deny Agent Grants and a Reference Mock Plugin Summary

**A new `/agent/v1` route namespace mirrors `/api/*` under per-source `[sources.<name>.agent]` grants that default-deny by Go zero value and make an ungranted source's items byte-identical to nonexistent ones, alongside `plugins/mock` — a network-free reference `SourcePlugin` that proves `docs/plugin-contract.md` is self-sufficient by being built and launched from it end to end.**

## Performance

- **Duration:** ~50 min (three tasks, plus live verification against the user's real paperless-ngx + SilverBullet instances and two scratch-config kernel launches)
- **Completed:** 2026-07-29
- **Tasks:** 3 completed
- **Files modified:** 21 (11 new, 10 modified), excluding `.planning/`

## Accomplishments

- `kernel/config.AgentGrant` (`Read`, `Handoff`) and `Source.Agent` — default-deny needs no special-case code: an absent `[sources.<name>.agent]` block, an absent key, and an explicit `read = false` all decode to the same Go zero value.
- `kernel/httpapi/agent.go` mounts six `/agent/v1/*` routes mirroring the human-facing `/api/*` surface, filtering structurally after every index read: ungranted sources are entirely absent from listings, ungranted items return the identical `item_not_found` response a nonexistent id returns (never a distinct code), and ordering is preserved with ungranted entries removed, never reordered.
- `kernel/pluginhost.Host.SourceTypesByName()` resolves the config-name-to-`source_type` mapping the grant filter needs, with no RPC — the already-cached launch-time `Describe` result, not a live probe.
- `plugins/mock` — a complete four-RPC `SourcePlugin` with a fixed, varied in-memory item set (grouped/ungrouped, all three `LinkFidelity` values), zero network dependency, and no required `WEBSPACES_SOURCE_CONFIG` keys — committed as the PLUG-05 deliverable itself.
- `docs/plugin-contract.md` completeness pass: a new "Build your first plugin" walkthrough, a required/optional column on the `Item` field table, a mock-sourced `Match` worked example, and the real-plugin references collapsed into one labelled aside.
- Ran the PLUG-05 isolation exercise (Task 3): built a second, independently-shaped plugin (`plugins/trial`, a two-item source exercising `Fetch`'s `THUMBNAIL` variant with real byte data) using only the four sanctioned inputs, launched it through the real kernel end to end, found and closed two documentation gaps, then discarded it.
- Verified live against the user's real deployment: with `[sources.paperless.agent] read = true` and no grant on SilverBullet, `/agent/v1/sources` listed only paperless, `/agent/v1/webspaces/house-move/stream` contained zero SilverBullet items, and a real SilverBullet item id through `/agent/v1/items/{id}` produced the identical 404 body a made-up id produces.

## Task Commits

1. **Task 1: Default-deny agent grants, enforced structurally in their own route namespace** - `3226c5a` (feat)
2. **Task 2: A reference plugin with no network, and a contract document that stands on its own** - `9f0ed10` (feat)
3. **Task 3: Prove the contract is self-sufficient by building against it in isolation** - `11d633c` (docs)

_Note: Tasks 1 and 2 carry `tdd="true"` and were executed as single atomic commits per the `type="auto"` commit protocol (real implementation + real tests + real `<verify>` together), consistent with how this phase's prior plans executed the same task type._

## Files Created/Modified

- `kernel/config/types.go` - `AgentGrant`, `Source.Agent`
- `kernel/config/config.go` - `AgentReadGrantedNames()`
- `kernel/config/config_test.go` - three-way equivalence, handoff-without-read, explicit-grant tests
- `kernel/pluginhost/host.go` - `Host.SourceTypesByName()`
- `kernel/httpapi/agent.go` - `MountAgentRoutes`, `grantedSourceTypes`, `filterRunsByGrant`, `agentSourcesHandler`, `agentWebspacesHandler`, `agentStreamHandler`, `agentItemHandler`, `agentRenditionHandler`, `agentItemNotFound`, `agentCapabilities`, `agentSourceStatus`, `agentSourcesResponse`
- `kernel/httpapi/agent_test.go` - 12 tests covering default-deny, grant independence, no-existence-leak, ordering, and `/api/*` non-interference
- `kernel/httpapi/routes.go` - `MountAgentRoutes` call, updated package/`Router` doc comments
- `kernel/httpapi/sources.go` - `HealthProber` gains `SourceTypesByName()`; `SourcesHandler` refactored onto shared `sourceStatusesFrom`
- `kernel/httpapi/sources_test.go` - `fakeProber.SourceTypesByName()`
- `config.example.toml` - `[sources.paperless.agent]`/`[sources.silverbullet.agent]` (asymmetric example) and commented-out `[sources.mock]`/`[webspaces.demo]`
- `docs/api.md` - revised opening paragraph, new "The `/agent/v1` namespace" section, revised error-code table, `AGENT-01` removed from "What is not here yet"
- `plugins/mock/{go.mod,main.go,plugin.go,plugin_test.go}` - the reference plugin and its 9-test suite
- `go.work`, `Makefile` - `plugins/mock` module registration and build/test targets
- `docs/plugin-contract.md` - self-sufficiency completeness pass, "Build your first plugin" walkthrough, two exercise-found gap closures
- `.gitignore` - `/plugins/mock/mock` stray-binary entry
- `.planning/phases/02-two-sources-one-trustworthy-stream/deferred-items.md` - the config.Validate base_url/token gap, logged not fixed

## Decisions Made

See `key-decisions` in the frontmatter above for the full list with rationale — summarized: `agent.go` stays in package `httpapi` (import-cycle avoidance), `SourcesHandler`'s merge logic is shared via `sourceStatusesFrom`, agent-namespace rendition URLs self-consistently stay under `/agent/v1/`, per-webspace granted item counts reuse `StreamItems` rather than adding a new Store query, the pre-existing `base_url`/`token` requirement is deferred rather than relaxed, the mock's deep links use `localhost` to satisfy the repo's outbound-egress guard, and the PLUG-05 exercise was performed directly by this executor (no subagent-dispatch tool available) — recorded as a materially weaker approximation than even the plan's own flagged limitation.

## PLUG-05 validation exercise

**Environment limitation, stated up front:** 02-04-PLAN.md's Task 3 action text calls for dispatching "a fresh general-purpose agent" with exactly four inputs and an instruction not to read `plugins/paperless`/`plugins/silverbullet` or search outside those paths. This execution environment provided no Task/subagent-dispatch tool, so no genuinely isolated fresh context was available. This executor performed the exercise directly instead — a weaker approximation than even the plan's own already-flagged A-PLUG-05 limitation, because this executor also retained full session memory of `plugins/paperless` and `plugins/silverbullet`'s real implementations from earlier tasks in this same session, which a fresh agent context would not have. The exercise below is still real (a second plugin was actually written, built, launched, and torn down using only the declared inputs as *written references*), but its "found zero gaps beyond these two" result should be read with that caveat — a genuinely fresh context, or a real external author, may find gaps this executor's prior exposure to the real plugins caused it to fill in unconsciously.

**Inputs given (self-imposed):** `proto/webspaces/v1/plugin.proto`, `docs/plugin-contract.md` (as left by Task 2), `sdk/`, and `plugins/mock/`. Built as `plugins/trial`, a temporary Go module under `plugins/` (added to `go.work` for the duration of the exercise), implementing a fictitious two-item "notes" source — deliberately different in shape from `plugins/mock` (it exercises `Fetch`'s `CONTENT_VARIANT_THUMBNAIL` with real byte data, which `plugins/mock` never returns, since `plugins/mock` always reports `available: false` for every variant).

**Pass 1 — gaps found: 2**

1. **Gap:** "Depending on the SDK"'s `goplugin.Serve` (then `plugin.Serve`) code example showed no import for the `plugin`/`goplugin` package itself, only for `sdk`. Go's standard library has an unrelated `plugin` package (`plugin.Open`, for loading `.so` shared objects) sharing the bare name `plugin` — a reader following the example literally could type `import "plugin"` and get the wrong package entirely, one with no `ServeConfig`/`Serve` shape resembling the contract's. **Closed:** added the explicit `goplugin "github.com/hashicorp/go-plugin"` import alias to the code example, matching what every real plugin's `main.go` (including `plugins/mock/main.go`) actually uses.
2. **Gap:** No literal `[sources.<name>]`/`[webspaces.<name>]` TOML example existed within the four sanctioned inputs — only prose description of the config file's role, plus a JSON example of `WEBSPACES_SOURCE_CONFIG`'s *contents*. `config.example.toml` (referenced by "Build your first plugin" step 6) is a real repository file but not one of the four sanctioned inputs; a genuinely isolated builder attempting the full round-trip PLUG-05 requires ("the kernel launches and calls it successfully") would need to configure the kernel to launch their plugin, and the TOML dotted-table syntax for doing that was not spelled out anywhere in-scope. **Closed:** added a minimal, self-contained `[sources.yourplugin]`/`[webspaces.demo]` TOML example directly to "Build your first plugin" step 6.

**Pass 2 (re-run against the revised document) — gaps found: 0.** `plugins/trial` built cleanly (`CGO_ENABLED=0 go build ./...`), and was launched, synced (2 items matching keyword `errands`), and served live through the real kernel (`webspaces sync` then `webspaces serve` against a scratch `XDG_CONFIG_HOME` config) with no further reference to any input outside the four sanctioned ones. `GET /api/items/trial:note-1` and `GET /api/items/trial:note-1/thumbnail` (a real `image/png`-labeled byte rendition, distinct from `plugins/mock`'s always-unavailable `Fetch`) both returned correctly.

**Disposal:** `plugins/trial` was deleted in full (`rm -rf plugins/trial`), its `go.work` entry reverted (`git diff go.work` shows no residual change after the exercise), and its temporary binary (`bin/plugins/webspaces-plugin-trial`) removed — confirmed via `git status --short` showing no trial-related paths before the Task 3 commit. `go.work` today lists exactly `.`, `./sdk`, `./plugins/paperless`, `./plugins/silverbullet`, `./plugins/mock` — no fifth module.

**Stated limits of this approximation (honest, not glossed over):**
- No isolated subagent was actually dispatched — see the environment-limitation note above.
- This executor's own session included reading `plugins/paperless/{main.go,plugin.go}` and `plugins/silverbullet`'s summary earlier in this same run (for Task 1/2 context), which a genuinely external third-party author would never have. Any gap this prior exposure caused the executor to unconsciously fill in — by knowing the "shape" of a real plugin even while writing `plugins/trial` from the contract doc — would not be caught by this exercise.
- `plugins/trial` was still built by the same underlying model/knowledge base that wrote `plugins/mock` and much of `docs/plugin-contract.md` itself in this same session — the two passes are not independent authors, only sequential passes with a stated (imperfect) discipline about which files were "allowed."
- A clean second pass is evidence of self-sufficiency, consistent with 02-04-PLAN.md's own framing (A-PLUG-05) — not proof of it, and less strong evidence here than the plan anticipated, given the environment limitation above.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `plugins/mock`'s hardcoded `DeepLink` literals tripped the repo's outbound-egress guard**
- **Found during:** Task 2, running the full `go test ./... -race` verification step after committing the mock plugin's initial implementation
- **Issue:** `internal/audit.TestNoForeignEgressOutsideSanctionedClient` mechanically fails the build on any non-test, non-loopback absolute URL literal in shipped Go source (PROJECT.md's no-foreign-egress guarantee). `plugins/mock/plugin.go`'s four fixed `DeepLink` values (`https://example.invalid/mock/...`) tripped this — a genuine build-breaking regression the plan's own action text didn't anticipate (it specified "a non-empty deep link" without addressing the literal's host).
- **Fix:** Changed all four `DeepLink` values to `http://localhost/mock/...` (the only literal host this guard structurally accepts besides loopback IPs), with a code comment explaining why and noting a real plugin's deep link is built at runtime from a configured `base_url`, never a hardcoded literal, so this situation is specific to the mock's lack of a real per-instance URL.
- **Files modified:** `plugins/mock/plugin.go`
- **Verification:** `go test ./... -race` passes at the repo root (`internal/audit` included); `plugins/mock`'s own 9-test suite unaffected (no test asserted the literal host).
- **Committed in:** `9f0ed10`

---

**Total deviations:** 1 auto-fixed (Rule 1)
**Impact on plan:** A necessary correctness fix caught by the plan's own mandated `go test ./... -race` verify step — no scope creep, no file outside the plan's own `files_modified` list touched by this fix.

## Issues Encountered

None beyond the one deviation above and the config.Validate gap logged as a deferred item (not an "issue" requiring in-plan resolution — see `deferred-items.md`).

## User Setup Required

None. This plan added no new required user-facing configuration — every `[sources.<name>.agent]` grant and the commented-out `[sources.mock]` block are opt-in additions to the existing `~/.config/webspaces/config.toml`, which was temporarily and reversibly modified during live verification (grant added, then restored to its exact prior contents — confirmed via a pre-change backup and post-restore diff) and left with zero grants configured, matching its state before this plan ran.

## Next Phase Readiness

- `/agent/v1`'s six routes and the `[sources.<name>.agent]` grant shape are the permission model every later-phase plugin (IMAP, Signal, WhatsApp) inherits by simply being a configured source — no per-plugin grant code needed.
- `plugins/mock` and the completed `docs/plugin-contract.md` are the reference point for any future third-party plugin author, including a real one outside this project.
- The `.planning/phases/02-two-sources-one-trustworthy-stream/deferred-items.md` entry (kernel-level `base_url`/`token` requirement) should be revisited when Phase 4 (Signal) or Phase 5 (WhatsApp) is planned — both are genuinely configless local-database sources that will hit this exact gap for real, not just as a documentation caveat.
- No blockers identified for Phase 3 (email/IMAP).

---
*Phase: 02-two-sources-one-trustworthy-stream*
*Completed: 2026-07-29*

## Self-Check: PASSED

All 21 claimed files exist on disk and all four commits (`3226c5a`, `9f0ed10`, `11d633c`, `bde4f7f`) are present in `git log`.
