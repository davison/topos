---
phase: 11-external-plugins-the-trust-boundary
reviewed: 2026-08-13T00:00:00Z
depth: standard
files_reviewed: 76
files_reviewed_list:
  - cmd/topos/externaldir_test.go
  - cmd/topos/main.go
  - config.example.toml
  - docs/api.md
  - docs/plugin-contract.md
  - docs/testing.md
  - go.work
  - go.work.sum
  - kernel/config/config.go
  - kernel/config/config_test.go
  - kernel/config/envrefs.go
  - kernel/config/envrefs_test.go
  - kernel/config/types.go
  - kernel/config/writer_test.go
  - kernel/httpapi/agent_live_config_test.go
  - kernel/httpapi/agent_test.go
  - kernel/httpapi/config.go
  - kernel/httpapi/config_test.go
  - kernel/httpapi/contract_test.go
  - kernel/httpapi/live_config_test.go
  - kernel/httpapi/routes.go
  - kernel/httpapi/sources.go
  - kernel/httpapi/sources_test.go
  - kernel/pluginhost/binaryhash.go
  - kernel/pluginhost/binaryhash_test.go
  - kernel/pluginhost/describe_test.go
  - kernel/pluginhost/describe_whatsapp_test.go
  - kernel/pluginhost/discover_binaries.go
  - kernel/pluginhost/discover_binaries_test.go
  - kernel/pluginhost/env_test.go
  - kernel/pluginhost/extras_test.go
  - kernel/pluginhost/host.go
  - kernel/pluginhost/matchconfig.go
  - kernel/pluginhost/matchconfig_test.go
  - kernel/pluginhost/pin_test.go
  - kernel/pluginhost/reconcile_test.go
  - kernel/pluginhost/stderr_test.go
  - kernel/pluginhost/tier_test.go
  - kernel/supervisor/externalproof_test.go
  - kernel/supervisor/externaltier_test.go
  - kernel/supervisor/launchlatency_test.go
  - kernel/supervisor/pinmismatch_test.go
  - kernel/supervisor/readiness_test.go
  - kernel/supervisor/supervisor.go
  - kernel/supervisor/supervisor_test.go
  - kernel/supervisor/suspend_test.go
  - Makefile
  - proto/topos/v1/plugin.proto
  - sdk/contract_test.go
  - sdk/gen/topos/v1/plugin.pb.go
  - testdata/external-plugin/go.mod
  - testdata/external-plugin/go.sum
  - testdata/external-plugin/main.go
  - testdata/external-plugin/plugin.go
  - testdata/external-plugin/README.md
  - web/e2e/e2e-builtins.d.ts
  - web/e2e/fixtures/config-builder.ts
  - web/e2e/fixtures/kernel.ts
  - web/e2e/fixtures/plugin-binaries.ts
  - web/e2e/specs/09-search-clear-and-previewer.spec.ts
  - web/e2e/specs/11-binary-changed-repin.spec.ts
  - web/e2e/specs/11-external-tier-badge.spec.ts
  - web/e2e/specs/11-untrusted-add.spec.ts
  - web/e2e/specs/uat-05-two-step-connect.spec.ts
  - web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts
  - web/src/lib/api.ts
  - web/src/lib/components/AddSourceModal.svelte
  - web/src/lib/components/add-source.test.ts
  - web/src/lib/components/chip-edit-menu.test.ts
  - web/src/lib/components/ConnectionForm.svelte
  - web/src/lib/components/extras-form.test.ts
  - web/src/lib/components/EditSourceModal.svelte
  - web/src/lib/components/ManageSourcesModal.svelte
  - web/src/lib/components/relink.test.ts
  - web/src/lib/components/repin.test.ts
  - web/src/lib/components/SourceChip.svelte
  - web/src/lib/components/sources.test.ts
  - web/src/lib/components/staleness.test.ts
  - web/src/lib/components/TrustBadge.svelte
  - web/src/lib/components/trust-badge.test.ts
  - web/src/lib/components/TrustUpdateDialog.svelte
  - web/src/lib/components/untrusted-add.test.ts
  - web/src/lib/components/WebspaceHeader.svelte
  - web/src/lib/config-edit.ts
  - web/src/lib/format.test.ts
  - web/src/lib/format.ts
  - web/src/lib/plugin-fields.ts
  - web/src/routes/w/[webspace]/+page.svelte
findings:
  critical: 0
  warning: 2
  info: 1
  total: 3
status: issues_found
---

# Phase 11: Code Review Report (re-review after gap closure)

**Reviewed:** 2026-08-13T00:00:00Z
**Depth:** standard
**Files Reviewed:** 76
**Status:** issues_found

## Summary

This is a re-review of Phase 11 (external plugins / trust boundary) after
plan 11-07 closed the prior review's CR-01 finding (`pluginhost.ResolveBinary`
path traversal / tier misclassification via a caller-supplied `Source.Plugin`).

**CR-01 verification: the fix is sound and complete.** I traced the full
confinement chain rather than trusting the plan's own claim:

- `kernel/pluginhost/discover_binaries.go`'s `validatePluginBinaryName` is the
  first statement of `ResolveBinary` — it rejects an empty/whitespace name, any
  name containing `/` or `\`, `.`/`..` exactly, and any name where
  `name != filepath.Base(name)`, before either configured directory is stat'd.
  `tier_test.go` proves this against real traversal payloads (`"../"+dir+"/topos-plugin-evil"`,
  an absolute path, a Windows-separator payload, `""`, `"."`, `".."`) with a file
  genuinely present at the escape target, so the tests prove the rejection is
  real rather than merely "the target happened not to exist."
- `kernel/config/config.go`'s new `validateSourcePlugins` is a byte-for-byte
  hand-kept twin of the same four rules, invoked from `Config.Validate` — which
  runs both at file load (`LoadRaw`) and at every `PUT /api/config` save
  (`Store.Save` → `dryRunExpand` → `Validate`) — so a malformed `plugin` value is
  rejected at config-write time, not only at next-launch time. `config_test.go`
  pins this with deterministic multi-error-ordering and empty/whitespace/unset-env
  cases.
- `kernel/httpapi/config.go`'s `DescribePluginHandler` and (outside this file
  list) `whatsapplink.go`'s start handler both independently gate on a directory
  *listing* (`DiscoverAllTiered`/`DiscoverAllBinaries`) before ever calling into
  `ResolveBinary`/`exec.Command` — defense in depth, not a single point of
  failure.

I found no way to reintroduce the original defect (empty name resolving to the
directory itself, `../` escaping a configured directory, a Windows-style
separator surviving on Linux, or a directory masquerading as a regular file)
through any call site reachable from this phase's files. No new Critical or
security-relevant findings in this pass.

Two Warnings and one Info item remain, none of which touch the fixed
confinement logic itself — see below.

## Warnings

### WR-01: config.example.toml's per-source `plugin` field docs are stale after the CR-01 fix

**File:** `config.example.toml:177,237,289,431,461`
**Issue:** Every `[sources.<name>]` block's `plugin =` comment still reads:

```
# Validation: none at load time; a missing file fails at startup, by path.
```

This was accurate before plan 11-07, but is no longer true: `kernel/config/config.go`'s
`validateSourcePlugins` (invoked from `Config.Validate`, which runs on every
`config.Load`/`LoadRaw` and every `PUT /api/config` save) now rejects a `plugin`
value at **load/save time** whenever it is empty, contains a path separator,
is `.`/`..`, or otherwise fails `filepath.Base` equality — well before any
binary-resolution attempt at launch. An operator reading this comment (or a
plugin author copying its convention for a sixth source block) is told the
`plugin` key is unchecked until process start, which is now wrong in a
security-relevant way: it undersells the load-time confinement guard this
phase's gap-closure plan just added. `docs/api.md` and `docs/plugin-contract.md`
have the identical gap — neither documents the "bare filename only, no path
separators or `..` segments" rule anywhere.
**Fix:** Update the five occurrences in `config.example.toml` to something like:

```
# Validation: must be a bare binary filename — no "/", "\", "." or ".."
# segments (fails config load, naming the offending source instance); a
# name that is otherwise well-formed but doesn't exist on disk still fails
# at startup, by path.
```

and add an equivalent one-paragraph note to `docs/plugin-contract.md`'s
"Configuration" section (or the "Trust tiers" section, where the confinement
rule's rationale already partially lives) so the constraint is documented
once outside test/code comments.

### WR-02: `TrustUpdateDialog.svelte` can write an empty-string pin if `current_hash` is ever absent

**File:** `web/src/lib/components/TrustUpdateDialog.svelte:74`
**Issue:**

```ts
const nextConfig = setPluginPin(config, source.plugin, source.current_hash ?? '');
```

This dialog is only ever opened for a `launch_failure === 'pin_mismatch'`
entry (`SourceChip.svelte`'s `isPinMismatch` gate), and
`kernel/pluginhost/host.go`'s `launch()` always populates `binaryHash` (and
therefore `LaunchFailure.CurrentHash`) for every `TierExternal` launch attempt
before a pin check can even run — so in practice `current_hash` should never
be empty here. But the `?? ''` fallback means that if this invariant is ever
violated (a future refactor, a partially-populated fixture in a test, or a
kernel response shape that omits the field), the dialog silently submits
`[plugins.pins].<binary> = ""` rather than surfacing an error to the operator.
The kernel-side `validatePins`/`pinHashPattern` guard (`kernel/config/config.go`)
would then reject the save with a generic 64-hex-char message that gives the
operator no clue this UI-side fallback is what produced the malformed value.
**Fix:** Guard the write instead of silently coercing to an empty string:

```ts
async function confirmTrustUpdate() {
	if (saving) return;
	if (!source.current_hash) {
		error = 'No current hash available for this binary — reload and try again.';
		return;
	}
	saving = true;
	...
	const nextConfig = setPluginPin(config, source.plugin, source.current_hash);
	...
}
```

## Info

### IN-01: `Tier` value for a rejected `ResolveBinary` name relies on the zero value rather than a named constant

**File:** `kernel/pluginhost/discover_binaries.go:368-371`
**Issue:** `ResolveBinary`'s first `return` on a `validatePluginBinaryName`
failure returns `""` for `tier` (relying on `Tier`'s zero value being the
empty string), rather than a named sentinel such as `TierNone`. `TierTrusted`
and `TierExternal` are the only two exported `Tier` constants, so a future
reader adding a third tier (or comparing `tier == ""` somewhere) has no
compiler-checked signal that this return path exists. This is purely a
readability/maintainability nit — every current call site correctly checks
`err != nil` first and never inspects `tier` on the error path (confirmed via
`tier_test.go`'s own assertions), so this is not a functional defect today.
**Fix:** Optional: introduce `const TierNone Tier = ""` with a doc comment
explaining it is returned only alongside a non-nil error, and return it
explicitly at each of `ResolveBinary`'s three failure returns, for
self-documentation rather than an implicit zero value.

---

_Reviewed: 2026-08-13T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
