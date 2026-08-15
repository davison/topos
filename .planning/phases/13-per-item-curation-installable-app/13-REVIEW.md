---
phase: 13-per-item-curation-installable-app
reviewed: 2026-08-15T00:00:00Z
depth: standard
files_reviewed: 3
files_reviewed_list:
  - docs/testing.md
  - web/e2e/specs/13-undo-across-webspace-switch.spec.ts
  - web/src/routes/w/[webspace]/+page.svelte
findings:
  critical: 0
  warning: 1
  info: 0
  total: 1
status: issues_found
---

# Phase 13: Code Review Report (13-07 gap-closure re-review)

**Reviewed:** 2026-08-15T00:00:00Z
**Depth:** standard
**Files Reviewed:** 3
**Status:** issues_found

**Scope note:** This is a re-review of only the delta introduced by gap-closure
plan 13-07 (`git diff 1ec0b43..HEAD`), which fixed 13-REVIEW.md WR-01 (the
exclude/include undo toast's mirror write could retarget itself against the
wrong webspace if the user switched webspaces inside the 5000ms undo window).
**This review supersedes the WR-01 finding of the prior full-phase review**
(committed as `e3ff3d8`, 67 files); it does not re-review those other 66
files. WR-01 itself is now CLOSED — see verification below.

## Summary

`web/src/routes/w/[webspace]/+page.svelte`'s three mark handlers
(`handleExclude`, `handleInclude`, `handleBulkPrimary`) now each snapshot
`webspace`/`navGeneration` into local `ws`/`gen` constants at function entry,
and every subsequent read inside both the immediate write and the `onUndo`
closure uses the snapshot, never the live `$derived(page.params.webspace)`
binding. This is verified correct and complete for the specific WR-01 defect:
grep confirms there are exactly three `onUndo` closures in the file, and the
diff shows all three were updated identically. `setItemMarks(ws, …)` inside
every `onUndo` now targets the webspace the toast was created in, and
`load(gen)` will simply no-op (by the pre-existing `gen !== navGeneration`
guard) if a navigation has superseded it — matching the documented,
intentional "no visible signal after navigation" behavior the new spec
exercises.

While verifying completeness, one adjacent, narrower race was found that
13-07 did not address (see WR-02 below): two side effects in these same
handlers (`closeDetail()` and `handleBulkClear()`) execute unconditionally
immediately after the initial (non-undo) write's `await`, with no
`gen === navGeneration` guard — unlike `load(gen)`, which is correctly
guarded. This is a much narrower window than WR-01 (it requires the initial
`setItemMarks` round-trip itself to race a navigation, not the 5000ms undo
window), so it is filed as a WARNING, not a blocker.

The new spec (`13-undo-across-webspace-switch.spec.ts`) and the
`docs/testing.md` section describing it were both checked against the real
kernel contract (`POST /api/webspaces/{webspace}/marks` request/response
shapes, `GET /api/webspaces/{webspace}/stream`'s default view and
`excluded_count` semantics) and found accurate — no defects found in either.

## Warnings

### WR-02: `closeDetail()`/`handleBulkClear()` still run unconditionally after a navigation, ungated by `navGeneration`

**File:** `web/src/routes/w/[webspace]/+page.svelte:126-127` (`handleExclude`), `149-150` (`handleInclude`), `230-231` (`handleBulkPrimary`)

**Issue:** 13-07 correctly snapshotted `ws`/`gen` so the *write itself*
(`setItemMarks`) and the stream refetch (`load(gen)`, which the pre-existing
`gen !== navGeneration` check inside `load()` already discards when stale)
can never target or display the wrong webspace. However, the UI-state side
effects that run between the initial write's `await` and `load(gen)` —
`closeDetail()` in `handleExclude`/`handleInclude`, and `handleBulkClear()`
in `handleBulkPrimary` — are called unconditionally, with no check against
`gen`/`navGeneration`. If a user triggers an exclude/include/bulk-exclude,
then navigates to a different webspace and opens a different item (or makes
a new bulk selection) in the new webspace before the *original*
`setItemMarks` request resolves, the original handler's `closeDetail()` (or
`handleBulkClear()`) will fire against the now-current state and spuriously
deselect the newly opened item / clear the newly made selection in the
*destination* webspace — a UI-only side effect (no data is written to the
wrong place; `ws` still correctly targets the origin webspace for the actual
mark write), but a real, unintended cross-webspace state clobber.

This window is far narrower than WR-01's (it requires the initial
`POST /marks` round-trip itself — normally single-digit milliseconds against
the local kernel — to still be in flight when the user completes a
navigation and a new selection/open, rather than WR-01's full 5000ms undo
window), so it is unlikely to be hit in the hermetic e2e harness or typical
use. It is nonetheless a real gap in the "the fix is complete" claim: the
snapshot discipline documented in the code comment ("ws/gen … are captured at
entry because markSuccessToast's Undo action can fire up to 5000ms later …")
only reasons about the identity of the mirror write, not about every
side-effecting statement in the same async function.

**Fix:** Gate the two unconditional side effects the same way `load()`
already gates its own state write, e.g.:

```ts
async function handleExclude(id: string) {
	const ws = webspace;
	const gen = navGeneration;
	markBusy = true;
	try {
		await setItemMarks(ws, 'add', [id]);
		if (gen === navGeneration) closeDetail();
		await load(gen);
		...
```

and correspondingly in `handleBulkPrimary`:

```ts
await setItemMarks(ws, action, ids);
if (gen === navGeneration) handleBulkClear();
await load(gen);
```

`handleInclude` needs the identical guard around its own `closeDetail()`
call.

---

_Reviewed: 2026-08-15T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
