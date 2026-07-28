// Command webspaces-plugin-mock is the reference "mock" source plugin
// subprocess (PLUG-05): a working SourcePlugin built entirely from the
// published contract (docs/plugin-contract.md), the wire contract
// (proto/webspaces/v1/plugin.proto) and the sdk module — with no network
// dependency, no real source system, and no configuration requirement.
// It is launched exactly like every other plugin, over the same go-plugin
// gRPC handshake (see docs/plugin-contract.md "Handshake and the
// plugin-map key").
package main

import (
	"os"

	goplugin "github.com/hashicorp/go-plugin"

	"github.com/davison/webspaces/sdk"
)

func main() {
	// Unlike every real plugin's main.go (plugins/paperless/main.go,
	// plugins/silverbullet/main.go), this plugin does NOT fail startup
	// when WEBSPACES_SOURCE_CONFIG is unset — the mock has no base_url,
	// no token, and no filesystem path to require. It reads the variable
	// if present (so a curious operator setting [sources.mock] with an
	// empty config block still works exactly as documented) but never
	// treats its absence as an error.
	_ = os.Getenv("WEBSPACES_SOURCE_CONFIG")

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: sdk.Handshake,
		Plugins: map[string]goplugin.Plugin{
			"source": &sdk.SourcePluginGRPCPlugin{Impl: NewSourcePlugin()},
		},
		// sdk.GRPCServer (not goplugin.DefaultGRPCServer) raises the gRPC
		// message-size ceiling to match the kernel's own dial options —
		// see sdk/shared.go's MaxMessageSize doc comment. The mock never
		// sends a rendition large enough to need this, but every plugin
		// uses the same GRPCServer so the contract's message-size
		// guarantee holds uniformly.
		GRPCServer: sdk.GRPCServer,
	})
}
