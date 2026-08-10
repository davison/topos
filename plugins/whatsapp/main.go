// Command topos-plugin-whatsapp is the WhatsApp (linked-device, whatsmeow)
// source plugin subprocess, launched by the kernel's plugin host over the
// go-plugin gRPC handshake. Unlike Signal, this binary is pure Go — no
// build tag, no cgo — and unlike every other plugin in this repo, it holds
// a persistent connection (a background whatsmeow Client.Connect) for its
// entire process lifetime rather than opening-and-closing per RPC call
// (plugins/signal/main.go's identical bootstrap shape, minus the cgo
// divergence — see plugins/signal/main.go's own doc comment).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/user"
	"strings"

	goplugin "github.com/hashicorp/go-plugin"

	wastore "go.mau.fi/whatsmeow/store"

	"github.com/davison/topos/sdk"
)

// sourceConfig is decoded from the WEBSPACES_SOURCE_CONFIG environment
// variable the kernel's pluginhost sets before launching this subprocess —
// mirrors plugins/signal/main.go's identical sourceConfig shape: a
// local-path source with no base_url/token/username at all. Path is the
// directory holding this plugin's TWO owned databases: whatsmeow's own
// session store (whatsmeow.db) and this plugin's own message-content store
// (messages.db) — see messagestore.go and connect.go.
type sourceConfig struct {
	Path string `json:"path"`
}

func main() {
	// Cosmetic fix (2026-08-10 real-device spike): whatsmeow's own
	// package-level DeviceProps.Os defaults to the literal string
	// "whatsmeow" — that's what a real phone's WhatsApp > Linked Devices
	// list showed for this plugin's linked session. SetOSInfo mutates
	// that shared global var, so it must run once, early, before EITHER
	// code path below ever constructs a whatsmeow.Client (link.go's
	// runLinkCLI or connect.go's startBackgroundClient) — both are
	// reached from this single main(), so calling it here covers both.
	wastore.SetOSInfo("topos", [3]uint32{0, 1, 0})

	linkMode := flag.Bool("link", false, "run the one-shot terminal QR link flow against -path, then exit")
	linkPath := flag.String("path", "", "the plugin's own data directory (same value as [sources.whatsapp].path in config.toml)")
	flag.Parse()

	if *linkMode {
		dir, err := expandHome(*linkPath)
		if err != nil {
			fatal(fmt.Errorf("expand -path: %w", err))
		}
		if dir == "" {
			fatal(fmt.Errorf("-link requires -path"))
		}
		if err := runLinkCLI(context.Background(), dir); err != nil {
			fatal(err)
		}
		os.Exit(0)
	}

	raw := os.Getenv("WEBSPACES_SOURCE_CONFIG")
	if raw == "" {
		fatal(fmt.Errorf("WEBSPACES_SOURCE_CONFIG is not set"))
	}

	var cfg sourceConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		fatal(fmt.Errorf("parse WEBSPACES_SOURCE_CONFIG: %w", err))
	}
	if cfg.Path == "" {
		fatal(fmt.Errorf("WEBSPACES_SOURCE_CONFIG: path is empty"))
	}

	dataDir, err := expandHome(cfg.Path)
	if err != nil {
		fatal(fmt.Errorf("expand path: %w", err))
	}

	impl, err := NewSourcePlugin(context.Background(), dataDir)
	if err != nil {
		fatal(fmt.Errorf("start whatsapp plugin: %w", err))
	}

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: sdk.Handshake,
		Plugins: map[string]goplugin.Plugin{
			"source": &sdk.SourcePluginGRPCPlugin{Impl: impl},
		},
		// sdk.GRPCServer (not goplugin.DefaultGRPCServer) raises the gRPC
		// message-size ceiling so a unary Fetch response carrying a full
		// rendition doesn't hit the 4 MB default — every plugin in this
		// repo uses the same GRPCServer so the contract's message-size
		// guarantee holds uniformly.
		GRPCServer: sdk.GRPCServer,
	})
}

// expandHome expands a leading "~" in path to the current user's home
// directory — mirrors plugins/signal/main.go's identical helper.
func expandHome(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	u, err := user.Current()
	if err != nil {
		return "", err
	}
	return strings.Replace(path, "~", u.HomeDir, 1), nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "topos-plugin-whatsapp:", err)
	os.Exit(1)
}
