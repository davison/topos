// Command handshakev1 is a launch fixture for kernel/pluginhost's
// handshake-incompatibility test (M1-R6/DIST-03, davison/topos#17): a
// REAL go-plugin server, serving the real plugin map over the real magic
// cookie, but at ProtocolVersion 1 — the pre-Phase-5 generation — so the
// kernel's client-side handshake refuses it exactly the way a stale
// third-party binary would be refused. Lives under testdata/ so `go
// build ./...` never builds it; the test builds it explicitly, like
// reconcile_test.go's own mock fixture.
package main

import (
	goplugin "github.com/hashicorp/go-plugin"

	"github.com/davison/topos/sdk"
)

func main() {
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: goplugin.HandshakeConfig{
			ProtocolVersion:  1,
			MagicCookieKey:   sdk.Handshake.MagicCookieKey,
			MagicCookieValue: sdk.Handshake.MagicCookieValue,
		},
		Plugins:    sdk.PluginMap,
		GRPCServer: sdk.GRPCServer,
	})
}
