---
phase: 02-two-sources-one-trustworthy-stream
verified: 2026-07-29T00:08:13Z
status: gaps_found
score: 5/6 must-haves verified
behavior_unverified: 0
overrides_applied: 0
gaps:
  - truth: "A SilverBullet sync that experiences a genuine outage or auth failure mid-Match (after ListFiles succeeds) is recorded as an error, not silently reported as a successful sync with zero items — and does not silently delete every previously-synced SilverBullet item for every webspace"
    status: failed
    reason: "plugins/silverbullet/plugin.go Match() (lines 94-120) swallows every ReadFile failure identically to 'page not found' — including a dropped connection, revoked token, TLS failure, or the instance going down entirely partway through a sync. The goroutine always returns nil, so g.Wait() can never observe an error (the codes.Unavailable branch immediately below it is dead code). This directly contradicts docs/plugin-contract.md's own stated Match contract ('return codes.Unavailable ... not a partial, silently-empty result ... when the source system cannot be reached'). Concretely: if every page read fails after a successful listing, Match returns a successful, empty MatchResponse; kernel/correlate.SyncSource then calls Store.ReplaceWebspaceSourceItems with an empty slice, which deletes every previously-synced SilverBullet row for every webspace and records the run as 'ok' with item_count 0 — the exact silent, unrecorded-failure mode the rest of this phase (source-scoped persistence, two-phase sync_runs, aggregate sync.status, the UI-05 prohibition against presenting stale/partial content as complete/current) was built to prevent, reintroduced one layer down inside this specific plugin. Confirmed still present and unfixed in the current tree (no commit after 02-REVIEW.md's aacda62 touches plugins/silverbullet/plugin.go); confirmed no test in plugins/silverbullet/*_test.go exercises Match's failure-aggregation path (no TestMatch* function exists at all)."
    artifacts:
      - path: "plugins/silverbullet/plugin.go"
        issue: "Match's per-page errgroup.Go closure (lines 98-116) returns nil on every ReadFile error instead of distinguishing errors.Is(err, ErrNotFound) (safe to skip) from any other error (must fail the whole Match with codes.Unavailable)"
    missing:
      - "Propagate non-ErrNotFound ReadFile failures out of the errgroup so g.Wait() returns a non-nil error and Match returns codes.Unavailable instead of an empty successful MatchResponse"
      - "A regression test that makes every page read fail with a non-404 error after a successful listing and asserts Match returns codes.Unavailable rather than an empty, successful result"
---

# Phase 2: Two Sources, One Trustworthy Stream Verification Report

**Phase Goal:** User sees two structurally different sources interleaved in one webspace and can filter, refresh, and tell at a glance whether what they are looking at is complete and current

**Verified:** 2026-07-29T00:08:13Z
**Status:** gaps_found
**Re-verification:** No — initial verification

**MVP mode note:** ROADMAP.md marks this phase `Mode: mvp`, but the ROADMAP goal line is not in canonical "As a / I want to / so that" form and all four plans explicitly declined to invent one (documented in each PLAN.md as an "MVP-mode note"), deferring the user-story framing to a future `/gsd-mvp-phase 2` re-plan. Per the MVP verification guard, this report does not force the User Flow Coverage structure onto a non-canonical goal string — it applies standard goal-backward verification against the ROADMAP Success Criteria instead, which is the authoritative contract regardless of mode.

## Goal Achievement

### Observable Truths

Primary rows are the five ROADMAP Success Criteria (authoritative contract, Step 2a). Row 6 is a truth derived from the phase's own core value proposition ("tell at a glance whether what they are looking at is complete and current") and from `02-01-PLAN.md`'s must-have ("a healthy source is never rolled back") — verified directly against the current codebase and a live code-review finding (CR-01), not merely restated from SUMMARY.md.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SilverBullet pages matching the webspace keyword by tag or page name appear interleaved with paperless documents in one chronological stream, each with an exact deep link to its page | VERIFIED | Live `GET /api/webspaces/house-move/stream` against the user's real deployment returned 52 items (35 paperless + 17 silverbullet), sorted strictly descending by `timestamp_unix` across both source types in one list; a sampled SilverBullet item carried `fidelity: "exact"` and `link: "https://notes.davisononline.home/Aston issues 26_12_22"`. `MatchesKeyword`/`isPage` (plugins/silverbullet/frontmatter.go:70-108) implement exact-only, case-insensitive tag/name matching with leading-underscore exclusion exactly as D-03 specifies. |
| 2 | User can filter the stream down to a single source and back to all sources | VERIFIED | `web/src/routes/w/[webspace]/+page.svelte` derives `selectedSource` from `page.url.searchParams` via `resolveSourceFilter` and writes it back with `goto(..., {replaceState:true})`; `SourceFilterChips.svelte` exists using `variant="default"`/`variant="outline"`. 44/44 vitest tests pass including `resolveSourceFilter`, `filterItemsBySource`, `streamVariant`'s unrecognised-value-falls-back-to-all case. Visual click-through not exercised in a browser this session (see Human Verification). |
| 3 | User can see per-source health in the UI — reachable, last sync time, last error — and trigger a manual refresh that brings in new items without duplicating existing ones or stacking concurrent syncs | VERIFIED | Live `GET /api/sources` returned both sources with correct `display_name`, `reachable:true`, `last_status:"ok"`. Live `POST /api/sources/silverbullet/refresh` issued twice in a row both returned `item_count: 17` (no duplication). `kernel/syncer/coordinator.go` wraps every sync in `singleflight.Group.Do`; `go test ./kernel/syncer/... -race` passes. `SourceHealthChip.svelte` contains `RefreshCw`/`Tooltip`; `WebspaceHeader.svelte` contains `SourceHealthChip` and "Refresh all". |
| 4 | An item whose source is unreachable or has been deleted shows an explicit unavailable/stale state in the stream and detail pane, never a blank pane or a silent 404 | VERIFIED (backend + logic layer); UI pixels not visually exercised | `detailPaneState` (web/src/lib/format.ts:166) is unit-tested for its full four-way matrix including deleted-over-unreachable precedence (`staleness.test.ts`); `StreamRow.svelte` carries a `stale` prop with no accent/primary class; the cached-preview-over-alert pattern keeps title/date/labels/open-in-source rendered in every failure branch (grep confirms no blank-pane branch). Not exercised against a deliberately-stopped source in a real browser this session (see Human Verification). |
| 5 | Config grants agent read access and action hand-off separately, per plugin, defaulting to deny — a plugin with no grant is absent from agent-facing API responses | VERIFIED | `kernel/config.AgentGrant{Read,Handoff}` decodes to Go zero-value false when absent (kernel/config/types.go:78). Live `GET /agent/v1/sources` with zero grants configured returned `{"sources":[]}`. `agentItemNotFound` (kernel/httpapi/agent.go:245-246) constructs the identical `item_not_found` code/status/message `kernel/httpapi/item.go` uses for a genuinely nonexistent id (byte-for-byte identical strings confirmed by grep). `go test ./kernel/httpapi/... -run Agent` passes (part of the full `go test ./...` run). |
| 6 | A SilverBullet sync that experiences a genuine outage or auth failure mid-Match is recorded as an error, not silently reported as a successful sync with zero items, and does not silently delete previously-synced items — the "tell at a glance whether complete and current" guarantee holds even when the failure happens *inside* a plugin's own Match call, not just when a sibling source is offline | ✗ FAILED | See Gaps below (CR-01, confirmed still present at `plugins/silverbullet/plugin.go:94-120`; no fix committed after `02-REVIEW.md`; no regression test exists). |

**Score:** 5/6 truths verified (1 present-and-tested-elsewhere-but-behaviorally-broken failure)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `plugins/silverbullet/client.go` | Host-pinned, GET-only `/.fs` client | VERIFIED | 295 lines; `allowHost` present (7 refs); TLS `ca_cert` pinning added live (deviation, documented) |
| `plugins/silverbullet/plugin.go` | Describe/Match/Fetch/Health | VERIFIED (with the CR-01 defect noted above) | 251 lines; `LINK_FIDELITY_EXACT` present |
| `plugins/silverbullet/frontmatter.go` | Tag/name matching, snippet | VERIFIED | 108 lines; exact D-03 matching confirmed by code read |
| `plugins/silverbullet/render.go` | goldmark + bluemonday sanitize | VERIFIED | 105 lines; `bluemonday` present (5 refs); `WrapDocument` theme-injection added live (deviation, documented, applied strictly post-sanitization) |
| `kernel/index/store.go` | Source-scoped persistence | VERIFIED | `ReplaceWebspaceSourceItems` defined (line 95); `ReplaceWebspaceItems` (whole-webspace write) confirmed removed entirely, not kept as a compatibility shim |
| `kernel/correlate/correlate.go` | Source-major sync loop | VERIFIED | `SyncSource` per source (line 74); non-test importer set of `kernel/correlate` is exactly `cmd/webspaces/main.go`, `kernel/correlate/correlate.go`, `kernel/syncer/coordinator.go` — matches the acceptance criterion exactly |
| `plugins/silverbullet/outbound_hosts_test.go` | Host-pinning proof | VERIFIED | 140 lines; `ErrForeignHost` present |
| `kernel/syncer/coordinator.go` | Single-flight coordinator | VERIFIED | `singleflight.Group` field + `Do` call confirmed; `-race` test suite passes |
| `kernel/syncer/scheduler.go` | Ticker-driven scheduler | VERIFIED | `time.NewTicker`/`ctx.Done()` confirmed |
| `kernel/httpapi/sources.go` | Health merge + refresh routes | VERIFIED | Live `GET /api/sources`/`POST /.../refresh`/`POST /api/sync` all behave per spec |
| `kernel/pluginhost/host.go` | `ProbeSources` | VERIFIED | Live probe confirmed reachable status for both real sources |
| `web/src/lib/components/SourceHealthChip.svelte` | Status dot + tooltip + refresh | VERIFIED | Exists, 40+ lines, contains `RefreshCw`/`Tooltip` |
| `web/src/lib/components/SourceFilterChips.svelte` | All + per-source filter row | VERIFIED | Exists, contains both button variants |
| `web/src/lib/api.ts` | Typed sources/refresh calls | VERIFIED | Contains `/api/sources`, `refreshSource`, `refreshAll`, `postJSON` |
| `web/src/lib/components/staleness.test.ts` | Stale/unavailable assertions | VERIFIED | `content_unavailable`/`detailPaneState` matrix present and passing |
| `kernel/httpapi/agent.go` | `/agent/v1` grant-filtered mirror | VERIFIED | 80+ lines; `agent/v1` present; structural filtering confirmed by code read + live curl |
| `kernel/config/types.go` | `AgentGrant` default-deny | VERIFIED | `AgentGrant` struct present with `toml:"read"`/`toml:"handoff"` |
| `plugins/mock/plugin.go` | Reference SourcePlugin, no network | VERIFIED | 272 lines; `contractVersion` present; `make build` produces `bin/plugins/webspaces-plugin-mock` |
| `docs/plugin-contract.md` | Self-sufficient contract doc | VERIFIED (with caveats, see Human Verification) | Contains "Build your first plugin"; PLUG-05 exercise documented with 2 gaps found and closed on pass 1, 0 on pass 2 |
| `kernel/httpapi/agent_test.go` | Default-deny/no-leak assertions | VERIFIED | `item_not_found` present; part of the passing `go test ./...` run |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `kernel/correlate/correlate.go` | `kernel/index/store.go` | source-scoped replace | VERIFIED | `ReplaceWebspaceSourceItems` called at correlate.go:104 |
| `plugins/silverbullet/plugin.go` | `plugins/silverbullet/render.go` | sanitized HTML in Fetch | VERIFIED | `RenderSanitized`/`WrapDocument` called in `fetchFull` (plugin.go:214-224) |
| `web/src/lib/components/DetailPane.svelte` | `kernel/httpapi/item.go` | sandboxed iframe rendition | VERIFIED | `contentUrl(item.id)` used in both `text/html` and PDF iframe branches |
| `kernel/syncer/scheduler.go` | `kernel/syncer/coordinator.go` | ticker calls Coordinator only | VERIFIED | Confirmed by the non-test-importer-set assertion above |
| `kernel/httpapi/sources.go` | `kernel/syncer/coordinator.go` | refresh routes use same coordinator | VERIFIED | Live: scheduler's startup sync and manual refresh both reflected the same `sync_runs` state (no duplication, coalescing intact) |
| `kernel/httpapi/sources.go` | `kernel/pluginhost/host.go` | live reachability via `ProbeSources` | VERIFIED | Live `GET /api/sources` reachability matches real instance state |
| `kernel/httpapi/stream.go` | `kernel/httpapi/sources.go` | aggregate sync status | VERIFIED | Live stream response's `sync` object present and correctly `ok` with both sources healthy |
| `web/src/lib/components/WebspaceHeader.svelte` | `web/src/lib/api.ts` | chips + filter from one `getSources` response | VERIFIED | `getSources` referenced in `+page.svelte`, threaded into `WebspaceHeader` props |
| `web/src/routes/w/[webspace]/+page.svelte` | `web/src/lib/components/SourceFilterChips.svelte` | URL query state | VERIFIED | `searchParams`/`replaceState: true` confirmed |
| `web/src/lib/components/DetailPane.svelte` | `web/src/lib/api.ts` | unreachable/unavailable branches | VERIFIED | `detailPaneState` consumes `content`, `fetchErrorCode`, `sourceReachable` |
| `kernel/httpapi/agent.go` | `kernel/config/types.go` | granted source types from agent read grant | VERIFIED | `grantedSourceTypes` intersects `AgentReadGrantedNames()` with launched-plugin map |
| `kernel/httpapi/agent.go` | `kernel/pluginhost/host.go` | `SourceTypesByName` (no RPC) | VERIFIED | Referenced in `grantedSourceTypes` call site |
| `kernel/httpapi/routes.go` | `kernel/httpapi/agent.go` | `/agent/v1` mounted alongside human API | VERIFIED | `MountAgentRoutes` called before `NotFound` fallback |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Interleaved cross-source stream, sorted descending by timestamp | Live `GET /api/webspaces/house-move/stream` against real paperless+SilverBullet | 52 items (35+17), strictly descending timestamp order, mixed source_types | PASS |
| No duplication on repeated manual refresh | Live `POST /api/sources/silverbullet/refresh` x2 | Both returned `item_count: 17`; stream item count unchanged (52) at 52 after both | PASS |
| Unconfigured refresh target returns structured 404 | Live `POST /api/sources/does-not-exist/refresh` | `404 {"code":"source_not_found",...}` | PASS |
| Zero-grant agent namespace returns empty array, not error | Live `GET /agent/v1/sources` (config has no `[sources.*.agent]` blocks) | `200 {"sources":[]}` | PASS |
| Non-test importer set of `kernel/correlate` is exactly the coordinator + CLI + itself | `grep -rl` shell assertion from 02-02-PLAN.md's own verify step | `./cmd/webspaces/main.go ./kernel/correlate/correlate.go ./kernel/syncer/coordinator.go` | PASS |
| SilverBullet Match failure-mode regression | Searched for `TestMatch*` in `plugins/silverbullet/*_test.go` | No such test exists | FAIL (see Gaps) |
| Repo-root, sdk, paperless, silverbullet, mock module test suites | `go test ./...` (once) in each module | All pass, no failures | PASS |
| Frontend typecheck/tests | `cd web && npm run check && npm run test` | 0 errors/warnings; 44/44 tests pass | PASS |
| `make build` produces kernel + 3 plugin binaries | `make build` | `bin/webspaces`, `bin/plugins/webspaces-plugin-{paperless,silverbullet,mock}` all produced | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| SRC-05 | 02-01 | SilverBullet plugin; tag/page matching; exact deep links | SATISFIED | Live cross-source stream, `MatchesKeyword`/`isPage` code read, `fidelity: exact` confirmed live |
| KERN-04 | 02-01, 02-02 | Sync scheduler + coordinator; dedups refreshes; manual refresh | SATISFIED (with the CR-01 caveat noted in truth #6 — the coordinator/scheduler/single-flight layer itself is correct; the defect is one layer down, inside a specific plugin's own error handling, and doesn't violate KERN-04's own text) | Live refresh dedup, `-race` singleflight test, scheduler ticker code read |
| PLUG-04 | 02-02 | Plugins report health (reachable, last sync, last error) | SATISFIED | Live `GET /api/sources` shape matches spec exactly |
| PLUG-05 | 02-04 | Third party can build a plugin from contract + mock alone | SATISFIED with a documented rigor caveat | `plugins/mock` built, tested, launched live; the isolation exercise was performed by the same executor (not a dispatched fresh subagent — no such tool was available), a materially weaker approximation than the plan's own already-weakened fallback, honestly recorded in 02-04-SUMMARY.md. Routed to Human Verification below rather than treated as a silent pass. |
| UI-02 | 02-03 | Filter by source | SATISFIED | URL-persisted filter wired and unit-tested; visual click-through not browser-verified this session |
| UI-05 | 02-03 | Stale/unavailable explicit states, never silent 404 | SATISFIED at the UI decision-logic layer; UNDERMINED at the sync layer by the CR-01 gap (truth #6), which can produce a *silent* data loss the UI has no way to detect or flag, since the sync run is recorded as `ok` | `detailPaneState` unit tests pass; CR-01 gap is the counter-example |
| UI-06 | 02-03 | Sync status and plugin health visible | SATISFIED | Health chips wired, live `GET /api/sources` correct |
| AGENT-01 | 02-04 | Per-plugin permission model, default-deny | SATISFIED | Default-deny by Go zero value confirmed in code and live; no-existence-leak confirmed by identical error construction |

No orphaned requirements: all 8 IDs the task specified (KERN-04, PLUG-04, PLUG-05, SRC-05, UI-02, UI-05, UI-06, AGENT-01) are exactly the union of every plan's `requirements:` frontmatter field, and match REQUIREMENTS.md's traceability table (all marked "Complete" for Phase 2) with no gaps in either direction.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `plugins/silverbullet/plugin.go` | 94-120 | Silent error-swallowing that contradicts the plugin's own documented contract (CR-01) | 🛑 Blocker | See Gaps — silent data loss on a genuine SilverBullet outage mid-sync |
| `web/src/lib/components/DetailPane.svelte` | 33-40, 72 | Local `formatDate` reintroduces the UTC timezone bug `web/src/lib/format.ts`'s `formatItemDate` was built to fix (WR-01 in 02-REVIEW.md, still unfixed) | ⚠️ Warning | For a paperless date-only midnight-UTC value, a user west of UTC can see the stream row and the detail pane disagree on the calendar day for the same item — a real, provable correctness bug, but narrow (date display only) and not core to this phase's must-haves |
| `web/src/lib/components/DetailPane.svelte` | 143, 149 | `<iframe>` elements carry no `sandbox` attribute despite `kernel/httpapi/item.go`'s comment claiming a second, independent client-side sandbox layer exists (WR-02, still unfixed) | ⚠️ Warning | Not currently exploitable — the CSP `sandbox` response header already sandboxes the framed document regardless of the iframe element's own attribute — but the comment/code mismatch is a latent hazard for a future CSP change |
| `kernel/httpapi/stream.go:79`, `webspaces.go:37`, `agent.go:232` | — | `LatestSyncRunPerSource` errors are silently downgraded to the neutral "unknown" sync state instead of a 500, inconsistent with the sibling `agentWebspacesHandler` in the same package (WR-03, still unfixed) | ⚠️ Warning | A genuine SQLite failure would present as "never synced" rather than a server error — low likelihood, narrow blast radius |
| `cmd/webspaces/main.go` | `runServe` | No `signal.Notify`/`signal.NotifyContext` handler; deferred cleanup (`store.Close()`, `host.Shutdown()`, scheduler `cancel()`) never runs on a normal `Ctrl+C`/`SIGTERM` (WR-04, still unfixed) | ⚠️ Warning | Operational hygiene issue, not a phase-goal blocker; the scheduler goroutines and plugin subprocesses currently rely on process-death cleanup rather than an explicit graceful shutdown path |

No `TBD`/`FIXME`/`XXX` debt markers found in any file this phase touched.

## Human Verification Required

### 1. Visual confirmation of health chips, filter chips, and staleness states in a real browser

**Test:** With both sources configured (already live-verified for reachable/ok state), deliberately stop the SilverBullet service and reload the webspace in a browser. Confirm: two health chips render with correct colors (paperless green, SilverBullet red); hovering SilverBullet's chip shows the unreachable tooltip with a relative time and the full untruncated error; SilverBullet rows carry an amber stale marker while paperless rows are visually unchanged; opening a SilverBullet item shows its cached title/date/labels/open-in-source link with the unreachable alert layered over the cached preview (never a blank pane); clicking the source filter chips narrows/restores the list and the URL gains/loses the `source` query parameter, surviving a reload.

**Expected:** Matches `02-UI-SPEC.md`'s Copywriting Contract and Design System exactly (colors, wording, layout).

**Why human:** No browser tooling was available in the executing sessions for plans 02-02/02-03/02-04 (documented in all three SUMMARY.md files). This is proven correct at the pure-decision-function/unit-test layer (`healthTone`, `detailPaneState`, `streamVariant`, `staleSourceTypes` — 44/44 vitest tests pass) and at the live API layer (this verification's own curl checks), but the actual rendered pixels, tooltip wrapping, and click interactions have not been visually confirmed since 02-01's tracer gate (which the user already approved in this session for interleaving/deep-links/detail-pane rendering only).

### 2. PLUG-05's third-party self-sufficiency claim, from a genuinely fresh/isolated context

**Test:** Have a truly isolated agent (a real dispatched subagent, or a human with no prior exposure to this repository) attempt to build a `SourcePlugin` using only `proto/webspaces/v1/plugin.proto`, `docs/plugin-contract.md`, the `sdk` module, and `plugins/mock` — with no access to `plugins/paperless` or `plugins/silverbullet` and no memory of them.

**Expected:** A clean build with zero or few gaps, corroborating the two-gap-then-zero-gap result 02-04's in-session exercise already found.

**Why human:** 02-04-SUMMARY.md honestly documents that no subagent-dispatch tool was available in that execution environment, so the "isolation" exercise was performed by the same executor that had just written `plugins/paperless` and `plugins/silverbullet` earlier in the same session and retained full memory of their real implementations — a materially weaker approximation than even the plan's own already-flagged A-PLUG-05 limitation ("a fresh agent context shares this project's general Go and gRPC knowledge"). The artifact itself (`plugins/mock`, builds and passes 9 tests, launches live through the real kernel) is fully verified; the *process* claim ("a third party can build a plugin from the docs alone") has real but weaker-than-designed evidence.

## Gaps Summary

One BLOCKER: `plugins/silverbullet/plugin.go`'s `Match` method (lines 94-120) treats every per-page `ReadFile` failure — not just "page not found" — as "this page doesn't match," discarding transport, TLS, and auth failures identically to a normal non-match. This was flagged as a critical finding (CR-01) in `02-REVIEW.md` (committed as part of this phase, `aacda62`) and remains unfixed in the current tree: no commit after the review touches `plugins/silverbullet/plugin.go`, and no test in the module exercises `Match`'s failure-aggregation behavior at all. The practical consequence is a silent-failure mode this phase's other architecture (source-scoped persistence, two-phase `sync_runs`, aggregate `sync.status`, the UI's explicit-stale-state requirement) was specifically built to eliminate: if SilverBullet becomes fully unreachable partway through a sync (after its file listing already succeeded), `Match` returns a *successful* empty result, the sync is recorded as `ok` with `item_count: 0`, and every previously-synced SilverBullet item is silently deleted from every webspace — with no error surfaced anywhere the UI or the health API can detect. This is a direct violation of the phase's own core value proposition ("tell at a glance whether what they are looking at is complete and current") and of `02-03-PLAN.md`'s test-tier prohibition ("MUST NOT present stale, partial or filtered content as the complete and current state of a webspace ... silently omitting it is the single worst failure mode"). The fix is scoped and mechanical (distinguish `errors.Is(err, ErrNotFound)` from any other error, propagate the latter so `Match` returns `codes.Unavailable`) — the review's own suggested patch is directly usable — plus one regression test.

Two human-verification items are outstanding and non-blocking: visual confirmation of the 02-02/02-03/02-04 UI surfaces in a real browser (no browser tooling was available across those three execution sessions), and a genuinely isolated re-run of the PLUG-05 validation exercise (the in-session exercise used an executor with prior knowledge of the real plugins, a materially weaker approximation than the plan intended).

Four non-blocking warnings from `02-REVIEW.md` remain unfixed (WR-01 through WR-04: a UI timezone-formatting inconsistency, a documented-but-missing iframe `sandbox` attribute with no active exploit, silently-swallowed sync-run lookup errors in three handlers, and no graceful-shutdown signal handling) — none of these block the phase goal on their own, but they should be tracked for a follow-up fix pass alongside the CR-01 closure.

---

_Verified: 2026-07-29T00:08:13Z_
_Verifier: Claude (gsd-verifier)_
