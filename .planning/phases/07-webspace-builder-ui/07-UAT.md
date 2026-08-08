---
status: testing
phase: 07-webspace-builder-ui
source: [07-VERIFICATION.md]
started: 2026-08-08T23:05:52Z
updated: 2026-08-08T23:05:52Z
---

## Current Test

number: 1
name: Save-as-filter UI states: with an empty filter stack the filter-chip row does not render at all; Save as filter and each chip's × disable while their write is in flight; a hash-conflict on a filter write surfaces the destructive Alert with the fixed copy 'Config changed on disk — review and retry.'; filter chips render rounded-md (distinguishable from a source chip's rounded-full); and a search query identical to an already-active filter term offers no Save as filter affordance (07-01 Task — four non-backstop truths tracked as behavior-unverified in the Summary by Plan table with no prior checklist entry; added during the 07-VERIFICATION.md internal-consistency repair)
expected: |
  All five states render exactly as UI-SPEC E9/E10 describe, with no visible affordance when the filter stack is empty
awaiting: user response

## Tests

### 1. Save-as-filter UI states: with an empty filter stack the filter-chip row does not render at all; Save as filter and each chip's × disable while their write is in flight; a hash-conflict on a filter write surfaces the destructive Alert with the fixed copy 'Config changed on disk — review and retry.'; filter chips render rounded-md (distinguishable from a source chip's rounded-full); and a search query identical to an already-active filter term offers no Save as filter affordance (07-01 Task — four non-backstop truths tracked as behavior-unverified in the Summary by Plan table with no prior checklist entry; added during the 07-VERIFICATION.md internal-consistency repair)
test: make dev; open a webspace with an active search; click Save as filter and confirm the write-in-flight disabled state, the resulting rounded-md chip, and that a byte-identical repeat search offers no Save as filter button; then force a hash conflict (edit config.toml externally between two saves) and confirm the fixed Alert copy; then remove every filter and confirm the chip row is fully absent, not an empty-styled row
expected: All five states render exactly as UI-SPEC E9/E10 describe, with no visible affordance when the filter stack is empty
result: [pending]

### 2. Webspace switcher: opening the drop-down lists every configured webspace, marks the current one aria-current at weight 600, and clicking a non-current entry navigates to it in one click (07-03 Task 2)
test: make dev; open a webspace with 2+ webspaces configured; open the title drop-down; confirm every webspace is listed, the current one is visually heavier; click another entry
expected: Menu lists all webspaces in GET /api/config order; current one bold; click navigates to /w/{name} with no full page reload artifact
result: [pending]

### 3. Create-webspace modal: submitting a name writes a new [webspaces.<name>] block through PUT /api/config and navigates to it without a kernel restart; a kernel rejection leaves the modal open with the typed name intact (07-03 Task 2)
test: make dev; + New webspace; type a name; submit; confirm config.toml gains the block and the app navigates there with no restart. Then submit a name the kernel rejects and confirm the Alert + retained input.
expected: One PUT /api/config call; success navigates; failure keeps modal open with kernel's verbatim message
result: [pending]

### 4. Root redirect: with no webspaces configured, `/` renders 'No webspaces yet' with a working Create webspace CTA and does not navigate; with webspaces, it lands on the remembered/first one (07-03 Task 3)
test: make dev with config.toml carrying zero [webspaces.*] blocks; load /; confirm the empty state and its CTA. Then add webspaces back and confirm redirect behavior.
expected: No redirect loop, no blank page, CTA opens the same CreateWebspaceModal
result: [pending]

### 5. Add-source '+' picker: opens a popover offering unparticipating instances plus New {plugin type}… rows, or the exact empty-state copy when none remain; choosing an existing instance opens a match-only modal that writes source+match+allowlist and the new chip appears without reload (07-04 Task 1)
test: make dev; click the dashed '+' chip; add an already-configured instance with match fields; confirm the chip appears and its items sync in
expected: Picker lists correctly, one-step modal round-trips through PUT /api/config, chip appears live
result: [pending]

### 6. Two-step 'New {plugin type}…' flow: Connect step trial-launches via describePlugin, advances to a vocabulary-driven Match step on success, offers 'Save anyway' + the exact failure copy on a Describe failure, and Step 2 issues exactly one PUT /api/config (07-04 Task 2)
test: make dev; + → New {plugin type}…; complete Connect against a real/fake service; confirm the vocabulary-driven Step 2 form and a single config.toml write carrying all three blocks. Additionally (07-06 D5): type a display name that collides with an existing instance at Save anyway and confirm the network tab shows no PUT /api/config, the victim instance's chip/connection/agent grants are byte-identical afterwards, and the rejection message + retry affordance render correctly.
expected: Step indicator, describe round trip, single write, chip appears; the collision case refuses the write client-side with no network call
result: [pending]

### 7. Secret field: shows a live Set/Not-set badge for the typed variable name, never displays or transmits a value, and never blocks submit either way (07-04 Task 2)
test: make dev; type an env var name that IS set in the kernel's environment, confirm 'Set'; type one that is NOT, confirm 'Not set — add it to .env and restart before this source can connect.'; confirm the network tab and DOM never contain a secret value
expected: Badge reflects truth; submit stays enabled either way
result: [pending]

### 8. Chip ⋮ menu: offers exactly Edit connection…/Edit match settings…/Remove from this webspace, opening it never toggles the chip's own filter state, and Edit connection… shows the cross-webspace notice before the fields (07-04 Task 3)
test: make dev; click a chip's ⋮ control and confirm the chip's filter state does NOT change; open each menu item and confirm the notice/pre-filled state.
expected: stopPropagation prevents filter toggle; notice visible before fields
result: [pending]

### 9. Manage sources modal: the sole entry point for instance/webspace deletion; deleting an instance shuts down its subprocess and deletes its indexed items across every webspace; deleting a webspace leaves every instance and other webspace untouched; Reload config applies a hand-edit or shows the kernel's verbatim failure with 'The previous configuration is still running.' (07-05 Task 1)
test: make dev; Manage sources…; delete an instance and confirm its chip/items/subprocess are gone everywhere; delete a webspace and confirm nothing else changed; hand-edit config.toml and click Reload config, both for a valid and an invalid edit
expected: Both AlertDialogs behave as documented; Reload's both branches behave as documented
result: [pending]

### 10. A kernel killed between the config.toml.bak write and the atomic rename leaves config.toml fully intact at its previous content (07-01 backstop)
test: Kill the topos process (SIGKILL) at the instant between the .bak write and the os.Rename call during a config save, then inspect config.toml
expected: config.toml is byte-identical to its pre-save content — never truncated, never half-written
result: [pending]

### 11. The webspace switcher drop-down and the add-source picker popover stay usable (height-capped, scrollable) as their list counts reach double digits (07-03/07-04 backstops)
test: Configure 15+ webspaces and 15+ source instances; open the switcher and the '+' picker; confirm both scroll internally rather than growing past the viewport
expected: Fixed max-height with internal scroll
result: [pending]

### 12. The manage-sources instance and webspace lists stay usable as their counts grow (07-05 backstop)
test: Configure 15+ instances and webspaces; open Manage sources…; confirm both lists scroll internally
expected: Fixed max-height with internal scroll
result: [pending]

### 13. Against a live kernel via make dev, the two-step New {plugin type}… flow's Save anyway path refuses a colliding display name in the browser exactly as the unit and structural guards assert, and the victim instance's chip, connection and agent grants are unchanged afterwards (07-06 backstop)
test: make dev; open the two-step New {plugin type}… flow; type a display name colliding with an existing instance; fail the connection test; click Save anyway; confirm no PUT /api/config in the network tab, the collision message renders, and the victim instance's chip/connection/agent grants are byte-identical afterwards
expected: Client-side refusal, no network write, victim instance untouched
result: [pending]

### 14. Against a live kernel via make dev, revoking a source's agent grant in the UI and saving makes that source vanish from an /agent/v1/sources call issued from a second terminal on the next request, with the kernel process never restarted, and the same source's /agent/v1/items/{id} starts answering 404 item_not_found (07-07 backstop)
test: make dev; revoke a source's agent.read grant via the UI's save path; from a second terminal, curl /agent/v1/sources and /agent/v1/items/{id} for an item of that source
expected: Source disappears from /agent/v1/sources, item 404s, no kernel restart
result: [pending]

### 15. Against a live kernel via make dev, opening a source's Edit connection…, typing a wrong base_url, clicking Cancel, then reopening the SAME source's Edit connection… shows the value currently in config.toml — and the same holds for Edit match settings… after a Cancelled match edit (07-08 backstop)
test: make dev; Edit connection… on a source; type a wrong base_url; Cancel; reopen Edit connection… on the same source; confirm the field shows the real stored value, not the discarded typed one. Repeat for Edit match settings…
expected: Reopen always shows current config, never a discarded draft
result: [pending]

### 16. handleChipEdit's match-mode describePlugin call resolves without ever letting a slower first request's response overwrite a faster second request's state (WR-01, 07-REVIEW.md prior round)
test: make dev; open 'Edit match settings…' on one chip, then before the vocabulary loads, open 'Edit match settings…' or 'Edit connection…' on a different chip; confirm the modal never briefly shows or reverts to the FIRST chip's vocabulary/open state
expected: The second (current) click's state always wins; the first click's late-resolving describePlugin response is a no-op
result: [pending]

### 17. Against a live kernel via make dev, editing a source's connection details AND introducing an invalid match field name in the same UI save produces the 500 apply_failed response, and that source's chip then continues to sync and report healthy on its next scheduled tick rather than failing continuously until the kernel is restarted (07-09 backstop, D3)
test: make dev; open a webspace; use the chip ⋮ menu's Edit connection… to change a source's base_url, and in the same session add a match field name the plugin does not declare, then save; confirm the UI surfaces the kernel's rejection; leave the kernel running and watch that source's chip through its next scheduled sync tick
expected: 500 apply_failed with the vocabulary error's own text; the source syncs and reports healthy on its next tick rather than failing every tick
result: [pending]

### 18. A kernel killed midway through the D-07 cleanup — between one instance's items delete and that same instance's sync-history delete — leaves at most that one instance's sync_runs rows behind; no still-configured instance's rows are ever deleted, and no other removed instance in the same batch is left in a half-cleaned state (07-10 backstop)
test: Kill the topos process (SIGKILL) at the instant between one removed instance's DeleteSourceItems call returning and its DeleteSyncRuns call starting, during an Apply that removes 2+ instances, then inspect the index
expected: At most the interrupted instance's sync_runs rows survive; every other instance in the batch is either fully cleaned or fully untouched, never half-cleaned; no still-configured instance's rows are ever touched
result: [pending]

### 19. Against a live kernel via make dev, hand-editing config.toml so that one save both deletes a [sources.<id>] block and typos a [webspaces.<name>.match.<other-instance>] field, then clicking Manage sources… → Reload config, surfaces the kernel's vocabulary rejection AND leaves the removed instance's items absent from every webspace stream and its sync history absent from the health surface — with the kernel never restarted (07-10 backstop)
test: make dev; hand-edit config.toml to remove a [sources.<id>] block and typo an unrelated [webspaces.<name>.match.<other-instance>] field in the same edit; Manage sources… → Reload config; confirm the kernel's rejection message renders, the removed source's items are gone from every webspace stream, its sync history is gone from the health surface, and no restart occurred; then re-add an instance under the same key and confirm no phantom history
expected: Rejection surfaces via the UI; removed source's items/health gone immediately; no restart; re-adding under the same key starts with a clean slate
result: [pending]

## Summary

total: 19
passed: 0
issues: 0
pending: 19
skipped: 0
blocked: 0

## Gaps
