package main

import (
	"strings"
	"testing"
)

func TestRenderSanitized_HeadingAndText(t *testing.T) {
	out, err := RenderSanitized([]byte("# Title\n\nhello"))
	if err != nil {
		t.Fatalf("RenderSanitized: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "<h1") {
		t.Errorf("expected an h1 element, got: %s", got)
	}
	if !strings.Contains(got, "hello") {
		t.Errorf("expected the word %q to survive rendering, got: %s", "hello", got)
	}
}

func TestRenderSanitized_StripsRawScriptElement(t *testing.T) {
	out, err := RenderSanitized([]byte("hello <script>alert(1)</script> world"))
	if err != nil {
		t.Fatalf("RenderSanitized: %v", err)
	}
	got := string(out)
	if strings.Contains(strings.ToLower(got), "<script") {
		t.Errorf("expected no <script element to survive sanitization, got: %s", got)
	}
}

func TestRenderSanitized_StripsEventHandlerAttribute(t *testing.T) {
	out, err := RenderSanitized([]byte(`<img src="x" onerror="alert(1)">`))
	if err != nil {
		t.Fatalf("RenderSanitized: %v", err)
	}
	got := string(out)
	if strings.Contains(strings.ToLower(got), "onerror") {
		t.Errorf("expected no onerror attribute to survive sanitization, got: %s", got)
	}
}

func TestRenderSanitized_StripsJavascriptSchemeLink(t *testing.T) {
	out, err := RenderSanitized([]byte(`[click me](javascript:alert(1))`))
	if err != nil {
		t.Fatalf("RenderSanitized: %v", err)
	}
	got := string(out)
	if strings.Contains(strings.ToLower(got), "javascript:") {
		t.Errorf("expected no javascript: scheme to survive in an href, got: %s", got)
	}
}

// TestRenderSanitized_OrdinaryMarkdownSurvives proves the sanitizer is
// non-vacuous: ordinary markdown constructs (headings, http links, lists,
// emphasis, fenced code) must render, not just be safely stripped.
func TestRenderSanitized_OrdinaryMarkdownSurvives(t *testing.T) {
	md := "# Heading\n\n" +
		"[a link](http://example.com/page)\n\n" +
		"- item one\n- item two\n\n" +
		"**bold** and *emphasis*\n\n" +
		"```\ncode block\n```\n"

	out, err := RenderSanitized([]byte(md))
	if err != nil {
		t.Fatalf("RenderSanitized: %v", err)
	}
	got := string(out)

	checks := []string{"<h1", "href=\"http://example.com/page\"", "<li", "<strong", "<em", "<pre", "code block"}
	for _, want := range checks {
		if !strings.Contains(got, want) {
			t.Errorf("expected rendered output to contain %q, got: %s", want, got)
		}
	}
}

// TestWrapDocument_InjectsThemeStyleAndPreservesFragment is the regression
// test for the live-UAT-found contrast bug: unstyled rendered markdown
// rendered near-black text on the pane's dark background. WrapDocument
// must produce a full document carrying the fixed theme stylesheet in
// <head>, with the sanitized fragment unchanged in <body>.
func TestWrapDocument_InjectsThemeStyleAndPreservesFragment(t *testing.T) {
	fragment := []byte("<h1>Title</h1><p>hello</p>")
	doc := WrapDocument(fragment)
	got := string(doc)

	if !strings.Contains(got, "<style>") {
		t.Fatal("expected a <style> block in the wrapped document")
	}
	// Theme tokens from web/src/app.css that must appear in the injected
	// stylesheet: dark card background and light foreground text, so the
	// iframe document is actually readable against the app's dark theme.
	for _, want := range []string{"#0f172a", "#f1f5f9", "#60a5fa"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected the injected stylesheet to reference theme token %q, got: %s", want, got)
		}
	}
	if !strings.Contains(got, string(fragment)) {
		t.Errorf("expected the original sanitized fragment to be preserved unchanged in the body, got: %s", got)
	}
	// The style block must never depend on an external resource — no
	// second request, and nothing for the sandboxed iframe to be denied
	// fetching.
	if strings.Contains(got, "@import") || strings.Contains(got, "url(") {
		t.Errorf("expected a fully self-contained stylesheet (no @import/url()), got: %s", got)
	}
}

// TestWrapDocument_StyleNeverReprocessedThroughSanitizer proves the
// wrapping step happens strictly after sanitization: a fragment that
// still (hypothetically) carried a raw <script> tag would not be further
// altered by WrapDocument — the guarantee against hostile content lives
// entirely in RenderSanitized, and WrapDocument's own fixed <style> output
// is not itself sanitized (it doesn't need to be — it's Go source, never
// derived from page content).
func TestWrapDocument_StyleNeverReprocessedThroughSanitizer(t *testing.T) {
	doc := WrapDocument([]byte("<p>plain</p>"))
	// The fixed stylesheet must appear byte-for-byte (bluemonday would
	// strip or mangle a <style> element if this were re-run through it).
	if !strings.Contains(string(doc), "font-family:") {
		t.Errorf("expected the literal stylesheet text to survive unmodified, got: %s", doc)
	}
}

// TestWrapDocument_InjectsThinThemeMatchedScrollbar is the regression test
// for the live-UAT-found gap (Quick task 260805-j98 follow-up): the
// SPA's stream/detail scrollbar was restyled thin and theme-matched via
// web/src/app.css's root-level scrollbar-width/scrollbar-color, but that
// document-level CSS cannot cross the iframe boundary into THIS
// self-contained document (the rendered-note's own themeStyle) — so a
// SilverBullet note rendition kept the browser-default scrollbar until
// this rule was added directly here.
func TestWrapDocument_InjectsThinThemeMatchedScrollbar(t *testing.T) {
	doc := WrapDocument([]byte("<p>hello</p>"))
	got := string(doc)

	if !strings.Contains(got, "scrollbar-width: thin") {
		t.Errorf("expected the wrapped document's stylesheet to declare scrollbar-width: thin, got: %s", got)
	}
	if !strings.Contains(got, "scrollbar-color:") {
		t.Errorf("expected the wrapped document's stylesheet to declare scrollbar-color, got: %s", got)
	}
	if !strings.Contains(got, "::-webkit-scrollbar-thumb") {
		t.Errorf("expected the WebKit pseudo-element fallback (pre-121 Chromium) in the wrapped document, got: %s", got)
	}
	// rgba(148, 163, 184, ...) is the resolved equivalent of
	// web/src/app.css's var(--muted-foreground) (#94a3b8) mixed via
	// color-mix() — this document cannot reference that custom property,
	// so the color must be this same resolved value, not a new one.
	if !strings.Contains(got, "148, 163, 184") {
		t.Errorf("expected the scrollbar color to resolve from the same --muted-foreground token as web/src/app.css (rgb 148,163,184), got: %s", got)
	}
}
