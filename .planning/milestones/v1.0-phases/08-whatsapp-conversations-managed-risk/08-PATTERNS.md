# Phase 8: WhatsApp Conversations (Managed Risk) - Pattern Map

**Mapped:** 2026-08-10
**Files analyzed:** 17 (plugin package) + 3 (kernel HTTP surface) + 3 (frontend) + 2 (config/docs)
**Analogs found:** 22 / 25

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|---|---|---|---|---|
| `plugins/whatsapp/main.go` | config/bootstrap | request-response (subprocess handshake) | `plugins/signal/main.go` | exact |
| `plugins/whatsapp/plugin.go` (Describe/Health) | controller (RPC impl) | request-response | `plugins/signal/plugin.go` (`Describe`, `Health`) | exact |
| `plugins/whatsapp/plugin.go` (Match) | controller (RPC impl) | CRUD (read) + error-signaling | `plugins/signal/plugin.go` (`Match`, `openGuarded`) | exact |
| `plugins/whatsapp/plugin.go` (Fetch) | controller (RPC impl) | request-response | `plugins/signal/plugin.go` (`Fetch`, `fetchTranscript`) | exact |
| `plugins/whatsapp/connect.go` | service (lifecycle/connection mgmt) | event-driven (persistent connection) | none in-repo — see "No Analog Found" | none |
| `plugins/whatsapp/eventhandler.go` | service (event ingestion) | event-driven | none in-repo — see "No Analog Found" | none |
| `plugins/whatsapp/messagestore.go` | model/store (own SQLite DB) | CRUD | `plugins/signal/plugin.go` (row-reading helpers: `readConversations`, `readMessages`, etc.) — read shape; write side has no analog (Signal plugin never writes) | role-match |
| `plugins/whatsapp/digest.go` | transform (grouping) | transform | `plugins/signal/digest.go` | exact |
| `plugins/whatsapp/match.go` | transform (matching) | transform | `plugins/signal/match.go` | exact (widen for D-05 1:1 support) |
| `plugins/whatsapp/render.go` | transform (HTML builder) | transform | `plugins/signal/render.go` | exact — reuse near-verbatim per RESEARCH.md ("the source-agnostic bubble/run transcript builder Phase 5's [8's] WhatsApp plugin reuses") |
| `plugins/whatsapp/health.go` | service (state translation) | transform | `plugins/signal/plugin.go` (`Health`) + `openGuarded`'s named-error pattern | role-match |
| `plugins/whatsapp/deeplink.go` | utility | transform | `plugins/signal/deeplink.go` | role-match (different URI scheme, same shape) |
| `plugins/whatsapp/readonly_test.go` | test (AST scan) | — | `plugins/signal/readonly_test.go` | role-match (behavioral variant per RESEARCH.md, since whatsmeow has no SQL-write surface — scan for send-capable `Client` method names instead) |
| `plugins/whatsapp/outbound_hosts_test.go` | test (AST scan) | — | `plugins/signal/outbound_hosts_test.go` | role-match (allowlist WhatsApp's own servers instead of Signal's zero-egress rule) |
| `plugins/whatsapp/go.mod` | config | — | `plugins/signal/go.mod` | role-match (pin/dated-comment convention; no `replace` needed — see RESEARCH.md "Alternatives Considered") |
| `plugins/whatsapp/link/` (CLI link-mode, D-04) | controller (CLI subcommand) | request-response (one-shot) | none in-repo — see "No Analog Found" | none |
| `kernel/httpapi/whatsapplink.go` (new link endpoint, D-01) | controller (HTTP handler) | streaming (SSE/short-poll, TBD) or request-response | `kernel/httpapi/config.go` `DescribePluginHandler` (trial-launch-subprocess-then-relay pattern) | role-match — **strongest available analog for the "spawn a plugin binary as a raw subprocess outside pluginhost, relay its output, then tear it down" shape** |
| `kernel/httpapi/routes.go` (route registration) | route | — | `kernel/httpapi/routes.go` (existing `r.Post("/api/config/describe-plugin", ...)` line) | exact |
| `web/src/lib/components/QRPanel.svelte` (new, D-01/D-02/D-03) | component | streaming (poll/SSE render) | none in-repo — see "No Analog Found" (closest UI precedent is `AddSourceModal.svelte`'s own step-transition state machine) | none |
| `web/src/lib/components/AddSourceModal.svelte` (extend Step 1, D-02) | component | request-response | itself (existing file, being extended) | exact |
| `web/src/lib/components/SourceChip.svelte` (extend menu, D-03) | component | request-response | itself (existing file, being extended) | exact |
| `config.example.toml` (WhatsApp source-block example) | config | — | Signal's existing `[sources.*]` local-path block in the same file | exact |
| `docs/api.md` (new link endpoint doc) | config/docs | — | existing `/api/config/describe-plugin` doc entry | exact |
| `web/e2e/specs/*.spec.ts` (WhatsApp QR/link flow spec) | test (e2e) | — | existing Signal/Phase 7 `AddSourceModal`/`SourceChip` specs under `web/e2e/specs/` | role-match |

## Pattern Assignments

### `plugins/whatsapp/main.go` (config/bootstrap, request-response)

**Analog:** `plugins/signal/main.go`

**Imports pattern** (lines 11-21):
```go
import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"strings"

	goplugin "github.com/hashicorp/go-plugin"

	"github.com/davison/topos/sdk"
)
```

**Core pattern — decode `WEBSPACES_SOURCE_CONFIG`, expand `~`, construct plugin, `goplugin.Serve`** (lines 33-71): copy verbatim, swapping `NewSourcePlugin(configDir)` for whatever constructor opens the WhatsApp plugin's two local paths (whatsmeow `sqlstore` + own message DB) instead of Signal's single `configDir`. Reuse `sdk.GRPCServer` (raised message-size ceiling) exactly as-is — every plugin uses it uniformly.

**Divergence from analog:** unlike Signal (cgo, needs a `sqlcipher` build tag), WhatsApp's `main.go` needs no build tag — pure Go. Also, main.go here should kick off the persistent `Client.Connect()` + background event handler (RESEARCH.md Pattern 1) as part of/just-after plugin construction, something Signal's `main.go` never does (Signal holds no connection between calls at all).

---

### `plugins/whatsapp/plugin.go` — `Describe`/`Health` (controller, request-response)

**Analog:** `plugins/signal/plugin.go` lines 81-88 (`Describe`) and 340-357 (`Health`)

**Describe pattern** (lines 81-88):
```go
func (p *SourcePlugin) Describe(_ context.Context, _ *toposv1.DescribeRequest) (*toposv1.DescribeResponse, error) {
	return &toposv1.DescribeResponse{
		SourceType:      sourceType,
		DisplayName:     displayName,
		ContractVersion: contractVersion,
		MatchVocabulary: matchVocabulary,
	}, nil
}
```
For WhatsApp, `matchVocabulary` must carry **two** fields per D-05 (`groups` and a second 1:1-contacts field — naming at Claude's discretion), not Signal's single `conversations` field.

**Health pattern** (lines 340-357) — reused almost verbatim, but WhatsApp's `Health` must distinguish **four** named states (not-linked / healthy / de-linked-or-banned / session-expired) per D-06/RESEARCH.md Pitfall 2 & Common Pitfalls, whereas Signal's only has "reachable" vs one generic key-resolution-failure message:
```go
func (p *SourcePlugin) Health(_ context.Context, _ *toposv1.HealthRequest) (*toposv1.HealthResponse, error) {
	db, err := p.openGuarded()
	if err != nil {
		return &toposv1.HealthResponse{Reachable: false, LastError: err.Error()}, nil
	}
	defer db.Close()
	return &toposv1.HealthResponse{Reachable: true, LastSyncUnix: time.Now().Unix()}, nil
}
```

---

### `plugins/whatsapp/plugin.go` — `Match` (controller, CRUD-read + error-signaling)

**Analog:** `plugins/signal/plugin.go` lines 152-218, and the error-vs-empty-success rule at lines 159-184

**Critical pattern — never return empty success when unhealthy** (RESEARCH.md Pitfall 1 / Architecture Pattern 3, and Signal's own `openGuarded` guard at line 165-169):
```go
func (p *SourcePlugin) Match(_ context.Context, req *toposv1.MatchRequest) (*toposv1.MatchResponse, error) {
	keywords := req.GetMatchFields()["conversations"].GetValues()
	if len(keywords) == 0 {
		return &toposv1.MatchResponse{}, nil
	}
	db, err := p.openGuarded()
	if err != nil {
		return nil, status.Errorf(codes.Unavailable, "signal: %v", err)
	}
	// ...
}
```
For WhatsApp, `openGuarded()`'s equivalent is a health-state check (`p.healthState()`), not a DB-open call — see RESEARCH.md's own Pattern 3 code example (`switch p.healthState() { case healthStateDelinked, ...: return nil, status.Errorf(codes.Unavailable, ...) }`). **This is the single most load-bearing pattern to copy exactly** — kernel/correlate/correlate.go:105-110 wipes all previously-synced rows on an empty *success*, never on an *error*.

**Structural pattern to copy (build items from digests)** (lines 186-217): unchanged shape — resolve keywords → matched groups/contacts → read local messages → `buildDigests` → `toItem` per digest. Only the underlying data source changes (WhatsApp's own local message store instead of Signal's `db.sqlite`).

---

### `plugins/whatsapp/plugin.go` — `Fetch` (controller, request-response)

**Analog:** `plugins/signal/plugin.go` lines 246-338 (`Fetch`, `fetchTranscript`)

**Core pattern** (lines 254-263, 272-338): re-derive `(conversationID, day)` from `source_id` (via `decodeSourceID`/`sourceIDForDigest` from `digest.go`), re-open the plugin's own local store (never a live remote call — RESEARCH.md Architecture Pattern 2), re-read that day's messages, `renderTranscript(...)`, return `ContentShape: CONTENT_SHAPE_CHAT_TRANSCRIPT`. Copy this whole shape directly; the only structural difference is the WhatsApp plugin's `Fetch` reads its own always-open `messagestore.go` DB (already-connected `*sql.DB` held on `p`) rather than opening-fresh-each-call the way Signal's `openGuarded()` does per RPC (Signal deliberately never holds a connection between calls; WhatsApp deliberately does — RESEARCH.md Summary #1).

---

### `plugins/whatsapp/digest.go` (transform, transform)

**Analog:** `plugins/signal/digest.go` (whole file, 181 lines) — copy near-verbatim.

**Pattern to copy**: `digest` struct shape (lines 12-21), `sourceIDForDigest`/`decodeSourceID` base64 round-trip (lines 33-61), `localDay`/`localDayKey` (`time.Local`, per D-04 restated for WhatsApp — RESEARCH.md Anti-Patterns note this needs restating as "digest everything the local message store has captured" rather than Signal's absolute "no time window"), `buildDigests` grouping by `(conversationID, local day)` (lines 85-116), `tailSnippet`/`Snippet` truncation (lines 129-181). Field/type names should be renamed generically (e.g. `ConversationID` may become `ChatJID`) but the grouping/preview logic is identical.

---

### `plugins/whatsapp/match.go` (transform, transform)

**Analog:** `plugins/signal/match.go` (whole file, 164 lines)

**Pattern to copy**: `matchesAnyKeyword` exact-case-insensitive matching (lines 52-62, "Phase 1 D-03" convention — same rule applies here, no substring/prefix matching). `candidateNames`/`matchesConversation`/`eligibleConversations` (lines 88-142) is the shape to widen for D-05: WhatsApp needs candidates for **both** `groups` (group's own cached name) and 1:1 chats (the address-book/system contact name **only** — D-06, "never the contact's self-chosen push/profile name," identical anti-injection rule to Signal's own `ProfileName`/`ProfileFamilyName` exclusion at lines 30-36, 85-88, 100-106). D-07 ("chats with unsaved contacts are not matchable at all — no phone-number rule") maps directly onto Signal's own precedent of excluding `IsNoteToSelf` unconditionally — apply the same "no candidate names at all" pattern to an unsaved-contact chat.

---

### `plugins/whatsapp/render.go` (transform, transform)

**Analog:** `plugins/signal/render.go` (whole file, 190 lines) — **reuse verbatim per RESEARCH.md's own file-header comment**, which already documents this file as "the source-agnostic bubble/run transcript builder Phase 5's [now 8's] WhatsApp plugin reuses."

**Pattern to copy in full**: `escapeText` (html.EscapeString every interpolated field before assembly — lines 58-60, the security-critical anti-injection guarantee T-05-17), `messageRun`/`buildMessageRuns` (5-minute gap / sender-change run grouping, lines 68-101), `renderTranscript`/`renderBubble` (lines 118-190, unsanitized structural fragment — sanitization happens at the kernel's `rendition.go` boundary, never in this file). No new CSS or content-shape needed (D-11 policy already covers `CONTENT_SHAPE_CHAT_TRANSCRIPT`).

---

### `plugins/whatsapp/deeplink.go` (utility, transform)

**Analog:** `plugins/signal/deeplink.go` (whole file, 74 lines)

**Pattern to copy**: fixed conversation-only fidelity (`LINK_FIDELITY_CONVERSATION_ONLY`), a validated-then-emitted-verbatim scheme URI (Signal's `sgnl://signal.me/#p/<e164>` pattern with an `isValidE164` allowlist regex mirroring the actual consumer app's own validator, lines 5-20, 69-74) — mirror this shape for whatever WhatsApp deep-link scheme the Task-1 spike confirms hands-on (RESEARCH.md Open Question #3 — **do not invent a URI shape without spike verification**, exactly as Signal's own comment warns about the `%2B`-encoding pitfall it hit).

---

### `plugins/whatsapp/readonly_test.go` (test, behavioral AST scan)

**Analog:** `plugins/signal/readonly_test.go` (whole file, 157 lines)

**Pattern to copy**: the `filepath.WalkDir` + `go/ast` scan structure (lines 44-92), including the negative-control fixtures proving the scanner isn't vacuous (lines 69-91) — **this negative-control pattern is mandatory to replicate**, not optional. **Adapt the scan target**: Signal's `disallowedSQLSelectors`/`disallowedSQLSubstrings` (lines 21-34) scan for write-shaped SQL against the *source* DB; WhatsApp's plugin instead needs an AST scan (per RESEARCH.md's own Anti-Patterns section) asserting no send-capable `whatsmeow.Client` method (e.g. `SendMessage`, `SendPresence`, mutation methods) is ever referenced outside `_test.go` files — same `ast.Inspect` + `*ast.SelectorExpr` selector-name-matching mechanism (see `scanASTForWriteShapedSQL`, lines 126-157), different disallowed-name set. Note: WhatsApp's own local message-content store (`messagestore.go`) legitimately calls `Exec`/write SQL against **its own** DB (unlike Signal, which writes nothing) — this test must scope its write-detection specifically to WhatsApp-server-facing calls, not to the plugin's own local store writes.

---

### `plugins/whatsapp/outbound_hosts_test.go` (test, AST scan)

**Analog:** `plugins/signal/outbound_hosts_test.go` (whole file, 149 lines)

**Pattern to copy**: identical `ast.Inspect` scan for `net/http` construction idioms and absolute network-scheme URL literals (lines 26-40, 112-149), plus the mandatory negative-control fixture (lines 76-88). **Adapt the assertion**: Signal asserts the *empty* set (zero legitimate outbound hosts — its only remote-shaped call is a local D-Bus round trip). WhatsApp's plugin legitimately dials WhatsApp's own multi-device servers via whatsmeow's WebSocket client (not `net/http`), so this scan's job narrows to: no `net/http` outbound call exists anywhere in this package (whatsmeow's own transport is not `net/http`-shaped, so the zero-`net/http`-usage assertion likely still holds), and no third-party telemetry/analytics host literal appears.

---

### `plugins/whatsapp/go.mod` (config)

**Analog:** `plugins/signal/go.mod`

**Pattern to copy**: the dated-comment pin convention (the whole `// go.mod replace (Task 1 checkpoint, ...)` comment block, lines 12-21) — for WhatsApp this becomes a **plain `require` with a dated comment** (RESEARCH.md: "no `replace` directive to a third-party fork is needed" — whatsmeow's own upstream has no missing-feature gap the way Signal's SQLCipher fork did), pinning the exact pseudo-version `go.mau.fi/whatsmeow@v0.0.0-20260806224404-e277b766ab33` with a comment recording why that commit was chosen (RESEARCH.md Pitfall 4 — treat any future bump as deliberate).

---

### `plugins/whatsapp/link/` — standalone CLI link mode (D-04, controller, request-response)

**No direct in-repo analog.** Closest conceptual precedent: `plugins/signal/main.go`'s own subcommand-free `main()` structure, extended with a `-link`/subcommand branch (per RESEARCH.md's Open Questions Option A code sketch: `topos-plugin-whatsapp -link -path ~/.config/topos/whatsapp`). Use `github.com/mdp/qrterminal/v3` (already package-legitimacy-audited OK in RESEARCH.md) to render the ASCII QR from whatsmeow's `Client.GetQRChannel`. Must enforce the "never both hold the same `sqlstore` open at once" mutual-exclusion rule (CONTEXT.md hard requirement) — mechanism at Claude's discretion (e.g. a lock file or `sqlstore`'s own file-lock behavior).

---

### `kernel/httpapi/whatsapplink.go` — new in-app QR link endpoint (D-01, controller, streaming/request-response)

**Analog:** `kernel/httpapi/config.go` `DescribePluginHandler` (lines 300-344) — **the strongest available in-repo precedent for "trial-launch a plugin binary as a subprocess, talk to it, then tear it down, without registering it on the running kernel's plugin host."**

**Authorization/validation pattern to copy** (lines 308-323): validate the requested plugin name against `pluginhost.DiscoverAllBinaries` **before** ever executing anything — directory listing, never a caller-supplied path, is the authority over what may be launched (T-07-09's rule, directly applicable to the new link endpoint too, since it also spawns a plugin binary from an HTTP request).

**Error-envelope pattern to copy** (lines 300-344 generally, and `kernel/httpapi/routes.go` lines 130-144 `WriteJSON`/`WriteError`): use the shared `{"schema_version":..., "error": {"code":..., "message":...}}` envelope for any linking failure, and the shared `WriteJSON` success envelope.

**Divergence from analog:** `DescribePluginHandler` is unary request/response (`pluginhost.DescribePluginType`, one round trip, subprocess killed before the handler returns). The new link endpoint needs to **stream** a rotating QR code over the endpoint's lifetime (SSE or short-poll, per CONTEXT.md's Claude's-Discretion note) — no existing handler in this codebase streams; the closest transport precedent for a long-lived per-request loop would need to be newly designed. Route registration should mirror `routes.go` line 87's `r.Post("/api/config/describe-plugin", DescribePluginHandler(...))` placement/shape.

---

### `web/src/lib/components/QRPanel.svelte` (new component, streaming/render)

**No direct in-repo analog** — no existing frontend component polls/streams a rotating image. Closest structural precedent for "a panel with a countdown/expiry and a state transition on external event" is `AddSourceModal.svelte`'s own step-transition state machine (Step 1 → Step 2, the "not linked" branch this panel slots into per D-02) — read `AddSourceModal.svelte`'s existing step-state handling for the modal-composition and prop-passing convention this new component should match, but the QR-polling/rotation logic itself has no reusable precedent in this codebase.

---

### `web/src/lib/components/AddSourceModal.svelte` (extend, controller/component)

Existing file — extend Step 1's "not linked" outcome branch (per D-02) to render `QRPanel` inline. Follow this file's own existing conventions for calling `POST /api/config/describe-plugin` (trial-launch) and branching on the response; the new QR flow is an additional branch alongside the existing ones, not a new file.

---

### `web/src/lib/components/SourceChip.svelte` (extend, component)

Existing file — extend the chip's edit-popover menu (Phase 7 D-12) with a "Re-link…" entry (D-03) opening the same `QRPanel` component in a small dialog. Follow this file's existing menu-item pattern for entry placement and click-handler wiring.

## Shared Patterns

### Match-error-vs-empty-success (criterion 4's entire correctness hinges on this)
**Source:** `kernel/correlate/correlate.go` lines 105-110 (read, not modified this phase); mirrored via `plugins/signal/plugin.go`'s `openGuarded` → `status.Errorf(codes.Unavailable, ...)` shape (lines 115-150, 165-169)
**Apply to:** `plugins/whatsapp/plugin.go`'s `Match` — every non-"linked and healthy" health state returns a gRPC error, **never** `&toposv1.MatchResponse{}`.
```go
switch p.healthState() {
case healthStateDelinked, healthStateBanned, healthStateExpired, healthStateNotLinked:
	return nil, status.Errorf(codes.Unavailable, "whatsapp: %s", p.healthState().Message())
}
```

### Read-only enforcement via AST scan + negative control
**Source:** `plugins/signal/readonly_test.go` lines 44-92 (the `filepath.WalkDir` + `go/ast.Inspect` scan structure and its mandatory negative-control fixtures)
**Apply to:** `plugins/whatsapp/readonly_test.go` — same mechanism, retargeted to whatsmeow's send-capable `Client` methods instead of write-shaped SQL.

### Chat-transcript rendering — zero new kernel/CSS work
**Source:** `kernel/httpapi/rendition.go`'s `CONTENT_SHAPE_CHAT_TRANSCRIPT` policy (sanitize/wrap/theme, unchanged this phase); `plugins/signal/render.go`'s escape-before-assemble discipline
**Apply to:** `plugins/whatsapp/render.go`, `plugins/whatsapp/plugin.go`'s `Fetch` — produce the identical unsanitized structural fragment shape Signal already produces; the kernel's existing policy handles the rest with no new frontend content-shape branch.

### JSON error/success envelope
**Source:** `kernel/httpapi/routes.go` lines 118-144 (`WriteJSON`, `WriteError`, `apiError`/`errorEnvelope` shapes)
**Apply to:** the new `kernel/httpapi/whatsapplink.go` endpoint and any other new HTTP surface this phase adds.

### Trial-launch-a-plugin-binary-as-subprocess, validate-name-before-exec
**Source:** `kernel/httpapi/config.go` `DescribePluginHandler`, lines 300-344 (esp. the `DiscoverAllBinaries` allowlist check at 308-323)
**Apply to:** `kernel/httpapi/whatsapplink.go` — the new link-mode endpoint's subprocess-spawn authorization must follow the identical "directory listing is the authority, never a caller-supplied path" rule.

## No Analog Found

Files with no close in-repo match (planner should lean on RESEARCH.md's own code sketches instead):

| File | Role | Data Flow | Reason |
|---|---|---|---|
| `plugins/whatsapp/connect.go` | service | event-driven | No existing plugin in this repo holds a persistent connection across RPC calls — every other plugin (Signal, Proton, paperless, SilverBullet) opens-and-closes per call. RESEARCH.md's own "Pattern 1" code sketch (this document's Architecture Patterns section) is the closest available reference. |
| `plugins/whatsapp/eventhandler.go` | service | event-driven | Same — no precedent for a background `AddEventHandler`-style continuous ingestion loop anywhere in this codebase. Follow RESEARCH.md's Pattern 1/3 sketches and the `handleEvent` example in "Code Examples." |
| `plugins/whatsapp/link/` (CLI link subcommand) | controller | request-response (one-shot, interactive) | No existing plugin binary has a subcommand/flag-branching `main()` — every plugin's `main()` does exactly one thing (`goplugin.Serve`). Design fresh per RESEARCH.md Open Questions Option A. |
| `kernel/httpapi/whatsapplink.go` streaming transport | controller | streaming | No existing kernel HTTP handler streams (SSE) or holds a request open for short-polling; `DescribePluginHandler` is the closest analog but is strictly unary. Transport choice (SSE vs short-poll) is explicitly left to Claude's discretion per CONTEXT.md. |
| `web/src/lib/components/QRPanel.svelte` | component | streaming | No existing frontend component polls/streams an image with a rotation/expiry countdown. |

## Metadata

**Analog search scope:** `plugins/signal/` (full directory, all 26 files), `plugins/proton/` (referenced but not re-read — already characterized in RESEARCH.md), `kernel/httpapi/` (routes.go, sources.go, config.go), `kernel/correlate/correlate.go` (referenced per RESEARCH.md's own citation, not re-read), `web/src/lib/components/` (AddSourceModal.svelte, SourceChip.svelte — listed, not fully read; RESEARCH.md and CONTEXT.md already characterize their relevant sections)
**Files scanned:** ~24 files read or grepped this session
**Pattern extraction date:** 2026-08-10
