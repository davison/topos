---
phase: 12-filesystem-source
plan: 05
subsystem: source-plugin
tags: [e2e, playwright, docs, plugin-contract, api-docs, config-example, external-plugins]

requires:
  - phase: 11-external-plugins-the-trust-boundary
    provides: "two-tier plugin discovery/launch, extras machinery, and the e2e fixture harness's externalPluginBinaries convention this plan's rehearsal spec reuses unmodified"
  - phase: 12-filesystem-source
    provides: "12-01's file://-scheme deep-link convention and kernel-mediated open route, 12-02's document-scope classifier and preview-shape Fetch dispatch, 12-03's recursion-aware walk and honest health — the real, full-featured plugin this plan proves against the external tier and documents in full"

provides:
  - "web/e2e/specs/12-external-rehearsal.spec.ts — criterion 5's proof against a real source plugin (topos-plugin-filesystem), not a fixture-only proof binary: the same binary, loaded from the external directory only, discovered/pin-verified/launched/synced identically to the trusted-tier specs, with the untrusted badge on exactly its own chip"
  - "docs/plugins/filesystem.md — the operator page: path/recursive settings, include_glob/exclude_glob precedence, the default document allowlist by preview shape, D-05 folder-vocabulary match values, symlink/dot-file policy, the mount-gone-vs-empty-folder distinction, and why the sync interval is the real freshness bound on a network mount"
  - "docs/plugin-contract.md's file:// local-path deep-link convention, published as a contract-level rule keyed on URL scheme alone — a third-party local-path plugin author can adopt kernel-mediated 'Open in ...' with zero kernel change"
  - "docs/api.md's POST /api/items/{id}/open route entry (request/response, all three failure modes, error-code table rows) and the text/plain rendition-type addition to the MIME-allowlist prose"
  - "docs/testing.md's account of this phase's four e2e specs, plus the honest scope statement of what a hermetic browser harness cannot prove (a real xdg-open handoff; a real NFS/SMB mount) and how each is covered instead"
  - "config.example.toml's worked [sources.filesystem] block, matching the existing local-path source style, with a commented extras example for both glob keys"

affects: [14-google-drive-plugin]

actuals:
  tokens: 9800
  tasks: 2
  commits: 2

tech-stack:
  added: []
  patterns:
    - "e2e rehearsal fixture shape for 'the same real binary, external tier only': name the binary in externalPluginBinaries and set pluginBinaries: [] explicitly, rather than relying on the harness default — makes the D-11 tier-collision avoidance visible in the fixture itself, not just in a comment"

key-files:
  created:
    - web/e2e/specs/12-external-rehearsal.spec.ts
    - docs/plugins/filesystem.md
    - .planning/phases/12-filesystem-source/deferred-items.md
  modified:
    - docs/plugins/README.md
    - docs/plugin-contract.md
    - docs/api.md
    - docs/testing.md
    - config.example.toml
    - README.md

key-decisions:
  - "The rehearsal spec sets pluginBinaries: [] rather than leaving it undefined — the harness's own default (['topos-plugin-mock']) would already have kept the trusted directory free of topos-plugin-filesystem, but an explicit empty list makes the D-11 tier-collision avoidance a visible, load-bearing fixture choice rather than an accidental byproduct of an unrelated default."
  - "README.md's Prebuilt section gets an explicit caveat (filesystem plugin binary not yet in the published release/nightly asset list) rather than silently omitting or falsely including it — the release workflow gap this surfaced is real (release.yml/nightly.yml never added it across 12-01-12-04) but is release-engineering work outside this plan's declared files; logged to deferred-items.md rather than fixed here."

patterns-established: []

requirements-completed: [SRC-04]

coverage:
  - id: D1
    description: "The same topos-plugin-filesystem binary, resolved from the external plugins directory instead of the trusted one, is discovered, pin-verified, launched and synced end to end — reporting tier external, with the untrusted badge on exactly its own source chip"
    requirement: SRC-04
    verification:
      - kind: e2e
        ref: "web/e2e/specs/12-external-rehearsal.spec.ts — 'GET /api/sources reports the instance reachable with tier external'"
        status: pass
      - kind: e2e
        ref: "web/e2e/specs/12-external-rehearsal.spec.ts — 'the source chip carries the untrusted badge, and it is the only chip that does'"
        status: pass
    human_judgment: false

  - id: D2
    description: "The external-tier instance's webspace stream carries the same item set (including the file://-rewritten open-route link) a trusted-tier corpus would — syncing behaves identically across tiers"
    requirement: SRC-04
    verification:
      - kind: e2e
        ref: "web/e2e/specs/12-external-rehearsal.spec.ts — 'the webspace stream carries the same item set a trusted-tier corpus would, including the rewritten open-route link'"
        status: pass
    human_judgment: false

  - id: D3
    description: "make e2e passes in full with the filesystem plugin's real binary present in the harness — no regression to any Phase 9/11 spec"
    requirement: SRC-04
    verification:
      - kind: e2e
        ref: "make e2e — 113 passed"
        status: pass
    human_judgment: false

  - id: D4
    description: "docs/plugin-contract.md documents the file:// deep-link convention as a contract-level rule keyed on URL scheme, so a third-party local-path plugin author can adopt it without reading kernel source"
    requirement: SRC-04
    verification:
      - kind: other
        ref: "grep -n 'file://' docs/plugin-contract.md and 'trigger is the URL scheme' — both present"
        status: pass
    human_judgment: false

  - id: D5
    description: "docs/api.md documents POST /api/items/{id}/open, its server-side path resolution rule, its error codes, and the new text/plain rendition type"
    requirement: SRC-04
    verification:
      - kind: other
        ref: "grep -n 'items/{id}/open' and 'text/plain' docs/api.md — both present, plus the three new error-code table rows"
        status: pass
    human_judgment: false

  - id: D6
    description: "docs/plugins/filesystem.md exists as an operator page covering path/recursive settings, include/exclude glob extras, which file types preview and which do not, and the sync interval as the real freshness bound on a network mount"
    requirement: SRC-04
    verification:
      - kind: other
        ref: "docs/plugins/filesystem.md written and linked from docs/plugins/README.md; every documented default/precedence rule checked against plugins/filesystem/scope.go and classify.go while writing"
        status: pass
    human_judgment: false

  - id: D7
    description: "config.example.toml carries a worked filesystem source block, and README.md's plugin list and count include it"
    requirement: SRC-04
    verification:
      - kind: other
        ref: "python3 tomllib parse of config.example.toml (valid TOML); grep -n 'filesystem' README.md docs/plugins/README.md docs/testing.md"
        status: pass
    human_judgment: false

  - id: D8
    description: "scripts/check-doc-links.sh passes — every link added by this pass resolves"
    requirement: SRC-04
    verification:
      - kind: other
        ref: "make docs-check — checked 38 link(s) across 19 file(s) — all resolve"
        status: pass
    human_judgment: false

  - id: D9
    description: "On your own desktop, with a real filesystem source configured: PDF inline preview, 'Open in ...' opens in the desktop's own handler, an office document shows the no-preview state with a working open action, a second source on an NFS/SMB mount picks up a remote write on the next sync, and unmounting reports the source unreachable by name rather than the stream quietly emptying"
    verification: []
    human_judgment: true
    rationale: "Explicitly scoped by the plan's own <verify> as a human-check — a real xdg-open handoff and a real network mount are both live, machine-dependent facts a hermetic browser harness cannot assert on. Covered instead, deterministically, by kernel/httpapi/fsopen_test.go's stubbed-Opener suite (the route contract) and by walk.go's mount-type-agnostic design (documented in docs/testing.md's new section) — but the actual desktop handoff and actual network mount remain unverified by this executor run."

duration: ~50min
completed: 2026-08-13
status: complete
---

# Phase 12 Plan 05: External-Tier Rehearsal and the Documentation Republish Summary

**Criterion 5 is now proven against a real source plugin (not a fixture binary) loaded from the external/untrusted directory, and every document Phase 12 made wrong — the plugin contract's new deep-link convention, the new HTTP route, the new rendition type, the operator page, the example config, and the testing map — is republished to match the shipped code.**

## Performance

- **Duration:** ~50 min
- **Started:** 2026-08-13T22:10:00Z (approx.)
- **Completed:** 2026-08-13T23:08:42Z
- **Tasks:** 2
- **Files modified:** 9 (3 created, 6 modified)

## Accomplishments

- `web/e2e/specs/12-external-rehearsal.spec.ts`: the real `topos-plugin-filesystem` binary — the same one every trusted-tier spec in this phase already exercises, not a purpose-built proof binary — is named only in `externalPluginBinaries` (with `pluginBinaries: []` making the trusted directory explicitly, legitimately empty) so Phase 11 D-11's "a binary present in both directories resolves trusted" rule can't silently make the spec prove nothing. Three assertions, in order: `GET /api/sources` reports `tier: "external"` and `reachable: true`; the webspace stream carries the same two-item set (PDF + markdown) and the same `file://`-rewritten open-route link a trusted-tier corpus would; and the browser shows the untrusted badge on exactly that one chip. `make e2e` passes in full (113 tests) with the new plugin present in the harness — no regression to any earlier phase's spec.
- `docs/plugin-contract.md` gained "The `file://` local-path deep-link convention" as a real subsection (not an appendix): a plugin whose items are local files sets `deep_link` to a `file://` URI over the real absolute path; the kernel rewrites it to the loopback open route at serve time, keyed on the URL scheme alone — never `source_type` — so a future third-party local-path plugin gets the behavior for free, with zero kernel change and no contract version bump.
- `docs/api.md` gained `POST /api/items/{id}/open` as a full route entry (request/response shape, the "path comes from index state plus config, never the request" rule, and all three failure modes: `404 item_not_found`, `400 invalid_path`, `502 open_failed`), plus three new error-code table rows and `text/plain` added to the rendition MIME-allowlist prose.
- `docs/plugins/filesystem.md` is a new operator page (from `_template.md`, in `docs/plugins/signal.md`'s local-path-source style): `path`/`recursive` settings, the `include_glob`/`exclude_glob` extras and their exclude-beats-include, include-replaces-not-widens precedence, the default document allowlist by preview shape (inline vs. metadata-only, with the deferred office-conversion idea named as a decision, not an oversight), D-05 folder-vocabulary match values for top-level and nested files, the symlink/dot-file policy, the mount-gone-vs-empty-folder distinction (`os.ReadDir`, not `os.Stat`), and why the sync interval — not any change-notification mechanism — is the real freshness bound on a network mount. Linked from `docs/plugins/README.md`.
- `docs/testing.md` gained a new section naming this phase's four e2e specs (including `12-filesystem-add-source.spec.ts`, the connection-form UI spec from the sibling 12-04 plan running in parallel) and stating plainly what a hermetic browser harness cannot prove — a real `xdg-open` handoff and a real NFS/SMB mount — and how each is covered instead (the stubbed-`Opener` Go test suite; the mount-type-agnostic polling design itself).
- `config.example.toml` gained a worked `[sources.filesystem]` block in the same depth/style as the existing `[sources.signal]`/`[sources.whatsapp]` local-path blocks, plus a commented `[sources.filesystem.extras]` example showing both glob keys.
- `README.md`: "six sources ship today" (was five), the `docs/plugins/` count corrected to six, and an explicit, honest caveat added to the Prebuilt section — the filesystem plugin binary is not yet in the published release/nightly asset list (a real gap this task surfaced but did not fix; see Deviations).

## Task Commits

1. **Task 1: External-tier rehearsal — the real filesystem binary, loaded untrusted** — `aabd3ae` (feat)
2. **Task 2: Republish the contract, the API reference, the operator pages and the example config** — `29dbbcd` (docs)

**Plan metadata:** this SUMMARY's own commit (pending, see below)

## Files Created/Modified

- `web/e2e/specs/12-external-rehearsal.spec.ts` — the new rehearsal spec (created)
- `docs/plugins/filesystem.md` — the new operator page (created)
- `.planning/phases/12-filesystem-source/deferred-items.md` — the release-workflow gap this task surfaced but did not fix (created)
- `docs/plugins/README.md` — added the filesystem page to the index list
- `docs/plugin-contract.md` — the `file://` deep-link convention subsection
- `docs/api.md` — the `POST /api/items/{id}/open` route entry, three new error-code rows, `text/plain` in the MIME-allowlist prose
- `docs/testing.md` — the new "filesystem source, real end to end" section and a dated "What changed" entry
- `config.example.toml` — the worked `[sources.filesystem]` block
- `README.md` — source count/list, `docs/plugins/` count, the Prebuilt-section caveat

## Decisions Made

- **The rehearsal spec's `pluginBinaries: []`:** the harness's own default (`['topos-plugin-mock']`) would already have kept `topos-plugin-filesystem` out of the trusted directory, but an explicit empty list makes the D-11 tier-collision avoidance a visible, load-bearing fixture choice rather than an accidental byproduct of an unrelated default — a future reader of this spec sees the safety property stated, not merely implied.
- **README's Prebuilt-section caveat over silence or a false claim:** the filesystem plugin genuinely isn't in `.github/workflows/release.yml`'s or `nightly.yml`'s `ASSETS` list yet (verified by reading both files), even though `make plugins-portable` already builds it. Claiming it ships in the prebuilt release (per the plan's threat register T-12-24, "documenting behaviour that was not built") would have been false; silently omitting it while the Status section now says "six sources ship today" would have been an inconsistency a careful reader could catch. An explicit caveat, matching the existing Signal caveat's shape, is the honest middle path — and the underlying gap is logged for a future fix (see Deviations).

## Deviations from Plan

### Auto-fixed Issues

None — plan executed exactly as written for both tasks; every acceptance criterion was met without needing a Rule 1/2/3 fix.

### Out-of-Scope Discovery (logged, not fixed)

**1. `topos-plugin-filesystem` missing from `.github/workflows/release.yml` and `nightly.yml`'s published `ASSETS` list**
- **Found during:** Task 2, verifying README.md's Prebuilt section against the actual CI release pipeline.
- **Issue:** Both workflows hardcode an `ASSETS` variable naming exactly four non-Signal plugin binaries; `topos-plugin-filesystem` was never added across 12-01 through 12-04, even though it needs no cgo and `make plugins-portable` (which `make build-portable`, the workflows' own entry point, delegates to) already builds it alongside the other five portable binaries.
- **Why not fixed here:** `.github/workflows/release.yml`/`nightly.yml` are not in this plan's declared `<files>` list and are release engineering, not documentation — outside the scope-boundary rule's "directly caused by the current task's changes" test. The gap predates this task.
- **Action taken:** Logged to `.planning/phases/12-filesystem-source/deferred-items.md` with the exact fix (add `plugins/topos-plugin-filesystem` to both `ASSETS` lines) named. README.md's Prebuilt section states the current, honest state (built from source only, for now) rather than a false or silent claim.
- **Also attempted:** `gsd_run windows append --kind deviation ...` to record this in `.planning/WINDOWS.md` — failed with `Error: Ledger entry 5 has invalid status: "resolved"`, a pre-existing malformed entry unrelated to this task. Per the ledger's own best-effort policy (population never blocks execution), this was not chased further; the deferred-items.md entry is the durable record.

---

**Total deviations:** 0 auto-fixed; 1 out-of-scope discovery logged (release-workflow asset-list gap, pre-existing, not caused by this task).
**Impact on plan:** None on either task's own scope or acceptance criteria — both were met exactly as written.

## Issues Encountered

- Stray `plugins/*/paperless`, `plugins/*/silverbullet`, `plugins/*/proton`, `plugins/*/whatsapp`, `plugins/*/mockstrict`, `plugins/*/mock`, and `plugins/*/filesystem` binaries were produced by `go build ./...` (no `-o` flag) inside each plugin's own directory during `make test-portable` — deleted before committing, same as every prior plan in this phase noted.
- `kernel/webui/build/.gitkeep` was overwritten by `npm run build` (invoked by `make e2e`) — restored via `git checkout -- kernel/webui/build/.gitkeep` before Task 1's commit, same as 12-01/12-03's own notes.
- The `gsd_run windows append` call for the deferred release-workflow gap failed on a pre-existing malformed WINDOWS.md entry (`entry 5 has invalid status: "resolved"`) unrelated to this task — not chased further, per the ledger's best-effort/non-blocking policy; recorded instead in `deferred-items.md`.

## User Setup Required

None for the documentation and test changes themselves. The plan's own end-of-phase human check (real desktop `xdg-open` handoff, an office document's no-preview state, a real NFS/SMB mount add/remove, and an unmount reporting the source unreachable by name) remains genuinely unverified by this executor run — it is explicitly scoped as a human-check in the plan's own `<verify>` block (D9, above), not something a hermetic harness can prove.

## Next Phase Readiness

- Criterion 5 is now met by a real source plugin's binary on the external path, not a fixture-only proof binary — Phase 14's out-of-repo Google Drive plugin work can proceed with this checkpoint closed.
- Every document a reader would have found wrong after 12-01 through 12-04 is now corrected: the plugin contract's new `file://` convention, the new HTTP route and rendition type, the new operator page, the worked example config, and the testing map's account of this phase's specs.
- The filesystem plugin's own release-publication gap (not in `release.yml`/`nightly.yml`'s `ASSETS` list) is logged in `deferred-items.md` for a future maintenance pass — genuinely out of this phase's own scope, not silently dropped.
- D6 (the narrow-layout "Couldn't open" overflow backstop, carried forward from 12-01/12-02/12-03) remains genuinely unverified — untouched by this plan's scope.

## Self-Check: PASSED

- FOUND: `web/e2e/specs/12-external-rehearsal.spec.ts`, `docs/plugins/filesystem.md`, `.planning/phases/12-filesystem-source/deferred-items.md`
- FOUND commits: `aabd3ae`, `29dbbcd` (both present in `git log --oneline`)

---
*Phase: 12-filesystem-source*
*Completed: 2026-08-13*
