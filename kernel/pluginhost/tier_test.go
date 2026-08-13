// tier_test.go pins ResolveBinary's launch-time provenance contract
// (Phase 11, T-11-01/D-11): the trusted-then-external resolution order,
// the loud-by-name collision warning, and the named not-found error —
// the sibling of discover_binaries_test.go's own DiscoverAllTiered/
// DiscoverTiered coverage, scoped to the single-name resolution function
// every launch() call site (Discover, Reconcile, DescribePluginType) goes
// through.
package pluginhost

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/go-hclog"
)

// TestResolveBinary_TrustedOnlyResolvesTrustedTier proves the simplest
// resolution case: a name present only in the trusted directory resolves
// there, tagged TierTrusted, with no warning logged.
func TestResolveBinary_TrustedOnlyResolvesTrustedTier(t *testing.T) {
	trustedDir := t.TempDir()
	writeFixtureFile(t, trustedDir, "topos-plugin-paperless")

	var buf bytes.Buffer
	logger := hclog.New(&hclog.LoggerOptions{Output: &buf})

	path, tier, err := ResolveBinary(Dirs{Trusted: trustedDir}, "topos-plugin-paperless", logger)
	if err != nil {
		t.Fatalf("ResolveBinary: %v", err)
	}
	if want := filepath.Join(trustedDir, "topos-plugin-paperless"); path != want {
		t.Errorf("expected path %q, got %q", want, path)
	}
	if tier != TierTrusted {
		t.Errorf("expected tier %q, got %q", TierTrusted, tier)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no warning logged for a non-colliding trusted binary, got: %s", buf.String())
	}
}

// TestResolveBinary_ExternalOnlyResolvesExternalTier proves a name absent
// from the trusted directory but present externally resolves there,
// tagged TierExternal.
func TestResolveBinary_ExternalOnlyResolvesExternalTier(t *testing.T) {
	trustedDir := t.TempDir()
	externalDir := t.TempDir()
	writeFixtureFile(t, externalDir, "topos-plugin-example")

	path, tier, err := ResolveBinary(Dirs{Trusted: trustedDir, External: externalDir}, "topos-plugin-example", hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("ResolveBinary: %v", err)
	}
	if want := filepath.Join(externalDir, "topos-plugin-example"); path != want {
		t.Errorf("expected path %q, got %q", want, path)
	}
	if tier != TierExternal {
		t.Errorf("expected tier %q, got %q", TierExternal, tier)
	}
}

// TestResolveBinary_CollisionResolvesTrustedAndLogsByName proves D-11's
// full launch-time contract: a name present in BOTH directories resolves
// to the TRUSTED path and tier, AND emits a warning whose text names the
// colliding binary — a shadow must never be silent.
func TestResolveBinary_CollisionResolvesTrustedAndLogsByName(t *testing.T) {
	trustedDir := t.TempDir()
	writeFixtureFile(t, trustedDir, "topos-plugin-shadowed")
	externalDir := t.TempDir()
	writeFixtureFile(t, externalDir, "topos-plugin-shadowed")

	var buf bytes.Buffer
	logger := hclog.New(&hclog.LoggerOptions{Output: &buf})

	path, tier, err := ResolveBinary(Dirs{Trusted: trustedDir, External: externalDir}, "topos-plugin-shadowed", logger)
	if err != nil {
		t.Fatalf("ResolveBinary: %v", err)
	}
	if want := filepath.Join(trustedDir, "topos-plugin-shadowed"); path != want {
		t.Errorf("expected the trusted path %q to win the collision, got %q", want, path)
	}
	if tier != TierTrusted {
		t.Errorf("expected tier %q to win the collision, got %q", TierTrusted, tier)
	}
	if !strings.Contains(buf.String(), "topos-plugin-shadowed") {
		t.Errorf("expected the emitted warning to name the colliding binary, got: %s", buf.String())
	}
}

// TestResolveBinary_NoCollisionWhenOnlyTrustedHasIt proves the negative
// of the collision test above: a name present ONLY in the trusted
// directory (no external counterpart at all) never emits a collision
// warning, even though both directories are configured.
func TestResolveBinary_NoCollisionWhenOnlyTrustedHasIt(t *testing.T) {
	trustedDir := t.TempDir()
	writeFixtureFile(t, trustedDir, "topos-plugin-paperless")
	externalDir := t.TempDir()
	writeFixtureFile(t, externalDir, "topos-plugin-example")

	var buf bytes.Buffer
	logger := hclog.New(&hclog.LoggerOptions{Output: &buf})

	_, tier, err := ResolveBinary(Dirs{Trusted: trustedDir, External: externalDir}, "topos-plugin-paperless", logger)
	if err != nil {
		t.Fatalf("ResolveBinary: %v", err)
	}
	if tier != TierTrusted {
		t.Errorf("expected tier %q, got %q", TierTrusted, tier)
	}
	if buf.Len() != 0 {
		t.Errorf("expected no warning logged when only the trusted directory has this name, got: %s", buf.String())
	}
}

// TestResolveBinary_NeitherDirectoryHasItReturnsNamedError proves neither
// directory holding name returns an error naming the binary and both
// directories searched.
func TestResolveBinary_NeitherDirectoryHasItReturnsNamedError(t *testing.T) {
	trustedDir := t.TempDir()
	externalDir := t.TempDir()

	_, _, err := ResolveBinary(Dirs{Trusted: trustedDir, External: externalDir}, "topos-plugin-does-not-exist", hclog.NewNullLogger())
	if err == nil {
		t.Fatal("expected an error when neither directory has the binary")
	}
	if !strings.Contains(err.Error(), "topos-plugin-does-not-exist") {
		t.Errorf("expected the error to name the missing binary, got: %v", err)
	}
	if !strings.Contains(err.Error(), trustedDir) || !strings.Contains(err.Error(), externalDir) {
		t.Errorf("expected the error to name both searched directories, got: %v", err)
	}
}

// TestResolveBinary_EmptyDirsReturnsNamedErrorNotPanic proves the
// zero-value Dirs{} (both fields the empty string) is handled as
// "nothing to check", not a separate failure mode or a panic.
func TestResolveBinary_EmptyDirsReturnsNamedErrorNotPanic(t *testing.T) {
	_, _, err := ResolveBinary(Dirs{}, "topos-plugin-does-not-exist", hclog.NewNullLogger())
	if err == nil {
		t.Fatal("expected an error for an empty Dirs")
	}
	if !strings.Contains(err.Error(), "topos-plugin-does-not-exist") {
		t.Errorf("expected the error to name the missing binary, got: %v", err)
	}
}
