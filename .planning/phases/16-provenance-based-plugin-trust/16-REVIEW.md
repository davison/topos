---
phase: 16-provenance-based-plugin-trust
reviewed: 2026-08-20T00:00:00Z
depth: standard
files_reviewed: 31
files_reviewed_list:
  - cmd/topos-provenance/main.go
  - docs/api.md
  - docs/install.md
  - docs/plugin-contract.md
  - docs/plugin-trust.md
  - docs/testing.md
  - .gitignore
  - kernel/config/types.go
  - kernel/httpapi/config_test.go
  - kernel/httpapi/sources_test.go
  - kernel/pluginhost/binaryhash.go
  - kernel/pluginhost/discover_binaries.go
  - kernel/pluginhost/discover_binaries_test.go
  - kernel/pluginhost/escalation_test.go
  - kernel/pluginhost/host.go
  - kernel/pluginhost/manifestgate_test.go
  - kernel/pluginhost/provenance.go
  - kernel/pluginhost/provenance_test.go
  - kernel/pluginhost/tier_test.go
  - kernel/supervisor/externaltier_test.go
  - Makefile
  - scripts/install.sh
  - scripts/install-smoke.sh
  - scripts/provenance-smoke.sh
  - scripts/smoke-lib.sh
  - web/e2e/fixtures/config-builder.ts
  - web/e2e/fixtures/kernel.ts
  - web/e2e/fixtures/plugin-binaries.ts
  - web/e2e/specs/11-external-tier-badge.spec.ts
  - web/e2e/specs/12-external-rehearsal.spec.ts
  - web/e2e/specs/13-manifest-unverified.spec.ts
  - web/e2e/specs/16-file-drop-external-tier.spec.ts
  - web/e2e/specs/16-signed-provenance-tier.spec.ts
findings:
  critical: 2
  warning: 2
  info: 1
  total: 5
status: issues_found
---

# Phase 16: Code Review Report

**Reviewed:** 2026-08-20T00:00:00Z
**Depth:** standard
**Files Reviewed:** 31
**Status:** issues_found

## Summary

This phase replaces directory-derived plugin trust with provenance-derived
trust (`EvaluateTrust`, `kernel/pluginhost/provenance.go`) and rewires the
launch gate (`discover_binaries.go`, `host.go`) accordingly. The
cryptographic core — ed25519 signature verification over raw manifest
bytes, key-id lookup against an explicit accepted-key set, platform/schema
checks, exhaustive multi-manifest scanning, and the "never demote-and-run"
tamper-refusal discipline — is implemented correctly and is backed by an
unusually thorough unit/escalation/e2e test suite (`provenance_test.go`,
`escalation_test.go`, `tier_test.go`, `manifestgate_test.go`, the two
`16-*.spec.ts` files). The install-time verification step in
`scripts/install.sh` correctly resolves and invokes the verifier before
any placement, and the shell quoting/allowlisting around `checksums.txt`
parsing is careful about traversal and injection.

Two real defects were found. The first is a genuine regression: the new
`EvaluateTrust` function forgets to populate `Trust.Tier` on its two
tamper-refusal return paths, so a `manifest_unverified` `LaunchFailure`'s
`tier` field silently reports `""` instead of the documented `"trusted"` —
confirmed by comparing against the pre-Phase-16 code, where this field was
always correctly `"trusted"` in the equivalent branch. The second is a
documentation defect: `docs/plugin-contract.md`'s "Trust tiers" section
still describes the *old*, pre-Phase-16 directory-derived trust model
almost verbatim, directly contradicting both the code this phase shipped
(`D-11`, location-independent trust) and this phase's own canonical
`docs/plugin-trust.md`.

## Critical Issues

### CR-01: `EvaluateTrust`'s tamper-refusal paths never set `Trust.Tier`, so `GET /api/sources` reports `tier: ""` instead of the documented `"trusted"` for a `manifest_unverified` entry

**File:** `kernel/pluginhost/provenance.go:635-647`
**Issue:**

```go
if expected, ok := TrustManifest()[name]; ok && expected != buildHash {
    // ...
    return Trust{Hash: buildHash}, buildErr        // <-- Tier left at zero value ""
}

provHash, evidence, diagnostics, provErr := VerifySignedProvenance(dirs, name, path)
if provErr != nil {
    return Trust{Hash: provHash, Diagnostics: diagnostics}, provErr   // <-- Tier left at zero value ""
}
```

Both tamper-refusal branches of `EvaluateTrust` (link-time-arm digest
mismatch, and signed-arm digest mismatch via `VerifySignedProvenance`)
construct a `Trust{}` value without setting `Tier`. `Tier` is a `string`
type, so its zero value is `""` — neither `TierTrusted` nor `TierExternal`.

This `Trust` value flows straight through `resolveBinaryDetailed` (which
returns `EvaluateTrust`'s result unmodified in the non-collision case,
`discover_binaries.go:516-523`) into `launch()`'s `tier := trust.Tier`
(`host.go:949`), and from there into `manifestUnverifiedError{tier: tier,
...}` (`host.go:978-985`) and `toLaunchFailure()`'s `Tier: e.tier`
(`host.go:205-214`). The result: `kernel/httpapi/sources.go`'s
`Tier: string(f.Tier)` serializes `"tier": ""` on the wire for **every**
`manifest_unverified` entry — a tampered/name-hijacked binary.

This contradicts `docs/api.md`'s own worked example for this exact case
(the `dropped-binary` entry, `docs/api.md:657`, which shows `"tier":
"trusted"`) and the field's own documented contract (`docs/api.md:721-724`:
"`tier` is `"trusted"` or `"external"`"). It is also a **regression**: the
pre-Phase-16 code (`git show d6b619d7:kernel/pluginhost/host.go`) computed
`tier` from `resolveBinaryDetailed` directly (directory-derived), so the
manifest-gate failure branch — reachable only when `tier == TierTrusted`
— always carried `tier: TierTrusted` into the same `manifestUnverifiedError`
literal. This phase's refactor to provenance-derived `Trust` silently lost
that guarantee.

On the frontend, `web/src/lib/components/SourceChip.svelte` derives
`isExternal = source.tier === 'external'` and passes `tier={source.tier}`
straight into `TrustBadge.svelte`, which renders its "untrusted" glyph
only `{#if tier === 'external'}`. With `tier === ""`, neither branch is
taken — the badge silently behaves as if the source were trusted-tier,
even though the chip's separate health-dot/tooltip/menu channels (driven
off `launch_failure`, not `tier`) still correctly show the failure. No
existing test — not `kernel/httpapi/sources_test.go`'s
`TestSourcesHandler_ManifestUnverifiedEntryCarriesNoLaunchAdvisory` (which
hand-constructs the `fakeProber` fixture with `Tier:
pluginhost.TierTrusted`, sidestepping the real code path) nor
`web/e2e/specs/13-manifest-unverified.spec.ts` (which never reads the wire
`tier` field at all) — exercises the real value produced by
`Discover()`/`launch()` here, which is why this shipped unnoticed.

**Fix:**
```go
if expected, ok := TrustManifest()[name]; ok && expected != buildHash {
    return Trust{Tier: TierTrusted, Hash: buildHash}, buildErr
}

provHash, evidence, diagnostics, provErr := VerifySignedProvenance(dirs, name, path)
if provErr != nil {
    return Trust{Tier: TierTrusted, Hash: provHash, Diagnostics: diagnostics}, provErr
}
```
Reaching either branch already means a trust arm positively named this
binary (with a digest that no longer matches) — exactly the condition
`docs/api.md` documents as `tier: "trusted"`. Add a regression test that
exercises the *real* `Discover()`/`launch()` path (not a hand-built
`LaunchFailure` fixture) and asserts `LaunchFailures()[0].Tier ==
TierTrusted` for a genuine digest-mismatch refusal, mirroring
`TestLaunch_ManifestGate_DiscoverRecordsRefusalAndSiblingsStillLaunch` but
asserting on `.Tier` as well as `.Reason`.

## Critical Issues (continued)

### CR-02: `docs/plugin-contract.md`'s "Trust tiers" section still documents the pre-Phase-16 directory-derived trust model, contradicting the shipped code and `docs/plugin-trust.md`

**File:** `docs/plugin-contract.md:206-241`
**Issue:**

`docs/plugin-trust.md` (this phase's own canonical trust document, added
by this same diff) opens with: "Trust derives from verifiable provenance
carried by the artifact itself, **never from where the binary sits on
disk**" and states every other trust-touching document "links back here
rather than restating the model."

`docs/plugin-contract.md`'s "Trust tiers" section does the opposite — it
restates the *old* model, and gets it wrong for this phase:

```
- **Trusted** — `[plugins] dir`, default `plugins`, ... A binary resolved
  from here is `pluginhost.TierTrusted`.
- **External** — `[plugins] external_dir`. A binary resolved from here is
  `pluginhost.TierExternal`.
...
**Tier is derived exclusively from which directory a binary resolved
from, launch time — never from anything the plugin itself declares.**
...
**The shadow rule.** ... the trusted directory always wins, silently to
the running kernel (loudly to its own logs), never the other way around.
```

Every one of these three claims is now false under D-11, as implemented
and tested by this very phase:

- A binary in the **external** directory *can* resolve `TierTrusted` —
  proven by `kernel/pluginhost/tier_test.go`'s
  `TestResolveBinary_ExternalWithSignedManifestResolvesTrustedTier` and
  `TestResolveBinary_LocationSymmetric`, and by
  `web/e2e/specs/16-signed-provenance-tier.spec.ts`'s whole premise
  ("success criterion 1: trust is no longer a property of location").
- Tier is **not** "derived exclusively from which directory a binary
  resolved from" — it is derived from `EvaluateTrust`'s provenance
  evidence; `discover_binaries.go:186-207`'s own `Dirs` doc comment says
  so explicitly ("Neither field grants anything by itself... tier is
  decided per binary by `pluginhost.EvaluateTrust`... wherever the binary
  sits").
- The shadow rule does **not** always resolve to "the trusted directory
  always wins" — `kernel/pluginhost/tier_test.go`'s
  `TestResolveBinary_CollisionResolvesToWhicheverCopyCarriesEvidence`
  proves the **external** copy wins a collision when only it carries
  provenance evidence, and `escalation_test.go`'s
  `TestEscalation_ShadowingCannotInheritTrust` "cross-directory shadow"
  subtest proves the same at the `ResolveBinary` boundary.

An operator reading only `docs/plugin-contract.md` (which is still the
document `docs/plugin-trust.md` calls its own audience-facing counterpart
for "the plugin contract and directory layout this trust model sits on
top of") would form an actively incorrect mental model of which binary
runs on a name collision and whether the external directory can ever be
trusted — exactly the kind of misunderstanding that leads to a
misconfigured, over-trusting deployment.

**Fix:** Rewrite `docs/plugin-contract.md:206-241` to state the two
directories are pure search paths (per `docs/plugin-trust.md` and
`Dirs`'s own doc comment), remove the per-directory
`TierTrusted`/`TierExternal` assertions, and correct the shadow-rule
paragraph to "whichever candidate carries valid evidence wins; if neither
or both do, the trusted-first search order decides" (matching
`resolveBinaryDetailed`'s actual precedence and its own tests). Point the
reader at `docs/plugin-trust.md` for the authoritative model rather than
restating a stale one, consistent with that document's stated intent.

## Warnings

### WR-01: No test asserts the real `tier` value produced by a genuine `manifest_unverified` `LaunchFailure` — the gap that let CR-01 ship

**File:** `kernel/pluginhost/manifestgate_test.go:246-297`, `kernel/httpapi/sources_test.go:943-989`
**Issue:** `TestLaunch_ManifestGate_DiscoverRecordsRefusalAndSiblingsStillLaunch`
(the real-`Discover()` test closest to this failure mode) asserts
`failures[0].Instance` and `failures[0].Reason` but never
`failures[0].Tier`. `TestSourcesHandler_ManifestUnverifiedEntryCarriesNoLaunchAdvisory`
constructs the `fakeProber`'s `LaunchFailure` fixture by hand with
`Tier: pluginhost.TierTrusted`, so it *asserts the documented contract*
without ever exercising the code that is supposed to produce that value.
`web/e2e/specs/13-manifest-unverified.spec.ts`'s typed response shape
(line 115) doesn't even include a `tier` field. As a result nothing in
this phase's otherwise-thorough test suite could have caught CR-01.
**Fix:** Add an assertion on `.Tier` to
`TestLaunch_ManifestGate_DiscoverRecordsRefusalAndSiblingsStillLaunch`
(expect `TierTrusted`), and widen `13-manifest-unverified.spec.ts`'s
response type/assertions to include `tier` and assert `'trusted'`,
mirroring how `16-signed-provenance-tier.spec.ts` and
`16-file-drop-external-tier.spec.ts` both already assert `tier` on their
own entries.

### WR-02: `scripts/install.sh` marks provenance manifest/signature data files executable

**File:** `scripts/install.sh:293-307`
**Issue:** The placement loop applies `chmod 0755 "$tmp"` unconditionally
to every asset named in `checksums.txt`, including `*.provenance.json`/
`*.provenance.sig` files that land under `$PLUGINS_DIR` when a release
publishes them (the mode is applied before the `case` that only
*chooses the destination path*, not before deciding whether the asset is
a binary). These are plain data files the kernel only ever `os.Stat`s and
`os.ReadFile`s (`provenance.go`'s `readAndVerifyProvenanceManifest`); the
execute bit is inert but incorrect, and inconsistent with the 0644 mode
`cmd/topos-provenance`'s own `runSign` uses when writing these same files
directly (`main.go:190-195`).
**Fix:** Branch the mode by asset shape in the placement loop — `0755`
for `topos`/`topos-provenance`/`plugins/<binary>`, `0644` for
`plugins/*.provenance.json`/`plugins/*.provenance.sig`.

## Info

### IN-01: Collision-fallback log line can misname a tamper refusal as "neither copy carries evidence"

**File:** `kernel/pluginhost/discover_binaries.go:495-515`
**Issue:** In `resolveBinaryDetailed`'s collision branch, when the trusted
candidate is a genuine tamper refusal (`trustedErr != nil`, i.e. a
manifest *does* name it, with a mismatched digest) and the external
candidate is a plain no-evidence success, neither `if` branch matches and
execution falls to:
```go
logger.Warn("plugin binary name collides across trusted and external directories: neither copy carries evidence, trusted-first search order decides (D-11)", ...)
return trustedPath, trustedTrust, shadowed, trustedErr
```
The emitted message says "neither copy carries evidence," but the trusted
copy in this scenario carries *positive, mismatching* evidence (a tamper
refusal), which is a materially different — and more actionable — fact
for an operator reading the log than "no evidence anywhere." The
resulting *behavior* (refuse, never fall back to the external copy) is
correct per D-13's "never demote-and-run" rule and is exercised by
`escalation_test.go`'s "digest mismatch under a legitimately-named
manifest entry" case; only the log wording for this specific sub-case is
imprecise.
**Fix:** Distinguish the "trusted side refused as tampered" sub-case in
the warning text (e.g. "trusted copy is a tamper refusal — collision
resolves to that refusal, never falling back to the external copy") so
an operator's first diagnostic signal names the real cause.

---

_Reviewed: 2026-08-20T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
