---
phase: 11-external-plugins-the-trust-boundary
reviewed: 2026-08-13T00:00:00Z
depth: standard
files_reviewed: 60
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
  critical: 1
  warning: 3
  info: 2
  total: 6
status: issues_found
---

# Phase 11: Code Review Report

**Reviewed:** 2026-08-13
**Depth:** standard
**Files Reviewed:** 60 (subset of the 88 files listed — every non-test/non-generated Go source file in the trust-boundary path was read in full; svelte/ts sources touching the trust UI were read in full; most `_test.go`/`*.test.ts`/`*.spec.ts` files were sampled rather than read exhaustively, per the depth budget)
**Status:** issues_found

## Summary

This phase's stated invariant is that plugin trust tier ("trusted" vs "external") is derived **exclusively** from *which configured directory a binary's bytes were found in* — never from the plugin's own claims, and never overridable by config. The implementation of the hash-pinning re-verification (`kernel/pluginhost/host.go`'s `launch`/`hashAndLookupPin`), the soft per-instance launch-failure bookkeeping (`LaunchFailure`, `Host.Reconcile`), the environment allowlist (`allowedEnv`/`SkipHostEnv`), and the UI trust-disclosure flow (`AddSourceModal.svelte`'s untrusted-confirm interstitial, `TrustUpdateDialog.svelte`'s re-pin flow) are all careful, well-tested, and internally consistent with that stated design — the pin-mismatch soft-failure path in particular is exercised end-to-end against a real out-of-repo binary in both a Go test (`kernel/supervisor/externalproof_test.go`) and a browser e2e spec (`web/e2e/specs/11-binary-changed-repin.spec.ts`).

However, the one function this whole tier system is actually *built on* — `pluginhost.ResolveBinary`, the launch-time authority that turns a `[sources.<id>].plugin` string into a filesystem path and a tier — performs no validation of that string at all before handing it to `filepath.Join` and `exec.Command`. `config.Validate` (`kernel/config/config.go`) likewise never validates the shape of `Source.Plugin`. The result is a directory-traversal / trust-tier-bypass bug that lets a value written through `PUT /api/config` — a request the SPA itself can never construct maliciously, but which is not itself authenticated or origin-restricted beyond loopback — resolve to an arbitrary file on disk, misclassified as `TierTrusted` (skipping the pin check the whole phase exists to enforce) whenever that path happens to exist under the trusted directory's join. This is the review's one BLOCKER; everything else found is a warning-tier robustness/defense-in-depth gap in the same area, plus two minor info-level notes.

## Critical Issues

### CR-01: `ResolveBinary`/`config.Validate` do not confine or validate `Source.Plugin`, allowing directory traversal that defeats the trust-tier boundary

**File:** `kernel/pluginhost/discover_binaries.go:317-343` (`ResolveBinary`), `kernel/pluginhost/host.go:785` (`launch`'s call site), `kernel/config/config.go:327-377` (`Validate`)

**Issue:**

`ResolveBinary` is documented as "the one launch-time authority for turning a plugin binary NAME into a filesystem PATH plus the Tier it resolved to" — the single point this entire phase's trust model rests on ("Tier is derived exclusively from which directory a binary resolved from... never from anything the plugin itself declares", `docs/plugin-contract.md`'s "Trust tiers" section). Its implementation is:

```go
func ResolveBinary(dirs Dirs, name string, logger hclog.Logger) (path string, tier Tier, err error) {
	if dirs.Trusted != "" {
		trustedPath := filepath.Join(dirs.Trusted, name)
		if _, statErr := os.Stat(trustedPath); statErr == nil {
			...
			return trustedPath, TierTrusted, nil
		}
	}
	if dirs.External != "" {
		externalPath := filepath.Join(dirs.External, name)
		if _, statErr := os.Stat(externalPath); statErr == nil {
			return externalPath, TierExternal, nil
		}
	}
	return "", "", fmt.Errorf(...)
}
```

`name` is `src.Plugin` — `config.Source.Plugin` — which is never validated anywhere in the load or save path. `kernel/config/config.go`'s `Validate` checks `[plugins.pins]` KEYS for the `topos-plugin-` prefix (`validatePins`), and checks `base_url`/`token`/`path` presence, but never checks `Source.Plugin` itself: it may be empty, may contain `/` or `..` path segments, and none of that is rejected at config-load time, at `config.Store.Save`'s dry-run validation, or anywhere in `ResolveBinary` itself.

Two independent gaps compound here:

1. **No confinement to the configured directory.** `filepath.Join(dirs.Trusted, name)` happily resolves `name = "../../../../home/user/.cache/some-binary"` (or an absolute-looking traversal) to a path *outside* `dirs.Trusted` entirely. If that resolved path exists and `os.Stat` succeeds, `ResolveBinary` returns it tagged **`TierTrusted`** — because the check is "did the trusted-directory join resolve to something that exists," not "does the resolved path still live inside `dirs.Trusted`." This means an operator-or-script-supplied `plugin` value with `../` segments can point at *any file on the filesystem* and have the kernel treat it as a trusted-tier binary, skipping pin verification entirely (`launch`'s pin-check block in `host.go` is gated on `tier == TierExternal` — a misclassified `TierTrusted` result never reaches it) and executing it via `exec.Command(binPath)` with no hash check at all. This is the exact bypass Phase 11 was built to prevent: an untrusted binary running with zero content-addressing.
2. **No regular-file check.** Unlike `DiscoverAllBinaries`/`isRegularFileFollowingSymlinks` (which explicitly checks `IsRegular()` after following one symlink hop), `ResolveBinary` only checks that `os.Stat` returns no error — it accepts a directory, device file, or anything else `os.Stat` can see. An empty `Source.Plugin` (never rejected by `Validate`) makes `filepath.Join(dirs.Trusted, "")` equal `dirs.Trusted` itself; `os.Stat(dirs.Trusted)` succeeds (it's a directory), so `ResolveBinary` returns `dirs.Trusted` tagged `TierTrusted` with a nil error — the failure only surfaces later, confusingly, when `exec.Command` tries to execute a directory.

**Reachability:** `POST /api/config/describe-plugin` (the only HTTP path that lets a caller *choose* a plugin binary name from a request body) is properly guarded — `DescribePluginHandler` checks `req.Plugin` against `pluginhost.DiscoverAllTiered`'s real, on-disk result set before ever calling `DescribePluginType`/`launch` (`kernel/httpapi/config.go:335-356`), and overwrites `req.Source.Plugin` with the validated value. That path is safe.

The bypass is reachable through `PUT /api/config` (`kernel/httpapi/config.go`'s `ConfigSaveHandler` → `config.Store.Save` → `dryRunExpand` → `Config.Validate`), which accepts and persists an **arbitrary** `sources.<id>.plugin` string with no membership check against any discovered binary and no shape check at all. A save immediately triggers `Supervisor.Apply` → `Host.Reconcile` → `launch` → `ResolveBinary` on the same request (`ConfigSaveHandler`'s own doc comment: "calls applier.Apply to reconcile the running kernel... in the same request"), so the traversal is exercised on the very save that introduces it — no restart, no second window. Nothing in the SPA's own code constructs such a value (`ConnectionForm.svelte` never exposes an editable `plugin` field — ConnectionForm's field table has no `plugin` entry, so `EditSourceModal`/`AddSourceModal` can never *change* an existing instance's plugin, and new instances only ever get a plugin value copied from a validated `pluginTypes` list) — but the HTTP surface itself performs no defense-in-depth check, so any other caller of this loopback API (a compromised browser extension, a bug in a future admin tool, a hand-typed `curl`/DevTools console command) can silently defeat the entire tiering and pinning mechanism this phase implements.

**Fix:** Add both checks:

```go
// In kernel/config/config.go's Validate (or a new validateSources check):
var pluginNamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
...
if !pluginNamePattern.MatchString(src.Plugin) {
    return fmt.Errorf("config: source %q has invalid plugin name %q — must be a bare filename with no path separators", name, src.Plugin)
}
```

and, as the load-bearing fix in `ResolveBinary` itself (defense-in-depth belongs at the authority function, not only at the config-validation layer):

```go
func ResolveBinary(dirs Dirs, name string, logger hclog.Logger) (path string, tier Tier, err error) {
	if name == "" || name != filepath.Base(name) || strings.Contains(name, "..") {
		return "", "", fmt.Errorf("pluginhost: invalid plugin binary name %q", name)
	}
	if dirs.Trusted != "" {
		trustedPath := filepath.Join(dirs.Trusted, name)
		if info, statErr := os.Stat(trustedPath); statErr == nil && info.Mode().IsRegular() {
			...
		}
	}
	...
}
```

(mirroring `isRegularFileFollowingSymlinks`'s existing `IsRegular()` discipline so `ResolveBinary` and `DiscoverAllBinaries` apply the identical file-type check).

## Warnings

### WR-01: `config.Validate` never requires `Source.Plugin` to be non-empty

**File:** `kernel/config/config.go:327-377` (`Validate`)

**Issue:** Every other required connection shape (`base_url`+`token`, or `path`) is validated at load time with a named, actionable error. `Source.Plugin` — the field that determines which binary is even launched — has no presence check at all. An operator's typo (a missing `plugin = "..."` line under `[sources.<id>]`) loads successfully and only fails later, confusingly, at launch (see CR-01's directory-stat-succeeds-on-empty-string case).

**Fix:** Add an explicit check inside the existing `for name, src := range cfg.Sources` loop in `Validate`:
```go
if strings.TrimSpace(src.Plugin) == "" {
    return fmt.Errorf("config: source %q has no plugin specified", name)
}
```

### WR-02: `AddSourceModal.svelte`'s untrusted "type-to-confirm" gate is enforced only client-side, with no corroborating server-side acknowledgement recorded

**File:** `web/src/lib/components/AddSourceModal.svelte:823-884` (`step === 'untrusted-confirm'`), `kernel/httpapi/config.go` (`ConfigSaveHandler`)

**Issue:** `confirmUntrusted()` is "a pure step transition... issues no network call of its own" — the operator's explicit acknowledgement (typing the binary name to confirm) is never itself sent to, or recorded by, the kernel; only the resulting pin write is. This is consistent with the phase's own stated stance (no sandboxing, no server-side enforcement beyond content-addressing — see `docs/plugin-contract.md`'s repeated "this is disclosure, not containment" framing), so this is not a design defect, but it does mean the confirm-typed gate is pure UX friction with no security value: any caller of `PUT /api/config` (see CR-01) can write a `[plugins.pins]` entry and a new external-tier `[sources.<id>]` block directly, bypassing the interstitial entirely, with the kernel never distinguishing "operator clicked through the warning" from "some other caller wrote the same bytes." Given CR-01 already demonstrates this API has no additional caller-intent verification, this is worth a second look if the project ever wants the "explicit trust decision" this dialog performs to be more than cosmetic.

**Fix:** Not required for this phase's stated scope (no sandboxing/authentication goal), but worth a follow-up decision record if a future milestone adds any authentication/CSRF protection to `/api/config` — at that point, consider whether the untrusted-confirm acknowledgement should be captured server-side (e.g. a per-save "acknowledged_untrusted: true" flag threaded through the same request) rather than trusted purely via UI flow control.

### WR-03: `validatePins` accepts a pins entry for a binary name that is never checked to actually exist or to be external-tier

**File:** `kernel/config/config.go:397-423` (`validatePins`)

**Issue:** `validatePins` checks only that a `[plugins.pins]` key looks like `topos-plugin-*` and that its value is a 64-hex-char string — by design, since "config has no way to know which binaries actually exist on disk" at config-load time (the comment is explicit about this). This is a reasonable load-time constraint, but it means a config can accumulate arbitrary stale/fictitious pin entries for binaries that were never external-tier (e.g. a pin entry for a *trusted*-tier binary name, which `launch`'s pin-check block would simply never consult since it's gated on `tier == TierExternal`) with no cleanup path and no warning surfaced anywhere. Not exploitable on its own, but it is dead config an operator has no way to discover is inert.

**Fix:** Low priority — consider a `GET /api/config`-time or `GET /api/sources`-time diagnostic (not a load-time hard failure, since a pin can legitimately predate its binary being copied in) that flags a `[plugins.pins]` key with no corresponding currently-discovered binary of either tier.

## Info

### IN-01: `pinMismatchError`/`HashBinary` error text embeds full filesystem paths that reach the HTTP response body

**File:** `kernel/pluginhost/binaryhash.go:26-38`, `kernel/pluginhost/host.go:829-836` (`hashAndLookupPin` error path), `kernel/httpapi/sources.go:206` (`LastError: f.Message`)

**Issue:** `HashBinary`'s wrapped error (`"pluginhost: hash binary %s: %w"`, using the resolved absolute path) can reach `GET /api/sources`'s `last_error` field via `LaunchFailure.Message` if hashing itself fails (e.g. a permissions error mid-file-read on a since-removed external directory). This is low severity — it's the operator's own machine and own paths, and `GET /api/sources` is already loopback-only — but it is a minor departure from the "never log a credential... log the *presence* or *name*" discipline the plugin contract documents for plugin authors; the kernel's own error paths are a little looser about what they surface to the HTTP boundary. Not actionable as a fix requirement, noted for awareness only.

### IN-02: `ManageSourcesModal.svelte`'s `handleEditSaved` best-effort refresh silently swallows a `getConfig()` failure

**File:** `web/src/lib/components/ManageSourcesModal.svelte:178-190`

**Issue:** The comment explicitly documents this as intentional ("Best-effort refresh only... `onchanged()` below still fires the parent's own `getConfig()`, which is the authoritative fallback"), so this is not a defect — noted only because the pattern (an empty `catch {}` with no logging) is otherwise a common code-smell signature; here it is a deliberate, documented, and safe degrade with a real fallback path, not an omission.

---

_Reviewed: 2026-08-13_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
