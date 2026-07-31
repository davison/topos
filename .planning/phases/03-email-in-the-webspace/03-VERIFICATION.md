---
phase: 03-email-in-the-webspace
verified: 2026-07-31T18:45:00Z
status: human_needed
score: 4/5 must-haves verified
behavior_unverified: 1
overrides_applied: 0
re_verification:
  previous_status: gaps_found
  previous_score: 3/5
  gaps_closed:
    - "plugins/proton/go.mod pinned golang.org/x/net to v0.26.0 (CVE-2024-45338 / GO-2024-3333, a non-linear HTML-tokenization DoS in golang.org/x/net/html, the tokenizer behind github.com/microcosm-cc/bluemonday and reachable from body.go's RenderSanitizedEmail over arbitrary inbound email HTML) — now bumped to v0.56.0 (matching the version go.work's MVS already selects, so the shipped binary's compiled dependency set is provably unchanged), with two new nil/empty-input boundary tests added to render_test.go and a new permanent repo-wide audit test (internal/audit/module_pins_test.go's TestNoModuleDeclaresAKnownVulnerablePin) that fails any future go test ./... if any of the six go.work modules regresses below a documented security floor. Confirmed independently in this round by reading plugins/proton/go.mod (golang.org/x/net v0.56.0 // indirect), by running the module-scoped resolution check (GOWORK=off go list -m golang.org/x/net -> v0.56.0, up from v0.26.0 pre-bump) and the workspace-selected check (go list -m golang.org/x/net -> v0.56.0, unchanged), by running the full plugins/proton suite (16 PASS, 1 environment-gated SKIP, including the two new render_test.go boundary tests), and by running the new audit test directly (internal/audit: 4/4 PASS, including the negative-control fixture test) and confirming git diff --exit-code across the other five modules' go.mod/go.sum plus go.work/go.work.sum is clean (no unrelated module touched)."
  gaps_remaining: []
  regressions: []
gaps: []
human_verification:
  - test: "Open a webspace with a real, currently-reachable Proton Bridge account configured. Confirm at least one real Proton email appears in the stream interleaved with paperless/SilverBullet items, showing sender before the date, with the subject as the row title. Click it: the detail pane opens and shows the message's body (plain text, or formatted HTML with headings/links/colors if the message has an HTML part). If more than one configured webspace matches Proton mail, additionally confirm the SAME check holds for a webspace that is NOT the last one synced in that cycle."
    expected: "The email is visible, correctly dated, and its body renders in the detail pane for every configured webspace that matches Proton mail — not just the one processed last in a sync cycle. This is the direct live confirmation of the mailboxCache fix (03-06) plus the CVE-remediated rendering path (03-07)."
    why_human: "Requires a running kernel/webUI against a real, currently-authenticating Proton Mail Bridge account and visual confirmation of rendered HTML/CSS in a browser — not mechanically checkable from source alone. Blocked again this session: the Bridge account credential rejection recorded in 03-01-SUMMARY.md's 'Notable Live-Environment Finding' is unchanged across four verification passes now."
  - test: "Run `WEBSPACES_PROTON_LIVE_IT=1 PROTON_BRIDGE_ADDR=<addr> PROTON_BRIDGE_USER=<user> PROTON_BRIDGE_PASS=<pass> go test -run TestSeenFlagUnchanged_LiveBridge -v ./plugins/proton/...` against the real Bridge account once its credential issue is corrected."
    expected: "The test passes, directly proving SRC-01's second success criterion (\\Seen flag unchanged across a full Match+Fetch cycle) against the real mailbox, not just the no-live-Bridge wire-transcript proxy for it (TestIMAPTranscript_ExamineAndPeekOnly / TestPluginIssuesNoIMAPMutatingCommands, both of which still pass, confirmed again this session)."
    why_human: "Requires a live, currently-authenticating Bridge connection and real mailbox state; the test is implemented and skips cleanly (confirmed again this session: SKIP, not FAIL) but has never been run to a PASS — same Bridge credential blocker as before, unchanged."
  - test: "After running a webspaces sync and opening an email in the detail pane, check the same email in the real Proton web or mobile client and confirm it is still shown as unread there."
    expected: "The email remains unread in Proton's own client — the direct, human-observable proof of the never-mark-read guarantee end to end."
    why_human: "Requires a live Bridge account, a real sync, and cross-checking read/unread state in Proton's own UI; unchanged blocker."
  - test: "Open a webspace in the browser, type a word present in a document, a note, and an email into the search box, and confirm ranked cross-source results appear with the matched word emboldened, a result opens the detail pane on click, clearing restores the unfiltered stream, and a nonsense or malformed (lone double-quote) query shows the no-matches state rather than an error."
    expected: "All of 03-04-PLAN.md's Task 1 human-check behaviors hold visually in a running browser."
    why_human: "Visual/interactive confirmation of debounce feel, snippet emphasis rendering, and detail-pane opening — beyond what the passing unit tests over parseSnippet/searchVariant can prove; not run this session (no running kernel/webUI session with live data was exercised)."
behavior_unverified_items:
  - truth: "Reading an email inside webspaces never marks it read in Proton — proven by an automated test asserting the \\Seen flag is unchanged after a full sync and a detail fetch (ROADMAP SC2)"
    test: "Run TestSeenFlagUnchanged_LiveBridge against a real, currently-authenticating Proton Mail Bridge account."
    expected: "\\Seen flag on the fetched message is unchanged after a full Match + Fetch (detail-open) cycle, asserted directly against the real mailbox rather than inferred from an absence of mutating IMAP commands in a wire transcript."
    why_human: "Code-level proof exists (TestPluginIssuesNoIMAPMutatingCommands' AST scan forbidding IMAP-mutating identifiers; TestIMAPTranscript_ExamineAndPeekOnly's wire-transcript assertion that only EXAMINE/BODY.PEEK[ are issued) and both pass, but no test has ever run to PASS against a live mailbox's actual flag state — the live-Bridge test exists and skips cleanly rather than failing, blocked by the unchanged Bridge credential issue recorded in 03-01-SUMMARY.md."
---

# Phase 3: Email in the Webspace Verification Report

**Phase Goal:** User's Proton mail for a topic appears in the webspace stream and stays readable there, and the volume it brings is navigable by searching within the webspace
**Verified:** 2026-07-31T18:45:00Z
**Status:** human_needed
**Re-verification:** Yes — after gap closure plan 03-07 (plugins/proton/go.mod's declared golang.org/x/net bumped past CVE-2024-45338's fix boundary; repo-wide dependency-floor audit test added)

**Note on MVP mode:** ROADMAP.md marks this phase `Mode: mvp`, but its `Goal:` line is not phrased as a User Story (`As a … I want to … so that …`) — a discrepancy the phase's own plans (03-05, 03-06, 03-07) already noted and deliberately did not resolve by inventing one. Running `user-story.validate` against the goal text confirms `valid: false`. This report follows the precedent set by all three prior verification rounds for this phase and uses standard goal-backward verification rather than the MVP User Flow Coverage format.

## Goal Achievement

### Observable Truths

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Emails in Proton folders/labels matching the webspace keyword appear in the stream with sender, subject, date, and render body inline in the detail pane (SC1) | ✓ VERIFIED | The two functional blockers found in earlier rounds (mailboxCache replace-not-merge; DetailPane date drift) remain fixed and test-locked, re-confirmed this round by reading plugin.go/DetailPane.svelte and re-running the full suites. The CRITICAL gap this round's prior verification pass found — `plugins/proton/go.mod` pinning `golang.org/x/net v0.26.0`, affected by CVE-2024-45338, reachable via `body.go`'s `RenderSanitizedEmail` (the function this truth's "render body inline" clause requires) — is now closed: `go.mod` declares `golang.org/x/net v0.56.0` (confirmed by direct read and by `go list -m golang.org/x/net` returning `v0.56.0` both module-scoped and workspace-scoped), and all pre-existing plus two new boundary-case sanitize/wrap tests pass unchanged. The live, in-browser confirmation with a real Bridge account remains a separate human-verification item below (unrelated to the now-closed code-level gap). |
| 2 | Reading an email inside webspaces never marks it read in Proton — proven by an automated test asserting `\Seen` unchanged after a full sync and a detail fetch (SC2) | ⚠️ PRESENT_BEHAVIOR_UNVERIFIED | `TestPluginIssuesNoIMAPMutatingCommands` (AST scan forbidding IMAP-mutating identifiers) and `TestIMAPTranscript_ExamineAndPeekOnly` (wire-transcript assertion: `EXAMINE`/`BODY.PEEK[` present, `SELECT`/`STORE`/`EXPUNGE`/`APPEND`/`COPY`/`MOVE`/`DELETE` absent) both re-confirmed passing this round — code-level proof no `\Seen`-setting command is ever issued. `TestSeenFlagUnchanged_LiveBridge`, the test that would assert the actual flag value on a real mailbox, still SKIPs (confirmed again this session) due to the unchanged Bridge credential blocker. No test has ever run to PASS against a real account, so the state-transition invariant itself (flag unchanged on the server) remains present-and-wired but behaviorally unexercised. |
| 3 | An email carrying several Proton labels appears exactly once in the stream, deduped by Message-ID (SC3) | ✓ VERIFIED | `plugin.go`'s `Match`'s `byMessageID` map merges matches by `normalizeMessageID(msg.Envelope.MessageId)`, appending the mailbox leaf to `labels` on a repeat hit. `TestIMAPTranscript_ExamineAndPeekOnly` asserts exactly 1 item for a message present in two matching mailboxes, both leaf names present in `item.GetLabels()`. Re-confirmed passing this round; unchanged since prior verification. |
| 4 | User can type a query and get ranked, clickable matches across every source in the webspace (SC4) | ✓ VERIFIED | `kernel/httpapi/search.go` + `kernel/index` FTS5 bm25-ranked query; `kernel/httpapi/search_test.go`'s 7 tests all pass (unknown webspace 404, empty/whitespace query, malformed single-quote query -> 200 not 500, zero-match, one-match-with-snippet). Frontend `format.test.ts`'s `parseSnippet`/`searchVariant` tests pass. Unchanged this round, re-confirmed passing. |
| 5 | The plugin reaches Proton Mail Bridge over LAN despite its self-signed certificate, and reports a clear, actionable health error rather than hanging when unreachable (SC5) | ✓ VERIFIED | `client.go`'s TLS `RootCAs` pinning to the exported Bridge cert (`InsecureSkipVerify` absent, confirmed via grep); `plugin.go`'s `Health` calls a bounded-timeout dial and returns a structured `HealthResponse{Reachable: false, LastError: ...}` on failure rather than hanging. Unchanged this round, re-confirmed. The real Bridge account's own credential-rejection error (`03-01-SUMMARY.md`) is itself further live evidence this success criterion holds — Bridge reported a specific `last_error` rather than hanging. |

**Score:** 4/5 truths verified (1 present, behavior-unverified)

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `plugins/proton/go.mod` | Declared `golang.org/x/net` requirement at or above v0.33.0 (CVE-2024-45338 fix boundary) | ✓ VERIFIED | `golang.org/x/net v0.56.0 // indirect` present in the second `require` block; confirmed by direct read. |
| `plugins/proton/go.sum` | Checksum entries for the newly-selected versions | ✓ VERIFIED | `golang.org/x/net v0.56.0` (hash + go.mod hash) and `golang.org/x/text v0.38.0` present alongside the retained (inert) older lines — `go get` appends, never prunes, consistent with the plan's documented rationale (`go mod tidy` cannot run cleanly against this module in isolation). |
| `plugins/proton/render_test.go` | Two new nil/empty-input boundary tests | ✓ VERIFIED | `TestRenderSanitizedEmail_EmptyAndNilInputYieldNoOutput` and `TestWrapDocument_NilFragmentStillYieldsADocument` both present and pass; substantive assertions (zero-length output, doctype/style/body markers), not stubs. |
| `internal/audit/module_pins_test.go` | Repo-wide declared-dependency-floor audit test, non-vacuity proven | ✓ VERIFIED | `TestNoModuleDeclaresAKnownVulnerablePin` walks `repoRoot`, asserts >= 6 go.mod files discovered, fails on any below-floor declaration; `TestPinScanner_FixtureReportsTheBelowFloorDeclaration` proves the scanner reports a real offence against the fixture. Both pass; ran directly in this session (4/4 `internal/audit` tests PASS, including the pre-existing egress-scanner tests, confirming no regression to that package). |
| `internal/audit/testdata/vulnerable_pin_go.mod.txt` | Negative-control fixture with a below-floor `golang.org/x/net` declaration | ✓ VERIFIED | Contains `golang.org/x/net v0.26.0 // indirect` in the standalone-require shape, plus a safe requirement and an untracked-module-path requirement; confirmed by direct read and by the fixture test passing. |
| Other five workspace modules' go.mod/go.sum, go.work, go.work.sum | Untouched by this plan | ✓ VERIFIED | `git diff --exit-code go.work go.work.sum go.mod go.sum sdk/go.mod sdk/go.sum plugins/silverbullet/go.mod plugins/paperless/go.mod plugins/mock/go.mod` returns clean (exit 0) against the pre-03-07 tree, confirming no unrelated module path or version was touched. All five independently confirmed already declaring `golang.org/x/net v0.53.0` (above the v0.33.0 floor). |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `plugins/proton/body.go RenderSanitizedEmail` | `golang.org/x/net/html` (via `bluemonday`) | `bluemonday.SanitizeBytes` running the HTML tokenizer over an arbitrary inbound email's body | ✓ REMEDIATED | The tokenizer version behind this link is now v0.56.0 (past the CVE-2024-45338 fix boundary), confirmed via `go list -m golang.org/x/net` returning `v0.56.0` both module- and workspace-scoped, closing the gap the prior verification round marked NOT REMEDIATED. All sanitize/wrap tests, including two new boundary cases, pass unchanged, confirming no rendering regression. |
| `internal/audit/module_pins_test.go` | Every workspace module's go.mod | `repoRoot` filesystem walk (shared with the existing egress scanner) | ✓ WIRED | Discovers 6 go.mod files (asserted, not assumed); scans both require-block shapes; confirmed passing directly in this session. |
| `plugins/proton/plugin.go Match` | `plugins/proton/plugin.go fetchFull` | Accumulated `mailboxCache` map (03-06 fix) | ✓ WIRED (unchanged) | Untouched by 03-07; re-confirmed passing (`TestMatch_MailboxCacheSurvivesASecondWebspaceMatch`, `TestMatch_ZeroMailboxMatchPreservesMailboxCache`). |
| `web/src/lib/components/DetailPane.svelte` | `web/src/lib/format.ts` | Shared `formatItemDate` (03-06 fix) | ✓ WIRED (unchanged) | Untouched by 03-07; `date-format.test.ts` guard still passes (72/72 frontend tests). |

### Behavioral Spot-Checks

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Declared golang.org/x/net requirement bumped past the CVE fix boundary | `grep -n "golang.org/x/net" plugins/proton/go.mod` | `golang.org/x/net v0.56.0 // indirect` | ✓ PASS |
| Module-scoped resolution now matches the fixed version | `cd plugins/proton && GOWORK=off go list -m golang.org/x/net` | `golang.org/x/net v0.56.0` (was v0.26.0 pre-bump per 03-07-SUMMARY.md's recorded measurement) | ✓ PASS |
| Workspace-selected resolution unchanged (shipped binary provably unaffected either way) | `cd plugins/proton && go list -m golang.org/x/net` | `golang.org/x/net v0.56.0` | ✓ PASS |
| Full `plugins/proton` suite, including two new boundary tests | `cd plugins/proton && go test ./... -count=1 -v` | 16 PASS, 1 environment-gated SKIP (`TestSeenFlagUnchanged_LiveBridge`) | ✓ PASS |
| Repo-wide dependency-floor audit test, plus non-vacuity fixture test | `go test ./internal/audit/... -run 'TestNoModuleDeclaresAKnownVulnerablePin\|TestPinScanner_FixtureReportsTheBelowFloorDeclaration' -v -count=1` | Both PASS | ✓ PASS |
| Other five modules' go.mod/go.sum + go.work untouched | `git diff --exit-code go.work go.work.sum go.mod go.sum sdk/go.mod sdk/go.sum plugins/silverbullet/go.mod plugins/paperless/go.mod plugins/mock/go.mod` | exit 0 | ✓ PASS |
| Full repo build/vet/test (all six Go modules) | `CGO_ENABLED=0 go build ./... && go test ./... -count=1` (root) + each of sdk/paperless/silverbullet/proton/mock built and tested individually | All `ok` or `[no test files]` | ✓ PASS |
| Frontend suite unaffected (this plan touches no frontend file) | `npm --prefix web run test` | 72/72 pass | ✓ PASS |
| No debt markers in files this plan touched | `grep -n -E "TBD\|FIXME\|XXX\|TODO\|HACK\|PLACEHOLDER" plugins/proton/go.mod plugins/proton/render_test.go internal/audit/module_pins_test.go internal/audit/testdata/vulnerable_pin_go.mod.txt internal/audit/doc.go` | no matches | ✓ PASS |

### Requirements Coverage

| Requirement | Source Plan | Description | Status | Evidence |
|-------------|-------------|-------------|--------|----------|
| SRC-01 | 03-01, 03-02, 03-05, 03-06, 03-07 | Email plugin (IMAP) works against Proton Mail Bridge (self-signed cert handling); uses `BODY.PEEK` so mail is never marked read; matches webspace keyword against folders/labels; dedups by Message-ID | ✓ SATISFIED (1 truth present-behavior-unverified pending live Bridge access) | Self-signed cert: `client.go` TLS pinning (Truth 5). `BODY.PEEK`: code-level proof passes (Truth 2); live-mailbox proof still blocked by an environment/credential issue, not a code defect. Keyword matching + dedup: Truth 3. The one previously-open code-level security gap (stale `golang.org/x/net`) is now closed (Truth 1). REQUIREMENTS.md's checkbox for SRC-01 still reads unchecked/"Gaps Found" — this is bookkeeping deliberately deferred to the phase's re-verify/seal step per 03-05/03-06/03-07 precedent, not a discrepancy in the underlying evidence. |
| KERN-05 | 03-03, 03-04 | User can full-text search within a webspace (FTS5 over indexed metadata/previews) | ✓ SATISFIED | Unchanged this round; `search_test.go`'s 7 cases + frontend `format.test.ts` all pass (Truth 4). |

No orphaned requirements: REQUIREMENTS.md maps exactly SRC-01 and KERN-05 to Phase 3, both declared in at least one plan's `requirements:` frontmatter field.

### Anti-Patterns Found

Mechanical grep for `TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER` across this round's modified files (`plugins/proton/go.mod`, `plugins/proton/go.sum`, `plugins/proton/render_test.go`, `internal/audit/module_pins_test.go`, `internal/audit/testdata/vulnerable_pin_go.mod.txt`, `internal/audit/doc.go`) found **none**. `git diff --stat` for the plan's three commits confirms the change surface matches exactly what 03-07-PLAN.md and 03-07-SUMMARY.md describe (7 files, +633/-3 lines) — no additional files touched, no scope creep.

The prior round's code review (`03-REVIEW.md`, `aee46d1`) surfaced several WR-*/IN-* non-blocking findings (context.Context not honored in IMAP calls; Subject stored verbatim with no control-character stripping; SEARCH HEADER substring-match trust; search-clear race on `searchRequestSeq`; cold-cache-vs-deleted-message ambiguity; assorted dead code) — none of these were re-scoped as blocking gaps by any plan, and 03-07 did not touch any of the files they concern. They remain recorded as non-blocking, left to the project's own backlog process, consistent with prior rounds.

### Human Verification Required

See the `human_verification` block in this report's frontmatter (four items, all environment-blocked by the unchanged Proton Mail Bridge credential issue first recorded in 03-01-SUMMARY.md, now unchanged across four verification passes). One of these four items is also captured as the phase's sole `behavior_unverified_items` entry (SC2's `\Seen`-unchanged invariant), since presence/wiring evidence for it is complete but the state-transition itself has never been exercised against a real mailbox.

### Gaps Summary

The single blocking gap from the previous verification round — `plugins/proton/go.mod` declaring `golang.org/x/net v0.26.0`, affected by CVE-2024-45338/GO-2024-3333 and reachable via `RenderSanitizedEmail`'s HTML tokenization of arbitrary inbound email — is now closed. Plan 03-07 bumped the declared requirement to v0.56.0 (matching the version the workspace already builds, so the shipped binary's dependency set is provably unchanged), added two boundary-case regression tests confirming no rendering change, and added a permanent repo-wide audit test that will catch any future regression across all six workspace modules. All of this was independently re-derived in this verification pass by reading the changed files directly and by running the relevant tests myself (not merely trusting SUMMARY.md's reported output) — the go.mod diff, the go.sum additions, the new render_test.go assertions, the new audit test and its fixture, and the confirmation that no other module's manifest was touched all check out.

No gaps remain. The phase's status is `human_needed` rather than `passed` because one observable truth (SC2's `\Seen`-unchanged guarantee) has full code-level proof (an AST scan and a wire-transcript assertion, both passing) but no test has ever run to PASS against a real Proton mailbox — the live-Bridge test exists, is well-formed, and skips cleanly rather than failing, but remains blocked by an unrelated environment/credential issue (the configured Bridge account rejects LOGIN) that is unchanged across four verification rounds now and is not something any of this phase's plans can fix from within the codebase. The other three human-verification items (live visual confirmation of stream/detail-pane rendering, live unread-state cross-check in Proton's own client, and live search UX confirmation) are likewise unchanged and are recorded for a human to close out once Bridge account access is corrected.

---

_Verified: 2026-07-31T18:45:00Z_
_Verifier: Claude (gsd-verifier)_
