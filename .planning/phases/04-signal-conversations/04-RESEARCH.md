# Phase 4: Signal Conversations - Research

**Researched:** 2026-08-03
**Domain:** Reading a live, SQLCipher-encrypted Electron desktop app database (Signal Desktop) read-only, with OS-keyring-backed key retrieval and a reused chat-thread renderer
**Confidence:** MEDIUM — the matching/digest/UI mechanics reuse proven Phase 1-3 patterns (HIGH), but the key-retrieval and driver-currency questions below are genuinely unresolved and require the mandatory hands-on spike (already scheduled by ROADMAP.md) to close out; do not skip it.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions

**Stream granularity**
- **D-01:** One Signal stream item = a **conversation-day digest** — one row per conversation per day with activity ("House Move — 23 messages"). Never one row per message and never a single per-conversation row that resurfaces. Reversibility: costly — item identity (`source_id` shape), digest assembly in the plugin, and the FTS preview contract all bake in the day-digest unit; Phase 5 (WhatsApp) reuses the same shape.
- **D-02:** Digest row content: title = conversation name + message count for the day; snippet = the last 2-3 messages of the day (the tail), each prefixed with sender name.
- **D-03:** FTS/search indexes only the tail snippet shown in the preview — no hidden full-day text in the index. Deliberate privacy trade: minimal Signal plaintext lands in the unencrypted index DB; finding an old message means opening the thread, not keyword search. Reversibility: reversible — widening what's indexed later is a re-sync, not a migration.
- **D-04:** A "day" is midnight-to-midnight in the user's local timezone. The digest's stream timestamp is its last message of that day. Today's digest updates in place (count, snippet, timestamp) as new messages sync.

**Conversation matching**
- **D-05:** Eligible conversations: groups and 1:1 chats. Note to Self is excluded. Matching itself stays exact, case-insensitive against the shared keyword list (Phase 1 D-02/D-03 — locked).
- **D-06:** For 1:1 chats, the keyword matches the user's own names for the contact — the nickname set in Signal and the address-book/system contact name — and never the contact's self-chosen profile name. Rationale (user-stated): a contact must not be able to pull themselves into a webspace by renaming their own profile.
- **D-07:** Renames mirror source truth: if a conversation's name changes so it no longer matches, its digests disappear at the next sync — identical to every other source. Re-adding the new name as a keyword restores full history. No sticky-membership memory.
- **D-08:** Full history backfill: every day with activity in Signal Desktop's DB gets a digest for matched conversations — no time window.

### Claude's Discretion
The user left these areas to research/planning:
- **Thread view rendering** in the detail pane (the renderer Phase 5 reuses): bubble vs transcript layout, how much surrounding context a digest opens into, sender grouping, day headers.
- **Message richness**: how attachments, reactions, quotes, edits, and disappearing/deleted messages render in digests and the thread view.
- **Deep-link mechanics** for "open in Signal" (conversation-only fidelity is fixed; the exact `sgnl://` or launch mechanism is discretion).
- Keyring-failure / Signal-not-installed UX (existing health-chip + fail-loudly patterns apply), sync cadence for a local DB file, and the config shape for a source with no `base_url`/`token`.

### Deferred Ideas (OUT OF SCOPE)
None — discussion stayed within phase scope.

### Additional binding note (ROADMAP.md, not restated in CONTEXT.md's decisions but equally binding)
Mandatory spike before planning: Signal Desktop DB schema, `safeStorage` keyring extraction tested hands-on against the user's actual Arch/DE setup, schema-version detection, and SQLCipher/SQLite version stability — **pin SQLite ≥ 3.51.3; never VACUUM or checkpoint.**
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SRC-02 | Signal plugin reads Signal Desktop DB strictly read-only (`mode=ro`); extracts key via OS keyring (backend-detected); detects schema version and fails loudly on unknown | See "Standard Stack" (driver/key-retrieval candidates), "Common Pitfalls" (WAL read-only semantics, safeStorage dual-shape key), "Runtime State Inventory" (this machine's actual on-disk state), "Code Examples" (schema-version guard, DSN construction) |
</phase_requirements>

## Summary

This phase's risk is concentrated entirely in the "read a live third-party Electron app's encrypted database without ever touching it" problem — the digest-assembly, matching, and UI-rendering mechanics are all straightforward extensions of patterns already proven in Phases 1-3. Three findings from this research session change the shape of the plan more than anything in CLAUDE.md's original stack pick, and all three were only discoverable by going past the locked stack's package name and actually checking dates, versions, and — critically — this machine's real files:

1. **The locked driver choice is stale and fails the phase's own safety requirement.** `mutecomm/go-sqlcipher/v4` (CLAUDE.md's pick) last released `v4.4.2` on 2020-12-07 [VERIFIED: Go module proxy — `go list -m -versions`] and statically bundles a SQLite amalgamation from that era — years before SQLite 3.51.3 (2026-03-13), the version ROADMAP.md's spike note explicitly requires as a floor because 3.51.3 fixed a critical **WAL-reset database-corruption bug** [CITED: sqlite.org/releaselog/3_51_3.html]. Using the stale driver against Signal Desktop's live, actively-written, WAL-mode database (`journal_mode = WAL`, confirmed in Signal Desktop's own `Server.ts` [CITED: github.com/signalapp/Signal-Desktop]) is precisely the scenario that bug affected. This is not a hypothetical: the mandatory spike exists because of exactly this risk, and the locked package fails it outright. See "Standard Stack" for the corrected recommendation (dynamically link the system's current `sqlcipher` package via `mattn/go-sqlite3`'s `libsqlcipher` build tag) and "Package Legitimacy Audit" for the full candidate comparison.

2. **This machine's real Signal Desktop config.json does NOT use the safeStorage/keyring scheme CLAUDE.md assumed.** A non-invasive, read-only check of this machine's `~/.config/Signal/config.json` (field *names* only — the key *value* was never read or logged) found `{"key": "<64 hex chars>", "mediaCameraPermissions", "mediaPermissions"}` — the **legacy plaintext-key shape**, not `encryptedKey`/`safeStorageBackend`. [VERIFIED: direct read of this machine's live config.json, field names only]. The 64-hex-char value is shaped exactly like a raw 256-bit SQLCipher key with no wrapping at all. CLAUDE.md's "What NOT to Use" section already correctly warns that the plaintext-key assumption is outdated for *modern* installs — but this specific, real, currently-installed Signal Desktop has evidently never migrated to safeStorage (the config file's mtime predates the July-2024 safeStorage-write fix by Signal, and hasn't been rewritten since). **Success criterion 4 ("detected at runtime, never assumed") is not a nice-to-have here — it is the only way this phase works on this machine.** The plugin must check which field is present and branch: `key` present → use it directly as the SQLCipher key; `encryptedKey`+`safeStorageBackend` present → do the full libsecret/D-Bus unwrap. See "Runtime State Inventory" and "Code Examples".

3. **This machine's desktop environment (`river`, a Wayland tiling WM) is not in Electron's safeStorage backend-detection allowlist** (`gnome_libsecret` is selected only for X-Cinnamon/Deepin/GNOME/Pantheon/XFCE/UKUI/unity; KWallet variants for KDE sessions) [CITED: electronjs.org/docs/latest/api/safe-storage]. Despite that, `org.freedesktop.secrets` **is** live on this session's D-Bus (confirmed via `dbus-send ... ListNames`) because `gnome-keyring-daemon` is installed and evidently started manually as part of this river session's init — a common pattern for tiling-WM users who still want Chromium/Electron-style secret storage. This means: (a) if this Signal Desktop install ever *does* migrate to safeStorage in a future update, the freedesktop-Secret-Service code path researched below should work despite `river` not being an Electron-recognized DE, but (b) this cannot be assumed for any other user/machine this kernel might run on — the plugin's keyring code must be driven entirely by `safeStorageBackend`'s literal value, never by DE detection of its own.

**Primary recommendation:** Build the Signal plugin as a new cgo Go module (own `go.work` member, mirroring `plugins/proton`'s shape) that (a) reads `~/.config/Signal/config.json` and branches on `key` vs `encryptedKey`/`safeStorageBackend` to obtain the raw SQLCipher key, using `crypto/aes`+`crypto/cipher`+`golang.org/x/crypto/pbkdf2` (stdlib-adjacent, never hand-rolled crypto primitives) for the safeStorage-wrapped case and `github.com/keybase/go-keychain/secretservice` (pure-Go, godbus/dbus-based, MIT) for the freedesktop Secret Service round-trip; (b) opens the DB via `mattn/go-sqlite3` built with the `libsqlcipher` build tag, dynamically linked against the distro's own current `sqlcipher` shared library (Arch's `extra/sqlcipher` package, `4.14.0-1`, confirmed to carry the SQLite 3.51.3 baseline the phase requires) with a `mode=ro` DSN — **not** `immutable=1**, since Signal Desktop is a live concurrent writer; (c) reads `PRAGMA user_version` and fails loudly (naming both the found and highest-supported version) before touching `messages`/`conversations`; (d) renders the thread view as sanitized HTML (bluemonday, following `plugins/proton/body.go`'s exact pattern) served through the existing `text/html` iframe `DetailPane` branch — reusing the UI contract exactly as CONTEXT.md requires, no new content-shape or proto change needed.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| SQLCipher key retrieval (config.json parse, safeStorage unwrap, D-Bus Secret Service call) | Plugin process (Signal plugin subprocess) | — | Plugin-local OS/filesystem/D-Bus access; the kernel never sees a key or touches Signal's config |
| Read-only SQLCipher DB access + schema-version guard | Plugin process | — | `SourcePlugin.Match`/`Fetch` boundary; DB handle lives entirely inside the plugin |
| Conversation-day digest assembly (grouping messages by conversation+local-day, tail-snippet selection) | Plugin process | — | Source-specific transform, same tier as every other plugin's `Match` implementation (paperless tag lookup, IMAP folder scan) |
| Keyword matching against conversation/contact names | Plugin process | — | `Match` RPC semantics — exact/case-insensitive against native categorization, identical contract to every other source |
| Thread-view HTML rendering + sanitization | Plugin process | — | "Producing plugin decides readability" (Phase 3 precedent) — Signal plugin owns its own `bluemonday` policy + HTML wrap, same as `plugins/proton/body.go` |
| Digest persistence, sync-time correlation, upsert-in-place for today's digest | Kernel (`kernel/index`, `kernel/correlate`) | — | Existing `ReplaceWebspaceSourceItems` per-(webspace,source_type) upsert already gives "today's digest updates in place" for free — no new kernel primitive needed, only a stable `source_id` per (conversation, local-date) from the plugin |
| Config validation relaxation (no `base_url`/`token` required) | Kernel (`kernel/config`) | — | Structural gap already logged in STATE.md as deferred to Phase 4/5; must land here |
| Thread view / digest display, "open in Signal" affordance | Browser (SvelteKit SPA) | — | Reuses `DetailPane`'s existing `html` body-variant branch and `OpenInSource` component unchanged — no new frontend content-shape |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/mattn/go-sqlite3` (built with `libsqlcipher` build tag, dynamically linked against the system's `sqlcipher`) | `v1.14.49` [VERIFIED: Go module proxy] driver; link against Arch `extra/sqlcipher` `4.14.0-1` [CITED: archlinux.org package page, updated 2026-03-29] | Read-only SQLCipher access to `db.sqlite` | `mattn/go-sqlite3` is the de facto standard, actively maintained (49 patch releases) Go SQLite driver; its `libsqlcipher` build tag dynamically links whatever SQLCipher the OS already provides instead of vendoring a static amalgamation the Go package maintainer must keep current. This sidesteps the currency problem entirely: Arch's `sqlcipher` package is a system package updated via `pacman`, already confirmed to carry SQLite 3.51.3 as its baseline (see "Common Pitfalls" #1). **Caveat (must be confirmed in the spike, not assumed):** the `libsqlcipher` build tag originates from an **unmerged** upstream PR (`mattn/go-sqlite3#1109`, open since Nov 2022) — see "Package Legitimacy Audit" for the exact fork/pin strategy this requires. |
| `github.com/keybase/go-keychain/secretservice` | `v0.0.1` (in the `go-keychain` module), published 2025-02-27 [VERIFIED: Go module proxy] | Freedesktop Secret Service (`org.freedesktop.secrets`) round-trip to unwrap the safeStorage master password, when `config.json` has `encryptedKey`/`safeStorageBackend` | Pure-Go, built directly on `godbus/dbus/v5` (the locked CLAUDE.md dependency) rather than cgo `libsecret` — avoids adding a second cgo surface to a plugin that already needs one for SQLCipher. Implements session negotiation (`AuthenticationDHAES`), collection search by attribute, and secret retrieval — the exact primitives a hand-rolled D-Bus client would otherwise need to reimplement (encrypted-session Diffie-Hellman handshake in particular is not something to hand-roll). MIT licensed. |
| `golang.org/x/crypto/pbkdf2` + stdlib `crypto/aes`, `crypto/cipher` | `golang.org/x/crypto` `v0.54.0` [VERIFIED: Go module proxy] | Derive the AES-128 key from the safeStorage master password (PBKDF2-HMAC-SHA1, salt `"saltysalt"`, 1 iteration) and AES-128-CBC-decrypt the `v10`/`v11`-prefixed blob in `encryptedKey` | This is Electron/Chromium's own documented `os_crypt` scheme [CITED: electronjs.org safe-storage docs; cross-checked against a public Electron internals writeup] — not something to reinvent, but small enough (one PBKDF2 call, one CBC decrypt, strip a fixed 16-space IV and PKCS7 padding) that no third-party "Electron safeStorage in Go" package is needed or should be trusted blindly. |
| `github.com/microcosm-cc/bluemonday` | `v1.0.27` [VERIFIED: Go module proxy] (repo floor: `golang.org/x/net >= v0.33.0` per `internal/audit/module_pins_test.go` — **the Signal plugin's own `go.mod` must declare this or above**, per the exact CVE this repo already tracks) | Sanitize the rendered thread-transcript HTML fragment before wrapping it as the `Fetch` response's `text/html` rendition | Already the project's standard HTML-sanitization library (`plugins/proton`, `plugins/silverbullet`); reusing it here means the Signal plugin needs no new dependency class, and the existing `module_pins_test.go` floor-check protects it from the same `x/net` DoS CVE class the audit test already exists to catch. |
| `hashicorp/go-plugin`, `google.golang.org/grpc`, `google.golang.org/protobuf`, `hashicorp/go-hclog` | pinned versions already in `go.work`/`sdk/go.mod` | Plugin transport, logging | Unchanged from every existing plugin — no new decision here. |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| stdlib `encoding/json` | stdlib | Parse `~/.config/Signal/config.json` and the `messages.json`/`conversations.json` blob columns | No third-party JSON library is warranted; Signal's JSON blobs are plain, no streaming/size concerns at row scale. |
| stdlib `database/sql` | stdlib | Query interface over the `libsqlcipher`-tagged `mattn/go-sqlite3` driver | Standard Go SQL access pattern, identical shape to every other plugin's client code (even though those are HTTP, not SQL). |
| stdlib `time` | stdlib | Midnight-to-midnight local-day bucketing (D-04) | `time.In(time.Local)` (or an explicit configured location, if one exists elsewhere in this project's config — check `kernel/config` for a `[server] timezone`-style key before assuming `time.Local` is correct for a machine that might run in a container with a different TZ than the user's desktop). Flagged as an open question below. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `mattn/go-sqlite3` (`libsqlcipher` tag, dynamic link) | Vendor a fresh SQLCipher amalgamation directly (build `sqlite3.c` from `sqlcipher/sqlcipher`'s own source via their `make target_source && make sqlite3.c`, link `-lcrypto`) | Full control over exactly which SQLCipher/SQLite version ships, independent of what any Linux distro packages — but adds a real build-system burden (autotools, OpenSSL linkage, a vendored multi-hundred-KB C file to keep updated) that this project's existing plugins have never needed. Worth it only if dynamic-linking against a distro package proves infeasible on some target (e.g. a distro that doesn't package `sqlcipher` at all) in the spike. |
| `mattn/go-sqlite3` (`libsqlcipher` tag, dynamic link) | `mutecomm/go-sqlcipher/v4` (CLAUDE.md's original pick) | Simpler DSN-driven API (`_pragma_key=x'...'`), no unmerged-PR fork risk — but its last release (`v4.4.2`, 2020-12-07) is 5+ years stale and its statically-bundled SQLite predates the required 3.51.3 WAL-corruption fix by years. **Do not use for this phase** unless the spike finds no viable alternative; if forced to use it, the plan must add an explicit mitigating control (e.g., verify no `-wal` recovery/checkpoint ever occurs under it, or fall back to option below). |
| `mattn/go-sqlite3` (`libsqlcipher` tag) | `jgiannuzzi/go-sqlite3` fork's `sqlite3mc-*` branch (SQLite3MultipleCiphers, SQLCipher-compatible cipher scheme, bundles a much newer SQLite — 3.51.x-class as of the branch found during this research) via a `go.mod` `replace` directive | Gets a current, self-contained (no dynamic-link-target-availability risk) SQLite/SQLCipher pairing without waiting for `mattn/go-sqlite3#1109` to merge — but it is a personal fork's untagged branch, referenced by pseudo-version, which is a real supply-chain trust and long-term-maintenance question or the mandatory spike to weigh explicitly (see "Package Legitimacy Audit"). |
| freedesktop Secret Service only (`org.freedesktop.secrets`) | Native `org.kde.kwalletd5`/`kwalletd6` D-Bus API for KDE sessions older than Frameworks 5.97 | Only the Secret Service path is in scope per CLAUDE.md's locked stack; native-KWallet-API support is a real gap for a KDE user on an older Frameworks release, flagged as an Open Question, not silently expanded into scope here. |

**Installation:**
```bash
# System dependency (must exist before building the plugin) — install via the
# distro package manager, NOT vendored:
#   Arch:   sudo pacman -S sqlcipher   # confirmed 4.14.0-1 in [extra], SQLite 3.51.3 baseline
#   Debian/Ubuntu: apt-get install libsqlcipher-dev  # version must be verified in the spike — Debian/Ubuntu's packaged version may lag Arch's

# Go module dependencies (inside plugins/signal/go.mod, its own go.work member):
go get github.com/mattn/go-sqlite3@v1.14.49
go get github.com/keybase/go-keychain@v0.0.1
go get golang.org/x/crypto@v0.54.0
go get github.com/microcosm-cc/bluemonday@v1.0.27

# Build (note the required tag; CGO_ENABLED=1 unlike the rest of the workspace):
CGO_ENABLED=1 go build -tags libsqlcipher -o bin/plugins/webspaces-plugin-signal ./plugins/signal
```

**Version verification:** confirmed live via `go list -m -versions` against the real module proxy during this research session (see inline `[VERIFIED: Go module proxy]` tags above) and against `archlinux.org`'s live package page for the system `sqlcipher` version. The `mattn/go-sqlite3` `libsqlcipher` build-tag support itself is **not yet in any tagged `mattn/go-sqlite3` release** — this must be re-verified at spike/plan time (see "Package Legitimacy Audit").

## Package Legitimacy Audit

> This phase installs external Go packages. Note: `gsd-tools query package-legitimacy check` only supports `npm`/`pypi`/`crates` ecosystems — Go modules were verified manually via `go list -m -versions` (proves registry/proxy existence and real version history) plus manual review of source/maintainer reputation, since the automated legitimacy gate does not cover this ecosystem.

| Package | Registry | Age / Last Release | Downloads | Source Repo | Verdict | Disposition |
|---------|----------|---------------------|-----------|-------------|---------|-------------|
| `github.com/mattn/go-sqlite3` | Go module proxy | 49 tagged releases through `v1.14.49`; actively maintained | Extremely widely used (one of the most-imported Go modules) | github.com/mattn/go-sqlite3 | OK | Approved — but see caveat below on the `libsqlcipher` build tag specifically |
| `github.com/mutecomm/go-sqlcipher/v4` | Go module proxy | Last tag `v4.4.2`, 2020-12-07 [VERIFIED: `go list -m -versions`] | Moderate historical use, cited in this project's own CLAUDE.md | github.com/mutecomm/go-sqlcipher | SUS | **Flagged — do not use as primary.** Real, non-malicious, long-standing package, but its staleness directly conflicts with this phase's own SQLite-version success criterion. Not a security/legitimacy concern in the "hallucinated package" sense — a currency/safety concern specific to this phase. |
| `github.com/keybase/go-keychain` (`secretservice` subpackage) | Go module proxy | `v0.0.1`, 2025-02-27 [VERIFIED: `go list -m -versions`] | Backed by a known org (Keybase, MIT licensed) | github.com/keybase/go-keychain | OK | Approved |
| `golang.org/x/crypto` | Go module proxy | `v0.54.0`, actively released (official Go sub-repo) | N/A — stdlib-adjacent | golang.org/x/crypto | OK | Approved |
| `github.com/microcosm-cc/bluemonday` | Go module proxy | `v1.0.27` [VERIFIED: `go list -m -versions`] | Already vetted and in production use by `plugins/proton`/`plugins/silverbullet` in this repo | github.com/microcosm-cc/bluemonday | OK | Approved (already-audited dependency, reused) |
| `github.com/jgiannuzzi/go-sqlite3` (`sqlite3mc-*` branch, pseudo-versioned) | Go module proxy (untagged branch commit, e.g. `v1.14.35-0.20260227142656-...`) | Personal fork, active as of the unmerged upstream PR's ongoing 2022-2026 discussion | Unknown — not a canonical module path | github.com/jgiannuzzi/go-sqlite3 | SUS | **Flagged — candidate only, requires `checkpoint:human-verify` before pinning via `go.mod replace` if chosen over the dynamic-link approach.** Single-maintainer fork referenced by commit pseudo-version, not a stable tag; acceptable as a fallback if the dynamic-link approach fails in the spike, but the plan must not silently default to it. |

**Packages removed due to [SLOP] verdict:** none — every candidate here is a real, non-hallucinated, actively-referenced package; this audit's job in this phase was currency and supply-chain-trust, not existence-fraud detection.
**Packages flagged as suspicious [SUS]:** `mutecomm/go-sqlcipher/v4` (stale, fails phase's own version floor — deprioritized, not removed, in case the spike finds the dynamic-link approach infeasible on some target); `jgiannuzzi/go-sqlite3` sqlite3mc fork (untagged personal fork — needs `checkpoint:human-verify` if selected).

## Architecture Patterns

### System Architecture Diagram

```
                     ~/.config/Signal/config.json
                              │  (read-only, key material never logged)
                              ▼
                 ┌─────────────────────────────┐
                 │ 1. Key resolution            │
                 │    - "key" present? use raw  │
                 │      hex directly            │
                 │    - "encryptedKey" present?  │
                 │      → Secret Service D-Bus   │
                 │        round-trip (GNOME/KDE) │
                 │        → PBKDF2 + AES-128-CBC │
                 │        unwrap → raw key       │
                 └───────────────┬───────────────┘
                                 │ raw SQLCipher key (never logged)
                                 ▼
      ~/.config/Signal/sql/db.sqlite (+ -wal/-shm, Signal Desktop is a live writer)
                                 │  mode=ro DSN, libsqlcipher-tagged driver
                                 ▼
                 ┌─────────────────────────────┐
                 │ 2. Schema-version guard      │
                 │    PRAGMA user_version       │
                 │    > known-max? FAIL LOUD    │
                 │    (name both versions)      │
                 └───────────────┬───────────────┘
                                 ▼
                 ┌─────────────────────────────┐
                 │ 3. Match RPC                 │
                 │    resolve keyword → matching │
                 │    conversations (group/1:1,  │
                 │    D-05/D-06 name rules)      │
                 │    → SELECT messages WHERE    │
                 │      conversationId IN (...)  │
                 │    → group by local calendar  │
                 │      day (D-04)                │
                 │    → build one Item per        │
                 │      (conversation, day)       │
                 └───────────────┬───────────────┘
                                 ▼
                     kernel/correlate + kernel/index
                 (ReplaceWebspaceSourceItems upserts
                  by source_id = conversationId+date —
                  today's digest updates in place, D-04)
                                 │
                                 ▼
                       GET /api/webspaces/{ws}/stream
                                 │
                                 ▼
                     StreamList / StreamRow (Svelte)
                          │ item opened
                          ▼
                 ┌─────────────────────────────┐
                 │ 4. Fetch RPC (request-time)   │
                 │    re-open DB read-only,       │
                 │    re-fetch full day's         │
                 │    messages for this            │
                 │    conversation+date,           │
                 │    render sanitized HTML        │
                 │    transcript (bluemonday +      │
                 │    WrapDocument-style helper)    │
                 └───────────────┬───────────────┘
                                 ▼
                DetailPane's existing `html` body-variant
                (iframe → kernel's sanitized /content route)
                  + OpenInSource (deep_link = sgnl://,
                    LINK_FIDELITY_CONVERSATION_ONLY)
```

### Recommended Project Structure
```
plugins/signal/
├── go.mod                  # own module, cgo-enabled, own go.work member
├── main.go                 # goplugin.Serve wiring — mirrors plugins/proton/main.go
├── plugin.go                # SourcePlugin: Describe/Match/Fetch/Health
├── keyresolve.go            # config.json parse + key/encryptedKey branch
├── safestorage_linux.go     # PBKDF2 + AES-128-CBC unwrap (Electron os_crypt scheme)
├── secretservice.go          # keybase/go-keychain/secretservice wrapper (GNOME/KDE)
├── dsn.go                   # mode=ro DSN construction, libsqlcipher-tagged driver open
├── schemaguard.go            # PRAGMA user_version check, fails loudly by name
├── digest.go                 # conversation+local-day grouping, tail-snippet, D-02/D-03/D-04
├── match.go                   # keyword → conversation resolution, D-05/D-06 name rules
├── render.go                  # thread HTML transcript + bluemonday sanitize + wrap (mirrors plugins/proton/body.go)
├── deeplink.go                 # sgnl:// construction
├── readonly_test.go             # AST scan: no write-shaped SQL (INSERT/UPDATE/DELETE/PRAGMA that mutates) anywhere in non-test .go files — Signal's own version of plugins/proton/readonly_test.go
├── byte_identical_test.go        # hash db.sqlite before/after a full sync, assert unchanged (success criterion 3) — the strongest read-only test in the repo yet
├── outbound_hosts_test.go         # proves ZERO outbound network hosts (this plugin has none — local file + local D-Bus only)
└── schema_version_fixture_test.go  # negative control: a fixture DB with an inflated user_version must fail loudly by name
```

### Pattern 1: Dual-shape key resolution (branch on field presence, never assume)
**What:** Read `config.json`, check which of `key` / `encryptedKey`+`safeStorageBackend` is present, and take the corresponding path. Treat "neither field present" and "both present" as the same fail-loud case as an unrecognized backend value.
**When to use:** Every time the plugin starts up or re-resolves the key (config.json could theoretically change between syncs if the user re-links Signal Desktop or a migration fires).
**Example:**
```go
// Source: this research session's direct inspection of a real
// ~/.config/Signal/config.json (field names only — never logs values).
type signalConfig struct {
    Key                string `json:"key,omitempty"`
    EncryptedKey       string `json:"encryptedKey,omitempty"`
    SafeStorageBackend string `json:"safeStorageBackend,omitempty"`
}

func resolveKey(cfg signalConfig) (rawHexKey string, err error) {
    switch {
    case cfg.Key != "" && cfg.EncryptedKey == "":
        // Legacy, unmigrated install — the key IS the DB key, already hex.
        return cfg.Key, nil
    case cfg.EncryptedKey != "" && cfg.Key == "":
        return unwrapSafeStorageKey(cfg.EncryptedKey, cfg.SafeStorageBackend)
    default:
        return "", fmt.Errorf("signal: config.json has an unrecognized key shape (key present=%v, encryptedKey present=%v) — refusing to guess", cfg.Key != "", cfg.EncryptedKey != "")
    }
}
```

### Pattern 2: Fail-loud schema-version guard before any table access
**What:** Read `PRAGMA user_version` immediately after opening the read-only connection and before running any query against `messages`/`conversations`. Compare against a hardcoded "highest version this plugin was built/tested against" constant.
**When to use:** On every `Match`/`Health` call, or at minimum once per plugin process lifetime — cheap enough to do every time.
**Example:**
```go
// Source: derived from Signal Desktop's own PRAGMA user_version / SCHEMA_VERSIONS
// pattern (github.com/signalapp/Signal-Desktop ts/sql/Server.ts), adapted for a
// read-only third-party reader.
const highestSupportedSchemaVersion = 1640 // update deliberately when re-verified against a newer Signal Desktop release

func guardSchemaVersion(db *sql.DB) error {
    var found int
    if err := db.QueryRow(`PRAGMA user_version`).Scan(&found); err != nil {
        return fmt.Errorf("signal: read schema version: %w", err)
    }
    if found > highestSupportedSchemaVersion {
        return fmt.Errorf(
            "signal: unrecognised database schema version %d (this plugin was built against up to %d) — refusing to import, not silently skipping",
            found, highestSupportedSchemaVersion,
        )
    }
    return nil
}
```

### Pattern 3: mode=ro DSN, never immutable=1, against a live WAL writer
**What:** Construct the SQLite URI DSN with `mode=ro` and the SQLCipher key pragma, explicitly WITHOUT `immutable=1`.
**When to use:** Every connection this plugin ever opens to `db.sqlite`.
**Example:**
```go
// Source: SQLite URI filename docs (sqlite.org) + mattn/go-sqlite3 DSN conventions,
// cross-checked against this research session's WAL-mode findings.
dsn := fmt.Sprintf(
    "file:%s?mode=ro&_pragma_key=x'%s'&_pragma_cipher_page_size=4096",
    dbPath, rawHexKey,
)
// Deliberately NOT adding &immutable=1: Signal Desktop is a live concurrent
// writer (journal_mode=WAL) whenever it's running — immutable=1 tells SQLite
// the file will never change and disables its own change-detection/locking,
// which risks stale or torn reads exactly in the "Signal Desktop running at
// the same time" case success criterion 3 calls out by name.
```

### Anti-Patterns to Avoid
- **Copying `db.sqlite` to a temp location before reading it:** tempting as a way to sidestep concurrent-access questions entirely, but it silently violates the spirit of "byte-identical... including when Signal Desktop is running at the same time" (you'd be reading a torn snapshot, not what the running app currently sees) and adds a second, unencrypted, unmanaged copy of the user's message plaintext to disk — directly against this project's own hybrid-data-model privacy stance. Read the live file with a correct `mode=ro` WAL-aware connection instead.
- **Logging the raw key or the `encryptedKey`/`key` field values, even at debug level:** the plugin contract's logging rule ("never log a credential") applies with extra force here — the SQLCipher key is a decryption key for the user's entire message history, not an API token. Log only "key resolved via legacy field" / "key resolved via safeStorage (`backend=gnome_libsecret`)" style presence/backend messages.
- **Hand-rolling the safeStorage AES/PBKDF2 unwrap "close enough":** the exact salt (`"saltysalt"`), iteration count (1), and hardcoded 16-space IV are load-bearing constants copied from Chromium's `os_crypt` — get any one wrong and you get silent garbage (CBC decryption with the wrong key doesn't error, it just produces corrupted output) rather than a clean failure. Write a unit test against a known-good `v10`/`v11` fixture pair before trusting this against a real key.
- **Assuming any single keyring backend for "the user's DE":** the desktop-environment-to-backend mapping is a *default*, not a guarantee (this research session's own environment — a non-GNOME/KDE tiling WM with `gnome-keyring-daemon` manually started — is a live counter-example). Always branch on the literal `safeStorageBackend` string value, never on `$XDG_CURRENT_DESKTOP`.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Freedesktop Secret Service session negotiation (Diffie-Hellman encrypted session, collection search, secret retrieval) | Raw `godbus/dbus` method calls against `org.freedesktop.Secret.Service` | `github.com/keybase/go-keychain/secretservice` | The DH-encrypted session mode in particular (`dh-ietf1024-sha256-aes128-cbc-pkcs7`) is exactly the kind of "looks simple, is easy to get subtly wrong" crypto handshake this project's own philosophy warns against hand-rolling; a maintained package already implements and (presumably) tests it. |
| HTML sanitization of the rendered chat transcript | A custom tag/attribute allowlist regex or DOM walker | `github.com/microcosm-cc/bluemonday` (already the project's standard, `plugins/proton`/`plugins/silverbullet` precedent) | Untrusted HTML sanitization is a security-critical, extensively-fuzzed problem class; this project has already made and validated this choice twice. |
| SQLite WAL/page-cache semantics for a "read-only while a writer is live" guarantee | A hand-rolled parser of `db.sqlite` + `db.sqlite-wal` frame format (reimplementing what SQLite's own C code does to merge WAL frames into a consistent read view) | A real SQLite/SQLCipher C library (via cgo) opened with a correct `mode=ro` DSN | This is precisely the trap the `sigtop` project (`github.com/tbvdm/sigtop`, ISC-licensed prior art for exactly this problem) avoided by vendoring a real SQLCipher amalgamation rather than reimplementing WAL merging in Go — WAL correctness under concurrent access is exactly the kind of intricate, stateful logic this project's own "Don't Hand-Roll" philosophy exists to warn against. |
| Electron `safeStorage` AES-128-CBC decrypt + PBKDF2 key derivation | A bespoke from-scratch crypto routine | Go stdlib `crypto/aes`/`crypto/cipher` + `golang.org/x/crypto/pbkdf2`, with the exact Chromium/Electron constants (salt, iteration count, IV) copied verbatim from the documented scheme | The *algorithm* (AES-128-CBC, PBKDF2-HMAC-SHA1) is genuinely simple enough that a full third-party wrapper library isn't warranted — but the constants must be copied exactly from the documented `os_crypt` scheme, not "close enough" reinvented, since a wrong constant fails silently rather than loudly (CBC has no built-in authentication). |

**Key insight:** every hand-roll temptation in this phase (WAL semantics, D-Bus crypto handshake, HTML sanitization) shares the same failure shape — it looks like "just a few lines of code" until the edge case (a partially-flushed WAL frame, a malformed encrypted-session handshake, a crafted `<script>` inside a message body) turns "a few lines" into a security or data-corruption bug. Every one of them already has a maintained, narrowly-scoped library in this project's existing dependency tree or trivially addable to it.

## Runtime State Inventory

> Not a rename/refactor phase in the classic sense, but SRC-02's own success criteria are fundamentally about *runtime state this plugin must never disturb* — the canonical question ("what runtime systems still have the old string cached, stored, or registered?") maps directly onto "what does Signal Desktop's OWN runtime state look like on the target machine, and what must this plugin never touch?". Answered explicitly below.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | `~/.config/Signal/sql/db.sqlite` (+ `-wal`/`-shm` sidecars) — SQLCipher-encrypted, WAL mode confirmed (`journal_mode = WAL`, `synchronous = NORMAL` set by Signal Desktop's own `Server.ts`). The plugin reads this via `mode=ro`; the `-shm` file will still receive writes from the plugin's own reader "end mark" bookkeeping (SQLite's own documented WAL semantics), which is normal and does not violate the byte-identical guarantee on `db.sqlite` itself — see "Common Pitfalls" #2. No data migration needed; this phase adds a new read-only consumer, nothing more. |
| Live service config | `~/.config/Signal/config.json`, confirmed to exist on this machine (141 bytes at last check, 3 top-level fields: `key`, `mediaCameraPermissions`, `mediaPermissions` — **legacy plaintext-key shape**, not the `encryptedKey`/`safeStorageBackend` shape CLAUDE.md's research assumed). This file is Signal Desktop's own live config, never written to by this project — read-only. | Code must branch on which shape is present (see Pattern 1) — no code assuming only the modern shape. |
| OS-registered state | signal-desktop's `.desktop` entry registers `x-scheme-handler/sgnl` (confirmed fixed upstream/on Arch since package `1.39.6-2`) — this is what makes the "open in Signal" deep link work at all; nothing this project needs to register or change. | None — purely a dependency on Signal Desktop's own existing OS registration. |
| Secrets/env vars | No env vars needed for this source at all (a local-path source, per CONTEXT.md's Claude's-Discretion note on config shape) — the "secret" here (the SQLCipher key) is derived entirely at runtime from `config.json` + optionally the OS keyring, never stored in this project's own config or environment. | `kernel/config.Validate`'s unconditional `base_url`/`token` requirement must be relaxed for this source (already logged as a deferred item in STATE.md, landing here). |
| Build artifacts | None yet — `plugins/signal` does not exist. First build will require `CGO_ENABLED=1` and the system `sqlcipher` dev package/library present at build time (see "Environment Availability"), a first for this codebase's plugin set (the Signal plugin is the first genuinely cgo-required plugin to actually ship, following the pattern already anticipated for it in CLAUDE.md). | `go.work` needs a new `./plugins/signal` member; `make build`-equivalent tooling needs a path for a cgo-enabled plugin build separate from the rest of the (cgo-free) workspace. |

## Common Pitfalls

### Pitfall 1: The locked driver package is 5+ years stale and misses the phase's own SQLite version floor
**What goes wrong:** Building the Signal plugin against `mutecomm/go-sqlcipher/v4` (as CLAUDE.md's stack table names) silently ships a SQLite core from ~2020, years before SQLite 3.51.3's WAL-reset-corruption fix — exactly the failure mode a read-only reader of a live, actively-written WAL database is exposed to.
**Why it happens:** The package name in the stack doc was correct when written, but the upstream project stopped releasing; nothing about `go get`-ing it warns you its bundled C code is ancient (Go module version numbers don't reflect the age of vendored/statically-linked C source).
**How to avoid:** Do not `go get` it without first checking (as this research did) its last-release date and cross-referencing the bundled SQLite version against the phase's explicit `>= 3.51.3` requirement. Prefer the dynamic-link-against-system-`sqlcipher` approach in "Standard Stack".
**Warning signs:** Any test that exercises the "Signal Desktop running concurrently" scenario (success criterion 3) intermittently reporting a changed hash on `db.sqlite`, or a `SQLITE_CORRUPT`/`disk image is malformed` error under load — both are consistent with the WAL-reset bug's symptoms.

### Pitfall 2: "byte-identical" does not (and cannot) extend to the `-shm` sidecar file
**What goes wrong:** A byte-identical test written against the whole `~/.config/Signal/sql/` directory (including `-wal`/`-shm`) will spuriously fail even with a perfectly correct read-only implementation, because SQLite's own documented WAL-reader protocol requires write access to `-shm` to record each reader's "end mark" — this is normal, expected, and happens even for genuinely read-only connections.
**Why it happens:** "Read-only" at the SQL level (`mode=ro`, no `INSERT`/`UPDATE`/`DELETE`) is not the same guarantee as "the OS-level bytes of every file this connection touches never change" — SQLite's WAL design deliberately uses the `-shm` file as shared, mutable bookkeeping between all readers and the one writer.
**How to avoid:** Scope the byte-identical assertion (success criterion 3) to `db.sqlite` itself (hash before/after a full sync), not the whole directory. Document this scoping explicitly in the test's own comments so a future reader doesn't "fix" the test to also check `-shm` and reintroduce a false failure.
**Warning signs:** A byte-identical test that passes on a machine where Signal Desktop is closed during the test run, but fails as soon as Signal Desktop is left open — that's the `-shm` bookkeeping, not real corruption.

### Pitfall 3: Assuming a DE ⇒ keyring backend mapping instead of reading `safeStorageBackend` literally
**What goes wrong:** Code that branches on `$XDG_CURRENT_DESKTOP` (or infers GNOME-vs-KDE from installed packages) to decide "use libsecret" vs "use KWallet" will misbehave on any DE outside Electron's own hardcoded allowlist (X-Cinnamon/Deepin/GNOME/Pantheon/XFCE/UKUI/unity for `gnome_libsecret`; KDE4/5/6 sessions for the `kwallet*` variants) — this research session's own machine (`river`) is a live example of exactly that gap, and yet still has a working `org.freedesktop.secrets` service via a manually-started `gnome-keyring-daemon`.
**Why it happens:** It's tempting to shortcut "detect the backend" with "detect the DE," since they're correlated most of the time — but Signal Desktop itself doesn't do that; it trusts Electron's own `getSelectedStorageBackend()` result, written verbatim into `config.json` as `safeStorageBackend`.
**How to avoid:** Always read `safeStorageBackend` from `config.json` and drive keyring-backend selection off that literal string (`gnome_libsecret` / `kwallet` / `kwallet5` / `kwallet6` / `basic_text` / `unknown`) — never off environment/DE detection performed independently by this plugin. `basic_text` (hardcoded `"peanuts"` password, `v10` prefix) still round-trips through the exact same AES-128-CBC/PBKDF2 code path, just with a different password source — worth a unit test fixture of its own since it needs zero D-Bus access at all.
**Warning signs:** The plugin working during development (author's own GNOME/KDE machine) but failing silently or asking for a keyring password prompt on a user's differently-configured Linux box.

### Pitfall 4: PBKDF2/AES-CBC decrypting to garbage instead of erroring on a wrong key/backend mismatch
**What goes wrong:** AES-CBC has no built-in integrity check — decrypting `encryptedKey`'s ciphertext with the wrong password (e.g., because the wrong keyring backend's secret was fetched, or KWallet returned a stale/pre-migration secret after a backend switch) produces a wrong-but-valid-looking byte string, not an error.
**Why it happens:** This is inherent to unauthenticated CBC mode — it's exactly why `signalapp/org.signal.Signal`-adjacent tooling documents a distinct `SafeStorageBackendChangeError` class of failure (a backend-migration mismatch) rather than relying on decryption itself to fail.
**How to avoid:** After decrypting, validate the result looks like a plausible SQLCipher key (expected byte length; and — the strongest check — actually try opening `db.sqlite` with it and confirm `PRAGMA user_version` (or any trivial read) succeeds) before treating key resolution as done. Surface a distinct, named error ("safeStorage backend mismatch — config.json says `%s` but the retrieved secret did not decrypt a usable key") rather than a generic SQLite open failure.
**Warning signs:** `Match`/`Health` failing with a generic "file is not a database" or "not a database, or corrupted" error from the SQLite layer — misleading, since the file is fine; the key was wrong.

### Pitfall 5: An unmerged upstream PR is the only source of the exact build tag this plan may depend on
**What goes wrong:** `mattn/go-sqlite3`'s `libsqlcipher` build tag (the recommended dynamic-link path) comes from PR #1109, open since November 2022 and still unmerged as of this research session — meaning it is not present in any tagged `mattn/go-sqlite3` release, and using it requires either a `replace` directive to a fork or vendoring the tag's diff locally.
**Why it happens:** SQLCipher support is a long-requested but contentious feature for `mattn/go-sqlite3`; the maintainer has not accepted it despite significant community interest.
**How to avoid:** The spike must concretely resolve this before planning locks in the approach — either (a) confirm a specific fork/branch/pseudo-version that carries the patch and pin it explicitly with a comment explaining why, or (b) fall back to vendoring a fresh SQLCipher amalgamation directly (the "Alternatives Considered" row above), or (c) reconsider `mutecomm/go-sqlcipher/v4` with an explicit, tested mitigation for the WAL-reset bug if neither (a) nor (b) proves workable. Do not let this remain unresolved into the plan.
**Warning signs:** `go build -tags libsqlcipher` failing with "unknown build tag" or falling back to the embedded (not dynamically linked) SQLCipher — silently defeating the whole point of the dynamic-link strategy.

## Code Examples

### Constructing the read-only SQLCipher DSN
```go
// Source: SQLite URI filename conventions (sqlite.org) + mattn/go-sqlite3 DSN
// parameter conventions, adapted per this research's WAL-mode findings.
func openReadOnly(dbPath, rawHexKey string) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"file:%s?mode=ro&_pragma_key=x'%s'&_pragma_cipher_page_size=4096",
		dbPath, rawHexKey,
	)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("signal: open %s read-only: %w", dbPath, err)
	}
	// A single trivial read proves both the mode=ro connection AND the key
	// are correct before Match/Fetch proceed any further (Pitfall 4).
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		db.Close()
		return nil, fmt.Errorf("signal: verify key by reading schema version: %w", err)
	}
	return db, nil
}
```

### Electron safeStorage AES-128-CBC unwrap (the exact, documented constants)
```go
// Source: Electron/Chromium os_crypt scheme, as documented at
// electronjs.org/docs/latest/api/safe-storage and cross-checked against a
// public Electron-internals writeup during this research session. These
// constants (salt, iteration count, IV) are Chromium's own, not invented
// here — do not "simplify" them.
const (
	osCryptSalt   = "saltysalt"
	osCryptIVSize = 16 // 16 space characters (0x20), not random
)

func decryptSafeStorageBlob(masterPassword []byte, blob []byte) ([]byte, error) {
	if len(blob) < 3 || (string(blob[:3]) != "v10" && string(blob[:3]) != "v11") {
		return nil, fmt.Errorf("signal: safeStorage blob missing v10/v11 prefix")
	}
	ciphertext := blob[3:]

	key := pbkdf2.Key(masterPassword, []byte(osCryptSalt), 1, 16, sha1.New)
	iv := bytes.Repeat([]byte{0x20}, osCryptIVSize)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("signal: aes.NewCipher: %w", err)
	}
	if len(ciphertext)%block.BlockSize() != 0 {
		return nil, fmt.Errorf("signal: safeStorage ciphertext not a multiple of block size")
	}
	plaintext := make([]byte, len(ciphertext))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(plaintext, ciphertext)
	return pkcs7Unpad(plaintext) // reject a malformed/wrong-key result loudly (Pitfall 4) rather than returning garbage
}
```

### Conversation-day digest grouping (D-01/D-02/D-04)
```go
// Source: this project's own D-01/D-02/D-04 decisions (04-CONTEXT.md) — no
// external reference, this is pure phase-specific business logic. Shown
// here because the local-day boundary and stable source_id shape are both
// load-bearing for correctness (D-04's "updates in place" and D-07's
// "renames mirror source truth" both depend on source_id being derived the
// same way, deterministically, on every sync).
func sourceIDForDigest(conversationID string, localDay time.Time) string {
	// A stable (conversation, local-date) key — re-syncing today produces
	// the SAME source_id, so kernel/index.ReplaceWebspaceSourceItems's
	// existing upsert-by-source_id semantics give "today's digest updates
	// in place" for free, no new kernel primitive required.
	return fmt.Sprintf("%s:%s", conversationID, localDay.Format("2006-01-02"))
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Signal Desktop DB key stored as plaintext `key` in `config.json` | Key wrapped via Electron `safeStorage`, stored as `encryptedKey`+`safeStorageBackend` | Merged upstream ~July 2024 [CITED: multiple sources cross-referenced during this research] | A reader written only against the old shape breaks on any freshly-provisioned or freshly-migrated install; a reader written only against the new shape breaks on any install (like this research session's own machine) that hasn't migrated yet. Both shapes are live in the wild today — this is exactly why success criterion 4 mandates runtime detection. |
| `mattn/go-sqlite3` has no SQLCipher support | An unmerged PR (`#1109`) adds `sqlcipher`/`libsqlcipher` build tags | PR opened Nov 2022, still open as of this research (2026-08) | The "obvious" standard driver for Go SQLite doesn't have first-class SQLCipher support even today — every viable path (dynamic-link fork, vendored amalgamation, or a different wrapper package entirely) carries some non-mainline risk that must be explicitly weighed, not glossed over. |
| SQLCipher bundling an old SQLite core indefinitely | SQLCipher 4.14.0 (2026-03-17) explicitly rebased its SQLite baseline to 3.51.3 specifically to fix the WAL-reset corruption bug | 2026-03-17 [CITED: zetetic.net/blog/2026/03/17/sqlcipher-4.14.0-release] | Confirms the phase's own "pin SQLite ≥ 3.51.3" requirement traces to a real, recent, named bug — not an arbitrary number — and confirms which SQLCipher release is the actual floor to target. |

**Deprecated/outdated:**
- The plaintext-`key`-in-`config.json` assumption CLAUDE.md's own "What NOT to Use" section already flags as outdated for *modern* Signal Desktop is, per this research's direct check, still the *live, current* state of at least one real installation — "outdated" and "still encountered in the wild" are not mutually exclusive, and the plugin must handle both.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | `mattn/go-sqlite3`'s `libsqlcipher` build tag (from unmerged PR #1109) can be obtained via a maintained fork or a `replace` directive with acceptable supply-chain trust, and actually dynamically links correctly against Arch's `sqlcipher` `4.14.0-1` package's headers/library naming | Standard Stack, Pitfall 5 | If the fork is abandoned or the ABI/pkg-config naming doesn't line up, the whole recommended driver strategy needs to fall back to vendoring an amalgamation (Alternative row) — a materially bigger build-system task that should be scoped into the plan's first task either way, as a spike checkpoint |
| A2 | Debian/Ubuntu (or other future deployment targets beyond this Arch machine) package a SQLCipher version also ≥ 4.14.0 (SQLite baseline ≥ 3.51.3) | Standard Stack (Installation block) | If a target distro's packaged `sqlcipher` is older, the dynamic-link approach silently reintroduces the WAL-reset-bug exposure on that distro even though it's fine on Arch — needs an explicit version check at plugin startup (read `sqlite3_libversion()` via the driver and fail loudly if below the floor), not just a install-doc assumption |
| A3 | The highest Signal Desktop schema `user_version` this plugin should treat as "known/supported" is a number the planner/executor will pin against a specific, currently-installed Signal Desktop version at spike time (not the placeholder `1640` used in the Code Examples snippet, which is an illustrative value from search-result citations, not independently confirmed against a live install) | Code Examples (Pattern 2), Common Pitfalls | Pinning the wrong ceiling either rejects a legitimately-supported schema (false failure) or accepts a genuinely newer, unverified schema silently (defeats success criterion 5's entire purpose) |
| A4 | `sgnl://` with no path/target reliably raises or focuses a running Signal Desktop window under this project's actual desktop environment(s), rather than opening a fresh, unfocused instance or doing nothing visible | Deep-link mechanics (Claude's Discretion area) | If it's a no-op or confusing on the user's actual setup, the "open in Signal" affordance (UI-04, a locked v1 requirement generally) technically exists but provides a poor/no user-visible result — worth an explicit hands-on check in the spike, not just a documentation-based assumption |
| A5 | `time.Local` (or an equivalent explicit local-timezone resolution) is the correct interpretation of "the user's local timezone" for D-04's midnight-to-midnight day boundary, on whatever machine/environment the kernel process actually runs in | Standard Stack (Supporting), D-04 | If the kernel runs in a container or service context with a different `TZ` than the user's desktop session, day-digest boundaries would silently use the wrong timezone — worth confirming there's no existing project-level timezone config to defer to instead of assuming `time.Local` |

**If this table is empty:** N/A — see rows above; every A-numbered row above needs explicit confirmation before or during planning, most urgently A1/A2 (they gate the entire driver strategy).

## Open Questions

1. **Which fork/strategy resolves `mattn/go-sqlite3`'s missing `libsqlcipher` build tag?**
   - What we know: the tag is real, documented, and requested (PR #1109), and at least one usable fork/branch (`jgiannuzzi/go-sqlite3`, `sqlite3mc-*`) exists carrying a working variant with a current SQLite baseline.
   - What's unclear: whether that specific fork (or another) is the right long-term dependency to pin, versus vendoring a fresh SQLCipher amalgamation directly, versus another approach the hands-on spike might surface.
   - Recommendation: make this the mandatory spike's Task 1, with a hard go/no-go checkpoint before any digest/matching code is written — this is the one finding in this document that, if wrong, invalidates the rest of the plan's plumbing.

2. **Does a KDE-session user (or any DE outside Electron's `gnome_libsecret` allowlist and without a manually-started `gnome-keyring-daemon`) actually get a working `safeStorageBackend` value this plugin can act on, or does Signal Desktop fall back to `basic_text`/`unknown` in ways not yet observed?**
   - What we know: KWallet has supported the freedesktop Secret Service API since Frameworks 5.97 (Plasma 6.2 improved this further), so the same code path *should* work for modern KDE too.
   - What's unclear: this has not been hands-on verified against an actual KDE session in this research pass (only against this machine's river+gnome-keyring-daemon setup).
   - Recommendation: if the project's actual target users include a KDE desktop, add that as an explicit spike check; otherwise, document the freedesktop-Secret-Service-only scope as a known, accepted gap for older/non-standard KWallet configurations.

3. **What is the actual, currently-correct "highest known Signal Desktop schema `user_version`" to hardcode as the fail-loud ceiling?**
   - What we know: Signal Desktop uses `PRAGMA user_version` with a `SCHEMA_VERSIONS`-array-derived maximum, and recently-observed values in public bug reports range from the 1000s into the 1600s.
   - What's unclear: the exact current number, since Signal Desktop ships frequent releases and this research's sources are a mix of dates.
   - Recommendation: read it directly off this machine's real `db.sqlite` (a completely safe, read-only, non-secret operation — just `PRAGMA user_version`) as the spike's first concrete data point, rather than trusting any web-sourced number.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Go C compiler (`gcc`/`cc`) | cgo build of the Signal plugin | ✓ | `gcc` present at `/usr/bin/gcc` | — |
| System `sqlcipher` library/headers (for `libsqlcipher`-tagged dynamic link) | Recommended driver strategy | ✗ (not currently installed on this dev machine, per `pacman -Qi sqlcipher`) | Available in Arch `extra`, `4.14.0-1` | `sudo pacman -S sqlcipher` before first build; document as a build-time prerequisite in the plugin's own README, mirroring how `plugins/proton` documents its Bridge-forwarder prerequisite |
| `~/.config/Signal/config.json` | Key resolution | ✓ | Legacy plaintext-`key` shape (see Runtime State Inventory) | — |
| `org.freedesktop.secrets` D-Bus service | safeStorage key unwrap path (not needed on this machine today, but must work if/when this install migrates) | ✓ (via `gnome-keyring-daemon`) | — | — |
| `signal-desktop`'s `sgnl://` scheme handler registration | "Open in Signal" deep link | ✓ (installed at `/usr/lib/signal-desktop/`) | — | — |

**Missing dependencies with no fallback:** none — the one missing item (`sqlcipher` system package) has a documented one-line fallback (install it).
**Missing dependencies with fallback:** system `sqlcipher` package — install via `pacman`/`apt` before first plugin build; not yet present on this dev machine.

## Validation Architecture

Skipped — `workflow.nyquist_validation` is explicitly `false` in `.planning/config.json`.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | No | This plugin authenticates to nothing (no username/password/API token) — it decrypts a local key |
| V3 Session Management | No | No session concept in this plugin |
| V4 Access Control | Indirect | Existing per-source `agent.read`/`agent.handoff` grant model (AGENT-01) applies unchanged; no new access-control surface introduced by this phase |
| V5 Input Validation | Yes | `config.json` field-presence validation (Pattern 1); `PRAGMA user_version` bounds check (Pattern 2); HTML sanitization of message content before rendering (bluemonday) |
| V6 Cryptography | Yes — this is the phase's core risk surface | AES-128-CBC + PBKDF2-HMAC-SHA1 via Go stdlib/`x/crypto`, using Chromium's own documented, fixed constants — never hand-rolled, never invented parameters. The SQLCipher layer itself (AES-256, per CLAUDE.md) is handled entirely by the C library, not reimplemented. |
| V12 Files and Resources | Yes | Read-only file access to `db.sqlite`/`-wal`/`-shm` and `config.json`; no path is ever derived from untrusted/remote input (all paths are fixed, local, user-owned) |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Accidental mutation of Signal Desktop's live database (a single stray `PRAGMA` or write call anywhere in the plugin) | Tampering | `mode=ro` DSN (defense layer 1) + a dedicated AST read-only scan test (mirroring `plugins/proton/readonly_test.go`'s IMAP-mutation scanner, adapted to flag any SQL string containing `INSERT`/`UPDATE`/`DELETE`/write-shaped `PRAGMA` outside test files) (defense layer 2) + the byte-identical hash test (defense layer 3, empirical proof) |
| Credential/plaintext-message disclosure via logging | Information Disclosure | Plugin contract's existing "never log a credential" rule, extended explicitly to the SQLCipher key and to raw message bodies (log presence/counts, never content, in any non-`Fetch`-response code path) |
| Wrong-key silent-garbage decryption masking as a generic DB error | Tampering / Information Disclosure (in the sense of misleading diagnostic surface) | Explicit post-decrypt validation (Pitfall 4) — attempt a trivial read and surface a named "backend mismatch" error rather than a generic SQLite error |
| Supply-chain risk from an unmerged-PR fork or untagged branch dependency | Tampering (of the build supply chain) | `checkpoint:human-verify` before pinning any non-canonical fork via `go.mod replace` (Package Legitimacy Audit); prefer the dynamic-link-to-system-package strategy specifically because it avoids this class of risk for the SQLCipher core itself |

## Sources

### Primary (HIGH confidence)
- sqlite.org/releaselog/3_51_3.html — official SQLite 3.51.3 release notes, WAL-reset corruption bug fix
- sqlite.org/wal.html — official WAL-mode documentation, read-only access semantics
- electronjs.org/docs/latest/api/safe-storage — official Electron `safeStorage` API docs, backend values and detection logic
- archlinux.org/packages/extra/x86_64/sqlcipher/ — live Arch package page, version and update-date confirmation
- go list -m -versions (Go module proxy) — direct, tool-verified version history for every Go module recommended in this document
- This machine's own `~/.config/Signal/config.json` (field names only, values never read/logged) and installed-package inspection (`pacman`, `dbus-send`, `which`, `XDG_CURRENT_DESKTOP`) — direct hands-on environment verification

### Secondary (MEDIUM confidence)
- zetetic.net/blog/2026/03/17/sqlcipher-4.14.0-release — official SQLCipher vendor blog, SQLite-baseline-rebase confirmation
- github.com/signalapp/Signal-Desktop (via search-indexed source excerpts) — `journal_mode = WAL`, `PRAGMA user_version`/`SCHEMA_VERSIONS` schema-version mechanism
- github.com/tbvdm/sigtop (ISC-licensed) — prior art confirming a real SQLCipher amalgamation (not hand-rolled WAL parsing) is the workable approach for this exact problem class
- github.com/keybase/go-keychain/secretservice — pkg.go.dev documentation, API surface and godbus/dbus foundation
- shkspr.mobi/blog/2023/02/signals-newish-uri-scheme/ + bugs.archlinux.org/task/69415 — `sgnl://` scheme scope and Arch registration-fix confirmation
- Flathub `org.signal.Signal` issues #753/#754 + Linux Mint forum thread — real-world `config.json` shape examples (`encryptedKey`/`safeStorageBackend` field format)

### Tertiary (LOW confidence)
- Various WebSearch-summarized secondary blog posts on Electron `safeStorage` internals (Gerald Chen's blog, rtfm.co.ua) — used to cross-check the AES-128-CBC/PBKDF2/fixed-IV claims against the official Electron docs, not relied on standalone
- Forum/issue-tracker excerpts giving specific Signal Desktop `user_version` numbers (50-1640 range across different reports) — directionally useful, not treated as the authoritative current ceiling (see Open Question 3)

## Metadata

**Confidence breakdown:**
- Standard stack: MEDIUM — the digest/matching/UI reuse is HIGH confidence (proven patterns), but the SQLCipher driver strategy is explicitly unresolved pending the mandatory spike (this is a finding, not a gap in this research pass)
- Architecture: HIGH — fits the existing plugin/kernel/UI contract with zero proto or DetailPane changes needed
- Pitfalls: HIGH — five of the five documented pitfalls are grounded in official docs, official vendor blogs, or this machine's own direct, hands-on verification, not speculation

**Research date:** 2026-08-03
**Valid until:** 14 days — this domain is unusually time-sensitive (SQLCipher/SQLite point releases, an actively-discussed unmerged Go PR, and Signal Desktop's own frequent schema-version bumps all move faster than this project's typical 30-day research validity window)
