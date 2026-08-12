---
phase: 8
slug: whatsapp-conversations-managed-risk
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-08-10
---

# Phase 8 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| WhatsApp servers → plugin | Untrusted remote content enters via the linked-device session | Message bodies, sender push names, group subjects, history-sync payloads, logout/ban reason codes |
| plugin → WhatsApp servers | Outbound direction the plugin must never use beyond passive linked-device participation | Protocol handshake/receipt traffic only — no sends, no presence |
| plugin subprocess → kernel (gRPC) | Plugin hands the kernel unsanitized structural HTML fragments, item metadata, and the error-vs-empty-success signal that decides row retention | Transcript fragments, chat metadata, health status |
| browser → kernel HTTP | Untrusted request body reaches a handler that spawns an OS process | Plugin name, instance id, store path |
| kernel → link subprocess | Kernel parses attacker-influenced-in-principle stdout from a child process | Machine-readable link event lines |
| link subprocess ↔ serve-mode instance | Two processes contend for one whatsmeow session store on disk | SQLite session store file |
| link subprocess → WhatsApp servers | A live pairing credential exists for the session's duration | Pairing QR payload |
| kernel → browser | The rotating pairing image and kernel error text cross into the SPA | QR PNG data URI, error strings |
| user → SPA → kernel | Instance ids, plugin binary names, and store paths typed by the user reach the link start endpoint | Configuration values |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-08-01 | Spoofing | `plugins/whatsapp/match.go` candidate-name derivation | medium | mitigate | `candidateNames` (match.go:32) admits only a group's own cached subject or a 1:1 chat's store `contact_name` (populated exclusively from the user's address book, D-06); no push/profile-name field is ever a candidate; empty candidate → zero candidates (D-07) | closed |
| T-08-02 | Tampering | whatsmeow `Client` send/mutate/presence surface | high | mitigate | `readonly_test.go` non-vacuous AST scan (`TestReadOnly_NoSendCapableClientSelector`) bans `SendMessage`, `SendPresence`, `Logout`, and the full send-capable selector set, with negative controls proving the scanner is not vacuous | closed |
| T-08-03 | Information Disclosure | plugin logging + outbound network surface | high | mitigate | `pluginLogger` (connect.go:16) writes to stderr; event-handler log lines carry counts and error values, never message bodies, contact names, or key material; `outbound_hosts_test.go` rejects self-constructed HTTP clients and non-allowlisted network-scheme literals | closed |
| T-08-04 | Tampering | `plugins/whatsapp/render.go` transcript fragment | medium | mitigate | `escapeText` (render.go:58) HTML-escapes every interpolated untrusted field (sender name, body, timestamp); fragment carries only the kernel's closed `chatTranscriptClassTokens` set; `kernel/httpapi/rendition.go` sanitizer is the second, authoritative layer | closed |
| T-08-05 | Denial of Service | `plugins/whatsapp/plugin.go` `Match` return channel | high | mitigate | Every non-healthy state returns `status.Errorf(codes.Unavailable, ...)` (plugin.go:196,207,230,309), never an empty success; `delink_test.go` (`TestDelink_MatchReturnsUnavailableForEveryNonHealthyState`) pins the error-vs-empty-success distinction the kernel treats oppositely | closed |
| T-08-06 | Elevation of Privilege | `kernel/httpapi/whatsapplink.go` start handler | high | mitigate | Requested plugin name validated against `pluginhost.DiscoverAllBinaries` BEFORE anything executes (whatsapplink.go:483-496); `exec.CommandContext` runs only the resolved discovered path, never a caller-supplied path; injected-spawner test proves the ordering | closed |
| T-08-07 | Tampering | whatsmeow session store contended by two processes | medium | mitigate | `SuspendInstance` stops the running serve-mode instance for the link session's duration; `storelock.go` exclusive `LOCK_EX\|LOCK_NB` flock (storelock.go:44) is the independent second layer — second process exits with `ErrStoreInUse` (`whatsapp_store_in_use`) rather than opening the store | closed |
| T-08-08 | Information Disclosure | relayed pairing QR | medium | mitigate | Link subprocess emits only the rendered PNG data URI, never the raw pairing payload; `QRPanel.svelte` renders the returned image only — no localStorage, no history entry, no URL parameter; kernel's loopback-only default binding (unchanged) keeps the image off the network; deadline reaper prevents serving a code past validity | closed |
| T-08-09 | Repudiation | `last_error` health copy | low | accept | Advisory UI copy, not an audit record — see Accepted Risks Log AR-08-01 | closed |
| T-08-10 | Denial of Service | accumulated/orphaned link subprocesses | medium | mitigate | `maxConcurrentLinkSessions` cap enforced before spawning (WR-01), per-session deadline with background reaper (`reapLoop`/`reapExpired`), cancel-on-DELETE, terminate-all on kernel shutdown; `QRPanel.svelte` `retireSession` cancels on explicit cancel AND on unmount, with a spec asserting the cancel request fires | closed |
| T-08-11 | Tampering | kernel parsing link subprocess stdout | low | mitigate | Machine-readable mode restricts stdout to well-formed event lines with diagnostics on stderr; kernel reader treats a malformed line as an error event rather than trusting a partial parse | closed |
| T-08-12 | Elevation of Privilege | store path submitted to link start endpoint | medium | mitigate | SPA submits only the instance's own stored/entered path; kernel refuses any plugin binary outside its discovered set regardless of what the SPA sends (whatsapplink.go:483-496), so the browser cannot widen what may be launched | closed |
| T-08-13 | Information Disclosure | kernel error text rendered in `QRPanel.svelte` | low | accept | Rendered as text via `Alert`/`AlertDescription` with Svelte's default escaping, never `{@html}` — see Accepted Risks Log AR-08-02 | closed |
| T-08-SC | Tampering | `plugins/whatsapp/go.mod` supply chain (whatsmeow + QR encoder) | high | mitigate | Blocking human legitimacy checkpoint completed for `go.mau.fi/whatsmeow` with dated pin-rationale comment in go.mod (approved 2026-08-10, re-verified against `go list -m -json @latest`); QR encoder (`rsc.io/qr` via `mdp/qrterminal`) passed the Go-ecosystem legitimacy protocol recorded in 08-RESEARCH.md's Package Legitimacy Audit; both dependencies confined to the plugin module — the kernel binary never carries them | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-08-01 | T-08-09 | The plugin's `last_error` failure text is advisory UI copy, not an audit record; the kernel's own `sync_runs` history is the authoritative outcome log and is unchanged by this phase | plan 08-02 threat model (executed) | 2026-08-10 |
| AR-08-02 | T-08-13 | Kernel error text is rendered as text through the existing `Alert` component; Svelte escapes interpolated text by default, so a hostile message cannot inject markup. The message's author is the local kernel, not a remote party | plan 08-04 threat model (executed) | 2026-08-10 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-10 | 14 | 14 | 0 | gsd-secure-phase (L1 grep-depth, short-circuit: register authored at plan time, ASVS 1) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-10
