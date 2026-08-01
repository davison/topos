package main

import (
	"net/url"
	"strings"
	"unicode/utf8"
)

// webmailAllMailSegment is the fixed, name-addressable Proton webmail
// system-view path segment this plugin's deep link targets. Proton
// addresses SYSTEM views (inbox, all-mail, sent, drafts, ...) by name, but
// custom labels and folders only by an internal id this plugin has no way
// to resolve — so a system view is the only name-addressable target
// available to a plugin with no id mapping, and All Mail is the correct
// one: a matched message may live in any folder, label, archive or trash
// view, and All Mail is guaranteed to contain it regardless of which
// mailbox the match came from.
const webmailAllMailSegment = "all-mail"

// deepLinkKeywordRuneCap bounds the search keyword's length in RUNES,
// mirroring body.go's Snippet — the package's existing rune-cap
// precedent. A cap exists at all because a pathological subject would
// otherwise produce a URL long enough to be truncated by a browser or a
// link handler, which fails silently rather than loudly.
const deepLinkKeywordRuneCap = 500

// encodeKeywordFragment percent-encodes s with net/url's query escaper,
// then replaces every plus sign the escaper produces for a space with the
// percent-encoded space "%20" instead. A fragment may be read either by a
// form-style parser (which decodes a plus as a space) or by a straight
// percent-decoder (which does not) — "%20" is the single form both decode
// identically. The escaper is also what makes a hostile subject inert: it
// percent-encodes every fragment, query, parameter and path character
// (#, &, =, /, ?, ...) that could otherwise restructure the URL it is
// embedded in.
func encodeKeywordFragment(s string) string {
	escaped := url.QueryEscape(s)
	return strings.ReplaceAll(escaped, "+", "%20")
}

// webmailSearchDeepLink builds a link into webmailBaseURL's All Mail view,
// optionally pre-filled with a search for subject. webmailBaseURL is
// trimmed of any trailing separator before joining, so a base supplied
// either with or without one produces an identical result. subject is
// trimmed before testing and using it; when the trimmed subject has no
// renderable content (absent, empty, or whitespace-only — reusing
// body.go's HasRenderableText, the package's one definition of "is there
// anything here", rather than re-testing for emptiness), the returned
// link carries no fragment at all — never a search for the empty string.
// Otherwise the trimmed subject is capped to deepLinkKeywordRuneCap
// RUNES (never bytes, so a multi-byte subject is never cut mid-codepoint,
// exactly as Snippet does), percent-encoded, and appended as a fragment
// keyword parameter.
func webmailSearchDeepLink(webmailBaseURL, subject string) string {
	base := strings.TrimRight(webmailBaseURL, "/")
	link := base + "/" + webmailAllMailSegment

	trimmed := strings.TrimSpace(subject)
	if !HasRenderableText(trimmed) {
		return link
	}

	capped := trimmed
	if utf8.RuneCountInString(capped) > deepLinkKeywordRuneCap {
		runes := []rune(capped)
		capped = string(runes[:deepLinkKeywordRuneCap])
	}

	return link + "#keyword=" + encodeKeywordFragment(capped)
}
