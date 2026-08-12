# Phase 10: Docs and Release Readiness - Pattern Map

**Mapped:** 2026-08-12
**Files analyzed:** 11
**Analogs found:** 11 / 11

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `README.md` (rewrite) | doc (root) | transform (content restructure) | `README.md` (existing, self) | exact — split, not greenfield |
| `CONTRIBUTING.md` | doc (root) | transform (content extraction from README) | `README.md` §"Repository layout"/"Development loop"/"Testing" | role-match (source material) |
| `SECURITY.md` | doc (root) | request-response (points at GitHub PVR) | none in-repo; GitHub-recommended template | no analog — use researched template |
| `docs/plugins/_template.md` | doc (config/reference) | transform | `plugins/signal/README.md` | exact — best existing one-pager shape |
| `docs/plugins/paperless.md` | doc (config/reference) | transform | `config.example.toml` `[sources.paperless]` block + `plugins/signal/README.md` (shape) | role-match |
| `docs/plugins/silverbullet.md` | doc (config/reference) | transform | `config.example.toml` `[sources.silverbullet]` block + `plugins/signal/README.md` (shape) | role-match |
| `docs/plugins/proton.md` | doc (config/reference) | transform | `config.example.toml` `[sources.proton]` block + `plugins/signal/README.md` (shape) | role-match |
| `docs/plugins/signal.md` | doc (config/reference) | transform | `plugins/signal/README.md` | exact — largely a distillation of the existing file |
| `docs/plugins/whatsapp.md` | doc (config/reference) | transform | `config.example.toml` `[sources.whatsapp]` block + `plugins/signal/README.md` (shape) | role-match |
| `web/README.md` (replace) | doc (subpackage) | transform | `web/README.md` (existing, self — SvelteKit stock) | exact — full replace |
| `docs/ss/.gitkeep` (or `docs/ss/README.md`) | config (placeholder) | file-I/O | none — trivial | no analog needed |
| `scripts/sync-milestones.sh` | utility (CI/ops script) | request-response (gh api wrapper) | `scripts/signal-readonly-smoke.sh` | role-match (shell script conventions: header comment block, `set -euo pipefail`, `SCRIPT_DIR`/`REPO_ROOT` resolution) |
| `.github/workflows/nightly.yml` | config (CI workflow) | event-driven (cron) | `.github/workflows/ci.yml` | exact — same runner/action conventions, new trigger type |
| `.github/workflows/release.yml` | config (CI workflow) | event-driven (tag push) | `.github/workflows/ci.yml` | exact — same runner/action conventions, new trigger type |

## Pattern Assignments

### `README.md` (doc, transform/split)

**Analog:** `README.md` itself (existing 223-line file, full file read).

**Section boundaries to split** (exact line ranges from the current file, read this session):

| Stays in README.md (new-user) | Lines | Moves to CONTRIBUTING.md | Lines |
|---|---|---|---|
| Title + core value | 1-8 | Repository layout (Go workspace rationale) | 48-69 |
| Status and roadmap (condense) | 10-37 | Development loop (`make dev`, port guard, `DEV_PORT`/`DEV_HOST`/`DEV_READY_TIMEOUT`) | 143-179 |
| Prerequisites | 39-46 | Testing (`make test`/`make e2e`/`make dev-check` breakdown — link to `docs/testing.md`, don't inline) | 181-210 |
| Configure | 71-107 | — | — |
| Build and run | 109-141 | — | — |
| Where to look next (add CONTRIBUTING.md link) | 212-223 | — | — |

**New sections to add** (no existing analog — greenfield within README):
- Screenshot section using indexed `docs/ss/1.png..n.png` placeholders with `<!-- TODO: replace with real screenshot -->` HTML comments (per RESEARCH.md Pattern 1).
- Credits section citing Claude (Anthropic) and openGSD (`https://github.com/open-gsd/gsd-core`).

**Tone/style to preserve** (from existing README, e.g. lines 1-8, 109-116): short declarative sentences, code fences for every command, explicit "Validation:"-style callouts are NOT used in README (that's `config.example.toml`'s style) — README uses plain prose with occasional bold for key terms (`**Go 1.23+**`, `**Node 20+**`).

---

### `CONTRIBUTING.md` (doc, transform)

**Analog:** `README.md` lines 48-69 (Repository layout), 143-179 (Development loop), 181-210 (Testing) — move verbatim, adjusting only cross-references (e.g. "See README.md's Configure section" instead of assuming those sections are still local).

**Structure to follow** (inferred from how README currently orders these three concerns): Repository layout → Development loop (`make dev`) → Testing (`make test`/`make e2e`/`make dev-check`, linking to `docs/testing.md` rather than re-explaining the full gate map).

**Cross-link convention** (from README.md lines 212-223, "Where to look next" section):
```markdown
- **`docs/plugin-contract.md`** — the published contract for writing a new
  source plugin: ...
- **`docs/api.md`** — the complete kernel HTTP JSON contract: ...
```
Reuse this bolded-bullet-with-em-dash-description pattern for CONTRIBUTING.md's own "where to look next"-style links (e.g. to `docs/testing.md`).

---

### `SECURITY.md` (doc, no in-repo analog)

**Source:** RESEARCH.md Pattern 2 provides a complete, ready-to-use template grounded in this repo's actual facts (public repo confirmed via `gh api`, no versioned releases yet, kernel binds `127.0.0.1` only). Use that template verbatim, adjusting only the milestone/version framing if release engineering (this same phase) lands a first tag before this file is written.

**Style note:** match README's plain-prose tone (no "Validation:" callouts) — this is a policy document, not a config reference.

---

### `docs/plugins/_template.md` and the five per-plugin pages

**Analog:** `plugins/signal/README.md` (full file, 97 lines, read this session) — this is the strongest existing analog for shape: title, one-line description, "Install prerequisite" section, "Configuration" section with a TOML snippet, a gotchas-shaped section, and read-only/security notes woven through.

**Structure extracted from `plugins/signal/README.md`:**
```markdown
# plugins/signal                                    <- title (lines 1)
Reads Signal Desktop's own local SQLCipher...        <- one-sentence description (lines 3-9)

## Build prerequisite: install `sqlcipher` first      <- Install Requirements section (lines 11-28)
[per-distro install commands in bash fences]

## SQLCipher version floor: refuses to run below...   <- Gotchas-shaped section (lines 30-42)

## Configuration: a local-path source, no credentials  <- Configuration section (lines 44-67)
[toml fence with [sources.signal] block]
See `config.example.toml` for the fully-commented reference block.

## Read-only, by construction and by test               <- Security & Privacy Notes section (lines 69-78)
```

**Config snippet source per plugin** — pull the minimal `[sources.<id>]` block directly from `config.example.toml` (do not re-derive):
- paperless: `config.example.toml` `[sources.paperless]` block (`plugin`, `display_name`, `base_url = "${PAPERLESS_URL}"`, `token = "${PAPERLESS_TOKEN}"`, `api_version = "10"`)
- silverbullet: `[sources.silverbullet]` block (`base_url = "${SILVERBULLET_URL}"`, `token = "${SB_AUTH_TOKEN}"`, optional `ca_cert`)
- proton: `[sources.proton]` block (`base_url = "imaps://${PROTON_BRIDGE_ADDR}"`, `username = "${PROTON_BRIDGE_USER}"`, `token = "${PROTON_BRIDGE_PASS}"`, required `ca_cert`, `webmail_base_url`)
- signal: `plugins/signal/README.md` lines 50-58 (`path = "~/.config/Signal"`, no credentials)
- whatsapp: `config.example.toml` `[sources.whatsapp]` block (`path`, no `base_url`/`token`, one-time `-link` CLI step)

**Env var placeholder rule (security-relevant, ASVS V14):** every config snippet in every plugin page MUST use `${VAR}` exactly as `config.example.toml` does — never a literal-looking host/token. Copy this convention directly from `config.example.toml` lines 55-63 (paperless `base_url`/`token` comments):
```toml
base_url = "${PAPERLESS_URL}"
token = "${PAPERLESS_TOKEN}"
```

**Link-don't-duplicate convention** (RESEARCH.md Anti-Pattern, reinforced by `plugins/signal/README.md` line 60): `See config.example.toml for the fully-commented reference block.` — every per-plugin page should end its Configuration section with an equivalent one-line pointer, never reproduce the full commented block.

---

### `web/README.md` (doc, replace)

**Analog:** the file itself — current content (42 lines, read this session) is unmodified SvelteKit `sv create` scaffold output (generic "Creating a project"/"Developing"/"Building" sections referencing `npx sv create`, `npm run dev`, `npm run build`/`preview`).

**Replacement pattern:** either remove the file entirely, or replace with a two-line pointer matching this repo's cross-link convention (see README.md "Where to look next" bullet style):
```markdown
# web

The SvelteKit SPA embedded into the topos kernel binary. See the repo
root's `CONTRIBUTING.md` for the dev loop (`make dev`), build, and test
commands — this package is not run standalone in this project.
```

---

### `scripts/sync-milestones.sh` (utility, request-response/gh api wrapper)

**Analog:** `scripts/signal-readonly-smoke.sh` (lines 1-40 read this session) — the repo's existing convention for a documented, standalone ops/smoke script.

**Header comment block pattern** (lines 1-36 of the analog): a `#!/usr/bin/env bash` shebang followed by a multi-paragraph comment explaining what the script does, why, what env vars it accepts and their defaults, and any safety guarantees — before any code. Apply the same shape to `sync-milestones.sh`'s header, documenting `$1`(title)/`$2`(action) args and idempotency guarantee (RESEARCH.md Pitfall 4).

**Shell boilerplate pattern** (lines 37-40 of the analog):
```bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
```
Reuse this exact `SCRIPT_DIR`/`REPO_ROOT` resolution idiom even though `sync-milestones.sh` may not need `REPO_ROOT` (keep it if the script ever needs to read `.planning/`).

**Core `gh api` idempotent-lookup pattern:** use RESEARCH.md's Code Examples section skeleton verbatim (already grounded against the real existing milestone #1 `v1.0`) — look up by title via `--jq` filter before deciding create vs. patch.

---

### `.github/workflows/nightly.yml` and `.github/workflows/release.yml` (config, event-driven)

**Analog:** `.github/workflows/ci.yml` (full file, 83 lines, read this session).

**Header comment convention** (lines 1-15 of the analog): a top-of-file comment block stating what the workflow does, its trigger, and any load-bearing invariants (e.g. "no repository secret... referenced anywhere in this file"). Apply the same shape: state the "official `actions/*` only" convention explicitly in both new files' headers, matching lines 10-15.

**Action pinning pattern** (lines 28-39):
```yaml
- uses: actions/checkout@v7

- uses: actions/setup-go@v7
  with:
    go-version-file: go.work
    cache: true

- uses: actions/setup-node@v7
  with:
    node-version: '26'
    cache: npm
    cache-dependency-path: web/package-lock.json
```
Reuse these exact pins (`@v7` for checkout/setup-go/setup-node) in both new workflows — do not introduce different version pins for the same actions (RESEARCH.md Standard Stack table).

**Job/trigger shape:** `ci.yml` has no `permissions:` block (it never needs `contents: write`) and a single `test:` job under `on: push/pull_request`. The two new workflows diverge here — this is new territory not directly precedented in-repo, but RESEARCH.md Pattern 4/5 skeletons (already grounded in `ci.yml`'s action pins and `Makefile` build targets) are the concrete templates to use, specifically:
  - `nightly.yml`: `on: schedule: cron` + `workflow_dispatch`, two-job (`check-changes` → `build`) structure with `needs:`/`if:` gating, `permissions: contents: write` at job level.
  - `release.yml`: `on: push: tags: 'v*.*.*'`, single job, same `permissions: contents: write`.

**Build step must reuse `Makefile` targets, not reinvent them** — `make build` already produces `bin/topos` and `bin/plugins/topos-plugin-*` in one call (Makefile lines ~40-58, read this session: `build` delegates to `plugins`, which delegates to `signal`). Do not hand-roll `go build` invocations in the workflow.

**Signal-binary system dependency step** (needed only if Task 1 checkpoint decides to include it — see RESEARCH.md Pitfall 1):
```yaml
- run: sudo apt-get update && sudo apt-get install -y libsqlcipher-dev
```
No existing workflow in this repo installs this — `ci.yml` deliberately runs `make test-portable`, never `make signal`/`make test-signal` (ci.yml lines 58-61 comment). This is new territory the planner must add only if Signal is included.

---

## Shared Patterns

### `${VAR}` placeholder convention for credentials in docs
**Source:** `config.example.toml` (every `[sources.*]` block, e.g. lines 55-63 for paperless)
**Apply to:** all five `docs/plugins/*.md` pages — every config example must use `${VAR}` exactly as the source file does, never a literal-looking value (ASVS V14, Information Disclosure mitigation per RESEARCH.md Security Domain table).

### Cross-link-don't-duplicate convention
**Source:** `README.md` lines 212-223 ("Where to look next") and `plugins/signal/README.md` line 60 ("See config.example.toml for the fully-commented reference block.")
**Apply to:** CONTRIBUTING.md (link to `docs/testing.md` rather than re-explaining every gate), all `docs/plugins/*.md` pages (link to `docs/plugin-contract.md` and `config.example.toml`, never copy their content).

### Official-actions-only / pinned-version convention
**Source:** `.github/workflows/ci.yml` lines 10-15 (stated explicitly) and lines 28-78 (`actions/checkout@v7`, `actions/setup-go@v7`, `actions/setup-node@v7`, `actions/cache@v6`, `actions/upload-artifact@v7`)
**Apply to:** `nightly.yml` and `release.yml` — reuse the exact same action + version pins; no third-party marketplace actions (`softprops/action-gh-release` etc. explicitly rejected per RESEARCH.md Alternatives Considered).

### Shell script header/boilerplate convention
**Source:** `scripts/signal-readonly-smoke.sh` lines 1-40
**Apply to:** `scripts/sync-milestones.sh` — explanatory header comment block before code, `set -euo pipefail`, `SCRIPT_DIR`/`REPO_ROOT` resolution idiom.

### Least-privilege `permissions:` block (new pattern, no in-repo precedent)
**Source:** RESEARCH.md Security Domain table (ASVS V1) — no existing workflow demonstrates this since `ci.yml` never needed `contents: write`.
**Apply to:** both `nightly.yml` and `release.yml` — declare `permissions: contents: write` at the job level (not workflow-level `write-all`), the minimum scope needed to push the `nightly` tag / create releases.

## No Analog Found

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `SECURITY.md` | doc | request-response | No existing repo file serves this purpose; use RESEARCH.md's grounded template directly (Pattern 2) |
| `docs/ss/.gitkeep` (or `docs/ss/README.md`) | config placeholder | file-I/O | Trivial empty-directory marker; no pattern needed — follow RESEARCH.md Pattern 1's guidance (empty dir, no placeholder PNGs committed) |
| `permissions: contents: write` block in new workflows | config | event-driven | No existing workflow in this repo declares elevated permissions (`ci.yml` never needed to write); use RESEARCH.md Pitfall 3's guidance directly |

## Metadata

**Analog search scope:** repo root (`README.md`, `Makefile`), `.github/workflows/`, `plugins/signal/`, `scripts/`, `config.example.toml`, `docs/`, `web/README.md`
**Files scanned:** `README.md`, `.github/workflows/ci.yml`, `plugins/signal/README.md`, `web/README.md`, `config.example.toml` (relevant sections), `Makefile` (build/plugins/signal targets), `scripts/signal-readonly-smoke.sh` (header), `docs/plugin-contract.md` (opening section)
**Pattern extraction date:** 2026-08-12
