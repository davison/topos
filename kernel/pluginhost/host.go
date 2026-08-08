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
	"reflect"
	"sort"
	"sync"

	"github.com/hashicorp/go-hclog"
	goplugin "github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/sdk"
	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
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
	name            string // config key under [sources.<name>] — THE instance identity (D-08)
	sourceType      string // learned via Describe, not trusted from the filename — plugin kind only, never identity
	pluginName      string // Describe-learned display name (e.g. "paperless-ngx") — the plugin KIND's own label
	displayName     string // resolved instance display name (config display_name, or name if unset) — D-09
	matchVocabulary []string
	// src is the config.Source this instance was launched with — Reconcile
	// (07-02-PLAN.md Task 1, D-06/D-07) compares a currently-configured
	// instance's config.Source against this value to decide whether the
	// instance is unchanged (subprocess left running untouched) or has a
	// different connection config (subprocess relaunched). Never used for
	// anything beyond that diff.
	src    config.Source
	client *goplugin.Client
	impl   sdk.SourcePlugin
}

// Name returns the config key this plugin was launched under (under
// [sources.<name>]) — the source instance's identity (D-08).
func (p *Plugin) Name() string { return p.name }

// SourceType returns the source_type learned from the plugin's own
// Describe RPC response — never trusted from the filename (T-01-07). This
// is the plugin KIND, never an identity key: two launched instances of one
// plugin binary share the same SourceType but always have distinct Name().
func (p *Plugin) SourceType() string { return p.sourceType }

// DisplayName returns this source INSTANCE's resolved display name (D-09):
// its config-authored [sources.<name>] display_name, or the instance id
// itself when that key is omitted. This is what the health/sources API and
// the UI show for this specific instance — distinct from PluginDisplayName,
// which names the plugin KIND regardless of how many instances of it are
// configured.
func (p *Plugin) DisplayName() string { return p.displayName }

// PluginDisplayName returns the Describe-learned display name for the
// plugin KIND this instance runs (e.g. "paperless-ngx", "SilverBullet") —
// the same value every instance of this plugin binary reports, regardless
// of its own instance display name. Provenance/diagnostic use only; never
// a substitute for DisplayName in anything the UI renders as a source
// label (02-01-PLAN.md's sourceDisplayName fix is the local-mapping
// predecessor this makes obsolete).
func (p *Plugin) PluginDisplayName() string { return p.pluginName }

// MatchVocabulary returns the field-name vocabulary this plugin declared in
// its Describe response (DescribeResponse.match_vocabulary) — the set of
// keys correlate.matchFieldsFor may populate in a Match request sent to
// this instance. Satisfies correlate.Source.
func (p *Plugin) MatchVocabulary() []string { return p.matchVocabulary }

// Match calls the plugin's Match RPC, wrapping each declared field's value
// list in a StringList (proto3 map values cannot be repeated fields
// directly). Satisfies correlate.Source.
func (p *Plugin) Match(ctx context.Context, fields map[string][]string) (*toposv1.MatchResponse, error) {
	matchFields := make(map[string]*toposv1.StringList, len(fields))
	for k, v := range fields {
		matchFields[k] = &toposv1.StringList{Values: v}
	}
	return p.impl.Match(ctx, &toposv1.MatchRequest{MatchFields: matchFields})
}

// Fetch calls the plugin's Fetch RPC.
func (p *Plugin) Fetch(ctx context.Context, sourceID string, variant toposv1.ContentVariant) (*toposv1.FetchResponse, error) {
	return p.impl.Fetch(ctx, &toposv1.FetchRequest{SourceId: sourceID, Variant: variant})
}

// Health calls the plugin's Health RPC.
func (p *Plugin) Health(ctx context.Context) (*toposv1.HealthResponse, error) {
	return p.impl.Health(ctx, &toposv1.HealthRequest{})
}

// Kill terminates the plugin subprocess.
func (p *Plugin) Kill() {
	p.client.Kill()
}

// Host owns the lifecycle of every launched plugin subprocess.
type Host struct {
	// pluginsDir is the directory Discover launched every plugin binary
	// from — retained (07-02-PLAN.md Task 1) so Reconcile can launch a
	// replacement/added instance later without a caller having to thread
	// the directory through a second time.
	pluginsDir string
	plugins    []*Plugin
}

// Discover launches one subprocess per configured source, in pluginsDir,
// and returns the resulting Host. Every launched plugin's Describe RPC is
// called immediately to learn its source_type — the kernel never trusts a
// plugin's identity from its filename (T-01-07).
func Discover(ctx context.Context, pluginsDir string, sources map[string]config.Source, logger hclog.Logger) (*Host, error) {
	h := &Host{pluginsDir: pluginsDir}

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

// Reconcile brings the launched plugin set in line with sources, in place
// (07-02-PLAN.md Task 1, D-06/D-07's hot-apply): for every configured
// instance not currently launched, or currently launched with a different
// config.Source than sources now declares, it launches a replacement via
// the same unexported launch() Discover uses and kills the instance it
// replaces; every currently launched instance absent from sources is
// killed and dropped; every other instance's subprocess is left running
// completely untouched, so a save that edits one source's config never
// blips an unrelated source's reachability.
//
// Every new/changed launch is attempted before anything currently running
// is torn down: a launch failure kills only what THIS call itself
// launched, leaves the previously running set fully intact, and returns
// an error naming the offending instance — a partial apply must never
// look successful (T-07-11). Instance names are iterated in sorted order
// so which instance is reported first on a multi-failure config is
// deterministic run to run, matching this package's existing discipline
// (matchconfig.go).
func (h *Host) Reconcile(ctx context.Context, sources map[string]config.Source, logger hclog.Logger) error {
	existing := make(map[string]*Plugin, len(h.plugins))
	for _, p := range h.plugins {
		existing[p.name] = p
	}

	names := make([]string, 0, len(sources))
	for name := range sources {
		names = append(names, name)
	}
	sort.Strings(names)

	kept := make(map[string]*Plugin, len(names))
	var toLaunch []string
	for _, name := range names {
		p, ok := existing[name]
		if ok && reflect.DeepEqual(p.src, sources[name]) {
			kept[name] = p
			continue
		}
		toLaunch = append(toLaunch, name)
	}

	launched := make(map[string]*Plugin, len(toLaunch))
	for _, name := range toLaunch {
		p, err := launch(ctx, h.pluginsDir, name, sources[name], logger)
		if err != nil {
			for _, lp := range launched {
				lp.Kill()
			}
			return fmt.Errorf("pluginhost: reconcile: launch source %q: %w", name, err)
		}
		launched[name] = p
	}

	// Every launch this call needed has now succeeded — only now kill the
	// instances being replaced or removed, and commit the new plugin set.
	for name, p := range existing {
		if _, stillKept := kept[name]; stillKept {
			continue
		}
		p.Kill()
	}

	next := make([]*Plugin, 0, len(kept)+len(launched))
	for _, name := range names {
		if p, ok := kept[name]; ok {
			next = append(next, p)
			continue
		}
		next = append(next, launched[name])
	}
	h.plugins = next
	return nil
}

func launch(ctx context.Context, pluginsDir, name string, src config.Source, logger hclog.Logger) (*Plugin, error) {
	binPath := filepath.Join(pluginsDir, src.Plugin)
	if _, err := os.Stat(binPath); err != nil {
		return nil, fmt.Errorf("plugin binary %s not found: %w", binPath, err)
	}

	sourceConfig, err := json.Marshal(map[string]string{
		"base_url":         src.BaseURL,
		"token":            src.Token,
		"api_version":      src.APIVersion,
		"ca_cert":          src.CACert,
		"username":         src.Username,
		"webmail_base_url": src.WebmailBaseURL,
		// path: the local filesystem path source config field (Source.Path
		// — kernel/config/types.go), needed by a local-path source like
		// SRC-02's Signal plugin, which has no base_url/token at all.
		"path": src.Path,
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

	desc, err := impl.Describe(ctx, &toposv1.DescribeRequest{})
	if err != nil {
		client.Kill()
		return nil, fmt.Errorf("describe plugin %s: %w", name, err)
	}

	// The instance display name resolves from the operator's own config
	// (D-09), falling back to the instance id itself when display_name is
	// omitted — never from anything the plugin process asserts. This
	// mirrors config.Config.DisplayNameFor without needing the whole
	// *config.Config here, since launch already holds this instance's own
	// config.Source.
	instanceDisplayName := src.DisplayName
	if instanceDisplayName == "" {
		instanceDisplayName = name
	}

	return &Plugin{
		name:            name,
		sourceType:      desc.GetSourceType(),
		pluginName:      desc.GetDisplayName(),
		displayName:     instanceDisplayName,
		matchVocabulary: desc.GetMatchVocabulary(),
		src:             src,
		client:          client,
		impl:            impl,
	}, nil
}

// Plugins returns every launched plugin.
func (h *Host) Plugins() []*Plugin {
	return h.plugins
}

// DescribeInfo is the three Describe-derived facts the "+" chip picker's
// two-step modal needs to build a match-vocabulary-driven form for a
// plugin type that has no configured instance yet (D-11 step 1 -> step 2,
// 07-02-PLAN.md Task 3).
type DescribeInfo struct {
	SourceType        string
	PluginDisplayName string
	MatchVocabulary   []string
}

// DescribePluginType trial-launches src (a config.Source naming the
// plugin binary to run, submitted but not yet persisted anywhere) under a
// fixed, non-persisted instance name, calls its Describe RPC, and kills
// the subprocess before returning — writing nothing to disk and
// registering nothing on any *Host (D-11 step 1 -> step 2). It reuses the
// existing unexported launch() verbatim (the same binary-stat, handshake,
// dispense and Describe-call sequence Discover already performs per
// configured instance) rather than duplicating any of it, and calls no
// RPC beyond the one Describe call launch() already makes — the trial-
// launch path must never become a general plugin-invocation surface for
// request-supplied input (PLUG-02, T-07-10); see
// kernel/httpapi/config_test.go's AST guard pinning exactly this over
// this function's own body.
func DescribePluginType(ctx context.Context, pluginsDir string, src config.Source, logger hclog.Logger) (DescribeInfo, error) {
	p, err := launch(ctx, pluginsDir, "__trial__", src, logger)
	if err != nil {
		return DescribeInfo{}, fmt.Errorf("pluginhost: trial-launch for describe: %w", err)
	}
	defer p.Kill()

	return DescribeInfo{
		SourceType:        p.SourceType(),
		PluginDisplayName: p.PluginDisplayName(),
		MatchVocabulary:   p.MatchVocabulary(),
	}, nil
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
	// ContentShape carries FetchResponse.content_shape (D-11) into the
	// kernel domain — meaningful only when MimeType is "text/html";
	// kernel/httpapi.sanitizeAndWrapRendition uses it to select the
	// sanitize/wrap/theme policy for this rendition. Zero value
	// (CONTENT_SHAPE_UNSPECIFIED) for every other MimeType.
	ContentShape toposv1.ContentShape
}

// Fetch calls the Fetch RPC on the plugin registered for source (the
// source INSTANCE id supplied by an index row — item.Item.Source, D-08),
// request-time only (never sync-time), and translates the result into
// kernel-domain types and errors. gRPC codes.NotFound maps to
// ErrItemNotFound, codes.Unavailable and any other transport failure maps
// to ErrSourceUnavailable.
func (h *Host) Fetch(ctx context.Context, source, sourceID string, variant toposv1.ContentVariant) (FetchResult, error) {
	p := h.byInstance(source)
	if p == nil {
		return FetchResult{}, fmt.Errorf("%w: no plugin registered for source %q", ErrItemNotFound, source)
	}

	resp, err := p.impl.Fetch(ctx, &toposv1.FetchRequest{SourceId: sourceID, Variant: variant})
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
		ContentShape:      resp.GetContentShape(),
	}, nil
}

// byInstance returns the launched plugin whose config-key instance id
// (Name()) matches, or nil if none is registered. Keyed on instance id, not
// on the Describe-learned plugin kind, so two launched instances of one
// plugin binary resolve to two distinct *Plugin values (D-08).
func (h *Host) byInstance(source string) *Plugin {
	for _, p := range h.plugins {
		if p.name == source {
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
