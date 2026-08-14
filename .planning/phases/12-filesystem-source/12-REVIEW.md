---
phase: 12-filesystem-source
reviewed: 2026-08-14T09:58:27Z
depth: standard
files_reviewed: 21
files_reviewed_list:
  - config.example.toml
  - docs/api.md
  - docs/plugins/filesystem.md
  - kernel/correlate/correlate.go
  - kernel/correlate/correlate_test.go
  - kernel/httpapi/sources.go
  - kernel/httpapi/sources_test.go
  - kernel/index/schema.go
  - kernel/index/store.go
  - kernel/index/store_test.go
  - kernel/syncer/coordinator.go
  - kernel/syncer/coordinator_test.go
  - plugins/filesystem/item.go
  - plugins/filesystem/item_test.go
  - web/e2e/specs/12-filesystem-root-label-match.spec.ts
  - web/e2e/specs/12-zero-match-diagnostic.spec.ts
  - web/src/lib/api.ts
  - web/src/lib/components/MatchFieldsForm.svelte
  - web/src/lib/components/SourceChip.svelte
  - web/src/lib/components/match-advisory.test.ts
  - web/src/lib/components/match-values-hint.test.ts
  - web/src/lib/format.ts
findings:
  critical: 1
  warning: 1
  info: 1
  total: 3
status: issues_found
---

# Phase 12: Code Review Report (gap-closure diff, 12-08/12-09/12-10)

**Reviewed:** 2026-08-14T09:58:27Z
**Depth:** standard
**Files Reviewed:** 21 (of the config's 22-file list — `web/src/lib/components/match-advisory.test.ts` and `web/src/lib/components/match-values-hint.test.ts` are new test files reviewed for reliability only, per the "do not report test-file issues unless they affect reliability" rule)
**Status:** issues_found

## Summary

This review supersedes the phase's earlier 12-REVIEW.md, which covered the
pre-gap-closure state of Phase 12. It reviews only what `git diff
8dac3e3..HEAD` actually touched: the three gap-closure plans (12-08, 12-09,
12-10) that (a) make a filesystem source's root folder name match every
file at every depth (`folderLabels`/`dedupeLabels`), (b) add a non-fatal
"zero-match" advisory that travels `correlate.SyncSource` ->
`syncer.Coordinator` -> `sync_runs.notice` -> `GET /api/sources`
(`last_notice`), and (c) surface that advisory on `SourceChip.svelte`'s
health dot (`healthTone`) and tooltip.

The backend plumbing (schema migration, `FinishSyncRunWithNotice`,
`joinNotices`' determinism/bound, `correlate.zeroMatchNotice`'s
error-vs-rejection-vs-notice boundary) is well tested and internally
consistent — I traced every call site and did not find a defect in the
Go code. The one real defect found is in the frontend: `SourceChip.svelte`
implements the "advisory never displaces a bigger problem" precedence
rule inconsistently with its own sibling function, `format.ts`'s
`healthTone` — the tooltip text can show a benign "synced ... — advisory"
message on a source whose health dot is simultaneously rendering red
(destructive/unreachable), directly contradicting the invariant this same
phase's own `healthTone` docstring states.

## Critical Issues

### CR-01: SourceChip tooltip's advisory branch can mask a currently-unreachable source, contradicting the phase's own "advisory never displaces a bigger problem" invariant

**File:** `web/src/lib/components/SourceChip.svelte:200-220`

**Issue:** `format.ts`'s `healthTone` (also touched by this diff,
`web/src/lib/format.ts:118-146`) explicitly documents and tests the rule
that the zero-match advisory must be the *lowest*-priority signal: "a pin
mismatch, a never-synced source, an unreachable source and a failed sync
are each a bigger fact than an advisory about an otherwise-successful
run: a real problem must never be displaced by 'also, here's a heads
up.'" `match-advisory.test.ts` pins this precedence for `healthTone`
directly, including `reachable: false + non-empty last_notice ->
destructive`.

`SourceChip.svelte`'s `tooltipText` derivation implements the *same*
precedence rule for the syncing and pin-mismatch cases only:

```svelte
const base = (() => {
    if (source.syncing) return `${source.display_name} — syncing…`;
    if (isPinMismatch) return `${source.display_name} — binary changed since it was trusted`;
    const relative = formatRelativeTime(source.last_sync_unix);
    if (advisory !== '' && source.last_status !== 'error') {
        return `${source.display_name} — synced ${relative} — ${advisory}`;
    }
    switch (tone) {
        case 'success':
            return `${source.display_name} — synced ${relative}`;
        case 'warning':
            return `${source.display_name} — last error ${relative}: ${source.last_error}`;
        case 'destructive':
            return `${source.display_name} — unreachable since ${relative}`;
        default:
            return `${source.display_name} — not yet synced`;
    }
})();
```

The advisory guard (`advisory !== '' && source.last_status !== 'error'`)
checks neither `source.reachable` nor `tone`. `docs/api.md` explicitly
documents that `reachable` is a **live** probe independent of
`last_status`/history ("a source can flip from reachable to unreachable
between two calls with no sync in between"), so the state `reachable:
false` + `last_status: 'ok'` + a non-empty `last_notice` left over from
that last completed sync is a real, expected, currently-reachable
combination — not a theoretical one. In that state:

- The health **dot** correctly renders `destructive` (red) via
  `healthTone`/`DOT_TONE_CLASS[tone]` (this part is correct and tested).
- The **tooltip** — the only place a user reads *why* the dot is red —
  hits the advisory branch first and renders `"{display_name} — synced
  {relative} — {advisory}"`, never mentioning that the source is
  currently unreachable at all.

This is precisely the class of defect this phase exists to close (a
healthy-looking chip hiding a real problem, G-12-1/G-12-3's own
motivating failure) — except inverted: here a genuinely *unreachable*
source is dressed up as a benign advisory, which will send an operator
troubleshooting the wrong thing (or nothing) while a real connectivity
problem goes unreported in the one place they'd read it.

Neither `match-advisory.test.ts` (which only exercises `healthTone`, not
`tooltipText`'s behavior) nor `12-zero-match-diagnostic.spec.ts` (which
never exercises `reachable: false`) catches this — the gap is real and
untested.

**Fix:** Gate the advisory branch on the same signal `healthTone` already
computed (`tone`), not just `last_status`, so the precedence rule is
expressed once and can't drift between the two files:

```svelte
const relative = formatRelativeTime(source.last_sync_unix);
if (advisory !== '' && tone === 'success') {
    return `${source.display_name} — synced ${relative} — ${advisory}`;
}
```

(`tone` is already `$derived(healthTone(source))` above in this same
file, and `healthTone` already returns `'success'` only when
`last_status !== 'error'` **and** `reachable` **and** the instance didn't
launch-fail — so this one condition replaces the current
`last_status !== 'error'` check and additionally closes the unreachable
gap.)

## Warnings

### WR-01: `sourceStatusesFrom` doesn't populate `LastNotice` for launch-failed (pin-mismatch) merged entries, while `LastStatus`/`LastSyncUnix` do come from the same historical row

**File:** `kernel/httpapi/sources.go:185-219`

**Issue:** For a configured instance that never launched (a
`LaunchFailure` record, e.g. `pin_mismatch`), the merge in
`sourceStatusesFrom` builds:

```go
run := runs[f.Instance]
statuses = append(statuses, sourceStatus{
    ...
    LastStatus:    run.Status,
    LastSyncUnix:  run.FinishedUnix,
    LastError:     f.Message, // deliberately NOT run.Error
})
```

`LastStatus`/`LastSyncUnix` are still read from `run` (the instance's
last *historical* completed sync, from before it started failing to
launch), but the new `LastNotice: run.Notice` line present in the probed
(`healths`) branch above it is absent here. If a since-pin-mismatched
instance's last successful sync (before the swap) carried a zero-match
notice, that fact is silently dropped for this entry even though its
sibling fields (`LastStatus`, `LastSyncUnix`) still surface that same
historical run. In practice this is masked in the UI today because
`isPinMismatch`/`launch_failure === 'pin_mismatch'` takes precedence over
both `healthTone`'s and `SourceChip`'s advisory handling regardless, so
it's not currently user-visible — but it's an inconsistency in what this
merge claims to expose from `run` versus what it actually copies, and
would become a real gap the moment any future UI surface reads
`last_notice` without first checking `launch_failure`.

**Fix:** For consistency with `LastStatus`/`LastSyncUnix` (and unless a
`docs/api.md` update explicitly narrows the contract), add
`LastNotice: run.Notice` to the launch-failure branch's struct literal
alongside the existing `LastStatus`/`LastSyncUnix` lines, or add an
explicit doc-comment note (mirroring the existing `LastError` comment)
stating that `LastNotice` is intentionally never populated for a
launch-failed entry.

## Info

### IN-01: `zeroMatchNotice`'s non-empty-fields guard doesn't catch a field present with an empty value list

**File:** `kernel/correlate/correlate.go:138, 284-312`

**Issue:** `SyncSource` guards notice emission with `explicit && len(fields)
> 0 && len(resp.GetItems()) == 0` — intended (per the doc comment) to stop
"a structurally impossible empty explicit block from producing a
contentless advisory." This guard checks `len(fields)` (the number of
field *keys*), not whether any field actually has non-empty values. An
explicit block declaring a field with an empty value list — e.g.
`[webspaces.x.match.instance] folders = []` — has `len(fields) == 1` and
therefore still fires `zeroMatchNotice`, which then renders as
`"...(folders=) — match values are compared exactly..."` (an empty
right-hand side after the `=`). This is a cosmetically odd but harmless
message, not a functional bug (and it's unclear whether config
validation permits an empty-array match value at all, given
`config.example.toml`'s documented "at least one non-empty,
non-whitespace-only keyword is required" rule for the `keywords`
fallback — the equivalent rule for an explicit `match` block's own value
lists isn't visible in the files reviewed here). Flagged for awareness
only; no test in `correlate_test.go` or `store_test.go` exercises this
specific shape, so its actual reachability against `config.Validate` is
unconfirmed from this diff alone.

**Fix:** Low priority. If reachable, either extend the guard to require
at least one non-empty value across every field (`len(fields) > 0 &&
anyNonEmptyValue(fields)`), or confirm via `kernel/config`'s validator
that an explicit block with an empty value list already fails config
load and is therefore unreachable at this point — in which case no code
change is needed, just a doc-comment note recording that assumption.

---

_Reviewed: 2026-08-14T09:58:27Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
