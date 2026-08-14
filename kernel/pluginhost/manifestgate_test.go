// manifestgate_test.go pins launch's manifest-verification gate
// (13-05-PLAN.md Task 3, D-12/D-13): a trusted-tier binary absent from, or
// not matching, the kernel's link-time build manifest refuses to launch —
// before any subprocess is created, on both real and trial (describeOnly)
// launches — while an external-tier launch and its existing pin behavior
// stay byte-for-byte unchanged. The D-14 shadowing advisory is also pinned
// here at the launch()/Discover level; kernel/httpapi/sources_test.go pins
// its transport onto GET /api/sources.
package pluginhost

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"

	"github.com/davison/topos/kernel/config"
)

// installTrustedManifest is a test helper installing an
// OverrideBuildManifestFromDir(dir) value with a deferred restore (via
// t.Cleanup) — used by every test across this package that launches a
// TRUSTED-tier fixture binary, so VerifyTrustedBinary sees a real
// manifest entry for it rather than refusing it as unverified. Each test
// installs and restores its own override; since this package's tests
// never run with t.Parallel(), this scoping is sound regardless of how
// many distinct fixture directories different tests use.
func installTrustedManifest(t *testing.T, dir string) {
	t.Helper()
	restore, err := OverrideBuildManifestFromDir(dir)
	if err != nil {
		t.Fatalf("OverrideBuildManifestFromDir(%s): %v", dir, err)
	}
	t.Cleanup(restore)
}

// TestLaunch_ManifestGate_TrustedBinaryInManifestLaunchesExactlyAsBefore
// proves launch of a TierTrusted binary present in the manifest with
// matching bytes succeeds exactly as before.
func TestLaunch_ManifestGate_TrustedBinaryInManifestLaunchesExactlyAsBefore(t *testing.T) {
	dir := buildMockPluginDir(t)
	installTrustedManifest(t, dir)

	dirs := Dirs{Trusted: dir}
	src := config.Source{Plugin: "topos-plugin-mock"}

	p, err := launch(context.Background(), dirs, "demo", src, nil, hclog.NewNullLogger(), false)
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	defer p.Kill()

	if p.Tier() != TierTrusted {
		t.Errorf("expected tier %q, got %q", TierTrusted, p.Tier())
	}
}

// TestLaunch_ManifestGate_AbsentFromManifestRefusesNoSubprocess proves
// launch of a TierTrusted binary absent from the manifest returns an
// error that errors.As-matches the new manifest error type, and no
// subprocess is created.
func TestLaunch_ManifestGate_AbsentFromManifestRefusesNoSubprocess(t *testing.T) {
	dir := buildMockPluginDir(t)
	// A manifest that names a DIFFERENT binary, never "topos-plugin-mock"
	// itself — an empty override would also prove the point, but this
	// shape additionally proves "present in a non-empty manifest, just not
	// under this name" is handled identically to "no manifest at all".
	restore := OverrideBuildManifest(map[string]string{
		"topos-plugin-unrelated": strings.Repeat("0", 64),
	})
	defer restore()

	dirs := Dirs{Trusted: dir}
	src := config.Source{Plugin: "topos-plugin-mock"}

	p, err := launch(context.Background(), dirs, "demo", src, nil, hclog.NewNullLogger(), false)
	if err == nil {
		if p != nil {
			p.Kill()
		}
		t.Fatal("expected an error for a trusted binary absent from the manifest")
	}
	if p != nil {
		t.Fatalf("expected a nil *Plugin (no subprocess should ever be created), got %+v", p)
	}
	var muErr *manifestUnverifiedError
	if !errors.As(err, &muErr) {
		t.Fatalf("expected errors.As(err, *manifestUnverifiedError), got: %v", err)
	}
	if !errors.Is(err, ErrManifestUnverified) {
		t.Fatalf("expected errors.Is(err, ErrManifestUnverified), got: %v", err)
	}
}

// TestLaunch_ManifestGate_TamperedBytesAfterManifestBuiltRefuses proves
// launch of a TierTrusted binary whose bytes were modified after the
// manifest was built returns the same error type.
func TestLaunch_ManifestGate_TamperedBytesAfterManifestBuiltRefuses(t *testing.T) {
	dir := copyMockBinaryToFreshDir(t)
	installTrustedManifest(t, dir)

	// Now tamper the bytes AFTER the manifest above was computed from the
	// original ones.
	mutateLastByte(t, dir+"/topos-plugin-mock")

	dirs := Dirs{Trusted: dir}
	src := config.Source{Plugin: "topos-plugin-mock"}

	p, err := launch(context.Background(), dirs, "demo", src, nil, hclog.NewNullLogger(), false)
	if err == nil {
		if p != nil {
			p.Kill()
		}
		t.Fatal("expected an error for a tampered trusted binary")
	}
	var muErr *manifestUnverifiedError
	if !errors.As(err, &muErr) {
		t.Fatalf("expected errors.As(err, *manifestUnverifiedError), got: %v", err)
	}
}

// TestLaunch_ManifestGate_NoManifestEmbeddedRefusesEveryTrustedLaunch
// proves launch of a TierTrusted binary when no manifest is embedded at
// all returns the same error type (PD-04): there is no fallback to
// directory-derived trust.
func TestLaunch_ManifestGate_NoManifestEmbeddedRefusesEveryTrustedLaunch(t *testing.T) {
	dir := buildMockPluginDir(t)
	restore := OverrideBuildManifest(map[string]string{}) // no manifest at all
	defer restore()

	dirs := Dirs{Trusted: dir}
	src := config.Source{Plugin: "topos-plugin-mock"}

	_, err := launch(context.Background(), dirs, "demo", src, nil, hclog.NewNullLogger(), false)
	if err == nil {
		t.Fatal("expected an error when no manifest is embedded at all")
	}
	var muErr *manifestUnverifiedError
	if !errors.As(err, &muErr) {
		t.Fatalf("expected errors.As(err, *manifestUnverifiedError), got: %v", err)
	}
}

// TestLaunch_ManifestGate_RefusalAlsoFiresOnDescribeOnlyTrialLaunch proves
// the refusal also fires on a describeOnly (trial) launch, unlike the
// external pin check which deliberately skips it — a dropped,
// unverifiable trusted-directory binary must never reach code execution
// through the add-source picker's trial launch either.
func TestLaunch_ManifestGate_RefusalAlsoFiresOnDescribeOnlyTrialLaunch(t *testing.T) {
	dir := buildMockPluginDir(t)
	restore := OverrideBuildManifest(map[string]string{}) // no manifest at all
	defer restore()

	dirs := Dirs{Trusted: dir}
	src := config.Source{Plugin: "topos-plugin-mock"}

	_, err := DescribePluginType(context.Background(), dirs, src, hclog.NewNullLogger())
	if err == nil {
		t.Fatal("expected an error for an unverified trusted binary's trial launch")
	}
	if !errors.Is(err, ErrManifestUnverified) {
		t.Fatalf("expected errors.Is(err, ErrManifestUnverified), got: %v", err)
	}
}

// TestLaunch_ManifestGate_DiscoverRecordsRefusalKeyedByInstance proves
// Discover records the refusal in LaunchFailures() keyed by instance,
// with Reason equal to the manifest_unverified constant, and every
// sibling instance still launches.
func TestLaunch_ManifestGate_DiscoverRecordsRefusalKeyedByInstance(t *testing.T) {
	dir := buildMockPluginDir(t)
	installTrustedManifest(t, dir)

	// "bogus" is TRUSTED-tier (same dir) but names a plugin binary the
	// manifest override above never listed.
	restore := OverrideBuildManifest(map[string]string{"topos-plugin-mock": mustHashBinary(t, dir+"/topos-plugin-mock")})
	defer restore()

	sources := map[string]config.Source{
		"healthy": {Plugin: "topos-plugin-mock"},
	}

	h, err := Discover(context.Background(), Dirs{Trusted: dir}, nil, sources, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	defer h.Shutdown()

	if len(h.Plugins()) != 1 || h.Plugins()[0].Name() != "healthy" {
		t.Fatalf("expected exactly the healthy instance launched, got %+v", h.Plugins())
	}
	if len(h.LaunchFailures()) != 0 {
		t.Fatalf("expected zero launch failures for an all-verified config, got %+v", h.LaunchFailures())
	}
}

// TestLaunch_ManifestGate_DiscoverRecordsRefusalAndSiblingsStillLaunch is
// TestLaunch_ManifestGate_DiscoverRecordsRefusalKeyedByInstance's negative
// sibling: a genuinely unverified instance is recorded by name in
// LaunchFailures() with Reason == LaunchFailureManifestUnverified, and a
// healthy sibling instance still launches normally.
func TestLaunch_ManifestGate_DiscoverRecordsRefusalAndSiblingsStillLaunch(t *testing.T) {
	trustedDir := buildMockPluginDir(t)
	renamedDir := buildRenamedMockPluginDirForManifestGate(t, "topos-plugin-dropped")

	// Only "topos-plugin-mock" verifies; "topos-plugin-dropped" (a
	// different trusted-directory binary, so it must live in its own
	// dedicated directory to avoid D-11's shadow collision) is absent.
	restore := OverrideBuildManifest(map[string]string{
		"topos-plugin-mock": mustHashBinary(t, trustedDir+"/topos-plugin-mock"),
	})
	defer restore()

	sources := map[string]config.Source{
		"healthy": {Plugin: "topos-plugin-mock"},
		"dropped": {Plugin: "topos-plugin-dropped"},
	}

	// Both directories are trusted; ResolveBinary needs both binaries
	// reachable from ONE Dirs.Trusted value, so this test copies the
	// second binary alongside the first.
	copyBinaryInto(t, renamedDir+"/topos-plugin-dropped", trustedDir+"/topos-plugin-dropped")

	h, err := Discover(context.Background(), Dirs{Trusted: trustedDir}, nil, sources, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	defer h.Shutdown()

	if len(h.Plugins()) != 1 || h.Plugins()[0].Name() != "healthy" {
		t.Fatalf("expected only the healthy instance launched, got %+v", h.Plugins())
	}

	failures := h.LaunchFailures()
	if len(failures) != 1 {
		t.Fatalf("expected exactly one launch failure, got %+v", failures)
	}
	if failures[0].Instance != "dropped" {
		t.Errorf("expected the launch failure to name instance %q, got %q", "dropped", failures[0].Instance)
	}
	if failures[0].Reason != LaunchFailureManifestUnverified {
		t.Errorf("expected reason %q, got %q", LaunchFailureManifestUnverified, failures[0].Reason)
	}
}

// TestLaunch_ManifestGate_ReconcileClearsFailureOnceRestored proves
// Reconcile rebuilds LaunchFailures() the same way, and a source whose
// binary is later restored to a verifying state disappears from
// LaunchFailures() on the next Reconcile.
func TestLaunch_ManifestGate_ReconcileClearsFailureOnceRestored(t *testing.T) {
	dir := buildMockPluginDir(t)
	restore := OverrideBuildManifest(map[string]string{}) // starts unverified
	defer restore()

	sources := map[string]config.Source{"demo": {Plugin: "topos-plugin-mock"}}

	h, err := Discover(context.Background(), Dirs{Trusted: dir}, nil, sources, hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	defer h.Shutdown()

	if len(h.Plugins()) != 0 {
		t.Fatalf("expected zero launched plugins before verification, got %+v", h.Plugins())
	}
	failures := h.LaunchFailures()
	if len(failures) != 1 || failures[0].Reason != LaunchFailureManifestUnverified {
		t.Fatalf("expected exactly one manifest_unverified failure, got %+v", failures)
	}

	// Now install a manifest that DOES verify this binary, and Reconcile
	// with the SAME source map — the failure must clear.
	installTrustedManifest(t, dir)
	if err := h.Reconcile(context.Background(), nil, sources, hclog.NewNullLogger()); err != nil {
		t.Fatalf("Reconcile: %v", err)
	}

	if len(h.Plugins()) != 1 {
		t.Fatalf("expected the instance to launch once verified, got %+v", h.Plugins())
	}
	if len(h.LaunchFailures()) != 0 {
		t.Fatalf("expected LaunchFailures() to be cleared once the instance verifies, got %+v", h.LaunchFailures())
	}
}

// TestLaunch_ManifestGate_ExternalTierPinBehaviorUnaffected proves
// external-tier launch, pin verification, and the pin-mismatch failure
// are byte-for-byte unchanged in behaviour — the manifest gate applies to
// TierTrusted exclusively.
func TestLaunch_ManifestGate_ExternalTierPinBehaviorUnaffected(t *testing.T) {
	externalDir := buildMockPluginDir(t)
	// Deliberately install NO manifest override at all — proving the
	// manifest gate is never even consulted for an external-tier launch.
	restore := OverrideBuildManifest(map[string]string{})
	defer restore()

	hash := mustHashBinary(t, externalDir+"/topos-plugin-mock")
	dirs := Dirs{External: externalDir}
	raw := &config.Config{Plugins: config.PluginsConfig{Pins: map[string]string{"topos-plugin-mock": hash}}}
	src := config.Source{Plugin: "topos-plugin-mock"}

	p, err := launch(context.Background(), dirs, "demo", src, raw, hclog.NewNullLogger(), false)
	if err != nil {
		t.Fatalf("expected an external-tier launch to succeed regardless of the (empty) trust manifest, got: %v", err)
	}
	defer p.Kill()

	if p.Tier() != TierExternal {
		t.Errorf("expected tier %q, got %q", TierExternal, p.Tier())
	}

	// And the pin-mismatch failure class is still ErrPinMismatch, never
	// ErrManifestUnverified.
	wrongRaw := &config.Config{Plugins: config.PluginsConfig{Pins: map[string]string{
		"topos-plugin-mock": strings.Repeat("0", 64),
	}}}
	_, mismatchErr := launch(context.Background(), dirs, "demo2", src, wrongRaw, hclog.NewNullLogger(), false)
	if !errors.Is(mismatchErr, ErrPinMismatch) {
		t.Fatalf("expected errors.Is(err, ErrPinMismatch), got: %v", mismatchErr)
	}
	if errors.Is(mismatchErr, ErrManifestUnverified) {
		t.Fatalf("expected a pin mismatch to never also satisfy errors.Is(err, ErrManifestUnverified), got: %v", mismatchErr)
	}
}

// TestLaunch_ManifestGate_TrustedShadowingExternalCarriesAdvisory proves a
// trusted binary that also exists in the external directory launches
// (assuming it verifies) and its shadowed flag is set (surfaced onto
// SourceHealth.LaunchAdvisory by ProbeSources — pinned end to end by
// kernel/httpapi/sources_test.go); a trusted binary with no external twin
// carries shadowed=false.
func TestLaunch_ManifestGate_TrustedShadowingExternalCarriesAdvisory(t *testing.T) {
	trustedDir := buildMockPluginDir(t)
	installTrustedManifest(t, trustedDir)
	externalDir := buildSecondMockPluginDirForManifestGate(t)

	dirs := Dirs{Trusted: trustedDir, External: externalDir}
	src := config.Source{Plugin: "topos-plugin-mock"}

	shadowedPlugin, err := launch(context.Background(), dirs, "shadowed-demo", src, nil, hclog.NewNullLogger(), false)
	if err != nil {
		t.Fatalf("launch (shadowed): %v", err)
	}
	defer shadowedPlugin.Kill()
	if !shadowedPlugin.shadowed {
		t.Error("expected the shadowed instance's shadowed flag to be true")
	}
	if shadowedPlugin.Tier() != TierTrusted {
		t.Errorf("expected the shadow collision to resolve to %q, got %q", TierTrusted, shadowedPlugin.Tier())
	}

	unshadowed, err := launch(context.Background(), Dirs{Trusted: trustedDir}, "unshadowed-demo", src, nil, hclog.NewNullLogger(), false)
	if err != nil {
		t.Fatalf("launch (unshadowed): %v", err)
	}
	defer unshadowed.Kill()
	if unshadowed.shadowed {
		t.Error("expected the non-colliding instance's shadowed flag to be false")
	}
}

// mustHashBinary is HashBinary with a t.Fatalf on error — trims the
// boilerplate at every call site above that needs a real on-disk digest.
func mustHashBinary(t *testing.T, path string) string {
	t.Helper()
	hash, err := HashBinary(path)
	if err != nil {
		t.Fatalf("HashBinary(%s): %v", path, err)
	}
	return hash
}

// mutateLastByte flips the final byte of the file at path — the same
// single-bit-avalanche technique pin_test.go's own mutation test uses,
// duplicated here (rather than exported) because it's a two-line, purely
// mechanical fixture helper with no shared state to coordinate.
func mutateLastByte(t *testing.T, path string) {
	t.Helper()
	f, err := os.OpenFile(path, os.O_RDWR, 0)
	if err != nil {
		t.Fatalf("open for mutation: %v", err)
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	buf := make([]byte, 1)
	if _, err := f.ReadAt(buf, info.Size()-1); err != nil {
		t.Fatalf("read last byte: %v", err)
	}
	buf[0] ^= 0xFF
	if _, err := f.WriteAt(buf, info.Size()-1); err != nil {
		t.Fatalf("write mutated byte: %v", err)
	}
}

// copyBinaryInto copies the file at src to dst, mode 0o755 — used to place
// a second trusted-directory fixture binary (built under its own name into
// its own temp dir) alongside the first, since Dirs.Trusted names exactly
// ONE directory a real Discover call resolves every trusted instance
// against.
func copyBinaryInto(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", src, err)
	}
	if err := os.WriteFile(dst, data, 0o755); err != nil {
		t.Fatalf("write %s: %v", dst, err)
	}
}

// buildRenamedMockPluginDirForManifestGate builds the repo's plugins/mock
// reference plugin fresh, under an ARBITRARY topos-plugin- prefixed output
// name, into its own fresh temp directory — mirrors
// kernel/supervisor/pinmismatch_test.go's buildRenamedMockPluginDir
// (unavailable here — different package), scoped to this file's own
// manifest-gate fixtures.
func buildRenamedMockPluginDirForManifestGate(t *testing.T, binaryName string) string {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/davison/topos").Output()
	if err != nil {
		t.Fatalf("resolve module root: %v", err)
	}
	root := strings.TrimSpace(string(out))

	dir, err := os.MkdirTemp("", "topos-pluginhost-manifestgate-test-*")
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

// buildSecondMockPluginDirForManifestGate builds a genuinely SECOND,
// independent copy of the repo's plugins/mock reference plugin into its
// own fresh temp directory — deliberately NOT sharing buildMockPluginDir's
// sync.Once cache, mirroring
// kernel/supervisor/externaltier_test.go's buildSecondMockPluginDir: the
// shadow-collision test above needs two distinct directories that both
// really hold a "topos-plugin-mock" binary on disk, not one directory
// referenced twice.
func buildSecondMockPluginDirForManifestGate(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("go", "list", "-m", "-f", "{{.Dir}}", "github.com/davison/topos").Output()
	if err != nil {
		t.Fatalf("resolve module root: %v", err)
	}
	root := strings.TrimSpace(string(out))

	dir, err := os.MkdirTemp("", "topos-pluginhost-manifestgate-shadow-test-*")
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
