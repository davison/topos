---
phase: 04-signal-conversations
reviewed: 2026-08-03T20:43:56Z
depth: standard
files_reviewed: 33
files_reviewed_list:
  - .gitignore
  - Makefile
  - config.example.toml
  - docs/plugin-contract.md
  - go.work
  - go.work.sum
  - kernel/config/config.go
  - kernel/config/config_test.go
  - kernel/config/types.go
  - kernel/pluginhost/host.go
  - plugins/signal/README.md
  - plugins/signal/byte_identical_test.go
  - plugins/signal/deeplink.go
  - plugins/signal/deeplink_test.go
  - plugins/signal/digest.go
  - plugins/signal/digest_test.go
  - plugins/signal/dsn.go
  - plugins/signal/dsn_test.go
  - plugins/signal/fetch_test.go
  - plugins/signal/go.mod
  - plugins/signal/go.sum
  - plugins/signal/health_test.go
  - plugins/signal/keyresolve.go
  - plugins/signal/keyresolve_test.go
  - plugins/signal/main.go
  - plugins/signal/match.go
  - plugins/signal/match_test.go
  - plugins/signal/message.go
  - plugins/signal/message_test.go
  - plugins/signal/outbound_hosts_test.go
  - plugins/signal/plugin.go
  - plugins/signal/readonly_test.go
  - plugins/signal/render.go
  - plugins/signal/render_test.go
  - plugins/signal/safestorage_linux.go
  - plugins/signal/safestorage_test.go
  - plugins/signal/schemaguard.go
  - plugins/signal/schema_version_fixture_test.go
  - plugins/signal/secretservice.go
  - plugins/signal/testdata/README.md
  - scripts/signal-readonly-smoke.sh
  - web/src/lib/api.ts
findings:
  critical: 0
  warning: 2
  info: 1
  total: 3
status: issues_found
---

# Phase 04: Code Review Report

**Reviewed:** 2026-08-03T20:43:56Z
**Depth:** standard
**Files Reviewed:** 33 (source scope of the 41-file list; lockfiles/generated artifacts excluded from per-line review)
**Status:** issues_found

## Summary

Reviewed the Signal plugin (SRC-02) end to end — key resolution
(legacy plaintext + Electron `safeStorage`/Secret-Service unwrap),
read-only SQLCipher DSN construction, schema-version guarding,
conversation matching (D-05/D-06 anti-spoofing), digest building,
transcript rendering/sanitization, deep-link construction, the
kernel-side config/pluginhost plumbing that supports the new
local-path source shape, and the AST-based mechanical enforcement
tests (read-only, no-outbound-network).

This is a strong submission: `go build`/`go vet`/`go test` all pass
cleanly under `CGO_ENABLED=1 -tags libsqlcipher`, D-06's anti-spoofing
rule (a contact's self-chosen profile name is never a match candidate
for a 1:1 conversation) is correctly implemented and has dedicated
negative-control tests, no key material or message body text was
found in any log/error string across the whole package, and the
sanitize-before-wrap HTML transcript pipeline is sound (verified
against the sanitizer for `<script>`/`onerror` payloads).

I adversarially probed the one place untrusted-ish data (Signal
Desktop's own `config.json`) flows unescaped into a raw DSN string
(`dsn.go`'s `_key=x'%s'` interpolation) with a hands-on proof-of-concept
attempting DSN/PRAGMA injection. The attempt did **not** succeed — the
underlying `go-sqlite3` driver's own `_key` parameter handling happens
to neutralize single-quote-containing values by re-wrapping them in
double quotes before building the `PRAGMA key = ...;` statement, so no
exploitable write path was found. I'm noting this here so the finding
below (WR-01) is understood correctly: it is a robustness/error-quality
gap, not a proven security hole, but it now depends on an undocumented
third-party driver behavior rather than validation this plugin controls
itself, which is worth closing directly.

Two Warnings and one Info item below; no Critical/Blocker findings.

## Warnings

### WR-01: Legacy SQLCipher key path has no format/length validation before DSN interpolation

**File:** `plugins/signal/keyresolve.go:70-86` (see `resolveKey`, the `hasKey && !hasEncrypted` branch, line ~80), consumed by `plugins/signal/dsn.go:81-86` (`buildReadOnlyDSN`)

**Issue:** `resolveKey` returns `cfg.Key` (the legacy plaintext `config.json` "key" field) completely unvalidated — no length check, no hex-charset check. This is inconsistent with the sibling `safeStorage` path in the same function, which does validate: `resolveSafeStorageKey` rejects a decrypted value whose length isn't exactly `expectedRawKeyHexLen` (64). Both paths feed the same `rawHexKey` into `buildReadOnlyDSN`, which splices it **unescaped** into a raw DSN string via `fmt.Sprintf("file:%s?mode=ro&_key=x'%s'&_cipher_page_size=4096", dbPath, rawHexKey)`.

I wrote a proof-of-concept (`TestDSNInjectionPoC{1,2,3}`, run against this package's real fixture-building machinery and the actual linked `jgiannuzzi/go-sqlite3` driver) attempting to smuggle a `DROP TABLE` via a malformed key value (including a percent-encoded semicolon to dodge Go's `url.ParseQuery` raw-`;`-separator rejection). It did not succeed: the driver's own `_key` parameter handler (`sqlite3.go`'s `SQLiteDriver.Open`) detects a value containing `'` and re-wraps the *entire* value in double quotes before building `PRAGMA key = %s;`, which turns any embedded quotes/semicolons into inert string data rather than separate SQL statements. A value containing **both** `'` and `"` is rejected outright by the driver with a clean error. So today, this is not exploitable — but that safety property is an implementation detail of a third-party dependency this plugin doesn't test against or document reliance on, not something this code enforces itself.

What *is* real and reproducible: a malformed legacy `key` field (e.g. a corrupted `config.json`, or any value containing a stray `'`) does not fail with this codebase's own carefully-designed "key resolution failed" error class (see `openGuarded`'s doc comment, which enumerates exactly this design goal — "fails loudly, never confusingly", ROADMAP criterion 5). Instead it silently falls through to `openReadOnly`, gets treated by SQLCipher as an ordinary passphrase, and surfaces as a generic `"file is not a database"` error — exactly the confusing failure mode `openGuarded`'s own doc comment says it exists to avoid for every *other* failure cause.

**Fix:** Validate `cfg.Key` (and, for defense in depth, the `safeStorage`-decrypted plaintext) against the exact expected shape before returning it from `resolveKey`, mirroring what `resolveSafeStorageKey` already does for length:

```go
var rawKeyHexPattern = regexp.MustCompile(`^[0-9a-fA-F]{64}$`)

func validateRawHexKey(k string) error {
	if !rawKeyHexPattern.MatchString(k) {
		return fmt.Errorf("signal: resolved key is not a %d-character hex string", expectedRawKeyHexLen)
	}
	return nil
}
```
Call this from both branches of `resolveKey` before returning, so a malformed key of either shape fails at key-resolution time with the same named, actionable error class every other failure in `openGuarded` already produces — rather than reaching `dsn.go`'s raw string interpolation at all.

### WR-02: `sourceIDForDigest`/`decodeSourceID` colon-delimiter assumes `conversationID` never contains `:`

**File:** `plugins/signal/digest.go:43-61`

**Issue:** `sourceIDForDigest` builds a `source_id` as `base64.RawURLEncoding.Encode(conversationID + ":" + day)`, and `decodeSourceID` reverses it with `strings.SplitN(string(b), ":", 2)`. This is correct only as long as `conversationID` itself never contains a `:` character. Signal Desktop's conversation ids are UUID-shaped today (no colons), so this holds in practice, but nothing in this codebase validates or asserts that invariant, and no test exercises a `conversationID` containing a colon. If it ever did (a future Signal Desktop schema change, or a conversation id derived differently for some conversation type), `decodeSourceID` would silently split at the wrong point and `Fetch` would look up the wrong (conversationID, day) pair — very likely surfacing as a `NotFound` for a real digest, or in the worst case matching a different, unintended day within the same conversation, rather than failing loudly the way this codebase otherwise insists on (see `openGuarded`'s explicit "fails loudly, never confusingly" design goal, and `guardSchemaVersion`'s "refusing to import, not silently skipping" precedent).

**Fix:** Either use a delimiter that's provably impossible in a conversation id (or the same base64-per-field + fixed-length-prefix encoding `plugins/proton`'s equivalent function reportedly mirrors — worth checking that file's actual delimiter choice for precedent), or validate `!strings.Contains(conversationID, ":")` at the point `sourceIDForDigest` is called and fail loudly if violated, so an assumption drift becomes a build-time-visible test failure instead of a silent mis-decode.

## Info

### IN-01: Two files are not `gofmt`-clean

**File:** `plugins/signal/render.go:38` (`editedSuffix` alignment inside the `const (...)` block), `plugins/signal/dsn_test.go:53` (`version` field alignment inside a struct literal)

**Issue:** `gofmt -l` flags both files. The diffs are purely whitespace-alignment (`gofmt -d` shows only column-alignment changes, no semantic difference), but they indicate these two files weren't run through `gofmt`/`goimports` before commit, unlike the rest of the package.

**Fix:** `gofmt -w plugins/signal/render.go plugins/signal/dsn_test.go`.

---

_Reviewed: 2026-08-03T20:43:56Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
