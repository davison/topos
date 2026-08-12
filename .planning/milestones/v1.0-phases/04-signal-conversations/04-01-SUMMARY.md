---
phase: 04-signal-conversations
plan: 01
subsystem: sync
tags: [go, cgo, sqlite, sqlcipher, signal, grpc, plugin-architecture]

# Dependency graph
requires:
  - phase: 02-plugin-architecture
    provides: plugin.proto contract, kernel/config.Source shape, kernel/pluginhost launch/env-marshal pattern, agent grant block convention
provides:
  - "First cgo plugin in the repo (plugins/signal), proving read-only SQLCipher access to a live, actively-written Signal Desktop database is safe"
  - "kernel/config.Source.Path — the local-path source shape (no base_url/token) other local-path sources (e.g. future WhatsApp) can reuse"
  - "Conversation-day digest unit (D-01): one item per (conversation, local calendar day) with activity, deterministic source_id, sender-prefixed tail preview"
  - "Dual-shape Signal key resolution branch point (legacy plaintext key implemented; safeStorage-wrapped branch stubbed for 04-02)"
affects: [04-02-signal-safestorage, 04-03-signal-fetch, phase-05-whatsapp]

# Actuals (#2632)
actuals:
  tokens: 9200
  tasks: 2
  commits: 1

# Tech tracking
tech-stack:
  added:
    - "github.com/mattn/go-sqlite3 v1.14.49, replaced via go.mod replace to jgiannuzzi/go-sqlite3's sqlcipher branch (commit f208443ec79de7edaf1b80276806005a5c0cf340) — dynamically links the system sqlcipher library"
  patterns:
    - "Local-path source config shape: Source.Path (toml:\"path,omitempty\"), validated as an alternative to base_url+token, marshalled into WEBSPACES_SOURCE_CONFIG's JSON map under \"path\""
    - "Dual-shape credential resolution branching strictly on JSON field presence (never assuming a shape), fail-loud when neither or both are present"
    - "PRAGMA user_version ceiling guard read live off the real target database at implementation time, never carried over from research-doc placeholders"
    - "Conversation-day digest source_id: base64.RawURLEncoding(conversationID + \":\" + day) — deterministic upsert key mirroring plugins/proton's encodeSourceID/decodeSourceID shape"

key-files:
  created:
    - plugins/signal/go.mod
    - plugins/signal/main.go
    - plugins/signal/dsn.go
    - plugins/signal/schemaguard.go
    - plugins/signal/keyresolve.go
    - plugins/signal/plugin.go
    - plugins/signal/match.go
    - plugins/signal/digest.go
    - plugins/signal/deeplink.go
    - plugins/signal/digest_test.go
    - plugins/signal/match_test.go
    - scripts/signal-readonly-smoke.sh
  modified:
    - go.work
    - Makefile
    - config.example.toml
    - kernel/config/types.go
    - kernel/config/config.go
    - kernel/config/config_test.go
    - kernel/pluginhost/host.go
    - web/src/lib/api.ts

key-decisions:
  - "Task 1 checkpoint: option-a selected — dynamically link the system SQLCipher via a libsqlcipher-tagged mattn/go-sqlite3 fork, pinned by go.mod replace to github.com/jgiannuzzi/go-sqlite3 v1.14.17-0.20230327162135-f208443ec79d (branch sqlcipher, upstream PR mattn/go-sqlite3#1109, commit f208443ec79de7edaf1b80276806005a5c0cf340, verified live against the GitHub API 2026-08-03). Dynamically links Arch system sqlcipher 4.14.0-1 (SQLite 3.51.3 baseline), meeting the phase's >=3.51.3 WAL-corruption-fix floor. Explicitly authorised by the developer this session, superseding CLAUDE.md's mutecomm/go-sqlcipher/v4 pin for this plugin only (that package bundles a pre-3.51.3 SQLite core and fails the phase's own safety floor)."
  - "PRAGMA user_version ceiling pinned to 1730, read live off the real ~/.config/Signal/sql/db.sqlite during schema introspection — not carried over from 04-RESEARCH.md's unconfirmed placeholder value of 1640."
  - "DSN parameter names diverge from 04-RESEARCH.md's illustrative snippet: the selected driver uses _key=x'<hex>' and _cipher_page_size, not _pragma_key/_pragma_cipher_page_size — confirmed against the real database during Task 2."

patterns-established:
  - "cgo plugins get their own go.work module and a Makefile target that carries the build tag (here `sqlcipher`) so the tag lives in exactly one seam; the kernel's own build stays CGO_ENABLED=0."
  - "1:1 conversation matching reads only the user's own name fields for a contact (nickname, system contact name) — the contact's self-chosen profile name and any derived title field that falls back to it are never match candidates, enforced by a named unit test case (D-06)."

requirements-completed: [SRC-02]

coverage:
  - id: D1
    description: "Signal conversation-day digests (one item per conversation per calendar day with activity) appear in the webspace stream, interleaved with other sources, titled with correct singular/plural grammar and a sender-prefixed 2-3 line preview"
    requirement: "SRC-02"
    verification:
      - kind: unit
        ref: "plugins/signal/digest_test.go — title grammar, tail snippet, day grouping, timestamp = last message cases"
        status: pass
      - kind: e2e
        ref: "scripts/signal-readonly-smoke.sh (1467 real digests synced against live ~/.config/Signal/sql/db.sqlite)"
        status: pass
      - kind: manual_procedural
        ref: "Human-verify checkpoint: browser check of the isolated signal-smoke webspace at 127.0.0.1:7777"
        status: pass
    human_judgment: false
  - id: D2
    description: "1:1 conversation matching excludes the contact's self-chosen profile name (D-06) — a contact cannot pull themselves into a webspace by renaming their own profile"
    requirement: "SRC-02"
    verification:
      - kind: unit
        ref: "plugins/signal/match_test.go — 1:1 conversation whose ONLY matching name field is the contact's self-chosen profile name does NOT match"
        status: pass
    human_judgment: false
  - id: D3
    description: "~/.config/Signal/sql/db.sqlite is byte-identical (SHA-256) before and after a full webspaces sync, including while Signal Desktop is running"
    requirement: "SRC-02"
    verification:
      - kind: e2e
        ref: "scripts/signal-readonly-smoke.sh — hash before/after comparison, SHA-256 f1a0d108e3f07d76fd47a60ae4fbf3388fc1b1113811214dedadf3d00030c6cf unchanged; re-run independently by the orchestrator after the checkpoint, hash unchanged again"
        status: pass
    human_judgment: false
  - id: D4
    description: "Kernel config accepts a local-path source (plugin + path, no base_url/token) and fails loudly naming both accepted shapes when a source declares neither"
    requirement: "SRC-02"
    verification:
      - kind: unit
        ref: "kernel/config/config_test.go — path-only source validates, base_url+token source still validates, neither-shape source fails naming both shapes"
        status: pass
    human_judgment: false
  - id: D5
    description: "Group deep links raise Signal Desktop via the bare sgnl:// scheme without navigating to the specific chat — designed conversation-only fidelity, not a defect"
    requirement: "SRC-02"
    verification: []
    human_judgment: true
    rationale: "Developer explicitly observed and accepted this behavior during the human-verify checkpoint as the documented, locked (04-CONTEXT.md) fidelity limit for group deep links; no automated check can distinguish 'intentional limitation' from 'defect' for a third-party app's URL scheme."

# Metrics
duration: ~3h (across original execution session + this close-out continuation)
completed: 2026-08-03
status: complete
---

# Phase 04 Plan 01: Signal conversation-day digest tracer Summary

**Signal conversation-day digests end to end against a live SQLCipher database, via a dynamically-linked libsqlcipher fork pinned by go.mod replace, with the config-relaxed kernel accepting local-path sources**

## Performance

- **Duration:** ~3h (original executor session, paused at final human-verify checkpoint; this continuation closes out documentation/state only)
- **Tasks:** 2 (Task 1 checkpoint:decision, Task 2 tracer)
- **Files modified:** 20 (per plan frontmatter `files_modified`)

## Accomplishments
- Proved the whole Signal vertical end to end on one thin path: kernel accepts a local-path source with no base_url/token; the Signal plugin opens Signal Desktop's live SQLCipher database strictly read-only; refuses an unrecognised schema version by name; resolves the decryption key from the legacy plaintext-key config.json shape; turns matched conversations' message history into conversation-day digests visible in the webspace stream alongside documents, notes and mail — database byte-identical afterwards.
- Closed the kernel config gap deferred from Phase 02-04: `kernel/config.Source.Path` plus relaxed `Validate` logic now accept a genuinely configless local-path source, with an error naming both accepted shapes (`base_url`+`token` or `path`) when neither is declared.
- Authorised and pinned the first cgo/non-canonical-fork dependency in the repo via an explicit human checkpoint (Task 1), with the full supply-chain audit trail recorded in `go.mod`'s replace-directive comment and here.
- Established D-05/D-06 1:1 matching discipline (own-name-only candidates, profile name never a candidate) with a load-bearing unit test proving the excluded case.

## Task Commits

Each task was committed atomically:

1. **Task 1: Lock the SQLCipher driver and link strategy** - decision-only checkpoint, no files modified (see Decisions Made below)
2. **Task 2: End-to-end Signal conversation-day digest** - `fd6e455` (feat) — "Signal conversation-day digests end to end (SRC-02 tracer)"

**Plan metadata:** (this commit)

_Note: Task 2 is `tdd="true"`; digest_test.go and match_test.go were authored table-driven before their implementations per the plan's `<behavior>` block, folded into the single tracer commit rather than split into separate RED/GREEN commits, consistent with the tracer task type._

## Files Created/Modified

- `plugins/signal/go.mod` - New cgo module; `replace github.com/mattn/go-sqlite3 => github.com/jgiannuzzi/go-sqlite3 v1.14.17-...` per the Task 1 decision
- `plugins/signal/main.go` - Decodes `WEBSPACES_SOURCE_CONFIG` (single `path` field), expands `~`, serves via `goplugin.Serve`
- `plugins/signal/dsn.go` - `mode=ro` URI DSN with `_key=x'<hex>'` and `_cipher_page_size=4096`, deliberately no `immutable=1`, `SetMaxOpenConns(1)`, key-proving trivial read
- `plugins/signal/schemaguard.go` - `PRAGMA user_version` ceiling guard, constant `highestSupportedSchemaVersion = 1730`
- `plugins/signal/keyresolve.go` - Dual-shape key resolution (`Key` vs `EncryptedKey`+`SafeStorageBackend`), legacy branch fully implemented, safeStorage branch stubbed with `errSafeStorageUnsupported` for 04-02
- `plugins/signal/plugin.go` - `Describe`/`Match`/`Fetch`/`Health`, `buildSenderNames`, live schema query against `conversations`/`messages` with `sourceServiceId` and JSON-blob fallback fields
- `plugins/signal/match.go` - `conversation` struct, `matchesAnyKeyword`, `candidateNames` (D-05/D-06), `eligibleConversations`, `conversationDisplayName`
- `plugins/signal/digest.go` - `message`/`digest` structs, `buildDigests`, `sourceIDForDigest`/`decodeSourceID`, `digestTitle`, `tailSnippet`, `Snippet`
- `plugins/signal/deeplink.go` - `sgnl://` deep-link construction at conversation-only fidelity
- `plugins/signal/digest_test.go`, `plugins/signal/match_test.go` - Table-driven unit tests written before their implementations (TDD)
- `scripts/signal-readonly-smoke.sh` - Hash-before/sync/hash-after/serve/poll/assert end-to-end guard
- `kernel/config/types.go` - New `Source.Path` field (`toml:"path,omitempty"`)
- `kernel/config/config.go` - `Validate` relaxed: path-only sources skip base_url/token requirement; neither-shape sources fail naming both accepted shapes
- `kernel/config/config_test.go` - Three new table cases (path-only valid, base_url+token still valid, neither-shape fails naming both)
- `kernel/pluginhost/host.go` - `"path": src.Path` added to the `WEBSPACES_SOURCE_CONFIG` JSON marshal
- `config.example.toml` - Documented `[sources.signal]` (plugin, path, no base_url/token/env-var note) and `[sources.signal.agent]` grant block
- `web/src/lib/api.ts` - `signal: 'Signal'` added to `SOURCE_DISPLAY_NAMES`
- `go.work`, `Makefile` - `./plugins/signal` workspace member; `signal`/`test-signal` targets carrying the `sqlcipher` build tag, invoked from `build`/`test`

## Decisions Made

**Task 1 checkpoint decision (supply-chain audit trail):**

Selected **option-a**: dynamically link the system SQLCipher via a libsqlcipher-tagged `mattn/go-sqlite3` fork.

- Pin: `go.mod replace github.com/mattn/go-sqlite3 => github.com/jgiannuzzi/go-sqlite3 v1.14.17-0.20230327162135-f208443ec79d`
- Fork branch: `sqlcipher`, tracking upstream unmerged PR `mattn/go-sqlite3#1109`
- Pinned commit: `f208443ec79de7edaf1b80276806005a5c0cf340`, verified live against the GitHub API on 2026-08-03 as the live PR head at authorisation time
- Dynamically links Arch system package `sqlcipher` 4.14.0-1, which carries a SQLite 3.51.3 baseline — meeting the phase's `>= 3.51.3` WAL-reset-corruption-fix floor
- Rejected `mutecomm/go-sqlcipher/v4` (the driver named in `CLAUDE.md`) because it last released `v4.4.2` in 2020-12 and statically bundles a pre-3.51.3 SQLite core, failing the phase's own safety floor against a live, actively-written WAL database
- Explicitly authorised by the developer this session as a supply-chain commitment to a single-maintainer, untagged fork — `internal/audit/module_pins_test.go` guards the dependency floor thereafter

Other decisions carried in the Deviations section below.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] DSN parameter names corrected from 04-RESEARCH.md's illustrative snippet**
- **Found during:** Task 2 (dsn.go implementation)
- **Issue:** 04-RESEARCH.md's Pattern 3 snippet used `_pragma_key`/`_pragma_cipher_page_size`, which are `mutecomm/go-sqlcipher`'s DSN convention — not the Task 1-selected `mattn/go-sqlite3`-family driver's convention. Written before Task 1's driver selection was known.
- **Fix:** Used the correct parameter names for the selected driver: `_key=x'<hex>'` and `_cipher_page_size`. Confirmed live against the real database — the unquoted hex form failed with "file is not a database" (SQLCipher misinterprets an unquoted string as a passphrase and runs it through KDF, silently deriving the wrong key from an already-raw key); the `x'...'` raw-key-literal form opened correctly.
- **Files modified:** `plugins/signal/dsn.go` (documented inline in the function's doc comment)
- **Verification:** `./scripts/signal-readonly-smoke.sh` opens the real database successfully with this DSN shape
- **Committed in:** `fd6e455` (Task 2 commit)

**2. [Rule 1 - Bug] `PRAGMA user_version` ceiling corrected from an unconfirmed research placeholder**
- **Found during:** Task 2 (schema introspection step)
- **Issue:** 04-RESEARCH.md's snippet illustrated `1640` as the schema ceiling but explicitly flagged it as never independently confirmed against a real install.
- **Fix:** Read `PRAGMA user_version` live off the real `~/.config/Signal/sql/db.sqlite` and pinned `highestSupportedSchemaVersion = 1730` to that live value, with a doc comment stating raising it later is a deliberate, re-verified act.
- **Files modified:** `plugins/signal/schemaguard.go`
- **Verification:** Guard permits the real database (found == ceiling); `go test ./...` schema-ceiling test cases pass
- **Committed in:** `fd6e455` (Task 2 commit)

**3. [Rule 1 - Bug] Sender-name fallback fixed for outgoing 1:1 messages**
- **Found during:** Task 2, live verification against the real database
- **Issue:** Signal Desktop's schema leaves `sourceServiceId` empty on a 1:1 conversation's own outgoing messages (there is no "from" party recorded for messages the user sent in a private chat). The naive lookup against `senderNames` (keyed by service id) misreported these as `unknownSenderName` ("Unknown") instead of the user's own messages.
- **Fix:** `readMessages` in `plugins/signal/plugin.go` now special-cases an empty `sourceServiceId` in a 1:1/outgoing context to the fixed label `"You"`, falling back to `unknownSenderName` only when a service id is present but genuinely unrecognised.
- **Files modified:** `plugins/signal/plugin.go`
- **Verification:** Live smoke run against the real database shows outgoing messages correctly prefixed "You:" in tail previews
- **Committed in:** `fd6e455` (Task 2 commit)

**4. [Rule 1 - Bug] Real schema column names diverge from 04-RESEARCH.md's illustrative field list**
- **Found during:** Task 2, mandated schema-introspection step (`PRAGMA table_info(conversations)`, `SELECT json FROM conversations LIMIT 1`)
- **Issue:** The `conversations` table has no SQL columns named `systemGivenName`/`systemFamilyName`/`nicknameGivenName`/`nicknameFamilyName` — these live only inside the row's `json` blob column, not as first-class SQL columns as the research doc's illustrative shape suggested.
- **Fix:** `match.go`'s D-06 name matching reads these fields out of the parsed JSON blob (`plugin.go`'s `readConversations`), never from a `name` SQL column for 1:1 conversations — defense in depth against ever accidentally reading the wrong name source for a 1:1.
- **Files modified:** `plugins/signal/plugin.go`, `plugins/signal/match.go` (doc comments record the real schema shape)
- **Verification:** `match_test.go`'s D-06 case (profile-name-only conversation does not match) passes against the real field-sourcing logic
- **Committed in:** `fd6e455` (Task 2 commit)

**5. [Rule 3 - Blocking, precedent-following] No `plugins/signal/go.sum` committed**
- **Found during:** Task 2, module setup
- **Issue:** A cgo module with a `go.mod replace` directive to a fork raised the question of whether a `go.sum` should be checked in.
- **Fix:** Followed the existing `plugins/mock` precedent in this repo — no local `go.sum`; `go.work.sum` at the workspace root carries checksums centrally, and this local module has no publishable remote target requiring its own sum file.
- **Files modified:** none (absence is the fix)
- **Verification:** `CGO_ENABLED=0 go build ./...` and `make test-signal` both succeed without a local `go.sum`
- **Committed in:** `fd6e455` (Task 2 commit)

---

**Total deviations:** 5 auto-fixed (4 Rule 1 bugs surfaced by live verification against real data, 1 Rule 3 precedent-following non-change)
**Impact on plan:** All five were necessary corrections against ground truth the plan itself flagged as "follow reality and record the difference" (research-doc placeholders for DSN params and schema ceiling) or genuine bugs only visible against the real, live 194 MB database. No scope creep — every fix stayed inside `plugins/signal/`'s Task 2 file list.

## Issues Encountered

None beyond the deviations above — the tracer's `<verify>` block (`CGO_ENABLED=0 go build ./...`, `go test ./kernel/config/... ./internal/audit/...`, `make test-signal`, `npm --prefix web run check`, `./scripts/signal-readonly-smoke.sh`) passed repeatedly against the real live database with Signal Desktop running, and the orchestrator independently re-ran the smoke script after the checkpoint with an unchanged hash.

## User Setup Required

Two `user_setup` items from the plan's frontmatter were completed this session, authorised by the developer:
- `sqlcipher` system package installed via `sudo pacman -S sqlcipher` (Arch, resolved to 4.14.0-1)
- `[sources.signal]` + `[sources.signal.agent]` block appended to the real `~/.config/webspaces/config.toml`, mirroring the documented block in `config.example.toml`; no secret or environment variable required

No further external service configuration is required — Signal Desktop's own config.json and keyring supply everything else at runtime.

## Checkpoint Evidence

**Automated verification (all passed, repeated against the real live database with Signal Desktop running):**
- `CGO_ENABLED=0 go build ./...`
- `go test ./kernel/config/... ./internal/audit/...`
- `make test-signal` (17 unit tests)
- `npm --prefix web run check` (0 errors)
- `./scripts/signal-readonly-smoke.sh` — 1467 real digests synced; `db.sqlite` SHA-256 `f1a0d108e3f07d76fd47a60ae4fbf3388fc1b1113811214dedadf3d00030c6cf` unchanged before/after
- Orchestrator independently re-ran the smoke script after the human-verify checkpoint: passed again, hash unchanged

**Human-verify checkpoint: APPROVED.** Developer confirmed in-browser (isolated `signal-smoke` webspace at `127.0.0.1:7777`): digest row titles carry correct singular/plural grammar, sender-prefixed 2-line previews render, rows use the standard dark-theme `StreamRow` component unmodified, and an "Open in Signal" button is present on every row.

**One observation, explicitly accepted as designed behavior, not a defect:** clicking "Open in Signal" on a GROUP conversation raises Signal Desktop but does not navigate to the specific chat. This is the intended conversation-only fidelity limit for groups (Signal has no group deep-link URL form; the bare `sgnl://` scheme is the honest maximum), locked in `04-CONTEXT.md`, and explicitly accepted by the developer during checkpoint review.

## Next Phase Readiness

- The plugin, config, and Makefile plumbing this plan built are the foundation the next two plans in this phase extend, not replace:
  - **04-02** completes the `EncryptedKey`/`SafeStorageBackend` (modern Electron `safeStorage`) key-resolution branch that `keyresolve.go` currently stubs with `errSafeStorageUnsupported`
  - **04-03** implements `Fetch`'s `FULL`/`PREVIEW` variants (currently `Available: false` stubs) and the transcript renderer's bluemonday sanitisation policy
- No blockers identified. The riskiest unknown this phase carried — reading a live third-party Electron app's SQLCipher database without ever touching it — is now proven against the user's real 194 MB database, de-risking both remaining plans.

---
*Phase: 04-signal-conversations*
*Completed: 2026-08-03*

## Self-Check: PASSED

All files listed in this plan's `files_modified` under `plugins/signal/` were verified present on disk, and commit `fd6e455` was verified present in `git log --oneline --all`. No missing items.
</content>
