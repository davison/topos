---
status: testing
phase: 12-filesystem-source
source: [12-VERIFICATION.md]
started: 2026-08-14T12:20:00Z
updated: 2026-08-14T12:20:00Z
---

## Current Test

number: 1
name: Real desktop — corrected match value surfaces documents
expected: |
  Re-save the actual webspace config so the filesystem source's `folders` match
  value is the source root's own base name (per 12-08's fix), then trigger a
  sync. Your own filesystem documents (previously invisible under the old
  match-all-inexpressible failure) now appear in the correct webspace stream,
  with previews.
awaiting: user response

## Tests

### 1. Real desktop — corrected match value surfaces documents
expected: Re-save the actual webspace config so the filesystem source's `folders` match value is the source root's own base name (per 12-08's fix), then trigger a sync. Your own filesystem documents (previously invisible under the old match-all-inexpressible failure) now appear in the correct webspace stream, with previews. (Re-test of prior round's failed tests 1/3 — G-12-1/G-12-3 fix confirmation.)
result: [pending]

### 2. Real desktop — zero-match state is diagnosed, not silent
expected: Before correcting the config, check `GET /api/sources` (or the Manage Sources UI) for the `files` source. A `last_notice` names the real webspace and the real mistyped match value you originally typed — the zero-match state is diagnosed rather than silent.
result: [pending]

### 3. Real desktop — full pre-fix/post-fix flow, including NFS/SMB
expected: (a) Before the fix, the `files` chip in the relevant webspace shows the warning tone with diagnostic text naming the zero-matching value; (b) after correcting the value, documents appear with previews and a working desktop-handler open action; (c) the same flow works when the source folder is a network (NFS/SMB) mount, not just a local path.
result: [pending]

### 4. Real desktop — unreachable mount with leftover advisory shows honest tooltip (CR-01 counterpart)
expected: Unmount (or otherwise make unreachable) a network-mounted filesystem source whose last completed sync carried a leftover advisory (e.g. a zero-match notice from before the mount went away), then check the chip. It shows the red/destructive dot AND its tooltip reads "{display_name} — unreachable since {relative}" — never the reassuring "synced … — advisory" text.
result: [pending]

### 5. Docs accurately describe the symlink-resolving containment discipline
expected: Read the symlink/containment passages in docs/plugin-contract.md, docs/api.md and docs/plugins/filesystem.md; they describe the symlink-resolving containment discipline the shipped code actually enforces (post-index symlink swaps refused at both the Fetch byte-serving site and the open route; vanished files reported honestly as not-found). (Carried forward — blocked in the prior round by the since-fixed sync bug.)
result: [pending]

### 6. Docs honestly describe the resolved-path discipline and residual TOCTOU window
expected: Read the resolved-path passages in docs/plugins/filesystem.md, docs/api.md and docs/plugin-contract.md; they describe reading/exec-ing the EvalSymlinks-resolved path the containment check approved, and are honest about the residual TOCTOU window ("narrows but does not eliminate"), matching the shipped code. (Carried forward — blocked in the prior round by the since-fixed sync bug.)
result: [pending]

## Summary

total: 6
passed: 0
issues: 0
pending: 6
skipped: 0
blocked: 0

## Gaps

<!-- G-12-1 and G-12-3 from the prior round (see git history of this file): root cause
fixed by plans 12-08..12-11 (root-base-name folder label, zero-match diagnostic,
docs exactness statement, tooltip precedence). Tests 1-4 above are their user-side
confirmation; the gaps re-open here only if those tests fail. -->
