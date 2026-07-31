package main

import (
	"bytes"
	"context"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/emersion/go-imap/backend/memory"
	imapclient "github.com/emersion/go-imap/client"
	imapserver "github.com/emersion/go-imap/server"

	webspacesv1 "github.com/davison/webspaces/sdk/gen/webspaces/v1"
)

// recordingRelay is a loopback TCP relay that sits in front of a real IMAP
// server: it accepts the plugin's connection, dials the real server, copies
// bytes in both directions, and tees the client-to-server direction into a
// mutex-guarded buffer. This is Proof one's deterministic, no-live-Bridge
// wire-level test — 03-02-PLAN.md Task 2.
type recordingRelay struct {
	mu  sync.Mutex
	buf bytes.Buffer
	wg  sync.WaitGroup
}

// Write implements io.Writer so recordingRelay can be used directly as one
// of io.MultiWriter's destinations.
func (r *recordingRelay) Write(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.Write(p)
}

// String returns the recorded transcript so far, safe for concurrent use.
func (r *recordingRelay) String() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.String()
}

// startRecordingRelay starts a loopback listener that relays every
// connection to serverAddr, tee-ing the client-to-server direction into the
// returned *recordingRelay. The listener is closed via t.Cleanup.
func startRecordingRelay(t *testing.T, serverAddr string) (string, *recordingRelay) {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen relay: %v", err)
	}
	t.Cleanup(func() { l.Close() })

	relay := &recordingRelay{}

	go func() {
		for {
			clientConn, err := l.Accept()
			if err != nil {
				return
			}
			relay.wg.Add(1)
			go func() {
				defer relay.wg.Done()
				serverConn, err := net.Dial("tcp", serverAddr)
				if err != nil {
					clientConn.Close()
					return
				}
				var innerWG sync.WaitGroup
				innerWG.Add(2)
				go func() {
					defer innerWG.Done()
					// The only direction this test asserts on: bytes the
					// plugin's IMAP client sends TOWARD the server.
					io.Copy(io.MultiWriter(serverConn, relay), clientConn)
				}()
				go func() {
					defer innerWG.Done()
					io.Copy(clientConn, serverConn)
				}()
				innerWG.Wait()
				clientConn.Close()
				serverConn.Close()
			}()
		}
	}()

	return l.Addr().String(), relay
}

// waitForRelayIdle blocks until every connection the relay has accepted has
// finished forwarding both directions (i.e. the plugin's Logout()-triggered
// connection close has propagated), or fails the test after a timeout.
func waitForRelayIdle(t *testing.T, relay *recordingRelay) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		relay.wg.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for the recording relay to finish forwarding")
	}
}

// seedInternalDate is the fixed IMAP INTERNALDATE newTestIMAPServer seeds
// every message with, distinct from seedEnvelopeDate by exactly one day so
// the two can never be confused for one another in an assertion. Package-
// level so every test in this package shares one source of truth (03-05
// Task 1).
var seedInternalDate = time.Date(2016, time.May, 12, 9, 15, 30, 0, time.UTC) // unix 1463044530

// seedEnvelopeDate matches the "Date:" header already present in
// sharedMessage below (Wed, 11 May 2016 14:31:59 +0000) — the message's own
// envelope Date, distinct from seedInternalDate, feeding
// Item.SecondaryTimestampUnix rather than the primary TimestampUnix.
var seedEnvelopeDate = time.Date(2016, time.May, 11, 14, 31, 59, 0, time.UTC) // unix 1462977119

// noMessageIDSubject is the distinctive subject seeded on
// "Labels/NoMessageID"'s single message — a test asserts this string never
// appears in the plugin's log output for the skip it causes (03-05 Task 2).
const noMessageIDSubject = "Unroutable seed subject"

// newTestIMAPServer starts a real github.com/emersion/go-imap server
// (server + backend/memory) on a loopback listener with insecure auth
// enabled, seeded with two mailboxes ("Labels/AlphaTeam",
// "Labels/BetaTeam") that both contain the SAME message (identical
// Message-Id) — exercising Match's dedup-by-Message-ID-merge-labels path
// (03-RESEARCH.md Pattern 2) alongside the read-only wire assertions — plus
// a third mailbox, "Labels/NoMessageID", holding a single message that
// deliberately omits the Message-Id header (03-05 Task 2). Adding this
// third mailbox does not affect TestIMAPTranscript_ExamineAndPeekOnly: that
// test's keyword list is "AlphaTeam" and "BetaTeam" only, so
// "Labels/NoMessageID" is listed but never EXAMINEd there.
//
// Every seeded message's IMAP INTERNALDATE is explicit (seedInternalDate),
// never the zero time.Time: go-imap v1's memory backend (CreateMessage)
// substitutes time.Now() whenever the date argument is the zero value, so
// passing the zero value here would make any later "is the primary
// timestamp non-zero" assertion pass for a reason that has nothing to do
// with INTERNALDATE — a trap for the very regression this fixture exists to
// catch (03-05 Task 1).
func newTestIMAPServer(t *testing.T) (addr string) {
	t.Helper()

	bkd := memory.New()
	user, err := bkd.Login(nil, "username", "password")
	if err != nil {
		t.Fatalf("seed backend: login: %v", err)
	}

	for _, name := range []string{"Labels/AlphaTeam", "Labels/BetaTeam", "Labels/NoMessageID"} {
		if err := user.CreateMailbox(name); err != nil {
			t.Fatalf("seed backend: create mailbox %q: %v", name, err)
		}
	}

	const sharedMessage = "From: Alice <alice@example.com>\r\n" +
		"To: bob@example.com\r\n" +
		"Subject: Cross-label message\r\n" +
		"Date: Wed, 11 May 2016 14:31:59 +0000\r\n" +
		"Message-Id: <shared-message@example.com>\r\n" +
		"Content-Type: text/plain\r\n" +
		"\r\n" +
		"Hello from both labels."

	for _, name := range []string{"Labels/AlphaTeam", "Labels/BetaTeam"} {
		mbox, err := user.GetMailbox(name)
		if err != nil {
			t.Fatalf("seed backend: get mailbox %q: %v", name, err)
		}
		if err := mbox.CreateMessage(nil, seedInternalDate, bytes.NewReader([]byte(sharedMessage))); err != nil {
			t.Fatalf("seed backend: create message in %q: %v", name, err)
		}
	}

	noMessageIDMsg := "From: Carol <carol@example.com>\r\n" +
		"To: bob@example.com\r\n" +
		"Subject: " + noMessageIDSubject + "\r\n" +
		"Date: Wed, 11 May 2016 14:31:59 +0000\r\n" +
		"Content-Type: text/plain\r\n" +
		"\r\n" +
		"This message deliberately has no Message-Id header."
	noMessageIDMbox, err := user.GetMailbox("Labels/NoMessageID")
	if err != nil {
		t.Fatalf("seed backend: get mailbox %q: %v", "Labels/NoMessageID", err)
	}
	if err := noMessageIDMbox.CreateMessage(nil, seedInternalDate, bytes.NewReader([]byte(noMessageIDMsg))); err != nil {
		t.Fatalf("seed backend: create message in %q: %v", "Labels/NoMessageID", err)
	}

	s := imapserver.New(bkd)
	s.AllowInsecureAuth = true

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen imap server: %v", err)
	}
	t.Cleanup(func() { s.Close() })

	go s.Serve(l)

	return l.Addr().String()
}

// TestIMAPTranscript_ExamineAndPeekOnly is Proof one (03-02-PLAN.md Task 2):
// a full Describe/Match/Fetch/Health cycle against a local fake IMAP server,
// asserting the recorded client-to-server wire transcript contains EXAMINE
// and BODY.PEEK[ and contains none of the IMAP-mutating command substrings.
// This is the no-live-Bridge counterpart to TestSeenFlagUnchanged_LiveBridge
// (live_bridge_test.go).
func TestIMAPTranscript_ExamineAndPeekOnly(t *testing.T) {
	serverAddr := newTestIMAPServer(t)
	relayAddr, relay := startRecordingRelay(t, serverAddr)

	plugin, err := NewSourcePlugin("imap://bridge.invalid:143", "username", "password", "", "https://mail.proton.me/u/0")
	if err != nil {
		t.Fatalf("NewSourcePlugin: %v", err)
	}
	// Substitute a plaintext dial to the local relay — the exact seam
	// client.go's doc comment on the dial field describes: no TLS, no
	// pinned-host check, so this test exercises everything downstream of
	// the connection (EXAMINE/FETCH/SEARCH/BODY.PEEK wire behaviour)
	// without needing a live Bridge or a self-signed certificate.
	plugin.client.dial = func(timeout time.Duration) (*imapclient.Client, error) {
		conn, err := imapclient.Dial(relayAddr)
		if err != nil {
			return nil, err
		}
		conn.Timeout = timeout
		return conn, nil
	}

	ctx := context.Background()

	if _, err := plugin.Describe(ctx, &webspacesv1.DescribeRequest{}); err != nil {
		t.Fatalf("Describe: %v", err)
	}

	matchResp, err := plugin.Match(ctx, &webspacesv1.MatchRequest{Keywords: []string{"AlphaTeam", "BetaTeam"}})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(matchResp.GetItems()) != 1 {
		t.Fatalf("Match: got %d items, want exactly 1 (dedup across two matching mailboxes)", len(matchResp.GetItems()))
	}
	item := matchResp.GetItems()[0]
	wantLabels := map[string]bool{"AlphaTeam": true, "BetaTeam": true}
	if len(item.GetLabels()) != len(wantLabels) {
		t.Fatalf("Match item labels = %v, want both mailbox leaf names %v", item.GetLabels(), wantLabels)
	}
	for _, l := range item.GetLabels() {
		if !wantLabels[l] {
			t.Errorf("Match item labels contains unexpected label %q", l)
		}
	}

	fetchResp, err := plugin.Fetch(ctx, &webspacesv1.FetchRequest{
		SourceId: item.GetSourceId(),
		Variant:  webspacesv1.ContentVariant_CONTENT_VARIANT_FULL,
	})
	if err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if !fetchResp.GetAvailable() {
		t.Fatalf("Fetch: Available = false, want true")
	}

	if _, err := plugin.Health(ctx, &webspacesv1.HealthRequest{}); err != nil {
		t.Fatalf("Health: %v", err)
	}

	waitForRelayIdle(t, relay)

	transcript := relay.String()

	mustContain := []string{"EXAMINE ", "BODY.PEEK["}
	for _, want := range mustContain {
		if !strings.Contains(transcript, want) {
			t.Errorf("transcript missing required substring %q; transcript:\n%s", want, transcript)
		}
	}

	mustNotContain := []string{"SELECT ", "STORE ", "EXPUNGE", "APPEND ", " COPY ", " MOVE ", "DELETE "}
	for _, forbidden := range mustNotContain {
		if strings.Contains(transcript, forbidden) {
			t.Errorf("transcript contains forbidden mutating substring %q (PLUG-02 violation); transcript:\n%s", forbidden, transcript)
		}
	}
}
