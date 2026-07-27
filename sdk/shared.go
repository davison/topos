// Package sdk provides the shared handshake, plugin map, and go-plugin
// GRPCPlugin adapter used by both the kernel (host side) and every source
// plugin (plugin side). This is the stable surface third-party plugin
// authors build against alongside proto/webspaces/v1/plugin.proto.
package sdk

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"

	webspacesv1 "github.com/davison/webspaces/sdk/gen/webspaces/v1"
)

// Handshake is the common handshake shared by the kernel and every plugin.
// ProtocolVersion must be bumped only for breaking wire-protocol changes,
// not for additive contract changes (which the SourcePlugin.Describe RPC's
// contract_version field is for).
var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "WEBSPACES_PLUGIN",
	MagicCookieValue: "webspaces-source-plugin-v1",
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
	Describe(ctx context.Context, req *webspacesv1.DescribeRequest) (*webspacesv1.DescribeResponse, error)
	Match(ctx context.Context, req *webspacesv1.MatchRequest) (*webspacesv1.MatchResponse, error)
	Fetch(ctx context.Context, req *webspacesv1.FetchRequest) (*webspacesv1.FetchResponse, error)
	Health(ctx context.Context, req *webspacesv1.HealthRequest) (*webspacesv1.HealthResponse, error)
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
	webspacesv1.RegisterSourcePluginServer(s, &grpcServer{impl: p.Impl})
	return nil
}

// GRPCClient returns a SourcePlugin implementation backed by the generated
// gRPC client. Called on the kernel (host-process) side.
func (p *SourcePluginGRPCPlugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &grpcClient{client: webspacesv1.NewSourcePluginClient(c)}, nil
}

// grpcServer adapts a SourcePlugin implementation to the generated
// webspacesv1.SourcePluginServer interface.
type grpcServer struct {
	webspacesv1.UnimplementedSourcePluginServer
	impl SourcePlugin
}

func (s *grpcServer) Describe(ctx context.Context, req *webspacesv1.DescribeRequest) (*webspacesv1.DescribeResponse, error) {
	return s.impl.Describe(ctx, req)
}

func (s *grpcServer) Match(ctx context.Context, req *webspacesv1.MatchRequest) (*webspacesv1.MatchResponse, error) {
	return s.impl.Match(ctx, req)
}

func (s *grpcServer) Fetch(ctx context.Context, req *webspacesv1.FetchRequest) (*webspacesv1.FetchResponse, error) {
	return s.impl.Fetch(ctx, req)
}

func (s *grpcServer) Health(ctx context.Context, req *webspacesv1.HealthRequest) (*webspacesv1.HealthResponse, error) {
	return s.impl.Health(ctx, req)
}

// grpcClient adapts the generated webspacesv1.SourcePluginClient to the
// SourcePlugin interface, used host-side after Dispense.
type grpcClient struct {
	client webspacesv1.SourcePluginClient
}

func (c *grpcClient) Describe(ctx context.Context, req *webspacesv1.DescribeRequest) (*webspacesv1.DescribeResponse, error) {
	return c.client.Describe(ctx, req)
}

func (c *grpcClient) Match(ctx context.Context, req *webspacesv1.MatchRequest) (*webspacesv1.MatchResponse, error) {
	return c.client.Match(ctx, req)
}

func (c *grpcClient) Fetch(ctx context.Context, req *webspacesv1.FetchRequest) (*webspacesv1.FetchResponse, error) {
	return c.client.Fetch(ctx, req)
}

func (c *grpcClient) Health(ctx context.Context, req *webspacesv1.HealthRequest) (*webspacesv1.HealthResponse, error) {
	return c.client.Health(ctx, req)
}
