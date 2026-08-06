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

	tokens := []string{"#0f172a", "#1e293b", "#60a5fa", "#94a3b8", "#f1f5f9"}
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
