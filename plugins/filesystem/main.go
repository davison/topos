// Command topos-plugin-filesystem is the local/network filesystem source
// plugin subprocess (SRC-04), launched by the kernel's plugin host over the
// go-plugin gRPC handshake. Follows plugins/signal/main.go's shape exactly:
// a local-path source with no base_url/token at all, since this plugin
// reads directly from an arbitrary folder rather than a network API.
//
// 12-01-PLAN.md Task 2 (the phase's tracer slice) implemented the thinnest
// complete path — top-level PDF files, no recursion, no extras-driven
// scope widening/narrowing. 12-02-PLAN.md Task 2 widens Match's document
// scope via extras-driven include/exclude globs (D-03); this bootstrap
// now decodes and forwards those extras, but still does not expand
// recursion — that remains a later plan's work.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"strings"

	goplugin "github.com/hashicorp/go-plugin"

	"github.com/davison/topos/sdk"
)

// sourceConfig is decoded from the WEBSPACES_SOURCE_CONFIG environment
// variable the kernel's pluginhost sets before launching this subprocess.
// Path is the configured source folder (kernel/config.Source.Path) — the
// kernel deliberately stores it unexpanded (see that field's own doc
// comment), so a leading "~" is expanded here, in the plugin subprocess,
// exactly like plugins/signal/main.go already does. Extras carries this
// instance's own config.Source.Extras verbatim (D-12/D-13) — the
// include_glob/exclude_glob scope-override keys scope.go's newScope reads
// (12-02-PLAN.md Task 2); omitted entirely (nil map) when this source
// declares no extras, exactly like kernel/pluginhost/host.go's
// sourceConfigEnvelope.Extras.
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

	root, err := expandHome(cfg.Path)
	if err != nil {
		fatal(fmt.Errorf("expand path: %w", err))
	}

	impl := NewSourcePlugin(root, cfg.Extras)

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: sdk.Handshake,
		Plugins: map[string]goplugin.Plugin{
			"source": &sdk.SourcePluginGRPCPlugin{Impl: impl},
		},
		// sdk.GRPCServer (not goplugin.DefaultGRPCServer) raises the gRPC
		// message-size ceiling so a unary Fetch response carrying a full PDF
		// rendition doesn't hit the 4 MB default — every plugin in this repo
		// uses the same GRPCServer so the contract's message-size guarantee
		// holds uniformly.
		GRPCServer: sdk.GRPCServer,
	})
}

// expandHome expands a leading "~" in path to the current user's home
// directory — copied verbatim from plugins/signal/main.go's identical
// helper; the kernel never expands Source.Path itself (kernel/config/
// types.go's Path doc comment), only the plugin subprocess does.
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
	fmt.Fprintln(os.Stderr, "topos-plugin-filesystem:", err)
	os.Exit(1)
}
