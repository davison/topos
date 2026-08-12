# Phase 6: UI — Scalable Source Surface - Pattern Map

**Mapped:** 2026-08-06
**Files analyzed:** 10 (created/modified, per CONTEXT.md + RESEARCH.md)
**Analogs found:** 10 / 10 (all in-repo — this phase touches only existing surfaces, no genuinely new file category)

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|--------------------|------|-----------|-----------------|----------------|
| `web/src/lib/components/SourceChip.svelte` (NEW, absorbs `SourceHealthChip.svelte` + `SourceFilterChips.svelte`) | component | request-response (renders client-held state, calls `onrefresh`/`onfilter` callbacks) | `web/src/lib/components/SourceHealthChip.svelte` | exact (same role, superset of both existing chips' behavior) |
| `web/src/lib/components/WebspaceHeader.svelte` (modified) | component | request-response | itself (pre-existing file, modified in place) | exact |
| `web/src/lib/components/OpenInSource.svelte` (modified) | component | request-response | itself (pre-existing file, modified in place) | exact |
| `web/src/lib/components/DetailPane.svelte` (modified) | component | request-response | itself (pre-existing file, modified in place) | exact |
| `web/src/lib/components/StreamList.svelte` (modified, date-marker overlay) | component | transform (client-side derive from already-fetched data) | itself (pre-existing file, modified in place) | exact |
| `web/src/lib/format.ts` (modified — multi-select filter helpers, `highlightText()`) | utility | transform | itself; `parseSnippet` (lines 244-287) is the closest existing analog for the new `highlightText()` helper | exact for filter helpers (extend existing functions); role-match for `highlightText()` (models `parseSnippet`) |
| `web/src/lib/format.test.ts` (modified) | test | transform | itself (pre-existing colocated test file) | exact |
| `web/src/lib/components/sources.test.ts` (modified/renamed) | test | request-response | itself | exact |
| `kernel/httpapi/rendition.go` (modified — new `highlightTextNodes()` step) | service (kernel content-sanitization pipeline) | transform | `sanitizeAndWrapRendition` (same file, lines 293-309) — the function this step is inserted into | exact (new function added to existing pipeline file) |
| `kernel/httpapi/item.go` (modified — `?hl=` query param threading) | controller (HTTP handler) | request-response | `renditionHandler` (same file, lines 153-244) | exact |
| `kernel/httpapi/agent.go` (decision point only, `sanitizeAndWrapRendition` call site at line 341) | controller (HTTP handler) | request-response | `kernel/httpapi/item.go`'s `renditionHandler` call site | role-match — same function, second caller, must be updated in lockstep for signature changes |
| `kernel/httpapi/rendition_test.go` (extended) | test | transform | itself (pre-existing test file with `TestChatRenditionPolicy...` tests) | exact |

## Pattern Assignments

### `web/src/lib/components/SourceChip.svelte` (component, request-response) — UI-07

**Analog:** `web/src/lib/components/SourceHealthChip.svelte` (dot/tooltip/refresh half) + `web/src/lib/components/SourceFilterChips.svelte` (filter/selection half)

**Imports pattern** (`SourceHealthChip.svelte` lines 1-12):
```svelte
import { Button } from '$lib/components/ui/button/index.js';
import {
	Tooltip,
	TooltipContent,
	TooltipProvider,
	TooltipTrigger
} from '$lib/components/ui/tooltip/index.js';
import RefreshCw from '@lucide/svelte/icons/refresh-cw';
import { cn } from '$lib/utils.js';
import { healthTone, formatRelativeTime, type HealthTone } from '$lib/format';
import type { SourceStatus } from '$lib/api';
```
The merged chip keeps this exact import set and adds nothing new (no new npm package per RESEARCH.md) — if the overflow/collapse treatment (Claude's Discretion, UI-07) needs a dropdown/popover, add `bits-ui`-backed primitives from `web/src/lib/components/ui/` (already installed, per RESEARCH.md's Don't Hand-Roll table), following the same relative-import shape used for `Tooltip`/`Button` above.

**Dot-tone mapping / tooltip copy pattern** (`SourceHealthChip.svelte` lines 21-46):
```svelte
let tone = $derived(healthTone(source));

const DOT_TONE_CLASS: Record<HealthTone, string> = {
	success: 'bg-success',
	warning: 'bg-warning',
	destructive: 'bg-destructive',
	unknown: 'bg-muted-foreground'
};

let tooltipText = $derived.by(() => {
	const relative = formatRelativeTime(source.last_sync_unix);
	switch (tone) {
		case 'success':
			return `${source.display_name} — synced ${relative} ago`;
		case 'warning':
			return `${source.display_name} — last error ${relative} ago: ${source.last_error}`;
		case 'destructive':
			return `${source.display_name} — unreachable since ${relative}`;
		default:
			return `${source.display_name} — not yet synced`;
	}
});
```
Copy verbatim into the merged chip — D-04 explicitly carries forward this tooltip copy contract (02-UI-SPEC.md rows) unchanged; only the trigger markup around it changes (whole chip becomes clickable for filter-toggle per D-01, not just wrapping the dot+name span).

**Click-to-filter selection pattern** (`SourceFilterChips.svelte` lines 12-44, `Button` `variant`/`aria-pressed` toggle):
```svelte
let {
	sources,
	selectedSource,
	onfilter
}: {
	sources: SourceStatus[];
	selectedSource: string | null;
	onfilter: (source: string | null) => void;
} = $props();
```
```svelte
<Button
	variant={selectedSource === source.name ? 'default' : 'outline'}
	size="sm"
	aria-pressed={selectedSource === source.name}
	onclick={() => onfilter(source.name)}
	class="max-w-40"
	title={source.display_name}
>
	<span class="truncate">{source.display_name}</span>
</Button>
```
Per D-02, `selectedSource: string | null` becomes a `Set<string>` (or equivalent membership check) — `aria-pressed` becomes `selectedSources.has(source.name)`, `onfilter(source.name)` becomes a toggle call (`onfilter(source.name)` adds/removes from the set), and there is no dedicated "All" chip in the merged design (D-02: "all-off = show everything").

**Refresh control, hover/focus reveal** (`SourceHealthChip.svelte` lines 73-82 is the always-visible baseline to adapt for D-03's hover/focus-reveal requirement):
```svelte
<Button
	variant="ghost"
	size="icon"
	class="size-11"
	disabled={source.syncing}
	aria-label={`Refresh ${source.display_name}`}
	onclick={() => onrefresh(source.name)}
>
	<RefreshCw class={cn('size-4', source.syncing && 'animate-spin')} />
</Button>
```
D-03 requires this button to be hidden at rest and revealed on chip `:hover`/`:focus-within` (CSS opacity/visibility toggle keyed off a wrapping element's hover/focus state — no new component needed, a Tailwind `opacity-0 group-hover:opacity-100 group-focus-within:opacity-100` pattern on the existing button is sufficient) except while `source.syncing` is true, when the spinning icon must stay visible regardless of hover state (D-03: "remains visible as the spinner while syncing").

---

### `web/src/lib/components/WebspaceHeader.svelte` (component, request-response) — UI-07 integration

**Analog:** itself (pre-existing, being rewired)

**Row-gating pattern to preserve** (lines 38, 52-62):
```svelte
let showSourceRows = $derived(shouldShowSourceRows(sourcesState, sources));
...
{#if showSourceRows}
	<div class="mt-4 flex flex-wrap items-center gap-2">
		{#each sources as source (source.name)}
			<SourceHealthChip {source} {onrefresh} />
		{/each}
		<Button variant="outline" size="sm" onclick={onrefreshall}>Refresh all</Button>
	</div>
	<div class="mt-3">
		<SourceFilterChips {sources} {selectedSource} {onfilter} />
	</div>
{/if}
```
Replace the two `{#each}`/child-component rows with a single row rendering `SourceChip` per source, still gated by the same `shouldShowSourceRows` derived value (CONTEXT.md/RESEARCH.md both flag this gating as must-preserve — a non-critical sources failure must never blank the stream). `selectedSource: string | null` prop becomes `selectedSources: Set<string>` (or similar) threaded down from `+page.svelte`. "Refresh all" placement is Claude's Discretion (CONTEXT.md) — keep it as a sibling `Button` in the same row per the existing pattern unless collapse/overflow forces it elsewhere.

---

### `web/src/lib/components/OpenInSource.svelte` (component, request-response) — UI-08

**Analog:** itself (pre-existing, being extended)

**Current fidelity Badge pattern** (lines 12-22, 24-35):
```svelte
let { link, displayName }: { link: Link; displayName: string } = $props();

const fidelityLabel: Record<string, string> = {
	exact: 'exact',
	anchored: 'anchored',
	'conversation-only': 'conversation-only'
};
```
```svelte
<div class="flex items-center gap-2">
	<Button href={link.url} target="_blank" rel="noopener noreferrer" class="min-h-11 max-w-64" title={`Open in ${displayName}`}>
		<span class="truncate">Open in {displayName}</span>
	</Button>
	<Badge variant="secondary">{fidelityLabel[link.fidelity] ?? link.fidelity}</Badge>
</div>
```
UI-08 differentiates the `conversation-only` case (raise-window-only, no deep navigation) from `exact`/`anchored` (real deep links) — the change is confined to this Badge/label block and possibly the button's own copy/icon for `conversation-only` (e.g. "Open Signal" vs "Open in {displayName}" plus a distinct Badge `variant` or icon). No prop-shape change is required (`link.fidelity` is already present, per PLUG-03/RESEARCH.md) — this is a pure rendering branch inside the existing component, keyed on `link.fidelity === 'conversation-only'`.

---

### `web/src/lib/components/DetailPane.svelte` (component, request-response) — UI-09

**Analog:** itself (pre-existing, being extended with a new `searchQuery` prop and highlight rendering)

**Prop shape to extend** (lines 18-22):
```svelte
let {
	item,
	displayName,
	sourceReachable
}: { item: StreamItem; displayName: string; sourceReachable: boolean } = $props();
```
Add `searchQuery: string` here (RESEARCH.md Pattern 2: "`DetailPane.svelte` currently receives no `searchQuery` prop... `+page.svelte` must pass its existing `searchQuery` state down").

**Plain-text body block to highlight client-side** (lines 86-92, the `loadedTextBlock` snippet):
```svelte
{#snippet loadedTextBlock()}
	<div class="min-h-0 flex-1 overflow-y-auto text-[16px] leading-[1.5] whitespace-pre-wrap text-foreground">
		{content?.text}
	</div>
{/snippet}
```
Replace the raw `{content?.text}` text binding with `{#each highlightText(content?.text ?? '', searchQuery) as segment}` rendering each `SnippetSegment` — mirror `StreamRow.svelte`'s existing `{#each parseSnippet(snippet) as segment, i (i)}` render pattern (referenced in RESEARCH.md line 246) so the `<mark>`/plain-span rendering shape is identical between the stream row snippets and this detail-pane block.

**Iframe body block — kernel-side highlighting only** (lines 144-160, the `html` branch):
```svelte
<div class="min-h-0 flex-1 overflow-hidden rounded-lg border border-border bg-card">
	<iframe title={item.title} src={contentUrl(item.id)} class="h-full w-full"></iframe>
</div>
```
Per RESEARCH.md's load-bearing finding, this branch (and the PDF sub-branch of `media`, line 165, though PDFs have no highlightable text layer via this mechanism) cannot be highlighted client-side — `contentUrl(item.id)` must gain an optional highlight query param (e.g. `contentUrl(item.id, searchQuery)`) so the browser requests an already-highlighted document from the kernel. Do not attempt `iframe.contentDocument` access (RESEARCH.md Pitfall 1 / Anti-Patterns — blocked by the `sandbox` CSP with no `allow-same-origin`).

---

### `web/src/lib/lib/format.ts` — multi-select filter + `highlightText()` — UI-07/UI-09

**Analog for filter-set extension:** existing `resolveSourceFilter` / `filterItemsBySource` (lines 120-129):
```typescript
export function resolveSourceFilter(requested: string | null, sources: SourceStatus[]): string | null {
	if (!requested) return null;
	return sources.some((s) => s.name === requested) ? requested : null;
}

export function filterItemsBySource(items: StreamItem[], source: string | null): StreamItem[] {
	if (source === null) return items;
	return items.filter((item) => item.source === source);
}
```
D-02's multi-select rewrite: `requested: string | null` → `requested: string[]` (or `Set<string>`), returning a `Set<string>` filtered per-member (T-02-17's "degrade unrecognised values silently" rule applies per-value, not all-or-nothing — see RESEARCH.md Pitfall 4). `filterItemsBySource` takes the set; empty set = return `items` unchanged (D-02: "all-off = show everything").

**Analog for `highlightText()`:** existing `parseSnippet` (lines 244-287) and its `SnippetSegment` type:
```typescript
export interface SnippetSegment {
	text: string;
	match: boolean;
}

export function parseSnippet(snippet: string): SnippetSegment[] {
	// splits a kernel-delimited (\x02/\x03) string into ordered
	// plain/matched segments, degrading to one plain-text segment on any
	// malformed delimiter run rather than throwing (T-03-22)
}
```
New `highlightText(text: string, query: string): SnippetSegment[]` returns the identical `SnippetSegment[]` shape (RESEARCH.md: "so `StreamRow.svelte`'s existing render pattern... can be reused verbatim in `DetailPane.svelte`") — but computes matches via case-insensitive substring scan (indexOf-based, never a raw `new RegExp(query)` per RESEARCH.md's injection warning) against whitespace-split terms of `query`, mirroring the kernel's own `ftsQuery` tokenization (`kernel/index/store.go:375-390`, `strings.Fields`-equivalent).

**Analog for output-safety discipline (V5 input validation):** `noMatchesHeading` (lines 331-333) and its doc comment:
```typescript
export function noMatchesHeading(query: string): string {
	return `No matches for "${query}"`;
}
// "Callers must render this through Svelte's default text binding... this
// function itself does no escaping because it produces plain text
// content, not markup"
```
`highlightText()` must follow the same discipline — it returns data (`SnippetSegment[]`), never a raw HTML string, so Svelte's default text-binding escaping in the `{#each}` render handles safety automatically (no `{@html}` anywhere in this path).

---

### `web/src/lib/components/StreamList.svelte` — date-marker overlay — UI-11

**Analog:** itself (pre-existing, being extended with an overlay alongside the existing `{#each visibleItems}` list)

**Existing derive-and-render pattern to extend** (lines 36-41, 62-78):
```svelte
let variant = $derived(response ? streamVariant(response, selectedSource) : null);
let visibleItems = $derived(response ? filterItemsBySource(response.items, selectedSource) : []);
```
```svelte
<div class="flex flex-col gap-3">
	{#each visibleItems as item (item.id)}
		<StreamRow ... />
	{/each}
</div>
```
UI-11's date markers are a new `$derived` computed from `visibleItems`' `timestamp_unix` values (client-side, no new fetch — RESEARCH.md: "computable client-side from `StreamItem.timestamp_unix`, which the stream response already carries"), rendered as an absolutely-positioned overlay sibling to the scroll container (RESEARCH.md Assumption A3: native `::-webkit-scrollbar` pseudo-elements can't host markup, so this must be a custom overlay, not a scrollbar-track pseudo-element). Base scrollbar theming (`--scrollbar-thumb` tokens, `web/src/app.css`, quick task 260805-j98) is unchanged — this phase only adds the marker overlay.

---

### `kernel/httpapi/rendition.go` — `highlightTextNodes()` step — UI-09

**Analog:** `sanitizeAndWrapRendition` itself (lines 293-309+, the function the new step is inserted into):
```go
func sanitizeAndWrapRendition(shape toposv1.ContentShape, fragment []byte) ([]byte, error) {
	policy, ok := renditionPolicies[shape]
	if !ok {
		return nil, fmt.Errorf("%w: %v", errUnrecognisedContentShape, shape)
	}

	sanitized := policy.SanitizeBytes(fragment)
	style := stylesheetForShape(shape)

	var buf bytes.Buffer
	buf.WriteString("<!doctype html>\n<html><head><meta charset=\"utf-8\"><style>")
	buf.WriteString(style)
	// ... (continues: </style></head><body>, sanitized bytes, </body></html>)
}
```
**Insertion point (documented as load-bearing by the function's own doc comment):** the new `highlightTextNodes(sanitized, terms)` call goes strictly between `sanitized := policy.SanitizeBytes(fragment)` and the `buf.Write(sanitized)` step — never re-entering `policy.SanitizeBytes`. Signature likely becomes `sanitizeAndWrapRendition(shape toposv1.ContentShape, fragment []byte, highlightTerms []string) ([]byte, error)` (both call sites — `item.go:202` and `agent.go:341` — need updating; see below).

**Tree-walk pattern (new, no existing analog in-repo — `golang.org/x/net/html` idiom, already a transitive dependency via bluemonday per `go.mod`/`go.sum`):**
```go
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
Insert `<mark>` as a bare element node (no class/attributes — style via a new `mark { ... }` rule added to `stylesheetForShape`'s per-shape delta constants, e.g. `renditionProseDelta`/`renditionChatDelta`), never as string-concatenated HTML (RESEARCH.md Pitfall 2 / V5 Input Validation: "build tree nodes, never format strings").

---

### `kernel/httpapi/item.go` — `?hl=` query param threading — UI-09

**Analog:** `renditionHandler` itself (lines 153-209, the call site being modified):
```go
func renditionHandler(store *index.Store, fetcher Fetcher, variant toposv1.ContentVariant) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		...
		if result.MimeType == "text/html" {
			fragment, err := io.ReadAll(result.Body)
			...
			wrapped, err := sanitizeAndWrapRendition(result.ContentShape, fragment)
			...
			body = wrapped
		}
		...
		h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; object-src 'none'; sandbox")
		...
	}
}
```
Add `hl := r.URL.Query().Get("hl")` (tokenize via whitespace-split, mirroring `ftsQuery`), thread as an extra argument into `sanitizeAndWrapRendition(result.ContentShape, fragment, terms)`. The CSP header (`sandbox`, no `allow-same-origin`) is explicitly untouchable per CONTEXT.md — this change never modifies the header-setting block quoted above, only the `body` construction above it.

**MIME/allowlist discipline to preserve** (lines 180-186, unrelated to this change but must not regress):
```go
if !allowedRenditionTypes[result.MimeType] {
	WriteError(w, http.StatusUnsupportedMediaType, "unsupported_rendition_type", ...)
	return
}
```

---

### `kernel/httpapi/agent.go` — second call site decision — UI-09 (Pitfall 3)

**Analog:** `kernel/httpapi/item.go`'s `renditionHandler` call site (line 202) — `agent.go:341` calls the same `sanitizeAndWrapRendition` and must compile against whatever new signature is chosen. RESEARCH.md's Open Question #2 recommends passing an empty highlight-term list from the agent path (no search UI on `/agent/v1`) while keeping the function signature shared.

---

## Shared Patterns

### Sanitizer/CSP untouchable boundary
**Source:** `kernel/httpapi/item.go` lines 235 (CSP header), `kernel/httpapi/rendition.go` lines 277-292 (`sanitizeAndWrapRendition` doc comment)
**Apply to:** `rendition.go`, `item.go`, `agent.go` — any UI-09 kernel-side change
```go
h.Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; object-src 'none'; sandbox")
```
Never add `allow-same-origin`/`allow-scripts`; never re-run `policy.SanitizeBytes` on wrapped/highlighted output.

### Copywriting/tooltip contract carry-forward
**Source:** `web/src/lib/components/SourceHealthChip.svelte` lines 34-46 (tooltip text switch), `.planning/phases/02-two-sources-one-trustworthy-stream/02-UI-SPEC.md`
**Apply to:** `SourceChip.svelte`
Reuse the exact tooltip copy strings verbatim per D-04 — no rewording.

### URL-persisted filter state, per-value degrade
**Source:** `web/src/lib/format.ts` lines 120-123 (`resolveSourceFilter`), `web/src/routes/w/[webspace]/+page.svelte` (owns `selectedSource`/URL sync, not yet re-read this session but referenced in RESEARCH.md as the integration point)
**Apply to:** `format.ts`, `WebspaceHeader.svelte`, `SourceChip.svelte`, the `+page.svelte` route
Multi-select set must degrade per-member (drop unrecognised instance ids one at a time), never all-or-nothing (RESEARCH.md Pitfall 4 / T-02-17).

### `SnippetSegment`-shaped highlight rendering
**Source:** `web/src/lib/format.ts` lines 229-243 (`SnippetSegment` type + doc comment), `StreamRow.svelte` line 119 (`{#each parseSnippet(snippet) as segment, i (i)}`)
**Apply to:** `format.ts` (`highlightText()`), `DetailPane.svelte` (`loadedTextBlock` snippet)
New client-side highlight helper returns the same `SnippetSegment[]` shape so the render loop is copy-pasteable between `StreamRow.svelte` and `DetailPane.svelte`.

## No Analog Found

None — every file this phase touches is a modification of an existing component/handler, or a new component (`SourceChip.svelte`) that is a direct merge of two existing, fully-read analogs. The one genuinely novel mechanism (kernel-side `<mark>` tree-walk highlighting) has no in-repo analog but RESEARCH.md already supplies a complete illustrative sketch (Pattern 3) sourced from the standard `golang.org/x/net/html` idiom, which is sufficient for the planner to scope a plan around.

## Metadata

**Analog search scope:** `web/src/lib/components/`, `web/src/lib/format.ts`, `web/src/lib/api.ts`, `web/src/routes/w/`, `kernel/httpapi/`
**Files scanned:** `SourceHealthChip.svelte`, `SourceFilterChips.svelte`, `WebspaceHeader.svelte`, `OpenInSource.svelte`, `DetailPane.svelte`, `StreamList.svelte`, `format.ts`, `api.ts`, `kernel/httpapi/item.go`, `kernel/httpapi/rendition.go` (all read directly this session, supplemented by RESEARCH.md's own direct reads of the same files plus `store.go`, `agent.go`, `rendition_test.go`)
**Pattern extraction date:** 2026-08-06
```
