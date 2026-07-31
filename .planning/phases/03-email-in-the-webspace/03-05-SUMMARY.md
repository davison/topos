---
phase: 03-email-in-the-webspace
plan: 05
subsystem: api
tags: [go, imap, go-imap, emersion, tdd, gap-closure]

# Dependency graph
requires:
  - phase: 03-01
    provides: "plugins/proton's SourcePlugin, Match/Fetch/Health over IMAP, the matched struct and toItem"
  - phase: 03-02
    provides: "imap_transcript_test.go's newTestIMAPServer fixture and the recording-relay wire-level proof"
provides:
  - "Item.TimestampUnix populated from the message's IMAP INTERNALDATE, with a zero-value guard"
  - "The empty-Message-Id skip is counted AND logged (count-only, no content) to SourcePlugin.logOut"
  - "plugins/proton/client.go carries no textual match for the TLS verification opt-out field"
  - "Three new regression tests preventing recurrence of the silently-dropped-field class of bug"
affects: [03-verification, requirements-src-01]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Zero-time.Time guard before Unix() conversion, mirroring the existing SecondaryTimestampUnix guard, to avoid the -62135596800 sentinel collision"
    - "Struct field substitution seam for a log sink (logOut io.Writer), mirroring the existing Client.dial substitution seam"

key-files:
  created:
    - plugins/proton/item_test.go
  modified:
    - plugins/proton/plugin.go
    - plugins/proton/imap_transcript_test.go
    - plugins/proton/client.go

key-decisions:
  - "Task 1's zero-guard test (TestToItem_ZeroInternalDateYieldsZeroTimestamp) cannot go RED against the truly unmodified plugin.go, because TimestampUnix defaults to Go's int64 zero value (0) whether or not the field is wired at all — the pre-fix and correctly-guarded outputs are identical (0). To produce a genuine, meaningful RED signal for this specific test, the internalDate field and an UNGUARDED assignment (TimestampUnix: m.internalDate.Unix()) were added first, observed to fail exactly as the plan predicted (-62135596800), then the IsZero() guard was added to reach GREEN. This is documented here rather than silently reported as 'both tests went red against the original code', which would be inaccurate."

requirements-completed: [SRC-01]

coverage:
  - id: D1
    description: "Every Proton email item's Item.TimestampUnix equals its IMAP INTERNALDATE's Unix seconds; SecondaryTimestampUnix independently carries the envelope Date"
    requirement: "SRC-01"
    verification:
      - kind: unit
        ref: "plugins/proton/item_test.go#TestMatch_ItemTimestampIsInternalDate"
        status: pass
    human_judgment: false
  - id: D2
    description: "A zero/absent INTERNALDATE yields TimestampUnix 0, never a negative sentinel"
    requirement: "SRC-01"
    verification:
      - kind: unit
        ref: "plugins/proton/item_test.go#TestToItem_ZeroInternalDateYieldsZeroTimestamp"
        status: pass
    human_judgment: false
  - id: D3
    description: "An empty-Message-Id skip is counted and logged as a count-only line, with the skipped message's own content never appearing in the log"
    requirement: "SRC-01"
    verification:
      - kind: unit
        ref: "plugins/proton/item_test.go#TestMatch_EmptyMessageIDSkipIsLogged"
        status: pass
    human_judgment: false
  - id: D4
    description: "plugins/proton has no textual match for the TLS verification opt-out field, and TLS configuration (RootCAs, ServerName) is unchanged"
    requirement: "SRC-01"
    verification:
      - kind: other
        ref: "! grep -rq 'InsecureSkipVerify' plugins/proton/"
        status: pass
    human_judgment: false
  - id: D5
    description: "Every Go and frontend suite in the repository passes, live-Bridge test skips cleanly"
    verification:
      - kind: unit
        ref: "CGO_ENABLED=0 go test ./... -count=1 (repo root); go test ./... -count=1 (plugins/proton, plugins/paperless); go test ./internal/audit/... -count=1; npm --prefix web run test"
        status: pass
    human_judgment: false

duration: 21min
completed: 2026-07-31
status: complete
---

# Phase 03 Plan 05: INTERNALDATE Gap Closure Summary

**Wired the already-fetched IMAP INTERNALDATE through to `Item.TimestampUnix` with a zero-value guard, made the counted empty-Message-Id skip actually log, and closed the TLS-comment anti-pattern — the one confirmed code-level BLOCKER from 03-VERIFICATION.md.**

## Performance

- **Duration:** 21 min
- **Started:** 2026-07-31T14:44:00Z (approx.)
- **Completed:** 2026-07-31T15:48:52Z
- **Tasks:** 3
- **Files modified:** 4 (1 new: `item_test.go`)

## Accomplishments

- `plugins/proton/plugin.go`'s `matched` struct gained an `internalDate time.Time` field, populated from `msg.InternalDate` in `Match`'s fetch loop and preserved (not overwritten) by the dedup-merge branch, exactly like `mailbox` and `envelope`.
- `toItem` now assigns a guarded primary timestamp to `Item.TimestampUnix` — `m.internalDate.Unix()` when non-zero, `0` otherwise — mirroring the existing `SecondaryTimestampUnix` guard pattern. Every Proton email now interleaves chronologically with paperless-ngx and SilverBullet items instead of bucketing at the oldest end of the stream at `timestamp_unix = 0`.
- The previously counted-but-discarded `skippedNoMessageID` is now written as a single count-only log line (`webspaces-plugin-proton: match: skipped N message(s) with no Message-Id header`) to a new `SourcePlugin.logOut` field (`os.Stderr` in production, substitutable in tests) — closing 03-01's half-implemented "counted, logged skip" must-have.
- `plugins/proton/client.go`'s `bridgeCertServerName` doc comment restated to assert the no-verification-bypass guarantee by concept, not by naming the TLS opt-out field — `plugins/proton` now has zero textual matches for it, closing 03-VERIFICATION.md's Anti-Patterns row.
- Three new regression tests, each observed failing before its fix landed (see RED evidence below).

## Task Commits

Each task was committed atomically:

1. **Task 1: INTERNALDATE reaches Item.TimestampUnix — test-first** - `2636c12` (feat)
2. **Task 2: the counted empty-Message-Id skip is actually logged** - `3a51100` (feat)
3. **Task 3: TLS-comment hygiene and full-suite regression green** - `e0d0405` (docs)

**Plan metadata:** (this commit, docs: complete plan)

## Files Created/Modified

- `plugins/proton/plugin.go` - Added `matched.internalDate`, `SourcePlugin.logOut`, the guarded `TimestampUnix` assignment in `toItem`, and the guarded skip-count log line replacing the blank-identifier discard
- `plugins/proton/imap_transcript_test.go` - Added package-level `seedInternalDate`/`seedEnvelopeDate` (replacing the zero-`time.Time` seed trap), a third `Labels/NoMessageID` mailbox with a Message-Id-less message
- `plugins/proton/item_test.go` (new) - `TestMatch_ItemTimestampIsInternalDate`, `TestToItem_ZeroInternalDateYieldsZeroTimestamp`, `TestMatch_EmptyMessageIDSkipIsLogged`, plus the shared `newTestPluginDialingServer` test helper
- `plugins/proton/client.go` - Comment-only: restated `bridgeCertServerName`'s closing parenthetical without naming the TLS verification opt-out field

## RED Evidence (TDD)

**Task 1, first RED (against the fully unmodified `plugin.go`):**
```
=== RUN   TestMatch_ItemTimestampIsInternalDate
    item_test.go:61: item.TimestampUnix = 0, want 1463044530 (seedInternalDate.Unix())
--- FAIL: TestMatch_ItemTimestampIsInternalDate (0.00s)
=== RUN   TestToItem_ZeroInternalDateYieldsZeroTimestamp
--- PASS: TestToItem_ZeroInternalDateYieldsZeroTimestamp (0.00s)
FAIL
```
`TestToItem_ZeroInternalDateYieldsZeroTimestamp` passed here for an uninteresting reason: `TimestampUnix` is entirely unset in unmodified code, so it defaults to Go's zero value (0) — identical to the guarded, correct output. This assertion cannot distinguish "never wired" from "wired and correctly guarded" at this point in the fixture's evolution. See Key Decision above.

**Task 1, second RED (deliberately, against a temporary UNGUARDED implementation — `TimestampUnix: m.internalDate.Unix()`, no `IsZero()` check — to produce a meaningful failure for the guard test specifically):**
```
=== RUN   TestMatch_ItemTimestampIsInternalDate
--- PASS: TestMatch_ItemTimestampIsInternalDate (0.00s)
=== RUN   TestToItem_ZeroInternalDateYieldsZeroTimestamp
    item_test.go:98: item.TimestampUnix = -62135596800, want 0 for a zero internalDate (got the zero time.Time's negative Unix seconds instead of the guarded sentinel)
--- FAIL: TestToItem_ZeroInternalDateYieldsZeroTimestamp (0.00s)
FAIL
```
This is exactly the failure mode the plan's Step 5 warned about (`-62135596800`, a year-1 date). The `IsZero()` guard was then added, reaching GREEN for both tests.

**Task 2 RED (build failure — `logOut` field did not yet exist):**
```
# github.com/davison/webspaces/plugins/proton [github.com/davison/webspaces/plugins/proton.test]
./item_test.go:115:9: plugin.logOut undefined (type *SourcePlugin has no field or method logOut)
FAIL	github.com/davison/webspaces/plugins/proton [build failed]
```

**All three new tests GREEN together** (with the shared-fixture regression test `TestIMAPTranscript_ExamineAndPeekOnly` and the AST scan `TestPluginIssuesNoIMAPMutatingCommands`), confirmed after each task's fix.

## Decisions Made

See `key-decisions` in frontmatter: the zero-guard test's RED signal was produced deliberately against a temporary unguarded implementation, not the original code, because the original code's output (0, by omission) is bit-identical to the correctly-guarded output (0, by design) — the two states are indistinguishable through this assertion alone. This is documented transparently rather than glossed over.

## Deviations from Plan

None beyond the RED-sequencing note above, which is a testing-methodology clarification, not a code deviation — every acceptance criterion in the plan (grep checks, test names, go.mod/go.sum untouched, comment-only `client.go` diff, full-suite green) was met exactly as specified.

## Issues Encountered

None.

## Verification (phase-level, from the plan's `<verification>` block)

1. `cd plugins/proton && go test ./... -count=1 -v` — **14 tests pass** (11 pre-existing + 3 new), **1 skip** (`TestSeenFlagUnchanged_LiveBridge`, environment-blocked). Matches exactly.
2. `CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go vet ./... && CGO_ENABLED=0 go test ./... -count=1` from the repo root — clean (all packages `ok` or `[no test files]`).
3. `npm --prefix web run test` — **69/69** passed, unchanged from 03-VERIFICATION.md's recorded count.
4. `! grep -rq 'InsecureSkipVerify' plugins/proton/` — passes; 03-01-PLAN.md's own acceptance criterion now holds on its own literal terms.
5. `cd plugins/paperless && go test ./... -count=1` — passes (sibling plugin unaffected).
6. `go test ./internal/audit/... -count=1` — passes (egress/read-only audit unaffected).
7. Traced the closed gap end to end by reading, not running: `plugins/proton/plugin.go` `toItem` sets `TimestampUnix` → `kernel/item/item.go` `FromProto` copies it verbatim → `kernel/index/store.go` writes `items.timestamp_unix` and orders the stream by it (already covered by existing `TestStreamItems_*` tests, unmodified by this plan) → `web/src/lib/components/StreamRow.svelte` renders `formatItemDate(item.timestamp_unix)`. Every link in that chain except the first was already verified WIRED in 03-VERIFICATION.md; this plan repaired only the first.

## Still Outstanding — NOT Closed by This Plan (verbatim from 03-VERIFICATION.md, unchanged, environment-blocked)

All four `human_verification` items require a live, currently-authenticating Proton Mail Bridge account. The Bridge account rejected LOGIN with "no such user" and then rate-limited (03-01-SUMMARY.md, "Notable Live-Environment Finding"). That is a credential/environment correction, not a code defect, and remains out of scope here:

1. **Live browser walkthrough of a real email in the stream and detail pane** (WINDOWS.md item 1) — *"Open a webspace with a real, currently-reachable Proton Bridge account configured. Confirm at least one real Proton email appears in the stream interleaved with paperless/SilverBullet items, showing sender before the date, with the subject as the row title. Click it: the detail pane opens and shows the message's body (plain text, or formatted HTML with headings/links/colors if the message has an HTML part)."* Expected: *"The email is visible, correctly dated once the TimestampUnix gap above is fixed, and its body renders in the detail pane."* — the date-correctness half of this expectation is now mechanically satisfied by this plan; the live visual walkthrough itself is still unrun.
2. **Run `TestSeenFlagUnchanged_LiveBridge` to a PASS against the real Bridge** (WINDOWS.md item 2) — *"Run `WEBSPACES_PROTON_LIVE_IT=1 PROTON_BRIDGE_ADDR=<addr> PROTON_BRIDGE_USER=<user> PROTON_BRIDGE_PASS=<pass> go test -run TestSeenFlagUnchanged_LiveBridge -v ./plugins/proton/...` against the real Bridge account once its credential issue is corrected."* Expected: *"The test passes, directly proving SRC-01's second success criterion (\Seen flag unchanged across a full Match+Fetch cycle) against the real mailbox, not just the no-live-Bridge wire-transcript proxy for it."*
3. **Confirm in the real Proton client that an opened email stays unread** (WINDOWS.md item 3) — *"After running a webspaces sync and opening an email in the detail pane, check the same email in the real Proton web or mobile client and confirm it is still shown as unread there."* Expected: *"The email remains unread in Proton's own client — the direct, human-observable proof of the never-mark-read guarantee end to end."*
4. **Live browser walkthrough of the search UI** (03-04-PLAN.md Task 1 human-check) — *"Open a webspace in the browser, type a word present in a document, a note, and an email into the search box, and confirm ranked cross-source results appear with the matched word emboldened, a result opens the detail pane on click, clearing restores the unfiltered stream, and a nonsense or malformed (lone double-quote) query shows the no-matches state rather than an error."* Expected: *"All of 03-04-PLAN.md's Task 1 human-check behaviors hold visually in a running browser."*

Once the Bridge credential is corrected, the live walkthrough will additionally confirm this fix against real mail.

## Requirements Bookkeeping — Deliberately Not Touched

`.planning/REQUIREMENTS.md` checkbox/status bookkeeping for SRC-01 and KERN-05 is **deliberately not touched by this plan** — that is owned by the re-verify/seal step, not by an execution plan. Doing it here would double-book it.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The one confirmed code-level BLOCKER from 03-VERIFICATION.md is closed: `Item.TimestampUnix` is now correctly populated from INTERNALDATE with a safe zero-guard.
- The related minor issue (unlogged skip count) is closed.
- The TLS-comment anti-pattern is closed.
- Re-verification (`/gsd-plan-phase --research-phase 3` or the phase's verify step) should re-run 03-VERIFICATION.md's checks; all code-level gaps are now closed, leaving only the four environment-blocked `human_verification` items, unchanged and out of this plan's scope.

## Self-Check: PASSED

- FOUND: `plugins/proton/plugin.go`
- FOUND: `plugins/proton/client.go`
- FOUND: `plugins/proton/imap_transcript_test.go`
- FOUND: `plugins/proton/item_test.go`
- FOUND: commit `2636c12` (Task 1)
- FOUND: commit `3a51100` (Task 2)
- FOUND: commit `e0d0405` (Task 3)

---
*Phase: 03-email-in-the-webspace*
*Completed: 2026-07-31*
