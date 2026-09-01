// Package sdk provides the shared handshake, plugin map, and go-plugin
// GRPCPlugin adapter used by both the kernel (host side) and every source
// plugin (plugin side). This is the stable surface third-party plugin
// authors build against alongside proto/topos/v1/plugin.proto.
package sdk

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

// Handshake is the common handshake shared by the kernel and every plugin.
// ProtocolVersion must be bumped only for breaking wire-protocol changes,
// not for additive contract changes (which the SourcePlugin.Describe RPC's
// contract_version field is for).
//
// ProtocolVersion 2 (Phase 5, 05-RESEARCH.md Pitfall 5): MatchRequest's
// shape changed from a flat `keywords` list to a typed `match_fields` map
// — a breaking wire change, not an additive one. This is the deliberate
// fail-fast for that break: a plugin binary built against ProtocolVersion
// 1 fails cleanly at the go-plugin handshake (before Describe or Match is
// ever called), never confusingly at its first Match call with an empty or
// misinterpreted match map.
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  2,
	MagicCookieKey:   "TOPOS_PLUGIN",
	MagicCookieValue: "topos-source-plugin-v1",
}

// ContractVersion is the contract GENERATION this sdk (and the kernel
// built beside it) speaks — the value a plugin built from this module
// returns in DescribeResponse.contract_version ("topos.v2" since Phase
// 5's typed match-field break; see proto/topos/v1/plugin.proto). One
// authority for the kernel's launch gate, both reference mocks, and
// third-party authors alike (M1-R6/DIST-03, davison/topos#17) — never
// retype the literal.
const ContractVersion = "topos.v2"

// SupportedContractVersions is the closed set of contract generations
// the kernel accepts at launch — exactly the current generation today.
// A future kernel that can still serve an older generation's
// MatchRequest shape widens this set additively; the launch gate
// (kernel/pluginhost's contract check) reads it through
// ContractSupported and refuses, loudly and by name, any plugin whose
// Describe declares a generation outside it.
var SupportedContractVersions = []string{ContractVersion}

// ContractSupported reports whether declared names a contract generation
// in SupportedContractVersions. The empty string is NEVER supported — a
// plugin that declares no generation at all predates (or ignores) the
// field, and silence is not compatibility (M1-R6): the kernel refuses it
// by name rather than guessing.
func ContractSupported(declared string) bool {
	for _, v := range SupportedContractVersions {
		if declared == v {
			return true
		}
	}
	return false
}

// PluginMap is the map of plugin name to implementation passed to both
// plugin.Serve (plugin side) and plugin.ClientConfig (host side). Every
// source plugin registers under the "source" key.
var PluginMap = map[string]plugin.Plugin{
	"source": &SourcePluginGRPCPlugin{},
}

// MaxMessageSize raises gRPC's default 4 MB message-size ceiling on both
// the plugin (server) and kernel (client) sides. Decision D-Task1 (01-01)
// locked Fetch as a single unary RPC carrying a rendition's full bytes in
// one message rather than a stream — this constant is what makes that
// decision viable for real scanned-PDF rendition sizes. 64 MiB comfortably
// covers a scanned-PDF preview or thumbnail; documents materially larger
// than this are expected to fail with a clear gRPC ResourceExhausted error
// rather than succeed silently truncated.
const MaxMessageSize = 64 * 1024 * 1024

// GRPCServer is passed as plugin.ServeConfig.GRPCServer on the plugin
// side in place of plugin.DefaultGRPCServer, applying MaxMessageSize to
// the server's receive/send limits.
func GRPCServer(opts []grpc.ServerOption) *grpc.Server {
	opts = append(opts,
		grpc.MaxRecvMsgSize(MaxMessageSize),
		grpc.MaxSendMsgSize(MaxMessageSize),
	)
	return grpc.NewServer(opts...)
}

// SourcePlugin is the Go interface plugin authors implement, mirroring the
// four RPCs declared in plugin.proto. Implementing this interface rather
// than the raw generated gRPC server type is the documented plugin
// contract surface.
type SourcePlugin interface {
	Describe(ctx context.Context, req *toposv1.DescribeRequest) (*toposv1.DescribeResponse, error)
	Match(ctx context.Context, req *toposv1.MatchRequest) (*toposv1.MatchResponse, error)
	Fetch(ctx context.Context, req *toposv1.FetchRequest) (*toposv1.FetchResponse, error)
	Health(ctx context.Context, req *toposv1.HealthRequest) (*toposv1.HealthResponse, error)
}

// ContentSearcher is the OPTIONAL Search RPC (M2-R2, davison/topos#50) as
// a separate interface, so SourcePlugin gains no method and every
// existing plugin compiles unchanged. A plugin that implements it must
// also declare searches_content in Describe; the plugin-side gRPC server
// discovers it by type assertion and answers Unimplemented otherwise.
// The kernel-side client always exposes Search; the kernel treats
// Unimplemented as "no body search from this source".
type ContentSearcher interface {
	Search(ctx context.Context, req *toposv1.SearchRequest) (*toposv1.SearchResponse, error)
}

// SourcePluginGRPCPlugin implements plugin.GRPCPlugin, wiring a SourcePlugin
// Go implementation to the generated gRPC server/client on the plugin and
// host sides respectively.
type SourcePluginGRPCPlugin struct {
	plugin.Plugin
	Impl SourcePlugin // concrete implementation, plugin-process side only
}

// GRPCServer registers the generated SourcePlugin gRPC server backed by Impl.
// Called on the plugin-process side.
func (p *SourcePluginGRPCPlugin) GRPCServer(broker *plugin.GRPCBroker, s *grpc.Server) error {
	toposv1.RegisterSourcePluginServer(s, &grpcServer{impl: p.Impl})
	return nil
}

// GRPCClient returns a SourcePlugin implementation backed by the generated
// gRPC client. Called on the kernel (host-process) side.
func (p *SourcePluginGRPCPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &grpcClient{client: toposv1.NewSourcePluginClient(c)}, nil
}

// grpcServer adapts a SourcePlugin implementation to the generated
// toposv1.SourcePluginServer interface.
type grpcServer struct {
	toposv1.UnimplementedSourcePluginServer
	impl SourcePlugin
}

func (s *grpcServer) Describe(ctx context.Context, req *toposv1.DescribeRequest) (*toposv1.DescribeResponse, error) {
	return s.impl.Describe(ctx, req)
}

func (s *grpcServer) Match(ctx context.Context, req *toposv1.MatchRequest) (*toposv1.MatchResponse, error) {
	return s.impl.Match(ctx, req)
}

func (s *grpcServer) Fetch(ctx context.Context, req *toposv1.FetchRequest) (*toposv1.FetchResponse, error) {
	return s.impl.Fetch(ctx, req)
}

func (s *grpcServer) Health(ctx context.Context, req *toposv1.HealthRequest) (*toposv1.HealthResponse, error) {
	return s.impl.Health(ctx, req)
}

// Search dispatches to the implementation only if it opted in (ContentSearcher);
// otherwise the plugin declines exactly as an older plugin without the RPC
// would — Unimplemented, which the kernel reads as a declared absence.
func (s *grpcServer) Search(ctx context.Context, req *toposv1.SearchRequest) (*toposv1.SearchResponse, error) {
	if cs, ok := s.impl.(ContentSearcher); ok {
		return cs.Search(ctx, req)
	}
	return nil, status.Error(codes.Unimplemented, "this plugin does not search its own content")
}

// grpcClient adapts the generated toposv1.SourcePluginClient to the
// SourcePlugin interface, used host-side after Dispense.
type grpcClient struct {
	client toposv1.SourcePluginClient
}

func (c *grpcClient) Describe(ctx context.Context, req *toposv1.DescribeRequest) (*toposv1.DescribeResponse, error) {
	return c.client.Describe(ctx, req)
}

func (c *grpcClient) Match(ctx context.Context, req *toposv1.MatchRequest) (*toposv1.MatchResponse, error) {
	return c.client.Match(ctx, req)
}

func (c *grpcClient) Fetch(ctx context.Context, req *toposv1.FetchRequest) (*toposv1.FetchResponse, error) {
	return c.client.Fetch(ctx, req)
}

func (c *grpcClient) Health(ctx context.Context, req *toposv1.HealthRequest) (*toposv1.HealthResponse, error) {
	return c.client.Health(ctx, req)
}

// Search on the kernel side always exists; a plugin without the RPC
// answers Unimplemented over the wire.
func (c *grpcClient) Search(ctx context.Context, req *toposv1.SearchRequest) (*toposv1.SearchResponse, error) {
	return c.client.Search(ctx, req)
}
