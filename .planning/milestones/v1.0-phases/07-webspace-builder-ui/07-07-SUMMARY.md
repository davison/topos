---
phase: 07-webspace-builder-ui
plan: 07
subsystem: api
tags: [go, http, agent-permissions, config-hot-apply, ast-guard]

# Dependency graph
requires:
  - phase: 07-webspace-builder-ui
    provides: "07-01/07-02's config.Store hot-apply write path (Store.Save = validate-then-apply, D-06) and 07-02 Task 2's per-request cfgStore.Expanded() pattern already applied to WebspacesHandler/ItemHandler/SourceRefreshHandler"
  - phase: 07-webspace-builder-ui
    provides: "07-06's re-verification (07-VERIFICATION.md) that identified gaps[0] — the agent-route config-staleness gap this plan closes"
provides:
  - "All five /agent/v1 handlers (agentSourcesHandler, agentWebspacesHandler, agentStreamHandler, agentItemHandler, agentRenditionHandler) resolve the running config from cfgStore fresh, as the first statement of their own request closure — no boot-time snapshot survives anywhere in agent.go"
  - "MountAgentRoutes holds no config value at all and its doc comment states the live per-request read, correcting the stale boot-snapshot parity claim (07-REVIEW.md IN-01)"
  - "kernel/httpapi/agent_live_config_test.go: end-to-end regression coverage (revocation and grant, both directions) plus an AST guard (TestAgentGuard_EveryHandlerResolvesConfigPerRequest) that fails the build if a resolved-config parameter or a MountAgentRoutes-time resolution ever returns"
affects: [07-webspace-builder-ui, agent-integrations]

# Actuals (#2632)
actuals:
  tokens: 7400
  tasks: 2
  commits: 2

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Per-request config resolution as the closure's first statement (cfg := cfgStore.Expanded()) — now applied uniformly across every /api/* AND /agent/v1/* handler; no handler in kernel/httpapi holds a boot-time *config.Config anymore"
    - "AST-based structural guard test (go/ast + go/parser + go/token over the plain source filename) as a durable recurrence guard for a defect class a behavioural test alone cannot catch on a not-yet-written future handler"

key-files:
  created:
    - kernel/httpapi/agent_live_config_test.go
  modified:
    - kernel/httpapi/agent.go

key-decisions:
  - "grantedSources keeps its existing *config.Config parameter unchanged — it is a pure function over an already-resolved config, not a handler; threading a *config.Store into it would push the resolution point down into a helper where the 'once per request, at the top' discipline is invisible"
  - "The AST guard (Task 2) enumerates the handler set by exact name rather than sampling — a set-equality assertion fails the build for both an added AND a removed/renamed handler, forcing a deliberate extension of the guard rather than a silent gap"
  - "Verified the guard actually bites (not just compiles) by temporarily reverting agentSourcesHandler to a resolved-config parameter, confirming TestAgentGuard failed with its own messages, then restoring agent.go byte-for-byte before committing — git diff confirmed zero drift"

patterns-established:
  - "A future handler added to kernel/httpapi/agent.go with a resolved *config.Config parameter, or any Expanded() call inside MountAgentRoutes, fails go test ./kernel/httpapi/ via TestAgentGuard_EveryHandlerResolvesConfigPerRequest before it can ship"

requirements-completed: [KERN-08]

coverage:
  - id: D1
    description: "All four previously boot-snapshotted /agent/v1 handlers (sources, webspaces, item, content/thumbnail rendition) now resolve the running config per request; a source's agent.read revocation via Store.Save is enforced on the very next request against the SAME router, with no kernel restart"
    requirement: KERN-08
    verification:
      - kind: unit
        ref: "kernel/httpapi/agent_live_config_test.go#TestAgentLiveConfig_RevokedReadGrantTakesEffectWithoutRestart"
        status: pass
      - kind: unit
        ref: "kernel/httpapi/agent_live_config_test.go#TestAgentLiveConfig_NewlyGrantedSourceIsVisibleWithoutRestart"
        status: pass
    human_judgment: false
  - id: D2
    description: "MountAgentRoutes' stale boot-snapshot doc comment corrected (07-REVIEW.md IN-01) — no remaining occurrence of the 'boot-snapshot' token, and the comment now states the live per-request read plus why the agent namespace cannot tolerate a router-construction-time resolution"
    verification:
      - kind: other
        ref: "grep -n boot-snapshot kernel/httpapi/agent.go (no matches)"
        status: pass
    human_judgment: false
  - id: D3
    description: "AST invariant guard: the agent handler set is enumerated exactly, none takes a *config.Config parameter, grantedSources keeps its one *config.Config parameter, MountAgentRoutes resolves zero configs, and every handler closure resolves exactly one config as its first statement — proven to actually fail (not just compile) via a temporary, reverted mutation"
    verification:
      - kind: unit
        ref: "kernel/httpapi/agent_live_config_test.go#TestAgentGuard_EveryHandlerResolvesConfigPerRequest"
        status: pass
    human_judgment: false
  - id: D4
    description: "Live make dev walkthrough: revoking then re-granting a source's agent.read in the UI is reflected on /agent/v1/sources and /agent/v1/items/{id} on the next request, kernel process id unchanged throughout"
    verification: []
    human_judgment: true
    rationale: "Requires a live kernel process, a browser UI save action, and a second-terminal HTTP call cross-referenced against an unchanged process id — folded into the plan's own <human-check> for the pending end-of-phase make dev walkthrough (07-VERIFICATION.md's backstop item for this gap), not automatable in this execution environment"

# Metrics
duration: 22min
completed: 2026-08-08
status: complete
---

# Phase 07 Plan 07: Agent-route live-config fix (CR-01 closure) Summary

**Every `/agent/v1` handler now resolves the running config fresh, per request, from `cfgStore` — closing the live authorization-bypass window where a revoked `agent.read` grant stayed in force until the kernel process restarted.**

## Performance

- **Duration:** 22 min
- **Started:** 2026-08-08T16:11:00Z
- **Completed:** 2026-08-08T16:33:00Z
- **Tasks:** 2
- **Files modified:** 2 (1 modified, 1 created)

## Accomplishments

- `agentSourcesHandler`, `agentWebspacesHandler`, `agentItemHandler` and `agentRenditionHandler` now take `cfgStore *config.Store` and resolve `cfg := cfgStore.Expanded()` as the first statement of their returned closure — the identical treatment `agentStreamHandler` and every `/api/*` handler already had. `MountAgentRoutes` holds no config value at all.
- `MountAgentRoutes`' doc comment corrected (07-REVIEW.md IN-01): it now states the live per-request read every handler performs and explains why the agent namespace specifically cannot tolerate a router-construction-time snapshot (AGENT-01's default-deny model, D-06's save-applies-immediately promise).
- New `kernel/httpapi/agent_live_config_test.go` proves both directions end to end against the SAME already-constructed router: a grant revoked via `Store.Save` disappears from `/agent/v1/sources` and its item, content and thumbnail routes 404 with `item_not_found` (the same envelope a nonexistent id gets — no distinct code, no existence leak), and `/agent/v1/webspaces` recomputes `item_count` to 0; the inverse (grant added) becomes visible identically.
- An AST-based structural guard (`TestAgentGuard_EveryHandlerResolvesConfigPerRequest`) pins the fix as an invariant: it enumerates the five agent handlers by exact name, asserts none takes a resolved `*config.Config` parameter, asserts `grantedSources` keeps its one `*config.Config` parameter, asserts `MountAgentRoutes` resolves zero configs, and asserts each handler closure resolves the config exactly once, first. Verified to actually fail (not just pass by construction) via a temporary, git-diff-confirmed-clean revert-and-restore of `agentSourcesHandler`.

## Task Commits

1. **Task 1: Close CR-01 — every /agent/v1 handler reads the live config per request, proven end to end** - `e3e7843` (fix)
2. **Task 2: Pin the invariant so a boot-time config snapshot cannot return to agent.go** - `7db44f5` (test)

**Plan metadata:** pending (this commit)

## Files Created/Modified

- `kernel/httpapi/agent.go` - four handler signatures changed to `cfgStore *config.Store` with per-request `Expanded()` resolution; `MountAgentRoutes` simplified to hold no config; doc comment rewritten
- `kernel/httpapi/agent_live_config_test.go` - new file: `agentLiveConfigRouter` test helper, two end-to-end live-config regression tests, and the AST invariant guard

## Decisions Made

- `grantedSources` was deliberately left unchanged (still takes `*config.Config`) — it's a pure helper over an already-resolved config, not a handler; threading the store into it would hide the "resolve once, at the top" discipline inside a helper instead of keeping it visible at every handler call site.
- The AST guard enumerates the handler set by exact name (set equality) rather than checking a sampled subset, so both an added and a renamed/removed handler fail the build outright, forcing the guard itself to be extended deliberately.
- Confirmed the guard's bite is real, not incidental: reverted `agentSourcesHandler` to a resolved-config parameter, ran `go test ./kernel/httpapi/ -run TestAgentGuard -v`, observed four assertion failures with their own descriptive messages (not a compile error or an unrelated test failure), then restored `agent.go` from a pre-mutation copy and confirmed `git diff --stat kernel/httpapi/agent.go` was empty before committing.

## Deviations from Plan

None - plan executed exactly as written. All acceptance criteria for both tasks were verified directly (parameter types via grep, doc-comment token absence, `routes.go` byte-identity, `go.mod`/`go.sum` untouched, `docs/api.md`/`docs/plugin-contract.md`/`proto/` untouched) before each commit.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- 07-VERIFICATION.md's `gaps[0]` (agent-route config staleness, 07-REVIEW.md CR-01) is now closed: all `missing[]` items satisfied — `cfgStore` threaded into all four handlers with per-request-first resolution, an end-to-end regression test proving revocation/grant visibility with no restart, and the stale doc comment corrected.
- Sibling gap-closure plan 07-08 (edit-modal stale state, `gaps[1]`) is unaffected by this plan — no shared files (`kernel/httpapi/agent.go` and `kernel/httpapi/agent_live_config_test.go` are httpapi-only; 07-08 is web-only). `cd web && npm run check && npm run test` were run as this plan's own phase-level sanity check per its `<verification>` block: 0 errors (9 pre-existing unrelated warnings), 469/469 tests passing across 29 files — confirming no collision.
- The one remaining item this plan's own scope lists as not-yet-verified is the live `make dev` human walkthrough (07-VERIFICATION.md's backstop item for this gap) — folded into the plan's `<human-check>` for the pending end-of-phase UAT pass, not executed in this run.

---
*Phase: 07-webspace-builder-ui*
*Completed: 2026-08-08*
