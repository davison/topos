package main

import (
	"bytes"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
)

// mdConverter and sanitizePolicy are built once at package init — both are
// documented safe for concurrent use, and rebuilding either per call would
// be wasteful (goldmark.Markdown parses its own extension chain on
// construction; bluemonday.UGCPolicy builds its allowlist tables).
//
// goldmark is left at its defaults deliberately: it does not render raw
// HTML passed through the source markdown, and it does not permit
// dangerous URL schemes (e.g. "javascript:") in link/image targets. Do NOT
// enable an "unsafe" HTML-passthrough extension to "make links work" — the
// sanitizer below is the second, independent layer of defense (T-02-01),
// not a substitute for goldmark's own safe-by-default behavior.
var mdConverter = goldmark.New()
var sanitizePolicy = bluemonday.UGCPolicy()

// RenderSanitized converts markdown to HTML via goldmark, then sanitizes
// the result via bluemonday.UGCPolicy().SanitizeBytes — two independent
// layers (T-02-01) against a hostile or malformed page body (SilverBullet
// pages are user-authored content of arbitrary shape; a page might contain
// pasted raw HTML, a script tag, or a javascript: link). The returned bytes
// are an HTML *fragment* (no doctype/html/head/body) — callers that serve
// this directly to a browser (rather than embedding it in a larger
// document) should use WrapDocument instead.
func RenderSanitized(markdown []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := mdConverter.Convert(markdown, &buf); err != nil {
		return nil, err
	}
	return sanitizePolicy.SanitizeBytes(buf.Bytes()), nil
}

// themeStyle is a fixed, hardcoded stylesheet — never derived from
// sanitized/user content — matching the web SPA's dark theme tokens
// (web/src/app.css: --background #020617, --card #0f172a, --foreground
// #f1f5f9, --primary #60a5fa, --muted #1e293b, --muted-foreground
// #94a3b8, --border #1e293b). Injected verbatim into the wrapped
// document's <head> by WrapDocument, entirely independent of bluemonday's
// sanitization pass over the page body: because this string is Go source,
// never markdown/HTML that passed through the sanitizer, appending it
// after sanitization cannot reintroduce any XSS surface the sanitizer
// removed, and it needs no further sanitizing itself.
//
// Fixed by design (found via live UAT of Task 2's detail-pane rendering,
// which shipped with no styling — default black-on-transparent text over
// the app's dark background is close to unreadable). Self-contained: no
// external stylesheet fetch, no @import, no url() references — matching
// the iframe's own sandboxed, no-network-of-its-own delivery (the
// document itself is served from the kernel's own /content route; this
// style block never causes a second request).
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
img { max-width: 100%; }
table { border-collapse: collapse; width: 100%; margin: 0.75em 0; }
th, td { border: 1px solid #1e293b; padding: 6px 10px; text-align: left; }
`

// WrapDocument wraps an already-sanitized HTML fragment (RenderSanitized's
// output) in a minimal, self-contained HTML document — doctype, a <head>
// carrying only a charset meta tag and the fixed themeStyle above, and the
// fragment as <body> content unchanged. This is what Fetch actually
// returns for FULL/PREVIEW: the detail pane's iframe navigates directly to
// the kernel's /content route and renders whatever document comes back, so
// wrapping (rather than injecting client-side) is what makes the
// stylesheet apply inside that iframe's own document without the SPA ever
// touching the sanitized bytes.
func WrapDocument(sanitizedFragment []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString("<!doctype html>\n<html><head><meta charset=\"utf-8\"><style>")
	buf.WriteString(themeStyle)
	buf.WriteString("</style></head><body>")
	buf.Write(sanitizedFragment)
	buf.WriteString("</body></html>")
	return buf.Bytes()
}
