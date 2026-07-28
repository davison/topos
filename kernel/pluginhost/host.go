// Package pluginhost discovers, launches, and supervises source plugin
// subprocesses over the go-plugin gRPC transport. It is the only package
// permitted to import hashicorp/go-plugin's client side; kernel/httpapi
// must never import this package (see kernel/httpapi/stream.go).
package pluginhost

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/davison/webspaces/kernel/config"
	"github.com/davison/webspaces/sdk"
	webspacesv1 "github.com/davison/webspaces/sdk/gen/webspaces/v1"
)

// ErrItemNotFound is returned by Host.Fetch when the plugin reports the
// requested source id does not exist (gRPC codes.NotFound), or when no
// launched plugin is registered for the item's source type.
var ErrItemNotFound = errors.New("pluginhost: item not found")

// ErrSourceUnavailable is returned by Host.Fetch when the plugin's live
// call to its source system failed transiently (gRPC codes.Unavailable,
// or any other transport-level failure calling the plugin).
var ErrSourceUnavailable = errors.New("pluginhost: source unavailable")

// Plugin is one launched, handshaken source plugin subprocess.
type Plugin struct {
	name        string // config key under [sources.<name>]
	sourceType  string // learned via Describe, not trusted from the filename
	displayName string // learned via Describe
	client      *goplugin.Client
	impl        sdk.SourcePlugin
}

// Name returns the config key this plugin was launched under (under
// [sources.<name>]).
func (p *Plugin) Name() string { return p.name }

// SourceType returns the source_type learned from the plugin's own
// Describe RPC response — never trusted from the filename (T-01-07).
func (p *Plugin) SourceType() string { return p.sourceType }

// DisplayName returns the display_name learned from the plugin's own
// Describe RPC response (e.g. "paperless-ngx", "SilverBullet") — the
// health/sources API surfaces this so the UI never hardcodes a
// per-source display string (02-01-PLAN.md's sourceDisplayName fix is
// the local-mapping predecessor this makes obsolete).
func (p *Plugin) DisplayName() string { return p.displayName }

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
		"ca_cert":     src.CACert,
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
		// Raises the gRPC message-size ceiling to match sdk.GRPCServer on
		// the plugin side, so a unary Fetch response carrying a full
		// rendition doesn't hit the 4 MB default (D-Task1, 01-01).
		GRPCDialOptions: []grpc.DialOption{
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(sdk.MaxMessageSize),
				grpc.MaxCallSendMsgSize(sdk.MaxMessageSize),
			),
		},
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
		name:        name,
		sourceType:  desc.GetSourceType(),
		displayName: desc.GetDisplayName(),
		client:      client,
		impl:        impl,
	}, nil
}

// Plugins returns every launched plugin.
func (h *Host) Plugins() []*Plugin {
	return h.plugins
}

// SourceTypesByName returns the config name ([sources.<name>]) to
// Describe-learned source_type mapping for every launched plugin, built
// from the already-cached launch-time Describe results — this issues no
// RPC. kernel/httpapi/agent.go calls this on every /agent/v1 request to
// resolve which source_types the config's agent.read grants apply to, and
// must not pay a live network probe cost just to answer that question.
func (h *Host) SourceTypesByName() map[string]string {
	out := make(map[string]string, len(h.plugins))
	for _, p := range h.plugins {
		out[p.name] = p.sourceType
	}
	return out
}

// SourceHealth is one plugin's live reachability probe result, keyed to
// its config name and Describe-learned identity. 02-02-PLAN.md's D-08:
// this is a liveness signal only — last-sync time and last-error are
// deliberately NOT part of this type, because those come from the
// kernel's own sync_runs history (kernel/index.Store), never from a
// plugin's self-reported HealthResponse fields (A-PLUG-04). A plugin that
// reports a rosier history than the kernel actually recorded must not be
// able to turn its own health chip green.
type SourceHealth struct {
	Name        string // config key under [sources.<name>]
	SourceType  string // Describe-learned
	DisplayName string // Describe-learned
	Reachable   bool
	ProbeError  string
}

// ProbeSources calls every launched plugin's Health RPC concurrently — a
// live reachability probe, not a cached value — and returns one
// SourceHealth per plugin, in launch order (so the response order is
// stable run to run, matching the deterministic ordering
// kernel/httpapi.SourcesHandler sorts on top of). A transport error or a
// Reachable:false response both map to Reachable:false with the error
// text in ProbeError; ProbeSources itself never returns a Go error — one
// plugin being unreachable is data, not a failure of the whole probe.
func (h *Host) ProbeSources(ctx context.Context) []SourceHealth {
	out := make([]SourceHealth, len(h.plugins))
	var wg sync.WaitGroup
	for i, p := range h.plugins {
		wg.Add(1)
		go func(i int, p *Plugin) {
			defer wg.Done()
			health := SourceHealth{Name: p.Name(), SourceType: p.SourceType(), DisplayName: p.DisplayName()}
			resp, err := p.Health(ctx)
			switch {
			case err != nil:
				health.Reachable = false
				health.ProbeError = err.Error()
			case !resp.GetReachable():
				health.Reachable = false
				health.ProbeError = resp.GetLastError()
			default:
				health.Reachable = true
			}
			out[i] = health
		}(i, p)
	}
	wg.Wait()
	return out
}

// FetchResult is the kernel-domain translation of a plugin's unary Fetch
// response. Body is nil when there is no rendition (available=false, or a
// full-variant fetch that carried only extracted text).
type FetchResult struct {
	Available         bool
	UnavailableReason string
	MimeType          string
	SizeBytes         int64
	Text              string
	Provenance        map[string]string
	Body              io.ReadCloser
}

// Fetch calls the Fetch RPC on the plugin registered for sourceType,
// request-time only (never sync-time), and translates the result into
// kernel-domain types and errors. gRPC codes.NotFound maps to
// ErrItemNotFound, codes.Unavailable and any other transport failure maps
// to ErrSourceUnavailable.
func (h *Host) Fetch(ctx context.Context, sourceType, sourceID string, variant webspacesv1.ContentVariant) (FetchResult, error) {
	p := h.bySourceType(sourceType)
	if p == nil {
		return FetchResult{}, fmt.Errorf("%w: no plugin registered for source type %q", ErrItemNotFound, sourceType)
	}

	resp, err := p.impl.Fetch(ctx, &webspacesv1.FetchRequest{SourceId: sourceID, Variant: variant})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				return FetchResult{}, fmt.Errorf("%w: %s", ErrItemNotFound, st.Message())
			case codes.Unavailable:
				return FetchResult{}, fmt.Errorf("%w: %s", ErrSourceUnavailable, st.Message())
			}
		}
		return FetchResult{}, fmt.Errorf("%w: %v", ErrSourceUnavailable, err)
	}

	var body io.ReadCloser
	if len(resp.GetData()) > 0 {
		body = io.NopCloser(bytes.NewReader(resp.GetData()))
	}

	return FetchResult{
		Available:         resp.GetAvailable(),
		UnavailableReason: resp.GetUnavailableReason(),
		MimeType:          resp.GetMimeType(),
		SizeBytes:         resp.GetSizeBytes(),
		Text:              resp.GetText(),
		Provenance:        resp.GetProvenance(),
		Body:              body,
	}, nil
}

// bySourceType returns the launched plugin whose Describe-learned
// source_type matches, or nil if none is registered.
func (h *Host) bySourceType(sourceType string) *Plugin {
	for _, p := range h.plugins {
		if p.sourceType == sourceType {
			return p
		}
	}
	return nil
}

// Shutdown kills every launched plugin subprocess.
func (h *Host) Shutdown() {
	for _, p := range h.plugins {
		p.Kill()
	}
}
