---
phase: 15
slug: installed-instance-dev-isolation
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-08-19
---

# Phase 15 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| GitHub release CDN → operator's machine | Untrusted bytes the operator will execute (kernel + plugin binaries) | Executable binaries |
| checksums.txt content → local filesystem paths | Attacker-influenced manifest text becomes local write paths | Path strings |
| `$PREFIX` → the operator's system | Install/uninstall write into a directory shared with unrelated software | Files under the prefix |
| GitHub redirect chain → local tag selection | An HTTP redirect decides which release's bytes get installed | Release tag identity |
| Installer/uninstaller → operator's home/XDG data | Config, index, and plugin stores sit adjacent to everything touched | Personal data, credentials-resolving config |
| Locally built binary → the installed kernel's plugin tiers | A machine-compiled binary crosses into a credential-bearing process | Executable + source credentials |
| Build toolchain → the base install path | A compiler on PATH could silently widen the install's dependency surface | Build execution |
| Dev config → installed instance's config and state | A dev run reads/writes locations holding live personal data | Config, index, chat session stores |
| Dev kernel → the loopback port the installed instance owns | Two kernels contending for one port, browser proxy pointed at the winner | HTTP API traffic |
| Guard escape hatch → the isolation guarantee | An opt-out is a deliberate hole in the isolation control | All of the above |
| Proof scripts / runbook → the operator's real machine and data | Tests boot kernels beside the live instance; a runbook is executed by a human against live data | Real ports, real personal data |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-15-01 | Tampering | install.sh download step | high | mitigate | SHA-256 of every asset checked against the release's own checksums.txt (`sha256sum -c`) before any placement; HTTPS default + `curl -f`. Pinned by install-smoke "corrupted asset" case | closed |
| T-15-02 | Tampering | checksums.txt path parsing | high | mitigate | Allowlist shape (`topos` \| `plugins/[a-z0-9-]+`); absolute/parent-segment paths rejected by name. Pinned by "traversal-shaped manifest" case | closed |
| T-15-03 | Elevation of Privilege | `$PREFIX` placement | medium | mitigate | No escalation invocation anywhere in install.sh; unwritable prefix fails loud naming dir + `sudo make install`. Pinned by "unwritable prefix" case (dir stays 555 and empty) | closed |
| T-15-04 | Denial of Service | replacement of a running binary | medium | mitigate | Temp-name copy + same-directory `mv -f` rename. Pinned by "live replacement" case (new inode, kernel survives) | closed |
| T-15-05 | Spoofing | installed-layout plugin resolution | medium | accept | Resolution probe selects a directory, not a binary; trust remains the kernel's link-time manifest (`VerifyTrustedBinary`), unweakened. See Accepted Risks | closed |
| T-15-06 | Spoofing | latest-tag resolution | high | mitigate | Effective URL validated: exact `https://github.com` host, this repo's release-tag path, refusals named. Pinned by offline validator table (6 refusal shapes) | closed |
| T-15-07 | Tampering | latest-tag prerelease selection | medium | mitigate | Bare three-part `v<maj>.<min>.<patch>` shape enforced by the script itself — nightly/prerelease structurally excluded. Pinned by validator table | closed |
| T-15-08 | Denial of Service (data destruction) | scripts/uninstall.sh | high | mitigate | Closed removal set mirroring install's placement set; non-recursive `rmdir` only; zero recursive removals and zero home/XDG references (negative greps); seeded home/XDG tree byte-identical across full cycle (digest manifest case) | closed |
| T-15-09 | Tampering | uninstall vs. operator files | medium | mitigate | Only `topos-plugin-*` files removed; foreign file + directory survive digest-identical. Pinned by "foreign file" case | closed |
| T-15-10 | Tampering | widened release asset list | medium | mitigate | Confirmed before the edit: `MANIFEST_PLUGIN_BINARIES_PORTABLE` already covers topos-plugin-filesystem, so the published kernel launch-verifies it like the existing four | closed |
| T-15-11 | Elevation of Privilege | Signal binary placement | high | mitigate | Placed in the external tier only (consent-and-pin flow); trusted-dir-shaped destination refused outright naming manifest_unverified; trusted-tier gate untouched | closed |
| T-15-12 | Tampering | Signal binary replaced on disk | high | mitigate | External tier's per-launch SHA-256 pin re-verification (Phase 11 machinery, unchanged); rebuild-means-re-accept stated in script output and both docs | closed |
| T-15-13 | Denial of Service | replacing Signal binary under a live kernel | medium | mitigate | Same temp-name + rename placement as install.sh | closed |
| T-15-14 | Tampering | base install growing a build step | medium | mitigate | Toolchain-tripwire case: failing go/cc/gcc/clang/npm/node shims first on PATH; install must exit 0 with the marker never created — a behavioural gate | closed |
| T-15-15 | Denial of Service (data destruction) | make uninstall-signal | medium | mitigate | Exactly one path removed; planted unrelated file + directory survive digest-identical; no recursive removal (negative grep). Pinned by "Signal removal" case | closed |
| T-15-16 | Information Disclosure | dev run reading installed config | high | mitigate | topos-devguard refuses pre-flight when config path/index/plugin dirs/source stores resolve inside the topos config or state roots; runs regardless of test-seam overrides. Pinned by 15 Go subtests + dev-check case 4 | closed |
| T-15-17 | Tampering | dev run writing installed index/stores | high | mitigate | Same refusal incl. omitted-external-dir default; template documents per-checkout store convention. Post-verification hardening: relative source paths resolve against cwd (the kernel's real base), closing the false-clear gap (15-VERIFICATION.md gap 1) | closed |
| T-15-18 | Tampering | dev run re-linking installed chat session | high | mitigate | Chat store = source store path, covered by T-15-17's guard; template's commented WhatsApp block makes the separate-device shape the default | closed |
| T-15-19 | Denial of Service | port contention with installed instance | medium | mitigate | Dev loop on 7778 across all three naming sites; stale-config port mismatch refused by name in seconds (dev-check case 6, elapsed <30s of a 60s readiness window) | closed |
| T-15-20 | Repudiation | silent bypass of the guard | medium | mitigate | DEV_ISOLATION_BYPASS is the only bypass — total (no per-key form), loud multi-line banner listing every permitted path. Pinned by dev-check case 5 | closed |
| T-15-21 | Denial of Service | false refusal on legitimate read-only source | low | mitigate | Only topos-owned roots in scope; read-only source location outside roots asserted clean by dedicated subtest; component-wise containment (prefix-sharing siblings pass) | closed |
| T-15-22 | Tampering | simultaneity smoke vs. the real machine | high | mitigate | All processes get HOME/XDG pointed into the mktemp tree; ephemeral self-selected ports; real-port safety baseline re-asserted after every case, offender killed by reported pid | closed |
| T-15-23 | Denial of Service | smoke binding the production port | medium | mitigate | Port contract asserted statically by reading both defaults from source — neither port bound | closed |
| T-15-24 | Tampering | migration runbook as data-loss path | high | mitigate | No instruction deletes/moves/renames operator data (negative pattern check: 0 matches); back-out relies on `make uninstall`, itself gated by the byte-identical cycle case | closed |
| T-15-25 | Repudiation | undocumented gates | low | mitigate | docs/testing.md gate-count heading equals gate-subsection count; every Makefile gate target has a subsection (7/7) | closed |
| T-15-SC | Tampering | package-manager installs (all five plans) | low | accept | No new npm/pip/cargo dependency entered the tree in any plan; `npm ci` installs from the committed lockfile only. See Accepted Risks | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-15-01 | T-15-05 | An attacker who can write `$PREFIX/lib/topos/plugins` already has the access needed to replace `$PREFIX/bin/topos` itself; the resolution probe selects a directory, never a binary, and launch trust remains the link-time manifest | 15-01-PLAN.md threat model (plan-time disposition) | 2026-08-18 |
| AR-15-02 | T-15-SC | Phase adds zero new package-manager dependencies; existing `npm ci` invocations install from the committed lockfile only, so no supply-chain legitimacy audit applies | 15-01..15-05 PLAN.md threat models (plan-time disposition) | 2026-08-18 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-19 | 25 | 25 | 0 | /gsd-secure-phase orchestrator (ASVS L1 short-circuit: plan-time register, all mitigations evidence-mapped to passing gate cases; independent corroboration from 15-REVIEW.md — no working traversal/injection/escalation found — and 15-VERIFICATION.md's live gate re-runs incl. the post-gap-closure devguard fix) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-19
