---
status: diagnosed
phase: 07-webspace-builder-ui
source: [07-VERIFICATION.md]
started: 2026-08-08T23:05:52Z
updated: 2026-08-09T10:59:10Z
---

## Current Test

[testing complete]

## Tests

### 1. Save-as-filter UI states: with an empty filter stack the filter-chip row does not render at all; Save as filter and each chip's × disable while their write is in flight; a hash-conflict on a filter write surfaces the destructive Alert with the fixed copy 'Config changed on disk — review and retry.'; filter chips render rounded-md (distinguishable from a source chip's rounded-full); and a search query identical to an already-active filter term offers no Save as filter affordance (07-01 Task — four non-backstop truths tracked as behavior-unverified in the Summary by Plan table with no prior checklist entry; added during the 07-VERIFICATION.md internal-consistency repair)
test: make dev; open a webspace with an active search; click Save as filter and confirm the write-in-flight disabled state, the resulting rounded-md chip, and that a byte-identical repeat search offers no Save as filter button; then force a hash conflict (edit config.toml externally between two saves) and confirm the fixed Alert copy; then remove every filter and confirm the chip row is fully absent, not an empty-styled row
expected: All five states render exactly as UI-SPEC E9/E10 describe, with no visible affordance when the filter stack is empty
result: pass

### 2. Webspace switcher: opening the drop-down lists every configured webspace, marks the current one aria-current at weight 600, and clicking a non-current entry navigates to it in one click (07-03 Task 2)
test: make dev; open a webspace with 2+ webspaces configured; open the title drop-down; confirm every webspace is listed, the current one is visually heavier; click another entry
expected: Menu lists all webspaces in GET /api/config order; current one bold; click navigates to /w/{name} with no full page reload artifact
result: pass

### 3. Create-webspace modal: submitting a name writes a new [webspaces.<name>] block through PUT /api/config and navigates to it without a kernel restart; a kernel rejection leaves the modal open with the typed name intact (07-03 Task 2)
test: make dev; + New webspace; type a name; submit; confirm config.toml gains the block and the app navigates there with no restart. Then submit a name the kernel rejects and confirm the Alert + retained input.
expected: One PUT /api/config call; success navigates; failure keeps modal open with kernel's verbatim message
result: issue
reported: "fails to create the new webspace with error: config: webspace \"uat\" declares neither a keywords fallback nor any match block — declare `keywords = [...]`, a `[webspaces.uat.match.<instance>]` block, or both"
severity: blocker

### 4. Root redirect: with no webspaces configured, `/` renders 'No webspaces yet' with a working Create webspace CTA and does not navigate; with webspaces, it lands on the remembered/first one (07-03 Task 3)
test: make dev with config.toml carrying zero [webspaces.*] blocks; load /; confirm the empty state and its CTA. Then add webspaces back and confirm redirect behavior.
expected: No redirect loop, no blank page, CTA opens the same CreateWebspaceModal
result: issue
reported: "after deleting all webspaces in config.toml and restarting make dev, the root page shows: "Couldn't load this webspace — the topos service didn't respond. Check that it's running, then retry." with no button or link to create one. When webspaces exist, it correctly redirects to the last used"
severity: major

### 5. Add-source '+' picker: opens a popover offering unparticipating instances plus New {plugin type}… rows, or the exact empty-state copy when none remain; choosing an existing instance opens a match-only modal that writes source+match+allowlist and the new chip appears without reload (07-04 Task 1)
test: make dev; click the dashed '+' chip; add an already-configured instance with match fields; confirm the chip appears and its items sync in
expected: Picker lists correctly, one-step modal round-trips through PUT /api/config, chip appears live
result: issue
reported: "attempt to add a second Signal source gives the following error when clicking "Next": Couldn't verify this connection. pluginhost: trial-launch for describe: connect to plugin subprocess: Unrecognized remote plugin message: Failed to read any lines from plugin's stdout (go-plugin handshake failure; plugin path bin/plugins/topos-plugin-signal, arch/permissions all correct)"
severity: major

### 6. Two-step 'New {plugin type}…' flow: Connect step trial-launches via describePlugin, advances to a vocabulary-driven Match step on success, offers 'Save anyway' + the exact failure copy on a Describe failure, and Step 2 issues exactly one PUT /api/config (07-04 Task 2)
test: make dev; + → New {plugin type}…; complete Connect against a real/fake service; confirm the vocabulary-driven Step 2 form and a single config.toml write carrying all three blocks. Additionally (07-06 D5): type a display name that collides with an existing instance at Save anyway and confirm the network tab shows no PUT /api/config, the victim instance's chip/connection/agent grants are byte-identical afterwards, and the rejection message + retry affordance render correctly.
expected: Step indicator, describe round trip, single write, chip appears; the collision case refuses the write client-side with no network call
result: issue
reported: "attempting to remove a plugin from a webspace in order to test this fails. There is no error, it simply displays the syncing spinner on all plugins briefly and then remains unmodified. Otherwise, same result as before (Signal trial-launch handshake failure blocks the Connect step, see test 5)"
severity: major

### 7. Secret field: shows a live Set/Not-set badge for the typed variable name, never displays or transmits a value, and never blocks submit either way (07-04 Task 2)
test: make dev; type an env var name that IS set in the kernel's environment, confirm 'Set'; type one that is NOT, confirm 'Not set — add it to .env and restart before this source can connect.'; confirm the network tab and DOM never contain a secret value
expected: Badge reflects truth; submit stays enabled either way
result: pass

### 8. Chip ⋮ menu: offers exactly Edit connection…/Edit match settings…/Remove from this webspace, opening it never toggles the chip's own filter state, and Edit connection… shows the cross-webspace notice before the fields (07-04 Task 3)
test: make dev; click a chip's ⋮ control and confirm the chip's filter state does NOT change; open each menu item and confirm the notice/pre-filled state.
expected: stopPropagation prevents filter toggle; notice visible before fields
result: pass

### 9. Manage sources modal: the sole entry point for instance/webspace deletion; deleting an instance shuts down its subprocess and deletes its indexed items across every webspace; deleting a webspace leaves every instance and other webspace untouched; Reload config applies a hand-edit or shows the kernel's verbatim failure with 'The previous configuration is still running.' (07-05 Task 1)
test: make dev; Manage sources…; delete an instance and confirm its chip/items/subprocess are gone everywhere; delete a webspace and confirm nothing else changed; hand-edit config.toml and click Reload config, both for a valid and an invalid edit
expected: Both AlertDialogs behave as documented; Reload's both branches behave as documented
result: pass

### 10. A kernel killed between the config.toml.bak write and the atomic rename leaves config.toml fully intact at its previous content (07-01 backstop)
test: Kill the topos process (SIGKILL) at the instant between the .bak write and the os.Rename call during a config save, then inspect config.toml
expected: config.toml is byte-identical to its pre-save content — never truncated, never half-written
result: skipped
reason: "hard to test — no deterministic way to kill the process between the .bak write and the atomic rename"

### 11. The webspace switcher drop-down and the add-source picker popover stay usable (height-capped, scrollable) as their list counts reach double digits (07-03/07-04 backstops)
test: Configure 15+ webspaces and 15+ source instances; open the switcher and the '+' picker; confirm both scroll internally rather than growing past the viewport
expected: Fixed max-height with internal scroll
result: blocked
blocked_by: other
reason: "defer until new webspaces can be added to test (blocked on G-07-3 create-webspace fix)"

### 12. The manage-sources instance and webspace lists stay usable as their counts grow (07-05 backstop)
test: Configure 15+ instances and webspaces; open Manage sources…; confirm both lists scroll internally
expected: Fixed max-height with internal scroll
result: blocked
blocked_by: other
reason: "same as test 11 — defer until new webspaces can be added (blocked on G-07-3 create-webspace fix)"

### 13. Against a live kernel via make dev, the two-step New {plugin type}… flow's Save anyway path refuses a colliding display name in the browser exactly as the unit and structural guards assert, and the victim instance's chip, connection and agent grants are unchanged afterwards (07-06 backstop)
test: make dev; open the two-step New {plugin type}… flow; type a display name colliding with an existing instance; fail the connection test; click Save anyway; confirm no PUT /api/config in the network tab, the collision message renders, and the victim instance's chip/connection/agent grants are byte-identical afterwards
expected: Client-side refusal, no network write, victim instance untouched
result: pass

### 14. Against a live kernel via make dev, revoking a source's agent grant in the UI and saving makes that source vanish from an /agent/v1/sources call issued from a second terminal on the next request, with the kernel process never restarted, and the same source's /agent/v1/items/{id} starts answering 404 item_not_found (07-07 backstop)
test: make dev; revoke a source's agent.read grant via the UI's save path; from a second terminal, curl /agent/v1/sources and /agent/v1/items/{id} for an item of that source
expected: Source disappears from /agent/v1/sources, item 404s, no kernel restart
result: pass

### 15. Against a live kernel via make dev, opening a source's Edit connection…, typing a wrong base_url, clicking Cancel, then reopening the SAME source's Edit connection… shows the value currently in config.toml — and the same holds for Edit match settings… after a Cancelled match edit (07-08 backstop)
test: make dev; Edit connection… on a source; type a wrong base_url; Cancel; reopen Edit connection… on the same source; confirm the field shows the real stored value, not the discarded typed one. Repeat for Edit match settings…
expected: Reopen always shows current config, never a discarded draft
result: pass

### 16. handleChipEdit's match-mode describePlugin call resolves without ever letting a slower first request's response overwrite a faster second request's state (WR-01, 07-REVIEW.md prior round)
test: make dev; open 'Edit match settings…' on one chip, then before the vocabulary loads, open 'Edit match settings…' or 'Edit connection…' on a different chip; confirm the modal never briefly shows or reverts to the FIRST chip's vocabulary/open state
expected: The second (current) click's state always wins; the first click's late-resolving describePlugin response is a no-op
result: skipped
reason: "can't test manually — the race window needs a browser driver, mouse not quick enough; user chose to assume pass. Note: WR-01 remains a confirmed code-level finding (no generation guard on handleChipEdit's describePlugin await), recorded as a non-blocking advisory for /gsd-code-review 7 --fix"

### 17. Against a live kernel via make dev, editing a source's connection details AND introducing an invalid match field name in the same UI save produces the 500 apply_failed response, and that source's chip then continues to sync and report healthy on its next scheduled tick rather than failing continuously until the kernel is restarted (07-09 backstop, D3)
test: make dev; open a webspace; use the chip ⋮ menu's Edit connection… to change a source's base_url, and in the same session add a match field name the plugin does not declare, then save; confirm the UI surfaces the kernel's rejection; leave the kernel running and watch that source's chip through its next scheduled sync tick
expected: 500 apply_failed with the vocabulary error's own text; the source syncs and reports healthy on its next tick rather than failing every tick
result: pass

### 18. A kernel killed midway through the D-07 cleanup — between one instance's items delete and that same instance's sync-history delete — leaves at most that one instance's sync_runs rows behind; no still-configured instance's rows are ever deleted, and no other removed instance in the same batch is left in a half-cleaned state (07-10 backstop)
test: Kill the topos process (SIGKILL) at the instant between one removed instance's DeleteSourceItems call returning and its DeleteSyncRuns call starting, during an Apply that removes 2+ instances, then inspect the index
expected: At most the interrupted instance's sync_runs rows survive; every other instance in the batch is either fully cleaned or fully untouched, never half-cleaned; no still-configured instance's rows are ever touched
result: skipped
reason: "non-deterministic kill-timing window, not manually reproducible; user suggests adding automated browser/driver testing for this class of verification"

### 19. Against a live kernel via make dev, hand-editing config.toml so that one save both deletes a [sources.<id>] block and typos a [webspaces.<name>.match.<other-instance>] field, then clicking Manage sources… → Reload config, surfaces the kernel's vocabulary rejection AND leaves the removed instance's items absent from every webspace stream and its sync history absent from the health surface — with the kernel never restarted (07-10 backstop)
test: make dev; hand-edit config.toml to remove a [sources.<id>] block and typo an unrelated [webspaces.<name>.match.<other-instance>] field in the same edit; Manage sources… → Reload config; confirm the kernel's rejection message renders, the removed source's items are gone from every webspace stream, its sync history is gone from the health surface, and no restart occurred; then re-add an instance under the same key and confirm no phantom history
expected: Rejection surfaces via the UI; removed source's items/health gone immediately; no restart; re-adding under the same key starts with a clean slate
result: pass

## Summary

total: 19
passed: 10
issues: 4
pending: 0
skipped: 3
blocked: 2

## Gaps

- gap_id: G-07-3
  truth: "One PUT /api/config call; success navigates; failure keeps modal open with kernel's verbatim message"
  status: failed
  reason: "User reported (corrected paste): creating a new webspace from the modal always fails — kernel rejects the fresh empty block: config: webspace \"uat\" declares neither a keywords fallback nor any match block. The modal correctly stays open showing the kernel's verbatim message, but a just-created webspace necessarily has no keywords/match yet, so UI creation can never succeed."
  severity: blocker
  test: 3
  root_cause: "Cross-phase contract conflict, not a coding bug in either component: 05-03's unconditional keywords-or-match invariant (kernel/config/config.go validateWebspaces ~323, independently re-derived by validateFallbackCoverage ~416) was never reconciled with 07-03/07-04's deliberate D-14 two-write creation flow (create empty shell first, populate match/allowlist in a later PUT). The shell write can never pass the gate on any install; Webspace.Participates treating an empty sources allowlist as all-participate (types.go ~211) blocks the naive exemption."
  artifacts:
    - path: "kernel/config/config.go"
      issue: "validateWebspaces (~323) + validateFallbackCoverage (~416) blanket-reject a source-less webspace shell; no participation-aware exemption"
    - path: "kernel/config/types.go"
      issue: "Participates (~211): empty sources allowlist means all-participate, so a fresh shell is not distinguishable as 'no participants yet'"
    - path: "web/src/lib/config-edit.ts"
      issue: "addWebspace writes {keywords:[], sources:[], match:{}} per D-14 spec — correct per its own design, but that design collides with the kernel invariant"
    - path: "web/src/lib/components/CreateWebspaceModal.svelte"
      issue: "the flow whose PUT can never validate"
  missing:
    - "A design decision, then one of: (a) participation-aware exemption in validation for a genuinely source-less shell, (b) the modal's single PUT writes a document that already satisfies the invariant, (c) single-write create-and-seed-first-source flow"
    - "A live-kernel round-trip test for webspace creation (07-03's tests only asserted the JS object shape)"
  debug_session: .planning/debug/create-webspace-rejected-empty.md

- gap_id: G-07-4
  truth: "With no webspaces configured, / renders 'No webspaces yet' with a working Create webspace CTA and does not navigate; no redirect loop, no blank page"
  status: failed
  reason: "User reported: with zero webspaces configured, / renders the service-unreachable error copy ('Couldn't load this webspace — the topos service didn't respond…') with no Create webspace CTA — the empty state is never reached. The webspaces-exist redirect to the remembered webspace works correctly."
  severity: major
  test: 4
  root_cause: "Client-side TypeError masquerading as kernel-unreachable: with zero [webspaces.*] blocks the kernel's nil Webspaces map (types.go:19, never defaulted in applyDefaults) marshals as \"webspaces\": null on GET /api/config; +page.svelte's onMount calls Object.keys(res.config.webspaces) unguarded inside the same try/catch that catches fetch failures, so the throw renders the generic service-didn't-respond copy. Kernel answers 200 OK. Confirmed by live ephemeral-kernel repro on port 7799."
  artifacts:
    - path: "web/src/routes/+page.svelte"
      issue: "lines 27-44: unguarded Object.keys(res.config.webspaces) inside a catch-all that conflates fetch failure with response-processing exceptions"
    - path: "kernel/config/types.go"
      issue: "line 19: Webspaces map never defaulted to {} — serializes as null when config has none"
    - path: "kernel/config/config.go"
      issue: "applyDefaults (150-163) does not seed empty Webspaces/Sources maps"
  missing:
    - "Frontend: read res.config.webspaces ?? {} defensively and stop conflating processing errors with fetch failures"
    - "Kernel: default Webspaces (and Sources) to empty maps so /api/config never serializes null, consistent with the existing null→[] normalization convention"
  debug_session: .planning/debug/root-empty-state-service-error.md

- gap_id: G-07-5
  truth: "Two-step New {plugin type}… Connect step trial-launches via describePlugin and advances to the Match step on success"
  status: failed
  reason: "User reported: adding a second Signal source fails at Connect/Next — trial-launch for describe cannot start the signal plugin subprocess: go-plugin handshake failure, 'Failed to read any lines from plugin's stdout' (arch, permissions, ELF all correct)."
  severity: major
  test: 5
  root_cause: "Two confirmed causes (AND): (1) web/src/lib/plugin-fields.ts marks Signal's path (and Proton's webmail_base_url) required:false with the mandatory value shown only as a placeholder; ConnectionForm never enforces required at submit, so the trial-launch receives path:\"\" and plugins/signal/main.go:47 fatals to stderr and exits pre-handshake. (2) pluginhost.launch never sets goplugin.ClientConfig.Stderr, and go-plugin drains plugin stderr only after handshake — so the child's one-line reason ('WEBSPACES_SOURCE_CONFIG: path is empty') is discarded and go-plugin's four-guess generic error surfaces instead. Not Signal-specific: Proton reproduces identically; blank required fields on any plugin hit the same wall. Confirmed by ephemeral-kernel describe calls: no path → 502 byte-identical to UAT report; with path → 200 with vocabulary; with a WRONG path → 200 (emptiness, not wrongness, triggers it)."
  artifacts:
    - path: "web/src/lib/plugin-fields.ts"
      issue: "signal path and proton webmail_base_url marked required:false though both plugins fatal without them; mandatory defaults live only in placeholders"
    - path: "web/src/lib/components/ConnectionForm.svelte"
      issue: "lines 71-82: required only styles the label — no HTML required, no submit-time validation, placeholder never seeds a value"
    - path: "kernel/pluginhost/host.go"
      issue: "launch (~222): ClientConfig.Stderr never set — pre-handshake plugin stderr is lost"
    - path: "plugins/signal/main.go"
      issue: "line 47: pre-Serve fatal on empty path, stderr-only (same pattern plugins/proton/main.go:56)"
    - path: "kernel/httpapi/config.go"
      issue: "DescribePluginHandler (~287): validates binary name only, no required-field check"
  missing:
    - "plugin-fields.ts: mark startup-mandatory fields required:true and seed real default values instead of placeholders"
    - "ConnectionForm/handleConnectNext: enforce required fields at submit so a blank mandatory field never reaches exec.Command"
    - "pluginhost.launch: capture child stderr (bounded buffer) and append its last line to the wrapped trial-launch error — recurrence guard for the whole pre-handshake-fatal class"
  debug_session: .planning/debug/signal-trial-launch-handshake.md

- gap_id: G-07-6
  truth: "Chip menu 'Remove from this webspace' removes the source from the webspace: the write round-trips through PUT /api/config and the chip disappears without reload"
  status: failed
  reason: "User reported: removing a source from a webspace fails silently — all chips briefly show the syncing spinner, then the webspace is unmodified. No error surfaced, no chip removed."
  severity: major
  test: 6
  root_cause: "Two independent, each-sufficient defects: (A) config-edit.ts removeSourceFromWebspace (156-172) filters existing.sources directly — for a webspace with no explicit allowlist (empty = all-participate, exactly the user's 'cars' shape) filtering [] yields [] again, a semantic no-op the kernel accepts with 200 (empty allowlist still means ALL participate per types.go Participates 211-221); the add direction already seeds Object.keys(cfg.sources) when empty but remove never mirrored it. (B) WebspaceHeader.svelte's chip row (visibleSources/hiddenSources, 275-276) derives from the unfiltered kernel-wide GET /api/sources with no participation filter at all — even a correct write wouldn't hide the chip. AddSourceModal's participatingSet already implements the needed client-side Participates mirror but was never shared. PUT genuinely succeeds (error path would render a destructive Alert; none appeared), explaining the brief resync spinner + no change."
  artifacts:
    - path: "web/src/lib/config-edit.ts"
      issue: "removeSourceFromWebspace (156-172): filters a possibly-empty allowlist without seeding the current participant set first"
    - path: "web/src/lib/components/WebspaceHeader.svelte"
      issue: "chip row (275-276) has no webspace-participation filter — driven by kernel-wide /api/sources verbatim"
    - path: "web/src/routes/w/[webspace]/+page.svelte"
      issue: "loadSources (426-444) passes the unfiltered instance list to the header"
  missing:
    - "removeSourceFromWebspace: seed currentParticipants = existing.sources.length > 0 ? existing.sources : Object.keys(cfg.sources) before filtering (mirror of addSourceToWebspace's existing pattern)"
    - "Extract AddSourceModal's participatingSet logic into a shared helper and apply it to WebspaceHeader's chip-row derivation"
    - "A removeSourceFromWebspace test covering the empty-allowlist (all-participate) fixture"
  debug_session: .planning/debug/remove-source-silent-noop.md
