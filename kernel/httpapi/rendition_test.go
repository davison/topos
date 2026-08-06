// rendition_test.go relocates every assertion that used to live in
// plugins/proton/render_test.go, plugins/silverbullet/render_test.go and
// plugins/signal/render_test.go against their now-removed RenderSanitized*/
// WrapDocument helpers (D-11) — this is a MOVE, not a rewrite: the sanitize/
// wrap/theme pipeline those three files implemented independently now lives
// in rendition.go, and this file is where its own test coverage lives.
package httpapi

import (
	"os"
	"strings"
	"testing"
	"unicode/utf8"

	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

// --- sanitizeAndWrapRendition: shared/structural behavior ---

// TestSanitizeAndWrapRendition_UnrecognisedShapeReturnsErrorNoBytes proves
// the kernel fails closed (T-05-16): an unrecognised or unspecified shape
// — including the zero value, CONTENT_SHAPE_UNSPECIFIED — returns an error
// and no bytes, never a guessed policy.
func TestSanitizeAndWrapRendition_UnrecognisedShapeReturnsErrorNoBytes(t *testing.T) {
	for _, shape := range []toposv1.ContentShape{
		toposv1.ContentShape_CONTENT_SHAPE_UNSPECIFIED,
		toposv1.ContentShape(99), // a value with no entry in renditionPolicies at all
	} {
		out, err := sanitizeAndWrapRendition(shape, []byte("<p>hello</p>"), nil)
		if err == nil {
			t.Errorf("shape %v: expected an error, got nil", shape)
		}
		if len(out) != 0 {
			t.Errorf("shape %v: expected zero bytes on error, got %d bytes: %q", shape, len(out), out)
		}
	}
}

// TestSanitizeAndWrapRendition_StripsRawScriptElement proves the sanitizer
// removes a raw <script> element and any event-handler attribute — the
// primary XSS control this pipeline provides, checked against every
// recognised shape.
func TestSanitizeAndWrapRendition_StripsRawScriptElement(t *testing.T) {
	for _, shape := range []toposv1.ContentShape{
		toposv1.ContentShape_CONTENT_SHAPE_EMAIL_HTML,
		toposv1.ContentShape_CONTENT_SHAPE_MARKDOWN_HTML,
		toposv1.ContentShape_CONTENT_SHAPE_CHAT_TRANSCRIPT,
	} {
		out, err := sanitizeAndWrapRendition(shape, []byte(`<p>hello</p><script>alert(1)</script><img src="x" onerror="alert(2)">`), nil)
		if err != nil {
			t.Fatalf("shape %v: sanitizeAndWrapRendition: %v", shape, err)
		}
		got := strings.ToLower(string(out))
		if strings.Contains(got, "<script") {
			t.Errorf("shape %v: expected no <script element to survive sanitization, got: %s", shape, got)
		}
		if strings.Contains(got, "onerror=") {
			t.Errorf("shape %v: expected no onerror attribute to survive sanitization, got: %s", shape, got)
		}
	}
}

// TestSanitizeAndWrapRendition_StripsJavascriptSchemeHref proves a
// javascript-scheme anchor href does not survive sanitization, for every
// recognised shape.
func TestSanitizeAndWrapRendition_StripsJavascriptSchemeHref(t *testing.T) {
	for _, shape := range []toposv1.ContentShape{
		toposv1.ContentShape_CONTENT_SHAPE_EMAIL_HTML,
		toposv1.ContentShape_CONTENT_SHAPE_MARKDOWN_HTML,
		toposv1.ContentShape_CONTENT_SHAPE_CHAT_TRANSCRIPT,
	} {
		out, err := sanitizeAndWrapRendition(shape, []byte(`<a href="javascript:alert(1)">click me</a>`), nil)
		if err != nil {
			t.Fatalf("shape %v: sanitizeAndWrapRendition: %v", shape, err)
		}
		got := strings.ToLower(string(out))
		if strings.Contains(got, "javascript:") {
			t.Errorf("shape %v: expected no javascript: scheme to survive in an href, got: %s", shape, got)
		}
	}
}

// TestSanitizeAndWrapRendition_StyleNeverReprocessedThroughSanitizer proves
// the wrapping step happens strictly after sanitization: the composed
// stylesheet's own fixed <style> output is not itself run back through the
// shape's policy — it doesn't need to be, since it's Go source, never
// derived from fragment content.
func TestSanitizeAndWrapRendition_StyleNeverReprocessedThroughSanitizer(t *testing.T) {
	out, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_EMAIL_HTML, []byte("<p>plain</p>"), nil)
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition: %v", err)
	}
	if !strings.Contains(string(out), "font-family:") {
		t.Errorf("expected the literal stylesheet text to survive unmodified, got: %s", out)
	}
}

// TestSanitizeAndWrapRendition_InjectsThinThemeMatchedScrollbar is the
// relocated regression test for the live-UAT-found gap (Quick task
// 260805-j98 follow-up), now checked once against the shared base rather
// than three times against three copies.
func TestSanitizeAndWrapRendition_InjectsThinThemeMatchedScrollbar(t *testing.T) {
	for _, shape := range []toposv1.ContentShape{
		toposv1.ContentShape_CONTENT_SHAPE_EMAIL_HTML,
		toposv1.ContentShape_CONTENT_SHAPE_MARKDOWN_HTML,
		toposv1.ContentShape_CONTENT_SHAPE_CHAT_TRANSCRIPT,
	} {
		doc, err := sanitizeAndWrapRendition(shape, []byte("<p>hello</p>"), nil)
		if err != nil {
			t.Fatalf("shape %v: sanitizeAndWrapRendition: %v", shape, err)
		}
		got := string(doc)
		if !strings.Contains(got, "scrollbar-width: thin") {
			t.Errorf("shape %v: expected scrollbar-width: thin, got: %s", shape, got)
		}
		if !strings.Contains(got, "scrollbar-color:") {
			t.Errorf("shape %v: expected scrollbar-color declared, got: %s", shape, got)
		}
		if !strings.Contains(got, "::-webkit-scrollbar-thumb") {
			t.Errorf("shape %v: expected the WebKit pseudo-element fallback, got: %s", shape, got)
		}
		if !strings.Contains(got, "148, 163, 184") {
			t.Errorf("shape %v: expected the scrollbar color to resolve from --muted-foreground (rgb 148,163,184), got: %s", shape, got)
		}
	}
}

// TestSanitizeAndWrapRendition_NoExternalReferenceOfAnyKind proves the
// wrapped document contains no external reference of any kind — no import
// rule, no url() token, no src attribute pointing off-document (T-05-18).
func TestSanitizeAndWrapRendition_NoExternalReferenceOfAnyKind(t *testing.T) {
	for _, shape := range []toposv1.ContentShape{
		toposv1.ContentShape_CONTENT_SHAPE_EMAIL_HTML,
		toposv1.ContentShape_CONTENT_SHAPE_MARKDOWN_HTML,
		toposv1.ContentShape_CONTENT_SHAPE_CHAT_TRANSCRIPT,
	} {
		doc, err := sanitizeAndWrapRendition(shape, []byte("<p>hello</p>"), nil)
		if err != nil {
			t.Fatalf("shape %v: sanitizeAndWrapRendition: %v", shape, err)
		}
		got := string(doc)
		if strings.Contains(got, "@import") || strings.Contains(got, "url(") {
			t.Errorf("shape %v: expected a fully self-contained stylesheet (no @import/url()), got: %s", shape, got)
		}
	}
}

// TestSanitizeAndWrapRendition_NilFragmentStillYieldsADocument pins the
// nil-input boundary: a nil fragment must not panic, and the wrapper
// (doctype, <style> block, structural markers) must still be emitted.
func TestSanitizeAndWrapRendition_NilFragmentStillYieldsADocument(t *testing.T) {
	doc, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_MARKDOWN_HTML, nil, nil)
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition(nil): %v", err)
	}
	got := string(doc)
	if len(got) == 0 {
		t.Fatal("expected a non-empty wrapped document for a nil fragment")
	}
	if !strings.HasPrefix(got, "<!doctype html>") {
		t.Errorf("expected the wrapped document to start with a doctype, got: %s", got)
	}
	if !strings.Contains(got, "<body></body>") {
		t.Errorf("expected empty but present body markers for a nil fragment, got: %s", got)
	}
}

// --- email content shape ---

func TestEmailShape_PreservesColorDropsPosition(t *testing.T) {
	out, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_EMAIL_HTML,
		[]byte(`<p style="color: #ff0000; position: fixed; top: 0;">styled text</p>`), nil)
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition: %v", err)
	}
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

func TestEmailShape_StyleAttributeScopedToNamedElements(t *testing.T) {
	out, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_EMAIL_HTML,
		[]byte(`<input style="color: red;" value="x">`), nil)
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition: %v", err)
	}
	// The fragment's own style attribute must be stripped; the wrapper's
	// OWN <style> block legitimately contains "style=" nowhere but does
	// contain style rules, so assert against the fragment's own markup
	// shape rather than the whole document string.
	if strings.Contains(string(out), `<input style=`) {
		t.Errorf("expected style attribute to be stripped from an element outside the named allowlist, got: %s", out)
	}
}

func TestEmailShape_RemoteImagePreservedButHidden(t *testing.T) {
	// bluemonday's UGCPolicy allows <img src> to survive sanitization (the
	// tracking-pixel defence lives in the kernel's rendition CSP, not in
	// stripping the element) — but the email stylesheet delta hides all
	// images outright, so the element is present yet never painted.
	out, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_EMAIL_HTML,
		[]byte(`<img src="https://attacker.example/pixel.gif" alt="">`), nil)
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "attacker.example/pixel.gif") {
		t.Errorf("expected the img src to survive sanitization (blocked by CSP, not stripped), got: %s", got)
	}
	if !strings.Contains(got, "img { display: none !important; }") {
		t.Errorf("expected the email profile to hide images, got: %s", got)
	}
}

func TestEmailShape_OrdinaryHTMLSurvives(t *testing.T) {
	html := `<h1>Subject line</h1>` +
		`<p style="color: #333333; font-weight: bold;">Hello there</p>` +
		`<a href="http://example.com/page">a link</a>` +
		`<ul><li>item one</li><li>item two</li></ul>`

	out, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_EMAIL_HTML, []byte(html), nil)
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition: %v", err)
	}
	got := string(out)

	checks := []string{"<h1", "Hello there", "href=\"http://example.com/page\"", "<li", "item one"}
	for _, want := range checks {
		if !strings.Contains(got, want) {
			t.Errorf("expected rendered output to contain %q, got: %s", want, got)
		}
	}
}

// TestEmailShape_EmailCannotMarkADeclarationImportant proves the property
// that makes the email readability layer a proof rather than an
// assumption: bluemonday's style sanitizer re-emits every surviving
// declaration as "property: value" only — douceur parses the CSS
// !important marker into a separate field bluemonday never writes back —
// so an email's own inline style can never outrank the wrapper
// stylesheet's important declarations, however the email author tries to
// mark it.
func TestEmailShape_EmailCannotMarkADeclarationImportant(t *testing.T) {
	out, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_EMAIL_HTML,
		[]byte(`<p style="color: #000000 !important;">hi</p>`), nil)
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition: %v", err)
	}
	// The sanitized fragment's OWN inline style must never carry
	// !important — the wrapper's readability-layer !important
	// declarations (asserted separately below) are a different, trusted
	// source.
	bodyStart := strings.Index(string(out), "<body>")
	if bodyStart == -1 {
		t.Fatalf("expected a <body> marker, got: %s", out)
	}
	fragment := string(out)[bodyStart:]
	if strings.Contains(fragment, "!important") {
		t.Errorf("expected no !important marker to survive sanitization in the fragment, got: %s", fragment)
	}
}

// TestEmailShape_NeutralizesEmailSuppliedColours proves the wrapper's
// readability layer (03-09-PLAN.md Task 3, gap G-03-2): the wrapped
// document declares an important foreground colour and an important
// transparent background for the body and its descendants — the
// neutralizer that outranks every email-supplied inline colour and
// background-color that survives sanitization — and still declares the
// readable link, code and blockquote colours.
func TestEmailShape_NeutralizesEmailSuppliedColours(t *testing.T) {
	out, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_EMAIL_HTML,
		[]byte(`<p style="color: #000000; background-color: #ffffff;">hi</p>`), nil)
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition: %v", err)
	}
	got := string(out)

	if !strings.Contains(got, "color: #f1f5f9 !important") {
		t.Errorf("expected an important theme foreground colour declaration in the wrapper, got: %s", got)
	}
	if !strings.Contains(got, "background-color: transparent !important") {
		t.Errorf("expected an important transparent background declaration in the wrapper, got: %s", got)
	}
	if !strings.Contains(got, "body, body *") {
		t.Errorf("expected the neutralizer to apply to the body and every descendant, got: %s", got)
	}
	for _, want := range []string{"#60a5fa", "#1e293b", "#94a3b8"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected the wrapper to still declare readable theme token %q for links/code/blockquote, got: %s", want, got)
		}
	}
}

// --- email vs markdown image divergence ---

// TestImagePolicy_EmailHidesMarkdownAllows proves the genuine per-shape
// divergence the policy table must preserve, not collapse: the email
// profile hides images outright while the markdown profile allows them at
// full container width.
func TestImagePolicy_EmailHidesMarkdownAllows(t *testing.T) {
	email, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_EMAIL_HTML, []byte(`<img src="https://example.com/tracker.png">`), nil)
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition(email): %v", err)
	}
	if !strings.Contains(string(email), "img { display: none !important; }") {
		t.Errorf("expected the email profile to hide images with an important declaration, got: %s", email)
	}

	markdown, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_MARKDOWN_HTML, []byte(`<img src="https://example.com/diagram.png">`), nil)
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition(markdown): %v", err)
	}
	if !strings.Contains(string(markdown), "img { max-width: 100%; }") {
		t.Errorf("expected the markdown profile to allow images at full container width, got: %s", markdown)
	}
	if strings.Contains(string(markdown), "img { display: none") {
		t.Errorf("expected the markdown profile to NOT hide images, got: %s", markdown)
	}
}

// --- markdown content shape ---

func TestMarkdownShape_OrdinaryHTMLSurvives(t *testing.T) {
	html := `<h1>Title</h1><p>hello <strong>world</strong></p><pre><code>code block</code></pre>` +
		`<a href="http://example.com/page">a link</a><ul><li>item one</li></ul>`

	out, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_MARKDOWN_HTML, []byte(html), nil)
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition: %v", err)
	}
	got := string(out)
	for _, want := range []string{"<h1", "<strong", "href=\"http://example.com/page\"", "<li", "code block"} {
		if !strings.Contains(got, want) {
			t.Errorf("expected rendered output to contain %q, got: %s", want, got)
		}
	}
}

func TestMarkdownShape_StyleAttributeNeverAllowed(t *testing.T) {
	// Unlike the email shape, the markdown policy allows no style
	// attribute at all — plain bluemonday.UGCPolicy().
	out, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_MARKDOWN_HTML,
		[]byte(`<p style="color: red;">hi</p>`), nil)
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition: %v", err)
	}
	bodyStart := strings.Index(string(out), "<body>")
	fragment := string(out)[bodyStart:]
	if strings.Contains(fragment, "style=") {
		t.Errorf("expected no style attribute to survive the markdown policy, got: %s", fragment)
	}
}

// --- chat content shape ---

// TestChatShape_NoAccentColourOnBubbleSenderOrTimestamp is the relocated
// regression test for plugins/signal/render_test.go's
// TestRenderTranscript_NoAccentColourOnBubbleSenderOrTimestamp: the accent
// hex may appear ONLY on the shared base's link rule (a { color: #60a5fa
// }) — never associated with a bubble background, sender-name rule or
// timestamp rule.
func TestChatShape_NoAccentColourOnBubbleSenderOrTimestamp(t *testing.T) {
	style := stylesheetForShape(toposv1.ContentShape_CONTENT_SHAPE_CHAT_TRANSCRIPT)
	for _, selector := range []string{".bubble.own", ".bubble.other", ".sender-name", ".timestamp"} {
		idx := strings.Index(style, selector)
		if idx == -1 {
			t.Fatalf("expected selector %q in the chat stylesheet", selector)
		}
		end := strings.Index(style[idx:], "}")
		if end == -1 {
			t.Fatalf("malformed rule for selector %q", selector)
		}
		rule := style[idx : idx+end]
		if strings.Contains(rule, "#60a5fa") {
			t.Errorf("expected selector %q to never use the accent colour, got rule: %s", selector, rule)
		}
	}
}

// TestChatShape_TombstoneQuoteAttachmentReactionRulesPresent proves the
// chat profile's partial/degraded message sub-state rules are carried
// forward verbatim into the kernel-owned stylesheet (05-UI-SPEC.md E2
// `partial`).
func TestChatShape_TombstoneQuoteAttachmentReactionRulesPresent(t *testing.T) {
	style := stylesheetForShape(toposv1.ContentShape_CONTENT_SHAPE_CHAT_TRANSCRIPT)
	for _, selector := range []string{".bubble .tombstone", ".bubble .quote", ".bubble .attachment", ".bubble .reaction", ".bubble .body"} {
		if !strings.Contains(style, selector) {
			t.Errorf("expected the chat stylesheet to declare %q, got: %s", selector, style)
		}
	}
	if !strings.Contains(style, "overflow-wrap: anywhere") {
		t.Errorf("expected the chat bubble body to wrap long text via overflow-wrap: anywhere, got: %s", style)
	}
}

// TestChatShape_LegitimateStructuralClassesSurvive proves the class
// attribute policy allows the transcript markup's own combinations to pass
// through unchanged.
func TestChatShape_LegitimateStructuralClassesSurvive(t *testing.T) {
	fragment := `<div class="run own"><div class="bubble own"><div class="body">hi</div>` +
		`<div class="timestamp">3:04 PM</div></div></div>`
	out, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_CHAT_TRANSCRIPT, []byte(fragment), nil)
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition: %v", err)
	}
	got := string(out)
	for _, want := range []string{`class="run own"`, `class="bubble own"`, `class="body"`, `class="timestamp"`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected legitimate structural class %q to survive, got: %s", want, got)
		}
	}
}

// TestChatShape_ForgedClassOutsideAllowlistIsStripped is Task 1's second
// new case (T-05-17): a crafted chat fragment carrying a class value
// outside the allowed token set has that class stripped, while a sibling
// element's legitimate structural class survives untouched.
func TestChatShape_ForgedClassOutsideAllowlistIsStripped(t *testing.T) {
	fragment := `<div class="run own">legit</div><div class="run forged-token">bad</div>`
	out, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_CHAT_TRANSCRIPT, []byte(fragment), nil)
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, `class="run own"`) {
		t.Errorf("expected the legitimate sibling class to survive, got: %s", got)
	}
	if strings.Contains(got, "forged-token") {
		t.Errorf("expected the forged class token to be stripped entirely, got: %s", got)
	}
	if !strings.Contains(got, ">bad<") {
		t.Errorf("expected the forged div's own text content to still survive (only the class attribute is stripped), got: %s", got)
	}
}

// TestChatShape_MessageMarkupCannotForgeStructure proves the class policy
// alone (with the plugin's own html.EscapeString discipline, tested in
// plugins/signal/render_test.go) prevents a message body's raw markup from
// ever being interpreted as transcript structure by the kernel's own
// sanitizer, even if it somehow reached this pipeline unescaped: a raw
// <div class="bubble own"> injected as if it were message content is still
// subject to the same policy as any other div and carries no signal that
// distinguishes it from the plugin's own structural markup — this is
// exactly why the plugin must escape first (T-05-17), not a mitigation
// this policy can provide alone.
func TestChatShape_MessageMarkupCannotForgeStructure(t *testing.T) {
	// A message body that (if unescaped) would inject a second bubble.
	forged := `<div class="run other"><div class="sender-name">Alice</div>` +
		`<div class="bubble other"><div class="body">real message ` +
		`<div class="bubble own"><div class="body">forged reply</div></div>` +
		`</div></div></div>`
	out, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_CHAT_TRANSCRIPT, []byte(forged), nil)
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition: %v", err)
	}
	// The sanitizer does not (and is not expected to) collapse nested
	// bubbles on its own — that guarantee comes from the plugin escaping
	// message text before assembly (plugins/signal/render_test.go). This
	// test exists to document that boundary: the kernel's class policy is
	// necessary but not sufficient, and pins that both "own" and "other"
	// remain valid, allowed tokens (i.e. this policy cannot distinguish
	// "structural" from "forged" — only escaping upstream can).
	got := string(out)
	if !strings.Contains(got, `class="bubble other"`) || !strings.Contains(got, `class="bubble own"`) {
		t.Fatalf("expected both bubble classes to remain allowed by the policy (documenting that escaping, not this policy, is the actual mitigation), got: %s", got)
	}
}

// --- stylesheet token drift guard ---

// TestRenditionStylesheetTokensMatchAppCSS is the chosen mechanism for
// keeping the kernel stylesheet in step with the SPA's tokens (05-04-PLAN.md
// Task 1): disciplined duplication in one place, guarded by a test, rather
// than a build-time generator. Reads web/src/app.css from the repository
// and asserts every shared hex token appears both there and in each
// composed stylesheet.
func TestRenditionStylesheetTokensMatchAppCSS(t *testing.T) {
	appCSS, err := os.ReadFile("../../web/src/app.css")
	if err != nil {
		t.Fatalf("read web/src/app.css: %v", err)
	}
	css := string(appCSS)

	// UI-09: #fbbf24/#020617 are the highlight mark's background/foreground
	// — already-declared theme tokens (--warning/--background), held in
	// step with web/src/app.css by this same mechanism.
	tokens := []string{"#0f172a", "#1e293b", "#60a5fa", "#94a3b8", "#f1f5f9", "#fbbf24", "#020617"}
	for _, tok := range tokens {
		if !strings.Contains(css, tok) {
			t.Errorf("expected web/src/app.css to declare token %q", tok)
		}
	}

	for _, shape := range []toposv1.ContentShape{
		toposv1.ContentShape_CONTENT_SHAPE_EMAIL_HTML,
		toposv1.ContentShape_CONTENT_SHAPE_MARKDOWN_HTML,
		toposv1.ContentShape_CONTENT_SHAPE_CHAT_TRANSCRIPT,
	} {
		style := stylesheetForShape(shape)
		for _, tok := range tokens {
			if !strings.Contains(style, tok) {
				t.Errorf("shape %v: expected the composed stylesheet to reference theme token %q, got: %s", shape, tok, style)
			}
		}
	}
}

// --- highlighting (UI-09) ---
//
// This section proves the kernel highlighter (Task 1, rendition.go)
// touches rendered text only — never attributes, never tag bytes, never
// the chat class allowlist, never a second sanitizer pass — and pins the
// term-derivation rule's exact behaviour. Every test here calls
// sanitizeAndWrapRendition directly, exactly like every pre-existing test
// above.

// TestHighlight_EmptyTermsInert proves nil and an empty (but non-nil)
// terms slice produce byte-identical output containing no mark element,
// for every recognised content shape — the no-search path must stay
// byte-identical to the pre-UI-09 output regardless of which "no terms"
// representation a caller passes.
func TestHighlight_EmptyTermsInert(t *testing.T) {
	for _, shape := range []toposv1.ContentShape{
		toposv1.ContentShape_CONTENT_SHAPE_EMAIL_HTML,
		toposv1.ContentShape_CONTENT_SHAPE_MARKDOWN_HTML,
		toposv1.ContentShape_CONTENT_SHAPE_CHAT_TRANSCRIPT,
	} {
		fragment := []byte(`<p>hello world</p>`)
		withNil, err := sanitizeAndWrapRendition(shape, fragment, nil)
		if err != nil {
			t.Fatalf("shape %v: sanitizeAndWrapRendition(nil terms): %v", shape, err)
		}
		withEmpty, err := sanitizeAndWrapRendition(shape, fragment, []string{})
		if err != nil {
			t.Fatalf("shape %v: sanitizeAndWrapRendition(empty terms): %v", shape, err)
		}
		if string(withNil) != string(withEmpty) {
			t.Errorf("shape %v: expected nil and empty terms to produce byte-identical output, got:\nnil:   %s\nempty: %s", shape, withNil, withEmpty)
		}
		if strings.Contains(string(withNil), "<mark") {
			t.Errorf("shape %v: expected no <mark> element with no terms, got: %s", shape, withNil)
		}
	}
}

// TestHighlight_TextNodesOnly_AttributesUntouched proves a fragment whose
// anchor href and title attribute both contain the search term produces a
// mark inside the anchor's visible text only — both attribute values
// survive completely unchanged (T-06-01).
func TestHighlight_TextNodesOnly_AttributesUntouched(t *testing.T) {
	fragment := []byte(`<a href="https://example.com/needle" title="needle">visible needle text</a>`)
	out, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_MARKDOWN_HTML, fragment, []string{"needle"})
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, `href="https://example.com/needle"`) {
		t.Errorf("expected the href attribute value to survive unchanged, got: %s", got)
	}
	if !strings.Contains(got, `title="needle"`) {
		t.Errorf("expected the title attribute value to survive unchanged, got: %s", got)
	}
	if !strings.Contains(got, "<mark>needle</mark>") {
		t.Errorf("expected the anchor's visible text to gain a mark element, got: %s", got)
	}
}

// TestHighlight_TagBytesUntouched proves a fragment where the term
// collides with an element or attribute name/value produces no mark
// inside any tag — the highlighter only ever scans html.TextNode Data, so
// a term matching "div"/"body"/"own" can only ever highlight those
// strings when they appear as genuine rendered text, never as markup.
// Uses the chat shape's real class-token vocabulary (T-05-17) as the
// collision source, since those tokens are both real attribute values on
// this fragment AND absent from its own text content.
func TestHighlight_TagBytesUntouched(t *testing.T) {
	fragment := []byte(`<div class="run own"><div class="bubble own"><div class="body">some content here</div></div></div>`)
	out, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_CHAT_TRANSCRIPT, fragment, []string{"div", "own", "body"})
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition: %v", err)
	}
	got := string(out)
	if strings.Contains(got, "<mark") {
		t.Errorf("expected no <mark> element when terms only match tag/attribute names and values, not text content, got: %s", got)
	}
	for _, want := range []string{`<div class="run own">`, `<div class="bubble own">`, `<div class="body">`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected the element structure to be fully preserved, missing %q, got: %s", want, got)
		}
	}
}

// TestHighlight_ChatClassAllowlistSurvives is the direct regression guard
// for RESEARCH.md Pitfall 2: a chat-transcript fragment carrying the real
// class tokens keeps every class attribute after highlighting, and the
// highlight lands inside the bubble body text.
func TestHighlight_ChatClassAllowlistSurvives(t *testing.T) {
	fragment := `<div class="run own"><div class="bubble own"><div class="body">hello needle world</div></div></div>`
	out, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_CHAT_TRANSCRIPT, []byte(fragment), []string{"needle"})
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition: %v", err)
	}
	got := string(out)
	for _, want := range []string{`class="run own"`, `class="bubble own"`, `class="body"`} {
		if !strings.Contains(got, want) {
			t.Errorf("expected class attribute %q to survive highlighting, got: %s", want, got)
		}
	}
	if !strings.Contains(got, `<div class="body">hello <mark>needle</mark> world</div>`) {
		t.Errorf("expected the highlight to land inside the bubble body text, got: %s", got)
	}
}

// TestHighlight_NoReSanitization proves a fragment the policy would strip
// is still stripped, and a mark inserted by the highlighter still
// survives to the output — the highlighted tree is never fed back
// through policy.SanitizeBytes.
func TestHighlight_NoReSanitization(t *testing.T) {
	fragment := []byte(`<script>alert(1)</script><p>hello needle world</p>`)
	out, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_MARKDOWN_HTML, fragment, []string{"needle"})
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition: %v", err)
	}
	got := string(out)
	if strings.Contains(strings.ToLower(got), "<script") {
		t.Errorf("expected the script element to remain stripped, got: %s", got)
	}
	if !strings.Contains(got, "<mark>needle</mark>") {
		t.Errorf("expected the highlighter's own mark insertion to survive to the output, got: %s", got)
	}
}

// TestHighlight_MultiByteSafety proves a fragment containing multi-byte
// runes adjacent to a match renders back with those runes intact —
// highlightTextNode's rune-based (never byte-index-based) scan is what
// guarantees this.
func TestHighlight_MultiByteSafety(t *testing.T) {
	fragment := []byte(`<p>héllo wörld 日本語 needle 表情</p>`)
	out, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_MARKDOWN_HTML, fragment, []string{"needle"})
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition: %v", err)
	}
	got := string(out)
	if !utf8.ValidString(got) {
		t.Fatal("expected valid UTF-8 output")
	}
	if !strings.Contains(got, "héllo wörld 日本語") {
		t.Errorf("expected multi-byte runes preceding the match to survive intact, got: %s", got)
	}
	if !strings.Contains(got, "表情") {
		t.Errorf("expected multi-byte runes following the match to survive intact, got: %s", got)
	}
	if !strings.Contains(got, "<mark>needle</mark>") {
		t.Errorf("expected the ASCII term to still be highlighted, got: %s", got)
	}
}

// TestHighlight_CaseInsensitiveOriginalCasingPreserved proves a lowercase
// term matches mixed-case text and the marked text keeps the document's
// own original casing.
func TestHighlight_CaseInsensitiveOriginalCasingPreserved(t *testing.T) {
	fragment := []byte(`<p>The Needle is here</p>`)
	out, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_MARKDOWN_HTML, fragment, []string{"needle"})
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition: %v", err)
	}
	got := string(out)
	if !strings.Contains(got, "<mark>Needle</mark>") {
		t.Errorf("expected the matched text to preserve its original document casing inside the mark element, got: %s", got)
	}
}

// TestHighlight_ElementBoundaryBackstop is the must_haves backstop row: a
// term split across two adjacent inline elements degrades to no highlight
// for that occurrence — highlightTextNodes only ever scans a single text
// node — and the document's element structure stays byte-for-byte
// identical to the unhighlighted render. This test asserts only the
// non-corruption half; under-highlighting across an element boundary is
// an accepted, deliberate limitation of single-text-node matching, not a
// defect.
func TestHighlight_ElementBoundaryBackstop(t *testing.T) {
	fragment := []byte(`<p><em>nee</em><em>dle</em></p>`)
	highlighted, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_MARKDOWN_HTML, fragment, []string{"needle"})
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition(highlighted): %v", err)
	}
	unhighlighted, err := sanitizeAndWrapRendition(toposv1.ContentShape_CONTENT_SHAPE_MARKDOWN_HTML, fragment, nil)
	if err != nil {
		t.Fatalf("sanitizeAndWrapRendition(unhighlighted): %v", err)
	}
	if string(highlighted) != string(unhighlighted) {
		t.Errorf("expected a term split across two adjacent inline elements to degrade to no highlight (structurally identical output), got:\nhighlighted:   %s\nunhighlighted: %s", highlighted, unhighlighted)
	}
	if strings.Contains(string(highlighted), "<mark") {
		t.Errorf("expected no mark element for an occurrence split across element boundaries, got: %s", highlighted)
	}
}

// TestHighlightTerms_Derivation is a direct table test of highlightTerms:
// whitespace splitting, sub-2-character drop, de-duplication, the 8-term
// cap, lowercasing, and empty/whitespace input returning nothing.
func TestHighlightTerms_Derivation(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{"whitespace splitting", "hello   world\tfoo\nbar", []string{"hello", "world", "foo", "bar"}},
		{"sub-2-character drop", "a bb c dd", []string{"bb", "dd"}},
		{"de-duplication", "hello world hello", []string{"hello", "world"}},
		{"8-term cap", "one two three four five six seven eight nine ten",
			[]string{"one", "two", "three", "four", "five", "six", "seven", "eight"}},
		{"empty input", "", nil},
		{"whitespace-only input", "   \t\n  ", nil},
		{"lowercases", "HELLO World", []string{"hello", "world"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := highlightTerms(tt.raw)
			if len(got) != len(tt.want) {
				t.Fatalf("highlightTerms(%q) = %v, want %v", tt.raw, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("highlightTerms(%q)[%d] = %q, want %q", tt.raw, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestHighlight_EmailStylesheetMarkSelectorIsImportant asserts the email
// shape's composed stylesheet carries a mark selector marked important —
// the restoring rule that keeps the email readability neutraliser from
// silently swallowing the highlight (renditionEmailReadabilityDelta).
func TestHighlight_EmailStylesheetMarkSelectorIsImportant(t *testing.T) {
	style := stylesheetForShape(toposv1.ContentShape_CONTENT_SHAPE_EMAIL_HTML)
	idx := strings.Index(style, "body mark")
	if idx == -1 {
		t.Fatalf("expected a 'body mark' selector in the email shape's composed stylesheet, got: %s", style)
	}
	end := strings.Index(style[idx:], "}")
	if end == -1 {
		t.Fatalf("malformed rule for the 'body mark' selector")
	}
	rule := style[idx : idx+end]
	if !strings.Contains(rule, "!important") {
		t.Errorf("expected the 'body mark' restoring rule to carry an !important marker, got rule: %s", rule)
	}
}
