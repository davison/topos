---
status: complete
phase: 11-external-plugins-the-trust-boundary
source: [11-01-SUMMARY.md, 11-02-SUMMARY.md, 11-03-SUMMARY.md, 11-04-SUMMARY.md, 11-05-SUMMARY.md, 11-06-SUMMARY.md, 11-07-SUMMARY.md]
started: 2026-08-13T16:14:54Z
updated: 2026-08-13T16:16:14Z
---

## Current Test

[testing complete]

## Tests

### 1. UI design-contract conformance (E1-E6)
expected: All six Phase 11 UI elements (E1 untrusted confirm interstitial, E2 trust badge, E3 picker Untrusted label, E4 binary-changed/re-pin walkthrough, E5 pinned-hash footer, E6 extras form) render as the design contract specifies, including the corrected E4 walkthrough and E5 footer shipped by plan 11-06.
result: pass
coverage_id: 11-06/D6

### 2. [11-01/D1] Two-tier plugin discovery/launch: a binary present only in the external directory is discovered, launched, and syncs exactly like an in-repo binary; a name collision resolves to the trusted copy with a named warning log 
expected: Two-tier plugin discovery/launch: a binary present only in the external directory is discovered, launched, and syncs exactly like an in-repo binary; a name collision resolves to the trusted copy with a named warning log 
result: pass
source: automated
coverage_id: 11-01/D1
verified_by: kernel/pluginhost/tier_test.go; kernel/pluginhost/discover_binaries_test.go; kernel/supervisor/externaltier_test.go; web/e2e/specs/11-external-tier-badge.spec.ts

### 3. [11-01/D2] Tier published on GET /api/sources (per-instance) and GET /api/config/plugin-types (plugin_type_tiers, additive, no schema_version bump); a missing external directory is a legitimate empty tier
expected: Tier published on GET /api/sources (per-instance) and GET /api/config/plugin-types (plugin_type_tiers, additive, no schema_version bump); a missing external directory is a legitimate empty tier
result: pass
source: automated
coverage_id: 11-01/D2
verified_by: kernel/httpapi/sources_test.go; kernel/httpapi/config_test.go; cmd/topos/externaldir_test.go

### 4. [11-01/D3] TrustBadge overlay renders a CircleAlert glyph on an external-tier source's chip icon at the specified chip/picker scales; a trusted-tier chip's pill/markup/tooltip stay byte-identical to before this phase (D-06)
expected: TrustBadge overlay renders a CircleAlert glyph on an external-tier source's chip icon at the specified chip/picker scales; a trusted-tier chip's pill/markup/tooltip stay byte-identical to before this phase (D-06)
result: pass
source: automated
coverage_id: 11-01/D3
verified_by: web/src/lib/components/trust-badge.test.ts; web/e2e/specs/11-external-tier-badge.spec.ts

### 5. [11-01/D4] The complete Phase 11 TypeScript wire surface declared once in web/src/lib/api.ts (SourceStatus, SourceConfig, KernelConfig, PluginTypesResponse, DescribePluginResponse, ExtrasFieldDecl) so no later plan needs to re-edit
expected: The complete Phase 11 TypeScript wire surface declared once in web/src/lib/api.ts (SourceStatus, SourceConfig, KernelConfig, PluginTypesResponse, DescribePluginResponse, ExtrasFieldDecl) so no later plan needs to re-edit
result: pass
source: automated
coverage_id: 11-01/D4
verified_by: npm --prefix web run check; npm --prefix web run test

### 6. [11-01/D5] e2e fixture support for a second plugin directory (externalPluginBinaries, hashPluginBinary, plugins.external_dir/pins) so every later Phase 11 e2e spec can populate a two-tier fixture without touching the harness core a
expected: e2e fixture support for a second plugin directory (externalPluginBinaries, hashPluginBinary, plugins.external_dir/pins) so every later Phase 11 e2e spec can populate a two-tier fixture without touching the harness core a
result: pass
source: automated
coverage_id: 11-01/D5
verified_by: make e2e

### 7. [11-02/D1] Pre-exec SHA-256 pin verification: a tampered or unpinned external-tier binary is refused before any subprocess is created, named by instance/binary/pinned-vs-current hash; trusted-tier binaries are never pin-checked
expected: Pre-exec SHA-256 pin verification: a tampered or unpinned external-tier binary is refused before any subprocess is created, named by instance/binary/pinned-vs-current hash; trusted-tier binaries are never pin-checked
result: pass
source: automated
coverage_id: 11-02/D1
verified_by: kernel/pluginhost/binaryhash_test.go; kernel/config/config_test.go

### 8. [11-02/D2] A pin mismatch is a soft, per-instance failure: Discover/Reconcile record it and continue — every other configured source still boots/applies; every other launch-failure class (missing/broken binary) keeps its existing h
expected: A pin mismatch is a soft, per-instance failure: Discover/Reconcile record it and continue — every other configured source still boots/applies; every other launch-failure class (missing/broken binary) keeps its existing h
result: pass
source: automated
coverage_id: 11-02/D2
verified_by: kernel/supervisor/pinmismatch_test.go

### 9. [11-02/D3] A launched plugin subprocess receives ONLY a documented desktop-session allowlist plus the values behind ${VAR} references its own instance's raw config declares — never the kernel's remaining environment; enforced via g
expected: A launched plugin subprocess receives ONLY a documented desktop-session allowlist plus the values behind ${VAR} references its own instance's raw config declares — never the kernel's remaining environment; enforced via g
result: pass
source: automated
coverage_id: 11-02/D3
verified_by: kernel/pluginhost/env_test.go; kernel/supervisor/readiness_test.go; make e2e specs/09-search-clear-and-previewer.spec.ts

### 10. [11-02/D4] One shared ${VAR}/$VAR scanner (config.EnvRefNames) serves both GET /api/config's env_vars field and the plugin-launch env allowlist — kernel/httpapi/config.go no longer defines its own regex/reflection scanner
expected: One shared ${VAR}/$VAR scanner (config.EnvRefNames) serves both GET /api/config's env_vars field and the plugin-launch env allowlist — kernel/httpapi/config.go no longer defines its own regex/reflection scanner
result: pass
source: automated
coverage_id: 11-02/D4
verified_by: kernel/config/envrefs_test.go; kernel/httpapi

### 11. [11-02/D5] Per-instance extras (config.Source.Extras) reach the plugin as a nested extras object inside WEBSPACES_SOURCE_CONFIG, with kernel-known top-level keys unchanged and byte-identical; a config carrying extras+pins round-tri
expected: Per-instance extras (config.Source.Extras) reach the plugin as a nested extras object inside WEBSPACES_SOURCE_CONFIG, with kernel-known top-level keys unchanged and byte-identical; a config carrying extras+pins round-tri
result: pass
source: automated
coverage_id: 11-02/D5
verified_by: kernel/pluginhost/extras_test.go; kernel/config/config_test.go; kernel/config/writer_test.go; sdk/contract_test.go

### 12. [11-03/D1] A source refused launch on a pin mismatch still appears in GET /api/sources as a real, named entry — one entry per instance whether it launched or failed, sorted by instance id, with the probe result winning any name col
expected: A source refused launch on a pin mismatch still appears in GET /api/sources as a real, named entry — one entry per instance whether it launched or failed, sorted by instance id, with the probe result winning any name col
result: pass
source: automated
coverage_id: 11-03/D1
verified_by: kernel/httpapi/sources_test.go; kernel/supervisor/pinmismatch_test.go

### 13. [11-03/D2] launch_failure is a closed-vocabulary, machine-readable field (empty or pin_mismatch), never gated on parsing last_error text; a launch-failed source's pinned_hash and current_hash are both populated; a healthy external-
expected: launch_failure is a closed-vocabulary, machine-readable field (empty or pin_mismatch), never gated on parsing last_error text; a launch-failed source's pinned_hash and current_hash are both populated; a healthy external-
result: pass
source: automated
coverage_id: 11-03/D2
verified_by: kernel/httpapi/sources_test.go

### 14. [11-03/D3] The agent route's grant filter is unaffected by the launch-failure merge: an ungranted launch-failed source is structurally absent from GET /agent/v1/sources, and present with capabilities.read=true once granted
expected: The agent route's grant filter is unaffected by the launch-failure merge: an ungranted launch-failed source is structurally absent from GET /agent/v1/sources, and present with capabilities.read=true once granted
result: pass
source: automated
coverage_id: 11-03/D3
verified_by: kernel/httpapi/sources_test.go

### 15. [11-03/D4] POST /api/config/describe-plugin publishes tier, the kernel-computed binary_hash (external tier only, empty for trusted), env_var_names (names referenced in the submitted source including extras, never values) and the pl
expected: POST /api/config/describe-plugin publishes tier, the kernel-computed binary_hash (external tier only, empty for trusted), env_var_names (names referenced in the submitted source including extras, never values) and the pl
result: pass
source: automated
coverage_id: 11-03/D4
verified_by: kernel/httpapi/config_test.go; kernel/pluginhost/describe_test.go

### 16. [11-03/D5] Unknown plugin binary still refused 404 plugin_binary_not_found before anything executes, unaffected by the widened response shape
expected: Unknown plugin binary still refused 404 plugin_binary_not_found before anything executes, unaffected by the widened response shape
result: pass
source: automated
coverage_id: 11-03/D5
verified_by: kernel/httpapi/config_test.go

### 17. [11-03/D6] docs/plugin-contract.md, docs/api.md and config.example.toml republished to document trust tiers, pinning, the exact nine-variable launch-environment allowlist, the extras config shape/Describe declaration, and every Pha
expected: docs/plugin-contract.md, docs/api.md and config.example.toml republished to document trust tiers, pinning, the exact nine-variable launch-environment allowlist, the extras config shape/Describe declaration, and every Pha
result: pass
source: automated
coverage_id: 11-03/D6
verified_by: make docs-check

### 18. [11-04/D1] A plugin binary built from a module outside the in-repo plugin set (its own go.mod, its own module path, its own build target, its own output directory) is discovered from the external directory, launched under a content
expected: A plugin binary built from a module outside the in-repo plugin set (its own go.mod, its own module path, its own build target, its own output directory) is discovered from the external directory, launched under a content
result: pass
source: automated
coverage_id: 11-04/D1
verified_by: kernel/supervisor/externalproof_test.go

### 19. [11-04/D2] The proof binary receives provider-specific extras keys the kernel has never heard of and emits them back as items, with a ${VAR}-referenced value already expanded — the passthrough is observable, not asserted
expected: The proof binary receives provider-specific extras keys the kernel has never heard of and emits them back as items, with a ${VAR}-referenced value already expanded — the passthrough is observable, not asserted
result: pass
source: automated
coverage_id: 11-04/D2
verified_by: kernel/supervisor/externalproof_test.go

### 20. [11-04/D3] The proof binary emits the names of every environment variable it can actually see: PATH (allowlisted) and a ${VAR}-referenced variable are visible; a variable set on the kernel process but referenced nowhere in the inst
expected: The proof binary emits the names of every environment variable it can actually see: PATH (allowlisted) and a ${VAR}-referenced variable are visible; a variable set on the kernel process but referenced nowhere in the inst
result: pass
source: automated
coverage_id: 11-04/D3
verified_by: kernel/supervisor/externalproof_test.go

### 21. [11-04/D4] A tampered copy of the proof binary is refused at the next launch by name, and a healthy source in the same config still boots and stays reachable (PLUG-07 proven on a genuinely external binary, not only on a fixture)
expected: A tampered copy of the proof binary is refused at the next launch by name, and a healthy source in the same config still boots and stays reachable (PLUG-07 proven on a genuinely external binary, not only on a fixture)
result: pass
source: automated
coverage_id: 11-04/D4
verified_by: kernel/supervisor/externalproof_test.go

### 22. [11-04/D5] The proof binary lives under testdata/, never enters the in-repo plugin set's audit scope, and make test-portable/CGO_ENABLED=0 go build ./... from the repo root are unaffected by the new module
expected: The proof binary lives under testdata/, never enters the in-repo plugin set's audit scope, and make test-portable/CGO_ENABLED=0 go build ./... from the repo root are unaffected by the new module
result: pass
source: automated
coverage_id: 11-04/D5
verified_by: go test ./internal/audit/...; CGO_ENABLED=0 go build ./... && CGO_ENABLED=0 go test ./... from repo root

### 23. [11-05/D1] Adding a source from an external-tier plugin routes through a new untrusted-confirm interstitial (E1): binary name, full 64-hex-char kernel-computed hash, env-var disclosure (zero vs. one-or-many referenced vars), and a 
expected: Adding a source from an external-tier plugin routes through a new untrusted-confirm interstitial (E1): binary name, full 64-hex-char kernel-computed hash, env-var disclosure (zero vs. one-or-many referenced vars), and a 
result: pass
source: automated
coverage_id: 11-05/D1
verified_by: web/src/lib/components/untrusted-add.test.ts; web/e2e/specs/11-untrusted-add.spec.ts

### 24. [11-05/D2] Every picker row backed by an external-tier plugin (both the existing-instance group and the install-catalog group) carries the TrustBadge + an 'Untrusted' text label; a trusted-tier row is unchanged and no third, untrus
expected: Every picker row backed by an external-tier plugin (both the existing-instance group and the install-catalog group) carries the TrustBadge + an 'Untrusted' text label; a trusted-tier row is unchanged and no third, untrus
result: pass
source: automated
coverage_id: 11-05/D2
verified_by: web/src/lib/components/untrusted-add.test.ts; web/e2e/specs/11-untrusted-add.spec.ts

### 25. [11-05/D3] Save anyway is never offered for an external-tier plugin type — a connection-only save has no kernel-computed hash to pin, so it would create an unstartable, never-warned-about source
expected: Save anyway is never offered for an external-tier plugin type — a connection-only save has no kernel-computed hash to pin, so it would create an unstartable, never-warned-about source
result: pass
source: automated
coverage_id: 11-05/D3
verified_by: web/src/lib/components/untrusted-add.test.ts

### 26. [11-05/D4] ConnectionForm.svelte's E6 extras section — a plugin's Describe-declared expected extras keys render as labeled inputs whose declared default is placeholder-only, never pre-filled into the bound value (D-14); an always-v
expected: ConnectionForm.svelte's E6 extras section — a plugin's Describe-declared expected extras keys render as labeled inputs whose declared default is placeholder-only, never pre-filled into the bound value (D-14); an always-v
result: pass
source: automated
coverage_id: 11-05/D4
verified_by: web/src/lib/components/extras-form.test.ts; web/e2e/specs/11-untrusted-add.spec.ts

### 27. [11-05/D5] Arbitrary provider-specific extras keys, entered in the UI, round-trip through config.toml to a genuinely out-of-repo plugin process unmodified, and the pin written equals the exact kernel-computed hash the confirm dialo
expected: Arbitrary provider-specific extras keys, entered in the UI, round-trip through config.toml to a genuinely out-of-repo plugin process unmodified, and the pin written equals the exact kernel-computed hash the confirm dialo
result: pass
source: automated
coverage_id: 11-05/D5
verified_by: web/e2e/specs/11-untrusted-add.spec.ts

### 28. [11-06/D1] A source whose external binary no longer matches its pin renders a chip in the binary-changed state: a destructive health dot and a tooltip naming the specific cause, taking priority over the generic unreachable wording 
expected: A source whose external binary no longer matches its pin renders a chip in the binary-changed state: a destructive health dot and a tooltip naming the specific cause, taking priority over the generic unreachable wording 
result: pass
source: automated
coverage_id: 11-06/D1
verified_by: web/src/lib/components/repin.test.ts; web/e2e/specs/11-binary-changed-repin.spec.ts

### 29. [11-06/D2] The chip menu offers 'Trust updated binary…' as the FIRST item only when the mismatch signal is set (absent, not merely disabled, otherwise); Refresh now disables alongside it; a pinned-hash footer (short display form, f
expected: The chip menu offers 'Trust updated binary…' as the FIRST item only when the mismatch signal is set (absent, not merely disabled, otherwise); Refresh now disables alongside it; a pinned-hash footer (short display form, f
result: pass
source: automated
coverage_id: 11-06/D2
verified_by: web/src/lib/components/repin.test.ts; web/e2e/specs/11-binary-changed-repin.spec.ts

### 30. [11-06/D3] The re-pin dialog shows the previously pinned hash (short form, or 'not pinned' when absent) and the on-disk hash (full, break-all); confirming calls setPluginPin + putConfig exactly once each, keyed on the binary name (
expected: The re-pin dialog shows the previously pinned hash (short form, or 'not pinned' when absent) and the on-disk hash (full, break-all); confirming calls setPluginPin + putConfig exactly once each, keyed on the binary name (
result: pass
source: automated
coverage_id: 11-06/D3
verified_by: web/src/lib/components/repin.test.ts; web/e2e/specs/11-binary-changed-repin.spec.ts

### 31. [11-06/D4] Swapping an external binary is caught, named, visible on the affected chip only, and repairable from that chip's own menu — proven end to end in a real browser against the genuinely out-of-repo topos-plugin-external-demo
expected: Swapping an external binary is caught, named, visible on the affected chip only, and repairable from that chip's own menu — proven end to end in a real browser against the genuinely out-of-repo topos-plugin-external-demo
result: pass
source: automated
coverage_id: 11-06/D4
verified_by: web/e2e/specs/11-binary-changed-repin.spec.ts

### 32. [11-06/D5] A pin-mismatched instance participating in a webspace's match config (explicit block or keywords fallback) no longer blocks kernel BOOT — pluginhost.ValidateMatchConfig excuses a currently-launch-failed instance from its
expected: A pin-mismatched instance participating in a webspace's match config (explicit block or keywords fallback) no longer blocks kernel BOOT — pluginhost.ValidateMatchConfig excuses a currently-launch-failed instance from its
result: pass
source: automated
coverage_id: 11-06/D5
verified_by: kernel/pluginhost/matchconfig_test.go; kernel/supervisor/pinmismatch_test.go; Live reproduction outside the test suite: real kernel binary + real topos-plugin-external-demo, real config.toml, boot

### 33. [11-07/D1] pluginhost.ResolveBinary rejects any non-bare plugin binary name (traversal, absolute path, Windows separator, '.', '..', empty) as its first statement, before either configured directory is consulted, and never returns 
expected: pluginhost.ResolveBinary rejects any non-bare plugin binary name (traversal, absolute path, Windows separator, '.', '..', empty) as its first statement, before either configured directory is consulted, and never returns 
result: pass
source: automated
coverage_id: 11-07/D1
verified_by: kernel/pluginhost/tier_test.go

### 34. [11-07/D2] ResolveBinary requires info.Mode().IsRegular() at all three os.Stat sites, so a directory sharing a binary's name is never resolved, while a symlinked regular file (the e2e harness's fixture shape) still resolves unchang
expected: ResolveBinary requires info.Mode().IsRegular() at all three os.Stat sites, so a directory sharing a binary's name is never resolved, while a symlinked regular file (the e2e harness's fixture shape) still resolves unchang
result: pass
source: automated
coverage_id: 11-07/D2
verified_by: kernel/pluginhost/tier_test.go; web/e2e/specs/11-external-tier-badge.spec.ts

### 35. [11-07/D3] config.Validate rejects an absent, whitespace-only, or non-bare Source.Plugin value at config load, naming the offending source and (for an unset ${VAR}) the missing variable, with deterministic multi-error reporting.
expected: config.Validate rejects an absent, whitespace-only, or non-bare Source.Plugin value at config load, naming the offending source and (for an unset ${VAR}) the missing variable, with deterministic multi-error reporting.
result: pass
source: automated
coverage_id: 11-07/D3
verified_by: kernel/config/config_test.go

### 36. [11-07/D4] PUT /api/config rejects a traversal or empty Source.Plugin value with 422 config_invalid, naming the source, leaving config.toml byte-identical to before the request.
expected: PUT /api/config rejects a traversal or empty Source.Plugin value with 422 config_invalid, naming the source, leaving config.toml byte-identical to before the request.
result: pass
source: automated
coverage_id: 11-07/D4
verified_by: kernel/httpapi/config_test.go

### 37. [11-07/D5] The newly-enforced bare-filename rule for [sources.<id>] plugin is published in docs/plugin-contract.md's Trust tiers section.
expected: The newly-enforced bare-filename rule for [sources.<id>] plugin is published in docs/plugin-contract.md's Trust tiers section.
result: pass
source: automated
coverage_id: 11-07/D5
verified_by: grep -n 'bare binary filename' docs/plugin-contract.md

## Summary

total: 37
passed: 37
issues: 0
pending: 0
skipped: 0
blocked: 0

## Gaps

[none yet]
