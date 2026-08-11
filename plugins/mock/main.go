// Command topos-plugin-mock is the reference "mock" source plugin
// subprocess (PLUG-05): a working SourcePlugin built entirely from the
// published contract (docs/plugin-contract.md), the wire contract
// (proto/topos/v1/plugin.proto) and the sdk module — with no network
// dependency, no real source system, and no configuration requirement.
// It is launched exactly like every other plugin, over the same go-plugin
// gRPC handshake (see docs/plugin-contract.md "Handshake and the
// plugin-map key").
package main

import (
	"fmt"
	"os"
	"time"

	goplugin "github.com/hashicorp/go-plugin"

	"github.com/davison/topos/sdk"
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

	// Fixture-only launch-readiness window (readiness.go, G-08-4) — off by
	// default (see readinessWindowFromEnv), so a real installation's mock
	// is unaffected. A malformed value fails startup loudly rather than
	// silently altering behaviour, matching every other plugin's
	// fail-loud-by-name startup discipline.
	ready, err := readinessWindowFromEnv(time.Now(), os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "topos-plugin-mock:", err)
		os.Exit(1)
	}

	// Fixture-only launch delay (readiness.go, G-08-5) — off by default
	// (see launchDelayFromEnv), so a real installation's mock is
	// unaffected. A malformed value fails startup loudly, exactly as the
	// readiness-window branch above. On a non-zero duration, sleep BEFORE
	// goplugin.Serve: the handshake line is not written until the sleep
	// ends, so the kernel's pluginhost.launch blocks for that long — the
	// whole point of this fixture.
	delay, err := launchDelayFromEnv(os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "topos-plugin-mock:", err)
		os.Exit(1)
	}
	if delay > 0 {
		time.Sleep(delay)
	}

	// Fixture-only rendition (renditionfixture.go, 09-04-PLAN.md Task 3) —
	// off by default (see renditionFixtureEnabled), so a real installation's
	// mock is unaffected. Unlike the two fixtures above, an unset/invalid
	// value is never a startup error — it is a simple boolean gate, exactly
	// mirroring readiness.go's off-by-default posture but with no malformed
	// value to reject.
	renditionFixture := renditionFixtureEnabled(os.Getenv)

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: sdk.Handshake,
		Plugins: map[string]goplugin.Plugin{
			"source": &sdk.SourcePluginGRPCPlugin{
				Impl: NewSourcePlugin().withReadinessWindow(ready).withRenditionFixture(renditionFixture),
			},
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
