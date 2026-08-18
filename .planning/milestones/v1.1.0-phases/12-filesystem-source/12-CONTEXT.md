# Phase 12: Filesystem Source - Context

**Gathered:** 2026-08-13
**Status:** Ready for planning

<domain>
## Phase Boundary

An in-repo, trusted source plugin that turns a folder — local or on a network mount (NFS/SMB) — into a topos source: its documents appear in the matching webspace streams with previews and deep links, kept current by stat-diff polling (no reliance on OS change notifications), strictly read-only, with subfolder recursion on or off per instance. Criterion 5 additionally rehearses the Phase 11 external path: the same binary, copied into the external plugins directory, loads and syncs identically under the untrusted badge.

Requirement: SRC-04. Depends on Phase 11 only for the external-path rehearsal criterion.

</domain>

<decisions>
## Implementation Decisions

### Stable item identity
- **D-01:** The per-source stable ID is the **path relative to the source folder**. Deterministic on every mount type (NFS/SMB fileid/inode stability is not guaranteed, and copy/restore/rsync churns inodes anyway). — **Reversibility:** costly — index rows, Phase 13 exclusions, and dedup all key on this ID; changing the scheme later forces an index rebuild for existing filesystem sources.
- **D-02:** A rename or move is honestly a **remove + add** (old item disappears, new item appears). No content-hash rename detection — the existing sync-integrity machinery already handles removals cleanly, and hash-assisted re-identification would cost a full read of every changed file plus a corner-case matrix.

### Document scope
- **D-03:** Default scope is a **document-ish allowlist** (PDF, office formats, markdown, plain text, images). Per-instance **extras keys widen or narrow it** (e.g. include/exclude globs) — the escape hatch for unusual folders, and a real exercise of Phase 11's extras machinery (D-12/D-13/D-15 from 11-CONTEXT.md) on an in-repo plugin.

### Previews
- **D-04:** Previews **reuse the existing Fetch bytes+MIME pipeline** — no new kernel or UI rendering work. PDFs and images return raw bytes with their MIME type and render inline via the existing media previewer (paperless precedent); markdown returns rendered content through the kernel rendition boundary (SilverBullet shape precedent); plain text returns as text; office formats (docx/xlsx/…) get metadata + deep link only — browsers can't render them natively, and server-side conversion is explicitly out of scope this phase.

### Webspace matching
- **D-05:** The plugin's declared match vocabulary is **folder paths** — the filesystem's native categorization, mirroring how email folders/labels work. A webspace's match block lists subfolder paths/names; the keywords fallback matches folder names the same way. One instance can serve many webspaces. No filename-token match field.

### Deep links
- **D-06:** Clicking a filesystem item triggers a **kernel-mediated open**: a small, loopback-only kernel endpoint runs `xdg-open` on the item's real path, so the document opens in the desktop's own handler — the full criterion-3 experience, declared at navigating fidelity. The endpoint MUST be constrained to paths of currently-indexed items (never an arbitrary caller-supplied path) and treated with the same care as the kernel's other exec surface (the WhatsApp link spawner precedent shows the shape, but this is new machinery, not reuse).

### Claude's Discretion
- Change-detection signal within stat-diff polling (mtime+size is the obvious default), sync cadence, and large-tree scan strategy.
- Exact allowlist contents (which extensions per category) and the extras key names/glob syntax for scope overrides.
- Hidden-file/dot-directory and in-tree symlink handling defaults.
- Read-only guard specifics (same committed-guard pattern as every other plugin).
- External-rehearsal test setup (Phase 11 D-11 already settled tier-collision semantics; the fixture/harness approach is the planner's call, with `testdata/external-plugin` and the Phase 11 e2e fixtures as precedent).
- Whether "open containing folder" is offered as a secondary affordance alongside the primary open action.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### Plugin contract & prior trust-boundary decisions
- `docs/plugin-contract.md` — the published `topos.v1` contract: Item/fidelity/deep-link validation rules, ContentShape, extras declaration, trust tiers, launch env allowlist
- `proto/topos/v1/plugin.proto` — wire truth for Match/Fetch/Describe, ContentVariant, LinkFidelity, ExtrasField
- `.planning/phases/11-external-plugins-the-trust-boundary/11-CONTEXT.md` — locked Phase 11 decisions this phase inherits: external dir/discovery (D-09/D-10), tier collision (D-11), extras shape (D-12/D-13/D-15), env hygiene (D-14)

### Working precedents in-repo
- `plugins/paperless/` — binary-rendition Fetch (PDF bytes + `application/pdf` MIME) that the fs plugin's preview path reuses; read-only + outbound-host guard test pattern
- `plugins/silverbullet/` — rendered-markdown shape through the kernel rendition boundary; frontmatter/tag handling reference
- `docs/testing.md` — the testing map every phase extends (e2e-as-definition-of-done convention)

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- Fetch bytes+MIME → media previewer pipeline (`kernel/httpapi/item.go`, `web/src/lib/components/DetailPane.svelte`): renders PDF/images inline today for paperless — fs plugin returns file bytes + MIME and gets previews for free
- Kernel rendition boundary (sanitize/wrap/theme by declared ContentShape): markdown previews follow the SilverBullet path
- Phase 11 extras machinery end to end (config `[sources.X.extras]` → env-scrubbed launch → plugin): carries the scope-override globs
- Phase 11 e2e fixtures (`web/e2e/fixtures/plugin-binaries.ts`, `config-builder.ts`): two-tier fixture support for the external rehearsal spec
- Per-plugin committed guards: `readonly_test.go` / `outbound_hosts_test.go` pattern (fs plugin needs the read-only guard; it has no outbound hosts at all)

### Established Patterns
- Per-instance typed `match` blocks with plugin-declared `match_fields` (Phase 5) — fs plugin declares folder paths as its vocabulary
- Sync-time item validation: non-UNSPECIFIED fidelity + non-empty deep_link required per item (kernel skips-and-logs violations)
- Named health states that degrade honestly (mount unreachable ≠ empty folder — the WhatsApp/Signal precedent for never silently emptying a stream)

### Integration Points
- New plugin module `plugins/filesystem/` (own go.mod like siblings), built into `bin/plugins/` by `make plugins`
- New kernel endpoint for the xdg-open deep-link action (loopback-only, indexed-items-only) + a UI affordance on filesystem items in the DetailPane/stream
- External rehearsal: same binary copied to the external dir per Phase 11 D-09/D-10 conventions

</code_context>

<specifics>
## Specific Ideas

- Preview parity matters: the user explicitly flagged that inline PDF already works for paperless-ngx and expects the filesystem plugin to reuse that pipeline rather than treating PDF previews as future work.

</specifics>

<deferred>
## Deferred Ideas

- Server-side office-document conversion (LibreOffice headless or similar) for inline docx/xlsx previews — new heavyweight machinery; revisit if metadata + deep link proves insufficient.

### Reviewed Todos (not folded)
- "Signal schema-version verify-and-accept tooling" — keyword-noise match; Signal plugin tooling, unrelated to the filesystem source. Stays pending for a phase that touches the Signal plugin.

</deferred>

---

*Phase: 12-filesystem-source*
*Context gathered: 2026-08-13*
