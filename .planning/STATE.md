---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: milestone
current_phase: 03
current_phase_name: email-in-the-webspace
status: executing
stopped_at: Completed 03-06-PLAN.md
last_updated: "2026-07-31T16:37:32.527Z"
last_activity: 2026-07-31
last_activity_desc: Phase 03 planning complete
progress:
  total_phases: 3
  completed_phases: 3
  total_plans: 19
  completed_plans: 18
  percent: 60
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-07-29)

**Core value:** Open one webspace and instantly see and grok all related information across every silo — without visiting each data store individually.
**Current focus:** Phase 03 — email-in-the-webspace

## Current Position

Phase: 03 (email-in-the-webspace) — EXECUTING
Plan: 2 of 6
Status: Ready to execute
Last activity: 2026-07-31 — Phase 03 planning complete

Progress: [██████████] 100%

## Performance Metrics

**Velocity:**

- Total plans completed: 12
- Average duration: —
- Total execution time: —

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 | 6 | - | - |
| 02 | 6 | - | - |

**Recent Trend:**

- Last 5 plans: —
- Trend: —

*Updated after each plan completion*
| Phase 01 P01 | 2h44m | 3 tasks | 90 files |
**Per-Plan Metrics:**

| Plan | Duration | Tasks | Files |
|------|----------|-------|-------|
| Phase 01 P02 | 40min | 2 tasks | 17 files |
| Phase 01 P03 | 25min | 2 tasks | 14 files |
| Phase 01 P04 | 47min | 2 tasks | 11 files |
| Phase 01 P05 | 4min | 3 tasks | 2 files |
| Phase 01 P06 | 15min | 3 tasks | 6 files |
| Phase 02 P01 | 68min | 3 tasks | 34 files |
| Phase 02 P02 | 35min | 3 tasks | 22 files |
| Phase 02 P03 | 55min | 3 tasks | 15 files |
| Phase 02 P04 | 50min | 3 tasks | 21 files |
| Phase 02 P05 | 5min | 2 tasks | 2 files |
| Phase 02 P06 | 15min | 2 tasks | 3 files |
| Phase 03 P05 | 21min | 3 tasks | 4 files |
| Phase 03 P06 | 20min | 2 tasks | 6 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- [Roadmap]: Vertical MVP slices — each phase adds one real source end to end; no mock-only phase. The kernel spine ships in Phase 1 behind paperless-ngx (real, low-risk) rather than behind a fixture plugin.
- [Roadmap]: Sources ordered by ascending integration risk — paperless-ngx → SilverBullet → IMAP → Signal → WhatsApp — so v1 is useful before the unpredictable sources are attempted.
- [Roadmap]: Agent permission model (AGENT-01) lands in Phase 2 while only two plugins exist, so later plugins declare grants rather than retrofitting.
- [Roadmap]: Full-text search (KERN-05) pairs with email (Phase 3), the first source to bring enough volume for scrolling to be insufficient.
- [Phase ?]: Task 1 checkpoint: locked plugin.proto v1 option-a (unary Fetch) over the plan's recommended streaming option-b
- [Phase ?]: shadcn-svelte's live CLI/registry retired baseColor slate and style new-york in favor of an encoded theme-preset system; components.json still records the plan's contract values and every actual color is hand-authored in src/app.css from UI-SPEC hex tokens
- [Phase ?]: lucide-svelte replaced with its upstream-recommended successor @lucide/svelte (deprecated package)
- [Phase ?]: Implemented plan 01-02's plugin.proto interfaces against the actual locked unary Fetch contract (01-01's D-Task1 decision), not the plan's own stale streaming-Fetch interfaces block
- [Phase ?]: Split paperless client.go's rendition fetch into named Preview/Thumbnail methods and added url.PathUnescape id decoding — both required to satisfy the plan's own acceptance criteria
- [Phase ?]: PLUG-03 sync-time validation rejects only the offending item (not the whole source batch) when fidelity is unspecified or deep_link is empty, recording the rejection in sync_runs.error
- [Phase ?]: [Phase 01-03]: Installed vitest as the frontend's first unit-test runner (no test infrastructure existed) to satisfy the plan's own npm run test acceptance criterion
- [Phase ?]: [Phase 01-03]: StreamList.svelte's sync-failure branch is checked and rendered strictly before the empty branch — a webspace whose sync failed and returned zero items must never render as 'Nothing here yet'
- [Phase ?]: [Phase 01-03]: Svelte 5 gotcha — a local variable literally named 'state' collides with the $state() rune's store auto-subscription parsing; renamed to loadState in the webspace route
- [Phase ?]: [Phase 01-04]: RPC allowlist (not blacklist) chosen for sdk/contract_test.go so any new RPC fails the build until deliberately allowlisted
- [Phase ?]: [Phase 01-04]: Item.SyncedAtUnix populated by kernel index at read time, never by a plugin — the sixth provenance key (synced_at_unix) the contract requires
- [Phase ?]: [Phase 01-04]: docs/api.md documents content_unavailable and internal_error error codes in addition to the four named in the plan's own interfaces block, matching the shipped kernel/httpapi code
- [Phase ?]: [Phase 01-05]: Restored SPA styling by importing app.css in the root layout (gap G-01-2) — single-line root cause fix, no other files touched
- [Phase ?]: [Phase 01-05]: e2e-smoke.sh hardened with a stale-listener pre-check and a three-part stylesheet assertion (link exists, fetches non-empty, contains #020617 token) as a recurrence guard
- [Phase ?]: [Phase 01-06]: Host predicate excludes port from comparison — the configured host is the user's own instance and a reverse proxy legitimately moves ports on it
- [Phase ?]: [Phase 01-06]: Same-host redirect test uses a distinct trailing-slash path, not the literal same path plus one more slash, because Go's net/url reference resolution collapses repeated slashes before the client's guard sees them
- [Phase ?]: [Phase 02-01]: Sync identity promoted from 'webspace' to '(webspace, source_type)' — ReplaceWebspaceSourceItems/SyncSource replace the whole-webspace write path outright so a healthy source's items are never discarded by a sibling source's failure
- [Phase ?]: [Phase 02-01]: Added an optional per-source ca_cert config field (not in the plan's original scope) to pin a self-signed CA for the user's real SilverBullet instance, discovered live during Task 1
- [Phase ?]: [Phase 02-01]: Fixed hardcoded 'paperless-ngx' UI copy (DetailPane failure alert, OpenInSource button) via a new sourceDisplayName() helper, reported live by the user during the tracer checkpoint
- [Phase ?]: [Phase 02-02]: kernel/syncer package name (not kernel/sync) — avoids aliasing against the standard library's own sync package, needed alongside golang.org/x/sync/singleflight
- [Phase ?]: [Phase 02-02]: correlate.Engine.SyncSource returns (results, rejections string) — the coordinator needs the aggregated rejection message to record on the sync_runs row it now owns
- [Phase ?]: [Phase 02-02]: GET /api/sources last_error is sourced exclusively from the kernel's own sync_runs history, never a plugin's self-reported HealthResponse fields (A-PLUG-04)
- [Phase ?]: [Phase 02-03]: RefreshResult TS shape follows the live kernel/httpapi/sources.go + docs/api.md exactly, not PLAN.md's interfaces sketch (field/wrapper-key names differ, no started_unix)
- [Phase ?]: [Phase 02-03]: WebspaceHeader moved from +layout.svelte into +page.svelte — a layout can't receive props back from the page it renders via {@render children()}, and the header's new props are all owned by page-level sources/filter state
- [Phase ?]: [Phase 02-03]: healthTone treats never-synced (last_status: '') as taking precedence over live reachability, per docs/api.md's 'render as neutral, never green ok' framing
- [Phase ?]: [Phase 02-04]: kernel/httpapi/agent.go stays in package httpapi (not a subpackage as 02-PATTERNS.md sketched) — a subpackage would need WriteJSON/WriteError/toStreamItem/etc from its parent while the parent mounts it, an import cycle
- [Phase ?]: [Phase 02-04]: SourcesHandler's merge logic factored into sourceStatusesFrom, reused unfiltered by /api/sources and filtered by /agent/v1/sources
- [Phase ?]: [Phase 02-04]: kernel/config.Validate's unconditional base_url/token requirement is NOT relaxed for plugins/mock's genuinely-configless case — logged as a deferred item for Phase 4/5 (Signal/WhatsApp) rather than fixed outside this plan's files_modified scope
- [Phase ?]: [Phase 02-04]: PLUG-05's isolation exercise (Task 3) was performed directly by this executor, not via a dispatched fresh subagent — no Task/subagent-dispatch tool was available in this execution environment, a materially weaker approximation than the plan's own already-flagged limitation, recorded honestly in 02-04-SUMMARY.md
- [Phase ?]: [Phase 02-05]: Fail-closed policy confirmed as-specified — Match propagates every non-ErrNotFound read error as codes.Unavailable, no partial-tolerance heuristic
- [Phase ?]: [Phase 02-05]: No kernel change needed — SyncSource/Coordinator already correctly skip persistence and record error status on a Match error; the SilverBullet plugin was the only broken link
- [Phase ?]: [Phase 02-06]: Deleted the seven --spacing-<key> theme entries outright rather than renaming/relocating them — zero utilities in web/src reference them
- [Phase ?]: [Phase 02-06]: assert-stylesheet.sh accepts either --container-<key> custom property or an inlined rem value for each named width, since Tailwind v4's @theme inline block inlines resolved values
- [Phase ?]: [Phase 03-05]: Zero-guard test's RED signal produced deliberately against a temporary unguarded implementation (TimestampUnix: m.internalDate.Unix(), no IsZero() check), not the original code — the original code's output (0, by field omission) is bit-identical to the correctly-guarded output (0, by design), so the assertion cannot distinguish them at the original-code state
- [Phase ?]: [Phase 03-06]: mergeMailboxCache is last-writer-wins per key (not insert-only) — a moved message is refreshed by whichever Match rediscovers it
- [Phase ?]: [Phase 03-06]: Added web/src/lib/node-builtins.d.ts (narrow ambient node:fs/node:path/node:url types) to satisfy svelte-check's 0 ERRORS without installing @types/node

### Pending Todos

[From .planning/todos/pending/ — ideas captured during sessions]

None yet.

### Blockers/Concerns

- Phase 3 (Email): Proton Bridge LAN exposure, self-signed cert handling in the Go IMAP client, and Proton webmail deep-link format are unverified — spike before planning.
- Phase 4 (Signal): Keyring backend extraction must be tested against the user's actual Arch/DE setup; schema-version detection required — spike before planning.
- Phase 5 (WhatsApp): Highest-risk area. No official API; linked-device route can be de-linked or banned. Spike must answer linking stability, backfill volume, event-stream persistence, and recovery before planning.
- Firewall/network access from the desktop to Proton Mail Bridge on the home server is not yet opened (bridge binds 127.0.0.1 by default).

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260729-p2n | create a wrapper script that exposes the env vars in .env to the webspaces binary | 2026-07-29 | 7becca1 | [260729-p2n-create-a-wrapper-script-that-exposes-the](./quick/260729-p2n-create-a-wrapper-script-that-exposes-the/) |

## Deferred Items

Items acknowledged and carried forward from previous milestone close:

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| *(none)* | | | |

## Session Continuity

Last session: 2026-07-31T15:51:31.575Z
Stopped at: Completed 03-06-PLAN.md
Resume file: None
