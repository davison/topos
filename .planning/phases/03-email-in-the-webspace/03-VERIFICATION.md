---
phase: 03-email-in-the-webspace
verified: 2026-08-01T01:15:00Z
status: human_needed
score: 4/5 must-haves verified
behavior_unverified: 1
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: 4/5
  gaps_closed:
    - "UAT gap G-03-1's two CODE-side missing items are closed: (1) plugins/proton/credentials.go's bridgeTokenShapeWarning now flags a configured Bridge token containing any byte outside base64url's A-Za-z0-9-_ alphabet as 'not a Bridge-generated app password', wired once at client.go connect()'s LOGIN-failure branch so the advice reaches both HealthResponse.LastError (Health path) and, via Match -> kernel/correlate -> sync_runs -> kernel/httpapi/sources.go's sourceStatusesFrom (LastError: run.Error), the UI's red-dot last_error — confirmed by reading both wiring arms directly and by re-running plugins/proton/credentials_test.go's four new tests (TestBridgeTokenShapeWarning_AlphabetBoundary's 10-row alphabet-boundary table plus three Health-path tests) myself, all PASS; (2) plugins/proton/live_bridge_test.go's LOGIN-failure hint no longer cites 03-01-SUMMARY.md or asserts a settled username-side cause ('not a code defect') — it now references the same bridgeAuthOrderNote constant the runtime advice is built from, confirmed absent by direct grep (`grep -q 'not a code defect'` and `grep -q '03-01-SUMMARY'` both exit 1) and present exactly once (`grep -c bridgeAuthOrderNote` = 1)."
  gaps_remaining:
    - "UAT gap G-03-1's root, user-side item is NOT closed and cannot be closed by any plan: .env's PROTON_BRIDGE_PASS still holds a non-Bridge-shaped 37-character symbol-heavy string (root-caused in .planning/debug/proton-bridge-no-such-user.md as ENVIRONMENTAL, code path audited clean). Until the user replaces it with the real Bridge app password, live LOGIN against the real Bridge account still fails, and UAT Tests 1-3 remain blocked. This is unchanged from the previous verification round and is carried forward as a human-verification item below, not as a code gap."
  regressions: []
gaps: []
human_verification:
  - test: "On monroe, read the real Proton Mail Bridge app password (Bridge GUI -> account -> Mailbox details, or `protonmail-bridge --cli` then `info`; sign in first if signed out) and replace PROTON_BRIDGE_PASS in .env with it (~20-22 chars, alphabet A-Za-z0-9-_ only, never the Proton account password). Wait out Bridge's login jail if 'too many login attempts' was recently hit. Restart the kernel and open a webspace with a real, currently-reachable Proton Bridge account configured. Confirm at least one real Proton email appears in the stream interleaved with paperless/SilverBullet items, sender before the date, subject as the row title; click it and confirm the detail pane renders the body. If more than one configured webspace matches Proton mail, additionally confirm the SAME check holds for a webspace that is NOT the last one synced in that cycle."
    expected: "The email is visible, correctly dated, and its body renders in the detail pane for every configured webspace that matches Proton mail — not just the one processed last in a sync cycle. This is the direct live confirmation of the mailboxCache fix (03-06) plus the CVE-remediated rendering path (03-07), now unblockable in principle since 03-08's diagnostic would have named the credential-shape defect immediately instead of costing four rounds."
    why_human: "Requires a running kernel/webUI against a real, currently-authenticating Proton Mail Bridge account and visual confirmation of rendered HTML/CSS in a browser — not mechanically checkable from source alone. Still blocked on the .env correction, which is a user action outside code/repo scope (root-caused ENVIRONMENTAL in .planning/debug/proton-bridge-no-such-user.md; code path audited clean)."
  - test: "Once the .env credential is corrected, run `WEBSPACES_PROTON_LIVE_IT=1 PROTON_BRIDGE_ADDR=<addr> PROTON_BRIDGE_USER=<user> PROTON_BRIDGE_PASS=<pass> go test -run TestSeenFlagUnchanged_LiveBridge -v ./plugins/proton/...` against the real Bridge account."
    expected: "The test passes, directly proving SRC-01's second success criterion (\\Seen flag unchanged across a full Match+Fetch cycle) against the real mailbox, not just the no-live-Bridge wire-transcript proxy for it (TestIMAPTranscript_ExamineAndPeekOnly / TestPluginIssuesNoIMAPMutatingCommands, both re-confirmed passing this session)."
    why_human: "Requires a live, currently-authenticating Bridge connection and real mailbox state; the test is implemented and skips cleanly (re-confirmed this session: SKIP, not FAIL) but has never run to a PASS — blocked on the same .env correction as above, unchanged."
  - test: "After running a webspaces sync and opening an email in the detail pane, check the same email in the real Proton web or mobile client and confirm it is still shown as unread there."
    expected: "The email remains unread in Proton's own client — the direct, human-observable proof of the never-mark-read guarantee end to end."
    why_human: "Requires a live Bridge account, a real sync, and cross-checking read/unread state in Proton's own UI; unchanged blocker."
  - test: "Re-run 03-UAT.md's Test 1 scenario after the .env correction and confirm the red-dot detail (if any error remains) no longer misattributes the cause to the username."
    expected: "If the corrected credential still fails for any reason, the surfaced error is now the server's own message plus (only if the new token is STILL shape-suspect) the actionable bridgeTokenShapeWarningText advice — never the old, removed 'not a code defect' / 03-01-SUMMARY.md framing, since that text no longer exists in the codebase (confirmed this session)."
    why_human: "Requires a live Bridge account and observing the actual red-dot text rendered in the browser after a real (possibly still-failing) sync attempt; the code-level guarantee (advice text is correct and reaches both surfaces) is verified below, but its rendered appearance in the running UI is not."
behavior_unverified_items:
  - truth: "Reading an email inside webspaces never marks it read in Proton — proven by an automated test asserting the \\Seen flag is unchanged after a full sync and a detail fetch (ROADMAP SC2)"
    test: "Run TestSeenFlagUnchanged_LiveBridge against a real, currently-authenticating Proton Mail Bridge account."
    expected: "\\Seen flag on the fetched message is unchanged after a full Match + Fetch (detail-open) cycle, asserted directly against the real mailbox rather than inferred from an absence of mutating IMAP commands in a wire transcript."
    why_human: "Code-level proof exists (TestPluginIssuesNoIMAPMutatingCommands' AST scan forbidding IMAP-mutating identifiers; TestIMAPTranscript_ExamineAndPeekOnly's wire-transcript assertion that only EXAMINE/BODY.PEEK[ are issued) and both re-confirmed passing this session, but no test has ever run to PASS against a live mailbox's actual flag state — the live-Bridge test exists and skips cleanly rather than failing, blocked by the unchanged .env credential-shape issue (now diagnosable in one round instead of four, per 03-08, but not yet corrected)."
---

# Phase 3: Email in the Webspace Verification Report

**Phase Goal:** User's Proton mail for a topic appears in the webspace stream and stays readable there, and the volume it brings is navigable by searching within the webspace
**Verified:** 2026-08-01T01:15:00Z
**Status:** human_needed
**Re-verification:** Yes — after gap-closure plan 03-08 (the two code-side items of UAT gap G-03-1)

**Note on MVP mode:** ROADMAP.md marks this phase `Mode: mvp`, but its `Goal:` line is not phrased as a User Story (`As a … I want to … so that …`). Consistent with all five prior verification rounds for this phase, this report uses standard goal-backward verification rather than the MVP User Flow Coverage format.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Emails in Proton folders/labels matching the webspace keyword appear in the stream with sender, subject, date, and render body inline in the detail pane (SC1) | ✓ VERIFIED | Unchanged since the last round: `mailboxCache` accumulation (03-06) and `DetailPane` date formatting (03-06), plus the CVE-remediated rendering path (03-07), all re-confirmed passing this session (`plugins/proton` full suite green; `render_test.go`'s sanitize/wrap tests including the two boundary cases pass). The live, in-browser confirmation with a real Bridge account remains a human-verification item below, now blocked purely on the `.env` credential correction rather than on any unresolved code defect. |
| 2 | Reading an email inside webspaces never marks it read in Proton — proven by an automated test asserting `\Seen` unchanged after a full sync and a detail fetch (SC2) | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | `TestPluginIssuesNoIMAPMutatingCommands` and `TestIMAPTranscript_ExamineAndPeekOnly` both re-run and pass this session — code-level proof no `\Seen`-setting command is ever issued. `TestSeenFlagUnchanged_LiveBridge` still reports SKIP (re-confirmed), unchanged, because the `.env` credential is still not a valid Bridge app password. No regression: still present, wired, and behaviorally unexercised. |
| 3 | An email carrying several Proton labels appears exactly once in the stream, deduped by Message-ID (SC3) | ✓ VERIFIED | Unchanged this round; `TestIMAPTranscript_ExamineAndPeekOnly`'s dedup assertion re-confirmed passing. `plugin.go`'s `Match` untouched by 03-08 (`git diff --exit-code plugins/proton/plugin.go` confirmed clean). |
| 4 | User can type a query and get ranked, clickable matches across every source in the webspace (SC4) | ✓ VERIFIED | Unchanged this round; `kernel/httpapi/search_test.go`'s 7 cases untouched and unaffected by 03-08 (no file under `kernel/` or `web/` was touched — `git diff --exit-code kernel/httpapi/sources.go` and `git diff --exit-code web/` both confirmed clean this session). 03-UAT.md's Test 4 (interactive human pass, with an email-hits caveat due to the same Bridge block) independently corroborates this. |
| 5 | The plugin reaches Proton Mail Bridge over LAN despite its self-signed certificate, and reports a clear, actionable health error rather than hanging when unreachable (SC5) | ✓ VERIFIED (strengthened this round) | TLS pinning (`client.go`) and bounded-timeout `Health` unchanged and re-confirmed. **New this round:** a LOGIN rejected because the configured token cannot be a Bridge-generated app password now produces a health error that *names that specific cause* and says where the real password lives, closing the one previously-uncovered case of "actionable" — verified directly: `plugins/proton/credentials.go`'s `bridgeTokenShapeWarning` (10-row alphabet-boundary table, all PASS, re-run myself) is wired exactly once in `client.go`'s `connect()` LOGIN-failure branch (`grep -c bridgeTokenShapeWarning plugins/proton/client.go` = 1), reaches `HealthResponse.LastError` (Health returns `connect`'s error verbatim, unedited) and, independently, the UI's `sync_runs`-fed `last_error` via `Match` (confirmed by reading `kernel/httpapi/sources.go`'s unedited `LastError: run.Error` line — this file was read, not modified, by 03-08). Never gates a connection: dial-counter assertion in `TestHealth_ShapeSuspectTokenYieldsActionableLastErrorAndStillDials` re-run and passes. Never echoes the credential: no-token-echo assertion re-run and passes; `credentials.go` declares zero imports, no `Sprintf`, no `strconv`, no absolute URL literal (all four re-confirmed by direct grep this session). |

**Score:** 4/5 truths verified (1 present, behavior-unverified)

### Gap Closure: UAT G-03-1 (from 03-UAT.md)

| Item | Owner | Status this round |
|------|-------|--------------------|
| Missing item 1 — replace `PROTON_BRIDGE_PASS` in `.env` with the real Bridge app password | User (not plannable) | **Still open.** Not a code gap; carried forward as human-verification item 1 below. |
| Missing item 2 — correct the misleading `live_bridge_test.go` hint that pointed at the username | Plan 03-08, Task 2 | **✓ Closed.** `! grep -q 'not a code defect'` and `! grep -q '03-01-SUMMARY'` both confirmed (exit 1 = no match) against the current file; `grep -c bridgeAuthOrderNote` = 1, confirming the hint now shares the runtime advice's own constant instead of restating a wrong claim. |
| Missing item 3 — make the misconfig class self-diagnosing at the Health path | Plan 03-08, Task 1 | **✓ Closed, and reaches a second surface the item's literal text didn't name.** `bridgeTokenShapeWarning` verified wired at `client.connect`'s LOGIN-failure branch, reaching both `HealthResponse.LastError` (the literal ask) and, independently walked in code, the `sync_runs`-fed UI red-dot detail via `Match` → `kernel/httpapi/sources.go`. |

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `plugins/proton/credentials.go` | Zero-import file declaring `bridgeAuthOrderNote`, `bridgeTokenShapeWarningText`, `bridgeTokenShapeWarning` | ✓ VERIFIED | Read in full; matches plan spec exactly. `grep -E '^import'` finds nothing (exit 1); no `Sprintf`/`strconv`; no `https?/wss?://` literal; `Bridge-generated app password` and the alphabet (`A-Za-z0-9`) both present in the advice text. |
| `plugins/proton/client.go` | `connect()`'s LOGIN-failure branch appends the shape warning exactly once, success path untouched | ✓ VERIFIED | Read lines 190-215: warning appended via `fmt.Errorf("proton: login: %v — %s", err, warning)` only when non-empty; `grep -c bridgeTokenShapeWarning` = 1. `git diff --exit-code plugins/proton/plugin.go plugins/proton/body.go plugins/proton/main.go kernel/httpapi/sources.go` confirms none of those files were touched. |
| `plugins/proton/credentials_test.go` | Alphabet-boundary table + 3 Health-path tests | ✓ VERIFIED | 4 new test functions present and independently re-run by this verifier (not merely trusted from SUMMARY): `TestBridgeTokenShapeWarning_AlphabetBoundary` (10 sub-tests incl. the `+`/`/`/`=` base64-vs-base64url boundary and a non-ASCII row), `TestHealth_ShapeSuspectTokenYieldsActionableLastErrorAndStillDials`, `TestHealth_WellShapedButWrongTokenGetsNoAddedAdvice`, `TestHealth_CorrectTokenIsReachableWithNoLastError` — all PASS. |
| `plugins/proton/live_bridge_test.go` | Corrected LOGIN-failure hint sourced from `bridgeAuthOrderNote` | ✓ VERIFIED | Confirmed by direct grep as above; `go vet ./...` clean (format verbs match); `TestSeenFlagUnchanged_LiveBridge` still SKIPs (re-run, confirmed). |
| Other proton files (`plugin.go`, `body.go`, `main.go`, five pre-existing test files) and `kernel/httpapi/sources.go`, `web/` | Untouched by 03-08 | ✓ VERIFIED | All four `git diff --exit-code` checks from the plan's own acceptance criteria re-run directly by this verifier; all exit 0 (clean). |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `plugins/proton/client.go connect()` LOGIN-failure branch | `plugins/proton/plugin.go Health() -> HealthResponse.LastError` | `Health` returns `connect`'s error verbatim | ✓ WIRED | Confirmed by reading `plugin.go` lines ~552-568 (unedited); `err.Error()` returned into `LastError` with no transformation. |
| `plugins/proton/client.go connect()` LOGIN-failure branch | `GET /api/sources last_error` (UI red-dot detail) | `Match` wraps the `connect` error as `codes.Unavailable` → kernel `sync_runs` row → `kernel/httpapi/sources.go`'s `sourceStatusesFrom` copies `run.Error` into `last_error` | ✓ WIRED | Confirmed by reading `kernel/httpapi/sources.go`'s unedited `LastError: run.Error` line (line ~108) — the file this verifier confirmed was not modified by 03-08 (clean `git diff`). This is the surface the original UAT gap was reported from, and it is independently reached from the same one `connect()` call site, not merely inherited from the `Health` path. |
| `plugins/proton/live_bridge_test.go` LOGIN-failure hint | `plugins/proton/credentials.go bridgeAuthOrderNote` | Direct reference to the shared constant | ✓ WIRED | `grep -c bridgeAuthOrderNote plugins/proton/live_bridge_test.go` = 1; `go vet` confirms the `t.Fatalf` format verbs match. |
| `plugins/proton/plugin.go Match` | `plugins/proton/plugin.go fetchFull` | Accumulated `mailboxCache` map (03-06 fix) | ✓ WIRED (unchanged) | Untouched by 03-08; re-confirmed via unchanged `plugin.go`. |
| `web/src/lib/components/DetailPane.svelte` | `web/src/lib/format.ts` | Shared `formatItemDate` (03-06 fix) | ✓ WIRED (unchanged) | Untouched by 03-08; `git diff --exit-code web/` clean. |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Shape check present, no imports, no runtime data | `grep -E '^import' plugins/proton/credentials.go`, `grep -c Sprintf/strconv` | no import line found; 0/0 | ✓ PASS |
| No absolute URL literal in the new file | `grep -qE '(https?\|wss?)://' plugins/proton/credentials.go` | no match (exit 1) | ✓ PASS |
| Wired exactly once in `client.go` | `grep -c bridgeTokenShapeWarning plugins/proton/client.go` | `1` | ✓ PASS |
| Test hint no longer misdirects | `grep -q 'not a code defect'` / `grep -q '03-01-SUMMARY'` in `live_bridge_test.go` | both no match (exit 1) | ✓ PASS |
| Test hint shares one source of truth | `grep -c bridgeAuthOrderNote plugins/proton/live_bridge_test.go` | `1` | ✓ PASS |
| Alphabet-boundary + Health-path tests | `cd plugins/proton && go build ./... && go vet ./... && go test -run 'TestBridgeTokenShapeWarning_\|TestHealth_' -count=1 -v ./...` | all PASS (10 sub-tests + 3 Health tests) | ✓ PASS |
| Full `plugins/proton` suite, incl. `TestSeenFlagUnchanged_LiveBridge` SKIP | `go test ./... -count=1 -v` | all PASS, 1 SKIP (unchanged) | ✓ PASS |
| `internal/audit` egress + dependency-floor scan over the new file | `go test ./internal/audit/... -count=1 -v` | 4/4 PASS | ✓ PASS |
| Untouched files confirmed clean | `git diff --exit-code plugin.go body.go main.go kernel/httpapi/sources.go`, and separately the 5 pre-existing proton test files, `go.mod`/`go.sum` (all 6 modules), `web/` | all exit 0 | ✓ PASS |
| Other five workspace modules unaffected | `go build ./... && go test ./... -count=1` in `sdk`, `plugins/paperless`, `plugins/silverbullet`, `plugins/mock` | all `ok` | ✓ PASS |
| No debt markers in the 4 files this plan touched | `grep -n -E "TBD\|FIXME\|XXX\|TODO\|HACK\|PLACEHOLDER" credentials.go client.go credentials_test.go live_bridge_test.go` | no matches | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| SRC-01 | 03-01, 03-02, 03-05, 03-06, 03-07, 03-08 | Email plugin (IMAP) works against Proton Mail Bridge (self-signed cert handling); uses `BODY.PEEK` so mail is never marked read; matches webspace keyword against folders/labels; dedups by Message-ID | ✓ SATISFIED (1 truth present-behavior-unverified pending live Bridge access, unchanged) | Self-signed cert + actionable health error, now including the credential-shape case: Truth 5. `BODY.PEEK`: code-level proof passes (Truth 2); live-mailbox proof still blocked by the unchanged `.env` credential, a user-side action, not a code defect. Keyword matching + dedup: Truth 3. Rendering: Truth 1. REQUIREMENTS.md's checkbox for SRC-01 still reads unchecked/"Gaps Found" — deliberately deferred to the phase's seal step per 03-05 through 03-08's unbroken precedent, not a discrepancy in the underlying code evidence. |
| KERN-05 | 03-03, 03-04 | User can full-text search within a webspace (FTS5 over indexed metadata/previews) | ✓ SATISFIED | Unchanged this round and untouched by 03-08; `search_test.go`'s 7 cases + frontend `format.test.ts` unaffected (Truth 4). |

No orphaned requirements: REQUIREMENTS.md maps exactly SRC-01 and KERN-05 to Phase 3, both declared in at least one plan's `requirements:` frontmatter field (including 03-08's).

### Anti-Patterns Found

Mechanical grep for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` across the four files this round's plan modified (`plugins/proton/credentials.go`, `plugins/proton/client.go`, `plugins/proton/credentials_test.go`, `plugins/proton/live_bridge_test.go`) found **none**. `git show --stat` on both of 03-08's commits (`0999d5d`, `765e87a`) confirms the change surface matches exactly what 03-08-PLAN.md and 03-08-SUMMARY.md describe — no additional files touched, no scope creep.

The prior round's code review (`03-REVIEW.md`, `aee46d1`) findings remain non-blocking and untouched by 03-08, unchanged from every prior round.

### Human Verification Required

See the `human_verification` block in this report's frontmatter (four items). Three are the unchanged live-Bridge-account blockers (unread-state cross-check, live-Bridge `\Seen` test, live in-browser stream/detail-pane rendering — all blocked purely on the `.env` credential correction, which is a user-side action outside code/repo scope per the debug session's ENVIRONMENTAL root cause). The fourth is new: confirming that if a re-run still fails after the credential correction, the surfaced error uses the corrected, non-misleading text rather than the old username-pointing framing (the code-level guarantee for this is verified below; only its rendered appearance in a live run is not). One of the four is also captured as this report's sole `behavior_unverified_items` entry (SC2's `\Seen`-unchanged invariant).

### Gaps Summary

No code gaps remain. UAT gap G-03-1's two code-side `missing` items — the misleading `live_bridge_test.go` hint and the silent Health/red-dot diagnostic — are both closed by plan 03-08, independently re-verified in this session by reading the changed files directly and by re-running every relevant test myself (not merely trusting SUMMARY.md's reported output): the alphabet-boundary table (10 rows, including the `+`/`/`/`=` base64-vs-base64url boundary this gap's own history flagged as load-bearing), the three Health-path tests (actionable message, no token echo, dial-counter proving the check never gates a connection), the wiring-count and content greps against `client.go` and `live_bridge_test.go`, and the four `git diff --exit-code` checks confirming no unintended file was touched. `internal/audit`'s repo-wide egress and dependency-floor scans stay green over the new file. All five other workspace modules and the frontend suite (per the prior round's already-passing state, plus 03-08's `git diff --exit-code web/` confirming zero frontend touch) are unaffected.

The phase's status remains `human_needed` rather than `passed` because UAT gap G-03-1's root item — `.env`'s `PROTON_BRIDGE_PASS` is not a valid Bridge app password — is environmental, root-caused with the code path audited clean, and is a user action that no plan can perform. This is unchanged from the previous verification round; what has changed is that once the user corrects it, any further diagnosis (should one be needed) will be immediate rather than costing another four rounds, because the diagnostic surfaces themselves are now fixed. SC2's `\Seen`-unchanged guarantee remains present-and-wired but behaviorally unexercised against a real mailbox for the same reason.

---

_Verified: 2026-08-01T01:15:00Z_
_Verifier: Claude (gsd-verifier)_
