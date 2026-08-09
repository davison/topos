---
phase: 07-webspace-builder-ui
verified: 2026-08-09T13:02:11Z
status: human_needed
score: 111/152 must-haves verified (independently re-run in this session)
behavior_unverified: 41
overrides_applied: 0
re_verification:
  previous_status: human_needed
  previous_score: 73/107
  gaps_closed:
    - "G-07-3 (blocker): create-webspace modal's PUT /api/config always rejected the fresh empty shell. Closed by 07-11: kernel/config/types.go now declares Webspace.IsEmptyShell (three-condition discriminator — no keywords, no sources, no match); kernel/config/config.go's validateWebspaces short-circuits on it at line 413, strictly before validateFallbackCoverage (line 433) and the keywords-or-match rejection (line 418), confirmed by direct line-order read. kernel/correlate/correlate.go's matchFieldsFor now returns non-participation (not an all-empty field map) when an instance has no block and no keywords, closing the safety hole the relaxation opens. web/src/lib/config-edit.ts's addSourceToWebspace is shell-aware (does not seed every configured instance into a shell's first write). Independently re-run: TestValidate_EmptyWebspaceShellIsAccepted, TestValidate_EmptyWebspaceShellIsAcceptedWithZeroSourcesConfigured, TestValidate_WebspaceWithAllowlistButNoMatchInputIsStillRejected, TestValidate_PartiallyCoveredWebspaceIsStillRejected, TestMatchFieldsFor_NoBlockAndNoKeywordsDoesNotParticipate, TestSave_CreateWebspaceThenAddFirstSource_RoundTrips — all PASS. validateFallbackCoverage and Webspace.Participates confirmed byte-unchanged by direct git diff."
    - "G-07-4 (major): with zero webspaces configured, / rendered the kernel-unreachable copy instead of the empty state, because GET /api/config serialized webspaces as null and +page.svelte's onMount read Object.keys(res.config.webspaces) inside the same catch that handled fetch failures. Closed by 07-12: kernel/config/config.go's applyDefaults now normalizes Sources/Webspaces and every webspace's Keywords/Sources/Match to non-nil empty values (confirmed by direct read); web/src/routes/+page.svelte's onMount now wraps only the getConfig() call in its own catch, with all downstream processing (including a defensive ?? {} read) outside it, confirmed by direct read of the current file. Independently re-run: TestConfigHandler_ZeroWebspacesSerializesEmptyObjectsNotNull, TestConfigHandler_WebspaceOmittingCollectionsSerializesEmptyNotNull, TestApplyDefaults_NormalizesCollectionsWithoutChangingValidateVerdicts, and the full go test ./kernel/... suite — all PASS. writer_test.go confirmed byte-unchanged."
    - "G-07-5 (major): adding a second Signal (or any Proton) source failed at the Connect step with a generic go-plugin handshake error, because plugin-fields.ts marked Signal's path/Proton's webmail_base_url required:false (placeholder only) and pluginhost.launch never captured the dying subprocess's stderr. Closed by 07-13: web/src/lib/plugin-fields.ts now marks both fields required:true (confirmed by direct read, with Signal's path additionally seeded with a real default); ConnectionForm.svelte's primary-fields loop carries the DOM required attribute; AddSourceModal.svelte and EditSourceModal.svelte both call missingRequiredFields() before describePlugin/putConfig (confirmed present at both call sites by grep); kernel/pluginhost/host.go now wires a bounded stderrTail into goplugin.ClientConfig.Stderr and appends the plugin's last stderr line to a pre-handshake connect failure (confirmed present by direct read). Independently re-run: TestDescribePluginType_PreHandshakeFatalSurfacesPluginStderr, TestDescribePluginType_SilentPreHandshakeExitLeavesErrorUnchanged, TestStderrTail_* (4 tests) — all PASS."
    - "G-07-6 (major): 'Remove from this webspace' silently no-opped for an all-participate webspace (empty allowlist), and even a correct write would not have hidden the chip because WebspaceHeader read the kernel-wide, unfiltered source list. Closed by 07-14: web/src/lib/config-edit.ts's removeSourceFromWebspace now seeds the current participant set (existing allowlist, or every configured instance) before filtering, confirmed by direct read; web/src/lib/participation.ts gained participatingInstances/participatesIn, the shared client mirror of kernel participation semantics; WebspaceHeader.svelte's chip content (visible/hidden slices, overflow count) now derives from participatingSources, filtered through participatesIn, confirmed by direct read (row visibility deliberately stays unfiltered so the '+' trigger survives a zero-participant webspace). Independently re-run: the full frontend suite (574/574 tests, 34 files) — PASS."
  gaps_remaining: []
  regressions: []
deferred: []
behavior_unverified_items:
  - truth: "07-11's two new backstops: creating a webspace via the UI (+ New webspace) navigates to it with an empty stream and no restart, and adding its first source via the chip-row '+' joins only that instance (not every configured instance)"
    test: "make dev; click the webspace title drop-down, choose + New webspace, type a name, submit. Confirm the modal closes, the app navigates to /w/<name> with no restart, config.toml gains a [webspaces.<name>] block, and the stream is EMPTY. Then click the chip-row +, add one existing instance with match fields, and confirm exactly that one chip appears."
    expected: "Both flows work end to end against a live kernel — this is the code path 07-UAT.md test 3 exercised and found broken; the fix (D-20 empty shell) has unit/integration test coverage but was never re-run live against a real browser after landing"
    why_human: "Requires a live make dev session; not available in this verification environment. The prior 07-UAT.md session tested the PRE-fix code and found it broken (test 3, gap G-07-3) — this is the first live confirmation opportunity since the fix landed."
  - truth: "07-12's backstop: with zero [webspaces.*] blocks, / shows 'No webspaces yet' with a working Create webspace CTA, and a genuinely unreachable kernel still shows the service-unreachable copy"
    test: "make dev with config.toml carrying zero [webspaces.*] blocks; load /; confirm the empty state and CTA. Then stop the kernel process and reload; confirm the service-unreachable copy still renders for a real outage."
    expected: "Empty state renders correctly when the kernel is healthy; unreachable copy renders only for an actual fetch failure"
    why_human: "Requires a live make dev session and a real config.toml edit; not available in this environment. 07-UAT.md test 4 exercised the PRE-fix code and found it broken (gap G-07-4) — this is the first live confirmation since the fix landed."
  - truth: "07-13's two backstops: the two-step 'New Signal…' flow completes end to end with a real path value (Connect step advances to a Conversations match form), and clearing a required field surfaces the missing-field message with zero network requests"
    test: "make dev; + → New Signal…; confirm the path field arrives pre-filled; clear it and click Next, confirm the missing-field message and no network request; restore the value and click Next, confirm the Match step loads and the finished instance appears as a chip."
    expected: "Blank required field blocks submission client-side with no request; a filled field proceeds through Connect → Match → chip appears"
    why_human: "Requires a live make dev session with a real (or fake) Signal/Proton binary; not available in this environment. 07-UAT.md test 5 exercised the PRE-fix code and found the Connect step fatally failing (gap G-07-5) — this is the first live confirmation since the fix landed."
  - truth: "07-14's two backstops: 'Remove from this webspace' makes the chip disappear immediately with no reload, leaves the instance's [sources.<id>] block intact, and leaves other webspaces unchanged; the '+' picker re-offers a removed instance and re-adding it restores its chip and items"
    test: "make dev; open a chip's ⋮ menu, choose Remove from this webspace; confirm the chip disappears immediately, config.toml narrows correctly, other webspaces are untouched. Then reopen the + picker and confirm the removed instance is offered again; re-add it and confirm its chip and items return."
    expected: "Both directions (remove, re-add) round-trip correctly against a live kernel"
    why_human: "Requires a live make dev session; not available in this environment. 07-UAT.md test 6 exercised the PRE-fix code and found the remove silently no-opping (gap G-07-6) — this is the first live confirmation since the fix landed."
  - truth: "UAT tests 11/12 (now unblocked by the G-07-3 fix): the webspace switcher, add-source picker, and manage-sources modal's instance/webspace lists stay usable (height-capped, scrollable) as their counts reach double digits"
    test: "Configure 15+ webspaces and 15+ source instances (now creatable via the UI since G-07-3 is closed); open the switcher, the '+' picker, and Manage sources…; confirm all three scroll internally rather than growing past the viewport"
    expected: "Fixed max-height with internal scroll in all three surfaces"
    why_human: "07-UAT.md tests 11/12 were explicitly blocked_by: other pending the G-07-3 fix and were never run; now unblockable but not yet retested. Tagged verification: backstop in 07-03/07-04/07-05-PLAN.md."
  - truth: "A kernel killed between the config.toml.bak write and the atomic rename leaves config.toml fully intact (07-01 backstop)"
    test: "Kill the topos process (SIGKILL) at the instant between the .bak write and the os.Rename call during a config save, then inspect config.toml"
    expected: "config.toml is byte-identical to its pre-save content — never truncated, never half-written"
    why_human: "07-UAT.md test 10: skipped — 'hard to test, no deterministic way to kill the process at that exact instant.' Genuinely non-deterministic timing window, unchanged since the prior verification round."
  - truth: "A kernel killed midway through the D-07 cleanup leaves at most the interrupted instance's sync_runs rows behind; no other instance is left half-cleaned (07-10 backstop)"
    test: "Kill the topos process (SIGKILL) at the instant between one removed instance's DeleteSourceItems call returning and its DeleteSyncRuns call starting, during an Apply that removes 2+ instances, then inspect the index"
    expected: "At most the interrupted instance's sync_runs rows survive; every other instance is either fully cleaned or fully untouched"
    why_human: "07-UAT.md test 18: skipped — 'non-deterministic kill-timing window, not manually reproducible.' Unchanged since the prior verification round."
  - truth: "handleChipEdit's match-mode describePlugin call resolves without a slower first request's response overwriting a faster second request's state (WR-01, carried code-review advisory)"
    test: "make dev; open 'Edit match settings…' on one chip, then before the vocabulary loads, open 'Edit match settings…' or 'Edit connection…' on a different chip; confirm the modal never briefly shows or reverts to the FIRST chip's vocabulary/open state"
    expected: "The second (current) click's state always wins"
    why_human: "07-UAT.md test 16: skipped — 'can't test manually, the race window needs a browser driver; user chose to assume pass.' Confirmed still present in code (no generation guard on handleChipEdit's describePlugin await); recorded by the user as a non-blocking accepted advisory, not a re-tested pass. Carried as an outstanding advisory for a future /gsd-code-review 7 --fix pass rather than a phase-blocking item."
---

# Phase 7: Webspace Builder UI Verification Report

**Phase Goal:** User can configure sources and webspaces from the UI instead of hand-editing TOML — pick plugin types from a list, configure named instances, save a configured set as a webspace, and promote a live search into the webspace's permanent filter.
**Verified:** 2026-08-09
**Status:** human_needed
**Re-verification:** Yes — after 07-11..07-14 (gap-closure wave closing 07-UAT.md's G-07-3, G-07-4, G-07-5, G-07-6)

## Context for this round

This phase previously reached `human_needed` (prior 07-VERIFICATION.md, 2026-08-08, 73/107 truths verified, no blocking gaps, 19 consolidated human-verification checklist items outstanding). A live `make dev` UAT session was then run against that state and recorded in `07-UAT.md`: 19 tests, **10 passed**, **4 issues** (G-07-3 blocker, G-07-4/G-07-5/G-07-6 major), 3 skipped (non-deterministic timing / driver-limited), 2 blocked (pending the G-07-3 fix). Four gap-closure plans (07-11..07-14, each `gap_closure: true`) were executed to close the four reported issues. This report independently re-verifies that closure against the current codebase — it does not take any SUMMARY.md or 07-UAT.md claim at face value where a build, test, or direct source read could confirm or refute it.

## Build, Test and Contract Evidence (independently re-run in this session, not taken from SUMMARY claims)

| Check | Command | Result |
|---|---|---|
| Go build | `CGO_ENABLED=0 go build ./...` | clean, exit 0 |
| Go vet | `go vet ./kernel/...` | clean, exit 0 |
| Go test suite (full) | `go test ./kernel/... -count=1` | all packages `ok` (config, correlate, httpapi, index, pluginhost, supervisor, syncer) |
| Go test — 07-11 targeted | `go test ./kernel/config/... ./kernel/correlate/... -run 'TestValidate_EmptyWebspaceShellIsAccepted\|TestValidate_EmptyWebspaceShellIsAcceptedWithZeroSourcesConfigured\|TestValidate_WebspaceWithAllowlistButNoMatchInputIsStillRejected\|TestValidate_PartiallyCoveredWebspaceIsStillRejected\|TestMatchFieldsFor_NoBlockAndNoKeywordsDoesNotParticipate\|TestSave_CreateWebspaceThenAddFirstSource_RoundTrips' -count=1 -v` | 6/6 PASS |
| Go test — 07-13 targeted | `go test ./kernel/pluginhost/... -run 'TestDescribePluginType_PreHandshakeFatalSurfacesPluginStderr\|TestDescribePluginType_SilentPreHandshakeExitLeavesErrorUnchanged\|TestStderrTail' -count=1 -v` | 6/6 PASS |
| Frontend test suite (full) | `cd web && npm test -- --run` | 34 test files, **574/574 PASS** |
| Frontend type/lint check | `cd web && npm run check` | 0 errors, 9 pre-existing warnings (unrelated files, unchanged this round) |
| Frontend production build | `cd web && npm run build` | exit 0, `kernel/webui/build` written |
| Diff scope for the whole gap-closure wave | `git diff 87dc71c HEAD --stat` | 35 files: exactly the files named across 07-11..07-14's `files_modified`, plus SUMMARYs/REVIEW/CONTEXT/ROADMAP/STATE/REQUIREMENTS docs — no plugin source, no proto file, no HTTP handler (only handler *tests*) |
| No new dependency | `git diff 87dc71c HEAD --stat -- go.mod go.sum web/package.json web/package-lock.json` | no output |
| `validateFallbackCoverage` untouched | `git diff 87dc71c HEAD -- kernel/config/config.go \| grep validateFallbackCoverage` | only comment-context lines outside the function body; confirmed no diff inside the function itself |
| `Webspace.Participates` untouched | direct read of `kernel/config/types.go:211-221` | byte-identical to pre-round semantics (empty allowlist ⇒ all-participate) |
| `writer_test.go` untouched | `git diff 87dc71c HEAD -- kernel/config/writer_test.go` | no output |
| Debt markers on touched files | `grep -riE 'TBD\|FIXME\|XXX\|TODO\|HACK\|PLACEHOLDER'` over all 17 kernel/web files touched by 07-11..07-14 | none found |
| Requirements traceability | `grep requirements: .07-*-PLAN.md` | KERN-08 and UI-12 are the only two REQ-IDs mapped to Phase 7 in REQUIREMENTS.md; both are claimed by 07-11 through 07-14 (and by most of 07-01..07-10) — no orphaned IDs |
| Fresh code review (this round's diff) | `.planning/phases/07-webspace-builder-ui/07-REVIEW.md`, 2026-08-09 | 0 critical, 4 warning, 1 info — all frontend defensive-robustness gaps, none reachable against the current kernel binary (confirmed: `applyDefaults` runs on every config-load path per `kernel/config/config.go:68,83,218`) |

## Gap Closure — Independently Confirmed

### G-07-3 (blocker) — CLOSED, confirmed by direct source read + passing tests

`kernel/config/types.go` declares `func (w Webspace) IsEmptyShell() bool { return len(w.Keywords) == 0 && len(w.Sources) == 0 && len(w.Match) == 0 }` (lines 244-246). `kernel/config/config.go`'s `validateWebspaces` calls it at line 413 — strictly before the keywords-or-match rejection at line 418 and `validateFallbackCoverage` at line 433, confirmed by direct read of lines 390-436. `kernel/correlate/correlate.go`'s `matchFieldsFor` was independently confirmed (via `TestMatchFieldsFor_NoBlockAndNoKeywordsDoesNotParticipate`, PASS) to return non-participation rather than an all-empty field map for the newly-reachable no-input case — the safety half of the fix. `TestSave_CreateWebspaceThenAddFirstSource_RoundTrips` proves the full create → save → reload → add-first-source → save → reload sequence through the real `config.Store.Save` path, PASS.

### G-07-4 (major) — CLOSED, confirmed by direct source read + passing tests

`kernel/config/config.go`'s `applyDefaults` (lines 157-215) normalizes `Sources`/`Webspaces` and every webspace's `Keywords`/`Sources`/`Match` to non-nil values — confirmed by direct read. `web/src/routes/+page.svelte`'s `onMount` (lines 31-67) confirmed by direct read: the `try/catch` wraps only `await getConfig()`; every downstream step (the `?? {}` defensive read, `resolveRedirectTarget`, navigation, the `phase = 'empty'` assignment) runs outside it. `TestConfigHandler_ZeroWebspacesSerializesEmptyObjectsNotNull` and `TestConfigHandler_WebspaceOmittingCollectionsSerializesEmptyNotNull` both PASS.

### G-07-5 (major) — CLOSED, confirmed by direct source read + passing tests

`web/src/lib/plugin-fields.ts` confirmed by direct read: Signal's `path` (line ~135) and Proton's `webmail_base_url` (line ~117) are both `required: true`, with Signal's `path` carrying a seeded default. `missingRequiredFields(` is called before `describePlugin(`/`putConfig(` in `AddSourceModal.svelte` (lines 219, 267) and before `putConfig(` in `EditSourceModal.svelte` (line 110), confirmed by grep. `kernel/pluginhost/host.go` confirmed by direct read: a bounded `stderrTail` (4 KiB cap, mutex-guarded, lines 223-296) is wired into `goplugin.ClientConfig.Stderr` in `launch` and its last line is appended to a pre-handshake connect failure. `TestDescribePluginType_PreHandshakeFatalSurfacesPluginStderr`, `TestDescribePluginType_SilentPreHandshakeExitLeavesErrorUnchanged`, and all four `TestStderrTail_*` tests PASS.

### G-07-6 (major) — CLOSED, confirmed by direct source read + passing tests

`web/src/lib/config-edit.ts`'s `removeSourceFromWebspace` (lines 203-225) confirmed by direct read: it seeds `currentAllowlist` from the pre-mutation document (falling back to `Object.keys(cfg.sources)` when the existing allowlist is empty) *before* filtering out the removed instance — closing the empty-array no-op the UAT report described. `web/src/lib/components/WebspaceHeader.svelte` confirmed by direct read: `participatingSources` (line 145) filters `sources` through `participatesIn(config, webspace, source.name)` and drives the visible/hidden chip slices and overflow count (lines 267-470), while `showSourceRows` (line 125) deliberately keeps reading the unfiltered `sources` prop so the "+" trigger survives a zero-participant webspace. Full frontend suite (574/574) PASS.

## Observable Truths — Summary by Plan

| Plan | Focus | Truths | Verified (code+test) | Behavior-unverified (human_needed) | Failed |
|---|---|---|---|---|---|
| 07-01 through 07-10 | (carried from prior verification round; unchanged this round — no file in this range was touched by 07-11..07-14, confirmed by `git diff 87dc71c HEAD --stat`) | 107 | 73 | 34 | 0 |
| 07-11 | Gap closure: empty webspace shell (G-07-3) | 12 | 10 | 2 (backstops) | 0 |
| 07-12 | Gap closure: config-null normalization + root-route catch isolation (G-07-4) | 9 | 8 | 1 (backstop) | 0 |
| 07-13 | Gap closure: required-field guard + plugin stderr capture (G-07-5) | 12 | 10 | 2 (backstops) | 0 |
| 07-14 | Gap closure: remove-from-webspace participation fix (G-07-6) | 12 | 10 | 2 (backstops) | 0 |
| **Total** | | **152** | **111** | **41** | **0** |

**Note on the carried 01-10 figures and the intervening UAT session.** The 34 behavior-unverified truths carried from 01-10 are reported unchanged at the truth-level (that ledger is sourced from immutable PLAN frontmatter and this round did not re-derive it). However, a real `make dev` UAT session (`07-UAT.md`) ran against that exact code state between the prior verification and this one, and **independently, behaviorally confirmed 10 of the 19 consolidated human-verification checklist items this phase had outstanding** — covering the webspace switcher, secret field, chip ⋮ menu, manage-sources modal, the two-step flow's collision guard, agent-grant revocation, edit-connection stale-reopen, the D3 `apply_failed`-then-recovers behavior, and the hand-edit Reload-config rejection path (07-UAT.md tests 1, 2, 7, 8, 9, 13, 14, 15, 17, 19 — all `result: pass`). This is genuine behavioral evidence this report did not have at the prior round; it is not reflected in the truth-level ledger above (which stays conservative/unchanged) but **is** reflected in the reduced Human Verification list below, which excludes those 10 now-confirmed scenarios rather than re-listing them as outstanding.

The remaining 9 previously-open checklist items are: 4 that UAT found genuinely broken and this round's gap-closure fixed (now needing a *fresh* live check against the fixed code, since the UAT pass tested the pre-fix code — see Human Verification below), 2 that were blocked pending the G-07-3 fix and are now unblockable but not yet retested, 2 that are genuinely non-deterministic process-kill timing windows (unchanged), and 1 (WR-01) that the user explicitly chose to accept without a fresh pass rather than block on, and which remains a confirmed, non-blocking code-review advisory.

## Required Artifacts

| Artifact | Expected | Status | Details |
|---|---|---|---|
| `kernel/config/types.go` | `Webspace.IsEmptyShell`, three-condition discriminator | ✓ VERIFIED | Confirmed present, lines 244-246; `validateFallbackCoverage`/`Participates` unchanged elsewhere in the file |
| `kernel/config/config.go` | Shell short-circuit in `validateWebspaces`, before all per-webspace checks; `applyDefaults` collection normalization | ✓ VERIFIED | Confirmed by direct read, both changes present and correctly ordered |
| `kernel/correlate/correlate.go` | `matchFieldsFor` non-participation rule for no-block-and-no-keywords | ✓ VERIFIED | Confirmed present; `TestMatchFieldsFor_NoBlockAndNoKeywordsDoesNotParticipate` PASS |
| `kernel/config/store_test.go` | Live round-trip test through `Store.Save` | ✓ VERIFIED | `TestSave_CreateWebspaceThenAddFirstSource_RoundTrips` present and PASS |
| `web/src/lib/participation.ts` | Null-tolerant readers, `isEmptyWebspaceShell`, `participatingInstances`/`participatesIn` | ✓ VERIFIED | Confirmed present; consumed by `config-edit.ts`, `AddSourceModal.svelte`, `WebspaceHeader.svelte` |
| `web/src/lib/config-edit.ts` | Shell-aware `addSourceToWebspace`; seed-then-filter `removeSourceFromWebspace` | ✓ VERIFIED | Both confirmed present by direct read |
| `web/src/routes/+page.svelte` | Isolated fetch catch, defensive `?? {}` read | ✓ VERIFIED | Confirmed present, lines 31-67 |
| `web/src/lib/plugin-fields.ts` | Signal `path` / Proton `webmail_base_url` marked `required: true` | ✓ VERIFIED | Confirmed present |
| `kernel/pluginhost/host.go` | Bounded `stderrTail` wired into `ClientConfig.Stderr` | ✓ VERIFIED | Confirmed present, lines 223-338 |
| `web/src/lib/components/WebspaceHeader.svelte` | Chip content filtered through `participatesIn`; row visibility deliberately unfiltered | ✓ VERIFIED | Confirmed present, lines 125-470 |

## Key Link Verification

| From | To | Via | Status | Details |
|---|---|---|---|---|
| `kernel/config/config.go` `validateWebspaces` | `kernel/config/types.go` `Webspace.IsEmptyShell` | `continue` at line 413/414, before line 418's rejection and line 433's `validateFallbackCoverage` | ✓ WIRED | Confirmed by direct read of line numbers |
| `kernel/correlate/correlate.go` `matchFieldsFor` | plugin `Match` RPC (never called) | Non-participation return when `len(ws.Keywords) == 0` and no explicit block | ✓ WIRED | Confirmed by passing `TestMatchFieldsFor_NoBlockAndNoKeywordsDoesNotParticipate` |
| `web/src/lib/config-edit.ts` `addSourceToWebspace` / `removeSourceFromWebspace` | `web/src/lib/participation.ts` `isEmptyWebspaceShell` / `webspaceSources` | Shell-aware seeding branch; seed-then-filter | ✓ WIRED | Confirmed by direct read |
| `web/src/routes/+page.svelte` `onMount` | `getConfig()` | Catch wraps only the request | ✓ WIRED | Confirmed by direct read |
| `web/src/lib/components/AddSourceModal.svelte` / `EditSourceModal.svelte` | `web/src/lib/plugin-fields.ts` `missingRequiredFields` | Called before `describePlugin`/`putConfig` at every submit site | ✓ WIRED | Confirmed by grep at all 3 call sites |
| `kernel/pluginhost/host.go` `launch` | `stderrTail.lastLine()` | Read strictly after `client.Kill()` returns, appended to the connect-failure error | ✓ WIRED | Confirmed by passing `TestDescribePluginType_PreHandshakeFatalSurfacesPluginStderr` |
| `web/src/lib/components/WebspaceHeader.svelte` | `web/src/lib/participation.ts` `participatesIn` | `participatingSources` derivation feeds every chip-rendering slice; `showSourceRows` deliberately excluded | ✓ WIRED | Confirmed by direct read, lines 125-470 |

## Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|---|---|---|---|---|
| KERN-08 | 07-01, 02, 05-14 | Webspace/source-instance config editable via kernel API (non-secret only), hand-editing remains supported | ✓ SATISFIED | All four UAT-reported defects blocking this requirement (G-07-3 create-webspace rejection, G-07-4 root-route false-negative, G-07-5 blank-required-field launches, G-07-6 remove-silent-no-op) are closed and independently confirmed against the current codebase. The remaining assurance gap is live-browser re-confirmation of the four just-fixed flows plus a handful of carried, non-deterministic backstops — not a code defect. |
| UI-12 | 07-01, 03-14 | Webspace builder UI: pick plugin types, configure named instances, save the set, promote a live search into a permanent filter | ✓ SATISFIED | A real `make dev` UAT session (07-UAT.md) independently confirmed 10 of 19 outstanding UI behaviors live and passing; the four it found broken are now code-fixed with unit/integration test coverage, pending a fresh live pass. No UI-facing must-have is FAILED. |

No orphaned requirement IDs: KERN-08 and UI-12 are the only two REQ-IDs mapped to Phase 7 in REQUIREMENTS.md, and both are claimed across the phase's plans including all four gap-closure plans.

## Anti-Patterns Found

Sourced from the fresh code review (`07-REVIEW.md`, 2026-08-09, scoped to this round's 26-file diff) plus this session's own debt-marker scan.

| File | Line | Pattern | Severity | Impact |
|---|---|---|---|---|
| `web/src/lib/components/ConnectionForm.svelte` | 108-120 | Advanced-fields loop drops the `required` DOM attribute and secret-field branching present in the primary-fields loop; also invisible to its own regression test (`extractBetween` only sees the first `<Input>`) | ⚠️ Warning | Not reachable today (no current field is both `advanced: true` and `required: true`), but a future plugin type with a required advanced field would silently reintroduce the exact class G-07-5/07-13 closed |
| `web/src/lib/config-edit.ts` | 270-285 | `removeSourceInstance` reads `ws.match`/`ws.sources` directly instead of through the null-tolerant `participation.ts` readers used everywhere else in this batch | ⚠️ Warning | Would throw on a hand-edited webspace whose collections arrive as `null` from an older kernel; not reachable against this phase's own kernel binary (`applyDefaults` guarantees non-null) |
| `web/src/lib/participation.ts` (also `web/src/lib/config-edit.ts:159`) | 103 | `participatingInstances` and `addSourceToWebspace`'s seeding branch read `cfg.sources` unguarded, inconsistent with the null-tolerant discipline the same module documents as load-bearing | ⚠️ Warning | Same defence-in-depth gap as above, inside the very module 07-11/07-14 wrote to prevent this class; not reachable against this phase's own kernel binary |
| `kernel/pluginhost/host.go` | 183 | `Host.Reconcile`'s `reflect.DeepEqual` relaunch check includes the `Agent` field, which the subprocess never observes — toggling only an agent grant unnecessarily relaunches the plugin | ⚠️ Warning | Brief reachability/health-probe gap on agent-grant-only changes; pre-existing, not introduced or worsened by this round |
| `web/src/lib/plugin-fields.ts` | 201-212 | `missingRequiredFields` treats an explicit `null` value as present, inconsistent with its own `undefined` handling | ℹ️ Info | Not reachable today (no caller assigns `null` into `SourceConfig`) |

No debt markers (`TBD`/`FIXME`/`XXX`/`TODO`/`HACK`/`PLACEHOLDER`) found in any of the 17 kernel/web files touched by 07-11..07-14 (independently re-scanned this session). 0 Critical findings in the fresh code review — none of the above block the phase goal; all are either unreachable against the current kernel or pre-existing/out-of-scope.

## Human Verification Required

See `behavior_unverified_items` in frontmatter for the full list — **8 consolidated checklist items**, down from the prior round's 19 (10 closed by the intervening `07-UAT.md` session's PASS results, as detailed in the Summary-by-Plan note above). Of the 8:

1. **Four are fresh live-confirmation requests for the just-fixed gaps** (G-07-3 create-webspace + first-source-add, G-07-4 zero-webspace empty state, G-07-5 Signal/Proton required-field guard + Connect flow, G-07-6 remove-from-webspace) — each is code-fixed and covered by passing unit/integration tests in this session, but the live browser confirmation that closed the loop on the *original* UAT report has not yet been re-run against the fixed code.
2. **Two are now-unblocked-but-untested** (UAT tests 11/12, scroll behavior at 15+ webspaces/instances) — previously blocked on G-07-3, now creatable via the UI.
3. **Two are genuinely non-deterministic process-kill timing windows** (config.toml.bak/rename race, D-07 cleanup mid-batch kill) — unchanged since the prior round, not newly introduced.
4. One item (WR-01, handleChipEdit's describePlugin race) is a **confirmed, non-blocking code-review advisory** the user already explicitly chose to accept without a fresh live pass (07-UAT.md test 16) — carried here for visibility, not as a blocking action item.

None of these are FAILED; all are present-and-wired code paths (or, for the two process-kill items, genuinely non-deterministic timing windows) whose live behavior a static/test-suite check cannot fully exercise. This is the expected terminal state under `workflow.human_verify_mode=end-of-phase`.

## Gaps Summary

**No blocking gaps remain.** All four UAT-reported defects (G-07-3 blocker, G-07-4/G-07-5/G-07-6 major) are confirmed closed by direct source reading and independently re-run tests — not taken on any SUMMARY.md's word. The fresh code review found 0 Critical issues; its 4 Warnings are frontend defensive-robustness gaps not reachable against this phase's own kernel binary, and 1 pre-existing Warning (`Host.Reconcile`'s agent-grant relaunch) predates this round. No regression was found in any of the 107 truths carried from plans 01-10 — the gap-closure diff touched exactly the files each plan declared, confirmed by `git diff --stat`.

**Status is `human_needed`, not `passed`,** because 8 human-verification checklist items remain outstanding (down from 19 at the start of this round). This is the expected terminal state for this phase under `workflow.human_verify_mode=end-of-phase`: the remaining items are live-browser confirmations of already-fixed, already-tested code, plus two non-deterministic backstops and one accepted advisory — not code gaps.

---

_Verified: 2026-08-09T13:02:11Z_
_Verifier: Claude (gsd-verifier)_
