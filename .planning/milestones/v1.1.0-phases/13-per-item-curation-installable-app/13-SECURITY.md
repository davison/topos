---
phase: 13
slug: per-item-curation-installable-app
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-08-15
---

# Phase 13 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

Register authored at plan time: all 8 plans (13-01 … 13-08) carry a `<threat_model>` block. No `## Threat Flags` entries in any SUMMARY. Threat IDs were reused across plans (e.g. T-13-20 names a different threat in 13-04 vs 13-07); rows below are suffixed with their plan of origin where a collision exists.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| browser → kernel `/api` | Untrusted request bodies and query values (webspace name, item ids, `?view=`) cross into SQL parameters and query dispatch | item ids, webspace names, view selectors |
| kernel → `/agent/v1` consumers | An automated caller reads a grant-filtered mirror of the same index | stream/search results (included view only) |
| plugin subprocess → kernel sync path | A plugin's reported item set drives which marks are swept | per-source item sets |
| service worker → kernel `/api` | A cache layer sits between the app and the API | stream/API responses (must never be cached) |
| npm registry → build output | Build-time dependencies produce code shipping inside the kernel binary | JS bundle, SW, manifest, icons |
| filesystem → kernel launch gate | Any binary in a configured plugin directory is a candidate subprocess with full OS access | plugin binaries |
| config file / `PUT /api/config` → plugin directory resolution | An operator-editable value chooses which directories are consulted | `plugins.dir` paths |
| build pipeline → kernel binary | The link-time manifest's contents are fixed at build time | trusted-binary SHA-256 digests |
| kernel `GET /api/sources` → browser | Trust state crosses into a rendering layer that must not soften or hide it | launch failures/advisories |
| published documentation → third-party plugin authors | Contract claims shape what authors and operators believe is guaranteed | security claims |
| browser DOM → kernel `/api` (deferred callbacks) | The undo toast (alive 5000ms across route changes) issues a mark write and stream read from state captured earlier | item-id sets, webspace names |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-13-01 | Tampering | mark SQL (`kernel/httpapi/marks.go`, `kernel/index/store.go`) | high | mitigate | All mark statements bind `?` placeholders (`store.go:419,457`); handler rejects non-`excluded` kind and non-`add`/`remove` action, trims/rejects blank ids | closed |
| T-13-02 | Elevation of Privilege | route registration (`kernel/httpapi/routes.go`) | high | mitigate | Marks route registered on `/api` only, above `MountAgentRoutes` (`routes.go:135`); `TestRoutesGuard_NonGetRoutesScopedToConfig` + `TestContract_MutatingRoutesAreConfigScoped` pin the non-GET allowlist | closed |
| T-13-03 | Denial of Service | `POST /api/webspaces/{webspace}/marks` | medium | mitigate | `maxMarksItemIDs = 1000` (`marks.go:18`); over-cap rejected 400 before any transaction | closed |
| T-13-04 | Information Disclosure | `/agent/v1` stream mirror | medium | mitigate | Exclusion filter lives in `index.Store.StreamItems`/`Search`; both agent call sites pass `index.ViewIncluded` explicitly (`agent.go:131,234`); no agent-facing view parameter | closed |
| T-13-05 | Tampering | `item_marks` durability | medium | mitigate | No FK on `item_id` (`schema.go:93`, deliberate); absent from `rebuildOnSchemaChange` drop list; only deletion paths are explicit include write and healthy-sync sweep | closed |
| T-13-SC (13-01) | Tampering | npm install of `sonner` block / `svelte-sonner` | high | mitigate | shadcn-svelte official registry sole component source; 13-RESEARCH Package Legitimacy Audit: all `OK`, no `[ASSUMED]`/`[SUS]`/`[SLOP]` | closed |
| T-13-13 | Tampering | `?view=` parsing (`kernel/httpapi/stream.go`) | medium | mitigate | `parseStreamView` matches a closed two-member set, rejects 400 otherwise; value is a branch selector, never SQL text | closed |
| T-13-14 | Tampering | prune sweep driven by plugin-reported item sets | high | mitigate | `pruneItemMarksTx` runs only inside `ReplaceWebspaceSourceItems`' transaction, scoped by kernel-owned `webspaceName`/`source` (subquery on `items.source`, never item self-reported Source); Match failure never reaches the call | closed |
| T-13-15 | Denial of Service | prune sweep's generated IN placeholder list | low | accept | See Accepted Risks AR-13-1 | closed |
| T-13-16 | Tampering | bulk mark write from the action bar | medium | mitigate | Kernel re-validates every id shape and caps the array on the same path as the single-item write (`marks.go`) | closed |
| T-13-17 | Denial of Service | double-submit of a bulk write | low | mitigate | `markBusy` disables controls for the request duration; toast Undo action self-disables via same-id toast upsert with no-op onClick (`toast.ts`) | closed |
| T-13-18 | Information Disclosure | client-side rendering of the excluded bucket | low | accept | See Accepted Risks AR-13-2 | closed |
| T-13-19 | Spoofing | toast body text | low | mitigate | `toast.ts` builds bodies from fixed enumerated verb strings + integer count only (`markPhrase`); no item titles or plugin-supplied text can render in a toast | closed |
| T-13-20 (13-04) | Tampering | Workbox service worker scope | high | mitigate | Precache app shell only; no `runtimeCaching` entry (`web/vite.config.ts` workbox block); `navigateFallbackDenylist: [/^\/api\//]`; `13-pwa-manifest-sw.spec.ts` asserts zero Cache Storage entries under `/api/` after a live stream fetch | closed |
| T-13-21 (13-04) | Information Disclosure | generated manifest and icon assets | medium | mitigate | Manifest defined inline in `vite.config.ts` (no external URLs present); icons generated at build time from `web/static/app-icon.png` via `pwa-assets.config.ts`; served from the kernel's embedded filesystem. Note: mitigation verified by direct config inspection; the plan's "no absolute external URL" check is not a literal automated assertion (see audit trail) | closed |
| T-13-22 (13-04) | Spoofing | install eligibility over a non-secure context | low | accept | See Accepted Risks AR-13-3 | closed |
| T-13-SC (13-04) | Tampering | npm install of `vite-plugin-pwa`, `@vite-pwa/sveltekit`, `@vite-pwa/assets-generator`, `workbox-*` | high | mitigate | 13-RESEARCH Package Legitimacy Audit against live npm registry: all `OK`, no postinstall scripts, known GitHub orgs; versions pinned in committed lockfile | closed |
| T-13-23 (13-04) | Denial of Service | autoUpdate reload loop | medium | mitigate | Reload driven by SW activation (revision change only); update toast fires from post-activation callback (`web/src/routes/+layout.svelte` registerSW wiring); human-verify checkpoint watched for repeated reload | closed |
| T-13-06 | Elevation of Privilege | file drop into the trusted plugin directory | critical | mitigate | Launch gate calls `VerifyTrustedBinary` before any `exec.Command`; absent/mismatched digest → `ErrManifestUnverified`, no subprocess constructed (`host.go:888-925`); covers the add-source picker's trial launch; pinned by `manifestgate_test.go` and `13-manifest-unverified.spec.ts` | closed |
| T-13-07 | Elevation of Privilege | `plugins.dir` retargeted via config edit or `PUT /api/config` | high | mitigate | Trust derives from the link-time manifest digest, not directory of origin — binaries in a retargeted directory fail manifest verification and refuse to launch | closed |
| T-13-08 | Spoofing | same-named trusted-directory binary shadowing a pinned external plugin | high | mitigate | Shadowing binary must itself verify against the manifest; verified collisions surface as structured `LaunchAdvisoryShadowed` (`host.go:77`, `discover_binaries.go:387-406`) | closed |
| T-13-24 (13-05) | Tampering | binary swapped between verification and exec (TOCTOU) | medium | accept | See Accepted Risks AR-13-4 | closed |
| T-13-25 (13-05) | Elevation of Privilege | a kernel built without a manifest | high | mitigate | Empty manifest verifies nothing — every trusted-tier launch refuses by name and logs it (`manifest.go:26,86`); deliberately no fallback to directory-derived trust | closed |
| T-13-26 | Tampering | manifest built from the wrong binary set | medium | mitigate | Build recipes drive `cmd/topos-manifest` from explicit binary lists (`Makefile:33-35`); generator exits non-zero with zero arguments (`main.go:28-29`) | closed |
| T-13-27 | Spoofing | plugin-declared identity influencing trust | high | mitigate | Verification consumes only the operator-configured binary name and on-disk bytes (`VerifyTrustedBinary(src.Plugin, binPath)`); nothing from Describe participates; gRPC contract unchanged | closed |
| T-13-28 | Spoofing | chip health tone precedence | high | mitigate | Both new states extend the single `healthTone`/`isAdvisoryOnly` chain (`format.ts:161,171`); `match-advisory.test.ts` asserts unreachable, errored, pin-mismatched, and manifest-unverified each outrank the shadowed advisory | closed |
| T-13-29 | Repudiation | refuse-to-load visibility | high | mitigate | `13-manifest-unverified.spec.ts` asserts the refusal appears in BOTH the kernel's captured log output and the chip tooltip (D-13 log-AND-UI) | closed |
| T-13-30 | Information Disclosure | overstated security claims in published docs | medium | mitigate | `docs/plugin-contract.md:269` states the manifest match is an integrity control, not publisher authentication, mirroring `binaryhash.go`'s framing | closed |
| T-13-31 | Elevation of Privilege | a remedial UI action applied to the wrong state | high | mitigate | Re-pin action stays gated on `isPinMismatch` alone (`SourceChip.svelte:121,128`); `13-manifest-unverified.spec.ts` asserts no re-pin action for a manifest-unverified source | closed |
| T-13-20 (13-07) | Tampering | undo toast's mirror write target webspace | high | mitigate | `handleExclude`/`handleInclude`/`handleBulkPrimary` bind webspace name + navigation generation to local constants at handler entry, before the first await; kernel independently re-validates `{webspace}` and binds ids as SQL parameters; pinned by `13-undo-across-webspace-switch.spec.ts` (tests 1–3) | closed |
| T-13-21 (13-07) | Repudiation | cross-webspace mark write with no user-visible signal | medium | mitigate | Snapshot fix removes the misdirection outright; excluded-items view (D-02) is the durable, inspectable record of every mark; spec test 3 asserts the corrupting `add` direction specifically | closed |
| T-13-22 (13-07) | Information Disclosure | regression spec fixture data | low | accept | See Accepted Risks AR-13-5 | closed |
| T-13-23 (13-08) | Denial of Service | `load()` stranded-skeleton state (G-13-1) | medium | mitigate | Entry guard `if (gen !== navGeneration) return;` at the top of `load()` makes a stale-generation call a true no-op for every caller (`+page.svelte:902-903`); pinned by spec test 4 (empty-webspace B4, asserts the rendered stream) | closed |
| T-13-24 (13-08) | Tampering | undo toast's mirror write target webspace (re-assertion) | high | mitigate | Same mitigation as T-13-20 (13-07), re-asserted not re-implemented; the 13-08 entry guard completes the safety argument for passing the snapshotted generation to `load` | closed |
| T-13-25 (13-08) | Information Disclosure | extended spec fixture data (empty webspace pair) | low | accept | See Accepted Risks AR-13-5 | closed |
| T-13-SC (13-08) | Tampering | npm/pip/cargo installs | n/a | accept | No packages installed; `make e2e` runs `npm ci` against the committed lockfile only | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-13-1 | T-13-15 | Prune sweep's IN placeholder list is bounded by what the source itself just reported inside the same transaction; SQLite's parameter limit is far above realistic personal-data sync sizes, and a breach fails the sync loudly rather than corrupting state | plan 13-02 threat model (plan-time) | 2026-08-15 |
| AR-13-2 | T-13-18 | Excluded bucket is the same user's own data in the same webspace over the same loopback API; showing it is the feature (KERN-10); no additional data leaves the machine | plan 13-03 threat model (plan-time) | 2026-08-15 |
| AR-13-3 | T-13-22 (13-04) | Browsers refuse SW registration/install over plain-HTTP LAN — the browser's own protection working correctly; UI-14 documents it; workarounds are user-provided TLS, not a kernel HTTPS mode (PD-07) | plan 13-04 threat model (plan-time) | 2026-08-15 |
| AR-13-4 | T-13-24 (13-05) | TOCTOU window between hash and exec: hash computed immediately before subprocess construction, same window the shipped external-tier pin check already accepts; full closure would require holding an fd through exec; residual window requires local write access at the exact launch moment; documented in `binaryhash.go` and the published contract as an integrity control, not publisher authentication | plan 13-05 threat model (plan-time) | 2026-08-15 |
| AR-13-5 | T-13-22 (13-07), T-13-25 (13-08) | e2e specs boot a hermetic kernel against `topos-plugin-mock`'s fixed synthetic corpus in a temp directory on an ephemeral loopback port — no real personal data, no network egress, nothing persisted outside the fixture temp dir | plans 13-07/13-08 threat models (plan-time) | 2026-08-15 |
| AR-13-6 | T-13-SC (13-08) | No packages installed by plan 13-08; `make e2e` runs `npm ci` against the committed lockfile only, so the package-legitimacy gate has no new entries | plan 13-08 threat model (plan-time) | 2026-08-15 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-15 | 36 | 36 | 0 | /gsd-secure-phase L1 (orchestrator grep-depth; short-circuit — register authored at plan time, ASVS L1) |

**2026-08-15 audit notes:**
- All 8 plans carried `<threat_model>` blocks; no SUMMARY `## Threat Flags` entries. threats_open: 0 at ASVS L1 → auditor subagent not spawned per short-circuit rule.
- Evidence nuance on T-13-21 (13-04): the plan claimed "an acceptance criterion asserts the built manifest contains no absolute external URL". No literal automated assertion of that sentence exists; the mitigation itself is directly verifiable (inline manifest in `web/vite.config.ts` contains no external URL, icons generated from the in-repo `web/static/app-icon.png`, assets served from the kernel's embedded filesystem), and `13-pwa-manifest-sw.spec.ts` pins manifest serving and API-free Cache Storage. Classified closed on mitigation substance; a future hardening pass could add the literal manifest-content assertion.
- Threat-ID collisions across plans (T-13-20/21/22 in 13-04 vs 13-07; T-13-23/24/25 in 13-04/13-05 vs 13-08) are disambiguated by plan suffix in the register above.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-15
