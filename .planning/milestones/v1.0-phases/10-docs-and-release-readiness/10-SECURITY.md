---
phase: 10
slug: docs-and-release-readiness
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-08-12
---

# Phase 10 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| GitHub Actions runner → repository contents | Workflow-minted token can create tags and releases | GITHUB_TOKEN (contents: write) |
| Marketplace action → runner | Third-party action executes with the job's token and filesystem | Token, workspace files |
| Published release asset → downloader's machine | User executes a downloaded binary against their own personal data stores | Binaries, checksums |
| Documentation → operator's config and trust decisions | Operators reproduce documented config verbatim | Credential shapes, cert-trust guidance |
| External security researcher → maintainer | Vulnerability report must not become public first | Report content |
| Operator's shell → GitHub repository metadata | Sync script mutates milestones with operator's gh credentials | Milestone titles/state |
| README → new user's first actions | Instructions run verbatim on the user's machine | Env vars, binaries, scripts |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-10-01 | Elevation of Privilege | release.yml / nightly.yml token scope | high | mitigate | Job-level `permissions: contents: write` only (release.yml:22-23, nightly.yml:51-52); no workflow-level grant, no write-all | closed |
| T-10-02 | Tampering | Marketplace action supply chain | high | mitigate | Official `actions/*` at ci.yml's exact @v7 pins only; release publish via preinstalled `gh` CLI | closed |
| T-10-03 | Tampering/Repudiation | Release binary integrity | medium | mitigate | sha256sum over exact asset list → checksums.txt in the same publishing step (release.yml:47-56, nightly.yml:75-89) | closed |
| T-10-04 | Information Disclosure | Credential leak into workflow logs | medium | mitigate | Only `secrets.GITHUB_TOKEN` referenced in any workflow (2 hits, both automatic token) | closed |
| T-10-05 | Tampering | Shipped Signal binary vs system SQLCipher | high | mitigate | Option-b: Signal excluded structurally (Makefile plugins-portable omits it; neither workflow's ASSETS names it); `make signal` documented | closed |
| T-10-06 | Denial of Service | Unconditional nightly builds | low | mitigate | check-changes job gates the build job (`needs:` + `if: outputs.changed`) | closed |
| T-10-SC | Tampering | Package-manager installs in CI | medium | accept | Zero dependency changes this phase (verified via git log over module manifests); no apt-get anywhere (option-b held); npm ci lockfile-pinned | closed |
| T-10-07 | Information Disclosure | Config examples in docs/plugins/ | high | mitigate | All credential keys use ${VAR} env-expansion form; no literal tokens or real hostnames (grep-verified across all seven files) | closed |
| T-10-08 | Information Disclosure | signal.md key handling | medium | mitigate | Page states affirmatively nothing secret is configured; key resolved at runtime from Signal Desktop's own files | closed |
| T-10-09 | Spoofing | Cert-verification bypass guidance | high | mitigate | ca_cert pinning documented as the only path; only "insecure/skip-verify" hit repo-wide is the sentence forbidding it | closed |
| T-10-10 | Tampering | WhatsApp path colliding with Signal store | high | mitigate | whatsapp.md Gotchas + Security state path isolation as load-bearing, naming Signal | closed |
| T-10-11 | Repudiation | Operator pages drifting from reference | medium | mitigate | Link-don't-duplicate enforced; all pages cite config.example.toml; length ceiling holds (max 75 lines of 120) | closed |
| T-10-12 | Information Disclosure | Private disclosure channel | high | mitigate | SECURITY.md links advisory intake; PVR verified live enabled ({"enabled":true}); advisory URL HTTP 200 | closed |
| T-10-13 | Repudiation | Acknowledgement window | medium | mitigate | SECURITY.md:34 states 7-day acknowledgement target | closed |
| T-10-14 | Spoofing | Invented reporting channel | medium | mitigate | Zero mailto/PGP/email/keybase hits; advisory intake sole channel | closed |
| T-10-15 | Information Disclosure | v1 boundary presented as unknown | low | accept | Boundary stated as deliberate and documented; four mattering report classes named | closed |
| T-10-16 | Tampering | Scaffold instructions bypassing embed | medium | mitigate | web/README.md is an 11-line pointer; zero scaffold remnants (grep-verified) | closed |
| T-10-17 | Tampering | Milestone create clobbering v1.0 | medium | mitigate | Lookup-by-title (state=all) precedes both write paths (sync-milestones.sh:82-95) | closed |
| T-10-18 | Tampering | Milestone delete orphaning issues | high | mitigate | No delete path exists in script — capability absent, not guarded | closed |
| T-10-19 | Elevation of Privilege | Script vs unintended repository | medium | mitigate | Repo pinned with one named env override; echoed before first mutation | closed |
| T-10-20 | Repudiation | Signal decision surviving only in SUMMARY | medium | mitigate | docs/releasing.md:98-122 records dated decision with reason | closed |
| T-10-21 | Information Disclosure | Credentials in sync script | low | mitigate | No token/secret/netrc references; auth delegated to gh CLI | closed |
| T-10-22 | Spoofing | README softening security disclosure | high | mitigate | All four boundary elements intact (README.md:153-157): loopback bind, no v1 API auth, LAN deliberate non-decision, warn-not-refuse | closed |
| T-10-23 | Tampering | Binary download without integrity check | high | mitigate | README Install names checksums.txt + `sha256sum -c` step, matching release.yml's real asset list | closed |
| T-10-24 | Information Disclosure | README config examples | medium | mitigate | Env vars with placeholders; hosts are *.example.lan; sole real host is the public Proton webmail URL | closed |
| T-10-25 | Tampering | README referencing nonexistent scripts | medium | mitigate | Zero scripts/*.sh references in README; all live references elsewhere resolve (manually verified) | closed |
| T-10-26 | Repudiation | Documentation link rot | medium | mitigate | check-doc-links.sh in make docs-check + CI test job; negative control independently reproduced (exit 1 on injected broken link) | closed |
| T-10-27 | Spoofing | Credits link to stale/squatted URL | low | mitigate | openGSD URL fetched live during audit: HTTP 200 | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-10-01 | T-10-SC | Zero new dependencies this phase; npm ci is a lockfile-pinned reinstall of already-audited deps; no apt-get added (option-b) | plan 10-01 (planner), re-verified by auditor | 2026-08-12 |
| AR-10-02 | T-10-15 | Loopback/no-auth v1 boundary stated as deliberate; report classes that matter are named for triage clarity | plan 10-03 (planner), re-verified by auditor | 2026-08-12 |

---

## Unregistered Flags (WARNING — outside plan-time register, does not count toward threats_open)

**UF-10-01 — Real personal data committed to the public repository in `docs/ss/*.png` (OPEN, awaiting operator decision).**
`docs/ss/1.png` renders real paperless-ngx content (an AXA/Inter Partner Assistance insurance policy with firm reference number and UK branch address, plus a stream item reading "Hello Mr D Davison" — the operator's real name). `docs/ss/2.png` shows a real inbox item from "Range Rover UK". Exposure is live: the repo is public and commit `0379bbb` is on `origin/main`; README.md embeds both on the landing page. This post-dates the plan-time register (screenshots were added after planning, outside every plan's threat model) and cuts against the project's own constraint ("no personal content leaves the user's machines"). Remediation requires replacement screenshots (mock/redacted data) **plus** history rewrite or repo-side blob purge — in-tree deletion alone leaves the blobs fetchable.

Residual observations (consistent with declared mitigations, recorded for future hardening): nightly.yml's check-changes job inherits the repo default workflow permission (currently `read` — verified live; adding `permissions: contents: read` would make it self-contained); action pins are mutable @v7 tags per ci.yml convention (SHA pinning would close tag-repoint); checksums.txt travels the same channel as binaries (defends corruption, not a release-publishing attacker); check-doc-links.sh checks markdown link targets only, not bare code-fence script mentions.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-12 | 27 | 27 | 0 | gsd-security-auditor (opus), ASVS L1, block_on high |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-12
