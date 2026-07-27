// Package pluginhost discovers, launches, and supervises source plugin
// subprocesses over the go-plugin gRPC transport. It is the only package
// permitted to import hashicorp/go-plugin's client side; kernel/httpapi
// must never import this package (see kernel/httpapi/stream.go).
package pluginhost

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"

	"github.com/darrendavison/webspaces/kernel/config"
	"github.com/darrendavison/webspaces/sdk"
	webspacesv1 "github.com/darrendavison/webspaces/sdk/gen/webspaces/v1"
)

// Plugin is one launched, handshaken source plugin subprocess.
type Plugin struct {
	name       string // config key under [sources.<name>]
	sourceType string // learned via Describe, not trusted from the filename
	client     *goplugin.Client
	impl       sdk.SourcePlugin
}

// Name returns the config key this plugin was launched under (under
// [sources.<name>]).
func (p *Plugin) Name() string { return p.name }

// SourceType returns the source_type learned from the plugin's own
// Describe RPC response — never trusted from the filename (T-01-07).
func (p *Plugin) SourceType() string { return p.sourceType }

// Match calls the plugin's Match RPC. Satisfies correlate.Source.
func (p *Plugin) Match(ctx context.Context, keywords []string) (*webspacesv1.MatchResponse, error) {
	return p.impl.Match(ctx, &webspacesv1.MatchRequest{Keywords: keywords})
}

// Fetch calls the plugin's Fetch RPC.
func (p *Plugin) Fetch(ctx context.Context, sourceID string, variant webspacesv1.ContentVariant) (*webspacesv1.FetchResponse, error) {
	return p.impl.Fetch(ctx, &webspacesv1.FetchRequest{SourceId: sourceID, Variant: variant})
}

// Health calls the plugin's Health RPC.
func (p *Plugin) Health(ctx context.Context) (*webspacesv1.HealthResponse, error) {
	return p.impl.Health(ctx, &webspacesv1.HealthRequest{})
}

// Kill terminates the plugin subprocess.
func (p *Plugin) Kill() {
	p.client.Kill()
}

// Host owns the lifecycle of every launched plugin subprocess.
type Host struct {
	plugins []*Plugin
}

// Discover launches one subprocess per configured source, in pluginsDir,
// and returns the resulting Host. Every launched plugin's Describe RPC is
// called immediately to learn its source_type — the kernel never trusts a
// plugin's identity from its filename (T-01-07).
func Discover(ctx context.Context, pluginsDir string, sources map[string]config.Source, logger hclog.Logger) (*Host, error) {
	h := &Host{}

	for name, src := range sources {
		p, err := launch(ctx, pluginsDir, name, src, logger)
		if err != nil {
			h.Shutdown()
			return nil, fmt.Errorf("pluginhost: launch source %q: %w", name, err)
		}
		h.plugins = append(h.plugins, p)
	}

	return h, nil
}

func launch(ctx context.Context, pluginsDir, name string, src config.Source, logger hclog.Logger) (*Plugin, error) {
	binPath := filepath.Join(pluginsDir, src.Plugin)
	if _, err := os.Stat(binPath); err != nil {
		return nil, fmt.Errorf("plugin binary %s not found: %w", binPath, err)
	}

	sourceConfig, err := json.Marshal(map[string]string{
		"base_url":    src.BaseURL,
		"token":       src.Token,
		"api_version": src.APIVersion,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal source config: %w", err)
	}

	cmd := exec.Command(binPath)
	cmd.Env = append(os.Environ(), "WEBSPACES_SOURCE_CONFIG="+string(sourceConfig))

	client := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig:  sdk.Handshake,
		Plugins:          sdk.PluginMap,
		Cmd:              cmd,
		AllowedProtocols: []goplugin.Protocol{goplugin.ProtocolGRPC},
		Logger:           logger.Named("plugin." + name),
	})

	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("connect to plugin subprocess: %w", err)
	}

	raw, err := rpcClient.Dispense("source")
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("dispense source plugin: %w", err)
	}

	impl, ok := raw.(sdk.SourcePlugin)
	if !ok {
		client.Kill()
		return nil, fmt.Errorf("plugin %s does not implement sdk.SourcePlugin", name)
	}

	desc, err := impl.Describe(ctx, &webspacesv1.DescribeRequest{})
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("describe plugin %s: %w", name, err)
	}

	return &Plugin{
		name:       name,
		sourceType: desc.GetSourceType(),
		client:     client,
		impl:       impl,
	}, nil
}

// Plugins returns every launched plugin.
func (h *Host) Plugins() []*Plugin {
	return h.plugins
}

// Shutdown kills every launched plugin subprocess.
func (h *Host) Shutdown() {
	for _, p := range h.plugins {
		p.Kill()
	}
}
