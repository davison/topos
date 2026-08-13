// pinmismatch_test.go is Phase 11's own soft-failure proof at the
// supervisor boundary (T-11-07/T-11-09, D-03): a pin-mismatched
// external-tier source must never take the whole kernel boot or a whole
// config save down with it — every OTHER configured source still boots,
// syncs, and applies normally, and the mismatched instance's refusal is
// retrievable by name from Supervisor.LaunchFailures(). A GENUINELY missing
// binary (a different failure class entirely) must keep today's hard-fail
// behavior completely unchanged (RESEARCH A3).
package supervisor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/pluginhost"
)

// buildRenamedMockPluginDir builds the repo's plugins/mock reference plugin
// fresh (the identical source buildMockPluginDir builds, from
// ./plugins/mock) but under an ARBITRARY topos-plugin- prefixed output
// name, into its own fresh temp directory — so this file's tests can give
// two source instances two DISTINCT binary names (one resolving trusted,
// one resolving external) without a name collision invoking D-11's shadow
// rule, which would otherwise make both instances resolve to the same
// tier.
func buildRenamedMockPluginDir(t *testing.T, binaryName string) string {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/davison/topos").Output()
	if err != nil {
		t.Fatalf("resolve module root: %v", err)
	}
	root := strings.TrimSpace(string(out))

	dir, err := os.MkdirTemp("", "topos-supervisor-pinmismatch-test-*")
	if err != nil {
		t.Fatalf("mkdir temp: %v", err)
	}

	bin := filepath.Join(dir, binaryName)
	cmd := exec.Command("go", "build", "-o", bin, "./plugins/mock")
	cmd.Dir = root
	if buildOut, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build renamed mock plugin %s: %v\n%s", binaryName, err, buildOut)
	}
	return dir
}

// TestPinMismatch_BootSucceedsHealthySourceSyncsFailureRecorded is this
// file's headline proof: a config carrying one healthy trusted-tier source
// and one pin-mismatched external-tier source boots successfully
// (NewSupervisor returns no error), the healthy source syncs normally, and
// the mismatched instance is recorded — by name — in LaunchFailures()
// rather than appearing as a launched plugin. A later Apply that edits only
// the healthy source still succeeds (returns nil), and the failure entry
// for the untouched, still-mismatched instance survives that apply
// unchanged — a pin mismatch is a standing condition, not a one-shot event
// that disappears on the next unrelated save.
func TestPinMismatch_BootSucceedsHealthySourceSyncsFailureRecorded(t *testing.T) {
	trustedDir := buildMockPluginDir(t)
	externalDir := buildRenamedMockPluginDir(t, "topos-plugin-mockext")
	idx := newTestIndex(t)
	ctx := context.Background()

	// A syntactically well-formed but WRONG 64-character hex pin for the
	// external binary — guaranteed not to equal its real SHA-256 (an
	// all-"f" digest colliding with a real build's hash is not a
	// practically reachable event).
	wrongPin := strings.Repeat("f", 64)

	// The "demo" webspace's sources allowlist deliberately names only
	// "healthy" — this test's own scope is the boot/Apply soft-failure
	// proof, not the match-vocabulary interaction. As of 11-06-PLAN.md
	// Task 3, pluginhost.ValidateMatchConfig DOES excuse a pin-mismatched
	// instance from its own match-vocabulary check (a third participant
	// class alongside "launched" and "suspended" — see
	// pluginhost.launchFailedNames and matchconfig_test.go's own
	// TestValidateMatchConfig_PinMismatchedInstanceExcused* pair), closing
	// the gap this comment previously flagged as unresolved. "bad-pin"
	// stays out of "demo"'s participation here purely to keep THIS test
	// focused on boot/Apply semantics rather than match-vocabulary
	// interaction — the participating-instance case (a pin-mismatched
	// instance named in a real webspace's match config, through a real
	// browser session) is proved end to end by
	// web/e2e/specs/11-binary-changed-repin.spec.ts.
	cfgStore := newTestConfigStore(t, fmt.Sprintf(`
[plugins.pins]
"topos-plugin-mockext" = %q

[sources.healthy]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[sources.bad-pin]
plugin = "topos-plugin-mockext"
base_url = "http://mock.test"
token = "unused"

[webspaces.demo]
keywords = ["demo"]
sources = ["healthy"]
`, wrongPin))

	dirs := pluginhost.Dirs{Trusted: trustedDir, External: externalDir}
	sup, err := NewSupervisor(ctx, idx, cfgStore, dirs, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("expected boot to succeed despite the pin-mismatched source, got: %v", err)
	}
	t.Cleanup(sup.Shutdown)

	plugins := sup.Host().Plugins()
	if len(plugins) != 1 {
		t.Fatalf("expected exactly one launched instance (the healthy one), got %d: %+v", len(plugins), plugins)
	}
	if plugins[0].Name() != "healthy" {
		t.Fatalf("expected the one launched instance to be %q, got %q", "healthy", plugins[0].Name())
	}

	failures := sup.LaunchFailures()
	if len(failures) != 1 {
		t.Fatalf("expected exactly one launch failure, got %d: %+v", len(failures), failures)
	}
	if failures[0].Instance != "bad-pin" {
		t.Errorf("expected the launch failure to name instance %q, got %q", "bad-pin", failures[0].Instance)
	}
	if failures[0].Reason != pluginhost.LaunchFailurePinMismatch {
		t.Errorf("expected reason %q, got %q", pluginhost.LaunchFailurePinMismatch, failures[0].Reason)
	}

	healths := sup.ProbeSources(ctx)
	if len(healths) != 1 {
		t.Fatalf("expected exactly one probed health (the healthy instance), got %d: %+v", len(healths), healths)
	}
	if !healths[0].Reachable {
		t.Errorf("expected the healthy instance to be reachable, got: %+v", healths[0])
	}

	// Apply an edit touching ONLY the healthy source — the save must not
	// be rejected on account of the unrelated, still-mismatched instance.
	next := cfgStore.Raw()
	healthySrc := next.Sources["healthy"]
	healthySrc.DisplayName = "Healthy (renamed)"
	next.Sources["healthy"] = healthySrc

	if err := cfgStore.Save(next, cfgStore.Hash()); err != nil {
		t.Fatalf("Save (editing only the healthy source): %v", err)
	}
	if err := sup.Apply(ctx); err != nil {
		t.Fatalf("expected Apply to succeed despite the standing pin mismatch, got: %v", err)
	}

	failuresAfterApply := sup.LaunchFailures()
	if len(failuresAfterApply) != 1 {
		t.Fatalf("expected the pin-mismatch failure to survive an unrelated apply, got %d entries: %+v", len(failuresAfterApply), failuresAfterApply)
	}
	if failuresAfterApply[0].Instance != "bad-pin" {
		t.Errorf("expected the surviving failure to still name %q, got %q", "bad-pin", failuresAfterApply[0].Instance)
	}
}

// TestPinMismatch_MissingBinaryStillHardFailsBoot proves RESEARCH A3's
// scoping decision at the supervisor boundary: a source naming a binary
// that exists in NEITHER configured directory (a genuinely different
// failure class from a pin mismatch) still fails NewSupervisor outright —
// this phase narrows the soft-failure treatment to pin mismatches only and
// must not silently widen it to every launch-failure class.
func TestPinMismatch_MissingBinaryStillHardFailsBoot(t *testing.T) {
	trustedDir := buildMockPluginDir(t)
	idx := newTestIndex(t)
	ctx := context.Background()

	cfgStore := newTestConfigStore(t, `
[sources.demo]
plugin = "topos-plugin-does-not-exist"
base_url = "http://mock.test"
token = "unused"
`)

	_, err := NewSupervisor(ctx, idx, cfgStore, pluginhost.Dirs{Trusted: trustedDir}, hclog.NewNullLogger())
	if err == nil {
		t.Fatal("expected NewSupervisor to fail outright for a genuinely missing plugin binary")
	}
}
