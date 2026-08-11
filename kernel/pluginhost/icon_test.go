package pluginhost

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/davison/topos/kernel/config"
	"github.com/davison/topos/sdk"
	toposv1 "github.com/davison/topos/sdk/gen/topos/v1"
)

// readMockIconAsset reads plugins/mock/assets/icon.svg straight off disk
// (relative to this package's directory), mirroring what
// plugins/mock/plugin.go's go:embed captures.
func readMockIconAsset(t *testing.T) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "plugins", "mock", "assets", "icon.svg"))
	if err != nil {
		t.Fatalf("read plugins/mock/assets/icon.svg: %v", err)
	}
	return b
}

// fakeIconDescriber is a minimal, in-process sdk.SourcePlugin whose
// Describe returns a fixed response — used instead of a real subprocess
// (the existing tests in this package launch one via buildMockPluginDir)
// so this file can exercise validation edge cases (oversized icon, bad
// mime, ...) that a real embedded asset can't represent.
type fakeIconDescriber struct {
	desc *toposv1.DescribeResponse
}

func (f *fakeIconDescriber) Describe(_ context.Context, _ *toposv1.DescribeRequest) (*toposv1.DescribeResponse, error) {
	return f.desc, nil
}
func (f *fakeIconDescriber) Match(_ context.Context, _ *toposv1.MatchRequest) (*toposv1.MatchResponse, error) {
	return &toposv1.MatchResponse{}, nil
}
func (f *fakeIconDescriber) Fetch(_ context.Context, _ *toposv1.FetchRequest) (*toposv1.FetchResponse, error) {
	return &toposv1.FetchResponse{}, nil
}
func (f *fakeIconDescriber) Health(_ context.Context, _ *toposv1.HealthRequest) (*toposv1.HealthResponse, error) {
	return &toposv1.HealthResponse{Reachable: true}, nil
}

var _ sdk.SourcePlugin = (*fakeIconDescriber)(nil)

// describeVia calls fakeIconDescriber's Describe RPC and feeds the result
// into captureIcon — the identical two-step sequence launch() performs at
// its own impl.Describe(...) call site, with no subprocess involved.
func describeVia(t *testing.T, desc *toposv1.DescribeResponse) ([]byte, string, bool) {
	t.Helper()
	fake := &fakeIconDescriber{desc: desc}
	got, err := fake.Describe(context.Background(), &toposv1.DescribeRequest{})
	if err != nil {
		t.Fatalf("fake Describe: %v", err)
	}
	return captureIcon(got)
}

// TestCaptureIcon_ValidSVGIsExposed proves a launched plugin whose Describe
// returns non-empty icon bytes with mime image/svg+xml exposes those bytes
// and that mime.
func TestCaptureIcon_ValidSVGIsExposed(t *testing.T) {
	want := []byte("<svg></svg>")
	b, m, ok := describeVia(t, &toposv1.DescribeResponse{Icon: want, IconMime: "image/svg+xml"})
	if !ok {
		t.Fatal("expected ok=true for a valid SVG icon")
	}
	if !bytes.Equal(b, want) {
		t.Errorf("expected icon bytes %q, got %q", want, b)
	}
	if m != "image/svg+xml" {
		t.Errorf("expected mime image/svg+xml, got %q", m)
	}
}

// TestCaptureIcon_ValidPNGIsExposed proves the same for the other allowed
// mime, image/png.
func TestCaptureIcon_ValidPNGIsExposed(t *testing.T) {
	want := []byte{0x89, 'P', 'N', 'G'}
	b, m, ok := describeVia(t, &toposv1.DescribeResponse{Icon: want, IconMime: "image/png"})
	if !ok {
		t.Fatal("expected ok=true for a valid PNG icon")
	}
	if !bytes.Equal(b, want) {
		t.Errorf("expected icon bytes %v, got %v", want, b)
	}
	if m != "image/png" {
		t.Errorf("expected mime image/png, got %q", m)
	}
}

// TestCaptureIcon_EmptyIconIsNoIconDeclared proves the pre-Phase-9-plugin
// case: a Describe with empty icon bytes exposes no icon (ok=false), and
// is never treated as a launch failure — captureIcon has no error return,
// so this is structurally guaranteed, not merely asserted here.
func TestCaptureIcon_EmptyIconIsNoIconDeclared(t *testing.T) {
	_, _, ok := describeVia(t, &toposv1.DescribeResponse{})
	if ok {
		t.Error("expected ok=false for an icon-less DescribeResponse")
	}
}

// TestCaptureIcon_OversizedIconIsDropped proves an icon larger than
// MaxIconBytes is dropped (ok=false), never truncated.
func TestCaptureIcon_OversizedIconIsDropped(t *testing.T) {
	oversized := bytes.Repeat([]byte{'a'}, MaxIconBytes+1)
	_, _, ok := describeVia(t, &toposv1.DescribeResponse{Icon: oversized, IconMime: "image/svg+xml"})
	if ok {
		t.Error("expected ok=false for an icon larger than MaxIconBytes")
	}
}

// TestCaptureIcon_AtMaxIconBytesIsKept proves the boundary is inclusive —
// exactly MaxIconBytes is accepted, only MaxIconBytes+1 is dropped.
func TestCaptureIcon_AtMaxIconBytesIsKept(t *testing.T) {
	atMax := bytes.Repeat([]byte{'a'}, MaxIconBytes)
	_, _, ok := describeVia(t, &toposv1.DescribeResponse{Icon: atMax, IconMime: "image/svg+xml"})
	if !ok {
		t.Error("expected ok=true for an icon exactly MaxIconBytes long")
	}
}

// TestCaptureIcon_DisallowedMimeIsDropped proves a mime outside
// {image/svg+xml, image/png} has the icon dropped entirely.
func TestCaptureIcon_DisallowedMimeIsDropped(t *testing.T) {
	_, _, ok := describeVia(t, &toposv1.DescribeResponse{Icon: []byte("<html>"), IconMime: "text/html"})
	if ok {
		t.Error("expected ok=false for a disallowed mime (text/html)")
	}
}

// TestCaptureIcon_NonEmptyIconEmptyMimeIsDropped proves a plugin cannot
// declare icon bytes with no mime and have them served with a guessed
// Content-Type — the kernel never trusts a mime it wasn't explicitly
// given.
func TestCaptureIcon_NonEmptyIconEmptyMimeIsDropped(t *testing.T) {
	_, _, ok := describeVia(t, &toposv1.DescribeResponse{Icon: []byte("<svg></svg>"), IconMime: ""})
	if ok {
		t.Error("expected ok=false for non-empty icon bytes with an empty mime")
	}
}

// TestHostPluginIcon_ResolvesByBinaryName proves Host.PluginIcon("topos-
// plugin-mock") returns the bytes of any launched instance of that binary
// — constructed directly (in-package) rather than via a real launch, since
// Plugin's icon fields are unexported and this is exactly the seam
// PluginIcon itself resolves over.
func TestHostPluginIcon_ResolvesByBinaryName(t *testing.T) {
	want := []byte("<svg>mock</svg>")
	h := &Host{plugins: []*Plugin{
		{
			name:      "house-move-mock",
			src:       config.Source{Plugin: "topos-plugin-mock"},
			iconBytes: want,
			iconMIME:  "image/svg+xml",
		},
	}}

	b, m, ok := h.PluginIcon("topos-plugin-mock")
	if !ok {
		t.Fatal("expected ok=true for a launched instance of topos-plugin-mock")
	}
	if !bytes.Equal(b, want) {
		t.Errorf("expected icon bytes %q, got %q", want, b)
	}
	if m != "image/svg+xml" {
		t.Errorf("expected mime image/svg+xml, got %q", m)
	}
}

// TestHostPluginIcon_TwoInstancesOfSameBinaryBothResolve proves the
// "which instance" choice doesn't matter — two instances of one plugin
// binary have byte-identical icons (both came from the same Describe
// implementation), so PluginIcon resolves via whichever launched instance
// it finds first.
func TestHostPluginIcon_TwoInstancesOfSameBinaryBothResolve(t *testing.T) {
	want := []byte("<svg>paperless</svg>")
	h := &Host{plugins: []*Plugin{
		{name: "home-docs", src: config.Source{Plugin: "topos-plugin-paperless"}, iconBytes: want, iconMIME: "image/svg+xml"},
		{name: "work-docs", src: config.Source{Plugin: "topos-plugin-paperless"}, iconBytes: want, iconMIME: "image/svg+xml"},
	}}

	b, _, ok := h.PluginIcon("topos-plugin-paperless")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if !bytes.Equal(b, want) {
		t.Errorf("expected icon bytes %q, got %q", want, b)
	}
}

// TestHostPluginIcon_NoLaunchedInstanceIsNotFound proves a binary with no
// launched instance anywhere returns ok=false — the "undescribed plugin
// type" case the kernel icon endpoint 404s for.
func TestHostPluginIcon_NoLaunchedInstanceIsNotFound(t *testing.T) {
	h := &Host{plugins: []*Plugin{
		{name: "house-move-mock", src: config.Source{Plugin: "topos-plugin-mock"}, iconBytes: []byte("x"), iconMIME: "image/svg+xml"},
	}}

	_, _, ok := h.PluginIcon("topos-plugin-does-not-exist")
	if ok {
		t.Error("expected ok=false for a binary with no launched instance")
	}
}

// TestHostPluginIcon_LaunchedInstanceWithoutIconIsNotFound proves a
// launched instance whose own icon capture failed (pre-Phase-9 binary, or
// a validation drop) also resolves to ok=false, not a zero-byte "icon."
func TestHostPluginIcon_LaunchedInstanceWithoutIconIsNotFound(t *testing.T) {
	h := &Host{plugins: []*Plugin{
		{name: "legacy-instance", src: config.Source{Plugin: "topos-plugin-legacy"}},
	}}

	_, _, ok := h.PluginIcon("topos-plugin-legacy")
	if ok {
		t.Error("expected ok=false for a launched instance with no captured icon")
	}
}

// TestPluginIcon_AccessorMirrorsCaptureIcon proves Plugin.Icon() applies
// the identical "both bytes and mime must be non-empty" rule captureIcon
// itself enforces at capture time, guarding against a future direct
// struct-literal construction (as this file's own fixtures do) leaving
// just one of the two fields set.
func TestPluginIcon_AccessorMirrorsCaptureIcon(t *testing.T) {
	cases := []struct {
		name  string
		bytes []byte
		mime  string
		ok    bool
	}{
		{"both set", []byte("x"), "image/svg+xml", true},
		{"bytes only", []byte("x"), "", false},
		{"mime only", nil, "image/svg+xml", false},
		{"neither", nil, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := &Plugin{iconBytes: tc.bytes, iconMIME: tc.mime}
			_, _, ok := p.Icon()
			if ok != tc.ok {
				t.Errorf("Icon() ok = %v, want %v", ok, tc.ok)
			}
		})
	}
}

// TestCaptureIcon_FromRealMockPluginDescribe is a light integration check
// (not a subprocess launch) that plugins/mock's actual embedded icon.svg
// asset — read straight off disk, mirroring what Describe returns — passes
// captureIcon's own validation: catches the case where the real asset
// itself violates the size/mime contract the unit tests above only
// exercise with synthetic fixtures.
func TestCaptureIcon_FromRealMockPluginDescribe(t *testing.T) {
	svg := readMockIconAsset(t)
	if len(svg) == 0 {
		t.Fatal("expected plugins/mock/assets/icon.svg to be non-empty")
	}
	if !strings.Contains(string(svg), "<svg") {
		t.Fatalf("expected plugins/mock/assets/icon.svg to contain an <svg> element, got: %s", svg)
	}

	_, _, ok := describeVia(t, &toposv1.DescribeResponse{Icon: svg, IconMime: "image/svg+xml"})
	if !ok {
		t.Error("expected the real mock plugin icon asset to pass captureIcon validation")
	}
}
