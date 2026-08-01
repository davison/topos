package main

import (
	"net/url"
	"strings"
	"testing"
	"unicode/utf8"

	imap "github.com/emersion/go-imap"

	webspacesv1 "github.com/davison/webspaces/sdk/gen/webspaces/v1"
)

// TestWebmailSearchDeepLink_Table is a URL contract, exact-match table —
// see the plan's must_haves.truths for the behavior each row proves.
func TestWebmailSearchDeepLink_Table(t *testing.T) {
	base := "https://mail.proton.me/u/1"

	tests := []struct {
		name    string
		base    string
		subject string
		want    string
	}{
		{
			name:    "ordinary subject with spaces",
			base:    base,
			subject: "Weekly team sync notes",
			want:    base + "/all-mail#keyword=Weekly%20team%20sync%20notes",
		},
		{
			name:    "absent subject",
			base:    base,
			subject: "",
			want:    base + "/all-mail",
		},
		{
			name:    "empty subject",
			base:    base,
			subject: "",
			want:    base + "/all-mail",
		},
		{
			name:    "whitespace-only subject",
			base:    base,
			subject: "   \t\n  ",
			want:    base + "/all-mail",
		},
		{
			// One fragment marker '#', an ampersand, an equals sign, a path
			// separator and a query marker — every one of those percent-
			// encoded, so the produced URL still carries exactly one
			// fragment marker and the same path segment.
			name:    "hostile punctuation subject",
			base:    base,
			subject: "re: #urgent? a&b=c /path",
			want:    base + "/all-mail#keyword=re%3A%20%23urgent%3F%20a%26b%3Dc%20%2Fpath",
		},
		{
			name:    "base with trailing separator",
			base:    base + "/",
			subject: "trailing base",
			want:    base + "/all-mail#keyword=trailing%20base",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := webmailSearchDeepLink(tt.base, tt.subject)
			if got != tt.want {
				t.Errorf("webmailSearchDeepLink(%q, %q) = %q, want %q", tt.base, tt.subject, got, tt.want)
			}
			// Every produced URL must carry AT MOST one fragment marker —
			// a hostile subject percent-encodes its own '#' rather than
			// introducing a second real one.
			if n := strings.Count(got, "#"); n > 1 {
				t.Errorf("webmailSearchDeepLink(%q, %q) = %q, contains %d fragment markers, want at most 1", tt.base, tt.subject, got, n)
			}
			if !strings.Contains(got, "/"+webmailAllMailSegment) {
				t.Errorf("webmailSearchDeepLink(%q, %q) = %q, want it to contain the All Mail path segment %q", tt.base, tt.subject, got, webmailAllMailSegment)
			}
		})
	}
}

// TestWebmailSearchDeepLink_OverCapMultiByteSubjectStaysValidUTF8 asserts
// the rune cap is applied by RUNE count, never byte count: a subject of
// multi-byte characters longer than the cap must produce a keyword that
// percent-decodes to valid UTF-8 whose rune count is exactly the cap.
func TestWebmailSearchDeepLink_OverCapMultiByteSubjectStaysValidUTF8(t *testing.T) {
	base := "https://mail.proton.me/u/1"
	// "世" is a 3-byte rune; repeating it well past the cap means a
	// byte-count truncation would produce an invalid partial codepoint at
	// the boundary, while a rune-count truncation never does.
	subject := strings.Repeat("世", deepLinkKeywordRuneCap+50)

	got := webmailSearchDeepLink(base, subject)

	const marker = "#keyword="
	idx := strings.Index(got, marker)
	if idx == -1 {
		t.Fatalf("webmailSearchDeepLink(%q, over-cap subject) = %q, want a %q fragment", base, got, marker)
	}
	encodedKeyword := got[idx+len(marker):]

	decoded, err := url.QueryUnescape(encodedKeyword)
	if err != nil {
		t.Fatalf("url.QueryUnescape(%q): %v", encodedKeyword, err)
	}

	if !utf8.ValidString(decoded) {
		t.Fatalf("decoded keyword %q is not valid UTF-8 (byte-truncated mid-codepoint)", decoded)
	}
	if gotRunes := utf8.RuneCountInString(decoded); gotRunes != deepLinkKeywordRuneCap {
		t.Errorf("decoded keyword rune count = %d, want %d (the cap)", gotRunes, deepLinkKeywordRuneCap)
	}
}

// TestToItem_DeepLinkIsAWebmailSearchNotALabelPath asserts toItem builds
// DeepLink from the constructor over the envelope's own subject, and that
// the result never contains the matched label's leaf name — the
// assertion that would catch a partial fix that merely appended a search
// fragment onto the old label path.
func TestToItem_DeepLinkIsAWebmailSearchNotALabelPath(t *testing.T) {
	plugin, err := NewSourcePlugin("imap://bridge.invalid:143", "username", "password", "", "https://mail.proton.me/u/0")
	if err != nil {
		t.Fatalf("NewSourcePlugin: %v", err)
	}

	m := &matched{
		envelope: &imap.Envelope{
			Subject:   "House move update",
			MessageId: "<house-move-update@example.com>",
		},
		mailbox: "Labels/House Move",
		labels:  []string{"House Move"},
	}

	item := plugin.toItem("test-source-id", m)

	want := webmailSearchDeepLink(plugin.webmailBaseURL, m.envelope.Subject)
	if got := item.GetDeepLink(); got != want {
		t.Errorf("item.DeepLink = %q, want %q (the constructor's own output for this subject)", got, want)
	}
	if strings.Contains(item.GetDeepLink(), "House Move") {
		t.Errorf("item.DeepLink = %q, must not contain the label leaf name %q", item.GetDeepLink(), "House Move")
	}
}

// TestToItem_FidelityRemainsAnchored asserts the fidelity declaration is
// unchanged by this plan: the link lands adjacent to the message, not on
// it, so it stays ANCHORED — asserted, not assumed.
func TestToItem_FidelityRemainsAnchored(t *testing.T) {
	plugin, err := NewSourcePlugin("imap://bridge.invalid:143", "username", "password", "", "https://mail.proton.me/u/0")
	if err != nil {
		t.Fatalf("NewSourcePlugin: %v", err)
	}

	m := &matched{
		envelope: &imap.Envelope{
			Subject:   "House move update",
			MessageId: "<house-move-update@example.com>",
		},
		mailbox: "Labels/House Move",
		labels:  []string{"House Move"},
	}

	item := plugin.toItem("test-source-id", m)

	if got := item.GetFidelity(); got != webspacesv1.LinkFidelity_LINK_FIDELITY_ANCHORED {
		t.Errorf("item.Fidelity = %v, want LINK_FIDELITY_ANCHORED", got)
	}
}
