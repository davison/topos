---
phase: 04-signal-conversations
plan: 02
subsystem: sync
tags: [go, cgo, sqlite, sqlcipher, signal, dbus, secretservice, safestorage, security]

# Dependency graph
requires:
  - phase: 04-signal-conversations
    provides: "04-01's working single-shape tracer — SQLCipher driver, dsn.go, schemaguard.go, keyresolve.go's stubbed safeStorage branch, plugin.go's Match/Fetch/Health skeleton"
provides:
  - "AST-enforced read-only guarantee for plugins/signal (readonly_test.go), byte-identical proof against both a fixture and the real live database (byte_identical_test.go), zero-egress proof (outbound_hosts_test.go), and a runtime SQLite version floor check (dsn.go)"
  - "Complete dual-shape key resolution: the safeStorage-wrapped branch (encryptedKey/safeStorageBackend) is now fully implemented alongside the legacy plaintext branch, dispatching strictly on the backend value Electron itself reported"
  - "Electron/Chromium os_crypt AES-128-CBC/PBKDF2-HMAC-SHA1 unwrap (safestorage_linux.go) and a freedesktop Secret Service client over an encrypted DH-AES session (secretservice.go), reusable by a future WhatsApp Desktop plugin if it needs the same scheme"
  - "Three distinct, named, actionable Health() failure causes (missing database, key-resolution failure, schema-version ceiling), each provably never leaking secret material"
affects: [04-03-signal-fetch, phase-05-whatsapp]

# Actuals (#2632)
actuals:
  tokens: 8300
  tasks: 3
  commits: 3

# Tech tracking
tech-stack:
  added:
    - "github.com/keybase/go-keychain v0.0.1 (secretservice subpackage) — freedesktop Secret Service D-Bus client, encrypted DH-AES session mode"
    - "golang.org/x/crypto v0.54.0 (pbkdf2) — PBKDF2-HMAC-SHA1 key derivation for the Electron os_crypt unwrap"
    - "github.com/keybase/dbus v0.0.0-20220506165403-5aa21ea2c23a — transitive D-Bus transport for go-keychain/secretservice"
  patterns:
    - "AST-walk read-only enforcement extended beyond identifier-selector scanning (plugins/proton's precedent) to also flag write-shaped SQL keywords hiding inside string literals — a mutation smuggled in as SQL text is caught even though it never touches a flagged Go identifier"
    - "Test-built (never committed-binary) SQLCipher fixtures: a shared buildFixtureDatabase helper creates a fresh encrypted database at test time from the same driver/DSN the plugin itself uses, parameterized by PRAGMA user_version — reused across Task 1's byte-identical test and Task 3's schema-ceiling negative control"
    - "Dispatch-only backend selection: safeStorageBackend's literal config.json value (never $XDG_CURRENT_DESKTOP or any DE probe) selects the keyring code path, with basic_text routing through the identical AES-CBC/PBKDF2 unwrap as every keyring-backed value, just with a fixed password and zero D-Bus"
    - "Three-tier openGuarded error naming (missing database / key resolution failed / schema ceiling), each independently constructed so Health()'s LastError is self-describing with no per-cause UI branching required"

key-files:
  created:
    - plugins/signal/readonly_test.go
    - plugins/signal/byte_identical_test.go
    - plugins/signal/outbound_hosts_test.go
    - plugins/signal/dsn_test.go
    - plugins/signal/testdata/README.md
    - plugins/signal/safestorage_linux.go
    - plugins/signal/safestorage_test.go
    - plugins/signal/secretservice.go
    - plugins/signal/keyresolve_test.go
    - plugins/signal/schema_version_fixture_test.go
    - plugins/signal/health_test.go
  modified:
    - plugins/signal/dsn.go
    - plugins/signal/keyresolve.go
    - plugins/signal/schemaguard.go
    - plugins/signal/plugin.go
    - plugins/signal/go.mod
    - plugins/signal/go.sum
    - go.work.sum

key-decisions:
  - "config.json's encryptedKey field is hex-encoded, not base64 — confirmed directly against signalapp/Signal-Desktop's live source (app/main.main.ts getSQLKey: `Buffer.from(modernKeyValue, 'hex')`), correcting 04-RESEARCH.md's illustrative snippet's implicit base64 assumption. The unwrapped safeStorage plaintext is used directly as the SQLCipher hex key string — Signal Desktop's own getSQLKey never re-encodes it after decrypting."
  - "The Secret Service search attribute value (`application`) is \"Signal\" — traced from Chromium's own source (components/os_crypt/{sync,async}/... both use os_crypt::Config.application_name when the embedder sets it) plus Electron's app.getName() defaulting to package.json's productName field, which signalapp/Signal-Desktop declares as \"Signal\". This machine's real install has never migrated to safeStorage (04-01-SUMMARY.md), so this specific value cannot be verified against a live install here — flagged for re-verification the first time this branch runs against a real safeStorage-migrated Signal Desktop."
  - "kwallet/kwallet5/kwallet6 backend values route through the same freedesktop Secret Service client as gnome_libsecret, per the plan's explicit instruction — not through KWallet's own native org.kde.KWallet D-Bus API, which Chromium itself actually uses for those backends. This is a known, documented scope limitation (04-RESEARCH.md Open Question 2), not an oversight: native KWallet support is out of scope for this plan."
  - "The database-not-found check in openGuarded runs before config.json is ever read, independent of key resolution — so 'Signal Desktop may not be installed for this user' is reachable even when config.json itself is also missing, rather than being masked by a config-read failure that would otherwise fire first."

patterns-established:
  - "A plugin's Health() LastError and the GET /api/sources response's last_error are architecturally distinct: Reachable (Health()'s own field) drives the health-chip dot tone via ProbeSources; last_error in the API response comes exclusively from the kernel's own recorded sync_runs history (a Match() failure), never from Health()'s LastError field directly (kernel/httpapi/sources.go's A-PLUG-04 discipline, unchanged by this plan) — confirmed identical to Phase 3's SRC-01 criterion 5 precedent."

requirements-completed: [SRC-02]

coverage:
  - id: D1
    description: "An AST scan of every non-test Go file in plugins/signal fails the build on any write-shaped SQL statement (Exec/ExecContext calls, or VACUUM/wal_checkpoint/INSERT/UPDATE/DELETE/DROP/ALTER/CREATE/REPLACE hiding inside a string literal), proven non-vacuous by two negative-control fixtures"
    requirement: "SRC-02"
    verification:
      - kind: unit
        ref: "plugins/signal/readonly_test.go — TestPluginIssuesNoWriteShapedSQL (repo scan + Exec-selector negative control + SQL-literal negative control)"
        status: pass
    human_judgment: false
  - id: D2
    description: "A fixture SQLCipher database is byte-identical (SHA-256) before and after a full Match+Fetch cycle, scoped to db.sqlite alone (never -wal/-shm); the identical assertion against the real, live ~/.config/Signal/sql/db.sqlite (opt-in) also passes with an unchanged hash"
    requirement: "SRC-02"
    verification:
      - kind: unit
        ref: "plugins/signal/byte_identical_test.go — TestDatabaseByteIdenticalAfterMatchAndFetch"
        status: pass
      - kind: e2e
        ref: "plugins/signal/byte_identical_test.go — TestLiveDatabaseByteIdentical (WEBSPACES_SIGNAL_LIVE_IT=1, run against the real database twice this session: hashes c2f698790559602f8412f882a689b78bdab16f68c59028b26b9bc365d8e6f3de and 5eac8fff48a7532067917e0b5f9160b648b35c5b3cedaca2017942a02a101e64, both unchanged before/after)"
        status: pass
      - kind: e2e
        ref: "scripts/signal-readonly-smoke.sh — run twice this session, both times unchanged hash, 1467 real items synced each time"
        status: pass
    human_judgment: false
  - id: D3
    description: "This plugin permits zero outbound network hosts — an AST scan of non-test files finds no net/http construction and no absolute network-scheme URL literal, proven non-vacuous by a negative control; internal/audit's repo-wide egress scan continues to pass with plugins/signal absent from sanctionedEgressFiles"
    requirement: "SRC-02"
    verification:
      - kind: unit
        ref: "plugins/signal/outbound_hosts_test.go — TestNoOutboundNetworkHosts"
        status: pass
      - kind: unit
        ref: "internal/audit/outbound_hosts_test.go — TestNoForeignEgressOutsideSanctionedClient"
        status: pass
    human_judgment: false
  - id: D4
    description: "The plugin refuses to run against a linked SQLite core below 3.51.3, naming the version found — checked immediately after the trivial key-proving read and before the schema guard"
    requirement: "SRC-02"
    verification:
      - kind: unit
        ref: "plugins/signal/dsn_test.go — TestSQLiteVersionFloor (against the real linked library) and TestSQLiteVersionFloor_ComparisonLogic (table-driven)"
        status: pass
    human_judgment: false
  - id: D5
    description: "The safeStorage-wrapped key shape (encryptedKey/safeStorageBackend) is fully resolved: the Electron os_crypt AES-128-CBC/PBKDF2-HMAC-SHA1 unwrap round-trips on known-good v10/v11 fixtures and rejects a wrong password via the PKCS7 padding check; resolveKey dispatches strictly on the literal safeStorageBackend value (gnome_libsecret/kwallet family to Secret Service, basic_text to the fixed password with zero D-Bus, anything else fails naming the value)"
    requirement: "SRC-02"
    verification:
      - kind: unit
        ref: "plugins/signal/safestorage_test.go — TestSafeStorage_V10RoundTrip, TestSafeStorage_V11RoundTrip, TestSafeStorage_MissingPrefixRejected, TestSafeStorage_NonBlockMultipleCiphertextRejected, TestSafeStorage_WrongPasswordRejectedByPaddingCheck, TestSafeStorage_ConstantsAreChromiumsOwn"
        status: pass
      - kind: unit
        ref: "plugins/signal/keyresolve_test.go — TestResolveKey_LegacyKeyOnly, TestResolveKey_NeitherFieldPresent, TestResolveKey_BothFieldsPresent, TestResolveKey_BasicTextNeverTouchesDBus, TestResolveKey_RoutesToSecretServiceForKeyringBackends, TestResolveKey_UnrecognisedBackend"
        status: pass
      - kind: e2e
        ref: "scripts/signal-readonly-smoke.sh — the legacy plaintext-key branch (this machine's real, only exercisable shape) still works end to end after the safeStorage branch landed"
        status: pass
    human_judgment: false
  - id: D6
    description: "No error message anywhere in the key-resolution path contains a key, ciphertext, or password value"
    requirement: "SRC-02"
    verification:
      - kind: unit
        ref: "plugins/signal/safestorage_test.go#TestSafeStorage_ErrorsNeverContainSecretMaterial, plugins/signal/keyresolve_test.go#TestResolveKey_ErrorsNeverContainSecretMaterial"
        status: pass
    human_judgment: false
  - id: D7
    description: "guardSchemaVersion is proven non-vacuous: a fixture one version above the ceiling fails naming both the version found and the ceiling; fixtures at and below the ceiling pass"
    requirement: "SRC-02"
    verification:
      - kind: unit
        ref: "plugins/signal/schema_version_fixture_test.go — TestSchemaVersionCeiling (above/at/below-ceiling subtests)"
        status: pass
    human_judgment: false
  - id: D8
    description: "Each of the three Signal Health() failure causes (missing database, key-resolution failure, schema-version ceiling) returns Reachable:false with an actionable, self-describing LastError and a nil Go error; a healthy fixture returns Reachable:true; no LastError leaks the fixture key or a message body"
    requirement: "SRC-02"
    verification:
      - kind: unit
        ref: "plugins/signal/health_test.go — TestHealth_MissingDatabase, TestHealth_KeyResolutionFailure, TestHealth_SchemaVersionCeiling, TestHealth_Healthy, TestHealth_NeverLeaksSecretMaterial"
        status: pass
      - kind: e2e
        ref: "Isolated throwaway config (missing Signal path) against real binaries: GET /api/sources reported reachable:false and last_error naming the missing database path and suggesting Signal Desktop may not be installed"
        status: pass
    human_judgment: false
  - id: D9
    description: "No frontend change is required for the three Signal failure causes to surface through the existing health chip — Reachable drives the dot tone via the existing ProbeSources merge"
    requirement: "SRC-02"
    verification:
      - kind: manual_procedural
        ref: "Read web/src/lib/components/SourceHealthChip.svelte and web/src/lib/api.ts; confirmed git diff --stat web/ is empty and npm --prefix web run check reports 0 errors"
        status: pass
    human_judgment: true
    rationale: "The exact tooltip-copy behavior for the 'destructive' (unreachable) tone is an existing, Phase 2-established architectural choice (last_error text renders only in the separate 'warning'-tone tooltip variant, sourced from sync_runs rather than Health()'s own LastError) — confirming this matches Phase 3's own established precedent for the identical criterion is a judgment call about UI architecture continuity, not something a unit test proves."

# Metrics
duration: ~2h
completed: 2026-08-03
status: complete
---

# Phase 04 Plan 02: Read-only by construction, complete key resolution, self-naming failures Summary

**Signal plugin's read-only/backend-detection/fail-loud guarantees are now mechanically enforced by tests (AST scans with negative controls, a real byte-identical proof against the live database) rather than demonstrated once by hand, and the safeStorage-wrapped key branch (Electron os_crypt AES-128-CBC/PBKDF2 unwrap + freedesktop Secret Service over an encrypted DH-AES session) is fully implemented and dispatches strictly on the declared backend value**

## Performance

- **Duration:** ~2h
- **Tasks:** 3 (all `type="auto" tdd="true"`)
- **Files modified:** 18 (11 created, 7 modified — see `key-files` above)

## Accomplishments
- Turned 04-01's three "must never go wrong" guarantees from hand-demonstrated into mechanically enforced: an AST scanner rejects any future write-shaped SQL (identifier or string-literal-hidden) with a proven-non-vacuous negative control; a fixture-database Match+Fetch cycle proves byte-identity, and the identical proof against the real, live 203 MB `~/.config/Signal/sql/db.sqlite` was run twice this session with an unchanged hash both times; a zero-outbound-hosts egress scanner covers the plugin.
- Completed the safeStorage-wrapped key branch 04-01 stubbed: the Electron/Chromium `os_crypt` AES-128-CBC/PBKDF2-HMAC-SHA1 unwrap (constants traced verbatim from Chromium's own source, read live during this task) and a freedesktop Secret Service client over an encrypted DH-AES session, dispatching strictly on the literal `safeStorageBackend` config.json value.
- Added a runtime SQLite version floor check (>= 3.51.3) that fails loudly naming the version found, closing 04-RESEARCH.md's assumption A2.
- Made the schema-version ceiling guard's non-vacuousness mechanical (above/at/below-ceiling fixture cases) and gave each of the three Signal `Health()` failure causes a distinct, actionable, self-describing message — verified end-to-end against real binaries.

## Task Commits

Each task was committed atomically:

1. **Task 1: Read-only by construction — AST scan, byte-identical proof, zero egress, version floor** - `aadddf5` (test)
2. **Task 2: Key resolution on any install — safeStorage unwrap driven by the declared backend** - `e6ad488` (feat)
3. **Task 3: Failures that name themselves — schema ceiling negative control and three-cause health** - `1c14dc5` (feat)

**Plan metadata:** (this commit)

_Note: all three tasks are `tdd="true"`; each behaves as a single cohesive commit per task (tests + implementation together), matching 04-01-SUMMARY.md's own precedent for TDD tasks in this plugin — no separate RED/GREEN split was warranted since each task's `<behavior>` block described a small, cohesive unit landing together._

## Files Created/Modified

- `plugins/signal/readonly_test.go` - AST scan (Exec/ExecContext selectors + write-shaped SQL string literals, VACUUM/wal_checkpoint named explicitly), two negative controls
- `plugins/signal/byte_identical_test.go` - `buildFixtureDatabase` shared helper, `TestDatabaseByteIdenticalAfterMatchAndFetch`, opt-in `TestLiveDatabaseByteIdentical`
- `plugins/signal/outbound_hosts_test.go` - Zero-outbound-hosts AST scan with a negative control
- `plugins/signal/dsn.go` - `buildReadOnlyDSN` factored out; `checkSQLiteVersionFloor`/`parseSQLiteVersion`/`compareSQLiteVersions` added, called right after the trivial key-proving read
- `plugins/signal/dsn_test.go` - DSN-shape assertions, SQLite floor assertions (real library + table-driven)
- `plugins/signal/testdata/README.md` - Explains the test-time-only fixture-build convention (no committed binary fixture)
- `plugins/signal/safestorage_linux.go` - `decryptSafeStorageBlob`/`deriveOSCryptKey`/`pkcs7Unpad`, Chromium's exact os_crypt constants
- `plugins/signal/safestorage_test.go` - v10/v11 round-trip, missing-prefix/non-block-multiple/wrong-password rejection, constants-are-Chromium's-own assertion
- `plugins/signal/secretservice.go` - `secretServicePassword`: freedesktop Secret Service round-trip over `AuthenticationDHAES`
- `plugins/signal/keyresolve.go` - `resolveSafeStorageKey` completes the branch: hex-decode, backend dispatch, length validation, `errSafeStorageBackendMismatch`
- `plugins/signal/keyresolve_test.go` - Legacy/neither/both/basic_text/keyring-backend-routing/unrecognised-backend/no-secret-leak cases
- `plugins/signal/schemaguard.go` - Ceiling comment now names the verified Signal Desktop version (8.21.0) and date
- `plugins/signal/schema_version_fixture_test.go` - Above/at/below-ceiling negative control
- `plugins/signal/plugin.go` - `openGuarded` now distinguishes its three failure causes with named, actionable messages
- `plugins/signal/health_test.go` - Four `Health()` cases plus a no-secret-leak case
- `plugins/signal/go.mod`, `plugins/signal/go.sum`, `go.work.sum` - New dependencies: `keybase/go-keychain`, `golang.org/x/crypto`, `keybase/dbus` (transitive)

## Decisions Made

See `key-decisions` in frontmatter for the four load-bearing ones (encryptedKey is hex not base64; the Secret Service application attribute value; kwallet routing through Secret Service rather than native KWallet by plan design; database-not-found checked independently of config.json). All four are documented inline in the corresponding source file's own comments, not just here.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] config.json's encryptedKey field is hex-encoded, not base64**
- **Found during:** Task 2, while implementing `resolveSafeStorageKey`
- **Issue:** 04-RESEARCH.md's Code Examples section did not specify encryptedKey's encoding explicitly, and the natural default assumption (base64, matching most "encrypted blob in JSON" conventions) would have been wrong.
- **Fix:** Traced Signal Desktop's own live source (`app/main.main.ts`'s `getSQLKey`) via Sourcegraph, confirming `Buffer.from(modernKeyValue, 'hex')` — encryptedKey is hex. `resolveSafeStorageKey` hex-decodes accordingly, documented inline with the exact source reference.
- **Files modified:** `plugins/signal/keyresolve.go` (doc comment records the finding)
- **Verification:** `keyresolve_test.go`'s round-trip tests build fixtures with `hex.EncodeToString` and pass
- **Committed in:** `e6ad488` (Task 2 commit)

**2. [Rule 3 - Blocking] `go mod tidy` fails inside a go.work workspace member depending on unpublished sibling modules**
- **Found during:** Task 2, after `go get`-ing the new dependencies
- **Issue:** `go mod tidy` run directly inside `plugins/signal/` tried to resolve `github.com/davison/webspaces/sdk` (a local workspace member, never published) as a remote module and failed with a "repository not found" error.
- **Fix:** Skipped `go mod tidy`; instead let `go build`/`go get` populate `go.mod`/`go.sum` directly (which correctly use `go.work`'s local module resolution), then manually removed the stale `// indirect` markers `go get` had left on the two directly-imported packages.
- **Files modified:** `plugins/signal/go.mod`
- **Verification:** `CGO_ENABLED=1 go build -tags libsqlcipher ./...` succeeds; `make test-signal` passes
- **Committed in:** `e6ad488` (Task 2 commit)

**3. [Rule 3 - Blocking] `TestResolveKey_*` fixtures initially used a 60-character (not 64-character) test key**
- **Found during:** Task 2, first test run
- **Issue:** A hand-typed hex literal came up 4 characters short of `expectedRawKeyHexLen` (64), tripping the length-validation check the test was meant to exercise successfully.
- **Fix:** Replaced the hand-counted literal with `strings.Repeat("ab", expectedRawKeyHexLen/2)`, so the fixture's length is correct by construction and can never silently drift again.
- **Files modified:** `plugins/signal/keyresolve_test.go`
- **Verification:** All `TestResolveKey_*` cases pass
- **Committed in:** `e6ad488` (Task 2 commit)

---

**Total deviations:** 3 auto-fixed (1 Rule 1 bug from live-source ground-truth, 2 Rule 3 blocking/tooling fixes)
**Impact on plan:** All three were necessary corrections discovered while implementing the plan's own explicit instructions ("resolve unclear details against ground truth, record the difference"). No scope creep — every fix stayed inside `plugins/signal/`'s Task 2 file list.

## Issues Encountered

None beyond the deviations above. Both live-database verification runs this session (`WEBSPACES_SIGNAL_LIVE_IT=1 go test`/`make test-signal`, and `scripts/signal-readonly-smoke.sh` run twice) passed with an unchanged hash each time against the real, live 203 MB database, with Signal Desktop running throughout.

## User Setup Required

None — this plan added no new external service dependency. The two new Go module dependencies (`github.com/keybase/go-keychain`, `golang.org/x/crypto`) are pulled automatically by `go build`/`make test-signal`; no system package or environment variable is required beyond what 04-01 already documented.

## Checkpoint Evidence

No `checkpoint:*` tasks in this plan (`autonomous: true`, all three tasks `type="auto"`). The two `<human-check>` verify items in Tasks 2 and 3 were performed directly by this executor rather than deferred to the user, consistent with this being a sequential autonomous run:

- **Task 2's human-check** (`./scripts/signal-readonly-smoke.sh` still passes on this machine's legacy plaintext-key install): ran twice this session — 1467 real digests synced each time, `db.sqlite` hash unchanged both times.
- **Task 3's human-check** (a Signal source pointed at a directory with no install shows the destructive-tone dot with an actionable tooltip): rather than editing the developer's real `~/.config/webspaces/config.toml` (unnecessary risk to the live deployment), verified via an isolated throwaway `XDG_CONFIG_HOME` pointing `[sources.signal] path` at a nonexistent directory, against the real (rebuilt) `bin/webspaces` and `bin/plugins/webspaces-plugin-signal` binaries: `GET /api/sources` returned `"reachable":false` with `"last_error"` reading *"...signal: Signal Desktop's database was not found at /tmp/.../sql/db.sqlite — Signal Desktop may not be installed for this user, or has not been run yet..."* — confirming the message end to end through the real kernel↔plugin gRPC boundary.

## Next Phase Readiness

- All three of ROADMAP.md's "must never go wrong" success criteria (read-only by construction, backend-detected key resolution, fail-loud schema versioning) are now mechanically enforced by tests rather than demonstrated once by hand — a future edit that reintroduces a write path, an assumed keyring backend, or a silent schema skip fails `make test-signal` before ever reaching the user's database.
- The safeStorage branch is implemented and unit-tested against known-good fixtures, but — as 04-RESEARCH.md flagged from the start — this machine's real Signal Desktop install has never migrated to safeStorage, so the branch has never been exercised against a real safeStorage-migrated `config.json` or a real freedesktop Secret Service round-trip carrying Signal Desktop's actual secret. This is an inherent limitation of this specific development machine, not a gap in the implementation; **04-03 or a future phase should flag this for re-verification the first time a safeStorage-migrated install is available.**
- **04-03** (next plan) implements `Fetch`'s `FULL`/`PREVIEW` variants (currently `Available: false` stubs) and the transcript renderer's bluemonday sanitisation policy — unaffected by this plan's changes.
- No blockers identified.

---
*Phase: 04-signal-conversations*
*Completed: 2026-08-03*

## Self-Check: PASSED

All 18 files listed under `key-files` were verified present on disk, and all three task commits (`aadddf5`, `e6ad488`, `1c14dc5`) were verified present in `git log --oneline --all`. No missing items.
