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
// pasted raw HTML, a script tag, or a javascript: link).
func RenderSanitized(markdown []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := mdConverter.Convert(markdown, &buf); err != nil {
		return nil, err
	}
	return sanitizePolicy.SanitizeBytes(buf.Bytes()), nil
}
