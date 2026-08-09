---
status: complete
phase: 07-webspace-builder-ui
source: [07-VERIFICATION.md]
started: 2026-08-09T13:10:01Z
updated: 2026-08-09T15:23:36Z
---

## Current Test

[testing complete]

## Tests

This is round 2 (post-gap-closure). Round 1 (2026-08-08, 19 tests: 10 passed, 4 issues,
3 skipped, 2 blocked) is in git history; its 10 passes are carried forward and excluded here.
Tests 1–4 are live re-confirmations of the four just-fixed gaps; 5 was blocked by G-07-3 and
is now unblocked; 6–7 are non-deterministic kill-window backstops carried from round 1;
8 is a carried code-review advisory the user previously accepted without a live pass.

### 1. Create webspace + add first source, live (G-07-3 fix)
expected: |
  `make dev`; webspace title drop-down → "+ New webspace" → type a name → submit. Modal
  closes, app navigates to /w/<name> with no restart, config.toml gains a [webspaces.<name>]
  block, stream is EMPTY. Then chip-row "+" → add one existing instance with match fields →
  exactly that one chip appears. (Code-fixed by 07-11 with unit/integration coverage; first
  live run since the fix.)
result: issue
reported: "pass with one caveat. When the space is initially created, an error appears in the newly created space; 'Couldn't load this webspace — The topos service didn't respond — check that it's running, then retry.' upon hitting retry it correctly shows 'nothing here yet'"
severity: minor

### 2. Zero-webspace empty state vs real outage (G-07-4 fix)
expected: |
  `make dev` with config.toml carrying zero [webspaces.*] blocks; load "/". The "No webspaces
  yet" empty state renders with a working Create webspace CTA. Then stop the kernel and
  reload: the service-unreachable copy renders only for the genuine outage. (Code-fixed by
  07-12; first live run since the fix.)
result: pass

### 3. Required-field guard + Signal/Proton Connect flow (G-07-5 fix)
expected: |
  `make dev`; "+" → "New Signal…". The path field arrives pre-filled (~/.config/Signal).
  Clear it and click Next: the missing-field message appears with ZERO network requests.
  Restore the value and click Next: the Match step loads and the finished instance appears
  as a chip. (Code-fixed by 07-13; first live run since the fix.)
result: pass

### 4. Remove from webspace + re-add round-trip (G-07-6 fix)
expected: |
  `make dev`; open a chip's ⋮ menu → "Remove from this webspace". The chip disappears
  immediately with no reload, config.toml narrows correctly (instance's [sources.<id>]
  block stays intact), other webspaces unchanged. Reopen the "+" picker: the removed
  instance is offered again; re-adding restores its chip and items. (Code-fixed by 07-14;
  first live run since the fix.)
result: issue
reported: "when the source is removed, its items remain visible in the stream until a refresh occurs. This is counter-intuitive, they should be removed as the chip is removed. Otherwise it's a pass"
severity: minor

### 5. Scroll behavior at 15+ webspaces/instances (round-1 tests 11/12, now unblocked)
expected: |
  Configure 15+ webspaces and 15+ source instances (now creatable via the UI since G-07-3
  is closed); open the webspace switcher, the "+" picker, and "Manage sources…". All three
  lists stay height-capped and scroll internally rather than growing past the viewport.
result: pass

### 6. config.toml.bak rename kill-window backstop (round-1 test 10, carried)
expected: |
  A kernel killed (SIGKILL) between the config.toml.bak write and the atomic os.Rename
  during a save leaves config.toml byte-identical to its pre-save content — never truncated,
  never half-written. (Genuinely non-deterministic timing window; round 1 recorded it as
  skipped — may be skipped again or accepted on code inspection.)
result: skipped
reason: Non-deterministic kill-window; user skipped (as in round 1)

### 7. D-07 cleanup mid-batch kill backstop (round-1 test 18, carried)
expected: |
  A kernel killed (SIGKILL) between one removed instance's DeleteSourceItems returning and
  its DeleteSyncRuns starting, during an Apply removing 2+ instances, leaves at most the
  interrupted instance's sync_runs rows behind; every other instance fully cleaned or fully
  untouched. (Non-deterministic; round 1 recorded it as skipped.)
result: skipped
reason: Non-deterministic kill-window; user skipped (as in round 1)

### 8. handleChipEdit describePlugin race (WR-01 advisory, carried)
expected: |
  `make dev`; open "Edit match settings…" on one chip, then before the vocabulary loads,
  open an edit modal on a different chip: the modal never shows or reverts to the first
  chip's vocabulary/open state. (Round 1 recorded the user accepting this without a live
  pass; no generation guard exists in code — candidate for /gsd-code-review 7 --fix rather
  than a phase blocker.)
result: skipped
reason: Carried advisory previously accepted without a live pass; user skipped again — route to /gsd-code-review 7 --fix, not a phase blocker

## Summary

total: 8
passed: 3
issues: 2
pending: 0
skipped: 3
blocked: 0

## Gaps

All four round-1 gaps are code-fixed by gap-closure plans 07-11..07-14 and independently
re-verified in 07-VERIFICATION.md (2026-08-09) by direct source reads and re-run test
suites. Live re-confirmation of each fix is this round's tests 1–4.

- gap_id: G-07-1
  truth: "Immediately after creating a webspace, the newly created /w/<name> shows the empty stream ('nothing here yet') without an intervening error"
  status: failed
  reason: "User reported: pass with one caveat. When the space is initially created, an error appears in the newly created space; 'Couldn't load this webspace — The topos service didn't respond — check that it's running, then retry.' upon hitting retry it correctly shows 'nothing here yet'"
  severity: minor
  test: 1
  artifacts: []  # Filled by diagnosis
  missing: []    # Filled by diagnosis

- gap_id: G-07-7
  truth: "Removing a source from a webspace also removes that instance's items from the visible stream immediately, without a manual refresh"
  status: failed
  reason: "User reported: when the source is removed, its items remain visible in the stream until a refresh occurs. This is counter-intuitive, they should be removed as the chip is removed. Otherwise it's a pass"
  severity: minor
  test: 4
  artifacts: []  # Filled by diagnosis
  missing: []    # Filled by diagnosis

- gap_id: G-07-3
  truth: "One PUT /api/config call; success navigates; failure keeps modal open with kernel's verbatim message"
  status: resolved
  resolution: "Fixed by 07-11 (D-20 empty webspace shell): Webspace.IsEmptyShell in kernel/config/types.go, validateWebspaces short-circuit before validateFallbackCoverage, correlate.matchFieldsFor no-participation rule, client mirror in web/src/lib/participation.ts, shell-aware addSourceToWebspace seeding. Verified by 07-VERIFICATION.md; live re-test = test 1."
  severity: blocker
  test: 3
  root_cause: "Cross-phase contract conflict, not a coding bug in either component: 05-03's unconditional keywords-or-match invariant (kernel/config/config.go validateWebspaces ~323, independently re-derived by validateFallbackCoverage ~416) was never reconciled with 07-03/07-04's deliberate D-14 two-write creation flow (create empty shell first, populate match/allowlist in a later PUT). The shell write can never pass the gate on any install; Webspace.Participates treating an empty sources allowlist as all-participate (types.go ~211) blocks the naive exemption."
  debug_session: .planning/debug/create-webspace-rejected-empty.md

- gap_id: G-07-4
  truth: "With no webspaces configured, / renders 'No webspaces yet' with a working Create webspace CTA and does not navigate; no redirect loop, no blank page"
  status: resolved
  resolution: "Fixed by 07-12: applyDefaults normalizes Sources/Webspaces and per-webspace collections to non-nil empties so /api/config never serializes null; +page.svelte's onMount fetch isolated in its own catch with redirect/empty-state logic outside it. Verified by 07-VERIFICATION.md; live re-test = test 2."
  severity: major
  test: 4
  root_cause: "Client-side TypeError masquerading as kernel-unreachable: with zero [webspaces.*] blocks the kernel's nil Webspaces map (types.go:19, never defaulted in applyDefaults) marshals as \"webspaces\": null on GET /api/config; +page.svelte's onMount calls Object.keys(res.config.webspaces) unguarded inside the same try/catch that catches fetch failures, so the throw renders the generic service-didn't-respond copy. Kernel answers 200 OK. Confirmed by live ephemeral-kernel repro on port 7799."
  debug_session: .planning/debug/root-empty-state-service-error.md

- gap_id: G-07-5
  truth: "Two-step New {plugin type}… Connect step trial-launches via describePlugin and advances to the Match step on success"
  status: resolved
  resolution: "Fixed by 07-13: plugin-fields.ts derives required flags from each plugin's pre-Serve fatal guards (Signal path, Proton webmail_base_url) and seeds Signal's path default; missingRequiredFields guard blocks every submit path client-side; pluginhost.launch wires a bounded stderrTail into ClientConfig.Stderr so pre-handshake fatals surface their real cause. Verified by 07-VERIFICATION.md; live re-test = test 3."
  severity: major
  test: 5
  root_cause: "Two confirmed causes (AND): (1) web/src/lib/plugin-fields.ts marks Signal's path (and Proton's webmail_base_url) required:false with the mandatory value shown only as a placeholder; ConnectionForm never enforces required at submit, so the trial-launch receives path:\"\" and plugins/signal/main.go:47 fatals to stderr and exits pre-handshake. (2) pluginhost.launch never sets goplugin.ClientConfig.Stderr, and go-plugin drains plugin stderr only after handshake — so the child's one-line reason ('WEBSPACES_SOURCE_CONFIG: path is empty') is discarded and go-plugin's four-guess generic error surfaces instead. Not Signal-specific: Proton reproduces identically; blank required fields on any plugin hit the same wall."
  debug_session: .planning/debug/signal-trial-launch-handshake.md

- gap_id: G-07-6
  truth: "Chip menu 'Remove from this webspace' removes the source from the webspace: the write round-trips through PUT /api/config and the chip disappears without reload"
  status: resolved
  resolution: "Fixed by 07-14: removeSourceFromWebspace seeds the participant set (participatingInstances) before filtering, mirroring addSourceToWebspace; WebspaceHeader's chip row derives from the shared participatesIn predicate; AddSourceModal reuses the shared helper. Verified by 07-VERIFICATION.md; live re-test = test 4."
  severity: major
  test: 6
  root_cause: "Two independent, each-sufficient defects: (A) config-edit.ts removeSourceFromWebspace (156-172) filters existing.sources directly — for a webspace with no explicit allowlist (empty = all-participate) filtering [] yields [] again, a semantic no-op the kernel accepts with 200; the add direction already seeded Object.keys(cfg.sources) when empty but remove never mirrored it. (B) WebspaceHeader.svelte's chip row derived from the unfiltered kernel-wide GET /api/sources with no participation filter — even a correct write wouldn't hide the chip."
  debug_session: .planning/debug/remove-source-silent-noop.md
