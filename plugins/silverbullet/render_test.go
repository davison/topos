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
