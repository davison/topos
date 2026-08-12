# Phase 6: UI — Scalable Source Surface - Research

**Researched:** 2026-08-06
**Domain:** SvelteKit 5 SPA frontend (header redesign, iframe-rendered content highlighting), Go kernel content-serving boundary (kernel-side HTML mutation under an existing sanitizer)
**Confidence:** HIGH (all four requirements resolve against code read directly this session — no unverified external library claims)

## Summary

This phase touches no new backend contract and installs no new package — it is a redesign of existing, already-shipped surfaces (`SourceHealthChip.svelte` + `SourceFilterChips.svelte` → one merged chip; `OpenInSource.svelte`'s fidelity `Badge`; `DetailPane.svelte`'s four body-variant branches; the stream pane's scrollbar) plus one genuinely new mechanism: injecting search-match highlighting into content the kernel already sanitizes and serves through a sandboxed iframe.

The single highest-leverage finding this session: **`DetailPane.svelte`'s iframe-rendered content (`bodyVariant === 'html'`, i.e. sanitized email HTML, SilverBullet markdown, and Signal chat transcripts) cannot be highlighted from the browser's JavaScript.** The kernel's rendition route (`kernel/httpapi/item.go`'s `renditionHandler`) serves every `text/html` rendition with `Content-Security-Policy: ...; sandbox` — a bare `sandbox` directive with no `allow-same-origin` token. Per the CSP spec (MDN, confirmed this session), that forces the framed document into a null/opaque origin regardless of the fact that it's served from the same host as the SPA — the parent frame's `iframe.contentDocument` access is blocked by the browser's cross-origin frame policy exactly as if the content were on a genuinely different domain. **Highlighting inside the three iframe-rendered content shapes is therefore only possible kernel-side**, inserted into the sanitized fragment after `bluemonday.SanitizeBytes` runs and before (or as part of) `sanitizeAndWrapRendition`'s document-wrapping step — never via client-side DOM manipulation of the iframe, and never via a second pass through the sanitizer (the existing doc comment on `sanitizeAndWrapRendition` already forbids re-sanitizing wrapped output).

The plain-text body variant (`bodyVariant === 'text'`/`'media'`'s trailing text block) is the opposite case: it's rendered directly into the SPA's own DOM via a plain Svelte text binding (`{content?.text}`), so highlighting there is a pure client-side string-segmentation problem — structurally the same shape as the already-shipped `parseSnippet`/`SnippetSegment` pattern in `web/src/lib/format.ts` (used today for FTS5 search-result snippets), just driven by substring-matching the live search query against the full text instead of parsing kernel-supplied STX/ETX delimiters.

The other three requirements are lower-risk, purely additive Svelte work: the merged source chip (UI-07) replaces two existing rows with one, extends an already-client-side filter (`filterItemsBySource` in `format.ts` — the stream endpoint has no server-side filter param today, confirmed by reading `docs/api.md`, so multi-select needs no kernel change); fidelity differentiation (UI-08) is a styling-only change to `OpenInSource.svelte`'s existing `Badge`; and the stream scrollbar's date markers (UI-11) are computable client-side from `StreamItem.timestamp_unix`, which the stream response already carries, over the existing non-virtualized, fixed-row-height list (`StreamList.svelte` renders a plain `{#each}`, no windowing library is in use).

**Primary recommendation:** Implement UI-09's iframe-variant highlighting as a kernel-side, tree-based (not regex) text-node walk over the already-sanitized HTML using `golang.org/x/net/html` (already an indirect transitive dependency via `bluemonday`, confirmed in `go.mod`/`go.sum` — promoting it to a direct import needs no new package). Implement everything else (chip merge, fidelity badge, plain-text highlighting, scrollbar markers) as client-side Svelte/TypeScript changes reusing the established `format.ts` pure-function-plus-component-test pattern.

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| Merged source chip (health + filter + refresh), UI-07 | Browser/Client (Svelte) | API/Backend (`GET /api/sources`, unchanged) | Purely a rendering/interaction redesign of data the API already returns; no new endpoint or field needed. |
| Multi-select source filtering, D-02 | Browser/Client (Svelte + URL state) | — | `GET /api/webspaces/{ws}/stream` has no source-filter query param today (confirmed against `docs/api.md`) — filtering has always been a client-side narrowing of the already-fetched `items` array (`filterItemsBySource`). Extending single-select to multi-select stays entirely client-side. |
| Deep-link fidelity differentiation, UI-08 | Browser/Client (Svelte) | — | `link.fidelity` is already on every stream item (`PLUG-03`, surfaced via `StreamItem.link.fidelity` in the API response) — UI-08 is styling/copy only, no contract change (per CONTEXT.md notes). |
| Search-term highlighting — plain-text variant, UI-09 | Browser/Client (Svelte) | — | Text is rendered directly into the SPA's own DOM (`{content?.text}` in `DetailPane.svelte`) — same-document, so client-side segmentation (à la `parseSnippet`) is sufficient and requires no kernel change. |
| Search-term highlighting — sanitized-HTML/markdown/chat-transcript variants, UI-09 | API/Backend (kernel content route) | — | These variants render inside a sandboxed iframe served under CSP `sandbox` (no `allow-same-origin`) — the browser enforces an opaque cross-origin boundary the parent SPA cannot cross with `contentDocument` regardless of same-host serving. Highlighting must be injected before the document reaches the browser. |
| Themed scrollbars + stream date markers, UI-11 | Browser/Client (Svelte + CSS) | — | Base theming (260805-j98) is pure CSS already shipped app-wide, including inside kernel-served rendition documents (`renditionBaseStyle` in `kernel/httpapi/rendition.go`). The remaining date-marker overlay is computed from `StreamItem.timestamp_unix`, already present in the already-fetched stream response — no new data needed. |

## Package Legitimacy Audit

**No new external packages this phase.** Every capability above is implemented with dependencies already present in the repo:

| Package | Registry | Status | Disposition |
|---------|----------|--------|-------------|
| `bits-ui` (already `^2.18.1` in `web/package.json`) | npm | Already installed; `dropdown-menu`, `popover`, and `collapsible` primitives confirmed present in `web/node_modules/bits-ui/dist/bits` this session `[VERIFIED: web/node_modules/bits-ui/dist/bits listing]` | If UI-07's overflow/collapse treatment (Claude's Discretion) needs a dropdown or popover affordance, scaffold a new shadcn-svelte-style wrapper component under `web/src/lib/components/ui/` — no `npm install` needed. |
| `golang.org/x/net` v0.53.0 (already `// indirect` in `go.mod`) | Go modules | Already resolved transitively via `github.com/microcosm-cc/bluemonday` `[VERIFIED: go.mod:23, go.sum:43-44]` | If kernel-side highlighting is implemented via `golang.org/x/net/html` tree walking (recommended below), the import becomes direct — `go mod tidy` will promote the existing entry, not add a new dependency. |

**Packages removed due to SLOP verdict:** none.
**Packages flagged as suspicious (SUS):** none.

## Architecture Patterns

### System Architecture Diagram

```
Browser (SvelteKit SPA)                          Kernel (Go, same origin, loopback HTTP)
┌───────────────────────────────┐                ┌─────────────────────────────────────┐
│ +page.svelte (webspace route)  │  GET /api/sources │
│  ├─ WebspaceHeader.svelte      │ ───────────────►│ SourcesHandler                        │
│  │   └─ SourceChip.svelte (NEW,│                │  (health + display_name, unchanged)   │
│  │      merges HealthChip+      │◄───────────────│                                        │
│  │      FilterChips, UI-07)     │                │                                        │
│  │      · toggles multi-select  │                │                                        │
│  │        filter set (D-02)     │                │                                        │
│  │      · hover/focus reveals   │  GET .../stream │                                        │
│  │        refresh (D-03)        │ ───────────────►│ StreamHandler (unfiltered items;      │
│  │      · hover tooltip = health│◄───────────────│  filter narrowing stays client-side)  │
│  │        detail (D-04)         │                │                                        │
│  ├─ StreamList.svelte           │                │                                        │
│  │   filterItemsBySource()      │                │                                        │
│  │   (extended to a Set, D-02)  │                │                                        │
│  │   + date-marker overlay      │                │                                        │
│  │     (UI-11, computed from    │                │                                        │
│  │     visible item timestamps) │                │                                        │
│  └─ DetailPane.svelte           │  GET /api/items/{id}                                     │
│      ├─ OpenInSource.svelte     │ ───────────────►│ ItemHandler (adds fidelity, unchanged)│
│      │   fidelity-differentiated│◄───────────────│                                        │
│      │   affordance (UI-08)     │                │                                        │
│      ├─ bodyVariant 'text'/     │                │                                        │
│      │   'media' text block:    │   (no request — text is already in the                  │
│      │   CLIENT-SIDE highlight  │    ItemHandler response fetched above)                  │
│      │   of searchQuery matches │                │                                        │
│      │   (new highlightText()   │                │                                        │
│      │   helper, UI-09)         │                │                                        │
│      └─ bodyVariant 'html':     │  GET /api/items/{id}/content?hl=<query>                  │
│          <iframe src=           │ ───────────────►│ renditionHandler                       │
│           contentUrl(id, query)>│                │  1. Fetch from plugin                  │
│          (UI-09 — query passed  │                │  2. sanitizeAndWrapRendition:           │
│          as a URL param so the  │                │     a. bluemonday.SanitizeBytes()       │
│          browser requests a     │                │     b. NEW: highlightTextNodes()        │
│          highlighted document,  │                │        (x/net/html tree walk, wraps     │
│          since it cannot inject │                │        query matches in <mark>,         │
│          into the iframe itself)│                │        text nodes only — never touches │
│                                  │                │        tag/attribute bytes)             │
│                                  │◄───────────────│     c. wrap in <!doctype html>...       │
│                                  │  sandboxed      │        (CSP: sandbox, no allow-same-   │
│                                  │  iframe document │        origin — opaque origin, parent  │
│                                  │  (opaque origin)│        cannot reach in with JS)        │
└───────────────────────────────┘                └─────────────────────────────────────┘
```

### Recommended Project Structure

No new top-level directories. Files this phase most likely touches or adds:

```
web/src/lib/
├── format.ts                      # extend: multi-select filter helpers, highlightText() for plain-text/media variant
├── format.test.ts                 # extend: new helpers get the same pure-function unit-test treatment
├── components/
│   ├── SourceChip.svelte          # NEW — replaces SourceHealthChip.svelte + SourceFilterChips.svelte
│   ├── WebspaceHeader.svelte      # rewire to render one SourceChip row instead of two rows
│   ├── OpenInSource.svelte        # UI-08 — fidelity-differentiated treatment
│   ├── DetailPane.svelte          # UI-09 — pass searchQuery down, highlight text/media branches, append ?hl= to contentUrl for the html branch
│   ├── StreamList.svelte          # UI-11 — date-marker overlay alongside the scroll region
│   └── sources.test.ts            # extend/rename for the merged chip's behavior
web/src/app.css                    # no change expected (base scrollbar theming already shipped, 260805-j98)
kernel/httpapi/
├── item.go                        # UI-09 — read `?hl=` query param, thread to sanitizeAndWrapRendition
├── rendition.go                   # UI-09 — new highlightTextNodes() step between sanitize and wrap
├── rendition_test.go              # extend — highlighting must not alter sanitizer test assertions
└── agent.go                       # decide: does /agent/v1's rendition mirror accept ?hl= too, or omit it (agent surface has no search UI)
```

### Pattern 1: Client-side multi-select source filter (extends D-09/A-UI-02)

**What:** `resolveSourceFilter`/`filterItemsBySource` in `format.ts` currently take a single `string | null`. D-02 requires a `Set<string>` (empty set = show everything), persisted in the URL as `?sources=a,b` (a rename/pluralization of today's `?source=` param).

**When to use:** `WebspaceHeader`/`SourceChip` toggle membership; `+page.svelte`'s `setFilter` becomes `toggleFilter(name)`; `StreamList`'s `filterItemsBySource` takes the set instead of one id.

**Example (existing single-select shape being extended):**
```typescript
// Source: web/src/lib/format.ts (read this session, lines 120-129)
export function resolveSourceFilter(requested: string | null, sources: SourceStatus[]): string | null {
  if (!requested) return null;
  return sources.some((s) => s.name === requested) ? requested : null;
}

export function filterItemsBySource(items: StreamItem[], source: string | null): StreamItem[] {
  if (source === null) return items;
  return items.filter((item) => item.source === source);
}
```
The multi-select versions should keep the same "unrecognised/stale value degrades silently" discipline (T-02-17) — a `?sources=` value naming an unconfigured instance is dropped from the set, not treated as an error, exactly like today's single-value fallback-to-`null`.

### Pattern 2: Plain-text query highlighting (client-side, new — sibling to `parseSnippet`)

**What:** `parseSnippet` (verified this session, `web/src/lib/format.ts:244-287`) already implements exactly this shape of problem — splitting a string into ordered `{text, match}` segments — but it parses kernel-supplied `SnippetOpen`/`SnippetClose` (`\x02`/`\x03`) delimiters from FTS5's `snippet()` output. UI-09's plain-text detail-pane case has no delimited string from the kernel (`ItemHandler`'s `itemContent.Text` is the plugin's raw extracted text) — highlighting must be computed client-side against the live `searchQuery` state already held in `+page.svelte`.

**When to use:** `DetailPane.svelte`'s `loadedTextBlock` snippet (both the `'text'` branch and the `'media'` branch's trailing text block).

**Recommended approach:** a new `highlightText(text: string, query: string): SnippetSegment[]` helper reusing the existing `SnippetSegment` type — case-insensitive substring match against each whitespace-split term of `query` (mirroring the kernel's own `ftsQuery` tokenization: `strings.Fields`-equivalent `query.trim().split(/\s+/)`, verified this session at `kernel/index/store.go:375-390`), never a regex built directly from unescaped user input (a raw `new RegExp(query)` would let a query containing `.`/`*`/`(` throw or over-match — escape every regex metacharacter first, or avoid regex entirely with indexOf-based scanning).

**⚠ Requires plumbing:** `DetailPane.svelte` currently receives no `searchQuery` prop (`web/src/lib/components/DetailPane.svelte:18-22`, confirmed this session) — `+page.svelte` must pass its existing `searchQuery` state down alongside `item`/`displayName`/`sourceReachable`.

### Pattern 3: Kernel-side tree-walk highlighting for iframe-rendered variants (new)

**What:** Insert a text-node-only highlighting pass between `bluemonday.SanitizeBytes` and the document-wrap step in `sanitizeAndWrapRendition` (`kernel/httpapi/rendition.go:293-309`, read in full this session).

**Why not regex-over-HTML-bytes:** `sanitized` at that point is a byte slice of well-formed (post-bluemonday) HTML. A naive `bytes.Replace`/regex substitution for the query string would match inside attribute values (`href="mailto:boiler@..."`), inside already-emitted `<mark>` tags on a repeat highlight, or split a multi-byte UTF-8 rune across a match boundary if not rune-aware — silently corrupting the document bluemonday just proved safe. This is exactly the kind of hand-rolled HTML mutation the project's existing `sanitizeAndWrapRendition` doc comment warns against re-running the sanitizer over (a wrapped/mutated document must never be fed back through `policy.SanitizeBytes`, since the wrapping step's own literal `<style>`/`<mark>` tags are Go-authored trusted markup, not sanitizer-approved user content).

**Recommended approach:** parse `sanitized` with `golang.org/x/net/html` (`html.Parse`), walk only `html.TextNode` nodes (skipping `<script>`/`<style>` subtrees — moot here since bluemonday already strips `<script>` and the only `<style>` is the kernel's own later-injected stylesheet, added after this step), replace each case-insensitive term match with a `<mark>` element node inserted into the tree (not as raw text), then `html.Render` the mutated tree back to bytes. This guarantees the walk only ever touches genuine rendered text content and can never produce malformed markup, because the DOM tree — not string bytes — is the unit of mutation.

**Example (`golang.org/x/net/html`'s documented tree-walk shape — pattern is well-established Go html package idiom, not project-specific code):**
```go
// Illustrative — no such helper exists in the repo yet; this sketches the
// approach for kernel/httpapi/rendition.go, to be written as part of this
// phase's plan. golang.org/x/net/html is already resolved transitively
// (go.mod:23, go.sum:43-44) via bluemonday.
func highlightTextNodes(doc *html.Node, terms []string) {
    var walk func(*html.Node)
    walk = func(n *html.Node) {
        if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
            return // never descend into non-rendered content
        }
        if n.Type == html.TextNode {
            // split n.Data on term matches, replace this text node with a
            // sequence of TextNode/mark-ElementNode siblings in place
        }
        for c := n.FirstChild; c != nil; c = c.NextSibling {
            walk(c)
        }
    }
    walk(doc)
}
```

**Wiring:** `renditionHandler` (`kernel/httpapi/item.go:153-244`) needs to read a new query parameter (e.g. `?hl=`) off the request and thread it through to `sanitizeAndWrapRendition`, which passes it to the new highlighting step before wrapping. The frontend's `contentUrl(id)` helper (`web/src/lib/api.ts:195-198`) needs an optional query-string param appended when a search is active, since `DetailPane.svelte`'s iframe `src` is the only channel available to reach kernel-side logic (confirmed: no `postMessage` channel exists into the sandboxed document, and none can be added — see Common Pitfalls).

### Anti-Patterns to Avoid

- **Regex substitution over already-sanitized HTML bytes to inject `<mark>` tags:** breaks on matches inside attribute values, inside existing tags, or across multi-byte rune boundaries. Use the tree-walk pattern above.
- **Attempting `iframe.contentDocument`/`contentWindow` access from the parent SPA to inject highlighting client-side:** will throw a `SecurityError` (or silently see an inaccessible/blank document) because the CSP `sandbox` directive (no `allow-same-origin`) forces an opaque origin — confirmed via MDN this session (see Common Pitfalls).
- **Re-running `bluemonday.Policy.SanitizeBytes` after inserting `<mark>` tags:** the existing `sanitizeAndWrapRendition` doc comment already establishes wrapped output is never fed back through the sanitizer — the highlighting step must sit *inside* that boundary (between sanitize and wrap), not as a second sanitize pass.
- **Building the multi-select filter as a server-side query param** when the stream endpoint already returns every item and filtering has always been client-side (`filterItemsBySource`) — adding a server round-trip per filter toggle would be a regression in interaction latency for no benefit, since the full item set is already in memory.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Overflow/collapse affordance for 10+ source chips (UI-07 discretion) | A custom popover/dropdown positioning implementation | `bits-ui`'s already-installed `dropdown-menu` or `popover` primitive (`web/node_modules/bits-ui/dist/bits/{dropdown-menu,popover}`, confirmed present this session), wrapped as a new shadcn-svelte-style component under `web/src/lib/components/ui/` — the same pattern every other primitive in that directory (`Tooltip`, `Alert`) already follows | Custom focus-trap/positioning/escape-key handling for a floating panel is exactly the class of accessibility bug shadcn/bits-ui primitives already solve; no new npm dependency is needed since bits-ui already ships it. |
| HTML text mutation for highlighting | String/regex splicing on raw HTML bytes | `golang.org/x/net/html`'s parse/walk/render (already transitively resolved) | See Pattern 3 and Anti-Patterns above — regex-over-markup is the single highest-risk hand-rolled approach in this phase. |
| Query-term tokenization for highlighting | A bespoke tokenizer/stemmer to "match what FTS5 matched" | Simple whitespace-split, case-insensitive substring match — mirroring the kernel's own `ftsQuery` (`kernel/index/store.go:375-390`, which itself just does `strings.Fields` + literal-phrase quoting, no stemming) | The kernel's own FTS5 usage (`unicode61` tokenizer, no stemming configured) already treats query terms as literal substrings; a fancier client/kernel-side highlighter would highlight *more* than FTS5 actually matched, misleading the user about why an item surfaced. |

**Key insight:** Every "don't hand-roll" risk in this phase is about *not* reaching for string-level hacks (regex-over-HTML, hand-rolled popover positioning) when a tree-based or component-based primitive already sits one import away.

## Common Pitfalls

### Pitfall 1: Assuming same-origin means script-accessible for the rendition iframe

**What goes wrong:** A plan that tries to highlight iframe content by having the parent SPA's JS reach into `iframe.contentDocument` after load (since the rendition route *is* same-host as the SPA) will fail silently or throw, because same-host does not mean same-origin once the response carries `Content-Security-Policy: sandbox` with no `allow-same-origin` token.
**Why it happens:** The `sandbox` CSP directive (verified this session at `kernel/httpapi/item.go:235`) forces the framed document into a unique/opaque origin per spec — this is an intentional, documented hardening choice from Phase 1/5 (T-01-10, D-11), not an oversight, and it must not be weakened to enable this phase's highlighting (CONTEXT.md: "sanitizer contract... untouchable").
**How to avoid:** Do all iframe-variant highlighting kernel-side, before the document is ever sent to the browser (Pattern 3).
**Warning signs:** A `DOMException: Blocked a frame with origin "..." from accessing a cross-origin frame` in the browser console, or `contentDocument` silently returning `null`/an inaccessible document during manual testing.

### Pitfall 2: Highlighting breaks the chat-transcript class-token allowlist

**What goes wrong:** `newChatRenditionPolicy` (`kernel/httpapi/rendition.go:103-107`, read this session) allows a `class` attribute on `div` elements *only* matching `chatTranscriptClassTokens` — a fixed, closed regex of tokens (`run`, `own`, `other`, `sender-name`, `bubble`, `tombstone`, `quote`, `timestamp`, `edited-suffix`, `attachment`, `reaction`, `body`). If the highlighting step is implemented as a second bluemonday sanitize pass (violating Pattern 3's guidance) rather than direct tree insertion, any new `<mark>` element or class the highlighter introduces would need its own policy entry or get stripped.
**Why it happens:** Reflexively reaching for "run it through the sanitizer again to be safe" when the actual contract is the opposite: sanitize once, then trust your own Go-authored insertions.
**How to avoid:** Insert `<mark>` as a bare element (no class/attributes needed — style it via the existing per-shape stylesheet composition in `rendition.go`, e.g. add a `mark { background: ...; color: ...; }` rule to each shape's delta constant) and never route the highlighted tree back through `policy.SanitizeBytes`.
**Warning signs:** `rendition_test.go`'s existing chat-transcript class-allowlist tests (`TestChatRenditionPolicy...`, confirmed present in the file read this session) start failing, or highlighted output loses its `<mark>` wrapping unexpectedly.

### Pitfall 3: Forgetting the `/agent/v1` rendition mirror also calls `sanitizeAndWrapRendition`

**What goes wrong:** `agent.go:341` (confirmed this session) calls the same `sanitizeAndWrapRendition` function `item.go`'s `renditionHandler` does. If the new `?hl=` parameter and highlighting step are wired only into `item.go`'s call site, the function signature still changes for every caller — a compile-time forcing function, not a silent gap, but the *decision* of whether the agent-facing mirror accepts a highlight query at all (it has no search UI — AGENT-01/AGENT-02 readiness, not a human surface) needs an explicit choice in planning, not a default.
**Why it happens:** `sanitizeAndWrapRendition` is a shared low-level function; a phase that only reads `item.go` before implementing will miss the second call site.
**How to avoid:** Grep for `sanitizeAndWrapRendition(` before implementing (two call sites confirmed this session: `kernel/httpapi/item.go:202`, `kernel/httpapi/agent.go:341`) and decide explicitly: pass `""` (no highlight) from the agent path, or thread it through identically.
**Warning signs:** A Go compile error the moment the function signature changes (safe — but decide the semantic answer, not just satisfy the compiler with an empty string by accident).

### Pitfall 4: Stale-value filter degradation must survive the single→multi-select rewrite

**What goes wrong:** T-02-17's rule ("an unrecognised or stale `?source=` value degrades to `null`/no-filter, never an error or empty list") must carry forward per-value into the new `Set<string>` shape — dropping only the unrecognised member(s), not the whole set, and not erroring if the URL contains a since-removed instance id alongside valid ones.
**Why it happens:** A straightforward `Set` port might validate the whole `?sources=` list as one unit (all-or-nothing) rather than filtering member-by-member, changing observable behavior for a partially-stale bookmark.
**How to avoid:** Filter membership per-value, exactly as `resolveSourceFilter`'s existing single-value degrade does, just applied across the set.
**Warning signs:** A saved/shared link with one now-deleted source instance and one still-valid one silently shows nothing instead of narrowing to the still-valid instance.

## Code Examples

### Existing snippet-segment pattern to model the new plain-text highlighter on

```typescript
// Source: web/src/lib/format.ts, lines 229-287 (read this session)
export interface SnippetSegment {
  text: string;
  match: boolean;
}

export function parseSnippet(snippet: string): SnippetSegment[] {
  // ... splits a kernel-delimited (SnippetOpen/SnippetClose = \x02/\x03)
  // string into ordered plain/matched segments, degrading to one
  // plain-text segment on any malformed delimiter run rather than
  // throwing (T-03-22). The new highlightText(text, query) helper should
  // return this identical SnippetSegment[] shape so StreamRow.svelte's
  // existing render pattern (line 119: `{#each parseSnippet(snippet) as
  // segment, i (i)}`) can be reused verbatim in DetailPane.svelte.
}
```

### Existing kernel query-term tokenization to mirror for highlight-term extraction

```go
// Source: kernel/index/store.go, lines 375-390 (read this session)
func ftsQuery(raw string) string {
  fields := strings.Fields(raw)
  // ... each field quoted as a literal phrase, joined with FTS5's
  // implicit AND, last term suffixed * for prefix-match. The
  // highlighting term list (both client-side and kernel-side) should
  // derive from the SAME strings.Fields(raw)-equivalent tokenization —
  // not FTS5's internal unicode61 tokenizer output — so "what gets
  // highlighted" matches "what the user typed", consistent with how
  // ftsQuery treats the raw query.
}
```

### Existing rendition sanitize/wrap boundary highlighting must respect

```go
// Source: kernel/httpapi/rendition.go, lines 293-309 (read this session)
func sanitizeAndWrapRendition(shape toposv1.ContentShape, fragment []byte) ([]byte, error) {
  policy, ok := renditionPolicies[shape]
  if !ok {
    return nil, fmt.Errorf("%w: %v", errUnrecognisedContentShape, shape)
  }
  sanitized := policy.SanitizeBytes(fragment)
  style := stylesheetForShape(shape)
  // NEW highlighting step belongs HERE — operating on `sanitized`,
  // producing a new []byte that is still wrapped exactly as before.
  // It must never re-enter policy.SanitizeBytes.
  var buf bytes.Buffer
  buf.WriteString("<!doctype html>\n<html><head><meta charset=\"utf-8\"><style>")
  buf.WriteString(style)
  buf.WriteString("</style></head><body>")
  buf.Write(sanitized) // <- highlighted bytes go here instead
  buf.WriteString("</body></html>")
  return buf.Bytes(), nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|---------------|--------|
| Two header rows (`SourceHealthChip` row + `SourceFilterChips` row) | One merged chip per instance (UI-07) | This phase | `SourceHealthChip.svelte`/`SourceFilterChips.svelte` are absorbed/retired; `WebspaceHeader.svelte`'s `shouldShowSourceRows` gating logic must be preserved on the new single row. |
| Single-select source filter (`?source=`, Phase 2 D-09) | Multi-select (`?sources=a,b`, this phase's D-02) | This phase | Superseded per CONTEXT.md — `resolveSourceFilter`/`filterItemsBySource` need new Set-based signatures; old single-value URL format needs a migration/fallback read if old bookmarks matter (confirm with planner — CONTEXT.md doesn't specify backward-compat for the old `?source=` param name). |
| Rendition sanitization/theming scattered per-plugin (pre-Phase-5) | Centralized in `kernel/httpapi/rendition.go` (D-11) | Phase 5 | This phase's highlighting must land in this same centralized file — do not reintroduce per-plugin sanitize/highlight logic. |

**Deprecated/outdated:** none specific to this phase's domain beyond the D-02 supersession above.

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|----------------|
| A1 | The recommended `?hl=` query-param name and mechanism for passing the highlight term to the kernel content route is a suggestion, not a locked contract — CONTEXT.md leaves the "mechanism per content variant" fully to Claude's Discretion. | Architecture Patterns / Pattern 3 | Low — this is explicitly flagged as discretionary in CONTEXT.md; the planner may choose a different param name or a POST-based approach. Only the *architectural conclusion* (kernel-side is required for iframe variants) is treated as fact, not the exact param shape. |
| A2 | Icon/visual treatment suggestions for UI-08 fidelity differentiation (e.g. a distinct icon or Badge variant for `conversation-only` vs `exact`/`anchored`) are illustrative, not researched against a specific icon name confirmed to exist in `@lucide/svelte`. | Don't Hand-Roll / general | Low — CONTEXT.md explicitly defers "wording, icon, treatment" to planning; no specific icon name is asserted as verified in this research. |
| A3 | The date-marker overlay for UI-11 is recommended as a client-side absolutely-positioned overlay computed from `StreamItem.timestamp_unix` (native `::-webkit-scrollbar` pseudo-elements cannot host arbitrary child markup) — this is general web-platform knowledge, not confirmed against an authoritative source this session (a websearch for prior art returned only generic CSS-timeline results, no canonical reference). | Common Pitfalls / Architecture (Recommended Project Structure) | Medium — if this assumption is wrong (e.g. if the target browser's `::-webkit-scrollbar-thumb` did support pseudo-content in some way this research missed), the planner would over-build a custom overlay where the native scrollbar sufficed. Low actual risk since `::-webkit-scrollbar-thumb` not supporting child content is a very long-standing, uncontroversial WebKit/Blink limitation — flagged as ASSUMED per the provenance rule rather than treated as CITED. |

## Open Questions

1. **Should `?sources=` replace `?source=` outright, or maintain backward compatibility with existing single-value bookmarks/deep links?**
   - What we know: CONTEXT.md D-02 says URL persistence "carries forward as a list (e.g. `?sources=home-email,work-email`)" — it does not say whether `?source=` (singular) should still be read as a fallback.
   - What's unclear: whether any existing bookmarked/shared deep links in the wild need to keep working.
   - Recommendation: Given this is a single-user, desktop-local tool with no external bookmark audience documented anywhere in PROJECT.md/STATE.md, treat this as low-stakes — the planner should pick one (rename outright is simpler) and note the choice; no research blocker either way.

2. **Does the `/agent/v1` rendition mirror accept the highlight query param?**
   - What we know: `agent.go:341` shares `sanitizeAndWrapRendition` with `item.go`'s human-facing route (Pitfall 3, confirmed this session); the agent surface has no search UI today (AGENT-10/11 are v1.x-deferred).
   - What's unclear: whether omitting highlight support there is a deliberate scope boundary or an oversight waiting to happen.
   - Recommendation: Pass no highlight term from the agent path (empty string / param absent) — the agent-facing route's job (AGENT-02: structured, programmatic consumption) has no rendering concern that highlighting serves; keep the function signature shared but the agent call site inert on this parameter.

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|----------------|---------|-------------------|
| V2 Authentication | No | Phase adds no auth surface — loopback-only, unauthenticated API is unchanged (`docs/api.md` "Loopback-only default, no auth", unaffected by this phase). |
| V3 Session Management | No | No session concept in this API. |
| V4 Access Control | No | No new access-control surface; the existing `/agent/v1` grant model (AGENT-01) is untouched except the Pitfall 3 call-site decision, which is a rendering choice, not an access decision. |
| V5 Input Validation | Yes | The kernel-side highlight term (from `?hl=` or equivalent) is untrusted request input reaching a text-mutation step over otherwise-sanitized HTML. Standard control: never treat the highlight term as markup — only ever insert it as a `<mark>` element's *matched substring of already-sanitized text*, constructed via the DOM tree API (`golang.org/x/net/html`), never string-concatenated into raw HTML bytes. This is the same discipline `noMatchesHeading` in `format.ts` already documents client-side ("Callers must render this through Svelte's default text binding... this function itself does no escaping because it produces plain text content, not markup" — `format.ts:324-333`, read this session) — the kernel-side equivalent is: build tree nodes, never format strings. |
| V6 Cryptography | No | No crypto surface in this phase. |

### Known Threat Patterns for this stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|----------------------|
| Search-term-as-markup injection into the highlighted rendition document | Tampering (of the served document's structure) | Tree-based node insertion only (Pattern 3) — never regex/string splicing of HTML bytes with user-supplied text. The existing sanitizer (`bluemonday`) already ran and produced trusted, well-formed output *before* this step; the highlighting step's obligation is to not reintroduce risk on top of that trusted output, not to re-sanitize it. |
| Weakening the `sandbox` CSP directive to enable client-side highlighting | Elevation of Privilege (of framed content relative to the parent SPA) | Do not add `allow-same-origin` or `allow-scripts` to the rendition route's CSP to work around the cross-origin `contentDocument` restriction (Pitfall 1) — that would reopen exactly the sandbox escape T-01-10/D-11 closed. The correct fix is kernel-side highlighting, not loosening the sandbox. |

## Sources

### Primary (HIGH confidence — read directly this session)
- `web/src/lib/components/SourceHealthChip.svelte`, `SourceFilterChips.svelte`, `WebspaceHeader.svelte`, `OpenInSource.svelte`, `DetailPane.svelte`, `StreamList.svelte`, `StreamRow.svelte` — full read
- `web/src/lib/format.ts`, `web/src/lib/api.ts` (relevant sections) — full/partial read
- `web/src/routes/w/[webspace]/+page.svelte` — full read
- `kernel/httpapi/item.go`, `kernel/httpapi/rendition.go`, `kernel/httpapi/search.go`, `kernel/httpapi/routes.go` — full read
- `kernel/item/item.go` (Fidelity enum), `kernel/index/store.go` (Search/ftsQuery/snippet delimiters) — relevant sections read
- `proto/topos/v1/plugin.proto` (LinkFidelity, ContentShape enums) — grepped and cross-checked against `kernel/item/item.go`
- `docs/api.md` (stream/search endpoint documentation) — relevant sections read
- `web/package.json`, `go.mod`, `go.sum` — read for dependency verification
- `web/node_modules/bits-ui/dist/bits/` directory listing — read for primitive availability
- `.planning/phases/02-two-sources-one-trustworthy-stream/02-UI-SPEC.md` — full read (copywriting/color contract carried forward)
- `.planning/phases/06-ui-scalable-source-surface/06-CONTEXT.md`, `.planning/REQUIREMENTS.md`, `.planning/STATE.md`, `.planning/config.json` — full read

### Secondary (MEDIUM confidence)
- MDN, "Content-Security-Policy: sandbox directive" — confirmed the opaque-origin behavior of a bare `sandbox` CSP token without `allow-same-origin` (via WebSearch this session, cross-referenced against the exact CSP header string read from `kernel/httpapi/item.go`)

### Tertiary (LOW confidence)
- General web-platform knowledge that native `::-webkit-scrollbar-thumb`/`-track` pseudo-elements cannot host arbitrary child markup (no canonical single source found this session — see Assumption A3)

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — no new packages; every dependency claim verified against `go.mod`/`go.sum`/`package.json`/`node_modules` directly.
- Architecture: HIGH — the load-bearing finding (iframe sandbox blocks client-side highlighting) is verified against the exact CSP header string in the live kernel code, cross-checked against MDN.
- Pitfalls: HIGH for Pitfalls 1-3 (each traced to a specific file/line read this session); MEDIUM for Pitfall 4 (T-02-17's rule is documented in `format.ts`'s own comments but its exact multi-value degrade behavior is a design carry-forward, not yet code).

**Research date:** 2026-08-06
**Valid until:** 30 days (stable in-repo domain; no fast-moving external dependency)
