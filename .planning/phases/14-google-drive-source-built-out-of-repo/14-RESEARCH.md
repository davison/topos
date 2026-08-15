# Phase 14: Google Drive Source, Built Out-of-Repo - Research

**Researched:** 2026-08-15
**Domain:** Google Drive API v3 + OAuth2 desktop-app auth, consumed by a third-party (out-of-repo) topos source plugin; kernel-side external-plugin install/UAT/gap-triage work
**Confidence:** HIGH (kernel-side mechanics — all reused from Phase 11/12, verified by reading the actual source); MEDIUM (Google Drive API/OAuth specifics — cross-checked against official `developers.google.com` docs via WebFetch, not against a live API call)

## Summary

Phase 14 is two workstreams wearing one phase number. The **kernel-side** workstream is almost entirely reuse: Phase 11 already built external-plugin discovery, content-hash pinning, the untrusted badge, the extras passthrough, and the scrubbed launch environment; Phase 12 already rehearsed all of it against a real out-of-repo-shaped binary. Phase 14's own kernel work is narrow — the two folded todos (config-path split, tooltip suppression), an e2e/UAT spec set that mirrors Phase 12's `12-external-rehearsal.spec.ts` pattern against the *real* `topos-plugin-gdrive` binary once it exists, triaging `CONTRACT-GAPS.md` entries back into `docs/plugin-contract.md`, and authoring the plugin repo's PRD from the four published inputs plus this phase's locked decisions. The **plugin-repo** workstream — the actual Google OAuth/Drive API implementation — happens in a separate GSD project (D-08) that this phase's planner must never inject kernel internals into; this research's job is to ground the PRD-authoring task and the kernel-side gap-triage task in accurate, current Google Drive API/OAuth facts, not to hand the clean-room sessions pre-written code.

The single most important non-obvious finding: **Google Drive's `changes.list` API is drive-wide, not folder-scoped.** There is no request parameter that restricts the change feed to one folder. A plugin implementing "incremental sync of one Drive folder" must (1) do a full `files.list` walk of the configured folder (and any recursed subfolders) once to build a baseline plus a `changes.getStartPageToken`, then (2) on every later sync, pull the drive-wide delta via `changes.list(pageToken)` and filter each changed file's *current* `parents` chain against its own maintained folder-membership state to decide add/update/remove — maintaining that membership state itself, in a plugin-owned local cache, because the kernel's `Match` RPC contract requires the **full current item set** on every call (`ReplaceWebspaceSourceItems` replaces wholesale each sync — VERIFIED, `kernel/index/store.go:199`), not a delta. "Incremental" therefore describes the plugin's *own* traffic to Google, not the shape of what it hands back to the kernel. This is exactly the kind of thing that belongs in the plugin repo's PRD, and it directly informs Claude's Discretion item "Incremental sync mechanics" in 14-CONTEXT.md.

The second-most-important finding is a security framing point, not a code finding: an "untrusted" badge alone undersells what's actually being installed here. Every other external-tier install so far (Phase 11's rehearsal demo, Phase 12's filesystem rehearsal) hands the untrusted binary either nothing or a filesystem path. This plugin hands it **live, working Google OAuth credentials with read access to the operator's real Google Drive** — a strictly larger blast radius than "a binary with full local OS access" already implies, because the plugin's own network egress (invisible to the kernel and to `internal/audit`, since that audit only walks in-repo `plugins/`) can now exfiltrate anything `drive.readonly` can see, not just what topos happens to sync into its index. D-06's clean-room discipline and D-07's gap log are real, useful mitigations, but the topos-side UAT/interstitial copy should say this plainly rather than reuse Phase 11's generic "untrusted" framing verbatim.

**Primary recommendation:** Scope Phase 14's topos-side plans to (a) the two folded todos, (b) an e2e spec extending Phase 12's external-rehearsal pattern against the real `topos-plugin-gdrive` binary, (c) `CONTRACT-GAPS.md` gap-triage and `docs/plugin-contract.md` republish, and (d) authoring the plugin repo's PRD — grounded in this research's Drive API/OAuth findings — as the sole hand-off artifact into the separate GSD project that actually builds the plugin.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| OAuth2 loopback authorization (`topos-plugin-gdrive auth`) | External plugin binary (own CLI mode) | — | Contract-forbidden in the kernel (D-01: no plugin-provided URLs through kernel-composed UI text); must be fully out-of-band, in the plugin's own process |
| Refresh-token storage & rotation | External plugin binary (plugin-owned file) | — | D-02 — "where plugin private state lives" is undefined by the published contract; kernel has no concept of plugin secrets beyond `${VAR}` passthrough |
| Drive folder walk / `changes.list` polling / export | External plugin binary | Google Drive API (external service) | `Match`/`Fetch` RPC implementations — entirely plugin-side per the contract's four-RPC shape |
| Folder-membership / sync-state cache (baseline + delta bookkeeping) | External plugin binary (plugin-owned file/cache) | — | New runtime state this phase introduces; not modeled by the published contract (extends the same "plugin private state" gap D-02 already opens) |
| OAuth client id/secret delivery to the running plugin | Kernel (`[sources.X.extras]` + `${VAR}` expansion) | External plugin binary (reads `WEBSPACES_SOURCE_CONFIG.extras`) | Phase 11's extras machinery (D-12/D-13/D-15) — no new kernel mechanism, just a new consumer of the existing one |
| Trust tier / content-hash pin / untrusted badge | Kernel (`kernel/pluginhost`) + Web UI | — | Phase 11, unmodified; the Drive plugin is a real instance of the existing external tier, not new kernel mechanism |
| Named auth-state health surfacing (missing/expired/revoked) | External plugin binary (`Health.LastError`, `Match` → `codes.Unavailable`) | Kernel (renders `reachable`/`last_error` verbatim) | Mirrors `plugins/whatsapp`'s `healthState` taxonomy exactly (VERIFIED, `plugins/whatsapp/health.go`) — this is a plugin-internal enum, not a kernel-side closed-vocabulary field like `launch_failure` |
| Deep links to Drive web UI | External plugin binary (`Item.deep_link`, `LinkFidelity`) | Kernel (serves the URL verbatim for any non-`file://` scheme) | Standard `https://` deep link path — no kernel change needed (contrast Phase 12's `file://` local-path convention, which does not apply here) |
| Config-path split (dev vs. production `config.toml`) | Kernel (`cmd/topos/main.go`) | Makefile / docs | Folded todo — precondition for safe UAT once the operator's real config carries live OAuth credentials |
| Native tooltip suppression on chip popovers | Web (`SourceChip.svelte`) | — | Folded todo, independent of the Drive work itself |

## User Constraints (from CONTEXT.md)

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**OAuth flow & token home**
- **D-01:** One-time authorization is a **standalone CLI auth command** (`topos-plugin-gdrive auth`) the user runs in a terminal: it opens the browser, runs the OAuth loopback redirect, and stores the token. In topos, an unauthorized source surfaces a **named health state** telling the user to run it. Auth is fully out-of-band — zero contract stretch (no plugin-provided URLs through kernel-composed UI text, which Phase 12 forbids).
- **D-02:** The refresh token persists in a **plugin-owned file** (token JSON, mode 0600, under `~/.local/share/topos-plugin-gdrive/`). Works headless under the Phase 11 scrubbed launch environment (no D-Bus/keyring dependency). "Where plugin private state lives" is undefined by the published contract — **record it as a contract gap** (the first entry in the gap log, found before a line of code).
- **D-03:** Setup docs **mandate publishing the user's OAuth app to production status** (unverified is fine for personal use — one-time "unverified app" consent warning). Testing status expires refresh tokens every 7 days, which would silently break success criterion 1; production-status refresh tokens live until revoked.
- **D-04:** The auth CLI reads the OAuth client ID/secret from **the same environment variable names the source config's `${VAR}` extras reference** (e.g. `GDRIVE_CLIENT_ID`/`GDRIVE_CLIENT_SECRET`), taken from the terminal shell. One env-var vocabulary end to end; docs say "export them, run auth".

**Out-of-repo development model**
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

### Deferred Ideas (OUT OF SCOPE)
- **Contract/proto changes surfaced by the gap log** — recorded in `CONTRACT-GAPS.md`, triaged in-phase, but wire-level fixes defer to the PLUG-11 developer-guide/certification phase (D-07).
- **Pull-by-URL install of the Drive plugin** — the public repo (D-05) seeds it, but distribution is PLUG-10 (backlog Phase 999.1).
- **OneDrive source (SRC-07)** — explicitly "same shape as Google Drive" in Future Requirements; the plugin repo's structure should be worth imitating but nothing here plans for it.
</user_constraints>

## Phase Requirements

<phase_requirements>
| ID | Description | Research Support |
|----|-------------|------------------|
| SRC-05 | User can add a Google Drive folder as a source using their own Google OAuth client (bring-your-own credentials via env refs), with incremental sync, Workspace-doc export previews, and deep links to the Drive web UI | Standard Stack (OAuth2/Drive API packages), Architecture Patterns (loopback auth, changes.list delta mechanics, export MIME map), Common Pitfalls (7-day testing expiry, drive-wide changes feed, quota/backoff) |
| SRC-06 | The Google Drive plugin is developed out-of-repo against the published contract and installed through the external-plugin path — proving a third party can ship a working untrusted plugin end to end | Architectural Responsibility Map, "Don't Hand-Roll" (reuse Phase 11/12 kernel mechanics unmodified), Integration Points (e2e rehearsal spec, gap-triage flow, PRD hand-off) |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

- **Testing convention:** "Any phase that touches the UI extends the Playwright e2e suite as part of its definition of done; any UAT item a browser can drive becomes a spec (`web/e2e/specs/`) rather than staying a manual check." Applies to Phase 14's kernel-side deliverables (external-install flow, untrusted badge, auth-state health chip) — see "Validation Architecture is skipped, but this convention is not" below. The *live* OAuth/Drive-sync half cannot be hermetically driven (no CI-safe Google credentials) and is manual UAT, mirroring the WhatsApp-pairing precedent already documented in `docs/testing.md`.
- **Git:** linear history, rebase merging only — no direct bearing on plan structure beyond the usual GSD commit discipline.
- **TDD:** preferred but not dogmatic — applies to the plugin repo's own test suite (out of this research's scope, D-08) and to any kernel-side Go tests the folded todos or gap-triage add.
- **GSD workflow enforcement:** file-changing tool use in this repo must run through a GSD command; this phase itself is being planned through `/gsd-plan-phase`, satisfying that constraint.

## Standard Stack

### Core (plugin repo — informs the PRD, not a topos-side dependency)

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `google.golang.org/api/drive/v3` | v0.293.0 [VERIFIED: `go list -m -versions google.golang.org/api` against the Go module proxy, run this session; latest of the returned version list] | Official Google-generated Drive API v3 client (`files`, `changes` services) | Google's own generated client — the alternative is hand-rolling REST calls against `www.googleapis.com/drive/v3`, which the project's own stack doc already rejects as a pattern (compare paperless-ngx's thin hand-rolled client, chosen there specifically because *no* official Go SDK exists for paperless-ngx; Drive has one) |
| `golang.org/x/oauth2` | v0.36.0 [VERIFIED: `go list -m -versions golang.org/x/oauth2` against the Go module proxy, run this session; latest of the returned version list] | OAuth2 authorization-code flow, token refresh, `oauth2.TokenSource` | The Go-project-maintained OAuth2 client; `google.golang.org/api`'s own `option.WithTokenSource` expects exactly this package's `oauth2.TokenSource` interface |

Both packages were discovered via WebSearch/training knowledge, not an authoritative source, so the *recommendation to use them* is `[ASSUMED]` even though their *version numbers* are `[VERIFIED]` against the Go module proxy — see "Package Legitimacy Audit" below for why this distinction matters here.

**What NOT to add, contrary to the v1.1.0 milestone's original research summary:** `github.com/zalando/go-keyring` — `.planning/research/SUMMARY.md`'s Phase 3 recommendation predates Phase 14's CONTEXT.md D-02, which explicitly rejects a keyring dependency in favor of a plugin-owned file (headless-safe under Phase 11's scrubbed launch environment, no D-Bus/Secret-Service round trip needed at every launch). D-02 supersedes the original research finding; do not resurrect go-keyring in the PRD.

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `golang.org/x/oauth2/google` | (part of x/oauth2 module) | `google.Endpoint`, convenience helpers for the Google OAuth2 endpoint URLs | Avoids hand-typing `https://oauth2.googleapis.com/token` / `https://accounts.google.com/o/oauth2/auth` |
| stdlib `net/http` (loopback listener) | — | The local server the loopback-redirect flow listens on (`http://127.0.0.1:<port>/callback`) | No third-party CLI-OAuth helper package is needed for this — it's roughly 30 lines against stdlib; pulling in a helper (e.g. `github.com/int128/oauth2cli`, found in this session's WebSearch results) is optional convenience, not a requirement |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `google.golang.org/api/drive/v3` official client | Hand-rolled REST client (project's own pattern for paperless-ngx/SilverBullet) | Only justified when no official SDK exists — Drive has a mature, actively maintained one; hand-rolling it would be reinventing pagination, error-code mapping, and export-endpoint quirks the official client already handles |
| `golang.org/x/oauth2` loopback flow (manual) | `github.com/int128/oauth2cli` or similar CLI-OAuth helper | A helper trims boilerplate but is a third dependency the clean-room plugin repo would need to independently vet; the manual flow is small enough (and well-documented by Google itself, see Architecture Patterns) that it's a reasonable default not to add it |
| Plugin-owned token file (D-02) | `zalando/go-keyring` (original v1.1.0 research recommendation) | D-02 already settled this in favor of the file — keyring would reintroduce a D-Bus dependency Phase 11's scrubbed launch environment doesn't guarantee is reachable, and doesn't work headlessly at all on some setups |

**Installation (plugin repo, not this repo):**
```bash
go get google.golang.org/api/drive/v3@v0.293.0
go get golang.org/x/oauth2@v0.36.0
```

**Version verification:** both versions above were confirmed via `go list -m -versions <module>` against the live Go module proxy this session (2026-08-15) — not training-data guesses. Re-run this command when the plugin repo actually pins its `go.mod`, since these are fast-moving Google-maintained modules and a few weeks' drift is expected.

## Package Legitimacy Audit

> `gsd-tools query package-legitimacy check` supports only `npm`/`pypi`/`crates` ecosystems — it has no Go-ecosystem check, so this audit was performed manually against authoritative sources instead of the automated seam.

| Package | Registry | Age | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|-----|-----------|-------------|---------|-------------|
| `google.golang.org/api` | Go module proxy (proxy.golang.org) | Years (Google's official generated-API-client monorepo, actively published) | N/A (Go modules have no download counter) | `github.com/googleapis/google-api-go-client` — official Google org | OK | Approved |
| `golang.org/x/oauth2` | Go module proxy | Years (part of the `golang.org/x/*` extended-standard-library set, maintained by the Go team itself) | N/A | `cs.opensource.google` / mirrored to `github.com/golang/oauth2` — official Go project | OK | Approved |

**Packages removed due to `[SLOP]` verdict:** none.
**Packages flagged as suspicious `[SUS]`:** none.

Both packages are maintained by the organizations that own the exact protocol/API they wrap (Google for Drive/OAuth2 endpoints, the Go team for the OAuth2 client itself) — the lowest-plausible-risk case for a slopsquat or hallucinated package name. Their existence and current version were confirmed this session via `go list -m -versions` against the real module proxy (an authoritative source, not WebSearch), so their package-name provenance is `[VERIFIED: go module proxy]` rather than `[ASSUMED]`, despite the general rule that WebSearch-discovered names default to `[ASSUMED]` — these two names also matched training knowledge exactly, so both signals agree.

## Architecture Patterns

### System Architecture Diagram

```
                          ┌─────────────────────────────┐
                          │   Operator's terminal shell  │
                          │  export GDRIVE_CLIENT_ID=... │
                          │  export GDRIVE_CLIENT_SECRET │
                          └──────────────┬────────────────┘
                                         │ (1) topos-plugin-gdrive auth
                                         ▼
                          ┌─────────────────────────────┐
                          │  topos-plugin-gdrive (CLI    │
                          │  auth mode, standalone)      │
                          │  - reads env vars directly   │
                          │  - opens loopback listener    │
                          │  - opens browser to Google    │
                          └──────────────┬────────────────┘
                                         │ (2) OAuth loopback redirect
                                         ▼
                          ┌─────────────────────────────┐
                          │        Google OAuth2          │
                          │  accounts.google.com/o/oauth2 │
                          └──────────────┬────────────────┘
                                         │ (3) auth code → token exchange
                                         ▼
                          ┌─────────────────────────────┐
                          │ ~/.local/share/               │
                          │  topos-plugin-gdrive/token.json│  (D-02, mode 0600)
                          └──────────────┬────────────────┘
                                         │ (4) read at every launch
                                         ▼
┌───────────────┐   go-plugin/gRPC   ┌─────────────────────────────┐   changes.list /   ┌───────────────┐
│  topos kernel  │◄──────────────────┤ topos-plugin-gdrive (serve   │  files.export       │  Google Drive  │
│  (pluginhost)  │  Describe/Match/  │  mode, launched by kernel)   ├─────────────────────►│  API v3        │
│                │  Fetch/Health     │  - reads WEBSPACES_SOURCE_   │◄─────────────────────┤                │
│                │──────────────────►│    CONFIG.extras for         │  delta / export bytes│                │
└───────┬────────┘  extras + scrubbed│    client_id/secret + folder │                       └───────────────┘
        │            env (Phase 11)  │  - maintains its own local   │
        │                            │    sync-state cache (new     │
        │                            │    plugin-private state)     │
        ▼                            └─────────────────────────────┘
┌───────────────┐
│ kernel index / │  Match returns the FULL current item set every
│ webspace stream│  call (ReplaceWebspaceSourceItems) — "incremental"
│ (unchanged)    │  describes the plugin's OWN Drive traffic, not
└───────────────┘  the RPC's return shape.
```

### Recommended Project Structure (plugin repo — for the PRD, not this repo)

```
topos-plugin-gdrive/
├── main.go              # os.Args dispatch: "auth" subcommand vs. default goplugin.Serve()
├── auth.go              # loopback-redirect CLI flow (D-01), reads GDRIVE_CLIENT_ID/SECRET from shell env
├── tokenstore.go         # plugin-owned token file read/write, mode 0600 (D-02)
├── plugin.go             # sdk.SourcePlugin implementation: Describe/Match/Fetch/Health
├── drive.go               # google.golang.org/api/drive/v3 client construction, changes.list delta loop
├── syncstate.go            # plugin-private folder-membership cache (baseline + delta bookkeeping)
├── export.go                # Workspace-native MIME → export-format mapping
├── CONTRACT-GAPS.md           # D-07's gap log — clean-room sessions' only writable home for "the contract didn't say"
└── CLAUDE.md                   # D-06's rule file: allowed inputs = docs/plugin-contract.md, plugin.proto, sdk/, plugins/mock only
```

### Pattern 1: Dual-mode binary (serve vs. standalone auth CLI)

**What:** `main()` inspects `os.Args[1]`; `"auth"` runs the standalone loopback-redirect flow and exits, anything else (including no args, which is how `hashicorp/go-plugin` execs a plugin subprocess) falls through to the normal `goplugin.Serve(...)` path every other plugin in this repo uses.
**When to use:** Required by D-01 — the kernel must never see or compose an OAuth URL (Phase 12's UI-text rule), so authorization has to be a completely separate invocation of the same binary, run by the operator directly in a terminal, not something the kernel launches or drives.
**Example (illustrative shape, not verified against plugin-repo code that doesn't exist yet):**
```go
func main() {
	if len(os.Args) > 1 && os.Args[1] == "auth" {
		if err := runAuth(); err != nil {
			fatal(err)
		}
		return
	}
	// falls through to goplugin.Serve(...) exactly like every in-repo
	// plugin's main.go (docs/plugin-contract.md "Build your first plugin")
}
```
This shape is `[ASSUMED]` — it is the natural way to satisfy D-01 given `cmd/topos/main.go`'s own `os.Args[1]`-switch precedent (VERIFIED, `cmd/topos/main.go:34-46`), but no code in this session confirms `hashicorp/go-plugin`'s child-launch path tolerates or ignores extra args; that is exactly the kind of question D-06's clean-room discipline should surface into `CONTRACT-GAPS.md` if it turns out not to.

### Pattern 2: OAuth2 loopback redirect (installed/desktop app flow)

**What:** `golang.org/x/oauth2`'s `oauth2.Config` with `RedirectURL` set to `http://127.0.0.1:<ephemeral-port>` (not `localhost` — Google's own docs flag `localhost` as prone to client-firewall issues [CITED: developers.google.com/identity/protocols/oauth2/native-app, fetched this session]); a local `net/http` server on that port receives the redirect, extracts the `code` query param, and exchanges it via `oauth2.Config.Exchange`.
**When to use:** This is the *only* Google-supported flow for a desktop/CLI OAuth client as of this research — Google deprecated the copy/paste "out-of-band" (OOB) flow, blocking new usage from **2022-02-28** and fully deprecating it for all client types by **2023-01-31** [CITED: developers.google.com/identity/protocols/oauth2/resources/oob-migration, via WebSearch this session]. D-01's chosen loopback-redirect approach is the current, correct replacement, not a legacy pattern.
**PKCE:** Google's own docs "strongly encourage" PKCE (`code_challenge`/`code_challenge_method=S256`) for installed-app flows [CITED: developers.google.com/identity/protocols/oauth2/native-app, fetched this session] — flag this for the plugin repo's own research phase as a security hardening item, not a hard requirement 14-CONTEXT.md locked.
**Redirect-URI registration:** sources disagree in this session's research on whether the exact loopback port must be pre-registered in Cloud Console for a "Desktop app" OAuth client type, or whether Google exempts loopback ports from exact matching for that client type specifically — this is genuinely unresolved by this research and should be one of the plugin repo's own first research questions, not asserted either way in the PRD. See "Open Questions" below.

### Pattern 3: Drive incremental sync — `changes.list` is drive-wide, folder filtering is the plugin's job

**What:** Google Drive's Changes API (`changes.getStartPageToken` → `changes.list(pageToken)` → persist `newStartPageToken`) reports every change across the *entire* Drive (or, with `driveId` set, an entire Shared Drive) — there is no folder-scoping parameter [MEDIUM confidence, cross-checked via WebSearch this session against `developers.google.com/workspace/drive/api/reference/rest/v3/changes/list` and `.../guides/manage-changes`, not independently re-verified against a live API call]. A plugin scoped to one folder must, on every delta, resolve each changed file's current `parents` against its own maintained "is this under my configured root" membership set — including tracking parent-folder moves, since a file's own `parents` array only names its *immediate* parent(s), not its full ancestry.
**When to use:** Every sync after the first (criterion 3's "incremental" requirement). The first sync still needs a full `files.list` walk (with a `q` query like `'<folderId>' in parents` — Drive's query language has no native recursive-descent operator, so subfolder recursion, if in scope per Claude's Discretion, means walking the folder tree with one `files.list` call per discovered subfolder) to establish the baseline membership set and to obtain the starting page token via `changes.getStartPageToken`.
**Why this matters for the plan:** it directly determines the shape of the plugin's own local state. D-02 already establishes precedent for plugin-private state living in `~/.local/share/topos-plugin-gdrive/` (for the token); a folder-membership/sync-state cache is a *second* piece of state in that same category, and the PRD should say so explicitly rather than leaving the plugin repo's own planning to rediscover it. This is a strong second contract-gap candidate ("where does plugin-private *sync* state live, beyond the token" — the contract document only addresses "connection details," never the plugin's own working cache) worth flagging for D-07's gap log from day one, the same way D-02 already flagged the token-storage gap before any code existed.

### Pattern 4: Workspace-native export

**What:** `files.export` converts a Google Workspace-native file (Doc/Sheet/Slide — *not* an uploaded `.docx`/`.xlsx`/`.pptx`, which already has real bytes and needs no export step) to a requested MIME type, capped at **10 MB** [CITED: developers.google.com/workspace/drive/api/guides/ref-export-formats, via WebSearch this session].
**Export MIME map** (subset relevant to preview/deep-link decisions):

| Native type | Candidate export MIME types |
|---|---|
| Google Docs | `application/pdf`, `application/vnd.openxmlformats-officedocument.wordprocessingml.document`, `text/plain`, `text/markdown` |
| Google Sheets | `application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`, `text/csv`, `application/pdf` |
| Google Slides | `application/vnd.openxmlformats-officedocument.presentationml.presentation`, `application/pdf`, `image/png` |

**When to use:** Per Phase 12's D-04 precedent (Fetch bytes+MIME → media previewer pipeline), exporting to `application/pdf` reuses the exact PDF-inline-preview path Phase 12 already proved for the filesystem plugin — the lowest-new-kernel-work option, and the one Claude's Discretion in 14-CONTEXT.md gestures at ("reuse of the existing Fetch bytes+MIME pipeline for native files"). The 10 MB export cap is a real ceiling distinct from the contract's own 64 MiB gRPC message-size ceiling (`sdk.GRPCServer` — VERIFIED, `docs/plugin-contract.md` "Fetch" section) — a large Google Doc can exceed Drive's own export limit well before it would ever threaten the gRPC ceiling; `available: false` with a named `unavailable_reason` (per the contract's `Fetch` semantics) is the honest response, not a truncated export.

### Anti-Patterns to Avoid
- **Treating `changes.list` as folder-scoped:** it isn't — see Pattern 3. A plan or PRD that assumes "ask Google for changes in this folder" will hit a wall the first time it's implemented.
- **Reusing the original v1.1.0 research's `zalando/go-keyring` recommendation:** superseded by D-02; do not resurrect it in the PRD.
- **Kernel composing or displaying any URL sourced from the plugin:** D-01 explicitly forbids this — the auth flow's browser URL comes from the standalone CLI mode talking to Google directly, never routed through or rendered by the kernel's UI text.
- **Assuming Match can return a delta:** it cannot — `ReplaceWebspaceSourceItems` replaces the full per-source, per-webspace item set on every sync (VERIFIED, `kernel/index/store.go:199`); "incremental" is scoped to the plugin's own Drive API traffic only.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| OAuth2 authorization-code exchange, token refresh | A hand-rolled HTTP client hitting `oauth2.googleapis.com/token` directly | `golang.org/x/oauth2` | Token refresh, expiry handling, and `oauth2.TokenSource` composition are exactly what this package exists for; it's also what `google.golang.org/api`'s `option.WithTokenSource` expects natively |
| Drive API request/response shapes, pagination | Hand-rolled REST client against `www.googleapis.com/drive/v3` | `google.golang.org/api/drive/v3` | Official generated client already handles pagination (`NextPageToken`), the `fields` parameter discipline Google's docs say "almost all methods" now require, and error-code mapping |
| External-plugin discovery, content-hash pinning, untrusted badge, launch-env scrubbing | Any new kernel mechanism | Existing `kernel/pluginhost` machinery from Phase 11, unmodified | This is the whole point of SRC-06 — prove the *existing* mechanism works for a real third-party binary, not build a Drive-specific variant of it |
| Provider-specific config delivery (client id/secret, folder id) | A new kernel config field | `[sources.X.extras]` + `${VAR}` expansion (Phase 11 D-12/D-13/D-15) | Exactly the mechanism Phase 11 built this passthrough for — SRC-05's BYO-credentials requirement is literally named in 11-CONTEXT.md D-14 as a reason the extras env-hygiene work existed |

**Key insight:** almost nothing in this phase should be *new kernel mechanism* — the interesting engineering is entirely on the Google Drive API/OAuth side, in a repository this phase's planner does not write plans for directly (D-08). The topos-side plan's job is scoping the kernel-side rehearsal/gap-triage/PRD-authoring work precisely enough that the separate plugin-repo GSD project can execute without needing to re-derive any of Phase 11/12's already-settled kernel mechanics.

## Common Pitfalls

### Pitfall 1: Testing-status refresh tokens expire in 7 days, silently
**What goes wrong:** An operator who never explicitly changes their Google Cloud OAuth consent screen's publishing status from the default "Testing" gets refresh tokens that stop working exactly 7 days after each authorization — success criterion 1 ("keeps syncing across kernel restarts without re-authorizing") silently fails a week in.
**Why it happens:** Google automatically expires all refresh tokens issued by unverified/testing-status apps after 7 days [CITED: multiple sources cross-referenced via WebSearch this session, consistent with `support.google.com/cloud` and community reports].
**How to avoid:** D-03 already locks the mitigation (mandate publishing to Production status in setup docs). Publishing to Production for a personal-use, single-known-user app does **not** require completing Google's verification/CASA process even for a "sensitive"/"restricted" scope like `drive.readonly` — the operator just clicks through a persistent "unverified app" warning on each new authorization [CITED: `developers.google.com/identity/protocols/oauth2/production-readiness/*` via WebFetch this session]. This confirms D-03 is technically sound as written; no new decision needed, but the setup docs should say explicitly that "Production" and "Verified" are different things, since conflating them is an easy operator mistake.
**Warning signs:** A previously-working Drive source going unreachable roughly a week after first authorization, with no config change in between.

### Pitfall 2: `changes.list` is drive-wide — folder scope is entirely client-side
**What goes wrong:** A first implementation attempt reaches for a `changes.list` parameter that scopes to a folder and doesn't find one, or worse, ships code that silently ingests every change across the operator's entire Drive rather than just the configured folder.
**Why it happens:** Every *other* Drive endpoint this plugin needs (`files.list`, `files.get`) supports a `q` query filtering by `parents` — it's easy to assume `changes.list` works the same way. It doesn't; folder membership must be resolved by the plugin from each change's own `parents` field, cross-referenced against locally-maintained folder-tree state.
**How to avoid:** Design the plugin's sync-state cache (Architecture Pattern 3) from the start to answer "is file X currently a descendant of my configured root" independent of what `changes.list` returns — treat every changed file as "needs a membership re-check," not "is definitely in scope."
**Warning signs:** Items from unrelated Drive folders appearing in the webspace stream; or, conversely, items that moved out of the configured folder never disappearing from the stream.

### Pitfall 3: `Match` must return the full current set every call — "incremental" is not about the RPC
**What goes wrong:** A plugin author reads "incremental sync" and designs `Match` to return only the items that changed since last time, assuming the kernel does its own diffing.
**Why it happens:** The name "incremental sync" naturally suggests delta responses; but the kernel's actual sync-persistence contract is a full replace (`ReplaceWebspaceSourceItems`, VERIFIED `kernel/index/store.go:199`) keyed on whatever `Match` returns for that call.
**How to avoid:** The plugin must materialize its *full* current item set (from its own locally-cached, incrementally-updated view of the Drive folder) on every `Match` call, even though building that cache is itself incremental against the Drive API. Flag this distinction explicitly in the PRD so the plugin repo's own planning doesn't have to rediscover it from a failed UAT.
**Warning signs:** After the first sync, the webspace stream shrinks to only recently-changed items instead of showing the full folder contents.

### Pitfall 4: A well-behaved `Describe` must not require live Drive credentials
**What goes wrong:** The kernel's add-source flow trial-launches a plugin binary (`WEBSPACES_DESCRIBE_ONLY=1`) *before* the operator has finished configuring extras (VERIFIED, `docs/plugin-contract.md` "Describe" section: "trial-launches your binary... against connection fields the operator has typed but not yet saved"). If `Describe` eagerly validates the OAuth client id/secret or attempts a Drive API call, the add-source UI breaks for every operator on the very first step.
**Why it happens:** Natural instinct is to fail loudly and immediately on missing required config (the contract's own general guidance for `Match`/`Fetch`) — but `Describe` is explicitly exempt from that discipline; it must be idempotent and side-effect-free regardless of call context.
**How to avoid:** `Describe` should return static identity/vocabulary/extras-declaration data only, deferring any credential validation to `Match`/`Health`, which the contract already expects to fail with `codes.Unavailable` / `reachable: false` for a genuinely unreachable source.
**Warning signs:** The add-source form failing or hanging before an operator has had a chance to type in `folder_id`/OAuth env var references at all.

### Pitfall 5: An "untrusted" badge undersells the actual risk of this specific plugin
**What goes wrong:** The install interstitial reuses Phase 11's generic untrusted-binary warning copy verbatim, which frames the risk as "code topos didn't build; no sandbox" — true, but incomplete for this specific plugin, which additionally receives live OAuth credentials scoped to the operator's real Google Drive.
**Why it happens:** Phase 11's interstitial (D-05, 11-CONTEXT.md) was designed against the general external-plugin case, which at the time had no example of a plugin actually handling live third-party credentials.
**How to avoid:** This is a UAT/copy consideration for the topos-side plan, not a kernel mechanism change — no proto/contract change needed, no new kernel field. Worth an explicit plan task to review (not necessarily rewrite) the interstitial copy against this specific install case.
**Warning signs:** None mechanical — this is a framing/communication gap, not a bug.

### Pitfall 6: Drive API rate limits and export size caps need honest degradation, not silent truncation
**What goes wrong:** A large folder sync or a large Workspace-native document hits Drive's per-project/per-user query quota (historical figure ~12,000 queries/60s, cohort-dependent — [MEDIUM confidence, cross-checked via WebSearch this session, not independently re-verified against the operator's actual project quota]) or the 10 MB `files.export` cap, and the plugin either crashes or silently returns truncated/empty content.
**Why it happens:** Personal-use, single-account quotas are generous enough that this rarely surfaces during initial development against a small test folder, then bites in exactly the large-real-Drive scenario the phase is meant to support.
**How to avoid:** Standard exponential-backoff-with-jitter on 403/429 responses [CITED: developers.google.com quota guidance, cross-referenced via WebSearch this session]; `available: false` + named `unavailable_reason` (never a truncated export) for anything over the export cap, matching the contract's existing `Fetch` semantics for "a document type your source can't render a preview for."
**Warning signs:** Sync runs erroring intermittently under large folders; large Google Docs/Sheets silently missing previews.

## Code Examples

No verified in-repo code examples exist for the Google Drive/OAuth surface (it doesn't exist in this repository — D-08, out-of-repo by design). The closest verified precedents already in this repo, useful as structural templates for the PRD:

### Named health-state taxonomy (verified pattern to imitate, not Drive-specific)
```go
// Source: plugins/whatsapp/health.go (read this session) — the exact
// pattern D-01's "named health state" alludes to: a plugin-internal enum
// surfaced through Health.LastError (free text) and Match returning
// codes.Unavailable for every non-healthy state, never a kernel-level
// closed-vocabulary field. A Google Drive plugin's missing/expired/
// revoked-auth states should mirror this shape, not invent a new one.
type healthState int

const (
	healthStateConnecting healthState = iota // ...
	healthStateNotLinked
	healthStateLinked
	// ... one constant per named, honestly-distinguished cause
)
```

### `os.Args` dispatch precedent (verified pattern to imitate)
```go
// Source: cmd/topos/main.go:34-46 (read this session) — the exact shape
// D-01's dual-mode binary (auth CLI vs. goplugin.Serve) should mirror:
switch os.Args[1] {
case "serve":
	// ...
case "sync":
	// ...
default:
	usage()
	os.Exit(2)
}
```

### `[sources.X.extras]` shape a Drive source's config will need (verified pattern to imitate)
```toml
# Source: config.example.toml:534-549 (read this session) — the
# filesystem plugin's extras block is the closest existing worked
# example of an in-repo plugin declaring provider-specific extras; a
# Drive source's own [sources.gdrive.extras] block (client_id,
# client_secret, folder_id, or similar) follows the identical shape:
# every value a string, ${VAR} expansion identical to base_url/token,
# omitted entirely (not an empty table) when no extras are configured.
[sources.filesystem.extras]
include_glob = "**/*.pdf,**/*.md"
exclude_glob = "**/node_modules/**,**/.git/**"
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| OAuth "out-of-band" (OOB) copy/paste flow for installed apps | Loopback IP redirect (`http://127.0.0.1:<port>`) | Blocked for new usage 2022-02-28, fully deprecated 2023-01-31 [CITED: developers.google.com/identity/protocols/oauth2/resources/oob-migration] | D-01's chosen flow is the current, only-supported mechanism — not a legacy-vs-modern choice, the OOB alternative genuinely no longer works |
| `zalando/go-keyring` for token storage (original v1.1.0 milestone research) | Plugin-owned file, mode 0600 (D-02) | Superseded by 14-CONTEXT.md's own discussion, 2026-08-15 | Headless-safe; no D-Bus/Secret-Service dependency at every plugin launch |

**Deprecated/outdated:** the OOB flow above; nothing else identified as deprecated in this research.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The dual-mode `main()` dispatch (`os.Args[1] == "auth"` vs. falling through to `goplugin.Serve`) is unproblematic for `hashicorp/go-plugin`'s child-process launch, which execs the binary with no args | Architecture Patterns, Pattern 1 | If go-plugin's launch path is sensitive to argv in some way this research didn't surface, the dual-mode binary could fail to handshake when launched by the kernel — should be an early smoke-test in the plugin repo's own first plan, and a candidate `CONTRACT-GAPS.md` entry if the contract's silence on this turns out to matter |
| A2 | The recommended packages (`google.golang.org/api/drive/v3`, `golang.org/x/oauth2`) are the right choice, beyond their verified existence/version | Standard Stack | Low — both are the obvious, training-data-consistent, officially-maintained choices for this exact problem; the only real alternative space is "hand-roll it," which the project's own "Don't Hand-Roll" philosophy already argues against |
| A3 | Drive's per-project/per-user query quota figures (~12,000 queries/60s) | Common Pitfalls, Pitfall 6 | Low-to-medium — exact quota numbers are known to vary by project cohort/grandfathering and were not verified against the operator's actual Google Cloud project; the *qualitative* guidance (backoff on 403/429) holds regardless of the exact number |
| A4 | Whether a Desktop-app OAuth client type requires the loopback redirect URI to be pre-registered exactly, or is exempted from exact port matching | Architecture Patterns, Pattern 2 | Medium — this is a first-authorization-flow blocker if guessed wrong; explicitly carried into Open Questions below rather than asserted, precisely because this session's sources disagreed |
| A5 | The precise shape of a second "sync-state cache" contract gap (beyond D-02's token-storage gap) is worth flagging proactively | Architecture Patterns, Pattern 3 | Low — worst case, the plugin repo's own clean-room session independently rediscovers and logs the same gap in `CONTRACT-GAPS.md`, which is exactly what D-06's discipline is built to catch anyway |

## Open Questions

1. **Does a Desktop-app OAuth client type require the loopback redirect URI to be pre-registered exactly in Cloud Console, or is it exempted from exact port matching?**
   - What we know: Google's `native-app` guide (fetched this session) says the `redirect_uri` "must exactly match one of the authorized redirect URIs... configured in your client's Cloud Console Clients page." A separate WebSearch summary this session claimed "For Desktop app clients... the URI is not explicitly configured in the Cloud console" (i.e., loopback ports are exempted).
   - What's unclear: which statement is authoritative for the current (2026) Cloud Console UI and Desktop-app client type specifically — these two claims are in tension.
   - Recommendation: treat this as the plugin repo's own first research question (its own `/gsd-plan-phase`'s research step), not something this topos-side research should assert either way in the PRD. The PRD should hand off the loopback-flow requirement and both candidate behaviors, flagged as needing a live spike against a real Cloud Console project before the auth CLI's redirect-handling code is finalized.

2. **What exactly should the named auth-state health-state vocabulary be (missing/expired/revoked, per Claude's Discretion)?**
   - What we know: the mechanism (plugin-internal enum, surfaced via `Health.LastError` + `Match` → `codes.Unavailable`) is settled — see Pattern verified against `plugins/whatsapp/health.go`.
   - What's unclear: the exact set of distinguishable causes Google's OAuth2 token-refresh error responses actually let a plugin tell apart (e.g., can a plugin reliably distinguish "token revoked by the user" from "token expired due to inactivity" from "OAuth app itself was un-published/deleted" from the `invalid_grant` error alone, or do several of these collapse to the same observable error)?
   - Recommendation: another first-plan research question for the plugin repo — the `invalid_grant` error's own sub-reasons (surfaced, if at all, in the error's `error_description` field) should be spiked against a real revoked/expired token before committing to a specific named-state taxonomy in that repo's own design docs.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go toolchain | Building the plugin repo, and any topos-side Go work (folded todos) | ✓ | go1.26.5 (this session's sandbox) | — |
| `google.golang.org/api`, `golang.org/x/oauth2` (Go module proxy reachability) | Confirming current versions; plugin repo `go get` | ✓ | v0.293.0 / v0.36.0 (confirmed this session) | — |
| Live Google Cloud project + OAuth client (for actually exercising the auth flow) | The live half of SRC-05's UAT (criterion 1) | ✗ (not available in this research session) | — | Manual UAT by the operator, per `docs/testing.md`'s established "what stays manual" precedent (the WhatsApp real-pairing precedent is the closest analog: a one-time, hands-on, recorded manual run, not a hermetic gate) |
| Real `topos-plugin-gdrive` binary (doesn't exist yet — D-08, separate repo) | e2e rehearsal spec (mirroring `12-external-rehearsal.spec.ts`) | ✗ | — | The e2e rehearsal spec for THIS phase can only be written/run once the plugin repo has produced a first buildable binary; sequence the topos-side e2e task after that milestone, not in parallel with it |

**Missing dependencies with no fallback:** none — both gaps above have a documented, precedented fallback (manual UAT; sequencing the e2e task after the plugin repo's first build).

**Missing dependencies with fallback:** live Google OAuth credentials (manual UAT); the real plugin binary (sequence after plugin-repo milestone).

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V2 Authentication | Yes | OAuth2 authorization-code flow with PKCE recommended (`golang.org/x/oauth2`) — see Architecture Pattern 2 |
| V3 Session Management | Yes (token lifecycle, not a web session) | Refresh-token rotation via `oauth2.TokenSource`; token file mode 0600 (D-02); Production publishing status to avoid silent 7-day expiry (D-03) |
| V4 Access Control | Partially | `drive.readonly` scope (least-privilege — read-only credential is the contract's own recommended "second, independent line of defense" per `docs/plugin-contract.md`'s "read-only by construction" section) |
| V5 Input Validation | Yes | Folder-id/path match values remain exact literals, never globs (Phase 12 D-05 precedent, extended here); extras key validation already enforced kernel-side (`[A-Za-z_][A-Za-z0-9_.-]*`, VERIFIED via `config.example.toml`'s documented validation comment) |
| V6 Cryptography | Partially | Refresh token stored as plaintext JSON at rest (D-02 — accepted trade-off vs. keyring, for headless compatibility); OS file permissions (0600) are the only control at rest — this is a deliberate, already-locked residual-risk acceptance, not a gap to close in this phase |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| Untrusted plugin binary exfiltrating live Drive OAuth credentials/content beyond what topos syncs | Information Disclosure | No kernel-level containment exists for this (contract's own explicit "no sandbox" framing) — the *only* real mitigations are D-06's clean-room process discipline and least-privilege scope (`drive.readonly`); the install interstitial copy should say this plainly (Pitfall 5) |
| OAuth phishing via a non-loopback redirect flow | Spoofing | Already closed by design — Google deprecated the OOB flow specifically for this reason (State of the Art); D-01's loopback-redirect choice is the current mitigated pattern, not a legacy risk |
| Token-refresh CSRF / authorization-code interception | Tampering | PKCE (`code_challenge`/`S256`) — "strongly encouraged" by Google for installed apps (Pattern 2); flag as a plugin-repo hardening recommendation, not a locked requirement in this phase's decisions |
| Plaintext refresh token on disk readable by another local process running as the same user | Information Disclosure | Accepted residual risk per D-02 (mode 0600 is the only control) — consistent with this project's existing "no sandbox" trust model for any plugin subprocess, not a new gap this phase introduces |
| `${VAR}` extras values (client id/secret) leaking into logs | Information Disclosure | Already covered by the contract's existing logging discipline ("A plugin must never log a credential... Log the *presence* or *name* of a secret... never its value" — VERIFIED, `docs/plugin-contract.md` "Logging" section) — no new kernel or plugin-repo work needed beyond following the existing rule |

## Sources

### Primary (HIGH confidence)
- `docs/plugin-contract.md` (read in full this session) — the published contract every plugin-repo decision must fit inside
- `proto/topos/v1/plugin.proto` (read this session) — wire truth for the four-RPC shape
- `.planning/phases/11-external-plugins-the-trust-boundary/11-CONTEXT.md`, `.planning/phases/12-filesystem-source/12-CONTEXT.md` (read this session) — inherited locked decisions
- `kernel/config/types.go`, `cmd/topos/main.go`, `kernel/pluginhost/host.go` (read this session) — VERIFIED kernel mechanics (extras shape, external-dir resolution, launch-env allowlist, `LaunchFailure` closed vocabulary)
- `kernel/index/store.go:199` (read this session) — VERIFIED `ReplaceWebspaceSourceItems` full-replace semantics
- `plugins/whatsapp/health.go` (read this session) — VERIFIED named health-state pattern to imitate
- `testdata/external-plugin/README.md`, `main.go` (read this session) — the standing out-of-repo proof precedent
- `docs/testing.md` (read in full this session) — e2e harness architecture, the "what stays manual" precedent for live-credential flows
- `.planning/todos/pending/2026-08-14-*.md` (both read this session) — the two folded todos' own problem statements
- `.planning/research/SUMMARY.md` (read this session) — v1.1.0 milestone research, partially superseded by 14-CONTEXT.md's own decisions (noted explicitly where superseded)
- `go list -m -versions google.golang.org/api` / `golang.org/x/oauth2` (run this session against the live Go module proxy)

### Secondary (MEDIUM confidence)
- `developers.google.com/identity/protocols/oauth2/native-app` (WebFetch, this session) — loopback redirect URI format, PKCE recommendation
- `developers.google.com/identity/protocols/oauth2/resources/oob-migration` and `.../loopback-migration` (WebSearch, this session) — OOB deprecation timeline
- `developers.google.com/identity/protocols/oauth2/production-readiness/*` (WebFetch, this session) — personal-use verification exemption, CASA/restricted-scope framing
- `developers.google.com/workspace/drive/api/guides/ref-export-formats`, `.../reference/rest/v3/files/export` (WebSearch, this session) — export MIME map, 10 MB cap
- `developers.google.com/workspace/drive/api/reference/rest/v3/changes/list`, `.../guides/manage-changes` (WebSearch, this session) — changes.list mechanics, drive-wide scope

### Tertiary (LOW confidence)
- Community sources on Drive API quota figures (Unipile, Nango, moldstud.com blog posts, via WebSearch this session) — quota numbers specifically, flagged as A3 in the Assumptions Log
- Community discussion of Desktop-app redirect-URI pre-registration exactness (conflicting claims, unresolved — Open Question 1)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — package choices and versions verified against the Go module proxy this session; the "why these packages" reasoning is standard, low-risk (official/team-maintained packages)
- Architecture (kernel-side): HIGH — every kernel mechanic claimed here was read directly from source this session, not assumed from documentation alone
- Architecture (Drive API/OAuth specifics): MEDIUM — cross-checked against official `developers.google.com` pages via WebFetch/WebSearch this session, but not independently re-verified against a live API call (no live Google Cloud project available in this research session)
- Pitfalls: MEDIUM-HIGH — the two most consequential findings (drive-wide `changes.list`, Match's full-replace contract) are each corroborated by an authoritative source (Google's own docs; this repo's own `kernel/index/store.go`, respectively)

**Research date:** 2026-08-15
**Valid until:** ~30 days for the kernel-side findings (stable, in-repo, unlikely to change); ~14 days for the Google Drive API/OAuth specifics (Google's own OAuth policy and quota pages have shown mid-year changes historically — re-verify before the plugin repo pins its `go.mod` if planning is delayed)
