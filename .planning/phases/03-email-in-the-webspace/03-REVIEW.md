---
phase: 03-email-in-the-webspace
reviewed: 2026-07-31T00:00:00Z
depth: standard
files_reviewed: 34
files_reviewed_list:
  - docs/api.md
  - kernel/config/types.go
  - kernel/httpapi/routes.go
  - kernel/httpapi/search.go
  - kernel/httpapi/search_test.go
  - kernel/index/schema.go
  - kernel/index/store.go
  - kernel/index/store_test.go
  - kernel/pluginhost/host.go
  - plugins/proton/body.go
  - plugins/proton/client.go
  - plugins/proton/go.mod
  - plugins/proton/go.sum
  - plugins/proton/imap_transcript_test.go
  - plugins/proton/item_test.go
  - plugins/proton/live_bridge_test.go
  - plugins/proton/mailbox_cache_test.go
  - plugins/proton/main.go
  - plugins/proton/outbound_hosts_test.go
  - plugins/proton/plugin.go
  - plugins/proton/readonly_test.go
  - plugins/proton/render_test.go
  - web/src/lib/api.ts
  - web/src/lib/components/date-format.test.ts
  - web/src/lib/components/DetailPane.svelte
  - web/src/lib/components/SearchBox.svelte
  - web/src/lib/components/SearchResults.svelte
  - web/src/lib/components/StreamRow.svelte
  - web/src/lib/components/ui/input/index.ts
  - web/src/lib/components/ui/input/input.svelte
  - web/src/lib/components/WebspaceHeader.svelte
  - web/src/lib/format.test.ts
  - web/src/lib/format.ts
  - web/src/lib/node-builtins.d.ts
  - web/src/routes/w/[webspace]/+page.svelte
findings:
  critical: 1
  warning: 5
  info: 4
  total: 10
status: issues_found
---

# Phase 03: Code Review Report

**Reviewed:** 2026-07-31T00:00:00Z
**Depth:** standard
**Files Reviewed:** 34
**Status:** issues_found

## Summary

This is a re-review of the current state of the phase 3 files, superseding
the prior 03-REVIEW.md round. That round's CR-01 (the Proton plugin's
`mailboxCache` being replaced instead of merged per `Match` call, breaking
`Fetch` for every webspace but the last one synced in a cycle) is
genuinely fixed — `mergeMailboxCache` now accumulates, never wholesale
replaces, and `plugins/proton/mailbox_cache_test.go` directly regression-
tests the two-webspace and zero-match cases that broke before. Its WR-01
(`DetailPane.svelte` reimplementing date formatting without the UTC pin)
is also fixed — `DetailPane.svelte` now imports and uses the shared
`formatItemDate` from `format.ts`.

The kernel-side FTS5 search (`kernel/index`, `kernel/httpapi/search.go`)
and the search UI (`SearchBox.svelte`, `SearchResults.svelte`,
`format.ts`'s `parseSnippet`/`searchVariant`) remain careful and well
tested: parameterized SQL throughout, correct bm25 ordering, and
deliberate degrade-to-empty-results behavior for malformed FTS5 input.
The Proton plugin's read-only discipline (PLUG-02, enforced by an AST
scan) and TLS/host-pinning are solid.

This round's one BLOCKER is a supply-chain issue specific to what this
phase newly does: `plugins/proton` is the first plugin in this codebase
to run a third-party HTML parser over fully attacker-controlled input
(an arbitrary sender's email HTML body), and its pinned
`golang.org/x/net` predates the fix for a known DoS CVE in exactly that
parser. The remaining findings are WARNING/INFO-level robustness,
data-integrity, and quality gaps — several of them carried over,
unaddressed, from the prior review round (see IN-02 through IN-04).

## Critical Issues

### CR-01: Outdated `golang.org/x/net` (CVE-2024-45338 / GO-2024-3333) parses attacker-controlled email HTML

**File:** `plugins/proton/go.mod:14`, `plugins/proton/go.sum:23-24`

**Issue:** `plugins/proton/go.mod` pins `golang.org/x/net v0.26.0`
(indirect, pulled in transitively by `github.com/microcosm-cc/bluemonday`,
which uses `golang.org/x/net/html` as its HTML tokenizer). Every
`golang.org/x/net` release before v0.33.0 is affected by CVE-2024-45338
(GO-2024-3333): a crafted HTML document can be tokenized non-linearly
with respect to its length by `golang.org/x/net/html`, causing excessive
CPU/memory consumption — a denial-of-service vector.

This matters specifically because of what this plugin does with that
parser: `body.go`'s `RenderSanitizedEmail` (called from `plugin.go`'s
`fetchFull`) runs `bluemonday.SanitizeBytes` — and therefore
`golang.org/x/net/html`'s tokenizer — directly over the HTML body of an
**arbitrary inbound email**. Any sender who can deliver mail to the
monitored Proton account can trigger this code path with fully
attacker-chosen bytes; nothing upstream filters or pre-validates the
HTML before it reaches the sanitizer. This is exactly the attack surface
CVE-2024-45338 describes, and it is genuinely new in this phase — no
other plugin in the reviewed tree parses third-party-controlled HTML
through this dependency at this version.

**Fix:**
```
# plugins/proton/go.mod
require golang.org/x/net v0.33.0 // or later — fixes GO-2024-3333/CVE-2024-45338
```
Bump the `require` line and regenerate `go.sum` for this module. Check
`plugins/silverbullet` (which also renders third-party-sourced content
via goldmark/bluemonday) for the same stale pin while addressing this.

## Warnings

### WR-01: `context.Context` is accepted but never honored anywhere in the IMAP call path

**File:** `plugins/proton/plugin.go:134` (`Match`), `plugins/proton/plugin.go:436,453` (`Fetch`/`fetchFull`), `plugins/proton/client.go:196` (`connect`)

**Issue:** `Match`, `Fetch`, and `fetchFull` all take a `ctx
context.Context` parameter — contrast with `Health`
(`plugins/proton/plugin.go:557`), which explicitly names its unused
context parameter `_`, suggesting the omission elsewhere is an oversight
rather than a deliberate "we never honor cancellation" style choice.
`ctx` is never read inside `Match`'s or `fetchFull`'s bodies (confirmed:
the only appearance of `ctx` in either function is in its signature),
and `Client.connect`/`realDial` don't accept a context at all — only a
fixed `time.Duration`. If a caller cancels the request (client
disconnect, a future request-scoped timeout upstream), a slow or hung
Bridge connection keeps the goroutine and the gRPC call alive for the
full `syncDialTimeout`/`healthDialTimeout` regardless.

**Fix:** Either thread `ctx` through to the dial/IMAP operations (e.g.
`net.Dialer.DialContext`, and select on `ctx.Done()` around the blocking
`conn.Fetch`/`conn.UidFetch`/`conn.UidSearch` calls), or rename the
unused parameters to `_` throughout so the lack of cancellation support
is explicit and consistent with `Health`.

### WR-02: Email Subject text is indexed without stripping the search snippet's reserved delimiter characters

**File:** `plugins/proton/plugin.go:352` (`toItem`, `title := m.envelope.Subject`)

**Issue:** `kernel/index/store.go`'s `SnippetOpen`/`SnippetClose` (STX
`\x02`/ETX `\x03`) scheme, and `web/src/lib/format.ts`'s `parseSnippet`,
both rely on the documented invariant that "no real subject line or
preview text can contain them." That was plausible for the pre-existing
sources (paperless OCR text, SilverBullet markdown), but this phase adds
the first source whose `Title` is fully attacker-influenced:
`m.envelope.Subject` is stored verbatim as `Item.Title` with no
control-character stripping, and any sender can put arbitrary bytes —
including literal STX/ETX — into a `Subject:` header (via RFC 2047
encoded-words, or a non-compliant MTA passing raw 8-bit header bytes
through). A crafted Subject could inject fake delimiters into the
FTS5-indexed title, corrupting `snippet()`'s output for that row.
Impact is bounded today only because `parseSnippet` degrades a malformed
delimiter sequence to one stripped, unmatched segment rather than
throwing (and Svelte's text binding prevents any HTML injection) — but
the documented safety invariant is violated by this plugin's own data.

**Fix:** Strip or replace ASCII control characters (at minimum
`\x02`/`\x03`, arguably the full C0 range) from `Title` (and `Preview`,
`GroupLabel`) before returning the item from `toItem`.

### WR-03: `fetchFull` trusts `IMAP SEARCH HEADER`'s substring match as if it were exact

**File:** `plugins/proton/plugin.go:474-484`

**Issue:** `UID SEARCH HEADER Message-Id "<msgID>"` is, per RFC 3501, a
**substring** match against the header's text, not an exact match. The
code takes `uids := conn.UidSearch(criteria); uid := uids[0]` without
verifying that the resolved message's actual `Message-Id` header equals
`<msgID>` exactly. In a mailbox containing a message whose `Message-Id`
value happens to contain another message's `<msgID>` as a literal
substring (a crafted/adversarial or unusually-generated Message-Id),
`fetchFull` could resolve and serve the **wrong message's body** under
the originally-requested item's identity, since nothing downstream
cross-checks the fetched message against the requested `sourceID`.

**Fix:** After `UidFetch`, also fetch `imap.FetchEnvelope` (or
`BODY.PEEK[HEADER.FIELDS (MESSAGE-ID)]`) for the resolved UID and assert
`normalizeMessageID(...) == msgID` before returning the body; treat a
mismatch as `codes.NotFound` rather than trusting the search result's
identity.

### WR-04: Clearing the search box does not advance `searchRequestSeq`, so a stale in-flight search can still land

**File:** `web/src/routes/w/[webspace]/+page.svelte:144-168` (`handleSearch`)

**Issue:** The non-empty-query branch guards against out-of-order
responses via `const seq = ++searchRequestSeq; ... if (seq !== searchRequestSeq) return;`.
The empty/whitespace-query branch (lines 150-154, the "user cleared the
box" path) resets `searchState`/`searchResults` directly but never
touches `searchRequestSeq`. If an earlier in-flight request resolves
*after* the user clears the box, its `seq` still equals the unchanged
`searchRequestSeq`, so the stale-response guard passes and
`searchResults`/`searchState` get silently overwritten with data the
user already dismissed — violating the comment's own stated invariant
("a slower earlier request can never overwrite a faster later one").
Today this has no visible effect only because `searchVariant` gates
rendering on `query.trim() === ''` first, so `SearchResults` still
renders nothing — but `searchResults` state is left inconsistent with
`searchQuery`, and `selectedItem`'s fallback lookup in the same file
reads `searchResults` directly.

**Fix:** Increment `searchRequestSeq` in the clear branch too (or gate
every `.then`/`catch` continuation on `query.trim() === ''` at
resolution time as well as at dispatch time).

### WR-05: A cold `mailboxCache` (post-restart) is indistinguishable from a genuinely deleted message

**File:** `plugins/proton/plugin.go:459-462`

**Issue:** `fetchFull` returns `codes.NotFound` both when a message has
genuinely been deleted/moved and when the plugin process simply hasn't
run `Match` yet since it (re)started — this is documented as an
intentional shared code path. The residual gap: immediately after a
kernel restart, **every** previously-indexed Proton item briefly
surfaces to the user as if broken (the detail pane shows "Couldn't load
this item... try again", or "unreachable"), and clicking Retry does not
help until the next scheduled/manual sync repopulates the cache. This is
a real, if transient, UX regression specific to the email source — other
plugins resolve content statelessly per-request, so they don't have this
restart window.

**Fix:** Consider persisting the mailbox resolution across restarts, or
falling back to a `UID SEARCH HEADER Message-Id` scan across previously-
known mailbox names when the cache misses, so a cold cache degrades to
"slightly slower" rather than "reported as gone." At minimum, surface
copy that distinguishes "not yet synced since restart" from "no longer
available," since Retry is not actually the correct recovery action for
the former.

## Info

### IN-01: `Match`'s IMAP FETCH requests `imap.FetchUid` but the result is discarded

**File:** `plugins/proton/plugin.go:190-214`

**Issue:** The FETCH item list includes `imap.FetchUid`, but the loop
below only reads `msg.Envelope` and `msg.InternalDate` into `matched` —
`msg.Uid` is never stored or used anywhere. This is consistent with the
(correct) design decision that `fetchFull` must re-resolve the UID
independently via `UID SEARCH` rather than trust a cached one, but it
leaves a fetched protocol field with no purpose, reading as leftover
from an earlier design.

**Fix:** Drop `imap.FetchUid` from the `items` list in `Match`, or add a
one-line comment noting it's intentionally unused if there's a reason to
keep requesting it.

### IN-02: `Snippet` helper in `body.go` is unused (carried over from the prior review round)

**File:** `plugins/proton/body.go:29,112-121`

**Issue:** `Snippet` (rune-count-capped truncation) and its backing
`previewRuneCap` constant are fully implemented and documented ("mirrors
plugins/silverbullet's Snippet helper") but nothing in `plugins/proton`
(production code or tests) calls it — `toItem` hardcodes `Preview: ""`
and `fetchFull` returns the full extracted `text` unmodified. Confirmed
via `grep -rn "Snippet(" plugins/proton/*.go`: only the definition site
matches. This was already flagged in the prior review round and remains
unaddressed.

**Fix:** Remove it until there's an actual call site, or wire it in if a
future change starts populating `Preview` or truncating `Fetch`'s `Text`.

### IN-03: Dead sentinel errors with misleading doc comments in the Proton plugin (carried over from the prior review round)

**File:** `plugins/proton/client.go:22-28, 39-44`

**Issue:** `ErrNotFound` and `ErrNoMessageID` are exported and each
documented as "returned when..." a specific condition occurs, but
neither is ever returned or wrapped anywhere in `plugins/proton/*.go`
(confirmed via grep: zero non-declaration references to either). The
actual no-Message-Id skip path uses a plain counter
(`skippedNoMessageID++`) plus a log line instead of `ErrNoMessageID`;
`fetchFull`'s not-found paths use `status.Errorf(codes.NotFound, ...)`
directly instead of `ErrNotFound`. The doc comments describe behavior
that does not exist in the code, which will mislead a future reader.
This was already flagged in the prior review round and remains
unaddressed.

**Fix:** Either wire these sentinels into the paths their doc comments
describe, or remove them if the counter/log-line and direct
`status.Errorf` approach is the intended, final design.

### IN-04: `getJSON`/`postJSON` in `api.ts` are near-duplicate implementations (carried over from the prior review round)

**File:** `web/src/lib/api.ts:121-137, 145-161`

**Issue:** `getJSON` and `postJSON` are identical except for the `fetch`
call's `init` argument — the entire error-envelope-parsing block (lines
124-134 and 148-158) is duplicated verbatim, including the comment
("mirrors getJSON's error-envelope handling exactly") that itself
documents the duplication rather than eliminating it. A future change to
the error-handling logic (e.g. a new failure shape) has to be applied in
two places in lockstep. This was already flagged in the prior review
round and remains unaddressed.

**Fix:** Factor the shared logic into one helper:
```ts
async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const res = await fetch(path, init);
	if (!res.ok) {
		let envelope: ApiErrorEnvelope | undefined;
		try {
			envelope = (await res.json()) as ApiErrorEnvelope;
		} catch {
			/* not the error envelope */
		}
		if (envelope?.error) throw new ApiError(envelope.error.code, envelope.error.message);
		throw new ApiError('http_error', `Request to ${path} failed with status ${res.status}`);
	}
	return (await res.json()) as T;
}
const getJSON = <T>(path: string) => request<T>(path);
const postJSON = <T>(path: string) => request<T>(path, { method: 'POST' });
```

---

_Reviewed: 2026-07-31T00:00:00Z_
_Reviewer: Claude (gsd-code-reviewer)_
_Depth: standard_
