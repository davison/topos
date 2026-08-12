---
phase: 03-email-in-the-webspace
plan: 06
subsystem: sync
tags: [imap, proton, go, svelte, date-formatting, gap-closure]

# Dependency graph
requires:
  - phase: 03-01
    provides: "normalizeMessageID / encodeSourceID contract for source_id identity; the mailboxCache field and Fetch-time mailbox lookup pattern this plan repairs"
  - phase: 03-02
    provides: "PLAIN/HTML body extraction (PlainTextPart/HTMLPart) fetchFull consumes"
  - phase: 03-05
    provides: "seedInternalDate/seedEnvelopeDate fixture consts and the item timestamp regression tests this plan's fixture change must not break"
provides:
  - "mergeMailboxCache: an under-lock, entry-wise upsert replacing the whole-map cache setter, so the plugin's source_id-to-mailbox resolution state accumulates across every webspace's Match call within a sync cycle instead of being discarded by the most recently processed one"
  - "A fourth seeded IMAP fixture mailbox (Labels/GammaTeam) with a distinct Message-Id, letting tests express two Match calls discovering disjoint source_ids"
  - "Two Go regression tests proving accumulation survives both a second webspace's Match and a zero-mailbox-matched Match"
  - "DetailPane.svelte renders its header date through the shared UTC-pinned formatItemDate instead of a locally declared, un-pinned toLocaleDateString formatter"
  - "A frontend source-scan guard test preventing any first-party component from reintroducing a local date renderer"
affects: [email-plugin, detail-pane, multi-webspace-sync]

tech-stack:
  added: []
  patterns:
    - "In-process plugin state that spans multiple sequential RPC calls (Match called once per configured webspace against one long-lived instance) must be upserted, never replaced wholesale"
    - "Frontend source-scan tests (reading component source text off disk via node:fs) as a lightweight guard against renderer drift, when no component-mount harness exists"

key-files:
  created:
    - plugins/proton/mailbox_cache_test.go
    - web/src/lib/components/date-format.test.ts
    - web/src/lib/node-builtins.d.ts
  modified:
    - plugins/proton/plugin.go
    - plugins/proton/imap_transcript_test.go
    - web/src/lib/components/DetailPane.svelte

key-decisions:
  - "mergeMailboxCache is last-writer-wins per key (not insert-only), so a message that has moved mailboxes is refreshed by whichever Match rediscovers it"
  - "Zero-mailboxes-matched Match path performs no cache mutation at all (previously reset the cache to an empty map)"
  - "Added web/src/lib/node-builtins.d.ts (not in the plan's files_modified list) — a narrow ambient type declaration for node:fs/node:path/node:url so svelte-check reports 0 ERRORS without installing @types/node, preserving the plan's no-new-dependency constraint"

patterns-established:
  - "Pattern: accumulate-across-calls in-process caches use an entry-wise merge method (never re-bind the field to a new map) plus an explicit no-op zero-result branch"

requirements-completed: []  # SRC-01 bookkeeping deliberately NOT marked here — see 'REQUIREMENTS.md bookkeeping' note below; owned by the re-verify/seal step.

coverage:
  - id: D1
    description: "One plugin instance serving two webspaces with disjoint keyword sets can Fetch an item discovered by either of them; a Match matching zero mailboxes erases nothing"
    requirement: "SRC-01"
    verification:
      - kind: unit
        ref: "plugins/proton/mailbox_cache_test.go#TestMatch_MailboxCacheSurvivesASecondWebspaceMatch"
        status: pass
      - kind: unit
        ref: "plugins/proton/mailbox_cache_test.go#TestMatch_ZeroMailboxMatchPreservesMailboxCache"
        status: pass
    human_judgment: false
  - id: D2
    description: "Detail pane header date renders through the same UTC-pinned formatItemDate the stream row uses, so both surfaces agree on one item's calendar day"
    requirement: "SRC-01"
    verification:
      - kind: unit
        ref: "web/src/lib/components/date-format.test.ts"
        status: pass
    human_judgment: false

duration: 20min
completed: 2026-07-31
status: complete
---

# Phase 03 Plan 06: Multi-Webspace Mailbox Cache & Detail Pane Date Fix Summary

**Inverted the Proton plugin's per-Match whole-map cache replace into an entry-wise upsert so every configured webspace's discoveries stay Fetch-able, and pointed DetailPane.svelte at the shared UTC-pinned formatItemDate instead of its own un-pinned toLocaleDateString formatter.**

## Performance

- **Duration:** ~20 min
- **Started:** 2026-07-31T15:44:00Z (approx)
- **Completed:** 2026-07-31T15:49:57Z
- **Tasks:** 2 completed
- **Files modified:** 6 (3 created, 3 modified)

## Accomplishments

- Closed the confirmed BLOCKER (03-REVIEW.md CR-01 / 03-VERIFICATION.md gap): the Proton plugin's `mailboxCache` is now merged entry-wise on every `Match` call instead of replaced wholesale, so opening an email from any webspace but the last one processed in a sync cycle no longer fails with `codes.NotFound`.
- Closed the WARNING (03-REVIEW.md WR-01): the detail pane header date now comes from the same `formatItemDate` the stream row uses, so the same item's date can no longer show a different calendar day between the two surfaces for a viewer west of UTC.
- Locked both fixes behind regression tests observed failing for the right reason (the plugin's `codes.NotFound` status; a source-scan assertion naming `DetailPane.svelte`) before they were made to pass.
- `kernel/correlate/correlate.go` and `kernel/pluginhost/host.go` are unmodified — confirmed by `git diff --exit-code`, per the plan's own instruction that the kernel-side per-webspace loop was already correctly scoped.

## Task Commits

Each task was committed atomically:

1. **Task 1: the Proton plugin's mailbox resolution state accumulates across webspaces — test-first** - `f3bac4f` (fix)
2. **Task 2: the detail pane's date comes from the one UTC-pinned formatter — and cannot drift again** - `3ac0553` (fix)

**Plan metadata:** (this commit, following SUMMARY.md write)

_Note: both tasks were `tdd="true"` — each commit bundles the RED test file(s) and the GREEN production fix together in one atomic per-task commit, per this plan's own task granularity (the plan did not request separate test/feat commits)._

## Files Created/Modified

- `plugins/proton/plugin.go` - `mergeMailboxCache` (entry-wise upsert) replaces `setMailboxCache` (whole-map replace); zero-mailboxes-matched `Match` branch no longer mutates the cache; `mailboxCache` field doc comment rewritten to record the per-configured-webspace call pattern, bounded growth, and stale-entry safety
- `plugins/proton/imap_transcript_test.go` - Seeded a fourth fixture mailbox `Labels/GammaTeam` with a distinct Message-Id (`gammaMessageID`); extracted `sharedMessageID`/`gammaMessageID` package-level consts
- `plugins/proton/mailbox_cache_test.go` - New: `TestMatch_MailboxCacheSurvivesASecondWebspaceMatch`, `TestMatch_ZeroMailboxMatchPreservesMailboxCache`
- `web/src/lib/components/DetailPane.svelte` - Imports and calls `formatItemDate` from `$lib/format`; deleted the local `toLocaleDateString`-based `formatDate` helper
- `web/src/lib/components/date-format.test.ts` - New: source-scan guard over top-level `web/src/lib/components/*.svelte`
- `web/src/lib/node-builtins.d.ts` - New: narrow ambient type declarations for `node:fs`/`node:path`/`node:url` (see Deviations)

## Decisions Made

- `mergeMailboxCache`'s merge semantics are last-writer-wins per key, not insert-only — a message that has moved mailboxes since it was last discovered is refreshed by whichever `Match` rediscovers it, rather than being pinned to a mailbox it has left.
- No time- or cycle-based eviction sweep was added to the cache, per the plan's explicit instruction: `Match` has no notion of sync-cycle identity, so any per-call expiry would reintroduce the exact per-webspace scoping error this plan fixes.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 3 - Blocking] Added `web/src/lib/node-builtins.d.ts` to satisfy `svelte-check`'s `0 ERRORS` acceptance criterion**
- **Found during:** Task 2, Step 4 (`npm --prefix web run check`)
- **Issue:** `date-format.test.ts` imports `node:fs`, `node:path`, and `node:url`. No `@types/node` package is installed anywhere in `web/` (confirmed via `npm ls @types/node` and a `node_modules` search) even though it is listed as an optional peer of a transitive dependency in `package-lock.json` — it was never actually installed. Without it, `svelte-check` reported 4 `ERROR`s ("Cannot find name 'node:fs'" etc.), failing the plan's own `0 ERRORS` acceptance criterion.
- **Fix:** Added a narrow ambient module declaration file (`web/src/lib/node-builtins.d.ts`) declaring only the exact function signatures the test uses (`readFileSync`, `readdirSync`, `dirname`, `join`, `fileURLToPath`). This is a new project source file, not a package install — `web/package.json` and `web/package-lock.json` are untouched, preserving the plan's explicit no-new-dependency constraint (and its threat register's T-03-06-SC disposition).
- **Files modified:** `web/src/lib/node-builtins.d.ts` (new)
- **Verification:** `npm --prefix web run check` reports `0 ERRORS 1 WARNINGS` (the pre-existing `SearchBox.svelte` warning, untouched)
- **Committed in:** `3ac0553` (Task 2 commit)

---

**Total deviations:** 1 auto-fixed (1 blocking, Rule 3; no package installed)
**Impact on plan:** Necessary to satisfy the plan's own `0 ERRORS` acceptance criterion without violating its no-new-dependency constraint. No scope creep — the file is scoped to exactly the three functions the guard test calls.

## Issues Encountered

None beyond the deviation above.

## RED/GREEN Evidence

### Task 1 — Go regression tests (RED, before the fix)

```
=== RUN   TestMatch_MailboxCacheSurvivesASecondWebspaceMatch
    mailbox_cache_test.go:70: Fetch(firstSourceID) after a second webspace's Match: got error rpc error: code = NotFound desc = proton: source_id "c2hhcmVkLW1lc3NhZ2VAZXhhbXBsZS5jb20" is not known — the index has not been synced since this plugin started (code NotFound), want nil — the plugin's mailbox resolution state must accumulate across Match calls, not be replaced by the most recent one
--- FAIL: TestMatch_MailboxCacheSurvivesASecondWebspaceMatch (0.00s)
=== RUN   TestMatch_ZeroMailboxMatchPreservesMailboxCache
    mailbox_cache_test.go:127: Fetch(firstSourceID) after a zero-mailbox-matched Match: got error rpc error: code = NotFound desc = proton: source_id "c2hhcmVkLW1lc3NhZ2VAZXhhbXBsZS5jb20" is not known — the index has not been synced since this plugin started (code NotFound), want nil — a webspace contributing nothing must never erase what an earlier webspace's Match contributed
--- FAIL: TestMatch_ZeroMailboxMatchPreservesMailboxCache (0.00s)
FAIL
FAIL	github.com/davison/webspaces/plugins/proton	0.008s
```

Both failures came from the Fetch assertion, carrying the plugin's `NotFound` status, exactly as required.

### Task 1 — GREEN (after the fix)

```
=== RUN   TestIMAPTranscript_ExamineAndPeekOnly
--- PASS: TestIMAPTranscript_ExamineAndPeekOnly (0.00s)
=== RUN   TestMatch_ItemTimestampIsInternalDate
--- PASS: TestMatch_ItemTimestampIsInternalDate (0.00s)
=== RUN   TestMatch_EmptyMessageIDSkipIsLogged
--- PASS: TestMatch_EmptyMessageIDSkipIsLogged (0.00s)
=== RUN   TestMatch_MailboxCacheSurvivesASecondWebspaceMatch
--- PASS: TestMatch_MailboxCacheSurvivesASecondWebspaceMatch (0.00s)
=== RUN   TestMatch_ZeroMailboxMatchPreservesMailboxCache
--- PASS: TestMatch_ZeroMailboxMatchPreservesMailboxCache (0.00s)
PASS
ok  	github.com/davison/webspaces/plugins/proton	0.010s
```

Full plugin suite: 16 tests pass, 1 skip (`TestSeenFlagUnchanged_LiveBridge`, environment-blocked, unchanged).

### Task 2 — Frontend guard test (RED, before the fix)

```
 FAIL  src/lib/components/date-format.test.ts > date-format source-scan guard > contains no locally-declared, un-pinned date formatter (toLocaleDateString)
AssertionError: DetailPane.svelte calls toLocaleDateString directly — item dates must render through the shared, UTC-pinned formatItemDate from $lib/format instead, so the calendar day can never disagree between surfaces: expected true to be false

 FAIL  src/lib/components/date-format.test.ts > date-format source-scan guard > imports formatItemDate in every component that renders timestamp_unix
AssertionError: DetailPane.svelte references timestamp_unix but does not import formatItemDate from $lib/format — an item date must always render through the shared UTC-pinned formatter: expected false to be true

 Test Files  1 failed | 3 passed (4)
      Tests  2 failed | 70 passed (72)
```

Both failures named `DetailPane.svelte` explicitly.

### Task 2 — GREEN (after the fix)

```
 Test Files  4 passed (4)
      Tests  72 passed (72)
```

(69 pre-existing tests + the new file's 3 tests.) `npm --prefix web run check`: `COMPLETED 746 FILES 0 ERRORS 1 WARNINGS 1 FILES_WITH_PROBLEMS` (the single pre-existing `SearchBox.svelte` `state_referenced_locally` warning, untouched).

## Full Regression Set (Task 2, Step 5)

- Repo root: `CGO_ENABLED=0 go build ./...` — clean. `CGO_ENABLED=0 go vet ./...` — clean. `CGO_ENABLED=0 go test ./... -count=1` — all packages `ok` (cmd/webspaces, kernel/webui, kernel/pluginhost, kernel/item have no test files).
- `plugins/proton`: `go build ./... && go vet ./... && go test ./... -count=1` — `ok`, 16 pass / 1 skip (`TestSeenFlagUnchanged_LiveBridge`, live-Bridge, environment-blocked).
- `plugins/paperless`: `go test ./... -count=1` — `ok`.
- `go test ./internal/audit/... -count=1` — `ok`.
- `npm --prefix web run test` — 72/72 pass. `npm --prefix web run check` — `0 ERRORS 1 WARNINGS`.

## Still Outstanding — NOT Closed by This Plan (environment-blocked, verbatim per plan's Output spec)

The four `human_verification` items in 03-VERIFICATION.md all require a live, currently-authenticating Proton Mail Bridge account. The Bridge account rejected LOGIN with "no such user" and then rate-limited (03-01-SUMMARY.md, "Notable Live-Environment Finding"), unchanged across two verification passes. That is a credential/environment correction, not a code defect, and is out of scope here. Once the credential is corrected, the first of those items additionally confirms this fix against real mail — specifically its second clause, "if more than one configured webspace matches Proton mail, confirm the SAME check holds for a webspace that is NOT the last one synced in that cycle."

Also unchanged by design: truth 2 of the phase (`\Seen` unchanged) remains PRESENT_BEHAVIOR_UNVERIFIED for the same live-Bridge reason; this plan issues no new IMAP command and changes no IMAP call path, so `TestPluginIssuesNoIMAPMutatingCommands` and `TestIMAPTranscript_ExamineAndPeekOnly` remain the standing proof and both still pass (confirmed above).

## kernel/correlate/correlate.go — Deliberately Unmodified

`kernel/correlate/correlate.go`'s `SyncSource` loops `e.Config.Webspaces`, calling `src.Match` once per webspace and persisting via `Store.ReplaceWebspaceSourceItems` — 03-VERIFICATION.md's artifact analysis confirmed this per-webspace loop and its persistence are correctly scoped; only the plugin's own in-memory `mailboxCache` state was not. `git diff --exit-code kernel/correlate/correlate.go kernel/pluginhost/host.go` reports no changes, confirming the kernel side was left untouched as instructed.

## REQUIREMENTS.md / COVERAGE.md Bookkeeping

`.planning/REQUIREMENTS.md` checkbox/status bookkeeping for SRC-01 and KERN-05 is deliberately NOT touched by this plan and is owned by the re-verify/seal step (03-05 precedent — doing it here would double-book it). `COVERAGE.md` is likewise unchanged: this plan integrates no new external API surface (an internal in-memory map fix and a frontend import change), so no matrix row is invalidated.

## Known Stubs

None.

## Threat Flags

None beyond what the plan's own `<threat_model>` already registers (T-03-06-01 through T-03-06-05, T-03-06-SC) — no new security-relevant surface was introduced beyond what that block already accounts for. The one file added outside the plan's `files_modified` list (`web/src/lib/node-builtins.d.ts`) is a compile-time-only ambient type declaration with no runtime code and no new sink.

## Out-of-Scope Pre-Existing Item (not fixed, logged only)

`git status` at the start of this session already showed `kernel/webui/build/.gitkeep` as deleted, predating this plan's work and unrelated to any file this plan touches. Left untouched per the executor's scope boundary (pre-existing, unrelated-file changes are out of scope for this plan).

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- The multi-webspace email-body-fetch BLOCKER and the detail-pane date-agreement WARNING from 03-VERIFICATION.md are both closed and locked by regression tests.
- Live-Bridge confirmation of the multi-webspace fix (03-VERIFICATION.md's first `human_verification` item, second clause) remains blocked on the Proton Bridge credential issue recorded since 03-01 — unchanged, tracked, not a code defect.
- Phase 03 re-verification / seal should re-run `.planning/REQUIREMENTS.md` bookkeeping for SRC-01 and KERN-05, which this plan deliberately left untouched.

## Self-Check: PASSED

All created files verified present on disk (`plugins/proton/mailbox_cache_test.go`, `web/src/lib/components/date-format.test.ts`, `web/src/lib/node-builtins.d.ts`, this SUMMARY.md). Both task commit hashes (`f3bac4f`, `3ac0553`) verified present in `git log`.

---
*Phase: 03-email-in-the-webspace*
*Completed: 2026-07-31*
