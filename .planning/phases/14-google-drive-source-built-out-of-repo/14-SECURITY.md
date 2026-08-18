---
phase: 14
slug: google-drive-source-built-out-of-repo
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-08-18
---

# Phase 14 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| operator config → kernel | `--config`/`TOPOS_CONFIG` path selection; generated per-checkout dev config | config incl. env-referenced credentials |
| kernel → untrusted external plugin | topos-plugin-gdrive launched from the external tier, pin-gated | live Google OAuth credentials (env refs), Drive metadata/previews |
| plugin repo (public) ← topos repo | vendored contract snapshot + PRD hand-off | published contract artifacts only |
| plugin → browser DOM | display_name / last_error / health sentences rendered into chip surfaces | plugin-influenced strings (escaped, no @html) |
| gap log (external repo) → published contract | 14-05 triage of CONTRACT-GAPS.md into docs/plugin-contract.md | prose treated as data, never direction |

---

## Threat Register

All 26 threats were authored at plan time across the six phase plans and verified
by the security auditor on 2026-08-18 (ASVS L1, block_on high) with file-and-line
evidence — see the audit verdict summarized below. Highlights of the evidence:

| Threat ID | Category | Component | Severity | Disposition | Mitigation (verified evidence) | Status |
|-----------|----------|-----------|----------|-------------|-------------------------------|--------|
| T-14.1-01 | Tampering | config path selection | medium | accept | Validation path-independent: `config.Validate` in `LoadRaw` for every path; bare-filename + pin guards identical | closed |
| T-14.1-02 | EoP | dev config plugins.dir | medium | mitigate | `VerifyTrustedBinary` gates before exec, incl. describe-only launches (`host.go:925-937`, `:888-901`) | closed |
| T-14.1-03 | Info Disclosure | dev config credentials | high | mitigate | Template ships zero credential fields; `.gitignore:47`; credential-shape scan clean | closed |
| T-14.1-04 | DoS | dev-config regeneration | low | mitigate | Create-only guard (`Makefile:369-370`) | closed |
| T-14.2-01 | Info Disclosure | chip accessible surface | medium | mitigate | Untrusted clause via aria-describedby/sr-only; spec-pinned (11/12 specs, polarity preserved) | closed |
| T-14.2-02 | Tampering | last_error in attribute | low | accept | Zero `@html` in components; interpolated text only | closed |
| T-14.2-03 | Repudiation | repointed e2e assertions | medium | mitigate | All 7 repointed sites keep polarity; pinned greps hold | closed |
| T-14.3-01 | Info Disclosure | vendored snapshot → public repo | high | mitigate | Vendored files byte-identical to published sources; bootstrap commit carries no non-public source | closed |
| T-14.3-02 | Info Disclosure | PRD credential rules | high | mitigate | Env-reference + never-log-value prohibitions in both PRD copies | closed |
| T-14.3-03 | EoP | OAuth scope | high | mitigate | `drive.readonly` is the sole scope constant in the built plugin (`oauthconfig.go:24,42`) | closed |
| T-14.3-04 | Tampering | seeded gap log | medium | mitigate | GAP-01/02 seeded pre-code in bootstrap commit; unedited at HEAD; no-retro-fit rule in plugin CLAUDE.md | closed |
| T-14.3-05 | Spoofing | public repo creation | medium | accept | Operator-identity-only at blocking-human checkpoint; agent barred from remote/push/gh | closed |
| T-14.4-01 | Info Disclosure | untrusted binary + live credentials | high | accept | Compensating controls verified: read-only scope, clean-room provenance, interstitial disclosure naming both env vars + type-to-confirm | closed |
| T-14.4-02 | Info Disclosure | credential values in files | high | mitigate | Only variable names/references anywhere; `secret: true` asserted; credential-shape scan clean | closed |
| T-14.4-03 | Info Disclosure | Drive content in local index | high | mitigate | Index schema has no full-content column (`schema.go:28-45`); plugin caps previews; contract prohibitions | closed |
| T-14.4-04 | EoP | Drive binary → trusted directory | high | mitigate | Trusted list empty in spec; Makefile bars gdrive from trusted recipes/manifest; no gdrive binary under bin/ | closed |
| T-14.4-05 | Spoofing | binary substitution | medium | mitigate | Pin recomputed from disk on every external launch (`host.go:967,974-983`); operator sha256 check in UAT | closed |
| T-14.4-06 | Repudiation | health state marked passed but unreached | medium | mitigate | Record corrected 2026-08-18: rate-limited row now reads not-reached per the document's own rule (operator confirmed) | closed |
| T-14.5-01 | Repudiation | gap-log import | high | mitigate | 20/20 ids reconcile; import diff removes 0 lines | closed |
| T-14.5-02 | Tampering | published contract edits | high | mitigate | Purely additive (194+/0−), traceable to gap ids; sdk contract test re-run green | closed |
| T-14.5-03 | Tampering | gap text as instructions | medium | mitigate | Per-id dispositions; GAP-13 resolved by reading kernel code, not the log's assertion | closed |
| T-14.5-04 | DoS | roadmap/requirements edits | low | mitigate | 2 insertions, 0 deletions, single sections | closed |
| T-14.6-01 | Info Disclosure | chip-row health visibility | medium | mitigate | Inline-chip floor + 375px browser gate; WR-01 single-source hardening (`af172e5`) verified present | closed |
| T-14.6-02 | Spoofing | long display_name | low | accept | Width cap + truncation bound rendered text; full name stays in accessible description | closed |
| T-14.6-03 | DoS | measure/resize loop | low | mitigate | `shrinkable` has exactly one call site (visible row); clones report natural width; invariant comments present | closed |
| T-14.6-04 | Repudiation | e2e sweep | medium | mitigate | Sweep commit touched only the two named specs; zero `expect(` lines added/removed | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-14-01 | T-14.4-01 | An untrusted external binary necessarily receives live Drive credentials to function; no kernel-level containment exists and the contract says so plainly. Compensating controls (read-only scope, clean-room provenance, interstitial disclosure + type-to-confirm) verified present. This is the residual risk the external tier is built around. | operator (plan-time disposition, audit-verified) | 2026-08-18 |
| AR-14-02 | T-14.1-01 | Operator already fully controls their own config file and process; path selection between operator-owned files adds no privilege. Validation verified path-independent. | operator (plan-time disposition, audit-verified) | 2026-08-18 |
| AR-14-03 | T-14.2-02, T-14.6-02 | Plugin-influenced strings reach only Svelte-escaped interpolation sinks; truncation strictly reduces rendered attacker-chosen text. | operator (plan-time disposition, audit-verified) | 2026-08-18 |
| AR-14-04 | T-14.3-05 | Public repository creation performed only under the operator's own GitHub identity at a blocking-human checkpoint; no agent-held credential exists. | operator (plan-time disposition, audit-verified) | 2026-08-18 |

*Accepted risks do not resurface in future audit runs.*

---

## Audit Observations (non-register)

1. No `## Threat Flags` sections exist in the six SUMMARYs; the auditor compensated
   with an independent diff review (only `eef30c4` touches `kernel/`/`cmd/`, and it
   is registered as T-14.1-01). No unmapped attack surface found.
2. `09-1-mobile-takeover.spec.ts` case 6 uses `dispatchEvent('click')` (bypasses
   actionability checks) — test-fidelity note, tracked by todo
   `2026-08-21-popover-clone-tooltip-intercepts-clicks.md`.
3. 14-02's "exactly one `toHaveAttribute('title'` line" criterion now matches two —
   the second is 14-06's *negative* assertion (a strengthening, not a regression).
4. T-14.4-03's UAT-observation leg is weaker than stated (no index-content
   inspection step); the threat closes on the schema/contract/plugin-cap legs.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-18 | 26 | 26 | 0 | gsd-security-auditor (opus), verify-mitigations mode, ASVS L1, block_on high; T-14.4-06 closed via operator-confirmed record correction |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-18
