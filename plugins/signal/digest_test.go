package main

import (
	"strings"
	"testing"
	"time"
)

// localMs builds an epoch-millisecond timestamp for the given local wall
// clock time — using time.Local throughout (matching digest.go's own
// D-04 convention) so these tests are correct regardless of the test
// runner's actual system timezone.
func localMs(year int, month time.Month, day, hour, min int) int64 {
	return time.Date(year, month, day, hour, min, 0, 0, time.Local).UnixMilli()
}

func TestBuildDigests_TwoLocalDaysProduceTwoDigests(t *testing.T) {
	msgs := []message{
		{ConversationID: "c1", SentAtUnixMs: localMs(2026, 8, 1, 9, 0), SenderName: "Dad", Body: "morning"},
		{ConversationID: "c1", SentAtUnixMs: localMs(2026, 8, 2, 9, 0), SenderName: "Dad", Body: "next day"},
	}
	got := buildDigests(msgs, map[string]string{"c1": "Dad"})
	if len(got) != 2 {
		t.Fatalf("expected 2 digests for messages spanning two local days, got %d: %+v", len(got), got)
	}
}

func TestBuildDigests_TimestampIsDaysLastMessageNotFirst(t *testing.T) {
	first := localMs(2026, 8, 1, 8, 0)
	last := localMs(2026, 8, 1, 22, 0)
	msgs := []message{
		{ConversationID: "c1", SentAtUnixMs: first, SenderName: "Dad", Body: "morning"},
		{ConversationID: "c1", SentAtUnixMs: last, SenderName: "Dad", Body: "night"},
	}
	got := buildDigests(msgs, map[string]string{"c1": "Dad"})
	if len(got) != 1 {
		t.Fatalf("expected 1 digest, got %d", len(got))
	}
	if got[0].LastMessageUnix != last/1000 {
		t.Errorf("expected timestamp to be the day's LAST message (%d), got %d", last/1000, got[0].LastMessageUnix)
	}
}

func TestSourceIDForDigest_DeterministicAndRoundTrips(t *testing.T) {
	id1 := sourceIDForDigest("c1", "2026-08-01")
	id2 := sourceIDForDigest("c1", "2026-08-01")
	if id1 != id2 {
		t.Fatalf("expected sourceIDForDigest to be deterministic, got %q and %q", id1, id2)
	}

	convID, day, err := decodeSourceID(id1)
	if err != nil {
		t.Fatalf("decodeSourceID: %v", err)
	}
	if convID != "c1" || day != "2026-08-01" {
		t.Errorf("expected round-trip to (c1, 2026-08-01), got (%q, %q)", convID, day)
	}
}

func TestSourceIDForDigest_DifferentDaysProduceDifferentIDs(t *testing.T) {
	id1 := sourceIDForDigest("c1", "2026-08-01")
	id2 := sourceIDForDigest("c1", "2026-08-02")
	if id1 == id2 {
		t.Fatalf("expected different days to produce different source_ids, both were %q", id1)
	}
}

func TestSourceIDForDigest_SameDayDifferentConversationsProduceDifferentIDs(t *testing.T) {
	id1 := sourceIDForDigest("c1", "2026-08-01")
	id2 := sourceIDForDigest("c2", "2026-08-01")
	if id1 == id2 {
		t.Fatalf("expected different conversations to produce different source_ids, both were %q", id1)
	}
}

func TestDigestTitle_SingularAndPluralGrammar(t *testing.T) {
	if got := digestTitle("Dad", 1); got != "Dad — 1 message" {
		t.Errorf("expected singular grammar, got %q", got)
	}
	if got := digestTitle("Dad", 2); got != "Dad — 2 messages" {
		t.Errorf("expected plural grammar for 2, got %q", got)
	}
	if got := digestTitle("Dad", 0); got != "Dad — 0 messages" {
		t.Errorf("expected plural grammar for 0, got %q", got)
	}
}

func TestTailSnippet_LastMessagesChronologicalSenderPrefixedNoEarlierLeak(t *testing.T) {
	msgs := []message{
		{SenderName: "Alice", Body: "one", SentAtUnixMs: 1},
		{SenderName: "Bob", Body: "two", SentAtUnixMs: 2},
		{SenderName: "Alice", Body: "three", SentAtUnixMs: 3},
		{SenderName: "Bob", Body: "four", SentAtUnixMs: 4},
	}
	got := tailSnippet(msgs)
	want := "Bob: two\nAlice: three\nBob: four"
	if got != want {
		t.Errorf("expected tail snippet %q, got %q", want, got)
	}
	if strings.Contains(got, "one") {
		t.Errorf("expected the tail snippet to exclude messages before the tail, got %q", got)
	}
}

func TestTailSnippet_TruncatesByRuneCountNotByteCount(t *testing.T) {
	// A multi-byte rune (é is 2 bytes in UTF-8) repeated well past
	// previewRuneCap runes — truncating by byte count would cut mid-rune
	// and corrupt the string; truncating by rune count never does.
	long := strings.Repeat("é", previewRuneCap+50)
	msgs := []message{{SenderName: "A", Body: long, SentAtUnixMs: 1}}
	got := tailSnippet(msgs)
	if !strings.HasPrefix(got, "A: ") {
		t.Fatalf("expected sender prefix, got %q", got[:min(20, len(got))])
	}
	// The truncated result must itself be valid UTF-8 (no partial rune)
	// and its rune count must not exceed previewRuneCap.
	body := strings.TrimPrefix(got, "A: ")
	runeCount := 0
	for range body {
		runeCount++
	}
	if runeCount > previewRuneCap {
		t.Errorf("expected at most %d runes, got %d", previewRuneCap, runeCount)
	}
	if !strings.HasSuffix(body, "é") && runeCount > 0 {
		t.Errorf("expected truncation to land on a full rune boundary, got trailing bytes %q", body[len(body)-4:])
	}
}

func TestTailSnippet_AllMessagesSameSenderStillPrefixedEachLine(t *testing.T) {
	msgs := []message{
		{SenderName: "Dad", Body: "one", SentAtUnixMs: 1},
		{SenderName: "Dad", Body: "two", SentAtUnixMs: 2},
	}
	got := tailSnippet(msgs)
	want := "Dad: one\nDad: two"
	if got != want {
		t.Errorf("expected each line prefixed even for a single sender, got %q", got)
	}
}
