---
status: partial
phase: 12-filesystem-source
source: [12-01-SUMMARY.md, 12-02-SUMMARY.md, 12-03-SUMMARY.md, 12-04-SUMMARY.md, 12-05-SUMMARY.md, 12-06-SUMMARY.md, 12-07-SUMMARY.md]
started: 2026-08-14T08:19:53.798Z
updated: 2026-08-14T08:33:40.000Z
---

## Current Test

[testing complete]

## Tests

### 1. Mobile takeover: failed-open swap string fits the narrow layout
expected: On a narrow/mobile viewport, open a webspace with a filesystem item and trigger a failed open (e.g. stop the kernel opener or use an item whose file was removed). The button label swaps to the "Couldn't open" message and the swap string does NOT overflow or break the narrow layout where the at-rest label fit.
coverage_id: 12-01/D6
rationale: 12-UI-SPEC.md itself scopes this as a backstop truth ('unverified, no visual check has been run against the mobile takeover combination') rather than an explicit one — carried forward unresolved, not newly introduced by this plan's execution.
result: issue
reported: "the filesystem plugin is not showing files that exist within the configured directory"
severity: major

### 2. Extras block height at pathological key counts
expected: In the Add/Edit source dialog for a plugin declaring extras fields, the extras block stays usable: at realistic key counts it fits naturally, and even with many keys the enclosing dialog's own scroll keeps every field reachable — no clipped or unreachable fields.
coverage_id: 12-04/D6
rationale: 12-UI-SPEC.md itself scopes this as a backstop truth, carried forward unresolved from 12-01/12-02/12-03 — not newly introduced by this plan.
result: pass

### 3. Real desktop end-to-end: previews, desktop-handler open, NFS/SMB mount
expected: With a real filesystem source on your desktop: a PDF previews inline; "Open in …" opens the file in the desktop's own handler (xdg-open); an office document shows the honest no-preview state with a working open action; a second source on an NFS/SMB mount picks up a remote write on the next sync; unmounting reports the source unreachable by name instead of the stream quietly emptying.
coverage_id: 12-05/D9
rationale: Explicitly scoped by the plan's own <verify> as a human-check — a real xdg-open handoff and a real network mount are both live, machine-dependent facts a hermetic browser harness cannot assert on. Covered instead, deterministically, by kernel/httpapi/fsopen_test.go's stubbed-Opener suite (the route contract) and by walk.go's mount-type-agnostic design (documented in docs/testing.md's new section) — but the actual desktop handoff and actual network mount remain unverified by this executor run.
result: issue
reported: "plugin shows no files in the stream"
severity: major

### 4. Docs accurately describe the symlink-resolving containment discipline
expected: Read the symlink/containment passages in docs/plugin-contract.md, docs/api.md and docs/plugins/filesystem.md: they describe the symlink-resolving containment discipline the shipped code actually enforces (post-index symlink swaps refused at both the Fetch byte-serving site and the open route; vanished files reported honestly as not-found).
coverage_id: 12-06/D6
rationale: Prose accuracy is a documentation-review judgment call, not something a unit test can assert; grep checks in the plan's acceptance criteria confirm the mechanism is named, but agreement with the code's actual behavior needs a human or reviewer read
result: blocked
blocked_by: other
reason: "as above — same underlying failure as tests 1/3: plugin shows no files in the stream; user is not validating further items until the core sync bug is fixed"

### 5. Docs honestly describe the resolved-path discipline and residual TOCTOU window
expected: Read the resolved-path passages in docs/plugins/filesystem.md, docs/api.md and docs/plugin-contract.md: they describe reading/exec-ing the EvalSymlinks-resolved path the containment check approved, and are honest about the residual TOCTOU window ("narrows but does not eliminate"), matching the shipped code.
coverage_id: 12-07/D7
rationale: Prose accuracy is a documentation-review judgment call, not something a unit test can assert; grep checks in the plan's acceptance criteria confirm the required phrases are present, but agreement with the code's actual behavior needs a human or reviewer read
result: blocked
blocked_by: other
reason: "as above — same underlying failure as tests 1/3: plugin shows no files in the stream; user is not validating further items until the core sync bug is fixed"

### 6. [12-01/D1] A folder configured as a filesystem source produces one stream item per top-level *.pdf file, with a D-01 relative-path …
expected: A folder configured as a filesystem source produces one stream item per top-level *.pdf file, with a D-01 relative-path source_id, LINK_FIDELITY_EXACT, an empty preview, and a file:// deep_link — no file body read at Match time
result: pass
source: automated
coverage_id: 12-01/D1

### 7. [12-01/D2] The kernel rewrites a file://-scheme deep_link to the loopback open route at serve time, keyed on the URL scheme alone —…
expected: The kernel rewrites a file://-scheme deep_link to the loopback open route at serve time, keyed on the URL scheme alone — every other item's deep_link is echoed unchanged
result: pass
source: automated
coverage_id: 12-01/D2

### 8. [12-01/D3] POST /api/items/{id}/open resolves the exec target exclusively from the indexed item's source_id + the source's configur…
expected: POST /api/items/{id}/open resolves the exec target exclusively from the indexed item's source_id + the source's configured Path, expands a leading ~, refuses a path escape (invalid_path), refuses a non-file:// item and an unknown item (item_not_found), and surfaces an opener failure as open_failed with the opener's own message
result: pass
source: automated
coverage_id: 12-01/D3

### 9. [12-01/D4] OpenInSource renders a real <button> (not <a>) for a same-origin /api/ link.url in both presentations, issues exactly on…
expected: OpenInSource renders a real <button> (not <a>) for a same-origin /api/ link.url in both presentations, issues exactly one POST on click, changes nothing visible on success, swaps to the destructive 'Couldn't open' label/title for 2.5s on failure (kernel detail verbatim, or the fixed fallback), and is never given a disabled attribute
result: pass
source: automated
coverage_id: 12-01/D4

### 10. [12-01/D5] The whole path proven against a real kernel and a real topos-plugin-filesystem binary: one PDF in a configured folder be…
expected: The whole path proven against a real kernel and a real topos-plugin-filesystem binary: one PDF in a configured folder becomes a stream item whose Open control POSTs to /api/items/{id}/open
result: pass
source: automated
coverage_id: 12-01/D5

### 11. [12-02/D1] A folder containing a PDF, a PNG, a markdown file, a plain-text file and a .docx yields one stream item for each, and no…
expected: A folder containing a PDF, a PNG, a markdown file, a plain-text file and a .docx yields one stream item for each, and no item for a file whose extension is outside the default document allowlist (D-03)
result: pass
source: automated
coverage_id: 12-02/D1

### 12. [12-02/D2] A PDF or an inline-renderable image (png, jpeg, gif, webp) fetches as raw bytes with its own MIME type and renders throu…
expected: A PDF or an inline-renderable image (png, jpeg, gif, webp) fetches as raw bytes with its own MIME type and renders through the existing media previewer with no kernel or UI change (D-04)
result: pass
source: automated
coverage_id: 12-02/D2

### 13. [12-02/D3] A markdown file fetches as a goldmark-rendered HTML fragment declaring CONTENT_SHAPE_MARKDOWN_HTML, and the kernel's own…
expected: A markdown file fetches as a goldmark-rendered HTML fragment declaring CONTENT_SHAPE_MARKDOWN_HTML, and the kernel's own rendition boundary is the only sanitizer (D-04)
result: pass
source: automated
coverage_id: 12-02/D3

### 14. [12-02/D4] A plain-text file fetches with its text populated and mime type text/plain, and GET /api/items/{id}/content serves it ra…
expected: A plain-text file fetches with its text populated and mime type text/plain, and GET /api/items/{id}/content serves it rather than answering unsupported_rendition_type
result: pass
source: automated
coverage_id: 12-02/D4

### 15. [12-02/D5] An office-format file (doc, docx, xls, xlsx, ppt, pptx, odt, ods, odp, rtf) fetches as unavailable with a named reason a…
expected: An office-format file (doc, docx, xls, xlsx, ppt, pptx, odt, ods, odp, rtf) fetches as unavailable with a named reason and never declares a mime type or bytes (D-04)
result: pass
source: automated
coverage_id: 12-02/D5

### 16. [12-02/D6] An image extension the kernel cannot render inline (svg, bmp, tiff, heic) appears in the stream but declares no preview,…
expected: An image extension the kernel cannot render inline (svg, bmp, tiff, heic) appears in the stream but declares no preview, rather than returning bytes the content route would refuse
result: pass
source: automated
coverage_id: 12-02/D6

### 17. [12-02/D7] A per-instance include_glob extras value widens the scope to files the default extension allowlist would have skipped, a…
expected: A per-instance include_glob extras value widens the scope to files the default extension allowlist would have skipped, and an exclude_glob value removes files the allowlist would have kept, with exclude winning over include (D-03)
result: pass
source: automated
coverage_id: 12-02/D7

### 18. [12-02/D8] The filesystem plugin's Describe response declares include_glob and exclude_glob as extras fields, so Phase 11's declare…
expected: The filesystem plugin's Describe response declares include_glob and exclude_glob as extras fields, so Phase 11's declared-fields editor renders them with no new UI code
result: pass
source: automated
coverage_id: 12-02/D8

### 19. [12-02/D9] A file larger than the byte-rendition cap fetches as unavailable with a named size reason rather than exceeding the gRPC…
expected: A file larger than the byte-rendition cap fetches as unavailable with a named size reason rather than exceeding the gRPC message ceiling
result: pass
source: automated
coverage_id: 12-02/D9

### 20. [12-03/D1] The recursive config key is typed end to end: a TOML source block declaring it decodes correctly, a canonical rewrite ne…
expected: The recursive config key is typed end to end: a TOML source block declaring it decodes correctly, a canonical rewrite never introduces it into a source that never declared it, and it reaches the plugin subprocess as a top-level envelope boolean outside extras
result: pass
source: automated
coverage_id: 12-03/D1

### 21. [12-03/D2] Recursion toggles between top-level-only and full-depth walking, each source_id forward-slash relative, proven against a…
expected: Recursion toggles between top-level-only and full-depth walking, each source_id forward-slash relative, proven against a real kernel and a real plugin binary — add and remove both reflected with no watcher dependency
result: pass
source: automated
coverage_id: 12-03/D2

### 22. [12-03/D3] A nested file carries per-segment folder labels plus the cumulative relative path (D-05), so a webspace match block can …
expected: A nested file carries per-segment folder labels plus the cumulative relative path (D-05), so a webspace match block can name either a subfolder or a full relative path
result: pass
source: automated
coverage_id: 12-03/D3

### 23. [12-03/D4] A real awkward tree — symlink loop, permission-denied subtree, dot-directory — walks to completion and returns exactly t…
expected: A real awkward tree — symlink loop, permission-denied subtree, dot-directory — walks to completion and returns exactly the documents an operator would expect
result: pass
source: automated
coverage_id: 12-03/D4

### 24. [12-03/D5] A cancelled context or a fatal root-read error returns an error rather than a partial set; a tree exceeding the per-sync…
expected: A cancelled context or a fatal root-read error returns an error rather than a partial set; a tree exceeding the per-sync item cap fails naming the cap and exclude_glob
result: pass
source: automated
coverage_id: 12-03/D5

### 25. [12-03/D6] Health distinguishes a readable-and-empty root from an unreachable one, with the OS error named as the cause
expected: Health distinguishes a readable-and-empty root from an unreachable one, with the OS error named as the cause
result: pass
source: automated
coverage_id: 12-03/D6

### 26. [12-03/D7] No non-test Go file in the filesystem plugin package references any os-package write selector — the guard is a committed…
expected: No non-test Go file in the filesystem plugin package references any os-package write selector — the guard is a committed AST scan with its own negative-control fixtures
result: pass
source: automated
coverage_id: 12-03/D7

### 27. [12-04/D1] The Checkbox primitive is installed under web/src/lib/components/ui/checkbox/, wrapping bits-ui's Checkbox with the stoc…
expected: The Checkbox primitive is installed under web/src/lib/components/ui/checkbox/, wrapping bits-ui's Checkbox with the stock unchecked-border/checked-fill mapping onto the existing --border/--primary tokens, and components.json is unchanged
result: pass
source: automated
coverage_id: 12-04/D1

### 28. [12-04/D2] ConnectionField gains optional kind ('text'|'checkbox', defaulting to text) and helperText properties; every pre-existin…
expected: ConnectionField gains optional kind ('text'|'checkbox', defaulting to text) and helperText properties; every pre-existing plugin row reports an absent (undefined) kind — no existing plugin type's form changed
result: pass
source: automated
coverage_id: 12-04/D2

### 29. [12-04/D3] The topos-plugin-filesystem connection row lists Display Name, required Local Path (two-example placeholder, no defaultV…
expected: The topos-plugin-filesystem connection row lists Display Name, required Local Path (two-example placeholder, no defaultValue), checkbox-kind Include subfolders (not required, exact helper text), and the shared advanced Sync Interval Override, in that order — required flags derived from plugins/filesystem/main.go's own fatal guard (path only)
result: pass
source: automated
coverage_id: 12-04/D3

### 30. [12-04/D4] ConnectionForm.svelte's checkbox branch: an unset/non-boolean stored value renders unchecked (never indeterminate), togg…
expected: ConnectionForm.svelte's checkbox branch: an unset/non-boolean stored value renders unchecked (never indeterminate), toggling emits a boolean under the field's own key via a boolean-widened setField, helper text renders only when declared, and the whole min-h-11 label row (not only the 16px control) is the clickable target
result: pass
source: automated
coverage_id: 12-04/D4

### 31. [12-04/D5] A user picks the filesystem plugin type from the source picker, fills in a local path, chooses whether to include subfol…
expected: A user picks the filesystem plugin type from the source picker, fills in a local path, chooses whether to include subfolders, saves, and the resulting source syncs its documents into the matching webspace — driven entirely from the UI, proving the checkbox's value actually reached the launched plugin subprocess (a nested document appears only because Include subfolders was ticked)
result: pass
source: automated
coverage_id: 12-04/D5

### 32. [12-05/D1] The same topos-plugin-filesystem binary, resolved from the external plugins directory instead of the trusted one, is dis…
expected: The same topos-plugin-filesystem binary, resolved from the external plugins directory instead of the trusted one, is discovered, pin-verified, launched and synced end to end — reporting tier external, with the untrusted badge on exactly its own source chip
result: pass
source: automated
coverage_id: 12-05/D1

### 33. [12-05/D2] The external-tier instance's webspace stream carries the same item set (including the file://-rewritten open-route link)…
expected: The external-tier instance's webspace stream carries the same item set (including the file://-rewritten open-route link) a trusted-tier corpus would — syncing behaves identically across tiers
result: pass
source: automated
coverage_id: 12-05/D2

### 34. [12-05/D3] make e2e passes in full with the filesystem plugin's real binary present in the harness — no regression to any Phase 9/1…
expected: make e2e passes in full with the filesystem plugin's real binary present in the harness — no regression to any Phase 9/11 spec
result: pass
source: automated
coverage_id: 12-05/D3

### 35. [12-05/D4] docs/plugin-contract.md documents the file:// deep-link convention as a contract-level rule keyed on URL scheme, so a th…
expected: docs/plugin-contract.md documents the file:// deep-link convention as a contract-level rule keyed on URL scheme, so a third-party local-path plugin author can adopt it without reading kernel source
result: pass
source: automated
coverage_id: 12-05/D4

### 36. [12-05/D5] docs/api.md documents POST /api/items/{id}/open, its server-side path resolution rule, its error codes, and the new text…
expected: docs/api.md documents POST /api/items/{id}/open, its server-side path resolution rule, its error codes, and the new text/plain rendition type
result: pass
source: automated
coverage_id: 12-05/D5

### 37. [12-05/D6] docs/plugins/filesystem.md exists as an operator page covering path/recursive settings, include/exclude glob extras, whi…
expected: docs/plugins/filesystem.md exists as an operator page covering path/recursive settings, include/exclude glob extras, which file types preview and which do not, and the sync interval as the real freshness bound on a network mount
result: pass
source: automated
coverage_id: 12-05/D6

### 38. [12-05/D7] config.example.toml carries a worked filesystem source block, and README.md's plugin list and count include it
expected: config.example.toml carries a worked filesystem source block, and README.md's plugin list and count include it
result: pass
source: automated
coverage_id: 12-05/D7

### 39. [12-05/D8] scripts/check-doc-links.sh passes — every link added by this pass resolves
expected: scripts/check-doc-links.sh passes — every link added by this pass resolves
result: pass
source: automated
coverage_id: 12-05/D8

### 40. [12-06/D1] A file swapped for an outward symlink after indexing is refused by the plugin's Fetch path (byte-serving site) with no b…
expected: A file swapped for an outward symlink after indexing is refused by the plugin's Fetch path (byte-serving site) with no byte of the symlink target ever served
result: pass
source: automated
coverage_id: 12-06/D1

### 41. [12-06/D2] The same post-index symlink swap is refused by POST /api/items/{id}/open with invalid_path (400), and the opener is neve…
expected: The same post-index symlink swap is refused by POST /api/items/{id}/open with invalid_path (400), and the opener is never called
result: pass
source: automated
coverage_id: 12-06/D2

### 42. [12-06/D3] A file that vanished between sync and request is reported honestly as NotFound/item_not_found at both sites, not as a co…
expected: A file that vanished between sync and request is reported honestly as NotFound/item_not_found at both sites, not as a containment violation
result: pass
source: automated
coverage_id: 12-06/D3

### 43. [12-06/D4] The context handed to the opener is detached from the HTTP request's own context, and the production opener structurally…
expected: The context handed to the opener is detached from the HTTP request's own context, and the production opener structurally cannot bind its child to a caller context
result: pass
source: automated
coverage_id: 12-06/D4

### 44. [12-06/D5] A symlinked or bind-mounted configured root still contributes its in-tree symlinked files instead of silently dropping e…
expected: A symlinked or bind-mounted configured root still contributes its in-tree symlinked files instead of silently dropping every one of them
result: pass
source: automated
coverage_id: 12-06/D5

### 45. [12-07/D1] A file admitted to the index only because include_glob widened past the default extension allowlist previews honestly as…
expected: A file admitted to the index only because include_glob widened past the default extension allowlist previews honestly as metadata-only (never a false NotFound) on both CONTENT_VARIANT_FULL and CONTENT_VARIANT_PREVIEW
result: pass
source: automated
coverage_id: 12-07/D1

### 46. [12-07/D2] A file genuinely outside scope (excluded by exclude_glob, or an unrecognized extension with no include_glob) still answe…
expected: A file genuinely outside scope (excluded by exclude_glob, or an unrecognized extension with no include_glob) still answers codes.NotFound — the honesty fix widens nothing
result: pass
source: automated
coverage_id: 12-07/D2

### 47. [12-07/D3] A malformed operator glob at Fetch time names the offending pattern and fails with codes.Unavailable, matching Match's o…
expected: A malformed operator glob at Fetch time names the offending pattern and fails with codes.Unavailable, matching Match's own answer for the identical pattern
result: pass
source: automated
coverage_id: 12-07/D3

### 48. [12-07/D4] Every fetched file's bytes are read from the filepath.EvalSymlinks-resolved path the containment check approved, through…
expected: Every fetched file's bytes are read from the filepath.EvalSymlinks-resolved path the containment check approved, through a single opened handle that also decided the size — not a second, separately-opened path
result: pass
source: automated
coverage_id: 12-07/D4

### 49. [12-07/D5] The kernel's desktop-open route execs the same resolved path its containment check approved, not the lexical one
expected: The kernel's desktop-open route execs the same resolved path its containment check approved, not the lexical one
result: pass
source: automated
coverage_id: 12-07/D5

### 50. [12-07/D6] Every filesystem item's provenance carries all five plugin-populated keys the contract documents, including source_syste…
expected: Every filesystem item's provenance carries all five plugin-populated keys the contract documents, including source_system, reaching a real API client end to end
result: pass
source: automated
coverage_id: 12-07/D6

## Summary

total: 50
passed: 46
issues: 2
pending: 0
blocked: 2
skipped: 0


## Gaps

- gap_id: G-12-1
  truth: "A configured filesystem source's documents appear in the matching webspace stream (files that exist within the configured directory are shown)"
  status: failed
  reason: "User reported: the filesystem plugin is not showing files that exist within the configured directory"
  severity: major
  test: 1
  artifacts: []  # Filled by diagnosis
  missing: []    # Filled by diagnosis

- gap_id: G-12-3
  truth: "With a real filesystem source configured, documents appear in the webspace stream with previews and a working desktop-handler open action, including on network mounts"
  status: failed
  reason: "User reported: plugin shows no files in the stream (same observable failure as G-12-1 — no filesystem items reach the stream)"
  severity: major
  test: 3
  artifacts: []  # Filled by diagnosis
  missing: []    # Filled by diagnosis
