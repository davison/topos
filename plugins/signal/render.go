// Chat-transcript HTML renderer: the source-agnostic bubble/run
// transcript builder Phase 5's WhatsApp plugin reuses (04-03-PLAN.md
// Task 2). Mirrors plugins/proton/body.go's sanitize-then-wrap pipeline
// shape — a package-level bluemonday policy built once at init, message
// text sanitized individually before assembly, and a fixed,
// self-contained themeStyle applied strictly AFTER sanitization — but
// this is its OWN policy variable and its OWN wrap helper: the content
// shape (message bubbles grouped into sender runs) differs from an
// email body, so nothing here shares state with plugins/proton/body.go.
package main

import (
	"bytes"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"
)

// ownSenderLabel is the fixed display name readMessages/buildSenderNames
// (plugin.go) already resolve the account's own messages to. Both
// buildMessageRuns (below, for own-vs-other alignment) and plugin.go's
// own resolution logic key off this single constant so the two can never
// drift out of sync.
const ownSenderLabel = "You"

// runGapThreshold is the maximum gap between two consecutive messages
// from the same sender that still collapses them into one run
// (04-UI-SPEC.md `## UI Considerations` E2 `populated` row: "a sender
// change or a >5 min gap starts a new run").
const runGapThreshold = 5 * time.Minute

// deletedMessageCopy and editedSuffix are the exact, fixed copy
// 04-UI-SPEC.md's Copywriting Contract specifies for a tombstone bubble
// and an edited-message suffix, respectively.
const (
	deletedMessageCopy = "This message was deleted"
	editedSuffix        = "(edited)"
)

// signalTranscriptSanitizePolicy is built once at init (bluemonday's own
// documented pattern for concurrent-safe reuse — the identical rationale
// plugins/proton/body.go's emailSanitizePolicy and
// plugins/silverbullet/render.go's sanitizePolicy already rely on). Built
// from bluemonday.UGCPolicy() and deliberately NOT widened at all: unlike
// plugins/proton/body.go's emailSanitizePolicy (which narrowly allows a
// presentational style attribute on a named element set), message text
// here is untrusted third-party chat content with no legitimate styling
// need, so no style attribute is ever allowed on any element — every
// bubble, run and chrome element's presentation comes exclusively from
// signalThemeStyle below, applied strictly after this policy runs.
var signalTranscriptSanitizePolicy = newSignalTranscriptSanitizePolicy()

func newSignalTranscriptSanitizePolicy() *bluemonday.Policy {
	return bluemonday.UGCPolicy()
}

// sanitizeText sanitizes s (a single untrusted text field — a message
// body, a quoted excerpt, an attachment filename, a reactor or sender
// display name) via signalTranscriptSanitizePolicy on its own, BEFORE it
// is concatenated into the assembled transcript's markup. Every
// interpolated string this file writes into HTML goes through this
// function — never the assembled document as a whole — so a crafted
// value can never break out of its own element boundary before the
// sanitizer sees it (mirrors plugins/proton/body.go's identical
// per-part-not-per-document discipline).
func sanitizeText(s string) string {
	return string(signalTranscriptSanitizePolicy.SanitizeBytes([]byte(s)))
}

// messageRun is a maximal run of consecutive messages from the SAME
// sender, no two of which are more than runGapThreshold apart
// (04-UI-SPEC.md E2 `populated`). The sender name renders once at the
// run's top (own runs render no name at all — ownership is signalled by
// alignment/background alone, never text or colour); the timestamp
// renders once, at the end of the run's LAST bubble.
type messageRun struct {
	SenderName string
	IsOwn      bool
	Messages   []messageRecord
}

// buildMessageRuns groups chronologically-sorted-ascending msgs into
// runs: a new run starts whenever the sender differs from the previous
// message's sender, or the gap since the previous message exceeds
// runGapThreshold — never on any other signal (no day boundary check
// here: callers already scope msgs to a single day, per D-04).
func buildMessageRuns(msgs []messageRecord) []messageRun {
	var runs []messageRun
	gapMs := runGapThreshold.Milliseconds()

	for _, m := range msgs {
		if n := len(runs); n > 0 {
			last := &runs[n-1]
			prev := last.Messages[len(last.Messages)-1]
			sameSender := last.SenderName == m.SenderName
			withinGap := m.SentAtUnixMs-prev.SentAtUnixMs <= gapMs
			if sameSender && withinGap {
				last.Messages = append(last.Messages, m)
				continue
			}
		}
		runs = append(runs, messageRun{
			SenderName: m.SenderName,
			IsOwn:      m.SenderName == ownSenderLabel,
			Messages:   []messageRecord{m},
		})
	}
	return runs
}

// formatTimestamp renders sentAtUnixMs as a local wall-clock time
// ("3:04 PM") — the transcript never spans more than one calendar day by
// construction (D-04), so no date component is ever needed here; the
// pane's own header already shows the digest's date (04-UI-SPEC.md E2
// `populated`: "no day-header divider is rendered").
func formatTimestamp(sentAtUnixMs int64) string {
	return time.UnixMilli(sentAtUnixMs).In(time.Local).Format("3:04 PM")
}

// renderTranscript renders msgs (chronologically ascending, already
// scoped to one conversation's one day) into a sanitized HTML fragment —
// NOT yet a complete document; callers pass this to WrapDocument. Each
// message's own text fields are sanitized individually via sanitizeText
// before being concatenated into the run/bubble markup below — this
// function never sanitizes the assembled result.
func renderTranscript(msgs []messageRecord) []byte {
	var buf bytes.Buffer
	for _, run := range buildMessageRuns(msgs) {
		align := "other"
		if run.IsOwn {
			align = "own"
		}

		buf.WriteString(`<div class="run ` + align + `">`)
		if !run.IsOwn {
			buf.WriteString(`<div class="sender-name">` + sanitizeText(run.SenderName) + `</div>`)
		}
		for i, m := range run.Messages {
			buf.WriteString(renderBubble(m, align, i == len(run.Messages)-1))
		}
		buf.WriteString(`</div>`)
	}
	return buf.Bytes()
}

// renderBubble renders one message as a bubble div. showTimestamp is
// true only for the run's LAST message (04-UI-SPEC.md E2 `populated`).
// Field order within the bubble — quoted excerpt, then body (or the
// deleted tombstone in its place), then attachment placeholder chips,
// then reaction line(s) — is fixed by 04-03-PLAN.md Task 2's own action
// text.
func renderBubble(m messageRecord, align string, showTimestamp bool) string {
	var b strings.Builder
	b.WriteString(`<div class="bubble ` + align + `">`)

	if m.QuoteExcerpt != "" {
		b.WriteString(`<div class="quote">` + sanitizeText(m.QuoteExcerpt) + `</div>`)
	}

	switch {
	case m.Deleted:
		// Sender/timestamp chrome is unchanged by deletion — only the
		// body content is replaced, never omitted from the bubble
		// entirely (04-UI-SPEC.md E2 `partial`).
		b.WriteString(`<div class="tombstone">` + deletedMessageCopy + `</div>`)
	case m.Body != "":
		b.WriteString(`<div class="body">` + sanitizeText(m.Body) + `</div>`)
	}

	for _, att := range m.Attachments {
		b.WriteString(`<div class="attachment">` + sanitizeText(attachmentPlaceholder(att)) + `</div>`)
	}
	for _, line := range reactionLines(m.Reactions) {
		b.WriteString(`<div class="reaction">` + sanitizeText(line) + `</div>`)
	}

	switch {
	case showTimestamp:
		// The edited suffix is appended directly after the timestamp
		// with a single space, on the SAME line — never its own line —
		// per 04-UI-SPEC.md Typography's exact rule. Only the run's
		// last bubble ever carries a timestamp, so this is the only
		// place the suffix can be co-located with one.
		ts := sanitizeText(formatTimestamp(m.SentAtUnixMs))
		if m.Edited {
			ts += " " + editedSuffix
		}
		b.WriteString(`<div class="timestamp">` + ts + `</div>`)
	case m.Edited:
		// A non-last bubble in a run carries no timestamp element at
		// all — its own edited status still renders, just without a
		// timestamp to attach to.
		b.WriteString(`<div class="edited-suffix">` + editedSuffix + `</div>`)
	}

	b.WriteString(`</div>`)
	return b.String()
}

// signalThemeStyle is a fixed, self-contained stylesheet — no external
// fetch, no @import, no url() reference of any kind — applied strictly
// AFTER sanitization (WrapDocument never re-runs its input through
// signalTranscriptSanitizePolicy), so it can never reintroduce anything
// sanitization removed. Base typography and the image-hiding rule copy
// plugins/proton/body.go's/plugins/silverbullet/render.go's existing
// rendered-content precedent verbatim (15px/1.6, the same system font
// stack) — deliberately NOT the SPA's Inter/16px chrome (04-UI-SPEC.md
// Typography).
//
// Every colour value below is an EXISTING theme token (04-UI-SPEC.md
// Color) — no new hex value is introduced by this file. The accent
// `#60a5fa` appears exactly once, reserved for inline links inside
// message text (the sanitize policy's own <a> allowance) — it is never
// applied to a bubble background, a sender name, a timestamp, or any
// other per-participant differentiation, protecting the accent budget
// exactly as Phase 3 protected it for search (04-UI-SPEC.md Color).
const signalThemeStyle = `
:root { color-scheme: dark; }
html, body {
  margin: 0;
  padding: 16px;
  background: #0f172a;
  color: #f1f5f9;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
  font-size: 15px;
  line-height: 1.6;
}
a { color: #60a5fa; text-decoration: underline; }
a:hover { color: #93c5fd; }
/* A message can never carry a remote image reference that actually
   loads — the rendition is served under a policy permitting no
   subresource of any kind (kernel/httpapi/item.go's Content-Security-
   Policy) — so any surviving <img> could only ever render broken. */
img { display: none !important; }

.run { display: flex; flex-direction: column; margin: 16px 0; }
.run.own { align-items: flex-end; }
.run.other { align-items: flex-start; }
.run .bubble + .bubble { margin-top: 4px; }

.sender-name { font-size: 14px; font-weight: 600; color: #f1f5f9; margin-bottom: 4px; }

.bubble {
  max-width: 75%;
  border-radius: 12px;
  padding: 8px 16px;
}
.bubble.other { background: #1e293b; }
.bubble.own { background: #0f172a; border: 1px solid #1e293b; }

.bubble .body { font-size: 15px; font-weight: 400; line-height: 1.6; white-space: pre-wrap; overflow-wrap: anywhere; }
.bubble .tombstone { font-size: 15px; line-height: 1.6; font-style: italic; color: #94a3b8; }
.bubble .quote {
  font-size: 14px; font-weight: 400; font-style: italic; color: #94a3b8;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  margin-bottom: 4px;
}
.bubble .timestamp, .bubble .edited-suffix { font-size: 14px; font-weight: 400; color: #94a3b8; margin-top: 4px; }
.bubble .attachment {
  display: inline-block; font-size: 14px; font-weight: 400; color: #f1f5f9;
  background: #334155; border-radius: 999px; padding: 4px 12px; margin-top: 4px;
}
.bubble .reaction { font-size: 14px; font-weight: 400; color: #94a3b8; margin-top: 4px; }
`

// WrapDocument wraps an already-sanitized HTML fragment
// (renderTranscript's output) in a minimal, self-contained HTML document
// — doctype, a <head> carrying only a charset meta tag and the fixed
// signalThemeStyle above, and the fragment as <body> content unchanged.
// Copies plugins/proton/body.go's WrapDocument shape verbatim — see that
// function's own doc comment for why sanitizing must always happen
// BEFORE this call, never after.
func WrapDocument(sanitizedFragment []byte) []byte {
	var buf bytes.Buffer
	buf.WriteString("<!doctype html>\n<html><head><meta charset=\"utf-8\"><style>")
	buf.WriteString(signalThemeStyle)
	buf.WriteString("</style></head><body>")
	buf.Write(sanitizedFragment)
	buf.WriteString("</body></html>")
	return buf.Bytes()
}
