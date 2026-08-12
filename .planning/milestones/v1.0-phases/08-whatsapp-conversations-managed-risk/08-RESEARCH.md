# Phase 8: WhatsApp Conversations (Managed Risk) - Research

**Researched:** 2026-08-10
**Domain:** Running an unofficial WhatsApp linked-device client (`go.mau.fi/whatsmeow`) as a long-lived, self-persisting source plugin — QR pairing, session durability, its own message-content store (WhatsApp is not a durable source), and graceful de-link/ban/expiry degradation inside topos's existing four-RPC, read-only plugin contract.
**Confidence:** MEDIUM — the digest/matching/chat-rendering mechanics are HIGH confidence (direct reuse of the Phase 4 Signal plugin's already-shipped, already-locked pattern, confirmed by reading its source this session). The QR-linking UX mechanism and the exact de-link/ban event taxonomy are genuinely unresolved and need the mandatory spike (already flagged by ROADMAP.md) to close out — do not plan those two areas as settled.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|------------------|
| SRC-03 | WhatsApp plugin runs as a whatsmeow linked device with its own persistent message store; degrades gracefully on de-link/ban; matches on group names | See "Standard Stack" (whatsmeow pin strategy), "Architecture Patterns" (persistent-subprocess event-loop pattern, digest reuse), "Common Pitfalls" (the Match-error-vs-empty-success trap that governs criterion 4), "Runtime State Inventory" (what the plugin's own store must and must not contain), "Open Questions" (the QR-linking UX decision, flagged for a Task 1 spike checkpoint per project convention) |
</phase_requirements>

## Summary

This phase's risk is concentrated in three places, and this research session found that two of them interact with a hard constraint already baked into this project's own contract, not just with whatsmeow's own quirks:

1. **The plugin transport model this project locked in Phase 1 is unary request/response with a fixed four-RPC surface — "no fifth RPC may ever be added" (`docs/plugin-contract.md`, quoted verbatim below) — but WhatsApp's own protocol is fundamentally a persistent, event-driven WebSocket, not something you poll.** The resolution is architectural, not a contract change: the plugin subprocess is *already* long-lived for this project (confirmed by reading `kernel/pluginhost/host.go` and `kernel/supervisor/supervisor.go` this session — a plugin is launched once at kernel boot/config-apply and stays running across every subsequent `Match`/`Fetch`/`Health` call, exactly like every other plugin). This means the WhatsApp plugin can hold a live `whatsmeow.Client` connection open for its own process lifetime, run its own background event-handler goroutine that appends every inbound message to its **own local SQLite store** as it arrives, and let `Match`/`Fetch` do nothing more exotic than query that local store — never talk to WhatsApp's servers synchronously inside an RPC call. This is a materially different runtime shape from every other plugin in this repo (Signal included — Signal's plugin holds no connection between calls at all), but it fits the *existing* subprocess-lifecycle contract with zero proto changes.

2. **QR-code pairing has no home in the locked four-RPC contract, and this is a real, unresolved design decision — not a research gap to paper over.** `Describe`/`Match`/`Fetch`/`Health` have no field for "here is a QR code, refresh me every ~20s until scanned or expired." Two structurally different approaches exist (detailed in "Open Questions" below) and this document deliberately does **not** pick one — ROADMAP.md's own note ("Spike must answer: linking stability... de-link/re-link recovery") plus this project's own Phase 4 precedent (the Signal plugin's SQLCipher driver strategy was *also* left as a Task 1 spike checkpoint rather than settled in research) both point to resolving this with a hands-on spike at plan time, not a research-time guess.

3. **A precise, previously-undocumented failure mode governs whether success criterion 4 ("previously captured messages remain browsable... every other source is unaffected") actually holds.** Reading `kernel/correlate/correlate.go` directly this session (lines 105-110) confirms: when a plugin's `Match` RPC returns a non-nil error, the coordinator `continue`s *without* calling `ReplaceWebspaceSourceItems` — so previously-persisted rows for that source are left untouched. But if `Match` instead returns a **successful, empty** `MatchResponse` (the correct behavior for "genuinely nothing matched," used by every plugin in this repo including Signal), the coordinator calls `ReplaceWebspaceSourceItems(..., nil)` and **wipes every previously-synced item for that source down to zero**. A de-linked/banned WhatsApp plugin that doesn't distinguish these two cases will silently empty the webspace stream on de-link — the exact opposite of what criterion 4 requires. See "Common Pitfalls" #1 for the concrete fix (mirror Signal's `openGuarded`-then-`status.Errorf(codes.Unavailable, ...)` pattern, applied to a "not currently linked" state).

**Primary recommendation:** Build `plugins/whatsapp` as a new pure-Go `go.work` member (no cgo — unlike Signal, `go.mau.fi/whatsmeow`'s own dependency tree is 100% pure Go, confirmed by reading its `go.mod` this session) that (a) at subprocess startup, opens a `whatsmeow/store/sqlstore.Container` at a configured local path (reusing the existing generic `Source.Path` config field Signal already established) and either finds an existing linked device or reports "not linked" via `Health`; (b) if linked, calls `Client.Connect()` once and keeps it connected for the plugin's process lifetime, with a background `AddEventHandler` writing every inbound `events.Message` (and group-metadata change) into the plugin's own separate, plain `modernc.org/sqlite` (pure Go, matching the kernel's own store choice, no cgo) message-content database — WhatsApp's own linked-device protocol is not a durable store (mirrors, but does not literally share code with, `mautrix-whatsapp`'s own documented separation between whatsmeow's session `sqlstore` and its own message table, confirmed via research this session); (c) implements `Match`/`Fetch` by reading **only** the plugin's own local database, reusing the (conversation, local-day) digest shape Phase 4's Signal plugin already established and that Phase 4's own `04-CONTEXT.md` (D-01) explicitly earmarked for this phase to reuse; (d) implements `Health`/`Match` to distinguish "not yet linked," "linked and healthy," and "de-linked/banned/expired" as three named states, returning a gRPC error (never an empty success) for the last two so existing kernel behavior (finding #3 above) does the criterion-4 preservation work for free; (e) leaves the actual QR-scanning UX as this document's central open question, to be resolved by a Task 1 spike checkpoint exactly as Phase 4 resolved its own driver-strategy open question.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| whatsmeow session/device state (keys, contacts cache) | Plugin process (`sqlstore.Container`) | — | whatsmeow's own store package owns this; the kernel never sees a WhatsApp session key |
| Persistent WebSocket connection to WhatsApp servers, background event ingestion | Plugin process (long-lived, held for the subprocess's whole lifetime) | — | This is the phase's core architectural departure from every other plugin — see Summary #1. Lives entirely inside the plugin subprocess, invisible to the kernel's request/response RPC model |
| WhatsApp message content persistence (the plugin's "own persistent message store," SRC-03) | Plugin process (a second, separate local SQLite DB from the session store) | — | whatsmeow's own store is session/device state only, never message content (confirmed via research into `mautrix-whatsapp`'s architecture) — the plugin must build this itself, exactly as SRC-03 requires |
| QR-pairing UX (rendering + relaying the pairing code to the user) | **Undecided — see Open Questions** | Possibly kernel HTTP API + browser, or possibly a standalone CLI entirely outside the kernel's plugin-subprocess lifecycle | Blocked on the locked "no fifth RPC" contract rule; genuinely two different architectures are viable, see below |
| Conversation-day digest assembly, tail-snippet, group-name matching | Plugin process (`Match` RPC) | — | Identical tier to every other plugin's `Match` implementation; reads only the plugin's own local store, never a live WhatsApp call |
| Chat-transcript HTML rendering (structural markup only, unsanitized) | Plugin process (`Fetch` RPC) | — | Mirrors Signal's `render.go` pattern exactly (D-11: sanitization/wrapping/theming is the kernel's job via `ContentShape`) |
| De-link/ban/expiry detection and health-state translation | Plugin process (`Health`/`Match` error surfacing) | — | Same tier and shape as every other plugin's `Health`; no proto change |
| Digest persistence, sync-time correlation, per-(webspace,source) upsert/preserve-on-error | Kernel (`kernel/correlate`, `kernel/index`) | — | Already does the right thing for free (Summary #3) **provided** the plugin returns an error, not an empty success, when de-linked |
| Stream display, "open in WhatsApp" affordance, plugin-health chip | Browser (SvelteKit SPA) | — | Reuses `DetailPane`'s existing `html`/chat-transcript branch (`CONTENT_SHAPE_CHAT_TRANSCRIPT`) and the existing source-health UI (Phase 6) unchanged — no new frontend content-shape needed for the steady-state path. A new frontend surface is needed **only** if the QR-linking UX decision (Open Questions) lands on the in-app option. |

## Standard Stack

### Core

| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `go.mau.fi/whatsmeow` | `v0.0.0-20260806224404-e277b766ab33` [VERIFIED: Go module proxy, `go list -m -json go.mau.fi/whatsmeow@latest`, checked this session] — **the module has never published a tagged release; every consumer pins a pseudo-version to a specific commit**, confirmed by the same lookup returning only pseudo-versions and no semver tags | The only actively-maintained WhatsApp multi-device linked-device client in Go (CLAUDE.md's own locked pick, reconfirmed current this session) | Implements WhatsApp's real multi-device Web protocol client-side; imported by 300+ projects including the production `mautrix-whatsapp` bridge (CLAUDE.md's own citation, reconfirmed live). **Caveat, mirrors Phase 4's Signal-driver finding exactly:** because there is no tagged release, "pin the version" means "pin an exact commit pseudo-version with a dated comment," precisely the pattern `plugins/signal/go.mod`'s own `replace` directive already established in this repo for `mattn/go-sqlite3`'s fork — this is not a new risk class for this project, just the second time it's been hit. |
| `modernc.org/sqlite` | `v1.54.0` [VERIFIED: this repo's own root `go.mod`, already a pinned dependency of the kernel's own index store — `kernel/index/store.go` imports it directly, read this session] | The plugin's own message-content store (separate from whatsmeow's `sqlstore`) | Pure Go — no cgo — matching CLAUDE.md's own explicit "What NOT to Use: `mattn/go-sqlite3` for the kernel's own local index... reserve cgo SQLite for the Signal plugin only, where it's unavoidable" guidance, extended here to the WhatsApp plugin's own local store since nothing about it needs cgo either (unlike Signal, which needs SQLCipher specifically). Reusing the exact package already vetted and pinned elsewhere in this repo is strictly lower-risk than introducing a second SQLite driver. |
| `go.mau.fi/whatsmeow/store/sqlstore` (part of the whatsmeow module, not a separate dependency) | same pseudo-version as whatsmeow | whatsmeow's own session/device/key/contact-cache store | Required by `whatsmeow.NewClient` — this is not optional infrastructure, it is how whatsmeow persists pairing so a restart doesn't require re-scanning (success criterion 1). Documented to auto-`Save()` after a successful pairing with no caller action needed [CITED: pkg.go.dev/go.mau.fi/whatsmeow/store/sqlstore, cross-checked against community usage examples this session]. |

### Supporting

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| `github.com/mdp/qrterminal/v3` | `v3.2.1` [VERIFIED: Go module proxy, `go list -m -versions`, checked this session — real tagged releases through v3.2.1] | Render a QR code as ASCII art to a terminal | **Only needed if the QR-linking UX decision (Open Questions) lands on the standalone-CLI option.** Small, single-purpose, MIT-style license, no further dependencies beyond stdlib-adjacent — the standard choice for every whatsmeow example/tutorial found during this research. |
| A QR-to-PNG/data-URI encoder (e.g. `github.com/skip2/go-qrcode`, not yet verified this session) | — | Render a QR code as an image for in-browser display | **Only needed if the QR-linking UX decision lands on the in-app/kernel-mediated option.** Not verified against the package-legitimacy gate in this session — if that option is chosen, the plan must run the verification protocol on whichever specific package is selected before use. |
| stdlib `database/sql` | stdlib | Query interface over the plugin's own `modernc.org/sqlite`-backed message store | Same pattern as every other plugin's local data access. |
| stdlib `time` | stdlib | Midnight-to-midnight local-day digest bucketing, reusing Signal's `localDay`/`localDayKey` convention verbatim | `time.Local` is correct for the same reason 04-RESEARCH.md's Assumption A5 gave: this is a desktop-local process, per `PROJECT.md`'s own deployment constraint. |

### Alternatives Considered

| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `go.mau.fi/whatsmeow` | `Baileys` (Node.js) | Explicitly rejected by CLAUDE.md's own "What NOT to Use" table — frequent breaking forks, and this project's kernel/plugin architecture has no Node.js runtime anywhere else. Not reconsidered here; nothing in this research changes that verdict. |
| Pinning a whatsmeow commit directly via `go get go.mau.fi/whatsmeow@<commit>` | Vendoring a fork the way Signal's `mattn/go-sqlite3` situation required | Not needed — whatsmeow's *upstream* itself has no tagged releases, but the upstream repo is the canonical, actively-committed source (last commit 2026-08-06, four days before this research session) — there is no missing-feature/unmerged-PR gap here the way there was for Signal's SQLCipher build tag. A plain `go get` pinned to an exact pseudo-version, with a dated comment (mirroring `plugins/signal/go.mod`'s own convention), is sufficient — no `replace` directive to a third-party fork is needed. |
| A second, separate local SQLite DB for message content (recommended) | Storing message content inside whatsmeow's own `sqlstore` schema (e.g. a custom table added to the same DB file) | whatsmeow's `sqlstore` schema is owned and migrated by the whatsmeow module itself — adding custom tables to that same file risks a future whatsmeow migration colliding with or dropping them. A separate file the plugin fully owns (mirroring `mautrix-whatsapp`'s own documented separation) avoids that coupling entirely, at the cost of one extra file to manage. |

**Installation:**
```bash
# Go module dependencies (inside plugins/whatsapp/go.mod, its own go.work member — pure Go, no cgo):
go get go.mau.fi/whatsmeow@v0.0.0-20260806224404-e277b766ab33
go get modernc.org/sqlite@v1.54.0
go get github.com/mdp/qrterminal/v3@v3.2.1   # only if the standalone-CLI QR option is chosen

# Build (no special tag or CGO_ENABLED needed, unlike plugins/signal):
go build -o bin/plugins/topos-plugin-whatsapp ./plugins/whatsapp
```

**Version verification:** every version above was confirmed live against the real Go module proxy during this research session (`go list -m -versions` / `go list -m -json ...@latest`), not taken from training-data memory. whatsmeow's own `go.mod` (read directly from the downloaded module cache this session) confirms a 100% pure-Go dependency tree — `filippo.io/edwards25519`, `go.mau.fi/libsignal`, `golang.org/x/crypto`, `google.golang.org/protobuf`, etc. — no cgo anywhere in it.

## Package Legitimacy Audit

> This phase installs external Go packages. `gsd-tools query package-legitimacy check` only supports `npm`/`pypi`/`crates` ecosystems — Go modules were verified manually via `go list -m -versions`/`go list -m -json` (proves registry/proxy existence and real version/commit history) plus manual review of source/maintainer reputation, mirroring exactly how Phase 4's own research audited its Go dependencies.

| Package | Registry | Age / Last Commit | Downloads/Usage | Source Repo | Verdict | Disposition |
|---------|----------|---------------------|-----------|-------------|---------|-------------|
| `go.mau.fi/whatsmeow` | Go module proxy | Last commit 2026-08-06 [VERIFIED: `go list -m -json ...@latest`], actively developed, no tagged releases ever | Imported by 300+ projects incl. the production `mautrix-whatsapp` bridge (CLAUDE.md's own citation) | github.com/tulir/whatsmeow (mirrored at mau.dev) | SUS | **Flagged, not removed** — real, non-hallucinated, extremely widely used, but the complete absence of tagged releases means "pin the version" is materially riskier than a normal semver dependency (a bad upstream commit ships straight to whatever pseudo-version the plan pins). This is the same risk shape Phase 4 already flagged for `mattn/go-sqlite3`'s fork and accepted with an explicit, dated `replace`/pin comment — the same mitigation applies here. **Requires a `checkpoint:human-verify`-style acknowledgment in the plan before pinning**, per this audit's own disposition rule for SUS packages. |
| `modernc.org/sqlite` | Go module proxy | Already a pinned, in-production dependency of this repo's own kernel index (`kernel/index/store.go`, `go.mod` line 33) [VERIFIED: read directly this session] | N/A — already audited and shipping in this codebase | gitlab.com/cznic/sqlite (mirrored to modernc.org) | OK | Approved — reused, not newly introduced |
| `github.com/mdp/qrterminal/v3` | Go module proxy | Tagged releases through `v3.2.1` [VERIFIED: `go list -m -versions`] | Long-standing, the de facto standard "print a QR code to a terminal" Go package, used across whatsmeow tutorials/examples found during this research | github.com/mdp/qrterminal | OK | Approved — only needed if the standalone-CLI QR option is chosen (Open Questions) |
| `rsc.io/qr` v0.2.0 | Go module proxy | Last tagged release 2018-06-05 [VERIFIED: `go list -m -json rsc.io/qr@latest`, checked 2026-08-10, 08-03-PLAN.md Task 1] — a stable, closed algorithm (ISO/IEC 18004 QR encoding) with no ongoing protocol churn to track, not abandonment risk | Zero transitive dependencies (own `go.mod` declares none); `(*qr.Code).PNG() []byte` returns PNG bytes directly, matching this phase's exact in-app relay need | github.com/rsc/qr (rsc.io — Russ Cox's own personal Go-module namespace; Russ Cox: Go project co-founder/former tech lead) | OK | **Verified 2026-08-10, 08-03-PLAN.md Task 1** — passed the manual Go-ecosystem protocol (`go list -m -versions` confirmed real tagged history: `v0.1.0 v0.2.0`; `go list -m -json rsc.io/qr@latest` confirmed registry existence and current version) on the first candidate — no fallback to `skip2/go-qrcode` or `yeqown/go-qrcode` was needed. Selected for zero dependencies, direct PNG output, and maintainer identity. Pinned in `plugins/whatsapp/go.mod` with a dated comment; confined to the plugin module — the root kernel `go.mod` carries no QR dependency. |

**Packages removed due to [SLOP] verdict:** none — every candidate here is real and non-hallucinated; this audit's job was currency/supply-chain-trust, exactly as it was for Phase 4.
**Packages flagged as suspicious [SUS]:** `go.mau.fi/whatsmeow` (no tagged releases — pin an exact, dated commit pseudo-version with a `go.mod` comment explaining why, mirroring `plugins/signal/go.mod`'s own precedent; this is an accepted, documented risk for this whole project's WhatsApp integration, not something a different package choice would avoid — whatsmeow genuinely is the only viable option, per CLAUDE.md's own analysis).

## Architecture Patterns

### System Architecture Diagram

```
                    Plugin subprocess boot (kernel launches once,
                    keeps running — kernel/pluginhost/host.go,
                    confirmed this session: identical lifecycle to
                    every other plugin)
                              │
                              ▼
              ┌───────────────────────────────┐
              │ 1. Open sqlstore.Container at  │
              │    Source.Path (existing       │
              │    generic config field,       │
              │    Signal's own precedent)     │
              │    → GetFirstDevice()          │
              └───────────────┬─────────────────┘
                    device found?        no device found
                        │                       │
                        ▼                       ▼
          ┌─────────────────────┐   ┌─────────────────────────┐
          │ 2a. Client.Connect() │   │ 2b. "not linked" state — │
          │  (background, held   │   │  Health reports it by    │
          │  for the process's   │   │  name; QR-pairing flow   │
          │  whole lifetime)     │   │  — SEE OPEN QUESTIONS,   │
          └──────────┬───────────┘   │  not resolved here       │
                     │                └─────────────────────────┘
                     ▼
         ┌─────────────────────────────┐
         │ 3. Background event handler  │   events.Message,
         │    (AddEventHandler)          │←──group-metadata events,
         │    → writes every inbound     │   events.LoggedOut,
         │      message + group metadata │   TempBanReason, etc.
         │      to the plugin's OWN,      │   (WhatsApp servers, via
         │      SEPARATE local SQLite     │   the live WebSocket
         │      message store             │   whatsmeow maintains)
         │    → on LoggedOut/ban/expiry,  │
         │      sets an internal named    │
         │      health state — does NOT   │
         │      delete already-stored     │
         │      messages (criterion 3)    │
         └──────────────┬─────────────────┘
                         │  (fully decoupled from Match/Fetch timing —
                         │   this loop runs continuously regardless of
                         │   when the kernel's scheduler next calls Match)
                         ▼
         ┌─────────────────────────────┐
         │ 4. Match RPC (kernel-driven,  │
         │    on the normal sync         │
         │    schedule)                  │
         │    - healthy: query local      │
         │      store, group by           │
         │      (group JID, local day)    │
         │      → conversation-day        │
         │      digests, SAME shape as    │
         │      Signal's D-01/D-02/D-04   │
         │      (04-CONTEXT.md: "Phase 5  │
         │      [now 8] reuses the same   │
         │      shape")                   │
         │    - de-linked/banned/expired: │
         │      return status.Errorf      │
         │      (Unavailable) — NEVER an  │
         │      empty success (see        │
         │      Common Pitfalls #1)       │
         └──────────────┬─────────────────┘
                         ▼
         kernel/correlate.SyncSource → kernel/index
         (error → existing rows untouched, VERIFIED this
          session by reading correlate.go:105-110;
          success → ReplaceWebspaceSourceItems upserts,
          same as every other plugin)
                         │
                         ▼
              GET /api/webspaces/{ws}/stream
                         │
                         ▼
              StreamList / StreamRow (Svelte) — plugin-health
              chip surfaces de-link/ban/expiry (Phase 6's
              existing health-chip pattern, no new UI needed
              for the steady-state error path)
                         │  item opened
                         ▼
         ┌─────────────────────────────┐
         │ 5. Fetch RPC (request-time)   │
         │    re-query the plugin's OWN   │
         │    local store (never a live   │
         │    WhatsApp call — the message │
         │    is already captured),       │
         │    render transcript HTML       │
         │    (mirrors plugins/signal/     │
         │    render.go's structural       │
         │    pattern), ContentShape =     │
         │    CONTENT_SHAPE_CHAT_TRANSCRIPT│
         └──────────────┬─────────────────┘
                         ▼
              DetailPane's existing `html` body-variant
              (unchanged — same sandboxed iframe route
              confirmed by reading DetailPane.svelte this
              session) + OpenInSource
              (LINK_FIDELITY_CONVERSATION_ONLY, deep link
              mechanism TBD — see Open Questions)
```

### Recommended Project Structure
```
plugins/whatsapp/
├── go.mod                  # own module, pure Go (no cgo), own go.work member
├── main.go                 # goplugin.Serve wiring, mirrors plugins/signal/main.go's shape
├── plugin.go                # SourcePlugin: Describe/Match/Fetch/Health
├── connect.go                # sqlstore.Container open, Client.Connect(), lifecycle management
├── eventhandler.go            # AddEventHandler background loop → writes to messagestore.go
├── messagestore.go             # the plugin's OWN modernc.org/sqlite message-content DB (schema, writes, reads)
├── digest.go                   # conversation+local-day grouping — mirrors plugins/signal/digest.go's shape
├── match.go                     # keyword → group-name resolution (groups only, per SRC-03's own wording)
├── render.go                     # thread HTML transcript builder — mirrors plugins/signal/render.go's structural pattern
├── health.go                      # named health-state translation (not-linked / healthy / de-linked / banned / expired)
├── deeplink.go                     # "open in WhatsApp" mechanism — TBD, see Open Questions
├── readonly_test.go                 # this plugin's own AST scan: no outbound WRITE calls to WhatsApp (send-message, etc.) anywhere — PLUG-02's enforcement, adapted (this plugin's "read-only" boundary is behavioral, not SQL-shaped like Signal's, since whatsmeow's Client type DOES expose send methods that must simply never be called)
├── outbound_hosts_test.go           # proves the only outbound network target is WhatsApp's own multi-device servers — no third-party telemetry/analytics call anywhere
└── (link tooling — shape depends on the Open Questions decision)
```

### Pattern 1: Long-lived background connection inside a long-lived plugin subprocess
**What:** Unlike every other plugin in this repo, this plugin holds a live connection open for its entire process lifetime, independent of `Match`/`Fetch` call timing.
**When to use:** Plugin startup (or immediately after a successful QR pairing), held until process shutdown.
**Example:**
```go
// Source: this project's own kernel/pluginhost/host.go (read this session) —
// confirms plugin subprocesses are launched once and reused across every
// RPC, not spawned per call. whatsmeow's own documented Connect/event-
// handler pattern (pkg.go.dev/go.mau.fi/whatsmeow, cross-checked via
// WebSearch this session) fits that lifecycle directly.
func (p *SourcePlugin) startBackgroundClient(ctx context.Context) error {
	container, err := sqlstore.New(ctx, "sqlite3", p.storeDSN(), p.waLog)
	if err != nil {
		return fmt.Errorf("whatsapp: open device store: %w", err)
	}
	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("whatsapp: read device store: %w", err)
	}
	p.client = whatsmeow.NewClient(device, p.waLog)
	p.client.AddEventHandler(p.handleEvent) // eventhandler.go — writes to messagestore.go

	if p.client.Store.ID == nil {
		// Not yet linked — Health reports this by name; QR flow TBD.
		p.setHealthState(healthStateNotLinked)
		return nil
	}
	return p.client.Connect() // held open for the plugin process's lifetime
}
```

### Pattern 2: Match/Fetch never talk to WhatsApp directly — only the plugin's own local store
**What:** `Match` and `Fetch` are pure reads against the plugin's own `modernc.org/sqlite` message-content database, never a live whatsmeow call.
**When to use:** Every `Match`/`Fetch` RPC.
**Why:** Decouples the kernel's sync-schedule cadence from WhatsApp's own event timing (messages arrive continuously via the background handler; the kernel only reads what's already captured, on its own schedule) — and keeps `Fetch`'s per-item latency bounded and independent of WhatsApp server reachability, mirroring Signal's own "content fetched live from THIS project's own local copy, never re-derived from the remote source on every open" pattern (Signal re-reads the local SQLCipher file; this plugin re-reads its own local SQLite file — same shape, different source).

### Pattern 3: Distinguish "genuinely empty" from "cannot currently answer" in Match's return value
**What:** A de-linked/banned/session-expired plugin state returns a gRPC error from `Match`, never a successful empty `MatchResponse`.
**When to use:** Always, whenever the plugin's own connection/health state is anything other than "linked and healthy."
**Example:**
```go
// Source: this project's own kernel/correlate/correlate.go, read directly
// this session (lines 105-110): a Match error leaves previously-persisted
// rows untouched (continue, no ReplaceWebspaceSourceItems call); a
// successful empty response WIPES them via ReplaceWebspaceSourceItems(...,
// nil). This is the single most consequential correctness fact this
// research session found for SRC-03's criterion 4. plugins/signal/
// plugin.go's openGuarded()-then-status.Errorf(codes.Unavailable, ...)
// pattern (read this session) is the exact shape to mirror.
func (p *SourcePlugin) Match(ctx context.Context, req *toposv1.MatchRequest) (*toposv1.MatchResponse, error) {
	switch p.healthState() {
	case healthStateDelinked, healthStateBanned, healthStateExpired, healthStateNotLinked:
		// NEVER return &toposv1.MatchResponse{} here — that is a
		// successful "nothing matched" response and will silently empty
		// this source's stream on the next sync (criterion 4 violation).
		return nil, status.Errorf(codes.Unavailable, "whatsapp: %s", p.healthState().Message())
	}
	// ... healthy path: query the local message store, build digests ...
}
```

### Anti-Patterns to Avoid
- **Calling any whatsmeow send/mutation method from anywhere in this plugin:** `Client` exposes methods like `SendMessage` because the underlying protocol is bidirectional — but this project's `SourcePlugin` contract is read-only by construction (`docs/plugin-contract.md`, quoted in Summary). Never call them; a dedicated AST/behavioral scan test (mirroring `plugins/proton/readonly_test.go`'s pattern, adapted since there's no HTTP-method-shaped signal to grep here) should assert this plugin's source never references a send-capable whatsmeow method name outside test files.
- **Treating "phone offline" the same as "de-linked" or "banned":** WhatsApp's multi-device protocol tolerates the primary phone being offline for up to 14 days before a linked device's session automatically expires [CITED: community-sourced, cross-checked across multiple sources this session — not an official Meta document, treat as MEDIUM confidence]; a session expiring after that window is a distinct, recoverable-by-relinking case from an outright ban (`TempBanReason`) or an explicit `LoggedOut`. Surface these as distinct named health states, not one generic "unavailable" — success criterion 4 explicitly calls out "de-link, ban, or session expiry" as three things the UI must be able to distinguish.
- **Storing message content inside whatsmeow's own `sqlstore` DB file:** see "Alternatives Considered" — a schema whatsmeow itself migrates is not a safe place to add custom tables.
- **Assuming Signal's D-08 ("full history backfill, no time window") applies unmodified to WhatsApp:** Signal Desktop's local DB is a complete, locally-owned copy with no external retention limit, which is what made D-08 safe to state as an absolute. WhatsApp's history-sync is phone/server-governed and bounded (see Open Questions #2) — the plugin can only ever digest what history-sync actually delivered plus whatever it captures live going forward. Restate this phase's equivalent of D-08 as "digest everything the local message store has captured" rather than literally "no time window," since the plugin does not control the window's origin the way Signal's plugin does.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| WhatsApp multi-device Web protocol (Noble/Signal-protocol handshake, E2E session management, WebSocket framing) | A from-scratch protocol client | `go.mau.fi/whatsmeow` | This is precisely CLAUDE.md's own "not a close call" analysis — whatsmeow is the only mature option and reimplementing WhatsApp's reverse-engineered wire protocol is far outside this phase's scope. |
| QR-code rendering (ASCII or image) | A hand-rolled QR encoder | `github.com/mdp/qrterminal/v3` (ASCII) or a dedicated PNG/data-URI encoder (image, TBD per Open Questions) | QR encoding has fiddly error-correction-level and module-size details that are easy to get subtly wrong; both options above are small, single-purpose, and already the community-standard choice for this exact use case. |
| WhatsApp session/device-key persistence | A hand-rolled key store | `go.mau.fi/whatsmeow/store/sqlstore` | This is whatsmeow's own documented, purpose-built store — reimplementing it risks getting the underlying Signal-protocol key material's persistence subtly wrong, a security-critical mistake class. |
| Chat-transcript HTML sanitization | A custom allowlist/DOM walker | The kernel's existing `kernel/httpapi/rendition.go` `CONTENT_SHAPE_CHAT_TRANSCRIPT` policy (already built for Signal, D-11) | Zero new work needed — this plugin produces the same unsanitized structural fragment shape Signal's `render.go` already produces, and the kernel's existing policy sanitizes/wraps/themes it identically. |

**Key insight:** almost nothing in this phase's own message-handling/rendering/matching logic is genuinely novel — it is the *lifecycle* (a plugin that must run a persistent background connection instead of answering stateless calls) and the *pairing UX* (a QR flow with no natural home in the locked four-RPC contract) that are new, and those are exactly the two things this document flags as needing a spike/plan-time decision rather than a research-time guess.

## Runtime State Inventory

> Not a rename/refactor phase, but SRC-03's own success criteria are fundamentally about runtime state this plugin creates and must never lose — answered explicitly below, mirroring how 04-RESEARCH.md treated this same question for Signal.

| Category | Items Found | Action Required |
|----------|-------------|------------------|
| Stored data | Two NEW local files this plugin creates and owns, under the source's configured `Path` (the existing generic local-path config field, `kernel/config/types.go`, read this session): (1) whatsmeow's own `sqlstore` SQLite DB — session/device keys, contact cache; (2) the plugin's own separate `modernc.org/sqlite` message-content DB — every message/group captured while the plugin has been running. Neither exists before this phase; both must survive a kernel/service restart (success criterion 1) and must NOT be deleted on de-link/ban (criterion 4 explicitly requires previously captured messages to remain browsable). |
| Live service config | None — WhatsApp's own servers are not something this project configures; the only "live config" is the pairing relationship itself, stored in file (1) above. |
| OS-registered state | None yet — no `.desktop` entry, no scheme handler. If the "open in WhatsApp" deep link (Open Questions) ends up needing a `whatsapp://` scheme, that's the OS's own WhatsApp Desktop app's registration, not something this project registers — mirrors Signal's identical situation with `sgnl://`. |
| Secrets/env vars | None new — like Signal, the "secret" here (the session's cryptographic keys) is generated and stored entirely inside file (1) above at runtime, never in this project's own config/environment. The only new config key is `path` (already a generic field, per Signal's precedent) pointing at a NEW directory distinct from Signal's — must not collide with `~/.config/Signal` or any other source's path. |
| Build artifacts | None yet — `plugins/whatsapp` does not exist. First build needs a new `go.work` member. Unlike Signal, this is a pure-Go build (`CGO_ENABLED=0`, no special tag) — fits directly into the existing `make build`/`make test-portable` targets (confirmed by reading the `Makefile` this session — Signal alone needed the special `make signal`/`test-signal` cgo targets; this plugin does not). |

## Common Pitfalls

### Pitfall 1: Returning an empty successful `MatchResponse` on de-link/ban silently wipes the stream
**What goes wrong:** A de-linked/banned plugin whose `Match` returns `&toposv1.MatchResponse{}` (the normal, correct way every plugin in this repo signals "genuinely nothing matched") causes `kernel/correlate.SyncSource` to call `ReplaceWebspaceSourceItems(..., nil)`, deleting every previously-captured item for that source from the index.
**Why it happens:** The kernel has no way to distinguish "this source currently has zero matching items" from "this source cannot currently answer" unless the plugin tells it — via the RPC's error channel, not its success payload.
**How to avoid:** Mirror `plugins/signal/plugin.go`'s `openGuarded` → `status.Errorf(codes.Unavailable, ...)` pattern exactly: any non-"linked and healthy" state returns a gRPC error from `Match`, never an empty success (Architecture Pattern 3, above).
**Warning signs:** A test that de-links/bans the plugin (simulated) and then asserts the webspace stream still contains its previously-synced items — if that test can only pass by NOT calling `Match` at all during the de-linked window, the plugin's error-signaling is wrong.

### Pitfall 2: "Session survives restarts" and "session survives phone being offline" are different guarantees
**What goes wrong:** A plan or implementation that treats "restart-durable" (success criterion 1, solved by `sqlstore` persistence) as sufficient for "never needs re-linking" misses that WhatsApp's own protocol independently expires a linked-device session if the primary phone stays offline for an extended period (community-sourced figure: ~14 days [CITED, MEDIUM confidence, not an official Meta document] — reconfirm empirically at spike time), regardless of how healthy this plugin's own process/store is.
**Why it happens:** These are two independent failure/durability axes — one is this project's own persistence (solved, straightforward), the other is WhatsApp server-side session policy (out of this project's control).
**How to avoid:** Health/Match must surface a session-expired state distinctly from a generic "de-linked," per criterion 4's own explicit three-way naming ("de-link, ban, or session expiry"). Do not conflate them into one generic "not available" message.
**Warning signs:** A user reports the plugin "randomly stopped working" after leaving their phone off for an extended trip — if the plugin's health message doesn't name this cause specifically, it's indistinguishable from a ban to the end user.

### Pitfall 3: Treating whatsmeow's own `sqlstore` as if it stores message content
**What goes wrong:** Assuming `Client.Store` or the `sqlstore.Container` gives you a queryable message history "for free" the way Signal Desktop's own DB does — it doesn't. whatsmeow's store is session/device/contact-cache state; message content only ever exists transiently as `events.Message` payloads unless something (this plugin) persists them.
**Why it happens:** The naming ("store") and the fact that Signal's plugin *does* get a complete pre-existing message history for free from a local file invites the same assumption here — but WhatsApp has no local equivalent of Signal Desktop's own complete on-disk history.
**How to avoid:** Build the separate message-content store explicitly (Architecture, Recommended Project Structure) and treat "how much history is available" as bounded by (a) whatever history-sync delivered at link time and (b) everything captured live since — never assume completeness.
**Warning signs:** A `Match` implementation that calls into whatsmeow's own store package looking for message bodies and gets nothing back, or a digest that's suspiciously empty right after linking despite the phone showing months of chat history.

### Pitfall 4: whatsmeow's lack of tagged releases makes "just `go get` the latest" a moving target
**What goes wrong:** Re-running `go get go.mau.fi/whatsmeow@latest` at different points during planning/execution can silently pull in a different commit each time, since there is no semver contract at all — unlike a normal dependency bump, this can change protocol-handling behavior with no changelog to review.
**Why it happens:** whatsmeow deliberately ships as a rolling `main` branch with no release process (confirmed this session — the module's version history contains zero semver tags).
**How to avoid:** Pin an exact pseudo-version in `go.mod` with a dated comment recording why that commit was chosen (mirrors `plugins/signal/go.mod`'s own `replace` directive commentary exactly), and treat any future bump as a deliberate, reviewed action — never an incidental side effect of an unrelated `go get`/`go mod tidy`.
**Warning signs:** `go.sum` changing for `go.mau.fi/whatsmeow` in a commit that isn't explicitly about upgrading it.

## Code Examples

### Group-name matching (mirrors Signal's own conversation-matching shape, groups-only per SRC-03's wording)
```go
// Source: this project's own D-04-equivalent convention (exact,
// case-insensitive matching against native categorization — established
// project-wide since Phase 1) applied to whatsmeow's GetJoinedGroups
// (pkg.go.dev/go.mau.fi/whatsmeow, cross-checked via WebSearch this
// session: returns []*types.GroupInfo, each carrying a Name field and a
// JID). SRC-03's own wording ("matches on group names") and this phase's
// success criterion 2 ("Messages from WhatsApp groups whose names
// match...") both scope this to groups only — no 1:1 chats, unlike
// Signal's D-05 which explicitly included 1:1s. This is a REQUIREMENTS.md-
// /ROADMAP.md-derived scope reading, not an invented decision.
var matchVocabulary = []string{"groups"}

func (p *SourcePlugin) Match(ctx context.Context, req *toposv1.MatchRequest) (*toposv1.MatchResponse, error) {
	keywords := req.GetMatchFields()["groups"].GetValues()
	if len(keywords) == 0 {
		return &toposv1.MatchResponse{}, nil
	}
	if !p.isHealthy() {
		return nil, status.Errorf(codes.Unavailable, "whatsapp: %s", p.healthState().Message())
	}
	// ... query the plugin's own local store for groups whose cached name
	// exact-case-insensitive-matches one of keywords, then digest their
	// captured messages the same way plugins/signal/digest.go does ...
}
```

### Distinguishing de-link/ban/expiry in the background event handler
```go
// Source: whatsmeow's own documented event types (pkg.go.dev/go.mau.fi/
// whatsmeow/types/events, cross-checked via WebSearch this session).
// LoggedOut fires for both a remote unpair AND certain connect-failure
// paths; TempBanReason values (101-106 observed) are a distinct ban
// signal; StreamReplaced signals a same-session conflict (a topos
// operational bug, not a WhatsApp-side event, if it ever fires — this
// plugin should never open two connections against the same store).
func (p *SourcePlugin) handleEvent(evt any) {
	switch e := evt.(type) {
	case *events.LoggedOut:
		// e.Reason distinguishes causes where whatsmeow surfaces one;
		// treat "unknown reason" as the generic de-linked state, never
		// silently as healthy.
		p.setHealthState(healthStateFromLogoutReason(e.Reason))
	case *events.StreamReplaced:
		p.setHealthState(healthStateStreamReplaced) // named distinctly — an operational bug signal, not a user-facing de-link
	case *events.Message:
		p.messageStore.Append(e) // the plugin's own local store — see Pitfall 3
	// ... group-metadata events similarly persisted ...
	}
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| whatsmeow historically required manually calling `Save()` after pairing | Pairing auto-`Save()`s the device store on success | Long-standing whatsmeow behavior, reconfirmed via WebSearch this session | A plan/implementation that adds a manual post-pairing save step is redundant, not wrong — but shouldn't be treated as the thing that "makes persistence work" if it's ever accidentally omitted; the auto-save is the actual guarantee. |
| Single-device WhatsApp Web (log out phone screen off) | Multi-device 2.0 — linked devices operate independently for up to ~14 days without the phone being online | Rolled out well before this research session (multi-device is now WhatsApp's standard behavior) | This is *why* this phase's "runs as a linked device" approach is viable at all for a service that should keep working when the user's phone is asleep/uncharged — but the 14-day figure is the operational boundary condition Pitfall 2 names. |

**Deprecated/outdated:**
- Any research or tutorial assuming whatsmeow requires the older single-device WhatsApp Web behavior (phone must stay actively connected) is describing pre-multi-device WhatsApp and does not apply to a 2026-era linked device.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | The QR-linking UX will be resolved via a Task 1 spike checkpoint choosing between a standalone-CLI flow (Option A) and a kernel-mediated in-app flow (Option B) — this research deliberately does not pick one | Architectural Responsibility Map, Open Questions | If the plan skips this decision and defaults to guessing, it risks building either a kernel HTTP-surface expansion nobody wanted or a CLI-only flow that surprises a user expecting an in-app QR scan |
| A2 | The ~14-day phone-offline session-expiry window is accurate for the WhatsApp version this plugin will run against | Common Pitfalls #2, State of the Art | Sourced from cross-checked community articles, not an official Meta/WhatsApp document — if the real figure differs materially, the plugin's own health-state timing/messaging (not its correctness) would be slightly off; worth a spike-time empirical note if observable |
| A3 | whatsmeow's default (non-`RequireFullSync`) history-sync amount for a linked device registering as a generic Web-protocol client is on the order of WhatsApp's own "web client" default (community-sourced: ~3 months), not the longer "desktop client" default (~1 year) | Open Questions #2 | If wrong, the plan may under- or over-estimate first-link backfill volume/storage — worth confirming empirically at spike time against a real account rather than trusting the community figure |
| A4 | `go.mau.fi/whatsmeow`'s own dependency tree remaining 100% cgo-free (confirmed via its `go.mod` this session) will still be true at whatever exact commit the plan ultimately pins | Standard Stack | If a future whatsmeow commit adds a cgo dependency before the plan pins its version, the "no cgo needed" claim would need re-verification at plan/spike time — re-check `go.mod` against the exact pinned commit, not this session's `@latest` snapshot |
| A5 | SRC-03's "matches on group names" and this phase's success criterion 2 wording together scope matching to WhatsApp GROUPS only, explicitly excluding 1:1 chats (unlike Signal's D-05, which included both) | Code Examples, Match vocabulary | If the user actually wants 1:1 WhatsApp chats included too, this narrower scope reading would need correcting before/during planning — this reading comes directly from REQUIREMENTS.md/ROADMAP.md's own wording, not an invented preference, but was never confirmed via a CONTEXT.md discussion for this phase (none exists) |

## Open Questions

1. **How does the user actually scan a QR code to link WhatsApp — in-app, or via a standalone linking step outside the kernel's normal plugin-subprocess flow?**
   - What we know: the `SourcePlugin` gRPC contract is locked at exactly four RPCs with an explicit "no fifth RPC may ever be added" rule (`docs/plugin-contract.md`, quoted verbatim in Summary) — there is no proto-level channel for a plugin to push a live, rotating QR code to the kernel/browser. Two structurally different resolutions exist:
     - **Option A (recommended default): a standalone one-time CLI linking step.** The same compiled plugin binary, invoked directly by the user in a terminal with a special flag/subcommand (e.g. `topos-plugin-whatsapp -link -path ~/.config/topos/whatsapp`), calls `GetQRChannel`, renders the code via `qrterminal` to the terminal, and on success the pairing is saved into the `sqlstore` DB at the configured path. The kernel's normal plugin-launch path (`pluginhost.launch`, unchanged) then finds an already-linked device on its next launch and just connects — no proto change, no new kernel HTTP surface, structurally respects the locked contract. UX cost: linking happens outside the web app, in a terminal — a real deviation from every other source's in-app "Add source" flow (07-04-PLAN.md's `AddSourceModal`).
     - **Option B: a kernel-mediated, in-app QR flow.** A new kernel HTTP endpoint spawns the plugin binary in a special linking mode as a raw subprocess (bypassing the `go-plugin` gRPC handshake entirely — this is NOT a `SourcePlugin` RPC, so it doesn't violate the four-RPC rule), captures the QR code as it's generated/rotated, and streams it to the browser (e.g. Server-Sent Events or short-poll) for display inside a new `AddSourceModal`-style panel. UX benefit: matches the polished in-app flow every other source already has. Cost: meaningfully more new surface — a new kernel HTTP endpoint, subprocess lifecycle management distinct from `pluginhost`'s existing one, a new frontend QR-display component, and a mutual-exclusion concern (the regular `pluginhost`-launched instance and a manual link-mode subprocess must never both hold the same `sqlstore` file open at once).
   - What's unclear: which UX the user actually wants, and whether Option B's added kernel surface is worth it for a source this project's own ROADMAP already frames as "Managed Risk" / best-effort-droppable.
   - Recommendation: resolve via a Task 1 spike checkpoint, exactly mirroring how Phase 4's own research left its driver-strategy question open for a spike decision rather than guessing. Do not let the plan silently default to either option without an explicit checkpoint.

2. **How much chat history actually backfills on first link, in practice, against a real WhatsApp account?**
   - What we know: without `RequireFullSync`, WhatsApp's servers push a phone-determined amount of recent history automatically on pairing; community sources suggest this varies by what kind of client the linked device presents itself as (community figures: ~3 months for a "web" client profile, ~1 year for a "desktop" client profile) — see Assumption A3.
   - What's unclear: the exact figure for whatsmeow's own default client-profile presentation, and whether it's configurable without also opting into `RequireFullSync`'s heavier sync footprint.
   - Recommendation: this is one of the four things ROADMAP.md's own spike note explicitly names ("how much history backfills on first link") — observe it directly against a real linked account during the spike rather than trusting any web-sourced number.

3. **What deep-link mechanism does "open in WhatsApp" actually use?**
   - What we know: Signal's precedent uses a `sgnl://` URI scheme registered by the Signal Desktop app itself. WhatsApp Desktop likely registers an analogous scheme, but this was not confirmed hands-on this session (no local WhatsApp Desktop install to test against, unlike Signal's Phase 4 research which had a live Signal Desktop install to check).
   - What's unclear: the exact scheme/URI shape, and whether it reliably raises/focuses a running WhatsApp Desktop window the way `sgnl://` was confirmed to for Signal (04-RESEARCH.md's own Assumption A4, never fully resolved there either).
   - Recommendation: verify hands-on during the spike, against whatever desktop WhatsApp client the target machine actually has installed (if any) — `LINK_FIDELITY_CONVERSATION_ONLY` is the fixed fidelity regardless (mirrors Signal), only the exact deep-link URI is open.

## Environment Availability

| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| A WhatsApp account with an active phone to scan the initial QR code | Linking (success criterion 1) | Unknown — not verified this session (no interactive WhatsApp linking was performed; this research did not pair a real device) | — | None — this is a hard prerequisite for the phase to be testable at all; must be confirmed available before planning proceeds to execution |
| Go toolchain, network access to `go.mau.fi`/the Go module proxy | Build | ✓ (confirmed this session — `go list -m` calls against the live proxy succeeded) | Go 1.25.0 (this repo's own `go.mod`/`go.work`) | — |
| No cgo/C toolchain requirement | Build | ✓ N/A — this plugin needs none, unlike Signal | — | — |

**Missing dependencies with no fallback:** a real WhatsApp account and phone available for the mandatory hands-on spike — must be confirmed before the spike (not this research session) proceeds.
**Missing dependencies with fallback:** none identified.

## Validation Architecture

Skipped — `workflow.nyquist_validation` is explicitly `false` in `.planning/config.json`.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | Indirect | This plugin authenticates to WhatsApp's own servers via the linked-device protocol (whatsmeow's own key exchange) — not a username/password/API-token this project manages directly |
| V3 Session Management | Yes | The linked-device "session" (whatsmeow's own key material in `sqlstore`) is the closest analog — must never be logged, must be file-permission-protected the same way Signal's SQLCipher key resolution is treated as sensitive |
| V4 Access Control | Indirect | Existing per-source `agent.read`/`agent.handoff` grant model (AGENT-01) applies unchanged; no new access-control surface introduced |
| V5 Input Validation | Yes | Group-name matching (exact, case-insensitive, no injection surface); HTML sanitization of rendered message content happens at the kernel's existing rendition boundary (unchanged, reused) |
| V6 Cryptography | Indirect | whatsmeow's own Signal-protocol implementation (`go.mau.fi/libsignal`) handles all cryptographic session material — this plugin never implements crypto itself, only calls the library, mirroring the "don't hand-roll" guidance above |
| V12 Files and Resources | Yes | Two new local files (session store, message store) under a fixed, locally-configured path — never derived from untrusted/remote input |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| Accidental outbound WRITE to WhatsApp (sending a message, reacting, etc.) from anywhere in this plugin | Tampering (of the source system, against PLUG-02's read-only guarantee) | A dedicated behavioral/AST test asserting no send-capable whatsmeow `Client` method is ever referenced outside test files — this plugin's version of Signal's `readonly_test.go`, adapted since there's no SQL-string signal to scan here |
| Session-key/credential disclosure via logging | Information Disclosure | Same rule as every other plugin: never log the `sqlstore` DB's key material, never log full message bodies outside the `Fetch` response path itself |
| A stale/orphaned `sqlstore` DB from a de-linked or banned account being silently reused as if still valid | Tampering (of this project's own health-reporting integrity) | Explicit, named health-state tracking (Common Pitfalls #1/#2) — never assume "the store file exists and opens" means "still linked and healthy" |
| Supply-chain risk from pinning an untagged, commit-pseudo-versioned upstream dependency (`go.mau.fi/whatsmeow`) | Tampering (of the build supply chain) | Dated, explicit pin comment (mirrors `plugins/signal/go.mod`'s own precedent); treat any future bump as a deliberate, reviewed action |

## Sources

### Primary (HIGH confidence)
- This repository's own source, read directly this session: `docs/plugin-contract.md` (the locked four-RPC rule), `kernel/pluginhost/host.go` + `kernel/supervisor/supervisor.go` (plugin subprocess lifecycle — long-lived, launched once), `kernel/correlate/correlate.go` (the Match-error-vs-empty-success behavior that governs criterion 4), `kernel/config/types.go` (the generic `Source.Path` field), `plugins/signal/plugin.go` + `render.go` + `digest.go` (the pattern this phase reuses), `.planning/phases/04-signal-conversations/04-RESEARCH.md` and its own `04-CONTEXT.md` D-01 note ("Phase 5 [now 8] reuses the same shape"), `go.mod`/`Makefile` (dependency and build-target confirmation), `web/src/lib/components/DetailPane.svelte` (the existing html-rendition iframe route)
- Go module proxy, queried live this session: `go list -m -json go.mau.fi/whatsmeow@latest`, `go list -m -versions github.com/mdp/qrterminal/v3`, whatsmeow's own downloaded `go.mod` (dependency-tree/cgo confirmation)
- pkg.go.dev/go.mau.fi/whatsmeow and pkg.go.dev/go.mau.fi/whatsmeow/store/sqlstore — official generated API documentation, cross-checked via WebFetch this session

### Secondary (MEDIUM confidence)
- WebSearch-sourced, cross-checked across multiple independent results: whatsmeow's `GetQRChannel` event shape, `LoggedOut`/`StreamReplaced`/`TempBanReason` event semantics, default history-sync behavior and its `RequireFullSync`/`HistorySyncConfig` knobs, `GetJoinedGroups`/`GroupInfo` matching shape, and `mautrix-whatsapp`'s documented separation between whatsmeow's own session store and its own message table

### Tertiary (LOW confidence)
- Community-sourced figures for the ~14-day phone-offline session-expiry window and the ~3-months/~1-year web-vs-desktop history-sync defaults — no official Meta/WhatsApp document was found confirming either number; both are flagged in the Assumptions Log for spike-time empirical confirmation
- General "automation ban risk" search results were dominated by unrelated Meta-API/bulk-messaging ban-risk content (LinkedIn/Instagram/official-WhatsApp-Business-API automation) rather than whatsmeow's own passive, read-only, never-sends usage pattern specifically — treated as not directly applicable; this project's read-only usage pattern structurally avoids the dominant ban vectors those sources describe (bulk sending, spam reports), but the underlying "unofficial client" ToS risk remains regardless of behavior, per CLAUDE.md's own already-accepted framing

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — every version claim was tool-verified against the live Go module proxy this session, not taken from training memory
- Architecture (digest/matching/rendering reuse): HIGH — direct reads of the already-shipped, already-locked Phase 4 Signal plugin source this session
- Architecture (persistent-connection lifecycle, QR-linking UX): MEDIUM/LOW — the lifecycle fit is confirmed by reading this project's own plugin-host code, but the QR-linking mechanism is a genuinely open architectural decision, not a research gap; do not plan it as settled
- Pitfalls: HIGH for the Match-error-vs-empty-success finding (directly verified by reading `kernel/correlate/correlate.go`'s actual behavior this session) — MEDIUM for the de-link/ban/session-expiry event taxonomy (community-sourced, needs spike confirmation against a real linked account)

**Research date:** 2026-08-10
**Valid until:** 14 days — whatsmeow ships as a rolling, untagged `main` branch (a commit landed four days before this research session) and WhatsApp's own server-side protocol/policy behavior can shift without notice; this domain is at least as time-sensitive as Phase 4's SQLCipher research, which used the same 14-day window for the same reason.
