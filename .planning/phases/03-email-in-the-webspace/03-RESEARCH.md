# Phase 3: Email in the Webspace - Research

**Researched:** 2026-07-29
**Domain:** Go IMAP client against Proton Mail Bridge (self-signed TLS, LAN exposure) + SQLite FTS5 full-text search + safe HTML email rendering
**Confidence:** MEDIUM — IMAP client mechanics and FTS5 are HIGH (verified directly against this repo's own dependencies and the library's own source); Proton Mail Bridge's LAN-exposure and deep-link behavior are MEDIUM (community-sourced, not officially documented, genuinely unverifiable without the user's live home-server Bridge instance — this is exactly the "spike" the roadmap flagged).

## Summary

This phase adds the third source plugin (IMAP against Proton Mail Bridge) and the first cross-source capability (SQLite FTS5 search). Both land on infrastructure this repo already has proven twice: the plugin contract/host pattern (paperless-ngx, SilverBullet) and a `modernc.org/sqlite` index whose schema was **deliberately pre-shaped for FTS5** since Phase 1 (`kernel/index/schema.go`'s own comment says so). This research verified that pre-shaped design works end-to-end against this repo's actual pinned SQLite dependency (`modernc.org/sqlite v1.54.0`) — external-content table, sync triggers, `bm25()` ranking, and `snippet()` all ran successfully in a throwaway test program built against the real go.mod.

The two genuinely hard, unverified-until-live-spike problems are exactly what the roadmap called out: (1) Proton Mail Bridge does **not** support binding to a LAN interface — it hard-binds `127.0.0.1` — so reaching it from the desktop requires a TCP forwarder (`socat`/`stunnel`) run on the home server, and the Bridge's self-signed certificate is issued for `127.0.0.1`, not the LAN hostname, which breaks default TLS hostname verification when connecting through a forwarder; and (2) there is no confirmed mapping between an IMAP `Message-ID`/UID and Proton's internal webmail message ID, so an `EXACT`-fidelity deep link into `mail.proton.me` for a specific message cannot be guaranteed — a folder-level `ANCHORED` link is the defensible default pending live confirmation.

Everything else — `BODY.PEEK[]` for the never-mark-read guarantee, `EXAMINE` (not `SELECT`) as a second, protocol-level read-only guarantee, `ENVELOPE.MessageId` for dedup, and a bluemonday-based HTML sanitization pipeline that reuses the SilverBullet plugin's exact rendition-serving path — is HIGH confidence, verified directly against the pinned library source or this repo's own code.

**Primary recommendation:** Build the email plugin as `plugins/proton` (or `plugins/imap` — naming is planner's call) following the SilverBullet plugin's exact shape (custom host-pinned dialer, `ca_cert`-based TLS trust, bluemonday+iframe rendition path for HTML bodies), using `github.com/emersion/go-imap` v1 (already the project's locked choice) with `Select(mailbox, /*readOnly=*/true)` + `imap.BodySectionName{Peek: true}` as two independent read-only guarantees, and `github.com/emersion/go-message/mail` for MIME part extraction at `Fetch` time. Implement search (KERN-05) as an FTS5 external-content table added to the existing `items` table with zero schema migration risk (already anticipated), exposed as a new `GET /api/webspaces/{webspace}/search?q=` route.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| IMAP connect/auth/TLS trust to Proton Bridge | API/Backend (email plugin subprocess) | — | Plugin owns its own source-system connection, per the established contract (PLUG-01) |
| Never-mark-read guarantee (`BODY.PEEK`, `EXAMINE`) | API/Backend (email plugin) | — | Protocol-level behavior; must be enforced at the IMAP command layer, nowhere else can guarantee it |
| Message-ID dedup across matched labels | API/Backend (email plugin, at `Match` time) | Database/Storage (kernel index's `id` primary key as a structural backstop) | Plugin must merge labels per Message-ID before returning `Match` results (see Pitfall below) — the index's `ON CONFLICT(id)` upsert only prevents duplicate *rows*, it does not merge label lists across multiple items proposed in the same response |
| Outbound host allowlist (LAN Bridge host + loopback only) | API/Backend (email plugin's custom `imap.Dialer`) | — | Same pattern as `plugins/silverbullet/client.go`'s `allowHost`, adapted to `go-imap`'s `Dialer` interface instead of `http.Transport.DialContext` |
| HTML email body sanitization | API/Backend (email plugin, at `Fetch` time) | Browser/Client (sandboxed iframe + kernel-set CSP, already enforced generically by `kernel/httpapi/item.go`'s `renditionHandler`) | Two independent layers, exactly the pattern already established for SilverBullet's rendered markdown — no kernel change needed here, `text/html` is already an allowlisted rendition MIME type |
| Full-text search index maintenance | Database/Storage (SQLite FTS5 triggers on `items`) | — | Triggers keep the FTS index synced on every `UpsertItems`/`ReplaceWebspaceSourceItems` write; no application-code sync logic needed |
| Search query + ranking | API/Backend (new kernel HTTP route reading the FTS5 table) | — | Same "httpapi never reaches a plugin" rule as `StreamHandler` — search reads only the index, never calls a plugin |
| Search UI (query box, ranked results, highlighted snippets) | Browser/Client (SvelteKit) | — | New control per phase notes; renders `snippet()`-highlighted results, reuses `StreamRow`-style presentation |
| Sender display in stream row / detail pane | Browser/Client (SvelteKit) | — | **Currently missing** — `group_label` (the field the proto's own comment designates for "mail conversation") flows correctly through the whole backend/TS type chain today but is never rendered by `StreamRow.svelte` or `DetailPane.svelte`. This phase must add that rendering, not just populate the field. |

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SRC-01 | Email plugin (IMAP) works against Proton Mail Bridge (self-signed cert handling); uses `BODY.PEEK` so mail is never marked read; matches webspace keyword against folders/labels; dedups by Message-ID | See "Standard Stack", "Architecture Patterns" (TLS pinning, EXAMINE+PEEK, folder/label matching, Message-ID dedup), "Common Pitfalls", "Code Examples" |
| KERN-05 | User can full-text search within a webspace (FTS5 over indexed metadata/previews) | See "Architecture Patterns: FTS5 external-content search" (verified end-to-end against this repo's pinned SQLite dependency) + "Code Examples" |
</phase_requirements>

## Project Constraints (from CLAUDE.md)

The project's `.claude/CLAUDE.md` locks these choices relevant to this phase — research below does not re-litigate them, only fills in the implementation detail CLAUDE.md left open:

- **`github.com/emersion/go-imap` v1 (IMAP4rev1), not v2** — v2 is still beta per its own maintainers. Confirmed still current guidance; v1.2.1 (2022, stable) is what this research verified against.
- **Plugins are separately-built Go modules launched as subprocesses over `hashicorp/go-plugin` gRPC** — the email plugin follows the exact `plugins/paperless` / `plugins/silverbullet` shape (own `go.mod`, `main.go` reading `WEBSPACES_SOURCE_CONFIG`, implements `sdk.SourcePlugin`).
- **Kernel index is `modernc.org/sqlite` (pure Go, no cgo), FTS5 compiled in** — verified locally in this session (see below); no build-tag or driver change needed for KERN-05.
- **Read-only by construction (PLUG-02)** — this phase's read-only guarantee is *stronger* than "don't call a mutating RPC": IMAP has its own mutating commands (`STORE`, `EXPUNGE`, `MOVE`, `COPY`, `APPEND`, `DELETE`) that the existing repo-wide AST scanner (`TestPluginsIssueOnlyGetRequests`, `net/http`-specific) does **not** catch, because IMAP doesn't use `net/http` at all. See "Don't Hand-Roll" and "Common Pitfalls" for the IMAP-specific enforcement this phase needs to add.
- **Never leave the desktop / all data stays local** — the Bridge-forwarder (`socat`/`stunnel`) work happens entirely on the home server the user already controls (same LAN trust boundary as the existing paperless-ngx and SilverBullet instances); no new cloud dependency.

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `github.com/emersion/go-imap` | v1.2.1 `[VERIFIED: go get + go list -m, published 2022-05-01]` | IMAP4rev1 client against Proton Mail Bridge | Locked by CLAUDE.md; de facto standard Go IMAP client (used by aerc, Delta Chat desktop tooling); this session verified its `BODY.PEEK`, `EXAMINE`, and `ENVELOPE.MessageId` mechanics directly against the v1.2.1 source |
| `github.com/emersion/go-message` (specifically its `mail` subpackage) | v0.18.2 `[VERIFIED: go get + go list -m, published 2024-09-28]` | MIME parsing at `Fetch` time — separates `text/plain` from `text/html` body parts, decodes charset/transfer-encoding | Same author as `go-imap`; used by aerc; the `mail.Reader`/`mail.NextPart()` API is exactly the shape needed to pick the HTML part (preferred) or fall back to plain text |
| `github.com/microcosm-cc/bluemonday` | v1.0.27 `[VERIFIED: already in plugins/silverbullet/go.mod at this exact version — zero new risk, already running in this codebase]` | Sanitize untrusted HTML email bodies before serving as a rendition | Already the project's chosen HTML sanitizer (SilverBullet's rendered markdown); reusing it for email keeps one sanitization dependency instead of two |
| `modernc.org/sqlite` | v1.54.0 (already in go.mod) | FTS5 external-content table for KERN-05 | No new dependency — already pure-Go with FTS5 compiled in; **verified locally in this session** (see Architecture Patterns) |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `golang.org/x/net` | already indirect in go.mod | Transitively required by `go-imap`/`go-message` | No direct import needed |
| `net`, `crypto/tls` (stdlib) | — | Custom host-pinned `imap.Dialer` + TLS certificate handling | Same stdlib-only approach `plugins/silverbullet/client.go` already uses for its outbound allowlist |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `go-imap` v1 | `go-imap/v2` (`imapclient`) | Rejected by CLAUDE.md already — still beta, maintainers say not production-ready |
| `go-message/mail` for MIME parsing | Hand-rolled `net/mail` + manual multipart walking | `net/mail` alone doesn't decode multipart/alternative or non-UTF-8 charsets; `go-message/mail` is a thin, purpose-built layer over the same RFC 2045/2046/2047 concerns SilverBullet's `adrg/frontmatter` choice already established the project's preference for well-scoped parsing libraries over hand-rolled ones |
| bluemonday for email HTML | A dedicated "email HTML sanitizer" library | None with meaningful Go ecosystem adoption exists beyond bluemonday itself — bluemonday's own repo ships a **worked example specifically for HTML email** (`cmd/sanitise_html_email/main.go`), which this research fetched and verified; reuse that policy shape rather than inventing one |
| Self-hosted forwarder (`socat`) for Bridge LAN exposure | Recompiling Proton Bridge from source with the host-binding patched to `0.0.0.0` | Recompiling is unofficial, loses Proton's signed release process, and must be redone on every Bridge update. `socat`/`stunnel` port-forwarding requires zero changes to the Bridge binary itself and is the community-documented approach. Recommended over the source-patch route. |

**Installation:**
```bash
cd plugins/proton   # or plugins/imap — planner's naming call
go mod init github.com/davison/webspaces/plugins/proton
go get github.com/emersion/go-imap@v1.2.1
go get github.com/emersion/go-message@v0.18.2
go get github.com/microcosm-cc/bluemonday@v1.0.27
```

**Version verification:** All three versions above were confirmed via `go get`/`go list -m` against the real module registry in this session (see Package Legitimacy Audit for full detail) — not taken from training-data memory.

## Package Legitimacy Audit

> The automated `gsd-tools query package-legitimacy check` seam was not available in this environment (this installation of `gsd-tools` predates that command). Verification below was performed manually: `go get`/`go list -m -json` against the real Go module proxy (confirms registry existence, resolved version, and publish timestamp) plus cross-checking real-world adoption via web search (confirms non-slopsquatted, actively-used packages).

| Package | Registry | Age | Known Usage | Source Repo | Verdict | Disposition |
|---------|----------|-----|-------------|-------------|---------|-------------|
| `github.com/emersion/go-imap` | Go proxy | v1.2.1 published 2022-05-01 (stable v1 line; project already depends on this exact library per CLAUDE.md) | Used by Delta Chat's desktop tooling ecosystem, aerc, dozens of IMAP-adjacent Go tools | github.com/emersion/go-imap | OK | Approved — already the project's locked choice |
| `github.com/emersion/go-message` | Go proxy | v0.18.2 published 2024-09-28 | Used by `aerc` (a well-known terminal email client); same author (`emersion`) as `go-imap`, `go-message` is that author's established MIME-parsing companion library | github.com/emersion/go-message | OK | Approved |
| `github.com/microcosm-cc/bluemonday` | Go proxy | v1.0.27 published 2024-07-04 | Already a direct dependency of `plugins/silverbullet` in this exact repo at this exact version | github.com/microcosm-cc/bluemonday | OK | Approved — zero new risk, already running in production in this codebase |

**Packages removed due to [SLOP] verdict:** none
**Packages flagged as suspicious [SUS]:** none

*All three packages above were confirmed via `go get`/`go list -m` against the live module proxy in this research session (not training-data memory), and cross-checked against real, named, actively-maintained consumer projects — treat as `[VERIFIED: Go module proxy + go-imap/go-message/bluemonday known-usage cross-check]`, not `[ASSUMED]`.*

## Architecture Patterns

### System Architecture Diagram

```
                          Home server (LAN)
   ┌───────────────────────────────────────────────────────┐
   │  Proton Mail Bridge (binds 127.0.0.1 only)             │
   │    IMAP :1143, self-signed cert ~/.config/protonmail/  │
   │                        bridge/{cert,key}.pem            │
   │              ▲                                          │
   │              │ 127.0.0.1                                │
   │  socat/stunnel forwarder (LAN-bound port → 127.0.0.1)  │
   └──────────────┬────────────────────────────────────────┘
                   │ TLS (Bridge's self-signed cert,
                   │ pinned + ServerName override — see Pitfall)
                   ▼
   ┌───────────────────────────────────────────────────────┐
   │  Desktop machine — webspaces kernel process             │
   │                                                          │
   │  ┌─────────────────────────────────────────────────┐   │
   │  │ email plugin subprocess (plugins/proton)          │   │
   │  │  - custom imap.Dialer (host-pinned)               │   │
   │  │  - Match(): LIST → filter Folders/Labels by       │   │
   │  │    webspace keyword → EXAMINE each → FETCH        │   │
   │  │    ENVELOPE+INTERNALDATE → dedup by Message-Id    │   │
   │  │  - Fetch(): EXAMINE remembered mailbox → SEARCH   │   │
   │  │    HEADER Message-Id → FETCH BODY.PEEK[] →        │   │
   │  │    go-message/mail parse → bluemonday sanitize    │   │
   │  └──────────────────┬──────────────────────────────┘   │
   │                     │ gRPC (Match/Fetch/Health/Describe) │
   │  ┌──────────────────▼──────────────────────────────┐   │
   │  │ kernel/correlate.SyncSource (unchanged)           │   │
   │  │  - persists items, id = "email:{message-id-hash}" │   │
   │  └──────────────────┬──────────────────────────────┘   │
   │  ┌──────────────────▼──────────────────────────────┐   │
   │  │ kernel/index (SQLite)                             │   │
   │  │  items table  ──AFTER INSERT/UPDATE/DELETE──▶     │   │
   │  │                          items_fts (FTS5, bm25)   │   │
   │  └──────────────────┬──────────────────────────────┘   │
   │  ┌──────────────────▼──────────────────────────────┐   │
   │  │ kernel/httpapi                                    │   │
   │  │  GET /api/webspaces/{ws}/stream   (unchanged)      │   │
   │  │  GET /api/webspaces/{ws}/search?q=  (NEW, KERN-05) │   │
   │  │  GET /api/items/{id}/content  (unchanged route,    │   │
   │  │    text/html already allowlisted — email body      │   │
   │  │    reuses SilverBullet's exact rendition path)     │   │
   │  └──────────────────┬──────────────────────────────┘   │
   │                     ▼                                    │
   │            SvelteKit SPA: StreamRow/DetailPane           │
   │            (add sender display + search box — the        │
   │             two UI changes this phase needs)             │
   └───────────────────────────────────────────────────────┘
```

### Recommended Project Structure

```
plugins/
└── proton/                  # (naming: planner's call — "proton" or "imap")
    ├── go.mod
    ├── main.go               # WEBSPACES_SOURCE_CONFIG parse, goplugin.Serve — same shape as paperless/silverbullet
    ├── plugin.go              # SourcePlugin: Describe/Match/Fetch/Health
    ├── client.go              # go-imap connection, host-pinned Dialer, TLS pinning
    ├── mailbox_scan.go         # LIST + folder/label keyword matching, EXAMINE, ENVELOPE fetch, dedup
    ├── body.go                # go-message/mail extraction + bluemonday sanitize + WrapDocument-equivalent
    ├── outbound_hosts_test.go  # host-allowlist predicate test (mirrors silverbullet's)
    └── readonly_test.go        # NEW pattern: AST/identifier scan forbidding IMAP mutating commands (see Don't Hand-Roll)

kernel/
├── config/types.go            # add Source.Username (new field, mirrors CACert precedent)
├── pluginhost/host.go          # launch(): add "username" to the sourceConfig JSON map (currently hardcodes exactly 4 keys — see Pitfall)
├── index/schema.go             # add items_fts virtual table + 3 triggers (additive migration, IF NOT EXISTS-safe)
├── index/store.go              # add Search(ctx, webspaceName, query) method
└── httpapi/
    ├── search.go               # NEW: GET /api/webspaces/{webspace}/search?q=
    └── routes.go                # register the new route

web/src/lib/
├── api.ts                     # add SearchResponse type + searchWebspace()
└── components/
    ├── StreamRow.svelte        # render item.group_label (sender) — currently unrendered despite flowing through the whole stack
    ├── DetailPane.svelte       # same — header currently shows title/date/labels only, no sender
    └── SearchBox.svelte        # NEW control
```

### Pattern 1: Two independent read-only guarantees (protocol + command)

**What:** Every mailbox this plugin opens is opened with `EXAMINE`, not `SELECT` — `client.Select(mailboxName, /*readOnly=*/true)` in `go-imap` v1 issues `EXAMINE` per RFC 3501, which makes the *server itself* refuse any `STORE`/`EXPUNGE` against that session. Independently, every body fetch uses `imap.BodySectionName{Peek: true}`, which serializes to `BODY.PEEK[...]` instead of `BODY[...]` — the specific mechanism that avoids the server *implicitly* setting `\Seen` on a plain read.

**When to use:** Every single mailbox open and every single body fetch in this plugin, with no exceptions — this is the direct implementation of SRC-01's success criterion 2 ("reading an email inside webspaces never marks it read in Proton").

**Example:**
```go
// Source: github.com/emersion/go-imap v1.2.1, client/cmd_auth.go (Select) and
// message.go lines 354-359 (BodySectionName.Peek) — verified directly against
// the pinned library source in this research session.

// 1. Protocol-level guarantee: EXAMINE, not SELECT.
mbox, err := c.Select(mailboxName, true) // readOnly=true -> IMAP EXAMINE command
if err != nil {
    return nil, fmt.Errorf("proton: examine %q: %w", mailboxName, err)
}

// 2. Command-level guarantee: BODY.PEEK[], not BODY[].
section := &imap.BodySectionName{Peek: true}
items := []imap.FetchItem{section.FetchItem(), imap.FetchEnvelope, imap.FetchUid}

seqset := new(imap.SeqSet)
seqset.AddNum(uid)
messages := make(chan *imap.Message, 1)
done := make(chan error, 1)
go func() { done <- c.UidFetch(seqset, items, messages) }()

msg := <-messages
r := msg.GetBody(section) // never nil if Peek fetch succeeded
if err := <-done; err != nil {
    return nil, err
}
```

**Automated verification for success criterion 2:** write an integration test (requires live Bridge access, so likely `checkpoint:human-verify`-gated or environment-conditional) that: (a) records a target message's `\Seen` flag state before any plugin call via a *second*, separate IMAP connection; (b) runs a full `Match` + `Fetch` cycle through the plugin; (c) re-checks the flag via the second connection and asserts it is unchanged. This mirrors how the phase's own success criterion is worded ("proven by an automated test").

### Pattern 2: Folder/Label matching and Message-ID dedup

**What:** Proton Mail Bridge exposes Proton's labels as IMAP mailboxes under a `Labels/` prefix and folders under a `Folders/` prefix `[CITED: community-sourced, cross-referenced across two independent sources — Proton's label model is "non-destructive": a message keeps living in one place but can appear under several `Labels/*` mailboxes simultaneously, unlike a `Folders/*` move which physically relocates it]`. This means the **same message can legitimately appear during `LIST`+scan under multiple mailboxes** that all match the webspace's keyword — the literal mechanism SRC-01's dedup requirement exists to handle.

**When to use:** `Match()`'s core loop.

**Example:**
```go
// dedupe map keyed by Message-ID, built while scanning every keyword-matched mailbox
type matched struct {
    envelope *imap.Envelope
    uid      uint32
    mailbox  string   // the FIRST mailbox this message was found in — used by Fetch's mailbox-lookup cache (see Critical Architecture Finding)
    labels   []string // every matched mailbox's leaf name, deduplicated
}

byMessageID := map[string]*matched{}

for _, mbox := range matchedMailboxes {
    if _, err := c.Select(mbox.Name, true); err != nil { // EXAMINE
        return nil, fmt.Errorf("proton: examine %q: %w", mbox.Name, err)
    }
    // ... FETCH 1:* ENVELOPE INTERNALDATE UID ...
    for _, msg := range fetched {
        id := msg.Envelope.MessageId
        if id == "" {
            continue // a message with no Message-Id header cannot be safely deduped or re-fetched later — skip (log it)
        }
        leaf := leafName(mbox.Name) // e.g. "Labels/ProjectX" -> "ProjectX"
        if m, ok := byMessageID[id]; ok {
            m.labels = appendUnique(m.labels, leaf) // MERGE labels — do not overwrite
            continue
        }
        byMessageID[id] = &matched{envelope: msg.Envelope, uid: msg.Uid, mailbox: mbox.Name, labels: []string{leaf}}
    }
}
```

**This is the load-bearing detail:** relying on the kernel's `ON CONFLICT(id) DO UPDATE` upsert (which does prevent duplicate *rows*) is **not sufficient** on its own — if `Match()` naively returned one `Item` per (mailbox, message) pair without merging, the *last* one processed would silently overwrite the `labels` field of every earlier one for the same message, discarding real label information the user configured their webspace around. Dedup must happen in the plugin before `MatchResponse` is built, keyed by `Envelope.MessageId`.

### Pattern 3: FTS5 external-content search (KERN-05) — verified end-to-end

**What:** `kernel/index/schema.go` already anticipated this exact design (its own comment: *"an external-content FTS5 virtual table can be added over items(title, preview) in Phase 3 (KERN-05) with content='items', content_rowid='rowid' and no migration"*). This research **ran that exact design against the project's real, pinned `modernc.org/sqlite v1.54.0` dependency** in a throwaway Go program and confirmed: table creation, all three sync triggers, insert/update/delete, `MATCH` query, `bm25()` ranking, and `snippet()` highlighting all work correctly — including with the `items` table's `TEXT PRIMARY KEY` composite id (`"{source_type}:{source_id}"`) preserved via the table's separate hidden `rowid` column (a `TEXT PRIMARY KEY` does **not** alias `rowid` the way `INTEGER PRIMARY KEY` does, so `content_rowid='rowid'` correctly refers to that distinct hidden column, and joining back via `items.rowid = items_fts.rowid` correctly recovers each row's real `id`).

**When to use:** KERN-05's entire implementation — schema addition, sync mechanism, and query.

**Example (schema — additive, `IF NOT EXISTS`-safe like the rest of `schema.go`):**
```sql
-- Source: sqlite.org/fts5.html "External Content Tables", adapted to this
-- repo's items(id TEXT PRIMARY KEY, title, preview, ...) shape — verified
-- working end-to-end against modernc.org/sqlite v1.54.0 in this session.

CREATE VIRTUAL TABLE IF NOT EXISTS items_fts USING fts5(
  title, preview, content='items', content_rowid='rowid'
);

CREATE TRIGGER IF NOT EXISTS items_ai AFTER INSERT ON items BEGIN
  INSERT INTO items_fts(rowid, title, preview) VALUES (new.rowid, new.title, new.preview);
END;

CREATE TRIGGER IF NOT EXISTS items_ad AFTER DELETE ON items BEGIN
  INSERT INTO items_fts(items_fts, rowid, title, preview) VALUES('delete', old.rowid, old.title, old.preview);
END;

CREATE TRIGGER IF NOT EXISTS items_au AFTER UPDATE ON items BEGIN
  INSERT INTO items_fts(items_fts, rowid, title, preview) VALUES('delete', old.rowid, old.title, old.preview);
  INSERT INTO items_fts(rowid, title, preview) VALUES (new.rowid, new.title, new.preview);
END;
```

**Example (query, scoped to one webspace via a join through `webspace_items` — mirrors `StreamItems`'s existing join pattern):**
```go
// Source: this session's own verified test program, adapted to this repo's
// webspace_items join (StreamItems already establishes this exact join
// shape in kernel/index/store.go).
const searchQuery = `
SELECT items.id, items.source_type, items.title, items.preview,
       items.timestamp_unix, items.deep_link, items.fidelity,
       snippet(items_fts, -1, '‹', '›', '…', 12) AS snip,
       bm25(items_fts) AS rank
FROM items_fts
JOIN items ON items.rowid = items_fts.rowid
JOIN webspace_items ON webspace_items.item_id = items.id
WHERE webspace_items.webspace_name = ?
  AND items_fts MATCH ?
ORDER BY rank ASC
LIMIT 50
`
```

Note `bm25()` returns **more-negative-is-better** — `ORDER BY rank ASC` is correct, not `DESC` (a common mistake `[CITED: sqlite.org/fts5.html]`).

**FTS5 query syntax caveat:** a raw user query string passed directly to `MATCH` is interpreted as FTS5 query syntax (supports `AND`/`OR`/`NOT`/phrase quotes/prefix `*` — useful, but also means a lone `"` or a token starting with `-` can produce a syntax error, not just "no results"). Wrap the raw query in a helper that either escapes/quotes it as a phrase or catches the syntax-error case and returns an empty result set rather than a 500 — do not hand the UI's raw search-box text straight to `MATCH` unescaped.

### Critical Architecture Finding: Fetch-time mailbox lookup

**Problem:** `FetchRequest` carries only `source_id` and a `ContentVariant` — no webspace context, no mailbox name. But IMAP has no cross-mailbox "fetch by Message-ID" operation; you must `SELECT`/`EXAMINE` a specific mailbox, then `SEARCH HEADER Message-Id "<...>"` within it to resolve the current UID (UIDs are only meaningful within one `SELECT`ed mailbox, and can be reassigned if `UIDVALIDITY` changes, so re-searching by Message-ID at fetch time rather than trusting a cached UID is the robust approach `[ASSUMED — standard IMAP practice, not specific to this project]`). So `Fetch()` needs to know *which mailbox* to open for a given `source_id`, and `Match()` is the only place that ever learns that.

**Recommended design:** since this repo's plugin subprocesses are launched once at kernel startup and live for the kernel process's entire lifetime (`kernel/pluginhost/host.go` — `Kill()` only happens at `Shutdown()`, never between syncs) `[VERIFIED: cmd/webspaces/main.go's defer host.Shutdown() plus the absence of any relaunch-per-sync code path]`, the plugin can hold a simple in-process `map[string]string` (Message-ID → mailbox name) populated every time `Match()` runs, and consulted by `Fetch()`. Because the kernel always runs a startup sync before the UI can show any item to click (`kernel/syncer` — startup sync is the scheduler's first scheduled run, established in Phase 2), this cache will be populated before any `Fetch` call could plausibly reference an id it doesn't know about, **except** immediately after a kernel restart before the very first sync completes — an acceptable, documented MVP edge case, not a silent failure (`Fetch` should return a clear `NotFound`/`Unavailable` in that narrow window, which the kernel already maps to the existing `content_unavailable`/`source_unavailable` UI states).

This is a plugin-internal implementation detail, not part of the wire contract — flagged here because it's exactly the kind of non-obvious cross-RPC state management a planner needs to know about before writing tasks, not something obvious from the proto alone.

### Pattern 4: TLS trust against a self-signed cert issued for a different hostname

**What:** Proton Mail Bridge generates a self-signed cert at `~/.config/protonmail/bridge/{cert,key}.pem` on first run `[CITED: cross-referenced Proton support docs + community source]`, issued for `127.0.0.1` (the only address Bridge itself ever binds). Once a `socat`/`stunnel` forwarder exposes it on the LAN, the Go TLS client is dialing a *different* address than the cert's Subject/SAN — default hostname verification (`tls.Config.ServerName` inferred from the dial address) will fail even though the cert itself is legitimately trusted once exported.

**When to use:** Building the plugin's `tls.Config`.

**Example:**
```go
// Recommended pattern — reuses the SilverBullet plugin's exact caCertPath
// convention (config.Source.CACert, already a generic field, already
// expanded/home-dir-resolved by kernel/config), extended with an explicit
// ServerName override. [ASSUMED: standard documented crypto/tls behavior
// — tls.Config.ServerName controls the hostname checked against the
// cert's SAN independently of the actual network address dialed — not
// verified against a live Bridge instance in this session; confirm the
// cert's actual SAN entries during the phase's Task-1 spike via
// `openssl x509 -in cert.pem -noout -text` against the real exported cert.]
tlsConfig := &tls.Config{
    ServerName: "127.0.0.1", // matches the Bridge cert's own SAN, NOT the LAN forwarder's hostname/IP
}
if caCertPath != "" {
    if pemBytes, err := os.ReadFile(caCertPath); err == nil {
        pool := x509.NewCertPool()
        if pool.AppendCertsFromPEM(pemBytes) {
            tlsConfig.RootCAs = pool
        }
    }
}
```

**Anti-pattern to avoid explicitly:** `tls.Config{InsecureSkipVerify: true}` with no companion check. This disables *all* verification (chain trust, hostname, expiry) and is a documented anti-pattern for exactly this "self-signed cert, wrong hostname" situation — the `ServerName` override above achieves the same practical goal (connect through a forwarder to a cert that wasn't issued for that forwarder's address) while keeping full chain and expiry verification intact via `RootCAs`.

### Anti-Patterns to Avoid

- **Blanket `InsecureSkipVerify: true`:** see Pattern 4 above — use `ServerName` override + `RootCAs` pinning instead.
- **Bearer-token-shaped config for IMAP:** `config.Source` currently hardcodes exactly 4 fields into the plugin's env-var JSON (`base_url`, `token`, `api_version`, `ca_cert`) inside `kernel/pluginhost/host.go`'s `launch()` function — adding IMAP support means this function needs a new field, it will **not** happen automatically just by adding a field to the `config.Source` struct. See Pitfall below.
- **Encoding the matched mailbox into `source_id`:** would break dedup (SRC-01 criterion 3) outright, since the same message discovered under two different `Labels/*` mailboxes would then get two different `source_id`s and appear twice in the stream. Keep `source_id` as a stable hash/encoding of the Message-ID alone; solve the Fetch-time mailbox lookup problem separately (see Critical Architecture Finding above).
- **Injecting sanitized HTML via Svelte's raw-HTML directive:** exactly the anti-pattern Phase 2's own research already called out for SilverBullet — serve the sanitized document through the kernel's existing `/api/items/{id}/content` route and render it in the existing sandboxed `<iframe>`, never `{@html ...}` in the SPA document itself.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| MIME multipart parsing (text/plain vs text/html, charset decoding) | A regex- or string-split-based MIME parser | `github.com/emersion/go-message/mail` (`mail.CreateReader` + `NextPart()`) | RFC 2045/2046/2047 have enough edge cases (nested multipart, base64/quoted-printable transfer encoding, non-UTF-8 charsets) that a hand-rolled parser will silently mis-render real-world email; this is exactly the class of problem CLAUDE.md's own `emersion/go-message` recommendation exists to prevent |
| HTML sanitization for untrusted email bodies | A custom tag/attribute blocklist | `bluemonday`, using the library's own published email-specific policy as a starting point | Manually blocklisting is the canonical way XSS sanitizers get bypassed (bluemonday's own README explicitly warns against attempting a custom CSS/style sanitizer) |
| Full-text search / ranking | A custom inverted index, LIKE '%term%' scan, or an external search service | SQLite FTS5 (already compiled into the project's pinned SQLite dependency, already anticipated by the existing schema) | Matches CLAUDE.md's own explicit "What NOT to Use: an external search engine" guidance — FTS5 is proven sufficient at personal-data scale and requires zero new infrastructure |
| IMAP read-only enforcement | Trusting code review alone | A mechanical test (AST or identifier scan) forbidding `client.Store`, `client.Expunge`, `client.Move`, `client.Append`, `client.Delete`, `client.Copy` anywhere under the plugin's package, mirroring `plugins/paperless/readonly_test.go`'s `net/http`-verb scan but targeted at `go-imap`'s mutating methods | PLUG-02's existing mechanical enforcement (`sdk/contract_test.go`'s RPC allowlist + the repo-wide `net/http` AST scan) does not cover IMAP at all — a plugin author could call `client.Store` and nothing in the existing test suite would catch it |
| TLS certificate pinning | A hand-rolled fingerprint comparison in `VerifyPeerCertificate` | `tls.Config.ServerName` override + `RootCAs` pool from the exported cert (Pattern 4 above) | Simpler, uses only documented `crypto/tls` behavior, and reuses the exact `ca_cert` config convention already established for SilverBullet — no new cryptographic code to get wrong |

**Key insight:** every "don't hand-roll" item in this phase already has a proven precedent *inside this exact repository* (SilverBullet's sanitizer, SilverBullet's cert-pinning field, paperless's read-only AST scan) — the work here is extension and adaptation of established patterns to IMAP's different transport, not invention of new patterns.

## Common Pitfalls

### Pitfall 1: Kernel-side plumbing for a new `config.Source` field doesn't happen automatically

**What goes wrong:** A planner adds `Username string` to `config.Source` (needed because IMAP auth is username+password, not a bearer token), confirms `config.Validate()` doesn't reject it, and assumes the plugin will receive it — but `kernel/pluginhost/host.go`'s `launch()` function **hardcodes** the exact set of keys serialized into `WEBSPACES_SOURCE_CONFIG`:
```go
sourceConfig, err := json.Marshal(map[string]string{
    "base_url": src.BaseURL, "token": src.Token,
    "api_version": src.APIVersion, "ca_cert": src.CACert,
})
```
A new field added to the struct is silently dropped here unless this map is also updated.

**Why it happens:** the config struct and the env-var serialization are two separate pieces of code that must be kept in sync by hand; there is no reflection-based "pass everything through" path (deliberately — CACert wasn't in the original Phase 1 scope either, and was added to both places together when SilverBullet needed it, per `02-01-PLAN.md`'s decision log entry).

**How to avoid:** any new `config.Source` field this phase needs (recommended: `Username`) must be added in three places together: `kernel/config/types.go` (struct field), `kernel/pluginhost/host.go`'s `launch()` (add to the JSON map), and the plugin's own `main.go` `sourceConfig` struct (to decode it).

**Warning signs:** the plugin subprocess fails at startup with "username is empty" even though the TOML config clearly has `username = "..."` set — the value never left the kernel process.

### Pitfall 2: Proton Bridge LAN exposure requires host-server work outside this repo's code

**What goes wrong:** planning proceeds assuming the Bridge is already reachable from the desktop, and Task 1 of the plan immediately hits a connection refused/timeout with no code-level fix available.

**Why it happens:** `STATE.md`'s own Blockers section already flags this as unresolved ("Firewall/network access from the desktop to Proton Mail Bridge on the home server is not yet opened (bridge binds 127.0.0.1 by default)"). This research confirmed Bridge has **no supported configuration option** to bind a LAN interface — the only community-documented paths are (a) a `socat`/`stunnel` TCP forwarder run on the home server (recommended — no Bridge binary changes), or (b) recompiling Bridge from source with the host binding patched (not recommended — unofficial, must be redone every update).

**How to avoid:** the plan's first task should include a `checkpoint:human-verify` step for the user to set up the forwarder on the home server (out of this repo's code) before any live-Bridge integration work can proceed; the plugin code itself should be written and unit-testable against a fake/mock IMAP server (`go-imap`'s own `server`/`backend/memory` packages, used for local IMAP servers in tests) independent of live Bridge availability.

**Warning signs:** `Health()` reports "connection refused" or times out entirely regardless of code changes.

### Pitfall 3: bluemonday's own email-sanitization example explicitly says "not safe"

**What goes wrong:** copying bluemonday's official `cmd/sanitise_html_email/main.go` example verbatim (`p.AllowAttrs("style").Globally()`) reintroduces CSS-based attack surface the library's own comment calls out: *"There are not safe, and is only being done here to demonstrate how to process HTML emails where styling has to be preserved. This is at the expense of security."* `[VERIFIED: fetched directly from microcosm-cc/bluemonday's own repository in this session]`

**Why it happens:** email HTML relies heavily on inline `style=` attributes for basic formatting (colors, fonts) that bluemonday's default `UGCPolicy()` strips entirely — the temptation is to reach for the official example's blanket `AllowStyling()`/`AllowAttrs("style").Globally()` without registering its caveat.

**How to avoid:** rely on defense-in-depth already present in this codebase rather than the sanitizer alone — the existing rendition CSP (`kernel/httpapi/item.go`'s `renditionHandler`: `default-src 'none'; style-src 'unsafe-inline'; object-src 'none'; sandbox`) already blocks every network request the sanitized document could attempt to make (no `img-src`, no `url()` background fetches succeed under `default-src 'none'`), which closes off the most dangerous class of CSS-based exfiltration (tracking pixels, CSS-exfil via `background-image: url(...)`) *regardless* of what the sanitizer allows through. Given that existing mitigation, allowing `style` attributes more narrowly than the official example (specific safe properties like `color`/`font-weight`/`text-align` on specific elements, rather than `AllowAttrs("style").Globally()`) is a reasonable, documented residual-risk tradeoff — but it should be a deliberate, reviewed choice in the plan, not an unexamined copy-paste. This also has the pleasant side effect of defeating email tracking pixels for free, which is worth calling out as a stated benefit in the plan, not just a security footnote.

### Pitfall 4: Sender is not currently rendered anywhere in the UI despite flowing through the whole stack

**What goes wrong:** the plugin populates `Item.group_label` with the sender (per the proto's own comment: `group_id`/`group_label` = "chat thread / mail conversation"), the field round-trips correctly through `kernel/item/item.go`, `kernel/httpapi/stream.go`'s `toStreamItem`, and `web/src/lib/api.ts`'s `StreamItem` type — and success criterion 1 ("appear in the stream with sender, subject, and date") still isn't met, because neither `StreamRow.svelte` nor `DetailPane.svelte` currently renders `group_label` anywhere `[VERIFIED: grep across web/src — group_label appears only in TypeScript type declarations and test fixtures, never in a `.svelte` template]`.

**Why it happens:** `group_id`/`group_label` were added to the wire contract in Phase 1 specifically anticipating chat/mail sources, but no source that populates them meaningfully has shipped until now — the UI gap was invisible with paperless-ngx/SilverBullet (both leave these fields empty).

**How to avoid:** the plan must include an explicit frontend task to render `item.group_label` in both `StreamRow.svelte` (near the date/label metadata strip) and `DetailPane.svelte`'s header — this is a small, contained change, not a new component, but it is a required change this phase's own success criteria depend on.

**Warning signs:** UAT for success criterion 1 fails ("I don't see who sent this") even though the backend/API response visibly contains the correct sender string.

### Pitfall 5: Deep-link fidelity — don't assume `EXACT` is achievable without live verification

**What goes wrong:** the plugin declares `LINK_FIDELITY_EXACT` with a constructed `https://mail.proton.me/u/0/{folder}/{some-id}` URL, but the `{some-id}` segment is Proton's internal webmail message ID, which this research found **no confirmed mapping** from the IMAP-visible `Message-Id`/UID to `[CITED: cross-referenced across community sources — general email-tooling guidance explicitly warns against conflating IMAP UID with any other system's internal message ID]`. A wrong or malformed ID in the URL either 404s in the browser or opens the wrong message — worse than an honest `ANCHORED` link to the right folder.

**Why it happens:** the roadmap's own phase notes already flagged this exact item as unverified ("the Proton webmail deep-link format... was not verified during landscape research").

**How to avoid:** default to `LINK_FIDELITY_ANCHORED` with `deep_link` pointing at the matched folder/label's webmail view (`https://mail.proton.me/u/0/{folder-slug}` — folder slug format itself should be confirmed live, not assumed), and treat upgrading to `EXACT` as an optional stretch goal gated behind a `checkpoint:human-verify` task where the user confirms a working per-message URL pattern against their own real account. Do not block the phase's core success criteria on this — SRC-01's own success criteria don't actually require `EXACT` fidelity for email, only that email appears and renders (`ANCHORED` is a first-class, fully valid fidelity value per the contract's own three-value design).

### Pitfall 6: A message with no `Message-Id` header

**What goes wrong:** a small but real fraction of real-world email (spam, some automated systems, very old mail) lacks a `Message-Id` header entirely. If `Match()` uses an empty string as the dedup key, every such message collapses into a single item (each overwriting the last).

**How to avoid:** skip messages with an empty `Envelope.MessageId` explicitly (log a count, don't silently drop without any trace) rather than deduping on an empty string, mirroring the existing "reject just this item, not the whole sync" pattern `kernel/correlate.validateCorrelatedItem` already establishes for missing fidelity/deep_link.

## Code Examples

### Host-pinned `imap.Dialer` (mirrors `plugins/silverbullet/client.go`'s `allowHost`)

```go
// Source: adapted from plugins/silverbullet/client.go's DialContext pattern,
// using go-imap v1.2.1's Dialer interface (verified: Dial(network, addr string)
// (net.Conn, error) — same shape as net.Dialer's own method).
type pinnedDialer struct {
    allowedHost string // lowercased hostname of the configured Bridge forwarder
    inner       *net.Dialer
}

func (d *pinnedDialer) Dial(network, addr string) (net.Conn, error) {
    host, _, err := net.SplitHostPort(addr)
    if err != nil {
        host = addr
    }
    host = strings.ToLower(host)
    if host != d.allowedHost && host != "localhost" {
        if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
            return nil, fmt.Errorf("proton: foreign host refused: %q", host)
        }
    }
    return d.inner.Dial(network, addr)
}

// Usage:
dialer := &pinnedDialer{allowedHost: bridgeHost, inner: &net.Dialer{Timeout: 10 * time.Second}}
c, err := client.DialWithDialerTLS(dialer, bridgeAddr, tlsConfig)
if err != nil {
    return fmt.Errorf("proton: dial: %w", err)
}
c.Timeout = 15 * time.Second // command timeout — REQUIRED, see Health below
```

### `Health()` with an explicit timeout (never hang — SRC-01 success criterion 5)

```go
// Nothing in the kernel wraps a plugin's Health RPC call with its own
// timeout beyond the calling HTTP request's own context [VERIFIED: grep
// across kernel/httpapi/sources.go and kernel/pluginhost/host.go found no
// context.WithTimeout around the Health call path] — the plugin MUST
// enforce its own dial + command timeout, or a hung Bridge connection
// hangs the health probe indefinitely.
func (p *SourcePlugin) Health(ctx context.Context, _ *webspacesv1.HealthRequest) (*webspacesv1.HealthResponse, error) {
    dialer := &pinnedDialer{allowedHost: p.host, inner: &net.Dialer{Timeout: 5 * time.Second}}
    c, err := client.DialWithDialerTLS(dialer, p.addr, p.tlsConfig)
    if err != nil {
        return &webspacesv1.HealthResponse{Reachable: false, LastError: err.Error()}, nil
    }
    defer c.Logout()
    c.Timeout = 5 * time.Second
    if err := c.Login(p.username, p.password); err != nil {
        return &webspacesv1.HealthResponse{Reachable: false, LastError: fmt.Sprintf("login: %v", err)}, nil
    }
    return &webspacesv1.HealthResponse{Reachable: true, LastSyncUnix: time.Now().Unix()}, nil
}
```

### `Fetch()` body extraction: prefer HTML, fall back to plain text

```go
// Source: github.com/emersion/go-message/mail API (pkg.go.dev, verified this session).
mr, err := mail.CreateReader(bytes.NewReader(rawMessage))
if err != nil {
    return nil, fmt.Errorf("proton: parse message: %w", err)
}
var htmlPart, textPart []byte
for {
    p, err := mr.NextPart()
    if err == io.EOF {
        break
    }
    if err != nil {
        return nil, fmt.Errorf("proton: read message part: %w", err)
    }
    if h, ok := p.Header.(*mail.InlineHeader); ok {
        ct, _, _ := h.ContentType()
        b, _ := io.ReadAll(io.LimitReader(p.Body, maxBodyBytes)) // bound reads — see Security Domain
        switch ct {
        case "text/html":
            htmlPart = b
        case "text/plain":
            textPart = b
        }
    }
}

if len(htmlPart) > 0 {
    sanitized := emailSanitizePolicy.SanitizeBytes(htmlPart) // bluemonday, email-specific policy (Pitfall 3)
    doc := WrapDocument(sanitized) // same wrapping shape as plugins/silverbullet/render.go
    return &webspacesv1.FetchResponse{Available: true, MimeType: "text/html", Data: doc, Text: string(textPart)}, nil
}
return &webspacesv1.FetchResponse{Available: true, Text: string(textPart)}, nil // plain-text-only email: DetailPane's existing content.text branch already handles this
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Plaintext key/credential extraction assumptions for local mail tools | Proton Bridge issues its own per-install, locally-scoped IMAP credentials distinct from the user's actual Proton account password, stored via the OS-native cert/key files this research located | Ongoing Proton Bridge design | A compromised or leaked Bridge IMAP credential cannot be used to log into the user's actual Proton account directly — a meaningful scoping benefit worth noting in the plan's threat model, consistent with CLAUDE.md's existing "prefer a scoped credential" guidance |
| `go-imap/v2` | Still beta as of this research (confirmed CLAUDE.md's existing assessment remains current) | N/A | No change to the project's already-locked v1 decision |

**Deprecated/outdated:** none specific to this phase beyond what CLAUDE.md already documents.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | Bridge's exported cert's SAN is exactly `127.0.0.1` (not also including `localhost` or the LAN hostname) | Pattern 4 / TLS trust | If wrong, the `ServerName: "127.0.0.1"` override may need adjusting to whatever SAN the live cert actually presents — low risk, a five-minute fix once the real cert is inspected (`openssl x509 -noout -text`) during Task 1 |
| A2 | Standard `crypto/tls.Config.ServerName` behavior (checked against SAN independent of dial address) applies unmodified to `go-imap`'s TLS dial path | Pattern 4 | Low risk — this is documented Go standard-library behavior, not Proton- or go-imap-specific, but was not exercised against a live Bridge connection in this session |
| A3 | There is no confirmed mapping from IMAP `Message-Id`/UID to Proton's internal webmail message ID for constructing an `EXACT` deep link | Pitfall 5 / Deep-link fidelity | If wrong (a mapping does exist and can be confirmed live), the phase could ship `EXACT` fidelity instead of the recommended `ANCHORED` fallback — upside only, not a correctness risk either way since `ANCHORED` is the safe default |
| A4 | Proton Bridge has no supported LAN-bind configuration option, requiring an external forwarder | Pitfall 2 / Environment Availability | Medium — if a supported option exists that this research missed, the forwarder step could be skipped entirely; worth a final confirmation check against Proton's own official docs (not just community sources) before committing the plan's infrastructure task |
| A5 | The plugin subprocess's in-process Message-ID→mailbox cache is an acceptable MVP design (vs. some persistent alternative) | Critical Architecture Finding / Fetch-time mailbox lookup | Low-medium — if unacceptable, the alternative is persisting the mailbox name in the kernel index (a new column) or re-deriving it by re-running the folder/label scan on every Fetch (slower, but stateless) — a planner-level tradeoff to confirm, not a hard blocker |

## Open Questions

1. **Does Proton Bridge expose the SAME message under an "All Mail"-equivalent canonical mailbox, in addition to per-label virtual mailboxes?**
   - What we know: labels are non-destructive (message physically lives in one place, exposed under multiple `Labels/*` views).
   - What's unclear: whether there's a single canonical mailbox (like Gmail's "All Mail") that could simplify the Fetch-time mailbox-lookup problem, versus needing to track exactly which `Labels/*`/`Folders/*` mailbox was scanned.
   - Recommendation: confirm via `LIST` output during the Task 1 live spike; if such a mailbox exists, prefer it as the canonical Fetch target over the "first matched mailbox" cache design.

2. **What is Proton's actual webmail folder-slug URL format for the `ANCHORED` fallback deep link?**
   - What we know: community sources describe `https://mail.proton.me/u/0/{folder}/{id}` loosely, with folder names like `inbox`/`archive`.
   - What's unclear: the exact slug for a custom user-created folder/label (URL-encoded display name? an internal short ID?).
   - Recommendation: confirm live against the user's real account during Task 1; this is exactly the kind of item the phase notes' "short spike" already anticipated.

3. **Bridge cert rotation behavior** — does the cert at `~/.config/protonmail/bridge/cert.pem` ever change (e.g. on Bridge major-version upgrade), and if so, does the plugin need a graceful re-pin flow, or is a clear `Health()` error (cert no longer trusted → actionable message telling the user to re-export) sufficient for MVP?
   - Recommendation: a clear Health error pointing at the re-export step is sufficient for MVP — treat automatic re-pinning as out of scope, consistent with the project's "clear, actionable health error" success criterion rather than silent self-healing.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| Proton Mail Bridge reachable from the desktop over LAN | SRC-01 (the entire phase) | ✗ | — | **None — blocking.** Per `STATE.md`'s own Blockers section, this is not yet set up. Requires home-server-side work (a `socat`/`stunnel` forwarder) outside this repo's code, plus opening a firewall port. This must be a `checkpoint:human-verify` task early in the plan, before any live-Bridge integration task can be verified. |
| Bridge's exported TLS cert (`ca_cert`-style pinning file) | Pattern 4 (TLS trust) | ✗ | — | None — must be exported from the Bridge GUI ("Export TLS certificates") on the home server and placed somewhere the desktop kernel process can read it, mirroring the existing SilverBullet `ca_cert` config convention |
| `modernc.org/sqlite` FTS5 support | KERN-05 | ✓ | v1.54.0 (already in go.mod) | — (verified locally in this session — no fallback needed) |
| `github.com/emersion/go-imap` v1.2.1 | SRC-01 | ✓ (resolvable via `go get`) | v1.2.1 | — |
| `github.com/emersion/go-message` v0.18.2 | SRC-01 (Fetch-time MIME parsing) | ✓ (resolvable via `go get`) | v0.18.2 | — |

**Missing dependencies with no fallback:**
- Proton Mail Bridge LAN reachability — blocking; requires a home-server infrastructure task before live integration can be verified. Code-level work (plugin implementation, unit tests against `go-imap`'s own mock server package) can and should proceed independently, but end-to-end verification is gated on this.
- Bridge's exported TLS certificate file — blocking for the same reason; a manual export step on the home server.

**Missing dependencies with fallback:**
- None applicable beyond the above — this phase's core dependency (Bridge reachability) has no code-level fallback by its nature.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | Partial | The plugin authenticates to Bridge via IMAP `LOGIN` using a Bridge-scoped local password (never the user's real Proton account password) — treat this credential with the same "never log it" discipline `plugins/silverbullet/client.go` already documents for its bearer token |
| V3 Session Management | No | No web sessions introduced by this phase; existing kernel HTTP surface is unchanged in this respect |
| V4 Access Control | Yes | Outbound host allowlist (Pattern: host-pinned `imap.Dialer`) restricts the plugin's own TCP connections to the configured Bridge host + loopback only — same property `plugins/silverbullet` and `plugins/paperless` already enforce for their HTTP clients, extended to IMAP's transport |
| V5 Input Validation | Yes | Untrusted HTML email body sanitization (bluemonday, Pitfall 3) is the primary V5 control this phase adds; also: bound MIME part count/size read from a single message (a maliciously crafted email with thousands of MIME parts or an enormous attachment could otherwise cause unbounded memory use during `Fetch` — use `io.LimitReader` per part, matching the existing `sdk.MaxMessageSize` gRPC ceiling convention already established for renditions) |
| V6 Cryptography | Yes | TLS certificate pinning must use `ServerName` override + `RootCAs` pool (Pattern 4) — never blanket `InsecureSkipVerify: true` with no compensating check; the IMAP credential (password) is never logged, following `plugins/silverbullet/client.go`'s explicit "never log the request object or its headers" precedent |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Email tracking pixels / remote-resource exfiltration via sanitized HTML rendered in-app | Information Disclosure | Already mitigated for free by the existing rendition CSP (`default-src 'none'`) — no new code needed, but worth an explicit test asserting a `<img src="https://attacker.example/pixel.gif">` in a fetched email does not trigger a network request from the sandboxed iframe |
| CSS-based UI redress / exfiltration via an over-permissive `style` attribute allowlist | Tampering / Information Disclosure | Bounded `AllowAttrs("style")` scope (specific safe properties on specific elements) rather than bluemonday's own demo's `Globally()` call, per Pitfall 3 — plus the CSP's `default-src 'none'` as a second layer |
| MITM against the Bridge↔desktop TLS connection if certificate pinning is done incorrectly (e.g. blanket `InsecureSkipVerify`) | Spoofing / Tampering | `ServerName` override + `RootCAs` pinning (Pattern 4) — connection remains fully verified against the pinned cert, just not via hostname-matches-dial-address |
| A crafted/malformed email (malformed MIME, huge part count, missing Message-Id) causing a panic, hang, or unbounded memory use during `Match`/`Fetch` | Denial of Service | `io.LimitReader` per part during body extraction; explicit nil/empty checks on `Envelope.MessageId` (Pitfall 6) rather than assuming well-formed input; the plugin's own per-command `client.Timeout` bounds any single IMAP round-trip |
| IMAP mutating commands (`STORE`/`EXPUNGE`/`MOVE`) accidentally called, violating PLUG-02's read-only guarantee | Tampering | Mechanical test (Don't Hand-Roll table) forbidding these identifiers anywhere under the plugin package, mirroring the existing `net/http`-verb AST scan's enforcement model |

## Sources

### Primary (HIGH confidence)
- `github.com/emersion/go-imap` v1.2.1 source, fetched and read directly in this session: `client/client.go` (Dial/DialWithDialerTLS/Dialer interface, Timeout field), `client/cmd_auth.go` (Select/EXAMINE), `message.go` (BodySectionName.Peek, Envelope.MessageId) — github.com/emersion/go-imap
- `github.com/microcosm-cc/bluemonday`'s own official HTML-email sanitization example, fetched directly: `cmd/sanitise_html_email/main.go` — github.com/microcosm-cc/bluemonday
- `sqlite.org/fts5.html` (official SQLite docs) — external content table syntax, trigger patterns, `bm25()`/`snippet()`/`highlight()` signatures
- This repository's own code, read directly: `kernel/config/{types,config}.go`, `kernel/pluginhost/host.go`, `kernel/index/{schema,store}.go`, `kernel/httpapi/{item,stream,routes,webspaces}.go`, `kernel/correlate/correlate.go`, `plugins/silverbullet/{client,plugin,render}.go`, `plugins/paperless/{main,readonly_test,outbound_hosts_test}.go`, `web/src/lib/{api.ts, components/{DetailPane,StreamRow,WebspaceHeader}.svelte}`, `proto/webspaces/v1/plugin.proto`, `docs/plugin-contract.md`
- Local verification runs in this session: `go get`/`go list -m -json` for `github.com/emersion/go-imap@v1.2.1`, `github.com/emersion/go-message@v0.18.2`, `github.com/microcosm-cc/bluemonday@v1.0.27` against the live Go module proxy; two throwaway Go programs run against the project's real `modernc.org/sqlite v1.54.0` dependency proving FTS5 availability and the full external-content-table + trigger + `bm25`/`snippet` pipeline end to end, including the `TEXT PRIMARY KEY` + `content_rowid='rowid'` interaction specific to this repo's schema

### Secondary (MEDIUM confidence)
- pkg.go.dev for `github.com/emersion/go-message/mail` (Reader/Part/NextPart API shape) — cross-checked against the package's own doc comments
- Proton's own support docs (`proton.me/support/comprehensive-guide-to-bridge-settings`, `proton.me/support/apple-mail-certificate`) — cert export flow, self-signed cert rationale
- Community sources on Bridge's LAN-exposure limitation and cert file location, cross-referenced across two to three independent sources each: `vimoire.com` (socat forwarder walkthrough), `ndo.dev` (headless Bridge / source-patch approach, used only to confirm the "no native LAN bind" finding, not recommended as the approach), ProtonMail community/GitHub PR #270 discussion, a community note on the `~/.config/protonmail/bridge/{cert,key}.pem` file location

### Tertiary (LOW confidence)
- Community forum answer describing a `mail.proton.me/u/0/{folder}/{id}` webmail URL pattern — explicitly flagged unverified throughout this document (Pitfall 5, Open Question 2); do not treat as authoritative without live confirmation

## Metadata

**Confidence breakdown:**
- Standard stack (go-imap/go-message/bluemonday choice and versions): HIGH — verified directly against pinned library source and the live module registry, not training-data recall
- FTS5 search architecture: HIGH — verified end-to-end against this repo's actual SQLite dependency in this session, not just cited from docs
- Proton Mail Bridge LAN exposure and TLS handling: MEDIUM — no official Proton documentation confirms the LAN-bind limitation or the forwarder approach; community-sourced and internally consistent across multiple independent sources, but genuinely unverifiable without live access to the user's home-server Bridge instance
- Proton webmail deep-link format: LOW — explicitly flagged as an open question requiring a live spike, consistent with the roadmap's own framing of this phase's research risk
- Kernel-side plumbing changes required (config.Source, pluginhost/host.go, UI sender rendering): HIGH — identified by direct code reading in this repository, not inference

**Research date:** 2026-07-29
**Valid until:** ~30 days for the Go-library findings (stable ecosystem); the Proton Bridge findings should be treated as needing re-confirmation at the start of Task 1 regardless of elapsed time, since they were never verified against a live instance in this session
