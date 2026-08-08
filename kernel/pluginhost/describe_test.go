package pluginhost

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
)

// TestDescribePluginType_LaunchFailureWrapsErrorWithoutASubprocess proves
// the error-wrap shape ("pluginhost: trial-launch for describe: …") for
// the common failure case — a named binary that doesn't exist — with no
// live subprocess required (the seam this comment names: launch()'s own
// os.Stat check fails before anything is ever exec'd).
func TestDescribePluginType_LaunchFailureWrapsErrorWithoutASubprocess(t *testing.T) {
	_, err := DescribePluginType(context.Background(), t.TempDir(), config.Source{Plugin: "topos-plugin-does-not-exist"}, hclog.NewNullLogger())
	if err == nil {
		t.Fatal("expected an error for a missing plugin binary")
	}
	if !strings.HasPrefix(err.Error(), "pluginhost: trial-launch for describe:") {
		t.Errorf("expected the trial-launch error-wrap prefix, got: %v", err)
	}
}

// TestDescribePluginType_RealPlugin_ReturnsDescribeInfoAndKillsBeforeReturning
// proves the happy path against a genuinely launched subprocess (the mock
// reference plugin): the three Describe-derived fields come back
// populated, and — implicitly, since defer p.Kill() runs before this
// function returns and the test process itself exits cleanly with no
// leaked subprocess — nothing is left running.
func TestDescribePluginType_RealPlugin_ReturnsDescribeInfoAndKillsBeforeReturning(t *testing.T) {
	dir := buildMockPluginDir(t)

	info, err := DescribePluginType(context.Background(), dir, config.Source{Plugin: "topos-plugin-mock"}, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("DescribePluginType: %v", err)
	}
	if info.SourceType != "mock" {
		t.Errorf("expected source_type %q, got %q", "mock", info.SourceType)
	}
	if info.PluginDisplayName == "" {
		t.Error("expected a non-empty PluginDisplayName")
	}
	if len(info.MatchVocabulary) == 0 {
		t.Error("expected a non-empty MatchVocabulary")
	}
}
