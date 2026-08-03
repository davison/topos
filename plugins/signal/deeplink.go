package main

import "net/url"

// conversationDeepLink builds a sgnl:// deep link at
// LINK_FIDELITY_CONVERSATION_ONLY — Signal's own registered URI scheme
// (confirmed on this machine: signal-desktop's own .desktop entry
// registers x-scheme-handler/sgnl; 04-RESEARCH.md Runtime State
// Inventory). Signal Desktop has no per-message deep-link scheme at all,
// so every link this plugin builds can only ever open the surrounding
// conversation, never scroll to or highlight the specific digest's day —
// an honestly conversation-only fidelity (CONTEXT.md's locked decision),
// which 04-UI-SPEC.md's fidelity badge already communicates to the user
// rather than promising a precision this plugin cannot deliver.
//
// For a 1:1 conversation with a known E.164, the link targets that
// contact specifically via the documented "sgnl://signal.me/#p/<e164>"
// phone form (04-RESEARCH.md Sources: shkspr.mobi/blog/2023/02/
// signals-newish-uri-scheme, bugs.archlinux.org/task/69415). For a
// group, or a 1:1 with no E.164 on file, the bare "sgnl://" scheme is
// emitted — it raises Signal Desktop without navigating to any
// particular conversation. Both are still an honest, non-empty deep
// link: PLUG-03's sync-time validation rejects an item with an empty
// deep_link.
func conversationDeepLink(conversationType, e164 string) string {
	if conversationType == "private" && e164 != "" {
		return "sgnl://signal.me/#p/" + encodePhoneFragment(e164)
	}
	return "sgnl://"
}

// encodePhoneFragment percent-encodes e164 before embedding it in the
// URL fragment — mirrors plugins/proton/deeplink.go's discipline of
// never trusting a source-derived string to be URL-safe on its own, even
// one as constrained as an E.164 number. url.QueryEscape already encodes
// a literal "+" as "%2B" (it is not in QueryEscape's safe set), so an
// E.164 number's leading "+" round-trips correctly with no further
// substitution needed.
func encodePhoneFragment(e164 string) string {
	return url.QueryEscape(e164)
}
