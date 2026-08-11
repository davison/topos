---
phase: 09-ui-polish-and-source-management-rework
reviewed: 2026-08-11T00:00:00Z
depth: standard
files_reviewed: 84
files_reviewed_list:
  - cmd/topos/main.go
  - config.example.toml
  - docs/api.md
  - docs/plugin-contract.md
  - docs/testing.md
  - internal/audit/plugin_icons_test.go
  - internal/audit/testdata/plugin_icons_violation/assets/icon.svg
  - internal/audit/testdata/plugin_icons_violation/plugin.go
  - kernel/httpapi/agent_live_config_test.go
  - kernel/httpapi/agent_test.go
  - kernel/httpapi/contract_test.go
  - kernel/httpapi/live_config_test.go
  - kernel/httpapi/pluginicon.go
  - kernel/httpapi/pluginicon_test.go
  - kernel/httpapi/routes.go
  - kernel/httpapi/sources.go
  - kernel/pluginhost/host.go
  - kernel/pluginhost/icon_test.go
  - kernel/supervisor/supervisor.go
  - plugins/mock/assets/icon.svg
  - plugins/mock/main.go
  - plugins/mock/plugin.go
  - plugins/mock/renditionfixture.go
  - plugins/mock/renditionfixture_test.go
  - plugins/mockstrict/assets/icon.svg
  - plugins/mockstrict/plugin.go
  - plugins/paperless/assets/icon.svg
  - plugins/paperless/plugin.go
  - plugins/proton/assets/icon.svg
  - plugins/proton/plugin.go
  - plugins/signal/assets/icon.svg
  - plugins/signal/plugin.go
  - plugins/silverbullet/assets/icon.svg
  - plugins/silverbullet/plugin.go
  - plugins/whatsapp/assets/icon.svg
  - plugins/whatsapp/describeonly.go
  - plugins/whatsapp/plugin.go
  - proto/topos/v1/plugin.proto
  - README.md
  - sdk/contract_test.go
  - sdk/gen/topos/v1/plugin.pb.go
  - web/e2e/fixtures/config-builder.ts
  - web/e2e/fixtures/kernel.ts
  - web/e2e/specs/09-chip-menu.spec.ts
  - web/e2e/specs/09-picker-groups.spec.ts
  - web/e2e/specs/09-plugin-icon.spec.ts
  - web/e2e/specs/09-search-clear-and-previewer.spec.ts
  - web/e2e/specs/09-static-assets-and-surfaces.spec.ts
  - web/e2e/specs/09-stream-row-plugin-icon.spec.ts
  - web/e2e/specs/09-switcher-reload.spec.ts
  - web/e2e/specs/mockstrict-discovery.spec.ts
  - web/e2e/specs/uat-01-create-webspace.spec.ts
  - web/e2e/specs/uat-02-remove-source-items.spec.ts
  - web/e2e/specs/uat-03-empty-shell-first-source.spec.ts
  - web/e2e/specs/uat-05-two-step-connect.spec.ts
  - web/e2e/specs/uat-06-remove-chip-roundtrip.spec.ts
  - web/e2e/specs/uat-08-whatsapp-qr-link.spec.ts
  - web/e2e/specs/uat-10-describe-race.spec.ts
  - web/src/app.css
  - web/src/lib/api.ts
  - web/src/lib/components/AddSourceModal.svelte
  - web/src/lib/components/add-source.test.ts
  - web/src/lib/components/chip-edit-menu.test.ts
  - web/src/lib/components/detail-media-box.test.ts
  - web/src/lib/components/DetailPane.svelte
  - web/src/lib/components/ManageSourcesModal.svelte
  - web/src/lib/components/manage-sources.test.ts
  - web/src/lib/components/PluginIcon.svelte
  - web/src/lib/components/plugin-icon.test.ts
  - web/src/lib/components/QRPanel.svelte
  - web/src/lib/components/qr-panel.test.ts
  - web/src/lib/components/save-state.test.ts
  - web/src/lib/components/search-box-clear.test.ts
  - web/src/lib/components/SearchBox.svelte
  - web/src/lib/components/SearchResults.svelte
  - web/src/lib/components/source-chip-pill.test.ts
  - web/src/lib/components/source-chip-selected.test.ts
  - web/src/lib/components/SourceChip.svelte
  - web/src/lib/components/source-chip-tooltip.test.ts
  - web/src/lib/components/sources.test.ts
  - web/src/lib/components/staleness.test.ts
  - web/src/lib/components/stream-row-plugin-icon.test.ts
  - web/src/lib/components/StreamList.svelte
  - web/src/lib/components/StreamRow.svelte
  - web/src/lib/components/ui/dropdown-menu/dropdown-menu-content.svelte
  - web/src/lib/components/ui/popover/popover-content.svelte
  - web/src/lib/components/WebspaceHeader.svelte
  - web/src/lib/components/WebspaceSwitcher.svelte
  - web/src/lib/components/webspace-switcher.test.ts
  - web/src/lib/format.test.ts
  - web/src/lib/popover-surface.test.ts
  - web/src/routes/+layout.svelte
  - web/src/routes/w/[webspace]/+page.svelte
  - web/static/robots.txt
findings:
  critical: 0
  warning: 2
  info: 1
  total: 3
status: issues_found
---

# Phase 09: Code Review Report

**Reviewed:** 2026-08-11T00:00:00Z
**Depth:** standard
**Files Reviewed:** 84 (of the 96 supplied; the remaining files are binary
assets excluded by the workflow, or generic reads of unmodified
supporting files consulted for cross-reference only)
**Status:** issues_found

## Summary

This phase adds the plugin-provided-icon contract (proto field, kernel
capture/validation/serving, seven embedded per-plugin SVG assets, and the
frontend `PluginIcon.svelte` component with its mandatory fallback chain)
plus a batch of UI defect fixes (search-box clear button centring,
detail-pane media-box aspect lock, popover/dropdown surface tokens,
relocated Reload-config action, folded chip refresh-menu, favicon/
robots.txt). I read every file in the supplied list, cross-referenced the
`git diff` against `8403c456` for every changed file, ran `go build`,
`go vet`, `go test ./kernel/... ./internal/... ./sdk/...`, and the
frontend `vitest`/`svelte-check` suites — all green.

The implementation is unusually well-documented and the icon-serving path
in particular (`kernel/httpapi/pluginicon.go`, `kernel/pluginhost/host.go`)
correctly handles the path-traversal, size-ceiling, MIME-allowlist and
CSP/nosniff hardening it claims to. I could not substantiate any
crash/security/data-loss-class defect. Two real defects survived the
review: an off-by-one boundary mismatch between the new icon-audit test
and the kernel behavior it claims to mirror, and a UX regression where
folding the source chip's refresh control into an overflow menu quietly
removed the chip's only at-rest (non-hover) indicator that a sync is in
progress — while a nearby comment still asserts that indicator exists.

## Warnings

### WR-01: `internal/audit/plugin_icons_test.go`'s icon-size check is off by one against the kernel boundary it claims to mirror

**File:** `internal/audit/plugin_icons_test.go:19,51-56`
**Issue:** The audit's `maxIconBytes` check uses `>=`:

```go
const maxIconBytes = 65536
...
if len(iconBytes) >= maxIconBytes {
    offenses = append(offenses, fmt.Sprintf(
        "%s: assets/icon.svg is %d bytes, must be under %d (kernel/pluginhost.MaxIconBytes)",
        name, len(iconBytes), maxIconBytes,
    ))
}
```

This rejects an icon of exactly 65536 bytes. But the kernel enforcement
this test's own doc comment claims to mirror
(`kernel/pluginhost/host.go:69`, `MaxIconBytes = 65536`) uses strict `>`:

```go
if len(b) > MaxIconBytes {
    return nil, "", false
}
```

…which *accepts* an icon of exactly `MaxIconBytes` bytes — proven
explicitly by `kernel/pluginhost/icon_test.go`'s
`TestCaptureIcon_AtMaxIconBytesIsKept` ("the boundary is inclusive —
exactly `MaxIconBytes` is accepted, only `MaxIconBytes+1` is dropped").

So a `plugins/<name>/assets/icon.svg` of exactly 65536 bytes is a file
the kernel would happily capture and serve, but this audit's mechanical
gate (`TestPluginIcons_EveryInRepoPluginDeclaresOne`, wired into CI as a
hard "no violations" fail) would reject as an "offense" with the
misleading message "must be under 65536" — the opposite boundary of what
the kernel actually enforces. No currently-committed icon asset hits this
boundary today (the largest, `plugins/silverbullet/assets/icon.svg`, is
~12.7KB), so this is currently dormant, but it will incorrectly block a
future plugin author who ships a legally-sized (per the kernel's own
contract) 65536-byte icon.
**Fix:**
```go
if len(iconBytes) > maxIconBytes {
    offenses = append(offenses, fmt.Sprintf(
        "%s: assets/icon.svg is %d bytes, must be at most %d (kernel/pluginhost.MaxIconBytes)",
        name, len(iconBytes), maxIconBytes,
    ))
}
```
Also update the doc comment at line 181 ("under `maxIconBytes`") to "at
most `maxIconBytes`" for consistency.

### WR-02: Folding the chip's refresh control into the overflow menu removed the only at-rest sync indicator, and a nearby comment no longer matches the code

**File:** `web/src/lib/components/SourceChip.svelte:104-105` (comment),
`:202-210` (current behavior)
**Issue:** Before 09-05-PLAN.md's Fix 5, `SourceChip.svelte` rendered a
standalone refresh `<Button>` whose class expression forced
`opacity-100` (visible with no hover/focus) whenever `source.syncing` was
true — see the now-deleted assertions this phase's own diff removed from
`source-chip-pill.test.ts` and `source-chip-selected.test.ts`
("`the refresh control still forces itself visible while syncing (D-03)`").
That was the chip's *only* ambient, glanceable proof that a source was
mid-sync without the user having to hover or open anything — valuable
specifically for "Refresh all", where several chips sync concurrently.

Fix 5 (this phase) folded that refresh control into the `⋮` overflow
dropdown as its first `DropdownMenuItem`, whose spinning `RefreshCw` icon
(`class={cn('size-4', source.syncing && 'animate-spin')}`, line 208) is
now only visible once that menu is explicitly opened — the menu itself
only renders when the user clicks the trigger. `healthTone()`
(`web/src/lib/format.ts`) never factors `source.syncing` into the health
dot's tone/color either, and the tooltip's "— syncing…" text
(`tooltipText`, line 115) requires a hover-and-dwell to surface. The net
result: at rest, with no interaction, there is now **no visual signal
anywhere in the chip row** that a source is currently syncing — a
regression from the pre-Fix-5 behavior.

Compounding this, the surviving doc comment at lines 104-105 was not
updated and is now factually wrong about the current implementation:

```go
// The old inline "Syncing…" text label (SourceHealthChip.svelte) is
// retired; the spinning refresh icon is now the sole in-place syncing
// indicator, kept compact at scale.
```

"In-place" no longer describes an icon that only renders inside a
menu-portal the user must open first. A future reader trusting this
comment would incorrectly believe the ambient indicator still exists.

Neither `web/e2e/specs/09-chip-menu.spec.ts` nor
`chip-edit-menu.test.ts` assert anything about the chip's *resting*
(unopened, unhovered) appearance while `source.syncing` is true, so this
regression is untested as well as unremediated.
**Fix:** Either restore an ambient indicator (e.g., have the health dot
itself pulse/spin while `source.syncing`, or force the `⋮` trigger's
`opacity-100` while any chip's source is syncing, mirroring the old
refresh button's rule), or — if the regression is accepted as a deliberate
trade-off of Fix 5 — correct the stale comment at lines 104-105 to state
that the syncing indicator is now menu-gated, not "in-place," and add a
regression spec proving the deliberately-accepted resting-state absence
so a later "fix" doesn't silently reintroduce or further erode it.

## Info

### IN-01: `GET /api/plugins/{plugin}/icon`'s CSP omits the (redundant but locally-conventional) `object-src 'none'` token the sibling rendition route sets

**File:** `kernel/httpapi/pluginicon.go:88`
**Issue:** `kernel/httpapi/item.go`'s rendition route (`renditionHandler`)
sets `Content-Security-Policy: default-src 'none'; style-src
'unsafe-inline'; object-src 'none'; sandbox` for the same class of
"kernel serves source-controlled bytes from its own origin" surface. The
new plugin-icon route sets `default-src 'none'; style-src
'unsafe-inline'; sandbox` — missing `object-src 'none'`. Per the CSP
spec `object-src` already falls back to `default-src` when unset, so
`default-src 'none'` alone is functionally equivalent here and this is
not a security gap — but the inconsistency between the two closely
related, newly-adjacent hardened-header call sites in the same package is
worth normalizing so a future reader doesn't wonder whether the omission
was deliberate.
**Fix:** Append `object-src 'none';` to the CSP string in
`kernel/httpapi/pluginicon.go` to match `item.go`'s convention exactly.

---

_Reviewed: 2026-08-11T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
