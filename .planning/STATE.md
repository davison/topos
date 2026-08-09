---
gsd_state_version: 1.0
milestone: v1.0
milestone_name: milestone
status: executing
last_updated: "2026-08-09T12:46:39.695Z"
last_activity: 2026-08-09
progress:
  total_phases: 9
  completed_phases: 6
  total_plans: 53
  completed_plans: 53
  percent: 78
---

# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-08-07)

**Core value:** Open one webspace and instantly see and grok all related information across every silo — without visiting each data store individually.
**Current focus:** Phase 07 — webspace-builder-ui

## Current Position

Phase: 07 (webspace-builder-ui) — AWAITING HUMAN VERIFICATION
Plan: 14 of 14 executed
Status: All plans executed; verification status human_needed — run /gsd-verify-work 7 (8 UAT tests in 07-UAT.md; 4 are live re-confirmations of the closed G-07-3..G-07-6 gaps)
Last activity: 2026-08-09

Progress: [██████████] 100%

## Performance Metrics

**Velocity:**

- Total plans completed: 39
- Average duration: —
- Total execution time: —

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01 | 6 | - | - |
| 02 | 6 | - | - |
| 03 | 10 | - | - |
| 04 | 4 | - | - |
| 05 | 5 | - | - |
| 06 | 8 | - | - |

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
| Phase 04 P01 | 3h | 2 tasks | 20 files |
| Phase 04 P02 | 2h | 3 tasks | 18 files |
| Phase 04 P03 | ~2.5h | 3 tasks | 13 files |
| Phase 260805-kt3 P01 | ~20min | 1 tasks | 3 files |
| Phase 05 P01 | 40min | 3 tasks | 35 files |
| Phase 05 P02 | ~2h | 3 tasks | 22 files |
| Phase 05 P03 | 55min | 2 tasks | 10 files |
| Phase 05 P04 | ~20min | 2 tasks | 22 files |
| Phase 05 P05 | 35min | 3 tasks | 5 files |
| Phase 06 P02 | ~20min | 3 tasks | 13 files |
| Phase 06 P03 | 20min | 3 tasks | 7 files |
| Phase 06 P04 | ~12min | 1 tasks | 3 files |
| Phase 06 P05 | ~15min | 3 tasks | 8 files |
| Phase 06 P06 | ~15min | 2 tasks | 4 files |
| Phase 06 P07 | 32min | 3 tasks | 7 files |
| Phase 06 P08 | 13min | 2 tasks | 4 files |
| Phase 07 P01 | 52min | 3 tasks | 31 files |
| Phase 07 P02 | ~2h | 3 tasks | 23 files |
| Phase 07 P03 | ~8min | 3 tasks | 32 files |
| Phase 07 P04 | ~1h | 3 tasks | 16 files |
| Phase 07 P05 | ~16 min | 3 tasks | 17 files |
| Phase 07 P06 | ~15min | 2 tasks | 4 files |
| Phase 07 P07 | 22min | 2 tasks | 2 files |
| Phase 07 P08 | ~20min | 2 tasks | 5 files |
| Phase 07 P09 | 17min | 2 tasks | 2 files |
| Phase 07 P10 | ~15min | 2 tasks | 2 files |
| Phase 07 P11 | 45min | 3 tasks | 14 files |
| Phase 07 P13 | ~18min | 3 tasks | 9 files |
| Phase 07 P12 | 10min | 2 tasks | 5 files |
| Phase 07 P14 | 8min | 3 tasks | 7 files |

## Accumulated Context

### Roadmap Evolution

- Phase 5 inserted after Phase 4: Source Instances & Per-Type Matching — named multi-instance plugins, per-plugin-type matching config replacing the shared keyword list (restructure 2026-08-05)
- Phase 6 inserted after Phase 5: UI: Scalable Source Surface — combined health/filter source affordance, deep-link fidelity differentiation, detail-pane search highlighting, themed scrollbars with date markers
- Phase 7 inserted after Phase 6: Webspace Builder UI — configure plugin instances from the UI, save as webspace, saved searches as permanent filters
- Phase 8 edited: WhatsApp Conversations (Managed Risk) shifted from Phase 5 to Phase 8; now depends on Phase 4 (chat renderer) and Phase 5 (per-instance matching contract)

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
- [Phase ?]: [Phase 04-01]: Task 1 checkpoint authorised dynamically linking the system SQLCipher via a libsqlcipher-tagged mattn/go-sqlite3 fork (option-a), pinned by go.mod replace to jgiannuzzi/go-sqlite3 v1.14.17-0.20230327162135-f208443ec79d (branch sqlcipher, upstream PR mattn/go-sqlite3#1109, commit f208443ec79de7edaf1b80276806005a5c0cf340) over CLAUDE.md's mutecomm/go-sqlcipher/v4, which bundles a pre-3.51.3 SQLite core and fails the phase's WAL-corruption-fix floor
- [Phase ?]: [Phase 04-01]: PRAGMA user_version ceiling pinned to 1730, read live off the real Signal Desktop database — not 04-RESEARCH.md's unconfirmed placeholder of 1640
- [Phase ?]: encryptedKey is hex-encoded (confirmed against Signal Desktop's live source), not base64 as illustrated in research
- [Phase ?]: Secret Service application attribute value pinned to 'Signal' (traced from Chromium os_crypt + Electron app.getName()), unverifiable on this machine (never safeStorage-migrated) — flagged for re-verification
- [Phase ?]: kwallet/kwallet5/kwallet6 route through freedesktop Secret Service (not native org.kde.KWallet), a documented scope limitation per plan instruction
- [Phase ?]: [Phase 04-03]: Attachments and reactions read from Signal Desktop's own dedicated message_attachments/reactions SQL tables, never the message row's json blob — corrects 04-RESEARCH.md's illustrative assumption, confirmed by direct schema introspection of the real, live database
- [Phase ?]: [Phase 04-03]: Fetch's FULL and PREVIEW variants share one path (fetchTranscript) — a Signal digest has no separate richer-preview-vs-extracted-text distinction the way an email does
- [Phase ?]: [Quick 260805-kt3]: webmailSearchDeepLink extended to a deepLinkCriteria struct (subject/sender/date) via strict RED->GREEN TDD, keeping all 03-10 assertions byte-identical; Task 2 (live Proton checkpoint) and Task 3 (land verdict, update docs) deliberately not executed this run
- [Phase ?]: [Quick 260805-kt3 Task 3]: Task 2's live-Proton checkpoint returned a bare 'approved' verdict (no corrections, no drops, no address-bar URL) — assumption register rows A-2..A-7 recorded as CONFIRMED-BY-BEHAVIOR (not the stronger CONFIRMED-BY-CANONICALIZED-URL the plan hoped for); no source change needed, config.example.toml/kernel/config/types.go docs updated to match shipped behavior
- [Phase ?]: [Phase 05-01]: Instance identity split across kernel/HTTP/agent/UI (D-08); kernel/httpapi/routes.go modified beyond the plan's declared scope (Rule 3) to wire cfg into StreamHandler/ItemHandler for source_display_name resolution
- [Phase ?]: [Phase 05-01]: search.go/toSearchResult intentionally kept calling the unchanged toStreamItem(it) signature (out of plan scope) — a new toStreamItemFor(it, resolveDisplayName) sibling serves every other caller, so a search result's source_display_name falls back to the instance id rather than any configured override
- [Phase ?]: [Phase 05-02]: Task 1 checkpoint locked option-a exactly as proposed — generic MatchRequest.match_fields map, proto package stays topos.v1, sdk.Handshake.ProtocolVersion 1->2, DescribeResponse.contract_version becomes topos.v2 (documented as independent of the proto package path)
- [Phase ?]: [Phase 05-02]: kernel/syncer test fixtures, four proton test files, and signal's D-06 regression test all required Rule 3 fixes outside this plan's declared file list — same-package compile dependencies and fixture infrastructure needs
- [Phase ?]: [Phase 05-03]: 05-RESEARCH.md Open Question 1 decided as specified — a match block for an instance excluded by the same webspace's sources allowlist is dead config and fails load, naming both
- [Phase ?]: [Phase 05-03]: A match field's zero-length value list is treated as a validation failure alongside the plan's named empty-field-name/empty-value cases, reading must_haves' silently-matching-nothing framing broadly
- [Phase ?]: [Phase 05-03]: pluginhost.ValidateMatchConfig also guards a match block or fallback naming an unlaunched instance, defensive completeness verified via fake-plugin-list fixtures rather than a live subprocess mismatch
- [Phase ?]: [Phase 05-04]: Proto ContentShape enum/field and sdk regeneration landed in Task 1's commit (Rule 3 blocking compile dependency) since rendition.go's policy map is keyed by toposv1.ContentShape before Task 2's own declared proto work
- [Phase ?]: [Phase 05-04]: Avoided 'go mod tidy' for the root module (pulls a synthetic pseudo-versioned require on the workspace-local sdk module plus unrelated buf/protoreflect transitives) — used a targeted 'go get' for bluemonday instead, per plugins/proton/go.mod's own documented limitation
- [Phase ?]: [Phase 05-04]: Email content-shape stylesheet carries forward proton's full readability-layer !important neutralizer even though 05-UI-SPEC.md's contract table omits it, since the plan's own must_haves require visual parity with pre-move output
- [Phase ?]: [Phase 05-05]: docs/plugin-contract.md's stale {source_type}:{source_id} id-scheme claim corrected to {source}:{source_id} beyond the plan's literal scope (Rule 1 bug, left over from before 05-01)
- [Phase ?]: [Phase 05-05]: Operator's real webspaces migrated with an explicit per-instance match block for every configured instance (same values as the old shared keyword list), reproducing the D-01 fallback byte-for-byte while keeping the keywords line as a documented safety net
- [Phase ?]: [Phase 05-05]: Live migration verified against an ephemeral second kernel instance (XDG_CONFIG_HOME override, separate port, same already-synced index) rather than restarting the user's own pre-existing make dev session on 127.0.0.1:7777
- [Phase ?]: [Phase 06-02]: implemented visibleChipCount per the plan's own <action> algorithm (full-fit check before charging the overflow trigger's width) rather than its internally-inconsistent [10,10,10],35,0,8->2 acceptance-criteria example; the 30,0,8->3 exact-fit example is satisfied verbatim
- [Phase ?]: [Phase 06-02]: worstHealthTone seeds its reduction with the least-alarming tone (success) rather than 'unknown', with the empty-input case handled by an explicit early return — seeding at 'unknown' silently mis-scored an all-success input
- [Phase ?]: [Phase 06-03]: Duplicated the stream-pane scroll div's conditional width classes onto a new outer relative wrapper (the real flex item main sizes) rather than moving them, preserving pane-layout.test.ts's existing fixed-width source-scan guard unchanged
- [Phase ?]: [Phase 06-04]: Observer attachment extracted into web/src/lib/resize-observer.ts (injectable createObserver factory) rather than inlined, so attachment is provable by behaviour under the node-environment test runner with no component-mount harness
- [Phase ?]: [Phase 06-04]: Structural proof is a comment-stripped source-scan guard with balanced-paren extraction, not a raw grep -- a bare grep is what let 06-VERIFICATION.md's gap through once already
- [Phase ?]: [Phase 06-05]: Promoted .search-highlight from DetailPane.svelte's component-scoped <style> block to app.css @layer components — a Svelte-scoped class cannot be shared across sibling components, the mechanical root cause of G-06-1's vocabulary drift
- [Phase ?]: [Phase 06-05]: Superseded (not deleted) 03-UI-SPEC.md's Phase 3 weight-only match-emphasis rule with a dated note naming G-06-1, arguing the colour-only treatment restores the 'exactly 2 weights' contract more purely than the retired weight exception did
- [Phase ?]: [Phase 06-06]: Chose a solid bg-primary fill over a color-corrected ring for the selected chip — a fill paints inside the border box and needs no change to WebspaceHeader.svelte's load-bearing overflow-hidden, whereas even a correctly-colored ring would still be clipped top/bottom by that same ancestor
- [Phase ?]: [Phase 06-07]: 65% color-mix of --muted-foreground chosen for --stream-marker's rest tone (computed 3.81:1 vs --background, 3.67:1 vs --card), superseding the retired --scrollbar-thumb reuse's 1.86:1 that caused G-06-6
- [Phase ?]: [Phase 06-07]: markerLaneTop clamps the degenerate case (track shorter than twice the inset) to zero usable range rather than allowing an inverted position mapping
- [Phase ?]: [Phase 06-07]: Declined date labels at major boundaries (recorded in 06-UI-SPEC.md) — a 12px lane cannot host legible label text without reintroducing the row-banding defect; tooltip + major/minor hierarchy already carry that information
- [Phase ?]: [Phase 06-08]: Kept source-chip-pill.test.ts as a file separate from source-chip-selected.test.ts (house pattern) — the two guard different axes (colour vs. geometry/interaction) of the same shipped defect class and should fail as distinguishable incidents
- [Phase ?]: [Phase 06-08]: group-has-[:focus-visible]:opacity-100 (the plan's primary Tailwind arbitrary variant) compiled on the first build attempt, confirmed via grep on the emitted stylesheet — the plan's named fallback (group-has-focus-visible) was not needed
- [Phase ?]: [Phase 07-01]: assumption-delta locked at Task 1's tracer checkpoint — the running configuration is promoted to the primary noun (config.Store); WebspacesHandler/ItemHandler/SourceRefreshHandler keep a boot-time cfg snapshot as accepted debt for this plan alone (07-02 Task 2 fills the gap)
- [Phase ?]: [Phase 07-01]: Store.Search's empty-result short circuit depends on BOTH filterTerms and rawQuery sanitizing to nothing — a filter-only call still queries and ranks by relevance rather than returning early
- [Phase ?]: [Phase 07-01]: $state.snapshot() (not structuredClone) for cloning a fetched config document before mutation — Svelte 5's reactive Proxy unconditionally rejects structuredClone in every engine, a live tracer-checkpoint repro (d8125cf)
- [Phase ?]: [Phase 07-02]: Apply replaces the *syncer.Coordinator wholesale on every reconcile — Supervisor itself satisfies Fetcher/HealthProber/Refresher (delegating fresh per call) rather than passing sup.Host()/sup.Coordinator() captured once into Router (Rule 1 bug, found during Task 1)
- [Phase ?]: [Phase 07-02]: In-flight sync during apply — cancel and BLOCK until the old scheduler generation fully returns before Host.Reconcile runs, relying on Coordinator.syncOne's pre-existing detached sync_runs finalize for the interrupted sync's own outcome
- [Phase ?]: [Phase 07-02]: 07-RESEARCH.md assumption A2 confirmed for paperless/silverbullet/proton/signal — all four defer live connectivity past process startup; Proton's NewClient only validates the base_url scheme (imap/imaps), never dials
- [Phase ?]: [Phase 07-03]: Hand-scaffolded dialog/dropdown-menu/alert-dialog overlay primitives directly against bits-ui's own type/export shape (no confirmed network access for npx shadcn-svelte add); config-edit.ts's cloneConfig uses a JSON round trip (not structuredClone/$state.snapshot) since it's a plain .ts module; last-webspace.ts pulled forward into Task 2 (Rule 3, blocking compile dependency)
- [Phase ?]: [Phase 07-04]: AddSourceModal mounted inline in WebspaceHeader's chip row (not route-level open/onclose) — bits-ui Popover trigger/content co-location constraint plus the row's own overflow-measurement need for the + button's real DOM position
- [Phase ?]: [Phase 07-04]: SecretField's set/unset badge reads the caller-supplied envVars snapshot synchronously, not a per-keystroke live lookup — kernel/httpapi/config.go's envVarsIn only reports on names already referenced in the persisted config, and no kernel/ files are in this plan's scope to add a live-arbitrary-name endpoint
- [Phase ?]: [Phase 07-04]: one-step existing-instance add and chip-menu Edit-match-settings both resolve vocabulary via describePlugin against the instance's own stored config, substituting for GET /api/sources (no match_vocabulary field there today)
- [Phase ?]: [Phase 07-05]: removeSourceInstance writes sources=[] (never omits the key) on a delete that empties a webspace's allowlist — Webspace.Participates treats empty identically to absent, so [] IS the kernel's own all-instances-participate default encoding
- [Phase ?]: [Phase 07-05]: CONFIG_CONFLICT_MESSAGE (api.ts) pulled forward into Task 1's commit — every config-writing surface now references the one exported hash-conflict-copy constant by name instead of duplicating the literal string
- [Phase ?]: [Phase 07-05]: TestContract_MutatingRoutesAreConfigScoped duplicates config_test.go's pre-existing routes.go AST scan by design (plan names this exact test/file); its only new coverage is asserting agent.go registers zero non-GET routes
- [Phase ?]: [Phase 07-06]: resolveNewInstanceId returns a discriminated InstanceIdResult rather than throwing — fits both call sites' existing connectError/message-string control flow with no new exception path
- [Phase ?]: [Phase 07-06]: saveAnyway's rejection branch deliberately does not clear describeFailed (unlike handleConnectNext) — preserves the Save anyway retry affordance on a colliding name, pinned by a structural test
- [Phase ?]: [Phase 07-07]: grantedSources kept its existing *config.Config parameter unchanged (pure helper, not a handler) — threading *config.Store into it would hide the 'resolve once, at the top' discipline inside a helper
- [Phase ?]: [Phase 07-07]: AST guard (TestAgentGuard_EveryHandlerResolvesConfigPerRequest) enumerates the agent handler set by exact name equality and was verified to actually fail via a temporary, git-diff-confirmed-clean revert-and-restore of agentSourcesHandler
- [Phase ?]: [Phase 07-08]: editMode resets to 'connection' (not null) in resetEditSession — clearing editInstance alone already destroys the {#if} guard's subtree, which is the whole CR-02 fix mechanism; widening editMode's type would fail npm run check
- [Phase ?]: [Phase 07-08]: edit-modal-state.ts's seedConnectionValues/seedMatchBlock return fresh objects/arrays, never aliasing the config document — deliberate divergence from prior inline seeding, load-bearing for the CR-02 regression test and for EditSourceModal's new untracked reset-on-open effect
- [Phase ?]: [Phase 07-09]: Rolled forward (never rolled back) every post-Reconcile Apply failure branch through a shared commitGeneration site — closes gaps[0]; folded in a previously-unreported ordering defect in the D-07 index-cleanup branches
- [Phase ?]: [Phase 07-10]: Apply's post-Reconcile region collapsed to one shared commitGeneration call and one errors.Join-based error return; cleanupRemovedInstances extracted and runs unconditionally before the match-vocabulary check, per-instance failures collected rather than returned early
- [Phase ?]: [Phase 07-11]: D-20 — empty webspace shell (no keywords/sources/match) accepted by config.Validate and correlates nothing; validateFallbackCoverage/Participates unchanged, shell exemption short-circuits before either is reached
- [Phase ?]: [Phase 07-11]: Two pre-existing config_test.go tests (TestLoad_ZeroKeywordsFails, TestLoad_WebspaceWithNeitherKeywordsNorMatchFails) encoded exactly the pre-D-20 fixture the plan's own must_haves require to load — updated in place (Rule 1), documented as a deviation since it contradicts the plan's blanket 'do not modify' instruction
- [Phase ?]: [Phase 07-13]: Required flags re-derived by reading all four plugins' own pre-Serve guards (not just the two the UAT report named) — Signal's path and Proton's webmail_base_url join the required set; only Signal's path gets a seeded default since Proton's is installation-specific
- [Phase ?]: [Phase 07-13]: Enforcement lives in three pure helper functions called from each of the three submit handlers' own bodies (AddSourceModal's handleConnectNext/saveAnyway, EditSourceModal's submitConnection) rather than inside the shared ConnectionForm component
- [Phase ?]: [Phase 07-13]: kernel/pluginhost.launch's stderrTail capture is bounded at 4 KiB, mutex-guarded, front-discard, and read only after client.Kill() returns — covers boot-time/hot-apply launches identically to UI trial launches since all three share launch()
- [Phase ?]: [Phase 07-12]: applyDefaults normalizes Sources/Webspaces top-level maps and per-webspace Keywords/Sources/Match collections to non-nil empty values (Filter deliberately excluded, D-17/D-18) — closes 07-UAT.md G-07-4's kernel-side half, GET /api/config never nulls a collection the SPA iterates
- [Phase ?]: [Phase 07-12]: root route's onMount isolates the getConfig() fetch in its own catch that returns immediately — all post-fetch processing (redirect resolution, empty-phase assignment) runs outside any catch so a downstream bug can no longer render the kernel-unreachable copy
- [Phase ?]: 07-14: removeSourceFromWebspace seeds the participant set before filtering (mirrors addSourceToWebspace), closing G-07-6's config-write no-op
- [Phase ?]: 07-14: WebspaceHeader chip content filters through participation.ts's shared predicate; row visibility deliberately stays unfiltered so the + trigger survives zero-participant webspaces

### Pending Todos

[From .planning/todos/pending/ — ideas captured during sessions]

- 2026-08-05 — Signal schema-version verify-and-accept tooling (minor, tooling)
- 2026-08-05 — Centralize rendition theming (and sanitization) in the kernel content boundary (major, api; schedule with Phase 5 contract work or ecosystem milestone)

### Blockers/Concerns

- Phase 8 (WhatsApp, formerly Phase 5): Highest-risk area. No official API; linked-device route can be de-linked or banned. Spike must answer linking stability, backfill volume, event-stream persistence, and recovery before planning.
- ⚠️ [Phase 6] 06-REVIEW.md WR-01 still open (advisory): client-side `highlightText` in `web/src/lib/format.ts` bulk-lowercases then indexes positionally — highlight spans mis-position after case-fold-expanding characters (e.g. İ). Narrow, untested edge case; fix via `/gsd-code-review 6 --fix` or fold into Phase 7 UI work.

### Quick Tasks Completed

| # | Description | Date | Commit | Directory |
|---|-------------|------|--------|-----------|
| 260729-p2n | create a wrapper script that exposes the env vars in .env to the webspaces binary | 2026-07-29 | 7becca1 | [260729-p2n-create-a-wrapper-script-that-exposes-the](./quick/260729-p2n-create-a-wrapper-script-that-exposes-the/) |
| 260805-irt | fix pane flex: stream pane fixed width, detail pane flexes on viewport resize | 2026-08-05 | 4e51006 | [260805-irt-fix-pane-flex-stream-pane-fixed-width-de](./quick/260805-irt-fix-pane-flex-stream-pane-fixed-width-de/) |
| 260805-j98 | style scrollbars: thin, theme-matched app-wide (incl. rendition iframes) | 2026-08-05 | 2604de1 | [260805-j98-style-scrollbars-thin-theme-matched-app-](./quick/260805-j98-style-scrollbars-thin-theme-matched-app-/) |
| 260805-kt3 | narrow Proton deep-link search with sender+date criteria (live-approved) | 2026-08-05 | 1fb1fa6 | [260805-kt3-narrow-proton-deep-link-search-for-gener](./quick/260805-kt3-narrow-proton-deep-link-search-for-gener/) |
| 260805-lry | accept Signal schema 1740 after live read-set verification (source recovered) | 2026-08-05 | 9f000c3 | [260805-lry-accept-signal-desktop-schema-version-174](./quick/260805-lry-accept-signal-desktop-schema-version-174/) |
| 260805-o5d | harden make dev: plugins rebuilt as prerequisite, fail loudly on port squat/dead kernel | 2026-08-05 | 6d0e6a8 | [260805-o5d-harden-make-dev-rebuild-plugin-binaries-](./quick/260805-o5d-harden-make-dev-rebuild-plugin-binaries-/) |
| 260806-f1 | (fast) refresh README: 8-phase roadmap, five-plugin layout, seven workspace modules | 2026-08-06 | cd8ba20 | — |
| 260806-f2 | (fast) expose Vite dev server (:5173) to the tailscale network via make dev | 2026-08-06 | 54d30b4 | — |

## Deferred Items

Items acknowledged and carried forward from previous milestone close:

| Category | Item | Status | Deferred At |
|----------|------|--------|-------------|
| *(none)* | | | |

## Session Continuity

Last session: 2026-08-09T12:46:39.686Z
Stopped at: Completed 07-14-PLAN.md
Resume file: None
