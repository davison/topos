// rendition.go is the kernel-owned content-serving boundary D-11
// consolidates into: plugins used to each carry their own near-identical
// sanitize policy, theme stylesheet, and document-wrapping helper
// (the proton, silverbullet and signal plugins' own render helpers,
// before this move — those plugins live in davison/topos-plugins now) — a theme change meant
// editing three plugins, and once plugins are third-party, sanitization
// sat outside the trust boundary entirely.
//
// After this file, a plugin returns content plus a declared
// toposv1.ContentShape and the kernel sanitizes, wraps and themes every
// text/html rendition from this one place (kernel/httpapi/item.go's
// renditionHandler is the only caller). Sanitization always runs BEFORE
// wrapping, and the wrapped output is never fed back through a sanitizer
// — see sanitizeAndWrapRendition's own doc comment.
package httpapi

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

// errUnrecognisedContentShape is returned by sanitizeAndWrapRendition when
// shape has no entry in renditionPolicies — including the zero value,
// CONTENT_SHAPE_UNSPECIFIED (T-05-16). The caller (renditionHandler) must
// write no body when this error is returned: the kernel fails closed
// rather than ever guessing a policy for an undeclared shape.
var errUnrecognisedContentShape = fmt.Errorf("httpapi: unrecognised or unspecified content shape")

// renditionStyledElements is the named set of block and inline elements the
// email content-shape policy permits to carry a sanitized style attribute —
// carried forward verbatim from the proton plugin's styledElements
// (body.go, at the D-11 move; the plugin lives in topos-plugins).
// bluemonday's own published HTML-email example allows the style attribute
// Globally(), and its own comment says that is "not safe" — this policy
// deliberately does not copy that; scoping the attribute to a named element
// set (never Globally()) is the residual-risk tradeoff 03-RESEARCH.md
// Pitfall 3 called for and D-11 preserves unchanged.
var renditionStyledElements = []string{"p", "span", "div", "td", "th", "h1", "h2", "h3", "h4", "h5", "h6", "li", "a"}

// chatTranscriptClassTokens is the fixed, closed set of class tokens the
// chat content-shape policy allows on a div element — every token the
// signal plugin's renderTranscript/renderBubble ever emitted at the
// D-11 move (the plugin lives in topos-plugins), and
// nothing else (T-05-17). bluemonday's Matching regexp is evaluated against
// a class attribute's ENTIRE value (not per space-separated token), so this
// pattern accepts any sequence of one or more of these tokens separated by
// single spaces — exactly the "<one or two tokens>" shapes renderTranscript
// produces (e.g. "run own", "bubble other", "sender-name") — and rejects
// (strips the whole class attribute for) anything else, including a
// legitimate token mixed with a forged one.
var chatTranscriptClassTokens = regexp.MustCompile(
	`^(?:run|own|other|sender-name|bubble|tombstone|quote|timestamp|edited-suffix|attachment|reaction|body)` +
		`(?: (?:run|own|other|sender-name|bubble|tombstone|quote|timestamp|edited-suffix|attachment|reaction|body))*$`,
)

// renditionPolicies maps each recognised toposv1.ContentShape to its
// bluemonday.Policy, built once at package init (bluemonday's own
// documented pattern for concurrent-safe reuse — each Policy builds its
// allowlist tables once and is then read-only). CONTENT_SHAPE_UNSPECIFIED
// deliberately has no entry: a lookup miss is exactly how
// sanitizeAndWrapRendition fails closed on an unrecognised or unspecified
// shape.
var renditionPolicies = map[toposv1.ContentShape]*bluemonday.Policy{
	toposv1.ContentShape_CONTENT_SHAPE_EMAIL_HTML:      newEmailRenditionPolicy(),
	toposv1.ContentShape_CONTENT_SHAPE_MARKDOWN_HTML:   bluemonday.UGCPolicy(),
	toposv1.ContentShape_CONTENT_SHAPE_CHAT_TRANSCRIPT: newChatRenditionPolicy(),
}

// newEmailRenditionPolicy widens bluemonday.UGCPolicy() exactly as the
// proton plugin's newEmailSanitizePolicy did at the D-11 move: the style attribute
// is allowed only on renditionStyledElements, and only a presentational
// CSS-property allowlist is permitted through it. Everything positional or
// behavioural (position, z-index, transform, animation, transition,
// content, clip, filter, mix-blend-mode, pointer-events) stays denied by
// omission. Safe because the kernel's rendition route
// (kernel/httpapi/item.go) already serves this document under
// `Content-Security-Policy: default-src 'none'; ...; sandbox`, so no CSS
// declaration this policy lets through can issue a network request — which
// also defeats email tracking pixels for free.
func newEmailRenditionPolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()

	p.AllowAttrs("style").OnElements(renditionStyledElements...)

	p.AllowStyles(
		"color", "background-color", "font-weight", "font-style", "font-size",
		"font-family", "text-align", "text-decoration", "padding", "margin",
		"border", "width", "height",
	).OnElements(renditionStyledElements...)

	return p
}

// newChatRenditionPolicy widens bluemonday.UGCPolicy() with exactly one
// addition: a class attribute on div, restricted to chatTranscriptClassTokens
// — nothing else (unlike the email policy, no style attribute is ever
// allowed here: chat transcript content has no legitimate styling need, and
// every bubble/run/chrome element's presentation comes exclusively from
// renditionChatDelta below).
func newChatRenditionPolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("class").Matching(chatTranscriptClassTokens).OnElements("div")
	return p
}

// renditionBaseStyle is the shared CSS floor every content shape's
// stylesheet is composed on top of (05-UI-SPEC.md's Rendition Content
// Contract) — carried forward verbatim from the three plugin themeStyle/
// signalThemeStyle constants this file replaces. Every hex/rgba token here
// is an EXISTING theme token (proven not to have drifted from
// web/src/app.css by TestRenditionStylesheetTokensMatchAppCSS in
// rendition_test.go) — no new literal color is introduced by this move.
const renditionBaseStyle = `
:root { color-scheme: dark; }
/* Scrollbar styling (Quick task 260805-j98 follow-up). This document is
   served through the kernel's own /content route and rendered inside the
   detail pane's iframe (web/src/lib/components/DetailPane.svelte) — a
   SEPARATE document from the SPA, so the SPA's own root-level
   scrollbar-width/scrollbar-color (web/src/app.css) cannot inherit across
   that document boundary. This block is the same thin, theme-matched
   treatment applied independently, inside this self-contained
   stylesheet. It cannot reference web/src/app.css's var(--muted-foreground)
   custom property (that document doesn't exist here), so the color is the
   resolved rgba() equivalent of that same file's derived scrollbar
   tokens: --muted-foreground (#94a3b8 / rgb(148,163,184)) mixed to 35%/60%
   opacity against transparent, matching color-mix(in srgb, X p%,
   transparent)'s well-established same-RGB/alpha-only-scaled result. */
:root {
  scrollbar-width: thin;
  scrollbar-color: rgba(148, 163, 184, 0.35) transparent;
}
::-webkit-scrollbar { width: 10px; height: 10px; }
::-webkit-scrollbar-track { background: transparent; }
::-webkit-scrollbar-thumb {
  background: rgba(148, 163, 184, 0.35);
  border-radius: 9999px;
  border: 2px solid transparent;
  background-clip: padding-box;
}
::-webkit-scrollbar-thumb:hover { background: rgba(148, 163, 184, 0.6); }
::-webkit-scrollbar-corner { background: transparent; }
html, body {
  margin: 0;
  padding: 16px;
  background: #0f172a;
  color: #f1f5f9;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
  font-size: 15px;
  line-height: 1.6;
}
a { color: #60a5fa; text-decoration: underline; }
a:hover { color: #93c5fd; }
/* UI-09 search-term highlighting: a bare, attribute-free mark element
   (highlightTextNodes below never sets a class or any attribute on it, so
   this rule alone is sufficient — no sanitizer-policy entry is needed).
   Both hex values are already-declared theme tokens (web/src/app.css's
   --warning and --background) — no new colour is introduced here. */
mark { background: #fbbf24; color: #020617; border-radius: 2px; padding: 0 1px; }
`

// renditionProseDelta is shared by the email and markdown content shapes —
// the "prose" rules neither the chat transcript's bubble markup nor the
// shared base need.
const renditionProseDelta = `
h1, h2, h3, h4, h5, h6 { color: #f1f5f9; font-weight: 600; margin-top: 1.4em; margin-bottom: 0.5em; }
h1 { font-size: 1.6em; }
h2 { font-size: 1.35em; }
h3 { font-size: 1.15em; }
p { margin: 0.75em 0; }
ul, ol { padding-left: 1.5em; margin: 0.75em 0; }
li { margin: 0.25em 0; }
code { background: #1e293b; color: #f1f5f9; padding: 0.15em 0.4em; border-radius: 4px; font-size: 0.9em; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
pre { background: #1e293b; padding: 12px; border-radius: 8px; overflow-x: auto; }
pre code { background: none; padding: 0; }
blockquote { border-left: 3px solid #1e293b; margin: 0.75em 0; padding-left: 1em; color: #94a3b8; }
hr { border: none; border-top: 1px solid #1e293b; margin: 1.5em 0; }
table { border-collapse: collapse; width: 100%; margin: 0.75em 0; }
th, td { border: 1px solid #1e293b; padding: 6px 10px; text-align: left; }
`

// renditionEmailImageDelta hides images outright — email-specific: images
// can never load under the rendition CSP (no subresource permitted), so
// they are hidden rather than shown as broken icons (the tracking-pixel/
// broken-image fix from Phase 3, T-03-09).
const renditionEmailImageDelta = `img { display: none !important; }`

// renditionMarkdownImageDelta allows images at full container width —
// SilverBullet pages may legitimately reference local/embedded images that
// DO load under that plugin's own CSP, unlike email — a genuine per-shape
// divergence the policy table preserves rather than collapsing.
const renditionMarkdownImageDelta = `img { max-width: 100%; }`

// renditionEmailReadabilityDelta is the email profile's readability layer
// (carried forward verbatim from the proton plugin's body.go, 03-09-PLAN.md Task
// 3 gap G-03-2): the theme wins over the email by construction, not by
// luck. bluemonday's style sanitizer re-emits every surviving declaration
// as "property: value" only — douceur parses the CSS important marker into
// a separate field bluemonday never writes back — so an email's inline
// style can never mark itself as taking priority that way. The neutralizer
// selector below is therefore always authoritative over any inline colour/
// background-color that survives sanitization, and the three restoring
// rules that follow it each use a more specific selector than the
// body-and-every-descendant neutralizer, so they win among priority-marked
// declarations without depending on being placed last in this file. Only
// theme tokens already declared above are reused: no new colour value and
// no additional external reference of any kind is introduced by this
// block.
//
// UI-09: this delta's own `body, body *` neutralizer matches the base
// stylesheet's `mark` rule too (mark is a descendant of body) and would
// otherwise silently swallow the highlight for the email shape. The
// `body mark, body mark *` rule below is more specific than the
// neutralizer, so it wins among priority-marked declarations without
// depending on source order — the same pattern its sibling link/code/
// blockquote restoring rules already use.
const renditionEmailReadabilityDelta = `
body, body * {
  color: #f1f5f9 !important;
  background-color: transparent !important;
}
body a, body a * { color: #60a5fa !important; }
body code, body pre { background-color: #1e293b !important; }
body blockquote { color: #94a3b8 !important; }
body mark, body mark * { background-color: #fbbf24 !important; color: #020617 !important; }
`

// renditionChatDelta is the chat content-shape's own delta — no headings/
// lists/blockquote/table rules apply (transcripts don't produce that
// markup) — carried forward verbatim from the signal plugin's
// signalThemeStyle at the D-11 move. The accent hex #60a5fa appears ONLY on
// renditionBaseStyle's `a` (link) rule above — never here, on a bubble
// background, sender-name rule or timestamp rule (protects the 10% accent
// budget from per-participant color differentiation, 05-UI-SPEC.md Color;
// TestRenditionStylesheetTokensMatchAppCSS's sibling
// TestChatStylesheet_NoAccentColourOnBubbleSenderOrTimestamp in
// rendition_test.go is the relocated regression test for this rule).
const renditionChatDelta = `
img { display: none !important; }

.run { display: flex; flex-direction: column; margin: 16px 0; }
.run.own { align-items: flex-end; }
.run.other { align-items: flex-start; }
.run .bubble + .bubble { margin-top: 4px; }

.sender-name { font-size: 14px; font-weight: 600; color: #f1f5f9; margin-bottom: 4px; }

.bubble {
  max-width: 75%;
  border-radius: 12px;
  padding: 8px 16px;
}
.bubble.other { background: #1e293b; }
.bubble.own { background: #0f172a; border: 1px solid #1e293b; }

.bubble .body { font-size: 15px; font-weight: 400; line-height: 1.6; white-space: pre-wrap; overflow-wrap: anywhere; }
.bubble .tombstone { font-size: 15px; line-height: 1.6; font-style: italic; color: #94a3b8; }
.bubble .quote {
  font-size: 14px; font-weight: 400; font-style: italic; color: #94a3b8;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  margin-bottom: 4px;
}
.bubble .timestamp, .bubble .edited-suffix { font-size: 14px; font-weight: 400; color: #94a3b8; margin-top: 4px; }
.bubble .attachment {
  display: inline-block; font-size: 14px; font-weight: 400; color: #f1f5f9;
  background: #334155; border-radius: 999px; padding: 4px 12px; margin-top: 4px;
}
.bubble .reaction { font-size: 14px; font-weight: 400; color: #94a3b8; margin-top: 4px; }
`

// stylesheetForShape composes shape's full stylesheet as
// renditionBaseStyle plus that shape's own delta blocks — the shared floor
// exists exactly once, with per-shape deltas layered on it, rather than
// three independently-authored stylesheets. Returns "" for an unrecognised
// shape; sanitizeAndWrapRendition never reaches this function for one
// (renditionPolicies' lookup miss returns earlier).
func stylesheetForShape(shape toposv1.ContentShape) string {
	switch shape {
	case toposv1.ContentShape_CONTENT_SHAPE_EMAIL_HTML:
		return renditionBaseStyle + renditionProseDelta + renditionEmailImageDelta + renditionEmailReadabilityDelta
	case toposv1.ContentShape_CONTENT_SHAPE_MARKDOWN_HTML:
		return renditionBaseStyle + renditionProseDelta + renditionMarkdownImageDelta
	case toposv1.ContentShape_CONTENT_SHAPE_CHAT_TRANSCRIPT:
		return renditionBaseStyle + renditionChatDelta
	default:
		return ""
	}
}

// highlightTermMaxRunes bounds an individual term's maximum length (WR-02):
// alongside the count cap and the <2-rune drop below, this is the third
// bounded-work control the docstring promises for threat T-06-03. Without
// it, a caller could supply a single, arbitrarily long whitespace-free
// "word" via ?hl= that survives strings.Fields as one term; the existing
// length-guard in highlightTextNode (a term longer than the remaining
// document text is skipped in O(1) per position) keeps that from being a
// practical CPU/DoS issue today, but bounding it here too keeps the
// docstring's claim accurate against a future refactor that weakens or
// removes that guard.
const highlightTermMaxRunes = 64

// highlightTerms is the kernel half of UI-09's shared term-derivation rule
// — the client half lives in web/src/lib/format.ts's own highlightTerms
// and MUST implement the identical rule, so what the client highlights
// never disagrees with what this function derives from the same query
// string. It trims raw, splits on whitespace (the same strings.Fields
// behaviour kernel/index/store.go's ftsQuery uses), lowercases every term,
// de-duplicates, drops any term shorter than 2 runes or longer than
// highlightTermMaxRunes, and caps the result at the first 8 terms — the
// bounded-work controls for threat T-06-03. Returns nil for an empty or
// all-dropped input.
func highlightTerms(raw string) []string {
	fields := strings.Fields(raw)
	seen := make(map[string]bool, len(fields))
	var terms []string
	for _, f := range fields {
		f = strings.ToLower(f)
		n := utf8.RuneCountInString(f)
		if n < 2 || n > highlightTermMaxRunes {
			continue
		}
		if seen[f] {
			continue
		}
		seen[f] = true
		terms = append(terms, f)
		if len(terms) == 8 {
			break
		}
	}
	return terms
}

// highlightTextNodes walks roots — the top-level nodes of a parsed
// fragment — and every descendant, wrapping each case-insensitive literal
// match of any term in terms in a bare <mark> element (T-06-01). Mutates
// the tree in place; never touches attribute values or tag bytes, which
// tree mutation gives for free (no byte-level or pattern substitution over
// the parsed markup is ever performed here).
//
// Descends into every element except script/style — bluemonday has already
// stripped any <script>, and the only <style> element a rendition document
// ever carries is the kernel's own later-injected stylesheet, added after
// this step runs, so this guard is defensive rather than load-bearing.
//
// Each node's children are snapshotted before the walk visits them, so
// replacing a matched text node's position among its siblings never
// perturbs the walk, and the newly inserted <mark> elements are never
// re-walked — re-walking inserted nodes is the infinite-loop failure mode
// this snapshot avoids.
func highlightTextNodes(roots []*html.Node, terms []string) {
	if len(terms) == 0 {
		return
	}

	// Longest-first (by rune count) so a longer term always wins over a
	// shorter overlapping one at the same scan position — the
	// deterministic tie-break this function guarantees.
	sorted := make([]string, len(terms))
	copy(sorted, terms)
	sort.Slice(sorted, func(i, j int) bool {
		return utf8.RuneCountInString(sorted[i]) > utf8.RuneCountInString(sorted[j])
	})

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return
		}
		var children []*html.Node
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			children = append(children, c)
		}
		for _, c := range children {
			if c.Type == html.TextNode {
				highlightTextNode(n, c, sorted)
				continue
			}
			walk(c)
		}
	}
	for _, root := range roots {
		walk(root)
	}
}

// highlightTextNode scans textNode's Data for terms (already sorted
// longest-first, already lowercase) and, if any match is found, replaces
// textNode in place among parent's children with an ordered sequence of
// plain text.TextNodes (the unmatched spans) and bare <mark>
// html.ElementNodes (the matched spans, each carrying a single TextNode
// child holding the matched source substring VERBATIM — the document's
// original casing is preserved — and no attributes and no class). A
// bare, attribute-free element needs no sanitizer-policy entry, which is
// why the chat-transcript class allowlist (T-05-17) is unaffected by this
// insertion.
//
// Matching walks textNode's Data rune-by-rune (never by byte index) and
// compares each candidate window against a term via unicode.ToLower per
// rune, rather than lower-casing the whole string up front — case-folding
// can change a string's byte length for some Unicode text, and comparing
// rune-by-rune against already-lowercased terms sidesteps that entirely,
// so a multi-byte rune adjacent to (or inside) a match is never split or
// corrupted (T-06-01's multi-byte-safety requirement).
//
// A term matching across a text-node boundary (i.e. split across two
// adjacent inline elements in the sanitized markup) is not detected — this
// function only ever scans a single text node — which degrades to no
// highlight for that occurrence rather than ever risking document
// corruption; see the must_haves backstop row this behaviour satisfies.
func highlightTextNode(parent *html.Node, textNode *html.Node, terms []string) {
	runes := []rune(textNode.Data)
	n := len(runes)

	var newNodes []*html.Node
	cursor := 0
	matched := false
	i := 0
	for i < n {
		matchLen := 0
		for _, term := range terms {
			termRunes := []rune(term)
			if i+len(termRunes) > n {
				continue
			}
			match := true
			for k, tr := range termRunes {
				if unicode.ToLower(runes[i+k]) != tr {
					match = false
					break
				}
			}
			if match {
				matchLen = len(termRunes)
				break
			}
		}
		if matchLen == 0 {
			i++
			continue
		}
		if i > cursor {
			newNodes = append(newNodes, &html.Node{Type: html.TextNode, Data: string(runes[cursor:i])})
		}
		mark := &html.Node{Type: html.ElementNode, DataAtom: atom.Mark, Data: "mark"}
		mark.AppendChild(&html.Node{Type: html.TextNode, Data: string(runes[i : i+matchLen])})
		newNodes = append(newNodes, mark)
		i += matchLen
		cursor = i
		matched = true
	}
	if !matched {
		return
	}
	if cursor < n {
		newNodes = append(newNodes, &html.Node{Type: html.TextNode, Data: string(runes[cursor:])})
	}

	for _, nn := range newNodes {
		parent.InsertBefore(nn, textNode)
	}
	parent.RemoveChild(textNode)
}

// highlightSanitizedFragment parses sanitized (already-sanitized-and-
// trusted HTML bytes) as an HTML fragment in a <body> context, mutates the
// parsed tree via highlightTextNodes, and renders it back to bytes. The
// parsed-then-rendered round trip is itself the guarantee that no matched
// term is ever spliced into raw HTML: golang.org/x/net/html's own encoder
// is what produces the output bytes, never string concatenation.
//
// A <body> context node (rather than html.Parse's default full-document
// context) makes ParseFragment round-trip the input as a fragment — it
// does not gain a nested <html>/<head>/<body> wrapper inside
// sanitizeAndWrapRendition's own later <body> wrap step.
func highlightSanitizedFragment(sanitized []byte, terms []string) ([]byte, error) {
	context := &html.Node{Type: html.ElementNode, Data: "body", DataAtom: atom.Body}
	nodes, err := html.ParseFragment(bytes.NewReader(sanitized), context)
	if err != nil {
		return nil, fmt.Errorf("httpapi: parsing sanitized fragment for highlighting: %w", err)
	}

	highlightTextNodes(nodes, terms)

	var buf bytes.Buffer
	for _, node := range nodes {
		if err := html.Render(&buf, node); err != nil {
			return nil, fmt.Errorf("httpapi: rendering highlighted fragment: %w", err)
		}
	}
	return buf.Bytes(), nil
}

// sanitizeAndWrapRendition is the kernel's one sanitize/wrap/theme pipeline
// (D-11): it looks up shape's policy, sanitizes fragment with it, and wraps
// the sanitized result in a minimal, self-contained HTML document —
// doctype, a <head> carrying only a charset meta tag and shape's composed
// stylesheet, and the sanitized fragment as <body> content unchanged.
//
// Sanitization always runs BEFORE wrapping, and the wrapped output is NEVER
// fed back through a sanitizer: the stylesheet text is Go source, never
// derived from fragment content, so injecting it after sanitization cannot
// reintroduce any XSS surface the policy removed.
//
// UI-09: when terms is non-empty, a highlighting step sits strictly
// between the sanitize and wrap steps — highlightSanitizedFragment parses
// the already-sanitized bytes, wraps each matched term in a bare <mark>
// element via tree mutation (highlightTextNodes), and renders the tree
// back to bytes. That highlighted output is itself never fed back through
// policy.SanitizeBytes — the same "sanitize once, trust your own
// Go-authored insertions after that" rule this doc comment already
// established, now covering the highlighter's own insertions too. When
// terms is empty, this step is skipped entirely and sanitized is used
// unchanged — the no-search path stays byte-identical to the pre-UI-09
// output.
//
// An unrecognised or unspecified shape (including the zero value,
// CONTENT_SHAPE_UNSPECIFIED) returns errUnrecognisedContentShape and no
// bytes — the kernel fails closed rather than ever guessing a policy
// (T-05-16). Callers (renditionHandler/agentRenditionHandler in item.go/
// agent.go) must write no body when this function returns an error.
func sanitizeAndWrapRendition(shape toposv1.ContentShape, fragment []byte, terms []string) ([]byte, error) {
	policy, ok := renditionPolicies[shape]
	if !ok {
		return nil, fmt.Errorf("%w: %v", errUnrecognisedContentShape, shape)
	}

	sanitized := policy.SanitizeBytes(fragment)

	if len(terms) > 0 {
		highlighted, err := highlightSanitizedFragment(sanitized, terms)
		if err != nil {
			return nil, err
		}
		sanitized = highlighted
	}

	style := stylesheetForShape(shape)

	var buf bytes.Buffer
	buf.WriteString("<!doctype html>\n<html><head><meta charset=\"utf-8\"><style>")
	buf.WriteString(style)
	buf.WriteString("</style></head><body>")
	buf.Write(sanitized)
	buf.WriteString("</body></html>")
	return buf.Bytes(), nil
}
