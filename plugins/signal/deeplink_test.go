package main

import "testing"

func TestDeepLink_Group(t *testing.T) {
	got := conversationDeepLink("group", "")
	if got != "sgnl://" {
		t.Errorf("expected the bare scheme for a group, got %q", got)
	}
}

func TestDeepLink_PrivateWithE164UsesContactForm(t *testing.T) {
	got := conversationDeepLink("private", "+15551234567")
	want := "sgnl://signal.me/#p/" + encodePhoneFragment("+15551234567")
	if got != want {
		t.Errorf("expected the contact form with the E.164 escaped, got %q, want %q", got, want)
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

func TestDeepLink_UnsafeCharactersAreEscapedNotEmittedRaw(t *testing.T) {
	// Not a realistic E.164, but proves the escaping discipline holds
	// for any URI-unsafe character the source might ever hand this
	// function, rather than assuming E.164 values are always simple.
	unsafe := "+1 555#123&456"
	got := conversationDeepLink("private", unsafe)
	if got == "sgnl://signal.me/#p/"+unsafe {
		t.Fatalf("expected the unsafe value to be escaped, got it emitted raw: %q", got)
	}
	want := "sgnl://signal.me/#p/" + encodePhoneFragment(unsafe)
	if got != want {
		t.Errorf("expected the escaped form, got %q, want %q", got, want)
	}
	// The raw unsafe characters (space, #, &) must never appear verbatim
	// in the emitted fragment.
	for _, c := range []string{" ", "#", "&"} {
		frag := got[len("sgnl://signal.me/#p/"):]
		if len(frag) > 0 && containsRaw(frag, c) {
			t.Errorf("expected character %q to be escaped, found raw in fragment %q", c, frag)
		}
	}
}

func containsRaw(s, substr string) bool {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestEncodePhoneFragment_PlusSignEscaped(t *testing.T) {
	got := encodePhoneFragment("+15551234567")
	if got != "%2B15551234567" {
		t.Errorf("expected the leading + to be percent-encoded, got %q", got)
	}
}
