---
phase: 02
slug: two-sources-one-trustworthy-stream
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-07-29
---

# Phase 02 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Source instance → plugin subprocess | SilverBullet/paperless-ngx return user-authored content of arbitrary shape (HTML, scripts, hostile links); the instance may be misconfigured or spoofed | Page/document bodies, tags, metadata |
| Plugin subprocess → kernel (gRPC) | Plugin supplies MIME types, deep links, provenance, and rendition bytes the kernel must not trust blindly | Item metadata, previews, rendition bytes |
| Kernel HTTP API → browser | Source-controlled bytes served from the kernel's loopback origin; includes `/api/items/{id}/content` and agent read-only routes | Rendered content, sync status, `last_error` strings |
| config/env → plugin subprocess | `SB_AUTH_TOKEN` / `PAPERLESS_TOKEN` cross into subprocess environments via `${VAR}` TOML references | Credentials |
| Plugin binaries / Go + npm modules | Locally built subprocesses run with full OS access (go-plugin is a transport, not a sandbox); third-party deps pinned to audited versions | Supply chain |
| Build/verification scripts → shell | `scripts/assert-stylesheet.sh`, `scripts/e2e-smoke.sh`, `scripts/run-with-env.sh` process fetched CSS and `.env` values | Built assets, credentials (never echoed) |

---

## Threat Register

All 35 plan-time threats verified 2026-07-29 by gsd-security-auditor (ASVS L1, block_on: high). Register authored at plan time across the six `<threat_model>` blocks; auditor verified mitigations against the implementation — evidence cited per row.

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-02-01 | Tampering/EoP | silverbullet render → item.go → DetailPane | high | mitigate | goldmark safe defaults + `bluemonday.UGCPolicy()` (`render.go:21-37`); nosniff + inline + `sandbox` CSP (`item.go:184-208`); iframe-only; zero `@html` in `web/src/` | closed |
| T-02-02 | Spoofing/Tampering | silverbullet client.go | high | mitigate | `allowHost` at `CheckRedirect` + `DialContext` backstop, 10-hop cap, `ErrForeignHost`; `outbound_hosts_test.go` passes | closed |
| T-02-03 | Info Disclosure | silverbullet client.go, main.go | high | mitigate | `${SB_AUTH_TOKEN}` env ref only; no request/header logging; errors name path+status only; `client_test.go:110` | closed |
| T-02-04 | Info Disclosure | kernel/index + silverbullet plugin.go | medium | mitigate | Rune-capped snippet only; no body column in items schema (`schema.go:15-30`) | closed |
| T-02-05 | DoS (self) | silverbullet plugin.go | medium | mitigate | `matchConcurrency=4` + errgroup `SetLimit`; `MaxConnsPerHost: 4` | closed |
| T-02-06 | Repudiation | kernel/correlate, kernel/index | medium | mitigate | Source-scoped deletes; one `sync_runs` row per source_type (`store.go:95-114,358-383`) | closed |
| T-02-07 | Tampering | kernel/httpapi/item.go | medium | accept | `text/html` allowlisted but always served under sandbox CSP + nosniff + inline, iframe-consumed | closed |
| T-02-SC (02-01) | Tampering | goldmark, bluemonday, frontmatter, x/sync | high | mitigate | Pinned to versions audited OK/Approved in 02-RESEARCH.md (`plugins/silverbullet/go.mod:6-10`) | closed |
| T-02-08 | DoS/Repudiation | kernel/syncer | high | mitigate | `singleflight.Group.Do` (`coordinator.go:97`); all refresh paths route through `Refresh` | closed |
| T-02-09 | Info Disclosure | kernel/httpapi/sources.go | medium | mitigate | `{name}` validated against `cfg.Sources`; message echoes request value only (`sources_test.go:254`) | closed |
| T-02-10 | Info Disclosure | client.go, scheduler.go | high | mitigate | Errors carry path+status only; scheduler logs source name + error string only | closed |
| T-02-11 | Spoofing/Repudiation | kernel/httpapi/sources.go | medium | mitigate | Health read from `sync_runs`, never from plugin `HealthResponse` (`sources.go:106-108`) | closed |
| T-02-12 | DoS (self) | kernel/config | medium | mitigate | `validatePositiveDuration` + loopback/15m defaults; 4 config tests pass | closed |
| T-02-13 | Tampering | kernel HTTP API bind | medium | accept | Loopback-only/no-auth documented (`docs/api.md:24-38`); non-loopback warning (`main.go:195`); POST routes only dispatch `Refresh` | closed |
| T-02-SC (02-02) | Tampering | golang.org/x/sync | high | mitigate | v0.22.0 pinned — the audited version | closed |
| T-02-14 | Tampering/EoP | web/src components | high | mitigate | Zero `@html` matches; tooltip renders text interpolation only (`SourceHealthChip.svelte:66`) | closed |
| T-02-15 | Info Disclosure | web/src | high | mitigate | `last_error` reaches exactly one text-only surface (`SourceHealthChip.svelte:40`) | closed |
| T-02-16 | Spoofing (user-facing) | format.ts, StreamList | high | mitigate | `streamVariant` uses unfiltered item count + sync status; stale rows marked; tests pass | closed |
| T-02-17 | Tampering | format.ts | low | mitigate | Unrecognised filter values degrade to `null`; never enter a request path | closed |
| T-02-18 | DoS (self) | webspace +page.svelte | low | accept | Poll interval cleared as soon as no source reports `syncing` | closed |
| T-02-19 | EoP | kernel/config, agent.go | high | mitigate | Default-deny by Go zero value; `grantedSourceTypes` in all six agent routes; `agent_test.go` passes | closed |
| T-02-20 | Info Disclosure | agent.go | high | mitigate | Byte-identical envelope to `item.go`; body-comparison tests pass | closed |
| T-02-21 | EoP | agent.go routes | medium | mitigate | `Handoff` read-only; `MountAgentRoutes` registers `r.Get` exclusively | closed |
| T-02-22 | Spoofing | agent API auth boundary | medium | accept | Documented: no authentication on loopback API; grants are authorization layered on top, not authn | closed |
| T-02-23 | Tampering | third-party plugin subprocesses | medium | accept | Accept rationale's required control now present: `docs/plugin-contract.md` states a plugin is a native subprocess with full local OS access — go-plugin is a transport, not a sandbox — same trust decision as the kernel binary (added 2026-07-29, closing the audit finding); RPC allowlist pinned by `sdk/contract_test.go:20`; AST read-only scan walks all of `plugins/` (`readonly_test.go:22`) | closed |
| T-02-24 | Info Disclosure | docs/plugin-contract.md | low | mitigate | "A plugin must never log a credential … at any log level, including debug" (`:433-441`) | closed |
| T-02-05-01 | Tampering (index integrity) | silverbullet plugin.go, correlate.go | high | mitigate | Non-`ErrNotFound` errors → `codes.Unavailable`; correlate `continue`s before delete; 4 tests pass | closed |
| T-02-05-02 | Info Disclosure | silverbullet Match errors | medium | mitigate | `TestMatch_UnavailableError_NeverContainsBearerToken` passes | closed |
| T-02-05-03 | DoS (sync availability) | correlate/scheduler | low | accept | Prior rows intact on error; per-interval retry; manual refresh route | closed |
| T-02-05-04 | Spoofing | silverbullet client | low | transfer | Transfer target (host pinning + redirect guards) verified present and unchanged | closed |
| T-02-05-SC | Tampering (supply chain) | silverbullet go.mod/go.sum | high | accept | Untouched by 02-05 (last touched at 02-01 commit `955028a`) | closed |
| T-02-06-01 | Tampering | scripts/assert-stylesheet.sh | low | mitigate | `set -euo pipefail`, quoted expansions, no `eval`, content only passed to `grep` | closed |
| T-02-06-02 | DoS (UI availability) | web/src/app.css + assertion script | high | mitigate | Shadowing entries removed; collapsed-rule assertion in `assert-stylesheet.sh`; smoke test delegates against served CSS | closed |
| T-02-06-03 | Info Disclosure | health tooltip error text | low | accept | `last_error` sourced from `sync_runs`; token barred by test; loopback bind | closed |
| T-02-06-SC | Tampering | web npm deps | high | accept | `web/package.json`/`package-lock.json` untouched since 01-0x commits | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-02-1 | T-02-07 | `text/html` widening confined by sandbox CSP + nosniff + iframe; plugin is locally-built trusted code | plan 02-01 (verified in audit) | 2026-07-29 |
| AR-02-2 | T-02-13 | Loopback-only unauthenticated API is the documented deployment model for a single-user desktop tool | plan 02-02 (verified in audit) | 2026-07-29 |
| AR-02-3 | T-02-18 | Self-inflicted polling bounded — interval cleared when no source is syncing | plan 02-03 (verified in audit) | 2026-07-29 |
| AR-02-4 | T-02-22 | Agent grants are authorization, not authentication; boundary documented in docs/api.md | plan 02-04 (verified in audit) | 2026-07-29 |
| AR-02-5 | T-02-23 | Third-party plugin binaries are trusted code; containment explicitly disclaimed in docs/plugin-contract.md (control added 2026-07-29 to satisfy the rationale's condition) | plan 02-04 + user (this audit) | 2026-07-29 |
| AR-02-6 | T-02-05-03 | Failed sync leaves prior rows intact; scheduler retries; manual refresh available | plan 02-05 (verified in audit) | 2026-07-29 |
| AR-02-7 | T-02-05-SC / T-02-06-SC | Dependency manifests untouched by the gap-closure plans | plans 02-05/02-06 (verified in audit) | 2026-07-29 |
| AR-02-8 | T-02-06-03 | Tooltip error text sourced from sync_runs, token-free by test, loopback-only exposure | plan 02-06 (verified in audit) | 2026-07-29 |

*Accepted risks do not resurface in future audit runs.*

---

## Audit Notes (non-blocking observations)

1. **Unregistered flag — `ca_cert` (from 02-01-SUMMARY Threat Flags):** new trust input loading a local CA file into the SilverBullet client's `tls.Config.RootCAs`. Verified benign at L1: PEM loaded only if it reads and parses, else falls back to system trust store; zero `InsecureSkipVerify` matches repo-wide. Assign a register ID if this surface changes.
2. **Inaccurate comment** at `kernel/httpapi/item.go:195-196`: cites an iframe `sandbox=` attribute in DetailPane.svelte that does not exist; the CSP `sandbox` response header is the actual (sufficient) control. Fix the comment on next touch.
3. **CSP deviation from plan text:** shipped policy adds `style-src 'unsafe-inline'` over the plan's literal string. Assessed as not weakening T-02-01 (bluemonday strips source-originated styles; `default-src 'none'` retained).
4. **`scripts/assert-stylesheet.sh` runs only via `make smoke`**, not `make build`/`make test` — the T-02-06-02 guard exists and passes but is not yet a build-time error on ordinary invocations.

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-07-29 | 35 | 35 | 0 | gsd-security-auditor (opus, ASVS L1) + orchestrator doc fix for T-02-23 |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-07-29
