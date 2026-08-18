---
status: complete
phase: 12-filesystem-source
source: [12-VERIFICATION.md]
started: 2026-08-14T12:20:00Z
updated: 2026-08-14T13:12:00Z
---

## Current Test

[testing complete]

## Tests

### 1. Real desktop — corrected match value surfaces documents
expected: Re-save the actual webspace config so the filesystem source's `folders` match value is the source root's own base name (per 12-08's fix), then trigger a sync. Your own filesystem documents (previously invisible under the old match-all-inexpressible failure) now appear in the correct webspace stream, with previews. (Re-test of prior round's failed tests 1/3 — G-12-1/G-12-3 fix confirmation.)
result: pass

### 2. Real desktop — zero-match state is diagnosed, not silent
expected: Before correcting the config, check `GET /api/sources` (or the Manage Sources UI) for the `files` source. A `last_notice` names the real webspace and the real mistyped match value you originally typed — the zero-match state is diagnosed rather than silent.
result: pass

### 3. Real desktop — full pre-fix/post-fix flow, including NFS/SMB
expected: (a) Before the fix, the `files` chip in the relevant webspace shows the warning tone with diagnostic text naming the zero-matching value; (b) after correcting the value, documents appear with previews and a working desktop-handler open action; (c) the same flow works when the source folder is a network (NFS/SMB) mount, not just a local path.
result: pass

### 4. Real desktop — unreachable mount with leftover advisory shows honest tooltip (CR-01 counterpart)
expected: Unmount (or otherwise make unreachable) a network-mounted filesystem source whose last completed sync carried a leftover advisory (e.g. a zero-match notice from before the mount went away), then check the chip. It shows the red/destructive dot AND its tooltip reads "{display_name} — unreachable since {relative}" — never the reassuring "synced … — advisory" text.
result: pass
note: "Pass recorded by user decision: could not produce genuine unreachability live (unmounting NFS exposes the underlying local mountpoint, so the root is readable-and-empty — and the observed amber dot + persisted zero-match notice is the CORRECT presentation of that state; initial failure report retracted). The unreachable-with-leftover-advisory rendering is proven in a real browser by 12-tooltip-precedence.spec.ts Test A against a fabricated API response — the coverage split docs/testing.md designs for."

### 5. Docs accurately describe the symlink-resolving containment discipline
expected: Read the symlink/containment passages in docs/plugin-contract.md, docs/api.md and docs/plugins/filesystem.md; they describe the symlink-resolving containment discipline the shipped code actually enforces (post-index symlink swaps refused at both the Fetch byte-serving site and the open route; vanished files reported honestly as not-found). (Carried forward — blocked in the prior round by the since-fixed sync bug.)
result: pass

### 6. Docs honestly describe the resolved-path discipline and residual TOCTOU window
expected: Read the resolved-path passages in docs/plugins/filesystem.md, docs/api.md and docs/plugin-contract.md; they describe reading/exec-ing the EvalSymlinks-resolved path the containment check approved, and are honest about the residual TOCTOU window ("narrows but does not eliminate"), matching the shipped code. (Carried forward — blocked in the prior round by the since-fixed sync bug.)
result: pass

## Summary

total: 6
passed: 6
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

<!-- G-12-1 and G-12-3 from the prior round (see git history of this file): root cause
fixed by plans 12-08..12-11 (root-base-name folder label, zero-match diagnostic,
docs exactness statement, tooltip precedence). Tests 1-3 confirmed the fix on the
user's real desktop this round. Test 4's initial failure report was retracted:
unmounting NFS exposes the empty local mountpoint, so the source was correctly
reported as readable-and-empty, not unreachable — no gap. -->
