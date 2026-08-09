---
phase: 07
slug: webspace-builder-ui
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-08-09
---

# Phase 07 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

Register provenance: union of the `<threat_model>` STRIDE tables across all 16 PLAN files (07-01…07-16), authored at plan time. Threat IDs collide across plans (07-05/07-06/07-10 reuse T-07-26…T-07-32; 07-04/07-09 both use T-07-21…T-07-25), so rows are disambiguated as `T-07-NN (07-XX)`. `T-07-SC` (supply-chain) appears once per plan and is verified once phase-wide.

Verification depth (2026-08-09 audit): ASVS L1 presence verification in cited files, supplemented by executing every pinning guard — `go test ./kernel/{config,supervisor,correlate,pluginhost,httpapi}` pass, `go test -race ./kernel/{pluginhost,supervisor}` clean, `npx vitest run` 36 files / 601 tests pass.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| browser → kernel `/api/config` | First request body the kernel accepts that becomes persisted state; loopback-only, no auth (v1 posture) | Full config document (references only, no secret values) |
| kernel → `config.toml` on disk | Kernel is now a writer of a file the operator also hand-edits — two writers, one file | Canonical config serialization |
| kernel process environment → API response | Secrets live only in `Store.Expanded()`/`os.Environ`; responses carry references and booleans only | `${VAR}` references, presence booleans |
| kernel → plugin subprocess (describe/trial-launch) | Config save/describe can spawn a plugin binary from the plugins directory | Binary path, connection fields |
| kernel `/agent/v1` → agent clients | Read-only agent surface with per-request grant checks | Source content gated by grants |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-07-01 (07-01) | Info Disclosure | ConfigHandler GET | high | mitigate | `kernel/httpapi/config.go:116` serializes `Store.Raw()`; booleans-only env probe at `:64`; sentinel assertion `config_test.go:104` | closed |
| T-07-02 (07-01) | Info Disclosure | WriteCanonical | high | mitigate | `kernel/config/writer.go:40` raw-config param; `store.go:187` passes raw, expanded only to swap at `:196` | closed |
| T-07-03 (07-01) | Tampering | concurrent save vs hand-edit | high | mitigate | `store.go:174` content-hash lock → `ErrConfigChangedOnDisk`; `writer.go:57-77` temp+Sync+Rename | closed |
| T-07-04 (07-01) | Tampering | rewrite drops unmodelled keys | high | mitigate | `store.go:178-180` `UnknownKeys` strict-decode probe refuses, naming keys | closed |
| T-07-05 (07-01) | EoP | config write reaching source data | high | mitigate | `contract_test.go:374-391` route allowlist; `config_test.go:572-579` forbids Fetch/Match selectors | closed |
| T-07-06 (07-01) | DoS | save leaving no valid config | medium | mitigate | `store.go:182` `dryRunExpand` + `Validate` before write | closed |
| T-07-07 (07-01) | Info Disclosure | logging a config blob | medium | mitigate | Zero logger invocations in `kernel/config/*` and `httpapi/config.go` | closed |
| T-07-08 (07-01) | EoP | plugins/ placement rights | low | accept | See Accepted Risks Log R-07-01 | closed |
| T-07-09 (07-02) | EoP | describe-plugin exec | high | mitigate | `httpapi/config.go:295-310` `DiscoverBinaries` membership → 404 before exec; `:316` overrides body-supplied Plugin | closed |
| T-07-10 (07-02) | Info Disclosure | trial-launch env | medium | mitigate | `pluginhost/host.go:454` `defer p.Kill()`; response carries type/name/vocabulary only | closed |
| T-07-11 (07-02) | DoS | half-launched apply | high | mitigate | `host.go:191-200` all-launches-before-teardown; `supervisor.go:367-374`; 500 `apply_failed` | closed |
| T-07-12 (07-02) | DoS | bad reload killing kernel | high | mitigate | `store.go:236-243` locals-then-swap; 422 on invalid | closed |
| T-07-13 (07-02) | Tampering | stale rows on re-added id | medium | mitigate | `supervisor.go:563,567` DeleteSourceItems + DeleteSyncRuns | closed |
| T-07-14 (07-02) | Repudiation | apply outcome attribution | low | accept | Acceptance premise FAILS: no apply log line exists (`supervisor.go` has zero logger invocations; `s.logger` only passed through at `:120`, `:367`). Fix or correct the acceptance — see audit note | open — below high threshold (non-blocking) |
| T-07-15 (07-03) | Tampering | unreviewed shadcn drop-in | medium | mitigate | `overlay-primitives.test.ts:174-222` no-hex over 21 files; dep-key-set assertion | closed |
| T-07-16 (07-03) | Spoofing | stored name drives nav | low | mitigate | `lib/last-webspace.ts:56` membership check | closed |
| T-07-17 (07-03) | Info Disclosure | config doc in a form | high | mitigate | Same control as T-07-01 — browser's only config source is `GET /api/config` (Raw) | closed |
| T-07-18 (07-03) | DoS | redirect loop | medium | mitigate | `last-webspace.ts:56-57`; `routes/+page.svelte:59-66` empty state | closed |
| T-07-19 (07-04) | Info Disclosure | SecretField | high | mitigate | `SecretField.svelte:56` text-not-password ref field, `:57` autocomplete off, `:46` boolean badge | closed |
| T-07-20 (07-04) | Info Disclosure | browser holds config | high | mitigate | As T-07-01/17 | closed |
| T-07-21 (07-04) | EoP | step 1 → subprocess | medium | mitigate | Kernel boundary: `httpapi/config.go:295-310` + `host.go:454` | closed |
| T-07-22 (07-04) | Tampering | first allowlist write drops sources | high | mitigate | `lib/config-edit.ts:157-163` seeds from all configured sources | closed |
| T-07-23 (07-04) | Tampering | partial two-step save | medium | mitigate | `AddSourceModal.svelte:310-312` single `putConfig`; `add-source.test.ts:161` | closed |
| T-07-24 (07-04) | DoS | edit leaving config invalid | medium | mitigate | `store.go:182` dry-run | closed |
| T-07-25 (07-04) | Repudiation | cross-webspace blast radius | medium | mitigate | `EditSourceModal.svelte:164` notice before form | closed |
| T-07-26 (07-05) | Tampering | delete leaves dangling refs | high | mitigate | `config-edit.ts:270-285` clears match block and allowlist in every webspace | closed |
| T-07-27 (07-05) | EoP | future mutating route | high | mitigate | `contract_test.go:374-391` explicit 5-route allowlist | closed |
| T-07-28 (07-05) | EoP | agent surface gaining write route | high | mitigate | `contract_test.go:399-401` zero non-GET in agent.go | closed |
| T-07-29 (07-05) | DoS | deleting open webspace | medium | mitigate | `ManageSourcesModal.svelte:155-162` navigates away safely | closed |
| T-07-30 (07-05) | Repudiation | delete blast radius unstated | medium | mitigate | `manage-sources.test.ts:116,139` verbatim dialog copy | closed |
| T-07-31 (07-05) | Repudiation | auth posture undocumented | low | accept | See Accepted Risks Log R-07-02 | closed |
| T-07-27 (07-06) | Tampering | `saveAnyway` id collision | high | mitigate | `instance-id.ts:55-68`; `AddSourceModal.svelte:274-278` returns before upsert; tests at `add-source.test.ts:178,224,240` | closed |
| T-07-28 (07-06) | EoP | agent grants on overwrite | high | mitigate | Collision guard + grants always `{read:false,handoff:false}` (`AddSourceModal.svelte:104,121,180`) | closed |
| T-07-29 (07-06) | Tampering | third write path | medium | mitigate | `add-source.test.ts:217,224,247,254` exactly two call sites, single guarded assignment | closed |
| T-07-30 (07-06) | DoS | lockout after rejected save | low | mitigate | `AddSourceModal.svelte:275-277`; asymmetry pinned in tests | closed |
| T-07-31 (07-06) | Tampering | id resolution reads values | low | accept | See Accepted Risks Log R-07-03 | closed |
| T-07-32 (07-07) | EoP | boot-snapshot grant set | high | mitigate | `agent.go:94,149,204,260,311` per-request `cfgStore.Expanded()`; revocation test passes | closed |
| T-07-33 (07-07) | Info Disclosure | rendition bytes after revocation | high | mitigate | `agent.go:320-323` grant check precedes Fetch | closed |
| T-07-34 (07-07) | Tampering | two config generations per response | medium | mitigate | `agent_live_config_test.go:556-560` first-statement + exactly-one-Expanded | closed |
| T-07-35 (07-07) | Info Disclosure | future handler pre-fix shape | medium | mitigate | `agent_live_config_test.go:473-501` handler-set enumeration | closed |
| T-07-36 (07-07) | Info Disclosure | agent serializing config | low | accept | See Accepted Risks Log R-07-04 | closed |
| T-07-37 (07-08) | Tampering | stale `$state` on submit | high | mitigate | `+page.svelte:161` resetEditSession; `:661-662` render guard + `{#key}`; `edit-modal-reset.test.ts:80-116` | closed |
| T-07-38 (07-08) | EoP | edited instance's grants | high | mitigate | Same fix + `edit-modal-state.ts:42` fresh nested agent copy | closed |
| T-07-39 (07-08) | Tampering | aliased seed reaching config doc | medium | mitigate | `edit-modal-state.ts:42` spread + `:63-66` fresh arrays | closed |
| T-07-40 (07-08) | DoS | reset destroying in-progress typing | medium | mitigate | `EditSourceModal.svelte:88` tracks open only; reads inside `untrack` | closed |
| T-07-41 (07-08) | Info Disclosure | secret value client-side | low | accept | See Accepted Risks Log R-07-05 | closed |
| T-07-21 (07-09) | DoS | Apply vocabulary-failure branch | high | mitigate | `supervisor.go:406,418,425` validate → commitGeneration → still errors | closed |
| T-07-22 (07-09) | DoS | D-07 cleanup failure branches | medium | mitigate | `supervisor.go:387` collected, shared commit at `:418` | closed |
| T-07-23 (07-09) | Tampering | rejected rollback option (b) | medium | mitigate | Exactly one Reconcile call; rejection documented `:285-288,:414-417` | closed |
| T-07-24 (07-09) | Info Disclosure | validation error text | low | accept | See Accepted Risks Log R-07-06 | closed |
| T-07-25 (07-09) | EoP | repair masking rejection | medium | mitigate | `supervisor.go:425-427` returns non-nil after commit | closed |
| T-07-26 (07-10) | Tampering | permanently orphaned index rows | high | mitigate | `supervisor.go:387` cleanup unconditional on every post-Reconcile path | closed |
| T-07-27 (07-10) | Tampering | wrong-instance deletion | high | mitigate | `supervisor.go:362-363` pre-mutation locals; `:562` removedInstances(old,new) | closed |
| T-07-28 (07-10) | Repudiation | cleanup fault masked by rejection | medium | mitigate | `supervisor.go:425` errors.Join | closed |
| T-07-29 (07-10) | DoS | batch abandonment | medium | mitigate | `supervisor.go:561-571` collect-and-continue | closed |
| T-07-30 (07-10) | EoP | cleanup against wrong generation | medium | mitigate | `supervisor.go:359` mutex; ordering `:367→:387→:418` | closed |
| T-07-31 (07-10) | Info Disclosure | joined error text | low | accept | See Accepted Risks Log R-07-07 | closed |
| T-07-32 (07-10) | Tampering | SQL injection in cleanup | low | accept | See Accepted Risks Log R-07-08 | closed |
| T-07-33 (07-11) | Info Disclosure | empty field map reaching plugin | high | mitigate | `correlate.go:213-215` non-participation; `ParticipatesIn:168-176`; test passes | closed |
| T-07-34 (07-11) | Tampering | over-broad validation relaxation | high | mitigate | `types.go:245` three-condition shell; `config.go:413,417`; both named tests pass | closed |
| T-07-35 (07-11) | Tampering | participation widening | medium | mitigate | `config-edit.ts:153,158` wasShell gates seeding | closed |
| T-07-36 (07-11) | DoS | D-20 revert breaks startup | medium | accept | See Accepted Risks Log R-07-09 | closed |
| T-07-37 (07-11) | Repudiation | undocumented semantic change | low | mitigate | `07-CONTEXT.md:46`, `config.example.toml:472`, doc comments | closed |
| T-07-38 (07-12) | Repudiation | blanket catch = false outage | high | mitigate | `routes/+page.svelte:42-47` catch wraps only getConfig | closed |
| T-07-39 (07-12) | DoS | null collection crashing clients | medium | mitigate | `config.go:205-227` normalizes all collections | closed |
| T-07-40 (07-12) | Tampering | serialization shift | medium | mitigate | `writer_test.go:24,95` backup + fixed-point tests | closed |
| T-07-41 (07-12) | Tampering | Validate verdict change | medium | mitigate | `config_test.go:880` table test incl. D-20 shell | closed |
| T-07-42 (07-12) | Info Disclosure | normalization changing response | low | accept | See Accepted Risks Log R-07-10 | closed |
| T-07-43 (07-13) | Info Disclosure | plugin stderr in UI error | medium | mitigate | `host.go:374-375` last line only, connect-failure path only | closed |
| T-07-44 (07-13) | DoS | unbounded stderr buffer | medium | mitigate | `host.go:229` 4096-byte cap, front-discard | closed |
| T-07-45 (07-13) | Tampering | stderr data race | medium | mitigate | Mutex; read strictly after Kill; `-race` clean | closed |
| T-07-46 (07-13) | DoS | blank field spawning doomed proc | low | mitigate | `AddSourceModal.svelte:219-224` before describePlugin | closed |
| T-07-47 (07-13) | Tampering | persisted unlaunchable instance | medium | mitigate | Add and Edit modals both gate on required fields | closed |
| T-07-48 (07-13) | Info Disclosure | secret value seeded | high | mitigate | `plugin-fields.ts` sole default is non-secret path; every token row secret:true no default; table-walking test | closed |
| T-07-49 (07-14) | Repudiation | remove reports success, changes nothing | high | mitigate | `config-edit.ts:215-222` seeds participant set before filtering | closed |
| T-07-50 (07-14) | Info Disclosure | chip row shows non-participants | medium | mitigate | `WebspaceHeader.svelte:145-146` shared participatesIn filter | closed |
| T-07-51 (07-14) | Tampering | over-broad removal | high | mitigate | `config-edit.ts:215-222` only target instance filtered | closed |
| T-07-52 (07-14) | DoS | filtering hides "+" trigger | high | mitigate | `WebspaceHeader.svelte:125` gate reads unfiltered sources | closed |
| T-07-53 (07-14) | Tampering | diverging participation impls | medium | mitigate | Shared `$lib/participation`; agreement test | closed |
| T-07-54 (07-14) | Tampering | narrowing route sources state | medium | mitigate | Fix commits touch zero files under `web/src/routes/` | closed |
| T-07-55 (07-15) | DoS | configured webspace unreachable | high | mitigate | `stream.go:134-137` config half answers first, no sync dependency | closed |
| T-07-56 (07-15) | Spoofing | false service-outage claim | medium | mitigate | Route maps `webspace_not_found` → not-found state; outage copy only on error | closed |
| T-07-57 (07-15) | Info Disclosure | gate true for unconfigured name | high | mitigate | `stream.go:134-139` closed two-set disjunction; index errors → 500 | closed |
| T-07-58 (07-15) | Tampering | three surfaces drifting | medium | mitigate | One definition; `WebspaceExists` referenced from exactly one non-comment line | closed |
| T-07-59 (07-15) | Repudiation | docs state stale 404 boundary | low | mitigate | `docs/api.md:154,257,869` | closed |
| T-07-60 (07-16) | Info Disclosure | de-participated rows survive save | high | mitigate | `supervisor.go:404` synchronous purge before response | closed |
| T-07-61 (07-16) | Repudiation | success reported, old view served | high | mitigate | Purge + quiet refetch (`+page.svelte:509`) | closed |
| T-07-62 (07-16) | DoS | synchronous eager resync | high | mitigate | `supervisor.go:441` Refresh stays detached; purge does zero plugin RPC | closed |
| T-07-63 (07-16) | Tampering | over-broad purge | high | mitigate | `supervisor.go:521-523` shared predicate, true→false flip only | closed |
| T-07-64 (07-16) | Tampering | purge resurrecting deleted webspace | medium | mitigate | `supervisor.go:500-505` intersection of both configs | closed |
| T-07-65 (07-16) | Tampering | two participation definitions | medium | mitigate | One exported `correlate.ParticipatesIn`, both consumers | closed |
| T-07-66 (07-16) | DoS | background refetch blanks stream | medium | mitigate | Quiet flag skips loading transition; failure leaves screen untouched | closed |
| T-07-SC (all 16 plans) | Tampering | package installs | high | mitigate | Phase-wide dep diff = one line (`go-toml/v2` indirect→direct, commit 08e17aa, as 07-01 declared); go.sum and npm manifests untouched; dep-key-set test | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

All acceptance premises were verified against the code in the 2026-08-09 audit (not taken from plan rationale). T-07-14 (07-02) is NOT logged here — its premise failed verification and it remains open (non-blocking).

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| R-07-01 | T-07-08 (07-01) | Phase neither widens nor narrows who can place a binary in `plugins/`; 07-02's new execution trigger is gated by directory-listing authority (T-07-09) | plan-time register, audit-verified | 2026-08-09 |
| R-07-02 | T-07-31 (07-05) | `docs/api.md:24-34` documents loopback-only/no-auth on both namespaces; mutating surface inherits it (implicit rather than literal sentence) | plan-time register, audit-verified | 2026-08-09 |
| R-07-03 | T-07-31 (07-06) | `resolveNewInstanceId` reads only `cfg.sources` map keys (`instance-id.ts:56,60`), never values | plan-time register, audit-verified | 2026-08-09 |
| R-07-04 | T-07-36 (07-07) | No agent handler serializes the expanded config; payloads are purpose-built structs (`agent.go:118,181,242,294`) | plan-time register, audit-verified | 2026-08-09 |
| R-07-05 | T-07-41 (07-08) | Edit seed copies the `${VAR}` reference verbatim; SecretField's value prop is the variable name, never a secret value | plan-time register, audit-verified | 2026-08-09 |
| R-07-06 | T-07-24 (07-09) | `ValidateMatchConfig` error text names only operator-authored config identifiers (`matchconfig.go:78,98,152,155`) | plan-time register, audit-verified | 2026-08-09 |
| R-07-07 | T-07-31 (07-10) | Joined cleanup error text names instance/webspace ids only (`supervisor.go:525,564,568`) | plan-time register, audit-verified | 2026-08-09 |
| R-07-08 | T-07-32 (07-10) | Cleanup DELETEs are parameterized (`kernel/index/store.go:299,312` `WHERE source = ?`) | plan-time register, audit-verified | 2026-08-09 |
| R-07-09 | T-07-36 (07-11) | Reverting the D-20 shell exemption after shells exist causes startup failure — deliberate, documented (`07-CONTEXT.md:46`, `config.example.toml:472`) | plan-time register, audit-verified | 2026-08-09 |
| R-07-10 | T-07-42 (07-12) | Kernel-side normalization leaves the config response body unchanged: same fields, same raw `${VAR}` references (`httpapi/config.go:115-124`) | plan-time register, audit-verified | 2026-08-09 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-09 | 108 | 107 | 1 (low, non-blocking) | gsd-security-auditor (opus), ASVS L1 + executed guards |

Audit notes (2026-08-09):

- **T-07-14 (07-02), open non-blocking:** the plan's acceptance asserts "Apply logs the instance names added, changed and removed plus any error string" — no such log line exists; `kernel/supervisor/supervisor.go` contains zero logger invocations. Remediation: either add the apply-outcome log line to `Supervisor.Apply` (names-and-errors-only discipline, matching `Scheduler.refreshAndLog`) or correct this entry to accept that apply outcomes are unlogged. Does not block ship.
- **Informational (no threat row):** `config-edit.ts:280` reads `ws.sources.includes(...)` without a null guard, unlike its siblings in `participation.ts`. Unreachable against a matching kernel binary — 07-12's normalization (`kernel/config/config.go:220-222`) guarantees no null collections in `GET /api/config`.
- **Unregistered surface check:** the phase's entire non-GET route set (`PUT /api/config`, `POST /api/config/reload`, `POST /api/config/describe-plugin`, two pre-existing refresh routes) is mechanically pinned by `contract_test.go:374-391` with `/agent/v1` asserted GET-only at `:399-401`. Every route maps to a registered threat.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed (T-07-14 is low severity, below the `high` block threshold)
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-09
