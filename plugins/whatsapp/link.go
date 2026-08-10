package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mdp/qrterminal/v3"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"

	_ "modernc.org/sqlite"
)

// postPairLoginTimeout bounds how long runLinkCLI waits for a real
// *events.Connected after the QR channel reports "success" (PairSuccess)
// before giving up — generous because the post-pair reconnection and
// login handshake involves a fresh websocket connection and a full
// app-state/key exchange, not just a fast ack.
const postPairLoginTimeout = 60 * time.Second

// postPairGraceWindow is a short additional wait AFTER *events.Connected
// fires, before this process calls Disconnect() — giving the client's
// own initial post-login exchange (app state sync, key distribution) a
// moment to get underway on the same socket, rather than dropping it the
// instant Connected is observed.
const postPairGraceWindow = 5 * time.Second

// runLinkCLI implements the one-shot terminal QR link flow
// (-link -path <dir>): acquire the store lock (exiting with a named
// store-in-use error if a serve-mode instance already holds it — the
// CONTEXT hard requirement's mutual-exclusion rule, storelock.go), open
// the SAME whatsmeow sqlstore serve-mode uses, and either report an
// already-linked device or render a rotating ASCII QR code until the
// phone scans it.
func runLinkCLI(ctx context.Context, dir string) error {
	lock, err := acquireStoreLock(dir)
	if err != nil {
		return err
	}
	defer lock.Release()

	dbPath := filepath.Join(dir, "whatsmeow.db")
	container, err := sqlstore.New(ctx, "sqlite", whatsmeowSessionDSN(dbPath), newPluginLogger("whatsmeow/link"))
	if err != nil {
		return fmt.Errorf("whatsapp: open whatsmeow session store %s: %w", dbPath, err)
	}
	defer container.Close()

	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("whatsapp: read linked device: %w", err)
	}

	if device.ID != nil {
		fmt.Println("Already linked — device", device.ID.String(), "is paired. Re-run without -link to serve.")
		return nil
	}

	client := whatsmeow.NewClient(device, newPluginLogger("whatsmeow/link"))

	// Registered BEFORE GetQRChannel/Connect, alongside the QR channel's
	// own internal event handler — both observe the same client's event
	// stream. GetQRChannel's own "success" fires on *events.PairSuccess,
	// which its own doc comment says is "generally followed by a
	// websocket reconnection" — pairLoginWaiter is what lets this
	// function wait for the SUBSEQUENT *events.Connected (or a
	// definitive failure) rather than disconnecting the instant
	// PairSuccess arrives, which strands the phone mid post-pair login
	// handshake (live-reported 2026-08-10: phone stuck on
	// "Logging in…" with an EOF on the plugin's own socket).
	loginWaiter := newPairLoginWaiter()
	client.AddEventHandler(loginWaiter.handleEvent)

	qrChan, err := client.GetQRChannel(ctx)
	if err != nil {
		return fmt.Errorf("whatsapp: get QR channel: %w", err)
	}
	if err := client.Connect(); err != nil {
		return fmt.Errorf("whatsapp: connect: %w", err)
	}
	defer client.Disconnect()

	paired := false
	for evt := range qrChan {
		switch evt.Event {
		case "code":
			fmt.Println("Scan with your phone to link (valid for", evt.Timeout.Round(time.Second), "):")
			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
		case "success":
			// Do NOT return/disconnect here — see loginWaiter's own doc
			// comment above. The phone shows "Logging in…" until the
			// wait below observes the real post-pair Connected.
			paired = true
			fmt.Println("Pairing accepted — completing login…")
		case "timeout":
			return fmt.Errorf("whatsapp: pairing timed out — re-run to try again")
		case "error":
			return fmt.Errorf("whatsapp: pairing error: %w", evt.Error)
		default:
			return fmt.Errorf("whatsapp: pairing failed: %s", evt.Event)
		}
	}
	if !paired {
		return fmt.Errorf("whatsapp: QR channel closed before pairing completed")
	}

	if err := loginWaiter.wait(postPairLoginTimeout); err != nil {
		return err
	}

	// Grace window for the client's own initial post-login exchange
	// (app-state sync, key distribution) to get underway on this same
	// socket before this process calls Disconnect() (deferred above).
	time.Sleep(postPairGraceWindow)

	fmt.Println("Linked successfully.")
	return nil
}
