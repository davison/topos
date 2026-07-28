---
phase: 01-first-webspace-end-to-end
verified: 2026-07-28T13:35:00Z
status: human_needed
score: 46/50 must-haves verified
behavior_unverified: 4
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: 62/64
  gaps_closed:
    - "G-01-2: SPA shipped zero CSS (missing `import '../app.css'` in +layout.svelte) — fixed, rebuilt, and confirmed live: production build emits a 33,334-byte CSS asset containing all required selectors/tokens, 200.html links it, and the currently-running server (started by 01-05's handoff, still up) serves it at /w/house-move with a 35-item populated stream."
    - "G-01-6: outbound-host allowlist was flagged as an unwired test-tier prohibition — now closed: plugins/paperless/client.go refuses foreign hosts at both CheckRedirect and DialContext (ErrForeignHost sentinel), 5 committed runtime tests pass, a repo-wide AST audit (internal/audit) fails the build on any foreign URL literal or unsanctioned outbound HTTP construction, proven non-vacuous by a negative-control fixture, and 01-01-PLAN.md's third prohibition now names both enforcing test files."
  gaps_remaining: []
  regressions: []
---

# Phase 01: First Webspace, End to End — Re-Verification Report

**Phase Goal:** User can define a webspace, open it in a browser, and see their paperless-ngx documents for that topic as one chronological stream with inline previews and working deep links back to paperless-ngx.

**Verified:** 2026-07-28T13:35:00Z
**Status:** human_needed
**Re-verification:** Yes — after gap-closure plans 01-05 (G-01-2) and 01-06 (G-01-6)

## Goal Achievement

This is a re-verification following two gap-closure plans executed against the initial verification's findings and a UAT round (2 passed, 4 issues — see `01-UAT.md`). Rather than trusting `01-05-SUMMARY.md`/`01-06-SUMMARY.md` claims, this run independently re-executed the evidence: a fresh `npm --prefix web run build`, inspection of the emitted CSS and `200.html`, live `curl` against the still-running handoff server from plan 01-05 (PID confirmed via `ps`, started 13:00, serving 35 items with a linked, token-bearing stylesheet), `npm run check` (0 errors), `npm run test -- --run` (5/5 pass), `cd plugins/paperless && go test ./... -count=1` plus the five named allowlist tests individually, `go test ./internal/audit/ -count=1 -v` (both tests including the non-vacuous negative control), and a full `make test` across all three Go modules (root, sdk, plugins/paperless) — all green.

### Observable Truths

Truths 1–39 are carried forward from the initial 01-VERIFICATION.md (2026-07-28T02:30:00Z); their supporting artifacts were re-spot-checked and no regression was found. Truths 40–50 are new, sourced from 01-05-PLAN.md and 01-06-PLAN.md `must_haves.truths`.

| # | Truth (source) | Status | Evidence |
|---|---|---|---|
| 1 | A webspace defined in `config.toml` is browsable at `/w/{name}` (Roadmap SC) | ✓ VERIFIED | Live: `curl /w/house-move` → 200 with populated, styled HTML (re-confirmed this run) |
| 2 | Documents shown are the user's own paperless-ngx docs, keyword-matched, in total stable chronological order (Roadmap SC) | ✓ VERIFIED | Unchanged since initial verification; `TestStreamItems_TotalOrderingWithTieBreak` passes |
| 3 | Every item carries a working `{PAPERLESS_URL}/documents/{id}` deep link declared `exact` fidelity (Roadmap SC) | ✓ VERIFIED | Unchanged; smoke-test assertion re-read and confirmed present |
| 4 | Stream renders from the local SQLite index with no paperless-ngx request in the request path (Roadmap SC) | ✓ VERIFIED | Unchanged; `stream.go` re-read, no `pluginhost` import |
| 5 | Paperless plugin runs as a separate subprocess over a gRPC contract declaring no mutating RPC (Roadmap SC) | ✓ VERIFIED | Unchanged; live `ps` still shows subprocess relationship |
| 6 | Same JSON the SPA consumes is fetchable at `/api/*` with stable IDs and machine-readable provenance (Roadmap SC) | ✓ VERIFIED | Unchanged; `contract_test.go` passes |
| 7 | User defines webspace, starts `webspaces serve`, opens in browser (01-01) | ✓ VERIFIED | Unchanged |
| 8 | Keyword-tag correlation resolved at sync time (KERN-02) (01-01) | ✓ VERIFIED | Unchanged |
| 9 | KERN-01/adjacency (shared keyword, no collision) | ✓ VERIFIED | `TestReplaceWebspaceItems_TwoWebspacesShareItemNoCollision` passes |
| 10 | KERN-01/empty (zero/whitespace keyword fails load) | ✓ VERIFIED | Unchanged |
| 11 | KERN-01/encoding (`EqualFold` only) | ✓ VERIFIED | Unchanged |
| 12 | KERN-01/ordering (keyword order irrelevant) | ✓ VERIFIED | Unchanged |
| 13 | KERN-03/adjacency (500-rune preview cap) | ✓ VERIFIED | Unchanged |
| 14 | KERN-03/ordering (stable 3-key sort) | ✓ VERIFIED | `TestStreamItems_TiesOnBothTimestampsBreakByIDAscending` passes |
| 15 | KERN-03/idempotency | ✓ VERIFIED | Unchanged |
| 16 | KERN-03/concurrency (one tx per sync) | ✓ VERIFIED | Unchanged |
| 17 | SRC-04/concurrency (4 in-flight cap) | ✓ VERIFIED | `plugins/paperless/client.go:53` re-read, `MaxConnsPerHost: 4` still present alongside new allowlist code |
| 18 | AGENT-02/adjacency (collision-free composite IDs) | ✓ VERIFIED | Unchanged |
| 19 | AGENT-02/empty (empty webspace → 200 `[]`) | ✓ VERIFIED | Unchanged |
| 20 | AGENT-02/ordering (byte-identical repeat) | ✓ VERIFIED | Unchanged |
| 21 | AGENT-02/concurrency: stream GET during in-flight sync returns a consistent set | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | Unchanged — still `verification: backstop` in the plan, no committed test exercises the race |
| 22 | Clicking a stream item opens a detail pane rendering instantly from local metadata (01-02) | ✓ VERIFIED | `DetailPane.svelte` unchanged, two-stage render confirmed by code read |
| 23 | Full content fetched live on item-open, never persisted (KERN-03) (01-02) | ✓ VERIFIED | Unchanged |
| 24 | Every item carries "Open in paperless-ngx" CTA at exact fidelity (PLUG-03/UI-04) (01-02) | ✓ VERIFIED | Unchanged |
| 25 | `/content` MIME allowlist + `nosniff` + sandbox CSP (01-02) | ✓ VERIFIED | Unchanged |
| 26 | `/api/items/{id}` accepts raw and percent-encoded id identically (01-02) | ✓ VERIFIED | Unchanged |
| 27 | KERN-03/empty, PLUG-03/adjacency-empty-ordering, UI-04/adjacency-empty-ordering (6 truths) | ✓ VERIFIED | Unchanged |
| 28 | UI-SPEC detail-pane loading/error/populated/empty/overflow considerations (8 truths) | ✓ VERIFIED | Component code unchanged and correct; **actual browser rendering was broken by G-01-2 until this gap-closure round** — see Human Verification |
| 29 | Stream row renders 40x52 thumbnail, title, date, tag pills, clamped preview (01-03) | ✓ VERIFIED | `StreamRow.svelte` unchanged and correct; CSS delivering these classes is now confirmed shipped (`.line-clamp-2`, `.truncate`, `.stream-row-surface` all present in emitted CSS) — full visual confirmation still pending human re-run (see below) |
| 30 | Stream/detail panes independently scrolled, no horizontal overflow (01-03) | ✓ VERIFIED | `overflow-y-auto overflow-x-hidden min-h-0` markup unchanged; layout CSS now confirmed shipped (`display:flex` present in emitted stylesheet) |
| 31 | Every stream state (empty/loading/error/populated/partial) renders explicitly (01-03) | ✓ VERIFIED | `StreamList.svelte` unchanged, five-branch state machine intact |
| 32 | UI-03/adjacency-empty-ordering + UI-SPEC stream-list considerations (12 truths) | ✓ VERIFIED | Unchanged |
| 33 | UI/error/stream-list (backstop): failed fetch shows approved error copy + retry | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | Unchanged — still no automated test exercises a genuinely-down kernel |
| 34 | plugin.proto RPC set mechanically pinned as an allowlist (PLUG-02) (01-04) | ✓ VERIFIED | Unchanged |
| 35 | Go AST scan proves no non-GET HTTP request under `plugins/` (PLUG-02) (01-04) | ✓ VERIFIED | `TestPluginsIssueOnlyGetRequests` re-run, passes |
| 36 | Agent-facing JSON envelope pinned by committed test (AGENT-02) (01-04) | ✓ VERIFIED | Unchanged |
| 37 | docs/plugin-contract.md and docs/api.md published, cross-checked (01-04) | ✓ VERIFIED | Unchanged |
| 38 | config.example.toml documents every key (01-04) | ✓ VERIFIED | Unchanged |
| 39 | README.md documents full local run (01-04) | ✓ VERIFIED | Unchanged; also independently confirmed by user's own pass on UAT test 5 |
| 40 | (G-01-2) Production SPA build emits ≥1 CSS asset and served HTML links it (01-05) | ✓ VERIFIED | Fresh `npm run build` this run: `kernel/webui/build/_app/immutable/assets/0.CaDOf_tl.css` (33,334 bytes); `200.html` contains `rel="stylesheet"`; all six required selectors/tokens present (`.line-clamp-2`, `.truncate`, `.stream-row-surface`, `.stream-row-meta`, `#020617`, `display:flex`) |
| 41 | (G-01-2) Stream rows render as designed fixed-height cards, not UA-default markup (01-05) | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | CSS now provably ships and contains the right selectors, but actual browser rendering fidelity (row height, thumbnail sizing, ellipsis) is a visual fact only a human can confirm — this is exactly UAT test 3, previously failed, now pending re-run |
| 42 | (G-01-2) Detail pane renders beside the stream (two-pane), never stacked below with page-level scroll (01-05) | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | Same reasoning as #41 — this is UAT test 2, previously failed on the exact "stacked below" symptom now attributable to the CSS bug; pending re-run |
| 43 | (G-01-2) Committed assertion fails the smoke test if a future build ships stylesheet-less/tokenless HTML (01-05) | ✓ VERIFIED | `scripts/e2e-smoke.sh` re-read: three-part stylesheet assertion (link exists → fetches non-empty → contains `#020617`) present at lines 58-84; negative control re-run this session confirms the matcher correctly rejects HTML with no `rel="stylesheet"` link |
| 44 | (G-01-2) Smoke test refuses a stale listener on 127.0.0.1:7777 | ✓ VERIFIED | Pre-check at lines 35-39 re-read: `curl`s `/api/webspaces` before starting its own server, exits 1 with a named FAIL if already answered |
| 45 | (G-01-6) Paperless HTTP client refuses to dial/redirect to any host that is neither the configured base host nor loopback, before any bytes leave the process (01-06) | ✓ VERIFIED | `client.go` re-read: `allowHost` predicate wired into both `Transport.DialContext` and `CheckRedirect`; `ErrForeignHost` sentinel confirmed |
| 46 | (G-01-6) Committed test proves cross-host redirect refused via sentinel; same-host redirect still followed (01-06) | ✓ VERIFIED | `TestDocument_CrossHostRedirect_Refused` and `TestDocument_SameHostRedirect_StillFollowed` both re-run individually this session — pass |
| 47 | (G-01-6) Committed test proves foreign pagination `next` re-pinned to base host (01-06) | ✓ VERIFIED | `TestListDocuments_ForeignNextURL_RepinnedToBaseHost` re-run — pass |
| 48 | (G-01-6) Repo-wide committed test fails the build on foreign URL literals or outbound HTTP construction outside `plugins/paperless/client.go` (01-06) | ✓ VERIFIED | `TestNoForeignEgressOutsideSanctionedClient` re-run — pass; `make test` green across all 3 modules including `internal/audit` |
| 49 | (G-01-6) Audit test proven non-vacuous by a fixture with both offense kinds (01-06) | ✓ VERIFIED | `internal/audit/testdata/foreign_host_violation.go.txt` inspected — contains one foreign URL literal + one `http.Get` call; `TestScanner_FixtureReportsBothOffenseKinds` re-run — pass |
| 50 | (G-01-6) 01-01-PLAN.md's outbound-transmission prohibition names the enforcing tests, making its `verification: test` claim traceable (01-06) | ✓ VERIFIED | `01-01-PLAN.md` re-read: third prohibition now carries `verified_by: [plugins/paperless/outbound_hosts_test.go, internal/audit/outbound_hosts_test.go]` plus a closure note; other two prohibitions and frontmatter byte-identical |

**Score:** 46/50 truths verified (4 present-but-behavior-unverified: 2 carried-forward backstops unaffected by this gap-closure round, plus 2 new items — #41/#42 — that are the direct subject of the pending UAT re-run of tests 2–4).

### Deferred Items

None — no gaps map to later milestone phases.

### Required Artifacts

All artifacts from the initial verification remain present, substantive, and wired (re-spot-checked, no regressions). New artifacts from the two gap-closure plans:

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `web/src/routes/+layout.svelte` | `import '../app.css'` | ✓ VERIFIED | Present as first statement in `<script>` block; all other content byte-identical to before |
| `scripts/e2e-smoke.sh` | stale-listener pre-check + stylesheet assertion | ✓ VERIFIED | Both edits present; `bash -n` parses clean |
| `kernel/webui/build/_app/immutable/assets/*.css` | emitted stylesheet (gitignored, proof artifact) | ✓ VERIFIED | 33,334 bytes; contains all required selectors/tokens (re-generated and re-checked this session) |
| `plugins/paperless/client.go` | `ErrForeignHost` sentinel + `allowHost` predicate | ✓ VERIFIED | 150+ lines added; wired into `DialContext` and `CheckRedirect` |
| `plugins/paperless/outbound_hosts_test.go` | predicate table, redirect/next-URL tests | ✓ VERIFIED | 5 named tests, all pass |
| `internal/audit/outbound_hosts_test.go` | repo-wide AST scan | ✓ VERIFIED | `go/parser`-based; 2 tests pass including non-vacuous negative control |
| `internal/audit/doc.go` | package clause for test-only dir | ✓ VERIFIED | `package audit`; `go build ./...` succeeds in root module |
| `internal/audit/testdata/foreign_host_violation.go.txt` | negative-control fixture | ✓ VERIFIED | Contains 1 foreign URL literal + 1 `http.Get` construction |

### Key Link Verification

All links from the initial verification remain wired (re-spot-checked). New links from the two gap-closure plans:

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `web/src/routes/+layout.svelte` | `web/src/app.css` | ES module import | ✓ WIRED | Import present; build graph now includes app.css (confirmed by non-empty emitted CSS) |
| `kernel/webui/build/200.html` | `kernel/webui/build/_app/immutable/assets/*.css` | `<link rel="stylesheet">` | ✓ WIRED | Confirmed via fresh build and via live `curl` against the running server |
| `scripts/e2e-smoke.sh` | running kernel binary | curl of SPA fallback + stylesheet URL | ✓ WIRED | Script logic re-read; matches described 3-part assertion; negative control re-run confirms non-vacuity |
| `plugins/paperless/client.go` `allowHost` | `http.Client.CheckRedirect` / `http.Transport.DialContext` | shared predicate on both hooks | ✓ WIRED | Both call sites confirmed by code read at lines ~95 and ~116 |
| `internal/audit/outbound_hosts_test.go` | every non-test `.go` file in the repo | `filepath.WalkDir` from repo root | ✓ WIRED | Confirmed passing against the live tree via `make test` |

### Data-Flow Trace (Level 4)

The CSS delivery path was the specific failure mode of G-01-2 (artifact present, wired via SvelteKit's routing, but the actual data — CSS rules — never reached the browser because the module graph excluded app.css). This session traced it end to end: `app.css` → imported by `+layout.svelte` → picked up by `@tailwindcss/vite` → emitted as `_app/immutable/assets/0.CaDOf_tl.css` → linked from `200.html` → served by the currently-running kernel at `/w/house-move` → fetched live via `curl` and confirmed to contain the `#020617` token and component-specific classes. Data flows end to end; only the final "does it *look* right" step remains a human judgment (see Human Verification).

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|---|---|---|---|
| Production build emits real CSS | `npm --prefix web run build` + selector grep | 33,334-byte CSS, all 6 selectors present | ✓ PASS |
| `svelte-check` clean | `npm run check` | 731 files, 0 errors, 0 warnings | ✓ PASS |
| Frontend unit tests | `npm run test -- --run` | 5/5 pass | ✓ PASS |
| Paperless module full suite | `go test ./... -count=1` (plugins/paperless) | ok | ✓ PASS |
| Named allowlist tests | `go test -run "Host\|Redirect\|Foreign\|Next" -v` | 5/5 `--- PASS` | ✓ PASS |
| Audit package | `go test ./internal/audit/ -count=1 -v` | 2/2 pass incl. non-vacuous negative control | ✓ PASS |
| Full workspace test | `make test` | ok across root, sdk, plugins/paperless | ✓ PASS |
| Live server serving styled build | `curl /w/house-move` + stylesheet fetch + `#020617` grep | stylesheet linked, contains token, stream has 35 items | ✓ PASS |
| Smoke-test stylesheet matcher non-vacuity | manual negative control against stylesheet-less HTML | matcher correctly rejects | ✓ PASS |

### Probe Execution

No `scripts/*/tests/probe-*.sh` convention exists in this repo; `scripts/e2e-smoke.sh` is the phase's declared end-to-end check but requires live `PAPERLESS_URL`/`PAPERLESS_TOKEN` credentials this verifier does not have direct read access to (`.env` is permission-restricted). Its logic was instead verified by static re-read (matches the plan's described 3-part assertion and stale-listener pre-check) plus live equivalent checks against the already-running handoff server from plan 01-05, which exercises the identical code path (`webspaces serve` + `curl` of `/`, the stylesheet, and `/api/webspaces/house-move/stream`). SKIPPED (full script run) — no credential access; PASS (equivalent live checks performed manually).

### Requirements Coverage

All 11 requirement IDs assigned to Phase 1 remain accounted for, cross-checked against `REQUIREMENTS.md` (all rows marked `Phase 1 | Complete`, no orphans):

| Requirement | Plan(s) | Status | Evidence |
|---|---|---|---|
| KERN-01 | 01-01, 01-04 | ✓ SATISFIED | Unchanged from initial verification |
| KERN-02 | 01-01 | ✓ SATISFIED | Unchanged |
| KERN-03 | 01-01, 01-02 | ✓ SATISFIED | Unchanged |
| PLUG-01 | 01-01 | ✓ SATISFIED | Unchanged |
| PLUG-02 | 01-01, 01-04, 01-06 | ✓ SATISFIED | Now additionally backed by the outbound-host allowlist (client + audit tests) |
| PLUG-03 | 01-02 | ✓ SATISFIED | Unchanged |
| SRC-04 | 01-01, 01-06 | ✓ SATISFIED | Now additionally backed by host-pinned redirect/dial policy |
| UI-01 | 01-01, 01-03, 01-05 | ✓ SATISFIED | Now additionally backed by the restored stylesheet delivery |
| UI-03 | 01-02, 01-03, 01-05 | ✓ SATISFIED | Same |
| UI-04 | 01-02, 01-05 | ✓ SATISFIED | Same |
| AGENT-02 | 01-01, 01-04 | ✓ SATISFIED | Unchanged |

### Anti-Patterns Found

No debt markers (`TBD`/`FIXME`/`XXX`), `TODO`/`HACK`/`PLACEHOLDER` comments, or "not yet implemented" strings found in any file modified by the two gap-closure plans (`web/src/routes/+layout.svelte`, `scripts/e2e-smoke.sh`, `plugins/paperless/client.go`, `plugins/paperless/outbound_hosts_test.go`, `internal/audit/*`). (One grep hit on `scripts/e2e-smoke.sh` for `mktemp ... .XXXXXX` is a shell mktemp template placeholder, not a debt marker — confirmed by inspection.)

`01-REVIEW.md` was updated post-gap-closure (commit `2842b0d`, re-reviewed 46 files including all gap-closure artifacts): **0 critical, 10 warnings, 8 info**. No new critical issues. The one warning carried forward from the initial verification as worth flagging:

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `web/src/lib/components/DetailPane.svelte` | 18-25 | Local `formatDate()` uses `toLocaleDateString` with no `timeZone`, reintroducing the UTC-boundary bug `format.ts`'s `formatItemDate` was built to prevent — **confirmed still present this session** | ⚠️ WARNING | Same as previously documented; not tied to any phase must-have, does not block this verification |

Two new info-level items from the post-gap-closure review are worth noting for backlog triage (neither blocks a phase must-have): `IN-07` — `readonly_test.go`'s AST walk doesn't exclude `_test.go` files, inconsistent with the new `internal/audit` scanner's policy; `IN-06` — the landing page's webspace link doesn't `encodeURIComponent` the name. Full detail in `01-REVIEW.md`.

### Human Verification Required

3 items remain (2 fewer than the initial verification: UAT tests 1 and 5 already passed and are not re-listed; the outbound-host prohibition, previously item 6, is now closed by committed tests and removed).

1. **Re-run UAT tests 2, 3, and 4 in the browser now that G-01-2 is fixed**
   **Test:** Hard-reload (Ctrl+Shift+R) `http://127.0.0.1:7777/` (already running, styled build confirmed live this session). Click a document row to open the detail pane; scroll the PDF/extracted text while watching the stream list; click "Open in paperless-ngx"; open a webspace row with several tags and a long title; check a no-OCR-text document; scroll the stream with the detail pane open; try an 80+ character webspace name; point `base_url` at an unreachable host and confirm the sync-error state.
   **Expected:** Per `01-05-PLAN.md` Task 3's human-check block — detail pane sits BESIDE the stream (never stacked below), title/date/tags appear instantly with a skeleton-then-fill preview, scroll containment is per-region, stream rows are fixed-height (152px) small-thumbnail cards with ellipsised titles and two-line clamped snippets, "Nothing here yet" / truncated-name-tooltip / sync-error states render correctly, and specifically the three original UAT symptoms (full-size centered images, unformatted stacked text, whole-page scrolling) are gone.
   **Why human:** Visual rendering fidelity, scroll containment, and layout correctness in a real browser cannot be asserted by grep or curl — this is exactly what UAT tests 2/3/4 previously caught as broken (correctly, since the CSS bug was real) and must be re-confirmed now that the fix is shipped.

2. **AGENT-02/concurrency backstop** (carried forward, unaffected by this gap-closure round)
   **Test:** Issue `GET /api/webspaces/house-move/stream` repeatedly in a tight loop while `webspaces sync` runs concurrently; diff each response against the two known-good pre/post item sets.
   **Expected:** Every response matches either the pre-sync or post-sync set exactly, never a partial mix.
   **Why human:** Plan-classified `verification: backstop` — SQLite WAL-mode snapshot isolation is relied on by construction; no committed test exercises the live race.

3. **UI/error/stream-list backstop** (carried forward, unaffected by this gap-closure round)
   **Test:** Stop `webspaces serve` (or block port 7777) and load a webspace route in the browser; confirm "Couldn't load this webspace" copy and a working retry control appear.
   **Expected:** Approved error copy + working retry that recovers the stream once the kernel is back.
   **Why human:** Plan-classified backstop requiring a genuinely-down kernel; `StreamError.svelte` contains the exact copy (grep-confirmed, unchanged) but no automated test exercises a live connection-refused condition.

### Gaps Summary

No BLOCKER gaps. Both gaps from the prior UAT round are closed with independently re-verified evidence:

- **G-01-2 (major, CSS/styling):** Root cause (`+layout.svelte` missing `import '../app.css'`) fixed at its single source, confirmed via a fresh build this session producing a complete, correctly-tokened stylesheet, and confirmed live against the still-running handoff server. A three-part recurrence guard now lives in `scripts/e2e-smoke.sh`, re-read and its negative control independently re-confirmed. The remaining piece — actual visual confirmation in a browser — is exactly what UAT tests 2/3/4 exist to check and is queued as human-verification item 1 above, not a gap.
- **G-01-6 (minor, outbound-host prohibition):** Now enforced by construction (host-pinned `CheckRedirect`/`DialContext`) and asserted by 7 committed, independently-re-run-and-passing tests across two files, plus a traceability amendment to `01-01-PLAN.md`. Fully closed — no further action needed.

Two pre-existing backstop items (AGENT-02/concurrency, stream-list error-path) remain present-but-behavior-unverified exactly as in the initial verification — both were always plan-classified `verification: backstop` and are unaffected by this gap-closure round.

The route to `passed` status is entirely the pending human re-confirmation of UAT tests 2–4 (server already running, ready for the user) plus the two long-standing backstop items — no further engineering work is indicated.
