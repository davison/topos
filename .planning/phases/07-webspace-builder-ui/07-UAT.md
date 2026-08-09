---
status: testing
phase: 07-webspace-builder-ui
source: [07-VERIFICATION.md]
started: 2026-08-09T17:55:00Z
updated: 2026-08-09T17:55:00Z
---

## Current Test

number: 1
name: G-07-1 fix live confirmation — create webspace lands on empty stream; unconfigured name shows not-configured copy
expected: |
  make dev; webspace title drop-down → + New webspace → type a name → submit. The modal
  closes, the app navigates to /w/<name>, and the stream shows "Nothing here yet"
  immediately — no error state, no Retry click. Then type an unconfigured name into the
  address bar: the not-configured copy renders naming that webspace, does NOT say the
  service didn't respond, and the switcher above still lists real webspaces and navigates.
awaiting: user response

## Tests

### 1. G-07-1 fix live confirmation — create webspace lands on empty stream; unconfigured name shows not-configured copy
expected: make dev; + New webspace → name → submit: modal closes, navigates to /w/<name>, stream shows "Nothing here yet" immediately with no error state and no Retry click (round-2 test 1 re-run against the fix). Then an unconfigured name typed into the address bar renders the not-configured copy naming it — not the outage copy — with the switcher still usable.
result: [pending]

### 2. G-07-7 fix live confirmation — removed source's items leave the stream; quiet refetch never flashes a skeleton
expected: make dev; chip ⋮ menu → Remove from this webspace: chip AND that source's items disappear together with no manual refresh. Re-add the instance via "+": chip returns immediately and its items appear once sync completes, without a manual refresh. A background sync completing on a viewed webspace never flashes the loading skeleton, and a failed background refetch leaves the rendered stream untouched.
result: [pending]

### 3. 07-11 backstops — UI-created webspace is empty and first source joins alone
expected: + New webspace → name → submit: navigates to /w/<name> with no restart, config.toml gains [webspaces.<name>], stream is EMPTY. Chip-row "+" then adds exactly the one chosen instance — not every configured instance.
result: [pending]

### 4. 07-12 backstop — zero-webspace empty state vs genuine outage
expected: With zero [webspaces.*] blocks, / shows "No webspaces yet" with a working Create webspace CTA. Stop the kernel and reload: the service-unreachable copy renders only for the real outage.
result: [pending]

### 5. 07-13 backstops — two-step New Signal… flow and blank-required-field guard
expected: + → New Signal…: path field arrives pre-filled. Clear it and click Next: missing-field message, zero network requests. Restore it and click Next: Match step loads, finished instance appears as a chip.
result: [pending]

### 6. 07-14 backstops — remove chip round-trip
expected: Remove from this webspace: chip disappears immediately with no reload, config.toml narrows correctly, other webspaces untouched. "+" picker re-offers the removed instance; re-adding restores its chip and items.
result: [pending]

### 7. Scroll behavior at 15+ webspaces/instances
expected: With 15+ webspaces and 15+ instances, the switcher, the "+" picker, and Manage sources… all scroll internally (fixed max-height) rather than growing past the viewport.
result: [pending]

### 8. 07-01 backstop — kill between .bak write and atomic rename (non-deterministic)
expected: SIGKILL the kernel between the config.toml.bak write and os.Rename during a save: config.toml is byte-identical to its pre-save content — never truncated or half-written. (Skipped in prior rounds — genuinely non-deterministic timing window.)
result: [pending]

### 9. 07-10 backstop — kill midway through D-07 cleanup (non-deterministic)
expected: SIGKILL between one removed instance's DeleteSourceItems returning and its DeleteSyncRuns starting, during an Apply removing 2+ instances: at most the interrupted instance's sync_runs rows survive; every other instance is fully cleaned or fully untouched. (Skipped in prior rounds.)
result: [pending]

### 10. Carried advisory — chip-edit describePlugin race
expected: Open "Edit match settings…" on one chip, then before its vocabulary loads, open an edit modal on a different chip: the modal never shows or reverts to the first chip's vocabulary/open state — the second click's state always wins. (User previously accepted as a non-blocking advisory; re-offer, may skip.)
result: [pending]

## Summary

total: 10
passed: 0
issues: 0
pending: 10
skipped: 0
blocked: 0

## Gaps
