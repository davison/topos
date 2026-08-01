// Command webspaces-plugin-proton: MIME part extraction from a peeked
// RFC822 message, plus (03-02) the bluemonday-based sanitize-and-wrap
// pipeline for the text/html rendition path — the HTML-body counterpart
// to plugins/silverbullet/render.go's markdown-based one.
package main

import (
	"bytes"
	"io"
	"strings"
	"unicode/utf8"

	message "github.com/emersion/go-message"
	"github.com/emersion/go-message/mail"
	"github.com/microcosm-cc/bluemonday"
)

const (
	// maxPartBytes bounds a single MIME part read, well under
	// sdk.MaxMessageSize (64 MiB), so a crafted message with one enormous
	// part cannot exhaust memory (T-03-07).
	maxPartBytes = 8 * 1024 * 1024
	// maxParts bounds the number of MIME parts read from a single
	// message, so a crafted message with thousands of parts cannot
	// exhaust memory or spin forever (T-03-07).
	maxParts = 256
	// previewRuneCap mirrors plugins/silverbullet's Snippet rune cap —
	// truncation is by rune count, never byte count, so a multi-byte
	// preview is never cut mid-codepoint.
	previewRuneCap = 500
)

// PlainTextPart extracts the first text/plain inline part from a peeked
// RFC822 message via mail.CreateReader/NextPart. Every part read goes
// through io.LimitReader bounded by maxPartBytes, and the loop stops
// after maxParts parts, so a maliciously crafted message cannot exhaust
// memory (T-03-07). A message with no text/plain part returns empty text
// rather than an error.
func PlainTextPart(raw []byte) (string, error) {
	return extractPart(raw, "text/plain")
}

// HTMLPart extracts the first text/html inline part from a peeked RFC822
// message, under the identical io.LimitReader/maxParts bounds
// PlainTextPart already established (T-03-12). A message with no
// text/html part (a plain-text-only email) returns empty text and a nil
// error — plugin.go's Fetch falls through to 03-01's existing
// text-only behaviour in that case.
func HTMLPart(raw []byte) (string, error) {
	return extractPart(raw, "text/html")
}

// extractPart walks raw's MIME structure via mail.CreateReader/NextPart
// once and returns the first inline part whose Content-Type equals
// wantContentType. Shared by PlainTextPart and HTMLPart so both
// extractions apply the identical part-count/byte-size bounds (T-03-12: a
// crafted message with an enormous single part or thousands of parts must
// not exhaust memory or spin unbounded).
//
// go-message's CreateReader/NextPart can both return a non-fatal
// "unknown charset/transfer-encoding" error alongside a still-usable
// reader/part (message.IsUnknownCharset / message.IsUnknownEncoding) —
// this function treats that class of error as recoverable (keep reading
// with whatever best-effort decode go-message produced) rather than
// failing extraction outright, since one malformed part must not make
// the whole message's body permanently unavailable.
func extractPart(raw []byte, wantContentType string) (string, error) {
	mr, err := mail.CreateReader(bytes.NewReader(raw))
	if err != nil && !isBenignParseError(err) {
		return "", err
	}
	if mr == nil {
		return "", nil
	}
	defer mr.Close()

	for i := 0; i < maxParts; i++ {
		p, perr := mr.NextPart()
		if perr == io.EOF {
			break
		}
		if perr != nil && !isBenignParseError(perr) {
			return "", perr
		}
		if p == nil {
			continue
		}

		h, ok := p.Header.(*mail.InlineHeader)
		if !ok {
			continue
		}
		ct, _, _ := h.ContentType()
		if ct != wantContentType {
			continue
		}

		b, _ := io.ReadAll(io.LimitReader(p.Body, maxPartBytes))
		return string(b), nil
	}

	return "", nil
}

// isBenignParseError reports whether err is the "unknown charset" or
// "unknown transfer encoding" class of error go-message returns
// alongside a still-usable reader/part — see the package doc comments on
// mail.CreateReader and mail.Reader.NextPart.
func isBenignParseError(err error) bool {
	return message.IsUnknownCharset(err) || message.IsUnknownEncoding(err)
}

// HasRenderableText reports whether s carries content a reader can
// actually see once whitespace is trimmed. fetchFull uses this to decide
// whether a message's extracted text/plain part IS the message's
// content: a present-but-blank part (a common multipart/alternative
// artifact) must not suppress the HTML fallback that follows it.
// Whitespace is defined by strings.TrimSpace (unicode.IsSpace), not ASCII
// spaces only, so a no-break-space-only part is correctly treated as
// blank.
func HasRenderableText(s string) bool {
	return strings.TrimSpace(s) != ""
}

// Snippet truncates s to at most previewRuneCap runes, by rune count
// never byte count, so a multi-byte body preview is never cut
// mid-codepoint. Mirrors plugins/silverbullet's Snippet helper.
func Snippet(s string) string {
	if utf8.RuneCountInString(s) <= previewRuneCap {
		return s
	}
	runes := []rune(s)
	return string(runes[:previewRuneCap])
}

// styledElements is the named set of block and inline elements permitted
// to carry a sanitized style attribute — paragraphs, spans, divs, table
// cells, headings, list items, and anchors. bluemonday's own published
// HTML-email example (cmd/sanitise_html_email/main.go) allows the style
// attribute Globally() and its own comment says that is "not safe" —
// this policy deliberately does not copy that. Scoping the attribute to a
// named element set (never Globally()) is the residual-risk tradeoff
// 03-RESEARCH.md Pitfall 3 calls for.
var styledElements = []string{"p", "span", "div", "td", "th", "h1", "h2", "h3", "h4", "h5", "h6", "li", "a"}

// emailSanitizePolicy is built once at init (documented safe for
// concurrent use — bluemonday.UGCPolicy builds its allowlist tables once,
// same rationale as plugins/silverbullet/render.go's package-level
// sanitizePolicy). Widened, deliberately and narrowly, from
// bluemonday.UGCPolicy(): the style attribute is allowed only on
// styledElements above, and only a presentational CSS-property allowlist
// is permitted through it — colour, background-color, font-weight,
// font-style, font-size, font-family, text-align, text-decoration,
// padding, margin, border, width, height. Everything positional or
// behavioural (position, z-index, transform, animation, transition,
// content, clip, filter, mix-blend-mode, pointer-events) stays denied by
// omission.
//
// The justification for allowing any styling at all: the kernel's
// rendition route (kernel/httpapi/item.go) already serves this document
// under `Content-Security-Policy: default-src 'none'; ...; sandbox`, so no
// CSS declaration this policy lets through can issue a network request —
// which also defeats email tracking pixels for free (03-RESEARCH.md
// Pitfall 3, T-03-10, T-03-11).
var emailSanitizePolicy = newEmailSanitizePolicy()

func newEmailSanitizePolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()

	p.AllowAttrs("style").OnElements(styledElements...)

	p.AllowStyles(
		"color", "background-color", "font-weight", "font-style", "font-size",
		"font-family", "text-align", "text-decoration", "padding", "margin",
		"border", "width", "height",
	).OnElements(styledElements...)

	return p
}

// RenderSanitizedEmail sanitizes an untrusted email HTML part via
// emailSanitizePolicy.SanitizeBytes. There is no markdown conversion step
// here (unlike plugins/silverbullet/render.go's RenderSanitized) — email
// bodies fetched by this plugin are already HTML.
func RenderSanitizedEmail(htmlPart []byte) []byte {
	return emailSanitizePolicy.SanitizeBytes(htmlPart)
}

// themeStyle and WrapDocument are copied verbatim in shape from
// plugins/silverbullet/render.go: the same fixed, self-contained
// dark-theme stylesheet (no external fetch, no @import, no url()
// reference) matching the web SPA's theme tokens, applied strictly AFTER
// sanitization (WrapDocument never re-runs its input through
// emailSanitizePolicy) so it cannot reintroduce any XSS surface
// RenderSanitizedEmail removed (T-03-09).
const themeStyle = `
:root { color-scheme: dark; }
html, body {
  margin: 0;
  padding: 16px;
  background: #0f172a;
  color: #f1f5f9;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
  font-size: 15px;
  line-height: 1.6;
}
h1, h2, h3, h4, h5, h6 { color: #f1f5f9; font-weight: 600; margin-top: 1.4em; margin-bottom: 0.5em; }
h1 { font-size: 1.6em; }
h2 { font-size: 1.35em; }
h3 { font-size: 1.15em; }
p { margin: 0.75em 0; }
a { color: #60a5fa; text-decoration: underline; }
a:hover { color: #93c5fd; }
ul, ol { padding-left: 1.5em; margin: 0.75em 0; }
li { margin: 0.25em 0; }
code { background: #1e293b; color: #f1f5f9; padding: 0.15em 0.4em; border-radius: 4px; font-size: 0.9em; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; }
pre { background: #1e293b; padding: 12px; border-radius: 8px; overflow-x: auto; }
pre code { background: none; padding: 0; }
blockquote { border-left: 3px solid #1e293b; margin: 0.75em 0; padding-left: 1em; color: #94a3b8; }
hr { border: none; border-top: 1px solid #1e293b; margin: 1.5em 0; }
/* Images in an email body can never load: the rendition is served under
   a policy that permits no subresource of any kind — remote, data-URI or
   otherwise (kernel/httpapi/item.go's Content-Security-Policy). Hiding
   them removes broken-image litter without hiding anything that could
   ever have been displayed, and changes nothing about the
   tracking-pixel defence, which is the blocked request itself, not the
   element (T-03-09-01/T-03-09-05). */
img { display: none !important; }
table { border-collapse: collapse; width: 100%; margin: 0.75em 0; }
th, td { border: 1px solid #1e293b; padding: 6px 10px; text-align: left; }

/* Readability layer (03-09-PLAN.md Task 3, gap G-03-2): the theme wins
   over the email by construction, not by luck. bluemonday's style
   sanitizer re-emits every surviving declaration as "property: value"
   only — douceur parses the CSS important marker into a separate field
   bluemonday never writes back (proven by
   TestRenderSanitizedEmail_EmailCannotMarkADeclarationImportant) — so an
   email's inline style can never mark itself as taking priority that
   way. That leaves two CSS rules deciding who wins when a non-priority
   email declaration and a priority-marked author declaration disagree:
   origin (a priority-marked author declaration always beats a
   non-priority one, regardless of specificity) and, among priority-
   marked declarations, specificity. The neutralizer selector below is
   therefore always authoritative over any inline colour/background-color
   that survives sanitization, and the three restoring rules that follow
   it each use a more specific selector (an anchor, a code/pre pair, a
   blockquote, versus the body-and-every-descendant selector), so they
   win among priority-marked declarations without depending on being
   placed last in this file. Only theme tokens already declared above are
   reused: no new colour value and no additional external reference of
   any kind is introduced by this block. */
body, body * {
  color: #f1f5f9 !important;
  background-color: transparent !important;
}
body a, body a * { color: #60a5fa !important; }
body code, body pre { background-color: #1e293b !important; }
body blockquote { color: #94a3b8 !important; }
`

// WrapDocument wraps an already-sanitized HTML fragment
// (RenderSanitizedEmail's output) in a minimal, self-contained HTML
// document — doctype, a <head> carrying only a charset meta tag and the
// fixed themeStyle above, and the fragment as <body> content unchanged.
// This is what Fetch returns for FULL/PREVIEW when an HTML part exists:
// the detail pane's iframe navigates directly to the kernel's /content
// route and renders whatever document comes back.
func WrapDocument(sanitizedFragment []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString("<!doctype html>\n<html><head><meta charset=\"utf-8\"><style>")
	buf.WriteString(themeStyle)
	buf.WriteString("</style></head><body>")
	buf.Write(sanitizedFragment)
	buf.WriteString("</body></html>")
	return buf.Bytes()
}
