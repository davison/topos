package main

import (
	"strings"
	"testing"
)

// TestRenderSanitizedEmail_StripsScriptElement proves the sanitizer removes
// a raw <script> element and any event-handler attribute from untrusted
// email HTML — the primary V5 control this plan adds (T-03-09).
func TestRenderSanitizedEmail_StripsScriptElement(t *testing.T) {
	out := RenderSanitizedEmail([]byte(`<p>hello</p><script>alert(1)</script><img src="x" onerror="alert(2)">`))
	got := strings.ToLower(string(out))
	if strings.Contains(got, "<script") {
		t.Errorf("expected no <script element to survive sanitization, got: %s", got)
	}
	if strings.Contains(got, "onerror=") {
		t.Errorf("expected no onerror attribute to survive sanitization, got: %s", got)
	}
}

// TestRenderSanitizedEmail_StripsJavascriptSchemeHref proves a
// javascript-scheme anchor href does not survive sanitization.
func TestRenderSanitizedEmail_StripsJavascriptSchemeHref(t *testing.T) {
	out := RenderSanitizedEmail([]byte(`<a href="javascript:alert(1)">click me</a>`))
	got := strings.ToLower(string(out))
	if strings.Contains(got, "javascript:") {
		t.Errorf("expected no javascript: scheme to survive in an href, got: %s", got)
	}
}

// TestRenderSanitizedEmail_PreservesColorDropsPosition proves the narrowed
// style-attribute allowlist (03-RESEARCH.md Pitfall 3, T-03-11): a
// presentational declaration (color) is preserved, but a
// positional/behavioural declaration (position) on the same element is
// dropped — never bluemonday's own email demo's Globally() allowance.
func TestRenderSanitizedEmail_PreservesColorDropsPosition(t *testing.T) {
	out := RenderSanitizedEmail([]byte(`<p style="color: #ff0000; position: fixed; top: 0;">styled text</p>`))
	got := string(out)
	if !strings.Contains(got, "color") {
		t.Errorf("expected the safe presentational color declaration to survive, got: %s", got)
	}
	if strings.Contains(got, "position") {
		t.Errorf("expected the unsafe position declaration to be dropped, got: %s", got)
	}
	if !strings.Contains(got, "styled text") {
		t.Errorf("expected the element's text content to survive, got: %s", got)
	}
}

// TestRenderSanitizedEmail_StyleAttributeScopedToNamedElements proves the
// style attribute is not permitted on an element outside styledElements
// (e.g. <body>, which this sanitizer never even sees standalone, or an
// element like <input> that is not in the named set) — the residual-risk
// scoping this plan deliberately narrows from bluemonday's own demo.
func TestRenderSanitizedEmail_StyleAttributeScopedToNamedElements(t *testing.T) {
	out := RenderSanitizedEmail([]byte(`<input style="color: red;" value="x">`))
	got := string(out)
	if strings.Contains(got, "style=") {
		t.Errorf("expected style attribute to be stripped from an element outside the named allowlist, got: %s", got)
	}
}

// TestRenderSanitizedEmail_RemoteImagePreservedButHarmless proves a remote
// image reference may survive sanitization (bluemonday's UGCPolicy allows
// <img src>) — the tracking-pixel defense lives in the kernel's CSP
// (default-src 'none'), not in stripping the element, per 03-RESEARCH.md
// Pitfall 3 / T-03-10.
func TestRenderSanitizedEmail_RemoteImagePreservedButHarmless(t *testing.T) {
	out := RenderSanitizedEmail([]byte(`<img src="https://attacker.example/pixel.gif" alt="">`))
	got := string(out)
	if !strings.Contains(got, "attacker.example/pixel.gif") {
		t.Errorf("expected the img src to survive sanitization (network fetch is blocked by CSP, not by the sanitizer), got: %s", got)
	}
}

// TestRenderSanitizedEmail_OrdinaryHTMLSurvives proves the sanitizer is
// non-vacuous: ordinary email HTML constructs (headings, links, styled
// text, lists) must render, not just be safely stripped.
func TestRenderSanitizedEmail_OrdinaryHTMLSurvives(t *testing.T) {
	html := `<h1>Subject line</h1>` +
		`<p style="color: #333333; font-weight: bold;">Hello there</p>` +
		`<a href="http://example.com/page">a link</a>` +
		`<ul><li>item one</li><li>item two</li></ul>`

	out := RenderSanitizedEmail([]byte(html))
	got := string(out)

	checks := []string{"<h1", "Hello there", "href=\"http://example.com/page\"", "<li", "item one"}
	for _, want := range checks {
		if !strings.Contains(got, want) {
			t.Errorf("expected rendered output to contain %q, got: %s", want, got)
		}
	}
}

// TestWrapDocument_InjectsThemeStyleAndPreservesFragment mirrors
// plugins/silverbullet/render_test.go's own regression test: WrapDocument
// must produce a full document carrying the fixed theme stylesheet in
// <head>, with the sanitized fragment unchanged in <body>.
func TestWrapDocument_InjectsThemeStyleAndPreservesFragment(t *testing.T) {
	fragment := []byte("<h1>Subject line</h1><p>hello</p>")
	doc := WrapDocument(fragment)
	got := string(doc)

	if !strings.HasPrefix(got, "<!doctype html>") {
		t.Fatalf("expected the wrapped document to start with a doctype, got: %s", got)
	}
	if !strings.Contains(got, "<style>") {
		t.Fatal("expected a <style> block in the wrapped document")
	}
	for _, want := range []string{"#0f172a", "#f1f5f9", "#60a5fa"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected the injected stylesheet to reference theme token %q, got: %s", want, got)
		}
	}
	if !strings.Contains(got, string(fragment)) {
		t.Errorf("expected the original sanitized fragment to be preserved unchanged in the body, got: %s", got)
	}
	if strings.Contains(got, "@import") || strings.Contains(got, "url(") {
		t.Errorf("expected a fully self-contained stylesheet (no @import/url()), got: %s", got)
	}
}

// TestWrapDocument_StyleNeverReprocessedThroughSanitizer proves the
// wrapping step happens strictly after sanitization: WrapDocument's own
// fixed <style> output is not itself run back through
// emailSanitizePolicy — it doesn't need to be, since it's Go source, never
// derived from message content.
func TestWrapDocument_StyleNeverReprocessedThroughSanitizer(t *testing.T) {
	doc := WrapDocument([]byte("<p>plain</p>"))
	if !strings.Contains(string(doc), "font-family:") {
		t.Errorf("expected the literal stylesheet text to survive unmodified, got: %s", doc)
	}
}
