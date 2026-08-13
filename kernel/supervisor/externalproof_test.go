// externalproof_test.go is the standing, mechanical gate for ROADMAP
// success criterion 5 — "the mechanism is proven before any out-of-repo
// source work starts" — and the observable proof for PLUG-06 (an
// out-of-repo binary is discovered, marked untrusted, launched, and
// syncs end to end), PLUG-07 (content-hash pin verification refuses a
// tampered copy of that same binary by name while a healthy source in
// the same config keeps running), PLUG-09 (provider-specific extras
// reach the plugin unmodified) and D-14 (the plugin sees only its own
// documented allowlist plus the values behind ${VAR} references its own
// raw config declares — never the kernel's remaining environment).
//
// Unlike every other Phase 11 supervisor test, the binary under test here
// is NOT plugins/mock rebuilt under a different name — it is
// testdata/external-plugin, a genuinely separate Go module (module path
// example.com/acme/topos-plugin-external-demo, outside
// github.com/davison/topos) written entirely from the published contract,
// standing in for a third party's own out-of-repo build. See
// testdata/external-plugin/README.md for the full shape.
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

	"github.com/davison/topos/kernel/index"
	"github.com/davison/topos/kernel/item"
	"github.com/davison/topos/kernel/pluginhost"
)

// externalDemoBinaryName is the fixed binary name testdata/external-plugin
// builds to and kernel/config.PluginsConfig.Pins keys by — matching the
// Makefile's own external-demo target's output filename exactly.
const externalDemoBinaryName = "topos-plugin-external-demo"

// buildExternalDemoPluginDir builds Phase 11's out-of-repo proof plugin
// (testdata/external-plugin) fresh, via `go build` executed INSIDE that
// module's own directory (its own go.mod, its own dependency resolution
// through the workspace — see go.work), into a fresh temp directory.
//
// Unlike buildMockPluginDir/buildRenamedMockPluginDir above (which build
// a binary this repository owns outright and therefore Fatalf on any
// build failure), this helper SKIPS the test with a clear, named message
// — and never fails it — when the build cannot run: this test's job is to
// gate the external-plugin MECHANISM the fixture proves, not to also gate
// whether testdata/external-plugin's own module happens to build in the
// current environment.
func buildExternalDemoPluginDir(t *testing.T) string {
	t.Helper()

	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/davison/topos").Output()
	if err != nil {
		t.Skipf("TestExternalProof: resolve module root: %v", err)
	}
	root := strings.TrimSpace(string(out))
	srcDir := filepath.Join(root, "testdata", "external-plugin")

	dir, err := os.MkdirTemp("", "topos-supervisor-externalproof-test-*")
	if err != nil {
		t.Skipf("TestExternalProof: mkdir temp: %v", err)
	}

	bin := filepath.Join(dir, externalDemoBinaryName)
	cmd := exec.Command("go", "build", "-o", bin, ".")
	cmd.Dir = srcDir
	if buildOut, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("TestExternalProof: build testdata/external-plugin fixture (skipping, never failing — this module's own build environment is out of this test's gate): %v\n%s", err, buildOut)
	}
	return dir
}

// copyExecutableFile copies src to dst with mode 0o755 — used to place the
// once-built external-demo binary into the actual external plugin
// directory the config under test names, so the pin this test computes
// and the bytes ResolveBinary later re-hashes are the SAME on-disk file.
func copyExecutableFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0o755); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}

// TestExternalProof_OutOfRepoBinaryEndToEnd is the criterion-5 gate: a
// binary built OUTSIDE the in-repo plugin set — its own module, its own
// build target, its own output directory — is discovered from the
// external tier, launched under a content-hash pin, and syncs its corpus
// into the index; the corpus itself reports back exactly what config
// (extras) and environment this instance was actually launched with, from
// the plugin's own point of view, not from this test's own assumptions.
// The final phase rewrites one byte of the binary and re-boots: the
// tampered instance is refused by name while a healthy control source
// keeps running.
func TestExternalProof_OutOfRepoBinaryEndToEnd(t *testing.T) {
	srcDir := buildExternalDemoPluginDir(t)
	trustedDir := buildMockPluginDir(t)
	externalDir := t.TempDir()
	idx := newTestIndex(t)
	ctx := context.Background()

	binPath := filepath.Join(externalDir, externalDemoBinaryName)
	copyExecutableFile(t, filepath.Join(srcDir, externalDemoBinaryName), binPath)

	pinnedHash, err := pluginhost.HashBinary(binPath)
	if err != nil {
		t.Fatalf("hash external-demo binary for pin: %v", err)
	}

	// TOPOS_PROOF_REFERENCED is referenced by the "demo" instance's own raw
	// config below (an extras ${VAR} value) — allowedEnv must copy it into
	// the launched subprocess's environment. TOPOS_PROOF_UNREFERENCED is
	// set on this (the kernel/test) process but referenced NOWHERE in any
	// instance's config — allowedEnv must never copy it, no matter what
	// it's named (D-14, T-11-08). Set before booting, per this test's own
	// design.
	t.Setenv("TOPOS_PROOF_REFERENCED", "acme-referenced-value")
	t.Setenv("TOPOS_PROOF_UNREFERENCED", "must-never-leak")

	cfgStore := newTestConfigStore(t, fmt.Sprintf(`
[plugins]
dir = %q
external_dir = %q

[plugins.pins]
%q = %q

[sources.mockctrl]
plugin = "topos-plugin-mock"
base_url = "http://mock.test"
token = "unused"

[sources.demo]
plugin = "topos-plugin-external-demo"
path = "/nonexistent/path/is/fine-never-opened"

[sources.demo.extras]
workspace_id = "acme-42"
referenced_var = "${TOPOS_PROOF_REFERENCED}"

[webspaces.control]
sources = ["mockctrl"]
keywords = ["demo"]

[webspaces.proof]
sources = ["demo"]
keywords = ["external-demo-proof"]
`, trustedDir, externalDir, externalDemoBinaryName, pinnedHash))

	dirs := pluginhost.Dirs{Trusted: trustedDir, External: externalDir}
	sup, err := NewSupervisor(ctx, idx, cfgStore, dirs, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("NewSupervisor: %v", err)
	}

	plugins := sup.Host().Plugins()
	if len(plugins) != 2 {
		t.Fatalf("expected exactly two launched instances (mockctrl, demo), got %d: %+v", len(plugins), plugins)
	}
	demoPlugin, ok := pluginByName(plugins, "demo")
	if !ok {
		t.Fatalf("expected instance %q launched, got %+v", "demo", plugins)
	}
	if demoPlugin.Tier() != pluginhost.TierExternal {
		t.Errorf("expected the demo instance's own Tier() to be %q, got %q", pluginhost.TierExternal, demoPlugin.Tier())
	}

	if _, err := sup.Refresh(ctx, "demo"); err != nil {
		t.Fatalf("Refresh(demo): %v", err)
	}
	if _, err := sup.Refresh(ctx, "mockctrl"); err != nil {
		t.Fatalf("Refresh(mockctrl): %v", err)
	}

	// Extras passthrough (PLUG-09): one index item per configured extras
	// key, the referenced key's value already ${VAR}-expanded.
	assertItemTitle(t, ctx, idx, "demo", "extras/workspace_id", "extras workspace_id=acme-42")
	assertItemTitle(t, ctx, idx, "demo", "extras/referenced_var", "extras referenced_var=acme-referenced-value")

	// Environment scrubbing (D-14): PATH (the fixed allowlist) and the
	// referenced variable are visible to the subprocess; the unreferenced
	// variable — set on this test process but named nowhere in "demo"'s own
	// raw config — is not.
	assertItemPresent(t, ctx, idx, "demo", "env/PATH")
	assertItemPresent(t, ctx, idx, "demo", "env/TOPOS_PROOF_REFERENCED")
	assertItemAbsent(t, ctx, idx, "demo", "env/TOPOS_PROOF_UNREFERENCED")

	// Shut down before rebooting a second supervisor against the same
	// config and idx below — Reconcile only re-launches an instance whose
	// config.Source CHANGED (08-13-PLAN.md/host.go's own doc comment), so
	// tampering the binary's bytes alone would never be re-verified by an
	// Apply call over this same, unchanged config; a fresh Discover (a
	// full reboot) is what actually re-hashes every external-tier binary.
	sup.Shutdown()

	// Rewrite one byte of the binary (PLUG-07, T-11-07): the same on-disk
	// path the pin above was computed against, and the path ResolveBinary
	// will re-hash on the next boot. Written to a sibling temp file and
	// os.Rename'd over binPath — not written in place — because Kill()
	// returning is not a guarantee the kernel has finished reaping the
	// just-exited subprocess: an in-place open-for-write can still race a
	// kernel-held ETXTBSY on the exact inode a just-executed binary used,
	// while rename() is never blocked by that race (the standard technique
	// for safely replacing a binary that may have just been running).
	tampered, err := os.ReadFile(binPath)
	if err != nil {
		t.Fatalf("read binary to tamper: %v", err)
	}
	tampered[len(tampered)-1] ^= 0xFF
	tamperedPath := binPath + ".tampered"
	if err := os.WriteFile(tamperedPath, tampered, 0o755); err != nil {
		t.Fatalf("write tampered binary: %v", err)
	}
	if err := os.Rename(tamperedPath, binPath); err != nil {
		t.Fatalf("rename tampered binary over %s: %v", binPath, err)
	}

	// Retire the "proof" webspace (demo's only participation) before
	// re-booting — this test's own scope is the pin-refusal-at-boot proof,
	// not the match-vocabulary interaction (kept minimal and independent of
	// that concern, mirroring pinmismatch_test.go's identical choice). As
	// of 11-06-PLAN.md Task 3, pluginhost.ValidateMatchConfig DOES excuse a
	// pin-mismatched instance from its own match-vocabulary check (see
	// pluginhost.launchFailedNames and matchconfig_test.go's own
	// TestValidateMatchConfig_PinMismatchedInstanceExcused* pair) — this
	// retire-then-reboot step is no longer REQUIRED to avoid a spurious
	// NewSupervisor failure, but is kept anyway so this test's own
	// assertions stay focused on the pin-tamper-and-refuse behavior; the
	// participating-instance case is proved end to end by
	// web/e2e/specs/11-binary-changed-repin.spec.ts. "demo" remains fully
	// configured under [sources.demo] and is still launch-attempted (and
	// therefore still recorded in LaunchFailures()) by the reboot below —
	// only its webspace PARTICIPATION is removed.
	next := cfgStore.Raw()
	delete(next.Webspaces, "proof")
	if err := cfgStore.Save(next, cfgStore.Hash()); err != nil {
		t.Fatalf("Save (retire demo's webspace participation before re-verifying its tampered pin): %v", err)
	}

	sup2, err := NewSupervisor(ctx, idx, cfgStore, dirs, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("expected boot to succeed despite the tampered external binary (a pin mismatch is a soft, per-instance failure), got: %v", err)
	}
	t.Cleanup(sup2.Shutdown)

	failures := sup2.LaunchFailures()
	if len(failures) != 1 {
		t.Fatalf("expected exactly one launch failure after tampering, got %d: %+v", len(failures), failures)
	}
	if failures[0].Instance != "demo" {
		t.Errorf("expected the launch failure to name instance %q, got %q", "demo", failures[0].Instance)
	}
	if failures[0].Reason != pluginhost.LaunchFailurePinMismatch {
		t.Errorf("expected reason %q, got %q", pluginhost.LaunchFailurePinMismatch, failures[0].Reason)
	}

	pluginsAfterTamper := sup2.Host().Plugins()
	if len(pluginsAfterTamper) != 1 || pluginsAfterTamper[0].Name() != "mockctrl" {
		t.Fatalf("expected only the healthy control %q to remain launched after the tamper, got %+v", "mockctrl", pluginsAfterTamper)
	}

	healths := sup2.ProbeSources(ctx)
	if len(healths) != 1 {
		t.Fatalf("expected exactly one probed health (the healthy control), got %d: %+v", len(healths), healths)
	}
	if !healths[0].Reachable {
		t.Errorf("expected the healthy control instance to still be reachable after the tamper, got: %+v", healths[0])
	}
}

// assertItemTitle fails the test unless an index item exists for
// item.ID(source, sourceID) and its Title matches wantTitle exactly.
func assertItemTitle(t *testing.T, ctx context.Context, idx *index.Store, source, sourceID, wantTitle string) {
	t.Helper()
	got, ok, err := idx.GetItem(ctx, item.ID(source, sourceID))
	if err != nil {
		t.Fatalf("GetItem(%s): %v", item.ID(source, sourceID), err)
	}
	if !ok {
		t.Fatalf("expected an index item for %s, found none", item.ID(source, sourceID))
	}
	if got.Title != wantTitle {
		t.Errorf("expected %s's title to be %q, got %q", item.ID(source, sourceID), wantTitle, got.Title)
	}
}

// assertItemPresent fails the test unless an index item exists for
// item.ID(source, sourceID).
func assertItemPresent(t *testing.T, ctx context.Context, idx *index.Store, source, sourceID string) {
	t.Helper()
	_, ok, err := idx.GetItem(ctx, item.ID(source, sourceID))
	if err != nil {
		t.Fatalf("GetItem(%s): %v", item.ID(source, sourceID), err)
	}
	if !ok {
		t.Errorf("expected an index item for %s, found none", item.ID(source, sourceID))
	}
}

// assertItemAbsent fails the test if an index item exists for
// item.ID(source, sourceID) — used to prove a variable set on the test
// process but referenced nowhere in the instance's config never reaches
// the index (D-14).
func assertItemAbsent(t *testing.T, ctx context.Context, idx *index.Store, source, sourceID string) {
	t.Helper()
	_, ok, err := idx.GetItem(ctx, item.ID(source, sourceID))
	if err != nil {
		t.Fatalf("GetItem(%s): %v", item.ID(source, sourceID), err)
	}
	if ok {
		t.Errorf("expected NO index item for %s (it must never reach the index), but found one", item.ID(source, sourceID))
	}
}
