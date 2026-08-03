package main

import (
	"strings"
	"testing"
)

func TestRenderTranscript_ConsecutiveSameSenderWithinFiveMinutesOneRun(t *testing.T) {
	msgs := []messageRecord{
		rec("c1", "Alice", "hi", localMs(2026, 8, 1, 9, 0)),
		rec("c1", "Alice", "how are you", localMs(2026, 8, 1, 9, 2)),
	}
	doc := string(renderTranscript(msgs))
	if strings.Count(doc, `class="sender-name"`) != 1 {
		t.Errorf("expected exactly one sender-name element for one run, got document: %s", doc)
	}
	if strings.Count(doc, "Alice") != 1 {
		t.Errorf("expected the sender name to appear exactly once, got document: %s", doc)
	}
}

func TestRenderTranscript_SenderChangeStartsNewRun(t *testing.T) {
	msgs := []messageRecord{
		rec("c1", "Alice", "hi", localMs(2026, 8, 1, 9, 0)),
		rec("c1", "Bob", "hello", localMs(2026, 8, 1, 9, 1)),
	}
	doc := string(renderTranscript(msgs))
	if strings.Count(doc, `class="sender-name"`) != 2 {
		t.Errorf("expected two runs (two sender-name elements), got document: %s", doc)
	}
	if !strings.Contains(doc, "Alice") || !strings.Contains(doc, "Bob") {
		t.Errorf("expected both sender names present, got document: %s", doc)
	}
}

func TestRenderTranscript_GapOverFiveMinutesStartsNewRunSameSender(t *testing.T) {
	msgs := []messageRecord{
		rec("c1", "Alice", "hi", localMs(2026, 8, 1, 9, 0)),
		rec("c1", "Alice", "still there?", localMs(2026, 8, 1, 9, 6)),
	}
	doc := string(renderTranscript(msgs))
	if strings.Count(doc, `class="sender-name"`) != 2 {
		t.Errorf("expected a >5 minute gap from the same sender to start a new run, got document: %s", doc)
	}
}

func TestRenderTranscript_GapUnderFiveMinutesSameRun(t *testing.T) {
	msgs := []messageRecord{
		rec("c1", "Alice", "hi", localMs(2026, 8, 1, 9, 0)),
		rec("c1", "Alice", "you there?", localMs(2026, 8, 1, 9, 5)),
	}
	doc := string(renderTranscript(msgs))
	if strings.Count(doc, `class="sender-name"`) != 1 {
		t.Errorf("expected a <=5 minute gap from the same sender to stay in one run, got document: %s", doc)
	}
}

func TestRenderTranscript_OwnMessagesCarryNoSenderNameLabel(t *testing.T) {
	msgs := []messageRecord{
		rec("c1", ownSenderLabel, "hi", localMs(2026, 8, 1, 9, 0)),
	}
	doc := string(renderTranscript(msgs))
	if strings.Contains(doc, `class="sender-name"`) {
		t.Errorf("expected no sender-name element for an own-message run, got document: %s", doc)
	}
	if !strings.Contains(doc, `class="run own"`) {
		t.Errorf("expected the run to carry the own class, got document: %s", doc)
	}
}

func TestRenderTranscript_DeletedMessageRendersTombstoneNotBody(t *testing.T) {
	deleted := rec("c1", "Alice", "", localMs(2026, 8, 1, 9, 0))
	deleted.Deleted = true
	doc := string(renderTranscript([]messageRecord{deleted}))
	if !strings.Contains(doc, deletedMessageCopy) {
		t.Errorf("expected the tombstone copy, got document: %s", doc)
	}
	if strings.Contains(doc, `class="body"`) {
		t.Errorf("expected no body element for a deleted message, got document: %s", doc)
	}
}

func TestRenderTranscript_EditedMessageSuffixOnSameLineAsTimestamp(t *testing.T) {
	m := rec("c1", "Alice", "the latest text", localMs(2026, 8, 1, 9, 0))
	m.Edited = true
	doc := string(renderTranscript([]messageRecord{m}))
	if !strings.Contains(doc, "the latest text") {
		t.Errorf("expected the latest text to render, got document: %s", doc)
	}
	// A single message is trivially the last (and only) bubble of its
	// run — the timestamp element must carry the suffix directly.
	idx := strings.Index(doc, `class="timestamp"`)
	if idx == -1 {
		t.Fatalf("expected a timestamp element, got document: %s", doc)
	}
	closeIdx := strings.Index(doc[idx:], "</div>")
	if closeIdx == -1 {
		t.Fatalf("malformed timestamp element, got document: %s", doc)
	}
	timestampElement := doc[idx : idx+closeIdx]
	if !strings.Contains(timestampElement, editedSuffix) {
		t.Errorf("expected %q on the same line (element) as the timestamp, got %q", editedSuffix, timestampElement)
	}
}

func TestRenderTranscript_AttachmentRendersPlaceholderWithFilename(t *testing.T) {
	m := rec("c1", "Alice", "", localMs(2026, 8, 1, 9, 0))
	m.Attachments = []attachment{{Filename: "report.pdf"}}
	doc := string(renderTranscript([]messageRecord{m}))
	if !strings.Contains(doc, "📎 report.pdf") {
		t.Errorf("expected the attachment placeholder with filename, got document: %s", doc)
	}
}

func TestRenderTranscript_ReactionsRenderEmojiAndCommaJoinedNames(t *testing.T) {
	m := rec("c1", "Alice", "great news", localMs(2026, 8, 1, 9, 0))
	m.Reactions = []reaction{{Emoji: "👍", ReactorName: "Bob"}, {Emoji: "👍", ReactorName: "Carol"}}
	doc := string(renderTranscript([]messageRecord{m}))
	if !strings.Contains(doc, "👍 Bob, Carol") {
		t.Errorf("expected the grouped reaction line, got document: %s", doc)
	}
}

func TestRenderTranscript_QuotedReplyRendersAboveOwnText(t *testing.T) {
	m := rec("c1", "Alice", "yes exactly", localMs(2026, 8, 1, 9, 0))
	m.QuoteExcerpt = "what time works?"
	doc := string(renderTranscript([]messageRecord{m}))
	quoteIdx := strings.Index(doc, "what time works?")
	bodyIdx := strings.Index(doc, "yes exactly")
	if quoteIdx == -1 || bodyIdx == -1 {
		t.Fatalf("expected both the quote excerpt and the reply's own text, got document: %s", doc)
	}
	if quoteIdx > bodyIdx {
		t.Errorf("expected the quoted excerpt to render ABOVE the replying message's own text")
	}
}

func TestRenderTranscript_ScriptAndEventHandlerStrippedBySanitization(t *testing.T) {
	m := rec("c1", "Alice", `<script>alert(1)</script><img src=x onerror="alert(2)">hi`, localMs(2026, 8, 1, 9, 0))
	fragment := string(renderTranscript([]messageRecord{m}))
	doc := string(WrapDocument([]byte(fragment)))
	if strings.Contains(doc, "<script") || strings.Contains(strings.ToLower(doc), "onerror") {
		t.Errorf("expected the script element and event-handler attribute to be stripped, got document: %s", doc)
	}
}

func TestWrapDocument_CompleteSelfContainedDocument(t *testing.T) {
	fragment := renderTranscript([]messageRecord{rec("c1", "Alice", "hi", localMs(2026, 8, 1, 9, 0))})
	doc := string(WrapDocument(fragment))
	if !strings.HasPrefix(doc, "<!doctype html>") {
		t.Errorf("expected the document to begin with a doctype, got: %s", doc[:min(40, len(doc))])
	}
	if !strings.Contains(doc, `<meta charset="utf-8">`) {
		t.Error("expected the charset meta tag")
	}
	if !strings.Contains(doc, "<style>") {
		t.Error("expected the stylesheet")
	}
	if strings.Contains(doc, "url(") || strings.Contains(doc, "@import") || strings.Contains(doc, "http://") || strings.Contains(doc, "https://") {
		t.Errorf("expected the stylesheet to carry no external reference of any kind, got: %s", doc)
	}
}

func TestRenderTranscript_SanitizationHappensBeforeWrapping(t *testing.T) {
	// A policy-stripped construct (a script element) must never reappear
	// in the wrapped output — proven by wrapping the ALREADY-sanitized
	// fragment renderTranscript produces, never a raw unsanitized string.
	m := rec("c1", "Alice", `<script>alert(1)</script>hi`, localMs(2026, 8, 1, 9, 0))
	fragment := renderTranscript([]messageRecord{m})
	doc := string(WrapDocument(fragment))
	if strings.Contains(doc, "<script") {
		t.Errorf("expected sanitization to have already stripped the script element before wrapping, got: %s", doc)
	}
}

func TestRenderTranscript_NoAccentColourOnBubbleSenderOrTimestamp(t *testing.T) {
	// The accent hex may appear ONLY on the link rule (a { color:
	// #60a5fa }) in the stylesheet — never associated with a bubble
	// background, sender-name rule or timestamp rule.
	for _, selector := range []string{".bubble.own", ".bubble.other", ".sender-name", ".timestamp"} {
		idx := strings.Index(signalThemeStyle, selector)
		if idx == -1 {
			t.Fatalf("expected selector %q in signalThemeStyle", selector)
		}
		end := strings.Index(signalThemeStyle[idx:], "}")
		rule := signalThemeStyle[idx : idx+end]
		if strings.Contains(rule, "#60a5fa") {
			t.Errorf("expected selector %q to never use the accent colour, got rule: %s", selector, rule)
		}
	}
}

func TestRenderTranscript_ZeroMessagesProducesNoRuns(t *testing.T) {
	doc := string(renderTranscript(nil))
	if strings.Contains(doc, `class="run`) {
		t.Errorf("expected no run elements for zero messages, got: %s", doc)
	}
}

func TestBuildMessageRuns_OrderPreserved(t *testing.T) {
	msgs := []messageRecord{
		rec("c1", "Alice", "1", localMs(2026, 8, 1, 9, 0)),
		rec("c1", "Alice", "2", localMs(2026, 8, 1, 9, 1)),
		rec("c1", "Bob", "3", localMs(2026, 8, 1, 9, 2)),
	}
	runs := buildMessageRuns(msgs)
	if len(runs) != 2 {
		t.Fatalf("expected 2 runs, got %d", len(runs))
	}
	if len(runs[0].Messages) != 2 || len(runs[1].Messages) != 1 {
		t.Errorf("unexpected run message counts: %+v", runs)
	}
}
