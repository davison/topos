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

// TestRenderSanitizedEmail_EmailCannotMarkADeclarationImportant proves the
// property that makes the readability fix (below) a proof rather than an
// assumption: bluemonday's style sanitizer re-emits every surviving
// declaration as "property: value" only (dec.Property + ": " + dec.Value)
// — douceur parses the CSS `!important` marker into a separate field that
// bluemonday never writes back — so an email's own inline style can never
// outrank the wrapper stylesheet's important declarations, however the
// email author tries to mark it.
func TestRenderSanitizedEmail_EmailCannotMarkADeclarationImportant(t *testing.T) {
	out := RenderSanitizedEmail([]byte(`<p style="color: #000000 !important;">hi</p>`))
	got := string(out)
	if strings.Contains(got, "!important") {
		t.Errorf("expected no !important marker to survive sanitization, got: %s", got)
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

// TestWrapDocument_NeutralizesEmailSuppliedColours proves the wrapper's
// readability layer (03-09-PLAN.md Task 3, gap G-03-2): the wrapped
// document declares an important foreground colour and an important
// transparent background for the body and its descendants — the
// neutralizer that outranks every email-supplied inline colour and
// background-color that survives sanitization — and still declares the
// readable link, code and blockquote colours, which must survive that
// neutralization by virtue of their own higher-specificity selectors.
func TestWrapDocument_NeutralizesEmailSuppliedColours(t *testing.T) {
	doc := WrapDocument([]byte(`<p style="color: #000000; background-color: #ffffff;">hi</p>`))
	got := string(doc)

	if !strings.Contains(got, "color: #f1f5f9 !important") {
		t.Errorf("expected an important theme foreground colour declaration in the wrapper, got: %s", got)
	}
	if !strings.Contains(got, "background-color: transparent !important") {
		t.Errorf("expected an important transparent background declaration in the wrapper, got: %s", got)
	}
	if !strings.Contains(got, "body, body *") {
		t.Errorf("expected the neutralizer to apply to the body and every descendant, not a hand-picked element list, got: %s", got)
	}
	// The restoring rules (links, code/pre, blockquote) must still
	// declare their own readable theme colours so they survive the
	// neutralizer above.
	for _, want := range []string{"#60a5fa", "#1e293b", "#94a3b8"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected the wrapper to still declare readable theme token %q for links/code/blockquote, got: %s", want, got)
		}
	}
}

// TestWrapDocument_HidesImagesThatCanNeverLoad proves the second half of
// the readability fix: an email body's img elements are hidden with an
// important declaration, because the rendition is served under a CSP
// that permits no subresource of any kind, so an image can never load
// and can only ever paint as a broken-image placeholder.
func TestWrapDocument_HidesImagesThatCanNeverLoad(t *testing.T) {
	doc := WrapDocument([]byte(`<img src="https://example.com/tracker.png">`))
	got := string(doc)
	if !strings.Contains(got, "display: none !important") {
		t.Errorf("expected images to be hidden with an important declaration, got: %s", got)
	}
}

// TestRenderSanitizedEmail_EmptyAndNilInputYieldNoOutput pins the
// input-boundary behaviour of the swapped HTML tokenizer (03-07-PLAN.md,
// CVE-2024-45338 / GO-2024-3333 fix bump): RenderSanitizedEmail called with
// a nil slice must not panic and must return a zero-length result, and
// called with an empty non-nil slice must likewise return a zero-length
// result. A tokenizer swap is exactly the change that can move boundary
// behaviour while leaving every mid-range case intact, and this file
// previously asserted nothing about nil or empty input for either
// RenderSanitizedEmail or WrapDocument.
func TestRenderSanitizedEmail_EmptyAndNilInputYieldNoOutput(t *testing.T) {
	nilOut := RenderSanitizedEmail(nil)
	if len(nilOut) != 0 {
		t.Errorf("expected a nil input to yield a zero-length result, got %d bytes: %q", len(nilOut), nilOut)
	}

	emptyOut := RenderSanitizedEmail([]byte{})
	if len(emptyOut) != 0 {
		t.Errorf("expected an empty non-nil input to yield a zero-length result, got %d bytes: %q", len(emptyOut), emptyOut)
	}
}

// TestWrapDocument_NilFragmentStillYieldsADocument pins WrapDocument's
// nil-input boundary: a nil fragment must not panic, and the theme
// wrapper (doctype, <style> block, structural markers) must still be
// emitted even with no fragment content to wrap. An exact byte length is
// deliberately not asserted — the wrapper's stylesheet is legitimately
// edited by other work, and an exact-length assertion would be brittle
// rather than protective.
func TestWrapDocument_NilFragmentStillYieldsADocument(t *testing.T) {
	doc := WrapDocument(nil)
	got := string(doc)

	if len(got) == 0 {
		t.Fatal("expected WrapDocument(nil) to still emit a non-empty wrapped document")
	}
	if !strings.HasPrefix(got, "<!doctype html>") {
		t.Errorf("expected the wrapped document to start with a doctype even with a nil fragment, got: %s", got)
	}
	if !strings.Contains(got, "<style>") {
		t.Error("expected a <style> block in the wrapped document even with a nil fragment")
	}
	if !strings.Contains(got, "<body>") || !strings.Contains(got, "</body>") {
		t.Errorf("expected the body wrapper markers to still be present with a nil fragment, got: %s", got)
	}
}
