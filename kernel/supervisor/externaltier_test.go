// externaltier_test.go is Phase 11's own tracer proof at the supervisor
// boundary (PLUG-06, D-10): a real supervisor boot whose trusted
// directory is EMPTY and whose external directory holds a freshly built
// topos-plugin-mock instance still launches it — end to end, through the
// exact boot sequence NewSupervisor performs in production — and reports
// its tier as external via ProbeSources, the same live-reachability path
// GET /api/sources itself calls.
package supervisor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/pluginhost"
)

// buildSecondMockPluginDir builds a genuinely SECOND, independent copy of
// the repo's plugins/mock reference plugin into its own fresh temp
// directory — deliberately NOT sharing buildMockPluginDir's sync.Once
// cache (supervisor_test.go), because the shadow-collision test below
// needs two distinct directories that both really hold a
// "topos-plugin-mock" binary on disk, not one directory referenced
// twice.
func buildSecondMockPluginDir(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/davison/topos").Output()
	if err != nil {
		t.Fatalf("resolve module root: %v", err)
	}
	root := strings.TrimSpace(string(out))

	dir, err := os.MkdirTemp("", "topos-supervisor-externaltier-test-*")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}

	bin := filepath.Join(dir, "topos-plugin-mock")
	cmd := exec.Command("go", "build", "-o", bin, "./plugins/mock")
	cmd.Dir = root
	if buildOut, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build second mock plugin: %v\n%s", err, buildOut)
	}
	return dir
}

// TestExternalTier_BinaryPresentOnlyExternallyLaunchesAndReportsTier
// proves the tracer slice this whole plan exists to prove: a plugin
// binary that lives ONLY in the external directory (the trusted
// directory is genuinely empty — no shadow, no ambiguity) is discovered,
// launched, and reports tier "external" through the identical
// ProbeSources call GET /api/sources uses in production.
func TestExternalTier_BinaryPresentOnlyExternallyLaunchesAndReportsTier(t *testing.T) {
	externalDir := buildMockPluginDir(t)
	trustedDir := t.TempDir() // empty — nothing trusted configured at all
	idx := newTestIndex(t)
	ctx := context.Background()

	cfgStore := newTestConfigStore(t, `
[sources.demo]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[webspaces.demo]
keywords = ["demo"]
`)

	dirs := pluginhost.Dirs{Trusted: trustedDir, External: externalDir}
	sup, err := NewSupervisor(ctx, idx, cfgStore, dirs, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("NewSupervisor: %v", err)
	}
	t.Cleanup(sup.Shutdown)

	plugins := sup.Host().Plugins()
	if len(plugins) != 1 {
		t.Fatalf("expected exactly one launched instance, got %d: %+v", len(plugins), plugins)
	}
	if plugins[0].Name() != "demo" {
		t.Fatalf("expected the launched instance to be %q, got %q", "demo", plugins[0].Name())
	}
	if plugins[0].Tier() != pluginhost.TierExternal {
		t.Errorf("expected the launched instance's own Tier() to be %q, got %q", pluginhost.TierExternal, plugins[0].Tier())
	}

	healths := sup.ProbeSources(ctx)
	if len(healths) != 1 {
		t.Fatalf("expected exactly one probed health, got %d: %+v", len(healths), healths)
	}
	if healths[0].Name != "demo" {
		t.Fatalf("expected the probed health to be for %q, got %q", "demo", healths[0].Name)
	}
	if healths[0].Tier != pluginhost.TierExternal {
		t.Errorf("expected ProbeSources to report tier %q for an external-only binary, got %q", pluginhost.TierExternal, healths[0].Tier)
	}
	if !healths[0].Reachable {
		t.Errorf("expected the external-tier instance to be reachable, got: %+v", healths[0])
	}
}

// TestExternalTier_TrustedShadowsExternalOnNameCollision proves D-11 at
// the supervisor boundary: when the SAME binary name is present in BOTH
// directories, the launched instance resolves to the trusted copy and
// reports tier "trusted" — the shadow rule holds through the whole real
// boot sequence, not merely inside pluginhost's own unit tests.
func TestExternalTier_TrustedShadowsExternalOnNameCollision(t *testing.T) {
	trustedDir := buildMockPluginDir(t)
	// The external directory ALSO carries a real topos-plugin-mock build,
	// so a collision genuinely exists to shadow rather than merely
	// naming an absent file — buildMockPluginDir's own sync.Once cache
	// means calling it twice returns the identical directory, so this
	// deliberately builds a SECOND, independent copy under a fresh temp
	// dir instead, mirroring the trusted binary byte-for-byte in shape
	// (both built from ./plugins/mock) but living in a distinct external
	// directory.
	externalDir := buildSecondMockPluginDir(t)

	idx := newTestIndex(t)
	ctx := context.Background()

	cfgStore := newTestConfigStore(t, `
[sources.demo]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[webspaces.demo]
keywords = ["demo"]
`)

	dirs := pluginhost.Dirs{Trusted: trustedDir, External: externalDir}
	sup, err := NewSupervisor(ctx, idx, cfgStore, dirs, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("NewSupervisor: %v", err)
	}
	t.Cleanup(sup.Shutdown)

	plugins := sup.Host().Plugins()
	if len(plugins) != 1 {
		t.Fatalf("expected exactly one launched instance, got %d: %+v", len(plugins), plugins)
	}
	if plugins[0].Tier() != pluginhost.TierTrusted {
		t.Errorf("expected a name collision to resolve to %q, got %q", pluginhost.TierTrusted, plugins[0].Tier())
	}
}
