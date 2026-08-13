package pluginhost

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
)

// buildMockPluginDir builds the repo's plugins/mock reference plugin
// fresh, once per test binary run (via sync.Once — go build is not free,
// and every test in this file wants the identical binary), and returns
// the directory it was written to — usable as a pluginsDir for any test
// that needs Host.Reconcile to actually launch or kill a REAL subprocess.
// This is required, not merely convenient: Plugin.Kill() calls
// p.client.Kill() unconditionally, which panics on a hand-built *Plugin
// whose client field was never set — so any test exercising Reconcile's
// kill path (a removed or changed instance) needs a genuinely launched
// plugin, never a bare struct literal.
var (
	mockPluginDirOnce sync.Once
	mockPluginDir     string
	mockPluginDirErr  error
)

func buildMockPluginDir(t *testing.T) string {
	t.Helper()
	mockPluginDirOnce.Do(func() {
		out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/davison/topos").Output()
		if err != nil {
			mockPluginDirErr = fmt.Errorf("resolve module root: %w", err)
			return
		}
		root := strings.TrimSpace(string(out))

		dir, err := os.MkdirTemp("", "topos-pluginhost-reconcile-test-*")
		if err != nil {
			mockPluginDirErr = err
			return
		}

		bin := filepath.Join(dir, "topos-plugin-mock")
		cmd := exec.Command("go", "build", "-o", bin, "./plugins/mock")
		cmd.Dir = root
		if buildOut, err := cmd.CombinedOutput(); err != nil {
			mockPluginDirErr = fmt.Errorf("build mock plugin: %w\n%s", err, buildOut)
			return
		}

		mockPluginDir = dir
	})
	if mockPluginDirErr != nil {
		t.Fatalf("build mock plugin fixture: %v", mockPluginDirErr)
	}
	return mockPluginDir
}

// TestReconcile_UnchangedSourceMapKeepsSamePluginPointers proves an apply
// touching nothing leaves a launched instance's subprocess completely
// undisturbed — the SAME *Plugin pointer survives the call, not merely an
// equal-looking replacement, so an unrelated source's subprocess is never
// restarted by a save that only touches a different source.
func TestReconcile_UnchangedSourceMapKeepsSamePluginPointers(t *testing.T) {
	dir := buildMockPluginDir(t)
	sources := map[string]config.Source{"demo": {Plugin: "topos-plugin-mock"}}

	h, err := Discover(context.Background(), Dirs{Trusted: dir}, nil, sources, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	defer h.Shutdown()

	before := h.Plugins()[0]

	if err := h.Reconcile(context.Background(), nil, sources, hclog.NewNullLogger()); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	if len(h.Plugins()) != 1 || h.Plugins()[0] != before {
		t.Fatalf("expected the identical *Plugin pointer to survive an unchanged reconcile, got %+v", h.Plugins())
	}
}

// TestReconcile_RemovedInstanceIsKilledAndDropped proves an instance
// absent from the new source map is killed and no longer reported by
// Plugins().
func TestReconcile_RemovedInstanceIsKilledAndDropped(t *testing.T) {
	dir := buildMockPluginDir(t)
	h, err := Discover(context.Background(), Dirs{Trusted: dir}, nil, map[string]config.Source{
		"demo": {Plugin: "topos-plugin-mock"},
	}, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}

	if err := h.Reconcile(context.Background(), nil, map[string]config.Source{}, hclog.NewNullLogger()); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	if len(h.Plugins()) != 0 {
		t.Fatalf("expected zero plugins after removing the only configured instance, got %d: %+v", len(h.Plugins()), h.Plugins())
	}
}

// TestReconcile_LaunchFailureLeavesPreviouslyRunningSetIntact proves
// T-07-11: a launch failure for one added instance names that instance in
// its error and leaves every already-running instance's subprocess still
// launched and unrestarted — a partial apply must never look successful.
func TestReconcile_LaunchFailureLeavesPreviouslyRunningSetIntact(t *testing.T) {
	dir := buildMockPluginDir(t)
	h, err := Discover(context.Background(), Dirs{Trusted: dir}, nil, map[string]config.Source{
		"demo": {Plugin: "topos-plugin-mock"},
	}, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	defer h.Shutdown()

	before := h.Plugins()[0]

	err = h.Reconcile(context.Background(), nil, map[string]config.Source{
		"demo":  {Plugin: "topos-plugin-mock"},
		"bogus": {Plugin: "topos-plugin-does-not-exist"},
	}, hclog.NewNullLogger())
	if err == nil {
		t.Fatal("expected an error for the missing plugin binary")
	}
	if !strings.Contains(err.Error(), "bogus") {
		t.Errorf("expected the error to name the offending instance %q, got: %v", "bogus", err)
	}

	if len(h.Plugins()) != 1 || h.Plugins()[0] != before {
		t.Fatalf("expected the previously running instance to survive a failed reconcile completely untouched, got %+v", h.Plugins())
	}
}

// TestReconcile_ChangedConnectionConfigRelaunches proves an instance whose
// config.Source differs from what it was launched with gets a NEW *Plugin
// (the old subprocess killed, a new one launched) — Reconcile does not
// treat "same name" as "unchanged" when the connection config itself
// changed.
func TestReconcile_ChangedConnectionConfigRelaunches(t *testing.T) {
	dir := buildMockPluginDir(t)
	h, err := Discover(context.Background(), Dirs{Trusted: dir}, nil, map[string]config.Source{
		"demo": {Plugin: "topos-plugin-mock", DisplayName: "before"},
	}, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	defer h.Shutdown()

	before := h.Plugins()[0]

	if err := h.Reconcile(context.Background(), nil, map[string]config.Source{
		"demo": {Plugin: "topos-plugin-mock", DisplayName: "after"},
	}, hclog.NewNullLogger()); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	if len(h.Plugins()) != 1 {
		t.Fatalf("expected exactly one plugin after reconciling a single changed instance, got %d", len(h.Plugins()))
	}
	if h.Plugins()[0] == before {
		t.Fatal("expected a NEW *Plugin pointer after a connection-config change, got the same one")
	}
	if got := h.Plugins()[0].DisplayName(); got != "after" {
		t.Errorf("expected the relaunched instance to reflect the new display name, got %q", got)
	}
}
