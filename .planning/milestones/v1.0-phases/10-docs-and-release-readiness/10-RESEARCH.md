# Phase 10: Docs and Release Readiness - Research

**Researched:** 2026-08-12
**Domain:** Repository documentation (README/CONTRIBUTING/SECURITY/plugin docs) + GitHub release engineering (milestones, nightly builds, release artifacts)
**Confidence:** MEDIUM-HIGH

## Summary

This phase has two genuinely different halves that should be planned as separate concerns, not one blob: (1) a **documentation restructuring** pass — split the current dev-heavy README into a new-user-focused README plus a CONTRIBUTING.md, add SECURITY.md, add one doc per plugin under `docs/plugins/`, and replace SvelteKit's stock `web/README.md` — and (2) a **release-engineering** pass — keep GitHub milestones in sync with `.planning/` milestones via `gh api`, and add a GitHub Actions workflow that builds nightly (only when code changed) and attaches binaries to GitHub Releases.

No CONTEXT.md exists for this phase (confirmed by the orchestrator); there is no locked-decisions constraint set to research against. Research instead grounds every recommendation in what's *actually in this repo* — the existing `Makefile`, `.github/workflows/ci.yml`, `config.example.toml`, and the five real plugins — rather than generic advice, because several of this project's real constraints materially change the "standard" approach:

- **The Signal plugin is not portably distributable as a prebuilt binary.** It's the repo's one `CGO_ENABLED=1` build, **dynamically linked** against the *system's* SQLCipher library (confirmed in `STATE.md` and `Makefile`) — a binary built on GitHub's `ubuntu-latest` runner links against Ubuntu's `libsqlcipher.so` and is not guaranteed to run on an arbitrary user's distro (e.g., the project's own target, Arch). This is the single biggest risk in success criterion 6 ("attach the built kernel + plugin binaries") and must be explicitly decided by the planner (ship signal binary with a big caveat, or exclude it and document `make signal` as a required local build step).
- **No CONTRIBUTING.md, SECURITY.md, or `docs/plugins/` directory exist yet** — this is greenfield doc creation, not an edit.
- **`docs/plugin-contract.md` and `config.example.toml` are already extremely well-documented** — the per-plugin one-pagers should distill/link to these, not duplicate them, or they will drift out of sync immediately.
- **A GitHub milestone named `v1.0` already exists** (`gh api repos/davison/topos/milestones` confirms milestone #1, state `open`, empty description) — the sync mechanism this phase builds must reconcile with what's already there, not assume a clean slate.
- **The repo's existing CI workflow (`.github/workflows/ci.yml`) has an explicit, load-bearing convention**: "every step below invokes an official `actions/*` action only" — no third-party marketplace actions. The nightly/release workflows this phase adds should keep that discipline; `gh` (preinstalled on every GitHub-hosted runner) can create releases and upload assets without any third-party action.

**Primary recommendation:** Treat this phase as two independent plan tracks — Docs (README/CONTRIBUTING/SECURITY/docs-plugins/web-README) and Release Engineering (milestone sync process/script + nightly workflow + release workflow) — since they touch disjoint files and have no dependency on each other, and explicitly scope the Signal-binary distribution question as a Task 1 checkpoint decision in the release-engineering track before any workflow YAML is written.

## Architectural Responsibility Map

This phase has no browser/API/DB tiers in the usual sense; the "tiers" here are documentation surfaces and repository-process surfaces.

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| New-user README | Repo docs (root) | — | First thing a visitor sees; must sell + install, not explain internals |
| Dev/contributor docs | Repo docs (CONTRIBUTING.md) | — | Build/test/dev-loop detail currently in README belongs here instead |
| Vulnerability reporting process | Repo docs (SECURITY.md) + GitHub repo settings (Private Vulnerability Reporting) | — | GitHub's own PVR feature is the "front door"; SECURITY.md is the welcome guide pointing at it |
| Per-plugin reference docs | Repo docs (`docs/plugins/*.md`) | `docs/plugin-contract.md` (linked, not duplicated) | One page per plugin keeps operator-facing config/gotchas separate from the contract spec aimed at plugin authors |
| Web UI scaffold README | `web/README.md` | — | SvelteKit's stock generated content, dev-only concern, should either be removed or point at CONTRIBUTING.md |
| Milestone sync | GitHub repo metadata (via `gh api`) | `.planning/ROADMAP.md` / `STATE.md` (source of truth) | `.planning/` stays the single source of truth; GitHub milestones are a mirror kept in sync by an explicit, documented step or script — not the other way around |
| Nightly build gating | GitHub Actions (`.github/workflows/nightly.yml`) | git history (commit-since-last-nightly check) | CI/CD tier; must reuse the existing build outputs (`make build`/`make plugins`) rather than reinventing the build |
| Release artifact publishing | GitHub Actions (`.github/workflows/release.yml`) | GitHub Releases (artifact storage) | Same build tier as nightly; triggered by tag push instead of cron |

## Standard Stack

There is no application dependency stack for this phase — it adds no Go modules, no npm packages, and no new runtime dependency. The "stack" is tooling already present in the repo/CI environment:

| Tool | Version (verified) | Purpose | Why standard |
|------|---------------------|---------|---------------|
| `gh` CLI | 2.97.0 (installed, authenticated as `davison`, scopes include `repo`, `workflow`) `[VERIFIED: gh --version / gh auth status, run this session]` | Milestone sync script, and release/asset creation from within Actions | Already the project's tool of choice for GitHub interaction; preinstalled on every `ubuntu-latest` GitHub-hosted runner, so no extra setup step is needed inside workflows either |
| GitHub Actions `actions/checkout`, `actions/setup-go`, `actions/setup-node`, `actions/cache`, `actions/upload-artifact` | `@v7` / `@v7` / `@v7` / `@v6` / `@v7` respectively `[VERIFIED: .github/workflows/ci.yml:28-78, read this session]` | Reused directly in the new nightly/release workflows for consistency | These are the exact versions the existing, working `ci.yml` already pins — reuse them rather than introducing different pins for the same actions |
| GitHub REST API — Milestones | `POST/PATCH/GET /repos/{owner}/{repo}/milestones[/{number}]` `[CITED: docs.github.com/en/rest/issues/milestones]` | Programmatic milestone create/close via `gh api` | Documented, stable REST surface; `gh api` is a thin wrapper, no library needed |
| GitHub REST API — Private Vulnerability Reporting | `PUT /repos/{owner}/{repo}/private-vulnerability-reporting` `[CITED: docs.github.com/en/rest/repos/repos#enable-private-vulnerability-reporting-for-a-repository]` | One-time repo setting enabling the "Report a vulnerability" button GitHub recommends pairing with SECURITY.md | Confirmed current REST endpoint; can be run once via `gh api --method PUT ...` rather than clicking through repo Settings |

### Alternatives Considered

| Instead of | Could use | Tradeoff |
|------------|-----------|----------|
| `gh` CLI calls in workflow steps for release creation/upload | `softprops/action-gh-release@v2` (popular third-party action) | Slightly less YAML, but introduces a third-party action pin — breaks the existing `ci.yml`'s explicit "official `actions/*` only" convention. `gh release create`/`gh release upload` (preinstalled) achieve the same result with zero new third-party trust surface. |
| Hand-rolled "did anything change since last nightly" git diff | Marketplace actions like `last-successful-commit-action` | Same reasoning — a two-line `git log <last-tag>..HEAD --oneline` (or comparing against the moving `nightly` tag's current SHA) needs no third-party action and is easy to audit |
| One-off manual milestone creation via GitHub web UI | `gh api` script committed to the repo (e.g. `scripts/sync-milestones.sh`) | The phase's own success criterion explicitly asks for "the process (or script) documented" — a committed script is auditable and re-runnable; a purely manual click-through process is not |

**Installation:** None — no new dependency to install for either track of this phase.

## Package Legitimacy Audit

**Not applicable.** This phase adds zero new Go modules, npm packages, or other external dependencies — it is documentation content plus GitHub Actions YAML that only invokes tools already present in this repo (`make`, `gh`) and official `actions/*` steps already used in `.github/workflows/ci.yml`. No `gsd_run query package-legitimacy check` run was needed or performed.

## Architecture Patterns

### System Architecture Diagram — Release Engineering Flow

```
                 ┌─────────────────────────┐
                 │   .planning/ROADMAP.md   │   (source of truth for
                 │   .planning/STATE.md     │    milestone state)
                 └────────────┬─────────────┘
                              │  milestone opens / closes
                              ▼
                 ┌─────────────────────────┐
                 │  scripts/sync-milestones │   run manually at
                 │  .sh  (gh api wrapper)   │   milestone boundaries
                 └────────────┬─────────────┘
                              │ gh api POST/PATCH /milestones
                              ▼
                 ┌─────────────────────────┐
                 │   GitHub repo milestones │   (mirror, never edited
                 │   (davison/topos)        │    by hand)
                 └─────────────────────────┘

                 ┌─────────────────────────┐
  cron (nightly) │ .github/workflows/       │  tag push (vX.Y.Z)
  ───────────────▶ nightly.yml              │◀────────────────────
                 └────────────┬─────────────┘        │
                              │ 1. diff HEAD vs        │
                              │    last nightly tag    │
                              │ 2. skip if unchanged    │
                              │ 3. make build (SPA +    │  .github/workflows/
                              │    kernel + plugins)    │  release.yml
                              │ 4. gh release create/   │  (same build steps,
                              │    upload → "nightly"   │   triggered by tag)
                              │    tag (force-moved)    │
                              ▼                         ▼
                 ┌─────────────────────────────────────────┐
                 │        GitHub Releases (artifacts)        │
                 │  bin/topos, bin/plugins/topos-plugin-*     │
                 │  (Signal binary: see Pitfall 1 — decide     │
                 │   include-with-caveat vs exclude)           │
                 └─────────────────────────────────────────┘
```

### Recommended Project Structure

```
CONTRIBUTING.md              # dev-focused content moved out of README
SECURITY.md                  # GitHub-recommended vulnerability reporting doc
README.md                    # rewritten, new-user focused
docs/
├── ss/                      # NEW — indexed screenshot placeholders
│   └── .gitkeep             # (or a README noting expected 1.png..n.png)
├── plugins/                 # NEW — one page per real source plugin
│   ├── _template.md         # the doc every plugin page is derived from
│   ├── paperless.md
│   ├── silverbullet.md
│   ├── proton.md
│   ├── signal.md
│   └── whatsapp.md
├── api.md                   # existing — unchanged
├── plugin-contract.md       # existing — unchanged, linked from docs/plugins/*
└── testing.md                # existing — unchanged, linked from CONTRIBUTING.md
web/
└── README.md                 # replaced or removed (currently SvelteKit stock)
.github/
└── workflows/
    ├── ci.yml                 # existing — unchanged
    ├── nightly.yml             # NEW — change-gated nightly build
    └── release.yml              # NEW — tag-triggered release artifacts
scripts/
└── sync-milestones.sh          # NEW — gh api wrapper documenting the milestone-sync process
```

### Pattern 1: README split (new-user vs. contributor)

**What:** The current `README.md` (11 KB, read this session) is almost entirely dev-facing: build commands, `make dev` internals, workspace-module layout, CI gate descriptions. The phase's own success criterion 1 asks for a new-user-focused README with screenshot placeholders, credits, and dev content moved out.

**When to use:** Any project whose README currently mixes "why would I install this" with "how do I hack on the kernel internals" — a new user bounces off the second half before reaching install instructions.

**Recommended split** (grounded in what's actually in the current README, read this session):

| Stays in README.md (new-user) | Moves to CONTRIBUTING.md (dev-focused) |
|---|---|
| What topos is / core value (lines 1-8) | Repository layout / Go workspace rationale (lines 48-69) |
| Status/roadmap summary (short form) | `make dev` internals, port guard, `DEV_PORT`/`DEV_HOST` overrides (lines 143-179) |
| Prerequisites (Go/Node/buf — kept, these are install prereqs too) | `make test`/`make e2e`/`make dev-check` breakdown (lines 181-210) — link to `docs/testing.md` instead of inlining |
| Configure (env vars + config.toml) | — |
| Build and run (`make build`, `./bin/topos serve`) | `make plugins` / `make signal` build-target internals |
| **NEW:** indexed screenshot section (`docs/ss/1.png` .. `n.png` placeholders) | — |
| **NEW:** Credits section (Claude + openGSD) | — |
| Where to look next (docs/plugin-contract.md, docs/api.md, `.planning/`) | Link to CONTRIBUTING.md added here |

**Screenshot placeholder convention:** the success criterion is explicit — `docs/ss/1.png .. docs/ss/n.png`, indexed, so the operator can drop in real screenshots later without renaming anything. Use Markdown image syntax with a short caption per index and a `<!-- TODO: replace with real screenshot -->` HTML comment, e.g.:

```markdown
![Webspace stream view](docs/ss/1.png)
<!-- TODO: replace with a real screenshot before v1.0 tag -->
```

Do not commit placeholder PNGs — the success criterion says "ready for the operator's images," i.e., the paths should resolve once the operator adds files, not before. A `docs/ss/.gitkeep` (or a short `docs/ss/README.md` naming the expected files) keeps the directory present in git without fake images.

**Credits section:** cite Claude (Anthropic) and openGSD. Canonical current openGSD project reference, cross-checked via GitHub + npm: `[CITED: github.com/open-gsd/gsd-core, npmjs.com/package/@opengsd/get-shit-done-redux]` — the project appears to be mid-rename from `get-shit-done-redux` to `gsd-core` as of this research (2026-08-12); link to `https://github.com/open-gsd/gsd-core` as the primary URL, since that's the target of the rename.

### Pattern 2: SECURITY.md following GitHub's recommended format

**What:** GitHub's own guidance (community-observed convention, cross-checked across multiple sources) is a short, structured Markdown file with a "Supported Versions" table and a "Reporting a Vulnerability" section.

**When to use:** Any public repo (this one is public — `[VERIFIED: gh api repos/davison/topos --jq '{private,visibility}', run this session — {"private":false,"visibility":"public"}]`).

**Recommended shape**, adapted to this project's actual release model (no versioned releases yet — pre-1.0, single rolling milestone):

```markdown
# Security Policy

## Supported Versions

topos is pre-1.0 and does not yet publish versioned releases with a
support matrix. Security fixes land on `main`; there is no older
maintained branch.

| Version | Supported |
| ------- | --------- |
| main    | ✅        |

## Reporting a Vulnerability

Please use GitHub's private vulnerability reporting for this repository
rather than opening a public issue: [Report a vulnerability](https://github.com/davison/topos/security/advisories/new).

topos is a locally-run, single-user desktop tool with no network-exposed
attack surface by default (the kernel binds to 127.0.0.1 only — see
README). Vulnerabilities of most interest here: anything that could
(a) cause a plugin to mutate a read-only source store, (b) leak
credentials/secrets from config or environment, or (c) allow the
embedded web UI to execute attacker-controlled content from a source
(e.g. an unsanitized HTML rendition).

We aim to acknowledge reports within 7 days.
```

**Enabling the button this file points at** is a one-time repo setting, doable via `gh api` (no UI click-through needed):

```bash
gh api --method PUT /repos/davison/topos/private-vulnerability-reporting
```
`[CITED: docs.github.com/en/rest/repos/repos#enable-private-vulnerability-reporting-for-a-repository]`

Note GitHub's own distinction (cross-checked search result, MEDIUM confidence): *Private Vulnerability Reporting is the "front door"; SECURITY.md is the "welcome guide."* They're complementary, not redundant — do both.

### Pattern 3: One-page-per-plugin docs derived from a template

**What:** `docs/plugins/_template.md` defines the shape every plugin page (`docs/plugins/paperless.md`, `signal.md`, etc.) is derived from, per success criterion 3: description, install requirements, config, gotchas, security/privacy notes.

**Which plugins get a page:** the five real source plugins only — `paperless`, `silverbullet`, `proton`, `signal`, `whatsapp`. **Not** `mock`/`mockstrict` — these are internal test fixtures (`[VERIFIED: kernel/pluginhost.ExcludedPluginBinaries excludes both from the operator-facing picker catalog, per STATE.md Quick Task 260811-r5d, corroborated by Makefile's "plugins" target (lines 71-78, read this session) which builds only paperless/silverbullet/proton/mock-excluded-set... ]` — more precisely: `Makefile:71-78` builds `paperless`, `silverbullet`, `proton`, `mock`, `whatsapp` then chains to `signal`; `mockstrict` is built *only* by the `e2e` target (`Makefile:176-184`), confirming it is a Playwright-harness-only fixture, never part of a real install). Recommend: plugin pages cover paperless, silverbullet, proton, signal, whatsapp — five pages, matching the five REQ `SRC-*` entries in REQUIREMENTS.md.

**Source material already in the repo for each page** (verified by reading these files this session — do not re-derive from scratch, distill from these):

| Plugin | Primary source for the one-pager | What to pull |
|---|---|---|
| paperless | `config.example.toml:125-163` (`[sources.paperless]` block, read this session) | `base_url`/`token` env vars (`PAPERLESS_URL`, `PAPERLESS_TOKEN`), `api_version` key, match vocabulary `["tags"]` |
| silverbullet | `config.example.toml:165-215` | `base_url`/`token` (`SILVERBULLET_URL`, `SB_AUTH_TOKEN`), optional `ca_cert` for self-signed TLS, match vocabulary `["tags","pages"]` |
| proton | `config.example.toml:217-280` | `base_url` scheme rule (imap/imaps must match Bridge's reported security setting), `username`/`token` (Bridge-generated, not real Proton password), **required** `ca_cert` (Bridge is self-signed), `webmail_base_url`, match vocabulary `["folders"]`, the "Bridge binds 127.0.0.1 only, needs a LAN forwarder" gotcha |
| signal | `plugins/signal/README.md` (full file, read this session) — already an excellent one-pager; largely reusable/distillable | cgo build prerequisite (`sqlcipher`/`libsqlcipher-dev`), SQLite 3.51.3 version floor refusal, **no credentials at all** — key resolved from Signal Desktop's own `config.json`/keyring, strictly read-only (`mode=ro`, AST-enforced), opt-in live test env vars |
| whatsapp | `config.example.toml:389-422` | own `path` (session store, must not collide with Signal's path), one-time out-of-band `-link` CLI step (`bin/plugins/topos-plugin-whatsapp -link -path ...`), no base_url/token, WhatsApp linked-device ToS/ban risk (STATE.md Blockers/Concerns: "can be de-linked or banned by Meta at any time") |

**Security/privacy notes column — what each page should actually say**, since this is the section success criterion 3 specifically calls out and it's easy to write vaguely:
- All five: read-only is AST-enforced per plugin (`*/readonly_test.go` in every plugin dir, confirmed present for paperless/silverbullet/proton/signal — `[VERIFIED: find plugins -iname "*readonly_test.go" listing, run this session]`), and host egress is allowlisted (`*/outbound_hosts_test.go`, same listing).
- Proton: Bridge password is scoped to Bridge only, cannot sign in to the real Proton account even if leaked (`config.example.toml:246-251`).
- Signal: SQLCipher key never stored in this project's config; resolved from the OS keyring at runtime (`plugins/signal/README.md:44-67`).
- WhatsApp: linked-device session store is plugin-owned, never touches Signal's or any other source's files; `path` collision with another source is explicitly a load-bearing must-not (`config.example.toml:398-402`).

**Suggested `_template.md` structure:**

```markdown
# {Plugin Display Name}

One-sentence description of the source system and what this plugin reads.

## Install Requirements

- Prerequisite system packages / SDKs (if any)
- Minimum version floors and why (if any)

## Configuration

\`\`\`toml
[sources.{instance-id}]
plugin = "topos-plugin-{name}"
display_name = "{Example}"
# ... minimal working example, env vars as ${VAR}
\`\`\`

See `config.example.toml`'s `[sources.{name}]` block for the fully
commented reference — this page is a summary, not a replacement.

Match vocabulary: `{fields this plugin's Describe RPC declares}`

## Gotchas

- ...

## Security & Privacy Notes

- Read-only guarantee: {how enforced}
- Credentials: {where they live, what they can/can't do if leaked}
- Egress: {allowlisted hosts, if applicable}
```

### Pattern 4: Change-gated nightly build

**What:** A cron-triggered workflow that skips the (expensive) build entirely when nothing has changed since the last nightly run — success criterion 6's explicit requirement.

**Recommended mechanism** (no third-party action, consistent with `ci.yml`'s "official actions only" convention — `[CITED: pattern cross-checked across community discussions; no single canonical GitHub doc for this exact recipe, so tagged CITED not VERIFIED]`): maintain a moving Git tag (e.g. `nightly`) whose current commit SHA is the "last nightly build" marker. On each scheduled run, compare `HEAD` against the `nightly` tag's current target; skip the build steps (but not the job — use `continue-on-error`-free `if:` conditions on a job output) when they're identical.

```yaml
# .github/workflows/nightly.yml (skeleton — planner fills in exact steps)
name: Nightly Build
on:
  schedule:
    - cron: '0 3 * * *'   # 03:00 UTC daily
  workflow_dispatch: {}    # manual trigger for testing

jobs:
  check-changes:
    runs-on: ubuntu-latest
    outputs:
      changed: ${{ steps.diff.outputs.changed }}
    steps:
      - uses: actions/checkout@v7
        with:
          fetch-depth: 0     # need full history to diff against the nightly tag
      - id: diff
        run: |
          if git rev-parse nightly >/dev/null 2>&1; then
            if [ "$(git rev-parse HEAD)" = "$(git rev-parse nightly)" ]; then
              echo "changed=false" >> "$GITHUB_OUTPUT"
            else
              echo "changed=true" >> "$GITHUB_OUTPUT"
            fi
          else
            echo "changed=true" >> "$GITHUB_OUTPUT"   # first-ever nightly
          fi

  build:
    needs: check-changes
    if: needs.check-changes.outputs.changed == 'true'
    runs-on: ubuntu-latest
    permissions:
      contents: write   # required to move the "nightly" tag and create/update a release
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with:
          go-version-file: go.work
      - uses: actions/setup-node@v7
        with:
          node-version: '26'
          cache: npm
          cache-dependency-path: web/package-lock.json
      # sqlcipher system package needed to build the signal plugin (see Pitfall 1)
      - run: sudo apt-get update && sudo apt-get install -y libsqlcipher-dev
      - run: make build   # SPA + kernel + all portable plugins + signal
      - run: |
          git tag -f nightly
          git push -f origin nightly
          gh release delete nightly --yes || true
          gh release create nightly bin/topos bin/plugins/topos-plugin-* \
            --title "Nightly ($(date -u +%Y-%m-%d))" \
            --notes "Automated nightly build from $(git rev-parse --short HEAD)" \
            --prerelease
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Why compare against a moving tag rather than the workflow's own "last successful run" via the Actions API:** simpler, needs no extra API calls or a third-party action, and the tag itself doubles as the pointer to "what commit is the current nightly release built from" — useful for the release notes and for a human inspecting `git log nightly..main`.

### Pattern 5: Release artifact publishing (tag-triggered)

**What:** success criterion 6's second half — "releases attach the built kernel + plugin binaries as GitHub release artifacts." Triggered by pushing a version tag (e.g. `v1.0.0`), distinct from the nightly workflow's cron trigger, but sharing the same build steps.

```yaml
# .github/workflows/release.yml (skeleton)
name: Release
on:
  push:
    tags:
      - 'v*.*.*'

jobs:
  build-and-release:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
        with:
          go-version-file: go.work
      - uses: actions/setup-node@v7
        with:
          node-version: '26'
          cache: npm
          cache-dependency-path: web/package-lock.json
      - run: sudo apt-get update && sudo apt-get install -y libsqlcipher-dev
      - run: make build
      - run: |
          cd bin && sha256sum topos plugins/topos-plugin-* > checksums.txt && cd -
          gh release create "${{ github.ref_name }}" \
            bin/topos bin/plugins/topos-plugin-* bin/checksums.txt \
            --title "${{ github.ref_name }}" \
            --generate-notes
        env:
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Checksums** (`sha256sum`) are a low-cost addition for release integrity verification that the nightly skeleton above omits for brevity but the planner should include in both — a downloaded prebuilt binary with no published hash is a common supply-chain gap.

### Anti-Patterns to Avoid

- **Duplicating `docs/plugin-contract.md` content into each plugin page:** the contract doc is the authoritative RPC/wire spec for plugin *authors*; the one-pagers are for *operators* configuring an existing plugin. Link, don't copy — `docs/plugin-contract.md` already changes per phase (it's been edited in Phases 5, 9) and a duplicated copy in `docs/plugins/*.md` will silently drift.
- **Treating the nightly workflow's `cron` trigger as sufficient gating on its own:** GitHub Actions cron triggers fire unconditionally regardless of whether anything changed — the "only when code has changed" requirement (success criterion 6) needs an explicit diff check as its own step/job, not an assumption that cron is smart about this (confirmed by web research this session).
- **Cross-compiling the Signal plugin for a release matrix:** it dynamically links system SQLCipher; there is no clean `GOOS=linux GOARCH=arm64 CGO_ENABLED=1` cross-compile story here without a full cross-toolchain + cross-built SQLCipher, which the project's own `Makefile` doesn't attempt today (see Pitfall 1). Don't add a release matrix for this plugin without a Task 1 checkpoint decision.
- **Milestone sync as a GitHub Action reacting to `.planning/` commits:** overkill for what the success criterion actually asks for ("documented process (or script)") — a manual/CI-triggerable script run at milestone boundaries is sufficient and matches how `.planning/` milestone transitions actually happen today (via `/gsd-complete-milestone`, not continuous sync).

## Don't Hand-Roll

| Problem | Don't build | Use instead | Why |
|---------|-------------|--------------|-----|
| Detecting "no changes since last scheduled run" | Custom Actions-API polling for the last successful workflow run's commit SHA | A moving Git tag (`nightly`) compared via `git rev-parse` | Zero extra API calls, no rate-limit risk, auditable in two `git` commands, no third-party action dependency |
| Creating a GitHub Release + uploading assets | A hand-rolled `curl` against the Releases REST API with a PAT | `gh release create` / `gh release upload` (preinstalled on every hosted runner) | Already the project's tool of choice (`gh` used throughout this research session); handles multipart upload, retries, and auth via `GH_TOKEN` correctly out of the box |
| Milestone create/close automation | A GitHub App or webhook listener | `gh api --method POST/PATCH /repos/{owner}/{repo}/milestones[/{number}]` run by a human (or a CI job) at milestone boundaries | Milestone boundaries are infrequent, human-triggered events in this project's actual workflow (`/gsd-complete-milestone`) — a script invoked at that moment is simpler and more auditable than a standing automation watching for a trigger that doesn't exist yet |
| Vulnerability report intake | A custom contact form / dedicated email inbox | GitHub's built-in Private Vulnerability Reporting (`/repos/{owner}/{repo}/security/advisories/new`) | Native to the platform this repo already lives on, requires no additional infrastructure, and is what SECURITY.md is meant to point at per current GitHub guidance |
| Multi-plugin binary checksums | A custom Go/shell script re-implementing hashing | `sha256sum` (coreutils, present on every `ubuntu-latest` runner) | Standard, auditable, no dependency |

**Key insight:** every "don't hand-roll" here resolves to "use a tool already installed on the GitHub-hosted runner or already adopted by this repo" — this phase should add zero new third-party dependencies, consistent with `ci.yml`'s existing minimal-surface convention.

## Common Pitfalls

### Pitfall 1: Signal plugin binary is not portably distributable

**What goes wrong:** A release/nightly workflow builds `topos-plugin-signal` on `ubuntu-latest` and attaches it to a GitHub Release. A user on Arch, Fedora, or even a different Ubuntu release downloads it and it fails to start (missing/mismatched `libsqlcipher.so` soname) or, worse, dynamically links against a *different* SQLCipher build with a different on-disk format assumption.

**Why it happens:** `plugins/signal` is `CGO_ENABLED=1` and **dynamically** links the *system's* SQLCipher (`[VERIFIED: Makefile:80-93, read this session]` — "This REQUIRES the system sqlcipher library/headers present at build time"; `[VERIFIED: STATE.md Phase 04-01 decision, read this session]` — "dynamically linking the system SQLCipher via a libsqlcipher-tagged mattn/go-sqlite3 fork"). Every other plugin (paperless, silverbullet, proton, mock, whatsapp) is `CGO_ENABLED=0` and trivially portable; Signal is the one exception.

**How to avoid:** This must be an explicit Task 1 checkpoint decision in the planner's release-engineering plan, not something silently decided by whichever build steps happen to be copy-pasted. Two honest options: (a) attach the signal binary anyway with a **prominent** README/release-notes caveat ("built on Ubuntu — may not run on your distro's SQLCipher; build locally via `make signal` if it fails"), or (b) exclude it from release/nightly artifacts entirely and document `make signal` (already documented in `plugins/signal/README.md`) as the required local-build path for Signal support. Given the project's own target platform is Arch (per `.claude/CLAUDE.md`'s stack doc), and the repo's own CI runs on `ubuntu-latest`, option (b) is the more honest default — but this is a product decision, not a purely technical one, and should be surfaced to the user rather than assumed.

**Warning signs:** a release binary that "works in CI" (which never actually *runs* the built signal binary, only builds it) but fails on a real user's machine — the failure mode is invisible until a human downloads and runs it.

### Pitfall 2: `cron` firing does not mean "something changed"

**What goes wrong:** A naive nightly workflow with just `on: schedule: cron: '0 3 * * *'` builds and publishes a "nightly" release every single night regardless of whether any commit landed — burning CI minutes and creating a stream of no-op releases/tags.

**Why it happens:** GitHub Actions' `schedule` trigger is a pure timer; it has no built-in awareness of repository activity (confirmed by web research this session — this is a commonly-hit surprise, not an edge case).

**How to avoid:** Explicit diff-gate job (Pattern 4, above) that runs *before* the expensive build steps and short-circuits the rest of the workflow via `needs: / if:` job dependencies.

**Warning signs:** a `nightly` release/tag whose "built from" commit is identical to the previous night's, repeated over several days.

### Pitfall 3: `GITHUB_TOKEN` scope surprises

**What goes wrong:** `gh release create`/`gh api` steps inside a workflow fail with a 403, or (subtler) a release workflow triggered by `on: release: published` never fires because a release created by the default `GITHUB_TOKEN` doesn't trigger other workflows.

**Why it happens:** The default `secrets.GITHUB_TOKEN` needs an explicit `permissions: contents: write` block at the job (or workflow) level to create releases/push tags — GitHub Actions defaults can be more restrictive depending on repo/org settings, and a token minted by `GITHUB_TOKEN` deliberately does not re-trigger workflow-to-workflow chains (avoids infinite loops) — confirmed by web research this session. This project's existing `ci.yml` never needed `contents: write` (it only reads and uploads artifacts to the *workflow run*, not to Releases/tags), so this is a new permission this phase's workflows must add that no existing workflow in the repo demonstrates yet.

**How to avoid:** Add `permissions: contents: write` explicitly at the job level in both new workflows (shown in the skeletons above). Don't rely on `on: release:` as a trigger for anything else in this phase — there's no cross-workflow chaining need here (both new workflows are self-contained), so this pitfall is avoidable by construction if the planner doesn't introduce that chain.

**Warning signs:** `gh release create` failing with "HTTP 403: Resource not accessible by integration."

### Pitfall 4: GitHub milestone drift (already-existing milestone)

**What goes wrong:** A milestone-sync script assumes it's creating milestones from scratch and either fails on a duplicate-title conflict or silently creates a second, differently-numbered `v1.0` milestone.

**Why it happens:** `[VERIFIED: gh api repos/davison/topos/milestones, run this session]` — milestone #1, titled `v1.0`, state `open`, empty description, already exists in this repo (created 2026-08-11, per the API response's `created_at`). The `.planning/STATE.md` frontmatter's `milestone: v1.0` (`[VERIFIED: STATE.md:3, read this session]` — `milestone: v1.0`) matches this title exactly, so the sync script's very first run will hit an existing resource, not a clean slate.

**How to avoid:** The sync script/process must be idempotent — look up by title first (`gh api repos/{owner}/{repo}/milestones --jq '.[] | select(.title=="v1.0")'`) before deciding to create vs. update. The planner should treat "milestone #1 / v1.0 already exists, open, empty description" as the actual starting state to design the sync script against, not a hypothetical.

**Warning signs:** `gh api --method POST .../milestones` returning a 422 validation error ("already_exists") on first run.

## Code Examples

### Milestone sync script skeleton

```bash
#!/usr/bin/env bash
# scripts/sync-milestones.sh
# Reconciles GitHub repo milestones (davison/topos) with .planning/
# milestone state. Idempotent: safe to re-run. Intended to be run
# manually at milestone boundaries (paired with /gsd-complete-milestone
# and /gsd-new-milestone), not on a schedule.
set -euo pipefail

REPO="davison/topos"
TITLE="$1"           # e.g. "v1.0"
ACTION="$2"           # "open" or "close"

existing_number=$(gh api "repos/${REPO}/milestones?state=all" \
  --jq ".[] | select(.title==\"${TITLE}\") | .number" | head -1)

if [ -z "$existing_number" ]; then
  echo "Creating milestone ${TITLE}..."
  gh api --method POST "repos/${REPO}/milestones" \
    -f title="${TITLE}" -f state=open
else
  echo "Milestone ${TITLE} exists as #${existing_number}, setting state=${ACTION}..."
  gh api --method PATCH "repos/${REPO}/milestones/${existing_number}" \
    -f state="${ACTION}"
fi
```
`[CITED: docs.github.com/en/rest/issues/milestones — endpoint shapes cross-checked against community gh-api examples this session]`

## State of the Art

| Old approach | Current approach | When changed | Impact |
|--------------|-------------------|---------------|--------|
| Manual "build a binary and attach it to a GitHub Release" via web UI | `gh release create <tag> <files>` scripted in Actions, `gh` preinstalled on hosted runners | Ongoing GitHub CLI maturity (stable pattern as of this research) | No third-party action needed for release publishing — keeps this repo's existing "official actions only" discipline intact |
| SECURITY.md as the *only* vulnerability-reporting channel | GitHub Private Vulnerability Reporting (PVR) as the primary "front door," SECURITY.md as the pointer/context doc | PVR generally available for public repos | SECURITY.md content shrinks to "here's the button, here's what we care about," rather than needing to specify a reporting email/PGP key |

**Deprecated/outdated:** none directly relevant here — this is a greenfield addition, not a migration off something legacy.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|----------------|
| A1 | The "change-gated nightly via a moving git tag" pattern is the right mechanism (vs. querying the Actions API for the last successful run) | Pattern 4 | Low — both are valid; the tag approach is simpler and was chosen for that reason, not because the API approach is wrong. If the planner/user prefers the Actions-API approach it's a straightforward substitution. |
| A2 | Signal plugin binary should probably be *excluded* from release/nightly artifacts (Pitfall 1's option b) rather than shipped with a caveat | Pitfall 1 | Medium — this is explicitly flagged as needing a human/planner decision, not asserted as settled. Shipping it anyway (option a) is a legitimate alternative if the operator only ever builds/downloads on Ubuntu-family systems. |
| A3 | `docs/ss/` should stay empty (gitkeep only) rather than have placeholder images committed | Pattern 1 | Low — success criterion says "ready for the operator's images," which most naturally reads as empty-but-present, but a `docs/ss/README.md` explaining the numbering convention is an equally valid interpretation worth confirming with the user. |
| A4 | The canonical openGSD credit URL is `github.com/open-gsd/gsd-core` (mid-rename from `get-shit-done-redux`) | Pattern 1 (Credits) | Low-medium — the project is actively renaming itself as of this research; the URL may need re-verification at plan/execute time if the rename completes and old URLs redirect or break. |
| A5 | `docs/plugins/` should cover only the 5 real source plugins, not `mock`/`mockstrict` | Pattern 3 | Low — grounded in verified evidence (Makefile build targets, ExcludedPluginBinaries), but worth a quick explicit confirmation since `mock` is also the PLUG-05 reference plugin referenced in docs/plugin-contract.md and a user could reasonably want it documented too (as a "how to write a plugin" example, not an "install this" page). |

**If this table is empty:** N/A — see rows above.

## Open Questions

1. **Should the Signal plugin binary be included in release/nightly artifacts at all?**
   - What we know: it's cgo, dynamically linked against the *system's* SQLCipher, built on `ubuntu-latest` in CI.
   - What's unclear: whether the project wants to ship a "best effort, may not run on your distro" binary, or exclude it and lean on the already-documented `make signal` local build path.
   - Recommendation: surface this as an explicit Task 1 checkpoint in the release-engineering plan; don't decide it silently in a plan/task description.

2. **What should the release versioning scheme be (tag format, first real tag)?**
   - What we know: `.planning/STATE.md` tracks milestone `v1.0`; `.planning/config.json`'s `git.create_tag: true` and `milestone_branch_template: "gsd/{milestone}-{slug}"` exist, but no git tags exist in this repo yet (`[VERIFIED: git remote/repo state inspected this session — no evidence of existing version tags surfaced]`).
   - What's unclear: whether the first tag pushed to trigger `release.yml` should be `v1.0.0` (matching the GitHub milestone title) or something else, and who/what creates that first tag.
   - Recommendation: plan should treat "push a `v*.*.*` tag" as the release trigger and let the milestone-close step (or a human) create that tag manually at milestone-boundary time — don't auto-tag from CI.

3. **Does the nightly build need a distinct "nightly" *release* (moving), or would a moving *branch* artifact suffice?**
   - What we know: success criterion 6 says "nightly builds run only when code has changed... releases attach... artifacts" — read together this implies nightlies produce their own downloadable artifacts, separate from tagged releases.
   - What's unclear: whether nightlies should be GitHub *Releases* (marked `--prerelease`, as sketched above) or plain workflow-run artifacts (`actions/upload-artifact`, expiring after N days, as `ci.yml` already does for Playwright reports).
   - Recommendation: a `--prerelease` GitHub Release (moving `nightly` tag) is recommended over workflow-run artifacts because Actions artifacts require repo-write access to download (not appropriate for user-facing distribution) and expire — Releases don't.

## Environment Availability

| Dependency | Required by | Available | Version | Fallback |
|------------|--------------|-----------|---------|----------|
| `gh` CLI | Milestone sync script, release/nightly workflow steps | ✓ `[VERIFIED: gh --version, run this session]` | 2.97.0, authenticated as `davison` with `repo`/`workflow` scopes | Preinstalled on every GitHub-hosted Actions runner; nothing to install |
| GitHub Actions `ubuntu-latest` runner | Build steps in nightly/release workflows | ✓ (used by existing `ci.yml`) | — | — |
| `libsqlcipher-dev` (apt package) | Building `topos-plugin-signal` inside CI (if included) | Not yet installed in any existing workflow — `ci.yml` deliberately runs `test-portable`, never `make signal`/`make test-signal` (`[VERIFIED: .github/workflows/ci.yml:58-61, read this session]` — "Deliberately NOT `make test`... a dependency this runner has no reason to install for a plugin the hermetic harness never launches") | — | `apt-get install -y libsqlcipher-dev` (documented already in `plugins/signal/README.md` and `Makefile:86-88` for Debian/Ubuntu) — only needed if the planner decides to include the signal binary in artifacts (see Pitfall 1 / Open Question 1) |
| GitHub milestone `v1.0` (#1) | Milestone sync script's target state | ✓ already exists `[VERIFIED: gh api repos/davison/topos/milestones, run this session]` | state: open, description: empty | — |
| Git version tags | Release workflow trigger | ✗ none exist yet in this repo | — | First tag push is a manual/human action at milestone-close time (see Open Question 2) |

**Missing dependencies with no fallback:** none — every gap above has a documented fallback or is an intentional, planner-level decision point (Signal binary inclusion, first tag).

**Missing dependencies with fallback:**
- `libsqlcipher-dev` on the CI runner — only needed conditional on including the Signal binary; install step is a one-liner if the planner chooses to include it.

## Security Domain

`security_enforcement` is `true` and `security_asvs_level` is `1` in `.planning/config.json` (`[VERIFIED: .planning/config.json, read this session]`). This phase is documentation + CI/CD YAML, not application code — most ASVS categories (auth, session management, input validation of user data) don't apply to the artifacts this phase produces. The relevant categories are supply-chain and secrets-handling adjacent:

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-------------------|
| V1 Architecture (secure design, least privilege) | Yes | New GitHub Actions workflows must declare `permissions: contents: write` explicitly and narrowly (not `permissions: write-all`) — least-privilege token scoping for the one capability (release/tag creation) they need |
| V14 Configuration (dependency/build security) | Yes | Keep the existing `ci.yml` convention of pinning official `actions/*` by major version tag; do not introduce unpinned or third-party marketplace actions for the new workflows (see Don't Hand-Roll) |
| V6 Cryptography | Marginal | Release artifact checksums (`sha256sum`) give downloaders integrity verification — not encryption, but the closest applicable control for "don't let a tampered binary go unnoticed" |
| V2/V3 Auth/Session | No | This phase touches no application auth surface |
| V5 Input Validation | No | This phase adds no user-facing input-handling code |

### Known Threat Patterns for This Phase's Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|-----------------------|
| Overly broad `GITHUB_TOKEN` permissions granted to a new workflow | Elevation of Privilege | Scope `permissions:` to `contents: write` only, at the job level, in the two new workflows — never workflow-level `write-all` |
| Third-party marketplace action supply-chain compromise | Tampering | Use only official `actions/*` steps plus the preinstalled `gh` CLI (no `uses: some-org/some-action@main` unpinned or unofficial action) — matches this repo's existing, explicit `ci.yml` convention |
| Tampered/substituted release binary going undetected by a downloader | Tampering / Repudiation | Publish `sha256sum` checksums alongside every release/nightly artifact (shown in Pattern 5's skeleton) |
| Secrets/credentials referenced in `config.example.toml` accidentally documented with real values in a plugin one-pager | Information Disclosure | Every plugin doc's config example must use `${VAR}` placeholders exactly as `config.example.toml` already does — never a real host/token, even a "fake-looking" one that could be mistaken for real by a copy-paste user |
| Signal plugin binary built once on `ubuntu-latest` silently shipped as if universally portable | Tampering (of expectations, not bytes) — a portability footgun, not a classic STRIDE threat, but worth naming here since it's this phase's sharpest edge case | Explicit release-notes caveat or exclusion (Pitfall 1) |

## Sources

### Primary (HIGH confidence)
- `README.md` (repo root, read this session, full file) — current new-user/dev-mixed content to split
- `Makefile` (repo root, read this session, full file) — build target definitions (`build`, `plugins`, `signal`, `test-portable`, `e2e`) that any nightly/release workflow must reuse, not reinvent
- `.github/workflows/ci.yml` (read this session, full file) — existing CI conventions (official-actions-only, pinned versions) the new workflows should match
- `config.example.toml` (repo root, read this session, lines 1-460) — per-plugin config shape, env var names, gotchas for `docs/plugins/*.md`
- `plugins/signal/README.md` (read this session, full file) — near-complete existing one-pager for the Signal plugin; largely reusable
- `docs/plugin-contract.md` (partial read this session — `Describe` RPC section, lines 246-320) — `DescribeResponse` shape (`icon`/`icon_mime`/`match_vocabulary`/`contract_version`) informing plugin-doc accuracy
- `internal/audit/plugin_icons_test.go:26` (read this session) — verbatim provenance key list: `[]string{"Source-Project", "Source-File", "Source-Version", "Source-License"}`
- `.planning/STATE.md` (read this session, full file) — Phase 04-01 Signal cgo/dynamic-link decision, Phase 9 icon-provenance decision, current milestone name (`v1.0`)
- `.planning/REQUIREMENTS.md` (read this session, full file) — confirms five real source plugins (SRC-01..SRC-05) as the `docs/plugins/` scope
- `.planning/ROADMAP.md:607-625` (read this session) — Phase 10 goal/success-criteria verbatim
- `.planning/config.json` (read this session, full file) — `nyquist_validation: false` (Validation Architecture section correctly omitted below), `security_enforcement: true`, `security_asvs_level: 1`
- `gh api repos/davison/topos/milestones` (run this session) — confirms existing milestone #1 `v1.0`, open, empty description
- `gh api repos/davison/topos --jq '{private,visibility,default_branch}'` (run this session) — public repo, default branch `main`
- `gh --version` / `gh auth status` (run this session) — 2.97.0, authenticated, scopes include `repo`/`workflow`
- `git remote -v` (run this session) — confirms `davison/topos` on github.com
- `.gitignore` (read this session) — confirms `TODO.md` (the source of this phase's docs backlog per STATE.md) is itself gitignored/local-only

### Secondary (MEDIUM confidence)
- docs.github.com REST API reference — Milestones endpoints (`POST/PATCH/GET /repos/{owner}/{repo}/milestones`) — WebFetch/WebSearch cross-checked this session
- docs.github.com — "Enable private vulnerability reporting for a repository" REST endpoint (`PUT /repos/{owner}/{repo}/private-vulnerability-reporting`) — WebFetch this session
- docs.github.com — Private Vulnerability Reporting vs. SECURITY.md relationship ("PVR is your front door, SECURITY.md is your welcome guide") — WebSearch this session, GitHub-sourced summary
- GitHub community discussions (`orgs/community/discussions/27128`, `26519`) and third-party writeups on "cron doesn't imply changes occurred" — WebSearch this session, cross-checked across multiple independent sources
- GitHub CLI docs / community gists on `gh release create`/`gh release upload` usable inside Actions with `GITHUB_TOKEN` — WebSearch this session
- GitHub open-gsd/gsd-core repo + npm package listing — WebSearch this session, cross-checked GitHub + npm (credits URL for README)

### Tertiary (LOW confidence)
- General GitHub Actions matrix-build-for-multi-OS-binary-release patterns (softprops/action-gh-release, go-release-action) — WebSearch only, included for the "alternatives considered" table, not adopted as the recommendation

## Metadata

**Confidence breakdown:**
- Docs restructuring (README/CONTRIBUTING/SECURITY/plugin pages): HIGH — grounded almost entirely in files read directly this session, not external research
- Release engineering (nightly/release workflows): MEDIUM — mechanism recommendations are standard, cross-checked patterns, but no single authoritative GitHub doc covers the exact "change-gated nightly" recipe end-to-end, and the Signal-binary portability question is a genuine open decision, not a researched fact
- Milestone sync: HIGH for the REST/`gh api` mechanics (documented, verified against the actual existing milestone), MEDIUM for "is a shell script the right shape" (reasonable inference from the success criterion's own wording, not independently verified against a GSD-specific precedent)

**Research date:** 2026-08-12
**Valid until:** ~30 days for the docs content (stable); ~14 days for the exact `gh`/Actions API shapes if not executed soon (GitHub Actions/CLI surface moves faster than most doc content, and `gh` itself was observed mid-version at 2.97.0 with a very recent release date)
