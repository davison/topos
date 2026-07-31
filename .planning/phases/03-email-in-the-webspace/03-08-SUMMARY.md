---
phase: 03-email-in-the-webspace
plan: 08
subsystem: auth-diagnostics
tags: [go, imap, proton-bridge, error-messages, tdd, gap-closure]

# Dependency graph
requires:
  - phase: 03-email-in-the-webspace (03-07)
    provides: Proton IMAP plugin's connect/Match/Fetch/Health path, and the CVE-pinned module set this plan adds zero dependencies to
provides:
  - A self-diagnosing Bridge app-password shape check (bridgeTokenShapeWarning) reachable from client.connect's LOGIN-failure branch, wired to both HealthResponse.LastError and, via Match -> sync_runs, the UI's red-dot last_error
  - A corrected live-Bridge test LOGIN-failure hint that states Bridge's real authentication order and points at the password, sharing one constant (bridgeAuthOrderNote) with the runtime advice
affects: [proton-plugin-diagnostics, live-bridge-verification]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Dependency-free diagnostic helper file (credentials.go declares zero imports) as a supply-chain-hardening pattern for advice text that inspects a secret"
    - "Compile-time constant concatenation to provably exclude runtime data from an operator-facing diagnostic string"
    - "Shared const between production code and its own test's failure hint, referenced by value, to prevent the two from drifting apart"

key-files:
  created:
    - plugins/proton/credentials.go
    - plugins/proton/credentials_test.go
  modified:
    - plugins/proton/client.go
    - plugins/proton/live_bridge_test.go

key-decisions:
  - "Wired the shape check into client.connect's LOGIN-failure branch, not plugin.go's Health — connect is the one call site both Health and Match/fetchFull share, and kernel/httpapi/sources.go's sourceStatusesFrom builds the UI's last_error exclusively from the kernel's own sync_runs row (fed by Match's error), never from HealthResponse, so a Health-only fix would not have reached the surface the gap was reported from"
  - "Both new constants declare zero imports and use only compile-time string concatenation, so the advice text is provably free of runtime data (T-03-03) by construction rather than by review"
  - "live_bridge_test.go's corrected hint references credentials.go's bridgeAuthOrderNote by value instead of restating it, closing off the specific drift pattern that let a wrong explanation survive four verification rounds"
  - "03-01-SUMMARY.md was deliberately left unedited — see 'Immutable execution record' section below"

requirements-completed: [SRC-01]

coverage:
  - id: D1
    description: "A LOGIN rejected while the configured token cannot be a Bridge app password produces an error naming that cause and directing the operator to the real password, reaching both HealthResponse.LastError and the UI's red-dot last_error from one wiring point"
    requirement: "SRC-01"
    verification:
      - kind: unit
        ref: "plugins/proton/credentials_test.go#TestHealth_ShapeSuspectTokenYieldsActionableLastErrorAndStillDials"
        status: pass
      - kind: unit
        ref: "plugins/proton/credentials_test.go#TestBridgeTokenShapeWarning_AlphabetBoundary"
        status: pass
    human_judgment: false
  - id: D2
    description: "A well-shaped (even wrong) token produces no added advice — the check discriminates one specific misconfiguration rather than decorating every authentication failure"
    requirement: "SRC-01"
    verification:
      - kind: unit
        ref: "plugins/proton/credentials_test.go#TestHealth_WellShapedButWrongTokenGetsNoAddedAdvice"
        status: pass
    human_judgment: false
  - id: D3
    description: "The shape check never gates a dial, LOGIN, or plugin construction — it only appends text to a login that has already failed"
    requirement: "SRC-01"
    verification:
      - kind: unit
        ref: "plugins/proton/credentials_test.go#TestHealth_ShapeSuspectTokenYieldsActionableLastErrorAndStillDials (dial-counter assertion)"
        status: pass
    human_judgment: false
  - id: D4
    description: "The advice is a compile-time constant containing no part of the credential"
    requirement: "SRC-01"
    verification:
      - kind: unit
        ref: "plugins/proton/credentials_test.go#TestHealth_ShapeSuspectTokenYieldsActionableLastErrorAndStillDials (no-token-echo assertion)"
        status: pass
      - kind: other
        ref: "grep -q 'const bridgeAuthOrderNote = ' / ! grep -q Sprintf / ! grep -q strconv / ! grep -qE '^import' plugins/proton/credentials.go"
        status: pass
    human_judgment: false
  - id: D5
    description: "The live-Bridge test's LOGIN-failure hint states Bridge's real authentication order, points at the password, and shares one constant with the runtime advice"
    requirement: "SRC-01"
    verification:
      - kind: other
        ref: "grep -q bridgeAuthOrderNote / ! grep -q '03-01-SUMMARY' / ! grep -q 'not a code defect' plugins/proton/live_bridge_test.go; go vet ./..."
        status: pass
    human_judgment: false
  - id: D6
    description: "Every Go suite across all six workspace modules passes, go vet is clean, the live-Bridge test still skips (not fails), and no go.mod/go.sum changed"
    requirement: "SRC-01"
    verification:
      - kind: integration
        ref: "CGO_ENABLED=0 go build ./... && go vet ./... && go test ./... -count=1 (repo root, sdk, plugins/paperless, plugins/silverbullet, plugins/proton, plugins/mock)"
        status: pass
    human_judgment: false
  - id: D7
    description: "Frontend suite (npm run test, npm run check) unaffected — no frontend file touched"
    verification: []
    human_judgment: true
    rationale: "web/node_modules is not installed in this isolated worktree (a pre-existing environment limitation of this worktree, not caused by this plan's changes), so npm run test/check could not be executed here. git diff --exit-code web/ confirms zero frontend files were modified, which is the only claim this plan makes about the frontend — but the suite itself was not run and should be spot-checked."

duration: 22min
completed: 2026-08-01
status: complete
---

# Phase 03 Plan 08: Self-diagnosing Bridge app-password shape check + corrected live-test hint Summary

**A rejected Proton Bridge LOGIN now names the specific cause when the configured token cannot be a Bridge-generated app password, reaching both `HealthResponse.LastError` and the UI's red dot from one wiring point in `client.connect`, and the live-Bridge test's own failure hint no longer misdirects the reader at the username.**

## Performance

- **Duration:** 22 min
- **Completed:** 2026-08-01T23:24:42Z (UTC)
- **Tasks:** 2
- **Files modified:** 4 (2 created, 2 modified)

## Accomplishments

- `plugins/proton/credentials.go` (new, zero imports): `bridgeAuthOrderNote`, `bridgeTokenShapeWarningText`, and `bridgeTokenShapeWarning` — the shared explanation of Bridge's real authentication order and the byte-wise base64url alphabet predicate that decides when to append it.
- `plugins/proton/client.go`: `connect`'s LOGIN-failure branch appends the shape warning (only when non-empty) after the server's own rejection message — exactly one call site, nowhere in `NewClient`, `realDial`, or the success path.
- `plugins/proton/credentials_test.go` (new): the alphabet-boundary table (including the base64-vs-base64url boundary at `+`/`/`/`=` and the double-quote that broke the TOML config) and three `Health`-path tests proving the connection is still attempted, no token echo, and the success path is untouched.
- `plugins/proton/live_bridge_test.go`: the LOGIN-failure hint now references `bridgeAuthOrderNote` by value instead of citing `03-01-SUMMARY.md` and asserting a settled username-side cause.

## Task Commits

1. **Task 1: a rejected LOGIN says which knob is wrong** — `0999d5d` (feat)
2. **Task 2: the live-Bridge test's failure hint stops pointing at the username** — `765e87a` (fix)

_Task 1 is `type="tracer" tdd="true"`: RED (compile failure) then GREEN (all tests passing) both happened inside the single `0999d5d` commit — the plan's own action explicitly directs writing the test file first, confirming the compile-failure RED signal, then adding the implementation and confirming GREEN, all before the first commit of this task's file set. The RED and GREEN transcripts are quoted verbatim below._

**Plan metadata:** committed alongside this SUMMARY (see final commit).

## RED / GREEN / Inversion (Task 1, tdd="true")

### RED — compile failure naming the missing predicate

```
$ CGO_ENABLED=0 go test ./... -count=1
# github.com/davison/webspaces/plugins/proton [github.com/davison/webspaces/plugins/proton.test]
./credentials_test.go:64:66: undefined: bridgeTokenShapeWarningText
./credentials_test.go:65:70: undefined: bridgeTokenShapeWarningText
./credentials_test.go:66:70: undefined: bridgeTokenShapeWarningText
./credentials_test.go:67:93: undefined: bridgeTokenShapeWarningText
./credentials_test.go:68:27: undefined: bridgeTokenShapeWarningText
./credentials_test.go:69:37: undefined: bridgeTokenShapeWarningText
./credentials_test.go:74:11: undefined: bridgeTokenShapeWarning
./credentials_test.go:133:43: undefined: bridgeTokenShapeWarningText
FAIL	github.com/davison/webspaces/plugins/proton [build failed]
FAIL
```

This is a valid and sufficient RED signal for this plan: the alphabet table and the `Health`-path assertions describe behaviour that did not exist yet, and the compiler names exactly the two missing symbols `credentials.go` was about to add.

### GREEN — full proton suite, all passing

```
$ CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go vet ./... && CGO_ENABLED=0 go test ./... -count=1 -v
=== RUN   TestBridgeTokenShapeWarning_AlphabetBoundary
    --- PASS: .../empty_token... (0.00s)
    --- PASS: .../all-alphanumeric_token (0.00s)
    --- PASS: .../hyphen_and_underscore... (0.00s)
    --- PASS: .../single-character_token_in_the_alphabet (0.00s)
    --- PASS: .../plus_sign... (0.00s)
    --- PASS: .../forward_slash... (0.00s)
    --- PASS: .../equals_padding... (0.00s)
    --- PASS: .../double_quote... (0.00s)
    --- PASS: .../space (0.00s)
    --- PASS: .../non-ASCII_rune (0.00s)
--- PASS: TestBridgeTokenShapeWarning_AlphabetBoundary (0.00s)
=== RUN   TestHealth_ShapeSuspectTokenYieldsActionableLastErrorAndStillDials
--- PASS: TestHealth_ShapeSuspectTokenYieldsActionableLastErrorAndStillDials (0.00s)
=== RUN   TestHealth_WellShapedButWrongTokenGetsNoAddedAdvice
--- PASS: TestHealth_WellShapedButWrongTokenGetsNoAddedAdvice (0.00s)
=== RUN   TestHealth_CorrectTokenIsReachableWithNoLastError
--- PASS: TestHealth_CorrectTokenIsReachableWithNoLastError (0.00s)
=== RUN   TestIMAPTranscript_ExamineAndPeekOnly
--- PASS: TestIMAPTranscript_ExamineAndPeekOnly (0.00s)
=== RUN   TestMatch_ItemTimestampIsInternalDate
--- PASS: TestMatch_ItemTimestampIsInternalDate (0.00s)
=== RUN   TestToItem_ZeroInternalDateYieldsZeroTimestamp
--- PASS: TestToItem_ZeroInternalDateYieldsZeroTimestamp (0.00s)
=== RUN   TestMatch_EmptyMessageIDSkipIsLogged
--- PASS: TestMatch_EmptyMessageIDSkipIsLogged (0.00s)
=== RUN   TestSeenFlagUnchanged_LiveBridge
    live_bridge_test.go:52: live-Bridge test skipped: set WEBSPACES_PROTON_LIVE_IT=1 and PROTON_BRIDGE_ADDR/PROTON_BRIDGE_USER/PROTON_BRIDGE_PASS to run it
--- SKIP: TestSeenFlagUnchanged_LiveBridge (0.00s)
=== RUN   TestMatch_MailboxCacheSurvivesASecondWebspaceMatch
--- PASS: TestMatch_MailboxCacheSurvivesASecondWebspaceMatch (0.00s)
=== RUN   TestMatch_ZeroMailboxMatchPreservesMailboxCache
--- PASS: TestMatch_ZeroMailboxMatchPreservesMailboxCache (0.00s)
=== RUN   TestAllowHost_PredicateTable
--- PASS: TestAllowHost_PredicateTable (0.00s)
=== RUN   TestPluginIssuesNoIMAPMutatingCommands
--- PASS: TestPluginIssuesNoIMAPMutatingCommands (0.00s)
=== RUN   TestRenderSanitizedEmail_* / TestWrapDocument_*  (9 tests)
--- PASS (all 9)
PASS
ok  	github.com/davison/webspaces/plugins/proton	0.018s
```

`internal/audit`'s repo-wide egress scan over the new production file also passed: `CGO_ENABLED=0 go test ./internal/audit/... -count=1` -> `ok github.com/davison/webspaces/internal/audit`.

### Step 4 — non-vacuity by inversion

A temporary, uncommitted edit commented out the `bridgeTokenShapeWarning` append in `connect`'s failure branch:

```
$ cd plugins/proton && CGO_ENABLED=0 go test -run 'TestHealth_' -count=1 -v ./...
=== RUN   TestHealth_ShapeSuspectTokenYieldsActionableLastErrorAndStillDials
    credentials_test.go:105: Health: LastError = "proton: login: Bad username or password", want it to identify the token as not a Bridge-generated app password
--- FAIL: TestHealth_ShapeSuspectTokenYieldsActionableLastErrorAndStillDials (0.00s)
=== RUN   TestHealth_WellShapedButWrongTokenGetsNoAddedAdvice
--- PASS: TestHealth_WellShapedButWrongTokenGetsNoAddedAdvice (0.00s)
=== RUN   TestHealth_CorrectTokenIsReachableWithNoLastError
--- PASS: TestHealth_CorrectTokenIsReachableWithNoLastError (0.00s)
FAIL
FAIL	github.com/davison/webspaces/plugins/proton	0.005s
```

The shape-suspect test genuinely depends on the wiring: it fails without it, while the well-shaped and success tests are unaffected (as expected — they never exercised the append). The edit was reverted immediately (restored from a pre-inversion copy) and GREEN was re-confirmed:

```
$ cd plugins/proton && CGO_ENABLED=0 go test -run 'TestHealth_' -count=1 -v ./...
--- PASS: TestHealth_ShapeSuspectTokenYieldsActionableLastErrorAndStillDials (0.00s)
--- PASS: TestHealth_WellShapedButWrongTokenGetsNoAddedAdvice (0.00s)
--- PASS: TestHealth_CorrectTokenIsReachableWithNoLastError (0.00s)
PASS
ok  	github.com/davison/webspaces/plugins/proton	0.005s
```

No commit exists for the inverted state — it was never staged or committed, only run in the working tree and reverted before Task 1's single commit.

## The two new constants, as shipped

```go
const bridgeAuthOrderNote = "Proton Mail Bridge validates the password before the username, and returns the identical rejection for every rejected (username, password) pair — so this error is not evidence that the username is wrong. Read the real Bridge app password from the Bridge account view's mailbox-details panel, or from the Bridge CLI's info command: it is roughly 20-22 characters drawn only from A-Za-z0-9-_, and it is never your Proton account password."

const bridgeTokenShapeWarningText = "the configured token is not a Bridge-generated app password (Bridge-generated app passwords contain only the characters A-Za-z0-9-_) — " + bridgeAuthOrderNote
```

A reviewer can check the authentication-order claim ("the password is validated before the username, and one rejection covers every rejected pair") against the debug session's upstream evidence (`.planning/debug/proton-bridge-no-such-user.md`'s citations of `proton-bridge`'s `CheckAuth` and `gluon`'s `getUserID`) without opening the source.

## Old vs new live-test hint

**Old** (`live_bridge_test.go:160`, before this plan):
```
live login: %v (if this says "no such user", see 03-01-SUMMARY.md's documented Bridge-account credential finding — not a code defect)
```

**New** (this plan):
```
live login: %v (%s)
```
— with `%s` filled by `bridgeAuthOrderNote`: *"Proton Mail Bridge validates the password before the username, and returns the identical rejection for every rejected (username, password) pair — so this error is not evidence that the username is wrong. Read the real Bridge app password from the Bridge account view's mailbox-details panel, or from the Bridge CLI's info command: it is roughly 20-22 characters drawn only from A-Za-z0-9-_, and it is never your Proton account password."*

**What was removed and why:** the old text cited a planning artifact (`03-01-SUMMARY.md`) as the authority for a settled, username-side cause ("not a code defect"). The debug session (`.planning/debug/proton-bridge-no-such-user.md`) records this exact claim as a contributing cause of four rounds of misdiagnosis — Bridge checks the password before the username, so a bad password makes every username, including the correct one, return the identical error. The new text states that order correctly and points the reader at the password instead.

## Measured justification: wired into `client.connect`, not `Health`

`plugins/proton/plugin.go`'s `Health` (unedited by this plan) returns `err.Error()` from `connect` verbatim:
```go
return &webspacesv1.HealthResponse{Reachable: false, LastError: err.Error()}, nil
```
So wiring into `connect` reaches the `Health` path — the gap's third `missing` item — without touching `plugin.go` at all.

But `kernel/httpapi/sources.go`'s `sourceStatusesFrom` — read, not modified, by this plan — builds the UI's `last_error` from a different source entirely:
```go
LastError:    run.Error,
```
`run` is the kernel's own `sync_runs` history row (`store.LatestSyncRunPerSource`), never a plugin's self-reported `HealthResponse` (the A-PLUG-04 rule from 02-02). That row is populated from `Match`'s error, and `Match` (`plugin.go`) also calls `p.client.connect` and wraps the same error as `codes.Unavailable`. The UI's red-dot detail — the surface the gap was actually reported from — is therefore fed by the `Match` path, not the `Health` path. A fix placed only inside `Health` would have been correct, tested, and invisible exactly where the operator was looking. Wiring into `connect` reaches both arms from one call site.

## Immutable execution record: `03-01-SUMMARY.md` was NOT edited

`03-01-SUMMARY.md` is an immutable execution record of what plan 03-01 actually did at the time it ran. This plan does not rewrite it — doing so would falsify the history that made this gap's diagnosis reconstructible in the first place. Instead, the live coupling to that framing is severed at its one consuming site: Task 2 removes `live_bridge_test.go`'s citation of `03-01-SUMMARY.md` as the authority for a cause, replacing it with the correct, shared `bridgeAuthOrderNote` explanation. The correction itself — that the "Bridge-account credential finding" framing was a misdiagnosis, and why — is recorded here, in this SUMMARY, and in `.planning/debug/proton-bridge-no-such-user.md`, rather than by editing history.

## Files Created/Modified

- `plugins/proton/credentials.go` (new) — the shared authentication-order note, the shape-warning text, and the alphabet predicate. Zero imports.
- `plugins/proton/client.go` (modified) — `connect`'s LOGIN-failure branch appends the shape warning when non-empty; doc comment extended by one clause; nothing else in the file changed.
- `plugins/proton/credentials_test.go` (new) — the alphabet-boundary table and three `Health`-path tests.
- `plugins/proton/live_bridge_test.go` (modified) — the LOGIN-failure hint now references `bridgeAuthOrderNote`; nothing else in the file changed (6 changed diff lines total).

## Decisions Made

See `key-decisions` in the frontmatter above. In summary: wiring lives in `client.connect` (the one call site that reaches both `Health` and the `sync_runs`-fed UI surface), both constants are import-free and built by compile-time concatenation so they are provably free of runtime data, and the live-test hint now shares its text with the runtime advice by reference rather than restating it.

## Deviations from Plan

None — plan executed exactly as written. The plan's own `<action>` for Task 1 explicitly specified writing `credentials_test.go` first (RED), then `credentials.go` and the `client.go` wiring (GREEN), then the Step 4 inversion — that sequence was followed, and Task 1's single commit reflects the completed GREEN state (RED itself, being a compile failure, has nothing meaningful to commit — the plan's own Step 1 language treats the compile-failure transcript as "the RED output," not a separate commit).

## Issues Encountered

- **Grep-count acceptance criteria caught two unintended self-references.** Both `client.go`'s extended doc comment and `live_bridge_test.go`'s new inline comment initially repeated the identifier names (`bridgeTokenShapeWarning`, `bridgeAuthOrderNote`) literally, which pushed `grep -c` past the plan's exact-count acceptance criteria (1 call site in `client.go`; 1 reference in `live_bridge_test.go`). Reworded both comments to describe the mechanism without repeating the identifier a second time; re-verified both counts and re-ran the full suite after each edit. No behavior change, comment-only.
- **Frontend suite could not be executed in this isolated worktree.** `web/node_modules` is not installed here (git worktrees do not carry untracked directories, and this worktree was never `npm install`-ed). This is a pre-existing environment limitation, not caused by this plan's changes — this plan touches zero files under `web/`, confirmed by `git diff --exit-code web/` (clean). Recorded as `coverage.D7` with `human_judgment: true` so a human/CI run with `node_modules` present spot-checks it; nothing this plan built should be affected, since no frontend code changed.

## User Setup Required

None — no external service configuration required. (The unrelated, still-outstanding `.env` credential correction is a pre-existing, out-of-scope user action — see "Still outstanding" below, not new setup this plan introduces.)

## Verification Results (plan-level, all 6 items from `<verification>`)

1. **`cd plugins/proton && go test ./... -count=1 -v`** — every pre-existing test plus the four new ones pass; `TestSeenFlagUnchanged_LiveBridge` reports SKIP. Quoted individually above under GREEN.
2. **Repo-root + all five module builds** — `CGO_ENABLED=0 go build ./... && go vet ./... && go test ./... -count=1` at repo root, plus `go build ./... && go test ./... -count=1` in `sdk`, `plugins/paperless`, `plugins/silverbullet`, `plugins/mock` — all green (see command outputs run during execution; every package reported `ok` or `no test files`).
3. **Frontend** — could not run `npm --prefix web run test`/`run check` in this worktree (see Issues Encountered); `git diff --exit-code web/` is clean, confirming no frontend file was touched by this plan.
4. **End-to-end diagnostic path, traced by reading** — confirmed both arms: (a) `client.connect` LOGIN failure -> wrapped error carries `bridgeTokenShapeWarningText` -> `plugin.go`'s `Health` returns it as `HealthResponse.LastError` verbatim; (b) the same `connect` error -> `Match` wraps as `codes.Unavailable` -> (by the kernel's own established sync_runs-recording path, per `kernel/httpapi/sources.go`'s doc comment and A-PLUG-04) -> `sourceStatusesFrom` copies `run.Error` into `last_error`. Both arms are walkable in the code; see the "Measured justification" section above for the exact quoted lines.
5. **Anti-drift link, confirmed by reading** — `live_bridge_test.go`'s hint references `bridgeAuthOrderNote` (`grep -c` = 1), and `bridgeTokenShapeWarningText` is built from that same constant by compile-time concatenation in `credentials.go`.
6. **Never-a-gate property, confirmed by reading** — `grep -n bridgeTokenShapeWarning plugins/proton/client.go` shows exactly one occurrence, inside the branch guarded by `if err := conn.Login(...); err != nil`, and zero occurrences in `NewClient` or `realDial`.

## Still outstanding and NOT closed by this plan

(Restated verbatim from this plan's own `<verification>` section, per the plan's explicit instruction, so re-verification does not mistake any of this for closed or for new.)

- **The user-side credential correction, gap G-03-1's first `missing` item, is unchanged and is the actual unblocker.** On monroe, read the real Bridge credentials (Bridge GUI -> account -> Mailbox details, or `protonmail-bridge --cli` then `info`; sign in first if signed out), replace `PROTON_BRIDGE_PASS` in `.env` with the ~20-22 character `A-Za-z0-9-_` app password, wait out Bridge's login jail (`too many login attempts`), restart the kernel, and re-run the live test. Nothing in this plan performs, simulates, or verifies any part of that.
- **Two user-side follow-ups from the debug session's `blind_spots`, to check only if the password correction alone does not resolve it:** (1) whether the account is currently signed in on monroe's Bridge — a signed-out account produces the identical rejection; (2) whether Bridge is in combined or split address mode, which restricts which address may authenticate. Neither is observable from this repository.
- **UAT Tests 1, 2 and 3 remain `issue`/`blocked` until that correction lands.** This plan improves the message the failure produces; it cannot make the failure stop happening. `03-UAT.md`'s Test 4 (`pass`, with its email-hits caveat) is unaffected.
- **Also deliberately not closed, owners named:** `03-01-SUMMARY.md`'s framing (owner: this plan's SUMMARY, which records the correction rather than rewriting history — see "Immutable execution record" above) and `.planning/REQUIREMENTS.md` status bookkeeping for SRC-01 (owner: the re-verify/seal step, per the 03-05/03-06/03-07 precedent).

## Next Phase Readiness

- The two code-side items of gap G-03-1 are closed: the runtime diagnostic and the live-test hint both now correctly identify a shape-suspect password and point at the right knob.
- Phase 03 is NOT ready to seal on SRC-01 until the user replaces `PROTON_BRIDGE_PASS` and UAT Tests 1-3 are re-run against a real, correctly-shaped credential — that is unchanged by this plan and is entirely a user action.
- No new blockers introduced. No dependency, schema, wire-contract, or frontend change in this plan — a single `git revert` per commit would restore the prior behavior exactly.

---
*Phase: 03-email-in-the-webspace*
*Completed: 2026-08-01*
