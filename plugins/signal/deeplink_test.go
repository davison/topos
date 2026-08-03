package main

import (
	"strings"
	"testing"
)

func TestDeepLink_PlusSignEmittedLiterally(t *testing.T) {
	got := conversationDeepLink("private", "+15551234567")
	want := "sgnl://signal.me/#p/+15551234567"
	if got != want {
		t.Errorf("expected the literal '+' form, got %q, want %q", got, want)
	}
	if strings.Contains(got, "%") {
		t.Errorf("expected no percent sign in the emitted link, got %q", got)
	}
}

func TestDeepLink_Group(t *testing.T) {
	got := conversationDeepLink("group", "")
	if got != "sgnl://" {
		t.Errorf("expected the bare scheme for a group, got %q", got)
	}
}

func TestDeepLink_PrivateWithE164UsesContactForm(t *testing.T) {
	got := conversationDeepLink("private", "+15551234567")
	want := "sgnl://signal.me/#p/+15551234567"
	if got != want {
		t.Errorf("expected the contact form with the E.164 emitted verbatim, got %q, want %q", got, want)
	}
}

func TestDeepLink_PrivateWithoutE164FallsBackToBareForm(t *testing.T) {
	got := conversationDeepLink("private", "")
	if got != "sgnl://" {
		t.Errorf("expected a 1:1 with no known E.164 to fall back to the same bare form a group gets, got %q", got)
	}
}

func TestDeepLink_NeverEmpty(t *testing.T) {
	for _, tc := range []struct {
		conversationType, e164 string
	}{
		{"group", ""},
		{"private", ""},
		{"private", "+15551234567"},
		{"", ""},
	} {
		if got := conversationDeepLink(tc.conversationType, tc.e164); got == "" {
			t.Errorf("expected a non-empty deep_link for type=%q e164=%q (PLUG-03 rejects an empty deep_link at sync time)", tc.conversationType, tc.e164)
		}
	}
}

func TestDeepLink_NonE164FallsBackToBareForm(t *testing.T) {
	// Not a realistic E.164, but proves the refusal discipline holds for
	// any URI-metacharacter-bearing value the source might ever hand this
	// function, rather than assuming E.164 values are always simple. A
	// value carrying URI metacharacters is not an E.164, so it must be
	// refused entry — never escaped into the fragment.
	unsafe := "+1 555#123&456"
	got := conversationDeepLink("private", unsafe)
	want := "sgnl://"
	if got != want {
		t.Errorf("expected the non-E.164 value to fall back to the bare form, got %q, want %q", got, want)
	}
}
