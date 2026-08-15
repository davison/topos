# Phase 14: Google Drive Source, Built Out-of-Repo - Context

**Gathered:** 2026-08-15
**Status:** Ready for planning

<domain>
## Phase Boundary

A Google Drive folder becomes a topos source — delivered by a plugin developed in a **separate repository** against nothing but the four published inputs (`docs/plugin-contract.md`, `proto/topos/v1/plugin.proto`, the `sdk` Go module, `plugins/mock`), and installed through the Phase 11 external path with the untrusted badge. The user supplies their own Google OAuth client via env references (no secrets in config), authorizes once, and the source keeps syncing across kernel restarts without re-authorizing. Documents in the chosen Drive folder appear in webspace streams with previews — including Workspace-native Docs/Sheets/Slides via export — with deep links to the Drive web UI. Syncs after the first are incremental. Every place the published contract or mock falls short is written down as a contract gap. (SRC-05, SRC-06)

The phase is as much a proof of the third-party path as a new source: the out-of-repo constraint is a deliverable, not an inconvenience.

Two folded todos extend the kernel-side scope: the real-vs-dev config split (major, kernel) and native-tooltip suppression under chip popovers (minor, web).

</domain>

<decisions>
## Implementation Decisions

### OAuth flow & token home
- **D-01:** One-time authorization is a **standalone CLI auth command** (`topos-plugin-gdrive auth`) the user runs in a terminal: it opens the browser, runs the OAuth loopback redirect, and stores the token. In topos, an unauthorized source surfaces a **named health state** telling the user to run it. Auth is fully out-of-band — zero contract stretch (no plugin-provided URLs through kernel-composed UI text, which Phase 12 forbids).
- **D-02:** The refresh token persists in a **plugin-owned file** (token JSON, mode 0600, under `~/.local/share/topos-plugin-gdrive/`). Works headless under the Phase 11 scrubbed launch environment (no D-Bus/keyring dependency). "Where plugin private state lives" is undefined by the published contract — **record it as a contract gap** (the first entry in the gap log, found before a line of code).
- **D-03:** Setup docs **mandate publishing the user's OAuth app to production status** (unverified is fine for personal use — one-time "unverified app" consent warning). Testing status expires refresh tokens every 7 days, which would silently break success criterion 1; production-status refresh tokens live until revoked.
- **D-04:** The auth CLI reads the OAuth client ID/secret from **the same environment variable names the source config's `${VAR}` extras reference** (e.g. `GDRIVE_CLIENT_ID`/`GDRIVE_CLIENT_SECRET`), taken from the terminal shell. One env-var vocabulary end to end; docs say "export them, run auth".

### Out-of-repo development model
- **D-05:** The plugin lives in a **public GitHub repository** (`github.com/davison/topos-plugin-gdrive`), checked out as a sibling of the topos checkout. Module path matches the public home. Strongest SRC-06 proof (a real third party could find and build it) and the natural seed for PLUG-10 pull-by-URL later.
- **D-06:** Clean-room discipline is enforced by **separate Claude sessions + a rule file**: plugin work happens in the plugin repo in its own sessions; a CLAUDE.md there restricts references to the four published inputs and **requires logging every question they couldn't answer**. The gap log falls out of the discipline itself. (A literal clean-room is impossible — the same operator built the kernel — so the honest version is enforced process, not pretense.)
- **D-07:** Contract gaps are captured in **`CONTRACT-GAPS.md` in the plugin repo** (the clean-room sessions' only writable home). Phase 14 verification pulls the log back into topos: gaps fixable by **documentation alone are republished in-phase**; contract/proto changes become **backlog items for PLUG-11**. — **Reversibility:** costly — in-phase doc republishes edit `docs/plugin-contract.md`, the published contract third parties read; a wrong "clarification" is itself a contract change to walk back.
- **D-08:** **The plugin repo is its own GSD project**, bootstrapped from a PRD that cites only the four published inputs plus this phase's locked decisions (D-01..D-04, scope discretion). Its clean-room sessions plan and execute there. topos Phase 14's plans cover the kernel side: folded todos, external-path install + UAT, gap triage, and the doc republish. Plans authored by the topos-side planner must never inject kernel internals into the plugin repo's briefs — the PRD is the only hand-off document.

### Claude's Discretion
- **Drive scope & matching:** how an instance targets a folder (folder ID vs path), subfolder recursion, My Drive vs Shared Drives vs shortcuts, which files become items (document-ish allowlist per Phase 12 D-03 precedent), and the declared match vocabulary (folder paths mirroring Phase 12 D-05 is the obvious shape). Match values are exact literals, never globs (Phase 12).
- **Previews & export shapes:** per-Workspace-type export format (Docs/Sheets/Slides), reuse of the existing Fetch bytes+MIME pipeline for native files (PDF/images inline; office formats metadata + deep link, per Phase 12 D-04), honest handling of export size caps and API quota. Deep links to the Drive web UI at the honest fidelity tier.
- **Incremental sync mechanics:** changes.list page-token flow vs alternatives, sync cadence, first-sync full listing strategy (criterion 3 requires increments after the first).
- **Auth CLI ergonomics** beyond D-04 (flag names, output, token-file refresh/rotation handling) and the exact named health states for missing/expired/revoked auth (fail loudly by name, per project pattern).
- **Plugin repo bootstrap mechanics:** PRD wording, rule-file (CLAUDE.md) contents, CONTRACT-GAPS.md entry format, release/build recipe that produces the binary for the external plugins directory.
- **Folded-todo implementation:** `--config` flag / `TOPOS_CONFIG` env precedence and Makefile dev-config convention (todo's own solution sketch); tooltip suppression via title→aria-label with a component test (todo's own solution sketch).

### Folded Todos
- **"Separate the real (production) config from dev-work configs"** (`.planning/todos/pending/2026-08-14-separate-real-config-from-dev-configs.md`, kernel, major) — dev-built kernels (including GSD worktree builds) read the production `~/.config/topos/config.toml`, which bit during Phase 13 UAT (manifest gate refused all trusted plugins built by a different `make build`). Fits here: Phase 14 UAT runs dev kernels while the operator's real config starts carrying live OAuth credentials — the split is a precondition for safe UAT this phase, not a nice-to-have.
- **"Suppress native browser tooltips that duplicate/cover source-chip popovers"** (`.planning/todos/pending/2026-08-14-suppress-native-tooltips-under-chip-popovers.md`, web, minor) — `title`/`alt` attributes on chip elements render native tooltips over the app's own popovers. Small ride-along; 13-06 (the alternative home its note suggested) already executed without it.

</decisions>

<canonical_refs>
## Canonical References

**Downstream agents MUST read these before planning or implementing.**

### The four published inputs (the plugin repo's entire allowed surface — D-06)
- `docs/plugin-contract.md` — the published `topos.v1` contract; also the artifact D-07's in-phase gap fixes republish
- `proto/topos/v1/plugin.proto` — wire truth: Describe/Match/Fetch/Health, ContentVariant, LinkFidelity, ExtrasField
- `sdk/` — the published Go module (`github.com/davison/topos/sdk`) the plugin imports
- `plugins/mock/` — the reference plugin built from exactly these inputs (PLUG-05 proof)

### Prior locked decisions this phase inherits
- `.planning/phases/11-external-plugins-the-trust-boundary/11-CONTEXT.md` — extras map strings-only (D-13), env allowlist + `${VAR}` expansion designed for SRC-05's BYO credentials (D-14), Describe extras declaration (D-15), external dir/discovery/pin conventions (D-09/D-10, D-01..D-04)
- `.planning/phases/12-filesystem-source/12-CONTEXT.md` — preview shapes over Fetch bytes+MIME (D-04), folder-path match vocabulary (D-05), match values never globs, remove+add identity (D-02)
- `.planning/phases/13-per-item-curation-installable-app/13-CONTEXT.md` — refuse-to-load manifest gate and consent+pin as the only path for external binaries (D-12/D-13); the Drive plugin passes through exactly this flow
- `.planning/research/SUMMARY.md` — v1.1.0 research: recommended stack (`google.golang.org/api/drive/v3`, `golang.org/x/oauth2`), incremental-sync and export pitfalls, credential-distribution finding (BYO settled; D-03 resolves the 7-day expiry it flags)

### Requirements & conventions
- `.planning/REQUIREMENTS.md` — SRC-05/SRC-06 definitions; Out of Scope table (embedded shared OAuth client explicitly rejected)
- `docs/testing.md` — testing map; UI-touching work extends the Playwright e2e suite as definition of done (project convention)
- `docs/api.md` — HTTP API surface; "boolean only, never the value" env-presence discipline for any auth-state surfaces

### Folded todos (their files carry the full problem statements and solution sketches)
- `.planning/todos/pending/2026-08-14-separate-real-config-from-dev-configs.md`
- `.planning/todos/pending/2026-08-14-suppress-native-tooltips-under-chip-popovers.md`

</canonical_refs>

<code_context>
## Existing Code Insights

### Reusable Assets
- `kernel/supervisor/externalproof_test.go` + `testdata/external-plugin/` (11-04) — a genuinely separate Go module (`example.com/acme/topos-plugin-external-demo`) already proves the external path end to end: discovery, untrusted marking, extras passthrough with `${VAR}` expansion, env-allowlist visibility, tamper refusal. The Drive plugin is the *real* instance of this rehearsed shape; the e2e/UAT plans should mirror its assertions.
- Phase 11 consent interstitial + pin flow + re-pin machinery (`kernel/pluginhost/`, picker, chip menu) — the install path the Drive plugin walks; no new kernel trust work expected.
- Fetch bytes+MIME → media previewer pipeline (`kernel/httpapi/item.go`, `DetailPane.svelte`) — Drive PDF/image previews for free; rendered-content shapes for exported Docs follow the SilverBullet/kernel-rendition path.
- `plugins/filesystem/` — the freshest in-repo example of scope/classify/health/readonly structure — **but off-limits to the clean-room sessions (D-06)**; it informs only the topos-side plans (e.g. what UAT parity looks like).
- `web/e2e/` hermetic Playwright harness + Phase 11 two-tier fixtures (`plugin-binaries.ts`, `config-builder.ts`) — external-tier specs for the install/badge/UAT flow.
- `cmd/topos/main.go` config resolution + `pluginsDir()` relative resolution — the folded config-split todo's named touch points (its file lists `cmd/topos/main.go`, `config.example.toml`, `Makefile`, `docs/testing.md`).
- `web/src/lib/components/SourceChip.svelte` + `source-chip-tooltip.test.ts` — the tooltip-suppression todo's touch points.

### Established Patterns
- Fail loudly by name — missing/expired/revoked auth becomes named health states (WhatsApp's five-state vocabulary is the precedent); never a silently empty stream.
- Kernel-composed UI text only (`last_notice` is never plugin text) — why D-01 keeps the auth URL out of the UI entirely.
- Secrets are environment-only `${VAR}` references in config — D-04 extends the same names to the auth CLI's shell.
- Read-only by construction — the contract's four-RPC shape; prefer a read-only Drive scope (`drive.readonly`) as the credential-side second line of defense the contract doc recommends.
- Every plugin binary change touches `internal/audit` provenance checks — installing the Drive binary into the external dir interacts with the same machinery the 11-04 proof exercised.

### Integration Points
- External plugins directory (`~/.local/share/topos/plugins-external`, Phase 11 D-09) — where the built `topos-plugin-gdrive` binary lands for install.
- Source picker install catalog + untrusted add flow — where the Drive plugin appears and gets consented/pinned.
- `[sources.X.extras]` + `${VAR}` env refs — carries client ID/secret references and any scope-override keys to the plugin.
- Sibling checkout `~/projects/davison/topos-plugin-gdrive` — its own GSD project (D-08); hand-off is the PRD only.
- `docs/plugin-contract.md` republish — D-07's in-phase gap fixes; `CONTRACT-GAPS.md` in the plugin repo is the source feed.

</code_context>

<specifics>
## Specific Ideas

- The out-of-repo constraint is the point: the operator treats SRC-06's proof as a first-class deliverable — D-06's rule-file discipline and D-07's gap log are the mechanism, and the gap log is expected to have entries (D-02 already seeds one: the contract says nothing about where a plugin keeps private state).
- The operator confirmed bring-your-own OAuth stays the credential model (REQUIREMENTS.md Out of Scope already rejects an embedded shared client) — the open question from research/SUMMARY.md "Gaps to Address" #3 is now closed by D-01..D-04.
- Phase 14 UAT context motivated folding the config-split todo: dev kernels must stop reading the production config before UAT runs against live Google credentials.

</specifics>

<deferred>
## Deferred Ideas

- **Contract/proto changes surfaced by the gap log** — recorded in `CONTRACT-GAPS.md`, triaged in-phase, but wire-level fixes defer to the PLUG-11 developer-guide/certification phase (D-07).
- **Pull-by-URL install of the Drive plugin** — the public repo (D-05) seeds it, but distribution is PLUG-10 (backlog Phase 999.1).
- **OneDrive source (SRC-07)** — explicitly "same shape as Google Drive" in Future Requirements; the plugin repo's structure should be worth imitating but nothing here plans for it.

### Reviewed Todos (not folded)
- **"Signal schema-version verify-and-accept tooling"** (2026-08-05) — fourth consecutive keyword-noise match; Signal plugin maintenance tooling, unrelated to Drive. Stays pending for a phase that touches the Signal plugin.
- **"Plugin trust tier is directory-location, not provenance"** (2026-08-13) — **stale**: folded into Phase 13 and delivered there (13-CONTEXT D-12..D-16, build-provenance manifest + refuse-to-load gate). Archived at this phase's context commit rather than left pending.

</deferred>

---

*Phase: 14-google-drive-source-built-out-of-repo*
*Context gathered: 2026-08-15*
