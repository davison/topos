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
  critical: 1
  warning: 2
  info: 1
  total: 4
status: issues_found
---

# Phase 16: Code Review Report

**Reviewed:** 2026-08-20T00:00:00Z
**Depth:** standard
**Files Reviewed:** 31
**Status:** issues_found

## Summary

This is a re-review of Phase 16 (provenance-based plugin trust) after two
gap-closure plans landed. The two previously-found issues both verify as
fixed:

- **CR-01 (prior review)** — `EvaluateTrust`'s tamper-refusal return paths
  now correctly carry `Tier: TierTrusted` (never the zero value) on both
  the link-time arm's tamper branch (`kernel/pluginhost/provenance.go`
  is not where that branch lives — it's in `EvaluateTrust` itself,
  `kernel/pluginhost/provenance.go:648-661` and `:663-671`), with direct
  regression coverage in `manifestgate_test.go`
  (`TestLaunch_ManifestGate_DiscoverRecordsRefusalAndSiblingsStillLaunch`,
  `TestLaunch_ManifestGate_DiscoverSignedArmRefusalReportsTrustedTier`) and
  `escalation_test.go`
  (`TestEscalation_ShadowingCannotInheritTrust`). Holds.
- **CR-02 (prior review)** — `docs/plugin-contract.md`'s "Trust tiers"
  section is now fully evidence-based (`EvaluateTrust`, the two evidence
  arms, the collision rule, no directory-derived language). Holds.

However, tracing the actual collision-resolution code in
`kernel/pluginhost/discover_binaries.go` against the invariant
`docs/plugin-contract.md` itself now documents ("a tamper refusal ...
resolves to the refusal itself and never falls back to launching the
other copy instead") turned up a real, untested divergence between that
documented invariant and the shipped code: see CR-01 below. No test in
`manifestgate_test.go`, `escalation_test.go`, `tier_test.go`, or the
`16-*` e2e specs exercises the specific combination that trips it (one
collision candidate a genuine tamper refusal, the other a clean win) —
every existing collision test uses "neither carries evidence" or "one
carries evidence, the other carries none," never "one is tampered, the
other is clean."

Two further, narrower issues were found in the install/CLI tooling
(`scripts/install.sh`, `cmd/topos-provenance`) — see Warnings below.

## Critical Issues

### CR-01: Collision resolver silently discards a tamper refusal when the colliding candidate resolves cleanly

**File:** `kernel/pluginhost/discover_binaries.go:496-516`

**Issue:** `resolveBinaryDetailed`'s two-directory collision branch evaluates
both the trusted-directory and external-directory candidates, then decides
the winner with two independent `if` checks:

```go
if trustedErr == nil && trustedTrust.Tier == TierTrusted {
    ... return trustedPath, trustedTrust, shadowed, nil
}
if externalErr == nil && externalTrust.Tier == TierTrusted {
    ... return externalPath, externalTrust, shadowed, nil
}
... return trustedPath, trustedTrust, shadowed, trustedErr
```

A tamper refusal (a manifest — link-time or signed — that positively
*names* the candidate binary with a digest that no longer matches what's on
disk) sets `Tier: TierTrusted` **and** a non-nil error on that candidate's
`Trust`/`err` pair (see `EvaluateTrust`, `provenance.go:642-677`). Because
both `if` conditions require `err == nil`, a tampered candidate never
satisfies either branch — but nothing then *refuses* on that candidate's
behalf either. Two concrete scenarios both launch (or fall through to the
existing pin flow) instead of refusing:

1. **Trusted-directory copy is tamper-refused; external-directory copy
   independently carries valid evidence.** `trustedErr != nil` fails the
   first `if`. `externalErr == nil && externalTrust.Tier == TierTrusted`
   passes the second `if` — the function returns the **external** copy
   with a **nil error**. The trusted-directory tamper is never surfaced to
   the caller at all; only a log line ("external copy carries evidence and
   wins") is emitted, with no mention that the trusted copy was refused as
   tampered.
2. **External-directory copy is tamper-refused; trusted-directory copy
   has no evidence at all (plain `TierExternal`, `err == nil`).** Both
   `if`s fail (trusted isn't `TierTrusted`; external's `err != nil`). The
   function falls to the final `return trustedPath, trustedTrust, shadowed,
   trustedErr` — and `trustedErr` is `nil` here, since the trusted side
   itself resolved cleanly (just with no evidence). The external tamper
   refusal is silently dropped; the caller proceeds to the ordinary
   external-tier/pin-check path with no indication a colliding binary was
   ever detected as tampered.

This directly contradicts the invariant `docs/plugin-contract.md`'s own
"Trust tiers" section states for this exact code path (added as part of
this phase): *"A candidate that a manifest positively names with a digest
that no longer matches what's on disk is a tamper refusal — that resolves
to the refusal itself and never falls back to launching the other copy
instead."* The shipped code does fall back to launching (or silently
accepting) the other copy in both directions. This is a genuine
integrity-signal-loss bug: it means an operator (or a compromised update
mechanism) that overwrites a previously-manifest-covered binary's bytes,
where a same-named binary with independent evidence also happens to exist
in the other configured directory, gets **no refusal and no
launch-failure entry** — only a debug-level log line, easy to miss,
that never even names the tamper.

No test in this package, `kernel/supervisor/externaltier_test.go`, or the
`web/e2e/specs/16-*.spec.ts` files exercises this combination — every
existing collision fixture uses "neither side has evidence" or "one side
has evidence, the other has none," never "one side is a genuine tamper
refusal."

**Fix:** Check for a tamper refusal on *either* candidate before checking
for a clean win on either, and refuse immediately if found — matching the
documented invariant:

```go
case trustedPath != "" && externalPath != "":
    shadowed = true
    trustedTrust, trustedErr := EvaluateTrust(dirs, name, trustedPath)
    logDiagnostics(trustedPath, trustedTrust.Diagnostics)
    externalTrust, externalErr := EvaluateTrust(dirs, name, externalPath)
    logDiagnostics(externalPath, externalTrust.Diagnostics)

    // A tamper refusal on EITHER candidate wins outright — it must never
    // be silently overridden by the other candidate's clean resolution
    // (docs/plugin-contract.md's collision rule: "resolves to the
    // refusal itself and never falls back to launching the other copy
    // instead").
    if trustedErr != nil {
        logger.Warn("plugin binary name collides across trusted and external directories: trusted copy is a tamper refusal, refusing regardless of the external copy (D-11)",
            "binary", name, "trusted_path", trustedPath, "external_path", externalPath)
        return trustedPath, trustedTrust, shadowed, trustedErr
    }
    if externalErr != nil {
        logger.Warn("plugin binary name collides across trusted and external directories: external copy is a tamper refusal, refusing regardless of the trusted copy (D-11)",
            "binary", name, "trusted_path", trustedPath, "external_path", externalPath)
        return externalPath, externalTrust, shadowed, externalErr
    }

    if trustedTrust.Tier == TierTrusted {
        logger.Warn("plugin binary name collides across trusted and external directories: trusted copy carries evidence and wins (D-11)",
            "binary", name, "trusted_path", trustedPath, "external_path", externalPath, "evidence", trustedTrust.Evidence)
        return trustedPath, trustedTrust, shadowed, nil
    }
    if externalTrust.Tier == TierTrusted {
        logger.Warn("plugin binary name collides across trusted and external directories: external copy carries evidence and wins (D-11)",
            "binary", name, "trusted_path", trustedPath, "external_path", externalPath, "evidence", externalTrust.Evidence)
        return externalPath, externalTrust, shadowed, nil
    }
    logger.Warn("plugin binary name collides across trusted and external directories: neither copy carries evidence, trusted-first search order decides (D-11)",
        "binary", name, "trusted_path", trustedPath, "external_path", externalPath)
    return trustedPath, trustedTrust, shadowed, nil
```

Add regression tests to `escalation_test.go` (or `tier_test.go`) for both
directions: (a) trusted-side tamper-refused + external-side clean
`TierTrusted` win must refuse, not launch external; (b) external-side
tamper-refused + trusted-side plain `TierExternal` must refuse (or at
minimum surface the tamper), not silently fall through to the ordinary
pin-check path.

## Warnings

### WR-01: install.sh may trust a provenance verifier shipped by the same (potentially compromised) release payload it is verifying

**File:** `scripts/install.sh:259-284`

**Issue:** When a staged release carries `*.provenance.json` files, the
installer resolves a `topos-provenance` verifier in this priority order:
(1) a `topos-provenance` binary present in the **staged payload itself**,
(2) one already installed at `$BIN_DIR/topos-provenance` from a prior
install, (3) one on `PATH`. Provenance verification exists specifically to
catch an attacker who can tamper with release artifacts *and* regenerate
`checksums.txt` to match (i.e., someone who controls the release
publishing pipeline but does not hold the ed25519 private signing key).
Under exactly that threat model, such an attacker can also publish a
`topos-provenance` binary in the same tampered release that unconditionally
reports success — since it is checked first, before the "already
installed" or "on PATH" fallbacks, a target machine with no prior install
and no separately-obtained `topos-provenance` on `PATH` would trust the
attacker's own (fake) verifier to check the attacker's own (tampered)
binaries, defeating the entire point of the provenance check while still
printing a normal success message.

This is an inherent bootstrapping-trust problem with any "verifier shipped
alongside the thing it verifies" design, and may be an accepted tradeoff,
but it is not documented anywhere in `docs/install.md`'s "Provenance
verification" section, and the current resolution order actively prefers
the least-trustworthy source (the payload under test) over the two more
trustworthy ones (a prior install, or an operator-controlled `PATH`
entry).

**Fix:** At minimum, document this bootstrap-trust caveat explicitly in
`docs/install.md`. Consider inverting the resolution order to prefer a
previously-installed or `PATH`-resolved `topos-provenance` over the
staged payload's own copy when one is available, only falling back to the
staged copy when no other verifier exists at all — narrowing (though not
eliminating) the window where a fully-compromised release payload can
supply its own verifier.

### WR-02: `topos-provenance keygen --key-id` and `ParseProvenanceKeys`/`FormatProvenanceKeys` have no charset validation, so a key id containing `,` or `=` silently breaks the round trip

**File:** `cmd/topos-provenance/main.go:79-112`, `kernel/pluginhost/provenance.go:227-272`

**Issue:** `runKeygen` accepts any non-empty `--key-id` string with no
character restriction, and `FormatProvenanceKeys`/`ParseProvenanceKeys`
serialize/parse the accepted-key spec as a bare comma-separated
`keyid=<base64>` list with no escaping. A key id containing a `,` splits
into multiple bogus segments on parse; a key id containing a bare `=`
shifts where `ParseProvenanceKeys` cuts the id from the base64 value. Both
shapes are rejected by `ParseProvenanceKeys` with a "malformed provenance
key segment" error rather than silently accepted with the wrong id/key —
`AcceptedProvenanceKeys` then fails safe (trusts nothing extra) per its
own doc comment — so this is not an authentication bypass. It is, however,
a real footgun for whoever runs `topos-provenance keygen --key-id
<typo-containing-a-comma>` during a release-key rotation: the failure mode
is "provenance verification silently trusts nothing from this key" rather
than a build-time or keygen-time error naming the actual problem.

**Fix:** Validate `--key-id` in `runKeygen` (and ideally in `runSign`,
which also accepts `--key-id`) against a restricted charset (e.g.
`^[A-Za-z0-9._-]+$`), rejecting `,` and `=` outright with a clear error at
the point the id is chosen, rather than surfacing as a downstream parse
failure inside `AcceptedProvenanceKeys`.

## Info

### IN-01: `EvaluateTrust`'s link-time-arm tamper refusal never populates `Trust.Diagnostics`, unlike the signed-arm tamper refusal

**File:** `kernel/pluginhost/provenance.go:648-661` vs `:663-671`

**Issue:** On a link-time-arm tamper refusal, `EvaluateTrust` returns
`Trust{Tier: TierTrusted, Hash: buildHash}` — `Diagnostics` is left nil.
On a signed-arm tamper refusal a few lines later, the equivalent return is
`Trust{Tier: TierTrusted, Hash: provHash, Diagnostics: diagnostics}` — the
scan's collected diagnostics are carried out. Both refusals are
already-informative via the wrapped error (`buildErr`/`provErr`), so this
is not a functional bug, but it's an easy-to-miss asymmetry: a caller that
logs `Trust.Diagnostics` (as `resolveBinaryDetailed`'s `logDiagnostics`
does) gets extra per-candidate detail for a signed-arm tamper refusal but
none for a link-time-arm one, purely because of which arm happened to
name the binary first.

**Fix:** Not required, but for consistency, consider whether the
link-time arm has any equivalent per-candidate diagnostic detail worth
threading through the same field (or explicitly note in the doc comment
that this asymmetry is intentional because the link-time arm has no
multi-candidate scan to report on).

---

_Reviewed: 2026-08-20T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
