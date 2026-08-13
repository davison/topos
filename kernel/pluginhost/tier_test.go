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
	"os"
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

// TestResolveBinary_TraversalNameIsRejectedBeforeAnyDirectoryIsConsulted
// proves CR-01's core fix: a "../"-containing name is rejected by
// validatePluginBinaryName as ResolveBinary's first statement, naming the
// offending value, an empty path, and the empty Tier — even when the
// traversal target genuinely exists on disk. A rejection that only works
// because the target is absent would prove nothing, so this test
// materialises a real file in a SIBLING temp directory (never inside
// dirs.Trusted/dirs.External) and proves rejection happens despite it.
func TestResolveBinary_TraversalNameIsRejectedBeforeAnyDirectoryIsConsulted(t *testing.T) {
	trustedDir := t.TempDir()
	outsideDir := t.TempDir()
	writeFixtureFile(t, outsideDir, "topos-plugin-evil")

	path, tier, err := ResolveBinary(Dirs{Trusted: trustedDir}, "../"+filepath.Base(outsideDir)+"/topos-plugin-evil", hclog.NewNullLogger())
	if err == nil {
		t.Fatal("expected an error for a traversal name, even though the target exists on disk")
	}
	if !strings.Contains(err.Error(), "invalid plugin binary name") {
		t.Errorf("expected the error to contain %q, got: %v", "invalid plugin binary name", err)
	}
	if !strings.Contains(err.Error(), "topos-plugin-evil") {
		t.Errorf("expected the error to name the offending value, got: %v", err)
	}
	if path != "" {
		t.Errorf("expected an empty path, got %q", path)
	}
	if tier != Tier("") {
		t.Errorf("expected an empty tier, got %q", tier)
	}
	if tier == TierTrusted {
		t.Errorf("must never return TierTrusted for a traversal name")
	}
}

// TestResolveBinary_AbsolutePathNameIsRejected proves an absolute-path
// value is rejected identically to a relative traversal — filepath.Base
// of an absolute path is not equal to the path itself, so rule (d) alone
// already catches this, but the test pins the observable behavior
// directly rather than relying on the rule's internal mechanics.
func TestResolveBinary_AbsolutePathNameIsRejected(t *testing.T) {
	trustedDir := t.TempDir()
	outsideDir := t.TempDir()
	writeFixtureFile(t, outsideDir, "topos-plugin-evil")

	_, tier, err := ResolveBinary(Dirs{Trusted: trustedDir}, filepath.Join(outsideDir, "topos-plugin-evil"), hclog.NewNullLogger())
	if err == nil {
		t.Fatal("expected an error for an absolute-path name")
	}
	if !strings.Contains(err.Error(), "invalid plugin binary name") {
		t.Errorf("expected the error to contain %q, got: %v", "invalid plugin binary name", err)
	}
	if tier == TierTrusted {
		t.Errorf("must never return TierTrusted for an absolute-path name")
	}
}

// TestResolveBinary_WindowsSeparatorNameIsRejected proves a
// backslash-separated name is rejected on Linux too — filepath.Base does
// not treat '\\' as a separator on this platform, so without the explicit
// backslash check in validatePluginBinaryName, a value like
// "..\\..\\topos-plugin-evil" would pass through as a single opaque
// "filename" and be joined verbatim onto a configured directory.
func TestResolveBinary_WindowsSeparatorNameIsRejected(t *testing.T) {
	trustedDir := t.TempDir()

	_, tier, err := ResolveBinary(Dirs{Trusted: trustedDir}, `..\..\topos-plugin-evil`, hclog.NewNullLogger())
	if err == nil {
		t.Fatal("expected an error for a Windows-separator name")
	}
	if !strings.Contains(err.Error(), "invalid plugin binary name") {
		t.Errorf("expected the error to contain %q, got: %v", "invalid plugin binary name", err)
	}
	if tier == TierTrusted {
		t.Errorf("must never return TierTrusted for a Windows-separator name")
	}
}

// TestResolveBinary_EmptyNameIsRejectedNotResolvedToTheDirectoryItself
// proves the empty-name edge case CR-01 calls out by name:
// filepath.Join(dir, "") equals dir, os.Stat on a directory succeeds, and
// before this fix the caller received the plugins directory itself
// tagged trusted. After the fix, an empty name is rejected outright.
func TestResolveBinary_EmptyNameIsRejectedNotResolvedToTheDirectoryItself(t *testing.T) {
	trustedDir := t.TempDir()

	path, tier, err := ResolveBinary(Dirs{Trusted: trustedDir}, "", hclog.NewNullLogger())
	if err == nil {
		t.Fatal("expected an error for an empty name")
	}
	if path == trustedDir {
		t.Errorf("must never resolve an empty name to the plugins directory itself")
	}
	if tier == TierTrusted {
		t.Errorf("expected tier to not be %q, got %q", TierTrusted, tier)
	}
}

// TestResolveBinary_DotAndDotDotNamesAreRejected proves "." and ".." are
// both rejected — the only remaining ".."-segment forms a name can take
// once bare separators are already barred.
func TestResolveBinary_DotAndDotDotNamesAreRejected(t *testing.T) {
	trustedDir := t.TempDir()

	for _, name := range []string{".", ".."} {
		t.Run(name, func(t *testing.T) {
			_, _, err := ResolveBinary(Dirs{Trusted: trustedDir}, name, hclog.NewNullLogger())
			if err == nil {
				t.Fatalf("expected an error for name %q", name)
			}
			if !strings.Contains(err.Error(), "invalid plugin binary name") {
				t.Errorf("expected the error to contain %q, got: %v", "invalid plugin binary name", err)
			}
		})
	}
}

// TestResolveBinary_DirectoryNamedLikeABinaryIsNotResolved proves T-11-36:
// a DIRECTORY sitting inside dirs.Trusted, named exactly like a plugin
// binary would be, yields the existing named not-found error mentioning
// both searched directories — never a successful resolution that later
// dies inside exec.Command.
func TestResolveBinary_DirectoryNamedLikeABinaryIsNotResolved(t *testing.T) {
	trustedDir := t.TempDir()
	externalDir := t.TempDir()
	if err := os.Mkdir(filepath.Join(trustedDir, "topos-plugin-notafile"), 0o755); err != nil {
		t.Fatalf("mkdir fixture: %v", err)
	}

	_, _, err := ResolveBinary(Dirs{Trusted: trustedDir, External: externalDir}, "topos-plugin-notafile", hclog.NewNullLogger())
	if err == nil {
		t.Fatal("expected an error when the only match is a directory, not a regular file")
	}
	if !strings.Contains(err.Error(), "topos-plugin-notafile") {
		t.Errorf("expected the error to name the binary, got: %v", err)
	}
	if !strings.Contains(err.Error(), trustedDir) || !strings.Contains(err.Error(), externalDir) {
		t.Errorf("expected the error to name both searched directories, got: %v", err)
	}
}

// TestResolveBinary_SymlinkedBinaryStillResolvesAfterRegularFileCheck is
// the regression guard for the browser e2e harness's own fixture shape
// (web/e2e/fixtures/plugin-binaries.ts's linkPluginBinaries symlinks
// rather than copies): a regular plugin binary symlinked into
// dirs.Trusted must still resolve to that path with TierTrusted and a nil
// error after the info.Mode().IsRegular() gate lands, because os.Stat
// (never os.Lstat) follows symlinks.
func TestResolveBinary_SymlinkedBinaryStillResolvesAfterRegularFileCheck(t *testing.T) {
	realDir := t.TempDir()
	writeFixtureFile(t, realDir, "topos-plugin-symlinked")

	trustedDir := t.TempDir()
	linkPath := filepath.Join(trustedDir, "topos-plugin-symlinked")
	if err := os.Symlink(filepath.Join(realDir, "topos-plugin-symlinked"), linkPath); err != nil {
		t.Fatalf("symlink fixture: %v", err)
	}

	path, tier, err := ResolveBinary(Dirs{Trusted: trustedDir}, "topos-plugin-symlinked", hclog.NewNullLogger())
	if err != nil {
		t.Fatalf("ResolveBinary: %v", err)
	}
	if path != linkPath {
		t.Errorf("expected path %q, got %q", linkPath, path)
	}
	if tier != TierTrusted {
		t.Errorf("expected tier %q, got %q", TierTrusted, tier)
	}
}
