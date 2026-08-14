---
phase: 13-per-item-curation-installable-app
reviewed: 2026-08-14T22:43:44Z
depth: standard
files_reviewed: 67
files_reviewed_list:
  - cmd/topos/main.go
  - cmd/topos-manifest/main.go
  - cmd/topos/mime_test.go
  - cmd/topos/shutdown_signal_test.go
  - docs/api.md
  - docs/plugin-contract.md
  - docs/plugins/signal.md
  - docs/testing.md
  - .gitignore
  - kernel/correlate/correlate_test.go
  - kernel/httpapi/agent.go
  - kernel/httpapi/agent_test.go
  - kernel/httpapi/config_test.go
  - kernel/httpapi/contract_test.go
  - kernel/httpapi/marks.go
  - kernel/httpapi/marks_test.go
  - kernel/httpapi/routes.go
  - kernel/httpapi/sources.go
  - kernel/httpapi/sources_test.go
  - kernel/httpapi/stream.go
  - kernel/httpapi/stream_test.go
  - kernel/index/marks_test.go
  - kernel/index/schema.go
  - kernel/index/store.go
  - kernel/index/store_test.go
  - kernel/pluginhost/describe_test.go
  - kernel/pluginhost/describe_whatsapp_test.go
  - kernel/pluginhost/discover_binaries.go
  - kernel/pluginhost/extras_test.go
  - kernel/pluginhost/host.go
  - kernel/pluginhost/manifestgate_test.go
  - kernel/pluginhost/manifest.go
  - kernel/pluginhost/manifest_test.go
  - kernel/pluginhost/pin_test.go
  - kernel/pluginhost/reconcile_test.go
  - kernel/pluginhost/stderr_test.go
  - kernel/pluginhost/tier_test.go
  - kernel/supervisor/readiness_test.go
  - kernel/supervisor/supervisor_test.go
  - kernel/syncer/coordinator_test.go
  - Makefile
  - README.md
  - web/e2e/fixtures/config-builder.ts
  - web/e2e/fixtures/kernel.ts
  - web/e2e/fixtures/plugin-binaries.ts
  - web/e2e/specs/13-excluded-view.spec.ts
  - web/e2e/specs/13-exclude-tracer.spec.ts
  - web/e2e/specs/13-manifest-unverified.spec.ts
  - web/e2e/specs/13-multi-select-bulk-exclude.spec.ts
  - web/e2e/specs/13-pwa-manifest-sw.spec.ts
  - web/e2e/specs/13-shadowed-advisory.spec.ts
  - web/e2e/specs/uat-04-zero-webspace-vs-outage.spec.ts
  - web/package.json
  - web/pwa-assets.config.ts
  - web/src/app.d.ts
  - web/src/app.html
  - web/src/lib/api.ts
  - web/src/lib/components/DetailPane.svelte
  - web/src/lib/components/match-advisory.test.ts
  - web/src/lib/components/save-state.test.ts
  - web/src/lib/components/SelectionActionBar.svelte
  - web/src/lib/components/selection-action-bar.test.ts
  - web/src/lib/components/SourceChip.svelte
  - web/src/lib/components/source-chip-tooltip.test.ts
  - web/src/lib/components/sources.test.ts
  - web/src/lib/components/StreamList.svelte
  - web/src/lib/components/StreamRow.svelte
  - web/src/lib/components/ui/overlay-primitives.test.ts
  - web/src/lib/components/ui/sonner/index.ts
  - web/src/lib/components/ui/sonner/sonner.svelte
  - web/src/lib/components/WebspaceHeader.svelte
  - web/src/lib/format.ts
  - web/src/lib/pwa-update.test.ts
  - web/src/lib/pwa-update.ts
  - web/src/lib/selection.test.ts
  - web/src/lib/selection.ts
  - web/src/lib/toast.test.ts
  - web/src/lib/toast.ts
  - web/src/routes/+layout.svelte
  - web/src/routes/+page.svelte
  - web/src/routes/root-empty-state.test.ts
  - web/src/routes/w/[webspace]/+page.svelte
  - web/vite.config.ts
findings:
  critical: 0
  warning: 3
  info: 2
  total: 5
status: issues_found
---

# Phase 13: Code Review Report

**Reviewed:** 2026-08-14T22:43:44Z
**Depth:** standard
**Files Reviewed:** 67
**Status:** issues_found

## Summary

This phase adds three largely independent surfaces: (1) per-item
exclude/include "marks" (kernel storage, `POST /api/webspaces/{ws}/marks`,
`?view=` on the stream route, and the bulk-select/undo-toast UI), (2) PWA
installability (manifest, service worker, self-update polling), and (3) a
link-time build-provenance manifest that gates trusted-tier plugin
launches against binary tampering/drop-in attacks.

The Go-side work is careful and well-tested: `SetItemMarks`/`ClearItemMarks`
bind every id as a parameter (no SQL injection surface), the mark-prune
sweep is correctly scoped to the healthy-sync transaction only, the
stream/search/agent-mirror exclusion parity is directly proven by a
three-surface test, and the new manifest gate (`VerifyTrustedBinary`,
`resolveBinaryDetailed`) closes the "drop a binary into the trusted
directory" bypass with both a unit test and a real e2e proof
(`13-manifest-unverified.spec.ts`). I did not find an injection, auth
bypass, or data-loss defect in the reviewed diff.

The frontend marks/undo flow has one real correctness gap: the toast
"Undo" action captures the page's reactive `webspace`/`navGeneration`
state by closure rather than snapshotting it at the moment the toast is
created, so clicking "Undo" after navigating to a different webspace
silently targets the wrong webspace instead of undoing the original
action. I also flag the marks write endpoint's lack of a request-body
size cap (relevant because its own doc comment explicitly claims a DoS
mitigation, T-13-03, that only partially holds), plus two minor
maintainability notes.

## Warnings

### WR-01: Toast "Undo" for exclude/include captures live reactive state, not a snapshot — undoing after a webspace switch silently no-ops

**File:** `web/src/routes/w/[webspace]/+page.svelte:116-156` (single-item `handleExclude`/`handleInclude`) and `:208-235` (`handleBulkPrimary`)

**Issue:** `webspace` is declared as `let webspace = $derived(page.params.webspace ?? '')` (line 50) — a reactive binding to the current route param. The same `+page.svelte` component instance is explicitly documented (and relied upon elsewhere via `navGeneration`, see the comment at line ~666) to be *reused* across `/w/A -> /w/B` navigation. `markSuccessToast`'s `onUndo` callback:

```js
onUndo: async () => {
  await setItemMarks(webspace, 'remove', [id]);
  await load(navGeneration);
}
```

(and the mirror in `handleInclude`, and `handleBulkPrimary`'s `onUndo` with `ids`/`undoAction`) reads `webspace` and `navGeneration` **at call time**, i.e. whenever the user actually clicks "Undo" — which can be up to 5000ms after the toast fired (`markSuccessToast`'s own `duration: 5000`), plenty of time for a user to switch webspaces via `WebspaceSwitcher` before clicking Undo.

If the user excludes an item in webspace A, then navigates to webspace B, then clicks "Undo" on the still-visible toast: `setItemMarks` is issued against webspace **B** (not A), and `load(navGeneration)` refreshes webspace B's stream. Because `item_marks` is keyed `(webspace_name, item_id, kind)`, this call typically matches zero rows in B (`changed: 0`) — the original exclusion in A is **never undone**, the toast still dismisses as if it succeeded, and no error is ever surfaced to the user. This directly undermines 13-UI-SPEC.md's "Undo" contract (D-02: trivially reversible) for exactly the case a toast's delayed-action design exists to support.

Every other delayed/async write path in this same file (`writeFilter`, `handleSearch`, `load` itself) captures `const gen = navGeneration` (or the equivalent) at the top of the function specifically to guard against this class of staleness — this is the one write path that doesn't, and it's also the only one with a multi-second delay window before it fires.

**Fix:** Snapshot both values when the toast is created, not when it fires:

```js
async function handleExclude(id: string) {
  const ws = webspace;
  const gen = navGeneration;
  markBusy = true;
  try {
    await setItemMarks(ws, 'add', [id]);
    closeDetail();
    await load(gen);
    markSuccessToast({
      verb: 'Excluded',
      count: 1,
      onUndo: async () => {
        await setItemMarks(ws, 'remove', [id]);
        await load(gen);
      }
    });
  } catch {
    markFailureToast({ verb: 'exclude', count: 1 });
  } finally {
    markBusy = false;
  }
}
```

Apply the same snapshot to `handleInclude` and `handleBulkPrimary`. (`load(gen)` will still correctly no-op if a *newer* navigation has since superseded `gen`, matching `load`'s existing generation-guard contract — the fix only stops the "silently write into whatever is current" failure mode.)

---

### WR-02: `POST /api/webspaces/{webspace}/marks` decodes the full request body before enforcing the 1000-id cap, leaving the DoS mitigation the code claims only partially closed

**File:** `kernel/httpapi/marks.go:78-100`

**Issue:** The handler's own doc comment states the `maxMarksItemIDs` cap exists specifically to close "T-13-03, DoS": *"an over-cap request is rejected 400 before any transaction opens, never partially processed."* That's true for the **database** side (no partial writes), but the cap is checked only *after* `json.NewDecoder(r.Body).Decode(&req)` has already fully materialized the request into memory:

```go
var req marksRequest
if err := json.NewDecoder(r.Body).Decode(&req); err != nil { ... }
...
if len(req.ItemIDs) > maxMarksItemIDs {
    WriteError(w, http.StatusBadRequest, "invalid_request", ...)
    return
}
```

There is no `http.MaxBytesReader` (or any other body-size limit) anywhere on this route, so a caller can send an arbitrarily large JSON body (e.g. one very long string, or hundreds of thousands of short ids) and the full cost of parsing/allocating it is paid before the cap ever has a chance to reject it. This is a pre-existing pattern across every other POST/PUT handler in `kernel/httpapi` (none of them use `MaxBytesReader` either), so it isn't unique to this phase — but this is the one handler whose own comment specifically frames the cap as a DoS control, so the gap is worth calling out here rather than silently inheriting it.

Given this is a loopback-only, single-user kernel by design (with a startup warning if bound elsewhere), the practical exploitability is low, but the code's own stated threat model is not fully met.

**Fix:** Wrap the body reader: `r.Body = http.MaxBytesReader(w, r.Body, someBound)` before decoding (a bound comfortably above `maxMarksItemIDs * len(longest realistic id)`, e.g. 256KB), and return `invalid_request` on the resulting decode error. Consider doing this at the `Router` level for every mutating route rather than one at a time.

---

### WR-03: `ParseManifest`/`validatePluginBinaryName` do not reject a comma in a plugin binary name, which would corrupt the manifest's comma-separated wire format

**File:** `kernel/pluginhost/manifest.go:83-117`, `kernel/pluginhost/discover_binaries.go:326-335`

**Issue:** `FormatManifest`/`ParseManifest` use `,` as the entry separator and `=` as the key/value separator. `validatePluginBinaryName` (shared by both `ParseManifest`'s segment validation and `ResolveBinary`'s launch-time confinement check) rejects `/`, `\`, `.`, `..`, and any name that isn't already `filepath.Base(name)` — but does **not** reject `,` or `=`. If a plugin binary's filename ever contained a comma (e.g. hand-renamed by an operator, or a future build script that isn't careful), `ManifestEntriesForBinaries`/`FormatManifest` would happily produce a manifest string with a spurious extra segment, and `TrustManifest()`'s parse-once cache would then hit `ParseManifest`'s error path and fail closed to an **empty** manifest — silently disabling every trusted-tier plugin launch kernel-wide (not a security regression, since it fails closed, but a confusing, hard-to-diagnose availability bug with no explicit error surfaced to the operator beyond "every trusted plugin now refuses to launch").

**Fix:** Either reject `,` and `=` in `validatePluginBinaryName` (tightening the four-rule confinement check that already exists), or switch the manifest wire format to something that can't collide with a legal (if unusual) filename character (e.g. newline-delimited, or a JSON array) so a binary's own name can never corrupt the format that names it. Low priority given real build recipes only ever produce filenames from the fixed `topos-plugin-<name>` convention, but worth tightening since the validator already exists and is meant to be the confinement authority.

## Info

### IN-01: `SetItemMarks`/`ClearItemMarks` issue one `Exec` per id inside the shared transaction rather than a single prepared statement

**File:** `kernel/index/store.go:406-473`

**Issue:** Both methods loop over `itemIDs` and call `tx.ExecContext` with the same SQL text on every iteration. This re-parses/re-plans the statement each time rather than using `tx.PrepareContext` once and reusing the prepared statement across the loop. Functionally correct (all writes happen in one transaction, `RowsAffected` is summed correctly), and the 1000-id cap bounds the worst case, but it's an easy, low-risk tidy-up now that the pattern exists in two places.

**Fix:** `stmt, err := tx.PrepareContext(ctx, ...); defer stmt.Close()`, then `stmt.ExecContext(ctx, webspaceName, id, kind)` inside the loop.

### IN-02: `handleGlobalKeydown`'s Esc-clears-selection listener is registered globally for the page's entire lifetime with no scoping to when bulk mode is active

**File:** `web/src/routes/w/[webspace]/+page.svelte:254-262`

**Issue:** The `$effect` registering the `keydown` listener has no reactive dependencies read in its own body (only inside the nested handler), so it runs exactly once per component mount and never re-runs — which is fine functionally (the handler internally checks `bulkSelection.size > 0` before acting), but means every keypress on the page invokes this handler for the component's entire lifetime, including while typing in the search box or any form field, purely to check a size. Not a bug (Escape is rarely typed into text inputs, and the check is cheap), but worth a code comment noting the handler is intentionally unscoped so a future reader doesn't "fix" it into a per-bulk-session listener that then leaks across the auto-flip/view-toggle reset paths.

**Fix:** None required; documentation-only note for future maintainers.

---

_Reviewed: 2026-08-14T22:43:44Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
