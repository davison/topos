// Command topos-plugin-mockstrict exists only for the browser E2E harness
// (07.1-browser-e2e-harness, D-05/D-06): a hermetic, cgo-free test plugin
// that reproduces the ONE mechanism plugins/mock deliberately omits — a
// required connection field with a pre-Serve fatal guard — so
// 07-UAT.md's item 5 "New Signal…" two-step flow (pre-fill from a seeded
// default, clear it, submission blocked with zero network requests,
// restore, proceed) can be exercised in Chromium without cgo, SQLCipher,
// or a real Signal Desktop database anywhere in the loop.
//
// It is deliberately shaped like the signal plugin's main.go: a single
// mandatory `path` field, decoded from WEBSPACES_SOURCE_CONFIG, that
// fatally exits this process before goplugin.Serve is ever reached when
// it is empty — the exact server-side guard the SPA's required-fields
// table (web/src/lib/plugin-fields.ts) mirrors for the
// `topos-plugin-mockstrict` row. Unlike Signal, the path is never opened:
// plugin.go serves a fixed in-memory corpus, so any non-empty string
// satisfies the guard and the fixture environment stays deterministic.
//
// `make build` and `make plugins` never build this binary — only
// `make e2e` does (Makefile's `e2e` target) — so it can never ship into a
// real installation's plugins directory or be offered in an operator's
// own "+ New …" picker.
package main

import (
	"encoding/json"
	"fmt"
	"os"

	goplugin "github.com/hashicorp/go-plugin"

	"github.com/davison/topos/sdk"
)

// sourceConfig is decoded from the WEBSPACES_SOURCE_CONFIG environment
// variable the kernel's pluginhost sets before launching this subprocess
// — mirrors the signal plugin's main.go's sourceConfig shape exactly (the
// only single-required-field connection shape kernel/config.Validate
// accepts: base_url+token OR path).
type sourceConfig struct {
	Path string `json:"path"`
}

func main() {
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

	impl := NewSourcePlugin(cfg.Path)

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: sdk.Handshake,
		Plugins: map[string]goplugin.Plugin{
			"source": &sdk.SourcePluginGRPCPlugin{Impl: impl},
		},
		// sdk.GRPCServer (not goplugin.DefaultGRPCServer) raises the gRPC
		// message-size ceiling to match every other plugin's dial options
		// — see sdk/shared.go's MaxMessageSize doc comment. This plugin
		// never sends a rendition large enough to need it, but every
		// plugin uses the same GRPCServer so the contract's message-size
		// guarantee holds uniformly.
		GRPCServer: sdk.GRPCServer,
	})
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "topos-plugin-mockstrict:", err)
	os.Exit(1)
}
