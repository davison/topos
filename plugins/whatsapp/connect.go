package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "modernc.org/sqlite"
)

// pluginLogger implements waLog.Logger, writing to os.Stderr (never
// os.Stdout, which the go-plugin subprocess handshake protocol owns) at a
// fixed WARN-and-above level (Debugf/Infof are silently dropped). This
// bounds what whatsmeow's OWN internal logging can emit (T-08-03's
// mitigation) — every log call this plugin's own code makes
// (eventhandler.go, connect.go, plugin.go) is separately written to never
// include a message body, contact name, or session key material,
// regardless of level.
type pluginLogger struct {
	module string
}

func newPluginLogger(module string) waLog.Logger { return pluginLogger{module: module} }

func (l pluginLogger) Errorf(msg string, args ...any) { l.printf("ERROR", msg, args...) }
func (l pluginLogger) Warnf(msg string, args ...any)  { l.printf("WARN", msg, args...) }
func (l pluginLogger) Infof(msg string, args ...any)  {}
func (l pluginLogger) Debugf(msg string, args ...any) {}
func (l pluginLogger) Sub(module string) waLog.Logger {
	return pluginLogger{module: l.module + "/" + module}
}

func (l pluginLogger) printf(level, msg string, args ...any) {
	fmt.Fprintf(os.Stderr, "topos-plugin-whatsapp[%s %s]: "+msg+"\n", append([]any{l.module, level}, args...)...)
}

// whatsmeowSessionDSN builds the modernc.org/sqlite connection string for
// whatsmeow's own session store (whatsmeow.db). whatsmeow's own
// sqlstore.Container.Upgrade checks `PRAGMA foreign_keys` on the
// connection it is handed and refuses to run its migrations if it comes
// back off — confirmed live (2026-08-10): a bare `file:<path>` DSN (or the
// `?_foreign_keys=on` shorthand whatsmeow's own doc comment illustrates,
// which is mattn/go-sqlite3's query-param convention, not
// modernc.org/sqlite's) fails at container open with "failed to upgrade
// database: foreign keys are not enabled" before ever reaching the QR
// flow. modernc.org/sqlite's own DSN pragma syntax is
// `_pragma=<pragma-body>` — one query param per pragma, applied via
// `PRAGMA <body>` on every new pooled connection (sqlite.go's
// applyQueryParams) — so `_foreign_keys=on` is silently ignored as an
// unrecognised query param rather than erroring, which is what made this
// fail quietly instead of at compile/lint time. Both link.go's one-shot
// link flow and connect.go's persistent serve-mode connection MUST open
// whatsmeow's sqlstore with this identical DSN — the CONTEXT hard
// requirement is that both open it the same way.
func whatsmeowSessionDSN(dbPath string) string {
	return "file:" + dbPath + "?_pragma=foreign_keys(1)&_pragma=busy_timeout(10000)"
}

// startBackgroundClient acquires the store lock, opens whatsmeow's own
// sqlstore (whatsmeow.db — a file distinct from this plugin's own
// messages.db, messagestore.go), constructs a whatsmeow.Client, registers
// this plugin's own event handler (eventhandler.go), and — only when a
// device is already linked — connects and holds that connection for the
// plugin's entire process lifetime. When no device is linked yet, this
// records that state and returns without connecting; the operator links
// via this binary's own -link flag (link.go), never through this
// RPC-serving process — storelock.go's exclusive lock is what makes the
// two mutually exclusive.
func (p *SourcePlugin) startBackgroundClient(ctx context.Context) error {
	lock, err := acquireStoreLock(p.dir)
	if err != nil {
		return err // already-named (ErrStoreInUse) or wrapped
	}
	p.lock = lock

	dbPath := filepath.Join(p.dir, "whatsmeow.db")
	container, err := sqlstore.New(ctx, "sqlite", whatsmeowSessionDSN(dbPath), newPluginLogger("whatsmeow/store"))
	if err != nil {
		return fmt.Errorf("whatsapp: open whatsmeow session store %s: %w", dbPath, err)
	}
	p.container = container

	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("whatsapp: read linked device: %w", err)
	}

	client := whatsmeow.NewClient(device, newPluginLogger("whatsmeow/client"))
	client.AddEventHandler(p.handleEvent)
	p.client = client

	if device.ID == nil {
		p.setUnhealthy("whatsapp: not linked — run this binary's -link flag once to pair a device")
		return nil
	}

	p.setLinked(true)
	if err := client.Connect(); err != nil {
		// Not fatal to process startup — a transient network failure at
		// boot should not crash-loop the plugin subprocess; Health/Match
		// report the unhealthy state until a future *events.Connected
		// fires (07-RESEARCH.md assumption A2's precedent: every plugin
		// in this repo defers live-connectivity failures past process
		// startup).
		p.setUnhealthy(fmt.Sprintf("whatsapp: connect failed: %v", err))
		return nil
	}

	return nil
}
