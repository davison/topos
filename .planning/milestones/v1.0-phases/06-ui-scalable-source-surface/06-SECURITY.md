---
phase: 06
slug: ui-scalable-source-surface
status: verified
# threats_open = count of OPEN threats at or above workflow.security_block_on severity (the blocking gate)
threats_open: 0
asvs_level: 1
created: 2026-08-07
---

# Phase 06 — Security

> Per-phase security contract: threat register, accepted risks, and audit trail.
> Register authored at plan time across all eight plans (06-01 … 06-08); consolidated here at
> phase close. Where plans reused a threat ID for different concerns, the originating plan is
> given in parentheses.

---

## Trust Boundaries

| Boundary | Description | Data Crossing |
|----------|-------------|---------------|
| browser → kernel HTTP (`GET /api/items/{id}/content?hl=`) | Highlight term is untrusted request input reaching the sanitized-HTML mutation step | User search terms |
| kernel → sandboxed iframe document | Served rendition crosses into an opaque-origin document; sanitizer output is the trust anchor | Rendered item content |
| plugin → kernel (`Fetch` rendition fragment) | Plugin fragments are untrusted and sanitized before use | Source item content |
| URL query → SPA filter state | `sources` query value (shared link, stale bookmark, hand-edited URL) selects the visible view | Filter selection |
| kernel → SPA (`GET /api/sources`) | Source display names and health fields rendered into header chrome | User-authored config values |
| user config → SPA render | `display_name` from `config.toml` rendered in chip label and native `title` | Operator config |
| user intent → filter state | The chip is the sole control and sole indicator of stream narrowing | UI state claims |
| plugin `link.url` / `link.fidelity` → browser | Deep links opened in new contexts; fidelity enum drives UI claims about click behaviour | Plugin-declared links |
| stream item timestamps / ids → overlay geometry & selectors | Plugin-supplied values drive pixel positions and a CSS attribute selector | Item metadata |
| browser layout engine → SPA measurement | ResizeObserver/scrollHeight callbacks drive reactive renders — feedback-loop availability risk | Element geometry |

---

## Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation | Status |
|-----------|----------|-----------|----------|-------------|------------|--------|
| T-06-01 (06-01) | Tampering | `highlightTextNodes` in `kernel/httpapi/rendition.go` | high | mitigate | `<mark>` inserted as tree nodes via `golang.org/x/net/html` into `html.TextNode` only, never byte substitution — verified: `rendition.go:383,451`; `rendition_test.go` asserts attributes/tags untouched, multi-byte runes survive | closed |
| T-06-02 (06-01) | Elevation of Privilege | rendition CSP in `kernel/httpapi/item.go` / `agent.go` | critical | mitigate | No same-origin sandbox token anywhere in `kernel/httpapi/` — verified by grep (clean) | closed |
| T-06-03 (06-01) | Denial of Service | `highlightTerms` derivation | medium | mitigate | Whitespace-split, de-duped, 2-rune floor, capped at 8 terms — verified `rendition.go:317,335` | closed |
| T-06-04 (06-01) | Information Disclosure | `hl` query param on loopback content route | low | accept | Loopback-only single-user API; `Cache-Control: private, no-store`; param not logged | closed |
| T-06-05 (06-01) | Repudiation | n/a | low | accept | Single-user read-only viewer; no actor model, nothing to repudiate | closed |
| T-06-SC (06-01) | Tampering | Go module installs | high | mitigate | No install ran; `golang.org/x/net v0.53.0` promoted from existing indirect via targeted `go get` — verified in `go.mod:23` | closed |
| T-06-06 (06-02) | Tampering | `resolveSourceFilters` in `web/src/lib/format.ts` | low | mitigate | Query resolved per-member against configured sources, unknown members dropped, never reaches a request/DOM sink — verified `format.ts:162` + partially-stale-case test | closed |
| T-06-07 (06-02/06-04) | Spoofing | overflow trigger in `WebspaceHeader.svelte` | medium | mitigate | `worstHealthTone` over the hidden set always visible on the row; live overflow recomputation restored by 06-04's `observeResize` wiring | closed |
| T-06-08 (06-02) | Spoofing | selected-chip styling | medium | mitigate | Superseded by G-06-3/G-06-3b closures: solid `bg-primary` fill + re-toned children (06-06), full-height hit area (06-08); `Clear filters` control when selection non-empty; guarded by `source-chip-selected.test.ts` + `source-chip-pill.test.ts` | closed |
| T-06-09 (06-02) | Information Disclosure | `display_name` in chip text/titles/tooltips | low | accept | Svelte default text binding (escaped); no raw-HTML directive in path; local single-user content | closed |
| T-06-10 (06-02) | Denial of Service | `ResizeObserver` measurement in `WebspaceHeader.svelte` | low | mitigate | Callback writes state only on change, no synchronous re-measure; made load-bearing and re-verified in 06-04 (see T-06-11 (06-04)) | closed |
| T-06-SC (06-02) | Tampering | npm installs | high | mitigate | No install; popover primitive wrapped from already-installed `bits-ui`; task gate failed on lockfile change | closed |
| T-06-11 (06-03) | Spoofing | `fidelityAffordance` / `OpenInSource.svelte` | medium | mitigate | Window-only class gets own verb/icon/title; unrecognised-value fallback unit-tested (`fidelity.test.ts`) | closed |
| T-06-12 (06-03) | Tampering | `link.url` as button href | low | accept | New-context target with no-opener/no-referrer preserved; plugin-supplied local config on single-user machine | closed |
| T-06-13 (06-03) | Denial of Service | `dateMarkers` over long streams | low | mitigate | Single linear pass, ≤3 thinning passes, 24px floor caps marker count by pane height | closed |
| T-06-14 (06-03) | Tampering | marker overlay stacking | medium | mitigate | Container `pointer-events-none`; only tick hit-areas opt back in — re-verified after 06-07 rebuild (`marker-overlay.test.ts`) | closed |
| T-06-15 (06-03) | Information Disclosure | marker tooltip content | low | accept | Renders only a formatted date already visible in stream rows | closed |
| T-06-11 (06-04) | Denial of Service | live observer callback in `WebspaceHeader.svelte` | medium | mitigate | Write-on-change only; no synchronous re-measure; no measured value feeds back into a measured width; row `overflow-hidden` — verified `WebspaceHeader.svelte:189` + `resize-observer.test.ts` | closed |
| T-06-12 (06-04) | Denial of Service | observer lifetime in `web/src/lib/resize-observer.ts` | low | mitigate | Idempotent disconnect returned as effect cleanup — verified `resize-observer.ts:15,32,57` + guard tests | closed |
| T-06-09 (06-04) | Information Disclosure | element refs passed into helper | low | accept | Helper dereferences nothing, reads no attribute/text, logs nothing | closed |
| T-06-SC (06-04) | Tampering | npm/pip/cargo installs | high | mitigate | No install; guard reuses ambient `node:*` type declarations; task gate on lockfile change | closed |
| T-06-11 (06-05) | Tampering | `DetailPane`/`StreamRow` title & snippet render | high | mitigate | All match segments render through Svelte text bindings over `SnippetSegment[]`; `highlightText` returns plain strings; no `{@html}` on any highlight surface — verified by grep (clean) + `search-emphasis.test.ts` guard | closed |
| T-06-12 (06-05) | Denial of Service | `highlightText` per stream row | low | accept | Query capped at 8 terms × 64 chars; ≤50 in-memory titles — bounded | closed |
| T-06-13 (06-05) | Information Disclosure | global `.search-highlight` class in `app.css` | low | accept | Colour-only declarations; single-declaration guard bounds collision surface | closed |
| T-06-14 (06-06) | Spoofing | `SourceChip.svelte` selected-state affordance | medium | mitigate | Solid fill + re-toned children + `aria-pressed`; recurrence guard `source-chip-selected.test.ts` — UAT round 3 test 3 passed | closed |
| T-06-15 (06-06) | Tampering | `display_name` in chip label/tooltip | low | accept | Svelte default text binding, operator's own local config | closed |
| T-06-16 (06-06) | Denial of Service | chip row `overflow-hidden` clip removal risk | medium | mitigate | Fill chosen (paints inside border box, no clip change); clip still present — verified `WebspaceHeader.svelte:189` | closed |
| T-06-17 (06-07) | Tampering | `scrollToMarker` querySelector over plugin item id | medium | mitigate | `CSS.escape` wraps id before selector interpolation; missing-container/target are no-ops — verified `StreamDateMarkers.svelte:78` + guard | closed |
| T-06-18 (06-07) | Denial of Service | `scrollHeight` measurement via `observeResize` | medium | mitigate | Write-on-change, no synchronous re-measure, idempotent teardown — same bounded pattern as chip row | closed |
| T-06-19 (06-07) | Denial of Service | overlay hit-areas over native scrollbar | medium | mitigate | Lane offset ≥ declared scrollbar width; hit-areas widen leftward only; container `pointer-events-none`; guard derives offset from `app.css` | closed |
| T-06-20 (06-07) | Information Disclosure | date ruler exposes temporal shape | low | accept | Data already on screen; single-user local-only tool | closed |
| T-06-17 (06-08) | Spoofing | `SourceChip.svelte` filter hit area | medium | mitigate | Filter button `self-stretch` across full pill height, no wrapper padding bands; negative padding assertions in `source-chip-pill.test.ts` — UAT round 3 test 1 passed | closed |
| T-06-18 (06-08) | Information Disclosure | refresh reveal pinned by mouse-click focus | low | mitigate | Reveal scoped to `group-has-[:focus-visible]`; focus-within form asserted absent | closed |
| T-06-19 (06-08) | Denial of Service | keyboard reachability of refresh control | medium | mitigate | Compiled `:has(:focus-visible)` rule verified present in production stylesheet (`kernel/webui/build/_app/immutable/assets/0.C4b0x9me.css`) — cannot pass vacuously | closed |
| T-06-20 (06-08) | Tampering | `display_name` in chip label/tooltip | low | accept | Svelte default text binding, unchanged interpolation path | closed |

*Status: open · closed · open — below high threshold (non-blocking)*
*Severity: critical > high > medium > low — only open threats at or above workflow.security_block_on count toward threats_open*
*Disposition: mitigate (implementation required) · accept (documented risk) · transfer (third-party)*

---

## Accepted Risks Log

| Risk ID | Threat Ref | Rationale | Accepted By | Date |
|---------|------------|-----------|-------------|------|
| AR-06-01 | T-06-04 (06-01) | Loopback-only unauthenticated API by design; user's own query on own machine; no-store cache policy | plan-time register (06-01) | 2026-08-07 |
| AR-06-02 | T-06-05 (06-01) | Single-user read-only viewer, no actor model or audit requirement | plan-time register (06-01) | 2026-08-07 |
| AR-06-03 | T-06-09 (06-02) | `display_name` escaped by Svelte text binding; local single-user content | plan-time register (06-02) | 2026-08-07 |
| AR-06-04 | T-06-12 (06-03) | Href/relationship attributes preserved unchanged from Phase 1 (no-opener/no-referrer, new context) | plan-time register (06-03) | 2026-08-07 |
| AR-06-05 | T-06-15 (06-03) | Tooltip renders only already-visible formatted date | plan-time register (06-03) | 2026-08-07 |
| AR-06-06 | T-06-09 (06-04) | Opaque element refs; nothing dereferenced, read, or logged | plan-time register (06-04) | 2026-08-07 |
| AR-06-07 | T-06-12 (06-05) | Bounded work: 8-term/64-char query cap over ≤50 in-memory titles | plan-time register (06-05) | 2026-08-07 |
| AR-06-08 | T-06-13 (06-05) | Colour-only global class; single-declaration guard bounds collision | plan-time register (06-05) | 2026-08-07 |
| AR-06-09 | T-06-15 (06-06) / T-06-20 (06-08) | `display_name` render path unchanged; operator's own config trust level | plan-time registers (06-06, 06-08) | 2026-08-07 |
| AR-06-10 | T-06-20 (06-07) | Ruler annotates data already on screen; local-only single-user tool | plan-time register (06-07) | 2026-08-07 |

*Accepted risks do not resurface in future audit runs.*

---

## Security Audit Trail

| Audit Date | Threats Total | Closed | Open | Run By |
|------------|---------------|--------|------|--------|
| 2026-08-07 | 35 | 35 | 0 | gsd-secure-phase orchestrator (L1 grep-depth, short-circuit: plan-time register, ASVS 1) |

---

## Sign-Off

- [x] All threats have a disposition (mitigate / accept / transfer)
- [x] Accepted risks documented in Accepted Risks Log
- [x] `threats_open: 0` confirmed
- [x] `status: verified` set in frontmatter

**Approval:** verified 2026-08-07
