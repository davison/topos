---
phase: 04
slug: signal-conversations
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-08-03
---

# Phase 04 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| Signal Desktop `db.sqlite` → plugin process | The user's entire message history crosses here; message bodies are untrusted input; the file itself must never be altered | Full message plaintext (SQLCipher-encrypted at rest) |
| `~/.config/Signal/config.json` / OS keyring → plugin process | Decryption-key material crosses here (legacy plaintext `key` or safeStorage `encryptedKey` + Secret Service unwrap) | SQLCipher key material |
| plugin subprocess → kernel (gRPC) | Items cross a process boundary into the unencrypted local index | D-03 tail snippet only (bounded message text) + metadata |
| `config.toml` → kernel config loader | Operator-controlled paths cross into filesystem access | Local filesystem paths |
| rendition route → browser iframe | Sanitized transcript HTML rendered in the SPA's detail pane | Sanitized per-message HTML, CSP-restricted |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-04-01 | Tampering | `plugins/signal/dsn.go`, queries in `match.go`/`digest.go` | critical | mitigate | `mode=ro` URI DSN (×3 in dsn.go), no immutability assertion (live-writer safe); smoke script SHA-256 hashes `db.sqlite` before/after full sync; verified byte-identical repeatedly this session with Signal Desktop running | closed |
| T-04-02 | Information Disclosure | `keyresolve.go`, `plugin.go` logging | high | mitigate | Errors/logs carry step names, counts, backend name only; field presence reported as boolean; `health_test.go` asserts fixture key/body never appear in `LastError`; independent code-review and verifier inspections found zero leakage | closed |
| T-04-03 | Information Disclosure | `digest.go` → `kernel/index` | medium | mitigate | `Item.Preview` carries only the D-03 tail snippet; all other item fields are metadata-only (verifier read `toItem` directly); nothing beyond the visible preview reaches the unencrypted index or FTS | closed |
| T-04-04 | Spoofing | `plugins/signal/match.go` | high | mitigate | 1:1 matching reads only the user's own name fields (nickname / system contact name) from the JSON blob; profile name and derived title excluded; `match_test.go` carries the D-06 profile-name negative cases | closed |
| T-04-05 | Denial of Service | full-history scan of the live DB inside a sync cycle | low | accept | Single-flight coordinator serialises syncs; per-source configurable `sync_interval`; local file read — no new machinery warranted (accepted at plan time) | closed |
| T-04-SC | Tampering | supply chain — `go.mod replace` to unmerged-PR fork | high | mitigate | Blocking human decision resolved this session: option-a authorized with exact pin `jgiannuzzi/go-sqlite3 v1.14.17-0.20230327162135-f208443ec79d` (PR mattn/go-sqlite3#1109 head, verified via GitHub API); pin + rationale commented in `go.mod`; `internal/audit/module_pins_test.go` guards the dependency floor; audit trail in 04-01-SUMMARY.md | closed |
| T-04-06 | Tampering | any future non-test file in `plugins/signal` | high | mitigate | `readonly_test.go` AST scan rejects `Exec`/`ExecContext` selectors and write-shaped SQL literals (`VACUUM` and `wal_checkpoint` forbidden by name) with a negative control proving the scanner has teeth; `byte_identical_test.go` as empirical backstop | closed |
| T-04-07 | Tampering | `keyresolve.go` post-unwrap validation | medium | mitigate | Key length validation plus read-only open proves the key before resolution is treated as done; distinct named backend-mismatch error (CBC has no integrity check); covered by `keyresolve_test.go` | closed |
| T-04-08 | Information Disclosure | any future network call from the plugin | high | mitigate | `outbound_hosts_test.go` asserts the empty host set; repo-wide `internal/audit` egress scan with `plugins/signal` deliberately absent from `sanctionedEgressFiles` | closed |
| T-04-09 | Tampering | linked SQLCipher below the SQLite 3.51.3 floor | high | mitigate | Runtime version check in `dsn.go` fails loudly naming the version found before any table read (system sqlcipher 4.14.0-1 = SQLite 3.51.3 baseline confirmed installed) | closed |
| T-04-10 | Information Disclosure | Secret Service session on the D-Bus session bus | medium | mitigate | `secretservice.go` opens `AuthenticationDHAES` encrypted sessions only (×3); plain session mode never referenced (0 occurrences); handshake via keybase/go-keychain, not hand-rolled | closed |
| T-04-11 | Information Disclosure | `LastError` strings rendered verbatim in the UI tooltip | medium | mitigate | Health messages name steps, paths, backends only; `health_test.go` asserts the fixture key and fixture message body never appear in any returned `LastError` (22 assertion refs) | closed |
| T-04-12 | Tampering | `render.go` — crafted message body containing markup | high | mitigate | Per-message sanitization through a `bluemonday.UGCPolicy()`-derived policy (no `style` attr) before assembly and wrapping; `render_test.go` asserts script elements and event-handler attributes are absent (36 refs) | closed |
| T-04-13 | Information Disclosure | remote references inside a message body | medium | mitigate | Kernel CSP on the rendition route permits no subresource (kernel/httpapi/item.go + tests); stylesheet keeps the image-hiding rule; attachments render as text placeholders, never fetched or decrypted this phase | closed |
| T-04-14 | Tampering | `deeplink.go` — URI built from conversation-derived values | medium | mitigate | **Superseded by T-04-17 (2026-08-04, 04-04-PLAN.md).** The escape-based mitigation recorded here as closed (`url.` escaping ×3) described `encodePhoneFragment`, which percent-encoded the E.164's mandatory leading `+` to `%2B` — a payload shape Signal Desktop's own shipped route rejects outright (traced in `.planning/debug/sgnl-link-didnt-make-sense.md`). 04-04-PLAN.md Task 1 deleted that function and replaced it with allowlist validate-and-refuse; see T-04-17 for the control that actually ships. This row is kept, not deleted, so the register records the supersession rather than quietly losing history of a mitigation that no longer exists. | superseded |
| T-04-15 | Information Disclosure | transcript widens what leaves the SQLCipher file | medium | mitigate | Transcript produced at `Fetch` time only from a fresh read-only open, never persisted (zero `os.WriteFile`/`io.Copy` in render/plugin paths); D-03 still bounds indexed text to the tail snippet | closed |
| T-04-16 | Denial of Service | corrupt or enormous message JSON blob | low | mitigate | Blob reads bounded following `plugins/proton/body.go` precedent; unparseable blob degrades to empty body rather than failing the day (`message.go`) | closed |
| T-04-17 | Tampering | `plugins/signal/deeplink.go` — URI built from a conversation-derived number | medium | mitigate | Source-derived numbers are now admitted by allowlist only, against `e164Pattern` (`^\+[1-9][0-9]{1,14}$`), a pattern bounding both alphabet and length to the exact shape Signal Desktop's own shipped validator accepts (traced statically from the installed `app.asar` bundle). A value that does not match contributes zero bytes to the emitted URI — the constant bare `sgnl://` scheme is returned instead, so no URI metacharacter, whitespace padding, or over-length value can restructure the emitted link. This is the stronger control versus T-04-14's escaping approach: an allowlist cannot be defeated by a producer/consumer encoder mismatch, which is precisely how the escape-based version failed (we percent-encoded `+`; Signal never percent-decodes the fragment before validating). Evidence: `plugins/signal/deeplink_test.go`'s `TestDeepLink_E164BoundaryMatrix` (metacharacters, padding, leading zero, length boundaries) plus `scripts/signal-readonly-smoke.sh`'s contact-form shape guard over a real stream response (verified non-vacuous: 105/105 links checked against a real 1:1 conversation). | closed |
| T-04-18 | Information Disclosure | The emitted URI carries the contact's phone number | low | accept | The number already lives in the local index's `deep_link` column and belongs to the user's own contact; the link is handed only to a local desktop application via the OS scheme handler, never to a network endpoint — `plugins/signal/outbound_hosts_test.go` still asserts the plugin's empty host set. Emitting the number without transformation (literal `+` instead of `%2B`) changes the bytes, not the exposure. | closed |
| T-04-19 | Tampering | Supply chain — 04-04-PLAN.md's dependency footprint | low | accept | This plan installs nothing: it removes the `net/url` import and adds the standard-library `regexp` import, with no npm/pip/cargo/`go get` operation and no `go.mod` change. No package-legitimacy audit applies; T-04-SC's existing pin and `internal/audit/module_pins_test.go` guard are untouched. | closed |

*Status: open · closed · superseded (mitigation replaced by a later threat id, row kept for history) · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-04-01 | T-04-05 | Full-history scan of the local live DB is serialised by the existing single-flight sync coordinator with per-source `sync_interval`; a local file read at personal-data scale does not warrant new throttling machinery | plan author (04-01-PLAN.md threat register, disposition: accept) | 2026-08-03 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-03 | 17 | 17 | 0 | /gsd-secure-phase orchestrator (L1 evidence classification; short-circuit — plan-time register, ASVS 1) |
| 2026-08-04 | 19 | 18 closed + 1 superseded | 0 | 04-04-PLAN.md Task 3 (gap closure G-04-1): corrected T-04-14's evidence to record supersession rather than claiming an escaping mitigation that this plan removed; added T-04-17 (allowlist replacement, mitigate/closed), T-04-18 (phone-number-in-URI exposure, accept/closed), T-04-19 (no new dependency, accept/closed) |

Corroborating evidence beyond L1 greps, from this session: code review (04-REVIEW.md, 0 critical — including a hands-on DSN-injection PoC that did not reproduce), goal verification (04-VERIFICATION.md, 5/5 criteria — verifier independently re-ran the named threat-tier tests `TestPluginIssuesNoWriteShapedSQL`, `TestDatabaseByteIdenticalAfterMatchAndFetch`, `TestNoOutboundNetworkHosts`, `TestSchemaVersionCeiling`), and two independent live byte-identical smoke runs against the real database.

Note: human sign-off on the three judgment-tier prohibitions (log hygiene, no on-disk plaintext copy, profile-name anti-spoofing) is tracked separately as 04-UAT.md test 2 — the mechanical mitigations above are verified; the UAT item is the plan-mandated human review, not an open threat.

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-03
