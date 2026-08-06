// rendition.go is the kernel-owned content-serving boundary D-11
// consolidates into: plugins used to each carry their own near-identical
// sanitize policy, theme stylesheet, and document-wrapping helper
// (plugins/proton/body.go, plugins/silverbullet/render.go,
// plugins/signal/render.go before this move) — a theme change meant
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

	"github.com/microcosm-cc/bluemonday"

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
// carried forward verbatim from plugins/proton/body.go's styledElements.
// bluemonday's own published HTML-email example allows the style attribute
// Globally(), and its own comment says that is "not safe" — this policy
// deliberately does not copy that; scoping the attribute to a named element
// set (never Globally()) is the residual-risk tradeoff 03-RESEARCH.md
// Pitfall 3 called for and D-11 preserves unchanged.
var renditionStyledElements = []string{"p", "span", "div", "td", "th", "h1", "h2", "h3", "h4", "h5", "h6", "li", "a"}

// chatTranscriptClassTokens is the fixed, closed set of class tokens the
// chat content-shape policy allows on a div element — every token
// plugins/signal/render.go's renderTranscript/renderBubble ever emit, and
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

// newEmailRenditionPolicy widens bluemonday.UGCPolicy() exactly as
// plugins/proton/body.go's newEmailSanitizePolicy did: the style attribute
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
// (carried forward verbatim from plugins/proton/body.go, 03-09-PLAN.md Task
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
const renditionEmailReadabilityDelta = `
body, body * {
  color: #f1f5f9 !important;
  background-color: transparent !important;
}
body a, body a * { color: #60a5fa !important; }
body code, body pre { background-color: #1e293b !important; }
body blockquote { color: #94a3b8 !important; }
`

// renditionChatDelta is the chat content-shape's own delta — no headings/
// lists/blockquote/table rules apply (transcripts don't produce that
// markup) — carried forward verbatim from plugins/signal/render.go's
// signalThemeStyle. The accent hex #60a5fa appears ONLY on
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
// An unrecognised or unspecified shape (including the zero value,
// CONTENT_SHAPE_UNSPECIFIED) returns errUnrecognisedContentShape and no
// bytes — the kernel fails closed rather than ever guessing a policy
// (T-05-16). Callers (renditionHandler/agentRenditionHandler in item.go/
// agent.go) must write no body when this function returns an error.
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
	buf.WriteString("</style></head><body>")
	buf.Write(sanitized)
	buf.WriteString("</body></html>")
	return buf.Bytes(), nil
}
