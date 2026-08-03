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
//
// 04-RESEARCH.md assumption A4, closed 2026-08-03: both forms were
// invoked against this machine's installed, running Signal Desktop via
// its own registered scheme handler (`gio open`, confirmed the handler
// is `signal.desktop` via `gio mime x-scheme-handler/sgnl` /
// `xdg-mime query default x-scheme-handler/sgnl`). The bare "sgnl://"
// form's behavior was already visually confirmed by the developer during
// 04-01-PLAN.md's own human-verify checkpoint (04-01-SUMMARY.md: raises
// Signal Desktop, does not navigate to a specific conversation for a
// group — the accepted, intended conversation-only fidelity limit, not a
// defect). This task additionally invoked the
// "sgnl://signal.me/#p/<e164>" contact form: `gio open` returned exit 0
// with no error, and Signal Desktop's own single-instance-lock IPC
// handoff was directly observable (its startup diagnostic output printed
// as the second launch attempt detected the running instance and forwarded
// the URI to it) — consistent with the handler correctly receiving and
// routing the request. Both forms survive; no fallback substitution was
// needed. A pixel-level "did the correct contact's window actually raise
// and focus" re-confirmation for the contact form specifically is
// deferred to this phase's end-of-phase human verification pass
// (workflow.human_verify_mode = "end-of-phase" — 04-01's checkpoint
// already set this precedent for the bare form).
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
