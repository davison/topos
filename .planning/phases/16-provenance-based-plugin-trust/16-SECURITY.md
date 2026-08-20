---
phase: 16
slug: provenance-based-plugin-trust
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-08-20
---

# Phase 16 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| plugins directories → kernel | Operator- or attacker-writable files (binaries, `*.provenance.json`, `*.provenance.sig`) crossing into the kernel's trust decision; both configured directories are pure search paths | Untrusted executable bytes + provenance evidence |
| config.toml → kernel | Operator- or attacker-editable `[plugins] dir` / `external_dir` selecting search locations only, never tier | Configuration |
| release signing key → published artifact | ed25519 private key held only in the topos-plugins GitHub Actions secret store; only the public half is compiled into the kernel | Signing key material |
| tag push → published artifact | Whoever can push a tag to topos-plugins can cause a signature — the stated, documented trust boundary (D-02) | Release provenance |
| kernel process → plugin subprocess | `exec.Command` on a binary the trust decision just approved | Execution control |
| kernel → HTTP clients / browser | `tier`, `launch_failure`, `launch_advisory` fields the web UI branches on | Trust verdict |
| downloaded release payload → installed prefix | Staged assets crossing into `$PREFIX` during `make install` | Release artifacts |
| e2e fixture key → kernel under test | Extra accepted verification key injected at build time via the link-time-only seam | Test key material |
| documentation → operator / third-party author | `docs/plugin-trust.md` (canonical) and `docs/plugin-contract.md` shape deployment and shipping decisions | Trust-model claims |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-16-01 | Spoofing | `VerifySignedProvenance` manifest scan | high | mitigate | `ed25519.Verify` over the manifest file's raw bytes against link-time-only accepted keys (`provenance.go:497`); unknown key/bad signature yield no evidence plus a named diagnostic via `Trust.Diagnostics` | closed |
| T-16-02 | Elevation of Privilege | Renamed signed binary shadowing another plugin name | high | mitigate | Manifest binds `name` → `sha256` (D-05); lookup by resolved binary name (`provenance.go:519`) fails for a renamed artifact | closed |
| T-16-03 | Tampering | Accepted key set injected at run time | high | mitigate | Keys only from compiled-in `embeddedProvenanceKeys` or `-ldflags -X provenanceKeysExtra`; no file/env/config path exists; `OverrideProvenanceKeys` carries never-in-production prohibition | closed |
| T-16-04 | Tampering | Binary swapped between hashing and `exec.Command` (TOCTOU) | medium | accept | Residual final-component window documented as narrowed, not eliminated (Phase 12 `EvalSymlinks` precedent); see Accepted Risks | closed |
| T-16-05 | Information Disclosure | Private signing key handled by `topos-provenance` | high | mitigate | `keygen` writes key at mode `0o600` (`main.go:105`), never prints it; `sign` reads from file or `--key-env` env var, never argv (`main.go:136`) | closed |
| T-16-06 | Denial of Service | Hostile oversized/nested manifest JSON | low | mitigate | `MaxProvenanceManifestBytes` (1 MiB) caps the read before parsing (`provenance.go:462`); malformed candidate is a named diagnostic, never scan-aborting | closed |
| T-16-07 | Repudiation | Junk manifest silently vetoing a valid sibling | medium | mitigate | Per-candidate failures recorded as named diagnostics in `Trust.Diagnostics`, logged by caller; scan continues (D-08) | closed |
| T-16-08 | Elevation of Privilege | `[plugins] dir` config edit | high | mitigate | Tier computed by `EvaluateTrust` per binary; `Dirs` select search location only — `TestResolveBinary_LocationSymmetric` + `TestEscalation_ConfigEditCannotGrantTrust`, re-run green 2026-08-20 | closed |
| T-16-09 | Elevation of Privilege | File dropped into the trusted directory | high | mitigate | No-evidence binary evaluates `TierExternal` wherever it sits, reaching consent-and-pin — `TestEscalation_FileDropCannotGrantTrust` | closed |
| T-16-10 | Spoofing | Name shadowing across the two directories | high | mitigate | Both candidates evaluated; evidence wins, collision logged by name with both paths; CR-01 fix (`1ab93f6`) hardened this: a tamper refusal on either candidate now refuses outright, never falls back — regression tests in both directions | closed |
| T-16-11 | Repudiation | Tamper refusal recategorised as "just untrusted" in listings | medium | mitigate | `DiscoverAllTiered` tags refusal `TierExternal` for listing only; refusal re-asserted at launch, surfaces as `manifest_unverified` with `Tier: TierTrusted` on the wire (16-06 fix) | closed |
| T-16-12 | Denial of Service | Per-call hashing of every binary on picker/describe | low | accept | Personal-scale single-user tool; caching deliberately omitted — a stale trust decision is the worse failure. See Accepted Risks | closed |
| T-16-13 | Tampering | Weakened assertion hiding a regression during suite realignment | medium | mitigate | Acceptance criteria asserted no wire assertion or `t.Errorf` removed; reverting resolver to directory-derived trust makes tests fail (fail-first demonstrated) | closed |
| T-16-14 | Elevation of Privilege | Config-edit escalation path | high | mitigate | `TestEscalation_ConfigEditCannotGrantTrust` — tier unchanged for both `Dirs` placements, unpinned launch refuses before any subprocess; re-run green 2026-08-20 | closed |
| T-16-15 | Elevation of Privilege | File-drop escalation path | high | mitigate | `TestEscalation_FileDropCannotGrantTrust` + browser proof `16-file-drop-external-tier.spec.ts` | closed |
| T-16-16 | Spoofing | Name-shadowing escalation path | high | mitigate | `TestEscalation_ShadowingCannotInheritTrust` — name-bound digest mismatch is a refusal, not a demotion; extended by CR-01 fix with tamper-on-either-collision-candidate subtests | closed |
| T-16-17 | Repudiation | Test passing for the wrong reason (false assurance) | high | mitigate | Fail-first criterion: suite observed RED under deliberately weakened gate, failing test names recorded in 16-03-SUMMARY | closed |
| T-16-18 | Tampering | Leaked trust override bleeding between tests | medium | mitigate | Overrides install through `t.Cleanup`-restored seams (`OverrideBuildManifest`, `OverrideProvenanceKeys`); no `t.Parallel()` in package | closed |
| T-16-19 | Information Disclosure | E2E fixture leaking operator data | low | mitigate | Fixtures use only in-repo build artifacts and temp directories per hermetic-harness rules (D-07) | closed |
| T-16-20 | Information Disclosure | Signing key in workflow logs or argv | high | mitigate | Secret referenced exactly twice in topos-plugins release.yml (`env:` + `secrets.`), passed via environment only, no `set -x`; grep acceptance criteria recorded in 16-04-SUMMARY | closed |
| T-16-21 | Information Disclosure | Private key left on local disk after keygen | high | mitigate | `keygen` wrote 0600 into a temp dir deleted after `gh secret set`; deletion asserted and recorded in 16-04-SUMMARY | closed |
| T-16-22 | Spoofing | Attacker pushing a tag to topos-plugins signs arbitrary bytes | high | accept | The stated trust boundary (D-02), documented in the topos-plugins README. See Accepted Risks | closed |
| T-16-23 | Tampering | TRUST-02 proof passing against a hand-signed local artifact | high | mitigate | Proof requires a successful workflow run URL + assets fetched from that release — `16-04-TRUST02-PROOF.md` records run 32325806543, conclusion `success`, offline re-verification | closed |
| T-16-24 | Tampering | Malformed embedded public key silently disabling the signed arm | medium | mitigate | Test asserts non-empty set, each key exactly `ed25519.PublicKeySize`, unique ids (`provenance_test.go:548-570`) — a bad key fails the build's tests | closed |
| T-16-25 | Elevation of Privilege | Pinned `topos-provenance` version drifting to untrusted revision | medium | mitigate | Workflow invokes the CLI at a pinned module version (acceptance criterion, 16-04-SUMMARY), not `@latest` | closed |
| T-16-26 | Tampering | Unverified bytes placed into `$PREFIX` | high | mitigate | Provenance verification inside the existing verify stage before first placement; payload with unresolvable verifier is a loud abort — install-smoke (17 cases incl. tamper refusal) re-run green 2026-08-20; WR-01 fix (`0ab88a9`) prefers a prior-install/PATH verifier over the staged payload's own copy | closed |
| T-16-27 | Elevation of Privilege | Extra accepted key reaching a production kernel via the e2e seam | high | mitigate | Seam is `-ldflags -X` only, written in one Makefile place (`PROVENANCE_LDFLAGS_VAR`), used only by `e2e`/`gdrive-external-rehearsal` targets; `build`/`build-portable`/release paths do not set it (confirmed: no reference in `.github/workflows/`) | closed |
| T-16-28 | Spoofing | E2E harness reimplementing the signature scheme and drifting | medium | mitigate | Fixture executes the real `topos-provenance` binary; no ed25519 implementation in TypeScript fixtures (grep-confirmed) | closed |
| T-16-29 | Repudiation | Docs asserting the superseded "no publisher verification" model | medium | mitigate | Stale claim removed from `docs/plugin-contract.md` and `binaryhash.go`; `docs/plugin-trust.md` is the single source — repo-wide grep sweep clean, re-confirmed by verifier 2026-08-20 | closed |
| T-16-30 | Denial of Service | Install aborting midway leaving a broken instance | medium | mitigate | Placement begins only after every asset verifies; smoke case asserts target prefix recursive checksum unchanged after a failed run — re-run green 2026-08-20 | closed |
| T-16-31 | Information Disclosure | Ephemeral e2e private key committed to the repository | medium | mitigate | Key generated into `bin/` at `make e2e` time; `git check-ignore` confirms `bin/e2e-provenance.key` is ignored, no key file tracked | closed |
| T-16-06-01 | Spoofing | `GET /api/sources` `tier` for `manifest_unverified` entries | high | mitigate | Both refusal paths set `Tier: TierTrusted` (`provenance.go:648-671`); pinned by two real-path Go tests + browser assertion — re-run green 2026-08-20 | closed |
| T-16-06-02 | Elevation of Privilege | `EvaluateTrust` tamper-refusal branches | high | mitigate | Fix added a reporting field only; returned error and `launch`'s `resolveErr != nil` early return untouched — escalation suite + `pin_test.go` pass unmodified | closed |
| T-16-06-03 | Tampering | The new regression tests themselves | medium | mitigate | Assertions run against real `Discover()`/`launch()` with genuine on-disk digest mismatches; each observed RED before the production edit | closed |
| T-16-06-04 | Information Disclosure | Corrected tier reading as reassurance on the chip | medium | mitigate | Browser spec asserts destructive health tone, contract-exact named cause, and absent re-pin action survive alongside the corrected tier | closed |
| T-16-07-01 | Spoofing | `docs/plugin-contract.md` "Trust tiers" section | high | mitigate | Section states directories are pure search paths, tier from evidence only; negative greps for the three superseded claims clean repo-wide — re-confirmed by verifier 2026-08-20 | closed |
| T-16-07-02 | Elevation of Privilege | Collision / shadow-rule paragraph | high | mitigate | Documents the shipped rule (evidence wins; trusted-first ties; tamper refusal never falls back) — now factually true in code since CR-01 fix (`1ab93f6`) | closed |
| T-16-07-03 | Repudiation | Overclaiming what verification proves | medium | mitigate | Honest-limits and publisher-authentication paragraphs preserved byte-intact, gated by positive greps | closed |
| T-16-07-04 | Tampering | Cross-document reference integrity | medium | mitigate | `## Trust tiers` heading preserved; `check-doc-links.sh` green (59 links across 22 files) 2026-08-20 | closed |
| T-16-SC-01 | Tampering | npm/pip/cargo installs (16-01) | high | accept | No package-manager install tasks; verifier uses stdlib `crypto/ed25519` only. See Accepted Risks | closed |
| T-16-SC-02 | Tampering | npm/pip/cargo installs (16-02) | high | accept | No new module or npm dependency added | closed |
| T-16-SC-03 | Tampering | npm/pip/cargo installs (16-03) | high | accept | `make e2e` installs only the already-pinned Playwright browser | closed |
| T-16-SC-04 | Tampering | npm/pip/cargo installs (16-04) | high | mitigate | New module's only deps are first-party `topos/sdk` + already-vetted `hashicorp/go-plugin` transitives; no net-new third-party package | closed |
| T-16-SC-05 | Tampering | npm/pip/cargo installs (16-05) | high | accept | No new Go module or npm package added | closed |
| T-16-SC-06 | Tampering | npm/pip/cargo installs (16-06) | low | accept | No dependency manifests in `files_modified` or the final diff | closed |
| T-16-SC-07 | Tampering | npm/pip/cargo installs (16-07) | low | accept | Documentation-only plan; one markdown file modified | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-16-01 | T-16-04 | TOCTOU window between hashing and `exec.Command` narrowed (evaluation completes before command construction), not eliminated — matches the Phase 12 `EvalSymlinks` precedent; full elimination requires fd-based exec not portably available | Plan 16-01 (operator-approved threat model) | 2026-08-20 |
| AR-16-02 | T-16-12 | Per-request binary hashing accepted at personal scale; caching deliberately rejected because a stale trust decision is the worse failure | Plan 16-02 | 2026-08-20 |
| AR-16-03 | T-16-22 | "Whoever can push a tag to topos-plugins, plus GitHub's secret store" is the stated first-party trust boundary (D-02), documented in the topos-plugins README | Plan 16-04 / D-02 | 2026-08-20 |
| AR-16-04 | T-16-SC-01/02/03/05/06/07 | Supply-chain rows for plans that add no package-manager dependency; asserted per plan by acceptance criteria on manifests/lockfiles | Plans 16-01..07 | 2026-08-20 |
| AR-16-05 | — (16-REVIEW.md WR-01 residual) | Bootstrap-trust caveat: on a first-ever install with no prior verifier on PATH, the staged payload's own `topos-provenance` is the last-resort verifier — reordering fix (`0ab88a9`) narrows the window; residual risk documented in `docs/install.md` | Code-review fix round, 2026-08-20 | 2026-08-20 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-20 | 44 | 44 | 0 | Claude (secure-phase L1, plan-time register; evidence from direct greps + verifier re-runs same day) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-20
