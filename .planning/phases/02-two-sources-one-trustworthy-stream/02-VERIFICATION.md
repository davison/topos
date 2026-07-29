---
phase: 02-two-sources-one-trustworthy-stream
verified: 2026-07-29T10:45:00Z
status: human_needed
score: 6/6 must-haves verified
behavior_unverified: 0
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 5/6
  gaps_closed:
    - "A SilverBullet sync that experiences a genuine outage or auth failure mid-Match (after ListFiles succeeds) is recorded as an error, not silently reported as a successful sync with zero items — and does not silently delete every previously-synced SilverBullet item for every webspace"
  gaps_remaining: []
  regressions: []
---

# Phase 2: Two Sources, One Trustworthy Stream Verification Report

**Phase Goal:** User sees two structurally different sources interleaved in one webspace and can filter, refresh, and tell at a glance whether what they are looking at is complete and current

**Verified:** 2026-07-29T10:45:00Z
**Status:** human_needed
**Re-verification:** Yes — after gap closure (plan 02-05)

**MVP mode note:** ROADMAP.md marks this phase `Mode: mvp`, but the ROADMAP goal line is not in canonical "As a / I want to / so that" form and all plans (including this gap-closure plan) explicitly declined to invent one. Per the MVP verification guard, standard goal-backward verification is applied against the ROADMAP Success Criteria, which remain the authoritative contract regardless of mode.

## Re-Verification Summary

The prior `02-VERIFICATION.md` (2026-07-29T00:08:13Z) scored 5/6 with exactly one FAILED truth (#6): `plugins/silverbullet/plugin.go`'s `Match` swallowed every per-page `ReadFile` failure identically to "page not found," so a genuine SilverBullet outage or auth failure occurring after a successful `ListFiles` produced a **successful, empty** `MatchResponse` — which the kernel would then persist as a deletion of every previously-synced SilverBullet item, recorded as a successful (`ok`) sync run.

Gap-closure plan `02-05` (commits `692573e` test, `ae0fa88` fix) closed this gap. This re-verification re-checks the fix directly against the code and the named regression tests — not against SUMMARY.md prose.

### Direct evidence gathered this session

1. **Read `plugins/silverbullet/plugin.go` lines 74-150 directly.** The `Match` closure now reads:
   ```go
   raw, err := p.client.ReadFile(gctx, f.Name)
   if err != nil {
       // ... [comment documents both branches, cites CR-01/truth #6] ...
       if errors.Is(err, ErrNotFound) {
           return nil
       }
       return err
   }
   ```
   and the `g.Wait()` branch (previously dead code) is now reachable:
   ```go
   if err := g.Wait(); err != nil {
       return nil, status.Errorf(codes.Unavailable, "silverbullet: match: %v", err)
   }
   ```
   This is exactly the two-way discrimination the gap-closure plan specified: `errors.Is(err, ErrNotFound)` is the sole safe-skip condition; every other error (dropped connection, TLS failure, revoked/expired token, 5xx, cancelled context) now propagates and fails the whole `Match` with `codes.Unavailable`.

2. **Ran the named regression tests directly** (not via SUMMARY claim):
   ```
   cd plugins/silverbullet && go test ./... -run TestMatch -v
   ```
   Result: all five `TestMatch_*` functions PASS, including the two named in the gap-closure plan's own re-verification instructions:
   - `TestMatch_AllPageReadsFail_ReturnsUnavailable` — PASS (asserts `codes.Unavailable` when every page read returns 500 after a successful listing)
   - `TestMatch_OutageMidSync_AuthFailure_ReturnsUnavailable` — PASS (asserts `codes.Unavailable` when the first read succeeds and every subsequent read returns 401, simulating mid-sync token revocation)
   - `TestMatch_UnavailableError_NeverContainsBearerToken` — PASS (asserts neither `err.Error()` nor the gRPC status message contains the configured token)
   - `TestMatch_HappyPath_ReturnsOnlyKeywordMatchedPages` — PASS (green-both-ways guard against over-correction)
   - `TestMatch_PageDeletedBetweenListingAndRead_SkippedNotFailed` — PASS (a 404 read is still skipped, not failed — proves the fix didn't convert the legitimate deleted-page case into a hard failure)

3. **Read `plugins/silverbullet/match_test.go` in full.** Confirmed the tests genuinely exercise the failure modes via real `httptest.Server` fixtures (500s and 401s returned by an actual HTTP handler, not mocked-away), asserting on `status.FromError(err).Code()` rather than string matching, with goroutine-safe (`sync/atomic`) shared handler state under the `errgroup` concurrency the code under test actually uses.

4. **Ran the full module test suite with the race detector**: `cd plugins/silverbullet && go test ./... -race` — PASS, no regressions in the pre-existing 30+ tests, no data race in the new fixtures.

5. **Ran `make test`** at the repo root — all five workspace modules (root, `sdk`, `plugins/paperless`, `plugins/silverbullet`, `plugins/mock`) build and pass.

6. **Ran `make build`** — produces `bin/webspaces` and all three plugin binaries (`webspaces-plugin-paperless`, `webspaces-plugin-silverbullet`, `webspaces-plugin-mock`).

7. **Ran the frontend sanity suite**: `cd web && npm run check && npm run test` — 0 errors/warnings; 44/44 vitest tests still pass (untouched by this gap-closure plan, confirmed no regression).

8. **Confirmed change scope discipline**: `git diff --stat` between the pre-gap-closure commit (`dacbd63`) and the current tree, restricted to non-`.planning` paths, touches exactly `plugins/silverbullet/match_test.go` (new, 228 lines) and `plugins/silverbullet/plugin.go` (+30/-5 lines) — nothing else in the codebase changed. This means the other five truths verified in the prior report have zero code-level regression risk; their supporting artifacts are byte-identical to when they were last verified VERIFIED.

9. **Cross-checked against an independent second code review** (`02-REVIEW.md`, re-reviewed 2026-07-29T00:00:00Z after the gap closure, 28 files, a separate review pass from this verification): confirms "CR-01 verified fixed... No remaining defect found in this path" and reports zero new Critical findings. This corroborates the direct code/test evidence above rather than substituting for it.

**Conclusion: truth #6 flips from FAILED to VERIFIED.** The fix is exactly the two-way error discrimination the gap called for, proven by tests that exercise genuinely faulty HTTP responses (not stubs), with no regression to the rest of the phase's surface.

## Goal Achievement

### Observable Truths

Primary rows are the five ROADMAP Success Criteria (authoritative contract) plus the derived truth #6 (core value proposition + 02-01-PLAN.md must-have), consistent with the original verification's structure.

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | SilverBullet pages matching the webspace keyword by tag or page name appear interleaved with paperless documents in one chronological stream, each with an exact deep link to its page | VERIFIED (regression-checked) | Unchanged since prior verification (live `GET /api/webspaces/house-move/stream` returned 52 items sorted descending across both source types; `MatchesKeyword`/`isPage` unchanged, all `TestMatchesKeyword_*` tests still pass). No code in this path was touched by the gap-closure plan. |
| 2 | User can filter the stream down to a single source and back to all sources | VERIFIED (regression-checked) | Unchanged since prior verification. `web/src/routes/w/[webspace]/+page.svelte`, `SourceFilterChips.svelte` untouched by this gap closure; 44/44 vitest tests re-run this session, still pass. Visual click-through still not exercised in a browser this session (see Human Verification, carried forward). |
| 3 | User can see per-source health in the UI — reachable, last sync time, last error — and trigger a manual refresh that brings in new items without duplicating existing ones or stacking concurrent syncs | VERIFIED (regression-checked) | Unchanged since prior verification. `kernel/syncer/coordinator.go` untouched; `go test ./kernel/syncer/... -race` re-confirmed passing as part of this session's `make test` run. |
| 4 | An item whose source is unreachable or has been deleted shows an explicit unavailable/stale state in the stream and detail pane, never a blank pane or a silent 404 | VERIFIED (regression-checked) | Unchanged since prior verification. `detailPaneState`/`StreamRow.svelte` untouched; `staleness.test.ts` re-confirmed passing as part of this session's `npm run test`. UI pixels still not visually exercised this session (see Human Verification, carried forward). |
| 5 | Config grants agent read access and action hand-off separately, per plugin, defaulting to deny — a plugin with no grant is absent from agent-facing API responses | VERIFIED (regression-checked) | Unchanged since prior verification. `kernel/config/types.go`, `kernel/httpapi/agent.go` untouched by this gap closure; `go test ./kernel/httpapi/... -run Agent` re-confirmed passing as part of this session's `make test` run. |
| 6 | A SilverBullet sync that experiences a genuine outage or auth failure mid-Match is recorded as an error, not silently reported as a successful sync with zero items, and does not silently delete previously-synced items — the "tell at a glance whether complete and current" guarantee holds even when the failure happens *inside* a plugin's own Match call, not just when a sibling source is offline | ✓ VERIFIED (gap closed) | `plugins/silverbullet/plugin.go` lines 94-140 read directly this session: `errors.Is(err, ErrNotFound)` now the sole safe-skip; every other error propagates and fails `Match` with `codes.Unavailable`. `TestMatch_AllPageReadsFail_ReturnsUnavailable` and `TestMatch_OutageMidSync_AuthFailure_ReturnsUnavailable` both run and PASS this session (not taken from SUMMARY). `go test ./... -race` and `make test` both green. |

**Score:** 6/6 truths verified (0 present-but-behaviorally-unproven; 0 failed)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `plugins/silverbullet/plugin.go` | Match discriminates `ErrNotFound` from all other read errors | VERIFIED | Read directly this session; contains `errors.Is(err, ErrNotFound)` exactly as the gap-closure plan specified; `g.Wait()` branch now reachable |
| `plugins/silverbullet/match_test.go` | Regression coverage for the five failure/success modes | VERIFIED | 228 lines; contains `codes.Unavailable`; declares exactly the five specified `TestMatch_*` functions; ran directly this session, all PASS |
| All 19 other artifacts from the original 02-VERIFICATION.md | Unchanged | VERIFIED (regression, not re-read line-by-line) | `git diff --stat` confirms zero changes to any file outside `plugins/silverbullet/{plugin.go,match_test.go}` since the prior verification; the prior report's per-artifact findings stand |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `plugins/silverbullet/plugin.go` (Match) | `kernel/correlate/correlate.go` (SyncSource) | a Match error now becomes `WebspaceResult.Err`, causing `SyncSource` to skip `ReplaceWebspaceSourceItems` entirely for that source | VERIFIED | Confirmed by reading `SyncSource` (unchanged, already correct) plus the plan's own cited pre-existing test `TestSyncSource_PartialSourceFailure_HealthySourceItemsPersist`, still passing in `go test ./kernel/correlate/...` |
| `kernel/correlate/correlate.go` | `kernel/syncer/coordinator.go` | the first non-nil `WebspaceResult.Err` sets the sync run's status to `error` rather than `ok` | VERIFIED | Pre-existing `TestRefresh_MatchErrorReturnsErrorStatusNotGoError`, still passing |
| All other key links from the original 02-VERIFICATION.md | Unchanged | VERIFIED (regression) | No code touched in this gap closure affects any other link; `git diff --stat` confirms scope |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| SilverBullet Match: total read failure after successful listing returns `codes.Unavailable`, not a successful empty result | `cd plugins/silverbullet && go test ./... -run TestMatch_AllPageReadsFail_ReturnsUnavailable -v` (run directly this session) | PASS | PASS |
| SilverBullet Match: mid-sync auth revocation (first read 200, rest 401) returns `codes.Unavailable` | `cd plugins/silverbullet && go test ./... -run TestMatch_OutageMidSync_AuthFailure_ReturnsUnavailable -v` (run directly this session) | PASS | PASS |
| SilverBullet Match: error never leaks the bearer token | `cd plugins/silverbullet && go test ./... -run TestMatch_UnavailableError_NeverContainsBearerToken -v` (run directly this session) | PASS | PASS |
| SilverBullet Match: 404-between-listing-and-read is still a skip, not a failure (guards against over-correction) | `cd plugins/silverbullet && go test ./... -run TestMatch_PageDeletedBetweenListingAndRead_SkippedNotFailed -v` (run directly this session) | PASS | PASS |
| No regression in the rest of the silverbullet module, including under `-race` | `cd plugins/silverbullet && go test ./... -race` | 0 failures | PASS |
| Full workspace build + test | `make test` (root, sdk, paperless, silverbullet, mock) | 0 failures | PASS |
| Full workspace build artifacts | `make build` | `bin/webspaces` + 3 plugin binaries produced | PASS |
| Frontend typecheck/tests (no-regression sanity check) | `cd web && npm run check && npm run test` | 0 errors/warnings; 44/44 tests pass | PASS |
| Change-scope discipline: only the two intended files changed | `git diff --stat` (dacbd63..HEAD, non-`.planning` paths) | `plugins/silverbullet/match_test.go` (+228), `plugins/silverbullet/plugin.go` (+30/-5) only | PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|------------|-------------|--------|----------|
| SRC-05 | 02-01, 02-05 | SilverBullet plugin; tag/page matching; exact deep links; failure-mode integrity | SATISFIED | Cross-source stream (unchanged), plus Match's error-discrimination fix and regression tests (this session's direct verification) |
| KERN-04 | 02-01, 02-02 | Sync scheduler + coordinator; dedups refreshes; manual refresh | SATISFIED | Unchanged since prior verification; the CR-01 caveat that previously qualified this requirement's SATISFIED status is now resolved — the coordinator/scheduler layer was always correct, and the one plugin-level defect that could produce a false-`ok` sync run is fixed |
| PLUG-04 | 02-02 | Plugins report health (reachable, last sync, last error) | SATISFIED | Unchanged since prior verification |
| PLUG-05 | 02-04 | Third party can build a plugin from contract + mock alone | SATISFIED with a documented rigor caveat (carried forward, unresolved by this gap-closure plan by design — out of scope) | See Human Verification item #2, unchanged from prior report |
| UI-02 | 02-03 | Filter by source | SATISFIED | Unchanged since prior verification; visual click-through still not browser-verified |
| UI-05 | 02-03, 02-05 | Stale/unavailable explicit states, never silent 404; sync-run integrity | SATISFIED — the CR-01 counter-example that previously undermined this requirement is now closed | `detailPaneState` unit tests still pass; the silent-data-loss path (empty successful `MatchResponse` on outage) is now closed by the Match fix, so a genuine SilverBullet outage now surfaces as an explicit `error` sync-run status rather than a silent, undetectable deletion |
| UI-06 | 02-03 | Sync status and plugin health visible | SATISFIED | Unchanged since prior verification |
| AGENT-01 | 02-04 | Per-plugin permission model, default-deny | SATISFIED | Unchanged since prior verification |

No orphaned requirements: all 8 IDs (KERN-04, PLUG-04, PLUG-05, SRC-05, UI-02, UI-05, UI-06, AGENT-01) are exactly the union of every plan's `requirements:` frontmatter field (including 02-05's `[SRC-05, UI-05]`), and match REQUIREMENTS.md's traceability table for Phase 2 in both directions. Note: `.planning/REQUIREMENTS.md`'s traceability table and per-item checkboxes still read "Gaps Found" / unchecked for KERN-04, PLUG-04, PLUG-05, UI-02, UI-06, AGENT-01 as of this verification — this reflects that the document has not yet been updated to this verification's outcome (it was last written when the phase was at 5/6, gaps_found); it is not itself a gap in the underlying implementation, and updating that document is expected to follow this verification's `human_needed` result, not precede it.

### Anti-Patterns Found

| File | Line | Pattern | Severity | Impact |
|------|------|---------|----------|--------|
| `web/src/lib/components/DetailPane.svelte` | 33-40, 72 | Local `formatDate` reintroduces the UTC timezone bug (WR-01, still unfixed — carried forward, deliberately out of scope for the gap-closure plan) | ⚠️ Warning | Unchanged since prior verification; narrow (date display only), not core to this phase's must-haves |
| `web/src/lib/components/DetailPane.svelte` | 143, 149 | `<iframe>` elements carry no `sandbox` attribute despite a code comment claiming otherwise (WR-02, still unfixed — carried forward) | ⚠️ Warning | Unchanged; not currently exploitable (CSP header already sandboxes), latent hazard only |
| `kernel/httpapi/stream.go`, `webspaces.go`, `agent.go` | various | `LatestSyncRunPerSource` errors silently downgraded to "unknown" sync state (WR-03, still unfixed — carried forward) | ⚠️ Warning | Unchanged; low likelihood, narrow blast radius |
| `cmd/webspaces/main.go` | `runServe` | No OS signal handling for graceful shutdown (WR-04, still unfixed — carried forward) | ⚠️ Warning | Unchanged; operational hygiene, not a phase-goal blocker |
| `web/src/routes/w/[webspace]/+page.svelte` | — | New: polling interval not cleared on component destroy (WR-05, newly identified by the independent re-review pass, `02-REVIEW.md`) | ⚠️ Warning | Not a phase-goal blocker; a resource-cleanup hygiene issue flagged by the code-review agent, not by this verification's own artifact checks. Recorded here for completeness since it surfaced in the same re-review cycle; does not affect any of the 6 truths above. |

No `TBD`/`FIXME`/`XXX` debt markers found in `plugins/silverbullet/plugin.go` or `plugins/silverbullet/match_test.go` (the two files this gap-closure plan touched).

The BLOCKER anti-pattern from the prior verification (`plugins/silverbullet/plugin.go:94-120`, silent error-swallowing) is **resolved** — confirmed by direct code read this session, not carried forward.

## Human Verification Required

Both items below are carried forward unchanged from the prior `02-VERIFICATION.md`. Neither was in scope for the `02-05` gap-closure plan (explicitly excluded in its `<scope_notes>`), and neither has been newly exercised this session — they remain open, non-blocking human-verification items.

### 1. Visual confirmation of health chips, filter chips, and staleness states in a real browser

**Test:** With both sources configured, deliberately stop the SilverBullet service and reload the webspace in a browser. Confirm: two health chips render with correct colors (paperless green, SilverBullet red); hovering SilverBullet's chip shows the unreachable tooltip with a relative time and the full untruncated error; SilverBullet rows carry an amber stale marker while paperless rows are visually unchanged; opening a SilverBullet item shows its cached title/date/labels/open-in-source link with the unreachable alert layered over the cached preview (never a blank pane); clicking the source filter chips narrows/restores the list and the URL gains/loses the `source` query parameter, surviving a reload.

**Expected:** Matches `02-UI-SPEC.md`'s Copywriting Contract and Design System exactly (colors, wording, layout).

**Why human:** No browser tooling was available in the executing sessions for plans 02-02/02-03/02-04, nor in this re-verification session. Proven correct at the pure-decision-function/unit-test layer (44/44 vitest tests re-confirmed passing this session) and at the live API layer (prior session's curl checks), but the actual rendered pixels, tooltip wrapping, and click interactions remain unconfirmed visually.

### 2. PLUG-05's third-party self-sufficiency claim, from a genuinely fresh/isolated context

**Test:** Have a truly isolated agent (a real dispatched subagent, or a human with no prior exposure to this repository) attempt to build a `SourcePlugin` using only `proto/webspaces/v1/plugin.proto`, `docs/plugin-contract.md`, the `sdk` module, and `plugins/mock` — with no access to `plugins/paperless` or `plugins/silverbullet` and no memory of them.

**Expected:** A clean build with zero or few gaps, corroborating the two-gap-then-zero-gap result 02-04's in-session exercise already found.

**Why human:** 02-04-SUMMARY.md honestly documents that no subagent-dispatch tool was available in that execution environment, so the "isolation" exercise was performed by an executor with prior knowledge of the real plugins — a materially weaker approximation than the plan intended. This gap-closure plan (02-05) did not touch `plugins/mock`, `docs/plugin-contract.md`, or the isolation exercise, so this item is unchanged and remains open.

## Gaps Summary

No gaps remain. The single BLOCKER from the prior verification — `plugins/silverbullet/plugin.go`'s `Match` treating every per-page read failure identically to "page not found," which could silently delete every previously-synced SilverBullet item on a genuine outage while recording the sync as `ok` — is closed, verified this session by direct code reading and by running (not merely trusting) the named regression tests (`TestMatch_AllPageReadsFail_ReturnsUnavailable`, `TestMatch_OutageMidSync_AuthFailure_ReturnsUnavailable`, `TestMatch_UnavailableError_NeverContainsBearerToken`, plus the two green-both-ways guard tests). `git diff --stat` confirms the fix's blast radius was exactly the two files the gap-closure plan declared, so the other five truths carry zero regression risk and are re-confirmed via the still-passing full test suite (`make test`, `-race`, `npm run test`).

Two non-blocking Human Verification items remain outstanding, carried forward unchanged: a real-browser visual confirmation of the health/filter/staleness UI surfaces, and a genuinely isolated (fresh-context) re-run of the PLUG-05 third-party-plugin exercise. Four (now five, per the independent re-review's new WR-05 finding) non-blocking warnings remain open for a future fix pass: WR-01 (DetailPane timezone formatter), WR-02 (missing iframe `sandbox` attribute), WR-03 (silently-downgraded sync-run lookup errors), WR-04 (no graceful-shutdown signal handling), and WR-05 (uncleared polling interval on component destroy). None of these block the phase goal.

---

_Verified: 2026-07-29T10:45:00Z_
_Verifier: Claude (gsd-verifier)_
