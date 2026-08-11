package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "modernc.org/sqlite"
)

// serveLoginTimeout bounds how long serve-mode startup (startBackgroundClient's
// success path, below) blocks before falling through to goplugin.Serve in the
// connecting state. Deliberately much shorter than link.go's
// postPairLoginTimeout: a link flow has a human watching a QR code and
// waiting for it, a boot does not. It must also stay comfortably under
// go-plugin's own client StartTimeout default of one minute
// (hashicorp/go-plugin), because main.go calls NewSourcePlugin BEFORE
// goplugin.Serve — every second spent here is a second the kernel's
// pluginhost.launch is blocked on the handshake completing.
const serveLoginTimeout = 15 * time.Second

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
		p.setHealthState(healthStateNotLinked, "")
		return nil
	}

	// G-08-4 (missing[0]): explicitly assign the connecting state BEFORE
	// dialing, belt and braces with health.go's zero-value fix — the
	// plugin's reported state is honest from the first instant of the
	// dial rather than only after Connect() returns.
	p.setHealthState(healthStateConnecting, "")

	// G-08-4 (missing[1]): register a pairLoginWaiter (pairwait.go — the
	// SAME primitive the link flow already proves against a real device)
	// AFTER p.handleEvent is already registered above. Ordering is
	// load-bearing: whatsmeow dispatches an event to its handlers
	// synchronously, in registration order, so by the time this waiter is
	// signalled, p.handleEvent's own *events.Connected case has ALREADY
	// assigned healthStateLinked. That is why the wait below adds no
	// post-wait state assignment of its own — doing so would risk
	// clobbering a LoggedOut or StreamReplaced that landed in the same
	// instant.
	loginWaiter := newPairLoginWaiter()
	client.AddEventHandler(loginWaiter.handleEvent)

	if err := client.Connect(); err != nil {
		// Not fatal to process startup — a transient network failure at
		// boot should not crash-loop the plugin subprocess; Health/Match
		// report the unhealthy state until a future *events.Connected
		// fires (07-RESEARCH.md assumption A2's precedent: every plugin
		// in this repo defers live-connectivity failures past process
		// startup).
		//
		// Reported as healthStateNotLinked, DELIBERATELY not one of
		// Task 1's three new named causes (de-link/ban/expiry are all
		// events WhatsApp's OWN server explicitly told us about via a
		// LATER *events.LoggedOut/TemporaryBan/ConnectFailure —
		// eventhandler.go — this is simply "haven't yet completed a
		// dial" for a device that IS already paired). whatsmeow's own
		// Client already schedules a background auto-reconnect for a
		// retryable error (EnableAutoReconnect defaults true, confirmed
		// in whatsmeow's client.go) — once that succeeds,
		// *events.Connected (eventhandler.go) transitions to
		// healthStateLinked automatically with zero further code here.
		// The real dial error is carried in the detail so Health's
		// LastError stays specific rather than merely the fixed
		// not-linked template verbatim.
		p.setHealthState(healthStateNotLinked, fmt.Sprintf("initial connect failed, retrying: %v", err))
		return nil
	}

	// Bounded wait for the SAME client to observe a real *events.Connected
	// (or a definitive login failure) before this function returns and
	// main.go reaches goplugin.Serve — so the kernel's first Match after a
	// (re)launch normally lands after login genuinely completed, not at
	// the first instant of the dial. The wait outcome is never fatal to
	// control flow: every path below still returns nil so goplugin.Serve
	// is always reached; a non-nil wait error is only logged, one line,
	// under this package's fixed pluginName prefix, carrying no chat name,
	// sender name, message body, or key material.
	if err := loginWaiter.wait(serveLoginTimeout); err != nil {
		fmt.Fprintf(p.logOut, "%s: serve-mode startup: %v\n", pluginName, err)
	}

	return nil
}
