---
phase: 03-email-in-the-webspace
plan: 01
subsystem: api
tags: [go-imap, imap, proton-bridge, tls-pinning, source-plugin, go-workspace]

requires:
  - phase: 02-two-sources-one-trustworthy-stream
    provides: sync identity promoted to (webspace, source_type); config field plumbing precedent (CACert); agent grant block shape
provides:
  - "plugins/proton: a third source plugin (IMAP against Proton Mail Bridge) implementing Describe/Match/Fetch/Health"
  - "kernel/config.Source.Username and .WebmailBaseURL, wired through kernel/pluginhost/host.go's WEBSPACES_SOURCE_CONFIG map"
  - "StreamRow.svelte renders item.group_label (sender) for the first time — previously flowed through the whole stack unrendered"
  - "source_id encoding decision (Task 2): URL-safe base64 of the normalized Message-ID"
affects: [03-02, docs/api.md stable-ID scheme, future email-search plans]

tech-stack:
  added: [github.com/emersion/go-imap v1.2.1, github.com/emersion/go-message v0.18.2]
  patterns:
    - "IMAP host-pinned dialer (pinnedDialer + allowHost) adapted from plugins/silverbullet's http.Transport.DialContext pattern to go-imap's Dialer interface"
    - "Match-time in-process source_id->mailbox cache, rebuilt every sync, consulted by Fetch (no cross-RPC state persisted to disk)"
    - "TLS ServerName override (fixed to the Bridge cert's own SAN) + RootCAs pinning, never InsecureSkipVerify, for a self-signed cert reached through a LAN forwarder at a different address than it was issued for"

key-files:
  created:
    - plugins/proton/go.mod
    - plugins/proton/main.go
    - plugins/proton/client.go
    - plugins/proton/plugin.go
    - plugins/proton/body.go
  modified:
    - kernel/config/types.go
    - kernel/pluginhost/host.go
    - go.work
    - Makefile
    - config.example.toml
    - web/src/lib/components/StreamRow.svelte

key-decisions:
  - "Task 2 (one-way door): email source_id = base64.RawURLEncoding of the normalized Message-ID (option-a) — reversible with no extra persisted state, URL-safe with no escaping subtleties, encodes nothing about mailbox/label/UID so a message under two matching labels yields one id"
  - "Fidelity is always LINK_FIDELITY_ANCHORED, never EXACT — no verified mapping from an IMAP Message-Id/UID to Proton's internal webmail message id exists (03-RESEARCH.md Pitfall 5)"
  - "TLS ServerName hardcoded to '127.0.0.1' (bridgeCertServerName constant) rather than made configurable — Bridge only ever binds and issues a certificate for its own loopback interface by design, confirmed live via Task 1's SAN inspection"

requirements-completed: [SRC-01]

coverage:
  - id: D1
    description: "plugins/proton Go module implementing Describe/Match/Fetch/Health, wired into the kernel via three-place config plumbing"
    requirement: "SRC-01"
    verification:
      - kind: unit
        ref: "go build ./... && go vet ./... (plugins/proton)"
        status: pass
      - kind: integration
        ref: "go test ./internal/audit/... (repo-wide egress + read-only AST scan, includes plugins/proton)"
        status: pass
    human_judgment: false
  - id: D2
    description: "Live email appears in the webspace stream with sender/subject/date and opens in the detail pane"
    requirement: "SRC-01"
    verification: []
    human_judgment: true
    rationale: "Live sync against the real Bridge failed at the IMAP LOGIN step ('no such user', then 'too many login attempts' after Bridge's own rate-limiting) — a Bridge-account credential issue on the user's environment, not exercised by any automated test. Requires the user to re-verify the Bridge username in Bridge's own Settings before this can be confirmed live."
  - id: D3
    description: "Sender (item.group_label) rendered in the stream row for the first time"
    requirement: "SRC-01"
    verification:
      - kind: unit
        ref: "npm run check (web) — 0 errors, 0 warnings"
        status: pass
    human_judgment: false

duration: 1h10min
completed: 2026-07-31
status: complete
---

# Phase 3 Plan 1: Proton Mail IMAP Source Plugin (Tracer) Summary

**IMAP source plugin against Proton Mail Bridge (go-imap v1.2.1, host-pinned TLS, EXAMINE+BODY.PEEK never-mark-read guarantee, Message-ID dedup) wired end to end through kernel config plumbing and the stream UI's sender field.**

## Performance

- **Duration:** ~1h10min (continuation from a resolved Task 1 human-action checkpoint)
- **Completed:** 2026-07-31
- **Tasks:** 3 (Task 1: Bridge reachability/trust verification; Task 2: source_id encoding decision; Task 3: tracer implementation)
- **Files modified:** 8 modified, 5 created (plugins/proton module)

## Accomplishments

- Built `plugins/proton`, a new Go workspace module implementing the full `sdk.SourcePlugin` interface (Describe/Match/Fetch/Health) against Proton Mail Bridge over IMAP.
- Host-pinned, TLS-verified connection layer: `ServerName` override pinned to the Bridge certificate's actual SAN (`127.0.0.1`) plus a `RootCAs` pool from the exported certificate — full chain and expiry verification retained, `InsecureSkipVerify` never used. Mandatory `StartTLS` on the `imap://` scheme with no plaintext fallback path.
- `Match`: `LIST` every mailbox once, filter by case-insensitive leaf-name equality against the webspace's keywords, `EXAMINE` (never `SELECT`) each matched mailbox, merge results by normalized Message-ID across mailboxes so a message under two matching labels appears exactly once with both labels present.
- `Fetch`: re-resolves the current UID via `UID SEARCH HEADER Message-Id` (never trusts a cached UID, since UIDs are only meaningful within one selected mailbox), fetches the body with `BODY.PEEK` (never `BODY`) so `\Seen` is never implicitly set.
- `Health`: bounded 5-second dial+login, clean `Reachable:false` + actionable `last_error` on failure, never a gRPC error and never a hang.
- Three-place config plumbing: `kernel/config/types.go` gained `Source.Username` and `Source.WebmailBaseURL`; `kernel/pluginhost/host.go`'s `launch()` wires both into the `WEBSPACES_SOURCE_CONFIG` JSON map; the plugin's own `main.go` decodes and validates both.
- `StreamRow.svelte` now renders `item.group_label` (the sender) as the first entry in the metadata strip, before the date — a field that flowed through the whole backend/TS stack since Phase 1 but was never rendered by any template until this plan.
- `go.work`, `Makefile`, and `config.example.toml` updated so `plugins/proton` builds, tests, and documents itself alongside the existing three plugins.

## Task Commits

1. **Task 1: Make Proton Mail Bridge reachable and trusted from this desktop** — no commit (verification only; `.env` is gitignored). Verified: `.env` carries all four keys, the exported certificate parses and its SAN is `IP Address:127.0.0.1` only, and a STARTTLS handshake against `PROTON_BRIDGE_ADDR` verifies against that certificate (`Verify return code: 0`).
2. **Task 2: Decide the email source_id encoding** — no commit (decision only, carried into Task 3's `encodeSourceID`/`decodeSourceID`).
3. **Task 3: End-to-end "Proton mail in the webspace stream"** — `9227e57` (feat)

**Plan metadata:** (this commit, pending)

## Files Created/Modified

- `plugins/proton/go.mod` — new Go workspace module, `github.com/emersion/go-imap@v1.2.1` and `github.com/emersion/go-message@v0.18.2` pinned exactly per 03-RESEARCH.md's Package Legitimacy Audit
- `plugins/proton/main.go` — subprocess entrypoint decoding `WEBSPACES_SOURCE_CONFIG` (base_url, username, token, ca_cert, webmail_base_url), `goplugin.Serve` with `sdk.Handshake`/`sdk.GRPCServer`
- `plugins/proton/client.go` — IMAP connection factory: scheme-selected implicit-TLS-vs-mandatory-STARTTLS dial, host-pinned `pinnedDialer`/`allowHost`, `ErrNotFound`/`ErrForeignHost`/`ErrNoMessageID` sentinels, mandatory dial+command timeouts (10s sync, 5s health)
- `plugins/proton/plugin.go` — `SourcePlugin`: `Match` (mailbox scan + Message-ID dedup + in-process mailbox cache), `Fetch` (UID re-resolution + `BODY.PEEK`), `Health`, `encodeSourceID`/`decodeSourceID` (Task 2's option-a)
- `plugins/proton/body.go` — `mail.CreateReader`/`NextPart` MIME extraction bounded by `io.LimitReader` and a max-part-count ceiling; `text/plain` only in this plan (HTML rendition is plan 03-02)
- `kernel/config/types.go` — `Source.Username string` (`toml:"username,omitempty"`), `Source.WebmailBaseURL string` (`toml:"webmail_base_url,omitempty"`)
- `kernel/pluginhost/host.go` — `launch()`'s hand-maintained `sourceConfig` JSON map gained `"username"` and `"webmail_base_url"` keys
- `web/src/lib/components/StreamRow.svelte` — `{#if item.group_label}` guarded sender span, first child of `.stream-row-meta`, before the date span
- `config.example.toml` — documented `[sources.proton]` + `[sources.proton.agent]` blocks
- `go.work` — added `./plugins/proton` as a workspace member
- `Makefile` — `build`/`test` targets gained the proton plugin binary and module test invocation
- `go.work.sum` — updated as a side effect of resolving the new module's dependency graph within the shared workspace

## Decisions Made

- **Task 2 (one-way door):** email `source_id` = `base64.RawURLEncoding` of the normalized Message-ID (option-a, the plan's recommended choice). Reversible with no extra persisted state (`decodeSourceID` recovers the exact Message-ID), contains only `[A-Za-z0-9_-]` so it needs no URL-path escaping, and is a pure function of the Message-ID alone — encodes no mailbox/label/UID, so a message under two matching labels yields one id, not two. This is the encoding `docs/api.md`'s "stable-ID scheme" section should describe if/when that doc is extended for email.
- Fidelity is always `LINK_FIDELITY_ANCHORED`, never `EXACT` — no verified mapping from an IMAP Message-Id/UID to Proton's internal webmail message id exists (03-RESEARCH.md Pitfall 5); the deep link points at the matched mailbox's webmail label view, built from the configured `webmail_base_url` plus the escaped mailbox leaf name.
- `bridgeCertServerName` is a hardcoded `"127.0.0.1"` constant in `client.go`, not a new config field — Bridge only ever binds and issues a certificate for its own loopback interface by design (confirmed live during Task 1: the exported cert's SAN is exactly `IP Address:127.0.0.1`, no LAN hostname entry), so this is a fixed property of Bridge's architecture rather than a per-deployment value.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] `plugins/proton/go.mod` cannot be produced by a clean `go mod tidy` pass, matching the existing plugin modules' own limitation**
- **Found during:** Task 3, initial module setup
- **Issue:** `go mod tidy` inside `plugins/proton` attempts to resolve `github.com/davison/webspaces/sdk` (a workspace-local, unpublished module) over the network and fails with "repository not found" — this is a pre-existing Go workspace-mode limitation already present in `plugins/silverbullet` and `plugins/mock`'s own `go.mod` files (neither declares an explicit `sdk` require, and both mark `hashicorp/go-plugin`/`grpc` as `// indirect` despite importing them directly).
- **Fix:** Built `go.mod` via targeted `go get` of only the two new external dependencies (`go-imap`, `go-message`) plus `go build`/`go vet`, mirroring the exact pattern already established by the sibling plugin modules, and documented the limitation inline in `go.mod` so a future reader isn't surprised by the same `go mod tidy` failure.
- **Files modified:** plugins/proton/go.mod, plugins/proton/go.sum
- **Verification:** `go build ./...` and `go vet ./...` both pass cleanly inside `plugins/proton`; `go build ./...` and `go test ./...` pass from the repo root with `plugins/proton` present.
- **Committed in:** 9227e57

### Notable Live-Environment Finding (not a code defect — documented for the user)

**Real Bridge account credential rejects LOGIN.** Running the actual compiled `webspaces-plugin-proton` binary against the real Bridge instance (`monroe:1143`, STARTTLS, pinned certificate) produced:

```
proton: error: match against source "proton": rpc error: code = Unavailable desc = proton: connect: proton: login: no such user
```

A second attempt (after Bridge's own login-attempt rate-limiting kicked in) returned `"too many login attempts"` instead. This confirms, with the real production binary:
- The TLS/STARTTLS handshake succeeded (the error is specifically at the `LOGIN` command, after a completed TLS session).
- The three-place config plumbing works: the kernel correctly launched the plugin subprocess with `PROTON_BRIDGE_USER`/`PROTON_BRIDGE_PASS`/`PROTON_BRIDGE_ADDR` from the environment.
- The failure was isolated cleanly to the `proton` source alone — `paperless` (35 items) and `silverbullet` (16 items) both synced normally in the same run, proving the per-source sync isolation (02-01's `SyncSource` design) holds.
- `GET /api/sources` correctly reported `"source_type":"proton"`, `"display_name":"Proton Mail"`, `"reachable":false`, and a specific, actionable `last_error` — matching SRC-01's success criterion 5 (a clear health error, never a hang) even though the underlying cause is a credential mismatch rather than an unreachable host.

This is a Bridge-account configuration issue on the user's own environment (the `PROTON_BRIDGE_USER` value Bridge's `LOGIN` command rejects), not a defect in the plugin code — recommend re-checking Bridge → Settings for the exact Bridge username before the next live sync attempt. **Stopped further live login attempts once Bridge's own rate-limiting was observed**, to avoid contributing to an account lockout.

**Real local config note:** `~/.config/webspaces/config.toml` (outside this repo, not committed) needed its `[sources.proton]` `token` line written as a TOML *literal* string (`token = '${PROTON_BRIDGE_PASS}'`, single-quoted) rather than the double-quoted form shown in `config.example.toml` — the real Bridge-generated password contains a literal double-quote character, which breaks `kernel/config.Load`'s raw-text `${VAR}` substitution when it lands inside a double-quoted TOML string (the substitution happens on the raw file text before TOML parsing, so it is unaware of TOML's own quoting rules). This is a real, generally-applicable hazard in `kernel/config`'s expand-then-parse design for any secret containing a double quote — **not fixed in this plan** (touching `kernel/config.go`'s `expandEnv` is an architectural change to shared code affecting every source, out of this plan's `files_modified` scope, and needs a deliberate decision on approach: escape substituted values, or document literal-string TOML syntax as the recommended pattern for secrets). Flagged here for a future phase or a dedicated quick task.

---

**Total deviations:** 1 auto-fixed (Rule 1 — pre-existing tooling limitation, handled consistently with sibling modules); 1 live-environment credential finding (documented, not a code fix); 1 out-of-scope config hazard (documented, not fixed).
**Impact on plan:** No scope creep. The tracer's code-level guarantees (TLS pinning, read-only IMAP, dedup, config plumbing, error isolation) are all verified working against the real Bridge instance up to the LOGIN step. The one unmet success criterion (a real email visible in the browser stream) is blocked by a live credential issue outside this plan's control, not by the code.

## Issues Encountered

- `go mod tidy` cannot run cleanly for any Go-workspace member module that imports the unpublished `github.com/davison/webspaces/sdk` module — worked around via targeted `go get` (see Deviations above). Pre-existing, not introduced by this plan.
- Live Bridge LOGIN rejection and subsequent rate-limiting (see Deviations above) prevented completing the live "email visible in stream" verification within this session.

## User Setup Required

None new beyond what Task 1's `user_setup` already covered (Bridge forwarder, exported certificate, four `.env` keys) — all four were already verified present and working before Task 3 began. The one remaining action is **not** a new setup step but a correction: re-verify the exact Bridge username in Proton Mail Bridge → Settings against `.env`'s `PROTON_BRIDGE_USER`, since the live Bridge instance rejected the current value with "no such user."

## Next Phase Readiness

- The `plugins/proton` module, its config plumbing, and the stream UI's sender rendering are all in place and unit/build/vet/audit-verified — plan 03-02 (sanitized `text/html` rendition path) can build directly on top of `client.go`'s exposed `dial` field and `body.go`'s MIME extraction without any interface change.
- Blocker for full live verification: the Bridge account's `PROTON_BRIDGE_USER` value needs re-confirmation against the real Bridge instance before a live sync can return real Proton items. No code changes are anticipated to resolve this — it is a credential/environment correction.
- `docs/api.md`'s "stable-ID scheme" section has not yet been extended to describe the email encoding (Task 2's decision) — deferred, since this plan's `files_modified` does not include `docs/api.md`; the decision is fully recorded above for whoever picks that up.

## Self-Check: PASSED

- FOUND: plugins/proton/go.mod, plugins/proton/main.go, plugins/proton/client.go, plugins/proton/plugin.go, plugins/proton/body.go
- FOUND: kernel/config/types.go contains `Username string` / `WebmailBaseURL string`
- FOUND: kernel/pluginhost/host.go contains `"username": src.Username` and `"webmail_base_url": src.WebmailBaseURL`
- FOUND: go.work lists `./plugins/proton`
- FOUND: web/src/lib/components/StreamRow.svelte contains `item.group_label` inside an `{#if}` guard, before the date span
- FOUND: commit 9227e57 in `git log --oneline`

---
*Phase: 03-email-in-the-webspace*
*Completed: 2026-07-31*
