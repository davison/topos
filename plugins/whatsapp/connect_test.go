package main

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
	"testing"

	"go.mau.fi/whatsmeow/store/sqlstore"
)

// TestWhatsmeowSessionDSN_EnablesForeignKeys proves the DSN both link.go
// and connect.go build for whatsmeow's own sqlstore carries the
// modernc.org/sqlite pragma syntax (`_pragma=foreign_keys(1)`) — NOT the
// `_foreign_keys=on` shorthand whatsmeow's own doc comment illustrates,
// which is a DIFFERENT sqlite driver's DSN convention and is silently
// ignored by modernc.org/sqlite, leaving foreign keys off and
// sqlstore.Container.Upgrade refusing to run (observed live, 2026-08-10:
// "failed to upgrade database: foreign keys are not enabled").
func TestWhatsmeowSessionDSN_EnablesForeignKeys(t *testing.T) {
	dsn := whatsmeowSessionDSN("/tmp/example/whatsmeow.db")

	if !strings.Contains(dsn, "_pragma=foreign_keys(1)") {
		t.Fatalf("whatsmeowSessionDSN() = %q, want it to contain modernc.org/sqlite's _pragma=foreign_keys(1) syntax", dsn)
	}
	if strings.Contains(dsn, "_foreign_keys=on") {
		t.Fatalf("whatsmeowSessionDSN() = %q, contains the WRONG (mattn/go-sqlite3-style) _foreign_keys=on shorthand, which modernc.org/sqlite silently ignores", dsn)
	}
}

// TestWhatsmeowSessionDSN_MigrationsRunAgainstRealSQLStore actually opens
// whatsmeow's own sqlstore.New against a fresh temp-file database using
// whatsmeowSessionDSN — the same call link.go's runLinkCLI and
// connect.go's startBackgroundClient both make. This is the regression
// test for the live failure a real -link run hit: a wrong DSN fails HERE,
// at Container.Upgrade's own foreign-keys precondition check, without
// needing a phone or a network connection to reproduce.
func TestWhatsmeowSessionDSN_MigrationsRunAgainstRealSQLStore(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "whatsmeow.db")

	container, err := sqlstore.New(context.Background(), "sqlite", whatsmeowSessionDSN(dbPath), newPluginLogger("whatsmeow/test"))
	if err != nil {
		t.Fatalf("sqlstore.New with whatsmeowSessionDSN: %v (this is exactly the failure mode a wrong DSN produces: 'failed to upgrade database: foreign keys are not enabled')", err)
	}
	defer container.Close()

	// GetFirstDevice on a brand-new store creates and persists a fresh,
	// unlinked device row — proving the migrated schema is actually
	// usable, not just that Upgrade returned nil.
	if _, err := container.GetFirstDevice(context.Background()); err != nil {
		t.Fatalf("GetFirstDevice against freshly migrated store: %v", err)
	}
}

// TestStartBackgroundClient_SuccessPathSetsConnectingAndWaitsForLogin is an
// AST structural guard (mirroring readonly_test.go's own house pattern)
// pinning startBackgroundClient's already-paired success-path ordering,
// which G-08-4 identified as this plugin's second defect: the success path
// used to call client.Connect() and return without ever assigning a health
// state or waiting for a real login. This is the only automated proof
// available for this path in-repo — a genuinely behavioral test would need
// a live WhatsApp server, precisely the blind spot the debug session
// (.planning/debug/whatsapp-paired-session-not-picked-up.md) recorded; this
// guard proves ORDERING of the relevant calls by source position, not that
// the wait actually blocks against a real server.
func TestStartBackgroundClient_SuccessPathSetsConnectingAndWaitsForLogin(t *testing.T) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "connect.go", nil, parser.SkipObjectResolution)
	if err != nil {
		t.Fatalf("parse connect.go: %v", err)
	}

	var fn *ast.FuncDecl
	for _, decl := range file.Decls {
		fd, ok := decl.(*ast.FuncDecl)
		if !ok || fd.Name.Name != "startBackgroundClient" {
			continue
		}
		fn = fd
		break
	}
	if fn == nil {
		t.Fatal("G-08-4: could not find startBackgroundClient FuncDecl in connect.go")
	}

	var (
		setConnectingPos token.Pos
		addHandlerPos    token.Pos
		connectPos       token.Pos
		waitPos          token.Pos
	)

	ast.Inspect(fn.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		switch sel.Sel.Name {
		case "setHealthState":
			if setConnectingPos == token.NoPos && len(call.Args) > 0 {
				if id, ok := call.Args[0].(*ast.Ident); ok && id.Name == "healthStateConnecting" {
					setConnectingPos = call.Pos()
				}
			}
		case "AddEventHandler":
			if addHandlerPos == token.NoPos && len(call.Args) == 1 {
				// The login waiter's handler is registered as
				// `loginWaiter.handleEvent` — a *ast.SelectorExpr whose
				// Sel.Name is "handleEvent" AND whose receiver identifier
				// is "loginWaiter" (distinguishing it from the earlier
				// `client.AddEventHandler(p.handleEvent)` registration,
				// whose argument selector's receiver is "p").
				if argSel, ok := call.Args[0].(*ast.SelectorExpr); ok && argSel.Sel.Name == "handleEvent" {
					if x, ok := argSel.X.(*ast.Ident); ok && x.Name == "loginWaiter" {
						addHandlerPos = call.Pos()
					}
				}
			}
		case "Connect":
			if connectPos == token.NoPos && len(call.Args) == 0 {
				connectPos = call.Pos()
			}
		case "wait":
			if waitPos == token.NoPos && len(call.Args) == 1 {
				if id, ok := call.Args[0].(*ast.Ident); ok && id.Name == "serveLoginTimeout" {
					waitPos = call.Pos()
				}
			}
		}
		return true
	})

	if setConnectingPos == token.NoPos {
		t.Fatal("G-08-4: no setHealthState(healthStateConnecting, ...) call found in startBackgroundClient")
	}
	if addHandlerPos == token.NoPos {
		t.Fatal("G-08-4: no AddEventHandler(loginWaiter.handleEvent) call found in startBackgroundClient")
	}
	if connectPos == token.NoPos {
		t.Fatal("G-08-4: no client.Connect() call found in startBackgroundClient")
	}
	if waitPos == token.NoPos {
		t.Fatal("G-08-4: no wait(serveLoginTimeout) call found in startBackgroundClient")
	}

	if !(setConnectingPos < connectPos) {
		t.Fatalf("G-08-4: setHealthState(healthStateConnecting, ...) at %s must appear BEFORE client.Connect() at %s — the connecting state must be assigned before dialing", fset.Position(setConnectingPos), fset.Position(connectPos))
	}
	if !(addHandlerPos < connectPos) {
		t.Fatalf("G-08-4: AddEventHandler(loginWaiter.handleEvent) at %s must appear BEFORE client.Connect() at %s — the waiter must be registered before dialing so it observes the SAME client's events", fset.Position(addHandlerPos), fset.Position(connectPos))
	}
	if !(connectPos < waitPos) {
		t.Fatalf("G-08-4: wait(serveLoginTimeout) at %s must appear AFTER client.Connect() at %s — the wait blocks on the connection just established", fset.Position(waitPos), fset.Position(connectPos))
	}
}
