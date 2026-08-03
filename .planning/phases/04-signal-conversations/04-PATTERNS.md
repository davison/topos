# Phase 4: Signal Conversations - Pattern Map

**Mapped:** 2026-08-03
**Files analyzed:** 17 (plugins/signal/* new package) + 2 kernel modifications
**Analogs found:** 17 / 17 (all via `plugins/proton`, the freshest full plugin exemplar; no frontend files needed — reuses `DetailPane`'s existing `html` variant unchanged)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `plugins/signal/go.mod` | config | — | `plugins/proton/go.mod` | exact (structure), note: signal's must be cgo-enabled + own `go.work` member |
| `plugins/signal/main.go` | controller (subprocess entrypoint) | request-response | `plugins/proton/main.go` | exact — but `sourceConfig` shape differs (local-path source, no `base_url`/`token`) |
| `plugins/signal/plugin.go` | controller (SourcePlugin RPC impl) | CRUD (Match=read+build items, Fetch=read) | `plugins/proton/plugin.go` | exact — role and RPC shape identical; data source is SQLCipher DB, not IMAP |
| `plugins/signal/keyresolve.go` | service | transform | *(no analog — new problem class)* | none — see "No Analog Found" |
| `plugins/signal/safestorage_linux.go` | utility | transform (crypto) | *(no analog)* | none |
| `plugins/signal/secretservice.go` | service | request-response (D-Bus) | *(no analog)* | none |
| `plugins/signal/dsn.go` | utility | file-I/O | *(no analog — closest is `plugins/proton/client.go`'s `allowHost`/connect pattern)* | partial |
| `plugins/signal/schemaguard.go` | utility | transform | *(no analog — new problem class, but "fail loud with named version" pattern mirrors error-handling style seen throughout `plugins/proton`)* | partial |
| `plugins/signal/digest.go` | service | batch/transform (CRUD grouping) | `plugins/proton/plugin.go` (`Match`'s merge-by-Message-ID loop) | role-match — same "scan rows, merge by stable key, build Items" shape |
| `plugins/signal/match.go` | service | CRUD (keyword match) | `plugins/proton/plugin.go` (`matchesAnyKeyword`, `leafName`, mailbox-matching loop) | exact — identical exact/case-insensitive keyword contract (D-05/D-06 layered on top) |
| `plugins/signal/render.go` | utility (HTML render+sanitize) | transform | `plugins/proton/body.go` | exact — sanitize+wrap pipeline, same bluemonday policy shape |
| `plugins/signal/deeplink.go` | utility | transform | `plugins/proton/deeplink.go` | role-match — same shape (build a URL from a stable identifier), but `sgnl://` scheme replaces Proton's HTTPS webmail-search URL |
| `plugins/signal/readonly_test.go` | test | — | `plugins/proton/readonly_test.go` | exact — AST-scan pattern reusable near-verbatim, swap IMAP-mutating idents for SQL-mutating idents |
| `plugins/signal/byte_identical_test.go` | test | — | *(no analog — new test class, strongest read-only guarantee in repo)* | none |
| `plugins/signal/outbound_hosts_test.go` | test | — | `plugins/proton/outbound_hosts_test.go` | role-match — same shape, but assertion is "zero hosts" not "one allowed host" |
| `plugins/signal/schema_version_fixture_test.go` | test | — | `plugins/proton/readonly_test.go`'s negative-control pattern (fixture proving the scanner isn't vacuous) | partial |
| `kernel/config/config.go` (`Validate`) | config/service | CRUD (validation) | itself (existing file, modify in place) | exact — extend, don't replace |
| `kernel/config/types.go` (`Source`) | model | — | itself (existing file, modify in place) | exact — extend, don't replace |

## Pattern Assignments

### `plugins/signal/main.go` (controller, request-response)

**Analog:** `plugins/proton/main.go`

**Imports pattern** (lines 1-16):
```go
package main

import (
	"encoding/json"
	"fmt"
	"os"

	goplugin "github.com/hashicorp/go-plugin"

	"github.com/davison/webspaces/sdk"
)
```

**Config-decode + fail-loud-on-missing-required-field pattern** (lines 27-58):
```go
type sourceConfig struct {
	BaseURL        string `json:"base_url"`
	Username       string `json:"username"`
	Token          string `json:"token"`
	CACert         string `json:"ca_cert"`
	WebmailBaseURL string `json:"webmail_base_url"`
}

func main() {
	raw := os.Getenv("WEBSPACES_SOURCE_CONFIG")
	if raw == "" {
		fatal(fmt.Errorf("WEBSPACES_SOURCE_CONFIG is not set"))
	}
	var cfg sourceConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		fatal(fmt.Errorf("parse WEBSPACES_SOURCE_CONFIG: %w", err))
	}
	if cfg.BaseURL == "" {
		fatal(fmt.Errorf("WEBSPACES_SOURCE_CONFIG: base_url is empty"))
	}
	// ... one guard per required field ...
}
```
**Signal deviation:** `sourceConfig` has no `base_url`/`token` (local-path source per CONTEXT.md discretion item) — instead needs a field for the Signal Desktop config dir path (default `~/.config/Signal`, overridable for tests) and possibly nothing else, since the key is resolved entirely at runtime from that directory. This is the concrete trigger for the `kernel/config.Validate` relaxation (see below).

**Serve wiring** (lines 64-73) — copy verbatim, unchanged:
```go
goplugin.Serve(&goplugin.ServeConfig{
	HandshakeConfig: sdk.Handshake,
	Plugins: map[string]goplugin.Plugin{
		"source": &sdk.SourcePluginGRPCPlugin{Impl: impl},
	},
	GRPCServer: sdk.GRPCServer,
})
```

---

### `plugins/signal/plugin.go` (controller, CRUD)

**Analog:** `plugins/proton/plugin.go`

**Struct + constructor shape** (lines 56-116): mirror `SourcePlugin` holding a `logOut io.Writer` (os.Stderr default, overridable in tests) and any long-lived resolved state (e.g. cached SQLCipher key, resolved config-dir path). Signal's plugin does NOT need `proton`'s `mailboxCache` accumulate-across-Match-calls pattern in the same shape, because Match re-derives everything from the DB fresh each call — but the doc-comment discipline of explaining *why* state is/isn't cached across calls should be copied.

**Describe** (lines 118-124) — copy verbatim shape:
```go
func (p *SourcePlugin) Describe(_ context.Context, _ *webspacesv1.DescribeRequest) (*webspacesv1.DescribeResponse, error) {
	return &webspacesv1.DescribeResponse{
		SourceType:      sourceType,
		DisplayName:     displayName,
		ContractVersion: contractVersion,
	}, nil
}
```

**Match pattern** (lines 133-237): "list candidates, filter by exact/case-insensitive keyword match, scan rows, merge by stable key, build Items, log a count-only line (never content) for skipped rows" — copy this control-flow shape directly:
```go
func (p *SourcePlugin) Match(ctx context.Context, req *webspacesv1.MatchRequest) (*webspacesv1.MatchResponse, error) {
	keywords := req.GetKeywords()
	if len(keywords) == 0 {
		return &webspacesv1.MatchResponse{}, nil
	}
	// open db read-only, guardSchemaVersion, resolve matching conversations (match.go)
	// group matched conversations' messages into day-digests (digest.go)
	// zero matches -> successful EMPTY response, never an error (Pitfall 2 precedent)
	return &webspacesv1.MatchResponse{Items: items}, nil
}
```

**toItem pattern** (lines 357-398) — build `webspacesv1.Item` with `GroupId` = conversation id (per CONTEXT.md canonical_refs: "Item already carries group_id... designed for exactly this"), `Fidelity: LINK_FIDELITY_CONVERSATION_ONLY` (not `ANCHORED` like Proton — CONTEXT.md fixes this at conversation-only), `DeepLink` from `deeplink.go`, `Preview` = tail snippet (D-02), `Provenance` map with `source_type`/`source_system`/`source_id`/`plugin`/`contract_version` keys — copy this map shape verbatim.

**Fetch pattern** (lines 429-565): `switch req.GetVariant()` dispatch, THUMBNAIL always unavailable (mirrors Signal digests having no image rendition either, same `noThumbnailReason` idiom), FULL/PREVIEW share one path that re-opens the DB read-only and re-fetches full content live (never cached from Match) — copy this "producer decides readability, sanitize+wrap only at Fetch time" discipline directly.

**Health pattern** (lines 572-583) — copy shape; Signal's equivalent "reachability" check is "can we resolve the key and open the DB read-only", not a network dial:
```go
func (p *SourcePlugin) Health(ctx context.Context, _ *webspacesv1.HealthRequest) (*webspacesv1.HealthResponse, error) {
	// attempt key resolution + read-only open + guardSchemaVersion
	// any failure -> Reachable:false, LastError naming the specific failing step, never a gRPC error
}
```

---

### `plugins/signal/match.go` (service, CRUD)

**Analog:** `plugins/proton/plugin.go` (`matchesAnyKeyword`, `leafName`, the mailbox-scan-and-filter loop, lines 145-284)

```go
func matchesAnyKeyword(leaf string, keywords []string) bool {
	for _, kw := range keywords {
		if strings.EqualFold(leaf, kw) {
			return true
		}
	}
	return false
}
```
Reuse this exact exact/case-insensitive predicate. Layer D-06's rule on top: for 1:1 chats, match candidate names are the user's own nickname + address-book/system contact name for that conversation — never the contact's self-chosen profile name field. Comment this rationale in the code exactly as CONTEXT.md's `<specifics>` section states it (a contact must not self-inject via profile rename).

---

### `plugins/signal/digest.go` (service, batch/transform)

**Analog:** `plugins/proton/plugin.go`'s merge-by-normalized-Message-ID loop (lines 172-226) — same "scan rows into a map keyed by a stable identity, accumulate, then build Items from the map" shape, applied to (conversationID, localDay) instead of Message-ID.

**Stable source_id construction** (already given in RESEARCH.md Code Examples, reuse verbatim):
```go
func sourceIDForDigest(conversationID string, localDay time.Time) string {
	return fmt.Sprintf("%s:%s", conversationID, localDay.Format("2006-01-02"))
}
```
This is the load-bearing D-04/D-07 "today's digest updates in place" mechanism — `kernel/index.ReplaceWebspaceSourceItems`'s existing upsert-by-source_id semantics (no kernel change needed) depend on this being deterministic across syncs, exactly as `plugins/proton`'s `encodeSourceID` is deterministic across syncs.

---

### `plugins/signal/render.go` (utility, transform)

**Analog:** `plugins/proton/body.go` (the entire sanitize+wrap pipeline)

**Sanitize policy pattern** (lines 165-179): build a `*bluemonday.Policy` once at package init, based on `bluemonday.UGCPolicy()`, narrowly widened for presentational styling only:
```go
var emailSanitizePolicy = newEmailSanitizePolicy()

func newEmailSanitizePolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("style").OnElements(styledElements...)
	p.AllowStyles(/* presentational-only allowlist */).OnElements(styledElements...)
	return p
}
```
Signal's transcript renderer should build its own equivalent `signalTranscriptSanitizePolicy` this way — do not reuse Proton's global var directly (different content shape: message bubbles/sender-grouped, not an email body), but copy the construction pattern and the CSP-defeats-tracking-pixel rationale comment verbatim where applicable (`img { display: none !important; }` equivalent still applies if Signal messages can carry inline images/attachments referenced by URL).

**WrapDocument pattern** (lines 261-276) — copy near-verbatim, same dark-theme `themeStyle` constant, same "sanitize THEN wrap, never the reverse" ordering discipline:
```go
func WrapDocument(sanitizedFragment []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString("<!doctype html>\n<html><head><meta charset=\"utf-8\"><style>")
	buf.WriteString(themeStyle)
	buf.WriteString("</style></head><body>")
	buf.Write(sanitizedFragment)
	buf.WriteString("</body></html>")
	return buf.Bytes()
}
```

**Snippet/rune-cap pattern** (lines 125-134) — reuse verbatim for the D-02 tail-snippet truncation:
```go
func Snippet(s string) string {
	if utf8.RuneCountInString(s) <= previewRuneCap {
		return s
	}
	runes := []rune(s)
	return string(runes[:previewRuneCap])
}
```

---

### `plugins/signal/deeplink.go` (utility, transform)

**Analog:** `plugins/proton/deeplink.go`

Same overall shape — a small, well-commented pure function building a URL from a stable identifier, with a documented reasoning trail for *why* this fidelity level and *why* this exact target (Proton: All Mail system view because labels aren't name-addressable; Signal: `sgnl://` scheme, conversation-only per CONTEXT.md D-fixed decision). Copy the doc-comment discipline (explain the fidelity choice inline, cite the locked decision by ID) and the "cap+encode a fragment/parameter defensively" idiom if the deep link carries any conversation-name-derived text:
```go
func encodeKeywordFragment(s string) string {
	escaped := url.QueryEscape(s)
	return strings.ReplaceAll(escaped, "+", "%20")
}
```
**Signal deviation:** exact `sgnl://` launch mechanism is Claude's Discretion (CONTEXT.md) — validate hands-on in the spike per RESEARCH.md Assumption A4 before finalizing the URL shape.

---

### `plugins/signal/readonly_test.go` (test)

**Analog:** `plugins/proton/readonly_test.go` — copy the entire AST-walk mechanism near-verbatim, retargeted:

```go
var disallowedSQLIdents = map[string]bool{
	"Exec": true, // any db.Exec call is presumptively a mutation in a read-only plugin
	// Signal needs its own careful audit: unlike IMAP's named verbs
	// (Store/Expunge/...), database/sql's Exec is used for both DDL/DML —
	// the scan should probably instead grep executed SQL TEXT for
	// INSERT/UPDATE/DELETE/PRAGMA-that-mutates, since "Exec" alone doesn't
	// distinguish a read (e.g. none — reads always use Query/QueryRow) from
	// a write. Consider scanning for db.Exec identifier presence at all
	// (this plugin should never call it) rather than a specific-verb list.
}
```
Keep the exact structure: `TestPluginIssuesNoXMutatingCommands` walks non-test `.go` files via `go/ast`, plus a negative-control fixture proving the scanner isn't vacuous (lines 82-97, 112-125) — copy this negative-control idiom verbatim, it directly satisfies the "prove read-only by construction, not convention" bar CONTEXT.md sets even higher for this plugin.

---

### `plugins/signal/outbound_hosts_test.go` (test)

**Analog:** `plugins/proton/outbound_hosts_test.go`

**Pattern, inverted** (lines 8-54): Proton asserts an allowlist of permitted hosts; Signal's egress test asserts **zero** outbound hosts of any kind (RESEARCH.md/CONTEXT.md: "a local-only plugin should have NO outbound network at all"). Structure:
```go
func TestNoOutboundNetworkHosts(t *testing.T) {
	// AST-scan (or similar) every non-test .go file for any import of
	// net/http, net.Dial, or any D-Bus call target other than the local
	// session bus (org.freedesktop.secrets) — assert the only network-
	// shaped call this plugin ever makes is the local D-Bus round-trip,
	// and even that is loopback/session-bus, never a TCP dial to a remote
	// host.
}
```

---

### Shared Patterns

### Config decode / fail-loud-on-startup
**Source:** `plugins/proton/main.go` lines 27-58
**Apply to:** `plugins/signal/main.go`
```go
if raw == "" {
	fatal(fmt.Errorf("WEBSPACES_SOURCE_CONFIG is not set"))
}
```
Every required-field check follows the same one-guard-per-field, fail-fast-on-stderr-then-exit(1) shape (`fatal` helper, lines 76-79). Signal's guards are different fields (no `base_url`/`token`) but the mechanism is identical.

### gRPC error handling
**Source:** `plugins/proton/plugin.go` (throughout — every RPC method)
**Apply to:** `plugins/signal/plugin.go`
```go
return nil, status.Errorf(codes.Unavailable, "proton: connect: %v", err)
// ...
return nil, status.Errorf(codes.NotFound, "proton: source_id %q is not known — the index has not been synced since this plugin started", sourceID)
```
Every plugin prefixes errors with its own name (`"signal: ..."`), uses `codes.Unavailable` for external-dependency failures (DB open failure, keyring failure), `codes.NotFound` for missing source_id, `codes.InvalidArgument` for malformed request, `codes.Internal` for parse/render failures. Signal's schema-version-guard failure should likely be `codes.FailedPrecondition` (a new case not yet used elsewhere in the repo — check other plugins before inventing a fifth code).

### Never-log-a-credential discipline
**Source:** `plugins/proton/credentials.go` (entire file's design), `plugins/proton/plugin.go` line 233's count-only log comment
**Apply to:** `plugins/signal/keyresolve.go`, `secretservice.go`, `safestorage_linux.go`
```go
// Count-only: never a subject, sender, Message-Id, mailbox name,
// base URL or credential. This log is forwarded verbatim into the
// kernel's log stream...
fmt.Fprintf(p.logOut, "webspaces-plugin-proton: match: skipped %d message(s) with no Message-Id header\n", skippedNoMessageID)
```
Signal's version is even stricter (RESEARCH.md: "never log the raw key... even at debug level" — log only "key resolved via legacy field" / "key resolved via safeStorage (backend=X)" presence/backend messages, never the key or ciphertext).

### Sanitize-then-wrap HTML rendition, served through the existing `html` DetailPane variant
**Source:** `plugins/proton/body.go` lines 165-276; frontend: `web/src/lib/components/DetailPane.svelte` line 144 (`{:else if bodyVariant === 'html'}`) and line 159 (`<iframe title={item.title} src={contentUrl(item.id)} class="h-full w-full"></iframe>`)
**Apply to:** `plugins/signal/render.go`, `plugins/signal/plugin.go`'s Fetch
No frontend file needs creating or modifying — CONTEXT.md is explicit ("extend the existing UI contract, don't define a new one"). The Signal thread-view HTML fragment (bubbles/sender grouping/day headers — Claude's Discretion) is sanitized via a Signal-specific bluemonday policy, wrapped via a `WrapDocument`-style helper, and returned as `MimeType: "text/html"` — the kernel's existing `/content` route and DetailPane iframe render it unchanged, exactly as `plugins/silverbullet` and `plugins/proton`'s HTML-fallback path already do.

### Config validation relaxation for a local-path source
**Source:** `kernel/config/config.go` lines 182-194 (`Validate`'s per-source loop)
**Apply to:** `kernel/config/config.go` (modify in place), `kernel/config/types.go`'s `Source` struct
```go
for name, src := range cfg.Sources {
	if strings.TrimSpace(src.BaseURL) == "" {
		return fmt.Errorf("config: source %q has empty base_url%s", name, missingSuffix(missing))
	}
	if strings.TrimSpace(src.Token) == "" {
		return fmt.Errorf("config: source %q has empty token%s", name, missingSuffix(missing))
	}
	...
}
```
This unconditionally requires `base_url`+`token`; Signal (and Phase 5 WhatsApp after it) has neither. The relaxation must be keyed off something structural — e.g. a `Plugin` name allowlist for "local-path sources", or a new explicit `[sources.<name>] local = true` / `path = "..."` field in `types.go`'s `Source` struct (mirroring how `Username`/`WebmailBaseURL`/`CACert` were each added as optional, well-commented fields for Proton's specific needs in Phase 3 — see `types.go` lines 45-83 for that precedent). Whichever shape planning picks, copy the doc-comment discipline of `types.go`'s existing optional fields (explain *why* the field exists, *which* source type needs it, and what "left empty" means for sources that don't).

## No Analog Found

Files with no close match in the codebase (planner should use RESEARCH.md's Code Examples / Pattern sections instead — these are the genuinely new problem classes this phase introduces):

| File | Role | Data Flow | Reason |
|------|------|-----------|--------|
| `plugins/signal/keyresolve.go` | service | transform | No existing plugin resolves a credential from a local Electron config file + dual-shape branch; RESEARCH.md Pattern 1 (dual-shape key resolution) is the reference implementation to follow instead |
| `plugins/signal/safestorage_linux.go` | utility | transform (crypto) | No existing plugin does AES/PBKDF2 unwrap; RESEARCH.md's "Electron safeStorage AES-128-CBC unwrap" code example is the reference, constants must be copied verbatim (not "close enough") |
| `plugins/signal/secretservice.go` | service | request-response (D-Bus) | No existing plugin talks to D-Bus/Secret Service; `github.com/keybase/go-keychain/secretservice` per RESEARCH.md, no in-repo analog for the wrapper shape |
| `plugins/signal/schemaguard.go` | utility | transform | No existing plugin reads/guards a schema version; RESEARCH.md Pattern 2 (fail-loud schema-version guard) is the reference implementation |
| `plugins/signal/dsn.go` | utility | file-I/O | No existing plugin opens a local SQLCipher file; RESEARCH.md Pattern 3 (`mode=ro` DSN, never `immutable=1`) and "Constructing the read-only SQLCipher DSN" code example are the reference |
| `plugins/signal/byte_identical_test.go` | test | — | No existing plugin has a byte-identical-before/after-sync guarantee to test; this is a new, stronger test class specific to SRC-02's success criterion 3 — write from scratch per CONTEXT.md/RESEARCH.md Pitfall 2's explicit scoping guidance (hash `db.sqlite` only, never the `-shm`/-wal sidecars) |
| `plugins/signal/schema_version_fixture_test.go` | test | — | No existing plugin tests an inflated-version negative control against a real fixture DB; `plugins/proton/readonly_test.go`'s negative-control *idiom* (fixture proving scanner isn't vacuous) is a partial style precedent, but the fixture-DB-construction mechanics are new |

## Metadata

**Analog search scope:** `plugins/proton/` (primary), `plugins/silverbullet/`, `plugins/paperless/` (secondary, not deeply read — proton is newer/closer per CONTEXT.md's own guidance: "the freshest full plugin exemplar"), `kernel/config/`, `kernel/syncer/`, `kernel/index/`, `web/src/lib/components/DetailPane.svelte`
**Files scanned:** ~10 read in full (proton's plugin.go, main.go, deeplink.go, credentials.go, body.go, readonly_test.go, outbound_hosts_test.go; kernel/config's config.go, types.go; DetailPane.svelte grepped)
**Pattern extraction date:** 2026-08-03
