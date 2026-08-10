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
	container, err := sqlstore.New(ctx, "sqlite", "file:"+dbPath+"?_foreign_keys=on", newPluginLogger("whatsmeow/link"))
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

	qrChan, err := client.GetQRChannel(ctx)
	if err != nil {
		return fmt.Errorf("whatsapp: get QR channel: %w", err)
	}
	if err := client.Connect(); err != nil {
		return fmt.Errorf("whatsapp: connect: %w", err)
	}
	defer client.Disconnect()

	for evt := range qrChan {
		switch evt.Event {
		case "code":
			fmt.Println("Scan with your phone to link (valid for", evt.Timeout.Round(time.Second), "):")
			qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
		case "success":
			fmt.Println("Linked successfully.")
			return nil
		case "timeout":
			return fmt.Errorf("whatsapp: pairing timed out — re-run to try again")
		case "error":
			return fmt.Errorf("whatsapp: pairing error: %w", evt.Error)
		default:
			return fmt.Errorf("whatsapp: pairing failed: %s", evt.Event)
		}
	}
	return fmt.Errorf("whatsapp: QR channel closed before pairing completed")
}
