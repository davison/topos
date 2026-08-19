# Phase 16: Provenance-Based Plugin Trust - Discussion Log

> **Audit trail only.** Do not use as input to planning, research, or execution agents.
> Decisions are captured in CONTEXT.md — this log preserves the alternatives considered.

**Date:** 2026-08-19
**Phase:** 16-Provenance-Based Plugin Trust
**Areas discussed:** Provenance mechanism, Provenance format & travel, Verification timing & manifest coexistence

---

## Todo Cross-Reference (pre-discussion)

| Option | Description | Selected |
|--------|-------------|----------|
| None (Recommended) | All three matched todos stay in backlog — unrelated to provenance trust | ✓ |
| Kernel OAuth/secrets services | Abstract OAuth/secrets into the kernel (2026-08-17) | |
| Signal schema verify tooling | Schema-version verify-and-accept tooling (2026-08-05) | |
| Popover tooltip click bug | Overflow popover clicks intercepted (2026-08-21) | |

**User's choice:** None folded.

---

## Provenance mechanism

### Q1 — Which provenance mechanism grants first-party trust?

| Option | Description | Selected |
|--------|-------------|----------|
| Embedded pubkey + signed manifest (Recommended) | CI signs a checksums manifest with ed25519; kernel embeds the public key; fully offline, ~zero new deps | ✓ |
| Sigstore/cosign keyless | OIDC identity signing, sigstore-go verification; no key custody but heavy deps + external trust root | |
| GitHub artifact attestations | GitHub build provenance via sigstore machinery; tightest GitHub coupling | |
| Signed checksums, external tool | minisign/signify format; standard format, one more tool | |

### Q2 — Key custody and signing location?

| Option | Description | Selected |
|--------|-------------|----------|
| GH Actions secret, CI signs (Recommended) | Repo secret in topos-plugins; tag-triggered workflow signs automatically | ✓ |
| Maintainer-held key, local signing | Key never leaves the machine; manual signing step per release | |
| CI secret + offline backup key | Two embedded pubkeys: CI + offline recovery | |

### Q3 — Key rotation posture?

| Option | Description | Selected |
|--------|-------------|----------|
| Embedded key set (Recommended) | List of accepted keys with IDs (initially one); rotation adds a key in a kernel release | ✓ |
| Single key, hard cutover | One key; rotation breaks older releases | |
| You decide | Claude picks during planning | |

### Q4 — TRUST-02 proving artifact before topos-plugins exists?

| Option | Description | Selected |
|--------|-------------|----------|
| Stand up minimal topos-plugins now (Recommended) | Skeletal sibling repo with signing workflow + trivial plugin; one real signed release verified on an installed kernel | ✓ |
| Signing workflow in kernel repo first | Exercise in kernel CI, lift in Phase 17; weaker same-path claim | |
| Reusable workflow, called from both | Shared GH Actions workflow; same-path by construction | |

---

## Provenance format & travel

### Q1 — Signed unit?

| Option | Description | Selected |
|--------|-------------|----------|
| One signed release manifest (Recommended) | Name→SHA-256 for every plugin, one signature; binds names to hashes | ✓ |
| Per-binary detached signatures | .sig per binary; proves origin, not identity (rename-shadowing possible) | |
| Manifest + per-binary sigs | Both; most flexible, most moving parts | |

### Q2 — Manifest location on the installed machine?

| Option | Description | Selected |
|--------|-------------|----------|
| Alongside binaries in plugins dir (Recommended) | manifest + .sig with the binaries; self-contained; forging needs the private key | ✓ |
| Kernel-owned provenance store | Separate evidence store; can drift from disk | |
| You decide | Claude picks during planning | |

### Q3 — Manifest metadata beyond name→hash?

| Option | Description | Selected |
|--------|-------------|----------|
| Version + contract generation (Recommended) | Per-entry version + gRPC contract generation; release tag + platform at manifest level; feeds DIST-03 | ✓ |
| Hashes only, minimal | Smallest surface; Phase 17 would extend anyway | |
| You decide | Claude settles exact fields in planning | |

### Q4 — Multi-release coexistence?

| Option | Description | Selected |
|--------|-------------|----------|
| Yes — multiple manifests accepted (Recommended) | Versioned manifest files accumulate; trusted if ANY valid manifest matches; independent upgrades | ✓ |
| No — one manifest, atomic set | Single current manifest; whole set re-placed per upgrade | |
| You decide | Claude picks during planning | |

---

## Verification timing & manifest coexistence

### Q1 — When does verification run?

| Option | Description | Selected |
|--------|-------------|----------|
| Install AND every launch (Recommended) | Verify before place (Phase 15 discipline) + re-verify per launch like today's link-time gate | ✓ |
| Install only, pin at launch | One-time signature check, hash pin thereafter | |
| Launch only | Kernel sole verifier; artifact placed before rejection | |

### Q2 — Coexistence with the link-time manifest until Phase 17?

| Option | Description | Selected |
|--------|-------------|----------|
| Either grants trusted (Recommended) | Link-time OR signed manifest ⇒ TierTrusted; Phase 17 deletes the link-time arm | ✓ |
| Signed primary, link-time fallback | Ordering adds spec for no security gain | |
| You decide | Claude picks during planning | |

### Q3 — What do the trusted/external directories become?

| Option | Description | Selected |
|--------|-------------|----------|
| Search paths only — tier per binary (Recommended) | Tier from provenance alone; config edit/file drop/shadowing all earn nothing | ✓ |
| Keep dir split as placement convention | Residual dir semantics; TRUST-04 must prove them toothless | |
| You decide | Claude picks during planning | |

### Q4 — Post-split dev loop anticipation?

| Option | Description | Selected |
|--------|-------------|----------|
| Note it, decide in Phase 17 (Recommended) | Record the constraint (keep manifest-injection seam clean); mechanism is REPO-05's problem | ✓ |
| Design the dev mechanism now | Designs against a repo layout that doesn't exist yet | |
| Dev builds use consent-and-pin | Re-triggers pin flow every rebuild — the false-alarm loop the manifest avoids | |

---

## Claude's Discretion

- Failure behavior & operator visibility (unselected fourth gray area) — within locked conventions: fail loudly by name, never demote-and-run, no silent downgrade, e2e extension on UI-visible change.
- Manifest file format, filename/versioning scheme, stale-manifest cleanup.
- Code structure of the two-arm coexistence (two verifiers behind one interface vs one verifier, two evidence sources).
- TRUST-04 test placement/shape, one committed test per escalation path.

## Deferred Ideas

- Post-split dev-loop trust mechanism (unsigned local builds → dev kernel) — Phase 17 (REPO-05).
- Retiring the link-time manifest arm — Phase 17.
- Reviewed-but-not-folded todos: kernel OAuth/secrets services, Signal schema tooling, popover tooltip bug — all stay in backlog.
