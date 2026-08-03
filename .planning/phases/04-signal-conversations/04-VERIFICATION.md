---
phase: 04-signal-conversations
verified: 2026-08-03T21:00:00Z
status: human_needed
score: 5/5 truths verified (ROADMAP success criteria); 2 items routed to human verification
behavior_unverified: 0
overrides_applied: 0
human_verification:
  - test: "Click the 'Open in Signal' button on a Signal digest for a 1:1 conversation whose contact has an E.164 number (the sgnl://signal.me/#p/<e164> contact-form deep link), and confirm Signal Desktop visibly raises/focuses on that contact's conversation."
    expected: "Signal Desktop window raises or gains focus, showing the correct contact's conversation (or at minimum the app, given conversation-only fidelity)."
    why_human: "04-03-SUMMARY.md's own D4 coverage entry explicitly defers this: the agent session confirmed the sgnl:// invocation exits 0 and observed the Signal Desktop single-instance-lock IPC handoff in terminal output, but has no way to observe the desktop's actual rendered window state. The bare-scheme (group) form's visual raise WAS confirmed by the developer during 04-01's own human-verify checkpoint (already APPROVED this session) — only the E.164 contact-form's pixel-level confirmation is outstanding, per workflow.human_verify_mode = end-of-phase."
  - test: "Confirm the three judgment-tier prohibitions in 04-01-PLAN.md's must_haves.prohibitions hold: (1) no SQLCipher key / config.json key-field value / message body text ever appears in a log line or returned error string; (2) no second on-disk copy of Signal message plaintext is ever created (no temp-file copy, no decrypted export, no cache file beyond the D-03 tail snippet); (3) 1:1 conversation matching never uses the contact's self-chosen profile name as a match candidate."
    expected: "All three hold with no exceptions found."
    why_human: "These are authored as judgment-tier prohibitions with no wired mechanical enforcement (per the plan's own flagged_assumptions), so per the verification protocol they dispose 'unverified' and require human sign-off rather than an automated pass. This verifier's own code-level inspection (grep across plugins/signal/*.go for os.WriteFile/io.Copy/CopyFile calls, and reading every fmt.Errorf/log line in keyresolve.go, safestorage_linux.go, secretservice.go, dsn.go, plugin.go) found no violation of any of the three — every returned error names steps/paths/booleans/backend names only, no code path writes db.sqlite anywhere, and match.go's D-06 test (TestEligibleConversations_PrivateProfileNameOnlyDoesNotMatch) passes — but this is a non-authoritative LLM-judge finding, not a substitute for the human sign-off the plan's own verification tier requires."
---

# Phase 4: Signal Conversations Verification Report

**Phase Goal:** User's Signal conversations for a topic appear in the webspace stream, read from Signal Desktop's local database without any risk of corrupting or altering it.
**Verified:** 2026-08-03T21:00:00Z
**Status:** human_needed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths (ROADMAP Success Criteria)

| # | Truth | Status | Evidence |
|---|-------|--------|----------|
| 1 | Messages from Signal conversations/groups whose names match the webspace keyword appear in the stream chronologically alongside documents, notes and mail | ✓ VERIFIED | `./scripts/signal-readonly-smoke.sh` synced 1467 real digests against `~/.config/Signal/sql/db.sqlite` and asserted at least one `source_type:"signal"` item in `GET /api/webspaces/{ws}/stream`. `plugins/signal/digest_test.go` covers day-grouping, timestamp = last message, deterministic `source_id`. The digest-row rendering (correct singular/plural title, sender-prefixed preview, standard `StreamRow`) was already visually confirmed and APPROVED by the developer during 04-01's human-verify checkpoint this session. |
| 2 | Detail pane shows the surrounding conversation thread, with an "open in Signal" affordance declared conversation-only fidelity | ✓ VERIFIED (bare-scheme form) / ? HUMAN NEEDED (E.164 contact form) | `Fetch`'s FULL/PREVIEW variants (`plugins/signal/plugin.go` `fetchTranscript`) return `MimeType: "text/html"`, routed with zero frontend change into `DetailPane.svelte`'s existing generic `html` body-variant iframe (confirmed: `git diff --stat` of `web/` across the whole phase touches only one line in `api.ts`). `render_test.go`/`fetch_test.go` cover run-grouping, tombstones, edited-suffix, sanitization-before-wrap, single-bubble day. `Fidelity: LINK_FIDELITY_CONVERSATION_ONLY` and non-empty `DeepLink` confirmed in `toItem`. The bare `sgnl://` scheme (groups, and 1:1s without E.164) was visually confirmed raising Signal Desktop during 04-01's approved checkpoint. The E.164 contact form was invoked hands-on this session (exit 0, IPC handoff observed) but its visual window-raise was never observed by any agent — routed to human verification below. |
| 3 | Signal Desktop's database is opened strictly read-only (`mode=ro`) and is byte-identical after a full sync, including while Signal Desktop is running | ✓ VERIFIED | `dsn.go` builds the DSN with `mode=ro` (`grep -c 'mode=ro' plugins/signal/dsn.go` ≥ 1) and deliberately never sets `immutable=1` (0 occurrences, with a doc comment explaining why). `readonly_test.go`'s AST scan (`TestPluginIssuesNoWriteShapedSQL`, verified passing) rejects any `Exec`/`ExecContext` call and any write-shaped SQL keyword hidden in a string literal, proven non-vacuous by two negative-control fixtures. `byte_identical_test.go`'s fixture test and the opt-in live test both pass. Empirically re-run by this verifier: `./scripts/signal-readonly-smoke.sh` against the real, live `~/.config/Signal/sql/db.sqlite` (with Signal Desktop running) hashed the file before (`ef6d2099...`) and after sync — unchanged. |
| 4 | The decryption key is obtained from whichever keyring backend the user's install actually uses, detected at runtime rather than assumed | ✓ VERIFIED | `keyresolve.go`'s `resolveSafeStorageKey` dispatches strictly on the literal `safeStorageBackend` string read from `config.json` (`gnome_libsecret`/`kwallet` family → Secret Service, `basic_text` → fixed password with zero D-Bus, anything else → named failure). Doc comment and code both confirm no `$XDG_CURRENT_DESKTOP` or desktop-environment probing exists anywhere in `keyresolve.go`/`secretservice.go` (grep confirms zero non-comment occurrences). `TestResolveKey_*` (legacy-only, neither, both, basic_text, keyring-routing, unrecognised-backend) all pass. This machine's real install uses the legacy plaintext-key shape only, so the safeStorage branch is fixture-tested rather than live-tested — expected per the execution-context notes, not a gap. |
| 5 | An unrecognised Signal database schema version fails loudly, naming the version it found, instead of silently importing nothing | ✓ VERIFIED | `schemaguard.go`'s `guardSchemaVersion` reads `PRAGMA user_version`, and for `found > highestSupportedSchemaVersion` (pinned to `1730`, read live off the real database) returns an error naming both the version found and the ceiling, stating it "refuses to import, not silently skipping." `TestSchemaVersionCeiling` (above/at/below-ceiling, re-run by this verifier — PASS) proves the guard is not vacuous, asserting both decimal versions appear in the message. |

**Score:** 5/5 ROADMAP success criteria verified by automated evidence and/or an already-approved human checkpoint; 2 items (a sub-facet of criterion 2, and the phase's 3 judgment-tier prohibitions) are routed to human verification below rather than closed automatically.

### Required Artifacts

| Artifact | Expected | Status | Details |
|----------|----------|--------|---------|
| `plugins/signal/dsn.go` | Read-only SQLCipher DSN (`mode=ro`) + key-proving read | ✓ VERIFIED | Contains `mode=ro`, no `immutable=`, SQLite version-floor check (`checkSQLiteVersionFloor`) added in 04-02 |
| `plugins/signal/schemaguard.go` | `PRAGMA user_version` ceiling guard, fails loudly naming both versions | ✓ VERIFIED | `highestSupportedSchemaVersion = 1730`, documented with source/date; guard tested with negative control |
| `plugins/signal/keyresolve.go` | Dual-shape key resolution | ✓ VERIFIED | Legacy + safeStorage branches both fully implemented and tested |
| `plugins/signal/safestorage_linux.go` | Electron os_crypt unwrap (PBKDF2 + AES-128-CBC) | ✓ VERIFIED | Chromium constants (`saltysalt`, PBKDF2, v10/v11 prefix) present; round-trip + wrong-password rejection tested |
| `plugins/signal/secretservice.go` | freedesktop Secret Service client | ✓ VERIFIED | Uses `AuthenticationDHAES` (encrypted session, never plaintext) |
| `plugins/signal/match.go` | D-05/D-06 conversation matching | ✓ VERIFIED | Own-name-only 1:1 candidates; profile-name-only case has a dedicated negative test |
| `plugins/signal/digest.go` | Conversation-day grouping, tail snippet, deterministic `source_id` | ✓ VERIFIED | Richness-aware (attachments, deleted, edited) as of plan 03; tested |
| `plugins/signal/plugin.go` | Describe/Match/Fetch/Health over the published contract | ✓ VERIFIED | `Fetch` FULL/PREVIEW implemented (no longer a stub); `Health` names each of 3 failure causes |
| `plugins/signal/render.go` | Sanitized chat-transcript HTML renderer | ✓ VERIFIED | `bluemonday.UGCPolicy()`-derived policy, sanitize-before-wrap, no accent colour on bubbles/names/timestamps |
| `plugins/signal/deeplink.go` | `sgnl://` deep-link construction | ✓ VERIFIED | Validated hands-on 2026-08-03 (doc comment records the observed forms and date) |
| `plugins/signal/readonly_test.go`, `byte_identical_test.go`, `outbound_hosts_test.go` | Mechanical read-only/egress enforcement | ✓ VERIFIED | All three re-run by this verifier — PASS, each with a non-vacuous negative control |
| `scripts/signal-readonly-smoke.sh` | End-to-end read-only + stream-presence guard | ✓ VERIFIED | Re-run by this verifier against the real live database — PASS |
| `kernel/config/types.go` (`Source.Path`) | Local-path source field | ✓ VERIFIED | `toml:"path,omitempty"`; validated in `kernel/config/config_test.go` |
| `docs/plugin-contract.md` | Local-path source shape documented | ✓ VERIFIED | `path` key documented in source-config section; `plugins/signal` added to reference-plugin list |

### Key Link Verification

| From | To | Via | Status | Details |
|------|-----|-----|--------|---------|
| `kernel/pluginhost/host.go` | `plugins/signal/main.go` | `WEBSPACES_SOURCE_CONFIG` JSON carries `"path"` | ✓ WIRED | `grep -c '"path"' kernel/pluginhost/host.go` = 1 |
| `plugins/signal/digest.go` | `kernel/index` (`ReplaceWebspaceSourceItems`) | Deterministic `source_id` upserts in place | ✓ WIRED | `TestDatabaseByteIdenticalAfterMatchAndFetch` and the live smoke run confirm repeated syncs don't duplicate; `sourceIDForDigest`/`decodeSourceID` round-trip tested |
| `plugins/signal/plugin.go` | `web/src/lib/api.ts` | `source_type` `"signal"` has a `SOURCE_DISPLAY_NAMES` entry | ✓ WIRED | `signal: 'Signal'` present |
| `plugins/signal/keyresolve.go` | `plugins/signal/secretservice.go` | Literal `safeStorageBackend` selects retrieval path | ✓ WIRED | Confirmed by code read + `TestResolveKey_RoutesToSecretServiceForKeyringBackends` |
| `plugins/signal/safestorage_linux.go` | `plugins/signal/dsn.go` | Unwrapped key validated by an actual read-only open | ✓ WIRED | `openReadOnly` called immediately after key resolution in `plugin.go`'s `openGuarded` |
| `plugins/signal/plugin.go` (`Health`) | `web/src/lib/components/SourceHealthChip.svelte` | `last_error` rendered verbatim | ✓ WIRED | Confirmed no frontend change needed (`git diff --stat web/` empty for that task); e2e-verified this session per 04-02-SUMMARY.md |
| `plugins/signal/plugin.go` (`Fetch`) | `web/src/lib/components/DetailPane.svelte` | `MimeType text/html` routes into existing `html` iframe branch | ✓ WIRED | Confirmed by reading `DetailPane.svelte`'s generic, mime-type-driven `detailBodyVariant()` — no Signal-specific branch exists |

### Behavioral Spot-Checks / Probe Execution

| Behavior | Command | Result | Status |
|----------|---------|--------|--------|
| Kernel builds cgo-free | `CGO_ENABLED=0 go build ./...` | exit 0 | ✓ PASS |
| Kernel config + audit tests | `go test ./kernel/config/... ./internal/audit/...` | exit 0 | ✓ PASS |
| Full Signal plugin test suite | `make test-signal` | exit 0, `ok ... plugins/signal 2.9s` | ✓ PASS |
| Whole-workspace test suite | `make test` (single run) | exit 0 across every module including `test-signal` | ✓ PASS |
| Web typecheck | `npm --prefix web run check` | 0 errors, 1 pre-existing unrelated warning (`SearchBox.svelte`) | ✓ PASS |
| SDK contract unchanged | `go test ./sdk/...` | exit 0 — RPC allowlist untouched | ✓ PASS |
| Read-only-by-construction AST scan | `go test -run TestPluginIssuesNoWriteShapedSQL ./plugins/signal/...` | PASS (non-vacuous, 2 negative controls) | ✓ PASS |
| Byte-identical fixture proof | `go test -run TestDatabaseByteIdenticalAfterMatchAndFetch ./plugins/signal/...` | PASS | ✓ PASS |
| Zero-egress proof | `go test -run TestNoOutboundNetworkHosts ./plugins/signal/...` | PASS | ✓ PASS |
| Schema-ceiling negative control | `go test -run TestSchemaVersionCeiling ./plugins/signal/...` | PASS (above/at/below) | ✓ PASS |
| Health three-cause naming | `go test -run TestHealth ./plugins/signal/...` | PASS (5 subtests) | ✓ PASS |
| D-06 anti-spoofing matcher | `go test -run TestEligibleConversations ./plugins/signal/...` | PASS (8 subtests) | ✓ PASS |
| Message parsing (deleted/edited/attachments/reactions/quote) | `go test -run TestParseMessage ./plugins/signal/...` | PASS (11 subtests) | ✓ PASS |
| **Live end-to-end guard** | `./scripts/signal-readonly-smoke.sh` against real `~/.config/Signal/sql/db.sqlite` | 1467 real digests synced; SHA-256 `ef6d2099...` unchanged before/after; stream contains `source_type:"signal"` items | ✓ PASS |

No separately-declared `scripts/*/tests/probe-*.sh` files exist for this phase — `scripts/signal-readonly-smoke.sh` is the plan-declared, phase-specific probe and was executed directly above.

### Requirements Coverage

| Requirement | Source Plan(s) | Description | Status | Evidence |
|-------------|-----------------|-------------|--------|----------|
| SRC-02 | 04-01, 04-02, 04-03 | Signal plugin reads Signal Desktop DB strictly read-only (`mode=ro`); extracts key via OS keyring (backend-detected); detects schema version and fails loudly on unknown | ✓ SATISFIED | All 5 ROADMAP criteria verified above; REQUIREMENTS.md already marks SRC-02 `[x]` / "Phase 4 / Complete" |

No orphaned requirements — SRC-02 is the only requirement ID mapped to Phase 4 in REQUIREMENTS.md, and all three plans declare it in frontmatter.

### Anti-Patterns Found

None. `grep -rn -E "TBD|FIXME|XXX|TODO|HACK|PLACEHOLDER" plugins/signal/*.go` (excluding test files) returned zero matches. No `os.WriteFile`/`io.Copy`/`CopyFile` calls exist anywhere in the non-test plugin source (no on-disk plaintext copy of the database is ever created). No log line or returned error string in `keyresolve.go`, `safestorage_linux.go`, `secretservice.go`, or `dsn.go` embeds a key, ciphertext, or config.json field value — every message names a step, a path, a backend, or a boolean.

### Human Verification Required

1. **E.164 contact-form `sgnl://` deep link — visual window-raise confirmation**
   - **Test:** Click "Open in Signal" on a Signal digest for a 1:1 conversation whose contact has an E.164 number.
   - **Expected:** Signal Desktop visibly raises/focuses.
   - **Why human:** No agent session can observe rendered window state; this is the one item 04-03-SUMMARY.md itself explicitly defers to end-of-phase human verification (`workflow.human_verify_mode = end-of-phase`). The bare-scheme (group) form was already visually confirmed and approved during 04-01's checkpoint.

2. **The phase's 3 judgment-tier prohibitions (04-01-PLAN.md `must_haves.prohibitions`)**
   - **Test:** Confirm (a) no key/ciphertext/message-body value ever appears in a log or error string, (b) no second on-disk copy of Signal plaintext is ever created, (c) 1:1 matching never uses the contact's self-chosen profile name.
   - **Expected:** All three hold.
   - **Why human:** Authored as judgment-tier with no wired mechanical check (per the plan's own `flagged_assumptions`), so per the verification protocol these dispose "unverified" rather than an automated pass. This verifier's own code inspection found no violation of any of the three (see Anti-Patterns Found above, and the D-06 negative-control test), but that is a non-authoritative LLM-judge finding — human sign-off is what the plan's own tier requires.

### Gaps Summary

No blocking gaps. Every ROADMAP success criterion has direct automated and/or already-approved-human evidence. The two items above are genuinely un-automatable (rendered window state; judgment-tier prohibitions with no wired mechanical check by the plan's own design) rather than missing or stubbed implementation — this verifier found no artifact missing, no stub, no broken wiring, and no anti-pattern in the phase's changed files. The phase's own code review (`04-REVIEW.md`) independently found 0 Critical, 2 Warning, 1 Info issues, none of which block goal achievement.

---

_Verified: 2026-08-03T21:00:00Z_
_Verifier: Claude (gsd-verifier)_
