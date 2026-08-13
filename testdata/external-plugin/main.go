// Command topos-plugin-external-demo is Phase 11's out-of-repo proof
// plugin (ROADMAP success criterion 5): a standalone Go module, module
// path example.com/acme/topos-plugin-external-demo — deliberately NOT
// under github.com/davison/topos — built entirely from the published
// contract (docs/plugin-contract.md), the wire contract
// (proto/topos/v1/plugin.proto), and the sdk module, exactly as a genuine
// third-party plugin author would build it.
//
// This binary is built ONLY by its own `make external-demo` target, into
// its own `bin/plugins-external/` output directory — never by `make
// build`, `make plugins`, or `make plugins-portable` — and it lives under
// testdata/, a directory this repository's own AST audits
// (internal/audit) deliberately skip everywhere they walk. It must NEVER
// be copied into a real installation's trusted plugin directory — see
// README.md.
//
// Its pre-Serve fatal-guard shape (a single required "path" key) mirrors
// plugins/mockstrict/main.go and plugins/signal/main.go exactly: a
// plugin MUST fail startup loudly, by name, non-zero, when a required
// config key is empty — never start up silently and fail later,
// mid-Match, with a confusing downstream error
// (docs/plugin-contract.md "Configuration: WEBSPACES_SOURCE_CONFIG").
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
// (docs/plugin-contract.md "Configuration"). Path is the one kernel-known
// key this plugin cares about (mirroring plugins/mockstrict/main.go's
// identical single-required-field shape); Extras carries this instance's
// own [sources.<id>.extras] table verbatim (D-12/D-13) — the
// provider-specific passthrough this whole module exists to prove reaches
// an out-of-repo plugin process unmodified.
type sourceConfig struct {
	Path   string            `json:"path"`
	Extras map[string]string `json:"extras"`
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

	impl := NewSourcePlugin(cfg.Extras)

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: sdk.Handshake,
		Plugins: map[string]goplugin.Plugin{
			"source": &sdk.SourcePluginGRPCPlugin{Impl: impl},
		},
		// sdk.GRPCServer (not goplugin.DefaultGRPCServer) raises the gRPC
		// message-size ceiling to match the kernel's own dial options —
		// see sdk/shared.go's MaxMessageSize doc comment. This proof
		// plugin never sends a rendition large enough to need it, but
		// every contract-conformant plugin uses the same GRPCServer so
		// the message-size guarantee holds uniformly.
		GRPCServer: sdk.GRPCServer,
	})
}

// fatal writes a named, non-secret error message to stderr and exits
// non-zero — mirrors plugins/mockstrict/main.go's identical helper. Never
// called with anything that could contain a secret value
// (docs/plugin-contract.md "Logging").
func fatal(err error) {
	fmt.Fprintln(os.Stderr, "topos-plugin-external-demo:", err)
	os.Exit(1)
}
