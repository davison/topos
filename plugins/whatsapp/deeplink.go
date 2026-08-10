package main

// conversationDeepLink builds a whatsapp:// deep link at
// LINK_FIDELITY_CONVERSATION_ONLY — the direct analogue of
// plugins/signal/deeplink.go's bare "sgnl://" fallback: it raises the
// desktop WhatsApp client (if one is installed and registered for the
// scheme) without navigating to a particular conversation, an honestly
// conversation-only fidelity and a non-empty value that satisfies
// PLUG-03's sync-time validation (an item with an empty deep_link is
// rejected).
//
// NOT YET hands-on verified: 08-RESEARCH.md Open Question 3 is closed by
// this plan's Task 3 spike (run against this plugin's own real linked
// device), which records whether the bare "whatsapp://" scheme is
// registered on this machine and, if not, which scheme actually raises
// the app — 08-01-SUMMARY.md records that answer for Plan 08-02 to correct
// this function against, if needed.
//
// Deliberately does NOT emit a "https://wa.me/<number>" link: that routes
// through a Meta-hosted redirect, discloses the contact's number to a
// third-party host on click, and contradicts this project's all-data-local
// constraint (PROJECT.md Constraints: "Privacy: All data stays local; no
// personal content leaves the user's machines").
func conversationDeepLink() string {
	return "whatsapp://"
}
