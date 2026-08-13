---
phase: 11
slug: external-plugins-the-trust-boundary
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-08-13
---

# Phase 11 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| External plugins directory → kernel process | Binaries the project did not build are resolved and executed with the kernel's own OS privileges | Executable code (untrusted) |
| Loopback HTTP client → `PUT /api/config` | Any local process reaching the loopback API supplies `[sources.<id>] plugin` and pin values | Plugin names, pinned hashes |
| Config file on disk → `config.Load` | Hand-edited `config.toml` supplies the same fields with no UI in the path | Plugin names, `${VAR}` secret references |
| Kernel process environment → plugin subprocess | Anything in the child's environment is readable by that child | Secrets behind `${VAR}` references |
| Kernel → browser SPA / agent callers | Trust facts (tier, hashes, `launch_failure`) cross to clients that must render, never decide | Tier strings, content hashes, env var NAMES |
| Plugin `Describe` response → kernel/UI/operator form | A plugin's self-description influences what the operator is asked to type | Field declarations, display-only defaults |
| Kernel → plugin subprocess `exec.Command` | Resolved path executed with kernel privileges; no sandbox by design (11-CONTEXT.md out-of-scope) | Process execution |
| Fixture module → repo audits / real installations | A test binary must never widen audit scope or reach a trusted plugin directory | Test executable |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-11-01 | Spoofing | `pluginhost` tier assignment | high | mitigate | Tier stored per-instance from the directory the binary resolved from at launch (`host.go` `tier` field, written once); never from `DescribeResponse` or config key | closed |
| T-11-02 | Elevation of Privilege | `DiscoverTiered` / `DescribePluginHandler` | high | mitigate | Directory listing remains the sole launch authority; handler membership check iterates `DiscoverAllTiered` (`kernel/httpapi/config.go`), undiscovered names refused 404 | closed |
| T-11-03 | Elevation of Privilege | `ExcludedPluginBinaries` across tiers | medium | mitigate | Exclusion filter applied in shared discovery for both tiers (`discover_binaries.go:71`) with fixture-exclusion tests | closed |
| T-11-04 | Tampering | External plugins directory default (D-09) | medium | mitigate | Default resolves to `$XDG_DATA_HOME/topos/...` (user-owned platform data dir, `cmd/topos/main.go` `externalPluginsDir`); kernel never creates it implicitly | closed |
| T-11-05 | Information Disclosure | `GET /api/sources` / plugin-types | low | accept | Binary names + tier string only; loopback-only listener is the existing control | closed |
| T-11-06 | Repudiation | Two-tier name collision | medium | mitigate | Shadowed external binary emits named WARN (D-11); trusted wins, covered by `discover_binaries_test.go` shadow test | closed |
| T-11-07 | Tampering | `launch` pin gate | critical | mitigate | SHA-256 recomputed from resolved file before `exec.Command`; mismatch or absent pin refuses launch (`ErrPinMismatch`, unpinned ≡ mismatch) and is recorded by name | closed |
| T-11-08 | Information Disclosure | `exec.Cmd.Env` | critical | mitigate | `allowedEnv` (`host.go:713`) is the sole producer of the child environment: explicit allowlist + only this instance's own `${VAR}` values; kernel environment never copied wholesale | closed |
| T-11-09 | Denial of Service | `Discover` / `Reconcile` | high | mitigate | Pin mismatch is a soft per-instance failure (`LaunchFailures` channel); kernel boot and unrelated config saves unaffected | closed |
| T-11-10 | Spoofing | HTTP-supplied hash | high | mitigate | Pin only compared against a kernel-recomputed disk hash (`binaryhash.go`); no request-body value is launch authority | closed |
| T-11-11 | Information Disclosure | `DescribeResponse.extras` placeholder | medium | mitigate | Declared defaults are display-only by contract (proto comment) and bound to `placeholder` only in the UI (D-14, `extras-form.test.ts`) | closed |
| T-11-12 | Information Disclosure | Canonical config rewrite | high | mitigate | `WriteCanonical` serialises `Store.Raw()`; round-trip test asserts extras keep literal `${VAR}` form (`writer_test.go:169`) | closed |
| T-11-13 | Elevation of Privilege | Session plumbing on env allowlist | medium | accept | `XDG_RUNTIME_DIR`/`DBUS_SESSION_BUS_ADDRESS` are addresses, not secrets; needed by Signal keyring retrieval; no containment claimed this phase | closed |
| T-11-14 | Tampering | Trial (describe-only) launch of unpinned external binary | medium | accept | Identity must be learned before a pin can exist; operator explicitly selected the plugin, directory-listing authority applies, warning interstitial precedes persistence; documented in plugin contract | closed |
| T-11-15 | Information Disclosure | `describePluginResponse.env_var_names` | high | mitigate | Names only; `TestDescribePluginHandler_EnvVarNames..._NeverLeaksValues` asserts values absent from response body | closed |
| T-11-16 | Spoofing | Client-side trust rendering | high | mitigate | `tier`/`binary_hash`/`pinned_hash`/`launch_failure` all kernel-computed; client (`api.ts`, `SourceChip.svelte`) renders only, no trust derivation path | closed |
| T-11-17 | Information Disclosure | Launch-failure merge → `/agent/v1` | medium | mitigate | Grant filtering applied after merge; `agent_live_config_test.go` asserts ungranted sources structurally absent | closed |
| T-11-18 | Tampering | UI action gated on error text | medium | mitigate | Re-pin action gated on closed-vocabulary `launch_failure` field, never `last_error` copy | closed |
| T-11-19 | Repudiation | Documentation drift | medium | mitigate | Contract's env-allowlist section asserted against implemented list (contract tests + `env_test.go`); `make docs-check` in CI | closed |
| T-11-20 | Elevation of Privilege | Fixture binary reaching a real plugins directory | high | mitigate | Built only by its own target into `bin/plugins-external/` (Makefile), never by `build`/`plugins`; module under `testdata/` with README prohibition | closed |
| T-11-21 | Tampering | Repo audit coverage | medium | mitigate | Module sits under audit-skipped `testdata/`; audit scope unchanged; `go test ./internal/audit/...` green in phase gates | closed |
| T-11-22 | Information Disclosure | Proof plugin's environment-reporting corpus | medium | mitigate | Emits variable NAMES only, never values (verified end-to-end by `externalproof_test.go`) | closed |
| T-11-23 | Tampering | Separate module's go.sum supply chain | medium | mitigate | Depends only on packages already in the workspace module graph (sdk, go-plugin, grpc, protobuf); no new third-party dependency | closed |
| T-11-24 | Social Engineering / Info Disclosure | Declared extras defaults | high | mitigate | Default binds to `placeholder` only; source-scan guard asserts no value binding reads it (`extras-form.test.ts` D-14) | closed |
| T-11-25 | Spoofing | Pinned hash written by client | high | mitigate | `setPluginPin` only called with `binary_hash` from a describe response; client has no hashing code; kernel re-verifies from disk at every launch | closed |
| T-11-26 | Repudiation | Adding an untrusted source | medium | mitigate | Confirm step names the binary, shows full hash + env var names, requires binary name typed exactly (`untrusted-add.test.ts` E1/E2/E3) | closed |
| T-11-27 | Tampering | "Save anyway" on an external plugin | medium | mitigate | Control excluded for external-tier plugin types (explicit T-11-27 test in `untrusted-add.test.ts:299`) | closed |
| T-11-28 | Elevation of Privilege | Best-effort describe on picker selection | medium | accept | Same describe-only trial the flow's Next step runs; nothing persisted, failures silent, directory-listing authority bounds what may run | closed |
| T-11-29 | Information Disclosure | Env disclosure copy | low | mitigate | Dialog lists kernel-supplied variable NAMES only | closed |
| T-11-30 | Spoofing | Re-pin write | high | mitigate | Dialog writes back kernel-published `current_hash`; client never hashes; kernel re-verifies from disk at relaunch (`repin.test.ts`) | closed |
| T-11-31 | Repudiation | Re-pin as a routine click | medium | mitigate | Dialog states topos cannot distinguish rebuild from replacement, shows both hashes, requires explicit confirm | closed |
| T-11-32 | Tampering | Action gated on error text | medium | mitigate | Menu item gated on `launch_failure` closed vocabulary; explicit test "never a last_error match (T-11-32)" (`repin.test.ts:134`) | closed |
| T-11-33 | Denial of Service | One bad pin blocking saves or boot | high | mitigate | e2e asserts save succeeds and unrelated sources stream during pin mismatch; `validateMatchConfig` excuses launch-failed instances (`matchconfig.go` `launchFailedNames`) so boot survives | closed |
| T-11-34 | Information Disclosure | Clipboard write | low | accept | Only a content hash is copied — not a secret; clipboard failure no-ops with value still in title attribute | closed |
| T-11-35 | Elevation of Privilege | `ResolveBinary` + `config.Validate` (`Source.Plugin`) | critical | mitigate | CR-01 closure (plan 11-07): `validatePluginBinaryName` rejects non-bare names as `ResolveBinary`'s first statement; `validateSourcePlugins` rejects same shapes at load and `PUT /api/config`; 12 regression tests; independently re-verified by 11-VERIFICATION.md and 11-REVIEW.md re-review | closed |
| T-11-36 | Tampering | `ResolveBinary` `os.Stat` sites | high | mitigate | `info.Mode().IsRegular()` required at all three stat sites; directories and device nodes cannot resolve to a launchable path | closed |
| T-11-37 | Spoofing | Symlink inside plugin dir pointing outside | medium | accept | One-hop symlink follow is by design (e2e harness depends on it); placing a symlink requires write access to an operator-owned directory — equivalent to replacing the binary; content pinning remains the external-tier control | closed |
| T-11-38 | Tampering | TOCTOU between stat/validate and exec open | low | accept | Unavoidable without fd-based exec; external-tier pin re-verified immediately before exec narrows the window to exec's own open; trusted tier repopulated by `make build` on the operator's machine | closed |
| T-11-39 | Repudiation | Rejected save leaving partial `config.toml` | low | mitigate | `Store.Save` runs `dryRunExpand`/`Validate` before `WriteCanonical`; test asserts file byte-identical after a rejected PUT (`config_test.go:476`) | closed |
| T-11-SC | Tampering | npm/pip/cargo installs (all 7 plans) | low | accept | Zero new package-manager dependencies across the phase (per-plan Package Legitimacy checks; `go.mod`/`go.sum`/`package.json` diffs clean apart from the workspace-resolved fixture module) | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-11-01 | T-11-05 | Sources/plugin-types API exposes binary names + tier only; loopback-only listener is the boundary control | plan 11-01 (plan-time disposition) | 2026-08-13 |
| AR-11-02 | T-11-13 | Session-bus addresses on the env allowlist are addresses, not secrets; required by Signal keyring retrieval; no containment claimed this phase | plan 11-02 (plan-time disposition) | 2026-08-13 |
| AR-11-03 | T-11-14 | Describe-only trial launch of an unpinned binary is required to learn its identity before a pin can exist; operator-selected, directory-bounded, pre-persistence warning | plan 11-02 (plan-time disposition) | 2026-08-13 |
| AR-11-04 | T-11-28 | Picker-selection describe is the same bounded trial as the flow's Next step; nothing persisted | plan 11-05 (plan-time disposition) | 2026-08-13 |
| AR-11-05 | T-11-34 | Clipboard carries a content hash, not a secret | plan 11-06 (plan-time disposition) | 2026-08-13 |
| AR-11-06 | T-11-37 | One-hop symlink follow is intentional; symlink placement requires the same local write access as binary replacement; content pinning remains the control | plan 11-07 (plan-time disposition) | 2026-08-13 |
| AR-11-07 | T-11-38 | TOCTOU window between validation and exec is irreducible without fd-based exec; pin re-verified immediately before exec | plan 11-07 (plan-time disposition) | 2026-08-13 |
| AR-11-08 | T-11-SC | No new package-manager dependencies introduced anywhere in the phase | plans 11-01…11-07 (plan-time disposition) | 2026-08-13 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-13 | 40 | 40 | 0 | /gsd-secure-phase (L1 short-circuit: plan-time register, grep-level evidence per threat; T-11-35/36 additionally deep-verified by 11-VERIFICATION.md re-run and 11-REVIEW.md re-review) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-13
